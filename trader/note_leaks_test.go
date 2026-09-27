package trader

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// NOTE-leak (armed_event_pass.go:67-72) RED→GREEN — the old close() armed a
// 5s time.After on EVERY call, including the already-finished fast path: a
// per-Stop timer allocation that lingers armed for 5s. The fix returns before
// allocating any timer when the loop has already finished, and stops the timer
// on the slow path. RED: AllocsPerRun pins the allocation — old code allocates
// the timer + channel per close, new code allocates nothing.
func TestArmedEventLoopCloseFastPathAllocatesNoTimer(t *testing.T) {
	l := &armedEventLoop{kick: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	close(l.done) // the pass already finished — the fast path
	allocs := testing.AllocsPerRun(200, func() { l.close() })
	if allocs >= 1 {
		t.Fatalf("a finished loop's close must not allocate a timer: %.1f allocs/run", allocs)
	}
}

// NOTE-leak (auto_trader.go:1081) RED→GREEN — the drawdown monitor must stop
// when the per-Run cancel fires (the grid-init error path calls the same
// cancelStopMonitor as Stop). Start the monitor, cancel, and the goroutine
// count must return to baseline — the old channel could only be closed by
// Stop, so the grid-init error path orphaned the monitor.
func TestDrawdownMonitorStopsOnCancel(t *testing.T) {
	at := &AutoTrader{}
	at.stopMonitorMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	at.stopMonitorCtx = ctx
	at.stopMonitorCancel = cancel
	at.stopMonitorMu.Unlock()

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()

	at.startDrawdownMonitor()
	time.Sleep(50 * time.Millisecond) // let the monitor reach its select
	at.cancelStopMonitor()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if got := runtime.NumGoroutine(); got <= base {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("drawdown monitor ignored the cancel: %d goroutines (base %d)", runtime.NumGoroutine(), base)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
