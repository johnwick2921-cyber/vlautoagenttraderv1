package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// P2 — THE CLOCK. Bar-close cadence + the skip-while-open gate. Both are GATED on
// day_plan being enabled for a futures trader, so the default (crypto / plan-off)
// keeps the scan-timer loop and same-side-refusal behavior byte-identical —
// additive + dormant until the owner arms day_plan at ★ RESTART 1.

// dayPlanEnabled is the shared activation for every P2 clock behavior: a futures
// (NinjaTrader) trader whose strategy has day_plan enabled. Default false → the
// whole clock is dormant and crypto/plan-off behavior is byte-identical.
func (at *AutoTrader) dayPlanEnabled() bool {
	if at.exchange != "ninjatrader" || at.config.StrategyConfig == nil {
		return false
	}
	dp := at.config.StrategyConfig.DayPlan
	return dp != nil && dp.PlanEnabled
}

// barCloseCadenceActive reports whether this trader should fire cycles on
// primary-TF bar closes instead of the scan timer.
func (at *AutoTrader) barCloseCadenceActive() bool { return at.dayPlanEnabled() }

// skipWhileOpen (P2.2) reports whether the AI decision cycle should be skipped
// because the strategy is already holding a position — calmer and cheaper than
// per-decision same-side refusal. GATED on day_plan → dormant by default. The
// held trade is still managed independently: the NT8 OCO bracket, auto-breakeven,
// and close-sync/reconcile all run outside the decision cycle, so exits and flips
// (bracket/target/stop) are unaffected.
func (at *AutoTrader) skipWhileOpen() (bool, string) {
	if !at.dayPlanEnabled() || at.store == nil {
		return false, ""
	}
	positions, err := at.store.Position().GetOpenPositions(at.id)
	if err != nil || len(positions) == 0 {
		return false, ""
	}
	p := positions[0]
	return true, fmt.Sprintf("%d open (%s %s)", len(positions), p.Symbol, p.Side)
}

// primaryTimeframe is the strategy's primary bar interval (default 5m).
func (at *AutoTrader) primaryTimeframe() string {
	if at.config.StrategyConfig == nil {
		return "5m"
	}
	tf := at.config.StrategyConfig.Indicators.Klines.PrimaryTimeframe
	if tf == "" {
		return "5m"
	}
	return tf
}

func (at *AutoTrader) futuresSymbol() string {
	s := at.config.NinjaTraderSymbol
	if s == "" {
		return "MNQ"
	}
	return s
}

// latestClosedPrimaryBarMs returns the CloseTime (ms) of the most recent CLOSED
// primary-TF bar, ok=false when none is available (provider down / warming).
func (at *AutoTrader) latestClosedPrimaryBarMs() (int64, bool) {
	if market.FuturesBarsProvider == nil {
		return 0, false
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), at.primaryTimeframe(), 5)
	nowMs := time.Now().UnixMilli()
	var latest int64
	for i := range bars {
		if bars[i].CloseTime < nowMs && bars[i].CloseTime > latest {
			latest = bars[i].CloseTime
		}
	}
	if latest == 0 {
		return 0, false
	}
	return latest, true
}

// barCloseGate decides whether a tick should run a decision cycle under bar-close
// cadence, and returns the updated last-close watermark. When cadence is not
// active it ALWAYS runs (scan-timer behavior unchanged). When active it runs only
// once per NEW primary-TF bar close — never mid-bar, and a restart resumes on the
// next close.
func barCloseGate(active bool, lastCloseMs, latestClosedMs int64, haveBar bool) (run bool, newLastMs int64) {
	if !active {
		return true, lastCloseMs
	}
	if !haveBar || latestClosedMs <= lastCloseMs {
		return false, lastCloseMs
	}
	return true, latestClosedMs
}

// ---- P2.3 — day-trader clock: last_entry + eod_flat ------------------------

// hhmmToMin parses "HH:MM" (24h) into minutes-since-midnight; ok=false on bad input.
func hhmmToMin(s string) (int, bool) {
	var h, m int
	n, err := fmt.Sscanf(strings.TrimSpace(s), "%d:%d", &h, &m)
	if err != nil || n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// ctMinutesNow returns now's minutes-since-midnight in America/Chicago.
func ctMinutesNow(now time.Time) int {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.UTC
	}
	ct := now.In(loc)
	return ct.Hour()*60 + ct.Minute()
}

// timeReachedCT reports whether now's CT wall-clock is at/after hhmm.
func timeReachedCT(now time.Time, hhmm string) bool {
	target, ok := hhmmToMin(hhmm)
	if !ok {
		return false
	}
	return ctMinutesNow(now) >= target
}

// effectiveEODFlatCT returns the flat time, pulled IN by a registered half-day
// early-close for the current CME session-day (holiday/half-day awareness via the
// P0 registry, which the P1.8 calendar populates). Empty HalfDays → configFlat.
func effectiveEODFlatCT(reg kernel.SessionRegistry, sessionDayKey, configFlat string) string {
	if early, ok := reg.HalfDays[sessionDayKey]; ok && strings.TrimSpace(early) != "" {
		em, ok1 := hhmmToMin(early)
		cm, ok2 := hhmmToMin(configFlat)
		if ok1 && (!ok2 || em < cm) {
			return early // half-day closes earlier → pull the flat in
		}
	}
	return configFlat
}

func (at *AutoTrader) lastEntryCT() string {
	if dp := at.config.StrategyConfig.DayPlan; dp != nil && strings.TrimSpace(dp.LastEntryCT) != "" {
		return dp.LastEntryCT
	}
	return "13:00" // 14:00 ET
}

