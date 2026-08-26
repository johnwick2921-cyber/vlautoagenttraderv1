package trader

import (
	"strings"
	"testing"

	"nofx/calendar"
	"nofx/kernel"
)

// W3 — red-news end-to-end (no network): a fixture FF feed with a High (T1) USD
// event → grouped by CT date → session-sliced → HARD blackout window → the
// executor gate blocks entries inside it, and the plan carries the no-trade line.
// The event is at 09:00 CT (outside first-5m + lunch) to isolate the red-news gate.
func TestW3RedNewsEndToEnd(t *testing.T) {
	// 09:00 CT (CDT, UTC-5) = 14:00 UTC, Monday 2026-08-17.
	ffJSON := `[{"title":"FOMC Rate Decision","country":"USD","date":"2026-08-17T14:00:00Z","impact":"High"},
	            {"title":"Fed Speak","country":"USD","date":"2026-08-17T15:30:00Z","impact":"Medium"}]`
	res := calendar.FetchWeek(func() ([]byte, error) { return []byte(ffJSON), nil }, nil)
	day, ok := res.Days["2026-08-17"]
	if !ok || len(day) == 0 {
		t.Fatalf("fixture events not grouped by CT date: %+v", res.Days)
	}

	pe := sessionPlannerEvents(day, "NY")
	var t1, t2 int
	var t1Time string
	for _, e := range pe {
		if e.Impact == "T1" {
			t1++
			t1Time = e.TimeCT
		}
		if e.Impact == "T2" {
			t2++
		}
	}
	if t1 != 1 || t2 != 1 || t1Time != "09:00" {
		t.Fatalf("session events wrong: t1=%d t2=%d t1Time=%q (%+v)", t1, t2, t1Time, pe)
	}

	windows := kernel.T1BlackoutWindows(pe)
	if len(windows) != 1 {
		t.Fatalf("only the T1 event makes a HARD window (T2 must not), got %d", len(windows))
	}
	reg := kernel.DefaultSessionRegistry()

	// HARD block inside the FOMC ±15m window (08:45–09:15).
	if reason, blocked := sessionGateDecision(reg, ctAt(t, 9, 0), windows, nil); !blocked || !strings.Contains(reason, "red-news") {
		t.Fatalf("entry must be red-news-blocked at 09:00, got blocked=%v reason=%q", blocked, reason)
	}
	// clear at 10:00 (NY session, no red news, not lunch/first-5m).
	if reason, blocked := sessionGateDecision(reg, ctAt(t, 10, 0), windows, nil); blocked {
		t.Fatalf("entry must be clear at 10:00, got blocked: %s", reason)
	}

	// the plan carries the HARD no-trade blackout line (§80).
	lines := kernel.T1NoTradeLines(pe)
	if len(lines) != 1 || !strings.Contains(lines[0], "FOMC") || !strings.Contains(lines[0], "HARD no-trade") {
		t.Fatalf("plan no-trade line missing/wrong: %v", lines)
	}

	// the assembled planner prompt SHOWS the calendar (input.Calendar → Calendar section).
	prompt := kernel.BuildPlannerPrompt(kernel.PlannerInput{
		TradeDate: "2026-08-17", Session: "NY", Calendar: pe,
	})
	if !strings.Contains(prompt, "FOMC") || !strings.Contains(prompt, "Calendar") {
		t.Fatalf("planner prompt must show the red-news event:\n%s", prompt)
	}
}
