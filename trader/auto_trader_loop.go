package trader

import (
	"encoding/json"
	"fmt"
	"nofx/config"
	"nofx/kernel"
	"nofx/logger"
	"nofx/store"
	"nofx/telemetry"
	"nofx/wallet"
	"strings"
	"time"
)

// resolvePromptVariant picks the live AI prompt mode (Strategy Studio Phase 2).
// A non-empty per-strategy saved variant wins; otherwise the original venue
// rule applies — CME futures (NinjaTrader) → "futures", everything else →
// "balanced". This is the back-compat guarantee: an empty savedVariant resolves
// EXACTLY as the pre-Phase-2 code, so strategies with no saved variant are
// byte-identical. Prompt-layer only — it never touches a risk gate.
func resolvePromptVariant(exchange, savedVariant string) string {
	if v := strings.TrimSpace(savedVariant); v != "" {
		return v
	}
	if exchange == "ninjatrader" {
		return "futures"
	}
	return "balanced"
}

// runCycle runs one trading cycle (using AI full decision-making)
func (at *AutoTrader) runCycle() error {
	at.callCount++

	logger.Info("\n" + strings.Repeat("=", 70) + "\n")
	logger.Infof("⏰ %s - AI decision cycle #%d", time.Now().Format("2006-01-02 15:04:05"), at.callCount)
	logger.Info(strings.Repeat("=", 70))

	// 0. Check if trader is stopped (early exit to prevent trades after Stop() is called)
	at.isRunningMutex.RLock()
	running := at.isRunning
	at.isRunningMutex.RUnlock()
	if !running {
		at.logInfof("⏹ Trader is stopped, aborting cycle #%d", at.callCount)
		return nil
	}

	// 0a. PART A — CME SESSION GATE (hoisted to the TOP, before the account gate
	// and buildTradingContext). When the futures market is closed we skip the
	// ENTIRE cycle — no context build, no NT8 round-trips, no AI — and idle with
	// a longer cadence, logging only on the open⇄closed edge. This is the
	// approved fix for the "bot scans while the market is closed" symptom.
	// Crypto (TradingMode != "futures") returns false here → byte-identical.
	if at.cmeSessionClosedSkip() {
		return nil
	}

	// 0b. MULTI-ACCOUNT GATE (Stage 1): a NinjaTrader trader must have an EXPLICITLY
	// chosen NT8 account before it trades — never auto-trade on whatever account NT8
	// streams (that could be the LIVE account). The pick is persisted on the trader
	// row by handleSelectAccount; until it's set, skip the whole cycle (no decision,
	// no order). Re-read each cycle so a fresh pick opens the gate without a restart.
	if at.exchange == "ninjatrader" {
		chosenAccount := ""
		if tr, err := at.store.Trader().GetByID(at.id); err == nil && tr != nil {
			chosenAccount = tr.Account
		}
		if chosenAccount == "" {
			at.logWarnf("🚫 No NT8 account selected for this trader — skipping cycle #%d. Pick an account in the dashboard to start trading.", at.callCount)
			return nil
		}
	}

	// 0c. B5 — DEAD-MAN WATCHDOG. Observe the NT8 TCP link each cycle: a dropped
	// link blocks NEW entries (open-position management/exits are never gated) and,
	// on reconnect, holds entries blocked until a clean positions/orders
	// reconciliation, then auto-resumes. Non-NT traders are a no-op (crypto
	// byte-identical). The block itself is enforced in executeDecisionWithRecord.
	at.driveDeadManWatchdog()

	// Check USDC balance periodically for claw402 users (every 10 cycles)
	if at.callCount%10 == 0 && store.IsClaw402Config(at.config.AIModel) {
		at.checkClaw402Balance()
	}

	// Create decision record
	record := &store.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
	}

	// 1. Check if trading needs to be stopped
	if time.Now().Before(at.stopUntil) {
		remaining := at.stopUntil.Sub(time.Now())
		at.logWarnf("⏸ Risk control: Trading paused, remaining %.0f minutes", remaining.Minutes())
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Risk control paused, remaining %.0f minutes", remaining.Minutes())
		at.saveDecision(record)
		return nil
	}

	// 2. Reset daily P&L (reset every day)
	if time.Since(at.lastResetTime) > 24*time.Hour {
		at.dailyPnL = 0
		at.lastResetTime = time.Now()
		logger.Info("📅 Daily P&L reset")
	}

	// 4. Collect trading context
	ctx, err := at.buildTradingContext()
	if err != nil {
		at.logErrorf("failed to build trading context: %v", err)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Failed to build trading context: %v", err)
		at.saveDecision(record)
		return fmt.Errorf("failed to build trading context: %w", err)
	}

	// Plan 4 Stage 4 — defer-until-balance guard (NinjaTrader TCP only)
	// If equity is 0 and no account_balance frame has arrived yet, skip the cycle silently.
	// This prevents phantom HOLD decisions while waiting for the AddOn to connect.
	// Once the first account_balance arrives, equity > 0, and the gate opens normally.
	if ctx.Account.TotalEquity == 0 && !at.HasReceivedBalance() {
		// Silent return: no decision record, no log entry (to avoid noise during startup)
		return nil
	}

	// Save equity snapshot independently (decoupled from AI decision, used for drawing profit curve)
	// NOTE: Must be called BEFORE candidate coins check to ensure equity is always recorded
	at.saveEquitySnapshot(ctx)

	// If no candidate coins available, log but do not error
	if len(ctx.CandidateCoins) == 0 {
		at.logInfof("ℹ️ No candidate coins available, skipping this cycle")
		record.Success = true // Not an error, just no candidate coins
		record.ExecutionLog = append(record.ExecutionLog, "No candidate coins available, cycle skipped")
		record.AccountState = store.AccountSnapshot{
			TotalBalance:          ctx.Account.TotalEquity,
			AvailableBalance:      ctx.Account.AvailableBalance,
			TotalUnrealizedProfit: ctx.Account.UnrealizedPnL,
			PositionCount:         ctx.Account.PositionCount,
			InitialBalance:        at.initialBalance,
		}
		at.saveDecision(record)
		return nil
	}

	logger.Info(strings.Repeat("=", 70))
	for _, coin := range ctx.CandidateCoins {
		record.CandidateCoins = append(record.CandidateCoins, coin.Symbol)
	}

	at.logInfof("📊 Account equity: %.2f USDT | Available: %.2f USDT | Positions: %d",
		ctx.Account.TotalEquity, ctx.Account.AvailableBalance, ctx.Account.PositionCount)

	// 5. Use strategy engine to call AI for decision
	at.logInfof("🤖 Requesting AI analysis and decision... [Strategy Engine]")
	// Plan 4 Task 25 — decision latency timer (start)
	decisionStart := time.Now()
	// Prompt mode (Strategy Studio Phase 2): a per-strategy saved variant wins;
	// when NONE is saved we keep the original venue rule EXACTLY (see
	// resolvePromptVariant) so strategies with no saved variant are byte-
	// identical to the prior behavior. Read the saved variant null-safely.
	savedVariant := ""
	if eng := at.strategyEngine; eng != nil {
		if cfg := eng.GetConfig(); cfg != nil {
			savedVariant = cfg.PromptVariant
		}
	}
	promptVariant := resolvePromptVariant(at.exchange, savedVariant)
	aiDecision, err := kernel.GetFullDecisionWithStrategy(ctx, at.mcpClient, at.strategyEngine, promptVariant)
	// Plan 4 Task 25 — decision metrics
	telemetry.DecisionLatency.WithLabelValues(at.id).Observe(time.Since(decisionStart).Seconds())
	if aiDecision != nil {
		for _, d := range aiDecision.Decisions {
			status := "queued"
			if err != nil {
				status = "rejected"
			}
			telemetry.DecisionsTotal.WithLabelValues(at.id, d.Action, status).Inc()
		}
	}

	if aiDecision != nil && aiDecision.AIRequestDurationMs > 0 {
		record.AIRequestDurationMs = aiDecision.AIRequestDurationMs
		at.logInfof("⏱️ AI call duration: %.2f seconds", float64(record.AIRequestDurationMs)/1000)
		record.ExecutionLog = append(record.ExecutionLog,
			fmt.Sprintf("AI call duration: %d ms", record.AIRequestDurationMs))
	}

	// Save chain of thought, decisions, and input prompt even if there's an error (for debugging)
	if aiDecision != nil {
		record.SystemPrompt = aiDecision.SystemPrompt // Save system prompt
		record.InputPrompt = aiDecision.UserPrompt
		record.CoTTrace = aiDecision.CoTTrace
		record.RawResponse = aiDecision.RawResponse // Save raw AI response for debugging
		if len(aiDecision.Decisions) > 0 {
			decisionJSON, _ := json.MarshalIndent(aiDecision.Decisions, "", "  ")
			record.DecisionJSON = string(decisionJSON)
		}
	}

	// Record AI charge (track cost regardless of decision outcome)
	if aiDecision != nil && at.store != nil {
		if chargeErr := at.store.AICharge().Record(at.id, at.aiModel, at.config.AIModel); chargeErr != nil {
			at.logWarnf("⚠️ Failed to record AI charge: %v", chargeErr)
		}
	}

	if err != nil {
		at.consecutiveAIFailures++
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Failed to get AI decision: %v", err)

		// Activate safe mode after 3 consecutive failures
		if at.consecutiveAIFailures >= 3 && !at.safeMode {
			at.safeMode = true
			at.safeModeReason = fmt.Sprintf("AI failed %d consecutive times: %v", at.consecutiveAIFailures, err)
			at.logErrorf("🛡️ SAFE MODE ACTIVATED — AI failed %d times in a row. No new positions will be opened. Existing positions are protected with current stop-loss settings.", at.consecutiveAIFailures)
			at.logErrorf("🛡️ Reason: %v", err)
			at.logErrorf("🛡️ Action: Will keep trying AI each cycle. Safe mode auto-deactivates when AI recovers.")
		}

		// Print system prompt and AI chain of thought (output even with errors for debugging)
		if aiDecision != nil {
			logger.Info("\n" + strings.Repeat("=", 70) + "\n")
			logger.Infof("📋 System prompt (error case)")
			logger.Info(strings.Repeat("=", 70))
			logger.Info(aiDecision.SystemPrompt)
			logger.Info(strings.Repeat("=", 70))

			if aiDecision.CoTTrace != "" {
				logger.Info("\n" + strings.Repeat("-", 70) + "\n")
				logger.Info("💭 AI chain of thought analysis (error case):")
				logger.Info(strings.Repeat("-", 70))
				logger.Info(aiDecision.CoTTrace)
				logger.Info(strings.Repeat("-", 70))
			}
		}

		at.saveDecision(record)

		// In safe mode, don't return error — keep the loop running to retry next cycle
		if at.safeMode {
			at.logWarnf("🛡️ Safe mode: skipping this cycle, will retry in %v", at.config.ScanInterval)
			return nil
		}

		return fmt.Errorf("failed to get AI decision: %w", err)
	}

	// AI succeeded — reset failure counter and deactivate safe mode
	if at.consecutiveAIFailures > 0 {
		at.logInfof("✅ AI recovered after %d consecutive failures", at.consecutiveAIFailures)
	}
	at.consecutiveAIFailures = 0
	if at.safeMode {
		at.logInfof("🛡️ SAFE MODE DEACTIVATED — AI is working again. Resuming normal trading.")
		at.safeMode = false
		at.safeModeReason = ""
	}

	// // 5. Print system prompt
	// logger.Infof("\n" + strings.Repeat("=", 70))
	// logger.Infof("📋 System prompt [template: %s]", at.systemPromptTemplate)
	// logger.Info(strings.Repeat("=", 70))
	// logger.Info(decision.SystemPrompt)
	// logger.Infof(strings.Repeat("=", 70) + "\n")

	// 6. Print AI chain of thought
	// logger.Infof("\n" + strings.Repeat("-", 70))
	// logger.Info("💭 AI chain of thought analysis:")
	// logger.Info(strings.Repeat("-", 70))
	// logger.Info(decision.CoTTrace)
	// logger.Infof(strings.Repeat("-", 70) + "\n")

	// 7. Print AI decisions
	// logger.Infof("📋 AI decision list (%d items):\n", len(kernel.Decisions))
	// for i, d := range kernel.Decisions {
	//     logger.Infof("  [%d] %s: %s - %s", i+1, d.Symbol, d.Action, d.Reasoning)
	//     if d.Action == "open_long" || d.Action == "open_short" {
	//        logger.Infof("      Leverage: %dx | Position: %.2f USDT | Stop loss: %.4f | Take profit: %.4f",
	//           d.Leverage, d.PositionSizeUSD, d.StopLoss, d.TakeProfit)
	//     }
	// }
	logger.Info()
	logger.Info(strings.Repeat("-", 70))
	// 8. Sort decisions: ensure close positions first, then open positions (prevent position stacking overflow)
	logger.Info(strings.Repeat("-", 70))

	// GetFullDecisionWithStrategy returns (nil, nil) when the risk gate HOLDs
	// the cycle (e.g. Plan 3 T21 daily-loss limit) — there is no decision to
	// execute. Guard before dereferencing aiDecision.Decisions below. (A real
	// $0 balance in the brief pre-first-account_balance-frame window, Plan
	// 4.11, can trip the gate and surface this otherwise-latent nil deref.)
	if aiDecision == nil {
		at.logInfof("ℹ️ No actionable decision this cycle (risk gate HOLD); skipping execution")
		at.saveDecision(record)
		return nil
	}

	// 8. Sort decisions: ensure close positions first, then open positions (prevent position stacking overflow)
	sortedDecisions := sortDecisionsByPriority(aiDecision.Decisions)

	logger.Info("🔄 Execution order (optimized): Close positions first → Open positions later")
	for i, d := range sortedDecisions {
		logger.Infof("  [%d] %s %s", i+1, d.Symbol, d.Action)
	}
	logger.Info()

	// Check if trader is stopped before executing any decisions (prevent trades after Stop())
	at.isRunningMutex.RLock()
	running = at.isRunning
	at.isRunningMutex.RUnlock()
	if !running {
		at.logInfof("⏹ Trader stopped before decision execution, aborting cycle #%d", at.callCount)
		return nil
	}

	// Safe mode: filter out open positions, only allow close/hold
	if at.safeMode {
		filtered := make([]kernel.Decision, 0)
		for _, d := range sortedDecisions {
			if d.Action == "open_long" || d.Action == "open_short" {
				at.logWarnf("🛡️ Safe mode: BLOCKED %s %s (no new positions allowed)", d.Action, d.Symbol)
				continue
			}
			filtered = append(filtered, d)
		}
		sortedDecisions = filtered
		if len(sortedDecisions) == 0 {
			at.logInfof("🛡️ Safe mode: all decisions were open positions, nothing to execute")
		}
	}

	// Execute decisions and record results
	for _, d := range sortedDecisions {
		// Check if trader is stopped before each decision (allow immediate stop during execution)
		at.isRunningMutex.RLock()
		running = at.isRunning
		at.isRunningMutex.RUnlock()
		if !running {
			at.logInfof("⏹ Trader stopped during decision execution, aborting remaining decisions")
			break
		}

		actionRecord := store.DecisionAction{
			Action:     d.Action,
			Symbol:     d.Symbol,
			Quantity:   0,
			Leverage:   d.Leverage,
			Price:      0,
			StopLoss:   d.StopLoss,
			TakeProfit: d.TakeProfit,
			Confidence: d.Confidence,
			Reasoning:  d.Reasoning,
			Timestamp:  time.Now().UTC(),
			Success:    false,
		}

		// Hold-lock: once a position is OPEN, suppress an AI-initiated close so the
		// trade rides to the stop/target the AI set (a real OCO bracket resting at
		// the exchange). Opt-in per strategy, default OFF. Only AI decisions pass
		// through here; Emergency Flat + the drawdown monitor call the trader
		// directly and bypass this. Opening logic and closing a flat symbol are
		// untouched.
		if d.Action == "close_long" || d.Action == "close_short" {
			if at.holdLockSuppressesClose(&d, &actionRecord) {
				record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("🔒 %s %s suppressed (hold-lock: riding to stop/target)", d.Symbol, d.Action))
				record.Decisions = append(record.Decisions, actionRecord)
				continue
			}
		}

		if err := at.executeDecisionWithRecord(&d, &actionRecord); err != nil {
			at.logErrorf("❌ Failed to execute decision (%s %s): %v", d.Symbol, d.Action, err)
			actionRecord.Error = err.Error()
			record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ %s %s failed: %v", d.Symbol, d.Action, err))
		} else {
			actionRecord.Success = true
			record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("✓ %s %s succeeded", d.Symbol, d.Action))
			// Brief delay after successful execution
			time.Sleep(1 * time.Second)
		}

		record.Decisions = append(record.Decisions, actionRecord)
	}

	// 9. Save decision record
	if err := at.saveDecision(record); err != nil {
		at.logWarnf("⚠ Failed to save decision record: %v", err)
	}

	return nil
}

