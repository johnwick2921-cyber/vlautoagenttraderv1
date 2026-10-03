package mentor

import (
	"testing"
	"time"

	"vl/kernel"
	"vl/market"
)

// §4.1 box tests. Geometry and helpers are synthetic; the golden test runs
// the real builder on the course frame's 13-Sep-2026 1m tape.

// TestHeikinAshiHandWorked — the 2025-extra TF option, hand-worked:
// HAclose = (O+H+L+C)/4; HAopen = (prev HAopen + prev HAclose)/2;
// HAhigh/HAlow take the extremes with them.
func TestHeikinAshiHandWorked(t *testing.T) {
	bars := []market.Kline{
		{Open: 10, High: 12, Low: 9, Close: 11},
		{Open: 11, High: 13, Low: 10, Close: 12},
	}
	ha := HeikinAshi(bars)
	// bar 1: close (10+12+9+11)/4 = 10.5; first open (10+11)/2 = 10.5
	if ha[0].Close != 10.5 || ha[0].Open != 10.5 || ha[0].High != 12 || ha[0].Low != 9 {
		t.Fatalf("HA[0] = %+v, want o=10.5 c=10.5 h=12 l=9", ha[0])
	}
	// bar 2: close (11+13+10+12)/4 = 11.5; open (10.5+10.5)/2 = 10.5
	if ha[1].Close != 11.5 || ha[1].Open != 10.5 || ha[1].High != 13 || ha[1].Low != 10 {
		t.Fatalf("HA[1] = %+v, want o=10.5 c=11.5 h=13 l=10", ha[1])
	}
}

// TestFTGHBoxFromPairGeometry — "râu ở cái đỉnh cao nhất, và cái BODY của
// cái đỉnh GẦN NHẤT" [Cách vẽ box @ 03:29].
func TestFTGHBoxFromPairGeometry(t *testing.T) {
	bars := []market.Kline{
		{Open: 96, High: 105, Low: 95, Close: 104},  // extreme high: wick 105
		{Open: 101, High: 103, Low: 99, Close: 102}, // nearest high: upper body 102
	}
	b := boxFromPair(bars, swingPairAt{kind: kernel.KindSWGH, price: 105, idx: 0}, swingPairAt{kind: kernel.KindSWGH, price: 103, idx: 1})
	if b == nil || b.Top != 105 || b.Bottom != 102 {
		t.Fatalf("FTGH = %+v, want [102, 105]", b)
	}
}

// TestFTGLBoxFromPairGeometry — "Mình lấy cái râu thấp nhất… của cái đáy
// trước đó… kéo lên tới… cái BODY của cái đáy GẦN NHẤT" [@ 02:16–02:41].
func TestFTGLBoxFromPairGeometry(t *testing.T) {
	bars := []market.Kline{
		{Open: 99, High: 101, Low: 92, Close: 94}, // extreme low: wick 92
		{Open: 95, High: 97, Low: 93, Close: 96},  // nearest low: lower body 95
	}
	b := boxFromPair(bars, swingPairAt{kind: kernel.KindSWGL, price: 92, idx: 0}, swingPairAt{kind: kernel.KindSWGL, price: 93, idx: 1})
	if b == nil || b.Bottom != 92 || b.Top != 95 {
		t.Fatalf("FTGL = %+v, want [92, 95]", b)
	}
}

// TestBoxEscapeDeletes — "ESCAPE = the candle's BODY outside (whole candle
// better)" [D3.2 p1 @ 19:05]; "Delete it once price escapes it" [D3.4 p2
// @ 12:11].
func TestBoxEscapeDeletes(t *testing.T) {
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	if escaped([]market.Kline{{Open: 103, High: 104.5, Low: 102.5, Close: 103, CloseTime: 1}}, b, -1) {
		t.Fatal("a candle inside the zone is not an escape")
	}
	if !escaped([]market.Kline{{Open: 106, High: 108, Low: 105.5, Close: 107, CloseTime: 1}}, b, -1) {
		t.Fatal("a candle whose whole body is above the FTGH must escape-delete it")
	}
	if escaped([]market.Kline{{Open: 103, High: 109, Low: 102.5, Close: 104, CloseTime: 1}}, b, -1) {
		t.Fatal("a wick beyond with the body inside must not delete the box")
	}
}

