package kernel

import (
	"strings"

	"nofx/market"
	"nofx/store"
)

// W11 — PLANNER INDICATOR MIRROR (owner override, spec §regime). The planner prompt
// gains an INDICATORS block that mirrors the EXACT per-timeframe indicator state the
// EXECUTOR prompt renders (kernel.FormatIndicatorState), driven by the strategy's
// EXISTING ai_config toggles — zero new config fields. Rendered from the same
// market.Data the executor computes (GetWithTimeframes), so periods/values match
// the executor byte-for-byte. The rendered string is FROZEN onto the plan row so a
// later toggle change never rewrites what the planner actually saw.

// RenderPlannerIndicatorBlock renders the per-timeframe indicator mirror. Timeframes
// render in the given order; a timeframe with no enabled indicator / no data is
// skipped. Returns "" when nothing renders (→ the planner omits the block, so the
// disabled state is byte-identical to the pre-W11 prompt).
func RenderPlannerIndicatorBlock(mkt *market.Data, indicators store.IndicatorConfig, timeframes []string) string {
	if mkt == nil || len(mkt.TimeframeData) == 0 || len(timeframes) == 0 {
		return ""
	}
	seen := map[string]bool{}
	var b strings.Builder
	for _, tf := range timeframes {
		if tf == "" || seen[tf] {
			continue
		}
		seen[tf] = true
		td := mkt.TimeframeData[tf]
		if td == nil {
			continue
		}
		var sec strings.Builder
		FormatIndicatorState(&sec, td, indicators)
		body := strings.TrimRight(sec.String(), "\n")
		if body == "" {
			continue // no enabled indicator produced output for this timeframe
		}
		b.WriteString("### " + tf + "\n")
		b.WriteString(body + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
