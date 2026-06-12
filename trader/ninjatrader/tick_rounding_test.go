package ninjatrader

import "testing"

func TestRoundToTick(t *testing.T) {
	// Note: math.Round in Go is round-half-AWAY-FROM-ZERO, not banker's rounding.
	// (math.RoundToEven would be banker's.) For tick rounding this is fine —
	// CME only cares that the resulting price is on a tick boundary; the bias
	// direction at exact halves doesn't matter operationally.
	cases := []struct {
		in, tick, want float64
	}{
		{21503.17, 0.25, 21503.25},  // nearest tick boundary is 21503.25 (dist 0.08)
		{21503.13, 0.25, 21503.25},  // nearest is 21503.25 (dist 0.12 vs 0.13 to 21503.00)
		{21503.125, 0.25, 21503.25}, // exact halfway → math.Round rounds away from zero
		{21503.10, 0.25, 21503.00},  // round down (dist 0.10 vs 0.15)
		{21500.0, 0.25, 21500.00},   // exact tick boundary, unchanged
	}
	for _, tc := range cases {
		if got := RoundToTick(tc.in, tc.tick); got != tc.want {
			t.Errorf("RoundToTick(%v, %v) = %v, want %v", tc.in, tc.tick, got, tc.want)
		}
	}
}

func TestRoundToTickZeroOrNegativeTick(t *testing.T) {
	cases := []struct {
		in, tick float64
	}{
		{21503.17, 0},
		{21503.17, -0.25},
	}
	for _, tc := range cases {
		if got := RoundToTick(tc.in, tc.tick); got != tc.in {
			t.Errorf("RoundToTick(%v, %v) = %v, want %v (unchanged)", tc.in, tc.tick, got, tc.in)
		}
	}
}

func TestInstrumentTickSize(t *testing.T) {
	cases := []struct {
		symbol string
		want   float64
	}{
		{"NQ", 0.25},
		{"MNQ", 0.25},
		{"ES", 0.25},
		{"MES", 0.25},
		{"YM", 1.0},
		{"MYM", 1.0},
		{"RTY", 0.10},
		{"M2K", 0.10},
		{"CL", 0.01},
		{"GC", 0.10},
		{"UNKNOWN", 0.25}, // default
		{"", 0.25},        // empty = default
	}
	for _, tc := range cases {
		if got := InstrumentTickSize(tc.symbol); got != tc.want {
			t.Errorf("InstrumentTickSize(%q) = %v, want %v", tc.symbol, got, tc.want)
		}
	}
}
