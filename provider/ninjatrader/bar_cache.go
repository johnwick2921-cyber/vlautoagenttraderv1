// Package ninjatrader — Plan 4.4 Stage 2 bar cache.
//
// Holds the latest N OHLCV bars per (symbol, timeframe) received over the
// TCP wire from the C# AddOn (frames bars_historical + bar_update). Thread-
// safe and bounded; read by the kernel (Stage 3 decision feed) and the
// chart relay (Stage 4) via Get().
//
// Backpressure invariant: writes from the socket read loop go through a
// bounded channel into a dedicated drain goroutine, which then calls into
// this cache. The cache itself uses a brief Lock() per write and an RLock
// + copy-on-read for snapshots, so reads never block writes for long.
package ninjatrader

import (
	"strings"
	"sync"
)

// DefaultBarCacheMaxBars is the per-(symbol, timeframe) ring-buffer
// capacity. 2500 (was 1024) retains the deeper re-seed window
// (defaultAutoBarsBack=2000) so a backfilled feed-down gap stays visible
// in the chart instead of being trimmed off; still covers EMA200 + ATR14
// with wide headroom.
const DefaultBarCacheMaxBars = 2500

// BarCache stores the most recent bars per (symbol, timeframe). Writes
// from bars_historical SEED the slice; writes from bar_update UPSERT,
// replacing the bar at the same t (in-progress bar) or appending new bars
// (capped at MaxBars — oldest bars are dropped FIFO once the ring fills).
type BarCache struct {
	mu      sync.RWMutex
	bars    map[string][]Bar // key: "SYMBOL|TIMEFRAME"
	maxBars int
	// dropped counts NT8 empty-minute placeholder bars refused at ingest.
	dropped int64
}

// NewBarCache constructs an empty cache. maxBars <= 0 uses
// DefaultBarCacheMaxBars.
func NewBarCache(maxBars int) *BarCache {
	if maxBars <= 0 {
		maxBars = DefaultBarCacheMaxBars
	}
	return &BarCache{
		bars:    make(map[string][]Bar),
		maxBars: maxBars,
	}
}

// NO SYNTHETIC BARS, EVER (P0 2026-08-17).
//
// NinjaTrader's own minute store keeps EMPTY-MINUTE PLACEHOLDER records, and its
// bar builder materialises each one as a bar with open==high==low==close (the
// .ncd file's base price) and volume 0. Whenever a declared-open session has no
// real ticks, NT8 therefore hands us a flat line rather than a gap.
//
// That is exactly what the owner saw. After the AddOn watchdog livelock
// (7aa521a1) forced NT8 to re-fetch Friday 2026-08-14 into an all-placeholder
// file, /api/klines returned 959 of 1500 one-minute bars with O=H=L=C=30147.50
// and volume 0, contiguous from 00:02 to 16:00 CT — a 16-hour horizontal line
// across the chart where TradingView (and this chart, before) would simply skip
// to the next real candle.
//
// Every hop we own is a verbatim pass-through, so nothing in our code invents
// these bars — but nothing rejected them either, and they reach the chart AND
// the kernel's detectors. A bar with no volume and no range carries no
// information by construction; the only thing it can do is lie. Drop it at
// ingest, which is the one place that protects both consumers at once.
//
// The test is deliberately narrow: BOTH zero volume AND zero range. A real but
// illiquid minute that still printed a range is kept, and so is a zero-volume
// bar that somehow carries one.
func isPlaceholderBar(b Bar) bool {
	return b.V == 0 && b.H == b.L && b.O == b.C && b.O == b.H
}

// dropPlaceholderBars returns bars with NT8's empty-minute placeholders removed,
// plus how many were dropped. It allocates only when something is actually
// dropped, so the healthy path stays free.
func dropPlaceholderBars(bars []Bar) ([]Bar, int) {
	bad := 0
	for i := range bars {
		if isPlaceholderBar(bars[i]) {
			bad++
		}
	}
	if bad == 0 {
		return bars, 0
	}
	out := make([]Bar, 0, len(bars)-bad)
	for i := range bars {
		if !isPlaceholderBar(bars[i]) {
			out = append(out, bars[i])
		}
	}
	return out, bad
}

// DroppedPlaceholders reports how many NT8 empty-minute placeholder bars this
// cache has refused, so the condition is observable instead of silent.
func (c *BarCache) DroppedPlaceholders() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dropped
}

