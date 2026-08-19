package trader

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/store"
)

// P4 — HalfDays PRODUCER (ledger-close dispatch 2026-08-19).
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

// HalfDayEntry is one owner-config row in half_days.json.
type HalfDayEntry struct {
	Date         string `json:"date"`           // YYYY-MM-DD (CME session-day key)
	EarlyCloseCT string `json:"early_close_ct"` // "HH:MM" CT
	Label        string `json:"label"`
}

func halfDaysPath() string {
	if p := os.Getenv("NOFX_HALF_DAYS"); p != "" {
		return p
	}
	return "half_days.json"
}

// LoadHalfDaysFile reads + validates the owner file. Invalid entries are
// dropped (CRITICAL-logged); a missing file returns (nil, nil) — not an error.
func LoadHalfDaysFile() ([]HalfDayEntry, error) {
	raw, err := os.ReadFile(halfDaysPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []HalfDayEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse %s: %w", halfDaysPath(), err)
	}
	valid := entries[:0]
	for _, e := range entries {
		if _, err := time.Parse("2006-01-02", e.Date); err != nil {
			logger.Errorf("🚨 half-days CRITICAL: entry date %q is not YYYY-MM-DD — entry skipped, trading continues normally", e.Date)
			continue
		}
		if _, ok := hhmmToMin(e.EarlyCloseCT); !ok {
			logger.Errorf("🚨 half-days CRITICAL: entry %s early_close_ct %q is not HH:MM — entry skipped, trading continues normally", e.Date, e.EarlyCloseCT)
			continue
		}
		valid = append(valid, e)
	}
	return valid, nil
}

// NextUpcomingHalfDay returns the first entry at/after now's CME session-day.
func NextUpcomingHalfDay(entries []HalfDayEntry, now time.Time) (HalfDayEntry, bool) {
	today := kernel.CMESessionDayKey(now)
	sorted := append([]HalfDayEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })
	for _, e := range sorted {
		if e.Date >= today {
			return e, true
		}
	}
	return HalfDayEntry{}, false
}

// SeedHalfDaysIntoRegistry merges the file entries into the STORED session
// registry (system_config), file-wins per key, never deleting DB-only keys.
// Returns (added-or-updated count, error). Idempotent — a no-change merge does
// not rewrite the row.
func SeedHalfDaysIntoRegistry(st *store.Store, entries []HalfDayEntry) (int, error) {
	if st == nil || len(entries) == 0 {
		return 0, nil
	}
	raw, _ := st.GetSystemConfig(kernel.SessionRegistryConfigKey)
	reg, _ := kernel.LoadSessionRegistry(raw)
	if reg.HalfDays == nil {
		reg.HalfDays = map[string]string{}
	}
	changed := 0
	for _, e := range entries {
		if reg.HalfDays[e.Date] != e.EarlyCloseCT {
			reg.HalfDays[e.Date] = e.EarlyCloseCT
			changed++
		}
	}
	if changed == 0 {
		return 0, nil
	}
	if err := kernel.ValidateSessionRegistry(reg); err != nil {
		return 0, fmt.Errorf("merged registry failed validation: %w", err)
	}
	out, err := json.Marshal(reg)
	if err != nil {
		return 0, err
	}
	if err := st.SetSystemConfig(kernel.SessionRegistryConfigKey, string(out)); err != nil {
		return 0, err
	}
	return changed, nil
}

// maybeSeedHalfDays runs the producer once per CME session-day per process
// (idempotent across traders — the merge is a no-op when nothing changed).
// Called from runCycle housekeeping ABOVE the session gate, so weekend/holiday
// boots still seed (the F0 calendar-producer precedent).
func (at *AutoTrader) maybeSeedHalfDays(now time.Time) {
	if at.config.Exchange != "ninjatrader" || at.store == nil {
		return
	}
	day := kernel.CMESessionDayKey(now)
	if at.lastHalfDaySeedDay == day {
		return
	}
	at.lastHalfDaySeedDay = day

	entries, err := LoadHalfDaysFile()
	if err != nil {
		logger.Errorf("🚨 half-days CRITICAL: %v — trading continues NORMALLY on standard hours (fail-open). Fix %s.", err, halfDaysPath())
		return
	}
	if len(entries) == 0 {
		return
	}
	changed, err := SeedHalfDaysIntoRegistry(at.store, entries)
	if err != nil {
		logger.Errorf("🚨 half-days CRITICAL: seed failed: %v — trading continues normally", err)
		return
	}
	if changed > 0 {
		labels := make([]string, 0, len(entries))
		for _, e := range entries {
			labels = append(labels, e.Date+" "+e.EarlyCloseCT+" ("+e.Label+")")
		}
		at.logInfof("📅 half-days seeded: %d entr%s updated in the session registry — %s. Effective from the NEXT session-day registry refresh.",
			changed, map[bool]string{true: "y", false: "ies"}[changed == 1], strings.Join(labels, ", "))
	}
}

// LogHalfDaysBoot is the P4 boot integrity line (main.go): loaded count + the
// next upcoming half-day.
func LogHalfDaysBoot(now time.Time) {
	entries, err := LoadHalfDaysFile()
	if err != nil {
		logger.Errorf("🚨 half-days CRITICAL at boot: %v — trading continues normally on standard hours", err)
		return
	}
	if next, ok := NextUpcomingHalfDay(entries, now); ok {
		logger.Infof("📅 half-days [boot]: %d loaded from %s · next half-day: %s %s CT (%s)",
			len(entries), halfDaysPath(), next.Date, next.EarlyCloseCT, next.Label)
	} else {
		logger.Infof("📅 half-days [boot]: %d loaded from %s · no upcoming half-day registered", len(entries), halfDaysPath())
	}
}
