package store

import "testing"

// ── BAR-SOURCE WAVE PINS (store half) ────────────────────────────────────────

func bsRow(t int64, c float64, src string) BarHistoryDB {
	return BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: t, O: c, H: c + 1, L: c - 1, C: c, V: 1, Contract: "MNQ 12-26", Source: src}
}

func readOne(t *testing.T, bh *BarHistoryStore, ms int64) BarHistoryDB {
	t.Helper()
	var r BarHistoryDB
	if err := bh.db.Where("symbol='MNQ' AND tf='1m' AND open_time_ms=?", ms).First(&r).Error; err != nil {
		t.Fatalf("read %d: %v", ms, err)
	}
	return r
}

// THE UPSERT RULE — store half. RED on the pre-wave upsert: DO UPDATE was
// unconditional and the replay overwrote the live row.
func TestReplayNeverOverwritesALiveRow(t *testing.T) {
	bh := rollStore(t)
	const ms = int64(1789097820000)
	if err := bh.InsertBars([]BarHistoryDB{bsRow(ms, 29358.25, BarSourceLive)}); err != nil {
		t.Fatal(err)
	}
	if err := bh.InsertBars([]BarHistoryDB{bsRow(ms, 29068.25, BarSourceHistorical)}); err != nil {
		t.Fatal(err)
	}
	r := readOne(t, bh, ms)
	if r.C != 29358.25 || r.Source != BarSourceLive {
		t.Fatalf("the replay overwrote a live row: close=%.2f source=%q — this is the 186-row damage of 2026-09-10 22:39", r.C, r.Source)
	}
}

// Live overwrites historical; historical fills a gap; mixed is kept over a
// later replay (it is the evidence of the seam).
func TestSourcePrecedence(t *testing.T) {
	bh := rollStore(t)
	const a, b, c = int64(1789097820000), int64(1789097880000), int64(1789097940000)
	// historical first, then live: live wins
	_ = bh.InsertBars([]BarHistoryDB{bsRow(a, 29068, BarSourceHistorical)})
	_ = bh.InsertBars([]BarHistoryDB{bsRow(a, 29358, BarSourceLive)})
	if r := readOne(t, bh, a); r.C != 29358 || r.Source != BarSourceLive {
		t.Fatalf("live must overwrite historical: %+v", r)
	}
	// historical into an empty minute: fills
	_ = bh.InsertBars([]BarHistoryDB{bsRow(b, 29068, BarSourceHistorical)})
	if r := readOne(t, bh, b); r.Source != BarSourceHistorical {
		t.Fatalf("historical must fill a gap: %+v", r)
	}
	// mixed, then a replay: mixed stays
	_ = bh.InsertBars([]BarHistoryDB{bsRow(c, 29355, BarSourceMixed)})
	_ = bh.InsertBars([]BarHistoryDB{bsRow(c, 29068, BarSourceHistorical)})
	if r := readOne(t, bh, c); r.Source != BarSourceMixed || r.C != 29355 {
		t.Fatalf("a replay must not erase the mixed evidence: %+v", r)
	}
}

// A bar with no source (or a made-up one) is refused, not guessed.
func TestInsertRefusesUnlabelledSource(t *testing.T) {
	bh := rollStore(t)
	r := bsRow(1789097820000, 29358, "")
	if err := bh.InsertBars([]BarHistoryDB{r}); err == nil {
		t.Fatal("a bar that does not name its feed must be refused")
	}
	r.Source = "replayish"
	if err := bh.InsertBars([]BarHistoryDB{r}); err == nil {
		t.Fatal("an unknown source label must be refused")
	}
	if n, _ := bh.Count(); n != 0 {
		t.Fatalf("refused writes landed %d row(s)", n)
	}
}

// Readers never hand out a mixed bar.
func TestReadersExcludeMixed(t *testing.T) {
	bh := rollStore(t)
	const a, b, c = int64(1789097820000), int64(1789097880000), int64(1789097940000)
	_ = bh.InsertBars([]BarHistoryDB{bsRow(a, 29358, BarSourceLive), bsRow(b, 29355, BarSourceMixed), bsRow(c, 29360, BarSourceLive)})
	last, _ := bh.LastNBarsOn("MNQ", "1m", "MNQ 12-26", 10)
	between, _ := bh.BarsBetweenOn("MNQ", "1m", "MNQ 12-26", a, c+1)
	for _, rows := range [][]BarHistoryDB{last, between} {
		if len(rows) != 2 {
			t.Fatalf("want 2 readable bars, got %d", len(rows))
		}
		for _, r := range rows {
			if r.Source == BarSourceMixed {
				t.Fatalf("a reader handed out a mixed bar at %d", r.OpenTimeMs)
			}
		}
	}
}

