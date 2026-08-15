package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

func mkPlanTrader(dp *store.DayPlanConfig) *AutoTrader {
	sc := store.StrategyConfig{
		Indicators: store.IndicatorConfig{Klines: store.KlineConfig{PrimaryTimeframe: "5m"}},
		DayPlan:    dp,
	}
	return &AutoTrader{
		id:       "t1",
		exchange: "ninjatrader",
		config:   AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &sc},
	}
}

// W9 — resolvers default to the shipped values (byte-identical behavior) and honor
// per-session overrides over strategy-level values.
func TestW9Resolvers(t *testing.T) {
	// nil day-plan → all defaults.
	at := mkPlanTrader(nil)
	if at.planModeFor("NY") != "advisory" {
		t.Fatal("default plan mode must be advisory")
	}
	if at.proximityFilterATR() != kernel.ActivationWindowK {
		t.Fatalf("default proximity must be %.1f", kernel.ActivationWindowK)
	}
	if at.scenarioCap() != 3 {
		t.Fatal("default scenario cap must be 3")
	}
	if at.replanCapFor("NY") != 2 {
		t.Fatal("default replan cap must be 2")
	}
	if !at.sessionEnabledForStrategy("NY") || at.sessionEnabledForStrategy("ASIA") {
		t.Fatal("default sessions_enabled must be [NY]")
	}
	if !at.eveningDigestEnabled() {
		t.Fatal("default evening digest must be on")
	}
	if at.approvalRequired() {
		t.Fatal("default approval_required must be off")
	}

	// explicit strategy-level values.
	dp := &store.DayPlanConfig{
		PlanEnabled: true, PlanMode: "strict", ProximityFilterATR: 2.0, ScenarioCap: 2,
		ReplanCap: 4, SessionsEnabled: []string{"NY", "LONDON"}, ApprovalRequired: true, EveningDigest: false,
	}
	at = mkPlanTrader(dp)
	if at.planModeFor("NY") != "strict" || at.proximityFilterATR() != 2.0 || at.scenarioCap() != 2 || at.replanCapFor("NY") != 4 {
		t.Fatal("strategy-level values must be read")
	}
	if !at.sessionEnabledForStrategy("LONDON") || at.sessionEnabledForStrategy("ASIA") {
		t.Fatal("sessions_enabled subset must be honored")
	}
	if at.eveningDigestEnabled() || !at.approvalRequired() {
		t.Fatal("evening_digest off + approval_required on must be read")
	}

	// per-session override wins.
	mode := "direction"
	cap := 1
	enable := false
	dp.Sessions = []store.DayPlanSessionOverride{{Session: "NY", PlanMode: &mode, ReplanCap: &cap, Enable: &enable}}
	if at.planModeFor("NY") != "direction" {
		t.Fatal("per-session plan_mode override must win")
	}
	if at.replanCapFor("NY") != 1 {
		t.Fatal("per-session replan_cap override must win")
	}
	if at.sessionEnabledForStrategy("NY") {
		t.Fatal("per-session Enable=false override must disable NY")
	}
	// bounds: out-of-range proximity/scenario fall back to defaults.
	dp.ProximityFilterATR = 9.0
	dp.ScenarioCap = 99
	if at.proximityFilterATR() != kernel.ActivationWindowK || at.scenarioCap() != 3 {
		t.Fatal("out-of-range proximity/scenario must fall back to defaults")
	}
}

// W9 — the plan-mode entry gate: advisory never blocks; direction blocks entries
// against the plan bias; strict blocks entries with no matched scenario cited.
func TestW9PlanModeBlocked(t *testing.T) {
	prev := kernel.ActivePlanProvider
	defer func() { kernel.ActivePlanProvider = prev }()

	longBiasPlan := func(string) *kernel.ActivePlan {
		return &kernel.ActivePlan{Doc: kernel.PlanDoc{
			Bias:      kernel.PlanBias{Direction: "long"},
			Scenarios: []kernel.PlanScenario{{ID: "S1", Direction: "long"}},
		}, Session: "NY", Version: 1}
	}

	// advisory → never blocks even against the bias.
	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"})
	kernel.ActivePlanProvider = longBiasPlan
	if _, blocked := at.planModeBlocked(&kernel.Decision{Action: "open_short"}); blocked {
		t.Fatal("advisory mode must never block")
	}

	// direction → block a short against a long bias, allow a long.
	at = mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "direction"})
	if _, blocked := at.planModeBlocked(&kernel.Decision{Action: "open_short"}); !blocked {
		t.Fatal("direction mode must block a short against a long bias")
	}
	if _, blocked := at.planModeBlocked(&kernel.Decision{Action: "open_long"}); blocked {
		t.Fatal("direction mode must allow a long with a long bias")
	}

	// strict → block an uncited entry, allow a matched-scenario citation.
	at = mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "strict"})
	if _, blocked := at.planModeBlocked(&kernel.Decision{Action: "open_long", CitedScenario: "off-plan"}); !blocked {
		t.Fatal("strict mode must block an off-plan entry")
	}
	if _, blocked := at.planModeBlocked(&kernel.Decision{Action: "open_long", CitedScenario: "S1"}); blocked {
		t.Fatal("strict mode must allow a matched-scenario entry")
	}

	// direction/strict with NO active plan → block (nothing authorized).
	kernel.ActivePlanProvider = func(string) *kernel.ActivePlan { return nil }
	at = mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "strict"})
	if _, blocked := at.planModeBlocked(&kernel.Decision{Action: "open_long", CitedScenario: "S1"}); !blocked {
		t.Fatal("strict mode with no active plan must block")
	}
}

// W9 — the approval gate: entries flow only after the owner grants the session-day.
func TestW9ApprovalGate(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, ApprovalRequired: true})
	at.store = st
	now := time.Now()

	if at.approvalGranted(now) {
		t.Fatal("no approval granted yet")
	}
	// owner approves this session-day.
	_ = st.SetSystemConfig(ApprovalKey("t1", kernel.CMESessionDayKey(now)), "granted")
	if !at.approvalGranted(now) {
		t.Fatal("approval must be honored after the grant")
	}
	// a DIFFERENT session-day is not covered by today's grant.
	if at.approvalGranted(now.Add(48 * time.Hour)) {
		t.Fatal("a grant must not carry to another session-day")
	}
}
