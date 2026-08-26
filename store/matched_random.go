package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P5.6 — durable matched-random verdicts + a weekly-summary table. One verdict
// row per graded level touch; the weekly summary is written on a fixed cadence
// (SaveWeeklyIfAbsent = first-writer-wins, so N traders don't re-peek). The
// honesty gate itself lives in kernel.EvaluateMatchedRandom.

// MatchedRandomStore persists matched-random reaction verdicts.
type MatchedRandomStore struct {
	db *gorm.DB
}

// MatchedRandomDB is one level-touch reaction verdict.
type MatchedRandomDB struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	LevelType  string `gorm:"column:level_type;not null;default:'';index:idx_mr_type" json:"level_type"`
	Reacted    bool   `gorm:"column:reacted;not null;default:false" json:"reacted"`
	SessionDay string `gorm:"column:session_day;not null;default:''" json:"session_day"`
	CreatedAt  int64  `gorm:"column:created_at" json:"created_at"`
}

// TableName implements the gorm Tabler interface.
func (MatchedRandomDB) TableName() string { return "matched_random_verdicts" }

// MatchedRandomWeeklyDB is one weekly-evaluation snapshot (idempotent per week).
type MatchedRandomWeeklyDB struct {
	ISOWeek     string `gorm:"column:iso_week;primaryKey" json:"iso_week"` // e.g. "2026-W33"
	ComputedAt  int64  `gorm:"column:computed_at" json:"computed_at"`
	SummaryJSON string `gorm:"column:summary_json;not null;default:'[]'" json:"summary_json"`
}

func (MatchedRandomWeeklyDB) TableName() string { return "matched_random_weekly" }

// TypeTally is the raw per-type count (store-local; kernel converts to TypeCounts).
type TypeTally struct {
	Touches   int
	Reactions int
}

// NewMatchedRandomStore creates a MatchedRandomStore.
func NewMatchedRandomStore(db *gorm.DB) *MatchedRandomStore { return &MatchedRandomStore{db: db} }

func (s *MatchedRandomStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var n int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'matched_random_verdicts'`).Scan(&n)
		if n > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&MatchedRandomDB{}, &MatchedRandomWeeklyDB{})
}

// RecordTouch appends one reaction verdict.
func (s *MatchedRandomStore) RecordTouch(levelType string, reacted bool, sessionDay string, createdAt int64) error {
	if levelType == "" {
		return fmt.Errorf("level_type required")
	}
	return s.db.Omit("ID").Create(&MatchedRandomDB{
		LevelType: levelType, Reacted: reacted, SessionDay: sessionDay, CreatedAt: createdAt,
	}).Error
}

// CountsByType aggregates all verdicts into per-type tallies.
func (s *MatchedRandomStore) CountsByType() (map[string]TypeTally, error) {
	var rows []struct {
		LevelType string
		Touches   int
		Reactions int
	}
	err := s.db.Model(&MatchedRandomDB{}).
		Select("level_type, COUNT(*) as touches, SUM(CASE WHEN reacted THEN 1 ELSE 0 END) as reactions").
		Group("level_type").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]TypeTally, len(rows))
	for _, r := range rows {
		out[r.LevelType] = TypeTally{Touches: r.Touches, Reactions: r.Reactions}
	}
	return out, nil
}

// SaveWeeklyIfAbsent writes the week's summary only if that ISO week has none yet
// (first-writer-wins → no optional-stopping re-peek across traders). Returns true
// when THIS call wrote the row.
func (s *MatchedRandomStore) SaveWeeklyIfAbsent(isoWeek, summaryJSON string, computedAt int64) (bool, error) {
	var n int64
	if err := s.db.Model(&MatchedRandomWeeklyDB{}).Where("iso_week = ?", isoWeek).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	err := s.db.Create(&MatchedRandomWeeklyDB{ISOWeek: isoWeek, ComputedAt: computedAt, SummaryJSON: summaryJSON}).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

// ResetWindow clears all matched-random verdicts + weekly snapshots (§128 — a
// planner model change starts a fresh window, no pooling across models). Both
// tables are day-plan learning data (not owner-sacred), so a full reset is the
// intended semantics.
func (s *MatchedRandomStore) ResetWindow() error {
	if err := s.db.Where("1 = 1").Delete(&MatchedRandomDB{}).Error; err != nil {
		return err
	}
	return s.db.Where("1 = 1").Delete(&MatchedRandomWeeklyDB{}).Error
}

// HasWeekly reports whether a snapshot already exists for an ISO week (a cheap
// pre-check so the weekly job skips recompute after the first write).
func (s *MatchedRandomStore) HasWeekly(isoWeek string) bool {
	var n int64
	s.db.Model(&MatchedRandomWeeklyDB{}).Where("iso_week = ?", isoWeek).Count(&n)
	return n > 0
}

// LatestWeekly returns the most recent weekly snapshot (nil when none).
func (s *MatchedRandomStore) LatestWeekly() (*MatchedRandomWeeklyDB, error) {
	var row MatchedRandomWeeklyDB
	err := s.db.Order("iso_week DESC").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
