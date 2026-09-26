package kernel

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/logger"
	"nofx/market"
	"os"
	"strings"
	"testing"
)

// Real stored authoring geometries; the economics declaration is a test input,
// not an inferred/backfilled field in the stored fixture.
func economicsRaw(t *testing.T, row int, id string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(fmt.Sprintf("testdata/scenario-economics/plan-%d.json", row))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for _, v := range doc["scenarios"].([]any) {
		s := v.(map[string]any)
		if s["id"] != id {
			continue
		}
		doc["scenarios"] = []any{s}
		a := s["arm"].(map[string]any)
		e, st, target := a["entry"].(float64), a["stop"].(float64), a["target"].(float64)
		obstacle := s["target_chain"].([]any)[0].(float64)
		risk := math.Abs(e - st)
		s["economics"] = map[string]any{"entry_zone": []float64{e, e}, "first_obstacle": map[string]any{"price": obstacle, "level": "fixture declared obstacle", "family": "reference", "response": "pass_through"}, "r_to_obstacle": math.Abs(obstacle-e) / risk, "r_to_arm_target": math.Abs(target-e) / risk}
		return doc
	}
	t.Fatalf("missing %d/%s", row, id)
	return nil
}
func economicsScenario(d map[string]any) map[string]any {
	return d["scenarios"].([]any)[0].(map[string]any)
}
func economicsJSON(t *testing.T, d map[string]any) string {
	t.Helper()
	b, e := json.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func TestScenarioEconomicsNewAuthoringRefusals(t *testing.T) {
	cases := []struct {
		name string
		row  int
		id   string
		mut  func(map[string]any)
		want []string
	}{
		{"C3 actual off-path", 178, "S1", func(s map[string]any) {}, []string{"target_path", "29418.62", "29422.12"}},
		{"new missing obstacle", 265, "S2", func(s map[string]any) { delete(s["economics"].(map[string]any), "first_obstacle") }, []string{"first_obstacle", "required"}},
		{"new missing whole contract", 265, "S2", func(s map[string]any) { delete(s, "economics") }, []string{"economics", "required"}},
		{"obstacle beyond long target", 265, "S2", func(s map[string]any) {
			s["economics"].(map[string]any)["first_obstacle"].(map[string]any)["price"] = 29730.0
		}, []string{"obstacle_beyond_target", "29730", "29721.25"}},
		{"wrong implied R", 265, "S2", func(s map[string]any) { s["economics"].(map[string]any)["r_to_arm_target"] = 3.0 }, []string{"implied_r", "3"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := economicsRaw(t, tc.row, tc.id)
			tc.mut(economicsScenario(d))
			_, err := ParsePlanDocCapped(economicsJSON(t, d), 12, 5)
			if err == nil {
				t.Fatal("new authoring accepted contradiction/missing contract")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("%v missing %q", err, w)
				}
			}
		})
	}
}

func TestScenarioEconomicsLegacyReadUnchanged(t *testing.T) {
	d := economicsRaw(t, 265, "S2")
	delete(economicsScenario(d), "economics")
	var stored PlanDoc
	if err := json.Unmarshal([]byte(economicsJSON(t, d)), &stored); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePlanDocWithCaps(&stored, 12, 5); err != nil {
		t.Fatalf("legacy refused: %v", err)
	}
	b, _ := json.Marshal(stored)
	var read map[string]any
	json.Unmarshal(b, &read)
	if _, exists := economicsScenario(read)["economics"]; exists {
		t.Fatal("legacy economics backfilled")
	}
}

func TestScenarioEconomicsSub1AcceptedAndPreserved(t *testing.T) {
	d := economicsRaw(t, 265, "S2")
	doc, err := ParsePlanDocCapped(economicsJSON(t, d), 12, 5)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(doc)
	var read map[string]any
	json.Unmarshal(b, &read)
	if economicsScenario(read)["economics"] == nil {
		t.Fatal("accepted economics discarded; cannot render/count first obstacle")
	}
}

