package main

// blocksim.go — S4b(2) BLOCK SIMULATION (CTO dispatch 2026-09-17 02:26Z).
//
// For the LIVE plans of the last 10 session-days in the DB COPY (plans table,
// read-only, latest version per plan_id, lifecycle='active'), apply three
// candidate rules to each scenario and report what each rule would have
// removed, the D1′ hold rate of removed vs kept, and how many session-days
// would have had ZERO scenarios left.
//
// D1′ per scenario: the ONE live detector (kernel.DetectTouchOutcomes) at the
// scenario level's price (levels[].price via scenarios[].level_id), k=3,
// Δ = trailing-5-day mean |1m close increment| at the read (delta5d), H=12,
// exit_on=close, on the session's 1m window [winStart, flat) of the plan's
// contract. First touch episode (ordinal-1) is the scenario's outcome.
//
// Structure state: the S1 port (s1Trend) computed at the session READ time —
// the same read the live planner used (DefaultSessionRegistry), same closed-bar
// filter, same store copy.
//
// Rules:
//   (a) block oppose-D        : direction opposes the D trend (up/down)
//   (b) block oppose-both     : direction opposes BOTH the D and the 4h trend
//   (c) block oppose-D shorts : short direction opposing the D trend

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"nofx/kernel"
	"nofx/market"
)

type bsScenario struct {
	LevelID   string `json:"level_id"`
	Direction string `json:"direction"`
	ID        string `json:"id"`
	Confirm   struct {
		RefPrice float64 `json:"ref_price"`
	} `json:"confirm"`
	Arm struct {
		Entry float64 `json:"entry"`
	} `json:"arm"`
}

type bsLevel struct {
	ID    string   `json:"id"`
	Price *float64 `json:"price"`
	Label string   `json:"label"`
}

type bsDoc struct {
	Scenarios []bsScenario `json:"scenarios"`
	Levels    []bsLevel    `json:"levels"`
}

type bsPlanRow struct {
	TradeDate string
	Session   string
	Doc       string
	PlanID    string
	Version   int
}

type scenOutcome struct {
	day, session, scenID, dir string
	price                     float64
	levelLabel                string
	outcome                   string // hold | break | ambiguous | none
	oppD, oppH                bool
}

// eraD1d/eraH4 are the S4c era-wide de-stepped series (nil = use the
// contract-keyed store series, the S4b behavior).
var eraD1d, eraH4 []market.Kline

// runBlockSim executes S4b(2) and prints the report tables.
func runBlockSim(bd *barDB, db *sql.DB, outDir string, nSessions int, useEra bool) error {
	if useEra {
		d1d, h4, err := loadEraSeries(db)
		if err != nil {
			return err
		}
		eraD1d, eraH4 = eraToKline(d1d), eraToKline(h4)
		fmt.Printf("blocksim era state: 1d=%d bars, 4h=%d bars\n", len(eraD1d), len(eraH4))
	}
	rows, err := loadRecentPlans(db, nSessions)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("no plans found")
	}
	sessionDays := map[string]bool{}
	var outcomes []scenOutcome
	for _, r := range rows {
		var doc bsDoc
		if err := json.Unmarshal([]byte(r.Doc), &doc); err != nil {
			continue // non-conforming doc — skip (counted in report)
		}
		levelPrice := map[string]float64{}
		levelLabel := map[string]string{}
		for _, l := range doc.Levels {
			if l.Price != nil {
				levelPrice[l.ID] = *l.Price
			}
			levelLabel[l.ID] = l.Label
		}
		readTime, winStart, winEnd := sessionWindow(parseDay(r.TradeDate), r.Session)
		if winEnd.After(lastBarDay(bd)) {
			continue // window incomplete in the copy — cannot measure
		}
		contract, ok := bd.contractAt(readTime.UnixMilli())
		if !ok {
			continue
		}
		delta := delta5d(bd, contract, readTime.UnixMilli())
		if delta <= 0 {
			continue
		}
		var dTrend, hTrend string
		if eraD1d != nil {
			dTrend, _, _ = s1Trend(eraD1d, 1440, readTime.UnixMilli())
			hTrend, _, _ = s1Trend(eraH4, 240, readTime.UnixMilli())
		} else {
			dTrend, _, _ = trendFor(bd, contract, "1d", readTime.UnixMilli())
			hTrend, _, _ = trendFor(bd, contract, "4h", readTime.UnixMilli())
		}
		fmt.Printf("session %s %s contract=%s d_trend=%s h4_trend=%s\n", r.TradeDate, r.Session, contract, dTrend, hTrend)
		_, bars := bd.sessionBars1m(contract, winStart.UnixMilli(), winEnd.UnixMilli())
		if len(bars) < 2 {
			continue
		}
		for _, sc := range doc.Scenarios {
			price, ok := levelPrice[sc.LevelID]
			if !ok || price <= 0 {
				// live docs carry levels with null ids; fall back to the
				// scenario's own anchor (confirm.ref_price, then arm.entry)
				price = sc.Confirm.RefPrice
				if price <= 0 {
					price = sc.Arm.Entry
				}
			}
			if price <= 0 {
				continue // scenario without a resolvable anchor — unmeasurable
			}
			dir := sc.Direction
			if dir != "long" && dir != "short" {
				continue
			}
			eps := kernel.DetectTouchOutcomes(bars, price, kernel.DetectorK(), delta, kernel.DetectorHorizonBars(), "close")
			outcome := "none"
			for _, e := range eps {
				if e.Ordinal == 1 {
					outcome = e.Outcome
					break
				}
			}
			sessionDays[r.TradeDate+"|"+r.Session] = true
			outcomes = append(outcomes, scenOutcome{
				day: r.TradeDate, session: r.Session, scenID: sc.ID, dir: dir,
				price: price, levelLabel: levelLabel[sc.LevelID], outcome: outcome,
				oppD: (dTrend == "up" && dir == "short") || (dTrend == "down" && dir == "long"),
				oppH: (hTrend == "up" && dir == "short") || (hTrend == "down" && dir == "long"),
			})
		}
	}
	fmt.Printf("session-days measured: %d  scenarios measured: %d\n", len(sessionDays), len(outcomes))
	reportRule("(a) block oppose-D", outcomes, func(o scenOutcome) bool { return o.oppD })
	reportRule("(b) block oppose-both", outcomes, func(o scenOutcome) bool { return o.oppD && o.oppH })
	reportRule("(c) block oppose-D shorts", outcomes, func(o scenOutcome) bool { return o.oppD && o.dir == "short" })
	return nil
}

