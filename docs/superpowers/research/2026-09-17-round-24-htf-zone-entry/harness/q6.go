//go:build r24harness

package main

// q6.go — seat-share replay (Round 23 Q6).
//
// Question: "If the top-24 table reserved N seats for HTF structure
// (N ∈ {4, 8, 12}), how many of the last 30 sessions' actual fills/refusals
// would have referenced a different level?"
//
// Method (all on the DB COPY, live store untouched):
//   1. Take the 30 most recent active plan rows for the hoang trader's
//      strategy (latest version per (trade_date, session)), ordered by
//      created_at. The live seat cap is READ from the plan's own
//      zone_map.options.shortlist_cap (measured 12 — the dispatch's "24" is
//      premise-corrected) and variants are run for cap = shortlist_cap and
//      cap = 24 both.
//   2. For each plan, re-run the live detection chain at plan.created_at
//      (same call sites as detect.go) → raw universe, ACTUAL seating
//      (kernel.AssembleResearchLevels / ScoreLevelsMinGradeFull output), and
//      the full scored pool.
//   3. Variant seating for N ∈ {4,8,12}: take the top-N HTF-scored pool
//      members, fill the rest of the cap with the top non-HTF pool members
//      (score desc; |price−level| asc on ties). Approximation documented:
//      scores are the production scores, untouched; only the reservation
//      rule changes.
//   4. Fills/refusals for the session: ab_confirm_log (real rows only:
//      is_counterfactual=0 AND normalized=1) and armed_orders rows for
//      (plan_id, version). Each row's scenario → plan doc scenarios[].level_id
//      → levels[].price (the level the fill/refusal referenced).
//   5. A referenced level is EVICTED by variant N when it is in the actual
//      seating and absent from the variant-N seating (≤1 tick tolerance).
//      Count evictions per session per N.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

type planRow struct {
	PlanID    string
	Version   int
	Date      string
	Session   string
	CreatedAt string
	Doc       string
}

type planDoc struct {
	Levels    []docLevel    `json:"levels"`
	Scenarios []docScenario `json:"scenarios"`
	ZoneMap   docZoneMap    `json:"zone_map"`
}

type docLevel struct {
	ID    string  `json:"id"`
	Price float64 `json:"price"`
	TF    string  `json:"tf"`
	HTF   *bool   `json:"htf"`
}

type docScenario struct {
	ID      string `json:"id"`
	LevelID string `json:"level_id"`
}

type docZoneMap struct {
	Options struct {
		ShortlistCap int `json:"shortlist_cap"`
	} `json:"options"`
}

type q6Result struct {
	PlanID   string
	Date     string
	Session  string
	Version  int
	Cap      int
	NActual  int
	NHTFPool int
	RefRows  int
	RefLvl   int
	Evicted  map[int]int // N → evicted count
	Gained   map[int]int // N → newly-seated HTF levels
}

