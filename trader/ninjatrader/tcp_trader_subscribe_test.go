package ninjatrader

import (
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

// TestNewTCPTrader_DrivesBarsSubscribeSymbol verifies Phase 2: the bar
// subscription tracks the trader's OWN symbol instead of the hardwired default
// "MNQ", so a non-MNQ strategy subscribes to its own root. MNQ is unchanged.
func TestNewTCPTrader_DrivesBarsSubscribeSymbol(t *testing.T) {
	// Default (no trader yet) is the server's hardwired root.
	def := ntwire.NewTCPServer(nil)
	if got := def.BarsSubscribeSymbol(); got != "MNQ" {
		t.Fatalf("default bars symbol = %q, want MNQ", got)
	}

	// A non-MNQ trader drives the subscription to ITS symbol.
	s1 := ntwire.NewTCPServer(nil)
	_ = NewTCPTrader(s1, "ES")
	if got := s1.BarsSubscribeSymbol(); got != "ES" {
		t.Errorf("after NewTCPTrader(ES): bars symbol = %q, want ES", got)
	}

	// MNQ still works (it is just one of the roots now — no regression).
	s2 := ntwire.NewTCPServer(nil)
	_ = NewTCPTrader(s2, "MNQ")
	if got := s2.BarsSubscribeSymbol(); got != "MNQ" {
		t.Errorf("after NewTCPTrader(MNQ): bars symbol = %q, want MNQ", got)
	}

	// Empty symbol keeps the default (defensive — never blanks the subscription).
	s3 := ntwire.NewTCPServer(nil)
	_ = NewTCPTrader(s3, "")
	if got := s3.BarsSubscribeSymbol(); got != "MNQ" {
		t.Errorf("after NewTCPTrader(\"\"): bars symbol = %q, want MNQ (default kept)", got)
	}
}
