package kernel

import (
	"fmt"
	"math"
	"strings"
)

// W-EXEC-TRUTH W3 — the entry policy. ONE table (beside entryLaw) that the
// prompt, the write-time validator and the executor all read.
//
//	market_in_zone — a LIMIT at the far edge of the planner's own
//	                 economics.entry_zone (buy → zone high, sell → zone low):
//	                 marketable inside the zone, can never fill beyond it on
//	                 the adverse side (R1). Legal on every known condition.
//	planned_order  — today's resting order at the authored entry; legal only
//	                 where a resting order is the play (reject, fvg_entry,
//	                 sweep_reclaim leg 0).
//	absent ("")    — LEGACY: today's arm kinds, byte-identical (R4).
const (
	EntryPolicyPlannedOrder = "planned_order"
	EntryPolicyMarketInZone = "market_in_zone"
	// EntryPolicyDefaultLegacy is the day_plan.entry_policy_default value that
	// stamps NOTHING on new plans (the explicit "off"): planned_order is not
	// the same thing — it is illegal on reclaim and the waterfalls.
	EntryPolicyDefaultLegacy = "legacy"
)

// plannedOrderLegal: the conditions where planned_order is a legal policy,
// and the only leg index it is legal on (-1 = any leg).
var plannedOrderLegal = map[string]int{
	"reject":        -1,
	"fvg_entry":     -1,
	"sweep_reclaim": 0,
}

func knownCondition(c string) bool {
	for _, k := range KnownConditions() {
		if k == c {
			return true
		}
	}
	return false
}

// EntryPolicyLegal reports whether policy is legal for the condition on the
// given leg (0 for a single arm). An absent policy is legacy: always legal.
func EntryPolicyLegal(condition, policy string, legIdx int) error {
	c := strings.ToLower(strings.TrimSpace(condition))
	switch policy {
	case "":
		return nil
	case EntryPolicyMarketInZone:
		if !knownCondition(c) {
			return fmt.Errorf("entry policy %s: unknown condition %q", policy, condition)
		}
		return nil
	case EntryPolicyPlannedOrder:
		leg, ok := plannedOrderLegal[c]
		if !ok {
			return fmt.Errorf("entry policy planned_order is not legal on %s (planned_order is for reject, fvg_entry and sweep_reclaim leg 0; use market_in_zone)", c)
		}
		if leg >= 0 && legIdx != leg {
			return fmt.Errorf("entry policy planned_order is legal on %s leg %d only (this is leg %d)", c, leg, legIdx)
		}
		return nil
	}
	return fmt.Errorf("unknown entry policy %q (market_in_zone | planned_order)", policy)
}

// EffectiveArmPolicy is the policy that governs one leg: the leg's own, else
// the arm's. nil leg = the single arm.
func EffectiveArmPolicy(arm *PlanArmSpec, leg *PlanArmLeg) string {
	if leg != nil && leg.Policy != "" {
		return leg.Policy
	}
	if arm != nil {
		return arm.Policy
	}
	return ""
}

// ArmableConditionFor: under market_in_zone every known condition is armable
// (the entry is a limit inside the planner's zone); any other policy keeps
// the legacy set exactly.
func ArmableConditionFor(condition, policy string) bool {
	if policy == EntryPolicyMarketInZone {
		return knownCondition(strings.ToLower(strings.TrimSpace(condition)))
	}
	return ArmableCondition(condition)
}

// ArmKindForPolicy: market_in_zone ⇒ a limit for every known condition;
// planned_order and legacy keep today's mapping (reclaim = stop entry).
func ArmKindForPolicy(condition, policy string) string {
	if policy == EntryPolicyMarketInZone {
		if knownCondition(strings.ToLower(strings.TrimSpace(condition))) {
			return ArmKindLimit
		}
		return ""
	}
	return ArmKindFor(condition)
}

// Zone verdict codes. "" = the zone is usable.
const (
	ZoneMissing      = "zone_missing"       // no economics.entry_zone, or not a positive [lo, hi]
	ZoneTooWide      = "zone_too_wide"      // hi − lo > day_plan.zone_max_pts
	ZoneEntryOutside = "zone_entry_outside" // the authored entry is not inside [lo, hi]
	ZoneBadSide      = "zone_bad_side"      // side is not long|short
	ZoneTriggerSide  = "zone_trigger_side"  // the zone is not on the permitted side of the trigger
	ZoneBracket      = "zone_bracket"       // stop/target not outside the zone (long: stop < lo, target > hi)
	ZoneEmpty        = "zone_empty"         // no tick lies inside the zone after inward rounding
)

