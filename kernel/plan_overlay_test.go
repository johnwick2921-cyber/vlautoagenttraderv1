package kernel

import (
	"encoding/json"
	"testing"
)

// P5.1 — RFC-6902 applier tests. These lock the plan-grammar ops (add/remove/
// replace/test on levels/bias/scenarios), the append token "-", the test-op
// concurrency guard, and the skip-bad-overlay read-time robustness.

const baseDoc = `{"reasoning":"r","bias":{"direction":"long","conviction":"medium","flip_condition":"x"},"levels":[{"price":100,"label":"PDH","grade":"A","instruction":"fade"}],"scenarios":[{"id":"S1","trigger":"t","condition":"reclaim","direction":"long","target_chain":[110],"invalid":"i","quality":"A"}],"no_trade":["lunch"],"death_condition":"d"}`

func mustApply(t *testing.T, doc, patch string) map[string]interface{} {
	t.Helper()
	out, err := ApplyPatchStrict([]byte(doc), patch)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("result not JSON object: %v", err)
	}
	return m
}

func TestApplyReplaceScalar(t *testing.T) {
	m := mustApply(t, baseDoc, `[{"op":"replace","path":"/bias/direction","value":"short"}]`)
	bias := m["bias"].(map[string]interface{})
	if bias["direction"] != "short" {
		t.Fatalf("direction = %v want short", bias["direction"])
	}
}

func TestApplyAddLevelAppend(t *testing.T) {
	patch := `[{"op":"add","path":"/levels/-","value":{"price":90,"label":"PDL","grade":"B","instruction":"reclaim-long"}}]`
	m := mustApply(t, baseDoc, patch)
	levels := m["levels"].([]interface{})
	if len(levels) != 2 {
		t.Fatalf("levels len = %d want 2", len(levels))
	}
	last := levels[1].(map[string]interface{})
	if last["label"] != "PDL" || last["price"].(float64) != 90 {
		t.Fatalf("appended level wrong: %v", last)
	}
}

func TestApplyRemoveLevel(t *testing.T) {
	m := mustApply(t, baseDoc, `[{"op":"remove","path":"/levels/0"}]`)
	if len(m["levels"].([]interface{})) != 0 {
		t.Fatalf("level not removed")
	}
}

func TestApplyReplaceNestedInArray(t *testing.T) {
	m := mustApply(t, baseDoc, `[{"op":"replace","path":"/scenarios/0/quality","value":"A+"}]`)
	sc := m["scenarios"].([]interface{})[0].(map[string]interface{})
	if sc["quality"] != "A+" {
		t.Fatalf("quality = %v want A+", sc["quality"])
	}
}

func TestApplyAddToTargetChain(t *testing.T) {
	m := mustApply(t, baseDoc, `[{"op":"add","path":"/scenarios/0/target_chain/-","value":120}]`)
	tc := m["scenarios"].([]interface{})[0].(map[string]interface{})["target_chain"].([]interface{})
	if len(tc) != 2 || tc[1].(float64) != 120 {
		t.Fatalf("target_chain = %v", tc)
	}
}

func TestTestOpGuardPasses(t *testing.T) {
	// test matches → the following replace applies.
	patch := `[{"op":"test","path":"/bias/direction","value":"long"},{"op":"replace","path":"/bias/conviction","value":"high"}]`
	m := mustApply(t, baseDoc, patch)
	if m["bias"].(map[string]interface{})["conviction"] != "high" {
		t.Fatalf("guarded replace did not apply")
	}
}

func TestTestOpGuardFailsAtomically(t *testing.T) {
	// test mismatches → the whole patch is rejected, NOTHING applied.
	patch := `[{"op":"test","path":"/bias/direction","value":"short"},{"op":"replace","path":"/bias/conviction","value":"high"}]`
	if _, err := ApplyPatchStrict([]byte(baseDoc), patch); err == nil {
		t.Fatal("expected test-op failure to reject the patch")
	}
}

func TestUnsupportedOpRejected(t *testing.T) {
	if _, err := ApplyPatchStrict([]byte(baseDoc), `[{"op":"move","path":"/levels/0","from":"/levels/1"}]`); err == nil {
		t.Fatal("move must be rejected")
	}
}

func TestApplyOverlayPatchesSkipsBad(t *testing.T) {
	patches := []string{
		`[{"op":"replace","path":"/bias/direction","value":"short"}]`, // good
		`[{"op":"replace","path":"/does/not/exist","value":1}]`,       // bad → skipped
		`[{"op":"add","path":"/no_trade/-","value":"FOMC"}]`,          // good
	}
	final, errs := ApplyOverlayPatches([]byte(baseDoc), patches)
	if len(errs) != 1 {
		t.Fatalf("expected 1 skipped-overlay error, got %d", len(errs))
	}
	var m map[string]interface{}
	_ = json.Unmarshal(final, &m)
	if m["bias"].(map[string]interface{})["direction"] != "short" {
		t.Fatal("good overlay 1 not applied")
	}
	nt := m["no_trade"].([]interface{})
	if len(nt) != 2 || nt[1] != "FOMC" {
		t.Fatalf("good overlay 3 not applied: %v", nt)
	}
}

func TestAppliedResultStillValidatesAsPlanDoc(t *testing.T) {
	patch := `[{"op":"add","path":"/levels/-","value":{"price":90,"label":"PDL","grade":"B","instruction":"reclaim-long"}}]`
	out, err := ApplyPatchStrict([]byte(baseDoc), patch)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var doc PlanDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal PlanDoc: %v", err)
	}
	if err := ValidatePlanDoc(&doc); err != nil {
		t.Fatalf("patched doc must still validate: %v", err)
	}
	if len(doc.Levels) != 2 {
		t.Fatalf("levels = %d want 2", len(doc.Levels))
	}
}

func TestLeadingZeroIndexRejected(t *testing.T) {
	if _, err := ApplyPatchStrict([]byte(baseDoc), `[{"op":"remove","path":"/levels/00"}]`); err == nil {
		t.Fatal("leading-zero index must be rejected")
	}
}

// hardening — RFC-6901 canonical form: "-0" (signed) is not a valid index.
func TestMinusZeroIndexRejected(t *testing.T) {
	if _, err := ApplyPatchStrict([]byte(baseDoc), `[{"op":"remove","path":"/levels/-0"}]`); err == nil {
		t.Fatal("signed '-0' index must be rejected (not index 0)")
	}
}

// hardening — ValidatePlanDoc rejects non-positive level + target prices, closing
// the armor gap for the whole-array replace / negative-price patch shapes.
func TestValidateRejectsNonPositivePrices(t *testing.T) {
	neg := PlanDoc{
		Reasoning: "r", Bias: PlanBias{Direction: "long"}, DeathCondition: "d",
		Levels:    []PlanLevel{{Price: -100, Grade: "A", Label: "X", Instruction: "fade"}},
		Scenarios: []PlanScenario{{ID: "S1", Condition: "reclaim", Direction: "long", Quality: "A"}},
	}
	if err := ValidatePlanDoc(&neg); err == nil {
		t.Fatal("negative level price must be rejected")
	}
	badTarget := PlanDoc{
		Reasoning: "r", Bias: PlanBias{Direction: "long"}, DeathCondition: "d",
		Levels:    []PlanLevel{{Price: 100, Grade: "A", Label: "X", Instruction: "fade"}},
		Scenarios: []PlanScenario{{ID: "S1", Condition: "reclaim", Direction: "long", Quality: "A", TargetChain: []float64{-5}}},
	}
	if err := ValidatePlanDoc(&badTarget); err == nil {
		t.Fatal("negative target_chain price must be rejected")
	}
}
