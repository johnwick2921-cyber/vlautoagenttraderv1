package store

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ── W-BARS-CONTRACT-KEY — the migration, its guards, and the readers ─────────

// legacyBarsDDL is the live 2026-09-18 shape, byte-for-byte from sqlite_master
// (PRAGMA on /home/hoang/nofx/data/data.db, read-only), plus its three indexes.
var legacyBarsDDL = []string{
	"CREATE TABLE `bars` (`symbol` text,`tf` text,`open_time_ms` integer,`o` real,`h` real,`l` real,`c` real,`v` real, `convention` text, `contract` text, `source` text,PRIMARY KEY (`symbol`,`tf`,`open_time_ms`))",
	"CREATE UNIQUE INDEX idx_bars_sym_tf_time_unique ON bars(symbol, tf, open_time_ms)",
	"CREATE INDEX idx_bars_contract ON bars(symbol, tf, contract, open_time_ms)",
	"CREATE INDEX idx_bars_source ON bars(symbol, tf, source, open_time_ms)",
}

// newLegacyBarStore opens a fresh store with the PRE-WAVE bars table and seeds
// it the way the old writers left it: September live rows, and December rows
// only where September had none (the CLASS 143 stray shape). Migrate is NOT
// called. Returns the store and the seeded row count.
func newLegacyBarStore(t *testing.T) (*BarHistoryStore, int64) {
	t.Helper()
	t.Setenv("BARS_KEY_BACKUP_DIR", t.TempDir())
	barsKeyed.Store(false)
	st, err := New(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	bh := st.BarHistory()
	for _, ddl := range legacyBarsDDL {
		if err := bh.db.Exec(ddl).Error; err != nil {
			t.Fatalf("legacy ddl: %v", err)
		}
	}
	base := int64(1_789_000_000_000)
	ins := "INSERT INTO bars(symbol, tf, open_time_ms, o, h, l, c, v, convention, contract, source) VALUES (?,?,?,?,?,?,?,?,?,?,?)"
	var n int64
	for i := 0; i < 100; i++ {
		if i%10 == 7 { // holes September never held
			continue
		}
		if err := bh.db.Exec(ins, "MNQ", "1m", base+int64(i)*60_000, 29000.0, 29010.0, 28990.0, 29005.0, 1.0, "epoch_floor", "MNQ 09-26", BarSourceLive).Error; err != nil {
			t.Fatalf("seed sep: %v", err)
		}
		n++
	}
	for i := 0; i < 100; i++ {
		if i%10 != 7 {
			continue // the December overlap the old key DROPPED
		}
		if err := bh.db.Exec(ins, "MNQ", "1m", base+int64(i)*60_000, 29290.0, 29300.0, 29280.0, 29295.0, 1.0, "epoch_floor", "MNQ 12-26", BarSourceHistorical).Error; err != nil {
			t.Fatalf("seed dec: %v", err)
		}
		n++
	}
	// one pre-column row with no contract at all (NULL, as GORM left them) on
	// a symbol the 2026-09-10 roll backfill does not claim (it labels every
	// unstamped MNQ/ES row), so it reaches the key migration still empty
	if err := bh.db.Exec(ins, "NQ", "1m", base, 6000.0, 6001.0, 5999.0, 6000.5, 1.0, "", nil, BarSourceLive).Error; err != nil {
		t.Fatalf("seed null contract: %v", err)
	}
	n++
	return bh, n
}

func pkOf(t *testing.T, bh *BarHistoryStore, table string) []string {
	t.Helper()
	pk, err := bh.barsPKColumns(table)
	if err != nil {
		t.Fatalf("pk of %s: %v", table, err)
	}
	return pk
}

func countOf(t *testing.T, bh *BarHistoryStore, sql string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	if err := bh.db.Raw(sql, args...).Scan(&n).Error; err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func TestBarsKeyMigrationMovesLegacyTableOntoContractKey(t *testing.T) {
	bh, seeded := newLegacyBarStore(t)
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyLegacyColumns) {
		t.Fatalf("fixture pk = %v, want legacy", got)
	}
	now := time.Date(2026, 9, 18, 1, 2, 3, 0, time.UTC)
	rep, err := bh.MigrateWithReport(now)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if rep.Err != nil || !rep.Migrated {
		t.Fatalf("report = %+v, want Migrated", rep)
	}
	if rep.Rows != seeded {
		t.Fatalf("rows after = %d, seeded %d", rep.Rows, seeded)
	}
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyColumns) {
		t.Fatalf("pk after = %v, want %v", got, barsKeyColumns)
	}
	if !BarsKeyHasContract() {
		t.Fatalf("BarsKeyHasContract() false after a successful migration")
	}
	// (d) the old table, renamed, WITH its unique index still attached.
	if rep.OldTable != "bars_pre_contract_key_2026-09-18" {
		t.Fatalf("old table = %q", rep.OldTable)
	}
	if n := countOf(t, bh, "SELECT COUNT(*) FROM `"+rep.OldTable+"`"); n != seeded {
		t.Fatalf("old table rows = %d, want %d", n, seeded)
	}
	if ok, _ := bh.barsIndexOnTable(idxBarsLegacyUnique, rep.OldTable); !ok {
		t.Fatalf("the renamed old table lost its unique index — a rename-back rollback would trip the old binary's dedupe block")
	}
	if ok, _ := bh.barsIndexOnTable(idxBarsLegacyUnique, "bars"); ok {
		t.Fatalf("the new table must NOT carry a unique (symbol,tf,open_time_ms) index")
	}
	for _, idx := range []string{idxBarsV2Time, idxBarsV2Source} {
		if ok, _ := bh.barsIndexOnTable(idx, "bars"); !ok {
			t.Fatalf("index %s missing on the new table", idx)
		}
	}
	// (b) the backup exists, is a database, and holds the pre-migration table.
	if rep.Backup == "" {
		t.Fatalf("no backup path in the report")
	}
	if fi, err := os.Stat(rep.Backup); err != nil || fi.Size() == 0 {
		t.Fatalf("backup %s: %v", rep.Backup, err)
	}
	if err := bh.db.Exec("ATTACH DATABASE ? AS bk", rep.Backup).Error; err != nil {
		t.Fatalf("attach backup: %v", err)
	}
	if got := countOf(t, bh, "SELECT COUNT(*) FROM bk.bars"); got != seeded {
		t.Fatalf("backup bars rows = %d, want %d", got, seeded)
	}
	if got := countOf(t, bh, "SELECT COUNT(*) FROM bk.sqlite_master WHERE type='table' AND name LIKE 'bars_pre_contract_key%'"); got != 0 {
		t.Fatalf("backup already holds a renamed table — it was taken AFTER the rename")
	}
	_ = bh.db.Exec("DETACH DATABASE bk")
	// the NULL contract became '' (NOT NULL key column)
	if got := countOf(t, bh, "SELECT COUNT(*) FROM bars WHERE contract = '' AND symbol = 'NQ'"); got != 1 {
		t.Fatalf("NULL contract row not carried as '': %d", got)
	}
	// (e) the boot line carries READ values
	line := rep.BootLine()
	for _, want := range []string{"🗄 bars: key migrated to (symbol,tf,contract,open_time_ms)", "rows=" + i64s(seeded), "backup=" + rep.Backup, "old_table=bars_pre_contract_key_2026-09-18"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line %q lacks %q", line, want)
		}
	}
}

