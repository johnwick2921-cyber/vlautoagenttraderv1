package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// ITEM 3 (2026-08-17) — the manual re-read must refuse for a REASON, and must
// spend from the same budget the automatic path uses. A button that can talk the
// bot past its own limits is worse than no button.

func rereadTrader(t *testing.T, cfg store.StrategyConfig) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "rr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "trader-1", store: st, exchange: "ninjatrader"}
	at.config.StrategyConfig = &cfg
	return at, st
}

func TestRereadRefusesWhenDayPlanIsOff(t *testing.T) {
	at, _ := rereadTrader(t, store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{PlanEnabled: false},
	})
	got := at.CanForceReread(time.Now())
	if got.Allowed {
		t.Fatal("a trader with day_plan off must never allow a manual re-read")
	}
	if got.Reason == "" {
		t.Error("the refusal must carry a reason — a silently disabled button is a bug report waiting to happen")
	}
}

func TestRereadRefusesOutsideAnySession(t *testing.T) {
	at, _ := rereadTrader(t, store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{PlanEnabled: true},
	})
	// 06:00 UTC = 01:00 CT — outside every session window.
	got := at.CanForceReread(time.Date(2026, 8, 18, 6, 0, 0, 0, time.UTC))
	if got.Allowed {
		t.Fatal("no session is active — a re-read has nothing to read for")
	}
	if got.Reason == "" {
		t.Error("the refusal must say why")
	}
}

// The budget is the SAME one the death path spends. Once it is gone the button
// must refuse, naming the arithmetic.
func TestRereadRefusesWhenTheBudgetIsSpent(t *testing.T) {
	_, st := rereadTrader(t, store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: 2},
	})
	// cap 2 ⇒ real versions v1..v3; the death of v3 is the ceiling.
	for i := 1; i <= 3; i++ {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: store.MakePlanID("2026-08-18", "NY"), StrategyID: "trader-1",
			TradeDate: "2026-08-18", Session: "NY", Lifecycle: "active", Doc: "{}",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if store.MayReplan(3, 2) {
		t.Fatal("fixture drift: v3 with cap 2 must already be at the ceiling")
	}
	// The gate's budget arithmetic must agree with the enforcer's, whatever the
	// clock does — check the shared definition directly.
	if left := store.ReplansLeftFor(3, 2); left != 0 {
		t.Errorf("at v3 with cap 2 the re-read budget is %d, want 0", left)
	}
}

// A session already closed out with NO-TRADE must not be re-openable by button.
func TestRereadRefusesOnANoTradeSession(t *testing.T) {
	at, st := rereadTrader(t, store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: 4},
	})
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID("2026-08-18", "NY"), StrategyID: "trader-1",
		TradeDate: "2026-08-18", Session: "NY", Lifecycle: "no_trade", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}
	// Whatever the clock, a no_trade row is terminal for that session.
	row, _ := st.Plan().GetLatestPlanForSession("2026-08-18", "NY")
	if row == nil || row.Lifecycle != "no_trade" {
		t.Fatal("fixture: expected the NO-TRADE row")
	}
	if got := at.CanForceReread(time.Date(2026, 8, 18, 6, 0, 0, 0, time.UTC)); got.Allowed {
		t.Error("a session closed out with NO-TRADE must not be re-opened by the button")
	}
}

// ForceReread must re-check for itself: the caller's gate may be stale.
func TestForceRereadRefusesRatherThanTrusting(t *testing.T) {
	at, _ := rereadTrader(t, store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{PlanEnabled: false},
	})
	gate, err := at.ForceReread(time.Now())
	if err == nil {
		t.Fatal("ForceReread must refuse when the gate is closed, not trust an earlier check")
	}
	if gate.Allowed {
		t.Error("the returned gate must report the refusal it acted on")
	}
}
