package trader

import (
	"sync"

	"nofx/telemetry"
)

// ── W-ONE-BUTTON M2 site 5 — planner claims are refused while held ─────────
//
// A planner read is a 5–20 minute AI call, and the installation gate waits for
// planner_in_flight to reach zero. A hold therefore refuses every NEW planner
// or weekly read BEFORE its claim (so the in-flight set can only drain): no
// plan row, no budget spent, no fail-closed NO-TRADE marker. Reads already in
// flight are left to finish; the gate waits for them.

// maintenanceHoldPlanReason is the owner-facing refusal on the plan card. No
// internals, no duration promise.
const maintenanceHoldPlanReason = "an update is in progress — plan reads resume when it completes"

// plannerHoldNoted dedupes the refusal (count + WARN) to once per read key per
// hold: key → the hold reason it was logged under.
var plannerHoldNoted sync.Map

func resetPlannerHoldNotedForTest() {
	plannerHoldNoted.Range(func(k, _ any) bool { plannerHoldNoted.Delete(k); return true })
}

// refusePlannerClaimWhileHeld reports whether the maintenance hold refuses the
// read identified by key; what names it for the log ("planner read",
// "weekly read").
func (at *AutoTrader) refusePlannerClaimWhileHeld(key, what string) bool {
	reason, held := MaintenanceHeld()
	if !held {
		plannerHoldNoted.Delete(key)
		return false
	}
	if prev, ok := plannerHoldNoted.Load(key); !ok || prev.(string) != reason {
		plannerHoldNoted.Store(key, reason)
		telemetry.IncGateBlock(at.id, "maintenance_hold")
		at.logWarnf("🔒 maintenance hold: %s %s REFUSED — %s. No plan written, no budget spent.", what, key, reason)
	}
	return true
}
