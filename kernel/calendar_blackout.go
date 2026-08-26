package kernel

import (
	"fmt"
	"strings"
)

// W3 — red-news (T1) HARD no-trade blackout windows. A T1 (red / High-impact)
// event opens a ±T1BlackoutMinutes window around its time in which NO entry may
// fire (§80). Pure + minute-of-day based so both the executor gate and the plan
// no-trade lines derive from the same source.

// T1BlackoutMinutes is the half-width (minutes) of the blackout around a red event.
const T1BlackoutMinutes = 15

// CTWindow is a [start,end] minute-of-day CT window (end may be < start ⇒ wraps).
type CTWindow struct {
	Start int
	End   int
	Label string
}

// T1BlackoutWindows returns the blackout windows for the T1 events in a session's
// calendar slice (±T1BlackoutMinutes around each). T2 and non-timed events are
// ignored (T2 is caution-only, not a hard block).
func T1BlackoutWindows(events []PlannerCalendarEvent) []CTWindow {
	var out []CTWindow
	for _, e := range events {
		if strings.ToUpper(strings.TrimSpace(e.Impact)) != "T1" {
			continue
		}
		m, ok := hhmmToMinK(e.TimeCT)
		if !ok {
			continue
		}
		out = append(out, CTWindow{
			Start: ((m-T1BlackoutMinutes)%1440 + 1440) % 1440,
			End:   ((m+T1BlackoutMinutes)%1440 + 1440) % 1440,
			Label: fmt.Sprintf("%s %s CT ±%dm", strings.TrimSpace(e.Title), e.TimeCT, T1BlackoutMinutes),
		})
	}
	return out
}

// InT1Blackout reports whether nowMin (CT minute-of-day) falls inside any window,
// with the matching label. Wrap-aware.
func InT1Blackout(nowMin int, windows []CTWindow) (string, bool) {
	for _, w := range windows {
		in := false
		if w.End >= w.Start {
			in = nowMin >= w.Start && nowMin <= w.End
		} else {
			in = nowMin >= w.Start || nowMin <= w.End
		}
		if in {
			return w.Label, true
		}
	}
	return "", false
}

// T1NoTradeLines renders the HARD no-trade blackout descriptions written into a
// plan's no_trade list (§80 — auto-written, not left to the model).
func T1NoTradeLines(events []PlannerCalendarEvent) []string {
	var out []string
	for _, w := range T1BlackoutWindows(events) {
		out = append(out, "🔴 "+w.Label+" — HARD no-trade (red news)")
	}
	return out
}

// hhmmToMinK parses "HH:MM" → minutes-of-day (kernel-local; the trader has its own).
func hhmmToMinK(s string) (int, bool) {
	var h, m int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d:%d", &h, &m); err != nil {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
