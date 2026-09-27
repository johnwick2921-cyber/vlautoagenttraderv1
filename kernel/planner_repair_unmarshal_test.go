package kernel

import (
	"strings"
	"testing"
)

// FOLD from DS-104 cross-check (PR #242) — schema_json killer: plan JSON
// unmarshal failures (persisted kills 357-359 / 391-393:
// "plan JSON unmarshal: json: cannot unmarshal string into Go struct field
// PlanBreakdownContinue.scenarios.breakdown.level of type float64") got the
// GENERIC repair excerpt because lawExcerptsFor has no unmarshal case. The fix:
// an unmarshal-specific excerpt that quotes the exact decode error and the
// minimal schema for the offending field. The error string is built by the
// PRODUCTION parse (parsePlanDocument, the same line the read loop reports).
func TestFpUnmarshalRepairExcerptQuotesDecodeErrorAndSchema(t *testing.T) {
	raw := `{"scenarios":[{"id":"S1","direction":"long","breakdown":{"level":"30644.25","entry_mode":"pullback"}}]}`
	_, err := parsePlanDocument(raw, 12, 5, false, AuthoringOpts{})
	if err == nil || !strings.Contains(err.Error(), "plan JSON unmarshal") || !strings.Contains(err.Error(), "cannot unmarshal string") {
		t.Fatalf("fixture must produce the production unmarshal refusal: %v", err)
	}
	if !strings.Contains(err.Error(), "breakdown.level") || !strings.Contains(err.Error(), "float64") {
		t.Fatalf("the decode error must name the field path and type: %v", err)
	}
	p := BuildPlannerRepairPrompt("{}", err.Error(), nil)
	for _, want := range []string{
		"breakdown.level",
		"float64",
		`"level": <number>`, // the minimal schema for the offending field
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("unmarshal repair excerpt lacks %q:\n%s", want, p)
		}
	}
	// The persisted killer's EXACT error string must route to the same excerpt
	// (the repair prompt receives the error verbatim from the read loop).
	persisted := `plan JSON unmarshal: json: cannot unmarshal string into Go struct field PlanBreakdownContinue.scenarios.breakdown.level of type float64`
	p2 := BuildPlannerRepairPrompt("{}", persisted, nil)
	if !strings.Contains(p2, `"level": <number>`) || !strings.Contains(p2, "float64") {
		t.Fatalf("the persisted kill's error must route to the schema excerpt:\n%s", p2)
	}
}
