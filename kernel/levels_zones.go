package kernel

import (
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
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
			// v3 (2026-08-24) — pattern classification: the leg BEFORE the base
			// vs the departure. Opposite signs → reversal (RBD/DBR, strongest);
			// same sign → continuation (RBR/DBD, weaker).
			pattern := ""
			if i > 0 {
				leg := cb[i-1].Close - cb[i-1].Open
				if (leg >= 0) != (move >= 0) {
					pattern = "reversal"
				} else {
					pattern = "continuation"
				}
			}
			if move >= departure {
				zl := zoneLevel(KindDemand, baseLo, baseHi, "Demand", origin)
				zl.ZonePattern = pattern
				zl.FormedAtMs = d.OpenTime // W6: birth = departure bar
				out = append(out, zl)
			} else if -move >= departure {
				zl := zoneLevel(KindSupply, baseLo, baseHi, "Supply", origin)
				zl.ZonePattern = pattern
				zl.FormedAtMs = d.OpenTime // W6: birth = departure bar
				out = append(out, zl)
			}
		}
		i = j + 1
	}
	return out
}

// FVGNoiseFloorPoints (grading audit §4.4, 2026-08-25) — the minimum FVG gap
// width in POINTS. Research: requiring a 1×ATR gap has no external support and
// kills the published 20–80 pt NQ sweet spot; the floor is instead
// max(2×tick, 2.0 pts) — any-gap detection with a noise floor, plus the
// size weighting zoneSizeMult already applies at scoring.
const FVGNoiseFloorPoints = 2.0

// fvgMinGapPoints resolves the FVG detection gap floor for a symbol:
// max(2 ticks, the noise floor). Ticks below the floor (e.g. MNQ 0.25 → 0.50)
// never raise it.
func fvgMinGapPoints(symbol string) float64 {
	tick := market.FuturesTickSize(symbol)
	if tick <= 0 {
		tick = 0.25
	}
	floor := FVGNoiseFloorPoints
	if 2*tick > floor {
		floor = 2 * tick
	}
	return floor
}

// FairValueGaps finds 3-candle imbalances: bullish when bar[i-2].High < bar[i].Low
// (gap between them), bearish when bar[i-2].Low > bar[i].High. UNFILLED gaps are
// emitted as KindFVG. W6 (2026-08-25): a gap whose far edge is later VIOLATED BY
// A CLOSE is no longer dropped — it is emitted as KindIFVG with INVERTED
// polarity (filled bullish FVG → bearish iFVG resistance; filled bearish FVG →
// bullish iFVG support), keeping the original bounds. Research-grounded: a
// stay-through close beyond the gap edge flips the imbalance's role.
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
		var bullish bool
		switch {
		case a.High < c.Low && c.Low-a.High >= minGap:
			gLo, gHi = a.High, c.Low
			bullish = true
		case a.Low > c.High && a.Low-c.High >= minGap:
			gLo, gHi = c.High, a.Low
			bullish = false
		default:
			continue
		}
		origin := time.UnixMilli(c.OpenTime).In(loc).Format("2006-01-02")
		// W6 inversion: a later CLOSE beyond the far edge (below a bullish
		// gap's low, above a bearish gap's high) flips the imbalance.
		inverted := false
		for j := i + 1; j < len(cb); j++ {
			if bullish && cb[j].Close < gLo {
				inverted = true
				break
			}
			if !bullish && cb[j].Close > gHi {
				inverted = true
				break
			}
		}
		if inverted {
			lvl := zoneLevel(KindIFVG, gLo, gHi, "iFVG", origin)
			if bullish {
				lvl.Label = "iFVG(bear)" // flipped to resistance
			} else {
				lvl.Label = "iFVG(bull)" // flipped to support
			}
			lvl.Info = "filled→inverted"
			lvl.FormedAtMs = c.OpenTime
			out = append(out, lvl)
			continue
		}
		lvl := zoneLevel(KindFVG, gLo, gHi, "FVG", origin)
		lvl.FormedAtMs = c.OpenTime
		out = append(out, lvl)
	}
	return out
}

// OBLookbackBarsDefault (grading audit §4.5, 2026-08-25) — the order-block
// pairing scan bound. An unbounded scan can pair a displacement with an
// opposing candle HOURS back (a stale, meaningless OB); research supports a
// tight lookback. Env OB_LOOKBACK_BARS overrides; the value is a USER knob,
// never a hardcoded rule.
const OBLookbackBarsDefault = 8

// obLookbackBars resolves the OB pairing lookback (env OB_LOOKBACK_BARS,
// default 8 bars).
func obLookbackBars() int {
	if v := os.Getenv("OB_LOOKBACK_BARS"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 {
			return n
		}
	}
	return OBLookbackBarsDefault
}

// OBLookbackBars is the exported read for boot/observability lines.
func OBLookbackBars() int { return obLookbackBars() }

// OrderBlocks finds the last opposing candle within a bounded lookback before a
// displacement ≥1.5×ATR: a big up move → the last DOWN candle before it (bullish
// OB); a big down move → the last UP candle before it (bearish OB). Zone = that
// candle's [low,high].
func OrderBlocks(bars []market.Kline, atr float64, now time.Time) []DetectedLevel {
	cb := closedBars(bars, now)
	if atr <= 0 || len(cb) < 2 {
		return nil
	}
	loc := chicago()
	displacement := 1.5 * atr
	lookback := obLookbackBars() // bounded scan (§4.5) — never pairs across hours
	var out []DetectedLevel
	for i := 1; i < len(cb); i++ {
		move := cb[i].Close - cb[i].Open
		origin := time.UnixMilli(cb[i].OpenTime).In(loc).Format("2006-01-02")
		if move >= displacement {
			// bullish displacement → last down candle before it (bounded)
			for j := i - 1; j >= 0 && i-j <= lookback; j-- {
				if cb[j].Close < cb[j].Open {
					zl := zoneLevel(KindOB, cb[j].Low, cb[j].High, "OB(bull)", origin)
					zl.FormedAtMs = cb[i].OpenTime // W6: birth = displacement bar
					out = append(out, zl)
					break
				}
			}
		} else if -move >= displacement {
			for j := i - 1; j >= 0 && i-j <= lookback; j-- {
				if cb[j].Close > cb[j].Open {
					zl := zoneLevel(KindOB, cb[j].Low, cb[j].High, "OB(bear)", origin)
					zl.FormedAtMs = cb[i].OpenTime // W6: birth = displacement bar
					out = append(out, zl)
					break
				}
			}
		}
	}
	return out
}
