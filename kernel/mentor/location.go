package mentor

import (
	"fmt"

	"vl/market"
)

// LocationVerdict implements the LOCATION GATE (fold item 1, owner ruling:
// setups happen ONLY at important levels — all three setups, ISB included).
// A setup's reference candle must touch one of:
//
//   - a key level (§4.3);
//   - EMA 34 on a HIGHER timeframe (knob EMALocationTFMinutes, default 4h =
//     240; allowed 60/15/5 — NEVER 1m);
//   - the 5m trigger-line retest [D3.4 p2 @ 20:51];
//   - (FTGH/FTGL box edges are out of v1 — boxes are not built).
//
// An old high/low ALONE is not a location: it counts only when it coincides
// within LocationCoincidePts (±2) of a key level. EMA 9 is a reverse-ISB
// location only (§10) — never a location for ISB/PHL/PLH in v1.
//
// levels may be the plain level set; the verdict recomputes the EMA34-HTF line
// from the bar history so a moving line is judged at its CURRENT value.
func LocationVerdict(ref market.Kline, levels []Level, trigger TriggerLine, bars []market.Kline, cfg Config) (ok bool, where string) {
	if !cfg.Enabled {
		return false, ""
	}
	if cfg.EMALocationTFMinutes == 1 {
		// NEVER 1m by default (owner ruling) — a 1-minute EMA is not a
		// location. Treated as absent, not an error.
	} else if len(bars) > 0 && (cfg.EMALocationTFMinutes == 240 || cfg.EMALocationTFMinutes == 60 ||
		cfg.EMALocationTFMinutes == 15 || cfg.EMALocationTFMinutes == 5) {
		v := emaValue(barsTF(bars, cfg.EMALocationTFMinutes), cfg.EMAPeriod34)
		if touchesPrice(ref, v, cfg.TouchBandPts) {
			return true, fmt.Sprintf("ema34_%dm", cfg.EMALocationTFMinutes)
		}
	}
	for _, l := range levels {
		switch l.Kind {
		case KindKeyLevel:
			if touchesPrice(ref, l.Price, cfg.TouchBandPts) {
				return true, "key_level"
			}
		case KindOldExtreme:
			// only when it coincides ±2 pts with a key level
			if coincidesWithKeyLevel(l, levels) && touchesPrice(ref, l.Price, cfg.TouchBandPts) {
				return true, "old_extreme@key_level"
			}
		case KindTriggerRetest:
			if touchesPrice(ref, l.Price, cfg.TouchBandPts) {
				return true, "trigger_retest"
			}
		}
		// KindEMA9 is deliberately NOT a location (reverse-ISB only, §10).
	}
	if trigger.Price != 0 && trigger.Dir != "" && touchesPrice(ref, trigger.Price, cfg.TouchBandPts) {
		return true, "trigger_line"
	}
	return false, ""
}

// LocationCoincidePts is the ±2-pt coincidence for an old high/low next to a
// key level (fold item 1).
const LocationCoincidePts = 2.0

func coincidesWithKeyLevel(l Level, levels []Level) bool {
	for _, k := range levels {
		if k.Kind == KindKeyLevel && abs(k.Price-l.Price) <= LocationCoincidePts {
			return true
		}
	}
	return false
}

// touchesPrice reports a literal touch: the candle's range reaches the price
// within the band (§3 step 1).
func touchesPrice(ref market.Kline, price, band float64) bool {
	return ref.High >= price-band && ref.Low <= price+band
}

// levelIsLocation reports whether a touched LEVEL itself is a valid entry
// location (PHL/PLH at it): key levels, the EMA34-HTF line, the trigger retest
// — or an old extreme coinciding ±2 pts with a key level. A bare old high/low
// is NOT a location (fold item 1).
func levelIsLocation(lvl Level, levels []Level) bool {
	switch lvl.Kind {
	case KindKeyLevel, KindEMA34HTF, KindTriggerRetest:
		return true
	case KindOldExtreme:
		return coincidesWithKeyLevel(lvl, levels)
	}
	return false
}

// EMALocationLevel returns the location-gate EMA 34 line as a Level (for the
// evaluator's PHL/PLH entry-level check, where the touched level itself must
// be a location).
func EMALocationLevel(bars []market.Kline, cfg Config) (Level, bool) {
	if !cfg.Enabled || cfg.EMALocationTFMinutes == 1 ||
		!(cfg.EMALocationTFMinutes == 240 || cfg.EMALocationTFMinutes == 60 ||
			cfg.EMALocationTFMinutes == 15 || cfg.EMALocationTFMinutes == 5) {
		return Level{}, false
	}
	v := emaValue(barsTF(bars, cfg.EMALocationTFMinutes), cfg.EMAPeriod34)
	return Level{
		Key:   string(KindEMA34HTF),
		Kind:  KindEMA34HTF,
		Price: v,
	}, true
}
