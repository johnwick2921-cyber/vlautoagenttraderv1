package mentor

import (
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"vl/market"
)

// §8 — THE SWING SETUP, 4-hour EMA 34 → 5-minute [D5.2], with the corrected
// §3 (reject = close back on the APPROACH side; close through = wrong way).
//
//  1. "On the 4-hour, draw the line at EMA 34 BEFORE price gets there"
//     [D5.2 p1 @ 04:36, 11:04]. The line is the EMA 34 known at the START of
//     the current 4h bar: closed buckets only, and the line FLIPS when a 4h
//     candle finishes — "do not wait for a retest" [p3 @ 12:30].
//  2. "Switch STRAIGHT DOWN TO THE 5-MINUTE — not the 1-minute" [p1 @ 05:30].
//  3. "Wait for a LITERAL touch. 'KHÔNG ĐƯỢC GẦN ĐỤNG'" [p2 @ 09:15] — the
//     only setup where a near-touch is not accepted. The 4-hour is too
//     coarse to place exactly, so the line placement offset is 5–10 points
//     (SwingCfg.LineOffsetPts, default 5) [table, "Line tolerance"].
//  4. "First touching candle is the reference; 2 candles of leeway allowed
//     here only" [p1 @ 15:21] — implemented as the ISB window below.
//  5. Two cases [p3 @ 21:30, corrected §3]:
//   - Touch, closes BACK on the approach side (a REJECT) → stop order
//     beyond that candle. "Stop (clean rejection): ~30 points above the
//     line" [table].
//   - Touch, closes THROUGH → CANCEL (invalid for this approach). "If it
//     then closes back, wait for a 5-MINUTE INSIDE BAR" — the ISB entry
//     has the "stop RIGHT AT the level" [table].
//  6. "Stop ~100 points → DO NOT ENTER" [table] → SwingCfg.MaxStopPts.
//  7. Targets [table]: first = "EMA 34 on the 5-minute" (or the 50-pt
//     fallback); BE at +1R; hold to the close of the 2nd 4h candle after
//     entry (the method does not state the hold length — knob [C]).
//  8. One setup per approach: no new one until price has left the line by
//     at least the stop distance and comes back.
//
// Hard invariant: the stop must sit on the correct side of the entry, else
// no intent is emitted — logged and counted (swingInvalidStops).

// Swing-specific intents (the package's Action set is extended here).
const (
	// ActionMoveStopBE asks the injector to move the stop to break-even.
	ActionMoveStopBE Action = "move_stop_to_break_even"
	// ActionClosePosition asks the injector to close the position.
	ActionClosePosition Action = "close_position"
)

// SwingCfg holds the §8 knobs.
type SwingCfg struct {
	EMAPeriod         int     // 34
	LineOffsetPts     float64 // literal-touch placement tolerance; default 5
	StopBeyondLinePts float64 // clean-rejection stop distance; default 30
	MaxStopPts        float64 // ≥ this → skip; default 100
	EntryBufferPts    float64 // stop order beyond the reference candle; default 1.5
	TargetFallbackPts float64 // first target when no 5m EMA34; default 50
	TargetEMA5mPeriod int     // first target EMA period on 5m; default 34
	LeewayCandles     int     // ISB window after a through-close; default 2
	Hold4hBars        int     // hold to the close of the N-th 4h candle after entry; default 2 [C]
}

// DefaultSwingCfg returns the §8 defaults.
func DefaultSwingCfg() SwingCfg {
	return SwingCfg{
		EMAPeriod:         34,
		LineOffsetPts:     5,
		StopBeyondLinePts: 30,
		MaxStopPts:        100,
		EntryBufferPts:    1.5,
		TargetFallbackPts: 50,
		TargetEMA5mPeriod: 34,
		LeewayCandles:     2,
		Hold4hBars:        2,
	}
}

// swingInvalidStops counts the hard-invariant violations (log + counter).
var swingInvalidStops atomic.Int64

// SwingInvalidStops exposes the counter for tests and telemetry.
func SwingInvalidStops() int64 { return swingInvalidStops.Load() }

// SwingState is everything the swing machine carries between ticks. It is
// rebuildable: replaying the same 5m series from a zero state reproduces it.
type SwingState struct {
	Line         float64 // the 4h EMA 34 in force
	BucketStart  int64   // ms, the 4h bucket the line belongs to
	LastBarTime  int64   // ms, last closed 5m bar processed
	FirstTouch   *swingTouch
	LeewayLeft   int     // candles left in the ISB window after a through-close
	ClearLongAt  float64 // price ≤ this re-arms longs (line − stop)
	ClearShortAt float64 // price ≥ this re-arms shorts (line + stop)
	Pos          *swingPosition
}

