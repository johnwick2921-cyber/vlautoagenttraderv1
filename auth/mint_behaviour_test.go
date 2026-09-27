package auth_test

// M3 hb verify note 1: TestServerNeverMintsAFutureIat (clock_window_test.go)
// is a census of how IssuedAt/NotBefore are SPELLED, so a compiling
// `claims.IssuedAt.Time = claims.IssuedAt.Time.Add(30 * time.Second)` in
// signToken minted a future iat with the census and the whole auth package
// green. This pins the VALUE: every production mint entry point is called,
// its token decoded, and iat <= now, nbf == iat, exp == iat + 24h asserted.
// (cmd/gate-jwt's mintGateToken is package main; it mints through
// GenerateScopedJWT(..., ScopeGateJWT), the entry point exercised here, and
// cmd/gate-jwt pins that main mints only through it.)

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"nofx/auth"
	"nofx/telegram/agent"
)

func TestEveryMintEntryPointStampsNowNotTheFuture(t *testing.T) {
	prev := auth.JWTSecret
	auth.SetJWTSecret("mint-behaviour-fixture-secret-0123456789abcdef")
	t.Cleanup(func() { auth.JWTSecret = prev })

	mints := []struct {
		name string
		mint func() (string, error)
	}{
		{"auth.GenerateJWT (login/register)", func() (string, error) { return auth.GenerateJWT("u", "u@example.test") }},
		{"agent.GenerateBotToken (the Telegram bot)", func() (string, error) { return agent.GenerateBotToken("u") }},
		{"auth.GenerateScopedJWT(gate-jwt)", func() (string, error) { return auth.GenerateScopedJWT("u", "u@example.test", auth.ScopeGateJWT) }},
	}
	for _, m := range mints {
		before := time.Now()
		s, err := m.mint()
		if err != nil {
			t.Fatalf("%s: %v", m.name, err)
		}
		after := time.Now()
		var cl auth.Claims
		if _, _, err := jwt.NewParser().ParseUnverified(s, &cl); err != nil {
			t.Fatalf("%s: decode: %v", m.name, err)
		}
		if cl.IssuedAt == nil || cl.NotBefore == nil || cl.ExpiresAt == nil {
			t.Fatalf("%s: iat/nbf/exp missing: %+v", m.name, cl.RegisteredClaims)
		}
		iat := cl.IssuedAt.Time
		if iat.Unix() > after.Unix() || iat.Unix() < before.Unix() {
			t.Errorf("%s: minted iat %v is not the mint instant (between %v and %v) — a future iat outlives a later password change once the clock reaches it", m.name, iat, before, after)
		}
		// signToken reads the clock separately for iat, nbf and exp, each
		// truncated to whole seconds, so a second boundary (or one of this
		// box's ~1 s chrony steps) between the reads moves a difference by
		// one second either way.
		if d := cl.NotBefore.Time.Sub(iat); d < -time.Second || d > time.Second {
			t.Errorf("%s: nbf %v is not iat %v (±1s)", m.name, cl.NotBefore.Time, iat)
		}
		if d := cl.ExpiresAt.Time.Sub(iat) - 24*time.Hour; d < -time.Second || d > time.Second {
			t.Errorf("%s: exp - iat = %v, want 24h (±1s)", m.name, cl.ExpiresAt.Time.Sub(iat))
		}
	}
}
