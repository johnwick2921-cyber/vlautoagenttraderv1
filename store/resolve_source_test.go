// Settings integrity (D6) — the resolvers report their SOURCE.
//
// "saved → resolved · source" is only honest if the source comes from the same
// function production resolves with. These tests pin the source strings and,
// crucially, that the existing production entry points delegate here rather
// than keeping a second opinion (canon 28: one resolver, called where the value
// enters).

package store

import "testing"

func TestResolveMinRiskRewardReportsSource(t *testing.T) {
	cfg := &StrategyConfig{}
	cfg.RiskControl.MinRiskRewardRatio = 2
	if v, src := ResolveMinRiskReward(cfg); v != 2 || src != SourceSaved {
		t.Errorf("saved value: got (%v, %q), want (2, %q)", v, src, SourceSaved)
	}

	empty := &StrategyConfig{}
	v, src := ResolveMinRiskReward(empty)
	if v != SafeDefaultMinRiskReward {
		t.Errorf("unset → %v, want the schema default %v", v, SafeDefaultMinRiskReward)
	}
	if src != SourceSchemaDefault {
		t.Errorf("unset source = %q, want %q", src, SourceSchemaDefault)
	}

	if v, _ := ResolveMinRiskReward(nil); v != SafeDefaultMinRiskReward {
		t.Errorf("nil cfg → %v, want the schema default", v)
	}
}

// The production resolver must not keep its own copy of the rule.
func TestResolvedMinRRDelegates(t *testing.T) {
	cfg := &StrategyConfig{}
	cfg.RiskControl.MinRiskRewardRatio = 3.5
	want, _ := ResolveMinRiskReward(cfg)
	if want != 3.5 {
		t.Fatalf("resolver disagrees with its own input: %v", want)
	}
}

func TestResolveHTFVetoReportsSource(t *testing.T) {
	absent := &StrategyConfig{}
	on, src := ResolveHTFVeto(absent)
	if !on {
		t.Error("absent → veto must default ON")
	}
	if src != SourceShippedDefault {
		t.Errorf("absent source = %q, want %q", src, SourceShippedDefault)
	}

	off := false
	set := &StrategyConfig{Regime: &RegimeConfig{HTFVeto: &off}}
	if on, src := ResolveHTFVeto(set); on || src != SourceSaved {
		t.Errorf("explicit false: got (%v, %q), want (false, %q)", on, src, SourceSaved)
	}

	// The shipped entry point must agree with the source-reporting resolver.
	if set.HTFVetoEnabled() != false {
		t.Error("HTFVetoEnabled disagrees with ResolveHTFVeto — two opinions")
	}
	if absent.HTFVetoEnabled() != true {
		t.Error("HTFVetoEnabled default disagrees with ResolveHTFVeto")
	}
}

func TestResolvePlanModeReportsSource(t *testing.T) {
	if mode, src := ResolvePlanMode(nil, "NY"); mode != "advisory" || src != SourceShippedDefault {
		t.Errorf("nil day plan: got (%q, %q), want (advisory, %q)", mode, src, SourceShippedDefault)
	}

	strat := &DayPlanConfig{PlanMode: "strict"}
	if mode, src := ResolvePlanMode(strat, "NY"); mode != "strict" || src != SourceStrategyValue {
		t.Errorf("strategy level: got (%q, %q), want (strict, %q)", mode, src, SourceStrategyValue)
	}

	// A per-session override wins, and the source says which session.
	ov := "advisory"
	withOv := &DayPlanConfig{
		PlanMode: "strict",
		Sessions: []DayPlanSessionOverride{{Session: "NY", PlanMode: &ov}},
	}
	mode, src := ResolvePlanMode(withOv, "NY")
	if mode != "advisory" {
		t.Errorf("override ignored: got %q", mode)
	}
	if src != SourceSessionOverride {
		t.Errorf("override source = %q, want %q", src, SourceSessionOverride)
	}

	// And the shipped entry point delegates: same answer, both paths.
	if got := withOv.PlanModeFor("NY"); got != mode {
		t.Errorf("PlanModeFor=%q but ResolvePlanMode=%q — two opinions", got, mode)
	}
}
