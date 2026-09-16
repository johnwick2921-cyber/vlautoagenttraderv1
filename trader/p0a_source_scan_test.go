package trader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// P0-A — STARTUP-ASSERTION EQUIVALENT, BY CONSTRUCTION. The cross-trader hole
// was three process-global singletons + an unscoped lookup. The globals no
// longer exist (compiler-enforced), and these source-scan tests pin the
// remaining invariant so the hole cannot be re-drilled without a loud failure:
//
//  1. no production Go file may call the UNscoped GetLatestPlanForSession;
//  2. no production Go file may reference the removed global providers.
//
// A test that fails loudly here IS the startup assertion for the next run.

var p0aScanRoots = []string{"trader", "api", "kernel", "store"}

func scanProductionGo(t *testing.T, root string, fn func(path string, lines []string) error) {
	t.Helper()
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return fn(p, strings.Split(string(b), "\n"))
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
}

func TestP0AUnscopedLookupBannedInProduction(t *testing.T) {
	banned := []string{"GetLatestPlanForSession(", "ListVersions(", "ListRecent("}
	for _, root := range p0aScanRoots {
		scanProductionGo(t, root, func(p string, lines []string) error {
			for i, ln := range lines {
				if strings.Contains(p, "store/plan.go") {
					return nil // the definitions themselves live here
				}
				for _, b := range banned {
					if strings.Contains(ln, b) && !strings.Contains(ln, "ForTrader") {
						t.Errorf("%s:%d — UNscoped plan lookup in production: %s\n"+
							"Use the trader-scoped variant (…ForTraderSession / …ForTrader).", p, i+1, strings.TrimSpace(ln))
					}
				}
			}
			return nil
		})
	}
}

func TestP0AGlobalProvidersGone(t *testing.T) {
	banned := []string{"kernel.ActivePlanProvider", "kernel.SessionRegistryProvider", "kernel.PlanProximityKProvider"}
	for _, root := range p0aScanRoots {
		scanProductionGo(t, root, func(p string, lines []string) error {
			for i, ln := range lines {
				for _, b := range banned {
					if strings.Contains(ln, b) {
						t.Errorf("%s:%d — removed global provider referenced: %s\n"+
							"Register per trader with kernel.SetTraderPlanProviders(traderID, ...).", p, i+1, strings.TrimSpace(ln))
					}
				}
			}
			return nil
		})
	}
}

// TestP0ASyncOnceProviderInstallBanned — the original hole was ONE sync.Once
// installing providers for whoever arrived first. No production code may install
// a kernel provider inside a sync.Once anymore.
func TestP0ASyncOnceProviderInstallBanned(t *testing.T) {
	for _, root := range p0aScanRoots {
		scanProductionGo(t, root, func(p string, lines []string) error {
			for i, ln := range lines {
				if strings.Contains(ln, "Once.Do") && (strings.Contains(p, "provider") || strings.Contains(strings.ToLower(p), "plan")) {
					// Context-check the next lines for a provider install.
					for j := i; j < len(lines) && j <= i+6; j++ {
						if strings.Contains(lines[j], "SetTraderPlanProviders") {
							t.Errorf("%s:%d — provider installed inside a sync.Once (the P0-A hole). Install per trader, every cycle.", p, i+1)
						}
					}
				}
			}
			return nil
		})
	}
}

// TestP1RawRegistryFlagBanned — H8 residuals (P1): no API-layer production code
// may decide on the RAW registry flag (sess.Enabled). Enablement must route
// through the trader's SessionRunnable (the resolver's own internals and the
// sessionGateDecision fallback live in trader/, outside this scan's scope).
func TestP1RawRegistryFlagBanned(t *testing.T) {
	scanProductionGo(t, "api", func(p string, lines []string) error {
		for i, ln := range lines {
			if strings.Contains(ln, "sess.Enabled") {
				t.Errorf("%s:%d — raw registry flag decides a plan surface: %s\n"+
					"Resolve via the trader's SessionRunnable(sess), never sess.Enabled.", p, i+1, strings.TrimSpace(ln))
			}
		}
		return nil
	})
}
