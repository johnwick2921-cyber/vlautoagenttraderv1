// F11b — a strategy-type switch must not silently destroy the ai_config bundle.
package store

import "testing"

func aiBase() StrategyConfig {
	return StrategyConfig{
		StrategyType: "ai_trading",
		RiskControl:  RiskControlConfig{MinRiskRewardRatio: 3.5, MinConfidence: 70},
		Indicators:   IndicatorConfig{EnableEMA: true, EMAPeriods: []int{9, 21, 200}},
		CustomPrompt: "MY DESK",
	}
}

// TestPreserveAIConfig_SwitchWithoutFlagKeepsIt: switching type without the confirm
// flag preserves the existing ai_config (the acceptance case).
func TestPreserveAIConfig_SwitchWithoutFlagKeepsIt(t *testing.T) {
	base := aiBase()
	// The merge for a grid switch drops the AI bundle (empty) + sets grid type.
	merged := StrategyConfig{StrategyType: "grid_trading"}

	got := PreserveAIConfigOnTypeSwitch(base, merged, false)

	if got.StrategyType != "grid_trading" {
		t.Errorf("strategy_type must still switch to grid, got %q", got.StrategyType)
	}
	if got.RiskControl.MinRiskRewardRatio != 3.5 || got.RiskControl.MinConfidence != 70 {
		t.Errorf("risk_control must be preserved, got %+v", got.RiskControl)
	}
	if !got.Indicators.EnableEMA || len(got.Indicators.EMAPeriods) != 3 {
		t.Errorf("indicators must be preserved, got %+v", got.Indicators)
	}
	if got.CustomPrompt != "MY DESK" {
		t.Errorf("custom_prompt must be preserved, got %q", got.CustomPrompt)
	}
}

// TestPreserveAIConfig_ConfirmedAllowsLoss: with the confirm flag, the merged
// (cleared) config stands — the user opted in.
func TestPreserveAIConfig_ConfirmedAllowsLoss(t *testing.T) {
	base := aiBase()
	merged := StrategyConfig{StrategyType: "grid_trading"}

	got := PreserveAIConfigOnTypeSwitch(base, merged, true)

	if got.RiskControl.MinRiskRewardRatio != 0 || got.Indicators.EnableEMA || got.CustomPrompt != "" {
		t.Errorf("confirmed switch must NOT preserve the AI bundle, got %+v / %+v / %q",
			got.RiskControl, got.Indicators, got.CustomPrompt)
	}
}

// TestPreserveAIConfig_RoundTripKeepsIt: ai→grid→ai (both unconfirmed) returns the
// original ai_config — the destroyer scenario the audit flagged.
func TestPreserveAIConfig_RoundTripKeepsIt(t *testing.T) {
	base := aiBase()

	toGrid := PreserveAIConfigOnTypeSwitch(base, StrategyConfig{StrategyType: "grid_trading"}, false)
	// Now switch back; the "merged" from the FE would be ai_trading with an empty bundle.
	backToAI := PreserveAIConfigOnTypeSwitch(toGrid, StrategyConfig{StrategyType: "ai_trading"}, false)

	if backToAI.RiskControl.MinRiskRewardRatio != 3.5 || backToAI.CustomPrompt != "MY DESK" {
		t.Errorf("round-trip ai→grid→ai lost the config: %+v / %q", backToAI.RiskControl, backToAI.CustomPrompt)
	}
}

// TestPreserveAIConfig_SameTypeIsNoop: a same-type save is not a switch — merged
// stands untouched.
func TestPreserveAIConfig_SameTypeIsNoop(t *testing.T) {
	base := aiBase()
	merged := aiBase()
	merged.RiskControl.MinRiskRewardRatio = 5.0 // a legit same-type edit
	got := PreserveAIConfigOnTypeSwitch(base, merged, false)
	if got.RiskControl.MinRiskRewardRatio != 5.0 {
		t.Errorf("same-type edit must stand, got %v", got.RiskControl.MinRiskRewardRatio)
	}
}
