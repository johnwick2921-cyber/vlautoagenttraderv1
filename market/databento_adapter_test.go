package market

import (
	"nofx/provider/databento"
	"testing"
	"time"
)

func TestBarsToKlines_Mapping(t *testing.T) {
	bars := []databento.Bar{
		{
			Timestamp: time.Unix(1746360000, 0).UTC(),
			Open:      21500.25,
			High:      21515.75,
			Low:       21498.00,
			Close:     21510.00,
			Volume:    4321,
		},
	}
	klines := BarsToKlines(bars)
	if len(klines) != 1 {
		t.Fatalf("want 1 kline, got %d", len(klines))
	}
	k := klines[0]
	if k.Open != 21500.25 || k.High != 21515.75 || k.Low != 21498.00 || k.Close != 21510.00 || k.Volume != 4321 {
		t.Errorf("kline OHLCV mismatch: %+v", k)
	}
	wantMs := int64(1746360000 * 1000)
	if k.OpenTime != wantMs {
		t.Errorf("kline.OpenTime = %d, want %d", k.OpenTime, wantMs)
	}
}

func TestBarsToKlines_Empty(t *testing.T) {
	got := BarsToKlines(nil)
	if len(got) != 0 {
		t.Errorf("want 0 klines, got %d", len(got))
	}
}
