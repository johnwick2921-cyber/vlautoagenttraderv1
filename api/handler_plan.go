package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"

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
	// P5.1 — plan_final = base doc + overlays (RFC-6902, applied in ASC order). A
	// bad overlay is skipped; the applied result is re-armored via ValidatePlanDoc
	// and falls back to the base doc on failure (the plan-doc analog of B2 armor).
	overlays, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
	if len(overlays) > 0 {
		patches := make([]string, 0, len(overlays))
		for _, ov := range overlays {
			patches = append(patches, ov.Patch)
		}
		if base, mErr := json.Marshal(doc); mErr == nil {
			final, _ := kernel.ApplyOverlayPatches(base, patches)
			var merged kernel.PlanDoc
			if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDoc(&merged) == nil {
				doc = merged // plan_final
			}
		}
	}

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
	// Scope the ack to the caller's trader (IDOR guard) — 404 if not owned.
	found, err := s.store.Alert().AckForTrader(traderID, body.AlertID)
	if err != nil {
		SafeInternalError(c, "ack alert", err)
		return
	}
	if !found {
		SafeNotFound(c, "Alert")
		return
	}
	c.JSON(200, gin.H{"acked": true, "alert_id": body.AlertID})
}

// marketRef returns the last CLOSED price + the daily-ATR proxy from the live 1m
// bars (the B2 armor scale for owner-entered level prices). Both 0 when no bars.
func marketRef(symbol string, now time.Time) (lastPrice, dATR float64) {
	if market.FuturesBarsProvider == nil {
		return 0, 0
	}
	bars := market.FuturesBarsProvider(symbol, "1m", kernel.AISVPBarCount)
	if len(bars) == 0 {
		return 0, 0
	}
	nowMs := now.UnixMilli()
	for i := len(bars) - 1; i >= 0; i-- {
		if bars[i].CloseTime < nowMs {
			lastPrice = bars[i].Close
			break
		}
	}
	dATR = kernel.DailyATRProxy(bars, now)
	return lastPrice, dATR
}