// cmeSessionClosedSkip is the hoisted CME session gate (PART A). It returns true
// when the whole cycle should be skipped because the futures market is closed —
// after logging the open⇄closed edge and idling with a backoff. In non-futures
// (crypto) mode it always returns false, so those traders are byte-identical.
func (at *AutoTrader) cmeSessionClosedSkip() bool {
	if config.Get().TradingMode != "futures" {
		return false
	}
	open := kernel.IsCMEOpen(time.Now())
	at.noteCMESessionEdge(open)
	if open {
		return false
	}
	at.backoffWhileClosed()
	return true
}

// noteCMESessionEdge logs the market open⇄closed transition ONCE (not every
// cycle). On a fresh start into a closed market it announces the closure + the
// next open immediately; a fresh start into an open market stays silent (there
// is nothing to "resume"). Single-goroutine (runCycle) → no locking on cmePrevOpen.
func (at *AutoTrader) noteCMESessionEdge(open bool) {
	prev := at.cmePrevOpen
	at.cmePrevOpen = &open
	if prev != nil && *prev == open {
		return // no edge — stay quiet
	}
	if !open {
		_, reason := kernel.CMEClosedReason(time.Now())
		next := kernel.NextCMEOpen(time.Now())
		chicago, err := time.LoadLocation("America/Chicago")
		if err != nil {
			chicago = time.UTC
		}
		at.logInfof("🌙 CME closed (%s) — next open %s", reason, next.In(chicago).Format("Mon 2006-01-02 15:04 MST"))
	} else if prev != nil {
		at.logInfof("☀️ CME open — resuming.")
	}
}

