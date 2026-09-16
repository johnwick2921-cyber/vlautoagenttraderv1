package trader

import (
	"testing"

	"nofx/kernel"
)

// S1-wave A3 (2026-08-29) — the write-site stamp map must cover the FULL
// HTF-zone universe, not just the cap-4 prompt section: the 13 Demand·1h
// escapes were rows the model wrote from the key-levels block while the map
// only knew the cap-4 section. This pins the behavior at the collection
// layer (the same pure helper the write site calls).
func TestCollectMachineGradesCoversHTFUniverse(t *testing.T) {
	in := kernel.PlannerInput{
		Levels:   []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Price: 100.0, Label: "seated"}, Grade: "B"}},
		Pool:     []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Price: 101.5, Label: "pool-row"}, Grade: "C"}},
		HTFZones: []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Price: 102.0, Label: "cap4-zone"}, Grade: "A"}},
		// The escape class: a 1h zone outside the cap-4 section.
		HTFZonesFull: []kernel.ScoredLevel{
			{DetectedLevel: kernel.DetectedLevel{Price: 102.0, Label: "cap4-zone"}, Grade: "A"},
			{DetectedLevel: kernel.DetectedLevel{Price: 103.25, Label: "Demand·1h (HTF)"}, Grade: "A"},
			{DetectedLevel: kernel.DetectedLevel{Price: 104.75, Label: "Supply·4h"}, Grade: "C"},
		},
	}
	grades := map[float64]string{}
	labels := map[float64]string{}
	collectMachineGrades(in, grades, labels)

	if grades[103.25] != "A" || labels[103.25] != "Demand·1h (HTF)" {
		t.Fatalf("full-universe 1h zone not recorded: g=%q l=%q", grades[103.25], labels[103.25])
	}
	if grades[104.75] != "C" {
		t.Fatalf("full-universe 4h zone not recorded: %q", grades[104.75])
	}
	// Regressions: the seated/pool/cap-4 sources keep working.
	if grades[100.0] != "B" || grades[101.5] != "C" || grades[102.0] != "A" {
		t.Fatalf("legacy sources regressed: %+v", grades)
	}
	// Collision rule: the stronger grade wins at a shared rounded price.
	in2 := kernel.PlannerInput{HTFZonesFull: []kernel.ScoredLevel{
		{DetectedLevel: kernel.DetectedLevel{Price: 200.0, Label: "weak"}, Grade: "C"}, {DetectedLevel: kernel.DetectedLevel{Price: 200.0, Label: "strong"}, Grade: "A"},
	}}
	g2 := map[float64]string{}
	collectMachineGrades(in2, g2, map[float64]string{})
	if g2[200.0] != "A" {
		t.Fatalf("collision rule regressed: %q", g2[200.0])
	}
}
