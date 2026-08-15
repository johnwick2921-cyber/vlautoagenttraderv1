package trader

import (
	"fmt"
	"math"
	"nofx/discipline"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"nofx/telemetry"
	ntTrader "nofx/trader/ninjatrader"
	"strings"
	"time"
)

// maxFuturesContracts caps the per-order contract count for CME futures
// (SIM-conservative). Tune per account size; 10 MNQ ≈ $600k notional ≈ $22k
// intraday margin, comfortably within a $50k SIM account.
const maxFuturesContracts = 10.0

// futuresOrderQuantity converts a decision's notional (position_size_usd) into
// a clamped contract count for CME futures: contracts = notional / (price ×
// pointValue), rounded, floored at 1, capped at maxFuturesContracts. Bypasses
// the crypto notional/leverage margin model (futures margin is per-contract).
func futuresOrderQuantity(symbol string, notionalUSD, price float64, maxContracts int) float64 {
	pv := market.FuturesPointValue(symbol)
	if pv <= 0 || price <= 0 {
		return 1 // safe default: 1 contract
	}
	contracts := math.Round(notionalUSD / (price * pv))
	if contracts < 1 {
		contracts = 1
	}
	// Chunk 3 — max-contracts clamp (was a silent hardcoded 10). maxContracts<=0
	// → no clamp (the guardrail is disabled by master/toggle). Log when clamping.
	if maxContracts > 0 && contracts > float64(maxContracts) {
		logger.Infof("  ⚠️ [RISK CONTROL] %s order %.0f contracts exceeds max %d — clamping to %d", symbol, contracts, maxContracts, maxContracts)
		contracts = float64(maxContracts)
	}
	return contracts
}

// resolveMaxContracts returns the futures max-contracts clamp for this trader's
// strategy (per-strategy value, else the 10-contract venue default). Hardening D3
// (audit F2): ALWAYS ON — the guardrails master switch no longer disables it.
func (at *AutoTrader) resolveMaxContracts() int {
	if at.config.StrategyConfig == nil {
		return int(maxFuturesContracts)
	}
	rc := at.config.StrategyConfig.RiskControl
	return kernel.ResolveMaxContracts(rc.MaxContractsPerOrder, int(maxFuturesContracts))
}

// hlBool resolves a three-state *bool toggle (nil → default).
func hlBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// holdLockSuppressesClose reports whether the "Hold discipline" hold-lock should
// suppress this AI-initiated close: the per-strategy toggle is ON AND an OPEN
// position exists for the decision's symbol/side. When it returns true it has
// already logged the suppression loudly and marked the record as a deliberate
// no-op (Success, no error). The trade then rides to the stop/target the AI set —
// a real OCO bracket resting at the exchange (VLTraderTCPClient.SubmitBracketOnEntryFill),
// so the position stays protected. Emergency Flat (handler_risk.go) and the
// drawdown monitor call the trader directly and never reach this path, so human
// and safety closes always go through. Default OFF ⇒ behavior byte-identical.
func (at *AutoTrader) holdLockSuppressesClose(d *kernel.Decision, rec *store.DecisionAction) bool {
	if d.Action != "close_long" && d.Action != "close_short" {
		return false
	}
	if at.store == nil || at.config.StrategyConfig == nil {
		return false
	}
	if !hlBool(at.config.StrategyConfig.RiskControl.HoldDisciplineEnabled, false) {
		return false
	}
	side := "LONG"
	if d.Action == "close_short" {
		side = "SHORT"
	}
	openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, market.Normalize(d.Symbol), side)
	if err != nil || openPos == nil {
		return false // flat for this side → nothing to hold; allow the (no-op) close
	}
	// Best-effort current price for the log line (non-fatal if unavailable).
	px := 0.0
	if md, e := market.GetWithExchange(d.Symbol, at.exchange); e == nil {
		px = md.CurrentPrice
	}
	at.logWarnf("🔒 HOLD-LOCK: AI %s suppressed for %s (price %.2f, entry %.2f) — trade rides to stop/target.",
		d.Action, d.Symbol, px, openPos.EntryPrice)
	rec.Price = px
	rec.Success = true // deliberate no-op, not a failure
	return true
}

