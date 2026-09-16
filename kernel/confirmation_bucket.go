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
