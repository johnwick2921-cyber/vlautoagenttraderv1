package auth

// P2-10 (audit 0926-system): the logout blacklist was in-memory only, so a
// restart revived logged-out tokens for the rest of their 24h life. With a
// TokenBlacklistStore set, BlacklistToken persists a fingerprint+expiry and
// IsTokenBlacklisted consults it after the in-memory map — a "restart"
// (memory cleared) can no longer revive the token.

import (
	"sync"
	"testing"
	"time"
)

// memStore is a minimal in-memory TokenBlacklistStore for these pins.
type memStore struct {
	mu    sync.Mutex
	rows  map[string]time.Time
	fails int // counts Save failures when >0
}

func (m *memStore) Save(tokenID string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rows == nil {
		m.rows = map[string]time.Time{}
	}
	m.rows[tokenID] = expiresAt
	return nil
}

func (m *memStore) IsRevoked(tokenID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.rows[tokenID]
	if !ok {
		return false, nil
	}
	return time.Now().Before(exp), nil
}

func (m *memStore) PruneExpired(now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, exp := range m.rows {
		if now.After(exp) {
			delete(m.rows, id)
		}
	}
	return nil
}

func resetBlacklistForTest(t *testing.T, store TokenBlacklistStore) {
	t.Helper()
	SetTokenBlacklistStore(store)
	tokenBlacklist.Lock()
	tokenBlacklist.items = make(map[string]time.Time)
	tokenBlacklist.Unlock()
	t.Cleanup(func() {
		SetTokenBlacklistStore(nil)
		tokenBlacklist.Lock()
		tokenBlacklist.items = make(map[string]time.Time)
		tokenBlacklist.Unlock()
	})
}

func TestBlacklistTokenPersistsAcrossMemoryReset(t *testing.T) {
	m := &memStore{}
	resetBlacklistForTest(t, m)

	token := "eyJ-test-token-one"
	exp := time.Now().Add(time.Hour)
	BlacklistToken(token, exp)

	fp := TokenFingerprint(token)
	if _, ok := m.rows[fp]; !ok {
		t.Fatal("BlacklistToken did not persist the fingerprint to the store")
	}

	// Simulate a restart: the in-memory map is empty, the store survives.
	tokenBlacklist.Lock()
	tokenBlacklist.items = make(map[string]time.Time)
	tokenBlacklist.Unlock()

	if !IsTokenBlacklisted(token) {
		t.Fatal("after memory reset the revoked token must STILL be blacklisted via the store")
	}
	// The store's answer must now be cached in memory too.
	tokenBlacklist.Lock()
	_, cached := tokenBlacklist.items[token]
	tokenBlacklist.Unlock()
	if !cached {
		t.Fatal("the store hit must be cached in memory")
	}
}

func TestBlacklistTokenExpiryStillBoundsRevocation(t *testing.T) {
	m := &memStore{}
	resetBlacklistForTest(t, m)

	token := "eyJ-test-token-two"
	BlacklistToken(token, time.Now().Add(-2*time.Hour)) // exp + leeway already past
	tokenBlacklist.Lock()
	tokenBlacklist.items = make(map[string]time.Time)
	tokenBlacklist.Unlock()

	if IsTokenBlacklisted(token) {
		t.Fatal("an expired revocation must not blacklist the token")
	}
}

func TestBlacklistWithoutStoreIsMemoryOnly(t *testing.T) {
	resetBlacklistForTest(t, nil)

	token := "eyJ-test-token-three"
	BlacklistToken(token, time.Now().Add(time.Hour))
	if !IsTokenBlacklisted(token) {
		t.Fatal("memory blacklist must still work with no store")
	}
}

func TestTokenFingerprintIsStableAndOneWay(t *testing.T) {
	a := TokenFingerprint("token-a")
	b := TokenFingerprint("token-a")
	if a != b {
		t.Fatal("fingerprint must be deterministic")
	}
	if a == "token-a" || len(a) != 64 {
		t.Fatalf("fingerprint must be a 64-char hex digest, got %q", a)
	}
	if TokenFingerprint("token-b") == a {
		t.Fatal("different tokens must differ")
	}
}
