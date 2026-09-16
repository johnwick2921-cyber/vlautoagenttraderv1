package kernel

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"

	"nofx/market"
)

type parityFixture struct {
	Level  float64 `json:"level"`
	K      float64 `json:"k"`
	Delta  float64 `json:"delta"`
	H      int     `json:"H"`
	ExitOn string  `json:"exit_on"`
	Bars   []struct {
		T             int64   `json:"t"`
		O, High, L, C float64 `json:"-"`
		OJ            float64 `json:"o"`
		HJ            float64 `json:"h"`
		LJ            float64 `json:"l"`
		CJ            float64 `json:"c"`
	} `json:"bars"`
	Episodes []struct {
		Ordinal    int     `json:"ordinal"`
		T          int64   `json:"t"`
		Entry      string  `json:"entry"`
		Exit       *string `json:"exit"`
		Outcome    string  `json:"outcome"`
		BarsToExit int     `json:"bars_to_exit"`
		MFE        float64 `json:"mfe"`
		MAE        float64 `json:"mae"`
	} `json:"episodes"`
}

// E1 — PARITY with the Python reference (detect_symmetric_v2) on the same tape.
// Byte-equal episode lists: ordinal, open time, entry side, exit side, outcome,
// bars-to-exit, MFE and MAE. This is what makes the Go detector THE detector
// rather than a second opinion.
func TestD1PrimeParityWithPythonReference(t *testing.T) {
	b, err := os.ReadFile("testdata/d1prime_parity.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var fx parityFixture
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatalf("decode: %v", err)
	}
	bars := make([]market.Kline, 0, len(fx.Bars))
	for _, r := range fx.Bars {
		bars = append(bars, market.Kline{OpenTime: r.T, Open: r.OJ, High: r.HJ, Low: r.LJ, Close: r.CJ, CloseTime: r.T + 60_000})
	}
	got := DetectTouchOutcomes(bars, fx.Level, fx.K, fx.Delta, fx.H, fx.ExitOn)

	if len(got) != len(fx.Episodes) {
		t.Fatalf("episode COUNT differs: go=%d python=%d", len(got), len(fx.Episodes))
	}
	for i, want := range fx.Episodes {
		g := got[i]
		wantExit := ""
		if want.Exit != nil {
			wantExit = *want.Exit
		}
		if g.Ordinal != want.Ordinal || g.OpenedAtMs != want.T || g.Entry != want.Entry ||
			g.Exit != wantExit || g.Outcome != want.Outcome || g.BarsToExit != want.BarsToExit {
			t.Fatalf("episode %d differs:\n  go     = %+v\n  python = ordinal=%d t=%d entry=%s exit=%q outcome=%s bars=%d",
				i, g, want.Ordinal, want.T, want.Entry, wantExit, want.Outcome, want.BarsToExit)
		}
		if math.Abs(g.MFE-want.MFE) > 1e-9 || math.Abs(g.MAE-want.MAE) > 1e-9 {
			t.Fatalf("episode %d excursions differ: go mfe=%.12f mae=%.12f · python mfe=%.12f mae=%.12f",
				i, g.MFE, g.MAE, want.MFE, want.MAE)
		}
	}
	p, n, amb := HoldRate(got)
	t.Logf("parity: %d episodes byte-equal · p(hold)=%.4f n=%d ambiguous=%d (excluded)", len(got), p, n, amb)
}

