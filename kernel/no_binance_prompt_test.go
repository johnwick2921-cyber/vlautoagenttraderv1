package kernel

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/hook"
	"nofx/market"
	"nofx/store"
)

// ── W-NO-BINANCE A — absent open interest / funding render n/a, never 0 ────
//
// The CME futures path reads no external market data (market.futuresOIFunding:
// OpenInterest nil, FundingRateKnown false). The prompt says n/a; it never
// prints the fabricated "Latest: 0.00 Average: 0.00" / "0.00e+00" the old
// path produced, and never advertises a feed that is not there.

func oiFundingEngine(coin string) *StrategyEngine {
	cfg := &store.StrategyConfig{Language: "en"}
	cfg.CoinSource.StaticCoins = []string{coin}
	cfg.Indicators.EnableOI = true
	cfg.Indicators.EnableFundingRate = true
	return &StrategyEngine{config: cfg}
}

func TestFuturesPromptRendersAbsentOIAsNA(t *testing.T) {
	e := oiFundingEngine("MNQ")
	md := e.formatMarketData(&market.Data{Symbol: "MNQ", CurrentPrice: 29000})
	if !strings.Contains(md, "Open Interest: n/a") {
		t.Fatalf("absent OI on futures must render n/a:\n%s", md)
	}
	if strings.Contains(md, "Latest: 0.00") || strings.Contains(md, "Funding Rate") {
		t.Fatalf("futures must never print a fabricated OI or any funding line:\n%s", md)
	}
	var sb strings.Builder
	e.writeAvailableIndicators(&sb)
	if !strings.Contains(sb.String(), "- Open Interest (OI) data: n/a (no external market data on the futures path)") {
		t.Fatalf("the futures Available-Data bullet must say n/a:\n%s", sb.String())
	}
	if strings.Contains(sb.String(), "- Funding rate") {
		t.Fatalf("no funding bullet on futures:\n%s", sb.String())
	}
}

func TestCryptoPromptRendersFundingOnlyWhenKnown(t *testing.T) {
	e := oiFundingEngine("BTCUSDT")
	known := e.formatMarketData(&market.Data{Symbol: "BTCUSDT", CurrentPrice: 60000,
		OpenInterest: &market.OIData{Latest: 12.5, Average: 12.4}, FundingRate: 0.0001, FundingRateKnown: true})
	if !strings.Contains(known, "Open Interest: Latest: 12.50 Average: 12.40") || !strings.Contains(known, "Funding Rate: 1.00e-04") {
		t.Fatalf("crypto with a read OI and funding renders the numbers, byte-identical to before:\n%s", known)
	}
	unknown := e.formatMarketData(&market.Data{Symbol: "BTCUSDT", CurrentPrice: 60000,
		OpenInterest: &market.OIData{Latest: 12.5, Average: 12.4}, FundingRate: 0, FundingRateKnown: false})
	if !strings.Contains(unknown, "Funding Rate: n/a") || strings.Contains(unknown, "0.00e+00") {
		t.Fatalf("a funding fetch that did not succeed is n/a, never 0.00e+00 (CTO F1):\n%s", unknown)
	}
	var sb strings.Builder
	e.writeAvailableIndicators(&sb)
	if !strings.Contains(sb.String(), "- Open Interest (OI) data\n") || !strings.Contains(sb.String(), "- Funding rate\n") {
		t.Fatalf("crypto's Available-Data bullets are unchanged:\n%s", sb.String())
	}
}

