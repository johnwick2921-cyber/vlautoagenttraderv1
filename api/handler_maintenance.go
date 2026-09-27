package api

import (
	"net/http"

	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// ── W-ONE-BUTTON M2 — the READ-ONLY maintenance surfaces ───────────────────
//
// There is deliberately no write route: the installation hold is written only
// by the operator CLI (and, from M4, the updater worker). No API route may
// write or clear it — pinned by store.TestOnlyTheOperatorCLIWritesTheMaintenanceHold.

func (s *Server) loadedTraders() map[string]*trader.AutoTrader {
	if s.traderManager == nil {
		return nil
	}
	return s.traderManager.GetAllTraders()
}

// handleMaintenanceStatus — GET /api/maintenance.
func (s *Server) handleMaintenanceStatus(c *gin.Context) {
	c.JSON(http.StatusOK, trader.MaintenanceStatus(s.loadedTraders()))
}

// handleInstallationGate — GET /api/installation-gate.
func (s *Server) handleInstallationGate(c *gin.Context) {
	c.JSON(http.StatusOK, trader.InstallationGateStatus(s.loadedTraders(), s.store))
}
