package kernel

import (
	"strings"
	"testing"
)

// FIX-PLANNER (2026-09-26) item 2 — the repair prompt must list EXACTLY the
// omitted seated levels with id, price and the three legal roles, plus the
// validator's one-line rule. The error string is built by the PRODUCTION call
// site (CheckScenarioWriteTruth — same fixture as
// TestA5ObstacleOmitsRefusalNamesIDAndPrice); the repair prompt must render a
// structured per-level block from it, not just carry it verbatim.
func TestFpRepairPromptListsOmittedSeatedLevelsWithRoles(t *testing.T) {
	cs := BuildMapCandidates([]ScoredLevel{
		{DetectedLevel: a5Level(KindSWGH, 30644.25, "SWG-H·15m"), Grade: "B", Score: 1},
	}, 30370, 10, MapCandidateOpts{})
	if len(cs) != 1 || cs[0].ID == nil {
		t.Fatalf("fixture: %+v", cs)
	}
	entry, target := 30370.75, 30686.50
	sc := PlanScenario{ID: "S3", Direction: "long",
		Economics: &ScenarioEconomics{Geometry: &ScenarioGeometry{Entry: entry, Stop: 30350, Target: target}}}
	d := &PlanDoc{Scenarios: []PlanScenario{sc}}
	v := CheckScenarioWriteTruth(d, cs, nil, 0.25)
	if v.Err() == nil || !strings.Contains(v.Err().Error(), "S3 obstacle chain: omits") {
		t.Fatalf("fixture must refuse with the omits class: %+v", v.Err())
	}

	p := BuildPlannerRepairPrompt("{}", v.Err().Error(), nil)
	for _, want := range []string{
		"## Omitted seated levels (copy EXACTLY, with their ids, into economics.path_levels)",
		"- SWG-H·15m 30644.25 (273.50 pts from entry) [id=" + *cs[0].ID + "] → legal roles: pass_through | reduce | exit",
		RepairObstacleChainLaw,
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("repair prompt lacks %q:\n%s", want, p)
		}
	}
}
