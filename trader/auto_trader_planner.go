package trader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"nofx/logger"
	"os"
	"strconv"
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
// ownerUserID resolves the owning user's id for C2 owner-level scoping (sticky
// levels are per-user now; an empty id degrades to the legacy ” bucket).
func (at *AutoTrader) ownerUserID() string {
	if at.store == nil {
		return ""
	}
	t, err := at.store.Trader().Get(at.id)
	if err != nil || t == nil {
		return ""
	}
	return t.UserID
}

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
	// 4.5 — per-model thinking knobs override the env defaults (best-effort).
	if at.store != nil {
		if row, err := at.store.AIModel().GetByID(modelID); err == nil && row != nil {
			mcp.ApplyThinking(client, row.ThinkingMode, row.ReasoningEffort)
		}
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
	loc := kernel.CTLocation()
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
	at.maybeRunSessionReadsAt(time.Now())
}

// maybeRunSessionReadsAt is maybeRunSessionReads with an injectable clock
// (P0-B: the 16:55 closed-market read and the midnight wrap are time-sensitive
// and unit-tested against a fixed `now`).
func (at *AutoTrader) maybeRunSessionReadsAt(now time.Time) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	reg := at.sessionRegistry(now) // W8 — admin registry from system_config (fallback default)
	for i := range reg.Sessions {
		s := &reg.Sessions[i]
		// W1 — fire the read ONLY inside this session's own read window. The
		// market-open test is made against the SESSION INSTANCE'S OPEN (P0-B):
		// the ASIA read is designed for 16:55 CT, inside the 16:00–17:00 CME
		// maintenance break, and the contract says it builds from STORED data
		// while the market is closed — gating on IsCMEOpen(now) made that read
		// UNREACHABLE (first ASIA read fired 17:12, post-open). Weekend/holiday
		// protection is preserved: a session instance whose OPEN falls on a
		// closed day never reads, and the death-check still runs through the
		// wrapped tail (Friday 00:30 → Thursday 17:00 open → live).
		if !inSessionReadWindow(now, s.ReadCT, s.WindowEndCT) {
			continue
		}
		instOpen, okOpen := kernel.SessionInstanceStart(s, now)
		if !okOpen || !kernel.IsCMEOpen(instOpen) {
			continue
		}
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
		// P0-B — the chain identity is the SESSION INSTANCE's date (wrap-aware),
		// never the midnight-roll calendar date: at 00:30 CT the cycle still
		// belongs to the ASIA instance that opened 17:00 yesterday.
		tradeDate, okDate := kernel.PlanChainTradeDate(s, now)
		if !okDate {
			tradeDate = plannerTradeDateCT(now)
		}
		existing, err := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, s.Name, at.id)
		if err != nil {
			// P0-cleanup — a DB read error must not silently skip the
			// session's plan read (the plan simply never appears).
			at.logErrorf("🚨 planner session-read skipped: GetLatestPlanForTraderSession %s %s failed: %v", tradeDate, s.Name, err)
			telemetry.RecordError(at.id, "plan_read_failed", "GetLatestPlanForTraderSession: "+err.Error(), telemetry.CostDecisionLost)
			continue
		}
		if existing == nil {
			at.runPlannerRead(s.Name, tradeDate) // first read this session-day
			continue
		}
		if existing.Lifecycle != "active" {
			continue // no_trade / died → done for the session
		}
		// C5 (2026-08-25) — DEATH CHECK FIRST: a dead vN must not also fire an
		// MSS wake this cycle — the wake dedupes by (plan,version,event), so a
		// same-cycle re-plan would re-wake on the SAME event under the NEW
		// version and double-append. G4.6 MSS wake runs only when no death was
		// handled below.
		handledDeath := false
		// P3.6 — RE-PLAN ON DEATH (cap replan_cap/session → NO-TRADE).
		if detail, dead := at.describeActivePlanDeath(existing); dead {
			handledDeath = true
			replanCap := at.replanCapFor(s.Name) // W9 — per-session override wins
			// P6 — the budget is measured from the CURRENT chain baseline (an
			// owner reset re-arms it; the default baseline is 1 = the original
			// chain). Every consumer reads the same seam.
			baseline := store.GetResetBaseline(at.store, at.id, tradeDate, s.Name)
			// A death must never again be an unexplained line. On 2026-08-16 six
			// plans died in 25 minutes and the only record was five identical
			// "DIED" lines with no condition and no price.
			at.logInfof("🗓️ plan %s %s v%d DIED — %s. Re-planning (cap %d/session).",
				tradeDate, s.Name, existing.Version, detail.Killer, replanCap)
			for _, l := range detail.Levels {
				at.logInfof("🗓️   ↳ %s", l)
			}
			if !store.MayReplanFrom(existing.Version, baseline, replanCap) {
				at.writeNoTradePlan(s.Name, tradeDate,
					fmt.Sprintf("re-plans exhausted (%d/%d) after %d deaths — last: %s",
						store.ReplansUsedFrom(existing.Version, baseline), replanCap, existing.Version-baseline, detail.Killer))
			} else {
				at.warnIfReplanOrphansOverlays(existing)
				// The guard the owner asked for: once deaths reach the cap, the alert
				// NAMES the killing condition and the price rather than the count.
				if !store.MayReplanFrom(existing.Version+1, baseline, replanCap) {
					at.emitAlert("P1", "plan-death-streak",
						fmt.Sprintf("deaths:%s:%s:v%d", tradeDate, s.Name, existing.Version),
						fmt.Sprintf("%s plan died %d× this session", s.Name, existing.Version),
						fmt.Sprintf("Killed by: %s. This is the last re-plan before the session sits out.", detail.Killer))
				}
				// P0.4-G (2026-08-25) — carry the killer + the prior plan's
				// levels into the re-plan: the map keeps continuity (no more
				// rebuilt-from-scratch level sets) and a fired flip's bias is
				// enforced at write.
				at.runPlannerReadWithCtx(s.Name, tradeDate, detail.Killer, priorPlanLevelLines(existing))
				// ITEM 4 — the owner's levels are sticky: re-establish them on the
				// version just written, re-anchored by price. Anything that cannot
				// be re-anchored is parked for review, never dropped.
				if fresh, fErr := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, s.Name, at.id); fErr == nil && fresh != nil {
					at.carryOwnerEditsInto(fresh.PlanID, existing.Version, fresh.Version)
				}
			}
		}
		if !handledDeath {
			// G4.6 (addendum, regime wave) — a fresh structure MSS on the
			// plan's bias TF is the FOURTH planner wake-up (one per MSS
			// event, deduped).
			at.maybeWakePlannerOnMSS(s.Name, tradeDate, existing)
			// W6 (2026-08-25) — level events (fresh zones/FVGs/OBs/iFVGs the
			// plan never saw + seated-level invalidations) are the FIFTH
			// wake-up. Death-first ordering is preserved: this runs only
			// when no death was handled above, and it shares the MSS wake's
			// min-interval throttle + per-session budget.
			at.maybeWakePlannerOnLevelEvents(s.Name, tradeDate, existing)
		}
	}
}

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
	at.logInfof("🗓️ re-plan %s v%d carries %d owner overlay(s) forward by price identity.",
		row.PlanID, row.Version, len(overlays))
}

