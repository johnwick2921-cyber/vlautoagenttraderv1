package mentor

import (
	"testing"
	"time"

	"vl/market"
)

// §8 swing tests, built from the source quotes [D5.2 p1–p3, table] with the
// corrected §3 (reject = close back on the approach side; through = cancel).
// Bars are synthetic 5m candles in fixed CT (−5); the 4h buckets are
// anchored at 17:00 CT and the line is the EMA 34 of CLOSED buckets.

// The tape: three closed 4h buckets with closes 10000, 11000, 12000 → the
// line is EMA 34 ≈ 10168.16. The closed-bucket bars stay FAR from the line
// (they never touch it); the touch happens only in the current bucket.
func mk5m(t *testing.T, day, hour, minute int, o, h, l, c float64) market.Kline {
	t.Helper()
	loc := time.FixedZone("CT", -5*3600)
	ot := time.Date(2026, 9, day, hour, minute, 0, 0, loc)
	return market.Kline{OpenTime: ot.UnixMilli(), Open: o, High: h, Low: l, Close: c, CloseTime: ot.UnixMilli() + 4*60_000}
}

func swingTape(t *testing.T, cur []market.Kline) []market.Kline {
	t.Helper()
	closed := []market.Kline{
		mk5m(t, 14, 17, 5, 9999, 10000, 9998, 10000),   // bucket 17:00 close 10000
		mk5m(t, 14, 21, 5, 10999, 11000, 10998, 11000), // bucket 21:00 close 11000
		mk5m(t, 15, 1, 5, 11999, 12000, 11998, 12000),  // bucket 01:00 close 12000
	}
	return append(closed, cur...)
}

const swingLineGolden = 10168.16 // EMA 34 of 10000/11000/12000 ≈ 10168.1633

// TestSwing4hRejectShort — resistance: price came from BELOW, the first
// touching 5m candle closes BACK below the line → sell stop beyond the
// reference candle, stop ~30 pts above the line [D5.2 p3 @ 21:30 corrected;
// table "Stop (clean rejection)"].
func TestSwing4hRejectShort(t *testing.T) {
	cfg := DefaultSwingCfg()
	cur := []market.Kline{
		mk5m(t, 15, 5, 0, 9950, 9960, 9945, 9955),     // prev: below the line
		mk5m(t, 15, 5, 5, 10160, 10175, 10155, 10160), // touches, closes back below (reject)
	}
	bars := swingTape(t, cur)
	s := &SwingState{}
	out := SwingTick(s, bars, cfg, cur[1].OpenTime+60_000)
	if len(out) != 1 {
		t.Fatalf("intents = %d, want exactly the reject stop entry; got %+v", len(out), out)
	}
	in := out[0]
	if in.Action != PlaceStopEntry || in.Side != SideShort {
		t.Fatalf("intent = %+v, want a SHORT stop entry", in)
	}
	if abs(in.Price-10153.5) > 0.01 { // ref low 10155 − buffer 1.5
		t.Fatalf("entry = %.2f, want 10153.5 (sell stop under the rejecting candle)", in.Price)
	}
	if abs(in.Stop-(swingLineGolden+30)) > 0.01 {
		t.Fatalf("stop = %.2f, want %.2f (30 pts above the line)", in.Stop, swingLineGolden+30)
	}
}

// TestSwing4hRejectLong — support: came from above, closes back above →
// buy stop over the reference candle, stop 30 below the line.
func TestSwing4hRejectLong(t *testing.T) {
	cfg := DefaultSwingCfg()
	cur := []market.Kline{
		mk5m(t, 15, 5, 0, 10200, 10210, 10195, 10205), // prev: above the line
		mk5m(t, 15, 5, 5, 10165, 10175, 10155, 10175), // touches, closes back above (reject)
	}
	bars := swingTape(t, cur)
	s := &SwingState{}
	out := SwingTick(s, bars, cfg, cur[1].OpenTime+60_000)
	if len(out) != 1 || out[0].Side != SideLong {
		t.Fatalf("intents = %+v, want one LONG stop entry", out)
	}
	in := out[0]
	if abs(in.Price-10176.5) > 0.01 { // ref high 10175 + 1.5
		t.Fatalf("entry = %.2f, want 10176.5 (buy stop over the rejecting candle)", in.Price)
	}
	if abs(in.Stop-(swingLineGolden-30)) > 0.01 {
		t.Fatalf("stop = %.2f, want %.2f (30 pts below the line)", in.Stop, swingLineGolden-30)
	}
}

