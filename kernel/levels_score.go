package kernel

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// P1.5 — CONFLUENCE SCORER → graded TOP-8.
//
// Deterministic (the LLM never sorts). Each in-band level is scored by
// type-evidence × freshness × confluence × HTF-origin → an A/B/C grade; the
// day-trade lock filters to ±1.5×dATR and seats a capped set with today's
// structural levels getting first seats. S/D + FVG/OB are confluence-only (grade
// capped at C, excluded when they stand alone).

// ScoredLevel is a graded, seated level for the KEY LEVELS block.
type ScoredLevel struct {
	DetectedLevel
	Grade      string  `json:"grade"`      // A | B | C
	Fresh      string  `json:"fresh"`      // freshness label: fresh | tested | B | C
	Score      float64 `json:"score"`      // deterministic composite
	Confluence int     `json:"confluence"` // # other in-band levels clustered nearby
	Distance   float64 `json:"distance"`   // signed points from price (level - price)
}

// freshLabel normalizes a level-state freshness grade to a display label.
func freshLabel(f string) string {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "", "a", "fresh":
		return "fresh"
	case "b":
		return "B"
	case "c", "tested":
		return "tested"
	case "done", "consumed":
		// P1c design rule — a consumed level ROLE-FLIPS (support↔resistance) and
		// STAYS on the map; it is never deleted, and the label says what it is.
		return "flipped"
	default:
		return f
	}
}

// DefaultMaxLevels is the KEY LEVELS table cap (spec max_levels default 8).
const DefaultMaxLevels = 8

var levelGradeRank = map[string]int{"A": 3, "B": 2, "C": 1}

// GradeRank orders a level grade A > B > C (unknown/empty → 0). Exported for
// the write-site machine-grade stamp's collision rule.
func GradeRank(g string) int {
	if r, ok := levelGradeRank[strings.ToUpper(strings.TrimSpace(g))]; ok {
		return r
	}
	return 0
}

// FilterLevelsByMinGrade drops levels graded below minGrade (A > B > C). An empty
// or unknown minGrade is a no-op (no filter). Owner levels grade "A", so they
// survive any minGrade (they are always seated by design).
func FilterLevelsByMinGrade(scored []ScoredLevel, minGrade string) []ScoredLevel {
	min, ok := levelGradeRank[strings.ToUpper(strings.TrimSpace(minGrade))]
	if !ok {
		return scored
	}
	out := make([]ScoredLevel, 0, len(scored))
	for _, l := range scored {
		if levelGradeRank[strings.ToUpper(l.Grade)] >= min {
			out = append(out, l)
		}
	}
	return out
}

// typeEvidence weights a level kind's standalone evidence.
func typeEvidence(k LevelKind) float64 {
	switch k {
	case KindPDH, KindPDL, KindPDC, KindRTHH, KindRTHL, KindPWH, KindPWL, KindPMH, KindPML:
		return 1.0 // strong structural / HTF references
	case KindONH, KindONL, KindNPOC:
		return 0.85
	case KindASH, KindASL, KindLDNH, KindLDNL, KindORH, KindORL, KindIBH, KindIBL, KindEQH, KindEQL:
		return 0.70
	case KindRound, KindGap:
		return 0.55
	case KindSupply, KindDemand, KindFVG, KindIFVG, KindOB:
		return 0.30 // confluence-only, never standalone
	default:
		return 0.50
	}
}

func isZoneKind(k LevelKind) bool {
	switch k {
	case KindSupply, KindDemand, KindFVG, KindIFVG, KindOB:
		return true
	}
	return false
}

// ── v3 zone grading (owner-approved 2026-08-24, research-grounded) ──────────
// Evidence = kindBase(TF) × reversalBonus × TFmult × freshness × (1+0.2·conf).
// Ground: freshness · departure speed · HTF alignment (SMC quality filters);
// RBD/DBR (reversal) > RBR/DBD (continuation). 1m zones stay C (noise); 15m
// floor B, cap B; 1h floor B, cap A (1h wave, 2026-08-25); 4h may reach A.
//
// R3 note (grading audit §4.1, 2026-08-25): zoneEvidenceByKind ALREADY tiers by
// TF and zoneTFMult multiplies AGAIN, so the EFFECTIVE 4h:1m spread is ≈2.3×
// (0.72/0.40 × 1.3/1.0 = 1.80 × 1.30), not the 1.3× a raw TFmult reading
// suggests. DOCUMENTED, NOT CHANGED — the 1h-wave calibration numbers are
// computed against this formula and 4h seniority rides TFMult 1.3 vs 1.2.
// Revisit after the 1h wave has live data (owner queue).

