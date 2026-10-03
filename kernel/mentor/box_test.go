package mentor

import (
	"testing"
	"time"

	"vl/kernel"
	"vl/market"
)

// §4.1 box tests. Construction geometry and the pure helpers are synthetic
// (the window logic itself); TestBoxesOnRecordedRTHDay runs the real build
// on the bot's recorded tape.

// TestFTGHBoxFromPairGeometry — "FTGH: from the HIGHEST WICK of top 1 down to
// the NEAREST BODY of top 2" [D3.2 p1 @ 03:21–04:25].
func TestFTGHBoxFromPairGeometry(t *testing.T) {
	bars := []market.Kline{
		{Open: 96, High: 105, Low: 95, Close: 104},  // top 1: highest wick 105
		{Open: 101, High: 103, Low: 99, Close: 102}, // top 2: nearest body 102
	}
	b := boxFromPair(bars, swingPairAt{kind: kernel.KindSWGH, price: 105, idx: 0}, swingPairAt{kind: kernel.KindSWGH, price: 103, idx: 1})
	if b == nil {
		t.Fatal("two tops within tolerance must form an FTGH")
	}
	if b.Top != 105 || b.Bottom != 102 {
		t.Fatalf("FTGH = [%.2f, %.2f], want [102, 105] (nearest body of top 2 → highest wick of top 1)", b.Bottom, b.Top)
	}
	// a degenerate pair (top 2's body above top 1's wick) is no zone
	deg := boxFromPair(bars,
		swingPairAt{kind: kernel.KindSWGH, price: 102, idx: 1}, swingPairAt{kind: kernel.KindSWGH, price: 105, idx: 0})
	if deg != nil {
		t.Fatalf("degenerate FTGH pair must be nil, got %+v", deg)
	}
}

// TestFTGLBoxFromPairGeometry — "FTGL: from the LOWEST WICK of bottom 1 up to
// the NEAREST BODY of bottom 2" [D3.2 p1 @ 03:21–04:25].
func TestFTGLBoxFromPairGeometry(t *testing.T) {
	bars := []market.Kline{
		{Open: 99, High: 101, Low: 92, Close: 94}, // bottom 1: lowest wick 92
		{Open: 95, High: 97, Low: 93, Close: 96},  // bottom 2: nearest body 95
	}
	b := boxFromPair(bars, swingPairAt{kind: kernel.KindSWGL, price: 92, idx: 0}, swingPairAt{kind: kernel.KindSWGL, price: 93, idx: 1})
	if b == nil {
		t.Fatal("two bottoms within tolerance must form an FTGL")
	}
	if b.Top != 95 || b.Bottom != 92 {
		t.Fatalf("FTGL = [%.2f, %.2f], want [92, 95]", b.Bottom, b.Top)
	}
}

// TestBoxEscapeDeletes — "ESCAPE = the candle's BODY outside (whole candle
// better)" [D3.2 p1 @ 19:05]; "Delete it once price escapes it" [D3.4 p2
// @ 12:11].
func TestBoxEscapeDeletes(t *testing.T) {
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	stay := []market.Kline{{Open: 103, High: 104.5, Low: 102.5, Close: 103, CloseTime: 1}}
	if escaped(stay, b, -1) {
		t.Fatal("a candle inside the zone is not an escape")
	}
	bodyOut := []market.Kline{{Open: 106, High: 108, Low: 105.5, Close: 107, CloseTime: 1}}
	if !escaped(bodyOut, b, -1) {
		t.Fatal("a candle whose whole body is above the FTGH must escape-delete it")
	}
	// a wick beyond but the body inside is NOT the escape (body outside)
	wickOnly := []market.Kline{{Open: 103, High: 109, Low: 102.5, Close: 104, CloseTime: 1}}
	if escaped(wickOnly, b, -1) {
		t.Fatal("a wick beyond with the body inside must not delete the box")
	}
}