// runQ6 executes the seat-share replay.
func runQ6(bd *barDB, db *sql.DB, outDir, traderID string, nSessions int) error {
	// NOTE: plans.strategy_id stores the TRADER id for the live path
	// (verified on the copy: 385 rows, all strategy_id = the hoang trader id).
	rows, err := db.Query(`SELECT plan_id, version, trade_date, session, created_at, doc
		FROM plans WHERE strategy_id=? ORDER BY created_at DESC LIMIT 500`, traderID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var plans []planRow
	seen := map[string]bool{}
	for rows.Next() {
		var p planRow
		if err := rows.Scan(&p.PlanID, &p.Version, &p.Date, &p.Session, &p.CreatedAt, &p.Doc); err != nil {
			return err
		}
		k := p.Date + "|" + p.Session
		if seen[k] {
			continue // keep the newest version per session (created_at desc)
		}
		seen[k] = true
		plans = append(plans, p)
		if len(plans) >= nSessions {
			break
		}
	}

	fmt.Printf("q6: %d session-plans (trader %s)\n", len(plans), traderID)
	var results []q6Result
	for _, p := range plans {
		ct, err := time.Parse("2006-01-02T15:04:05.999999999-07:00", p.CreatedAt)
		if err != nil {
			ct, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", p.CreatedAt)
		}
		if err != nil {
			// fallback: strip fractional seconds
			base := strings.SplitN(strings.SplitN(p.CreatedAt, ".", 2)[0], "T", 2)[0]
			if ct, err = time.Parse("2006-01-02 15:04:05-07:00", strings.Replace(strings.SplitN(p.CreatedAt, ".", 2)[0], "T", " ", 1)); err != nil {
				return fmt.Errorf("parse created_at %q (base %q): %w", p.CreatedAt, base, err)
			}
		}
		var doc planDoc
		if err := json.Unmarshal([]byte(p.Doc), &doc); err != nil {
			return fmt.Errorf("doc parse %s v%d: %w", p.PlanID, p.Version, err)
		}
		cap := doc.ZoneMap.Options.ShortlistCap
		if cap <= 0 {
			cap = 12
		}
		res := q6Result{PlanID: p.PlanID, Date: p.Date, Session: p.Session,
			Version: p.Version, Cap: cap, Evicted: map[int]int{}, Gained: map[int]int{}}
		seated, pool, ok := detectAt(bd, db, ct)
		if !ok {
			fmt.Printf("  %s %s v%d: no tape at created_at — skipped\n", p.Date, p.Session, p.Version)
			continue
		}
		res.NActual = len(seated)
		htfPool := 0
		for _, sc := range pool {
			if sc.HTF || sc.TF == "4h" || sc.TF == "1d" {
				htfPool++
			}
		}
		res.NHTFPool = htfPool

		// referenced levels from real fills/refusals
		refPrices := referencedPrices(db, p, doc)
		res.RefRows = len(refPrices)
		actualSet := priceSet(seated)
		var refActual []float64
		for _, pr := range refPrices {
			if nearAny(pr, actualSet) {
				refActual = append(refActual, pr)
			}
		}
		res.RefLvl = len(refActual)

		for _, n := range []int{4, 8, 12} {
			variant := variantSeating(pool, cap, n)
			variantSet := make([]float64, 0, len(variant))
			ev, gained := 0, 0
			for _, sc := range variant {
				variantSet = append(variantSet, sc.Price)
				if !inSet(sc.Price, actualSet) {
					gained++
				}
			}
			for _, pr := range refActual {
				if !nearAny(pr, variantSet) {
					ev++
				}
			}
			res.Evicted[n] = ev
			res.Gained[n] = gained
		}
		results = append(results, res)
		fmt.Printf("  %s %s v%d: actual=%d htfPool=%d refRows=%d refActual=%d ev[N4/8/12]=%d/%d/%d\n",
			p.Date, p.Session, p.Version, res.NActual, res.NHTFPool, res.RefRows, res.RefLvl,
			res.Evicted[4], res.Evicted[8], res.Evicted[12])
	}

	// totals
	tot := map[int]int{}
	totG := map[int]int{}
	sess := map[int]int{} // sessions with ≥1 eviction
	totRef := 0
	for _, r := range results {
		totRef += r.RefLvl
		for _, n := range []int{4, 8, 12} {
			tot[n] += r.Evicted[n]
			totG[n] += r.Gained[n]
			if r.Evicted[n] > 0 {
				sess[n]++
			}
		}
	}
	fmt.Printf("\nq6 totals over %d sessions: referenced-seated levels=%d\n", len(results), totRef)
	for _, n := range []int{4, 8, 12} {
		fmt.Printf("  N=%2d: evicted=%3d (%.1f%% of referenced)  gained HTF=%3d  sessions affected=%d/%d\n",
			n, tot[n], pct(tot[n], totRef), totG[n], sess[n], len(results))
	}
	return nil
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

// detectAt re-runs the live detection chain at an arbitrary instant (the plan's
// created_at) and returns the ACTUAL seating and the full scored pool.
func detectAt(bd *barDB, db *sql.DB, at time.Time) (seated, pool []kernel.ScoredLevel, ok bool) {
	if len(bd.merged1m) == 0 || at.UnixMilli() > bd.merged1m[len(bd.merged1m)-1].openMs {
		return nil, nil, false
	}
	contract, cok := bd.contractAt(at.UnixMilli())
	if !cok {
		return nil, nil, false
	}
	bars1m := bd.lastClosed(contract, "1m", fetch1mCount, at)
	if len(bars1m) == 0 {
		return nil, nil, false
	}
	zoneSeries := map[string][]market.Kline{
		"1m":  bars1m,
		"5m":  kernel.AggregateBars(bars1m, 5*60_000),
		"15m": kernel.AggregateBars(bars1m, 15*60_000),
	}
	fetch := func(tf string, count int) []market.Kline {
		series := bd.lastClosed(contract, tf, count, at)
		zoneSeries[tf] = series
		return series
	}
	htfLevels := kernel.DetectHTFLevels(fetch, plannerTFs, "MNQ", at)
	extra := append([]kernel.DetectedLevel(nil), htfLevels...)
	if pocs := npocFor(db, at); len(pocs) > 0 {
		extra = append(extra, kernel.NakedPOCs(pocs, bars1m, at)...)
	}
	reg := kernel.DefaultSessionRegistry()
	seated, pool, _, _, _ = kernel.AssembleResearchLevels(
		"round-23-q6", bars1m, reg, "MNQ", maxLevelsReplay, at, proximityKReplay, "", extra...)
	return seated, pool, true
}

// variantSeating reserves N HTF seats and fills the rest of cap with the top
// non-HTF pool members. Score desc, then price asc (map ordering).
func variantSeating(pool []kernel.ScoredLevel, cap, n int) []kernel.ScoredLevel {
	htf, non := make([]kernel.ScoredLevel, 0), make([]kernel.ScoredLevel, 0)
	for _, sc := range pool {
		if sc.HTF || sc.TF == "4h" || sc.TF == "1d" {
			htf = append(htf, sc)
		} else {
			non = append(non, sc)
		}
	}
	sortScored(htf)
	sortScored(non)
	out := make([]kernel.ScoredLevel, 0, cap)
	for i := 0; i < n && i < len(htf) && len(out) < cap; i++ {
		out = append(out, htf[i])
	}
	for i := 0; i < len(non) && len(out) < cap; i++ {
		out = append(out, non[i])
	}
	// If fewer than n HTF members exist, remaining HTF ranks are irrelevant —
	// non-HTF fills the cap. Nothing else to do: seats never exceed cap.
	return out
}

func sortScored(list []kernel.ScoredLevel) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Score != list[j].Score {
			return list[i].Score > list[j].Score
		}
		return list[i].Price < list[j].Price
	})
}

