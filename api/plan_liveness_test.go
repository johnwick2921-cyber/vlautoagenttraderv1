package api

import (
	"nofx/store"
	"path/filepath"
	"testing"
)

func TestScenarioAPIReadsDisplayedVersionOnly(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := &Server{store: st}
	const plan = "2026-09-08:NY"
	st.SetSystemConfig("scenario_status:t:"+plan, `{"S1":"invalidated"}`)
	st.SetSystemConfig(store.ScenarioStatusKey("t", plan, 1), `{"S1":"invalidated"}`)
	st.SetSystemConfig(store.ScenarioStatusKey("t", plan, 2), `{"S1":"armed"}`)
	if got := s.scenarioStatus("t", plan, 2); got["S1"] != "armed" {
		t.Fatalf("displayed v2 borrowed v1 or legacy verdict: %v", got)
	}
	if got := s.scenarioStatus("t", plan, 3); got != nil {
		t.Fatalf("unevaluated v3 borrowed a verdict: %v", got)
	}
	if got := s.scenarioMeta("t", plan, 3); got != nil {
		t.Fatalf("unevaluated v3 borrowed metadata: %v", got)
	}
}
