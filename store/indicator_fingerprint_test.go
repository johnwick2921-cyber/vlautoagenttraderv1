package store

import "testing"

// W11 — the ai_config fingerprint is deterministic, flips on any indicator-relevant
// change, and treats slice ORDER as significant (periods/timeframes render verbatim).
func TestAIConfigFingerprint(t *testing.T) {
	base := IndicatorConfig{
		EnableEMA: true, EnableATR: true,
		EMAPeriods: []int{20, 50}, ATRPeriods: []int{14},
		Klines: KlineConfig{PrimaryTimeframe: "5m", PrimaryCount: 200, SelectedTimeframes: []string{"5m", "1h", "1d"}},
	}

	// deterministic: same config → same hash.
	if base.AIConfigFingerprint() != base.AIConfigFingerprint() {
		t.Fatal("fingerprint must be deterministic")
	}

	// a toggle flip changes the hash.
	on := base
	on.EnableRSI = true
	if on.AIConfigFingerprint() == base.AIConfigFingerprint() {
		t.Fatal("toggling EnableRSI must change the fingerprint")
	}

	// a period change changes the hash.
	per := base
	per.EMAPeriods = []int{9, 21, 200}
	if per.AIConfigFingerprint() == base.AIConfigFingerprint() {
		t.Fatal("changing EMA periods must change the fingerprint")
	}

	// period ORDER is significant (rendered verbatim).
	ord := base
	ord.EMAPeriods = []int{50, 20}
	if ord.AIConfigFingerprint() == base.AIConfigFingerprint() {
		t.Fatal("period order must be significant")
	}

	// a timeframe change changes the hash.
	tf := base
	tf.Klines.SelectedTimeframes = []string{"5m", "15m", "1h"}
	if tf.AIConfigFingerprint() == base.AIConfigFingerprint() {
		t.Fatal("changing selected timeframes must change the fingerprint")
	}

	// nil vs empty slice must NOT change the hash (omitempty normalizes both).
	a := IndicatorConfig{EnableEMA: true}                     // EMAPeriods nil
	b := IndicatorConfig{EnableEMA: true, EMAPeriods: []int{}} // EMAPeriods empty
	if a.AIConfigFingerprint() != b.AIConfigFingerprint() {
		t.Fatal("nil and empty period slices must hash identically")
	}
}
