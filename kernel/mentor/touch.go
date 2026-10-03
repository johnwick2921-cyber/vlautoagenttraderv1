package mentor

import (
	"vl/market"
)

// TouchOutcome is the §3 entry-procedure classification of the FIRST candle
// that touches a level [D3.3 p1 @ 00:13 — "lúc nào cũng lấy cái nến ĐẦU TIÊN
// chạm level làm chuẩn"].
type TouchOutcome string

const (
	// TouchNone: not touched yet.
	TouchNone TouchOutcome = ""
	// TouchReject: the reference candle closed on the FAR side of the level —
	// "REJECT là cái việc mà NÓ ĐÓNG DƯỚI" [D5.2 p3 @ 23:32] — the setup
	// layer places the stop order beyond that candle [D5.2 p3 @ 17:32].
	TouchReject TouchOutcome = "reject"
	// TouchWrongWay: the reference candle closed back on the approach side —
	// "Closes on the wrong side → CANCEL. Không thắc mắc gì hết" [D4.1 p1
	// @ 14:14] — and the LEVEL IS INVALID: only ISBs may ever trade there
	// after [D5.2 p1 @ 19:51; D5.2 p2 @ 20:48].
	TouchWrongWay TouchOutcome = "wrong_way"
)

// Touch is the first-touch state at one level. It is rebuilt by replaying bars
// (see Replay), never RAM-only.
type Touch struct {
	LevelKey string
	Outcome  TouchOutcome
	RefBar   market.Kline // the first touching candle
	// ApproachedFrom names the side price stood on before the first touch
	// (the previous bar's close vs the level).
	ApproachedFrom Side
}

// TouchTick evaluates one closed 1m candle against one level (§3 steps 1–3):
//
//  1. WAIT for a literal touch (the candle's range reaches the level)
//     [D3.3 p1 @ 00:13];
//  2. only the FIRST touching candle matters — later candles never reclassify
//     [D3.3 p1 @ 00:31; D5.3 p1 @ 20:40];
//  3. close on the far side → TouchReject; close back on the approach side →
//     TouchWrongWay, which also emits CancelArm + LevelInvalid intents.
//
// A candle whose BODY closes through a level is NOT an entry: "A BREAK IS NOT
// A SETUP" [D4.1 p1 @ 18:27] — no PlaceStopEntry is ever emitted here.
func TouchTick(t *Touch, level Level, prevClose float64, bar market.Kline, cfg Config) []Intent {
	if !cfg.Enabled || t.Outcome != TouchNone {
		return nil
	}
	if bar.Low > level.Price+cfg.TouchBandPts || bar.High < level.Price-cfg.TouchBandPts {
		return nil // no touch
	}
	t.ApproachedFrom = approachSide(prevClose, level.Price)
	t.RefBar = bar
	switch t.ApproachedFrom {
	case SideShort: // price came from below: level acts as resistance
		if bar.Close > level.Price {
			t.Outcome = TouchReject // closed on the far side (above)
		} else {
			t.Outcome = TouchWrongWay
		}
	default: // price came from above: level acts as support
		if bar.Close < level.Price {
			t.Outcome = TouchReject // "đóng dưới"
		} else {
			t.Outcome = TouchWrongWay
		}
	}
	if t.Outcome != TouchWrongWay {
		return nil
	}
	return []Intent{
		{
			Action:   CancelArm,
			Reason:   "first touch closed on the wrong side — cancel [D4.1 p1 @ 14:14]",
			LevelKey: level.Key,
		},
		{
			Action:   LevelInvalid,
			Reason:   "first touch closed on the wrong side — level invalid, ISB-only after [D5.2 p1 @ 19:51]",
			LevelKey: level.Key,
		},
	}
}

// RejectEntry is the §3 order for a reject-close: the stop order beyond the
// reference candle [D5.2 p3 @ 17:32] — buy stop above it when it closed below,
// sell stop below it when it closed above, buffered by ISBBufferPts
// [D2.1 p1 @ 05:36]. Stop and target are completed by the setup layer
// (PHL/PLH commit); here only the order side and trigger price are known.
func RejectEntry(t Touch, cfg Config) (side Side, price float64, ok bool) {
	if t.Outcome != TouchReject {
		return "", 0, false
	}
	switch t.ApproachedFrom {
	case SideShort: // closed above the level (far side) → sell stop below it
		return SideShort, t.RefBar.Low - cfg.ISBBufferPts, true
	default: // closed below → buy stop above it
		return SideLong, t.RefBar.High + cfg.ISBBufferPts, true
	}
}

func approachSide(prevClose, price float64) Side {
	if prevClose > price {
		return SideLong // above → level is support, approached from above
	}
	return SideShort // below → resistance, approached from below
}
