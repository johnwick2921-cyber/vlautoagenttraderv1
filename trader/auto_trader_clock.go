package trader

import (
	"fmt"
	"time"

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
