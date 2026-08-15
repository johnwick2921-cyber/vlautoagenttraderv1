package kernel

import (
	"math"
	"time"

	"nofx/market"
)

// P1.1 — session-tagged bars → multi-day level extractor.
//
// Pure + deterministic: given intraday bars, the session registry, and `now`,
// derive the day-trader map's structural levels (facts=Go). Bars are tagged by
// CME session (via the P0.3 registry) and grouped by CT calendar day / futures
// session-day, per the contract's level naming (PDH = calendar-day only).
//
// Only levels that HAVE data are emitted — a missing window (cold start, thin
// cache) simply omits that level, which the WARMING badge surfaces honestly. No
// backfill: the map warms forward as the durable session-profile store (P1.3)
// accumulates.

type calDayAgg struct {
	high, low       float64
	closePx         float64
	closeOpenTime   int64
	rthHigh, rthLow float64
	hasRTH          bool
}

func newCalDayAgg() *calDayAgg {
	return &calDayAgg{
		high: math.Inf(-1), low: math.Inf(1),
		rthHigh: math.Inf(-1), rthLow: math.Inf(1),
	}
}

// ExtractMultiDayLevels returns the structural map levels relative to `now`:
// PDH/PDL/PDC (prior calendar day), RTH-H/L (prior day's NY session), AS/LDN/ON
// (the current overnight of now's futures session-day), and PW/PM (prior week /
// month). Closed bars only.
func ExtractMultiDayLevels(bars []market.Kline, reg SessionRegistry, now time.Time) []DetectedLevel {
	loc := chicago()
	nowMs := now.UnixMilli()
	nowCT := now.In(loc)
	todayCal := nowCT.Format("2006-01-02")
	nowFut := CMESessionDayKey(now)

	weekStart := startOfWeekCT(nowCT)
	priorWeekStart := weekStart.AddDate(0, 0, -7)
	priorWeekEnd := weekStart
	monthStart := time.Date(nowCT.Year(), nowCT.Month(), 1, 0, 0, 0, 0, loc)
	priorMonthStart := monthStart.AddDate(0, -1, 0)
	priorMonthEnd := monthStart

	cal := map[string]*calDayAgg{}
	var asH, asL, ldnH, ldnL float64 = math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(1)
	var hasAS, hasLDN bool
	var pwH, pwL float64 = math.Inf(-1), math.Inf(1)
	var pmH, pmL float64 = math.Inf(-1), math.Inf(1)
	var hasPW, hasPM bool

	for i := range bars {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue // closed bars only
		}
		bt := time.UnixMilli(b.OpenTime).In(loc)
		calKey := bt.Format("2006-01-02")

		a := cal[calKey]
		if a == nil {
			a = newCalDayAgg()
			cal[calKey] = a
		}
		if b.High > a.high {
			a.high = b.High
		}
		if b.Low < a.low {
			a.low = b.Low
		}
		if b.OpenTime >= a.closeOpenTime {
			a.closePx = b.Close
			a.closeOpenTime = b.OpenTime
		}

		session := ""
		if s, ok := reg.ActiveSession(bt); ok {
			session = s.Name
		}
		if session == SessionNY {
			a.hasRTH = true
			if b.High > a.rthHigh {
				a.rthHigh = b.High
			}
			if b.Low < a.rthLow {
				a.rthLow = b.Low
			}
		}

		// Overnight of the CURRENT futures session-day.
		if CMESessionDayKey(bt) == nowFut {
			switch session {
			case SessionAsia:
				hasAS = true
				if b.High > asH {
					asH = b.High
				}
				if b.Low < asL {
					asL = b.Low
				}
			case SessionLondon:
				hasLDN = true
				if b.High > ldnH {
					ldnH = b.High
				}
				if b.Low < ldnL {
					ldnL = b.Low
				}
			}
		}

		if !bt.Before(priorWeekStart) && bt.Before(priorWeekEnd) {
			hasPW = true
			if b.High > pwH {
				pwH = b.High
			}
			if b.Low < pwL {
				pwL = b.Low
			}
		}
		if !bt.Before(priorMonthStart) && bt.Before(priorMonthEnd) {
			hasPM = true
			if b.High > pmH {
				pmH = b.High
			}
			if b.Low < pmL {
				pmL = b.Low
			}
		}
	}

	var out []DetectedLevel

	// Prior calendar day (PDH/PDL/PDC + its RTH).
	if priorCal := mostRecentPriorDay(cal, todayCal); priorCal != "" {
		a := cal[priorCal]
		out = append(out,
			lineLevel(KindPDH, a.high, "PDH", priorCal, true),
			lineLevel(KindPDL, a.low, "PDL", priorCal, true),
			lineLevel(KindPDC, a.closePx, "PDC", priorCal, true),
		)
		if a.hasRTH {
			out = append(out,
				lineLevel(KindRTHH, a.rthHigh, "RTH-H", priorCal, false),
				lineLevel(KindRTHL, a.rthLow, "RTH-L", priorCal, false),
			)
		}
	}

	// Current overnight (Asia / London / composite ON).
	nowFutDate := CMESessionDayStart(now).Format("2006-01-02")
	if hasAS {
		out = append(out,
			lineLevel(KindASH, asH, "AS-H", nowFutDate, false),
			lineLevel(KindASL, asL, "AS-L", nowFutDate, false),
		)
	}
	if hasLDN {
		out = append(out,
			lineLevel(KindLDNH, ldnH, "LDN-H", nowFutDate, false),
			lineLevel(KindLDNL, ldnL, "LDN-L", nowFutDate, false),
		)
	}
	if hasAS || hasLDN {
		onH := math.Max(asH, ldnH)
		onL := math.Min(asL, ldnL)
		out = append(out,
			lineLevel(KindONH, onH, "ONH", nowFutDate, false),
			lineLevel(KindONL, onL, "ONL", nowFutDate, false),
		)
	}

	// Prior week / month (HTF).
	if hasPW {
		wLabel := priorWeekStart.Format("2006-01-02")
		out = append(out,
			lineLevel(KindPWH, pwH, "PWH", wLabel, true),
			lineLevel(KindPWL, pwL, "PWL", wLabel, true),
		)
	}
	if hasPM {
		mLabel := priorMonthStart.Format("2006-01")
		out = append(out,
			lineLevel(KindPMH, pmH, "PMH", mLabel, true),
			lineLevel(KindPML, pmL, "PML", mLabel, true),
		)
	}

	return out
}

// mostRecentPriorDay returns the latest calendar-day key strictly before today
// that has data, or "" if none.
func mostRecentPriorDay(cal map[string]*calDayAgg, todayCal string) string {
	best := ""
	for k := range cal {
		if k < todayCal && k > best {
			best = k
		}
	}
	return best
}

// startOfWeekCT returns Monday 00:00 of t's ISO week, in t's location.
func startOfWeekCT(t time.Time) time.Time {
	daysSinceMon := (int(t.Weekday()) + 6) % 7 // Sun=0 → 6, Mon=1 → 0, ...
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -daysSinceMon)
}
