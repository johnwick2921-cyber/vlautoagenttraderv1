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

// TestPlaceEntry_UnboundRefuses locks the multi-account safety rail: an UNBOUND
// trader (empty account) must REFUSE to place an entry — it must NOT fall back to
// the shared active account (which could be another trader's account). A BOUND
// trader passes the empty-account gate (and only then fails on a different gate),
// proving the non-empty path is unchanged.
func TestPlaceEntry_UnboundRefuses(t *testing.T) {
	s := ntwire.NewTCPServer(nil)

	// UNBOUND → hard refuse, before any signal frame is built.
	unbound := NewTCPTrader(s, "MNQ")
	if _, err := unbound.OpenLong("MNQ", 1, 1); err == nil || !strings.Contains(err.Error(), "no bound account") {
		t.Fatalf("unbound OpenLong must refuse with 'no bound account'; got err=%v", err)
	}
	if _, err := unbound.OpenShort("MNQ", 1, 1); err == nil || !strings.Contains(err.Error(), "no bound account") {
		t.Fatalf("unbound OpenShort must refuse with 'no bound account'; got err=%v", err)
	}

	// BOUND → passes the empty-account gate; it then fails on a DIFFERENT gate
	// (account not in the mock's list / no SL-TP), never on 'no bound account'.
	bound := NewTCPTrader(s, "MNQ", "Sim101")
	if _, err := bound.OpenLong("MNQ", 1, 1); err == nil || strings.Contains(err.Error(), "no bound account") {
		t.Fatalf("bound OpenLong must clear the empty-account gate; got err=%v", err)
	}
}
