package kernel

import (
	"fmt"
	"strings"
)

// P2 — DARK REGIME DETECTION.
//
// The regime block is 7 auto-computed fields (spec §regime). Each one degrades
// honestly to "n/a"/WARMING when its input is missing — which is correct, but
// SILENT: a plan can be written on half a map with nothing saying so. The AddOn
// watchdog livelock that starves timeframes turns 4 of the 7 dark at once, and
// the only symptom today is a quieter-looking REGIME line.
//
// So: name the dark fields, alert on them, stamp the count on the plan row, and
// flag the plan DEGRADED past a threshold so the card can say it out loud.

// DarkRegimeThreshold is the default "too dark to trust" count. More than this
// many unavailable fields ⇒ the plan is written but flagged DEGRADED.
const DarkRegimeThreshold = 3

// RegimeFieldCount is the number of fields the regime block promises.
const RegimeFieldCount = 7

// DarkRegimeFields returns the names of the regime fields that are UNAVAILABLE,
// in the spec's order. A field is dark when its computation could not run at all
// — not merely when it is warming toward a baseline.
//
// realized_vol is the one nuance: RVWarming with a value means "computed, no 20d
// baseline yet" (honest, usable), while a zero value means genuinely dark.
func DarkRegimeFields(r RegimeBlock) []string {
	var dark []string
	if r.TrendDaily == "" || r.TrendDaily == "n/a" {
		dark = append(dark, "trend_state_daily")
	}
	if r.Trend1h == "" || r.Trend1h == "n/a" {
		dark = append(dark, "trend_state_1h")
	}
	if r.ATR14 <= 0 || r.ATRRegime == "" || r.ATRRegime == "n/a" {
		dark = append(dark, "atr_regime")
	}
	if r.RealizedVolPct <= 0 {
		dark = append(dark, "realized_vol_pct")
	}
	if r.VIXLevel <= 0 {
		dark = append(dark, "vix_level")
	}
	if r.ExpectedRangePts <= 0 {
		dark = append(dark, "expected_range_pts")
	}
	if !r.HasGap {
		dark = append(dark, "overnight_gap")
	}
	return dark
}

// RegimeHealth summarizes the regime block for alerting + the plan row.
type RegimeHealth struct {
	Dark      []string // unavailable field names
	DarkCount int
	Degraded  bool // DarkCount > threshold ⇒ the plan is flagged on the card
	Threshold int
}

// AssessRegime evaluates a regime block. threshold <= 0 uses DarkRegimeThreshold.
func AssessRegime(r RegimeBlock, threshold int) RegimeHealth {
	if threshold <= 0 {
		threshold = DarkRegimeThreshold
	}
	dark := DarkRegimeFields(r)
	return RegimeHealth{
		Dark: dark, DarkCount: len(dark),
		Degraded: len(dark) > threshold, Threshold: threshold,
	}
}

// AlertBody renders the human line for the dark-regime alert: which fields went
// dark, and how bad it is. Empty when nothing is dark.
func (h RegimeHealth) AlertBody() string {
	if h.DarkCount == 0 {
		return ""
	}
	verdict := "plan still trustworthy"
	if h.Degraded {
		verdict = "DEGRADED — plan written on a partial map"
	}
	return fmt.Sprintf("%d/%d regime fields unavailable (%s) — %s",
		h.DarkCount, RegimeFieldCount, strings.Join(h.Dark, ", "), verdict)
}
