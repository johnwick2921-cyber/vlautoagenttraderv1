package trader

import (
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// S3 (mega-research 2026-08-26) — the entry-time attribution guarantee: the
// plan identity stamped onto a position comes from the ACTIVE plan at DECISION
// time. Across a session handoff (LONDON → NY) the citation must follow the
// NEW plan — the version-leak class (register S3: pos 563 plan_version=9 vs
// LONDON max v2) is impossible when the stamp is captured at decision time.
func TestRecordPlanCitationFollowsActivePlanAcrossHandoff(t *testing.T) {
	at := &AutoTrader{
		id:       "t-s3",
		exchange: "ninjatrader",
		config: AutoTraderConfig{
			NinjaTraderSymbol: "MNQ",
			StrategyConfig: &store.StrategyConfig{DayPlan: &store.DayPlanConfig{
				PlanEnabled: true,
			}},
		},
	}
	london := &kernel.ActivePlan{PlanID: "2026-08-25:LONDON", Session: "LONDON", Version: 2}
	ny := &kernel.ActivePlan{PlanID: "2026-08-25:NY", Session: "NY", Version: 3}
	cur := london
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{
		ActivePlan: func(symbol string) *kernel.ActivePlan { return cur },
	})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })

	d := &kernel.Decision{Action: "open_long", CitedScenario: "S1", Price: 29300, StopLoss: 29280, TakeProfit: 29360}

	// Decision 1 — LONDON plan active.
	at.recordPlanCitation(d)
	if !at.lastCitation.valid || at.lastCitation.planID != "2026-08-25:LONDON" ||
		at.lastCitation.session != "LONDON" || at.lastCitation.tradeDate != "2026-08-25" {
		t.Fatalf("citation 1 must carry the LONDON identity: %+v", at.lastCitation)
	}

	// Session handoff — the active plan switches; the citation must follow.
	cur = ny
	at.recordPlanCitation(d)
	if !at.lastCitation.valid || at.lastCitation.planID != "2026-08-25:NY" ||
		at.lastCitation.session != "NY" || at.lastCitation.tradeDate != "2026-08-25" ||
		at.lastCitation.planVersion != 3 {
		t.Fatalf("citation 2 must carry the NY identity (entry-time plan, not the prior session's): %+v", at.lastCitation)
	}
}
