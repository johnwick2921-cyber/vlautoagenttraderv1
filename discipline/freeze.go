package discipline

import "sync"

// A2/A4 — per-trader trading FREEZE. When the identity echo from the AddOn does not
// match the pending op (a forged / cross-wired / corrupted fill or close), or when
// the reconcile belief diverges from the broker beyond tolerance, the owning trader
// is FROZEN: NEW entries are blocked, but open-position management (exits, stop
// moves, reconcile) continues. An owner clears it via the API. In-memory — a safety
// latch, not a ledger.

type freezeInfo struct {
	reason  string
	sinceMs int64
}

var (
	freezeMu sync.Mutex
	frozen   = map[string]freezeInfo{}
)

// FreezeTrader latches a freeze on traderID with a reason. Returns true if this was
// a NEW freeze (so the caller logs the 🚨 once), false if already frozen. A blank
// traderID is a no-op.
func FreezeTrader(traderID, reason string, atMs int64) bool {
	if traderID == "" {
		return false
	}
	freezeMu.Lock()
	defer freezeMu.Unlock()
	if _, ok := frozen[traderID]; ok {
		return false
	}
	frozen[traderID] = freezeInfo{reason: reason, sinceMs: atMs}
	return true
}

// IsFrozen reports whether traderID is frozen and why.
func IsFrozen(traderID string) (reason string, isFrozen bool) {
	freezeMu.Lock()
	defer freezeMu.Unlock()
	fi, ok := frozen[traderID]
	return fi.reason, ok
}

// ClearFreeze removes a freeze (owner action). Returns true if the trader was frozen.
func ClearFreeze(traderID string) bool {
	freezeMu.Lock()
	defer freezeMu.Unlock()
	if _, ok := frozen[traderID]; !ok {
		return false
	}
	delete(frozen, traderID)
	return true
}

// FrozenSnapshot returns a copy of the trader→reason table (for the API).
func FrozenSnapshot() map[string]string {
	freezeMu.Lock()
	defer freezeMu.Unlock()
	out := make(map[string]string, len(frozen))
	for id, fi := range frozen {
		out[id] = fi.reason
	}
	return out
}

// ResetFreezeForTest clears all freezes (test-only).
func ResetFreezeForTest() {
	freezeMu.Lock()
	frozen = map[string]freezeInfo{}
	freezeMu.Unlock()
}
