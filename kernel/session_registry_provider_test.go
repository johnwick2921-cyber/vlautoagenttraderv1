package kernel

import "testing"

// H7 — the executor prompt path read DefaultSessionRegistry() directly, so an
// admin-edited persisted registry would have been silently ignored by the one
// surface that renders the plan to the AI. The seam must default to the shipped
// registry and honor an installed provider.

func TestResolvedSessionRegistryFallbacks(t *testing.T) {
	prev := SessionRegistryProvider
	defer func() { SessionRegistryProvider = prev }()

	SessionRegistryProvider = nil
	if got := resolvedSessionRegistry(); got.Sessions == nil || len(got.Sessions) != 3 {
		t.Fatalf("nil provider must yield the shipped 3-session registry, got %d sessions", len(got.Sessions))
	}

	// An installed provider wins — even a registry with a single custom session.
	SessionRegistryProvider = func() SessionRegistry {
		return SessionRegistry{Sessions: []SessionDef{{Name: "CUSTOM", WindowStartCT: "09:00", WindowEndCT: "10:00", Enabled: true}}}
	}
	got := resolvedSessionRegistry()
	if len(got.Sessions) != 1 || got.Sessions[0].Name != "CUSTOM" {
		t.Fatalf("provider registry must win, got %+v", got.Sessions)
	}

	// A broken provider (zero sessions) falls back — the executor is never left
	// registry-less.
	SessionRegistryProvider = func() SessionRegistry { return SessionRegistry{} }
	if got := resolvedSessionRegistry(); len(got.Sessions) != 3 {
		t.Fatalf("empty provider registry must fall back to the default, got %d sessions", len(got.Sessions))
	}
}
