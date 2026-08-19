package trader

import (
	"testing"

	"nofx/kernel"
)

// The digest "session closed" predicate is wrap-aware (audit finding, class 2).
// Old: ctMinutesNow >= end — true for the rest of the DAY once the end minute
// passed, so ASIA (end 02:00) read as closed at 21:00 CT while its evening leg
// was running, producing a mid-session digest with mid-session P&L.
func TestDigestClosePredicateIsWrapAware(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	var asia *kernel.SessionDef
	for i := range reg.Sessions {
		if reg.Sessions[i].Name == kernel.SessionAsia {
			asia = &reg.Sessions[i]
		}
	}
	if asia == nil {
		t.Fatal("no ASIA in the default registry")
	}
	for _, tc := range []struct {
		name     string
		date, tm string
		closed   bool
	}{
		{"asia 21:00 RUNNING (the old bug said closed)", "2026-08-18", "21:00", false},
		{"asia 03:00 closed (instance ended 02:00)", "2026-08-18", "03:00", true},
		{"asia 16:30 still closed (next opens 17:00)", "2026-08-18", "16:30", true},
		{"asia 17:05 running again", "2026-08-18", "17:05", false},
	} {
		now := ctTimeAt(t, tc.date, tc.tm)
		if got := !asia.InWindow(now); got != tc.closed {
			t.Errorf("%s: closed=%v want %v", tc.name, got, tc.closed)
		}
	}
}
