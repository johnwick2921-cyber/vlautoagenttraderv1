// Settings integrity (D5) — GET /api/config/resolved.
//
// The endpoint exists so the UI can render the registry's OWN labels instead of
// inventing its own. The two-label ruling is the thing under test: "no consumer"
// and "cannot take effect" are different tests and must not collapse into one
// word on the wire.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"nofx/store"

	"github.com/gin-gonic/gin"
)

func resolvedPayload(t *testing.T) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	s := &Server{}
	r := gin.New()
	r.GET("/api/config/resolved", s.handleConfigResolved)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config/resolved", nil))

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v — body %s", err, w.Body.String())
	}
	return w, body
}

// The endpoint answers, and its counts are the registry's counts — not a second
// tally that could drift from the ⚙ boot line.
func TestConfigResolvedMatchesRegistrySummary(t *testing.T) {
	w, body := resolvedPayload(t)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	sum, ok := body["summary"].(map[string]any)
	if !ok {
		t.Fatalf("no summary object in payload")
	}
	want := store.KnobStatusSummary()
	for field, expected := range map[string]int{
		"live":                 want.Live,
		"ineffective":          want.Ineffective,
		"candidate_unverified": want.Candidate,
		"suspended":            want.Suspended,
		"advisory":             want.Advisory,
		"infra":                want.Infra,
	} {
		got, present := sum[field].(float64)
		if !present {
			t.Errorf("summary.%s missing", field)
			continue
		}
		if int(got) != expected {
			t.Errorf("summary.%s = %d, registry says %d", field, int(got), expected)
		}
	}
}

// The ruling, on the wire: every knob carries the registry's own label, and the
// two statuses never render as the same sentence.
func TestConfigResolvedCarriesBothLabelsDistinctly(t *testing.T) {
	_, body := resolvedPayload(t)
	knobs, ok := body["knobs"].([]any)
	if !ok || len(knobs) == 0 {
		t.Fatalf("no knobs in payload")
	}

	labels := map[string]string{}
	for _, k := range knobs {
		e := k.(map[string]any)
		path, _ := e["path"].(string)
		status, _ := e["status"].(string)
		label, _ := e["ui_label"].(string)
		if strings.TrimSpace(label) == "" {
			t.Fatalf("%s (%s) has an empty ui_label — the UI would invent one", path, status)
		}
		if prev, seen := labels[status]; seen && prev != label && status == string(store.KnobCandidate) {
			t.Fatalf("candidate label is not stable: %q vs %q", prev, label)
		}
		labels[status] = label
	}

	ineff, hasIneff := labels[string(store.KnobIneffective)]
	cand, hasCand := labels[string(store.KnobCandidate)]
	if !hasIneff || !hasCand {
		t.Fatalf("payload must expose both labels; ineffective=%v candidate=%v", hasIneff, hasCand)
	}
	if ineff == cand {
		t.Fatalf("the two tests collapsed into one label: %q", ineff)
	}
	if !strings.Contains(cand, "pending verification") {
		t.Errorf("candidate label must read as unverified, not dead; got %q", cand)
	}
	if !strings.Contains(ineff, "does not take effect") {
		t.Errorf("ineffective label must say it cannot take effect; got %q", ineff)
	}
}

// A25 — the registry holds no values, and this endpoint must never start
// carrying them: infra knobs are ports, paths and KEYS.
func TestConfigResolvedNeverCarriesValues(t *testing.T) {
	w, _ := resolvedPayload(t)
	var probe []map[string]json.RawMessage
	full := map[string]json.RawMessage{}
	if err := json.Unmarshal(w.Body.Bytes(), &full); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := json.Unmarshal(full["knobs"], &probe); err != nil {
		t.Fatalf("unmarshal knobs: %v", err)
	}
	for _, e := range probe {
		for _, banned := range []string{"value", "effective_value", "saved_value", "env_value"} {
			if _, present := e[banned]; present {
				t.Fatalf("knob entry exposes %q — the registry carries no values and this endpoint must not add them", banned)
			}
		}
	}
}

