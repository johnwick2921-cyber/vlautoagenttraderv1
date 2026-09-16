package ninjatrader

import (
	"net"
	"testing"
	"time"
)

// W2 (SUNDAY-SHIELD 2026-08-29) — the mid-flush conn-death fixture: when the
// wire dies partway through a flush, EVERY unsent signal (the one in flight
// AND the remaining tail) must be re-queued for the next reconnect flush.
// The pre-fix code re-queued only the in-flight signal and silently ate the
// tail — armed entries queued behind a dead conn could vanish.
func TestFlushPendingRequeuesUnsentTailOnConnDeath(t *testing.T) {
	s := NewTCPServer(nil)

	// Queue three signals behind a conn whose peer is already dead — the
	// first WriteFrame fails immediately (ErrClosedPipe).
	srv, cli := net.Pipe()
	_ = cli.Close() // peer gone → writes on srv error out
	s.connMu.Lock()
	s.conn = srv
	s.connMu.Unlock()

	sig := func(id string) SignalPayload {
		return SignalPayload{Symbol: "MNQ", Side: "short", Quantity: 1, SignalID: id, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	s.pendingMu.Lock()
	for _, id := range []string{"sig-a", "sig-b", "sig-c"} {
		s.pending = append(s.pending, timedSignal{payload: sig(id), timestamp: time.Now()})
	}
	s.pendingMu.Unlock()

	err := s.flushPending()
	if err == nil {
		t.Fatal("flush against a dead peer must error")
	}

	// ALL THREE must be re-queued, in order (failed first, then the tail).
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if len(s.pending) != 3 {
		t.Fatalf("want 3 re-queued signals, got %d (tail was dropped)", len(s.pending))
	}
	want := []string{"sig-a", "sig-b", "sig-c"}
	for i, ts := range s.pending {
		if ts.payload.SignalID != want[i] {
			t.Fatalf("requeue[%d] = %q, want %q (order or content wrong)", i, ts.payload.SignalID, want[i])
		}
	}
	if s.conn != nil {
		t.Fatal("closeConn must clear the dead conn after the failed flush")
	}
}
