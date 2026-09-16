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

// T1NoTradeLinesDrift is T1NoTradeLines with each window widened by the measured
// clock drift (F6, 2026-08-30): a skewed clock shifts when an event actually
// fires relative to the local clock, so the blackout must cover the uncertainty.
func T1NoTradeLinesDrift(events []PlannerCalendarEvent, driftMs int64) []string {
	var out []string
	for _, w := range WidenCTWindows(T1BlackoutWindows(events), driftMs) {
		out = append(out, "🔴 "+w.Label+" — HARD no-trade (red news)")
	}
	return out
}

// WidenCTWindows shifts every window's Start earlier and End later by
// ceil(|driftMs|/60s) minutes (min 1) so blackout protection survives a skewed
// clock. A zero drift returns the windows unchanged.
//
// The widening itself is unchanged for ANY nonzero measurement: even a 108 ms
// offset can carry an event across a minute boundary, so the one-minute guard
// is real protection. What changed (2026-09-02) is the CLAIM. The label said
// "+1m (clock drift)" whenever the offset was nonzero, so a perfectly healthy
// clock produced a card that told the reader the machine's time was drifting.
// Only a skew large enough to move an event by a whole minute on its own is
// stated; below that the extra minute is boundary rounding and goes unlabelled.
func WidenCTWindows(windows []CTWindow, driftMs int64) []CTWindow {
	m := driftWidenMinutes(driftMs)
	if m <= 0 {
		return windows
	}
	out := make([]CTWindow, 0, len(windows))
	for _, w := range windows {
		label := w.Label
		if driftIsSkew(driftMs) {
			label = fmt.Sprintf("%s +%dm (clock drift)", w.Label, m)
		}
		out = append(out, CTWindow{
			Start: ((w.Start-m)%1440 + 1440) % 1440,
			End:   (w.End + m) % 1440,
			Label: label,
		})
	}
	return out
}

// driftIsSkew reports whether a measured clock offset is large enough to move
// an event by a whole minute by itself — the only case where a card may tell
// the reader the widening is caused by clock drift.
func driftIsSkew(driftMs int64) bool {
	if driftMs < 0 {
		driftMs = -driftMs
	}
	return driftMs >= 60_000
}

// driftWidenMinutes rounds |driftMs| up to whole minutes (min 1 for any
// positive drift).
func driftWidenMinutes(driftMs int64) int {
	if driftMs < 0 {
		driftMs = -driftMs
	}
	if driftMs <= 0 {
		return 0
	}
	return int((driftMs + 59_999) / 60_000)
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
