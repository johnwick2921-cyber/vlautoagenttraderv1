package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W4 / D23 — the entry window belongs to the H1 CLOSE that
// confirmed the break, not to whichever 5m candle happens to be newest.
//
// The window used to be measured from `newest5m.OpenTime + 5m`. Every new 5m
// candle moved that anchor forward, so the window RE-OPENED every five
// minutes, indefinitely, for a break that had been confirmed long before. The
// owner's method is "a 4H body pivot → an H1 close breaks it → the NEXT 5m
// interval sends" — one interval, once.

// fiveMPlus extends the 5m ladder by n further candles, so the newest candle
// is well past the H1 close that confirmed the break.
func fiveMPlus(n int) []market.Kline {
	base := pictureBars5M(true)
	last := base[len(base)-1]
	c := last.Close
	for i := 1; i <= n; i++ {
		c += 0.02
		base = append(base, mkBar(last.OpenTime+int64(i)*5*60*1000, 5*60*1000, c, c+0.3, c-0.3, c+0.02))
	}
	return base
}

func TestPictureHtf_ALater5mCandleCannotReopenTheEntryWindow(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	// Six further 5m candles: the newest one now opens 30 minutes after the
	// confirming H1 close, and we evaluate half a second into ITS interval.
	env.seed(pictureBars4H(), pictureBarsH1(), fiveMPlus(6))
	late := env.now.Add(30 * time.Minute)
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 40), 1), late)

	res := env.eval.Evaluate("MNQ", late)
	// The window belongs to the confirming H1 close and shut 30 minutes ago.
	// What matters is that a newer candle cannot re-open it; the terminal the
	// evaluator reports is "never actionable" rather than "expired" because
	// this completed candle only reached us AFTER the window had closed, so
	// there was never an opportunity here to refuse (D23/R2).
	if res.Stage == "confirmed" || res.Stage == "submitted" {
		t.Fatalf("a newer 5m candle must not re-open the window — got stage=%q reason=%q", res.Stage, res.Reason)
	}
	if !strings.Contains(res.Reason, "entry window") {
		t.Fatalf("the verdict must say the entry window is not open, got %q", res.Reason)
	}
	if len(env.submits) != 0 {
		t.Fatalf("a re-opened window must never submit, got %d", len(env.submits))
	}
}

// The mirror: inside the interval that follows the confirming H1 close, the
// entry still goes. The anchor must not simply shut the window forever.
func TestPictureHtf_TheIntervalAfterTheConfirmingH1CloseStillSends(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now)
	if len(env.submits) != 1 {
		t.Fatalf("the interval following the confirming H1 close must still send, got %d submit(s)", len(env.submits))
	}
}

// D23/R2 — THE REAL ARRIVAL ORDER. The earlier version of this pin tested an
// ordering that cannot happen: it evaluated BEFORE the boundary, where the
// elapsed<0 guard already waits. But the H1 closed frame is itself emitted
// AFTER the bar closes, so elapsed is always >= 0 by the time it arrives, and
// it typically lands ~200ms BEFORE the 5m closed frame. In that gap the
// freshness stamps still describe the PREVIOUS 5m candle — every clock reads
// ~5 minutes stale — and a refusal written there is a durable `expired` that
// the qualifying frame moments later inherits.
func TestPictureHtf_H1ClosedFrameBeforeThe5mClosedFrameNeverWritesDurableExpired(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, FreshnessSec: 2, EntryWindowSec: 10})
	env.seedPictureTape()
	boundary := t4h0 + 42*3600*1000 // the interval following the confirming H1 close

	// FIRST: the PREVIOUS interval's completed candle, received on time five
	// minutes ago. This is what makes the gap dangerous — the stamps hold real
	// data that is now ~5 minutes old, so every clock reads STALE rather than
	// "nothing yet". Without it the evaluator short-circuits on "no completed
	// 5m frame at all" and never reaches the code under test.
	all5m := market.FuturesBarsProvider("MNQ", "5m", 28)
	prev := []market.Kline{all5m[len(all5m)-2]}
	prev[0].Final = true
	prev[0].EmittedAt = prev[0].CloseTime + 1200
	env.eval.OnBars("MNQ", "5m", prev, time.UnixMilli(prev[0].CloseTime+1200))

	// +1.0s — the H1 closed frame lands first; the boundary 5m candle is not
	// in hand, so the evaluator must WAIT and key nothing.
	res := env.eval.Evaluate("MNQ", time.UnixMilli(boundary+1000))
	if res.Stage == "expired" || res.Stage == "refused" {
		t.Fatalf("the H1 frame arrived before the 5m close; that must WAIT, not refuse: %+v", res)
	}
	if res.OppKey != "" {
		t.Fatalf("nothing may be keyed before the boundary candle is in hand, got %q", res.OppKey)
	}

	// +1.1s — a forming tick. Not completed data; still waiting.
	tick := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range tick {
		tick[i].Final = false
	}
	env.eval.OnBars("MNQ", "5m", tick, time.UnixMilli(boundary+1100))
	if len(env.submits) != 0 {
		t.Fatalf("a forming tick must not send, got %d", len(env.submits))
	}

	// +1.2s — the completed 5m close arrives and the entry proceeds. If the
	// gap had written a durable refusal, this frame would find it dead.
	closed := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range closed {
		closed[i].Final = true
		closed[i].EmittedAt = boundary + 1200
	}
	env.eval.OnBars("MNQ", "5m", closed, time.UnixMilli(boundary+1200))
	if len(env.submits) != 1 {
		t.Fatalf("the qualifying 5m frame 200ms later must still send, got %d", len(env.submits))
	}
}

