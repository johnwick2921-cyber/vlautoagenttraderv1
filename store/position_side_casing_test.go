package store

import (
	"path/filepath"
	"strings"
	"testing"
)

// SIDE-CASING LOOKUP (2026-09-03) — mechanism 2 from nofx-89's §D-9 chain,
// verified against the live store and confirmed.
//
// armed_orders.side is ALWAYS lowercase (armed_executor.go builds it with
// strings.ToLower(sc.Direction)): live counts long 19 / short 17, no uppercase.
// trader_positions.side is overwhelmingly UPPERCASE: LONG 280 / SHORT 304,
// against long 1 / short 2.
//
// GetOpenPositionBySymbol compared `side = ?` — and SQLite's `=` on a plain
// TEXT column is case-sensitive. Measured on the live store:
//
//	select count(*) … where side='short' and id=591  →  0
//	select count(*) … where side='SHORT' and id=591  →  1
//
// So the fill handler, which passes the ARMED row's lowercase side, could never
// find an armed-entry position however well the timing went. The
// materialization race (mechanism 1) merely got there first: position 591
// materialized at 09:05:14, 81s after row 35 filled at 09:03:53.
func TestGetOpenPositionBySymbolIsCaseInsensitiveOnSide(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := st.Position()

	// stored the way the reconcile writes it
	pos := &TraderPosition{
		TraderID: "t1", Symbol: "MNQ", Side: "SHORT", Quantity: 1,
		EntryPrice: 29285, EntryTime: 1, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := ps.Create(pos); err != nil {
		t.Fatalf("create: %v", err)
	}

	// looked up the way an armed row asks for it
	got, err := ps.GetOpenPositionBySymbol("t1", "MNQ", "short")
	if err != nil || got == nil {
		t.Fatalf("a lowercase side must find an uppercase row — this is why the fill-time stamp never landed: err=%v got=%v", err, got)
	}
	if got.ID != pos.ID {
		t.Errorf("found the wrong row: %d want %d", got.ID, pos.ID)
	}

	// and the reverse, since three live rows are stored lowercase
	lower := &TraderPosition{
		TraderID: "t2", Symbol: "MNQ", Side: "long", Quantity: 1,
		EntryPrice: 29285, EntryTime: 1, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := ps.Create(lower); err != nil {
		t.Fatalf("create lower: %v", err)
	}
	if got, err := ps.GetOpenPositionBySymbol("t2", "MNQ", "LONG"); err != nil || got == nil {
		t.Fatalf("an uppercase side must find a lowercase row too: err=%v got=%v", err, got)
	}

	// A genuinely different side still misses — the fix must not match everything.
	if got, _ := ps.GetOpenPositionBySymbol("t1", "MNQ", "long"); got != nil {
		t.Errorf("a LONG lookup must not find the SHORT row, got id %d", got.ID)
	}
}

// WRITE CHOKEPOINT (owner ruling 2026-09-03) — armed_orders stores UPPERCASE
// from now on, so the two tables stop disagreeing at the source.
//
// UPPER(side)=UPPER(?) fixes the READ. Canonicalizing at the write is class 28
// proper: "one canonicalizer per identifier, called where the value ENTERS,
// never at each comparison". Existing lowercase rows keep working through the
// fold-insensitive read; new ones do not need it.
func TestArmedOrderSideIsCanonicalizedAtWrite(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	for _, in := range []string{"short", "SHORT", " Short ", "long"} {
		row := ArmedOrderDB{
			TraderID: "t1", PlanID: "p:" + in, Version: 1, Session: "NY",
			Scenario: "S1", Side: in, EntryPx: 29285, StopPx: 29362.5,
			TargetPx: 29130, State: "armed",
		}
		if err := st.ArmedOrders().UpsertArm(&row); err != nil {
			t.Fatalf("upsert %q: %v", in, err)
		}
		got, err := st.ArmedOrders().ListForPlan(row.PlanID)
		if err != nil || len(got) != 1 {
			t.Fatalf("read back %q: %v n=%d", in, err, len(got))
		}
		want := strings.ToUpper(strings.TrimSpace(in))
		if got[0].Side != want {
			t.Errorf("stored side for input %q = %q, want %q — canonicalize where the value ENTERS", in, got[0].Side, want)
		}
	}
}
