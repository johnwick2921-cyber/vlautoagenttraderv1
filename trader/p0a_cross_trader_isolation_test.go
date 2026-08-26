package trader

import (
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// P0-A (2026-08-18) — CROSS-TRADER PLAN GOVERNANCE. With two day-plan traders
// live, one trader's plan, levels, overlays, digests or config must NEVER reach
// the other. Before this, GetLatestPlanForSession had no trader filter and the
// kernel providers were process-global singletons installed once under a
// sync.Once — the last writer's plan governed both executors. These tests pin
// the two seams that had the hole: the store lookup and the provider map.

func TestP0AStoreLookupIsTraderScoped(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "iso.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	tradeDate, session := "2026-08-17", "NY"
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID(tradeDate, session), StrategyID: "trader-A",
		TradeDate: tradeDate, Session: session, Lifecycle: "active", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID(tradeDate, session), StrategyID: "trader-B",
		TradeDate: tradeDate, Session: session, Lifecycle: "active", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	// Each trader sees exactly its own row — same (date, session), two chains.
	a, _ := st.Plan().GetLatestPlanForTraderSession(tradeDate, session, "trader-A")
	b, _ := st.Plan().GetLatestPlanForTraderSession(tradeDate, session, "trader-B")
	if a == nil || a.StrategyID != "trader-A" {
		t.Fatalf("trader-A lookup must return A's row, got %+v", a)
	}
	if b == nil || b.StrategyID != "trader-B" {
		t.Fatalf("trader-B lookup must return B's row, got %+v", b)
	}
	// An unknown trader gets nothing — never the other trader's plan.
	if c, _ := st.Plan().GetLatestPlanForTraderSession(tradeDate, session, "trader-C"); c != nil {
		t.Fatalf("trader-C must see no plan, got %+v", c)
	}
}

func TestP0AProviderMapIsPerTrader(t *testing.T) {
	tradeDate, session := "2026-08-17", "NY"
	st, err := store.New(filepath.Join(t.TempDir(), "iso2.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID(tradeDate, session), StrategyID: "trader-A",
		TradeDate: tradeDate, Session: session, Lifecycle: "active", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	// Register a FIXED (clock-independent) provider for each trader, reading its
	// OWN scoped row — the production closure shape (installActivePlanProvider).
	planFor := func(traderID string) kernel.TraderPlanProviders {
		return kernel.TraderPlanProviders{
			ActivePlan: func(symbol string) *kernel.ActivePlan {
				row, err := st.Plan().GetLatestPlanForTraderSession(tradeDate, session, traderID)
				if err != nil || row == nil || row.Lifecycle != "active" {
					return nil
				}
				return &kernel.ActivePlan{Session: session, Version: row.Version}
			},
		}
	}
	kernel.SetTraderPlanProviders("trader-A", planFor("trader-A"))
	kernel.SetTraderPlanProviders("trader-B", planFor("trader-B"))
	t.Cleanup(func() {
		kernel.SetTraderPlanProviders("trader-A", kernel.TraderPlanProviders{})
		kernel.SetTraderPlanProviders("trader-B", kernel.TraderPlanProviders{})
	})

	if p := kernel.ActivePlanFor("trader-A", "MNQ"); p == nil || p.Version != 1 {
		t.Fatalf("A must receive its own plan, got %+v", p)
	}
	// THE BUG THIS KILLS: B has a registered provider but no plan of its own —
	// it must receive NOTHING, not A's plan.
	if p := kernel.ActivePlanFor("trader-B", "MNQ"); p != nil {
		t.Fatalf("B must NEVER receive A's plan, got %+v", p)
	}
	// An unregistered trader receives nothing.
	if p := kernel.ActivePlanFor("trader-C", "MNQ"); p != nil {
		t.Fatalf("unregistered trader must receive nothing, got %+v", p)
	}
}

// TestP0AInstallRegistersPerTraderID proves installActivePlanProvider registers
// under the trader's OWN id (the production wiring), not a shared singleton.
func TestP0AInstallRegistersPerTraderID(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "iso3.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	a := &AutoTrader{id: "trader-A", exchange: "ninjatrader", store: st}
	a.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	installActivePlanProvider(a, st)

	if _, ok := kernel.TraderPlanProvidersFor("trader-A"); !ok {
		t.Fatalf("install must register the provider under the trader's own id")
	}
	if _, ok := kernel.TraderPlanProvidersFor("trader-B"); ok {
		t.Fatalf("B was never installed — it must have no providers")
	}
	t.Cleanup(func() { kernel.SetTraderPlanProviders("trader-A", kernel.TraderPlanProviders{}) })
}

// TestP0AScenarioStatusKeyIsTraderScoped pins the scenario-status system_config
// key: two traders sharing a plan_id used to share one key (last writer won).
func TestP0AScenarioStatusKeyIsTraderScoped(t *testing.T) {
	a := store.ScenarioStatusKey("trader-A", "2026-08-17:NY")
	b := store.ScenarioStatusKey("trader-B", "2026-08-17:NY")
	if a == b {
		t.Fatalf("scenario-status keys must differ per trader, both %q", a)
	}
	if a != "scenario_status:trader-A:2026-08-17:NY" {
		t.Fatalf("unexpected key shape %q", a)
	}
}