func i64s(n int64) string { return strconv.FormatInt(n, 10) }

func TestBarsKeyMigrateIsIdempotent(t *testing.T) {
	bh, seeded := newLegacyBarStore(t)
	now := time.Date(2026, 9, 18, 1, 2, 3, 0, time.UTC)
	if rep, err := bh.MigrateWithReport(now); err != nil || !rep.Migrated {
		t.Fatalf("first: %+v %v", rep, err)
	}
	var sqlBefore string
	_ = bh.db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='bars'").Scan(&sqlBefore).Error
	backups, _ := filepath.Glob(filepath.Join(os.Getenv("BARS_KEY_BACKUP_DIR"), "pre-bars-key-*.db"))
	// second boot, a day later
	rep, err := bh.MigrateWithReport(now.Add(24 * time.Hour))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if rep.Migrated || !rep.AlreadyKeyed || rep.Err != nil {
		t.Fatalf("second report = %+v, want AlreadyKeyed only", rep)
	}
	if rep.MigratedOn != "2026-09-18" || rep.Rows != seeded {
		t.Fatalf("second report = %+v", rep)
	}
	if want := "🗄 bars: key=(symbol,tf,contract,open_time_ms) (migrated 2026-09-18) rows=" + i64s(seeded); rep.BootLine() != want {
		t.Fatalf("second boot line = %q want %q", rep.BootLine(), want)
	}
	var sqlAfter string
	_ = bh.db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='bars'").Scan(&sqlAfter).Error
	if sqlAfter != sqlBefore {
		t.Fatalf("second Migrate (incl. AutoMigrate) rewrote the table:\n%s\n→\n%s", sqlBefore, sqlAfter)
	}
	if again, _ := filepath.Glob(filepath.Join(os.Getenv("BARS_KEY_BACKUP_DIR"), "pre-bars-key-*.db")); len(again) != len(backups) || len(backups) != 1 {
		t.Fatalf("backups after second run = %d (first run %d), want exactly 1", len(again), len(backups))
	}
	if n := countOf(t, bh, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'bars_pre_contract_key%'"); n != 1 {
		t.Fatalf("renamed tables = %d, want 1", n)
	}
	// a third call through the plain Migrate() entry is the same no-op
	if err := bh.Migrate(); err != nil {
		t.Fatalf("third: %v", err)
	}
}

