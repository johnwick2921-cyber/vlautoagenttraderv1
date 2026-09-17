package kernel

import (
	"math"
	"testing"

	"nofx/market"
)

// ── S1 STRUCTURE LAYER — pins ───────────────────────────────────────────────

// zigzag builds closed bars whose fractal swings are exactly the given
// extremes (alternating, first is a LOW), each leg drawn over `legBars` bars so
// the k=2 fractal window sees every turn. Bars are `ivMin` minutes wide and
// end well before nowMs so all are closed.
func zigzag(extremes []float64, legBars int, ivMin int, startMs int64) []market.Kline {
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
			// the leg's LAST bar carries the extreme wick; every other bar a
			// smaller one, so the turn is strictly the fractal extreme (the
			// detector compares with >=; an equal neighbour is not a swing)
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
// The last impulse is the last low (108) → the last high (118); price 115.5
// sits at 0.75 of it. The map must say up, name both swings, and place price.
func TestStructureMapReadsTrendImpulseAndPremiumDiscount(t *testing.T) {
	start := int64(1_789_000_000_000)
	ext := []float64{100, 110, 104, 114, 108, 118, 112}
	reader := func(tf string) []market.Kline {
		switch tf {
		case "D":
			return zigzag(ext, 5, 24*60, start)
		case "4h":
			return zigzag(ext, 6, 240, start)
		case "1h":
			return zigzag(ext, 8, 60, start)
		}
		return nil
	}
	now := start + 400*24*60*60_000
	m := ComputeStructureMap(reader, "MNQ 12-26", nil, 115.5, now, DefaultStructureTrendSwings)
	if m == nil {
		t.Fatal("map is nil with bars on every TF")
	}
	if m.Contract != "MNQ 12-26" || m.AsOf != now {
		t.Fatalf("contract/as_of not carried: %+v", m)
	}
	for _, tf := range StructureMapTFs {
		s, ok := m.TFs[tf]
		if !ok {
			t.Fatalf("%s missing from the map", tf)
		}
		if s.Trend != "up" {
			t.Fatalf("%s trend=%q, want up (HL/HL, HH/HH)", tf, s.Trend)
		}
		if s.LastSwingHigh == nil || math.Abs(s.LastSwingHigh.Price-118.5) > 0.6 {
			t.Fatalf("%s last swing high=%+v, want ≈118", tf, s.LastSwingHigh)
		}
		if s.LastSwingLow == nil || math.Abs(s.LastSwingLow.Price-107.5) > 0.6 {
			t.Fatalf("%s last swing low=%+v, want ≈108", tf, s.LastSwingLow)
		}
		if s.ImpulseLo >= s.ImpulseHi || s.ImpulseLo > 108.6 || s.ImpulseHi < 117.4 {
			t.Fatalf("%s impulse=[%.2f, %.2f], want ≈[108, 118]", tf, s.ImpulseLo, s.ImpulseHi)
		}
		if math.Abs(s.PremiumDiscount-0.75) > 0.08 {
			t.Fatalf("%s premium_discount=%.3f, want ≈0.75 for price 115.5 in [108,118]", tf, s.PremiumDiscount)
		}
		if s.Bars == 0 {
			t.Fatalf("%s bars=0 — the read must say what it rested on", tf)
		}
	}
}

// A downtrend reads down; a saw with no ordering reads range. Never a guess.
func TestStructureMapReadsDownAndRange(t *testing.T) {
	start := int64(1_789_000_000_000)
	down := []float64{120, 110, 118, 106, 114, 102, 108}
	saw := []float64{100, 110, 104, 112, 100, 111, 105}
	now := start + 400*24*60*60_000
	m := ComputeStructureMap(func(tf string) []market.Kline {
		if tf == "1h" {
			return zigzag(saw, 8, 60, start)
		}
		return zigzag(down, 6, 240, start)
	}, "MNQ 12-26", nil, 105, now, DefaultStructureTrendSwings)
	if m.TFs["4h"].Trend != "down" || m.TFs["D"].Trend != "down" {
		t.Fatalf("down series read as %q/%q", m.TFs["D"].Trend, m.TFs["4h"].Trend)
	}
	if m.TFs["1h"].Trend != "range" {
		t.Fatalf("saw read as %q, want range", m.TFs["1h"].Trend)
	}
}

// ABSENT, NEVER FABRICATED (canon): no bars on any TF → nil map; a TF with no
// bars is simply not in the map (no zero-valued entry).
func TestStructureMapIsAbsentWithoutBars(t *testing.T) {
	if m := ComputeStructureMap(func(string) []market.Kline { return nil }, "MNQ 12-26", nil, 100, 1, 3); m != nil {
		t.Fatalf("no bars must yield a nil map, got %+v", m)
	}
	start := int64(1_789_000_000_000)
	m := ComputeStructureMap(func(tf string) []market.Kline {
		if tf == "1h" {
			return zigzag([]float64{100, 110, 104, 114, 108, 118, 112}, 8, 60, start)
		}
		return nil
	}, "MNQ 12-26", nil, 115, start+400*24*3600_000, 3)
	if m == nil || len(m.TFs) != 1 {
		t.Fatalf("only 1h has bars → exactly one TF in the map, got %+v", m)
	}
}

// Zones: top-N HTF zones of THAT TF from the scored pool, label intact, best
// score first, capped at 6; a line (Lo == Hi) is not a zone.
func TestStructureMapZonesAreTheTFsOwnTopZonesLabelIntact(t *testing.T) {
	start := int64(1_789_000_000_000)
	pool := []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: LevelKind("OB"), Lo: 100, Hi: 102, TF: "4h", HTF: true, Label: "OB·4h"}, Score: 5, Fresh: "fresh"},
		{DetectedLevel: DetectedLevel{Kind: LevelKind("Supply"), Lo: 120, Hi: 123, TF: "4h", HTF: true, Label: "Supply·4h"}, Score: 9, Fresh: "tested"},
		{DetectedLevel: DetectedLevel{Kind: LevelKind("OB"), Lo: 90, Hi: 91, TF: "1h", HTF: true, Label: "OB·1h"}, Score: 7, Fresh: "fresh"},
		{DetectedLevel: DetectedLevel{Kind: LevelKind("PDH"), Lo: 130, Hi: 130, TF: "D", HTF: true, Label: "PDH"}, Score: 8, Fresh: "fresh"},
	}
	for i := 0; i < 8; i++ { // 8 more 4h zones → the cap of 6 applies
		pool = append(pool, ScoredLevel{DetectedLevel: DetectedLevel{Kind: LevelKind("FVG"), Lo: 140 + float64(i), Hi: 141 + float64(i), TF: "4h", HTF: true, Label: "FVG·4h"}, Score: float64(i), Fresh: "fresh"})
	}
	m := ComputeStructureMap(func(tf string) []market.Kline {
		return zigzag([]float64{100, 110, 104, 114, 108, 118, 112}, 6, 240, start)
	}, "MNQ 12-26", pool, 115, start+400*24*3600_000, 3)
	z4 := m.TFs["4h"].Zones
	if len(z4) != 6 {
		t.Fatalf("4h zones=%d, want 6 (cap)", len(z4))
	}
	if z4[0].Kind != "Supply" || z4[0].Score != 9 || z4[0].Fresh != "tested" || z4[0].TF != "4h" {
		t.Fatalf("best 4h zone first, label intact: got %+v", z4[0])
	}
	if len(m.TFs["1h"].Zones) != 1 || m.TFs["1h"].Zones[0].Kind != "OB" {
		t.Fatalf("1h zones=%+v, want the one 1h OB", m.TFs["1h"].Zones)
	}
	if len(m.TFs["D"].Zones) != 0 {
		t.Fatalf("D: a line (PDH, Lo==Hi) is not a zone; got %+v", m.TFs["D"].Zones)
	}
}
