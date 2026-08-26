package store

import (
	"encoding/json"
	"time"

	"nofx/logger"
)

// 6.7 (final-bundle 2026-08-19) — entry_confidence backfill.
//
// The column only started populating with the P5 wiring; 67 of the last 71
// closed positions carried 0, starving the Phase-3.6 watcher scoring table of
// history. This one-time, boot-run migration recovers the value from each
// position's entry decision record; unrecoverable rows are stamped -1 ("looked,
// not found") so 0 stays reserved for "never populated".
//
// Guardrails: runs ONCE (system_config flag), WHERE-scoped to CLOSED rows with
// entry_confidence=0, purely additive to a column nothing gates on. Rollback:
// `UPDATE trader_positions SET entry_confidence=0 WHERE entry_confidence=-1`
// clears the not-found markers; recovered values are indistinguishable from
// organically-recorded ones — full rollback = the C1 backup (documented, not
// executed).

const backfillEntryConfidenceFlag = "backfill_entry_confidence_done"

// BackfillEntryConfidence recovers entry_confidence for historical closed
// positions from their entry decision records. Idempotent via the flag.
func (s *Store) BackfillEntryConfidence() {
	if v, err := s.GetSystemConfig(backfillEntryConfidenceFlag); err == nil && v == "1" {
		return
	}
	type posRow struct {
		ID        int64
		TraderID  string
		Symbol    string
		Side      string
		EntryTime int64
	}
	var rows []posRow
	if err := s.gdb.Table("trader_positions").
		Select("id, trader_id, symbol, side, entry_time").
		Where("status = 'CLOSED' AND entry_confidence = 0").
		Order("id asc").Limit(2000).Scan(&rows).Error; err != nil {
		logger.Warnf("6.7 backfill: position scan failed: %v", err)
		return
	}
	recovered, notFound := 0, 0
	for _, p := range rows {
		conf := s.entryConfidenceFromDecisions(p.TraderID, p.Symbol, p.Side, p.EntryTime)
		val := conf
		if conf <= 0 {
			val = -1 // looked, not recoverable
			notFound++
		} else {
			recovered++
		}
		if err := s.gdb.Table("trader_positions").
			Where("id = ? AND entry_confidence = 0", p.ID).
			Update("entry_confidence", val).Error; err != nil {
			logger.Warnf("6.7 backfill: row %d update failed: %v", p.ID, err)
		}
	}
	_ = s.SetSystemConfig(backfillEntryConfidenceFlag, "1")
	logger.Warnf("🧮 6.7 entry_confidence backfill complete: %d recovered from decision records, %d marked -1 (unrecoverable), of %d candidates. Rollback for markers: UPDATE trader_positions SET entry_confidence=0 WHERE entry_confidence=-1.",
		recovered, notFound, len(rows))
}

// entryConfidenceFromDecisions finds the entry decision that produced a
// position (same trader, [-10min, +2min] around entry, matching action+symbol)
// and returns its confidence, 0 when unrecoverable.
func (s *Store) entryConfidenceFromDecisions(traderID, symbol, side string, entryMs int64) int {
	wantAction := "open_long"
	if side == "SHORT" {
		wantAction = "open_short"
	}
	lo := time.UnixMilli(entryMs - 10*60_000).UTC()
	hi := time.UnixMilli(entryMs + 2*60_000).UTC()
	type recRow struct {
		Decisions string
		Timestamp time.Time
	}
	var recs []recRow
	if err := s.gdb.Table("decision_records").
		Select("decisions, timestamp").
		Where("trader_id = ? AND timestamp BETWEEN ? AND ?", traderID, lo, hi).
		Order("timestamp desc").Limit(10).Scan(&recs).Error; err != nil {
		return 0
	}
	for _, r := range recs {
		var ds []struct {
			Action     string  `json:"action"`
			Symbol     string  `json:"symbol"`
			Confidence float64 `json:"confidence"`
		}
		if json.Unmarshal([]byte(r.Decisions), &ds) != nil {
			continue
		}
		for _, d := range ds {
			if d.Action == wantAction && d.Symbol == symbol && d.Confidence > 0 {
				return int(d.Confidence)
			}
		}
	}
	return 0
}
