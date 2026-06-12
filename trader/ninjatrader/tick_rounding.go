// Package ninjatrader — tick-size rounding for CME instruments.
//
// AI decisions return floating-point prices (e.g. 21503.17). CME rejects
// orders that aren't on a tick boundary, so we round before writing the
// CSV signal. NQ/MNQ/ES/MES tick = 0.25 (4 ticks per point).
package ninjatrader

import "math"

// InstrumentTickSize returns the tick size in points for a CME instrument.
// Returns 0.25 for NQ/MNQ/ES/MES (index futures default). Other instruments
// can be added as needed.
func InstrumentTickSize(symbol string) float64 {
	switch symbol {
	case "NQ", "MNQ", "ES", "MES":
		return 0.25
	case "YM", "MYM":
		return 1.0
	case "RTY", "M2K":
		return 0.10
	case "CL": // crude oil
		return 0.01
	case "GC": // gold
		return 0.10
	default:
		return 0.25 // safe default for indices
	}
}

// RoundToTick rounds price to the nearest tick boundary.
// Uses math.Round (round-half-away-from-zero). For tick rounding the bias
// at exact halves is operationally irrelevant — CME only requires the price
// land on a tick boundary. If tick <= 0, returns price unchanged.
func RoundToTick(price, tick float64) float64 {
	if tick <= 0 {
		return price
	}
	return math.Round(price/tick) * tick
}
