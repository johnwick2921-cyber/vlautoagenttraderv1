package store

import (
	"path/filepath"
	"testing"
	"time"
)

// revokedTokensStore opens a temp sqlite Store for the revocation pins.
func revokedTokensStore(t *testing.T) *RevokedTokenStore {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st.RevokedTokens()
}

func TestRevokedTokenStoreSaveIsRevokedPrune(t *testing.T) {
	s := revokedTokensStore(t)

	if err := s.Save("abc", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("save: %v", err)
	}
	ok, err := s.IsRevoked("abc")
	if err != nil || !ok {
		t.Fatalf("want revoked=true, got %v/%v", ok, err)
	}
	ok, err = s.IsRevoked("missing")
	if err != nil || ok {
		t.Fatalf("missing id must not be revoked: %v/%v", ok, err)
	}

	if err := s.PruneExpired(time.Now().Add(2 * time.Hour)); err != nil {
		t.Fatalf("prune: %v", err)
	}
	ok, err = s.IsRevoked("abc")
	if err != nil || ok {
		t.Fatalf("pruned id must not be revoked: %v/%v", ok, err)
	}
}

func TestRevokedTokenStoreSaveIsIdempotent(t *testing.T) {
	s := revokedTokensStore(t)
	if err := s.Save("dup", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("dup", time.Now().Add(2*time.Hour)); err != nil {
		t.Fatalf("second save: %v", err)
	}
	n, err := s.Count()
	if err != nil || n != 1 {
		t.Fatalf("want 1 row, got %d (%v)", n, err)
	}
}
