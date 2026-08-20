package ninjatrader

import (
	"nofx/kernel"
	"nofx/logger"
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
// OpenTime + the timeframe duration. A10 (T7) comment-truth: the NEWEST bar is
// usually FORMING — the AddOn re-emits the same-open bar as it builds and the
// cache replaces it in place; CloseTime here is the bar's SCHEDULED close, not
// proof it closed. Consumers that need closed bars filter CloseTime < now
// (latestClosedPrimaryBarMs, closedBars, acceptance buckets).
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
	// B1 (T3): delegate to THE one table. The old unknown→60_000 default made
	// the bridge fabricate CloseTimes for unmapped TFs (a forming bar could
	// count as closed, corrupting the bar-close gate + supersession watermark).
	// Unknown now returns 0 and the caller logs loudly; the PRIMARY timeframe
	// is boot-validated upstream so this is a defensive edge only.
	if ms, ok := kernel.TFDurationMs(timeframe); ok {
		return ms
	}
	logger.Warnf("🧪 bars bridge: UNMAPPED timeframe %q — bars passed through without a derived CloseTime (add it to kernel/timeframes.go)", timeframe)
	return 0
}
