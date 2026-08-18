package kernel

import (
	"encoding/json"
	"fmt"
	"nofx/config"
	"nofx/discipline"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
	"nofx/telemetry"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// Pre-compiled regular expressions (performance optimization)
// ============================================================================

var (
	// Safe regex: precisely match ```json code blocks
	reJSONFence      = regexp.MustCompile(`(?is)` + "```json\\s*(\\[\\s*\\{.*?\\}\\s*\\])\\s*```")
	reJSONArray      = regexp.MustCompile(`(?is)\[\s*\{.*?\}\s*\]`)
	reArrayHead      = regexp.MustCompile(`^\[\s*\{`)
	reArrayOpenSpace = regexp.MustCompile(`^\[\s+\{`)
	reInvisibleRunes = regexp.MustCompile("[\u200B\u200C\u200D\uFEFF]")

	// XML tag extraction (supports any characters in reasoning chain)
	reReasoningTag = regexp.MustCompile(`(?s)<reasoning>(.*?)</reasoning>`)
	reDecisionTag  = regexp.MustCompile(`(?s)<decision>(.*?)</decision>`)
)

// ============================================================================
// Entry Functions - Main API
// ============================================================================

// GetFullDecision gets AI's complete trading decision (batch analysis of all coins and positions)
// Uses default strategy configuration - for production use GetFullDecisionWithStrategy with explicit config
// holdCycle (F10) builds the no-actionable-decision result carrying the gate
// reason, so the caller records execution_status="guardrail_skip" + reason instead
// of a silently-empty success. systemPrompt/userPrompt are passed through when a
// prompt was already built (e.g. a schema-parse hold), else empty.
func holdCycle(reason string) *FullDecision {
	return &FullDecision{SkipReason: reason}
}

func GetFullDecision(ctx *Context, mcpClient mcp.AIClient) (*FullDecision, error) {
	defaultConfig := store.GetDefaultStrategyConfig("en")
	engine := NewStrategyEngine(&defaultConfig)
	return GetFullDecisionWithStrategy(ctx, mcpClient, engine, "")
}

// GetFullDecisionWithStrategy uses StrategyEngine to get AI decision (unified prompt generation)
func GetFullDecisionWithStrategy(ctx *Context, mcpClient mcp.AIClient, engine *StrategyEngine, variant string) (*FullDecision, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	// Plan 2 Task 18: skip decision cycle when CME is closed (futures mode only).
	// Returns nil decision + nil error so callers treat it as a clean no-op cycle.
	if ShouldSkipDecisionCycle() {
		// Plan 4 Task 25 — gate instrumentation
		telemetry.RiskGateTrips.WithLabelValues("task18_cme_closed").Inc()
		telemetry.IncGateBlock(ctx.TraderID, "task18_cme_closed")
		return holdCycle("cme_closed"), nil
	}
	// Plan 2 Task 19: filter candidates near contract expiry (futures mode only).
	// Within 5 days of the 3rd-Friday quarterly expiry, liquidity collapses on
	// the front month and rolls to the next contract. Block new entries on any
	// such symbol by dropping it from CandidateCoins BEFORE prompt assembly.
	// (Existing positions on the expiring contract are NOT dropped — the AI
	// still needs to evaluate close/hold for them.)
	if config.Get().TradingMode == "futures" && len(ctx.CandidateCoins) > 0 {
		filtered := ctx.CandidateCoins[:0]
		now := time.Now()
		for _, c := range ctx.CandidateCoins {
			if blocked, days := ShouldBlockEntryForExpiry(c.Symbol, now); blocked {
				logger.Warnf("contract %s expires in %d days, blocking new entries", c.Symbol, days)
				continue
			}
			filtered = append(filtered, c)
		}
		ctx.CandidateCoins = filtered
		if len(ctx.CandidateCoins) == 0 && len(ctx.Positions) == 0 {
			logger.Info("Plan 2 T19: all candidates near expiry and no open positions — skipping cycle")
			// Plan 4 Task 25 — gate instrumentation
			telemetry.RiskGateTrips.WithLabelValues("task19_contract_roll").Inc()
			telemetry.IncGateBlock(ctx.TraderID, "task19_contract_roll")
			return holdCycle("contract_roll"), nil
		}
	}
	// ============================================================================
	// Plan 3 Task 21 — Daily loss limit + kill switch (owned by T21)
	// T22 must add its block AFTER this section.
	// ============================================================================
	// Plan 3 Task 21 — risk gate: enforce hard daily-loss + concurrent-trades
	// + notional caps BEFORE expensive prompt assembly. We only check the
	// daily-loss + concurrent-position gates here (pre-prompt) because the
	// per-decision notional requires a parsed Decision which doesn't exist
	// yet. Notional + per-order contract gates are evaluated at execution
	// time via RiskLimits.CheckPreTrade(...) from the trader loop.
	//
	// Boundary: PnL strictly less than -MaxDailyLossUSD trips; equality is
	// permissive (matches plan source `< -r.MaxDailyLossUSD`).
	if ctx != nil {
		MaybeResetDaily(time.Now())
		limits := LoadRiskLimitsFromConfig()
		// Pre-prompt gate enforces ONLY daily-loss + concurrent-position (per the
		// comment above). The NOTIONAL cap is intentionally NOT applied here: it is
		// a futures-unaware crypto cap (RiskMaxNotionalUSD, default $50k) that any
		// single open MNQ contract (~$61k notional) exceeds, which previously
		// tripped EVERY cycle while a futures position was open and skipped it
		// before BuildUserPrompt/the AI — so the AI could never see, manage, or
		// close its own open position. Pass 0 for both requested + existing notional
		// so this gate never trips on notional. Notional IS still enforced at
		// EXECUTION time, futures-aware (engine_position.go: max position notional =
		// equity × futuresMaxNotionalLeverage).
		// DAILY-LOSS is no longer enforced here on session-cumulative TotalPnL —
		// Strategy Studio P1 EVOLVES it to TRUE daily-realized P&L below (pass 0
		// for pnl so CheckPreTrade enforces ONLY the concurrent-position cap).
		// D4 (audit F3): the concurrent-position cap is the PER-STRATEGY
		// max_positions; env RISK_MAX_CONCURRENT_TRADES is only the fallback.
		// Previously this gate always used the env value (default 2), so a
		// configured max_positions of 3 was unreachable (effective cap ≠ UI).
		perStrategyMaxPositions := 0
		if engine != nil {
			perStrategyMaxPositions = engine.GetRiskControlConfig().MaxPositions
		}
		concurrentCap, concurrentCapSource := ResolveConcurrentCap(perStrategyMaxPositions, limits.MaxConcurrentTrades)
		limits.MaxConcurrentTrades = concurrentCap
		if err := limits.CheckPreTrade(0, len(ctx.Positions), 0, 0); err != nil {
			logger.Warnf("⚠️ concurrent-position gate tripped: %v [cap source: %s] — skipping decision cycle (HOLD)", err, concurrentCapSource)
			// Plan 4 Task 25 — gate instrumentation
			telemetry.RiskGateTrips.WithLabelValues("task21_risk_limit").Inc()
			telemetry.IncGateBlock(ctx.TraderID, "task21_concurrent_cap")
			return holdCycle("concurrent_cap"), nil
		}

		// Strategy Studio P1 — per-strategy DAILY guardrails on TRUE daily-realized
		// P&L over the CME session-day (daily loss/profit + max-daily-trades). This
		// EVOLVES the env daily-loss gate (no duplicate): per-strategy value
		// overrides; env RISK_MAX_DAILY_LOSS_USD is the daily-loss fallback; the
		// master + per-guardrail toggles govern enforcement. Controls ONLY the
		// prop-firm guardrails — NEVER the SIM-only / live-account block (which is
		// enforced untoggleably in the broker layer).
		if engine != nil {
			rc := engine.GetRiskControlConfig()
			g := DailyGuardrails{
				MasterEnabled:    boolOrDefault(rc.GuardrailsEnabled, true),
				DailyRealizedPnL: ctx.DailyRealizedPnL,
				TradesToday:      ctx.TradesToday,

				DailyLossEnabled:  boolOrDefault(rc.DailyLossEnabled, true), // preserve the live daily-loss gate
				DailyLossLimitUSD: firstPositive(rc.DailyLossLimitUSD, limits.MaxDailyLossUSD),

				DailyProfitEnabled:   boolOrDefault(rc.DailyProfitEnabled, false),
				DailyProfitTargetUSD: rc.DailyProfitTargetUSD,

				MaxDailyTradesEnabled: boolOrDefault(rc.MaxDailyTradesEnabled, false),
				MaxDailyTrades:        rc.MaxDailyTrades,
			}
			if !g.MasterEnabled {
				logger.Warnf("⚠️ Strategy Studio: risk guardrails master OFF — daily loss/profit/trade limits + blackout NOT enforced this cycle (futures SIZE caps — notional×N ceiling + per-order contract clamp — REMAIN enforced; master-independent venue safety, hardening D3)")
				// P0-cleanup (2026-08-19) — SOFT-ALERT: show what the cage
				// WOULD have caught without ever blocking.
				for _, hit := range g.CheckSoft() {
					logger.Warnf("🔍 guardrail WOULD have tripped (master OFF, not enforced): %s", hit)
					telemetry.RecordError(ctx.TraderID, "guardrail_would_trip", hit, telemetry.CostNone)
					if SoftGuardrailFunc != nil {
						SoftGuardrailFunc(ctx.TraderID, hit)
					}
				}
			} else if _, gErr := g.Check(); gErr != nil {
				logger.Warnf("⚠️ Strategy Studio daily guardrail tripped: %v — skipping decision cycle (HOLD)", gErr)
				telemetry.RiskGateTrips.WithLabelValues("strategy_studio_daily").Inc()
				telemetry.IncGateBlock(ctx.TraderID, "strategy_studio_daily")
				return holdCycle("daily_guardrail"), nil
			} else if boolOrDefault(rc.BlackoutEnabled, false) && InBlackoutWindow(time.Now(), rc.BlackoutStartCT, rc.BlackoutEndCT) {
				// Chunk 4 — time/news blackout: go passive during a configured daily
				// [start,end] CT window (master + toggle governed). NT8-side SL/TP
				// still protect open positions; the bot just makes no new decisions.
				logger.Warnf("⚠️ Strategy Studio blackout window active (%s–%s CT) — skipping decision cycle (HOLD)", rc.BlackoutStartCT, rc.BlackoutEndCT)
				telemetry.RiskGateTrips.WithLabelValues("strategy_studio_blackout").Inc()
				telemetry.IncGateBlock(ctx.TraderID, "strategy_studio_blackout")
				return holdCycle("blackout_window"), nil
			} else if boolOrDefault(rc.ConsistencyEnabled, false) && ConsistencyBreached(ctx.DailyRealizedPnL, ctx.TotalRealizedPnL, rc.ConsistencyMaxDayPct) {
				// Chunk 5 — consistency rule: today's realized profit is too large a
				// share of total → go passive so this session-day does not exceed the
				// configured % of total realized profit.
				logger.Warnf("⚠️ Strategy Studio consistency rule: today's realized profit %.2f ≥ %.0f%% of total %.2f — skipping decision cycle (HOLD)", ctx.DailyRealizedPnL, rc.ConsistencyMaxDayPct, ctx.TotalRealizedPnL)
				telemetry.RiskGateTrips.WithLabelValues("strategy_studio_consistency").Inc()
				telemetry.IncGateBlock(ctx.TraderID, "strategy_studio_consistency")
				return holdCycle("consistency_rule"), nil
			}
		}
	}
	// ============================================================================
	// Plan 3 Task 22 — stale-data + drift detection: REMOVED AS DEAD CODE (W16/R7)
	// ============================================================================
	// The T22 block used to sit HERE, guarded by `len(ctx.MarketDataMap) > 0`.
	// That guard was false on 100% of production evaluations, so the gate never
	// once ran:
	//
	//   - the map is created and filled by fetchMarketDataWithStrategy, called
	//     ~70 lines BELOW this point in this same function;
	//   - a fresh Context is built every cycle (trader/auto_trader_loop.go), so
	//     nothing carries over from the previous cycle;
	//   - grep for MarketDataMap under trader/ returns zero hits — the loop has
	//     never pre-populated it, which is the case the old comment hoped for.
	//
	// Keeping unreachable safety code is worse than not having it: it reads like
	// a live gate in review and in audits, and nothing fails when it rots.
	//
	// WHAT ACTUALLY PROTECTS ENTRIES (and does run): applyStaleDataBlock (B4) at
	// the bottom of this function, AFTER the fetch — it neutralizes open_long /
	// open_short to `wait` when the freshest 1m/5m bar is older than 2× its
	// interval, and files a `stale_data` gate block. See kernel/stale_data.go.
	//
	// WHAT IS NOT REPLACED: suspicious-DRIFT detection (a >5% close-to-close move
	// inside 60s). It never ran either, and re-landing it belongs after the fetch
	// with timeframe-aware thresholds — market.DefaultFreshnessConfig's flat
	// 90s/5m limits would spuriously hold cycles on higher timeframes, which is
	// why this must not simply be moved. Tracked in the W16 report.

	if engine == nil {
		defaultConfig := store.GetDefaultStrategyConfig("en")
		engine = NewStrategyEngine(&defaultConfig)
	}

	// Clamp strategy limits to prevent token overflow
	engineConfig := engine.GetConfig()
	engineConfig.ClampLimits()

	// Token estimation check — block if exceeding the specific model's context limit
	estimate := engineConfig.EstimateTokens()

	// Determine context limit for the specific model being used
	contextLimit := 131072 // safe default (strictest common limit)
	var providerName string
	if embedder, ok := mcpClient.(mcp.ClientEmbedder); ok {
		base := embedder.BaseClient()
		providerName = base.Provider
		contextLimit = store.GetContextLimitForClient(base.Provider, base.Model)
	}

	if estimate.Total > contextLimit {
		logger.Errorf("🚫 Token estimate %d exceeds %s context limit %d — blocking analysis",
			estimate.Total, providerName, contextLimit)
		return nil, fmt.Errorf("estimated %d tokens exceeds model context limit of %d; reduce coins, timeframes, or K-line count",
			estimate.Total, contextLimit)
	}
	if estimate.Total*100/contextLimit >= 80 {
		logger.Infof("⚠️  Token estimate %d — approaching %s context limit %d",
			estimate.Total, providerName, contextLimit)
	}

	// 1. Fetch market data using strategy config
	if len(ctx.MarketDataMap) == 0 {
		if err := fetchMarketDataWithStrategy(ctx, engine); err != nil {
			return nil, fmt.Errorf("failed to fetch market data: %w", err)
		}
	}

	// Ensure OITopDataMap is initialized
	if ctx.OITopDataMap == nil {
		ctx.OITopDataMap = make(map[string]*OITopData)
		oiPositions, err := engine.nofxosClient.GetOITopPositions()
		if err == nil {
			for _, pos := range oiPositions {
				ctx.OITopDataMap[pos.Symbol] = &OITopData{
					Rank:              pos.Rank,
					OIDeltaPercent:    pos.OIDeltaPercent,
					OIDeltaValue:      pos.OIDeltaValue,
					PriceDeltaPercent: pos.PriceDeltaPercent,
				}
			}
		}
	}

	// 2. Build System Prompt using strategy engine
	riskConfig := engine.GetRiskControlConfig()
	// Active symbol for the futures system prompt (Phase 3): the open position's
	// symbol, else the first candidate, else "MNQ". Ignored on the crypto path.
	activeSymbol := "MNQ"
	if len(ctx.Positions) > 0 && ctx.Positions[0].Symbol != "" {
		activeSymbol = ctx.Positions[0].Symbol
	} else if len(ctx.CandidateCoins) > 0 && ctx.CandidateCoins[0].Symbol != "" {
		activeSymbol = ctx.CandidateCoins[0].Symbol
	}
	// SVP (Part B3): when svp_enabled is ON and we're on the futures prompt,
	// compute the session volume profile from 1m bars (AISVPBarInterval/
	// AISVPBarCount = "1m"/2000) and thread ONE line into the system prompt. This
	// is the AI's OWN, fixed SVP input — INDEPENDENT of whatever timeframe the
	// dashboard chart is displaying (the chart's SVP follows its own selected
	// interval; see api/handler_svp.go). Keep this at 1m: changing it changes the
	// AI prompt. OFF / insufficient bars → SetSVPContext("") keeps it byte-identical.
	engine.SetSVPContext("")
	svpOn := false
	if cfg := engine.GetConfig(); cfg != nil {
		svpOn = cfg.Indicators.EnableSVP
	}
	// P5.4 — ONE PROMPT, ONE SNAPSHOT: SVP, KEY LEVELS and PLAN STATUS used to
	// call FuturesBarsProvider separately, so one prompt carried prices ~2pt /
	// ~2min apart and distances were computed off the older snapshot. Fetch the
	// 1m cache ONCE per cycle; every section derives from it at a single now.
	var snapshotBars []market.Kline
	snapshotNow := time.Now()
	// P0 timezone — ONE labelled clock per prompt, derived from the same
	// snapshot instant as every other section (one snapshot → one clock).
	engine.SetClockContext("## Clock\n" + ClockCTAndUTC(snapshotNow) + " — ALL times in this prompt are CT (America/Chicago), including every session/window bound. Never apply CT window numbers to a UTC clock.")
	if market.FuturesBarsProvider != nil {
		snapshotBars = market.FuturesBarsProvider(activeSymbol, AISVPBarInterval, AISVPBarCount)
	}
	if isFut, _ := futuresVariantMode(variant); isFut && svpOn {
		svpLine := ""
		if len(snapshotBars) > 0 {
			svpLine = FormatSVPLine(BuildSVPProfile(snapshotBars, snapshotNow))
		}
		engine.SetSVPContext(svpLine)
		if svpLine == "" {
			// B9 (audit F6): SVP toggle ON but no line this cycle — make the
			// otherwise-silent skip observable. Causes: no 1m bars provider, no
			// bars, or no closed session bars (pre-open / maintenance / feed gap).
			logger.Infof("📐 SVP ON but omitted this cycle for %s — no session profile available (no 1m bars / provider down / pre-open); prompt has no SVP line.", activeSymbol)
		}
	}

	// KEY LEVELS (day-plan map, P1.7): when day_plan is ENABLED and we're on the
	// futures prompt, assemble the detected+scored structural level map from the
	// same 1m bars and thread ONE block into the system prompt. Gated identically
	// to SVP → disabled (the default) / empty keeps the golden byte-identical.
	engine.SetKeyLevelsContext("")
	planOn := false
	maxLevels := DefaultMaxLevels
	proximityK := ActivationWindowK
	if cfg := engine.GetConfig(); cfg != nil && cfg.DayPlan != nil {
		planOn = cfg.DayPlan.PlanEnabled
		if cfg.DayPlan.MaxLevels > 0 {
			maxLevels = cfg.DayPlan.MaxLevels
		}
		// H1/H2 — the day-trade lock is THIS ENGINE's config (the deciding
		// trader's strategy), never a process-global provider that could close
		// over a different trader (P0-A).
		if cfg.DayPlan.ProximityFilterATR >= 0.5 && cfg.DayPlan.ProximityFilterATR <= 3.0 {
			proximityK = cfg.DayPlan.ProximityFilterATR
		}
	}
	if isFut, _ := futuresVariantMode(variant); isFut && planOn {
		klBlock := ""
		if len(snapshotBars) > 0 {
			// nPOC from the durable session-profile store (P1.3), when the
			// trader layer has wired the provider; nil → none.
			var extra []DetectedLevel
			if NakedPOCProvider != nil {
				extra = NakedPOCProvider(activeSymbol)
			}
			// H7 — the registry is the admin registry the DECIDING trader
			// resolves (per-trader provider; never another trader's).
			klBlock = BuildKeyLevelsBlock(ctx.TraderID, snapshotBars, ResolvedSessionRegistryFor(ctx.TraderID), activeSymbol, maxLevels, snapshotNow, proximityK, extra...)
		}
		engine.SetKeyLevelsContext(klBlock)
		if klBlock == "" {
			// B9-style: day_plan ON but no KEY LEVELS this cycle — observable skip.
			logger.Infof("🗺️ day-plan KEY LEVELS ON but omitted this cycle for %s — no levels available (no 1m bars / provider down / warming forward); prompt has no KEY LEVELS block.", activeSymbol)
		}
	}

	// P3.4 — EXECUTOR PLAN INJECTION: resolve the active plan and thread the
	// byte-stable PLAN BLOCK (cached prefix) + dynamic PLAN STATUS (tail from the
	// P0.4 evaluator). Gated on day_plan; no active plan → cleared (prompt
	// unchanged). When present, this triggers the RECON #4 reorder.
	// P0-A — the plan is resolved for the DECIDING trader (ctx.TraderID); the
	// provider is per-trader, so one trader's plan can never reach another.
	engine.SetPlanContext("", "")
	if isFut, _ := futuresVariantMode(variant); isFut && planOn {
		if plan := ActivePlanFor(ctx.TraderID, activeSymbol); plan != nil {
			// W15.B — resolve the acceptance rule for THE PLAN'S OWN SESSION, so a
			// per-session override reaches the executor prompt (it previously read
			// only the strategy-level field, making the override inert).
			rule := ""
			if cfg := engine.GetConfig(); cfg != nil && cfg.DayPlan != nil {
				rule = cfg.DayPlan.AcceptanceRuleFor(plan.Session)
			}
			block := RenderPlanBlock(plan.Doc, plan.Session)
			status := ""
			if len(snapshotBars) > 0 {
				_, price, dATR := AssembleScoredLevels(ctx.TraderID, snapshotBars, ResolvedSessionRegistryFor(ctx.TraderID), activeSymbol, maxLevels, snapshotNow, proximityK)
				status = RenderPlanStatus(ctx.TraderID, activeSymbol, plan.Doc, snapshotBars, price, dATR, rule, plan.ReplansLeft, snapshotNow.UnixMilli(), plan.BirthMs)
			}
			engine.SetPlanContext(block, status)
		}
	}
	// A5 (G5) — PROMPT-OWNERSHIP assertion: every account-scoped context field must
	// be owned by the DECIDING trader. A field tagged with another trader_id is
	// cross-trader contamination — skip the cycle rather than decide on foreign data.
	if field, badOwner, violated := assertPromptOwnership(ctx); violated {
		logger.Errorf("🚨 A5 prompt-ownership VIOLATION: context field %q is owned by %q ≠ deciding trader %q — SKIPPING cycle (cross-trader contamination).", field, badOwner, ctx.TraderID)
		telemetry.IncGateBlock(ctx.TraderID, "prompt_ownership")
		return holdCycle("prompt_ownership_violation"), nil
	}

	systemPrompt := engine.BuildSystemPrompt(ctx.Account.TotalEquity, variant, activeSymbol)

	// 3. Build User Prompt using strategy engine
	userPrompt := engine.BuildUserPrompt(ctx)

	// 4-5. Call AI + parse, with B2a schema-strict BOUNDED RETRY (callWithSchemaRetry):
	// a malformed/missing-field/rule-invalid response is retried up to maxParseRetries
	// with the parse error fed back; still bad → skip the cycle with a NAMED reason
	// (HOLD, never crash). schema_parse_failed is the named reason B6 will count.
	const maxParseRetries = 2
	parse := func(resp string) (*FullDecision, error) {
		return parseFullDecisionResponse(
			resp,
			ctx.Account.TotalEquity,
			riskConfig.BTCETHMaxLeverage,
			riskConfig.AltcoinMaxLeverage,
			riskConfig.BTCETHMaxPositionValueRatio,
			riskConfig.AltcoinMaxPositionValueRatio,
			riskConfig.MinRiskRewardRatio,
			riskConfig.MinConfidence,
			ResolveNotionalLeverage(riskConfig.MaxNotionalLeverage, futuresMaxNotionalLeverage),
			ctx, // F1: entry reference (current-price snapshot) + TraderID for rr_gate
		)
	}
	decision, aiResponse, aiCallDuration, parseErr, callErr := callWithSchemaRetry(
		mcpClient, systemPrompt, userPrompt, parse, maxParseRetries)
	if callErr != nil {
		return nil, fmt.Errorf("AI API call failed: %w", callErr)
	}
	if parseErr != nil {
		logger.Warnf("🚫 schema_parse_failed: AI response unparseable after %d attempts — skipping decision cycle (HOLD). Last error: %v",
			maxParseRetries+1, parseErr)
		// F10 — a real AI call happened; preserve the prompts/response so the record
		// is truthful (not a silently-empty success), and stamp the skip reason.
		return &FullDecision{SkipReason: "schema_parse_failed", SystemPrompt: systemPrompt, UserPrompt: userPrompt, RawResponse: aiResponse}, nil
	}

	if decision != nil {
		decision.Timestamp = time.Now()
		decision.SystemPrompt = systemPrompt
		decision.UserPrompt = userPrompt
		decision.AIRequestDurationMs = aiCallDuration.Milliseconds()
		decision.RawResponse = aiResponse
	}

	// B2b — price-sanity armor: neutralize an open decision whose stop/target/entry
	// are physically implausible, using the freshest market data (fail-open).
	applyPriceSanity(decision, ctx)
	// B4 — stale-data entry block: refuse a NEW entry when the freshest 1m/5m bar is
	// stale (feed frozen); exits / open-position management are never touched.
	applyStaleDataBlock(decision, ctx, time.Now().UnixMilli())

	// C2 — clock-drift guard: refuse a NEW entry when the local clock is skewed
	// >60s from the freshest feed timestamp (either direction); signals would be
	// mis-timed / NT8-rejected. Exits untouched.
	applyClockDriftBlock(decision, ctx, time.Now().UnixMilli())

	// B7 — re-entry cooldown: after a stop-loss exit, refuse a same-direction
	// re-entry on that symbol until the cooldown elapses OR price moved ≥1×ATR15
	// from the stop (whichever first). Per-strategy; 0 = OFF. Exits untouched.
	if engine != nil {
		applyReentryCooldown(decision, ctx, engine.GetRiskControlConfig().ReentryCooldownMinutes, time.Now().UnixMilli())
	}

	return decision, nil
}

// applyReentryCooldown (B7) neutralizes a same-direction re-entry to `wait` while
// the re-entry cooldown for (trader, symbol, side) is active (see discipline.
// ReentryBlocked). cooldownMinutes ≤ 0 → OFF. Missing market data → fail-open
// (leave the decision). Only open_long/open_short are affected.
func applyReentryCooldown(fd *FullDecision, ctx *Context, cooldownMinutes int, nowMs int64) {
	if fd == nil || ctx == nil || cooldownMinutes <= 0 {
		return
	}
	for i := range fd.Decisions {
		d := &fd.Decisions[i]
		side := ""
		switch d.Action {
		case "open_long":
			side = "long"
		case "open_short":
			side = "short"
		default:
			continue
		}
		md := ctx.MarketDataMap[d.Symbol]
		if md == nil || md.CurrentPrice <= 0 {
			continue // no data → fail-open
		}
		if remaining, reason, blocked := discipline.ReentryBlocked(ctx.TraderID, d.Symbol, side, cooldownMinutes, atr15From(md), md.CurrentPrice, nowMs); blocked {
			logger.Warnf("⛔ re-entry cooldown: %s %s → WAIT — %s (%ds left).", d.Symbol, d.Action, reason, remaining)
			d.RefusalReason = "re-entry cooldown: " + reason
			d.Action = "wait"
			telemetry.IncGateBlock(ctx.TraderID, "reentry_cooldown")
			telemetry.RecordError(ctx.TraderID, "entry_refused", "re-entry cooldown: "+reason, telemetry.CostTradeLost)
		}
	}
}

// callWithSchemaRetry (B2a) calls the AI + parses, retrying up to maxRetries with
// the parse error appended to the user prompt (schema-strict bounded retry). It
// returns the last decision, raw response, call duration, the final parseErr
// (nil = success) and callErr (a transport error, returned immediately). A caller
// that sees a non-nil parseErr should skip the cycle with a named reason.
func callWithSchemaRetry(mcpClient mcp.AIClient, systemPrompt, userPrompt string, parse func(string) (*FullDecision, error), maxRetries int) (decision *FullDecision, raw string, dur time.Duration, parseErr, callErr error) {
	// P5.5 — the last non-empty output survives even when a LATER retry errors
	// (e.g. the 180s cap): the owner's record keeps what the model actually
	// said, instead of an empty raw_response with no explanation.
	lastRaw := ""
	for attempt := 0; ; attempt++ {
		up := userPrompt
		if attempt > 0 {
			up = userPrompt + fmt.Sprintf(
				"\n\n[RETRY %d/%d] Your previous reply could not be parsed: %v. Reply with ONLY the required decision JSON envelope — every required field present, correct types, prices as concrete numbers.",
				attempt, maxRetries, parseErr)
		}
		start := time.Now()
		resp, cErr := mcpClient.CallWithMessages(systemPrompt, up)
		dur = time.Since(start)
		if cErr != nil {
			return nil, lastRaw, dur, nil, cErr
		}
		if strings.TrimSpace(resp) != "" {
			lastRaw = resp
		}
		raw = resp
		decision, parseErr = parse(resp)
		if parseErr == nil || attempt >= maxRetries {
			return decision, lastRaw, dur, parseErr, nil
		}
		logger.Warnf("⚠️ AI response parse failed (attempt %d/%d) — retrying with the error fed back: %v",
			attempt+1, maxRetries+1, parseErr)
	}
}

// applyPriceSanity (B2b) neutralizes to `wait` any open decision whose stop /
// target / entry are implausible per priceSanityViolation (the one authority),
// using ctx.MarketDataMap for the reference price + ATR. It NEVER rewrites the
// AI's prices — a violation becomes a loud ⛔ refusal, not a silent correction.
// Missing market data → fail-open (log + leave the decision) so the cycle never
// crashes. Composes with validateDecision's wrong-side authority.
func applyPriceSanity(fd *FullDecision, ctx *Context) {
	if fd == nil || ctx == nil {
		return
	}
	for i := range fd.Decisions {
		d := &fd.Decisions[i]
		if d.Action != "open_long" && d.Action != "open_short" {
			continue
		}
		md := ctx.MarketDataMap[d.Symbol]
		if md == nil || md.CurrentPrice <= 0 {
			logger.Infof("⚠️ price-sanity: no market data for %s — skipping sanity check (fail-open).", d.Symbol)
			continue
		}
		side := "long"
		if d.Action == "open_short" {
			side = "short"
		}
		entryRef := md.CurrentPrice // market entry; explicit-entry paths pass their own ref
		atr15 := atr15From(md)
		if reason, bad := priceSanityViolation(side, entryRef, d.StopLoss, d.TakeProfit, atr15, md.CurrentPrice); bad {
			logger.Warnf("⛔ price-sanity REJECT: %s %s → neutralized to WAIT — %s [AI stop=%.4f target=%.4f, price=%.4f, ATR15=%.4f].",
				d.Symbol, d.Action, reason, d.StopLoss, d.TakeProfit, md.CurrentPrice, atr15)
			d.RefusalReason = "price-sanity: " + reason
			d.Action = "wait"
			telemetry.IncGateBlock(ctx.TraderID, "price_sanity")
			telemetry.RecordError(ctx.TraderID, "entry_refused", "price-sanity: "+reason, telemetry.CostTradeLost)
		}
	}
}

// atr15From returns the 14-period ATR on the 15-minute timeframe (the design's
// "atr15") from a symbol's multi-TF data, with a deterministic fallback to nearby
// timeframes if 15m wasn't fetched. 0 → no ATR available (the caller's ATR-distance
// check is then skipped, fail-open).
func atr15From(md *market.Data) float64 {
	if md == nil {
		return 0
	}
	for _, tf := range []string{"15m", "5m", "3m", "1h", "4h", "1d"} {
		if td := md.TimeframeData[tf]; td != nil && td.ATR14 > 0 {
			return td.ATR14
		}
	}
	return 0
}

// ============================================================================
// Market Data Fetching
// ============================================================================

// fetchMarketDataWithStrategy fetches market data using strategy config (multiple timeframes)
func fetchMarketDataWithStrategy(ctx *Context, engine *StrategyEngine) error {
	config := engine.GetConfig()
	ctx.MarketDataMap = make(map[string]*market.Data)

	timeframes := config.Indicators.Klines.SelectedTimeframes
	primaryTimeframe := config.Indicators.Klines.PrimaryTimeframe
	klineCount := config.Indicators.Klines.PrimaryCount

	// Compatible with old configuration
	if len(timeframes) == 0 {
		if primaryTimeframe != "" {
			timeframes = append(timeframes, primaryTimeframe)
		} else {
			timeframes = append(timeframes, "3m")
		}
		if config.Indicators.Klines.LongerTimeframe != "" {
			timeframes = append(timeframes, config.Indicators.Klines.LongerTimeframe)
		}
	}
	if primaryTimeframe == "" {
		primaryTimeframe = timeframes[0]
	}
	if klineCount <= 0 {
		klineCount = 30
	}

	logger.Infof("📊 Strategy timeframes: %v, Primary: %s, Kline count: %d", timeframes, primaryTimeframe, klineCount)

	// Strategy-configured indicator periods → the market layer computes the user's
	// periods (e.g. EMA 9/21/200, BOLL 10/12) instead of the legacy fixed defaults.
	indPeriods := market.IndicatorPeriods{
		EMA:  config.Indicators.EMAPeriods,
		RSI:  config.Indicators.RSIPeriods,
		ATR:  config.Indicators.ATRPeriods,
		BOLL: config.Indicators.BOLLPeriods,
	}

	// 1. First fetch data for position coins (must fetch)
	for _, pos := range ctx.Positions {
		data, err := market.GetWithTimeframes(pos.Symbol, timeframes, primaryTimeframe, klineCount, indPeriods)
		if err != nil {
			logger.Infof("⚠️  Failed to fetch market data for position %s: %v", pos.Symbol, err)
			continue
		}
		ctx.MarketDataMap[pos.Symbol] = data
	}

	// 2. Fetch data for all candidate coins
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[pos.Symbol] = true
	}

	const minOIThresholdMillions = 15.0 // 15M USD minimum open interest value

	for _, coin := range ctx.CandidateCoins {
		if _, exists := ctx.MarketDataMap[coin.Symbol]; exists {
			continue
		}

		data, err := market.GetWithTimeframes(coin.Symbol, timeframes, primaryTimeframe, klineCount, indPeriods)
		if err != nil {
			logger.Infof("⚠️  Failed to fetch market data for %s: %v", coin.Symbol, err)
			continue
		}

		// Liquidity filter (skip for xyz dex assets - they don't have OI data from Binance).
		// CME futures (NT8 path) likewise have no crypto open-interest feed, so the
		// OI gate would wrongly drop them (OI=0 < threshold) — exempt them too.
		isExistingPosition := positionSymbols[coin.Symbol]
		isXyzAsset := market.IsXyzDexAsset(coin.Symbol)
		isFutures := market.IsCMEFuturesSymbol(coin.Symbol)
		if !isExistingPosition && !isXyzAsset && !isFutures && data.OpenInterest != nil && data.CurrentPrice > 0 {
			oiValue := data.OpenInterest.Latest * data.CurrentPrice
			oiValueInMillions := oiValue / 1_000_000
			if oiValueInMillions < minOIThresholdMillions {
				logger.Infof("⚠️  %s OI value too low (%.2fM USD < %.1fM), skipping coin",
					coin.Symbol, oiValueInMillions, minOIThresholdMillions)
				continue
			}
		}

		ctx.MarketDataMap[coin.Symbol] = data
	}

	logger.Infof("📊 Successfully fetched multi-timeframe market data for %d coins", len(ctx.MarketDataMap))
	return nil
}

