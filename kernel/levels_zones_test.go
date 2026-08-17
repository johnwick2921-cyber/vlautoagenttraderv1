package kernel

import (
	"testing"
	"time"
)

func firstOfKind(levels []DetectedLevel, k LevelKind) (DetectedLevel, bool) {
	for _, l := range levels {
		if l.Kind == k {
			return l, true
		}
	}
	return DetectedLevel{}, false
}

func TestEqualHighs(t *testing.T) {
	bars := series([][4]float64{
		{15540, 15550, 15530, 15545},
		{15550, 15560, 15540, 15555},
		{15560, 15600, 15555, 15570}, // pivot high 15600
		{15560, 15570, 15550, 15560},
		{15550, 15560, 15545, 15555},
		{15560, 15580, 15550, 15570},
		{15570, 15602, 15560, 15580}, // pivot high 15602 (≈ equal within tol)
		{15560, 15570, 15550, 15560},
		{15540, 15550, 15530, 15545},
	})
	levels := EqualHighsLows(bars, 5, time.UnixMilli(nowAfter(bars)))
	eqh, ok := firstOfKind(levels, KindEQH)
	if !ok {
		t.Fatalf("expected an EQH liquidity level, got %v", levels)
	}
	if eqh.Price != 15602 { // group max of {15600,15602}
		t.Fatalf("EQH price = %v want 15602", eqh.Price)
	}
}

func TestSupplyDemandZones(t *testing.T) {
	// small-bodied base then a +35 departure (atr 20 → departure ≥30) → demand.
	bars := series([][4]float64{
		{15500, 15505, 15498, 15503}, // base (body 3)
		{15503, 15508, 15500, 15505}, // base (body 2)
		{15505, 15545, 15503, 15540}, // departure up (body 35)
	})
	zones := SupplyDemandZones(bars, 20, time.UnixMilli(nowAfter(bars)))
	z, ok := firstOfKind(zones, KindDemand)
	if !ok {
		t.Fatalf("expected a demand zone, got %v", zones)
	}
	if z.Lo != 15498 || z.Hi != 15508 {
		t.Fatalf("demand zone = [%v,%v] want [15498,15508]", z.Lo, z.Hi)
	}
}

func TestFairValueGaps(t *testing.T) {
	// bar0.High 15505 < bar2.Low 15520 → bullish FVG [15505,15520] (gap 15 ≥ minGap 10).
	bars := series([][4]float64{
		{15500, 15505, 15498, 15503},
		{15510, 15530, 15508, 15525},
		{15528, 15540, 15520, 15535},
	})
	fvgs := FairValueGaps(bars, 10, time.UnixMilli(nowAfter(bars)))
	f, ok := firstOfKind(fvgs, KindFVG)
	if !ok {
		t.Fatalf("expected an FVG, got %v", fvgs)
	}
	if f.Lo != 15505 || f.Hi != 15520 {
		t.Fatalf("FVG = [%v,%v] want [15505,15520]", f.Lo, f.Hi)
	}
}

func TestFairValueGapsFilled(t *testing.T) {
	// same gap, then a bar that trades fully through it → not emitted.
	bars := series([][4]float64{
		{15500, 15505, 15498, 15503},
		{15510, 15530, 15508, 15525},
		{15528, 15540, 15520, 15535},
		{15530, 15522, 15500, 15505}, // low 15500 ≤ 15505, high 15522 ≥ 15520 → fills
	})
	if got := FairValueGaps(bars, 10, time.UnixMilli(nowAfter(bars))); len(got) != 0 {
		t.Fatalf("filled FVG must not be emitted, got %v", got)
	}
}

func TestOrderBlocks(t *testing.T) {
	// +40 displacement (atr 20 → ≥30); OB = the last DOWN candle before it.
	bars := series([][4]float64{
		{15500, 15505, 15495, 15498}, // down candle → the OB
		{15498, 15502, 15494, 15500}, // up
		{15500, 15545, 15498, 15540}, // displacement up (body 40)
	})
	obs := OrderBlocks(bars, 20, time.UnixMilli(nowAfter(bars)))
	ob, ok := firstOfKind(obs, KindOB)
	if !ok {
		t.Fatalf("expected an order block, got %v", obs)
	}
	if ob.Lo != 15495 || ob.Hi != 15505 || ob.Label != "OB(bull)" {
		t.Fatalf("OB = [%v,%v] %q want [15495,15505] OB(bull)", ob.Lo, ob.Hi, ob.Label)
	}
}

func TestZonesAreConfluenceOnlyInScorer(t *testing.T) {
	// A standalone supply zone must NOT survive scoring (confluence-only rule).
	zone := zoneLevel(KindSupply, 15540, 15560, "Supply", "d")
	scored := ScoreLevels([]DetectedLevel{zone}, 15530, 200, nil, 8, 1.5)
	if len(scored) != 0 {
		t.Fatalf("standalone zone must be excluded by the scorer, got %v", scored)
	}
}
