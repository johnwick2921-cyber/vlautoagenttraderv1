package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// CLASS 145 (W-DRIFT-WIDEN-CAP, 2026-09-17) — PRODUCTION CALL SITES. The
// 2026-09-17 ASIA read authored v1 at 16:38:46 CT inside the CME halt; the
// freshest 1m bar was the 15:59 bar, so the F6 measurement read 2,326,426 ms
// (journal 45208) and both the plan-write path (auto_trader_planner.go, the
// stored "+31m (clock drift)" from the 16:30:11 measurement, journal 44334)
// and the arm path (auto_trader_calendar.go t1WindowsFor, the rendered
// "+39m") widened the BOJ ±15m band by the feed's AGE. These tests inject
// that measurement through the F6 seam and read the two production functions.

const driftCapDate = "2026-09-17" // Thursday, CDT (UTC-5): 21:30 CT = 02:30Z next day

func driftCapTrader(t *testing.T, driftMs int64) (*AutoTrader, *store.Store) {
	t.Helper()
	// T1Currencies ALL: the class was born under the pre-W-T1-CURRENCIES regime
	// where every T1 hard-blocked; under the shipped USD default this JPY
	// event is advisory and opens no window (trader/t1_currencies_test.go).
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, SessionsEnabled: []string{"ASIA", "NY"}, T1Currencies: []string{store.T1CurrencyAll}}}
	at, st := resetTrader(t, cfg)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(string) (int64, bool) { return driftMs, true }
	t.Cleanup(func() { clockHoldDriftFn = orig })
	slice := &store.CalendarSliceDB{
		TradeDate: driftCapDate, Source: "forexfactory",
		EventsJSON: `[{"time":"2026-09-18T02:30:00Z","currency":"JPY","title":"BOJ Policy Rate","impact":"T1"}]`,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if _, err := st.Calendar().SaveSliceIfAbsent(slice); err != nil {
		t.Fatalf("save slice: %v", err)
	}
	return at, st
}

// asiaNow is 18:00 CT on the trade date — ASIA active, the halt over, the
// evaluation clock the arm gate runs on. The DRIFT is injected, so the
// wall-clock offset from the 16:38 read is carried by the measurement.
func asiaNow() time.Time { return time.Date(2026, 9, 17, 18, 0, 0, 0, chicagoLoc()) }

// Arm path: t1WindowsFor via currentT1Windows with the halt-age measurement.
func TestArmPathT1WindowsCappedUnderHaltAge(t *testing.T) {
	at, _ := driftCapTrader(t, 2_326_426)
	windows := at.currentT1Windows(asiaNow())
	if len(windows) != 1 {
		t.Fatalf("the BOJ slice must produce exactly one window, got %+v", windows)
	}
	w := windows[0]
	if w.Start != 21*60+13 || w.End != 21*60+47 {
		t.Fatalf("arm path must widen ±15m by the %dm cap, not 39m: got %d–%d (%s)", kernel.ClockWidenCapMinutes, w.Start, w.End, w.Label)
	}
	if strings.Contains(w.Label, "clock drift") {
		t.Fatalf("feed age must not be called clock drift on the arm path: %q", w.Label)
	}
	// 20:36 CT — inside the old 39m-widened band, outside the capped one — must be tradeable.
	at2036 := 20*60 + 36
	for _, x := range windows {
		in := (x.Start <= x.End && at2036 >= x.Start && at2036 <= x.End) || (x.Start > x.End && (at2036 >= x.Start || at2036 <= x.End))
		if in {
			t.Fatalf("20:36 CT must NOT be blacked out under the cap: window %d–%d", x.Start, x.End)
		}
	}
}

