package ninjatrader

import (
	"sync"
	"sync/atomic"
	"time"

	"nofx/logger"
)

// LIVE BAR SINK (W-PICTURE-HTF, 2026-09-20) — event fan-out for deterministic
// evaluators (the two-picture mode). The picture evaluator runs from NATIVE
// LIVE bar events, never historical replays: a Sept-17 replay must not mint a
// real opportunity, so drainBarIngest fans out ONLY non-historical frames.
//
// One process-wide sink (one TCP server). Registration is idempotent (last
// writer wins — exactly one evaluator dispatcher installs it). The sink call
// is dispatched through a small worker channel so the cache drain and the
// socket read loop are never stalled by evaluator work (DB writes only on
// qualified setups, but the drain-loop non-blocking invariant is absolute).

var (
	liveBarSink   atomic.Value // func(symbol, tf, contract string, bars []Bar, receivedAt time.Time)
	liveSinkCh    = make(chan liveSinkMsg, 256)
	liveSinkOnce  sync.Once
	liveSinkDrops atomic.Int64
)

type liveSinkMsg struct {
	symbol string
	tf     string
	// contract is the front month the AddOn named on this frame; "" is
	// UNKNOWN. It travels WITH the frame because identity is a property of
	// the frame, not of the cache key it lands in.
	contract   string
	bars       []Bar
	receivedAt time.Time
}

// SetLiveBarSink installs the process-wide live-bar consumer. nil uninstalls.
func SetLiveBarSink(fn func(symbol, tf, contract string, bars []Bar, receivedAt time.Time)) {
	liveBarSink.Store(fn)
	startLiveSinkWorker()
}

func startLiveSinkWorker() {
	liveSinkOnce.Do(func() {
		go liveSinkLoop()
	})
}

func liveSinkLoop() {
	for msg := range liveSinkCh {
		fn, _ := liveBarSink.Load().(func(string, string, string, []Bar, time.Time))
		if fn == nil {
			continue
		}
		// The evaluator dispatcher never panics on purpose, but a panic in a
		// sink must never take the worker (and the frames behind it) down.
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("live bar sink panic recovered: %v", r)
				}
			}()
			fn(msg.symbol, msg.tf, msg.contract, msg.bars, msg.receivedAt)
		}()
	}
}

// fanOutLiveBars enqueues LIVE frames for the sink. Non-blocking; on a full
// queue the pending frame is DROPPED and counted loudly — evaluators rebuild
// their state from the cache on every event, so a dropped frame only delays
// an evaluation to the next one (the wall-clock fallback covers the rest).
func fanOutLiveBars(symbol, tf, contract string, bars []Bar) {
	if len(bars) == 0 {
		return
	}
	if _, ok := liveBarSink.Load().(func(string, string, string, []Bar, time.Time)); !ok {
		return
	}
	select {
	case liveSinkCh <- liveSinkMsg{symbol: symbol, tf: tf, contract: contract, bars: bars, receivedAt: time.Now()}:
	default:
		liveSinkDrops.Add(1)
		if liveSinkDrops.Load()%100 == 1 {
			logger.Warnf("picture-htf: live sink queue full — %d frame(s) dropped (self-heals: evaluators rebuild from cache)", liveSinkDrops.Load())
		}
	}
}

// LiveSinkDrops exposes the dropped-frame counter (boot lines / dashboards).
func LiveSinkDrops() int64 { return liveSinkDrops.Load() }
