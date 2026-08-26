package store

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestDayPlanClockFieldsPersistThroughRow is the P2.3/P2.5 config-truth proof:
// save → row → reload → read. The clock fields (last_entry / eod_flat) and
// plan_enabled survive a real DB round-trip via the strategy config column.
func TestDayPlanClockFieldsPersistThroughRow(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	cfg := GetDefaultStrategyConfig("en")
	cfg.DayPlan = DefaultDayPlanConfig()
	cfg.DayPlan.PlanEnabled = true
	cfg.DayPlan.LastEntryCT = "13:15"
	cfg.DayPlan.EODFlatCT = "14:30"

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := st.Strategy().Create(&Strategy{
		ID: "s1", UserID: "u1", Name: "day-plan", Config: string(raw),
	}); err != nil {
		t.Fatalf("create strategy: %v", err)
	}

	// Reload from the row and read back.
	got, err := st.Strategy().Get("u1", "s1")
	if err != nil || got == nil {
		t.Fatalf("get strategy: %v", err)
	}
	var back StrategyConfig
	if err := json.Unmarshal([]byte(got.Config), &back); err != nil {
		t.Fatalf("unmarshal reloaded config: %v", err)
	}
	if back.DayPlan == nil {
		t.Fatalf("day_plan lost on reload")
	}
	if !back.DayPlan.PlanEnabled {
		t.Fatalf("plan_enabled=true did not persist")
	}
	if back.DayPlan.LastEntryCT != "13:15" || back.DayPlan.EODFlatCT != "14:30" {
		t.Fatalf("clock fields not persisted: last=%q eod=%q", back.DayPlan.LastEntryCT, back.DayPlan.EODFlatCT)
	}
}
