package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ── BAR HORIZON (wave BARS HORIZON, 2026-09-09) ──────────────────────────────
//
// A17/A28: every clock in this file is STATED. Nothing reads time.Now().
//
// CIRCULARITY, NAMED: the fixtures below generate one bar per OPEN minute using
// IsCMEOpen — the same oracle HorizonOf consults. That makes these pins a test
// of the ARITHMETIC (open intervals minus served, closed runs skipped), not of
// the calendar. The calendar itself is pinned separately and by hand in
// kernel/bar_expectancy_test.go (item 3's P3), against a 12-row table nobody
// generated from IsCMEOpen.

func bhCT(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, CTLocation())
}

// openTape emits one 1m bar per OPEN minute in [from, to] inclusive, skipping
// any minute inside one of the holes (each hole is an inclusive [start, end]).
func openTape(from, to time.Time, holes ...[2]time.Time) []market.Kline {
	var out []market.Kline
	for t := from; !t.After(to); t = t.Add(time.Minute) {
		if !IsCMEOpen(t) {
			continue
		}
		holed := false
		for _, h := range holes {
			if !t.Before(h[0]) && !t.After(h[1]) {
				holed = true
				break
			}
		}
		if holed {
			continue
		}
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Close: 100, CloseTime: t.UnixMilli() + 59_999})
	}
	return out
}

// naiveGaps is the WRONG answer this wave exists to reject: span/step + 1 minus
// served, with no calendar. Computed here so every failure message carries the
// number the naive implementation would have produced.
func naiveGaps(h BarHorizon, stepMs int64) int {
	if h.Served == 0 {
		return -1
	}
	return int(h.SpanMs/stepMs) + 1 - h.Served
}

func TestHorizonCountsOnlyOpenMarketGapsNotClosures(t *testing.T) {
	// CASE A — a tape carrying every closure shape at once: the Friday 16:00 CT
	// weekly close, the whole weekend, the 2026-09-07 Labor Day shortened close
	// (12:00 CT, read from session_calendar.json), the Tuesday 16:00-17:00 daily
	// break — plus ONE deliberate 5-minute hole on Wednesday at 10:00 CT.
	// A weekend is not a gap. Only the five minutes are.
	nowA := bhCT(2026, time.September, 9, 10, 20)
	fromA, toA := bhCT(2026, time.September, 4, 15, 0), bhCT(2026, time.September, 9, 10, 14)
	tapeA := openTape(fromA, toA, [2]time.Time{bhCT(2026, time.September, 9, 10, 0), bhCT(2026, time.September, 9, 10, 4)})
	hA := HorizonOf(tapeA, "1m", len(tapeA)+5, nowA)
	if hA.GapCount != 5 {
		t.Fatalf("case A GapCount=%d, want 5 (naive span/step count says %d — the closures)", hA.GapCount, naiveGaps(hA, 60_000))
	}
	if n := naiveGaps(hA, 60_000); n != 3305 {
		t.Fatalf("case A naive gap count=%d, want 3305 — the fixture drifted; the pin's whole value is that 5 != naive", n)
	}
	if !hA.Holed() {
		t.Fatalf("case A Holed()=false, want true (GapCount=%d)", hA.GapCount)
	}

	// CASE B — the LIVE shape, 2026-09-09. A 1m tape whose oldest bar is
	// 2026-09-07 10:23 CT and whose newest is 13:18 CT on 09-09, with the
	// powered-off machine's 01:29 -> 13:04 CT hole inside it. Reconstructed
	// independently: this window yields EXACTLY 2000 served bars, which is what
	// planner_read_facts ids 65 and 66 recorded as scope_bars.
	nowB := bhCT(2026, time.September, 9, 13, 18)
	fromB, toB := bhCT(2026, time.September, 7, 10, 23), bhCT(2026, time.September, 9, 13, 18)
	tapeB := openTape(fromB, toB, [2]time.Time{bhCT(2026, time.September, 9, 1, 29), bhCT(2026, time.September, 9, 13, 4)})
	if len(tapeB) != 2000 {
		t.Fatalf("case B fixture served=%d, want 2000 (the live rows' scope_bars)", len(tapeB))
	}
	hB := HorizonOf(tapeB, "1m", 2000, nowB)
	if hB.GapCount != 696 {
		t.Fatalf("case B GapCount=%d, want 696 (naive says %d — 360 of those are the Labor Day halt and the Tuesday break)", hB.GapCount, naiveGaps(hB, 60_000))
	}
	if n := naiveGaps(hB, 60_000); n != 1056 {
		t.Fatalf("case B naive gap count=%d, want 1056", n)
	}
	// THE POINT OF THE WAVE: served == requested, so a served-count check alone
	// is SILENT here. Only the gap count speaks.
	if hB.Short() {
		t.Fatalf("case B Short()=true, want false — 2000 of 2000 were served; the defect is continuity, not count")
	}
	if !hB.Holed() {
		t.Fatalf("case B Holed()=false, want true")
	}
	if hB.SpanMs != 3055*60_000 {
		t.Fatalf("case B SpanMs=%d, want %d (3055 minutes)", hB.SpanMs, int64(3055)*60_000)
	}
	if hB.OldestAgeMs != 3055*60_000 {
		t.Fatalf("case B OldestAgeMs=%d, want %d", hB.OldestAgeMs, int64(3055)*60_000)
	}

	// CASE C — a zero-length tape is UNKNOWN, never a plausible zero (A24).
	hC := HorizonOf(nil, "1m", 2000, nowB)
	if hC.Served != 0 || hC.OldestAgeMs != -1 || hC.GapCount != -1 || hC.SpanMs != -1 {
		t.Fatalf("case C = %+v, want Served=0 OldestAgeMs=-1 GapCount=-1 SpanMs=-1", hC)
	}
	if hC.Note == "" {
		t.Fatalf("case C Note is empty — an UNKNOWN must say why (A24)")
	}
	if !hC.Empty() {
		t.Fatalf("case C Empty()=false, want true")
	}
}

