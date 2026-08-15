package trader

import "testing"

func TestResolvePlannerModelID(t *testing.T) {
	// Empty binding → primary.
	if id, prim := resolvePlannerModelID("", "deepseek"); id != "deepseek" || !prim {
		t.Fatalf("empty → primary: got %q/%v", id, prim)
	}
	if id, prim := resolvePlannerModelID("   ", "deepseek"); id != "deepseek" || !prim {
		t.Fatalf("whitespace → primary: got %q/%v", id, prim)
	}
	// Explicit binding → the pinned ID, not the primary.
	if id, prim := resolvePlannerModelID("deepseek-reasoner", "deepseek"); id != "deepseek-reasoner" || prim {
		t.Fatalf("pinned: got %q/%v want deepseek-reasoner/false", id, prim)
	}
}

func TestResolvePlannerClientEmptyUsesPrimary(t *testing.T) {
	// Empty planner_model → returns the primary client + primary model id (no
	// registry call). mcpClient nil is fine here — we only assert the model id.
	at := mkTrader("ninjatrader", boolp(true), "5m")
	at.aiModel = "deepseek"
	_, id := at.resolvePlannerClient()
	if id != "deepseek" {
		t.Fatalf("empty binding should use primary model id, got %q", id)
	}
	// A set planner_model returns that pinned id (registry may or may not resolve
	// a client; on failure it falls back to primary — either way not a crash).
	at.config.StrategyConfig.DayPlan.PlannerModel = "deepseek"
	_, id2 := at.resolvePlannerClient()
	if id2 == "" {
		t.Fatalf("resolved planner model id must be non-empty")
	}
}
