// B5 — dead-man watchdog: disconnect blocks entries; a reconnect alone does NOT
// resume (must reconcile first); a clean reconcile resumes.
package trader

import "testing"

func TestDeadManWatchdog_BothPaths(t *testing.T) {
	w := &deadManWatchdog{}
	if w.entriesBlocked() {
		t.Fatal("fresh watchdog (live) must allow entries")
	}

	// Path 1 — disconnect blocks entries.
	if st, changed := w.observe(false); !changed || st != dmDisconnected || !w.entriesBlocked() {
		t.Fatalf("disconnect must block entries; state=%v changed=%v", st, changed)
	}

	// Reconnect ALONE must NOT resume — the link must reconcile first.
	if st, changed := w.observe(true); !changed || st != dmAwaitingReconcile || !w.entriesBlocked() {
		t.Fatalf("reconnect alone must stay blocked (awaiting reconcile); state=%v", st)
	}

	// Path 2 — a clean reconcile resumes.
	if !w.reconciled() || w.entriesBlocked() {
		t.Fatal("clean reconcile must resume entries")
	}
	// Reconcile while already live → no-op.
	if w.reconciled() {
		t.Fatal("reconcile while live must be a no-op")
	}
}

func TestDeadManWatchdog_FlapAndStrayReconcile(t *testing.T) {
	w := &deadManWatchdog{}
	w.observe(false) // disconnected
	w.observe(true)  // awaiting reconcile

	// Flap back down before reconciling → returns to disconnected, still blocked.
	if st, _ := w.observe(false); st != dmDisconnected || !w.entriesBlocked() {
		t.Fatalf("flap back down must return to disconnected; state=%v", st)
	}
	// A stray reconcile while disconnected must NOT resume.
	if w.reconciled() || !w.entriesBlocked() {
		t.Fatal("reconcile while disconnected must not resume")
	}
	// Reconnect → awaiting → reconcile → live.
	w.observe(true)
	if !w.reconciled() || w.entriesBlocked() {
		t.Fatal("reconnect then reconcile must resume")
	}
}
