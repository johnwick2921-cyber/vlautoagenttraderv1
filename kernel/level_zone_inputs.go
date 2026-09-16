package kernel

import (
	"math"
	"nofx/market"
	"time"
)

// The defining wick is on the selected swing side, not the opposite tail.
func ZonePivotWick(b market.Kline, high bool) float64 {
	if high {
		return b.High - math.Max(b.Open, b.Close)
	}
	return math.Min(b.Open, b.Close) - b.Low
}

func ZoneBarWick(b market.Kline) float64 {
	return math.Max(b.High-math.Max(b.Open, b.Close), math.Min(b.Open, b.Close)-b.Low)
}

// Consume the SAME frozen series used by detection, never a later live read.
// Missing defining-bar evidence remains NULL. A confirmation candle is not
// silently substituted for a pivot's defining candle.
func LevelZoneInputs(raw []DetectedLevel, series map[string][]market.Kline, now time.Time) map[string]ZoneWidthInput {
	out := map[string]ZoneWidthInput{}
	closed := map[string][]market.Kline{}
	for tf, bars := range series {
		for _, b := range bars {
			if b.CloseTime < now.UnixMilli() {
				closed[tf] = append(closed[tf], b)
			}
		}
	}
	base := closed["1m"]
	atr := map[string]float64{}
	for tf, bars := range closed {
		atr[tf] = market.ExportCalculateATR(bars, 14)
	}
	for _, l := range raw {
		tf := l.TF
		if tf == "" {
			tf = "1m"
		}
		in := ZoneWidthInput{ATR: atr[tf], Wick: l.ZoneDefiningWick}
		if in.Wick == nil && l.FormedAtMs > 0 {
			for _, b := range closed[tf] {
				if b.OpenTime == l.FormedAtMs {
					w := ZoneBarWick(b)
					in.Wick = &w
					break
				}
			}
		}
		// Only count a complete available post-formation window. Unknown birth
		// or missing early tape is NULL, never a plausible zero. Episodes are
		// detector-defined observations, not an asserted bounce probability.
		if l.FormedCloseMs != nil && len(base) > 0 && base[0].OpenTime <= *l.FormedCloseMs {
			post := []market.Kline{}
			for _, b := range base {
				if b.OpenTime >= *l.FormedCloseMs {
					post = append(post, b)
				}
			}
			delta := MeanAbsIncrement(base)
			if delta > 0 && len(post) > DetectorHorizonBars() {
				eps := DetectTouchOutcomes(post, l.Price, DetectorK(), delta, DetectorHorizonBars(), DetectorExitOn())
				n := 0
				for _, e := range eps {
					if e.ClosedAtMs > 0 && e.ClosedAtMs <= now.UnixMilli() {
						n++
					}
				}
				in.PriorTouches = &n
			}
		}
		out[ZoneSourceKey(l)] = in
	}
	return out
}
