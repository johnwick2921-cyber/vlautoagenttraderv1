package trader

import (
	"path/filepath"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// W4 — owner overlays reach the EXECUTOR: resolveActivePlanDoc folds overlays into
// plan_final, so the executor's PLAN BLOCK carries the owner's edit (not the base).
func TestW4OverlayReachesExecutor(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	base := `{"reasoning":"r","bias":{"direction":"long","conviction":"medium","flip_condition":"x"},` +
		`"levels":[{"price":30246,"label":"1h-LH","grade":"B","instruction":"fade"}],` +
		`"scenarios":[{"id":"S1","trigger":"t","condition":"reclaim","direction":"long","target_chain":[30300],"invalid":"i","quality":"A"}],` +
		`"no_trade":["first 5m"],"death_condition":"d"}`
	row := &store.PlanDB{
		PlanID: store.MakePlanID("2026-08-17", "NY"), StrategyID: "s", TradeDate: "2026-08-17",
		Session: "NY", TriggerReason: "scheduled", Lifecycle: "active", ModelID: "m", Doc: base,
	}
	ver, err := st.Plan().AppendPlan(row)
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}
	row.Version = ver

	// base resolution (no overlays): the executor sees the AI level.
	doc, ok := resolveActivePlanDoc(st, row)
	if !ok || doc.Levels[0].Label != "1h-LH" {
		t.Fatalf("base resolve wrong: ok=%v %+v", ok, doc.Levels)
	}
	block := kernel.RenderPlanBlock(doc, "NY")
	if !strings.Contains(block, "1h-LH") {
		t.Fatal("base executor block should show the AI level")
	}

	// owner overlay: replace the level with a graded D-4h demand.
	patch := `[{"op":"replace","path":"/levels/0","value":{"price":30240,"label":"D-4h","grade":"A","instruction":"reclaim-long"}}]`
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{OverlayID: "ov1", PlanID: row.PlanID, PlanVersion: ver, Patch: patch, Origin: "owner"}); err != nil {
		t.Fatalf("append overlay: %v", err)
	}

	// resolved plan_final: the executor now cites the OWNER's edit.
	doc2, ok := resolveActivePlanDoc(st, row)
	if !ok || doc2.Levels[0].Label != "D-4h" || doc2.Levels[0].Grade != "A" || doc2.Levels[0].Price != 30240 {
		t.Fatalf("overlay did not reach the executor doc: %+v", doc2.Levels)
	}
	block2 := kernel.RenderPlanBlock(doc2, "NY")
	if !strings.Contains(block2, "D-4h") || strings.Contains(block2, "1h-LH") {
		t.Fatalf("executor PLAN BLOCK must reflect the owner overlay, not the base:\n%s", block2)
	}
}

// A malformed/invalid overlay must NOT corrupt the executor plan — it falls back
// to the base doc (armor).
func TestW4BadOverlayFallsBackToBase(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	base := `{"reasoning":"r","bias":{"direction":"long","conviction":"medium","flip_condition":"x"},` +
		`"levels":[{"price":30000,"label":"PDH","grade":"A","instruction":"fade"}],` +
		`"scenarios":[{"id":"S1","trigger":"t","condition":"reclaim","direction":"long","target_chain":[30300],"invalid":"i","quality":"A"}],` +
		`"no_trade":["x"],"death_condition":"d"}`
	row := &store.PlanDB{PlanID: store.MakePlanID("2026-08-17", "NY"), StrategyID: "s", TradeDate: "2026-08-17", Session: "NY", Lifecycle: "active", Doc: base}
	ver, _ := st.Plan().AppendPlan(row)
	row.Version = ver
	// an overlay that would set an invalid bias direction (ValidatePlanDoc rejects).
	_, _ = st.Plan().AppendOverlay(&store.PlanOverlayDB{OverlayID: "bad", PlanID: row.PlanID, PlanVersion: ver, Patch: `[{"op":"replace","path":"/bias/direction","value":"sideways"}]`, Origin: "owner"})

	doc, ok := resolveActivePlanDoc(st, row)
	if !ok || doc.Bias.Direction != "long" {
		t.Fatalf("invalid overlay must fall back to base (long), got %q", doc.Bias.Direction)
	}
}
