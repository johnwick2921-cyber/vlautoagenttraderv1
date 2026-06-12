package ninjatrader

import (
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
)

// wireFuturesBarsProvider connects the live NT8 BarCache to the market
// package's futures kline path (Stage 3). Called once when the TCP server
// starts. The crypto/CoinAnk path is unaffected — market.GetWithTimeframes
// only calls this hook for CME futures symbols (IsCMEFuturesSymbol).
//
// The cache is keyed by the NT8 instrument symbol the AddOn subscribed to
// (e.g. "MNQ"); the kernel passes the same canonical symbol, so the lookup
// is direct with no remapping.
func wireFuturesBarsProvider(server *ntwire.TCPServer) {
	market.FuturesBarsProvider = func(symbol, timeframe string, count int) []market.Kline {
		if server == nil {
			return nil
		}
		bars := server.BarCache().Get(symbol, timeframe)
		if len(bars) == 0 {
			return nil
		}
		if count > 0 && len(bars) > count {
			bars = bars[len(bars)-count:]
		}
		return barsToKlines(bars, timeframe)
	}
}

// barsToKlines adapts the NT8 wire Bar shape to market.Kline. NT8 bars carry
// no quote-volume / trade-count / taker-split, so those Kline fields stay
// zero — the indicator engine reads only OHLCV. CloseTime is derived from
// OpenTime + the timeframe duration (NT8 bars are closed bars).
func barsToKlines(bars []ntwire.Bar, timeframe string) []market.Kline {
	durMs := timeframeDurationMs(timeframe)
	out := make([]market.Kline, len(bars))
	for i, b := range bars {
		out[i] = market.Kline{
			OpenTime:  b.T,
			Open:      b.O,
			High:      b.H,
			Low:       b.L,
			Close:     b.C,
			Volume:    b.V,
			CloseTime: b.T + durMs - 1,
		}
	}
	return out
}

// timeframeDurationMs maps a timeframe string to its millisecond span. Falls
// back to 60_000 (1m) for unrecognized values. Covers the coded TF
// vocabulary (store/strategy.go::normalizeTimeframe).
func timeframeDurationMs(timeframe string) int64 {
	switch timeframe {
	case "1m":
		return 60_000
	case "3m":
		return 180_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "30m":
		return 1_800_000
	case "1h":
		return 3_600_000
	case "2h":
		return 7_200_000
	case "4h":
		return 14_400_000
	case "6h":
		return 21_600_000
	case "8h":
		return 28_800_000
	case "12h":
		return 43_200_000
	case "1d":
		return 86_400_000
	case "3d":
		return 259_200_000
	case "1w":
		return 604_800_000
	default:
		return 60_000
	}
}
