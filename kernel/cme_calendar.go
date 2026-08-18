package kernel

import (
	"fmt"
	"time"
)

// IsCMEOpen reports whether CME Globex is open for index futures at the given time.
// Globex hours (Chicago time):
//
//	Sunday 17:00 → Friday 16:00, with a 60-minute daily break at 16:00–17:00.
//
// Holidays observed: New Year, MLK Day, Presidents Day, Good Friday, Memorial Day,
// Juneteenth, Independence Day, Labor Day, Thanksgiving (+ day after), Christmas Eve,
// Christmas Day, New Year's Eve. Each may have shortened hours; for v1 we treat them
// as full closures and refuse to trade. Refine in Plan 3 if it becomes restrictive.
func IsCMEOpen(t time.Time) bool {
	chicago := CTLocation()
	ct := t.In(chicago)
	if isCMEHoliday(ct) {
		return false
	}
	wd := ct.Weekday()
	hour := ct.Hour()
	switch wd {
	case time.Saturday:
		return false
	case time.Sunday:
		return hour >= 17
	case time.Friday:
		return hour < 16
	default: // Mon-Thu
		return hour != 16
	}
}

// CMEClosedReason mirrors IsCMEOpen and, when the market is closed, returns a
// short human-readable reason for the transition log ("holiday" / "weekend" /
// "Friday close" / "daily break"). Invariant (asserted by tests): the returned
// bool is exactly !IsCMEOpen(t), so the two can never disagree.
func CMEClosedReason(t time.Time) (closed bool, reason string) {
	chicago := CTLocation()
	ct := t.In(chicago)
	if isCMEHoliday(ct) {
		return true, "holiday"
	}
	switch ct.Weekday() {
	case time.Saturday:
		return true, "weekend"
	case time.Sunday:
		if ct.Hour() < 17 {
			return true, "weekend"
		}
		return false, ""
	case time.Friday:
		if ct.Hour() >= 16 {
			return true, "Friday close"
		}
		return false, ""
	default: // Mon-Thu
		if ct.Hour() == 16 {
			return true, "daily break"
		}
		return false, ""
	}
}

// NextCMEOpen returns the earliest instant strictly after t at which the market
// re-opens (IsCMEOpen flips true). It walks forward hour-by-hour using IsCMEOpen
// itself as the oracle, so it stays exactly consistent with the gate — holidays
// and DST included — instead of duplicating the schedule. Hour granularity is
// exact: IsCMEOpen only changes state on hour boundaries (it inspects ct.Hour(),
// the weekday, and the holiday date). The 14-day cap is a safety backstop; the
// longest real closed stretch is ~3 days, so it never triggers in practice.
func NextCMEOpen(t time.Time) time.Time {
	chicago := CTLocation()
	ct := t.In(chicago)
	// Truncate to the top of the current Chicago hour, then walk forward.
	cur := time.Date(ct.Year(), ct.Month(), ct.Day(), ct.Hour(), 0, 0, 0, chicago)
	for i := 0; i < 24*14; i++ {
		cur = cur.Add(time.Hour)
		if IsCMEOpen(cur) {
			return cur
		}
	}
	return t // unreachable in practice; sane fallback keeps callers total
}

// CMESessionDayStart returns the start of the CME trading session-day that
// contains `now` — the most recent 17:00 America/Chicago boundary. The CME
// index-futures session day rolls at 17:00 CT (the daily break), so daily
// realized-P&L / trade-count guardrails measure realized P&L and entries from
// this instant (not midnight UTC).
func CMESessionDayStart(now time.Time) time.Time {
	chicago := CTLocation()
	ct := now.In(chicago)
	boundary := time.Date(ct.Year(), ct.Month(), ct.Day(), 17, 0, 0, 0, chicago)
	if ct.Hour() < 17 {
		boundary = boundary.AddDate(0, 0, -1)
	}
	return boundary
}

