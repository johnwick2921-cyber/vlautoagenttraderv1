package kernel

import (
	"testing"
	"time"
)

// W12 — session-window + day-roll DST correctness across BOTH 2026 US transitions
// (spring-forward Sun 2026-03-08, fall-back Sun 2026-11-01). Every window is
// wall-clock CT via America/Chicago (tzdb), NOT a fixed UTC offset. The proof: the
// SAME CT wall-clock is IN-window on both sides of a transition, yet maps to
// DIFFERENT UTC instants (a 1h DST shift).
func TestW12SessionWindowsDSTCorrect(t *testing.T) {
	ct, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("tz: %v", err)
	}

	// 08:30 CT (NY RTH open) is in [08:30,15:00) on a CST day AND a CDT day.
	cst := time.Date(2026, 3, 7, 8, 30, 0, 0, ct) // Sat before spring-forward (CST, -6)
	cdt := time.Date(2026, 3, 9, 8, 30, 0, 0, ct) // Mon after (CDT, -5)
	if !InBlackoutWindow(cst, "08:30", "15:00") || !InBlackoutWindow(cdt, "08:30", "15:00") {
		t.Fatal("08:30 CT must be in the NY window on both CST and CDT days")
	}
	// 08:29 CT is NOT (boundary exactness holds across DST).
	if InBlackoutWindow(time.Date(2026, 3, 9, 8, 29, 0, 0, ct), "08:30", "15:00") {
		t.Fatal("08:29 CT must be outside the window")
	}
	// DST IS applied: same wall-clock → UTC differs by exactly 1h (14:30 vs 13:30).
	if cst.UTC().Hour() != 14 || cdt.UTC().Hour() != 13 {
		t.Fatalf("expected 14:30 UTC (CST) and 13:30 UTC (CDT), got %02d / %02d", cst.UTC().Hour(), cdt.UTC().Hour())
	}

	// Fall-back: 08:30 CT on 2026-10-31 (CDT) vs 2026-11-02 (CST) — both in-window.
	fallCDT := time.Date(2026, 10, 31, 8, 30, 0, 0, ct)
	fallCST := time.Date(2026, 11, 2, 8, 30, 0, 0, ct)
	if !InBlackoutWindow(fallCDT, "08:30", "15:00") || !InBlackoutWindow(fallCST, "08:30", "15:00") {
		t.Fatal("08:30 CT must be in-window across the fall-back too")
	}
	if fallCDT.UTC().Hour() != 13 || fallCST.UTC().Hour() != 14 {
		t.Fatalf("fall-back UTC: got %02d / %02d, want 13 / 14", fallCDT.UTC().Hour(), fallCST.UTC().Hour())
	}
}

// W12 — the 17:00 CT session-day roll is DST-correct across the transitions.
func TestW12SessionDayRollDST(t *testing.T) {
	ct, _ := time.LoadLocation("America/Chicago")
	cases := []struct {
		when time.Time
		want string
	}{
		{time.Date(2026, 3, 8, 16, 0, 0, 0, ct), "2026-03-07"}, // before 17:00 → prior day
		{time.Date(2026, 3, 8, 18, 0, 0, 0, ct), "2026-03-08"}, // after 17:00 → same day
		{time.Date(2026, 3, 9, 8, 30, 0, 0, ct), "2026-03-08"}, // Mon morning → Sun-evening session-day
		{time.Date(2026, 11, 1, 16, 0, 0, 0, ct), "2026-10-31"}, // fall-back before 17:00
		{time.Date(2026, 11, 1, 18, 0, 0, 0, ct), "2026-11-01"}, // fall-back after 17:00
	}
	for _, c := range cases {
		if got := CMESessionDayKey(c.when); got != c.want {
			t.Fatalf("CMESessionDayKey(%s) = %s, want %s", c.when.Format("2006-01-02 15:04 MST"), got, c.want)
		}
	}
}

// W12 — the wrap-around ASIA window [17:00,02:00) CT is correct (crosses midnight).
func TestW12WrapWindowDST(t *testing.T) {
	ct, _ := time.LoadLocation("America/Chicago")
	if !InBlackoutWindow(time.Date(2026, 3, 8, 20, 0, 0, 0, ct), "17:00", "02:00") {
		t.Fatal("20:00 CT must be inside the wrapping ASIA window")
	}
	if !InBlackoutWindow(time.Date(2026, 3, 9, 1, 0, 0, 0, ct), "17:00", "02:00") {
		t.Fatal("01:00 CT (post-midnight) must be inside the wrapping ASIA window")
	}
	if InBlackoutWindow(time.Date(2026, 3, 9, 3, 0, 0, 0, ct), "17:00", "02:00") {
		t.Fatal("03:00 CT must be OUTSIDE the wrapping window")
	}
}
