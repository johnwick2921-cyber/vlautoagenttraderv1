package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/manager"
	"nofx/store"
)

// ── W1 SETTINGS TRUTH — save → store → reload through the REAL handlers ─────
//
// The breaker (ai_config.risk_control.consecutive_loss_halt) and the strategy
// replan cap (day_plan.replan_cap) are *int: absent inherits, an explicit 0
// is 0. The Studio sends inherit as JSON null (the PUT merge keeps absent
// keys, so "absent" in a PUT can never clear a stored 0).

// storedRaw reads the persisted config bytes and the parsed config.
func storedRaw(t *testing.T, s *Server, id string) (map[string]any, *store.StrategyConfig) {
	t.Helper()
	got, err := s.store.Strategy().Get("u1", id)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(got.Config), &raw); err != nil {
		t.Fatal(err)
	}
	cfg, err := got.ParseConfig()
	if err != nil {
		t.Fatal(err)
	}
	return raw, cfg
}

// storedBreaker returns the stored breaker value, and whether the key exists.
func storedBreaker(raw map[string]any) (any, bool) {
	ac, _ := raw["ai_config"].(map[string]any)
	rc, _ := ac["risk_control"].(map[string]any)
	v, ok := rc["consecutive_loss_halt"]
	return v, ok
}

func storedReplan(raw map[string]any) (any, bool) {
	dp, _ := raw["day_plan"].(map[string]any)
	v, ok := dp["replan_cap"]
	return v, ok
}