// ── CANONICAL TIME CONTRACT (2026-08-19, chart-timestamp dispatch) ──────────
//
//	A Bar's T in THIS CACHE is the bar's OPEN time, epoch ms UTC.
//
// NT8 stamps bars at their period END (NinjaScript Bars.GetTime(i) is the
// close; proven live: a forming 5m bar covering 01:30–01:35 CT arrives stamped
// 01:35 while the clock reads 01:31). The C# AddOn forwards that close stamp
// verbatim, and this cache used to store it unchanged while every reader —
// barsToKlines, /api/klines, the SSE relay, the kernel detectors, the charts —
// treated T as the OPEN. Net effect: every bar was labelled one full period
// late everywhere downstream.
//
// The conversion happens HERE, once, at ingest, because every consumer reads
// this cache (the SSE relay polls it; REST serves it; the kernel bridges it).
// Converting in any single reader would leave the twins wrong (the multi-
// instance defect class). The C# side and its LastEmittedTimeUtcMs dedup
// cursor stay in close-stamp domain untouched — no wire change.
//
// Side effect worth naming: C2's clockDriftMs and the clock-health line both
// compute "feed now" as newestT + interval. Under close stamps that OVERSHOT
// the true close by one interval; with open stamps it lands exactly on NT8's
// own stamp again — a strict accuracy improvement, thresholds untouched.
func openStampBars(bars []Bar, timeframe string) []Bar {
	dur := timeframeMs(timeframe)
	if dur <= 0 || len(bars) == 0 {
		return bars
	}
	out := make([]Bar, len(bars))
	for i, b := range bars {
		b.T -= dur
		out[i] = b
	}
	return out
}

// timeframeMs mirrors the coded TF vocabulary (bars_market_bridge.go keeps the
// kernel-side twin; both fall back to 1m).
func timeframeMs(timeframe string) int64 {
	switch timeframe {
	case "1m":
		return 60_000
	case "2m": // parity with kernel.TFDurationMs
		return 120_000
	case "3m":
		return 180_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "30m":
		return 1_800_000
	case "1h":
		return 3_600_000
	case "2h":
		return 7_200_000
	case "4h":
		return 14_400_000
	case "6h": // C11 (2026-08-25) — previously fell through to 1m
		return 21_600_000
	case "8h":
		return 28_800_000
	case "12h":
		return 43_200_000
	case "1d", "1D":
		return 86_400_000
	case "3d": // C11 — previously fell through to 1m
		return 259_200_000
	case "1w", "1W":
		return 604_800_000
	default:
		return 60_000
	}
}

// SeedHistorical MERGES the provided bars into the cache for (symbol,
// timeframe). Called when the C# AddOn emits a bars_historical frame: the
// initial seed after a fresh BarsRequest, a Go-reconnect re-seed (N3), OR a
// data-feed reconnect recreate.
//
// MERGE, not replace: a re-seed can legitimately carry FEWER bars than the
// cache already holds — e.g. a reconnect recreate whose BarsRequest only loaded
// the current session, or an empty cursor-deduped frame. Replacing on such a
// frame WIPED the deeper history we'd accumulated (the Phase-0 reconnect
// regression: the chart collapsed to "today only"). Merging makes the persistent
// ring durable — it never shrinks on a thinner re-seed, accumulates real depth
// across sessions, survives reconnects, and lets a future deeper load fill in
// older bars in front. Union is by bar time T; incoming wins on overlap
// (freshest OHLCV for that bar); ascending order preserved; tail-capped at
// maxBars. Both inputs are assumed ascending by T (protocol contract
// vltrader_tcp_PROTOCOL.md §6).
func (c *BarCache) SeedHistorical(symbol, timeframe string, bars []Bar) {
	if symbol == "" || timeframe == "" {
		return
	}
	bars, bad := dropPlaceholderBars(bars)
	bars = openStampBars(bars, timeframe) // close-stamp → OPEN-stamp, once, for every reader
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropped += int64(bad)
	key := barKey(symbol, timeframe)
	existing := c.bars[key]
	if len(existing) == 0 {
		// First seed for this key — copy the tail (detaches from caller's array).
		if len(bars) > c.maxBars {
			bars = bars[len(bars)-c.maxBars:]
		}
		stored := make([]Bar, len(bars))
		copy(stored, bars)
		c.bars[key] = stored
		return
	}
	if len(bars) == 0 {
		// Empty re-seed (cursor-deduped reconnect frame) — keep what we have.
		return
	}
	merged := mergeBarsByTime(existing, bars)
	if len(merged) > c.maxBars {
		merged = merged[len(merged)-c.maxBars:]
	}
	c.bars[key] = merged
}

// mergeBarsByTime merges two ascending-by-T bar slices into one ascending,
// time-deduped slice. On equal T, the bar from `incoming` wins (it is the
// freshest OHLCV for that bar boundary); bars present only in `existing`
// (older history a thinner re-seed didn't carry) are preserved. Linear
// two-pointer merge — both inputs must be ascending by T.
func mergeBarsByTime(existing, incoming []Bar) []Bar {
	out := make([]Bar, 0, len(existing)+len(incoming))
	i, j := 0, 0
	for i < len(existing) && j < len(incoming) {
		switch {
		case existing[i].T < incoming[j].T:
			out = append(out, existing[i])
			i++
		case existing[i].T > incoming[j].T:
			out = append(out, incoming[j])
			j++
		default: // same bar time — incoming is freshest
			out = append(out, incoming[j])
			i++
			j++
		}
	}
	for ; i < len(existing); i++ {
		out = append(out, existing[i])
	}
	for ; j < len(incoming); j++ {
		out = append(out, incoming[j])
	}
	return out
}

