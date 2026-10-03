package store

import "testing"

// TestApplyFuturesIndicatorDefaults verifies the futures new-strategy indicator
// defaults: the technical indicators ATR/EMA/RSI are enabled (the futures prompt
// leans on them — ATR sizes stops), Open Interest (the Binance crypto-perp feed)
// stays off, and MACD/BOLL are deliberately left off.
func TestApplyFuturesIndicatorDefaults(t *testing.T) {
	// Start from the crypto base defaults: technical indicators OFF.
	ind := IndicatorConfig{
		EnableEMA: false, EnableRSI: false, EnableATR: false,
		EnableMACD: false, EnableBOLL: false,
		EnableOI: true,
	}
	applyFuturesIndicatorDefaults(&ind)

	if !ind.EnableATR || !ind.EnableEMA || !ind.EnableRSI {
		t.Errorf("futures default must enable ATR/EMA/RSI; got ATR=%v EMA=%v RSI=%v",
			ind.EnableATR, ind.EnableEMA, ind.EnableRSI)
	}
	// Open Interest is the Binance crypto-perp feed — absent (n/a) on MNQ since
	// W-NO-BINANCE A — off on futures so a new strategy doesn't list it.
	if ind.EnableOI {
		t.Errorf("futures default must disable Open Interest (crypto-perp feed, empty on MNQ); got EnableOI=%v", ind.EnableOI)
	}
	if ind.EnableMACD || ind.EnableBOLL {
		t.Errorf("futures default must NOT auto-enable MACD/BOLL; got MACD=%v BOLL=%v",
			ind.EnableMACD, ind.EnableBOLL)
	}
}
