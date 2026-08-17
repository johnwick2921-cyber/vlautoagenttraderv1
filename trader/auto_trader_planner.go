package trader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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
		exact := at.pinExactModel(at.mcpClient, modelID)
		at.logInfof("🧠 planner model: empty binding → using primary, pinned %q", exact)
		return at.mcpClient, exact
	}

	client := mcp.NewAIClientByProvider(modelID)
	if client == nil {
		exact := at.pinExactModel(at.mcpClient, primaryModel)
		at.logWarnf("🧠 planner model %q unresolved by the registry → falling back to primary %q", modelID, exact)
		return at.mcpClient, exact
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
	exact := at.pinExactModel(client, modelID)
	at.logInfof("🧠 planner model resolved (pinned): %q", exact)
	return client, exact
}

// pinExactModel resolves a possibly-alias model id to the EXACT model string
// (§125 — never stamp a provider alias on a plan): prefer the client's own
// resolved model, else map the alias to its provider default, else keep it as-is
// with a warning. Also records the model + resets the matched-random stats window
// on a model change (§128 — no pooling across models).
func (at *AutoTrader) pinExactModel(client mcp.AIClient, modelID string) string {
	exact := modelID
	if client != nil {
		if rm := strings.TrimSpace(client.ResolvedModel()); rm != "" && !mcp.IsProviderAlias(rm) {
			exact = rm
		}
	}
	if mcp.IsProviderAlias(exact) {
		if def := mcp.DefaultModelForAlias(exact); def != "" {
			at.logInfof("🧠 model %q is a provider alias → pinned exact %q", exact, def)
			exact = def
		} else {
			at.logWarnf("⚠️ planner model %q is a provider alias and could not be pinned to an exact string", exact)
		}
	}
	at.maybeResetStatsOnModelChange(exact)
	return exact
}

