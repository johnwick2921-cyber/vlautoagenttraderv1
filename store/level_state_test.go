package store

import (
	"path/filepath"
	"testing"
)

func newLevelStateStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		st.Plan().Close()
		_ = st.Close()
	})
	return st
}

func poc(symbol, originDate string, bin int, price float64) *LevelStateDB {
	return &LevelStateDB{
		Symbol:     symbol,
		LevelType:  LevelTypePOC,
		OriginDate: originDate,
		BinIndex:   bin,
		Price:      price,
	}
}

func TestMakeLevelKeyDeterministic(t *testing.T) {
	a := MakeLevelKey("", "MNQ", LevelTypePOC, "2026-08-14", 12345)
	b := MakeLevelKey("", "MNQ", LevelTypePOC, "2026-08-14", 12345)
	if a != b || a != "MNQ|POC|2026-08-14|12345" {
		t.Fatalf("level key = %q", a)
	}
}

func TestEnsureLevelCreatesAndPreservesState(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	l := poc("MNQ", "2026-08-14", 12345, 15431.75)
	if err := ls.EnsureLevel(l); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, err := ls.Get(l.LevelKey)
	if err != nil || got == nil {
		t.Fatalf("get: %v %v", got, err)
	}
	if got.Freshness != FreshnessA || got.TimesTested != 0 || got.Consumed {
		t.Fatalf("fresh level state wrong: %+v", got)
	}
	// Accumulate some state, then re-derive the same identity next session.
	if err := ls.RecordTest(l.LevelKey); err != nil {
		t.Fatalf("record test: %v", err)
	}
	redrawn := poc("MNQ", "2026-08-14", 12345, 15432.00) // new price, same identity
	if err := ls.EnsureLevel(redrawn); err != nil {
		t.Fatalf("re-ensure: %v", err)
	}
	after, _ := ls.Get(l.LevelKey)
	if after.TimesTested != 1 {
		t.Fatalf("re-derive must preserve times_tested, got %d", after.TimesTested)
	}
	if after.Price != 15432.00 {
		t.Fatalf("re-derive should refresh price, got %v", after.Price)
	}
	if after.Freshness != FreshnessA {
		t.Fatalf("re-derive must not reset a burned level's freshness")
	}
}

func TestRecordTestIncrements(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	l := poc("MNQ", "2026-08-14", 1, 100)
	_ = ls.EnsureLevel(l)
	for i := 0; i < 3; i++ {
		if err := ls.RecordTest(l.LevelKey); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	got, _ := ls.Get(l.LevelKey)
	if got.TimesTested != 3 {
		t.Fatalf("times_tested = %d want 3", got.TimesTested)
	}
}

func TestMarkConsumed(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	l := poc("MNQ", "2026-08-14", 1, 100)
	_ = ls.EnsureLevel(l)
	if err := ls.MarkConsumed(l.LevelKey); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, _ := ls.Get(l.LevelKey)
	if !got.Consumed || got.Freshness != FreshnessDone {
		t.Fatalf("consumed state wrong: %+v", got)
	}
}

func TestDecrementFreshness(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	l := poc("MNQ", "2026-08-14", 1, 100)
	_ = ls.EnsureLevel(l)
	for _, want := range []string{FreshnessB, FreshnessC, FreshnessDone, FreshnessDone} {
		got, err := ls.DecrementFreshness(l.LevelKey)
		if err != nil {
			t.Fatalf("decrement: %v", err)
		}
		if got != want {
			t.Fatalf("decrement -> %q want %q", got, want)
		}
	}
	final, _ := ls.Get(l.LevelKey)
	if !final.Consumed {
		t.Fatalf("reaching done must mark consumed: %+v", final)
	}
}

func TestCrossSessionPersistenceDistinctRows(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	// Same symbol/type/bin but different origin_date → two distinct levels.
	_ = ls.EnsureLevel(poc("MNQ", "2026-08-13", 1, 100))
	_ = ls.EnsureLevel(poc("MNQ", "2026-08-14", 1, 100))
	rows, err := ls.ListForSymbol("MNQ")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 distinct rows, got %d", len(rows))
	}
}

func TestListValidExcludesConsumedAndDone(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	fresh := poc("MNQ", "2026-08-14", 1, 100)
	used := poc("MNQ", "2026-08-14", 2, 200)
	done := poc("MNQ", "2026-08-14", 3, 300)
	_ = ls.EnsureLevel(fresh)
	_ = ls.EnsureLevel(used)
	_ = ls.EnsureLevel(done)
	_ = ls.MarkConsumed(used.LevelKey)
	for i := 0; i < 3; i++ {
		_, _ = ls.DecrementFreshness(done.LevelKey) // A->B->C->done
	}
	valid, err := ls.ListValid("MNQ")
	if err != nil {
		t.Fatalf("list valid: %v", err)
	}
	if len(valid) != 1 || valid[0].LevelKey != fresh.LevelKey {
		t.Fatalf("ListValid = %+v want only the fresh level", valid)
	}
}
