package kernel

import (
	"math"
	"sort"
	"time"

	"nofx/market"
)

// P1.4 — liquidity + zone detectors: EQH/EQL, S/D zones, FVG, OB.
//
// Pure + deterministic. Per the spec these ALL enter the scorer as
// C/confluence-only (never standalone triggers) — the scorer enforces that via
// isZoneKind + the EQH/EQL kinds being low type-evidence. Here we only DETECT.

// closedBars returns the closed subset (CloseTime < nowMs), preserving order.
func closedBars(bars []market.Kline, now time.Time) []market.Kline {
	nowMs := now.UnixMilli()
	out := make([]market.Kline, 0, len(bars))
	for i := range bars {
		if bars[i].CloseTime < nowMs {
			out = append(out, bars[i])
		}
	}
	return out
}

// EqualHighsLows finds swing highs/lows that repeat within `tol` price units —
// resting liquidity (EQH above, EQL below). k=2 pivot lookback.
func EqualHighsLows(bars []market.Kline, tol float64, now time.Time) []DetectedLevel {
	cb := closedBars(bars, now)
	if tol <= 0 || len(cb) < 5 {
		return nil
	}
	const k = 2
	var hi, lo []float64
	for i := k; i < len(cb)-k; i++ {
		if isStrictPivotHigh(cb, i, k) {
			hi = append(hi, cb[i].High)
		}
		if isStrictPivotLow(cb, i, k) {
			lo = append(lo, cb[i].Low)
		}
	}
	origin := time.UnixMilli(cb[len(cb)-1].OpenTime).In(chicago()).Format("2006-01-02")
	var out []DetectedLevel
	out = append(out, clusterEqual(hi, tol, KindEQH, "EQH", true, origin)...)
	out = append(out, clusterEqual(lo, tol, KindEQL, "EQL", false, origin)...)
	return out
}

func isStrictPivotHigh(b []market.Kline, i, k int) bool {
	for j := i - k; j <= i+k; j++ {
		if j != i && b[j].High >= b[i].High {
			return false
		}
	}
	return true
}

func isStrictPivotLow(b []market.Kline, i, k int) bool {
	for j := i - k; j <= i+k; j++ {
		if j != i && b[j].Low <= b[i].Low {
			return false
		}
	}
	return true
}

// clusterEqual groups pivot prices within tol; a group of ≥2 is an equal-highs/
// lows liquidity level (at the group max for highs, min for lows).
func clusterEqual(prices []float64, tol float64, kind LevelKind, label string, high bool, origin string) []DetectedLevel {
	if len(prices) < 2 {
		return nil
	}
	sort.Float64s(prices)
	var out []DetectedLevel
	i := 0
	for i < len(prices) {
		j := i + 1
		for j < len(prices) && prices[j]-prices[i] <= tol {
			j++
		}
		if j-i >= 2 {
			p := prices[i] // group min (lows)
			if high {
				p = prices[j-1] // group max (highs)
			}
			out = append(out, lineLevel(kind, p, label, origin, false))
		}
		i = j
	}
	return out
}

// SupplyDemandZones finds a small-bodied base (≤6 candles, bodies ≤0.5×ATR)
// followed by a departure ≥1.5×ATR: up departure → demand (base below), down →
// supply (base above). Zone = the base's [low,high].
func SupplyDemandZones(bars []market.Kline, atr float64, now time.Time) []DetectedLevel {
	cb := closedBars(bars, now)
	if atr <= 0 || len(cb) < 3 {
		return nil
	}
	loc := chicago()
	smallBody := 0.5 * atr
	departure := 1.5 * atr

	var out []DetectedLevel
	i := 0
	for i < len(cb) {
		// grow a base of small-bodied candles (1..6)
		if math.Abs(cb[i].Close-cb[i].Open) > smallBody {
			i++
			continue
		}
		baseLo, baseHi := cb[i].Low, cb[i].High
		j := i
		for j+1 < len(cb) && j-i < 5 && math.Abs(cb[j+1].Close-cb[j+1].Open) <= smallBody {
			j++
			baseLo = math.Min(baseLo, cb[j].Low)
			baseHi = math.Max(baseHi, cb[j].High)
		}
		// departure candle right after the base?
		if j+1 < len(cb) {
			d := cb[j+1]
			move := d.Close - d.Open
			origin := time.UnixMilli(d.OpenTime).In(loc).Format("2006-01-02")
			if move >= departure {
				out = append(out, zoneLevel(KindDemand, baseLo, baseHi, "Demand", origin))
			} else if -move >= departure {
				out = append(out, zoneLevel(KindSupply, baseLo, baseHi, "Supply", origin))
			}
		}
		i = j + 1
	}
	return out
}

// FairValueGaps finds 3-candle imbalances: bullish when bar[i-2].High < bar[i].Low
// (gap between them), bearish when bar[i-2].Low > bar[i].High. Only UNFILLED gaps
// (no later bar has traded back through the gap) are emitted.
func FairValueGaps(bars []market.Kline, minGap float64, now time.Time) []DetectedLevel {
	cb := closedBars(bars, now)
	if len(cb) < 3 {
		return nil
	}
	loc := chicago()
	var out []DetectedLevel
	for i := 2; i < len(cb); i++ {
		a, c := cb[i-2], cb[i]
		var gLo, gHi float64
		switch {
		case a.High < c.Low && c.Low-a.High >= minGap:
			gLo, gHi = a.High, c.Low
		case a.Low > c.High && a.Low-c.High >= minGap:
			gLo, gHi = c.High, a.Low
		default:
			continue
		}
		filled := false
		for j := i + 1; j < len(cb); j++ {
			if cb[j].Low <= gLo && cb[j].High >= gHi {
				filled = true
				break
			}
		}
		if filled {
			continue
		}
		origin := time.UnixMilli(c.OpenTime).In(loc).Format("2006-01-02")
		out = append(out, zoneLevel(KindFVG, gLo, gHi, "FVG", origin))
	}
	return out
}

// OrderBlocks finds the last opposing candle before a displacement ≥1.5×ATR:
// a big up move → the last DOWN candle before it (bullish OB); a big down move →
// the last UP candle before it (bearish OB). Zone = that candle's [low,high].
func OrderBlocks(bars []market.Kline, atr float64, now time.Time) []DetectedLevel {
	cb := closedBars(bars, now)
	if atr <= 0 || len(cb) < 2 {
		return nil
	}
	loc := chicago()
	displacement := 1.5 * atr
	var out []DetectedLevel
	for i := 1; i < len(cb); i++ {
		move := cb[i].Close - cb[i].Open
		origin := time.UnixMilli(cb[i].OpenTime).In(loc).Format("2006-01-02")
		if move >= displacement {
			// bullish displacement → last down candle before it
			for j := i - 1; j >= 0; j-- {
				if cb[j].Close < cb[j].Open {
					out = append(out, zoneLevel(KindOB, cb[j].Low, cb[j].High, "OB(bull)", origin))
					break
				}
			}
		} else if -move >= displacement {
			for j := i - 1; j >= 0; j-- {
				if cb[j].Close > cb[j].Open {
					out = append(out, zoneLevel(KindOB, cb[j].Low, cb[j].High, "OB(bear)", origin))
					break
				}
			}
		}
	}
	return out
}
