package kernel

import (
	"regexp"
	"strconv"
	"strings"

	"nofx/market"
)

// W16/R1 — PER-SCENARIO STATE, derived from the SAME P0.4 facts the executor sees.
//
// Until now `scenario_status:<plan_id>` had exactly one writer in the whole
// repo — cmd/sandbox-seed. In production the key never existed, so
// ScenarioList.tsx fell through to `?? 'armed'` and EVERY play on the card
// painted "armed" forever: the owner could not tell which scenario was actually
// live. That is the reporting lie this file removes.
//
// DESIGN CONSTRAINT that shapes everything below: PlanScenario carries NO price
// (kernel/plan_doc.go:29-37). Trigger and Invalid are free text the planner
// wrote ("sweep 21480 then reclaim", "2x5m < 21470"). To evaluate a scenario we
// must first decide WHICH LEVEL it is about, and that resolution is a heuristic.
//
// A confidently-wrong dot would be a NEW lie replacing the old one, so the rule
// is: WHEN THE ANCHOR CANNOT BE RESOLVED, EMIT NOTHING. The writer skips the
// scenario, no status is stored, and the card keeps its existing fallback. Being
// silent is honest; being wrong is not.
//
// This file is pure and DB-blind (the sibling of scenario_facts.go) so every
// transition is unit-testable on bar fixtures.

// Scenario status vocabulary. These strings are the wire contract with
// web/src/components/plan/chips.tsx:93-98 — do not rename one side only.
const (
	ScenarioWaiting     = "waiting"     // ◌ too far away to be live
	ScenarioArmed       = "armed"       // ○ price is in the activation window
	ScenarioTriggered   = "triggered"   // ● the scenario's condition has occurred
	ScenarioInvalidated = "invalidated" // ✕ price accepted through the level
	ScenarioExpired     = "expired"     // ○ the plan itself is no longer live
)

// ScenarioEval is one scenario's resolved state plus the evidence behind it.
type ScenarioEval struct {
	ID        string
	Status    string
	Anchor    float64    // the level price this scenario was judged against
	HasAnchor bool       // false → caller MUST NOT store a status
	Facts     LevelFacts // the P0.4 facts the status was derived from
	Reason    string     // short human explanation, for logs and debugging
}

// priceToken matches a price-shaped number inside free text: 21480, 21480.25.
// Deliberately requires 4+ leading digits so "2x5m" and "15m" cannot match.
var priceToken = regexp.MustCompile(`\b\d{4,}(?:\.\d+)?\b`)

// anchorTolerance is how close a number in the trigger text must be to a plan
// level (in points) to be considered a reference TO that level rather than a
// coincidence. Index futures levels are whole/quarter points, so a tight window
// is right: a number 3+ points off is a different price, not a typo.
const anchorTolerance = 2.0

// ScenarioAnchor resolves the plan level a scenario is about.
//
// Order of preference, most explicit first:
//  1. a price in the scenario's Trigger text that snaps to a plan level
//  2. a price in its Invalid text that snaps to a plan level
//  3. nothing — ok=false
//
// Deliberately NOT attempted: "nearest level to price on the side implied by
// direction". That always produces an answer, which is precisely the failure
// mode to avoid — it would attach a confident status to a scenario we never
// actually identified.
func ScenarioAnchor(s PlanScenario, levels []PlanLevel) (float64, bool) {
	for _, text := range []string{s.Trigger, s.Invalid} {
		for _, tok := range priceToken.FindAllString(text, -1) {
			v, err := strconv.ParseFloat(tok, 64)
			if err != nil || v <= 0 {
				continue
			}
			best, found := 0.0, false
			bestDiff := anchorTolerance
			for _, l := range levels {
				if l.Price <= 0 {
					continue
				}
				if d := absF(l.Price - v); d <= bestDiff {
					best, bestDiff, found = l.Price, d, true
				}
			}
			if found {
				return best, true
			}
		}
	}
	return 0, false
}

// conditionTriggered maps a scenario's declared condition onto the P0.4 facts.
// The vocabulary is fixed by ValidatePlanDoc (kernel/plan_doc.go:54).
//
// Each mapping answers one question: "has the thing the planner described
// actually happened at this level yet?"
func conditionTriggered(condition string, f LevelFacts) bool {
	switch strings.ToLower(strings.TrimSpace(condition)) {
	case "sweep_reclaim":
		// the wick took the level and price came back — both halves required
		return f.Swept && f.Reclaimed
	case "reclaim":
		return f.Reclaimed
	case "reject":
		return f.Rejected
	case "hold":
		// the level held as support/resistance: tested and not accepted through
		return f.Rejected && f.StillValid
	case "acceptance", "breakout_retest":
		// these want price to have ACCEPTED beyond the level, which is exactly
		// what Accepted reports
		return f.Accepted
	}
	return false
}

