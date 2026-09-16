package kernel

import (
	"encoding/json"
	"testing"
)

func TestClassifyCitation(t *testing.T) {
	doc := PlanDoc{Scenarios: []PlanScenario{
		{ID: "S1", Direction: "long"},
		{ID: "S2", Direction: "short"},
	}}

	// off-plan variants.
	for _, c := range []string{"", "off-plan", "offplan", "S9-nope"} {
		r := ClassifyCitation("open_long", c, doc)
		if !r.OffPlan || r.Cited != "off-plan" {
			t.Fatalf("cite %q should be off-plan: %+v", c, r)
		}
	}
	// matched: S1 is long, action open_long.
	if r := ClassifyCitation("open_long", "S1", doc); !r.Matched || r.OffPlan || r.Cited != "S1" {
		t.Fatalf("S1 long + open_long should match: %+v", r)
	}
	// cited a real scenario but direction mismatches → cited, not matched, not off-plan.
	if r := ClassifyCitation("open_short", "S1", doc); r.Matched || r.OffPlan || r.Cited != "S1" {
		t.Fatalf("S1 long + open_short should be cited-not-matched: %+v", r)
	}
	// case-insensitive id.
	if r := ClassifyCitation("open_short", "s2", doc); !r.Matched {
		t.Fatalf("s2 short + open_short should match (case-insensitive): %+v", r)
	}
}

func TestDecisionCitedScenarioParses(t *testing.T) {
	raw := `{"symbol":"MNQ","action":"open_long","confidence":72,"reasoning":"S1 setup","cited_scenario":"S1"}`
	var d Decision
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.CitedScenario != "S1" {
		t.Fatalf("cited_scenario = %q want S1", d.CitedScenario)
	}
	// Absent field → empty (back-compat: existing decisions unchanged).
	var d2 Decision
	_ = json.Unmarshal([]byte(`{"symbol":"MNQ","action":"wait"}`), &d2)
	if d2.CitedScenario != "" {
		t.Fatalf("absent cited_scenario must be empty")
	}
}