// zoneEvidenceByKind maps kind → TF tier → base evidence. Tiers: 1m / 15m / 1h / 4h.
// 1h values raised by the 1h wave (2026-08-25, owner R1): the 1h is the
// setup/context rung of the intraday ladder (4H trend → 1H setup → 15M entry);
// its zones may now reach A with confluence like 4h.
var zoneEvidenceByKind = map[LevelKind]map[string]float64{
	KindOB:     {"1m": 0.40, "15m": 0.50, "1h": 0.70, "4h": 0.72},
	KindFVG:    {"1m": 0.35, "15m": 0.45, "1h": 0.65, "4h": 0.65},
	KindIFVG:   {"1m": 0.35, "15m": 0.45, "1h": 0.65, "4h": 0.65},
	KindSupply: {"1m": 0.35, "15m": 0.45, "1h": 0.65, "4h": 0.65},
	KindDemand: {"1m": 0.35, "15m": 0.45, "1h": 0.65, "4h": 0.65},
}

// zoneTFMult is the "HTF alignment" multiplier per detection timeframe tier.
var zoneTFMult = map[string]float64{"1m": 1.0, "15m": 1.1, "1h": 1.2, "4h": 1.3}

// zoneReversalBonus rewards RBD/DBR (reversal) over RBR/DBD (continuation).
const zoneReversalBonus = 1.1

// zoneTierFor maps a detection timeframe string to one of the four v3 tiers
// ("" and short TFs → 1m; 30m → 15m; 2h → 1h; 6h/8h/12h → 4h). ANY unknown TF
// falls back to "1m" — a raw pass-through used to miss the zoneTFMult table
// and ZERO the entire zone score (the comment said "noise floor" but the code
// returned the unknown TF as-is).
func zoneTierFor(tf string) string {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "", "1m", "3m", "5m":
		return "1m"
	case "30m":
		return "15m"
	case "2h":
		return "1h"
	case "6h", "8h", "12h":
		return "4h"
	default:
		// Keep the known tier names as-is; anything else → 1m (noise floor),
		// never a missing-map zero.
		if _, ok := zoneTFMult[strings.ToLower(strings.TrimSpace(tf))]; ok {
			return strings.ToLower(strings.TrimSpace(tf))
		}
		return "1m"
	}
}

// ConfluenceCap (C14, 2026-08-25) — the confluence count is capped before it
// multiplies the score: research shows diminishing returns beyond ~3 confirming
// levels, and an uncapped count let a crowded price cluster inflate its own
// grade. Env CONFLUENCE_CAP overrides; the value is a USER knob, never a
// hardcoded rule.
func ConfluenceCap() int {
	if v := os.Getenv("CONFLUENCE_CAP"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			return n
		}
	}
	return 3
}

// zoneSizeMult (C13 / A5, 2026-08-25) — the zone-size axis: a tight small base
// is stronger than an oversized one (S/D literature: a wide zone is a poorly
// defined decision point). Banded 0.5..1.25 in daily-ATR units; a missing
// [Lo,Hi] (line levels, not zones) is neutral 1.0.
func zoneSizeMult(lo, hi, atr float64) float64 {
	if lo <= 0 || hi < lo || atr <= 0 {
		return 1.0
	}
	size := (hi - lo) / atr
	switch {
	case size <= 0.30:
		return 1.25
	case size <= 0.60:
		return 1.10
	case size <= 1.00:
		return 1.0
	case size <= 1.50:
		return 0.85
	case size <= 2.50:
		return 0.70
	default:
		return 0.50
	}
}

// zoneEvidence computes the v3 base evidence for a zone level (kind × TF tier ×
// reversal pattern). Unknown TF → 1m tier (noise floor).
func zoneEvidence(l DetectedLevel) float64 {
	tier := zoneTierFor(l.TF)
	table, ok := zoneEvidenceByKind[l.Kind]
	if !ok {
		return 0.30
	}
	base, ok := table[tier]
	if !ok {
		base = table["1m"]
	}
	if l.ZonePattern == "reversal" {
		base *= zoneReversalBonus
	}
	return base
}

