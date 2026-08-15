package trader

import (
	"errors"
	"path/filepath"
	"testing"

	"nofx/store"
)

const validTraderPlanJSON = `{
  "reasoning": "Balance below PDH; fade edges, long the reclaim.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "2x5m < 15480"},
  "levels": [{"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade"}],
  "scenarios": [{"id": "S1", "trigger": "sweep 15480 reclaim", "condition": "sweep_reclaim", "direction": "long", "target_chain": [15550], "invalid": "2x5m<15470", "quality": "A"}],
  "no_trade": ["first 5m"],
  "death_condition": "acceptance above 15620",
  "day_type": "balance"
}`

func plannerTestTrader(t *testing.T) *AutoTrader {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	return &AutoTrader{
		id: "t1", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{StrategyConfig: &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}},
	}
}

func TestRunPlannerReadCoreSuccess(t *testing.T) {
	at := plannerTestTrader(t)
	ver, lc, err := at.runPlannerReadCore("NY", "2026-08-14", "deepseek-reasoner", "hashA",
		func() (string, error) { return validTraderPlanJSON, nil })
	if err != nil || ver != 1 || lc != "active" {
		t.Fatalf("success: ver=%d lc=%q err=%v", ver, lc, err)
	}
	got, _ := at.store.Plan().GetLatestPlanForSession("2026-08-14", "NY")
	if got == nil || got.Lifecycle != "active" || got.ModelID != "deepseek-reasoner" {
		t.Fatalf("stored plan wrong: %+v", got)
	}
}

func TestRunPlannerReadCoreFailClosed(t *testing.T) {
	at := plannerTestTrader(t)
	ver, lc, err := at.runPlannerReadCore("NY", "2026-08-14", "deepseek-reasoner", "hashB",
		func() (string, error) { return "", errors.New("timeout") })
	if err != nil || lc != "no_trade" {
		t.Fatalf("fail-closed: ver=%d lc=%q err=%v want no_trade", ver, lc, err)
	}
	got, _ := at.store.Plan().GetLatestPlanForSession("2026-08-14", "NY")
	if got == nil || got.Lifecycle != "no_trade" || got.TriggerReason != "planner_fail_closed" {
		t.Fatalf("fail-closed plan not written correctly: %+v", got)
	}
}

func TestRunPlannerReadCoreRetryThenSuccess(t *testing.T) {
	at := plannerTestTrader(t)
	n := 0
	_, lc, err := at.runPlannerReadCore("NY", "2026-08-14", "m", "hashC", func() (string, error) {
		n++
		if n < 3 {
			return "not json", nil // 2 invalid → retried
		}
		return validTraderPlanJSON, nil
	})
	if err != nil || lc != "active" {
		t.Fatalf("retry-then-success: lc=%q err=%v (calls=%d)", lc, err, n)
	}
	if n != 3 {
		t.Fatalf("expected 3 attempts (1+2 retries), got %d", n)
	}
}
