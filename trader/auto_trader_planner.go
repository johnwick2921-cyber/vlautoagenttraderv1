package trader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nofx/calendar"
	"nofx/kernel"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
	"nofx/telemetry"
)

const plannerSystemPrompt = "You are a disciplined CME index-futures day-plan reasoner. Output ONLY the single JSON object requested — reasoning first, then the answer fields. No prose outside the JSON."

// P3.2 — PLANNER MODEL BINDING (RECON #12). The day-plan reasoner runs on a
// SECOND per-strategy model binding (day_plan.planner_model), independent of the
// executor's primary model. The EXACT pinned model ID is used (never an alias)
// and logged on every plan; an empty binding falls back to the primary model.

// resolvePlannerModelID is the pure decision: an empty planner model → the
// primary (usePrimary=true); otherwise the pinned planner model ID.
func resolvePlannerModelID(plannerModel, primaryModel string) (modelID string, usePrimary bool) {
	pm := strings.TrimSpace(plannerModel)
	if pm == "" {
		return primaryModel, true
	}
	return pm, false
}

// ResolvePlannerClient is the exported entry point (P5.4 Ask-Planner) so the API
// layer can reach the SAME planner model that authored the plan. Read-only use:
// the caller must never mutate traders/plans/bindings through it.
func (at *AutoTrader) ResolvePlannerClient() (mcp.AIClient, string) {
	return at.resolvePlannerClient()
}

// resolvePlannerClient returns the AI client for the planner + the resolved
// (pinned) model ID. Empty binding → the executor's primary client. A model that
// the registry can't resolve → the primary client (never a silent nil).
func (at *AutoTrader) resolvePlannerClient() (mcp.AIClient, string) {
	primaryModel := at.aiModel
	if primaryModel == "" {
		primaryModel = "deepseek"
	}
	var plannerModel string
	if at.config.StrategyConfig != nil && at.config.StrategyConfig.DayPlan != nil {
		plannerModel = at.config.StrategyConfig.DayPlan.PlannerModel
	}

	modelID, usePrimary := resolvePlannerModelID(plannerModel, primaryModel)
	if usePrimary {
		at.logInfof("🧠 planner model: empty binding → using primary %q", modelID)
		return at.mcpClient, modelID
	}

	client := mcp.NewAIClientByProvider(modelID)
	if client == nil {
		at.logWarnf("🧠 planner model %q unresolved by the registry → falling back to primary %q", modelID, primaryModel)
		return at.mcpClient, primaryModel
	}
	// Mirror the primary key resolution (provider-specific overrides).
	apiKey := at.config.CustomAPIKey
	customURL := at.config.CustomAPIURL
	switch modelID {
	case "qwen":
		if at.config.QwenKey != "" {
			apiKey = at.config.QwenKey
		}
	case "deepseek":
		if at.config.DeepSeekKey != "" {
			apiKey = at.config.DeepSeekKey
		}
	}
	client.SetAPIKey(apiKey, customURL, at.config.CustomModelName)
	at.logInfof("🧠 planner model resolved (pinned): %q", modelID)
	return client, modelID
}

// ---- P3.3 — read jobs + the planner call --------------------------------------

// plannerTradeDateCT returns the trade_date (CT calendar date) a read belongs to.
func plannerTradeDateCT(now time.Time) string {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.UTC
	}
	return now.In(loc).Format("2006-01-02")
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

// maybeRunSessionReads fires the per-session planner read at the registry read
// time for each ENABLED session, once per session-day (idempotent via the plan
// store). GATED on day_plan → dormant by default. Called per-cycle.
func (at *AutoTrader) maybeRunSessionReads() {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	reg := kernel.DefaultSessionRegistry() // TODO(P4): admin registry from system_config
	now := time.Now()
	tradeDate := plannerTradeDateCT(now)
	for i := range reg.Sessions {
		s := &reg.Sessions[i]
		if !s.Enabled || !timeReachedCT(now, s.ReadCT) {
			continue
		}
		existing, err := at.store.Plan().GetLatestPlanForSession(tradeDate, s.Name)
		if err != nil {
			continue
		}
		if existing == nil {
			at.runPlannerRead(s.Name, tradeDate) // first read this session-day
			continue
		}
		if existing.Lifecycle != "active" {
			continue // no_trade / died → done for the session
		}
		// P3.6 — RE-PLAN ON DEATH (cap replan_cap/session → NO-TRADE).
		if at.activePlanIsDead(existing) {
			replanCap := 2
			if dp := at.config.StrategyConfig.DayPlan; dp != nil && dp.ReplanCap > 0 {
				replanCap = dp.ReplanCap
			}
			if existing.Version-1 >= replanCap {
				at.writeNoTradePlan(s.Name, tradeDate, "re-plans exhausted after death condition")
			} else {
				at.logInfof("🗓️ plan %s %s v%d DIED — re-planning (cap %d/session).", tradeDate, s.Name, existing.Version, replanCap)
				at.runPlannerRead(s.Name, tradeDate) // appends a new version
			}
		}
	}
}