// Arm path, real skews: 42 s → +1m unlabelled (existing behaviour); 90 s →
// "+2m (clock drift)".
func TestArmPathT1WindowsRealSkewUnchanged(t *testing.T) {
	at, _ := driftCapTrader(t, 42_000)
	w := at.currentT1Windows(asiaNow())
	if len(w) != 1 || w[0].Start != 21*60+14 || w[0].End != 21*60+46 || strings.Contains(w[0].Label, "clock drift") {
		t.Fatalf("42 s skew: want 21:14–21:46 unlabelled, got %+v", w)
	}
	at2, _ := driftCapTrader(t, 90_000)
	w2 := at2.currentT1Windows(asiaNow())
	if len(w2) != 1 || w2[0].Start != 21*60+13 || w2[0].End != 21*60+47 || !strings.HasSuffix(w2[0].Label, "+2m (clock drift)") {
		t.Fatalf("90 s skew: want 21:13–21:47 labelled, got %+v", w2)
	}
}

// Plan-write path: plannerT1Lines (the step runPlannerReadWithTriggerClaimed
// calls with the clockHoldAuthoring verdict) with the LIVE 16:38 measurement.
func TestPlannerT1LinesCappedUnderHaltAge(t *testing.T) {
	at, _ := driftCapTrader(t, 2_326_426)
	deferred, widen, drift, have := at.clockHoldAuthoring()
	if deferred || !have || widen != 2_326_426 || drift != 2_326_426 {
		t.Fatalf("positive feed age never defers and is carried raw: deferred=%v widen=%d drift=%d have=%v", deferred, widen, drift, have)
	}
	cal := []kernel.PlannerCalendarEvent{{TimeCT: "21:30", Currency: "JPY", Title: "BOJ Policy Rate", Impact: "T1"}}
	lines := at.plannerT1Lines(cal, have, widen, drift, driftCapDate, "ASIA")
	if len(lines) != 1 {
		t.Fatalf("want one T1 line, got %v", lines)
	}
	if strings.Contains(lines[0], "clock drift") || strings.Contains(lines[0], "+31m") || strings.Contains(lines[0], "+39m") {
		t.Fatalf("the stored plan line must not carry the halt age as drift: %q", lines[0])
	}
	if !strings.Contains(lines[0], "BOJ Policy Rate 21:30 CT ±15m") {
		t.Fatalf("the T1 line must survive: %q", lines[0])
	}
	// The same step with a real 90 s skew keeps the existing labelled behaviour.
	skew := at.plannerT1Lines(cal, true, 90_000, 90_000, driftCapDate, "ASIA")
	if !strings.Contains(skew[0], "+2m (clock drift)") {
		t.Fatalf("a real 90 s skew must still be stated: %q", skew[0])
	}
	// The stale note the journal carries names the cause and the cap.
	note := kernel.ClockDriftStaleNote(drift)
	if !strings.Contains(note, "feed stale 38m") || !strings.Contains(note, "NOT widened beyond the 2m cap") {
		t.Fatalf("stale note: %q", note)
	}
}

// Brief item (3): the arm path reads the LIVE calendar slice, not the plan's
// frozen no_trade lines — when the feed corrects BOJ from 21:30 to 22:00 CT
// the band follows the feed. Proven at the production function.
func TestArmPathFollowsLiveCalendarCorrection(t *testing.T) {
	at, st := driftCapTrader(t, 0)
	before := at.currentT1Windows(asiaNow())
	if len(before) != 1 || before[0].Start != 21*60+15 {
		t.Fatalf("seeded BOJ 21:30 CT must give a 21:15 start, got %+v", before)
	}
	changed, err := st.Calendar().UpdateLiveSliceIfChanged(&store.CalendarSliceDB{
		TradeDate: driftCapDate, Source: "forexfactory",
		EventsJSON: `[{"time":"2026-09-18T03:00:00Z","currency":"JPY","title":"BOJ Policy Rate","impact":"T1"}]`,
		CreatedAt:  time.Now().UnixMilli(),
	})
	if err != nil || !changed {
		t.Fatalf("live correction must write: changed=%v err=%v", changed, err)
	}
	after := at.currentT1Windows(asiaNow())
	if len(after) != 1 || after[0].Start != 21*60+45 || after[0].End != 22*60+15 {
		t.Fatalf("the arm path must follow the corrected 22:00 CT event, got %+v", after)
	}
}
