package trader

import (
	"errors"
	"testing"
	"time"

	"nofx/store"
	"nofx/telemetry"
)

// TestArmedLifecycleFailureFailsSlotClosed is the P1-E pin (audit 2026-09-26):
// a failed armed-order lifecycle write must be loud (WARN + recorded on the
// errors API) AND fail the slot closed — the next placement for that slot is
// refused until a successful write clears it. Every one of the 16 discard
// sites in armed_executor.go rides armLifecycleWrite, so this pins them all.
func TestArmedLifecycleFailureFailsSlotClosed(t *testing.T) {
	resetArmedLedgerFailuresForTest()
	t.Cleanup(resetArmedLedgerFailuresForTest)
	at := plannerTestTrader(t)
	at.id = "p1e-arm-loud"

	r := store.ArmedOrderDB{
		TraderID: at.id, PlanID: "p1", Version: 1, Session: "ASIA",
		Scenario: "S1", Side: "long", EntryPx: 100, StopPx: 99, TargetPx: 102,
		State: store.StateWorking, SignalID: "sig-live-1", LegIndex: 0, LegCount: 1,
		Kind: "limit", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	// A FAILED write on a row that still carries a signal → slot fails closed.
	at.armLifecycleWrite("set_state(cancelled)", r, errors.New("injected SQLITE_BUSY"))
	if !armSlotBlockedForPlacement(r) {
		t.Fatalf("a failed lifecycle write on a live-signal row must block the slot's next placement")
	}
	found := false
	for _, row := range telemetry.ErrorSummary(at.id) {
		if row.Type == "armed_ledger_write_failed" && row.Count >= 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("the failure must be RECORDED on the errors API (armed_ledger_write_failed)")
	}

	// A SUCCESSFUL write clears the latch (reconcile retries clear it).
	at.armLifecycleWrite("set_state(cancelled)", r, nil)
	if armSlotBlockedForPlacement(r) {
		t.Fatalf("a successful lifecycle write must clear the slot's latch")
	}

	// A failed write on a row with NO signal (nothing at the broker) blocks
	// nothing — there is no second order to prevent.
	unplaced := r
	unplaced.SignalID = ""
	at.armLifecycleWrite("set_state(cancelled)", unplaced, errors.New("injected"))
	if armSlotBlockedForPlacement(unplaced) {
		t.Fatalf("a signal-less row must not fail a slot closed — nothing is live at the broker")
	}
}
