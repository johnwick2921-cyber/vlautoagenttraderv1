package ninjatrader

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// W117 slice A — the ordered-execution worker pins. Every test drives a REAL
// TCPServer through a REAL connected fake client (startedServer + dialWire
// from maintenance_wire_test.go) so the production read loop enqueues the
// frames itself — no stubbed handlers, no re-built inputs (canon 53).

type recHandlers struct {
	mu     sync.Mutex
	events []string
}

func (r *recHandlers) order(u OrderUpdatePayload) {
	r.mu.Lock()
	r.events = append(r.events, "order:"+u.SignalID)
	r.mu.Unlock()
}
func (r *recHandlers) fill(f FillPayload) {
	r.mu.Lock()
	r.events = append(r.events, "fill:"+f.SignalID)
	r.mu.Unlock()
}
func (r *recHandlers) close_(p PositionClosePayload) {
	r.mu.Lock()
	r.events = append(r.events, "close:"+p.SignalID)
	r.mu.Unlock()
}
func (r *recHandlers) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

func (r *recHandlers) snapshotLen() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func waitEvents(t *testing.T, r *recHandlers, n int, timeout time.Duration) []string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if ev := r.snapshot(); len(ev) >= n {
			return ev
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d ordered events; got %d: %v", n, len(r.snapshot()), r.snapshot())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// R1 — receive order preserved, one worker per owner, applied exactly once.
func TestOrderedWorkerAppliesFramesInReceiveOrder(t *testing.T) {
	s := startedServer(t, nil)
	w := dialWire(t, s, HelloPayload{})
	rec := &recHandlers{}
	unreg, err := s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{
		Order: rec.order, Fill: rec.fill, Close: rec.close_,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer unreg()

	// Interleave the three frame kinds exactly as NT8 can emit them.
	frames := []struct {
		ft  FrameType
		any any
	}{
		{FrameOrderUpdate, OrderUpdatePayload{SignalID: "sig-1", State: "accepted", Symbol: "MNQ", Account: "Sim101", Quantity: 1}},
		{FrameFill, FillPayload{SignalID: "sig-1", Symbol: "MNQ", Account: "Sim101", Side: "long", Quantity: 1, Status: "filled", FillPrice: 29347.25}},
		{FrameOrderUpdate, OrderUpdatePayload{SignalID: "sig-2", State: "accepted", Symbol: "MNQ", Account: "Sim101", Quantity: 1}},
		{FramePositionClose, PositionClosePayload{SignalID: "sig-1", Symbol: "MNQ", PositionSide: "long", Account: "Sim101", Quantity: 1, ExitPrice: 29350}},
		{FrameOrderUpdate, OrderUpdatePayload{SignalID: "sig-3", State: "working", Symbol: "MNQ", Account: "Sim101", Quantity: 1}},
		{FrameFill, FillPayload{SignalID: "sig-2", Symbol: "MNQ", Account: "Sim101", Side: "short", Quantity: 1, Status: "filled", FillPrice: 29340}},
	}
	// The sig-3 order_update has NO post-update snapshot: the worker applies it
	// after the bounded wait with the book suppressed — the frame is still
	// applied (order evidence is never dropped), so it appears in the order list.
	// All other order_updates get snapshots right after (the AddOn's real order).
	for _, f := range frames {
		if err := WriteFrame(w.conn, f.ft, f.any); err != nil {
			t.Fatalf("write %s: %v", f.ft, err)
		}
		if f.ft == FrameOrderUpdate && f.any.(OrderUpdatePayload).SignalID != "sig-3" {
			if err := WriteFrame(w.conn, FrameOrderSnapshot, OrderSnapshotPayload{BuildID: "test", Account: "Sim101", Orders: []NT8Order{}}); err != nil {
				t.Fatalf("write snapshot: %v", err)
			}
		}
	}

	ev := waitEvents(t, rec, 6, 5*time.Second)
	want := []string{
		"order:sig-1", "fill:sig-1", "order:sig-2", "close:sig-1",
		"order:sig-3", "fill:sig-2",
	}
	for i := range want {
		if ev[i] != want[i] {
			t.Fatalf("receive order broken at %d: got %q want %q (all: %v)", i, ev[i], want[i], ev)
		}
	}
	// Give any double-application a chance to show, then pin exactly-once.
	time.Sleep(100 * time.Millisecond)
	if got := rec.snapshot(); len(got) != len(want) {
		t.Fatalf("frames applied %d times, want exactly %d: %v", len(got), len(want), got)
	}
}

// R3 — a handler that re-registers must never wedge the worker loop (the
// callback runs with NO registry lock held).
func TestOrderedWorkerCallbackReregistersWithoutWedging(t *testing.T) {
	s := startedServer(t, nil)
	w := dialWire(t, s, HelloPayload{})
	rec := &recHandlers{}
	var once sync.Once
	replaced := make(chan struct{}, 1)
	var unreg func()
	var err error
	unreg, err = s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{
		Fill: func(f FillPayload) {
			rec.fill(f)
			once.Do(func() {
				// Re-register FROM INSIDE the callback: retire the old owner
				// (its worker drains) and install the new one. The callback
				// holds NO registry lock, so this must not wedge the loop.
				unreg()
				_, err := s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{
					Fill: rec.fill,
				})
				if err != nil {
					t.Errorf("re-register from callback: %v", err)
				}
				replaced <- struct{}{}
			})
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer unreg()

	// Write ONE fill first — its callback performs the unreg + re-register.
	// A frame that lands in the re-registration gap finds no owner and goes to
	// the advisory channel BY DESIGN (enqueueOrdered returns false), so the
	// remaining fills are written only AFTER the replacement is installed.
	writeFill := func() {
		if err := WriteFrame(w.conn, FrameFill, FillPayload{SignalID: "sig", Symbol: "MNQ", Account: "Sim101", Side: "long", Quantity: 1, Status: "filled"}); err != nil {
			t.Fatalf("write fill: %v", err)
		}
	}
	writeFill()
	select {
	case <-replaced:
	case <-time.After(15 * time.Second):
		t.Fatal("the worker wedged inside the re-registering callback")
	}
	writeFill()
	writeFill()
	waitEvents(t, rec, 3, 15*time.Second)
}

// R5 — a second LIVE owner for the same (symbol, account) is refused loudly;
// after the first unregisters, registration succeeds again.
func TestSecondLiveOwnerRefusedLoudly(t *testing.T) {
	s := startedServer(t, nil)
	_ = dialWire(t, s, HelloPayload{})
	rec := &recHandlers{}
	unreg, err := s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{Fill: rec.fill})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err = s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{Fill: rec.fill})
	if !errors.Is(err, ErrOrderedOwnerExists) {
		t.Fatalf("second live registration must be refused loudly, got: %v", err)
	}
	unreg()
	unreg2, err := s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{Fill: rec.fill})
	if err != nil {
		t.Fatalf("register after unregister: %v", err)
	}
	unreg2()
}