// consecutiveLossHalted reports whether new entries are blocked by the D1
// consecutive-loss halt: N consecutive LOSING closed trades in the current CME
// session-day (0 = OFF; resets on a win/break-even close or a new session). It is
// a per-strategy circuit breaker, NOT gated by the guardrails master switch.
// Fail-OPEN on a query error — never block a trade because the DB hiccuped.
func (at *AutoTrader) consecutiveLossHalted() (string, bool) {
	if at.store == nil || at.config.StrategyConfig == nil {
		return "", false
	}
	n := at.config.StrategyConfig.RiskControl.ConsecutiveLossHalt
	if n <= 0 {
		return "", false // OFF
	}
	sinceMs := kernel.CMESessionDayStart(time.Now()).UnixMilli()
	losses, err := at.store.Position().CountConsecutiveLossesSince(at.id, sinceMs)
	if err != nil {
		at.logWarnf("consecutive-loss halt: count query failed (%v) — allowing entry (fail-open)", err)
		return "", false
	}
	if losses >= n {
		return fmt.Sprintf("%d consecutive losing trades this session (limit %d)", losses, n), true
	}
	return "", false
}

// executeDecisionWithRecord executes AI decision and records detailed information
func (at *AutoTrader) executeDecisionWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	// Feed-down gate (NinjaTrader, TRACK A): the SIM cannot fill without market
	// data, so an entry/flatten issued while the feed is down is rejected ("no
	// market data") — the upstream condition behind the phantom-close mess. Refuse
	// opens AND closes until the feed is Connected; the position simply waits (the
	// close path retries on the next cycle / reconnect). Default-ALLOW until a
	// feed_status frame arrives, so a healthy bot is never false-halted.
	switch decision.Action {
	case "open_long", "open_short", "close_long", "close_short":
		if down, status := at.ninjaFeedDown(); down {
			at.logWarnf("⛔ feed-gate: %s %s skipped — NT8 price feed not Connected (status=%q); SIM would reject 'no market data'. Will act when the feed returns.", decision.Action, decision.Symbol, status)
			telemetry.IncGateBlock(at.id, "feed_down")
			return nil
		}
	}

	// B5 — dead-man watchdog: after an NT8 TCP disconnect, NEW entries stay blocked
	// until a clean positions/orders reconciliation, so we never open on top of a
	// state we haven't re-verified across a link gap. Closes / open-position
	// management are NEVER blocked. State is advanced once per cycle in runCycle
	// (driveDeadManWatchdog); here we only enforce the block.
	switch decision.Action {
	case "open_long", "open_short":
		if at.deadMan.entriesBlocked() {
			at.logWarnf("⛔ dead-man watchdog: %s %s REFUSED — NT8 link not yet reconciled after a disconnect; entries resume after a clean reconciliation.", decision.Symbol, decision.Action)
			telemetry.IncGateBlock(at.id, "dead_man")
			actionRecord.Success = false
			actionRecord.Error = "dead_man_watchdog: awaiting reconciliation after link gap"
			return nil
		}
	}

	// A4 (G4) — FREEZE gate: a trader frozen by an identity/account echo mismatch
	// (A2) or a reconcile belief≠broker divergence is blocked from NEW entries until
	// the owner clears it (POST /api/risk/clear-freeze). Open-position management
	// (closes/stop-moves/reconcile) is NEVER blocked — a frozen trader can still be
	// brought flat.
	switch decision.Action {
	case "open_long", "open_short":
		if reason, frozen := discipline.IsFrozen(at.id); frozen {
			at.logErrorf("🚨 A4 FROZEN: %s %s REFUSED — trader is frozen (%s). Investigate, then clear via /api/risk/clear-freeze to resume.", decision.Symbol, decision.Action, reason)
			telemetry.IncGateBlock(at.id, "frozen")
			actionRecord.Success = false
			actionRecord.Error = "frozen: " + reason
			return nil
		}
	}

	// D1 — consecutive-loss halt: after N consecutive losing closed trades this CME
	// session-day, block NEW entries until the next session. Closes (open-position
	// management) are NEVER blocked. 0 = OFF.
	switch decision.Action {
	case "open_long", "open_short":
		if reason, halted := at.consecutiveLossHalted(); halted {
			at.logWarnf("🛑 consecutive-loss halt: %s entry REFUSED — %s. No new entries until the next CME session.", decision.Symbol, reason)
			telemetry.IncGateBlock(at.id, "consecutive_loss")
			actionRecord.Success = false
			actionRecord.Error = "consecutive_loss_halt: " + reason
			return nil
		}
	}

	// P2.3 — LAST-ENTRY cutoff: block NEW entries after the day-trader last-entry
	// time (default 13:00 CT = 14:00 ET). Gated on day_plan → dormant by default.
	// Closes (open-position management) are NEVER blocked.
	switch decision.Action {
	case "open_long", "open_short":
		if reason, blocked := at.entryBlockedByLastEntry(); blocked {
			at.logWarnf("🕒 last-entry cutoff: %s %s REFUSED — %s. Entries reopen next session.", decision.Symbol, decision.Action, reason)
			telemetry.IncGateBlock(at.id, "last_entry")
			actionRecord.Success = false
			actionRecord.Error = "last_entry_cutoff: " + reason
			return nil
		}
	}

	switch decision.Action {
	case "open_long":
		return at.executeOpenLongWithRecord(decision, actionRecord)
	case "open_short":
		return at.executeOpenShortWithRecord(decision, actionRecord)
	case "close_long":
		return at.executeCloseLongWithRecord(decision, actionRecord)
	case "close_short":
		return at.executeCloseShortWithRecord(decision, actionRecord)
	case "hold", "wait":
		// No execution needed, just record
		return nil
	default:
		return fmt.Errorf("unknown action: %s", decision.Action)
	}
}

