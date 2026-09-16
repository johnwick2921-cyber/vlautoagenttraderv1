package kernel

// WEEKLY-BIAS WAVE (2026-08-30) — W6 knob resolvers. Every knob is env-only,
// resolved here through a small resolver func with a shipped default; garbage
// input silently falls back to the default (mirror of the
// persistWatchdogSeconds() pattern). Documented in the guide (guide law):
// any knob change ships with its guide card in the SAME change.

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// WeeklyReadCTDefault is the shipped Sunday read time: Sunday 16:30 CT
// (before the 17:00 CT open of the new CME week).
const WeeklyReadCTDefault = "sun 16:30"

// WeeklyReadSpec parses the WEEKLY_READ_CT knob ("<dow> HH:MM", dow ∈
// mon|tue|wed|thu|fri|sat|sun, case-insensitive). Missing or garbage →
// (Sunday, 16, 30). Hour/minute out of range → default too.
func WeeklyReadSpec() (wd time.Weekday, hour, min int) {
	wd, hour, min = time.Sunday, 16, 30
	v := strings.TrimSpace(os.Getenv("WEEKLY_READ_CT"))
	if v == "" {
		return
	}
	parts := strings.Fields(strings.ToLower(v))
	if len(parts) != 2 {
		return
	}
	dow := map[string]time.Weekday{
		"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday,
		"wed": time.Wednesday, "thu": time.Thursday, "fri": time.Friday,
		"sat": time.Saturday,
	}
	d, ok := dow[parts[0]]
	if !ok {
		return
	}
	hm := strings.Split(parts[1], ":")
	if len(hm) != 2 {
		return
	}
	h, err1 := strconv.Atoi(hm[0])
	m, err2 := strconv.Atoi(hm[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return
	}
	wd = d
	return wd, h, m
}

// WeekGoverningMonday returns the Monday that governs the CME week containing
// t (the Sunday 17:00 CT session belongs to the FOLLOWING Monday). Exported
// form of the W1 computation helper.
func WeekGoverningMonday(t time.Time) time.Time { return weekStartMonday(t) }

// WeeklyReadDeadline is the read instant of the week governing `now`: the
// Sunday BEFORE the week's Monday, at the resolved WEEKLY_READ_CT, in CT.
// E.g. for a week whose Monday is 2026-08-31 → Sunday 2026-08-30 16:30 CT.
func WeeklyReadDeadline(now time.Time) time.Time {
	_, hour, min := WeeklyReadSpec()
	monday := WeekGoverningMonday(now)
	sunday := monday.AddDate(0, 0, -int((monday.Weekday()-time.Sunday+7)%7))
	loc := CTLocation()
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(sunday.Year(), sunday.Month(), sunday.Day(), hour, min, 0, 0, loc)
}

// WeeklyConfluenceBandATR resolves WEEKLY_CONFLUENCE_BAND_ATR (the 5.1 shadow
// confluence band in ATR5m units). Garbage → 0.25.
func WeeklyConfluenceBandATR() float64 {
	if v := os.Getenv("WEEKLY_CONFLUENCE_BAND_ATR"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 {
			return f
		}
	}
	return 0.25
}

// WeeklyShadowMult resolves WEEKLY_SHADOW_MULT (the shadow grade multiplier for
// a weekly-confluent level). Garbage → 1.5.
func WeeklyShadowMult() float64 {
	if v := os.Getenv("WEEKLY_SHADOW_MULT"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 {
			return f
		}
	}
	return 1.5
}

// WeeklyCounterMode resolves WEEKLY_COUNTER_MODE: "warn" (default — the
// ⚖️ WEEKLY-COUNTER annotation logs) or "off" (silent). Anything else → warn.
func WeeklyCounterMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("WEEKLY_COUNTER_MODE"))) {
	case "off":
		return "off"
	default:
		return "warn"
	}
}

// WeeklyInvalidationTFDefault resolves WEEKLY_INVALIDATION_TF_DEFAULT (the
// basis TF used when a stored weekly doc's invalidation basis has no
// parseable timeframe token). Garbage → "1h".
func WeeklyInvalidationTFDefault() string {
	v := strings.TrimSpace(os.Getenv("WEEKLY_INVALIDATION_TF_DEFAULT"))
	if v == "" {
		return "1h"
	}
	return v
}

// PlannerCandlesEnabled resolves PLANNER_CANDLES ("on"|"off"): whether the
// session planner prompt carries the ## Candles table block. Default on.
// Garbage → on (fail toward the eyes).
func PlannerCandlesEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PLANNER_CANDLES"))) {
	case "off":
		return false
	default:
		return true
	}
}
