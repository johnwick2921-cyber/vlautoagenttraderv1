package kernel

import (
	"strings"
	"testing"
)

// ITEM 6 (2026-08-17) — EVERY DECISION MUST CARRY ITS "WHY".
//
// The dashboard could not answer "why did it wait?" even though the answer was
// in the same row. The model writes its analysis in the <reasoning> block and
// leaves the decision JSON's own `reasoning` empty — and that field is what the
// UI renders. Live evidence before the fix: the six newest decision_records rows
// all had a full <reasoning> block in raw_response and `"reasoning": ""` in
// decision_json.
//
// Nothing here invents text: it is copied from the same response, only when the
// model left the field empty.

// A realistic wait response, shaped like the ones in the DB.
const waitResponseWithReasoning = `<reasoning>
MNQ is flat at 30195.75, sitting almost exactly on the session POC (30195.62).
Price is between EQH 30207.25 and ONH 30245.00 — a no-chase zone with no edge
either way, so the correct action is to stand aside and wait for acceptance.
</reasoning>
<decision>
[{"action":"wait","symbol":"MNQ","entry":0,"stop_loss":0,"take_profit":0,"reasoning":"","leverage":1,"confidence":0}]
</decision>`

func TestWaitDecisionKeepsItsReasoning(t *testing.T) {
	full, err := parseFullDecisionResponse(waitResponseWithReasoning, 50000, 5, 5, 5, 1, 3, 65, 20, nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(full.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(full.Decisions))
	}
	got := full.Decisions[0].Reasoning
	if strings.TrimSpace(got) == "" {
		t.Fatal("a wait decision stored an EMPTY reasoning while the response explained itself — this is the bug")
	}
	if !strings.Contains(got, "no-chase zone") {
		t.Errorf("the stored reasoning is not the model's own text: %q", got)
	}
}

// A decision that DID fill in its own reasoning must keep it verbatim — the
// backfill is a floor, never an override.
func TestExplicitReasoningIsNeverOverwritten(t *testing.T) {
	resp := `<reasoning>
Broad context: trend up, POC below, buyers in control.
</reasoning>
<decision>
[{"action":"open_long","symbol":"MNQ","entry":30200,"stop_loss":30150,"take_profit":30400,"reasoning":"long the ONH retest, stop under the POC","leverage":1,"confidence":72,"position_size_usd":5000}]
</decision>`
	// The decision may or may not pass the risk gate depending on sizing rules;
	// either way its reasoning must survive, so we do not require err == nil.
	full, _ := parseFullDecisionResponse(resp, 50000, 5, 5, 5, 1, 1.5, 50, 20, nil)
	if len(full.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(full.Decisions))
	}
	if got := full.Decisions[0].Reasoning; got != "long the ONH retest, stop under the POC" {
		t.Errorf("the decision's own reasoning was replaced by the trace: %q", got)
	}
}

// THE REGRESSION GUARD the dispatch asked for: if the response contains a
// reasoning block, no decision parsed from it may store an empty reasoning.
func TestNoDecisionIsStoredWithoutAReasonWhenOneWasGiven(t *testing.T) {
	for name, resp := range map[string]string{
		"wait":       waitResponseWithReasoning,
		"multi-wait": strings.Replace(waitResponseWithReasoning, `[{"action":"wait"`, `[{"action":"wait","x":0},{"action":"wait"`, 1),
	} {
		full, err := parseFullDecisionResponse(resp, 50000, 5, 5, 5, 1, 3, 65, 20, nil)
		if err != nil {
			continue // a malformed variant is a different test's problem
		}
		if !strings.Contains(resp, "<reasoning>") {
			t.Fatalf("%s: fixture must carry a reasoning block", name)
		}
		for i, d := range full.Decisions {
			if strings.TrimSpace(d.Reasoning) == "" {
				t.Errorf("%s: decision %d stored an empty reasoning while raw_response contained a <reasoning> block", name, i)
			}
		}
	}
}

// No trace, no invention: an empty <reasoning> must not fabricate one.
func TestBackfillInventsNothing(t *testing.T) {
	decisions := []Decision{{Action: "wait"}}
	backfillReasoningFromCoT(decisions, "   \n  ")
	if decisions[0].Reasoning != "" {
		t.Errorf("an empty trace must leave the field empty, got %q", decisions[0].Reasoning)
	}
}
