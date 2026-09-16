package api

import (
	"github.com/gin-gonic/gin"

	"nofx/store"
)

// handleStreamCuts serves the idle-before-vs-outcome table (owner ruling
// 2026-09-03). Read-only: it reads the record and computes nothing that any
// trading path consults.
//
// The question it exists to answer: do peer-side stream cuts cluster above some
// connection-idle threshold? If they do, setting IdleConnTimeout below it is
// the entire fix and needs nothing from the provider. NOTHING IS SET — the
// ruling was three more cuts before deciding, and the payload says so.
func (s *Server) handleStreamCuts(c *gin.Context) {
	if s.store == nil || s.store.WatchdogFires() == nil {
		c.JSON(200, gin.H{"rows": []any{}, "table": "", "note": "no store"})
		return
	}
	rows, err := s.store.WatchdogFires().IdleOutcomeTable()
	if err != nil {
		SafeInternalError(c, "failed to read the stream-cut record", err)
		return
	}
	c.JSON(200, gin.H{
		"rows":  rows,
		"table": store.RenderIdleOutcomeTable(rows),
		"note":  "No threshold is set. IdleConnTimeout is untouched until the evidence is in; an unresolved resend is counted as unresolved, never as a loss, and a connection that was not reused has its own bucket so fresh dials never read as evidence about idleness.",
	})
}
