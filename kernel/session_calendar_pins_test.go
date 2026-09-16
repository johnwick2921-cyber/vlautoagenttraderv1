// E2/E3/E4/E6 — the rest of the calendar's pins.
package kernel

import (
	"strings"
	"testing"
	"time"
)

func at(y int, m time.Month, d, h, mi int) time.Time {
	return time.Date(y, m, d, h, mi, 0, 0, CTLocation())
}

// E2 — a FULL-CLOSURE date never trades, at any hour.
func TestE2_FullClosureNeverTrades(t *testing.T) {
	for _, h := range []int{0, 6, 10, 13, 18, 23} {
		now := at(2026, 12, 25, h, 0) // Christmas Day
		if IsCMEOpen(now) {
			t.Errorf("E2: Christmas Day %02d:00 CT must be closed", h)
		}
	}
	if st := SessionStateAt(at(2026, 12, 25, 10, 0)); st.Class != SessionClosed {
		t.Errorf("E2: want closed, got %s", st.Class)
	}
}

// E3 — a NORMAL day is decided by the weekly rules alone, exactly as before the
// calendar existed. The calendar must add nothing on an ordinary Tuesday.
func TestE3_NormalDayIsTheWeeklyRuleAlone(t *testing.T) {
	for _, c := range []struct {
		h    int
		want bool
	}{{9, true}, {16, false}, {18, true}} { // Mon-Thu: open except the 16:00 break
		now := at(2026, 8, 18, c.h, 0) // an ordinary Tuesday
		if got := IsCMEOpen(now); got != c.want {
			t.Errorf("E3: normal Tuesday %02d:00 = %v, want %v", c.h, got, c.want)
		}
		if got := weeklyCMEOpen(now); got != c.want {
			t.Errorf("E3: the weekly rule alone must give the same answer at %02d:00", c.h)
		}
	}
	// And it contributes no status note.
	if note := SessionDayNote(at(2026, 8, 18, 9, 0)); note != "" {
		t.Errorf("E3: a normal unlisted day must add no note, got %q", note)
	}
}

// E4 — a date present but UNCLASSIFIABLE is closed AND named, never silently
// normal. Driven through the real data: the five unestablished rows.
func TestE4_UnestablishedIsClosedAndNamed(t *testing.T) {
	n := SessionCalendarUnestablishedCount()
	if n == 0 {
		t.Fatalf("E4: fixture — the calendar must carry unestablished rows to pin this")
	}
	for _, d := range sessionCal.Dates {
		if !d.Unestablished {
			continue
		}
		when, err := time.ParseInLocation("2006-01-02", d.Date, CTLocation())
		if err != nil {
			t.Fatalf("bad date %q", d.Date)
		}
		noon := when.Add(10 * time.Hour)
		if IsCMEOpen(noon) {
			t.Errorf("E4: %s is UNESTABLISHED and must not trade", d.Date)
		}
		note := SessionDayNote(noon)
		if !strings.Contains(note, "UNESTABLISHED") {
			t.Errorf("E4: %s must be NAMED as unestablished on the surface, got %q", d.Date, note)
		}
		if !strings.Contains(SessionCalendarBootLine(noon), "unestablished") {
			t.Errorf("E4: %s must be named unestablished on the boot line", d.Date)
		}
	}
	// An unreadable close_ct is the other unclassifiable shape.
	st := SessionStateAt(at(2026, 8, 18, 9, 0))
	if st.CloseUnreadable {
		t.Errorf("E4: a normal day must not be flagged unreadable")
	}
}

// E6 — the clock seam. Every function takes `now`; a fixed clock gives the same
// answer whenever the suite runs, and the two sides of a shortened day differ.
func TestE6_ClockSeamFixedNotWall(t *testing.T) {
	morning := at(2026, 9, 7, 8, 0)     // before the 12:00 close
	afternoon := at(2026, 9, 7, 14, 50) // after it
	if !IsCMEOpen(morning) {
		t.Errorf("E6: 08:00 CT on the shortened day must be open")
	}
	if IsCMEOpen(afternoon) {
		t.Errorf("E6: 14:50 CT on the shortened day must be closed")
	}
	// Same instants, evaluated twice — no hidden wall-clock read anywhere.
	for i := 0; i < 3; i++ {
		if !IsCMEOpen(morning) || IsCMEOpen(afternoon) {
			t.Fatalf("E6: repeated evaluation changed the answer — something reads time.Now()")
		}
	}
	// The boot line is stable for a fixed clock too.
	if SessionCalendarBootLine(morning) != SessionCalendarBootLine(morning) {
		t.Errorf("E6: the boot line must be a pure function of its clock")
	}
}
