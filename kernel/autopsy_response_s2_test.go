package kernel

import (
	"strings"
	"testing"
)

// S2 (autopsy-response wave) — sweep_reclaim becomes armable ONLY as a chained
// arm (wait_confirm + confirm{}), and fantasy targets (planned R > 6) WARN.
func TestS2SweepReclaimChainedArmValidation(t *testing.T) {
	mk := func(cond string, wait bool, conf bool) PlanScenario {
		sc := PlanScenario{ID: "S1", Condition: cond, Direction: "short",
			Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 105, Target: 90, WaitConfirm: wait}}
		if conf {
			sc.Confirm = &PlanConfirm{Rule: "2x5m_close", RefPrice: 110, Side: "below"}
		}
		return sc
	}
	if err := ArmSpecValid(mk("sweep_reclaim", false, true)); err == nil {
		t.Fatal("sweep_reclaim arm WITHOUT wait_confirm must be rejected")
	}
	if err := ArmSpecValid(mk("sweep_reclaim", true, false)); err == nil {
		t.Fatal("sweep_reclaim chained arm without a confirm{} must be rejected")
	}
	if err := ArmSpecValid(mk("sweep_reclaim", true, true)); err != nil {
		t.Fatalf("sweep_reclaim chained arm (wait_confirm + confirm) must pass, got %v", err)
	}
	if err := ArmSpecValid(mk("acceptance", true, true)); err == nil {
		t.Fatal("acceptance arm must stay non-armable")
	}
}

func TestS2FantasyTargetWarnings(t *testing.T) {
	doc := PlanDoc{Scenarios: []PlanScenario{
		{ID: "S1", Direction: "long", Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110}},  // R 2.0 → clean
		{ID: "S2", Direction: "long", Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 98, Target: 120}},  // R 10.0 → warn
		{ID: "S3", Direction: "short", Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 102, Target: 80}}, // R 10.0 short → warn
		{ID: "S4", Direction: "long"}, // no arm → clean
	}}
	got := FantasyTargetWarnings(doc)
	if len(got) != 2 {
		t.Fatalf("warnings = %d, want 2: %v", len(got), got)
	}
	for _, w := range got {
		if !strings.Contains(w, "S2") && !strings.Contains(w, "S3") {
			t.Fatalf("unexpected warning %q", w)
		}
	}
}
