package ninjatrader

import (
	"sync"
	"testing"
	"time"
)

// scaleCapture collects listener events under a lock — the ring fires listeners
// on their own goroutine, so an unguarded slice/int is a data race under -race
// (the CI coverage job runs -race; cleanup batch 2, B8).
type scaleCapture struct {
	mu  sync.Mutex
	got []ScaleMismatch
}

func (c *scaleCapture) listen() {
	OnScaleMismatch(func(m ScaleMismatch) { c.mu.Lock(); c.got = append(c.got, m); c.mu.Unlock() })
}
func (c *scaleCapture) events() []ScaleMismatch {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ScaleMismatch(nil), c.got...)
}
func (c *scaleCapture) fired() int { return len(c.events()) }

// ── BAR-SOURCE WAVE PINS (ring half) ─────────────────────────────────────────
//
// The fixture reproduces 2026-09-10 22:39 CT: a cold ring seeded by a replay
// on the LOW scale (~29068), then live updates on the HIGH scale (~29358).
// Boundary and scales are FIXTURE CONSTANTS (A28).
const (
	bsT0    int64   = 1789097820000 // 22:37 CT
	bsLow   float64 = 29068.25
	bsHigh  float64 = 29358.25
	bsDelta         = bsHigh - bsLow // 290.00
)

func seededRing(t *testing.T) *BarCache {
	t.Helper()
	c := NewBarCache(2500)
	var seed []Bar
	for i := 0; i < 30; i++ { // 22:07 .. 22:36, replay, low scale
		ms := bsT0 - int64(30-i)*60_000
		seed = append(seed, Bar{T: ms, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow, V: 1})
	}
	c.SeedHistorical("MNQ", "1m", seed)
	return c
}

// A REPLAY NEVER OVERWRITES A LIVE BAR — ring half. RED on the pre-wave cache:
// mergeBarsByTime took "incoming is freshest" and the live bar was replaced.
func TestReplayNeverOverwritesALiveBarInTheRing(t *testing.T) {
	c := NewBarCache(2500)
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsHigh, H: bsHigh + 1, L: bsHigh - 1, C: bsHigh, V: 1}})
	c.SeedHistorical("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow, V: 1}})
	got := c.Get("MNQ", "1m")
	if len(got) != 1 {
		t.Fatalf("want 1 bar, got %d", len(got))
	}
	if got[0].C != bsHigh || got[0].Source != BarSourceLive {
		t.Fatalf("the replay overwrote a bar that traded: close=%.2f source=%q (want %.2f live)", got[0].C, got[0].Source, bsHigh)
	}
}

// Live DOES overwrite a historical seed — that direction is correct and must
// keep working (the seed is the stale one when they share a scale).
func TestLiveOverwritesAHistoricalSeedOnTheSameScale(t *testing.T) {
	c := NewBarCache(2500)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow, V: 1}})
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 2, L: bsLow - 1, C: bsLow + 1.5, V: 2}})
	got := c.Get("MNQ", "1m")
	if got[0].C != bsLow+1.5 || got[0].Source != BarSourceLive {
		t.Fatalf("a same-scale live update must replace the seed: close=%.2f source=%q", got[0].C, got[0].Source)
	}
}

