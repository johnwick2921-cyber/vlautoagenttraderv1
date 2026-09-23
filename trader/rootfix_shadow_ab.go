package trader

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
)

// ── ROOT-FIX PART B (2026-09-02) — FAST-MODE SHADOW A/B ──────────────────────
// Measured baseline (n=67 full-author calls, 2026-08-31 → 09-02): p50 completion
// 23,769 tokens, mean wall 349 s, mean reasoning 72,477 chars. The stored plan
// JSON averages 3,088 bytes ≈ 920 tokens — about 4% of the output. The other
// ~96% is REASONING, which only the reasoning MODE can move. This harness
// measures whether reasoning=fast authors the same legal plans in half the
// time, on the SAME prompts, before anything is promoted.
//
// It writes NO plan, holds NO replan budget, and NEVER runs concurrently with
// a live planner stream: it fires only after the live read has finished (the
// transport-cut investigation has not excluded concurrency, class 41 P3).

// ShadowABEnabled — SHADOW_AB_ENABLED (default OFF). Ships dormant.
func ShadowABEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("SHADOW_AB_ENABLED")))
	return v == "1" || v == "true" || v == "on" || v == "yes"
}

// ShadowABTarget — SHADOW_AB_N (default 10): the pre-registered sample size the
// B-3 criterion is judged at. The harness stops firing once it is reached, so
// an enabled knob cannot bill the provider indefinitely.
func ShadowABTarget() int {
	if v := strings.TrimSpace(os.Getenv("SHADOW_AB_N")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			return n
		}
	}
	return 10
}

// shadowABInFlight guarantees ONE shadow call at a time, process-wide.
var shadowABInFlight atomic.Bool

// shadowRunner carries what a shadow call needs, captured at the live call
// site so this file never re-derives the client, model or budget.
type shadowRunner struct {
	client    mcp.AIClient
	system    string
	maxTokens int
	// fast wire — the mode under test.
	mode, effort string
}

var shadowRunners sync.Map // trader id → *shadowRunner

// live call metrics, captured at the live call site so the shadow row can put
// fast and max side by side on the SAME prompt.
var (
	liveCallTokens sync.Map // trader id → int
	liveCallWallMs sync.Map // trader id → int64
)

// RecordLiveCallMetrics is called by the live planner closure after each call.
func (at *AutoTrader) RecordLiveCallMetrics(tokens int, wallMs int64) {
	liveCallTokens.Store(at.id, tokens)
	liveCallWallMs.Store(at.id, wallMs)
}

func (at *AutoTrader) liveCallMetrics() (int, int64) {
	t, _ := liveCallTokens.Load(at.id)
	w, _ := liveCallWallMs.Load(at.id)
	ti, _ := t.(int)
	wi, _ := w.(int64)
	return ti, wi
}

// RegisterShadowRunner is called at the live planner call site.
func (at *AutoTrader) RegisterShadowRunner(client mcp.AIClient, system string, maxTokens int, mode, effort string) {
	shadowRunners.Store(at.id, &shadowRunner{client: client, system: system, maxTokens: maxTokens, mode: mode, effort: effort})
}

// ShadowABVerdict is one shadow row: what fast mode produced on the identical
// prompt, and whether the FULL validator chain would have accepted it.
type ShadowABVerdict struct {
	Legal      bool
	Reasons    []string
	Tokens     int
	ReasonChar int
	WallMs     int64
	Err        error
}

