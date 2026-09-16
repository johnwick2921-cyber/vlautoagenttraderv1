package ninjatrader

import (
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Bar sources, mirrored from store so the ring can stamp without importing it.
const (
	BarSourceLive       = "live"
	BarSourceHistorical = "historical"
	BarSourceMixed      = "mixed"
)

// ScaleMismatchPct and ScaleMismatchRangeMult are the detection thresholds for
// "the replay and the live feed are on different price scales": the first live
// close after a historical seed differing from the last historical close by
// more than BOTH
//   - this fraction of price (a one-minute bar does not move half a percent of
//     an index; a back-adjusted replay against an unadjusted live feed did —
//     MNQ 1.0%, ES 0.86% on 2026-09-10), AND
//   - this multiple of the seed's own median bar body (the tape's own scale, so
//     the rule needs no assumption about what the instrument's price IS —
//     the percent alone fired on a one-point move in a fixture priced at 100).
//
// Both are [I], stated on the boot line. NOFX_BAR_SCALE_MISMATCH_PCT and
// NOFX_BAR_SCALE_MISMATCH_MULT override.
var (
	ScaleMismatchPct       = 0.005
	ScaleMismatchRangeMult = 20.0
)

func init() {
	// A knob that is documented is a knob that exists (class 19). Positive
	// finite values only; anything else keeps the default and says nothing —
	// the boot line prints the value in force either way.
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv("NOFX_BAR_SCALE_MISMATCH_PCT")), 64); err == nil && v > 0 && !math.IsInf(v, 0) {
		ScaleMismatchPct = v
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv("NOFX_BAR_SCALE_MISMATCH_MULT")), 64); err == nil && v > 0 && !math.IsInf(v, 0) {
		ScaleMismatchRangeMult = v
	}
}

// medianBody is the median |close-open| of the bars given — the tape's own
// scale. Zero when there are no bars or every body is zero.
func medianBody(bars []Bar) float64 {
	bodies := make([]float64, 0, len(bars))
	for _, b := range bars {
		if d := math.Abs(b.C - b.O); d > 0 {
			bodies = append(bodies, d)
		}
	}
	if len(bodies) == 0 {
		return 0
	}
	// insertion sort — n is small (a seed, not a history)
	for i := 1; i < len(bodies); i++ {
		for j := i; j > 0 && bodies[j] < bodies[j-1]; j-- {
			bodies[j], bodies[j-1] = bodies[j-1], bodies[j]
		}
	}
	return bodies[len(bodies)/2]
}

// ScaleMismatch records one detection for the boot line and the P0.
type ScaleMismatch struct {
	Symbol, Timeframe string
	At                time.Time
	LastHistoricalC   float64
	FirstLiveC        float64
	DeltaPts          float64
	HistoricalDropped int
}

// ScaleMismatchListener is called ONCE per (symbol, timeframe) per process.
type ScaleMismatchListener func(m ScaleMismatch)

var (
	scaleListenersMu sync.RWMutex
	scaleListeners   []ScaleMismatchListener
)

// OnScaleMismatch registers a listener (the persist wire registers one to
// re-rehydrate from the store's live rows and raise the P0).
func OnScaleMismatch(fn ScaleMismatchListener) {
	if fn == nil {
		return
	}
	scaleListenersMu.Lock()
	scaleListeners = append(scaleListeners, fn)
	scaleListenersMu.Unlock()
}

func stampSource(bars []Bar, src string) []Bar {
	for i := range bars {
		bars[i].Source = src
	}
	return bars
}

// mergeSeedKeepingLive is mergeBarsByTime with the one rule this wave exists
// for: on a same-time collision an EXISTING live (or mixed) bar is kept over an
// INCOMING historical one. The old merge said "incoming is freshest" — true for
// a live update over a stale seed, false for a replay over a bar that traded.
func mergeSeedKeepingLive(existing, incoming []Bar) []Bar {
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
		default:
			if existing[i].Source == BarSourceLive || existing[i].Source == BarSourceMixed {
				out = append(out, existing[i]) // the minute as it traded stays
			} else {
				out = append(out, incoming[j])
			}
			i++
			j++
		}
	}
	out = append(out, existing[i:]...)
	out = append(out, incoming[j:]...)
	return out
}

