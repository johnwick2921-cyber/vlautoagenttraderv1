package kernel

import (
	"encoding/json"
	"strings"
	"testing"
)

// W3 foundation — the entry policy table, the policy-aware armable/kind
// resolution, and the ONE zone verdict every W3 seam (write-time check,
// composer, executor) reads.

func TestEntryPolicyLegalTable(t *testing.T) {
	for _, c := range KnownConditions() {
		if err := EntryPolicyLegal(c, EntryPolicyMarketInZone, 0); err != nil {
			t.Errorf("market_in_zone must be legal on %s: %v", c, err)
		}
		if err := EntryPolicyLegal(c, "", 0); err != nil {
			t.Errorf("an absent policy is legacy and always legal (%s): %v", c, err)
		}
	}
	for _, c := range []string{"reject", "fvg_entry"} {
		if err := EntryPolicyLegal(c, EntryPolicyPlannedOrder, 0); err != nil {
			t.Errorf("planned_order must be legal on %s: %v", c, err)
		}
	}
	if err := EntryPolicyLegal("sweep_reclaim", EntryPolicyPlannedOrder, 0); err != nil {
		t.Errorf("planned_order is legal on sweep_reclaim leg 0: %v", err)
	}
	if err := EntryPolicyLegal("sweep_reclaim", EntryPolicyPlannedOrder, 1); err == nil {
		t.Error("planned_order is NOT legal on sweep_reclaim leg 1")
	}
	for _, c := range []string{"reclaim", "hold", "acceptance", "breakout_retest", "breakdown_continue", "breakup_continue"} {
		if err := EntryPolicyLegal(c, EntryPolicyPlannedOrder, 0); err == nil {
			t.Errorf("planned_order must NOT be legal on %s", c)
		}
	}
	if err := EntryPolicyLegal("reject", "market", 0); err == nil || !strings.Contains(err.Error(), "unknown entry policy") {
		t.Errorf("an unknown policy token must be refused by name, got %v", err)
	}
	if err := EntryPolicyLegal("not_a_condition", EntryPolicyMarketInZone, 0); err == nil {
		t.Error("market_in_zone on an unknown condition must be refused")
	}
}

func TestArmableAndKindUnderThePolicy(t *testing.T) {
	for _, c := range KnownConditions() {
		if !ArmableConditionFor(c, EntryPolicyMarketInZone) {
			t.Errorf("%s must be armable under market_in_zone", c)
		}
		if k := ArmKindForPolicy(c, EntryPolicyMarketInZone); k != ArmKindLimit {
			t.Errorf("%s under market_in_zone must be a limit, got %q", c, k)
		}
		// Legacy (absent) and planned_order keep today's answers exactly.
		for _, p := range []string{"", EntryPolicyPlannedOrder} {
			if ArmableConditionFor(c, p) != ArmableCondition(c) {
				t.Errorf("%s policy %q must keep the legacy armable answer", c, p)
			}
			if ArmKindForPolicy(c, p) != ArmKindFor(c) {
				t.Errorf("%s policy %q must keep the legacy kind", c, p)
			}
		}
	}
}

func TestEffectiveArmPolicyLegOverridesArm(t *testing.T) {
	arm := &PlanArmSpec{Policy: EntryPolicyMarketInZone}
	if got := EffectiveArmPolicy(arm, nil); got != EntryPolicyMarketInZone {
		t.Fatalf("no leg → the arm's policy, got %q", got)
	}
	if got := EffectiveArmPolicy(arm, &PlanArmLeg{Policy: EntryPolicyPlannedOrder}); got != EntryPolicyPlannedOrder {
		t.Fatalf("a leg's policy overrides the arm's, got %q", got)
	}
	if got := EffectiveArmPolicy(nil, nil); got != "" {
		t.Fatalf("no arm → legacy, got %q", got)
	}
}

func TestLegacyArmMarshalsWithoutAPolicyKey(t *testing.T) {
	b, _ := json.Marshal(PlanArmSpec{Enabled: true, Entry: 1, Stop: 2, Target: 3, Legs: []PlanArmLeg{{Entry: 1}}})
	if strings.Contains(string(b), "policy") {
		t.Fatalf("a legacy arm must re-marshal byte-identically (no policy key): %s", b)
	}
}

func zoneScenario(cond, dir string, lo, hi float64, confirm *PlanConfirm) PlanScenario {
	return PlanScenario{ID: "S1", Condition: cond, Direction: dir, Confirm: confirm,
		Economics: &ScenarioEconomics{EntryZone: []float64{lo, hi}}}
}