// isTodayPriority marks the kinds that get first seats at the cap.
func isTodayPriority(k LevelKind) bool {
	switch k {
	case KindPDH, KindPDL, KindPDC, KindRTHH, KindRTHL, KindORH, KindORL, KindONH, KindONL:
		return true
	}
	return false
}

// freshMult maps a freshness grade to a score multiplier. "" → fresh.
// P1c design rule: a consumed level is NOT zeroed (that deleted the map's best
// levels) — it role-flips and stays seated at a reduced score.
func freshMult(f string) float64 {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "", "a", "fresh":
		return 1.0
	case "b":
		return 0.8
	case "c", "tested":
		return 0.6
	case "done", "consumed":
		return 0.5 // role-flipped, still on the map
	default:
		return 1.0
	}
}

// ScoreLevels applies the day-trade lock + confluence grading and returns the
// seated TOP-N graded levels, nearest-first. `freshness` returns a level's grade
// from the level-state table (nil → everything fresh). maxLevels ≤ 0 → default 8.
// proximityK is the resolved day-trade lock half-width in daily-ATR multiples
// (the owner's proximity_filter_atr; ≤0 → the spec constant 1.5) — the band
// OUTSIDE which no level is generated or seated.
func ScoreLevels(levels []DetectedLevel, price, dATR float64, freshness func(DetectedLevel) string, maxLevels int, proximityK float64) []ScoredLevel {
	return scoreLevelsPool(levels, price, dATR, freshness, maxLevels, proximityK)
}

// scoreLevelsPool is the full scorer (lock → grade → collapse → seat → top-N).
func scoreLevelsPool(levels []DetectedLevel, price, dATR float64, freshness func(DetectedLevel) string, maxLevels int, proximityK float64) []ScoredLevel {
	if price <= 0 || dATR <= 0 {
		return nil
	}
	if maxLevels <= 0 {
		maxLevels = DefaultMaxLevels
	}
	if proximityK <= 0 {
		proximityK = ActivationWindowK
	}
	band := proximityK * dATR
	confBand := 0.10 * dATR // cluster tolerance

	// Proximity filter (day-trade lock).
	inBand := make([]DetectedLevel, 0, len(levels))
	for _, l := range levels {
		if math.Abs(l.Price-price) <= band {
			inBand = append(inBand, l)
		}
	}

	scored := make([]ScoredLevel, 0, len(inBand))
	for _, l := range inBand {
		fRaw := ""
		if freshness != nil {
			fRaw = freshness(l)
		}
		fm := freshMult(fRaw)
		if fm == 0 {
			continue // consumed
		}
		conf := 0
		for _, o := range inBand {
			if o.Price == l.Price && o.Kind == l.Kind && o.Label == l.Label {
				continue
			}
			if math.Abs(o.Price-l.Price) <= confBand {
				conf++
			}
		}
		// C14 (2026-08-25) — diminishing returns: the confluence count feeding
		// the score is capped (env CONFLUENCE_CAP, default 3).
		effConf := float64(conf)
		if capC := ConfluenceCap(); conf > capC {
			effConf = float64(capC)
		}
		if isZoneKind(l.Kind) && conf == 0 {
			// P0.1 (2026-08-19) — a zone with an HTF origin seats on its own
			// merit (grade C): large-account auctions don't need a crowd. Pure
			// intraday S/D + FVG/OB remain confluence-only, never standalone.
			if !(l.HTF) {
				continue
			}
		}
		htf := 1.0
		if l.HTF {
			htf = 1.2
		}
		var score float64
		if isZoneKind(l.Kind) {
			// v3 zone grading: kindBase × size × TFmult × freshness × confluence.
			score = zoneEvidence(l) * zoneSizeMult(l.Lo, l.Hi, dATR) * fm * (1 + 0.20*effConf) * zoneTFMult[zoneTierFor(l.TF)]
		} else {
			score = typeEvidence(l.Kind) * fm * (1 + 0.20*effConf) * htf
		}
		grade := gradeFromScore(score)
		if isZoneKind(l.Kind) {
			// v3 floors/caps: 1m zones never above C; 15m floor B, cap B
			// (entry TF, not a zone-defining TF — research R20); 1h floor B,
			// cap A (1h wave 2026-08-25 — setup/context rung may reach A with
			// confluence); 4h floor B and may reach A (the professionals' level).
			switch zoneTierFor(l.TF) {
			case "15m":
				if grade != "B" {
					grade = "B"
				}
			case "1h":
				if grade == "C" {
					grade = "B"
				}
			case "4h":
				if grade == "C" {
					grade = "B"
				}
			default: // 1m tier
				if grade != "C" {
					grade = "C"
				}
			}
		}
		scored = append(scored, ScoredLevel{
			DetectedLevel: l,
			Grade:         grade,
			Fresh:         freshLabel(fRaw),
			Score:         score,
			Confluence:    conf,
			Distance:      l.Price - price,
		})
	}

	// P0.4 (2026-08-19) — CLUSTER COLLAPSE: levels within LevelClusterTolerance
	// merge into ONE entry keeping the strongest provenance (highest score,
	// then today-priority kind, then nearer distance). Before this, an equal-
	// high/low family 3 points wide shipped as 4 separate "A" rows and the
	// planner copied every duplicate into the plan (grade inflation + wasted
	// seats).
	scored = collapseLevelClusters(scored, clusterToleranceFor(price))

	// Seat: today's priority kinds first, then the rest, both by score desc
	// (deterministic tie-break: nearer distance, then lower price).
	sort.SliceStable(scored, func(i, j int) bool {
		pi, pj := isTodayPriority(scored[i].Kind), isTodayPriority(scored[j].Kind)
		if pi != pj {
			return pi
		}
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		di, dj := math.Abs(scored[i].Distance), math.Abs(scored[j].Distance)
		if di != dj {
			return di < dj
		}
		return scored[i].Price < scored[j].Price
	})

	// P0.1 (2026-08-19) — BOTH-SIDE SEATING: the plan must always carry levels
	// on BOTH sides of price. On a gap-down day every today-priority kind sits
	// above price and used to fill all 8 seats (the 2026-08-18 one-sided map
	// that left the model with "no trigger" for 110 points of breakdown). When
	// the pure top-N leaves one side under-supplied while candidates exist,
	// swap in the best in-band levels from the missing side.
	// G2/G3/G6 (2026-08-24) — seatHTF first: the model only ever sees the
	// top-N table, so HTF swing/zone levels must WIN seats to reach the plan.
	// The P0.1 side-balance pass may still swap a promoted seat if a side ends
	// under-supplied (the hard rule wins).
	scored = seatHTF(scored, maxLevels)
	scored = seatBothSides(scored, maxLevels)

	if len(scored) > maxLevels {
		scored = scored[:maxLevels]
	}

// Output nearest-first for the executor table.
	sort.SliceStable(scored, func(i, j int) bool {
		return math.Abs(scored[i].Distance) < math.Abs(scored[j].Distance)
	})
	return scored
}

