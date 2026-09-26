package ninjatrader

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// G1 (CTO review part 2) — the three OrderedOwned skip guards are pinned at the
// production call sites: with an owner installed AND the legacy advisory
// consumers running, one frame must apply exactly once. Each pin's RED is the
// removal of one skip guard (the close one is the fill-row count).

type orderedOnceFixture struct {
	t    *testing.T
	tr   *TCPTrader
	st   *store.Store
	srv  *ntwire.TCPServer
	conn net.Conn
}

func newOrderedOnceFixture(t *testing.T) *orderedOnceFixture {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "once.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel(); _ = st.Close() })
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: ntwire.MinAddonBuildPictureHtf}); err != nil {
		t.Fatal(err)
	}
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	return &orderedOnceFixture{t: t, tr: tr, st: st, srv: s, conn: conn}
}

func (f *orderedOnceFixture) countFills(exchangeOrderID string) int64 {
	var n int64
	if err := f.st.GormDB().Table("trader_fills").Where("exchange_order_id = ?", exchangeOrderID).Count(&n).Error; err != nil {
		f.t.Fatalf("count fills: %v", err)
	}
	return n
}

// One position_close frame → exactly ONE exit fill row. A PARTIAL close over
// a 2-lot row is the deterministic shape: the worker reduces the exact qty (the
// row STAYS OPEN with residual 1), so an un-guarded advisory consumer always
// finds an OPEN row and full-applies a SECOND time (a second fill + a whole-row
// close). The advisory close-sync loop is RUNNING (StartCloseSync) — it must
// skip the owned frame.
func TestOneCloseFrameAppliesExactlyOnce(t *testing.T) {
	f := newOrderedOnceFixture(t)
	tr, st := f.tr, f.st
	tr.StartCloseSync("t1", "ex", "ninjatrader", st) // the legacy advisory consumer
	unreg, err := tr.InstallOrderedExecutions("t1", "ex", "ninjatrader", st, nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	defer unreg()

	now := time.Now().UTC().UnixMilli()
	pos := &store.TraderPosition{
		TraderID: "t1", ExchangeType: "ninjatrader", ExchangePositionID: "p-g1",
		Symbol: "MNQ", Side: "LONG", Quantity: 2, EntryQuantity: 2,
		EntryPrice: 29350, EntryTime: now - 60_000, EntryOrderID: "sig-g1",
		Leverage: 1, Status: "OPEN", Source: "armed_entry", Account: "Sim101",
		CreatedAt: now - 60_000, UpdatedAt: now - 60_000,
	}
	if err := st.Position().CreateOpenPosition(pos); err != nil {
		t.Fatal(err)
	}
	if err := ntwire.WriteFrame(f.conn, ntwire.FramePositionClose, ntwire.PositionClosePayload{
		SignalID: "sig-g1", Symbol: "MNQ", PositionSide: "long", Account: "Sim101",
		Quantity: 1, ExitPrice: 29360, ExitReason: "sl", ExitTime: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	// Wait until the ordered worker has provably applied (one fill exists).
	deadline := time.Now().Add(5 * time.Second)
	for f.countFills("sig-g1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	// Give the advisory consumer its full chance to double-apply, then count.
	time.Sleep(300 * time.Millisecond)
	if n := f.countFills("sig-g1"); n != 1 {
		t.Fatalf("one close frame must produce exactly ONE exit fill, got %d — the advisory consumer double-applied", n)
	}
	row, err := st.Position().GetOpenPositionBySymbol("t1", "MNQ", "LONG")
	if err != nil || row == nil {
		t.Fatalf("the partial close must leave the row OPEN with residual 1, got %+v err=%v", row, err)
	}
	if row.Quantity != 1 {
		t.Fatalf("the partial close must reduce exactly the frame's qty (residual 1), got %.0f — the advisory consumer closed the whole row", row.Quantity)
	}
	pending, _ := st.Position().PendingNT8Exits("Sim101")
	if len(pending) != 0 {
		t.Fatalf("no receipt may be parked when the worker applied cleanly, got %d", len(pending))
	}
}

// One fill frame → the pending marker resolves ONCE and the recent-fill ring
// records the fill ONCE (the legacy fill goroutine runs and must skip the
// owned frame; its double-apply would ring the fill twice).
func TestOneFillResolvesThePendingMarkerOnce(t *testing.T) {
	f := newOrderedOnceFixture(t)
	tr, st := f.tr, f.st
	unreg, err := tr.InstallOrderedExecutions("t1", "ex", "ninjatrader", st, nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	defer unreg()

	tr.pendingMu.Lock()
	tr.pending["sig-g3"] = "Sim101"
	tr.pendingAt["sig-g3"] = time.Now().UnixMilli()
	tr.pendingMu.Unlock()
	if err := ntwire.WriteFrame(f.conn, ntwire.FrameFill, ntwire.FillPayload{
		SignalID: "sig-g3", Symbol: "MNQ", Account: "Sim101", Side: "long",
		Quantity: 1, Status: "filled", FillPrice: 29355,
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		tr.pendingMu.Lock()
		_, pending := tr.pending["sig-g3"]
		tr.pendingMu.Unlock()
		tr.mu.Lock()
		n := len(tr.recentFills)
		tr.mu.Unlock()
		if !pending && n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the fill never applied: pending=%v recentFills=%d", pending, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	tr.mu.Lock()
	n := len(tr.recentFills)
	tr.mu.Unlock()
	if n != 1 {
		t.Fatalf("one fill frame must ring the recent-fill record ONCE, got %d — the advisory consumer double-applied", n)
	}
	tr.pendingMu.Lock()
	_, still := tr.pending["sig-g3"]
	tr.pendingMu.Unlock()
	if still {
		t.Fatal("the pending marker must be resolved")
	}
}
