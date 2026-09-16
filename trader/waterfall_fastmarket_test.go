package trader

import "testing"

// TestFastMarketWireDefaults — F3 knobs resolve to the shipped defaults.
func TestFastMarketWireDefaults(t *testing.T) {
	if fastMarketATR() != 1.5 {
		t.Fatalf("FAST_MARKET_ATR default want 1.5, got %.2f", fastMarketATR())
	}
	m, e := fastMarketReasoningWire()
	if m != "enabled" || e != "low" {
		t.Fatalf("FAST_MARKET_REASONING default want enabled/low (fast), got %s/%s", m, e)
	}
	if fastMarketReasoningLabel() != "fast→low" {
		t.Fatalf("label want fast→low, got %s", fastMarketReasoningLabel())
	}
}
