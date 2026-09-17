package ninjatrader

import (
	"testing"
	"time"
)

// ── 101 D1' — THE 2026-09-16 09:22:11 EVENT AS THE FIXTURE ──────────────────
//
// The live tape: NT8 delivered 2,000 5m bars at 22:15 CT on 09-15 (last replay
// bar 22:10, close 29320.50). Live bars accrued through the night. On 09-16 the
// feed flapped five times; each reconnect's BarsRequest ran while the feed was
// down and the AddOn emitted `bars=0`. At 09:22:11 the zero-bar replay RE-ARMED
// the scale check; the next live bar (09:22, close 29467.25) was judged against
// the 22:10 bar from eleven hours earlier; 146.75 pts of overnight move read as
// a scale break; 1,999 historical bars were dropped. The 5m horizon went from
// served=2131 span=285h55m to served=134 span=11h5m.
//
// Three rules, each pinned here RED-first:
//   (1) a zero-bar replay must not re-arm the check;
//   (2) the check compares ADJACENT bars only — a reference older than one
//       interval means SKIP + WARN, never a drop;
//   (3) a REAL break — adjacent bars, >0.5% — must still drop.

const (
	// 09-15 22:10 CT — the boot replay's last 5m bar (store: c=29320.5 live)
	sep15_2210 int64 = 1789096200000
	sep15_2215 int64 = sep15_2210 + 5*60_000
	// 09-16 09:22 CT — the first live bar after the zero-bar reconnect replay
	sep16_0922  int64 = 1789136520000
	lastReplayC       = 29320.50
	firstLiveC        = 29467.25 // delta 146.75 = 0.50054% of 29320.50 — over the line by 0.15 pt
)

// sep16Ring builds the 5m ring as it stood at 09:22:10 on 09-16: a 2,000-bar
// historical seed ending 22:10 09-15, then live bars from 22:15 09-15 to 09:20
// 09-16 on a drifting scale. The first live bar at 22:15 already judged the
// seed (12 pts from it — same scale) and passed.
func sep16Ring(t *testing.T, tf string, intervalMs int64) *BarCache {
	t.Helper()
	c := NewBarCache(2500)
	var seed []Bar
	for i := 1999; i >= 0; i-- {
		ts := sep15_2210 - int64(i)*intervalMs
		body := 12.0
		seed = append(seed, Bar{T: ts, O: lastReplayC - 3, H: lastReplayC + body/2, L: lastReplayC - body/2, C: lastReplayC, V: 1})
	}
	c.SeedHistorical("MNQ", tf, seed)
	// live from 22:15 09-15 to 09:20 09-16, drifting up to ~29463
	n := int((sep16_0922 - sep15_2215) / intervalMs)
	for i := 0; i < n; i++ {
		ts := sep15_2215 + int64(i)*intervalMs
		px := lastReplayC + 12 + float64(i)*(firstLiveC-lastReplayC-12)/float64(n)
		c.Upsert("MNQ", tf, []Bar{{T: ts, O: px - 2, H: px + 6, L: px - 6, C: px, V: 2}})
	}
	return c
}

func historicalCount(c *BarCache, sym, tf string) int {
	n := 0
	for _, b := range c.Get(sym, tf) {
		if b.Source == BarSourceHistorical {
			n++
		}
	}
	return n
}

