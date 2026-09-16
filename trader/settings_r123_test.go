package trader

import (
	"os"
	"strings"
	"testing"

	"nofx/store"
)

// ── SETTINGS INTEGRITY R1/R2/R3 (owner rulings 2026-09-03) ───────────────────

// E2 — R1: ONE R:R FLOOR. Both paths read the SAME Studio value, and changing
// it in the config moves BOTH. ARM_MIN_RR is gone.
func TestR1OneFloorBothPaths(t *testing.T) {
	at := &AutoTrader{}
	cfg := &store.StrategyConfig{}
	cfg.RiskControl.MinRiskRewardRatio = 2.0
	if got := at.armMinRRFor(cfg); got != 2.0 {
		t.Fatalf("arm seam must read the Studio value: %.2f", got)
	}
	// Move it in Studio → BOTH move, because there is one resolver.
	cfg.RiskControl.MinRiskRewardRatio = 4.5
	if got := at.armMinRRFor(cfg); got != 4.5 {
		t.Errorf("a Studio change must move the arm floor: %.2f", got)
	}
	if got := resolvedMinRR(cfg); got != 4.5 {
		t.Errorf("the decision path shares the resolver: %.2f", got)
	}
	// No config → the SCHEMA's safe default, not a second hardcoded opinion.
	if got := resolvedMinRR(nil); got != store.SafeDefaultMinRiskReward {
		t.Errorf("no config must fall to the schema default %.1f, got %.2f", store.SafeDefaultMinRiskReward, got)
	}
	// The env var is deleted: setting it must change nothing.
	t.Setenv("ARM_MIN_RR", "9.9")
	if got := at.armMinRRFor(cfg); got != 4.5 {
		t.Errorf("ARM_MIN_RR is DELETED — it must not resurrect a second floor, got %.2f", got)
	}
}

// E3 — R2: the per-session plan_mode override reaches the ARM SEAM.
func TestR2SessionPlanModeReachesTheArmSeam(t *testing.T) {
	b, err := readFile("armed_executor.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if strings.Contains(src, `at.planModeFor("")`) {
		t.Error(`the arm seam still calls planModeFor("") — a per-session override is dropped`)
	}
	if !strings.Contains(src, "at.planModeFor(session)") {
		t.Error("the arm seam must resolve plan_mode for the SESSION")
	}
}

// E4 — R3: htf_veto=false stops the ARM chain too. One switch, both consumers.
func TestR3HtfVetoOffStopsTheArmChain(t *testing.T) {
	b, err := readFile("armed_executor.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	i := strings.Index(src, "HTFVetoVerdict(snap")
	if i < 0 {
		t.Fatal("the arm-chain veto call moved — re-locate before trusting this pin")
	}
	window := src[max0(i-400):i]
	if !strings.Contains(window, "cfg.HTFVetoEnabled()") {
		t.Error("the arm-chain veto must be gated on the SAME switch as the decision path (regime.htf_veto)")
	}
	// And the switch itself must exist and default ON.
	cfg := &store.StrategyConfig{}
	if !cfg.HTFVetoEnabled() {
		t.Error("htf_veto defaults ON — turning it OFF is the owner's action, not the default")
	}
}

func readFile(n string) ([]byte, error) { return os.ReadFile(n) }

func max0(i int) int {
	if i < 0 {
		return 0
	}
	return i
}
