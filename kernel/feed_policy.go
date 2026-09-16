package kernel

import (
	"os"
	"strconv"

	"nofx/market"
)

// B2 (T6, fail-register wave 2026-08-20) — ONE feed-aliveness policy.
//
// The anatomy census found the "is the feed alive?" question answered on two
// different TFs with a 5× threshold gap: the B4 entry gate probed the
// snapshot's 1m-else-5m (tolerating up to ~10 min of silence on 5m, and NEVER
// assessing when the owner selected only {15m,1h}), while the in-position
// alert read live 1m at 120s. This object is now the single source both sides
// consume:
//
//	flat        → FEED_ALERT_S          (default 600s — the old 10-min const)
//	in-position → INTRADE_FEED_ALERT_S  (default 120s — existing env)
//
// The B4 per-period age contract STAYS primary for 1m/5m (the #48 fix); this
// policy adds the any-subscribed-TF fallback so no timeframe selection is
// blind, and centralizes the alert thresholds.

const (
	DefaultFeedAlertSeconds       = 600
	DefaultIntradeFeedAlertSecond = 120
)

type FeedPolicy struct {
	FlatAlertMs       int64
	InPositionAlertMs int64
}

func envSeconds(name string, def int64) int64 {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n * 1000
		}
	}
	return def * 1000
}

// LoadFeedPolicy resolves the per-state thresholds from env with the shipped
// defaults. Cheap — callers load per use.
func LoadFeedPolicy() FeedPolicy {
	return FeedPolicy{
		FlatAlertMs:       envSeconds("FEED_ALERT_S", DefaultFeedAlertSeconds),
		InPositionAlertMs: envSeconds("INTRADE_FEED_ALERT_S", DefaultIntradeFeedAlertSecond),
	}
}

// FreshestSubscribedAge returns the age of the newest bar across ALL subscribed
// timeframes in the snapshot (ok=false when no bars at all). The stamp is the
// bar's open (the NT8 open-stamp contract), so a forming bar reads near-zero.
func FreshestSubscribedAge(md *market.Data, nowMs int64) (ageMs int64, tf string, ok bool) {
	if md == nil || md.TimeframeData == nil {
		return 0, "", false
	}
	var newest int64
	for name, td := range md.TimeframeData {
		if td == nil || len(td.Klines) == 0 {
			continue
		}
		if b := td.Klines[len(td.Klines)-1].Time; b > newest {
			newest, tf = b, name
		}
	}
	if newest <= 0 {
		return 0, "", false
	}
	return nowMs - newest, tf, true
}
