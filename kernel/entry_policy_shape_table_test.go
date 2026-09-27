package kernel

import (
	"strings"
	"testing"
)

// FIX-PLANNER (2026-09-26) item 3 — the legal (condition × policy × legs)
// shapes are stated as a compact table GENERATED from the validator's own
// entry-law table (plannedOrderLegal + KnownConditions), never retyped by hand.
// Guard: the prompt sentence and the repair law CONTAIN the generated table,
// so a change to the law is a change to both renderings.

func TestFpEntryPolicyShapeTableGeneratedFromLaw(t *testing.T) {
	table := EntryPolicyShapeTable()
	for _, want := range []string{
		"LEGAL ENTRY-POLICY SHAPES",
		"market_in_zone",
		"planned_order",
		"leg 0 ONLY",
		"reject",
		"fvg_entry",
	} {
		if !strings.Contains(table, want) {
			t.Fatalf("generated table lacks %q:\n%s", want, table)
		}
	}
	// Every planned_order-legal condition from the LAW map appears in the table
	// (a row the map adds must appear without a hand edit).
	for c := range plannedOrderLegal {
		if !strings.Contains(table, c) {
			t.Fatalf("table misses legal condition %q:\n%s", c, table)
		}
	}
	// The prompt sentence reads the table — no hand-rettyped drift.
	frag := entryPolicyPlannedOrderFrag(true)
	if !strings.Contains(frag, EntryPolicyShapeTable()) {
		t.Fatalf("the planned_order prompt sentence must carry the generated table:\n%s", frag)
	}
	// The repair law reads the same table, routed on the validator's own
	// refusal text (EntryPolicyLegal's error).
	verr := EntryPolicyLegal("breakdown_continue", EntryPolicyPlannedOrder, 0)
	if verr == nil {
		t.Fatal("fixture: planned_order on breakdown_continue must be illegal")
	}
	p := BuildPlannerRepairPrompt("{}", verr.Error(), nil)
	if !strings.Contains(p, EntryPolicyShapeTable()) {
		t.Fatalf("the repair prompt must carry the generated entry-policy table:\n%s", p)
	}
}
