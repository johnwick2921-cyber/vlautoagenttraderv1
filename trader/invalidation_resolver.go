package trader

import (
	"fmt"
	"strings"
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

// scenarioInvalidationResolverClock builds the gate's resolver for one plan,
// judged at clock(). Returns nil when the plan is absent, which switches the leg
// off entirely. There is NO wall-clock form (W1b E3): its one production caller,
// entryGateForArm, runs inside the armed pass and hands over the pass's clock —
// the wall-clock wrapper that used to sit here passed time.Now as a VALUE, which
// the seam walk (it looks for time.Now() CALLS) could not see.
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
	// W5 D13 — a MACHINE scenario is judged by its own rule, BEFORE the
	// prose-anchor path: ScenarioAnchor would call it unevaluable and the gate
	// would pass it with a WARN on every pass (F14), and its prose is not its
	// invalidation — its evidence is.
	if sc, ok := machineScenarioIn(plan, scenarioID); ok {
		return machineScenarioInvalidation(sc, bars, tapeClock(bars, now))
	}
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

// ── W5 D13 — machine (Picture) invalidation ─────────────────────────────────

// machineScenarioIn finds a machine scenario of the plan by id.
func machineScenarioIn(plan *kernel.ActivePlan, id string) (kernel.PlanScenario, bool) {
	if plan == nil {
		return kernel.PlanScenario{}, false
	}
	for _, s := range plan.Doc.Scenarios {
		if s.ID == id && kernel.IsMachineScenario(s) {
			return s, true
		}
	}
	return kernel.PlanScenario{}, false
}

// tapeClock is the instant the resolver judges a machine scenario at: the
// caller's clock, but never later than the end of the newest bar it read —
// the verdict is about the tape it holds: on a tape that stops short of the
// clock, a 5m bucket the tape holds only part of is still forming, not closed.
// (The caller's clock is the armed pass's since W1b E3; the pass-clock machine
// gate, pictureScenarioGate, still judges the deadline first and authoritatively.)
func tapeClock(bars []market.Kline, now time.Time) time.Time {
	if len(bars) > 0 {
		if end := time.UnixMilli(bars[len(bars)-1].CloseTime + 1); end.Before(now) {
			return end
		}
	}
	return now
}

// machineScenarioInvalidation is D13, pure: a Picture scenario is INVALID when
// its eligibility deadline has passed at asOf, or when the newest COMPLETED 5m
// close is back inside the broken 4H body — long: close < BodyTop; short:
// close > BodyBot (the frozen evidence's body). The tape's 5m buckets are the
// planner's own aggregation (kernel.AggregateToMinutes). When the tape holds no
// completed 5m close at or after the confirming H1 close, the newest completed
// 5m close known is that H1 close itself (the last 5m close of its hour, from
// the evidence). ok=false: no verdict (no evidence, no body, no direction).
func machineScenarioInvalidation(sc kernel.PlanScenario, bars []market.Kline, asOf time.Time) (InvalidationVerdict, bool) {
	if sc.Machine == nil {
		return InvalidationVerdict{}, false
	}
	asOfMs := asOf.UnixMilli()
	if asOfMs > sc.Machine.EligibleUntilMs {
		return InvalidationVerdict{
			Invalidated: true,
			AtCT:        kernel.FormatCT(time.UnixMilli(sc.Machine.EligibleUntilMs)),
			Reason:      "picture eligibility window closed (" + sc.Machine.Rule + ")",
		}, true
	}
	ev, err := pictureScenarioEvidence(sc)
	if err != nil {
		return InvalidationVerdict{}, false
	}
	closePx, closeMs, ok := newestCompleted5mClose(bars, asOfMs, ev.H1CloseMs)
	if !ok {
		closePx, closeMs = ev.H1NewClose, ev.H1CloseMs
	}
	if closePx <= 0 {
		return InvalidationVerdict{}, false
	}
	anchor := 0.0
	switch strings.ToLower(strings.TrimSpace(sc.Direction)) {
	case "long":
		if ev.BodyTop <= 0 {
			return InvalidationVerdict{}, false
		}
		if closePx < ev.BodyTop {
			anchor = ev.BodyTop
		}
	case "short":
		if ev.BodyBot <= 0 {
			return InvalidationVerdict{}, false
		}
		if closePx > ev.BodyBot {
			anchor = ev.BodyBot
		}
	default:
		return InvalidationVerdict{}, false
	}
	if anchor == 0 {
		return InvalidationVerdict{}, true // a verdict, and it is "alive"
	}
	atCT := ""
	if closeMs > 0 {
		atCT = kernel.FormatCT(time.UnixMilli(closeMs))
	}
	return InvalidationVerdict{
		Invalidated: true,
		AtCT:        atCT,
		Anchor:      anchor,
		Reason: fmt.Sprintf("the newest completed 5m close %.2f is back inside the broken 4H body %.2f–%.2f (%s)",
			closePx, ev.BodyBot, ev.BodyTop, sc.Machine.Rule),
	}, true
}

// newestCompleted5mClose is the newest 5m bucket that has CLOSED at asOfMs
// (bucket end ≤ asOf) and ends at or after the confirming H1 close; ok=false
// when the tape holds none (the caller falls back to the evidence).
func newestCompleted5mClose(bars []market.Kline, asOfMs, h1CloseMs int64) (float64, int64, bool) {
	b5 := kernel.AggregateToMinutes(bars, 5)
	for i := len(b5) - 1; i >= 0; i-- {
		b := b5[i]
		if b.CloseTime >= asOfMs {
			continue // still forming at asOf
		}
		if h1CloseMs > 0 && b.CloseTime+1 < h1CloseMs {
			return 0, 0, false // older than the break: nothing after it on the tape
		}
		return b.Close, b.CloseTime + 1, true
	}
	return 0, 0, false
}

// ArmGateBootLine (F5) — what the arm gate now reads and renders. Every field
// resolved from the code that implements it, never a literal (A24).
func ArmGateBootLine() string {
	return fmt.Sprintf("arm gate: invalidation-wired=%s · armed-under surfaces=%s — the evaluator's own ≈invalidated verdict REFUSES an arm (same function as the display path; unresolved = pass + a line), and a position states the version it was armed under before the live plan's rows are read as its own",
		onOffWord(true), onOffWord(true))
}
