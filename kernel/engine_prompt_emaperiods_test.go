package kernel

import (
	"strings"
	"testing"

	"nofx/market"
	"nofx/store"
)

// TestFormatTimeframeSeriesData_EMAPeriods locks the prompt EMA labeling:
//   - no configured periods → legacy "EMA20:"/"EMA50:" (byte-identical fallback);
//   - configured [20,50] with the same values → byte-identical to legacy;
//   - configured [9,21,200] → "EMA9:/EMA21:/EMA200:" (sorted), never EMA20/EMA50.
func TestFormatTimeframeSeriesData_EMAPeriods(t *testing.T) {
	eng := NewStrategyEngine(&store.StrategyConfig{})
	ind := store.IndicatorConfig{EnableEMA: true}
	ema20 := []float64{100, 101, 102}
	ema50 := []float64{90, 91, 92}

	var legacy strings.Builder
	eng.formatTimeframeSeriesData(&legacy, &market.TimeframeSeriesData{
		EMA20Values: ema20, EMA50Values: ema50,
	}, ind)
	if !strings.Contains(legacy.String(), "EMA20:") || !strings.Contains(legacy.String(), "EMA50:") {
		t.Fatalf("legacy must emit EMA20/EMA50; got:\n%s", legacy.String())
	}

	// Configured [20,50] with identical values → byte-identical to the legacy output.
	var cfg2050 strings.Builder
	eng.formatTimeframeSeriesData(&cfg2050, &market.TimeframeSeriesData{
		EMA20Values: ema20, EMA50Values: ema50,
		EMAByPeriod: map[int][]float64{20: ema20, 50: ema50},
	}, ind)
	if cfg2050.String() != legacy.String() {
		t.Fatalf("configured [20,50] must be byte-identical to legacy\nlegacy:\n%q\ngot:\n%q", legacy.String(), cfg2050.String())
	}

	// Configured [9,21,200] → real labels, sorted ascending, no legacy labels.
	var cfg strings.Builder
	eng.formatTimeframeSeriesData(&cfg, &market.TimeframeSeriesData{
		EMA20Values: ema20, EMA50Values: ema50, // present but MUST be ignored
		EMAByPeriod: map[int][]float64{21: {1, 2}, 9: {3, 4}, 200: {5}},
	}, ind)
	out := cfg.String()
	for _, want := range []string{"EMA9:", "EMA21:", "EMA200:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("configured must emit %s; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "EMA20:") || strings.Contains(out, "EMA50:") {
		t.Fatalf("configured must NOT emit legacy EMA20/EMA50; got:\n%s", out)
	}
	if !(strings.Index(out, "EMA9:") < strings.Index(out, "EMA21:") &&
		strings.Index(out, "EMA21:") < strings.Index(out, "EMA200:")) {
		t.Fatalf("EMA periods must be sorted ascending; got:\n%s", out)
	}
}

// TestFormatTimeframeSeriesData_BollRsiAtrLabels locks the RSI/ATR/BOLL labels:
// configured periods → RSI9/RSI21, ATR10, BOLL10/BOLL12; none → legacy labels.
func TestFormatTimeframeSeriesData_BollRsiAtrLabels(t *testing.T) {
	eng := NewStrategyEngine(&store.StrategyConfig{})
	ind := store.IndicatorConfig{EnableRSI: true, EnableATR: true, EnableBOLL: true}

	// Legacy (no *ByPeriod) → fixed labels, byte-identical to prior output.
	var legacy strings.Builder
	eng.formatTimeframeSeriesData(&legacy, &market.TimeframeSeriesData{
		RSI7Values: []float64{50, 51}, RSI14Values: []float64{55, 56},
		ATR14:     12.5,
		BOLLUpper: []float64{1, 2}, BOLLMiddle: []float64{3, 4}, BOLLLower: []float64{5, 6},
	}, ind)
	for _, w := range []string{"RSI7:", "RSI14:", "ATR14:", "BOLL Upper:", "BOLL Middle:", "BOLL Lower:"} {
		if !strings.Contains(legacy.String(), w) {
			t.Fatalf("legacy must emit %s; got:\n%s", w, legacy.String())
		}
	}

	// Configured → period-labeled, no legacy labels.
	var cfg strings.Builder
	eng.formatTimeframeSeriesData(&cfg, &market.TimeframeSeriesData{
		RSI7Values: []float64{50}, RSI14Values: []float64{55}, // present but ignored
		ATR14: 12.5,
		RSIByPeriod: map[int][]float64{21: {60, 61}, 9: {40, 41}},
		ATRByPeriod: map[int]float64{10: 9.9},
		BOLLByPeriod: map[int]*market.BollBands{
			12: {Upper: []float64{1}, Middle: []float64{2}, Lower: []float64{3}},
			10: {Upper: []float64{4}, Middle: []float64{5}, Lower: []float64{6}},
		},
	}, ind)
	out := cfg.String()
	for _, w := range []string{"RSI9:", "RSI21:", "ATR10:", "BOLL10 Upper:", "BOLL12 Upper:"} {
		if !strings.Contains(out, w) {
			t.Fatalf("configured must emit %s; got:\n%s", w, out)
		}
	}
	for _, bad := range []string{"RSI7:", "RSI14:", "ATR14:", "BOLL Upper:"} {
		if strings.Contains(out, bad) {
			t.Fatalf("configured must NOT emit legacy %s; got:\n%s", bad, out)
		}
	}
}

// TestFormatMarketData_CurrentEMAByPeriod locks the summary line: configured
// periods → "current_ema9/21/200 ="; none → legacy "current_ema20 =".
func TestFormatMarketData_CurrentEMAByPeriod(t *testing.T) {
	eng := NewStrategyEngine(&store.StrategyConfig{Indicators: store.IndicatorConfig{EnableEMA: true}})

	legacy := eng.formatMarketData(&market.Data{Symbol: "MNQ", CurrentPrice: 100, CurrentEMA20: 99})
	if !strings.Contains(legacy, "current_ema20 =") {
		t.Fatalf("legacy must emit current_ema20; got:\n%s", legacy)
	}

	cfg := eng.formatMarketData(&market.Data{
		Symbol: "MNQ", CurrentPrice: 100, CurrentEMA20: 99,
		CurrentEMAByPeriod: map[int]float64{21: 98, 9: 97, 200: 95},
	})
	for _, want := range []string{"current_ema9 =", "current_ema21 =", "current_ema200 ="} {
		if !strings.Contains(cfg, want) {
			t.Fatalf("configured must emit %s; got:\n%s", want, cfg)
		}
	}
	if strings.Contains(cfg, "current_ema20 =") {
		t.Fatalf("configured must NOT emit current_ema20; got:\n%s", cfg)
	}
}
