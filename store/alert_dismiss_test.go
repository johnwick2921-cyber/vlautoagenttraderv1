package store

import (
	"errors"
	"path/filepath"
	"testing"
)

// ITEM 5 (2026-08-17) — the feed can be cleared, the record cannot.
//
// Alerts could be acknowledged but never removed, so the feed only grew. These
// pin the contract: dismissal hides a row from the FEED, never destroys it;
// clear-all leaves unacknowledged alerts; an unacknowledged P0 refuses; and one
// trader can never touch another's alert.

func alertStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func emit(t *testing.T, st *Store, trader, level, event string, acked bool) int64 {
	t.Helper()
	if _, err := st.Alert().Emit(&AlertDB{
		TraderID: trader, Level: level, EventID: event,
		Kind: "k", Title: "t", Body: "b", Acked: acked, CreatedAt: 1_700_000_000,
	}); err != nil {
		t.Fatal(err)
	}
	rows, _ := st.Alert().List(trader, 100)
	for _, r := range rows {
		if r.EventID == event {
			return r.ID
		}
	}
	t.Fatalf("emitted alert %s not found in the feed", event)
	return 0
}

func feedIDs(t *testing.T, st *Store, trader string) []int64 {
	t.Helper()
	rows, err := st.Alert().List(trader, 100)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

func TestDismissRemovesFromFeedButKeepsTheRow(t *testing.T) {
	st := alertStore(t)
	id := emit(t, st, "t1", "P1", "e1", true)
	emit(t, st, "t1", "P1", "e2", true)

	found, err := st.Alert().DismissForTrader("t1", id, 1_700_000_100)
	if err != nil || !found {
		t.Fatalf("dismiss failed: found=%v err=%v", found, err)
	}
	if got := feedIDs(t, st, "t1"); len(got) != 1 || got[0] == id {
		t.Errorf("the dismissed alert is still in the feed: %v", got)
	}

	// AUDIT, NOT AMNESIA — the row survives, flagged.
	var row AlertDB
	if err := st.GormDB().Where("id = ?", id).First(&row).Error; err != nil {
		t.Fatalf("the underlying event row must survive dismissal: %v", err)
	}
	if !row.Dismissed || row.DismissedAt == 0 {
		t.Errorf("the row must be flagged dismissed with a timestamp, got %+v", row)
	}
}

// The whole point of the persistent P0 banner is that a halt cannot be swiped
// away unseen.
func TestUnackedP0RefusesDismissal(t *testing.T) {
	st := alertStore(t)
	id := emit(t, st, "t1", "P0", "halt", false)

	_, err := st.Alert().DismissForTrader("t1", id, 1_700_000_100)
	if !errors.Is(err, ErrP0NotAcked) {
		t.Fatalf("an unacknowledged P0 must refuse dismissal, got err=%v", err)
	}
	if got := feedIDs(t, st, "t1"); len(got) != 1 {
		t.Errorf("the P0 must still be in the feed, got %v", got)
	}

	// Once acknowledged it may be cleared like anything else.
	if _, err := st.Alert().AckForTrader("t1", id); err != nil {
		t.Fatal(err)
	}
	if found, err := st.Alert().DismissForTrader("t1", id, 1_700_000_200); err != nil || !found {
		t.Fatalf("an ACKNOWLEDGED P0 must be dismissible: found=%v err=%v", found, err)
	}
}

func TestClearAllReadLeavesUnackedAlerts(t *testing.T) {
	st := alertStore(t)
	emit(t, st, "t1", "P1", "read1", true)
	emit(t, st, "t1", "P2", "read2", true)
	unackedP0 := emit(t, st, "t1", "P0", "halt", false)
	unackedP1 := emit(t, st, "t1", "P1", "live", false)

	n, err := st.Alert().DismissAckedForTrader("t1", 1_700_000_100)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("cleared %d, want the 2 acknowledged alerts", n)
	}
	got := feedIDs(t, st, "t1")
	if len(got) != 2 {
		t.Fatalf("want the 2 unacked alerts left, got %v", got)
	}
	left := map[int64]bool{got[0]: true, got[1]: true}
	if !left[unackedP0] || !left[unackedP1] {
		t.Errorf("clear-all removed an UNACKNOWLEDGED alert: feed=%v", got)
	}
}

// The same IDOR guard as AckForTrader: one trader must never reach another's
// alert, and the attempt must 404 rather than silently succeed.
func TestDismissIsTraderScoped(t *testing.T) {
	st := alertStore(t)
	mine := emit(t, st, "t1", "P1", "mine", true)
	emit(t, st, "t2", "P1", "theirs", true)

	found, err := st.Alert().DismissForTrader("t2", mine, 1)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("t2 dismissed t1's alert — IDOR")
	}
	if got := feedIDs(t, st, "t1"); len(got) != 1 || got[0] != mine {
		t.Errorf("t1's alert must be untouched, got %v", got)
	}
}

// Dismissed alerts must also leave the bell badge.
func TestUnackedCountIgnoresDismissed(t *testing.T) {
	st := alertStore(t)
	id := emit(t, st, "t1", "P1", "e1", false)
	if n, _ := st.Alert().UnackedCount("t1"); n != 1 {
		t.Fatalf("badge = %d, want 1", n)
	}
	// Unacked P1 may be dismissed (only P0 is protected).
	if _, err := st.Alert().DismissForTrader("t1", id, 1); err != nil {
		t.Fatal(err)
	}
	if n, _ := st.Alert().UnackedCount("t1"); n != 0 {
		t.Errorf("badge = %d after dismissal, want 0 — a hidden alert must not keep the bell lit", n)
	}
}

// 5e — acknowledged P2 noise ages out; P0/P1 never do.
func TestPruneOnlyTouchesOldAckedP2(t *testing.T) {
	st := alertStore(t)
	oldP2 := emit(t, st, "t1", "P2", "old-digest", true)
	keepP1 := emit(t, st, "t1", "P1", "old-p1", true)
	if _, err := st.Alert().Emit(&AlertDB{
		TraderID: "t1", Level: "P2", EventID: "fresh", Kind: "k",
		Acked: true, CreatedAt: 1_900_000_000,
	}); err != nil {
		t.Fatal(err)
	}

	n, err := st.Alert().PruneAckedOlderThan("t1", 1_800_000_000, 1_900_000_100)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("pruned %d, want only the old acknowledged P2", n)
	}
	got := feedIDs(t, st, "t1")
	for _, id := range got {
		if id == oldP2 {
			t.Error("the old acknowledged P2 should have been pruned")
		}
	}
	found := false
	for _, id := range got {
		if id == keepP1 {
			found = true
		}
	}
	if !found {
		t.Error("P1 must never be auto-pruned — those are the ones worth scrolling back to")
	}
}
