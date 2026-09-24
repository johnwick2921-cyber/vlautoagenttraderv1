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

// W1b E9 — FLIPPED DELIBERATELY. This was
// TestChatEntryIsAdmittedInAdvisoryModeLegs5And6Abstain: "the REAL chain
// ADMITS a chat entry in advisory mode (NY session, flat, no plan) ... EntryGate's
// R:R leg (5) and stop-floor leg (6) ABSTAIN because a chat entry carries no
// stop or target. That abstention is today's contract, pinned here so it cannot
// change silently". It was the defect: on a path that NEVER supplies a stop,
// leg 5/6's fail-open abstention is not "occasionally skipped", it is
// permanently OFF — and the NT8 broker then sent the LAST AI decision's SL/TP
// maps as this entry's bracket. CTO ruling (W1b): a chat/agent-door entry with
// no explicit stop is REFUSED, fail-closed, with a named reason.
func TestChatEntryWithNoStopIsRefused(t *testing.T) {
	withMaintenanceDir(t)
	futuresTape(t) // the live read succeeds: the refusal is the missing stop, not a missing price
	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"})
	at.id = "agent-door-advisory"
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan { return nil }})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) // 11:00 CT Tuesday
	for _, act := range []string{"open_long", "open_short"} {
		reason, refused := at.AdmitManualEntryAt("MNQ", act, nyMidday)
		if !refused || !strings.Contains(reason, "no explicit stop") || !strings.HasPrefix(reason, "entry_gate:") {
			t.Fatalf("%s: a stop-less chat entry must be REFUSED by name (fail-closed), got refused=%v %q", act, refused, reason)
		}
	}
	if _, refused := at.AdmitManualEntryAt("MNQ", "close_long", nyMidday); refused {
		t.Fatal("a close is never admitted (position management)")
	}
}

// W1b E9 — a chat entry that CARRIES its bracket is judged by EntryGate legs 5
// and 6 like any decision: a stop inside the ATR floor and a 1:1 target are
// refused, a sane bracket on the right side of live is admitted, and a
// bracket on the wrong side of live is refused before the gate.
func TestChatEntryBracketIsJudgedByLegs5And6(t *testing.T) {
	withMaintenanceDir(t)
	live := agentTape(t)
	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"})
	at.id = "agent-door-bracket"
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan { return nil }})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	atr := armSeamATR5m("MNQ")
	if atr <= 0 {
		t.Fatal("fixture: the tape must yield an ATR5m so leg 6 can judge")
	}
	floor := kernel.MinSLATRMult() * atr
	wide := floor + 10
	cases := []struct {
		name, act    string
		stop, target float64
		want         string // "" = admitted
	}{
		{"stop inside the ATR floor", "open_long", live - 0.25, live + 100*floor, "too close"},
		{"1:1 target", "open_long", live - wide, live + wide, "below floor"},
		{"short stop inside the floor", "open_short", live + 0.25, live - 100*floor, "too close"},
		{"long stop above live", "open_long", live + wide, live + 5*wide, "wrong side"},
		{"short target above live", "open_short", live + wide, live + 5*wide, "wrong side"},
		{"stop but no target", "open_long", live - wide, 0, "no explicit target"},
		{"sane long", "open_long", live - wide, live + 4*wide, ""},
		{"sane short", "open_short", live + wide, live - 4*wide, ""},
	}
	for _, c := range cases {
		reason, refused := at.AdmitManualEntryBracketAt("MNQ", c.act, c.stop, c.target, nyMidday)
		switch {
		case c.want == "" && refused:
			t.Errorf("%s: a sane bracket must be ADMITTED, got %q", c.name, reason)
		case c.want != "" && (!refused || !strings.Contains(reason, c.want)):
			t.Errorf("%s: want a refusal naming %q, got refused=%v %q", c.name, c.want, refused, reason)
		}
	}
}
