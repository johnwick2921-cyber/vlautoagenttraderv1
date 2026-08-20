package kernel

import (
	"strings"
	"testing"
)

func validBaseDoc() *PlanDoc {
	return &PlanDoc{
		Reasoning:      "r",
		Bias:           PlanBias{Direction: "short", FlipCondition: "15m close above 29648.25 flips long"},
		Levels:         []PlanLevel{{Price: 29648.25, Label: "OR-L", Grade: "A"}, {Price: 29441, Label: "PDL", Grade: "A"}},
		Scenarios:      []PlanScenario{{ID: "S1", Trigger: "reject at 29648.25", Condition: "reject", Direction: "short", TargetChain: []float64{29441}, Invalid: "15m close above 29648.25", Quality: "B"}},
		NoTrade:        []string{"lunch"},
		DeathCondition: "15m close below 29441.00 kills the plan",
		DeathStructured: &PlanCondition{Price: 29441, Side: "below", Rule: "15m_close"},
		FlipStructured:  &PlanCondition{Price: 29648.25, Side: "above", Rule: "15m_close", FlipTo: "long"},
	}
}

// A3 (F5) — object↔prose agreement is validated now.
func TestDeathFlipProseCrossCheck(t *testing.T) {
	d := validBaseDoc()
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("valid doc must pass: %v", err)
	}
	d.DeathStructured.Price = 29300 // no such number in the prose
	if err := ValidatePlanDocWithCaps(d, 8, 3); err == nil || !strings.Contains(err.Error(), "death{") {
		t.Fatalf("death object/prose mismatch must fail validation (got %v)", err)
	}
	d = validBaseDoc()
	d.FlipStructured.Price = 29999
	if err := ValidatePlanDocWithCaps(d, 8, 3); err == nil || !strings.Contains(err.Error(), "flip{") {
		t.Fatalf("flip object/prose mismatch must fail validation (got %v)", err)
	}
}

// A5 (F11) — the id format is a contract.
func TestScenarioIDFormatEnforced(t *testing.T) {
	d := validBaseDoc()
	d.Scenarios[0].ID = "scenario-one"
	if err := ValidatePlanDocWithCaps(d, 8, 3); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("non-S# id must fail (got %v)", err)
	}
	d.Scenarios[0].ID = "S2"
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("S2 must pass: %v", err)
	}
}