// THE BOOT MINUTE IS NEVER A MIXED BAR. The first live bar after a replay seed
// closes 290 points from the last replay close: the seed is on another scale.
// The seed is DROPPED, the straddling bar is labelled MIXED (values kept), and
// the listener fires ONCE with the numbers.
func TestScaleMismatchDropsTheSeedAndLabelsTheBootMinute(t *testing.T) {
	c := seededRing(t)
	if n := c.Count("MNQ", "1m"); n != 30 {
		t.Fatalf("seed: want 30, got %d", n)
	}
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

	// 22:37 arrives live: the AddOn copied its open from its own historical
	// series (low) and its close is live (high) — the mixed shape.
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsHigh + 1, L: bsLow - 1, C: bsHigh, V: 3}})
	time.Sleep(20 * time.Millisecond) // listener runs on its own goroutine

	bars := c.Get("MNQ", "1m")
	if len(bars) != 1 {
		t.Fatalf("the historical seed must be DROPPED on a scale mismatch; ring holds %d", len(bars))
	}
	if bars[0].Source != BarSourceMixed {
		t.Fatalf("the straddling bar must be labelled mixed, got %q", bars[0].Source)
	}
	if bars[0].O != bsLow || bars[0].C != bsHigh {
		t.Fatalf("A24: the mixed bar's VALUES must be untouched; got o=%.2f c=%.2f", bars[0].O, bars[0].C)
	}
	if got := cap.events(); len(got) != 1 || got[0].DeltaPts != bsDelta || got[0].HistoricalDropped != 30 {
		t.Fatalf("listener: want one event Δ=%.2f dropped=30, got %+v", bsDelta, got)
	}
	// A second live bar is ordinary live — no second event, no second mixed.
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0 + 60_000, O: bsHigh, H: bsHigh + 1, L: bsHigh - 1, C: bsHigh + 0.5, V: 1}})
	time.Sleep(20 * time.Millisecond)
	bars = c.Get("MNQ", "1m")
	if cap.fired() != 1 || bars[len(bars)-1].Source != BarSourceLive {
		t.Fatalf("only the FIRST live bar is checked; events=%d last source=%q", cap.fired(), bars[len(bars)-1].Source)
	}
}

// Same scale → no mismatch, nothing dropped, nothing labelled.
func TestNoMismatchOnTheSameScale(t *testing.T) {
	c := seededRing(t)
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 2, L: bsLow - 1, C: bsLow + 1, V: 1}})
	time.Sleep(20 * time.Millisecond)
	if cap.fired() != 0 || c.Count("MNQ", "1m") != 31 {
		t.Fatalf("same scale: fired=%d ring=%d (want 0, 31)", cap.fired(), c.Count("MNQ", "1m"))
	}
	for _, b := range c.Get("MNQ", "1m") {
		if b.Source == BarSourceMixed {
			t.Fatal("nothing may be labelled mixed on a same-scale boot")
		}
	}
}

// Every bar the ring holds names its source — a replay is historical, an
// update is live, and nothing is unlabelled.
func TestEveryRingBarNamesItsSource(t *testing.T) {
	c := seededRing(t)
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow + 0.25, V: 1}})
	for _, b := range c.Get("MNQ", "1m") {
		if b.Source != BarSourceLive && b.Source != BarSourceHistorical {
			t.Fatalf("unlabelled bar at %d: %q", b.T, b.Source)
		}
	}
}

// A MID-SESSION RECONNECT RE-ARMS THE CHECK. Boot on one scale (no mismatch),
// live for a while, then a reconnect replays on a DIFFERENT scale: the first
// live bar after THAT seed must catch it. RED with a per-process guard.
func TestReconnectReseedIsCheckedAgain(t *testing.T) {
	c := seededRing(t) // low-scale seed
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

	// live on the SAME (low) scale — a clean boot, nothing fires
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow + 0.5, V: 1}})
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0 + 60_000, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow + 0.25, V: 1}})
	time.Sleep(20 * time.Millisecond)
	if cap.fired() != 0 {
		t.Fatalf("same-scale boot must not fire; fired=%d", cap.fired())
	}
	// reconnect: the replay comes back on the HIGH scale for later minutes
	var re []Bar
	for i := 2; i < 6; i++ {
		re = append(re, Bar{T: bsT0 + int64(i)*60_000, O: bsHigh, H: bsHigh + 1, L: bsHigh - 1, C: bsHigh, V: 1})
	}
	c.SeedHistorical("MNQ", "1m", re)
	// first live bar after the re-seed, on the LOW scale (live never moved)
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0 + 6*60_000, O: bsHigh, H: bsHigh + 1, L: bsLow - 1, C: bsLow, V: 1}})
	time.Sleep(20 * time.Millisecond)
	if cap.fired() != 1 {
		t.Fatalf("a reconnect re-seed on another scale must be caught by the first live bar after it; fired=%d", cap.fired())
	}
}

