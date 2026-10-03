package mentor

import (
	"strings"
	"testing"
	"time"

	"vl/market"
)

// Call-site tests for the box trade wire: boxEntryIntent (reject/cancel/
// filters/confluence) and the Tick loop (R1: every return trades, each
// exactly once).

func TestBoxEntryIntentRejectPlacesStopOrder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.RoomMultiple = 0.5
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	ref := market.Kline{High: 104.5, Low: 100.5, Close: 101} // touch + close below = reject
	levels := []Level{{Kind: KindKeyLevel, Price: 98}}
	out := boxEntryIntent(ref, b, []Box{b}, levels, TriggerLine{}, cfg)
	if len(out) != 1 {
		t.Fatalf("reject return = %d intents, want 1", len(out))
	}
	in := out[0]
	if in.Action != PlaceStopEntry || in.Side != SideShort || in.Price != 100.5 || in.Stop != 104.5 || in.Target != 98 {
		t.Fatalf("reject intent = %+v, want SHORT entry 100.5 stop 104.5 target 98", in)
	}
	if in.Confluence {
		t.Fatal("no key level at the box and no trigger — confluence must be off")
	}
	if in.Reason == "" {
		t.Fatal("intent without a reason")
	}
}

func TestBoxEntryIntentInsideCloseCancels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	ref := market.Kline{High: 105.1, Low: 103, Close: 104} // touch, close inside = cancel [D3.2 p1 @ 21:04–21:33]
	if out := boxEntryIntent(ref, b, []Box{b}, nil, TriggerLine{}, cfg); len(out) != 0 {
		t.Fatalf("inside close = %d intents, want 0 (cancel)", len(out))
	}
}

func TestBoxEntryIntentTriggerBlock(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	ref := market.Kline{High: 104.5, Low: 100.5, Close: 101}
	levels := []Level{{Kind: KindKeyLevel, Price: 98}}
	trig := TriggerLine{Dir: SideLong, Price: 106} // short entry below the buy line → blocked
	if out := boxEntryIntent(ref, b, []Box{b}, levels, trig, cfg); len(out) != 0 {
		t.Fatalf("wrong-side trigger = %d intents, want 0", len(out))
	}
}

// R2 [00-METHOD Risk-reward, D3.4 p3 @ 07:38]: box edge + key level inside
// the box or within 2 pts of its edge + the 5m trigger agrees → the flag
// rides the intent for DS-102's exit-C / size-10.
func TestBoxEntryIntentConfluenceFlag(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.RoomMultiple = 0.3
	b := Box{Kind: FTGL, Top: 96, Bottom: 94}
	ref := market.Kline{High: 97.5, Low: 96.5, Close: 97} // reject for the long
	levels := []Level{
		{Kind: KindKeyLevel, Price: 97.9}, // inside the box + 2-pt band → confluence
		{Kind: KindKeyLevel, Price: 99.5}, // target ladder
	}
	trig := TriggerLine{Dir: SideLong, Price: 95}
	out := boxEntryIntent(ref, b, []Box{b}, levels, trig, cfg)
	if len(out) != 1 {
		t.Fatalf("reject return = %d intents, want 1", len(out))
	}
	if !out[0].Confluence || out[0].Side != SideLong {
		t.Fatalf("intent = %+v, want LONG with Confluence=true", out[0])
	}

	// The same setup without the trigger line: confluence stays off.
	if o := boxEntryIntent(ref, b, []Box{b}, levels, TriggerLine{}, cfg); len(o) != 1 || o[0].Confluence {
		t.Fatalf("no trigger line can never agree — got %+v", o)
	}

	// Key level far from the box: confluence off.
	far := []Level{{Kind: KindKeyLevel, Price: 99.5}}
	if o := boxEntryIntent(ref, b, []Box{b}, far, trig, cfg); len(o) != 1 || o[0].Confluence {
		t.Fatalf("key level outside the 2-pt band must not confluence — got %+v", o)
	}
}

