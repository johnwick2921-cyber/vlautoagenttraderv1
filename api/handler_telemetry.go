package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nofx/telemetry"
)

// handleTelemetry (P2-8) serves the process-lifetime telemetry counters that
// previously had writers but no readers: far arms, repair regression, shadowed
// arm refusals, research-snapshot drops/rows, weekly counters, GA4 send
// failures, and the log-event sink's queue-full/write-failure drops.
// READ-ONLY; every value is read from the live counter, never inferred.
func (s *Server) handleTelemetry(c *gin.Context) {
	snap := telemetry.Snapshot()
	if s.store != nil {
		full, writes := s.store.LogEvent().Dropped()
		snap["log_event_dropped"] = map[string]any{
			"queue_full":  full,
			"write_fails": writes,
		}
	}
	c.JSON(http.StatusOK, snap)
}