// (1) A ZERO-BAR REPLAY MUST NOT RE-ARM THE CHECK. Reproduces 09:22:11 on 5m
// and 1m exactly: a `bars=0` SeedHistorical, then the live 09:22 bar.
func TestZeroBarReplayDoesNotReArmTheScaleCheck(t *testing.T) {
	for _, tc := range []struct {
		tf  string
		ivl int64
	}{{"5m", 5 * 60_000}, {"1m", 60_000}} {
		t.Run(tc.tf, func(t *testing.T) {
			c := sep16Ring(t, tc.tf, tc.ivl)
			// 5m holds all 2000; 1m holds ~1833 because 2000 + 11h of live 1m
			// bars overflows the 2,500 ring — which is exactly the live line's
			// "1832 historical bars DROPPED". The fixture asserts against what
			// the ring MEASURES, not a literal.
			before := historicalCount(c, "MNQ", tc.tf)
			if before < 1800 {
				t.Fatalf("fixture: want a deep seed, got %d historical", before)
			}
			cap := &scaleCapture{}
			cap.listen()
			t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

			// the reconnect's replay: the feed was down, NT8 returned nothing
			c.SeedHistorical("MNQ", tc.tf, nil)
			// the first live bar after it — the overnight move, not a scale
			c.Upsert("MNQ", tc.tf, []Bar{{T: sep16_0922, O: firstLiveC - 4, H: firstLiveC + 3, L: firstLiveC - 5, C: firstLiveC, V: 3}})
			time.Sleep(20 * time.Millisecond)

			// The 1m ring sits AT its 2,500 cap here, so the live upsert trims
			// ONE oldest bar — a cap eviction, not a verdict. The check must not
			// drop the seed: anything beyond that single trim is the defect.
			if got := historicalCount(c, "MNQ", tc.tf); before-got > 1 {
				t.Errorf("09:22:11 %s: a zero-bar replay re-armed the check and the seed was dropped: historical %d → %d", tc.tf, before, got)
			}
			if cap.fired() != 0 {
				t.Errorf("09:22:11 %s: a scale-mismatch listener fired on an eleven-hour-old reference: %+v", tc.tf, cap.events())
			}
		})
	}
}

// (2) ADJACENT BARS ONLY. Even when a NON-empty replay legitimately re-arms the
// check, a reference older than one interval is SKIPPED with a WARN naming the
// ages — never dropped. Here the replay's last bar is eleven hours before the
// live bar it would be judged against.
func TestStaleReferenceIsSkippedNotDropped(t *testing.T) {
	c := sep16Ring(t, "5m", 5*60_000)
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

	// a real (non-empty) re-seed whose newest bar is still the 22:10 bar
	c.SeedHistorical("MNQ", "5m", []Bar{{T: sep15_2210, O: lastReplayC - 3, H: lastReplayC + 6, L: lastReplayC - 6, C: lastReplayC, V: 1}})
	c.Upsert("MNQ", "5m", []Bar{{T: sep16_0922, O: firstLiveC - 4, H: firstLiveC + 3, L: firstLiveC - 5, C: firstLiveC, V: 3}})
	time.Sleep(20 * time.Millisecond)

	if got := historicalCount(c, "MNQ", "5m"); got != 2000 {
		t.Errorf("a reference 11h old must be SKIPPED, not judged: historical 2000 → %d", got)
	}
	if cap.fired() != 0 {
		t.Errorf("no listener may fire on a non-adjacent reference: %+v", cap.events())
	}
	skips := c.ScaleCheckSkips()
	if len(skips) != 1 || skips[0].Timeframe != "5m" || skips[0].ReferenceAge < 10*time.Hour {
		t.Errorf("the skip must be recorded with the reference's age; got %+v", skips)
	}
}

// (2b) A NON-EMPTY RE-SEED THAT LANDS ENTIRELY IN THE PAST — a gap backfill
// for 03:00–04:00 arriving at 09:22 — legitimately re-arms the check, but its
// newest bar is five hours before the live bar. Not adjacent: SKIP, never drop.
func TestPastGapBackfillDoesNotDrop(t *testing.T) {
	c := sep16Ring(t, "5m", 5*60_000)
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

	// 03:00–04:00 CT on 09-16 = 12 five-minute bars, on the live scale of that hour
	gapStart := sep16_0922 - int64(6*60+22)*60_000
	var gap []Bar
	for i := 0; i < 12; i++ {
		ts := gapStart + int64(i)*5*60_000
		px := 29380.0 + float64(i)
		gap = append(gap, Bar{T: ts, O: px - 2, H: px + 5, L: px - 5, C: px, V: 1})
	}
	before := historicalCount(c, "MNQ", "5m")
	c.SeedHistorical("MNQ", "5m", gap)
	c.Upsert("MNQ", "5m", []Bar{{T: sep16_0922, O: firstLiveC - 4, H: firstLiveC + 3, L: firstLiveC - 5, C: firstLiveC, V: 3}})
	time.Sleep(20 * time.Millisecond)

	if got := historicalCount(c, "MNQ", "5m"); got < before {
		t.Errorf("a gap backfill in the past must not cost the seed: historical %d → %d", before, got)
	}
	if cap.fired() != 0 {
		t.Errorf("no listener may fire on a five-hour-old reference: %+v", cap.events())
	}
	if len(c.ScaleCheckSkips()) != 1 {
		t.Errorf("the skip must be recorded once; got %+v", c.ScaleCheckSkips())
	}
}

