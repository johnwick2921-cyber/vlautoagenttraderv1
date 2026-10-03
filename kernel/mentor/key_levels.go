package mentor

import (
	"fmt"

	"vl/market"
)

// KeyLevels implements the KEY-LEVEL RULING (final, slide 31, verified from
// the .pptx) — the level SOURCE is the 1H REGULAR-trading-hours series,
// anchored at the market open 08:30 CT:
//
//  1. bars = 1H RTH only: candles anchored at 08:30 CT (first candle
//     08:30–09:29), external (ETH) candles excluded [slide 31: "Khung 1H ONLY
//     · Không tính external hours · Bắt đầu từ lúc market open"];
//  2. each candle colour (close > open = green) is one trend;
//  3. at EVERY colour change a line is drawn at the OPEN of the SECOND candle
//     (red → green: open of the green candle; green → red: open of the red).
//     NEVER the wick. Draw them even after a gap [D5.3 p1 @ 14:24–15:58];
//  4. PRUNE: two levels less than KeyLevelPrunePts (20) apart → delete one,
//     keep the more recent [@ 16:08–16:44];
//  5. DELETE: a level is deleted when a 1H RTH candle's BODY closes through
//     it (open one side, close the other) — handled by the evaluator via
//     levelDeletedBy1HBody, not here (deletion is session state).
//
// The 2025 tiers (15m ETH wicks, 30–60 spacing, 4h red recolouring) are
// SUPERSEDED and not built. No lookback cap (F7 stands).
//
// When the knob KeyLevelTFMinutes is 60 (the default) the walk runs on the
// anchored 1H RTH candles; any other TF walks barsTF(tf) + RTH-open filtering
// for experimentation only.
func KeyLevels(bars []market.Kline, cfg Config) []Level {
	if !cfg.Enabled || cfg.KeyLevelTFMinutes <= 0 {
		return nil
	}
	var walk []market.Kline
	if cfg.KeyLevelTFMinutes == 60 {
		walk = keyLevel1HBars(bars)
	} else {
		walk = barsTF(bars, cfg.KeyLevelTFMinutes)
		if cfg.KeyLevelRTHOnly {
			walk = rthOnly(walk)
		}
	}
	if len(walk) < 2 {
		return nil
	}
	var cand []Level
	for i := 1; i < len(walk); i++ {
		if candleColour(walk[i-1]) == candleColour(walk[i]) {
			continue
		}
		cand = append(cand, Level{
			Key:    keyLevelKey(walk[i]),
			Kind:   KindKeyLevel,
			Price:  walk[i].Open,
			AtTime: walk[i].OpenTime,
		})
	}
	// newest-first keep: a level survives unless a newer kept level sits
	// within the prune distance.
	var kept []Level
	for i := len(cand) - 1; i >= 0; i-- {
		l := cand[i]
		drop := false
		for _, k := range kept {
			if abs(l.Price-k.Price) < cfg.KeyLevelPrunePts {
				drop = true
				break
			}
		}
		if !drop {
			kept = append(kept, l)
		}
	}
	out := make([]Level, len(kept))
	for i, l := range kept {
		out[len(kept)-1-i] = l // restore time order
	}
	return out
}

