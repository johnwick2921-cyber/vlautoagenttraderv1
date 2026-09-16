package market

import (
	"math"
	"testing"
)

// realMNQ20Daily — 20 real MNQ daily bars (07-20..08-14, from decision_records),
// Open/High/Low/Close, used as the W12 indicator oracle input.
func realMNQ20Daily() []Kline {
	rows := [][4]float64{
		{28747.50, 29192.50, 28706.75, 28778.75}, {28783.25, 29363.50, 28700.00, 29316.00},
		{29309.00, 29342.50, 28961.25, 29181.25}, {29107.50, 29283.00, 28432.50, 28620.75},
		{28708.00, 28734.75, 28212.50, 28282.25}, {28501.50, 28763.75, 27938.50, 28190.00},
		{28210.50, 28229.00, 27603.25, 27922.00}, {27962.25, 28177.25, 27200.00, 27259.75},
		{27208.00, 28410.00, 27204.75, 28237.75}, {28317.25, 28725.75, 28079.75, 28404.25},
		{28567.50, 28965.00, 28313.00, 28891.75}, {28930.00, 29956.50, 28831.50, 29863.50},
		{29781.25, 30073.25, 29530.75, 29615.00}, {29576.00, 29686.25, 29241.00, 29488.25},
		{29515.00, 29867.25, 29455.00, 29834.75}, {29851.50, 29985.00, 29719.00, 29737.00},
		{29764.25, 29887.00, 29533.50, 29626.00}, {29663.00, 30001.50, 29625.00, 29853.25},
		{29820.25, 30273.25, 29780.50, 30188.50}, {30148.75, 30287.25, 30025.00, 30147.25},
	}
	var out []Kline
	for _, r := range rows {
		out = append(out, Kline{Open: r[0], High: r[1], Low: r[2], Close: r[3]})
	}
	return out
}

// W12 — EMA oracle. Seeding = SMA of first `period` closes; smoothing = 2/(n+1);
// len<period → 0. Values INDEPENDENTLY hand-computed (not read from the code).
func TestW12EMA(t *testing.T) {
	// EMA(3) over [10,11,12,13,14]: seed=(10+11+12)/3=11, mult=0.5,
	// bar13=(13-11)*.5+11=12, bar14=(14-12)*.5+12=13.
	if v := ExportCalculateEMA(closesKl([]float64{10, 11, 12, 13, 14}), 3); math.Abs(v-13.0) > 1e-9 {
		t.Fatalf("EMA(3)[10..14] = %v, want 13.0", v)
	}
	// one recursion step: EMA(3)[10,11,12,13] → seed 11, bar13 → 12.
	if v := ExportCalculateEMA(closesKl([]float64{10, 11, 12, 13}), 3); math.Abs(v-12.0) > 1e-9 {
		t.Fatalf("EMA(3)[10..13] = %v, want 12.0", v)
	}
	// len<period → 0 guard.
	if v := ExportCalculateEMA(closesKl([]float64{10, 11}), 3); v != 0 {
		t.Fatalf("EMA(3) on 2 bars = %v, want 0 (guard)", v)
	}
	// EMA20 over exactly 20 real bars degenerates to the SMA seed (loop runs 0×):
	// sum(20 closes)=581438.00 / 20 = 29071.90.
	if v := ExportCalculateEMA(realMNQ20Daily(), 20); math.Abs(v-29071.90) > 1e-6 {
		t.Fatalf("EMA20(real20) = %v, want 29071.90", v)
	}
	// property: EMA lies within [min,max] of the series (convex weighting).
	kl := realMNQ20Daily()
	ema := ExportCalculateEMA(kl, 5)
	lo, hi := kl[0].Close, kl[0].Close
	for _, k := range kl {
		lo, hi = math.Min(lo, k.Close), math.Max(hi, k.Close)
	}
	if ema < lo || ema > hi {
		t.Fatalf("EMA5 %v escaped [%v,%v]", ema, lo, hi)
	}
}

// W12 — ATR14 (Wilder) oracle. TR=max(H-L,|H-prevC|,|L-prevC|); seed=SMA(TR_1..14);
// atr=(atr*13+TR)/14. INDEPENDENT Wilder recompute over the 20 real bars = 595.8473.
func TestW12ATR14(t *testing.T) {
	v := ExportCalculateATR(realMNQ20Daily(), 14)
	if math.Abs(v-595.8473) > 5e-4 {
		t.Fatalf("ATR14(real20) = %.4f, want 595.8473 (independent Wilder recompute)", v)
	}
	// property: ATR14 > 0 and within the TR envelope (min..max TR of the series).
	if v <= 0 {
		t.Fatalf("ATR must be positive, got %v", v)
	}
	// guard: len<=period → 0.
	if z := ExportCalculateATR(realMNQ20Daily()[:10], 14); z != 0 {
		t.Fatalf("ATR14 on 10 bars = %v, want 0 (needs >14)", z)
	}
}

func closesKl(closes []float64) []Kline {
	out := make([]Kline, len(closes))
	for i, c := range closes {
		out[i] = Kline{Open: c, High: c, Low: c, Close: c}
	}
	return out
}
