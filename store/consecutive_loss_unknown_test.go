package store

import (
	"path/filepath"
	"testing"
	"time"
)

// TestUnresolvedCloseBreaksTheLosingRun — D2's UNKNOWN rule.
//
// The dispatch: "UNKNOWN (an unresolvable P&L on a recent close) never counts
// toward the run and never blocks."
//
// CountConsecutiveLossesSince excluded unresolved rows in its WHERE and said so
// in a comment — "never counted either way". But excluding a row from the scan
// does not make it neutral: it makes it TRANSPARENT. A run of 3 losses, an
// unresolvable close, then 3 more losses was counted as SIX, because the
// unknown row simply was not there to break it.
//
// Transparent is the destructive direction. Blocking is the harmful branch of
// this gate, and bridging makes the run LONGER, so an unknown close makes a
// halt MORE likely — which is exactly what A24 forbids: UNKNOWN never takes the
// destructive branch. An unresolvable close means we do not know whether the
// trade won; the honest reading of "we do not know" is that the run of KNOWN
// losses has ended.
func TestUnresolvedCloseBreaksTheLosingRun(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().Add(-2 * time.Hour).UnixMilli()
	mk := func(i int, pnl *float64, reason string) {
		p := &TraderPosition{
			TraderID: "t1", Symbol: "MNQ", Side: "LONG", Status: "CLOSED",
			EntryQuantity: 1, Quantity: 1, EntryPrice: 100,
			ExitTime: base + int64(i)*60000, CloseReason: reason,
		}
		if err := st.Position().Create(p); err != nil {
			t.Fatal(err)
		}
		// Create() forces status=OPEN; the closed shape is set explicitly.
		upd := map[string]any{"status": "CLOSED"}
		if pnl != nil {
			upd["pnl_corrected"] = *pnl
		}
		if err := st.GormDB().Model(&TraderPosition{}).Where("id = ?", p.ID).
			Updates(upd).Error; err != nil {
			t.Fatal(err)
		}
	}
	loss := -50.0
	// oldest → newest: L L L  <unresolved>  L L L
	mk(1, &loss, CloseReasonStop)
	mk(2, &loss, CloseReasonStop)
	mk(3, &loss, CloseReasonStop)
	mk(4, nil, CloseReasonSync) // pnl_corrected NULL — UNRESOLVABLE
	mk(5, &loss, CloseReasonStop)
	mk(6, &loss, CloseReasonStop)
	mk(7, &loss, CloseReasonStop)

	got, err := st.Position().CountConsecutiveLossesSince("t1", base)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Fatalf("run = %d, want 3 — an UNRESOLVABLE close bridged two runs of 3 into one of %d. "+
			"An unknown P&L must END the run of known losses, not join them: bridging makes a halt "+
			"MORE likely, which is the destructive branch (A24).", got, got)
	}
}

// A resolved WIN still breaks the run, unchanged.
func TestWinBreaksTheLosingRun(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().Add(-2 * time.Hour).UnixMilli()
	for i, pnl := range []float64{-50, -50, 25, -50, -50} {
		v := pnl
		p := &TraderPosition{
			TraderID: "t1", Symbol: "MNQ", Side: "LONG", Status: "CLOSED",
			EntryQuantity: 1, Quantity: 1, EntryPrice: 100,
			ExitTime: base + int64(i+1)*60000, CloseReason: CloseReasonStop,
		}
		if err := st.Position().Create(p); err != nil {
			t.Fatal(err)
		}
		if err := st.GormDB().Model(&TraderPosition{}).Where("id = ?", p.ID).
			Updates(map[string]any{"status": "CLOSED", "pnl_corrected": v}).Error; err != nil {
			t.Fatal(err)
		}
	}
	got, _ := st.Position().CountConsecutiveLossesSince("t1", base)
	if got != 2 {
		t.Fatalf("run = %d, want 2 (a win ends the run)", got)
	}
}
