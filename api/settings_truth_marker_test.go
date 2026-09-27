package api

// CTO ruling msg 1790176346377 (W1 marker) at the PRODUCTION call sites: the
// real router (NewServer → setupRoutes), the real JWT + ownership path, a real
// sqlite store.
//
//   R1 — the Studio's save writes the strategy row and its explicit-zero record
//        in ONE transaction: when the record cannot be written, the row is not
//        written either and the handler answers an error.
//   R2 — the effective endpoint's origin for an explicit 0 says which applies:
//        "OFF — confirmed by Studio save <time CT>" or "explicit 0 UNCONFIRMED
//        — re-save in Studio".

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"nofx/store"
)

// failStudioRecordWrites makes every write of a settings_truth_zero:* row fail
// INSIDE SQLite (triggers — no seam in production code), so the handlers run
// exactly as they do live and only the record's write is refused.
func failStudioRecordWrites(t *testing.T, st *store.Store) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TRIGGER t_fail_zero_ins BEFORE INSERT ON system_config WHEN NEW.key LIKE 'settings_truth_zero:%' BEGIN SELECT RAISE(ABORT, 'test: record write refused'); END`,
		`CREATE TRIGGER t_fail_zero_upd BEFORE UPDATE ON system_config WHEN NEW.key LIKE 'settings_truth_zero:%' BEGIN SELECT RAISE(ABORT, 'test: record write refused'); END`,
		`CREATE TRIGGER t_fail_zero_del BEFORE DELETE ON system_config WHEN OLD.key LIKE 'settings_truth_zero:%' BEGIN SELECT RAISE(ABORT, 'test: record write refused'); END`,
	} {
		if err := st.GormDB().Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
}

// R1: PUT and POST through the real router. The record's write fails → the
// save answers 500 and the strategies table is exactly as it was.
func TestStudioSaveRowAndRecordAreOneTransaction(t *testing.T) {
	s, st, tok := newEffectiveServer(t)
	const id = "w1-marker-rb"
	effPutStrategy(t, st, id, effUser, `{"strategy_type":"ai_trading","ai_config":{"risk_control":{"min_confidence":70}}}`)
	before, err := st.Strategy().Get(effUser, id)
	if err != nil {
		t.Fatal(err)
	}
	failStudioRecordWrites(t, st)

	// PUT — the Studio turns the breaker OFF: an explicit 0 whose record fails.
	rec := effDo(t, s, tok, http.MethodPut, "/api/strategies/"+id,
		`{"name":"renamed","config":{"ai_config":{"risk_control":{"consecutive_loss_halt":0}}}}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("PUT with a failing record: %d %s — want 500", rec.Code, rec.Body.String())
	}
	after, err := st.Strategy().Get(effUser, id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Config != before.Config || after.Name != before.Name || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("PUT persisted the row WITHOUT its record:\n before %q %s\n after  %q %s", before.Name, before.Config, after.Name, after.Config)
	}
	if _, ok := storedBreakerOf(t, after.Config); ok {
		t.Fatal("the breaker 0 reached the stored row although its record failed")
	}

	// POST — a new strategy with the breaker OFF: no row at all.
	rec = effDo(t, s, tok, http.MethodPost, "/api/strategies",
		`{"name":"w1-marker-new","config":{"ai_config":{"risk_control":{"consecutive_loss_halt":0}}}}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("POST with a failing record: %d %s — want 500", rec.Code, rec.Body.String())
	}
	list, err := st.Strategy().List(effUser)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range list {
		if row.Name == "w1-marker-new" {
			t.Fatalf("POST created strategy %s WITHOUT its record", row.ID)
		}
	}
}

func storedBreakerOf(t *testing.T, config string) (any, bool) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		t.Fatal(err)
	}
	return storedBreaker(raw)
}

// R2: one config, two states. Written raw (no Studio save, no record) every
// strategy-level explicit 0 reads UNCONFIRMED; saved through POST
// /api/strategies it reads confirmed, with the save's time in CT.
func TestEffectiveExplicitZeroSaysConfirmedOrUnconfirmed(t *testing.T) {
	t.Setenv("BREAKER_HALT_N", "")
	s, st, tok := newEffectiveServer(t)
	const cfg = `{"strategy_type":"ai_trading","ai_config":{"risk_control":{"consecutive_loss_halt":0}},` +
		`"day_plan":{"plan_enabled":true,"replan_cap":0,"sessions":[{"session":"NY","replan_cap":3}]}}`
	const (
		breakerPath    = "ai_config.risk_control.consecutive_loss_halt"
		replanPath     = "day_plan.replan_cap"
		replanSessPath = "day_plan.sessions.replan_cap"
		unconfirmed    = "explicit 0 UNCONFIRMED — re-save in Studio"
	)
	check := func(label, id, session, path string, wantEff float64, wantOrigin string) {
		t.Helper()
		url := "/api/strategies/" + id + "/effective"
		if session != "" {
			url += "?session=" + session
		}
		row := effRow(t, effGet(t, s, tok, url), path)
		if num(t, row.Effective) != wantEff || row.Origin != wantOrigin {
			t.Fatalf("%s %s @%q: effective %v origin %q — want %v / %q", label, path, session, row.Effective, row.Origin, wantEff, wantOrigin)
		}
	}

	// UNCONFIRMED.
	effPutStrategy(t, st, "w1-raw-zero", effUser, cfg)
	check("raw", "w1-raw-zero", "", breakerPath, 0, store.SourceSaved+" — "+unconfirmed)
	check("raw", "w1-raw-zero", "", replanPath, 0, store.SourceStrategyValue+" — "+unconfirmed)
	check("raw", "w1-raw-zero", "ASIA", replanSessPath, 0, store.SourceStrategyValue+" — "+unconfirmed)
	check("raw", "w1-raw-zero", "NY", replanSessPath, 3, store.SourceSessionOverride) // a session 0/3 is not the strategy 0

	// CONFIRMED — the Studio's own create.
	t0 := time.Now().UTC().Add(-time.Second)
	rec := effDo(t, s, tok, http.MethodPost, "/api/strategies", `{"name":"w1-conf-zero","config":`+cfg+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	t1 := time.Now().UTC().Add(time.Second)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	saved := st.Strategy().ExplicitZeroRecordOf(created.ID).SavedAt
	if saved.Before(t0) || saved.After(t1) {
		t.Fatalf("saved_at %v is not the create's time [%v, %v]", saved, t0, t1)
	}
	// The expected time is rendered HERE, independently: America/Chicago,
	// minute resolution, " CT" — the kernel.FormatCT convention.
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skipf("no zone database: %v", err)
	}
	at := saved.In(chicago).Format("2006-01-02 15:04") + " CT"
	check("studio", created.ID, "", breakerPath, 0, store.SourceSaved+" — OFF — confirmed by Studio save "+at)
	check("studio", created.ID, "", replanPath, 0, store.SourceStrategyValue+" — 0 — confirmed by Studio save "+at)
	check("studio", created.ID, "LONDON", replanSessPath, 0, store.SourceStrategyValue+" — 0 — confirmed by Studio save "+at)

	// A PUT back to inherit drops the record: the explicit 0 is gone and so is
	// the verdict — no stale "confirmed" on a value that is not a 0.
	if rec := effDo(t, s, tok, http.MethodPut, "/api/strategies/"+created.ID,
		`{"config":{"ai_config":{"risk_control":{"consecutive_loss_halt":null}}}}`); rec.Code != http.StatusOK {
		t.Fatalf("PUT inherit: %d %s", rec.Code, rec.Body.String())
	}
	check("inherit", created.ID, "", breakerPath, 8, store.SourceShippedDefault)
	check("inherit", created.ID, "", replanPath, 0, store.SourceStrategyValue+" — 0 — confirmed by Studio save "+
		st.Strategy().ExplicitZeroRecordOf(created.ID).SavedAt.In(chicago).Format("2006-01-02 15:04")+" CT")
}