// ScoreLevelsMinGrade (grading audit §4.6/4.7, 2026-08-25) — ScoreLevels with
// a minGrade floor that CANNOT gut a side of the table: the scorer runs on a
// 2× pool, the sub-floor rows are filtered, and seatBothSides re-balances
// AFTER the filter from the surviving pool, so a min_grade cut refills seats
// with in-band same-side candidates instead of leaving the executor/planner
// table one-sided. Empty minGrade → byte-identical to ScoreLevels.
func ScoreLevelsMinGrade(levels []DetectedLevel, price, dATR float64, freshness func(DetectedLevel) string, maxLevels int, proximityK float64, minGrade string) []ScoredLevel {
	eff := maxLevels
	if eff <= 0 {
		eff = DefaultMaxLevels
	}
	if price <= 0 || dATR <= 0 {
		return nil
	}
	pool := scoreLevelsPool(levels, price, dATR, freshness, eff*2, proximityK)
	filtered := FilterLevelsByMinGrade(pool, minGrade)
	if minGrade == "" || len(filtered) <= eff {
		if len(filtered) > eff {
			filtered = filtered[:eff]
		}
		return filtered
	}
	// Re-seat the filtered pool with the SAME seating rules as ScoreLevels so
	// the both-side guarantee (P0.1) holds after the cut.
	sort.SliceStable(filtered, func(i, j int) bool {
		pi, pj := isTodayPriority(filtered[i].Kind), isTodayPriority(filtered[j].Kind)
		if pi != pj {
			return pi
		}
		if filtered[i].Score != filtered[j].Score {
			return filtered[i].Score > filtered[j].Score
		}
		di, dj := math.Abs(filtered[i].Distance), math.Abs(filtered[j].Distance)
		if di != dj {
			return di < dj
		}
		return filtered[i].Price < filtered[j].Price
	})
	filtered = seatHTF(filtered, eff)
	filtered = seatBothSides(filtered, eff)
	if len(filtered) > eff {
		filtered = filtered[:eff]
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return math.Abs(filtered[i].Distance) < math.Abs(filtered[j].Distance)
	})
	return filtered
}

