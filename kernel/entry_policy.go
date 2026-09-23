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
