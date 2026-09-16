package trader

import (
	"nofx/kernel"
	"nofx/market"
)

// W11 — render the planner's INDICATOR MIRROR: the SAME per-timeframe indicator
// state the executor prompt shows, computed from the SAME market path
// (market.GetWithTimeframes → FuturesBarsProvider for CME, offline) with the SAME
// ai_config toggles + periods. Returns (renderedBlock, aiConfigFingerprint). The
// block is "" when no indicator is enabled or bars are unavailable (→ the planner
// omits it, disabled state = byte-identical prompt). Zero new config: everything is
// read from the strategy's existing ai_config.
func (at *AutoTrader) renderIndicatorMirror(symbol string) (string, string) {
	if at.config.StrategyConfig == nil {
		return "", ""
	}
	ic := at.config.StrategyConfig.Indicators
	hash := ic.AIConfigFingerprint()

	// Resolve the executor's timeframe set exactly like fetchMarketDataWithStrategy.
	tfs := append([]string{}, ic.Klines.SelectedTimeframes...)
	primary := ic.Klines.PrimaryTimeframe
	if len(tfs) == 0 {
		if primary != "" {
			tfs = append(tfs, primary)
		}
		if ic.Klines.LongerTimeframe != "" {
			tfs = append(tfs, ic.Klines.LongerTimeframe)
		}
	}
	if len(tfs) == 0 {
		return "", hash // no timeframes configured → no mirror, hash still recorded
	}
	if primary == "" {
		primary = tfs[0]
	}
	count := ic.Klines.PrimaryCount
	if count <= 0 {
		count = 30
	}
	indPeriods := market.IndicatorPeriods{
		EMA:  ic.EMAPeriods,
		RSI:  ic.RSIPeriods,
		ATR:  ic.ATRPeriods,
		BOLL: ic.BOLLPeriods,
	}
	mkt, err := market.GetWithTimeframes(symbol, tfs, primary, count, indPeriods)
	if err != nil || mkt == nil {
		return "", hash // bars unavailable (e.g. NT8 down at read time) → no mirror
	}
	return kernel.RenderPlannerIndicatorBlock(mkt, ic, tfs), hash
}
