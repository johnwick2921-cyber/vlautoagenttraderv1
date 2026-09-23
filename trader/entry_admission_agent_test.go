package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W-EXEC-TRUTH W0 (CTO Q17): a chat entry is refused under STRICT exactly like
// an uncited AI decision — the same chain, the same plan-mode refusal.
func TestChatEntryIsRefusedUnderStrictLikeADecision(t *testing.T) {
	withMaintenanceDir(t)
	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "strict"})
	at.id = "agent-door"
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan {
		return &kernel.ActivePlan{Doc: kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long"},
			Scenarios: []kernel.PlanScenario{{ID: "S1", Direction: "long"}}}, Session: "NY", Version: 1}
	}})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	// 11:00 CT on a Tuesday: inside the NY session, clear of first-5m, lunch
	// and the last-entry cutoff — so the refusal is the plan mode's.
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	reason, refused := at.AdmitManualEntryAt("MNQ", "open_long", nyMidday)
	if !refused || !strings.Contains(reason, "plan_mode") || !strings.Contains(reason, "strict") {
		t.Fatalf("an uncited chat entry under strict must be refused by plan_mode, got refused=%v %q", refused, reason)
	}
	if _, refused := at.AdmitManualEntryAt("MNQ", "close_long", nyMidday); refused {
		t.Fatal("a close is never admitted (position management)")
	}
}

// CTO #1 Q — the REAL chain ADMITS a chat entry in advisory mode (NY session,
// flat, no plan): every gate passes, and EntryGate's R:R leg (5) and stop-floor
// leg (6) ABSTAIN because a chat entry carries no stop or target. That
// abstention is today's contract, pinned here so it cannot change silently;
// a stop-less chat entry passing the 0B floor is recorded as OWED (not W0b).
func TestChatEntryIsAdmittedInAdvisoryModeLegs5And6Abstain(t *testing.T) {
	withMaintenanceDir(t)
	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"})
	at.id = "agent-door-advisory"
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan { return nil }})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) // 11:00 CT Tuesday
	if reason, refused := at.AdmitManualEntryAt("MNQ", "open_long", nyMidday); refused {
		t.Fatalf("advisory, flat, no plan, NY midday: the chat entry must be ADMITTED, got %q", reason)
	}
	if reason, refused := at.AdmitManualEntryAt("MNQ", "open_short", nyMidday); refused {
		t.Fatalf("the short door must be admitted the same way, got %q", reason)
	}
}
