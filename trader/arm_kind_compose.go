// D3 (arms-follow-bias, 2026-09-04) — the composer derives an arm's entry TYPE
// from the scenario's condition, using the same kernel.ArmKindFor table the
// planner prompt is built from. One table, two readers.
//
// Before this, every single-leg arm was authored Kind:"limit" unconditionally
// (armed_executor.go). That was harmless while the only armable conditions
// rested at a level — and wrong the moment reclaim became armable, because a
// reclaim is only valid once price trades back THROUGH the level: a resting
// limit there fills on the wrong side of the move.

package trader

import (
	"fmt"

	"nofx/kernel"
)

// armLegKindFor returns the entry type for one arm leg, or a REFUSAL reason.
//
// An authored kind that agrees is kept; an absent one is derived; one that
// contradicts the condition is refused by name rather than silently corrected,
// because a silent correction hides a planner that has misunderstood the play.
// A condition with no kind at all is refused too — the composer must not paper
// over an arm the validator should have caught.
func armLegKindFor(sc kernel.PlanScenario, leg kernel.PlanArmLeg) (kind string, refusal string) {
	want := kernel.ArmKindFor(sc.Condition)
	if want == "" {
		return "", fmt.Sprintf("%s condition %q has no arm kind — not armable", sc.ID, sc.Condition)
	}
	if err := kernel.ArmKindMismatch(sc.Condition, leg.Kind); err != nil {
		return "", fmt.Sprintf("%s %v", sc.ID, err)
	}
	return want, ""
}

// stopEntryNeedsRetestWindow reports whether a stop_entry ledger row is the E7
// no-retest FALLBACK (window applies) rather than a play whose PRIMARY entry is
// a stop (window must not apply).
//
// The distinction matters because the same Kind means two things: a reclaim's
// buy stop is the entry itself and must rest immediately, while a waterfall's
// stop entry only exists because the pullback never came.
//
// An UNKNOWN condition (legacy rows carry ”) keeps the window. Unknown is not
// permission: the safe branch is to wait, never to fire a stop on boot.
func stopEntryNeedsRetestWindow(condition string) bool {
	return kernel.ArmKindFor(condition) != kernel.ArmKindStopEntry
}
