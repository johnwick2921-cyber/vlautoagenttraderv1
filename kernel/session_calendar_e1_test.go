// E1 — THE PIN THE BRANCH NEVER RAN.
//
// 2026-09-07 is Labor Day. The calendar classifies it SHORTENED with a 12:00 CT
// close, because the feed carried 980 session bars across 153.50 points while
// the gate called the market shut. A shortened session is a TRADING session:
// the gate must be OPEN inside it and CLOSED after its stated close.
//
// Against the binary running on 2026-09-07 this test FAILS: isCMEHoliday treats
// the first Monday of September as a full closure and IsCMEOpen returns false
// for every hour of the day.
package kernel

import (
	"testing"
	"time"
)

// laborDay2026 at the given CT hour/minute.
func laborDayAt(h, m int) time.Time {
	return time.Date(2026, 9, 7, h, m, 0, 0, CTLocation())
}

// E1.a — inside the shortened session the market is OPEN.
func TestE1_ShortenedDayTradesBeforeItsClose(t *testing.T) {
	day, listed := SessionDayFor(laborDayAt(9, 0))
	if !listed || day.Class != SessionShortened {
		t.Fatalf("fixture: 2026-09-07 must be listed SHORTENED, got listed=%v class=%q", listed, day.Class)
	}
	if day.CloseCT != "12:00" {
		t.Fatalf("fixture: close_ct must be 12:00, got %q", day.CloseCT)
	}
	for _, hm := range [][2]int{{4, 0}, {9, 0}, {11, 59}} {
		now := laborDayAt(hm[0], hm[1])
		if !IsCMEOpen(now) {
			t.Errorf("E1: %02d:%02d CT on a SHORTENED day must be OPEN — a shortened session is a trading session; IsCMEOpen said closed", hm[0], hm[1])
		}
		if closed, reason := CMEClosedReason(now); closed {
			t.Errorf("E1: %02d:%02d CT on a SHORTENED day must not be closed; got reason %q", hm[0], hm[1], reason)
		}
	}
}

// E1.b — after the stated close the market is CLOSED, and says why.
func TestE1_ShortenedDayIsFlatAfterItsClose(t *testing.T) {
	for _, hm := range [][2]int{{12, 0}, {13, 30}, {15, 0}} {
		now := laborDayAt(hm[0], hm[1])
		if IsCMEOpen(now) {
			t.Errorf("E1: %02d:%02d CT is past the 12:00 CT early close — must be CLOSED", hm[0], hm[1])
		}
		closed, reason := CMEClosedReason(now)
		if !closed {
			t.Errorf("E1: %02d:%02d CT must report closed", hm[0], hm[1])
			continue
		}
		if reason == "holiday" {
			t.Errorf("E1: %02d:%02d CT reason must name the EARLY CLOSE, not \"holiday\" — the day traded; got %q", hm[0], hm[1], reason)
		}
	}
}

// E1.c — the invariant the old code already guarantees must survive: the bool
// from CMEClosedReason is exactly !IsCMEOpen, on a shortened day too.
func TestE1_ReasonMirrorsOpenOnAShortenedDay(t *testing.T) {
	for h := 0; h < 24; h++ {
		now := laborDayAt(h, 0)
		closed, _ := CMEClosedReason(now)
		if closed == IsCMEOpen(now) {
			t.Fatalf("E1: invariant broken at %02d:00 CT — CMEClosedReason=%v but IsCMEOpen=%v", h, closed, IsCMEOpen(now))
		}
	}
}

// E1.d — THE EVENING REOPEN. A shortened day halts at its stated close and
// REOPENS at the ordinary daily boundary; the evening belongs to the next
// trading day. CME's own wording for 2026-09-07 is "equity futures halt 12:00
// CT, reopen 17:00 CT".
//
// Without this the whole calendar date stayed shut and tonight's ASIA session
// was skipped — the same loss this calendar exists to prevent, one layer down.
// Found at cutover, by reading the source line the fold had carried over.
func TestE1_ShortenedDayReopensInTheEvening(t *testing.T) {
	for _, c := range []struct {
		h    int
		open bool
		why  string
	}{
		{11, true, "before the early close"},
		{12, false, "the early close itself"},
		{16, false, "still halted"},
		{17, true, "the ordinary daily reopen"},
		{22, true, "the evening session runs"},
	} {
		now := laborDayAt(c.h, 0)
		if got := IsCMEOpen(now); got != c.open {
			t.Errorf("E1.d: %02d:00 CT (%s) open=%v, want %v", c.h, c.why, got, c.open)
		}
	}
	// And the reason names the early close during the halt, not "holiday".
	if _, r := CMEClosedReason(laborDayAt(14, 0)); r != "early close 12:00 CT" {
		t.Errorf("E1.d: halt reason = %q, want \"early close 12:00 CT\"", r)
	}
}
