package store

import (
	"path/filepath"
	"testing"
)

// workerEpochStore opens a temp sqlite Store for the epoch pins.
func workerEpochStore(t *testing.T) *WorkerEpochStore {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st.WorkerEpoch()
}

func TestWorkerEpochStoreCurrentIsZeroBeforeBump(t *testing.T) {
	s := workerEpochStore(t)
	n, err := s.Current()
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if n != 0 {
		t.Fatalf("epoch before any bump = %d — want 0", n)
	}
}

func TestWorkerEpochStoreBumpIncrements(t *testing.T) {
	s := workerEpochStore(t)
	for want := int64(1); want <= 3; want++ {
		got, err := s.Bump()
		if err != nil {
			t.Fatalf("bump %d: %v", want, err)
		}
		if got != want {
			t.Fatalf("bump #%d returned %d — want %d", want, got, want)
		}
		cur, err := s.Current()
		if err != nil {
			t.Fatalf("current after bump %d: %v", want, err)
		}
		if cur != want {
			t.Fatalf("current after bump = %d — want %d", cur, want)
		}
	}
	if n, err := s.WorkEpochRowCount(); err != nil || n != 1 {
		t.Fatalf("row count = %d (err %v) — the epoch is a single row", n, err)
	}
}
