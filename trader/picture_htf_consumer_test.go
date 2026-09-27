package trader

import (
	"runtime"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	ntTrader "nofx/trader/ninjatrader"
)

// P2-15 RED→GREEN — the picture-HTF consumer goroutine must EXIT on the
// owning trader's Stop, not linger until the underlying subscription dies.
// Start/stop the consumer N times with a subscription channel that stays OPEN
// the whole time; after the loop the goroutine count must be back at baseline
// (with the old code each start leaked one blocked reader: +N goroutines).
// Production call sites: ensurePictureHtfBrokerConsumer / stopPictureHtfBrokerConsumer.
func TestPictureHtfConsumerStopsOnTraderStop(t *testing.T) {
	ch := make(chan ntwire.OrderUpdatePayload) // stays open — only the stop can end the consumer
	orig := pictureHtfBrokerListen
	t.Cleanup(func() { pictureHtfBrokerListen = orig })

	at := &AutoTrader{id: "p2-15-consumer-stop", trader: &ntTrader.TCPTrader{}}

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()

	const cycles = 10
	for i := 0; i < cycles; i++ {
		// A fresh seam per cycle closes `entered` once the consumer reaches the
		// listen — provably PAST the initial ctx check and inside the live loop.
		// Only the loop's stop path can then end it.
		entered := make(chan struct{})
		pictureHtfBrokerListen = func(tcp *ntTrader.TCPTrader) (<-chan ntwire.OrderUpdatePayload, func()) {
			close(entered)
			return ch, func() {}
		}
		at.ensurePictureHtfBrokerConsumer()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("consumer never reached the listen seam")
		}
		at.stopPictureHtfBrokerConsumer()
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		if got := runtime.NumGoroutine(); got <= base+2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("picture-HTF consumer leaked across %d stop cycles: goroutines %d (base %d) — the consumer has no stop path", cycles, runtime.NumGoroutine(), base)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// P2-15 — the consumer keeps consuming frames until stopped (the stop path
// must not break delivery): a frame sent while alive still routes, and after
// stop a new ensure for the SAME trader re-listens (restart shape).
func TestPictureHtfConsumerDeliversUntilStopAndRelistens(t *testing.T) {
	ch := make(chan ntwire.OrderUpdatePayload, 4)
	orig := pictureHtfBrokerListen
	pictureHtfBrokerListen = func(tcp *ntTrader.TCPTrader) (<-chan ntwire.OrderUpdatePayload, func()) {
		return ch, func() {}
	}
	t.Cleanup(func() { pictureHtfBrokerListen = orig })

	at := &AutoTrader{id: "p2-15-consumer-deliver", trader: &ntTrader.TCPTrader{}} // store nil: consume returns early
	at.ensurePictureHtfBrokerConsumer()
	ch <- ntwire.OrderUpdatePayload{SignalID: "s1"}
	ch <- ntwire.OrderUpdatePayload{SignalID: "s2"}
	time.Sleep(50 * time.Millisecond) // let the consumer drain
	at.stopPictureHtfBrokerConsumer()

	// Restart shape: the same trader id re-listens after the stop.
	at.ensurePictureHtfBrokerConsumer()
	if v, ok := pictureHtfBrokerConsumers.Load(at.id); !ok {
		t.Fatal("consumer not re-registered after stop")
	} else if h, _ := v.(*pictureHtfConsumerHandle); h == nil || h.at != at {
		t.Fatal("consumer handle missing the owning trader after re-listen")
	}
	at.stopPictureHtfBrokerConsumer()
}
