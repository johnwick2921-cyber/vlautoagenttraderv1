package kernel

import (
	"os"
	"strings"
	"testing"

	"nofx/store"
)

const futuresKeyLevelsGolden = "testdata/futures_mnq_keylevels.golden"

// sampleKeyLevelsBlock is a fixed, representative KEY LEVELS block (the exact
// shape RenderKeyLevelsBlock emits) so the enabled-state golden is deterministic.
const sampleKeyLevelsBlock = "KEY LEVELS (map, nearest-first; price 21500.00):\n" +
	"  21500.00  PDC            A  fresh     +0.0\n" +
	"  21520.00  PDH            A  fresh·x2  +20.0\n" +
	"  21480.00  RTH-L          B  fresh     -20.0\n" +
	"Anchor: react AT these levels (grade A>B>C); do not chase price between them."

// keyLevelsEnabledEngine is the empty-box futures engine with day_plan ENABLED
// and a fixed KEY LEVELS block set.
func keyLevelsEnabledEngine(line string) *StrategyEngine {
	e := emptyBoxFuturesEngine()
	e.config.DayPlan = &store.DayPlanConfig{PlanEnabled: true, MaxLevels: 8}
	e.SetKeyLevelsContext(line)
	return e
}

// TestFuturesKeyLevelsInjection proves P1.7 gating: the KEY LEVELS block appears
// ONLY when day_plan is enabled AND a non-empty block was threaded in. Any other
// state injects nothing — which is what keeps every existing futures golden
// byte-identical (the default has no day_plan).
func TestFuturesKeyLevelsInjection(t *testing.T) {
	// ON (day_plan enabled) + non-empty → block + anchor appear.
	on := keyLevelsEnabledEngine(sampleKeyLevelsBlock).BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if !strings.Contains(on, "KEY LEVELS (map") {
		t.Error("KEY LEVELS block missing when day_plan is enabled")
	}
	if !strings.Contains(on, "Anchor: react AT these levels") {
		t.Error("anchor line missing when day_plan is enabled")
	}

	// OFF (no day_plan) even with a context set → nothing injected.
	off := emptyBoxFuturesEngine()
	off.SetKeyLevelsContext(sampleKeyLevelsBlock)
	if strings.Contains(off.BuildFuturesDecisionSystemPrompt("MNQ", 50000), "KEY LEVELS (map") {
		t.Error("KEY LEVELS leaked while day_plan is absent (golden would break)")
	}

	// plan_enabled=false explicitly → nothing injected.
	disabled := emptyBoxFuturesEngine()
	disabled.config.DayPlan = &store.DayPlanConfig{PlanEnabled: false}
	disabled.SetKeyLevelsContext(sampleKeyLevelsBlock)
	if strings.Contains(disabled.BuildFuturesDecisionSystemPrompt("MNQ", 50000), "KEY LEVELS (map") {
		t.Error("KEY LEVELS leaked while plan_enabled=false")
	}

	// ON but empty block (warming / no bars) → nothing injected.
	onEmpty := keyLevelsEnabledEngine("")
	if strings.Contains(onEmpty.BuildFuturesDecisionSystemPrompt("MNQ", 50000), "KEY LEVELS (map") {
		t.Error("KEY LEVELS appeared with an empty block")
	}
}

// TestFuturesKeyLevelsGolden captures the ENABLED-state live futures prompt (the
// deliberate new golden). The DISABLED state remains futures_mnq_empty.golden,
// which is proven unchanged by TestFuturesPromptEmptyBoxesByteIdentical.
// Recapture: UPDATE_GOLDEN=1 go test ./kernel -run FuturesKeyLevelsGolden
func TestFuturesKeyLevelsGolden(t *testing.T) {
	got := keyLevelsEnabledEngine(sampleKeyLevelsBlock).BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(futuresKeyLevelsGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Skip("golden captured/updated")
	}
	want, err := os.ReadFile(futuresKeyLevelsGolden)
	if err != nil {
		t.Fatalf("read golden (capture with UPDATE_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		t.Fatalf("enabled-state KEY LEVELS futures prompt DIFFERS from golden")
	}
}
