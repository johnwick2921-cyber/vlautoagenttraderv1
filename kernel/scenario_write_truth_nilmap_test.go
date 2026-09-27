package kernel

import (
	"strings"
	"testing"
)

// CTO note (msg 1790182338037): a nil frozen map means there is nothing an id
// could name. A scenario that carries a non-null level_id (or a two-anchor id)
// while the map is nil is REFUSED as identity_unresolved — never skipped as
// UNKNOWN. A scenario with no ids stays unjudged (the chain check needs a map).
func TestWriteTruthNilMapRefusesAnyNamedID(t *testing.T) {
	id, sweep := "L-PDH-30917.50", "ref|abc123"
	d := &PlanDoc{Scenarios: []PlanScenario{
		{ID: "S1", LevelID: &id},
		{ID: "S2", SweepLevelID: &sweep},
		{ID: "S3"},
	}}
	v := CheckScenarioWriteTruth(d, nil, nil, 0.25)
	got := map[string]string{}
	for _, is := range v.Issues {
		got[is.Scenario] = is.Class
		if !strings.HasPrefix(is.Text, is.Scenario+" identity unresolved") || !strings.Contains(is.Text, "map is nil") {
			t.Errorf("reason must lead with the scenario and say the map is nil: %q", is.Text)
		}
	}
	if got["S1"] != WriteTruthIdentityUnresolved || got["S2"] != WriteTruthIdentityUnresolved {
		t.Fatalf("named ids with a nil map must be refused identity_unresolved, got %+v", v.Issues)
	}
	if _, judged := got["S3"]; judged {
		t.Fatalf("a scenario naming no id is not refused for identity: %+v", v.Issues)
	}
	if v.ChainChecked {
		t.Fatal("the obstacle chain needs a map: it must stay unjudged with a nil map")
	}
	if v.Err() == nil {
		t.Fatal("the verdict must refuse")
	}
	// No ids at all + nil map → nothing to refuse (UNKNOWN, not refused).
	if v := CheckScenarioWriteTruth(&PlanDoc{Scenarios: []PlanScenario{{ID: "S1"}}}, nil, nil, 0.25); v.Err() != nil {
		t.Fatalf("no ids and a nil map must not refuse: %v", v.Err())
	}
}
