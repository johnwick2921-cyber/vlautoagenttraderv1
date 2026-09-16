package kernel

import (
	"strings"
	"testing"
)

// A6 (F12) — with an active plan, an open without cited_scenario fails the
// parse (→ retry); with it, the decision passes this check; waits are exempt.
func TestRequireCitedScenario(t *testing.T) {
	fd := &FullDecision{Decisions: []Decision{{Action: "open_short", Symbol: "MNQ"}}}
	if err := requireCitedScenario(fd); err == nil || !strings.Contains(err.Error(), "cited_scenario") {
		t.Fatalf("uncited open must fail the parse for the retry loop (got %v)", err)
	}
	fd.Decisions[0].CitedScenario = "S1"
	if err := requireCitedScenario(fd); err != nil {
		t.Fatalf("cited open must pass: %v", err)
	}
	fd.Decisions[0].CitedScenario = "off-plan"
	if err := requireCitedScenario(fd); err != nil {
		t.Fatalf("off-plan is a valid citation: %v", err)
	}
	if err := requireCitedScenario(&FullDecision{Decisions: []Decision{{Action: "wait", Symbol: "MNQ"}}}); err != nil {
		t.Fatalf("waits are exempt: %v", err)
	}
}