// shadowABLine renders the one loud line (pure — fixture-pinned).
func shadowABLine(n, target int, session, tradeDate string, v ShadowABVerdict, liveTokens int, liveWallMs int64) string {
	verdict := "LEGAL"
	if v.Err != nil {
		verdict = "CALL-FAILED"
	} else if !v.Legal {
		verdict = "ILLEGAL"
	}
	reasons := "—"
	if len(v.Reasons) > 0 {
		reasons = strings.Join(v.Reasons, " | ")
		if len(reasons) > 400 {
			reasons = reasons[:400] + "…(truncated)"
		}
	}
	speed := "n/a"
	if liveWallMs > 0 && v.WallMs > 0 {
		speed = fmt.Sprintf("%.0f%% of live", float64(v.WallMs)/float64(liveWallMs)*100)
	}
	return fmt.Sprintf("🔬 shadow A/B %d/%d (%s %s): fast=%s tokens=%d wall=%.1fs (%s) · live max tokens=%d wall=%.1fs · reasons=%s — SHADOW ONLY, no plan written, no replan budget",
		n, target, tradeDate, session, verdict, v.Tokens, float64(v.WallMs)/1000, speed, liveTokens, float64(liveWallMs)/1000, reasons)
}

// shadowVerdictFor replays the LIVE validator chain on the shadow output, in
// the same order and with the same inputs, and returns every reason it would
// have been rejected for. It writes nothing.
func (at *AutoTrader) shadowVerdictFor(raw string, maxLevels, scenarioCap int, facts kernel.PlanFacts, machineLabels, htfLabels map[float64]string, requiredBias string) (bool, []string) {
	var reasons []string
	d, perr := kernel.ParsePlanDocCappedWithMinRR(raw, maxLevels, scenarioCap, at.armMinRRFor(nil))
	if perr != nil {
		return false, []string{"parse/schema: " + perr.Error()}
	}
	if requiredBias != "" && !strings.EqualFold(strings.TrimSpace(d.Bias.Direction), requiredBias) {
		reasons = append(reasons, fmt.Sprintf("required bias %s, got %q", requiredBias, d.Bias.Direction))
	}
	if mis := kernel.MislabeledStructuralLevels(d, machineLabels); len(mis) > 0 {
		reasons = append(reasons, "level label provenance: "+strings.Join(mis, "; "))
	}
	if verr := kernel.ValidatePlanDocWithFactsMachine(d, facts, machineLabels, maxLevels, scenarioCap); verr != nil {
		reasons = append(reasons, verr.Error())
	}
	if kernel.HasFvgScenario(d) {
		bars := market.FuturesBarsProvider(at.futuresSymbol(), "1m", kernel.AISVPBarCount)
		// The SAME origin universe the live path builds — a shadow verdict that
		// used a different origin set would not be a comparison.
		origin := make(map[string]bool, len(machineLabels)+len(htfLabels))
		for _, lbl := range machineLabels {
			origin[lbl] = true
		}
		for _, lbl := range htfLabels {
			origin[lbl] = true
		}
		if verr := kernel.ValidateFvgEntryScenarios(d, bars, at.futuresSymbol(), origin, time.Now()); verr != nil {
			reasons = append(reasons, verr.Error())
		}
	}
	if kernel.HasBreakdownScenario(d) {
		scope := kernel.ResolveVoidScope(at.futuresSymbol(), time.Now())
		if verr := kernel.ValidateBreakdownContinueScenarios(d, scope, kernel.StaleConfirmATR5m(scope.Bars), facts.Price, time.Now().UnixMilli()); verr != nil {
			reasons = append(reasons, verr.Error())
		}
	}
	// W-EXEC-TRUTH W2 A1/A2 — the live chain's born check, PURE here (no
	// liveness events): without it the shadow's legal rate overstates.
	if market.FuturesBarsProvider != nil {
		if berr := kernel.EvaluateBornCheck(d, market.FuturesBarsProvider(at.futuresSymbol(), "1m", kernel.AISVPBarCount), facts.ReadAt, time.Now()).Err(); berr != nil {
			reasons = append(reasons, berr.Error())
		}
	}
	// W-EXEC-TRUTH W2 A3+A4 — the SAME kernel check the write loop runs; the
	// shadow records nothing.
	if verr := kernel.CheckScenarioWriteTruth(d, facts.IdentityMap, facts.CapacityCut, market.FuturesTickSize(at.futuresSymbol())).Err(); verr != nil {
		reasons = append(reasons, verr.Error())
	}
	return len(reasons) == 0, reasons
}

