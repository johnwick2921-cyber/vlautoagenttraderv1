package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// PRE-SUNDAY F2 (2026-08-28) — the repair matcher missed #568 and #570 because
// it keyed on the ledger row's entry_px, which DRIFTS on re-arm: the v6 re-spec
// overwrote the v1 entry that actually filled (#568: row entry 29702 vs real
// fill 29642.00; #570: row entry 29480 vs fill 29463.25). The matcher now keys
// on fill_price, then the "fill@…" reason — the fill the ledger recorded at
// fill time.
func TestStampArmedLineageMatchesFillFromReason(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "f2.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	// #568 shape: short fill at 29642.00, but the row was re-specced to 29702.
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: "t", PlanID: "2026-08-28:LONDON:t", Version: 6, Session: "LONDON",
		Scenario: "S1", Side: "short", EntryPx: 29702.0, StopPx: 29726.7, TargetPx: 29644.38,
		State: "filled", StateReason: "fill@29642.00", SignalID: "sig-568",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	pos := &store.TraderPosition{
		TraderID: "t", Symbol: "MNQ", Side: "SHORT", Account: "Sim101",
		ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
		EntryPrice: 29642.00, EntryTime: now, Status: "CLOSED", Source: "reconcile",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatal(err)
	}
	stamped, sig := StampArmedLineageIfMatched(st, "t", pos.ID, "MNQ", "SHORT", 29642.00)
	if !stamped || sig != "sig-568" {
		t.Fatalf("#568 shape: stamped=%v sig=%q, want true sig-568", stamped, sig)
	}
}

func TestStampArmedLineageMatchesFillPriceColumn(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "f2b.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	// #570 shape: long fill at 29463.25; fill_price written at fill time.
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: "t", PlanID: "2026-08-28:NY:t", Version: 5, Session: "NY",
		Scenario: "S2", Side: "long", EntryPx: 29480.0, StopPx: 29430.0, TargetPx: 29588.25,
		State: "filled", StateReason: "fill@29463.25", SignalID: "sig-570",
		FillPrice: 29463.25, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	pos := &store.TraderPosition{
		TraderID: "t", Symbol: "MNQ", Side: "LONG", Account: "SimAccount1",
		ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
		EntryPrice: 29463.25, EntryTime: now, Status: "CLOSED", Source: "reconcile",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatal(err)
	}
	stamped, sig := StampArmedLineageIfMatched(st, "t", pos.ID, "MNQ", "LONG", 29463.25)
	if !stamped || sig != "sig-570" {
		t.Fatalf("#570 shape: stamped=%v sig=%q, want true sig-570", stamped, sig)
	}
}

func TestStampArmedLineageRejectsDriftedEntryWithoutFill(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "f2c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	// Drifted entry_px and NO fill record at all → must NOT stamp (no match).
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: "t", PlanID: "2026-08-28:LONDON:t", Version: 6, Session: "LONDON",
		Scenario: "S1", Side: "short", EntryPx: 29702.0, StopPx: 29726.7, TargetPx: 29644.38,
		State: "filled", StateReason: "filled", SignalID: "sig-x",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	pos := &store.TraderPosition{
		TraderID: "t", Symbol: "MNQ", Side: "SHORT", Account: "Sim101",
		ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
		EntryPrice: 29642.00, EntryTime: now, Status: "CLOSED", Source: "reconcile",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatal(err)
	}
	if stamped, _ := StampArmedLineageIfMatched(st, "t", pos.ID, "MNQ", "SHORT", 29642.00); stamped {
		t.Fatal("no fill record on the row → the drifted entry must NOT match")
	}
}
