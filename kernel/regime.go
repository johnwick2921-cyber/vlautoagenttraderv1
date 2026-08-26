package kernel

import (
	"fmt"
	"math"
	"strings"

	"nofx/market"
)

// P1.6 — REGIME block (7 auto fields, zero config).
//
// A deterministic, planner-INPUT context summary (never a card readout). Every
// field degrades gracefully: missing data → "n/a"/WARMING rather than a fake
// number. VIX has no feed in this repo (recon) and realized-vol-vs-20d needs a
// warmed baseline, so both report unavailable/warming until wired + warmed.

// RegimeInputs are the (partial) series the caller can supply. Any empty series
// yields an "n/a" field — honest cold-start behavior.
type RegimeInputs struct {
	Price         float64
	DailyBars     []market.Kline // EMA200-daily trend, dATR, ATR14 percentile
	Hour1Bars     []market.Kline // 1h trend
	Min5Bars      []market.Kline // realized vol
	PriorClose    float64        // prior session close (overnight gap)
	SessionOpen   float64        // this session's open (overnight gap)
	VIX           float64        // 0 = unavailable (no feed)
	RVBaseline20d float64        // 0 = warming (no 20d baseline yet)
}

// RegimeBlock is the computed regime snapshot.
type RegimeBlock struct {
	TrendDaily       string  `json:"trend_daily"`  // up | down | flat | n/a
	Trend1h          string  `json:"trend_1h"`     // up | down | flat | n/a
	ATR14            float64 `json:"atr14"`        // daily ATR (dATR); 0 = n/a
	ATRRegime        string  `json:"atr_regime"`   // LOW | NORMAL | HIGH | EXTREME | n/a
	ATRPercentile    float64 `json:"atr_pctile"`   // 0..100 (−1 = n/a)
	RealizedVolPct   float64 `json:"rv_pct"`       // recent daily RV%, or recent/baseline%
	RVWarming        bool    `json:"rv_warming"`   // true → no 20d baseline yet
	VIXLevel         float64 `json:"vix_level"`    // 0 = unavailable
	VIXRegime        string  `json:"vix_regime"`   // <15 | 15-20 | 20-30 | >30 | unavailable
	ExpectedRangePts float64 `json:"expected_range_pts"`
	OvernightGapATR  float64 `json:"overnight_gap_atr"` // gap / dATR (0 if unavailable)
	HasGap           bool    `json:"has_gap"`
}

const regimeTrendEps = 0.0005 // ±0.05% dead-band around the MA → "flat"

// ComputeRegime builds the regime block from whatever inputs are present.
func ComputeRegime(in RegimeInputs) RegimeBlock {
	r := RegimeBlock{
		TrendDaily: "n/a", Trend1h: "n/a", ATRRegime: "n/a", ATRPercentile: -1,
		VIXRegime: "unavailable",
	}

	// trend_state_daily: price vs EMA200 (daily).
	if in.Price > 0 && len(in.DailyBars) >= 200 {
		if ema := market.ExportCalculateEMA(in.DailyBars, 200); ema > 0 {
			r.TrendDaily = trendVsMA(in.Price, ema)
		}
	}
	// trend_state_1h: price vs EMA50 (1h).
	if in.Price > 0 && len(in.Hour1Bars) >= 50 {
		if ema := market.ExportCalculateEMA(in.Hour1Bars, 50); ema > 0 {
			r.Trend1h = trendVsMA(in.Price, ema)
		}
	}

	// ATR14 (daily = dATR) + percentile regime.
	if len(in.DailyBars) >= 15 {
		r.ATR14 = market.ExportCalculateATR(in.DailyBars, 14)
		if r.ATR14 > 0 && len(in.DailyBars) >= 30 {
			series := rollingDailyATR(in.DailyBars, 14)
			if len(series) >= 10 {
				r.ATRPercentile = percentileRank(series, r.ATR14)
				r.ATRRegime = atrBucket(r.ATRPercentile)
			}
		}
	}

	// realized_vol_pct (5m RV; vs-20d when a baseline is supplied).
	if rv, ok := recentDailyVolPct(in.Min5Bars); ok {
		if in.RVBaseline20d > 0 {
			r.RealizedVolPct = 100 * rv / in.RVBaseline20d
			r.RVWarming = false
		} else {
			r.RealizedVolPct = rv
			r.RVWarming = true
		}
	} else {
		r.RVWarming = true
	}

	// VIX (no feed → unavailable).
	if in.VIX > 0 {
		r.VIXLevel = in.VIX
		r.VIXRegime = vixBucket(in.VIX)
	}

	// expected_range_pts: VIX-implied when available, else the ATR cross-check.
	switch {
	case in.VIX > 0 && in.Price > 0:
		r.ExpectedRangePts = in.VIX / 100 / math.Sqrt(252) * in.Price
	case r.ATR14 > 0:
		r.ExpectedRangePts = r.ATR14
	}

	// overnight_gap (×ATR).
	if in.PriorClose > 0 && in.SessionOpen > 0 && r.ATR14 > 0 {
		r.OvernightGapATR = (in.SessionOpen - in.PriorClose) / r.ATR14
		r.HasGap = true
	}

	return r
}