// TestSwing4hThroughCancelsThenISB — touch closes THROUGH → CANCEL; if it
// then closes back within the leeway, a 5m inside bar enters with the stop
// AT the level [D5.2 p3 @ 21:30, 21:56; table "Stop (inside-bar entry)"].
func TestSwing4hThroughCancelsThenISB(t *testing.T) {
	cfg := DefaultSwingCfg()
	cur := []market.Kline{
		mk5m(t, 15, 5, 0, 9950, 9960, 9945, 9955),      // prev: below
		mk5m(t, 15, 5, 5, 10155, 10220, 10150, 10220),  // touches, closes THROUGH (above)
		mk5m(t, 15, 5, 10, 10175, 10180, 10150, 10160), // closes back below; ISB of the touch candle
	}
	bars := swingTape(t, cur)
	s := &SwingState{}
	out := SwingTick(s, bars, cfg, cur[2].OpenTime+60_000)
	if len(out) != 2 {
		t.Fatalf("intents = %d, want cancel + ISB entry; got %+v", len(out), out)
	}
	if out[0].Action != CancelArm {
		t.Fatalf("first intent = %+v, want CancelArm for the through-close", out[0])
	}
	in := out[1]
	if in.Action != PlaceStopEntry || in.Side != SideShort {
		t.Fatalf("ISB intent = %+v, want a SHORT stop entry", in)
	}
	if abs(in.Stop-swingLineGolden) > 0.01 { // RIGHT AT the level
		t.Fatalf("ISB stop = %.2f, want %.2f (AT the level)", in.Stop, swingLineGolden)
	}
	if abs(in.Price-10148.5) > 0.01 { // ISB candle low 10150 − 1.5
		t.Fatalf("ISB entry = %.2f, want 10148.5", in.Price)
	}
}

// TestSwing4hStopCeiling100 — "Stop ~100 points → DO NOT ENTER" [table].
func TestSwing4hStopCeiling100(t *testing.T) {
	cfg := DefaultSwingCfg()
	cur := []market.Kline{
		mk5m(t, 15, 5, 0, 9950, 9960, 9945, 9955),
		mk5m(t, 15, 5, 5, 10055, 10175, 10050, 10060), // touches, reject, entry 10048.5 → spread ~149.7
	}
	bars := swingTape(t, cur)
	s := &SwingState{}
	out := SwingTick(s, bars, cfg, cur[1].OpenTime+60_000)
	if len(out) != 0 {
		t.Fatalf("stop ≥ 100 must skip; got %+v", out)
	}
}

// TestSwing4hHardInvariantCounter — the stop must sit on the correct side of
// the entry, else no intent and the counter ticks (log + counter).
func TestSwing4hHardInvariantCounter(t *testing.T) {
	cfg := DefaultSwingCfg()
	before := SwingInvalidStops()
	// line 155, long approach: entry = 120 + 1.5 = 121.5, stop = 155 − 30 =
	// 125 → the stop sits ABOVE the entry — the invariant breaks.
	if _, ok := swingRejectIntent(SideLong, market.Kline{High: 120, Low: 119}, 155, cfg); ok {
		t.Fatal("invariant violation must be refused")
	}
	if SwingInvalidStops() != before+1 {
		t.Fatalf("counter = %d, want %d (one refused intent logged)", SwingInvalidStops(), before+1)
	}
}

// TestSwing4hOneSetupPerApproach — one setup per approach: no new one until
// price has left the line by at least the stop distance and comes back. The
// return path crosses through (cancel), the leeway expires, and the next
// below-the-line touch is the new setup.
func TestSwing4hOneSetupPerApproach(t *testing.T) {
	cfg := DefaultSwingCfg()
	cur := []market.Kline{
		mk5m(t, 15, 5, 0, 9950, 9960, 9945, 9955),      // prev below
		mk5m(t, 15, 5, 5, 10160, 10175, 10155, 10160),  // first setup (short)
		mk5m(t, 15, 5, 10, 10160, 10175, 10155, 10160), // same approach again — blocked
		mk5m(t, 15, 5, 15, 10190, 10210, 10190, 10205), // leaves the line by ≥ 30 (entirely above)
		mk5m(t, 15, 5, 20, 10200, 10200, 10150, 10160), // crosses back through → cancel
		mk5m(t, 15, 5, 25, 10195, 10205, 10150, 10155), // leeway candle 1 (closes below → no ISB)
		mk5m(t, 15, 5, 30, 10195, 10205, 10150, 10155), // leeway candle 2 (closes below → no ISB)
		mk5m(t, 15, 5, 35, 10060, 10100, 10050, 10060), // back below without touching
		mk5m(t, 15, 5, 40, 10160, 10175, 10155, 10160), // below-line touch → new setup
	}
	bars := swingTape(t, cur)
	s := &SwingState{}
	out := SwingTick(s, bars, cfg, cur[8].OpenTime+60_000)
	entries, cancels := 0, 0
	for _, in := range out {
		switch in.Action {
		case PlaceStopEntry:
			entries++
		case CancelArm:
			cancels++
		}
	}
	if entries != 2 {
		t.Fatalf("stop entries = %d, want 2 (the blocked repeat produces none); got %+v", entries, out)
	}
	if cancels != 1 {
		t.Fatalf("cancels = %d, want 1 (the through-cross back); got %+v", cancels, out)
	}
}

