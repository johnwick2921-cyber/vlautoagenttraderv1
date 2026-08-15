package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"

	"github.com/gin-gonic/gin"
)

// P4.1 — /api/plan/* (mirror /api/risk/* inline trader_id). Additive + read-only
// except the alert ack. Handles the "day_plan enabled but no plan yet" pre-★2
// state gracefully: found=false, so the card renders its no-plan-yet state.

func planChicago() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.UTC
	}
	return loc
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// handlePlanToday GET /api/plan/today?trader_id=xxx[&symbol=MNQ] — the active
// plan (overlay-resolved) + live per-level facts from the P0.4 evaluator.
func (s *Server) handlePlanToday(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		symbol = "MNQ"
	}

	reg := kernel.DefaultSessionRegistry()
	now := time.Now()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	sess, ok := reg.ActiveSession(now)
	sessName := ""
	if ok {
		sessName = sess.Name
	}
	night := reg.IsNightMode(now)

	base := gin.H{"found": false, "trade_date": tradeDate, "session": sessName, "night": night, "mode": "advisory"}
	if !ok || !sess.Enabled {
		c.JSON(200, base) // night / disabled session → no active plan
		return
	}
	row, err := s.store.Plan().GetLatestPlanForSession(tradeDate, sessName)
	if err != nil || row == nil {
		c.JSON(200, base) // enabled but no plan yet (pre-★2) → graceful
		return
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil {
		c.JSON(200, base)
		return
	}
	// Overlays are empty pre-P5; plan_final = base doc (RFC-6902 application = P5).
	overlays, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)

	facts, price := planLevelFacts(symbol, doc, now)
	warming := ""
	if n, _ := s.store.SessionProfile().Count(symbol); n < 10 {
		warming = fmt.Sprintf("%d/10", n)
	}

	c.JSON(200, gin.H{
		"found":         true,
		"trade_date":    tradeDate,
		"session":       sessName,
		"version":       row.Version,
		"overlay_count": len(overlays),
		"lifecycle":     row.Lifecycle,
		"model_id":      row.ModelID,
		"night":         false,
		"mode":          "advisory", // P3.5 wired advisory only
		"doc":           doc,
		"level_facts":   facts,
		"price":         price,
		"replans_left":  maxI(0, 2-(row.Version-1)),
		"warming":       warming,
	})
}

// planLevelFacts computes per-level live facts from the latest 1m bars.
func planLevelFacts(symbol string, doc kernel.PlanDoc, now time.Time) ([]gin.H, float64) {
	if market.FuturesBarsProvider == nil {
		return nil, 0
	}
	bars := market.FuturesBarsProvider(symbol, "1m", kernel.AISVPBarCount)
	if len(bars) == 0 {
		return nil, 0
	}
	nowMs := now.UnixMilli()
	price := 0.0
	for i := len(bars) - 1; i >= 0; i-- {
		if bars[i].CloseTime < nowMs {
			price = bars[i].Close
			break
		}
	}
	out := make([]gin.H, 0, len(doc.Levels))
	for _, l := range doc.Levels {
		dir := kernel.DirAbove
		if l.Price < price {
			dir = kernel.DirBelow
		}
		f := kernel.EvaluateLevelFacts(bars, l.Price, dir, "2x5m", 3, nowMs)
		out = append(out, gin.H{
			"price":         l.Price,
			"label":         l.Label,
			"grade":         l.Grade,
			"instruction":   l.Instruction,
			"distance":      f.DistancePoints,
			"sweep":         f.Swept,
			"closes_beyond": maxI(f.ClosesBeyondUp, f.ClosesBeyondDown),
			"accept_have":   f.AcceptHave,
			"accept_need":   f.AcceptNeed,
			"still_valid":   f.StillValid,
		})
	}
	return out, price
}

// handlePlanHistory GET /api/plan/history?trader_id=xxx — recent plan versions.
func (s *Server) handlePlanHistory(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	rows, err := s.store.Plan().ListRecent(30)
	if err != nil {
		SafeInternalError(c, "list plans", err)
		return
	}
	hist := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		hist = append(hist, gin.H{
			"trade_date": r.TradeDate, "session": r.Session, "version": r.Version,
			"lifecycle": r.Lifecycle, "model_id": r.ModelID, "trigger_reason": r.TriggerReason,
		})
	}
	c.JSON(200, gin.H{"history": hist})
}

// handlePlanAlerts GET /api/plan/alerts?trader_id=xxx — the alert feed.
func (s *Server) handlePlanAlerts(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	rows, err := s.store.Alert().List(traderID, 50)
	if err != nil {
		SafeInternalError(c, "list alerts", err)
		return
	}
	unacked, _ := s.store.Alert().UnackedCount(traderID)
	c.JSON(200, gin.H{"alerts": rows, "unacked": unacked})
}

// handlePlanAlertAck POST /api/plan/alert-ack — ack an alert (query or JSON body).
func (s *Server) handlePlanAlertAck(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	var body struct {
		TraderID string `json:"trader_id"`
		AlertID  int64  `json:"alert_id"`
	}
	_ = c.ShouldBindJSON(&body)
	if traderID == "" {
		traderID = strings.TrimSpace(body.TraderID)
	}
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if body.AlertID <= 0 {
		SafeBadRequest(c, "alert_id is required")
		return
	}
	if err := s.store.Alert().Ack(body.AlertID); err != nil {
		SafeInternalError(c, "ack alert", err)
		return
	}
	c.JSON(200, gin.H{"acked": true, "alert_id": body.AlertID})
}
