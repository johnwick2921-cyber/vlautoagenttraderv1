package store

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// EXCURSION DISTRIBUTIONS (wave 1A, E6) — the read-only view the stop-size
// ruling is derived from.
//
// A24 governs the output: every row carries its n, no rate is printed without
// one, and a group whose rows have no measured path is reported as unmeasured
// rather than folded into the percentiles as zeros.

// ExcursionBucket is one group's distribution. Percentiles are in points.
type ExcursionBucket struct {
	Key string // the group ("reject", "NY", …)
	N   int    // rows WITH a measured path — the n every number below rests on

	MAEp50, MAEp80, MAEp95 float64
	MFEp50, MFEp80, MFEp95 float64

	// AmbiguousRows is how many of the N had at least one bar reaching both
	// the stop and the target; UnknownLevels is how many could not be judged
	// because the levels were never resolved. The second number is why the
	// first is not a rate on its own.
	AmbiguousRows int
	UnknownLevels int

	// Unmeasured rows exist in the group but carry no path (resolution="none").
	Unmeasured int
}

// AmbiguousShare returns the share of judgeable rows that were ambiguous, and
// the n it rests on. It returns ok=false when nothing could be judged — a rate
// with no denominator is not printed (A24).
func (b ExcursionBucket) AmbiguousShare() (share float64, n int, ok bool) {
	n = b.N - b.UnknownLevels
	if n <= 0 {
		return 0, 0, false
	}
	return float64(b.AmbiguousRows) / float64(n), n, true
}

// ExcursionDistribution groups the rows by one dimension.
func (s *TradeExcursionStore) ExcursionDistribution(dimension string) ([]ExcursionBucket, error) {
	col := map[string]func(TradeExcursion) string{
		"condition": func(r TradeExcursion) string { return orUnknown(r.Condition) },
		"session":   func(r TradeExcursion) string { return orUnknown(r.Session) },
		"scenario":  func(r TradeExcursion) string { return orUnknown(r.Scenario) },
		"side":      func(r TradeExcursion) string { return orUnknown(r.Side) },
	}[dimension]
	if col == nil {
		return nil, fmt.Errorf("unknown dimension %q (condition|session|scenario|side)", dimension)
	}
	var rows []TradeExcursion
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}

	type acc struct {
		mae, mfe              []float64
		ambiguous, unknownLvl int
		unmeasured            int
	}
	groups := map[string]*acc{}
	for _, r := range rows {
		k := col(r)
		g := groups[k]
		if g == nil {
			g = &acc{}
			groups[k] = g
		}
		if r.MAEPts == nil || r.MFEPts == nil {
			g.unmeasured++ // no path — NOT a zero
			continue
		}
		g.mae = append(g.mae, *r.MAEPts)
		g.mfe = append(g.mfe, *r.MFEPts)
		if r.StopPxInitial <= 0 || r.TargetPx <= 0 {
			g.unknownLvl++
		} else if (r.AmbiguousBars != nil && *r.AmbiguousBars > 0) || r.AmbiguousExit {
			g.ambiguous++
		}
	}

	out := make([]ExcursionBucket, 0, len(groups))
	for k, g := range groups {
		b := ExcursionBucket{
			Key: k, N: len(g.mae),
			AmbiguousRows: g.ambiguous, UnknownLevels: g.unknownLvl, Unmeasured: g.unmeasured,
		}
		b.MAEp50, b.MAEp80, b.MAEp95 = pct(g.mae, 50), pct(g.mae, 80), pct(g.mae, 95)
		b.MFEp50, b.MFEp80, b.MFEp95 = pct(g.mfe, 50), pct(g.mfe, 80), pct(g.mfe, 95)
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(unknown)"
	}
	return s
}

// pct is the nearest-rank percentile. An empty sample returns NaN, which the
// renderer prints as "—": there is no p50 of nothing.
func pct(v []float64, p int) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	idx := int(math.Ceil(float64(p)/100*float64(len(s)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

// RenderExcursionDistribution formats one dimension as a fixed-width table.
func RenderExcursionDistribution(dimension string, buckets []ExcursionBucket) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-22s %5s  %8s %8s %8s  %8s %8s %8s  %s\n",
		"by "+dimension, "n", "MAE p50", "p80", "p95", "MFE p50", "p80", "p95", "ambiguous")
	for _, k := range buckets {
		amb := "— (no levels)"
		if share, n, ok := k.AmbiguousShare(); ok {
			amb = fmt.Sprintf("%.1f%% of %d", share*100, n)
		}
		unmeasured := ""
		if k.Unmeasured > 0 {
			unmeasured = fmt.Sprintf("  · %d unmeasured", k.Unmeasured)
		}
		fmt.Fprintf(&b, "%-22s %5d  %8s %8s %8s  %8s %8s %8s  %s%s\n",
			k.Key, k.N, num(k.MAEp50), num(k.MAEp80), num(k.MAEp95),
			num(k.MFEp50), num(k.MFEp80), num(k.MFEp95), amb, unmeasured)
	}
	return b.String()
}

func num(f float64) string {
	if math.IsNaN(f) {
		return "—"
	}
	return fmt.Sprintf("%.2f", f)
}
