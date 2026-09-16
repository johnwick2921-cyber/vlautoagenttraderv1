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
// The SVP is a CHART indicator: it profiles WHATEVER candle data the chart is
// showing, at the chart's selected timeframe (?interval). More candles → more
// session-days (1m ≈ ~1.4 days, 5m ≈ ~7 days), exactly like a TradingView Session
// Volume Profile follows the chart. It reads live NT8 bars via
// market.FuturesBarsProvider and runs kernel.BuildSVPProfile — the chart never
// recomputes the profile client-side (so the dashboard and strategy-page charts
// stay identical at the same timeframe). The AI computes its OWN, independent SVP
// (always 1m — see kernel/engine_analysis.go) for its prompt line; that path is
// unaffected by this display interval.
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
	// Profile the chart's DISPLAY timeframe so the SVP covers the same candle data
	// the chart is showing (like TradingView): 1m → ~1.4 days, 5m → ~7 days, etc.
	// Default 5m. Pull up to 2000 bars — that is kernel.AISVPBarCount, the SVP
	// profile depth, and it is the CORRECT argument here. It is NOT "the cache
	// cap": the ring's capacity is provider/ninjatrader.DefaultBarCacheMaxBars
	// = 2500, a different number for a different purpose. (Class 105: this
	// comment said "the cache cap" from 2026-08-02 until 2026-09-10, and was
	// wrong from 2026-08-27, when the ring moved 1024 → 2500 and nothing could
	// compare the prose to the code.) Cleanup batch 2 (2026-09-11): the
	// literal IS the constant now — kernel.AISVPBarCount — so the two cannot
	// drift; api/handler_svp_test.go asserts no bare 2000 remains in this file.
	// Sessions off the visible range are skipped by the renderer.
	interval := c.Query("interval")
	if interval == "" {
		interval = "5m"
	}
	bars := provider(symbol, interval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		c.JSON(http.StatusOK, empty)
		return
	}

	c.JSON(http.StatusOK, kernel.BuildSVPProfile(bars, time.Now()))
}
