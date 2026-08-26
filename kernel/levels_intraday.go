package kernel

import (
	"fmt"
	"math"
	"sort"
	"time"

	"nofx/market"
)

// P1.2 — intraday computed levels: round numbers, unfilled gaps, opening range +
// initial balance. All pure + deterministic; the caller supplies the daily-ATR
// (dATR) scale where proximity/size thresholds apply.

// RoundNumberLevels emits round-number levels at 100/50/25 spacing within
// proximityK×dATR of price. proximityK is the resolved day-trade lock width
// (proximity_filter_atr, default 1.5); k≤0 → the spec constant. A price that is
// a multiple of a larger step is claimed by that step (100 > 50 > 25), so each
// round price appears once, tagged with its strongest granularity.
func RoundNumberLevels(price, dATR, proximityK float64) []DetectedLevel {
	if price <= 0 || dATR <= 0 {
		return nil
	}
	if proximityK <= 0 {
		proximityK = ActivationWindowK
	}
	band := proximityK * dATR
	lo, hi := price-band, price+band
	steps := []struct {
		step float64
		tag  string
	}{{100, "100"}, {50, "50"}, {25, "25"}}

	seen := map[int64]bool{}
	var out []DetectedLevel
	for _, s := range steps {
		n0 := int64(math.Ceil(lo / s.step))
		for n := n0; float64(n)*s.step <= hi; n++ {
			m := float64(n) * s.step
			key := int64(math.Round(m * 100))
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, lineLevel(KindRound, m, fmt.Sprintf("RN %.0f (%s)", m, s.tag), "", false))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Price < out[j].Price })
	return out
}

// GapLevels finds UNFILLED price gaps between consecutive closed bars whose size
// ≥ minGapATR×atr, and emits each gap's fill target (the near/prior-session edge
// price toward which price would fill). A gap is filled once a later bar trades
// back through its near edge.
func GapLevels(bars []market.Kline, atr, minGapATR float64, now time.Time) []DetectedLevel {
	if atr <= 0 || minGapATR <= 0 {
		return nil
	}
	nowMs := now.UnixMilli()
	loc := chicago()

	closed := make([]market.Kline, 0, len(bars))
	for i := range bars {
		if bars[i].CloseTime < nowMs {
			closed = append(closed, bars[i])
		}
	}
	if len(closed) < 2 {
		return nil
	}
	minGap := minGapATR * atr

	var out []DetectedLevel
	for i := 1; i < len(closed); i++ {
		p, c := closed[i-1], closed[i]
		var gLo, gHi float64
		var up bool
		switch {
		case c.Low > p.High && c.Low-p.High >= minGap:
			gLo, gHi, up = p.High, c.Low, true
		case c.High < p.Low && p.Low-c.High >= minGap:
			gLo, gHi, up = c.High, p.Low, false
		default:
			continue
		}
		// Fill check: gap-up fills when a later bar's low reaches the near edge
		// (gLo); gap-down fills when a later bar's high reaches gHi.
		filled := false
		for j := i + 1; j < len(closed); j++ {
			if up && closed[j].Low <= gLo {
				filled = true
				break
			}
			if !up && closed[j].High >= gHi {
				filled = true
				break
			}
		}
		if filled {
			continue
		}
		target := gLo
		dir := "↑"
		if !up {
			target = gHi
			dir = "↓"
		}
		origin := time.UnixMilli(c.OpenTime).In(loc).Format("2006-01-02")
		out = append(out, DetectedLevel{
			Kind:       KindGap,
			Price:      target,
			Lo:         gLo,
			Hi:         gHi,
			Label:      fmt.Sprintf("GAP%s fill %.2f", dir, target),
			OriginDate: origin,
			Info:       fmt.Sprintf("size %.2f, unfilled", gHi-gLo),
		})
	}
	return out
}

// OpeningRangeLevels emits today's opening-range (first 5m) high/low and initial
// balance (first 60m, i.e. 9:30–10:30 ET) high/low plus the 1.5× and 2× IB
// extensions. Returns nil before today's RTH open (nothing to measure yet).
func OpeningRangeLevels(bars []market.Kline, reg SessionRegistry, now time.Time) []DetectedLevel {
	loc := chicago()
	nowCT := now.In(loc)
	nowMs := now.UnixMilli()

	// RTH open = 08:30 CT of now's calendar day (from the NY registry row).
	openMin := 8*60 + 30
	if ny, ok := reg.SessionByName(SessionNY); ok {
		if m, ok := parseHHMM(ny.WindowStartCT); ok {
			openMin = m
		}
	}
	rthOpen := time.Date(nowCT.Year(), nowCT.Month(), nowCT.Day(), openMin/60, openMin%60, 0, 0, loc)
	if nowCT.Before(rthOpen) {
		return nil // pre-open: no OR/IB yet
	}
	orEnd := rthOpen.Add(5 * time.Minute)
	ibEnd := rthOpen.Add(60 * time.Minute)

	var orH, orL, ibH, ibL float64 = math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(1)
	var hasOR, hasIB bool
	for i := range bars {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue
		}
		bt := time.UnixMilli(b.OpenTime).In(loc)
		if !bt.Before(rthOpen) && bt.Before(orEnd) {
			hasOR = true
			orH, orL = math.Max(orH, b.High), math.Min(orL, b.Low)
		}
		if !bt.Before(rthOpen) && bt.Before(ibEnd) {
			hasIB = true
			ibH, ibL = math.Max(ibH, b.High), math.Min(ibL, b.Low)
		}
	}

	origin := rthOpen.Format("2006-01-02")
	var out []DetectedLevel
	if hasOR {
		out = append(out,
			lineLevel(KindORH, orH, "OR-H", origin, false),
			lineLevel(KindORL, orL, "OR-L", origin, false),
		)
	}
	if hasIB {
		r := ibH - ibL
		out = append(out,
			lineLevel(KindIBH, ibH, "IB-H", origin, false),
			lineLevel(KindIBL, ibL, "IB-L", origin, false),
			lineLevel(KindIBH, ibH+0.5*r, "IB+1.5x", origin, false),
			lineLevel(KindIBH, ibH+1.0*r, "IB+2x", origin, false),
			lineLevel(KindIBL, ibL-0.5*r, "IB-1.5x", origin, false),
			lineLevel(KindIBL, ibL-1.0*r, "IB-2x", origin, false),
		)
	}
	return out
}
