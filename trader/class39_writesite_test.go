package trader

import (
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// CLASS 39 — the write-site half: the ⚖ WARN fires, the recorded counter bumps,
// and a still-invalid single arm rejects with the ORIGINAL reason that the
// model's retry then reads (D2, D3, D5).

// class39LegsPlanJSON — a `reject` (non-sweep) long fade at PWL 15480 whose arm
// carries ONE leg that mirrors the top-level arm (the row-69 S1 shape, on the
// fixture universe validTraderPlanJSON already passes with).
func class39LegsPlanJSON(target string) string {
	return `{
  "reasoning": "Balance below PDH; fade the PWL touch.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "2x5m < 15480"},
  "levels": [
    {"price": 15480, "label": "PWL", "grade": "A", "instruction": "fade"},
    {"price": 15520, "label": "RN 15525", "grade": "B", "instruction": "fade"},
    {"price": 15575, "label": "RN 15575", "grade": "B", "instruction": "fade"},
    {"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade"},
    {"price": 15650, "label": "RN 15650", "grade": "B", "instruction": "fade"},
    {"price": 15700, "label": "RN 15700", "grade": "B", "instruction": "fade"}
  ],
  "scenarios": [{"id": "S1", "trigger": "fade the touch at 15480 PWL", "condition": "reject", "direction": "long",
    "target_chain": [15550, 15620], "invalid": "5m close below 15470", "quality": "A",
    "confirm": {"rule": "touch", "ref_price": 15480, "side": "below"},
    "economics":{"entry_zone":[15480,15480],"first_obstacle":{"price":15550,"level":"fixture reference","family":"reference","response":"pass_through"},"r_to_obstacle":7,"r_to_arm_target":7},
    "arm": {"enabled": true, "entry": 15480, "stop": 15470, "target": ` + target + `, "wait_confirm": true,
      "legs": [{"entry": 15480, "stop": 15470, "target": ` + target + `, "size": 1, "wait_confirm": false, "rule": "touch"}]}}],
  "no_trade": ["first 5m"],
  "death_condition": "acceptance above 15620",
  "death": {"price": 15620, "side": "above", "rule": "2x5m"},
  "flip": {"price": 15480, "side": "below", "rule": "2x5m", "flip_to": "short"},
  "day_type": "balance"
}`
}

// D2/D5 — a legged non-sweep arm with a VALID single arm lands on attempt 1;
// the recorded counter reads 1 afterwards.
func TestClass39WriteSiteNormalizesAndRecords(t *testing.T) {
	at := plannerTestTrader(t)
	if n := store.ArmsNormalizedCount(at.store); n != 0 {
		t.Fatalf("fresh counter = %d", n)
	}
	facts := kernel.PlanFacts{Price: 15550, DATR: 300}
	machine := map[float64]string{15480: "PWL", 15700: "RN 15700"}
	calls := 0
	ver, lc, err := at.runPlannerReadCoreWithFactsGrades("NY", "2026-09-01", "owner_reset",
		"deepseek-v4-pro", "hashC39a", "", "", "", "PROMPT", facts, nil, machine, nil, true,
		func(userPrompt string) (string, error) {
			calls++
			return class39LegsPlanJSON("15550"), nil // valid single arm: stop 15470 < entry 15480 < target 15550
		})
	if err != nil || lc != "active" {
		t.Fatalf("legged non-sweep arm with a valid single arm must land on attempt 1: ver=%d lc=%q err=%v", ver, lc, err)
	}
	if calls != 1 {
		t.Fatalf("must NOT burn a retry — %d calls", calls)
	}
	if n := store.ArmsNormalizedCount(at.store); n != 1 {
		t.Fatalf("recorded counter arms_normalized_class39 = %d, want 1", n)
	}
}

// D3 / E3 — a legged non-sweep arm whose single arm is INVALID (long with the
// target below the entry) rejects with the ORIGINAL reason; attempt 2's repair
// prompt carries that reason verbatim and no normalization text; the counter
// does not move.
func TestClass39RejectPathCarriesOriginalReason(t *testing.T) {
	at := plannerTestTrader(t)
	facts := kernel.PlanFacts{Price: 15550, DATR: 300}
	machine := map[float64]string{15480: "PWL", 15700: "RN 15700"}
	prompts := []string{}
	_, lc, err := at.runPlannerReadCoreWithFactsGrades("NY", "2026-09-01", "owner_reset",
		"deepseek-v4-pro", "hashC39b", "", "", "", "PROMPT", facts, nil, machine, nil, true,
		func(userPrompt string) (string, error) {
			prompts = append(prompts, userPrompt)
			if len(prompts) == 1 {
				return class39LegsPlanJSON("15475"), nil // target BELOW entry on a long → the single arm is invalid too
			}
			return validTraderPlanJSON, nil
		})
	if err != nil || lc != "active" {
		t.Fatalf("attempt 2 with a clean plan must land: lc=%q err=%v", lc, err)
	}
	if len(prompts) < 2 {
		t.Fatalf("attempt 1 must have been rejected; got %d call(s)", len(prompts))
	}
	retry := prompts[1]
	if !strings.Contains(retry, "arm_legs_sweep_reclaim_only") {
		t.Fatalf("the retry must carry the ORIGINAL validator reason (arm_legs_sweep_reclaim_only), got:\n%s", retry)
	}
	if strings.Contains(strings.ToLower(retry), "normaliz") {
		t.Fatalf("the retry must never mention normalization:\n%s", retry)
	}
	if n := store.ArmsNormalizedCount(at.store); n != 0 {
		t.Fatalf("a rejected doc must not bump the counter, got %d", n)
	}
}
