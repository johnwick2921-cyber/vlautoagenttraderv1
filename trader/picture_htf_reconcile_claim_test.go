package trader

import (
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// W-EXEC-TRUTH W5 R11 — the broker reconcile sweep never annotates a row that
// was never sent.
//
// Since W5 the evaluator's claim (confirmed → place_pending, synthetic claim
// id "picture-htf-<ms>", submitted_at 0) is handed to the Day Plan, never to
// the wire. While that hand-off is in flight — or after it was interrupted —
// the row sits place_pending with the claim id and no submission stamp. Such
// a row belongs to the D17 interrupted-hand-off sweep
// (PictureHtfHandOffPendingByTrader), not to the broker reconcile: stamping
// "reconcile:no_received_evidence" on it claims an order was sent, and the
// note survives the row settling "planned".
//
// Both tests drive the production sweep (pictureHtfReconcilePending — the
// function pictureHtfTickFallback calls) over rows written by the evaluator's
// own store calls (claimHandOff = PictureHtfClaim + PictureHtfClaimSubmission),
// against a real TCPTrader whose broker book is the wire's explicit empty
// snapshot, so every received-evidence step falls through to step 4.

func r11EmptyBook(t *testing.T, at *AutoTrader) {
	t.Helper()
	pictureBrokerTestReset(at)
	t.Cleanup(func() { pictureBrokerTestReset(at) })
	s := pictureWireBroker(t, at)
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
}

// An unstamped claim (mid-hand-off or interrupted) is never annotated by the
// broker reconcile — not while it is pending, and not after it settles
// "planned". It stays with the D17 sweep.
func TestPictureHtfReconcileNeverAnnotatesAnUnstampedClaim(t *testing.T) {
	at, st := handOffTrader(t)
	r11EmptyBook(t, at)
	ev := handOffEvidence(handOffNow(), 1)
	claimHandOff(t, st, ev)

	before := pictureRow(t, st, ev.OppKey)
	if !strings.HasPrefix(before.SignalID, store.PictureHtfClaimPrefix) || before.SubmittedAt != 0 || before.Stage != store.StatePlacePending {
		t.Fatalf("fixture: want an unstamped place_pending claim, got stage=%q signal=%q submitted_at=%d", before.Stage, before.SignalID, before.SubmittedAt)
	}

	pictureHtfReconcilePending(at)
	pictureHtfReconcilePending(at)

	got := pictureRow(t, st, ev.OppKey)
	if strings.Contains(got.BrokerStatus, "no_received_evidence") || got.BrokerStatus != before.BrokerStatus {
		t.Fatalf("R11: the broker reconcile annotated a claim that was never sent (signal %q, submitted_at %d): broker_status=%q",
			got.SignalID, got.SubmittedAt, got.BrokerStatus)
	}
	if got.Stage != store.StatePlacePending || got.SignalID != ev.ClaimID || got.SubmittedAt != 0 {
		t.Fatalf("R11: the reconcile must leave an unstamped claim untouched: stage=%q signal=%q submitted_at=%d", got.Stage, got.SignalID, got.SubmittedAt)
	}

	// Ownership: the D17 sweep's input still lists it.
	pend, err := st.PictureHtfHandOffPendingByTrader(handOffTraderID)
	if err != nil || len(pend) != 1 || pend[0].OppKey != ev.OppKey {
		t.Fatalf("the unstamped claim must stay with the D17 sweep: rows=%d err=%v", len(pend), err)
	}

	// The hand-off settles it "planned": the ledger carries no send claim.
	moved, err := st.PictureHtfHandOff(ev.OppKey, ev.ClaimID, "recorded as a Day Plan scenario")
	if err != nil || !moved {
		t.Fatalf("hand-off settle: moved=%v err=%v", moved, err)
	}
	pictureHtfReconcilePending(at)
	planned := pictureRow(t, st, ev.OppKey)
	if planned.Stage != store.PictureStagePlanned {
		t.Fatalf("want planned, got %q", planned.Stage)
	}
	if strings.Contains(planned.BrokerStatus, "no_received_evidence") {
		t.Fatalf("R11: a planned row carries a send-outcome note although nothing was ever sent: broker_status=%q", planned.BrokerStatus)
	}
}

// No over-skip: a row that WAS sent (submitted_at > 0) under the same claim
// prefix is still the broker reconcile's, and still gets the one-time unknown
// marker when nothing was received.
func TestPictureHtfReconcileStillAnnotatesAStampedClaimPrefixRow(t *testing.T) {
	at, st := handOffTrader(t)
	r11EmptyBook(t, at)
	ev := handOffEvidence(handOffNow(), 2)
	claimHandOff(t, st, ev)
	// The send-side stamp (the pre-W5 send path's ledger write) keeps the
	// prefixed id and sets submitted_at.
	if err := st.PictureHtfStampSignal(ev.OppKey, ev.ClaimID, ev.ClaimID); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	stamped := pictureRow(t, st, ev.OppKey)
	if !strings.HasPrefix(stamped.SignalID, store.PictureHtfClaimPrefix) || stamped.SubmittedAt <= 0 {
		t.Fatalf("fixture: want a stamped claim-prefix row, got signal=%q submitted_at=%d", stamped.SignalID, stamped.SubmittedAt)
	}
	// Not the D17 sweep's: it lists unstamped claims only.
	if pend, err := st.PictureHtfHandOffPendingByTrader(handOffTraderID); err != nil || len(pend) != 0 {
		t.Fatalf("a stamped row must not be listed for the D17 sweep: rows=%d err=%v", len(pend), err)
	}

	pictureHtfReconcilePending(at)
	pictureHtfReconcilePending(at)

	got := pictureRow(t, st, ev.OppKey)
	if got.Stage != store.StatePlacePending {
		t.Fatalf("an absent book order proves nothing — the row must stay place_pending: stage=%q", got.Stage)
	}
	if n := strings.Count(got.BrokerStatus, "reconcile:no_received_evidence"); n != 1 {
		t.Fatalf("over-skip: a SENT claim-prefix row must still get the unknown marker exactly once, got %d: broker_status=%q", n, got.BrokerStatus)
	}
}