// carryOwnerEditsInto re-establishes the PREVIOUS version's owner edits on the
// version just written (ITEM 4, 2026-08-17).
//
// Owner levels are sticky by contract. Before this, a re-plan silently stranded
// them: overlays are keyed (plan_id, plan_version) and every reader resolves
// against the LATEST version, so the owner's levels quietly left the plan the bot
// was trading. The carry re-anchors by PRICE, never by array index — replaying
// `/levels/3` against a re-planned doc would move an edit onto a different level,
// which is worse than losing it.
//
// Anything that cannot be re-anchored (structural edits, and deletes the planner
// has undone) is recorded for REVIEW and alerted, never silently dropped.
func (at *AutoTrader) carryOwnerEditsInto(planID string, oldVersion, newVersion int) {
	if at.store == nil || oldVersion <= 0 || newVersion <= oldVersion {
		return
	}
	overlays, err := at.store.Plan().ListOverlays(planID, oldVersion)
	if err != nil || len(overlays) == 0 {
		return
	}
	oldRow, err := at.store.Plan().GetPlan(planID, oldVersion)
	if err != nil || oldRow == nil {
		return
	}
	newRow, err := at.store.Plan().GetPlan(planID, newVersion)
	if err != nil || newRow == nil {
		return
	}
	var oldBase, newDoc kernel.PlanDoc
	if json.Unmarshal([]byte(oldRow.Doc), &oldBase) != nil || json.Unmarshal([]byte(newRow.Doc), &newDoc) != nil {
		return
	}
	patches := make([]string, 0, len(overlays))
	for _, ov := range overlays {
		patches = append(patches, ov.Patch)
	}
	// Resolve what the owner actually SAW on the old version. If that resolution
	// fails, oldFinal stays equal to oldBase, which carries nothing — fail-safe
	// (an edit is never mis-applied) but worth saying out loud, because silently
	// carrying nothing is the very failure this whole item exists to end.
	oldFinal := oldBase
	resolved := false
	if merged, mErr := json.Marshal(oldBase); mErr == nil {
		if applied, _ := kernel.ApplyOverlayPatches(merged, patches); len(applied) > 0 {
			var f kernel.PlanDoc
			// Re-validation of an overlay-resolved doc is an integrity check, not
			// the write-time policy gate — use the HARD ceilings (12/5) so a plan
			// validly written under a raised max_levels/scenario_cap never fails
			// here.
			if json.Unmarshal(applied, &f) == nil && kernel.ValidatePlanDocWithCaps(&f, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios) == nil {
				oldFinal, resolved = f, true
			}
		}
	}
	if !resolved {
		at.logErrorf("🚨 %s v%d: plan_final would not resolve, so %d owner overlay(s) cannot be carried into v%d — re-apply them by hand.",
			planID, oldVersion, len(overlays), newVersion)
		at.emitAlert("P1", "overlays-need-review",
			fmt.Sprintf("review:%s:v%d", planID, newVersion),
			fmt.Sprintf("%d owner edit(s) could not be carried into v%d", len(overlays), newVersion),
			"The previous version's edits could not be resolved, so nothing was carried forward. Re-apply them on the new version.")
		return
	}

	res := kernel.CarryOwnerEdits(oldBase, oldFinal, newDoc, patches)

	if res.Patch != "" {
		if _, aErr := at.store.Plan().AppendOverlay(&store.PlanOverlayDB{
			PlanID: planID, PlanVersion: newVersion,
			Patch: res.Patch, Origin: "owner-carried",
		}); aErr != nil {
			at.logErrorf("🚨 carrying %d owner level(s) into %s v%d FAILED: %v",
				len(res.Carried), planID, newVersion, aErr)
			at.emitAlert("P1", "overlays-orphaned",
				fmt.Sprintf("orphan:%s:v%d", planID, oldVersion),
				fmt.Sprintf("%d owner edit(s) could not be carried into v%d", len(res.Carried), newVersion),
				"The re-plan could not re-apply your levels. Re-add them on the new version.")
			return
		}
		at.logInfof("🗓️ carried %d owner level(s) into %s v%d.", len(res.Carried), planID, newVersion)
	}

	if len(res.Uncarried) > 0 {
		lines := make([]string, 0, len(res.Uncarried))
		for _, u := range res.Uncarried {
			lines = append(lines, "• "+u.Summary)
		}
		at.storeUncarriedEdits(planID, newVersion, res.Uncarried)
		at.logErrorf("🚨 %d owner edit(s) could NOT carry into %s v%d — awaiting review.",
			len(res.Uncarried), planID, newVersion)
		at.emitAlert("P1", "overlays-need-review",
			fmt.Sprintf("review:%s:v%d", planID, newVersion),
			fmt.Sprintf("%d edit(s) could not carry into v%d — review", len(res.Uncarried), newVersion),
			strings.Join(lines, "\n"))
	}
}

// storeUncarriedEdits parks the review list where the card can read it. Follows
// the scenario_status precedent (system_config keyed by plan), so no migration.
func (at *AutoTrader) storeUncarriedEdits(planID string, version int, items []kernel.UncarriedEdit) {
	blob, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = at.store.SetSystemConfig(store.UncarriedEditsKey(planID, version), string(blob))
}

// (A7/F13, fail-register wave) activePlanIsDead removed — a second, WEAKER
// death definition with zero production callers; the live path is
// describeActivePlanDeath (structured death → flip → legacy consumption).

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
	// P0.3 (2026-08-19) — the planner's own stated death/flip conditions are now
	// MACHINE-EVALUATED (they used to be display-only prose; on 2026-08-18 the
	// plan's death text fired at ~09:00 and nothing re-planned). The structured
	// predicate runs first; the legacy all-levels-consumed check stays as the
	// fallback for old stored plans.
	// G7 (2026-08-21) — freshness-gated: a condition whose rule-TF series is
	// provably stale is SKIPPED (logged flip_eval_skipped), never guessed. A
	// fully-stale cycle defers the whole death check to the next fresh one.
	killer, fired, skipped := kernel.PlanDeathOrFlipSinceFresh(doc, bars, at.acceptanceRuleFor(row.Session), sinceMs, now.UnixMilli())
	for _, s := range skipped {
		at.logWarnf("flip_eval_skipped plan=%s v%d %s", row.PlanID, row.Version, s)
	}
	if fired {
		return kernel.PlanDeathDetail{Killer: killer, Price: priceOf(bars)}, true
	}
	if len(skipped) > 0 {
		return kernel.PlanDeathDetail{}, false
	}
	return kernel.DescribePlanDeath(doc, bars, at.acceptanceRuleFor(row.Session), sinceMs, now.UnixMilli())
}

