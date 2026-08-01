package api

import (
	"net/http"
	"time"

	"nofx/kernel"
	"nofx/market"

	"github.com/gin-gonic/gin"
)

// handleKlinesSVP serves the server-computed Session Volume Profile (SVP) for a
// CME futures symbol: POC / VAH / VAL and the histogram bins for the developing
// and prior RTH sessions.
//
// ONE source of truth: it reads the SAME live NT8 1-minute bars the AI kernel
// uses (market.FuturesBarsProvider) and runs the SAME kernel.BuildSVPProfile
// engine — the chart never recomputes the profile itself. The SVP is always
// built from 1m bars regardless of the chart's display interval, so the ?interval
// query param is intentionally ignored here.
//
// A cold/empty cache, an unbound provider, or a non-futures symbol returns a
// well-formed zero-value 200 (empty bins) — NEVER a 500 and never a crypto
// fallthrough — mirroring getKlinesFromNinjaTrader so the chart degrades to
// "no profile" gracefully.
func (s *Server) handleKlinesSVP(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		symbol = "MNQ"
	}

	empty := kernel.SVPProfile{RowHeight: kernel.SVPRowHeight}

	if !market.IsCMEFuturesSymbol(symbol) {
		c.JSON(http.StatusOK, empty)
		return
	}
	provider := market.FuturesBarsProvider
	if provider == nil {
		c.JSON(http.StatusOK, empty)
		return
	}
	bars := provider(symbol, "1m", 2000) // profile granularity = 1m
	if len(bars) == 0 {
		c.JSON(http.StatusOK, empty)
		return
	}

	c.JSON(http.StatusOK, kernel.BuildSVPProfile(bars, time.Now()))
}