// The endpoint enumerates strategy knobs and their consumers, so it must stay
// behind auth. routeRegistry records only the full path — both groups sit under
// /api — so the group cannot be recovered at runtime and the pin reads the
// registration site itself. A grep pin is the right instrument here: the failure
// this guards against is an edit that moves the line, and moving the line is
// exactly what the grep sees.
func TestConfigResolvedIsRegisteredProtected(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	text := string(src)

	idx := strings.Index(text, `"/config/resolved"`)
	if idx < 0 {
		t.Fatal("/config/resolved is not registered in server.go")
	}
	line := text[strings.LastIndex(text[:idx], "\n")+1:]
	line = line[:strings.Index(line, "\n")]
	if !strings.Contains(line, "s.route(protected,") {
		t.Fatalf("registered on a non-protected group — it would serve the knob registry unauthenticated:\n  %s", strings.TrimSpace(line))
	}

	// And it must sit after the protected group is opened, not merely mention it.
	if g := strings.Index(text, `protected := api.Group(`); g < 0 || g > idx {
		t.Fatal("route appears before the protected group is declared")
	}
}

// "saved → resolved · source" — the three fields R1/R2/R3 made single-sourced.
func TestResolvedFieldsRenderSavedArrowResolvedWithSource(t *testing.T) {
	// A config that exercises all three sources at once: R:R saved, plan_mode
	// absent (shipped default), htf_veto explicitly turned off.
	off := false
	cfg := &store.StrategyConfig{Regime: &store.RegimeConfig{HTFVeto: &off}}
	cfg.RiskControl.MinRiskRewardRatio = 2

	got := buildResolvedFields(cfg, "NY")
	if len(got) != 3 {
		t.Fatalf("want 3 resolved fields, got %d", len(got))
	}
	by := map[string]resolvedField{}
	for _, f := range got {
		by[f.Path] = f
	}

	rr := by["risk_control.min_risk_reward_ratio"]
	if rr.Saved != "2" || rr.Resolved != "2" || rr.Source != store.SourceSaved {
		t.Errorf("min_rr: %q → %q · %q", rr.Saved, rr.Resolved, rr.Source)
	}

	// Absent must render as "(unset)" on the SAVED side while the resolved side
	// shows what the engine will actually use. Collapsing those two is the whole
	// bug this line exists to prevent.
	pm := by["day_plan.plan_mode"]
	if pm.Saved != "(unset)" || pm.Resolved != "advisory" || pm.Source != store.SourceShippedDefault {
		t.Errorf("plan_mode: %q → %q · %q", pm.Saved, pm.Resolved, pm.Source)
	}

	veto := by["regime.htf_veto"]
	if veto.Saved != "false" || veto.Resolved != "false" || veto.Source != store.SourceSaved {
		t.Errorf("htf_veto: %q → %q · %q", veto.Saved, veto.Resolved, veto.Source)
	}

	// Every field must carry a rendered line the UI can print verbatim.
	for _, f := range got {
		if f.Line == "" || !strings.Contains(f.Line, "→") || !strings.Contains(f.Line, "·") {
			t.Errorf("%s has no rendered line: %q", f.Path, f.Line)
		}
	}
}

// An UNCOMPUTED list is absent; an empty computed one is []. Without a trader
// there is no config to resolve against, so the key must not appear at all.
func TestResolvedAbsentWithoutTraderContext(t *testing.T) {
	w, body := resolvedPayload(t)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if _, present := body["resolved"]; present {
		t.Fatal("resolved must be ABSENT without a trader — an uncomputed list is not an empty one")
	}
}

// The shipped resolver is the source of truth for the rendered value.
func TestResolvedFieldsAgreeWithProductionResolvers(t *testing.T) {
	cfg := &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanMode: "strict"}}
	fields := buildResolvedFields(cfg, "NY")
	for _, f := range fields {
		if f.Path != "day_plan.plan_mode" {
			continue
		}
		if want := cfg.DayPlan.PlanModeFor("NY"); f.Resolved != want {
			t.Fatalf("rendered %q but PlanModeFor says %q", f.Resolved, want)
		}
		if f.Source != store.SourceStrategyValue {
			t.Errorf("source = %q, want %q", f.Source, store.SourceStrategyValue)
		}
	}
}

// With a trader context the key appears, and it carries the three lines.
func TestResolvedPresentWithTraderContext(t *testing.T) {
	cfg := &store.StrategyConfig{}
	cfg.RiskControl.MinRiskRewardRatio = 2

	payload := configResolvedPayload(cfg, "NY")
	fields, present := payload["resolved"].([]resolvedField)
	if !present {
		t.Fatal("resolved missing despite a config being supplied")
	}
	if len(fields) != 3 {
		t.Fatalf("want 3 lines, got %d", len(fields))
	}
	for _, f := range fields {
		t.Logf("  %-34s %s", f.Path, f.Line)
	}

	// And the no-context path still omits it, in the same test, so the two can
	// never quietly converge.
	if _, present := configResolvedPayload(nil, "NY")["resolved"]; present {
		t.Fatal("nil cfg must omit resolved")
	}
}
