package trader

import (
	"strings"
	"testing"

	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W4 / D21 — the evaluator must KNOW it has enough history.
//
// Before this wave the only depth guard was "len(bars) == 0", so the two
// pictures were drawn from whatever history happened to be in the cache. Worse,
// the fetch asked for exactly PivotWindow+4 bars and THEN filtered to completed
// ones, so the count could never reach the requirement even in principle: the
// tail always holds the bar that is still forming. The fetch must exceed the
// requirement, and the refusal must report the count it actually read.

func TestPictureHtf_InsufficientDepthRefusesWithTheTrueCount(t *testing.T) {
	// PivotWindow 8 → the rule needs 8+4 = 12 completed 4H candles.
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, PivotWindow: 8})
	env.seedPictureTape()
	// Serve only the last 9 candles of the ladder. The newest of them has not
	// closed at env.now, so exactly 8 are COMPLETED — and the refusal must say
	// 8, the number it read, not 9, the number it was handed.
	env.seed(tailOf(pictureBars4H(), 9), pictureBarsH1(), pictureBars5M(true))

	res := env.eval.Evaluate("MNQ", env.now)

	if !strings.Contains(res.Reason, "insufficient depth") {
		t.Fatalf("short 4H history must refuse for depth, got stage=%q reason=%q", res.Stage, res.Reason)
	}
	if !strings.Contains(res.Reason, "8/12") {
		t.Fatalf("the refusal must report the TRUE completed count 8/12, got %q", res.Reason)
	}
	if len(env.submits) != 0 {
		t.Fatalf("a depth refusal must never submit")
	}
}

// The mirror: with the requirement met the evaluator proceeds, so the depth
// guard cannot be satisfied by simply refusing everything.
func TestPictureHtf_SufficientDepthProceeds(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, PivotWindow: 8})
	env.seedPictureTape() // 13 completed candles >= 12
	res := env.eval.Evaluate("MNQ", env.now)
	if strings.Contains(res.Reason, "insufficient depth") {
		t.Fatalf("13 completed candles satisfy a requirement of 12, got %q", res.Reason)
	}
}

// The fetch must ask for MORE than the requirement, or the forming bar
// guarantees a short count. This pins the margin at the provider boundary:
// the evaluator is observed asking for strictly more than PivotWindow+4.
func TestPictureHtf_FetchExceedsTheDepthRequirement(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5, PivotWindow: 8})
	env.seedPictureTape()
	asked := recordFourHFetch(env)
	env.eval.Evaluate("MNQ", env.now)
	if *asked <= 8+4 {
		t.Fatalf("the 4H fetch asked for %d, which cannot yield %d completed bars once the forming bar is filtered out", *asked, 8+4)
	}
}

// recordFourHFetch wraps the installed provider and reports the largest 4H
// count the evaluator asked for.
func recordFourHFetch(env *pictureHtfTestEnv) *int {
	inner := market.FuturesBarsProvider
	largest := 0
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		if tf == "4h" && count > largest {
			largest = count
		}
		return inner(symbol, tf, count)
	}
	return &largest
}

// pictureTestResolved resolves a Picture config the way production does, but
// defaults PivotWindow to a window these ladders can satisfy.
//
// W4/D21 gave the evaluator a real depth rule: it refuses until it holds
// PivotWindow+4 COMPLETED 4H candles, and production defaults PivotWindow to
// 120, i.e. 124 candles. The Picture fixtures are a dozen candles long because
// they are about pivot GEOMETRY, lifecycle and admission — not depth. Left on
// the production default every one of them would refuse for depth and stop
// testing what it was written to test. Depth itself is driven from both sides
// of its boundary by the tests above, which set the knob explicitly.
func pictureTestResolved(c *store.PictureHtfConfig) store.PictureHtfConfig {
	if c != nil && c.PivotWindow <= 0 {
		c.PivotWindow = 16
	}
	return store.PictureHtfResolved(c)
}