func TestArmZoneVerdictLongAndShort(t *testing.T) {
	// long reject at a zone 31000–31010, touch ref 31005, stop below, target above.
	sc := zoneScenario("reject", "long", 31000, 31010, &PlanConfirm{Rule: "touch", RefPrice: 31005, Side: "below"})
	v := ArmZoneVerdict(sc, 31005, 30980, 31060, "long", 0.25, 10)
	if v.Code != "" || v.Lo != 31000 || v.Hi != 31010 || v.Far != 31010 || v.Near != 31000 {
		t.Fatalf("long verdict = %+v", v)
	}
	ss := zoneScenario("reject", "short", 31000, 31010, &PlanConfirm{Rule: "touch", RefPrice: 31005, Side: "above"})
	vs := ArmZoneVerdict(ss, 31005, 31030, 30950, "short", 0.25, 10)
	if vs.Code != "" || vs.Far != 31000 || vs.Near != 31010 {
		t.Fatalf("short verdict = %+v (far must be the zone LOW for a sell)", vs)
	}
}

func TestArmZoneVerdictRefusals(t *testing.T) {
	touch := &PlanConfirm{Rule: "touch", RefPrice: 31005, Side: "below"}
	cases := []struct {
		name  string
		sc    PlanScenario
		entry float64
		stop  float64
		tgt   float64
		side  string
		max   float64
		code  string
	}{
		{"missing zone", PlanScenario{ID: "S1", Condition: "reject", Confirm: touch}, 31005, 30980, 31060, "long", 10, ZoneMissing},
		{"inverted zone", zoneScenario("reject", "long", 31010, 31000, touch), 31005, 30980, 31060, "long", 10, ZoneMissing},
		{"too wide", zoneScenario("reject", "long", 31000, 31010.25, touch), 31005, 30980, 31060, "long", 10, ZoneTooWide},
		{"width exactly max passes", zoneScenario("reject", "long", 31000, 31010, touch), 31005, 30980, 31060, "long", 10, ""},
		{"entry below lo", zoneScenario("reject", "long", 31000, 31010, touch), 30999.75, 30980, 31060, "long", 10, ZoneEntryOutside},
		{"entry at lo inclusive", zoneScenario("reject", "long", 31000, 31010, touch), 31000, 30980, 31060, "long", 10, ""},
		{"entry at hi inclusive", zoneScenario("reject", "long", 31000, 31010, touch), 31010, 30980, 31060, "long", 10, ""},
		{"stop inside the zone", zoneScenario("reject", "long", 31000, 31010, touch), 31005, 31002, 31060, "long", 10, ZoneBracket},
		{"target inside the zone", zoneScenario("reject", "long", 31000, 31010, touch), 31005, 30980, 31008, "long", 10, ZoneBracket},
		{"bad side", zoneScenario("reject", "long", 31000, 31010, touch), 31005, 30980, 31060, "sideways", 10, ZoneBadSide},
		{"touch ref outside the zone", zoneScenario("reject", "long", 31000, 31010, &PlanConfirm{Rule: "touch", RefPrice: 31020, Side: "below"}), 31005, 30980, 31060, "long", 10, ZoneTriggerSide},
		{"close-confirm zone on the wrong side of ref", zoneScenario("reclaim", "long", 31000, 31010, &PlanConfirm{Rule: "1x5m_close", RefPrice: 31008, Side: "above"}), 31005, 30980, 31060, "long", 10, ZoneTriggerSide},
		{"close-confirm zone above ref passes", zoneScenario("reclaim", "long", 31008, 31016, &PlanConfirm{Rule: "1x5m_close", RefPrice: 31008, Side: "above"}), 31010, 30980, 31060, "long", 10, ""},
	}
	for _, tc := range cases {
		v := ArmZoneVerdict(tc.sc, tc.entry, tc.stop, tc.tgt, tc.side, 0.25, tc.max)
		if v.Code != tc.code {
			t.Errorf("%s: code %q, want %q (%+v)", tc.name, v.Code, tc.code, v)
		}
	}
}

func TestArmZoneVerdictRoundsInward(t *testing.T) {
	touch := &PlanConfirm{Rule: "touch", RefPrice: 31005, Side: "below"}
	v := ArmZoneVerdict(zoneScenario("reject", "long", 31000.10, 31009.90, touch), 31005, 30980, 31060, "long", 0.25, 10)
	if v.Code != "" || v.Lo != 31000.25 || v.Hi != 31009.75 || v.Far != 31009.75 {
		t.Fatalf("inward rounding: lo up, hi down; got %+v", v)
	}
	e := ArmZoneVerdict(zoneScenario("reject", "long", 100.10, 100.20, &PlanConfirm{Rule: "touch", RefPrice: 100.15, Side: "below"}), 100.15, 99, 102, "long", 0.25, 10)
	if e.Code != ZoneEmpty {
		t.Fatalf("a zone with no tick inside rounds to empty: %+v", e)
	}
}