func (at *AutoTrader) eodFlatCT() string {
	if dp := at.config.StrategyConfig.DayPlan; dp != nil && strings.TrimSpace(dp.EODFlatCT) != "" {
		return dp.EODFlatCT
	}
	return "14:45" // 15:45 ET
}

// entryBlockedByLastEntry (P2.3) reports (reason, blocked): whether NEW entries
// are blocked because the last-entry CT time has passed. Gated on day_plan →
// dormant by default. (reason, ok) order matches the sibling entry gates.
func (at *AutoTrader) entryBlockedByLastEntry() (string, bool) {
	if !at.dayPlanEnabled() {
		return "", false
	}
	last := at.lastEntryCT()
	if timeReachedCT(time.Now(), last) {
		return fmt.Sprintf("past last-entry %s CT", last), true
	}
	return "", false
}

// enforceEODFlat (P2.3) force-flattens any open position at/after the effective
// EOD-flat time (config, pulled in on a half-day) by routing DIRECTLY through the
// trader close path — bypassing hold-lock naturally (RECON #10). Returns true
// when it acted (the caller then skips the rest of the cycle). Gated on day_plan.
func (at *AutoTrader) enforceEODFlat() bool {
	if !at.dayPlanEnabled() || at.store == nil || at.trader == nil {
		return false
	}
	now := time.Now()
	flat := effectiveEODFlatCT(kernel.DefaultSessionRegistry(), kernel.CMESessionDayKey(now), at.eodFlatCT())
	if !timeReachedCT(now, flat) {
		return false
	}
	positions, err := at.store.Position().GetOpenPositions(at.id)
	if err != nil || len(positions) == 0 {
		return false
	}
	at.logWarnf("🕒 EOD-FLAT (%s CT): session close — flattening %d open position(s) via the trader close path.", flat, len(positions))
	for _, p := range positions {
		var e error
		if strings.EqualFold(p.Side, "LONG") {
			_, e = at.trader.CloseLong(p.Symbol, 0) // 0 = close all
		} else {
			_, e = at.trader.CloseShort(p.Symbol, 0)
		}
		if e != nil {
			at.logErrorf("🕒 EOD-FLAT: close %s %s failed: %v", p.Symbol, p.Side, e)
			continue
		}
		// Cancel the resting OCO bracket after the flatten (belt-and-suspenders;
		// the OCO also auto-cancels its other leg on the close fill).
		if err := at.trader.CancelStopOrders(p.Symbol); err != nil {
			at.logWarnf("🕒 EOD-FLAT: cancel bracket %s failed (non-fatal): %v", p.Symbol, err)
		}
	}
	return true
}

// recordExcursionForClosedSymbol (P2.4) computes the MAE/MFE over the just-closed
// trade's hold from 1m bars and stores them on the position row. Gated on
// day_plan → dormant by default (crypto/plan-off write nothing).
func (at *AutoTrader) recordExcursionForClosedSymbol(symbol string) {
	if !at.dayPlanEnabled() || at.store == nil || market.FuturesBarsProvider == nil {
		return
	}
	closed, err := at.store.Position().GetClosedPositions(at.id, 1)
	if err != nil || len(closed) == 0 {
		return
	}
	p := closed[0]
	if p.Symbol != symbol || p.EntryPrice <= 0 || p.EntryTime <= 0 {
		return // the latest close is a different symbol / incomplete
	}
	exitMs := p.ExitTime
	if exitMs <= 0 {
		exitMs = time.Now().UnixMilli()
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), "1m", kernel.AISVPBarCount)
	ex := kernel.ComputeExcursion(p.EntryPrice, p.Side, bars, p.EntryTime, exitMs)
	if err := at.store.Position().UpdateExcursion(p.ID, ex.MAE, ex.MFE); err != nil {
		at.logWarnf("📐 excursion update failed for %s: %v", symbol, err)
		return
	}
	at.logInfof("📐 excursion %s: MAE %.2f / MFE %.2f pts (entry conf %d)", symbol, ex.MAE, ex.MFE, p.EntryConfidence)

	// P5.5 — ADHERENCE GRADE (A–F), separate from P&L: from the plan link stamped
	// at open + the entry-time window facts (killzone / no-trade).
	inKZ, inNoTrade := kernel.SessionWindowFacts(kernel.DefaultSessionRegistry(), time.UnixMilli(p.EntryTime))
	grade, _ := kernel.GradeAdherence(kernel.AdherenceInput{
		Cited:      p.CitedScenarioID != "",
		Matched:    p.PlanMatched,
		OffPlan:    p.CitedScenarioID == "",
		InNoTrade:  inNoTrade,
		InKillzone: inKZ,
	})
	if err := at.store.Position().SetAdherence(p.ID, grade); err != nil {
		at.logWarnf("🎓 adherence grade update failed for %s: %v", symbol, err)
		return
	}
	at.logInfof("🎓 adherence %s: %s (%s) — cited=%q matched=%v", symbol, grade, kernel.AdherenceLabel(grade), p.CitedScenarioID, p.PlanMatched)
}

// tickOnce runs one loop iteration: a grid cycle, or (for AI strategies) a
// decision cycle gated by bar-close cadence.
func (at *AutoTrader) tickOnce(isGrid bool) {
	if isGrid {
		if err := at.RunGridCycle(); err != nil {
			at.logErrorf("❌ Grid execution failed: %v", err)
		}
		return
	}
	active := at.barCloseCadenceActive()
	var latest int64
	var have bool
	if active {
		latest, have = at.latestClosedPrimaryBarMs()
	}
	run, newLast := barCloseGate(active, at.lastBarCloseMs, latest, have)
	at.lastBarCloseMs = newLast
	if !run {
		return // bar-close cadence: no new primary-TF bar closed → idle this tick
	}
	if err := at.runCycle(); err != nil {
		at.logErrorf("❌ Execution failed: %v", err)
	}
}