// keyLevel1HBars buckets the 1m history into 1H candles ANCHORED AT THE
// MARKET OPEN 08:30 CT (KEY-LEVEL RULING item 1): the first candle is
// 08:30–09:29. Only candles OPENING inside RTH survive and bars opening at or
// after 15:00 CT are dropped (external minutes never contaminate the 14:30
// candle). A PRE-BUCKETED 1H input (bars already ~60m apart, e.g. the
// recorded 1h fixtures) cannot reconstruct the 08:30 anchor: each bar is a
// candle and the RTH filter keeps candles opening in [08:00, 15:00) — the
// hour that contains the market open. The evaluator always feeds 1m bars.
func keyLevel1HBars(bars []market.Kline) []market.Kline {
	const (
		anchorMin = 8*60 + 30 // 08:30 CT
		rthEnd    = 15 * 60   // 15:00 CT
	)
	// pre-bucketed 1H input: every bar is already a candle
	if len(bars) >= 2 && bars[1].OpenTime-bars[0].OpenTime >= 60*60_000 {
		res := bars[:0]
		for _, b := range bars {
			m := (b.OpenTime / 60_000) % (24 * 60)
			if m >= 8*60 && m < rthEnd {
				res = append(res, b)
			}
		}
		return res
	}
	openMin := func(t int64) int64 {
		t /= 60_000 // minutes, CT basis (DS-108 §1.2)
		d := t - anchorMin
		day := d / (24 * 60)
		rem := d % (24 * 60)
		if rem < 0 {
			day--
			rem += 24 * 60
		}
		return (day*(24*60) + anchorMin + (rem/60)*60) * 60_000
	}
	var out []market.Kline
	var cur *market.Kline
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	key := int64(-1)
	for _, b := range bars {
		if (b.OpenTime/60_000)%(24*60) >= rthEnd {
			continue // post-close external minutes never enter the last candle
		}
		k := openMin(b.OpenTime)
		if k != key {
			flush()
			c := b
			c.OpenTime = k // the anchored candle opens at 08:30, 09:30, …
			cur = &c
			key = k
			continue
		}
		if b.High > cur.High {
			cur.High = b.High
		}
		if b.Low < cur.Low {
			cur.Low = b.Low
		}
		cur.Close = b.Close
		cur.CloseTime = b.CloseTime
	}
	flush()
	res := out[:0]
	for _, c := range out {
		m := (c.OpenTime / 60_000) % (24 * 60)
		if m >= anchorMin && m < rthEnd {
			res = append(res, c)
		}
	}
	return res
}

// rthOnly keeps bars opened inside regular trading hours 08:30–15:00 CT.
// RTH membership is judged on the bar OPEN in CT. (The bar timestamps stored
// by the bot are CT-based epoch millis — DS-108 §1.2.)
func rthOnly(bars []market.Kline) []market.Kline {
	out := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		_, hh, mm := ctOf(b.OpenTime)
		from := 8*60 + 30
		to := 15 * 60
		t := hh*60 + mm
		if t >= from && t < to {
			out = append(out, b)
		}
	}
	return out
}

// candleColour: green iff close > open, red otherwise (§4.3 step 2).
func candleColour(b market.Kline) bool {
	return b.Close > b.Open
}

// levelDeletedBy1HBody implements the KEY-LEVEL RULING deletion (final,
// slide 31 step 4 + the slide-32 correction): a level is deleted when a 1H
// RTH candle CLOSES THROUGH it — its open on one side of the level, its
// close on the other (the BODY crosses; a wick through does NOT). The 1H
// candles are the SAME anchored 08:30 CT RTH series the level walk uses
// (keyLevel1HBars), and only a candle that CLOSED at or after the level was
// drawn can delete it. The still-forming candle (whose close time has not
// been reached) never counts.
func levelDeletedBy1HBody(lvl Level, bars []market.Kline, now int64) bool {
	b60 := keyLevel1HBars(bars)
	if len(b60) > 0 && b60[len(b60)-1].CloseTime >= now {
		b60 = b60[:len(b60)-1] // the forming 1H candle has not closed
	}
	for _, b := range b60 {
		if b.CloseTime < lvl.AtTime {
			continue // this 1H candle closed before the level existed
		}
		crossed := b.Open < lvl.Price && b.Close > lvl.Price ||
			b.Open > lvl.Price && b.Close < lvl.Price
		if crossed {
			return true // the BODY closed through the level
		}
	}
	return false
}

func keyLevelKey(b market.Kline) string {
	return fmt.Sprintf("%s:%s:%d", KindKeyLevel, fnum(b.Open), b.OpenTime)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func fnum(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

// ctOf splits a CT-based epoch-millis bar time into CT date and hh:mm.
// epoch_floor convention per the bars table (DS-108 §1.2: maintenance gap
// lands at 17:00–18:00 CT, which confirms the CT basis).
func ctOf(ms int64) (day int, hh, mm int) {
	t := ms / 1000 / 60 // minutes since epoch in CT
	day = int(t / (24 * 60))
	mod := int(t % (24 * 60))
	return day, mod / 60, mod % 60
}
