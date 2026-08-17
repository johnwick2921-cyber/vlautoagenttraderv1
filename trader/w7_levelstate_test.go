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
	base := time.Now().Add(-time.Duration(n+2) * time.Minute)
	var bars []market.Kline
	for i := 0; i < n; i++ {
		ct := base.Add(time.Duration(i) * time.Minute)
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
	base := time.Now().Add(-time.Duration(n+2) * time.Minute)
	var bars []market.Kline
	for i := 0; i < n; i++ {
		ct := base.Add(time.Duration(i) * time.Minute)
		delta := 1.0
		if i%2 == 0 {
			delta = -1.0
		}
		px := level + delta
		if i == n-1 {
			px = level // last close exactly at the level
		}
		bars = append(bars, market.Kline{
			OpenTime: ct.UnixMilli(), Open: level, High: level + 3, Low: level - 3, Close: px,
			CloseTime: ct.Add(time.Minute).UnixMilli(),
		})
	}
	return bars
}

// W7 — a level ACCEPTED THROUGH in one session is persisted consumed, and when the
// SAME level (same price/type identity) is re-derived in a later session and
// re-touched, it does NOT return fresh — it stays burned and re-arm refuses it.
func TestW7LevelBurnedStaysBurnedAcrossSessions(t *testing.T) {
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
	key := store.MakeLevelKey("MNQ", kernel.LevelTypeFromLabel(label), "", kernel.LevelBinIndex(levelPx))

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

	// SESSION 1 — price accepts THROUGH the level → consumed.
	market.FuturesBarsProvider = func(string, string, int) []market.Kline {
		return barsClosingAbove(levelPx, 20)
	}
	at.recordLevelState()

	cur, _ := st.LevelState().Get(key)
	if cur == nil {
		t.Fatalf("level %s must be persisted after session 1", key)
	}
	if !cur.Consumed {
		t.Fatalf("accepted-through level must be consumed, got consumed=%v freshness=%s", cur.Consumed, cur.Freshness)
	}

	// re-arm must refuse a consumed level regardless of cooldown/re-form.
	if ok, why := store.ReArmEligible(cur, time.Now().UnixMilli(), store.ReArmCooldownMin, true); ok {
		t.Fatalf("a consumed level must never re-arm, got eligible (why=%q)", why)
	}

	// SESSION 2 — same level re-derived, price returns and re-touches it (fresh facts).
	market.FuturesBarsProvider = func(string, string, int) []market.Kline {
		return barsHoveringAt(levelPx, 20)
	}
	at.recordLevelState()

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
