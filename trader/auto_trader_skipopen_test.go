package trader

import (
	"path/filepath"
	"testing"

	"nofx/store"
)

func skipTestTrader(t *testing.T, planEnabled bool) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{
		id:       "tr1",
		exchange: "ninjatrader",
		store:    st,
		config: AutoTraderConfig{
			NinjaTraderSymbol: "MNQ",
			StrategyConfig: &store.StrategyConfig{
				DayPlan: &store.DayPlanConfig{PlanEnabled: planEnabled},
			},
		},
	}
	return at, st
}

func openAPosition(t *testing.T, st *store.Store, traderID string) {
	t.Helper()
	if err := st.Position().Create(&store.TraderPosition{
		TraderID: traderID, Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 21500, Status: "OPEN", EntryTime: 1, CreatedAt: 1, UpdatedAt: 1,
	}); err != nil {
		t.Fatalf("create open position: %v", err)
	}
}

func TestSkipWhileOpen(t *testing.T) {
	at, st := skipTestTrader(t, true)

	// day_plan on, but flat → do NOT skip (the AI can look for an entry).
	if skip, _ := at.skipWhileOpen(); skip {
		t.Fatalf("flat trader must not skip")
	}

	// Now holding → skip the decision cycle.
	openAPosition(t, st, "tr1")
	skip, why := at.skipWhileOpen()
	if !skip {
		t.Fatalf("holding a position must skip the cycle")
	}
	if why == "" {
		t.Fatalf("skip reason should describe the open position")
	}
}

func TestSkipWhileOpenDormantWhenPlanOff(t *testing.T) {
	// day_plan OFF: the gate is dormant even while holding — existing flip/refusal
	// behavior is unchanged (flip-safety for non-day-plan strategies).
	at, st := skipTestTrader(t, false)
	openAPosition(t, st, "tr1")
	if skip, _ := at.skipWhileOpen(); skip {
		t.Fatalf("plan-off trader must not engage skip-while-open (dormant)")
	}

	// A crypto trader is likewise unaffected regardless of day_plan.
	at.exchange = "binance"
	at.config.StrategyConfig.DayPlan.PlanEnabled = true
	if skip, _ := at.skipWhileOpen(); skip {
		t.Fatalf("crypto trader must never engage skip-while-open")
	}
}
