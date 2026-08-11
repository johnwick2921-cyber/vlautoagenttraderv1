package ninjatrader

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// recvID waits for a payload on ch and returns whether its SignalID matches want.
func recvFill(t *testing.T, ch <-chan FillPayload, want string) {
	t.Helper()
	select {
	case f := <-ch:
		if f.SignalID != want {
			t.Fatalf("got fill %q, want %q (wrong-account delivery)", f.SignalID, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for fill %q", want)
	}
}

func mustNotReceiveFill(t *testing.T, ch <-chan FillPayload) {
	t.Helper()
	select {
	case f := <-ch:
		t.Fatalf("subscriber received a foreign payload it must NOT: %+v", f)
	case <-time.After(150 * time.Millisecond):
	}
}

// TestFanout_SameSymbolTwoAccounts_RoutesByAccount is the regression lock for the
// H3 class: two traders on the SAME symbol (MNQ) bound to DIFFERENT accounts must
// each receive ONLY their own account's fills / closes / rejects, and the second
// subscription must NOT evict the first's channel. instrument_info (account-less)
// reaches BOTH. Run under -race — the existing fan-out tests use DISTINCT symbols
// and so never exercised the same-symbol collision.
func TestFanout_SameSymbolTwoAccounts_RoutesByAccount(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")

	hoang := s.SubscribeFillsFor("MNQ", "Sim101")
	m15 := s.SubscribeFillsFor("MNQ", "SimAccount1") // same symbol, other account

	// The second (same-symbol) subscribe must NOT have closed hoang's channel.
	select {
	case _, open := <-hoang:
		if !open {
			t.Fatal("hoang's MNQ fill channel was CLOSED by the same-symbol SimAccount1 subscribe — collision not fixed")
		}
	default: // open and empty — correct
	}

	// Fills route to the OWNING account, not the other same-symbol trader.
	s.fillCh <- FillPayload{SignalID: "h1", Symbol: "MNQ", Account: "Sim101", Status: "filled"}
	s.fillCh <- FillPayload{SignalID: "m1", Symbol: "MNQ", Account: "SimAccount1", Status: "filled"}
	recvFill(t, hoang, "h1")
	recvFill(t, m15, "m1")
	mustNotReceiveFill(t, hoang) // hoang never sees SimAccount1's fill
	mustNotReceiveFill(t, m15)   // m15 never sees Sim101's fill

	// A fill for an UNKNOWN account is dropped, never cross-delivered.
	s.fillCh <- FillPayload{SignalID: "ghost", Symbol: "MNQ", Account: "SimAccountX", Status: "filled"}
	mustNotReceiveFill(t, hoang)
	mustNotReceiveFill(t, m15)

	// position_close routes by account too.
	ch := s.SubscribeClosesFor("MNQ", "Sim101")
	cm := s.SubscribeClosesFor("MNQ", "SimAccount1")
	s.closeCh <- PositionClosePayload{Symbol: "MNQ", PositionSide: "long", Quantity: 1, Account: "SimAccount1"}
	select {
	case p := <-cm:
		if p.Account != "SimAccount1" {
			t.Fatalf("close delivered with account %q, want SimAccount1", p.Account)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SimAccount1 close not delivered to its owner")
	}
	select {
	case p := <-ch:
		t.Fatalf("Sim101 close subscriber received SimAccount1's close: %+v", p)
	case <-time.After(150 * time.Millisecond):
	}

	// close_rejected routes by account.
	rh := s.SubscribeRejectsFor("MNQ", "Sim101")
	s.SubscribeRejectsFor("MNQ", "SimAccount1")
	s.rejectCh <- PositionCloseRejectedPayload{Symbol: "MNQ", PositionSide: "long", Account: "Sim101", Reason: "x"}
	select {
	case r := <-rh:
		if r.Account != "Sim101" {
			t.Fatalf("reject account %q, want Sim101", r.Account)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Sim101 reject not delivered to its owner")
	}

	// instrument_info (no account) BROADCASTS to both same-symbol subscribers.
	ih := s.SubscribeInstrumentInfoFor("MNQ", "Sim101")
	im := s.SubscribeInstrumentInfoFor("MNQ", "SimAccount1")
	s.instrCh <- InstrumentInfoPayload{Symbol: "MNQ", Contract: "MNQ 09-26", PointValue: 2, TickSize: 0.25}
	for _, ch := range []<-chan InstrumentInfoPayload{ih, im} {
		select {
		case in := <-ch:
			if in.Symbol != "MNQ" {
				t.Fatalf("instrument_info symbol %q", in.Symbol)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("instrument_info must broadcast to BOTH same-symbol subscribers")
		}
	}
}

// TestFanout_SameSymbol_ConcurrentRace hammers the (symbol,account) fan-out from
// multiple goroutines so `go test -race` exercises the subsMu-guarded dispatch +
// per-account delivery with two same-symbol traders concurrently. The dispatch
// DROPS on a full subscriber channel by design, so this does NOT assert an exact
// receipt count — the invariant it locks is: a consumer NEVER receives a foreign
// account's fill (and the whole thing is race-clean).
func TestFanout_SameSymbol_ConcurrentRace(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	accounts := []string{"Sim101", "SimAccount1"}
	const perAccount = 500

	done := make(chan struct{})
	var consumers sync.WaitGroup
	for _, acct := range accounts {
		acct := acct
		ch := s.SubscribeFillsFor("MNQ", acct)
		consumers.Add(1)
		go func() {
			defer consumers.Done()
			for {
				select {
				case f := <-ch:
					if f.Account != acct {
						t.Errorf("account %s consumer got FOREIGN fill for %s (%s)", acct, f.Account, f.SignalID)
						return
					}
				case <-done:
					return
				}
			}
		}()
	}

	var producers sync.WaitGroup
	for _, acct := range accounts {
		acct := acct
		producers.Add(1)
		go func() {
			defer producers.Done()
			for i := 0; i < perAccount; i++ {
				s.fillCh <- FillPayload{SignalID: fmt.Sprintf("%s-%d", acct, i), Symbol: "MNQ", Account: acct, Status: "filled"}
			}
		}()
	}
	producers.Wait()
	time.Sleep(100 * time.Millisecond) // let the router drain
	close(done)
	consumers.Wait()
}
