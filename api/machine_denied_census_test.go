package api

// M3 red-team H1 — CTO ruling 1790231205208 item (2), as a CENSUS over the
// production router (canon 53): machine tokens (scope=telegram — the bot's —
// and a bot@internal token with NO scope claim, fail closed) are DENIED BY
// DEFAULT on /user/password, /reset-account, /reset-password, /telegram*
// config and /updates* — and (PR #200 fold F4a, CTO 1790252194343) on
// /api/logout: a machine token has no user session to end, and a bot that
// logs itself out blacklists its own token for the token's whole life. The
// ruled list below is written from the ruling, not
// read from production's machineDeniedRoutes, so a route the production list
// forgets is caught here; every REGISTERED route under a ruled prefix is
// probed with both machine tokens, public routes included.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ruledMachineDenied: the CTO ruling's list, verbatim (1790231205208 item 2),
// plus /api/logout (1790252194343 F4a).
var ruledMachineDenied = []string{
	"/api/user/password",
	"/api/reset-account",
	"/api/reset-password",
	"/api/telegram",
	"/api/updates",
	"/api/logout",
}

func underRuledMachineDenied(path string) bool {
	for _, p := range ruledMachineDenied {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// censusBody: a body each handler would otherwise accept.
func censusBody(method, path string) string {
	switch {
	case path == "/api/user/password":
		return `{"current_password":"` + updAdminPass + `","new_password":"machine-chosen-pass-9"}`
	case path == "/api/reset-account":
		return `{"confirm":"RESET-ALL-DATA"}`
	case path == "/api/reset-password":
		return `{"email":"` + updAdminEmail + `","new_password":"machine-chosen-pass-9"}`
	case path == "/api/telegram" && method == "POST":
		return `{"bot_token":"123:abc","model_id":"m"}`
	case path == "/api/telegram/model":
		return `{"model_id":"m"}`
	case method == "GET" || method == "DELETE":
		return ""
	}
	return `{}`
}

func concretePath(pattern string) string {
	segs := strings.Split(pattern, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "*") {
			segs[i] = "0123456789abcdef"
		}
	}
	return strings.Join(segs, "/")
}

func TestEveryRuledMachineDeniedRouteRefusesMachineTokens(t *testing.T) {
	e := newUpdEnv(t)
	t.Setenv("ALLOW_ACCOUNT_RESET", "1")
	before := e.adminRow()
	toks := map[string]string{
		"bot token (scope=telegram)":   mustBotToken(t),
		"bot@internal, no scope claim": mintMapJWT(t, ownerClaims("bot@internal")),
	}
	seen := map[string]bool{}
	for _, r := range e.s.router.Routes() {
		if !underRuledMachineDenied(r.Path) {
			continue
		}
		seen[r.Method+" "+r.Path] = true
		path, body := concretePath(r.Path), censusBody(r.Method, r.Path)
		for name, tok := range toks {
			var rec *httptest.ResponseRecorder
			if strings.HasPrefix(r.Path, "/api/updates") {
				// every OTHER factor of the updates gate is valid (e.do)
				rec = e.do(r.Method, path, body, withToken(tok))
			} else {
				rec = credCall(t, e, r.Method, path, tok, body)
			}
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s: %s %s = %d %s — want 403 (machine tokens are denied by default there)", name, r.Method, r.Path, rec.Code, rec.Body.String())
			}
		}
	}
	// The walk must have seen every ruled surface (a census that walked
	// nothing would pass vacuously).
	for _, want := range []string{
		"PUT /api/user/password", "POST /api/reset-account", "POST /api/reset-password",
		"GET /api/telegram", "POST /api/telegram", "POST /api/telegram/model", "DELETE /api/telegram/binding",
		"GET /api/updates", "POST /api/updates/install",
		"POST /api/logout",
	} {
		if !seen[want] {
			t.Fatalf("census never walked %s — is it still registered?", want)
		}
	}
	if e.adminRow() != before {
		t.Fatal("a refused machine token wrote the admin row")
	}
	if n, _ := e.st.User().Count(); n != 1 {
		t.Fatalf("users = %d after refused resets", n)
	}
	// Positive controls: a user token is unchanged on the public route (410,
	// the route stays disabled for everyone) and the bot keeps its ordinary
	// routes.
	if w := credCall(t, e, "POST", "/api/reset-password", e.tok, censusBody("POST", "/api/reset-password")); w.Code != http.StatusGone {
		t.Fatalf("positive control: the owner's token on POST /api/reset-password = %d — want the unchanged 410", w.Code)
	}
	if w := credCall(t, e, "POST", "/api/reset-password", "", censusBody("POST", "/api/reset-password")); w.Code != http.StatusGone {
		t.Fatalf("positive control: anonymous POST /api/reset-password = %d — want the unchanged 410", w.Code)
	}
	if w := credCall(t, e, "GET", "/api/my-traders", mustBotToken(t), ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: the bot token on GET /api/my-traders = %d", w.Code)
	}
}