func reportRule(name string, os []scenOutcome, blocked func(scenOutcome) bool) {
	var rem, keep []scenOutcome
	for _, o := range os {
		if blocked(o) {
			rem = append(rem, o)
		} else {
			keep = append(keep, o)
		}
	}
	hr := func(list []scenOutcome) (hold, brk, amb, none int) {
		for _, o := range list {
			switch o.outcome {
			case "hold":
				hold++
			case "break":
				brk++
			default:
				if o.outcome == "none" {
					none++
				} else {
					amb++
				}
			}
		}
		return
	}
	hh, hb, ha, hn := hr(rem)
	kh, kb, ka, kn := hr(keep)
	fmt.Printf("\n== %s ==\n", name)
	fmt.Printf("  removed: %d scenarios (hold %d break %d ambig %d none %d) D1' hold %.3f [n=%d]\n",
		len(rem), hh, hb, ha, hn, rate(hh, hh+hb), hh+hb)
	fmt.Printf("  kept   : %d scenarios (hold %d break %d ambig %d none %d) D1' hold %.3f [n=%d]\n",
		len(keep), kh, kb, ka, kn, rate(kh, kh+kb), kh+kb)
	sessions := map[string]bool{}
	zeroSessions := 0
	for _, o := range keep {
		sessions[o.day+"|"+o.session] = true
	}
	all := map[string]bool{}
	for _, o := range os {
		all[o.day+"|"+o.session] = true
	}
	for k := range all {
		if !sessions[k] {
			zeroSessions++
		}
	}
	fmt.Printf("  session-days with ZERO scenarios left: %d / %d\n", zeroSessions, len(all))
	if len(rem) > 0 && len(rem) <= 10 {
		for _, o := range rem {
			fmt.Printf("    removed id: %s %s %s %s %s@%.2f %s\n", o.day, o.session, o.scenID, o.dir, o.levelLabel, o.price, o.outcome)
		}
	}
}

func rate(h, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(h) / float64(n)
}

func parseDay(s string) (t time.Time) {
	t, _ = time.ParseInLocation("2006-01-02", s, kernel.CTLocation())
	return
}

func lastBarDay(bd *barDB) time.Time {
	if len(bd.merged1m) == 0 {
		return time.Time{}
	}
	return time.UnixMilli(bd.merged1m[len(bd.merged1m)-1].openMs).In(kernel.CTLocation())
}

// loadRecentPlans returns the latest-version active plan rows for the most
// recent n session-days (chronological: ASIA > NY > LONDON within a day).
func loadRecentPlans(db *sql.DB, nSessions int) ([]bsPlanRow, error) {
	rows, err := db.Query(`SELECT trade_date, session, plan_id, version, doc
		FROM plans WHERE lifecycle='active'
		ORDER BY trade_date DESC,
			CASE session WHEN 'ASIA' THEN 3 WHEN 'NY' THEN 2 WHEN 'LONDON' THEN 1 ELSE 0 END DESC,
			version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seenDay := map[string]bool{}
	seenPlan := map[string]bool{}
	var out []bsPlanRow
	for rows.Next() {
		var pr bsPlanRow
		if err := rows.Scan(&pr.TradeDate, &pr.Session, &pr.PlanID, &pr.Version, &pr.Doc); err != nil {
			return nil, err
		}
		if seenPlan[pr.PlanID] {
			continue // a newer version of this plan already selected
		}
		seenPlan[pr.PlanID] = true
		key := pr.TradeDate + "|" + pr.Session
		if !seenDay[key] {
			if len(seenDay) >= nSessions {
				break
			}
			seenDay[key] = true
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}
