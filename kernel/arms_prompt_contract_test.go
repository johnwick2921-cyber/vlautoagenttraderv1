// D1 / E2 — the ARMED block states the arm rules direction-neutrally, names the
// conditions that can actually be armed, and gives the entry TYPE per
// condition. A grep pin, because the failure mode is prose drifting away from
// the table the machine enforces.

package kernel

import (
	"strings"
	"testing"
)

func armedPromptBlock(t *testing.T) string {
	t.Helper()
	return plannerOutputContract(8, 3, true, true)
}

// The vocabulary must come from the SAME derived line the validator warns with.
func TestPromptCarriesTheArmableVocabulary(t *testing.T) {
	p := armedPromptBlock(t)
	line := ArmableConditionsLine(ResolvedConditionStatuses(nil, nil, ShadowConditionsEnv()))
	if !strings.Contains(p, line) {
		t.Fatalf("the prompt must embed the derived armable line verbatim.\nwant: %s", line)
	}
	// The entry TYPE must be visible per condition, or the model cannot choose.
	for _, want := range []string{"reclaim→stop_entry", "reject→limit"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt must name the entry type %q", want)
		}
	}
}

// The bias rule, stated without favouring a side.
func TestPromptStatesTheBiasCoherenceRule(t *testing.T) {
	p := armedPromptBlock(t)
	for _, want := range []string{
		"bias direction",
		"A long plan with no long arm is invalid",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt must state the bias-coherence rule (%q missing)", want)
		}
	}
	// Mirror wording: the rule must read for both sides, not just longs.
	if !strings.Contains(p, "short plan with no short arm") {
		t.Error("the rule must be stated for BOTH sides — a one-sided statement is the bug")
	}
}

// The old sentence named an exhaustive arm set of two conditions and omitted
// every other armable one, including the only long-capable play. It must be gone.
func TestPromptNoLongerNamesATwoConditionArmSet(t *testing.T) {
	p := armedPromptBlock(t)
	for _, stale := range []string{
		"every fvg_entry / reject scenario you author",
	} {
		if strings.Contains(p, stale) {
			t.Errorf("stale side-narrowing wording still present: %q", stale)
		}
	}
}

// fvg_entry is SHADOWED — the prompt must not urge the model to arm it.
func TestPromptDoesNotUrgeArmingAShadowedCondition(t *testing.T) {
	p := armedPromptBlock(t)
	if strings.Contains(p, "fvg_entry") && !strings.Contains(strings.ToLower(p), "shadowed") {
		t.Error("fvg_entry appears without the shadowed warning")
	}
}
