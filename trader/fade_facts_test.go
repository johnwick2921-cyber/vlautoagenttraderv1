package trader

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── W2 E1 — THE 09-03 REPLAY, THROUGH THE FACTS BUILDER ─────────────────────
//
// kernel/fade_permission_test.go pins the PREDICATE against hand-fed facts.
// This pins the PATH: the real 2026-09-03 5m tape goes in through the bars
// seam, the session registry says NY opens 08:30, and the builder must
// produce the facts the predicate then judges. If the builder read a wall
// clock, dropped the closed-bar rule, or took the OR from the wrong bar, this
// is where it shows. One clock per evaluation, stated (A28/E9).

// sep3Bars is the NY session tape measured from the store (C2), 08:30–12:00 CT
// as UTC-5 open times. h/l/c per 5m bar.
func sep3Bars(t *testing.T) []market.Kline {
	t.Helper()
	loc, _ := time.LoadLocation("America/Chicago")
	type b struct {
		hm      string
		o, h, l float64
		c       float64
	}
	rows := []b{
		{"08:30", 29249.5, 29285.0, 29222.75, 29253.75}, {"08:35", 29254.0, 29334.5, 29199.25, 29329.25},
		{"08:40", 29329.0, 29359.75, 29313.75, 29345.25}, {"08:45", 29345.5, 29368.0, 29318.75, 29359.25},
		{"08:50", 29359.25, 29375.25, 29324.5, 29348.75}, {"08:55", 29349.25, 29354.75, 29296.75, 29332.25},
		{"09:00", 29332.25, 29335.5, 29250.0, 29267.5}, {"09:05", 29267.5, 29283.75, 29241.5, 29267.5},
		{"09:10", 29267.75, 29303.75, 29249.5, 29301.5}, {"09:15", 29301.0, 29349.5, 29295.5, 29342.75},
		{"09:20", 29343.5, 29363.75, 29328.0, 29350.75}, {"09:25", 29350.5, 29364.75, 29327.5, 29343.25},
		{"09:30", 29342.75, 29360.5, 29284.0, 29287.75}, {"09:35", 29288.25, 29319.25, 29280.25, 29284.5},
		{"09:40", 29284.25, 29311.75, 29272.5, 29283.75}, {"09:45", 29283.75, 29315.25, 29275.5, 29309.25},
		{"09:50", 29309.75, 29315.0, 29285.0, 29313.0}, {"09:55", 29313.0, 29367.75, 29246.0, 29363.25},
		{"10:00", 29363.5, 29437.75, 29362.0, 29436.75}, {"10:05", 29436.75, 29445.75, 29422.0, 29429.75},
		{"10:10", 29430.25, 29466.25, 29426.25, 29465.0}, {"10:15", 29464.75, 29480.0, 29460.75, 29472.75},
		{"10:20", 29472.75, 29494.75, 29453.5, 29481.75}, {"10:25", 29481.5, 29539.75, 29478.75, 29535.5},
		{"10:30", 29535.75, 29543.75, 29517.75, 29528.25}, {"11:55", 29490.0, 29495.0, 29485.0, 29490.0},
	}
	out := make([]market.Kline, 0, len(rows))
	for _, r := range rows {
		tm, _ := time.ParseInLocation("2006-01-02 15:04", "2026-09-03 "+r.hm, loc)
		out = append(out, market.Kline{OpenTime: tm.UnixMilli(), Open: r.o, High: r.h, Low: r.l, Close: r.c})
	}
	return out
}

func sep3Trader(t *testing.T) *AutoTrader {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "fade.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	reg := kernel.SessionRegistry{Sessions: []kernel.SessionDef{
		{Name: kernel.SessionNY, WindowStartCT: "08:30", WindowEndCT: "15:00", ReadCT: "08:00", FlatCT: "14:45", Enabled: true},
	}}
	blob, _ := json.Marshal(reg)
	if err := st.SetSystemConfig(kernel.SessionRegistryConfigKey, string(blob)); err != nil {
		t.Fatal(err)
	}
	prev := market.FuturesBarsProvider
	bars := sep3Bars(t)
	market.FuturesBarsProvider = func(_, tf string, _ int) []market.Kline {
		if tf == "5m" {
			return bars
		}
		return nil
	}
	prevHist := fadeORHistoryProvider
	// C5: the 13-session median is 81.25. Handed in as history so the test
	// does not depend on what the temp store holds.
	fadeORHistoryProvider = func(*AutoTrader, string, time.Time, int) (float64, int) { return 81.25, 13 }
	t.Cleanup(func() { market.FuturesBarsProvider = prev; fadeORHistoryProvider = prevHist })
	at := &AutoTrader{id: "t1", store: st, exchange: "ninjatrader"}
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	return at
}

