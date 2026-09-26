package market

import (
	"math"
	"testing"
	"time"
)

// W1 — a price change named "1h" or "4h" measures that duration of wall time
// on bar close times, and is ABSENT (nil → "n/a") when the series cannot
// measure it. Before W1 the futures "1h" was 20 bars of 5m (100 min), the
// "4h" was the previous long bar's close, and both were a fabricated 0 when
// the bars ran short.

// series builds n contiguous bars of `step`, closes base+i, ending now.
func series(step time.Duration, n int, base float64) []Kline {
	out := make([]Kline, n)
	start := time.Now().Add(-time.Duration(n) * step)
	for i := range out {
		ot := start.Add(time.Duration(i) * step).UnixMilli()
		c := base + float64(i)
		out[i] = Kline{OpenTime: ot, Open: c, High: c + 1, Low: c - 1, Close: c, Volume: 100, CloseTime: ot + step.Milliseconds() - 1}
	}
	return out
}

func pctFrom(last, ref float64) float64 { return (last - ref) / ref * 100 }

func wantPct(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is absent, want %.6f", name, want)
	}
	if math.Abs(*got-want) > 1e-9 {
		t.Fatalf("%s = %.6f, want %.6f", name, *got, want)
	}
}

func TestChangeOverWindowMeasuresBarTime(t *testing.T) {
	m5 := series(5*time.Minute, 200, 20000)
	last := m5[199].Close
	wantPct(t, "5m 1h", ChangeOverWindow(m5, time.Hour), pctFrom(last, m5[199-12].Close))
	wantPct(t, "5m 4h", ChangeOverWindow(m5, 4*time.Hour), pctFrom(last, m5[199-48].Close))
	m3 := series(3*time.Minute, 100, 100)
	wantPct(t, "3m 1h", ChangeOverWindow(m3, time.Hour), pctFrom(m3[99].Close, m3[99-20].Close))

	for name, got := range map[string]*float64{
		"empty":            ChangeOverWindow(nil, time.Hour),
		"zero window":      ChangeOverWindow(m5, 0),
		"too short for 1h": ChangeOverWindow(series(5*time.Minute, 12, 20000), time.Hour),
		"too short for 4h": ChangeOverWindow(series(5*time.Minute, 48, 20000), 4*time.Hour),
		"reference close ≤ 0": ChangeOverWindow(func() []Kline {
			s := series(5*time.Minute, 13, 20000)
			s[0].Close = 0
			return s
		}(), time.Hour),
	} {
		if got != nil {
			t.Errorf("%s: want absent (nil), got %v", name, *got)
		}
	}
	// 13 bars of 5m is exactly enough for 1h.
	if ChangeOverWindow(series(5*time.Minute, 13, 20000), time.Hour) == nil {
		t.Error("13 bars of 5m reach back 1h and must measure it")
	}
}

// A halt: the reference is the last bar that closed at least the window
// before the latest one, however far back that is.
func TestChangeOverWindowAcrossAGap(t *testing.T) {
	before := series(5*time.Minute, 20, 100)
	after := series(5*time.Minute, 5, 200)
	// shift `before` two hours earlier than `after`'s start.
	shift := (2 * time.Hour).Milliseconds()
	for i := range before {
		before[i].OpenTime -= shift + int64(len(after))*(5*time.Minute).Milliseconds()
		before[i].CloseTime -= shift + int64(len(after))*(5*time.Minute).Milliseconds()
	}
	s := append(before, after...)
	wantPct(t, "1h across a halt", ChangeOverWindow(s, time.Hour), pctFrom(s[len(s)-1].Close, before[len(before)-1].Close))
}

func TestResolutionGuardRefusesAWindowTheSeriesCannotResolve(t *testing.T) {
	m5 := series(5*time.Minute, 200, 20000)
	h1 := series(time.Hour, 200, 20000)
	if changeOnTimeframe(m5, "5m", time.Hour) == nil || changeOnTimeframe(m5, "5m", 4*time.Hour) == nil {
		t.Fatal("5m bars resolve both 1h and 4h")
	}
	if got := changeOnTimeframe(h1, "1h", time.Hour); got != nil {
		t.Fatalf("a 1h series cannot measure a 1h change (it would be the previous close); got %v", *got)
	}
	if changeOnTimeframe(h1, "1h", 4*time.Hour) == nil {
		t.Fatal("1h bars resolve a 4h window (4 bars)")
	}
	if got := changeOnTimeframe(m5, "7m", time.Hour); got != nil {
		t.Fatalf("an unknown timeframe cannot be resolved; got %v", *got)
	}
}

