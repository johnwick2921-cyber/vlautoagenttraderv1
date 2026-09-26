package api

// M3 red-team H2 (red1 R2), ported to a pin at the PRODUCTION router: the
// owner rotates the password (e.g. after suspecting a token theft); Q8 then
// retires the thief's older token on /api/updates — but the retired token
// still passed authMiddleware, so it could change the password AGAIN, log in
// with it, and lock the owner out. A token issued before the row's last
// credential change must not act on the credentials.
//
// Since the CTO ruled H2 GLOBAL (1790231205208), authMiddleware refuses the
// retired token (401) before the credential guard is reached; the guard's own
// re-check (403) is pinned separately by driving the handler with retired
// claims (TestCredentialGuardReChecksRetirement).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nofx/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestRetiredTokenCannotChangeThePasswordAgain(t *testing.T) {
	e := newUpdEnv(t)
	stolen := e.tok // a genuine owner token, iat 5 s ago

	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	if w := credCall(t, e, "PUT", "/api/user/password", owner, `{"current_password":"`+updAdminPass+`","new_password":"owner-rotated-pass-9"}`); w.Code != http.StatusOK {
		t.Fatalf("positive control: the owner's rotation = %d %s", w.Code, w.Body.String())
	}
	e.expectAllForbidden("stolen token after the owner's rotation (control: Q8 holds)", withToken(stolen))
	rotated := e.adminRow()

	// The revival attempt — carrying the CORRECT (rotated) current password,
	// so the refusal is the retirement's alone, not the current-password check.
	if w := credCall(t, e, "PUT", "/api/user/password", stolen, `{"current_password":"owner-rotated-pass-9","new_password":"thief-pass-0001"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("PUT /api/user/password with the RETIRED token = %d %s — want 401 (H2: refused at authMiddleware)", w.Code, w.Body.String())
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
	if w := credCall(t, e, "POST", "/api/reset-account", stolen, `{"confirm":"RESET-ALL-DATA"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/reset-account with the RETIRED token = %d %s — want 401", w.Code, w.Body.String())
	}
	if n, _ := e.st.User().Count(); n != 1 {
		t.Fatalf("users = %d after a refused reset", n)
	}
	// Positive control: a session opened AFTER the rotation — past the
	// change's own second (the whole-second rule retires that second) — may
	// change it.
	untilNextSecond()
	after, _ := credLogin(t, e, updAdminEmail, "owner-rotated-pass-9")
	if w := credCall(t, e, "PUT", "/api/user/password", after, `{"current_password":"owner-rotated-pass-9","new_password":"owner-rotated-pass-10"}`); w.Code != http.StatusOK {
		t.Fatalf("positive control: a post-rotation session's change = %d %s", w.Code, w.Body.String())
	}
}

// The credential guard re-checks the retirement with the SAME predicate
// (auth.RetiredBy), so the credential handlers can never be wired without it:
// driven directly with the claims authMiddleware would set, a token issued in
// the change's own second is refused 403 and nothing is written; one issued
// the next second changes the password.
func TestCredentialGuardReChecksRetirement(t *testing.T) {
	e := newUpdEnv(t)
	if err := e.st.User().UpdatePassword(updAdminID, e.adminRow().hash); err != nil { // a credential change NOW (same hash)
		t.Fatal(err)
	}
	u, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		t.Fatal(err)
	}
	drive := func(iat time.Time) *httptest.ResponseRecorder {
		cl := &auth.Claims{UserID: updAdminID, Email: updAdminEmail, RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(iat)}}
		r := gin.New()
		r.PUT("/api/user/password", func(c *gin.Context) {
			setAuthContext(c, cl)
			e.s.handleChangePassword(c)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/user/password", strings.NewReader(`{"current_password":"`+updAdminPass+`","new_password":"guard-pass-0001"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	before := e.adminRow()
	if w := drive(u.UpdatedAt.Truncate(time.Second)); w.Code != http.StatusForbidden {
		t.Fatalf("credential guard, iat in the change's own second = %d %s — want 403", w.Code, w.Body.String())
	}
	if e.adminRow() != before {
		t.Fatal("a guard refusal wrote the admin row")
	}
	if w := drive(u.UpdatedAt.Truncate(time.Second).Add(time.Second)); w.Code != http.StatusOK {
		t.Fatalf("positive control: credential guard, iat the next second = %d %s — want 200", w.Code, w.Body.String())
	}
}