// activePlanIsDead reports whether the stored plan's thesis is spent (all its
// levels accepted through) per the P0.4 evaluator over the live bars.
func (at *AutoTrader) activePlanIsDead(row *store.PlanDB) bool {
	if market.FuturesBarsProvider == nil {
		return false
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil {
		return false
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return false
	}
	rule := ""
	if dp := at.config.StrategyConfig.DayPlan; dp != nil {
		rule = dp.AcceptanceRule
	}
	return kernel.PlanIsDead(doc, bars, rule, time.Now().UnixMilli())
}

// writeNoTradePlan appends a NO-TRADE plan (re-plans exhausted) + an alert event.
func (at *AutoTrader) writeNoTradePlan(session, tradeDate, reason string) {
	doc := kernel.NoTradePlanDoc(reason)
	docJSON, _ := json.Marshal(doc)
	_, err := at.store.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID(tradeDate, session), StrategyID: at.id,
		TradeDate: tradeDate, Session: session, TriggerReason: "replans_exhausted",
		Lifecycle: "no_trade", Doc: string(docJSON),
	})
	if err != nil {
		at.logErrorf("🗓️ planner: write NO-TRADE plan failed %s %s: %v", tradeDate, session, err)
		return
	}
	at.logErrorf("🚨 PLAN NO-TRADE %s %s: %s — session sits out.", tradeDate, session, reason)
	telemetry.IncGateBlock(at.id, "plan_replans_exhausted")
}

// runPlannerRead assembles the input package, calls the pinned planner client,
// and persists the plan (or a fail-closed NO-TRADE plan).
func (at *AutoTrader) runPlannerRead(session, tradeDate string) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	client, modelID := at.resolvePlannerClient()
	if client == nil {
		at.logErrorf("🗓️ planner: no client resolved for %s %s", tradeDate, session)
		return
	}
	input := at.assemblePlannerInput(session, tradeDate)
	prompt := kernel.BuildPlannerPrompt(input)
	hash := shortHash(prompt)
	at.runPlannerReadCore(session, tradeDate, modelID, hash, func() (string, error) {
		return client.CallWithMessages(plannerSystemPrompt, prompt)
	})
}

// runPlannerReadCore is the testable core: ≤2 retries, then FAIL-CLOSED to a
// NO-TRADE plan (never a stale plan, never nothing). Writes the append-only plan
// row. Returns (version, lifecycle, err).
func (at *AutoTrader) runPlannerReadCore(session, tradeDate, modelID, promptHash string, call func() (string, error)) (int, string, error) {
	var doc *kernel.PlanDoc
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ { // 1 + ≤2 retries
		raw, err := call()
		if err != nil {
			lastErr = err
			continue
		}
		d, perr := kernel.ParsePlanDoc(raw)
		if perr != nil {
			lastErr = perr
			continue
		}
		doc = d
		break
	}

	lifecycle := "active"
	trigger := session + "_scheduled_read"
	if doc == nil {
		doc = kernel.NoTradePlanDoc(fmt.Sprintf("read failed after retries: %v", lastErr))
		lifecycle = "no_trade"
		trigger = "planner_fail_closed"
		at.logErrorf("🚨 PLANNER FAIL-CLOSED %s %s: %v — writing a NO-TRADE plan (never stale, never uncalibrated).", tradeDate, session, lastErr)
		telemetry.IncGateBlock(at.id, "planner_fail_closed") // alert-event proxy until the P4 alert center
	}

	docJSON, _ := json.Marshal(doc)
	version, err := at.store.Plan().AppendPlan(&store.PlanDB{
		PlanID:        store.MakePlanID(tradeDate, session),
		StrategyID:    at.id,
		TradeDate:     tradeDate,
		Session:       session,
		TriggerReason: trigger,
		Lifecycle:     lifecycle,
		ModelID:       modelID,
		PromptHash:    promptHash,
		Doc:           string(docJSON),
	})
	if err != nil {
		at.logErrorf("🗓️ planner: write plan row failed for %s %s: %v", tradeDate, session, err)
		return 0, lifecycle, err
	}
	at.logInfof("🗓️ PLAN written %s %s v%d (model %s, lifecycle %s, prompt %s)", tradeDate, session, version, modelID, lifecycle, promptHash)
	return version, lifecycle, nil
}

