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
	if res.Stage != "expired" {
		t.Fatalf("the window belongs to the confirming H1 close and passed 30 minutes ago; a newer 5m candle must not re-open it — got stage=%q reason=%q", res.Stage, res.Reason)
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

// H1-before-5m arrival order must never write a DURABLE expired row: the 5m
// boundary frame may be milliseconds behind the H1 frame, and a refusal
// recorded in that gap is one a qualifying frame would have to live with.
func TestPictureHtf_H1BeforeThe5mBoundaryNeverWritesDurableExpired(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	// Evaluate BEFORE the 5m interval opens — the H1 frame has landed first.
	early := time.UnixMilli(t4h0 + 42*3600*1000 - 200)
	res := env.eval.Evaluate("MNQ", early)
	if res.Stage == "expired" {
		t.Fatalf("an H1 frame arriving before the 5m boundary must WAIT, not expire: %+v", res)
	}
	if res.OppKey != "" {
		if row, ok, _ := env.st.PictureHtfGet(res.OppKey); ok && row.Stage == "expired" {
			t.Fatalf("no durable expired row may exist for a setup whose interval has not opened: %+v", row)
		}
	}
	if !strings.Contains(res.Stage, "watching") {
		t.Fatalf("expected to be watching before the interval opens, got %q", res.Stage)
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
