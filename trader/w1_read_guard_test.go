package trader

import (
	"testing"
	"time"

	"nofx/kernel"
)

// W1 — the planner-read guard + daily-roll window, proven with a weekend clock
// sweep. 2026-08-14=Fri · 08-15=Sat · 08-16=Sun · 08-17=Mon.

func w1CT(y int, mo time.Month, d, h, mi int) time.Time {
	loc, _ := time.LoadLocation("America/Chicago")
	return time.Date(y, mo, d, h, mi, 0, 0, loc)
}

// nyReadWouldFire is the EXACT predicate maybeRunSessionReads uses for NY.
func nyReadWouldFire(now time.Time) bool {
	return kernel.IsCMEOpen(now) && inSessionReadWindow(now, "08:25", "15:00")
}

func TestW1SundayReadGuard(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"Mon 08:25 → fire", w1CT(2026, 8, 17, 8, 25), true},
		{"Mon 08:24 → not yet", w1CT(2026, 8, 17, 8, 24), false},
		{"Mon 14:59 → still in window", w1CT(2026, 8, 17, 14, 59), true},
		{"Mon 15:00 → window closed", w1CT(2026, 8, 17, 15, 0), false},
		{"Sun 17:00 reopen → NO spurious NY read", w1CT(2026, 8, 16, 17, 0), false},
		{"Sun 20:00 → no", w1CT(2026, 8, 16, 20, 0), false},
		{"Sat 10:00 → CME closed", w1CT(2026, 8, 15, 10, 0), false},
		{"Fri 08:25 → fire", w1CT(2026, 8, 14, 8, 25), true},
	}
	for _, c := range cases {
		if got := nyReadWouldFire(c.at); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

// The whole-weekend sweep: NY read must be FALSE at every Sat + Sun step — no plan
// is ever written for a weekend, so Monday's is built fresh at 08:25.
func TestW1WeekendSweepNoNYRead(t *testing.T) {
	for _, day := range []int{15, 16} { // Sat, Sun
		for h := 0; h < 24; h++ {
			for _, mi := range []int{0, 15, 30, 45} {
				now := w1CT(2026, 8, day, h, mi)
				if nyReadWouldFire(now) {
					t.Fatalf("NY read must NOT fire on weekend 08-%d %02d:%02d", day, h, mi)
				}
			}
		}
	}
}

func TestW1DailyRollWindow(t *testing.T) {
	if !inDailyRollWindow(w1CT(2026, 8, 14, 15, 30)) {
		t.Fatal("Fri 15:30 (RTH-close→break) daily roll must fire")
	}
	for _, bad := range []time.Time{
		w1CT(2026, 8, 14, 14, 30), // pre-close
		w1CT(2026, 8, 14, 16, 30), // CME break
		w1CT(2026, 8, 14, 17, 0),  // post-roll (old buggy trigger)
	} {
		if inDailyRollWindow(bad) {
			t.Fatalf("daily roll must NOT fire at %s", bad.Format("15:04"))
		}
	}
	// Friday reachability: CME is open in the roll window (Globex runs to 16:00).
	if !kernel.IsCMEOpen(w1CT(2026, 8, 14, 15, 30)) {
		t.Fatal("CME must be open Fri 15:30 so the daily roll is reachable")
	}
}

// The ASIA wrap-around window is handled (16:55→02:00): Sunday 17:00 IS Monday's
// ASIA read time (spec: Sunday 17:00 = Asia of Monday's trade date).
func TestW1AsiaWrapWindow(t *testing.T) {
	if !inSessionReadWindow(w1CT(2026, 8, 16, 17, 0), "16:55", "02:00") {
		t.Fatal("ASIA read window must include Sunday 17:00 (Monday's ASIA)")
	}
	if !inSessionReadWindow(w1CT(2026, 8, 17, 1, 0), "16:55", "02:00") {
		t.Fatal("ASIA window must include 01:00 (wrapped past midnight)")
	}
	if inSessionReadWindow(w1CT(2026, 8, 17, 3, 0), "16:55", "02:00") {
		t.Fatal("ASIA window must EXCLUDE 03:00 (past 02:00 end)")
	}
}