// assemblePlannerInput builds the input package from stored + cached data (the
// 16:55 read builds entirely from stored data). Digests + owner note arrive with
// P3.6; regime daily/1h fields degrade to n/a until those TFs are fetched.
func (at *AutoTrader) assemblePlannerInput(session, tradeDate string) kernel.PlannerInput {
	symbol := at.futuresSymbol()
	now := time.Now()
	reg := kernel.DefaultSessionRegistry()

	var bars []market.Kline
	if market.FuturesBarsProvider != nil {
		bars = market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount)
	}
	var extra []kernel.DetectedLevel
	if kernel.NakedPOCProvider != nil {
		extra = kernel.NakedPOCProvider(symbol)
	}
	scored, price, dATR := kernel.AssembleScoredLevels(bars, reg, symbol, kernel.DefaultMaxLevels, now, extra...)

	// P3.6-C — STICKY OWNER LEVELS: always seated, tagged 👤, persisted across
	// sessions. Prepended so they lead the ranked table.
	if owned, err := at.store.OwnerLevel().ListActive(symbol); err == nil && len(owned) > 0 {
		ownerScored := make([]kernel.ScoredLevel, 0, len(owned))
		for _, o := range owned {
			label := "👤 " + o.Label
			if o.Note != "" {
				label += " (" + o.Note + ")"
			}
			ownerScored = append(ownerScored, kernel.ScoredLevel{
				DetectedLevel: kernel.DetectedLevel{Kind: kernel.KindOwner, Price: o.Price, Lo: o.Price, Hi: o.Price, Label: label, OriginDate: "owner", HTF: true, Info: o.ScenarioTag},
				Grade:         "A", Fresh: "owner", Distance: o.Price - price,
			})
		}
		scored = append(ownerScored, scored...)
	}

	var daily, hour1, min5 []market.Kline
	if market.FuturesBarsProvider != nil {
		daily = market.FuturesBarsProvider(symbol, "1d", 300)
		hour1 = market.FuturesBarsProvider(symbol, "1h", 300)
		min5 = market.FuturesBarsProvider(symbol, "5m", 300)
	}
	regime := kernel.ComputeRegime(kernel.RegimeInputs{Price: price, DailyBars: daily, Hour1Bars: hour1, Min5Bars: min5})

	var calEvents []kernel.PlannerCalendarEvent
	if slice, err := at.store.Calendar().GetSlice(tradeDate); err == nil && slice != nil {
		var evs []calendar.Event
		if json.Unmarshal([]byte(slice.EventsJSON), &evs) == nil {
			loc, _ := time.LoadLocation("America/Chicago")
			if loc == nil {
				loc = time.UTC
			}
			for _, e := range calendar.EventsForSession(evs, session) {
				calEvents = append(calEvents, kernel.PlannerCalendarEvent{
					TimeCT:   e.Time.In(loc).Format("15:04"),
					Currency: e.Currency,
					Title:    e.Title,
					Impact:   string(e.Impact),
				})
			}
		}
	}

	warming := ""
	if n, _ := at.store.SessionProfile().Count(symbol); n < 10 {
		warming = fmt.Sprintf("session-profile store warming (%d/10)", n)
	}

	// P3.6-A — tapered week digest chain (current-date sessions + 3 full dailies +
	// days 4-7 one-liners).
	sessionDigests, _ := at.store.Digest().SessionDigests(symbol, tradeDate)
	dailies, _ := at.store.Digest().RecentDailies(symbol, 7)
	digestChain := kernel.BuildDigestChain(sessionDigests, dailies)

	return kernel.PlannerInput{
		TradeDate:   tradeDate,
		Session:     session,
		ReadKind:    session + " scheduled read (stored+cached data)",
		Price:       price,
		DATR:        dATR,
		Regime:      regime,
		Levels:      scored,
		Calendar:    calEvents,
		DigestChain: digestChain,
		Warming:     warming,
	}
}

