package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// validateAuthoredScenariosAt runs inside the existing candidate retry loop.
// W-EXEC-TRUTH W2 (2026-09-23): an invalid sentence outside the grammar is a
// REFUSAL (A1, one counted event per scenario, every sentence quoted in ONE
// error); every 5m group closed between the read clock and now is judged, not
// only the latest (A2); a death{} met or a flip{} fired in that span refuses
// too (D5). Tape UNKNOWN stays accepted and counted — a missing minute is never
// the model's fault. No arm/level/entry-gate policy and no runtime lifecycle
// (buffers, windows, flip hold) changes. read.IsZero() = legacy latest-window.
func (at *AutoTrader) validateAuthoredScenariosAt(doc *kernel.PlanDoc, session, tradeDate string, read, now time.Time) (*kernel.BornCheck, error) {
	var bars []market.Kline
	if market.FuturesBarsProvider != nil {
		bars = market.FuturesBarsProvider(at.futuresSymbol(), "1m", kernel.AISVPBarCount)
	}
	r := kernel.EvaluateBornCheck(doc, bars, read, now)
	record := r.Check.JSON()
	for _, sc := range r.Grammar {
		reason := fmt.Sprintf("%s invalid %q is outside the invalidation grammar (%s)", sc.ID, sc.Invalid, strings.Join(kernel.AuthoredInvalidationGrammarForms(), " | "))
		at.logWarnf("plan liveness write: %s %s %s — GRAMMAR REFUSAL: %s; re-author within existing attempts", tradeDate, session, sc.ID, reason)
		if _, err := at.store.RecordPlanLivenessEventWithCheck(store.LivenessAuthoredGrammarRefusal, fmt.Sprintf("%s:%s:%s:%s:%d", at.id, tradeDate, session, sc.ID, now.UnixNano()), now, reason, record); err != nil {
			at.logWarnf("plan liveness telemetry write failed: %v", err)
		}
	}
	for _, v := range r.TapeUnknown {
		at.logWarnf("plan liveness write: %s %s %s — %s; ACCEPTED for this check (tape)", tradeDate, session, v.ScenarioID, v.Reason)
		if _, err := at.store.RecordPlanLivenessEvent(store.LivenessAuthoredUnknown, fmt.Sprintf("%s:%s:%s:%s:%d", at.id, tradeDate, session, v.ScenarioID, now.UnixNano()), now, v.Reason); err != nil {
			at.logWarnf("plan liveness telemetry write failed: %v", err)
		}
	}
	err := r.Err()
	if err == nil {
		return &r.Check, nil
	}
	if r.BornDead() || r.Flip != nil {
		at.logWarnf("plan liveness born-dead REFUSAL: %s %s at %s — %v; re-author within existing attempts", tradeDate, session, kernel.FormatCT(now), err)
		if _, rerr := at.store.RecordPlanLivenessEventWithCheck(store.LivenessBornDeadRefusal, fmt.Sprintf("%s:%s:%s:%d", at.id, tradeDate, session, now.UnixNano()), now, err.Error(), record); rerr != nil {
			at.logWarnf("plan liveness telemetry write failed: %v", rerr)
		}
	}
	return &r.Check, err
}

// observePlanExhaustionAt is WARN-only per the corrected dispatch. It never
// changes wake clocks, arms, cutoff decisions, or the replan budget.
func (at *AutoTrader) observePlanExhaustionAt(plan *kernel.ActivePlan, states map[string]string, now time.Time) {
	if plan == nil || plan.PlanID == "" || plan.Version <= 0 || len(plan.Doc.Scenarios) == 0 {
		return
	}
	for _, sc := range plan.Doc.Scenarios {
		switch states[sc.ID] {
		case kernel.ScenarioInvalidated, kernel.ScenarioExpired:
		default:
			return
		}
	}
	budget := store.GetReplanBudget(at.store, at.id, kernel.PlanTradeDateFor(plan), plan.Session, at.replanCapFor(plan.Session))
	detail := fmt.Sprintf("v%d tradeable 0/%d by current scenario evaluator; %s, no exhaustion wake; replan budget %d remaining; wake_min_interval_min=%d; cutoff=%dm, cooldown=%dm unchanged", plan.Version, len(plan.Doc.Scenarios), planExhaustionPolicy(), budget.Left(), at.dayPlanCfg().WakeMinIntervalMinutes(), wakeCutoffMinutes(), wakeCooldownMinutes())
	wrote, err := at.store.RecordPlanLivenessEvent(store.LivenessExhaustionWarning, fmt.Sprintf("%s:%s:v%d", at.id, plan.PlanID, plan.Version), now, detail)
	if err != nil {
		at.logWarnf("plan exhaustion warning record failed: %v", err)
	} else if wrote {
		at.logWarnf("plan exhaustion observed at %s: %s", kernel.FormatCT(now), detail)
	}
}

// PlanLivenessBootLine reads persisted event counts. Before the first current
// plan snapshot there is no defensible tradeable number, so startup says n/a.
// W2 A1: the invalidation policy is READ from kernel.AuthoredInvalidationPolicy
// (the same source the born_check record stores), and the grammar refusals are
// their own count; authored UNKNOWN is now tape-only, with pre-W2 events mixing
// grammar and tape (the line says so rather than splitting history it cannot).
func PlanLivenessBootLine(st *store.Store) string {
	counts, err := st.PlanLivenessCounts()
	if err != nil {
		return fmt.Sprintf("plan liveness: tradeable=n/a · exhausted-warnings=UNKNOWN · born-dead refusals=UNKNOWN · deaths recorded=UNKNOWN · invalidation: %s · grammar refusals=UNKNOWN (event store unavailable)", kernel.AuthoredInvalidationPolicy())
	}
	return fmt.Sprintf("plan liveness: tradeable=n/a · exhausted-warnings=%d · born-dead refusals=%d · deaths recorded=%d · invalidation: %s · grammar refusals=%d · authored UNKNOWN(tape; pre-W2 events mix grammar+tape)=%d · exhaustion=%s · flip→reread=n/a(strategy loads at trader start; W-FLIP-REREAD)", counts.ExhaustionWarnings, counts.BornDeadRefusals, counts.DeathsRecorded, kernel.AuthoredInvalidationPolicy(), counts.GrammarRefusals, counts.AuthoredUnknown, planExhaustionPolicy())
}

func planExhaustionPolicy() string { return "warn-only" }
