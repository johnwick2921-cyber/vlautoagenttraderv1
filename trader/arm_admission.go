package trader

import (
	"strconv"
	"strings"
	"time"

	"nofx/store"
	"nofx/telemetry"
)

// ── W-EXEC-TRUTH W0 (G1 + a) — the armed path at its send point ────────────
//
// G1: an 'armed' row a later pass placed without re-running the authoring
// gates. Proven by execution before this wave: a daily_force_flat refusal and
// a limit placement for the SAME leg in ONE pass. The authoring loop now
// records the legs whose every gate passed THIS pass (armAdmission) and the
// placement places only those. Then admitEntry — the one admission chain —
// runs for the row at its send point.

// armAdmission is the set of legs this pass's authoring gates admitted, keyed
// plan|scenario|leg (version-insensitive: the row's version is refreshed each
// pass by the authoring loop).
type armAdmission map[string]bool

func armAdmitKey(planID, scenario string, leg int) string {
	return planID + "|" + scenario + "|" + strconv.Itoa(leg)
}

func (a armAdmission) admit(planID, scenario string, leg int) {
	a[armAdmitKey(planID, scenario, leg)] = true
}

// armAdmitted is the placement-time check for one armed row: G1 first, then
// the one admission chain. false = do not place (the row stays armed).
func (at *AutoTrader) armAdmitted(r store.ArmedOrderDB, side string, price float64, now time.Time, admitted armAdmission) bool {
	key := r.PlanID + ":" + r.Scenario + ":leg" + strconv.Itoa(r.LegIndex+1)
	// FAIL-CLOSED (CTO M2): no admitted set means no authoring pass stands in
	// front of this placement — nothing is admitted. A nil default would let a
	// direct caller (W3's event-driven pass) bypass G1 silently.
	if admitted == nil || !admitted[armAdmitKey(r.PlanID, r.Scenario, r.LegIndex)] {
		why := "not admitted this pass"
		if admitted == nil {
			why = "no authoring pass"
		}
		if at.admitLast.changed("arm-g1|"+key, why) {
			at.logWarnf("⏸ armed %s leg %d NOT placed — %s (G1); it stays armed until a pass admits it", r.Scenario, r.LegIndex+1, why)
			// counted once per change, like every arm refusal
			telemetry.IncGateBlock(at.id, "arm_not_admitted")
		}
		return false
	}
	at.admitLast.clear("arm-g1|" + key)
	action := "open_long"
	if strings.EqualFold(side, "short") {
		action = "open_short"
	}
	if _, refused := at.admitEntry(admitIntent{
		Path: admitArm, Symbol: at.futuresSymbol(), Action: action, Now: now, Key: key, Price: price,
	}); refused {
		return false
	}
	return true
}
