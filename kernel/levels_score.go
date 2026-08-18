package kernel

import (
	"fmt"
	"math"
	"sort"
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
	case KindSupply, KindDemand, KindFVG, KindOB:
		return 0.30 // confluence-only, never standalone
	default:
		return 0.50
	}
}

func isZoneKind(k LevelKind) bool {
	switch k {
	case KindSupply, KindDemand, KindFVG, KindOB:
		return true
	}
	return false
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
		score := typeEvidence(l.Kind) * fm * (1 + 0.20*float64(conf)) * htf
		grade := gradeFromScore(score)
		if isZoneKind(l.Kind) && grade < "C" { // zones capped at C
			grade = "C"
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
		sign := "+"
		if s.Distance < 0 {
			sign = "-"
		}
		fmt.Fprintf(&b, "  %-9.2f %-14s %s  %-9s %s%.1f\n",
			s.Price, s.Label, s.Grade, fresh, sign, math.Abs(s.Distance))
	}
	// W-why-no-trades (2026-08-18): "do not chase price between them" was the
	// base-prompt line the model quoted while waiting cycle after cycle (0/3
	// stripped-prompt replays produced an entry). Between levels, a confirmed
	// momentum/breakout setup stays tradeable.
	b.WriteString("Anchor: react AT these levels (grade A>B>C); between them, a confirmed momentum/breakout may still be traded.")
	return b.String()
}
