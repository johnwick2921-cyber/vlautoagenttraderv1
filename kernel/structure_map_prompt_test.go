package kernel

import (
	"strings"
	"testing"
)

// KNOB OFF → BYTE-IDENTICAL. A nil Structure renders nothing: the planner
// prompt with the field unset equals the prompt from a PlannerInput that never
// heard of it (the existing goldens are the second proof — they set nothing).
func TestStructureSectionIsAbsentWhenTheMapIsNil(t *testing.T) {
	base := PlannerInput{TradeDate: "2026-09-16", Session: "NY", Price: 29000, DATR: 250}
	withNil := base
	withNil.Structure = nil
	if BuildPlannerPrompt(base) != BuildPlannerPrompt(withNil) {
		t.Fatal("a nil structure map must leave the prompt byte-identical")
	}
	if strings.Contains(BuildPlannerPrompt(base), "STRUCTURE — bias only") {
		t.Fatal("the section rendered with no map")
	}
	if RenderStructureSection(nil) != "" {
		t.Fatal("RenderStructureSection(nil) must be empty")
	}
}

// KNOB ON → the section sits BEFORE the ranked level table, says bias-only,
// and carries one line per TF with the read values — never a placeholder.
func TestStructureSectionRendersBeforeTheLevelTable(t *testing.T) {
	m := &StructureMap{AsOf: 1, Contract: "MNQ 12-26", TFs: map[string]StructureTF{
		"D":  {Trend: "up", ImpulseLo: 28500, ImpulseHi: 29500, PremiumDiscount: 0.62, Bars: 69},
		"4h": {Trend: "range", ImpulseLo: 28900, ImpulseHi: 29200, PremiumDiscount: 0.33, Bars: 406, Zones: []StructureZone{{Kind: "OB", Lo: 28950, Hi: 28980, TF: "4h", Fresh: "fresh", Score: 7.1}}},
		"1h": {Trend: "down", ImpulseLo: 29050, ImpulseHi: 29180, PremiumDiscount: 0.80, Bars: 1553},
	}}
	in := PlannerInput{TradeDate: "2026-09-16", Session: "NY", Price: 29000, DATR: 250, Structure: m,
		Levels: []ScoredLevel{{DetectedLevel: DetectedLevel{Kind: LevelKind("PDH"), Price: 29100, Lo: 29100, Hi: 29100, Label: "PDH"}, Grade: "A", Score: 9}}}
	p := BuildPlannerPrompt(in)
	i := strings.Index(p, "## STRUCTURE — bias only, not entries")
	if i < 0 {
		t.Fatalf("section missing:\n%s", p)
	}
	for _, want := range []string{"D: up", "4h: range", "1h: down", "pd=0.62", "pd=0.33", "pd=0.80", "OB 28950.00–28980.00 (fresh)", "never an entry"} {
		if !strings.Contains(p, want) {
			t.Fatalf("section lacks %q:\n%s", want, p[i:min(len(p), i+900)])
		}
	}
	j := strings.Index(p, "PDH")
	if j < i {
		t.Fatalf("the level table (PDH at %d) must come AFTER the structure section (at %d)", j, i)
	}
}

// The per-read 🗺 line and the boot line are READ, never literal.
func TestStructureLogLines(t *testing.T) {
	m := &StructureMap{TFs: map[string]StructureTF{
		"D": {Trend: "up"}, "4h": {Trend: "range", PremiumDiscount: 0.33, Zones: []StructureZone{{}, {}}}, "1h": {Trend: "down", Zones: []StructureZone{{}}},
	}}
	line := StructureLogLine(m, "NY")
	for _, want := range []string{"🗺 structure @NY:", "D=up", "4h=range", "1h=down", "zones=3", "pd4h=0.33"} {
		if !strings.Contains(line, want) {
			t.Fatalf("log line lacks %q: %s", want, line)
		}
	}
	if !strings.Contains(StructureLogLine(nil, "NY"), "n/a") {
		t.Fatal("a nil map logs n/a, never zeros")
	}
	if StructureBootLine(false, true) != "🗺 structure: off" || StructureBootLine(true, true) != "🗺 structure: on(D/4h/1h)" || StructureBootLine(false, false) != "🗺 structure: n/a" {
		t.Fatalf("boot line: %q / %q / %q", StructureBootLine(false, true), StructureBootLine(true, true), StructureBootLine(false, false))
	}
}
