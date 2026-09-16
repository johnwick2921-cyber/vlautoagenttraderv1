package store

import (
	"path/filepath"
	"testing"
)

// P4.4 — in-app alert store tests. Lock the dedup-by-event_id invariant (the
// idempotent bus contract), newest-first ordering, the unacked bell count, and
// the ack transition.

func newAlertTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// TestAlertEmitAndList proves a plain emit writes a row and List returns it.
func TestAlertEmitAndList(t *testing.T) {
	st := newAlertTestStore(t)
	as := st.Alert()

	wrote, err := as.Emit(&AlertDB{TraderID: "t1", Level: "P1", Kind: "armed", Title: "S2 armed"})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if !wrote {
		t.Fatal("first emit must write a new row")
	}
	rows, err := as.List("t1", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].Title != "S2 armed" {
		t.Fatalf("list = %+v want one 'S2 armed' row", rows)
	}
}

// TestAlertEmitRequiresTrader: an alert with no trader_id is rejected.
func TestAlertEmitRequiresTrader(t *testing.T) {
	st := newAlertTestStore(t)
	if _, err := st.Alert().Emit(&AlertDB{Level: "P0", Kind: "halt"}); err == nil {
		t.Fatal("emit without trader_id must error")
	}
}

// TestAlertDedupeByEventID is the idempotent-bus proof: the same event_id emitted
// twice writes exactly one row (the second is a silent no-op).
func TestAlertDedupeByEventID(t *testing.T) {
	st := newAlertTestStore(t)
	as := st.Alert()

	a := &AlertDB{TraderID: "t1", Level: "P0", EventID: "fill:cycle-42", Kind: "fill", Title: "filled"}
	wrote1, err := as.Emit(a)
	if err != nil {
		t.Fatalf("emit 1: %v", err)
	}
	wrote2, err := as.Emit(&AlertDB{TraderID: "t1", Level: "P0", EventID: "fill:cycle-42", Kind: "fill", Title: "filled again"})
	if err != nil {
		t.Fatalf("emit 2: %v", err)
	}
	if !wrote1 || wrote2 {
		t.Fatalf("dedupe: wrote1=%v wrote2=%v want true,false", wrote1, wrote2)
	}
	rows, _ := as.List("t1", 10)
	if len(rows) != 1 {
		t.Fatalf("deduped row count = %d want 1", len(rows))
	}
	// A different trader with the same event_id is NOT a duplicate (scoped per trader).
	wrote3, err := as.Emit(&AlertDB{TraderID: "t2", Level: "P0", EventID: "fill:cycle-42", Kind: "fill"})
	if err != nil {
		t.Fatalf("emit 3: %v", err)
	}
	if !wrote3 {
		t.Fatal("same event_id under a different trader must write")
	}
}

// TestAlertListNewestFirst: List orders by id DESC and honors limit.
func TestAlertListNewestFirst(t *testing.T) {
	st := newAlertTestStore(t)
	as := st.Alert()
	for _, title := range []string{"one", "two", "three"} {
		if _, err := as.Emit(&AlertDB{TraderID: "t1", Level: "P2", Kind: "digest", Title: title}); err != nil {
			t.Fatalf("emit %q: %v", title, err)
		}
	}
	rows, err := as.List("t1", 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("limit not honored: got %d want 2", len(rows))
	}
	if rows[0].Title != "three" || rows[1].Title != "two" {
		t.Fatalf("order = %q,%q want three,two (newest first)", rows[0].Title, rows[1].Title)
	}
}

// TestAlertUnackedCountAndAck: unacked count drops after ack; ack is scoped by id.
func TestAlertUnackedCountAndAck(t *testing.T) {
	st := newAlertTestStore(t)
	as := st.Alert()
	for i := 0; i < 3; i++ {
		if _, err := as.Emit(&AlertDB{TraderID: "t1", Level: "P0", Kind: "halt"}); err != nil {
			t.Fatalf("emit: %v", err)
		}
	}
	n, err := as.UnackedCount("t1")
	if err != nil {
		t.Fatalf("unacked: %v", err)
	}
	if n != 3 {
		t.Fatalf("unacked = %d want 3", n)
	}
	rows, _ := as.List("t1", 10)
	if err := as.Ack(rows[0].ID); err != nil {
		t.Fatalf("ack: %v", err)
	}
	n, _ = as.UnackedCount("t1")
	if n != 2 {
		t.Fatalf("unacked after one ack = %d want 2", n)
	}
	// Another trader's count is untouched.
	if m, _ := as.UnackedCount("t2"); m != 0 {
		t.Fatalf("other trader unacked = %d want 0", m)
	}
}

// TestAlertAckForTraderScoping is the IDOR guard: AckForTrader only acks a row
// owned by the caller; a foreign trader_id matches nothing (found=false) and
// leaves the alert unacked.
func TestAlertAckForTraderScoping(t *testing.T) {
	st := newAlertTestStore(t)
	as := st.Alert()
	if _, err := as.Emit(&AlertDB{TraderID: "owner", Level: "P0", Kind: "halt"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	rows, _ := as.List("owner", 10)
	id := rows[0].ID

	// Wrong trader → no row matched, alert stays unacked.
	found, err := as.AckForTrader("attacker", id)
	if err != nil {
		t.Fatalf("ack (wrong trader): %v", err)
	}
	if found {
		t.Fatal("AckForTrader must NOT ack another trader's alert")
	}
	if n, _ := as.UnackedCount("owner"); n != 1 {
		t.Fatalf("alert must remain unacked after foreign ack; unacked=%d want 1", n)
	}

	// Correct trader → acked.
	found, err = as.AckForTrader("owner", id)
	if err != nil {
		t.Fatalf("ack (owner): %v", err)
	}
	if !found {
		t.Fatal("AckForTrader must ack the owner's own alert")
	}
	if n, _ := as.UnackedCount("owner"); n != 0 {
		t.Fatalf("alert must be acked; unacked=%d want 0", n)
	}
}
