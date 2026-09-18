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
	// W-KNOB-PRUNE (2026-09-18): the multiplier is a constant (1.0, owner
	// ruling); READ from the resolver so the line can never drift from it.
	multPart := fmt.Sprintf("htf_mult=%.1f(const)", mult)
	return fmt.Sprintf("🧮 %s · %s (S3)", seatsPart, multPart)
}
