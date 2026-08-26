package kernel

import (
	"math"
	"strings"
	"testing"

	"nofx/market"
)

// genBars builds n bars trending up by `step`/bar with a fixed range and a small
// alternating wiggle (so return variance is non-zero).
func genBars(n int, start, step, rng float64, intervalMs int64) []market.Kline {
	bars := make([]market.Kline, n)
	prev := start
	for i := 0; i < n; i++ {
		c := start + float64(i)*step
		if i%2 == 1 {
			c += rng * 0.1
		}
		o := prev
		hi := math.Max(o, c) + rng/2
		lo := math.Min(o, c) - rng/2
		open := int64(i) * intervalMs
		bars[i] = market.Kline{
			OpenTime: open, Open: o, High: hi, Low: lo, Close: c,
			CloseTime: open + intervalMs - 1,
		}
		prev = c
	}
	return bars
}

func TestComputeRegimeUptrend(t *testing.T) {
	daily := genBars(250, 15000, 1, 40, 86400000)
	hour1 := genBars(60, 15100, 2, 20, 3600000)
	min5 := genBars(80, 15200, 0.2, 6, 300000)
	in := RegimeInputs{
		Price:       15400, // above both EMAs → up
		DailyBars:   daily,
		Hour1Bars:   hour1,
		Min5Bars:    min5,
		PriorClose:  15240,
		SessionOpen: 15250, // +10 gap
	}
	r := ComputeRegime(in)

	if r.TrendDaily != "up" {
		t.Fatalf("trend daily = %q want up", r.TrendDaily)
	}
	if r.Trend1h != "up" {
		t.Fatalf("trend 1h = %q want up", r.Trend1h)
	}
	if r.ATR14 <= 0 {
		t.Fatalf("ATR14 should be positive, got %v", r.ATR14)
	}
	if r.ATRRegime == "n/a" || r.ATRPercentile < 0 || r.ATRPercentile > 100 {
		t.Fatalf("ATR regime not computed: %q p=%v", r.ATRRegime, r.ATRPercentile)
	}
	if r.RealizedVolPct <= 0 || !r.RVWarming {
		t.Fatalf("RV should be positive + warming (no baseline): %v warming=%v", r.RealizedVolPct, r.RVWarming)
	}
	if r.VIXRegime != "unavailable" || r.VIXLevel != 0 {
		t.Fatalf("VIX should be unavailable (no feed): %+v", r)
	}
	if r.ExpectedRangePts != r.ATR14 {
		t.Fatalf("no-VIX expected range should equal ATR14: %v vs %v", r.ExpectedRangePts, r.ATR14)
	}
	if !r.HasGap || r.OvernightGapATR <= 0 {
		t.Fatalf("expected a positive overnight gap: %+v", r)
	}
	line := r.Render()
	if !strings.Contains(line, "trend D=up 1h=up") || !strings.Contains(line, "VIX=n/a") {
		t.Fatalf("render line wrong: %q", line)
	}
}

func TestComputeRegimeColdStart(t *testing.T) {
	r := ComputeRegime(RegimeInputs{Price: 15000})
	if r.TrendDaily != "n/a" || r.Trend1h != "n/a" || r.ATRRegime != "n/a" {
		t.Fatalf("cold start must be all n/a: %+v", r)
	}
	if r.ATR14 != 0 || r.HasGap {
		t.Fatalf("cold start must have no ATR/gap: %+v", r)
	}
	if !strings.Contains(r.Render(), "trend D=n/a 1h=n/a") {
		t.Fatalf("cold render wrong: %q", r.Render())
	}
}

func TestComputeRegimeVIXAndDowntrend(t *testing.T) {
	// Downtrend: price below EMA200.
	daily := genBars(250, 16000, -1, 40, 86400000) // descending
	in := RegimeInputs{
		Price:     15600, // below EMA200 of a descending series
		DailyBars: daily,
		VIX:       25,
	}
	r := ComputeRegime(in)
	if r.TrendDaily != "down" {
		t.Fatalf("trend daily = %q want down", r.TrendDaily)
	}
	if r.VIXRegime != "20-30" || r.VIXLevel != 25 {
		t.Fatalf("VIX bucket wrong: %+v", r)
	}
	// VIX-implied expected range (not the ATR fallback).
	wantER := 25.0 / 100 / math.Sqrt(252) * 15600
	if math.Abs(r.ExpectedRangePts-wantER) > 1e-6 {
		t.Fatalf("VIX expected range = %v want %v", r.ExpectedRangePts, wantER)
	}
}

func TestRegimeBuckets(t *testing.T) {
	cases := []struct {
		p    float64
		want string
	}{{10, "LOW"}, {24.9, "LOW"}, {25, "NORMAL"}, {74, "NORMAL"}, {75, "HIGH"}, {89, "HIGH"}, {90, "EXTREME"}, {99, "EXTREME"}}
	for _, c := range cases {
		if got := atrBucket(c.p); got != c.want {
			t.Fatalf("atrBucket(%v) = %q want %q", c.p, got, c.want)
		}
	}
	vix := []struct {
		v    float64
		want string
	}{{12, "<15"}, {17, "15-20"}, {25, "20-30"}, {35, ">30"}}
	for _, c := range vix {
		if got := vixBucket(c.v); got != c.want {
			t.Fatalf("vixBucket(%v) = %q want %q", c.v, got, c.want)
		}
	}
}
