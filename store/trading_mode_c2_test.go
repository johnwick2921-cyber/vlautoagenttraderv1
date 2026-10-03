package store

import (
	"bytes"
	"strings"
	"testing"

	"vl/config"
	"vl/logger"
)

// TestTradingModeNonFuturesWarnsOnceAndStaysFutures (plan C2): a non-futures
// TRADING_MODE value is IGNORED with exactly ONE boot warning, never a refusal,
// and the process stays futures-only — the fresh-strategy template still seeds
// the NT8 instrument with the futures indicator defaults. Drives the production
// call sites (config.Init + GetDefaultStrategyConfig), not rebuilt inputs.
func TestTradingModeNonFuturesWarnsOnceAndStaysFutures(t *testing.T) {
	var buf bytes.Buffer
	prev := logger.Log.Out
	logger.Log.SetOutput(&buf)
	defer logger.Log.SetOutput(prev)

	t.Setenv("TRADING_MODE", "crypto")
	config.Init()
	config.Init() // the warning must fire exactly ONCE, not once per Init

	if got := strings.Count(buf.String(), "TRADING_MODE=crypto ignored — futures-only build"); got != 1 {
		t.Fatalf("want exactly 1 boot warning, got %d\n%s", got, buf.String())
	}

	cfg := GetDefaultStrategyConfig("en")
	if cfg.CoinSource.SourceType != "static" ||
		len(cfg.CoinSource.StaticCoins) != 1 || cfg.CoinSource.StaticCoins[0] != "MNQ" {
		t.Fatalf("default strategy must seed the NT8 instrument (futures-only); got %+v", cfg.CoinSource)
	}
	if !cfg.Indicators.EnableEMA || !cfg.Indicators.EnableRSI || !cfg.Indicators.EnableATR {
		t.Fatalf("futures indicator defaults must stay ON; got EMA=%v RSI=%v ATR=%v",
			cfg.Indicators.EnableEMA, cfg.Indicators.EnableRSI, cfg.Indicators.EnableATR)
	}
}
