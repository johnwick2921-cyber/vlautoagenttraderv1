package kernel

import (
	"fmt"
	"strings"

	"nofx/store"
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

// T1Split is the ONE classification of a session's T1 (red) events under a
// currency set (W-T1-CURRENCIES, 2026-09-18). Hard windows gate; advisory
// lines are shown (plan no_trade, card, prompt) and gate NOTHING; Uncurrencied
// names the T1 events that carried no currency and were therefore treated as
// HARD (fail closed) — the caller logs them once per day.
type T1Split struct {
	Hard         []CTWindow
	Advisory     []string
	Uncurrencied []string
}

// t1CurrencyHard decides whether one event currency hard-blocks under the set.
// Case-insensitive. "ALL"/"*" anywhere in the set → everything hard-blocks
// (the pre-wave behaviour). An empty/blank event currency → HARD + unknown
// (fail closed: a red event we cannot place must not be waved through).
func t1CurrencyHard(ccy string, set []string) (hard bool, unknown bool) {
	ccy = strings.TrimSpace(ccy)
	if ccy == "" {
		return true, true
	}
	for _, s := range set {
		s = strings.TrimSpace(s)
		if s == "*" || strings.EqualFold(s, store.T1CurrencyAll) || strings.EqualFold(s, ccy) {
			return true, false
		}
	}
	return false, false
}

// T1CurrencySetLabel renders the resolved set for log/advisory text: "USD",
// "USD,EUR" or "ALL". Never empty: a nil/empty set is the shipped default.
func T1CurrencySetLabel(set []string) string {
	if len(set) == 0 {
		set = store.DefaultT1Currencies()
	}
	for _, s := range set {
		s = strings.TrimSpace(s)
		if s == "*" || strings.EqualFold(s, store.T1CurrencyAll) {
			return store.T1CurrencyAll
		}
	}
	return strings.Join(set, ",")
}

// SplitT1 classifies the T1 events of a session's calendar slice under the
// currency set: events whose currency is in the set (or any event when the set
// is ALL, or any event WITHOUT a currency) open a HARD ±T1BlackoutMinutes
// window; every other T1 event becomes one advisory line. T2 and non-timed
// events are ignored (T2 is caution-only, not a hard block). A nil/empty set
// resolves to the shipped default so no caller can accidentally gate nothing.
func SplitT1(events []PlannerCalendarEvent, currencies []string) T1Split {
	if len(currencies) == 0 {
		currencies = store.DefaultT1Currencies()
	}
	var out T1Split
	setLabel := T1CurrencySetLabel(currencies)
	for _, e := range events {
		if strings.ToUpper(strings.TrimSpace(e.Impact)) != "T1" {
			continue
		}
		m, ok := hhmmToMinK(e.TimeCT)
		if !ok {
			continue
		}
		hard, unknown := t1CurrencyHard(e.Currency, currencies)
		if unknown {
			out.Uncurrencied = append(out.Uncurrencied, strings.TrimSpace(e.Title))
		}
		if !hard {
			out.Advisory = append(out.Advisory, fmt.Sprintf("🟠 %s %s CT (%s) — red news, advisory only (t1_currencies=%s)",
				strings.TrimSpace(e.Title), e.TimeCT, strings.ToUpper(strings.TrimSpace(e.Currency)), setLabel))
			continue
		}
		out.Hard = append(out.Hard, CTWindow{
			Start: ((m-T1BlackoutMinutes)%1440 + 1440) % 1440,
			End:   ((m+T1BlackoutMinutes)%1440 + 1440) % 1440,
			Label: fmt.Sprintf("%s %s CT ±%dm", strings.TrimSpace(e.Title), e.TimeCT, T1BlackoutMinutes),
		})
	}
	return out
}

// T1BlackoutWindows returns the HARD blackout windows for the T1 events in a
// session's calendar slice under the currency set (±T1BlackoutMinutes around
// each). The ONE gate-window source: the arm gate, the plan write, the fade
// facts and the card all read it (class 52). Advisory events never appear here.
func T1BlackoutWindows(events []PlannerCalendarEvent, currencies []string) []CTWindow {
	return SplitT1(events, currencies).Hard
}

// T1AdvisoryLines returns the advisory lines for the T1 events NOT in the
// currency set. They render; they never gate.
func T1AdvisoryLines(events []PlannerCalendarEvent, currencies []string) []string {
	return SplitT1(events, currencies).Advisory
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

// T1NoTradeLines renders the no-trade lines written into a plan's no_trade
// list (§80 — auto-written, not left to the model): the HARD blackout lines
// for the events in the currency set, then the advisory lines for the rest.
func T1NoTradeLines(events []PlannerCalendarEvent, currencies []string) []string {
	sp := SplitT1(events, currencies)
	var out []string
	for _, w := range sp.Hard {
		out = append(out, "🔴 "+w.Label+" — HARD no-trade (red news)")
	}
	return append(out, sp.Advisory...)
}

// T1NoTradeLinesDrift is T1NoTradeLines with each HARD window widened by the
// measured clock drift (F6, 2026-08-30): a skewed clock shifts when an event
// actually fires relative to the local clock, so the blackout must cover the
// uncertainty. Advisory lines carry no window and are not widened.
func T1NoTradeLinesDrift(events []PlannerCalendarEvent, currencies []string, driftMs int64) []string {
	sp := SplitT1(events, currencies)
	var out []string
	for _, w := range WidenCTWindows(sp.Hard, driftMs) {
		out = append(out, "🔴 "+w.Label+" — HARD no-trade (red news)")
	}
	return append(out, sp.Advisory...)
}

// WidenCTWindows shifts every window's Start earlier and End later by
// ClockWidenMinutes(driftMs) — ceil(|driftMs|/60s) minutes (min 1), HARD-CAPPED
// at ClockWidenCapMinutes — so blackout protection survives a skewed clock. A
// zero drift returns the windows unchanged.
//
// The widening itself is unchanged for ANY nonzero measurement: even a 108 ms
// offset can carry an event across a minute boundary, so the one-minute guard
// is real protection. What changed (2026-09-02) is the CLAIM. The label said
// "+1m (clock drift)" whenever the offset was nonzero, so a perfectly healthy
// clock produced a card that told the reader the machine's time was drifting.
// Only a skew large enough to move an event by a whole minute on its own is
// stated; below that the extra minute is boundary rounding and goes unlabelled.
//
// W-DRIFT-WIDEN-CAP (2026-09-17, CLASS 145): the widening is CAPPED. The
// measurement is local clock minus the freshest 1m bar's close, and a POSITIVE
// value beyond the clock-plausible range is the AGE OF THE FEED, not clock
// skew — exactly what a CME halt (16:00–17:00 CT) or a feed gap looks like.
// The 2026-09-17 ASIA read authored at 16:38 CT inside the halt measured
// 2,326 s against the 15:59 bar and widened the BOJ ±15m band by 39 minutes a
// side (1h48m blocked). No clock on this host has ever skewed past the 60 s
// tolerance without being deferred (negative) or logged CRITICAL, so the
// widening a clock can honestly demand is ceil(tolerance/60s) = 1 min plus one
// boundary minute = 2. Anything beyond that is staleness and the caller says
// so via ClockDriftStaleNote instead of widening.
func WidenCTWindows(windows []CTWindow, driftMs int64) []CTWindow {
	m := ClockWidenMinutes(driftMs)
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

// ClockWidenCapMinutes is the HARD cap on news-window widening from a clock
// measurement: ceil(clockDriftToleranceMs/60s) = 1 minute of tolerable skew
// plus at most one boundary-rounding minute. A measurement that asks for more
// is not a clock (CLASS 145).
const ClockWidenCapMinutes = 2

// clockPlausibleSkewMs bounds what a measured offset may be CALLED. Above it
// a positive offset is feed age (halt, gap) and a negative one is a clock so
// broken that authoring is already deferred; neither is "(clock drift)" on a
// card.
const clockPlausibleSkewMs = 5 * 60_000

// ClockWidenMinutes is the capped widening in whole minutes: ceil(|driftMs|/60s)
// (min 1 for any nonzero measurement), never more than ClockWidenCapMinutes.
func ClockWidenMinutes(driftMs int64) int {
	m := driftWidenMinutes(driftMs)
	if m > ClockWidenCapMinutes {
		m = ClockWidenCapMinutes
	}
	return m
}

// ClockDriftStaleNote names the real cause when a measurement is outside the
// clock-plausible range, for the caller's WARN line. "" when the measurement
// may honestly be called clock skew (|drift| ≤ 5 min).
func ClockDriftStaleNote(driftMs int64) string {
	abs := absI64(driftMs)
	if abs <= clockPlausibleSkewMs {
		return ""
	}
	mins := abs / 60_000
	if driftMs > 0 {
		return fmt.Sprintf("feed stale %dm — halt or gap, not clock skew; news windows NOT widened beyond the %dm cap", mins, ClockWidenCapMinutes)
	}
	return fmt.Sprintf("feed labels bars %dm in the FUTURE — a broken local clock beyond the plausible skew range (authoring is deferred separately); news windows widened only by the %dm cap", mins, ClockWidenCapMinutes)
}

// driftIsSkew reports whether a measured clock offset is large enough to move
// an event by a whole minute by itself AND small enough to plausibly be a clock
// — the only case where a card may tell the reader the widening is caused by
// clock drift. Beyond clockPlausibleSkewMs the measurement is feed age, and
// the card says nothing about the clock (ClockDriftStaleNote carries the
// cause to the journal instead).
func driftIsSkew(driftMs int64) bool {
	abs := absI64(driftMs)
	return abs >= 60_000 && abs <= clockPlausibleSkewMs
}

// driftWidenMinutes rounds |driftMs| up to whole minutes (min 1 for any
// positive drift). Uncapped; ClockWidenMinutes applies the cap.
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