// maybeResetStatsOnModelChange resets the matched-random window when the pinned
// planner model changes (§128). Idempotent; the first-ever pin only records.
func (at *AutoTrader) maybeResetStatsOnModelChange(exactModel string) {
	if at.store == nil || strings.TrimSpace(exactModel) == "" {
		return
	}
	const key = "dayplan_pinned_model"
	prev, _ := at.store.GetSystemConfig(key)
	if prev == exactModel {
		return
	}
	if prev != "" {
		if err := at.store.MatchedRandom().ResetWindow(); err != nil {
			at.logWarnf("📊 stats-window reset on model change failed: %v", err)
		} else {
			at.logInfof("📊 planner model changed %q → %q — matched-random stats window RESET (no cross-model pooling).", prev, exactModel)
		}
	}
	_ = at.store.SetSystemConfig(key, exactModel)
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
	now := time.Now()
	reg := at.sessionRegistry(now) // W8 — admin registry from system_config (fallback default)
	tradeDate := plannerTradeDateCT(now)
	for i := range reg.Sessions {
		s := &reg.Sessions[i]
		// W1 — fire the read ONLY inside this session's own read window on a live
		// (holiday-aware) CME day. inSessionReadWindow stops the Sunday-17:00 NY
		// read; IsCMEOpen stops holiday/weekend reads independently of loop order.
		// W9 — the strategy's sessions_enabled subset (default [NY]) + per-session
		// Enable override gate which sessions THIS trader reads, on top of the
		// registry Enabled flag.
		// PART A — one resolver for both enable layers (explicit per-session override
		// wins; else registry + sessions_enabled). Enabling a session NEVER backfills
		// a past read: the read still only fires inside this session's own read
		// window, and the plan-store dedupe keeps it to once per session-day.
		if runnable, _ := at.sessionRunnable(s); !runnable {
			continue
		}
		if !kernel.IsCMEOpen(now) || !inSessionReadWindow(now, s.ReadCT, s.WindowEndCT) {
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
		if detail, dead := at.describeActivePlanDeath(existing); dead {
			replanCap := at.replanCapFor(s.Name) // W9 — per-session override wins
			// A death must never again be an unexplained line. On 2026-08-16 six
			// plans died in 25 minutes and the only record was five identical
			// "DIED" lines with no condition and no price.
			at.logInfof("🗓️ plan %s %s v%d DIED — %s. Re-planning (cap %d/session).",
				tradeDate, s.Name, existing.Version, detail.Killer, replanCap)
			for _, l := range detail.Levels {
				at.logInfof("🗓️   ↳ %s", l)
			}
			if existing.Version-1 >= replanCap {
				at.writeNoTradePlan(s.Name, tradeDate,
					fmt.Sprintf("re-plans exhausted (%d/%d) after %d deaths — last: %s",
						existing.Version-1, replanCap, existing.Version, detail.Killer))
			} else {
				at.warnIfReplanOrphansOverlays(existing)
				// The guard the owner asked for: once deaths reach the cap, the alert
				// NAMES the killing condition and the price rather than the count.
				if existing.Version-1 >= replanCap-1 {
					at.emitAlert("P1", "plan-death-streak",
						fmt.Sprintf("deaths:%s:%s:v%d", tradeDate, s.Name, existing.Version),
						fmt.Sprintf("%s plan died %d× this session", s.Name, existing.Version),
						fmt.Sprintf("Killed by: %s. This is the last re-plan before the session sits out.", detail.Killer))
				}
				at.runPlannerRead(s.Name, tradeDate) // appends a new version
			}
		}
	}
}

// warnIfReplanOrphansOverlays makes a known data loss AUDIBLE.
//
// Owner overlays are keyed (plan_id, plan_version) and every reader resolves
// them against the LATEST version, so appending a new version silently orphans
// every edit attached to the old one: the card stops showing them and, worse, the
// executor stops citing them — the owner's levels quietly leave the plan the bot
// is trading. Nothing carries, re-points or rebases them.
//
// The real fix is a rebase (an index-based RFC-6902 patch like /levels/3 cannot
// simply be re-pointed at a doc whose level array changed) and is sized M in the
// report. It is deliberately NOT attempted here: getting it wrong would move an
// owner's edit onto a DIFFERENT level, which is worse than dropping it.
//
// What ships now is the honesty: a P1 alert naming exactly how many edits the
// re-plan is about to strand. The defect is live in code but has never fired in
// production — plan_overlays holds one demo-seeded row — so the first REAL owner
// edit is the one at risk, and this is what will tell them.
func (at *AutoTrader) warnIfReplanOrphansOverlays(row *store.PlanDB) {
	if at.store == nil || row == nil {
		return
	}
	overlays, err := at.store.Plan().ListOverlays(row.PlanID, row.Version)
	if err != nil || len(overlays) == 0 {
		return
	}
	at.logErrorf("🚨 re-plan %s v%d strands %d owner overlay(s) — edits are keyed to the version they were made on.",
		row.PlanID, row.Version, len(overlays))
	at.emitAlert("P1", "overlays-orphaned",
		fmt.Sprintf("orphan:%s:v%d", row.PlanID, row.Version),
		fmt.Sprintf("%d owner edit(s) dropped by the re-plan", len(overlays)),
		fmt.Sprintf("Your edits were attached to %s v%d. The plan has been re-read as v%d, and overlays do not carry across versions — re-apply them on the new version if you still want them.",
			row.PlanID, row.Version, row.Version+1))
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
	now := time.Now()
	// Use the rule of the session whose plan this IS, not whichever session happens
	// to be live at the check. activeSessionName returns "" outside every window,
	// which silently fell back to the strategy-level rule — benign while every
	// session shares "2x5m", but the moment one sets "15m-close" (need=1) the wrong
	// rulebook would decide whether that session's plan lives or dies.
	rule := at.acceptanceRuleFor(row.Session) // W15.B — per-session
	// Judge the plan ONLY on what happened AFTER it was written. Feeding the full
	// 2000-bar cache asked "was this level ever traded through in the last ~33
	// hours", which is true of nearly any level once real history exists — so a
	// brand-new plan was born dead and burned the whole re-plan budget in minutes
	// (2026-08-16:ASIA v1..v5, then a levels:null NO-TRADE plan).
	sinceMs := row.CreatedAt.UnixMilli()
	if row.CreatedAt.IsZero() {
		sinceMs = 0 // unknown write time → fall back to the old whole-window behavior
	}
	return kernel.PlanIsDeadSince(doc, bars, rule, sinceMs, now.UnixMilli())
}

// describeActivePlanDeath is activePlanIsDead plus the EVIDENCE. Same window,
// same timeframe, same verdict — the explanation is derived from the decision
// rather than reconstructed alongside it, so the two cannot disagree.
func (at *AutoTrader) describeActivePlanDeath(row *store.PlanDB) (kernel.PlanDeathDetail, bool) {
	if market.FuturesBarsProvider == nil || row == nil {
		return kernel.PlanDeathDetail{}, false
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil {
		return kernel.PlanDeathDetail{}, false
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return kernel.PlanDeathDetail{}, false
	}
	now := time.Now()
	sinceMs := row.CreatedAt.UnixMilli()
	if row.CreatedAt.IsZero() {
		sinceMs = 0
	}
	return kernel.DescribePlanDeath(doc, bars, at.acceptanceRuleFor(row.Session), sinceMs, now.UnixMilli())
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
	// W6 — P0 plan-died → no-trade alert.
	at.emitAlert("P0", "plan-died", "notrade:"+tradeDate+":"+session,
		fmt.Sprintf("%s plan died — sitting out", session), reason)
}

// plannerReadInFlight claims a planner read for one (trade_date, session) for as
// long as the AI call is running. W16/R6.
//
// The dedupe in maybeRunSessionReads is `GetLatestPlanForSession(...) == nil`,
// but that check and the AppendPlan that satisfies it are separated by a FULL AI
// round trip (up to 3 attempts — seconds to minutes). Nothing held a claim across
// that window.
//
// Intra-trader this was safe: each trader drives one sequential loop goroutine
// (auto_trader.go:776-790), so its cycles cannot overlap. The exposure is
// CROSS-TRADER: plan identity is MakePlanID(tradeDate, session) — session-GLOBAL,
// not per-trader — so two day-plan traders on the same symbol both see "no plan
// yet" at the read time, both pay for a full planner call, and both append a
// version of the same session's plan. That is the live configuration today (two
// MNQ day-plan traders).
//
// All traders share one process, so a process-wide claim covers both axes. The
// key deliberately excludes the trader id: two traders reading the SAME session
// is exactly what must be collapsed to one call.
var plannerReadInFlight sync.Map // "tradeDate:session" -> struct{}

// claimPlannerRead returns false when another read for this session is already
// running. The winner must call releasePlannerRead.
func claimPlannerRead(key string) bool {
	_, loaded := plannerReadInFlight.LoadOrStore(key, struct{}{})
	return !loaded
}

func releasePlannerRead(key string) { plannerReadInFlight.Delete(key) }

// runPlannerRead assembles the input package, calls the pinned planner client,
// and persists the plan (or a fail-closed NO-TRADE plan).
func (at *AutoTrader) runPlannerRead(session, tradeDate string) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	// W16/R6 — one planner call per (trade_date, session) at a time. Claimed here
	// rather than at the call sites so no future caller can forget it.
	key := store.MakePlanID(tradeDate, session)
	if !claimPlannerRead(key) {
		at.logInfof("🗓️ planner read for %s already in flight — skipping duplicate call.", key)
		return
	}
	defer releasePlannerRead(key)
	client, modelID := at.resolvePlannerClient()
	if client == nil {
		at.logErrorf("🗓️ planner: no client resolved for %s %s", tradeDate, session)
		return
	}
	input := at.assemblePlannerInput(session, tradeDate)
	prompt := kernel.BuildPlannerPrompt(input)
	hash := shortHash(prompt)
	// W3 — HARD red-news blackout lines auto-written into the plan (§80).
	t1Lines := kernel.T1NoTradeLines(input.Calendar)
	// W11 — carry the frozen indicator mirror + ai_config hash to the write site.
	at.runPlannerReadCore(session, tradeDate, modelID, hash, input.IndicatorsBlock, input.AIConfigHash, func() (string, error) {
		return client.CallWithMessages(plannerSystemPrompt, prompt)
	}, t1Lines...)
}

// runPlannerReadCore is the testable core: ≤2 retries, then FAIL-CLOSED to a
// NO-TRADE plan (never a stale plan, never nothing). Writes the append-only plan
// row. Returns (version, lifecycle, err).
func (at *AutoTrader) runPlannerReadCore(session, tradeDate, modelID, promptHash, indicatorsBlock, aiConfigHash string, call func() (string, error), extraNoTrade ...string) (int, string, error) {
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

	// W3 — auto-write the HARD red-news no-trade blackouts into the plan (§80),
	// deduped. The fail-closed NO-TRADE plan already sits out the whole session.
	if doc != nil && len(extraNoTrade) > 0 {
		have := map[string]bool{}
		for _, nt := range doc.NoTrade {
			have[nt] = true
		}
		for _, nt := range extraNoTrade {
			if !have[nt] {
				doc.NoTrade = append(doc.NoTrade, nt)
				have[nt] = true
			}
		}
	}

	lifecycle := "active"
	trigger := session + "_scheduled_read"
	if doc == nil {
		doc = kernel.NoTradePlanDoc(fmt.Sprintf("read failed after retries: %v", lastErr))
		lifecycle = "no_trade"
		trigger = "planner_fail_closed"
		at.logErrorf("🚨 PLANNER FAIL-CLOSED %s %s: %v — writing a NO-TRADE plan (never stale, never uncalibrated).", tradeDate, session, lastErr)
		telemetry.IncGateBlock(at.id, "planner_fail_closed")
		// W6 — P0 read-fail / fail-closed alert.
		at.emitAlert("P0", "read-fail", "failclosed:"+tradeDate+":"+session,
			fmt.Sprintf("%s planner fail-closed — NO-TRADE", session), "read failed after retries")
	}

	// W9 — scenario_cap: keep at most N scenarios (default 3 = the schema hardcap,
	// so a no-op unless the owner lowered it). Applied post-parse so the executor
	// prompt reflects the cap.
	if doc != nil {
		if cap := at.scenarioCap(); cap > 0 && len(doc.Scenarios) > cap {
			doc.Scenarios = doc.Scenarios[:cap]
		}
	}
	docJSON, _ := json.Marshal(doc)
	version, err := at.store.Plan().AppendPlan(&store.PlanDB{
		PlanID:          store.MakePlanID(tradeDate, session),
		StrategyID:      at.id,
		TradeDate:       tradeDate,
		Session:         session,
		TriggerReason:   trigger,
		Lifecycle:       lifecycle,
		ModelID:         modelID,
		PromptHash:      promptHash,
		IndicatorsBlock: indicatorsBlock, // W11 — frozen at read time (replay-safe)
		AIConfigHash:    aiConfigHash,
		DarkRegimeCount: at.lastRegimeHealth.DarkCount, // P2
		Degraded:        at.lastRegimeHealth.Degraded,
		Doc:             string(docJSON),
	})
	if err != nil {
		at.logErrorf("🗓️ planner: write plan row failed for %s %s: %v", tradeDate, session, err)
		return 0, lifecycle, err
	}
	at.logInfof("🗓️ PLAN written %s %s v%d (model %s, lifecycle %s, prompt %s, ai_config %s)", tradeDate, session, version, modelID, lifecycle, promptHash, aiConfigHash)
	// W6 — P1 plan-born/armed alert (active plans only; fail-closed already alerted P0).
	if lifecycle == "active" {
		at.emitAlert("P1", "armed", fmt.Sprintf("planborn:%s:%s:%d", tradeDate, session, version),
			fmt.Sprintf("%s plan v%d armed", session, version), fmt.Sprintf("model %s", modelID))
	}
	return version, lifecycle, nil
}

// resolveSessionPlanCfg resolves the effective planner config for a session:
// strategy-level day_plan values + the per-session override (min_grade). Nil /
// unset fields fall back to the spec defaults, so a default config reproduces the
// prior behavior byte-for-byte (max_levels 8, no min_grade filter, D/4h/1h/15m).
// Pure — unit-tested without an AutoTrader.
func resolveSessionPlanCfg(dp *store.DayPlanConfig, session string) (maxLevels int, minGrade string, timeframes []string) {
	maxLevels = kernel.DefaultMaxLevels
	timeframes = []string{"D", "4h", "1h", "15m"}
	if dp == nil {
		return maxLevels, minGrade, timeframes
	}
	if dp.MaxLevels > 0 {
		maxLevels = dp.MaxLevels
	}
	if len(dp.PlannerTimeframes) > 0 {
		timeframes = dp.PlannerTimeframes
	}
	for _, so := range dp.Sessions {
		if so.Session == session && so.MinGrade != nil {
			minGrade = *so.MinGrade
		}
	}
	return maxLevels, minGrade, timeframes
}

// assemblePlannerInput builds the input package from stored + cached data (the
// 16:55 read builds entirely from stored data). Digests + owner note arrive with
// P3.6; regime daily/1h fields degrade to n/a until those TFs are fetched. It
// HONORS the day_plan config (max_levels, per-session min_grade, timeframes) —
// edits apply at the NEXT read (never mid-plan).
func (at *AutoTrader) assemblePlannerInput(session, tradeDate string) kernel.PlannerInput {
	symbol := at.futuresSymbol()
	now := time.Now()
	reg := at.sessionRegistry(now) // W8

	var dp *store.DayPlanConfig
	if at.config.StrategyConfig != nil {
		dp = at.config.StrategyConfig.DayPlan
	}
	maxLevels, minGrade, timeframes := resolveSessionPlanCfg(dp, session)

	var bars []market.Kline
	if market.FuturesBarsProvider != nil {
		bars = market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount)
	}
	var extra []kernel.DetectedLevel
	if kernel.NakedPOCProvider != nil {
		extra = kernel.NakedPOCProvider(symbol)
	}
	scored, price, dATR := kernel.AssembleScoredLevels(bars, reg, symbol, maxLevels, now, extra...)

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

	// Per-session min_grade filter (owner levels grade A → always survive).
	scored = kernel.FilterLevelsByMinGrade(scored, minGrade)

	var daily, hour1, min5, min5Long []market.Kline
	if market.FuturesBarsProvider != nil {
		daily = market.FuturesBarsProvider(symbol, "1d", 300)
		hour1 = market.FuturesBarsProvider(symbol, "1h", 300)
		min5 = market.FuturesBarsProvider(symbol, "5m", 300)      // recent (~1 day) → RV recent
		min5Long = market.FuturesBarsProvider(symbol, "5m", 3000) // multi-day → RV baseline
	}
	// W10 — supply the realized-vol baseline (was never fed → RV stuck "warming").
	// Same 5m estimator as the recent value; VIX stays honest n/a (no feed).
	rvBaseline, _ := kernel.RVBaselineFrom5m(min5Long, 20, 5)
	// W11b — supply overnight-gap inputs (prior close + session open, ×ATR) from the
	// daily bars (was never fed → the gap field stayed inert).
	priorClose, sessionOpen := kernel.PriorCloseSessionOpen(daily)
	regime := kernel.ComputeRegime(kernel.RegimeInputs{
		Price: price, DailyBars: daily, Hour1Bars: hour1, Min5Bars: min5, RVBaseline20d: rvBaseline,
		PriorClose: priorClose, SessionOpen: sessionOpen,
	})

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
	// W16/R4 — digests are per-TRADER: the text reports this trader's entries and
	// realized P&L, so another trader's day must never seed this planner read.
	sessionDigests, _ := at.store.Digest().SessionDigests(at.id, symbol, tradeDate)
	dailies, _ := at.store.Digest().RecentDailies(at.id, symbol, 7)
	digestChain := kernel.BuildDigestChain(sessionDigests, dailies)

	// StructureSummary declares which timeframes the planner read (honors the
	// configured planner_timeframes). Real per-TF structure lines land later; for
	// now the header makes the read-set explicit + config-driven.
	structure := make([]string, 0, len(timeframes))
	for _, tf := range timeframes {
		structure = append(structure, tf+": structure read")
	}

	// W11 — INDICATOR MIRROR: render the executor's per-TF indicator state + the
	// ai_config fingerprint (both frozen onto the plan row downstream).
	indicatorsBlock, aiConfigHash := at.renderIndicatorMirror(symbol)

	// P2 — DARK REGIME: name what the planner could NOT see, alert on it, and carry
	// the verdict to the write site so it lands on the plan row.
	at.lastRegimeHealth = kernel.AssessRegime(regime, 0)
	if at.lastRegimeHealth.DarkCount > 0 {
		body := at.lastRegimeHealth.AlertBody()
		level := "P1"
		if at.lastRegimeHealth.Degraded {
			level = "P0" // a half-blind plan is a decision-quality event, not FYI
		}
		at.logWarnf("🌑 dark regime at the %s read: %s", session, body)
		at.emitAlert(level, "regime-dark",
			fmt.Sprintf("regime-dark:%s:%s", tradeDate, session),
			fmt.Sprintf("%s plan: %d/%d regime fields dark", session, at.lastRegimeHealth.DarkCount, kernel.RegimeFieldCount),
			body)
	}

	return kernel.PlannerInput{
		TradeDate:        tradeDate,
		Session:          session,
		ReadKind:         session + " scheduled read (stored+cached data)",
		Price:            price,
		DATR:             dATR,
		Regime:           regime,
		Levels:           scored,
		StructureSummary: structure,
		Calendar:         calEvents,
		DigestChain:      digestChain,
		Warming:          warming,
		IndicatorsBlock:  indicatorsBlock,
		AIConfigHash:     aiConfigHash,
	}
}

