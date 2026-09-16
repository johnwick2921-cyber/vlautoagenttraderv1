package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W8 — the gates read the admin registry from system_config (the dead wire: every
// gate called kernel.DefaultSessionRegistry() directly). loadStoredRegistry is the
// resolver; it fail-safes to the default on empty/malformed input.
func TestW8LoadStoredRegistry(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// empty → the shipped default (NY enabled, ASIA/LONDON off).
	reg := loadStoredRegistry(st)
	ny, ok := reg.SessionByName(kernel.SessionNY)
	if !ok || !ny.Enabled {
		t.Fatal("empty config must yield the default registry (NY enabled)")
	}

	// a stored edit (disable NY) is reflected.
	edited := kernel.DefaultSessionRegistry()
	for i := range edited.Sessions {
		if edited.Sessions[i].Name == kernel.SessionNY {
			edited.Sessions[i].Enabled = false
		}
	}
	raw, _ := edited.Marshal()
	if err := st.SetSystemConfig(kernel.SessionRegistryConfigKey, raw); err != nil {
		t.Fatalf("save: %v", err)
	}
	reg = loadStoredRegistry(st)
	ny, _ = reg.SessionByName(kernel.SessionNY)
	if ny.Enabled {
		t.Fatal("stored edit (NY disabled) must be honored")
	}

	// malformed → fail-safe to the default (a junk edit can't disable the gates).
	if err := st.SetSystemConfig(kernel.SessionRegistryConfigKey, "{not valid json"); err != nil {
		t.Fatalf("save junk: %v", err)
	}
	reg = loadStoredRegistry(st)
	ny, ok = reg.SessionByName(kernel.SessionNY)
	if !ok || !ny.Enabled {
		t.Fatal("malformed config must fail-safe to the default (NY enabled)")
	}
}

// W8 — sessionRegistry caches per CME session-day: an edit is honored by the NEXT
// session-day's gates, never mid-session (a running session's windows never move).
func TestW8RegistryCachedPerSessionDay(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	at := mkTrader("ninjatrader", boolp(true), "5m")
	at.store = st

	day1 := time.Now()
	// warm the cache on day1 (empty → default, NY enabled).
	if r := at.sessionRegistry(day1); func() bool { ny, _ := r.SessionByName(kernel.SessionNY); return !ny.Enabled }() {
		t.Fatal("day1 must start from the default (NY enabled)")
	}

	// edit mid-day1: disable NY.
	edited := kernel.DefaultSessionRegistry()
	for i := range edited.Sessions {
		if edited.Sessions[i].Name == kernel.SessionNY {
			edited.Sessions[i].Enabled = false
		}
	}
	raw, _ := edited.Marshal()
	_ = st.SetSystemConfig(kernel.SessionRegistryConfigKey, raw)

	// same session-day → still the cached default (NOT honored mid-session).
	r := at.sessionRegistry(day1)
	if ny, _ := r.SessionByName(kernel.SessionNY); !ny.Enabled {
		t.Fatal("an edit must NOT take effect mid-session-day (cache holds)")
	}

	// next session-day → the edit is honored.
	day2 := day1.Add(48 * time.Hour)
	r = at.sessionRegistry(day2)
	if ny, _ := r.SessionByName(kernel.SessionNY); ny.Enabled {
		t.Fatal("the edit must be honored on the next session-day")
	}
}
