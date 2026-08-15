package trader

import (
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
	"nofx/telemetry"
)

// W7 — LEVEL-STATE WRITER (the audit's dead wire: store.LevelStateStore —
// times_tested / consumed / freshness / re-arm cooldown — had ZERO production
// callers). Runs each bar-close cycle while a plan is active. It:
//
//   - EnsureLevel   → persists a level's durable identity (type|price-bin),
//                     PRESERVING prior-session state so a burned level stays burned.
//   - MarkConsumed  → when the evaluator shows price accepted THROUGH the level.
//   - RecordPlay    → on a fresh sweep/rejection, decays freshness one grade.
//                     Debounced by re-arm: RecordPlay fires only when ReArmEligible
//                     (persisted cooldown elapsed) — this is the READER of the state.
//
// A burned level re-touched inside the active window emits a telemetry gate-block +
// a P1 alert. It does NOT touch the executor prompt (no golden change); surfacing
// persisted freshness INTO RenderPlanStatus is a deliberate prompt-regression
// follow-up, flagged in the W7 report — not silently done here.
func (at *AutoTrader) recordLevelState() {
	if !at.dayPlanEnabled() || at.store == nil || kernel.ActivePlanProvider == nil {
		return
	}
	symbol := at.config.NinjaTraderSymbol
	if symbol == "" || market.FuturesBarsProvider == nil {
		return
	}
	plan := kernel.ActivePlanProvider(symbol)
	if plan == nil {
		return
	}
	bars := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return
	}
	now := time.Now()
	nowMs := now.UnixMilli()
	_, price, dATR := kernel.AssembleScoredLevels(bars, at.sessionRegistry(now), symbol, 8, now)
	if price <= 0 {
		return
	}
	rule := "2x5m"
	if sc := at.config.StrategyConfig; sc != nil && sc.DayPlan != nil && sc.DayPlan.AcceptanceRule != "" {
		rule = sc.DayPlan.AcceptanceRule
	}

	ls := at.store.LevelState()
	active := kernel.ActivePlanLevels(plan.Doc.Levels, price, dATR, kernel.ActivationWindowK)
	for _, l := range active {
		typ := kernel.LevelTypeFromLabel(l.Label)
		bin := kernel.LevelBinIndex(l.Price)
		key := store.MakeLevelKey(symbol, typ, "", bin)

		// Identity: create fresh (grade→initial freshness) or preserve prior state.
		if err := ls.EnsureLevel(&store.LevelStateDB{
			Symbol:    symbol,
			LevelType: typ,
			BinIndex:  bin,
			Price:     l.Price,
			Freshness: gradeToFreshness(l.Grade),
		}); err != nil {
			continue
		}

		dir := kernel.DirAbove
		if l.Price < price {
			dir = kernel.DirBelow
		}
		f := kernel.EvaluateLevelFacts(bars, l.Price, dir, rule, 3, nowMs)

		cur, err := ls.Get(key)
		if err != nil || cur == nil {
			continue
		}

		// A burned level (persisted consumed/done) re-entering the active window and
		// re-touched is a fact the state is meant to catch — surface it (no prompt change).
		if (cur.Consumed || cur.Freshness == store.FreshnessDone) && f.StillValid {
			telemetry.IncGateBlock(at.id, "level_burned_retouch")
			at.emitAlert("P1", "level-burned", "burned:"+key,
				"Burned level re-touched: "+l.Label, "consumed in a prior session — no fresh play")
			continue
		}

		if !f.StillValid {
			_ = ls.MarkConsumed(key) // accepted through → burned for the day (and forward)
			continue
		}

		// A fresh sweep/rejection decays freshness — but only once per re-arm cooldown.
		// ReArmEligible READS the persisted last_play_ms (the reader wire): true only
		// when not consumed, not done, and the cooldown has elapsed.
		if f.Rejected || f.Swept {
			if ok, _ := store.ReArmEligible(cur, nowMs, store.ReArmCooldownMin, true); ok {
				_, _ = ls.RecordPlay(key, nowMs)
			}
		}
	}
}

// gradeToFreshness maps a plan level's quality grade (A|B|C) to the initial
// freshness a NEW level row starts at, so a C-grade level enters more decayed than
// an A. (EnsureLevel preserves freshness on levels that already exist.)
func gradeToFreshness(grade string) string {
	switch grade {
	case "B":
		return store.FreshnessB
	case "C":
		return store.FreshnessC
	default:
		return store.FreshnessA
	}
}
