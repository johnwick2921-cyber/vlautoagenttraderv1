package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

// TestWriteAvailableIndicators_OIGatedByEnableOI locks the OI honesty fix: the
// "Open Interest (OI) data" availability line appears IFF EnableOI is true. A
// futures strategy (EnableOI=false via applyFuturesIndicatorDefaults) therefore
// omits it from the prompt; crypto (EnableOI=true) lists it byte-identically as
// before. The user-prompt OI value line is gated by the same EnableOI flag
// (engine_prompt.go), so it follows the same on/off behavior.
func TestWriteAvailableIndicators_OIGatedByEnableOI(t *testing.T) {
	const oiLine = "- Open Interest (OI) data\n"

	withOI := &store.StrategyConfig{Indicators: store.IndicatorConfig{
		Klines:   store.KlineConfig{SelectedTimeframes: []string{"5m"}},
		EnableOI: true,
	}}
	var on strings.Builder
	NewStrategyEngine(withOI).writeAvailableIndicators(&on)
	if !strings.Contains(on.String(), oiLine) {
		t.Fatalf("EnableOI=true must list OI (crypto byte-identical); got:\n%s", on.String())
	}

	withoutOI := &store.StrategyConfig{Indicators: store.IndicatorConfig{
		Klines:   store.KlineConfig{SelectedTimeframes: []string{"5m"}},
		EnableOI: false,
	}}
	var off strings.Builder
	NewStrategyEngine(withoutOI).writeAvailableIndicators(&off)
	if strings.Contains(off.String(), oiLine) {
		t.Fatalf("EnableOI=false (futures default) must NOT list OI; got:\n%s", off.String())
	}
}
