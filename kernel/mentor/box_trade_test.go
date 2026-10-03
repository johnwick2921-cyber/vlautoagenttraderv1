package mentor

import (
	"testing"

	"vl/market"
)

// R1 [D3.2 p2 @ 06:25]: every return trades, not only the first. Each NEW
// visit — price outside the box on the approach side, then a touch — gets
// its own reference candle. The first return is the third touch overall
// [D3.2 p1 @ 04:29].
func TestBoxReturnBarsEveryVisit(t *testing.T) {
	cfg := DefaultBoxCfg()
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	bars := []market.Kline{
		{}, // formedAt = 0
		{High: 105.1, Low: 102.5, Close: 104, CloseTime: 1}, // touch, no outside first → not a visit
		{High: 101, Low: 99, Close: 100.5, CloseTime: 2},     // close below 102 → outside
		{High: 105.0, Low: 100, Close: 104, CloseTime: 3},    // visit 1 — reference bar 3
		{High: 105.2, Low: 103, Close: 104, CloseTime: 4},    // same visit, not a new reference
		{High: 100, Low: 98.5, Close: 99.5, CloseTime: 5},    // outside again
		{High: 105.1, Low: 99, Close: 104.5, CloseTime: 6},   // visit 2 — reference bar 6
	}
	rs := BoxReturnBars(bars, b, 0, cfg)
	if len(rs) != 2 {
		t.Fatalf("returns = %d, want 2 (every return trades, per visit)", len(rs))
	}
	if rs[0].N != 1 || rs[0].RefBar != 3 {
		t.Fatalf("return 1 = %+v, want N=1 RefBar=3", rs[0])
	}
	if rs[1].N != 2 || rs[1].RefBar != 6 {
		t.Fatalf("return 2 = %+v, want N=2 RefBar=6 — later returns must not be skipped [D3.2 p2 @ 06:25]", rs[1])
	}
}

// R2 [00-METHOD Risk-reward, D3.4 p3 @ 07:38]: confluence = box edge + key
// level inside the box or within 2 pts of its edge + the 5m trigger agrees.
// LONG = FTGL (support); SHORT = FTGH. Feeds DS-102's exit-C / size-10.
func TestConfluenceVerdict(t *testing.T) {
	b := Box{Kind: FTGL, Top: 29020, Bottom: 29000} // support box
	kls := []Level{{Kind: KindKeyLevel, Price: 28998}} // 2 pts below the edge → "at"
	buy := TriggerLine{Dir: SideLong, Price: 28990}    // buy line under the entry → agrees
	sell := TriggerLine{Dir: SideShort, Price: 29100}

	if f := ConfluenceVerdict(b, SideLong, kls, buy); !f.On || f.Side != SideLong {
		t.Fatalf("FTGL + key level 2 pts off the edge + buy trigger = %+v, want On Long", f)
	}
	if f := ConfluenceVerdict(b, SideLong, kls, sell); f.On {
		t.Fatal("sell trigger must not confluence a long")
	}
	if f := ConfluenceVerdict(b, SideLong, kls, TriggerLine{}); f.On {
		t.Fatal("no trigger line can never agree — fail closed")
	}
	far := []Level{{Kind: KindKeyLevel, Price: 28950}}
	if f := ConfluenceVerdict(b, SideLong, far, buy); f.On {
		t.Fatal("key level 50 pts away is not 'at' the box")
	}
	inside := []Level{{Kind: KindKeyLevel, Price: 29010}}
	if f := ConfluenceVerdict(b, SideLong, inside, buy); !f.On {
		t.Fatal("a key level INSIDE the box must confluence")
	}
	// SHORT mirror: FTGH + key level + sell trigger.
	c := Box{Kind: FTGH, Top: 29100, Bottom: 29080}
	skl := []Level{{Kind: KindKeyLevel, Price: 29102}} // 2 pts above the edge
	if f := ConfluenceVerdict(c, SideShort, skl, sell); !f.On || f.Side != SideShort {
		t.Fatalf("FTGH + key level 2 pts off the edge + sell trigger = %+v, want On Short", f)
	}
	if f := ConfluenceVerdict(c, SideLong, skl, buy); f.On {
		t.Fatal("a long against an FTGH must not confluence")
	}
	// Wrong box kind for the side.
	if f := ConfluenceVerdict(b, SideShort, skl, sell); f.On {
		t.Fatal("SHORT needs an FTGH, not an FTGL")
	}
}
