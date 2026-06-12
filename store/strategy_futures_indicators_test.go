package store

import "testing"

// TestApplyFuturesIndicatorDefaults verifies the futures new-strategy indicator
// defaults: the technical indicators ATR/EMA/RSI are enabled (the futures prompt
// leans on them — ATR sizes stops), the crypto-only NofxOS/ranking feeds are
// disabled, and MACD/BOLL are deliberately left off.
func TestApplyFuturesIndicatorDefaults(t *testing.T) {
	// Start from the crypto base defaults: technical indicators OFF, feeds ON.
	ind := IndicatorConfig{
		EnableEMA: false, EnableRSI: false, EnableATR: false,
		EnableMACD: false, EnableBOLL: false,
		EnableOI:        true,
		EnableQuantData: true, EnableQuantOI: true, EnableQuantNetflow: true,
		EnableOIRanking: true, EnableNetFlowRanking: true, EnablePriceRanking: true,
	}
	applyFuturesIndicatorDefaults(&ind)

	if !ind.EnableATR || !ind.EnableEMA || !ind.EnableRSI {
		t.Errorf("futures default must enable ATR/EMA/RSI; got ATR=%v EMA=%v RSI=%v",
			ind.EnableATR, ind.EnableEMA, ind.EnableRSI)
	}
	// Open Interest is the Binance crypto-perp feed (empty zeros on MNQ) — off on
	// futures so a new strategy doesn't list/value an empty OI section.
	if ind.EnableOI {
		t.Errorf("futures default must disable Open Interest (crypto-perp feed, empty on MNQ); got EnableOI=%v", ind.EnableOI)
	}
	if ind.EnableQuantData || ind.EnableQuantOI || ind.EnableQuantNetflow ||
		ind.EnableOIRanking || ind.EnableNetFlowRanking || ind.EnablePriceRanking {
		t.Errorf("futures default must disable NofxOS/ranking feeds; got %+v", ind)
	}
	if ind.EnableMACD || ind.EnableBOLL {
		t.Errorf("futures default must NOT auto-enable MACD/BOLL; got MACD=%v BOLL=%v",
			ind.EnableMACD, ind.EnableBOLL)
	}
}

// TestCryptoDefaultLeavesTechnicalIndicatorsOff confirms the crypto path stays
// byte-identical: in the default (non-futures) environment, GetDefaultStrategyConfig
// keeps EMA/RSI/ATR OFF (the futures helper is never applied) and the NofxOS
// Quant Data feed ON. Skipped if the env happens to be in futures mode.
func TestCryptoDefaultLeavesTechnicalIndicatorsOff(t *testing.T) {
	if isFuturesMode() {
		t.Skip("test environment is in futures mode; crypto-default assertion N/A")
	}
	cfg := GetDefaultStrategyConfig("en")
	if cfg.Indicators.EnableEMA || cfg.Indicators.EnableRSI || cfg.Indicators.EnableATR {
		t.Errorf("crypto default must keep EMA/RSI/ATR OFF; got EMA=%v RSI=%v ATR=%v",
			cfg.Indicators.EnableEMA, cfg.Indicators.EnableRSI, cfg.Indicators.EnableATR)
	}
	if !cfg.Indicators.EnableQuantData {
		t.Errorf("crypto default must keep NofxOS Quant Data ON; got %v", cfg.Indicators.EnableQuantData)
	}
}
