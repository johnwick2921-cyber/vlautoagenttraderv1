package trader

import (
	"os"
	"strings"
	"time"

	"vl/store"
)

// stopLimitEntriesEnabled reads the mentor stop-limit knob (PR B, 2026-10-03).
// Default OFF — L4: additive and fail-closed. With the knob ON, an adjudicated
// stop-entry arm routes through PlaceStopEntryWithLimit: the AddOn builds
// OrderType.StopLimit (LimitPrice == StopPrice) instead of StopMarket, so the
// entry fills at its price or misses (D1.4 p1 @24:41, p2 @00:00); an unfilled
// stop-limit is cancelled at the candle close (N12). Go sets the frame flag
// only when the far side proves MinAddonBuildStopLimit.
func stopLimitEntriesEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MENTOR_STOP_LIMIT"))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// stopLimitStaleAtCandleClose is PURE (N12, PR B 2026-10-03): an unfilled
// stop-entry row is stale when the mentor knob is ON and the row's last
// lifecycle write predates the last closed 1m candle. Filled (working),
// terminal, cancel_pending (the settlement pass owns re-requests), and
// within-the-current-candle rows are never stale; the knob OFF returns false
// so the behaviour is byte-identical to today.
func stopLimitStaleAtCandleClose(r store.ArmedOrderDB, lastClose time.Time) bool {
	if !stopLimitEntriesEnabled() || r.Kind != "stop_entry" {
		return false
	}
	if r.State != store.StateArmed && r.State != store.StatePlacePending {
		return false
	}
	return r.UpdatedAt.Before(lastClose)
}