func sep3At(t *testing.T, hhmm string) time.Time {
	t.Helper()
	loc, _ := time.LoadLocation("America/Chicago")
	tm, err := time.ParseInLocation("2006-01-02 15:04", "2026-09-03 "+hhmm, loc)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestSep3ReplayThroughTheBuilder(t *testing.T) {
	at := sep3Trader(t)

	// 09:02 — arm 35, the one that FILLED. OR closed at 08:35; IB not until
	// 09:30. Clock for this evaluation: 2026-09-03 09:02 CT.
	f := at.fadeFactsAt(sep3At(t, "09:02"), "MNQ", 29267.5, nil, "short")
	if !f.ORComplete || f.OR5mPts != 62.25 {
		t.Fatalf("09:02: OR must be the CLOSED 08:30 bar, 62.25 pts; got complete=%v pts=%.2f", f.ORComplete, f.OR5mPts)
	}
	if f.IBComplete {
		t.Fatalf("09:02: the IB does not exist yet — the builder read bars that had not closed")
	}
	v := kernel.FadePermissionAt(sep3At(t, "09:02"), f)
	if !v.Evaluated || !v.Permitted || len(v.Exclusions) != 0 {
		t.Errorf("09:02 arm 35 must read PERMITTED with no exclusion — that is the finding; got %+v", v)
	}

	// 09:31 — one minute after the IB completes. 12 bars closed, no break.
	f = at.fadeFactsAt(sep3At(t, "09:31"), "MNQ", 29287.75, nil, "short")
	if !f.IBComplete || f.IBHigh != 29375.25 || f.IBLow != 29199.25 {
		t.Fatalf("09:31: IB must be 29199.25..29375.25; got complete=%v %.2f..%.2f", f.IBComplete, f.IBLow, f.IBHigh)
	}
	if f.ClosedBucketBeyondIB {
		t.Errorf("09:31: no closed bucket beyond the IB yet")
	}

	// 10:05 — the 10:00 bar has CLOSED at 29436.75, beyond the IB high.
	// (b) continuous fires here, per the owner's ruling.
	f = at.fadeFactsAt(sep3At(t, "10:05"), "MNQ", 29436.75, nil, "short")
	if !f.ClosedBucketBeyondIB {
		t.Fatalf("10:05: the 10:00 close 29436.75 > IB high 29375.25 must register as a held closed bucket")
	}
	v = kernel.FadePermissionAt(sep3At(t, "10:05"), f)
	if v.Permitted || !hasEx(v, kernel.FadeExIBBrokenHeld) {
		t.Errorf("10:05: must be excluded naming ib_held; got %+v", v)
	}

	// 10:04 — the 10:00 bar has NOT closed. Knowable-at-now: no exclusion.
	f = at.fadeFactsAt(sep3At(t, "10:04"), "MNQ", 29436.75, nil, "short")
	if f.ClosedBucketBeyondIB {
		t.Errorf("10:04: the 10:00 bar closes at 10:05 — reading it early is reading the future")
	}
}

// (c) BEYOND THE MAP is judged against the seated map handed in, in the
// scenario's own direction. Price above every seated level excludes a SHORT
// and does not exclude a LONG.
func TestBeyondMapIsDirectional(t *testing.T) {
	at := sep3Trader(t)
	seated := []kernel.ScoredLevel{
		{DetectedLevel: kernel.DetectedLevel{Price: 29375.25}},
		{DetectedLevel: kernel.DetectedLevel{Price: 29300.0}},
	}
	now := sep3At(t, "10:05")
	short := kernel.FadePermissionAt(now, at.fadeFactsAt(now, "MNQ", 29436.75, seated, "short"))
	long := kernel.FadePermissionAt(now, at.fadeFactsAt(now, "MNQ", 29436.75, seated, "long"))
	if !hasEx(short, kernel.FadeExBeyondMap) {
		t.Errorf("a SHORT at 29436.75 above max seated 29375.25 is beyond the map; got %+v", short.Exclusions)
	}
	if hasEx(long, kernel.FadeExBeyondMap) {
		t.Errorf("a LONG above the map is not fading into an unmapped move; got %+v", long.Exclusions)
	}
}

func hasEx(v kernel.FadeVerdict, name string) bool {
	for _, e := range v.Exclusions {
		if e.Name == name {
			return true
		}
	}
	return false
}
