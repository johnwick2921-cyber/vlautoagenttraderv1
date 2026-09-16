package kernel

import (
	"strings"

	"nofx/market"
)

// P2.4 — MAE / MFE excursion analytics.
//
// Pure: given a trade's entry, side, and the 1m bars spanning the hold, compute
// the Maximum Adverse Excursion (worst drawdown against the position) and Maximum
// Favorable Excursion (best unrealized profit reached), in points, both ≥ 0.
// These + entry confidence are logged per closed trade for later edge analysis.

// Excursion holds a trade's MAE/MFE in points (non-negative).
type Excursion struct {
	MAE float64 `json:"mae"` // max adverse excursion (points, ≥0)
	MFE float64 `json:"mfe"` // max favorable excursion (points, ≥0)
}

// ComputeExcursion scans the bars whose OpenTime falls in [entryMs, exitMs] and
// returns the MAE/MFE for a position opened at entryPrice on the given side.
func ComputeExcursion(entryPrice float64, side string, bars []market.Kline, entryMs, exitMs int64) Excursion {
	if entryPrice <= 0 || entryMs < 0 || exitMs < entryMs {
		return Excursion{}
	}
	long := isLongSide(side)
	var ex Excursion
	for i := range bars {
		b := bars[i]
		if b.OpenTime < entryMs || b.OpenTime > exitMs {
			continue
		}
		var fav, adv float64
		if long {
			fav = b.High - entryPrice // price up = favorable for a long
			adv = entryPrice - b.Low  // price down = adverse
		} else {
			fav = entryPrice - b.Low
			adv = b.High - entryPrice
		}
		if fav > ex.MFE {
			ex.MFE = fav
		}
		if adv > ex.MAE {
			ex.MAE = adv
		}
	}
	return ex
}

func isLongSide(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "LONG", "BUY":
		return true
	}
	return false
}
