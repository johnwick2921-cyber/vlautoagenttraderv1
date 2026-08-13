// Package discipline holds small, dependency-free trade-discipline primitives
// shared across the broker layer (which detects exits) and the kernel (which gates
// entries). It imports nothing from the rest of nofx, so both layers can depend on
// it without an import cycle.
package discipline

import (
	"fmt"
	"math"
	"sync"
)

// B7 — RE-ENTRY COOLDOWN.
//
// After a STOP-LOSS exit, a same-direction re-entry on that symbol is blocked until
// EITHER a cooldown elapses OR price has moved at least 1×ATR15 away from the stop
// — whichever happens first. This stops the classic revenge/chop re-entry right
// back into the level that just stopped the trade out, without permanently locking
// the symbol out. State is per (trader, symbol, side) and in-memory (a discipline
// timer, not a ledger).
//
// The broker close path calls NoteStopLossExit when a position closes with the NT8
// exit reason "sl"; the kernel entry gate calls ReentryBlocked before admitting an
// open_long/open_short. side is normalized to "long"/"short".

type reentryKey struct {
	trader string
	symbol string
	side   string
}

type reentryRecord struct {
	stopPrice float64
	atStopMs  int64
}

var (
	reentryMu  sync.Mutex
	reentryRec = map[reentryKey]reentryRecord{}
)

func normSide(side string) string {
	switch side {
	case "long", "LONG", "Long", "buy", "BUY":
		return "long"
	case "short", "SHORT", "Short", "sell", "SELL":
		return "short"
	default:
		return side
	}
}

// NoteStopLossExit arms the re-entry cooldown for (trader, symbol, side): a
// same-direction re-entry on this symbol is now subject to the cooldown gate,
// measured from stopPrice at atMs. A later stop-loss on the same key overwrites the
// prior one (the most recent stop is the relevant level).
func NoteStopLossExit(trader, symbol, side string, stopPrice float64, atMs int64) {
	reentryMu.Lock()
	reentryRec[reentryKey{trader, symbol, normSide(side)}] = reentryRecord{stopPrice: stopPrice, atStopMs: atMs}
	reentryMu.Unlock()
}

// ReentryBlocked reports whether a same-direction re-entry on (trader, symbol,
// side) is currently blocked, and why. It is blocked while BOTH hold:
//
//   - still within cooldownMinutes of the stop-loss exit, AND
//   - price has NOT moved ≥ atr15 (absolute distance) away from the stop.
//
// Unlock happens on the EARLIER of the two (cooldown elapsed OR price moved a full
// ATR15 from the stop); on unlock the record is cleared so it never blocks again.
// cooldownMinutes ≤ 0 disables the gate (returns not-blocked). If atr15 ≤ 0 the
// price-move unlock cannot be evaluated, so the timer is the only unlock path
// (fail-safe: rely on the cooldown, never crash). nowMs is the current time in ms.
func ReentryBlocked(trader, symbol, side string, cooldownMinutes int, atr15, currentPrice float64, nowMs int64) (remainingSec int, reason string, blocked bool) {
	if cooldownMinutes <= 0 {
		return 0, "", false
	}
	key := reentryKey{trader, symbol, normSide(side)}

	reentryMu.Lock()
	rec, ok := reentryRec[key]
	reentryMu.Unlock()
	if !ok {
		return 0, "", false
	}

	// Price-move unlock: a full ATR15 away from the stop → conditions changed enough.
	if atr15 > 0 && math.Abs(currentPrice-rec.stopPrice) >= atr15 {
		clearReentry(key)
		return 0, "", false
	}

	// Time unlock: cooldown elapsed.
	elapsedMs := nowMs - rec.atStopMs
	cooldownMs := int64(cooldownMinutes) * 60_000
	if elapsedMs >= cooldownMs {
		clearReentry(key)
		return 0, "", false
	}

	remainingSec = int((cooldownMs - elapsedMs) / 1000)
	moved := math.Abs(currentPrice - rec.stopPrice)
	reason = fmt.Sprintf("stop-loss %s exit at %.4f — same-direction re-entry blocked for %dm (price %.4f is %.4f from stop, need ≥%.4f=1×ATR15)",
		normSide(side), rec.stopPrice, cooldownMinutes, currentPrice, moved, atr15)
	return remainingSec, reason, true
}

func clearReentry(key reentryKey) {
	reentryMu.Lock()
	delete(reentryRec, key)
	reentryMu.Unlock()
}

// ResetReentryForTest clears all state (test-only).
func ResetReentryForTest() {
	reentryMu.Lock()
	reentryRec = map[reentryKey]reentryRecord{}
	reentryMu.Unlock()
}
