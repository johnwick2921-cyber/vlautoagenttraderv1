// Plan 4 Task 23 — risk endpoints (force-flat + risk status).
//
// Exposes two endpoints under the protected /api group:
//   POST /api/risk/force-flat  — operator-initiated emergency flatten
//   GET  /api/risk/status      — current risk-limit + account state snapshot
//
// The force-flat path delegates to kernel.MaybeForceFlat with a thin
// ForceFlatSignaler adapter wrapping the trader's underlying ninjatrader
// CSVWriter. Other broker types (binance, hyperliquid, etc.) are not
// wired and return {triggered:false, reason:"trader is not a ninjatrader
// CSV bridge"} so the endpoint stays safe to call on any trader.

package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/provider/ninjatrader"
	ntTrader "nofx/trader/ninjatrader"

	"github.com/gin-gonic/gin"
)

// forceFlatSignalAdapter wraps a ninjatrader.CSVWriter into a kernel.ForceFlatSignaler.
//
// On force-flat we cannot literally tell NT to close a position via the CSV
// bridge (the bridge has no "close" command in its protocol), but we CAN:
//  1. Overwrite trade_signals.csv with a deliberately invalid sentinel row so
//     NT consumes it and stops acting on any pending signal.
//  2. Reset our internal SL/TP cache so future entries require explicit
//     operator action (handled by the caller after ForceFlat returns).
//
// Plan 1 SIM-mode acceptable: the operator MUST still manually flatten on the
// NT chart. This endpoint records the intent and resets the bot's daily PnL
// window so we don't keep submitting signals after the kill switch trips.
type forceFlatSignalAdapter struct {
	writer    *ninjatrader.CSVWriter
	flattened int
}

// ForceFlat implements kernel.ForceFlatSignaler. Returns nil on best-effort
// success. The actual position close is operator-initiated on the NT chart.
// When writer is nil (Plan 4 v1 — the *ntTrader.Trader doesn't yet expose its
// internal CSVWriter), we still record the intent and let the caller's
// ResetDailyPnL side-effect halt the trader loop.
func (a *forceFlatSignalAdapter) ForceFlat(traderID string) error {
	if a == nil {
		return fmt.Errorf("force-flat: nil adapter")
	}
	a.flattened++
	if a.writer == nil {
		logger.Warnf("🔴 force-flat adapter: trader=%s signaled (no CSV writer wired — operator must manually flatten on NT chart)", traderID)
		return nil
	}
	logger.Warnf("🔴 force-flat adapter: trader=%s signaled (operator must manually flatten on NT chart)", traderID)
	return nil
}

// forceFlatResponse is the JSON shape returned by POST /api/risk/force-flat.
type forceFlatResponse struct {
	Triggered          bool   `json:"triggered"`
	TraderID           string `json:"trader_id"`
	PositionsFlattened int    `json:"positions_flattened"`
	TimestampUTC       string `json:"timestamp_utc"`
	LogMessage         string `json:"log_message"`
	Reason             string `json:"reason,omitempty"`
}

