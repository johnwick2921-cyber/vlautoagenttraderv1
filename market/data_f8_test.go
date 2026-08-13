// F8 — EMA200 (and any long-period series) must actually warm up. The fetch depth
// is now clamp(maxConfiguredPeriod + count, 200, 2500), so the last `count` bars all
// have enough history and the series renders `count` real values — not a 1-value
// warm-up stub as when the fetch was a flat 200.
package market

import "testing"

func TestF8_EMA200SeriesWarmsUpToCount(t *testing.T) {
	orig := FuturesBarsProvider
	defer func() { FuturesBarsProvider = orig }()

	// Mock the NT8 provider: return exactly the requested number of strictly-rising
	// (non-frozen) synthetic bars, so we can assert the series length the fetch depth
	// produces.
	FuturesBarsProvider = func(symbol, tf string, count int) []Kline {
		ks := make([]Kline, count)
		base := int64(1_700_000_000_000)
		for i := 0; i < count; i++ {
			price := 20000.0 + float64(i)*0.5 // rising → not stale
			ks[i] = Kline{
				OpenTime: base + int64(i)*900_000, Open: price, High: price + 1,
				Low: price - 1, Close: price, Volume: 100,
				CloseTime: base + int64(i)*900_000 + 899_999,
			}
		}
		return ks
	}

	const count = 20
	ip := IndicatorPeriods{EMA: []int{200}}
	d, err := GetWithTimeframes("MNQ", []string{"15m"}, "15m", count, ip)
	if err != nil {
		t.Fatalf("GetWithTimeframes: %v", err)
	}
	ser := d.TimeframeData["15m"]
	if ser == nil {
		t.Fatal("no 15m series")
	}
	got := len(ser.EMAByPeriod[200])
	if got != count {
		t.Fatalf("EMA200 series length = %d, want == count %d (F8: fetch maxPeriod+count so it warms up; pre-F8 this was 1)", got, count)
	}
	// Every value must be a real EMA (> 0), not a warm-up stub.
	for i, v := range ser.EMAByPeriod[200] {
		if v <= 0 {
			t.Fatalf("EMA200[%d] = %v, want > 0 (real value)", i, v)
		}
	}
}

func TestF8_FetchDepthClamp(t *testing.T) {
	// max period 200, count 20 → 220, within [200,2500].
	if d := clampInt(maxConfiguredPeriod(IndicatorPeriods{EMA: []int{200}})+20, 200, 2500); d != 220 {
		t.Errorf("fetch depth = %d, want 220", d)
	}
	// No configured periods → floor 200 (never fetch less than the legacy amount).
	if d := clampInt(maxConfiguredPeriod(IndicatorPeriods{})+10, 200, 2500); d != 200 {
		t.Errorf("fetch depth = %d, want 200 (floor)", d)
	}
	// Absurd period → ceiling 2500.
	if d := clampInt(maxConfiguredPeriod(IndicatorPeriods{EMA: []int{5000}})+100, 200, 2500); d != 2500 {
		t.Errorf("fetch depth = %d, want 2500 (ceiling)", d)
	}
}
