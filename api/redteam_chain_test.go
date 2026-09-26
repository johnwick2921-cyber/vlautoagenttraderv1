package api

// M3 red-team — the FULL chain the CTO ruled (1790231205208), at the
// PRODUCTION router NewServer builds with its real middleware (canon 53):
//
//	bot JWT → PUT /api/user/password                       → refused
//	bot JWT → /reset-account, /reset-password,
//	          /telegram config, /updates*                  → refused
//	owner token + WRONG current_password                   → refused
//	owner token + the right current_password               → ok
//	every token issued before that change (H2)             → refused everywhere
//
// Each leg asserts the SECURE outcome and that nothing was written.

import (
	"net/http"
	"testing"

	"nofx/internal/updateauth"
)

func TestRedTeamChainAtTheProductionRouter(t *testing.T) {
	e := newUpdEnv(t)
	t.Setenv("ALLOW_ACCOUNT_RESET", "1")
	bot := mustBotToken(t)
	before := e.adminRow()
	enrollment := snapshotTree(t, updateauth.Dir(e.dataDir))

	// Leg 1: the bot's JWT cannot change the owner's password — not even
	// carrying the owner's REAL current password (an LLM that was told it).
	if w := credCall(t, e, "PUT", "/api/user/password", bot, `{"current_password":"`+updAdminPass+`","new_password":"bot-chosen-pass-01"}`); w.Code != http.StatusForbidden {
		t.Fatalf("leg 1: bot JWT → PUT /api/user/password = %d %s — want 403", w.Code, w.Body.String())
	}

	// Leg 2: the bot's JWT on every other machine-denied surface.
	for _, p := range []struct{ method, path, body string }{
		{"POST", "/api/reset-account", `{"confirm":"RESET-ALL-DATA"}`},
		{"POST", "/api/reset-password", `{"email":"` + updAdminEmail + `","new_password":"bot-chosen-pass-01"}`},
		{"GET", "/api/telegram", ""},
		{"POST", "/api/telegram", `{"bot_token":"123:abc","model_id":"m"}`},
		{"POST", "/api/telegram/model", `{"model_id":"m"}`},
		{"DELETE", "/api/telegram/binding", ""},
	} {
		if w := credCall(t, e, p.method, p.path, bot, p.body); w.Code != http.StatusForbidden {
			t.Fatalf("leg 2: bot JWT → %s %s = %d %s — want 403", p.method, p.path, w.Code, w.Body.String())
		}
	}
	e.expectAllForbidden("leg 2: bot JWT → /api/updates*", withToken(bot))
	if n, _ := e.st.User().Count(); n != 1 {
		t.Fatalf("users = %d after the bot's refused reset", n)
	}

	// Leg 3: the owner's own token with a WRONG current password.
	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	if w := credCall(t, e, "PUT", "/api/user/password", owner, `{"current_password":"guessed-wrong-01","new_password":"owner-new-pass-01"}`); w.Code != http.StatusForbidden {
		t.Fatalf("leg 3: owner token + WRONG current_password → PUT /api/user/password = %d %s — want 403", w.Code, w.Body.String())
	}
	if e.adminRow() != before {
		t.Fatal("a refused leg wrote the admin row")
	}
	if !sameTree(enrollment, snapshotTree(t, updateauth.Dir(e.dataDir))) {
		t.Fatal("a refused leg changed the enrollment files")
	}
	e.expectAllAdmitted("the owner's session is untouched by every refused leg")

	// Leg 4: the owner's own token with the right current password.
	if w := credCall(t, e, "PUT", "/api/user/password", owner, `{"current_password":"`+updAdminPass+`","new_password":"owner-new-pass-01"}`); w.Code != http.StatusOK {
		t.Fatalf("leg 4: owner token + right current_password → PUT /api/user/password = %d %s — want 200", w.Code, w.Body.String())
	}
	if _, code := credLogin(t, e, updAdminEmail, "owner-new-pass-01"); code != http.StatusOK {
		t.Fatalf("leg 4: login with the new password = %d", code)
	}

	// Leg 5 (H2): every token issued before that change — the Q8-retired
	// older session, the session that made the change, the bot's — acts
	// nowhere: 401 on ordinary and bot-config routes, 403 on /updates.
	for name, tok := range map[string]string{"older session (Q8-retired)": e.tok, "the changing session": owner, "bot": bot} {
		for _, p := range []string{"/api/my-traders", "/api/config/resolved"} {
			if w := credCall(t, e, "GET", p, tok, ""); w.Code != http.StatusUnauthorized {
				t.Fatalf("leg 5: %s after the change → GET %s = %d — want 401", name, p, w.Code)
			}
		}
		e.expectAllForbidden("leg 5: "+name+" after the change → /api/updates*", withToken(tok))
	}
	if w := credCall(t, e, "DELETE", "/api/telegram/binding", e.tok, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("leg 5: the Q8-retired session → DELETE /api/telegram/binding = %d — want 401", w.Code)
	}
}
