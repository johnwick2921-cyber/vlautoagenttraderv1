package trader

import (
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// 6.4 (ruling B) — the clamps stay enforced with the deprecated toggles stored
// FALSE (the owner's live values): the toggle can not disable venue safety.
func TestSizeCapsIgnoreDeprecatedToggles(t *testing.T) {
	off := false
	rc := store.RiskControlConfig{
		MaxContractsEnabled: &off,
		NotionalCapEnabled:  &off,
		// no explicit values → the researched defaults
	}
	if got := kernel.ResolveMaxContracts(rc.MaxContractsPerOrder, 2); got != 2 {
		t.Errorf("contracts clamp = %d with toggle false, want 2 (always-on)", got)
	}
	if got := kernel.ResolveNotionalLeverage(rc.MaxNotionalLeverage, 20); got != 20 {
		t.Errorf("notional cap = %.0f with toggle false, want 20 (always-on)", got)
	}
	// Explicit per-strategy values still win (the VALUE is live, only the
	// toggle was dead).
	rc.MaxContractsPerOrder = 1
	rc.MaxNotionalLeverage = 10
	if got := kernel.ResolveMaxContracts(rc.MaxContractsPerOrder, 2); got != 1 {
		t.Errorf("explicit contracts value ignored (got %d)", got)
	}
	if got := kernel.ResolveNotionalLeverage(rc.MaxNotionalLeverage, 20); got != 10 {
		t.Errorf("explicit notional value ignored (got %.0f)", got)
	}
}
