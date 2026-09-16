// F12 — the received order_snapshot frames, persisted.
//
// One row per RECEIVED snapshot, orders held as JSON. The table is a forensic
// record, not the gate's source: leg 4 reads the in-memory cache, because a
// gate that needs a database round-trip to answer "is the book flat" has added
// a failure mode to the one check that must not have any. What the table buys
// is the ability to answer, after the fact, what the broker's book looked like
// at a moment nobody was watching — which is exactly what could not be answered
// about position 588 on 2026-09-02.
package store

import (
	"time"

	"gorm.io/gorm"
)

// NT8OrderSnapshot is one received frame.
type NT8OrderSnapshot struct {
	ID      int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Account string `gorm:"column:account;index:idx_nt8_snap_key" json:"account"`
	Symbol  string `gorm:"column:symbol;index:idx_nt8_snap_key" json:"symbol"`
	// BuildID is the AddOn build that produced the frame — the received-side
	// proof of which DLL is running (class 6).
	BuildID string `gorm:"column:build_id" json:"build_id"`
	Reason  string `gorm:"column:reason" json:"reason"`
	// OrdersJSON is the verbatim order list. Stored as text rather than
	// normalised into rows: the value here is the untouched record of what the
	// broker said, and a schema of our own would be one more thing to drift.
	OrdersJSON string `gorm:"column:orders_json" json:"orders_json"`
	OrderCount int    `gorm:"column:order_count" json:"order_count"`
	// WorkingCount is the non-terminal subset, precomputed so a forensic query
	// does not have to re-implement the definition of "working".
	WorkingCount int   `gorm:"column:working_count" json:"working_count"`
	EmittedMs    int64 `gorm:"column:emitted_at_ms" json:"emitted_at_ms"`
	ReceivedMs   int64 `gorm:"column:received_at_ms;index" json:"received_at_ms"`
}

func (NT8OrderSnapshot) TableName() string { return "nt8_order_snapshots" }

// NT8OrderSnapshotStore wraps the gorm handle.
type NT8OrderSnapshotStore struct{ db *gorm.DB }

func NewNT8OrderSnapshotStore(db *gorm.DB) *NT8OrderSnapshotStore {
	return &NT8OrderSnapshotStore{db: db}
}

func (s *NT8OrderSnapshotStore) Migrate() error {
	return s.db.AutoMigrate(&NT8OrderSnapshot{})
}

// Insert records one frame. Errors are returned, never swallowed — but the
// caller logs and continues: losing the forensic row must never cost the link
// its snapshot (A10).
func (s *NT8OrderSnapshotStore) Insert(row *NT8OrderSnapshot) error {
	return s.db.Create(row).Error
}

// Latest returns the most recent stored snapshot for a key, for forensics.
func (s *NT8OrderSnapshotStore) Latest(account, symbol string) (*NT8OrderSnapshot, error) {
	var r NT8OrderSnapshot
	err := s.db.Where("account = ? AND symbol = ?", account, symbol).
		Order("received_at_ms DESC").First(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// PruneBefore drops rows older than cutoff. A 30-second snapshot is ~2,880 rows
// a day; without a prune this table is the biggest writer in the store.
func (s *NT8OrderSnapshotStore) PruneBefore(cutoff time.Time) (int64, error) {
	res := s.db.Where("received_at_ms < ?", cutoff.UnixMilli()).Delete(&NT8OrderSnapshot{})
	return res.RowsAffected, res.Error
}
