// D3 — one function decides limit vs stop-entry, and both the prompt table and
// the executor read it.
//
// The mapping is MECHANICS, not a market belief: a play that rests AT a level
// is a limit; a play that only becomes valid once price travels BEYOND a
// trigger is a stop-entry. Encoding it once is what stops the prompt promising
// one thing while the composer does another.

package kernel

import "testing"

func TestArmKindForRestsAtALevelIsLimit(t *testing.T) {
	// Waterfalls stay as shipped (owner ruling 2026-09-04): the arm is a
	// PULLBACK LIMIT resting at the broken level, chained on confirm leg 1.
	// stop_entry remains the executor's no-retest FALLBACK, not the primary —
	// flipping it would change live behaviour, not just labelling.
	for _, c := range []string{"reject", "fvg_entry", "sweep_reclaim", "breakup_continue", "breakdown_continue"} {
		if got := ArmKindFor(c); got != ArmKindLimit {
			t.Errorf("ArmKindFor(%q) = %q, want %q — it rests AT a price", c, got, ArmKindLimit)
		}
	}
}

// reclaim is the stop-entry play, and the one that makes a LONG armable: a buy
// stop above the reclaim trigger (sell stop below, short). Owner ruling
// 2026-09-04 (B) — the gate change that unstrands 19 long plans.
func TestArmKindForReclaimIsStopEntry(t *testing.T) {
	if got := ArmKindFor("reclaim"); got != ArmKindStopEntry {
		t.Errorf("ArmKindFor(reclaim) = %q, want %q — it triggers BEYOND the reclaim level", got, ArmKindStopEntry)
	}
	if !ArmableCondition("reclaim") {
		t.Error("reclaim must be armable — this is the gate change that makes longs armable")
	}
}

// Case and padding must not change the answer: the condition arrives from model
// JSON (canon 28 — canonicalize where the value enters).
func TestArmKindForIsCanonicalized(t *testing.T) {
	if ArmKindFor("  ReClaim  ") != ArmKindStopEntry {
		t.Error("ArmKindFor must canonicalize case and padding")
	}
	if ArmKindFor("  BreakUp_Continue  ") != ArmKindLimit {
		t.Error("canonicalization must hold for the limit side too")
	}
}

// A condition that cannot be armed at all has no kind — the caller must not get
// a plausible default it would then act on.
func TestArmKindForNonArmableIsEmpty(t *testing.T) {
	// acceptance/hold stay un-armable pending a separate ruling.
	for _, c := range []string{"acceptance", "hold", "", "nonsense"} {
		if got := ArmKindFor(c); got != "" {
			t.Errorf("ArmKindFor(%q) = %q, want \"\" — non-armable conditions have no kind", c, got)
		}
	}
}

// Every armable condition must have a kind, or the table and the whitelist have
// drifted apart — which is exactly how the prompt and the composer disagree.
func TestEveryArmableConditionHasAKind(t *testing.T) {
	for _, c := range []string{"fvg_entry", "reject", "breakdown_continue", "breakup_continue", "sweep_reclaim", "reclaim"} {
		if !ArmableCondition(c) && c != "sweep_reclaim" {
			t.Fatalf("test premise wrong: %q is not armable", c)
		}
		if ArmKindFor(c) == "" {
			t.Errorf("%q is armable but ArmKindFor gives no kind — table drifted from the whitelist", c)
		}
	}
}

// The mismatch the owner asked the executor to refuse.
func TestArmKindMismatchIsNamed(t *testing.T) {
	err := ArmKindMismatch("reject", "stop_entry")
	if err == nil {
		t.Fatal("a stop_entry authored for a reject fade must be refused")
	}
	if got := err.Error(); got == "" {
		t.Fatal("the refusal must name what was wrong")
	}
	if ArmKindMismatch("reclaim", "stop_entry") != nil {
		t.Error("the correct kind must not be refused")
	}
	// And the waterfall must be refused a stop_entry PRIMARY — it is a
	// pullback limit; stop_entry there is the executor's fallback, not an
	// authored kind.
	if ArmKindMismatch("breakup_continue", "stop_entry") == nil {
		t.Error("an authored stop_entry on a waterfall must be refused — it rests as a pullback limit")
	}
	// An unset kind is not a mismatch — the composer derives it.
	if ArmKindMismatch("reject", "") != nil {
		t.Error("an absent kind is derived, not refused")
	}
}
