package store

import (
	"path/filepath"
	"testing"
)

// C2 (2026-08-25) — sticky owner levels are per-user: reads, note-tag updates
// and deletes are scoped to the creator. Pre-C2 '' (legacy) rows remain
// visible to everyone (they were backfilled to the original owner at migration).
func TestOwnerLevelsUserScoped(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "ol.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	o := st.OwnerLevel()
	if err := o.Save(&OwnerLevelDB{Symbol: "MNQ", Price: 100, Label: "u1-level", UserID: "u1"}); err != nil {
		t.Fatalf("save u1: %v", err)
	}
	if err := o.Save(&OwnerLevelDB{Symbol: "MNQ", Price: 101, Label: "u2-level", UserID: "u2"}); err != nil {
		t.Fatalf("save u2: %v", err)
	}
	if err := o.Save(&OwnerLevelDB{Symbol: "MNQ", Price: 102, Label: "legacy", UserID: ""}); err != nil {
		t.Fatalf("save legacy: %v", err)
	}

	u1, err := o.ListActiveForUser("u1", "MNQ")
	if err != nil || len(u1) != 2 {
		t.Fatalf("u1 should see own + legacy rows, got %d (err=%v)", len(u1), err)
	}
	u2, err := o.ListActiveForUser("u2", "MNQ")
	if err != nil || len(u2) != 2 {
		t.Fatalf("u2 should see own + legacy rows, got %d (err=%v)", len(u2), err)
	}
	// The cross-user row must never appear in the other user's list.
	for _, r := range u1 {
		if r.Label == "u2-level" {
			t.Fatal("u1 saw u2's owner level — cross-user leak")
		}
	}
	for _, r := range u2 {
		if r.Label == "u1-level" {
			t.Fatal("u2 saw u1's owner level — cross-user leak")
		}
	}

	// Cross-user delete refused.
	var u2id int64
	for _, r := range u2 {
		if r.Label == "u2-level" {
			u2id = r.ID
		}
	}
	if err := o.DeleteForUser(u2id, "u1"); err != nil {
		t.Fatalf("cross-user delete error: %v", err)
	}
	if left, _ := o.ListActiveForUser("u2", "MNQ"); len(left) != 2 {
		t.Fatalf("u1 must not delete u2's row; u2 still has %d", len(left))
	}
	if err := o.DeleteForUser(u2id, "u2"); err != nil {
		t.Fatalf("own delete error: %v", err)
	}
	if left, _ := o.ListActiveForUser("u2", "MNQ"); len(left) != 1 {
		t.Fatalf("u2's delete should leave only legacy row, got %d", len(left))
	}

	// Cross-user note-tag update refused.
	if ok, err := o.UpdateNoteTag("u2", "MNQ", 100, "u1-level", "hijack", "x"); err != nil || ok {
		t.Fatalf("u2 must not update u1's row (ok=%v err=%v)", ok, err)
	}
	if ok, err := o.UpdateNoteTag("u1", "MNQ", 100, "u1-level", "mine", "y"); err != nil || !ok {
		t.Fatalf("u1's own update must succeed (ok=%v err=%v)", ok, err)
	}
}
