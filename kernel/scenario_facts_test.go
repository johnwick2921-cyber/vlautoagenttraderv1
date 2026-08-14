package kernel

import (
	"testing"

	"nofx/market"
)

const fiveMinMs = 5 * 60 * 1000

// series builds chronological 5m bars from [open,high,low,close] rows.
func series(rows [][4]float64) []market.Kline {
	bars := make([]market.Kline, len(rows))
	for i, r := range rows {
		open := int64(i) * fiveMinMs
		bars[i] = market.Kline{
			OpenTime:  open,
			Open:      r[0],
			High:      r[1],
			Low:       r[2],
			Close:     r[3],
			CloseTime: open + fiveMinMs - 1,
		}
	}
	return bars
}

func nowAfter(bars []market.Kline) int64 { return bars[len(bars)-1].CloseTime + 1 }

func TestClosesBeyondConsecutive(t *testing.T) {
	// closes: 99, 101, 102 vs level 100 → 2 consecutive above from newest.
	bars := series([][4]float64{{99, 99, 98, 99}, {100, 102, 100, 101}, {101, 103, 101, 102}})
	now := nowAfter(bars)
	if got := ClosesBeyond(bars, 100, DirAbove, now); got != 2 {
		t.Fatalf("closes above = %d want 2", got)
	}
	if got := ClosesBeyond(bars, 100, DirBelow, now); got != 0 {
		t.Fatalf("closes below = %d want 0", got)
	}
}

func TestAcceptance2x5m(t *testing.T) {
	accepted := series([][4]float64{{99, 99, 98, 99}, {100, 102, 100, 101}, {101, 103, 101, 102}})
	if ok, have, need := Acceptance(accepted, 100, DirAbove, "2x5m", nowAfter(accepted)); !ok || have != 2 || need != 2 {
		t.Fatalf("accepted 2x5m = %v have=%d need=%d", ok, have, need)
	}
	notYet := series([][4]float64{{99, 99, 98, 99}, {99, 100, 98, 99}, {101, 103, 101, 102}})
	if ok, have, _ := Acceptance(notYet, 100, DirAbove, "2x5m", nowAfter(notYet)); ok || have != 1 {
		t.Fatalf("single close should not accept: ok=%v have=%d", ok, have)
	}
}

func TestAcceptance15mClose(t *testing.T) {
	bars := series([][4]float64{{99, 99, 98, 99}, {101, 103, 101, 102}})
	if ok, have, need := Acceptance(bars, 100, DirAbove, "15m-close", nowAfter(bars)); !ok || have != 1 || need != 1 {
		t.Fatalf("15m-close single beyond = %v have=%d need=%d", ok, have, need)
	}
	// The unicode "2×5m" spelling must normalize to need=2.
	if need := acceptanceNeed("2×5m"); need != 2 {
		t.Fatalf("2×5m need = %d want 2", need)
	}
}

func TestSweptAbove(t *testing.T) {
	// last bar wicks to 101 (above 100) but closes 99 (back below) → sweep.
	bars := series([][4]float64{{98, 99, 97, 98}, {99, 101, 98, 99}})
	if !Swept(bars, 100, DirAbove, 3, nowAfter(bars)) {
		t.Fatalf("expected sweep above")
	}
	// A clean acceptance (close above) is NOT a sweep.
	clean := series([][4]float64{{98, 99, 97, 98}, {99, 101, 98, 101}})
	if Swept(clean, 100, DirAbove, 3, nowAfter(clean)) {
		t.Fatalf("close-above is not a sweep")
	}
}

func TestSweptBelow(t *testing.T) {
	bars := series([][4]float64{{102, 103, 101, 102}, {101, 102, 99, 101}})
	if !Swept(bars, 100, DirBelow, 3, nowAfter(bars)) {
		t.Fatalf("expected sweep below")
	}
}

func TestReclaimedAbove(t *testing.T) {
	// close below (98), then close above (101) → reclaimed above.
	bars := series([][4]float64{{99, 99, 97, 98}, {99, 102, 98, 101}})
	if !Reclaimed(bars, 100, DirAbove, 3, nowAfter(bars)) {
		t.Fatalf("expected reclaim above")
	}
	// Never dipped below → not a reclaim (no opposite-side close in lookback).
	noCross := series([][4]float64{{101, 102, 100, 101}, {101, 103, 101, 102}})
	if Reclaimed(noCross, 100, DirAbove, 3, nowAfter(noCross)) {
		t.Fatalf("no opposite-side close should not reclaim")
	}
}

