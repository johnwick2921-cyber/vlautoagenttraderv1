package kernel

import (
	"strings"
	"testing"

	"nofx/market"
	"nofx/store"
)

func sampleMkt() *market.Data {
	return &market.Data{
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"1h": {
				EMAByPeriod: map[int][]float64{20: {15590, 15600}, 200: {15400, 15410}},
				RSIByPeriod: map[int][]float64{14: {55.5, 58.2}},
				ATR14:       42.5,
			},
			"1d": {
				EMAByPeriod: map[int][]float64{200: {14800, 14810}},
				ATR14:       580.25,
			},
		},
	}
}

// W11 — the mirror renders the SAME toggle-gated indicator state the executor shows;
// off-toggles / no-data → empty (the planner omits the block).
func TestRenderPlannerIndicatorBlock(t *testing.T) {
	on := store.IndicatorConfig{EnableEMA: true, EnableRSI: true, EnableATR: true}

	blk := RenderPlannerIndicatorBlock(sampleMkt(), on, []string{"1h", "1d"})
	for _, want := range []string{"### 1h", "EMA20:", "EMA200:", "RSI14:", "ATR14: 42.5000", "### 1d", "ATR14: 580.2500"} {
		if !strings.Contains(blk, want) {
			t.Fatalf("indicator block missing %q\n---\n%s", want, blk)
		}
	}
	// timeframe ORDER honored + only requested TFs.
	if strings.Index(blk, "### 1h") > strings.Index(blk, "### 1d") {
		t.Fatal("timeframes must render in the given order (1h before 1d)")
	}

	// all toggles OFF → nothing renders (disabled state).
	if b := RenderPlannerIndicatorBlock(sampleMkt(), store.IndicatorConfig{}, []string{"1h", "1d"}); b != "" {
		t.Fatalf("no enabled indicator → empty block, got:\n%s", b)
	}
	// no data → empty.
	if b := RenderPlannerIndicatorBlock(&market.Data{}, on, []string{"1h"}); b != "" {
		t.Fatalf("no data → empty block, got:\n%s", b)
	}
	// a requested TF with no data is skipped, not errored.
	if b := RenderPlannerIndicatorBlock(sampleMkt(), on, []string{"5m"}); b != "" {
		t.Fatalf("absent TF → empty, got:\n%s", b)
	}
}

// W11 — BuildPlannerPrompt includes the INDICATORS section only when the block is
// non-empty (disabled state = byte-identical to the pre-W11 prompt).
func TestBuildPlannerPromptIndicatorsGated(t *testing.T) {
	in := samplePlannerInput()

	// disabled: no block → no section.
	if strings.Contains(BuildPlannerPrompt(in), "## Indicators") {
		t.Fatal("empty IndicatorsBlock must NOT emit an Indicators section")
	}

	// enabled: block present → section present, verbatim.
	in.IndicatorsBlock = "### 1h\nEMA20: 15600\nRSI14: 58.2"
	p := BuildPlannerPrompt(in)
	if !strings.Contains(p, "## Indicators") || !strings.Contains(p, "### 1h") || !strings.Contains(p, "RSI14: 58.2") {
		t.Fatalf("enabled: Indicators section/content missing\n---\n%s", p)
	}
	// positioned after Regime, before Ranked levels.
	if !(strings.Index(p, "## Regime") < strings.Index(p, "## Indicators") &&
		strings.Index(p, "## Indicators") < strings.Index(p, "Ranked levels")) {
		t.Fatal("Indicators block must sit between Regime and Ranked levels")
	}
}

// W11 — the executor's FormatIndicatorState and the planner mirror agree byte-for-
// byte for a given series (the extraction is the SAME renderer, not a re-derivation).
func TestPlannerMirrorMatchesExecutorRenderer(t *testing.T) {
	on := store.IndicatorConfig{EnableEMA: true, EnableATR: true}
	td := sampleMkt().TimeframeData["1h"]
	var direct strings.Builder
	FormatIndicatorState(&direct, td, on)
	mirror := RenderPlannerIndicatorBlock(&market.Data{TimeframeData: map[string]*market.TimeframeSeriesData{"1h": td}}, on, []string{"1h"})
	// the mirror is the same body under a "### 1h" header.
	if !strings.Contains(mirror, strings.TrimRight(direct.String(), "\n")) {
		t.Fatalf("mirror body diverges from executor renderer\nexec:\n%s\nmirror:\n%s", direct.String(), mirror)
	}
}
