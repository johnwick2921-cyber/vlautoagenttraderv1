package store

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// TelegramConfig stores the Telegram bot binding (single row, always ID=1)
type TelegramConfig struct {
	ID        uint      `gorm:"primaryKey"`
	BotToken  string    `gorm:"column:bot_token"`
	ChatID    int64     `gorm:"column:chat_id"`
	Username  string    `gorm:"column:username"` // @username for display
	BoundAt   time.Time `gorm:"column:bound_at"`
	ModelID   string    `gorm:"column:model_id;default:''"` // AI model used for Telegram replies
	Language  string    `gorm:"column:language;default:''"` // "zh" or "en"; empty = not chosen yet
	// P2-12 (audit 0926-system): the first-/start bind needs a confirmation
	// gate — a one-time code issued in the owner-authenticated app that the
	// chat must send back. Empty code = none issued.
	PendingBindCode   string    `gorm:"column:pending_bind_code;default:''"`
	BindCodeExpiresAt time.Time `gorm:"column:bind_code_expires_at"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// String returns a safe string representation of TelegramConfig with the token masked.
func (tc TelegramConfig) String() string {
	token := "***"
	if tc.BotToken == "" {
		token = "<not set>"
	}
	return fmt.Sprintf("TelegramConfig{ID:%d, ChatID:%d, Username:%q, BotToken:%s, BoundAt:%v}",
		tc.ID, tc.ChatID, tc.Username, token, tc.BoundAt)
}

// TelegramConfigStore defines the interface for Telegram bot binding operations
type TelegramConfigStore interface {
	Get() (*TelegramConfig, error)                // Get current config (may not exist)
	SaveToken(botToken string) error              // Save bot token only (Web UI sets this)
	Save(botToken, modelID string) error          // Save bot token + selected AI model
	BindUser(chatID int64, username string) error // Called on first /start
	IsBound() (bool, error)                       // Check if any user is bound
	GetBoundChatID() (int64, error)               // Get bound chat ID (0 if not bound)
	Unbind() error                                // Remove binding
	SetLanguage(lang string) error                // Set UI language ("en" or "zh")
	GetLanguage() string                          // Get UI language; returns "en" if not set
	IssueBindCode(code string, expiresAt time.Time) error // P2-12: store a one-time bind code
	ConsumeBindCode(code string) (bool, error)    // P2-12: true if code matches and is unexpired; clears it
}

type telegramConfigStore struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewTelegramConfigStore creates a new TelegramConfigStore
func NewTelegramConfigStore(db *gorm.DB) TelegramConfigStore {
	return &telegramConfigStore{db: db}
}

func (s *telegramConfigStore) initTables() error {
	return s.db.AutoMigrate(&TelegramConfig{})
}

func (s *telegramConfigStore) Get() (*TelegramConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var cfg TelegramConfig
	if err := s.db.First(&cfg, 1).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *telegramConfigStore) SaveToken(botToken string) error {
	return s.Save(botToken, "")
}

func (s *telegramConfigStore) Save(botToken, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var cfg TelegramConfig
	result := s.db.First(&cfg, 1)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	cfg.ID = 1
	cfg.BotToken = botToken
	cfg.ModelID = modelID
	return s.db.Save(&cfg).Error
}

func (s *telegramConfigStore) BindUser(chatID int64, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var cfg TelegramConfig
	result := s.db.First(&cfg, 1)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	cfg.ID = 1
	cfg.ChatID = chatID
	cfg.Username = username
	cfg.BoundAt = time.Now()
	return s.db.Save(&cfg).Error
}

func (s *telegramConfigStore) IsBound() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var cfg TelegramConfig
	if err := s.db.First(&cfg, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return cfg.ChatID != 0, nil
}

func (s *telegramConfigStore) GetBoundChatID() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var cfg TelegramConfig
	if err := s.db.First(&cfg, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return cfg.ChatID, nil
}

func (s *telegramConfigStore) Unbind() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&TelegramConfig{}).Where("id = 1").Updates(map[string]interface{}{
		"chat_id":  0,
		"username": "",
	}).Error
}

// IssueBindCode (P2-12) stores a one-time bind code with its expiry on the
// single config row.
func (s *telegramConfigStore) IssueBindCode(code string, expiresAt time.Time) error {
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("store: empty bind code")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var cfg TelegramConfig
	result := s.db.First(&cfg, 1)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	cfg.ID = 1
	cfg.PendingBindCode = code
	cfg.BindCodeExpiresAt = expiresAt.UTC()
	return s.db.Save(&cfg).Error
}

// ConsumeBindCode (P2-12) verifies a submitted code against the stored one in
// constant time. On a match the code is cleared (single use) and true is
// returned. A mismatch or an expired code returns false and clears an expired
// code.
func (s *telegramConfigStore) ConsumeBindCode(code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var cfg TelegramConfig
	result := s.db.First(&cfg, 1)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	if cfg.PendingBindCode == "" {
		return false, nil
	}
	stored := cfg.PendingBindCode
	clear := func() error {
		cfg.PendingBindCode = ""
		cfg.BindCodeExpiresAt = time.Time{}
		return s.db.Save(&cfg).Error
	}
	if !cfg.BindCodeExpiresAt.IsZero() && time.Now().After(cfg.BindCodeExpiresAt) {
		if err := clear(); err != nil {
			return false, fmt.Errorf("store: clear expired bind code: %w", err)
		}
		return false, nil
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(strings.TrimSpace(code))) == 1 {
		if err := clear(); err != nil {
			// The code MATCHED; the clear failed, so it stays valid and a
			// retry can still consume it. The error is surfaced, never
			// swallowed (B1).
			return true, fmt.Errorf("store: bind code matched but clearing failed: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func (s *telegramConfigStore) SetLanguage(lang string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var cfg TelegramConfig
	result := s.db.First(&cfg, 1)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	cfg.ID = 1
	cfg.Language = lang
	return s.db.Save(&cfg).Error
}

func (s *telegramConfigStore) GetLanguage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var cfg TelegramConfig
	if err := s.db.First(&cfg, 1).Error; err != nil {
		return "en" // default: English
	}
	if cfg.Language == "" {
		return "en"
	}
	return cfg.Language
}
