package trader

import (
	"fmt"

	"nofx/kernel"
	"nofx/store"
)

// W-EXEC-TRUTH W3 (2026-09-23) — the entry-policy knobs at the WRITE site.
//
// ONE resolution for the live write loop and the shadow A/B replay (parity):
// the R4 default stamp, the R:R floor, and the armable hold floor all come from
// the bound strategy through the store resolvers, and reach the kernel as a
// single kernel.AuthoringOpts value.

// plannerAuthoringOpts resolves the new-authoring parse options from the
// bound strategy (nil config → the shipped defaults, exactly as the resolvers
// answer for a nil DayPlanConfig).
func (at *AutoTrader) plannerAuthoringOpts() kernel.AuthoringOpts {
	dp := at.dayPlanCfg()
	policy, _ := store.ResolveEntryPolicyDefault(dp)
	hold, _ := store.ResolveMinHoldMin(dp)
	return kernel.AuthoringOpts{
		MinRR:              at.armMinRRFor(nil),
		EntryPolicyDefault: policy,
		MinHoldMin:         hold,
	}
}

// entryPolicyForPrompt is the resolved day_plan.entry_policy_default the
// prompt, the bias-arm warning and the stamp all read (legacy = the explicit
// off: legacy prompt, no stamp).
func entryPolicyForPrompt(dp *store.DayPlanConfig) string {
	p, _ := store.ResolveEntryPolicyDefault(dp)
	return p
}

// zoneMaxPtsForPrompt is the resolved day_plan.zone_max_pts the prompt states
// (the same resolver the zone-at-write verdict judges with).
func zoneMaxPtsForPrompt(dp *store.DayPlanConfig) float64 {
	v, _ := store.ResolveZoneMaxPts(dp)
	return v
}

// entryLawBootLine renders the 🎛 entry law boot line. Every value is READ
// from its resolver and tagged with its origin letter ([O] owner, [I] shipped
// default) — never typed.
func entryLawBootLine(dp *store.DayPlanConfig) string {
	policy, pSrc := store.ResolveEntryPolicyDefault(dp)
	zmax, zSrc := store.ResolveZoneMaxPts(dp)
	rest, rSrc := store.ResolveZoneRestMaxMin(dp)
	hold, hSrc := store.ResolveMinHoldMin(dp)
	return fmt.Sprintf("🎛 entry law: write_feas=%s · entry_policy_default=%s%s zone_max_pts=%g%s zone_rest_max_min=%d%s min_hold_min=%d%s",
		writeFeasLabel(dp),
		policy, store.OriginLetter(pSrc),
		zmax, store.OriginLetter(zSrc),
		rest, store.OriginLetter(rSrc),
		hold, store.OriginLetter(hSrc))
}

// entryPolicyOneSetupWarn is the D12 READ boot WARN: with the default policy
// market_in_zone and one_setup resolving ON, the one-setup seam declines every
// non-reject play (play_not_reject) — so only reject arms can place. No
// behaviour change; "" when the two do not collide.
func entryPolicyOneSetupWarn(cfg *store.StrategyConfig) string {
	var dp *store.DayPlanConfig
	if cfg != nil {
		dp = cfg.DayPlan
	}
	policy, pSrc := store.ResolveEntryPolicyDefault(dp)
	on, _, oSrc, _ := store.ResolveOneSetup(cfg)
	if policy != store.EntryPolicyMarketInZone || !on {
		return ""
	}
	return fmt.Sprintf("⚠ entry law: entry_policy_default=%s%s but one_setup_enabled=true%s — one setup declines every non-reject play (play_not_reject), so under STRICT only reject arms can place; set day_plan.one_setup_enabled=false to trade the other conditions (precondition, W3)",
		policy, store.OriginLetter(pSrc), store.OriginLetter(oSrc))
}