// (3) A REAL BREAK STILL DROPS. Adjacent bars — the replay's last bar is the
// interval immediately before the live bar — and the live close is 290 pts
// away: that is the 2026-09-10 back-adjusted seed, and it must be dropped.
func TestAdjacentRealBreakStillDrops(t *testing.T) {
	c := NewBarCache(2500)
	var seed []Bar
	for i := 29; i >= 0; i-- {
		seed = append(seed, Bar{T: bsT0 - int64(i+1)*60_000, O: bsLow, H: bsLow + 1, L: bsLow - 1, C: bsLow, V: 1})
	}
	c.SeedHistorical("MNQ", "1m", seed) // last replay bar = bsT0-60s, ADJACENT to bsT0
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsHigh + 1, L: bsLow - 1, C: bsHigh, V: 3}})
	time.Sleep(20 * time.Millisecond)

	if got := historicalCount(c, "MNQ", "1m"); got != 0 {
		t.Errorf("an ADJACENT 290-pt break is a real scale break and must drop the seed; %d historical remain", got)
	}
	if cap.fired() != 1 {
		t.Errorf("the listener must fire exactly once on a real break, got %d", cap.fired())
	}
}

// (1b) RULE 1 ON ITS OWN. The 09:22 fixture is covered by rule 2 as well
// (the reference is eleven hours old), so reverting rule 1 alone survived
// mutation. This is rule 1's independent case: the seed was ALREADY judged
// and passed by the first live bar; a zero-bar re-seed arrives; the next live
// bar is ADJACENT to the seed's tail (rule 2 does not intervene) and carries a
// large but legitimate move. Without rule 1 the verified seed is re-judged
// against that move and dropped. A verdict once given is not re-opened by an
// empty frame.
func TestZeroBarReseedDoesNotRejudgeAVerifiedSeed(t *testing.T) {
	c := NewBarCache(2500)
	var seed []Bar
	for i := 30; i >= 1; i-- { // flat seed, tiny bodies — 20x median body is small
		seed = append(seed, Bar{T: bsT0 - int64(i)*60_000, O: bsLow, H: bsLow + 0.5, L: bsLow - 0.5, C: bsLow, V: 1})
	}
	c.SeedHistorical("MNQ", "1m", seed)
	// first live bar: same scale → the seed is VERIFIED
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0, O: bsLow, H: bsLow + 2, L: bsLow - 1, C: bsLow + 1, V: 2}})
	if chk, off := c.SeedVerdict("MNQ", "1m"); !chk || off {
		t.Fatalf("fixture: seed must be verified on-scale, got checked=%v off=%v", chk, off)
	}
	cap := &scaleCapture{}
	cap.listen()
	t.Cleanup(func() { scaleListenersMu.Lock(); scaleListeners = nil; scaleListenersMu.Unlock() })

	c.SeedHistorical("MNQ", "1m", nil) // reconnect, feed down, nothing returned
	// next live bar, ONE interval after the seed's tail: a 200-pt spike (news)
	c.Upsert("MNQ", "1m", []Bar{{T: bsT0 + 60_000, O: bsLow + 1, H: bsLow + 205, L: bsLow, C: bsLow + 200, V: 9}})
	time.Sleep(20 * time.Millisecond)

	if got := historicalCount(c, "MNQ", "1m"); got != 30 {
		t.Errorf("an empty re-seed re-opened a verdict already given: historical 30 → %d", got)
	}
	if cap.fired() != 0 {
		t.Errorf("no listener may fire — the seed was verified before the empty frame: %+v", cap.events())
	}
}
