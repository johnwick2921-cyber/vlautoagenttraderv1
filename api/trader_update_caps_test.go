package api

// P2 (CTO 10-01, C1): editing a trader via PUT /api/traders/:id must PRESERVE
// the stored BTCETHLeverage/AltcoinLeverage columns — live futures risk caps
// in crypto-named storage (P0 CAPS ruling) — instead of writing 0 over them.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vl/store"
)

func traderUpdateCall(t *testing.T, e *updEnv, tok, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("PUT", "/api/traders/"+id, strings.NewReader(body))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	return w
}

func TestTraderUpdatePreservesStoredRiskCaps(t *testing.T) {
	e := newUpdEnv(t)

	if err := e.st.AIModel().UpdateWithName(updAdminID, "default_deepseek", "DeepSeek", true, "sk-test", "", "deepseek-chat"); err != nil {
		t.Fatalf("seed model: %v", err)
	}
	exID, err := e.st.Exchange().Create(updAdminID, "ninjatrader", "Main", true, "", "", "", false, "/tmp/nt-fixture", "MNQ", 1)
	if err != nil {
		t.Fatalf("seed exchange: %v", err)
	}
	cfg := store.GetDefaultStrategyConfig("zh")
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	if err := e.st.Strategy().Create(&store.Strategy{ID: "s1", UserID: updAdminID, Name: "S", Config: string(rawCfg)}); err != nil {
		t.Fatalf("seed strategy: %v", err)
	}
	tr := &store.Trader{
		ID:                  "t1",
		UserID:              updAdminID,
		Name:                "T",
		AIModelID:           "default_deepseek",
		ExchangeID:          exID,
		StrategyID:          "s1",
		InitialBalance:      1000,
		BTCETHLeverage:      7,
		AltcoinLeverage:     3,
		ScanIntervalMinutes: 3,
		IsCrossMargin:       true,
		ShowInCompetition:   true,
	}
	if err := e.st.Trader().Create(tr); err != nil {
		t.Fatalf("seed trader: %v", err)
	}

	body := `{"name":"T","ai_model_id":"default_deepseek","exchange_id":"` + exID + `","scan_interval_minutes":5}`
	if w := traderUpdateCall(t, e, e.tok, "t1", body); w.Code != http.StatusOK {
		t.Fatalf("update: want 200, got %d (%s)", w.Code, w.Body.String())
	}

	got, err := e.st.Trader().GetFullConfig(updAdminID, "t1")
	if err != nil || got == nil || got.Trader == nil {
		t.Fatalf("reload trader: %v", err)
	}
	if got.Trader.BTCETHLeverage != 7 || got.Trader.AltcoinLeverage != 3 {
		t.Fatalf("stored caps were overwritten by the update: btc=%d alt=%d, want 7/3",
			got.Trader.BTCETHLeverage, got.Trader.AltcoinLeverage)
	}
}
