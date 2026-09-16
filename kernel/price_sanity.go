package kernel

import (
	"fmt"
	"math"
)

// B2b — AI-output price sanity. These bounds catch model hallucinations / bad
// output whose stop/target/entry are physically implausible, BEFORE the decision
// is forwarded to execution. This is the ONE authority for the bounds below; it
// COMPOSES with — never duplicates — the wrong-side SL/TP authority that lives in
// validateDecision.

const (
	// atrSanityMultiple: a stop or target farther than this many ATRs from the
	// entry reference is treated as implausible.
	atrSanityMultiple = 8.0
	// entryImplausiblePct: an explicit entry farther than this fraction from the
	// last price is implausible for a market / near-touch order.
	entryImplausiblePct = 0.01
)

// priceSanityViolation reports whether an open decision's stop / target / entry
// are implausible. Bounds (side-agnostic, absolute distance):
//   - |entryRef - stopLoss|   > 8 × atr15   → bad
//   - |entryRef - takeProfit| > 8 × atr15   → bad
//   - |entryRef - lastPrice| / lastPrice > 1%  (entry implausibly far from last)
//
// atr15 <= 0 or lastPrice <= 0 → the check that needs it is skipped (fail-open;
// the caller logs the missing-market-data case). Returns a numbered reason and
// true on a violation. `side` is accepted for signature stability and future use;
// the wrong-side authority is validateDecision, not this function.
func priceSanityViolation(side string, entryRef, stopLoss, takeProfit, atr15, lastPrice float64) (string, bool) {
	_ = side // wrong-side is validateDecision's authority; not re-checked here.

	if lastPrice > 0 && entryRef > 0 {
		if dev := math.Abs(entryRef-lastPrice) / lastPrice; dev > entryImplausiblePct {
			return fmt.Sprintf("entry %.4f is %.2f%% from last %.4f (>%.0f%%)",
				entryRef, dev*100, lastPrice, entryImplausiblePct*100), true
		}
	}

	if atr15 > 0 && entryRef > 0 {
		maxDist := atrSanityMultiple * atr15
		if stopLoss > 0 {
			if d := math.Abs(entryRef - stopLoss); d > maxDist {
				return fmt.Sprintf("stop %.4f is %.4f from entry %.4f (>8×ATR=%.4f)",
					stopLoss, d, entryRef, maxDist), true
			}
		}
		if takeProfit > 0 {
			if d := math.Abs(entryRef - takeProfit); d > maxDist {
				return fmt.Sprintf("target %.4f is %.4f from entry %.4f (>8×ATR=%.4f)",
					takeProfit, d, entryRef, maxDist), true
			}
		}
	}
	return "", false
}

// LevelPriceViolation is the B2 armor for an OWNER-entered plan level price
// (P5.1): a level farther than 8 × the DAILY ATR from the last price is a
// fat-finger, rejected exactly as an implausible AI price would be. Level-scale
// (dATR), not execution-scale (atr15) — a legitimate PDL can sit far from price,
// but not 8× the whole day's range. Fail-open when dATR<=0 or lastPrice<=0 (no
// live market data — the caller notes it), mirroring priceSanityViolation.
func LevelPriceViolation(price, lastPrice, dATR float64) (string, bool) {
	if dATR <= 0 || lastPrice <= 0 || price <= 0 {
		return "", false // insufficient market data → fail-open (caller logs)
	}
	maxDist := atrSanityMultiple * dATR
	if d := math.Abs(price - lastPrice); d > maxDist {
		return fmt.Sprintf("level %.2f is %.2f from price %.2f (>8×dATR=%.2f)",
			price, d, lastPrice, maxDist), true
	}
	return "", false
}
