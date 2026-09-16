// D4 (arms-follow-bias, 2026-09-04) — the far-arm counter. WARN-FIRST.
//
// 71% of arm-enabled scenarios (34/48) were authored at a price the tape never
// reached in that version's life; on 2026-09-02 six versions carried a short
// arm 62.25 points above the day's high. Nothing counted it, so nothing could
// separate an unlucky level from a habit.
//
// This wave only COUNTS. No arm is refused for being far: the threshold is
// [I] PROVISIONAL at 3.0×ATR5m and a week of counts decides whether it is a
// gate, a different number, or nothing at all.

package trader

import (
	"math"
	"os"
	"strconv"
	"strings"
)

// farArmFactor is the arm's distance from price measured in 5m ATRs.
//
// Returns 0 for an UNKNOWN input rather than a computed-looking number: a zero
// ATR would otherwise render as "right at the money", which is the opposite of
// what an unknown ATR means (A24 — no plausible zero).
func farArmFactor(entry, price, atr5m float64) float64 {
	if atr5m <= 0 || entry <= 0 || price <= 0 {
		return 0
	}
	return math.Abs(entry-price) / atr5m
}

// farArmThreshold resolves ARM_FAR_ATR_MULT (default 3.0). An unparseable or
// non-positive override falls back to the default rather than disabling the
// counter silently.
func farArmThreshold() float64 {
	if v := strings.TrimSpace(os.Getenv("ARM_FAR_ATR_MULT")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return 3.0
}

// armIsFar reports whether this arm should be COUNTED as far. An unknown ATR
// never flags: unknown is not far.
func armIsFar(entry, price, atr5m float64) bool {
	f := farArmFactor(entry, price, atr5m)
	return f > 0 && f > farArmThreshold()
}