// ============================================================================
// AI Response Parsing
// ============================================================================

func parseFullDecisionResponse(aiResponse string, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, minRiskReward float64, minConfidence int, maxNotionalLev float64, ctx *Context) (*FullDecision, error) {
	cotTrace := extractCoTTrace(aiResponse)

	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: []Decision{},
		}, fmt.Errorf("failed to extract decisions: %w", err)
	}

	// Backfill BEFORE validation so a REJECTED decision keeps its "why" too — the
	// refusals panel is exactly where the owner asks why something was refused.
	backfillReasoningFromCoT(decisions, cotTrace)

	if err := validateDecisions(decisions, accountEquity, btcEthLeverage, altcoinLeverage, btcEthPosRatio, altcoinPosRatio, minRiskReward, minConfidence, maxNotionalLev, ctx); err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: decisions,
		}, fmt.Errorf("decision validation failed: %w", err)
	}

	return &FullDecision{
		CoTTrace:  cotTrace,
		Decisions: decisions,
	}, nil
}

// backfillReasoningFromCoT gives every decision the "why" the model actually
// wrote (ITEM 6, 2026-08-17).
//
// The model puts its analysis in the <reasoning> block and — for `wait`
// especially — leaves the decision JSON's own `reasoning` field empty. That
// field is what the dashboard renders, so the screen could not answer "why did
// it wait?" even though the answer was sitting in the SAME response, correctly
// extracted into cot_trace. Live evidence: decision_records rows whose
// raw_response opens "<reasoning> Current price 30195.75 is sitting almost
// exactly on the session POC…" every one of which stored reasoning "".
//
// Nothing is invented: the text is copied from the same response, only when the
// model left the field empty. When one response carries several decisions they
// share the trace, which is accurate — it is the reasoning for that cycle.
func backfillReasoningFromCoT(decisions []Decision, cotTrace string) {
	trace := strings.TrimSpace(cotTrace)
	if trace == "" {
		return
	}
	for i := range decisions {
		if strings.TrimSpace(decisions[i].Reasoning) == "" {
			decisions[i].Reasoning = trace
		}
	}
}

