package trader

import (
	"strings"
	"testing"

	"nofx/market"
)

// W-NO-BINANCE A — the 📊 market-data boot line is READ.
func TestMarketDataBootLineIsRead(t *testing.T) {
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	nt := &AutoTrader{exchange: "ninjatrader"}
	crypto := &AutoTrader{exchange: "okx"}
	market.FuturesBarsProvider = nil
	if got := MarketDataBootLine(map[string]*AutoTrader{"a": nt, "b": crypto}); !strings.Contains(got, "futures traders=1 bars=UNWIRED") || !strings.Contains(got, "non-futures traders=1") {
		t.Fatalf("an NT8 trader with no bar provider must read UNWIRED: %q", got)
	}
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return nil }
	got := MarketDataBootLine(map[string]*AutoTrader{"a": nt})
	if !strings.Contains(got, "bars=NT8 BarCache") || !strings.Contains(got, market.FuturesOIFundingBootLine()) ||
		!strings.Contains(got, "oi/funding: n/a (no external market data on the futures path)") {
		t.Fatalf("the line must READ the provider and the futures OI/funding decision: %q", got)
	}
	if got := MarketDataBootLine(map[string]*AutoTrader{}); !strings.Contains(got, "futures traders=0 bars=n/a") {
		t.Fatalf("no trader loaded → bars n/a, never a literal: %q", got)
	}
}
