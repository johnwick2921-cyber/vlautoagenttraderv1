package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// openPositionProvenance describes the OPEN position in the terms it was
// ENTERED under, never the terms currently on screen (invalidation-wired F2,
// owner ruling 2026-09-03).
//
// 2026-09-03: the plan card rendered NY v3 S1 long, written 09:15, while the
// account held a position armed under v2 S1 short — filled 09:03, stopped
// 09:20. Both are "S1". The card showed one and the owner was holding the
// other.
//
// nil when flat: the card then has nothing to disambiguate and renders the
// live plan alone, as before.
func (s *Server) openPositionProvenance(traderID string, livePlanVersion int) gin.H {
	if s.store == nil || strings.TrimSpace(traderID) == "" {
		return nil
	}
	opens, err := s.store.Position().GetOpenPositions(traderID)
	if err != nil || len(opens) == 0 {
		return nil
	}
	p := opens[0]
	out := gin.H{
		"symbol":            p.Symbol,
		"side":              p.Side,
		"entry_price":       p.EntryPrice,
		"quantity":          p.Quantity,
		"cited_scenario_id": p.CitedScenarioID,
		"live_plan_version": livePlanVersion,
		// True when the plan on screen is NOT the plan this position was
		// entered under — the card raises the second line only then.
		"version_differs": p.PlanVersion > 0 && livePlanVersion > 0 && p.PlanVersion != livePlanVersion,
	}
	// PlanVersion is stamped at OPEN from the arm's own version (S3
	// SetPlanLinkFull). Zero means it was never stamped — a pre-attribution
	// row — and the card must say so rather than render "v0".
	if p.PlanVersion > 0 {
		out["armed_under_version"] = p.PlanVersion
	} else {
		out["armed_under_version"] = nil
		out["armed_under_note"] = "version not recorded"
	}
	return out
}
