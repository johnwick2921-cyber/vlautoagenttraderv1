package mentor

import (
	"testing"
	"time"

	"vl/market"
)

// limitsK builds a 1m candle.
func limitsK(o, h, l, c float64, minute int) market.Kline {
	ct := time.Date(2026, 10, 2, 9, minute, 0, 0, time.UTC).UnixMilli()
	return market.Kline{Open: o, High: h, Low: l, Close: c, OpenTime: ct, CloseTime: ct + 60_000}
}

// limitsNow is a CT-morning timestamp matching the candle series above.
func limitsNow(minute int) int64 {
	return time.Date(2026, 10, 2, 9, minute, 0, 0, time.UTC).UnixMilli()
}

var limitsLvls = []Level{{Key: "old_extreme:100", Kind: KindOldExtreme, Price: 100}}

// limitsLvlsG2 adds a KEY LEVEL at 97 (distinct from the old extreme 100) so
// the G2 tests prove the loss box keys on the setup's PLACE, not the extreme.
var limitsLvlsG2 = []Level{
	{Key: "key_level:97", Kind: KindKeyLevel, Price: 97},
	{Key: "old_extreme:100", Kind: KindOldExtreme, Price: 100},
}

func limitsCfg() Config {
	cfg := DefaultConfig()
	cfg.Enabled = true
	return cfg
}

// limitsCfgNoLeg is G2-only: the G1 leg budget is off.
func limitsCfgNoLeg() Config {
	cfg := limitsCfg()
	cfg.LegBudgetEnabled = false
	return cfg
}

// limitsPHL is a long PHL entry anchored at the old extreme 100.
func limitsPHL(entry, stop, target float64, expiry int64) Intent {
	return Intent{
		Action: PlaceStopEntry, Side: SideLong,
		Price: entry, Stop: stop, Target: target, ExpiryMs: expiry,
	}
}

// limitsPHLAt is a long PHL entry at a named place (the touch level).
func limitsPHLAt(entry, stop, target float64, expiry int64, place string, anchor float64) Intent {
	return Intent{
		Action: PlaceStopEntry, Side: SideLong,
		Price: entry, Stop: stop, Target: target, ExpiryMs: expiry,
		Anchor: anchor, AnchorKey: place,
	}
}

// limitsISB is a long ISB entry.
func limitsISB(entry, stop, target float64, expiry int64) Intent {
	return Intent{
		Action: PlaceStopLimitEntry, Side: SideLong,
		Price: entry, Stop: stop, Target: target, ExpiryMs: expiry,
	}
}

func applyAt(l *Limits, in []Intent, prev, cur market.Kline, minute int) []Intent {
	return l.Apply(in, prev, cur, limitsNow(minute), limitsLvls, limitsCfg())
}

func applyAtCfg(l *Limits, in []Intent, prev, cur market.Kline, minute int, cfg Config) []Intent {
	return l.Apply(in, prev, cur, limitsNow(minute), limitsLvls, cfg)
}

func applyAtG2(l *Limits, in []Intent, prev, cur market.Kline, minute int, cfg Config) []Intent {
	return l.Apply(in, prev, cur, limitsNow(minute), limitsLvlsG2, cfg)
}

// G1 (a): the third entry in one leg is refused — the PHL fills, one
// same-direction ISB fills, and the next ISB hits the budget.
// MUTANT: change the ISB budget check to `leg.Entries >= 3` → this test
// goes RED (the third entry is emitted).
func TestLimitsThirdEntryInLegRefused(t *testing.T) {
	var l Limits
	prev := limitsK(94, 96, 94, 95, 0)
	exp := limitsNow(60) + 86400_000

	out1 := applyAt(&l, []Intent{limitsPHL(90, 88, 99, exp)}, prev, limitsK(93, 94, 92, 93, 1), 1)
	if len(out1) != 1 {
		t.Fatalf("PHL placement: want 1 entry, got %d", len(out1))
	}
	out2 := applyAt(&l, []Intent{limitsISB(92, 89, 98, exp)}, limitsK(93, 94, 92, 93, 1), limitsK(89, 91, 88.5, 89, 2), 2)
	if len(out2) != 1 {
		t.Fatalf("first ISB: want 1 entry, got %d", len(out2))
	}
	if l.Long == nil || l.Long.Entries != 1 {
		t.Fatalf("leg after PHL fill: want entries=1, got %+v", l.Long)
	}
	out3 := applyAt(&l, []Intent{limitsISB(93, 90, 99, exp)}, limitsK(89, 91, 88.5, 89, 2), limitsK(91, 93, 90, 92, 3), 3)
	if len(out3) != 0 {
		t.Fatalf("second ISB: want 0 entries (budget full), got %d", len(out3))
	}
	if l.Long == nil || l.Long.Entries != 2 {
		t.Fatalf("leg after ISB fill: want entries=2, got %+v", l.Long)
	}
}