func trendVsMA(price, ma float64) string {
	switch {
	case price > ma*(1+regimeTrendEps):
		return "up"
	case price < ma*(1-regimeTrendEps):
		return "down"
	default:
		return "flat"
	}
}

// rollingDailyATR computes ATR(period) as of each day (from day period+1 on).
func rollingDailyATR(daily []market.Kline, period int) []float64 {
	var out []float64
	for i := period + 1; i <= len(daily); i++ {
		if v := market.ExportCalculateATR(daily[:i], period); v > 0 {
			out = append(out, v)
		}
	}
	return out
}

func percentileRank(series []float64, v float64) float64 {
	if len(series) == 0 {
		return 0
	}
	below := 0
	for _, x := range series {
		if x < v {
			below++
		}
	}
	return 100 * float64(below) / float64(len(series))
}

func atrBucket(p float64) string {
	switch {
	case p < 25:
		return "LOW"
	case p < 75:
		return "NORMAL"
	case p < 90:
		return "HIGH"
	default:
		return "EXTREME"
	}
}

func vixBucket(v float64) string {
	switch {
	case v < 15:
		return "<15"
	case v < 20:
		return "15-20"
	case v <= 30:
		return "20-30"
	default:
		return ">30"
	}
}

// recentDailyVolPct returns the recent realized daily volatility (%) from 5m
// close-to-close returns, annualized to one day (√288 5m bars/day). ok=false
// when there is too little data.
func recentDailyVolPct(min5 []market.Kline) (float64, bool) {
	if len(min5) < 50 {
		return 0, false
	}
	rets := make([]float64, 0, len(min5)-1)
	for i := 1; i < len(min5); i++ {
		if min5[i-1].Close > 0 && min5[i].Close > 0 {
			rets = append(rets, math.Log(min5[i].Close/min5[i-1].Close))
		}
	}
	if len(rets) < 30 {
		return 0, false
	}
	var mean float64
	for _, x := range rets {
		mean += x
	}
	mean /= float64(len(rets))
	var varSum float64
	for _, x := range rets {
		d := x - mean
		varSum += d * d
	}
	sd := math.Sqrt(varSum / float64(len(rets)))
	return sd * math.Sqrt(288) * 100, true
}

// Render produces the one-line planner-input annotation (max one line).
func (r RegimeBlock) Render() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "REGIME: trend D=%s 1h=%s", r.TrendDaily, r.Trend1h)
	if r.ATR14 > 0 {
		if r.ATRRegime != "n/a" {
			fmt.Fprintf(&sb, " · ATR14=%.1f (%s p%.0f)", r.ATR14, r.ATRRegime, r.ATRPercentile)
		} else {
			fmt.Fprintf(&sb, " · ATR14=%.1f", r.ATR14)
		}
	}
	if r.RealizedVolPct > 0 {
		if r.RVWarming {
			fmt.Fprintf(&sb, " · RV=%.1f%%(warming)", r.RealizedVolPct)
		} else {
			fmt.Fprintf(&sb, " · RV=%.0f%%-of-normal", r.RealizedVolPct)
		}
	}
	if r.VIXLevel > 0 {
		fmt.Fprintf(&sb, " · VIX=%.1f(%s)", r.VIXLevel, r.VIXRegime)
	} else {
		sb.WriteString(" · VIX=n/a")
	}
	if r.ExpectedRangePts > 0 {
		fmt.Fprintf(&sb, " · exp-range≈%.0fpts", r.ExpectedRangePts)
	}
	if r.HasGap {
		fmt.Fprintf(&sb, " · o/n-gap=%+.2f×ATR", r.OvernightGapATR)
	}
	return sb.String()
}
