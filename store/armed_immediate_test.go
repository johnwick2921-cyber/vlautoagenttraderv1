package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConfirmCancelImmediateNotDeferred is the P2-4 pin (audit 2026-09-26):
// ConfirmCancel must take the WRITE lock UP FRONT (BEGIN IMMEDIATE). The old
// deferred transaction reads first; if another writer commits between that
// read and the write upgrade (WAL), the upgrade fails with SQLITE_BUSY/
// BUSY_SNAPSHOT even though busy_timeout is set — the exact lost-close shape
// behind the #227 immediate-tx helper. This test holds a RESERVED lock on a
// side connection, commits it mid-call, and pins that the confirmation still
// applies.
func TestConfirmCancelImmediateNotDeferred(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "ccimmed.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ledger := st.ArmedOrders()
	row := &ArmedOrderDB{
		TraderID: "t1", PlanID: "p1", Version: 1, Session: "ASIA",
		Scenario: "S1", Side: "long", EntryPx: 100, StopPx: 99, TargetPx: 102,
		State: "working", SignalID: "sig-1", Kind: "limit",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.RequestCancel(row.ID, "test cancel", time.Now().UnixMilli()); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	sqlDB, err := st.GormDB().DB()
	if err != nil {
		t.Fatal(err)
	}
	holder, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Close() }()
	if _, err := holder.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("holder BEGIN IMMEDIATE: %v", err)
	}
	if _, err := holder.ExecContext(ctx, "INSERT INTO system_config (key, value) VALUES ('ccimmed-holder', '1')"); err != nil {
		t.Fatalf("holder INSERT: %v", err)
	}

	type out struct {
		err error
	}
	res := make(chan out, 1)
	go func() {
		res <- out{err: ledger.ConfirmCancel(row.ID, 4242, "settled by test")}
	}()

	// The holder commits while the confirmation is in flight — exactly the
	// window a deferred read-then-write cannot survive.
	time.Sleep(300 * time.Millisecond)
	if _, err := holder.ExecContext(ctx, "COMMIT"); err != nil {
		t.Fatalf("holder COMMIT: %v", err)
	}

	select {
	case r := <-res:
		if r.err != nil {
			t.Fatalf("ConfirmCancel failed under a committed interleaver — the deferred read→write upgrade lost the row to SQLITE_BUSY/SNAPSHOT: %v", r.err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("ConfirmCancel never returned")
	}

	var got ArmedOrderDB
	if err := st.GormDB().First(&got, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.State != StateCancelled || got.CancelSettledSnapshotID != 4242 || !strings.Contains(got.StateReason, "settled by test") {
		t.Fatalf("the cancel must settle: state=%s snapshot=%d reason=%q", got.State, got.CancelSettledSnapshotID, got.StateReason)
	}
}
