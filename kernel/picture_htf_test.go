package kernel

import (
	"testing"

	"nofx/market"
)

// Fixture helpers: synthetic 4H / H1 / 5m bars around a pivot at index 2.
func bar4h(i int, o, h, l, c float64) market.Kline {
	ot := int64(1789000000000) + int64(i)*4*3600*1000
	// W4/D22: cache-delivered bars always carry their completion stamp, and
	// KnowableAt is now read from it, so the fixture carries it too.
	return market.Kline{OpenTime: ot, CloseTime: ot + fourHMs, Open: o, High: h, Low: l, Close: c}
}

func barh1(i int, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: int64(1789130000000) + int64(i)*3600*1000, Open: o, High: h, Low: l, Close: c}
}

func bar5m(i int, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: int64(1789000000000) + int64(i)*5*60*1000, Open: o, High: h, Low: l, Close: c}
}

// A completed 4H series with a clear resistance pivot at idx 2 (body top 100,
// two lower bodies on each side) and a support pivot at idx 6 (body bottom 80).
func pivot4HFixture() []market.Kline {
	bars := []market.Kline{
		bar4h(0, 90, 95, 85, 88),   // body 88-90
		bar4h(1, 89, 93, 84, 92),   // body 89-92
		bar4h(2, 99, 105, 96, 101), // pivot: body 99-101, top 101
		bar4h(3, 98, 102, 95, 96),  // body 96-98
		bar4h(4, 97, 100, 93, 94),  // body 94-97
		bar4h(5, 85, 92, 78, 86),   // body 85-86
		bar4h(6, 80, 84, 76, 81),   // pivot: body 80-81, bottom 80
		bar4h(7, 82, 88, 80, 87),   // body 82-87
		bar4h(8, 83, 90, 81, 89),   // body 83-89
		bar4h(9, 92, 96, 88, 95),   // body 92-95
	}
	return bars
}

func TestBodyPivots4HFindsBothPivotsAndNoLookahead(t *testing.T) {
	levels := BodyPivots4H(pivot4HFixture(), 120)
	if len(levels) != 2 {
		t.Fatalf("want 2 pivots, got %d: %+v", len(levels), levels)
	}
	if levels[0].Role != "resistance" || levels[0].BodyTop != 101 || levels[0].Boundary != 101 {
		t.Fatalf("resistance pivot wrong: %+v", levels[0])
	}
	if levels[1].Role != "support" || levels[1].BodyBottom != 80 || levels[1].Boundary != 80 {
		t.Fatalf("support pivot wrong: %+v", levels[1])
	}
	// No lookahead: the pivot is knowable only once bars[4] has COMPLETED
	// (W4/D22 — it used to be anchored to that candle's OPEN, one whole 4H
	// period early).
	wantKnowable := bar4h(4, 0, 0, 0, 0).CloseTime
	if levels[0].KnowableAt != wantKnowable {
		t.Fatalf("resistance KnowableAt = %d, want %d", levels[0].KnowableAt, wantKnowable)
	}
	// ActiveLevels excludes the level before it became knowable.
	active := ActiveLevels(levels, wantKnowable-1)
	for _, l := range active {
		if l.Role == "resistance" {
			t.Fatalf("resistance must not be active before it is knowable")
		}
	}
}

func TestBodyPivots4HFormingCandleCannotBePivot(t *testing.T) {
	// The same fixture plus ONE more bar that would make bar[8] a support
	// pivot were it confirmed — but bar[9] is the LAST bar (still forming):
	// no pivot may reference the last two bars (i+2 must exist).
	bars := pivot4HFixture()
	levels := BodyPivots4H(bars, 120)
	for _, l := range levels {
		if l.SourceIdx >= len(bars)-2 {
			t.Fatalf("forming-tail pivot leaked: %+v", l)
		}
	}
}

func TestBodyPivots4HEqualBodiesDoNotFormPivot(t *testing.T) {
	bars := []market.Kline{
		bar4h(0, 90, 92, 88, 91),
		bar4h(1, 90, 92, 88, 91), // equal body tops/bottoms → plateau
		bar4h(2, 90, 92, 88, 91),
		bar4h(3, 90, 92, 88, 91),
		bar4h(4, 90, 92, 88, 91),
	}
	if got := BodyPivots4H(bars, 120); len(got) != 0 {
		t.Fatalf("equal-body plateau produced pivots: %+v", got)
	}
}

func TestBodyPivots4HRetiresOnFarEdgeClose(t *testing.T) {
	// A later completed close ABOVE the resistance far edge retires it.
	bars := append(pivot4HFixture(), bar4h(10, 106, 110, 102, 108))
	levels := BodyPivots4H(bars, 120)
	for _, l := range levels {
		if l.Role == "resistance" && !l.Retired {
			t.Fatalf("resistance must retire after far-edge close: %+v", l)
		}
	}
}