func TestHorizonUnknownWhenScanIsCappedOrTFUnmapped(t *testing.T) {
	now := bhCT(2026, time.September, 9, 13, 18)
	from, to := bhCT(2026, time.September, 7, 10, 23), bhCT(2026, time.September, 9, 13, 18)
	tape := openTape(from, to, [2]time.Time{bhCT(2026, time.September, 9, 1, 29), bhCT(2026, time.September, 9, 13, 4)})

	// A capped scan reports UNKNOWN with the reason. It never reports a
	// truncated number as fact (A24).
	orig := barHorizonScanCap
	barHorizonScanCap = 10
	t.Cleanup(func() { barHorizonScanCap = orig; resetBarHorizonMemo() })
	resetBarHorizonMemo()
	h := HorizonOf(tape, "1m", 2000, now)
	if h.GapCount != -1 {
		t.Fatalf("capped scan GapCount=%d, want -1 (UNKNOWN when the scan is capped)", h.GapCount)
	}
	if !strings.Contains(h.Note, "cap") {
		t.Fatalf("capped scan Note=%q, want it to name the cap", h.Note)
	}

	// An unmapped timeframe is UNKNOWN too, not zero.
	barHorizonScanCap = orig
	resetBarHorizonMemo()
	hu := HorizonOf(tape, "7m", 2000, now)
	if hu.GapCount != -1 || hu.Note == "" {
		t.Fatalf("unmapped tf horizon = %+v, want GapCount=-1 with a Note", hu)
	}
}

func TestHorizonMemoIsKeyedOnTheWindowNotTheClock(t *testing.T) {
	resetBarHorizonMemo()
	t.Cleanup(resetBarHorizonMemo)
	from, to := bhCT(2026, time.September, 7, 10, 23), bhCT(2026, time.September, 9, 13, 18)
	tape := openTape(from, to, [2]time.Time{bhCT(2026, time.September, 9, 1, 29), bhCT(2026, time.September, 9, 13, 4)})

	before := barHorizonScans.Load()
	h1 := HorizonOf(tape, "1m", 2000, bhCT(2026, time.September, 9, 13, 18))
	afterFirst := barHorizonScans.Load()
	if afterFirst != before+1 {
		t.Fatalf("first HorizonOf ran %d calendar scans, want exactly 1", afterFirst-before)
	}
	// Same window, a clock 40 minutes later (the bridge is hit ~40x per cycle):
	// the age changes, the gap count cannot, and the scan must not repeat.
	h2 := HorizonOf(tape, "1m", 2000, bhCT(2026, time.September, 9, 13, 58))
	if got := barHorizonScans.Load(); got != afterFirst {
		t.Fatalf("second identical-window HorizonOf ran %d more calendar scans, want 0 (the memo)", got-afterFirst)
	}
	if h2.GapCount != h1.GapCount {
		t.Fatalf("memo returned GapCount=%d, want %d", h2.GapCount, h1.GapCount)
	}
	if h2.OldestAgeMs == h1.OldestAgeMs {
		t.Fatalf("OldestAgeMs did not move with the clock (%d) — the memo must not cache the age", h2.OldestAgeMs)
	}
}
