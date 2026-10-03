package mentor

import "vl/market"

// emaCross30 counts how many times the close CROSSED the EMA line over the
// last 30 closed 1m candles (sign changes of close - ema between consecutive
// bars). E4: "xien len xien xuong" [D4.2 p1 @ 22:27] — crossing back and forth
// at the line is noise; the mentor never gives a number, so the knob gates.
func emaCross30(bars []market.Kline, ema float64) int {
	if len(bars) < 2 {
		return 0
	}
	n := len(bars)
	start := n - 30
	if start < 1 {
		start = 1
	}
	cross := 0
	prev := bars[start-1].Close - ema
	for _, b := range bars[start:] {
		cur := b.Close - ema
		if (prev < 0 && cur >= 0) || (prev > 0 && cur <= 0) || (prev == 0 && cur != 0) {
			cross++
		}
		prev = cur
	}
	return cross
}

// emaSetupAllowed — the EMA34 setup seam (E2 + E4, CTO 12:27:25Z, moved to
// DS-103 12:33:43Z). Only the EMA34 line is gated here; key levels keep their
// own rules.
func emaSetupAllowed(e *Evaluator, lvl Level, bars []market.Kline, cfg Config) bool {
	if lvl.Kind != KindEMA34 {
		return true
	}
	// E2: one loss at the line blocks the EMA until a departure.
	if e.State.EmaBlocked {
		return false
	}
	// E4: crossing the line too many times in 30m is noise.
	if cfg.EmaMaxCross30m > 0 && emaCross30(bars, lvl.Price) >= cfg.EmaMaxCross30m {
		return false
	}
	return true
}

// emaLossTick — E2: after an EMA34 stop entry is emitted, the pending stop is
// watched on closed candles. Stop-out BEFORE the fill = one loss at the line →
// the EMA is blocked for new setups until a closed candle whose range does NOT
// touch the line (the departure) — the same rule as the level lost-box.
func emaLossTick(e *Evaluator, emaPrice float64, cur market.Kline) {
	if e.State.EmaPendingSide != "" {
		// A STOP entry fills when price REACHES the entry (high for longs, low
		// for shorts); it is stopped out when the stop trades without that.
		filled := e.State.EmaPendingSide == SideLong && cur.High >= e.State.EmaPendingEntry ||
			e.State.EmaPendingSide == SideShort && cur.Low <= e.State.EmaPendingEntry
		stopped := e.State.EmaPendingSide == SideLong && cur.Low <= e.State.EmaPendingStop ||
			e.State.EmaPendingSide == SideShort && cur.High >= e.State.EmaPendingStop
		if filled {
			e.State.EmaPendingSide = "" // filled first: the trade is live, not a loss
		} else if stopped {
			e.State.EmaBlocked = true // one loss at the line
			e.State.EmaPendingSide = ""
		}
	}
	if e.State.EmaBlocked && (cur.High < emaPrice || cur.Low > emaPrice) {
		e.State.EmaBlocked = false // departure: the line is left alone
	}
}

func isEMA34(lvl Level) bool {
	return lvl.Kind == KindEMA34
}