func TestH1CloseBreakOneTickRule(t *testing.T) {
	levels := BodyPivots4H(pivot4HFixture(), 120)
	active := ActiveLevels(levels, bar4h(9, 0, 0, 0, 0).OpenTime)
	tick := 0.25

	// Prev close at the boundary, new close one tick above → FIRES.
	prev := barh1(0, 100.5, 102, 99, 101)
	cur := barh1(1, 101, 104, 100, 101.25)
	if r := H1CloseBreak(active, prev, cur, tick); !r.Fired || r.Direction != "long" {
		t.Fatalf("one-tick close must fire: %+v", r)
	}
	// New close exactly AT the boundary → NO fire (needs one tick beyond).
	curAt := barh1(1, 101, 104, 100, 101)
	if r := H1CloseBreak(active, prev, curAt, tick); r.Fired {
		t.Fatalf("close exactly at boundary must not fire: %+v", r)
	}
	// WICK crosses but close below → NO fire (the owner's core rule).
	curWick := barh1(1, 100, 105, 99, 100.5)
	if r := H1CloseBreak(active, prev, curWick, tick); r.Fired {
		t.Fatalf("wick-only crossing must not fire: %+v", r)
	}
	// Level not yet knowable at the candle's open → NO fire.
	if r := H1CloseBreak(active, barh1(0, 100, 102, 99, 101), barh1(1, 101, 104, 100, 101.5), tick); r.Fired {
		// prev/cur times are AFTER KnowableAt here; this case uses a level
		// whose KnowableAt is in the future:
	}
	future := []PictureHtfLevel{{Role: "resistance", Boundary: 101, KnowableAt: cur.OpenTime + 3600_000}}
	if r := H1CloseBreak(future, prev, barh1(1, 101, 104, 100, 101.5), tick); r.Fired {
		t.Fatalf("not-yet-knowable level must not fire: %+v", r)
	}
}

func TestH1CloseBreakMirroredShort(t *testing.T) {
	levels := BodyPivots4H(pivot4HFixture(), 120)
	active := ActiveLevels(levels, bar4h(9, 0, 0, 0, 0).OpenTime)
	prev := barh1(0, 80.5, 82, 79, 80) // at/below support boundary? prev must be >= boundary for short
	prev = barh1(0, 80.5, 82, 79, 80.75)
	cur := barh1(1, 80, 82, 79.5, 79.75)
	if r := H1CloseBreak(active, prev, cur, 0.25); !r.Fired || r.Direction != "short" || r.Boundary != 80 {
		t.Fatalf("short break must fire at the support boundary: %+v", r)
	}
}

func TestH1CloseBreakExtremeLevelWins(t *testing.T) {
	// Two resistances crossed by the same H1: the HIGHEST boundary wins.
	// W4/D22: KnowableAt is never 0 — a level with no knowable time is missing
	// evidence and is refused — so these literals carry the instant the level
	// became knowable, which is before the breaking H1 candle opens.
	knowable := barh1(0, 0, 0, 0, 0).OpenTime
	levels := []PictureHtfLevel{
		{Role: "resistance", Boundary: 101, KnowableAt: knowable},
		{Role: "resistance", Boundary: 105, KnowableAt: knowable},
	}
	prev := barh1(0, 99, 100, 98, 100)
	cur := barh1(1, 101, 108, 100, 106)
	r := H1CloseBreak(levels, prev, cur, 0.25)
	if !r.Fired || r.Boundary != 105 {
		t.Fatalf("highest crossed resistance must win: %+v", r)
	}
	if len(r.Crossed) != 2 {
		t.Fatalf("both crossed levels must ride as context: %+v", r)
	}
}