// ZoneVerdict is the ONE reading of a scenario's entry zone. Lo/Hi are the
// zone rounded INWARD to the tick grid (so the wire's nearest-tick rounding is
// the identity); Far is the limit price (buy → Hi, sell → Lo) and the worst
// fill for R:R; Near is the other bound and the worst fill for stop distance.
type ZoneVerdict struct {
	Lo, Hi, Far, Near float64
	Code              string
}

const zoneEps = 1e-9

func roundTick6(p float64) float64 { return math.Round(p*1e6) / 1e6 }

// ArmZoneVerdict judges one market_in_zone leg against its scenario's
// economics.entry_zone (R2): present; the authored entry inside [lo, hi]
// (inclusive); width ≤ maxWidthPts (0 = no cap); the permitted side of the
// scenario's trigger (touch: the zone contains the confirm ref ±1 tick; a
// close / time_hold / mss confirm: the whole zone lies on the confirm side of
// the ref ±1 tick; no confirm → not judged); the bracket outside the zone;
// then inward rounding to the tick (lo up, hi down) must leave a tick inside.
func ArmZoneVerdict(sc PlanScenario, entry, stop, target float64, side string, tick, maxWidthPts float64) ZoneVerdict {
	var v ZoneVerdict
	e := sc.Economics
	if e == nil || len(e.EntryZone) != 2 || e.EntryZone[0] <= 0 || e.EntryZone[1] <= 0 || e.EntryZone[0] > e.EntryZone[1] {
		v.Code = ZoneMissing
		return v
	}
	lo, hi := e.EntryZone[0], e.EntryZone[1]
	if maxWidthPts > 0 && hi-lo > maxWidthPts+zoneEps {
		v.Code = ZoneTooWide
		return v
	}
	if entry < lo-zoneEps || entry > hi+zoneEps {
		v.Code = ZoneEntryOutside
		return v
	}
	long := strings.EqualFold(side, "long")
	short := strings.EqualFold(side, "short")
	if !long && !short {
		v.Code = ZoneBadSide
		return v
	}
	if r, ok := ResolveScenarioConfirm(sc); ok && r.RefPrice > 0 {
		t := tick
		if t <= 0 {
			t = 0
		}
		switch r.Kind {
		case ConfirmKindTouch:
			if r.RefPrice < lo-t-zoneEps || r.RefPrice > hi+t+zoneEps {
				v.Code = ZoneTriggerSide
				return v
			}
		default:
			switch strings.ToLower(r.Side) {
			case "above":
				if lo < r.RefPrice-t-zoneEps {
					v.Code = ZoneTriggerSide
					return v
				}
			case "below":
				if hi > r.RefPrice+t+zoneEps {
					v.Code = ZoneTriggerSide
					return v
				}
			}
		}
	}
	if long && !(stop < lo-zoneEps && target > hi+zoneEps) || short && !(stop > hi+zoneEps && target < lo-zoneEps) {
		v.Code = ZoneBracket
		return v
	}
	if tick > 0 {
		lo = roundTick6(math.Ceil(lo/tick-zoneEps) * tick)
		hi = roundTick6(math.Floor(hi/tick+zoneEps) * tick)
	}
	if lo > hi+zoneEps {
		v.Code = ZoneEmpty
		return v
	}
	v.Lo, v.Hi = lo, hi
	if long {
		v.Far, v.Near = hi, lo
	} else {
		v.Far, v.Near = lo, hi
	}
	return v
}

// armHasPolicy: the arm or any of its legs names an entry policy. A legacy
// arm (no policy anywhere) never reaches the policy branch of ArmSpecValid.
func armHasPolicy(a *PlanArmSpec) bool {
	if a == nil {
		return false
	}
	if a.Policy != "" {
		return true
	}
	for _, l := range a.Legs {
		if l.Policy != "" {
			return true
		}
	}
	return false
}

// ArmKindMismatchPolicy is ArmKindMismatch under an entry policy: under
// market_in_zone every arm is a LIMIT (at the far edge of the entry zone), so
// an authored stop_entry is refused by name; any other policy (planned_order,
// legacy "") keeps ArmKindMismatch exactly.
func ArmKindMismatchPolicy(condition, policy, authored string) error {
	if policy != EntryPolicyMarketInZone {
		return ArmKindMismatch(condition, authored)
	}
	want := ArmKindForPolicy(condition, policy)
	got := strings.ToLower(strings.TrimSpace(authored))
	if got == "" || want == "" || got == want {
		return nil
	}
	return fmt.Errorf("%s authored for a %s under market_in_zone — market_in_zone requires %s (a limit at the far edge of the entry zone)",
		got, strings.ToLower(strings.TrimSpace(condition)), want)
}

