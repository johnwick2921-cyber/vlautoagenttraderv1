package api

// M3 red-team H2 — CTO ruling 1790231205208: authMiddleware (GLOBAL, not only
// the credential routes) refuses a token issued at or before the account's
// retire epoch — the credential-change instant (users.updated_at, written only
// by UpdatePassword) — with the SAME whole-second rule as the /updates Q8
// gate: iat must be STRICTLY after updated_at truncated to the second.
//
// red1 R2 / the fh verifier's reproduction: after the owner's rotation, a
// retired stolen token still got GET /api/my-traders = 200, DELETE
// /api/telegram/binding = 200 and POST /api/telegram = 200 — it could unbind
// the bot or swap in its own bot token for up to 24 h. Driven at the
// PRODUCTION router (canon 53); "retired" here means a Q8-retired token.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"nofx/auth"
	"nofx/manager"
	"nofx/store"
	"nofx/trader"
)

// seedTokenOwner creates the account row a test token names: since H2,
// authMiddleware refuses a token whose account row does not exist (fail
// closed — no retire epoch can be established for it). The row is created
// now, so updated_at == created_at (never changed) and it retires nothing.
func seedTokenOwner(t *testing.T, st *store.Store, userID, email string) {
	t.Helper()
	if err := st.User().Create(&store.User{ID: userID, Email: email, PasswordHash: "x"}); err != nil {
		t.Fatalf("seed the token's account row %s: %v", userID, err)
	}
}

// boundaryAfter is the whole-second wait target, as a pure function of now —
// the seam that makes the chrony step-back flake drivable (this box steps the
// clock BACK ~130×/day, median 1.09 s).
func boundaryAfter(now time.Time) time.Time {
	return now.Truncate(time.Second).Add(time.Second + 20*time.Millisecond)
}

// untilNextSecondRetries is the number of re-targets untilNextSecond gets.
// 1 = the old single-read semantics (the flake); the test drives both sides.
var untilNextSecondRetries = 3

// untilNextSecond sleeps past the next whole-second boundary (the retire rule
// compares whole seconds, so a token minted in the change's own second is
// retired by design — red-team red-1 #5). After each sleep it RE-READS the
// clock: a chrony step-back across the boundary leaves the first read stale
// (the H2 whole-second pin failed once in a full run at exactly this —
// "Time jumped backwards" at the failing second), so it re-targets.
func untilNextSecond() {
	for i := 0; i < untilNextSecondRetries; i++ {
		now := time.Now()
		time.Sleep(boundaryAfter(now).Sub(now))
		if time.Now().After(now.Truncate(time.Second).Add(time.Second)) {
			return
		}
	}
}

// TestWholeSecondWaitReTargetsAcrossAClockStepBack reproduces the H2 flake
// deterministically: read 1 lands 900 ms into the change's second, chrony
// steps BACK across the boundary, and the whole-second rule judged under the
// stepped clock refuses a token minted at read 1's boundary — the exact
// positive-control failure line. The re-target (one more read) moves the
// boundary past the stepped clock and the wait recovers.
func TestWholeSecondWaitReTargetsAcrossAClockStepBack(t *testing.T) {
	sec0 := time.Date(2026, 9, 24, 3, 0, 5, 0, time.UTC)
	changed := sec0

	// read 1: just before the boundary.
	target1 := boundaryAfter(sec0.Add(900 * time.Millisecond))
	if !target1.Equal(sec0.Add(time.Second + 20*time.Millisecond)) {
		t.Fatalf("boundaryAfter = %v, want %v", target1, sec0.Add(time.Second+20*time.Millisecond))
	}

	// chrony steps BACK across the boundary: the next whole-second boundary
	// from the stepped clock is now the change's own second again.
	now2 := sec0.Add(-300 * time.Millisecond)
	if boundaryAfter(now2).After(sec0.Add(time.Second)) {
		t.Fatalf("step-back not reproduced: boundaryAfter(%v) = %v", now2, boundaryAfter(now2))
	}

	// The whole-second rule refuses a token minted UNDER the stepped clock
	// (now2 ≤ sec0 lands at or before the change's second) — the H2 failure
	// line: the positive control minted after the wait is refused because
	// chrony stepped the clock back into the change's own second
	// ("Time jumped backwards" at the failing second).
	if !auth.IssuedNotAfter(jwtAt(now2), changed) {
		t.Fatal("step-back shape not reproduced: a token minted under the stepped clock is NOT refused")
	}

	// A single-read wait cannot recover (the old flake); with the re-target
	// the boundary moves past the stepped clock's second.
	if untilNextSecondRetries <= 1 {
		t.Fatal("the wait has no re-target — a chrony step-back strands the test in the change's own second (the H2 flake)")
	}
}

func jwtAt(ts time.Time) *jwt.NumericDate { return jwt.NewNumericDate(ts) }