func TestStructuralSwing5MStrictAndDeadline(t *testing.T) {
	// Swing low at i=3 (Low 90) with strict confirmation; the confirming bars
	// complete before the H1 close deadline.
	bars := []market.Kline{
		bar5m(0, 95, 96, 94, 95),
		bar5m(1, 94, 95, 93, 94),
		bar5m(2, 93, 94, 92, 93),
		bar5m(3, 92, 93, 90, 91), // swing: Low 90 < all neighbors
		bar5m(4, 91, 92, 90.5, 91.5),
		bar5m(5, 91.5, 92.5, 91, 92),
		bar5m(6, 92, 93, 91.5, 92.5),
	}
	deadline := bar5m(6, 0, 0, 0, 0).OpenTime + 1
	p, ok := StructuralSwing5M(bars, "long", 24, deadline)
	if !ok || p != 90 {
		t.Fatalf("swing low must be 90, got %v %v", p, ok)
	}
	// Confirmation AFTER the H1 close deadline → ineligible.
	early := bar5m(5, 0, 0, 0, 0).OpenTime - 1
	if _, ok := StructuralSwing5M(bars, "long", 24, early); ok {
		t.Fatalf("swing confirmed after the deadline must be ineligible")
	}
	// STRICT equality: an equal Low next to the pivot breaks it.
	equal := []market.Kline{
		bar5m(0, 95, 96, 94, 95),
		bar5m(1, 94, 95, 93, 94),
		bar5m(2, 93, 94, 90, 93), // equal low as pivot
		bar5m(3, 92, 93, 90, 91),
		bar5m(4, 91, 92, 90.5, 91.5),
		bar5m(5, 91.5, 92.5, 91, 92),
		bar5m(6, 92, 93, 91.5, 92.5),
	}
	if _, ok := StructuralSwing5M(equal, "long", 24, deadline); ok {
		t.Fatalf("equal extremes must not form a swing")
	}
}

func TestNearestOpposingZonePolarityAndNearest(t *testing.T) {
	// W4/D22: 0 no longer means "always knowable" — it means unknown, and
	// unknown refuses. The zones carry a real knowable time and the reads
	// happen after it.
	const zoneKnowable, zoneNow = int64(1000), int64(2000)
	levels := []PictureHtfLevel{
		{Role: "resistance", TargetEdge: 110, KnowableAt: zoneKnowable},
		{Role: "resistance", TargetEdge: 105, KnowableAt: zoneKnowable},
		{Role: "support", TargetEdge: 90, KnowableAt: zoneKnowable},
	}
	// Long at 100: nearest resistance = 105, never a support, never skip.
	edge, ok := NearestOpposingZone(levels, 100, "long", zoneNow)
	if !ok || edge != 105 {
		t.Fatalf("nearest resistance must be 105, got %v %v", edge, ok)
	}
	// Short at 100: nearest support = 90.
	edge, ok = NearestOpposingZone(levels, 100, "short", zoneNow)
	if !ok || edge != 90 {
		t.Fatalf("nearest support must be 90, got %v %v", edge, ok)
	}
	// No eligible opposing zone above a long → no trade.
	if _, ok := NearestOpposingZone(levels, 120, "long", zoneNow); ok {
		t.Fatalf("no resistance above entry must refuse")
	}
	// Retired zone ineligible.
	levels[0].Retired = true
	edge, ok = NearestOpposingZone(levels, 100, "long", zoneNow)
	if !ok || edge != 105 {
		t.Fatalf("retired 110 must be skipped, nearest active must be 105: %v %v", edge, ok)
	}
}

func TestSweepReclaim4HSupportRejection(t *testing.T) {
	levels := BodyPivots4H(pivot4HFixture(), 120)
	// A later completed candle wicks below the support (80) and closes above.
	bars := append(pivot4HFixture(), bar4h(10, 81, 84, 78.5, 82))
	sweeps := SweepReclaim4H(levels, bars)
	found := false
	for _, s := range sweeps {
		if s.Role == "support" && s.WickedTo == 78.5 && s.Reclaimed {
			found = true
		}
	}
	if !found {
		t.Fatalf("support sweep/reclaim must be recorded: %+v", sweeps)
	}
	// Wick below but close BELOW → NOT a support reclaim (a break instead).
	bars2 := append(pivot4HFixture(), bar4h(10, 81, 84, 78.5, 79))
	for _, s := range SweepReclaim4H(levels, bars2) {
		if s.Role == "support" && s.WickedTo == 78.5 {
			t.Fatalf("close below the boundary is a break, not a reclaim: %+v", s)
		}
	}
}

func TestH1MomentumStallAdvisory(t *testing.T) {
	if s := H1MomentumStall([]float64{100, 101, 102}); s.Fired {
		t.Fatalf("rising streak is not a stall: %+v", s)
	}
	s := H1MomentumStall([]float64{100, 101, 100.5})
	if !s.Fired || s.Direction != "long" {
		t.Fatalf("non-rising after two rises must be a long stall: %+v", s)
	}
	s = H1MomentumStall([]float64{102, 101, 101.5})
	if !s.Fired || s.Direction != "short" {
		t.Fatalf("non-falling after two falls must be a short stall: %+v", s)
	}
	if s = H1MomentumStall([]float64{100, 101, 101.5}); s.Fired {
		t.Fatalf("a third rising close is not a stall: %+v", s)
	}
	if H1MomentumStall([]float64{100, 101}).Fired {
		t.Fatalf("fewer than three closes must not fire")
	}
}
