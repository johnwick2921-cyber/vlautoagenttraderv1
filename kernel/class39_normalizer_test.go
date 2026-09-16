package kernel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// CLASS 39 — the normalizer's own contract (E1b, E2, E3, E5, E9). The row
// fixtures live in class39_pin_test.go.

// E1b — the WARN names the dropped leg and the kept arm, exactly.
func TestClass39WarnNamesDroppedLeg(t *testing.T) {
	d := row69S1()
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("row 69 S1 must normalize: %v", err)
	}
	if len(d.ArmNormalizations) != 1 {
		t.Fatalf("expected exactly one recorded normalization, got %d", len(d.ArmNormalizations))
	}
	n := d.ArmNormalizations[0]
	if n.Scenario != "S1" || n.Condition != "breakdown_continue" || len(n.DroppedLegs) != 1 || n.DroppedLegs[0].Rule != "touch" {
		t.Fatalf("record wrong: %+v", n)
	}
	w := ArmNormalizationWarn(n)
	for _, want := range []string{
		"⚖ arm normalized (class 39): breakdown_continue S1",
		"dropped legs[1]",
		"#1 entry=29130.00 stop=29168.00 target=29040.00 rule=touch",
		"single arm kept entry=29130.00 stop=29168.00 target=29040.00 rule=1x5m_close wait_confirm=true",
	} {
		if !strings.Contains(w, want) {
			t.Errorf("WARN missing %q\n  %s", want, w)
		}
	}
	if got := ArmNormalizationFor(d, "S1"); got == nil || got.Scenario != "S1" {
		t.Fatalf("ArmNormalizationFor(S1) = %+v", got)
	}
	if got := ArmNormalizationFor(d, "S9"); got != nil {
		t.Fatalf("ArmNormalizationFor(S9) must be nil, got %+v", got)
	}
	if js := DroppedLegsJSON(&n); !strings.Contains(js, `"rule":"touch"`) || !strings.Contains(js, "29130") {
		t.Fatalf("DroppedLegsJSON = %s", js)
	}
}

// E2 — several legs on breakdown_continue with a valid top-level → all dropped,
// the arm proceeds, and the WARN lists EVERY dropped leg.
func TestClass39MultiLegNormalized(t *testing.T) {
	d := row69S1()
	d.Scenarios[0].Arm.Legs = []PlanArmLeg{
		{Entry: 29130.00, Stop: 29168.00, Target: 29040.00, Size: 1, WaitConfirm: false, Rule: "touch"},
		{Entry: 29122.50, Stop: 29168.00, Target: 29040.00, Size: 1, WaitConfirm: true, Rule: "1x5m_close"},
		{Entry: 29115.00, Stop: 29168.00, Target: 29040.00, Size: 1, WaitConfirm: true, Rule: "1m_mss"},
	}
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("multi-leg on breakdown_continue with a valid top-level must normalize: %v", err)
	}
	if n := len(d.Scenarios[0].Arm.Legs); n != 0 {
		t.Fatalf("all legs must be dropped, %d remain", n)
	}
	rec := d.ArmNormalizations[0]
	if len(rec.DroppedLegs) != 3 {
		t.Fatalf("record must list all 3 dropped legs, got %d", len(rec.DroppedLegs))
	}
	w := ArmNormalizationWarn(rec)
	for _, want := range []string{"dropped legs[3]", "#1 entry=29130.00", "#2 entry=29122.50", "#3 entry=29115.00", "rule=1m_mss"} {
		if !strings.Contains(w, want) {
			t.Errorf("WARN must list every dropped leg — missing %q\n  %s", want, w)
		}
	}
}

