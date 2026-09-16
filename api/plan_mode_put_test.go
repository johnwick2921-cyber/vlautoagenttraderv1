package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"

	"github.com/gin-gonic/gin"
)

// ADDENDUM S (2026-08-26, chat-Q test gap) — plan_mode (global + per-session)
// survives BOTH the create and edit PUT handlers. Previously the persistence
// was pinned only at the store layer (save→reload round-trip); this test walks
// the actual handlers: create POST persists the whole day_plan block, edit PUT
// preserves unmentioned fields and persists sent ones. No behavior change —
// pure test.

func strPtr(s string) *string { return &s }

func planModeStrategyServer(t *testing.T) (*Server, *gin.Engine) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "planmode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := &Server{store: st}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "u1"); c.Next() })
	r.POST("/api/strategies", s.handleCreateStrategy)
	r.PUT("/api/strategies/:id", s.handleUpdateStrategy)
	return s, r
}

func doStrategyRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestPlanModeSurvivesCreateAndEditPut(t *testing.T) {
	s, r := planModeStrategyServer(t)

	createBody := `{
		"name": "MNQ",
		"config": {
			"day_plan": {
				"plan_enabled": true,
				"plan_mode": "strict",
				"proximity_filter_atr": 0.3,
				"sessions": [
					{"session": "NY", "plan_mode": "direction"},
					{"session": "ASIA", "plan_mode": "advisory"}
				]
			}
		}
	}`
	rec := doStrategyRequest(r, http.MethodPost, "/api/strategies", createBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("create PUT: %d body=%s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("create response: %v %s", err, rec.Body.String())
	}

	read := func() *store.DayPlanConfig {
		got, err := s.store.Strategy().Get("u1", created.ID)
		if err != nil || got == nil {
			t.Fatalf("read strategy: %v", err)
		}
		var cfg store.StrategyConfig
		if err := json.Unmarshal([]byte(got.Config), &cfg); err != nil {
			t.Fatalf("config unmarshal: %v", err)
		}
		return cfg.DayPlan
	}

	// 1. CREATE persisted the global + per-session modes verbatim.
	dp := read()
	if dp == nil || dp.PlanMode != "strict" {
		t.Fatalf("create: global plan_mode = %v, want strict", dp)
	}
	if got := dp.PlanModeFor("NY"); got != "direction" {
		t.Fatalf("create: NY plan_mode = %q, want direction", got)
	}
	if got := dp.PlanModeFor("ASIA"); got != "advisory" {
		t.Fatalf("create: ASIA plan_mode = %q, want advisory", got)
	}
	if got := dp.PlanModeFor("LONDON"); got != "strict" {
		t.Fatalf("create: LONDON must inherit the global strict, got %q", got)
	}

	// 2. EDIT touching a DIFFERENT field must preserve every plan_mode.
	editBody := `{"config": {"day_plan": {"scenario_cap": 4}}}`
	rec = doStrategyRequest(r, http.MethodPut, "/api/strategies/"+created.ID, editBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit PUT: %d body=%s", rec.Code, rec.Body.String())
	}
	dp = read()
	if dp.PlanMode != "strict" || dp.PlanModeFor("NY") != "direction" || dp.PlanModeFor("ASIA") != "advisory" {
		t.Fatalf("edit (unmentioned) must preserve plan_mode: global=%q NY=%q ASIA=%q",
			dp.PlanMode, dp.PlanModeFor("NY"), dp.PlanModeFor("ASIA"))
	}

	// 3. EDIT overwriting plan_mode persists the new values.
	editBody2 := `{"config": {"day_plan": {"plan_mode": "advisory", "sessions": [
		{"session": "NY", "plan_mode": "strict"}
	]}}}`
	rec = doStrategyRequest(r, http.MethodPut, "/api/strategies/"+created.ID, editBody2)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit PUT 2: %d body=%s", rec.Code, rec.Body.String())
	}
	dp = read()
	if dp.PlanMode != "advisory" || dp.PlanModeFor("NY") != "strict" {
		t.Fatalf("edit (overwrite) must persist: global=%q NY=%q", dp.PlanMode, dp.PlanModeFor("NY"))
	}
}
