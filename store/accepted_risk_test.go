package store

import (
	"path/filepath"
	"testing"
)

// E7 — THE ACCEPTED RECORD IS IMMUTABLE, AND BOTH VALUES STAY READABLE.
//
// Fixture is arm 35, the one trade in this system's history whose broker terms
// were recoverable at all — and only from a rotating Windows log file. The
// ledger says stop 29351.6284728996; NT8 accepted, and later filled, 29355.
// The difference is 3.3715271 points of LEDGER DRIFT, not slippage: the far-side
// protective order was placed at 29355 and filled at 29355.
func TestAcceptedRiskIsAppendOnlyAndKeepsBothValues(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "ar.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ar := st.AcceptedRisk()

	const brokerStop = 29355.0
	const ledgerStop = 29351.6284728996

	accepted := brokerStop
	first := &AcceptedRisk{
		TraderID: "hoang", SignalID: "f2b1eb20", OrderName: "f2b1eb20-sl",
		Symbol: "MNQ", Side: "SHORT", Quantity: 1,
		AcceptedStopPx: &accepted,
		LedgerStopPx:   ledgerStop,
		AcceptedAtMs:   1788442433811,
		BookSource:     "f12",
	}
	if err := ar.Append(first); err != nil {
		t.Fatal(err)
	}

	// THE RE-AUTHORIZATION. A later cycle re-composes the ledger to a different
	// stop. Under the old ledger this OVERWROTE the row and the broker's number
	// was lost. Here it appends.
	reauth := &AcceptedRisk{
		TraderID: "hoang", SignalID: "f2b1eb20", OrderName: "f2b1eb20-sl",
		Symbol: "MNQ", Side: "SHORT", Quantity: 1,
		LedgerStopPx: 29348.0,
		AcceptedAtMs: 1788442533811,
		BookSource:   "f12",
	}
	if err := ar.Append(reauth); err != nil {
		t.Fatal(err)
	}

	rows, err := ar.ForSignal("f2b1eb20")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("a re-authorization must APPEND, not overwrite: got %d row(s), want 2", len(rows))
	}
	// The first record is exactly as written.
	got := rows[0]
	if got.AcceptedStopPx == nil || *got.AcceptedStopPx != brokerStop {
		t.Fatalf("the accepted stop was mutated by a later cycle: got %v, want %.4f — this is the whole defect the table exists to fix", got.AcceptedStopPx, brokerStop)
	}
	if got.LedgerStopPx != ledgerStop {
		t.Fatalf("the ledger value recorded at acceptance was mutated: got %.10f want %.10f", got.LedgerStopPx, ledgerStop)
	}
	// And the drift is a subtraction, not an archaeology.
	drift := *got.AcceptedStopPx - got.LedgerStopPx
	if d := drift - 3.3715271004; d > 1e-6 || d < -1e-6 {
		t.Fatalf("arm 35's ledger drift must read 3.3715271, got %.10f", drift)
	}
	t.Logf("accepted stop %.4f vs ledger %.10f → drift %.7f pts, both readable", *got.AcceptedStopPx, got.LedgerStopPx, drift)

	// A24: an unknown accepted price is NULL, never 0.
	if rows[1].AcceptedStopPx != nil {
		t.Fatalf("an acceptance with no broker book must leave the price NULL, got %v", *rows[1].AcceptedStopPx)
	}
	if n := ar.WithAcceptedStop(); n != 1 {
		t.Fatalf("exactly one row recovered a broker stop, WithAcceptedStop says %d", n)
	}
}
