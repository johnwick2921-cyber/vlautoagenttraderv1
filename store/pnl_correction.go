package store

import (
	"fmt"
	"time"

	"nofx/logger"
)

// P0 pnl-record-integrity (2026-08-20) — one-time correction pass.
//
// Position #526 recorded −$1,458 on a 1-lot row (a manual NT8 flatten's
// 21-contract frame was attributed wholesale); the scope check found 37 rows
// since Aug 6 whose recorded P&L disagrees with their OWN stored prices ×
// row quantity (the aggregate-frame class + the Aug 9–12 two-trader
// contamination era). Corrections are ADDITIVE: pnl_corrected + a note; the
// original realized_pnl is never touched. All aggregate readers COALESCE.
//
// Idempotent via the system_config flag; WHERE-scoped; recompute = the best
// available truth (the dispatch's 1.4 rule — wire fills are not recoverable
// for the old rows; rows whose stored prices are themselves non-tick averages
// carry that suspicion in the note).

const pnlCorrectionFlag = "pnl_correction_2026_08_20_done"

func (s *Store) CorrectHistoricalPnL() {
	if v, err := s.GetSystemConfig(pnlCorrectionFlag); err == nil && v == "1" {
		return
	}
	type row struct {
		ID          int64
		Side        string
		Quantity    float64
		EntryPrice  float64
		ExitPrice   float64
		RealizedPnl float64
	}
	var rows []row
	// MNQ $2/pt; scope = the audited window (BE era start, 2026-08-06 UTC).
	sinceMs := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC).UnixMilli()
	if err := s.gdb.Table("trader_positions").
		Select("id, side, quantity, entry_price, exit_price, realized_pnl").
		Where("status='CLOSED' AND symbol='MNQ' AND exit_time >= ? AND pnl_corrected IS NULL", sinceMs).
		Scan(&rows).Error; err != nil {
		logger.Warnf("pnl-correction: scan failed: %v", err)
		return
	}
	corrected := 0
	for _, r := range rows {
		pts := r.ExitPrice - r.EntryPrice
		if r.Side == "SHORT" {
			pts = -pts
		}
		recomputed := pts * r.Quantity * 2.0
		delta := r.RealizedPnl - recomputed
		if delta < 0.5 && delta > -0.5 {
			continue
		}
		note := fmt.Sprintf("P0 2026-08-20: recorded %.2f disagreed with stored prices × row qty by %+.2f — corrected to the recompute (aggregate-frame / contamination class; original preserved)", r.RealizedPnl, delta)
		if nonTick(r.EntryPrice) || nonTick(r.ExitPrice) {
			note += "; stored price itself is a non-tick AVERAGE from an aggregate frame — best available truth"
		}
		if err := s.gdb.Table("trader_positions").
			Where("id = ? AND pnl_corrected IS NULL", r.ID).
			Updates(map[string]any{"pnl_corrected": recomputed, "pnl_correction_note": note}).Error; err != nil {
			logger.Warnf("pnl-correction: row %d failed: %v", r.ID, err)
			continue
		}
		corrected++
	}
	_ = s.SetSystemConfig(pnlCorrectionFlag, "1")
	logger.Warnf("⚖️ pnl-correction complete: %d row(s) corrected additively (pnl_corrected + note; originals preserved) of %d candidates since Aug 6.", corrected, len(rows))
}

func nonTick(p float64) bool {
	q := p * 4
	return q != float64(int64(q))
}