// backoffWhileClosed idles for cmeClosedBackoff while the market is shut, in
// short stop-responsive slices so Stop() returns promptly instead of the loop
// spinning a full cycle every ScanInterval (~1 min) all weekend. Returns early
// the moment the trader is stopped.
func (at *AutoTrader) backoffWhileClosed() {
	const slice = 10 * time.Second
	const cmeClosedBackoff = 3 * time.Minute // within the ~2–5 min target
	for waited := time.Duration(0); waited < cmeClosedBackoff; waited += slice {
		at.isRunningMutex.RLock()
		running := at.isRunning
		at.isRunningMutex.RUnlock()
		if !running {
			return
		}
		time.Sleep(slice)
	}
}

// buildTradingContext builds trading context
func (at *AutoTrader) buildTradingContext() (*kernel.Context, error) {
	// 1. Get account information
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	// Get account fields
	totalWalletBalance := 0.0
	totalUnrealizedProfit := 0.0
	availableBalance := 0.0
	totalEquity := 0.0

	if wallet, ok := balance["totalWalletBalance"].(float64); ok {
		totalWalletBalance = wallet
	}
	if unrealized, ok := balance["totalUnrealizedProfit"].(float64); ok {
		totalUnrealizedProfit = unrealized
	}
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Use totalEquity directly if provided by trader (more accurate)
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		totalEquity = eq
	} else {
		// Fallback: Total Equity = Wallet balance + Unrealized profit
		totalEquity = totalWalletBalance + totalUnrealizedProfit
	}

	// 2. Get position information
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var positionInfos []kernel.PositionInfo
	totalMarginUsed := 0.0

	// Current position key set (for cleaning up closed position records)
	currentPositionKeys := make(map[string]bool)

	for _, pos := range positions {
		// Comma-ok every assert: NT futures positions omit Binance-only fields
		// (e.g. liquidationPrice — futures have no liquidation), so an unchecked
		// .(float64) on a missing key panics and takes down the whole bot.
		symbol, _ := pos["symbol"].(string)
		if symbol == "" {
			continue
		}
		side, _ := pos["side"].(string)
		entryPrice, _ := pos["entryPrice"].(float64)
		markPrice, _ := pos["markPrice"].(float64)
		quantity, _ := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity // Short position quantity is negative, convert to positive
		}

		// Skip closed positions (quantity = 0), prevent "ghost positions" from being passed to AI
		if quantity == 0 {
			continue
		}

		unrealizedPnl, _ := pos["unRealizedProfit"].(float64)
		liquidationPrice, _ := pos["liquidationPrice"].(float64)

		// Calculate margin used (estimated)
		leverage := 10 // Default value, should actually be fetched from position info
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		totalMarginUsed += marginUsed

		// Calculate P&L percentage (based on margin, considering leverage)
		pnlPct := calculatePnLPercentage(unrealizedPnl, marginUsed)

		// Get position open time from exchange (preferred) or fallback to local tracking
		posKey := symbol + "_" + side
		currentPositionKeys[posKey] = true

		var updateTime int64
		// Priority 1: Get from database (trader_positions table) - most accurate
		if at.store != nil {
			if dbPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, symbol, side); err == nil && dbPos != nil {
				if dbPos.EntryTime > 0 {
					updateTime = dbPos.EntryTime
				}
			}
		}
		// Priority 2: Get from exchange API (Bybit: createdTime, OKX: createdTime)
		if updateTime == 0 {
			if createdTime, ok := pos["createdTime"].(int64); ok && createdTime > 0 {
				updateTime = createdTime
			}
		}
		// Priority 3: Fallback to local tracking
		if updateTime == 0 {
			if _, exists := at.positionFirstSeenTime[posKey]; !exists {
				at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()
			}
			updateTime = at.positionFirstSeenTime[posKey]
		}

		// Get peak profit rate for this position
		at.peakPnLCacheMutex.RLock()
		peakPnlPct := at.peakPnLCache[posKey]
		at.peakPnLCacheMutex.RUnlock()

		positionInfos = append(positionInfos, kernel.PositionInfo{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         quantity,
			Leverage:         leverage,
			UnrealizedPnL:    unrealizedPnl,
			UnrealizedPnLPct: pnlPct,
			PeakPnLPct:       peakPnlPct,
			LiquidationPrice: liquidationPrice,
			MarginUsed:       marginUsed,
			UpdateTime:       updateTime,
		})
	}

	// Clean up closed position records
	for key := range at.positionFirstSeenTime {
		if !currentPositionKeys[key] {
			delete(at.positionFirstSeenTime, key)
		}
	}

	// 3. Use strategy engine to get candidate coins (must have strategy engine)
	var candidateCoins []kernel.CandidateCoin
	if at.strategyEngine == nil {
		at.logWarnf("⚠️ No strategy engine configured, skipping candidate coins")
	} else {
		coins, err := at.strategyEngine.GetCandidateCoins()
		if err != nil {
			// Log warning but don't fail - equity snapshot should still be saved
			at.logWarnf("⚠️ Failed to get candidate coins: %v (will use empty list)", err)
		} else {
			candidateCoins = coins
			logger.Infof("📋 [%s] Strategy engine fetched candidate coins: %d", at.name, len(candidateCoins))
		}
	}

	// 4. Calculate total P&L
	totalPnL := totalEquity - at.initialBalance
	totalPnLPct := 0.0
	if at.initialBalance > 0 {
		totalPnLPct = (totalPnL / at.initialBalance) * 100
	}

	marginUsedPct := 0.0
	if totalEquity > 0 {
		marginUsedPct = (totalMarginUsed / totalEquity) * 100
	}

	// 5. Get leverage from strategy config
	strategyConfig := at.strategyEngine.GetConfig()
	btcEthLeverage := strategyConfig.RiskControl.BTCETHMaxLeverage
	altcoinLeverage := strategyConfig.RiskControl.AltcoinMaxLeverage
	logger.Infof("📋 [%s] Strategy leverage config: BTC/ETH=%dx, Altcoin=%dx", at.name, btcEthLeverage, altcoinLeverage)

	// 6. Build context
	ctx := &kernel.Context{
		CurrentTime:     time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		RuntimeMinutes:  int(time.Since(at.startTime).Minutes()),
		CallCount:       at.callCount,
		BTCETHLeverage:  btcEthLeverage,
		AltcoinLeverage: altcoinLeverage,
		Account: kernel.AccountInfo{
			TotalEquity:      totalEquity,
			AvailableBalance: availableBalance,
			UnrealizedPnL:    totalUnrealizedProfit,
			TotalPnL:         totalPnL,
			TotalPnLPct:      totalPnLPct,
			MarginUsed:       totalMarginUsed,
			MarginUsedPct:    marginUsedPct,
			PositionCount:    len(positionInfos),
		},
		Positions:      positionInfos,
		CandidateCoins: candidateCoins,
	}

	// 7. Add recent closed trades (if store is available)
	if at.store != nil {
		// Strategy Studio P1 — daily-guardrail inputs on the CME session-day:
		// today's realized P&L (closed trades) + entry count, scoped to the active
		// account. The kernel daily-guardrail gate (engine_analysis.go) reads these.
		sinceMs := kernel.CMESessionDayStart(time.Now()).UnixMilli()
		if dayPnL, dayTrades, derr := at.store.Position().GetSessionDayActivity(at.id, sinceMs, at.currentAccountName()); derr != nil {
			at.logWarnf("⚠️ Failed to compute daily-guardrail activity: %v", derr)
		} else {
			ctx.DailyRealizedPnL = dayPnL
			ctx.TradesToday = dayTrades
		}

		// Get recent 10 closed trades for AI context, scoped to the ACTIVE
		// account so a post-switch cycle does not carry the old account's
		// trades into the prompt (empty account → trader-global fallback).
		recentTrades, err := at.store.Position().GetRecentTrades(at.id, 10, at.currentAccountName())
		if err != nil {
			at.logWarnf("⚠️ Failed to get recent trades: %v", err)
		} else {
			logger.Infof("📊 [%s] Found %d recent closed trades for AI context", at.name, len(recentTrades))
			for _, trade := range recentTrades {
				// Convert Unix timestamps to formatted strings for AI readability
				entryTimeStr := ""
				if trade.EntryTime > 0 {
					entryTimeStr = time.Unix(trade.EntryTime, 0).UTC().Format("01-02 15:04 UTC")
				}
				exitTimeStr := ""
				if trade.ExitTime > 0 {
					exitTimeStr = time.Unix(trade.ExitTime, 0).UTC().Format("01-02 15:04 UTC")
				}

				ctx.RecentOrders = append(ctx.RecentOrders, kernel.RecentOrder{
					Symbol:       trade.Symbol,
					Side:         trade.Side,
					EntryPrice:   trade.EntryPrice,
					ExitPrice:    trade.ExitPrice,
					RealizedPnL:  trade.RealizedPnL,
					PnLPct:       trade.PnLPct,
					EntryTime:    entryTimeStr,
					ExitTime:     exitTimeStr,
					HoldDuration: trade.HoldDuration,
				})
			}
		}
		// Get trading statistics for AI context, scoped to the ACTIVE account so
		// the prompt's aggregate stats reflect the current account (not the old
		// one after a switch); empty account → trader-global fallback.
		stats, err := at.store.Position().GetFullStats(at.id, at.currentAccountName())
		if err != nil {
			at.logWarnf("⚠️ Failed to get trading stats: %v", err)
		} else if stats == nil {
			at.logWarnf("⚠️ GetFullStats returned nil")
		} else if stats.TotalTrades == 0 {
			at.logWarnf("⚠️ GetFullStats returned 0 trades")
		} else {
			ctx.TotalRealizedPnL = stats.TotalPnL // Chunk 5 — consistency-rule input (all-time realized P&L)
			ctx.TradingStats = &kernel.TradingStats{
				TotalTrades:    stats.TotalTrades,
				WinRate:        stats.WinRate,
				ProfitFactor:   stats.ProfitFactor,
				SharpeRatio:    stats.SharpeRatio,
				TotalPnL:       stats.TotalPnL,
				AvgWin:         stats.AvgWin,
				AvgLoss:        stats.AvgLoss,
				MaxDrawdownPct: stats.MaxDrawdownPct,
			}
			logger.Infof("📈 [%s] Trading stats: %d trades, %.1f%% win rate, PF=%.2f, Sharpe=%.2f, DD=%.1f%%",
				at.name, stats.TotalTrades, stats.WinRate, stats.ProfitFactor, stats.SharpeRatio, stats.MaxDrawdownPct)
		}
	} else {
		at.logWarnf("⚠️ Store is nil, cannot get recent trades")
	}

	// 8. Get quantitative data (if enabled in strategy config)
	if strategyConfig.Indicators.EnableQuantData {
		// Collect symbols to query (candidate coins + position coins)
		symbolsToQuery := make(map[string]bool)
		for _, coin := range candidateCoins {
			symbolsToQuery[coin.Symbol] = true
		}
		for _, pos := range positionInfos {
			symbolsToQuery[pos.Symbol] = true
		}

		symbols := make([]string, 0, len(symbolsToQuery))
		for sym := range symbolsToQuery {
			symbols = append(symbols, sym)
		}

		logger.Infof("📊 [%s] Fetching quantitative data for %d symbols...", at.name, len(symbols))
		ctx.QuantDataMap = at.strategyEngine.FetchQuantDataBatch(symbols)
		logger.Infof("📊 [%s] Successfully fetched quantitative data for %d symbols", at.name, len(ctx.QuantDataMap))
	}

	// 9. Get OI ranking data (market-wide position changes)
	if strategyConfig.Indicators.EnableOIRanking {
		logger.Infof("📊 [%s] Fetching OI ranking data...", at.name)
		ctx.OIRankingData = at.strategyEngine.FetchOIRankingData()
		if ctx.OIRankingData != nil {
			logger.Infof("📊 [%s] OI ranking data ready: %d top, %d low positions",
				at.name, len(ctx.OIRankingData.TopPositions), len(ctx.OIRankingData.LowPositions))
		}
	}

	// 10. Get NetFlow ranking data (market-wide fund flow)
	if strategyConfig.Indicators.EnableNetFlowRanking {
		logger.Infof("💰 [%s] Fetching NetFlow ranking data...", at.name)
		ctx.NetFlowRankingData = at.strategyEngine.FetchNetFlowRankingData()
		if ctx.NetFlowRankingData != nil {
			logger.Infof("💰 [%s] NetFlow ranking data ready: inst_in=%d, inst_out=%d",
				at.name, len(ctx.NetFlowRankingData.InstitutionFutureTop), len(ctx.NetFlowRankingData.InstitutionFutureLow))
		}
	}

	// 11. Get Price ranking data (market-wide gainers/losers)
	if strategyConfig.Indicators.EnablePriceRanking {
		logger.Infof("📈 [%s] Fetching Price ranking data...", at.name)
		ctx.PriceRankingData = at.strategyEngine.FetchPriceRankingData()
		if ctx.PriceRankingData != nil {
			logger.Infof("📈 [%s] Price ranking data ready for %d durations",
				at.name, len(ctx.PriceRankingData.Durations))
		}
	}

	return ctx, nil
}

