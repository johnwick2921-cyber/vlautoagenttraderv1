// D4 (arms-follow-bias, 2026-09-04) — far-arm counts, per side.
//
// WARN-first: nothing is refused for being far. The counter exists so a week of
// evidence can decide whether 3.0×ATR5m is a gate, a different number, or
// nothing. Per SIDE because the 09-02 evidence was one-sided: six versions
// carried a SHORT arm 62.25 points above the day's high.
//
// Counters RECORD (class 35). Reading them never resets them.

package telemetry

import "sync/atomic"

var (
	farArmsLong  atomic.Int64
	farArmsShort atomic.Int64
	armsAuthored atomic.Int64
)

// IncFarArm counts one arm authored beyond the far threshold. side is
// "long"|"short"; anything else is counted as authored but not attributed,
// which is preferable to guessing a side.
func IncFarArm(side string) {
	switch side {
	case "long":
		farArmsLong.Add(1)
	case "short":
		farArmsShort.Add(1)
	}
}

// IncArmAuthored counts every arm authored, so a far-rate has a denominator
// (A24 — never a rate without n).
func IncArmAuthored() { armsAuthored.Add(1) }

// FarArmCounts returns long, short and the authored denominator.
func FarArmCounts() (long, short, authored int64) {
	return farArmsLong.Load(), farArmsShort.Load(), armsAuthored.Load()
}
