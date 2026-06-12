package market

import "time"

// ============================================================================
// Plan 3 Task 22 — Stale-data + drift detection
// ============================================================================
//
// This module provides freshness + drift checks for OHLCV input before the
// decision engine consumes it. The kernel calls these helpers; if the latest
// bar is stale (clock skew or upstream feed gap) or shows a suspicious limit-
// move-style jump in a short window, the engine should HOLD the cycle rather
// than feed degraded data to the AI.
//
// Defaults are picked for CME NQ/MNQ futures via Databento:
//   - RTH (regular trading hours, 08:30–15:00 CT): 90s tolerance. CME prints
//     1-minute bars; anything > 90s without a fresh bar implies a feed gap.
//   - ETH (overnight Globex): 5min tolerance. Lower-volume sessions can have
//     genuine quiet bars; we don't want to false-positive there.
//   - Drift 5% in <60s: CME index futures have a 7% limit-down circuit at the
//     primary band, and intraday vol is typically <1%; a >5% jump inside 60s
//     is almost certainly a bad print, a feed glitch, or an actual limit move
//     — none of which should be traded on without human review.

// FreshnessConfig holds tunable thresholds for the stale + drift gates.
type FreshnessConfig struct {
	// MaxAgeRTH is the maximum allowed age of the latest bar during regular
	// trading hours. Tighter than ETH because RTH prints are continuous.
	MaxAgeRTH time.Duration
	// MaxAgeETH is the maximum allowed age of the latest bar during extended
	// trading hours (overnight Globex). Looser to tolerate quiet sessions.
	MaxAgeETH time.Duration
	// DriftThresholdPct is the percent move between the previous and latest
	// bar's close that triggers a drift warning when paired with DriftWindow.
	DriftThresholdPct float64
	// DriftWindow is the time horizon within which a >DriftThresholdPct move
	// is considered suspicious. A 5% move in 10 min is normal; 5% in 30s is
	// not.
	DriftWindow time.Duration
}

// DefaultFreshnessConfig returns the production defaults for NQ/MNQ.
func DefaultFreshnessConfig() FreshnessConfig {
	return FreshnessConfig{
		MaxAgeRTH:         90 * time.Second,
		MaxAgeETH:         5 * time.Minute,
		DriftThresholdPct: 5.0,
		DriftWindow:       60 * time.Second,
	}
}

// DataHealth classifies the state of the latest bar relative to "now".
type DataHealth int

const (
	// HealthOK — data is fresh and free of suspicious drift; safe to trade.
	HealthOK DataHealth = iota
	// HealthStale — the latest bar is older than the configured max age
	// (or carries a future-dated timestamp; we treat that as stale too).
	HealthStale
	// HealthSuspiciousDrift — the latest close jumped > DriftThresholdPct
	// from the previous close within the DriftWindow. Likely bad print, feed
	// glitch, or limit move. Skip the cycle.
	HealthSuspiciousDrift
)

// IsFresh reports whether a kline whose OpenTime is `klineOpenTimeMs`
// milliseconds since epoch is no older than `maxAge` relative to `now`.
//
// Defensive behaviour: if the kline's timestamp is in the future (clock skew
// between Databento and the local host), we return false rather than trust
// it. A future-dated bar is just as bad as a stale one.
func IsFresh(klineOpenTimeMs int64, now time.Time, maxAge time.Duration) bool {
	openTime := time.UnixMilli(klineOpenTimeMs)
	if openTime.After(now) {
		// Clock skew: refuse to treat future-dated bars as fresh.
		return false
	}
	return now.Sub(openTime) <= maxAge
}

// IsDriftSuspicious reports whether `currentPrice` is suspiciously far from
// `prevPrice` given `elapsed` time between them. It triggers only when BOTH
// the percent move exceeds `thresholdPct` AND the elapsed time is below
// `window` — i.e. the jump is fast as well as large. A 10% move over an
// hour is just a market move; the same 10% in 30 seconds is a problem.
//
// prevPrice <= 0 returns false (cannot compute a percent move).
func IsDriftSuspicious(currentPrice, prevPrice float64, elapsed time.Duration, thresholdPct float64, window time.Duration) bool {
	if prevPrice <= 0 {
		return false
	}
	if elapsed >= window {
		return false
	}
	diff := currentPrice - prevPrice
	if diff < 0 {
		diff = -diff
	}
	pctMove := diff / prevPrice * 100
	return pctMove > thresholdPct
}

// CheckDataHealth combines the freshness + drift checks against the latest
// two bars and returns a DataHealth verdict. The caller passes the OpenTime
// (ms) + Close price for each of the last two bars; we use OpenTime rather
// than CloseTime because Databento bars don't always include CloseTime and
// the offset is well-defined by the bar's timeframe anyway.
//
// `now` lets the caller inject a clock for tests; production passes
// time.Now().
func CheckDataHealth(
	latestOpenTimeMs int64,
	latestClose float64,
	prevOpenTimeMs int64,
	prevClose float64,
	now time.Time,
	cfg FreshnessConfig,
) DataHealth {
	maxAge := cfg.MaxAgeETH
	if IsRTH(now) {
		maxAge = cfg.MaxAgeRTH
	}
	if !IsFresh(latestOpenTimeMs, now, maxAge) {
		return HealthStale
	}
	elapsed := time.UnixMilli(latestOpenTimeMs).Sub(time.UnixMilli(prevOpenTimeMs))
	if elapsed < 0 {
		elapsed = -elapsed
	}
	if IsDriftSuspicious(latestClose, prevClose, elapsed, cfg.DriftThresholdPct, cfg.DriftWindow) {
		return HealthSuspiciousDrift
	}
	return HealthOK
}

// IsRTH reports whether `t` falls inside CME index-futures regular trading
// hours: weekday, 08:30 <= local time of day < 15:00 in America/Chicago.
//
// This is a "within-the-trading-window" helper local to the freshness
// gate; the broader "is CME open at all" check lives in
// kernel.IsCMEOpen which also handles weekend + holidays + the daily
// maintenance break. The two are intentionally separate: IsCMEOpen returns
// true for both RTH and ETH; IsRTH discriminates between them so the
// freshness threshold can tighten when prints are continuous.
//
// On any time-zone load failure we conservatively return false (treat as
// ETH / looser tolerance) rather than crashing.
func IsRTH(t time.Time) bool {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return false
	}
	ct := t.In(loc)
	switch ct.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	}
	minutesOfDay := ct.Hour()*60 + ct.Minute()
	const rthOpen = 8*60 + 30  // 08:30 CT
	const rthClose = 15 * 60   // 15:00 CT
	return minutesOfDay >= rthOpen && minutesOfDay < rthClose
}
