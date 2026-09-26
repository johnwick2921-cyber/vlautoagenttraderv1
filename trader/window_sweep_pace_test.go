package trader

import (
	"fmt"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W1b FOLD-12 — the window sweep paces its cancel to a cancel_pending row ──
//
// When a window class refuses (the E13 force-flat window, the no-trade band),
// maybeManageArmedOrdersAtOpts cancels the plan's resting arms through
// cancelArmedOrdersSync — on the scan AND on every live-bar pass. It re-sent
// the cancel to EVERY non-terminal row each time, cancel_pending ones included
// (cancelArmedOrdersSyncWith filtered only SignalID == ""): on a dark book
// every re-send blocked the pass under armedPassMu for up to 2× the ack
// timeout per row and bumped cancel_attempts on no ack, burning the re-request
// cap the settlement pass paces by. The rule (CTO FOLD-12): the sweep skips a
// cancel_pending row whose cancel was requested < 30 s ago, on the pass's
// clock; past it the cancel is re-sent ONCE (a lost frame is still retried),
// and the pace runs again from that re-send. Driven at maybeManageArmedOrdersAt
// over the real TCP wire (canon 53); the AddOn never acks — a dark book.

// windowPass runs one armed pass at `at` inside the T1 force-flat lead and
// returns the cancel frames it put on the wire for sid (any other cancel fails).
func (r *zoneRig) windowPass(at time.Time, sid string) int {
	r.t.Helper()
	r.setTape(zoneTape(101.95, at, 0))
	r.at.maybeManageArmedOrdersAt(nil, at)
	sigs, cancels := r.drain()
	if len(sigs) != 0 {
		r.t.Fatalf("no entry may reach the wire inside the force-flat lead: %+v", sigs)
	}
	for _, c := range cancels {
		if c.SignalID != sid {
			r.t.Fatalf("a cancel for a signal the sweep does not own: %+v", c)
		}
	}
	return len(cancels)
}

// cancelBrief is the row's cancel lifecycle, for a failure message.
func cancelBrief(r store.ArmedOrderDB) string {
	return fmt.Sprintf("#%d %s attempts=%d requested_at_ms=%d reason=%q", r.ID, r.State, r.CancelAttempts, r.CancelRequestedAtMs, r.StateReason)
}

func windowSweepRig(t *testing.T, id string) (*zoneRig, string) {
	t.Helper()
	t.Setenv("ARMED_CANCEL_ACK_TIMEOUT_MS", "40") // a dark-book send costs 2×40 ms here, not 2×2 s
	r := newZoneRig(t, id, zoneDoc(zoneScenario("S1", kernel.EntryPolicyMarketInZone, zone, false)))
	t1LeadSlice(t, r.st)
	r.now = e13Before // 09:57:30 CT, before the lead: the arm is placed and working
	return r, r.placeWorking(100.5).SignalID
}

func TestWindowSweepPacesItsCancelToACancelPendingRow(t *testing.T) {
	r, sid := windowSweepRig(t, "w1b-fold12-pace")

	// Pass 1 — 09:58:30 CT, the lead is open: the sweep cancels the working
	// arm; no ack comes, so the row is held cancel_pending (one request).
	sweep := r.windowPass(e13Lead, sid)
	if sweep == 0 {
		t.Fatal("fixture: the window sweep must send the working arm's cancel")
	}
	if got := r.row("S1"); got.State != store.StateCancelPending || got.CancelAttempts != 1 {
		t.Fatalf("fixture: an unacked sweep cancel holds the row cancel_pending after one request: %s", cancelBrief(got))
	}

	// Passes 2 and 3 — within 10 s of the request: nothing on the wire,
	// nothing counted. Exactly ONE cancel per row.
	for i, dt := range []time.Duration{4 * time.Second, 9 * time.Second} {
		if n := r.windowPass(e13Lead.Add(dt), sid); n != 0 {
			t.Fatalf("pass %d (+%s): the sweep re-sent %d cancel frame(s) to a cancel_pending row requested %s ago — exactly ONE cancel per row inside the 30 s pace", i+2, dt, n, dt)
		}
	}
	if got := r.row("S1"); got.State != store.StateCancelPending || got.CancelAttempts != 1 {
		t.Fatalf("three passes in 10 s must leave ONE cancel request on the row (cancel_attempts=%d): %s", got.CancelAttempts, cancelBrief(got))
	}

	// Pass 4 — 31 s after the request: re-sent once, so a lost frame is
	// still retried.
	if n := r.windowPass(e13Lead.Add(31*time.Second), sid); n != sweep {
		t.Fatalf("pass 4 (+31s): past the pace the sweep must re-send the cancel once (%d frame(s), as pass 1), got %d", sweep, n)
	}
	if got := r.row("S1"); got.State != store.StateCancelPending || got.CancelAttempts != 2 {
		t.Fatalf("the paced re-send is one more request (cancel_attempts=2): %s", cancelBrief(got))
	}

	// Pass 5 — 4 s after that re-send: the pace runs from the LATEST request.
	if n := r.windowPass(e13Lead.Add(35*time.Second), sid); n != 0 {
		t.Fatalf("pass 5 (+35s): the pace must run from the latest request (+31s), not the first — re-sent %d frame(s)", n)
	}
	if got := r.row("S1"); got.CancelAttempts != 2 {
		t.Fatalf("pass 5 counted a request it did not make (cancel_attempts=%d)", got.CancelAttempts)
	}
}

// The pace is the WINDOW SWEEP'S alone: the EOD / session-close / news / T1
// enforce callers of cancelArmedOrdersSync keep today's behaviour — every
// non-terminal row with a signal gets its cancel on every call, a row
// cancel_pending for a second included.
func TestWindowSweepPaceLeavesTheUnpacedCallersUnchanged(t *testing.T) {
	r, sid := windowSweepRig(t, "w1b-fold12-legacy")
	sweep := r.windowPass(e13Lead, sid)
	if sweep == 0 {
		t.Fatal("fixture: the window sweep must send the working arm's cancel")
	}
	if n, unacked := r.at.cancelArmedOrdersSync("session close — EOD flat"); n != 0 || unacked != 1 {
		t.Fatalf("the EOD flat's cancel of a cancel_pending row is unpaced: n=%d unacked=%d", n, unacked)
	}
	if _, cancels := r.drain(); len(cancels) != sweep {
		t.Fatalf("the EOD flat must re-send the cancel at once (%d frame(s)), got %d", sweep, len(cancels))
	}
	if got := r.row("S1"); got.CancelAttempts != 2 {
		t.Fatalf("the EOD flat's request is counted (cancel_attempts=2): %s", cancelBrief(got))
	}
}

// No broker link (at.armedTrader() nil): the sweep sends nothing, it records
// the INTENT (cancel_pending) — and that record is paced the same way, so a
// dark link does not bump cancel_attempts on every pass either.
func TestWindowSweepPacesTheNoLinkIntentRecord(t *testing.T) {
	r, _ := windowSweepRig(t, "w1b-fold12-nolink")
	r.at.trader = nil // the NT8 bridge is gone
	for _, dt := range []time.Duration{0, 4 * time.Second, 9 * time.Second} {
		r.setTape(zoneTape(101.95, e13Lead.Add(dt), 0))
		r.at.maybeManageArmedOrdersAt(nil, e13Lead.Add(dt))
	}
	if got := r.row("S1"); got.State != store.StateCancelPending || got.CancelAttempts != 1 {
		t.Fatalf("three no-link passes in 10 s must record ONE cancel intent (cancel_attempts=%d): %s", got.CancelAttempts, cancelBrief(got))
	}
	r.setTape(zoneTape(101.95, e13Lead.Add(31*time.Second), 0))
	r.at.maybeManageArmedOrdersAt(nil, e13Lead.Add(31*time.Second))
	if got := r.row("S1"); got.State != store.StateCancelPending || got.CancelAttempts != 2 {
		t.Fatalf("past the pace the intent is recorded once more (cancel_attempts=2): %s", cancelBrief(got))
	}
}

// Only a cancel_pending row is paced: a row in any other non-terminal state is
// sent its cancel even when the sweep's own record says it sent one 4 s ago.
func TestWindowSweepPacesOnlyACancelPendingRow(t *testing.T) {
	// One subtest per non-terminal state, called by name — no retyped state
	// list (store TestArmStateNoRetypedLists).
	check := func(t *testing.T, st string) {
		r, sid := windowSweepRig(t, "w1b-fold12-only-pending-"+st)
		sweep := r.windowPass(e13Lead, sid)
		if sweep == 0 {
			t.Fatal("fixture: the window sweep must send the working arm's cancel")
		}
		got := r.row("S1")
		if got.State != store.StateCancelPending {
			t.Fatalf("fixture: %s", cancelBrief(got))
		}
		if err := r.st.ArmedOrders().DB().Model(&store.ArmedOrderDB{}).Where("id = ?", got.ID).UpdateColumn("state", st).Error; err != nil {
			t.Fatal(err)
		}
		if n := r.windowPass(e13Lead.Add(4*time.Second), sid); n != sweep {
			t.Fatalf("a %s row (the sweep's record 4 s old) was paced: sent %d frame(s), want %d", st, n, sweep)
		}
	}
	t.Run(store.StateWorking, func(t *testing.T) { check(t, store.StateWorking) })
	t.Run(store.StatePlacePending, func(t *testing.T) { check(t, store.StatePlacePending) })
}

// "Requested" is any path's request the ledger records, not only the sweep's
// own: a cancel another path requested 10 s ago paces the sweep; one requested
// 60 s ago does not.
func TestWindowSweepPaceCountsAnotherPathsRequest(t *testing.T) {
	for _, c := range []struct {
		age  time.Duration
		send bool
	}{{10 * time.Second, false}, {60 * time.Second, true}} {
		t.Run(c.age.String(), func(t *testing.T) {
			r, sid := windowSweepRig(t, "w1b-fold12-other-path-"+c.age.String())
			got := r.row("S1")
			if err := r.st.ArmedOrders().RequestCancel(got.ID, "another path", e13Lead.Add(-c.age).UnixMilli()); err != nil {
				t.Fatal(err)
			}
			if n := r.windowPass(e13Lead, sid); (n > 0) != c.send {
				t.Fatalf("a cancel another path requested %s ago: the sweep sent %d frame(s), want send=%v", c.age, n, c.send)
			}
		})
	}
}

// Over a long dark window (the T1 lead running into the blackout band) a lost
// cancel is re-sent every 30 s — never starved, never twice inside 30 s — with
// passes every 5 s.
func TestWindowSweepReSendsEveryThirtySecondsAcrossALongWindow(t *testing.T) {
	r, sid := windowSweepRig(t, "w1b-fold12-cadence")
	var sends []time.Duration
	for dt := time.Duration(0); dt <= 150*time.Second; dt += 5 * time.Second {
		if n := r.windowPass(e13Lead.Add(dt), sid); n > 0 {
			sends = append(sends, dt)
		}
	}
	want := []time.Duration{0, 30 * time.Second, 60 * time.Second, 90 * time.Second, 120 * time.Second, 150 * time.Second}
	if fmt.Sprint(sends) != fmt.Sprint(want) {
		t.Fatalf("send cadence %v, want %v (%s)", sends, want, cancelBrief(r.row("S1")))
	}
}
