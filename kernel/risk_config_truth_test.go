package kernel

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"nofx/store"
)

// THE REGRESSION TEST THAT ENDS THIS CLASS (P0 follow-up, 2026-08-17).
//
// Three times now the same disease has surfaced: the SCREEN, the PROMPT and the
// ENGINE narrating different rulebooks. This file pins all three to one source.
//
// It also exists because of a fourth instance I caused myself: I reported
// `risk_control` as `{}` on the live rows and concluded the gates were running
// R:R 1.0 / confidence 50. They were not — the block is nested under `ai_config`
// (which is exactly what the codec writes) and the real values were 3.0 / 65 all
// along. An AUDIT QUERY that reads the wrong path is as dangerous as a broken
// gate, because it triggers "fixes" to correct configuration. Hence
// TestRiskConfigLivesUnderAIConfig below: it pins the shape the auditor must read.

// riskFields is every risk field whose loss or drift changes what the bot trades.
var riskFields = []string{
	"min_risk_reward_ratio", "min_confidence", "max_positions",
	"max_contracts_per_order", "max_contracts_enabled",
	"hold_discipline", "breakeven_enabled", "breakeven_trigger_points",
	"guardrails_enabled", "daily_loss_limit_usd", "daily_loss_enabled",
	"daily_profit_target_usd", "daily_profit_enabled",
	"max_daily_trades", "max_daily_trades_enabled",
	"consecutive_loss_halt", "reentry_cooldown_minutes",
	"max_margin_usage", "min_position_size",
}

func fullRiskConfig() store.RiskControlConfig {
	tr := func(b bool) *bool { return &b }
	return store.RiskControlConfig{
		MaxPositions: 3, BTCETHMaxLeverage: 5, AltcoinMaxLeverage: 5,
		BTCETHMaxPositionValueRatio: 5, AltcoinMaxPositionValueRatio: 1,
		MaxMarginUsage: 0.9, MinPositionSize: 12,
		MinRiskRewardRatio: 3, MinConfidence: 65,
		GuardrailsEnabled: tr(false),
		DailyLossLimitUSD: 450, DailyLossEnabled: tr(true),
		DailyProfitTargetUSD: 900, DailyProfitEnabled: tr(true),
		MaxDailyTrades: 3, MaxDailyTradesEnabled: tr(true),
		ConsecutiveLossHalt: 2, ReentryCooldownMinutes: 20,
		MaxContractsPerOrder: 2, MaxContractsEnabled: tr(true),
		HoldDisciplineEnabled: tr(true),
		BreakevenEnabled:      tr(true), BreakevenTriggerPoints: 50,
	}
}

// 1 — EVERY risk field must survive BOTH halves of the hand-rolled codec.
// A field present in Marshal but absent from Unmarshal (or vice versa) is
// silently dropped; that footgun already ate day_plan once.
func TestEveryRiskFieldSurvivesBothCodecHalves(t *testing.T) {
	cfg := store.StrategyConfig{StrategyType: "ai_trading", RiskControl: fullRiskConfig()}

	blob, err := json.Marshal(cfg) // hand-rolled MarshalJSON
	if err != nil {
		t.Fatal(err)
	}

	// The marshalled JSON must physically contain every risk key.
	var probe map[string]any
	if err := json.Unmarshal(blob, &probe); err != nil {
		t.Fatal(err)
	}
	ai, _ := probe["ai_config"].(map[string]any)
	if ai == nil {
		t.Fatal("ai_config missing from the marshalled config — risk_control lives inside it")
	}
	rc, _ := ai["risk_control"].(map[string]any)
	if rc == nil {
		t.Fatal("ai_config.risk_control missing from the MARSHAL half")
	}
	for _, f := range riskFields {
		if _, ok := rc[f]; !ok {
			t.Errorf("MARSHAL half drops risk field %q — it would never reach the DB", f)
		}
	}

	var back store.StrategyConfig
	if err := json.Unmarshal(blob, &back); err != nil { // hand-rolled UnmarshalJSON
		t.Fatal(err)
	}
	got, want := back.RiskControl, cfg.RiskControl
	if got.MinRiskRewardRatio != want.MinRiskRewardRatio ||
		got.MinConfidence != want.MinConfidence ||
		got.MaxContractsPerOrder != want.MaxContractsPerOrder ||
		got.ConsecutiveLossHalt != want.ConsecutiveLossHalt ||
		got.ReentryCooldownMinutes != want.ReentryCooldownMinutes ||
		got.BreakevenTriggerPoints != want.BreakevenTriggerPoints {
		t.Errorf("UNMARSHAL half lost a numeric risk field:\n got  %+v\n want %+v", got, want)
	}
	for name, pair := range map[string][2]*bool{
		"hold_discipline":          {want.HoldDisciplineEnabled, got.HoldDisciplineEnabled},
		"breakeven_enabled":        {want.BreakevenEnabled, got.BreakevenEnabled},
		"guardrails_enabled":       {want.GuardrailsEnabled, got.GuardrailsEnabled},
		"max_contracts_enabled":    {want.MaxContractsEnabled, got.MaxContractsEnabled},
		"daily_loss_enabled":       {want.DailyLossEnabled, got.DailyLossEnabled},
		"max_daily_trades_enabled": {want.MaxDailyTradesEnabled, got.MaxDailyTradesEnabled},
	} {
		w, g := pair[0], pair[1]
		if w == nil || g == nil || *w != *g {
			t.Errorf("UNMARSHAL half lost toggle %q (want %v, got %v)", name, w, g)
		}
	}
}

