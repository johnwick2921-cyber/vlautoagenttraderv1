package ninjatrader

import (
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"context"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// GAR-F1 (grand-audit response, 2026-08-28) — move_stop for MATERIALIZED
// positions. Live proof of the dead cell: position #566 (a reconcile-
// materialized entry with no Go-side signal) fired auto-breakeven 4× while
// open and every send failed "no open entry to move the stop" because
// MoveStopToBreakeven only knew lastEntrySignalID.
//
// Fix: (a) the armed ledger's signal identity is persisted as the position's
// entry_order_id at materialization/repair, and (b) MoveStopToBreakeven
// resolves through lastEntrySignalID → materialization cache → persisted row.

// TestMaterializedArmedFillCarriesEntryOrderID — the persistence half, twin
// long/short: a filled armed row + a materialized position row → the stamp
// returns the signal id and the row carries it as entry_order_id.
func TestMaterializedArmedFillCarriesEntryOrderID(t *testing.T) {
	for _, tc := range []struct {
		side   string
		entry  float64
		stop   float64
		target float64
		sig    string
	}{
		{"short", 29621.00, 29642.00, 29576.50, "sig-short-1"},
		{"long", 29621.00, 29590.00, 29680.00, "sig-long-1"},
	} {
		st, err := store.New(filepath.Join(t.TempDir(), "f1.db"))
		if err != nil {
			t.Fatal(err)
		}
		if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
			TraderID: "t", PlanID: "2026-08-28:NY:t", Version: 1,
			Session: "NY", Scenario: "S1", Side: tc.side,
			EntryPx: tc.entry, StopPx: tc.stop, TargetPx: tc.target,
			State: "filled", StateReason: "fill", EntryClass: "armed_fill",
			SignalID: tc.sig, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		now := time.Now().UnixMilli()
		pos := &store.TraderPosition{
			TraderID: "t", Symbol: "MNQ", Side: strings.ToUpper(tc.side), Account: "Sim101",
			ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
			EntryPrice: tc.entry, EntryTime: now, Status: "OPEN", Source: "reconcile",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := st.Position().Create(pos); err != nil {
			t.Fatal(err)
		}
		stamped, sig := StampArmedLineageIfMatched(st, "t", pos.ID, "MNQ", strings.ToUpper(tc.side), tc.entry)
		if !stamped || sig != tc.sig {
			t.Fatalf("%s: stamped=%v sig=%q, want true %q", tc.side, stamped, sig, tc.sig)
		}
		open, err := st.Position().GetOpenPositions("t")
		if err != nil || len(open) != 1 {
			t.Fatalf("%s: open rows err=%v len=%d", tc.side, err, len(open))
		}
		if open[0].EntryOrderID != tc.sig {
			t.Fatalf("%s: entry_order_id=%q, want %q", tc.side, open[0].EntryOrderID, tc.sig)
		}
		// Idempotent: a second stamp never overwrites a non-empty identity.
		if err := st.Position().SetEntryOrderID(pos.ID, "other"); err != nil {
			t.Fatal(err)
		}
		open, _ = st.Position().GetOpenPositions("t")
		if open[0].EntryOrderID != tc.sig {
			t.Fatalf("%s: entry_order_id was overwritten: %q", tc.side, open[0].EntryOrderID)
		}
		_ = st.Close()
	}
}

// TestMoveStopUsesMaterializedIdentity — the wire half, twin long/short: a
// materialized OPEN row whose persisted entry_order_id is the only identity
// available (no in-process entry signal, empty cache) → MoveStopToBreakeven
// must resolve it and SEND a move_stop frame carrying that exact SignalID.
// The trailing path shares this funnel (auto_trader_trailing.go →
// MoveStopToBreakeven), so one wire proof covers BE+40 and trail.
func TestMoveStopUsesMaterializedIdentity(t *testing.T) {
	for _, tc := range []struct {
		side string
		be   float64
		sig  string
	}{
		{"LONG", 29611.50, "sig-long-1"},
		{"SHORT", 29611.50, "sig-short-1"},
	} {
		st, err := store.New(filepath.Join(t.TempDir(), "wire.db"))
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now().UnixMilli()
		pos := &store.TraderPosition{
			TraderID: "t", Symbol: "MNQ", Side: tc.side, Account: "Sim101",
			ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
			EntryPrice: 29611.50, EntryTime: now, EntryOrderID: tc.sig,
			Status: "OPEN", Source: "reconcile", CreatedAt: now, UpdatedAt: now,
		}
		if err := st.Position().Create(pos); err != nil {
			t.Fatal(err)
		}

		s := ntwire.NewTCPServer(nil)
		s.SetAddrForTest("127.0.0.1:0")
		ctx, cancel := context.WithCancel(context.Background())
		if err := s.Start(ctx); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = s.Stop() }()
		defer cancel()

		conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		frames := make(chan ntwire.MoveStopPayload, 2)
		go func() {
			for {
				env, err := ntwire.ReadFrame(conn)
				if err != nil {
					return
				}
				if env.Type != ntwire.FrameMoveStop {
					continue
				}
				var p ntwire.MoveStopPayload
				if json.Unmarshal(env.Payload, &p) == nil {
					frames <- p
				}
			}
		}()

		tr := NewTCPTrader(s, "MNQ", "Sim101")
		tr.mu.Lock()
		tr.st = st                            // wired store (StartPositionReconcile does this live)
		tr.entryOrderID = map[string]string{} // empty cache → forces the DB fallback
		tr.mu.Unlock()

		// The server registers the dialed conn in its accept loop (async) — the
		// send can race it. Retry only the client-not-yet-registered error
		// (PRE-REOPEN: this test flaked ~50% under -count=1 before this fix).
		var moveErr error
		for i := 0; i < 50; i++ {
			moveErr = tr.MoveStopToBreakeven(tc.side, tc.be)
			if moveErr == nil || !strings.Contains(moveErr.Error(), "no NT client connected") {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if moveErr != nil {
			t.Fatalf("%s: move-stop with materialized identity failed: %v", tc.side, moveErr)
		}
		select {
		case p := <-frames:
			if p.SignalID != tc.sig {
				t.Fatalf("%s: frame SignalID=%q, want the materialized identity %q", tc.side, p.SignalID, tc.sig)
			}
			if p.NewStopLoss != tc.be {
				t.Fatalf("%s: frame NewStopLoss=%.2f, want %.2f", tc.side, p.NewStopLoss, tc.be)
			}
			if p.Account != "Sim101" {
				t.Fatalf("%s: frame Account=%q, want Sim101 (A2 identity stamp)", tc.side, p.Account)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: no move_stop frame arrived on the loopback", tc.side)
		}
		_ = st.Close()
	}
}

// TestMoveStopStillFailsWithoutAnyIdentity — the honest failure stays: no
// signal, no cache, no persisted identity → the same explicit error (never a
// silent no-op).
func TestMoveStopStillFailsWithoutAnyIdentity(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "none.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	tr.mu.Lock()
	tr.st = st
	tr.entryOrderID = map[string]string{}
	tr.mu.Unlock()
	err = tr.MoveStopToBreakeven("LONG", 29611.50)
	if err == nil || !strings.Contains(err.Error(), "no open entry to move the stop") {
		t.Fatalf("want the explicit no-identity error, got %v", err)
	}
}

// F3 GAP (2026-09-03) — fill_quantity must be stamped on the RECONCILE path.
//
// Found via nofx-89's 2026-09-01 audit: 584 of 586 armed fills carried
// ";stamp_pending", because the fill frame lands before the position row is
// materialized and stampArmedFillLineage returns early on that path. Stamping
// only at fill time covered 2 of 586. Armed row 35 today took the same path and
// still read fill_quantity=0 with the fill-time stamp live.
func TestReconcileStampsFillQuantityOnThePendingPath(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "f3gap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pos := &store.TraderPosition{
		TraderID: "t", Symbol: "MNQ", Side: "short", Quantity: 1,
		EntryPrice: 29285, EntryTime: 1, Status: "OPEN", CreatedAt: 1, UpdatedAt: 1,
	}
	if cErr := st.Position().Create(pos); cErr != nil {
		t.Fatalf("create position: %v", cErr)
	}
	row := store.ArmedOrderDB{
		TraderID: "t", PlanID: "2026-09-03:NY:t", Version: 2, Session: "NY",
		Scenario: "S1", Side: "short", EntryPx: 29285, StopPx: 29362.5, TargetPx: 29130,
		State: "armed", FillPrice: 29285,
	}
	if err := st.ArmedOrders().UpsertArm(&row); err != nil {
		t.Fatalf("arm: %v", err)
	}
	// exactly the marker the fill-time path leaves when it cannot stamp
	if err := st.ArmedOrders().SetState(row.ID, "filled", "armed_fill;stamp_pending"); err != nil {
		t.Fatalf("pending marker: %v", err)
	}

	stamped, _ := StampArmedLineageIfMatched(st, "t", pos.ID, "MNQ", "SHORT", 29285)
	if !stamped {
		t.Fatal("the reconcile must match and stamp this fill")
	}
	rows, lErr := st.ArmedOrders().ListForPlan(row.PlanID)
	if lErr != nil || len(rows) != 1 {
		t.Fatalf("read back: %v n=%d", lErr, len(rows))
	}
	if rows[0].FillQuantity != 1 {
		t.Errorf("fill_quantity = %d, want 1 — the pending path is where 584 of 586 fills go", rows[0].FillQuantity)
	}
	if strings.HasSuffix(rows[0].StateReason, ";stamp_pending") {
		t.Error("the pending marker must be cleared once the stamp completes")
	}
}
