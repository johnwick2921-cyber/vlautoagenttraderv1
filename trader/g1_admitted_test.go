package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W-EXEC-TRUTH W0 (G1, dispatch D11) — an armed row is placed only by a
// pass whose authoring gates admitted it ─────────────────────────────────────
//
// Proven by execution before this wave (the W0 map): with the arm already
// 'armed' from an earlier pass, a daily force-flat trip made the authoring
// EntryGate log "🚦 entry-gate REFUSED arm … daily_force_flat" AND, in the SAME
// pass, "📌 armed S1 placement requested" — the placement loop re-ran no gate,
// and the refusal branches only cancel rows that already carry a signal.
func TestArmRefusedAtAuthoringIsNotPlacedThatPass(t *testing.T) {
	dir := withMaintenanceDir(t)
	at, st, sigs, now := liveArmFixture(t)

	// Pass 1: the hold keeps the freshly authored arm 'armed' (never placed).
	setHold(t, dir, "job-g1")
	at.maybeManageArmedOrdersAt(nil, now)
	if err := store.ClearMaintenanceHold(dir, "job-g1"); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-sigs:
		t.Fatalf("fixture: nothing may place under the hold (signal %s)", s.sid)
	case <-time.After(200 * time.Millisecond):
	}

	// Pass 2: the daily force-flat trips — EntryGate leg D refuses S1 at
	// authoring. The row is still 'armed'; it must NOT be placed this pass.
	kernel.SetDailyForceFlat(at.id, "g1 test trip")
	t.Cleanup(func() { kernel.ClearDailyForceFlat(at.id) })
	at.maybeManageArmedOrdersAt(nil, now.Add(time.Second))
	select {
	case s := <-sigs:
		t.Fatalf("G1: a leg the authoring gates REFUSED this pass reached the wire (signal %s)", s.sid)
	case <-time.After(300 * time.Millisecond):
	}
	rows, err := st.ArmedOrders().ListNonTerminal(at.id)
	if err != nil {
		t.Fatal(err)
	}
	armed := 0
	for _, r := range rows {
		if r.State == store.StateArmed {
			armed++
		}
	}
	if armed == 0 {
		t.Fatalf("a refusal is never a cancellation — the arm stays armed: %+v", rows)
	}

	// Pass 3: the trip clears → the same arm is admitted and places.
	kernel.ClearDailyForceFlat(at.id)
	at.maybeManageArmedOrdersAt(nil, now.Add(2*time.Second))
	select {
	case s := <-sigs:
		if s.sid == "" {
			t.Fatal("empty signal id")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("once admitted again the surviving arm must place")
	}
}
