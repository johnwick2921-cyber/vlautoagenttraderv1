package ninjatrader

import (
	"errors"
	"net"
	"testing"
	"time"
)

// ── W-ONE-BUTTON M2, review F3 (CTO MUST-FIX on #182) ─────────────────────
//
// ErrEntryHeld tells SendSignal's caller its entry was PROVABLY UNSENT, and the
// caller settles on it (IsMaintenanceHold → Picture 'refused', AI records
// nothing). An entry a concurrent flush had already STARTED writing may be at
// NT8: its drop must come back as something that is NOT ErrEntryHeld.

// Production path: the queue holds the attempted copy a concurrent flush left
// behind when it failed mid-write; the hold has landed; SendSignal for that id.
func TestSendSignalOwnDropAfterAnAttemptIsNotReportedUnsent(t *testing.T) {
	s, _ := pipeServer(t)
	s.SetEntryHoldCheck(func() bool { return true })
	s.pendingMu.Lock()
	s.pending = append(s.pending, timedSignal{payload: qsig("sig-a"), timestamp: time.Now(), attempted: true})
	s.pendingMu.Unlock()
	err := s.SendSignal(qsig("sig-a"))
	if err == nil {
		t.Fatal("a dropped entry must be reported to its caller")
	}
	if errors.Is(err, ErrEntryHeld) {
		t.Fatalf("an entry whose write was already started may be at NT8 — it must NOT be reported as provably unsent: %v", err)
	}
}

// And the never-attempted own drop is still ErrEntryHeld (unchanged).
func TestSendSignalOwnDropNeverAttemptedIsErrEntryHeld(t *testing.T) {
	s, _ := pipeServer(t)
	s.SetEntryHoldCheck(func() bool { return true })
	if err := s.SendSignal(qsig("sig-b")); !errors.Is(err, ErrEntryHeld) {
		t.Fatalf("want ErrEntryHeld, got %v", err)
	}
}

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