// FilterPlanLevelsByMinGrade (grading audit §4.7, 2026-08-25) — the plan-doc
// twin of FilterLevelsByMinGrade, for surfaces that consume []PlanLevel
// (PLAN STATUS, level-state writers). Empty/unknown minGrade is a no-op.
func FilterPlanLevelsByMinGrade(levels []PlanLevel, minGrade string) []PlanLevel {
	min, ok := levelGradeRank[strings.ToUpper(strings.TrimSpace(minGrade))]
	if !ok {
		return levels
	}
	out := make([]PlanLevel, 0, len(levels))
	for _, l := range levels {
		if levelGradeRank[strings.ToUpper(strings.TrimSpace(l.Grade))] >= min {
			out = append(out, l)
		}
	}
	return out
}

func gradeFromScore(s float64) string {
	switch {
	case s >= 1.0:
		return "A"
	case s >= 0.70:
		return "B"
	default:
		return "C"
	}
}

// LevelClusterTicks is the P0.4 cluster-collapse width in ticks (12 MNQ ticks =
// 3.00 points): levels closer than this are the SAME reference on an intraday
// map, not separate seats. An equal-high/low family within 3 points collapses to
// one entry instead of four separate "A" rows.
const LevelClusterTicks = 12

// clusterToleranceFor derives the cluster width from the MNQ tick size (0.25;
// fallback when price is unset). 12 ticks = 3.00 points.
func clusterToleranceFor(price float64) float64 {
	_ = price
	return LevelClusterTicks * 0.25
}

// collapseLevelClusters merges levels within tol of a STRONGER survivor (kept in
// the same relative position). Kept: highest score, then today-priority kind,
// then nearer distance, then lower price. The survivor's confluence absorbs the
// merged member count; duplicates are removed before seating.
func collapseLevelClusters(scored []ScoredLevel, tol float64) []ScoredLevel {
	if len(scored) < 2 || tol <= 0 {
		return scored
	}
	// Prefer survivors: sort by strength so the first member of any cluster is
	// the keeper.
	order := append([]ScoredLevel(nil), scored...)
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		ai, bi := isTodayPriority(a.Kind), isTodayPriority(b.Kind)
		if ai != bi {
			return ai
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		di, dj := math.Abs(a.Distance), math.Abs(b.Distance)
		if di != dj {
			return di < dj
		}
		return a.Price < b.Price
	})
	kept := make([]ScoredLevel, 0, len(scored))
	for _, cand := range order {
		if isZoneKind(cand.Kind) {
			// Zones are BANDS with their own semantics (proximal/distal),
			// never duplicates of a line level — they survive collapse.
			kept = append(kept, cand)
			continue
		}
		merged := false
		for i := range kept {
			if isZoneKind(kept[i].Kind) {
				continue
			}
			if math.Abs(kept[i].Price-cand.Price) <= tol {
				kept[i].Confluence += cand.Confluence + 1
				merged = true
				break
			}
		}
		if !merged {
			kept = append(kept, cand)
		}
	}
	return kept
}

// MinSideLevels is the P0.1 floor: the plan must carry at least this many levels
// on EACH side of price (when the in-band universe can supply them).
const MinSideLevels = 3

// seatBothSides rebalances the seated top-N so each side of price keeps at least
// MinSideLevels when candidates exist. The input must already be in seating
// order (priority, then score). Pure + deterministic: only swaps the WEAKEST
// currently-seated entries of the over-supplied side for the STRONGEST
// un-seated candidates of the under-supplied side.
// isHTFSwingZone reports whether a scored level is an HTF-origin swing/zone
// (EQH/EQL/S/D/FVG/OB with the HTF flag) — the kinds the seatHTF guarantee
// protects. Multi-day anchors (PDH/PWL…) are today-priority and don't need it.
func isHTFSwingZone(l ScoredLevel) bool {
	if !l.HTF {
		return false
	}
	switch l.Kind {
	case KindEQH, KindEQL, KindSupply, KindDemand, KindFVG, KindIFVG, KindOB:
		return true
	}
	return false
}

