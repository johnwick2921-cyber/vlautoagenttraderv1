// Package ninjatrader — tick-size rounding for CME instruments.
//
// AI decisions return floating-point prices (e.g. 21503.17). CME rejects
// orders that aren't on a tick boundary, so we round before writing the
// CSV signal. NQ/MNQ/ES/MES tick = 0.25 (4 ticks per point).
package ninjatrader

import (
	"fmt"
	"math"
	"strings"
)

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
// Uses math.Round (round-half-away-from-zero). CME only requires the price
// land on a tick boundary, so nearest is right for an entry or a target — but
// NOT for a protective stop judged against a floor: nearest can move it toward
// the entry. Entry stops go through WireStop / WireBracket (W1b E12(a)).
// If tick <= 0, returns price unchanged.
func RoundToTick(price, tick float64) float64 {
	if tick <= 0 {
		return price
	}
	return math.Round(price/tick) * tick
}

// wireTickEps keeps an ON-GRID price fixed under Floor/Ceil. price/tick is not
// exact in binary for every tick (2000.3/0.10 = 20002.999999999996), so a bare
// Floor would move an on-grid stop a whole tick. 1e-6 of a tick is far below
// any price the wire can carry and far above float noise at index levels.
const wireTickEps = 1e-6

// WireStop is the stop the wire sends (W1b E12(a)): on the tick grid and
// rounded AWAY from the entry — a long's stop DOWN, a short's stop UP — so the
// order NT8 receives is never tighter than the one the gate approved. Nearest
// rounding (RoundToTick) put a stop composed exactly on the min-SL floor up to
// half a tick INSIDE it (29575.90 → 29576.00 on a 24.10 floor = 24.00).
//
// An on-grid stop is returned byte-identical to RoundToTick's result (same
// integer × tick). side accepts long/short in any case; anything else
// (buy/sell included — the AddOn reads side == "long" ? Buy : SellShort, so a
// "buy" would not even be sent as a buy) is an ERROR, and the caller refuses
// the send rather than guess a direction. tick <= 0 returns the price
// unchanged, like RoundToTick.
func WireStop(side string, stop, tick float64) (float64, error) {
	if tick <= 0 {
		return stop, nil
	}
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "long":
		return math.Floor(stop/tick+wireTickEps) * tick, nil
	case "short":
		return math.Ceil(stop/tick-wireTickEps) * tick, nil
	default:
		return 0, fmt.Errorf("ninjatrader: wire stop rounding needs a long/short side, got %q — refusing to guess which way is away from the entry", side)
	}
}

// WireBracket is THE rounding the wire applies to an entry's prices, shared by
// the sender (tcp_trader.go) and the gate that judges them (trader.EntryGate
// legs 5 and 6), so the R:R and min-SL floors are judged on exactly the numbers
// the broker receives: entry and target to the NEAREST tick, the stop AWAY
// from the entry (WireStop). An unknown side is an error (fail-closed).
func WireBracket(side string, entry, stop, target, tick float64) (wEntry, wStop, wTarget float64, err error) {
	wStop, err = WireStop(side, stop, tick)
	if err != nil {
		return 0, 0, 0, err
	}
	return RoundToTick(entry, tick), wStop, RoundToTick(target, tick), nil
}
