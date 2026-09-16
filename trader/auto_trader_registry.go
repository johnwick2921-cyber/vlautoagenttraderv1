package trader

import (
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W8 — ADMIN SESSION-REGISTRY loader (the audit's dead wire: every gate called
// kernel.DefaultSessionRegistry() directly — the admin registry stored in
// system_config was never read; see the removed TODO(P4) markers). The registry is
// GLOBAL (one row), edited via GET/POST /api/plan/session-registry.

// loadStoredRegistry reads the admin registry from system_config (under the
// canonical kernel.SessionRegistryConfigKey), falling back to the shipped default.
// Empty key OR a parse error → the default (fail-safe: the gates are NEVER left
// registry-less, and a malformed edit can't disable trading).
func loadStoredRegistry(st *store.Store) kernel.SessionRegistry {
	if st == nil {
		return kernel.DefaultSessionRegistry()
	}
	raw, _ := st.GetSystemConfig(kernel.SessionRegistryConfigKey)
	reg, _ := kernel.LoadSessionRegistry(raw) // returns the default on empty/malformed
	return reg
}

// sessionRegistry returns THIS trader's effective session registry: the admin
// registry from system_config, cached and refreshed once per CME session-day. An
// edit is therefore honored by the next session-day's gates, never mid-session.
// Fail-safe to the shipped default.
func (at *AutoTrader) sessionRegistry(now time.Time) kernel.SessionRegistry {
	dayKey := kernel.CMESessionDayKey(now)
	at.regMu.Lock()
	defer at.regMu.Unlock()
	if at.regCacheDay != dayKey || at.regCache.Sessions == nil {
		at.regCache = loadStoredRegistry(at.store)
		at.regCacheDay = dayKey
	}
	return at.regCache
}
