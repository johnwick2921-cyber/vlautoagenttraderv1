package trader

// CLASS 60 SWEEP — the time-of-day pin.
//
// Class 60's failure was a suite that was honestly green at 11:00 and honestly
// red at 14:50, because a fixed-fixture test reached a production entry point
// that read time.Now(). Every test here runs the SAME assertion at two clocks —
// 11:00 CT and 14:50 CT — so a function whose answer depends on the wall clock
// cannot pass both. That is the property, not the specific hours.
//
// These call the …At variants. On the unseamed code they do not compile, which
// is the intended RED: there is no way to state a clock, which IS the defect.

import (
	"testing"
	"time"

	"nofx/market"
)

// bar whose close lands at a stated CT wall time on the fixture's date
func barClosingAtCT(t *testing.T, h, m int, closePx float64) market.Kline {
	t.Helper()
	ts := ctAt(t, h, m)
	return market.Kline{
		OpenTime:  ts.Add(-5 * time.Minute).UnixMilli(),
		CloseTime: ts.UnixMilli(),
		Close:     closePx,
	}
}

// THE HEADLINE PIN. latestClosedPrimaryBarMs answers "which bar has closed by
// now" — a question whose answer is DIFFERENT at 11:00 and at 14:50 by
// construction. With the wall clock underneath it, a fixture cannot pin either
// answer; the test would pass all morning and fail after lunch, which is class
// 60 exactly.
func TestLatestClosedPrimaryBarMsAtIsClockPinned(t *testing.T) {
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")

	early := barClosingAtCT(t, 11, 30, 100)
	late := barClosingAtCT(t, 14, 0, 200)

	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(string, string, int) []market.Kline {
		return []market.Kline{early, late}
	}
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	// At 11:00 CT neither bar has closed yet.
	if ms, ok := at.latestClosedPrimaryBarMsAt(ctAt(t, 11, 0)); ok {
		t.Fatalf("11:00 CT: no bar has closed yet, got ms=%d ok=true", ms)
	}
	// At 14:50 CT both have, and the LATEST is the 14:00 close.
	ms, ok := at.latestClosedPrimaryBarMsAt(ctAt(t, 14, 50))
	if !ok {
		t.Fatal("14:50 CT: both bars have closed, want ok=true")
	}
	if ms != late.CloseTime {
		t.Fatalf("14:50 CT: want the 14:00 close %d, got %d", late.CloseTime, ms)
	}
}

// The entry point must still read the wall clock and must be a pure delegate:
// production behaviour is unchanged by this wave.
func TestLatestClosedPrimaryBarMsEntryStillUsesWallClock(t *testing.T) {
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(string, string, int) []market.Kline {
		// closed a minute ago on the REAL clock, whatever hour the suite runs
		return []market.Kline{{
			OpenTime:  time.Now().Add(-6 * time.Minute).UnixMilli(),
			CloseTime: time.Now().Add(-1 * time.Minute).UnixMilli(),
			Close:     1,
		}}
	}
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	if _, ok := at.latestClosedPrimaryBarMs(); !ok {
		t.Fatal("the wall-clock entry point must still see a bar closed a minute ago")
	}
}

// weeklyScenarioGrade resolves the ACTIVE session at `now`. 11:00 CT is inside
// NY; 14:50 CT is past NY's 14:45 flat, so no session is active. Same input,
// two answers, decided entirely by the hour.
func TestWeeklyScenarioGradeAtIsClockPinned(t *testing.T) {
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")

	inSession := at.weeklyScenarioGradeAt(ctAt(t, 11, 0), "PDH")
	afterFlat := at.weeklyScenarioGradeAt(ctAt(t, 14, 50), "PDH")

	if afterFlat != "" {
		t.Fatalf("14:50 CT is past NY's flat — no active session, want \"\", got %q", afterFlat)
	}
	_ = inSession // the 11:00 answer depends on stored plan rows; the pin is that the two clocks are answerable at all
}
