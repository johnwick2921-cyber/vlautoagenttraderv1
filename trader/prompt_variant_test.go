package trader

import "testing"

// TestResolvePromptVariant is the keystone back-compat proof for Strategy
// Studio Phase 2: a saved variant must drive the live loop, and an EMPTY
// variant must resolve EXACTLY as the pre-Phase-2 venue rule (byte-identical),
// so existing strategies (no variant saved) do not change behavior.
func TestResolvePromptVariant(t *testing.T) {
	cases := []struct {
		name     string
		exchange string
		saved    string
		want     string
	}{
		// --- No variant saved → the ORIGINAL venue rule, unchanged ---
		{"crypto, no variant → balanced (byte-identical)", "binance", "", "balanced"},
		{"ninjatrader, no variant → futures (byte-identical)", "ninjatrader", "", "futures"},
		{"blank/whitespace variant → venue rule (crypto)", "bybit", "   ", "balanced"},
		{"blank/whitespace variant → venue rule (futures)", "ninjatrader", "  ", "futures"},

		// --- A saved variant WINS over the venue default ---
		{"crypto + saved aggressive → aggressive", "binance", "aggressive", "aggressive"},
		{"crypto + saved conservative → conservative", "okx", "conservative", "conservative"},
		{"crypto + saved scalping → scalping", "bybit", "scalping", "scalping"},
		{"crypto + saved futures → futures (owner override)", "binance", "futures", "futures"},
		{"futures venue + saved balanced → balanced (owner override)", "ninjatrader", "balanced", "balanced"},
		{"saved variant trimmed of surrounding space", "binance", " aggressive ", "aggressive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolvePromptVariant(tc.exchange, tc.saved); got != tc.want {
				t.Fatalf("resolvePromptVariant(%q, %q) = %q, want %q", tc.exchange, tc.saved, got, tc.want)
			}
		})
	}
}
