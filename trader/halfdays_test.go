// HALF-DAYS AFTER THE FOLD (owner ruling 2026-09-07).
//
// These tests used to pin a FILE loader: half_days.json at the repo root, parsed
// and seeded into SessionRegistry.HalfDays. That mechanism is gone — it was a
// second dated calendar with its own key convention that, on three dates,
// disagreed with kernel/session_calendar.json, and the gate stopped trading at
// 12:00 on days its own SOURCED rows said 12:15.
//
// The coverage is not dropped, it is re-aimed: what must now hold is that the
// half-day surfaces DERIVE from the calendar and cannot drift from the gate.
package trader

import (
	"testing"
	"time"

	"nofx/kernel"
)

// The derivation returns exactly the calendar's shortened days — no more, no
// fewer, and with the calendar's values.
func TestHalfDaysDeriveFromTheCalendar(t *testing.T) {
	entries, err := LoadHalfDaysFile()
	if err != nil {
		t.Fatalf("derivation must not error: %v", err)
	}
	want := kernel.SessionShortenedDays()
	if len(entries) != len(want) {
		t.Fatalf("derived %d half-days, calendar holds %d shortened days", len(entries), len(want))
	}
	byDate := map[string]string{}
	for _, e := range entries {
		byDate[e.Date] = e.EarlyCloseCT
	}
	for _, d := range want {
		got, ok := byDate[d.Date]
		if !ok {
			t.Errorf("calendar has shortened day %s but the derivation dropped it", d.Date)
			continue
		}
		if got != d.CloseCT {
			t.Errorf("%s: derived close %q != calendar close %q — a second copy of the fact", d.Date, got, d.CloseCT)
		}
	}
}

// The two dates the fold corrected carry the SOURCED time, not the guess.
func TestHalfDaysCarryTheSourcedCloseTimes(t *testing.T) {
	entries, _ := LoadHalfDaysFile()
	want := map[string]string{"2026-11-27": "12:15", "2026-12-24": "12:15"}
	seen := map[string]bool{}
	for _, e := range entries {
		if w, ok := want[e.Date]; ok {
			seen[e.Date] = true
			if e.EarlyCloseCT != w {
				t.Errorf("%s early close = %q, want the sourced %q (CME archived calendar: equity FINAL close, settlement 12:00)", e.Date, e.EarlyCloseCT, w)
			}
		}
		// Thanksgiving is a FULL closure by owner ruling — never a half-day.
		if e.Date == "2026-11-26" {
			t.Errorf("2026-11-26 is a full closure and must not appear as a half-day; got %q", e.EarlyCloseCT)
		}
	}
	for d := range want {
		if !seen[d] {
			t.Errorf("%s must be derived as a half-day", d)
		}
	}
}

// NextUpcomingHalfDay is clock-driven and takes its clock (A28).
func TestNextUpcomingHalfDayTakesItsClock(t *testing.T) {
	entries, _ := LoadHalfDaysFile()
	// Just after Labor Day 2026: the next early close is the day after
	// Thanksgiving, because Thanksgiving itself is a full closure.
	got, ok := NextUpcomingHalfDay(entries, time.Date(2026, 9, 8, 9, 0, 0, 0, kernel.CTLocation()))
	if !ok {
		t.Fatalf("expected an upcoming half-day after 2026-09-08")
	}
	if got.Date != "2026-11-27" {
		t.Errorf("next half-day after 2026-09-08 = %s, want 2026-11-27", got.Date)
	}
	// Past the last one, there is none — reported as absent, never as a zero row.
	if _, ok := NextUpcomingHalfDay(entries, time.Date(2027, 1, 2, 9, 0, 0, 0, kernel.CTLocation())); ok {
		t.Errorf("after the last calendar entry there must be no upcoming half-day")
	}
}

// The deleted file must not come back by accident.
func TestHalfDaysFileIsGone(t *testing.T) {
	if _, err := LoadHalfDaysFile(); err != nil {
		t.Fatalf("the derivation must work with no half_days.json present: %v", err)
	}
}
