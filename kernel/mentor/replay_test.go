package mentor

import (
	"testing"
	"time"

	"vl/market"
)

// TestThirtyDayReplaySeconds: the CTO's parity gate — 30 days of 1m bars
// replayed through the SEEDED evaluator must finish in seconds, not the
// 10-minute timeout DS-105 hit on the unseeded (quadratic) Tick. Duration is
// logged (measure, not a hard assert); a hard ceiling of 120s guards the
// regression class.
func TestThirtyDayReplaySeconds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	var bars1m, bars1h []market.Kline
	cl := 20000.0
	for d := 0; d < 31; d++ {
		// RTH 1m: 08:30–15:00 CT = 390 bars/day
		for m := 0; m < 390; m++ {
			ot := ctMs(t, d, 8, 30) + int64(m)*60_000
			cl += 0.25
			bars1m = append(bars1m, mkBar(cl-0.125, cl, ot, 1))
		}
	}
	// 19 days of 1h candles around the clock (RTH filtered inside the walk):
	// 114 closed 4h buckets — past the 102 warm-up.
	for d := 0; d < 19; d++ {
		for h := 0; h < 24; h++ {
			ot := ctMs(t, d, h, 0)
			c := 20000 + float64(d)*12 + float64(h)
			bars1h = append(bars1h, mkBar(c-0.5, c, ot, 60))
		}
	}
	now := ctMs(t, 30, 15, 0)

	e := New(cfg)
	if m := Seed(e, bars1m, bars1h, now); len(m) != 0 {
		t.Fatalf("30-day store must be warm, got %v", m)
	}

	start := time.Now()
	var intents int
	// Replay the last 3 days tick-by-tick (the live pattern: a capped slice
	// plus the seeded state), the worst-case slice = the full tape.
	for i := 2; i <= len(bars1m); i++ {
		intents += len(e.Tick(bars1m[:i], bars1m[i-1].CloseTime+1))
	}
	elapsed := time.Since(start)
	t.Logf("30-day seeded replay: %d ticks, %d intents, %s", len(bars1m)-1, intents, elapsed)
	if elapsed > 120*time.Second {
		t.Fatalf("30-day replay took %s — the quadratic class is back", elapsed)
	}
}
