package kernel

import (
	"time"
)

// P0-B (2026-08-18) — ONE SESSION INSTANCE = ONE PLAN CHAIN.
//
// The plan identity used to be plannerTradeDateCT(now), which rolls at MIDNIGHT
// CT. ASIA's window (17:00 → 02:00) wraps midnight, so at 00:30 CT the bot
// looked up the NEXT calendar date, found no plan, and wrote a SECOND plan for
// the SAME session instance (live receipt: 2026-08-17:ASIA v1 written at
// 00:07 CT while the 08-16 chain had died at 17:37 CT). PlanChainTradeDate maps
// a moment to the CT calendar date of the session INSTANCE it belongs to:
//
//   - at/after today's window start → today (the current instance);
//   - before the start, inside a wrapped window's tail (00:00–02:00 for ASIA)
//     → yesterday (the instance that opened 17:00 yesterday);
//   - before the start, outside the tail (the pre-open read gap, e.g. ASIA's
//     16:55 read) → today (the instance about to open).
//
// ok=false when the session has no parseable window start (callers fall back to
// the midnight-roll date).
// SessionInstanceStart returns the CT wall-clock time the CURRENT (or, in the
// pre-open read gap, the upcoming) session instance opens, wrap-aware:
//
//   - at/after today's window start → today's start (the current instance);
//   - before the start, inside a wrapped window's tail (00:00–02:00 for ASIA)
//     → yesterday's start (the instance that opened 17:00 yesterday);
//   - before the start, outside the tail (the pre-open read gap, e.g. ASIA's
//     16:55 read) → today's start (the instance about to open).
//
// ok=false when the session has no parseable window start.
func SessionInstanceStart(sess *SessionDef, now time.Time) (time.Time, bool) {
	if sess == nil {
		return time.Time{}, false
	}
	startMin, ok := hhmmToMinLocal(sess.WindowStartCT)
	endMin, okEnd := hhmmToMinLocal(sess.WindowEndCT)
	if !ok {
		return time.Time{}, false
	}
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.Time{}, false
	}
	ct := now.In(loc)
	todayStart := time.Date(ct.Year(), ct.Month(), ct.Day(), startMin/60, startMin%60, 0, 0, loc)
	switch {
	case !ct.Before(todayStart):
		return todayStart, true // the instance that opened today
	case okEnd && endMin < startMin && ctMinutes(ct) < endMin:
		return todayStart.AddDate(0, 0, -1), true // tail of yesterday's wrapped instance
	default:
		return todayStart, true // pre-open: the instance about to open
	}
}

// PlanChainTradeDate maps a moment to the CT calendar date of the session
// INSTANCE it belongs to — the plan-chain identity (P0-B). ok=false when the
// session has no parseable window start (callers fall back to the midnight-roll
// date).
func PlanChainTradeDate(sess *SessionDef, now time.Time) (string, bool) {
	instStart, ok := SessionInstanceStart(sess, now)
	if !ok {
		return "", false
	}
	return instStart.Format("2006-01-02"), true
}

// hhmmToMinLocal parses "HH:MM" (24h) into minutes-since-midnight.
func hhmmToMinLocal(s string) (int, bool) {
	var h, m int
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	for _, ch := range s {
		if ch == ':' {
			continue
		}
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	h = int(s[0]-'0')*10 + int(s[1]-'0')
	m = int(s[3]-'0')*10 + int(s[4]-'0')
	if h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func ctMinutes(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}
