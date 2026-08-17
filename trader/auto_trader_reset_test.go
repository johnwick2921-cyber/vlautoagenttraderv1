package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// P6 — the owner reset. Distinct from the re-read: it ABANDONS the chain
// (append-only — history and death reasons preserved), restores the FULL re-plan
// budget via a baseline marker, clears the NO-TRADE state, and writes a fresh
// plan through the normal read path with trigger_reason "owner reset". It never
// touches positions, brackets, guardrail counters or the daily cage.

func resetTrader(t *testing.T, cfg store.StrategyConfig) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "rst.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "trader-1", store: st, exchange: "ninjatrader"}
	at.config.StrategyConfig = &cfg
	at.mcpClient = &fakeDecisionClient{} // planner falls back to the primary client
	return at, st
}

func TestResetRefusesWhenDayPlanIsOff(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: false}})
	got := at.CanForceReset(time.Now())
	if got.Allowed || got.Reason == "" {
		t.Fatalf("day_plan off must refuse with a reason, got %+v", got)
	}
}

func TestResetRefusesOutsideAnySession(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	// 06:00 UTC = 01:00 CT — outside every session window (night).
	got := at.CanForceReset(time.Date(2026, 8, 18, 6, 0, 0, 0, time.UTC))
	if got.Allowed || got.Reason == "" {
		t.Fatalf("night must refuse with a reason, got %+v", got)
	}
}

func TestResetRefusesWithNoPlanYet(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	// NY window (08:30–14:45 CT) on a weekday the CME is open.
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, chicagoLoc())
	got := at.CanForceReset(now)
	if got.Allowed {
		t.Fatalf("no plan yet → nothing to abandon, must refuse, got %+v", got)
	}
	if got.Reason == "" {
		t.Fatal("the refusal must say why")
	}
}

func TestForceResetWritesFreshChainAndRestoresBudget(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: 4}})
	tradeDate := "2026-08-18"
	// Seed the exhausted chain: v1..v5 active + v6 NO-TRADE (cap 4).
	for i := 1; i <= 5; i++ {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: store.MakePlanID(tradeDate, "NY"), StrategyID: "trader-1",
			TradeDate: tradeDate, Session: "NY", Lifecycle: "active", Doc: "{}",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID(tradeDate, "NY"), StrategyID: "trader-1",
		TradeDate: tradeDate, Session: "NY", Lifecycle: "no_trade", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 18, 14, 0, 0, 0, chicagoLoc())
	// The planner returns a VALID plan → the reset writes an ACTIVE plan.
	at.mcpClient = &planClient{}
	gate, err := at.ForceReset(now)
	if err != nil {
		t.Fatalf("ForceReset: %v", err)
	}
	if !gate.Allowed {
		t.Fatalf("reset must be allowed mid-session, got %+v", gate)
	}

	// The marker records the seam at v7 (latest+1) → the new chain's first plan
	// is its v1, free, with the FULL budget restored.
	if got := store.GetResetBaseline(st, tradeDate, "NY"); got != 7 {
		t.Fatalf("baseline = %d, want 7", got)
	}
	// Budget restored: v7 measured from baseline 7 has the FULL cap.
	if left := store.ReplansLeftFrom(7, store.GetResetBaseline(st, tradeDate, "NY"), 4); left != 4 {
		t.Fatalf("replans left after reset = %d, want 4", left)
	}
	// The new row is an ACTIVE plan (NO-TRADE cleared), trigger "owner reset".
	row, _ := st.Plan().GetLatestPlanForSession(tradeDate, "NY")
	if row == nil || row.Lifecycle != "active" {
		t.Fatalf("the reset must write an ACTIVE plan, got %+v", row)
	}
	if row.TriggerReason != "owner_reset" {
		t.Fatalf("trigger_reason = %q, want owner_reset", row.TriggerReason)
	}
	// History preserved: the old chain still holds all six rows, NO-TRADE included.
	old, _ := st.Plan().GetPlan(store.MakePlanID(tradeDate, "NY"), 6)
	if old == nil || old.Lifecycle != "no_trade" {
		t.Fatalf("the abandoned chain's NO-TRADE marker must survive, got %+v", old)
	}
}

func TestForceResetFailsClosedOnBadRead(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: 4}})
	tradeDate := "2026-08-18"
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID(tradeDate, "NY"), StrategyID: "trader-1",
		TradeDate: tradeDate, Session: "NY", Lifecycle: "no_trade", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}
	// The planner call fails every attempt → the normal path writes a NO-TRADE
	// plan (fail-closed), never an error, never a mutation of anything else.
	at.mcpClient = &errorDecisionClient{}
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, chicagoLoc())
	gate, err := at.ForceReset(now)
	if err != nil {
		t.Fatalf("a failed read must not surface as a reset error: %v", err)
	}
	if !gate.Allowed {
		t.Fatalf("gate must have allowed the reset, got %+v", gate)
	}
	row, _ := st.Plan().GetLatestPlanForSession(tradeDate, "NY")
	if row == nil || row.Lifecycle != "no_trade" || row.TriggerReason != "planner_fail_closed" {
		t.Fatalf("bad read must fail-closed into a NO-TRADE plan, got %+v", row)
	}
}

// errorDecisionClient fails every call so the planner read fail-closes.
type errorDecisionClient struct{ fakeDecisionClient }

func (e *errorDecisionClient) CallWithMessages(string, string) (string, error) {
	return "", &timeoutErr2{}
}

// planClient returns a schema-valid plan so the read writes an ACTIVE row.
type planClient struct{ fakeDecisionClient }

func (p *planClient) CallWithMessages(string, string) (string, error) {
	return validTraderPlanJSON, nil
}

type timeoutErr2 struct{}

func (e *timeoutErr2) Error() string { return "timeout" }

func chicagoLoc() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}
	return loc
}
