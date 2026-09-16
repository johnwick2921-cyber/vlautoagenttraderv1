package ninjatrader

import (
	"sync/atomic"
	"testing"
	"time"
)

// ClosedBarsOnly: closed bars persist, forming bars never do.
func TestClosedBarsOnly(t *testing.T) {
	// 1m bars with OPEN time T: a bar closes at T+60s.
	now := int64(1_800_000_000_000)
	closed := Bar{T: now - 90_000}
	forming := Bar{T: now - 10_000}
	got := ClosedBarsOnly([]Bar{closed, forming}, "1m", now)
	if len(got) != 1 || got[0].T != closed.T {
		t.Fatalf("ClosedBarsOnly = %+v want only the closed bar", got)
	}
}

// fanOutBarPersist: a panicking persister is recovered and never propagates;
// the drain loop stays alive (the warn sink fires).
func TestFanOutBarPersistWorkerDrainsAndSurvivesPanic(t *testing.T) {
	var called atomic.Int64
	SetBarPersister(func(historical bool, symbol, tf string, bars []Bar) {
		called.Add(1)
		panic("injected persister failure")
	})
	defer SetBarPersister(nil)
	fanOutBarPersist(nil, false, "MNQ", "1m", []Bar{{T: 1}})
	fanOutBarPersist(nil, false, "MNQ", "1m", []Bar{{T: 2}})
	deadline := time.Now().Add(2 * time.Second)
	for called.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if called.Load() < 2 {
		t.Fatal("persist worker did not drain both messages (a panic must not kill the worker)")
	}
}

// fanOutBarPersist: no persister installed → no-op, no panic.
func TestFanOutBarPersistNoop(t *testing.T) {
	SetBarPersister(nil)
	fanOutBarPersist(func(string, ...interface{}) {}, false, "MNQ", "1m", []Bar{{T: 1}})
}

// BAR-TRUTH WAVE (2026-08-28): a full queue drops with a COUNTER (no per-drop
// WARN flood); the summary line fires at most 1/min. Drop self-heals because
// the next live frame re-derives the closed cache tail.
func TestFanOutBarPersistQueueFullDropsCounted(t *testing.T) {
	block := make(chan struct{})
	SetBarPersister(func(historical bool, symbol, tf string, bars []Bar) {
		<-block // slow persister keeps the worker busy
	})
	defer func() { close(block); SetBarPersister(nil) }()
	persistDropped.Store(0)
	persistDroppedCloses.Store(0)
	// FORMING bars (open time = now) take the drop path; close-carrying batches
	// now wait instead (F2, closes are sacred) — so this flood is intrabar-only.
	nowMs := time.Now().UnixMilli()
	// The worker consumes up to 256 into its batch before the blocking flush —
	// flood well past channel cap + batch so drops are guaranteed.
	for i := 0; i < barPersistQueueCap*2; i++ {
		fanOutBarPersist(nil, false, "MNQ", "1m", []Bar{{T: nowMs}})
	}
	if persistDropped.Load() == 0 {
		t.Fatal("expected queue-full drops to be counted")
	}
	if persistDroppedCloses.Load() != 0 {
		t.Fatalf("closes_dropped = %d — forming bars must never be counted as closes", persistDroppedCloses.Load())
	}
}
