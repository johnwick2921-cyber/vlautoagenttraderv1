package store

import (
	"path/filepath"
	"testing"
	"time"
)

// W-FLIP-HOLD-ANCHOR — the chain reads the anchor resolver rests on.
func TestListVersionFactsAndLifecycleLogForPlan(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "fh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	pid := MakePlanIDForTrader("tr", "2026-09-16", "ASIA")
	born := time.Date(2026, 9, 16, 16, 35, 56, 0, time.FixedZone("CT", -5*3600))
	if _, err := st.Plan().AppendPlan(&PlanDB{PlanID: pid, TradeDate: "2026-09-16", Session: "ASIA", StrategyID: "tr", TriggerReason: "ASIA_scheduled_read", Doc: `{"bias":{"direction":"short"}}`, CreatedAt: born}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendPlan(&PlanDB{PlanID: pid, TradeDate: "2026-09-16", Session: "ASIA", StrategyID: "tr", TriggerReason: "level_event", Doc: `{}`, CreatedAt: born.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := st.Plan().UpdatePlanLifecycle(pid, 2, "dormant", "dormant:flip:x"); err != nil {
		t.Fatal(err)
	}
	facts, err := st.Plan().ListVersionFacts(pid)
	if err != nil || len(facts) != 2 {
		t.Fatalf("facts: %v %+v", err, facts)
	}
	if facts[0].Version != 1 || facts[0].BiasDirection != "short" || facts[0].TriggerReason != "ASIA_scheduled_read" {
		t.Fatalf("v1 fact wrong: %+v", facts[0])
	}
	if !facts[0].CreatedAt.Equal(born) {
		t.Fatalf("created_at must round-trip: got %v want %v", facts[0].CreatedAt, born)
	}
	if facts[1].BiasDirection != "" {
		t.Fatalf("a doc without bias must read as empty, got %q", facts[1].BiasDirection)
	}
	events, err := st.Plan().LifecycleLogForPlan(pid)
	if err != nil || len(events) != 1 || events[0].Version != 2 || events[0].Event != "dormant" {
		t.Fatalf("lifecycle log for plan: %v %+v", err, events)
	}
	if _, err := st.Plan().ListVersionFacts(""); err == nil {
		t.Fatal("empty plan_id must be refused")
	}
}
