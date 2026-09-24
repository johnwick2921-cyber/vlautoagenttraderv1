package trader

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// TestFollowPlanBiasFoldsOverlay (WAVE 1a-plan P2) — followPlanBias reads the
// plan's frozen bias for a follow-plan record; it must read the FOLDED final
// doc (CTO spot-check 2026-09-24 lists follow_plan_wiring.go:51 as fold), not
// the base alone. An owner overlay that flips bias.direction must be visible.
// RED on the base-only reader: bias stays "long" with the overlay stored.
func TestFollowPlanBiasFoldsOverlay(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	at.store = st
	at.id = "trader-1"

	base := kernel.PlanDoc{
		Reasoning:      "wave-1a-p2-follow",
		Bias:           kernel.PlanBias{Direction: "long"},
		DeathCondition: "flat",
		Scenarios: []kernel.PlanScenario{
			{ID: "S1", Condition: "reclaim", Direction: "long", Quality: "A"},
		},
	}
	baseJSON, _ := json.Marshal(&base)
	planID := "2026-09-14:NY"
	_, err = st.Plan().AppendPlan(&store.PlanDB{
		PlanID:        planID,
		StrategyID:    at.id,
		TradeDate:     "2026-09-14",
		Session:       "NY",
		TriggerReason: "wave-1a-p2-follow",
		Lifecycle:     "active",
		ModelID:       "deepseek-v4-pro",
		PromptHash:    "deadbeef",
		Doc:           string(baseJSON),
	})
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}
	_, err = st.Plan().AppendOverlay(&store.PlanOverlayDB{
		PlanID:      planID,
		PlanVersion: 1,
		OverlayID:   "owner-flip-bias",
		Origin:      "owner",
		Patch:       `[{"op":"replace","path":"/bias/direction","value":"short"}]`,
	})
	if err != nil {
		t.Fatalf("append overlay: %v", err)
	}

	if got := at.followPlanBias(map[string]string{}, planID, 1); got != "short" {
		t.Fatalf("the folded final doc's bias must read short, got %q", got)
	}
}
