package kernel

import (
	"strings"
	"testing"
)

// CLASS 145 (W-DRIFT-WIDEN-CAP, 2026-09-17) — a halt's age read as clock drift
// widened a news blackout by half an hour. The 16:38:46 CT ASIA read inside
// the CME 16:00–17:00 halt measured 2,326,426 ms against the 15:59 bar's close
// (journal 45208: "widened by |drift| 2326426ms") and wrote "+39m (clock
// drift)" onto the BOJ ±15m band. These pins hold the widening to the cap and
// keep the clock claim off the card when the measurement is feed age.

// haltDriftMs is the LIVE measurement of 2026-09-17 16:38:46 CT: local clock
// minus (15:59 bar open + 60 s).
const haltDriftMs = int64(2_326_426)

func TestWidenCTWindowsCapsHaltAgeAtTwoMinutes(t *testing.T) {
	base := []CTWindow{{Start: 21*60 + 15, End: 21*60 + 45, Label: "BOJ Policy Rate 21:30 CT ±15m"}}
	got := WidenCTWindows(base, haltDriftMs)
	if got[0].Start != 21*60+13 || got[0].End != 21*60+47 {
		t.Fatalf("a 38m feed age must widen by the %dm cap, not 39m: got %d–%d", ClockWidenCapMinutes, got[0].Start, got[0].End)
	}
	if strings.Contains(got[0].Label, "clock drift") {
		t.Fatalf("feed age is not clock skew; the card must not claim it: %q", got[0].Label)
	}
	if got[0].Label != base[0].Label {
		t.Fatalf("label must be untouched when the measurement is staleness: %q", got[0].Label)
	}
	if ClockWidenMinutes(haltDriftMs) != ClockWidenCapMinutes {
		t.Fatalf("ClockWidenMinutes(%d) = %d, want the cap %d", haltDriftMs, ClockWidenMinutes(haltDriftMs), ClockWidenCapMinutes)
	}
}

func TestT1NoTradeLinesDriftHaltAgeUnlabelled(t *testing.T) {
	evs := []PlannerCalendarEvent{{TimeCT: "21:30", Impact: "T1", Title: "BOJ Policy Rate", Currency: "JPY"}}
	lines := T1NoTradeLinesDrift(evs, haltDriftMs)
	if len(lines) != 1 {
		t.Fatalf("want one line, got %v", lines)
	}
	if strings.Contains(lines[0], "clock drift") || strings.Contains(lines[0], "+39m") || strings.Contains(lines[0], "+31m") {
		t.Fatalf("the plan line must carry neither the clock claim nor the uncapped minutes: %q", lines[0])
	}
	if !strings.Contains(lines[0], "BOJ Policy Rate 21:30 CT ±15m") || !strings.Contains(lines[0], "HARD no-trade (red news)") {
		t.Fatalf("the line must still be the T1 blackout: %q", lines[0])
	}
}

// The existing behaviour below the cap is unchanged: a 42 s skew widens by one
// unlabelled boundary minute, a 90 s skew by two minutes labelled as drift.
func TestWidenCTWindowsBelowCapUnchanged(t *testing.T) {
	base := []CTWindow{{Start: 8*60 + 45, End: 9*60 + 15, Label: "ISM PMI 09:00 CT ±15m"}}

	w42 := WidenCTWindows(base, 42_000)
	if w42[0].Start != 8*60+44 || w42[0].End != 9*60+16 || strings.Contains(w42[0].Label, "clock drift") {
		t.Fatalf("42 s skew: want +1m unlabelled, got %d–%d %q", w42[0].Start, w42[0].End, w42[0].Label)
	}
	w90 := WidenCTWindows(base, 90_000)
	if w90[0].Start != 8*60+43 || w90[0].End != 9*60+17 || !strings.HasSuffix(w90[0].Label, "+2m (clock drift)") {
		t.Fatalf("90 s skew: want +2m (clock drift), got %d–%d %q", w90[0].Start, w90[0].End, w90[0].Label)
	}
	// Exactly at the plausible ceiling (5 min) the claim is still honest.
	w300 := WidenCTWindows(base, 5*60_000)
	if w300[0].Start != 8*60+43 || !strings.HasSuffix(w300[0].Label, "+2m (clock drift)") {
		t.Fatalf("5 min skew: want the cap, labelled: got %d %q", w300[0].Start, w300[0].Label)
	}
	// One millisecond past it the same widening loses the claim.
	w301 := WidenCTWindows(base, 5*60_000+1)
	if w301[0].Start != 8*60+43 || strings.Contains(w301[0].Label, "clock drift") {
		t.Fatalf("beyond 5 min: want the cap, unlabelled: got %d %q", w301[0].Start, w301[0].Label)
	}
	// Negative (feed in the future) beyond the plausible range: capped, unlabelled.
	wneg := WidenCTWindows(base, -20*60_000)
	if wneg[0].Start != 8*60+43 || strings.Contains(wneg[0].Label, "clock drift") {
		t.Fatalf("-20 min: want the cap, unlabelled: got %d %q", wneg[0].Start, wneg[0].Label)
	}
}

func TestClockDriftStaleNote(t *testing.T) {
	for _, d := range []int64{0, 108, 42_000, 90_000, 5 * 60_000, -61_000} {
		if n := ClockDriftStaleNote(d); n != "" {
			t.Fatalf("drift %d is clock-plausible, note must be empty, got %q", d, n)
		}
	}
	n := ClockDriftStaleNote(haltDriftMs)
	for _, want := range []string{"feed stale 38m", "halt or gap", "not clock skew", "NOT widened beyond the 2m cap"} {
		if !strings.Contains(n, want) {
			t.Fatalf("stale note %q missing %q", n, want)
		}
	}
	neg := ClockDriftStaleNote(-20 * 60_000)
	if !strings.Contains(neg, "20m in the FUTURE") || !strings.Contains(neg, "2m cap") {
		t.Fatalf("negative note must name the broken clock and the cap: %q", neg)
	}
}
