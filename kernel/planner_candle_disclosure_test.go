package kernel

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ── D1 PINS (wave BARS HORIZON, 2026-09-09) ─────────────────────────────────
//
// THE HEADING LIES, AND THE MODEL IS TOLD TO TRUST IT.
//
// BuildPlannerCandleTablesAt truncates only when the slice is LONG and bakes
// the count into the title as a literal, so "daily session candles (last 8)"
// stood over 3 rows. MEASURED on the stored prompts (planner_rejected_prompts,
// n=54 carrying a Candles block, ids 70..142): 15m rendered 12 in 54/54, 1h 12
// in 54/54, 4h 8 in 54/54 — and the daily table rendered 2 rows in 19 and 3
// rows in 35. NEVER 8, in 0 of 54 (re-measured 18:3x CT: n=55, ids 70..143,
// daily 2 rows in 20 and 3 in 35 — NEVER 8, in 0 of 55).
//
// Worse, the OLDEST daily row is a PARTIAL session candle presented as a whole
// one: DailySessionBars takes the first bar it sees as the session Open with no
// completeness check. Live example, prompt id 142: the oldest daily row is
// stamped "09-07 02:39" — the session it claims opened at 09-06 17:00 CT.
//
// And kernel/planner_prompt.go tells the model "On conflict, trust the candles".
//
// A24 ABSOLUTE: these pins require the partial row to be MARKED. Nothing here
// may synthesise, interpolate or carry forward a bar to complete it.

// d1Tape builds 1m bars at every OPEN CME minute in [from, to), so a fixture
// never has to hand-model the daily break or the weekend.
func d1Tape(from, to time.Time) []market.Kline {
	var out []market.Kline
	i := 0
	for t := from; t.Before(to); t = t.Add(time.Minute) {
		if !IsCMEOpen(t) {
			continue
		}
		o := 29500.0 + float64(i%40)*0.25
		out = append(out, market.Kline{
			OpenTime: t.UnixMilli(), Open: o, High: o + 2, Low: o - 2, Close: o + 0.5, Volume: 12,
		})
		i++
	}
	return out
}

// d1Fixture reproduces the live shape of prompt id 142: a tape that starts
// MID-SESSION (02:39 CT) and ends mid-session, so the oldest daily row is
// front-truncated and the newest is still forming.
func d1Fixture(t *testing.T) ([]market.Kline, time.Time) {
	t.Helper()
	from := time.Date(2026, 9, 8, 2, 39, 0, 0, CTLocation())
	now := time.Date(2026, 9, 9, 13, 18, 13, 0, CTLocation())
	bars := d1Tape(from, now)
	if len(bars) == 0 {
		t.Fatalf("fixture built no bars")
	}
	return bars, now
}

func d1Section(t *testing.T, table, heading string) string {
	t.Helper()
	i := strings.Index(table, heading)
	if i < 0 {
		t.Fatalf("section %q absent from table:\n%s", heading, table)
	}
	rest := table[i:]
	if j := strings.Index(rest[len(heading):], "\n### "); j >= 0 {
		rest = rest[:len(heading)+j]
	}
	return rest
}

// d1Rows returns ONLY the rendered candle rows of a section (lines stamped
// "MM-DD HH:MM"), so an assertion about a ROW MARKER can never be satisfied by
// the heading — which names ⚠PARTIAL / ⏳FORMING / ❓COVERAGE-UNKNOWN in its own
// legend clause. The first version of these pins DID match the heading and a
// mutation that disabled the whole UNCOVERED branch passed (class 89).
func d1Rows(sec string) string {
	var out []string
	for _, ln := range strings.Split(sec, "\n") {
		if len(ln) > 11 && ln[2] == '-' && ln[5] == ' ' && ln[8] == ':' {
			out = append(out, ln)
		}
	}
	return strings.Join(out, "\n")
}

