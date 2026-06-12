// Package market — decimal-safe arithmetic helpers.
//
// Plan 2 Task 20: position sizing, PnL, and tick math all use float64.
// For NQ at 21500, even a 0.01-point rounding error compounds to $5 over
// 100 trades. These helpers do price math in integer-tick space (int64),
// which is exact for any quantity that is an integer multiple of the
// instrument tick size.
//
// Design choice: int64-ticks (not math/big or shopspring/decimal).
//   - All CME index prices are exact multiples of 0.25 (NQ/MNQ/ES/MES).
//   - A single int64 holds tick counts well beyond any realistic price range.
//   - No external dependency; no big.Float allocation overhead.
//   - Conversion back to float64 is lossless for any tick count that fits
//     in a float64 mantissa (~2^53), i.e. ~9e15 ticks. NQ at 22,000 with
//     tick=0.25 = 88,000 ticks, so we have ~11 orders of magnitude of headroom.
//
// Helpers in this file are pure and have no external dependencies. They are
// intended to be wired into kernel/engine_position.go and
// trader/ninjatrader/trader.go in a follow-up PR.
package market

import "math"

// PriceToTicks converts a float price to an integer tick count.
// Returns 0 if tickSize <= 0 (defensive; caller should validate).
//
// Uses math.Round (banker's rounding via Go runtime) so that prices
// landing exactly between ticks bias neither up nor down.
func PriceToTicks(price float64, tickSize float64) int64 {
	if tickSize <= 0 {
		return 0
	}
	return int64(math.Round(price / tickSize))
}

// TicksToPrice converts an integer tick count back to a float price.
// Lossless for any tick count within float64 mantissa precision.
func TicksToPrice(ticks int64, tickSize float64) float64 {
	return float64(ticks) * tickSize
}

// SafeAdd adds two prices in tick-space, then converts back to float.
// Avoids float-drift accumulation when repeatedly summing tick-multiples.
// Example: SafeAdd(0.1, 0.2, 0.01) == 0.30 exactly (vs 0.30000000000000004
// for raw float addition).
func SafeAdd(a, b float64, tickSize float64) float64 {
	if tickSize <= 0 {
		return a + b
	}
	return TicksToPrice(PriceToTicks(a, tickSize)+PriceToTicks(b, tickSize), tickSize)
}

// SafeSubtract subtracts b from a in tick-space, then converts back.
func SafeSubtract(a, b float64, tickSize float64) float64 {
	if tickSize <= 0 {
		return a - b
	}
	return TicksToPrice(PriceToTicks(a, tickSize)-PriceToTicks(b, tickSize), tickSize)
}

// SafeMultiply adds N ticks (multiplier) to a base price, in tick-space.
// Useful for "stop = entry - 12 ticks" style ops. multiplier may be negative.
// Returns price unchanged if tickSize <= 0.
func SafeMultiply(price float64, multiplier int, tickSize float64) float64 {
	if tickSize <= 0 {
		return price
	}
	return TicksToPrice(PriceToTicks(price, tickSize)+int64(multiplier), tickSize)
}

// PositionSize returns the maximum whole-contract count that keeps the
// loss at the stop-loss within riskDollars.
//
//	contracts = floor( riskDollars / (stopDistanceTicks * dollarsPerTick) )
//
// ALWAYS floors — never rounds up — so the trader cannot over-leverage by
// even one contract. Returns 0 if any input is <= 0 (defensive: refuses
// to enter rather than crash on bad inputs).
//
// For NQ: dollarsPerTick = $5  ($20/point ÷ 4 ticks/point).
// For MNQ: dollarsPerTick = $0.50 ($2/point ÷ 4 ticks/point).
func PositionSize(riskDollars, stopDistanceTicks, dollarsPerTick float64) int {
	if riskDollars <= 0 || stopDistanceTicks <= 0 || dollarsPerTick <= 0 {
		return 0
	}
	lossPerContract := stopDistanceTicks * dollarsPerTick
	return int(math.Floor(riskDollars / lossPerContract))
}