// is1HSDZone reports whether a scored level is a 1h-tier supply/demand zone —
// the kind the 1h-wave seat guarantee protects (research: the 1h is the
// setup rung; its S/D bases are the large-account references day traders
// respect). TF survives into ScoredLevel via DetectedLevel.TF.
func is1HSDZone(l ScoredLevel) bool {
	if zoneTierFor(l.TF) != "1h" {
		return false
	}
	switch l.Kind {
	case KindSupply, KindDemand:
		return true
	}
	return false
}

// Seat1HZone (1h wave, 2026-08-25 — research §4 item 2, owner R1) — reserve
// ONE of the two HTF seats for an in-band 1h supply/demand zone when one
// exists. seatHTF fills both seats with the strongest HTF swing/zones, which
// are 4h every time (4h evidence outranks 1h); this post-pass swaps the
// weakest demotable head entry for the best 1h S/D zone in the tail. No-op
// when a 1h S/D zone is already seated, when nothing was cut, or when no
// candidate exists. Pure + deterministic; the seat_1h_zone knob gates the
// CALL SITE, never this function.
func Seat1HZone(scored []ScoredLevel, maxLevels int) []ScoredLevel {
	if maxLevels <= 0 {
		maxLevels = DefaultMaxLevels
	}
	if len(scored) <= maxLevels {
		return scored
	}
	head := append([]ScoredLevel(nil), scored[:maxLevels]...)
	tail := append([]ScoredLevel(nil), scored[maxLevels:]...)
	for _, l := range head {
		if is1HSDZone(l) {
			return scored // already seated
		}
	}
	best := -1
	for i, l := range tail {
		if !is1HSDZone(l) {
			continue
		}
		if best < 0 || l.Score > tail[best].Score {
			best = i
		}
	}
	if best < 0 {
		return scored
	}
	cand := tail[best]
	dropIdx := -1
	for i := len(head) - 1; i >= 0; i-- {
		if isTodayPriority(head[i].Kind) || isHTFSwingZone(head[i]) {
			continue
		}
		dropIdx = i
		break
	}
	if dropIdx < 0 {
		return scored // everything in the head is protected
	}
	tail = append(tail[:best], tail[best+1:]...)
	tail = append(tail, head[dropIdx])
	head = append(head[:dropIdx], head[dropIdx+1:]...)
	head = append(head, cand)
	out := append(head, tail...)
	// Restore strict seating order (same sort as the seatHTF post-pass).
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := isTodayPriority(out[i].Kind), isTodayPriority(out[j].Kind)
		if pi != pj {
			return pi
		}
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		di, dj := math.Abs(out[i].Distance), math.Abs(out[j].Distance)
		if di != dj {
			return di < dj
		}
		return out[i].Price < out[j].Price
	})
	return out
}

// seatHTF (G2/G3/G6, 2026-08-24) — promote up to 2 HTF swing/zone levels into
// the head seats (the top-N table is all the model sees). Demotes the weakest
// non-today-priority, non-HTF entries to make room. Runs on the FULL sorted
// list before seatBothSides; takes no action when nothing was cut.
func seatHTF(scored []ScoredLevel, maxLevels int) []ScoredLevel {
	if maxLevels <= 0 {
		maxLevels = DefaultMaxLevels
	}
	if len(scored) <= maxLevels {
		return scored
	}
	const maxHTFSeats = 2
	head := append([]ScoredLevel(nil), scored[:maxLevels]...)
	tail := append([]ScoredLevel(nil), scored[maxLevels:]...)
	seated := 0
	for _, l := range head {
		if isHTFSwingZone(l) {
			seated++
		}
	}
	need := maxHTFSeats - seated
	if need <= 0 {
		return scored
	}
	var cands []ScoredLevel
	for _, l := range tail {
		if isHTFSwingZone(l) {
			cands = append(cands, l)
		}
	}
	for need > 0 {
		if len(cands) == 0 {
			break
		}
		dropIdx := -1
		for i := len(head) - 1; i >= 0; i-- {
			if isTodayPriority(head[i].Kind) || isHTFSwingZone(head[i]) {
				continue
			}
			dropIdx = i
			break
		}
		if dropIdx < 0 {
			break
		}
		cand := cands[0]
		cands = cands[1:]
		// Remove the candidate from the tail (it now sits in the head) and
		// move the demoted head entry into the tail.
		for i, l := range tail {
			if l.Price == cand.Price && l.Kind == cand.Kind && l.Label == cand.Label && l.HTF == cand.HTF {
				tail = append(tail[:i], tail[i+1:]...)
				break
			}
		}
		tail = append(tail, head[dropIdx])
		head = append(head[:dropIdx], head[dropIdx+1:]...)
		head = append(head, cand)
		need--
	}
	out := append(head, tail...)
	// Restore strict seating order (same sort as the pre-pass).
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := isTodayPriority(out[i].Kind), isTodayPriority(out[j].Kind)
		if pi != pj {
			return pi
		}
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		di, dj := math.Abs(out[i].Distance), math.Abs(out[j].Distance)
		if di != dj {
			return di < dj
		}
		return out[i].Price < out[j].Price
	})
	return out
}