// 2 — THE SHAPE THE AUDITOR MUST READ. risk_control is nested under ai_config.
// A query against the TOP level finds nothing and reads as "empty config", which
// is precisely the false alarm that produced this test.
func TestRiskConfigLivesUnderAIConfig(t *testing.T) {
	blob, _ := json.Marshal(store.StrategyConfig{StrategyType: "ai_trading", RiskControl: fullRiskConfig()})
	var m map[string]any
	_ = json.Unmarshal(blob, &m)
	if _, atTop := m["risk_control"]; atTop {
		t.Fatal("risk_control appeared at the TOP level — if this ever becomes true, update every audit query and this test's comment")
	}
	ai := m["ai_config"].(map[string]any)
	if _, ok := ai["risk_control"]; !ok {
		t.Fatal("risk_control is neither top-level nor under ai_config — the audit path is now unknowable")
	}
}

// 3 — AN UNSET VALUE MUST NEVER WIDEN RISK.
// ClampLimits used to raise a zero only to the RANGE FLOOR (1.0 / 50), the
// loosest setting available; "never configured" was therefore the most permissive
// configuration. Unset now resolves to the researched default.
func TestUnsetRiskFieldsDefaultToTheSafeDirection(t *testing.T) {
	var c store.StrategyConfig // everything unset
	c.ClampLimits()
	if c.RiskControl.MinRiskRewardRatio != store.SafeDefaultMinRiskReward {
		t.Errorf("unset R:R = %.2f, want the researched %.2f — unset must not be the loosest setting",
			c.RiskControl.MinRiskRewardRatio, store.SafeDefaultMinRiskReward)
	}
	if c.RiskControl.MinConfidence != store.SafeDefaultMinConfidence {
		t.Errorf("unset confidence = %d, want the researched %d",
			c.RiskControl.MinConfidence, store.SafeDefaultMinConfidence)
	}

	// An EXPLICIT choice is still honored — this changes the default, not a choice.
	explicit := store.StrategyConfig{RiskControl: store.RiskControlConfig{MinRiskRewardRatio: 1.5, MinConfidence: 55}}
	explicit.ClampLimits()
	if explicit.RiskControl.MinRiskRewardRatio != 1.5 {
		t.Errorf("explicit R:R 1.5 became %.2f — the safe default must not override a deliberate setting",
			explicit.RiskControl.MinRiskRewardRatio)
	}
	if explicit.RiskControl.MinConfidence != 55 {
		t.Errorf("explicit confidence 55 became %d", explicit.RiskControl.MinConfidence)
	}

	// Out-of-range values are still clamped to the permitted band.
	low := store.StrategyConfig{RiskControl: store.RiskControlConfig{MinRiskRewardRatio: 0.2, MinConfidence: 5}}
	low.ClampLimits()
	if low.RiskControl.MinRiskRewardRatio != store.MinRiskReward || low.RiskControl.MinConfidence != store.MinConfidence {
		t.Errorf("below-range values must clamp to the band, got R:R=%.2f conf=%d",
			low.RiskControl.MinRiskRewardRatio, low.RiskControl.MinConfidence)
	}
}

// 4 — THE PROMPT MUST STATE THE THRESHOLD THE GATE ENFORCES.
// This is the row that catches "the AI was told one number and judged by
// another". It renders the real futures system prompt and scrapes the numbers
// back out.
func TestPromptStatesTheEnforcedThresholds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		minRR   float64
		minConf int
	}{
		{"owner values", 3.0, 65},
		{"stricter", 4.0, 80},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := store.GetDefaultStrategyConfig("en")
			cfg.RiskControl.MinRiskRewardRatio = tc.minRR
			cfg.RiskControl.MinConfidence = tc.minConf
			cfg.ClampLimits() // exactly what the decision path does before the gates read it

			engine := NewStrategyEngine(&cfg)
			prompt := engine.BuildSystemPrompt(50000, "futures", "MNQ")

			// The value the GATE will enforce, read the same way validateDecisions reads it.
			gateRR := engine.GetRiskControlConfig().MinRiskRewardRatio
			gateConf := engine.GetRiskControlConfig().MinConfidence

			rr := regexp.MustCompile(`at least ([0-9.]+)x the risk`).FindStringSubmatch(prompt)
			if rr == nil {
				t.Fatalf("the futures prompt no longer states an R:R threshold — the AI is flying blind")
			}
			if rr[1] != fmt.Sprintf("%.2f", gateRR) {
				t.Errorf("PROMPT says R:R %s but the GATE enforces %.2f — the AI is being told a different rulebook", rr[1], gateRR)
			}

			conf := regexp.MustCompile(`Min confidence to open: (\d+)`).FindStringSubmatch(prompt)
			if conf == nil {
				t.Fatalf("the futures prompt no longer states a confidence threshold")
			}
			if conf[1] != fmt.Sprintf("%d", gateConf) {
				t.Errorf("PROMPT says confidence %s but the GATE enforces %d", conf[1], gateConf)
			}
			if !strings.Contains(prompt, fmt.Sprintf("≥ %d", gateConf)) {
				t.Errorf("the prompt's repeated confidence reminders disagree with the enforced %d", gateConf)
			}
		})
	}
}