func TestBarsKeyMigrationRefusedWhenBackupCannotBeWritten(t *testing.T) {
	bh, seeded := newLegacyBarStore(t)
	// a regular FILE where the backup dir should be: MkdirAll fails
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BARS_KEY_BACKUP_DIR", blocker)
	rep, err := bh.MigrateWithReport(time.Date(2026, 9, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatalf("Migrate must not fail the boot: %v", err)
	}
	if rep.Err == nil || rep.Migrated {
		t.Fatalf("report = %+v, want a refused migration", rep)
	}
	if !strings.HasPrefix(rep.BootLine(), "🗄 bars: migration FAILED — backup refused:") || !strings.Contains(rep.BootLine(), "old table intact") {
		t.Fatalf("boot line = %q", rep.BootLine())
	}
	// old table intact, on the legacy key, unique index present, no rename
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyLegacyColumns) {
		t.Fatalf("pk = %v, want legacy", got)
	}
	if ok, _ := bh.barsIndexOnTable(idxBarsLegacyUnique, "bars"); !ok {
		t.Fatalf("legacy unique index missing after a refused migration")
	}
	if n := countOf(t, bh, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'bars_pre_contract_key%' OR name='bars_v2'"); n != 0 {
		t.Fatalf("a refused migration left tables behind: %d", n)
	}
	if n := countOf(t, bh, "SELECT COUNT(*) FROM bars"); n != seeded {
		t.Fatalf("rows = %d, want %d", n, seeded)
	}
	if BarsKeyHasContract() {
		t.Fatalf("BarsKeyHasContract() true after a refused migration")
	}
	// the bot keeps running: the writer uses the LEGACY conflict target
	if err := bh.InsertBars([]BarHistoryDB{mkBar("MNQ", "1m", 1_789_000_000_000+200*60_000, 29001)}); err != nil {
		t.Fatalf("InsertBars on the legacy key after fail-open: %v", err)
	}
	if n := countOf(t, bh, "SELECT COUNT(*) FROM bars"); n != seeded+1 {
		t.Fatalf("rows = %d, want %d", n, seeded+1)
	}
}

