package kernel

import "testing"

// GAR-F2 (grand-audit response, 2026-08-28) — the ONE day-trade-lock clamp.
// The engine prompt path alone still had a 0.5 floor, so a 0.3 owner retune
// would have been honored by the bot gate and silently dropped by the engine.
// Both consumers now resolve through ResolveProximityK (0.1–3.0).
func TestResolveProximityK(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0.3, 0.3}, // the owner retune value — must survive on every path
		{1.5, 1.5}, // current live value
		{0.1, 0.1}, // UI range floor
		{3.0, 3.0}, // UI range ceiling
		{0.05, 1.5},
		{3.5, 1.5},
		{0, 1.5},
		{-1, 1.5},
	}
	for _, c := range cases {
		if got := ResolveProximityK(c.in); got != c.want {
			t.Errorf("ResolveProximityK(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
