package ninjatrader

import (
	"errors"
	"net"
	"testing"
	"time"
)

// W-ONE-BUTTON M2 — site-map gap U2: an entry SendSignal queued while NT8 was
// disconnected returns nil, releases its permit, and is written on the next
// reconnect flush by a goroutine that holds no permit. So the queue itself
// must honour the maintenance hold: queued entries are DROPPED (never
// re-queued) while held, and a SendSignal whose own entry was dropped returns
// ErrEntryHeld so the caller never records a placement that did not happen.

func qsig(id string) SignalPayload {
	return SignalPayload{Symbol: "MNQ", Side: "long", Quantity: 1, SignalID: id, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
}

// pipeServer returns a server whose conn is one end of a pipe; the other end's
// frames are counted.
func pipeServer(t *testing.T) (*TCPServer, chan FrameType) {
	t.Helper()
	s := NewTCPServer(nil)
	srv, cli := net.Pipe()
	t.Cleanup(func() { _ = srv.Close(); _ = cli.Close() })
	s.connMu.Lock()
	s.conn = srv
	s.connMu.Unlock()
	got := make(chan FrameType, 16)
	go func() {
		for {
			env, err := ReadFrame(cli)
			if err != nil {
				return
			}
			got <- env.Type
		}
	}()
	return s, got
}

func TestQueuedEntriesAreDroppedNotFlushedWhileHeld(t *testing.T) {
	s, got := pipeServer(t)
	s.SetEntryHoldCheck(func() bool { return true })
	s.pendingMu.Lock()
	for _, id := range []string{"q-a", "q-b"} {
		s.pending = append(s.pending, timedSignal{payload: qsig(id), timestamp: time.Now()})
	}
	s.pendingMu.Unlock()
	if err := s.flushPending(); err != nil {
		t.Fatalf("a held flush drops, it does not error: %v", err)
	}
	select {
	case f := <-got:
		t.Fatalf("a queued entry reached the wire while held (frame %s)", f)
	case <-time.After(150 * time.Millisecond):
	}
	if n := s.PendingSignalCount(); n != 0 {
		t.Fatalf("held flush must drop, not re-queue: %d still pending", n)
	}
}

func TestQueuedEntriesFlushWhenNotHeld(t *testing.T) {
	s, got := pipeServer(t)
	s.SetEntryHoldCheck(func() bool { return false })
	if err := s.SendSignal(qsig("q-ok")); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-got:
		if f != FrameSignal {
			t.Fatalf("want a signal frame, got %s", f)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("not held: the entry never reached the wire")
	}
}

// The hold lands between the caller's permit and the flush: SendSignal must
// report ITS entry as not sent.
func TestSendSignalReportsItsOwnEntryDroppedByAHold(t *testing.T) {
	s, got := pipeServer(t)
	s.SetEntryHoldCheck(func() bool { return true })
	err := s.SendSignal(qsig("q-mine"))
	if !errors.Is(err, ErrEntryHeld) {
		t.Fatalf("want ErrEntryHeld for a dropped own entry, got %v", err)
	}
	select {
	case f := <-got:
		t.Fatalf("frame %s reached the wire", f)
	case <-time.After(100 * time.Millisecond):
	}
}

// Disconnected: SendSignal queues (today's behaviour) and the queue depth is
// visible to the installation gate.
func TestPendingSignalCountSeesTheQueue(t *testing.T) {
	s := NewTCPServer(nil)
	if err := s.SendSignal(qsig("q-1")); err != nil {
		t.Fatalf("disconnected SendSignal queues and returns nil today: %v", err)
	}
	if n := s.PendingSignalCount(); n != 1 {
		t.Fatalf("PendingSignalCount=%d, want 1", n)
	}
}

// Unset hold check = today's behaviour (never held).
func TestUnsetEntryHoldCheckIsNotHeld(t *testing.T) {
	s, got := pipeServer(t)
	if err := s.SendSignal(qsig("q-unset")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("unset hold check must not hold")
	}
}
