package kernel

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nofx/market"
)

// 1m-MSS (E5, entry-mechanics wave 2026-08-30) — the new confirm primitive.
//
// Semantics: the LAST CONFIRMED 1m fractal swing (k=2, strictly higher/lower
// than its 2 neighbors each side) broken by a 1m CLOSE beyond it, with
// displacement ≥ MSS_MIN_DISP_ATR × ATR5m on the breaking bar.
// Closed bars only — a wick beyond the swing NEVER counts (close-only, like
// every acceptance rule). Direction-aware: side "above" hunts the last swing
// HIGH (a bullish MSS break); side "below" hunts the last swing LOW (bearish).

// mssMinDispATR resolves MSS_MIN_DISP_ATR (default 0.5 × ATR5m).
func mssMinDispATR() float64 {
	if v := strings.TrimSpace(os.Getenv("MSS_MIN_DISP_ATR")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			return n
		}
	}
	return 0.5
}

// MSSMinDispATR is the exported knob resolver (E9 guide card cites this).
func MSSMinDispATR() float64 { return mssMinDispATR() }

// MSSVerdict is the machine-computed 1m-MSS state for one confirm object.
type MSSVerdict struct {
	Met         bool    // a qualifying 1m close broke the swing
	SwingPrice  float64 // the confirmed swing extreme (the line broken)
	SwingHigh   bool    // true = the line was a swing HIGH (bullish break)
	SwingTimeMs int64   // the swing bar's close instant (repo convention)
	BreakClose  float64 // the breaking 1m close
	BreakTimeMs int64   // the breaking bar's open time
	DispPts     float64 // |BreakClose − SwingPrice| on the breaking bar
	Detail      string
}

// lastFractalSwing finds the LAST confirmed fractal swing of the wanted type
// over CLOSED bars (k=2: strictly extreme vs its 2 neighbors each side).
// Returns (price, timeMs, found).
func lastFractalSwing(bars []market.Kline, wantHigh bool, nowMs int64) (float64, int64, bool) {
	const k = 2
	closed := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		if b.CloseTime < nowMs {
			closed = append(closed, b)
		}
	}
	n := len(closed)
	if n < 2*k+1 {
		return 0, 0, false
	}
	for i := n - 1 - k; i >= k; i-- {
		extreme := true
		if wantHigh {
			for j := i - k; j <= i+k; j++ {
				if j == i {
					continue
				}
				if closed[j].High >= closed[i].High {
					extreme = false
					break
				}
			}
			if extreme {
				return closed[i].High, closed[i].OpenTime + 60_000, true
			}
		} else {
			for j := i - k; j <= i+k; j++ {
				if j == i {
					continue
				}
				if closed[j].Low <= closed[i].Low {
					extreme = false
					break
				}
			}
			if extreme {
				return closed[i].Low, closed[i].OpenTime + 60_000, true
			}
		}
	}
	return 0, 0, false
}

// EvaluateMSS computes the 1m-MSS verdict over the bars since plan birth
// (the windowed series EvaluateConfirm hands it — BarsSince output).
// side: "above" (bullish break of a swing HIGH) | "below" (bearish).
// ref: the authored ref_price — prose-anchored, but the MACHINE swing is the
// line judged (the swing is what exists on the tape at runtime; the ref may
// be the plan-write snapshot of it).
func EvaluateMSS(bars []market.Kline, side string, nowMs int64) MSSVerdict {
	return evaluateMSSAfter(bars, side, nowMs, nil)
}

func evaluateMSSAfter(bars []market.Kline, side string, nowMs int64, after *int64) MSSVerdict {
	v := MSSVerdict{}
	wantHigh := strings.EqualFold(side, "above")
	p, t, ok := lastFractalSwing(bars, wantHigh, nowMs)
	if !ok {
		v.Detail = "no confirmed 1m swing yet (needs 5 closed bars)"
		return v
	}
	v.SwingPrice, v.SwingTimeMs, v.SwingHigh = p, t, wantHigh
	atr5m := StaleConfirmATR5m(bars)
	need := mssMinDispATR() * atr5m
	for _, b := range bars {
		event := EvaluateBucketClose(b.OpenTime, 1, nowMs)
		if !event.Closed || (after != nil && confirmationOrderedSequence && event.CloseMs <= *after) || b.OpenTime < t-60_000 {
			continue // closed bars AFTER the swing only (the swing itself can't break itself)
		}
		beyond := (wantHigh && b.Close > p) || (!wantHigh && b.Close < p)
		if !beyond {
			continue
		}
		disp := b.Close - p
		if !wantHigh {
			disp = p - b.Close
		}
		if atr5m > 0 && disp < need {
			continue // displacement too weak — not a true MSS
		}
		v.Met = true
		v.BreakClose = b.Close
		v.BreakTimeMs = b.OpenTime
		v.DispPts = disp
		break
	}
	if v.Met {
		v.Detail = fmt.Sprintf("swing %.2f @%s broken by 1m close %.2f (disp %.2f pts ≥ %.1f×ATR5m)",
			p, FormatCT(time.UnixMilli(v.SwingTimeMs)), v.BreakClose, v.DispPts, mssMinDispATR())
		return v
	}
	v.Detail = fmt.Sprintf("swing %.2f @%s not yet broken by a qualifying 1m close (need ≥ %.1f×ATR5m)", p, FormatCT(time.UnixMilli(v.SwingTimeMs)), mssMinDispATR())
	return v
}
