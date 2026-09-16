package store

import (
	"testing"
)

// P0-A2 — plan identity is trader-scoped: two day-plan traders on the same
// (trade_date, session) each get their OWN chain id, and a trader with an
// existing chain (legacy id included) CONTINUES it — version continuity
// across owner resets must survive.
func TestP0A2TraderScopedPlanIdentity(t *testing.T) {
	ps := newPlanTestStore(t).Plan()
	td, sess := "2026-08-18", "NY"

	// Trader A: no chain yet → fresh trader-scoped id.
	idA1 := ps.ResolvePlanID(td, sess, "trader-A")
	if idA1 != MakePlanIDForTrader("trader-A", td, sess) {
		t.Fatalf("fresh chain must be trader-scoped, got %q", idA1)
	}
	v1, err := ps.AppendPlan(&PlanDB{PlanID: idA1, StrategyID: "trader-A",
		TradeDate: td, Session: sess, Lifecycle: "active", Doc: "{}"})
	if err != nil || v1 != 1 {
		t.Fatalf("append A v1: %v %v", v1, err)
	}

	// Trader B: same session → a DIFFERENT id.
	idB := ps.ResolvePlanID(td, sess, "trader-B")
	if idB == idA1 {
		t.Fatalf("trader B must not share trader A's chain id")
	}
	if _, err := ps.AppendPlan(&PlanDB{PlanID: idB, StrategyID: "trader-B",
		TradeDate: td, Session: sess, Lifecycle: "active", Doc: "{}"}); err != nil {
		t.Fatalf("append B: %v", err)
	}

	// Trader A again → same chain, version 2.
	if got := ps.ResolvePlanID(td, sess, "trader-A"); got != idA1 {
		t.Fatalf("trader A must continue its chain, got %q", got)
	}
	v2, err := ps.AppendPlan(&PlanDB{PlanID: idA1, StrategyID: "trader-A",
		TradeDate: td, Session: sess, Lifecycle: "active", Doc: "{}"})
	if err != nil || v2 != 2 {
		t.Fatalf("append A v2: %v %v", v2, err)
	}

	// Each trader's reader sees ONLY its own chain.
	a, _ := ps.GetLatestPlanForTraderSession(td, sess, "trader-A")
	b, _ := ps.GetLatestPlanForTraderSession(td, sess, "trader-B")
	if a == nil || a.Version != 2 || a.PlanID != idA1 {
		t.Fatalf("trader A latest wrong: %+v", a)
	}
	if b == nil || b.Version != 1 || b.PlanID != idB {
		t.Fatalf("trader B latest wrong: %+v", b)
	}

	// Legacy continuity: trader C already has a date:session chain → resolver
	// returns the legacy id, so its next append continues at version N+1.
	legacy := MakePlanID(td, sess)
	if _, err := ps.AppendPlan(&PlanDB{PlanID: legacy, StrategyID: "trader-C",
		TradeDate: td, Session: sess, Lifecycle: "no_trade", Doc: "{}"}); err != nil {
		t.Fatalf("append legacy: %v", err)
	}
	if got := ps.ResolvePlanID(td, sess, "trader-C"); got != legacy {
		t.Fatalf("legacy chain must be continued, got %q", got)
	}

	// And the legacy chain never bleeds into trader A or B.
	c, _ := ps.GetLatestPlanForTraderSession(td, sess, "trader-C")
	if c == nil || c.PlanID != legacy {
		t.Fatalf("trader C latest wrong: %+v", c)
	}
}
