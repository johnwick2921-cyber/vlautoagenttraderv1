package ninjatrader

import "testing"

// A HOOK THAT LOSES WHEN ITS EVENT BEATS ITS INSTALLER (class candidate,
// 2026-09-16 15:32 CT boot of c6579347). The trader installs
// afterBackfillHook at load (auto_trader_dayplan.go); the backfill goroutine
// checked it at 15:32:03 and found nil; the trader installed it at 15:32:05.
// Silent: the 📈 regime line, the 🧮 planner-tape line and the R1 "📊 bars
// after backfill" line never printed. Order must not matter: whichever side
// arrives second fires the hook, exactly once.
func TestAfterBackfillHookFiresWhenInstalledAfterTheEvent(t *testing.T) {
	resetAfterBackfillHookForTest()
	fireAfterBackfillHook() // the event lands first — nobody is listening yet
	n := 0
	SetAfterBackfillHook(func() { n++ })
	if n != 1 {
		t.Fatalf("hook installed AFTER the backfill landed fired %d time(s), want exactly 1", n)
	}
	fireAfterBackfillHook() // a second landing (never happens; must not double-fire)
	if n != 1 {
		t.Fatalf("hook fired again on a second event: %d", n)
	}
}

func TestAfterBackfillHookFiresOnceWhenInstalledBeforeTheEvent(t *testing.T) {
	resetAfterBackfillHookForTest()
	n := 0
	SetAfterBackfillHook(func() { n++ })
	if n != 0 {
		t.Fatalf("hook fired before the backfill landed: %d", n)
	}
	fireAfterBackfillHook()
	fireAfterBackfillHook()
	if n != 1 {
		t.Fatalf("want exactly one firing, got %d", n)
	}
}
