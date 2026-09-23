package trader

import (
	"fmt"

	"nofx/market"
)

// MarketDataBootLine — W-NO-BINANCE A. Replaces main.go's literal
// "📊 Using CoinAnk API for all market data (WebSocket cache disabled)", which
// was false on the futures path (audit H20: futures reads the NT8 BarCache,
// never CoinAnk) and said nothing about the Binance OI/funding calls every AI
// open made. Every field is READ: the trader counts from the loaded traders
// (the NT8 exchange IS the futures path), bars from whether the NT8 bar
// provider is wired, and oi/funding from market.FuturesOIFundingBootLine —
// the same function the futures market read calls. A field the process cannot
// know yet prints n/a.
func MarketDataBootLine(loaded map[string]*AutoTrader) string {
	futures, other := 0, 0
	for _, at := range loaded {
		if at == nil {
			continue
		}
		if at.GetExchange() == "ninjatrader" {
			futures++
		} else {
			other++
		}
	}
	bars := "n/a"
	if futures > 0 {
		bars = "UNWIRED"
		if market.FuturesBarsProvider != nil {
			bars = "NT8 BarCache"
		}
	}
	return fmt.Sprintf("📊 market data: futures traders=%d bars=%s · %s · non-futures traders=%d",
		futures, bars, market.FuturesOIFundingBootLine(), other)
}
