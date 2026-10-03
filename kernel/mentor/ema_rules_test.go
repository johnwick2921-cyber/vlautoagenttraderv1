package mentor

import (
	"testing"

	"vl/market"
)

// TestEMARoomRule — E1: the EMA34 setup refuses for room exactly like a level
// (reward < RoomMultiple x risk). The EMA34 rides the shared PHL/PLH path, so
// this pins the room rule at the EMA call shape.
func TestEMARoomRule(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	ema := Level{Key: string(KindEMA34), Kind: KindEMA34, Price: 100}

	tr := Touch{
		LevelKey:       ema.Key,
		Outcome:        TouchReject,
		ApproachedFrom: SideLong, // came from above: support → LONG (buy stop)
		RefBar:         market.Kline{High: 101, Low: 100},
	}

	// Old high 107.5: target 102.5 (shy 5) → reward 1.5 < 2x risk 1 → refuse.
	if in, ok, reason := PHLPLHR2(tr, Level{Price: 107.5}, 0, 10, 0, cfg); ok {
		t.Fatalf("EMA room rule did not refuse (T < R): %+v", in)
	} else if reason == "" {
		t.Fatal("empty refusal reason")
	}

	// Old high 110: target 105 (shy 5) → reward 4 ≥ 2 → pass.
	if _, ok, reason := PHLPLHR2(tr, Level{Price: 110}, 0, 10, 0, cfg); !ok {
		t.Fatalf("EMA with room refused: %s", reason)
	}
}

// TestEMAVisitReclassifies — E3: per-visit references apply to the EMA too (the
// line moves, so a visit ends on one closed candle not touching it).
func TestEMAVisitReclassifies(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	lvl := Level{Key: string(KindEMA34), Kind: KindEMA34, Price: 100}
	var tr Touch

	prev := market.Kline{Close: 99}
	bar1 := market.Kline{Open: 99.4, High: 100.2, Low: 99.3, Close: 99.5}
	visitTick(&tr, lvl, prev, bar1, cfg)
	if tr.Outcome != TouchReject {
		t.Fatalf("visit 1 outcome = %q, want reject", tr.Outcome)
	}
	// non-touching candle ends the visit
	bar2 := market.Kline{Open: 99.5, High: 99.8, Low: 99.2, Close: 99.6}
	visitTick(&tr, lvl, bar1, bar2, cfg)
	if tr.Outcome != TouchNone {
		t.Fatalf("EMA visit must end on a non-touching candle, got %q", tr.Outcome)
	}
	// the next touching candle is the new reference
	bar3 := market.Kline{Open: 99.6, High: 100.1, Low: 99.5, Close: 99.4}
	visitTick(&tr, lvl, bar2, bar3, cfg)
	if tr.Outcome != TouchReject || tr.RefBar != bar3 {
		t.Fatalf("EMA visit 2 did not reclassify with the new reference")
	}
}

// TestEmaLossTick — E2: a stop-out before the fill = one loss at the line →
// blocked; a departure (a closed candle whose range does not touch the line)
// unblocks. A fill clears the pending stop without blocking.
func TestEmaLossTick(t *testing.T) {
	cfg := DefaultConfig()
	e := New(cfg)
	e.State.EmaPendingSide = SideLong
	e.State.EmaPendingEntry = 102
	e.State.EmaPendingStop = 98

	// filled first: candle trades through the entry — no loss.
	emaLossTick(e, 100, market.Kline{High: 103, Low: 99})
	if e.State.EmaBlocked || e.State.EmaPendingSide != "" {
		t.Fatalf("fill must clear the pending stop without blocking")
	}

	// pending again, then stopped out: the stop is hit while the entry (102)
	// never traded (high stays under it) — that is the loss.
	e.State.EmaPendingSide = SideLong
	e.State.EmaPendingEntry = 102
	e.State.EmaPendingStop = 98
	emaLossTick(e, 100, market.Kline{High: 100.9, Low: 97.5})
	if !e.State.EmaBlocked {
		t.Fatal("stop-out before fill must block the EMA line")
	}

	// still touching: stays blocked.
	emaLossTick(e, 100, market.Kline{High: 100.5, Low: 99})
	if !e.State.EmaBlocked {
		t.Fatal("a touching candle must not lift the block")
	}

	// departure: range entirely above the line.
	emaLossTick(e, 100, market.Kline{High: 102, Low: 100.5})
	if e.State.EmaBlocked {
		t.Fatal("departure must lift the EMA block")
	}
}

// TestEmaSetupAllowed — E2 block + E4 crossing knob on the seam.
func TestEmaSetupAllowed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EmaMaxCross30m = 4
	e := New(cfg)
	ema := Level{Key: string(KindEMA34), Kind: KindEMA34, Price: 100}

	// non-EMA levels are never gated here.
	if !emaSetupAllowed(e, keyLevel(100), nil, cfg) {
		t.Fatal("key levels must pass the EMA seam")
	}

	// crossing noise: 4 sign changes in the last 30 bars.
	bars := []market.Kline{
		{Close: 101}, {Close: 99}, {Close: 101}, {Close: 99}, {Close: 100.5},
	}
	if emaSetupAllowed(e, ema, bars, cfg) {
		t.Fatalf("cross count %d must refuse at knob 4", emaCross30(bars, 100))
	}

	// knob OFF: the same tape is allowed.
	off := cfg
	off.EmaMaxCross30m = 0
	if !emaSetupAllowed(e, ema, bars, off) {
		t.Fatal("knob OFF must allow the EMA setup")
	}

	// loss block wins regardless of the knob.
	e.State.EmaBlocked = true
	if emaSetupAllowed(e, ema, bars, off) {
		t.Fatal("a blocked EMA must refuse even with the knob off")
	}
}

// TestEmaCross30Window: only the last 30 candles count.
func TestEmaCross30Window(t *testing.T) {
	var bars []market.Kline
	for i := 0; i < 40; i++ {
		c := 100.0
		if i%2 == 1 {
			c = 99.0
		}
		bars = append(bars, market.Kline{Close: c})
	}
	if got := emaCross30(bars, 99.5); got != 30 {
		t.Fatalf("cross count = %d, want 30 (window cap)", got)
	}
}