// Upsert merges streaming bar_update bars into the cache. For each input
// bar:
//   - If the t matches the last cached bar's t: REPLACE (in-progress bar
//     update — the current minute's bar is still forming).
//   - If the t is strictly greater than the last cached bar's t: APPEND
//     (a new bar boundary was crossed).
//   - If the t is strictly less than the last cached bar's t: IGNORE
//     (out-of-order — the protocol contract is ascending, but defend
//     against partial misalignment after reconnect).
//
// The ring is bounded: when the slice exceeds maxBars, the oldest bar is
// dropped (FIFO).
//
// The input slice may contain MULTIPLE bars (the NT8 multi-bar gotcha:
// a single tick can update MinIndex..MaxIndex). Each is applied in input
// order via the rules above.
func (c *BarCache) Upsert(symbol, timeframe string, bars []Bar) {
	if symbol == "" || timeframe == "" || len(bars) == 0 {
		return
	}
	bars, bad := dropPlaceholderBars(bars)
	bars = openStampBars(bars, timeframe) // close-stamp → OPEN-stamp, once, for every reader
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropped += int64(bad)
	if len(bars) == 0 {
		return // the whole update was placeholders
	}
	key := barKey(symbol, timeframe)
	existing := c.bars[key]
	for _, b := range bars {
		if len(existing) == 0 {
			existing = append(existing, b)
			continue
		}
		last := existing[len(existing)-1]
		switch {
		case b.T == last.T:
			existing[len(existing)-1] = b
		case b.T > last.T:
			existing = append(existing, b)
			if len(existing) > c.maxBars {
				// Drop the oldest bar; copy the tail forward in place
				// to keep the slice's backing array bounded.
				copy(existing, existing[1:])
				existing = existing[:c.maxBars]
			}
		default:
			// Out-of-order bar — defensive: ignore.
		}
	}
	c.bars[key] = existing
}

// Get returns a SNAPSHOT (defensive copy) of the cached bars for
// (symbol, timeframe). The returned slice is safe for the caller to hold
// or mutate without affecting the cache. Returns nil if no bars are
// cached yet.
//
// The RLock is held only for the duration of the slice copy, which is
// microseconds. Readers never block writers for long.
func (c *BarCache) Get(symbol, timeframe string) []Bar {
	c.mu.RLock()
	defer c.mu.RUnlock()
	src, ok := c.bars[barKey(symbol, timeframe)]
	if !ok || len(src) == 0 {
		return nil
	}
	out := make([]Bar, len(src))
	copy(out, src)
	return out
}

// Count returns the number of cached bars for (symbol, timeframe). Useful
// for tests and metrics.
func (c *BarCache) Count(symbol, timeframe string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.bars[barKey(symbol, timeframe)])
}

// Keys returns the list of currently-populated (symbol, timeframe) pairs.
// Returned in unspecified order. Useful for diagnostics + the Stage 4
// chart relay enumerating its outbound subscriptions.
func (c *BarCache) Keys() [][2]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([][2]string, 0, len(c.bars))
	for k := range c.bars {
		s, t, ok := splitBarKey(k)
		if !ok {
			continue
		}
		out = append(out, [2]string{s, t})
	}
	return out
}

// PurgeSymbol removes EVERY timeframe's cached bars for a symbol (P5.3 clean
// teardown after bars_unsubscribe — a removed symbol must not leave stale data
// for the chart/API to read). Case-insensitive on the symbol half of the key.
// Returns the number of (symbol|timeframe) entries removed.
func (c *BarCache) PurgeSymbol(symbol string) int {
	if symbol == "" {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for k := range c.bars {
		s, _, ok := splitBarKey(k)
		if ok && strings.EqualFold(s, symbol) {
			delete(c.bars, k)
			removed++
		}
	}
	return removed
}

// barKey encodes (symbol, timeframe) as a single map key. We deliberately
// keep the encoding simple (single delimiter) and document it so the
// splitBarKey reverse is unambiguous. Symbols + timeframes are bounded
// strings without "|" so the delimiter is safe.
func barKey(symbol, timeframe string) string {
	return symbol + "|" + timeframe
}

// splitBarKey reverses barKey. Returns ok=false if the format is
// unexpected (defensive — should only be reachable if external code
// mutated the cache directly).
func splitBarKey(k string) (symbol, timeframe string, ok bool) {
	for i := 0; i < len(k); i++ {
		if k[i] == '|' {
			return k[:i], k[i+1:], true
		}
	}
	return "", "", false
}
