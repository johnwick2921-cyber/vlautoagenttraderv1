package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// bar builds one closed 1-minute bar at minute i after base.
func bar(base time.Time, i int, o, h, l, c float64) market.Kline {
	ct := base.Add(time.Duration(i) * time.Minute)
	return market.Kline{
		OpenTime: ct.UnixMilli(), Open: o, High: h, Low: l, Close: c,
		CloseTime: ct.Add(time.Minute).UnixMilli(),
	}
}

// P1c — a wick through a level does NOT consume it: consumption needs N
// consecutive RULE-TIMEFRAME closes beyond, and a bar whose close returns
// to the near side breaks the count.
func TestP1CWickDoesNotConsume(t *testing.T) {
	// Fixed base (not time.Now()-relative) so bucket completion never depends
	// on the wall-clock second the test happens to run at.
	base := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	const level = 30000.0
	bars := []market.Kline{
		bar(base, 0, level-2, level+2, level-4, level-1),  // straddle = touch
		bar(base, 1, level-1, level+25, level-2, level+1), // WICK through, close back
		bar(base, 2, level-1, level+30, level-3, level-1), // WICK again, close back
		bar(base, 3, level-2, level+1, level-4, level+1),
		bar(base, 4, level-2, level+1, level-4, level-1),
		bar(base, 5, level-3, level+2, level-5, level+1),
		bar(base, 6, level-2, level+1, level-4, level-1),
		bar(base, 7, level-3, level, level-5, level+1),
	}
	// now sits 2 minutes past the last close so every 5m bucket is closed
	// regardless of where the sub-minute offset falls on the wall clock.
	now := base.Add(10 * time.Minute).UnixMilli()
	since := base.UnixMilli()

	if !LevelTouchedOn(bars, level, "2x5m", now) {
		t.Fatal("straddle must touch the level")
	}
	if ConsumedSince(bars, level, "2x5m", since, now) {
		t.Fatal("wicks alone must NOT consume (closes never held beyond)")
	}
	f := EvaluateLevelFacts(bars, level, DirAbove, "2x5m", 3, now)
	if !f.StillValid {
		t.Fatalf("wick series must keep the level valid, got cbUp=%d cbDn=%d", f.ClosesBeyondUp, f.ClosesBeyondDown)
	}
}

// P1c — sitting beyond a level without ever touching it does NOT consume
// (the 2026-08-17 windowless burn bug: support below price all session read
// \"accepted through\").
func TestP1CSittingBeyondDoesNotConsume(t *testing.T) {
	base := time.Now().Add(-30 * time.Minute)
	const level = 30000.0
	var bars []market.Kline
	for i := 0; i < 20; i++ {
		px := level + 20 + float64(i)
		bars = append(bars, bar(base, i, px-1, px+2, px-3, px))
	}
	now := base.Add(25 * time.Minute).UnixMilli() // 5m slack: all buckets closed
	if ConsumedSince(bars, level, "2x5m", base.UnixMilli(), now) {
		t.Fatal("untouched beyond-side bars must NOT consume the level")
	}
}

// P1c — the honest sequence: touch, then N consecutive rule-TF closes beyond.
func TestP1CTouchThenAcceptConsumes(t *testing.T) {
	base := time.Now().Add(-30 * time.Minute)
	const level = 30000.0
	bars := []market.Kline{bar(base, 0, level-2, level+2, level-4, level-1)} // touch
	for i := 1; i < 20; i++ {
		px := level + 10 + float64(i)
		bars = append(bars, bar(base, i, px-1, px+2, px-3, px))
	}
	now := base.Add(25 * time.Minute).UnixMilli() // 5m slack: all buckets closed
	if !ConsumedSince(bars, level, "2x5m", base.UnixMilli(), now) {
		t.Fatal("touch + 2×5m closes beyond must consume")
	}
}

// P1c — acceptance that happened BEFORE the window (row birth) does not count:
// the counter only sees bars since the level entered play.
func TestP1CPreWindowAcceptanceDoesNotConsume(t *testing.T) {
	base := time.Now().Add(-60 * time.Minute)
	const level = 30000.0
	var bars []market.Kline
	// Minutes 0-4: sitting near the level (below).
	for i := 0; i < 5; i++ {
		bars = append(bars, bar(base, i, level-2, level+1, level-4, level-2))
	}
	// Minutes 5-9: closes beyond (acceptance begins).
	for i := 5; i < 10; i++ {
		px := level + 5 + float64(i)
		bars = append(bars, bar(base, i, px-1, px+2, px-3, px))
	}
	// Minute 10: the touch (straddle).
	bars = append(bars, bar(base, 10, level-2, level+2, level-4, level-1))
	// Minutes 11-19: closes beyond (acceptance holds through the tail).
	for i := 11; i < 20; i++ {
		px := level + 10 + float64(i)
		bars = append(bars, bar(base, i, px-1, px+2, px-3, px))
	}
	now := base.Add(25 * time.Minute).UnixMilli() // 5m slack: all buckets closed
	// Window from minute 15: the touch (minute 10) is OUTSIDE the window.
	since := base.Add(15 * time.Minute).UnixMilli()
	if ConsumedSince(bars, level, "2x5m", since, now) {
		t.Fatal("acceptance without an in-window touch must not consume")
	}
	// And with the full window it WOULD consume (sanity on the fixture).
	if !ConsumedSince(bars, level, "2x5m", base.UnixMilli(), now) {
		t.Fatal("fixture sanity: full-window evaluation must consume")
	}
}

// P1c — 2×1m closes must NOT satisfy \"2×5m\" (the H10 interval invariant,
// now also enforced on the level-state consumption path).
func TestP1CIntervalNotConfused(t *testing.T) {
	base := time.Now().Add(-30 * time.Minute)
	const level = 30000.0
	bars := []market.Kline{bar(base, 0, level-2, level+2, level-4, level-1)} // touch
	// exactly TWO 1-minute closes beyond, then back below — a 2×5m rule needs
	// two 5-MINUTE closes, which never form here.
	bars = append(bars, bar(base, 1, level+1, level+4, level-2, level+3))
	bars = append(bars, bar(base, 2, level+2, level+5, level-1, level+4))
	bars = append(bars, bar(base, 3, level-3, level+1, level-5, level-2))
	bars = append(bars, bar(base, 4, level-3, level, level-6, level-2))
	now := base.Add(5 * time.Minute).UnixMilli()
	if ConsumedSince(bars, level, "2x5m", base.UnixMilli(), now) {
		t.Fatal("two 1m closes must not satisfy a 2×5m acceptance")
	}
}
