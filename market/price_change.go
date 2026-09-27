package market

import (
	"fmt"
	"time"
)

// W1 (presence-aware values) — a price change named by a duration measures
// that duration of wall time, on bar CLOSE TIMES, and is ABSENT when the
// series cannot measure it. Before W1 the futures "1h" was 20 bars of 5m
// (100 min), the "4h" was the previous long bar's close, the timeframes read
// counted bars (a 1h primary's "1h" was the previous close), and every one
// was a fabricated 0 when the bars ran short.

// ChangeOverWindow is the percent change from the close of the latest bar
// that closed at least `window` before the series' last bar, to that last
// close. nil — absent, never a fabricated 0 — when the series is empty, the
// window is not positive, no bar reaches back that far, or the reference
// close is not positive. Across a halt the reference is the last bar before
// the halt (wall-clock semantics: a feed hole reads the same way).
func ChangeOverWindow(bars []Kline, window time.Duration) *float64 {
	if len(bars) == 0 || window <= 0 {
		return nil
	}
	last := bars[len(bars)-1]
	target := last.CloseTime - window.Milliseconds()
	for i := len(bars) - 2; i >= 0; i-- {
		if bars[i].CloseTime > target {
			continue
		}
		if bars[i].Close <= 0 {
			return nil
		}
		pct := (last.Close - bars[i].Close) / bars[i].Close * 100
		return &pct
	}
	return nil
}

// changeWindowMinBars is the resolution a window needs: at least this many
// bars of the series' timeframe must fit inside it, so the reference bar sits
// within [window, window + one bar) of the last close. A coarser series would
// report "the previous bar's close" under the window's name.
const changeWindowMinBars = 4

// changeOnTimeframe measures `window` on a series of timeframe `tf`, or
// returns nil when the timeframe is unknown or too coarse for the window.
func changeOnTimeframe(bars []Kline, tf string, window time.Duration) *float64 {
	m := TFMinutes(tf)
	if m <= 0 || time.Duration(m*changeWindowMinBars)*time.Minute > window {
		return nil
	}
	return ChangeOverWindow(bars, window)
}

// PctOrNA renders a presence-aware percent: "n/a" when absent, otherwise the
// value with two decimals and a % sign (signed adds an explicit +).
func PctOrNA(p *float64, signed bool) string {
	if p == nil {
		return "n/a"
	}
	if signed {
		return fmt.Sprintf("%+.2f%%", *p)
	}
	return fmt.Sprintf("%.2f%%", *p)
}
