package store

import "testing"

// day_plan.structure_map — default OFF (every S1 knob is), saved true → on.
func TestResolveStructureMapDefaultsOff(t *testing.T) {
	if en, src := ResolveStructureMap(nil); en || src != SourceShippedDefault {
		t.Fatalf("nil cfg: %v %q", en, src)
	}
	if en, _ := ResolveStructureMap(&StrategyConfig{DayPlan: &DayPlanConfig{}}); en {
		t.Fatal("unset knob must read OFF")
	}
	on := true
	if en, src := ResolveStructureMap(&StrategyConfig{DayPlan: &DayPlanConfig{StructureMap: &on}}); !en || src != SourceSaved {
		t.Fatalf("saved true: %v %q", en, src)
	}
}
