package trader

import (
	"testing"

	"nofx/market"
	"nofx/store"
)

// W11 — the rendered indicator block + ai_config hash are FROZEN onto the plan row
// at write time (replay-safe: a later toggle change never rewrites history).
func TestW11IndicatorMirrorFrozenOnPlanRow(t *testing.T) {
	at := plannerTestTrader(t)
	frozenBlock := "### 1h\nEMA20: 15600.00\nRSI14: 58.20"
	frozenHash := "deadbeefcafef00d"

	ver, lc, err := at.runPlannerReadCore("NY", "2026-08-14", "deepseek-v4-pro", "hashZ",
		frozenBlock, frozenHash,
		func() (string, error) { return validTraderPlanJSON, nil })
	if err != nil || ver != 1 || lc != "active" {
		t.Fatalf("write: ver=%d lc=%q err=%v", ver, lc, err)
	}
	got, _ := at.store.Plan().GetLatestPlanForSession("2026-08-14", "NY")
	if got == nil {
		t.Fatal("no plan row written")
	}
	if got.IndicatorsBlock != frozenBlock {
		t.Fatalf("indicators_block not frozen verbatim\nwant %q\ngot  %q", frozenBlock, got.IndicatorsBlock)
	}
	if got.AIConfigHash != frozenHash {
		t.Fatalf("ai_config_hash not stored: want %q got %q", frozenHash, got.AIConfigHash)
	}
}

// W11 — renderIndicatorMirror is graceful when bars are unavailable (NT8 down at
// read time): empty block, but the ai_config hash is still computed (recorded).
func TestW11RenderIndicatorMirrorGraceful(t *testing.T) {
	prev := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prev }()
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return nil } // no bars

	at := &AutoTrader{
		id: "t1", exchange: "ninjatrader",
		config: AutoTraderConfig{
			NinjaTraderSymbol: "MNQ",
			StrategyConfig: &store.StrategyConfig{
				DayPlan: &store.DayPlanConfig{PlanEnabled: true},
				Indicators: store.IndicatorConfig{
					EnableEMA: true, EnableATR: true, EMAPeriods: []int{20, 50}, ATRPeriods: []int{14},
					Klines: store.KlineConfig{PrimaryTimeframe: "5m", PrimaryCount: 200, SelectedTimeframes: []string{"5m", "1h"}},
				},
			},
		},
	}
	block, hash := at.renderIndicatorMirror("MNQ")
	if block != "" {
		t.Fatalf("no bars → empty block, got:\n%s", block)
	}
	if hash == "" {
		t.Fatal("ai_config hash must be computed even when bars are unavailable")
	}
}
