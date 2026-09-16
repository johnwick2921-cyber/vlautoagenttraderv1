// Settings integrity (D6) — resolvers that report WHERE the value came from.
//
// The UI renders "saved → resolved · source". That line is only honest if the
// source is produced by the same function production resolves with; a second
// copy in the API layer would drift the moment either side changed, and the
// page would then narrate a rule the engine does not follow.
//
// So the rule lives here once, and the shipped entry points (resolvedMinRR,
// HTFVetoEnabled, PlanModeFor) delegate to these. Canon 28: one resolver,
// called where the value enters.

package store

import "strings"

// Where a resolved value came from. These strings are rendered to the operator,
// so they say what happened, not which branch ran.
const (
	SourceSaved           = "saved value"
	SourceSchemaDefault   = "schema default"
	SourceShippedDefault  = "shipped default"
	SourceStrategyValue   = "strategy value"
	SourceSessionOverride = "session override"
)

// ResolveMinRiskReward is the single R:R floor — the arm seam and the decision
// path both land here. A saved value above zero wins; absent means the schema's
// own safe default, never a second opinion from an env var.
func ResolveMinRiskReward(cfg *StrategyConfig) (float64, string) {
	if cfg != nil && cfg.RiskControl.MinRiskRewardRatio > 0 {
		return cfg.RiskControl.MinRiskRewardRatio, SourceSaved
	}
	return SafeDefaultMinRiskReward, SourceSchemaDefault
}

// ResolveHTFVeto reports the higher-timeframe veto and why. Absent → ON: the
// veto is a safety default, so a missing block must not read as "off".
func ResolveHTFVeto(cfg *StrategyConfig) (bool, string) {
	if cfg == nil || cfg.Regime == nil || cfg.Regime.HTFVeto == nil {
		return true, SourceShippedDefault
	}
	return *cfg.Regime.HTFVeto, SourceSaved
}

// ResolvePlanMode resolves the plan-restriction mode for a session:
// per-session override → strategy-level → "advisory". advisory never gates.
func ResolvePlanMode(c *DayPlanConfig, session string) (string, string) {
	mode, source := "advisory", SourceShippedDefault
	if c != nil && strings.TrimSpace(c.PlanMode) != "" {
		mode, source = c.PlanMode, SourceStrategyValue
	}
	if ov := c.SessionOverride(session); ov != nil && ov.PlanMode != nil && strings.TrimSpace(*ov.PlanMode) != "" {
		mode, source = *ov.PlanMode, SourceSessionOverride
	}
	return strings.ToLower(strings.TrimSpace(mode)), source
}

// ResolveOneSetup (dispatch 102) resolves the two one-setup knobs from THE
// BOUND STRATEGY, with their sources: enabled defaults ON [O] (nil, never a
// plain false), min grade defaults B [O]. A malformed grade falls back to B
// and says so — a boot line that prints a grade the predicate does not use is
// worse than none (class 45/49).
func ResolveOneSetup(cfg *StrategyConfig) (enabled bool, minGrade, enabledSource, gradeSource string) {
	enabled, enabledSource = true, SourceShippedDefault
	minGrade, gradeSource = "B", SourceShippedDefault
	if cfg == nil || cfg.DayPlan == nil {
		return
	}
	if cfg.DayPlan.OneSetupEnabled != nil {
		enabled, enabledSource = *cfg.DayPlan.OneSetupEnabled, SourceSaved
	}
	switch g := strings.ToUpper(strings.TrimSpace(cfg.DayPlan.OneSetupMinGrade)); g {
	case "A+", "A", "B", "C":
		minGrade, gradeSource = g, SourceSaved
	case "":
	default:
		gradeSource = SourceShippedDefault + " (saved value " + g + " is not a grade)"
	}
	return
}