// PIN D1-A — THE HEADING STATES HELD-vs-CLAIMED, ON EVERY TABLE.
// A disclosure that only appears on failure is one the reader learns to skim
// past, so the COMPLETE tables must say so too.
func TestCandleHeadingsStateHeldVsClaimed(t *testing.T) {
	bars, now := d1Fixture(t)
	table := BuildPlannerCandleTablesAt(bars, 12000, now)

	for _, want := range []string{
		"### 15m — HELD 12 of 12 requested rows",
		"### 1h — HELD 12 of 12 requested rows",
		"### 4h — HELD 8 of 8 requested rows",
		"### daily session candles — HELD 2 of 8 requested rows",
	} {
		if !strings.Contains(table, want) {
			t.Errorf("heading missing %q\n--- rendered ---\n%s", want, table)
		}
	}
	// The literal count must be GONE from every heading — it is the lie.
	for _, gone := range []string{"(last 12)", "(last 8)"} {
		if strings.Contains(table, gone) {
			t.Errorf("heading still carries the baked literal %q — a heading cannot say 8 over 2 rows", gone)
		}
	}
	// SHORT is named, with the shortfall, and the absent rows are not inferable.
	if !strings.Contains(table, "SHORT BY 6") {
		t.Errorf("daily heading does not name the shortfall (want \"SHORT BY 6\")\n%s", d1Section(t, table, "### daily session candles"))
	}
}

// PIN D1-B — A PARTIAL SESSION CANDLE IS MARKED, AND NO BAR IS INVENTED.
func TestPartialSessionCandleIsMarkedNeverSynthesised(t *testing.T) {
	bars, now := d1Fixture(t)
	table := BuildPlannerCandleTablesAt(bars, 12000, now)
	sec := d1Section(t, table, "### daily session candles")

	rows := d1Rows(sec)
	if !strings.Contains(rows, "⚠PARTIAL — holds ") {
		t.Errorf("the front-truncated session ROW is NOT marked partial:\n%s", sec)
	}
	if !strings.Contains(rows, "⏳FORMING — the window has not closed") {
		t.Errorf("the still-open session ROW is NOT marked forming:\n%s", sec)
	}
	// A24: marking must not change the DATA. The rendered OHLC rows must still
	// be exactly what DailySessionBars produced — same count, same open time.
	want := DailySessionBars(bars)
	if len(want) > 8 {
		want = want[len(want)-8:]
	}
	if n := len(strings.Split(rows, "\n")); n != len(want) {
		t.Errorf("rendered %d daily rows, DailySessionBars produced %d — a row was invented or dropped", n, len(want))
	}
	for _, k := range want {
		stamp := TableTimeCT(time.UnixMilli(k.OpenTime).In(CTLocation()))
		if !strings.Contains(sec, stamp) {
			t.Errorf("daily row %q absent — the rendered rows are not DailySessionBars' rows", stamp)
		}
	}
	// The partial row must say WHY its Open is not the session open.
	if !strings.Contains(rows, "first bar HELD") {
		t.Errorf("the partial row does not disclose that its Open is the first bar HELD, not the session open:\n%s", sec)
	}
}

// PIN D1-C — THE PROMPT DISCLOSES ITS TAPE DEPTH. It disclosed it nowhere.
func TestCandleBlockDisclosesItsTape(t *testing.T) {
	bars, now := d1Fixture(t)
	table := BuildPlannerCandleTablesAt(bars, 12000, now)
	first := strings.SplitN(table, "\n", 2)[0]

	for _, want := range []string{"TAPE:", "1m bars HELD of 12000 requested", "oldest", "gaps"} {
		if !strings.Contains(first, want) {
			t.Errorf("tape line missing %q — got %q", want, first)
		}
	}
	if !strings.Contains(table, "NEVER interpolated") {
		t.Errorf("the marker legend does not state that a missing bar is never interpolated")
	}
}