// R1 at the CALL SITE [D3.2 p2 @ 06:25]: Tick evaluates EVERY return visit
// of a live box — two reject returns in the day emit two box intents — and
// each return exactly once (BoxRefs dedup).
func TestEvaluatorBoxPathEveryReturnTrades(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.KeyLevelTFMinutes = 1
	cfg.EMAPeriod34 = 0
	cfg.EMAPeriod9 = 0
	cfg.RoomMultiple = 0.05
	t0 := time.Date(2026, time.September, 15, 9, 0, 0, 0, ctime()).UnixMilli()
	mk := func(i int, o, h, l, c float64) market.Kline {
		return market.Kline{OpenTime: t0 + int64(i)*60_000, CloseTime: t0 + int64(i)*60_000 + 59_000, Open: o, High: h, Low: l, Close: c}
	}
	bars := []market.Kline{
		mk(0, 100, 101, 99, 100),
		mk(1, 100, 102, 99, 101),
		mk(2, 101, 103, 95, 96), // swing low @2
		mk(3, 97.3, 98.3, 96.5, 97.5),
		mk(4, 97, 98, 94, 95), // swing low @4 — the extreme; FTGL [94, 96]
		mk(5, 98, 98.2, 96.5, 97.2),
		mk(6, 96.5, 97.1, 95.9, 97),   // REJECT return 1 (literal touch: low ≤ 96, close above 96)
		mk(7, 97.7, 97.9, 95.9, 96.9), // REJECT return 2
		mk(8, 98.2, 98.5, 97, 98.4),
	}
	e := New(cfg)
	now := bars[8].OpenTime + 59_999
	// ORB preset (the §7 gate is drawn+escaped long; the tape alone never
	// draws an ORB and the gate would refuse every intraday entry).
	e.State.ORB = ORB{Day: dayStartCT(now), High: 90, Low: 85, Drawn: true, Escaped: SideLong}
	boxIntents := func(ins []Intent) []Intent {
		var out []Intent
		for _, in := range ins {
			if strings.HasPrefix(in.Reason, "box edge return") {
				out = append(out, in)
			}
		}
		return out
	}
	first := boxIntents(e.Tick(bars[:9], now))
	if len(first) != 2 {
		t.Fatalf("tick 1 box intents = %d (%+v), want 2 — every return trades [D3.2 p2 @ 06:25]", len(first), first)
	}
	if first[0].Price != 97.1 || first[0].Side != SideLong || first[0].Stop != 95.9 || first[0].Target != 98.2 {
		t.Fatalf("return 1 = %+v, want LONG 97.1 / stop 95.9 / target 98.2", first[0])
	}
	if first[1].Price != 97.9 {
		t.Fatalf("return 2 = %+v, want entry 97.9", first[1])
	}
	// Next tick: no new candle — the same returns must not re-emit.
	bars2 := append(bars, mk(9, 98.5, 98.8, 97.5, 98.6))
	second := boxIntents(e.Tick(bars2[:10], bars2[9].OpenTime+59_999))
	if len(second) != 0 {
		t.Fatalf("tick 2 box intents = %d (%+v), want 0 — a return is evaluated once", len(second), second)
	}
}

// TestBoxReturnExactlyOneEntry — BOX PATH DECISION (CTO 12:38:50Z): one box
// return visit yields EXACTLY ONE entry intent — the box path's. The box
// edges are OUT of the level touch loop, so the level path must not add a
// second entry for the same return. Mutant (edges back in the loop) → 2 != 1.
func TestBoxReturnExactlyOneEntry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.KeyLevelTFMinutes = 1
	cfg.EMAPeriod34 = 0
	cfg.EMAPeriod9 = 0
	cfg.RoomMultiple = 0.05
	t0 := time.Date(2026, time.September, 15, 9, 0, 0, 0, ctime()).UnixMilli()
	mk := func(i int, o, h, l, c float64) market.Kline {
		return market.Kline{OpenTime: t0 + int64(i)*60_000, CloseTime: t0 + int64(i)*60_000 + 59_000, Open: o, High: h, Low: l, Close: c}
	}
	bars := []market.Kline{
		mk(0, 100, 101, 99, 100), mk(1, 100, 102, 99, 101), mk(2, 101, 103, 95, 96),
		mk(3, 97.3, 98.3, 96.5, 97.5), mk(4, 97, 98, 94, 95), mk(5, 98, 98.2, 96.5, 97.2),
		mk(6, 97.5, 98, 96.7, 97.6), // spacer: the return sits >=3 candles from the extreme
		mk(7, 96.5, 97.1, 95.9, 97), // the ONE reject return (touches 96, closes above)
	}
	e := New(cfg)
	now := bars[7].OpenTime + 59_999
	e.State.ORB = ORB{Day: dayStartCT(now), High: 90, Low: 85, Drawn: true, Escaped: SideLong}
	// Presets so the LEVEL path could emit a PHL from the edge too (under the
	// mutant): trigger long, 4h long (1h silent), day measured OK.
	e.State.Trigger = TriggerLine{Dir: SideLong, Price: 93}
	e.State.HTF = HTF{FourH: TriggerLine{Dir: SideLong, Price: 93}}
	e.State.Day = DayLatch{Verdict: DayTrade}

	ins := e.Tick(bars[:8], now)
	entries := 0
	for _, in := range ins {
		if in.Action == PlaceStopEntry {
			entries++
		}
	}
	if entries != 1 {
		t.Fatalf("one box return yielded %d entries, want EXACTLY ONE: %+v", entries, ins)
	}
	// The box edges must NEVER be classified by the level touch loop — if
	// they were, the same return would trade twice (box path + level path).
	for _, lvl := range BoxEdgeLocations(BoxesBuild(bars, cfg.Box, time.UnixMilli(now))) {
		if tr, ok := e.State.Touches[lvl.Key]; ok && tr.Outcome != TouchNone {
			t.Fatalf("box edge %s was touched as a LEVEL (outcome %q) — the edge must be out of the level touch loop", lvl.Key, tr.Outcome)
		}
	}

	if ins[0].Reason != "" && !strings.HasPrefix(ins[0].Reason, "box edge return") && entries > 0 {
		// the single entry must be the box path's
		var boxOne bool
		for _, in := range ins {
			if in.Action == PlaceStopEntry && strings.HasPrefix(in.Reason, "box edge return") {
				boxOne = true
			}
		}
		if !boxOne {
			t.Fatalf("the single entry is not the box path's: %+v", ins)
		}
	}
}
