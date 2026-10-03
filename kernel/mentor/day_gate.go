package mentor

import (
	"time"

	"vl/market"
)

// §7 — the pre-session routine, under 5 minutes [D5.1 p1 @ 18:36]:
//  1. turn on the key levels;
//  2. switch to the DAILY and measure how far the current daily candle has
//     already run (Asia high to low, through pre-market);
//  3. check the 4-HOUR trigger, then the 1-HOUR trigger.
//
// Table verbatim [D5.1 p1]:
//   normal run      + agree    → TRADE, following them
//   300–400+ pts    + conflict → "TẮT MÁY NGHỈ LUÔN CHO EM" (shut it off)
//   300–400+ pts    + agree    → trade, but DON'T target big —
//                                "15 điểm bán, 10 điểm bán"
//
// The Globex session runs 17:00 CT → 08:30 CT; "today's run" is that window
// ending at the most recent 08:30 CT boundary (the measurement is taken in
// the pre-session routine, before the RTH open).

const (
	globexOpenMin  = 17 * 60   // 17:00 CT
	globexCloseMin = 8*60 + 30 // 08:30 CT
)

// DayVerdict is one row of the §7 table.
type DayVerdict int

const (
	// DayTrade: normal run, HTF agrees (or 1h silent) — trade, following them.
	DayTrade DayVerdict = iota
	// DaySpent: the day already ran SpentPts+ and the HTFs agree — trade but
	// cap the target ("15 điểm bán, 10 điểm bán").
	DaySpent
	// DayOff: 4h/1h conflict (spent or not) — shut the machine off for the
	// day ("TẮT MÁY NGHỈ LUÔN CHO EM").
	DayOff
)

// DayGate holds the §7 knobs (the Config integration diff is routed via the
// CTO; until it lands, DefaultDayGate supplies the method defaults).
type DayGate struct {
	// SpentPts: a run at/above this before the open means "already run"
	// — 300 [D5.1 p1 @ 15:57 "Already run 300–400 before the open → 80–90%
	// it ranges"].
	SpentPts float64
	// TargetCapPts: on a spent day, the target distance cap — 15
	// ("15 điểm bán, 10 điểm bán" [D5.1 p1]).
	TargetCapPts float64
}

// DefaultDayGate returns the §7 defaults (300 / 15).
func DefaultDayGate() DayGate {
	return DayGate{SpentPts: 300, TargetCapPts: 15}
}

// ctime loads America/Chicago (the mentor quotes all times in US Central
// [D4.4 p1 @ 01:45 "em tính giờ Texas"]). Falls back to a fixed −6h zone if
// tzdata is unavailable.
func ctime() *time.Location {
	if loc, err := time.LoadLocation("America/Chicago"); err == nil {
		return loc
	}
	return time.FixedZone("CST6", -6*3600)
}

// GlobexRun measures the §7 step-2 run: the high−low of the Globex session
// that ended at the most recent 08:30 CT boundary at or before now (i.e.
// yesterday 17:00 CT → today 08:30 CT once the RTH day has opened, and the
// previous session's window before 08:30). ok=false when fewer than 2 bars
// cover the window — the gate then reads as "not measured yet", never as a
// spent day.
func GlobexRun(bars []market.Kline, now int64, loc *time.Location) (run float64, ok bool) {
	if loc == nil {
		loc = ctime()
	}
	t := time.UnixMilli(now).In(loc)
	end := time.Date(t.Year(), t.Month(), t.Day(), globexCloseMin/60, globexCloseMin%60, 0, 0, loc)
	if t.Before(end) {
		end = end.AddDate(0, 0, -1)
	}
	start := time.Date(end.Year(), end.Month(), end.Day()-1, globexOpenMin/60, globexOpenMin%60, 0, 0, loc)
	loMs, hiMs := start.UnixMilli(), end.UnixMilli()

	hi, lo := 0.0, 0.0
	n := 0
	for _, b := range bars {
		if b.OpenTime < loMs || b.OpenTime >= hiMs {
			continue
		}
		if n == 0 {
			hi, lo = b.High, b.Low
		} else {
			if b.High > hi {
				hi = b.High
			}
			if b.Low < lo {
				lo = b.Low
			}
		}
		n++
	}
	if n < 2 {
		return 0, false
	}
	return hi - lo, true
}

// DayGateVerdict applies the §7 table. conflict is HTFConflict(htf): a 4h/1h
// conflict shuts the day off regardless of the run — the ban list reads
// "4-hour and 1-hour triggers in conflict — ESPECIALLY with the daily range
// already spent" [D4.4 p1 @ 13:18; D5.1 p1 @ 19:22].
func DayGateVerdict(run float64, haveRun bool, conflict bool, g DayGate) (DayVerdict, string) {
	if g.SpentPts <= 0 || g.TargetCapPts <= 0 {
		g = DefaultDayGate()
	}
	if conflict {
		return DayOff, "4h/1h conflict — 'TẮT MÁY NGHỈ LUÔN CHO EM', no trades today [D5.1 p1 @ 19:22; D4.4 p1 @ 13:18]"
	}
	if haveRun && run >= g.SpentPts {
		return DaySpent, "already run 300–400+ before the open — trade but don't target big, 15/10 pt sales [D5.1 p1 @ 15:57]"
	}
	return DayTrade, ""
}

// CapTargetForDay applies the §7 spent-day cap: on a DaySpent day the target
// distance is capped at TargetCapPts ("15 điểm bán, 10 điểm bán" — he sells
// at 15 or 10, not at the full target). The cap moves the target CLOSER to
// entry, never further, and never flips the side.
func CapTargetForDay(in Intent, v DayVerdict, g DayGate) Intent {
	if v != DaySpent || in.Price == 0 || in.Target == 0 {
		return in
	}
	if g.TargetCapPts <= 0 {
		g = DefaultDayGate()
	}
	dist := in.Target - in.Price
	if in.Side == SideShort {
		dist = in.Price - in.Target
	}
	if dist <= g.TargetCapPts {
		return in
	}
	if in.Side == SideShort {
		in.Target = in.Price - g.TargetCapPts
	} else {
		in.Target = in.Price + g.TargetCapPts
	}
	return in
}