func extractCoTTrace(response string) string {
	if match := reReasoningTag.FindStringSubmatch(response); match != nil && len(match) > 1 {
		logger.Infof("✓ Extracted reasoning chain using <reasoning> tag")
		return strings.TrimSpace(match[1])
	}

	if decisionIdx := strings.Index(response, "<decision>"); decisionIdx > 0 {
		logger.Infof("✓ Extracted content before <decision> tag as reasoning chain")
		return strings.TrimSpace(response[:decisionIdx])
	}

	jsonStart := strings.Index(response, "[")
	if jsonStart > 0 {
		logger.Infof("⚠️  Extracted reasoning chain using old format ([ character separator)")
		return strings.TrimSpace(response[:jsonStart])
	}

	return strings.TrimSpace(response)
}

func extractDecisions(response string) ([]Decision, error) {
	s := removeInvisibleRunes(response)
	s = strings.TrimSpace(s)
	s = fixMissingQuotes(s)

	var jsonPart string
	if match := reDecisionTag.FindStringSubmatch(s); match != nil && len(match) > 1 {
		jsonPart = strings.TrimSpace(match[1])
		logger.Infof("✓ Extracted JSON using <decision> tag")
	} else {
		jsonPart = s
		logger.Infof("⚠️  <decision> tag not found, searching JSON in full text")
	}

	jsonPart = fixMissingQuotes(jsonPart)

	if m := reJSONFence.FindStringSubmatch(jsonPart); m != nil && len(m) > 1 {
		jsonContent := strings.TrimSpace(m[1])
		jsonContent = compactArrayOpen(jsonContent)
		jsonContent = fixMissingQuotes(jsonContent)
		if err := validateJSONFormat(jsonContent); err != nil {
			return nil, fmt.Errorf("JSON format validation failed: %w\nJSON content: %s\nFull response:\n%s", err, jsonContent, response)
		}
		var decisions []Decision
		if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
			return nil, fmt.Errorf("JSON parsing failed: %w\nJSON content: %s", err, jsonContent)
		}
		return decisions, nil
	}

	jsonContent := strings.TrimSpace(reJSONArray.FindString(jsonPart))
	if jsonContent == "" {
		// Robust recovery before declaring a miss: the model may have emitted a
		// SINGLE decision object (not wrapped in an array) anywhere in the prose.
		// Recover it with the same brace-balanced technique ParsePlanDoc uses,
		// then wrap it into the array shape the decision schema expects. A prose
		// sentence can never false-positive here: the object must be valid JSON
		// AND carry a non-empty `action`.
		if obj := extractJSONObject(jsonPart); obj != "" {
			var d Decision
			if err := json.Unmarshal([]byte(obj), &d); err == nil && d.Action != "" {
				logger.Infof("✓ Recovered a single decision object embedded in prose")
				return []Decision{d}, nil
			}
		}
		// No JSON anywhere → a real parse miss, not a wait. Return an error so
		// callWithSchemaRetry RETRIES with "reply JSON only" instead of silently
		// swallowing the decision into a wait. Before this, a fully-reasoned setup
		// was lost to a format failure with no retry and no record of the loss.
		return nil, fmt.Errorf("no decision JSON found in AI response (prose-only)")
	}

	jsonContent = compactArrayOpen(jsonContent)
	jsonContent = fixMissingQuotes(jsonContent)

	if err := validateJSONFormat(jsonContent); err != nil {
		return nil, fmt.Errorf("JSON format validation failed: %w\nJSON content: %s\nFull response:\n%s", err, jsonContent, response)
	}

	var decisions []Decision
	if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w\nJSON content: %s", err, jsonContent)
	}

	return decisions, nil
}

