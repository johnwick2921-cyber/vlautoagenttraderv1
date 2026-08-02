package kernel

import (
	"encoding/json"
	"fmt"
	"nofx/config"
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
		return nil, nil
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
			return nil, nil
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
		if err := limits.CheckPreTrade(0, len(ctx.Positions), 0, 0); err != nil {
			logger.Warnf("⚠️ Plan 3 T21 risk gate tripped: %v — skipping decision cycle (HOLD)", err)
			// Plan 4 Task 25 — gate instrumentation
			telemetry.RiskGateTrips.WithLabelValues("task21_risk_limit").Inc()
			return nil, nil
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
				logger.Warnf("⚠️ Strategy Studio: risk guardrails DISABLED by master switch — daily limits + blackout NOT enforced this cycle")
			} else if _, gErr := g.Check(); gErr != nil {
				logger.Warnf("⚠️ Strategy Studio daily guardrail tripped: %v — skipping decision cycle (HOLD)", gErr)
				telemetry.RiskGateTrips.WithLabelValues("strategy_studio_daily").Inc()
				return nil, nil
			} else if boolOrDefault(rc.BlackoutEnabled, false) && InBlackoutWindow(time.Now(), rc.BlackoutStartCT, rc.BlackoutEndCT) {
				// Chunk 4 — time/news blackout: go passive during a configured daily
				// [start,end] CT window (master + toggle governed). NT8-side SL/TP
				// still protect open positions; the bot just makes no new decisions.
				logger.Warnf("⚠️ Strategy Studio blackout window active (%s–%s CT) — skipping decision cycle (HOLD)", rc.BlackoutStartCT, rc.BlackoutEndCT)
				telemetry.RiskGateTrips.WithLabelValues("strategy_studio_blackout").Inc()
				return nil, nil
			} else if boolOrDefault(rc.ConsistencyEnabled, false) && ConsistencyBreached(ctx.DailyRealizedPnL, ctx.TotalRealizedPnL, rc.ConsistencyMaxDayPct) {
				// Chunk 5 — consistency rule: today's realized profit is too large a
				// share of total → go passive so this session-day does not exceed the
				// configured % of total realized profit.
				logger.Warnf("⚠️ Strategy Studio consistency rule: today's realized profit %.2f ≥ %.0f%% of total %.2f — skipping decision cycle (HOLD)", ctx.DailyRealizedPnL, rc.ConsistencyMaxDayPct, ctx.TotalRealizedPnL)
				telemetry.RiskGateTrips.WithLabelValues("strategy_studio_consistency").Inc()
				return nil, nil
			}
		}
	}
	// ============================================================================
	// Plan 3 Task 22 — Stale-data + drift detection (owned by T22)
	// ============================================================================
	// Verify the latest OHLCV bar is fresh and free of suspicious limit-move-
	// style drift before feeding it to the AI. RTH gets a tight 90s tolerance;
	// ETH gets 5min. A >5% close-to-close move inside 60s trips the drift gate.
	// On any HOLD here we return (nil, nil) so the loop treats it as a clean
	// skip — same convention as T18/T19/T21.
	//
	// Skipped silently when MarketDataMap is empty (data not yet fetched in
	// this cycle); the downstream fetch will populate it and the next cycle
	// will gate normally. We deliberately check pre-fetch in case the loop
	// pre-populated the map from a hotter source.
	if ctx != nil && len(ctx.MarketDataMap) > 0 {
		cfg := market.DefaultFreshnessConfig()
		now := time.Now()
		for symbol, data := range ctx.MarketDataMap {
			if data == nil || data.TimeframeData == nil {
				continue
			}
			for tf, tfData := range data.TimeframeData {
				if tfData == nil || len(tfData.Klines) < 2 {
					continue
				}
				bars := tfData.Klines
				latest := bars[len(bars)-1]
				prev := bars[len(bars)-2]
				health := market.CheckDataHealth(
					latest.Time, latest.Close,
					prev.Time, prev.Close,
					now, cfg,
				)
				switch health {
				case market.HealthStale:
					age := now.Sub(time.UnixMilli(latest.Time))
					logger.Warnf("⚠️ Plan 3 T22: stale data for %s [%s] (last bar %v old) — skipping cycle", symbol, tf, age)
					// Plan 4 Task 25 — gate instrumentation
					telemetry.RiskGateTrips.WithLabelValues("task22_drift").Inc()
					return nil, nil
				case market.HealthSuspiciousDrift:
					logger.Warnf("⚠️ Plan 3 T22: suspicious drift for %s [%s] (prev=%.4f latest=%.4f) — skipping cycle", symbol, tf, prev.Close, latest.Close)
					// Plan 4 Task 25 — gate instrumentation
					telemetry.RiskGateTrips.WithLabelValues("task22_drift").Inc()
					return nil, nil
				}
			}
		}
	}

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
	// compute the session volume profile from the live 1m bars (same bar source
	// the chart uses — ONE source of truth) and thread ONE line into the system
	// prompt. OFF / insufficient bars → SetSVPContext("") keeps it byte-identical.
	engine.SetSVPContext("")
	svpOn := false
	if cfg := engine.GetConfig(); cfg != nil {
		svpOn = cfg.Indicators.EnableSVP
	}
	if isFut, _ := futuresVariantMode(variant); isFut && svpOn && market.FuturesBarsProvider != nil {
		if bars1m := market.FuturesBarsProvider(activeSymbol, "1m", 2000); len(bars1m) > 0 {
			engine.SetSVPContext(FormatSVPLine(BuildSVPProfile(bars1m, time.Now())))
		}
	}
	systemPrompt := engine.BuildSystemPrompt(ctx.Account.TotalEquity, variant, activeSymbol)

	// 3. Build User Prompt using strategy engine
	userPrompt := engine.BuildUserPrompt(ctx)

	// 4. Call AI API
	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	// 5. Parse AI response
	decision, err := parseFullDecisionResponse(
		aiResponse,
		ctx.Account.TotalEquity,
		riskConfig.BTCETHMaxLeverage,
		riskConfig.AltcoinMaxLeverage,
		riskConfig.BTCETHMaxPositionValueRatio,
		riskConfig.AltcoinMaxPositionValueRatio,
		riskConfig.MinRiskRewardRatio,
		riskConfig.MinConfidence,
		ResolveNotionalLeverage(riskConfig.GuardrailsEnabled, riskConfig.NotionalCapEnabled, riskConfig.MaxNotionalLeverage, futuresMaxNotionalLeverage),
	)

	if decision != nil {
		decision.Timestamp = time.Now()
		decision.SystemPrompt = systemPrompt
		decision.UserPrompt = userPrompt
		decision.AIRequestDurationMs = aiCallDuration.Milliseconds()
		decision.RawResponse = aiResponse
	}

	if err != nil {
		return decision, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return decision, nil
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

func parseFullDecisionResponse(aiResponse string, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, minRiskReward float64, minConfidence int, maxNotionalLev float64) (*FullDecision, error) {
	cotTrace := extractCoTTrace(aiResponse)

	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: []Decision{},
		}, fmt.Errorf("failed to extract decisions: %w", err)
	}

	if err := validateDecisions(decisions, accountEquity, btcEthLeverage, altcoinLeverage, btcEthPosRatio, altcoinPosRatio, minRiskReward, minConfidence, maxNotionalLev); err != nil {
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
		logger.Infof("⚠️  [SafeFallback] AI didn't output JSON decision, entering safe wait mode")

		cotSummary := jsonPart
		if len(cotSummary) > 240 {
			cotSummary = cotSummary[:240] + "..."
		}

		fallbackDecision := Decision{
			Symbol:    "ALL",
			Action:    "wait",
			Reasoning: fmt.Sprintf("Model didn't output structured JSON decision, entering safe wait; summary: %s", cotSummary),
		}

		return []Decision{fallbackDecision}, nil
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
