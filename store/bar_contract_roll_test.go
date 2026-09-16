package store

import (
	"path/filepath"
	"testing"
)

// ── ROLL WAVE PINS (store half) ──────────────────────────────────────────────
//
// THE ROLL BOUNDARY IS A FIXTURE CONSTANT STATED HERE (A28), never read from the
// wall: the tape of 2026-09-10, ASIA. Epoch ms, CT in comments.
const (
	rollT2114 int64 = 1789092840000 // 21:14 — MNQ last clean September; ES first spans-roll
	rollT2115 int64 = 1789092900000 // 21:15 — MNQ first spans-roll
	rollT2117 int64 = 1789093020000 // 21:17 — last spans-roll, both
	rollT2118 int64 = 1789093080000 // 21:18 — first clean December, both
)

func rollStore(t *testing.T) *BarHistoryStore {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "roll.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	bh := st.BarHistory()
	if err := bh.Migrate(); err != nil {
		t.Fatal(err)
	}
	return bh
}

// seedUnstamped writes rows the way the PRE-WAVE writer did: no contract.
// Goes straight to SQL because InsertBars now refuses exactly this.
func seedUnstamped(t *testing.T, bh *BarHistoryStore, symbol, tf string, opens []int64, o, c float64) {
	t.Helper()
	for _, ms := range opens {
		if err := bh.db.Exec(`INSERT INTO bars(symbol, tf, open_time_ms, o, h, l, c, v, convention, contract) VALUES (?,?,?,?,?,?,?,?,?,'')`,
			symbol, tf, ms, o, o+1, o-1, c, 1.0, "epoch_floor").Error; err != nil {
			t.Fatal(err)
		}
	}
}

// E4 / D2 — THREE-STATE BACKFILL BY WINDOW INTERSECTION.
//
// A 1m bar is classified by its own minute; a 5m bar opening at 21:15 and a 1h
// bar opening at 21:00 BOTH contain the roll and are spans-roll even though
// only one of them opens inside the window. Nothing is deleted.
func TestBackfillThreeStateByWindow(t *testing.T) {
	bh := rollStore(t)
	// MNQ 1m: 21:12 .. 21:20
	var m1 []int64
	for ms := rollT2114 - 2*60_000; ms <= rollT2118+2*60_000; ms += 60_000 {
		m1 = append(m1, ms)
	}
	seedUnstamped(t, bh, "MNQ", "1m", m1, 29130, 29131)
	// MNQ 5m: 21:10, 21:15, 21:20
	seedUnstamped(t, bh, "MNQ", "5m", []int64{rollT2115 - 300_000, rollT2115, rollT2115 + 300_000}, 29130, 29131)
	// MNQ 1h: 20:00, 21:00, 22:00
	h21 := rollT2115 - int64(15*60_000)
	seedUnstamped(t, bh, "MNQ", "1h", []int64{h21 - 3_600_000, h21, h21 + 3_600_000}, 29130, 29131)
	before, _ := bh.Count()

	if err := bh.migrateContractColumn(); err != nil {
		t.Fatal(err)
	}

	after, _ := bh.Count()
	if before != after {
		t.Fatalf("A24: the backfill DELETED rows — %d before, %d after", before, after)
	}
	want := map[int64]string{
		rollT2114:            "MNQ 09-26",
		rollT2115:            ContractMixed,
		rollT2115 + 60_000:   ContractMixed,
		rollT2117:            ContractMixed,
		rollT2118:            "MNQ 12-26",
		rollT2118 + 60_000:   "MNQ 12-26",
		rollT2114 - 2*60_000: "MNQ 09-26",
	}
	for ms, exp := range want {
		var got string
		bh.db.Raw("SELECT contract FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms=?", ms).Scan(&got)
		if got != exp {
			t.Errorf("1m @%d: want %q got %q", ms, exp, got)
		}
	}
	check := func(tf string, open int64, exp string) {
		var got string
		bh.db.Raw("SELECT contract FROM bars WHERE symbol='MNQ' AND tf=? AND open_time_ms=?", tf, open).Scan(&got)
		if got != exp {
			t.Errorf("%s @%d: want %q got %q", tf, open, exp, got)
		}
	}
	check("5m", rollT2115-300_000, "MNQ 09-26") // 21:10-21:15 ends AT the window start: clean
	check("5m", rollT2115, ContractMixed)       // 21:15-21:20 contains it
	check("5m", rollT2115+300_000, "MNQ 12-26") // 21:20-21:25 after
	check("1h", h21-3_600_000, "MNQ 09-26")     // 20:00-21:00
	check("1h", h21, ContractMixed)             // 21:00-22:00 contains it — the dispatch's own example
	check("1h", h21+3_600_000, "MNQ 12-26")     // 22:00-23:00
	var nulls int64
	bh.db.Raw("SELECT COUNT(*) FROM bars WHERE contract='' OR contract IS NULL").Scan(&nulls)
	if nulls != 0 {
		t.Fatalf("%d rows left unstamped", nulls)
	}
}