func TestBarsKeyFreshDatabaseIsCreatedOnTheContractKey(t *testing.T) {
	t.Setenv("BARS_KEY_BACKUP_DIR", t.TempDir())
	bh := newBarStore(t)
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyColumns) {
		t.Fatalf("fresh pk = %v", got)
	}
	rep, err := bh.MigrateWithReport(time.Now())
	if err != nil || !rep.AlreadyKeyed || rep.MigratedOn != "" {
		t.Fatalf("fresh report = %+v err=%v", rep, err)
	}
	if !strings.HasPrefix(rep.BootLine(), "🗄 bars: key=(symbol,tf,contract,open_time_ms) (created on this key)") {
		t.Fatalf("fresh boot line = %q", rep.BootLine())
	}
	if files, _ := filepath.Glob(filepath.Join(os.Getenv("BARS_KEY_BACKUP_DIR"), "*")); len(files) != 0 {
		t.Fatalf("a fresh database must not take a backup: %v", files)
	}
}

// TestBarsKeyTwoContractsSameMinuteBothStored is the wave's reason: the
// December history for a minute September holds LANDS, and every
// contract-scoped reader sees exactly its own contract.
func TestBarsKeyTwoContractsSameMinuteBothStored(t *testing.T) {
	t.Setenv("BARS_KEY_BACKUP_DIR", t.TempDir())
	bh := newBarStore(t)
	base := int64(1_789_000_000_000)
	var sep, dec []BarHistoryDB
	for i := 0; i < 20; i++ {
		ts := base + int64(i)*60_000
		sep = append(sep, BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: ts, O: 29000, H: 29010, L: 28990, C: 29005, V: 1, Contract: "MNQ 09-26", Source: BarSourceLive})
		if i >= 10 {
			dec = append(dec, BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: ts, O: 29290, H: 29300, L: 29280, C: 29295, V: 1, Contract: "MNQ 12-26", Source: BarSourceHistorical})
		}
	}
	if err := bh.InsertBars(sep); err != nil {
		t.Fatal(err)
	}
	if err := bh.InsertBars(dec); err != nil {
		t.Fatal(err)
	}
	if n, _ := bh.Count(); n != 30 {
		t.Fatalf("rows = %d, want 30 (20 sep + 10 dec overlap)", n)
	}
	// a replay never overwrites live WITHIN a contract (rule unchanged)
	if err := bh.InsertBars([]BarHistoryDB{{Symbol: "MNQ", TF: "1m", OpenTimeMs: base, O: 1, H: 1, L: 1, C: 1, V: 1, Contract: "MNQ 09-26", Source: BarSourceHistorical}}); err != nil {
		t.Fatal(err)
	}
	if rows, _ := bh.LastNBarsOn("MNQ", "1m", "MNQ 09-26", 100); len(rows) != 20 || rows[0].C != 29005 {
		t.Fatalf("sep reader: %d rows, first close %.2f", len(rows), rows[0].C)
	}
	if rows, _ := bh.LastNBarsOn("MNQ", "1m", "MNQ 12-26", 100); len(rows) != 10 || rows[0].C != 29295 {
		t.Fatalf("dec reader: %d rows", len(rows))
	}
	// the planner door reads ONE contract, NT8-only
	if rows, _ := bh.LastNBarsFromNT8On("MNQ", "1m", "MNQ 12-26", 100); len(rows) != 10 {
		t.Fatalf("planner door: %d rows", len(rows))
	}
	nt8Rows, nt8Err := bh.LastNBarsFromNT8On("MNQ", "1m", "MNQ 12-26", 100)
	for _, r := range mustRows(t, nt8Rows, nt8Err) {
		if r.Contract != "MNQ 12-26" {
			t.Fatalf("planner door leaked %s", r.Contract)
		}
	}
	if rows, _ := bh.BarsBetweenOn("MNQ", "1m", "MNQ 09-26", base+10*60_000, base+20*60_000); len(rows) != 10 {
		t.Fatalf("BarsBetweenOn sep: %d", len(rows))
	}
	if rows, _ := bh.BarsBetweenFromNT8On("MNQ", "1m", "MNQ 12-26", 0, base+100*60_000); len(rows) != 10 {
		t.Fatalf("BarsBetweenFromNT8On dec: %d", len(rows))
	}
	// the unfiltered audit read shows the seam: two rows per overlap minute
	if rows, _ := bh.BarsBetween("MNQ", "1m", base+10*60_000, base+11*60_000); len(rows) != 2 {
		t.Fatalf("unfiltered overlap minute: %d rows, want 2", len(rows))
	}
	if n, _ := bh.RollOverlaps(); n != 10 {
		t.Fatalf("RollOverlaps = %d, want 10", n)
	}
	if dups, _, total, err := bh.BarsIntegrity(); err != nil || dups != 0 || total != 30 {
		t.Fatalf("integrity dups=%d total=%d err=%v — an overlap is not a duplicate", dups, total, err)
	}
	// an IMPORT of contract B at a minute contract A holds lands (CLASS 143's
	// "occupied slot" no longer exists); the same contract's own bar is a skip
	ins, skip, err := bh.ImportBars([]BarHistoryDB{
		{Symbol: "MNQ", TF: "1m", OpenTimeMs: base, O: 1, H: 1, L: 1, C: 1, V: 1, Contract: "MNQ 12-26", Source: BarSourceHistoricalImport},
		{Symbol: "MNQ", TF: "1m", OpenTimeMs: base + 10*60_000, O: 1, H: 1, L: 1, C: 1, V: 1, Contract: "MNQ 12-26", Source: BarSourceHistoricalImport},
	})
	if err != nil || ins != 1 || skip != 1 {
		t.Fatalf("import ins=%d skip=%d err=%v, want 1/1", ins, skip, err)
	}
	// the chart's prior-contract reader still gives ONE row per minute
	prior, err := bh.PriorContractBarsBefore("MNQ", "1m", "MNQ 12-26", base+20*60_000, 100)
	if err != nil || len(prior) != 20 {
		t.Fatalf("prior reader: %d rows err=%v", len(prior), err)
	}
	for _, r := range prior {
		if r.Contract != "MNQ 09-26" {
			t.Fatalf("prior reader leaked %s", r.Contract)
		}
	}
}

