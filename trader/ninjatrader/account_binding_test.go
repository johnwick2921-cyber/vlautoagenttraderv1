package ninjatrader

import (
	"encoding/json"
	"strings"
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

// TestSignalAccountWire locks the P5.4 wire rule: an UNBOUND trader's signal
// JSON carries NO "account" key (byte-identical legacy wire → the AddOn uses
// its active account); a BOUND trader's signal carries it (activating the
// AddOn's Phase-3 per-order routing).
func TestSignalAccountWire(t *testing.T) {
	legacy, _ := json.Marshal(ntwire.SignalPayload{Symbol: "MNQ", Side: "long", SignalID: "x"})
	if strings.Contains(string(legacy), "account") {
		t.Fatalf("unbound signal must omit the account key (legacy wire); got %s", legacy)
	}
	bound, _ := json.Marshal(ntwire.SignalPayload{Symbol: "ES", Account: "SimAccount1", Side: "long", SignalID: "y"})
	if !strings.Contains(string(bound), `"account":"SimAccount1"`) {
		t.Fatalf("bound signal must carry the account; got %s", bound)
	}
}

// TestNewTCPTrader_AccountBinding locks the constructor binding: no account →
// unbound (legacy); the variadic account binds (trimmed).
func TestNewTCPTrader_AccountBinding(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	if tr := NewTCPTrader(s, "MNQ"); tr.boundAccount != "" {
		t.Fatalf("no-account constructor must stay unbound; got %q", tr.boundAccount)
	}
	if tr := NewTCPTrader(s, "ES", " SimAccount1 "); tr.boundAccount != "SimAccount1" {
		t.Fatalf("bound constructor must trim+bind; got %q", tr.boundAccount)
	}
}
