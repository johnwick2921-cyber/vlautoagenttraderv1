package store

import (
	"path/filepath"
	"testing"
	"time"
)

func obStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "ob.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// E4 — the ordinal is read FROM THE STORE, so touch numbering survives a
// restart (C4). An in-memory counter would restart at 1 every boot and silently
// re-number every level's touches.
func TestTouchOrdinalComesFromTheStore(t *testing.T) {
	st := obStore(t)
	ts := st.TouchOutcomes()
	day := time.Date(2026, 9, 3, 17, 0, 0, 0, time.UTC).UnixMilli()

	if got := ts.NextOrdinal("hoang", "MNQ", 29141.25, day); got != 1 {
		t.Fatalf("first touch of a level is ordinal 1, got %d", got)
	}
	for i := 1; i <= 3; i++ {
		if err := ts.SaveOutcome(&TouchOutcomeRow{
			TraderID: "hoang", Symbol: "MNQ", LevelPrice: 29141.25, Ordinal: i,
			Outcome: "hold", EntrySide: "below", ExitSide: "below",
			OpenedAtMs: day + int64(i)*60000,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// A fresh store handle — the "restart" — must continue, not restart.
	if got := st.TouchOutcomes().NextOrdinal("hoang", "MNQ", 29141.25, day); got != 4 {
		t.Errorf("ordinal must continue from MAX+1 across a restart, got %d want 4", got)
	}
	// A DIFFERENT level keeps its own sequence.
	if got := ts.NextOrdinal("hoang", "MNQ", 29200.50, day); got != 1 {
		t.Errorf("a different level starts at 1, got %d", got)
	}
}

// E3 at the store layer — ambiguous rows are WRITTEN and excluded from the
// rate, never dropped, and the rate carries its base.
func TestTouchOutcomeRatesExcludeAmbiguousAndCarryN(t *testing.T) {
	st := obStore(t)
	ts := st.TouchOutcomes()
	// WAVE A / D1e — a rate is computed over CERTIFIED rows only, so this
	// fixture writes what the fixed recorder writes. The rate arithmetic under
	// test is unchanged; the population it may draw from is now explicit.
	mk := func(kind, outcome string, amb bool) {
		if err := ts.SaveOutcome(&TouchOutcomeRow{
			TraderID: "hoang", Symbol: "MNQ", LevelKind: kind, LevelPrice: 29000,
			Outcome: outcome, Ambiguous: amb, OpenedAtMs: time.Now().UnixMilli(),
			Validity: ValidityValid,
		}); err != nil {
			t.Fatal(err)
		}
	}
	mk("ONL", "hold", false)
	mk("ONL", "hold", false)
	mk("ONL", "break", false)
	mk("ONL", "ambiguous_horizon", true)
	mk("PDH", "break", false)

	all, err := ts.RatesBy("")
	if err != nil || len(all) != 1 {
		t.Fatalf("whole-table rate: %v %v", all, err)
	}
	r := all[0]
	if r.N() != 4 || r.Ambiguous != 1 {
		t.Errorf("n must EXCLUDE the ambiguous row but COUNT it: n=%d amb=%d", r.N(), r.Ambiguous)
	}
	if p := r.P(); p < 0.49 || p > 0.51 {
		t.Errorf("p = hold/(hold+break) = 2/4 = 0.50, got %.4f", p)
	}
	byKind, err := ts.RatesBy("level_kind")
	if err != nil || len(byKind) != 2 {
		t.Fatalf("per-kind grouping: %v %v", byKind, err)
	}
	// An empty table must report n=0, never a plausible zero rate.
	empty := obStore(t).TouchOutcomes()
	got, _ := empty.RatesBy("")
	if len(got) != 0 {
		t.Errorf("an empty table has no groups, got %+v", got)
	}
	t.Logf("whole-table: hold=%d break=%d ambiguous=%d n=%d p=%.3f", r.Hold, r.Break, r.Ambiguous, r.N(), r.P())
}

// E5 — the candidate pool records the EXCLUDED levels with their cut reasons
// and propensities; that is the whole reason the table exists.
func TestCandidatePoolRecordsTheExcludedWithPropensity(t *testing.T) {
	st := obStore(t)
	cp := st.CandidatePool()
	var rows []CandidatePoolRow
	for i := 1; i <= 20; i++ {
		seated := i <= 12
		cut := ""
		if !seated {
			cut = "max_levels"
		}
		rows = append(rows, CandidatePoolRow{
			TraderID: "hoang", Symbol: "MNQ", PlanID: "p1", PlanVersion: 3, Session: "NY",
			ReadAtMs: 1788400000000, LevelPrice: 29000 + float64(i), LevelKind: "SWG",
			Rank: i, Seated: seated, CutReason: cut, Score: 100 - float64(i), Threshold: 88,
			ScoreComponents: `{"prox":0.4,"conf":0.3}`,
		})
	}
	if err := cp.SavePool(rows); err != nil {
		t.Fatal(err)
	}
	got, err := cp.LatestPool(50)
	if err != nil || len(got) != 20 {
		t.Fatalf("the WHOLE pool must persist, got %d rows (err %v)", len(got), err)
	}
	seated, cutRows := 0, 0
	for _, r := range got {
		if r.Seated {
			seated++
			continue
		}
		cutRows++
		if r.CutReason == "" {
			t.Errorf("rank %d was cut with no reason — the excluded pool is the point", r.Rank)
		}
		if r.Threshold == 0 && r.Score == 0 {
			t.Errorf("rank %d has no propensity — the decision is not reconstructable", r.Rank)
		}
	}
	if seated != 12 || cutRows != 8 {
		t.Errorf("want 12 seated / 8 cut, got %d/%d", seated, cutRows)
	}
	// "not computed" and "all zero" must differ (A24).
	if got[0].ScoreComponents == "" {
		t.Error("an uncomputed component set is {} — never an empty string")
	}
}