// EvaluateScenario derives one scenario's status from live bars.
//
// The ladder, strongest evidence first:
//
//	the plan is dead                       → expired
//	price accepted THROUGH the level       → invalidated  (the level flipped roles)
//	the declared condition has occurred    → triggered
//	price is inside the activation window  → armed
//	otherwise                              → waiting
//
// planLive=false short-circuits to expired: a scenario on a dead plan is not
// "armed", and saying so was part of the original lie.
func EvaluateScenario(s PlanScenario, levels []PlanLevel, bars []market.Kline,
	price, dATR, k float64, rule string, planLive bool, nowMs int64) ScenarioEval {

	out := ScenarioEval{ID: s.ID}

	anchor, ok := ScenarioAnchor(s, levels)
	if !ok {
		// Honest silence — see the file header.
		out.Reason = "no resolvable anchor level; status intentionally not stored"
		return out
	}
	out.Anchor, out.HasAnchor = anchor, true

	if !planLive {
		out.Status, out.Reason = ScenarioExpired, "plan is no longer active"
		return out
	}

	// DIRECTION = the way price would move to KILL this trade, not the way the
	// trade wants to go. A short fading resistance dies if price accepts ABOVE
	// the level; a long defending support dies if price accepts BELOW it.
	//
	// Getting this backwards was a real bug caught by the fixture-day test: with
	// the trade direction, a level price had merely not reached yet read as
	// "accepted through" and the scenario showed invalidated on a quiet morning.
	dir := DirBelow // long: danger is downward through support
	if strings.EqualFold(strings.TrimSpace(s.Direction), "short") {
		dir = DirAbove // short: danger is upward through resistance
	}
	f := EvaluateLevelFacts(bars, anchor, dir, rule, 3, nowMs)
	out.Facts = f

	switch {
	// Acceptance IN THE DANGEROUS DIRECTION is what invalidates. Deliberately not
	// !StillValid, which reports acceptance through EITHER side and therefore
	// fires on a level price simply sits below all morning.
	case f.Accepted:
		out.Status = ScenarioInvalidated
		out.Reason = "price accepted through the level against the trade — it flipped roles"
	case conditionTriggered(s.Condition, f):
		out.Status = ScenarioTriggered
		out.Reason = "condition " + s.Condition + " occurred at " + strconv.FormatFloat(anchor, 'f', -1, 64)
	case withinActivation(price, anchor, dATR, k):
		out.Status = ScenarioArmed
		out.Reason = "price is inside the activation window"
	default:
		out.Status = ScenarioWaiting
		out.Reason = "level is outside the activation window"
	}
	return out
}

// withinActivation reports whether price is close enough to the level to call
// the scenario live. Mirrors ActivePlanLevels' half-width so the scenario dot and
// the level table can never disagree about what "near" means. A non-positive
// dATR or k means we cannot judge distance → treat as live rather than silently
// parking every scenario at "waiting".
func withinActivation(price, level, dATR, k float64) bool {
	if dATR <= 0 || k <= 0 {
		return true
	}
	return absF(price-level) <= k*dATR
}

// EvaluatePlanScenarios evaluates every scenario in a plan and returns the
// id→status map to persist. Scenarios with no resolvable anchor are OMITTED, so
// a caller can distinguish "we know it is waiting" from "we could not tell".
func EvaluatePlanScenarios(doc PlanDoc, bars []market.Kline,
	price, dATR, k float64, rule string, planLive bool, nowMs int64) (map[string]string, []ScenarioEval) {

	evals := make([]ScenarioEval, 0, len(doc.Scenarios))
	statuses := make(map[string]string, len(doc.Scenarios))
	for _, s := range doc.Scenarios {
		e := EvaluateScenario(s, doc.Levels, bars, price, dATR, k, rule, planLive, nowMs)
		evals = append(evals, e)
		if e.HasAnchor && e.Status != "" {
			statuses[e.ID] = e.Status
		}
	}
	return statuses, evals
}