// maybeRunShadowAB fires ONE shadow call, serialized after the live read.
// Every guard is fail-closed: knob off, sample complete, another shadow in
// flight, or no runner registered → nothing happens and nothing is billed.
func (at *AutoTrader) maybeRunShadowAB(session, tradeDate, userPrompt string, maxLevels, scenarioCap int,
	facts kernel.PlanFacts, machineLabels, htfLabels map[float64]string, requiredBias string) {
	liveTokens, liveWallMs := at.liveCallMetrics()
	if !ShadowABEnabled() || strings.TrimSpace(userPrompt) == "" {
		return
	}
	v, ok := shadowRunners.Load(at.id)
	runner, _ := v.(*shadowRunner)
	if !ok || runner == nil || runner.client == nil {
		return
	}
	target := ShadowABTarget()
	done, _ := store.ShadowABCount(at.store)
	if done >= target {
		return
	}
	if !shadowABInFlight.CompareAndSwap(false, true) {
		at.logWarnf("🔬 shadow A/B skipped: another shadow call is still in flight (never concurrent)")
		return
	}
	go func() {
		defer shadowABInFlight.Store(false)
		defer func() {
			if r := recover(); r != nil { // A10 — a measurement never takes the bot down
				at.logWarnf("🔬 shadow A/B panicked (measurement only, live path unaffected): %v", r)
			}
		}()
		mcp.ApplyThinking(runner.client, runner.mode, runner.effort)
		cap := runner.maxTokens
		req := &mcp.Request{
			Messages:  []mcp.Message{mcp.NewSystemMessage(runner.system), mcp.NewUserMessage(userPrompt)},
			MaxTokens: &cap,
		}
		start := time.Now()
		var raw string
		var err error
		if bc, ok := runner.client.(interface{ BaseClient() *mcp.Client }); ok {
			raw, err = bc.BaseClient().CallWithRequestStreamRetryDeadlines(req, nil, plannerStreamIdle(), plannerStreamTotal())
		} else {
			raw, err = runner.client.CallWithMessages(runner.system, userPrompt)
		}
		verdict := ShadowABVerdict{WallMs: time.Since(start).Milliseconds(), Err: err}
		if err == nil {
			verdict.Legal, verdict.Reasons = at.shadowVerdictFor(raw, maxLevels, scenarioCap, facts, machineLabels, htfLabels, requiredBias)
		} else {
			verdict.Reasons = []string{"shadow call failed: " + err.Error()}
		}
		verdict.Tokens = mcp.LastCompletionTokens(runner.client)
		verdict.ReasonChar = mcp.LastReasoningChars(runner.client)
		n, _ := store.IncShadowAB(at.store, 1)
		at.logInfof("%s", shadowABLine(n, target, session, tradeDate, verdict, liveTokens, liveWallMs))
		// Restore the LIVE wire on the shared client so the next planner or
		// executor call is never left on the shadow mode.
		pm, pe := planReasoningWire()
		mcp.ApplyThinking(runner.client, pm, pe)
	}()
}

// FactsSnapshotJSON (B-1) renders the facts a rejected attempt was validated
// against, for the store's facts column. Errors degrade to "" — a measurement
// aid must never break a write path.
func FactsSnapshotJSON(f kernel.PlanFacts) string {
	b, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(b)
}

// ShadowABBootLine is the boot ledger line (pure).
func ShadowABBootLine(enabled bool, target, done int) string {
	state := "OFF"
	if enabled {
		state = "ON"
	}
	return fmt.Sprintf("🔬 shadow A/B (root-fix part B): %s target_n=%d done=%d (SHADOW_AB_ENABLED, SHADOW_AB_N) — fast-mode plans validated offline, never written; promotion criterion: legal-rate ≥ max AND median wall ≤50%% of max at n≥%d",
		state, target, done, target)
}
