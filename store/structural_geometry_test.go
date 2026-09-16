package store

import (
	"path/filepath"
	"testing"
)

func TestStructuralGeometryPersistenceAndCounts(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "geometry.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	r := StructuralGeometryRecord{TraderID: "t1", PlanID: "P1", Version: 1, Scenario: "S1", Leg: 1, TradeDate: "2026-09-12", Reason: "rr", Quantity: 0, Entry: 100}
	for i := 0; i < 2; i++ {
		if err := st.SaveStructuralGeometry(r); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := st.StructuralGeometryCounts([]string{"t1"}, r.TradeDate)
	if err != nil || counts["rr"] != 1 {
		t.Fatalf("repeated same refusal must not inflate count: %v %v", counts, err)
	}
	// Production records pending gates before their final decision each cycle.
	for i := 0; i < 2; i++ {
		r.Reason = "pending_gates"
		if err := st.SaveStructuralGeometry(r); err != nil {
			t.Fatal(err)
		}
		r.Reason = "rr"
		if err := st.SaveStructuralGeometry(r); err != nil {
			t.Fatal(err)
		}
	}
	counts, err = st.StructuralGeometryCounts([]string{"t1"}, r.TradeDate)
	if err != nil || counts["rr"] != 1 {
		t.Fatalf("pending gate cycle inflated distinct refusals: %v %v", counts, err)
	}
	r.Reason = "admitted"
	r.Quantity = 1
	if err := st.SaveStructuralGeometry(r); err != nil {
		t.Fatal(err)
	}
	counts, _ = st.StructuralGeometryCounts([]string{"t1"}, r.TradeDate)
	if counts["rr"] != 1 {
		t.Fatal("later admission erased earlier refusal")
	}
	rows, err := st.StructuralGeometryFor("t2", "P1", 1)
	if err != nil || len(rows) != 0 {
		t.Fatalf("cross-trader record leak: %v %v", rows, err)
	}
	rows, err = st.StructuralGeometryFor("t1", "P1", 2)
	if err != nil || len(rows) != 0 {
		t.Fatalf("cross-version record leak: %v %v", rows, err)
	}
	rows, err = st.StructuralGeometryFor("t1", "P1", 1)
	if err != nil || len(rows) != 1 || rows[0].Reason != "admitted" {
		t.Fatalf("current record unavailable: %v %v", rows, err)
	}
}

func TestStructuralPolicyMeasuredBufferAndMissingValues(t *testing.T) {
	c := StrategyConfig{DayPlan: &DayPlanConfig{}}
	p := ResolveStructuralStop(&c, "MNQ")
	if !p.BufferKnown || p.BufferPoints != 4.5 {
		t.Fatalf("measured buffer must resolve independently of daily risk settings: %+v", p)
	}
	zero := 0.0
	c.DayPlan.StructuralStop = &StructuralStopConfig{BufferPoints: &zero}
	if ResolveStructuralStop(&c, "MNQ").BufferKnown {
		t.Fatal("invalid explicit buffer silently became default")
	}
}
