// P0 pnl-record-integrity (2026-08-20) — V2 pins for the one-time correction
// pass. The #526 shape: a manual NT8 flatten's 21-contract aggregate frame was
// attributed to a 1-lot row (recorded −1458 vs a true ≈ −69.43). The pass must
// correct ADDITIVELY (original preserved), skip agreeing rows, be idempotent,
// and every aggregate reader must see the corrected value.
package store

import (
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestCorrectHistoricalPnL(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	exitMs := time.Date(2026, 8, 19, 21, 16, 27, 0, time.UTC).UnixMilli()

	bad := &TraderPosition{
		TraderID: "tr1", Account: "Sim101", Symbol: "MNQ", Side: "SHORT",
		Quantity: 1, EntryPrice: 29626.25, ExitPrice: 29660.964286,
		RealizedPnL: -1458.00000000002, Status: "CLOSED",
		CloseReason: "sync", EntryTime: exitMs - 3600_000, ExitTime: exitMs, CreatedAt: exitMs,
	}
	clean := &TraderPosition{
		TraderID: "tr1", Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 29600.00, ExitPrice: 29610.00,
		RealizedPnL: 20.0, Status: "CLOSED",
		CloseReason: "take_profit", EntryTime: exitMs - 7200_000, ExitTime: exitMs - 3600_000, CreatedAt: exitMs,
	}
	if err := st.gdb.Create(bad).Error; err != nil {
		t.Fatal(err)
	}
	if err := st.gdb.Create(clean).Error; err != nil {
		t.Fatal(err)
	}

	st.CorrectHistoricalPnL()

	var got TraderPosition
	if err := st.gdb.First(&got, bad.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.PnlCorrected == nil {
		t.Fatal("the #526-shape row must receive pnl_corrected")
	}
	// SHORT: (29626.25 − 29660.964286) × 1 × $2 = −69.428572
	if want := -69.428572; math.Abs(*got.PnlCorrected-want) > 0.01 {
		t.Fatalf("pnl_corrected = %.4f, want ≈ %.4f", *got.PnlCorrected, want)
	}
	if got.RealizedPnL != bad.RealizedPnL {
		t.Fatal("original realized_pnl must NEVER be edited (audit trail)")
	}
	if got.PnlCorrectionNote == "" {
		t.Fatal("a correction must carry its note")
	}
	if got.EffectivePnL() != *got.PnlCorrected {
		t.Fatal("EffectivePnL must prefer the correction")
	}

	var cleanGot TraderPosition
	if err := st.gdb.First(&cleanGot, clean.ID).Error; err != nil {
		t.Fatal(err)
	}
	if cleanGot.PnlCorrected != nil {
		t.Fatal("an agreeing row must not be touched")
	}
	if cleanGot.EffectivePnL() != 20.0 {
		t.Fatal("EffectivePnL must fall back to realized_pnl when uncorrected")
	}

	// The session-day guardrail sum reads the CORRECTED value.
	pnl, _, err := st.Position().GetSessionDayActivity("tr1", exitMs-86400_000)
	if err != nil {
		t.Fatal(err)
	}
	if want := -69.428572 + 20.0; math.Abs(pnl-want) > 0.01 {
		t.Fatalf("session-day sum = %.4f, want corrected %.4f (raw would be %.4f)", pnl, want, bad.RealizedPnL+20.0)
	}

	// Idempotent: flag set, second run touches nothing further.
	if v, _ := st.GetSystemConfig("pnl_correction_2026_08_20_done"); v != "1" {
		t.Fatal("done-flag must be set")
	}
	st.CorrectHistoricalPnL()
	var again TraderPosition
	_ = st.gdb.First(&again, bad.ID).Error
	if *again.PnlCorrected != *got.PnlCorrected {
		t.Fatal("second run must be a no-op")
	}
}
