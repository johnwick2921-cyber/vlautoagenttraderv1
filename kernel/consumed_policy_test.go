package kernel

import (
	"strings"
	"testing"
)

func TestG5MarkConsumedScenarios(t *testing.T) {
	doc := &PlanDoc{
		Levels: []PlanLevel{
			{Price: 29470.25, Label: "RTH-H"},
			{Price: 29358.25, Label: "PDC"},
		},
		Scenarios: []PlanScenario{
			{ID: "S1", Trigger: "Price lifts into 29470.25 RTH-H and shows a 5m rejection", Quality: "A"},
			{ID: "S2", Trigger: "Breakout retest of 29358.25 PDC", Quality: "B"},
		},
	}
	// 29470.25 consumed → S1 demoted to C + badge; S2 (fresh) untouched.
	n := MarkConsumedScenarios(doc, map[float64]bool{29470.25: true})
	if n != 1 {
		t.Fatalf("want 1 demotion, got %d", n)
	}
	if doc.Scenarios[0].Quality != "C" || !doc.Scenarios[0].Consumed {
		t.Fatalf("S1 must be capped C + badge, got %+v", doc.Scenarios[0])
	}
	if doc.Scenarios[1].Quality != "B" || doc.Scenarios[1].Consumed {
		t.Fatalf("S2 must stay fresh, got %+v", doc.Scenarios[1])
	}
	// Already-consumed stays consumed without double demotion.
	n = MarkConsumedScenarios(doc, map[float64]bool{29470.25: true})
	if n != 0 {
		t.Fatalf("idempotent re-mark must return 0, got %d", n)
	}
}

func TestG5PlannerPromptListsConsumedLevels(t *testing.T) {
	in := PlannerInput{
		Levels:         []ScoredLevel{{DetectedLevel: DetectedLevel{Price: 29470.25, Label: "RTH-H"}, Grade: "A"}},
		ConsumedLevels: []string{"29470.25 RTH-H"},
	}
	out := BuildPlannerPrompt(in)
	if !strings.Contains(out, "## Consumed levels") || !strings.Contains(out, "29470.25 RTH-H") {
		t.Fatalf("planner prompt must list consumed levels, got:\n%s", out)
	}
}