func TestScenarioEconomicsCountersAndRoleWarnings(t *testing.T) {
	before := ScenarioEconomicsCounters()
	raw := economicsRaw(t, 265, "S2")
	d, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5)
	if err != nil {
		t.Fatal(err)
	}
	v := EconomicsFor(d.Scenarios[0])
	after := ScenarioEconomicsCounters()
	if !v.Sub1 || v.ObstacleR == nil || v.ArmR == nil || after.Sub1 != before.Sub1+1 {
		t.Fatalf("WARN not counted/marked: %+v %+v -> %+v", v, before, after)
	}
	if !strings.Contains(EconomicsSummary(d.Scenarios[0]), "sub-1R") {
		t.Fatal("sub-1R absent from production summary")
	}
	for _, id := range []string{"S2", "S3"} {
		raw = economicsRaw(t, 270, id)
		before = ScenarioEconomicsCounters()
		d, err = ParsePlanDocCapped(economicsJSON(t, raw), 12, 5)
		if err != nil {
			t.Fatalf("role difference must WARN only: %v", err)
		}
		after = ScenarioEconomicsCounters()
		if after.RoleWarnings <= before.RoleWarnings {
			t.Fatal("role WARN was not counted at production call site")
		}
		warnings := strings.Join(scenarioRoleWarnings(d.Scenarios[0], d.Levels), " | ")
		want := "Supply·1h 29657.38 role=confluence use=target"
		if id == "S3" {
			want = "SWG-L·5m 29675.75 role=target use=invalidation"
		}
		if !strings.Contains(warnings, want) {
			t.Fatalf("missing real role/use evidence %s: %s", want, warnings)
		}
	}
}

func TestScenarioEconomicsExceptionTickAndShortMirror(t *testing.T) {
	raw := economicsRaw(t, 178, "S1")
	s := economicsScenario(raw)
	e := s["economics"].(map[string]any)
	e["target_path_exception"] = "authored order objective before the next path reference"
	if _, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5); err != nil {
		t.Fatal(err)
	}
	e["first_obstacle"].(map[string]any)["price"] = 29400.0
	if _, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5); err == nil || !strings.Contains(err.Error(), "obstacle_beyond_target") {
		t.Fatalf("short mirror accepted: %v", err)
	}
	raw = economicsRaw(t, 265, "S2")
	s = economicsScenario(raw)
	e = s["economics"].(map[string]any)
	tick := market.FuturesTickSize("MNQ")
	e["r_to_arm_target"] = 56.75/24.5 + tick/24.5
	if _, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5); err != nil {
		t.Fatalf("one tick boundary refused: %v", err)
	}
	e["r_to_arm_target"] = 56.75/24.5 + (tick+0.001)/24.5
	if _, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5); err == nil {
		t.Fatal("greater than one tick accepted")
	}
}

// R4 (owner ruling 2026-09-15): a misstated r_to_arm_target whose computed
// value is at or above the minimum floor is auto-corrected and accepted, not
// refused. The strict contradiction survives for sub-floor computed R and for
// call sites that pass no floor (stored readers, offline validator).
func TestScenarioEconomicsOverMinAutoCorrected(t *testing.T) {
	before := ScenarioEconomicsCounters()
	raw := economicsRaw(t, 265, "S2")
	s := economicsScenario(raw)
	e := s["economics"].(map[string]any)
	e["r_to_arm_target"] = 2.0 // rounded shorthand; the true value is well above 2
	e["target_path_exception"] = "R4 auto-correct test"
	doc, err := ParsePlanDocCappedWithMinRR(economicsJSON(t, raw), 12, 5, 2.0)
	if err != nil {
		t.Fatal(err)
	}
	after := ScenarioEconomicsCounters()
	if after.Corrected != before.Corrected+1 || after.Contradictions != before.Contradictions {
		t.Fatalf("over-min mismatch must correct, not refuse: %+v -> %+v", before, after)
	}
	got := doc.Scenarios[0].Economics.RToArmTarget
	want := EconomicsFor(doc.Scenarios[0]).ArmR
	if got == nil || want == nil || math.Abs(*got-*want) > 1e-9 {
		t.Fatalf("stated R was not corrected to computed: stated=%v computed=%v", got, want)
	}
}

func TestScenarioEconomicsUnderMinStillRefused(t *testing.T) {
	raw := economicsRaw(t, 265, "S2")
	s := economicsScenario(raw)
	e := s["economics"].(map[string]any)
	arm := s["arm"].(map[string]any)
	entry, stop := arm["entry"].(float64), arm["stop"].(float64)
	risk := math.Abs(entry - stop)
	arm["target"] = entry + 1.9*risk // computed R 1.9 < floor 2.0
	e["r_to_arm_target"] = 2.5       // claimed above the floor
	e["target_path_exception"] = "R4 lie-direction test"
	if _, err := ParsePlanDocCappedWithMinRR(economicsJSON(t, raw), 12, 5, 2.0); err == nil || !strings.Contains(err.Error(), "disagrees") {
		t.Fatalf("sub-floor computed R must stay refused: %v", err)
	}
}

