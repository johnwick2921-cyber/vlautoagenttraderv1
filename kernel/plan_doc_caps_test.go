package kernel

import (
	"fmt"
	"strings"
	"testing"
)

// H4/H5 — max_levels (9–12) and scenario_cap (4–5) are offered in the UI but the
// validator hardcoded 8/3, so raising either setting made EVERY planner read
// fail-closed (whole plan rejected → NO-TRADE + P0 alert). The validator and the
// prompt must both honor the RESOLVED caps, with 12/5 as hard ceilings.

func capsPlanJSON(levels, scenarios int) string {
	var lv strings.Builder
	for i := 0; i < levels; i++ {
		if i > 0 {
			lv.WriteString(",")
		}
		fmt.Fprintf(&lv, `{"price": %d, "label": "PDH", "grade": "A", "instruction": "fade"}`, 15000+i*10)
	}
	var sc strings.Builder
	for i := 0; i < scenarios; i++ {
		if i > 0 {
			sc.WriteString(",")
		}
		fmt.Fprintf(&sc, `{"id": "S%d", "trigger": "t", "condition": "reject", "direction": "long", "target_chain": [15100], "invalid": "n/a", "quality": "B"}`, i+1)
	}
	return fmt.Sprintf(`{"reasoning": "ok", "bias": {"direction": "neutral", "conviction": "low", "flip_condition": "n/a"}, "levels": [%s], "scenarios": [%s], "no_trade": [], "death_condition": "n/a"}`, lv.String(), sc.String())
}

func TestValidatePlanDocHonorsRaisedCaps(t *testing.T) {
	// max_levels=12 + scenario_cap=5 must produce a VALID plan — before this the
	// hardcoded 8/3 rejected it and every read fail-closed.
	doc, err := ParsePlanDocCapped(capsPlanJSON(12, 5), 12, 5)
	if err != nil {
		t.Fatalf("12 levels / 5 scenarios with raised caps must validate, got: %v", err)
	}
	if len(doc.Levels) != 12 || len(doc.Scenarios) != 5 {
		t.Fatalf("levels=%d scenarios=%d, want 12/5", len(doc.Levels), len(doc.Scenarios))
	}
}

func TestValidatePlanDocHardCeilingsReject(t *testing.T) {
	// 13/6 exceed the hard ceilings; a caller asking for more is clamped DOWN and
	// the plan is still rejected — the schema never widens past the UI range.
	if _, err := ParsePlanDocCapped(capsPlanJSON(13, 5), 13, 5); err == nil {
		t.Fatalf("13 levels must be rejected even when the caller asked for 13")
	}
	if _, err := ParsePlanDocCapped(capsPlanJSON(12, 6), 12, 6); err == nil {
		t.Fatalf("6 scenarios must be rejected even when the caller asked for 6")
	}
	// And the shipped defaults still reject what they always rejected.
	if _, err := ParsePlanDoc(capsPlanJSON(9, 3)); err == nil {
		t.Fatalf("9 levels at the shipped default cap must still be rejected")
	}
	if _, err := ParsePlanDoc(capsPlanJSON(8, 4)); err == nil {
		t.Fatalf("4 scenarios at the shipped default cap must still be rejected")
	}
}

func TestPlannerOutputContractQuotesResolvedCaps(t *testing.T) {
	// The prompt must ask for EXACTLY what validation accepts: a raised cap both
	// gets requested and passes, instead of the planner being asked for 8/3 while
	// the validator demands it (or vice versa).
	p := BuildPlannerPrompt(PlannerInput{MaxLevels: 12, ScenarioCap: 5})
	if !strings.Contains(p, "// max 12") || !strings.Contains(p, "// 1..5") {
		t.Fatalf("raised caps missing from the output contract:\n%s", p)
	}
	// Defaults render byte-equivalent to the old const (max 8 / 1..3).
	pDef := BuildPlannerPrompt(PlannerInput{})
	if !strings.Contains(pDef, "// max 8") || !strings.Contains(pDef, "// 1..3") {
		t.Fatalf("default caps changed:\n%s", pDef)
	}
}

func TestNoTradePlanDocValidAtHardCeilings(t *testing.T) {
	if err := ValidatePlanDocWithCaps(NoTradePlanDoc("timeout"), PlanHardMaxLevels, PlanHardMaxScenarios); err != nil {
		t.Fatalf("fail-closed no-trade plan must stay valid at the hard ceilings: %v", err)
	}
}