// armSpecValidPolicy is ArmSpecValid for an arm that carries an entry policy
// (W3 (a)(b)). The law it enforces:
//   - every policy token is legal for the condition on its leg
//     (EntryPolicyLegal — an unknown token is refused by name);
//   - the condition is armable under the leg's policy (ArmableConditionFor:
//     market_in_zone arms every known condition);
//   - sweep_reclaim keeps its chain (single: wait_confirm:true + confirm{});
//   - a waterfall arm takes entry_mode pullback OR immediate, and still needs
//     breakdown{} + wait_confirm:true + confirm{} (leg 1);
//   - D4: a single arm whose PRIMARY confirm (ResolveScenarioConfirm) is not a
//     touch must chain (wait_confirm:true) — the zone limit is placed only once
//     the confirm is MET;
//   - under market_in_zone an authored kind other than limit is refused;
//   - the split-leg lock and the bracket shape are the legacy helpers, with the
//     IDENTICAL error strings.
func armSpecValidPolicy(sc PlanScenario) error {
	a := sc.Arm
	cond := strings.ToLower(strings.TrimSpace(sc.Condition))
	if err := EntryPolicyLegal(cond, a.Policy, 0); err != nil {
		return fmt.Errorf("arm on %s: %v", sc.ID, err)
	}
	for i := range a.Legs {
		if err := EntryPolicyLegal(cond, EffectiveArmPolicy(a, &a.Legs[i]), i); err != nil {
			return fmt.Errorf("arm on %s leg %d: %v", sc.ID, i+1, err)
		}
	}
	policy := EffectiveArmPolicy(a, nil)
	switch {
	case cond == "sweep_reclaim":
		if len(a.Legs) == 0 && !a.WaitConfirm {
			return fmt.Errorf("sweep_reclaim arm on %s requires wait_confirm:true (the retrace arm must chain on its confirm)", sc.ID)
		}
		if sc.Confirm == nil {
			return fmt.Errorf("sweep_reclaim arm on %s requires a confirm{} object to chain on", sc.ID)
		}
	case IsBreakdownCondition(cond):
		if sc.Breakdown == nil {
			return fmt.Errorf("%s arm requires the breakdown{} facts object", sc.ID)
		}
		switch strings.ToLower(strings.TrimSpace(sc.Breakdown.EntryMode)) {
		case "pullback", "immediate":
		default:
			return fmt.Errorf("%s arm requires entry_mode=pullback or entry_mode=immediate (under the %s entry policy both arm: the entry_zone sits at the broken level for a pullback, at the confirming close for immediate)", sc.ID, policy)
		}
		if !a.WaitConfirm {
			return fmt.Errorf("%s arm requires wait_confirm:true (it chains on confirm leg 1 before the zone limit is placed)", sc.ID)
		}
		if sc.Confirm == nil {
			return fmt.Errorf("%s arm requires a confirm{} (leg 1) to chain on", sc.ID)
		}
	default:
		if !ArmableConditionFor(cond, policy) {
			return fmt.Errorf("arm enabled on %q is not armable under the %s entry policy (%s arms %s)", sc.Condition, policy, policy, armableUnderPolicyPipe(policy))
		}
	}
	// D4 — a non-touch primary confirm chains. Split arms carry their own chain
	// (leg 1 touch at the sweep ref, leg 2 on confirm2) in the split lock.
	if len(a.Legs) == 0 && !a.WaitConfirm {
		if r, ok := ResolveScenarioConfirm(sc); ok && r.Kind != ConfirmKindTouch {
			return fmt.Errorf("arm on %s requires wait_confirm:true — its confirm %s is not a touch, so under the %s entry policy the limit is placed only once that confirm is MET", sc.ID, r.Rule, policy)
		}
	}
	// A single arm carries no authored kind (the composer derives it); a split
	// leg's authored kind must agree with its policy.
	for i := range a.Legs {
		if err := ArmKindMismatchPolicy(cond, EffectiveArmPolicy(a, &a.Legs[i]), a.Legs[i].Kind); err != nil {
			return fmt.Errorf("arm on %s leg %d: %v", sc.ID, i+1, err)
		}
	}
	if err := armSplitLock(sc); err != nil {
		return err
	}
	return armPricesValid(sc)
}

// armableUnderPolicyPipe renders the conditions a policy arms, derived from
// the same predicate the validator runs (never typed).
func armableUnderPolicyPipe(policy string) string {
	out := make([]string, 0, 9)
	for _, c := range KnownConditions() {
		if ArmableConditionFor(c, policy) {
			out = append(out, c)
		}
	}
	return strings.Join(out, "|")
}

