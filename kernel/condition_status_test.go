package kernel

import (
	"strings"
	"testing"

	"nofx/market"
)

// 0C shadow demotion (owner ruling 2026-08-31) — resolver + boot-ledger tests.
// 7.8: the boot line renders the RESOLVED map and changes when config changes.

func TestConditionStatusDefaults(t *testing.T) {
	if !IsConditionShadowed("fvg_entry", nil, nil, "") {
		t.Fatal("fvg_entry must default SHADOW (owner ruling)")
	}
	if !IsConditionShadowed("breakout_retest", nil, nil, "") {
		t.Fatal("breakout_retest must default SHADOW (owner ruling)")
	}
	for _, c := range []string{"reclaim", "hold", "sweep_reclaim", "reject", "acceptance", "breakdown_continue", "breakup_continue"} {
		if IsConditionShadowed(c, nil, nil, "") {
			t.Fatalf("%s must default LIVE", c)
		}
	}
}

func TestConditionStatusPrecedence(t *testing.T) {
	// base flips fvg_entry to live.
	base := map[string]string{"fvg_entry": "live"}
	if IsConditionShadowed("fvg_entry", base, nil, "") {
		t.Fatal("base=live must resolve live")
	}
	// session override wins over base.
	session := map[string]string{"fvg_entry": "shadow"}
	if !IsConditionShadowed("fvg_entry", base, session, "") {
		t.Fatal("session=shadow must beat base=live")
	}
	// env adds shadows; env beats defaults but not config.
	env := "breakdown_continue=shadow"
	if !IsConditionShadowed("breakdown_continue", nil, nil, env) {
		t.Fatal("env=shadow must shadow breakdown_continue")
	}
	if !IsConditionShadowed("fvg_entry", nil, nil, env) {
		t.Fatal("env must not unshadow the default")
	}
	// config beats env.
	base2 := map[string]string{"breakdown_continue": "live"}
	if IsConditionShadowed("breakdown_continue", base2, nil, env) {
		t.Fatal("base=live must beat env=shadow")
	}
}

func TestShadowConditionsEnvComposition(t *testing.T) {
	t.Setenv("SHADOW_CONDITIONS", "fade, acceptance")
	t.Setenv("LIVE_CONDITIONS", "fvg_entry")
	env := ShadowConditionsEnv()
	if !strings.Contains(env, "fade=shadow") || !strings.Contains(env, "acceptance=shadow") {
		t.Fatalf("SHADOW_CONDITIONS must compose, got %q", env)
	}
	if !strings.Contains(env, "fvg_entry=live") {
		t.Fatalf("LIVE_CONDITIONS must compose, got %q", env)
	}
}

func TestConditionStatusLedgerRendersResolvedMap(t *testing.T) {
	def := ConditionStatusLedger(nil, nil, "")
	if !strings.Contains(def, "shadow [breakout_retest, fvg_entry]") {
		t.Fatalf("default ledger must list both shadows, got %q", def)
	}
	flipped := ConditionStatusLedger(map[string]string{"fvg_entry": "live"}, nil, "")
	if strings.Contains(flipped, "shadow [breakout_retest, fvg_entry]") {
		t.Fatalf("ledger must render the RESOLVED map (config flip visible), got %q", flipped)
	}
	if !strings.Contains(flipped, "fvg_entry") || !strings.Contains(flipped, "live [") {
		t.Fatalf("flipped ledger malformed: %q", flipped)
	}
}

// 7.5 — the complete counterfactual: a replay bar that contains BOTH stop and
// target must be flagged AMBIGUOUS and resolved worst-case (stop).
func TestShadowABAmbiguousBarWorstCase(t *testing.T) {
	sc := PlanScenario{
		ID: "S1", Condition: "fvg_entry", Direction: "long",
		Confirm: &PlanConfirm{Rule: "touch", RefPrice: 100, Side: "above"},
		Arm:     &PlanArmSpec{Enabled: true, Entry: 100, Stop: 99.0, Target: 101.0},
	}
	// One bar whose range straddles BOTH stop and target: high 101.5, low 98.5.
	bars := []market.Kline{
		{OpenTime: 1000, CloseTime: 1059_999, Open: 100, High: 101.5, Low: 98.5, Close: 99.5},
		{OpenTime: 1060_000, CloseTime: 1119_999, Open: 99.5, High: 99.6, Low: 99.4, Close: 99.5},
	}
	rows := ShadowABForScenario(sc, bars, "MNQ", 999, 2000_000)
	if len(rows) == 0 {
		t.Fatal("touch must fill on the straddling bar")
	}
	r := rows[0]
	if r.FillPx != 100 {
		t.Fatalf("fill px want 100, got %.2f", r.FillPx)
	}
	if !r.Ambiguous {
		t.Fatal("a bar containing BOTH stop and target must be flagged AMBIGUOUS")
	}
	if r.Outcome != "stop" {
		t.Fatalf("ambiguous must resolve WORST CASE (stop), got %q", r.Outcome)
	}
	if r.NetPnL >= 0 {
		t.Fatalf("worst-case stop must be a net loss, got %.2f", r.NetPnL)
	}
	if r.StopPx != 99.0 || r.TargetPx != 101.0 || r.RR <= 0 {
		t.Fatalf("complete authored fields missing: %+v", r)
	}
	if r.MFER < 0 || r.MAER < 0 || r.MAER <= 0 {
		t.Fatalf("R-multiple fields must be populated: %+v", r)
	}
}
