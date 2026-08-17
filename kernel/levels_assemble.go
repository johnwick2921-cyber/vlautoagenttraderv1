package kernel

import (
	"math"
	"time"

	"nofx/market"
)

// P1.7 — assemble every detector → confluence scorer → the KEY LEVELS prompt
// block. One entry point the decision loop calls (mirror of FormatSVPLine +
// BuildSVPProfile). Pure: no LLM, no DB. Warms forward with the bar cache; the
// durable session-profile store (P1.3) feeds nPOC via extraLevels.

// PlanProximityKProvider supplies the RESOLVED day-trade lock width
// (proximity_filter_atr in daily-ATR multiples, valid 0.5–3.0) to kernel call
// sites that have no trader handle — the executor prompt path in
// engine_analysis. The trader installs it (mirrors FuturesBarsProvider /
// NakedPOCProvider / ActivePlanProvider). nil → the spec constant 1.5.
var PlanProximityKProvider func() float64

// resolvedProximityK returns the configured day-trade lock k, falling back to
// the spec constant when no provider is installed or it returns an invalid value.
func resolvedProximityK() float64 {
	if PlanProximityKProvider != nil {
		if k := PlanProximityKProvider(); k > 0 {
			return k
		}
	}
	return ActivationWindowK
}

// BuildKeyLevelsBlock assembles the structural map from `bars`, scores it, and
// renders the executor prompt block. Returns "" when there is nothing to show
// (no closed bars / no in-band levels) so the caller injects nothing.
// proximityK is the resolved day-trade lock width (≤0 → spec constant 1.5).
func BuildKeyLevelsBlock(bars []market.Kline, reg SessionRegistry, symbol string, maxLevels int, now time.Time, proximityK float64, extraLevels ...DetectedLevel) string {
	scored, price, _ := AssembleScoredLevels(bars, reg, symbol, maxLevels, now, proximityK, extraLevels...)
	if price <= 0 {
		return ""
	}
	return RenderKeyLevelsBlock(scored, price)
}

// AssembleScoredLevels runs every detector, scores them, and returns the graded
// TOP-N levels + the reference price + the daily-ATR used. Shared by the executor
// KEY LEVELS block (P1.7) and the planner input package (P3.3). Returns
// (nil, 0, 0) when there are no closed bars. proximityK is the resolved day-trade
// lock width (≤0 → spec constant 1.5) and threads into BOTH the round-number
// generator and the scorer — H1/H2: the config must govern which levels are
// generated AND which are seated, not just the activation-window paths.
func AssembleScoredLevels(bars []market.Kline, reg SessionRegistry, symbol string, maxLevels int, now time.Time, proximityK float64, extraLevels ...DetectedLevel) (scored []ScoredLevel, price, dATR float64) {
	cb := closedBars(bars, now)
	if len(cb) == 0 {
		return nil, 0, 0
	}
	price = cb[len(cb)-1].Close
	if price <= 0 {
		return nil, 0, 0
	}
	dATR = DailyATRProxy(bars, now)
	if dATR <= 0 {
		dATR = 0.008 * price // fallback until the map warms
	}
	atr := market.ExportCalculateATR(cb, 14)
	if atr <= 0 {
		atr = dATR / 20
	}
	tick := market.FuturesTickSize(symbol)
	if tick <= 0 {
		tick = 0.25
	}
	tol := 3 * tick

	var all []DetectedLevel
	all = append(all, ExtractMultiDayLevels(bars, reg, now)...)
	all = append(all, RoundNumberLevels(price, dATR, proximityK)...)
	all = append(all, OpeningRangeLevels(bars, reg, now)...)
	all = append(all, GapLevels(bars, atr, 1.0, now)...)
	all = append(all, EqualHighsLows(bars, tol, now)...)
	all = append(all, SupplyDemandZones(bars, atr, now)...)
	all = append(all, FairValueGaps(bars, atr, now)...)
	all = append(all, OrderBlocks(bars, atr, now)...)
	all = append(all, extraLevels...) // nPOC etc. from the durable store (P1.3)

	// W11b — persisted level-state (freshness A→B→C, consumed) now surfaces: the
	// trader installs LevelStateProvider over store.LevelStateStore. Nil provider →
	// all-fresh (byte-identical to the pre-W11b output the goldens capture).
	scored = ScoreLevels(all, price, dATR, levelFreshnessFn(symbol), maxLevels, proximityK)
	return scored, price, dATR
}

// DailyATRProxy estimates the daily ATR from intraday bars by averaging each
// COMPLETED CME session-day's range (the developing day is skipped). 0 when no
// completed day is present — the caller falls back. Improves as the cache /
// durable store warms forward.
func DailyATRProxy(bars []market.Kline, now time.Time) float64 {
	cb := closedBars(bars, now)
	if len(cb) == 0 {
		return 0
	}
	type dayRange struct{ hi, lo float64 }
	days := map[string]*dayRange{}
	for _, b := range cb {
		key := CMESessionDayKey(time.UnixMilli(b.OpenTime))
		d := days[key]
		if d == nil {
			d = &dayRange{hi: math.Inf(-1), lo: math.Inf(1)}
			days[key] = d
		}
		d.hi = math.Max(d.hi, b.High)
		d.lo = math.Min(d.lo, b.Low)
	}
	nowFut := CMESessionDayKey(now)
	var sum float64
	var n int
	for key, d := range days {
		if key == nowFut {
			continue // developing day is not a completed range
		}
		if d.hi > d.lo {
			sum += d.hi - d.lo
			n++
		}
	}
	if n > 0 {
		return sum / float64(n)
	}
	// Fallback: the developing day's range so far.
	if d := days[nowFut]; d != nil && d.hi > d.lo {
		return d.hi - d.lo
	}
	return 0
}
