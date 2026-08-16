package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// traderDayPlanEnabled reports whether a trader has the day-plan feature on — the
// gate for mutating the (session-global) day plan.
func traderDayPlanEnabled(at *trader.AutoTrader) bool {
	cfg := at.GetStrategyConfig()
	return cfg != nil && cfg.DayPlan != nil && cfg.DayPlan.PlanEnabled
}

// planOverlayMu serializes the read-fold-test-append span of an overlay write so
// the §42 test-op concurrency guard is atomic (two racing writers can't both test
// against the same stale plan_final and both append). Overlay writes are rare
// (owner edits), so a single coarse mutex is ample.
var planOverlayMu sync.Mutex

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

	reg := s.planRegistry()
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

// mergedPriceViolations runs the B2 level armor over every price in the RESOLVED
// plan_final (level prices + scenario target-chain prices), so an implausible
// owner/planner price is rejected regardless of which patch shape introduced it
// (a whole-array /levels replace, a target_chain add, etc.) — armoring the result,
// not the patch text.
func mergedPriceViolations(doc *kernel.PlanDoc, lastPrice, dATR float64) []string {
	var out []string
	check := func(p float64) {
		if reason, bad := kernel.LevelPriceViolation(p, lastPrice, dATR); bad {
			out = append(out, reason)
		}
	}
	for _, l := range doc.Levels {
		check(l.Price)
	}
	for _, s := range doc.Scenarios {
		for _, t := range s.TargetChain {
			check(t)
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
	at, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	// Only a day-plan-owning trader may mutate the (session-global) day plan — a
	// crypto / plan-off trader must not edit the futures plan.
	if !traderDayPlanEnabled(at) {
		SafeForbidden(c, "day_plan is not enabled for this trader")
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

	overlayVersion, planVersion, code, msg := s.applyPlanOverlay(symbol, body.Patch, origin, time.Now())
	if code != 0 {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(200, gin.H{"overlay_version": overlayVersion, "plan_version": planVersion, "origin": origin})
}

// applyPlanOverlay resolves the active plan, applies patchJSON strictly onto the
// current plan_final (test-op concurrency guard), B2-armors every owner price,
// validates enums/counts, then appends the overlay. Shared by the overlay POST
// and the Ask-Planner Apply. Returns (overlayVersion, planVersion, httpCode, msg);
// httpCode==0 means success.
func (s *Server) applyPlanOverlay(symbol, patchJSON, origin string, now time.Time) (int, int, int, string) {
	if strings.TrimSpace(patchJSON) == "" {
		return 0, 0, 400, "patch is required"
	}
	// Atomic read-fold-test-append (test-op concurrency is only a guard if the
	// span can't interleave with another writer).
	planOverlayMu.Lock()
	defer planOverlayMu.Unlock()
	reg := s.planRegistry()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	sess, ok := reg.ActiveSession(now)
	if !ok || !sess.Enabled {
		return 0, 0, 400, "no active plan to edit (night / disabled session)"
	}
	row, err := s.store.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
	if err != nil || row == nil {
		return 0, 0, 404, "active plan not found"
	}
	overlays, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
	patches := make([]string, 0, len(overlays))
	for _, ov := range overlays {
		patches = append(patches, ov.Patch)
	}
	current, _ := kernel.ApplyOverlayPatches([]byte(row.Doc), patches)
	candidate, err := kernel.ApplyPatchStrict(current, patchJSON)
	if err != nil {
		return 0, 0, 409, "overlay rejected: " + err.Error() // test-op / validity conflict
	}
	var merged kernel.PlanDoc
	if json.Unmarshal(candidate, &merged) != nil || kernel.ValidatePlanDoc(&merged) != nil {
		return 0, 0, 400, "patch produces an invalid plan (enum/count/sign armor)"
	}
	// B2 price armor — sweep the RESULTING plan_final's prices (levels +
	// scenario targets), robust to every patch shape (/levels, /levels/N,
	// /levels/-, a whole-array /levels replace, /scenarios/N/target_chain/-…).
	lastPrice, dATR := marketRef(symbol, now)
	if v := mergedPriceViolations(&merged, lastPrice, dATR); len(v) > 0 {
		return 0, 0, 422, "⛔ price armor: " + strings.Join(v, "; ")
	}
	ov := &store.PlanOverlayDB{
		OverlayID:   fmt.Sprintf("%s:o:%d", row.PlanID, now.UnixNano()),
		PlanID:      row.PlanID,
		PlanVersion: row.Version,
		Patch:       patchJSON,
		Origin:      origin,
	}
	overlayVersion, err := s.store.Plan().AppendOverlay(ov)
	if err != nil {
		return 0, 0, 500, "append overlay: " + err.Error()
	}
	return overlayVersion, row.Version, 0, ""
}

// handlePlanAsk POST /api/plan/ask — a plan-scoped Q&A with the planner. The
// verbatim anti-sycophancy contract is the system prompt AND enforced in code
// (kernel.ParsePlannerReply): a bare disagreement never yields a patch. The owner
// question + structured reply are persisted; the verdict is logged for the KPI.
func (s *Server) handlePlanAsk(c *gin.Context) {
	var body struct {
		TraderID string `json:"trader_id"`
		Symbol   string `json:"symbol"`
		Question string `json:"question"`
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
	at, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	if strings.TrimSpace(body.Question) == "" {
		SafeBadRequest(c, "question is required")
		return
	}
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "MNQ"
	}

	// Resolve the SAME active plan the card shows.
	now := time.Now()
	reg := s.planRegistry()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	sess, ok := reg.ActiveSession(now)
	if !ok || !sess.Enabled {
		SafeBadRequest(c, "no active plan to ask about (night / disabled session)")
		return
	}
	row, err := s.store.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
	if err != nil || row == nil {
		SafeNotFound(c, "Active plan")
		return
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil {
		SafeInternalError(c, "parse plan", fmt.Errorf("bad plan doc"))
		return
	}

	// Build the plan block + live status for the planner's context.
	planBlock := kernel.RenderPlanBlock(doc, sess.Name)
	liveStatus := ""
	if market.FuturesBarsProvider != nil {
		bars := market.FuturesBarsProvider(symbol, "1m", kernel.AISVPBarCount)
		if len(bars) > 0 {
			price, dATR := marketRef(symbol, now)
			liveStatus = kernel.RenderPlanStatus(symbol, doc, bars, price, dATR, "2x5m", maxI(0, 2-(row.Version-1)), now.UnixMilli())
		}
	}
	userPrompt := kernel.BuildAskPlannerUserPrompt(planBlock, liveStatus, body.Question)

	// Call the SAME planner model that authored the plan.
	client, _ := at.ResolvePlannerClient()
	if client == nil && !(SandboxMode() && !SandboxRealLLM()) {
		SafeInternalError(c, "resolve planner client", fmt.Errorf("no planner client"))
		return
	}
	var raw string
	if SandboxMode() && !SandboxRealLLM() {
		raw = sandboxPlannerRaw(userPrompt) // canned, zero cost
	} else {
		var callErr error
		raw, callErr = client.CallWithMessages(kernel.AskPlannerSystemPrompt, userPrompt)
		if callErr != nil {
			SafeInternalError(c, "ask planner", callErr)
			return
		}
	}
	reply, err := kernel.ParsePlannerReply(raw)
	if err != nil {
		c.JSON(502, gin.H{"error": "planner reply malformed", "raw": raw})
		return
	}

	// Persist the owner question + the structured planner reply (verdict logged).
	_, _ = s.store.PlanQA().Append(&store.PlanQADB{
		TraderID: traderID, PlanID: row.PlanID, TradeDate: tradeDate, Session: sess.Name,
		Role: "owner", Content: body.Question, CreatedAt: now.Unix(),
	})
	patchStr := ""
	if len(reply.Patch) > 0 {
		patchStr = string(reply.Patch)
	}
	qaID, _ := s.store.PlanQA().Append(&store.PlanQADB{
		TraderID: traderID, PlanID: row.PlanID, TradeDate: tradeDate, Session: sess.Name,
		Role: "planner", Content: reply.Summary, Evidence: reply.Evidence,
		PointClass: reply.PointClass, Verdict: reply.Verdict, Patch: patchStr,
		CreatedAt: now.Unix() + 1, // keep the reply after the question in id/time order
	})

	c.JSON(200, gin.H{
		"qa_id":        qaID,
		"plan_id":      row.PlanID,
		"plan_version": row.Version,
		"reply": gin.H{
			"evidence": reply.Evidence, "point_class": reply.PointClass,
			"verdict": reply.Verdict, "summary": reply.Summary, "patch": patchStr,
		},
	})
}

// handlePlanThread GET /api/plan/ask?trader_id=xxx — the plan's Q&A thread + the
// sycophancy KPI (verdict counts).
func (s *Server) handlePlanThread(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	planID := strings.TrimSpace(c.Query("plan_id"))
	if planID == "" {
		// default to today's active plan id
		now := time.Now()
		reg := s.planRegistry()
		tradeDate := now.In(planChicago()).Format("2006-01-02")
		if sess, ok := reg.ActiveSession(now); ok {
			planID = store.MakePlanID(tradeDate, sess.Name)
		}
	}
	msgs, _ := s.store.PlanQA().ListForPlan(traderID, planID, 200)
	kpi, _ := s.store.PlanQA().VerdictStats(traderID)
	c.JSON(200, gin.H{"thread": msgs, "kpi": kpi})
}

// handlePlanAskApply POST /api/plan/ask/apply — apply a PROPOSE-MERGE patch as an
// overlay (origin planner-revised) and mark the reply applied. Bare-disagreement
// replies carry no patch, so there is nothing to apply (guarded here too).
func (s *Server) handlePlanAskApply(c *gin.Context) {
	var body struct {
		TraderID string `json:"trader_id"`
		Symbol   string `json:"symbol"`
		QaID     int64  `json:"qa_id"`
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
	if body.QaID <= 0 {
		SafeBadRequest(c, "qa_id is required")
		return
	}
	msg, err := s.store.PlanQA().GetForTrader(traderID, body.QaID)
	if err != nil {
		SafeInternalError(c, "get qa message", err)
		return
	}
	if msg == nil {
		SafeNotFound(c, "Ask-Planner message")
		return
	}
	if msg.Verdict != "PROPOSE-MERGE" || strings.TrimSpace(msg.Patch) == "" {
		SafeBadRequest(c, "this reply has no patch to apply")
		return
	}
	// One reply = one overlay: a re-tap / retry must not stack duplicate overlays.
	if msg.Applied {
		c.JSON(200, gin.H{"applied": true, "already_applied": true})
		return
	}
	// Bind the apply to the plan the reply was authored against: a PROPOSE-MERGE
	// from an earlier (rolled/expired) plan must not silently patch a different
	// active plan.
	now := time.Now()
	reg := s.planRegistry()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	sess, ok := reg.ActiveSession(now)
	if !ok || !sess.Enabled || store.MakePlanID(tradeDate, sess.Name) != msg.PlanID {
		c.JSON(409, gin.H{"error": "this reply was authored against a plan that is no longer active"})
		return
	}
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "MNQ"
	}
	overlayVersion, planVersion, code, emsg := s.applyPlanOverlay(symbol, msg.Patch, "planner-revised", now)
	if code != 0 {
		c.JSON(code, gin.H{"error": emsg})
		return
	}
	_ = s.store.PlanQA().MarkApplied(traderID, body.QaID)
	c.JSON(200, gin.H{"applied": true, "overlay_version": overlayVersion, "plan_version": planVersion})
}

// handlePlanTrades GET /api/plan/trades?trader_id=xxx — graded closed trades (the
// trade-review feed) + an adherence summary (counts + GPA). Grade is SEPARATE
// from P&L; both are shown so discipline and outcome can be compared.
func (s *Server) handlePlanTrades(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	rows, err := s.store.Position().GetGradedClosedPositions(traderID, 50)
	if err != nil {
		SafeInternalError(c, "list graded trades", err)
		return
	}
	trades := make([]gin.H, 0, len(rows))
	grades := make([]string, 0, len(rows))
	for _, p := range rows {
		grades = append(grades, p.AdherenceGrade)
		trades = append(trades, gin.H{
			"symbol": p.Symbol, "side": p.Side,
			"entry_price": p.EntryPrice, "exit_price": p.ExitPrice,
			"entry_time": p.EntryTime, "exit_time": p.ExitTime,
			"realized_pnl": p.RealizedPnL, "mae": p.MAE, "mfe": p.MFE,
			"entry_confidence":  p.EntryConfidence,
			"cited_scenario_id": p.CitedScenarioID, "plan_matched": p.PlanMatched,
			"plan_version":    p.PlanVersion,
			"adherence_grade": p.AdherenceGrade,
			"adherence_label": kernel.AdherenceLabel(p.AdherenceGrade),
		})
	}
	c.JSON(200, gin.H{"trades": trades, "summary": kernel.SummarizeAdherence(grades)})
}

// handlePlanStats GET /api/plan/stats?trader_id=xxx — the matched-random honesty
// gate. Returns the FROZEN weekly snapshot (the authoritative "beats random"
// verdict — never recomputed live, so no optional-stopping) + live progress
// counts (n/target) for the WARMING badges. Progress makes NO significance claim.
func (s *Server) handlePlanStats(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	mr := s.store.MatchedRandom()

	// The authoritative gate = the frozen weekly snapshot (may be null pre-first-eval).
	var weekly gin.H
	if row, _ := mr.LatestWeekly(); row != nil {
		var verdicts []kernel.TypeVerdict
		_ = json.Unmarshal([]byte(row.SummaryJSON), &verdicts)
		weekly = gin.H{"iso_week": row.ISOWeek, "computed_at": row.ComputedAt, "verdicts": verdicts}
	}

	// Live progress: raw counts only (WARMING framing) — no green claim here.
	progress := make([]gin.H, 0)
	if counts, err := mr.CountsByType(); err == nil {
		for t, cc := range counts {
			rate := 0.0
			if cc.Touches > 0 {
				rate = float64(cc.Reactions) / float64(cc.Touches)
			}
			progress = append(progress, gin.H{
				"level_type": t, "n": cc.Touches, "reactions": cc.Reactions,
				"target_n": kernel.PreRegisteredN, "react_rate": rate,
				"warming": cc.Touches < kernel.PreRegisteredN,
			})
		}
	}

	c.JSON(200, gin.H{
		"weekly":   weekly, // null until the first Sunday eval
		"progress": progress,
		"target_n": kernel.PreRegisteredN,
		"alpha":    kernel.BonferroniAlpha(),
	})
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

// handlePlanSessionRegistry GET /api/plan/session-registry?trader_id=xxx — returns
// the EFFECTIVE admin session registry (stored or the shipped default) + whether the
// default is in force. W8 — the read side of the admin-registry wire the gates honor.
func (s *Server) handlePlanSessionRegistry(c *gin.Context) {
	traderID := strings.TrimSpace(c.Query("trader_id"))
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}
	if _, err := s.traderManager.GetTrader(traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	raw, _ := s.store.GetSystemConfig(kernel.SessionRegistryConfigKey)
	reg, perr := kernel.LoadSessionRegistry(raw) // fail-safe to default
	c.JSON(200, gin.H{
		"registry":      reg,
		"is_default":    strings.TrimSpace(raw) == "" || perr != nil,
		"sessions":      len(reg.Sessions),
		"parse_warning": errString(perr),
	})
}

// handlePlanSessionRegistrySave POST /api/plan/session-registry — validate + persist
// an admin-edited registry. Unlike the fail-safe loader, a MALFORMED edit is REFUSED
// (400), never silently defaulted — an admin must not be able to disable the gates by
// posting junk. The next CME session-day's gates pick it up (never mid-session).
func (s *Server) handlePlanSessionRegistrySave(c *gin.Context) {
	var body struct {
		TraderID string                 `json:"trader_id"`
		Registry kernel.SessionRegistry `json:"registry"`
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
	if err := kernel.ValidateSessionRegistry(body.Registry); err != nil {
		SafeBadRequest(c, "invalid registry: "+err.Error())
		return
	}
	raw, err := body.Registry.Marshal()
	if err != nil {
		SafeInternalError(c, "marshal registry", err)
		return
	}
	if err := s.store.SetSystemConfig(kernel.SessionRegistryConfigKey, raw); err != nil {
		SafeInternalError(c, "save registry", err)
		return
	}
	c.JSON(200, gin.H{"saved": true, "sessions": len(body.Registry.Sessions),
		"note": "honored by the next CME session-day's gates (not mid-session)"})
}

// errString renders an error as a string ("" when nil) for JSON responses.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// handlePlanApprove POST /api/plan/approve — grant entries for THIS trader's current
// CME session-day when approval_required is ON (W9). The owner clicks it after
// reviewing the plan; the entry gate then lets entries through for that session-day.
func (s *Server) handlePlanApprove(c *gin.Context) {
	var body struct {
		TraderID string `json:"trader_id"`
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
	dayKey := kernel.CMESessionDayKey(time.Now())
	if err := s.store.SetSystemConfig(trader.ApprovalKey(traderID, dayKey), "granted"); err != nil {
		SafeInternalError(c, "grant approval", err)
		return
	}
	c.JSON(200, gin.H{"approved": true, "session_day": dayKey,
		"note": "entries flow for this CME session-day"})
}

// planRegistry returns the EFFECTIVE session registry for the read-side plan
// APIs: the admin-edited registry stored in system_config (the same source the
// trader gates read since W8), falling back to the shipped default when the key
// is absent or malformed. Before this, every /api/plan/* handler hardcoded
// DefaultSessionRegistry, so an admin who edited the registry got a card, an
// overlay gate and an Ask-Planner scope running on a DIFFERENT clock than the
// executor — two clocks, silently disagreeing (design-conformance audit, D7).
// Empty key → byte-identical to the previous behavior.
func (s *Server) planRegistry() kernel.SessionRegistry {
	if s.store == nil {
		return kernel.DefaultSessionRegistry()
	}
	raw, _ := s.store.GetSystemConfig(kernel.SessionRegistryConfigKey)
	reg, _ := kernel.LoadSessionRegistry(raw) // fail-safe to the default
	return reg
}

// ─── W13 · PLAN RE-ALIGNMENT ON OWNER EDIT ────────────────────────────────────
// Owner decision 2026-08-16: an overlay save AUTO-triggers a whole-plan
// re-examination that returns a PROPOSAL. Nothing changes without an Apply tap —
// the apply path is the existing POST /api/plan/ask/apply (same qa row, same
// planner-revised overlay), so there is exactly one mutation door.

// resolvePlanFinal folds a plan row's overlays into plan_final (the doc the card
// and the executor see), armored: a bad overlay falls back to the base doc.
func (s *Server) resolvePlanFinal(row *store.PlanDB) (kernel.PlanDoc, bool) {
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil {
		return kernel.PlanDoc{}, false
	}
	overlays, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
	if len(overlays) == 0 {
		return doc, true
	}
	patches := make([]string, 0, len(overlays))
	for _, ov := range overlays {
		patches = append(patches, ov.Patch)
	}
	base, err := json.Marshal(doc)
	if err != nil {
		return doc, true
	}
	final, _ := kernel.ApplyOverlayPatches(base, patches)
	var merged kernel.PlanDoc
	if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDoc(&merged) == nil {
		return merged, true
	}
	return doc, true
}

// RealignDebounceSec collapses rapid saves (bulk-add rows, fast successive edits)
// into ONE call.
const RealignDebounceSec int64 = 20

// realignGateDecision is the pure auto-trigger gate: "" = proceed with the call,
// "debounced" = an auto re-align already ran inside the window (this is what makes
// a bulk-add batch ONE call), "capped" = the auto budget for this plan is spent and
// the owner should use the manual button. A MANUAL re-align bypasses both — the
// owner asked for it explicitly. lastAt < 0 means "no previous auto re-align".
func realignGateDecision(manual bool, used, capN int, lastAt, nowUnix int64) string {
	if manual {
		return ""
	}
	if lastAt >= 0 && nowUnix-lastAt < RealignDebounceSec {
		return "debounced"
	}
	if capN > 0 && used >= capN {
		return "capped"
	}
	return ""
}

// handlePlanRealign POST /api/plan/realign — re-examine the whole plan after an
// owner edit. Always 200 with a `status` the FE renders:
//
//	proposal   → PROPOSE-MERGE + patch (the card slides in; Apply-gated)
//	no-change  → the plan stands (quiet auto-fading chip)
//	skipped    → no active plan / night / disabled / expired / day-plan off
//	debounced  → an identical-window call already ran (bulk-add safety)
//	capped     → auto budget spent → the FE shows the manual "Re-align plan" button
//	failed     → fail-closed: plan untouched, alert row written, NO partial patch
func (s *Server) handlePlanRealign(c *gin.Context) {
	var body struct {
		TraderID string `json:"trader_id"`
		Symbol   string `json:"symbol"`
		Manual   bool   `json:"manual"` // owner tapped Re-align → bypasses the cap
		Change   struct {
			Kind        string  `json:"kind"`
			Summary     string  `json:"summary"`
			Price       float64 `json:"price"`
			Label       string  `json:"label"`
			Grade       string  `json:"grade"`
			Instruction string  `json:"instruction"`
			Note        string  `json:"note"`
			ScenarioTag string  `json:"scenario_tag"`
			BatchCount  int     `json:"batch_count"`
		} `json:"change"`
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
	at, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		SafeNotFound(c, "Trader")
		return
	}
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "MNQ"
	}
	if !at.DayPlanOn() {
		c.JSON(200, gin.H{"status": "skipped", "reason": "day_plan_off"})
		return
	}

	// SKIP: no active plan · night / disabled session · expired plan.
	now := time.Now()
	reg := s.planRegistry()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	sess, ok := reg.ActiveSession(now)
	if !ok || !sess.Enabled {
		c.JSON(200, gin.H{"status": "skipped", "reason": "night_or_disabled_session"})
		return
	}
	row, err := s.store.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
	if err != nil || row == nil {
		c.JSON(200, gin.H{"status": "skipped", "reason": "no_active_plan"})
		return
	}
	if row.Lifecycle != "active" {
		c.JSON(200, gin.H{"status": "skipped", "reason": "plan_" + row.Lifecycle})
		return
	}

	trigger := kernel.TriggerOwnerEdit
	if body.Manual {
		trigger = kernel.TriggerManual
	}

	// DEBOUNCE + CAP (auto only). The debounce is what makes a bulk-add batch ONE
	// call even if the FE fires per row; the cap bounds automation spend.
	var lastAt int64 = -1
	var lastID int64
	used := 0
	if !body.Manual {
		if last, _ := s.store.PlanQA().LastPlannerByTrigger(traderID, row.PlanID, trigger); last != nil {
			lastAt, lastID = last.CreatedAt, last.ID
		}
		used, _ = s.store.PlanQA().CountPlannerByTrigger(traderID, row.PlanID, trigger)
	}
	capN := at.RealignCap()
	switch realignGateDecision(body.Manual, used, capN, lastAt, now.Unix()) {
	case "debounced":
		c.JSON(200, gin.H{"status": "debounced", "window_sec": RealignDebounceSec, "qa_id": lastID})
		return
	case "capped":
		c.JSON(200, gin.H{"status": "capped", "used": used, "cap": capN})
		return
	}

	doc, okDoc := s.resolvePlanFinal(row) // overlay-resolved: what the owner sees
	if !okDoc {
		c.JSON(200, gin.H{"status": "failed", "reason": "bad_plan_doc"})
		return
	}

	// SAME context blocks Ask-Planner uses.
	planBlock := kernel.RenderPlanBlock(doc, sess.Name)
	liveStatus := ""
	if market.FuturesBarsProvider != nil {
		if bars := market.FuturesBarsProvider(symbol, "1m", kernel.AISVPBarCount); len(bars) > 0 {
			price, dATR := marketRef(symbol, now)
			liveStatus = kernel.RenderPlanStatus(symbol, doc, bars, price, dATR, "2x5m", maxI(0, 2-(row.Version-1)), now.UnixMilli())
		}
	}
	change := kernel.OwnerChange{
		Kind: body.Change.Kind, Summary: body.Change.Summary, Price: body.Change.Price,
		Label: body.Change.Label, Grade: body.Change.Grade, Instruction: body.Change.Instruction,
		Note: body.Change.Note, ScenarioTag: body.Change.ScenarioTag, BatchCount: body.Change.BatchCount,
	}
	userPrompt := kernel.BuildRealignUserPrompt(planBlock, liveStatus, change)

	client, modelID := at.ResolvePlannerClient()
	if client == nil && !(SandboxMode() && !SandboxRealLLM()) {
		s.realignFailClosed(traderID, row, sess.Name, tradeDate, "no planner client")
		c.JSON(200, gin.H{"status": "failed", "reason": "no_planner_client"})
		return
	}

	// FAIL-CLOSED: any error or malformed reply → no proposal, plan untouched.
	started := time.Now()
	var raw string
	var callErr error
	if SandboxMode() && !SandboxRealLLM() {
		raw = sandboxPlannerRaw(userPrompt) // canned, zero cost
	} else {
		raw, callErr = client.CallWithMessages(kernel.AskPlannerSystemPrompt, userPrompt)
	}
	latencyMs := time.Since(started).Milliseconds()
	if callErr != nil {
		s.realignFailClosed(traderID, row, sess.Name, tradeDate, "planner call failed: "+callErr.Error())
		c.JSON(200, gin.H{"status": "failed", "reason": "planner_call_failed", "latency_ms": latencyMs})
		return
	}
	reply, perr := kernel.ParsePlannerReply(raw)
	if perr != nil {
		s.realignFailClosed(traderID, row, sess.Name, tradeDate, "planner reply malformed")
		c.JSON(200, gin.H{"status": "failed", "reason": "planner_reply_malformed", "latency_ms": latencyMs})
		return
	}

	// LOG into the SAME KPI table as Ask-Planner (one comparable series).
	patchStr := ""
	if len(reply.Patch) > 0 {
		patchStr = string(reply.Patch)
	}
	noChange := kernel.IsNoChange(reply)
	if noChange {
		patchStr = "" // a no-change verdict never carries a patch
	}
	costUSD := estimateRealignCostUSD(userPrompt, raw)
	qaID, _ := s.store.PlanQA().Append(&store.PlanQADB{
		TraderID: traderID, PlanID: row.PlanID, TradeDate: tradeDate, Session: sess.Name,
		Role: "planner", Content: reply.Summary, Evidence: reply.Evidence,
		PointClass: reply.PointClass, Verdict: reply.Verdict, Patch: patchStr,
		Trigger: trigger, ChallengeType: strings.TrimSpace(body.Change.Kind),
		CostUSD: costUSD, LatencyMs: latencyMs, CreatedAt: now.Unix(),
	})
	logger.Infof("🔄 plan re-align (%s/%s) %s → %s · %dms · ~$%.4f · model %s",
		trigger, body.Change.Kind, row.PlanID, reply.Verdict, latencyMs, costUSD, modelID)

	status := "proposal"
	if noChange {
		status = "no-change"
	}
	c.JSON(200, gin.H{
		"status": status, "qa_id": qaID,
		"plan_id": row.PlanID, "plan_version": row.Version,
		"would_become": fmt.Sprintf("v%d+o%d", row.Version, s.nextOverlayVersion(row)),
		"latency_ms":   latencyMs, "cost_usd": costUSD,
		"reply": gin.H{
			"evidence": reply.Evidence, "point_class": reply.PointClass,
			"verdict": reply.Verdict, "summary": reply.Summary, "patch": patchStr,
		},
	})
}

// realignFailClosed records the failure as a P1 alert row so a silent-failing
// automation is visible. The plan is never touched on this path.
func (s *Server) realignFailClosed(traderID string, row *store.PlanDB, session, tradeDate, reason string) {
	logger.Warnf("🔄 plan re-align FAIL-CLOSED (%s %s): %s — plan untouched", tradeDate, session, reason)
	if s.store == nil {
		return
	}
	_, _ = s.store.Alert().Emit(&store.AlertDB{
		TraderID: traderID, Level: "P1", Kind: "realign-failed",
		EventID: fmt.Sprintf("realign-fail:%s:%d", row.PlanID, time.Now().Unix()),
		Title:   "Plan re-align failed — plan unchanged", Body: reason,
		CreatedAt: time.Now().Unix(),
	})
}

// nextOverlayVersion is the overlay version an Apply would create (for the card's
// "would become v1+oN" header).
func (s *Server) nextOverlayVersion(row *store.PlanDB) int {
	overlays, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
	return len(overlays) + 1
}

// estimateRealignCostUSD approximates the spend of one re-align call from prompt +
// reply size (~4 chars/token) at the pinned reasoner's published rate. Approximate
// by construction — it exists so the owner can see the automation's running cost,
// not for billing.
func estimateRealignCostUSD(prompt, reply string) float64 {
	const (
		inPerMTok   = 0.28 // deepseek-v4-pro input, USD / 1M tokens
		outPerMTok  = 0.42 // deepseek-v4-pro output
		charsPerTok = 4.0
	)
	inTok := float64(len(prompt)) / charsPerTok
	outTok := float64(len(reply)) / charsPerTok
	return (inTok*inPerMTok + outTok*outPerMTok) / 1_000_000
}
