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

// pnlBackfillAllFlag gates the T7 backfill (2026-08-27) — unlike the P0 pass
// (which wrote ONLY the disagreeing rows), this one stamps pnl_corrected on
// EVERY closed MNQ row whose stored prices allow a recompute, so the column is
// complete for the audit trail and "all PnL = pnl_corrected" becomes literally
// true. Disagreements (|realized − recompute| ≥ $0.50) re-arm the class-killer
// WARN.
const pnlBackfillAllFlag = "pnl_backfill_all_2026_08_27_done"

// BackfillPnlCorrectedAll stamps pnl_corrected = recomputed (stored prices ×
// row qty × point value) on every CLOSED MNQ row with a NULL correction since
// the P0 era start (2026-08-06 UTC). Rows with unusable stored prices (entry or
// exit ≤ 0) are skipped and logged. Idempotent per row (WHERE pnl_corrected IS
// NULL) + flag-gated. Disagreeing rows WARN as the Δ>$0.50 class-killer.
func (s *Store) BackfillPnlCorrectedAll() {
	if v, err := s.GetSystemConfig(pnlBackfillAllFlag); err == nil && v == "1" {
		return
	}
	sinceMs := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC).UnixMilli()
	var rows []struct {
		ID          int64
		Side        string
		Quantity    float64
		EntryPrice  float64
		ExitPrice   float64
		RealizedPnl float64
	}
	if err := s.gdb.Table("trader_positions").
		Select("id, side, quantity, entry_price, exit_price, realized_pnl").
		Where("status='CLOSED' AND symbol='MNQ' AND exit_time >= ? AND pnl_corrected IS NULL", sinceMs).
		Scan(&rows).Error; err != nil {
		logger.Warnf("pnl-backfill: scan failed: %v", err)
		return
	}
	stamped, warned, skipped := 0, 0, 0
	for _, r := range rows {
		if r.EntryPrice <= 0 || r.ExitPrice <= 0 {
			skipped++
			continue
		}
		pts := r.ExitPrice - r.EntryPrice
		if r.Side == "SHORT" {
			pts = -pts
		}
		recomputed := pts * r.Quantity * 2.0
		delta := r.RealizedPnl - recomputed
		note := fmt.Sprintf("T7 2026-08-27: pnl_corrected = recompute (stored prices × row qty × $2) — realized was %.2f (Δ%+.2f)", r.RealizedPnl, delta)
		if delta >= 0.5 || delta <= -0.5 {
			warned++
			logger.Warnf("⚖️ pnl class-killer: row %d realized %.2f vs recompute %.2f (Δ%+.2f) — aggregate-frame/contamination class; corrected additively.", r.ID, r.RealizedPnl, recomputed, delta)
		}
		if err := s.gdb.Table("trader_positions").
			Where("id = ? AND pnl_corrected IS NULL", r.ID).
			Updates(map[string]any{"pnl_corrected": recomputed, "pnl_correction_note": note}).Error; err != nil {
			logger.Warnf("pnl-backfill: row %d failed: %v", r.ID, err)
			continue
		}
		stamped++
	}
	_ = s.SetSystemConfig(pnlBackfillAllFlag, "1")
	logger.Warnf("⚖️ pnl-backfill complete: %d stamped · %d class-killer disagreements · %d skipped (no stored prices) of %d candidates.", stamped, warned, skipped, len(rows))
}

// StampPnlCorrectedOnClose is the CLOSE-PATH writer (T7): every closed row gets
// pnl_corrected immediately, so the column is non-NULL on all NEW closes. The
// Δ≥$0.50 WARN fires when a foreign writer recorded a realized value that
// disagrees with the stored-price recompute (the class the P0 pass caught).
func (s *Store) StampPnlCorrectedOnClose(id int64, realized, recomputed float64) {
	if id <= 0 {
		return
	}
	delta := realized - recomputed
	note := fmt.Sprintf("T7 close-path: pnl_corrected = recompute (Δ%+.2f vs realized)", delta)
	if delta >= 0.5 || delta <= -0.5 {
		logger.Warnf("⚖️ pnl class-killer on close: row %d realized %.2f vs recompute %.2f (Δ%+.2f) — additive correction stamped.", id, realized, recomputed, delta)
	}
	if err := s.gdb.Table("trader_positions").
		Where("id = ? AND pnl_corrected IS NULL", id).
		Updates(map[string]any{"pnl_corrected": recomputed, "pnl_correction_note": note}).Error; err != nil {
		logger.Warnf("pnl close-stamp: row %d failed: %v", id, err)
	}
}