// reconcileFlattenTimeout / PollInterval bound the flatten-first await in
// reconcileBeforeOpenNT — the auto-flatten polls NT8 net until flat, then opens.
const (
	// Heartbeat-aware backstop: the primary confirmation is the fill-confirmed
	// position_close FRAME (arrives ~instantly), so this timeout is only hit when no
	// frame comes — in which case we must give the 30s all-account positions
	// heartbeat a chance before refusing. 35s > the 30s heartbeat (was 6s, which
	// starved a non-active account whose snapshot only refreshes every 30s).
	reconcileFlattenTimeout      = 35 * time.Second
	reconcileFlattenPollInterval = 500 * time.Millisecond
)

// ntHeldPosition returns the side ("long"/"short") NT8 currently holds for symbol,
// or "" if flat / unreadable. Reads the NT8 positions snapshot via GetPositions.
func (at *AutoTrader) ntHeldPosition(symbol string) string {
	positions, err := at.trader.GetPositions()
	if err != nil {
		return "" // read error → treat as flat; the existing open-path checks + reconcile cover it
	}
	for _, pos := range positions {
		if pos["symbol"] != symbol {
			continue
		}
		amt, _ := pos["positionAmt"].(float64)
		if amt > 0 {
			// Normalize casing: NT8 GetPositions/positionMap emits UPPERCASE
			// "LONG"/"SHORT"; reconcileBeforeOpenNT compares held == "long", so an
			// un-normalized "LONG" fell to the else branch and flattened the WRONG
			// side (CloseShort on a long orphan) — the flatten never confirmed flat
			// and the open was refused every cycle. Return lowercase to match.
			if s, _ := pos["side"].(string); s != "" {
				return strings.ToLower(s)
			}
			return "long"
		}
	}
	return ""
}