// G1 (b): a stop-out inside the leg closes it — no more entries until a
// break beyond the extreme.
// MUTANT: delete the `leg.Stopped = true` in loss() → this test goes RED
// (the post-stopout ISB is emitted).
func TestLimitsStopOutClosesLeg(t *testing.T) {
	var l Limits
	prev := limitsK(94, 96, 94, 95, 0)
	exp := limitsNow(60) + 86400_000

	if out := applyAt(&l, []Intent{limitsPHL(90, 88, 99, exp)}, prev, limitsK(93, 94, 92, 93, 1), 1); len(out) != 1 {
		t.Fatalf("PHL placement: want 1 entry, got %d", len(out))
	}
	// The fill candle trades through the entry AND the stop: filled, then
	// stopped out — a loss that closes the leg.
	out2 := applyAt(&l, []Intent{limitsISB(92, 89, 98, exp)}, limitsK(93, 94, 92, 93, 1), limitsK(89, 91, 87, 88, 2), 2)
	if len(out2) != 0 {
		t.Fatalf("ISB after stopout: want 0 entries (leg closed), got %d", len(out2))
	}
	if l.Long == nil || !l.Long.Stopped {
		t.Fatalf("leg: want Stopped=true, got %+v", l.Long)
	}
}

// G1 (c): a 1m CLOSE beyond the prior high opens a NEW leg — after the
// stop-out, a close above the extreme resets and the next PHL is allowed.
// MUTANT: delete the long-leg reset line in resetBreaks → this test goes RED
// (the PHL after the break is refused).
func TestLimitsCloseBeyondExtremeOpensNewLeg(t *testing.T) {
	var l Limits
	prev := limitsK(94, 96, 94, 95, 0)
	exp := limitsNow(60) + 86400_000

	applyAt(&l, []Intent{limitsPHL(90, 88, 99, exp)}, prev, limitsK(93, 94, 92, 93, 1), 1)
	applyAt(&l, nil, limitsK(93, 94, 92, 93, 1), limitsK(89, 91, 87, 88, 2), 2) // fill + stop-out
	if l.Long == nil || !l.Long.Stopped {
		t.Fatalf("leg: want Stopped=true, got %+v", l.Long)
	}
	// prev closes at 101 — strictly beyond the prior high 100 — the new leg
	// (the break clears the leg; the loss place clears on its own candle).
	applyAt(&l, nil, limitsK(100, 101.5, 99, 101, 3), limitsK(98, 99, 97, 98, 4), 4)
	if l.Long != nil {
		t.Fatalf("leg: want the stopped leg cleared by the break, got %+v", l.Long)
	}
	// The candle after the break departs the lost place — the next PHL is a
	// new leg.
	out5 := applyAt(&l, []Intent{limitsPHL(90, 88, 99, exp)}, limitsK(98, 99, 97, 98, 4), limitsK(94, 95, 93, 94, 5), 5)
	if len(out5) != 1 {
		t.Fatalf("PHL after the break: want 1 entry (new leg), got %d", len(out5))
	}
	// The leg is created at FILL, not placement — the fresh placement has
	// not filled yet.
	if l.Long != nil {
		t.Fatalf("leg: want nil before the fill, got %+v", l.Long)
	}
}