// executorPlanDeadReason (C6 / S2-1, 2026-08-25) — the executor evaluates the
// ACTIVE plan's death with the SAME machine predicate the planner uses, so a
// dead-but-not-yet-replanned plan stops producing entries the moment the
// condition fires. A day_plan trader with no active plan (no_trade / budget
// exhausted / never written) is refused too: planless futures entries are not
// allowed. Empty string = no block.
func (at *AutoTrader) executorPlanDeadReason() string {
	sc := at.GetStrategyConfig()
	if sc == nil || sc.DayPlan == nil || !sc.DayPlan.PlanEnabled {
		return ""
	}
	if at.store == nil {
		return "day_plan on but store unavailable — planless entries refused"
	}
	now := time.Now()
	reg := at.sessionRegistry(now)
	sess, ok := reg.ActiveSession(now)
	if !ok {
		return ""
	}
	tradeDate, okDate := kernel.PlanChainTradeDate(sess, now)
	if !okDate {
		tradeDate = plannerTradeDateCT(now)
	}
	row, err := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, sess.Name, at.id)
	if err != nil || row == nil {
		return "no active day plan for this session (day_plan on) — planless entries refused"
	}
	if row.Lifecycle != "active" {
		return fmt.Sprintf("plan lifecycle %q — entries refused", row.Lifecycle)
	}
	if detail, dead := at.describeActivePlanDeath(row); dead {
		return "active plan is MACHINE-DEAD (" + detail.Killer + ") — entries refused until the planner re-plans"
	}
	return ""
}

// priceOf returns the latest closed close of the bar series (0 if empty).
func priceOf(bars []market.Kline) float64 {
	if len(bars) == 0 {
		return 0
	}
	return bars[len(bars)-1].Close
}

// warnFlipDeathSanity (P0.4-F, 2026-08-25) — advisory checks at plan write:
// the 11-plan audit found 7/11 active plans with a flip that can never fire
// (death preempts it), flip==death degeneracy, or a flip/death anchored on a
// level absent from the plan's own list. These are WARN-only by design — the
// machine evaluator + fail-closed path stay the enforcers; this just makes the
// defect visible in the journal at write time instead of after a session.
func (at *AutoTrader) warnFlipDeathSanity(d *kernel.PlanDoc) {
	if d == nil {
		return
	}
	levelPrice := func(p float64) bool {
		for _, l := range d.Levels {
			if math.Abs(l.Price-p) <= 3.0 {
				return true
			}
		}
		return false
	}
	if d.DeathStructured != nil && d.DeathStructured.Price > 0 && !levelPrice(d.DeathStructured.Price) {
		at.logWarnf("⚠️ flip/death sanity: death{price %.2f} matches NO level in this plan's list (orphan anchor).", d.DeathStructured.Price)
	}
	if d.FlipStructured != nil && d.FlipStructured.Price > 0 && !levelPrice(d.FlipStructured.Price) {
		at.logWarnf("⚠️ flip/death sanity: flip{price %.2f} matches NO level in this plan's list (orphan anchor).", d.FlipStructured.Price)
	}
	if d.DeathStructured != nil && d.FlipStructured != nil && d.DeathStructured.Price > 0 && d.FlipStructured.Price > 0 {
		dd, ff := d.DeathStructured, d.FlipStructured
		if math.Abs(dd.Price-ff.Price) <= 0.01 && dd.Side == ff.Side && dd.Rule == ff.Rule {
			at.logWarnf("⚠️ flip/death sanity: flip == death (%.2f %s %s) — flip_to fires at the same tick the plan dies; void.", ff.Price, ff.Side, ff.Rule)
		}
		// death preempts flip when death sits closer in the same direction
		// with an easier/equal rule.
		easier := func(rule string) int {
			switch rule {
			case "15m_close":
				return 1
			case "5m_close":
				return 2
			default: // 2x5m
				return 3
			}
		}
		if dd.Side == ff.Side && easier(dd.Rule) <= easier(ff.Rule) {
			up := dd.Side == "above"
			if (up && dd.Price <= ff.Price) || (!up && dd.Price >= ff.Price) {
				at.logWarnf("⚠️ flip/death sanity: death (%.2f %s %s) preempts flip (%.2f %s %s) — flip likely unreachable.", dd.Price, dd.Side, dd.Rule, ff.Price, ff.Side, ff.Rule)
			}
		}
	}
}

// noTradeLevelMap assembles the CURRENT detector/scorer output as plan levels
// for a NO-TRADE doc (P7) — the same pipeline every other surface uses
// (bars → AssembleScoredLevels at the resolved proximity + max_levels), with the
// sticky owner levels prepended like the planner input. Returns nil when the
// detector genuinely has nothing (no bars provider / no bars), which the doc
// turns into the explicit "detector data unavailable" line.
func (at *AutoTrader) noTradeLevelMap(session string) []kernel.PlanLevel {
	symbol := at.futuresSymbol()
	if market.FuturesBarsProvider == nil {
		return nil
	}
	bars := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return nil
	}
	now := time.Now()
	maxLevels, minGrade, _ := resolveSessionPlanCfg(at.dayPlanCfg(), session)
	// R2 4.7 (2026-08-25) — fail-closed maps obey min_grade: a NO-TRADE doc's
	// level map must match what an active plan would have carried.
	scored, _, _ := kernel.AssembleScoredLevelsMinGrade(at.id, bars, at.sessionRegistry(now), symbol, maxLevels, now, at.proximityFilterATR(), minGrade)

	out := make([]kernel.PlanLevel, 0, len(scored)+4)
	if owned, err := at.store.OwnerLevel().ListActiveForUser(at.ownerUserID(), symbol); err == nil {
		for _, o := range owned {
			label := "👤 " + o.Label
			out = append(out, kernel.PlanLevel{Price: o.Price, Label: label, Grade: "A", Instruction: "monitor"})
		}
	}
	for _, s := range scored {
		out = append(out, kernel.PlanLevel{Price: s.Price, Label: s.Label, Grade: s.Grade, Instruction: "monitor"})
	}
	return out
}

