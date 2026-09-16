package trader

import "testing"

// limitClosePrice is the pure 4.3 boundary: LONG exits ABOVE the reference
// (favorable side), SHORT BELOW; garbage inputs return 0 (caller market-falls).
func TestLimitClosePrice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		close float64
		ticks int
		tick  float64
		long  bool
		want  float64
	}{
		{"long above", 30300.0, 4, 0.25, true, 30301.0},
		{"short below", 30300.0, 4, 0.25, false, 30299.0},
		{"zero ticks dormant", 30300.0, 0, 0.25, true, 0},
		{"zero tick size", 30300.0, 4, 0, true, 0},
		{"zero close", 0, 4, 0.25, true, 0},
	} {
		if got := limitClosePrice(tc.close, tc.ticks, tc.tick, tc.long); got != tc.want {
			t.Errorf("%s: got %.4f want %.4f", tc.name, got, tc.want)
		}
	}
}
