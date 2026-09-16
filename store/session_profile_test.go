package store

import "testing"

func TestSessionProfileSaveIfAbsentRestartSafe(t *testing.T) {
	sp := newLevelStateStore(t).SessionProfile()

	first := &SessionProfileDB{Symbol: "MNQ", SessionDate: "2026-08-13", POC: 15600, VAH: 15620, VAL: 15580}
	wrote, err := sp.SaveIfAbsent(first)
	if err != nil || !wrote {
		t.Fatalf("first save: wrote=%v err=%v", wrote, err)
	}

	// Restart replay: the same session re-derived — must NOT dup or overwrite.
	wrote2, err := sp.SaveIfAbsent(&SessionProfileDB{Symbol: "MNQ", SessionDate: "2026-08-13", POC: 99999})
	if err != nil || wrote2 {
		t.Fatalf("re-save must be a no-op: wrote=%v err=%v", wrote2, err)
	}

	rows, err := sp.List("MNQ", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want exactly 1 row (no dupes), got %d", len(rows))
	}
	if rows[0].POC != 15600 {
		t.Fatalf("frozen session must not be overwritten: POC=%v want 15600", rows[0].POC)
	}

	if n, _ := sp.Count("MNQ"); n != 1 {
		t.Fatalf("count = %d want 1", n)
	}
	if ex, _ := sp.Exists("MNQ", "2026-08-13"); !ex {
		t.Fatalf("Exists should be true for a stored session")
	}
	if ex, _ := sp.Exists("MNQ", "2026-08-14"); ex {
		t.Fatalf("Exists should be false for an unstored session")
	}
}

func TestSessionProfileListNewestFirst(t *testing.T) {
	sp := newLevelStateStore(t).SessionProfile()
	for _, d := range []string{"2026-08-11", "2026-08-13", "2026-08-12"} {
		if _, err := sp.SaveIfAbsent(&SessionProfileDB{Symbol: "MNQ", SessionDate: d, POC: 15000}); err != nil {
			t.Fatalf("save %s: %v", d, err)
		}
	}
	rows, _ := sp.List("MNQ", 10)
	if len(rows) != 3 || rows[0].SessionDate != "2026-08-13" {
		t.Fatalf("List must be newest-first: %+v", rows)
	}
}
