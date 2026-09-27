package ninjatrader

import (
	"sync/atomic"
	"testing"
	"time"
)

// ── W-ONE-BUTTON M2.1 (review F1) — the hold is re-checked AFTER the writer
// lock. The flush used to check the hold, then wait for writeMu (contended at
// connect by the hello reply and the heartbeat), then write: a hold landing in
// that wait did not stop the entry. The check now happens with the lock held.
func TestFlushRechecksTheHoldAfterWaitingForTheWriter(t *testing.T) {
	s, frames := pipeServer(t)
	var held atomic.Bool
	s.SetEntryHoldCheck(func() bool { return held.Load() })
	rec := &dropRecorder{}
	s.AddDroppedEntrySink("test", rec.sink)
	s.pendingMu.Lock()
	s.pending = append(s.pending, timedSignal{payload: qsig("sig-wait"), timestamp: time.Now()})
	s.pendingMu.Unlock()

	s.writeMu.Lock() // the writer is busy (hello reply / heartbeat at connect)
	done := make(chan struct{})
	go func() { _ = s.flushPending(); close(done) }()
	time.Sleep(100 * time.Millisecond) // the flush has passed its first check and waits on writeMu
	held.Store(true)                   // the hold lands while it waits
	s.writeMu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the flush never finished")
	}
	select {
	case f := <-frames:
		t.Fatalf("an entry reached the wire after the hold landed (frame %s)", f)
	case <-time.After(200 * time.Millisecond):
	}
	if ds := rec.all(); len(ds) != 1 || ds[0].SignalID != "sig-wait" || ds[0].Attempted {
		t.Fatalf("the entry must be dropped and reported, never attempted: %+v", ds)
	}
}
