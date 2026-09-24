package api

// M3 red-team H1/H2 — the credential routes (PUT /api/user/password,
// POST /api/reset-account) at the PRODUCTION call site: the router NewServer
// builds with the real middleware (canon 53), via the updEnv harness.
//
// H1 (red1 R1 / red2 #1): the Telegram bot's JWT carries the OWNER's user_id
// with email bot@internal. PUT /api/user/password asked for nothing but
// new_password and acted on the JWT's user_id, so the bot (an LLM steerable by
// prompt injection or by whoever first /start-s an unbound bot) could set the
// owner's password, log in as the owner, pass every M3 identity check, and
// lock the owner out. Every pin here asserts the SECURE behaviour.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/telegram/agent"
)

// credCall drives one ordinary (non-/updates) request through the production
// router from a loopback peer, exactly as the web UI's fetch or the agent's
// api_request tool would send it (no update header).
func credCall(t *testing.T, e *updEnv, method, path, tok, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	return w
}

// credLogin: POST /api/login through the production router; "" + the status
// when login is refused.
func credLogin(t *testing.T, e *updEnv, email, pass string) (string, int) {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"email": email, "password": pass})
	w := credCall(t, e, "POST", "/api/login", "", string(b))
	if w.Code != http.StatusOK {
		return "", w.Code
	}
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Token == "" {
		t.Fatal("login 200 without a token")
	}
	return out.Token, w.Code
}

type credRow struct {
	hash    string
	updated time.Time
}

func (e *updEnv) adminRow() credRow {
	e.t.Helper()
	u, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		e.t.Fatalf("admin row: %v", err)
	}
	return credRow{hash: u.PasswordHash, updated: u.UpdatedAt}
}

func mustBotToken(t *testing.T) string {
	t.Helper()
	bot, err := agent.GenerateBotToken(updAdminID) // exactly what telegram/bot.go mints
	if err != nil {
		t.Fatal(err)
	}
	return bot
}

// H1: the bot token cannot set the owner's password; the row is untouched and
// the password the bot chose opens nothing.
func TestBotTokenCannotChangeTheOwnersPassword(t *testing.T) {
	e := newUpdEnv(t)
	bot := mustBotToken(t)
	before := e.adminRow()

	w := credCall(t, e, "PUT", "/api/user/password", bot, `{"new_password":"bot-chosen-password-1"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("PUT /api/user/password with the Telegram bot token = %d %s — want 403", w.Code, w.Body.String())
	}
	if after := e.adminRow(); after != before {
		t.Fatal("a refused password change still wrote the admin row")
	}
	if _, code := credLogin(t, e, updAdminEmail, "bot-chosen-password-1"); code != http.StatusUnauthorized {
		t.Fatalf("login with the bot's password = %d, want 401", code)
	}
	if _, code := credLogin(t, e, updAdminEmail, updAdminPass); code != http.StatusOK {
		t.Fatalf("positive control: the owner's own password no longer logs in (%d)", code)
	}
	// Same user_id, any email that is not the row's (not only bot@internal).
	for _, email := range []string{"someone@example.test", "OWNER@example.test", "owner@example.test ", ""} {
		tok := mintJWT(t, updAdminID, email, time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
		if w := credCall(t, e, "PUT", "/api/user/password", tok, `{"new_password":"look-alike-pass-1"}`); w.Code != http.StatusForbidden {
			t.Fatalf("owner id with email %q: PUT /api/user/password = %d — want 403", email, w.Code)
		}
	}
	if after := e.adminRow(); after != before {
		t.Fatal("a look-alike token wrote the admin row")
	}
}

// H1: with the operator's reset flag ON, the bot token still cannot wipe the
// installation's users (the flag is the owner's opt-in, not the bot's).
func TestBotTokenCannotResetTheAccount(t *testing.T) {
	e := newUpdEnv(t)
	t.Setenv("ALLOW_ACCOUNT_RESET", "1")
	bot := mustBotToken(t)
	before := snapshotTree(t, updateauth.Dir(e.dataDir))

	w := credCall(t, e, "POST", "/api/reset-account", bot, `{"confirm":"RESET-ALL-DATA"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST /api/reset-account with the Telegram bot token = %d %s — want 403", w.Code, w.Body.String())
	}
	if n, _ := e.st.User().Count(); n != 1 {
		t.Fatalf("users after a refused reset = %d, want 1", n)
	}
	after := snapshotTree(t, updateauth.Dir(e.dataDir))
	for _, p := range []string{updateauth.AdminPath(e.dataDir), updateauth.DeviceKeyPath(e.dataDir)} {
		if before[p] == "" || before[p] != after[p] {
			t.Fatalf("a refused reset changed the enrollment")
		}
	}
	e.expectAllAdmitted("admin after the refused bot reset")
}

// H1, the red teams' full chain (red1 R1, red2 #1): bot token → PUT
// /api/user/password → POST /api/login → /api/updates. Every link that could
// hand the bot the owner's identity ends in a refusal.
func TestBotToAdminChainEndsForbidden(t *testing.T) {
	e := newUpdEnv(t)
	bot := mustBotToken(t)
	e.expectAllForbidden("bot token, direct", withToken(bot))

	if w := credCall(t, e, "PUT", "/api/user/password", bot, `{"new_password":"bot-chosen-password-2"}`); w.Code != http.StatusForbidden {
		t.Fatalf("chain step 1: PUT /api/user/password with the bot token = %d %s — want 403", w.Code, w.Body.String())
	}
	if tok, code := credLogin(t, e, updAdminEmail, "bot-chosen-password-2"); code != http.StatusUnauthorized {
		e.expectAllForbidden("token minted from the bot's password", withToken(tok))
		t.Fatalf("chain step 2: login with the bot's password = %d — want 401", code)
	}
	e.expectAllAdmitted("the owner's own token is untouched by the attempt")
}

// Positive control: the owner's OWN password change through the web flow —
// login, then PUT {new_password} only (web/src/pages/SettingsPage.tsx sends
// nothing else today; requiring current_password is a UI change for the CTO
// to rule) — still works, and the new password is the one that logs in.
func TestOwnerWebPasswordChangeStillWorks(t *testing.T) {
	e := newUpdEnv(t)
	tok, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	w := credCall(t, e, "PUT", "/api/user/password", tok, `{"new_password":"owner-new-password-1"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("the owner's own password change = %d %s — want 200", w.Code, w.Body.String())
	}
	if _, code := credLogin(t, e, updAdminEmail, "owner-new-password-1"); code != http.StatusOK {
		t.Fatalf("login with the new password = %d, want 200", code)
	}
	if _, code := credLogin(t, e, updAdminEmail, updAdminPass); code != http.StatusUnauthorized {
		t.Fatalf("login with the old password = %d, want 401", code)
	}
}
