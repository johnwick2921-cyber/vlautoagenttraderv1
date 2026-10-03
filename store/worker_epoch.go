package store

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

// WorkerEpoch is the single-row worker_token_epoch table (P-E E3): one
// integer, bumped by the operator's `vl-updater-bootstrap revoke-worker
// --all`. A cutover-worker JWT carries the epoch it was minted under (WTE);
// the API refuses any worker token whose WTE is below the current epoch, so
// one bump invalidates every worker credential ever minted before it.
type WorkerEpoch struct {
	ID    uint  `gorm:"primaryKey;column:id"`
	Epoch int64 `gorm:"column:epoch"`
}

// TableName pins the table name for WorkerEpoch.
func (WorkerEpoch) TableName() string { return "worker_epoch" }

// WorkerEpochStore reads and bumps the worker_token_epoch.
type WorkerEpochStore struct {
	db *gorm.DB
	mu sync.Mutex
}

// NewWorkerEpochStore creates the store.
func NewWorkerEpochStore(db *gorm.DB) *WorkerEpochStore {
	return &WorkerEpochStore{db: db}
}

func (s *WorkerEpochStore) initTables() error {
	return s.db.AutoMigrate(&WorkerEpoch{})
}

// Current returns the current epoch; no row yet reads as 0 (the epoch every
// first mint starts from).
func (s *WorkerEpochStore) Current() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row WorkerEpoch
	if err := s.db.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.Epoch, nil
}

// Bump increments the epoch by one (idempotent single row) and returns the
// NEW value, so the caller can mint a fresh worker token under it in one
// attend.
func (s *WorkerEpochStore) Bump() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row WorkerEpoch
	if err := s.db.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = WorkerEpoch{ID: 1, Epoch: 1}
			return row.Epoch, s.db.Create(&row).Error
		}
		return 0, err
	}
	row.Epoch++
	if err := s.db.Model(&row).Update("epoch", row.Epoch).Error; err != nil {
		return 0, err
	}
	return row.Epoch, nil
}

// WorkEpochRowCount is test/telemetry only.
func (s *WorkerEpochStore) WorkEpochRowCount() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	if err := s.db.Model(&WorkerEpoch{}).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