type swingTouch struct {
	RefBar   market.Kline // the first touching 5m candle
	Approach Side         // price came from this side: SideLong = from above
	Through  bool         // the touch closed THROUGH the line
}

type swingPosition struct {
	Side        Side
	Entry       float64
	Stop        float64
	Target      float64
	Risk        float64
	EntryBucket int64
	ArmID       string
	MovedBE     bool
}

// fourHBucketStart returns the start (ms) of the 4h bucket a bar belongs to.
// Buckets are anchored at 17:00 CT (the Globex session open) — a local
// helper; TODO: switch to DS-103's B1 clock-aligned buckets once they land.
func fourHBucketStart(ot int64, loc *time.Location) int64 {
	if loc == nil {
		loc = ctime()
	}
	t := time.UnixMilli(ot).In(loc)
	anchor := time.Date(t.Year(), t.Month(), t.Day(), globexOpenMin/60, globexOpenMin%60, 0, 0, loc)
	if t.Before(anchor) {
		anchor = anchor.AddDate(0, 0, -1)
	}
	delta := t.Sub(anchor)
	step := 4 * time.Hour
	return anchor.Add(delta - delta%step).UnixMilli()
}

// swingLine computes the 4h EMA 34 known at the START of the current 4h bar:
// the EMA over closed 4h buckets (17:00 CT anchored), built from the closed
// 5m series.
func swingLine(bars5m []market.Kline, now int64, cfg SwingCfg, loc *time.Location) (line float64, bucketStart int64, ok bool) {
	if loc == nil {
		loc = ctime()
	}
	cur := fourHBucketStart(now, loc)
	var closes []float64
	var last int64
	for _, b := range bars5m {
		bs := fourHBucketStart(b.OpenTime, loc)
		if bs >= cur {
			continue // the current bucket is still open — never counted
		}
		if last == bs {
			closes[len(closes)-1] = b.Close // keep the bucket's last close
			continue
		}
		closes = append(closes, b.Close)
		last = bs
	}
	if len(closes) < 2 {
		return 0, cur, false
	}
	kbars := make([]market.Kline, len(closes))
	for i, c := range closes {
		kbars[i] = market.Kline{Close: c}
	}
	return emaValue(kbars, cfg.EMAPeriod), cur, true
}

