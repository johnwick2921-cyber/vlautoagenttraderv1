package trader

import (
	"fmt"

	"nofx/kernel"
	"nofx/store"
)

// HtfKnobsBootLine (S3, 2026-09-16) renders the resolved HTF knobs at trader
// load — the moment the strategy config exists. READ from the resolver, never
// from a file default: nil seats → the legacy (non-effective) path; a saved
// value → the effective promotion. The multiplier names its source the same way.
func HtfKnobsBootLine(dp *store.DayPlanConfig) string {
	_, seats, mult, _, _ := resolveSessionPlanCfg(dp, "")
	seatsPart := fmt.Sprintf("htf_seats=%d(legacy, non-effective)", kernel.LegacyHtfSeats)
	if seats != nil {
		seatsPart = fmt.Sprintf("htf_seats=%d(saved, effective)", *seats)
	}
	multPart := fmt.Sprintf("htf_mult=%.1f(default)", kernel.HTFScoreMultiplier)
	if dp != nil && dp.HtfScoreMultiplier != nil {
		multPart = fmt.Sprintf("htf_mult=%.1f(saved)", mult)
	}
	return fmt.Sprintf("🧮 %s · %s (S3)", seatsPart, multPart)
}
