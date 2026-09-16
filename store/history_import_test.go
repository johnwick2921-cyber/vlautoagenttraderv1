package store

import (
	"strings"
	"testing"
	"time"
)

// ── HISTORY IMPORT pins (wave 101, 2026-09-11) ───────────────────────────────

func mkImport(sym, contract, tf string, tMs int64, c float64) BarHistoryDB {
	return BarHistoryDB{Symbol: sym, TF: tf, OpenTimeMs: tMs, O: c, H: c, L: c, C: c, V: 1,
		Convention: "epoch_floor", Contract: contract, Source: BarSourceHistoricalImport}
}

// E1 — THE OVERWRITE PIN. An import whose key collides with an existing bar
// keeps the existing values and counts the skip. There is NO upsert in
// ImportBars: ON CONFLICT DO NOTHING is the whole write path (the 09-10 damage
// was an import-shaped write overwriting live tape).
func TestImportBarsNeverOverwritesExistingRow(t *testing.T) {
	st := newBarStore(t)
	if err := st.InsertBars([]BarHistoryDB{mkBar("MNQ", "1m", 1000, 100.0)}); err != nil {
		t.Fatalf("seed live: %v", err)
	}
	ins, skp, err := st.ImportBars([]BarHistoryDB{mkImport("MNQ", "MNQ 09-23", "1m", 1000, 999.0)})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if ins != 0 || skp != 1 {
		t.Fatalf("imported=%d skipped=%d want 0/1 — a colliding import must skip, never overwrite", ins, skp)
	}
	rows, err := st.BarsBetween("MNQ", "1m", 0, 2000)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if rows[0].C != 100.0 || rows[0].Source != BarSourceLive || rows[0].Contract != "MNQ 09-26" {
		t.Fatalf("existing row changed by the import: %+v", rows[0])
	}
}

// E2 — THE STAMP PIN. Every imported bar has a non-NULL contract and
// source='historical_import'; anything else is rejected by the import, never
// written.
func TestImportBarsRefusesUnstampedRows(t *testing.T) {
	st := newBarStore(t)
	if _, _, err := st.ImportBars([]BarHistoryDB{mkImport("MNQ", "", "1m", 1000, 1.0)}); err == nil {
		t.Fatalf("empty contract must be refused")
	}
	if _, _, err := st.ImportBars([]BarHistoryDB{{Symbol: "MNQ", TF: "1m", OpenTimeMs: 1000, Contract: "MNQ 09-23", Source: BarSourceLive}}); err == nil {
		t.Fatalf("a live-source row must not enter through the import door")
	}
	if n, _ := st.Count(); n != 0 {
		t.Fatalf("refused rows were written: n=%d", n)
	}
}

// E2b — a good row writes stamped and is readable back by contract.
func TestImportBarsWritesStampedRows(t *testing.T) {
	st := newBarStore(t)
	ins, skp, err := st.ImportBars([]BarHistoryDB{mkImport("MNQ", "MNQ 09-23", "1m", 1000, 1.0)})
	if err != nil || ins != 1 || skp != 0 {
		t.Fatalf("ins=%d skp=%d err=%v", ins, skp, err)
	}
	rows, err := st.BarsBetweenOn("MNQ", "1m", "MNQ 09-23", 0, 2000)
	if err != nil || len(rows) != 1 {
		t.Fatalf("BarsBetweenOn rows=%d err=%v", len(rows), err)
	}
	if rows[0].Contract != "MNQ 09-23" || rows[0].Source != BarSourceHistoricalImport || rows[0].Convention != "epoch_floor" {
		t.Fatalf("stamp wrong: %+v", rows[0])
	}
	if !IsBacktestReadable(rows[0].Source) {
		t.Fatalf("imported rows must be backtest-readable")
	}
	if IsReadableSource(rows[0].Source) {
		t.Fatalf("imported rows must NOT be live-readable — the live path never sees imported history")
	}
}

