package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// barsClosingAbove builds n 1m bars ending ~2m before now, all CLOSED, each closing
// `above` the level (a clean acceptance-through → the level is consumed).
func barsClosingAbove(level float64, n int) []market.Kline {
	return barsClosingAboveSince(time.Now().Add(-time.Duration(n+2)*time.Minute), level, n)
}

// barsClosingAboveSince builds n CLOSED bars that STRADDLE the level once
// (the touch that gates acceptance) and then close strictly above it,
// starting at `base` — the honest "accepted through" sequence.
func barsClosingAboveSince(base time.Time, level float64, n int) []market.Kline {
	var bars []market.Kline
	for i := 0; i < n; i++ {
		ct := base.Add(time.Duration(i) * time.Minute)
		if i == 0 {
			bars = append(bars, market.Kline{ // straddle = touch
				OpenTime: ct.UnixMilli(), Open: level - 2, High: level + 2, Low: level - 4, Close: level - 1,
				CloseTime: ct.Add(time.Minute).UnixMilli(),
			})
			continue
		}
		px := level + 8 + float64(i) // strictly above, trending up
		bars = append(bars, market.Kline{
			OpenTime: ct.UnixMilli(), Open: px - 1, High: px + 2, Low: px - 3, Close: px,
			CloseTime: ct.Add(time.Minute).UnixMilli(),
		})
	}
	return bars
}

// barsHoveringAt builds n CLOSED bars oscillating AT the level (never 2 consecutive
// closes beyond either side → StillValid stays true = a fresh re-touch).
func barsHoveringAt(level float64, n int) []market.Kline {
	return barsHoveringSince(time.Now().Add(-time.Duration(n+2)*time.Minute), level, n)
}

func barsHoveringSince(base time.Time, level float64, n int) []market.Kline {
	var bars []market.Kline
	for i := 0; i < n; i++ {
		ct := base.Add(time.Duration(i) * time.Minute)
		// ENTRY-MECHANICS ADDENDUM: closes EXACTLY at the level (highs/lows
		// straddle it) — a close BEYOND would accept-through under the new
		// one-close default (5m_close) and the re-touch alert would never fire.
		bars = append(bars, market.Kline{
			OpenTime: ct.UnixMilli(), Open: level, High: level + 3, Low: level - 3, Close: level,
			CloseTime: ct.Add(time.Minute).UnixMilli(),
		})
	}
	return bars
}

// W7 — a level ACCEPTED THROUGH in one session is persisted consumed, and when the
// SAME level (same price/type identity) is re-derived in a later session and
// re-touched, it does NOT return fresh — it stays burned and re-arm refuses it.
func TestW7LevelBurnedStaysBurnedAcrossSessions(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	for _, hm := range [][2]int{{11, 0}, {17, 17}, {17, 18}, {17, 19}, {17, 20}, {17, 21}, {17, 22}, {17, 23}} {
		now := time.Date(2026, 9, 8, hm[0], hm[1], 13, 0, loc)
		t.Run(now.Format("15:04"), func(t *testing.T) { testW7LevelBurnedAt(t, now) })
	}
}

