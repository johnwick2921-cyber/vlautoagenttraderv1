package api

import (
	"net/http"

	"vl/logger"
	ntTrader "vl/trader/ninjatrader"

	"github.com/gin-gonic/gin"
)

// handleGetGridRiskInfo returns current risk information for a grid trader
func (s *Server) handleGetGridRiskInfo(c *gin.Context) {
	traderID := c.Param("id")

	autoTrader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}

	riskInfo := autoTrader.GetGridRiskInfo()
	c.JSON(http.StatusOK, riskInfo)
}

// handleSyncBalance Sync exchange balance to initial_balance (Option B: Manual Sync + Option C: Smart Detection)
func (s *Server) handleSyncBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	logger.Infof("🔄 User %s requested balance sync for trader %s", userID, traderID)

	// Get trader configuration from database (including exchange info)
	fullConfig, err := s.store.Trader().GetFullConfig(userID, traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trader does not exist"})
		return
	}

	traderConfig := fullConfig.Trader
	exchangeCfg := fullConfig.Exchange

	if exchangeCfg == nil || !exchangeCfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange not configured or not enabled"})
		return
	}

	tempTrader, createErr := buildExchangeProbeTrader(exchangeCfg, userID)
	if createErr != nil {
		logger.Infof("⚠️ Failed to create temporary trader: %v", createErr)
		SafeInternalError(c, "Failed to connect to exchange", createErr)
		return
	}

	// Query actual balance
	balanceInfo, balanceErr := tempTrader.GetBalance()
	if balanceErr != nil {
		logger.Infof("⚠️ Failed to query exchange balance: %v", balanceErr)
		SafeInternalError(c, "Failed to query balance", balanceErr)
		return
	}

	// Extract total equity (for P&L calculation, we need total account value, not available balance)
	actualBalance, found := extractExchangeTotalEquity(balanceInfo)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get total equity"})
		return
	}

	s.exchangeAccountStateCache.Invalidate(userID)

	oldBalance := traderConfig.InitialBalance

	// Smart balance change detection
	changePercent := ((actualBalance - oldBalance) / oldBalance) * 100
	changeType := "increase"
	if changePercent < 0 {
		changeType = "decrease"
	}

	logger.Infof("✓ Queried actual exchange balance: %.2f (current config: %.2f, change: %.2f%%)",
		actualBalance, oldBalance, changePercent)

	// Update initial_balance in database
	err = s.store.Trader().UpdateInitialBalance(userID, traderID, actualBalance)
	if err != nil {
		logger.Infof("❌ Failed to update initial_balance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
		return
	}

	// Reload traders into memory
	err = s.traderManager.LoadUserTradersFromStore(s.store, userID)
	if err != nil {
		logger.Infof("⚠️ Failed to reload user traders into memory: %v", err)
	}

	logger.Infof("✅ Synced balance: %.2f → %.2f (%s %.2f%%)", oldBalance, actualBalance, changeType, changePercent)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Balance synced successfully",
		"old_balance":    oldBalance,
		"new_balance":    actualBalance,
		"change_percent": changePercent,
		"change_type":    changeType,
	})
}

// handleClosePosition One-click close position
func (s *Server) handleClosePosition(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	var req struct {
		Symbol string `json:"symbol" binding:"required"`
		Side   string `json:"side" binding:"required"` // "LONG" or "SHORT"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter error: symbol and side are required"})
		return
	}

	logger.Infof("🔻 User %s requested position close: trader=%s, symbol=%s, side=%s", userID, traderID, req.Symbol, req.Side)

	// Get trader configuration from database (including exchange info)
	fullConfig, err := s.store.Trader().GetFullConfig(userID, traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trader does not exist"})
		return
	}

	exchangeCfg := fullConfig.Exchange

	if exchangeCfg == nil || !exchangeCfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange not configured or not enabled"})
		return
	}

	// NinjaTrader: route to the LIVE running trader (bound to the active SIM
	// account) and reuse the proven TCP flatten path, exactly as Emergency Flat
	// does (handler_risk.go). The key-built tempTrader switch below has no
	// ninjatrader case (it would 400) and could not reach the live TCPTrader
	// anyway. Per-position close passes the symbol; quantity 0 = close all.
	if exchangeCfg.ExchangeType == "ninjatrader" {
		at, gtErr := s.traderManager.GetTrader(traderID)
		if gtErr != nil {
			SafeNotFound(c, "Trader")
			return
		}
		ntTCP, ok := at.GetUnderlyingTrader().(*ntTrader.TCPTrader)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "NinjaTrader manual close requires the running TCP bridge (NT_TRANSPORT=tcp); CSV bridge closes via SL/TP only"})
			return
		}
		var ntResult map[string]interface{}
		var ntErr error
		switch req.Side {
		case "LONG":
			ntResult, ntErr = ntTCP.CloseLong(req.Symbol, 0)
		case "SHORT":
			ntResult, ntErr = ntTCP.CloseShort(req.Symbol, 0)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "side must be LONG or SHORT"})
			return
		}
		if ntErr != nil {
			logger.Infof("❌ NT close failed: symbol=%s, side=%s, error=%v", req.Symbol, req.Side, ntErr)
			SafeInternalError(c, "Close position", ntErr)
			return
		}
		logger.Infof("✅ NT manual close sent: symbol=%s, side=%s", req.Symbol, req.Side)
		c.JSON(http.StatusOK, gin.H{"message": "Position close sent to NinjaTrader", "result": ntResult})
		return
	}

	// After the crypto removal there is no broker left to close against here —
	// the NinjaTrader path above handles the futures venue. Any other type is
	// a legacy stored row that must not trade.
	c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported exchange type"})
	return
}

// getSideFromAction Get order side (BUY/SELL) from order action
func getSideFromAction(action string) string {
	switch action {
	case "open_long", "close_short":
		return "BUY"
	case "open_short", "close_long":
		return "SELL"
	default:
		return "BUY"
	}
}
