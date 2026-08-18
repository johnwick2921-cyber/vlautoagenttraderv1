package kernel

import (
	"strings"
	"testing"
)

const validPlanJSON = `{
  "reasoning": "Overnight swept ONH then reclaimed; balance below PDH. Fade edges, long the reclaim.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "flips short on 2x5m < 15480"},
  "levels": [
    {"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade first tap"},
    {"price": 15480, "label": "ONL", "grade": "B", "instruction": "reclaim-long"}
  ],
  "scenarios": [
    {"id": "S1", "trigger": "sweep 15480 then reclaim", "condition": "sweep_reclaim", "direction": "long", "target_chain": [15550, 15620], "invalid": "2x5m < 15470", "quality": "A"},
    {"id": "S2", "trigger": "reject 15620", "condition": "reject", "direction": "short", "target_chain": [15550], "invalid": "acceptance > 15625", "quality": "B"}
  ],
  "no_trade": ["first 5m", "12:00-13:30 CT lunch"],
  "death_condition": "acceptance above 15620 kills the fade thesis",
  "day_type": "balance"
}`

func TestParsePlanDocValid(t *testing.T) {
	doc, err := ParsePlanDoc(validPlanJSON)
	if err != nil {
		t.Fatalf("valid plan should parse: %v", err)
	}
	if doc.Bias.Direction != "long" || len(doc.Scenarios) != 2 || len(doc.Levels) != 2 {
		t.Fatalf("parsed doc wrong: %+v", doc)
	}
	if doc.Scenarios[0].Condition != "sweep_reclaim" {
		t.Fatalf("scenario condition = %q", doc.Scenarios[0].Condition)
	}
}

func TestParsePlanDocFromWrappedOutput(t *testing.T) {
	wrapped := "Here is the plan:\n```json\n" + validPlanJSON + "\n```\nThanks."
	if _, err := ParsePlanDoc(wrapped); err != nil {
		t.Fatalf("should extract JSON from prose/fence: %v", err)
	}
}

func TestValidatePlanDocRejects(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*PlanDoc)
		hint string
	}{
		{"no reasoning", func(d *PlanDoc) { d.Reasoning = "" }, "reasoning"},
		{"bad bias dir", func(d *PlanDoc) { d.Bias.Direction = "sideways" }, "direction"},
		{"no death", func(d *PlanDoc) { d.DeathCondition = "" }, "death_condition"},
		{"bad grade", func(d *PlanDoc) { d.Levels[0].Grade = "Z" }, "grade"},
		{"bad condition", func(d *PlanDoc) { d.Scenarios[0].Condition = "vibes" }, "condition"},
		{"bad quality", func(d *PlanDoc) { d.Scenarios[0].Quality = "S" }, "quality"},
		{"zero scenarios", func(d *PlanDoc) { d.Scenarios = nil }, "scenarios count"},
	}
	for _, c := range cases {
		doc, _ := ParsePlanDoc(validPlanJSON)
		c.mut(doc)
		err := ValidatePlanDoc(doc)
		if err == nil {
			t.Fatalf("%s: expected validation error", c.name)
		}
		if !strings.Contains(err.Error(), c.hint) {
			t.Fatalf("%s: error %q missing %q", c.name, err.Error(), c.hint)
		}
	}
}

func TestTooManyScenariosRejected(t *testing.T) {
	doc, _ := ParsePlanDoc(validPlanJSON)
	doc.Scenarios = append(doc.Scenarios, doc.Scenarios[0], doc.Scenarios[1]) // now 4
	if err := ValidatePlanDoc(doc); err == nil {
		t.Fatalf("4 scenarios must be rejected (max 3)")
	}
}

func TestNoTradePlanDocIsValid(t *testing.T) {
	// The fail-closed doc must itself pass validation (it becomes a real row).
	if err := ValidatePlanDoc(NoTradePlanDoc("planner timeout")); err != nil {
		t.Fatalf("fail-closed no-trade plan must be valid: %v", err)
	}
}

func TestExtractJSONObjectBalanced(t *testing.T) {
	// Nested braces + a brace inside a string must not confuse the extractor.
	in := `prefix {"a": {"b": 1}, "s": "has } brace"} suffix`
	got := extractJSONObject(in)
	if got != `{"a": {"b": 1}, "s": "has } brace"}` {
		t.Fatalf("extract = %q", got)
	}
}
