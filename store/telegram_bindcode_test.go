package store

import (
	"path/filepath"
	"testing"
	"time"
)

// P2-12 (audit 0926-system): the Telegram first-/start bind gets a
// confirmation gate — a one-time code issued in the owner-authenticated app
// that the chat must send back.

func TestTelegramBindCodeIssueConsumeSingleUse(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tc := st.TelegramConfig()

	if err := tc.Save("tok", ""); err != nil {
		t.Fatal(err)
	}
	if err := tc.IssueBindCode("CODE1234", time.Now().Add(10*time.Minute)); err != nil {
		t.Fatalf("issue: %v", err)
	}
	ok, err := tc.ConsumeBindCode("CODE1234")
	if err != nil || !ok {
		t.Fatalf("correct code must consume: %v/%v", ok, err)
	}
	ok, err = tc.ConsumeBindCode("CODE1234")
	if err != nil || ok {
		t.Fatalf("the code must be single-use: second consume %v/%v", ok, err)
	}
}

func TestTelegramBindCodeWrongAndExpired(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tc := st.TelegramConfig()

	if err := tc.Save("tok", ""); err != nil {
		t.Fatal(err)
	}
	if err := tc.IssueBindCode("CODE1234", time.Now().Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if ok, _ := tc.ConsumeBindCode("WRONG999"); ok {
		t.Fatal("wrong code must not consume")
	}
	// A wrong attempt must NOT clear the pending code.
	if ok, _ := tc.ConsumeBindCode("CODE1234"); !ok {
		t.Fatal("the pending code must survive a wrong attempt")
	}

	// Expired code: refused and cleared.
	if err := tc.IssueBindCode("CODE5678", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if ok, _ := tc.ConsumeBindCode("CODE5678"); ok {
		t.Fatal("expired code must not consume")
	}
	if ok, _ := tc.ConsumeBindCode("CODE5678"); ok {
		t.Fatal("expired code must be cleared after the refused attempt")
	}
}