// writeNoTradePlan appends a NO-TRADE plan (re-plans exhausted) + an alert event.
func (at *AutoTrader) writeNoTradePlan(session, tradeDate, reason string) {
	// P7 — levels are market FACTS; the plan is an opinion about them. A no-trade
	// decision must never erase the map: the fail-closed doc carries the current
	// detector/scorer output (owner sticky levels included) so the card keeps
	// showing the map under the NO-TRADE banner. Unavailable detector data says so.
	doc := kernel.NoTradePlanDocWithLevels(reason, at.noTradeLevelMap(session))
	docJSON, _ := json.Marshal(doc)
	_, err := at.store.Plan().AppendPlan(&store.PlanDB{
		PlanID: at.store.Plan().ResolvePlanID(tradeDate, session, at.id), StrategyID: at.id,
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
// CROSS-TRADER: two day-plan traders on the same symbol both see "no plan
// yet" at the read time, both pay for a full planner call, and both append a
// version of the same session's plan. That is the live configuration today (two
// MNQ day-plan traders).
//
// All traders share one process, so a process-wide claim covers both axes. The
// key deliberately excludes the trader id: two traders reading the SAME session
// is exactly what must be collapsed to one call. (P0-A2: the PLAN ROWS are now
// trader-scoped — MakePlanIDForTrader — while this claim collapses only the AI
// call, not the identity.)
var plannerReadInFlight sync.Map // "tradeDate:session" -> struct{}

// claimPlannerRead returns false when another read for this session is already
// running. The winner must call releasePlannerRead.
func claimPlannerRead(key string) bool {
	_, loaded := plannerReadInFlight.LoadOrStore(key, struct{}{})
	return !loaded
}

func releasePlannerRead(key string) { plannerReadInFlight.Delete(key) }

// PlannerReadInFlight reports whether a planner read for this trader's chain is
// currently running (the claim is held). The API exposes it as the card's
// "reading…" state; ForceReset uses it to avoid a silent claim-skip — the
// 2026-08-18 live finding: the owner's reset read silently no-op'd because a
// death re-plan held the claim, and the fresh plan was written by that OTHER
// read (wrong trigger_reason, zero UI feedback).
func (at *AutoTrader) PlannerReadInFlight(tradeDate, session string) bool {
	_, held := plannerReadInFlight.Load(store.MakePlanIDForTrader(at.id, tradeDate, session))
	return held
}

// runPlannerReadWithTriggerClaimed is runPlannerReadWithTrigger returning whether
// THIS call claimed the read (false = another read was already in flight and
// this one skipped). The wrapper keeps the old signature for existing callers.
func (at *AutoTrader) runPlannerReadWithTriggerClaimed(session, tradeDate, triggerOverride string) bool {
	return at.runPlannerReadWithTriggerClaimedCtx(session, tradeDate, triggerOverride, "", nil, true)
}

// runPlannerReadWithTriggerClaimedCtx (P0.4-G, 2026-08-25) is the claimed read
// with the prior-plan context: priorKiller (the dead plan's killer line) and
// priorLevels (the previous version's levels for map continuity). A flip-fired
// killer carries a MANDATORY bias the new plan must honor; the prior levels keep
// the map from being rebuilt from scratch every re-plan.
//
// failClosed=true (scheduled reads, death re-plans, owner re-reads/resets): a
// read that fails every retry writes the terminal NO-TRADE marker.
// failClosed=false (W6 wake reads): the wake is OPPORTUNISTIC — if the re-read
// fails, the still-active plan keeps trading and nothing is written.
func (at *AutoTrader) runPlannerReadWithTriggerClaimedCtx(session, tradeDate, triggerOverride, priorKiller string, priorLevels []string, failClosed bool) bool {
	if !at.dayPlanEnabled() || at.store == nil {
		return false
	}
	key := store.MakePlanIDForTrader(at.id, tradeDate, session)
	if !claimPlannerRead(key) {
		at.logInfof("🗓️ planner read for %s already in flight — skipping duplicate call.", key)
		return false
	}
	defer releasePlannerRead(key)
	// U1 3.2 — never call the LLM on an empty/stale bar window (the 08-19
	// outage produced 0-scenario fail-closed stubs this way). No plan row is
	// written and no budget consumed; the read window retries next cycle.
	if !at.plannerPreflight(session, tradeDate) {
		return false
	}
	client, modelID := at.resolvePlannerClient()
	if client == nil {
		// C9 (2026-08-25) — HONEST failure: return false so the reread path
		// reports the real outcome instead of a silent "success" that wrote
		// nothing (the fail-closed NO-TRADE write lives in the retry core,
		// which this branch never reached).
		at.logErrorf("🗓️ planner: no client resolved for %s %s", tradeDate, session)
		return false
	}
	input := at.assemblePlannerInputWithCtx(session, tradeDate, priorKiller, priorLevels)
	prompt := kernel.BuildPlannerPrompt(input)
	hash := shortHash(prompt)
	// W3 — HARD red-news blackout lines auto-written into the plan (§80).
	t1Lines := kernel.T1NoTradeLines(input.Calendar)
	// P0.1/P0.2 (2026-08-19) — write-time facts: both-side levels, continuation
	// scenario on gaps. PDH/PDL come from the detector universe (seated or raw).
	facts := kernel.PlanFacts{Price: input.Price, DATR: input.DATR}
	// 8.4 — machine grades from the Go-ranked candidate table, keyed by rounded
	// price so the write-site stamp can match the model's levels.
	machineGrades := map[float64]string{}
	machineLabels := map[float64]string{} // P0.4-H: price → detector label
	record := func(price float64, grade string) {
		if grade == "" {
			return
		}
		k := math.Round(price*100) / 100
		// Collision rule: keep the STRONGER grade per rounded price — an
		// owner "A" prepended first is never clobbered by a same-price
		// detector entry later in the slice.
		if old, ok := machineGrades[k]; !ok || kernel.GradeRank(grade) > kernel.GradeRank(old) {
			machineGrades[k] = grade
		}
	}
	recordLabel := func(price float64, label string) {
		if label == "" {
			return
		}
		k := math.Round(price*100) / 100
		if _, ok := machineLabels[k]; !ok {
			machineLabels[k] = label
		}
	}
	for _, l := range input.Levels {
		switch l.Kind {
		case kernel.KindPDH:
			facts.PDH = l.Price
		case kernel.KindPDL:
			facts.PDL = l.Price
		}
		record(l.Price, l.Grade)
		recordLabel(l.Price, l.Label)
	}
	// The HTF zones section rows lost the top-N seat race and are not in
	// input.Levels — but the model reads them and may write them into the
	// plan (live proof: ASIA v3's Supply·4h row went unstamped). Merge their
	// grades into the same map so those rows get stamped too.
	for _, z := range input.HTFZones {
		record(z.Price, z.Grade)
		recordLabel(z.Price, z.Label)
	}
	requiredBias := kernel.FlipToDirection(priorKiller)
	// W11 — carry the frozen indicator mirror + ai_config hash to the write site.
	at.runPlannerReadCoreWithFactsGrades(session, tradeDate, triggerOverride, modelID, hash, input.IndicatorsBlock, input.AIConfigHash, requiredBias, facts, machineGrades, machineLabels, failClosed, func() (string, error) {
		return client.CallWithMessages(plannerSystemPrompt, prompt)
	}, t1Lines...)
	return true
}

// runPlannerRead assembles the input package, calls the pinned planner client,
// and persists the plan (or a fail-closed NO-TRADE plan).
func (at *AutoTrader) runPlannerRead(session, tradeDate string) {
	at.runPlannerReadWithTrigger(session, tradeDate, "")
}

// runPlannerReadWithCtx (P0.4-G, 2026-08-25) is the re-plan form: the dead
// plan's killer line + its levels ride into the prompt (flip bias enforced at
// write, level-set continuity).
func (at *AutoTrader) runPlannerReadWithCtx(session, tradeDate, priorKiller string, priorLevels []string) {
	_ = at.runPlannerReadWithTriggerClaimedCtx(session, tradeDate, "", priorKiller, priorLevels, true)
}

// priorPlanLevelLines renders the previous version's levels as "price label"
// lines for the continuity block (empty when the stored doc is unreadable).
func priorPlanLevelLines(row *store.PlanDB) []string {
	if row == nil {
		return nil
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil {
		return nil
	}
	lines := make([]string, 0, len(doc.Levels))
	for _, l := range doc.Levels {
		lines = append(lines, fmt.Sprintf("%.2f %s", l.Price, l.Label))
	}
	return lines
}

// runPlannerReadWithTrigger is runPlannerReadWithTriggerClaimed with the old
// signature (claim result discarded) for existing callers.
func (at *AutoTrader) runPlannerReadWithTrigger(session, tradeDate, triggerOverride string) {
	_ = at.runPlannerReadWithTriggerClaimed(session, tradeDate, triggerOverride)
}

// runPlannerReadCore is the testable core: ≤2 retries, then FAIL-CLOSED to a
// NO-TRADE plan (never a stale plan, never nothing). Writes the append-only plan
// row. Returns (version, lifecycle, err).
func (at *AutoTrader) runPlannerReadCore(session, tradeDate, modelID, promptHash, indicatorsBlock, aiConfigHash string, call func() (string, error), extraNoTrade ...string) (int, string, error) {
	return at.runPlannerReadCoreWithTrigger(session, tradeDate, "", modelID, promptHash, indicatorsBlock, aiConfigHash, call, extraNoTrade...)
}

// runPlannerReadCoreWithTrigger is runPlannerReadCore with an explicit
// trigger_reason and NO facts (schema-only validation — legacy callers/tests).
func (at *AutoTrader) runPlannerReadCoreWithTrigger(session, tradeDate, triggerOverride, modelID, promptHash, indicatorsBlock, aiConfigHash string, call func() (string, error), extraNoTrade ...string) (int, string, error) {
	return at.runPlannerReadCoreWithFacts(session, tradeDate, triggerOverride, modelID, promptHash, indicatorsBlock, aiConfigHash, kernel.PlanFacts{}, call, extraNoTrade...)
}

// runPlannerReadCoreWithFacts is the production core: same retry/fail-closed
// loop PLUS the P0.1/P0.2 facts validation (both-side levels, continuation
// scenario on gaps, duplicate/target reachability). Legacy callers keep the
// facts-free signature (schema-only validation).
func (at *AutoTrader) runPlannerReadCoreWithFacts(session, tradeDate, triggerOverride, modelID, promptHash, indicatorsBlock, aiConfigHash string, facts kernel.PlanFacts, call func() (string, error), extraNoTrade ...string) (int, string, error) {
	return at.runPlannerReadCoreWithFactsGrades(session, tradeDate, triggerOverride, modelID, promptHash, indicatorsBlock, aiConfigHash, "", facts, nil, nil, true, call, extraNoTrade...)
}

// runPlannerReadCoreWithFactsGrades is the production core + the machine-grade
// stamp (master-audit finding 8.4): machineGrades maps rounded level price →
// deterministic detector grade from the Go-ranked candidate table; plan levels
// that match get their machine grade persisted for the card to display beside
// the model-written one.
func (at *AutoTrader) runPlannerReadCoreWithFactsGrades(session, tradeDate, triggerOverride, modelID, promptHash, indicatorsBlock, aiConfigHash, requiredBias string, facts kernel.PlanFacts, machineGrades map[float64]string, machineLabels map[float64]string, failClosed bool, call func() (string, error), extraNoTrade ...string) (int, string, error) {
	// H4/H5 — validation must accept EXACTLY what the config allows: the resolved
	// max_levels / scenario_cap (hard ceilings 12/5). Before this the parse
	// hardcoded 8/3, so raising either setting made EVERY read fail-closed into a
	// NO-TRADE plan + P0 alert — the upper half of the UI range was unreachable.
	maxLevels, _, _ := resolveSessionPlanCfg(at.dayPlanCfg(), session)
	scenarioCap := at.scenarioCap()

	var doc *kernel.PlanDoc
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ { // 1 + ≤2 retries
		raw, err := call()
		if err != nil {
			lastErr = err
			continue
		}
		d, perr := kernel.ParsePlanDocCapped(raw, maxLevels, scenarioCap)
		if perr != nil {
			lastErr = perr
			continue
		}
		// W6-D (2026-08-25) — the G5 write-time demotion is REMOVED: stamping
		// scenarios consumed + C at plan birth contradicted P1c (a consumed
		// level's RETOUCH is the tradeable role-flip event) and poisoned 88%
		// of scenarios at write (42/48 on 2026-08-25). The LIVE per-cycle
		// evaluator still tracks consumption — scenarios demote only when a
		// level is actually re-touched after the plan is born, never at
		// birth. demoteConsumedScenarios stays for reference/tests only.
		// P0.4-C (2026-08-24) — the model may write near-duplicate levels (2.13
		// pts apart killed ASIA v2 with the duplicate-seat rule). Collapse them
		// with the same cluster tolerance the scorer uses, instead of burning
		// the retries → fail-closed session.
		if collapsed, n := kernel.CollapsePlanLevels(d.Levels, kernel.LevelClusterTicks*0.25); n > 0 {
			d.Levels = collapsed
			at.logWarnf("📐 plan level auto-collapse: %d near-duplicate level(s) merged at %s write.", n, session)
		}
		// P0.4-G (2026-08-25) — flip-fired re-plan bias enforcement: the prior
		// plan's flip ALREADY FIRED on machine-evaluated bars, so the new plan's
		// bias MUST match the flipped direction. Live bug: ASIA v3's flip fired
		// "→ bias long" but v4 came back short-biased — the flip was honored by
		// the evaluator and ignored by the re-planner.
		if requiredBias != "" && strings.ToLower(strings.TrimSpace(d.Bias.Direction)) != requiredBias {
			lastErr = fmt.Errorf("prior plan flip already fired → bias %s is MANDATORY, got %q — the flip cannot be re-written away", requiredBias, d.Bias.Direction)
			continue
		}
		// P0.4-H (2026-08-25) — label provenance: a plan level whose price
		// matches a machine-table row must NOT re-label it as a DIFFERENT
		// structural anchor. Live bug: LONDON v1 labeled 29297.75 "PDH" when
		// the table row at that price is a zone and the true prior-day high
		// was 29290.5 — the flip anchor rode a phantom label.
		if mis := kernel.MislabeledStructuralLevels(d, machineLabels); len(mis) > 0 {
			lastErr = fmt.Errorf("level label provenance: %s — copy the machine table's label for these prices", strings.Join(mis, "; "))
			continue
		}
		// P0.1/P0.2 (2026-08-19) — facts rules: both-side levels, continuation
		// scenario on a gap out of the prior range, no duplicate seats, reachable
		// targets. A plan that fails these is NOT shipped (retry → fail-closed).
		if verr := kernel.ValidatePlanDocWithFacts(d, facts, maxLevels, scenarioCap); verr != nil {
			lastErr = verr
			continue
		}
		// P0.4-F (2026-08-25) — advisory flip/death sanity (never a gate; the
		// machine evaluator is the enforcer). 11-plan audit found 7/11 active
		// plans with flip unreachable/void: death preempting flip, flip==death,
		// or flip anchored on a level absent from the plan's own list.
		at.warnFlipDeathSanity(d)
		doc = d
		break
	}
	// A3 (F5, fail-register wave): a plan whose death/flip is prose-only has NO
	// machine evaluation — the owner must know (the prompt line says it too).
	if doc != nil && doc.DeathStructured == nil {
		at.logWarnf("📜 plan death is PROSE-ONLY (no structured death{} object) — AI-judged, not machine-evaluated; only the all-levels-consumed fallback protects the chain.")
	}
	// C1 (F3) — DUAL-ACCEPT window: scenarios missing confirm{} are accepted
	// with a WARN for the first CONFIRM_GRACE_SESSIONS (default 3) distinct
	// plan sessions after this feature landed; afterwards the plan is REJECTED
	// back into the retry loop (the planner model may lag the contract).
	if doc != nil {
		missing := 0
		for _, sc := range doc.Scenarios {
			if sc.Confirm == nil {
				missing++
			}
		}
		if missing > 0 {
			if at.confirmGraceExhausted() {
				lastErr = fmt.Errorf("%d scenario(s) missing the REQUIRED confirm{} object (grace window over)", missing)
				at.logWarnf("📐 plan REJECTED: %v", lastErr)
				doc = nil
			} else {
				at.logWarnf("📐 confirm-grace: %d scenario(s) missing confirm{} — accepted during the grace window (CONFIRM_GRACE_SESSIONS).", missing)
			}
		} else {
			at.noteConfirmCompliantSession()
		}
	}

	// 8.4 — stamp the deterministic machine grade beside the model-written one
	// (matched back to the Go-ranked candidate table by price). The card can then
	// show both, so a model's A next to a machine C is visible at a glance.
	if doc != nil && len(machineGrades) > 0 {
		for i := range doc.Levels {
			if g, ok := machineGrades[math.Round(doc.Levels[i].Price*100)/100]; ok && g != "" {
				doc.Levels[i].MachineGrade = g
			}
		}
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
	if strings.TrimSpace(triggerOverride) != "" {
		trigger = triggerOverride
	}
	if doc == nil {
		// W6-C (2026-08-25) — wake reads are NON-fatal: a failed wake re-read
		// must NOT no-trade a session whose active plan is still alive (live
		// bug 2026-08-25: a seated-OB invalidation wake's re-read timed out and
		// the fail-closed marker killed a healthy ASIA v1). Wake failures keep
		// the active plan and simply skip the wake.
		if !failClosed {
			at.logWarnf("🗓️ wake re-read failed for %s %s (benign — active plan kept): %v", tradeDate, session, lastErr)
			return 0, "kept_active", nil
		}
		// P7 — the fail-closed doc still carries the map: levels from the current
		// detector/scorer output (same pipeline), scenarios empty, explicit reason.
		doc = kernel.NoTradePlanDocWithLevels(
			fmt.Sprintf("read failed after retries: %v", lastErr), at.noTradeLevelMap(session))
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
		PlanID:          at.store.Plan().ResolvePlanID(tradeDate, session, at.id),
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

// structureSummaryLines fetches one bar request per CONFIGURED planner timeframe
// and returns one honest prompt line per TF: "<tf>: structure read" when bars
// came back, "<tf>: unavailable" otherwise. "D" maps to the provider's "1d"
// interval; every other configured TF is requested verbatim. A nil fetch (no
// bars provider) marks every TF unavailable — the planner is told the read-set
// truth instead of a hardcoded claim that diverges from planner_timeframes (H9).
// consumedLevels (G5) computes which plan levels are CONSUMED right now using
// the same facts evaluator the card reads (EvaluateLevelFacts → StillValid).
func (at *AutoTrader) consumedLevels(bars []market.Kline, levels []kernel.PlanLevel, rule string, now int64) map[float64]bool {
	if len(bars) == 0 {
		return nil
	}
	out := map[float64]bool{}
	for _, l := range levels {
		if kernel.EvaluateLevelFacts(bars, l.Price, kernel.DirAbove, rule, 3, now).StillValid {
			continue
		}
		out[l.Price] = true
	}
	return out
}

// demoteConsumedScenarios (G5) applies the write-time demotion.
func (at *AutoTrader) demoteConsumedScenarios(session string, d *kernel.PlanDoc) int {
	if market.FuturesBarsProvider == nil {
		return 0
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return 0
	}
	consumed := at.consumedLevels(bars, d.Levels, at.acceptanceRuleFor(session), time.Now().UnixMilli())
	return kernel.MarkConsumedScenarios(d, consumed)
}

// structureSummaryLines (G2, regime wave 2026-08-21) — the REAL machine
// structure detector replaces the old "read/unavailable" placeholder: per-TF
// trend + newest swing + the latest BOS/CHoCH/MSS/SWEEP event, computed from
// the same 1m cache the executor reads (kernel.StructureSnapshot).
func structureSummaryLines(fetch func(tf string, count int) []market.Kline, timeframes []string) []string {
	lines := make([]string, 0, len(timeframes)+1)
	var bars1m []market.Kline
	if fetch != nil {
		bars1m = fetch(kernel.AISVPBarInterval, kernel.AISVPBarCount)
	}
	snap := kernel.StructureSnapshot(bars1m, time.Now().UnixMilli())
	for _, tf := range timeframes {
		st, ok := snap[tf]
		if !ok {
			lines = append(lines, tf+": unavailable")
			continue
		}
		label := st.Trend
		if st.Swing != nil {
			label += fmt.Sprintf(" (%s %.2f @%s)", st.Swing.Kind, st.Swing.Price, kernel.ClockCT(time.UnixMilli(st.Swing.TimeMs)))
		}
		lines = append(lines, tf+": "+label)
	}
	var lastEv *kernel.StructureEvent
	var lastTF string
	for _, tf := range kernel.StructureTFs {
		st, ok := snap[tf]
		if !ok {
			continue
		}
		for i := range st.LastEvents {
			if lastEv == nil || st.LastEvents[i].TimeMs > lastEv.TimeMs {
				e := st.LastEvents[i]
				lastEv = &e
				lastTF = tf
			}
		}
	}
	if lastEv != nil {
		lines = append(lines, fmt.Sprintf("last event: %s-%s %s @%s", lastEv.Type, lastEv.Dir, lastTF, kernel.ClockCT(time.UnixMilli(lastEv.TimeMs))))
	}
	return lines
}

// assemblePlannerInput builds the input package from stored + cached data (the
// 16:55 read builds entirely from stored data). Digests + owner note arrive with
// P3.6; regime daily/1h fields degrade to n/a until those TFs are fetched. It
// HONORS the day_plan config (max_levels, per-session min_grade, timeframes) —
// edits apply at the NEXT read (never mid-plan).
func (at *AutoTrader) assemblePlannerInput(session, tradeDate string) kernel.PlannerInput {
	return at.assemblePlannerInputWithCtx(session, tradeDate, "", nil)
}

// assemblePlannerInputWithCtx (P0.4-G, 2026-08-25) is assemblePlannerInput with
// the prior-plan context for re-plans: the dead plan's killer line and its
// levels (map continuity).
func (at *AutoTrader) assemblePlannerInputWithCtx(session, tradeDate, priorKiller string, priorLevels []string) kernel.PlannerInput {
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
	// G2/G3 (2026-08-24) — per-TF swing/zone detection on the CONFIGURED
	// planner timeframes: real 1h/4h support/resistance now enters the
	// candidate pool (before this, every detector ran on the 1m slice only).
	// G2.1 — one observability line per read so a missing-zone question is
	// answered with facts, not theories.
	var htfLevels []kernel.DetectedLevel
	if market.FuturesBarsProvider != nil {
		htfLevels = kernel.DetectHTFLevels(func(tf string, count int) []market.Kline {
			return market.FuturesBarsProvider(symbol, tf, count)
		}, timeframes, symbol, now)
		if len(htfLevels) > 0 {
			counts := map[string]int{}
			for _, l := range htfLevels {
				counts[l.Label]++
			}
			at.logInfof("🗺️ G2 HTF detection @%s: %d level(s) %v", session, len(htfLevels), counts)
		} else {
			at.logInfof("🗺️ G2 HTF detection @%s: 0 levels — no swings/zones found on %v", session, timeframes)
		}
		extra = append(extra, htfLevels...)
	}
	scored, price, dATR := kernel.AssembleScoredLevelsMinGrade(at.id, bars, reg, symbol, maxLevels, now, at.proximityFilterATR(), minGrade, extra...)
	// 1h wave (2026-08-25) — the ranked table's HTF seats guarantee an in-band
	// 1h S/D zone when one exists. Gated by the seat_1h_zone knob (default ON).
	if dp != nil && dp.Seat1HZoneEnabled() {
		scored = kernel.Seat1HZone(scored, maxLevels)
	}

	// G2.2 (2026-08-24) — the nearest in-band HTF ZONES become their own prompt
	// section: the top-8 seat race hides them, but the model must know where the
	// 1h/4h bases are.
	var htfZoneScored []kernel.ScoredLevel
	if len(htfLevels) > 0 && price > 0 && dATR > 0 {
		zones := make([]kernel.DetectedLevel, 0, len(htfLevels))
		for _, l := range htfLevels {
			switch l.Kind {
			case kernel.KindSupply, kernel.KindDemand, kernel.KindFVG, kernel.KindOB:
				zones = append(zones, l)
			}
		}
		if len(zones) > 0 {
			// ScoreLevels filters to the ±proximity band and returns nearest-first.
			zs := kernel.ScoreLevels(zones, price, dATR, nil, 4, at.proximityFilterATR())
			// 1h wave (2026-08-25) — the cap-4 MUST keep a 1h S/D zone when one
			// is in band, so the prompt's conditional 1h mandate has data to
			// point at. Gated by the seat_1h_zone knob (default ON).
			if at.dayPlanCfg().Seat1HZoneEnabled() {
				zs = kernel.Seat1HZone(zs, 4)
			}
			htfZoneScored = zs
		}
	}

	// P3.6-C — STICKY OWNER LEVELS: always seated, tagged 👤, persisted across
	// sessions. Prepended so they lead the ranked table.
	if owned, err := at.store.OwnerLevel().ListActiveForUser(at.ownerUserID(), symbol); err == nil && len(owned) > 0 {
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
		// R2 4.6/4.7 (2026-08-25) — the filter now runs inside
		// AssembleScoredLevelsMinGrade on a 2× pool so the cut REFILLS seats
		// from the same side instead of thinning the table; this final pass
		// keeps the owner-prepended levels above the floor.
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
			loc := kernel.CTLocation()
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

	// H9 — the structure summary must only claim what was actually fetched. Each
	// CONFIGURED timeframe gets one honest line: "4h: structure read" when bars
	// came back, "4h: unavailable" when the provider is down or the TF is dark.
	// Before this the fetch was hardcoded 1d/1h/5m while the lines asserted the
	// configured set — the planner was told it read structure it never saw.
	var structureFetch func(tf string, count int) []market.Kline
	if market.FuturesBarsProvider != nil {
		structureFetch = func(tf string, count int) []market.Kline {
			return market.FuturesBarsProvider(symbol, tf, count)
		}
	}
	structure := structureSummaryLines(structureFetch, timeframes)

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

	// G5 (regime wave 2026-08-21) — consumed levels at read time, listed so the
	// planner works around them.
	var consumedLines []string
	if len(bars) > 0 {
		lvls := make([]kernel.PlanLevel, len(scored))
		for i, s := range scored {
			lvls[i] = kernel.PlanLevel{Price: s.Price, Label: s.Label}
		}
		consumed := at.consumedLevels(bars, lvls, at.acceptanceRuleFor(session), now.UnixMilli())
		for _, s := range scored {
			if consumed[s.Price] {
				consumedLines = append(consumedLines, fmt.Sprintf("%.2f %s", s.Price, s.Label))
			}
		}
	}

	return kernel.PlannerInput{
		TradeDate:        tradeDate,
		Session:          session,
		Now:              now, // P0 timezone — the planner's labelled CT clock
		ReadKind:         session + " scheduled read (stored+cached data)",
		Price:            price,
		DATR:             dATR,
		Regime:           regime,
		Levels:           scored,
		HTFZones:         htfZoneScored,
		StructureSummary: structure,
		ConsumedLevels:   consumedLines,
		Calendar:         calEvents,
		DigestChain:      digestChain,
		Warming:          warming,
		IndicatorsBlock:  indicatorsBlock,
		AIConfigHash:     aiConfigHash,
		// H4/H5 — the prompt asks for EXACTLY what validation accepts: the
		// resolved max_levels / scenario_cap (never a hardcoded 8/3 the owner
		// cannot raise).
		MaxLevels:       maxLevels,
		ScenarioCap:     at.scenarioCap(),
		PriorPlanKiller: priorKiller,
		PriorPlanLevels: priorLevels,
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
		if runnable, _ := at.sessionRunnable(s); !runnable {
			continue
		}
		// Wrap-aware "is this session closed right now": the old test was
		// ctMinutesNow(now) >= end — TRUE for the whole rest of the DAY once the
		// end minute passed, so ASIA (end 02:00) read as "closed" at 21:00 CT
		// while its evening leg was RUNNING, and a mid-session digest with
		// mid-session P&L was written for the just-started instance (class 2).
		// Not-in-window is the correct predicate: the most recent instance has
		// ended, and sessionChainDate keys the digest to THAT instance's date.
		if _, ok := hhmmToMin(s.WindowEndCT); !ok {
			continue // malformed registry times — never digest on garbage
		}
		if s.InWindow(now) {
			continue // session running — not closed yet
		}
		// P0-B — a session digest carries the SESSION INSTANCE's date, so the
		// next read of the SAME session picks it up (ASIA closes 02:00 CT, after
		// the midnight roll; its digest must land on the day the plan was keyed).
		sessTradeDate := sessionChainDate(s, now)
		text := kernel.FormatSessionDigest(s.Name, sessTradeDate, "", entries, pnl)
		if wrote, _ := at.store.Digest().SaveIfAbsent(&store.DigestDB{
			TraderID: at.id,
			Symbol:   symbol, TradeDate: sessTradeDate, Session: s.Name, Kind: "session", Text: text, CreatedAt: now.UnixMilli(),
		}); wrote {
			at.logInfof("📓 session digest written %s %s.", sessTradeDate, s.Name)
		}
	}

	// W1 — daily roll-up in the [15:00,16:00) RTH-close→break window, where
	// tradeDate + the P&L window are still the CLOSING day's (they roll at 17:00).
	// Reachable Mon–Fri; idempotent (SaveIfAbsent). W9 — gated on evening_digest
	// (default true; the end-of-day roll-up IS the "evening digest" toggle).
	if inDailyRollWindow(now) && at.eveningDigestEnabled() {
		sessions, _ := at.store.Digest().SessionDigests(at.id, symbol, tradeDate)
		text := kernel.FormatDailyDigest(tradeDate, "", len(sessions), entries, pnl)
		if ll := at.learningDigestLine(); ll != "" {
			text += "\n" + ll
		}
		if el := telemetry.ErrorDigestLine(at.id); el != "" {
			text += "\n" + el
		}
		if el := telemetry.ErrorDigestLine(at.id); el != "" {
			text += "\n" + el
		}
		if wrote, _ := at.store.Digest().SaveIfAbsent(&store.DigestDB{
			TraderID: at.id,
			Symbol:   symbol, TradeDate: tradeDate, Kind: "daily", Text: text, CreatedAt: now.UnixMilli(),
		}); wrote {
			at.logInfof("📓 daily digest written %s.", tradeDate)
		}
	}
}

// learningDigestLine renders the learning-loop line (avg MAE/MFE + adherence
// grades, linkable to plan versions) from the last graded closed positions.
// P0-cleanup (2026-08-19) — MAE/MFE + grades now reach the digest.
func (at *AutoTrader) learningDigestLine() string {
	if at.store == nil {
		return ""
	}
	rows, err := at.store.Position().GetGradedClosedPositions(at.id, 20)
	if err != nil || len(rows) == 0 {
		return ""
	}
	trades := make([]kernel.LearningTrade, 0, len(rows))
	for _, p := range rows {
		trades = append(trades, kernel.LearningTrade{
			MAE: p.MAE, MFE: p.MFE, Grade: p.AdherenceGrade, PlanVersion: p.PlanVersion,
		})
	}
	return kernel.LearningLine(trades)
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

// installActivePlanProvider registers THIS trader's active-plan provider with
// the kernel (P3.4). P0-A (2026-08-18): this used to write a PROCESS-GLOBAL
// singleton closing over whichever trader reached the sync.Once first — so a
// second day-plan trader's executor read the first trader's plan, session
// enablement and budget. Now the provider is registered per-traderID and its
// plan lookup is trader-scoped: a trader can NEVER receive another trader's
// plan row. no_trade/died plans and off-session return nil → the executor
// prompt is unchanged.
func installActivePlanProvider(at *AutoTrader, st *store.Store) {
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{
		SessionRegistry: func() kernel.SessionRegistry { return at.sessionRegistry(time.Now()) },
		ActivePlan: func(symbol string) *kernel.ActivePlan {
			now := time.Now()
			reg := at.sessionRegistry(now) // W8 — provider honors the admin registry too
			sess, ok := reg.ActiveSession(now)
			if !ok {
				return nil
			}
			// H8 — the executor must honor the SAME resolver the read scheduler uses.
			// The registry flag is a DEFAULT, never a veto: a session the owner switched
			// on at the strategy level must reach the executor (before this it was
			// written by the read and then dropped here).
			if runnable, _ := at.sessionRunnable(sess); !runnable {
				return nil
			}
			// P0-B — chain identity is the session INSTANCE's date (wrap-aware):
			// at 00:30 CT the provider still resolves the ASIA instance that
			// opened 17:00 yesterday, never tomorrow's empty chain.
			tradeDate, okDate := kernel.PlanChainTradeDate(sess, now)
			if !okDate {
				tradeDate = plannerTradeDateCT(now)
			}
			// P0-A — trader-scoped lookup: THIS trader's row only.
			row, err := st.Plan().GetLatestPlanForTraderSession(tradeDate, sess.Name, at.id)
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
			// AI was being told 0. P6 — the budget is measured from the chain baseline
			// (an owner reset re-arms it), same seam the death path reads.
			replansLeft := store.ReplansLeftFrom(row.Version,
				store.GetResetBaseline(st, at.id, row.TradeDate, sess.Name),
				storedReplanCap(st, row.StrategyID, sess.Name))
			// P0-cleanup — decision records carry the full plan attribution
			// (plan_id, plan_version, overlay_version).
			overlayVersion := 0
			if ovs, err := st.Plan().ListOverlays(row.PlanID, row.Version); err == nil {
				overlayVersion = len(ovs)
			}
			return &kernel.ActivePlan{Doc: doc, Session: sess.Name, Version: row.Version, ReplansLeft: replansLeft, BirthMs: row.CreatedAt.UnixMilli(), PlanID: row.PlanID, OverlayVersion: overlayVersion}
		},
	})
	// P0-A — loud startup/runtime assertion: if MORE THAN ONE day-plan trader
	// has registered providers, announce per-trader isolation so the state is
	// visible in the journal on day one, not discovered after the fact.
	if n := kernel.TraderPlanProviderCount(); n > 1 {
		at.logWarnf("🔀 %d day-plan traders registered — per-trader plan/provider isolation is in force (P0-A); no shared plan or provider state.", n)
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
	// H4/H5 — re-validation integrity check at the HARD ceilings (12/5): a plan
	// validly written under raised caps must survive overlay resolution.
	if vErrDoc := func() error {
		if err := json.Unmarshal(final, &merged); err != nil {
			return err
		}
		return kernel.ValidatePlanDocWithCaps(&merged, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios)
	}(); vErrDoc != nil {
		// A8 (F14): the fallback-to-base is no longer silent — the owner's
		// overlay is NOT in what the executor reads, and they must know.
		// (free function — package logger, still WARN → log_events sink)
		logger.Warnf("⚠️ merged plan+overlay FAILED re-validation for %s v%d (%v) — falling back to the BASE plan; the overlay edits are NOT active.", row.PlanID, row.Version, vErrDoc)
	}
	if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDocWithCaps(&merged, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios) == nil {
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
	if !kernel.HasTraderPlanProvider(at.id) {
		return
	}
	ap := kernel.ActivePlanFor(at.id, at.futuresSymbol())
	if ap == nil {
		return
	}
	res := kernel.ClassifyCitation(d.Action, d.CitedScenario, ap.Doc)
	// B3 (F6): structural verdict — entry in the cited scenario's band, SL/TP
	// consistent. Forward-only; "" = unknown/fail-open.
	band := kernel.CitationStructure(d.Action, d.CitedScenario, ap.Doc, d.Price, d.StopLoss, d.TakeProfit, kernel.PlanDATRFor(at.id))
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
		band:        band,
		valid:       true,
	}
}

// ---- C1 (F3) confirm-grace window ------------------------------------------

// confirmGraceSessions resolves CONFIRM_GRACE_SESSIONS (default 3).
func confirmGraceSessions() int {
	if v := os.Getenv("CONFIRM_GRACE_SESSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 3
}

const confirmGraceKey = "confirm_grace_sessions_seen"

// confirmGraceExhausted reports whether the dual-accept window is over: after
// CONFIRM_GRACE_SESSIONS distinct plan-write sessions, confirm{} is REQUIRED.
func (at *AutoTrader) confirmGraceExhausted() bool {
	if at.store == nil {
		return false
	}
	raw, _ := at.store.GetSystemConfig(confirmGraceKey)
	n, _ := strconv.Atoi(strings.TrimSpace(raw))
	if n >= confirmGraceSessions() {
		return true
	}
	_ = at.store.SetSystemConfig(confirmGraceKey, strconv.Itoa(n+1))
	return false
}

// noteConfirmCompliantSession fast-forwards the grace window once the model
// proves it authors confirm{} — no reason to keep accepting regressions.
func (at *AutoTrader) noteConfirmCompliantSession() {
	if at.store == nil {
		return
	}
	_ = at.store.SetSystemConfig(confirmGraceKey, strconv.Itoa(confirmGraceSessions()))
}
