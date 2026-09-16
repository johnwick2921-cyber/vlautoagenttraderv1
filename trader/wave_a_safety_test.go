package trader

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	nt "nofx/provider/ninjatrader"
	"nofx/store"
)

// E8 / A10 / CLASS 23 — A MEASURING INSTRUMENT MAY NEVER STOP THE TRADING LOOP.
// Every writer this wave adds is telemetry. A closed database, a nil dependency
// or a degenerate input must WARN and return; nothing may propagate.
func TestWaveARecordersNeverPanicTheTradingLoop(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "safety.db"))
	if err != nil {
		t.Fatal(err)
	}
	at := &AutoTrader{id: "hoang", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{
			DayPlan: &store.DayPlanConfig{PlanEnabled: true},
		}}}

	// The database is CLOSED underneath every writer — the harshest realistic
	// failure. None of these may panic and none may return an error to a caller
	// who could act on it.
	_ = st.Close()

	if r := recoverOf(func() {
		at.recordAcceptedRisk(store.ArmedOrderDB{SignalID: "sig", Side: "SHORT", StopPx: 29351.63},
			nt.OrderUpdatePayload{SignalID: "sig", OrderName: "sig-sl", State: "accepted"})
	}); r != nil {
		t.Fatalf("recordAcceptedRisk panicked through to the loop: %v", r)
	}
	if r := recoverOf(func() {
		at.recordDetectorOutputs("MNQ", "P1", "NY", 1, nil, nil, 29000, 10, 2.0, 12, time.Now(), nil)
	}); r != nil {
		t.Fatalf("recordDetectorOutputs panicked through to the loop: %v", r)
	}
	// A nil store must be a no-op, not a nil dereference.
	bare := &AutoTrader{id: "hoang"}
	if r := recoverOf(func() {
		bare.recordAcceptedRisk(store.ArmedOrderDB{}, nt.OrderUpdatePayload{})
	}); r != nil {
		t.Fatalf("recordAcceptedRisk with no store panicked: %v", r)
	}
}

func recoverOf(fn func()) (out any) {
	defer func() { out = recover() }()
	fn()
	return nil
}

// E10 — ONE CLOCK. Every timestamp this wave writes is epoch MILLISECONDS UTC.
// The trap it guards: on armed_orders, created_at carries a -05:00 offset while
// updated_at is UTC, and reading them together already produced one false
// "after the boot" conclusion. This wave's columns mix nothing.
func TestWaveAWritesEpochMillisUTCOnly(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "clock.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	before := time.Now().UTC().UnixMilli()
	if err := st.AcceptedRisk().Append(&store.AcceptedRisk{
		TraderID: "hoang", SignalID: "sig", OrderName: "sig-sl", Side: "SHORT",
	}); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC().UnixMilli()

	rows, err := st.AcceptedRisk().ForSignal("sig")
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected one row: %v", err)
	}
	ts := rows[0].AcceptedAtMs
	if ts < before || ts > after {
		t.Fatalf("accepted_at_ms is not an epoch-ms UTC stamp taken at write time: got %d, expected within [%d, %d]", ts, before, after)
	}
	// Epoch MILLIS, not seconds — the trap trader_positions.updated_at fell into,
	// where 68 of the 71 day-plan-era rows hold SECONDS in a column documented
	// as milliseconds, so an era filter silently returns 3 rows instead of 71.
	const year2001Ms = 1000000000000
	if ts < year2001Ms {
		t.Fatalf("accepted_at_ms looks like epoch SECONDS (%d) in a column that must hold MILLISECONDS", ts)
	}
	// And no ISO/offset string may reach the epoch column.
	var raw string
	if err := st.GormDB().Raw("SELECT CAST(accepted_at_ms AS TEXT) FROM accepted_risk LIMIT 1").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"-05:00", "+00:00", "T", "Z"} {
		if len(raw) > 0 && strings.Contains(raw, bad) {
			t.Fatalf("an ISO/offset string reached the epoch-ms column: %q contains %q", raw, bad)
		}
	}
}

// A31 — AN ACCEPTANCE RECORDS AND CHANGES NOTHING.
//
// This pin exists because the first draft of the D4 hook accidentally replaced
// `case "filled", "partfilled":` with the new accepted case, which put the whole
// fill body — SetState("filled"), SetFillPrice, materializeArmedEntry — under an
// ACCEPTANCE. A resting order being acknowledged by the broker would have
// materialized a position. The existing transition test caught it; this pin
// states the rule directly so it cannot come back.
func TestAcceptedOrderUpdateChangesNoLedgerState(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	ledger := st.ArmedOrders()
	if err := ledger.UpsertArm(&store.ArmedOrderDB{
		TraderID: at.id, PlanID: "2026-09-05:NY:t", Version: 1, Session: "NY",
		Scenario: "S1", Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110,
		State: "armed", SignalID: "sig-a",
	}); err != nil {
		t.Fatal(err)
	}
	rows, _ := ledger.ListForPlan("2026-09-05:NY:t")
	before := rows[0]

	at.onArmedOrderUpdate(nt.OrderUpdatePayload{
		SignalID: "sig-a", OrderName: "sig-a", State: "accepted", Quantity: 1,
	}, ledger)

	rows, _ = ledger.ListForPlan("2026-09-05:NY:t")
	after := rows[0]
	if after.State != before.State {
		t.Fatalf("an acceptance must not change ledger state: %q → %q", before.State, after.State)
	}
	if after.FillPrice != 0 || after.FillQuantity != 0 {
		t.Fatalf("an acceptance must not record a fill: price=%.2f qty=%d", after.FillPrice, after.FillQuantity)
	}
	// But it MUST have recorded the immutable row.
	ar, err := st.AcceptedRisk().ForSignal("sig-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(ar) != 1 {
		t.Fatalf("the acceptance must append exactly one accepted-risk row, got %d", len(ar))
	}
	if ar[0].LedgerStopPx != 95 {
		t.Fatalf("the ledger's stop at acceptance must be recorded for drift measurement, got %.4f", ar[0].LedgerStopPx)
	}
	if ar[0].AcceptedStopPx != nil {
		t.Fatalf("with no broker book the accepted stop must be NULL, got %v", *ar[0].AcceptedStopPx)
	}
}
