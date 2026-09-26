package trader

import (
	"testing"
	"time"

	"nofx/market"
)

// ── W1b E3 — THE ARM'S INVALIDATION LEG JUDGES ON THE PASS CLOCK ─────────────
//
// maybeManageArmedOrdersAtOpts takes its clock ONCE, at the head of the pass,
// and every leg beneath it judges on that `now` — except, before this wave,
// EntryGate leg 3 (scenario invalidation), which entryGateForArm built through
// a wall-clock resolver (time.Now handed over as a function VALUE, so the seam
// walk, which looks for time.Now() CALLS, never saw it).
//
// The two clocks differ by however long the pass has run (seconds in
// production: the boot sweep, the update drain, synchronous cancels). A 1m bar
// that closes inside that gap was CLOSED for leg 3 and FORMING for every other
// leg of the same pass. On the pass clock, leg 3 sees it on the NEXT pass (the
// event pass runs at most once a second). This is a CONSISTENCY fix: it can
// LOOSEN leg 3 by exactly those bars, never tighten it.
//
// The fixture: an alive tape that ends one instant before the pass clock, plus
// five 1m bars that close AFTER the pass clock, forming a 5m bucket whose close
// (98.40) is back inside the broken 4H body (top 99.00), plus one more bar back
// at 100.25 so the placement price is in the zone either way. Those bars close
// after the pass clock but before any wall clock this suite can run at, so the
// verdict does not depend on the hour the suite runs.
//
// REACH (critic, w1b): a leg-3 refusal count of 0 proves nothing unless the pass
// got TO leg 3. Two proofs, one each way:
//   - main: the limit is PLACED — placement is below leg 3 in the pass, so the
//     pass went through it;
//   - control: the SAME tape judged by a pass whose clock is after that 5m
//     bucket closed refuses at leg 3 exactly once — the fixture can invalidate.

// passClockTape is zoneTape(100.25, now) (80 closed bars, the last one closing
// at now−1ms) followed by the bars that close after `now`.
func passClockTape(now time.Time) []market.Kline {
	tape := zoneTape(100.25, now, 0)
	add := func(i int, cl float64) {
		o := now.Add(time.Duration(i) * time.Minute).UnixMilli()
		tape = append(tape, market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 0.5, Low: cl - 0.5, Close: cl})
	}
	for i := 0; i < 5; i++ { // the 5m bucket [now, now+5m): close 98.40, inside the body
		add(i, 98.4)
	}
	add(5, 100.25) // [now+5m, now+6m): back in the zone — the placement price
	return tape
}

func TestArmInvalidationLegJudgesThePassClock(t *testing.T) {
	const class = pictureClassSharedPrefix + "entry_gate:invalidated"

	t.Run("a bucket that closes after the pass clock is not judged by this pass", func(t *testing.T) {
		r, epoch := newPicRig(t, "w1b-e3-pass", nil)
		picPlan(r, picScenario("P1", "opp-e3-pass", r.now, epoch, picDefault))
		r.setTape(passClockTape(r.now))
		r.at.maybeManageArmedOrdersAt(nil, r.now)
		if n := r.armRefusals(class); n != 0 {
			t.Fatalf("leg 3 judged a 5m bucket that closes after the pass clock (%d %s refusal(s)) — the resolver is not on the pass clock", n, class)
		}
		sigs, _ := r.drain()
		if len(sigs) != 1 || sigs[0].LimitPrice != 100.5 {
			t.Fatalf("REACH: the pass must go through leg 3 to placement (one 100.50 limit), got %+v", sigs)
		}
		limitOnly(t, sigs)
	})

	t.Run("control: the same tape on a pass clock after the bucket closed refuses at leg 3", func(t *testing.T) {
		r, epoch := newPicRig(t, "w1b-e3-control", nil)
		picPlan(r, picScenario("P1", "opp-e3-control", r.now, epoch, picDefault))
		r.setTape(passClockTape(r.now))
		at := r.now.Add(6 * time.Minute)
		r.flatBook(at)
		r.at.maybeManageArmedOrdersAt(nil, at)
		if n := r.armRefusals(class); n != 1 {
			t.Fatalf("control: the fixture must invalidate at leg 3 once the bucket has closed on the pass clock, got %d %s refusal(s)", n, class)
		}
		if sigs, _ := r.drain(); len(sigs) != 0 {
			t.Fatalf("control: an invalidated scenario places nothing, got %+v", sigs)
		}
	})
}
