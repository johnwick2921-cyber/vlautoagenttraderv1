// D-25 — the class-38 contract row must DERIVE the armable set, not type it.
//
// prompt_contract.go pinned "fvg_entry|reject|breakdown_continue|breakup_continue"
// and planner_prompt.go rendered the same typed list. Both halves were stale in
// the SAME way after reclaim joined the set, so the guard passed while
// confirming the stale text — a contract that agrees with the thing it is
// checking, because both were typed by the same hand.

package kernel

import (
	"os"
	"strings"
	"testing"
)

// The derived list must contain every armable condition, from the one source.
func TestArmableConditionsPipeIsDerived(t *testing.T) {
	pipe := ArmableConditionsPipe()
	for _, c := range KnownConditions() {
		if !ArmableCondition(c) {
			continue
		}
		if !strings.Contains(pipe, c) {
			t.Errorf("%q is armable but missing from the derived list: %s", c, pipe)
		}
	}
	// reclaim is the condition that exposed this — it must be present.
	if !strings.Contains(pipe, "reclaim") {
		t.Errorf("reclaim joined the armable set and must appear: %s", pipe)
	}
	// A non-armable condition must never appear.
	if strings.Contains(pipe, "acceptance") || strings.Contains(pipe, "hold") {
		t.Errorf("non-armable conditions leaked into the list: %s", pipe)
	}
}

// The contract row and the prompt must both use the DERIVED string, so adding a
// condition to ArmableCondition needs no edit in either place.
func TestContractRowUsesTheDerivedArmableSet(t *testing.T) {
	pipe := ArmableConditionsPipe()

	var row *PromptContract
	for i := range PromptContracts() {
		if strings.Contains(PromptContracts()[i].Rule, "armable conditions") {
			row = &PromptContracts()[i]
			break
		}
	}
	if row == nil {
		t.Fatal("the armable-conditions contract row is gone")
	}
	found := false
	for _, frag := range row.MustAppear {
		if strings.Contains(frag, pipe) {
			found = true
		}
	}
	if !found {
		t.Fatalf("the contract row must expect the DERIVED set %q, got %v", pipe, row.MustAppear)
	}

	// And the rendered prompt must satisfy it from the same source.
	if !strings.Contains(plannerOutputContract(8, 3, true, true), pipe) {
		t.Errorf("the prompt must render the derived set %q", pipe)
	}
}

// The other half of the pin: a TYPED condition list anywhere in the prompt or
// the contract fails, because that is how both halves went stale together.
func TestNoTypedArmableListSurvives(t *testing.T) {
	for _, f := range []string{"prompt_contract.go", "planner_prompt.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(src)
		// The pre-ruling set, in any order that omits reclaim, is the specific
		// staleness this pin exists for.
		for _, typed := range []string{
			"fvg_entry|reject|breakdown_continue|breakup_continue",
			"breakout_retest|reclaim|hold|acceptance",
		} {
			if strings.Contains(text, typed) {
				t.Errorf("%s still types the condition set %q — derive it from ArmableCondition", f, typed)
			}
		}
	}
}

// Adding a condition must not require touching the contract: the row's
// expectation and the prompt's text come from the same call, so they move
// together by construction.
func TestContractFollowsTheSetWithoutEditing(t *testing.T) {
	pipe := ArmableConditionsPipe()
	if !strings.Contains(pipe, "reclaim") {
		t.Fatal("premise: reclaim is in the set")
	}
	var rowFrag string
	for _, c := range PromptContracts() {
		if strings.Contains(c.Rule, "armable conditions") {
			rowFrag = strings.Join(c.MustAppear, " ")
		}
	}
	// The row must quote the CURRENT set — including the condition that was
	// added after the row was written.
	if !strings.Contains(rowFrag, "reclaim") {
		t.Errorf("the contract row did not follow the set: %s", rowFrag)
	}
	if err := ValidatePromptContracts(plannerOutputContract(8, 3, true, true)); err != nil {
		t.Errorf("the rendered prompt must satisfy every contract row: %v", err)
	}
}
