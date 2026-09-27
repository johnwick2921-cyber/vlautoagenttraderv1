package api

// P2-12 (audit 0926-system): POST /api/telegram/bind-code issues the one-time
// Telegram bind code — owner-only, through the production router.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func telegramBindCall(t *testing.T, e *updEnv, tok string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", "/api/telegram/bind-code", strings.NewReader("{}"))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	return w
}

func TestTelegramBindCodeOwnerOnly(t *testing.T) {
	e := newUpdEnv(t)
	secondTok := seedSecondUser(t, e)

	if w := telegramBindCall(t, e, secondTok); w.Code != http.StatusForbidden {
		t.Fatalf("non-owner bind-code: want 403, got %d (%s)", w.Code, w.Body.String())
	}
	botTok := mintJWT(t, updAdminID, "bot@internal",
		time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
	if w := telegramBindCall(t, e, botTok); w.Code != http.StatusForbidden {
		t.Fatalf("machine-token bind-code: want 403, got %d", w.Code)
	}
}

func TestTelegramBindCodeIssuedAndConsumable(t *testing.T) {
	e := newUpdEnv(t)
	w := telegramBindCall(t, e, e.tok)
	if w.Code != http.StatusOK {
		t.Fatalf("owner bind-code: want 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Code              string `json:"code"`
		ExpiresInMinutes  int    `json:"expires_in_minutes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad response shape: %v (%s)", err, w.Body.String())
	}
	if len(resp.Code) != telegramBindCodeLen {
		t.Fatalf("want %d-char code, got %q", telegramBindCodeLen, resp.Code)
	}
	if resp.ExpiresInMinutes != int(telegramBindCodeTTL.Minutes()) {
		t.Fatalf("want %d minutes, got %d", int(telegramBindCodeTTL.Minutes()), resp.ExpiresInMinutes)
	}
	ok, err := e.st.TelegramConfig().ConsumeBindCode(resp.Code)
	if err != nil || !ok {
		t.Fatalf("issued code must consume from the store: %v/%v", ok, err)
	}
}
