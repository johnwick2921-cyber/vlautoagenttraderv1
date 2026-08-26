package kernel

import "testing"

// H1/H2 — proximity_filter_atr was hardcoded 1.5 in the scorer (ScoreLevels) and
// the round-number generator (RoundNumberLevels), so the owner's config moved
// only the activation-window paths and never WHICH levels were generated or
// seated. proximityK is now threaded from the resolved config into both, at every
// call site. These tests prove the config actually changes the seated set.

func farLevels() []DetectedLevel {
	// price 20000, dATR 1000: 1.6×..2.4×dATR away from price.
	return []DetectedLevel{
		lineLevel(KindPDH, 20000-1600, "PDH-low", "d", true),
		lineLevel(KindPDC, 20000+1800, "PDC-high", "d", true),
		lineLevel(KindONH, 20000+2400, "ONH-far", "d", true),
	}
}

func TestScoreLevelsProximityKChangesSeatedSet(t *testing.T) {
	price, dATR := 20000.0, 1000.0
	// Default 1.5×: everything beyond 1500 pts is filtered out — NOTHING seats.
	seatedDefault := ScoreLevels(farLevels(), price, dATR, nil, 8, 0)
	if len(seatedDefault) != 0 {
		t.Fatalf("k=default must seat none of the far levels, got %d: %+v", len(seatedDefault), seatedDefault)
	}
	// Owner's proximity_filter_atr = 2.5: all three are inside the band and seat.
	seatedWide := ScoreLevels(farLevels(), price, dATR, nil, 8, 2.5)
	if len(seatedWide) != 3 {
		t.Fatalf("k=2.5 must seat all three far levels, got %d: %+v", len(seatedWide), seatedWide)
	}
	// The middle value (1.8×) is exactly the class of level the setting exists for.
	mid := ScoreLevels(farLevels()[:2], price, dATR, nil, 8, 2.0)
	if len(mid) != 2 {
		t.Fatalf("k=2.0 must seat the 1.6×/1.8× levels, got %d", len(mid))
	}
	if len(ScoreLevels(farLevels()[:2], price, dATR, nil, 8, 1.5)) != 0 {
		t.Fatalf("k=1.5 must still reject the 1.6× level")
	}
}

func TestRoundNumberLevelsProximityK(t *testing.T) {
	// price 20000, dATR 1000. The 18000 round is 2.0×dATR away: OUTSIDE the
	// default 1.5× band (edge 18500) but INSIDE a 2.5× band (edge 17500).
	if lv := RoundNumberLevels(20000, 1000, 1.5); containsPrice(lv, 18000) {
		t.Fatalf("k=1.5 must not generate 18000 (outside the band)")
	}
	if lv := RoundNumberLevels(20000, 1000, 2.5); !containsPrice(lv, 18000) {
		t.Fatalf("k=2.5 must generate 18000 (inside the wider band)")
	}
}

func containsPrice(levels []DetectedLevel, price float64) bool {
	for _, l := range levels {
		if l.Price == price {
			return true
		}
	}
	return false
}
