package kernel

import (
	"fmt"
	"testing"
)

// Wave 2 armed orders — Phase 1 contract tests (classification, spec validity,
// entry-price derivation).

func TestArmableCondition(t *testing.T) {
	for _, c := range []string{"fvg_entry", "reject"} {
		if !ArmableCondition(c) {
			t.Fatalf("%s must be armable", c)
		}
	}
	// GAR-F4: breakout_retest was EXCLUDED (negative at every R-floor in the
	// 2026-08-28 autopsy) — it stays a normal AI play and is never armed.
	//
	// RECLAIM MOVED OUT 2026-09-04 (owner ruling, arms-follow-bias B). It was
	// excluded here as "close-confirm first → AI path". It arms now as a
	// STOP-ENTRY beyond the trigger, which only fills if price actually trades
	// back through the level — the mechanism, not a waiver of the reasoning.
	// Note the nuance honestly: a trade-through is NOT a close confirm, and the
	// original exclusion asked for a close. The ruling accepts that trade, and
	// it is the change that makes long-biased plans armable at all (19 long
	// plans were stranded with no armable play in their own direction).
	for _, c := range []string{"acceptance", "sweep_reclaim", "hold", "breakout_retest"} {
		if ArmableCondition(c) {
			t.Fatalf("%s must NOT be armable (close-confirm first → AI path)", c)
		}
	}
}

func TestArmSpecValidLongShort(t *testing.T) {
	long := PlanScenario{ID: "S1", Condition: "fvg_entry", Direction: "long",
		Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110}}
	if err := ArmSpecValid(long); err != nil {
		t.Fatalf("valid long arm rejected: %v", err)
	}
	short := PlanScenario{ID: "S2", Condition: "reject", Direction: "short",
		Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 105, Target: 90}}
	if err := ArmSpecValid(short); err != nil {
		t.Fatalf("valid short arm rejected: %v", err)
	}
	// non-armable condition
	bad := PlanScenario{ID: "S3", Condition: "acceptance", Direction: "long",
		Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110}}
	if err := ArmSpecValid(bad); err == nil {
		t.Fatal("arm on acceptance must be rejected")
	}
	// inverted long ordering
	inv := PlanScenario{ID: "S4", Condition: "reject", Direction: "long",
		Arm: &PlanArmSpec{Enabled: true, Entry: 100, Stop: 105, Target: 110}}
	if err := ArmSpecValid(inv); err == nil {
		t.Fatal("long with stop above entry must be rejected")
	}
	// disabled arm on any condition is fine
	off := PlanScenario{ID: "S5", Condition: "acceptance", Direction: "long",
		Arm: &PlanArmSpec{Enabled: false}}
	if err := ArmSpecValid(off); err != nil {
		t.Fatalf("disabled arm must pass: %v", err)
	}
}

func TestArmedEntryPx(t *testing.T) {
	fvg := PlanScenario{ID: "S1", Condition: "fvg_entry", Direction: "long",
		Fvg: &PlanFvgEntry{Lo: 98, Hi: 102, CE: 100, EntryMode: "ce"}}
	if got := ArmedEntryPx(fvg, 0, 0.25); got != 100 {
		t.Fatalf("fvg ce entry = %.2f want 100", got)
	}
	fvg.Fvg.EntryMode = "edge"
	if got := ArmedEntryPx(fvg, 0, 0.25); got != 102 {
		t.Fatalf("fvg edge long entry = %.2f want 102 (gap top)", got)
	}
	fvg.Direction = "short"
	if got := ArmedEntryPx(fvg, 0, 0.25); got != 98 {
		t.Fatalf("fvg edge short entry = %.2f want 98 (gap bottom)", got)
	}
	bt := PlanScenario{ID: "S2", Condition: "breakout_retest", Direction: "long"}
	if got := ArmedEntryPx(bt, 29628.5, 0.25); got != 29628.5 {
		t.Fatalf("breakout_retest entry = %.2f want the anchor", got)
	}
	rjLong := PlanScenario{ID: "S3", Condition: "reject", Direction: "long"}
	if got := ArmedEntryPx(rjLong, 100, 0.25); got != 99.75 {
		t.Fatalf("reject long entry = %.2f want anchor − tick", got)
	}
	rjShort := PlanScenario{ID: "S4", Condition: "reject", Direction: "short"}
	if got := ArmedEntryPx(rjShort, 100, 0.25); got != 100.25 {
		t.Fatalf("reject short entry = %.2f want anchor + tick", got)
	}
}

func TestArmSpecValidatedByPlanValidator(t *testing.T) {
	base := `{"reasoning":"r","bias":{"direction":"long","conviction":"low","flip_condition":"n/a"},"levels":[{"price":100,"label":"PDH","grade":"A","instruction":"fade"}],"scenarios":[%s],"no_trade":[],"death_condition":"n/a"}`
	// arm on a non-armable condition → the plan validator rejects it.
	bad := `{"id":"S1","trigger":"t","condition":"acceptance","direction":"long","target_chain":[110],"invalid":"i","quality":"B","arm":{"enabled":true,"entry":100,"stop":95,"target":110}}`
	if _, err := ParsePlanDoc(fmt.Sprintf(base, bad)); err == nil {
		t.Fatal("plan validator must reject arm on acceptance")
	}
	// arm on fvg_entry with valid prices → accepted.
	good := `{"id":"S1","trigger":"t","condition":"fvg_entry","direction":"long","target_chain":[110],"invalid":"i","quality":"B","fvg":{"fvg_lo":98,"fvg_hi":102,"ce":100,"entry_mode":"ce","displacement_atr":1.5,"origin_level":"ONH","direction":"long"},"arm":{"enabled":true,"entry":100,"stop":95,"target":110}}`
	if _, err := ParsePlanDoc(fmt.Sprintf(base, good)); err != nil {
		t.Fatalf("valid armed fvg plan rejected: %v", err)
	}
}