// PIN D1-D — A COMPLETE, CLOSED SESSION IS NOT MARKED. A marker on everything
// is a marker on nothing.
func TestCompleteSessionRowCarriesNoMarker(t *testing.T) {
	// Two WHOLE session-days, ending exactly at a session roll so nothing forms.
	from := time.Date(2026, 9, 7, 17, 0, 0, 0, CTLocation())
	now := time.Date(2026, 9, 9, 17, 0, 0, 0, CTLocation())
	bars := d1Tape(from, now)
	table := BuildPlannerCandleTablesAt(bars, 12000, now)
	sec := d1Section(t, table, "### daily session candles")
	if r := d1Rows(sec); strings.Contains(r, "PARTIAL") || strings.Contains(r, "FORMING") || strings.Contains(r, "UNKNOWN") {
		t.Errorf("a complete closed session ROW was marked:\n%s", r)
	}
	if !strings.Contains(sec, "every row PRINTED here is COMPLETE") {
		t.Errorf("a complete table does not SAY it is complete:\n%s", sec)
	}
}

// PIN D1-E — AN UNCLASSIFIABLE DAY IS UNKNOWN, NEVER GUESSED (calendar
// discipline, inherited from kernel/session_calendar.json).
func TestUncoveredCalendarYearRendersUnknownNotZero(t *testing.T) {
	yr := 2031
	if SessionCalendarCoversYear(yr) {
		t.Skipf("calendar now covers %d — pick an uncovered year", yr)
	}
	from := time.Date(yr, 9, 8, 2, 39, 0, 0, CTLocation())
	now := time.Date(yr, 9, 9, 13, 18, 0, 0, CTLocation())
	bars := d1Tape(from, now)
	if len(bars) == 0 {
		t.Skip("no open minutes generated for the uncovered year")
	}
	sec := d1Section(t, BuildPlannerCandleTablesAt(bars, 12000, now), "### daily session candles")
	rows := d1Rows(sec)
	want := fmt.Sprintf("❓COVERAGE-UNKNOWN — %d is not covered by kernel/session_calendar.json", yr)
	if !strings.Contains(rows, want) {
		t.Errorf("a session-day in a year the calendar does not cover must render %q on the ROW, never a computed-looking number:\n%s", want, rows)
	}
	// A24 — and it must NOT also print a plausible count beside the UNKNOWN.
	if strings.Contains(rows, "open 1m intervals in this window") {
		t.Errorf("an UNKNOWN row also printed a computed interval count:\n%s", rows)
	}
}

// PIN D1-F — THE HEADING AND THE ROWS MUST AGREE.
//
// Found by MUTATION, not by design: forcing RowCoverage.Marked() to false left
// every pin above green while the daily heading said "every row PRINTED here is COMPLETE"
// over a row that carried "⚠PARTIAL". A heading that contradicts its own rows
// is the exact defect this wave exists to remove, one layer up.
func TestHeadingCompletenessAgreesWithTheRows(t *testing.T) {
	bars, now := d1Fixture(t)
	table := BuildPlannerCandleTablesAt(bars, 12000, now)

	for _, h := range []string{"### 15m", "### 1h", "### 4h", "### daily session candles"} {
		sec := d1Section(t, table, h)
		head := strings.SplitN(sec, "\n", 2)[0]
		rows := d1Rows(sec)
		markedRows := 0
		for _, ln := range strings.Split(rows, "\n") {
			if strings.Contains(ln, "⚠PARTIAL") || strings.Contains(ln, "⏳FORMING") || strings.Contains(ln, "❓COVERAGE-UNKNOWN") || strings.Contains(ln, "ℹ️HELD") {
				markedRows++
			}
		}
		claimsComplete := strings.Contains(head, "every row PRINTED here is COMPLETE")
		if claimsComplete && markedRows > 0 {
			t.Errorf("%s heading claims \"every row PRINTED here is COMPLETE\" over %d MARKED row(s):\n%s\n%s", h, markedRows, head, rows)
		}
		if !claimsComplete && markedRows == 0 {
			t.Errorf("%s heading withholds the COMPLETE claim although no row is marked:\n%s\n%s", h, head, rows)
		}
		if markedRows > 0 && !strings.Contains(head, fmt.Sprintf("%d of ", markedRows)) {
			t.Errorf("%s heading does not count its %d marked rows:\n%s", h, markedRows, head)
		}
	}
}

