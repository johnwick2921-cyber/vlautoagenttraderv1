package store

import (
	"path/filepath"
	"testing"
	"time"
)

// WAVE A / D3 — THE EXIT CAUSE IS RECORDED, NOT INFERRED.
//
// All 58 eligible closed positions carry close_reason='sync' — a word about how
// the ROW was written, not about why the TRADE ended. The cause then had to be
// reconstructed from raw NT8 logs by side/price/nearest-time, and two of the
// exits that reconstruction labelled "stop" (ids 557, 570) were PROFITABLE.
//
// It never needed reconstructing. NT8 sends the cause on the wire
// (provider/ninjatrader/tcp_framing.go:288), Go parses it, and
// trader/ninjatrader/close_sync.go:204 already reads it to arm the re-entry
// cooldown — eight lines after the value is dropped on the way to the column.

func exitCauseStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "exitcause.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// openThenClose runs one full lifecycle through the production builder and
// returns the stored row.
func openThenClose(t *testing.T, st *Store, brokerReason string) *TraderPosition {
	t.Helper()
	pb := NewPositionBuilder(st.Position())
	now := time.Now().UnixMilli()
	if err := pb.ProcessTrade("hoang", "nt8", "futures", "MNQ", "SHORT", "open_short",
		1, 29285, 0, 0, now, "sig-open"); err != nil {
		t.Fatal(err)
	}
	if err := pb.ProcessTradeWithExitReason("hoang", "nt8", "futures", "MNQ", "SHORT", "close_short",
		1, 29355, 0, -140, now+1000, "sig-exit", brokerReason); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Position().GetClosedPositions("hoang", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected exactly one closed row, got %d (%v)", len(rows), err)
	}
	return rows[0]
}

// E6 — the broker's -sl fill becomes 'stop', -tp becomes 'target', and an
// absent broker event becomes 'unknown'. NEVER inferred from price.
func TestCloseReasonComesFromTheBrokerEvent(t *testing.T) {
	for _, tc := range []struct{ broker, want string }{
		{"sl", CloseReasonStop},
		{"tp", CloseReasonTarget},
		{"manual", CloseReasonManual},
		{"", CloseReasonSync}, // the eleven CEX paths: unchanged, byte for byte
	} {
		t.Run("broker="+tc.broker, func(t *testing.T) {
			st := exitCauseStore(t)
			got := openThenClose(t, st, tc.broker)
			if got.CloseReason != tc.want {
				t.Fatalf("E6 RED: broker said %q, the row records %q, expected %q — the cause is being dropped between close_sync.go and the column",
					tc.broker, got.CloseReason, tc.want)
			}
		})
	}
}

// A PROFITABLE STOP IS STILL A STOP. Two live rows (557, 570) exited at a stop
// with positive P&L, so "stopped out" must never be derived from the sign of
// the result — which is exactly what a price/PnL-based reconstruction does.
func TestAProfitableStopIsRecordedAsAStop(t *testing.T) {
	st := exitCauseStore(t)
	pb := NewPositionBuilder(st.Position())
	now := time.Now().UnixMilli()
	if err := pb.ProcessTrade("hoang", "nt8", "futures", "MNQ", "LONG", "open_long",
		1, 29000, 0, 0, now, "sig-open"); err != nil {
		t.Fatal(err)
	}
	// A WINNING exit that the broker says was the protective stop.
	if err := pb.ProcessTradeWithExitReason("hoang", "nt8", "futures", "MNQ", "LONG", "close_long",
		1, 29050, 0, +100, now+1000, "sig-exit", "sl"); err != nil {
		t.Fatal(err)
	}
	rows, _ := st.Position().GetClosedPositions("hoang", 10)
	if len(rows) != 1 || rows[0].CloseReason != CloseReasonStop {
		t.Fatalf("a profitable stop must record %q, got %q (pnl %.2f) — the cause must not follow the sign of the result",
			CloseReasonStop, rows[0].CloseReason, rows[0].RealizedPnL)
	}
}
