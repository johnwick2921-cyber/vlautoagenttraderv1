package trader

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── W1b E1 + E2 — a WORKING arm under a plan that re-priced it ─────────────
//
// E1 (the dead churn guard). The arm loop built its row from the NEW leg and
// copied only the id from the ledger, then asked whether the row was working
// and whether its stop differed from the new stop — it never was, and it never
// did. A re-spec'd plan therefore never moved a resting order's bracket. The
// guard's ModifyBracket call could not have helped either: the AddOn's
// modify_bracket acts only on a FILLED entry's bracket (placedBrackets), and a
// working row is by definition an unfilled entry, so the frame is refused
// ("no live bracket") and nothing in Go reads the reply.
//
// E1 judges the plan's AUTHORED bracket, never the composed one (W1b repair):
// the stop that goes to the wire is composed at each pass from live ATR (the
// min-SL floor), the structure anchor and the obstacle target, so comparing
// the placed bracket with this pass's composition turned every version bump
// into a "re-spec" of accumulated ATR drift — a cancel, and an
// arm:respec_cancel count, for a re-spec that never happened (class 35). The
// prior version's authored leg is read from plan history through the one fold;
// when it cannot be read, E1 does not fire (absent ≠ changed) and says why once.
// E1 judges the authored ENTRY too (W1b FOLD-1, CTO): a non-market_in_zone row
// rests AT its authored entry, so a v2 that moved only the entry left the v1
// limit resting at the old price with v1's bracket until the rest cap.
//
// E2 (the moved zone). A working market_in_zone limit is keyed by (plan,
// scenario, leg), which survives a re-plan, but the zone that justified its
// price does not. Nothing compared the resting limit with the CURRENT zone, so
// only the 30-minute rest cap ever ended it.
//
// The one fix for both: CANCEL, then let the normal authoring re-arm. The
// cancel goes through the filled-arm guard (cancelSafetyFor) — a refusal sends
// nothing, WARNs once, and is retried on the next pass. The row moves to
// cancel_pending (a send is not a settlement); once the broker's persisted
// book confirms it, the next pass finds no live row, UpsertArm mints the next
// placement (seq+1) under the new version — the D15 pin blocks the mint only
// within the version the old row was placed under — and every existing gate
// judges the new placement.
//
// AT MOST ONE CANCEL PER ROW: E2 is asked first, E1 only when E2 is silent.

// armRespecCancel is one decided re-spec: the ledger reason, the counter it is
// recorded under, and the short class the WARN names.
type armRespecCancel struct {
	reason  string
	counter string
	class   string
}

// armScenarioLegs is the ONE turn of a scenario into its arm legs AS AUTHORED
// — the prices the plan doc states, before any stop composition, zone edge or
// obstacle target touches them. The arm loop takes its legs from here, and so
// does the prior-version reader (armAuthoredLegAt), so "what the plan said"
// is one function on both sides of the E1 compare. A single arm is leg 0 of a
// one-leg list; its Kind is the loop's (armLegKindFor), never needed here.
func armScenarioLegs(sc kernel.PlanScenario) []kernel.PlanArmLeg {
	if sc.Arm == nil {
		return nil
	}
	if len(sc.Arm.Legs) > 0 {
		return sc.Arm.Legs
	}
	return []kernel.PlanArmLeg{{Entry: sc.Arm.Entry, Stop: sc.Arm.Stop, Target: sc.Arm.Target,
		WaitConfirm: sc.Arm.WaitConfirm, Rule: "touch", Policy: sc.Arm.Policy}}
}