// PIN D1-G — A HOLE INSIDE A WINDOW IS NOT A TRUNCATED FRONT.
//
// Found in the LIVE render, not by design. Against the store's real
// 2026-09-09 01:28→13:05 CT hole the 15m row read:
//
//	⚠PARTIAL — holds 14 of the 15 open 1m intervals in this window; its Open is
//	the first bar HELD (01:15 CT), not the window open (01:15 CT)
//
// The two clock times are IDENTICAL: that row's Open is exactly the window
// open, and the missing minute is inside it. A disclosure that states a false
// reason is the same class of defect as a heading that states a false count.
func TestInteriorHoleIsNotReportedAsATruncatedOpen(t *testing.T) {
	// One whole 15m bucket, minus a single minute in its middle. The bucket
	// starts exactly on the tape's first bar, so nothing is truncated.
	start := time.Date(2026, 9, 8, 20, 0, 0, 0, CTLocation())
	var bars []market.Kline
	for i := 0; i < 15; i++ {
		if i == 7 {
			continue // the hole
		}
		ts := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: ts.UnixMilli(), Open: 29500, High: 29502, Low: 29498, Close: 29501, Volume: 10})
	}
	now := start.Add(15 * time.Minute)
	sec := d1Section(t, BuildPlannerCandleTablesAt(bars, 12000, now), "### 15m")
	rows := d1Rows(sec)
	if !strings.Contains(rows, "⚠PARTIAL") {
		t.Fatalf("a 14-of-15 window is not marked partial:\n%s", rows)
	}
	if strings.Contains(rows, "its Open is the first bar HELD") {
		t.Errorf("an INTERIOR hole was reported as a truncated front — the row's Open IS the window open:\n%s", rows)
	}
	if !strings.Contains(rows, "missing INSIDE") {
		t.Errorf("an interior hole must say the missing intervals are INSIDE the window:\n%s", rows)
	}
}

// PIN D1-H — THE TAPE AGE IS READABLE, NOT SUB-SECOND NOISE.
// The live render printed "320h36m50.387s ago"; a millisecond-precise age in a
// prompt invites a model to reason about precision that means nothing here.
func TestTapeAgeIsMinuteResolution(t *testing.T) {
	bars, _ := d1Fixture(t)
	// The clock MUST carry a sub-second part, or this pin cannot fail: the
	// first version used d1Fixture's whole-second clock and a mutation that
	// restored msDurTxt(raw) passed green (class 89).
	now := time.Date(2026, 9, 9, 13, 18, 13, 387_000_000, CTLocation())
	first := strings.SplitN(BuildPlannerCandleTablesAt(bars, 12000, now), "\n", 2)[0]
	if strings.Contains(first, ".") && strings.Contains(first, "s ago") {
		i := strings.Index(first, "s ago")
		if j := strings.LastIndex(first[:i], "."); j >= 0 && i-j <= 5 {
			t.Errorf("tape age carries sub-second noise: %q", first)
		}
	}
	if !strings.Contains(first, "m ago") && !strings.Contains(first, "h") {
		t.Errorf("tape age does not render as a duration: %q", first)
	}
}

// ── REVIEW PINS, 2026-09-09 (added after three reviewers; each one names the
// defect it would have caught) ──────────────────────────────────────────────

