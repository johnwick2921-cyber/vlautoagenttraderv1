package api

// token-iat-same-second (owner order 2026-10-02) at the PRODUCTION call site
// (canon 53): the CI flake was login 200 then the very next PUT 401
// "Token expired, please login again" — the logout blacklist held a
// byte-identical token minted in the same wall-clock second (the cutover-
// worker census logs out its own fresh owner token while walking POST
// /api/logout, and a same-second re-login re-mints the same string). With the
// jti claim the re-login mints a distinct string and survives. The mint clock
// is frozen through auth.Now — deterministic, never a sleep.
import (
	"testing"
	"time"

	"vl/auth"
)

func TestALogoutDoesNotKillTheNextSameSecondLogin(t *testing.T) {
	e := newUpdEnv(t)
	prev := auth.Now
	frozen := time.Now()
	auth.Now = func() time.Time { return frozen }
	t.Cleanup(func() { auth.Now = prev })

	first, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != 200 {
		t.Fatalf("first login = %d", code)
	}
	if w := credCall(t, e, "POST", "/api/logout", first, ""); w.Code != 200 {
		t.Fatalf("logout = %d %s", w.Code, w.Body.String())
	}
	// The same second, the same account: the new mint must be a NEW string.
	second, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != 200 {
		t.Fatalf("second login = %d", code)
	}
	if second == first {
		t.Fatal("the same-second re-login produced the blacklisted string again")
	}
	if w := credCall(t, e, "GET", "/api/my-traders", second, ""); w.Code != 200 {
		t.Fatalf("the fresh same-second token was refused: %d %s", w.Code, w.Body.String())
	}
	if w := credCall(t, e, "GET", "/api/my-traders", first, ""); w.Code != 401 {
		t.Fatalf("the logged-out token must stay revoked: %d", w.Code)
	}
	// The CI failure's exact positive control: the password change with the
	// fresh token.
	if w := credCall(t, e, "PUT", "/api/user/password", second, `{"current_password":"`+updAdminPass+`","new_password":"owner-rotated-same-second"}`); w.Code != 200 {
		t.Fatalf("password change with the fresh same-second token = %d %s", w.Code, w.Body.String())
	}
}
