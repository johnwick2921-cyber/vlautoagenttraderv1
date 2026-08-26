package kernel

import (
	"math"
	"sort"
)

// P5.6 — the STATS HONESTY GATE (MUST-V1). Matched-random reaction stats decide
// whether a level TYPE beats a coin-flip by the pre-registered bar. The
// non-negotiable rule: NO green "beats random" on an underpowered sample, ever —
// below the pre-registered N a type is always WARMING, whatever the point
// estimate looks like. Bonferroni-corrected across the 8 level types, evaluated
// on a fixed weekly cadence (never nightly optional-stopping).

const (
	// PreRegisteredN is the sample per type needed for the 5pp bar (spec ≈1,565).
	PreRegisteredN = 1565
	// MatchedRandomTypes is the family size for the Bonferroni correction.
	MatchedRandomTypes = 8
	// RandomBaseline is the coin-flip reaction rate a real level must beat.
	RandomBaseline = 0.50
	// MinEffectPP is the minimum edge over random (5 percentage points).
	MinEffectPP = 0.05
)

// BonferroniAlpha is the family-wise-corrected significance threshold (≈0.006).
func BonferroniAlpha() float64 { return 0.05 / float64(MatchedRandomTypes) }

// LevelTypeFromLabel maps a level's provenance label to one of the 8 tracked
// matched-random types (or "other"). Case-insensitive prefix match.
func LevelTypeFromLabel(label string) string {
	l := upper(label)
	switch {
	case has(l, "PDH"), has(l, "PDL"), has(l, "PDC"):
		return "PDH/PDL/PDC"
	case has(l, "ONH"), has(l, "ONL"):
		return "ONH/ONL"
	case has(l, "POC"): // nPOC / naked POC
		return "naked-POC"
	case has(l, "PW"), has(l, "PM"), has(l, "WK"), has(l, "MO"):
		return "prior-wk/mo"
	case has(l, "OR"), has(l, "IB"):
		return "opening-range"
	case has(l, "RN"):
		return "round-number"
	case has(l, "EQH"), has(l, "EQL"), has(l, "EQ"):
		return "equal-H/L"
	case has(l, "FVG"), has(l, "OB"), has(l, "S/D"), has(l, "SUPPLY"), has(l, "DEMAND"), has(l, "ZONE"):
		return "S/D+FVG/OB"
	default:
		return "other"
	}
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
func has(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TypeCounts is the raw tally for one level type.
type TypeCounts struct {
	Touches   int `json:"touches"`
	Reactions int `json:"reactions"`
}

// MatchedRandomStatus is the honesty verdict for a level type.
type MatchedRandomStatus string

const (
	StatusWarming     MatchedRandomStatus = "WARMING"      // n < target → never a green claim
	StatusBeatsRandom MatchedRandomStatus = "BEATS-RANDOM" // powered AND significant AND ≥5pp
	StatusNoEdge      MatchedRandomStatus = "NO-EDGE"      // powered but not a real edge
)

// TypeVerdict is the per-type honesty result.
type TypeVerdict struct {
	LevelType string              `json:"level_type"`
	N         int                 `json:"n"`
	Reactions int                 `json:"reactions"`
	ReactRate float64             `json:"react_rate"`
	Baseline  float64             `json:"baseline"`
	DeltaPP   float64             `json:"delta_pp"` // (reactRate - baseline) in points
	PValue    float64             `json:"p_value"`
	Alpha     float64             `json:"alpha"`
	TargetN   int                 `json:"target_n"`
	Status    MatchedRandomStatus `json:"status"`
	Label     string              `json:"label"` // "WARMING 320/1565" etc.
}

// EvaluateMatchedRandom applies the honesty gate to each type's counts. The gate
// ordering is deliberate: the power check (n >= target) comes FIRST and short-
// circuits to WARMING — significance can never override an underpowered sample.
func EvaluateMatchedRandom(perType map[string]TypeCounts) []TypeVerdict {
	alpha := BonferroniAlpha()
	types := make([]string, 0, len(perType))
	for t := range perType {
		types = append(types, t)
	}
	sort.Strings(types)

	out := make([]TypeVerdict, 0, len(types))
	for _, t := range types {
		c := perType[t]
		v := TypeVerdict{
			LevelType: t, N: c.Touches, Reactions: c.Reactions,
			Baseline: RandomBaseline, Alpha: alpha, TargetN: PreRegisteredN,
		}
		if c.Touches > 0 {
			v.ReactRate = float64(c.Reactions) / float64(c.Touches)
		}
		v.DeltaPP = v.ReactRate - RandomBaseline
		v.PValue = oneSidedProportionP(c.Reactions, c.Touches, RandomBaseline)

		switch {
		case c.Touches < PreRegisteredN:
			// UNDERPOWERED → WARMING, no matter how good the estimate looks.
			v.Status = StatusWarming
			v.Label = warmingLabel(c.Touches)
		case v.DeltaPP >= MinEffectPP && v.PValue < alpha:
			v.Status = StatusBeatsRandom
			v.Label = "beats random"
		default:
			v.Status = StatusNoEdge
			v.Label = "no edge"
		}
		out = append(out, v)
	}
	return out
}

func warmingLabel(n int) string {
	// "WARMING 320/1565"
	return "WARMING " + itoa(n) + "/" + itoa(PreRegisteredN)
}

// oneSidedProportionP is the one-sided p-value that the reaction rate exceeds p0
// (H1: phat > p0), via the normal approximation to the binomial. n==0 → p=1.
func oneSidedProportionP(reactions, n int, p0 float64) float64 {
	if n <= 0 {
		return 1
	}
	phat := float64(reactions) / float64(n)
	se := math.Sqrt(p0 * (1 - p0) / float64(n))
	if se == 0 {
		return 1
	}
	z := (phat - p0) / se
	return 1 - normalCDF(z) // upper tail
}

// normalCDF is the standard-normal CDF via the error function.
func normalCDF(z float64) float64 {
	return 0.5 * math.Erfc(-z/math.Sqrt2)
}

// itoa is a tiny int→string (avoids importing strconv into this leaf math file).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