// maybeWriteDigests writes the 3-line session digest at each enabled session's
// close and the daily roll-up at the trade-date close (16:00 CT). Idempotent
// (SaveIfAbsent) → restart-safe. GATED on day_plan → dormant by default.
func (at *AutoTrader) maybeWriteDigests() {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	reg := kernel.DefaultSessionRegistry()
	now := time.Now()
	tradeDate := plannerTradeDateCT(now)
	symbol := at.futuresSymbol()
	sinceMs := kernel.CMESessionDayStart(now).UnixMilli()
	pnl, entries, _ := at.store.Position().GetSessionDayActivity(at.id, sinceMs)

	for i := range reg.Sessions {
		s := &reg.Sessions[i]
		if !s.Enabled {
			continue
		}
		end, ok := hhmmToMin(s.WindowEndCT)
		if !ok || ctMinutesNow(now) < end {
			continue // session not closed yet
		}
		text := kernel.FormatSessionDigest(s.Name, tradeDate, "", entries, pnl)
		if wrote, _ := at.store.Digest().SaveIfAbsent(&store.DigestDB{
			Symbol: symbol, TradeDate: tradeDate, Session: s.Name, Kind: "session", Text: text, CreatedAt: now.UnixMilli(),
		}); wrote {
			at.logInfof("📓 session digest written %s %s.", tradeDate, s.Name)
		}
	}

	// Daily roll-up at the 16:00 CT trade-date close.
	if ctMinutesNow(now) >= 16*60 {
		sessions, _ := at.store.Digest().SessionDigests(symbol, tradeDate)
		text := kernel.FormatDailyDigest(tradeDate, "", len(sessions), entries, pnl)
		if wrote, _ := at.store.Digest().SaveIfAbsent(&store.DigestDB{
			Symbol: symbol, TradeDate: tradeDate, Kind: "daily", Text: text, CreatedAt: now.UnixMilli(),
		}); wrote {
			at.logInfof("📓 daily digest written %s.", tradeDate)
		}
	}
}

// installActivePlanProvider wires kernel.ActivePlanProvider to read this store's
// latest ACTIVE plan for the current session (P3.4). no_trade/died plans and
// off-session return nil → the executor prompt is unchanged.
func installActivePlanProvider(st *store.Store) {
	kernel.ActivePlanProvider = func(symbol string) *kernel.ActivePlan {
		reg := kernel.DefaultSessionRegistry()
		now := time.Now()
		sess, ok := reg.ActiveSession(now)
		if !ok || !sess.Enabled {
			return nil
		}
		tradeDate := plannerTradeDateCT(now)
		row, err := st.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
		if err != nil || row == nil || row.Lifecycle != "active" {
			return nil
		}
		var doc kernel.PlanDoc
		if json.Unmarshal([]byte(row.Doc), &doc) != nil {
			return nil
		}
		replansLeft := 2 - (row.Version - 1) // default replan_cap 2 (P3.6 refines)
		if replansLeft < 0 {
			replansLeft = 0
		}
		return &kernel.ActivePlan{Doc: doc, Session: sess.Name, Version: row.Version, ReplansLeft: replansLeft}
	}
}

// recordPlanCitation records the executor's plan citation for an entry decision
// (P3.5 advisory): match-rate counters via B6 + a log line. Advisory only — it
// never gates the trade (plan restricts, never compels). GATED on day_plan.
func (at *AutoTrader) recordPlanCitation(d *kernel.Decision) {
	if !at.dayPlanEnabled() || d == nil {
		return
	}
	if d.Action != "open_long" && d.Action != "open_short" {
		return
	}
	// P5.5 hardening — a new open decision invalidates any citation a PRIOR
	// rejected/failed open left valid, so an open with no active plan can't
	// inherit a stale plan link. Only a live ActivePlan below re-arms it.
	at.lastCitation.valid = false
	if kernel.ActivePlanProvider == nil {
		return
	}
	ap := kernel.ActivePlanProvider(at.futuresSymbol())
	if ap == nil {
		return
	}
	res := kernel.ClassifyCitation(d.Action, d.CitedScenario, ap.Doc)
	switch {
	case res.OffPlan:
		telemetry.IncGateBlock(at.id, "plan_off_plan")
		at.logInfof("📋 advisory: %s cited off-plan (plan v%d).", d.Action, ap.Version)
	case res.Matched:
		telemetry.IncGateBlock(at.id, "plan_matched")
		at.logInfof("📋 advisory: %s cited %s ✓ matched (plan v%d).", d.Action, res.Cited, ap.Version)
	default:
		telemetry.IncGateBlock(at.id, "plan_cited_mismatch")
		at.logInfof("📋 advisory: %s cited %s (direction mismatch; plan v%d).", d.Action, res.Cited, ap.Version)
	}
	// P5.5 — capture the citation so the next position-open stamps its plan link.
	at.lastCitation = planCitation{
		planVersion: ap.Version,
		scenarioID:  res.Cited, // "" when off-plan
		matched:     res.Matched,
		valid:       true,
	}
}
