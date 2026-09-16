package trader

import (
	"fmt"
	"sort"
	"time"

	"nofx/kernel"
	"nofx/logger"
)

// P4 — HalfDays, FOLDED INTO THE SESSION CALENDAR (owner ruling 2026-09-07).
//
// This file used to LOAD half_days.json and seed a HalfDays map into the session
// registry — a second dated calendar with its own key convention that, on three
// dates, disagreed with kernel/session_calendar.json: the gate stopped at 12:00
// on days this file's SOURCED rows said 12:15. One fact, one owner. The rows and
// their archived CME citations were folded into the calendar verbatim,
// half_days.json was deleted, and what remains here DERIVES from the calendar so
// the boot line and any remaining caller cannot drift from the gate.
//
// The consumer chain existed for months with an EMPTY map: the session registry
// (system_config key "session_registry") carries HalfDays{date → early-close
// CT}, consumed by the EOD-flat + last-entry pull-ins in auto_trader_clock.go —
// but nothing ever populated it, so Labor Day Sep 7 (and every other early
// close) was unprotected. This producer seeds it from an owner-editable JSON
// file (half_days.json at the repo root, path env NOFX_HALF_DAYS) — the
// calendarStaticLoader pattern.
//
// OFFICIAL 2026 SOURCES (fetched as archived CME originals, ledger-close recon):
//   Labor Day 2026-09-07 — equities halt 12:00 CT, reopen 17:00 CT:
//     https://www.cmegroup.com/tools-information/holiday-calendar/files/2026/labor-day-holiday-settlement-times-2026.pdf
//     + archived 2026 trading-hours page (web.archive.org/web/20260103182546/
//     https://www.cmegroup.com/trading-hours.html)
//   Thanksgiving Thu 2026-11-26 halt 12:00 CT · Fri 2026-11-27 equity final
//     close 12:15 CT (settlement 12:00):
//     https://www.cmegroup.com/tools-information/holiday-calendar/files/2026/thanksgiving-holiday-settlement-times-2026.pdf
//   Christmas Eve 2026-12-24 — equity final close 12:15 CT (settlement 12:00):
//     https://www.cmegroup.com/tools-information/holiday-calendar/files/2026/christmas-holiday-settlement-times-2026.pdf
//   NYE 2026-12-31 — NORMAL equity session (only rates settle early) → NOT a
//     half-day; deliberately absent from the seed:
//     https://www.cmegroup.com/tools-information/holiday-calendar/files/2026/new-years-eve-holiday-settlement-times-2027.pdf
//
// KNOWN INTERACTION (reported, not changed here): isCMEHoliday treats Labor
// Day, Thanksgiving, day-after-Thanksgiving and Dec 24/31 as FULL closures, so
// the decision cycle idles on those calendar dates regardless — the half-day
// entries are the TRUTH layer and protect any date the bot does trade (and
// become live protection the day isCMEHoliday is refined).
//
// Fail-open (4.5): a malformed/missing file logs CRITICAL and trading proceeds
// normally; a malformed ENTRY is skipped with CRITICAL, valid ones still land.

// HalfDayEntry is one early-close day, DERIVED from the session calendar.
type HalfDayEntry struct {
	Date         string `json:"date"`           // YYYY-MM-DD (CME session-day key)
	EarlyCloseCT string `json:"early_close_ct"` // "HH:MM" CT
	Label        string `json:"label"`
}

// LoadHalfDaysFile now reads the SESSION CALENDAR rather than half_days.json.
// The name and signature are kept so callers and their pins survive the fold;
// the source of truth moved, the API did not.
func LoadHalfDaysFile() ([]HalfDayEntry, error) {
	out := kernel.SessionShortenedDays()
	entries := make([]HalfDayEntry, 0, len(out))
	for _, d := range out {
		entries = append(entries, HalfDayEntry{Date: d.Date, EarlyCloseCT: d.CloseCT, Label: d.Name})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date < entries[j].Date })
	return entries, nil
}

// NextUpcomingHalfDay is the next early close at or after now, in CT.
func NextUpcomingHalfDay(entries []HalfDayEntry, now time.Time) (HalfDayEntry, bool) {
	key := now.In(kernel.CTLocation()).Format("2006-01-02")
	for _, e := range entries {
		if e.Date >= key {
			return e, true
		}
	}
	return HalfDayEntry{}, false
}

// LogHalfDaysBoot states the early-close days the calendar holds. Every field is
// READ from the calendar the gate consults (A11), so this line and the gate can
// never disagree.
func LogHalfDaysBoot(now time.Time) {
	entries, _ := LoadHalfDaysFile()
	next := "none upcoming"
	if e, ok := NextUpcomingHalfDay(entries, now); ok {
		next = fmt.Sprintf("%s %s CT (%s)", e.Date, e.EarlyCloseCT, e.Label)
	}
	logger.Infof("🗓 half-days: %d early close(s) in the session calendar · next: %s · source=kernel/session_calendar.json (folded 2026-09-07; half_days.json deleted)",
		len(entries), next)
}

// maybeSeedHalfDays is now a NO-OP kept as a call-site seam. Seeding existed to
// copy half_days.json into the registry; after the fold there is nothing to copy
// — EffectiveFlatCT resolves from the calendar directly. Removing the call site
// would touch the cycle's hot path for no behavioural gain, so the seam stays
// and states why it does nothing.
func (at *AutoTrader) maybeSeedHalfDays(time.Time) {}
