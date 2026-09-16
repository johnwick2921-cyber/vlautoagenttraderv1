package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// handleArmedTestArm POST /api/armed/test-arm — E2 DEBUG SEAM (2026-08-27).
// Drives the REAL armed placement path with a TEST-E2 ledger row so the
// armed-orders cutover can be proven end-to-end (place → working in NT8 →
// cancel → cancelled chain) even when the planner cannot produce an active
// plan. Defended on BOTH sides: this handler (env gate + ownership) and the
// AutoTrader methods (env gate + SIM-only). Default state is OFF.
func (s *Server) handleArmedTestArm(c *gin.Context) {
	var body struct {
		TraderID string  `json:"trader_id"`
		Action   string  `json:"action"` // place | cancel
		Side     string  `json:"side"`
		Entry    float64 `json:"entry"`
		Stop     float64 `json:"stop"`
		Target   float64 `json:"target"`
		SignalID string  `json:"signal_id"`
	}
	_ = c.ShouldBindJSON(&body)
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		traderID = strings.TrimSpace(body.TraderID)
	}
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	// Ownership: the caller must own the trader (the owner's user id).
	if !s.traderOwnedBy(c.GetString("user_id"), traderID) {
		SafeUnauthorized(c)
		return
	}
	at, err := s.traderManager.GetTrader(traderID)
	if err != nil || at == nil {
		SafeNotFound(c, "Trader")
		return
	}

	switch strings.ToLower(strings.TrimSpace(body.Action)) {
	case "place":
		row, perr := at.TestArmPlace(body.Side, body.Entry, body.Stop, body.Target)
		if perr != nil {
			c.JSON(409, gin.H{"error": perr.Error()})
			return
		}
		c.JSON(200, gin.H{
			"ok": true, "action": "place",
			"armed_order": gin.H{
				"id": row.ID, "trader_id": row.TraderID, "plan_id": row.PlanID,
				"scenario": row.Scenario, "side": row.Side,
				"entry": row.EntryPx, "stop": row.StopPx, "target": row.TargetPx,
				"state": row.State, "signal_id": row.SignalID,
			},
		})
	case "place_stop":
		// E7 far-side proof: stop-market entry on the REAL wire (PlaceStopEntry).
		row, perr := at.TestArmPlaceStop(body.Side, body.Entry, body.Stop, body.Target)
		if perr != nil {
			c.JSON(409, gin.H{"error": perr.Error()})
			return
		}
		c.JSON(200, gin.H{
			"ok": true, "action": "place_stop",
			"armed_order": gin.H{
				"id": row.ID, "trader_id": row.TraderID, "plan_id": row.PlanID,
				"scenario": row.Scenario, "side": row.Side,
				"entry": row.EntryPx, "stop": row.StopPx, "target": row.TargetPx,
				"state": row.State, "signal_id": row.SignalID,
			},
		})
	case "cancel":
		if cerr := at.TestArmCancel(strings.TrimSpace(body.SignalID)); cerr != nil {
			c.JSON(409, gin.H{"error": cerr.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true, "action": "cancel", "signal_id": strings.TrimSpace(body.SignalID)})
	default:
		SafeBadRequest(c, "action must be place|place_stop|cancel")
	}
}