// E3 — THE SEAM PIN. (a) The live contract filter returns only the asked
// contract; (b) a deliberately continuous series is labelled adjusted and the
// store refuses to persist it, so it cannot be mistaken for raw.
func TestImportSeamContractFilterAndContinuousLabel(t *testing.T) {
	st := newBarStore(t)
	if err := st.InsertBars([]BarHistoryDB{mkBar("MNQ", "1m", 1000, 10.0)}); err != nil {
		t.Fatalf("seed live 09-26: %v", err)
	}
	if _, _, err := st.ImportBars([]BarHistoryDB{mkImport("MNQ", "MNQ 09-23", "1m", 2000, 20.0)}); err != nil {
		t.Fatalf("import: %v", err)
	}
	only, err := st.BarsBetweenOn("MNQ", "1m", "MNQ 09-23", 0, 3000)
	if err != nil || len(only) != 1 || only[0].Contract != "MNQ 09-23" {
		t.Fatalf("BarsBetweenOn(09-23) = %d rows, want exactly the 09-23 bar", len(only))
	}
	cur, err := st.BarsBetweenOn("MNQ", "1m", "MNQ 09-26", 0, 3000)
	if err != nil || len(cur) != 1 || cur[0].Contract != "MNQ 09-26" {
		t.Fatalf("BarsBetweenOn(09-26) = %d rows, want exactly the 09-26 bar", len(cur))
	}

	// Continuous series: explicit basis, labelled, never persistable.
	segs := []ContinuousSegment{
		{Contract: "MNQ 09-23", Bars: []BarHistoryDB{mkImport("MNQ", "MNQ 09-23", "1m", 100, 100.0)}},
		{Contract: "MNQ 12-23", Bars: []BarHistoryDB{mkImport("MNQ", "MNQ 12-23", "1m", 200, 400.0)}},
	}
	if _, err := BuildContinuous(segs, []float64{}); err == nil {
		t.Fatalf("a missing basis must be a refusal, not a guess")
	}
	adj, err := BuildContinuous(segs, []float64{290.0})
	if err != nil || len(adj) != 2 {
		t.Fatalf("BuildContinuous err=%v rows=%d", err, len(adj))
	}
	if adj[1].C != 110.0 { // 400 − 290 basis
		t.Fatalf("adjusted close=%.2f want 110.0", adj[1].C)
	}
	for _, b := range adj {
		if b.Source != BarSourceContinuous || b.Contract != ContractMixed {
			t.Fatalf("adjusted row not labelled: %+v", b)
		}
	}
	if _, _, err := st.ImportBars(adj); err == nil {
		t.Fatalf("an adjusted series must be refused by the import door")
	}
	if err := st.InsertBars(adj); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("an adjusted series must be refused by the live writer, got %v", err)
	}
}

// E4 — THE PRUNE PIN. Imported history survives the retention sweep; live rows
// past the cutoff do not.
func TestPruneNeverDeletesImportedHistory(t *testing.T) {
	st := newBarStore(t)
	old := time.Now().AddDate(0, 0, -200).UnixMilli()
	if err := st.InsertBars([]BarHistoryDB{mkBar("MNQ", "1m", old, 1.0)}); err != nil {
		t.Fatalf("seed live old: %v", err)
	}
	if _, _, err := st.ImportBars([]BarHistoryDB{mkImport("MNQ", "MNQ 09-22", "1m", old+60_000, 2.0)}); err != nil {
		t.Fatalf("seed import old: %v", err)
	}
	deleted, err := st.PruneByTF(time.Now())
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if deleted["1m"] != 1 {
		t.Fatalf("prune deleted %d live rows, want 1", deleted["1m"])
	}
	left, err := st.BarsBetween("MNQ", "1m", 0, time.Now().UnixMilli())
	if err != nil || len(left) != 1 || left[0].Source != BarSourceHistoricalImport {
		t.Fatalf("imported history must survive the sweep, left=%d %+v", len(left), left)
	}
}
