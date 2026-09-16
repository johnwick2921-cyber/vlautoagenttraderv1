package ninjatrader

import (
	"path/filepath"
	"testing"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// TestBoundAccount_DecoupledFromStreamedCurrent locks the G6 fix source: a
// trader's BoundAccount() is its OWN account and is NEVER the shared streamed
// `currentAccount` (which flaps to whichever account's frame landed last). Two
// same-symbol MNQ traders bound to different accounts must each report their own
// account regardless of what the connection is currently streaming — this is what
// makes currentAccountName() stamp records with the correct owner.
func TestBoundAccount_DecoupledFromStreamedCurrent(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	hoang := NewTCPTrader(s, "MNQ", "Sim101")
	m15 := NewTCPTrader(s, "MNQ", "SimAccount1")

	for _, streamed := range []string{"Sim101", "SimAccount1", ""} {
		s.SetCurrentAccountForTest(streamed)
		if got := hoang.BoundAccount(); got != "Sim101" {
			t.Fatalf("hoang.BoundAccount()=%q while streamed=%q, want Sim101 (must ignore streamed current)", got, streamed)
		}
		if got := m15.BoundAccount(); got != "SimAccount1" {
			t.Fatalf("m15.BoundAccount()=%q while streamed=%q, want SimAccount1 (must ignore streamed current)", got, streamed)
		}
	}

	// An unbound trader reports "" (crypto/legacy) — never borrows the streamed account.
	if got := NewTCPTrader(s, "MNQ").BoundAccount(); got != "" {
		t.Fatalf("unbound BoundAccount()=%q, want \"\"", got)
	}
}

// TestReconcile_ProcessesOwnAccount_SkipsMisstamped locks how the record-ownership
// fix interacts with reconcile: an orphan row stamped with THIS trader's bound
// account is picked up as an orphan candidate (and will orphan-close after the
// grace), while a row stamped with the WRONG account (a pre-fix, mis-attributed
// row) is SKIPPED by the account guard (reconcile.go:107) and therefore SURVIVES.
// This is exactly why the 40 historical phantom rows do NOT auto-clear and need a
// separate guarded cleanup — the fix only guarantees correct attribution going
// forward.
func TestReconcile_ProcessesOwnAccount_SkipsMisstamped(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "recon.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderID = "trader-15m"

	// Correctly-stamped OPEN row on the trader's own account (ancient entry → past
	// orphan grace). NT8 will report SimAccount1 FLAT, so this is an orphan candidate.
	own := &store.TraderPosition{
		TraderID: traderID, Account: "SimAccount1", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29800, EntryTime: 1, Status: "OPEN",
	}
	if err := st.Position().Create(own); err != nil {
		t.Fatalf("create own: %v", err)
	}
	// Mis-stamped OPEN row (pre-fix contamination): SAME trader, but stamped Sim101.
	bad := &store.TraderPosition{
		TraderID: traderID, Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29800, EntryTime: 1, Status: "OPEN",
	}
	if err := st.Position().Create(bad); err != nil {
		t.Fatalf("create bad: %v", err)
	}

	s := ntwire.NewTCPServer(nil)
	// NT8 reports a POSITIVE-but-flat snapshot for SimAccount1 (ok==true, no MNQ held).
	s.SeedPositionsForTest("SimAccount1", []ntwire.OpenPosition{})
	tr := NewTCPTrader(s, "MNQ", "SimAccount1")

	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)

	// Own-account orphan is a candidate: its flat-timer was recorded (deferring to close-sync).
	if tr.flatSince[own.ID] == 0 {
		t.Fatalf("own-account orphan row=%d was not processed (no flat-timer) — reconcile should consider it", own.ID)
	}
	// Mis-stamped row was SKIPPED by the account guard — no flat-timer, and it survives.
	if tr.flatSince[bad.ID] != 0 {
		t.Fatalf("mis-stamped row=%d must be SKIPPED by the account guard, but it was processed", bad.ID)
	}
	if got, _ := st.Position().GetOpenPositions(traderID); len(got) != 2 {
		t.Fatalf("both rows must remain OPEN after one grace-window pass, got %d open", len(got))
	}
}
