package trader

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
	"nofx/market"
	"nofx/store"
)

// Exercise the production loop -> TCPTrader.PlaceLimitEntry -> BeginPlacement
// -> TCPServer.SendSignal. A tiny existing server age seam rejects the actual
// payload AFTER registration; this is not a fake that merely returns the
// expected registered flag. It proves post-registration errors commit the pass,
// without claiming to reproduce a partially written socket frame.
// Port of #117 b63747ea, adapted to the armAdmission signature.
func TestLimitRegistrationCommitsPassDespiteSendError(t *testing.T) {
	for _, samePlan := range []bool{true, false} {
		name := "other-plan stays unplaced"
		if samePlan {
			name = "same-plan sibling retires"
		}
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, 9, 14, 15, 0, 0, 0, time.UTC)
			at, st, signals, _ := shadowWireHarnessAt(t, store.StrategyConfig{}, now)
			server := at.armedTrader().GetServer()
			server.SetStaleSignalAgeForTest(time.Nanosecond)
			ledger := st.ArmedOrders()
			first := store.ArmedOrderDB{TraderID: at.id, PlanID: "limit-first", Version: 1, Scenario: "S1", State: store.StateArmed, Side: "long", Kind: "limit", EntryPx: 29600, StopPx: 29590, TargetPx: 29630}
			other := store.ArmedOrderDB{TraderID: at.id, PlanID: first.PlanID, Version: 1, Scenario: "S2", State: store.StateArmed, Side: "short", Kind: "limit", EntryPx: 29600, StopPx: 29610, TargetPx: 29570}
			if !samePlan {
				other.PlanID = "limit-other"
			}
			for _, row := range []*store.ArmedOrderDB{&first, &other} {
				if err := ledger.UpsertArm(row); err != nil {
					t.Fatal(err)
				}
			}
			registrations := 0
			const callback = "test:limit-registration-age"
			if err := ledger.DB().Callback().Update().After("gorm:update").Register(callback, func(tx *gorm.DB) {
				values, ok := tx.Statement.Dest.(map[string]interface{})
				if ok && tx.Statement.Table == "armed_orders" && values["state"] == store.StatePlacePending && tx.Error == nil && tx.RowsAffected == 1 {
					registrations++
					// Payload timestamp precedes this callback. Make its age
					// deterministically exceed the tiny cutoff before SendSignal.
					time.Sleep(2 * time.Millisecond)
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { ledger.DB().Callback().Update().Remove(callback) })
			admitted := armAdmission{
				armAdmitKey(first.PlanID, "S1", 0): true,
				armAdmitKey(other.PlanID, "S2", 0): true,
			}
			at.runArmedPlacementAt([]market.Kline{{Close: 29600}}, now.Add(-time.Hour).UnixMilli(), now, admitted)
			for _, row := range []*store.ArmedOrderDB{&first, &other} {
				if err := ledger.DB().First(row, row.ID).Error; err != nil {
					t.Fatal(err)
				}
			}
			if registrations != 1 || first.State != store.StatePlacePending || first.SignalID == "" {
				t.Fatalf("registration did not exclusively commit first attempt: count=%d first=%+v other=%+v", registrations, first, other)
			}
			want := store.StateArmed
			if samePlan {
				want = store.StateCancelled
			}
			if other.State != want || other.SignalID != "" {
				t.Fatalf("post-registration send error lost pass/sibling commitment: want %s without signal, got %+v", want, other)
			}
			select {
			case signal := <-signals:
				t.Fatalf("fixture should reject actual signal before socket write: %+v", signal)
			default:
			}
		})
	}
}

func TestLimitRegistrationRefusalLeavesUnplacedSiblingArmed(t *testing.T) {
	now := time.Date(2026, 9, 14, 15, 0, 0, 0, time.UTC)
	at, st, signals, _ := shadowWireHarnessAt(t, store.StrategyConfig{}, now)
	ledger := st.ArmedOrders()
	first := store.ArmedOrderDB{TraderID: at.id, PlanID: "limit-refusal", Version: 1, Scenario: "S1", State: store.StateArmed, Side: "long", Kind: "limit", EntryPx: 29600, StopPx: 29590, TargetPx: 29630}
	other := first
	other.Scenario, other.EntryPx, other.StopPx, other.TargetPx = "S2", 29000, 28990, 29030
	for _, row := range []*store.ArmedOrderDB{&first, &other} {
		if err := ledger.UpsertArm(row); err != nil {
			t.Fatal(err)
		}
	}
	refusals := 0
	const callback = "test:limit-registration-refuse"
	if err := ledger.DB().Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		values, ok := tx.Statement.Dest.(map[string]interface{})
		if ok && tx.Statement.Table == "armed_orders" && values["state"] == store.StatePlacePending {
			refusals++
			tx.AddError(errors.New("fixture registration unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ledger.DB().Callback().Update().Remove(callback) })
	admitted := armAdmission{
		armAdmitKey(first.PlanID, "S1", 0): true,
		armAdmitKey(other.PlanID, "S2", 0): true,
	}
	at.runArmedPlacementAt([]market.Kline{{Close: 29600}}, now.Add(-time.Hour).UnixMilli(), now, admitted)
	if refusals != 1 {
		t.Fatalf("test never reached the registration refusal: %d", refusals)
	}
	for _, row := range []*store.ArmedOrderDB{&first, &other} {
		if err := ledger.DB().First(row, row.ID).Error; err != nil {
			t.Fatal(err)
		}
		if row.State != store.StateArmed || row.SignalID != "" {
			t.Fatalf("refusal before registration changed unplaced row: %+v", row)
		}
	}
	select {
	case signal := <-signals:
		t.Fatalf("registration refusal sent a signal: %+v", signal)
	default:
	}
}
