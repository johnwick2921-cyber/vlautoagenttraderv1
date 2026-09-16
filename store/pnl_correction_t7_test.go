// T7 (2026-08-27) — the pnl_corrected column must be COMPLETE, not just the
// P0 disagreement rows: the backfill stamps every reconstructable row and the
// close path stamps every new close.
package store

import (
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestT7BackfillStampsAllRows(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	exitMs := time.Date(2026, 8, 19, 21, 16, 27, 0, time.UTC).UnixMilli()
	agree := &TraderPosition{
		TraderID: "tr1", Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 29600.00, ExitPrice: 29610.00,
		RealizedPnL: 20.0, Status: "CLOSED", CloseReason: "take_profit",
		EntryTime: exitMs - 7200_000, ExitTime: exitMs, CreatedAt: exitMs,
	}
	disagree := &TraderPosition{
		TraderID: "tr1", Account: "Sim101", Symbol: "MNQ", Side: "SHORT",
		Quantity: 1, EntryPrice: 29626.25, ExitPrice: 29660.964286,
		RealizedPnL: -1458.0, Status: "CLOSED", CloseReason: "sync",
		EntryTime: exitMs - 3600_000, ExitTime: exitMs, CreatedAt: exitMs,
	}
	noprice := &TraderPosition{
		TraderID: "tr1", Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 0, ExitPrice: 0,
		RealizedPnL: 0, Status: "CLOSED", CloseReason: "reconcile_flat",
		EntryTime: exitMs - 3600_000, ExitTime: exitMs, CreatedAt: exitMs,
	}
	for _, p := range []*TraderPosition{agree, disagree, noprice} {
		if err := st.gdb.Create(p).Error; err != nil {
			t.Fatal(err)
		}
	}

	st.BackfillPnlCorrectedAll()

	var got TraderPosition
	_ = st.gdb.First(&got, agree.ID).Error
	if got.PnlCorrected == nil || math.Abs(*got.PnlCorrected-20.0) > 0.01 {
		t.Fatalf("agreeing row pnl_corrected = %v, want 20.0", got.PnlCorrected)
	}
	got = TraderPosition{}
	_ = st.gdb.Where("id = ?", disagree.ID).First(&got).Error
	if got.PnlCorrected == nil || math.Abs(*got.PnlCorrected-(-69.428572)) > 0.01 {
		t.Fatalf("disagreeing row pnl_corrected = %v, want ≈ -69.43", *got.PnlCorrected)
	}
	if got.RealizedPnL != -1458.0 {
		t.Fatal("realized_pnl must remain untouched")
	}
	got = TraderPosition{}
	_ = st.gdb.Where("id = ?", noprice.ID).First(&got).Error
	if got.PnlCorrected != nil {
		t.Fatalf("no-price row must stay NULL (unreconstructable), got %v", got.PnlCorrected)
	}
	// Idempotence: the flag blocks a second run.
	if v, _ := st.GetSystemConfig(pnlBackfillAllFlag); v != "1" {
		t.Fatal("backfill flag not set")
	}
}

func TestT7ClosePathStamp(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	exitMs := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC).UnixMilli()
	p := &TraderPosition{
		TraderID: "tr1", Account: "Sim101", Symbol: "MNQ", Side: "SHORT",
		Quantity: 1, EntryPrice: 29580, ExitPrice: 29611.25,
		RealizedPnL: -62.5, Status: "CLOSED", CloseReason: "sync",
		EntryTime: exitMs - 3600_000, ExitTime: exitMs, CreatedAt: exitMs,
	}
	if err := st.gdb.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	st.StampPnlCorrectedOnClose(p.ID, -62.5, -62.5)
	var got TraderPosition
	_ = st.gdb.First(&got, p.ID).Error
	if got.PnlCorrected == nil || math.Abs(*got.PnlCorrected-(-62.5)) > 0.01 {
		t.Fatalf("close-path stamp = %v, want -62.5", got.PnlCorrected)
	}
	// A second stamp must NOT overwrite (WHERE pnl_corrected IS NULL).
	st.StampPnlCorrectedOnClose(p.ID, -100, -100)
	_ = st.gdb.First(&got, p.ID).Error
	if math.Abs(*got.PnlCorrected-(-62.5)) > 0.01 {
		t.Fatalf("re-stamp overwrote: %v", got.PnlCorrected)
	}
}