func TestRejectedSupportAndResistance(t *testing.T) {
	// Support held: low pierced to 99, closed 101 above → rejected(+1).
	sup := series([][4]float64{{101, 102, 99, 101}})
	if !Rejected(sup, 100, DirAbove, 3, nowAfter(sup)) {
		t.Fatalf("expected support-held rejection")
	}
	// Resistance held: high pierced to 101, closed 99 below → rejected(-1).
	res := series([][4]float64{{99, 101, 98, 99}})
	if !Rejected(res, 100, DirBelow, 3, nowAfter(res)) {
		t.Fatalf("expected resistance-held rejection")
	}
}

func TestLevelStillValid(t *testing.T) {
	// One close above → still valid (not yet accepted through).
	oneAbove := series([][4]float64{{99, 99, 98, 99}, {99, 100, 98, 99}, {101, 103, 101, 102}})
	if !LevelStillValid(oneAbove, 100, "2x5m", nowAfter(oneAbove)) {
		t.Fatalf("single close beyond should keep level valid")
	}
	// Two consecutive closes above → consumed → invalid.
	twoAbove := series([][4]float64{{99, 99, 98, 99}, {100, 102, 100, 101}, {101, 103, 101, 102}})
	if LevelStillValid(twoAbove, 100, "2x5m", nowAfter(twoAbove)) {
		t.Fatalf("two closes beyond should consume the level")
	}
}

func TestDistance(t *testing.T) {
	if d := SignedDistancePoints(101, 100); d != 1 {
		t.Fatalf("distance = %v want 1", d)
	}
	if tk := DistanceTicks(101, 100, 0.25); tk != 4 {
		t.Fatalf("ticks = %v want 4", tk)
	}
	if tk := DistanceTicks(101, 100, 0); tk != 0 {
		t.Fatalf("tick<=0 must return 0, got %v", tk)
	}
}

func TestOpenBarSkipped(t *testing.T) {
	// Newest bar closes 105 (above) but is NOT yet closed; the closed bar below
	// it (99) breaks the run → 0 consecutive closes above.
	bars := series([][4]float64{{99, 99, 98, 99}, {104, 106, 104, 105}})
	// nowMs falls BEFORE the last bar's CloseTime → last bar is open.
	nowMid := bars[1].OpenTime + 1
	if got := ClosesBeyond(bars, 100, DirAbove, nowMid); got != 0 {
		t.Fatalf("open bar must be skipped; got %d want 0", got)
	}
	// The closed bar's close IS visible as latest closed close.
	if lc, ok := latestClosedClose(bars, nowMid); !ok || lc != 99 {
		t.Fatalf("latest closed close = %v ok=%v want 99", lc, ok)
	}
}

func TestEvaluateLevelFactsAggregate(t *testing.T) {
	// A sweep-then-reclaim-then-accept sequence around level 100:
	//  1) close 98 (below)
	//  2) wick 101, close 99  (sweep above, still below)
	//  3) close 101           (reclaim above)
	//  4) low 100, close 102  (2nd close above → accepted; support tap)
	bars := series([][4]float64{
		{98, 99, 97, 98},
		{98, 101, 98, 99},
		{99, 102, 99, 101},
		{101, 103, 100, 102},
	})
	f := EvaluateLevelFacts(bars, 100, DirAbove, "2x5m", 3, nowAfter(bars))

	if f.LatestClose != 102 || f.DistancePoints != 2 {
		t.Fatalf("close/distance = %v/%v want 102/2", f.LatestClose, f.DistancePoints)
	}
	if f.ClosesBeyondUp != 2 || f.ClosesBeyondDown != 0 {
		t.Fatalf("closes up/down = %d/%d want 2/0", f.ClosesBeyondUp, f.ClosesBeyondDown)
	}
	if !f.Swept {
		t.Fatalf("expected sweep in lookback (bar 2 wick 101 / close 99)")
	}
	if !f.Reclaimed {
		t.Fatalf("expected reclaim (dipped below then closed above)")
	}
	if !f.Rejected {
		t.Fatalf("expected support-held (bar 4 low 100)")
	}
	if !f.Accepted || f.AcceptHave != 2 || f.AcceptNeed != 2 {
		t.Fatalf("expected acceptance 2/2, got %v %d/%d", f.Accepted, f.AcceptHave, f.AcceptNeed)
	}
	if f.StillValid {
		t.Fatalf("accepted-through level must be consumed (not still valid)")
	}
}
