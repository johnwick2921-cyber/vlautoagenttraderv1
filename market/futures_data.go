package market

// FuturesBarsProvider, when non-nil, supplies OHLCV klines for CME futures
// symbols from the live NT8 bar feed (provider/ninjatrader BarCache),
// bypassing the crypto CoinAnk path entirely. It is wired at startup by the
// ninjatrader transport layer (trader/ninjatrader) so the market package
// itself takes no provider/trader dependency — keeping the dependency edge
// one-directional (trader/ninjatrader -> market) and cycle-free.
//
// Contract: returns up to the most recent `count` bars for (symbol,
// timeframe) in ascending time order, or nil if none are cached yet. A nil
// provider (crypto-only build, or NT transport not started) means the
// futures branch in GetWithTimeframes simply skips the symbol.
var FuturesBarsProvider func(symbol, timeframe string, count int) []Kline
