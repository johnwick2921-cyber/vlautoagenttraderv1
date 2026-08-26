package kernel

import (
	"os"
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
	if z.FormedAtMs != bars[2].OpenTime {
		t.Fatalf("zone formed_at = %d want %d (departure bar)", z.FormedAtMs, bars[2].OpenTime)
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
	if f.FormedAtMs != bars[2].OpenTime {
		t.Fatalf("FVG formed_at = %d want %d", f.FormedAtMs, bars[2].OpenTime)
	}
}

func TestFairValueGapsFilled(t *testing.T) {
	// A wick fully through the gap but a close INSIDE it does not invert (SMC:
	// inversion needs a stay-through CLOSE beyond the gap edge).
	bars := series([][4]float64{
		{15500, 15505, 15498, 15503},
		{15510, 15530, 15508, 15525},
		{15528, 15540, 15520, 15535},
		{15520, 15522, 15499, 15512}, // wick 15499–15522 through, close 15512 inside
	})
	lvls := FairValueGaps(bars, 10, time.UnixMilli(nowAfter(bars)))
	f, ok := firstOfKind(lvls, KindFVG)
	if !ok {
		t.Fatalf("wick-through with close inside must stay an FVG, got %v", lvls)
	}
	if f.Lo != 15505 || f.Hi != 15520 {
		t.Fatalf("FVG = [%v,%v] want [15505,15520]", f.Lo, f.Hi)
	}
	if _, inv := firstOfKind(lvls, KindIFVG); inv {
		t.Fatalf("close inside the gap must NOT invert, got %v", lvls)
	}
}

func TestFairValueGapsInvertsOnCloseThrough(t *testing.T) {
	// Bullish gap [15505,15520]; a later CLOSE below the gap low flips it to a
	// bearish iFVG (resistance) with the same bounds — W6 (2026-08-25).
	bars := series([][4]float64{
		{15500, 15505, 15498, 15503},
		{15510, 15530, 15508, 15525},
		{15528, 15540, 15520, 15535},
		{15530, 15532, 15500, 15502}, // close 15502 < 15505 → inversion
	})
	lvls := FairValueGaps(bars, 10, time.UnixMilli(nowAfter(bars)))
	f, ok := firstOfKind(lvls, KindIFVG)
	if !ok {
		t.Fatalf("close-through fill must emit an iFVG, got %v", lvls)
	}
	if f.Lo != 15505 || f.Hi != 15520 || f.Label != "iFVG(bear)" {
		t.Fatalf("iFVG = [%v,%v] %q want [15505,15520] iFVG(bear)", f.Lo, f.Hi, f.Label)
	}
	if f.Info != "filled→inverted" {
		t.Fatalf("iFVG info = %q want filled→inverted", f.Info)
	}
	if f.FormedAtMs != bars[2].OpenTime {
		t.Fatalf("iFVG formed_at = %d want %d", f.FormedAtMs, bars[2].OpenTime)
	}
	if _, still := firstOfKind(lvls, KindFVG); still {
		t.Fatalf("an inverted gap must not ALSO emit a plain FVG, got %v", lvls)
	}
}

func TestFairValueGapsBearishInversion(t *testing.T) {
	// Bearish gap: bar0.Low 15522 > bar2.High 15512 → gap [15512,15522]; a later
	// close ABOVE the gap high flips it to a bullish iFVG (support).
	bars := series([][4]float64{
		{15525, 15540, 15522, 15535},
		{15510, 15512, 15500, 15505},
		{15510, 15512, 15500, 15505},
		{15518, 15530, 15516, 15524}, // close 15524 > 15522 → inversion
	})
	lvls := FairValueGaps(bars, 10, time.UnixMilli(nowAfter(bars)))
	f, ok := firstOfKind(lvls, KindIFVG)
	if !ok {
		t.Fatalf("bearish close-through fill must emit an iFVG, got %v", lvls)
	}
	if f.Lo != 15512 || f.Hi != 15522 || f.Label != "iFVG(bull)" {
		t.Fatalf("iFVG = [%v,%v] %q want [15512,15522] iFVG(bull)", f.Lo, f.Hi, f.Label)
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
	if ob.FormedAtMs != bars[2].OpenTime {
		t.Fatalf("OB formed_at = %d want %d (displacement bar)", ob.FormedAtMs, bars[2].OpenTime)
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

// R2 4.4 (2026-08-25) — the FVG detection floor is max(2×tick, noise floor),
// not 1×ATR: a 3-pt gap must detect (MNQ tick 0.25 → floor 2.0).
func TestFVGMinGapNoiseFloor(t *testing.T) {
	if got := fvgMinGapPoints("MNQ"); got != FVGNoiseFloorPoints {
		t.Fatalf("fvgMinGapPoints(MNQ) = %v want %v (2×tick=0.5 < floor)", got, FVGNoiseFloorPoints)
	}
	// bar0.High 15505 < bar2.Low 15508 → gap 3.0 ≥ floor 2.0 → FVG detected.
	bars := series([][4]float64{
		{15500, 15505, 15498, 15503},
		{15510, 15520, 15508, 15518},
		{15515, 15525, 15508, 15522},
	})
	fvgs := FairValueGaps(bars, fvgMinGapPoints("MNQ"), time.UnixMilli(nowAfter(bars)))
	if _, ok := firstOfKind(fvgs, KindFVG); !ok {
		t.Fatalf("3-pt gap must detect under the noise floor, got %v", fvgs)
	}
	// A sub-floor gap must NOT detect.
	tiny := series([][4]float64{
		{15500, 15501, 15499, 15500.5},
		{15501, 15502, 15501, 15501.5},
		{15502, 15503, 15502, 15502.5}, // gap ~1.0 < 2.0
	})
	if fvgs := FairValueGaps(tiny, fvgMinGapPoints("MNQ"), time.UnixMilli(nowAfter(bars))); len(fvgs) != 0 {
		t.Fatalf("sub-floor gap must not detect, got %v", fvgs)
	}
}

// R2 4.5 (2026-08-25) — the OB pairing scan is bounded: an opposing candle
// OUTSIDE the lookback window must never pair with a displacement.
func TestOBLookbackBounded(t *testing.T) {
	defer os.Unsetenv("OB_LOOKBACK_BARS")
	if err := os.Setenv("OB_LOOKBACK_BARS", "3"); err != nil {
		t.Fatal(err)
	}
	if got := obLookbackBars(); got != 3 {
		t.Fatalf("obLookbackBars() = %d want 3 (env override)", got)
	}
	// Down candle at bar0, then 3 neutral candles, then the displacement at
	// bar4: distance 4 > lookback 3 → NO OB.
	bars := series([][4]float64{
		{15500, 15505, 15495, 15498}, // the only down candle (far outside window)
		{15500, 15502, 15499, 15501},
		{15501, 15503, 15500, 15502},
		{15502, 15504, 15501, 15503},
		{15503, 15545, 15502, 15540}, // displacement up (body 37)
	})
	obs := OrderBlocks(bars, 20, time.UnixMilli(nowAfter(bars)))
	if _, ok := firstOfKind(obs, KindOB); ok {
		t.Fatalf("opposing candle outside the lookback must NOT pair, got %v", obs)
	}
	// Same window with lookback 5 → pairs.
	if err := os.Setenv("OB_LOOKBACK_BARS", "5"); err != nil {
		t.Fatal(err)
	}
	obs = OrderBlocks(bars, 20, time.UnixMilli(nowAfter(bars)))
	if _, ok := firstOfKind(obs, KindOB); !ok {
		t.Fatalf("opposing candle inside the lookback must pair, got %v", obs)
	}
}
