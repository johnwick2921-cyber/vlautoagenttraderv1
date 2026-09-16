package store

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func waveAStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "wavea_mig.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// D1e — THE CLASSIFIER. Duplicates are marked, everything else the old recorder
// wrote is marked UNVERIFIED, and NOTHING legacy is ever marked valid.
func TestWaveAMigrationClassifiesAndNeverBlessesLegacyRows(t *testing.T) {
	st := waveAStore(t)
	ts := st.TouchOutcomes()
	// One episode written FIVE times — the live shape (RTH-L is 140 rows of 14).
	for i := 0; i < 5; i++ {
		if err := ts.SaveOutcome(&TouchOutcomeRow{
			TraderID: "hoang", Symbol: "MNQ", LevelKind: "RTH-L", LevelPrice: 29199.25,
			Outcome: "break", OpenedAtMs: 1788400000000, Ordinal: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// One genuinely distinct episode.
	if err := ts.SaveOutcome(&TouchOutcomeRow{
		TraderID: "hoang", Symbol: "MNQ", LevelKind: "RTH-L", LevelPrice: 29199.25,
		Outcome: "hold", OpenedAtMs: 1788400060000, Ordinal: 1,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := st.RunWaveARecordMigration()
	if err != nil {
		t.Fatal(err)
	}
	if got.TouchDuplicate != 4 {
		t.Fatalf("5 copies of one episode must leave 1 and mark 4, marked %d", got.TouchDuplicate)
	}
	if got.TouchLegacy != 2 {
		t.Fatalf("the surviving copy + the distinct episode must be legacy:unverified, marked %d", got.TouchLegacy)
	}
	v := ts.CountByValidity()
	if v[ValidityValid] != 0 {
		t.Fatalf("NO legacy row may ever be marked valid — found %d. Certification is something only the fixed recorder can confer", v[ValidityValid])
	}
	// And none of them may reach a rate.
	rates, _ := ts.RatesBy("")
	n := 0
	for _, r := range rates {
		n += r.N()
	}
	if n != 0 {
		t.Fatalf("legacy rows reached a rate: n=%d", n)
	}
	// IDEMPOTENT: a second run finds nothing.
	again, err := st.RunWaveARecordMigration()
	if err != nil {
		t.Fatal(err)
	}
	if again.TouchDuplicate != 0 || again.TouchLegacy != 0 {
		t.Fatalf("migration is not idempotent: second run marked dup=%d legacy=%d", again.TouchDuplicate, again.TouchLegacy)
	}
}

// E5 — NULL IS NOT ZERO. The pre-E4 path wrote 0 where it could not compute.
// After the migration those read NULL (UNMEASURED); a genuine zero written
// from here on stays 0.0 and remains distinguishable from it.
func TestWaveAMigrationConvertsPreE4ZerosToNullAndCountsThem(t *testing.T) {
	st := waveAStore(t)
	ps := st.Position()
	mk := func(mae, mfe float64) int64 {
		maeP, mfeP := mae, mfe
		p := &TraderPosition{
			TraderID: "hoang", Symbol: "MNQ", Side: "SHORT", Status: "CLOSED",
			EntryPrice: 29000, ExitPrice: 29010, Quantity: 0, EntryQuantity: 1,
			EntryTime: time.Now().UnixMilli(), ExitTime: time.Now().UnixMilli(),
			MAE: &maeP, MFE: &mfeP,
		}
		if err := st.GormDB().Create(p).Error; err != nil {
			t.Fatal(err)
		}
		return int64(p.ID)
	}
	zeroID := mk(0, 0)
	realID := mk(17.25, 71.75)

	got, err := st.RunWaveARecordMigration()
	if err != nil {
		t.Fatal(err)
	}
	if got.MAEZeroed != 1 || got.MFEZeroed != 1 {
		t.Fatalf("exactly one row's mae and mfe were 0 and must be nulled WITH A COUNT, got mae=%d mfe=%d", got.MAEZeroed, got.MFEZeroed)
	}
	var nullCount int64
	st.GormDB().Model(&TraderPosition{}).Where("id = ? AND mae IS NULL", zeroID).Count(&nullCount)
	if nullCount != 1 {
		t.Fatalf("the pre-E4 zero must now read NULL (UNMEASURED), not 0.0")
	}
	// The measured row is untouched — a migration that also rewrites real data
	// is not a migration.
	var p TraderPosition
	if err := st.GormDB().First(&p, realID).Error; err != nil {
		t.Fatal(err)
	}
	if p.MAE == nil || p.MFE == nil || *p.MAE != 17.25 || *p.MFE != 71.75 {
		t.Fatalf("a measured excursion must be untouched, got mae=%v mfe=%v", p.MAE, p.MFE)
	}
	_ = ps
}

// The boot line must SAY how many arms are live, read from the table — the
// cutover on 2026-09-05 was blocked by two never-placed arms (104, 105) whose
// terminalization otherwise lived only in a chat log.
func TestRecordBootLineReportsTheArmsCensusFromTheTable(t *testing.T) {
	st := waveAStore(t)
	ao := st.ArmedOrders()
	// Distinct scenarios: UpsertArm keys on (trader, plan, session, scenario,
	// side), so identical rows collapse into one.
	for i, tc := range []struct{ state string }{{"armed"}, {"superseded"}, {"superseded"}, {"cancelled"}} {
		row := &ArmedOrderDB{TraderID: "hoang", PlanID: "P", Session: "NY",
			Scenario: fmt.Sprintf("S%d", i+1),
			Side:     "long", EntryPx: 100, StopPx: 95, TargetPx: 110, State: "armed"}
		if err := ao.UpsertArm(row); err != nil {
			t.Fatal(err)
		}
		if tc.state != "armed" {
			if err := ao.SetState(row.ID, tc.state, "fixture"); err != nil {
				t.Fatal(err)
			}
		}
	}
	line := st.WaveARecordBootLine(false, WaveACounts{}, "")
	for _, want := range []string{"arms live=1", "superseded=2", "cancelled=1"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line must report the census READ from the table; missing %q in:\n%s", want, line)
		}
	}
	t.Logf("%s", line)
}
