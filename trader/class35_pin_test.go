package trader

import (
	"testing"
	"time"

	"nofx/store"
)

// CLASS 35 (2026-09-01) — PIN TODAY'S LONDON CHAIN.
//
// Live diagnostic 07:58 CT: version=6, baseline=1 (no dayplan_reset row),
// cap=4 (LONDON session override). The chain was
//
//	v1 planner_fail_closed · v2 level_event · v3 dormant:flip ·
//	v4 level_event · v5 level_event · v6 level_event
//
// — ZERO death re-plans, ZERO owner re-reads — and the budget arithmetic said
// used = 6−1 = 5, replans_left = 0. The next scenario death would have
// fail-closed a session whose budget was never spent.
//
// This test uses ONLY the pre-fix surface (CanForceReread over an appended
// chain) so it compiles against the old tree and FAILS there: the gate must
// report the FULL budget for a chain that spent nothing.
func TestClass35PinTodayChain(t *testing.T) {
	enable, capFour := true, 4
	at, st := rereadTrader(t, store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{
			PlanEnabled: true,
			ReplanCap:   2, // strategy level — LONDON overrides to 4, exactly like the live config
			Sessions:    []store.DayPlanSessionOverride{{Session: "LONDON", Enable: &enable, ReplanCap: &capFour}},
		},
	})
	tradeDate := "2026-09-01"
	chain := []struct{ trigger, lifecycle string }{
		{"planner_fail_closed", "no_trade"},
		{"level_event", "active"},
		{"dormant:flip:flip-condition: 2x5m close above 29231.63 (buffer 0.5×ATR14, 2× 5m closes) → bias long", "dormant"},
		{"level_event", "active"},
		{"level_event", "active"},
		{"level_event", "active"},
	}
	for _, c := range chain {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: store.MakePlanID(tradeDate, "LONDON"), StrategyID: "trader-1",
			TradeDate: tradeDate, Session: "LONDON", TriggerReason: c.trigger, Lifecycle: c.lifecycle, Doc: "{}",
		}); err != nil {
			t.Fatal(err)
		}
	}
	row, _ := st.Plan().GetLatestPlanForTraderSession(tradeDate, "LONDON", "trader-1")
	if row == nil || row.Version != 6 {
		t.Fatalf("fixture: expected v6 latest, got %+v", row)
	}
	if got := store.GetResetBaseline(st, "trader-1", tradeDate, "LONDON"); got != 1 {
		t.Fatalf("fixture: no reset row → baseline 1, got %d", got)
	}
	// 07:58 CT Tue 2026-09-01 — inside the LONDON window, CME open.
	now := time.Date(2026, 9, 1, 7, 58, 0, 0, chicagoLoc())
	gate := at.CanForceReread(now)
	if gate.Session != "LONDON" {
		t.Fatalf("fixture: expected the LONDON gate, got %+v", gate)
	}
	if gate.ReplanCap != 4 {
		t.Fatalf("resolved cap = %d, want the LONDON override 4", gate.ReplanCap)
	}
	if gate.ReplansLeft != 4 {
		t.Errorf("replans_left = %d, want 4 — nothing in this chain spent budget (chain: fail_closed, level_event, dormant:flip, level_event ×3)", gate.ReplansLeft)
	}
	if !gate.Allowed {
		t.Errorf("MayReplan must be TRUE for an unspent budget; the gate refused: %q", gate.Reason)
	}
}
