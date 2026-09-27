package kernel

import (
	"fmt"
	"nofx/market"
	"time"
)

// These policies are consumed by the evaluator and the boot receipt.
const confirmationClosedBuckets = true
const confirmationOrderedSequence = true
const confirmationUnknown = "UNKNOWN"
const immediateDisplacementRule = "1m_displacement"

// BucketClose records the actual end boundary, not the last millisecond inside
// the bucket. Every confirmation/void/displacement reader uses this predicate.
type BucketClose struct {
	OpenMs  int64 `json:"open_ms"`
	CloseMs int64 `json:"close_ms"`
	Minutes int   `json:"minutes"`
	Closed  bool  `json:"closed"`
}

// EvaluateBucketClose answers one question for every caller. It neither counts
// source bars nor reads a wall clock. An unknown/nonpositive interval is closed=false.
func EvaluateBucketClose(openMs int64, minutes int, nowMs int64) BucketClose {
	b := BucketClose{OpenMs: openMs, Minutes: minutes}
	if minutes <= 0 {
		return b
	}
	span := int64(minutes) * 60_000
	b.OpenMs = openMs - openMs%span
	b.CloseMs = b.OpenMs + span
	b.Closed = nowMs >= b.CloseMs
	return b
}

func (b BucketClose) String() string {
	return fmt.Sprintf("%dm bucket %s–%s · close %s · closed=%t", b.Minutes,
		FormatCT(time.UnixMilli(b.OpenMs)), FormatCT(time.UnixMilli(b.CloseMs)), FormatCT(time.UnixMilli(b.CloseMs)), b.Closed)
}

// confirmationTape (W2 (d), 2026-09-23) is THE canonical confirmation input:
// the bars whose OPEN lies in [sinceMs, nowMs), in strictly increasing OpenTime,
// one bar per minute. Every confirmation count (evaluateConfirmAfter and
// confirmationBuckets) reads it, so no input order can mint a count:
//   - a bar with the SAME OpenTime as the last kept bar REPLACES it (the later
//     copy is the fresher print of that minute — the live cache's own upsert
//     rule, provider/ninjatrader/bar_cache.go Upsert) and is counted;
//   - a bar OLDER than the last kept bar is DROPPED and counted — a late copy
//     of an earlier minute can no longer re-open an earlier bucket as a phantom
//     extra close, or add a second vote for a minute already counted;
//   - a pre-birth bar anywhere in the slice stays out (BarsSince assumed sorted
//     input and kept every bar after the first in-window one).
//
// The forming-bar guard is unchanged: the window's upper edge is the bar's OPEN,
// and EvaluateBucketClose alone decides whether its bucket has closed.
// Plan death / flip (plan_lifecycle.go) keep their own windowing — untouched.
func confirmationTape(bars []market.Kline, sinceMs, nowMs int64) []market.Kline {
	out := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		if b.OpenTime < sinceMs || b.OpenTime >= nowMs {
			continue
		}
		if n := len(out); n > 0 {
			last := out[n-1].OpenTime
			if b.OpenTime == last {
				out[n-1] = b
				confirmationTapeReplaced.Add(1)
				continue
			}
			if b.OpenTime < last {
				recordConfirmationTapeDrop(b.OpenTime, last)
				continue
			}
		}
		out = append(out, b)
	}
	return out
}

func closedConfirmationBuckets(bars []market.Kline, sinceMs, nowMs int64, minutes int) ([]market.Kline, *BucketClose) {
	var closed []market.Kline
	var last *BucketClose
	for _, b := range confirmationBuckets(bars, sinceMs, nowMs, minutes) {
		e := EvaluateBucketClose(b.OpenTime, minutes, nowMs)
		last = &e
		if confirmationClosedBuckets && !e.Closed {
			continue
		}
		closed = append(closed, b)
	}
	return closed, last
}

// MinuteDisplacement is an observation, never a 5m confirmation or void verdict.
type MinuteDisplacement struct {
	Rule     string       `json:"rule"`
	Pts      float64      `json:"pts"`
	Observed bool         `json:"observed"`
	Bucket   *BucketClose `json:"bucket,omitempty"`
}

// Evaluate1mDisplacement preserves immediate-mode pre-confirmation authoring:
// excursion on completed 1m bars closing beyond the level, without claiming a
// completed 5m break. Reclaim/void is still judged separately on closed 5m bars.
func Evaluate1mDisplacement(bars []market.Kline, level float64, short bool, sinceMs, nowMs int64) MinuteDisplacement {
	v := MinuteDisplacement{Rule: immediateDisplacementRule}
	for _, b := range confirmationBuckets(bars, sinceMs, nowMs, 1) {
		e := EvaluateBucketClose(b.OpenTime, 1, nowMs)
		if !e.Closed {
			continue
		}
		beyond := (short && b.Close < level) || (!short && b.Close > level)
		if !beyond {
			continue
		}
		pts := b.High - level
		if short {
			pts = level - b.Low
		}
		if !v.Observed || pts > v.Pts {
			v.Pts, v.Observed, v.Bucket = pts, true, &e
		}
	}
	return v
}