// PIN D1-I — A HEALTHY TAPE CARRIES NO ⚠PARTIAL, AND THE ANSWER DOES NOT
// DEPEND ON WHETHER THE FORMING BAR HAS BEEN DELIVERED YET.
//
// THE DEFECT THIS CATCHES: observableEndMs rounded UP to the end of the minute
// `now` falls in, counting the minute still IN PROGRESS as an interval the tape
// ought to hold. Bars are delivered on close, so on a PERFECTLY GAPLESS tape
// the newest row of all four tables rendered "⏳FORMING ⚠PARTIAL — holds only
// 1309 of the 1310" while the TAPE line six lines above it said "gaps 0" — a
// self-contradiction, in 100% of live reads, under a prompt that tells the
// model to trust the candles. A marker on every read is a marker on nothing.
func TestHealthyTapeIsNotMarkedPartial(t *testing.T) {
	from := time.Date(2026, 9, 8, 17, 0, 0, 0, CTLocation())
	// Newest DELIVERED bar opens 14:48 and closed at 14:49; 14:49 is forming.
	bars := d1Tape(from, time.Date(2026, 9, 9, 14, 49, 0, 0, CTLocation()))
	now := time.Date(2026, 9, 9, 14, 49, 30, 0, CTLocation())
	table := BuildPlannerCandleTablesAt(bars, 12000, now)
	if !strings.Contains(table, "gaps 0 open 1m intervals") {
		t.Fatalf("fixture is not gapless — the pin would prove nothing:\n%s", d1Section(t, table, "TAPE:"))
	}
	for _, h := range []string{"### 15m", "### 1h", "### 4h", "### daily session candles"} {
		sec := d1Section(t, table, h)
		rows := strings.Split(d1Rows(sec), "\n")
		newest := rows[len(rows)-1]
		if strings.Contains(newest, "⚠PARTIAL") {
			t.Errorf("%s: the NEWEST row of a GAPLESS tape is marked ⚠PARTIAL while the tape line says gaps 0:\n%s", h, sec)
		}
		if !strings.Contains(newest, "⏳FORMING") {
			t.Errorf("%s: the newest row of an open window must still say FORMING:\n%s", h, sec)
		}
		// Nothing INTERIOR may be partial either. Only the OLDEST row may be,
		// and only because the tape BEGINS inside its window (front truncation
		// by where the tape starts, which is a true statement about the tape).
		for i := 1; i < len(rows); i++ {
			if strings.Contains(rows[i], "⚠PARTIAL") {
				t.Errorf("%s: an interior row of a GAPLESS tape is marked ⚠PARTIAL:\n%s", h, rows[i])
			}
		}
	}
	// And the SAME clock with the forming bar already delivered must agree.
	bars2 := d1Tape(from, time.Date(2026, 9, 9, 14, 50, 0, 0, CTLocation()))
	t2 := BuildPlannerCandleTablesAt(bars2, 12000, now)
	for _, h := range []string{"### 15m", "### 1h", "### 4h", "### daily session candles"} {
		a := strings.SplitN(d1Section(t, table, h), "\n", 2)[0]
		b := strings.SplitN(d1Section(t, t2, h), "\n", 2)[0]
		if a != b {
			t.Errorf("%s heading depends on whether the FORMING bar has landed yet:\n  not delivered: %s\n  delivered:     %s", h, a, b)
		}
	}
}

// PIN D1-J — WHOLE ROWS THAT ARE NOT THERE ARE COUNTED AND NAMED.
//
// THE DEFECT THIS CATCHES: RowCoverage measures INSIDE a rendered row and the
// heading counts ROWS, so a bucket holding no bars produced no row and was
// invisible to both. A tape with a market-open hole rendered
// "### 15m — HELD 12 of 12 requested rows · all held rows COMPLETE" with FOUR
// whole 15m windows missing between two adjacent-LOOKING rows. That is this
// wave's own thesis — a COUNT cannot express a SPAN or a HOLE — one layer down,
// inside the fix.
func TestWholeAbsentRowsAreNamedNotSilentlySkipped(t *testing.T) {
	a := d1Tape(time.Date(2026, 9, 8, 18, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 19, 0, 0, 0, CTLocation()))
	b := d1Tape(time.Date(2026, 9, 8, 20, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 22, 0, 0, 0, CTLocation()))
	now := time.Date(2026, 9, 8, 22, 0, 0, 0, CTLocation())
	sec := d1Section(t, BuildPlannerCandleTablesAt(append(a, b...), 12000, now), "### 15m")
	head := strings.SplitN(sec, "\n", 2)[0]
	if !strings.Contains(head, "4 WHOLE ROWS ABSENT BETWEEN HELD ROWS") {
		t.Fatalf("the heading does not count the rows that are NOT there (4×15m, market open):\n%s", head)
	}
	if strings.Contains(head, "every row PRINTED here is COMPLETE") && !strings.Contains(head, "ABSENT BETWEEN HELD ROWS") {
		t.Fatalf("a table straddling a hole must not read as complete:\n%s", head)
	}
	rows := d1Rows(sec)
	if !strings.Contains(rows, "⛔4×15m ABSENT BEFORE THIS ROW") {
		t.Fatalf("the row FOLLOWING the hole carries no ⛔ marker:\n%s", rows)
	}
	if !strings.Contains(rows, "no bars between 18:45 CT and 20:00 CT") {
		t.Fatalf("the ⛔ marker must name the CT bounds of the break:\n%s", rows)
	}
	// A24: nothing was invented to fill it.
	for _, ln := range strings.Split(rows, "\n") {
		if strings.HasPrefix(ln, "09-08 19:") {
			t.Fatalf("a bar inside the hole was SYNTHESISED: %s", ln)
		}
	}
}

