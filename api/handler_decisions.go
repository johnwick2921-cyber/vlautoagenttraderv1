// Plan 4 Task 23 — decision audit trail endpoint.
//
// GET /api/audit/decisions?trader_id=X&since=YYYY-MM-DD&limit=N
//
// Returns []store.DecisionRecord JSON ordered by timestamp DESC. The
// extended struct fields (PromptVersion, AIModel, AILatencyMs, RiskCheck*,
// ExecutionStatus, FillPrice, FillLatencyMs) round-trip through this
// endpoint so the UI Decisions tab can render the audit columns.

package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// handleDecisionAudit handles GET /api/audit/decisions.
//
// Query parameters:
//
//	trader_id (required) — EXACT trader_id from GET /api/my-traders
//	since     (optional) — YYYY-MM-DD, defaults to 7 days ago (UTC)
//	limit     (optional) — default 100, max 1000 (silently capped)
func (s *Server) handleDecisionAudit(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}

	// Parse since (default 7 days ago UTC).
	since := time.Now().UTC().AddDate(0, 0, -7)
	if v := strings.TrimSpace(c.Query("since")); v != "" {
		// Accept YYYY-MM-DD or RFC3339.
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			since = parsed.UTC()
		} else if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			since = parsed.UTC()
		} else {
			SafeBadRequest(c, "invalid since (use YYYY-MM-DD or RFC3339)")
			return
		}
	}

	// Parse limit (default 100, silently capped at 1000).
	limit := 100
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		SafeNotFound(c, "Trader")
		return
	}

	// Scope to the selected account when provided (mirrors handleStatistics);
	// empty → trader-global so crypto + legacy callers are unaffected.
	records, err := trader.GetStore().Decision().GetAuditRecords(traderID, since, limit, c.Query("account"))
	if err != nil {
		SafeInternalError(c, "Get decision audit records", err)
		return
	}

	c.JSON(http.StatusOK, records)
}