// Critic G10 — composed at the production call sites: the engine's own market
// fetch (fetchMarketDataWithStrategy → market.GetWithTimeframes on the futures
// route) feeds the prompt render, and absent OI arrives as n/a — not a
// hand-built Data literal.
func TestFuturesFetchThenRenderSaysOINA(t *testing.T) {
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		out := make([]market.Kline, 120)
		for i := range out {
			c := 29000 + float64(i%7)*1.75
			out[i] = market.Kline{OpenTime: int64(i) * 300000, Open: c - 1, High: c + 2, Low: c - 2, Close: c, Volume: 10, CloseTime: int64(i)*300000 + 299999}
		}
		return out
	}
	e := oiFundingEngine("MNQ")
	e.config.Indicators.Klines.PrimaryTimeframe = "5m"
	e.config.Indicators.Klines.SelectedTimeframes = []string{"5m"}
	e.config.Indicators.Klines.PrimaryCount = 60
	ctx := &Context{CandidateCoins: []CandidateCoin{{Symbol: "MNQ"}}}
	if err := fetchMarketDataWithStrategy(ctx, e); err != nil {
		t.Fatalf("fixture: the engine's futures fetch failed: %v", err)
	}
	d := ctx.MarketDataMap["MNQ"]
	if d == nil || d.OpenInterest != nil || d.FundingRateKnown {
		t.Fatalf("the engine's futures fetch must carry OI/funding ABSENT: %+v", d)
	}
	if md := e.formatMarketData(d); !strings.Contains(md, "Open Interest: n/a") || strings.Contains(md, "Latest: 0.00") {
		t.Fatalf("fetched futures data must render OI n/a:\n%s", md)
	}
}

// kernelTrap answers every outbound request offline and records it, on BOTH
// ways out (hook.SET_HTTP_CLIENT and http.DefaultTransport).
type kernelTrap struct {
	mu    sync.Mutex
	hosts []string
}

func (k *kernelTrap) RoundTrip(r *http.Request) (*http.Response, error) {
	k.mu.Lock()
	k.hosts = append(k.hosts, r.URL.Host)
	k.mu.Unlock()
	return nil, fmt.Errorf("no network in this test")
}

func trapOutbound(t *testing.T) *kernelTrap {
	t.Helper()
	k := &kernelTrap{}
	prev, had := hook.Hooks[hook.SET_HTTP_CLIENT]
	prevEnabled := hook.EnableHooks
	hook.EnableHooks = true
	hook.RegisterHook(hook.SET_HTTP_CLIENT, func(args ...any) any {
		return &hook.SetHttpClientResult{Client: &http.Client{Transport: k, Timeout: time.Second}}
	})
	prevDT := http.DefaultTransport
	http.DefaultTransport = k
	t.Cleanup(func() {
		http.DefaultTransport = prevDT
		if had {
			hook.Hooks[hook.SET_HTTP_CLIENT] = prev
		} else {
			delete(hook.Hooks, hook.SET_HTTP_CLIENT)
		}
		hook.EnableHooks = prevEnabled
	})
	return k
}

// CTO F2 — the engine's cycle read on the NinjaTrader venue never reads a
// non-CME symbol from a crypto source: fetchMarketDataWithStrategy (the
// production call site) refuses it, with ZERO outbound requests, while the
// CME candidate is read from the NT8 bars.
func TestEngineCycleReadOnNT8RefusesANonCMESymbol(t *testing.T) {
	trap := trapOutbound(t)
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		out := make([]market.Kline, 120)
		for i := range out {
			c := 29000 + float64(i%7)*1.75
			out[i] = market.Kline{OpenTime: int64(i) * 300000, Open: c - 1, High: c + 2, Low: c - 2, Close: c, Volume: 10, CloseTime: int64(i)*300000 + 299999}
		}
		return out
	}
	e := oiFundingEngine("MNQ")
	e.SetVenue("ninjatrader")
	e.config.Indicators.Klines.PrimaryTimeframe = "5m"
	e.config.Indicators.Klines.SelectedTimeframes = []string{"5m"}
	e.config.Indicators.Klines.PrimaryCount = 60
	ctx := &Context{CandidateCoins: []CandidateCoin{{Symbol: "MNQ"}, {Symbol: "BTCUSDT"}}}
	_ = fetchMarketDataWithStrategy(ctx, e)
	if _, ok := ctx.MarketDataMap["BTCUSDT"]; ok {
		t.Fatalf("a non-CME symbol on the NinjaTrader venue must never be read: %+v", ctx.MarketDataMap["BTCUSDT"])
	}
	if _, ok := ctx.MarketDataMap["MNQ"]; !ok {
		t.Fatalf("fixture: the CME candidate must still be read from the NT8 bars")
	}
	trap.mu.Lock()
	defer trap.mu.Unlock()
	if len(trap.hosts) != 0 {
		t.Fatalf("the NT8 cycle read went out to %v", trap.hosts)
	}
}
