package mentor

import (
	"vl/market"
)

// ISB — inside bar (RULES-FIX-v3 R1, CTO-verified 2026-10-03; this file was
// reworked from the P2 draft, whose 5–6-pt stop cap killed 17,001 ISBs in the
// replay and whose body-vs-body test was wrong).
//
// R1 [D1.4 p1 @ 06:13–11:25, confirmed "Go over lớp học" @ 08:44–09:55 + deck
// slide 21]: ISB = cur's BODY inside prev's FULL range, WICKS INCLUDED ("Inside
// có nghĩa là cái BODY của cây nến đó phải nằm BÊN TRONG cây nến trend"); cur's
// wicks and colour are free.

// IsISB reports whether cur's BODY sits inside prev's FULL range (wicks
// included) [D1.4 p1 @ 06:13–08:47].
func IsISB(prev, cur market.Kline) bool {
	return curHigh(cur) <= prev.High && curLow(cur) >= prev.Low
}

// ISBDirection is the ISB direction: the COLOUR OF CANDLE 1 — green → long,
// red → short; candle 2's colour is irrelevant [D1.4 p1 @ 09:20–10:20].
func ISBDirection(prev market.Kline) Side {
	if prev.Close >= prev.Open {
		return SideLong
	}
	return SideShort
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

// ISBOrders computes both candidate stop-limit orders for an inside-bar
// candle: ENTRY at the ISB candle's OWN extreme, never candle 1's
// [D1.4 p1 @ 10:33–11:25]; STOP at the opposite extreme [D1.4 p1 @ 14:42–14:44];
// the 1–1.5-pt buffer applies to BOTH entry and stop, outward
// [D1.4 p1 @ 22:22–22:30]. ORDER TYPE is STOP-LIMIT with limit = the stop
// (trigger) price: "đặt buy stop limit… nếu nó không fill thì thôi"
// [D1.4 p1 @ 24:41–24:55]. The caller picks the side from ISBDirection.
//
//	long:  buy stop-limit  at High + buffer, stop loss = Low − buffer
//	short: sell stop-limit at Low − buffer,  stop loss = High + buffer
//
// Risk = (High − Low) + 2×buffer.
func ISBOrders(candle market.Kline, cfg Config) (long, short Intent) {
	long = Intent{
		Action: PlaceStopLimitEntry,
		Side:   SideLong,
		Price:  candle.High + cfg.ISBBufferPts,
		Limit:  candle.High + cfg.ISBBufferPts,
		Stop:   candle.Low - cfg.ISBBufferPts,
		Reason: "ISB: buy stop-limit above the ISB high + buffer, stop below the ISB low − buffer [D1.4 p1 @ 10:33–11:25, 14:42–14:44, 22:22–22:30, 24:41–24:55]",
	}
	short = Intent{
		Action: PlaceStopLimitEntry,
		Side:   SideShort,
		Price:  candle.Low - cfg.ISBBufferPts,
		Limit:  candle.Low - cfg.ISBBufferPts,
		Stop:   candle.High + cfg.ISBBufferPts,
		Reason: "ISB: sell stop-limit below the ISB low − buffer, stop above the ISB high + buffer [D1.4 p1 @ 10:33–11:25, 14:42–14:44, 22:22–22:30, 24:41–24:55]",
	}
	return long, short
}

// ISBStopLimitOrder is the R1 order the evaluator emits: the single stop-limit
// entry in the CANDLE-1 direction. ok is false (with a cited reason) when the
// stop is in the twenties — the ONLY stop-size skip R1 has [D4.1 p1 @ 05:41].
func ISBStopLimitOrder(prev, cur market.Kline, cfg Config) (Side, Intent, bool, string) {
	dir := ISBDirection(prev)
	long, short := ISBOrders(cur, cfg)
	if dir == SideLong {
		if _, ok, reason := ISBStopVerdict(cur, cfg); !ok {
			return dir, Intent{}, false, reason
		}
		return dir, long, true, ""
	}
	if _, ok, reason := ISBStopVerdict(cur, cfg); !ok {
		return dir, Intent{}, false, reason
	}
	return dir, short, true, ""
}

// ISBStopVerdict applies the ONLY stop-size rule R1 has (RULES-FIX-v3): an ISB
// whose stop is IN THE TWENTIES must NOT be taken [D4.1 p1 @ 05:41]. There is
// NO fixed 5–6-pt stop and NO ceiling here — the replay's 5–6-pt cap was wrong
// and killed 17,001 ISBs (CTO, 2026-10-03). "In the twenties" is the window
// [ISBTwentiesPts, ISBTwentiesPts+10); a stop of thirty-plus is not skipped by
// R1 (R9's cap is DS-103's wiring, not this verdict).
func ISBStopVerdict(candle market.Kline, cfg Config) (stopPts float64, ok bool, reason string) {
	stopPts = (candle.High - candle.Low) + 2*cfg.ISBBufferPts
	if stopPts >= cfg.ISBTwentiesPts && stopPts < cfg.ISBTwentiesPts+10 {
		return stopPts, false, "ISB stop in the twenties — must not be taken [D4.1 p1 @ 05:41]"
	}
	return stopPts, true, ""
}

// ISBArm tracks one resting inside-bar order through the stacking arithmetic.
// The reference is ALWAYS the FIRST inside-bar candle [D4.2 p2 @ 08:49–16:08].
type ISBArm struct {
	FirstBar market.Kline // the first ISB candle (the reference)
	Inside   int          // candles whose BODIES stayed inside the first ISB since placement
	Side     Side         // the arm's R1 direction (candle-1 colour) — stored, not derived:
	// FirstBar is the ISB candle, whose own colour is the OPPOSITE side
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
	case 1, 2, 3:
		// CTO parity ruling 2026-10-03 (mail 1791008332386 #1, replay audit g):
		// ONE arm, held through the 2nd and 3rd inside candle, cancelled at the
		// 4th. The 1st inside candle KEEPS the arm (the old cancel-at-1 was dead
		// code that dominated the stacking holds).
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
