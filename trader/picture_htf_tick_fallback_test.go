package trader

import (
	"testing"

	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W4 / D24 — the wall-clock fallback exists to cover a MISSED
// boundary frame. Where no completed frame has EVER arrived there is nothing
// to be late about: the evaluation would run against zero stamps, and a feed
// that never delivers would look identical to one that is merely quiet.

func TestPictureHtf_TickFallbackWaitsForACompletedFrame(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	ev := env.at.pictureHtfEvaluator()
	if ev == nil {
		t.Fatalf("fixture: the trader must own an evaluator")
	}

	// No frame has arrived. The fallback must not evaluate, and must say so.
	env.at.pictureHtfTickFallback(env.now)
	if got := ev.TickFallbackSkips(); got != 1 {
		t.Fatalf("a fallback with no completed frame must be counted, got %d", got)
	}
	if len(env.submits) != 0 {
		t.Fatalf("nothing may send before any completed frame, got %d", len(env.submits))
	}

	// A completed frame arrives; the fallback now has something to be late
	// about, and evaluates.
	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Final = true
		frame[i].EmittedAt = env.now.UnixMilli()
	}
	ev.OnBars("MNQ", "5m", frame, env.now)
	before := ev.TickFallbackSkips()
	env.at.pictureHtfTickFallback(env.now)
	if got := ev.TickFallbackSkips(); got != before {
		t.Fatalf("with a completed frame in hand the fallback must evaluate, not skip (%d -> %d)", before, got)
	}
}
