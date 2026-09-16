package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P1.3 — DURABLE SESSION-PROFILE STORE (RECON #2, MANDATORY).
//
// The SVP engine is stateless and the 1m bar cache holds only ~1 prior session
// (wiped on restart). To give naked-POC and multi-day levels real cross-session
// memory, each COMPLETED session's volume profile (POC/VAH/VAL + range + the
// full profile JSON) is persisted here at the 17:00-CT roll — append-only, one
// row per (symbol, session_date). Warms FORWARD from install (no backfill); the
// writer is idempotent (create-if-absent) so a restart replays with no loss and
// no dupes.

// SessionProfileStore persists frozen session volume profiles.
type SessionProfileStore struct {
	db *gorm.DB
}

// SessionProfileDB is one completed session's profile. PK (symbol, session_date).
type SessionProfileDB struct {
	Symbol      string  `gorm:"column:symbol;primaryKey"`
	SessionDate string  `gorm:"column:session_date;primaryKey"` // YYYY-MM-DD CME session-day
	POC         float64 `gorm:"column:poc;not null;default:0"`
	VAH         float64 `gorm:"column:vah;not null;default:0"`
	VAL         float64 `gorm:"column:val;not null;default:0"`
	SessHigh    float64 `gorm:"column:sess_high;not null;default:0"`
	SessLow     float64 `gorm:"column:sess_low;not null;default:0"`
	ProfileJSON string  `gorm:"column:profile_json;not null;default:'{}'"`
	CreatedAt   int64   `gorm:"column:created_at"` // Unix ms (set by the writer)
}

// TableName implements the gorm Tabler interface.
func (SessionProfileDB) TableName() string { return "session_profiles" }

// NewSessionProfileStore creates a SessionProfileStore.
func NewSessionProfileStore(db *gorm.DB) *SessionProfileStore {
	return &SessionProfileStore{db: db}
}

func (s *SessionProfileStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var exists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'session_profiles'`).Scan(&exists)
		if exists > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&SessionProfileDB{})
}

// SaveIfAbsent persists a session profile only if (symbol, session_date) is not
// already stored — append-only + restart-safe (idempotent, no dupes). Returns
// true when a new row was written.
func (s *SessionProfileStore) SaveIfAbsent(p *SessionProfileDB) (bool, error) {
	if p == nil || p.Symbol == "" || p.SessionDate == "" {
		return false, fmt.Errorf("symbol and session_date required")
	}
	var existing SessionProfileDB
	err := s.db.Where("symbol = ? AND session_date = ?", p.Symbol, p.SessionDate).First(&existing).Error
	if err == nil {
		return false, nil // already stored — never overwrite a frozen session
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	if err := s.db.Create(p).Error; err != nil {
		return false, err
	}
	return true, nil
}

// Exists reports whether a session profile is already stored.
func (s *SessionProfileStore) Exists(symbol, sessionDate string) (bool, error) {
	var n int64
	err := s.db.Model(&SessionProfileDB{}).
		Where("symbol = ? AND session_date = ?", symbol, sessionDate).Count(&n).Error
	return n > 0, err
}

// List returns the most recent stored sessions for a symbol (newest first).
func (s *SessionProfileStore) List(symbol string, limit int) ([]*SessionProfileDB, error) {
	if limit <= 0 {
		limit = 60
	}
	var rows []*SessionProfileDB
	err := s.db.Where("symbol = ?", symbol).
		Order("session_date DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Count returns how many sessions are stored for a symbol (the WARMING n/10
// numerator — cold-start honesty).
func (s *SessionProfileStore) Count(symbol string) (int, error) {
	var n int64
	err := s.db.Model(&SessionProfileDB{}).Where("symbol = ?", symbol).Count(&n).Error
	return int(n), err
}
