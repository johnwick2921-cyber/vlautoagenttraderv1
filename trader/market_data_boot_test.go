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
	if !strings.Contains(got, "bars=NT8 BarCache") || !strings.Contains(got, "oi/funding: n/a (no external market data on the futures path)") ||
		strings.Contains(got, "REFUSED") {
		t.Fatalf("the line must READ the provider and the route of the NT8 trader's MNQ: %q", got)
	}
	if got := MarketDataBootLine(map[string]*AutoTrader{}); !strings.Contains(got, "futures traders=0 bars=n/a") || !strings.Contains(got, "oi/funding: n/a (no NinjaTrader trader loaded)") {
		t.Fatalf("no trader loaded → bars and oi/funding n/a, never a literal: %q", got)
	}
	// Critic G1/G5: an NT8 trader configured with a NON-CME symbol is on the
	// refused route — the line says so, READ from the same route decision.
	odd := &AutoTrader{exchange: "ninjatrader"}
	odd.config.NinjaTraderSymbol = "BTCUSDT"
	if got := MarketDataBootLine(map[string]*AutoTrader{"a": nt, "b": odd}); !strings.Contains(got, "REFUSED on the NinjaTrader venue (non-CME symbol): BTCUSDT") {
		t.Fatalf("a non-CME symbol on the NT8 venue must read REFUSED: %q", got)
	}
}
