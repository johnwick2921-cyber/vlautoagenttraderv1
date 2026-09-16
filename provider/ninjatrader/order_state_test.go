package ninjatrader

import "testing"

// TestUnknownIsNotHistory — the owner's ruling, 2026-09-07:
//
//	"terminalOrderStates containing "unknown": true is wrong — an unreadable
//	 state is not history. UNKNOWN is non-terminal and takes no destructive
//	 branch; a cancel or a reconciliation that meets it does nothing and logs."
//
// Until today `unknown` sat in terminalOrderStates beside `filled` and
// `cancelled`, so an order whose state we could not read vanished from the
// working book — from cutover leg 4's count, from entryIsResting's view, from
// every reader that asks "does this still stand". An order we cannot read is
// the one case where we know least and must assume most.
func TestUnknownIsNotHistory(t *testing.T) {
	for _, st := range []string{"unknown", "Unknown", " UNKNOWN "} {
		if !(NT8Order{State: st}).IsWorking() {
			t.Fatalf("state %q reads as HISTORY — an unreadable state is not a closed order; "+
				"it must stay in the working book so no reader treats it as gone", st)
		}
	}
}

// TestOrderStateVocabulary is the whole vocabulary in one table — the contract
// every classifier in the tree now reads. C3, dispatch 2026-09-07, as ruled.
func TestOrderStateVocabulary(t *testing.T) {
	cases := []struct {
		state string
		want  OrderLiveness
	}{
		{"Initialized", LivenessPending},
		{"Submitted", LivenessPending},
		{"ChangePending", LivenessPending},

		{"Accepted", LivenessLive},
		{"Working", LivenessLive},
		{"Suspended", LivenessLive},

		{"TriggerPending", LivenessLocal},

		{"CancelPending", LivenessDying},
		{"CancelSubmitted", LivenessDying},
		{"cancel_pending", LivenessDying},
		{"cancel-submitted", LivenessDying},

		{"Filled", LivenessTerminal},
		{"Cancelled", LivenessTerminal},
		{"Canceled", LivenessTerminal},
		{"Rejected", LivenessTerminal},
		{"Expired", LivenessTerminal},
		{"partfilled_done", LivenessTerminal},

		{"Unknown", LivenessUnknown},
		{"", LivenessUnknown},
		{"NoSuchStateNT8NeverSends", LivenessUnknown},
	}
	for _, c := range cases {
		if got := ClassifyOrderState(c.state); got != c.want {
			t.Errorf("ClassifyOrderState(%q) = %s, want %s", c.state, got, c.want)
		}
	}
}

// TestOnlyTerminalStatesLeaveTheBook — the flat gate's question. Everything not
// provably history keeps counting, so no reader can conclude an order is gone
// from a state it merely failed to understand.
func TestOnlyTerminalStatesLeaveTheBook(t *testing.T) {
	standing := []string{"Accepted", "Working", "Submitted", "Initialized",
		"CancelPending", "CancelSubmitted", "TriggerPending", "Unknown", ""}
	for _, st := range standing {
		if !(NT8Order{State: st}).IsWorking() {
			t.Errorf("state %q left the working book — only Filled/Cancelled/Rejected/Expired may", st)
		}
	}
	for _, st := range []string{"Filled", "Cancelled", "Rejected", "Expired"} {
		if (NT8Order{State: st}).IsWorking() {
			t.Errorf("terminal state %q is still counted as working", st)
		}
	}
}

// TestProtectionAtTheExchangeIsNarrowerThanStanding — the reconciler's
// question, and the reason one bool could not serve both. A TriggerPending stop
// is held on THIS PC: it counts as standing, and it is not protection.
func TestProtectionAtTheExchangeIsNarrowerThanStanding(t *testing.T) {
	local := NT8Order{State: "TriggerPending"}
	if !local.IsWorking() {
		t.Fatal("a locally-held order still stands and must stay in the book")
	}
	if local.IsLiveAtExchange() {
		t.Fatal("TriggerPending reported as live AT THE EXCHANGE — NT8 is holding it on this PC; " +
			"if the machine is down it will not fire, so it is not protection")
	}
	if !local.IsHeldLocally() {
		t.Fatal("TriggerPending must be identifiable as local-only")
	}
	for _, st := range []string{"CancelPending", "CancelSubmitted"} {
		if (NT8Order{State: st}).IsLiveAtExchange() {
			t.Errorf("dying state %q reported as live protection", st)
		}
	}
	// UNKNOWN is not live — but it is not a "no" either, and callers must be
	// able to tell those apart before acting.
	u := NT8Order{State: "Unknown"}
	if u.IsLiveAtExchange() {
		t.Fatal("an unreadable state reported as live protection")
	}
	if u.IsStateReadable() {
		t.Fatal("an unreadable state must report itself unreadable so no branch is taken on it")
	}
}
