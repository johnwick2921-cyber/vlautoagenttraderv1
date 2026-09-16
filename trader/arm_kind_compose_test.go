// D3 — the composer derives the entry TYPE from the condition and refuses a
// kind that contradicts it. Before this, every single-leg arm was authored
// Kind:"limit" unconditionally, so a reclaim (which is only valid once price
// trades back THROUGH the level) would have rested as a limit on the wrong side
// of the move.

package trader

import (
	"strings"
	"testing"

	"nofx/kernel"
)

func TestArmLegKindDerivedFromCondition(t *testing.T) {
	cases := map[string]string{
		"reclaim":            kernel.ArmKindStopEntry, // buy stop above the trigger
		"reject":             kernel.ArmKindLimit,
		"fvg_entry":          kernel.ArmKindLimit,
		"breakup_continue":   kernel.ArmKindLimit, // pullback at the broken level, as shipped
		"breakdown_continue": kernel.ArmKindLimit,
	}
	for cond, want := range cases {
		sc := kernel.PlanScenario{ID: "S1", Condition: cond}
		got, refusal := armLegKindFor(sc, kernel.PlanArmLeg{})
		if refusal != "" {
			t.Errorf("%s: unexpected refusal %q", cond, refusal)
		}
		if got != want {
			t.Errorf("%s → kind %q, want %q", cond, got, want)
		}
	}
}

// An authored kind that contradicts the condition is refused BY NAME, not
// silently corrected — a silent correction would hide a planner that has
// misunderstood the play.
func TestArmLegKindRefusesAMismatch(t *testing.T) {
	sc := kernel.PlanScenario{ID: "S2", Condition: "reject"}
	_, refusal := armLegKindFor(sc, kernel.PlanArmLeg{Kind: "stop_entry"})
	if refusal == "" {
		t.Fatal("a stop_entry authored for a reject fade must be refused")
	}
	if !strings.Contains(refusal, "reject") || !strings.Contains(refusal, "stop_entry") {
		t.Errorf("the refusal must name both halves: %s", refusal)
	}
}

// An authored kind that AGREES is kept, and an absent one is derived.
func TestArmLegKindAcceptsAgreementAndDerivesAbsent(t *testing.T) {
	sc := kernel.PlanScenario{ID: "S3", Condition: "reclaim"}
	if got, refusal := armLegKindFor(sc, kernel.PlanArmLeg{Kind: "stop_entry"}); refusal != "" || got != kernel.ArmKindStopEntry {
		t.Errorf("agreement must be kept: kind=%q refusal=%q", got, refusal)
	}
	if got, refusal := armLegKindFor(sc, kernel.PlanArmLeg{}); refusal != "" || got != kernel.ArmKindStopEntry {
		t.Errorf("absent kind must be derived: kind=%q refusal=%q", got, refusal)
	}
}

// A condition with no kind at all (non-armable) must not silently become a
// limit — the validator owns that refusal, and the composer must not paper it.
func TestArmLegKindOnNonArmableIsRefused(t *testing.T) {
	sc := kernel.PlanScenario{ID: "S4", Condition: "acceptance"}
	kind, refusal := armLegKindFor(sc, kernel.PlanArmLeg{})
	if refusal == "" {
		t.Fatal("an arm on a non-armable condition must be refused, not defaulted to limit")
	}
	if kind != "" {
		t.Errorf("no kind may be returned for a non-armable condition, got %q", kind)
	}
}

// A stop-entry row means two different things depending on the play, and the
// executor must not treat them alike:
//
//	reclaim            — stop_entry is the PRIMARY entry. The buy stop rests
//	                     immediately; waiting for a no-retest window would miss
//	                     the reclaim it exists to catch.
//	breakup_continue   — the arm rests as a pullback LIMIT; stop_entry is the
//	                     E7 no-retest FALLBACK and keeps its window.
func TestStopEntryWindowAppliesOnlyToTheFallback(t *testing.T) {
	if stopEntryNeedsRetestWindow("reclaim") {
		t.Error("a reclaim buy-stop is the primary entry — it must not wait for a no-retest window")
	}
	for _, c := range []string{"breakup_continue", "breakdown_continue", "reject", "fvg_entry"} {
		if !stopEntryNeedsRetestWindow(c) {
			t.Errorf("%s: a stop_entry row here is the E7 fallback and must keep its window", c)
		}
	}
	// A legacy row carries no condition (A30: '' means UNKNOWN, not "limit").
	// Unknown must take the SAFE branch — keep the window rather than fire a
	// stop the moment the process boots.
	if !stopEntryNeedsRetestWindow("") {
		t.Error("an unknown condition must keep the window — unknown is not permission")
	}
}