// SwingTick advances the swing machine over the closed 5m bars since the
// last tick and returns the intents. Pure in (state, bars): the same tape
// rebuilds the same state.
func SwingTick(s *SwingState, bars5m []market.Kline, cfg SwingCfg, now int64) []Intent {
	if cfg.EMAPeriod <= 0 || cfg.StopBeyondLinePts <= 0 {
		cfg = DefaultSwingCfg()
	}
	loc := ctime()
	var out []Intent
	if s == nil {
		s = &SwingState{}
	}
	if s.Pos != nil {
		out = append(out, manageSwingPosition(s, bars5m, cfg)...)
	}
	line, bucketStart, ok := swingLine(bars5m, now, cfg, loc)
	if !ok {
		s.Line, s.BucketStart = 0, bucketStart
		return out
	}
	if bucketStart != s.BucketStart {
		// A 4h candle finished: FLIP to the new line, do not wait for a
		// retest [D5.2 p3 @ 12:30]. The old line's pending work dies.
		s.Line = line
		s.BucketStart = bucketStart
		s.FirstTouch = nil
		s.LeewayLeft = 0
		s.ClearLongAt, s.ClearShortAt = 0, 0
	}

	for i := 1; i < len(bars5m); i++ {
		b := bars5m[i]
		if b.OpenTime <= s.LastBarTime || b.CloseTime == 0 {
			continue
		}
		s.LastBarTime = b.OpenTime
		prev := bars5m[i-1]
		// one-setup-per-approach re-arm: price left the line by the stop
		// distance and has room to come back.
		if b.High >= line+cfg.StopBeyondLinePts {
			s.ClearShortAt = 0
		}
		if b.Low <= line-cfg.StopBeyondLinePts {
			s.ClearLongAt = 0
		}
		if s.FirstTouch != nil && s.LeewayLeft > 0 {
			s.LeewayLeft--
			// closes back after a through-close → the 5m INSIDE BAR entry
			// with the stop AT the level [D5.2 p3 @ 21:56].
			if s.FirstTouch.Through && closedBack(s.FirstTouch.Approach, b.Close, line) {
				if IsISB(prev, b) {
					if in, eok := swingISBIntent(s.FirstTouch, b, line, cfg); eok {
						s.FirstTouch = nil
						s.LeewayLeft = 0
						in.ArmID = "swing-" + strconv.FormatInt(b.OpenTime, 10)
						s.openPosition(in, b.OpenTime)
						out = append(out, in)
						continue
					}
				}
			}
			continue
		}
		if !touchesLine(b, line, cfg.LineOffsetPts) {
			continue
		}
		approach, hasApproach := swingApproach(prev.Close, line)
		if !hasApproach {
			continue // tie: prev closed exactly on the line — no approach yet
		}
		if approach == SideLong && s.ClearLongAt != 0 && b.Low > s.ClearLongAt {
			continue // not yet re-armed
		}
		if approach == SideShort && s.ClearShortAt != 0 && b.High < s.ClearShortAt {
			continue // not yet re-armed
		}
		if !closedBack(approach, b.Close, line) {
			// closes THROUGH → CANCEL [D5.2 p3 @ 21:30, corrected §3]
			out = append(out, Intent{
				Action: CancelArm,
				Reason: "swing: touch closed through the line — cancel, invalid for this approach [D5.2 p3 @ 21:30, corrected §3]",
			})
			s.FirstTouch = &swingTouch{RefBar: b, Approach: approach, Through: true}
			s.LeewayLeft = cfg.LeewayCandles
			continue
		}
		in, eok := swingRejectIntent(approach, b, line, cfg)
		if !eok {
			s.FirstTouch = nil
			continue
		}
		if ema5 := swingTargetEMA(bars5m, cfg); targetOnSide(in.Side, in.Price, ema5) {
			in.Target = ema5
		}
		in.ArmID = "swing-" + strconv.FormatInt(b.OpenTime, 10)
		if approach == SideLong {
			s.ClearLongAt = line - cfg.StopBeyondLinePts
		} else {
			s.ClearShortAt = line + cfg.StopBeyondLinePts
		}
		s.FirstTouch = &swingTouch{RefBar: b, Approach: approach}
		s.openPosition(in, b.OpenTime)
		out = append(out, in)
	}
	return out
}

// openPosition records the entry for the BE/hold management.
func (s *SwingState) openPosition(in Intent, at int64) {
	p := &swingPosition{Side: in.Side, Entry: in.Price, Stop: in.Stop, Target: in.Target, EntryBucket: s.BucketStart, ArmID: in.ArmID}
	p.Risk = in.Stop - in.Price
	if in.Side == SideShort {
		p.Risk = in.Price - in.Stop
	}
	s.Pos = p
}

// swingTargetEMA is the first target: the 5m EMA 34 [table].
func swingTargetEMA(bars5m []market.Kline, cfg SwingCfg) float64 {
	if len(bars5m) < cfg.TargetEMA5mPeriod {
		return 0
	}
	return emaValue(bars5m[len(bars5m)-cfg.TargetEMA5mPeriod:], cfg.TargetEMA5mPeriod)
}

// targetOnSide reports whether the EMA target sits on the profitable side of
// the entry (otherwise the fallback stands).
func targetOnSide(side Side, entry, ema float64) bool {
	if ema == 0 {
		return false
	}
	if side == SideLong {
		return ema > entry
	}
	return ema < entry
}

// touchesLine is the LITERAL touch: the candle's range reaches the line
// within the placement offset — "KHÔNG ĐƯỢC GẦN ĐỤNG" [D5.2 p2 @ 09:15].
func touchesLine(b market.Kline, line, offset float64) bool {
	return b.Low <= line+offset && b.High >= line-offset
}

// swingApproach is the approach side from the previous bar's close vs the
// line. A close EXACTLY on the line is a tie: no approach yet — wait for a
// bar that closes off the level (CTO 2026-10-03, mirror of touch.go's
// approachSide rule).
func swingApproach(prevClose, line float64) (Side, bool) {
	if prevClose == line {
		return "", false
	}
	if prevClose < line {
		return SideShort, true // came from below → resistance
	}
	return SideLong, true // came from above → support
}

