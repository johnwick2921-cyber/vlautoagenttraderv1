package telemetry

import "sync/atomic"

// ── 101 D1'(3), narrow form — a confirmed scale break is COUNTED ─────────────
//
// After a true break the ring drops its historical seed; for every non-1m
// timeframe that leaves the ring live-only until NT8's next full replay,
// because the store rehydrate is 1m-only by the owner's 2026-09-09 condition.
// The WARN says so once per event and this counter records how often it has
// happened since boot, so a thin ring after a reconnect is a number on a line
// rather than a surprise on a chart.
var (
	scaleBreakDrops       atomic.Int64 // events
	scaleBreakBarsDropped atomic.Int64 // historical bars removed, summed
)

// IncScaleBreakDrop records one confirmed break and the bars it cost.
func IncScaleBreakDrop(barsDropped int) {
	scaleBreakDrops.Add(1)
	scaleBreakBarsDropped.Add(int64(barsDropped))
}

// ScaleBreakCounts is READ by the boot/summary line (A11).
func ScaleBreakCounts() (events, barsDropped int64) {
	return scaleBreakDrops.Load(), scaleBreakBarsDropped.Load()
}
