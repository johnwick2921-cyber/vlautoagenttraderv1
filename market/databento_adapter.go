package market

import (
	"nofx/provider/databento"
)

// BarsToKlines converts Databento bars into the project's canonical Kline shape.
// This is the single bridge that lets every existing indicator, formatter, and
// strategy work with NQ data unchanged.
//
// Databento OHLCV records don't include quote-volume, trade count, or taker-side
// breakdowns, so those fields are left at zero. Downstream indicators (EMA, RSI,
// MACD, ATR, BOLL) only read OpenTime/OHLC/Volume.
func BarsToKlines(bars []databento.Bar) []Kline {
	if len(bars) == 0 {
		return nil
	}
	out := make([]Kline, 0, len(bars))
	for _, b := range bars {
		out = append(out, Kline{
			OpenTime: b.Timestamp.UnixMilli(),
			Open:     b.Open,
			High:     b.High,
			Low:      b.Low,
			Close:    b.Close,
			Volume:   b.Volume,
		})
	}
	return out
}
