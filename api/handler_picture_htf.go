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
	out := make([]pictureHtfRowDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, pictureHtfRowToDTO(r))
	}
	c.JSON(http.StatusOK, gin.H{"rows": out, "count": len(out)})
}
