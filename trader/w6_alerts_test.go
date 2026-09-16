package trader

import (
	"path/filepath"
	"testing"

	"nofx/store"
)

// W6 — emitAlert is the single production producer for the alert bus (the audit's
// dead wire: AlertStore.Emit had zero callers). It must: (a) no-op when day_plan is
// off, (b) write when on, (c) dedupe by event_id (idempotent bus), (d) feed the
// per-trader ack path.
func TestW6EmitAlertGatedDedupedAckable(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// day_plan OFF → no-op (crypto / no-plan traders must never spam the bus).
	no := false
	off := mkTrader("ninjatrader", &no, "5m")
	off.store = st
	off.id = "t1"
	off.emitAlert("P0", "fill", "fill:1", "Filled", "")
	if rows, _ := st.Alert().List("t1", 10); len(rows) != 0 {
		t.Fatalf("day_plan off must not emit, got %d rows", len(rows))
	}

	// day_plan ON → writes.
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	at.store = st
	at.id = "t1"
	at.emitAlert("P0", "fill", "fill:42", "Filled MNQ", "body")
	rows, _ := st.Alert().List("t1", 10)
	if len(rows) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(rows))
	}
	if rows[0].Level != "P0" || rows[0].Kind != "fill" {
		t.Fatalf("wrong alert fields: %+v", rows[0])
	}

	// dedupe: same event_id → silent no-op (a retried fill/close must not double-alert).
	at.emitAlert("P0", "fill", "fill:42", "Filled MNQ again", "")
	rows, _ = st.Alert().List("t1", 10)
	if len(rows) != 1 {
		t.Fatalf("dedupe by event_id failed, got %d rows", len(rows))
	}

	// distinct event_id → new row.
	at.emitAlert("P0", "close", "close:42", "Closed MNQ", "adherence B")
	rows, _ = st.Alert().List("t1", 10)
	if len(rows) != 2 {
		t.Fatalf("expected 2 alerts after distinct event, got %d", len(rows))
	}

	// ack path is per-trader (IDOR guard): another trader can't ack t1's alert.
	id := rows[0].ID
	if ok, _ := st.Alert().AckForTrader("intruder", id); ok {
		t.Fatal("a foreign trader must not ack t1's alert")
	}
	if ok, _ := st.Alert().AckForTrader("t1", id); !ok {
		t.Fatal("owner must be able to ack own alert")
	}
	n, _ := st.Alert().UnackedCount("t1")
	if n != 1 {
		t.Fatalf("one alert acked → 1 unacked remaining, got %d", n)
	}
}