// armAuthoredLegAt reads the AUTHORED leg li of scenario id in planID v<version>
// from plan history, through the ONE fold every reader uses (GetPlan +
// ListOverlays + kernel.ResolvePlanFinal, as resolveActivePlanDoc folds the
// live version). why != "" means ABSENT — a read error, a missing version, a
// scenario the version did not carry, a leg it did not author — and absent is
// never "changed" (L7): the caller does not re-spec on it.
func (at *AutoTrader) armAuthoredLegAt(planID string, version int, scenario string, li int) (kernel.PlanArmLeg, string) {
	if at == nil || at.store == nil {
		return kernel.PlanArmLeg{}, "plan history unavailable (no store)"
	}
	plans := at.store.Plan()
	p, err := plans.GetPlan(planID, version)
	if err != nil {
		return kernel.PlanArmLeg{}, fmt.Sprintf("v%d read failed: %v", version, err)
	}
	if p == nil {
		return kernel.PlanArmLeg{}, fmt.Sprintf("v%d not in plan history", version)
	}
	ovs, err := plans.ListOverlays(planID, version)
	if err != nil {
		return kernel.PlanArmLeg{}, fmt.Sprintf("v%d overlays read failed: %v", version, err)
	}
	pf, err := kernel.ResolvePlanFinal([]byte(p.Doc), kernel.OverlayRefsFrom(ovs))
	if err != nil {
		return kernel.PlanArmLeg{}, fmt.Sprintf("v%d doc unparseable: %v", version, err)
	}
	for _, sc := range pf.Doc.Scenarios {
		if sc.ID != scenario {
			continue
		}
		legs := armScenarioLegs(sc)
		if li < 0 || li >= len(legs) {
			return kernel.PlanArmLeg{}, fmt.Sprintf("v%d %s authored no leg %d", version, scenario, li+1)
		}
		return legs[li], ""
	}
	return kernel.PlanArmLeg{}, fmt.Sprintf("v%d carried no scenario %s", version, scenario)
}

// armRespecFor decides whether a working planner row must be replaced. PURE
// given priorAuthored (called at most once, only when E1 is asked).
//
//   - E2: a market_in_zone row whose resting limit lies outside the zone the
//     current plan authorizes. NOT version-gated: within one zone the far edge
//     is always contained, so this is silent when nothing moved — and an
//     overlay that moves the zone inside the SAME version also cancels (the
//     D15 pin then keeps that version from re-arming: fail-closed, named).
//   - E1: under a NEWER plan version, the AUTHORED entry (non-market_in_zone
//     rows only — "entry re-spec by vN: E a→b", asked first), stop or target of
//     this scenario's leg (authored = the plan doc's price, armScenarioLegs)
//     moved by ≥ 2 ticks from the one the version the row was placed under
//     authored. A market_in_zone row's entry is E2's (the zone decides).
//     The composed bracket (live-ATR min-SL floor, structure anchor, obstacle
//     target) is NEVER compared: it drifts with ATR inside and across versions,
//     and that drift is not a re-spec (W1b verifier: an identical scenario
//     re-published as v2 cancelled a working order as "SL 97.70→96.82"). When
//     the prior version's authored leg cannot be read, E1 does not fire and
//     absent names why (absent ≠ changed).
//
// Only a working row with a signal is ever judged; a Picture row (Source set)
// belongs to its own pin and deadline (pictureDeadlineCancel).
func armRespecFor(planVersion int, prior store.ArmedOrderDB, authored, leg kernel.PlanArmLeg, zl zoneLeg, tick float64,
	priorAuthored func() (kernel.PlanArmLeg, string)) (c armRespecCancel, absent string, ok bool) {
	if prior.State != store.StateWorking || strings.TrimSpace(prior.SignalID) == "" || strings.TrimSpace(prior.Source) != "" {
		return armRespecCancel{}, "", false
	}
	if tick <= 0 {
		tick = 0.25
	}
	const eps = 1e-9
	if zl.on && prior.Policy == kernel.EntryPolicyMarketInZone &&
		(prior.EntryPx < zl.v.Lo-eps || prior.EntryPx > zl.v.Hi+eps) {
		return armRespecCancel{
			reason:  fmt.Sprintf("zone moved by v%d: limit %.2f outside %.2f–%.2f", planVersion, prior.EntryPx, zl.v.Lo, zl.v.Hi),
			counter: "market_in_zone:zone_moved",
			class:   "zone moved",
		}, "", true
	}
	if planVersion <= prior.Version || priorAuthored == nil {
		return armRespecCancel{}, "", false
	}
	was, why := priorAuthored()
	if why != "" {
		return armRespecCancel{}, why, false
	}
	// W1b FOLD-1 (CTO, P1) — a re-priced ENTRY. A non-market_in_zone row rests
	// AT its authored entry, and the AddOn cannot move a resting entry, so an
	// authored entry moved by ≥ 2 ticks (churnNeedsModify's threshold, one
	// definition) is a re-spec like the bracket's. A market_in_zone row rests
	// at the zone's far edge and keeps E2's zone rule. A zero entry on either
	// side is absent, never a change (L7). Asked before the bracket: one
	// decision per row.
	if prior.Policy != kernel.EntryPolicyMarketInZone && was.Entry > 0 && authored.Entry > 0 &&
		churnNeedsModify(was.Entry, 0, authored.Entry, 0, tick) {
		return armRespecCancel{
			reason: fmt.Sprintf("entry re-spec by v%d: E %.2f→%.2f (v%d→v%d; resting E %.2f, composed now E %.2f)",
				planVersion, was.Entry, authored.Entry, prior.Version, planVersion, prior.EntryPx, leg.Entry),
			counter: "arm:respec_cancel",
			class:   "entry re-spec",
		}, "", true
	}
	if !churnNeedsModify(was.Stop, was.Target, authored.Stop, authored.Target, tick) {
		return armRespecCancel{}, "", false
	}
	return armRespecCancel{
		reason: fmt.Sprintf("bracket re-spec by v%d: authored SL %.2f→%.2f TP %.2f→%.2f (v%d→v%d; resting SL %.2f TP %.2f, composed now SL %.2f TP %.2f)",
			planVersion, was.Stop, authored.Stop, was.Target, authored.Target, prior.Version, planVersion, prior.StopPx, prior.TargetPx, leg.Stop, leg.Target),
		counter: "arm:respec_cancel",
		class:   "bracket re-spec",
	}, "", true
}

