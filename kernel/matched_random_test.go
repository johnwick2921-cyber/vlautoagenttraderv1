package kernel

import "testing"

// P5.6 — the honesty gate is the MUST-V1 guarantee: never green on an
// underpowered sample. These lock exactly that.

func TestUnderpoweredIsAlwaysWarming(t *testing.T) {
	// A gorgeous 90% reaction rate on a tiny sample must STILL be WARMING —
	// the power check short-circuits before significance.
	res := EvaluateMatchedRandom(map[string]TypeCounts{
		"PDH": {Touches: 50, Reactions: 45}, // 90% but n=50 << 1565
	})
	if len(res) != 1 || res[0].Status != StatusWarming {
		t.Fatalf("underpowered must be WARMING, got %+v", res)
	}
	if res[0].Label != "WARMING 50/1565" {
		t.Fatalf("label = %q", res[0].Label)
	}
}

func TestPoweredAndSignificantBeatsRandom(t *testing.T) {
	// n at target, reaction rate ~57% → clears 5pp and Bonferroni α.
	res := EvaluateMatchedRandom(map[string]TypeCounts{
		"ONH": {Touches: 2000, Reactions: 1140}, // 57%
	})
	if res[0].Status != StatusBeatsRandom {
		t.Fatalf("powered + significant + ≥5pp must beat random, got %+v", res[0])
	}
	if res[0].PValue >= BonferroniAlpha() {
		t.Fatalf("p-value %.5f should clear alpha %.5f", res[0].PValue, BonferroniAlpha())
	}
}

func TestPoweredButNoEdge(t *testing.T) {
	// n at target, reaction ~51% → real sample, no edge (below 5pp).
	res := EvaluateMatchedRandom(map[string]TypeCounts{
		"RN": {Touches: 2000, Reactions: 1020}, // 51%
	})
	if res[0].Status != StatusNoEdge {
		t.Fatalf("powered sub-5pp must be NO-EDGE, got %+v", res[0])
	}
}

func TestPoweredJustUnderFivePP(t *testing.T) {
	// exactly ~54.9% at large n: significant vs 0.5 but < 5pp effect → NO-EDGE
	// (the effect-size floor, not just significance, must be met).
	res := EvaluateMatchedRandom(map[string]TypeCounts{
		"EQH": {Touches: 5000, Reactions: 2745}, // 54.9%
	})
	if res[0].Status == StatusBeatsRandom {
		t.Fatalf("a <5pp effect must not be BEATS-RANDOM even if significant: %+v", res[0])
	}
}

func TestBonferroniAcross8Types(t *testing.T) {
	a := BonferroniAlpha()
	if a > 0.007 || a < 0.006 {
		t.Fatalf("Bonferroni alpha = %.5f, want ≈0.00625 (0.05/8)", a)
	}
}

func TestZeroTouchesIsWarming(t *testing.T) {
	res := EvaluateMatchedRandom(map[string]TypeCounts{"nPOC": {}})
	if res[0].Status != StatusWarming || res[0].PValue != 1 {
		t.Fatalf("zero-touch must be WARMING p=1, got %+v", res[0])
	}
}
