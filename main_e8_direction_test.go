package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// TestE8ScenarioDirectionFoldsOverlay (WAVE 1a-plan P2) — the E8 short-row
// backfill resolves the scenario direction through the ONE fold: an owner
// overlay that flips the direction must be visible to the recompute. RED on
// the base-only reader: direction stays "long" with the flip overlay stored.
func TestE8ScenarioDirectionFoldsOverlay(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	base := kernel.PlanDoc{
		Reasoning:      "wave-1a-p2-e8",
		Bias:           kernel.PlanBias{Direction: "neutral"},
		DeathCondition: "flat",
		Scenarios: []kernel.PlanScenario{
			{ID: "S1", Condition: "reject", Direction: "long", Quality: "A"},
		},
	}
	blob, _ := json.Marshal(&base)
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: "p1", StrategyID: "trader-1", TradeDate: "2026-09-14", Session: "NY", Lifecycle: "active", Doc: string(blob)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: "p1", PlanVersion: 1, OverlayID: "owner-flip", Origin: "owner",
		Patch: `[{"op":"replace","path":"/scenarios/0/direction","value":"short"}]`}); err != nil {
		t.Fatal(err)
	}
	dir, ok := e8ScenarioDirection(st, "p1", 1, "S1")
	if !ok || dir != "short" {
		t.Fatalf("the folded direction must read short, got %q ok=%v", dir, ok)
	}
}