func TestW1BreakerSaveReloadThroughHandlers(t *testing.T) {
	t.Setenv("BREAKER_HALT_N", "")
	s, r := planModeStrategyServer(t)

	// CREATE with the breaker OFF: an explicit 0 is stored, resolves OFF [O],
	// and the save records it as the owner's 0.
	rec := doStrategyRequest(r, http.MethodPost, "/api/strategies",
		`{"name":"W1","config":{"ai_config":{"risk_control":{"consecutive_loss_halt":0}}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.ID

	check := func(step string, wantKey bool, wantStored float64, wantN int, wantOrigin string) {
		t.Helper()
		raw, cfg := storedRaw(t, s, id)
		v, ok := storedBreaker(raw)
		if ok != wantKey {
			t.Fatalf("%s: key present=%v want %v (stored %v)", step, ok, wantKey, v)
		}
		if wantKey && v != wantStored {
			t.Fatalf("%s: stored %v want %v", step, v, wantStored)
		}
		n, src := store.ResolveBreakerHalt(cfg)
		if n != wantN || store.OriginLetter(src) != wantOrigin {
			t.Fatalf("%s: reload resolves %d%s (%s), want %d%s", step, n, store.OriginLetter(src), src, wantN, wantOrigin)
		}
		st := &store.Strategy{ID: id, Config: mustConfig(t, s, id)}
		if reason := s.store.Strategy().SettingsTruthRefusal(st, func(string) string { return "" }); reason != "" {
			t.Fatalf("%s: a Studio-saved value must load, got refusal %q", step, reason)
		}
	}
	check("create OFF", true, 0, 0, "[O]")
	if !s.store.Strategy().ExplicitZerosConfirmed(id)[store.KnobBreaker] {
		t.Fatal("create OFF: the save must record the explicit 0 as the owner's")
	}

	// A PUT that does not mention the breaker keeps the stored OFF.
	put := func(body string) {
		t.Helper()
		if rec := doStrategyRequest(r, http.MethodPut, "/api/strategies/"+id, body); rec.Code != http.StatusOK {
			t.Fatalf("PUT %s: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	put(`{"config":{"ai_config":{"risk_control":{"min_confidence":70}}}}`)
	check("unrelated PUT keeps OFF", true, 0, 0, "[O]")

	// ON = inherit is sent as null: the key is removed, the breaker is 8 [I].
	put(`{"config":{"ai_config":{"risk_control":{"consecutive_loss_halt":null}}}}`)
	check("null = inherit", false, 0, 8, "[I]")
	if s.store.Strategy().ExplicitZerosConfirmed(id)[store.KnobBreaker] {
		t.Fatal("inherit: the record must drop the breaker")
	}

	// An explicit 5 is stored and wins.
	put(`{"config":{"ai_config":{"risk_control":{"consecutive_loss_halt":5}}}}`)
	check("explicit 5", true, 5, 5, "[O]")

	// A legacy FLAT patch is normalised into ai_config and a 0 still lands.
	put(`{"config":{"risk_control":{"consecutive_loss_halt":0}}}`)
	check("flat patch 0", true, 0, 0, "[O]")

	// Env inherits only when the strategy saved nothing.
	put(`{"config":{"ai_config":{"risk_control":{"consecutive_loss_halt":null}}}}`)
	t.Setenv("BREAKER_HALT_N", "3")
	check("absent + env 3", false, 0, 3, "[E]")
}

func mustConfig(t *testing.T, s *Server, id string) string {
	t.Helper()
	got, err := s.store.Strategy().Get("u1", id)
	if err != nil {
		t.Fatal(err)
	}
	return got.Config
}

func TestW1ReplanCapSaveReloadThroughHandlers(t *testing.T) {
	s, r := planModeStrategyServer(t)
	rec := doStrategyRequest(r, http.MethodPost, "/api/strategies",
		`{"name":"W1R","config":{"day_plan":{"plan_enabled":true,"replan_cap":0,"sessions":[{"session":"NY","replan_cap":3}]}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.ID

	raw, cfg := storedRaw(t, s, id)
	if v, ok := storedReplan(raw); !ok || v != float64(0) {
		t.Fatalf("strategy replan_cap 0 must be STORED (omitempty used to drop it): %v %v", v, ok)
	}
	for sess, want := range map[string]int{"NY": 3, "ASIA": 0, "LONDON": 0, "": 0} {
		if n, src := store.ResolveReplanCap(cfg.DayPlan, sess); n != want {
			t.Fatalf("%q: %d (%s), want %d", sess, n, src, want)
		}
	}
	if !s.store.Strategy().ExplicitZerosConfirmed(id)[store.KnobReplanStrategy] {
		t.Fatal("the save must record the replan 0 as the owner's")
	}

	// Clear → null → inherit the shipped 2.
	if rec := doStrategyRequest(r, http.MethodPut, "/api/strategies/"+id, `{"config":{"day_plan":{"replan_cap":null}}}`); rec.Code != http.StatusOK {
		t.Fatalf("PUT: %d %s", rec.Code, rec.Body.String())
	}
	raw, cfg = storedRaw(t, s, id)
	if v, ok := storedReplan(raw); ok {
		t.Fatalf("null must remove the key, stored %v", v)
	}
	if n, src := store.ResolveReplanCap(cfg.DayPlan, "ASIA"); n != 2 || store.OriginLetter(src) != "[I]" {
		t.Fatalf("absent → %d%s, want 2[I]", n, store.OriginLetter(src))
	}
	if n, _ := store.ResolveReplanCap(cfg.DayPlan, "NY"); n != 3 {
		t.Fatalf("the NY override must survive, got %d", n)
	}
}

// ── The load seat: a legacy 0 refuses the trader; a Studio save confirms it
// and the reload brings the trader back with the value the UI showed. ───────

const w1User, w1Trader, w1Strategy = "u-w1-truth", "t-w1-truth", "s-w1-truth"

func newW1TruthServer(t *testing.T, strategyJSON string) (*Server, *manager.TraderManager, string) {
	t.Helper()
	t.Setenv("BREAKER_HALT_N", "")
	st, err := store.New(filepath.Join(t.TempDir(), "w1.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	if err := st.User().Create(&store.User{ID: w1User, Email: "w1@test", PasswordHash: "x"}); err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := st.AIModel().Create(w1User, "m-w1", "m", "deepseek", true, "sk-test-not-a-real-key", ""); err != nil {
		t.Fatalf("ai model: %v", err)
	}
	exID, err := st.Exchange().Create(w1User, "binance", "Default", true,
		"test-key", "test-secret", "", false, "", true, "", "", "", "", "", "", 0, "", "", 0)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	// Written RAW — a stored shape no W1 save produced (no record).
	if err := st.Strategy().Create(&store.Strategy{ID: w1Strategy, UserID: w1User, Name: "s", Config: strategyJSON}); err != nil {
		t.Fatalf("strategy: %v", err)
	}
	if err := st.Trader().Create(&store.Trader{ID: w1Trader, UserID: w1User, Name: "t", AIModelID: "m-w1", ExchangeID: exID, StrategyID: w1Strategy, InitialBalance: 1000}); err != nil {
		t.Fatalf("trader: %v", err)
	}
	tm := manager.NewTraderManager()
	if err := tm.LoadUserTradersFromStore(st, w1User); err != nil {
		t.Fatalf("load traders: %v", err)
	}
	auth.SetJWTSecret("w1-settings-truth-test-secret")
	tok, err := auth.GenerateJWT(w1User, "w1@test")
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return NewServer(tm, st, nil, "127.0.0.1", 0), tm, tok
}

func TestW1LegacyZeroRefusesTheTraderUntilAStudioSave(t *testing.T) {
	s, tm, tok := newW1TruthServer(t, `{"ai_config":{"risk_control":{"consecutive_loss_halt":0}},"day_plan":{"plan_enabled":true,"replan_cap":0}}`)

	// The load seat (addTraderFromStore via LoadUserTradersFromStore) refused it.
	if _, err := tm.GetTrader(w1Trader); err == nil {
		t.Fatal("a trader bound to an UNCONFIRMED stored 0 must be refused at load — it meant 'inherit' before W1")
	}

	// The Studio shows OFF / 0 and the owner saves: the handler records the
	// zeros, then reloads the trader, which now loads with what the UI showed.
	rec, _ := olDo(t, s, tok, http.MethodPut, "/api/strategies/"+w1Strategy,
		`{"config":{"ai_config":{"risk_control":{"consecutive_loss_halt":0}},"day_plan":{"plan_enabled":true,"replan_cap":0}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT: %d %s", rec.Code, rec.Body.String())
	}
	at, err := tm.GetTrader(w1Trader)
	if err != nil {
		t.Fatalf("after the Studio save the trader must reload: %v", err)
	}
	cfg := at.GetStrategyConfig()
	if n, src := store.ResolveBreakerHalt(cfg); n != 0 || store.OriginLetter(src) != "[O]" {
		t.Fatalf("runtime breaker %d%s, want off[O] — the value the Studio showed", n, store.OriginLetter(src))
	}
	// The card's cap seat reads the same 0.
	if _, _, left, cap := s.planRulesWithCap(w1Trader, "ASIA", "2026-09-22"); cap != 0 || left != 0 {
		t.Fatalf("planRulesWithCap: cap=%d left=%d, want 0/0 (the saved strategy 0 used to read 2)", cap, left)
	}
}
