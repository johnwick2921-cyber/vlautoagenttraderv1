package trader

import (
	"time"

	"nofx/market"
)

// HealthIsRunning reports whether this trader's loop is currently running
// (P2-6: /api/health's trader-running count). Read-only.
func (at *AutoTrader) HealthIsRunning() bool {
	if at == nil {
		return false
	}
	at.isRunningMutex.RLock()
	defer at.isRunningMutex.RUnlock()
	return at.isRunning
}

// HealthFeedStatus returns the NT8 price-feed status string from the most
// recent feed_status frame ("" until one arrives — the admission gate is
// default-ALLOW in that state, and the health payload says so, never a
// fabricated "up").
func (at *AutoTrader) HealthFeedStatus() string {
	if at == nil {
		return ""
	}
	nt := at.armedTrader()
	if nt == nil {
		return ""
	}
	return nt.FeedStatus()
}

// HealthLastBarAgeMs returns the age of the newest 1m bar from the shared
// futures bars provider (the same source the kernel reads), or ok=false when
// no bar is available.
func (at *AutoTrader) HealthLastBarAgeMs(now time.Time) (ageMs int64, ok bool) {
	if at == nil || market.FuturesBarsProvider == nil {
		return 0, false
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), "1m", 1)
	if len(bars) == 0 {
		return 0, false
	}
	return now.UnixMilli() - bars[len(bars)-1].OpenTime, true
}
