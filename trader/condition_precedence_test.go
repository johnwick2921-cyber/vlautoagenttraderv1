package trader

import (
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// W1 (c) at the production seams that act on a condition's status: the arm
// seam (conditionShadowedFor → maybeManageArmedOrdersAt) and EntryGate's
// shadow leg (entryGateForArm). A name in BOTH env lists resolves LIVE; a
// strategy override still outranks either env list.

func TestEnvCollisionLetsTheConditionArm(t *testing.T) {
	t.Setenv("SHADOW_CONDITIONS", "fvg_entry")
	t.Setenv("LIVE_CONDITIONS", "fvg_entry")
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	oneSetupOff(&cfg)
	cfg.RiskControl.MinRiskRewardRatio = 2
	at, st := resetTrader(t, cfg)
	if at.conditionShadowedFor("fvg_entry", "NY") {
		t.Fatal("fvg_entry is in LIVE_CONDITIONS and SHADOW_CONDITIONS — it must resolve live")
	}
	shadowPlanAt(t, at, st, armedDoc())
	at.maybeManageArmedOrdersAt(nil, armTestClock(t, at))
	rows, err := st.ArmedOrders().ListNonTerminal(at.id)
	if err != nil || len(rows) != 1 || rows[0].State != "armed" {
		t.Fatalf("the env-live condition must arm normally, got %+v (err=%v)", rows, err)
	}
}

func TestEntryGateArmSeamAdmitsAConditionLiveOverShadow(t *testing.T) {
	t.Setenv("SHADOW_CONDITIONS", "breakout_retest")
	t.Setenv("LIVE_CONDITIONS", "breakout_retest")
	at := &AutoTrader{id: "pin-test"}
	plan := &kernel.ActivePlan{PlanID: "2026-09-02:NY:pin-test", Version: 5, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S3", Direction: "long", Condition: "breakout_retest"}
	leg := kernel.PlanArmLeg{Entry: 29192.50, Stop: 29115.00, Target: 29317.25}
	if reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0); refused && strings.Contains(reason, "SHADOW") {
		t.Fatalf("a condition in both env lists resolves LIVE; EntryGate must not refuse it as SHADOW: %q", reason)
	}
}

func TestStrategyShadowOutranksLiveConditionsAtEntryGate(t *testing.T) {
	t.Setenv("LIVE_CONDITIONS", "breakout_retest")
	cfg := &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true,
		ConditionStatus: map[string]string{"breakout_retest": "shadow"}}}
	at := &AutoTrader{id: "pin-test", config: AutoTraderConfig{StrategyConfig: cfg}}
	plan := &kernel.ActivePlan{PlanID: "2026-09-02:NY:pin-test", Version: 5, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S3", Direction: "long", Condition: "breakout_retest"}
	leg := kernel.PlanArmLeg{Entry: 29192.50, Stop: 29115.00, Target: 29317.25}
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0)
	if !refused || !strings.Contains(reason, "SHADOW") {
		t.Fatalf("the strategy's own shadow must outrank LIVE_CONDITIONS; got refused=%v %q", refused, reason)
	}
}