// overlayPriceViolations runs the B2 level armor over every price an owner patch
// introduces (a level object's "price", or a "/price" replace value), so an owner
// fat-finger is rejected exactly as an implausible AI price would be.
func overlayPriceViolations(patchJSON string, lastPrice, dATR float64) []string {
	var ops []struct {
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	if json.Unmarshal([]byte(patchJSON), &ops) != nil {
		return nil
	}
	var out []string
	check := func(p float64) {
		if reason, bad := kernel.LevelPriceViolation(p, lastPrice, dATR); bad {
			out = append(out, reason)
		}
	}
	for _, op := range ops {
		if len(op.Value) == 0 {
			continue
		}
		if strings.HasSuffix(op.Path, "/price") {
			var f float64
			if json.Unmarshal(op.Value, &f) == nil {
				check(f)
			}
			continue
		}
		// a level object carrying a numeric price
		var obj struct {
			Price float64 `json:"price"`
		}
		if json.Unmarshal(op.Value, &obj) == nil && obj.Price > 0 {
			check(obj.Price)
		}
	}
	return out
}

// handlePlanOverlay POST /api/plan/overlay — append an RFC-6902 overlay to the
// active plan. test-op concurrency (§42): the patch is applied strictly to the
// current plan_final; a failed `test` or bad op → 409. Owner prices pass B2 armor.
// Origin defaults to "owner"; the Ask-Planner Apply passes "planner-revised".
func (s *Server) handlePlanOverlay(c *gin.Context) {
	var body struct {
		TraderID string `json:"trader_id"`
		Symbol   string `json:"symbol"`
		Patch    string `json:"patch"`  // JSON array of RFC-6902 ops
		Origin   string `json:"origin"` // owner | planner-revised (default owner)
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
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	if strings.TrimSpace(body.Patch) == "" {
		SafeBadRequest(c, "patch is required")
		return
	}
	origin := "owner"
	if body.Origin == "planner-revised" {
		origin = "planner-revised"
	}
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "MNQ"
	}

	now := time.Now()
	reg := kernel.DefaultSessionRegistry()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	sess, ok := reg.ActiveSession(now)
	if !ok || !sess.Enabled {
		SafeBadRequest(c, "no active plan to edit (night / disabled session)")
		return
	}
	row, err := s.store.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
	if err != nil || row == nil {
		SafeNotFound(c, "Active plan")
		return
	}

	// Build the current plan_final (base + existing overlays), then apply the new
	// patch STRICTLY on top — this runs the test-op concurrency guard + validity.
	overlays, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
	patches := make([]string, 0, len(overlays))
	for _, ov := range overlays {
		patches = append(patches, ov.Patch)
	}
	current, _ := kernel.ApplyOverlayPatches([]byte(row.Doc), patches)
	candidate, err := kernel.ApplyPatchStrict(current, body.Patch)
	if err != nil {
		// A failed test-op or bad path is a concurrency/validity conflict.
		c.JSON(409, gin.H{"error": "overlay rejected: " + err.Error()})
		return
	}
	var merged kernel.PlanDoc
	if json.Unmarshal(candidate, &merged) != nil || kernel.ValidatePlanDoc(&merged) != nil {
		SafeBadRequest(c, "patch produces an invalid plan (enum/count armor)")
		return
	}

	// B2 price armor on every owner-entered price in the patch.
	lastPrice, dATR := marketRef(symbol, now)
	if v := overlayPriceViolations(body.Patch, lastPrice, dATR); len(v) > 0 {
		c.JSON(422, gin.H{"error": "⛔ price armor: " + strings.Join(v, "; ")})
		return
	}

	ov := &store.PlanOverlayDB{
		OverlayID:   fmt.Sprintf("%s:o:%d", row.PlanID, now.UnixNano()),
		PlanID:      row.PlanID,
		PlanVersion: row.Version,
		Patch:       body.Patch,
		Origin:      origin,
	}
	overlayVersion, err := s.store.Plan().AppendOverlay(ov)
	if err != nil {
		SafeInternalError(c, "append overlay", err)
		return
	}
	c.JSON(200, gin.H{"overlay_version": overlayVersion, "plan_version": row.Version, "origin": origin})
}

// handlePlanOwnerLevel POST /api/plan/owner-level — add a STICKY owner level
// (P3.6-C store). Guarded write: B2-armored price, WHERE-scoped by symbol, note +
// scenario tag ride along to the planner. Owner data is SACRED — never a live acct.
func (s *Server) handlePlanOwnerLevel(c *gin.Context) {
	var body struct {
		TraderID    string  `json:"trader_id"`
		Symbol      string  `json:"symbol"`
		Price       float64 `json:"price"`
		Label       string  `json:"label"`
		Note        string  `json:"note"`
		ScenarioTag string  `json:"scenario_tag"`
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
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "MNQ"
	}
	if body.Price <= 0 {
		SafeBadRequest(c, "price is required")
		return
	}
	lastPrice, dATR := marketRef(symbol, time.Now())
	if reason, bad := kernel.LevelPriceViolation(body.Price, lastPrice, dATR); bad {
		c.JSON(422, gin.H{"error": "⛔ price armor: " + reason})
		return
	}
	lvl := &store.OwnerLevelDB{
		Symbol:      symbol,
		Price:       body.Price,
		Label:       strings.TrimSpace(body.Label),
		Note:        strings.TrimSpace(body.Note),
		ScenarioTag: strings.TrimSpace(body.ScenarioTag),
		CreatedAt:   time.Now().Unix(), // no autoCreateTime on this table
	}
	if err := s.store.OwnerLevel().Save(lvl); err != nil {
		SafeInternalError(c, "save owner level", err)
		return
	}
	c.JSON(200, gin.H{"id": lvl.ID, "symbol": symbol, "price": body.Price})
}

// handlePlanOwnerLevelDelete POST /api/plan/owner-level/delete — remove a sticky
// owner level by id (the edit sheet's Delete for a 👤 level).
func (s *Server) handlePlanOwnerLevelDelete(c *gin.Context) {
	var body struct {
		TraderID string `json:"trader_id"`
		ID       int64  `json:"id"`
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
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	if body.ID <= 0 {
		SafeBadRequest(c, "id is required")
		return
	}
	if err := s.store.OwnerLevel().Delete(body.ID); err != nil {
		SafeInternalError(c, "delete owner level", err)
		return
	}
	c.JSON(200, gin.H{"deleted": true, "id": body.ID})
}
