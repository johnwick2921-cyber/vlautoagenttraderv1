package trader

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// H7 — prove the SEAM end to end: the executor prompt's level assembly reads the
// PERSISTED admin registry via the provider the trader installs, not the
// hardcoded default. A registry edit (LONDON enabled) must reach the kernel's
// resolvedSessionRegistry() through snapshotSessionProfiles' wiring.
func TestH7SessionRegistryProviderWired(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	// Admin edits the persisted registry: LONDON enabled.
	reg := kernel.SessionRegistry{Sessions: []kernel.SessionDef{
		{Name: kernel.SessionLondon, WindowStartCT: "02:00", WindowEndCT: "08:30", ReadCT: "01:55", FlatCT: "08:30", Enabled: true},
	}}
	blob, _ := json.Marshal(reg)
	if err := st.SetSystemConfig(kernel.SessionRegistryConfigKey, string(blob)); err != nil {
		t.Fatalf("set registry: %v", err)
	}

	prevBars := market.FuturesBarsProvider
	prevReg := kernel.SessionRegistryProvider
	prevK := kernel.PlanProximityKProvider
	t.Cleanup(func() {
		market.FuturesBarsProvider = prevBars
		kernel.SessionRegistryProvider = prevReg
		kernel.PlanProximityKProvider = prevK
	})
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return nil }

	at := &AutoTrader{
		id: "t1", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{
			DayPlan: &store.DayPlanConfig{PlanEnabled: true},
		}},
	}
	at.snapshotSessionProfiles() // the sync.Once that installs every kernel provider

	if kernel.SessionRegistryProvider == nil {
		t.Fatalf("SessionRegistryProvider was never installed")
	}
	got := kernel.SessionRegistryProvider()
	var london *kernel.SessionDef
	for i := range got.Sessions {
		if got.Sessions[i].Name == kernel.SessionLondon {
			london = &got.Sessions[i]
		}
	}
	if london == nil || !london.Enabled {
		t.Fatalf("the persisted admin registry (LONDON enabled) must reach the kernel executor path, got %+v", got.Sessions)
	}
	if kernel.PlanProximityKProvider == nil {
		t.Fatalf("PlanProximityKProvider was never installed alongside")
	}
	// The kernel-side consumer (resolvedSessionRegistry) honors the provider —
	// proven in kernel/session_registry_provider_test.go.
}
