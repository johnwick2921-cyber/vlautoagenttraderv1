package market

import (
	"math"
	"testing"
)

// TestPriceToTicks verifies round-trip conversion for realistic NQ prices.
// NQ tick size is 0.25 (4 ticks per point).
func TestPriceToTicks(t *testing.T) {
	const tick = 0.25

	cases := []struct {
		name      string
		price     float64
		wantTicks int64
	}{
		{"NQ 21503.25", 21503.25, 86013},
		{"NQ 21503.50", 21503.50, 86014},
		{"NQ 21500.00", 21500.00, 86000},
		{"NQ 0",        0.00,     0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PriceToTicks(tc.price, tick)
			if got != tc.wantTicks {
				t.Fatalf("PriceToTicks(%v) = %d, want %d", tc.price, got, tc.wantTicks)
			}
			back := TicksToPrice(got, tick)
			if back != tc.price {
				t.Fatalf("round-trip lost precision: %v -> %d -> %v", tc.price, got, back)
			}
		})
	}
}

// TestPriceToTicks_ZeroTick guards the defensive branch.
func TestPriceToTicks_ZeroTick(t *testing.T) {
	if got := PriceToTicks(21500.25, 0); got != 0 {
		t.Fatalf("PriceToTicks with tick=0 should return 0, got %d", got)
	}
	if got := PriceToTicks(21500.25, -1); got != 0 {
		t.Fatalf("PriceToTicks with tick<0 should return 0, got %d", got)
	}
}

// TestTicksToPrice_AccumulatedDrift is the whole point of int64-ticks.
// Adding 0.10 three thousand times in raw float64 accumulates measurable
// drift (0.10 is NOT a binary fraction — repeating in IEEE-754). Doing
// the same sum in tick-space is exact.
//
// Note: 0.25 (= 2^-2) IS exactly representable, so accumulating it does
// not drift. We deliberately use 0.10 so the drift is observable.
func TestTicksToPrice_AccumulatedDrift(t *testing.T) {
	const tick = 0.10
	const iter = 3000

	// Tick-space accumulation
	var ticks int64
	for i := 0; i < iter; i++ {
		ticks += PriceToTicks(0.10, tick)
	}
	tickResult := TicksToPrice(ticks, tick)

	// Raw float accumulation (the drift baseline)
	var floatSum float64
	for i := 0; i < iter; i++ {
		floatSum += 0.10
	}

	expected := float64(iter) * tick // == 300.0

	// Tick-space MUST be exact in tick-count space; the conversion back is
	// lossless for tick counts within float64 mantissa range. We compare
	// in tick-count space to avoid any representation gotchas:
	if PriceToTicks(tickResult, tick) != int64(iter) {
		t.Fatalf("tick-space accumulation drifted: %d ticks, want %d", PriceToTicks(tickResult, tick), int64(iter))
	}

	// Raw float MUST exhibit drift — if it doesn't, our test premise is
	// wrong and the comparison is meaningless.
	if floatSum == expected {
		t.Fatalf("raw float 0.10*%d unexpectedly exact; drift premise broken", iter)
	}

	t.Logf("DRIFT REPORT (tick=%.2f, iter=%d):", tick, iter)
	t.Logf("  expected   = %.20f", expected)
	t.Logf("  tick-space = %.20f  (ticks=%d)", tickResult, PriceToTicks(tickResult, tick))
	t.Logf("  raw float  = %.20f  (drift = %.20e)", floatSum, math.Abs(floatSum-expected))
}

// TestPositionSize_NQ — canonical example from the plan.
// $500 risk, 12 ticks of stop, $5/tick (NQ) → floor(500 / 60) = 8 contracts.
func TestPositionSize_NQ(t *testing.T) {
	got := PositionSize(500.0, 12.0, 5.0)
	want := 8
	if got != want {
		t.Fatalf("PositionSize(500, 12, 5) = %d, want %d", got, want)
	}
}

// TestPositionSize_CantAffordOne — must refuse to enter when even one
// contract exceeds the risk budget. floor(100 / 125) = 0.
func TestPositionSize_CantAffordOne(t *testing.T) {
	got := PositionSize(100.0, 25.0, 5.0)
	if got != 0 {
		t.Fatalf("PositionSize(100, 25, 5) = %d, want 0 (must not over-leverage)", got)
	}
}

