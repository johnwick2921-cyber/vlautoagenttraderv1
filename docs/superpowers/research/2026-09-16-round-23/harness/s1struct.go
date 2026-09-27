package main

// s1struct.go — EXACT PORT of Claude-101's S1 structure-state definition for
// Q-B. Provenance: kernel/structure.go + kernel/structure_map.go on
// origin/feat/structure-map @ f36ddca2 (fetched 2026-09-16 23:41Z). The CTO's
// S4 dispatch: "define the state EXACTLY as S1 does (HH/HL vs LH/LL from the
// swing detector ... until then implement the same definition and mark [B])".
// This is that implementation: same constants (k=2, min-move=0.25×ATR14), same
// Wilder-smoothed simpleATR14, same fractal window and same-side-extreme
// replacement, same HH/HL/LH/LL labelling, same trend rule (last 3 labelled
// swings: all HH/HL → up, all LH/LL → down, otherwise range; <3 swings →
// range). Parity guard: s1ParityZigzagTest in s1struct_test.go reproduces S1's
// own TestStructureMapReadsTrendImpulseAndPremiumDiscount pin (trend=up,
// swings ≈118/108 on the identical zigzag fixture) through this port.

import "nofx/market"

// s1 constants — verbatim defaults from S1's structure.go (STRUCTURE_SWING_K,
// STRUCTURE_MIN_SWING_ATR, DefaultStructureTrendSwings).
const (
	s1SwingK      = 2
	s1MinSwingATR = 0.25
	s1TrendSwings = 3
)

// s1Swing is the port of S1's `swing` (kind/price/timeMs/high).
type s1Swing struct {
	kind   string
	price  float64
	timeMs int64
	high   bool
}

// s1SimpleATR14 is the verbatim port of S1's simpleATR14 (Wilder ATR(14),
// trs[0]=H-L, smoothed from index n onward; short-series mean).
func s1SimpleATR14(highs, lows, closes []float64) float64 {
	if len(closes) == 0 {
		return 0
	}
	if len(closes) == 1 {
		return highs[0] - lows[0]
	}
	trs := make([]float64, len(closes))
	trs[0] = highs[0] - lows[0]
	for i := 1; i < len(closes); i++ {
		tr := highs[i] - lows[i]
		if hc := highs[i] - closes[i-1]; hc > tr {
			tr = hc
		}
		if lc := closes[i-1] - lows[i]; lc > tr {
			tr = lc
		}
		trs[i] = tr
	}
	n := 14
	if len(trs) <= n {
		sum := 0.0
		for i := 1; i < len(trs); i++ {
			sum += trs[i]
		}
		return sum / float64(len(trs)-1)
	}
	sum := 0.0
	for i := 1; i <= n; i++ {
		sum += trs[i]
	}
	atr := sum / float64(n)
	for i := n + 1; i < len(trs); i++ {
		atr = (atr*float64(n-1) + trs[i]) / float64(n)
	}
	return atr
}

// s1FractalSwings is the verbatim port of S1's fractalSwings(closed, iv, atr):
// k-window extremes, alternating high/low with same-side replacement by the
// more extreme value, min-move filter vs the prior opposite swing, then
// HH/HL/LH/LL labelling against the previous same-type swing.
func s1FractalSwings(closed []market.KlineBar, iv int64, atr float64) []s1Swing {
	n := len(closed)
	if n == 0 {
		return nil
	}
	highs := make([]float64, n)
	lows := make([]float64, n)
	for i, b := range closed {
		highs[i] = b.High
		lows[i] = b.Low
	}
	k := s1SwingK
	swings := make([]s1Swing, 0, 16)
	for i := k; i < n-k; i++ {
		isHigh, isLow := true, true
		for j := i - k; j <= i+k; j++ {
			if j == i {
				continue
			}
			if highs[j] >= highs[i] {
				isHigh = false
			}
			if lows[j] <= lows[i] {
				isLow = false
			}
		}
		if !isHigh && !isLow {
			continue
		}
		price, hi := highs[i], true
		if isLow {
			price, hi = lows[i], false
		}
		t := closed[i].Time + iv
		if len(swings) > 0 && swings[len(swings)-1].high == hi {
			// Same-side extreme: keep the more extreme one.
			if (hi && price <= swings[len(swings)-1].price) || (!hi && price >= swings[len(swings)-1].price) {
				continue
			}
			swings[len(swings)-1] = s1Swing{price: price, timeMs: t, high: hi}
			continue
		}
		// Min-move significance vs the prior opposite swing.
		if len(swings) > 0 {
			move := price - swings[len(swings)-1].price
			if move < 0 {
				move = -move
			}
			if atr > 0 && move < s1MinSwingATR*atr {
				continue
			}
		}
		swings = append(swings, s1Swing{price: price, timeMs: t, high: hi})
	}
	// 2. label HH/HL/LH/LL against the previous same-type swing.
	for i := range swings {
		prev := -1
		for j := i - 1; j >= 0; j-- {
			if swings[j].high == swings[i].high {
				prev = j
				break
			}
		}
		switch {
		case swings[i].high && prev < 0:
			swings[i].kind = "HH"
		case swings[i].high && swings[i].price > swings[prev].price:
			swings[i].kind = "HH"
		case swings[i].high:
			swings[i].kind = "LH"
		case !swings[i].high && prev < 0:
			swings[i].kind = "LL"
		case !swings[i].high && swings[i].price > swings[prev].price:
			swings[i].kind = "HL"
		default:
			swings[i].kind = "LL"
		}
	}
	return swings
}

// s1Trend computes S1's per-TF trend from closed bars exactly as
// ComputeStructureMap does: closed bars (OpenTime+iv <= nowMs) →
// s1FractalSwings → the last s1TrendSwings labelled swings: all HH/HL → up,
// all LH/LL → down, otherwise (or fewer than N swings) → range, never a guess.
func s1Trend(bars []market.Kline, tfMinutes int, nowMs int64) (trend string, swings int, closedN int) {
	iv := int64(tfMinutes) * 60_000
	var closed []market.KlineBar
	for _, k := range bars {
		if k.OpenTime+iv <= nowMs {
			closed = append(closed, market.KlineBar{Time: k.OpenTime, Open: k.Open, High: k.High, Low: k.Low, Close: k.Close, Volume: k.Volume})
		}
	}
	if len(closed) == 0 {
		return "range", 0, 0
	}
	highs := make([]float64, len(closed))
	lows := make([]float64, len(closed))
	closes := make([]float64, len(closed))
	for i, b := range closed {
		highs[i], lows[i], closes[i] = b.High, b.Low, b.Close
	}
	atr := s1SimpleATR14(highs, lows, closes)
	sw := s1FractalSwings(closed, iv, atr)
	if len(sw) >= s1TrendSwings {
		last := sw[len(sw)-s1TrendSwings:]
		up, down := true, true
		for _, s := range last {
			if s.kind != "HH" && s.kind != "HL" {
				up = false
			}
			if s.kind != "LH" && s.kind != "LL" {
				down = false
			}
		}
		switch {
		case up:
			trend = "up"
		case down:
			trend = "down"
		default:
			trend = "range"
		}
	} else {
		trend = "range"
	}
	return trend, len(sw), len(closed)
}
