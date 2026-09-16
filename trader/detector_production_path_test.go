package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// 1B WIRING PROOF (owner ruling 2026-09-03). The pre-existing
// TestDetectorWritesThroughTheProductionPath does NOT drive the production
// path — it calls at.recordDetectorOutputs directly, so it passed for the
// whole period in which nothing called the hook and both stores booted 0/0.
//
// This test drives the REAL read: it calls assemblePlannerInputWithCtx (the
// one production entry that resolves the void scope) on a fixture tape and
// then asserts the stores are non-empty. If the call site is ever removed
// again, this goes red — the previous test would not.
func TestPlannerReadWritesDetectorStores(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "prodpath.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// A28: assemblePlannerInputWithCtx is an ENTRY POINT and takes its own
	// time.Now(), so the fixture is anchored to the real clock rather than a
	// stated one. The tape ends at now so the void scope is non-empty.
	now := time.Now()
	bars := oscillatingTape(29141.25, now.Add(-700*time.Minute), 700)
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline { return bars }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	at := &AutoTrader{
		id: "hoang", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{
			DayPlan: &store.DayPlanConfig{PlanEnabled: true, PlannerTimeframes: []string{"D", "4h", "1h", "15m"}},
		}},
	}

	if n := st.CandidatePool().CountPool(); n != 0 {
		t.Fatalf("precondition: candidate_pool should start empty, got %d", n)
	}

	// THE PRODUCTION READ — no detector call anywhere in this test.
	_ = at.assemblePlannerInput("NY", kernel.CMESessionDayStart(now).Format("2026-01-02"))

	pool := st.CandidatePool().CountPool()
	outcomes := st.TouchOutcomes().CountOutcomes()
	t.Logf("after ONE production read: candidate_pool=%d touch_outcomes=%d", pool, outcomes)
	if pool == 0 {
		t.Fatalf("candidate_pool EMPTY after a planner read — the 1B hook is not wired into the read path")
	}
	if outcomes == 0 {
		t.Fatalf("touch_outcomes EMPTY after a planner read over an oscillating tape — the detector did not run on the read's seated levels")
	}
}