// G2 (d): a loss at a level blocks re-entry there until price departs —
// the departure is a closed candle whose range does NOT touch the place
// (the same rule as DS-103's E2). G1 is off to isolate G2.
// MUTANT: delete the departure unblock in Apply → this test goes RED (the
// post-departure PHL stays refused).
func TestLimitsLossBlocksUntilDeparture(t *testing.T) {
	var l Limits
	cfg := limitsCfgNoLeg() // G2 only — the leg budget must not shadow it
	prev := limitsK(94, 96, 94, 95, 0)
	exp := limitsNow(60) + 86400_000

	applyAtG2(&l, []Intent{limitsPHLAt(90, 88, 99, exp, "key_level:97", 97)}, prev, limitsK(93, 94, 92, 93, 1), 1, cfg)
	applyAtG2(&l, nil, limitsK(93, 94, 92, 93, 1), limitsK(89, 91, 87, 88, 2), 2, cfg) // fill + stop-out → loss at 97

	// A candle that still TOUCHES the place keeps it blocked.
	out3 := applyAtG2(&l, []Intent{limitsPHLAt(90, 88, 99, exp, "key_level:97", 97)}, limitsK(89, 91, 87, 88, 2), limitsK(99, 100.2, 90, 99, 3), 3, cfg)
	if len(out3) != 0 {
		t.Fatalf("re-entry at the blocked place: want 0 entries, got %d", len(out3))
	}
	// A departure candle (range below the place) clears the block.
	out4 := applyAtG2(&l, []Intent{limitsPHLAt(90, 88, 99, exp, "key_level:97", 97)}, limitsK(99, 100.2, 90, 99, 3), limitsK(94, 95, 93, 94, 4), 4, cfg)
	if len(out4) != 1 {
		t.Fatalf("re-entry after departure: want 1 entry, got %d", len(out4))
	}
}

// G2 (e): two losses at one place -> that place is OFF FOR THE DAY — even a
// departure does not clear it; the next trading day does. G1 is off.
// MUTANT: change the off-for-day threshold to `p.Losses > 2` → this test
// goes RED (the third re-entry is emitted).
func TestLimitsTwoLossesOffForDay(t *testing.T) {
	var l Limits
	cfg := limitsCfgNoLeg() // G2 only — the leg budget must not shadow it
	prev := limitsK(94, 96, 94, 95, 0)
	exp := limitsNow(60) + 86400_000

	// Loss 1 at the key level 97 (the loss candle never counts as the
	// departure).
	applyAtG2(&l, []Intent{limitsPHLAt(90, 88, 99, exp, "key_level:97", 97)}, prev, limitsK(93, 94, 92, 93, 1), 1, cfg)
	applyAtG2(&l, nil, limitsK(93, 94, 92, 93, 1), limitsK(89, 91, 87, 88, 2), 2, cfg)
	// Departure candle 1 (clears the just-flag), departure candle 2
	// (unblocks) — then loss 2.
	applyAtG2(&l, nil, limitsK(89, 91, 87, 88, 2), limitsK(94, 95, 93, 94, 3), 3, cfg)
	if out := applyAtG2(&l, []Intent{limitsPHLAt(90, 88, 99, exp, "key_level:97", 97)}, limitsK(94, 95, 93, 94, 3), limitsK(92, 93, 91, 92, 4), 4, cfg); len(out) != 1 {
		t.Fatalf("re-entry after departure: want 1 entry, got %d", len(out))
	}
	applyAtG2(&l, nil, limitsK(92, 93, 91, 92, 4), limitsK(89, 91, 87, 88, 5), 5, cfg)
	if l.Places == nil || l.Places["key_level:97"] == nil || !l.Places["key_level:97"].OffDay {
		t.Fatalf("place after two losses: want OffDay=true, got %+v", l.Places)
	}
	// Departures do NOT clear an off-for-day place.
	applyAtG2(&l, nil, limitsK(89, 91, 87, 88, 5), limitsK(94, 95, 93, 94, 6), 6, cfg)
	out7 := applyAtG2(&l, []Intent{limitsPHLAt(90, 88, 99, exp, "key_level:97", 97)}, limitsK(94, 95, 93, 94, 6), limitsK(92, 93, 91, 92, 7), 7, cfg)
	if len(out7) != 0 {
		t.Fatalf("same-day re-entry after two losses: want 0 entries, got %d", len(out7))
	}
	// The next trading day clears the registry.
	next := time.Date(2026, 10, 3, 9, 8, 0, 0, time.UTC).UnixMilli()
	out8 := l.Apply([]Intent{limitsPHLAt(90, 88, 99, next+86400_000, "key_level:97", 97)}, limitsK(92, 93, 91, 92, 7), limitsK(93, 94, 92, 93, 8), next, limitsLvlsG2, cfg)
	if len(out8) != 1 {
		t.Fatalf("next-day re-entry: want 1 entry, got %d", len(out8))
	}
}
