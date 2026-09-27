package kernel

import (
	"strings"
	"testing"
	"time"
)

// WAVE PLANNER A5 — rows 348/351/354/367. Class-250 probes: each test IS the
// caller, at the production CheckScenarioWriteTruth call site. The mutations
// that make these RED are named per test.

func a5Level(kind LevelKind, price float64, label string) DetectedLevel {
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, CTLocation())
	closeMs := now.Add(-10 * time.Hour).UnixMilli()
	return DetectedLevel{Kind: kind, Price: price, Lo: price, Hi: price, Label: label,
		OriginDate: "2026-09-09", FormedAtMs: closeMs - 60000, FormedCloseMs: &closeMs,
		FormationTF: "5m", IdentitySymbol: "MNQ", FormationLookback: 400}
}

// A5 row 348/354: the identity-unresolved refusal must LIST the nearest valid
// ids (nearest-first by |level price − scenario anchor|) so the planner copies
// one verbatim. RED mutation: nearestValidIDSuffix returning "" (or the list
// missing the nearest id) re-refuses this test.
func TestA5IdentityRefusalListsNearestValidIDs(t *testing.T) {
	cs := BuildMapCandidates([]ScoredLevel{
		{DetectedLevel: a5Level(KindSWGH, 30703.50, "SWG-H·15m"), Grade: "B", Score: 1},
		{DetectedLevel: a5Level(KindSWGL, 30644.25, "SWG-L·15m"), Grade: "C", Score: 0.5},
	}, 30690, 10, MapCandidateOpts{})
	if len(cs) != 2 {
		t.Fatalf("candidates = %d, want 2", len(cs))
	}
	var id307, id306 *string
	for i := range cs {
		if cs[i].Price == 30703.50 {
			id307 = cs[i].ID
		}
		if cs[i].Price == 30644.25 {
			id306 = cs[i].ID
		}
	}
	if id307 == nil || id306 == nil {
		t.Fatalf("fixture ids missing: %+v", cs)
	}
	mangled := "ref|64e783f826d26eeabf7715cbeb4d8d11e7aa3afe79f244692c103ce2dff4c5bd" // row 348 verbatim
	levels := IdentityLevelsFromCandidates(cs)
	sc := PlanScenario{ID: "S1", LevelID: &mangled, Trigger: "fade 30703.50", Direction: "short"}
	d := &PlanDoc{Scenarios: []PlanScenario{sc}, Levels: levels}
	v := CheckScenarioWriteTruth(d, cs, nil, 0.25)
	if v.Err() == nil || !strings.Contains(v.Err().Error(), "S1 identity unresolved") {
		t.Fatalf("the mangled id must be refused identity_unresolved: %+v", v.Err())
	}
	text := v.Err().Error()
	if !strings.Contains(text, "nearest valid ids") {
		t.Fatalf("refusal must list nearest valid ids: %s", text)
	}
	if !strings.Contains(text, *id307+"=30703.50") || !strings.Contains(text, *id306+"=30644.25") {
		t.Fatalf("refusal must list the map ids with their prices: %s", text)
	}
	if strings.Index(text, *id307) > strings.Index(text, *id306) {
		t.Fatalf("nearest valid ids must be nearest-first (anchor 30703.50): %s", text)
	}
}

// A5 row 367: the omits refusal names the missing level with its ID and price.
// RED mutation: dropping the [id=…] clause re-refuses this test.
func TestA5ObstacleOmitsRefusalNamesIDAndPrice(t *testing.T) {
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
		t.Fatalf("the omitted seated level must be refused by name: %+v", v.Err())
	}
	text := v.Err().Error()
	for _, want := range []string{"SWG-H·15m 30644.25 (273.50 pts from entry)", "[id=" + *cs[0].ID + "]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("omits refusal must name the level with its id and price, lacking %q: %s", want, text)
		}
	}
}

// A5 row 351: the comes-first refusal names the nearer level with its ID and
// price. RED mutation: dropping the [id=…] clause re-refuses this test.
func TestA5ObstacleComesFirstRefusalNamesIDAndPrice(t *testing.T) {
	cs := BuildMapCandidates([]ScoredLevel{
		{DetectedLevel: a5Level(KindSWGL, 30703.50, "SWG-L·15m"), Grade: "B", Score: 1},
	}, 30647, 10, MapCandidateOpts{})
	if len(cs) != 1 || cs[0].ID == nil {
		t.Fatalf("fixture: %+v", cs)
	}
	entry, target := 30647.83, 30710.00
	fo := 30706.00 // on the path, but not the nearest seated level
	sc := PlanScenario{ID: "S2", Direction: "long",
		Economics: &ScenarioEconomics{
			Geometry:      &ScenarioGeometry{Entry: entry, Stop: 30630, Target: target},
			FirstObstacle: &ScenarioObstacle{Price: &fo},
		}}
	d := &PlanDoc{Scenarios: []PlanScenario{sc}}
	v := CheckScenarioWriteTruth(d, cs, nil, 0.25)
	if v.Err() == nil || !strings.Contains(v.Err().Error(), "S2 obstacle chain: first_obstacle") {
		t.Fatalf("the out-of-order first_obstacle must be refused: %+v", v.Err())
	}
	text := v.Err().Error()
	for _, want := range []string{"SWG-L·15m 30703.50 (55.67 pts from entry 30647.83)", "[id=" + *cs[0].ID + "]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("comes-first refusal must name the level with its id and price, lacking %q: %s", want, text)
		}
	}
}
