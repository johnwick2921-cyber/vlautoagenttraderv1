package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// RevokedToken is one logged-out JWT, keyed by the SHA-256 fingerprint of the
// token string (the full token is NEVER stored). P2-10 (audit 0926-system):
// the logout blacklist was in-memory only (auth/auth.go:24-58), so a restart
// revived logged-out tokens for the rest of their 24h life. This table is the
// restart-persistence layer; auth.IsTokenBlacklisted consults it after the
// in-memory map.
type RevokedToken struct {
	ID        string    `gorm:"primaryKey;column:id"` // hex sha256(token)
	ExpiresAt time.Time `gorm:"column:expires_at;index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName pins the table name for RevokedToken.
func (RevokedToken) TableName() string { return "revoked_tokens" }

// RevokedTokenStore persists logout revocations.
type RevokedTokenStore struct {
	db *gorm.DB
	mu sync.Mutex
}

// NewRevokedTokenStore creates the store.
func NewRevokedTokenStore(db *gorm.DB) *RevokedTokenStore {
	return &RevokedTokenStore{db: db}
}

func (s *RevokedTokenStore) initTables() error {
	return s.db.AutoMigrate(&RevokedToken{})
}

// Save upserts a revocation record (idempotent).
func (s *RevokedTokenStore) Save(tokenID string, expiresAt time.Time) error {
	if tokenID == "" {
		return errors.New("store: empty revocation id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row := RevokedToken{ID: tokenID, ExpiresAt: expiresAt.UTC(), CreatedAt: time.Now().UTC()}
	return s.db.Save(&row).Error
}

// IsRevoked reports whether the fingerprint is revoked and still unexpired.
// An expired row is treated as not revoked and deleted opportunistically.
func (s *RevokedTokenStore) IsRevoked(tokenID string) (bool, error) {
	if tokenID == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var row RevokedToken
	if err := s.db.First(&row, "id = ?", tokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if !time.Now().After(row.ExpiresAt) {
		return true, nil
	}
	// Expired: remove it and report not revoked. The delete failure is
	// returned — never swallowed (B1: no silent failure).
	if err := s.db.Delete(&row).Error; err != nil {
		return false, fmt.Errorf("store: delete expired revocation %q: %w", tokenID, err)
	}
	return false, nil
}

// PruneExpired removes every revocation whose expiry has passed (audit 0926
// system report: "store revoked token ids with expiry in the DB; prune
// expired"). Called at boot and opportunistically on Save.
func (s *RevokedTokenStore) PruneExpired(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Where("expires_at < ?", now.UTC()).Delete(&RevokedToken{}).Error
}

// Count is for tests and telemetry only.
func (s *RevokedTokenStore) Count() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	if err := s.db.Model(&RevokedToken{}).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
