package trader

import (
	"errors"
	"testing"
	"time"

	nt "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// F10 (port of #117 da2f76c9) — STALE OR ABSENT EVIDENCE CANNOT AUTHORIZE.
func TestCancelRequiresFreshBookAndReportsSendFailure(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	srv := nt.NewTCPServer(nil)
	at := &AutoTrader{id: "cancel-fixture", trader: ntTrader.NewTCPTrader(srv, "MNQ", "Sim101")}
	row := store.ArmedOrderDB{State: store.StateWorking, SignalID: "entry"}
	calls := 0
	send := func(string) error { calls++; return nil }

	// A stale book (older than the snapshot bound) must not authorize.
	srv.OrderSnapshots().PutAt(nt.OrderSnapshotPayload{Account: "Sim101", Orders: []nt.NT8Order{{Name: "entry", State: "Working"}}},
		now.Add(-snapshotMaxAge()-time.Second))
	if v := at.cancelSafetyFor(row, now); v.Allow {
		t.Errorf("stale book authorized row cancellation: %+v", v)
	}
	if at.cancelSignalIfSafe(send, "entry", "test", now) || calls != 0 {
		t.Errorf("stale book sent cancel; calls=%d", calls)
	}

	// A fresh book still authorizes and the send goes out.
	srv.OrderSnapshots().PutAt(nt.OrderSnapshotPayload{Account: "Sim101", Orders: []nt.NT8Order{{Name: "entry", State: "Working"}}}, now)
	if v := at.cancelSafetyFor(row, now); !v.Allow {
		t.Errorf("fresh resting entry refused: %+v", v)
	}
	if !at.cancelSignalIfSafe(send, "entry", "test", now) || calls != 1 {
		t.Errorf("fresh valid cancellation must send exactly once; calls=%d", calls)
	}

	// A failed send is NOT a completed cancellation.
	if at.cancelSignalIfSafe(func(string) error { return errors.New("synthetic send failure") }, "entry", "test", now) {
		t.Error("failed send reported success")
	}
}
