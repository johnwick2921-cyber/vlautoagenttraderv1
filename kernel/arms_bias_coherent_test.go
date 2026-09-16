// D2 / E1 — a plan must be able to trade its own bias.
//
// NY 2026-09-03 v7 authored a long AND a short scenario on the identical level
// 29543.75; both confirms went true at 11:58 CT; only the SHORT carried an arm.
// With strict live the decision path is closed, so arms are the only entry: the
// plan's own long bias had no way to reach the market. The validator accepted
// it, which is the hole this pins.

package kernel

import (
	"fmt"
	"strings"
	"testing"
)

// The real NY 2026-09-03 v7, read from the stored plan:
//
//	S1 long  breakout_retest  arm=none   ← un-armable AND shadowed
//	S2 long  reclaim          arm=none   ← un-armable
//	S3 short reject           arm=1      ← the only armable+live play it wrote
//
// So v7's long side had TWO scenarios and neither COULD be armed. The fixture
// below models the same failure with an armable long, which is the harder case
// to catch (the model could have armed and did not); the un-armable case v7
// actually hit is pinned in TestBiasArmWarningNamesTheUnarmableCondition.
//
// nyV7Level is the level both sides were authored on.
const nyV7Level = 29543.75

// nyV7Doc builds the v7 shape: long bias, a short reject that is armed and a
// long play on the same level that is not.
func nyV7Doc(armTheLong bool) *PlanDoc {
	d := lawDoc("reject", "touch", func(s *PlanScenario) {
		s.Confirm.RefPrice = nyV7Level
		s.Trigger = fmt.Sprintf("reject from %.2f", nyV7Level)
		s.Invalid = fmt.Sprintf("5m close above %.2f", nyV7Level)
		s.Arm = &PlanArmSpec{
			Enabled: true,
			Entry:   nyV7Level, Stop: nyV7Level + 20, Target: nyV7Level - 60,
		}
	})
	// Both scenarios sit on the SAME level, as v7 did, so the level table and
	// the death condition are re-anchored to it rather than lawDoc's default.
	d.Levels = []PlanLevel{
		{Price: nyV7Level, Label: "OR-L", Grade: "A"},
		{Price: nyV7Level - 100, Label: "PDL", Grade: "A"},
	}
	d.DeathCondition = fmt.Sprintf("5m close below %.2f kills the plan", nyV7Level-100)
	d.DeathStructured = &PlanCondition{Price: nyV7Level - 100, Side: "below", Rule: "5m_close"}
	d.Scenarios[0].TargetChain = []float64{nyV7Level - 100}

	// The plan's stated bias is LONG — this is the v7 fact that matters.
	d.Bias.Direction = "long"

	long := PlanScenario{
		ID: "S2", Condition: "breakup_continue", Direction: "long",
		Trigger: fmt.Sprintf("continue above %.2f", nyV7Level), Invalid: fmt.Sprintf("loses %.2f", nyV7Level),
		TargetChain: []float64{nyV7Level + 80}, Quality: "B",
		Confirm:   &PlanConfirm{Rule: "1x5m_close", RefPrice: nyV7Level, Side: "above"},
		Breakdown: &PlanBreakdownContinue{Level: nyV7Level, LevelLabel: "OR-L", BreakLeg: 12, EntryMode: "pullback"},
	}
	if armTheLong {
		long.Arm = &PlanArmSpec{
			Enabled: true, WaitConfirm: true,
			Entry: nyV7Level + 2, Stop: nyV7Level - 20, Target: nyV7Level + 80,
		}
	}
	d.Scenarios = append(d.Scenarios, long)
	return d
}

// E1: the v7 plan as authored must be FLAGGED — long bias, no long arm.
//
// WARN-first (owner ruling 2026-09-04): the plan is still stored. What must not
// happen is silence — before this wave nothing anywhere observed that v7 could
// not trade the direction it had just argued for.
func TestArmsPinNYv7(t *testing.T) {
	if err := ValidatePlanDocWithFacts(nyV7Doc(false), PlanFacts{}, 8, 3); err != nil {
		t.Fatalf("WARN-first: v7 must still be accepted, got %v", err)
	}
	w := BiasArmWarning(nyV7Doc(false), ResolvedConditionStatuses(nil, nil, ""))
	if !strings.Contains(w, "no long scenario carries an arm") {
		t.Fatalf("NY v7 (bias=long, only the SHORT armed) must warn; got %q", w)
	}
	if !strings.Contains(w, "arm(s) authored on the other side") {
		t.Fatalf("the warning must say the arms went the other way — that is the v7 mistake: %s", w)
	}
}

// The same plan with the long armed must pass — the rule must not reject a plan
// that CAN trade its bias.
func TestArmsPinNYv7PassesOnceTheLongIsArmed(t *testing.T) {
	if err := ValidatePlanDocWithFacts(nyV7Doc(true), PlanFacts{}, 8, 3); err != nil {
		t.Fatalf("with the long armed the plan must validate: %v", err)
	}
}

// E6: neutral bias is exempt — there is no direction to be incoherent with.
func TestNeutralBiasNeedsNoArm(t *testing.T) {
	d := nyV7Doc(false)
	d.Bias.Direction = "neutral"
	if err := ValidatePlanDocWithFacts(d, PlanFacts{}, 8, 3); err != nil {
		t.Fatalf("neutral bias with no arms must be accepted: %v", err)
	}
}

// A short-biased plan whose only arm is a long is the mirror case, and must
// reject with the mirrored wording.
func TestShortBiasWithOnlyALongArmRejects(t *testing.T) {
	d := nyV7Doc(true)
	d.Bias.Direction = "short"
	d.Scenarios[0].Arm.Enabled = false // drop the short arm, keep the long one
	if err := ValidatePlanDocWithFacts(d, PlanFacts{}, 8, 3); err != nil {
		t.Fatalf("WARN-first: must still be accepted, got %v", err)
	}
	w := BiasArmWarning(d, ResolvedConditionStatuses(nil, nil, ""))
	if !strings.Contains(w, "no short scenario carries an arm") {
		t.Fatalf("bias=short with only a long arm must warn, got: %q", w)
	}
}
