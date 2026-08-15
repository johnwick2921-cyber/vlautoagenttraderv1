package kernel

import (
	"fmt"
	"strings"

	"nofx/market"
)

// P3.4 — executor injection of the active plan.
//
// RenderPlanBlock is BYTE-STABLE for a given plan (the cached prefix content):
// bias+flip, levels, scenarios, no-trade, death, cite-rule. RenderPlanStatus is
// the DYNAMIC tail: per-level Go facts from the P0.4 evaluator + re-plans left +
// restated prices. The kernel is DB-blind, so the trader supplies the active plan
// through ActivePlanProvider (mirror of NakedPOCProvider).

// ActivePlan is the executor's current plan + lifecycle counters.
type ActivePlan struct {
	Doc         PlanDoc
	Session     string
	Version     int
	ReplansLeft int
}

// ActivePlanProvider, when set by the trader layer, returns the active plan for a
// symbol's current session (nil → no plan → the executor prompt is unchanged).
var ActivePlanProvider func(symbol string) *ActivePlan

// RenderPlanBlock renders the byte-stable PLAN BLOCK for the cached prefix.
func RenderPlanBlock(doc PlanDoc, session string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# DAY PLAN (%s) — follow it; entries per plan only\n", session)
	fmt.Fprintf(&b, "Bias: %s (%s) · flips: %s\n", doc.Bias.Direction, doc.Bias.Conviction, doc.Bias.FlipCondition)
	if len(doc.Levels) > 0 {
		b.WriteString("Levels:\n")
		for _, l := range doc.Levels {
			fmt.Fprintf(&b, "  %.2f %s [%s] %s\n", l.Price, l.Label, l.Grade, l.Instruction)
		}
	}
	b.WriteString("Scenarios:\n")
	for _, s := range doc.Scenarios {
		fmt.Fprintf(&b, "  %s [%s] %s %s: %s → %s · invalid %s\n",
			s.ID, s.Quality, s.Condition, s.Direction, s.Trigger, joinFloats(s.TargetChain), s.Invalid)
	}
	if len(doc.NoTrade) > 0 {
		b.WriteString("No-trade: " + strings.Join(doc.NoTrade, " · ") + "\n")
	}
	b.WriteString("Plan dies if: " + doc.DeathCondition + "\n")
	b.WriteString(`Cite rule: your decision JSON MUST include "cited_scenario" = "S1"|"S2"|…|"off-plan".`)
	return b.String()
}

// RenderPlanStatus renders the dynamic PLAN STATUS tail: current price, re-plans
// left, and per-level Go facts (distance/sweep/closes-beyond/acceptance/valid)
// from the P0.4 evaluator.
func RenderPlanStatus(doc PlanDoc, bars []market.Kline, price, dATR float64, rule string, replansLeft int, now int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# PLAN STATUS (live) — facts=Go, judgment=you\nprice %.2f · re-plans left %d\n", price, replansLeft)
	if rule == "" {
		rule = "2x5m"
	}
	lookback := 3
	// Activation window (P3.6): only levels within 1.5×dATR are live candidates.
	active := ActivePlanLevels(doc.Levels, price, dATR, ActivationWindowK)
	if hidden := len(doc.Levels) - len(active); hidden > 0 {
		fmt.Fprintf(&b, "(%d level(s) outside the %.1f×dATR activation window — re-arm when price returns)\n", hidden, ActivationWindowK)
	}
	for _, l := range active {
		dir := DirAbove
		if l.Price < price {
			dir = DirBelow
		}
		f := EvaluateLevelFacts(bars, l.Price, dir, rule, lookback, now)
		sweep := "F"
		if f.Swept {
			sweep = "T"
		}
		valid := "valid"
		if !f.StillValid {
			valid = "CONSUMED"
		}
		fmt.Fprintf(&b, "  %.2f %s: dist %+.1f · sweep=%s · closes-beyond %d · acceptance %d/%d · %s\n",
			l.Price, l.Label, f.DistancePoints, sweep, maxInt(f.ClosesBeyondUp, f.ClosesBeyondDown), f.AcceptHave, f.AcceptNeed, valid)
	}
	return strings.TrimRight(b.String(), "\n")
}

// PlanCitationResult classifies an executor's plan citation (P3.5 advisory).
type PlanCitationResult struct {
	Cited   string // the resolved scenario id, or "off-plan"
	OffPlan bool
	Matched bool // cited a real scenario whose direction aligns with the action
}

// ClassifyCitation resolves the executor's cited_scenario against the active
// plan: empty/"off-plan"/unknown-id → off-plan; a known scenario → matched iff
// the action's direction aligns with the scenario's. Advisory only — it never
// gates the trade.
func ClassifyCitation(action, cited string, doc PlanDoc) PlanCitationResult {
	c := strings.TrimSpace(cited)
	if c == "" || strings.EqualFold(c, "off-plan") || strings.EqualFold(c, "offplan") {
		return PlanCitationResult{Cited: "off-plan", OffPlan: true}
	}
	for _, s := range doc.Scenarios {
		if strings.EqualFold(s.ID, c) {
			return PlanCitationResult{Cited: s.ID, Matched: actionMatchesDirection(action, s.Direction)}
		}
	}
	return PlanCitationResult{Cited: "off-plan", OffPlan: true} // unknown id → off-plan
}

func actionMatchesDirection(action, dir string) bool {
	switch action {
	case "open_long":
		return dir == "long"
	case "open_short":
		return dir == "short"
	}
	return false
}

func joinFloats(xs []float64) string {
	if len(xs) == 0 {
		return "—"
	}
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = fmt.Sprintf("%.2f", x)
	}
	return strings.Join(parts, ",")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
