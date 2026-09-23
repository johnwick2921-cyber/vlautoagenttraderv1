package ninjatrader

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

// ── W-ONE-BUTTON M2 — queued entries the hold drops are REPORTED ───────────
//
// A dropped entry leaves its caller's records (an armed row place_pending, a
// Picture row, the AI path's pending map) describing an order NT8 never got.
// The server therefore reports every drop to its sinks, with ATTEMPTED: true
// once a write of that signal's frame was STARTED on any connection (bytes may
// have reached the AddOn even when the write errored), false only when zero
// bytes were ever written — the one case the caller may settle as never sent
// (CTO condition 1). The hold also drops the queue while DISCONNECTED: a
// queued entry must not sit in the queue for as long as NT8 is down.

type dropRecorder struct {
	mu  sync.Mutex
	got []DroppedEntry
}

func (r *dropRecorder) sink(d DroppedEntry) { r.mu.Lock(); r.got = append(r.got, d); r.mu.Unlock() }
func (r *dropRecorder) all() []DroppedEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]DroppedEntry(nil), r.got...)
}

func TestAttemptedMeansAWriteOfThatSignalWasStarted(t *testing.T) {
	s := NewTCPServer(nil)
	srv, cli := net.Pipe()
	_ = cli.Close() // every write to srv now fails
	s.connMu.Lock()
	s.conn = srv
	s.connMu.Unlock()
	s.pendingMu.Lock()
	for _, id := range []string{"sig-first", "sig-tail"} {
		s.pending = append(s.pending, timedSignal{payload: qsig(id), timestamp: time.Now()})
	}
	s.pendingMu.Unlock()
	if err := s.flushPending(); err == nil {
		t.Fatal("fixture: the write to a closed pipe must fail")
	}
	// sig-first was written to (and failed); sig-tail was never touched.
	rec := &dropRecorder{}
	s.AddDroppedEntrySink("test", rec.sink)
	s.SetEntryHoldCheck(func() bool { return true })
	if _, err := s.flushPendingReport(); err != nil {
		t.Fatalf("a held flush drops, it does not error: %v", err)
	}
	got := map[string]bool{}
	for _, d := range rec.all() {
		got[d.SignalID] = d.Attempted
	}
	if a, ok := got["sig-first"]; !ok || !a {
		t.Fatalf("sig-first had a write started — it must be reported attempted=true: %+v", rec.all())
	}
	if a, ok := got["sig-tail"]; !ok || a {
		t.Fatalf("sig-tail never had a byte written — it must be reported attempted=false: %+v", rec.all())
	}
	if n := s.PendingSignalCount(); n != 0 {
		t.Fatalf("dropped entries must leave the queue, %d remain", n)
	}
}

// Disconnected + held: the queue is dropped (and reported) at once — the
// maintenance loop does it without waiting for a reconnect — and nothing is
// written when the AddOn comes back.
func TestHeldQueueIsDroppedWhileDisconnectedAndNeverResent(t *testing.T) {
	pt := maintenanceTick
	maintenanceTick = 20 * time.Millisecond
	t.Cleanup(func() { maintenanceTick = pt })
	s := startedServer(t, nil)
	rec := &dropRecorder{}
	s.AddDroppedEntrySink("test", rec.sink)
	s.pendingMu.Lock()
	s.pending = append(s.pending, timedSignal{payload: qsig("sig-q"), timestamp: time.Now()})
	s.pendingMu.Unlock()
	s.SetEntryHoldCheck(func() bool { return true })
	// The queue empties under its lock and the report follows it: wait for both.
	deadline := time.Now().Add(2 * time.Second)
	for (s.PendingSignalCount() > 0 || len(rec.all()) == 0) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := s.PendingSignalCount(); n != 0 {
		t.Fatalf("a held, disconnected queue must be dropped by the maintenance loop; %d remain", n)
	}
	ds := rec.all()
	if len(ds) != 1 || ds[0].SignalID != "sig-q" || ds[0].Attempted {
		t.Fatalf("the drop must be reported once, never attempted: %+v", ds)
	}
	s.SetEntryHoldCheck(func() bool { return false })
	w := dialWire(t, s, HelloPayload{})
	if env, ok := w.next(FrameSignal, 400*time.Millisecond); ok {
		t.Fatalf("a dropped entry was resent on reconnect: %s", env.Payload)
	}
}

// SendSignal while held and disconnected: its own entry is dropped at once and
// reported to the caller (ErrEntryHeld) — it is never parked in the queue.
func TestSendSignalHeldWhileDisconnectedReturnsErrEntryHeld(t *testing.T) {
	s := NewTCPServer(nil)
	rec := &dropRecorder{}
	s.AddDroppedEntrySink("test", rec.sink)
	s.SetEntryHoldCheck(func() bool { return true })
	err := s.SendSignal(qsig("sig-send"))
	if err == nil || !errors.Is(err, ErrEntryHeld) {
		t.Fatalf("want ErrEntryHeld, got %v", err)
	}
	if s.PendingSignalCount() != 0 || len(rec.all()) != 1 {
		t.Fatalf("the entry must be dropped and reported, not queued: pending=%d drops=%+v", s.PendingSignalCount(), rec.all())
	}
}
