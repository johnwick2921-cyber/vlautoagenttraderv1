package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newBarStore(t *testing.T) *BarHistoryStore {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	bh := st.BarHistory()
	if err := bh.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return bh
}

func mkBar(sym, tf string, tMs int64, c float64) BarHistoryDB {
	// ROLL WAVE — every written bar carries its contract; the fixture names one.
	return BarHistoryDB{Symbol: sym, TF: tf, OpenTimeMs: tMs, O: c, H: c, L: c, C: c, V: 1, Contract: sym + " 09-26", Source: BarSourceLive}
}

func TestBarInsertAndDedup(t *testing.T) {
	st := newBarStore(t)
	rows := []BarHistoryDB{mkBar("MNQ", "1m", 1000, 1.0), mkBar("MNQ", "1m", 2000, 2.0)}
	if err := st.InsertBars(rows); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// NEW (F5): a re-inserted revision UPDATES the row (the natural key is
	// unique) — the old INSERT OR IGNORE kept the stale first write.
	rev := []BarHistoryDB{mkBar("MNQ", "1m", 1000, 1.5)}
	if err := st.InsertBars(rev); err != nil {
		t.Fatalf("re-insert: %v", err)
	}
	n, _ := st.Count()
	if n != 2 {
		t.Fatalf("count = %d want 2 (upsert on the natural key)", n)
	}
	got, err := st.BarsBetween("MNQ", "1m", 0, 3000)
	if err != nil || len(got) != 2 {
		t.Fatalf("BarsBetween = %d rows err=%v", len(got), err)
	}
	for _, b := range got {
		if b.OpenTimeMs == 1000 && b.C != 1.5 {
			t.Fatalf("revision update must win: close=%.2f want 1.5", b.C)
		}
	}
	// BAR-SOURCE WAVE (owner ruling 2026-09-02) — SUPERSEDES the old F5 rule
	// that only 1m was stored. Every TF the NT8 cache holds is persisted: the
	// 1m-only gate silently discarded 383 weekly and 1500 daily bars on every
	// restart, which is exactly the history the weekly reader was starved of.
	if err := st.InsertBars([]BarHistoryDB{mkBar("MNQ", "5m", 3000, 3.0)}); err != nil {
		t.Fatalf("5m insert: %v", err)
	}
	if n, _ := st.Count(); n != 3 {
		t.Fatalf("count after 5m insert = %d want 3 (every TF is stored now)", n)
	}
	if got, err := st.BarsBetween("MNQ", "5m", 0, 4000); err != nil || len(got) != 1 {
		t.Fatalf("the 5m row must be readable back: %d rows err=%v", len(got), err)
	}
	// Integrity now REPORTS the tf set instead of asserting {1m}; dups stay 0.
	if dups, tfs, total, err := st.BarsIntegrity(); err != nil || dups != 0 || total != 3 || len(tfs) != 2 {
		t.Fatalf("integrity = dups:%d tfs:%v total:%d err:%v, want dups 0 / 2 tfs / 3 rows", dups, tfs, total, err)
	}
}