// CMESessionDayKey returns a stable key (the date of the session's 17:00 CT
// start) for the CME session-day containing now. Used to detect a session
// rollover for the daily-window reset.
func CMESessionDayKey(now time.Time) string {
	return CMESessionDayStart(now).Format("2006-01-02")
}

// InBlackoutWindow (Chunk 4) reports whether `now` (evaluated in America/Chicago)
// falls within the daily [startCT, endCT] window (HH:MM, 24h, Chicago time). The
// Strategy Studio time/news blackout guardrail uses it. An empty/malformed window
// → false (a misconfig never silently halts trading). Supports windows that wrap
// midnight (start > end); a zero-width window (start == end) is never in blackout.
func InBlackoutWindow(now time.Time, startCT, endCT string) bool {
	start, ok1 := parseHHMM(startCT)
	end, ok2 := parseHHMM(endCT)
	if !ok1 || !ok2 || start == end {
		return false
	}
	chicago := CTLocation()
	ct := now.In(chicago)
	cur := ct.Hour()*60 + ct.Minute()
	if start < end {
		return cur >= start && cur < end
	}
	return cur >= start || cur < end // wraps midnight
}

// parseHHMM parses "HH:MM" (24h) into minutes-since-midnight. ok=false on any
// malformed/out-of-range input.
func parseHHMM(s string) (int, bool) {
	var h, m int
	n, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil || n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// isCMEHoliday returns true if t falls on a CME-observed full-closure holiday.
// CME may have shortened-hours days (e.g. Good Friday, day after Thanksgiving),
// but for v1 we treat shortened days as full closures and refuse to trade.
// Refine in a later plan if this becomes operationally restrictive.
func isCMEHoliday(ct time.Time) bool {
	year := ct.Year()
	month := ct.Month()
	day := ct.Day()
	weekday := ct.Weekday()

	// Fixed-date holidays
	md := ct.Format("01-02")
	switch md {
	case "01-01": // New Year's Day
		return true
	case "06-19": // Juneteenth
		return true
	case "07-04": // Independence Day
		return true
	case "12-24": // Christmas Eve (early close treated as closure)
		return true
	case "12-25": // Christmas Day
		return true
	case "12-31": // New Year's Eve (early close treated as closure)
		return true
	}

	// MLK Day — 3rd Monday of January
	if month == time.January && weekday == time.Monday && (day-1)/7 == 2 {
		return true
	}

	// Presidents Day — 3rd Monday of February
	if month == time.February && weekday == time.Monday && (day-1)/7 == 2 {
		return true
	}

	// Good Friday — Friday before Easter
	if month == time.March || month == time.April {
		easter := easterSunday(year)
		goodFri := easter.AddDate(0, 0, -2)
		if ct.Year() == goodFri.Year() && ct.Month() == goodFri.Month() && ct.Day() == goodFri.Day() {
			return true
		}
	}

	// Memorial Day — last Monday of May
	if month == time.May && weekday == time.Monday {
		// Check if next Monday is in June (i.e. this is the last Monday of May)
		nextMon := ct.AddDate(0, 0, 7)
		if nextMon.Month() == time.June {
			return true
		}
	}

	// Labor Day — 1st Monday of September
	if month == time.September && weekday == time.Monday && day <= 7 {
		return true
	}

	// Thanksgiving — 4th Thursday of November (plus day after as early-close)
	if month == time.November && weekday == time.Thursday && (day-1)/7 == 3 {
		return true
	}
	// Day after Thanksgiving — Friday after 4th Thursday
	if month == time.November && weekday == time.Friday {
		thursday := ct.AddDate(0, 0, -1)
		if thursday.Month() == time.November && (thursday.Day()-1)/7 == 3 {
			return true
		}
	}

	return false
}

// easterSunday returns the date of Easter Sunday in the given year (Western/Gregorian).
// Used only for Good Friday calculation.
func easterSunday(year int) time.Time {
	// Anonymous Gregorian algorithm (Meeus/Jones/Butcher)
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
