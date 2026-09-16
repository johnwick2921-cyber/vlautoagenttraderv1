package kernel

import "strings"

// ── REPAIR-PARSE (2026-09-02) — OUTCOME CLASSIFICATION ───────────────────────
// The dispatch expected "unparseable" repairs (fences, prose, fragments). The
// journals say otherwise: extractJSONObject already scans to the first `{` and
// walks to its matching `}`, so fences and surrounding prose were ALREADY
// tolerated — 17 of 18 rejected repairs parsed cleanly and were rejected on
// FIELD VALUES. Classifying the outcome is what was missing: "parse/schema
// rejected" covered a packaging failure and a vocabulary error with one label.

// RepairOutcome is the classified result of a repair attempt's output.
type RepairOutcome string

const (
	RepairOK        RepairOutcome = "ok"         // parsed and validated
	RepairPackaging RepairOutcome = "packaging"  // no JSON object, or JSON that will not unmarshal (type/shape)
	RepairFragment  RepairOutcome = "fragment"   // valid JSON, but not a whole plan
	RepairContent   RepairOutcome = "content"    // whole plan, rejected on field values
	RepairNoOutcome RepairOutcome = "no_outcome" // empty output
)

// FragmentReason is the specific message a fragment gets, instead of the
// generic schema error a partial document would otherwise produce.
const FragmentReason = "repair returned a fragment, not the full plan — the contract is the COMPLETE plan document (bias + levels + scenarios), not the repaired scenario alone"

// ClassifyRepairOutcome labels what a repair attempt produced. `raw` is the
// model's output; `err` is the parse/validate error (nil when it landed).
func ClassifyRepairOutcome(raw string, err error) RepairOutcome {
	if strings.TrimSpace(raw) == "" {
		return RepairNoOutcome
	}
	if err == nil {
		return RepairOK
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no JSON object found"), strings.Contains(msg, "plan JSON unmarshal"):
		return RepairPackaging
	case strings.Contains(msg, FragmentReason), IsPlanFragment(raw):
		return RepairFragment
	}
	return RepairContent
}

// IsPlanFragment reports whether the output is valid JSON that is NOT a whole
// plan: the extractable object carries none of the three structural keys a
// plan document must have. A scenario object alone is the shape the repair
// contract most plausibly degrades to; it was NOT observed in the 2026-09-01
// journals, so this is a guard, not a fix for a measured failure.
func IsPlanFragment(raw string) bool {
	js := extractJSONObject(raw)
	if js == "" {
		return false
	}
	for _, key := range []string{`"levels"`, `"scenarios"`, `"bias"`} {
		if strings.Contains(js, key) {
			return false
		}
	}
	// It is an object, and it has none of the plan's structural keys.
	return true
}