func mustRows(t *testing.T, rows []BarHistoryDB, err error) []BarHistoryDB {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// TestLatestContractTieBreakOnSharedMinute: on the contract key two contracts
// can hold the newest minute. The row that TRADED wins; among equals the
// later expiry — parsed from the broker's label, never a calendar.
func TestLatestContractTieBreakOnSharedMinute(t *testing.T) {
	t.Setenv("BARS_KEY_BACKUP_DIR", t.TempDir())
	bh := newBarStore(t)
	base := int64(1_789_000_000_000)
	mk := func(c, src string, ts int64) BarHistoryDB {
		return BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: ts, O: 1, H: 1, L: 1, C: 1, V: 1, Contract: c, Source: src}
	}
	// September traded the minute; December's row is served history
	if err := bh.InsertBars([]BarHistoryDB{mk("MNQ 09-26", BarSourceLive, base), mk("MNQ 12-26", BarSourceHistorical, base)}); err != nil {
		t.Fatal(err)
	}
	if c, ok := bh.LatestContract("MNQ"); !ok || c != "MNQ 09-26" {
		t.Fatalf("live must beat replay on the shared minute: %q %v", c, ok)
	}
	if c, ok := bh.ContractAt("MNQ", base); !ok || c != "MNQ 09-26" {
		t.Fatalf("ContractAt: %q %v", c, ok)
	}
	// both live on the same minute (the roll minute itself): later expiry wins,
	// across a year boundary too
	if err := bh.InsertBars([]BarHistoryDB{mk("MNQ 12-26", BarSourceLive, base+60_000), mk("MNQ 03-27", BarSourceLive, base+60_000)}); err != nil {
		t.Fatal(err)
	}
	if c, _ := bh.LatestContract("MNQ"); c != "MNQ 03-27" {
		t.Fatalf("later expiry must win across the year boundary: %q", c)
	}
	if c, _ := bh.ContractAt("MNQ", base); c != "MNQ 09-26" {
		t.Fatalf("ContractAt(base) must not see the later minute: %q", c)
	}
	if !contractNewer("MNQ 03-27", "MNQ 12-26") || contractNewer("MNQ 09-26", "MNQ 12-26") {
		t.Fatalf("contractNewer ordering wrong")
	}
}

