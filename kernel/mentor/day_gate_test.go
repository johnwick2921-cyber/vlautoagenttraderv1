package mentor

import (
	"testing"
	"time"

	"vl/market"
)

// §7 — the pre-session routine table [D5.1 p1 @ 18:36, 19:22, 15:57]:
//   normal run      + agree    → TRADE
//   300–400+ pts    + conflict → "TẮT MÁY NGHỈ LUÔN CHO EM" (off)
//   300–400+ pts    + agree    → trade, small targets only (15/10 pt sales)

// TestDayGateVerdictTable — the three table rows, one per case.
func TestDayGateVerdictTable(t *testing.T) {
	g := DefaultDayGate()

	if v, reason := DayGateVerdict(120, true, false, g); v != DayTrade || reason != "" {
		t.Fatalf("normal run + agree: v=%v reason=%q, want DayTrade", v, reason)
	}
	if v, _ := DayGateVerdict(120, true, true, g); v != DayOff {
		t.Fatalf("normal run + conflict: v=%v, want DayOff (conflict shuts the day [D4.4 p1 @ 13:18])", v)
	}
	if v, reason := DayGateVerdict(350, true, true, g); v != DayOff || reason == "" {
		t.Fatalf("spent + conflict: v=%v, want DayOff 'TẮT MÁY NGHỈ LUÔN CHO EM'", v)
	}
	if v, reason := DayGateVerdict(350, true, false, g); v != DaySpent || reason == "" {
		t.Fatalf("spent + agree: v=%v, want DaySpent (trade, don't target big)", v)
	}
	// 1h silent is not a conflict — a spent day with the 4h alone is still
	// "agree" (case 2: follow the 4h).
	if v, _ := DayGateVerdict(350, true, false, g); v != DaySpent {
		t.Fatalf("spent + 1h-silent: v=%v, want DaySpent", v)
	}
}

// TestDayGateVerdictUnmeasuredRunIsNotSpent — an unmeasured run (fewer than 2
// bars in the window) reads as "not measured yet", never as a spent day.
func TestDayGateVerdictUnmeasuredRunIsNotSpent(t *testing.T) {
	g := DefaultDayGate()
	if v, _ := DayGateVerdict(0, false, false, g); v != DayTrade {
		t.Fatalf("unmeasured run: v=%v, want DayTrade", v)
	}
	if v, _ := DayGateVerdict(0, false, true, g); v != DayOff {
		t.Fatalf("unmeasured run + conflict: v=%v, want DayOff", v)
	}
}

// TestGlobexRunOnRecorded5mGlobex — canon 53: the recorded Globex session
// 2026-09-14 17:00 CT → 2026-09-15 08:30 CT (db-copy, MNQ 12-26, 5m, 186
// bars). Golden run = 264.5 pts (high 29495.75 − low 29231.25, computed from
// the same fixture rows on extraction, 2026-10-03).
func TestGlobexRunOnRecorded5mGlobex(t *testing.T) {
	bars := loadFixture(t, "mnq_5m_2026-09-15_globex", "5m")
	loc := ctime()
	// 09:00 CT on 2026-09-15: the most recent 08:30 boundary is today's, so
	// the window is 09-14 17:00 CT → 09-15 08:30 CT.
	now := time.Date(2026, 9, 15, 9, 0, 0, 0, loc).UnixMilli()
	run, ok := GlobexRun(bars, now, loc)
	if !ok {
		t.Fatal("Globex window found fewer than 2 bars")
	}
	if abs(run-264.5) > 0.01 {
		t.Fatalf("run = %.2f, want the golden 264.5", run)
	}
}

// TestGlobexRunWindowEdges — the window is [17:00 CT the previous day,
// 08:30 CT of the session's day]: before 08:30 the measurement reads the
// PREVIOUS session, after 08:30 it reads the one that just ended. Synthetic
// time arithmetic (the window logic itself); the recorded fixture covers the
// value path.
func TestGlobexRunWindowEdges(t *testing.T) {
	loc := time.FixedZone("CT", -5*3600)
	mk := func(day int, hour, minute int, h, l float64) market.Kline {
		ot := time.Date(2026, 9, day, hour, minute, 0, 0, loc)
		return market.Kline{OpenTime: ot.UnixMilli(), High: h, Low: l}
	}
	// Session A: 09-13 17:00 → 09-14 08:30 (run 10). Session B: 09-14 17:00
	// → 09-15 08:30 (run 40).
	bars := []market.Kline{
		mk(13, 17, 30, 100, 98),
		mk(13, 20, 0, 101, 99),  // A hi
		mk(14, 6, 0, 100.5, 91), // A lo → run 10
		mk(14, 17, 30, 130, 110),
		mk(14, 20, 0, 135, 105), // B hi
		mk(15, 7, 0, 115, 95),   // B lo → run 40
	}
	// before 08:30 CT on 09-15 the latest boundary is 09-14 08:30 → session A
	if run, ok := GlobexRun(bars, time.Date(2026, 9, 15, 8, 0, 0, 0, loc).UnixMilli(), loc); !ok || abs(run-10) > 1e-9 {
		t.Fatalf("pre-open read: run=%.2f ok=%v, want 10 (session A)", run, ok)
	}
	// after 08:30 CT on 09-15 → session B
	if run, ok := GlobexRun(bars, time.Date(2026, 9, 15, 9, 0, 0, 0, loc).UnixMilli(), loc); !ok || abs(run-40) > 1e-9 {
		t.Fatalf("post-open read: run=%.2f ok=%v, want 40 (session B)", run, ok)
	}
}

// TestCapTargetForDay — on a spent day the target distance is capped at
// TargetCapPts ("15 điểm bán, 10 điểm bán" [D5.1 p1 @ 15:57]); on a normal
// day the target is untouched.
func TestCapTargetForDay(t *testing.T) {
	g := DefaultDayGate()
	long := Intent{Side: SideLong, Price: 30000, Stop: 29990, Target: 30040}
	if got := CapTargetForDay(long, DayTrade, g); got.Target != 30040 {
		t.Fatalf("DayTrade must not move the target: %+v", got)
	}
	got := CapTargetForDay(long, DaySpent, g)
	if got.Target != 30015 {
		t.Fatalf("DaySpent long target = %.2f, want the 15-pt cap 30015", got.Target)
	}
	if got.Stop != long.Stop || got.Price != long.Price {
		t.Fatalf("the cap must only touch the target: %+v", got)
	}
	short := Intent{Side: SideShort, Price: 30000, Stop: 30010, Target: 29950}
	gotS := CapTargetForDay(short, DaySpent, g)
	if gotS.Target != 29985 {
		t.Fatalf("DaySpent short target = %.2f, want the 15-pt cap 29985", gotS.Target)
	}
	// an already-small target is not moved further away
	if got2 := CapTargetForDay(Intent{Side: SideLong, Price: 30000, Target: 30010}, DaySpent, g); got2.Target != 30010 {
		t.Fatalf("a 10-pt target must stay (never pushed further): %+v", got2)
	}
}
