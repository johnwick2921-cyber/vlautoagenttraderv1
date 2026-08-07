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

// TestValidateNTAccountBoundForStart locks the NT8 multi-account start guard: a
// NinjaTrader trader with no bound account is refused (so it can never trade on the
// shared active account); a bound one passes; non-NT exchanges are never blocked.
func TestValidateNTAccountBoundForStart(t *testing.T) {
	ntCfg := func(acct string) *store.TraderFullConfig {
		return &store.TraderFullConfig{
			Exchange: &store.Exchange{ExchangeType: "ninjatrader"},
			Trader:   &store.Trader{Name: "t", Account: acct},
		}
	}
	// Empty account → blocked with the actionable code.
	if msg, code, _ := validateNTAccountBoundForStart(ntCfg(""), "t"); msg == "" || code != "trader.start.no_account" {
		t.Fatalf("empty NT account must block start; got msg=%q code=%q", msg, code)
	}
	// Whitespace-only account → also blocked.
	if msg, _, _ := validateNTAccountBoundForStart(ntCfg("   "), "t"); msg == "" {
		t.Fatalf("whitespace-only NT account must block start")
	}
	// Bound account → allowed (no block).
	if msg, code, _ := validateNTAccountBoundForStart(ntCfg("Sim101"), "t"); msg != "" || code != "" {
		t.Fatalf("bound NT account must pass; got msg=%q code=%q", msg, code)
	}
	// Non-NinjaTrader exchange with empty account → not blocked (no account concept).
	crypto := &store.TraderFullConfig{Exchange: &store.Exchange{ExchangeType: "binance"}, Trader: &store.Trader{Account: ""}}
	if msg, _, _ := validateNTAccountBoundForStart(crypto, "t"); msg != "" {
		t.Fatalf("non-NT exchange must not be blocked by the NT account rule; got msg=%q", msg)
	}
}
