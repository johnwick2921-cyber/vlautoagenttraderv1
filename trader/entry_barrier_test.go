package trader

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// W-ONE-BUTTON M2 — the in-process entry barrier. Every entry SEND holds a
// permit for the duration of the send. Hold() refuses new permits at once and
// waits until every permit taken before it is released ("drained"). Checking
// a boolean and sending later is exactly the race this exists to close.

func TestEntryBarrierGrantsPermitsWhileReleased(t *testing.T) {
	var b EntryBarrier
	rel, ok := b.Permit()
	if !ok || rel == nil {
		t.Fatal("a released barrier must grant a permit")
	}
	if b.InFlight() != 1 {
		t.Fatalf("in-flight=%d, want 1", b.InFlight())
	}
	rel()
	if b.InFlight() != 0 {
		t.Fatalf("in-flight=%d after release, want 0", b.InFlight())
	}
}

func TestEntryBarrierRefusesPermitsWhileHeld(t *testing.T) {
	var b EntryBarrier
	if err := b.Hold(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rel, ok := b.Permit(); ok || rel != nil {
		t.Fatal("a held barrier must refuse every permit")
	}
	b.Release()
	if _, ok := b.Permit(); !ok {
		t.Fatal("after Release, permits are granted again")
	}
}

// THE RACE: a send already holding a permit completes; Hold() returns only
// after it; the next send is refused. Run with -race.
func TestEntryBarrierHoldWaitsForInFlightSendAndRefusesTheNext(t *testing.T) {
	var b EntryBarrier
	rel, ok := b.Permit()
	if !ok {
		t.Fatal("permit")
	}
	var sendDone atomic.Bool
	holdReturned := make(chan struct{})
	go func() {
		if err := b.Hold(context.Background()); err != nil {
			t.Error(err)
		}
		if !sendDone.Load() {
			t.Error("Hold returned while a permitted send was still in flight")
		}
		close(holdReturned)
	}()
	// while Hold is waiting, new permits are already refused
	deadline := time.Now().Add(2 * time.Second)
	for !b.Held() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if _, ok := b.Permit(); ok {
		t.Fatal("once Hold has begun, a new permit must be refused even before drain completes")
	}
	select {
	case <-holdReturned:
		t.Fatal("Hold returned before the in-flight send released its permit")
	case <-time.After(50 * time.Millisecond):
	}
	sendDone.Store(true)
	rel()
	select {
	case <-holdReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("Hold did not return after the in-flight send finished")
	}
	if !b.Drained() {
		t.Fatal("after Hold returns the barrier reports drained")
	}
}

// Many concurrent senders racing a Hold: after Hold returns, no permit is
// outstanding and none is ever granted again until Release.
func TestEntryBarrierConcurrentSendersNeverSlipPastAHold(t *testing.T) {
	var b EntryBarrier
	var wg sync.WaitGroup
	var afterHold atomic.Bool
	var violations atomic.Int64
	stop := make(chan struct{})
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				rel, ok := b.Permit()
				if ok {
					if afterHold.Load() {
						violations.Add(1)
					}
					time.Sleep(100 * time.Microsecond)
					rel()
				}
			}
		}()
	}
	time.Sleep(5 * time.Millisecond)
	if err := b.Hold(context.Background()); err != nil {
		t.Fatal(err)
	}
	afterHold.Store(true)
	if b.InFlight() != 0 {
		t.Fatalf("in-flight=%d after Hold returned", b.InFlight())
	}
	time.Sleep(10 * time.Millisecond)
	close(stop)
	wg.Wait()
	if v := violations.Load(); v != 0 {
		t.Fatalf("%d permit(s) granted after Hold returned", v)
	}
}

// A send that never finishes must not wedge the updater forever: Hold honours
// its context, returns the error, and the barrier STAYS held (fail closed).
func TestEntryBarrierHoldTimesOutButStaysHeld(t *testing.T) {
	var b EntryBarrier
	rel, _ := b.Permit()
	defer rel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := b.Hold(ctx); err == nil {
		t.Fatal("Hold with a stuck send must return the context error")
	}
	if !b.Held() || b.Drained() {
		t.Fatalf("a timed-out Hold leaves the barrier held and not drained: held=%v drained=%v", b.Held(), b.Drained())
	}
	if _, ok := b.Permit(); ok {
		t.Fatal("still held after the timeout")
	}
}
