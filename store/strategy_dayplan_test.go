package store

import (
	"encoding/json"
	"reflect"
	"testing"
)

// P0.1 — day_plan config on strategy rows. RECON #1: StrategyConfig has
// HAND-ROLLED Marshal/Unmarshal; the field must live at ROOT (a grid switch
// nukes ai_config) and be added to BOTH codecs or it silently drops. These
// goldens are written FIRST and lock the wire schema + the additive guarantee.

// fullDayPlan is a fully-populated block used across the round-trip goldens.
func fullDayPlan() *DayPlanConfig {
	return &DayPlanConfig{
		PlanEnabled:        true,
		PlannerModel:       "deepseek-reasoner",
		PlanMode:           "advisory",
		PlannerTimeframes:  []string{"D", "4h", "1h", "15m"},
		ProximityFilterATR: 1.5,
		MaxLevels:          8,
		ScenarioCap:        3,
		AcceptanceRule:     "2x5m",
		ReplanCap:          2,
		SessionsEnabled:    []string{"NY"},
		ApprovalRequired:   false,
		EveningDigest:      true,
		Sessions: []DayPlanSessionOverride{
			{Session: "NY"}, // all-inherit override (every pointer nil)
		},
		// W6 (2026-08-25) — wake knobs participate in the wire golden.
		WakeOn15mZone:            wakeBoolPtr(true),
		WakeOnHTFZone:            wakeBoolPtr(true),
		WakeOnHTFOB:              true,
		WakeOnSeatedInvalidation: wakeBoolPtr(true),
		WakeOnIFVG:               wakeBoolPtr(true),
		WakeMinIntervalMin:       10,
	}
}

// TestDayPlanConfigGolden locks the exact wire bytes (field names + order) so a
// future refactor that renames/reorders a field is caught. DayPlanConfig is a
// plain struct (no custom codec) → json.Marshal is deterministic.
func TestDayPlanConfigGolden(t *testing.T) {
	const golden = `{"plan_enabled":true,"planner_model":"deepseek-reasoner","plan_mode":"advisory","planner_timeframes":["D","4h","1h","15m"],"proximity_filter_atr":1.5,"max_levels":8,"scenario_cap":3,"acceptance_rule":"2x5m","replan_cap":2,"sessions_enabled":["NY"],"approval_required":false,"evening_digest":true,"sessions":[{"session":"NY"}],"wake_on_15m_zone":true,"wake_on_htf_zone":true,"wake_on_htf_ob":true,"wake_on_seated_invalidation":true,"wake_on_ifvg":true,"wake_min_interval_min":10}`
	got, err := json.Marshal(fullDayPlan())
	if err != nil {
		t.Fatalf("marshal day plan: %v", err)
	}
	if string(got) != golden {
		t.Fatalf("day_plan golden mismatch:\n got: %s\nwant: %s", string(got), golden)
	}
}

// TestDayPlanWakeKnobDefaults locks the W6 knob resolution seam: nil config or
// unset pointer → ON (except HTF OBs, OFF); explicit false disables; interval
// 0 → 10.
func TestDayPlanWakeKnobDefaults(t *testing.T) {
	var nilCfg *DayPlanConfig
	if !nilCfg.WakeOn15mZoneEnabled() || !nilCfg.WakeOnHTFZoneEnabled() ||
		!nilCfg.WakeOnSeatedInvalidationEnabled() || !nilCfg.WakeOnIFVGEnabled() {
		t.Fatalf("nil config must resolve every ON-default knob to ON")
	}
	if nilCfg.WakeOnHTFOBEnabled() {
		t.Fatalf("nil config must resolve HTF-OB knob to OFF")
	}
	if nilCfg.WakeMinIntervalMinutes() != 30 {
		t.Fatalf("nil config interval = %d want 30", nilCfg.WakeMinIntervalMinutes())
	}

	empty := &DayPlanConfig{}
	if !empty.WakeOn15mZoneEnabled() || !empty.WakeOnHTFZoneEnabled() {
		t.Fatalf("unset pointers must resolve to ON (defaults)")
	}

	off := false
	c := &DayPlanConfig{
		WakeOn15mZone:            &off,
		WakeOnHTFZone:            &off,
		WakeOnHTFOB:              false,
		WakeOnSeatedInvalidation: &off,
		WakeOnIFVG:               &off,
		WakeMinIntervalMin:       25,
	}
	if c.WakeOn15mZoneEnabled() || c.WakeOnHTFZoneEnabled() ||
		c.WakeOnSeatedInvalidationEnabled() || c.WakeOnIFVGEnabled() ||
		c.WakeOnHTFOBEnabled() {
		t.Fatalf("explicit false must disable")
	}
	if c.WakeMinIntervalMinutes() != 25 {
		t.Fatalf("interval = %d want 25", c.WakeMinIntervalMinutes())
	}

	if d := DefaultDayPlanConfig(); !d.WakeOn15mZoneEnabled() || !d.WakeOnHTFZoneEnabled() ||
		!d.WakeOnSeatedInvalidationEnabled() || !d.WakeOnIFVGEnabled() ||
		d.WakeOnHTFOBEnabled() || d.WakeMinIntervalMinutes() != 30 {
		t.Fatalf("DefaultDayPlanConfig must seed ON/OFF/30 defaults")
	}
}