// TestBoxThirdTouchCounting — the 3rd touch is the TRADE [D3.2 p1 @ 04:29;
// D3.4 p3 @ 07:02]; the box is valid by construction [D3.2 p2 @ 03:45].
func TestBoxThirdTouchCounting(t *testing.T) {
	cfg := DefaultBoxCfg()
	b := Box{Kind: FTGH, Top: 105, Bottom: 102}
	bars := []market.Kline{
		{},
		{High: 105.1, Low: 103, Close: 103.5, CloseTime: 1},
		{High: 104, Low: 102.5, Close: 103, CloseTime: 2},
		{High: 105.2, Low: 103, Close: 104, CloseTime: 3},
		{High: 105.0, Low: 103, Close: 103.5, CloseTime: 4},
	}
	if n := countBoxTouches(bars, b, 0, cfg); n != 3 {
		t.Fatalf("touches = %d, want 3", n)
	}
}

// TestInsideAnyBoxBan — "NEVER trade inside the box — 'hoàn toàn không'"
// [D3.2 p1 @ 06:59].
func TestInsideAnyBoxBan(t *testing.T) {
	boxes := []Box{{Kind: FTGH, Top: 105, Bottom: 102}, {Kind: FTGL, Top: 95, Bottom: 92}}
	if !InsideAnyBox(boxes, 103.5) || !InsideAnyBox(boxes, 93) {
		t.Fatal("price inside either zone must be banned")
	}
	if InsideAnyBox(boxes, 100) {
		t.Fatal("price between the boxes is not inside either")
	}
}

// TestBoxEdgeLocationsExport — one location per edge for DS-103's location
// gate.
func TestBoxEdgeLocationsExport(t *testing.T) {
	loc := BoxEdgeLocations([]Box{{Kind: FTGH, Top: 105, Bottom: 102, Key: "ftgh:105:102"}})
	if len(loc) != 2 || loc[0].Kind != KindFTGHEdge || loc[1].Kind != KindFTGLEdge || loc[0].Price != 105 || loc[1].Price != 102 {
		t.Fatalf("locations = %+v, want ftgh_edge@105 + ftgl_edge@102", loc)
	}
}

// TestBoxesGolden13SepFrame — the course frame (D3.3 FTGH/FTGL
// part1_06-25.jpg, verified by the CTO): two 1m boxes on Sun 13 Sep 2026,
// FTGL ≈ 28,982 → 29,015 and FTGH ≈ 29,097 → 29,105. The builder must draw
// both (±1 pt) from the recorded db-copy tape. The 17:00 open candle is
// excluded (the frame's first candle is later).
func TestBoxesGolden13SepFrame(t *testing.T) {
	bars := loadFixture(t, "mnq_1m_2026-09-13_boxframe", "1m")
	start := bars[0].OpenTime + 3*60_000 // from 17:03 CT (rolling-3 needs 2 lead bars)
	trimmed := bars[3:]
	_ = start
	cfg := DefaultBoxCfg()
	now := time.UnixMilli(trimmed[len(trimmed)-1].OpenTime + 60_000)
	boxes := BoxesBuild(trimmed, cfg, now)
	var ftgl, ftgh *Box
	for i := range boxes {
		switch boxes[i].Kind {
		case FTGL:
			b := boxes[i]
			ftgl = &b
		case FTGH:
			b := boxes[i]
			ftgh = &b
		}
	}
	if ftgl == nil {
		t.Fatal("no FTGL box drawn on the golden frame")
	}
	if abs(ftgl.Bottom-28981) > 1 || abs(ftgl.Top-29015) > 1 {
		t.Fatalf("FTGL = [%.2f, %.2f], want ≈ [28981, 29015] ±1 (golden ~28,982 → 29,015)", ftgl.Bottom, ftgl.Top)
	}
	if ftgh == nil {
		t.Fatal("no FTGH box drawn on the golden frame")
	}
	if abs(ftgh.Top-29105) > 1 || abs(ftgh.Bottom-29097) > 1 {
		t.Fatalf("FTGH = [%.2f, %.2f], want ≈ [29097, 29105] ±1 (golden ~29,097 → 29,105)", ftgh.Bottom, ftgh.Top)
	}
	if len(boxes) > 2 {
		t.Fatalf("boxes = %d — 'Draw TWO zones, never a third' [D3.2 p2 @ 09:14]", len(boxes))
	}
}
