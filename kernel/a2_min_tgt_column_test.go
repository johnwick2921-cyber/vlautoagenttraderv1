package kernel

import (
	"strings"
	"testing"
)

// WAVE PLANNER A2 + CTO fold 4 (2026-09-25): the candidate table's min_tgt
// column is COMPUTED (MinSLATRMult() × atr5m × the resolved R:R floor), set on
// EVERY candidate in BOTH builder paths, and rendered only when ATR is present
// (canon 49). The executor passes its resolvedMinRR(cfg) through
// PlannerInput.MinTargetRR so the prompt and the arm seam judge with ONE floor.
//
// The mutation that makes this RED: removing the setMinTargetColumn call (the
// column then never renders — the exact pre-fold dead state, where the field
// existed with no setter and no test).

func a2Levels() []ScoredLevel {
	return []ScoredLevel{
		{DetectedLevel: a5Level(KindSWGL, 90, "SWG-L·15m"), Grade: "B", Score: 1},
		{DetectedLevel: a5Level(KindSWGH, 115, "SWG-H·15m"), Grade: "C", Score: 0.5},
	}
}

func TestA2MinTargetColumnComputedFromResolvedRR(t *testing.T) {
	// Default floor: MinSLATRMult() 1.5 × atr5m 8 × 2.0 = 24 pts.
	cs := BuildMapCandidates(a2Levels(), 100, 8, MapCandidateOpts{})
	if len(cs) == 0 {
		t.Fatal("fixture must build candidates")
	}
	for i, c := range cs {
		if c.MinTargetPts != 24 {
			t.Fatalf("candidate %d: MinTargetPts = %g, want 24 (1.5×8×2.0)", i, c.MinTargetPts)
		}
	}
	if block := RenderIdentityMapBlock(cs, 100); !strings.Contains(block, "min_tgt≥24pts") {
		t.Fatalf("the map block must render the computed column:\n%s", block)
	}

	// Resolved floor 2.5 (opts.MinRR): 1.5 × 8 × 2.5 = 30 pts.
	cs2 := BuildMapCandidates(a2Levels(), 100, 8, MapCandidateOpts{MinRR: 2.5})
	for i, c := range cs2 {
		if c.MinTargetPts != 30 {
			t.Fatalf("candidate %d: MinTargetPts = %g, want 30 (1.5×8×2.5)", i, c.MinTargetPts)
		}
	}
	if block := RenderIdentityMapBlock(cs2, 100); !strings.Contains(block, "min_tgt≥30pts") {
		t.Fatalf("resolved floor must drive the column:\n%s", block)
	}

	// ATR absent → no column (an uncomputed value is absent, never fabricated).
	cs3 := BuildMapCandidates(a2Levels(), 100, 0, MapCandidateOpts{})
	for i, c := range cs3 {
		if c.MinTargetPts != 0 {
			t.Fatalf("candidate %d: MinTargetPts must stay 0 without ATR, got %g", i, c.MinTargetPts)
		}
	}
	if block := RenderIdentityMapBlock(cs3, 100); strings.Contains(block, "min_tgt≥") {
		t.Fatalf("no ATR → no column:\n%s", block)
	}
}

// The projections path stamps the column too (BuildMapWithProjections).
func TestA2MinTargetColumnOnTheProjectionsPath(t *testing.T) {
	proj := []MapCandidate{{Price: 120, Names: []string{"PWH"}, Projection: true}}
	cs := BuildMapWithProjections(a2Levels(), proj, 100, 8, MapCandidateOpts{})
	for i, c := range cs {
		if c.MinTargetPts != 24 {
			t.Fatalf("projections path candidate %d: MinTargetPts = %g, want 24", i, c.MinTargetPts)
		}
	}
	cs2 := BuildMapWithProjections(a2Levels(), proj, 100, 8, MapCandidateOpts{MinRR: 3})
	for i, c := range cs2 {
		if c.MinTargetPts != 36 {
			t.Fatalf("projections path candidate %d: MinTargetPts = %g, want 36 (1.5×8×3)", i, c.MinTargetPts)
		}
	}
	if len(cs) == 0 || cs[0].Price == 0 {
		t.Fatalf("projection fixture: %+v", cs)
	}
}
