package trader

import (
	"testing"

	"nofx/store"
)

// B2 (the F8 gap the #216 PR body flagged): the boot sweep must REQUEST the
// cancel — the row becomes cancel_pending with the boot_sweep reason — and the
// settlement pass confirms it through ConfirmCancel with a persisted snapshot
// id once a fresh post-request book shows the order absent. The sweep must
// NEVER write 'cancelled' at send time: that is the exact 'cancel without
// broker proof' defect F8 closed everywhere else.
//
// RED on today's code: the sweep writes SetState('cancelled') immediately, so
// ListCancelPending returns 0 and ConfirmCancel refuses a non-pending row.
func TestClass33BootSweepLeavesCancelPendingUntilBookConfirms(t *testing.T) {
	at := class33Trader(t)
	class33Seed(t, at, "S1", "sig-S1", "999-1", "working")

	n := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error { return nil })
	if n != 1 {
		t.Fatalf("swept %d, want 1", n)
	}
	// The row must be cancel_pending with the sweep's reason and a request
	// stamp — never 'cancelled' at send time.
	pending, err := at.store.ArmedOrders().ListCancelPending(at.id)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("after the sweep the row must be cancel_pending; got %d pending (today's code wrote 'cancelled')", len(pending))
	}
	row := pending[0]
	if row.StateReason != BootSweepReason || row.CancelRequestedAtMs <= 0 {
		t.Fatalf("pending row must carry the boot_sweep reason and a request stamp: %+v", row)
	}
	// The send is what the recorded counter counts (B2 says so).
	if got, _ := store.BootSweptCount(at.store); got != 1 {
		t.Fatalf("recorded counter = %d, want 1 (counts the SEND)", got)
	}
	// Settlement: ConfirmCancel requires the persisted snapshot whose book no
	// longer listed the order. A caller without one is refused.
	if err := at.store.ArmedOrders().ConfirmCancel(row.ID, 0, BootSweepReason); err == nil {
		t.Fatal("ConfirmCancel without a persisted snapshot id must refuse")
	}
	if err := at.store.ArmedOrders().ConfirmCancel(row.ID, 4242, BootSweepReason); err != nil {
		t.Fatalf("settlement with a snapshot id must confirm: %v", err)
	}
	var final store.ArmedOrderDB
	at.store.ArmedOrders().DB().Where("id = ?", row.ID).First(&final)
	if final.State != store.StateCancelled || final.CancelSettledSnapshotID != 4242 {
		t.Fatalf("settled row: state=%q snap=%d", final.State, final.CancelSettledSnapshotID)
	}
}
