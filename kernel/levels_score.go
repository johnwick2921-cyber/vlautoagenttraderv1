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
			continue // S/D + FVG/OB never stand alone
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
	b.WriteString("Anchor: react AT these levels (grade A>B>C); do not chase price between them.")
	return b.String()
}
