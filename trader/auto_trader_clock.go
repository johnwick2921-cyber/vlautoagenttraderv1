package trader

import (
	"time"

	"nofx/market"
)

// P2 — THE CLOCK. Bar-close cadence + the skip-while-open gate. Both are GATED on
// day_plan being enabled for a futures trader, so the default (crypto / plan-off)
// keeps the scan-timer loop and same-side-refusal behavior byte-identical —
// additive + dormant until the owner arms day_plan at ★ RESTART 1.

// barCloseCadenceActive reports whether this trader should fire cycles on
// primary-TF bar closes instead of the scan timer.
func (at *AutoTrader) barCloseCadenceActive() bool {
	if at.exchange != "ninjatrader" || at.config.StrategyConfig == nil {
		return false
	}
	dp := at.config.StrategyConfig.DayPlan
	return dp != nil && dp.PlanEnabled
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
