package trader

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// THE HOOK'S OWN CALIBRATION (owner ruling 2026-09-03).
//
// E2 proved the DETECTOR is a coin flip on IID-shuffled real tape. It did not
// prove the HOOK feeds it correctly: the writer test used a deterministic
// zig-zag and read p(hold)=0.00, which is indistinguishable from a hook that
// inverts a side or reorders bars. This runs the PRODUCTION CALL PATH over the
// same tape E2 used and reads the rate back OUT OF THE TABLE.
//
// If this reads ~0 or ~1, the hook is broken however green the unit test is.
func TestDetectorHookIsACoinFlipThroughTheProductionPath(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "kernel", "testdata", "d1prime_calibration_tape.json"))
	if err != nil {
		t.Skipf("calibration tape unavailable: %v", err)
	}
	var tape struct {
		StartClose float64   `json:"start_close"`
		Increments []float64 `json:"increments"`
		WickUp     []float64 `json:"wick_up"`
		WickDown   []float64 `json:"wick_down"`
	}
	if err := json.Unmarshal(b, &tape); err != nil {
		t.Fatal(err)
	}

	st, err := store.New(filepath.Join(t.TempDir(), "hookcal.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	at := &AutoTrader{id: "hoang", store: st}

	rng := rand.New(rand.NewSource(20260903))
	now := time.Date(2026, 9, 3, 20, 0, 0, 0, kernel.CTLocation())
	dayStart := kernel.CMESessionDayStart(now)

	inc := append([]float64(nil), tape.Increments...)
	wu := append([]float64(nil), tape.WickUp...)
	wd := append([]float64(nil), tape.WickDown...)
	rng.Shuffle(len(inc), func(i, j int) { inc[i], inc[j] = inc[j], inc[i] })
	rng.Shuffle(len(wu), func(i, j int) { wu[i], wu[j] = wu[j], wu[i] })
	rng.Shuffle(len(wd), func(i, j int) { wd[i], wd[j] = wd[j], wd[i] })

	// Bars are timestamped INSIDE the session day, exactly as the hook expects.
	bars := make([]market.Kline, 0, len(inc))
	p := tape.StartClose
	lo, hi := p, p
	for i, d := range inc {
		o, c := p, p+d
		h := math.Max(o, c) + math.Abs(wu[i%len(wu)])
		l := math.Min(o, c) - math.Abs(wd[i%len(wd)])
		ts := dayStart.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: ts.UnixMilli(), Open: o, High: h, Low: l, Close: c,
			CloseTime: ts.Add(time.Minute).UnixMilli()})
		p = c
		lo, hi = math.Min(lo, c), math.Max(hi, c)
	}
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline { return bars }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	// A grid of seated levels across the traversed range — the same shape E2
	// used to get an adequate sample.
	step := (hi - lo) / 40.0
	var seated []kernel.ScoredLevel
	for i := 1; i < 40; i++ {
		// WAVE A / D1e — certified so these episodes can form a rate (see
		// TestLineLevelsCannotBeCertifiedAndAreExcludedFromRates for the
		// uncertified case, which is what production line levels hit).
		seated = append(seated, kernel.ScoredLevel{
			DetectedLevel: kernel.DetectedLevel{
				Price: lo + float64(i)*step, Label: "GRID", FormedAtMs: bars[0].OpenTime,
			}, Score: 90, Grade: "A",
		})
	}
	at.recordDetectorOutputs("MNQ", "2026-09-03:ASIA:hoang", "ASIA", 1, nil, seated,
		tape.StartClose, 300, 99.0, 64, now, nil)

	ts := st.TouchOutcomes()
	rates, err := ts.RatesBy("")
	if err != nil || len(rates) == 0 {
		t.Fatalf("the hook wrote nothing to read back: %v", err)
	}
	r := rates[0]
	if r.N() < 1000 {
		t.Fatalf("underpowered through the hook: n=%d (ambiguous=%d)", r.N(), r.Ambiguous)
	}
	lo95, hi95 := 0.0, 0.0
	{
		const z = 1.959963984540054
		nf := float64(r.N())
		den := 1 + z*z/nf
		c := (r.P() + z*z/(2*nf)) / den
		h := z * math.Sqrt(r.P()*(1-r.P())/nf+z*z/(4*nf*nf)) / den
		lo95, hi95 = c-h, c+h
	}
	t.Logf("THROUGH THE HOOK: p(hold)=%.4f [%.4f, %.4f] n=%d ambiguous=%d (%.1f%%)",
		r.P(), lo95, hi95, r.N(), r.Ambiguous, 100*float64(r.Ambiguous)/float64(r.N()+r.Ambiguous))

	if r.P() < 0.05 || r.P() > 0.95 {
		t.Fatalf("p(hold)=%.4f through the hook — a degenerate rate means the HOOK feeds the detector a wrong side or bar order, however green the unit test is", r.P())
	}
	if math.Abs(r.P()-0.50) > 0.05 {
		t.Errorf("p(hold)=%.4f is outside 0.50±0.05 through the production path", r.P())
	}
}
