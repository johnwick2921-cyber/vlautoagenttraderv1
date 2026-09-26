// W-PICTURE-HTF (2026-09-20) — the two-picture opportunity ledger endpoint.
//
// GET /api/picture-htf/opportunities?trader_id=<id>&limit=<n>
// Returns the trader's opportunity rows, newest first: intended (entry/stop/
// target, R:R estimate, the rule's configured minimum) and the broker's
// answer side by side (signal, submitted_at, order id/status, fill price/qty,
// actual R:R, rejection reason) — plus the command age, the stage and the
// stage reason. Read-only: no row is ever written through this route.

package api

import (
	"net/http"
	"strings"

	"nofx/store"

	"github.com/gin-gonic/gin"
)

// pictureHtfRowDTO is the wire shape — snake_case, only what the dashboard
// renders. A missing broker answer stays zero/empty (unknown, never a
// fabricated success).
type pictureHtfRowDTO struct {
	OppKey        string  `json:"opp_key"`
	Stage         string  `json:"stage"`
	StageReason   string  `json:"stage_reason"`
	Symbol        string  `json:"symbol"`
	Direction     string  `json:"direction"`
	LevelRole     string  `json:"level_role"`
	LevelBodyTop  float64 `json:"level_body_top"`
	H1CloseTime   int64   `json:"h1_close_time_ms"`
	WindowClose   int64   `json:"window_close_ms"`
	EntryRef      float64 `json:"entry_ref"`
	StopPx        float64 `json:"stop_px"`
	TargetPx      float64 `json:"target_px"`
	RREstimate    float64 `json:"rr_estimate"`
	RRConfigured  float64 `json:"rr_configured"`
	MomentumStall bool    `json:"momentum_stall"`
	// The broker's answer.
	SignalID      string  `json:"signal_id"`
	SubmittedAt   int64   `json:"submitted_at_ms"`
	BrokerOrderID string  `json:"broker_order_id"`
	BrokerStatus  string  `json:"broker_status"`
	FillPrice     float64 `json:"fill_price"`
	FillQty       float64 `json:"fill_qty"`
	FillRR        float64 `json:"fill_rr"`
	RejectReason  string  `json:"reject_reason"`
	CreatedAt     int64   `json:"created_at_ms"`
	// W-EXEC-TRUTH W5 — where the opportunity went in the Day Plan: the
	// armed_orders row whose source_ref is this opp_key (a Picture scenario
	// P<n> of a plan). ABSENT when no such row exists — a legacy row, or a
	// scenario not yet armed — never an empty object.
	PlanLink *pictureHtfPlanLinkDTO `json:"plan_link,omitempty"`
}

// pictureHtfPlanLinkDTO is read from ONE armed_orders row (source_ref =
// opp_key): the plan chain it belongs to, the scenario id, the plan version
// that last authorized it, and the row's own id / state / broker signal.
type pictureHtfPlanLinkDTO struct {
	PlanID      string `json:"plan_id"`
	ScenarioID  string `json:"scenario_id"`
	PlanVersion int    `json:"plan_version"`
	ArmRowID    int64  `json:"arm_row_id"`
	ArmState    string `json:"arm_state"`
	ArmSignalID string `json:"arm_signal_id,omitempty"`
}

func pictureHtfRowToDTO(r store.PictureHtfOpportunityDB) pictureHtfRowDTO {
	return pictureHtfRowDTO{
		OppKey: r.OppKey, Stage: r.Stage, StageReason: r.StageReason,
		Symbol: r.Symbol, Direction: r.Direction, LevelRole: r.LevelRole,
		LevelBodyTop: r.LevelBodyTop, H1CloseTime: r.H1CloseTime, WindowClose: r.WindowClose,
		EntryRef: r.EntryRef, StopPx: r.StopPx, TargetPx: r.TargetPx,
		RREstimate: r.RREstimate, RRConfigured: r.RRConfigured, MomentumStall: r.MomentumStall,
		SignalID: r.SignalID, SubmittedAt: r.SubmittedAt, BrokerOrderID: r.BrokerOrderID,
		BrokerStatus: r.BrokerStatus, FillPrice: r.FillPrice, FillQty: r.FillQty,
		FillRR: r.FillRR, RejectReason: r.RejectReason,
		CreatedAt: r.CreatedAt.UnixMilli(),
	}
}

func (s *Server) handlePictureHtfOpportunities(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}
	rows, err := s.store.PictureHtfByTrader(traderID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	links, linkErr := s.pictureHtfPlanLinks(traderID, rows)
	out := make([]pictureHtfRowDTO, 0, len(rows))
	for _, r := range rows {
		dto := pictureHtfRowToDTO(r)
		dto.PlanLink = links[r.OppKey]
		out = append(out, dto)
	}
	resp := gin.H{"rows": out, "count": len(out)}
	if linkErr != "" {
		// A link read that failed is NOT "no link": say so instead of letting
		// the absent plan_link read as "never became a Day Plan scenario".
		resp["plan_links_unread"] = linkErr
	}
	c.JSON(http.StatusOK, resp)
}

// pictureHtfPlanLinks joins the listed opportunities to the armed_orders rows
// that carry them (source = picture, source_ref = opp_key), one row per
// opportunity: a live row first, else the newest — the order UpsertArm's
// source pin reads them in. Read-only. The second value is the reason the
// ledger could not be read ("" when it was).
func (s *Server) pictureHtfPlanLinks(traderID string, rows []store.PictureHtfOpportunityDB) (map[string]*pictureHtfPlanLinkDTO, string) {
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		if k := strings.TrimSpace(r.OppKey); k != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil, ""
	}
	if s.store == nil {
		return nil, "arm ledger unavailable"
	}
	var arms []store.ArmedOrderDB
	if err := s.store.ArmedOrders().DB().
		Where("trader_id = ? AND source = ? AND source_ref IN ?", traderID, store.ArmSourcePicture, keys).
		Order("id DESC").Find(&arms).Error; err != nil {
		return nil, "arm ledger unavailable: " + err.Error()
	}
	chosen := map[string]*store.ArmedOrderDB{}
	for i := range arms {
		a := &arms[i]
		cur := chosen[a.SourceRef]
		// id DESC: the first row seen is the newest; a later (older) row
		// replaces it only when it is live and the chosen one is terminal.
		if cur == nil || (store.IsTerminalArmState(cur.State) && !store.IsTerminalArmState(a.State)) {
			chosen[a.SourceRef] = a
		}
	}
	out := make(map[string]*pictureHtfPlanLinkDTO, len(chosen))
	for ref, a := range chosen {
		out[ref] = &pictureHtfPlanLinkDTO{
			PlanID: a.PlanID, ScenarioID: a.Scenario, PlanVersion: a.Version,
			ArmRowID: a.ID, ArmState: a.State, ArmSignalID: a.SignalID,
		}
	}
	return out, ""
}