// E3 (kernel half) — legs on `reject` whose top-level arm is itself INVALID:
// REJECT UNCHANGED with the ORIGINAL reason, no record, legs left in place.
// The trader half (the retry prompt carries that reason) is
// trader.TestClass39RejectPathCarriesOriginalReason.
func TestClass39InvalidTopLevelRejectsUnchanged(t *testing.T) {
	d := lawDoc("reject", "touch", func(s *PlanScenario) {
		// short fade: contract is target < entry < stop — put the target ABOVE
		// the entry so the single arm is invalid on its own.
		s.Arm = &PlanArmSpec{
			Enabled: true, Entry: 29648.25, Stop: 29700.00, Target: 29800.00, WaitConfirm: true,
			Legs: []PlanArmLeg{{Entry: 29648.25, Stop: 29700.00, Target: 29800.00, Size: 1, Rule: "touch"}},
		}
	})
	sc := d.Scenarios[0]
	original := ArmSpecValid(sc) // what the validator says about the AUTHORED scenario
	if original == nil || !strings.Contains(original.Error(), "arm_legs_sweep_reclaim_only") {
		t.Fatalf("fixture must reject on the legs branch as authored, got %v", original)
	}
	single := *sc.Arm
	single.Legs = nil
	trial := sc
	trial.Arm = &single
	if ArmSpecValid(trial) == nil {
		t.Fatal("fixture's single arm must be invalid on its own (otherwise this tests E1, not E3)")
	}

	err := ValidatePlanDocWithCaps(d, 8, 3)
	if err == nil {
		t.Fatal("still-invalid single arm must REJECT")
	}
	if err.Error() != original.Error() {
		t.Fatalf("reject must carry the ORIGINAL reason unchanged\n  got:  %v\n  want: %v", err, original)
	}
	if strings.Contains(strings.ToLower(err.Error()), "normaliz") {
		t.Fatalf("the model's retry must never see a normalization message: %v", err)
	}
	if len(d.ArmNormalizations) != 0 {
		t.Fatalf("no normalization may be recorded on a reject, got %+v", d.ArmNormalizations)
	}
	if n := len(d.Scenarios[0].Arm.Legs); n != 1 {
		t.Fatalf("legs must be left untouched on a reject (no second pass), got %d", n)
	}
}

// E5 — NEVER SYNTHESIZE. (a) fixture: normalization only ever removes legs and
// only ever removes legs the model wrote; (b) source: no non-test kernel file
// constructs a PlanArmLeg literal or appends to a .Legs slice.
func TestClass39NeverSynthesizeALeg(t *testing.T) {
	docs := map[string]*PlanDoc{"row69S1": row69S1(), "row69S2": row69S2(), "row85S1": row85S1()}
	multi := row69S1()
	multi.Scenarios[0].Arm.Legs = append(multi.Scenarios[0].Arm.Legs, PlanArmLeg{Entry: 29122.5, Stop: 29168, Target: 29040, Size: 1, WaitConfirm: true, Rule: "1x5m_close"})
	docs["multi"] = multi
	for name, d := range docs {
		before := map[string][]PlanArmLeg{}
		total := 0
		for _, sc := range d.Scenarios {
			if sc.Arm != nil {
				before[sc.ID] = append([]PlanArmLeg(nil), sc.Arm.Legs...)
				total += len(sc.Arm.Legs)
			}
		}
		NormalizePlanDocRules(d)
		after := 0
		for _, sc := range d.Scenarios {
			if sc.Arm == nil {
				continue
			}
			after += len(sc.Arm.Legs)
			if n := len(sc.Arm.Legs); n != 0 && n != len(before[sc.ID]) {
				t.Errorf("%s %s: legs went %d → %d — normalization may only drop ALL or leave ALL", name, sc.ID, len(before[sc.ID]), n)
			}
			for _, l := range sc.Arm.Legs {
				found := false
				for _, b := range before[sc.ID] {
					if b == l {
						found = true
					}
				}
				if !found {
					t.Errorf("%s %s: leg %+v was not authored — SYNTHESIZED", name, sc.ID, l)
				}
			}
		}
		if after > total {
			t.Errorf("%s: total legs grew %d → %d", name, total, after)
		}
		for _, rec := range d.ArmNormalizations {
			for _, l := range rec.DroppedLegs {
				found := false
				for _, b := range before[rec.Scenario] {
					if b == l {
						found = true
					}
				}
				if !found {
					t.Errorf("%s: recorded dropped leg %+v was never authored", name, l)
				}
			}
		}
	}

	// (b) the source guard — the kernel package must contain no leg factory.
	literal := regexp.MustCompile(`PlanArmLeg\{\s*[A-Za-z]`)
	appendLegs := regexp.MustCompile(`\.Legs\s*=\s*append\(`)
	files, _ := filepath.Glob("*.go")
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		checked++
		if m := literal.Find(src); m != nil {
			t.Errorf("%s constructs a PlanArmLeg literal (%q) — the kernel must never manufacture a leg", f, m)
		}
		if m := appendLegs.Find(src); m != nil {
			t.Errorf("%s appends to a .Legs slice (%q) — the kernel must never grow legs", f, m)
		}
	}
	if checked < 50 {
		t.Fatalf("source guard scanned only %d files — is the test running in the kernel package dir?", checked)
	}
}