// THE TAPE'S OWN SCALE. A move that clears the percent threshold but is
// ordinary against the seed's bar bodies is a MOVE, not a scale shift. At price
// 100 with one-point bodies, a five-point close-to-close is 5% of price — and
// five bodies. Not a mismatch. (The percent rule alone fired on the ring-bound
// test's one-point step.)
func TestOrdinaryMoveAtSmallPriceIsNotAMismatch(t *testing.T) {
	c := NewBarCache(2500)
	var seed []Bar
	for i := 0; i < 20; i++ {
		ms := bsT0 - int64(20-i)*60_000
		o := 100.0 + float64(i)
		seed = append(seed, Bar{T: ms, O: o, H: o + 1.5, L: o - 0.5, C: o + 1, V: 1}) // one-point bodies
	}
	c.SeedHistorical("MNQ", "1m", seed)
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: 120, H: 126, L: 119, C: 125, V: 1}}) // +5 from last close 120: 4.2% of price, 5 bodies
	time.Sleep(20 * time.Millisecond)
	if cap.fired() != 0 || c.Count("MNQ", "1m") != 21 {
		t.Fatalf("an ordinary move must not read as a scale shift: fired=%d ring=%d", cap.fired(), c.Count("MNQ", "1m"))
	}
	// And the real thing at the same price, for contrast: +50 is 50 bodies.
	c2 := NewBarCache(2500)
	c2.SeedHistorical("MNQ", "1m", seed)
	c2.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: 120, H: 171, L: 119, C: 170, V: 1}})
	time.Sleep(20 * time.Millisecond)
	if cap.fired() != 1 {
		t.Fatalf("a fifty-body jump at the same price IS a scale shift; fired=%d", cap.fired())
	}
}

// THE VERDICT FOLLOWS THE SEED. Unjudged until a live bar; off-scale sticks
// for that seed; a fresh replay clears both so it is judged on its own.
func TestSeedVerdictFollowsTheSeed(t *testing.T) {
	c := NewBarCache(100)
	seed := func(price float64) []Bar {
		out := make([]Bar, 0, 5)
		for i := int64(1); i <= 5; i++ {
			out = append(out, Bar{T: i * 60_000, O: price, H: price + 2, L: price - 2, C: price + 1, V: 1})
		}
		return out
	}
	if ch, off := c.SeedVerdict("MNQ", "1m"); ch || off {
		t.Fatal("a cold key has no verdict")
	}
	c.SeedHistorical("MNQ", "1m", seed(29060))
	if ch, off := c.SeedVerdict("MNQ", "1m"); ch || off {
		t.Fatal("a seed nobody has compared to a live bar has no verdict")
	}
	c.Upsert("MNQ", "1m", []Bar{{T: 6 * 60_000, O: 29350, H: 29352, L: 29348, C: 29351, V: 1}})
	if ch, off := c.SeedVerdict("MNQ", "1m"); !ch || !off {
		t.Fatalf("judged off-scale expected, got checked=%v offScale=%v", ch, off)
	}
	c.SeedHistorical("MNQ", "1m", seed(29350)) // reconnect: a replay on the live scale
	if ch, off := c.SeedVerdict("MNQ", "1m"); ch || off {
		t.Fatalf("a fresh seed must clear the verdict, got checked=%v offScale=%v", ch, off)
	}
	c.Upsert("MNQ", "1m", []Bar{{T: 7 * 60_000, O: 29350, H: 29352, L: 29348, C: 29351, V: 1}})
	if ch, off := c.SeedVerdict("MNQ", "1m"); !ch || off {
		t.Fatalf("judged on-scale expected, got checked=%v offScale=%v", ch, off)
	}
}
