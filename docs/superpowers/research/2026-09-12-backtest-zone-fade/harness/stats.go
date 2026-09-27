package main

import (
	"math"
	"sort"
)

// stats.go — small statistics helpers for the backtest. Wilson is the kernel's
// own (kernel.WilsonInterval is used for hold rates); the difference interval
// and permutation helpers live here.

// wilsonCI returns the 95% Wilson score interval for p over n.
func wilsonCI(p float64, n int) (lo, hi float64) {
	if n <= 0 {
		return 0, 0
	}
	const z = 1.959963984540054
	nf := float64(n)
	den := 1 + z*z/nf
	centre := (p + z*z/(2*nf)) / den
	half := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / den
	return centre - half, centre + half
}

// diffCI is the asymptotic 95% interval for p1-p2 (normal approx on each SE).
func diffCI(p1 float64, n1 int, p2 float64, n2 int) (lo, hi float64) {
	if n1 <= 0 || n2 <= 0 {
		return 0, 0
	}
	d := p1 - p2
	se1 := math.Sqrt(p1 * (1 - p1) / float64(n1))
	se2 := math.Sqrt(p2 * (1 - p2) / float64(n2))
	se := math.Sqrt(se1*se1 + se2*se2)
	return d - 1.959963984540054*se, d + 1.959963984540054*se
}

// quartiles of a sorted copy of xs (min, q1, med, q3, max).
func quartiles(xs []float64) (min, q1, med, q3, max float64) {
	if len(xs) == 0 {
		return 0, 0, 0, 0, 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	med = percentileOf(s, 0.50)
	q1 = percentileOf(s, 0.25)
	q3 = percentileOf(s, 0.75)
	return s[0], q1, med, q3, s[n-1]
}

// percentileOf uses the linear-interpolation method on the sorted slice.
func percentileOf(s []float64, p float64) float64 {
	if len(s) == 0 {
		return 0
	}
	if len(s) == 1 {
		return s[0]
	}
	pos := p * float64(len(s)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return s[lo]
	}
	return s[lo] + (s[hi]-s[lo])*(pos-float64(lo))
}

// maxDD computes the maximum drawdown (in the metric's units) of a cumulative
// series. Negative distance below a running peak.
func maxDD(cum []float64) float64 {
	if len(cum) == 0 {
		return 0
	}
	peak := cum[0]
	dd := 0.0
	for _, v := range cum {
		if v > peak {
			peak = v
		}
		if peak-v > dd {
			dd = peak - v
		}
	}
	return dd
}

// longestLosingStreak counts the longest run of consecutive non-positive values.
func longestLosingStreak(xs []float64) int {
	best, cur := 0, 0
	for _, v := range xs {
		if v <= 0 {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

// meanOf / sdOf.
func meanOf(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func sdOf(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := meanOf(xs)
	s := 0.0
	for _, x := range xs {
		d := x - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(xs)-1))
}

// rng is a tiny xorshift RNG with a fixed seed — deterministic permutations.
type rng struct{ s uint64 }

func newRNG(seed uint64) *rng { return &rng{s: seed} }

func (r *rng) next() uint64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return r.s
}

func (r *rng) intn(n int) int { return int(r.next() % uint64(n)) }

// shuffledSigns returns n ±1 flips from the seeded generator.
func shuffledSigns(n int, seed uint64) []float64 {
	r := newRNG(seed)
	out := make([]float64, n)
	for i := range out {
		if r.next()&1 == 1 {
			out[i] = -1
		} else {
			out[i] = 1
		}
	}
	return out
}
