package mentor

import (
	"fmt"

	"vl/market"
)

// KeyLevels implements §4.3 key levels [D5.3 p1 @ 12:55–17:30]:
//
//  1. the chart session is RTH (KeyLevelRTHOnly);
//  2. each candle colour (close > open = green) is one trend;
//  3. at EVERY colour change a line is drawn at the OPEN of the SECOND candle
//     (red → green: open of the green candle; green → red: open of the red);
//  4. PRUNE: two levels less than KeyLevelPrunePts apart → delete one, keep
//     the more recent.
//
// The walk runs on the TF given by KeyLevelTFMinutes (default 1m — the RTH
// chart the mentor draws on; DS-108 §2.1 measured ~200 colour changes per RTH
// day on 1m). F7 (LEVELS-KEEP fold): NO lookback cap — every bar passed in is
// walked ("key level nó là QUÁ KHỨ — muốn vẽ bao nhiêu tùy thích"
// [D5.3 p1 @ 14:09–14:16]); a cap would be a knob defaulting to ALL and
// logging the held count, and none exists. The prune applies to the FINAL set
// (the method's intent: no two surviving lines closer than the prune distance
// — a smaller gap is unreadable): levels are kept newest-first, dropping any
// candidate within the prune distance of an already-kept (newer) level, so of
// every close pair the more recent survives.
func KeyLevels(bars []market.Kline, cfg Config) []Level {
	if !cfg.Enabled || cfg.KeyLevelTFMinutes <= 0 {
		return nil
	}
	tf := barsTF(bars, cfg.KeyLevelTFMinutes)
	walk := tf
	if cfg.KeyLevelRTHOnly {
		walk = rthOnly(tf)
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
