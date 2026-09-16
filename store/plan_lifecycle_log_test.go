package store

import (
	"path/filepath"
	"testing"
)

// LIFECYCLE LOG (data-integrity wave, D3) — E3.
//
// UpdatePlanLifecycle overwrote trigger_reason with the lifecycle marker, so a
// plan row could answer "why was this parked" OR "why was this authored", never
// both — and the second answer was destroyed by the first. Measured on the live
// store, real rows whose authoring trigger is gone:
//
//	2026-08-27:ASIA … v7  active   trigger_reason "rearmed:2x5m close back below 29678.25 …"
//	2026-08-28:NY   … v2  dormant  trigger_reason "dormant:death:death-condition: 15m_close …"
//	2026-08-26:ASIA … v10 dormant  trigger_reason "dormant:flip:flip-condition: 15m_close …"
//
// trigger_reason is the AUTHORING trigger and nothing else now; every lifecycle
// transition appends to plan_lifecycle_log instead.
func TestPlanLifecycleLogKeepsTheAuthoringTrigger(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := st.Plan()

	row := &PlanDB{
		PlanID: "2026-09-03:NY:t1", Version: 1, StrategyID: "t1",
		TradeDate: "2026-09-03", Session: "NY",
		TriggerReason: "NY_scheduled_read", Lifecycle: "active",
		Doc: `{"bias":{"direction":"neutral"}}`,
	}
	if _, err := ps.AppendPlan(row); err != nil {
		t.Fatalf("create: %v", err)
	}

	// park it, then re-arm it — the shapes the live rows took
	if err := ps.UpdatePlanLifecycle(row.PlanID, 1, "dormant", "dormant:death:death-condition: 2x5m close above 29631.14"); err != nil {
		t.Fatalf("park: %v", err)
	}
	if err := ps.UpdatePlanLifecycle(row.PlanID, 1, "active", "rearmed:2x5m close back below 29678.25"); err != nil {
		t.Fatalf("re-arm: %v", err)
	}

	got, err := ps.GetPlan(row.PlanID, 1)
	if err != nil || got == nil {
		t.Fatalf("read back: %v", err)
	}
	if got.TriggerReason != "NY_scheduled_read" {
		t.Errorf("trigger_reason = %q — the AUTHORING trigger must survive every lifecycle transition", got.TriggerReason)
	}
	if got.Lifecycle != "active" {
		t.Errorf("lifecycle = %q, want active", got.Lifecycle)
	}

	// and the park reason is readable from the log, in order
	events, err := ps.LifecycleLog(row.PlanID, 1)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("log has %d events, want 2 (park, re-arm)", len(events))
	}
	if events[0].Event != "dormant" || events[0].Reason == "" {
		t.Errorf("first event = %+v, want the park with its reason", events[0])
	}
	if events[1].Event != "active" {
		t.Errorf("second event = %+v, want the re-arm", events[1])
	}
	for _, e := range events {
		if e.At.IsZero() {
			t.Errorf("event %q has no timestamp — a log that cannot be ordered is not a log", e.Event)
		}
	}
}

// A lifecycle write to a row that does not exist must still error, as before.
func TestPlanLifecycleLogMissingRowStillErrors(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Plan().UpdatePlanLifecycle("nope", 1, "dormant", "x"); err == nil {
		t.Fatal("a lifecycle update on a missing plan row must error")
	}
}
