package ninjatrader

import "testing"

// W1b FOLD-7 — WireMoved is the gate's "did rounding move this price" test.
// It must say NO for an on-grid price on a non-power-of-two tick (where the
// float round trip lands a few ulps off), YES for a real one-tick move, and
// agree with != on an exact tick and when there is no tick at all.
func TestWireMovedJudgesMovementWithinTheWireTickEps(t *testing.T) {
	cases := []struct {
		name           string
		authored, wire float64
		tick           float64
		want           bool
	}{
		{"on-grid 0.10 entry (float round trip is a few ulps off)", 2000.3, RoundToTick(2000.3, 0.10), 0.10, false},
		{"on-grid 0.10 long stop", 1990.3, mustWireStop(t, "long", 1990.3, 0.10), 0.10, false},
		{"on-grid 0.10 short stop", 2010.3, mustWireStop(t, "short", 2010.3, 0.10), 0.10, false},
		{"off-grid 0.10 stop moves to the grid", 1990.33, mustWireStop(t, "long", 1990.33, 0.10), 0.10, true},
		{"on-grid 0.25 is bit-identical", 29600.25, RoundToTick(29600.25, 0.25), 0.25, false},
		{"off-grid 0.25 entry moves", 29600.12, RoundToTick(29600.12, 0.25), 0.25, true},
		{"no tick: exact compare, equal", 0.12, 0.12, 0, false},
		{"no tick: exact compare, differs", 0.12, 0.1200000001, 0, true},
	}
	for _, c := range cases {
		if got := WireMoved(c.authored, c.wire, c.tick); got != c.want {
			t.Errorf("%s: WireMoved(%v, %v, %v) = %v, want %v", c.name, c.authored, c.wire, c.tick, got, c.want)
		}
	}
}

func mustWireStop(t *testing.T, side string, stop, tick float64) float64 {
	t.Helper()
	ws, err := WireStop(side, stop, tick)
	if err != nil {
		t.Fatalf("WireStop(%q, %v, %v): %v", side, stop, tick, err)
	}
	return ws
}
