// Config-truth 4-step for min_risk_reward_ratio (audit follow-up). Proves the value
// flows SAVED → ROW (JSON round-trip through the ai_config nesting) → READ (into the
// validator) → ENFORCED against the REAL risk:reward. The audit's prior BROKEN
// verdict — the synthetic entry pinned every R:R at 4.0 so the gate never bound —
// is formally flipped here: min_risk_reward_ratio now gates a real sub-threshold
// trade and admits a compliant one.
package kernel

import (
	"encoding/json"
	"testing"

	"nofx/market"
	"nofx/store"
)

func TestConfigTruth_MinRR_SavedRowReadEnforced(t *testing.T) {
	// 1. SAVED — a strategy config carries min_risk_reward_ratio (under ai_config).
	cfg := store.StrategyConfig{StrategyType: "ai_trading"}
	cfg.RiskControl.MinRiskRewardRatio = 3.5

	// 2. ROW — marshal + reload exactly as the DB persists it, and confirm the value
	// survives the ai_config nesting round-trip.
	blob, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var row store.StrategyConfig
	if err := json.Unmarshal(blob, &row); err != nil {
		t.Fatal(err)
	}
	if row.RiskControl.MinRiskRewardRatio != 3.5 {
		t.Fatalf("ROW read-back min_rr = %v, want 3.5 — value lost through persistence", row.RiskControl.MinRiskRewardRatio)
	}

	ctx := &Context{MarketDataMap: map[string]*market.Data{"MNQ": {Symbol: "MNQ", CurrentPrice: 100}}}
	mk := func(sl, tp float64) *Decision {
		return &Decision{Symbol: "MNQ", Action: "open_long", Leverage: 1, PositionSizeUSD: 60000, StopLoss: sl, TakeProfit: tp, Confidence: 90}
	}

	// 3+4. READ + ENFORCED — the row's min_rr gates the REAL R:R.
	// entry 100, SL 90, TP 130 → risk 10, reward 30, real R:R 3.0 < 3.5 → REJECT.
	if err := validateDecision(mk(90, 130), 50000, 10, 5, 5, 1, row.RiskControl.MinRiskRewardRatio, 0, 20, ctx); err == nil {
		t.Fatal("BROKEN-verdict regression: real R:R 3.0 must be REJECTED at the configured min 3.5")
	}
	// entry 100, SL 90, TP 140 → risk 10, reward 40, real R:R 4.0 ≥ 3.5 → PASS.
	if err := validateDecision(mk(90, 140), 50000, 10, 5, 5, 1, row.RiskControl.MinRiskRewardRatio, 0, 20, ctx); err != nil {
		t.Fatalf("real R:R 4.0 must PASS at the configured min 3.5 (proves the gate binds, not blocks-all), got: %v", err)
	}
}