// PIN D1-K — A CLOSED INSTANT IS NEVER NAMED AS "THE WINDOW OPEN".
//
// THE DEFECT THIS CATCHES: WindowOpenMs was the raw epoch-floor bucket start.
// The Sunday 4h bucket floors to 15:00 CT, two hours before CME reopens, so a
// front-truncated row printed "its Open is the first bar HELD (17:30 CT), not
// the window open (15:00 CT)" — counts right, reason false. Same class as the
// interior-hole false reason this file already fixed, one bucket over.
func TestWindowOpenIsAnOpenInstant(t *testing.T) {
	sundayFloor := time.Date(2026, 9, 6, 15, 0, 0, 0, CTLocation())
	if IsCMEOpen(sundayFloor) {
		t.Skipf("%s is open — pick a bucket floor that lands in the halt", sundayFloor)
	}
	got := firstOpenGridMs(sundayFloor.UnixMilli(), sundayFloor.Add(4*time.Hour).UnixMilli())
	if !IsCMEOpen(time.UnixMilli(got)) {
		t.Fatalf("firstOpenGridMs returned %s CT, which is CLOSED", ClockHHMMCT(time.UnixMilli(got)))
	}
	if want := time.Date(2026, 9, 6, 17, 0, 0, 0, CTLocation()); got != want.UnixMilli() {
		t.Fatalf("firstOpenGridMs = %s CT, want %s CT (the reopen)", ClockHHMMCT(time.UnixMilli(got)), ClockHHMMCT(want))
	}
	// And a bucket that is open at its floor is unchanged.
	openFloor := time.Date(2026, 9, 8, 11, 0, 0, 0, CTLocation())
	if !IsCMEOpen(openFloor) {
		t.Skip("fixture floor is closed")
	}
	if got := firstOpenGridMs(openFloor.UnixMilli(), openFloor.Add(4*time.Hour).UnixMilli()); got != openFloor.UnixMilli() {
		t.Fatalf("an OPEN bucket floor was moved to %s CT", ClockHHMMCT(time.UnixMilli(got)))
	}
}