// TestSwing4hFlipOnNewBucket — when a 4h candle finishes the line FLIPS to
// the new one and pending work dies: "do not wait for a retest" [D5.2 p3
// @ 12:30].
func TestSwing4hFlipOnNewBucket(t *testing.T) {
	cfg := DefaultSwingCfg()
	cur := []market.Kline{
		mk5m(t, 15, 5, 0, 9950, 9960, 9945, 9955),
		mk5m(t, 15, 5, 5, 10155, 10220, 10150, 10220), // through-close → pending
	}
	bars := swingTape(t, cur)
	s := &SwingState{}
	SwingTick(s, bars, cfg, cur[1].OpenTime+60_000)
	if s.FirstTouch == nil || s.LeewayLeft == 0 {
		t.Fatalf("setup: pending touch not armed: %+v", s.FirstTouch)
	}
	// advance into the NEXT 4h bucket (09:00), one more closed bucket close
	next := append(bars, mk5m(t, 15, 9, 5, 12999, 13000, 12998, 13000))
	SwingTick(s, next, cfg, next[len(next)-1].OpenTime+60_000)
	if s.FirstTouch != nil {
		t.Fatal("the flip must drop the pending touch on the old line")
	}
	if s.Line == 0 || s.BucketStart != fourHBucketStart(next[len(next)-1].OpenTime+60_000, nil) {
		t.Fatal("the line must flip to the new bucket")
	}
}

// TestSwing4hBEAndHold — BE at +1R, then hold to the close of the 2nd 4h
// candle after entry [table]. Ticks are incremental: the BE/hold management
// runs at the START of a tick over the position's book.
func TestSwing4hBEAndHold(t *testing.T) {
	cfg := DefaultSwingCfg()
	entryBars := swingTape(t, []market.Kline{
		mk5m(t, 15, 5, 0, 9950, 9960, 9945, 9955),     // prev below
		mk5m(t, 15, 5, 5, 10160, 10175, 10155, 10160), // short entry 10153.5, risk ≈ 44.66
	})
	s := &SwingState{}
	out := SwingTick(s, entryBars, cfg, entryBars[len(entryBars)-1].OpenTime+60_000)
	entries := 0
	for _, in := range out {
		if in.Action == PlaceStopEntry {
			entries++
		}
	}
	if entries != 1 {
		t.Fatalf("setup entries = %d, want 1; got %+v", entries, out)
	}
	// next tick: a bar closing at −1R (≤ 10108.84) → BE
	beBars := append(entryBars, mk5m(t, 15, 5, 10, 10095, 10100, 10090, 10095))
	out = SwingTick(s, beBars, cfg, beBars[len(beBars)-1].OpenTime+60_000)
	var be bool
	for _, in := range out {
		if in.Action == ActionMoveStopBE {
			be = true
		}
	}
	if !be {
		t.Fatalf("no BE intent at +1R; got %+v", out)
	}
	// advance past the close of the 2nd 4h candle after entry (05:00 + 8h)
	late := append(beBars, mk5m(t, 15, 13, 5, 10095, 10100, 10090, 10095))
	out2 := SwingTick(s, late, cfg, late[len(late)-1].OpenTime+60_000)
	var closed bool
	for _, in := range out2 {
		if in.Action == ActionClosePosition {
			closed = true
		}
	}
	if !closed {
		t.Fatalf("no close at the 2nd-4h-candle hold limit; got %+v", out2)
	}
}