func seatBothSides(scored []ScoredLevel, maxLevels int) []ScoredLevel {
	if maxLevels <= 0 {
		maxLevels = DefaultMaxLevels
	}
	n := len(scored)
	if n <= maxLevels || maxLevels < 2*MinSideLevels {
		// Nothing was cut (the cap is not the constraint) or the cap is too
		// small to hold MinSideLevels on each side — no rebalance.
		return scored
	}
	seated := append([]ScoredLevel(nil), scored[:maxLevels]...)
	rest := append([]ScoredLevel(nil), scored[maxLevels:]...)
	for side := 0; side < 2; side++ {
		count := func(s []ScoredLevel, below bool) int {
			c := 0
			for _, l := range s {
				if (l.Distance < 0) == below {
					c++
				}
			}
			return c
		}
		for _, below := range []bool{true, false} {
			if count(seated, below) >= MinSideLevels {
				continue
			}
			need := MinSideLevels - count(seated, below)
			// Candidates from the missing side, in seating order (strongest
			// first, since rest preserves the sorted order).
			cands := make([]ScoredLevel, 0)
			for _, l := range rest {
				if (l.Distance < 0) == below {
					cands = append(cands, l)
				}
			}
			// Drop the weakest seated levels of the OPPOSITE side to make room.
			for len(cands) > 0 && need > 0 {
				dropIdx := -1
				for i := len(seated) - 1; i >= 0; i-- {
					if (seated[i].Distance < 0) != below {
						dropIdx = i
						break
					}
				}
				if dropIdx < 0 {
					break
				}
				rest = append(rest, seated[dropIdx])
				seated = append(seated[:dropIdx], seated[dropIdx+1:]...)
				seated = append(seated, cands[0])
				cands = cands[1:]
				need--
			}
		}
	}
	// Restore strict seating order for the final nearest-first pass downstream.
	sort.SliceStable(seated, func(i, j int) bool {
		pi, pj := isTodayPriority(seated[i].Kind), isTodayPriority(seated[j].Kind)
		if pi != pj {
			return pi
		}
		if seated[i].Score != seated[j].Score {
			return seated[i].Score > seated[j].Score
		}
		di, dj := math.Abs(seated[i].Distance), math.Abs(seated[j].Distance)
		if di != dj {
			return di < dj
		}
		return seated[i].Price < seated[j].Price
	})
	return seated
}

// RenderKeyLevelsBlock renders the seated levels as the executor prompt block
// (price · label · grade · fresh · signed distance) + one anchor-instruction
// line. Returns "" when there are no levels (the caller injects nothing and logs
// INFO — B9-style dormant block).
func RenderKeyLevelsBlock(scored []ScoredLevel, price float64) string {
	if len(scored) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "KEY LEVELS (map, nearest-first; price %.2f):\n", price)
	for _, s := range scored {
		fresh := s.Fresh
		if s.Confluence > 0 {
			fresh = fmt.Sprintf("%s·x%d", s.Fresh, s.Confluence)
		}
		label := s.Label
		if isHTFSwingZone(s) {
			label = label + " (HTF)"
		}
		sign := "+"
		if s.Distance < 0 {
			sign = "-"
		}
		fmt.Fprintf(&b, "  %-9.2f %-20s %s  %-9s %s%.1f\n",
			s.Price, label, s.Grade, fresh, sign, math.Abs(s.Distance))
	}
	// W-why-no-trades (2026-08-18): "do not chase price between them" was the
	// base-prompt line the model quoted while waiting cycle after cycle (0/3
	// stripped-prompt replays produced an entry). Between levels, a confirmed
	// momentum/breakout setup stays tradeable.
	b.WriteString("Anchor: react AT these levels (grade A>B>C); between them, a confirmed momentum/breakout may still be traded.")
	return b.String()
}