// reconcileBeforeOpenNT (TRACK B): NinjaTrader-only defense-in-depth run before an
// entry. The bot is about to OPEN, so it believes it is flat; if NT8 still holds a
// position for this symbol (an orphan), flatten-first AWAITING its own fill, then
// allow the open. If the flatten cannot be confirmed flat (timeout / feed drop),
// REFUSE the open — never compound onto an unreconciled net (the id=46 harm).
// No-op for non-NT traders or when NT8 is flat.
//
// STEP-A note: the open path already refuses to open on a SAME-side held position,
// and 0118ca77 (no phantom close → the bot knows it is still in a position) + the
// Track-A feed gate prevent the orphan at the source. This adds AUTO-RECOVERY
// (flatten + proceed instead of refusing every cycle) and widens to EITHER side.
// It does NOT cure a stale snapshot (the actual id=46 bypass) — that is Track A +
// 0118ca77; this acts only on a snapshot that positively reports a held position.
func (at *AutoTrader) reconcileBeforeOpenNT(symbol, intendedSide string) error {
	if at.exchange != "ninjatrader" {
		return nil
	}
	// Never flatten into a dead feed (Track A also gates upstream; be defensive).
	if down, status := at.ninjaFeedDown(); down {
		return fmt.Errorf("reconcile-before-open: NT8 feed not Connected (%s) — refusing open", status)
	}
	held := at.ntHeldPosition(symbol)
	if held == "" {
		return nil // NT8 flat → proceed
	}
	at.logWarnf("🚨 reconcile-before-open: NT8 holds a %s %s before an intended %s open — flattening first (awaiting fill) to avoid compounding onto an orphan.", held, symbol, intendedSide)
	// Timestamp BEFORE the flatten so we only accept a close that our flatten caused.
	t0 := time.Now().UnixMilli()
	var ferr error
	if held == "long" {
		_, ferr = at.trader.CloseLong(symbol, 0)
	} else {
		_, ferr = at.trader.CloseShort(symbol, 0)
	}
	if ferr != nil {
		return fmt.Errorf("reconcile-before-open: flatten submit failed: %w", ferr)
	}
	// AWAIT its own fill (NOT fire-and-forget — the trap 0118ca77 fixed). Prefer the
	// FILL-CONFIRMED close FRAME (position_close), which arrives ~instantly for the
	// bound account even when it is NOT the streamed/active account — so a non-active
	// trader no longer waits on the 30s positions-snapshot heartbeat. The snapshot
	// (ntHeldPosition) remains a fallback, and the timeout is heartbeat-aware.
	ntTCP, _ := at.trader.(*ntTrader.TCPTrader)
	deadline := time.Now().Add(reconcileFlattenTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(reconcileFlattenPollInterval)
		if down, _ := at.ninjaFeedDown(); down {
			return fmt.Errorf("reconcile-before-open: feed dropped during flatten — refusing open")
		}
		// Frame path (fast, account-correct): our flatten's close was fill-confirmed.
		if ntTCP != nil && ntTCP.CloseConfirmedSince(symbol, held, t0) {
			at.logInfof("✅ reconcile-before-open: %s flatten fill-confirmed via position_close frame — proceeding to open.", symbol)
			return nil
		}
		// Snapshot fallback (covers a manual/external flatten with no close frame).
		if at.ntHeldPosition(symbol) == "" {
			at.logInfof("✅ reconcile-before-open: %s flattened + confirmed flat (snapshot) — proceeding to open.", symbol)
			return nil
		}
	}
	return fmt.Errorf("reconcile-before-open: flatten not confirmed flat within %s — refusing open (never compound)", reconcileFlattenTimeout)
}

