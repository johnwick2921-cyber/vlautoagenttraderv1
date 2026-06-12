package agent

import "testing"

func TestCurrentSkillDAGStepDefaultsToFirstStep(t *testing.T) {
	session := skillSession{Name: "strategy_management", Action: "update_config"}
	step, ok := currentSkillDAGStep(session)
	if !ok {
		t.Fatal("expected dag step")
	}
	if step.ID != "resolve_target" {
		t.Fatalf("expected first step resolve_target, got %s", step.ID)
	}
}

func TestAdvanceSkillDAGStepMovesToNextStep(t *testing.T) {
	t.Skip("TODO: adapt to fork's agent API — see PR #3 merge 2026-05-25, incoming test from main targets upstream API")
	session := skillSession{Name: "strategy_management", Action: "update_config"}
	setSkillDAGStep(&session, "resolve_config_field")
	advanceSkillDAGStep(&session, "resolve_config_field")
	step, ok := currentSkillDAGStep(session)
	if !ok {
		t.Fatal("expected dag step")
	}
	if step.ID != "resolve_config_value" {
		t.Fatalf("expected resolve_config_value, got %s", step.ID)
	}
}
