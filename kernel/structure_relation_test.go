package kernel

import (
	"strings"
	"testing"
)

// ── S3 (2026-09-16) — structure seating + relation tests ─────────────────────

// S3(b): structure-table zones live in the separate `structure` block — they do
// NOT count against max_levels. The 12 ceiling stays for the ENTRY table only.
func TestS3StructureZonesDoNotCountAgainstMaxLevels(t *testing.T) {
	doc, err := ParsePlanDoc(validPlanJSON)
	if err != nil {
		t.Fatalf("fixture parse: %v", err)
	}
	for i := 0; i < 10; i++ {
		doc.Levels = append(doc.Levels, PlanLevel{Price: 15000 + float64(i), Label: "RN", Grade: "C"})
	}
	if len(doc.Levels) != 12 {
		t.Fatalf("fixture should hold 12 entry levels, got %d", len(doc.Levels))
	}
	doc.Structure = &StructureMap{
		TFs: map[string]StructureTF{
			"4h": {Trend: "up", Zones: []StructureZone{
				{Kind: "OB", Lo: 100, Hi: 101, TF: "4h"},
				{Kind: "FVG", Lo: 90, Hi: 92, TF: "4h"},
			}},
			"D": {Trend: "down"},
		},
	}
	if err := ValidatePlanDocWithCaps(doc, 12, 5); err != nil {
		t.Fatalf("12 entry levels + structure zones must pass (structure is NOT the entry table): %v", err)
	}
	doc.Levels = append(doc.Levels, PlanLevel{Price: 16000, Label: "RN", Grade: "C"})
	if err := ValidatePlanDocWithCaps(doc, 12, 5); err == nil || !strings.Contains(err.Error(), "too many levels") {
		t.Fatalf("13 entry levels must hit the 12 ceiling, got: %v", err)
	}
}

// S3(c): the validator stamps relation_d / relation_4h from the structure
// table; the model's own values are preserved as a claim and overwritten.
func TestStampScenarioRelations(t *testing.T) {
	doc := &PlanDoc{
		Scenarios: []PlanScenario{
			{ID: "S1", Direction: "long", RelationD: "counter-trend"},     // model claim → moved
			{ID: "S2", Direction: "short", RelationClaimed: "with-trend"}, // explicit claim kept
			{ID: "S3", Direction: "long"},                                 // no claim
			{ID: "S4", Direction: "neutral"},                              // neutral → range
		},
		Structure: &StructureMap{TFs: map[string]StructureTF{
			"D":  {Trend: "up"},
			"4h": {Trend: "down"},
		}},
	}
	StampScenarioRelations(doc)
	s := doc.Scenarios
	if s[0].RelationD != RelationWithTrend || s[0].Relation4h != RelationCounterTrend {
		t.Fatalf("S1 long vs D-up/4h-down: relation_d=%q relation_4h=%q", s[0].RelationD, s[0].Relation4h)
	}
	if s[0].RelationClaimed != "counter-trend" {
		t.Fatalf("S1 model-supplied relation_d must move to relation_claimed, got %q", s[0].RelationClaimed)
	}
	if s[1].RelationD != RelationCounterTrend || s[1].Relation4h != RelationWithTrend || s[1].RelationClaimed != "with-trend" {
		t.Fatalf("S2 short: %+v", s[1])
	}
	if s[2].RelationD != RelationWithTrend || s[2].Relation4h != RelationCounterTrend || s[2].RelationClaimed != "" {
		t.Fatalf("S3 long: %+v", s[2])
	}
	if s[3].RelationD != RelationRange || s[3].Relation4h != RelationRange {
		t.Fatalf("S4 neutral must resolve range, got %+v", s[3])
	}
}

// absent structure / missing TF leaves the fields EMPTY — absent ≠ fabricated.
func TestStampScenarioRelationsAbsentLeavesEmpty(t *testing.T) {
	doc := &PlanDoc{Scenarios: []PlanScenario{{ID: "S1", Direction: "long"}}}
	StampScenarioRelations(doc)
	if doc.Scenarios[0].RelationD != "" || doc.Scenarios[0].Relation4h != "" {
		t.Fatalf("no structure → fields must stay empty, got %+v", doc.Scenarios[0])
	}
	doc.Structure = &StructureMap{TFs: map[string]StructureTF{"1h": {Trend: "up"}}}
	StampScenarioRelations(doc)
	if doc.Scenarios[0].RelationD != "" || doc.Scenarios[0].Relation4h != "" {
		t.Fatalf("no D/4h in structure → fields must stay empty, got %+v", doc.Scenarios[0])
	}
}

// fail-closed rate must not move: a counter-trend scenario WITHOUT any relation
// passes validation (restriction-with-hint, NOT a block). The flag lives in the
// prompt and the census, never in the reject list.
func TestCounterTrendScenarioNotRejected(t *testing.T) {
	doc := &PlanDoc{
		Reasoning:      "counter-trend fade at the top of a 4h uptrend",
		Bias:           PlanBias{Direction: "short", Conviction: "low", FlipCondition: "n/a"},
		Levels:         []PlanLevel{{Price: 29680.75, Label: "ONL", Grade: "A"}, {Price: 29853, Label: "PDL", Grade: "A"}, {Price: 29919, Label: "PDC", Grade: "A"}, {Price: 30079, Label: "RTH-L", Grade: "A"}, {Price: 29400, Label: "RN 29400", Grade: "B"}, {Price: 29360, Label: "PWL", Grade: "A"}, {Price: 29300, Label: "RN 29300", Grade: "B"}},
		Scenarios:      []PlanScenario{{ID: "S1", Trigger: "reject at 29360", Condition: "reject", Direction: "short", TargetChain: []float64{29300}, Invalid: "5m close above 29400", Quality: "B"}},
		NoTrade:        []string{"first 5m"},
		DeathCondition: "n/a",
		Structure:      &StructureMap{TFs: map[string]StructureTF{"4h": {Trend: "up"}}},
	}
	facts := PlanFacts{Price: 29687.5, DATR: 300, PDL: 29853, PDH: 30054}
	machine := map[float64]string{29680.75: "ONL", 29853: "PDL", 29919: "PDC", 30079: "RTH-L", 29400: "RN", 29360: "PWL", 29300: "RN"}
	err := ValidatePlanDocWithFactsMachine(doc, facts, machine, 8, 3)
	if err != nil {
		t.Fatalf("a counter-trend short without relation must NOT be rejected (restriction-with-hint): %v", err)
	}
	if doc.Scenarios[0].Relation4h != RelationCounterTrend {
		t.Fatalf("the validator must STAMP relation_4h=counter-trend, got %q", doc.Scenarios[0].Relation4h)
	}
}