// maybeWriteDigests writes the 3-line session digest at each enabled session's
// close and the daily roll-up at the trade-date close (16:00 CT). Idempotent
// (SaveIfAbsent) → restart-safe. GATED on day_plan → dormant by default.
func (at *AutoTrader) maybeWriteDigests() {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	now := time.Now()
	reg := at.sessionRegistry(now) // W8
	tradeDate := plannerTradeDateCT(now)
	symbol := at.futuresSymbol()
	sinceMs := kernel.CMESessionDayStart(now).UnixMilli()
	// W16/R4 — scope the P&L to the ACTIVE ACCOUNT too, matching every other
	// session-day activity call (loop.go, auto_trader_session.go). Without it a
	// two-account trader's digest mixed both accounts' numbers.
	pnl, entries, _ := at.store.Position().GetSessionDayActivity(at.id, sinceMs, at.currentAccountName())

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
			TraderID: at.id,
			Symbol:   symbol, TradeDate: tradeDate, Session: s.Name, Kind: "session", Text: text, CreatedAt: now.UnixMilli(),
		}); wrote {
			at.logInfof("📓 session digest written %s %s.", tradeDate, s.Name)
		}
	}

	// W1 — daily roll-up in the [15:00,16:00) RTH-close→break window, where
	// tradeDate + the P&L window are still the CLOSING day's (they roll at 17:00).
	// Reachable Mon–Fri; idempotent (SaveIfAbsent). W9 — gated on evening_digest
	// (default true; the end-of-day roll-up IS the "evening digest" toggle).
	if inDailyRollWindow(now) && at.eveningDigestEnabled() {
		sessions, _ := at.store.Digest().SessionDigests(at.id, symbol, tradeDate)
		text := kernel.FormatDailyDigest(tradeDate, "", len(sessions), entries, pnl)
		if wrote, _ := at.store.Digest().SaveIfAbsent(&store.DigestDB{
			TraderID: at.id,
			Symbol:   symbol, TradeDate: tradeDate, Kind: "daily", Text: text, CreatedAt: now.UnixMilli(),
		}); wrote {
			at.logInfof("📓 daily digest written %s.", tradeDate)
		}
	}
}

