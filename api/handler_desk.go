package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── GET /api/desk — THE DESK STRIP (2026-09-06) ──────────────────────────────
//
// ONE request returns every row of the strip. The browser makes no second call
// and computes no value: the rows arrive rendered, with their source, their
// as-of instant and their age, so the screen cannot show a number the engine
// did not stand behind.
//
// READ-ONLY. This handler writes nothing to the trading store, touches no gate
// and changes nothing the bot does (A31).
//
// A10: a failure inside any single row is contained in trader.DeskStripAt and
// renders that row UNKNOWN with its reason. The endpoint answers 200 with a
// complete strip whenever it can reach the trader at all, because a blank
// screen and a screen full of UNKNOWNs mean very different things and only the
// second one is honest.
func (s *Server) handleDesk(c *gin.Context) {
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
	// The entry point owns the clock (A28/class 60); everything beneath takes it.
	c.JSON(http.StatusOK, at.DeskStripAt(time.Now()))
}
