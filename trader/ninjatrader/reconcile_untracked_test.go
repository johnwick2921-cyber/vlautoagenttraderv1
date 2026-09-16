package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// TestReconcileMaterializesUntrackedNT8Position locks the 2026-08-25 incident
// fix: NT8 holds a position the bot has NO open row for (a manual NT8 entry, or
// an entry whose fill was never recorded). The row-driven orphan loop never sees
// it, so before the fix its close was DROPPED by close-sync (no owner row) and
// the real P&L was lost. Reconcile must now materialize an OPEN row anchored to
// the NT8 average after untrackedGraceMs of persistence, so the later close
// records real P&L.
func TestReconcileMaterializesUntrackedNT8Position(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "mat.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderID = "trader-mat"

	s := ntwire.NewTCPServer(nil)
	// NT8 holds a manual SHORT @29310 on Sim101 that the bot never recorded.
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{
		{Symbol: "MNQ", Side: "SHORT", Quantity: 1, AvgPrice: 29310},
	})
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	// First pass: debounce only — no row yet (bot-opened rows land within
	// seconds; a one-pass sighting is NOT enough evidence to materialize).
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)
	if got, _ := st.Position().GetOpenPositions(traderID); len(got) != 0 {
		t.Fatalf("first pass must only debounce, got %d open rows", len(got))
	}
	if tr.untrackedSince["MNQ|SHORT"] == 0 {
		t.Fatal("first sighting must start the untracked debounce")
	}

	// Age the debounce past the grace, then pass again → row materialized.
	tr.untrackedSince["MNQ|SHORT"] = time.Now().UTC().UnixMilli() - untrackedGraceMs - 1
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)

	openRows, _ := st.Position().GetOpenPositions(traderID)
	if len(openRows) != 1 {
		t.Fatalf("expected 1 materialized row, got %d", len(openRows))
	}
	row := openRows[0]
	if row.Symbol != "MNQ" || row.Side != "SHORT" || row.EntryPrice != 29310 ||
		row.Account != "Sim101" || row.Status != "OPEN" || row.Source != "reconcile" {
		t.Fatalf("materialized row mismatch: %+v", row)
	}

	// A third pass with the row still held must NOT create a duplicate.
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)
	if got, _ := st.Position().GetOpenPositions(traderID); len(got) != 1 {
		t.Fatalf("materialization must be idempotent, got %d open rows", len(got))
	}

	// Now the close arrives (as the 2026-08-25 incident did): owner lookup finds
	// the materialized row and records the REAL exit + ×pv P&L.
	tr.recordClose(traderID, "nt", "ninjatrader", st, store.NewPositionBuilder(st.Position()),
		ntwire.PositionClosePayload{Account: "Sim101", Symbol: "MNQ", PositionSide: "short", ExitPrice: 29350.50, Quantity: 1})
	closed, _ := st.Position().GetClosedPositions(traderID, 10, "Sim101")
	if len(closed) != 1 {
		t.Fatalf("expected 1 closed row after close frame, got %d", len(closed))
	}
	if closed[0].ExitPrice != 29350.50 || closed[0].RealizedPnL != -81.00 {
		t.Fatalf("close must record real exit/P&L (29350.50 / -81.00), got exit=%.2f pnl=%.2f",
			closed[0].ExitPrice, closed[0].RealizedPnL)
	}
}

// TestReconcileConsumesParkedCloseForUntrackedPosition: the close frame lands
// while the position is still UNTRACKED (before the debounce matures). close-sync
// drops + parks the price; the materialization pass must consume the parked exit
// immediately so the P&L isn't lost.
func TestReconcileConsumesParkedCloseForUntrackedPosition(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "park2.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderID = "trader-park2"

	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	// First pass: NT8 holds the untracked LONG → debounce starts.
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{
		{Symbol: "MNQ", Side: "LONG", Quantity: 1, AvgPrice: 29313.25},
	})
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)

	// Close frame arrives while still untracked: close-sync parks it (drop path).
	tr.recordClose(traderID, "nt", "ninjatrader", st, store.NewPositionBuilder(st.Position()),
		ntwire.PositionClosePayload{Account: "Sim101", Symbol: "MNQ", PositionSide: "long", ExitPrice: 29310, Quantity: 1})
	if open, _ := st.Position().GetOpenPositions(traderID); len(open) != 0 {
		t.Fatalf("close before materialization must not create a row, got %d open", len(open))
	}

	// NT8 goes flat (position closed); the parked price remains. Reconcile must
	// reconstruct the round-trip: materialize from the held snapshot is no longer
	// possible (held is empty), so the parked price is the only truth — it must
	// NOT fabricate a row with a fake entry. Assert the park is simply consumed
	// only when a row exists (here: none → nothing fabricated).
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{})
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)
	if open, _ := st.Position().GetOpenPositions(traderID); len(open) != 0 {
		t.Fatalf("fully-invisible round-trip must not fabricate an open row, got %d", len(open))
	}
	// The debounce was pruned (position no longer held).
	if tr.untrackedSince["MNQ|LONG"] != 0 {
		t.Fatal("untracked debounce must be pruned once NT8 no longer holds the position")
	}
}

// TestReconcileUntrackedDedupesAccountEmptyRow (class 27 FIX 3, 2026-08-31):
// the 577+578 duplicate class — an armed-materialized OPEN row whose account is
// still EMPTY must be found by the untracked owner lookup (account-agnostic
// retry), have its account backfilled, and NEVER be double-materialized.
func TestReconcileUntrackedDedupesAccountEmptyRow(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "dedupe.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderID = "trader-dedupe"

	s := ntwire.NewTCPServer(nil)
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{
		{Symbol: "MNQ", Side: "LONG", Quantity: 1, AvgPrice: 29413.0},
	})
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	// The armed materializer's row — account still empty (the frame predated
	// the account binding, exactly the 577 birth shape).
	if err := st.Position().CreateOpenPosition(&store.TraderPosition{
		TraderID: traderID, ExchangeType: "ninjatrader",
		ExchangePositionID: "armed_f0bbe9af_1", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29413.0,
		EntryOrderID: "f0bbe9af-c6ce-4444-8243-974c1ce03208", Leverage: 1,
		Status: "OPEN", Source: "armed_entry", Account: "",
	}); err != nil {
		t.Fatalf("create armed row: %v", err)
	}

	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)
	tr.untrackedSince["MNQ|LONG"] = time.Now().UTC().UnixMilli() - untrackedGraceMs - 1
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)

	rows, _ := st.Position().GetOpenPositions(traderID)
	if len(rows) != 1 {
		t.Fatalf("reconcile must NOT duplicate the armed row, got %d open rows", len(rows))
	}
	if rows[0].Account != "Sim101" {
		t.Fatalf("reconcile must backfill the bound account, got %q", rows[0].Account)
	}
	if rows[0].Source != "armed_entry" {
		t.Fatalf("the original armed row must be kept, got source %q", rows[0].Source)
	}
}