// TestPositionSize_ZeroInputs — any zero/negative input → 0.
func TestPositionSize_ZeroInputs(t *testing.T) {
	cases := []struct {
		risk, stop, dpt float64
	}{
		{0, 12, 5},
		{500, 0, 5},
		{500, 12, 0},
		{-1, 12, 5},
		{500, -1, 5},
		{500, 12, -1},
	}
	for _, tc := range cases {
		if got := PositionSize(tc.risk, tc.stop, tc.dpt); got != 0 {
			t.Fatalf("PositionSize(%v,%v,%v) = %d, want 0", tc.risk, tc.stop, tc.dpt, got)
		}
	}
}

// TestSafeAdd_NoDrift — proves repeated SafeAdd never drifts off the tick
// grid, while repeated raw float addition does. The single-call case
// (SafeAdd(0.1, 0.2)) returns the same bit pattern as Go's `0.30` literal
// — both round to the same IEEE-754 value — so we can't show divergence in
// a single addition. The win shows up in accumulation:
//
//   - SafeAdd: each step quantizes to the tick grid, so error never grows.
//   - raw +:   error compounds linearly with the iteration count.
func TestSafeAdd_NoDrift(t *testing.T) {
	const tick = 0.10
	const iter = 1000

	safe := 0.0
	raw := 0.0
	for i := 0; i < iter; i++ {
		safe = SafeAdd(safe, 0.10, tick)
		raw = raw + 0.10
	}

	expected := float64(iter) * tick // 100.0

	// SafeAdd result must land exactly on the tick grid.
	if PriceToTicks(safe, tick) != int64(iter) {
		t.Fatalf("SafeAdd drifted off tick grid: got %.20f (%d ticks), want %d ticks",
			safe, PriceToTicks(safe, tick), int64(iter))
	}

	// Raw float MUST drift on this platform — premise check.
	if raw == expected {
		t.Fatalf("raw float accumulation unexpectedly exact; drift premise broken")
	}

	t.Logf("SafeAdd vs raw float (iter=%d, tick=%.2f):", iter, tick)
	t.Logf("  expected = %.20f", expected)
	t.Logf("  SafeAdd  = %.20f  (drift = %.20e)", safe, math.Abs(safe-expected))
	t.Logf("  raw +    = %.20f  (drift = %.20e)", raw, math.Abs(raw-expected))
}

// TestSafeSubtract_NoDrift — same shape as SafeAdd, but for subtraction.
func TestSafeSubtract_NoDrift(t *testing.T) {
	const tick = 0.25
	// NQ: entry 21503.25, stop 21500.25 → 3.00 distance, exactly.
	got := SafeSubtract(21503.25, 21500.25, tick)
	if got != 3.00 {
		t.Fatalf("SafeSubtract = %.20f, want exactly 3.00", got)
	}
}

// TestSafeMultiply_Ticks — "entry + 12 ticks" style ops on NQ.
func TestSafeMultiply_Ticks(t *testing.T) {
	const tick = 0.25
	// entry 21500.00 + 12 ticks * 0.25 = 21503.00
	got := SafeMultiply(21500.00, 12, tick)
	if got != 21503.00 {
		t.Fatalf("SafeMultiply(21500, +12, 0.25) = %.20f, want 21503.00", got)
	}
	// entry 21500.00 - 8 ticks * 0.25 = 21498.00
	got = SafeMultiply(21500.00, -8, tick)
	if got != 21498.00 {
		t.Fatalf("SafeMultiply(21500, -8, 0.25) = %.20f, want 21498.00", got)
	}
}

// TestDecimalSafe_Suite is a sanity umbrella so `go test -run TestDecimalSafe`
// catches the whole module via one regex.
func TestDecimalSafe_Suite(t *testing.T) {
	t.Run("PriceToTicks_NQ", func(t *testing.T) {
		if PriceToTicks(21503.25, 0.25) != 86013 {
			t.Fatal("PriceToTicks drift")
		}
	})
	t.Run("PositionSize_NQ", func(t *testing.T) {
		if PositionSize(500, 12, 5) != 8 {
			t.Fatal("PositionSize drift")
		}
	})
}