// E9 — replay every retained rejected prompt that carries a legs array through
// the new path and report the split: normalize-and-proceed vs still-reject.
// The docs are the verbatim model outputs embedded in the repair prompts of
// planner_rejected_prompts rows 69 and 85 (kernel/testdata/class39_row*.json).
func TestClass39ReplayRetainedRows(t *testing.T) {
	type expect struct {
		normalized []string // scenario ids that must normalize
		untouched  []string // scenario ids with legs that must stay
		verdictHas string   // substring the FINAL doc verdict must carry ("" = valid)
	}
	cases := map[string]expect{
		"class39_row69.json": {normalized: []string{"S1"}, untouched: []string{"S2"}, verdictHas: "needs EXACTLY 2 legs"},
		"class39_row85.json": {normalized: nil, untouched: []string{"S1"}, verdictHas: "needs EXACTLY 2 legs"},
	}
	for file, exp := range cases {
		raw, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		var d PlanDoc
		if err := json.Unmarshal(raw, &d); err != nil {
			t.Fatalf("%s: unmarshal: %v", file, err)
		}
		legsBefore := map[string]int{}
		for _, sc := range d.Scenarios {
			if sc.Arm != nil {
				legsBefore[sc.ID] = len(sc.Arm.Legs)
			}
		}
		NormalizePlanDocRules(&d)
		gotNorm := map[string]bool{}
		for _, n := range d.ArmNormalizations {
			gotNorm[n.Scenario] = true
			t.Logf("%s %s %s: NORMALIZE-AND-PROCEED — %s", file, n.Scenario, n.Condition, ArmNormalizationWarn(n))
		}
		for _, sc := range d.Scenarios {
			if sc.Arm != nil && len(sc.Arm.Legs) > 0 {
				t.Logf("%s %s %s: legs UNTOUCHED (%d) — reverse direction, stays a reject", file, sc.ID, sc.Condition, len(sc.Arm.Legs))
			}
		}
		for _, id := range exp.normalized {
			if !gotNorm[id] {
				t.Errorf("%s: %s must normalize (had %d leg(s))", file, id, legsBefore[id])
			}
		}
		for _, id := range exp.untouched {
			if gotNorm[id] {
				t.Errorf("%s: %s (sweep_reclaim) must NOT normalize", file, id)
			}
		}
		// A11/A21 — the caps the LIVE read resolved, not the file defaults:
		//   sqlite3 -readonly data/data.db "SELECT json_extract(config,'$.day_plan.max_levels'),
		//     json_extract(config,'$.day_plan.scenario_cap') FROM strategies
		//     WHERE id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8'"  → 12 | 5
		// (row 69's doc carries 12 levels; at the shipped default of 8 the replay
		// would reject on level count before ever reaching the arm branch.)
		verdict := ValidatePlanDocWithCaps(&d, 12, 5)
		t.Logf("%s: FINAL VERDICT → %v", file, verdict)
		if exp.verdictHas == "" && verdict != nil {
			t.Errorf("%s: expected a valid doc, got %v", file, verdict)
		}
		if exp.verdictHas != "" && (verdict == nil || !strings.Contains(verdict.Error(), exp.verdictHas)) {
			t.Errorf("%s: expected verdict containing %q, got %v", file, exp.verdictHas, verdict)
		}
	}
}
