package kernel

import "nofx/market"

// TRADE EXCURSION LOGGING (wave 1A, 2026-09-02) — the intrabar path of a
// position from entry to exit, on the 1m tape.
//
// The pre-existing ComputeExcursion (mae_mfe.go) returns two floats and drops
// everything an exit rule needs: WHEN the extreme happened, how many bars in,
// how long the trade was held, and whether any single bar reached both the stop
// and the target — the case where the tape cannot say which came first. It also
// drops the ENTRY BAR: its filter is `b.OpenTime < entryMs → skip`, so unless a
// fill lands exactly on a bar boundary the bar containing it is excluded, and
// with it the first adverse move.
//
// This computes the whole path. It decides nothing and changes no exit.

// PathExcursion is one position's excursion, all of it derived from bars.
// Computed=false means the path could NOT be built (no bar coverage); its
// numbers are then meaningless and the caller stores NULLs, never zeros.
type PathExcursion struct {
	MAEPts        float64 // adverse extent from entry in points, >= 0
	MAETs         int64   // OpenTime of the bar holding the adverse extreme
	MAEBars       int     // bars after the entry bar (0 = the entry bar itself)
	MFEPts        float64 // favorable extent from entry in points, >= 0
	MFETs         int64
	MFEBars       int
	BarsHeld      int    // bars intersecting the hold, entry bar INCLUDED
	AmbiguousBars int    // bars reaching BOTH the stop and the target
	Resolution    string // "1m" | "5m" | "none" — which tape this came from
	Computed      bool
}

// tfDurationMs maps a resolution label to its bar length. An unknown label
// returns 0 and the caller refuses to compute rather than assuming a minute.
func tfDurationMs(resolution string) int64 {
	switch resolution {
	case "1m":
		return 60_000
	case "5m":
		return 300_000
	}
	return 0
}

// ComputePathExcursion walks the bars covering [entryMs, exitMs]. A bar counts
// when its own [open, close) window INTERSECTS the hold, so the bar containing
// the fill is included — the bar the old computation dropped.
//
// stopPx/targetPx are read ONLY to count ambiguous bars. Pass 0 for either and
// no bar is counted ambiguous: an unknown level cannot be shown to be spanned.
func ComputePathExcursion(entryPx float64, side string, stopPx, targetPx float64, bars []market.Kline, entryMs, exitMs int64, resolution string) PathExcursion {
	out := PathExcursion{Resolution: "none"}
	dur := tfDurationMs(resolution)
	if entryPx <= 0 || dur == 0 || entryMs <= 0 || exitMs < entryMs || len(bars) == 0 {
		return out
	}
	long := isLongSide(side)
	lo, hi := stopPx, targetPx
	if lo > hi {
		lo, hi = hi, lo
	}
	spanKnown := stopPx > 0 && targetPx > 0

	entryBarOpen := int64(-1)
	for i := range bars {
		b := bars[i]
		barOpen, barClose := b.OpenTime, b.OpenTime+dur
		if barClose <= entryMs || barOpen > exitMs {
			continue // wholly before the fill, or wholly after the exit
		}
		if entryBarOpen < 0 {
			entryBarOpen = barOpen
		}
		out.BarsHeld++
		idx := int((barOpen - entryBarOpen) / dur)

		var fav, adv float64
		if long {
			fav, adv = b.High-entryPx, entryPx-b.Low
		} else {
			fav, adv = entryPx-b.Low, b.High-entryPx
		}
		if fav > out.MFEPts {
			out.MFEPts, out.MFETs, out.MFEBars = fav, barOpen, idx
		}
		if adv > out.MAEPts {
			out.MAEPts, out.MAETs, out.MAEBars = adv, barOpen, idx
		}
		// C4 — one bar reaching BOTH levels. A bar carries no ordering, so this
		// is counted and surfaced, never resolved by assumption.
		if spanKnown && b.Low <= lo && b.High >= hi {
			out.AmbiguousBars++
		}
	}
	if out.BarsHeld == 0 {
		return out // no coverage: Computed stays false, resolution "none"
	}
	out.Resolution, out.Computed = resolution, true
	return out
}

// ResolveAmbiguousExit reads a bar that may have reached both the stop and the
// target. C4: a bar carries no ordering, so when both are inside its range the
// record resolves AGAINST the trade — the stop is taken as the exit — and the
// row is flagged so the ambiguity is visible in the distribution rather than
// dissolved into a favourable number.
//
// This changes NO exit. The position closed where it closed; this is what the
// excursion record says happened, and it is deliberately the pessimistic read.
func ResolveAmbiguousExit(side string, stopPx, targetPx float64, bar market.Kline) (exitPx float64, ambiguous bool) {
	if stopPx <= 0 || targetPx <= 0 {
		return 0, false
	}
	lo, hi := stopPx, targetPx
	if lo > hi {
		lo, hi = hi, lo
	}
	if bar.Low <= lo && bar.High >= hi {
		return stopPx, true
	}
	return 0, false
}
