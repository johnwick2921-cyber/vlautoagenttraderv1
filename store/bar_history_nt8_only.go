package store

import (
	"fmt"
	"strings"
)

// ── THE PLANNER TAPE IS NT8-ONLY ────────────────────────────────────────────
// (CTO ruling under the owner's delegation, 2026-09-16, follow-up to dispatch 101)
//
// LastNBarsOn is the SHARED contract-scoped reader: its filter is mixed +
// off-scale only, so historical_import rows (the 09-07..09-14 file) reach
// every caller. The CHART keeps them, labelled (owner: "i want fuull data").
// The PLANNER does not: measured 2026-09-16 on data/data.db, MNQ 12-26 1m held
// 2,873 live + 25 replay + 426 historical_import rows — all inside the newest
// 12,000 the planner's tape asks for — so 426 imported bars fed the regime
// baseline and the levels. Excluding them changes the planner's input (class
// 82), so the boot line names every moved value; the exclusion itself lives
// here, in a reader of its own, and LastNBarsOn is byte-identical.

// LastNBarsFromNT8On is LastNBarsOn minus historical_import: the newest n
// rows NT8 itself produced (live, or replay verified on the live scale) on one
// contract, ASCENDING. It is the ONLY reader a planner door may use.
func (s *BarHistoryStore) LastNBarsFromNT8On(symbol, tf, contract string, n int) ([]BarHistoryDB, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store required")
	}
	if n <= 0 {
		return nil, nil
	}
	var desc []BarHistoryDB
	q := s.db.Where("symbol = ? AND tf = ? AND COALESCE(source, '') NOT IN (?, ?, ?)",
		symbol, tf, BarSourceMixed, BarSourceOffScale, BarSourceHistoricalImport)
	if c := strings.TrimSpace(contract); c != "" {
		q = q.Where("contract = ?", c)
	}
	if err := q.Order("open_time_ms DESC").Limit(n).Find(&desc).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(desc)-1; i < j; i, j = i+1, j-1 {
		desc[i], desc[j] = desc[j], desc[i]
	}
	return desc, nil
}

// ImportRowsOn counts the historical_import rows on one contract — the number
// the planner-tape boot line prints as "excluded", read, never inferred.
func (s *BarHistoryStore) ImportRowsOn(symbol, tf, contract string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("store required")
	}
	var n int64
	q := s.db.Model(&BarHistoryDB{}).Where("symbol = ? AND tf = ? AND COALESCE(source, '') = ?", symbol, tf, BarSourceHistoricalImport)
	if c := strings.TrimSpace(contract); c != "" {
		q = q.Where("contract = ?", c)
	}
	return n, q.Count(&n).Error
}

// BarsBetweenFromNT8On is BarsBetweenOn minus historical_import — the weekly
// reader's door (its weeks from epoch, and its Own1m window). ASCENDING.
func (s *BarHistoryStore) BarsBetweenFromNT8On(symbol, tf, contract string, fromMs, toMs int64) ([]BarHistoryDB, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store required")
	}
	var out []BarHistoryDB
	q := s.db.Where("symbol = ? AND tf = ? AND open_time_ms >= ? AND open_time_ms < ? AND COALESCE(source, '') NOT IN (?, ?, ?)",
		symbol, tf, fromMs, toMs, BarSourceMixed, BarSourceOffScale, BarSourceHistoricalImport)
	if c := strings.TrimSpace(contract); c != "" {
		q = q.Where("contract = ?", c)
	}
	err := q.Order("open_time_ms").Find(&out).Error
	return out, err
}
