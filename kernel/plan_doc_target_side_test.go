package kernel

import (
	"strings"
	"testing"
)

// TestTargetChainWrongSideRefused (WAVE 1a-plan P5, #190) — a target_chain
// entry on the wrong side of the scenario's own entry is refused at write.
func TestTargetChainWrongSideRefused(t *testing.T) {
	facts := asiaV2Facts() // price 29764, DATR 150 → proximity band 225

	longDoc := func(targets []float64) *PlanDoc {
		return &PlanDoc{
			Reasoning:      "r",
			Bias:           PlanBias{Direction: "long"},
			DeathCondition: "flat",
			Levels:         []PlanLevel{{Price: 29700, Label: "PDL", Grade: "A"}, {Price: 29800, Label: "PDH", Grade: "A"}},
			Scenarios: []PlanScenario{{ID: "S1", Condition: "reject", Direction: "long", Quality: "A",
				Arm:         &PlanArmSpec{Enabled: true, Entry: 29764, Stop: 29700, Target: 29800},
				TargetChain: targets}},
		}
	}
	shortDoc := func(targets []float64) *PlanDoc {
		return &PlanDoc{
			Reasoning:      "r",
			Bias:           PlanBias{Direction: "short"},
			DeathCondition: "flat",
			Levels:         []PlanLevel{{Price: 29700, Label: "PDL", Grade: "A"}, {Price: 29800, Label: "PDH", Grade: "A"}},
			Scenarios: []PlanScenario{{ID: "S1", Condition: "reject", Direction: "short", Quality: "A",
				Arm:         &PlanArmSpec{Enabled: true, Entry: 29764, Stop: 29800, Target: 29700},
				TargetChain: targets}},
		}
	}

	if err := ValidatePlanDocWithFactsMachine(longDoc([]float64{29700}), facts, nil, 8, 3); err == nil || !strings.Contains(err.Error(), "WRONG SIDE of entry") {
		t.Fatalf("a long target BELOW its entry must be refused, got %v", err)
	}
	if err := ValidatePlanDocWithFactsMachine(shortDoc([]float64{29800}), facts, nil, 8, 3); err == nil || !strings.Contains(err.Error(), "WRONG SIDE of entry") {
		t.Fatalf("a short target ABOVE its entry must be refused, got %v", err)
	}
	// Correct sides pass (and the proximity band still guards).
	if err := ValidatePlanDocWithFactsMachine(longDoc([]float64{29800}), facts, nil, 8, 3); err != nil {
		t.Fatalf("a long target above entry must pass, got %v", err)
	}
	if err := ValidatePlanDocWithFactsMachine(shortDoc([]float64{29700}), facts, nil, 8, 3); err != nil {
		t.Fatalf("a short target below entry must pass, got %v", err)
	}
}
