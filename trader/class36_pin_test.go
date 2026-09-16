package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// CLASS 36 (2026-09-01) — PIN: the 16:30 CT scheduled ASIA read must AUTHOR
// during the CME halt from the last stored bars.
//
// Live evidence 2026-09-01: class 32 fired the read at 16:31:05 on wall-clock,
// then plannerPreflight refused it — `stale_bars_1865s … stale_bars_3545s`,
// fifteen refusals 16:31→16:59 — because FEED_ALERT_S (600s) is unsatisfiable
// inside a halt by definition. The read launched 17:01:05 on the reopen tick
// and fail-closed at 17:23:14 (ASIA v1 planner_fail_closed). The halt refusal
// ate the 31 minutes that would have absorbed a retry BEFORE the open.
//
// Uses only the pre-fix surface (maybeRunSessionReadsAt + the frozen provider)
// so it compiles on the old tree and FAILS there: no plan row lands.

// class36HaltBars — bars ending at 16:00 CT (the last bar before the halt), so
// at 16:30 the newest bar is 30 minutes old: fresh enough to author from, far
// past FEED_ALERT_S=600s.
func class36HaltBars(lastBarOpen time.Time) func(string, string, int) []market.Kline {
	base := lastBarOpen.Add(-389 * time.Minute).UnixMilli()
	bars := make([]market.Kline, 0, 390)
	for i := 0; i < 390; i++ {
		o := base + int64(i)*60_000
		bars = append(bars, market.Kline{OpenTime: o, Open: 15600 + float64(i%10), High: 15650 + float64(i%10), Low: 15550 + float64(i%10), Close: 15600 + float64(i%10), CloseTime: o + 59_000})
	}
	return func(string, string, int) []market.Kline { return bars }
}

func TestClass36PinAsiaHalt(t *testing.T) {
	at, st := asiaClockTrader(t)
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	now := ctTime(t, 2026, 9, 1, 16, 30)     // Tuesday 16:30 CT — inside the 16:00-17:00 halt
	lastBar := ctTime(t, 2026, 9, 1, 15, 59) // newest stored 1m bar opened 15:59 (closed 16:00)
	market.FuturesBarsProvider = class36HaltBars(lastBar)
	testNow = func() time.Time { return now }
	t.Cleanup(func() { testNow = nil })

	if kernel.IsCMEOpen(now) {
		t.Fatal("fixture: 16:30 must be inside the CME maintenance halt")
	}
	if age, have := at.feedNewestBarAge(now); !have || age < 25*time.Minute {
		t.Fatalf("fixture: newest bar must be ~30m old at 16:30, got have=%v age=%s", have, age)
	}

	fired := at.maybeRunSessionReadsAt(now) // the scheduled path (class 32 evaluates it on wall-clock)
	if len(fired) != 1 || fired[0].Session != "ASIA" {
		t.Fatalf("fixture: the 16:30 ASIA read must be scheduled, fired=%+v", fired)
	}
	row := waitPlan(t, st, "2026-09-01", "ASIA", "t1")
	if row == nil {
		t.Fatal("CLASS 36: the scheduled ASIA read fired at 16:30 but the planner call was never MADE — the preflight refused it on staleness during the halt (stale_bars_*); the plan must be on the desk before 17:00")
	}
	if row.Lifecycle != "active" || row.TriggerReason != "ASIA_scheduled_read" {
		t.Fatalf("plan row wrong: %+v", row)
	}
}