// storedReplanCap resolves the re-plan cap for (plan owner, session) from the
// LIVE stored strategy config — the same resolver GET /api/plan/today uses, so
// the executor prompt and the owner's card can never narrate different numbers.
//
// It reads the store rather than a cached struct on purpose: a mid-session cap
// edit is exactly the case that exposed the divergence. Note that plans.strategy_id
// holds the TRADER id (the column is misnamed), hence the trader→strategy hop.
// Every failure path falls back to DayPlanConfig's shipped default, which is what
// the literal it replaces assumed anyway.
func storedReplanCap(st *store.Store, traderID, session string) int {
	var dp *store.DayPlanConfig // nil → ReplanCapFor returns the shipped default
	if st == nil || traderID == "" {
		return dp.ReplanCapFor(session)
	}
	tr, err := st.Trader().GetByID(traderID)
	if err != nil || tr == nil || tr.StrategyID == "" {
		return dp.ReplanCapFor(session)
	}
	var strat store.Strategy
	if err := st.GormDB().Where("id = ?", tr.StrategyID).First(&strat).Error; err != nil {
		return dp.ReplanCapFor(session)
	}
	var cfg store.StrategyConfig
	if json.Unmarshal([]byte(strat.Config), &cfg) != nil {
		return dp.ReplanCapFor(session)
	}
	return cfg.DayPlan.ReplanCapFor(session)
}

