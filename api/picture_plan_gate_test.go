package api

import (
	"net/http"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/manager"
	"nofx/store"
	"nofx/trader"
)

// ── W-EXEC-TRUTH W0 (CTO Q6) — the plan card says when strict refuses Picture ─
//
// The card's payload carries the trader's own READ of Picture's plan-mode
// verdict (the same read pictureEntryGate refuses on and the 📷 boot line
// prints), on the no-plan-yet payload as well as a plan's.

const ppgUser, ppgTrader = "u-picture-gate", "t-picture-gate"

func newPicturePlanGateServer(t *testing.T, strategyJSON string) (*Server, string) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "ppg.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	if err := st.User().Create(&store.User{ID: ppgUser, Email: "ppg@test", PasswordHash: "x"}); err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := st.AIModel().Create(ppgUser, "m-ppg", "m", "deepseek", true, "sk-test-not-a-real-key", ""); err != nil {
		t.Fatalf("ai model: %v", err)
	}
	exID, err := st.Exchange().Create(ppgUser, "binance", "Default", true,
		"test-key", "test-secret", "", false, "", true, "", "", "", "", "", "", 0, "", "", 0)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{ID: "s-ppg", UserID: ppgUser, Name: "s", Config: strategyJSON}); err != nil {
		t.Fatalf("strategy: %v", err)
	}
	if err := st.Trader().Create(&store.Trader{ID: ppgTrader, UserID: ppgUser, Name: "t", AIModelID: "m-ppg", ExchangeID: exID, StrategyID: "s-ppg", InitialBalance: 1000}); err != nil {
		t.Fatalf("trader: %v", err)
	}
	tm := manager.NewTraderManager()
	if err := tm.LoadUserTradersFromStore(st, ppgUser); err != nil {
		t.Fatalf("load traders: %v", err)
	}
	auth.SetJWTSecret("picture-plan-gate-test-secret")
	tok, err := auth.GenerateJWT(ppgUser, "ppg@test")
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return NewServer(tm, st, nil, "127.0.0.1", 0), tok
}

func picturePayload(t *testing.T, strategyJSON string) map[string]any {
	t.Helper()
	s, tok := newPicturePlanGateServer(t, strategyJSON)
	rec, body := olDo(t, s, tok, http.MethodGet, "/api/plan/today?trader_id="+ppgTrader, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("plan/today: %d %s", rec.Code, rec.Body.String())
	}
	pic, ok := body["picture"].(map[string]any)
	if !ok {
		t.Fatalf("the card payload must carry the picture read, got %#v (body %s)", body["picture"], rec.Body.String())
	}
	return pic
}

func TestPlanCardCarriesPictureStrictRefusal(t *testing.T) {
	pic := picturePayload(t, `{"day_plan":{"plan_enabled":true,"plan_mode":"strict"}}`)
	if pic["refusal"] != trader.PictureStrictRefusal {
		t.Fatalf("under strict the card must carry the refusal text %q, got %#v", trader.PictureStrictRefusal, pic)
	}
}

func TestPlanCardPictureAdmittedOutsideStrict(t *testing.T) {
	pic := picturePayload(t, `{"day_plan":{"plan_enabled":true,"plan_mode":"advisory"}}`)
	if pic["refusal"] != "" {
		t.Fatalf("outside strict the card carries no refusal, got %#v", pic)
	}
	if pic["enabled"] != false {
		t.Fatalf("a non-NT8 trader never runs Picture, got %#v", pic)
	}
}