// Per-TF retention (BAR-SOURCE WAVE): 1m follows BAR_RETENTION_DAYS; coarse
// TFs are kept FOREVER. A TF-blind prune would have deleted the deep weekly
// history the moment it was persisted — that is what this pins.
func TestPruneByTFKeepsCoarseHistory(t *testing.T) {
	st := newBarStore(t)
	now := time.Now()
	old1m := now.AddDate(0, 0, -400).UnixMilli()
	oldWeek := now.AddDate(0, 0, -2000).UnixMilli()
	oldDay := now.AddDate(0, 0, -900).UnixMilli()
	fresh1m := now.Add(-time.Hour).UnixMilli()
	if err := st.InsertBars([]BarHistoryDB{
		mkBar("MNQ", "1m", old1m, 1), mkBar("MNQ", "1m", fresh1m, 2),
		mkBar("MNQ", "1w", oldWeek, 3), mkBar("MNQ", "1d", oldDay, 4),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	byTF, err := st.PruneByTF(now)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if byTF["1m"] != 1 {
		t.Fatalf("the 400-day-old 1m bar must be pruned, got %v", byTF)
	}
	if byTF["1w"] != 0 || byTF["1d"] != 0 {
		t.Fatalf("coarse TFs are kept forever, got %v", byTF)
	}
	if w, _ := st.BarsBetween("MNQ", "1w", 0, now.UnixMilli()); len(w) != 1 {
		t.Fatal("the 2000-day-old weekly bar must survive the prune")
	}
	if d, _ := st.BarsBetween("MNQ", "1d", 0, now.UnixMilli()); len(d) != 1 {
		t.Fatal("the 900-day-old daily bar must survive the prune")
	}
	if RetentionDaysFor("1w") != 0 || RetentionDaysFor("1d") != 0 {
		t.Fatal("coarse retention must resolve to keep-forever")
	}
	if RetentionDaysFor("1m") != BarRetentionDays() {
		t.Fatal("1m must keep following BAR_RETENTION_DAYS")
	}
}

func TestBarPrune(t *testing.T) {
	st := newBarStore(t)
	now := time.Now().UnixMilli()
	old := now - 200*24*time.Hour.Milliseconds()
	if err := st.InsertBars([]BarHistoryDB{mkBar("MNQ", "1m", old, 1.0), mkBar("MNQ", "1m", now, 2.0)}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	n, err := st.PruneOlderThan(RetentionCutoffMs(time.Now()))
	if err != nil || n != 1 {
		t.Fatalf("prune = %d err=%v want 1", n, err)
	}
	c, _ := st.Count()
	if c != 1 {
		t.Fatalf("count after prune = %d want 1", c)
	}
}

func TestBarRetentionEnv(t *testing.T) {
	t.Setenv("BAR_RETENTION_DAYS", "7")
	if got := BarRetentionDays(); got != 7 {
		t.Fatalf("retention = %d want 7", got)
	}
	t.Setenv("BAR_RETENTION_DAYS", "")
	if got := BarRetentionDays(); got != 90 {
		t.Fatalf("default retention = %d want 90", got)
	}
}

// TestBarMergeReplayKeepsExistingRows (CLASS 52, owner pin 2026-09-02): a
// 14,000-row 1m table + a 2,000-bar replay must end with 14,000+ rows — the
// replay's INSERT OR REPLACE replaces EXACTLY the bars it returns and never
// deletes rows it cannot replace (the pre-52 backfill wiped the window first
// and a capped provider replay shrank a 14,508-row table to ~2,000).
func TestBarMergeReplayKeepsExistingRows(t *testing.T) {
	st := newBarStore(t)
	const n = 14000
	rows := make([]BarHistoryDB, 0, n)
	base := time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC).UnixMilli()
	for i := 0; i < n; i++ {
		rows = append(rows, mkBar("MNQ", "1m", base+int64(i)*60_000, float64(i)))
	}
	if err := st.InsertBars(rows); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// A 2,000-bar replay overlapping the TAIL of the table (like NT8's capped
	// return for a huge ask).
	const replay = 2000
	replayBars := make([]BarHistoryDB, 0, replay)
	replayBase := base + int64(n-replay)*60_000
	for i := 0; i < replay; i++ {
		replayBars = append(replayBars, mkBar("MNQ", "1m", replayBase+int64(i)*60_000, float64(i)+0.5))
	}
	if err := st.InsertBars(replayBars); err != nil {
		t.Fatalf("replay: %v", err)
	}
	count, _ := st.Count()
	if count < n {
		t.Fatalf("after a %d-bar replay the table holds %d rows — want >= %d (merge, never wipe)", replay, count, n)
	}
	// The rows the replay could NOT replace (outside its range) must survive.
	first, err := st.BarsBetween("MNQ", "1m", base, base+60_000)
	if err != nil || len(first) != 1 || first[0].OpenTimeMs != base {
		t.Fatalf("the earliest pre-replay row was deleted: %+v err=%v", first, err)
	}
	// Inside the replay range the REVISED values win (the upsert works).
	got, _ := st.BarsBetween("MNQ", "1m", replayBase, replayBase+60_000)
	if len(got) != 1 || got[0].C != 0.5 {
		t.Fatalf("replay revision must win inside its range: %+v", got)
	}
}
