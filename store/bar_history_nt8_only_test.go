package store

import "testing"

// THE PLANNER TAPE IS NT8-ONLY (CTO ruling under the owner's delegation,
// 2026-09-16): historical_import rows never reach a planner door. The shared
// reader LastNBarsOn hands imports to the CHART (which keeps them, labelled —
// pinned in TestCurrentContractReaderReturnsImportsUnfiltered); this reader is
// the planner's. The imports here are NEWER than the live rows so a reader
// that merely trims the oldest could not pass by accident.
func TestNT8OnlyReaderExcludesImportsTheSharedReaderReturns(t *testing.T) {
	bh := newBarStore(t)
	const oneMin = int64(60_000)
	base := int64(1_789_000_000_000)
	var live, imports []BarHistoryDB
	for i := 0; i < 500; i++ {
		live = append(live, BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: base + int64(i)*oneMin, O: 29290, H: 29300, L: 29280, C: 29295, V: 1, Contract: "MNQ 12-26", Source: BarSourceLive})
	}
	for i := 0; i < 5; i++ { // newer than every live row, no slot collides
		imports = append(imports, BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: base + int64(600+i)*oneMin, O: 29290, H: 29300, L: 29280, C: 29295, V: 1, Contract: "MNQ 12-26", Source: BarSourceHistoricalImport})
	}
	if err := bh.InsertBars(live); err != nil {
		t.Fatal(err)
	}
	if ins, skip, err := bh.ImportBars(imports); err != nil || ins != 5 || skip != 0 {
		t.Fatalf("fixture imports inserted=%d skipped=%d err=%v, want 5/0 (class 128)", ins, skip, err)
	}
	shared, err := bh.LastNBarsOn("MNQ", "1m", "MNQ 12-26", 12000)
	if err != nil || len(shared) != 505 {
		t.Fatalf("premise: the shared reader returns 505 (imports included), got %d err=%v", len(shared), err)
	}
	rows, err := bh.LastNBarsFromNT8On("MNQ", "1m", "MNQ 12-26", 12000)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 500 {
		t.Fatalf("NT8-only reader: want 500 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Source == BarSourceHistoricalImport {
			t.Fatalf("an import row (%d) reached the planner reader", r.OpenTimeMs)
		}
	}
	// the census the boot line prints is a COUNT, read from the store
	n, err := bh.ImportRowsOn("MNQ", "1m", "MNQ 12-26")
	if err != nil || n != 5 {
		t.Fatalf("ImportRowsOn = %d err=%v, want 5", n, err)
	}
}

// The weekly reader builds its weeks from BarsBetweenOn (epoch → now) and its
// Own1m window from the same reader — both planner doors. Same fixture shape.
func TestNT8OnlyRangeReaderExcludesImports(t *testing.T) {
	bh := newBarStore(t)
	const oneMin = int64(60_000)
	base := int64(1_789_000_000_000)
	var live, imports []BarHistoryDB
	for i := 0; i < 100; i++ {
		live = append(live, BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: base + int64(i)*oneMin, O: 1, H: 1, L: 1, C: 1, V: 1, Contract: "MNQ 12-26", Source: BarSourceLive})
	}
	for i := 0; i < 3; i++ {
		imports = append(imports, BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: base + int64(50+i)*oneMin + 30_000, O: 1, H: 1, L: 1, C: 1, V: 1, Contract: "MNQ 12-26", Source: BarSourceHistoricalImport})
	}
	if err := bh.InsertBars(live); err != nil {
		t.Fatal(err)
	}
	if ins, skip, err := bh.ImportBars(imports); err != nil || ins != 3 || skip != 0 {
		t.Fatalf("fixture imports inserted=%d skipped=%d err=%v, want 3/0", ins, skip, err)
	}
	shared, _ := bh.BarsBetweenOn("MNQ", "1m", "MNQ 12-26", 0, base+200*oneMin)
	if len(shared) != 103 {
		t.Fatalf("premise: shared range reader returns 103, got %d", len(shared))
	}
	rows, err := bh.BarsBetweenFromNT8On("MNQ", "1m", "MNQ 12-26", 0, base+200*oneMin)
	if err != nil || len(rows) != 100 {
		t.Fatalf("NT8-only range reader: want 100, got %d err=%v", len(rows), err)
	}
}
