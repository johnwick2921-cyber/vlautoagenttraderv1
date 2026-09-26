package telegram

// PR #200 review F4b (the bot half; CTO bridge msg 1790252194343): the bot's
// agent calls the API with its own token through apicall, which has no
// allowlist — so one prompt-injected "log out" puts the bot's token on the
// logout blacklist (auth.BlacklistToken — the ONE write handleLogout makes,
// api/handler_user.go). authMiddleware refuses a blacklisted token on every
// route, but botTokenStale never consulted the blacklist: refresh kept the
// token, and every later message 401'd until a restart or the token's 24 h
// expiry. The bot now re-mints instead of going dark.
//
// Pinned where runBot reaches it — botIdentity.refresh (runBot calls it at
// start, on /start and before every AI call; TestRunBotMintsOnlyThroughRefresh)
// — against the PRODUCTION server (api.NewServer + Server.Start, real
// authMiddleware). The API half (POST /api/logout machine-denied) is another
// fold's; this pin blacklists through auth.BlacklistToken directly, so it
// holds whichever route can still write the blacklist.

import (
	"net/http"
	"testing"
	"time"

	"nofx/auth"
)

// btPrivateSecret gives the test its own JWT secret (btBoot's cleanup restores
// the one before btBoot). The logout blacklist is a process-wide map keyed by
// the exact token string, and a bot token minted for the same user in the
// same second under the same secret is byte-identical — so a token this test
// blacklists must never be a string another test of the package can mint.
func btPrivateSecret(t *testing.T) {
	t.Helper()
	auth.SetJWTSecret(btSecret + "-" + t.Name())
}

func TestBotRefreshReMintsAfterItsTokenIsBlacklistedAtItsCallSite(t *testing.T) {
	base, st := btBoot(t)
	btPrivateSecret(t)
	ident := newBotIdentity(st, 0)
	if !ident.refresh() {
		t.Fatal("refresh with an account on the box = false")
	}
	first, firstAgents := ident.token, ident.agents
	if ident.userID != btOwnerID || first == "" || firstAgents == nil {
		t.Fatalf("refresh resolved user=%q token-set=%v agents-set=%v", ident.userID, first != "", firstAgents != nil)
	}
	if code, _ := btCall(t, base, "GET", "/api/my-traders", first, ""); code != http.StatusOK {
		t.Fatalf("control: the bot's first token on GET /api/my-traders = %d", code)
	}

	// The prompt-injected "log out": the bot's own token on the blacklist,
	// exactly as handleLogout writes it (its exp).
	cl, err := auth.ValidateJWT(first)
	if err != nil || cl.ExpiresAt == nil {
		t.Fatalf("the bot's token does not parse to an exp: %v", err)
	}
	auth.BlacklistToken(first, cl.ExpiresAt.Time)
	if code, _ := btCall(t, base, "GET", "/api/my-traders", first, ""); code != http.StatusUnauthorized {
		t.Fatalf("control: the blacklisted bot token on GET /api/my-traders = %d — want 401", code)
	}
	if !botTokenStale(first, btRow(t, st)) {
		t.Error("botTokenStale = false for the bot's blacklisted token — the API refuses it on every route")
	}

	// The bot's next message, past the first mint's second: a re-mint in
	// that same second is byte-identical to the blacklisted token (the
	// bounded same-second re-mint, F7, covers that case).
	now := time.Now()
	time.Sleep(now.Truncate(time.Second).Add(time.Second + 20*time.Millisecond).Sub(now))
	if !ident.refresh() {
		t.Fatal("refresh after the bot's token was blacklisted = false")
	}
	if ident.token == first {
		t.Fatal("refresh kept the blacklisted token — every agent call 401s until a restart or the token's expiry")
	}
	if ident.agents == firstAgents {
		t.Fatal("refresh re-minted but kept the agent manager built on the blacklisted token")
	}
	if code, body := btCall(t, base, "GET", "/api/my-traders", ident.token, ""); code != http.StatusOK {
		t.Fatalf("the bot's token after refresh on GET /api/my-traders = %d %s — want 200", code, body)
	}
}