func fixMissingQuotes(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "\u201c", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\u201d", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\u2018", "'")
	jsonStr = strings.ReplaceAll(jsonStr, "\u2019", "'")

	jsonStr = strings.ReplaceAll(jsonStr, "［", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "］", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "｛", "{")
	jsonStr = strings.ReplaceAll(jsonStr, "｝", "}")
	jsonStr = strings.ReplaceAll(jsonStr, "：", ":")
	jsonStr = strings.ReplaceAll(jsonStr, "，", ",")

	jsonStr = strings.ReplaceAll(jsonStr, "【", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "】", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "〔", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "〕", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "、", ",")

	jsonStr = strings.ReplaceAll(jsonStr, "　", " ")

	return jsonStr
}

func validateJSONFormat(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)

	if !reArrayHead.MatchString(trimmed) {
		if strings.HasPrefix(trimmed, "[") && !strings.Contains(trimmed[:min(20, len(trimmed))], "{") {
			return fmt.Errorf("not a valid decision array (must contain objects {}), actual content: %s", trimmed[:min(50, len(trimmed))])
		}
		return fmt.Errorf("JSON must start with [{ (whitespace allowed), actual: %s", trimmed[:min(20, len(trimmed))])
	}

	if strings.Contains(jsonStr, "~") {
		return fmt.Errorf("JSON cannot contain range symbol ~, all numbers must be precise single values")
	}

	for i := 0; i < len(jsonStr)-4; i++ {
		if jsonStr[i] >= '0' && jsonStr[i] <= '9' &&
			jsonStr[i+1] == ',' &&
			jsonStr[i+2] >= '0' && jsonStr[i+2] <= '9' &&
			jsonStr[i+3] >= '0' && jsonStr[i+3] <= '9' &&
			jsonStr[i+4] >= '0' && jsonStr[i+4] <= '9' {
			return fmt.Errorf("JSON numbers cannot contain thousand separator comma, found: %s", jsonStr[i:min(i+10, len(jsonStr))])
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func removeInvisibleRunes(s string) string {
	return reInvisibleRunes.ReplaceAllString(s, "")
}

func compactArrayOpen(s string) string {
	return reArrayOpenSpace.ReplaceAllString(strings.TrimSpace(s), "[{")
}

// ============================================================================
// Plan 3 Task 21 — Force-flat helper (owned by T21)
// ============================================================================

// ForceFlatSignaler is the minimal interface a broker bridge must satisfy so
// that kernel can ForceFlat without importing the concrete broker package
// (e.g. provider/ninjatrader). Keeps risk_limits + engine decoupled.
//
// The expected concrete implementation today is *ninjatrader.CSVWriter, which
// satisfies WriteSignal(SignalRow) error. Anonymous struct satisfaction works
// only if SignalRow is in scope, so callers pass a thin adapter — see the
// (future) POST /api/risk/force-flat endpoint for the adapter shape.
//
// API note: this helper is invoked by an HTTP endpoint (POST /api/risk/force-flat),
// NOT by the engine itself. The engine only HOLDs on risk trips; closing
// positions is an operator-initiated action surfaced via a red "EMERGENCY FLAT"
// button on TraderDashboardPage. The endpoint + button are deferred to a
// follow-up PR.
type ForceFlatSignaler interface {
	// ForceFlat emits whatever broker-specific signal closes all positions.
	// For ninjatrader v1: cancel pending CSV signals (NT does not yet expose
	// a "close all" command via the bridge — manual close on the chart).
	ForceFlat(traderID string) error
}

// MaybeForceFlat invokes the broker's ForceFlat path and logs the outcome.
// Returns nil for the no-op nil-signaler case so the API endpoint can be
// safely called when no broker is wired up (e.g. paper test).
func MaybeForceFlat(traderID string, signaler ForceFlatSignaler) error {
	if signaler == nil {
		logger.Warnf("Plan 3 T21 MaybeForceFlat: no signaler wired for trader %s — no-op", traderID)
		return nil
	}
	logger.Warnf("🔴 Plan 3 T21 FORCE-FLAT invoked for trader %s", traderID)
	if err := signaler.ForceFlat(traderID); err != nil {
		return fmt.Errorf("force-flat trader %s: %w", traderID, err)
	}
	// After a force-flat, reset the daily PnL window so the operator can
	// resume after they have addressed whatever tripped the kill switch.
	ResetDailyPnL()
	return nil
}