// sortDecisionsByPriority sorts decisions: close positions first, then open positions, finally hold/wait
// This avoids position stacking overflow when changing positions
func sortDecisionsByPriority(decisions []kernel.Decision) []kernel.Decision {
	if len(decisions) <= 1 {
		return decisions
	}

	// Define priority
	getActionPriority := func(action string) int {
		switch action {
		case "close_long", "close_short":
			return 1 // Highest priority: close positions first
		case "open_long", "open_short":
			return 2 // Second priority: open positions later
		case "hold", "wait":
			return 3 // Lowest priority: wait
		default:
			return 999 // Unknown actions at the end
		}
	}

	// Copy decision list
	sorted := make([]kernel.Decision, len(decisions))
	copy(sorted, decisions)

	// Sort by priority
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if getActionPriority(sorted[i].Action) > getActionPriority(sorted[j].Action) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// checkClaw402Balance checks USDC balance and logs warnings if low
func (at *AutoTrader) checkClaw402Balance() {
	scanMinutes := int(at.config.ScanInterval.Minutes())
	if scanMinutes <= 0 {
		scanMinutes = 3
	}
	dailyCost, _ := store.EstimateRunway(1.0, at.config.CustomModelName, scanMinutes)
	logger.Infof("💰 [%s] Estimated daily AI cost: ~$%.2f (model: %s, interval: %dm)",
		at.name, dailyCost, at.config.CustomModelName, scanMinutes)

	if at.claw402WalletAddr != "" {
		balance, err := wallet.QueryUSDCBalance(at.claw402WalletAddr)
		if err != nil {
			at.logWarnf("⚠️ Failed to query USDC balance: %v", err)
			return
		}

		if balance < 1.0 {
			at.logWarnf("⚠️ Low USDC balance: $%.2f — AI may stop soon!", balance)
		}
		if balance <= 0 {
			at.logErrorf("🚨 USDC balance is ZERO — AI calls will fail!")
		}

		runway := float64(0)
		if dailyCost > 0 {
			runway = balance / dailyCost
		}
		logger.Infof("💰 [%s] USDC Balance: $%.2f | Daily AI cost: ~$%.2f | Runway: ~%.1f days",
			at.name, balance, dailyCost, runway)
	}
}
