package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
)

// M2.1 tail (CTO, 2026-09-23): the live desk strip logged
// "🔭 desk strip: line 10 (planner) panicked and was contained: nil pointer"
// on every scan. A scenario naming a REFERENCE level ("ref|…", W-GEOMETRY-
// REFUSAL b1) resolves to an anchor that has NO formation close by
// construction, and the planner line dereferenced Level.FormedCloseMs. At the
// production entry (DeskStripAt): line 10 renders, with formed_close_ms=n/a.
func TestDeskPlannerLineRendersAReferenceLevelWithoutAFormationClose(t *testing.T) {
	at := plannerTestTrader(t)
	at.config.NinjaTraderSymbol = "MNQ"
	now, _ := time.Parse(time.RFC3339, "2026-09-23T00:50:00-05:00")
	id, kind := "ref|onh-test", "ONH"
	plan := &kernel.ActivePlan{PlanID: "2026-09-23:ASIA:test", Session: "ASIA", Version: 1,
		Doc: kernel.PlanDoc{
			Levels:         []kernel.PlanLevel{{Price: 30000, Label: "ONH", Grade: "A"}},
			IdentityLevels: []kernel.PlanLevel{{Price: 30000, Label: "ONH", ID: &id, Kind: &kind}}, // no FormedCloseMs
			Scenarios:      []kernel.PlanScenario{{ID: "S1", LevelID: &id, Direction: "short", Condition: "reject", Trigger: "reject 30000"}},
		}}
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan { return plan }})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })

	var line DeskLine
	for _, l := range at.DeskStripAt(now).Lines {
		if l.Key == "planner" {
			line = l
		}
	}
	if strings.Contains(line.Reason, "panicked") {
		t.Fatalf("the planner line panicked on a reference level: %+v", line)
	}
	if !strings.Contains(line.Text, "level_id=ref|onh-test") || !strings.Contains(line.Text, "formed_close_ms=n/a") {
		t.Fatalf("a reference level renders with its id and formed_close_ms=n/a: %q", line.Text)
	}
}