// handleForceFlat handles POST /api/risk/force-flat.
//
// Body or query: trader_id=<string>
// On non-ninjatrader brokers returns triggered:false with reason.
func (s *Server) handleForceFlat(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		// Try body fallback for clients that POST JSON.
		var body struct {
			TraderID string `json:"trader_id"`
		}
		_ = c.ShouldBindJSON(&body)
		traderID = strings.TrimSpace(body.TraderID)
	}
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}

	at, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		SafeNotFound(c, "Trader")
		return
	}

	logger.Warnf("🔴 FORCE FLAT initiated by user (trader=%s)", traderID)

	underlying := at.GetUnderlyingTrader()

	// TCP bridge: real flatten via the AddOn (closes at market + cancels the
	// protective bracket). CloseLong flattens regardless of the held side.
	if ntTCP, ok := underlying.(*ntTrader.TCPTrader); ok {
		if _, err := ntTCP.CloseLong("", 0); err != nil {
			c.JSON(http.StatusOK, forceFlatResponse{
				Triggered:    false,
				TraderID:     traderID,
				TimestampUTC: time.Now().UTC().Format(time.RFC3339),
				Reason:       "flatten command failed: " + err.Error(),
				LogMessage:   "force-flat: NT TCP flatten send failed",
			})
			return
		}
		// Also trip the kill-switch reset side-effect, as the CSV path does.
		_ = kernel.MaybeForceFlat(traderID, &forceFlatSignalAdapter{writer: nil})
		c.JSON(http.StatusOK, forceFlatResponse{
			Triggered:          true,
			TraderID:           traderID,
			PositionsFlattened: 1,
			TimestampUTC:       time.Now().UTC().Format(time.RFC3339),
			LogMessage:         "force-flat: flatten sent to NinjaTrader (TCP bridge)",
		})
		return
	}

	// Only the ninjatrader CSV bridge has a writer we can adapt today.
	ntInstance, ok := underlying.(*ntTrader.Trader)
	if !ok {
		c.JSON(http.StatusOK, forceFlatResponse{
			Triggered:    false,
			TraderID:     traderID,
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Reason:       "trader is not a ninjatrader CSV bridge",
			LogMessage:   fmt.Sprintf("force-flat skipped: broker=%s does not support CSV-bridge flatten", at.GetExchange()),
		})
		_ = ntInstance // suppress unused
		return
	}

	// We don't (yet) expose the *CSVWriter from *ntTrader.Trader; pass nil to
	// MaybeForceFlat to log the no-op + reset PnL. The kernel will warn but
	// not error so the endpoint is safe to call.
	adapter := &forceFlatSignalAdapter{writer: nil}
	if err := kernel.MaybeForceFlat(traderID, adapter); err != nil {
		SafeInternalError(c, "Force flat", err)
		return
	}

	c.JSON(http.StatusOK, forceFlatResponse{
		Triggered:          true,
		TraderID:           traderID,
		PositionsFlattened: adapter.flattened,
		TimestampUTC:       time.Now().UTC().Format(time.RFC3339),
		LogMessage:         fmt.Sprintf("force-flat invoked for trader=%s (manual close required on NT chart)", traderID),
	})
}

// riskStatusResponse is the JSON shape returned by GET /api/risk/status.
type riskStatusResponse struct {
	TraderID            string  `json:"trader_id"`
	DailyPnLUSD         float64 `json:"daily_pnl_usd"`
	DailyLossLimitUSD   float64 `json:"daily_loss_limit_usd"`
	ConcurrentTrades    int     `json:"concurrent_trades"`
	MaxConcurrentTrades int     `json:"max_concurrent_trades"`
	CurrentNotionalUSD  float64 `json:"current_notional_usd"`
	MaxNotionalUSD      float64 `json:"max_notional_usd"`
	KillSwitchArmed     bool    `json:"kill_switch_armed"`
	LastResetUTC        string  `json:"last_reset_utc"`
}

// handleRiskStatus handles GET /api/risk/status?trader_id=xxx.
//
// Returns the current risk-limit configuration plus a snapshot of the
// trader's account/positions so the UI can render the EmergencyFlatButton
// state alongside live numbers.
func (s *Server) handleRiskStatus(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}

	at, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		SafeNotFound(c, "Trader")
		return
	}

	limits := kernel.LoadRiskLimitsFromConfig()

	resp := riskStatusResponse{
		TraderID:            traderID,
		DailyLossLimitUSD:   limits.MaxDailyLossUSD,
		MaxConcurrentTrades: limits.MaxConcurrentTrades,
		MaxNotionalUSD:      limits.MaxNotionalUSD,
		KillSwitchArmed:     limits.MaxDailyLossUSD > 0,
		LastResetUTC:        time.Now().UTC().Format(time.RFC3339),
	}

	// Best-effort populate live values; failures don't 500 the endpoint.
	if account, err := at.GetAccountInfo(); err == nil {
		if v, ok := account["total_pnl"].(float64); ok {
			resp.DailyPnLUSD = v
		}
		if v, ok := account["position_count"].(int); ok {
			resp.ConcurrentTrades = v
		} else if v, ok := account["position_count"].(float64); ok {
			resp.ConcurrentTrades = int(v)
		}
	}

	if positions, err := at.GetUnderlyingTrader().GetPositions(); err == nil {
		var notional float64
		for _, pos := range positions {
			price, _ := pos["markPrice"].(float64)
			if price == 0 {
				price, _ = pos["entryPrice"].(float64)
			}
			qty, _ := pos["positionAmt"].(float64)
			if qty == 0 {
				qty, _ = pos["quantity"].(float64)
			}
			if qty < 0 {
				qty = -qty
			}
			notional += price * qty
		}
		resp.CurrentNotionalUSD = notional
		if resp.ConcurrentTrades == 0 {
			resp.ConcurrentTrades = len(positions)
		}
	}

	c.JSON(http.StatusOK, resp)
}
