package trader

import (
	"os"
	"strings"

	"vl/store"
)

// stopLimitEntriesEnabled reads the mentor stop-limit knob (PR B, 2026-10-03).
// Default OFF — L4: additive and fail-closed. With the knob ON, an adjudicated
// stop-entry arm routes through PlaceStopEntryWithLimit: the AddOn builds
// OrderType.StopLimit (LimitPrice == StopPrice) instead of StopMarket, so the
// entry fills at its price or misses (D1.4 p1 @24:41, p2 @00:00); the order's
// expiry (expiry_ms) is authored by the evaluator's intent (DS-102) and the
// armed pass cancels it when it lapses unfilled (N12). Go sets the frame flag
// only when the far side proves MinAddonBuildStopLimit.
func stopLimitEntriesEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MENTOR_STOP_LIMIT"))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// armExpired is PURE (N12, PR B 2026-10-03): an order with a stored expiry is
// due when now >= expiry_ms and the row is unfilled (armed or place_pending).
// Filled (working), terminal, cancel_pending (the settlement pass owns
// re-requests) and expiry-less rows are never due — 0 means no expiry was ever
// authored, and this code never sweeps what no intent expired.
func armExpired(r store.ArmedOrderDB, nowMs int64) bool {
	if r.ExpiryMs <= 0 {
		return false
	}
	if r.State != store.StateArmed && r.State != store.StatePlacePending {
		return false
	}
	return nowMs >= r.ExpiryMs
}