// D23/R1 — NT8 emits a closed bar on the FIRST TICK OF THE NEXT BAR, which in
// slow tape is seconds after the boundary. That ordinary lateness must not be
// mistaken for stale data: the candle closed on time, it simply reached us a
// few seconds later, and the entry window is 10s for exactly this reason.
func TestPictureHtf_LateEmissionInsideTheWindowStillSends(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, FreshnessSec: 2, EntryWindowSec: 10})
	env.seedPictureTape()
	boundary := t4h0 + 42*3600*1000
	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Final = true
		frame[i].EmittedAt = boundary + 3000 // emitted 3s after the boundary
	}
	env.eval.OnBars("MNQ", "5m", frame, time.UnixMilli(boundary+3000))
	if len(env.submits) != 1 {
		t.Fatalf("a candle emitted 3s after its boundary is 3s late, not stale — it must send inside the 10s window, got %d", len(env.submits))
	}
}

// ...but past the window it was never actionable, and that is NOT a refusal:
// no durable `expired` may be written for an opportunity we never held in time.
func TestPictureHtf_EmissionAfterTheWindowIsNeverActionable(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, FreshnessSec: 2, EntryWindowSec: 10})
	env.seedPictureTape()
	boundary := t4h0 + 42*3600*1000
	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Final = true
		frame[i].EmittedAt = boundary + 11000 // 11s — past the 10s window
	}
	env.eval.OnBars("MNQ", "5m", frame, time.UnixMilli(boundary+11000))
	res := env.eval.Evaluate("MNQ", time.UnixMilli(boundary+11000))
	if res.Stage == "expired" || res.Stage == "refused" {
		t.Fatalf("a candle that only reached us after the window was never actionable; no refusal may be recorded: %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("past the window nothing may send, got %d", len(env.submits))
	}
	if res.OppKey != "" {
		if row, ok, _ := env.st.PictureHtfGet(res.OppKey); ok && row.Stage == "expired" {
			t.Fatalf("no durable expired row may exist: %+v", row)
		}
	}
}

// D23, the source clock. This frame ARRIVES at the evaluation instant, so the
// receipt clock reads ~0ms and the old freshness rule — which measured only
// `now - receivedAt` — passed it. The data itself was emitted a minute ago.
// Only a rule that reads the SOURCE stamp can refuse this.
func TestPictureHtf_StaleSourceStampExpiresEvenWhenTheFrameArrivesNow(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, FreshnessSec: 2})
	env.seedPictureTape()
	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Final = true
		frame[i].EmittedAt = env.now.UnixMilli() - 60_000 // emitted a minute ago
	}
	env.eval.OnBars("MNQ", "5m", frame, env.now) // ...delivered this instant

	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "expired" {
		t.Fatalf("data emitted a minute ago must expire on the SOURCE clock even though the frame arrived now: %+v", res)
	}
	if !strings.Contains(res.Reason, "source age") {
		t.Fatalf("the refusal must NAME the clock it was measured on, got %q", res.Reason)
	}
	if len(env.submits) != 0 {
		t.Fatalf("stale source data must never submit, got %d", len(env.submits))
	}
}

// The mirror: a fresh source stamp does not expire, so the source clock cannot
// be satisfied by refusing everything.
func TestPictureHtf_FreshSourceStampStillSends(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, FreshnessSec: 2})
	env.seedPictureTape()
	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Final = true
		frame[i].EmittedAt = env.now.UnixMilli() - 100 // 100ms old at source
	}
	env.eval.OnBars("MNQ", "5m", frame, env.now)
	if len(env.submits) != 1 {
		t.Fatalf("fresh source data must still send, got %d submit(s)", len(env.submits))
	}
}

// markFresh5mReceivedAt stamps the freshness evidence that a COMPLETED
// boundary 5m frame delivered at `at` would have produced: the candle that
// closed at the start of `at`'s own 5m interval, emitted and received then.
//
// Tests that drive Evaluate directly (rather than through OnBars) used to set
// freshest5mAt alone. Since W4/D23 the evaluator also needs to know the
// BOUNDARY candle is in hand — otherwise it correctly reports that it is still
// waiting for it — so the shortcut has to stamp all three or it no longer
// describes a real frame. Does NOT take e.mu: some callers already hold it.
func (e *PictureHtfEvaluator) markFresh5mReceivedAt(at time.Time) {
	ms := at.UnixMilli()
	e.freshest5mAt = at
	e.freshest5mClose = (ms/fiveMMs)*fiveMMs - 1
	e.freshest5mEmitted = ms
}
