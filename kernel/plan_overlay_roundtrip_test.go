package kernel_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// P5.1 — the OVERLAY ROUND-TRIP: an owner appends an RFC-6902 overlay to a stored
// plan, and resolving (base doc + overlays) reproduces the exact plan_final that
// GET /api/plan/today serves. Proves the store↔kernel path end-to-end.

func TestOverlayRoundTripThroughStore(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	ps := st.Plan()

	base := `{"reasoning":"r","bias":{"direction":"long","conviction":"medium","flip_condition":"2x5m<30148"},` +
		`"levels":[{"price":30246.5,"label":"1h-LH","grade":"B","instruction":"fade"}],` +
		`"scenarios":[{"id":"S1","trigger":"reclaim ONL","condition":"reclaim","direction":"long","target_chain":[30300],"invalid":"below 30148","quality":"A"}],` +
		`"no_trade":["first 5m"],"death_condition":"2x5m accept < 30148"}`

	plan := &store.PlanDB{
		PlanID: store.MakePlanID("2026-08-17", "NY"), StrategyID: "s1",
		TradeDate: "2026-08-17", Session: "NY", TriggerReason: "scheduled",
		Lifecycle: "active", ModelID: "deepseek-reasoner", Doc: base,
	}
	ver, err := ps.AppendPlan(plan)
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}

	// Owner edit: replace the weak 1h supply with a graded 4h demand + retarget S1.
	patch := `[` +
		`{"op":"test","path":"/levels/0/label","value":"1h-LH"},` +
		`{"op":"replace","path":"/levels/0","value":{"price":30240,"label":"D-4h","grade":"A","instruction":"reclaim-long"}},` +
		`{"op":"add","path":"/scenarios/0/target_chain/-","value":30350}]`
	if _, err := ps.AppendOverlay(&store.PlanOverlayDB{
		OverlayID: "ov1", PlanID: plan.PlanID, PlanVersion: ver, Patch: patch, Origin: "owner",
	}); err != nil {
		t.Fatalf("append overlay: %v", err)
	}

	// Resolve exactly as handlePlanToday does.
	overlays, err := ps.ListOverlays(plan.PlanID, ver)
	if err != nil {
		t.Fatalf("list overlays: %v", err)
	}
	patches := make([]string, 0, len(overlays))
	for _, ov := range overlays {
		patches = append(patches, ov.Patch)
	}
	final, applyErrs := kernel.ApplyOverlayPatches([]byte(base), patches)
	if len(applyErrs) != 0 {
		t.Fatalf("unexpected apply errors: %v", applyErrs)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal(final, &doc); err != nil {
		t.Fatalf("plan_final not a PlanDoc: %v", err)
	}
	if err := kernel.ValidatePlanDoc(&doc); err != nil {
		t.Fatalf("plan_final must validate: %v", err)
	}

	if len(doc.Levels) != 1 || doc.Levels[0].Label != "D-4h" || doc.Levels[0].Grade != "A" || doc.Levels[0].Price != 30240 {
		t.Fatalf("owner level edit not reflected: %+v", doc.Levels)
	}
	if len(doc.Scenarios[0].TargetChain) != 2 || doc.Scenarios[0].TargetChain[1] != 30350 {
		t.Fatalf("scenario retarget not reflected: %+v", doc.Scenarios[0].TargetChain)
	}
}

// TestOverlayTestOpConcurrencyRejects proves the test-op guard: an overlay whose
// test no longer matches the current plan_final is rejected atomically (the
// concurrency control a second editor's stale patch would hit).
func TestOverlayTestOpConcurrencyRejects(t *testing.T) {
	base := `{"reasoning":"r","bias":{"direction":"long","conviction":"medium","flip_condition":"x"},` +
		`"levels":[{"price":100,"label":"PDH","grade":"A","instruction":"fade"}],` +
		`"scenarios":[{"id":"S1","trigger":"t","condition":"reclaim","direction":"long","target_chain":[110],"invalid":"i","quality":"A"}],` +
		`"no_trade":["x"],"death_condition":"d"}`
	// stale patch expects the OLD label "ONH" that isn't there → strict apply fails.
	stale := `[{"op":"test","path":"/levels/0/label","value":"ONH"},{"op":"replace","path":"/levels/0/grade","value":"C"}]`
	if _, err := kernel.ApplyPatchStrict([]byte(base), stale); err == nil {
		t.Fatal("stale test-op overlay must be rejected (409 concurrency conflict)")
	}
}