// The migration labels pre-column rows LIVE — the record of what traded, which
// a future replay must not repaint — and the roll wave's spans-roll rows mixed.
// Idempotent. (The first draft labelled them historical and would have licensed
// the next boot's replay to overwrite the whole store again.)
func TestSourceBackfillIsConservativeAndIdempotent(t *testing.T) {
	bh := rollStore(t)
	// pre-column rows, written raw
	for _, x := range []struct {
		ms       int64
		contract string
	}{{1789092840000, "MNQ 09-26"}, {1789092900000, ContractMixed}, {1789093080000, "MNQ 12-26"}} {
		if err := bh.db.Exec(`INSERT INTO bars(symbol,tf,open_time_ms,o,h,l,c,v,convention,contract,source) VALUES ('MNQ','1m',?,1,2,0,1,1,'epoch_floor',?,'')`, x.ms, x.contract).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := bh.migrateSourceColumn(); err != nil {
		t.Fatal(err)
	}
	if r := readOne(t, bh, 1789092840000); r.Source != BarSourceLive {
		t.Fatalf("a pre-column row is the record of what traded and must be labelled live, got %q", r.Source)
	}
	// And the reason: a replay arriving later must NOT repaint it.
	if err := bh.InsertBars([]BarHistoryDB{{Symbol: "MNQ", TF: "1m", OpenTimeMs: 1789092840000, O: 9, H: 9, L: 9, C: 9, V: 1, Contract: "MNQ 09-26", Source: BarSourceHistorical}}); err != nil {
		t.Fatal(err)
	}
	if r := readOne(t, bh, 1789092840000); r.C == 9 {
		t.Fatal("a replay repainted a pre-column row — the label the migration chose licensed the 22:39 damage again")
	}
	if r := readOne(t, bh, 1789092900000); r.Source != BarSourceMixed {
		t.Fatalf("a spans-roll row is mixed, got %q", r.Source)
	}
	c1, _ := bh.SourceCensus("MNQ")
	if err := bh.migrateSourceColumn(); err != nil {
		t.Fatal(err)
	}
	c2, _ := bh.SourceCensus("MNQ")
	for k, v := range c1 {
		if c2[k] != v {
			t.Fatalf("second run changed %q: %d→%d", k, v, c2[k])
		}
	}
	if c1[""] != 0 {
		t.Fatalf("%d rows left unlabelled", c1[""])
	}
}

// THE MEASURED BACKFILL. Rows shaped like the live DB's 2026-09-10 window,
// written raw and unlabelled, come out with the label the tape supports:
// off-scale only for a December row inside the window with both sides below
// the dividing value; mixed for a straddle; live for everything else — a
// September row at the same low price, a December row at the same low price
// OUTSIDE the window, a December row inside the window on the live scale.
func TestSourceBackfillLabelsTheMeasuredOffScaleRowsByValueNotRowid(t *testing.T) {
	bh := rollStore(t)
	type raw struct {
		ms       int64
		o, c     float64
		contract string
		want     string
		why      string
	}
	cases := []raw{
		{1789096440000, 29076.5, 29078.75, "MNQ 12-26", BarSourceOffScale, "22:14 CT, both sides on the replay scale, December"},
		{1789093140000, 29080, 29081, "MNQ 12-26", BarSourceOffScale, "21:19 CT, a restart-gap minute the 22:14 boot's replay filled"},
		{1789092840000, 29131.5, 29130, "MNQ 09-26", BarSourceLive, "21:14 CT September: low is its own scale"},
		{1789096440000 - 86400000, 29076.5, 29078.75, "MNQ 12-26", BarSourceLive, "same price a day earlier: outside the window the value means nothing"},
		{1789097940000, 29068.25, 29355.25, "MNQ 12-26", BarSourceMixed, "22:39 CT boot minute: replay open, live close"},
		{1789092900000, 29131.5, 29416.0, "MNQ 12-26", BarSourceMixed, "21:15 CT roll minute whose spans-roll label the 22:39 replay overwrote"},
		{1789097880000, 29360, 29367.25, "MNQ 12-26", BarSourceLive, "22:38 CT on the live scale (as a live row would be after reconstruction)"},
		{1789097760000, 29068.75, 29067.25, "MNQ 12-26", BarSourceOffScale, "22:36 CT 3m, both sides on the replay scale (the aggregates were repainted too)"},
	}
	for i, x := range cases {
		tf := "1m"
		if i == 7 {
			tf = "3m"
		}
		if err := bh.db.Exec(`INSERT INTO bars(symbol,tf,open_time_ms,o,h,l,c,v,convention,contract,source) VALUES ('MNQ',?,?,?,?,?,?,1,'epoch_floor',?,'')`,
			tf, x.ms, x.o, x.o+1, x.c-1, x.c, x.contract).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := bh.migrateSourceColumn(); err != nil {
		t.Fatal(err)
	}
	for i, x := range cases {
		tf := "1m"
		if i == 7 {
			tf = "3m"
		}
		var r BarHistoryDB
		if err := bh.db.Where("symbol='MNQ' AND tf=? AND open_time_ms=? AND contract=?", tf, x.ms, x.contract).First(&r).Error; err != nil {
			t.Fatal(err)
		}
		if r.Source != x.want {
			t.Fatalf("case %d (%s): labelled %q, want %q", i, x.why, r.Source, x.want)
		}
		if r.O != x.o || r.C != x.c {
			t.Fatalf("case %d: VALUES changed (A24): o=%.2f c=%.2f", i, r.O, r.C)
		}
	}
	// idempotent
	c1, _ := bh.SourceCensus("MNQ")
	if err := bh.migrateSourceColumn(); err != nil {
		t.Fatal(err)
	}
	c2, _ := bh.SourceCensus("MNQ")
	for k, v := range c1 {
		if c2[k] != v {
			t.Fatalf("second run changed %q: %d→%d", k, v, c2[k])
		}
	}
}

// NO READER TAKES AN OFF-SCALE ROW, and the two repair paths — a live bar, or
// a VERIFIED replay released by the hold — overwrite it. Nothing else may
// write the label: the wire discards an off-scale replay, it never files one.
func TestOffScaleRowsAreUnreadableAndRepairable(t *testing.T) {
	bh := rollStore(t)
	const a, b, c = int64(1789096440000), int64(1789096500000), int64(1789096560000)
	for _, ms := range []int64{a, b, c} {
		if err := bh.db.Exec(`INSERT INTO bars(symbol,tf,open_time_ms,o,h,l,c,v,convention,contract,source) VALUES ('MNQ','1m',?,29076,29077,29075,29076,1,'epoch_floor','MNQ 12-26',?)`, ms, BarSourceOffScale).Error; err != nil {
			t.Fatal(err)
		}
	}
	last, _ := bh.LastNBarsOn("MNQ", "1m", "MNQ 12-26", 10)
	between, _ := bh.BarsBetweenOn("MNQ", "1m", "MNQ 12-26", a, c+1)
	if len(last) != 0 || len(between) != 0 {
		t.Fatalf("a reader handed out off-scale rows: last=%d between=%d", len(last), len(between))
	}
	if IsReadableSource(BarSourceOffScale) {
		t.Fatal("IsReadableSource must refuse off-scale")
	}
	// repair path 1: the minute as it traded (a reconstruction writes live)
	if err := bh.InsertBars([]BarHistoryDB{bsRow(a, 29366.5, BarSourceLive)}); err != nil {
		t.Fatal(err)
	}
	if r := readOne(t, bh, a); r.C != 29366.5 || r.Source != BarSourceLive {
		t.Fatalf("a live row must overwrite an off-scale row, got %.2f %q", r.C, r.Source)
	}
	// repair path 2: a replay the ring judged on-scale, released by the hold
	if err := bh.InsertBars([]BarHistoryDB{bsRow(b, 29365, BarSourceHistorical)}); err != nil {
		t.Fatal(err)
	}
	if r := readOne(t, bh, b); r.C != 29365 || r.Source != BarSourceHistorical {
		t.Fatalf("a verified replay must overwrite an off-scale row, got %.2f %q", r.C, r.Source)
	}
	// the label is the migration's alone
	if err := bh.InsertBars([]BarHistoryDB{bsRow(c, 1, BarSourceOffScale)}); err == nil {
		t.Fatal("InsertBars accepted replay:off-scale from a caller — only the measured backfill may write it")
	}
	if r := readOne(t, bh, c); r.C != 29076 {
		t.Fatal("the refused insert changed the row")
	}
}