// detectScaleMismatch runs on a LIVE upsert. If the bar it is about to write
// is the first live bar for this key, the ring holds a historical seed, and the
// live close differs from the last historical close by more than
// ScaleMismatchPct of price: the replay is on a different scale.
//
// Returns the (possibly re-stamped) bar and whether a mismatch was found. On a
// mismatch the incoming bar — whose open the AddOn copied from its own
// historical series and whose close is live — is stamped MIXED, and every
// historical bar for the key is DROPPED from the ring: the store's live rows,
// which the upsert rule now protects, are the right thing to refill it from.
//
// Not a value edit (A24): the bar keeps the values the wire delivered; it is
// labelled so no reader takes it, and the label is the finding.
func (c *BarCache) detectScaleMismatch(key string, b Bar, now time.Time) (Bar, bool) {
	existing := c.bars[key]
	if len(existing) == 0 {
		return b, false
	}
	if c.liveSeen == nil {
		c.liveSeen = make(map[string]bool)
	}
	if c.liveSeen[key] {
		return b, false
	}
	c.liveSeen[key] = true
	// last historical bar strictly before this minute
	var last *Bar
	for i := len(existing) - 1; i >= 0; i-- {
		if existing[i].T < b.T && existing[i].Source == BarSourceHistorical {
			last = &existing[i]
			break
		}
	}
	if last == nil || last.C == 0 {
		return b, false
	}
	delta := math.Abs(b.C - last.C)
	if delta <= ScaleMismatchPct*math.Abs(last.C) {
		return b, false
	}
	// The tape's own scale: a move that is large in percent terms but ordinary
	// against this seed's bar bodies is a move, not a scale shift. Requires a
	// seed with SOME body to measure; a flat seed cannot vouch either way and
	// the percent rule stands alone.
	if mb := medianBody(existing); mb > 0 && delta <= ScaleMismatchRangeMult*mb {
		return b, false
	}
	// Different scales. Drop the seed; label the straddling bar.
	kept := existing[:0]
	dropped := 0
	for _, e := range existing {
		if e.Source == BarSourceHistorical {
			dropped++
			continue
		}
		kept = append(kept, e)
	}
	c.bars[key] = kept
	b.Source = BarSourceMixed
	if c.seedOffScale == nil {
		c.seedOffScale = make(map[string]bool)
	}
	c.seedOffScale[key] = true
	sym, tf, _ := splitBarKey(key)
	m := ScaleMismatch{Symbol: sym, Timeframe: tf, At: now, LastHistoricalC: last.C, FirstLiveC: b.C, DeltaPts: delta, HistoricalDropped: dropped}
	if c.mismatches == nil {
		c.mismatches = make(map[string]ScaleMismatch)
	}
	c.mismatches[key] = m
	scaleListenersMu.RLock()
	ls := append([]ScaleMismatchListener(nil), scaleListeners...)
	scaleListenersMu.RUnlock()
	for _, fn := range ls {
		go fn(m)
	}
	return b, true
}

// SeedVerdict reports, for one symbol×timeframe, whether the ring has compared
// a live bar against the seed it currently holds (checked) and whether that
// seed was found on another scale (offScale). The persist wire holds a
// replay's rows OUT of the store until checked is true, and discards them when
// offScale is: an unverified replay is not the record, and a replay on the
// wrong scale must never become one (the 22:39 boot wrote 186 such rows over
// live ones through the old unconditional upsert; this is the second door).
func (c *BarCache) SeedVerdict(symbol, timeframe string) (checked, offScale bool) {
	if c == nil {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := barKey(symbol, timeframe)
	return c.liveSeen[key], c.seedOffScale[key]
}

// ScaleMismatches returns every detection this process, for the boot line.
func (c *BarCache) ScaleMismatches() []ScaleMismatch {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ScaleMismatch, 0, len(c.mismatches))
	for _, m := range c.mismatches {
		out = append(out, m)
	}
	return out
}