func testW7LevelBurnedAt(t *testing.T, now time.Time) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	at.store = st
	at.id = "t1"

	const levelPx = 30050.0
	label := "nPOC·Tue"
	key := store.MakeLevelKey("t1", "MNQ", kernel.LevelTypeFromLabel(label), "", kernel.LevelBinIndex(levelPx))

	// A fixed active plan with one level; restore the provider after.
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan {
		return &kernel.ActivePlan{
			Doc:     kernel.PlanDoc{Levels: []kernel.PlanLevel{{Price: levelPx, Label: label, Grade: "A"}}},
			Session: "NY", Version: 1, ReplansLeft: 2,
		}
	}})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	prevBars := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prevBars }()

	// SESSION 1a — the plan is born and the level row is CREATED while price
	// hovers at the level. Windowed (P1c): nothing has touched+accepted yet →
	// must NOT be consumed at creation time.
	market.FuturesBarsProvider = func(string, string, int) []market.Kline {
		return barsHoveringSince(now.Add(-22*time.Minute), levelPx, 20)
	}
	at.recordLevelStateAt(now)

	cur, _ := st.LevelState().Get(key)
	if cur == nil {
		t.Fatalf("level %s must be persisted after session 1", key)
	}
	if cur.Consumed {
		t.Fatalf("a just-created, untouched level must NOT be consumed, got %+v", cur)
	}

	// SESSION 1b — bars SINCE the row's birth accept through (close beyond on
	// the rule timeframe) → consumed (role-flip). Consumption is windowed on
	// created_at (P1c), so backdate the row to when the synthetic bars began.
	// Keep the 20-bar acceptance sequence inside one completed hour. The old
	// now-22m sequence straddled the 17:00 CME day and accidentally shrank
	// DailyRangeProxy enough to exclude this level from the activation window.
	base := now.Truncate(time.Hour).Add(-time.Hour)
	if err := st.LevelState().Backdate(key, base.Add(-time.Minute)); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	acceptanceBars := barsClosingAboveSince(base, levelPx, 20)
	rangeProxy := kernel.DailyRangeProxy(acceptanceBars, now)
	if active := kernel.ActivePlanLevels([]kernel.PlanLevel{{Price: levelPx}}, acceptanceBars[len(acceptanceBars)-1].Close, rangeProxy, kernel.ActivationWindowK); len(active) != 1 {
		t.Fatalf("fixture must keep the level in the activation window: dATR=%v", rangeProxy)
	}
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return acceptanceBars }
	at.recordLevelStateAt(now)

	cur, _ = st.LevelState().Get(key)
	if !cur.Consumed {
		t.Fatalf("touched-and-accepted-through level must be consumed, got consumed=%v freshness=%s", cur.Consumed, cur.Freshness)
	}

	// re-arm must refuse a consumed level regardless of cooldown/re-form.
	if ok, why := store.ReArmEligible(cur, now.UnixMilli(), store.ReArmCooldownMin, true); ok {
		t.Fatalf("a consumed level must never re-arm, got eligible (why=%q)", why)
	}

	// SESSION 2 — same level re-derived, price returns and re-touches it (fresh facts).
	market.FuturesBarsProvider = func(string, string, int) []market.Kline {
		return barsHoveringSince(now.Add(-22*time.Minute), levelPx, 20)
	}
	at.recordLevelStateAt(now)

	after, _ := st.LevelState().Get(key)
	if after == nil || !after.Consumed {
		t.Fatalf("burned level must STAY burned across sessions, got %+v", after)
	}

	// the re-touch of a burned level surfaces a P1 alert (state influenced behavior).
	alerts, _ := st.Alert().List("t1", 20)
	found := false
	for _, a := range alerts {
		if a.Kind == "level-burned" {
			found = true
		}
	}
	if !found {
		t.Fatal("re-touch of a burned level must emit a P1 level-burned alert")
	}
}

// W7 — the writer is GATED: day_plan off (crypto / no-plan) never writes level state.
func TestW7GatedOff(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	no := false
	at := mkTrader("ninjatrader", &no, "5m") // day_plan OFF
	at.store = st
	at.id = "t1"

	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan {
		return &kernel.ActivePlan{Doc: kernel.PlanDoc{Levels: []kernel.PlanLevel{{Price: 30050, Label: "PDH", Grade: "A"}}}, Session: "NY", Version: 1}
	}})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	prevBars := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prevBars }()
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return barsClosingAbove(30050, 20) }

	at.recordLevelState()

	rows, _ := st.LevelState().ListForSymbol("MNQ")
	if len(rows) != 0 {
		t.Fatalf("day_plan off must write NO level state, got %d rows", len(rows))
	}
}
