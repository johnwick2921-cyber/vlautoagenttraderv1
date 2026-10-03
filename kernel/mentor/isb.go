package mentor

import (
	"vl/market"
)

// ISB — inside bar [§2.1]. Identification is BY THE BODY: "cây nến ISB em
// không tính râu, em chỉ tính phần thân" [D2.1 p1 @ 04:18]. Orders are placed
// at the WICK-INCLUSIVE extremes [D2.1 p1 @ 04:39] with the 1.5-pt buffer
// [D2.1 p1 @ 05:36].

// IsISB reports whether cur's BODY sits inside prev's BODY.
func IsISB(prev, cur market.Kline) bool {
	return curHigh(cur) <= curHigh(prev) && curLow(cur) >= curLow(prev)
}

func curHigh(b market.Kline) float64 {
	if b.Close > b.Open {
		return b.Close
	}
	return b.Open
}

func curLow(b market.Kline) float64 {
	if b.Close < b.Open {
		return b.Close
	}
	return b.Open
}

// ISBOrders computes both stop-entry orders for an inside-bar candle
// [D3.4 p1 @ 23:04 "for a short, sell stop BELOW the inside bar, stop loss
// ABOVE it. Mirror for longs"], buffered by ISBBufferPts:
//
//	long:  buy stop  = High + buffer, stop loss = Low − buffer
//	short: sell stop = Low − buffer,  stop loss = High + buffer
//
// Risk = (High − Low) + 2×buffer.
func ISBOrders(candle market.Kline, cfg Config) (long, short Intent) {
	long = Intent{
		Action: PlaceStopEntry,
		Side:   SideLong,
		Price:  candle.High + cfg.ISBBufferPts,
		Stop:   candle.Low - cfg.ISBBufferPts,
		Reason: "ISB: buy stop above the wick high + buffer, stop below the wick low [D2.1 p1 @ 04:39, 05:36; D3.4 p1 @ 23:04]",
	}
	short = Intent{
		Action: PlaceStopEntry,
		Side:   SideShort,
		Price:  candle.Low - cfg.ISBBufferPts,
		Stop:   candle.High + cfg.ISBBufferPts,
		Reason: "ISB: sell stop below the wick low − buffer, stop above the wick high [D2.1 p1 @ 04:39, 05:36; D3.4 p1 @ 23:04]",
	}
	return long, short
}

// ISBStopVerdict applies the stop-size rules to an inside-bar order:
// an ISB stop is normally 5–6 points [D3.2 p1 @ 14:18]; one whose stop is in
// the twenties must NOT be taken [D4.1 p1 @ 05:41]; and the 25-pt ceiling is
// the global hard stop [D3.3 p1 @ 02:04] (twenties already covers it).
func ISBStopVerdict(candle market.Kline, cfg Config) (stopPts float64, ok bool, reason string) {
	stopPts = (candle.High - candle.Low) + 2*cfg.ISBBufferPts
	switch {
	case stopPts < cfg.ISBStopMinPts:
		return stopPts, false, "ISB stop too small (< min) — skip [D3.2 p1 @ 14:18]"
	case stopPts >= cfg.ISBTwentiesPts:
		return stopPts, false, "ISB stop in the twenties — must not be taken [D4.1 p1 @ 05:41]"
	case stopPts > cfg.StopCeilingPts:
		return stopPts, false, "stop over the ceiling — not worth trading [D3.3 p1 @ 02:06]"
	default:
		return stopPts, true, ""
	}
}

// ISBArm tracks one resting inside-bar order through the stacking arithmetic.
// The reference is ALWAYS the FIRST inside-bar candle [D4.2 p2 @ 08:49–16:08].
type ISBArm struct {
	FirstBar market.Kline // the first ISB candle (the reference)
	Inside   int          // candles whose BODIES stayed inside the first ISB since placement
}

// ISBStackAdvice is the stacking arithmetic [D4.2 p2 @ 08:49–16:08]:
//
//	1 candle inside → CANCEL (the optional rule [D4.1 p1 @ 08:05]);
//	2–3 → HOLD (a 2–3-minute inside bar);
//	4 → CANCEL (a 4-minute ISB does not exist — it becomes a 5-minute ISB,
//	"INSIDE BAR KHUNG 5 PHÚT LÀ MÌNH HOÀN TOÀN KHÔNG ĐƯỢC TRADE" [@ 15:43]).
//
// It returns "" (no action) for hold.
func ISBStackAdvice(inside int) string {
	switch inside {
	case 1:
		return "cancel"
	case 2, 3:
		return "hold"
	case 4:
		return "cancel"
	default:
		return "hold" // 0: just placed; >4: already cancelled by the evaluator
	}
}

// ISBStackTick advances the arm by one closed candle. If the candle's BODY
// stays inside the first ISB's body range, the counter advances and the
// arithmetic runs; otherwise the setup is over (price escaped) and the arm
// ends with an empty advice. A cancel advice emits a CancelArm intent.
func ISBStackTick(arm *ISBArm, bar market.Kline, cfg Config) []Intent {
	if !cfg.Enabled {
		return nil
	}
	if !(curHigh(bar) <= curHigh(arm.FirstBar) && curLow(bar) >= curLow(arm.FirstBar)) {
		arm.Inside = -1 // escaped — the arm is over (P3 cancels the order)
		return nil
	}
	arm.Inside++
	if ISBStackAdvice(arm.Inside) == "cancel" {
		return []Intent{{
			Action: CancelArm,
			Reason: "ISB stacking: " + itoa(arm.Inside) + " candle(s) inside without a fill — cancel [D4.2 p2 @ 08:49–16:08]",
		}}
	}
	return nil
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return "9+"
}
