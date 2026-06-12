package api

import (
	"testing"

	"nofx/store"
)

func TestValidateTraderLeverageRangeMatchesManualLimits(t *testing.T) {
	if msg, code := validateTraderLeverageRange(20, 20); msg != "" || code != "" {
		t.Fatalf("expected 20/20 leverage to be accepted, got msg=%q code=%q", msg, code)
	}

	if msg, code := validateTraderLeverageRange(21, 20); msg == "" || code != "trader.create.invalid_btc_eth_leverage" {
		t.Fatalf("expected BTC/ETH leverage > 20 to be rejected, got msg=%q code=%q", msg, code)
	}

	if msg, code := validateTraderLeverageRange(20, 21); msg == "" || code != "trader.create.invalid_altcoin_leverage" {
		t.Fatalf("expected altcoin leverage > 20 to be rejected, got msg=%q code=%q", msg, code)
	}
}

// TestCreateTrader_NinjaTraderExchange is a unit test for BUG 4.1.A.b2. The
// validateExchangeForTraderCreation allowlist switch at handler_trader.go:184
// must include "ninjatrader", otherwise creating a trader on an NT exchange
// returns 400 with "trader.create.exchange_unsupported". This regression catches
// the case where someone adds a new exchange type to /api/exchanges enum but
// forgets to add it here.
func TestCreateTrader_NinjaTraderExchange(t *testing.T) {
	exchange := &store.Exchange{
		ID:                   "ex-nt-test",
		ExchangeType:         "ninjatrader",
		AccountName:          "NT8 Test",
		Name:                 "NinjaTrader",
		Type:                 "futures",
		Enabled:              true,
		NTDataDir:            "/tmp/nt-test-dir",
		NTInstrumentName:     "MNQ",
		NTDefaultContractQty: 1,
	}

	msg, code, _ := validateExchangeForTraderCreation(exchange)
	if msg != "" || code != "" {
		t.Fatalf("expected NinjaTrader exchange to pass allowlist validation, got msg=%q code=%q", msg, code)
	}

	// Negative control: an unknown type must still be rejected so this test
	// asserts behavior, not just absence-of-error.
	bogus := &store.Exchange{
		ID:           "ex-bogus",
		ExchangeType: "frobnicator",
		AccountName:  "bogus",
		Enabled:      true,
	}
	msg, code, _ = validateExchangeForTraderCreation(bogus)
	if msg == "" || code != "trader.create.exchange_unsupported" {
		t.Fatalf("expected unknown exchange type to be rejected with exchange_unsupported, got msg=%q code=%q", msg, code)
	}
}
