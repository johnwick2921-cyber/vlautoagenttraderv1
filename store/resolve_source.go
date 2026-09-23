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

import (
	"os"
	"strconv"
	"strings"
)

// Where a resolved value came from. These strings are rendered to the operator,
// so they say what happened, not which branch ran.
const (
	SourceSaved           = "saved value"
	SourceSchemaDefault   = "schema default"
	SourceShippedDefault  = "shipped default"
	SourceStrategyValue   = "strategy value"
	SourceSessionOverride = "session override"
	// SourceEnv prefixes a value read from a process environment variable; the
	// variable's name follows ("env BREAKER_HALT_N").
	SourceEnv = "env"
)

// OriginLetter maps a resolver source onto the boot-line evidence tag (W1):
// [O] the owner set it (saved / strategy / session), [E] the process env set
// it, [I] the shipped default. ONE mapping, so two boot lines cannot tag the
// same source two ways. An unrecognised source reads [?] — never a guess.
func OriginLetter(source string) string {
	switch {
	case strings.HasPrefix(source, SourceSaved),
		strings.HasPrefix(source, SourceStrategyValue),
		strings.HasPrefix(source, SourceSessionOverride):
		return "[O]"
	case strings.HasPrefix(source, SourceEnv+" "):
		return "[E]"
	case strings.HasPrefix(source, SourceShippedDefault),
		strings.HasPrefix(source, SourceSchemaDefault):
		return "[I]"
	}
	return "[?]"
}

// BreakerHaltDefault is the consecutive-loss halt's shipped N — vet-06's [I]
// proposal (8 of the last 10 losing). It never fires on the retained tape (max
// run 7, ids 585-591); trader/session_risk.go carries the measurement.
const BreakerHaltDefault = 8

// BreakerHaltEnv is the process-wide override consulted only when the strategy
// saved no value.
const BreakerHaltEnv = "BREAKER_HALT_N"

// ReplanCapDefault is the shipped re-plan cap when neither the session nor the
// strategy saved one.
const ReplanCapDefault = 2

// ResolveBreakerHalt resolves the consecutive-loss halt N and where it came
// from (W1, settings truth). Precedence: the SAVED value — INCLUDING 0, which
// is OFF — → env BREAKER_HALT_N (valid ≥0; 0 = OFF) → the shipped default 8.
// An invalid env value falls to the default AND SAYS SO in the source, never a
// silent 8. Absent in the strategy means INHERIT, never OFF: the breaker is a
// safety default, so a missing key must not read as "off".
func ResolveBreakerHalt(cfg *StrategyConfig) (int, string) {
	return ResolveBreakerHaltEnv(cfg, os.Getenv)
}

// ResolveBreakerHaltEnv is ResolveBreakerHalt with the environment injected —
// the migration report evaluates a fixed environment, and tests pin one.
func ResolveBreakerHaltEnv(cfg *StrategyConfig, getenv func(string) string) (int, string) {
	if cfg != nil && cfg.RiskControl.ConsecutiveLossHalt != nil {
		if n := *cfg.RiskControl.ConsecutiveLossHalt; n >= 0 {
			return n, SourceSaved
		}
		// A negative saved value is not a threshold. Fall through to what an
		// absent key would resolve to, and name why.
		n, src := breakerHaltFromEnv(getenv)
		return n, src + " (saved value is negative)"
	}
	return breakerHaltFromEnv(getenv)
}

func breakerHaltFromEnv(getenv func(string) string) (int, string) {
	n, state := BreakerHaltEnvState(getenv)
	switch state {
	case EnvValid:
		return n, SourceEnv + " " + BreakerHaltEnv
	case EnvInvalid:
		return BreakerHaltDefault, SourceShippedDefault + " (env " + BreakerHaltEnv + " invalid)"
	}
	return BreakerHaltDefault, SourceShippedDefault
}

// Env states for a numeric process override.
const (
	EnvUnset   = "unset"
	EnvValid   = "valid"
	EnvInvalid = "invalid"
)

// BreakerHaltEnvState reads BREAKER_HALT_N once: unset, valid (an integer ≥0)
// or invalid. The same test envInt applied before W1 (Atoi, ≥0), so the
// pre-W1 and post-W1 readings of one environment agree.
func BreakerHaltEnvState(getenv func(string) string) (int, string) {
	if getenv == nil {
		getenv = os.Getenv
	}
	v := getenv(BreakerHaltEnv)
	if v == "" {
		return 0, EnvUnset
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, EnvInvalid
	}
	return n, EnvValid
}

// ResolveReplanCap resolves the per-session re-plan cap and its source (W1):
// session override (≥0) → strategy value (≥0) → the shipped default 2. An
// explicit 0 is honoured at BOTH levels — "no re-plan after death" — where the
// strategy level used to read 0 as "unset → 2". session "" resolves the
// strategy level (outside every session window).
func ResolveReplanCap(c *DayPlanConfig, session string) (int, string) {
	n, source := ReplanCapDefault, SourceShippedDefault
	if c != nil && c.ReplanCap != nil && *c.ReplanCap >= 0 {
		n, source = *c.ReplanCap, SourceStrategyValue
	}
	if ov := c.SessionOverride(session); ov != nil && ov.ReplanCap != nil && *ov.ReplanCap >= 0 {
		n, source = *ov.ReplanCap, SourceSessionOverride
	}
	return n, source
}

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

// ResolveStructureMap resolves day_plan.structure_map (S1, 2026-09-16): the
// STRUCTURE table is OFF unless the strategy saved true. Every S1 knob
// defaults OFF — nothing changes the live plan until S4 measures it.
func ResolveStructureMap(cfg *StrategyConfig) (enabled bool, source string) {
	if cfg == nil || cfg.DayPlan == nil || cfg.DayPlan.StructureMap == nil {
		return false, SourceShippedDefault
	}
	return *cfg.DayPlan.StructureMap, SourceSaved
}
