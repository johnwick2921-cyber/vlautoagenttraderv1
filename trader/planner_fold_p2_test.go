package trader

import (
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

func foldTestTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	at.store = st
	at.id = "trader-1"
	return at, st
}

// TestPriorPlanLevelLinesFoldsOverlay (WAVE 1a-plan P2, planner :1551) — the
// prior-plan continuity lines must read the folded final doc. RED: 1 line with
// the owner-added level stored. GREEN: 2.
func TestPriorPlanLevelLinesFoldsOverlay(t *testing.T) {
	at, st := foldTestTrader(t)
	base := `{"reasoning":"p2","bias":{"direction":"neutral"},"death_condition":"flat","levels":[{"price":29100,"label":"PDL","grade":"A"}],"scenarios":[{"id":"S1","condition":"reject","direction":"long","quality":"A"}]}`
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: "p1", StrategyID: at.id, TradeDate: "2026-09-14", Session: "NY", Lifecycle: "active", Doc: base}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: "p1", PlanVersion: 1, OverlayID: "owner-add", Origin: "owner",
		Patch: `[{"op":"add","path":"/levels/-","value":{"price":29105,"label":"ONH","grade":"B"}}]`}); err != nil {
		t.Fatal(err)
	}
	row, _ := st.Plan().GetPlan("p1", 1)
	if lines := priorPlanLevelLines(at, row); len(lines) != 2 {
		t.Fatalf("the folded prior plan must yield 2 level lines, got %d: %v", len(lines), lines)
	}
}

// TestCarryMachineGradesFoldsOverlay (WAVE 1a-plan P2, planner :1621) — the
// machine-grade carry from the previous plan reads the folded final doc. RED:
// the overlay-added grade is not carried. GREEN: carried by price.
func TestCarryMachineGradesFoldsOverlay(t *testing.T) {
	at, st := foldTestTrader(t)
	prev := `{"reasoning":"p2","bias":{"direction":"neutral"},"death_condition":"flat","levels":[{"price":29100,"label":"PDL","grade":"A","machine_grade":"B"}],"scenarios":[{"id":"S1","condition":"reject","direction":"long","quality":"A"}]}`
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: "p1", StrategyID: at.id, TradeDate: "2026-09-14", Session: "NY", Lifecycle: "active", Doc: prev}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: "p1", PlanVersion: 1, OverlayID: "owner-add", Origin: "owner",
		Patch: `[{"op":"add","path":"/levels/-","value":{"price":29105,"label":"ONH","grade":"B","machine_grade":"A"}}]`}); err != nil {
		t.Fatal(err)
	}
	doc := kernel.PlanDoc{Levels: []kernel.PlanLevel{{Price: 29100}, {Price: 29105}}}
	at.carryMachineGrades("2026-09-14", "NY", &doc)
	if doc.Levels[1].MachineGrade != "A" {
		t.Fatalf("the overlay-added 29105 grade must be carried through the fold, got %q", doc.Levels[1].MachineGrade)
	}
	if doc.Levels[0].MachineGrade != "B" {
		t.Fatalf("the base 29100 grade must still carry, got %q", doc.Levels[0].MachineGrade)
	}
}