// E2 — CALIBRATION on IID-SHUFFLED REAL TAPE, which is the redesign's own pass
// rule. Real MNQ 1m increments and wick geometry are shuffled independently, so
// every serial structure is destroyed and only the marginal distribution
// survives: a driftless tape with the instrument's real scale.
//
// TWO SYNTHETIC TAPES WERE TRIED FIRST AND BOTH LIED, which is worth recording:
// an Ornstein-Uhlenbeck tape pulled price back to the level and read 0.5221
// (biased toward HOLD by construction), and a pure random walk measured at ONE
// median level read 0.4573 on n=2,445 — underpowered and selection-skewed. The
// fixture, not the detector, was wrong both times. This version reproduces the
// report: p=0.5070 vs its 0.5067, ambiguous 17.8% vs its 17.8%.
func TestD1PrimeIsACoinFlipOnShuffledRealTape(t *testing.T) {
	b, err := os.ReadFile("testdata/d1prime_calibration_tape.json")
	if err != nil {
		t.Fatalf("tape: %v", err)
	}
	var tape struct {
		StartClose float64   `json:"start_close"`
		Increments []float64 `json:"increments"`
		WickUp     []float64 `json:"wick_up"`
		WickDown   []float64 `json:"wick_down"`
	}
	if err := json.Unmarshal(b, &tape); err != nil {
		t.Fatalf("decode: %v", err)
	}
	delta := MeanAbsIncrementOf(tape.Increments)
	k, H := DetectorK(), DetectorHorizonBars()
	rng := rand.New(rand.NewSource(20260903))
	var all []TouchOutcome
	for run := 0; run < 6; run++ {
		inc := append([]float64(nil), tape.Increments...)
		wu := append([]float64(nil), tape.WickUp...)
		wd := append([]float64(nil), tape.WickDown...)
		rng.Shuffle(len(inc), func(i, j int) { inc[i], inc[j] = inc[j], inc[i] })
		rng.Shuffle(len(wu), func(i, j int) { wu[i], wu[j] = wu[j], wu[i] })
		rng.Shuffle(len(wd), func(i, j int) { wd[i], wd[j] = wd[j], wd[i] })
		bars := make([]market.Kline, 0, len(inc))
		p := tape.StartClose
		for i, d := range inc {
			o, c := p, p+d
			h := math.Max(o, c) + math.Abs(wu[i%len(wu)])
			l := math.Min(o, c) - math.Abs(wd[i%len(wd)])
			ts := int64(1788400000000 + i*60000)
			bars = append(bars, market.Kline{OpenTime: ts, Open: o, High: h, Low: l, Close: c, CloseTime: ts + 60_000})
			p = c
		}
		lo, hi := bars[0].Close, bars[0].Close
		for _, x := range bars {
			lo, hi = math.Min(lo, x.Close), math.Max(hi, x.Close)
		}
		step := (hi - lo) / 60.0
		for gi := 1; gi < 60; gi++ {
			all = append(all, DetectTouchOutcomes(bars, lo+float64(gi)*step, k, delta, H, "close")...)
		}
	}
	p, n, amb := HoldRate(all)
	wlo, whi := WilsonInterval(p, n)
	t.Logf("calibration (IID-shuffled real MNQ 1m, %d-level grid): p(hold)=%.4f [%.4f, %.4f] n=%d ambiguous=%d (%.1f%%) · Δ=%.4f band=±%.2f",
		59, p, wlo, whi, n, amb, 100*float64(amb)/float64(n+amb), delta, k*delta)
	if n < 5000 {
		t.Fatalf("calibration underpowered: n=%d", n)
	}
	if math.Abs(p-0.50) > 0.03 {
		t.Errorf("D1′ must be a coin flip on driftless tape: p=%.4f outside 0.50±0.03 — the instrument is biased", p)
	}
}

// E3 — an ambiguous episode is FLAGGED, COUNTED and EXCLUDED from the rate,
// never dropped. A rate that silently drops its hard cases lies about its base.
func TestD1PrimeAmbiguousIsExcludedNotDropped(t *testing.T) {
	eps := []TouchOutcome{
		{Outcome: "hold"}, {Outcome: "hold"}, {Outcome: "break"},
		{Outcome: "ambiguous_horizon"}, {Outcome: "ambiguous_span"},
	}
	p, n, amb := HoldRate(eps)
	if n != 3 || amb != 2 {
		t.Fatalf("n must exclude ambiguous but COUNT them: n=%d amb=%d", n, amb)
	}
	if math.Abs(p-2.0/3.0) > 1e-9 {
		t.Errorf("p must be hold/(hold+break) = 0.667, got %.4f", p)
	}
	for _, o := range []string{"ambiguous_horizon", "ambiguous_span"} {
		if !(TouchOutcome{Outcome: o}).IsAmbiguous() {
			t.Errorf("%s must classify as ambiguous", o)
		}
	}
	// And the rendered form can never appear without its base.
	s := FormatHoldRate(eps)
	for _, want := range []string{"p(hold)=0.667", "n=3", "ambiguous=2 excluded", "DESCRIPTIVE ONLY"} {
		if !strings.Contains(s, want) {
			t.Errorf("rendered rate missing %q: %s", want, s)
		}
	}
	t.Logf("rendered: %s", s)
}

// A rate at or above the floor drops the descriptive warning but keeps n.
func TestD1PrimeRateFloorWording(t *testing.T) {
	var eps []TouchOutcome
	for i := 0; i < DetectorRateFloor+50; i++ {
		o := "hold"
		if i%3 == 0 {
			o = "break"
		}
		eps = append(eps, TouchOutcome{Outcome: o})
	}
	s := FormatHoldRate(eps)
	if strings.Contains(s, "DESCRIPTIVE ONLY") {
		t.Errorf("n≥%d must not be labelled descriptive: %s", DetectorRateFloor, s)
	}
	if !strings.Contains(s, "n=") {
		t.Errorf("every rate carries its n: %s", s)
	}
	t.Logf("rendered: %s", s)
}
