// D3 (arms-follow-bias, 2026-09-04) — ONE table decides the entry TYPE of an
// armed scenario, and the prompt and the composer both read it.
//
// This encodes NO market belief. It is mechanics: a play whose entry rests AT a
// price is a LIMIT; a play that only becomes valid once price travels BEYOND a
// trigger is a STOP-ENTRY. Which plays the planner chooses, and in which
// direction, is untouched by this file.
//
// Why it exists: stop_entry was built (armed_executor.go), wired
// (tcp_trader.go PlaceStopEntry), and its seam was ON — and it had been used
// ZERO times, because the planner prompt never named it. Built is not wired and
// wired is not used. With the type derived from the condition rather than taken
// on trust from the model, a continuation play cannot silently become a limit
// resting at a price the tape has already left behind.

package kernel

import (
	"fmt"
	"strings"
)

// The two entry types an arm can take. These are the values store.ArmedOrderDB
// carries in Kind and the executor branches on.
const (
	ArmKindLimit     = "limit"
	ArmKindStopEntry = "stop_entry"
)

// ArmKindFor returns the entry type an armable condition must use, or "" when
// the condition cannot be armed at all. "" is deliberate: a caller must not
// receive a plausible default for a play that has no arm.
//
// Keep this in step with ArmableCondition — TestEveryArmableConditionHasAKind
// fails if the two drift apart.
func ArmKindFor(condition string) string {
	switch strings.ToLower(strings.TrimSpace(condition)) {
	// Rests AT a price: the entry is the level itself.
	//   reject                    — the anchor, one tick into the trade's favour
	//   fvg_entry                 — the gap edge or CE
	//   sweep_reclaim             — the split arm's legs rest at the sweep ref
	//   breakup/breakdown_continue — the PULLBACK limit at the broken level,
	//       chained on confirm leg 1. This is the shipped waterfall design
	//       (ArmSpecValid requires entry_mode=pullback) and it is NOT flipped
	//       here: the executor's stop_entry branch remains the no-retest
	//       FALLBACK it has always been, not an authored primary. Changing that
	//       would be a live behaviour change, not a labelling one.
	case "reject", "fvg_entry", "sweep_reclaim", "breakup_continue", "breakdown_continue":
		return ArmKindLimit

	// Triggers BEYOND a price. A reclaim is only valid once price travels back
	// through the level, so a resting limit would fill on the wrong side of the
	// move — a BUY STOP above the trigger (SELL STOP below, short) is the only
	// honest expression of it.
	//
	// Owner ruling 2026-09-04 (B), the gate change that makes LONGS armable: 19
	// long plans were stranded because every long play they wrote (reclaim,
	// breakout_retest) was un-armable, so a long-biased plan had no way to
	// reach the market with the decision path closed.
	case "reclaim":
		return ArmKindStopEntry
	}
	return ""
}

// ArmKindMismatch refuses an authored kind that contradicts the condition. An
// EMPTY authored kind is not a mismatch: the composer derives it, which is the
// normal path. Only a kind that actively disagrees is refused, and the refusal
// names both halves so the log says what the planner asked for and what the
// condition requires.
func ArmKindMismatch(condition, authored string) error {
	want := ArmKindFor(condition)
	got := strings.ToLower(strings.TrimSpace(authored))
	if got == "" || want == "" || got == want {
		return nil
	}
	return fmt.Errorf("%s authored for a %s — %s requires %s (entry type follows the condition)",
		got, strings.ToLower(strings.TrimSpace(condition)), condition, want)
}

// ArmableConditionsPipe renders the armable set as the pipe-joined string the
// prompt sentence and the class-38 contract row both quote.
//
// D-25 (2026-09-04): both halves used to TYPE the same list, so when reclaim
// joined the set they went stale together and the guard passed while confirming
// the stale text. A contract that agrees with the thing it checks — because the
// same hand typed both — checks nothing. Derived here, one source, and adding a
// condition to ArmableCondition needs no edit in either place.
func ArmableConditionsPipe() string {
	out := make([]string, 0, 8)
	for _, c := range KnownConditions() {
		if ArmableCondition(c) {
			out = append(out, c)
		}
	}
	return strings.Join(out, "|")
}

// NonArmableConditionsPipe is the complement: conditions that can NEVER carry
// an arm. Derived for the same reason as ArmableConditionsPipe — the prompt
// used to type "breakout_retest|reclaim|hold|acceptance NEVER arm", which
// contradicted the armable set outright once reclaim joined it.
func NonArmableConditionsPipe() string {
	out := make([]string, 0, 8)
	for _, c := range KnownConditions() {
		if !ArmableCondition(c) && c != "sweep_reclaim" {
			out = append(out, c)
		}
	}
	return strings.Join(out, "|")
}
