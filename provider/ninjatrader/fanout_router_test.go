package ninjatrader

import (
	"testing"
	"time"
)

// TestFanoutRouter_SymbolRouting locks the P5.4 fan-out: a symbol-tagged fill
// reaches ONLY its symbol's subscriber (no cross-trader racing/loss); a
// legacy EMPTY-symbol fill routes to the PRIMARY trading symbol; an
// unsubscribed symbol's payload is dropped (logged) without disturbing others.
func TestFanoutRouter_SymbolRouting(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ") // primary (trader 1)
	s.SetBarsSubscribeSymbol("ES")  // trader 2 registers

	mnq := s.SubscribeFillsFor("MNQ", "")
	es := s.SubscribeFillsFor("ES", "")

	// Inject through the source channel exactly as the readLoop does.
	s.fillCh <- FillPayload{SignalID: "a", Symbol: "ES", Status: "filled"}
	s.fillCh <- FillPayload{SignalID: "b", Symbol: "MNQ", Status: "filled"}
	s.fillCh <- FillPayload{SignalID: "c", Symbol: "", Status: "filled"}    // legacy → primary (MNQ)
	s.fillCh <- FillPayload{SignalID: "d", Symbol: "RTY", Status: "filled"} // no subscriber → dropped

	recv := func(ch <-chan FillPayload, want string) {
		t.Helper()
		select {
		case f := <-ch:
			if f.SignalID != want {
				t.Fatalf("got fill %q, want %q", f.SignalID, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for fill %q", want)
		}
	}
	recv(es, "a")
	recv(mnq, "b")
	recv(mnq, "c") // the legacy-empty fill landed on the primary
	select {
	case f := <-es:
		t.Fatalf("ES must not receive anything else; got %+v", f)
	case <-time.After(150 * time.Millisecond):
	}
}

// TestFanoutRouter_ResubscribeClosesOld locks the reload fix: re-subscribing a
// symbol CLOSES the previous channel (the dead trader instance's consumer
// terminates) and the fresh channel receives subsequent payloads.
func TestFanoutRouter_ResubscribeClosesOld(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")

	old := s.SubscribeFillsFor("MNQ", "")
	fresh := s.SubscribeFillsFor("MNQ", "") // the reload

	select {
	case _, open := <-old:
		if open {
			t.Fatal("the OLD channel must be closed (not delivered to)")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the OLD channel must be closed on resubscribe")
	}

	s.fillCh <- FillPayload{SignalID: "x", Symbol: "MNQ", Status: "filled"}
	select {
	case f := <-fresh:
		if f.SignalID != "x" {
			t.Fatalf("fresh channel got %q, want x", f.SignalID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the FRESH channel must receive post-resubscribe payloads")
	}
}

// TestFanoutRouter_CloseRouting mirrors the fill test for position_close — the
// frame class whose mis-delivery cross-contaminated trade history in the audit.
func TestFanoutRouter_CloseRouting(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	s.SetBarsSubscribeSymbol("ES")
	mnq := s.SubscribeClosesFor("MNQ", "")
	es := s.SubscribeClosesFor("ES", "")

	s.closeCh <- PositionClosePayload{Symbol: "MNQ", PositionSide: "long", Quantity: 1}
	s.closeCh <- PositionClosePayload{Symbol: "ES", PositionSide: "short", Quantity: 2}

	select {
	case p := <-mnq:
		if p.Symbol != "MNQ" {
			t.Fatalf("MNQ subscriber got %q", p.Symbol)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MNQ close not delivered")
	}
	select {
	case p := <-es:
		if p.Symbol != "ES" {
			t.Fatalf("ES subscriber got %q", p.Symbol)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ES close not delivered")
	}
}
