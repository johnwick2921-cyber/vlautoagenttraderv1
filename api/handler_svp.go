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

	empty := kernel.SVPProfile{RowHeight: kernel.SVPRowHeight, Sessions: []kernel.SVPSession{}}

	if !market.IsCMEFuturesSymbol(symbol) {
		c.JSON(http.StatusOK, empty)
		return
	}
	provider := market.FuturesBarsProvider
	if provider == nil {
		c.JSON(http.StatusOK, empty)
		return
	}
	// Profile the SAME timeframe the chart is displaying so the SVP covers the
	// same visible range (like TradingView). A 5m chart spans ~5+ days → several
	// session profiles; a 1m chart spans ~1 day. Default 5m. We pull up to 2000
	// bars (the cache cap) so more historical sessions are available; sessions
	// that fall off the visible chart are simply skipped by the renderer.
	interval := c.Query("interval")
	if interval == "" {
		interval = "5m"
	}
	bars := provider(symbol, interval, 2000)
	if len(bars) == 0 {
		c.JSON(http.StatusOK, empty)
		return
	}

	c.JSON(http.StatusOK, kernel.BuildSVPProfile(bars, time.Now()))
}