// TestDayPlanRoundTripThroughStrategyConfig proves the field survives the
// hand-rolled StrategyConfig codec: marshal the whole config, confirm day_plan
// sits at ROOT (not inside ai_config), unmarshal, and deep-equal the block.
func TestDayPlanRoundTripThroughStrategyConfig(t *testing.T) {
	cfg := GetDefaultStrategyConfig("en")
	cfg.DayPlan = fullDayPlan()

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, ok := asMap["day_plan"]; !ok {
		t.Fatalf("day_plan must be at ROOT of the config JSON: %s", string(raw))
	}
	if ai, ok := asMap["ai_config"].(map[string]any); ok {
		if _, nested := ai["day_plan"]; nested {
			t.Fatalf("day_plan must NOT nest inside ai_config (grid switch would drop it)")
		}
	}

	var back StrategyConfig
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal back: %v", err)
	}
	if back.DayPlan == nil {
		t.Fatalf("day_plan dropped on unmarshal (missing from rawStrategyConfig codec?)")
	}
	if !reflect.DeepEqual(back.DayPlan, cfg.DayPlan) {
		t.Fatalf("day_plan round-trip mismatch:\n got: %+v\nwant: %+v", back.DayPlan, cfg.DayPlan)
	}
}

// TestDayPlanAbsentIsByteIdentical is the additive guarantee: a config with no
// day_plan must NOT emit the key, so every existing strategy stays byte-identical.
func TestDayPlanAbsentIsByteIdentical(t *testing.T) {
	cfg := GetDefaultStrategyConfig("en") // DayPlan nil
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := asMap["day_plan"]; ok {
		t.Fatalf("absent day_plan must not appear in JSON: %s", string(raw))
	}
}

// TestDayPlanSurvivesGridSwitch is RECON #1's core hazard: switching to
// grid_trading drops ai_config, but day_plan (root) must persist.
func TestDayPlanSurvivesGridSwitch(t *testing.T) {
	cfg := GetDefaultStrategyConfig("en")
	cfg.DayPlan = fullDayPlan()
	cfg.StrategyType = "grid_trading"
	cfg.GridConfig = &GridStrategyConfig{Symbol: "BTCUSDT", GridCount: 20, Distribution: "uniform"}

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal grid config: %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := asMap["ai_config"]; ok {
		t.Fatalf("grid config must not carry ai_config: %s", string(raw))
	}
	if _, ok := asMap["day_plan"]; !ok {
		t.Fatalf("day_plan must survive a grid switch (root placement): %s", string(raw))
	}

	var back StrategyConfig
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal back: %v", err)
	}
	if back.DayPlan == nil || !back.DayPlan.PlanEnabled {
		t.Fatalf("day_plan lost across grid round-trip: %+v", back.DayPlan)
	}
}

// TestDayPlanSessionOverrideInheritSemantics: nil pointers mean "inherit"; set
// pointers mean "override". Both must round-trip exactly.
func TestDayPlanSessionOverrideInheritSemantics(t *testing.T) {
	enable := true
	mode := "strict"
	grade := "A"
	dp := &DayPlanConfig{
		PlanEnabled: true,
		Sessions: []DayPlanSessionOverride{
			{Session: "ASIA", Enable: &enable, MinGrade: &grade}, // partial override
			{Session: "NY", PlanMode: &mode},                     // different partial
			{Session: "LONDON"},                                  // all-inherit
		},
	}
	cfg := GetDefaultStrategyConfig("en")
	cfg.DayPlan = dp

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back StrategyConfig
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(back.DayPlan, dp) {
		t.Fatalf("session override inherit/override semantics not preserved:\n got: %+v\nwant: %+v", back.DayPlan, dp)
	}
	// LONDON row: every override pointer must remain nil (pure inherit).
	london := back.DayPlan.Sessions[2]
	if london.Enable != nil || london.ReplanCap != nil || london.PlanMode != nil ||
		london.AcceptanceRule != nil || london.MinGrade != nil || london.MaxTrades != nil {
		t.Fatalf("LONDON all-inherit row gained a non-nil override: %+v", london)
	}
}

// TestDayPlanSurvivesMergeStrategyConfig is the config-truth persistence proof:
// handleUpdateStrategy runs every edit through MergeStrategyConfig. An unrelated
// edit must NOT wipe day_plan (the RECON #1 hazard class), and a partial day_plan
// patch must deep-merge (keep sibling fields).
func TestDayPlanSurvivesMergeStrategyConfig(t *testing.T) {
	base := GetDefaultStrategyConfig("en")
	base.DayPlan = fullDayPlan()

	// (1) An unrelated edit preserves the whole day_plan block.
	merged, err := MergeStrategyConfig(base, map[string]any{"prompt_variant": "aggressive"})
	if err != nil {
		t.Fatalf("merge unrelated: %v", err)
	}
	if merged.PromptVariant != "aggressive" {
		t.Fatalf("unrelated field not applied: %q", merged.PromptVariant)
	}
	if merged.DayPlan == nil || !reflect.DeepEqual(merged.DayPlan, fullDayPlan()) {
		t.Fatalf("unrelated edit wiped/changed day_plan: %+v", merged.DayPlan)
	}

	// (2) A partial day_plan patch deep-merges: toggles plan_enabled, keeps the
	// rest (max_levels, sessions_enabled, etc.).
	merged2, err := MergeStrategyConfig(base, map[string]any{
		"day_plan": map[string]any{"plan_enabled": false},
	})
	if err != nil {
		t.Fatalf("merge day_plan patch: %v", err)
	}
	if merged2.DayPlan == nil || merged2.DayPlan.PlanEnabled {
		t.Fatalf("plan_enabled not toggled off: %+v", merged2.DayPlan)
	}
	if merged2.DayPlan.MaxLevels != 8 || len(merged2.DayPlan.SessionsEnabled) != 1 {
		t.Fatalf("partial patch dropped sibling day_plan fields: %+v", merged2.DayPlan)
	}
}
