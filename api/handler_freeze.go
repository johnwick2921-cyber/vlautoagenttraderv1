// A4 (G4) — trader FREEZE visibility + owner clear.
//
//	GET  /api/risk/freezes      → {"frozen":{"<trader_id>":"<reason>"}}
//	POST /api/risk/clear-freeze  ?trader_id=<id>  → {"trader_id","was_frozen","cleared"}
//
// A trader is frozen by an A2 identity/account echo mismatch, an A4 post-fill
// account mismatch, or an A4 reconcile belief≠broker divergence: NEW entries are
// blocked while open-position management continues. The owner clears it here after
// investigating; entries resume immediately.

package api

import (
	"net/http"

	"nofx/discipline"
	"nofx/logger"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleListFreezes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"frozen": discipline.FrozenSnapshot()})
}

func (s *Server) handleClearFreeze(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	cleared := discipline.ClearFreeze(traderID)
	if cleared {
		logger.Warnf("🔓 A4 freeze CLEARED by owner for trader %s — new entries may resume", traderID)
	}
	c.JSON(http.StatusOK, gin.H{
		"trader_id":  traderID,
		"was_frozen": cleared,
		"cleared":    cleared,
	})
}