// StampEntryPolicyDefault is the R4 stamp (W3 D2): on a NEWLY authored plan,
// every arm the planner left without a policy takes the default — but ONLY
// when that policy is legal on EVERY leg of the arm (a single arm is leg 0).
// An arm the default is illegal for is left absent (LEGACY) and nothing is
// recorded: a policy the validator would refuse is never stamped (fail-closed;
// e.g. planned_order on a reclaim stays legacy). A planner-set policy is never
// overwritten. def "" or "legacy" stamps nothing. Leg policies are stamped on
// sweep_reclaim only — the one condition whose legs survive normalization.
func StampEntryPolicyDefault(d *PlanDoc, def string) {
	if d == nil || (def != EntryPolicyMarketInZone && def != EntryPolicyPlannedOrder) {
		return
	}
	for i := range d.Scenarios {
		sc := &d.Scenarios[i]
		a := sc.Arm
		if a == nil || armHasPolicy(a) {
			continue
		}
		cond := strings.ToLower(strings.TrimSpace(sc.Condition))
		legal := EntryPolicyLegal(cond, def, 0) == nil
		for li := range a.Legs {
			legal = legal && EntryPolicyLegal(cond, def, li) == nil
		}
		if !legal {
			continue
		}
		a.Policy = def
		if cond == "sweep_reclaim" {
			for li := range a.Legs {
				a.Legs[li].Policy = def
			}
		}
	}
}

// ArmableHoldFloorMarker opens the armable hold-floor refusal. The message
// never contains the bare field token the A5 prose law routes on, so the
// repair prompt carries THIS law (RepairArmableHoldFloorLaw), not that one.
const ArmableHoldFloorMarker = "armable hold floor:"

// ValidateArmableHoldFloor (W3 D10, CTO ruling 1790181002671) — new authoring
// only, beside ValidateConfirmHoldProse: an ENABLED arm whose effective policy
// is market_in_zone and whose confirm or confirm2 is a time_hold must RESOLVE
// (ResolveConfirm: the stored minutes, else the ACCEPT_HOLD_MIN authoring
// default) to at least floor minutes. An unarmed scenario, a legacy arm and a
// planned_order arm are never judged; floor ≤ 0 = off.
func ValidateArmableHoldFloor(d *PlanDoc, floor int) error {
	if d == nil || floor <= 0 {
		return nil
	}
	for i, s := range d.Scenarios {
		if s.Arm == nil || !s.Arm.Enabled || EffectiveArmPolicy(s.Arm, nil) != EntryPolicyMarketInZone {
			continue
		}
		for _, c := range []struct {
			label string
			c     *PlanConfirm
		}{{"confirm", s.Confirm}, {"confirm2", s.Confirm2}} {
			if c.c == nil || c.c.Rule != "time_hold" {
				continue
			}
			r := ResolveConfirm(*c.c)
			if r.HoldMin < floor {
				return fmt.Errorf("%s scenario[%d] %s %s time_hold resolves to %d min (%s) — an ARMED market_in_zone time_hold must hold at least %d min (day-plan minimum-hold floor); store a longer hold in %s (and state the same minutes in the prose), or leave the scenario unarmed",
					ArmableHoldFloorMarker, i, s.ID, c.label, r.HoldMin, r.Source, floor, c.label)
			}
		}
	}
	return nil
}

// RepairArmableHoldFloorLaw is the repair excerpt for the armable hold floor.
const RepairArmableHoldFloorLaw = "ARMABLE HOLD FLOOR: an ARMED market_in_zone scenario whose confirm or confirm2 is a time_hold is judged on the minutes it RESOLVES to (the stored confirm.hold_min, else the ACCEPT_HOLD_MIN authoring default), and that must be at least the floor the refusal names (day_plan.min_hold_min, default 3) — a shorter hold places the zone limit on noise. Raise the stored minutes and the minutes in the trigger/invalid prose together (they must agree), or set arm.enabled false; an unarmed acceptance or hold is never judged by this floor."

// EntryZoneRefusalMarker opens every write-time zone issue the trader hook
// renders (trader/write_time_feasibility.go writeTimeZoneVerdicts), and routes
// RepairEntryZoneLaw in the repair prompt (lawExcerptsFor).
const EntryZoneRefusalMarker = "entry zone:"

// RepairEntryZoneLaw is the repair excerpt for a market_in_zone zone refusal.
const RepairEntryZoneLaw = "ENTRY ZONE LAW (market_in_zone): the entry is a LIMIT at the FAR edge of economics.entry_zone [low, high] — a long buys at high, a short sells at low — so it fills at once inside the zone and never beyond it. The zone must: be present; contain arm.entry; be at most the width the refusal names (day_plan.zone_max_pts); sit on the permitted side of the confirm ref (a touch zone contains the ref; a close / time_hold / 1m_mss zone lies wholly on confirm.side of the ref); keep the stop and the target OUTSIDE the zone; and still hold one tick after rounding inward. R:R is judged at the far edge and the stop distance at the near edge. Fix the zone, arm.entry or the bracket — never drop the zone."