// PIN D1-L — THE TAPE LINE'S TWO NUMBERS ARE PINNED BY VALUE, NOT BY SHAPE.
//
// THE DEFECT THIS CATCHES: D1-C asserted only the substrings "TAPE:",
// "1m bars HELD of 12000 requested" and "gaps". A reviewer hardcoded the gap
// count to "0" and then replaced the SERVED count with the REQUESTED count —
// the exact defect this wave exists to remove, moved one layer up — and the
// whole package stayed green. The headline sentence had no pin behind it.
func TestTapeLineNumbersArePinnedByValue(t *testing.T) {
	// A tape with a KNOWN hole: 09-08 18:00→19:00 and 20:00→22:00 CT, market
	// open throughout, so exactly 60 open 1m intervals are missing inside it.
	a := d1Tape(time.Date(2026, 9, 8, 18, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 19, 0, 0, 0, CTLocation()))
	b := d1Tape(time.Date(2026, 9, 8, 20, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 22, 0, 0, 0, CTLocation()))
	bars := append(a, b...)
	if len(bars) != 180 {
		t.Fatalf("fixture holds %d bars, want 180", len(bars))
	}
	line := strings.SplitN(BuildPlannerCandleTablesAt(bars, 12000, time.Date(2026, 9, 8, 22, 0, 0, 0, CTLocation())), "\n", 2)[0]
	if !strings.Contains(line, "TAPE: 180 1m bars HELD of 12000 requested") {
		t.Fatalf("the TAPE line must state the SERVED count, by value:\n%s", line)
	}
	if !strings.Contains(line, "gaps 60 open 1m intervals missing INSIDE that span") {
		t.Fatalf("the TAPE line must state the COMPUTED gap count, by value:\n%s", line)
	}
	// A contiguous twin: the 0 must be a real computed 0, and it must render.
	cont := d1Tape(time.Date(2026, 9, 8, 18, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 21, 0, 0, 0, CTLocation()))
	line2 := strings.SplitN(BuildPlannerCandleTablesAt(cont, 12000, time.Date(2026, 9, 8, 21, 0, 0, 0, CTLocation())), "\n", 2)[0]
	if !strings.Contains(line2, "TAPE: 180 1m bars HELD of 12000 requested") || !strings.Contains(line2, "gaps 0 open 1m intervals") {
		t.Fatalf("a contiguous tape must report served=180 gaps=0:\n%s", line2)
	}
}

// PIN D1-M — AN UNMEASURABLE BREAK IS UNKNOWN, NEVER ZERO (A24).
func TestAbsentRowCountIsUnknownWhenTheScanCaps(t *testing.T) {
	old := absentRowScanCap
	absentRowScanCap = 2
	defer func() { absentRowScanCap = old }()
	a := d1Tape(time.Date(2026, 9, 8, 18, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 19, 0, 0, 0, CTLocation()))
	b := d1Tape(time.Date(2026, 9, 8, 20, 0, 0, 0, CTLocation()), time.Date(2026, 9, 8, 22, 0, 0, 0, CTLocation()))
	sec := d1Section(t, BuildPlannerCandleTablesAt(append(a, b...), 12000, time.Date(2026, 9, 8, 22, 0, 0, 0, CTLocation())), "### 15m")
	if !strings.Contains(sec, "absent-row count UNKNOWN, never read as zero") {
		t.Fatalf("a capped between-row walk must report UNKNOWN in the heading:\n%s", strings.SplitN(sec, "\n", 2)[0])
	}
	if !strings.Contains(d1Rows(sec), "⛔ABSENT-BEFORE UNKNOWN") {
		t.Fatalf("a capped between-row walk must report UNKNOWN on the row:\n%s", d1Rows(sec))
	}
	if strings.Contains(sec, "0 WHOLE ROWS ABSENT") {
		t.Fatalf("an uncomputed count was rendered as 0:\n%s", sec)
	}
}

// PIN D1-N — A29. The flagship renderer HAS a production call site.
//
// THE DEFECT THIS CATCHES: a reviewer replaced the whole call with
// `candleTables = ""` — deleting the entire D1 disclosure from every planner
// prompt — and the module stayed green. Four A29 source-scans shipped in this
// wave and none of them named this function.
func TestPlannerCandleTablesAreWiredIntoThePrompt(t *testing.T) {
	const src = "../trader/auto_trader_planner.go"
	b, err := os.ReadFile(src)
	if err != nil {
		// NOT a skip: a pin that cannot fail is not a pin (A8).
		t.Fatalf("cannot read %s to prove the call site exists: %v", src, err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "//") {
			continue
		}
		if strings.Contains(t, "kernel.BuildPlannerCandleTablesAt(") {
			return
		}
	}
	t.Fatalf("kernel.BuildPlannerCandleTablesAt has 0 production call sites in %s (A29) — the whole D1 disclosure can be deleted from every planner prompt with the suite green", src)
}
