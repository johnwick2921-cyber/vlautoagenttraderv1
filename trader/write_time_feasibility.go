package trader

import (
	"fmt"
	"strings"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W-WRITE-TIME-FEASIBILITY (2026-09-18, owner ruling "fix all" 08:3x CT).
//
// The planner wrote scenarios the executor's gate-at-arm chain would refuse and
// only WARNed (kernel.ArmFeasibilityWarnings). The model never learned why its
// plan did not trade and re-authored the same shape. This wave judges the SAME
// predicates the executor runs at arm — via the executor's own functions (canon
// 53, no re-implementation) — BEFORE the plan is written:
//
//   - attempts 1..N-1: unarmable scenarios become a restriction-with-hint error
//     that feeds the existing repair prompt (class 38 machinery);
//   - the last attempt: the scenarios are written with arm.enabled=false +
//     arm_disabled_reason + ONE WARN each + a system counter.
//
// Knob `day_plan.write_time_feasibility` — nil/unset = ON (owner default);
// explicit false = today's WARN-only behaviour byte-identical (the verdict
// function returns nil and the prompt sentence is absent).
//
// GEOMETRY NOTE (CTO BLOCKER 3, 2026-09-18): after the W-GEOMETRY-REFUSAL merge
// the executor resolves structural geometry through
// `ArmGeometryVerdict(doc, sc, cfg.DayPlan.GeometryRefIDsEnabled())`. The write
// site goes through `composeArmStop` — the executor's own composition function
// — so when #175 lands and the executor's compose path switches to
// ArmGeometryVerdict, the write site inherits the same resolution and knob.

// writeTimeFeasibilityOn resolves the knob the way the trading path does
// (store.DayPlanConfig.WriteTimeFeasibilityEnabled): nil config → ON.
func (at *AutoTrader) writeTimeFeasibilityOn() bool {
	if at == nil || at.config.StrategyConfig == nil || at.config.StrategyConfig.DayPlan == nil {
		return true
	}
	return at.config.StrategyConfig.DayPlan.WriteTimeFeasibilityEnabled()
}

// writeFeasLabel renders the boot-line word from the RESOLVED knob (read,
// never typed).
func writeFeasLabel(dp *store.DayPlanConfig) string {
	if dp.WriteTimeFeasibilityEnabled() {
		return "on"
	}
	return "off"
}

type writeTimeFeasibilityIssue struct {
	Scenario string
	// Class is the REASON CLASS for arm_disabled_reason and the counter key
	// (CTO SHOULD-FIX 6): min_sl / rr / geometry_<code> / stop_side_wrong —
	// bounded cardinality, one system_config row per class, never per ATR value.
	Class string
	// Verbose is the full refusal text the repair hint carries (with numbers).
	Verbose string
	// Kind is "gate" (armGateVerdictFor / geometry) or "stop_side" (the
	// executor's stop-side placement guard, CTO amendment 2026-09-18).
	Kind    string
	Cond    string  // scenario condition, for the stop-side hint words
	Trigger float64 // stop-side only: the tick-rounded trigger the wire would carry
	Price   float64 // stop-side only: the verdict-time tape price the guard judged
	Side    string  // stop-side only: canonical lowercase side
}

// writeTimeFeasibilityVerdicts runs the executor's own gate-at-arm predicates
// for every enabled single arm in the doc, at write time. The session-risk band
// is time-based and deliberately NOT judged here (spec). Returns nil when the
// knob is OFF — byte-identical to today.
//
// The caller MUST stamp the frozen zone map and the scenario identity into d
// first (CTO BLOCKER 1) — the geometry predicate resolves against them and the
// final stamp would otherwise run only after the attempt loop.
func (at *AutoTrader) writeTimeFeasibilityVerdicts(d *kernel.PlanDoc, atr5m float64, cfg *store.StrategyConfig, session string) []writeTimeFeasibilityIssue {
	if !at.writeTimeFeasibilityOn() || d == nil {
		return nil
	}
	minQuality := ""
	if cfg != nil && cfg.DayPlan != nil {
		minQuality = cfg.DayPlan.MinGradeFor(session)
	}
	bias := biasDirectionFor(d.Bias.Direction)
	var out []writeTimeFeasibilityIssue
	for i := range d.Scenarios {
		sc := &d.Scenarios[i]
		if sc.Arm == nil || !sc.Arm.Enabled {
			continue
		}
		leg := kernel.PlanArmLeg{Entry: sc.Arm.Entry, Stop: sc.Arm.Stop, Target: sc.Arm.Target}
		structuralFade := sc.Condition == kernel.OneSetupPlay && !strings.EqualFold(leg.Kind, "exit")
		if structuralFade {
			// The executor applies the structural-geometry path only to the
			// level-fade play and COMPOSES entry/stop/target before its gates.
			// Mirror through the SAME composition (CTO SHOULD-FIX 5 — never a
			// different number than the executor) and thread the SAME knob the
			// executor threads (CTO BLOCKER 3: GeometryRefIDs =
			// cfg.DayPlan.GeometryRefIDsEnabled(), nil = ON).
			policy := store.ResolveStructuralStop(cfg, at.futuresSymbol())
			policy.MinRR = at.armMinRRFor(cfg)
			// ONE resolution site (CTO RECHECK item 1): the nil-receiver-safe
			// seam, exactly the form the executor uses.
			refIDs := at.dayPlanCfg().GeometryRefIDsEnabled()
			comp := composeArmStop(sc.Direction, leg.Entry, leg.Stop, atr5m,
				market.FuturesTickSize(at.futuresSymbol()), d.Levels, kernel.MinSLATRMult(),
				kernel.MinSLTickClearance, armStopAnchorMaxATR(),
				armStructuralContext{Doc: d, Scenario: *sc, Leg: leg, Policy: policy,
					PointValue: market.FuturesPointValue(at.futuresSymbol()), GeometryRefIDs: refIDs})
			if comp.Geometry != nil && comp.Geometry.Reason != "" {
				out = append(out, writeTimeFeasibilityIssue{
					Scenario: sc.ID,
					Class:    geometryClass(comp.Geometry.Reason),
					Verbose:  fmt.Sprintf("geometry: %s (%s)", comp.Geometry.Reason, comp.Geometry.Detail),
					Kind:     "gate",
				})
				continue
			}
			if comp.Geometry != nil && comp.Geometry.Entry > 0 {
				leg.Entry = comp.Geometry.Entry
			}
			if comp.Geometry != nil && comp.Geometry.Stop != nil {
				leg.Stop = *comp.Geometry.Stop
			}
			if comp.Geometry != nil && comp.Geometry.Target != nil {
				leg.Target = *comp.Geometry.Target
			}
		} else {
			// N1 (CTO RECHECK 2026-09-18): for every NON-fade leg the executor
			// composes the stop BEFORE its gates via the legacy composeArmStop
			// branch (armed_executor.go:531-535 — ATR floor entry ∓ mult×ATR5m,
			// nearest seated level + clearance, widest wins, never tighter than
			// authored), so the min-SL leg can never fire at arm on the authored
			// stop. Compose the SAME stop here and judge the composed leg.
			comp := composeArmStop(sc.Direction, leg.Entry, leg.Stop, atr5m,
				market.FuturesTickSize(at.futuresSymbol()), d.Levels, kernel.MinSLATRMult(),
				kernel.MinSLTickClearance, armStopAnchorMaxATR())
			if comp.Stop > 0 {
				leg.Stop = comp.Stop
			}
		}
		if v := at.armGateVerdictFor(*sc, leg, bias, nil, atr5m, minQuality, cfg, session, structuralFade); v != "" {
			out = append(out, writeTimeFeasibilityIssue{Scenario: sc.ID, Class: armRefusalClass(v), Verbose: v, Kind: "gate"})
			continue
		}
		// STOP-SIDE (CTO amendment 2026-09-18) — the executor's OWN placement
		// guard, adjudicated here with the same function and the same inputs the
		// placement branch uses (armed_executor.go decideStopEntry: entry ±
		// STOP_ENTRY_OFFSET_TICKS, tick-rounded). A stop-entry trigger already
		// through the verdict-time price cancels at placement (`✕ armed …
		// stop-entry CANCELLED [guard=stop-side …]`) — the model must be told at
		// write instead of authoring a dead trigger 21× in a row.
		//
		// SHOULD-FIX 7: the price is the LAST TAPE CLOSE re-read at verdict
		// time (the placement branch judges the live tape, not the read-start
		// facts price). No tape → the predicate is skipped, never guessed.
		if kind, _ := armLegKindFor(*sc, kernel.PlanArmLeg{}); kind == kernel.ArmKindStopEntry {
			price := lastTapeClose(at.futuresSymbol())
			if price > 0 {
				tick := market.FuturesTickSize(at.futuresSymbol())
				dec := decideStopEntry(sc.Direction, leg.Entry, float64(stopEntryOffsetTicks())*tick, tick, price)
				if dec.Verdict == stopGuardThrough {
					out = append(out, writeTimeFeasibilityIssue{
						Scenario: sc.ID, Class: "stop_side_wrong", Kind: "stop_side",
						Cond: sc.Condition, Trigger: dec.Trigger, Price: price, Side: dec.Side,
					})
				}
			}
		}
	}
	return out
}

// lastTapeClose re-reads the last 1m close at verdict time (SHOULD-FIX 7). 0
// when the provider is absent — the predicate is then skipped, never guessed.
func lastTapeClose(symbol string) float64 {
	if market.FuturesBarsProvider == nil {
		return 0
	}
	b := market.FuturesBarsProvider(symbol, "1m", 2)
	if len(b) == 0 {
		return 0
	}
	return b[len(b)-1].Close
}

// verdictClass is REMOVED (CTO RECHECK S6): the executor's own
// armRefusalClass (armed_executor.go:1746) is the ONE classifier — the dedup
// vocabulary can never diverge again.

// geometryClass maps a composeArmStop geometry refusal code to the counter
// class geometry_<code> (SHOULD-FIX 6).
func geometryClass(reason string) string {
	return "geometry_" + strings.ReplaceAll(strings.TrimSpace(reason), " ", "_")
}

// writeTimeFeasibilityHint renders the restriction-with-hint text the repair
// prompt carries: the refusal, the numbers, and the vocabulary to fix it.
func writeTimeFeasibilityHint(issues []writeTimeFeasibilityIssue) string {
	var b strings.Builder
	b.WriteString("write-time feasibility: ")
	gateIssues := 0
	for i, is := range issues {
		if i > 0 {
			b.WriteString("; ")
		}
		if is.Kind == "stop_side" {
			// The amendment's own vocabulary (CTO 2026-09-18): name the
			// trigger, its relation to price, and the two fixes.
			aboveBelow := "above"
			if strings.EqualFold(is.Side, "long") {
				aboveBelow = "below"
			}
			b.WriteString(fmt.Sprintf("%s %s trigger %.2f is already %s price %.2f: a stop entry there fills at market on placement — author the trigger ahead of price, or author a reject/limit at the level", is.Scenario, is.Cond, is.Trigger, aboveBelow, is.Price))
			continue
		}
		gateIssues++
		b.WriteString(fmt.Sprintf("%s would be refused at arm — %s", is.Scenario, is.Verbose))
	}
	if gateIssues > 0 {
		b.WriteString(" — widen the stop past the min-SL floor / raise the arm R:R / pick a mapped level with an id / choose a different condition")
	}
	return b.String()
}

// applyWriteTimeArmDisable is the LAST-attempt path: scenarios that are still
// unarmable are written with arm.enabled=false + arm_disabled_reason (the
// reason CLASS — SHOULD-FIX 6), ONE WARN each, and a counter recorded
// (counters record, never infer; the key is keyed by CLASS, never by the
// ATR-bearing verdict text).
func (at *AutoTrader) applyWriteTimeArmDisable(d *kernel.PlanDoc, issues []writeTimeFeasibilityIssue, tradeDate, session string) {
	if d == nil || len(issues) == 0 {
		return
	}
	byID := make(map[string]writeTimeFeasibilityIssue, len(issues))
	for _, is := range issues {
		byID[is.Scenario] = is
	}
	for i := range d.Scenarios {
		sc := &d.Scenarios[i]
		if sc.Arm == nil || !sc.Arm.Enabled {
			continue
		}
		issue, ok := byID[sc.ID]
		if !ok {
			continue
		}
		sc.Arm.Enabled = false
		sc.Arm.DisabledReason = issue.Class
		// The WARN carries BOTH the class and the verbose verdict (CTO RECHECK
		// S6): an arm first authored infeasible on the last attempt still shows
		// the numbers on this line.
		at.logWarnf("⚔️ arm disabled at write: %s %s %s (%s)", session, sc.ID, issue.Class, issue.Verbose)
		if at.store == nil {
			continue
		}
		key := fmt.Sprintf("arm_disabled_at_write:%s:%s:%s:%s", at.id, tradeDate, session, issue.Class)
		if _, err := store.IncSystemCounter(at.store, key); err != nil {
			at.logWarnf("⚔️ arm-disabled counter write failed: %v", err)
		}
	}
}