// respecWorkingArm is the arm loop's call for a leg that already has a live
// ledger row (prior) and just passed every authoring gate under the current
// plan. authored is the loop's leg BEFORE composition (legs[li]); leg is the
// composed one (context for the ledger reason only). true = a cancel was
// REQUESTED this pass (the row is cancel_pending).
func (at *AutoTrader) respecWorkingArm(ledger *store.ArmedOrderStore, plan *kernel.ActivePlan, sc kernel.PlanScenario, li int, prior store.ArmedOrderDB, authored, leg kernel.PlanArmLeg, zl zoneLeg, now time.Time) bool {
	if ledger == nil || plan == nil {
		return false
	}
	key := plan.PlanID + ":" + strconv.Itoa(plan.Version) + ":" + sc.ID + ":leg" + strconv.Itoa(li+1) + ":respec"
	c, absent, ok := armRespecFor(plan.Version, prior, authored, leg, zl, market.FuturesTickSize(at.futuresSymbol()),
		func() (kernel.PlanArmLeg, string) { return at.armAuthoredLegAt(plan.PlanID, prior.Version, sc.ID, li) })
	if absent != "" {
		if armRefusalChanged(&at.armRefusalLast, key+"_prior", absent) {
			at.logWarnf("⚠️ armed re-spec NOT judged: %s %s leg %d signal=%s — the authored bracket of v%d is absent (%s); absent is not changed, the working order is left alone",
				plan.Session, sc.ID, li+1, shortID(prior.SignalID), prior.Version, absent)
		}
		return false
	}
	if !ok {
		return false
	}
	nt := at.armedTrader()
	if nt == nil {
		return false
	}
	if v := at.cancelSafetyFor(prior, now); !v.Allow {
		if armRefusalChanged(&at.armRefusalLast, key, "refused: "+v.Why) {
			at.logWarnf("🛟 armed cancel REFUSED (%s): %s %s leg %d signal=%s — %s; retried next pass",
				c.class, plan.Session, sc.ID, li+1, shortID(prior.SignalID), v.Why)
		}
		return false
	}
	if cerr := nt.CancelOrder(prior.SignalID); cerr != nil {
		at.logWarnf("✕ armed cancel SEND failed (%s): %s %s leg %d signal=%s: %v", c.class, plan.Session, sc.ID, li+1, shortID(prior.SignalID), cerr)
	}
	if err := ledger.RequestCancel(prior.ID, c.reason, now.UnixMilli()); err != nil {
		at.logWarnf("✕ armed cancel (%s): ledger write failed for %s leg %d: %v", c.class, sc.ID, li+1, err)
		return false
	}
	shown := ""
	if at.store != nil {
		if n, err := store.IncSystemCounter(at.store, c.counter); err == nil {
			shown = fmt.Sprintf(" · %s recorded: %d", c.counter, n)
		}
	}
	if armRefusalChanged(&at.armRefusalLast, key, "requested") {
		at.logWarnf("✕ armed cancel REQUESTED (%s): %s %s leg %d signal=%s — pending broker confirmation; re-arms under v%d once the book confirms%s",
			c.reason, plan.Session, sc.ID, li+1, shortID(prior.SignalID), plan.Version, shown)
	}
	return true
}