// installActivePlanProvider wires kernel.ActivePlanProvider to read this store's
// latest ACTIVE plan for the current session (P3.4). no_trade/died plans and
// off-session return nil → the executor prompt is unchanged.
func installActivePlanProvider(st *store.Store) {
	kernel.ActivePlanProvider = func(symbol string) *kernel.ActivePlan {
		now := time.Now()
		reg := loadStoredRegistry(st) // W8 — provider honors the admin registry too
		sess, ok := reg.ActiveSession(now)
		if !ok || !sess.Enabled {
			return nil
		}
		tradeDate := plannerTradeDateCT(now)
		row, err := st.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
		if err != nil || row == nil || row.Lifecycle != "active" {
			return nil
		}
		// W4 — the executor cites the OVERLAY-RESOLVED plan_final (owner edits reach
		// the brain), not the base doc. resolveActivePlanDoc folds overlays + armors.
		doc, ok := resolveActivePlanDoc(st, row)
		if !ok {
			return nil
		}
		// The cap must come from the SAME resolver the card uses. A literal 2 here
		// meant the executor prompt and the dashboard narrated different rulebooks
		// the moment a session overrode replan_cap: on 2026-08-16 the owner raised
		// ASIA to 4 mid-session, so at v3 the card said "replans left 2" while the
		// AI was being told 0.
		replansLeft := storedReplanCap(st, row.StrategyID, sess.Name) - (row.Version - 1)
		if replansLeft < 0 {
			replansLeft = 0
		}
		return &kernel.ActivePlan{Doc: doc, Session: sess.Name, Version: row.Version, ReplansLeft: replansLeft}
	}
}

// resolveActivePlanDoc folds a plan's overlays (RFC-6902) into plan_final and
// armors the result via ValidatePlanDoc (falling back to the base doc on any
// failure) — the SAME resolution GET /api/plan/today does, so the card and the
// executor can never diverge. Returns (doc, ok=false) only when the base itself is
// unparseable.
func resolveActivePlanDoc(st *store.Store, row *store.PlanDB) (kernel.PlanDoc, bool) {
	var base kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &base) != nil {
		return kernel.PlanDoc{}, false
	}
	overlays, _ := st.Plan().ListOverlays(row.PlanID, row.Version)
	if len(overlays) == 0 {
		return base, true
	}
	patches := make([]string, 0, len(overlays))
	for _, o := range overlays {
		patches = append(patches, o.Patch)
	}
	final, _ := kernel.ApplyOverlayPatches([]byte(row.Doc), patches)
	var merged kernel.PlanDoc
	if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDoc(&merged) == nil {
		return merged, true // plan_final
	}
	return base, true // armor: a bad overlay never corrupts the executor's plan
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
