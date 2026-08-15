package trader

import (
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W9 — DAY-PLAN CONFIG READERS. The audit found six DayPlanConfig fields that the
// FE persists but NO production code reads: plan_mode, proximity_filter_atr,
// scenario_cap, sessions_enabled, approval_required, evening_digest (+ per-session
// overrides). These resolvers read them (per-session override wins over the
// strategy-level value) and default to the shipped values so behavior is
// byte-identical until the owner changes something.

func (at *AutoTrader) dayPlanCfg() *store.DayPlanConfig {
	if at.config.StrategyConfig == nil {
		return nil
	}
	return at.config.StrategyConfig.DayPlan
}

// sessionOverride returns this session's minimal override block, or nil.
func (at *AutoTrader) sessionOverride(session string) *store.DayPlanSessionOverride {
	dp := at.dayPlanCfg()
	if dp == nil {
		return nil
	}
	for i := range dp.Sessions {
		if strings.EqualFold(dp.Sessions[i].Session, session) {
			return &dp.Sessions[i]
		}
	}
	return nil
}

// planModeFor resolves the plan-restriction mode for a session: per-session override
// → strategy PlanMode → "advisory". advisory (default) never gates.
func (at *AutoTrader) planModeFor(session string) string {
	mode := "advisory"
	if dp := at.dayPlanCfg(); dp != nil && strings.TrimSpace(dp.PlanMode) != "" {
		mode = dp.PlanMode
	}
	if ov := at.sessionOverride(session); ov != nil && ov.PlanMode != nil && strings.TrimSpace(*ov.PlanMode) != "" {
		mode = *ov.PlanMode
	}
	return strings.ToLower(strings.TrimSpace(mode))
}

// replanCapFor resolves the per-session re-read cap (default 2); a per-session
// override wins (0 = no re-plan after death).
func (at *AutoTrader) replanCapFor(session string) int {
	cap := 2
	if dp := at.dayPlanCfg(); dp != nil && dp.ReplanCap > 0 {
		cap = dp.ReplanCap
	}
	if ov := at.sessionOverride(session); ov != nil && ov.ReplanCap != nil && *ov.ReplanCap >= 0 {
		cap = *ov.ReplanCap
	}
	return cap
}

// sessionEnabledForStrategy gates which sessions THIS trader runs (on top of the
// registry Enabled flag): the SessionsEnabled subset (default [NY]). A per-session
// Enable override wins over the subset.
func (at *AutoTrader) sessionEnabledForStrategy(session string) bool {
	if ov := at.sessionOverride(session); ov != nil && ov.Enable != nil {
		return *ov.Enable
	}
	dp := at.dayPlanCfg()
	if dp == nil || len(dp.SessionsEnabled) == 0 {
		return strings.EqualFold(session, kernel.SessionNY) // default [NY]
	}
	for _, s := range dp.SessionsEnabled {
		if strings.EqualFold(strings.TrimSpace(s), session) {
			return true
		}
	}
	return false
}

// proximityFilterATR is the level activation half-width in daily-ATR multiples
// (day-trade lock). Valid 0.5–3.0; anything else → the default 1.5.
func (at *AutoTrader) proximityFilterATR() float64 {
	if dp := at.dayPlanCfg(); dp != nil && dp.ProximityFilterATR >= 0.5 && dp.ProximityFilterATR <= 3.0 {
		return dp.ProximityFilterATR
	}
	return kernel.ActivationWindowK // 1.5
}

// scenarioCap is the max scenarios kept from a planner read (1–5; default 3).
func (at *AutoTrader) scenarioCap() int {
	if dp := at.dayPlanCfg(); dp != nil && dp.ScenarioCap >= 1 && dp.ScenarioCap <= 5 {
		return dp.ScenarioCap
	}
	return 3
}

// approvalRequired reports whether entries must be owner-approved before firing.
func (at *AutoTrader) approvalRequired() bool {
	dp := at.dayPlanCfg()
	return dp != nil && dp.ApprovalRequired
}

// eveningDigestEnabled gates the end-of-day roll-up digest (default true; the FE
// seeds it via DefaultDayPlanConfig, so absent → the writer keeps writing).
func (at *AutoTrader) eveningDigestEnabled() bool {
	dp := at.dayPlanCfg()
	if dp == nil {
		return true
	}
	return dp.EveningDigest
}

// ApprovalKey is the system_config key granting entries for one trader + CME
// session-day (set by POST /api/plan/approve; consumed by the entry gate).
func ApprovalKey(traderID, sessionDayKey string) string {
	return "dayplan_approval:" + traderID + ":" + sessionDayKey
}

// approvalGranted reports whether the owner has approved entries for the current
// CME session-day.
func (at *AutoTrader) approvalGranted(now time.Time) bool {
	if at.store == nil {
		return false
	}
	v, _ := at.store.GetSystemConfig(ApprovalKey(at.id, kernel.CMESessionDayKey(now)))
	return v == "granted"
}

// planModeBlocked applies the plan-restriction mode to a NEW entry. advisory (or
// no plan mode) never blocks. direction: refuse entries against the plan's bias.
// strict: refuse entries that don't cite a matched scenario. In direction/strict
// with NO active plan, nothing is authorized → block (plan restricts).
func (at *AutoTrader) planModeBlocked(d *kernel.Decision) (string, bool) {
	if d == nil {
		return "", false
	}
	now := time.Now()
	session := ""
	if s, ok := at.sessionRegistry(now).ActiveSession(now); ok {
		session = s.Name
	}
	mode := at.planModeFor(session)
	if mode == "" || mode == "advisory" {
		return "", false
	}
	if kernel.ActivePlanProvider == nil {
		return "", false // provider not installed (e.g. tests) → don't block
	}
	ap := kernel.ActivePlanProvider(at.futuresSymbol())
	if ap == nil {
		return "no active plan (" + mode + " mode restricts to the plan)", true
	}
	switch mode {
	case "direction":
		bias := strings.ToLower(strings.TrimSpace(ap.Doc.Bias.Direction))
		if bias == "long" && d.Action == "open_short" {
			return "short entry against a long plan bias", true
		}
		if bias == "short" && d.Action == "open_long" {
			return "long entry against a short plan bias", true
		}
		return "", false
	case "strict":
		res := kernel.ClassifyCitation(d.Action, d.CitedScenario, ap.Doc)
		if !res.Matched {
			return "no matched scenario cited (strict mode)", true
		}
		return "", false
	}
	return "", false
}
