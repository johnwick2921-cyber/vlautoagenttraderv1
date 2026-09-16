package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// F6 — the backfill on a fixture. Three closed positions: one fully covered by
// 1m bars, one with no bars at all, one whose bars stop half way. The first
// gets numbers; the other two get resolution="none" and keep their NULLs. A
// backfill that guesses is worse than no backfill (A24).
func TestExcursionBackfillCountsAndNoneRows(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// the bars table is migrated on demand (nothing in boot does it for a
	// fresh test store), same as store/bar_history_test.go
	if err := st.BarHistory().Migrate(); err != nil {
		t.Fatalf("migrate bars: %v", err)
	}

	base := int64(1_700_000_000_000)
	base -= base % 60_000
	mk := func(i int64, o, h, l, c float64) store.BarHistoryDB {
		return store.BarHistoryDB{Contract: "MNQ 09-26", Source: store.BarSourceLive, Symbol: "MNQ", TF: "1m", OpenTimeMs: base + i*60_000, O: o, H: h, L: l, C: c}
	}
	// covered: bars 0..4 ; half: bars 0..1 only (its hold runs to bar 4)
	if err := st.BarHistory().InsertBars([]store.BarHistoryDB{
		mk(0, 100, 104, 96, 101), mk(1, 101, 108, 100, 107),
		mk(2, 107, 110, 103, 104), mk(3, 104, 106, 92, 95), mk(4, 95, 99, 94, 98),
	}); err != nil {
		t.Fatalf("insert bars: %v", err)
	}

	ps := st.Position()
	mkPos := func(entryOff, exitOff int64, entryPx, exitPx float64) int64 {
		p := &store.TraderPosition{
			TraderID: "t1", Symbol: "MNQ", Side: "LONG", Quantity: 1,
			EntryPrice: entryPx, EntryTime: base + entryOff, ExitPrice: exitPx,
			ExitTime: base + exitOff, Status: "CLOSED", CloseReason: "stop",
			CreatedAt: base, UpdatedAt: base,
		}
		// PositionStore.Create forces status=OPEN (it is the entry writer), so
		// the fixture closes the row the way the live path does.
		if err := ps.Create(p); err != nil {
			t.Fatalf("create pos: %v", err)
		}
		// ClosePosition stamps its own exit_time (now). That is fine here: the
		// hold then spans from the fixture entry to now, and the bars that
		// exist inside it are exactly the ones under test.
		if _, err := ps.ClosePosition(p.ID, exitPx, "", -10, 0, "stop"); err != nil {
			t.Fatalf("close pos: %v", err)
		}
		_ = exitOff
		return p.ID
	}
	covered := mkPos(30_000, 4*60_000+30_000, 100, 98) // inside bars 0..4
	noBars := mkPos(500*60_000, 504*60_000, 100, 98)   // far past the tape
	_ = noBars

	res, err := BackfillExcursions(st, "MNQ", "t1", time.UnixMilli(base-60_000), time.UnixMilli(base+600*60_000))
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if res.Scanned != 2 {
		t.Fatalf("scanned = %d, want 2 closed positions", res.Scanned)
	}
	if res.Computed != 1 || res.NoCoverage != 1 {
		t.Fatalf("computed=%d no_coverage=%d, want 1 and 1", res.Computed, res.NoCoverage)
	}

	row, err := st.TradeExcursions().GetByPosition(covered)
	if err != nil || row == nil {
		t.Fatalf("covered row: %v", err)
	}
	if row.Resolution != "1m" || row.MAEPts == nil {
		t.Fatalf("covered row must carry a 1m path, got resolution=%q mae=%v", row.Resolution, row.MAEPts)
	}
	// entry 100, lowest low over bars 0..4 is 92 (bar 3) → MAE 8; highest high 110 (bar 2) → MFE 10
	if *row.MAEPts != 8 || *row.MFEPts != 10 {
		t.Errorf("MAE/MFE = %v/%v, want 8/10", *row.MAEPts, *row.MFEPts)
	}
	if row.BarsHeld == nil || *row.BarsHeld != 5 {
		t.Errorf("bars_held = %v, want 5 (entry bar included)", row.BarsHeld)
	}

	uncovered, _ := st.TradeExcursions().GetByPosition(noBars)
	if uncovered == nil {
		t.Fatal("a position with no bars still gets a row — the absence is the record")
	}
	if uncovered.Resolution != "none" {
		t.Errorf("resolution = %q, want \"none\"", uncovered.Resolution)
	}
	if uncovered.MAEPts != nil || uncovered.MFEPts != nil || uncovered.BarsHeld != nil {
		t.Error("a row with no coverage must keep NULLs — never a guessed number")
	}

	// Idempotent: a second run recomputes without duplicating rows.
	res2, err := BackfillExcursions(st, "MNQ", "t1", time.UnixMilli(base-60_000), time.UnixMilli(base+600*60_000))
	if err != nil {
		t.Fatalf("backfill 2: %v", err)
	}
	if res2.Scanned != 2 {
		t.Errorf("second run scanned %d, want 2", res2.Scanned)
	}
	n, _, _, err := st.TradeExcursions().Counts()
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if n != 2 {
		t.Errorf("%d excursion rows after two runs, want 2", n)
	}
}
