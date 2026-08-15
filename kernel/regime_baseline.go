package kernel

import (
	"time"

	"nofx/market"
)

// W10 — REALIZED-VOL BASELINE from stored 5m bars (the audit's dead wire: the
// realized-vol-vs-20d baseline was never supplied to ComputeRegime, so RV reported
// "warming" forever instead of "% of normal"). VIX legitimately has no feed and
// stays honest n/a — this only feeds the RV baseline, which IS computable from the
// bars we already have.
//
// The baseline uses the SAME estimator as the recent RV (recentDailyVolPct):
// bucket the 5m history by CME session-day, compute each COMPLETE day's realized
// vol, and average up to the last maxDays complete days. Self-consistent with the
// recent value (both 5m-intraday annualized), so "recent / baseline" is apples-to-
// apples. Fewer than minDays complete days → (0, false) = keep warming (honest).

// rvBaselineMinBarsPerDay is the near-complete-session bar count a day bucket needs
// to count toward the baseline (a full ~23h CME day is ~276 5m bars; the developing
// day and truncated history heads fall below this and are excluded).
const rvBaselineMinBarsPerDay = 200

// PriorCloseSessionOpen derives the overnight-gap inputs from daily bars (W11b):
// the prior completed session's close and the current session's open (the last two
// daily bars). Returns (0, 0) when fewer than two daily bars are present, so
// ComputeRegime's guard (both > 0) naturally skips the gap. Feeds
// RegimeInputs.PriorClose / SessionOpen → OvernightGapATR.
func PriorCloseSessionOpen(daily []market.Kline) (priorClose, sessionOpen float64) {
	if len(daily) < 2 {
		return 0, 0
	}
	return daily[len(daily)-2].Close, daily[len(daily)-1].Open
}

// RVBaselineFrom5m returns (baselineRVPct, ok). ok=false means warming (too few
// complete session-days). maxDays caps the lookback window; minDays is the minimum
// complete days required before a baseline is trustworthy.
func RVBaselineFrom5m(min5 []market.Kline, maxDays, minDays int) (float64, bool) {
	if len(min5) < rvBaselineMinBarsPerDay || minDays <= 0 {
		return 0, false
	}
	// bucket by CME session-day, preserving chronological order.
	type dayBucket struct {
		key  string
		bars []market.Kline
	}
	var buckets []dayBucket
	idx := map[string]int{}
	for _, b := range min5 {
		k := CMESessionDayKey(time.UnixMilli(b.OpenTime))
		if i, ok := idx[k]; ok {
			buckets[i].bars = append(buckets[i].bars, b)
			continue
		}
		idx[k] = len(buckets)
		buckets = append(buckets, dayBucket{key: k, bars: []market.Kline{b}})
	}

	// per-day RV for each COMPLETE day (partial/developing days excluded by count).
	var perDay []float64
	for _, d := range buckets {
		if len(d.bars) < rvBaselineMinBarsPerDay {
			continue
		}
		if rv, ok := recentDailyVolPct(d.bars); ok {
			perDay = append(perDay, rv)
		}
	}
	if len(perDay) < minDays {
		return 0, false
	}
	// average the last maxDays complete days.
	if maxDays > 0 && len(perDay) > maxDays {
		perDay = perDay[len(perDay)-maxDays:]
	}
	var sum float64
	for _, v := range perDay {
		sum += v
	}
	return sum / float64(len(perDay)), true
}