// TestBarsKeyOldBinaryStatementsOnTheMigratedTable answers the rollback
// question with the OLD code's exact statements (store/bar_history.go at
// origin/dev 0dd27940 lines 211 and 315-318): its upsert is REFUSED by SQLite
// (no unique constraint matches its conflict target — persistence dies, reads
// live), and its name-only unique-index check is satisfied by the renamed
// table's index, so its 2026-08-27 dedupe block does NOT run.
func TestBarsKeyOldBinaryStatementsOnTheMigratedTable(t *testing.T) {
	bh, _ := newLegacyBarStore(t)
	if rep, err := bh.MigrateWithReport(time.Date(2026, 9, 18, 1, 2, 3, 0, time.UTC)); err != nil || !rep.Migrated {
		t.Fatalf("migrate: %+v %v", rep, err)
	}
	oldUpsert := "INSERT INTO bars(symbol, tf, open_time_ms, o, h, l, c, v, convention, contract, source) VALUES (?,?,?,?,?,?,?,?,?,?,?)" +
		" ON CONFLICT(symbol, tf, open_time_ms) DO UPDATE SET o=excluded.o, h=excluded.h, l=excluded.l, c=excluded.c, v=excluded.v, convention=excluded.convention, contract=excluded.contract, source=excluded.source" +
		" WHERE NOT (bars.source IN ('live','mixed') AND excluded.source = 'historical')"
	err := bh.db.Exec(oldUpsert, "MNQ", "1m", int64(1_789_000_000_000), 1.0, 1.0, 1.0, 1.0, 1.0, "", "MNQ 12-26", "live").Error
	if err == nil {
		t.Fatalf("the old upsert must be refused on the contract key (its conflict target is no longer unique)")
	}
	if !strings.Contains(err.Error(), "ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint") {
		t.Fatalf("unexpected refusal: %v", err)
	}
	// the old Migrate's guard, verbatim (name only, no tbl_name)
	if n := countOf(t, bh, "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_bars_sym_tf_time_unique'"); n != 1 {
		t.Fatalf("old guard sees %d unique indexes; with 0 its dedupe block would DELETE every tf<>'1m' row", n)
	}
	// old reads still work
	if rows, err := bh.LastNBarsOn("MNQ", "1m", "MNQ 09-26", 5); err != nil || len(rows) != 5 {
		t.Fatalf("old-shape read: %d %v", len(rows), err)
	}
}

