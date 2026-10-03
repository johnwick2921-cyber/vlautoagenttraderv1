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
// NOTE (CTO review E1): a NORMAL day with a 4h/1h conflict is NOT off — that
// is §5.4 case 3, a per-tick sit-out until the 1h flips. DayOff is ONLY
// spent AND conflict, read at the pre-open boundary (08:30 CT) and LATCHED
// for the rest of that RTH day: even if the 1h flips later, the machine
// stays off.
//
// The Globex session runs 17:00 CT → 08:30 CT. Before 08:30 the run is
// measured on the CURRENT session, 17:00 CT to NOW; from 08:30 on the
// 17:00→08:30 value is frozen (CTO review E2).

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
	// DayOff: spent AND conflict at the pre-open read — shut the machine off
	// for the day ("TẮT MÁY NGHỈ LUÔN CHO EM"). Latched once read.
	DayOff
	// DayNotMeasured: fewer than 2 bars cover the window — fail closed, no
	// mentor entries ("any trade you are vague about — don't" [§12]).
	DayNotMeasured
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

// DayLatch is the frozen §7 verdict for the current RTH day (CTO review E1).
// Once frozen, nothing re-reads it — a DayOff survives a later 1h flip.
type DayLatch struct {
	Date    string // CT date "2006-01-02" the verdict was frozen for
	Verdict DayVerdict
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

// sessionBounds returns the Globex session window for a moment in time:
// the 17:00 CT open and the 08:30 CT close of the session now belongs to.
func sessionBounds(now int64, loc *time.Location) (openMs, closeMs int64) {
	t := time.UnixMilli(now).In(loc)
	open := time.Date(t.Year(), t.Month(), t.Day(), globexOpenMin/60, globexOpenMin%60, 0, 0, loc)
	if open.After(t) {
		open = open.AddDate(0, 0, -1)
	}
	close := time.Date(open.Year(), open.Month(), open.Day()+1, globexCloseMin/60, globexCloseMin%60, 0, 0, loc)
	return open.UnixMilli(), close.UnixMilli()
}

// GlobexRun measures the §7 step-2 run [D5.1 p1 @ 18:36]. Before 08:30 CT the
// window is the CURRENT session from 17:00 CT to NOW (the "current daily
// candle" is still being drawn); from 08:30 CT on, the value is the frozen
// 17:00→08:30 high−low of the session that just ended (CTO review E2).
// ok=false when fewer than 2 bars cover the window — the gate then reads
// DayNotMeasured, never a spent day.
func GlobexRun(bars []market.Kline, now int64, loc *time.Location) (run float64, ok bool) {
	if loc == nil {
		loc = ctime()
	}
	openMs, closeMs := sessionBounds(now, loc)
	hiMs := now
	if !time.UnixMilli(now).In(loc).Before(time.UnixMilli(closeMs).In(loc)) {
		hiMs = closeMs // session ended: freeze the 17:00→08:30 value
	}
	loMs := openMs

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

// DayGateVerdict applies the §7 table. conflict is HTFConflict(htf):
//
//	unmeasured run               → DayNotMeasured (fail closed, §12)
//	run >= SpentPts AND conflict → DayOff (spent + conflict only)
//	run >= SpentPts              → DaySpent (small targets)
//	otherwise (incl. a conflict  → DayTrade — §5.4 case 3 handles the
//	on a normal day)              per-tick sit-out until the flip
func DayGateVerdict(run float64, haveRun bool, conflict bool, g DayGate) (DayVerdict, string) {
	if g.SpentPts <= 0 || g.TargetCapPts <= 0 {
		g = DefaultDayGate()
	}
	if !haveRun {
		return DayNotMeasured, "day run not measured — no mentor entries ('any trade you are vague about — don't' [§12])"
	}
	if conflict && run >= g.SpentPts {
		return DayOff, "spent AND 4h/1h conflict at the pre-open read — 'TẮT MÁY NGHỈ LUÔN CHO EM', no trades today [D5.1 p1 @ 19:22]"
	}
	if run >= g.SpentPts {
		return DaySpent, "already run 300–400+ before the open — trade but don't target big, 15/10 pt sales [D5.1 p1 @ 15:57]"
	}
	return DayTrade, ""
}

// LatchDay is the call site the evaluator uses (CTO review E1): the §7
// verdict is read at the pre-open boundary and then LATCHES for the rest of
// that RTH day. A latched DayOff survives everything — a later 1h flip does
// not reopen the machine. Before 08:30 CT the verdict re-reads live (a
// pre-open spent+conflict latches immediately); at/after 08:30 the verdict
// freezes as read.
func LatchDay(l DayLatch, now int64, loc *time.Location, run float64, haveRun bool, conflict bool, g DayGate) DayLatch {
	if loc == nil {
		loc = ctime()
	}
	t := time.UnixMilli(now).In(loc)
	date := t.Format("2006-01-02")
	open := time.Date(t.Year(), t.Month(), t.Day(), globexCloseMin/60, globexCloseMin%60, 0, 0, loc)
	if l.Date == date {
		// DayOff latches permanently; at/after 08:30 the whole verdict
		// freezes (the pre-session read is done).
		if l.Verdict == DayOff || !t.Before(open) {
			return l
		}
	}
	v, _ := DayGateVerdict(run, haveRun, conflict, g)
	return DayLatch{Date: date, Verdict: v}
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
