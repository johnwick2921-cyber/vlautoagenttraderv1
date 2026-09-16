package store

import (
	"fmt"

	"nofx/logger"
)

// E4 (wave 1A, 2026-09-02) — trader_positions.mae / mfe: DEFAULT 0 made a
// computed zero and a value nobody ever computed the same bit pattern (audit
// D15). The Go type is now *float64 so the two are distinguishable going
// forward, and this migration retires the historical ambiguity.
//
// It nulls ONLY the rows where BOTH columns are 0. A real trade cannot have
// both a zero adverse and a zero favourable excursion — price moves — so that
// pair is the never-computed signature. Rows with one genuine zero beside a
// real number are LEFT ALONE: measured on 2026-09-02, 517 closed rows carried
// the pair and 9 carried a single genuine zero, and destroying those 9 to
// satisfy a blanket rule would be the same disease in the other direction.
//
// Idempotent (a second run matches nothing) and WHERE-scoped to CLOSED rows.
func (s *PositionStore) MigrateExcursionZerosToNull() (int64, error) {
	res := s.db.Exec(`
		UPDATE trader_positions
		   SET mae = NULL, mfe = NULL
		 WHERE status = 'CLOSED' AND mae = 0 AND mfe = 0`)
	if res.Error != nil {
		return 0, fmt.Errorf("null excursion zeros: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		logger.Infof("📐 excursion migration: %d closed positions had mae=0 AND mfe=0 (never computed) → NULL; rows with one genuine zero were left as measured", res.RowsAffected)
	}
	return res.RowsAffected, nil
}
