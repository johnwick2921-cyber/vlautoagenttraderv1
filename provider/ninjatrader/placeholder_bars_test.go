package ninjatrader

import "testing"

// NO SYNTHETIC BARS, EVER (P0 2026-08-17) — the weekend-boundary fixture.
//
// The owner's report: "over the weekend the chart draws ONE FLAT SIDEWAYS LINE
// across the no-data period instead of skipping the empty time and continuing
// with the next real candle, timestamps intact."
//
// Measured cause: NT8's own minute store keeps empty-minute PLACEHOLDER records
// and its bar builder materialises each as O=H=L=C=<file base price>, volume 0.
// After the AddOn watchdog livelock forced NT8 to re-fetch Friday 2026-08-14
// into an all-placeholder file (10,611 → 4,346 bytes = 32-byte header + 1438
// identical 3-byte records, base price 30147.50), /api/klines returned 959 of
// 1500 one-minute bars flat at exactly 30147.50 with zero volume, contiguous
// 00:02 → 16:00 CT. Every hop we own is a verbatim pass-through, so we did not
// invent them — but we accepted them, and they reached both the chart and the
// kernel's detectors.

const (
	friClose = int64(1_755_205_200_000) // Fri 2026-08-14 16:00 CT
	sunOpen  = int64(1_755_378_000_000) // Sun 2026-08-16 17:00 CT
)

func realBar(t int64, px float64) Bar {
	return Bar{T: t, O: px, H: px + 2, L: px - 2, C: px + 1, V: 120}
}

// NT8's placeholder: no range, no volume, the file's base price.
func placeholder(t int64, base float64) Bar {
	return Bar{T: t, O: base, H: base, L: base, C: base, V: 0}
}

func TestWeekendGapStaysAGapNotAFlatLine(t *testing.T) {
	c := NewBarCache(5000)

	var seed []Bar
	seed = append(seed,
		realBar(friClose-120_000, 30140),
		realBar(friClose-60_000, 30142),
	)
	// The 16-hour flat block NT8 handed us across the closed market.
	for ms := friClose; ms < sunOpen; ms += 60_000 {
		seed = append(seed, placeholder(ms, 30147.5))
	}
	seed = append(seed,
		realBar(sunOpen, 30160),
		realBar(sunOpen+60_000, 30163),
	)
	c.SeedHistorical("MNQ", "1m", seed)

	got := c.Get("MNQ", "1m")

	// (1) No bar may exist with a synthetic timestamp inside the closed period.
	for _, b := range got {
		if b.T >= friClose && b.T < sunOpen {
			t.Fatalf("a bar survived inside the closed market at t=%d (px %g) — no synthetic bars, ever", b.T, b.C)
		}
	}

	// (2) The series jumps Friday close → Sunday open with no flat segment.
	if len(got) != 4 {
		t.Fatalf("want the 4 REAL bars, got %d", len(got))
	}
	if got[1].T != friClose-60_000 || got[2].T != sunOpen {
		t.Errorf("the series must step straight from the last Friday bar to the first Sunday bar, got %d → %d",
			got[1].T, got[2].T)
	}
	for _, b := range got {
		if b.H == b.L {
			t.Errorf("a flat bar survived at t=%d — this is the sideways line the owner saw", b.T)
		}
	}
	// Timestamps intact: nothing was re-stamped or shifted to close the gap.
	if got[0].T != friClose-120_000 || got[3].T != sunOpen+60_000 {
		t.Errorf("real bars must keep their own timestamps, got %d..%d", got[0].T, got[3].T)
	}

	if n := c.DroppedPlaceholders(); n != int64(len(seed)-4) {
		t.Errorf("dropped counter = %d, want %d — the refusal must be observable, not silent", n, len(seed)-4)
	}
}

func TestPlaceholderFilterKeepsGenuinelyThinBars(t *testing.T) {
	// The test is deliberately narrow: BOTH zero volume AND zero range. A real
	// but illiquid minute that still printed a range must survive, and so must a
	// zero-volume bar that carries one — dropping those would be inventing a
	// different lie.
	keep := []Bar{
		{T: 1, O: 30100, H: 30102, L: 30099, C: 30101, V: 0}, // no volume, real range
		{T: 2, O: 30100, H: 30100, L: 30100, C: 30100, V: 3}, // no range, real volume
		{T: 3, O: 30100, H: 30101, L: 30100, C: 30100, V: 1}, // thin but real
	}
	got, dropped := dropPlaceholderBars(keep)
	if dropped != 0 || len(got) != 3 {
		t.Errorf("kept %d/%d bars (dropped %d) — only a bar with NO volume AND NO range is a placeholder",
			len(got), len(keep), dropped)
	}

	// A doji with real volume is a real bar and must never be confused with one.
	if isPlaceholderBar(Bar{T: 9, O: 30100, H: 30100, L: 30100, C: 30100, V: 42}) {
		t.Error("a zero-range bar WITH volume is a real doji, not a placeholder")
	}
	if !isPlaceholderBar(Bar{T: 9, O: 30147.5, H: 30147.5, L: 30147.5, C: 30147.5, V: 0}) {
		t.Error("the exact shape NT8 emitted 959 times must be recognised")
	}
}

func TestPlaceholderFilterOnStreamingUpdates(t *testing.T) {
	// The same shape arrives on bar_update while a declared-open session has no
	// ticks; an all-placeholder update must leave the cache untouched rather than
	// appending a flat bar onto the tail.
	c := NewBarCache(100)
	c.SeedHistorical("MNQ", "1m", []Bar{realBar(1000, 30100)})
	c.Upsert("MNQ", "1m", []Bar{placeholder(2000, 30147.5), placeholder(3000, 30147.5)})

	got := c.Get("MNQ", "1m")
	if len(got) != 1 || got[0].T != 1000 {
		t.Fatalf("placeholder updates must not reach the cache, got %+v", got)
	}
	if c.DroppedPlaceholders() != 2 {
		t.Errorf("dropped counter = %d, want 2", c.DroppedPlaceholders())
	}

	// A real bar still appends normally afterwards.
	c.Upsert("MNQ", "1m", []Bar{realBar(4000, 30155)})
	if got := c.Get("MNQ", "1m"); len(got) != 2 || got[1].T != 4000 {
		t.Errorf("a real bar after a placeholder run must still append, got %+v", got)
	}
}
