package ninjatrader

import (
	"fmt"
	"testing"
	"time"
)

// W-PICTURE-HTF (2026-09-20, CTO round 2) — coordinated order_update fan-out.
// subscribeFor REPLACES the (symbol, account) channel, so two in-process
// consumers subscribing directly (the armed executor and the picture broker
// consumer) would close each other's channel and one would silently stop
// receiving. ListenOrderUpdates owns the ONE direct subscription per key and
// copies every received frame to each listener; no listener can evict
// another, and a listener's channel closes only when the underlying
// subscription dies.

func recvOrderUpdate(t *testing.T, ch <-chan OrderUpdatePayload) OrderUpdatePayload {
	t.Helper()
	select {
	case u, open := <-ch:
		if !open {
			t.Fatal("listener channel closed while the fan-out should be live")
		}
		return u
	case <-time.After(3 * time.Second):
		t.Fatal("no frame on the listener channel")
		return OrderUpdatePayload{}
	}
}

func TestOrderUpdateFanoutCoexistsNoEviction(t *testing.T) {
	s := NewTCPServer(nil)
	l1, rm1 := s.ListenOrderUpdates("MNQ", "Sim101")
	l2, rm2 := s.ListenOrderUpdates("MNQ", "Sim101")
	_ = rm1

	// Both listeners receive every frame the router delivers.
	for i := 0; i < 3; i++ {
		s.orderUpdCh <- OrderUpdatePayload{SignalID: fmt.Sprintf("s%d", i), Symbol: "MNQ", Account: "Sim101"}
	}
	for i := 0; i < 3; i++ {
		want := fmt.Sprintf("s%d", i)
		if u := recvOrderUpdate(t, l1); u.SignalID != want {
			t.Fatalf("l1 frame %d = %q, want %q", i, u.SignalID, want)
		}
		if u := recvOrderUpdate(t, l2); u.SignalID != want {
			t.Fatalf("l2 frame %d = %q, want %q", i, u.SignalID, want)
		}
	}

	// A listener joining mid-stream receives subsequent frames without
	// disturbing the existing ones.
	l3, rm3 := s.ListenOrderUpdates("MNQ", "Sim101")
	s.orderUpdCh <- OrderUpdatePayload{SignalID: "s3", Symbol: "MNQ", Account: "Sim101"}
	if u := recvOrderUpdate(t, l3); u.SignalID != "s3" {
		t.Fatalf("late listener got %q", u.SignalID)
	}
	if u := recvOrderUpdate(t, l1); u.SignalID != "s3" {
		t.Fatalf("l1 lost the frame after l3 joined: %q", u.SignalID)
	}

	// Removing one listener closes ONLY that listener. (The removed listener
	// may still hold buffered frames it had not consumed — drain until the
	// close is observed.)
	rm2()
	deadline := time.Now().Add(2 * time.Second)
	closed := false
	for time.Now().Before(deadline) {
		select {
		case _, open := <-l2:
			if !open {
				closed = true
			}
		default:
		}
		if closed {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !closed {
		t.Fatal("removed listener channel was not closed")
	}
	s.orderUpdCh <- OrderUpdatePayload{SignalID: "s4", Symbol: "MNQ", Account: "Sim101"}
	if u := recvOrderUpdate(t, l1); u.SignalID != "s4" {
		t.Fatalf("l1 lost a frame after l2's removal: %q", u.SignalID)
	}
	if u := recvOrderUpdate(t, l3); u.SignalID != "s4" {
		t.Fatalf("l3 lost a frame after l2's removal: %q", u.SignalID)
	}
	_ = rm3

	// A legacy DIRECT subscribe replaces the fan-out's underlying channel:
	// every listener closes (the consumers' self-heal signal), never silently
	// dying on an open channel.
	s.SubscribeOrderUpdatesFor("MNQ", "Sim101")
	deadline = time.Now().Add(3 * time.Second)
	for {
		closed := func() bool {
			select {
			case _, open := <-l1:
				return !open
			default:
				return false
			}
		}()
		if closed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("l1 did not close when the direct subscription replaced the fan-out source")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if closed := func() bool {
		select {
		case _, open := <-l3:
			return !open
		default:
			return false
		}
	}(); !closed {
		t.Fatal("l3 did not close when the direct subscription replaced the fan-out source")
	}

	// Re-listening after the teardown re-establishes a live subscription.
	l4, rm4 := s.ListenOrderUpdates("MNQ", "Sim101")
	defer rm4()
	s.orderUpdCh <- OrderUpdatePayload{SignalID: "s5", Symbol: "MNQ", Account: "Sim101"}
	if u := recvOrderUpdate(t, l4); u.SignalID != "s5" {
		t.Fatalf("re-listened channel got %q", u.SignalID)
	}
}

// Different accounts on one symbol are separate fan-outs: a listener for one
// account never receives the other account's frames and is never evicted by
// the other key's subscription.
func TestOrderUpdateFanoutPerAccountIsolation(t *testing.T) {
	s := NewTCPServer(nil)
	la, _ := s.ListenOrderUpdates("MNQ", "Sim101")
	lb, _ := s.ListenOrderUpdates("MNQ", "SimAccount1")
	s.orderUpdCh <- OrderUpdatePayload{SignalID: "a1", Symbol: "MNQ", Account: "Sim101"}
	s.orderUpdCh <- OrderUpdatePayload{SignalID: "b1", Symbol: "MNQ", Account: "SimAccount1"}
	if u := recvOrderUpdate(t, la); u.SignalID != "a1" {
		t.Fatalf("Sim101 listener got %q", u.SignalID)
	}
	if u := recvOrderUpdate(t, lb); u.SignalID != "b1" {
		t.Fatalf("SimAccount1 listener got %q", u.SignalID)
	}
}
