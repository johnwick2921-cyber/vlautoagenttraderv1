package api

// M3 red-team H2 (red1 R2), ported to a pin at the PRODUCTION router: the
// owner rotates the password (e.g. after suspecting a token theft); Q8 then
// retires the thief's older token on /api/updates — but the retired token
// still passed authMiddleware, so it could change the password AGAIN, log in
// with it, and lock the owner out. A token issued before the row's last
// credential change must not act on the credentials.

import (
	"net/http"
	"testing"
)

func TestRetiredTokenCannotChangeThePasswordAgain(t *testing.T) {
	e := newUpdEnv(t)
	stolen := e.tok // a genuine owner token, iat 5 s ago

	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	if w := credCall(t, e, "PUT", "/api/user/password", owner, `{"new_password":"owner-rotated-pass-9"}`); w.Code != http.StatusOK {
		t.Fatalf("positive control: the owner's rotation = %d %s", w.Code, w.Body.String())
	}
	e.expectAllForbidden("stolen token after the owner's rotation (control: Q8 holds)", withToken(stolen))
	rotated := e.adminRow()

	// The revival attempt.
	if w := credCall(t, e, "PUT", "/api/user/password", stolen, `{"new_password":"thief-pass-0001"}`); w.Code != http.StatusForbidden {
		t.Fatalf("PUT /api/user/password with the RETIRED token = %d %s — want 403", w.Code, w.Body.String())
	}
	if e.adminRow() != rotated {
		t.Fatal("a refused revival still wrote the admin row")
	}
	if _, code := credLogin(t, e, updAdminEmail, "thief-pass-0001"); code != http.StatusUnauthorized {
		t.Fatalf("login with the thief's password = %d, want 401", code)
	}
	if _, code := credLogin(t, e, updAdminEmail, "owner-rotated-pass-9"); code != http.StatusOK {
		t.Fatalf("positive control: the owner's rotated password no longer logs in (%d)", code)
	}
	// The same retirement guards the other credential route.
	t.Setenv("ALLOW_ACCOUNT_RESET", "1")
	if w := credCall(t, e, "POST", "/api/reset-account", stolen, `{"confirm":"RESET-ALL-DATA"}`); w.Code != http.StatusForbidden {
		t.Fatalf("POST /api/reset-account with the RETIRED token = %d %s — want 403", w.Code, w.Body.String())
	}
	if n, _ := e.st.User().Count(); n != 1 {
		t.Fatalf("users = %d after a refused reset", n)
	}
	// Positive control: a session opened AFTER the rotation may change it.
	after, _ := credLogin(t, e, updAdminEmail, "owner-rotated-pass-9")
	if w := credCall(t, e, "PUT", "/api/user/password", after, `{"new_password":"owner-rotated-pass-10"}`); w.Code != http.StatusOK {
		t.Fatalf("positive control: a post-rotation session's change = %d %s", w.Code, w.Body.String())
	}
}
