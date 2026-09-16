// W1 item 1 — the touch → scenario link, recorded as the HEURISTIC it is.
//
// PlanScenario carries NO level reference (kernel/plan_doc.go — ID, Trigger,
// Condition, Direction, TargetChain, Invalid, Confirm, Quality, Fvg, Breakdown,
// ChainAfter, Arm; not one names a level). Trigger and Invalid are free text.
// The codebase already says so at kernel/scenario_state.go:19-22: "To evaluate a
// scenario we must first decide WHICH LEVEL it is about, and that resolution is
// a heuristic."
//
// So this link cannot be identity, at authoring or at read. Moving a heuristic
// earlier does not make it a fact — it only makes a guess look like one. It is
// therefore recorded as NEAREST-BY-PRICE with its basis and its distance in
// BOTH points and Δ, and it is NULL whenever the answer is ambiguous.
//
// The rule this file follows is the one already written beside the heuristic it
// replaces: "a confidently-wrong dot would be a NEW lie replacing the old one."

package store

import "testing"

func anchors(pairs ...any) []ScenarioAnchor {
	out := []ScenarioAnchor{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, ScenarioAnchor{ID: pairs[i].(string), Price: pairs[i+1].(float64)})
	}
	return out
}

// The happy case still carries its basis and BOTH distances — never a bare id.
func TestNearestScenarioCarriesItsBasisAndDistance(t *testing.T) {
	l := ResolveScenarioLink(29600, anchors("S1", 29602.0, "S2", 29750.0), 4.0, 12.0)
	if l.Scenario == nil || *l.Scenario != "S1" {
		t.Fatalf("nearest should be S1, got %v", l.Scenario)
	}
	if l.Basis != ScenarioLinkPriceProximity {
		t.Errorf("basis = %q, want %q — the link must say HOW it was made", l.Basis, ScenarioLinkPriceProximity)
	}
	if l.DistPts == nil || *l.DistPts != 2.0 {
		t.Errorf("distance in points = %v, want 2", l.DistPts)
	}
	if l.DistDelta == nil || *l.DistDelta != 0.5 {
		t.Errorf("distance in Δ = %v, want 0.5 (2 pts / Δ 4)", l.DistDelta)
	}
}

// AMBIGUITY IS NULL. Two scenarios inside the band means the record cannot say
// which one the touch belongs to, and a guess would be the lie.
func TestTwoScenariosInsideTheBandResolveToNull(t *testing.T) {
	l := ResolveScenarioLink(29600, anchors("S1", 29602.0, "S2", 29596.0), 4.0, 12.0)
	if l.Scenario != nil {
		t.Fatalf("two scenarios inside the band must resolve to NULL, got %q", *l.Scenario)
	}
	if l.Basis != ScenarioLinkAmbiguous {
		t.Errorf("basis = %q, want %q", l.Basis, ScenarioLinkAmbiguous)
	}
}

// Nothing close is NOT the same as nothing authored. Both are NULL, and the
// basis is what tells them apart.
func TestNearestOutsideTheBandIsNullWithItsOwnReason(t *testing.T) {
	l := ResolveScenarioLink(29600, anchors("S1", 29900.0), 4.0, 12.0)
	if l.Scenario != nil {
		t.Fatalf("a scenario outside the band must not be linked, got %q", *l.Scenario)
	}
	if l.Basis != ScenarioLinkOutsideBand {
		t.Errorf("basis = %q, want %q", l.Basis, ScenarioLinkOutsideBand)
	}
	none := ResolveScenarioLink(29600, nil, 4.0, 12.0)
	if none.Basis != ScenarioLinkNoScenario {
		t.Errorf("no scenarios at all → %q, want %q", none.Basis, ScenarioLinkNoScenario)
	}
	if none.Basis == l.Basis {
		t.Error("'nothing authored' and 'nothing close' must be distinguishable")
	}
}

// Δ is resolved per read and can be 0 (fewer than 2 bars). A 0 Δ must not
// produce a divide-by-zero or a fabricated ratio — the points distance stands,
// the Δ distance is NULL.
func TestZeroDeltaLeavesTheDeltaDistanceNull(t *testing.T) {
	l := ResolveScenarioLink(29600, anchors("S1", 29602.0), 0, 12.0)
	if l.Scenario == nil {
		t.Fatal("a 0 Δ must not prevent the points-based link")
	}
	if l.DistPts == nil || *l.DistPts != 2.0 {
		t.Errorf("points distance must still be recorded, got %v", l.DistPts)
	}
	if l.DistDelta != nil {
		t.Errorf("with Δ unavailable the Δ-distance must be NULL, got %v", *l.DistDelta)
	}
}

// The link never claims identity. Nothing in the basis vocabulary reads as a
// fact about what the planner meant.
func TestBasisVocabularyNeverClaimsIdentity(t *testing.T) {
	for _, b := range []string{ScenarioLinkPriceProximity, ScenarioLinkAmbiguous, ScenarioLinkOutsideBand, ScenarioLinkNoScenario} {
		if b == "" {
			t.Fatal("every basis must be a stated reason, never empty")
		}
	}
	if ScenarioLinkPriceProximity != "price_proximity" {
		t.Errorf("the basis must name the heuristic outright, got %q", ScenarioLinkPriceProximity)
	}
}