func TestScenarioEconomicsZeroMinStrict(t *testing.T) {
	raw := economicsRaw(t, 265, "S2")
	s := economicsScenario(raw)
	e := s["economics"].(map[string]any)
	e["r_to_arm_target"] = 2.0
	e["target_path_exception"] = "R4 zero-floor test"
	if _, err := ParsePlanDocCappedWithMinRR(economicsJSON(t, raw), 12, 5, 0); err == nil || !strings.Contains(err.Error(), "disagrees") {
		t.Fatalf("zero floor must keep the strict contradiction: %v", err)
	}
}

func TestScenarioEconomicsLegacyViewAndNoVersionBypass(t *testing.T) {
	raw := economicsRaw(t, 265, "S2")
	s := economicsScenario(raw)
	delete(s, "economics")
	var doc PlanDoc
	json.Unmarshal([]byte(economicsJSON(t, raw)), &doc)
	v := EconomicsFor(doc.Scenarios[0])
	if v.ObstacleR != nil || v.ArmR != nil {
		t.Fatal("inferred legacy R")
	}
	before := ScenarioEconomicsCounters()
	if err := ValidatePlanDocWithCaps(&doc, 12, 5); err != nil {
		t.Fatal(err)
	}
	if ScenarioEconomicsCounters() != before {
		t.Fatal("legacy read emitted authoring telemetry")
	}
	s["economics"] = map[string]any{"version": 0}
	if _, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5); err == nil {
		t.Fatal("model-controlled version bypassed authoring contract")
	}
}

func TestScenarioEconomicsNewHypotheticalGeometryNeverArms(t *testing.T) {
	raw := economicsRaw(t, 265, "S2")
	s := economicsScenario(raw)
	s["economics"].(map[string]any)["geometry"] = s["arm"]
	delete(s, "arm")
	d, err := ParsePlanDocCapped(economicsJSON(t, raw), 12, 5)
	if err != nil {
		t.Fatal(err)
	}
	if d.Scenarios[0].Arm != nil || EconomicsFor(d.Scenarios[0]).ArmR == nil {
		t.Fatal("hypothetical geometry lost or became arm permission")
	}
}

func TestScenarioEconomicsBootAndProductionWiring(t *testing.T) {
	c := ScenarioEconomicsCounters()
	line := ScenarioEconomicsBootLine()
	if !strings.Contains(line, "legacy UNKNOWN by design") {
		t.Fatalf("boot must explain intentional legacy UNKNOWN: %s", line)
	}
	for _, w := range []string{"contract=on", "obstacle-required=on", fmt.Sprintf("target-path-coherent=%d/%d", c.PathCoherent, c.PathEvaluated), fmt.Sprintf("sub-1R-first-obstacle=%d", c.Sub1), fmt.Sprintf("role-use-disagreements=%d", c.RoleWarnings), fmt.Sprintf("contradictions refused=%d", c.Contradictions), fmt.Sprintf("corrected=%d", c.Corrected)} {
		if !strings.Contains(line, w) {
			t.Fatalf("boot not reading counters: %s missing %s", line, w)
		}
	}
	// Call-site pins: moving/removing the call may leave the pure tests green.
	// W3: the write loop parses through ParsePlanDocForAuthoring with the
	// resolved opts, whose MinRR is still at.armMinRRFor(nil).
	for _, pin := range []struct{ path, call string }{{"levels_volume_boot.go", "logger.Info(ScenarioEconomicsBootLine())"}, {"../trader/auto_trader_planner.go", "kernel.ParsePlanDocForAuthoring(raw, maxLevels, scenarioCap, at.plannerAuthoringOpts())"}, {"../trader/entry_policy_authoring.go", "at.armMinRRFor(nil)"}} {
		b, err := os.ReadFile(pin.path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), pin.call) {
			t.Fatalf("production call missing: %s", pin.call)
		}
	}
}

type economicsPanicWriter struct{}

func (economicsPanicWriter) Write([]byte) (int, error) { panic("telemetry test") }
func TestScenarioEconomicsTelemetryCannotPanicOrPermit(t *testing.T) {
	old := logger.Log.Out
	logger.Log.SetOutput(economicsPanicWriter{})
	defer logger.Log.SetOutput(old)
	d := economicsRaw(t, 178, "S1")
	if _, err := ParsePlanDocCapped(economicsJSON(t, d), 12, 5); err == nil {
		t.Fatal("telemetry failure permitted contradiction")
	}
}
