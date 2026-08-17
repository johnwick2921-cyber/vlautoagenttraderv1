package kernel

import "testing"

// H7 — the executor prompt path read DefaultSessionRegistry() directly, so an
// admin-edited persisted registry would have been silently ignored by the one
// surface that renders the plan to the AI. The seam must default to the shipped
// registry and honor an installed provider — and P0-A: the resolution is
// PER-TRADER, so one trader can never receive another's registry or plan.

func TestResolvedSessionRegistryFallbacks(t *testing.T) {
	// No provider → the shipped 3-session registry.
	if got := ResolvedSessionRegistryFor("t1"); len(got.Sessions) != 3 {
		t.Fatalf("no provider must yield the shipped 3-session registry, got %d sessions", len(got.Sessions))
	}
	// Empty trader id → nothing, ever (the cross-trader hole by another name).
	if _, ok := TraderPlanProvidersFor(""); ok {
		t.Fatalf("empty trader id must never resolve a provider")
	}

	// An installed provider wins for ITS trader only.
	SetTraderPlanProviders("t1", TraderPlanProviders{
		SessionRegistry: func() SessionRegistry {
			return SessionRegistry{Sessions: []SessionDef{{Name: "CUSTOM", WindowStartCT: "09:00", WindowEndCT: "10:00", Enabled: true}}}
		},
	})
	defer SetTraderPlanProviders("t1", TraderPlanProviders{})

	got := ResolvedSessionRegistryFor("t1")
	if len(got.Sessions) != 1 || got.Sessions[0].Name != "CUSTOM" {
		t.Fatalf("provider registry must win for its trader, got %+v", got.Sessions)
	}
	// A DIFFERENT trader never receives t1's registry.
	if other := ResolvedSessionRegistryFor("t2"); len(other.Sessions) != 3 {
		t.Fatalf("t2 must never see t1's registry, got %d sessions", len(other.Sessions))
	}

	// A broken provider (zero sessions) falls back — the executor is never left
	// registry-less.
	SetTraderPlanProviders("t1", TraderPlanProviders{
		SessionRegistry: func() SessionRegistry { return SessionRegistry{} },
	})
	if got := ResolvedSessionRegistryFor("t1"); len(got.Sessions) != 3 {
		t.Fatalf("empty provider registry must fall back to the default, got %d sessions", len(got.Sessions))
	}
}

// P0-A — the active-plan resolution is per-trader too: t2 must never receive
// t1's plan.
func TestActivePlanForIsPerTrader(t *testing.T) {
	p1 := &ActivePlan{Doc: PlanDoc{Bias: PlanBias{Direction: "long"}}, Session: "NY", Version: 3}
	SetTraderPlanProviders("t1", TraderPlanProviders{
		ActivePlan: func(string) *ActivePlan { return p1 },
	})
	defer SetTraderPlanProviders("t1", TraderPlanProviders{})

	if got := ActivePlanFor("t1", "MNQ"); got == nil || got.Version != 3 {
		t.Fatalf("t1 must receive its own plan, got %+v", got)
	}
	if got := ActivePlanFor("t2", "MNQ"); got != nil {
		t.Fatalf("t2 must NEVER receive t1's plan, got %+v", got)
	}
	if got := ActivePlanFor("", "MNQ"); got != nil {
		t.Fatalf("empty trader id must never receive a plan, got %+v", got)
	}
}