// The production futures read: both windows on the NT8 5m series by time.
func TestFuturesReadMeasuresBothWindowsByTime(t *testing.T) {
	m5 := series(5*time.Minute, 200, 20000)
	h1 := series(time.Hour, 200, 19000)
	stubFuturesBars(t, m5, h1)
	d, err := GetWithExchange("MNQ", "ninjatrader")
	if err != nil {
		t.Fatal(err)
	}
	last := m5[199].Close
	wantPct(t, "futures 1h", d.PriceChange1h, pctFrom(last, m5[199-12].Close))
	wantPct(t, "futures 4h", d.PriceChange4h, pctFrom(last, m5[199-48].Close))
	if old := pctFrom(last, m5[199-20].Close); math.Abs(*d.PriceChange1h-old) < 1e-9 {
		t.Fatal("the 1h change is still the 20-bar (100 min) lookback")
	}
	if old := pctFrom(last, h1[198].Close); math.Abs(*d.PriceChange4h-old) < 1e-9 {
		t.Fatal("the 4h change is still the previous 1h close")
	}
}

func TestFuturesReadWithAShortTapeReportsTheChangesAbsent(t *testing.T) {
	stubFuturesBars(t, series(5*time.Minute, 10, 20000), series(time.Hour, 10, 19000))
	d, err := GetWithExchange("MNQ", "ninjatrader")
	if err != nil {
		t.Fatal(err)
	}
	if d.PriceChange1h != nil || d.PriceChange4h != nil {
		t.Fatalf("10 bars of 5m reach back 45 min: both changes must be absent, got %v / %v", d.PriceChange1h, d.PriceChange4h)
	}
}

// The multi-timeframe read measures on the PRIMARY series and refuses a
// window its resolution cannot measure.
func TestTimeframesReadMeasuresOnThePrimarySeries(t *testing.T) {
	m5 := series(5*time.Minute, 300, 20000)
	h1 := series(time.Hour, 300, 19000)
	stubFuturesBars(t, m5, h1)
	d, err := GetWithTimeframesVenue("MNQ", "ninjatrader", []string{"5m", "1h"}, "5m", 100)
	if err != nil {
		t.Fatal(err)
	}
	if d.PriceChange1h == nil || d.PriceChange4h == nil {
		t.Fatal("a 5m primary measures both windows")
	}
	d1, err := GetWithTimeframesVenue("MNQ", "ninjatrader", []string{"1h"}, "1h", 100)
	if err != nil {
		t.Fatal(err)
	}
	if d1.PriceChange1h != nil {
		t.Fatalf("a 1h primary cannot measure a 1h change; got %v", *d1.PriceChange1h)
	}
	if d1.PriceChange4h == nil {
		t.Fatal("a 1h primary measures a 4h change (4 bars)")
	}
}

func TestPctOrNA(t *testing.T) {
	v := 1.5
	if PctOrNA(&v, true) != "+1.50%" || PctOrNA(&v, false) != "1.50%" || PctOrNA(nil, true) != "n/a" {
		t.Fatalf("PctOrNA: %q %q %q", PctOrNA(&v, true), PctOrNA(&v, false), PctOrNA(nil, true))
	}
}

func stubFuturesBars(t *testing.T, m5, h1 []Kline) {
	t.Helper()
	prev := FuturesBarsProvider
	FuturesBarsProvider = func(symbol, tf string, count int) []Kline {
		var s []Kline
		switch tf {
		case "5m":
			s = m5
		case "1h":
			s = h1
		default:
			return nil
		}
		if count > 0 && count < len(s) {
			s = s[len(s)-count:]
		}
		return s
	}
	t.Cleanup(func() { FuturesBarsProvider = prev })
}
