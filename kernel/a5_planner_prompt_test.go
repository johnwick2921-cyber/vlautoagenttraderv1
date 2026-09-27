package kernel

import (
	"strings"
	"testing"
	"time"
)

// WAVE PLANNER A5 — prompt side. Class-250: the tests are the callers, at the
// production BuildPlannerPrompt call site (the planner read path renders via
// RenderIdentityMapBlock → renderMapBlock).

// A5 item 1: the planner prompt table prints level ids VERBATIM (the full id
// string, never truncated) — the map row the model must copy. Pin at the
// production renderer; the RED mutation is truncating the id in renderMapBlock.
func TestA5PlannerPromptPrintsLevelIDsVerbatim(t *testing.T) {
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, CTLocation())
	lvl := ScoredLevel{DetectedLevel: a5Level(KindSWGH, 100, "SWG-H·15m"), Grade: "B", Score: 1}
	cs := BuildMapCandidates([]ScoredLevel{lvl}, 99, 3, MapCandidateOpts{})
	if len(cs) == 0 || cs[0].ID == nil {
		t.Fatalf("fixture candidate must carry a strict id: %+v", cs)
	}
	p := BuildPlannerPrompt(PlannerInput{Now: now, Session: "NY", Price: 99, ATR5m: 3, Levels: []ScoredLevel{lvl}})
	if !strings.Contains(p, "id="+*cs[0].ID) {
		t.Fatalf("the planner prompt must print the full id verbatim; got map rows:\n%s", mapBlockOf(p))
	}
}

// A5 item 4: the prompt shows the ordered obstacle list PER candidate — each
// ENTRY SHORTLIST row names, nearest-first in the fade direction, the seated
// levels between it and the targets, with id and price. Gated by
// planner_contract (nil=ON); OFF renders today's block byte-identically.
func TestA5PlannerPromptOrderedObstacleListPerCandidate(t *testing.T) {
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, CTLocation())
	levels := []ScoredLevel{
		{DetectedLevel: a5Level(KindSWGL, 90, "SWG-L·15m"), Grade: "B", Score: 1},
		{DetectedLevel: a5Level(KindPDL, 100, "PDL"), Grade: "A", Score: 2},
		{DetectedLevel: a5Level(KindSWGH, 115, "SWG-H·15m"), Grade: "C", Score: 0.5},
	}
	cs := BuildMapCandidates(levels, 100, 3, MapCandidateOpts{})
	legacy := RenderIdentityMapBlock(cs, 100)
	if contract := RenderIdentityMapBlockContract(cs, 100, false); contract != legacy {
		t.Fatalf("knob OFF must render today's block byte-identically\n--- OFF ---\n%s\n--- legacy ---\n%s", contract, legacy)
	}
	p := BuildPlannerPrompt(PlannerInput{Now: now, Session: "NY", Price: 100, ATR5m: 3,
		Levels: levels, PlannerContractOn: true})
	if !strings.Contains(p, "obstacles→") {
		t.Fatalf("knob ON must show the ordered obstacle list per candidate:\n%s", mapBlockOf(p))
	}
	// The 90.00 entry candidate fades long: obstacles are PDL 100 then
	// SWG-H·15m 115, nearest-first, each with its id (trimFloat drops trailing
	// zeros — the same rendering the map rows use).
	if !strings.Contains(p, "PDL 100 [id=") || !strings.Contains(p, "SWG-H·15m 115 [id=") {
		t.Fatalf("obstacle chain must name each seated level with label, price and id:\n%s", mapBlockOf(p))
	}
	if strings.Index(p, "PDL 100 [id=") > strings.Index(p, "SWG-H·15m 115 [id=") {
		t.Fatalf("obstacle chain must be nearest-first:\n%s", mapBlockOf(p))
	}
}

func mapBlockOf(p string) string {
	if i := strings.Index(p, "MAP (merged"); i >= 0 {
		return p[i:]
	}
	return p
}