func priceSet(seated []kernel.ScoredLevel) []float64 {
	out := make([]float64, 0, len(seated))
	for _, s := range seated {
		out = append(out, s.Price)
	}
	return out
}

func nearAny(price float64, list []float64) bool {
	for _, p := range list {
		if price-p <= 0.25 && p-price <= 0.25 {
			return true
		}
	}
	return false
}

func inSet(price float64, list []float64) bool { return nearAny(price, list) }

// referencedPrices resolves every real fill/refusal row for (plan, version) to
// the price of the level its scenario referenced (via doc.scenarios → levels).
func referencedPrices(db *sql.DB, p planRow, doc planDoc) []float64 {
	levelByID := map[string]float64{}
	for _, l := range doc.Levels {
		if l.ID != "" {
			levelByID[l.ID] = l.Price
		}
	}
	scenLevel := map[string]float64{}
	for _, s := range doc.Scenarios {
		if pr, ok := levelByID[s.LevelID]; ok {
			scenLevel[s.ID] = pr
		}
	}
	var out []float64
	add := func(scenario string) {
		if pr, ok := scenLevel[scenario]; ok {
			out = append(out, pr)
		}
	}
	rows, err := db.Query(`SELECT scenario FROM ab_confirm_log
		WHERE plan_id=? AND version=? AND is_counterfactual=0 AND normalized=1`, p.PlanID, p.Version)
	if err == nil {
		for rows.Next() {
			var sc string
			if rows.Scan(&sc) == nil {
				add(sc)
			}
		}
		rows.Close()
	}
	rows, err = db.Query(`SELECT scenario FROM armed_orders WHERE plan_id=? AND version=?`,
		p.PlanID, p.Version)
	if err == nil {
		for rows.Next() {
			var sc string
			if rows.Scan(&sc) == nil {
				add(sc)
			}
		}
		rows.Close()
	}
	return out
}
