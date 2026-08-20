package kernel

import (
	"fmt"
	"strings"
	"sync"

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
	BirthMs     int64 // plan-row created_at; consumption is judged from here

	// P0-cleanup (2026-08-19) — full attribution for decision records.
	PlanID         string
	OverlayVersion int
}

// TraderPlanProviders is the per-trader seam between the trader layer and the
// kernel's executor-prompt assembly. P0-A (2026-08-18): these used to be
// PROCESS-GLOBAL singletons installed once under a sync.Once closing over the
// FIRST day-plan trader — so with two live day-plan traders, one trader's plan,
// session enablement and proximity governed the other's executor. Now every
// kernel consumer resolves by traderID: a trader can NEVER receive another
// trader's plan, registry view or config.
type TraderPlanProviders struct {
	// ActivePlan returns THIS trader's current plan for a symbol's session
	// (nil → no plan → the executor prompt is unchanged).
	ActivePlan func(symbol string) *ActivePlan
	// SessionRegistry returns the admin registry THIS trader resolves (the
	// registry is admin-global data, but the seam is per-trader so no
	// process-global singleton can ever again close over one trader).
	SessionRegistry func() SessionRegistry
}

// traderPlanProviders maps traderID → providers. traderID == "" is never stored
// (the empty key would be the same cross-trader hole in disguise).
var traderPlanProviders sync.Map

// SetTraderPlanProviders registers (or replaces) one trader's providers. A nil
// fn removes the trader's registration.
func SetTraderPlanProviders(traderID string, p TraderPlanProviders) {
	if strings.TrimSpace(traderID) == "" {
		return
	}
	if p.ActivePlan == nil && p.SessionRegistry == nil {
		traderPlanProviders.Delete(traderID)
		return
	}
	traderPlanProviders.Store(traderID, p)
}

// TraderPlanProvidersFor returns the providers registered for traderID
// (ok=false → none → callers fall back to default behavior).
func TraderPlanProvidersFor(traderID string) (TraderPlanProviders, bool) {
	if strings.TrimSpace(traderID) == "" {
		return TraderPlanProviders{}, false
	}
	if v, ok := traderPlanProviders.Load(traderID); ok {
		return v.(TraderPlanProviders), true
	}
	return TraderPlanProviders{}, false
}

// ActivePlanFor resolves the ACTIVE PLAN for the DECIDING trader: the trader's
// own provider, never another trader's. nil provider → nil plan.
func ActivePlanFor(traderID, symbol string) *ActivePlan {
	p, ok := TraderPlanProvidersFor(traderID)
	if !ok || p.ActivePlan == nil {
		return nil
	}
	return p.ActivePlan(symbol)
}

// HasTraderPlanProvider reports whether ANY provider is registered for the
// trader (used by trader-side writers that need the active plan).
func HasTraderPlanProvider(traderID string) bool {
	_, ok := TraderPlanProvidersFor(traderID)
	return ok
}

// TraderPlanProviderCount returns how many traders have providers registered
// (P0-A: the trader layer announces loudly when this exceeds 1).
func TraderPlanProviderCount() int {
	n := 0
	traderPlanProviders.Range(func(_, _ any) bool { n++; return true })
	return n
}

// ResolvedSessionRegistryFor returns the registry the DECIDING trader resolves
// (per-trader provider; admin-global data). No provider / empty registry → the
// shipped default (byte-identical pre-H7 behavior; crypto never installs one).
func ResolvedSessionRegistryFor(traderID string) SessionRegistry {
	p, ok := TraderPlanProvidersFor(traderID)
	if ok && p.SessionRegistry != nil {
		if reg := p.SessionRegistry(); len(reg.Sessions) > 0 {
			return reg
		}
	}
	return DefaultSessionRegistry()
}

// RenderPlanBlock renders the byte-stable PLAN BLOCK for the cached prefix.
func RenderPlanBlock(doc PlanDoc, session string) string {
	var b strings.Builder
	// W-why-no-trades (2026-08-18): "follow it; entries per plan only" read as
	// DIRECTION by persuasion — the advisory plan was cited by the model as the
	// reason to refuse every setup. The plan stays informative; valid off-plan
	// entries are now explicitly permitted (cite "off-plan").
	fmt.Fprintf(&b, "# DAY PLAN (%s) — preferred: follow it · a valid off-plan setup may still be traded (cite \"off-plan\")\n", session)
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
	deathSuffix := ""
	if doc.DeathStructured == nil {
		// A3 (F5): no structured object → nothing machine-evaluates this line;
		// say so, so nobody (owner or model) assumes machine enforcement.
		deathSuffix = " (prose-only — AI-judged, no machine evaluation)"
	}
	b.WriteString("Plan dies if: " + doc.DeathCondition + deathSuffix + "\n")
	b.WriteString(`Cite rule: your decision JSON MUST include "cited_scenario" = "S1"|"S2"|…|"off-plan".`)
	return b.String()
}

// RenderPlanStatus renders the dynamic PLAN STATUS tail: current price, re-plans
// left, and per-level Go facts (distance/sweep/closes-beyond/acceptance/valid)
// from the P0.4 evaluator.
func RenderPlanStatus(traderID, symbol string, doc PlanDoc, bars []market.Kline, price, dATR float64, rule string, replansLeft int, now, birthMs int64) string {
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
		fresh := levelFreshnessFn(traderID, symbol) // W11b — persisted cross-session state (nil → none)
	for _, l := range active {
		dir := DirAbove
		if l.Price < price {
			dir = DirBelow
		}
		f := EvaluateLevelFacts(BarsSince(bars, birthMs), l.Price, dir, rule, lookback, now)
		sweep := "F"
		if f.Swept {
			sweep = "T"
		}
		valid := "valid"
		if !f.StillValid {
			valid = "CONSUMED"
		}
		fmt.Fprintf(&b, "  %.2f %s: dist %+.1f · sweep=%s · closes-beyond %d · acceptance %d/%d · %s",
			l.Price, l.Label, f.DistancePoints, sweep, maxInt(f.ClosesBeyondUp, f.ClosesBeyondDown), f.AcceptHave, f.AcceptNeed, valid)
		// W11b — append persisted cross-session state so a level burned in an EARLIER
		// session reads burned NOW (not just consumed-this-session). Only emitted when
		// there IS persisted state → nil provider = byte-identical to the pre-W11b line.
		if fresh != nil {
			if state := planStateLabel(fresh(DetectedLevel{Price: l.Price, Label: l.Label})); state != "" {
				b.WriteString(" · " + state)
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// planStateLabel maps a persisted freshness grade to a PLAN-STATUS annotation.
// "" / A / fresh → "" (no annotation, keeps the fresh line unchanged).
// P1c design rule: a consumed level ROLE-FLIPS and stays on the map — the label
// says "flipped", never "burned", because a consumed level is an OPPORTUNITY
// (the trap S1 trades), not a scar.
func planStateLabel(grade string) string {
	switch strings.ToLower(strings.TrimSpace(grade)) {
	case "", "a", "fresh":
		return ""
	case "b":
		return "state=B(tested)"
	case "c", "tested":
		return "state=C(tested×2)"
	case "done", "consumed":
		// W-why-no-trades: "watch the trap" made the model AVOID the level — a
		// consumed level is a role-flipped opportunity, not a hazard to flee.
		return "flipped(consumed — tradeable both directions)"
	default:
		return "state=" + grade
	}
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