// executeOpenLongWithRecord executes open long position and records detailed information
func (at *AutoTrader) executeOpenLongWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  📈 Open long: %s", decision.Symbol)

	// TRACK B — reconcile NT8 net before opening; flatten an orphan first or refuse.
	if err := at.reconcileBeforeOpenNT(decision.Symbol, "long"); err != nil {
		return err
	}

	// ⚠️ Get current positions for multiple checks
	positions, err := at.trader.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// [CODE ENFORCED] Check max positions limit
	if err := at.enforceMaxPositions(len(positions)); err != nil {
		return err
	}

	// Check if there's already a position in the same symbol and direction
	for _, pos := range positions {
		if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
			return fmt.Errorf("❌ %s already has long position, close it first", decision.Symbol)
		}
	}

	// Get current price
	marketData, err := market.GetWithExchange(decision.Symbol, at.exchange)
	if err != nil {
		return err
	}

	// Get balance (needed for multiple checks)
	balance, err := at.trader.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}
	availableBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Get equity for position value ratio check
	equity := 0.0
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		equity = eq
	} else if eq, ok := balance["totalWalletBalance"].(float64); ok && eq > 0 {
		equity = eq
	} else {
		equity = availableBalance // Fallback to available balance
	}

	// [CODE ENFORCED] Position Value Ratio Check: position_value <= equity × ratio
	adjustedPositionSize, wasCapped := at.enforcePositionValueRatio(decision.PositionSizeUSD, equity, decision.Symbol)
	if wasCapped {
		decision.PositionSizeUSD = adjustedPositionSize
	}

	// Calculate order quantity.
	var quantity float64
	if market.IsCMEFuturesSymbol(decision.Symbol) {
		// CME futures: size in contracts (notional / (price × point value)),
		// clamped. Skip the crypto notional/leverage margin model — futures
		// margin is per-contract, not notional/leverage.
		quantity = futuresOrderQuantity(decision.Symbol, decision.PositionSizeUSD, marketData.CurrentPrice, at.resolveMaxContracts())
	} else {
		// ⚠️ Auto-adjust position size if insufficient margin (crypto)
		// Formula: totalRequired = positionSize/leverage + positionSize*0.001 + positionSize/leverage*0.01
		//        = positionSize * (1.01/leverage + 0.001)
		marginFactor := 1.01/float64(decision.Leverage) + 0.001
		maxAffordablePositionSize := availableBalance / marginFactor

		actualPositionSize := decision.PositionSizeUSD
		if actualPositionSize > maxAffordablePositionSize {
			// Use 98% of max to leave buffer for price fluctuation
			adjustedSize := maxAffordablePositionSize * 0.98
			logger.Infof("  ⚠️ Position size %.2f exceeds max affordable %.2f, auto-reducing to %.2f",
				actualPositionSize, maxAffordablePositionSize, adjustedSize)
			actualPositionSize = adjustedSize
			decision.PositionSizeUSD = actualPositionSize
		}

		// [CODE ENFORCED] Minimum position size check
		if err := at.enforceMinPositionSize(decision.PositionSizeUSD); err != nil {
			return err
		}

		// Calculate quantity with adjusted position size
		quantity = actualPositionSize / marketData.CurrentPrice
	}
	actionRecord.Quantity = quantity
	actionRecord.Price = marketData.CurrentPrice

	// Set margin mode
	if err := at.trader.SetMarginMode(decision.Symbol, at.config.IsCrossMargin); err != nil {
		logger.Infof("  ⚠️ Failed to set margin mode: %v", err)
		// Continue execution, doesn't affect trading
	}

	// CME futures (NT8) require SL/TP set BEFORE the entry — the AddOn places
	// the market entry + protective OCO bracket atomically from the signal,
	// which carries SL/TP. (Crypto sets them after the fill, below.) Without
	// this, placeEntry errors "SetStopLoss and SetTakeProfit must be called
	// before long".
	if market.IsCMEFuturesSymbol(decision.Symbol) {
		_ = at.trader.SetStopLoss(decision.Symbol, "LONG", quantity, decision.StopLoss)
		_ = at.trader.SetTakeProfit(decision.Symbol, "LONG", quantity, decision.TakeProfit)
	}

	// Open position
	order, err := at.trader.OpenLong(decision.Symbol, quantity, decision.Leverage)
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	logger.Infof("  ✓ Position opened successfully, order ID: %v, quantity: %.4f", order["orderId"], quantity)

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "open_long", quantity, marketData.CurrentPrice, decision.Leverage, 0)

	// Record position opening time
	posKey := decision.Symbol + "_long"
	at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()

	// Set stop loss and take profit
	if err := at.trader.SetStopLoss(decision.Symbol, "LONG", quantity, decision.StopLoss); err != nil {
		logger.Infof("  ⚠ Failed to set stop loss: %v", err)
	}
	if err := at.trader.SetTakeProfit(decision.Symbol, "LONG", quantity, decision.TakeProfit); err != nil {
		logger.Infof("  ⚠ Failed to set take profit: %v", err)
	}

	return nil
}