func TestRetiredTokenCannotActAnywhere(t *testing.T) {
	e := newUpdEnv(t)
	stolen := e.tok // a genuine owner token, iat 5 s ago
	botBefore := mustBotToken(t)

	// Controls before the change: the stolen token works everywhere.
	for _, p := range []string{"/api/my-traders", "/api/config/resolved", "/api/telegram"} {
		if w := credCall(t, e, "GET", p, stolen, ""); w.Code != http.StatusOK {
			t.Fatalf("control: GET %s with the owner's token before any change = %d", p, w.Code)
		}
	}

	// The owner rotates the password (right current password).
	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	if w := credCall(t, e, "PUT", "/api/user/password", owner, `{"current_password":"`+updAdminPass+`","new_password":"owner-rotated-pass-5"}`); w.Code != http.StatusOK {
		t.Fatalf("the owner's rotation = %d %s", w.Code, w.Body.String())
	}
	e.expectAllForbidden("stolen token after the rotation (Q8 control)", withToken(stolen))

	// The Q8-retired token cannot act ANYWHERE — non-credential routes, the
	// bot-config routes, the credential routes (even with the right current
	// password) — and neither can the session that made the change, nor the
	// bot token minted before it.
	probes := []struct{ method, path, body string }{
		{"GET", "/api/my-traders", ""},
		{"GET", "/api/config/resolved", ""},
		{"GET", "/api/telegram", ""},
		{"DELETE", "/api/telegram/binding", ""},
		{"POST", "/api/telegram", `{"bot_token":"123:abc","model_id":"m"}`},
		{"PUT", "/api/user/password", `{"current_password":"owner-rotated-pass-5","new_password":"thief-pass-0005"}`},
		{"POST", "/api/logout", ""},
	}
	for name, tok := range map[string]string{
		"stolen (Q8-retired) token":          stolen,
		"the session that made the change":   owner,
		"bot token minted before the change": botBefore,
	} {
		for _, p := range probes {
			if name == "bot token minted before the change" && (strings.HasPrefix(p.path, "/api/telegram") || p.path == "/api/user/password") {
				continue // machine-denied anyway (403); the retire rule is shown on the others
			}
			if w := credCall(t, e, p.method, p.path, tok, p.body); w.Code != http.StatusUnauthorized {
				t.Fatalf("%s: %s %s = %d %s — want 401 (issued before the account's last credential change)", name, p.method, p.path, w.Code, w.Body.String())
			}
		}
	}
	if _, code := credLogin(t, e, updAdminEmail, "thief-pass-0005"); code != http.StatusUnauthorized {
		t.Fatalf("login with the thief's password = %d, want 401", code)
	}

	// The whole-second rule, exactly Q8's: iat in the change's own second is
	// retired; iat one second later is not.
	u, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		t.Fatal(err)
	}
	sameSecond := mintJWT(t, updAdminID, updAdminEmail, u.UpdatedAt.Truncate(time.Second), time.Now().Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", sameSecond, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("token issued in the change's own second: GET /api/my-traders = %d — want 401", w.Code)
	}
	nextSecond := mintJWT(t, updAdminID, updAdminEmail, u.UpdatedAt.Truncate(time.Second).Add(time.Second), time.Now().Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", nextSecond, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: token issued the second after the change: GET /api/my-traders = %d — want 200", w.Code)
	}

	// Positive controls: a session opened after the change works everywhere
	// ordinary, and a bot token minted after it keeps the agent's routes.
	untilNextSecond()
	fresh, code := credLogin(t, e, updAdminEmail, "owner-rotated-pass-5")
	if code != http.StatusOK {
		t.Fatalf("login with the rotated password = %d", code)
	}
	for _, p := range []string{"/api/my-traders", "/api/config/resolved", "/api/telegram"} {
		if w := credCall(t, e, "GET", p, fresh, ""); w.Code != http.StatusOK {
			t.Fatalf("positive control: a post-change session on GET %s = %d", p, w.Code)
		}
	}
	if w := credCall(t, e, "GET", "/api/my-traders", mustBotToken(t), ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: a bot token minted after the change on GET /api/my-traders = %d", w.Code)
	}
}

// A token whose account row no longer exists (reset-account wiped it) or
// that carries no iat is refused everywhere (fail closed: no epoch can be
// established for it).
func TestTokenWithoutAnAccountRowOrIatIsRefusedEverywhere(t *testing.T) {
	e := newUpdEnv(t)
	noIAT := mintNoIAT(t, updAdminID, updAdminEmail, time.Now().Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", noIAT, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("token without iat: GET /api/my-traders = %d — want 401", w.Code)
	}
	ghost := mintJWT(t, "dddddddd-0000-0000-0000-000000000000", "ghost@example.test", time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", ghost, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("token for a user with no row: GET /api/my-traders = %d — want 401", w.Code)
	}
	if w := credCall(t, e, "GET", "/api/my-traders", e.tok, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: the owner's token = %d", w.Code)
	}
}

// Registration is not a credential change: the token /api/register returns —
// minted in the same second the row was created — works at once (a row that
// never changed has no retire epoch; updated_at == created_at).
func TestRegistrationTokenWorksAtOnce(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(filepath.Join(dataDir, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	prevDir := trader.MaintenanceDataDir()
	trader.SetMaintenanceDataDir(dataDir)
	t.Cleanup(func() { trader.SetMaintenanceDataDir(prevDir) })
	prev := auth.JWTSecret
	auth.SetJWTSecret(updSecret)
	t.Cleanup(func() { auth.JWTSecret = prev })
	e := &updEnv{t: t, st: st, dataDir: dataDir, s: NewServer(manager.NewTraderManager(), st, nil, "127.0.0.1", 0)}

	w := credCall(t, e, "POST", "/api/register", "", `{"email":"new-owner@example.test","password":"new-owner-pass-1"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("register = %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w := credCall(t, e, "GET", "/api/my-traders", out.Token, ""); w.Code != http.StatusOK {
		t.Fatalf("the registration token on GET /api/my-traders = %d %s — want 200", w.Code, w.Body.String())
	}
}
