package api

// PR #200 fold F4a (CTO 1790252194343, token-retire lens — the API half):
// POST /api/logout was reachable by a MACHINE token. The Telegram agent's
// api_request tool has no path allowlist, so the bot (an LLM steerable by
// prompt injection) could call /api/logout with its OWN token; the handler
// blacklists that exact string until its exp, and every later bot call is
// 401 for the rest of the token's life. A machine token has no user session
// to end: /api/logout is machine-denied now (credential_guard.go
// machineDeniedRoutes), refused 403 in authMiddleware BEFORE the handler, so
// the token is not blacklisted. The owner's own logout is unchanged. Driven
// at the PRODUCTION router (canon 53). (The bot half — botTokenStale
// consulting the blacklist — is another builder's.)

import (
	"net/http"
	"testing"
	"time"

	"nofx/auth"
)

func TestMachineTokenCannotLogOutAndIsNotBlacklisted(t *testing.T) {
	e := newUpdEnv(t)
	gate, err := auth.GenerateScopedJWT(updAdminID, updAdminEmail, auth.ScopeGateJWT) // what cmd/gate-jwt mints
	if err != nil {
		t.Fatal(err)
	}
	for name, tok := range map[string]string{
		"bot token (scope=telegram)":         mustBotToken(t),
		"bot@internal, no scope claim":       mintMapJWT(t, ownerClaims("bot@internal")),
		"gate-jwt token (owner email+scope)": gate,
	} {
		w := credCall(t, e, "POST", "/api/logout", tok, "")
		if w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
			t.Fatalf("%s: POST /api/logout = %d %s — want 403 %s (a machine token has no user session to end)", name, w.Code, w.Body.String(), forbiddenBody)
		}
		if auth.IsTokenBlacklisted(tok) {
			t.Fatalf("%s: a REFUSED logout still blacklisted the token", name)
		}
		// The token keeps working where machine tokens belong.
		if w := credCall(t, e, "GET", "/api/my-traders", tok, ""); w.Code != http.StatusOK {
			t.Fatalf("%s: GET /api/my-traders after the refused logout = %d %s — want 200 (the token must survive)", name, w.Code, w.Body.String())
		}
	}

	// The owner still logs out: 200, blacklisted, refused afterwards. A
	// distinctive exp keeps this string unique in the process-wide blacklist.
	now := time.Now()
	owner := mintJWT(t, updAdminID, updAdminEmail, now.Add(-4*time.Second), now.Add(time.Hour+47*time.Second), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", owner, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: the owner token before logout = %d %s", w.Code, w.Body.String())
	}
	if w := credCall(t, e, "POST", "/api/logout", owner, ""); w.Code != http.StatusOK {
		t.Fatalf("the owner's POST /api/logout = %d %s — want 200", w.Code, w.Body.String())
	}
	if !auth.IsTokenBlacklisted(owner) {
		t.Fatal("the owner's logout did not blacklist the owner token")
	}
	if w := credCall(t, e, "GET", "/api/my-traders", owner, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("the owner token after logout = %d — want 401", w.Code)
	}
}
