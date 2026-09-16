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
//
// It is now a thin wrapper over RVBaselineFrom5mDays so the two can never
// disagree about a value (D2, BARS HORIZON 2026-09-09).
//
// A29, STATED PLAINLY: THIS HAS ZERO PRODUCTION CALL SITES. Production reads
// RVBaselineFrom5mDays through trader.ResolveRVBaselineTape. The wrapper is
// KEPT — deliberately, not by oversight — as the parity anchor for pin D2-E,
// which asserts the day-carrying function returns a byte-identical value to the
// pre-wave one. Delete the wrapper and that parity claim loses the only thing
// it can be checked against. Named here rather than counted as covered by the
// wave A29 sweep (corrected in review, 2026-09-09).
func RVBaselineFrom5m(min5 []market.Kline, maxDays, minDays int) (float64, bool) {
	v, _, ok := RVBaselineFrom5mDays(min5, maxDays, minDays)
	return v, ok
}

// RVBaselineFrom5mDays is RVBaselineFrom5m plus THE NUMBER OF COMPLETE
// SESSION-DAYS THE VALUE WAS ACTUALLY AVERAGED OVER.
//
// D2 (BARS HORIZON 2026-09-09): the estimator was always honest INTERNALLY —
// it drops incomplete days (:below) and returns ok=false under minDays — but
// nothing carried the real count out, so a value averaged over about 7 days
// filled a struct field named RVBaseline20d and rendered as
// "RV=103%-of-normal": a baseline with no stated window. THE COMPUTED VALUE IS
// UNCHANGED; only the count now travels with it.
//
// days is 0 only when ok is false — there is then no value to qualify, and a
// caller must not read that 0 as "zero days of a real baseline" (A24).
func RVBaselineFrom5mDays(min5 []market.Kline, maxDays, minDays int) (float64, int, bool) {
	if len(min5) < rvBaselineMinBarsPerDay || minDays <= 0 {
		return 0, 0, false
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
		return 0, 0, false
	}
	// average the last maxDays complete days.
	if maxDays > 0 && len(perDay) > maxDays {
		perDay = perDay[len(perDay)-maxDays:]
	}
	var sum float64
	for _, v := range perDay {
		sum += v
	}
	return sum / float64(len(perDay)), len(perDay), true
}
