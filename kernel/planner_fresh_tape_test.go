package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// freshTape1m builds a run of 1m bars ending at openTime `until`, stepping
// backwards, with closes decreasing by step. The LAST bar of every 5m bucket
// carries the bucket's close.
func freshTape1m(until int64, minutes int, closeAtEnd float64, step float64) []market.Kline {
	bars := make([]market.Kline, 0, minutes)
	closeVal := closeAtEnd + step*float64(minutes)
	for i := minutes - 1; i >= 0; i-- {
		open := until - int64(i+1)*60_000
		closeVal -= step
		bars = append(bars, market.Kline{
			OpenTime:  open,
			Close:     closeVal,
			CloseTime: open + 60_000,
		})
	}
	return bars
}

func TestPlannerFreshTapeIncludesOnlyCompletedBarsBetweenReadAndPublish(t *testing.T) {
	loc := CTLocation()
	read := time.Date(2026, 9, 23, 19, 0, 0, 0, loc)
	publish := time.Date(2026, 9, 23, 19, 7, 0, 0, loc)

	// 1m bars 18:55..19:06 plus a FORMING bar (openTime 19:07, close 99999).
	bars := freshTape1m(publish.Add(-time.Minute).UnixMilli(), 12, 15485, 1)
	forming := market.Kline{OpenTime: publish.UnixMilli(), Close: 99999, CloseTime: publish.UnixMilli() + 60_000}
	bars = append(bars, forming)

	reason := `born-dead authored scenario: S1: authored condition "2x5m<15470" breached between read and publish`
	got := PlannerFreshTape(bars, read, publish, reason)

	for _, want := range []string{
		"FRESH TAPE SINCE YOUR READ (previous attempt refused born-dead / flip-met)",
		reason,
		"Completed 1m closes between the read clock",
		"19:06 CT 1m close",
		"Completed 5m closes (last 6, newest last)",
		"5m close",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("block missing %q:\n%s", want, got)
		}
	}
	// The forming bar's close does not exist yet — it must never be printed.
	if strings.Contains(got, "99999") {
		t.Fatalf("forming bar leaked into the fresh tape:\n%s", got)
	}
	// The close at exactly the read clock is not "between" read and publish.
	if strings.Contains(got, "19:00 CT 1m close") {
		t.Fatalf("a close at the read clock leaked into the window:\n%s", got)
	}
}

func TestPlannerFreshTapeBoundsAndEmptyTape(t *testing.T) {
	loc := CTLocation()
	read := time.Date(2026, 9, 23, 19, 0, 0, 0, loc)
	publish := time.Date(2026, 9, 23, 19, 47, 0, 0, loc)

	// 60 minutes of completed 1m bars → the 1m list is bounded to the last 30
	// minutes and the 5m list to the last 6 buckets.
	bars := freshTape1m(publish.UnixMilli(), 60, 15500, 1)
	got := PlannerFreshTape(bars, read, publish, "flip met")
	if strings.Contains(got, "18:48 CT 1m close") {
		t.Fatalf("1m list is not bounded to the last 30 completed minutes:\n%s", got)
	}
	// Count 5m rows: exactly 6.
	if n := strings.Count(got, " 5m close "); n != 6 {
		t.Fatalf("5m list must carry exactly 6 rows, got %d:\n%s", n, got)
	}

	// No tape at all → explicit empty lines, never fabricated closes.
	empty := PlannerFreshTape(nil, read, publish, "flip met")
	for _, want := range []string{"(none — no completed 1m closes in the window)", "(none — no 5m bucket closed in the window)"} {
		if !strings.Contains(empty, want) {
			t.Fatalf("empty tape must say so (%q):\n%s", want, empty)
		}
	}
}
