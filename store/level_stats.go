package store

import (
	"time"

	"gorm.io/gorm"
)

// B4 — LEVEL_STATS (forward-validation table, Pack B owner override 2026-08-26).
// One row per seated level per session-day: the nightly job evaluates each
// level the day's plans seated against the day's 1m bars (kernel.
// EvaluateLevelOutcome) and records TOUCHED / REACTED(≥8pt in 3 bars) /
// BROKE-CLEAN / CHOPPED. The 2-week verdict on the volume family's weights
// reads from this table.

// LevelStatsDB is one evaluated level row. PK = (trader_id, session_day, price,
// label) — one row per level per day.
type LevelStatsDB struct {
	TraderID   string    `gorm:"column:trader_id;primaryKey"`
	SessionDay string    `gorm:"column:session_day;primaryKey"` // CME session-day key YYYY-MM-DD
	Price      float64   `gorm:"column:price;primaryKey"`
	Label      string    `gorm:"column:label;primaryKey"`
	Kind       string    `gorm:"column:kind;not null;default:''"`
	Grade      string    `gorm:"column:grade;not null;default:''"`
	Role       string    `gorm:"column:role;not null;default:''"`
	Family     string    `gorm:"column:family;not null;default:''"`
	Touched    bool      `gorm:"column:touched;not null;default:false"`
	Reacted    bool      `gorm:"column:reacted;not null;default:false"`
	BrokeClean bool      `gorm:"column:broke_clean;not null;default:false"`
	Chopped    bool      `gorm:"column:chopped;not null;default:false"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName implements the gorm Tabler interface.
func (LevelStatsDB) TableName() string { return "level_stats" }

// LevelStatsStore wraps the gorm handle for level_stats.
type LevelStatsStore struct {
	db *gorm.DB
}

// NewLevelStatsStore builds the store on the root gorm handle.
func NewLevelStatsStore(db *gorm.DB) *LevelStatsStore { return &LevelStatsStore{db: db} }

// Migrate creates the table (additive, idempotent).
func (s *LevelStatsStore) Migrate() error {
	return s.db.AutoMigrate(&LevelStatsDB{})
}

// UpsertStats writes evaluated rows with UPSERT semantics (idempotent re-runs).
func (s *LevelStatsStore) UpsertStats(rows []LevelStatsDB) error {
	if len(rows) == 0 {
		return nil
	}
	return s.db.Save(&rows).Error
}

// PruneOlderThan removes rows older than the cutoff (retention mirror of bars).
func (s *LevelStatsStore) PruneOlderThan(cutoffMs int64) (int64, error) {
	if cutoffMs <= 0 {
		return 0, nil
	}
	res := s.db.Where("created_at < ?", time.UnixMilli(cutoffMs).UTC()).Delete(&LevelStatsDB{})
	return res.RowsAffected, res.Error
}

// AggregateByGrade returns per-grade touched/reacted/broke/chopped COUNTS for
// the verdict ("grades predictive: YES/NO/PARTIAL").
type GradeAgg struct {
	Grade      string `gorm:"column:grade"`
	Rows       int64  `gorm:"column:rows"`
	Touched    int64  `gorm:"column:touched"`
	Reacted    int64  `gorm:"column:reacted"`
	BrokeClean int64  `gorm:"column:broke_clean"`
	Chopped    int64  `gorm:"column:chopped"`
}

func (s *LevelStatsStore) AggregateByGrade() ([]GradeAgg, error) {
	var out []GradeAgg
	err := s.db.Model(&LevelStatsDB{}).
		Select("grade, COUNT(*) AS rows, SUM(touched) AS touched, SUM(reacted) AS reacted, SUM(broke_clean) AS broke_clean, SUM(chopped) AS chopped").
		Group("grade").Order("grade").Scan(&out).Error
	return out, err
}

// AggregateByFamily mirrors the grade aggregate across confluence families.
type FamilyAgg struct {
	Family     string `gorm:"column:family"`
	Rows       int64  `gorm:"column:rows"`
	Touched    int64  `gorm:"column:touched"`
	Reacted    int64  `gorm:"column:reacted"`
	BrokeClean int64  `gorm:"column:broke_clean"`
}

func (s *LevelStatsStore) AggregateByFamily() ([]FamilyAgg, error) {
	var out []FamilyAgg
	err := s.db.Model(&LevelStatsDB{}).
		Select("family, COUNT(*) AS rows, SUM(touched) AS touched, SUM(reacted) AS reacted, SUM(broke_clean) AS broke_clean").
		Group("family").Order("family").Scan(&out).Error
	return out, err
}

// Count returns the current row count (boot line).
func (s *LevelStatsStore) Count() (int64, error) {
	var n int64
	err := s.db.Model(&LevelStatsDB{}).Count(&n).Error
	return n, err
}
