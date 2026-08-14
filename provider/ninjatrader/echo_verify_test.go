// A2 (G1) — echo verification: an inbound ack/fill/close whose echoed identity does
// not match the pending op is refused and freezes the owning trader; a matching echo
// (or a pre-v3 seq==0 legacy echo) is processed.
package ninjatrader

import (
	"testing"

	"nofx/discipline"
)

func TestCheckEcho(t *testing.T) {
	pend := pendingOp{traderID: "T1", account: "Sim101", signalID: "sig-1"}

	if acc, mm, _, _ := checkEcho(pend, true, 0, "", ""); !acc || mm {
		t.Fatal("seq==0 (pre-v3 legacy) must be accepted, no mismatch")
	}
	if acc, mm, _, _ := checkEcho(pend, true, 5, "T1", "Sim101"); !acc || mm {
		t.Fatal("a matching echo must be accepted")
	}
	if acc, mm, _, reason := checkEcho(pendingOp{}, false, 9, "T7", "Sim101"); !acc || mm || reason == "" {
		t.Fatalf("an unknown/retired seq must be TOLERATED (accept, no freeze, with a warn reason); acc=%v mm=%v reason=%q", acc, mm, reason)
	}
	if acc, mm, owner, _ := checkEcho(pend, true, 5, "T2", "Sim101"); acc || !mm || owner != "T1" {
		t.Fatalf("a trader_id mismatch must freeze the PENDING owner; owner=%q", owner)
	}
	if acc, mm, _, _ := checkEcho(pend, true, 5, "T1", "Sim102"); acc || !mm {
		t.Fatal("an account mismatch must MISMATCH")
	}
}

func TestEchoRegistry_ProcessMatch_FreezeMismatch(t *testing.T) {
	discipline.ResetFreezeForTest()
	defer discipline.ResetFreezeForTest()
	s := NewTCPServer(nil)

	// Register an op; a matching fill echo processes and does NOT freeze.
	seq := s.assignSeqRegister("T1", "Sim101", "sig-1")
	if seq == 0 {
		t.Fatal("seq must be non-zero after assign")
	}
	if !s.verifyInbound("fill", seq, "T1", "Sim101", "sig-1") {
		t.Fatal("a matching echo must be processed")
	}
	if _, frozen := discipline.IsFrozen("T1"); frozen {
		t.Fatal("a matching echo must NOT freeze")
	}

	// A mismatched echo (wrong account) is NOT processed and FREEZES the trader.
	seq2 := s.assignSeqRegister("T1", "Sim101", "sig-2")
	if s.verifyInbound("fill", seq2, "T1", "Sim999", "sig-2") {
		t.Fatal("a mismatched echo must NOT be processed")
	}
	reason, frozen := discipline.IsFrozen("T1")
	if !frozen {
		t.Fatal("a mismatched echo must FREEZE the owning trader")
	}
	if reason == "" {
		t.Fatal("the freeze must carry a reason")
	}

	// retirePending drops the op.
	s.retirePending(seq, "sig-1")
	if _, ok := s.lookupPending(seq, "sig-1"); ok {
		t.Fatal("retirePending must drop the op")
	}

	// A pre-v3 legacy echo (seq==0) resolves by signal_id and is tolerated.
	s.assignSeqRegister("T2", "Sim102", "sig-3")
	if !s.verifyInbound("fill", 0, "", "", "sig-3") {
		t.Fatal("a legacy seq==0 echo must be tolerated")
	}
	if _, frozen := discipline.IsFrozen("T2"); frozen {
		t.Fatal("a legacy echo must not freeze")
	}
}

// TestEchoRegistry_EntryStaysPendingThroughFill proves the correlation model: a
// FILLED entry keeps its pending op alive (the position_close is keyed by the SAME
// entry signal_id and must still verify), and the close retires it.
func TestEchoRegistry_EntryStaysPendingThroughFill(t *testing.T) {
	discipline.ResetFreezeForTest()
	defer discipline.ResetFreezeForTest()
	s := NewTCPServer(nil)

	seq := s.assignSeqRegister("T1", "Sim101", "entry-1")
	// The entry FILL verifies but does NOT retire (position is now open).
	if !s.verifyInbound("fill", seq, "T1", "Sim101", "entry-1") {
		t.Fatal("a filled entry must verify")
	}
	if _, ok := s.lookupPending(seq, "entry-1"); !ok {
		t.Fatal("a FILLED entry op must stay pending for its eventual close")
	}
	// The position_close (SL/TP/manual) echoes the entry's identity → still verifies.
	if !s.verifyInbound("position_close", seq, "T1", "Sim101", "entry-1") {
		t.Fatal("the position_close must verify against the still-pending entry op")
	}
	s.retirePending(seq, "entry-1") // readLoop retires on position_close
	if _, ok := s.lookupPending(seq, "entry-1"); ok {
		t.Fatal("the close must retire the entry op")
	}
	if _, frozen := discipline.IsFrozen("T1"); frozen {
		t.Fatal("the normal entry→fill→close flow must never freeze")
	}
}

func TestEchoRegistry_SeqIsMonotonic(t *testing.T) {
	s := NewTCPServer(nil)
	a := s.assignSeqRegister("T1", "Sim101", "s1")
	b := s.assignSeqRegister("T1", "Sim101", "s2")
	c := s.assignSeqRegister("T2", "Sim102", "s3")
	if !(a < b && b < c) {
		t.Fatalf("seq must be strictly monotonic: %d %d %d", a, b, c)
	}
}
