package trader

import (
	"nofx/store"
)

// GeometryRefBootLine (W-GEOMETRY-REFUSAL, 2026-09-18) is the per-trader boot
// line for day_plan.geometry_reference_levels — READ from the resolved knob:
// the owner ruled the contract fix ON by default ("both fix now", 08:1x CT);
// explicit false = OFF (today's behaviour byte-identical). The process-level
// 🎛 entry law line prints n/a because the knob is per-strategy.
func GeometryRefBootLine(dp *store.DayPlanConfig) string {
	// nil config or nil pointer = the OWNER DEFAULT (ON, ruling 08:1x CT); an
	// explicit true is a SAVED value and must read differently (F6).
	if dp == nil || dp.GeometryReferenceLevels == nil {
		return "🎛 geom_ref_ids=on(default) (day_plan.geometry_reference_levels; W-GEOMETRY-REFUSAL)"
	}
	if !*dp.GeometryReferenceLevels {
		return "🎛 geom_ref_ids=off (day_plan.geometry_reference_levels; W-GEOMETRY-REFUSAL)"
	}
	return "🎛 geom_ref_ids=on(saved) (day_plan.geometry_reference_levels; W-GEOMETRY-REFUSAL)"
}
