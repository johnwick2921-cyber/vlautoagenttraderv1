package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// TestStickyOwnerLevelInPlannerInput proves P3.6-C: an owner level set in one
// session (independent of any plan/session) appears in a LATER session's
// assembled planner input, tagged 👤.
func TestStickyOwnerLevelInPlannerInput(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	// Owner level set "in session N".
	if err := st.OwnerLevel().Save(&store.OwnerLevelDB{
		Symbol: "MNQ", Price: 15555, Label: "my level", Note: "watch this", ScenarioTag: "S1",
	}); err != nil {
		t.Fatalf("save owner level: %v", err)
	}

	prev := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prev }()
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		now := time.Now().UnixMilli()
		return []market.Kline{{OpenTime: now - 600_000, High: 15610, Low: 15590, Close: 15600, CloseTime: now - 300_000}}
	}

	at := &AutoTrader{
		id: "t1", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}},
	}

	// "session N+1" read (a different trade date) — the sticky owner level persists.
	input := at.assemblePlannerInput("NY", "2026-08-15")
	found := false
	for _, l := range input.Levels {
		if l.Kind == kernel.KindOwner && l.Price == 15555 {
			found = true
			if l.Grade != "A" || l.Info != "S1" {
				t.Fatalf("owner level tag/grade wrong: %+v", l)
			}
		}
	}
	if !found {
		t.Fatalf("sticky owner level must appear in the later session's planner input; got %d levels", len(input.Levels))
	}
}
