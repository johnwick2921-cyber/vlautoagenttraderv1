package store

import (
	"path/filepath"
	"testing"
)

// P0-cleanup (2026-08-19) — two day-plan traders must NEVER share burn/freshness
// state: the identity key is trader-scoped.
func TestLevelStateTraderIsolation(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ls := st.LevelState()

	keyA := MakeLevelKey("trader-A", "MNQ", LevelTypePDL, "2026-08-18", 42)
	keyB := MakeLevelKey("trader-B", "MNQ", LevelTypePDL, "2026-08-18", 42)
	if keyA == keyB {
		t.Fatalf("trader-scoped keys must differ: %q", keyA)
	}

	rowA := &LevelStateDB{LevelKey: keyA, TraderID: "trader-A", Symbol: "MNQ", LevelType: LevelTypePDL, OriginDate: "2026-08-18", BinIndex: 42, Price: 29680.75}
	if err := ls.EnsureLevel(rowA); err != nil {
		t.Fatal(err)
	}
	// Burn trader A's level.
	if _, err := ls.RecordPlay(keyA, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := ls.RecordPlay(keyA, 2000); err != nil {
		t.Fatal(err)
	}
	// Trader B's state is untouched (separate row).
	got, err := ls.Get(keyB)
	if err == nil && got != nil {
		t.Fatalf("trader-B must have NO state for a level only trader-A played: %+v", got)
	}
	// Trader A's row decayed.
	a, err := ls.Get(keyA)
	if err != nil || a == nil {
		t.Fatalf("trader-A state must exist: %v %v", a, err)
	}
	if a.Freshness == FreshnessA {
		t.Fatalf("trader-A freshness must have decayed after two plays, got %q", a.Freshness)
	}
}
