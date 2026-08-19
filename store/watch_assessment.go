package store

import (
	"gorm.io/gorm"
)

// Phase 3.6 — WATCHER SCORING TABLE (final-bundle 2026-08-19).
//
// Data now, judgment later: every in-position watch assessment is stored with
// the market state at read time; when the position closes, the rows are
// backfilled with what happened AFTER each read (MFE/MAE-after, final outcome).
// This is the evidence base for any FUTURE decision about giving the watcher
// authority — no analysis UI is built on it yet, deliberately.
//
// Additive migration: gorm AutoMigrate creates the table; nothing reads it on
// the decision path.

type WatchAssessment struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID    string `gorm:"column:trader_id;not null;index:idx_watch_trader_pos" json:"trader_id"`
	PositionID  int64  `gorm:"column:position_id;not null;index:idx_watch_trader_pos" json:"position_id"`
	CycleNumber int    `gorm:"column:cycle_number;not null" json:"cycle_number"`
	Timestamp   int64  `gorm:"column:timestamp;not null" json:"timestamp"` // unix ms

	// The read itself.
	Status            string `gorm:"column:status;default:''" json:"status"`                       // raw model status (post-parse)
	AcceptedStatus    string `gorm:"column:accepted_status;default:''" json:"accepted_status"`     // after hysteresis rails R1/R2
	Confidence        int    `gorm:"column:confidence;default:0" json:"confidence"`
	InvalidationCited string `gorm:"column:invalidation_cited;default:''" json:"invalidation_cited"`
	Note              string `gorm:"column:note;default:''" json:"note"`
	PriceAtRead       float64 `gorm:"column:price_at_read;default:0" json:"price_at_read"`
	Warned            bool    `gorm:"column:warned;default:false" json:"warned"` // this read fired the R3 WARN

	// Backfilled at position close (recordClosedTradeAnalytics).
	MFEAfterPts  float64 `gorm:"column:mfe_after_pts;default:0" json:"mfe_after_pts"`
	MAEAfterPts  float64 `gorm:"column:mae_after_pts;default:0" json:"mae_after_pts"`
	FinalOutcome string  `gorm:"column:final_outcome;default:''" json:"final_outcome"` // e.g. "win:+123.5" | "loss:-88.0"
}

func (WatchAssessment) TableName() string { return "watch_assessments" }

type WatchAssessmentStore struct{ db *gorm.DB }

func NewWatchAssessmentStore(db *gorm.DB) *WatchAssessmentStore { return &WatchAssessmentStore{db: db} }

func (s *WatchAssessmentStore) initTables() error {
	return s.db.AutoMigrate(&WatchAssessment{})
}

func (s *WatchAssessmentStore) Create(a *WatchAssessment) error {
	return s.db.Create(a).Error
}

// BackfillClose stamps every assessment row of a closed position with the final
// outcome (idempotent: WHERE-scoped to this position's rows without one).
func (s *WatchAssessmentStore) BackfillClose(positionID int64, finalOutcome string) error {
	return s.db.Model(&WatchAssessment{}).
		Where("position_id = ? AND final_outcome = ''", positionID).
		Update("final_outcome", finalOutcome).Error
}

// SetExcursionAfter backfills one row's after-the-read excursion (best effort;
// bars may have rotated out of the cache by close time).
func (s *WatchAssessmentStore) SetExcursionAfter(id int64, mfeAfter, maeAfter float64) error {
	return s.db.Model(&WatchAssessment{}).Where("id = ?", id).
		Updates(map[string]any{"mfe_after_pts": mfeAfter, "mae_after_pts": maeAfter}).Error
}

// ByPosition returns a position's assessments, oldest first (for close backfill).
func (s *WatchAssessmentStore) ByPosition(positionID int64) ([]WatchAssessment, error) {
	var rows []WatchAssessment
	err := s.db.Where("position_id = ?", positionID).Order("id asc").Find(&rows).Error
	return rows, err
}
