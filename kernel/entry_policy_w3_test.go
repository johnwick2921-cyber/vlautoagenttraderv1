package kernel

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"nofx/store"
)

// W-EXEC-TRUTH W3 (K) — the validator, the R4 stamp and the armable hold
// floor, driven through the PRODUCTION parse (ParsePlanDocForAuthoring = the
// planner write loop's call, trader/auto_trader_planner.go), and the stored
// reader (ParsePlanDoc) for the legacy byte-identity half.

// w3ScenarioDoc builds a one-scenario plan whose ONLY variable is the arm's
// shape and policy: every other contract (entry law, economics, fvg/breakdown
// facts) is satisfied, so a refusal can only come from the arm.
//
// policy "-" = no policy key at all (legacy).
func w3ScenarioDoc(t *testing.T, cond, policy string, split, waitConfirm bool) string {
	t.Helper()
	short := IsBreakdownCondition(cond) && breakdownShort(cond)
	dir, side, lvl := "long", "above", 21500.0
	entry, stop, target, obstacle := 21500.0, 21480.0, 21580.0, 21540.0
	zone := []float64{21496, 21502}
	if short {
		dir, side = "short", "below"
		entry, stop, target, obstacle = 21500, 21520, 21420, 21460
		zone = []float64{21498, 21504}
	}
	sc := map[string]any{
		"id": "S1", "condition": cond, "direction": dir, "quality": "A",
		"trigger":      fmt.Sprintf("the %s play at %.2f", cond, lvl),
		"target_chain": []float64{obstacle, target},
		"invalid":      fmt.Sprintf("5m close below %.2f", stop),
	}
	if short {
		sc["invalid"] = fmt.Sprintf("5m close above %.2f", stop)
	}
	switch cond {
	case "reject", "fvg_entry", "breakout_retest":
		sc["confirm"] = map[string]any{"rule": "touch", "ref_price": lvl, "side": "below"}
	case "sweep_reclaim":
		sc["confirm"] = map[string]any{"rule": "touch", "ref_price": lvl, "side": "below"}
		sc["confirm2"] = map[string]any{"rule": "1m_mss", "ref_price": lvl, "side": "above"}
	case "reclaim":
		sc["confirm"] = map[string]any{"rule": "1x5m_close", "ref_price": lvl, "side": "above"}
	case "hold", "acceptance":
		sc["confirm"] = map[string]any{"rule": "time_hold", "ref_price": lvl, "side": "above", "hold_min": 5}
	default: // waterfalls
		sc["confirm"] = map[string]any{"rule": "1x5m_close", "ref_price": lvl, "side": side}
		sc["breakdown"] = map[string]any{"level": lvl, "level_label": "VWAP", "entry_mode": "pullback"}
	}
	if cond == "fvg_entry" {
		sc["fvg"] = map[string]any{"fvg_lo": 21490, "fvg_hi": 21502, "entry_mode": "edge", "displacement_atr": 1.5, "origin_level": "VWAP", "direction": "long"}
	}
	risk := entry - stop
	if risk < 0 {
		risk = -risk
	}
	abs := func(v float64) float64 {
		if v < 0 {
			return -v
		}
		return v
	}
	sc["economics"] = map[string]any{
		"entry_zone":      zone,
		"first_obstacle":  map[string]any{"price": obstacle, "level": "fixture reference", "family": "reference", "response": "pass_through"},
		"r_to_obstacle":   abs(obstacle-entry) / risk,
		"r_to_arm_target": abs(target-entry) / risk,
	}
	arm := map[string]any{"enabled": true, "entry": entry, "stop": stop, "target": target, "wait_confirm": waitConfirm}
	if policy != "-" {
		arm["policy"] = policy
	}
	if split {
		leg1Entry, leg1Stop := entry+4, stop+4
		if short {
			leg1Entry, leg1Stop = entry-4, stop-4
		}
		arm["legs"] = []any{
			map[string]any{"entry": entry, "stop": stop, "target": target, "size": 1, "wait_confirm": false},
			map[string]any{"entry": leg1Entry, "stop": leg1Stop, "target": target, "size": 1, "wait_confirm": true, "rule": "1m_mss"},
		}
	}
	sc["arm"] = arm
	doc := map[string]any{
		"reasoning": "bias-tree: fixture", "bias": map[string]any{"direction": dir, "conviction": "medium", "flip_condition": "fixture"},
		"levels":          []any{map[string]any{"price": lvl, "label": "VWAP", "grade": "A", "instruction": "fixture"}},
		"scenarios":       []any{sc},
		"death_condition": "fixture",
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// w3Parse is the production new-authoring parse with the stamp OFF (the
// arm's own policy is the one under test) and the hold floor at its default.
func w3Parse(raw string, def string) (*PlanDoc, error) {
	return ParsePlanDocForAuthoring(raw, 12, 5, AuthoringOpts{MinRR: 2.0, EntryPolicyDefault: def, MinHoldMin: store.MinHoldMinDefault})
}

// The expected verdict is DERIVED from the policy table (EntryPolicyLegal,
// ArmableCondition), never typed per condition.
func w3ExpectSingleOK(cond, policy string) bool {
	switch policy {
	case "-":
		return ArmableCondition(cond) || cond == "sweep_reclaim"
	default:
		return EntryPolicyLegal(cond, policy, 0) == nil
	}
}

func TestW3ValidatorNineConditionsByPolicyAndShape(t *testing.T) {
	policies := []string{"-", EntryPolicyMarketInZone, EntryPolicyPlannedOrder}
	for _, cond := range KnownConditions() {
		for _, pol := range policies {
			for _, split := range []bool{false, true} {
				name := fmt.Sprintf("%s/%s/split=%v", cond, pol, split)
				doc, err := w3Parse(w3ScenarioDoc(t, cond, pol, split, true), EntryPolicyDefaultLegacy)
				want := w3ExpectSingleOK(cond, pol)
				if split && cond == "sweep_reclaim" {
					// The split: leg 1 inherits the arm's policy — planned_order is
					// illegal on sweep_reclaim leg 1 (the table), the other two pass.
					want = pol == "-" || EntryPolicyLegal(cond, pol, 1) == nil
				}
				if want && err != nil {
					t.Errorf("%s: must be accepted, refused: %v", name, err)
					continue
				}
				if !want {
					if err == nil {
						t.Errorf("%s: must be REFUSED, accepted", name)
					} else if pol != "-" && !strings.Contains(err.Error(), "planned_order") {
						t.Errorf("%s: a policy refusal must name the policy, got %v", name, err)
					}
					continue
				}
				if split && cond != "sweep_reclaim" {
					// class 39: legs on a non-sweep condition collapse to the single arm.
					if len(doc.Scenarios[0].Arm.Legs) != 0 || len(doc.ArmNormalizations) != 1 {
						t.Errorf("%s: non-sweep legs must normalize to the single arm (class 39), got legs=%d norms=%d", name, len(doc.Scenarios[0].Arm.Legs), len(doc.ArmNormalizations))
					}
				}
				if pol == "-" && strings.Contains(mustJSON(t, doc.Scenarios[0].Arm), `"policy"`) {
					t.Errorf("%s: a legacy arm must carry no policy key after parse", name)
				}
			}
		}
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The legacy validator path is untouched: acceptance/hold/breakout_retest
// arms with no policy keep the GAR-F4 refusal, immediate waterfalls keep the
// AI-path refusal — byte-identical strings.
func TestW3LegacyRefusalsUnchanged(t *testing.T) {
	for _, cond := range []string{"acceptance", "hold", "breakout_retest"} {
		_, err := w3Parse(w3ScenarioDoc(t, cond, "-", false, true), EntryPolicyDefaultLegacy)
		want := fmt.Sprintf("arm enabled on non-armable condition %q (fvg_entry | reject | breakdown_continue | breakup_continue; sweep_reclaim via wait_confirm; breakout_retest is a normal AI play and never arms — GAR-F4)", cond)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s legacy: want %q, got %v", cond, want, err)
		}
	}
	imm := strings.Replace(w3ScenarioDoc(t, "breakdown_continue", "-", false, true), `"entry_mode":"pullback"`, `"entry_mode":"immediate"`, 1)
	if _, err := w3Parse(imm, EntryPolicyDefaultLegacy); err == nil || !strings.Contains(err.Error(), "S1 arm requires entry_mode=pullback (immediate-mode entries are AI-path only)") {
		t.Errorf("legacy immediate waterfall: want the AI-path-only refusal, got %v", err)
	}
}

// (a)(b) under market_in_zone: immediate waterfalls arm; D4 — a non-touch
// confirm must chain; a touch play need not; an authored stop_entry is refused.
func TestW3PolicyBranchLaw(t *testing.T) {
	imm := strings.Replace(w3ScenarioDoc(t, "breakdown_continue", EntryPolicyMarketInZone, false, true), `"entry_mode":"pullback"`, `"entry_mode":"immediate"`, 1)
	if _, err := w3Parse(imm, EntryPolicyDefaultLegacy); err != nil {
		t.Fatalf("market_in_zone immediate waterfall must arm: %v", err)
	}
	bad := strings.Replace(imm, `"entry_mode":"immediate"`, `"entry_mode":"sideways"`, 1)
	if _, err := w3Parse(bad, EntryPolicyDefaultLegacy); err == nil || !strings.Contains(err.Error(), "entry_mode=pullback or entry_mode=immediate") {
		t.Fatalf("an unknown entry_mode must be refused naming both legal modes, got %v", err)
	}
	for _, cond := range []string{"reclaim", "acceptance", "hold", "breakdown_continue", "breakup_continue"} {
		_, err := w3Parse(w3ScenarioDoc(t, cond, EntryPolicyMarketInZone, false, false), EntryPolicyDefaultLegacy)
		if err == nil || !strings.Contains(err.Error(), "wait_confirm:true") {
			t.Errorf("D4 %s (non-touch confirm) without wait_confirm must be refused, got %v", cond, err)
		}
	}
	for _, cond := range []string{"reject", "fvg_entry", "breakout_retest"} {
		if _, err := w3Parse(w3ScenarioDoc(t, cond, EntryPolicyMarketInZone, false, false), EntryPolicyDefaultLegacy); err != nil {
			t.Errorf("D4 %s (touch confirm) may rest without wait_confirm: %v", cond, err)
		}
	}
	unk := w3ScenarioDoc(t, "reject", "market", false, true)
	if _, err := w3Parse(unk, EntryPolicyDefaultLegacy); err == nil || !strings.Contains(err.Error(), `unknown entry policy "market"`) {
		t.Fatalf("an unknown policy token must be refused by name, got %v", err)
	}
	stopLeg := strings.Replace(w3ScenarioDoc(t, "sweep_reclaim", EntryPolicyMarketInZone, true, true), `"rule":"1m_mss","size":1`, `"kind":"stop_entry","rule":"1m_mss","size":1`, 1)
	if !strings.Contains(stopLeg, "stop_entry") {
		t.Fatal("fixture: the stop_entry leg was not injected")
	}
	if _, err := w3Parse(stopLeg, EntryPolicyDefaultLegacy); err == nil || !strings.Contains(err.Error(), "under market_in_zone") {
		t.Fatalf("a stop_entry leg under market_in_zone must be refused, got %v", err)
	}
	// The same split, legacy: stop_entry on leg 2 stays legal (E7).
	legacyStop := strings.Replace(w3ScenarioDoc(t, "sweep_reclaim", "-", true, true), `"rule":"1m_mss","size":1`, `"kind":"stop_entry","rule":"1m_mss","size":1`, 1)
	if _, err := w3Parse(legacyStop, EntryPolicyDefaultLegacy); err != nil {
		t.Fatalf("legacy split leg 2 stop_entry must stay legal: %v", err)
	}
}

// The split lock under the policy refuses with the IDENTICAL legacy strings.
func TestW3SplitLockIdenticalStrings(t *testing.T) {
	for _, pol := range []string{"-", EntryPolicyMarketInZone} {
		raw := strings.Replace(w3ScenarioDoc(t, "sweep_reclaim", pol, true, true), `"entry":21500,"size":1,"stop":21480,"target":21580,"wait_confirm":false`, `"entry":21500,"size":1,"stop":21480,"target":21580,"wait_confirm":true`, 1)
		_, err := w3Parse(raw, EntryPolicyDefaultLegacy)
		want := "arm on S1 leg 1 must rest at the sweep ref (wait_confirm false) — it fills ON the touch"
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("policy %s: want %q, got %v", pol, want, err)
		}
	}
}

// R4 — the stamp: the default is written onto every arm it is LEGAL for,
// before the validator; an illegal default leaves the arm legacy (fail-closed);
// a planner-set policy is never overwritten; "legacy" stamps nothing; a
// STORED doc (ParsePlanDoc) is never stamped.
func TestW3StampAtParse(t *testing.T) {
	for _, cond := range KnownConditions() {
		raw := w3ScenarioDoc(t, cond, "-", false, true)
		doc, err := w3Parse(raw, EntryPolicyMarketInZone)
		if err != nil {
			t.Errorf("%s: the market_in_zone stamp must make the arm valid: %v", cond, err)
			continue
		}
		if got := doc.Scenarios[0].Arm.Policy; got != EntryPolicyMarketInZone {
			t.Errorf("%s: stamped policy %q, want market_in_zone", cond, got)
		}
		// planned_order default: stamped only where legal; elsewhere legacy.
		pdoc, perr := w3Parse(raw, EntryPolicyPlannedOrder)
		legal := EntryPolicyLegal(cond, EntryPolicyPlannedOrder, 0) == nil
		switch {
		case legal && (perr != nil || pdoc.Scenarios[0].Arm.Policy != EntryPolicyPlannedOrder):
			t.Errorf("%s: planned_order is legal here and must be stamped: %v", cond, perr)
		case !legal && perr == nil && pdoc.Scenarios[0].Arm.Policy != "":
			t.Errorf("%s: an illegal planned_order must never be stamped, got %q", cond, pdoc.Scenarios[0].Arm.Policy)
		case !legal && perr != nil && strings.Contains(perr.Error(), "planned_order"):
			t.Errorf("%s: a skipped stamp must never surface as a policy refusal: %v", cond, perr)
		}
		// legacy default: no stamp.
		if ldoc, lerr := w3Parse(raw, EntryPolicyDefaultLegacy); lerr == nil && ldoc.Scenarios[0].Arm.Policy != "" {
			t.Errorf("%s: legacy must stamp nothing", cond)
		}
		// stored reader: never stamped (and no policy key after a re-marshal).
		if sdoc, serr := ParsePlanDoc(raw); serr == nil {
			if strings.Contains(mustJSON(t, sdoc), `"policy"`) {
				t.Errorf("%s: ParsePlanDoc (stored reader) must never stamp", cond)
			}
		}
	}
	// A planner-set policy is kept.
	raw := w3ScenarioDoc(t, "reject", EntryPolicyPlannedOrder, false, true)
	doc, err := w3Parse(raw, EntryPolicyMarketInZone)
	if err != nil || doc.Scenarios[0].Arm.Policy != EntryPolicyPlannedOrder {
		t.Fatalf("a planner-set policy must never be overwritten: %v", err)
	}
	// Split sweep under market_in_zone: arm + both legs stamped.
	sdoc, err := w3Parse(w3ScenarioDoc(t, "sweep_reclaim", "-", true, true), EntryPolicyMarketInZone)
	if err != nil {
		t.Fatal(err)
	}
	for i, l := range sdoc.Scenarios[0].Arm.Legs {
		if l.Policy != EntryPolicyMarketInZone {
			t.Errorf("split leg %d not stamped: %q", i+1, l.Policy)
		}
	}
	// Split sweep under planned_order: leg 1 is illegal → the WHOLE arm stays legacy (accepted).
	pdoc, err := w3Parse(w3ScenarioDoc(t, "sweep_reclaim", "-", true, true), EntryPolicyPlannedOrder)
	if err != nil || armHasPolicy(pdoc.Scenarios[0].Arm) {
		t.Fatalf("planned_order on a split sweep must leave the arm legacy: err=%v", err)
	}
}

// D10 — the armable hold floor: an ARMED market_in_zone time_hold below the
// floor is refused with its OWN marker (never the A5 prose law's token); an
// unarmed or legacy one is never judged; the RESOLVED hold is what counts.
func TestW3ArmableHoldFloor(t *testing.T) {
	two := strings.Replace(w3ScenarioDoc(t, "acceptance", "-", false, true), `"hold_min":5`, `"hold_min":2`, 1)
	_, err := w3Parse(two, EntryPolicyMarketInZone)
	if err == nil || !strings.HasPrefix(err.Error(), ArmableHoldFloorMarker) {
		t.Fatalf("hold 2 on an armed market_in_zone acceptance must be refused with the floor marker, got %v", err)
	}
	if strings.Contains(err.Error(), "hold_min") {
		t.Fatalf("the floor refusal must not carry the A5 routing token: %v", err)
	}
	if ex := lawExcerptsFor(err.Error()); !strings.Contains(ex, RepairArmableHoldFloorLaw) || strings.Contains(ex, RepairHoldMinLaw) {
		t.Fatalf("the repair must route THIS law only, got %q", ex)
	}
	// Legacy default → not armable-by-policy → not judged (the legacy arm is
	// refused as non-armable — a different law, not the floor).
	if _, err := w3Parse(two, EntryPolicyDefaultLegacy); err != nil && strings.HasPrefix(err.Error(), ArmableHoldFloorMarker) {
		t.Fatalf("a legacy arm is never judged by the floor: %v", err)
	}
	// Unarmed → accepted.
	unarmed := strings.Replace(two, `"enabled":true`, `"enabled":false`, 1)
	if _, err := w3Parse(unarmed, EntryPolicyMarketInZone); err != nil {
		t.Fatalf("an unarmed hold 2 must be accepted: %v", err)
	}
	// At the floor → accepted.
	three := strings.Replace(two, `"hold_min":2`, `"hold_min":3`, 1)
	if _, err := w3Parse(three, EntryPolicyMarketInZone); err != nil {
		t.Fatalf("hold 3 = the floor must be accepted: %v", err)
	}
	// The RESOLVED hold: no stored hold_min → ACCEPT_HOLD_MIN (env). With the
	// env at 2 the resolved hold is 2 → refused.
	t.Setenv("ACCEPT_HOLD_MIN", "2")
	absent := strings.Replace(two, `"hold_min":2,`, ``, 1)
	if strings.Contains(absent, "hold_min") {
		t.Fatal("fixture: hold_min not removed")
	}
	if _, err := w3Parse(absent, EntryPolicyMarketInZone); err == nil || !strings.Contains(err.Error(), "authoring_default") {
		t.Fatalf("the floor must judge the RESOLVED hold (authoring default 2), got %v", err)
	}
	// Floor 0 = off.
	if _, err := ParsePlanDocForAuthoring(two, 12, 5, AuthoringOpts{MinRR: 2, EntryPolicyDefault: EntryPolicyMarketInZone}); err != nil {
		t.Fatalf("MinHoldMin 0 = no floor: %v", err)
	}
}

// LEGACY byte-identity: the stored corpus + the class-39 rows read through the
// stored reader give the same verdict as the pre-W3 legacy validator and never
// gain a policy key.
func TestW3LegacyCorpusByteIdentical(t *testing.T) {
	files := []string{"testdata/class39_row69.json", "testdata/class39_row85.json"}
	for _, r := range []int{170, 178, 180, 182, 185, 187, 252, 265, 270} {
		files = append(files, fmt.Sprintf("testdata/scenario-economics/plan-%d.json", r))
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		raw := string(b)
		if strings.Contains(raw, `"policy"`) {
			t.Fatalf("%s: a stored fixture must predate the policy", f)
		}
		doc, err := ParsePlanDoc(raw)
		if err != nil {
			// A stored row the validator refuses must refuse identically through
			// the legacy branch (no arm has a policy → armHasPolicy false).
			continue
		}
		for _, sc := range doc.Scenarios {
			if armHasPolicy(sc.Arm) {
				t.Errorf("%s %s: a stored reader stamped a policy", f, sc.ID)
			}
			// Every armed scenario's verdict equals the legacy branch's.
			if err1, err2 := ArmSpecValid(sc), armSpecValidLegacyOnly(sc); fmt.Sprint(err1) != fmt.Sprint(err2) {
				t.Errorf("%s %s: verdict drift %v vs legacy %v", f, sc.ID, err1, err2)
			}
		}
		if strings.Contains(mustJSON(t, doc), `"policy"`) {
			t.Errorf("%s: re-marshalled stored doc gained a policy key", f)
		}
	}
}

// armSpecValidLegacyOnly runs ArmSpecValid on a copy with every policy
// removed — the legacy branch's verdict for comparison.
func armSpecValidLegacyOnly(sc PlanScenario) error {
	if sc.Arm == nil {
		return ArmSpecValid(sc)
	}
	a := *sc.Arm
	a.Policy = ""
	a.Legs = append([]PlanArmLeg(nil), a.Legs...)
	for i := range a.Legs {
		a.Legs[i].Policy = ""
	}
	sc.Arm = &a
	return ArmSpecValid(sc)
}

// The store and kernel spell the policy tokens identically (store cannot
// import kernel).
func TestStoreEntryPolicyTokensMatchKernel(t *testing.T) {
	want := []string{EntryPolicyMarketInZone, EntryPolicyPlannedOrder, EntryPolicyDefaultLegacy}
	got := store.EntryPolicyDefaultValues()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("store tokens %v != kernel %v", got, want)
	}
	if store.EntryPolicyDefaultShipped != EntryPolicyMarketInZone {
		t.Fatalf("R4: the shipped default must be market_in_zone, got %q", store.EntryPolicyDefaultShipped)
	}
}

// ArmKindMismatchPolicy / ArmableConditionsLineFor / BiasArmWarningFor under
// the policy never call acceptance/hold/breakout_retest "not armable"; off the
// policy they are the legacy functions exactly.
func TestW3ArmableLineAndBiasWarningFollowThePolicy(t *testing.T) {
	st := ResolvedConditionStatuses(nil, nil, "")
	if got := ArmableConditionsLineFor(st, ""); got != ArmableConditionsLine(st) {
		t.Fatal("legacy line must be ArmableConditionsLine exactly")
	}
	line := ArmableConditionsLineFor(st, EntryPolicyMarketInZone)
	if strings.Contains(line, "NOT armable") {
		t.Fatalf("market_in_zone must not call any condition not armable: %s", line)
	}
	for _, c := range []string{"acceptance→limit", "hold→limit", "reclaim→limit"} {
		if !strings.Contains(line, c) {
			t.Errorf("line missing %s: %s", c, line)
		}
	}
	d := &PlanDoc{Bias: PlanBias{Direction: "long"}, Scenarios: []PlanScenario{{ID: "S1", Condition: "acceptance", Direction: "long"}}}
	if w := BiasArmWarningFor(d, st, EntryPolicyMarketInZone); !strings.Contains(w, "S1 acceptance is armable but carries no arm") {
		t.Fatalf("under market_in_zone an unarmed acceptance is armable-but-unarmed, got %q", w)
	}
	if BiasArmWarningFor(d, st, "") != BiasArmWarning(d, st) {
		t.Fatal("legacy warning must be BiasArmWarning exactly")
	}
	if err := ArmKindMismatchPolicy("reclaim", EntryPolicyMarketInZone, "stop_entry"); err == nil {
		t.Fatal("stop_entry under market_in_zone must be a mismatch")
	}
	if ArmKindMismatchPolicy("reclaim", "", "stop_entry") != nil {
		t.Fatal("legacy reclaim stop_entry must stay legal")
	}
}
