// B6 — gate-block counters endpoint.
//
// Exposes the per-trader, per-CME-session-day tally of how many times each named
// risk/safety gate blocked an entry or decision cycle today (feed_down, dead_man,
// consecutive_loss, price_sanity, stale_data, the Strategy-Studio guardrails, the
// B3 order guard, …). This is the queryable companion to the global Prometheus
// RiskGateTrips counter and the source of the daily journal summary line.
//
//	GET /api/risk/gate-blocks
//	Returns: {"session_day_utc":"<RFC3339>","summary":"<one-line>",
//	          "by_trader":{"<trader_id>":{"<gate>":<count>}}}
//
// The empty-string trader key ("") holds process-wide gates that fire at a shared
// chokepoint with no per-trader identity (the B3 order guard).

package api

import (
	"net/http"
	"time"

	"nofx/telemetry"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleGateBlocks(c *gin.Context) {
	dayMs, table := telemetry.GateBlockSnapshot()
	sessionDayUTC := ""
	if dayMs > 0 {
		sessionDayUTC = time.UnixMilli(dayMs).UTC().Format(time.RFC3339)
	}
	if table == nil {
		table = map[string]map[string]int{}
	}
	c.JSON(http.StatusOK, gin.H{
		"session_day_utc": sessionDayUTC,
		"summary":         telemetry.GateBlockDailySummary(),
		"by_trader":       table,
	})
}

// P0-cleanup (2026-08-19) — the ONE error surface: structured error events
// (type / cause / cost) per trader per session-day, plus the daily summary
// line. Mirrors the gate-blocks endpoint.
//
//	GET /api/risk/errors
//	Returns: {"rows":[{trader,type,cause,cost,count,decisions_lost,trades_lost}],
//	          "summary":"errors today: N (types: …), decisions lost: N"}
func (s *Server) handleErrors(c *gin.Context) {
	trader := c.Query("trader_id")
	rows := telemetry.ErrorSummary(trader)
	if rows == nil {
		rows = []telemetry.ErrorEventRow{}
	}
	c.JSON(http.StatusOK, gin.H{
		"rows":    rows,
		"summary": telemetry.ErrorDigestLine(trader),
	})
}
