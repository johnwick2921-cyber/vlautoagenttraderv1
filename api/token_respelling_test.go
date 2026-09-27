package api

// M3 red-team M2 (red-1 #4), CTO ruling 1790231205208 "M2 strict base64url —
// fold": auth.ValidateJWT decoded with jwt v5's default NON-strict base64url,
// and the logout blacklist is an exact-string map. An HS256 signature is 32
// bytes = 43 base64url chars carrying 258 bits, so the last char has 2 bits
// the lenient decoder ignores: 3 other spellings of a logged-out token
// verified identically and were live again — on the protected group AND on
// /api/updates. Ported red-1 probe, at the PRODUCTION router (canon 53).

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// respell returns the other spellings of tok's last base64url character that
// decode to the same bytes under lenient decoding (only its 2 unused low bits
// differ).
func respell(tok string) []string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	v := strings.IndexByte(alpha, tok[len(tok)-1])
	var out []string
	for low := 0; low < 4; low++ {
		if nv := (v &^ 3) | low; nv != v {
			out = append(out, tok[:len(tok)-1]+string(alpha[nv]))
		}
	}
	return out
}

func TestLoggedOutTokenStaysRevokedUnderASignatureRespelling(t *testing.T) {
	e := newUpdEnv(t)
	now := time.Now()
	tok := mintJWT(t, updAdminID, updAdminEmail, now.Add(-3*time.Second), now.Add(time.Hour), updSecret)
	if n := len(strings.Split(tok, ".")[2]); n != 43 {
		t.Fatalf("control: HS256 signature segment is %d chars, want 43", n)
	}
	alts := respell(tok)
	if len(alts) != 3 {
		t.Fatalf("control: %d respellings, want 3", len(alts))
	}
	e.expectAllAdmitted("before logout", withToken(tok))
	if w := credCall(t, e, "POST", "/api/logout", tok, ""); w.Code != http.StatusOK {
		t.Fatalf("logout = %d %s", w.Code, w.Body.String())
	}
	if w := credCall(t, e, "GET", "/api/config/resolved", tok, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("control: the exact logged-out string on a protected route = %d, want 401", w.Code)
	}
	e.expectAllForbidden("control: the exact logged-out string", withToken(tok))
	for i, alt := range alts {
		if w := credCall(t, e, "GET", "/api/config/resolved", alt, ""); w.Code != http.StatusUnauthorized {
			t.Fatalf("respelling %d of a LOGGED-OUT token on GET /api/config/resolved = %d — want 401 (strict base64url)", i, w.Code)
		}
		if w := e.do("GET", "/api/updates", "", withToken(alt)); w.Code != http.StatusForbidden {
			t.Fatalf("respelling %d of a LOGGED-OUT token on GET /api/updates = %d — want 403", i, w.Code)
		}
		if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease)), withToken(alt)); w.Code != http.StatusForbidden {
			t.Fatalf("respelling %d of a LOGGED-OUT token on POST /api/updates/install = %d — want 403", i, w.Code)
		}
	}
	// Positive control: a fresh, canonical token still works everywhere.
	fresh := mintJWT(t, updAdminID, updAdminEmail, now.Add(-2*time.Second), now.Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/config/resolved", fresh, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: a canonical token on GET /api/config/resolved = %d", w.Code)
	}
	e.expectAllAdmitted("positive control: a canonical token", withToken(fresh))
}
