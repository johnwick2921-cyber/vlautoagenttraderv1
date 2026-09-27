package main

// s1struct_test.go — parity guard for the S1 port: reproduces Claude-101's own
// TestStructureMapReadsTrendImpulseAndPremiumDiscount zigzag pin (kernel/
// structure_map_test.go on feat/structure-map @ f36ddca2) through s1Trend.
// The zigzag builder is S1's verbatim: alternating extremes, each leg drawn
// over `legBars` bars, last bar of a leg carries the extreme wick.

import (
	"math"
	"testing"

	"nofx/market"
)

func s1Zigzag(extremes []float64, legBars int, ivMin int, startMs int64) []market.Kline {
	iv := int64(ivMin) * 60_000
	var out []market.Kline
	t := startMs
	for i := 0; i+1 < len(extremes); i++ {
		from, to := extremes[i], extremes[i+1]
		for j := 0; j < legBars; j++ {
			f0 := float64(j) / float64(legBars)
			f1 := float64(j+1) / float64(legBars)
			o := from + (to-from)*f0
			c := from + (to-from)*f1
			w := 0.25
			if j == legBars-1 {
				w = 0.5
			}
			hi, lo := math.Max(o, c)+w, math.Min(o, c)-w
			out = append(out, market.Kline{OpenTime: t, Open: o, High: hi, Low: lo, Close: c, Volume: 1})
			t += iv
		}
	}
	return out
}

// An uptrend on every TF: lows 100→104→108, highs 110→114→118 (HL/HL, HH/HH).
// S1's pin says trend must be "up" on D, 4h and 1h with this fixture.
func TestS1PortTrendMatchesS1ZigzagPin(t *testing.T) {
	start := int64(1_789_000_000_000)
	ext := []float64{100, 110, 104, 114, 108, 118, 112}
	cases := []struct {
		name    string
		legBars int
		ivMin   int
	}{
		{"D", 5, 24 * 60},
		{"4h", 6, 240},
		{"1h", 8, 60},
	}
	now := start + 400*24*60*60_000
	for _, c := range cases {
		trend, swings, closed := s1Trend(s1Zigzag(ext, c.legBars, c.ivMin, start), c.ivMin, now)
		if trend != "up" {
			t.Fatalf("%s trend=%q, want up (S1 pin: HL/HL, HH/HH)", c.name, trend)
		}
		if closed != 6*c.legBars {
			t.Fatalf("%s closed=%d, want %d", c.name, closed, 6*c.legBars)
		}
		if swings < 3 {
			t.Fatalf("%s swings=%d, want >=3", c.name, swings)
		}
	}
}

// The same fixture flipped: highs 110→106→102, lows 100→96→92 → LH/LH, LL/LL.
// S1's rule: all LH/LL → down.
func TestS1PortTrendDown(t *testing.T) {
	start := int64(1_789_000_000_000)
	ext := []float64{110, 100, 106, 96, 102, 92, 98}
	trend, _, _ := s1Trend(s1Zigzag(ext, 5, 24*60, start), 24*60, start+400*24*60*60_000)
	if trend != "down" {
		t.Fatalf("trend=%q, want down (LH/LH, LL/LL)", trend)
	}
}

// Mixed labels (HH then LH) → range, never a guess.
func TestS1PortTrendMixedIsRange(t *testing.T) {
	start := int64(1_789_000_000_000)
	ext := []float64{100, 110, 104, 114, 108, 112, 106} // last high 112 < 114 → LH
	trend, _, _ := s1Trend(s1Zigzag(ext, 5, 24*60, start), 24*60, start+400*24*60*60_000)
	if trend != "range" {
		t.Fatalf("trend=%q, want range (mixed last-3 labels)", trend)
	}
}
