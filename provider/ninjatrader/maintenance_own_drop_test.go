package ninjatrader

import (
	"errors"
	"net"
	"testing"
	"time"
)

// ── W-ONE-BUTTON M2.1 (review F3) — an ATTEMPTED own drop is not "unsent" ──
//
// SendSignal reports its own entry's drop to its caller. ErrEntryHeld means
// PROVABLY UNSENT (IsMaintenanceHold → the Picture row settles 'refused', the
// AI path records nothing). An entry a concurrent flush had already STARTED to
// write may be at NT8, so its drop must come back as a distinct, ambiguous
// error that is NOT a hold refusal — its records stay pending and the gate
// stays closed (CTO condition 1 on M-2).

func TestFlushReportsWhetherEachDroppedEntryWasAttempted(t *testing.T) {
	s := NewTCPServer(nil)
	srv, cli := net.Pipe()
	_ = cli.Close()
	s.connMu.Lock()
	s.conn = srv
	s.connMu.Unlock()
	s.pendingMu.Lock()
	s.pending = append(s.pending, timedSignal{payload: qsig("sig-tried"), timestamp: time.Now()},
		timedSignal{payload: qsig("sig-clean"), timestamp: time.Now()})
	s.pendingMu.Unlock()
	_ = s.flushPending() // sig-tried: write started and failed; sig-clean: untouched
	s.SetEntryHoldCheck(func() bool { return true })
	drops, err := s.flushPendingReport()
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := drops["sig-tried"]; !ok || !a {
		t.Fatalf("sig-tried must be reported dropped AND attempted: %v", drops)
	}
	if a, ok := drops["sig-clean"]; !ok || a {
		t.Fatalf("sig-clean must be reported dropped, NOT attempted: %v", drops)
	}
}

func TestOwnDropErrorSeparatesUnsentFromAmbiguous(t *testing.T) {
	drops := map[string]bool{"sig-tried": true, "sig-clean": false}
	if err := ownDropError("sig-clean", drops); !errors.Is(err, ErrEntryHeld) {
		t.Fatalf("a never-attempted own drop is provably unsent → ErrEntryHeld, got %v", err)
	}
	err := ownDropError("sig-tried", drops)
	if !errors.Is(err, ErrEntryDropAmbiguous) {
		t.Fatalf("an attempted own drop must be ErrEntryDropAmbiguous, got %v", err)
	}
	if errors.Is(err, ErrEntryHeld) {
		t.Fatal("an attempted own drop must NOT read as a hold refusal (it may be at NT8)")
	}
	if err := ownDropError("sig-other", drops); err != nil {
		t.Fatalf("an entry that was not dropped gets no error, got %v", err)
	}
}
