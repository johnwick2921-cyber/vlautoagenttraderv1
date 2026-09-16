package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// F3 (LONDON-FORENSICS 2026-08-28) — armed-fill lineage on materialization.
//
// Live proof of the bug class: position #567 (the first live armed fill) was
// materialized by reconcile AFTER the fill-time lineage stamp ran, so it kept
// plan_version 0 / plan_band "" / adherence grade F. The repair pass stamps
// the plan linkage back from the armed ledger (match: trader + side + entry
// within one tick), idempotently.

func TestRepairArmedLineageStampsMaterializedPosition(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "fx3.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// The armed ledger's filled row — row 5's shape from the live night:
	// ASIA v12 S1 short, entry 29621.01, filled.
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: "trader-1", PlanID: "2026-08-27:ASIA:trader-1", Version: 12,
		Session: "ASIA", Scenario: "S1", Side: "short",
		EntryPx: 29621.01, StopPx: 29642, TargetPx: 29576.5,
		State: "filled", StateReason: "fill@29621.00", EntryClass: "armed_fill",
		SignalID: "f14ea5dd", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// The materialized position — #567's shape: no plan linkage at all.
	pos := &store.TraderPosition{
		TraderID: "trader-1", Symbol: "MNQ", Side: "SHORT", Account: "Sim101",
		ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
		EntryPrice: 29621.00, EntryTime: time.Now().Add(-time.Hour).UnixMilli(),
		Status: "CLOSED", Source: "reconcile", ExitPrice: 29642,
		ExitTime: time.Now().Add(-30 * time.Minute).UnixMilli(), RealizedPnL: -42,
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatal(err)
	}

	if n := RepairArmedLineage(st, "trader-1"); n != 1 {
		t.Fatalf("RepairArmedLineage stamped %d rows, want 1 (the materialized #567 class)", n)
	}
	rows, err := st.Position().ListUnlinked("trader-1", 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("unlinked rows after repair = %d err=%v, want 0", len(rows), err)
	}
	closed, err := st.Position().GetOpenPositions("trader-1")
	if err != nil || len(closed) != 1 {
		t.Fatalf("open rows: %+v err=%v", closed, err)
	}
	p := closed[0]
	if p.PlanVersion != 12 || p.CitedScenarioID != "S1" || p.PlanBand != "armed_fill" || p.PlanID != "2026-08-27:ASIA:trader-1" {
		t.Fatalf("lineage stamp wrong: version=%d scenario=%q band=%q plan=%q", p.PlanVersion, p.CitedScenarioID, p.PlanBand, p.PlanID)
	}
	// Idempotent: a second pass stamps nothing new.
	if n := RepairArmedLineage(st, "trader-1"); n != 0 {
		t.Fatalf("second repair pass stamped %d rows, want 0 (idempotent)", n)
	}
}

func TestStampArmedLineageSkipsMismatchedEntry(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "fx3b.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: "trader-1", PlanID: "2026-08-27:ASIA:trader-1", Version: 12,
		Session: "ASIA", Scenario: "S1", Side: "short",
		EntryPx: 29621.01, StopPx: 29642, TargetPx: 29576.5,
		State: "filled", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	pos := &store.TraderPosition{
		TraderID: "trader-1", Symbol: "MNQ", Side: "SHORT",
		ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
		EntryPrice: 29500, EntryTime: time.Now().UnixMilli(), Status: "OPEN",
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatal(err)
	}
	if stamped, _ := StampArmedLineageIfMatched(st, "trader-1", pos.ID, "MNQ", "SHORT", 29500); stamped {
		t.Fatal("a 121-pt entry mismatch must NOT match the ledger fill (tick window only)")
	}
}
