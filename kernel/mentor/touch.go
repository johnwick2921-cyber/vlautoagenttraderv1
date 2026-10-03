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
	// TouchReject: the reference candle closed BACK on the side price came
	// FROM — the CTO-corrected reading (2026-10-02): at a resistance the
	// candle closes BELOW the line ("ĐỤNG, ĐÓNG DƯỚI — ĐẶT LỆNH SELL STOP…
	// đang đụng RESISTANCE" [D5.2 p3 @ 21:30]; "REJECT là cái việc mà NÓ
	// ĐÓNG DƯỚI" [@ 23:32]) — the setup layer places the stop order beyond
	// that candle [D5.2 p3 @ 17:32].
	TouchReject TouchOutcome = "reject"
	// TouchWrongWay: the reference candle closed THROUGH the line —
	// "ĐỤNG, ĐÓNG LÊN TRÊN — CANCEL LỆNH…" [D5.2 p3 @ 21:30] — and the
	// LEVEL IS INVALID: only ISBs may ever trade there after
	// [D5.2 p1 @ 19:51; D5.2 p2 @ 20:48].
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
	// PriceAtTouch is the level price when the touch was classified — for a
	// MOVING line (EMA) the evaluator resets the touch when the line drifts
	// away, so a touch against yesterday's EMA position is not treated as a
	// touch of today's.
	PriceAtTouch float64
}

// TouchTick evaluates one closed 1m candle against one level (§3 steps 1–3,
// CTO-corrected reading 2026-10-02):
//
//  1. WAIT for a literal touch (the candle's range reaches the level)
//     [D3.3 p1 @ 00:13];
//  2. only the FIRST touching candle matters — later candles never reclassify
//     [D3.3 p1 @ 00:31; D5.3 p1 @ 20:40];
//  3. REJECT = the candle closes BACK on the approach side → the setup layer
//     places the stop order beyond the candle. Close THROUGH the line =
//     the wrong side → CancelArm + LevelInvalid intents.
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
	if t.ApproachedFrom == "" {
		// tie: the previous close sat exactly ON the level — no approach side
		// yet (CTO review 2d3aad1ef). Wait for a bar that closes off the level.
		return nil
	}
	t.RefBar = bar
	t.PriceAtTouch = level.Price
	switch t.ApproachedFrom {
	case SideShort: // came from below: resistance — reject = closes BACK below
		if bar.Close < level.Price {
			t.Outcome = TouchReject // "ĐỤNG, ĐÓNG DƯỚI — ĐẶT LỆNH SELL STOP" [D5.2 p3 @ 21:30]
		} else {
			t.Outcome = TouchWrongWay // closed through (above) — "ĐÓNG LÊN TRÊN — CANCEL" [@ 21:30]
		}
	default: // came from above: support — reject = closes back above
		if bar.Close > level.Price {
			t.Outcome = TouchReject
		} else {
			t.Outcome = TouchWrongWay // closed through (below)
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
// reference candle [D5.2 p3 @ 17:32] — at a RESISTANCE (came from below) the
// candle closed back below → SELL stop under RefBar.Low; at a SUPPORT (came
// from above) it closed back above → BUY stop over RefBar.High, buffered by
// ISBBufferPts [D2.1 p1 @ 05:36]. Stop and target are completed by the setup
// layer (PHL/PLH commit); here only the order side and trigger price are known.
func RejectEntry(t Touch, cfg Config) (side Side, price float64, ok bool) {
	if t.Outcome != TouchReject {
		return "", 0, false
	}
	switch t.ApproachedFrom {
	case SideShort: // resistance: rejected back below → sell stop under the candle
		return SideShort, t.RefBar.Low - cfg.ISBBufferPts, true
	default: // support: rejected back above → buy stop over the candle
		return SideLong, t.RefBar.High + cfg.ISBBufferPts, true
	}
}

func approachSide(prevClose, price float64) Side {
	if prevClose > price {
		return SideLong // above → level is support, approached from above
	}
	if prevClose < price {
		return SideShort // below → resistance, approached from below
	}
	return "" // tie: no approach yet (CTO review 2d3aad1ef)
}