// TestBarsKeyMigrationOnLiveCopy runs the migration against a COPY of the
// live database when BARS_KEY_LIVE_COPY names one (the wave's scratch
// .backup); it copies that file again into a temp dir so the run is
// repeatable. Skipped otherwise. Prints the boot lines and the timing.
func TestBarsKeyMigrationOnLiveCopy(t *testing.T) {
	src := os.Getenv("BARS_KEY_LIVE_COPY")
	if src == "" {
		t.Skip("BARS_KEY_LIVE_COPY not set")
	}
	dst := filepath.Join(t.TempDir(), "live-copy.db")
	if os.Getenv("BARS_KEY_LIVE_COPY_INPLACE") == "1" {
		dst = src // the rollback drill: migrate the scratch copy itself
	} else {
		in, err := os.Open(src)
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(dst)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(out, in); err != nil {
			t.Fatal(err)
		}
		_ = in.Close()
		_ = out.Close()
	}
	if os.Getenv("BARS_KEY_BACKUP_DIR") == "" {
		t.Setenv("BARS_KEY_BACKUP_DIR", filepath.Join(t.TempDir(), "backups"))
	}
	barsKeyed.Store(false)
	st, err := New(dst)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	bh := st.BarHistory()
	before := countOf(t, bh, "SELECT COUNT(*) FROM bars")
	byKeyBefore := countOf(t, bh, "SELECT COUNT(*) FROM (SELECT symbol, tf, contract, open_time_ms FROM bars GROUP BY 1,2,3,4)")
	pkBefore := pkOf(t, bh, "bars")
	t0 := time.Now()
	rep, err := bh.MigrateWithReport(time.Now())
	took := time.Since(t0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("live copy: pk before=%v rows before=%d distinct-by-new-key=%d", pkBefore, before, byKeyBefore)
	t.Logf("boot line 1: %s", rep.BootLine())
	t.Logf("migration took %s", took.Truncate(time.Millisecond))
	if rep.Err != nil || !rep.Migrated {
		t.Fatalf("report = %+v", rep)
	}
	after := countOf(t, bh, "SELECT COUNT(*) FROM bars")
	if after != before || rep.Rows != before {
		t.Fatalf("rows before=%d after=%d report=%d", before, after, rep.Rows)
	}
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyColumns) {
		t.Fatalf("pk after = %v", got)
	}
	if n := countOf(t, bh, "SELECT COUNT(*) FROM `"+rep.OldTable+"`"); n != before {
		t.Fatalf("old table rows = %d", n)
	}
	overlaps, _ := bh.RollOverlaps()
	dups, tfs, total, _ := bh.BarsIntegrity()
	t.Logf("after: rows=%d dups=%d overlaps=%d tfs=%v", total, dups, overlaps, tfs)
	t0 = time.Now()
	rep2, err := bh.MigrateWithReport(time.Now().Add(24 * time.Hour))
	if err != nil || rep2.Migrated || !rep2.AlreadyKeyed {
		t.Fatalf("second: %+v %v", rep2, err)
	}
	t.Logf("boot line 2: %s (took %s)", rep2.BootLine(), time.Since(t0).Truncate(time.Millisecond))
	if fi, err := os.Stat(rep.Backup); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("backup %s size=%d bytes", rep.Backup, fi.Size())
	}
}

// ── review fixes 1 + 2 (2026-09-18) ──────────────────────────────────────────

