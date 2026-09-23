package kernel

import (
	"strings"
	"testing"

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
