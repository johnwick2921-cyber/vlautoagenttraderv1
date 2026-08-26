package kernel

import (
	"math"
	"strings"
	"time"

	"nofx/market"
)

// P1.7 — assemble every detector → confluence scorer → the KEY LEVELS prompt
// block. One entry point the decision loop calls (mirror of FormatSVPLine +
// BuildSVPProfile). Pure: no LLM, no DB. Warms forward with the bar cache; the
// durable session-profile store (P1.3) feeds nPOC via extraLevels.

// BuildKeyLevelsBlock assembles the structural map from `bars`, scores it, and
// renders the executor prompt block. Returns "" when there is nothing to show
// (no closed bars / no in-band levels) so the caller injects nothing.
// proximityK is the resolved day-trade lock width (≤0 → spec constant 1.5).
func BuildKeyLevelsBlock(traderID string, bars []market.Kline, reg SessionRegistry, symbol string, maxLevels int, now time.Time, proximityK float64, extraLevels ...DetectedLevel) string {
	return BuildKeyLevelsBlockOpts(traderID, bars, reg, symbol, maxLevels, now, proximityK, false, "", extraLevels...)
}

// BuildKeyLevelsBlockOpts (grading audit §4.6/4.7 + 1h wave, 2026-08-25) —
// BuildKeyLevelsBlock with the seat_1h_zone guarantee and the minGrade floor
// applied. The executor KEY LEVELS block must obey the SAME rules as the
// planner table (one-sided tables and sub-min rows reached the executor before
// this).
func BuildKeyLevelsBlockOpts(traderID string, bars []market.Kline, reg SessionRegistry, symbol string, maxLevels int, now time.Time, proximityK float64, seat1HZone bool, minGrade string, extraLevels ...DetectedLevel) string {
	scored, price, _ := AssembleScoredLevelsMinGrade(traderID, bars, reg, symbol, maxLevels, now, proximityK, minGrade, extraLevels...)
	if price <= 0 {
		return ""
	}
	if seat1HZone {
		scored = Seat1HZone(scored, maxLevels)
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
func AssembleScoredLevels(traderID string, bars []market.Kline, reg SessionRegistry, symbol string, maxLevels int, now time.Time, proximityK float64, extraLevels ...DetectedLevel) (scored []ScoredLevel, price, dATR float64) {
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
	all = append(all, FairValueGaps(bars, fvgMinGapPoints(symbol), now)...)
	all = append(all, OrderBlocks(bars, atr, now)...)
	all = append(all, extraLevels...) // nPOC etc. from the durable store (P1.3)

	// W11b — persisted level-state (freshness A→B→C, consumed) now surfaces: the
	// trader installs LevelStateProvider over store.LevelStateStore. Nil provider →
	// all-fresh (byte-identical to the pre-W11b output the goldens capture).
	scored = ScoreLevels(all, price, dATR, levelFreshnessFn(traderID, symbol), maxLevels, proximityK)
	return scored, price, dATR
}

// AssembleScoredLevelsMinGrade (grading audit §4.6/4.7, 2026-08-25) is
// AssembleScoredLevels with a minGrade floor: the scorer runs on a 2× pool so
// a sub-floor row filtered OUT can be replaced by an in-band same-side
// candidate — seatBothSides re-balances AFTER the filter, so the minGrade cut
// can never leave the executor/planner table one-sided when candidates exist.
// Empty minGrade → byte-identical to AssembleScoredLevels.
func AssembleScoredLevelsMinGrade(traderID string, bars []market.Kline, reg SessionRegistry, symbol string, maxLevels int, now time.Time, proximityK float64, minGrade string, extraLevels ...DetectedLevel) (scored []ScoredLevel, price, dATR float64) {
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
	all = append(all, FairValueGaps(bars, fvgMinGapPoints(symbol), now)...)
	all = append(all, OrderBlocks(bars, atr, now)...)
	all = append(all, extraLevels...) // nPOC etc. from the durable store (P1.3)

	scored = ScoreLevelsMinGrade(all, price, dATR, levelFreshnessFn(traderID, symbol), maxLevels, proximityK, minGrade)
	return scored, price, dATR
}

// DetectHTFLevels (G2/G3, 2026-08-24) — per-TF swing/zone detection on the
// CONFIGURED higher timeframes, so real 1h/4h support/resistance enters the
// candidate pool (before this, every detector ran on the 1m slice only). Every
// output carries HTF=true: the scorer's ×1.2 multiplier and the P0.1
// standalone-zone rule then apply. TFs below 15m and "D" are skipped —
// intraday noise adds nothing, and the daily anchors are already covered by
// ExtractMultiDayLevels. Cluster tolerance scales with the TF's own ATR
// (3 ticks is meaningless on a 1h bar).
func DetectHTFLevels(fetch func(tf string, count int) []market.Kline, timeframes []string, symbol string, now time.Time) []DetectedLevel {
	if fetch == nil || len(timeframes) == 0 {
		return nil
	}
	tick := market.FuturesTickSize(symbol)
	if tick <= 0 {
		tick = 0.25
	}
	seen := map[string]bool{}
	var out []DetectedLevel
	for _, tf := range timeframes {
		tf = strings.ToLower(strings.TrimSpace(tf))
		if tf == "" || seen[tf] || !isHTFDetectionTF(tf) {
			continue
		}
		seen[tf] = true
		cb := closedBars(fetch(tf, 500), now)
		if len(cb) < 5 {
			continue
		}
		atr := market.ExportCalculateATR(cb, 14)
		if atr <= 0 {
			atr = 0.002 * cb[len(cb)-1].Close
		}
		tol := 3 * tick
		if alt := 0.15 * atr; alt > tol {
			tol = alt
		}
		for _, l := range EqualHighsLows(cb, tol, now) {
			out = append(out, tagHTFLevel(l, tf))
		}
		for _, l := range SupplyDemandZones(cb, atr, now) {
			out = append(out, tagHTFLevel(l, tf))
		}
		for _, l := range FairValueGaps(cb, atr, now) {
			out = append(out, tagHTFLevel(l, tf))
		}
		for _, l := range OrderBlocks(cb, atr, now) {
			out = append(out, tagHTFLevel(l, tf))
		}
	}
	return out
}

// isHTFDetectionTF lists the timeframes DetectHTFLevels runs on (≥15m, not "D").
func isHTFDetectionTF(tf string) bool {
	switch tf {
	case "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h":
		return true
	}
	return false
}

// tagHTFLevel marks a detected level with its HTF origin + a TF-suffixed label
// ("EQH·1h", "Demand·4h") so the ranked table and the card show provenance.
// Also sets the structured TF field the v3 zone tiers grade on.
func tagHTFLevel(l DetectedLevel, tf string) DetectedLevel {
	l.HTF = true
	l.TF = tf
	l.Label = l.Label + "·" + tf
	return l
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
