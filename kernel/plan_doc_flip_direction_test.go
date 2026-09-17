package kernel

import (
	"strings"
	"testing"
)

// W-FLIP-DIRECTION (2026-09-17) — A FLIP THAT POINTS THE WRONG WAY.
//
// Evidence [A]: plan 2026-09-17:LONDON v3 on the live machine carried
// bias.direction="short" with flip={price 29474.90, side "below", rule "2x5m",
// flip_to "long"} and death={29604.25 above}. A short bias can only flip to
// long on a close ABOVE a level; "below → long" is inverted, so the overnight
// rally could never flip the plan. The validator checked that the flip NUMBER
// appeared in the prose and never asked which way the flip pointed. These
// cases run at the production call site (ValidatePlanDocWithCaps).

// validLongBaseDoc mirrors validBaseDoc for a LONG bias: flips short on a
// close BELOW the low, dies on a close ABOVE the high.
func validLongBaseDoc() *PlanDoc {
	return &PlanDoc{
		Reasoning:       "r",
		Bias:            PlanBias{Direction: "long", FlipCondition: "5m close below 29441.00 flips short"},
		Levels:          []PlanLevel{{Price: 29648.25, Label: "OR-H", Grade: "A"}, {Price: 29441, Label: "PDL", Grade: "A"}},
		Scenarios:       []PlanScenario{{ID: "S1", Trigger: "reject at 29441.00", Condition: "reject", Direction: "long", TargetChain: []float64{29648.25}, Invalid: "5m close below 29441.00", Quality: "B"}},
		NoTrade:         []string{"lunch"},
		DeathCondition:  "5m close above 29648.25 kills the plan",
		DeathStructured: &PlanCondition{Price: 29648.25, Side: "above", Rule: "5m_close"},
		FlipStructured:  &PlanCondition{Price: 29441, Side: "below", Rule: "5m_close", FlipTo: "short"},
	}
}

func TestFlipSideMustOpposeBias_ShortBelowToLongRejected(t *testing.T) {
	d := validBaseDoc()             // short bias
	d.FlipStructured.Side = "below" // the LONDON v3 shape
	err := ValidatePlanDocWithCaps(d, 8, 3)
	if err == nil || !strings.Contains(err.Error(), "contradicts bias short") {
		t.Fatalf("short bias + flip below → long must be REJECTED with the direction error (got %v)", err)
	}
	if !strings.Contains(err.Error(), "close above the line") {
		t.Fatalf("the rejection must name the side the model should have written (got %v)", err)
	}
}

func TestFlipSideMustOpposeBias_LongAboveToShortRejected(t *testing.T) {
	d := validLongBaseDoc()
	d.FlipStructured.Side = "above"
	err := ValidatePlanDocWithCaps(d, 8, 3)
	if err == nil || !strings.Contains(err.Error(), "contradicts bias long") {
		t.Fatalf("long bias + flip above → short must be REJECTED with the direction error (got %v)", err)
	}
	if !strings.Contains(err.Error(), "close below the line") {
		t.Fatalf("the rejection must name the side the model should have written (got %v)", err)
	}
}

func TestFlipSideMustOpposeBias_ShortAboveToLongAccepted(t *testing.T) {
	d := validBaseDoc()
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("short bias + flip above → long is the correct shape and must pass: %v", err)
	}
}

func TestFlipSideMustOpposeBias_LongBelowToShortAccepted(t *testing.T) {
	d := validLongBaseDoc()
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("long bias + flip below → short is the correct shape and must pass: %v", err)
	}
}

func TestFlipSideMustOpposeBias_EmptyFlipToInferred(t *testing.T) {
	// flip_to empty → inferred as the opposite of the bias for this check.
	d := validBaseDoc()
	d.FlipStructured.FlipTo = ""
	d.FlipStructured.Side = "below"
	err := ValidatePlanDocWithCaps(d, 8, 3)
	if err == nil || !strings.Contains(err.Error(), "contradicts bias short") {
		t.Fatalf("short bias + flip below with empty flip_to must be REJECTED (inferred long) (got %v)", err)
	}
	if !strings.Contains(err.Error(), "→ long}") {
		t.Fatalf("the rejection must show the INFERRED flip_to (got %v)", err)
	}
	d = validBaseDoc()
	d.FlipStructured.FlipTo = ""
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("short bias + flip above with empty flip_to must pass: %v", err)
	}
}

func TestFlipSideMustOpposeBias_NoFlipUnchanged(t *testing.T) {
	d := validBaseDoc()
	d.FlipStructured = nil
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("a plan with no structured flip must be unchanged by the direction check: %v", err)
	}
	d = validLongBaseDoc()
	d.FlipStructured = nil
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("a long plan with no structured flip must be unchanged by the direction check: %v", err)
	}
}

// The death condition is a SEPARATE question: the direction check must never
// reach it. A death that points the same way as the flip is the existing
// preempt WARN's business, not a schema reject.
func TestFlipSideMustOpposeBias_DeathUntouched(t *testing.T) {
	d := validBaseDoc()
	d.DeathStructured.Side = "above" // same side as the flip, still a valid schema
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("death side must not be judged by the flip-direction check: %v", err)
	}
}

// Class-38 style: the law the validator judges by must reach the model, both
// in the prompt contract (registry + rendered prompt) and in the repair
// excerpt routed from the new error text.
func TestFlipDirectionLawReachesTheModel(t *testing.T) {
	found := false
	for _, c := range PromptContracts() {
		if strings.Contains(c.Rule, "flip side must oppose the bias") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the prompt-contract registry must carry the flip-direction restriction")
	}
	if err := ValidatePromptContracts(plannerOutputContract(0, 0, true, true)); err != nil {
		t.Fatalf("rendered prompt must state every restriction: %v", err)
	}
	err := "flip{below 29474.90 → long} contradicts bias short: a short bias flips to long only on a close above the line"
	if got := lawExcerptsFor(err); !strings.Contains(got, RepairFlipDirectionLaw) {
		t.Fatalf("repair excerpt for the direction error must carry RepairFlipDirectionLaw, got %q", got)
	}
}