// R5 — unregister drains the queue then exits: nothing queued is lost, and
// nothing later is applied.
func TestUnregisterDrainsQueueThenExits(t *testing.T) {
	s := startedServer(t, nil)
	w := dialWire(t, s, HelloPayload{})
	rec := &recHandlers{}
	gate := make(chan struct{})
	release := sync.OnceFunc(func() { close(gate) })
	unreg, err := s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{
		Fill: func(f FillPayload) {
			<-gate // the worker is busy: later frames are QUEUED, not applied
			rec.fill(f)
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	for i := 0; i < 4; i++ {
		if err := WriteFrame(w.conn, FrameFill, FillPayload{SignalID: "sig", Symbol: "MNQ", Account: "Sim101", Side: "long", Quantity: 1, Status: "filled"}); err != nil {
			t.Fatalf("write fill: %v", err)
		}
	}
	// Wait until the worker is provably inside the first callback (the queue
	// holds the other three), THEN unregister. The gate guarantees the frames
	// are QUEUED, not yet applied.
	deadline := time.Now().Add(2 * time.Second)
	for rec.snapshotLen() == 0 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	unreg() // drain-then-exit: all four queued fills must still apply.
	release()
	ev := waitEvents(t, rec, 4, 3*time.Second)
	if len(ev) != 4 {
		t.Fatalf("unregister must drain the queue (4 queued), applied %d", len(ev))
	}
	// After exit, a new frame has no owner: it must NOT be applied.
	if err := WriteFrame(w.conn, FrameFill, FillPayload{SignalID: "late", Symbol: "MNQ", Account: "Sim101", Side: "long", Quantity: 1, Status: "filled"}); err != nil {
		t.Fatalf("write late fill: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if got := rec.snapshot(); len(got) != 4 {
		t.Fatalf("a frame after unregister must not be applied: %v", got)
	}
}

// R1 (legacy) — with NO owner registered, frames flow through the advisory
// channels unchanged (today's behavior, byte-identical).
func TestNoOwnerLeavesLegacyPathUnchanged(t *testing.T) {
	s := startedServer(t, nil)
	w := dialWire(t, s, HelloPayload{})
	ch := s.SubscribeFillsFor("MNQ", "Sim101")
	if err := WriteFrame(w.conn, FrameFill, FillPayload{SignalID: "sig", Symbol: "MNQ", Account: "Sim101", Side: "long", Quantity: 1, Status: "filled"}); err != nil {
		t.Fatalf("write fill: %v", err)
	}
	select {
	case f := <-ch:
		if f.OrderedOwned {
			t.Fatal("a fill with no ordered owner must not be flagged OrderedOwned")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("legacy fill delivery timed out")
	}
}

// F-B (CTO review) — the snapshot watermark is PER ACCOUNT. A snapshot for
// account B arriving between A's order_update and A's own snapshot must NOT
// satisfy A's wait: A keeps waiting (times out → BookGateNotFresh), and only
// A's own snapshot flips it to BookGateFresh. RED: the global counter.
func TestSnapshotWatermarkIsPerAccount(t *testing.T) {
	s := startedServer(t, nil)
	w := dialWire(t, s, HelloPayload{})
	got := make(chan OrderUpdatePayload, 8)
	unreg, err := s.RegisterOrderedExecutionsFor("MNQ", "Sim101", OrderedExecutionHandlers{
		Order: func(u OrderUpdatePayload) { got <- u },
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer unreg()

	// Scenario 1 — B's snapshot must NOT certify A's update.
	if err := WriteFrame(w.conn, FrameOrderUpdate, OrderUpdatePayload{SignalID: "a1", State: "accepted", Symbol: "MNQ", Account: "Sim101", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(w.conn, FrameOrderSnapshot, OrderSnapshotPayload{BuildID: "test", Account: "Sim102", Orders: []NT8Order{}}); err != nil {
		t.Fatal(err)
	}
	select {
	case u := <-got:
		t.Fatalf("B's snapshot certified A's update (BookGate=%d) — the watermark must be per-account", u.BookGate)
	case <-time.After(300 * time.Millisecond):
	}
	// A's own snapshot flips the gate: the update applies promptly as Fresh.
	if err := WriteFrame(w.conn, FrameOrderSnapshot, OrderSnapshotPayload{BuildID: "test", Account: "Sim101", Orders: []NT8Order{}}); err != nil {
		t.Fatal(err)
	}
	select {
	case u := <-got:
		if u.BookGate != BookGateFresh {
			t.Fatalf("A's own snapshot must flip the gate to Fresh, got %d", u.BookGate)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("A's update never applied after A's own snapshot")
	}

	// Scenario 2 — no A snapshot at all: A times out → BookGateNotFresh (the
	// book is suppressed, never the pre-change one).
	if err := WriteFrame(w.conn, FrameOrderUpdate, OrderUpdatePayload{SignalID: "a2", State: "working", Symbol: "MNQ", Account: "Sim101", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case u := <-got:
		if u.BookGate != BookGateNotFresh {
			t.Fatalf("timeout must stamp BookGateNotFresh, got %d", u.BookGate)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("the bounded wait never timed out")
	}
}
