package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

func ctMs(y int, mo time.Month, d, h, mi int) int64 {
	return time.Date(y, mo, d, h, mi, 0, 0, CTLocation()).UnixMilli()
}

// mkBar builds a 5m bar with the given open/high/low/close.
func mkBar(openMs int64, o, h, l, c float64) market.Kline {
	return market.Kline{
		OpenTime:  openMs,
		CloseTime: openMs + 5*60_000 - 1,
		Open:      o, High: h, Low: l, Close: c,
	}
}

// g7Fixture is the 2026-08-21 London v3 flip fixture: plan born 04:49 CT, flip
// "2x5m close below 29470.25" (the actual machine line that killed LONDON v3).
func g7Fixture() []market.Kline {
	return []market.Kline{
		mkBar(ctMs(2026, 8, 21, 7, 55), 29510.50, 29516.25, 29505.00, 29511.50),
		mkBar(ctMs(2026, 8, 21, 8, 0), 29511.75, 29516.25, 29466.75, 29469.25),
		mkBar(ctMs(2026, 8, 21, 8, 5), 29469.50, 29485.25, 29460.00, 29469.75),
	}
}

func g7FlipCond() PlanCondition {
	return PlanCondition{Price: 29470.25, Side: "below", Rule: "2x5m"}
}

func TestFlipEvalAgeAndAllowed(t *testing.T) {
	bars := g7Fixture()
	now := ctMs(2026, 8, 21, 8, 10)
	age, ok := FlipEvalAge(bars, "2x5m", now)
	if !ok || age < 0 || age > 5*60_000 {
		t.Fatalf("fresh series: ok=%v age=%d, want ~1ms old", ok, age)
	}
	allowed, _, why := FlipEvalAllowed(bars, "2x5m", now)
	if !allowed {
		t.Fatalf("fresh series should be allowed, got why=%s", why)
	}

	// Stale: same series, now 80 minutes later with no newer bars.
	staleNow := ctMs(2026, 8, 21, 9, 30)
	allowed, age2, why := FlipEvalAllowed(bars, "2x5m", staleNow)
	if allowed {
		t.Fatalf("stale series must not be allowed")
	}
	if why != "stale_bars" || age2 < 70*60_000 {
		t.Fatalf("want stale_bars with ~80m age, got %s age=%d", why, age2)
	}

	// No closed bars.
	emptyNow := ctMs(2026, 8, 21, 7, 50) // before the first bar closes
	allowed, _, why = FlipEvalAllowed(bars, "2x5m", emptyNow)
	if allowed || why != "no_closed_bars" {
		t.Fatalf("want no_closed_bars, got allowed=%v why=%s", allowed, why)
	}
}

func g7Doc() PlanDoc {
	c := g7FlipCond()
	return PlanDoc{
		Bias:           PlanBias{FlipCondition: "2x5m close below 29470.25 flips bias to short"},
		FlipStructured: &c,
	}
}

func TestPlanDeathOrFlipSinceFresh_StaleSkipsAndFreshFires(t *testing.T) {
	doc := g7Doc()
	since := ctMs(2026, 8, 21, 4, 49)

	// Fresh — the flip fires on the 08:00/08:05 closes (2 consecutive below).
	now := ctMs(2026, 8, 21, 8, 10)
	killer, fired, skipped := PlanDeathOrFlipSinceFresh(doc, g7Fixture(), "2x5m", since, now)
	if !fired {
		t.Fatalf("fresh bars must fire, skipped=%v", skipped)
	}
	if !strings.Contains(killer, "flip-condition") || !strings.Contains(killer, "29470.25") {
		t.Fatalf("unexpected killer: %q", killer)
	}
	if len(skipped) != 0 {
		t.Fatalf("fresh eval must not skip: %v", skipped)
	}

	// Stale — the same transition must NOT fire; it defers.
	staleNow := ctMs(2026, 8, 21, 9, 30)
	killer, fired, skipped = PlanDeathOrFlipSinceFresh(doc, g7Fixture(), "2x5m", since, staleNow)
	if fired {
		t.Fatalf("stale eval must not fire, got killer=%q", killer)
	}
	if len(skipped) == 0 || !strings.Contains(skipped[0], "flip=stale_bars") {
		t.Fatalf("want flip=stale_bars skip, got %v", skipped)
	}
}

// TestG7Replay_LondonV3FlipTiming quantifies what G7 buys on the real day: the
// actual flip/death logged at 08:20:01 CT; on a fresh feed the machine fires at
// the first evaluation after the 08:05 5m close (bar closes 08:09:59.999 → first
// fresh eval 08:10) — a delta of ~10 min.
func TestG7Replay_LondonV3FlipTiming(t *testing.T) {
	doc := g7Doc()
	since := ctMs(2026, 8, 21, 4, 49)
	// First fresh evaluation instant after the qualifying close.
	firedAt := ctMs(2026, 8, 21, 8, 10)
	if _, fired, _ := PlanDeathOrFlipSinceFresh(doc, g7Fixture(), "2x5m", since, firedAt); !fired {
		t.Fatalf("replay: flip must fire at the first fresh eval (08:10 CT)")
	}
	// The stale path at the actual log time must skip, not fire.
	if _, fired, skipped := PlanDeathOrFlipSinceFresh(doc, g7Fixture(), "2x5m", since, ctMs(2026, 8, 21, 8, 20)); fired || len(skipped) == 0 {
		t.Fatalf("replay: at 08:20 the stale series must skip (fired=%v skipped=%v)", fired, skipped)
	}
}
