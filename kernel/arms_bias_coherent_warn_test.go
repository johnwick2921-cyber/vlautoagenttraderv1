// D2 WARN-FIRST + (a) the armable/live vocabulary the model is told about.
//
// Measured on 171 stored directional plans, a hard reject would have refused
// 50/68 longs and 66/103 shorts. The rule is right and the magnitude is not
// shippable, so it counts first and a later ruling promotes it.
//
// 18 of those longs could NEVER have complied: their long scenarios lean on
// breakout_retest, which is both un-armable and shadowed — and the prompt never
// said so. That is what (a) fixes.

package kernel

import (
	"strings"
	"testing"
)

// The line must be DERIVED from ArmableCondition + ArmKindFor + the resolved
// statuses. A hand-kept copy would drift the moment a condition changed status.
func TestArmableConditionsLineIsDerived(t *testing.T) {
	statuses := ResolvedConditionStatuses(nil, nil, "")
	line := ArmableConditionsLine(statuses)

	for _, c := range KnownConditions() {
		armable := ArmableCondition(c) || ArmKindFor(c) != ""
		mentioned := strings.Contains(line, c)
		if armable && !mentioned {
			t.Errorf("%q is armable but the prompt line never names it: %s", c, line)
		}
	}
	// fvg_entry is armable but SHADOWED (owner ruling 2026-08-31) — the line
	// must not offer it as if it were live.
	if strings.Contains(line, "fvg_entry") && !strings.Contains(strings.ToLower(line), "shadowed") {
		t.Errorf("fvg_entry is shadowed and the line must say so: %s", line)
	}
	// breakout_retest is the one the 18 long plans leaned on: un-armable AND
	// shadowed. It must be named as unavailable, not silently omitted.
	if !strings.Contains(line, "breakout_retest") {
		t.Errorf("breakout_retest must be named as unavailable — 18 long plans leaned on it: %s", line)
	}
}

// The warning names the condition the model actually chose, so the message is
// actionable rather than a restatement of the rule.
func TestBiasArmWarningNamesTheUnarmableCondition(t *testing.T) {
	// v7's real long side: breakout_retest (un-armable AND shadowed) and
	// reclaim (un-armable). Neither could carry an arm.
	d := nyV7Doc(false)
	d.Scenarios[1].Condition = "breakout_retest"
	d.Scenarios[1].Breakdown = nil

	w := BiasArmWarning(d, ResolvedConditionStatuses(nil, nil, ""))
	if w == "" {
		t.Fatal("a long-biased plan whose only long play is un-armable must warn")
	}
	// And the reclaim case v7 also wrote must warn by name too.
	r := nyV7Doc(false)
	r.Scenarios[1].Condition = "reclaim"
	r.Scenarios[1].Breakdown = nil
	if rw := BiasArmWarning(r, ResolvedConditionStatuses(nil, nil, "")); !strings.Contains(rw, "reclaim") {
		t.Errorf("a long plan leaning on reclaim must name it: %s", rw)
	}

	for _, want := range []string{"long", "breakout_retest", "S2"} {
		if !strings.Contains(w, want) {
			t.Errorf("warning must name %q: %s", want, w)
		}
	}
}

// WARN-FIRST: the plan is STORED. No reject in this wave.
func TestBiasCoherentArmsDoesNotRejectYet(t *testing.T) {
	if err := ValidatePlanDocWithFacts(nyV7Doc(false), PlanFacts{}, 8, 3); err != nil {
		t.Fatalf("D2 is WARN-first — the plan must still be accepted: %v", err)
	}
}

// But the v7 shape must still WARN, or the counter never moves.
func TestNYv7StillWarns(t *testing.T) {
	if w := BiasArmWarning(nyV7Doc(false), ResolvedConditionStatuses(nil, nil, "")); w == "" {
		t.Fatal("NY v7 (bias=long, only the short armed) must warn")
	}
	if w := BiasArmWarning(nyV7Doc(true), ResolvedConditionStatuses(nil, nil, "")); w != "" {
		t.Fatalf("a plan that arms its own bias must NOT warn: %s", w)
	}
}

// Neutral stays exempt.
func TestNeutralBiasNeverWarns(t *testing.T) {
	d := nyV7Doc(false)
	d.Bias.Direction = "neutral"
	if w := BiasArmWarning(d, ResolvedConditionStatuses(nil, nil, "")); w != "" {
		t.Fatalf("neutral bias must never warn: %s", w)
	}
}
