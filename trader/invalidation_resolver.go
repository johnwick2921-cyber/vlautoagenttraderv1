package trader

import (
	"fmt"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// ── INVALIDATION RESOLVER (owner ruling 2026-09-03) ─────────────────────────
//
// The evaluator that writes "🎯 scenario S1 → ≈invalidated @ 29285.00" lives at
// trader/auto_trader_levelstate.go:260 and calls kernel.EvaluatePlanScenarios.
// This resolver calls THE SAME FUNCTION with the same windowing, so the gate
// refuses on the verdict the system already published rather than on a second
// opinion that could drift from it — the void-parity lesson.
//
// It returns ok=false whenever it cannot reach a verdict (no store, no bars, no
// price, the scenario unevaluable). The gate then PASSES and says so out loud:
// an unresolved check is not a refusal.

// scenarioInvalidationResolver builds the gate's resolver for one plan.
// Returns nil when the plan is absent, which switches the leg off entirely.
func (at *AutoTrader) scenarioInvalidationResolver(plan *kernel.ActivePlan) func(string) (InvalidationVerdict, bool) {
	return at.scenarioInvalidationResolverClock(plan, time.Now)
}

func (at *AutoTrader) scenarioInvalidationResolverClock(plan *kernel.ActivePlan, clock func() time.Time) func(string) (InvalidationVerdict, bool) {
	if plan == nil || at == nil {
		return nil
	}
	return func(scenarioID string) (InvalidationVerdict, bool) {
		return at.scenarioInvalidationAt(plan, scenarioID, clock())
	}
}

func (at *AutoTrader) scenarioInvalidationAt(plan *kernel.ActivePlan, scenarioID string, now time.Time) (InvalidationVerdict, bool) {
	if scenarioID == "" || market.FuturesBarsProvider == nil {
		return InvalidationVerdict{}, false
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return InvalidationVerdict{}, false
	}
	price := bars[len(bars)-1].Close
	dATR := kernel.PlanDATRFor(at.id)
	if price <= 0 {
		return InvalidationVerdict{}, false
	}
	// The SAME windowing the display path uses: only bars closed after the
	// plan was born, so a pre-plan sweep never reads as a verdict.
	windowed := kernel.BarsSince(bars, plan.BirthMs)
	rule := at.acceptanceRuleFor(at.activeSessionName(now))
	_, evals := kernel.EvaluatePlanScenarios(
		plan.Doc, windowed, price, dATR, kernel.ActivationWindowK, rule, true, now.UnixMilli())

	for _, e := range evals {
		if e.ID != scenarioID {
			continue
		}
		if !e.HasAnchor {
			// The display path calls this UNEVALUABLE and refuses to store
			// a status. The gate must not invent one either.
			return InvalidationVerdict{}, false
		}
		if e.Status != kernel.ScenarioInvalidated {
			return InvalidationVerdict{}, true // a verdict, and it is "alive"
		}
		// WHEN it became invalidated, from the stamp the evaluator wrote
		// on the transition. Absent → say nothing rather than pass the
		// CHECK time off as the VERDICT time; the gate then renders
		// "at an earlier cycle", which is true.
		atCT := ""
		if at.store != nil {
			r, err := at.store.ScenarioDeathFor(at.id, plan.PlanID, plan.Version, scenarioID, e.Anchor)
			if err != nil {
				at.logWarnf("scenario death evidence unavailable: v%d %s: %v", plan.Version, scenarioID, err)
			} else if r != nil {
				atCT = kernel.FormatCT(r.ObservedAt)
			}

		}
		return InvalidationVerdict{
			Invalidated: true,
			AtCT:        atCT,
			Anchor:      e.Anchor,
			Reason:      e.Reason,
		}, true
	}
	return InvalidationVerdict{}, false // scenario not in this plan
}

// ArmGateBootLine (F5) — what the arm gate now reads and renders. Every field
// resolved from the code that implements it, never a literal (A24).
func ArmGateBootLine() string {
	return fmt.Sprintf("arm gate: invalidation-wired=%s · armed-under surfaces=%s — the evaluator's own ≈invalidated verdict REFUSES an arm (same function as the display path; unresolved = pass + a line), and a position states the version it was armed under before the live plan's rows are read as its own",
		onOffWord(true), onOffWord(true))
}