// executeOpenShortWithRecord executes open short position and records detailed information
func (at *AutoTrader) executeOpenShortWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  📉 Open short: %s", decision.Symbol)

	// TRACK B — reconcile NT8 net before opening; flatten an orphan first or refuse.
	if err := at.reconcileBeforeOpenNT(decision.Symbol, "short"); err != nil {
		return err
	}

	// ⚠️ Get current positions for multiple checks
	positions, err := at.trader.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// [CODE ENFORCED] Check max positions limit
	if err := at.enforceMaxPositions(len(positions)); err != nil {
		return err
	}

	// Check if there's already a position in the same symbol and direction
	for _, pos := range positions {
		if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
			return fmt.Errorf("❌ %s already has short position, close it first", decision.Symbol)
		}
	}

	// Get current price
	marketData, err := market.GetWithExchange(decision.Symbol, at.exchange)
	if err != nil {
		return err
	}

	// Get balance (needed for multiple checks)
	balance, err := at.trader.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}
	availableBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}

	// Get equity for position value ratio check
	equity := 0.0
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		equity = eq
	} else if eq, ok := balance["totalWalletBalance"].(float64); ok && eq > 0 {
		equity = eq
	} else {
		equity = availableBalance // Fallback to available balance
	}

	// [CODE ENFORCED] Position Value Ratio Check: position_value <= equity × ratio
	adjustedPositionSize, wasCapped := at.enforcePositionValueRatio(decision.PositionSizeUSD, equity, decision.Symbol)
	if wasCapped {
		decision.PositionSizeUSD = adjustedPositionSize
	}

	// Calculate order quantity.
	var quantity float64
	if market.IsCMEFuturesSymbol(decision.Symbol) {
		// CME futures: size in contracts (notional / (price × point value)),
		// clamped. Skip the crypto notional/leverage margin model — futures
		// margin is per-contract, not notional/leverage.
		quantity = futuresOrderQuantity(decision.Symbol, decision.PositionSizeUSD, marketData.CurrentPrice, at.resolveMaxContracts())
	} else {
		// ⚠️ Auto-adjust position size if insufficient margin (crypto)
		// Formula: totalRequired = positionSize/leverage + positionSize*0.001 + positionSize/leverage*0.01
		//        = positionSize * (1.01/leverage + 0.001)
		marginFactor := 1.01/float64(decision.Leverage) + 0.001
		maxAffordablePositionSize := availableBalance / marginFactor

		actualPositionSize := decision.PositionSizeUSD
		if actualPositionSize > maxAffordablePositionSize {
			// Use 98% of max to leave buffer for price fluctuation
			adjustedSize := maxAffordablePositionSize * 0.98
			logger.Infof("  ⚠️ Position size %.2f exceeds max affordable %.2f, auto-reducing to %.2f",
				actualPositionSize, maxAffordablePositionSize, adjustedSize)
			actualPositionSize = adjustedSize
			decision.PositionSizeUSD = actualPositionSize
		}

		// [CODE ENFORCED] Minimum position size check
		if err := at.enforceMinPositionSize(decision.PositionSizeUSD); err != nil {
			return err
		}

		// Calculate quantity with adjusted position size
		quantity = actualPositionSize / marketData.CurrentPrice
	}
	actionRecord.Quantity = quantity
	actionRecord.Price = marketData.CurrentPrice

	// Set margin mode
	if err := at.trader.SetMarginMode(decision.Symbol, at.config.IsCrossMargin); err != nil {
		logger.Infof("  ⚠️ Failed to set margin mode: %v", err)
		// Continue execution, doesn't affect trading
	}

	// CME futures (NT8) require SL/TP set BEFORE the entry — the AddOn places
	// the market entry + protective OCO bracket atomically from the signal,
	// which carries SL/TP. (Crypto sets them after the fill, below.) Without
	// this, placeEntry errors "SetStopLoss and SetTakeProfit must be called
	// before short".
	if market.IsCMEFuturesSymbol(decision.Symbol) {
		_ = at.trader.SetStopLoss(decision.Symbol, "SHORT", quantity, decision.StopLoss)
		_ = at.trader.SetTakeProfit(decision.Symbol, "SHORT", quantity, decision.TakeProfit)
	}

	// Open position
	order, err := at.trader.OpenShort(decision.Symbol, quantity, decision.Leverage)
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	logger.Infof("  ✓ Position opened successfully, order ID: %v, quantity: %.4f", order["orderId"], quantity)

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "open_short", quantity, marketData.CurrentPrice, decision.Leverage, 0)

	// Record position opening time
	posKey := decision.Symbol + "_short"
	at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()

	// Set stop loss and take profit
	if err := at.trader.SetStopLoss(decision.Symbol, "SHORT", quantity, decision.StopLoss); err != nil {
		logger.Infof("  ⚠ Failed to set stop loss: %v", err)
	}
	if err := at.trader.SetTakeProfit(decision.Symbol, "SHORT", quantity, decision.TakeProfit); err != nil {
		logger.Infof("  ⚠ Failed to set take profit: %v", err)
	}

	return nil
}

