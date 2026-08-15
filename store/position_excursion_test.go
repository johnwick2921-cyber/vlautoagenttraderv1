package store

import (
	"path/filepath"
	"testing"
)

func TestPositionExcursionColumns(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := st.Position()

	pos := &TraderPosition{
		TraderID: "t1", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 15600,
		Status: "OPEN", EntryTime: 1, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := ps.Create(pos); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ps.SetEntryConfidence(pos.ID, 72); err != nil {
		t.Fatalf("set confidence: %v", err)
	}
	if err := ps.UpdateExcursion(pos.ID, 25.5, 40.0); err != nil {
		t.Fatalf("update excursion: %v", err)
	}

	got, err := ps.GetOpenPositions("t1")
	if err != nil || len(got) != 1 {
		t.Fatalf("get: %v n=%d", err, len(got))
	}
	p := got[0]
	if p.EntryConfidence != 72 || p.MAE != 25.5 || p.MFE != 40.0 {
		t.Fatalf("excursion columns not persisted: conf=%d mae=%v mfe=%v", p.EntryConfidence, p.MAE, p.MFE)
	}
}