// E6 — HISTORY PIN: retired bars are never deleted, and the backfill is
// idempotent — a second run changes nothing.
func TestRetiredBarsAreNeverDeletedAndBackfillIsIdempotent(t *testing.T) {
	bh := rollStore(t)
	var opens []int64
	for ms := rollT2114 - 10*60_000; ms <= rollT2118+10*60_000; ms += 60_000 {
		opens = append(opens, ms)
	}
	seedUnstamped(t, bh, "MNQ", "1m", opens, 29130, 29131)
	if err := bh.migrateContractColumn(); err != nil {
		t.Fatal(err)
	}
	c1, _ := bh.ContractCensus("MNQ")
	if err := bh.migrateContractColumn(); err != nil {
		t.Fatal(err)
	}
	c2, _ := bh.ContractCensus("MNQ")
	for k, v := range c1 {
		if c2[k] != v {
			t.Fatalf("second backfill changed %q: %d → %d", k, v, c2[k])
		}
	}
	if c1["MNQ 09-26"] == 0 {
		t.Fatal("no retired rows survived — history was deleted")
	}
}

// D1 — the writer REFUSES a bar with no contract.
func TestInsertRefusesUnstampedBar(t *testing.T) {
	bh := rollStore(t)
	err := bh.InsertBars([]BarHistoryDB{{Symbol: "MNQ", TF: "1m", OpenTimeMs: rollT2118, O: 1, H: 1, L: 1, C: 1}})
	if err == nil {
		t.Fatal("a bar on an unknown price scale must be refused, not written")
	}
	n, _ := bh.Count()
	if n != 0 {
		t.Fatalf("refused write still landed %d row(s)", n)
	}
}

// D4 — READER PIN: given a mixed store, the filtered readers return only the
// asked-for contract, and the unfiltered form is the only way to see the seam.
func TestFilteredReadersNeverCrossTheRoll(t *testing.T) {
	bh := rollStore(t)
	var opens []int64
	for ms := rollT2114 - 5*60_000; ms <= rollT2118+5*60_000; ms += 60_000 {
		opens = append(opens, ms)
	}
	seedUnstamped(t, bh, "MNQ", "1m", opens, 29130, 29131)
	if err := bh.migrateContractColumn(); err != nil {
		t.Fatal(err)
	}
	cur, err := bh.LastNBarsOn("MNQ", "1m", "MNQ 12-26", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range cur {
		if r.Contract != "MNQ 12-26" || r.OpenTimeMs < rollT2118 {
			t.Fatalf("current-contract read returned a %q bar at %d", r.Contract, r.OpenTimeMs)
		}
	}
	if len(cur) != 6 { // 21:18 .. 21:23
		t.Fatalf("want 6 December bars, got %d", len(cur))
	}
	all, _ := bh.LastNBars("MNQ", "1m", 100)
	if len(all) <= len(cur) {
		t.Fatal("the unfiltered read should see the seam; it saw the same rows as the filtered one")
	}
	win, ok := bh.WindowContract("MNQ", rollT2114-60_000, rollT2118+60_000)
	if ok || win != ContractMixed {
		t.Fatalf("a window across the roll must be unrecomputable, got %q ok=%v", win, ok)
	}
	win2, ok2 := bh.WindowContract("MNQ", rollT2118, rollT2118+3*60_000)
	if !ok2 || win2 != "MNQ 12-26" {
		t.Fatalf("a window wholly on December must resolve to it, got %q ok=%v", win2, ok2)
	}
	lc, _ := bh.LatestContract("MNQ")
	if lc != "MNQ 12-26" {
		t.Fatalf("latest usable contract: want MNQ 12-26, got %q", lc)
	}
}
