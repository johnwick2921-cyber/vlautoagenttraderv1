package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestWorkerTokenTTLIsTenYearsAndOtherScopesStay24h pins the per-scope TTL
// (P-E E3): only ScopeCutoverWorker gets the human-timescale lifetime; user,
// telegram and gate-jwt tokens keep 24 h.
func TestWorkerTokenTTLIsTenYearsAndOtherScopesStay24h(t *testing.T) {
	SetJWTSecret("test-secret")
	now := time.Now()
	user, err := GenerateJWT("u1", "owner@example.com")
	if err != nil {
		t.Fatalf("user mint: %v", err)
	}
	worker, err := GenerateWorkerJWT("u1", "owner@example.com", 0)
	if err != nil {
		t.Fatalf("worker mint: %v", err)
	}
	expOf := func(tok string) int64 {
		t.Helper()
		var claims Claims
		if _, err := jwt.ParseWithClaims(tok, &claims, func(*jwt.Token) (any, error) {
			return []byte("test-secret"), nil
		}); err != nil {
			t.Fatalf("parse: %v", err)
		}
		return claims.ExpiresAt.Unix()
	}
	userLife := expOf(user) - now.Unix()
	workerLife := expOf(worker) - now.Unix()
	if userLife < 23*3600 || userLife > 25*3600 {
		t.Fatalf("user token life = %ds — want ~24h", userLife)
	}
	if workerLife < int64(WorkerTokenTTL/time.Second)-3600 {
		t.Fatalf("worker token life = %ds — want ~%s", workerLife, WorkerTokenTTL)
	}
}

// TestWorkerTokenCarriesScopeAndWTE pins the claim shape the API guard reads.
func TestWorkerTokenCarriesScopeAndWTE(t *testing.T) {
	SetJWTSecret("test-secret")
	tok, err := GenerateWorkerJWT("u1", "owner@example.com", 4)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	var claims Claims
	if _, err := jwt.ParseWithClaims(tok, &claims, func(*jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Scope != ScopeCutoverWorker {
		t.Fatalf("scope = %q — want %q", claims.Scope, ScopeCutoverWorker)
	}
	if claims.WTE != 4 {
		t.Fatalf("wte = %d — want 4", claims.WTE)
	}
	if !claims.IsMachine() {
		t.Fatal("a worker token is a machine token")
	}
}

// TestCurrentWorkerEpochFailsClosedWithNoReader: before boot installs the
// reader, the API must refuse every worker token.
func TestCurrentWorkerEpochFailsClosedWithNoReader(t *testing.T) {
	SetWorkerEpochReader(nil)
	if _, err := CurrentWorkerEpoch(); err == nil {
		t.Fatal("CurrentWorkerEpoch with no reader must error (fail closed)")
	}
	SetWorkerEpochReader(func() (int64, error) { return 9, nil })
	n, err := CurrentWorkerEpoch()
	if err != nil || n != 9 {
		t.Fatalf("epoch = %d (err %v) — want 9", n, err)
	}
	SetWorkerEpochReader(nil)
}
