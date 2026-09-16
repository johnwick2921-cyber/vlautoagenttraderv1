package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"nofx/trader"
)

// handleAdminBarsImport POST /api/admin/bars/import — HISTORY IMPORT (wave 101).
// Pulls one NAMED contract's history for each requested timeframe through the
// live wire and writes it contract-stamped via the store importer. This is a
// maintenance path, not a trading path: it is env-gated
// (HISTORICAL_IMPORT_SEAM=on, default OFF) on top of the auth group, it only
// ever ADDS historical_import rows, and it never touches a detector, level,
// gate, arm or exit. A31: zero trading-path diffs.
func (s *Server) handleAdminBarsImport(c *gin.Context) {
	if !trader.HistoryImportSeamOn() {
		c.JSON(409, gin.H{"error": "historical import seam is OFF — set HISTORICAL_IMPORT_SEAM=on to write history (wave 101)"})
		return
	}
	var body struct {
		Symbol     string   `json:"symbol"`
		Contract   string   `json:"contract"`
		Timeframes []string `json:"timeframes"`
		FromMs     int64    `json:"from_ms"`
		ToMs       int64    `json:"to_ms"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		SafeBadRequest(c, "bad body: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Contract) == "" {
		SafeBadRequest(c, "contract is required and must be the explicit name (e.g. \"MNQ 09-23\") — importing by date inference is the 09-10 bug")
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(body.Symbol))
	if symbol == "" {
		symbol = "MNQ"
	}
	if len(body.Timeframes) == 0 {
		body.Timeframes = []string{"1m"}
	}
	ctx := c.Request.Context()
	results, err := trader.ImportNamedContractHistory(ctx, s.store, symbol, strings.TrimSpace(body.Contract), body.Timeframes, body.FromMs, body.ToMs)
	if err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"contract": body.Contract,
		"results":  results,
		"at_ms":    time.Now().UnixMilli(),
	})
}
