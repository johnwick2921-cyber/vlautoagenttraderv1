package store

import (
	"path/filepath"
	"testing"
)

// W-FLIP-REREAD BLOCKER 3 (2026-09-17): the structure_flip supersede must be a
// compare-and-set from "dormant". A row that was re-armed meanwhile (dormant →
// active by the close-back predicate) is NOT superseded, the refusal is
// reported as false (not an error), and the newest version still governs at
// read time via GetLatestPlanForTraderSession (ORDER BY version DESC).
func TestUpdatePlanLifecycleIfCompareAndSet(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "cas.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	ps := st.Plan()
	td, sess, id := "2026-09-17", "ASIA", "trader-cas"
	pid := MakePlanIDForTrader(id, td, sess)
	if _, err := ps.AppendPlan(&PlanDB{PlanID: pid, TradeDate: td, Session: sess, StrategyID: id, Lifecycle: "active", Doc: "{}"}); err != nil {
		t.Fatal(err)
	}
	if err := ps.UpdatePlanLifecycle(pid, 1, "dormant", "dormant:flip:flip-condition: 2x5m close above 100 → bias long"); err != nil {
		t.Fatal(err)
	}
	// The flip read lands v2 active while v1 sleeps.
	if v, err := ps.AppendPlan(&PlanDB{PlanID: pid, TradeDate: td, Session: sess, StrategyID: id, Lifecycle: "active", Doc: "{}"}); err != nil || v != 2 {
		t.Fatalf("append v2: v=%d err=%v", v, err)
	}

	// Happy path: still dormant → moves, log carries the reason.
	moved, err := ps.UpdatePlanLifecycleIf(pid, 1, "dormant", "superseded:flip", "superseded:flip:v2")
	if err != nil || !moved {
		t.Fatalf("dormant → superseded:flip: moved=%v err=%v", moved, err)
	}
	if row, _ := ps.GetPlan(pid, 1); row == nil || row.Lifecycle != "superseded:flip" {
		t.Fatalf("v1 lifecycle after CAS: %+v", row)
	}
	if log, _ := ps.LifecycleLog(pid, 1); len(log) == 0 || log[len(log)-1].Reason != "superseded:flip:v2" || log[len(log)-1].Event != "superseded:flip" {
		t.Fatalf("lifecycle log after CAS: %+v", log)
	}

	// Refusal path: a SECOND CAS from "dormant" finds the row elsewhere → false, nil, row untouched.
	moved, err = ps.UpdatePlanLifecycleIf(pid, 1, "dormant", "superseded:flip", "superseded:flip:v2")
	if err != nil || moved {
		t.Fatalf("CAS on a row no longer dormant must refuse without error: moved=%v err=%v", moved, err)
	}

	// The race the method exists for: v1 re-armed (active) before the supersede.
	if err := ps.UpdatePlanLifecycle(pid, 1, "active", "rearmed:2x5m close back below 100"); err != nil {
		t.Fatal(err)
	}
	moved, err = ps.UpdatePlanLifecycleIf(pid, 1, "dormant", "superseded:flip", "superseded:flip:v2")
	if err != nil || moved {
		t.Fatalf("re-armed row must not be superseded: moved=%v err=%v", moved, err)
	}
	v1, _ := ps.GetPlan(pid, 1)
	v2, _ := ps.GetPlan(pid, 2)
	if v1 == nil || v1.Lifecycle != "active" || v2 == nil || v2.Lifecycle != "active" {
		t.Fatalf("both rows must stand as written: v1=%+v v2=%+v", v1, v2)
	}
	// Read-time winner: the newest version, regardless of v1's re-arm.
	latest, err := ps.GetLatestPlanForTraderSession(td, sess, id)
	if err != nil || latest == nil || latest.Version != 2 {
		t.Fatalf("newest active version must govern at read time: %+v err=%v", latest, err)
	}
	// No such row / bad args.
	if moved, err := ps.UpdatePlanLifecycleIf(pid, 9, "dormant", "superseded:flip", "x"); err != nil || moved {
		t.Fatalf("missing row: moved=%v err=%v", moved, err)
	}
	if _, err := ps.UpdatePlanLifecycleIf(pid, 1, "", "superseded:flip", "x"); err == nil {
		t.Fatal("empty from must be refused")
	}
}