// closedBack reports the corrected §3 reject: the close is BACK on the side
// price came FROM. Came from above (support, long) → close above the line;
// came from below (resistance, short) → close below.
func closedBack(approach Side, close, line float64) bool {
	if approach == SideLong {
		return close > line
	}
	return close < line
}

// swingRejectIntent builds the clean-rejection stop order: beyond the
// reference candle, stop 30 pts beyond the line. Refused when the stop
// distance reaches MaxStopPts or the hard invariant (stop on the correct
// side of the entry) breaks.
func swingRejectIntent(approach Side, ref market.Kline, line float64, cfg SwingCfg) (Intent, bool) {
	var in Intent
	if approach == SideShort { // resistance: came from below → sell stop
		in = Intent{
			Action: PlaceStopEntry,
			Side:   SideShort,
			Price:  ref.Low - cfg.EntryBufferPts,
			Stop:   line + cfg.StopBeyondLinePts,
		}
	} else {
		in = Intent{
			Action: PlaceStopEntry,
			Side:   SideLong,
			Price:  ref.High + cfg.EntryBufferPts,
			Stop:   line - cfg.StopBeyondLinePts,
		}
	}
	if abs(in.Stop-in.Price) >= cfg.MaxStopPts {
		return Intent{}, false
	}
	if in.Side == SideLong && in.Stop >= in.Price || in.Side == SideShort && in.Stop <= in.Price {
		swingInvalidStops.Add(1)
		log.Printf("swing4h: hard invariant broken — %s entry %.2f stop %.2f refused",
			in.Side, in.Price, in.Stop)
		return Intent{}, false
	}
	in.Target = in.Price + cfg.TargetFallbackPts
	if in.Side == SideShort {
		in.Target = in.Price - cfg.TargetFallbackPts
	}
	in.Reason = "swing §8: reject touch, stop order beyond the reference candle, stop 30 pts beyond the 4h EMA 34 [D5.2 p3 @ 21:30 corrected, table]"
	return in, true
}

// swingISBIntent builds the inside-bar entry after a through-close: stop AT
// the level, "even if it feels big" [table]. ref is the INSIDE-BAR candle.
func swingISBIntent(t *swingTouch, ref market.Kline, line float64, cfg SwingCfg) (Intent, bool) {
	var in Intent
	if t.Approach == SideShort {
		in = Intent{Action: PlaceStopEntry, Side: SideShort, Price: ref.Low - cfg.EntryBufferPts, Stop: line}
	} else {
		in = Intent{Action: PlaceStopEntry, Side: SideLong, Price: ref.High + cfg.EntryBufferPts, Stop: line}
	}
	if abs(in.Stop-in.Price) >= cfg.MaxStopPts {
		return Intent{}, false
	}
	if in.Side == SideLong && in.Stop >= in.Price || in.Side == SideShort && in.Stop <= in.Price {
		swingInvalidStops.Add(1)
		log.Printf("swing4h: hard invariant broken — ISB entry %s %.2f stop %.2f refused", in.Side, in.Price, in.Stop)
		return Intent{}, false
	}
	in.Target = in.Price + cfg.TargetFallbackPts
	if in.Side == SideShort {
		in.Target = in.Price - cfg.TargetFallbackPts
	}
	in.Reason = "swing §8: through-close, then closes back → 5m inside bar with the stop AT the level [D5.2 p3 @ 21:56, table]"
	return in, true
}

// manageSwingPosition emits the BE and hold-close intents for an open swing.
func manageSwingPosition(s *SwingState, bars []market.Kline, cfg SwingCfg) []Intent {
	p := s.Pos
	if p == nil || len(bars) == 0 {
		return nil
	}
	cur := bars[len(bars)-1]
	var out []Intent
	if !p.MovedBE {
		if p.Side == SideLong && cur.Close >= p.Entry+p.Risk || p.Side == SideShort && cur.Close <= p.Entry-p.Risk {
			out = append(out, Intent{Action: ActionMoveStopBE, ArmID: p.ArmID,
				Reason: "swing §8: +1R reached — stop to break-even"})
			p.MovedBE = true
		}
	}
	holdMs := p.EntryBucket + int64(cfg.Hold4hBars)*4*3600_000
	if cur.OpenTime >= holdMs {
		out = append(out, Intent{Action: ActionClosePosition, ArmID: p.ArmID,
			Reason: "swing §8: hold to the close of the 2nd 4h candle after entry"})
		s.Pos = nil
	}
	return out
}
