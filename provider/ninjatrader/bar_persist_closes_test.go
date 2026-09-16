package ninjatrader

import (
	"sync"
	"testing"
	"time"
)

// F2 (LONDON-FORENSICS 2026-08-28) — CLOSES ARE SACRED fixtures.
//
// The 06:09 event dropped 8 CLOSED bars because the queue-full path dropped
// close-carrying batches while the GORM writer was stalled. The fixed path
// bounded-blocks (backpressure) for close-carrying batches and only counts a
// close drop after a LOUD last resort. A writer stall shorter than the retry
// window must therefore produce ZERO close drops.

// ── DETERMINISM HELPERS (class 60 sweep, 2026-09-03) ────────────────────────
//
// TestFanOutClosesLastResortIsHonest failed roughly 1 run in 4-6 of the full
// suite and passed in isolation every time. Root cause: the persist queue and
// its worker are PACKAGE-LEVEL singletons (`persistOnce`, `barPersistCh`), so
// consecutive tests — and `-count=N` iterations of one test — inherit whatever
// the previous one left in the queue. The last-resort test then ASSUMED the
// queue was full after sending cap+512 frames. Under load, or with a partly
// drained queue from a neighbour, the close-carrying frame was accepted instead
// of dropped and the assertion read closes_dropped=0.
//
// The fix is to stop assuming and start establishing: begin from a
// provably-empty queue, and fill until the queue is provably FULL. No retry
// loop, no sleep-and-hope, no production change.

// idlePersistQueue detaches the persister and waits until the worker has fully
// drained, so a test starts from a known-empty singleton.
func idlePersistQueue(t *testing.T) {
	t.Helper()
	SetBarPersister(nil)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if barPersistCh == nil || len(barPersistCh) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("persist queue did not drain: %d still queued", len(barPersistCh))
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// fillPersistQueueUntilFull sends forming (non-close) frames until the queue is
// observably at capacity. Forming frames bypass the close-drop path, so this
// establishes the precondition without touching either counter.
func fillPersistQueueUntilFull(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if barPersistCh != nil && len(barPersistCh) == cap(barPersistCh) {
			return
		}
		if time.Now().After(deadline) {
			got := 0
			if barPersistCh != nil {
				got = len(barPersistCh)
			}
			t.Fatalf("could not fill the persist queue: %d/%d", got, cap(barPersistCh))
		}
		fanOutBarPersist(nil, false, "MNQ", "1m", []Bar{{T: time.Now().UnixMilli()}})
	}
}

func TestFanOutClosesSurviveWriterStall(t *testing.T) {
	idlePersistQueue(t)
	t.Cleanup(func() { idlePersistQueue(t) })

	stalled := make(chan struct{})
	release := make(chan struct{})
	var first sync.Once
	releaseOnce := sync.Once{}
	SetBarPersister(func(historical bool, symbol, tf string, bars []Bar) {
		first.Do(func() {
			close(stalled)
			<-release // only the FIRST flush stalls
		})
	})
	defer func() { releaseOnce.Do(func() { close(release) }); SetBarPersister(nil) }()
	persistDropped.Store(0)
	persistDroppedCloses.Store(0)

	closed := []Bar{{T: time.Now().Add(-2 * time.Minute).UnixMilli(), O: 1, H: 1, L: 1, C: 1}}
	var wg sync.WaitGroup
	for i := 0; i < barPersistQueueCap+20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fanOutBarPersist(nil, false, "MNQ", "1m", closed)
		}()
	}
	// The worker picks the first batch and stalls on it → the queue fills and
	// the overflow enters the bounded-blocking retry.
	<-stalled
	time.Sleep(500 * time.Millisecond)
	releaseOnce.Do(func() { close(release) }) // the stall resolves well inside the retry window
	wg.Wait()
	time.Sleep(500 * time.Millisecond) // let the worker drain

	if got := persistDroppedCloses.Load(); got != 0 {
		t.Fatalf("closes_dropped = %d, want 0 (a writer stall < the retry window must never drop closes)", got)
	}
	if got := persistDropped.Load(); got != 0 {
		t.Fatalf("queue_drops = %d, want 0 (close-carrying batches must wait, not drop)", got)
	}
}

func TestFanOutClosesLastResortIsHonest(t *testing.T) {
	// A stall LONGER than the retry window must eventually drop with the LOUD
	// ERROR path — but the counter says exactly what happened (never silent).
	//
	// SEQUENCING IS THE WHOLE TEST (class 60 sweep). A full queue is not enough:
	// while the close-carrying frame bounded-blocks for 2s×3, the worker can
	// still pull up to a batch out of the channel and FREE ROOM, at which point
	// the frame is accepted and nothing drops. That is the flake — 1 run in 4-6,
	// always green in isolation. So the precondition is not "the queue is full",
	// it is "the worker is already blocked INSIDE flush and can free nothing",
	// and the test now establishes that before it fills.
	idlePersistQueue(t)
	block := make(chan struct{})
	entered := make(chan struct{})
	var once sync.Once
	SetBarPersister(func(historical bool, symbol, tf string, bars []Bar) {
		once.Do(func() { close(entered) })
		<-block
	})
	t.Cleanup(func() { close(block); idlePersistQueue(t) })

	// 1. Get the worker INTO flush and wedged there.
	fanOutBarPersist(nil, false, "MNQ", "1m", []Bar{{T: time.Now().UnixMilli()}})
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the worker never entered the persister — cannot establish the stall")
	}

	// 2. Only now fill to capacity. The worker is wedged, so nothing drains.
	fillPersistQueueUntilFull(t)

	// 3. Counters zeroed after the precondition holds, never before.
	persistDropped.Store(0)
	persistDroppedCloses.Store(0)

	// 4. THE ACTUAL ROOT CAUSE of this test's flakiness, and it was never load.
	//    The drop path increments both counters and then calls
	//    barPersistSummary(), which is rate-limited to once per 60 WALL-CLOCK
	//    seconds and, when it fires, Swap(0)s both counters — erasing the
	//    increment one line before the test reads it. At ~6.3s per iteration
	//    roughly every tenth run crossed a 60s boundary and read 0. Measured:
	//    failures returned in 6.01s with closes_dropped=0 AND queue_drops=0 —
	//    neither branch's counter survived — while passes took 6.30s with both
	//    at 1. Holding the rate limiter open for this window makes the
	//    destructive reset impossible during the assertion. Production is
	//    unchanged; the summary still fires on its own schedule in the bot.
	persistLastSum.Store(time.Now().Unix())

	// 5. One close-carrying frame: queue full, worker wedged → 2s×3 then the
	//    loud last resort, counted exactly once.
	closed := []Bar{{T: time.Now().Add(-2 * time.Minute).UnixMilli(), O: 1, H: 1, L: 1, C: 1}}
	done := make(chan struct{})
	go func() { fanOutBarPersist(nil, false, "MNQ", "1m", closed); close(done) }()
	select {
	case <-done:
		if got := persistDroppedCloses.Load(); got != 1 {
			t.Fatalf("closes_dropped = %d, want exactly 1 after the last resort", got)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("close-carrying frame did not resolve within the retry window + margin")
	}
}
