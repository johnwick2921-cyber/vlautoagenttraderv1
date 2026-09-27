package kernel

import (
	"nofx/market"
	"testing"
	"time"
)

// F21 (port of #117 05a1775c) — SWING PROVENANCE AND AGGREGATE VOLUME.

func TestSwingZoneWickComesFromSelectedPivot(t *testing.T) {
	start := time.Date(2026, 9, 10, 9, 0, 0, 0, CTLocation())
	bars := make([]market.Kline, 21)
	for i := range bars {
		open := start.Add(time.Duration(i) * 5 * time.Minute)
		bars[i] = market.Kline{OpenTime: open.UnixMilli(), CloseTime: open.Add(5 * time.Minute).UnixMilli(), Open: 100, High: 101, Low: 99, Close: 100, Volume: 10}
	}
	bars[10].Open = 104
	bars[10].High = 110
	bars[10].Close = 105
	levels := swingPointsFor(bars, 5, start.Add(106*time.Minute))
	found := false
	for _, level := range levels {
		if level.Kind == KindSWGH && level.Price == 110 {
			found = true
			if level.ZoneDefiningWick == nil || *level.ZoneDefiningWick != 5 {
				t.Fatalf("selected pivot wick=5, got %v (next bar wick=1)", level.ZoneDefiningWick)
			}
		}
	}
	if !found {
		t.Fatal("fixture high was not selected")
	}
}

func TestSwingAggregationConservesFirstBarVolume(t *testing.T) {
	start := time.Date(2026, 9, 10, 9, 0, 0, 0, CTLocation()).UnixMilli()
	bars := []market.Kline{
		{OpenTime: start, Open: 100, High: 102, Low: 99, Close: 101, Volume: 10},
		{OpenTime: start + 60000, Open: 101, High: 103, Low: 100, Close: 102, Volume: 7},
		{OpenTime: start + 300000, Open: 102, High: 104, Low: 101, Close: 103, Volume: 11},
	}
	got := aggregateBars(bars, 5)
	if len(got) != 2 || got[0].Volume != 17 || got[1].Volume != 11 {
		t.Fatalf("aggregation lost source volume: %+v", got)
	}
	if got[0].Open != 100 || got[0].High != 103 || got[0].Low != 99 || got[0].Close != 102 {
		t.Fatalf("OHLC changed: %+v", got[0])
	}
}