// executeCloseLongWithRecord executes close long position and records detailed information
func (at *AutoTrader) executeCloseLongWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  🔄 Close long: %s", decision.Symbol)

	// Get current price
	marketData, err := market.GetWithExchange(decision.Symbol, at.exchange)
	if err != nil {
		return err
	}
	actionRecord.Price = marketData.CurrentPrice

	// Normalize symbol for database lookup
	normalizedSymbol := market.Normalize(decision.Symbol)

	// Get entry price and quantity - prioritize local database for accurate quantity
	var entryPrice float64
	var quantity float64

	// First try to get from local database (more accurate for quantity)
	if at.store != nil {
		if openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, normalizedSymbol, "LONG"); err == nil && openPos != nil {
			quantity = openPos.Quantity
			entryPrice = openPos.EntryPrice
			logger.Infof("  📊 Using local position data: qty=%.8f, entry=%.2f", quantity, entryPrice)
		}
	}

	// Fallback to exchange API if local data not found
	if quantity == 0 {
		positions, err := at.trader.GetPositions()
		if err == nil {
			for _, pos := range positions {
				if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
					if ep, ok := pos["entryPrice"].(float64); ok {
						entryPrice = ep
					}
					if amt, ok := pos["positionAmt"].(float64); ok && amt > 0 {
						quantity = amt
					}
					break
				}
			}
		}
		logger.Infof("  📊 Using exchange position data: qty=%.8f, entry=%.2f", quantity, entryPrice)
	}

	// Close position
	order, err := at.trader.CloseLong(decision.Symbol, 0) // 0 = close all
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "close_long", quantity, marketData.CurrentPrice, 0, entryPrice)

	logger.Infof("  ✓ Position closed successfully")
	return nil
}

// executeCloseShortWithRecord executes close short position and records detailed information
func (at *AutoTrader) executeCloseShortWithRecord(decision *kernel.Decision, actionRecord *store.DecisionAction) error {
	logger.Infof("  🔄 Close short: %s", decision.Symbol)

	// Get current price
	marketData, err := market.GetWithExchange(decision.Symbol, at.exchange)
	if err != nil {
		return err
	}
	actionRecord.Price = marketData.CurrentPrice

	// Normalize symbol for database lookup
	normalizedSymbol := market.Normalize(decision.Symbol)

	// Get entry price and quantity - prioritize local database for accurate quantity
	var entryPrice float64
	var quantity float64

	// First try to get from local database (more accurate for quantity)
	if at.store != nil {
		if openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, normalizedSymbol, "SHORT"); err == nil && openPos != nil {
			quantity = openPos.Quantity
			entryPrice = openPos.EntryPrice
			logger.Infof("  📊 Using local position data: qty=%.8f, entry=%.2f", quantity, entryPrice)
		}
	}

	// Fallback to exchange API if local data not found
	if quantity == 0 {
		positions, err := at.trader.GetPositions()
		if err == nil {
			for _, pos := range positions {
				if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
					if ep, ok := pos["entryPrice"].(float64); ok {
						entryPrice = ep
					}
					if amt, ok := pos["positionAmt"].(float64); ok {
						quantity = -amt // positionAmt is negative for short
					}
					break
				}
			}
		}
		logger.Infof("  📊 Using exchange position data: qty=%.8f, entry=%.2f", quantity, entryPrice)
	}

	// Close position
	order, err := at.trader.CloseShort(decision.Symbol, 0) // 0 = close all
	if err != nil {
		return err
	}

	// Record order ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	// Record order to database and poll for confirmation
	at.recordAndConfirmOrder(order, decision.Symbol, "close_short", quantity, marketData.CurrentPrice, 0, entryPrice)

	logger.Infof("  ✓ Position closed successfully")
	return nil
}