// TestBarsKeyBackupVerifyIsPinnedToOneConnection proves the ATTACH / COUNT /
// DETACH sequence executes on the connection it was given: the ATTACH is
// visible to the COUNT on that conn (deterministic), and a pool hammered by
// concurrent SELECTs on other connections cannot come between them.
func TestBarsKeyBackupVerifyIsPinnedToOneConnection(t *testing.T) {
	t.Setenv("BARS_KEY_BACKUP_DIR", t.TempDir())
	bh := newBarStore(t)
	if err := bh.InsertBars([]BarHistoryDB{mkBar("MNQ", "1m", 1000, 1), mkBar("MNQ", "1m", 2000, 2)}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := bh.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// hammer the pool from other goroutines for the whole verify
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					var n int64
					_ = bh.db.Raw("SELECT COUNT(*) FROM bars").Scan(&n).Error
				}
			}
		}()
	}
	defer func() { close(stop); wg.Wait() }()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	path := filepath.Join(t.TempDir(), "copy.db")
	if _, err := conn.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		n, err := verifyBackupOnConn(ctx, conn, path)
		if err != nil || n != 2 {
			t.Fatalf("iteration %d: rows=%d err=%v — the ATTACH was not visible to the COUNT", i, n, err)
		}
	}
	// and the pinned conn is the only place the attachment ever existed: on
	// the pool at large the schema name is unknown
	var n int64
	if err := bh.db.Raw("SELECT COUNT(*) FROM barskey_backup.bars").Scan(&n).Error; err == nil {
		t.Fatalf("barskey_backup leaked onto the pool (still attached somewhere)")
	}
	// the whole migration-side backup under the same hammer
	if p, rows, err := bh.backupBeforeKeyMigration(time.Now()); err != nil || rows != 2 || p == "" {
		t.Fatalf("backupBeforeKeyMigration under pool load: path=%q rows=%d err=%v", p, rows, err)
	}
}

// TestBarsKeyMigrationRefusedWhenDiskIsShort injects a required-free limit no
// disk can meet: the migration is refused BEFORE VACUUM INTO writes a byte,
// with a line naming the dir, the free bytes, the db size and the need.
func TestBarsKeyMigrationRefusedWhenDiskIsShort(t *testing.T) {
	bh, seeded := newLegacyBarStore(t)
	prev := barsKeyRequiredFree
	barsKeyRequiredFree = func(dbBytes int64) int64 { return 1 << 60 }
	t.Cleanup(func() { barsKeyRequiredFree = prev })
	rep, err := bh.MigrateWithReport(time.Date(2026, 9, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Err == nil || rep.Migrated {
		t.Fatalf("report = %+v, want refused", rep)
	}
	if !strings.Contains(rep.Err.Error(), "insufficient disk for the backup") || !strings.Contains(rep.Err.Error(), "2× the file") {
		t.Fatalf("refusal = %v", rep.Err)
	}
	if files, _ := filepath.Glob(filepath.Join(os.Getenv("BARS_KEY_BACKUP_DIR"), "pre-bars-key-*")); len(files) != 0 {
		t.Fatalf("a refused backup wrote a file: %v", files)
	}
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyLegacyColumns) || countOf(t, bh, "SELECT COUNT(*) FROM bars") != seeded {
		t.Fatalf("old table not intact: pk=%v", got)
	}
	// the precheck reads a real number for a real dir
	if free, err := barsKeyFreeBytes(os.Getenv("BARS_KEY_BACKUP_DIR")); err != nil || free <= 0 {
		t.Fatalf("free bytes = %d err=%v", free, err)
	}
}

// TestBarsKeyIndexNameHeldByAnotherTableIsAnError: a rollback's renamed copy
// that kept idx_bars_v2_* would make CREATE INDEX IF NOT EXISTS a silent
// no-op on the new bars; the migration must say so instead.
func TestBarsKeyIndexNameHeldByAnotherTableIsAnError(t *testing.T) {
	bh, _ := newLegacyBarStore(t)
	if err := bh.db.Exec("CREATE TABLE leftover(symbol text, tf text, open_time_ms integer)").Error; err != nil {
		t.Fatal(err)
	}
	if err := bh.db.Exec("CREATE INDEX " + idxBarsV2Time + " ON leftover(symbol, tf, open_time_ms)").Error; err != nil {
		t.Fatal(err)
	}
	rep, err := bh.MigrateWithReport(time.Date(2026, 9, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Err == nil || !strings.Contains(rep.Err.Error(), "already belongs to table leftover") {
		t.Fatalf("report = %+v", rep)
	}
	if got := pkOf(t, bh, "bars"); !sameColumns(got, barsKeyLegacyColumns) {
		t.Fatalf("transaction did not roll back: pk=%v", got)
	}
}
