package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// token-iat-same-second (owner order 2026-10-02): HS256 mints with the same
// user/email/iat are byte-identical, so a logout's exact-string blacklist
// killed a fresh re-login minted in the same wall-clock second (CI flake on
// api TestPasswordChangeUnbindsTheEnrollment: login 200 then the very next
// PUT 401 "Token expired, please login again"). Every mint now carries a
// distinct jti. This pin freezes the mint clock (the Now seam — deterministic,
// never a sleep): two mints at the same instant must differ, and blacklisting
// the first must leave the second admitted.
func TestSameSecondMintsAreDistinctAndLogoutKillsOnlyItsOwn(t *testing.T) {
	SetJWTSecret("test-secret")
	prev := Now
	frozen := time.Now().Truncate(time.Second)
	Now = func() time.Time { return frozen }
	t.Cleanup(func() { Now = prev })

	first, err := GenerateJWT("u1", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateJWT("u1", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two mints at the same instant produced the same token string — a logout would kill the next login")
	}
	jtiOf := func(tok string) string {
		var c Claims
		if _, err := jwt.ParseWithClaims(tok, &c, func(*jwt.Token) (any, error) { return []byte("test-secret"), nil }); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if c.JTI == "" {
			t.Fatal("the minted token carries no jti")
		}
		return c.JTI
	}
	if a, b := jtiOf(first), jtiOf(second); a == b {
		t.Fatalf("two mints share jti %q", a)
	}
	BlacklistToken(first, frozen.Add(time.Hour))
	if !IsTokenBlacklisted(first) {
		t.Fatal("the logged-out token must be blacklisted")
	}
	if IsTokenBlacklisted(second) {
		t.Fatal("a same-second re-login minted a token the logout blacklist kills")
	}
}