// TestBoxThirdTouchCounting — "The trade: price returns → stop order away
// from the box, with the REJECTING candle as the reference. Taken on the
// THIRD touch" [D3.2 p1 @ 04:29; D3.4 p3 @ 07:02]. The box is valid by
// construction [D3.2 p2 @ 03:45]; the counting the entry layer consumes is
// what this pins.
func TestBoxThirdTouchCounting(t *testing.T) {
	cfg := DefaultBoxCfg()
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	bars := []market.Kline{
		{}, // formation bar (index 0)
		{High: 105.1, Low: 103, Close: 103.5, CloseTime: 1}, // touch 1
		{High: 104, Low: 102.5, Close: 103, CloseTime: 2},   // no touch
		{High: 105.2, Low: 103, Close: 104, CloseTime: 3},   // touch 2
		{High: 105.0, Low: 103, Close: 103.5, CloseTime: 4}, // touch 3
	}
	if n := countBoxTouches(bars, b, 0, cfg); n != 3 {
		t.Fatalf("touches = %d, want 3 (the third touch is the trade)", n)
	}
}

// TestInsideAnyBoxBan — "NEVER trade inside the box — 'hoàn toàn không' —
// neither the candle nor your entry point" [D3.2 p1 @ 06:59].
func TestInsideAnyBoxBan(t *testing.T) {
	boxes := []Box{{Kind: FTGH, Top: 105, Bottom: 102}, {Kind: FTGL, Top: 95, Bottom: 92}}
	if !InsideAnyBox(boxes, 103.5) {
		t.Fatal("price inside the FTGH zone must be banned")
	}
	if !InsideAnyBox(boxes, 93) {
		t.Fatal("price inside the FTGL zone must be banned")
	}
	if InsideAnyBox(boxes, 100) {
		t.Fatal("price between the boxes is not inside either")
	}
}

// TestBoxEdgeLocationsExport — one location per edge for DS-103's location
// gate.
func TestBoxEdgeLocationsExport(t *testing.T) {
	boxes := []Box{{Kind: FTGH, Top: 105, Bottom: 102, Key: "ftgh:105:102"}}
	loc := BoxEdgeLocations(boxes)
	if len(loc) != 2 {
		t.Fatalf("locations = %d, want 2 (top + bottom edge)", len(loc))
	}
	if loc[0].Kind != KindFTGHEdge || loc[1].Kind != KindFTGLEdge {
		t.Fatalf("kinds = %q/%q, want ftgh_edge/ftgl_edge", loc[0].Kind, loc[1].Kind)
	}
	if loc[0].Price != 105 || loc[1].Price != 102 {
		t.Fatalf("edge prices = %.2f/%.2f, want 105/102", loc[0].Price, loc[1].Price)
	}
}

// TestBoxesOnRecordedRTHDay — canon 53: build boxes from the bot's recorded
// 1m tape. Whatever the detector finds must satisfy the invariants: zones
// are non-degenerate, at most MaxBoxes, formed from this trading day's bars,
// and the build is deterministic (same bars → same boxes, never redrawn).
func TestBoxesOnRecordedRTHDay(t *testing.T) {
	cfg := DefaultBoxCfg()
	bars := loadFixture(t, "mnq_1m_2026-09-15_rth", "1m")
	now := time.UnixMilli(bars[len(bars)-1].OpenTime + 60_000)
	a := BoxesBuild(bars, cfg, now)
	b := BoxesBuild(bars, cfg, now)
	if len(a) != len(b) {
		t.Fatalf("rebuild changed the box set: %d vs %d — a drawn box is never redrawn [D4.1 p2 @ 02:39]", len(a), len(b))
	}
	for i := range a {
		if a[i].Top <= a[i].Bottom {
			t.Fatalf("box %d degenerate: [%.2f, %.2f]", i, a[i].Bottom, a[i].Top)
		}
		if a[i].Top != b[i].Top || a[i].Bottom != b[i].Bottom || a[i].Kind != b[i].Kind {
			t.Fatalf("box %d differs across rebuilds", i)
		}
	}
	if len(a) > cfg.MaxBoxes {
		t.Fatalf("%d boxes > MaxBoxes %d — 'Draw TWO zones, never a third' [D3.2 p2 @ 09:14]", len(a), cfg.MaxBoxes)
	}
	t.Logf("recorded 2026-09-15 RTH: %d boxes (tol %.0f pts): %+v", len(a), cfg.TolPts, a)
}
