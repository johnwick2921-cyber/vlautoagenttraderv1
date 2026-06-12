package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	ntTrader "nofx/trader/ninjatrader"
)

// handleBarsStream streams live NT8 OHLCV bars to the FuturesChart over SSE
// (Plan 4.4 Stage 4). The chart consumes the SAME BarCache the kernel reads, so
// candles match decisions — one feed, no second-source divergence.
//
// Auth: EventSource cannot set an Authorization header, so the JWT rides in the
// ?token= query param (validated here, same as authMiddleware). The route is
// therefore public-group + self-authed.
//
// Relay strategy: POLL the BarCache (read-only Get) — no change to the proven
// Stage-2 bar receive path, no BarCache pub/sub. Sends a one-shot snapshot then
// ~1s incremental updates (re-send the in-progress bar + append new bars).
func (s *Server) handleBarsStream(c *gin.Context) {
	// Auth via a short-lived single-use ticket (minted by the protected
	// /v1/bars/stream-ticket endpoint) — the long-lived JWT never appears in
	// the URL. See stream_ticket.go.
	userID, ok := consumeStreamTicket(c.Query("ticket"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing, invalid, or expired stream ticket"})
		return
	}
	// Scope trader resolution to the caller, exactly like the protected routes
	// (authMiddleware sets user_id; this route self-auths so we set it here).
	c.Set("user_id", userID)

	symbol := c.DefaultQuery("symbol", "MNQ")
	tf := c.DefaultQuery("tf", "5m")

	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	nt, ok := trader.GetUnderlyingTrader().(*ntTrader.TCPTrader)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader is not an NT8 TCP trader"})
		return
	}
	cache := nt.BarCache()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	writeEvent := func(event string, v interface{}) bool {
		data, _ := json.Marshal(v)
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// One-shot snapshot of the current window.
	bars := cache.Get(symbol, tf)
	if !writeEvent("snapshot", gin.H{"symbol": symbol, "tf": tf, "bars": bars}) {
		return
	}
	var lastT int64
	if len(bars) > 0 {
		lastT = bars[len(bars)-1].T
	}

	ctx := c.Request.Context()
	poll := time.NewTicker(1 * time.Second)
	defer poll.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-poll.C:
			cur := cache.Get(symbol, tf)
			if len(cur) == 0 {
				continue
			}
			// Re-send the in-progress bar (t == lastT, updated OHLCV) and any
			// new bars (t > lastT). The chart's series.update() updates the
			// matching time or appends.
			for _, b := range cur {
				if b.T >= lastT {
					if !writeEvent("bar", b) {
						return
					}
				}
			}
			lastT = cur[len(cur)-1].T
		}
	}
}
