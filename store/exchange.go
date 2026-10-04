package store

import (
	"fmt"
	"strings"
	"time"
	"vl/crypto"
	"vl/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExchangeStore exchange storage
type ExchangeStore struct {
	db *gorm.DB
}

// Exchange exchange configuration
type Exchange struct {
	ID           string                 `gorm:"primaryKey" json:"id"`
	ExchangeType string                 `gorm:"column:exchange_type;not null;default:''" json:"exchange_type"`
	AccountName  string                 `gorm:"column:account_name;not null;default:''" json:"account_name"`
	UserID       string                 `gorm:"column:user_id;not null;default:default;index" json:"user_id"`
	Name         string                 `gorm:"not null" json:"name"`
	Type         string                 `gorm:"not null" json:"type"` // "cex" or "dex"
	Enabled      bool                   `gorm:"default:false" json:"enabled"`
	APIKey       crypto.EncryptedString `gorm:"column:api_key;default:''" json:"apiKey"`
	SecretKey    crypto.EncryptedString `gorm:"column:secret_key;default:''" json:"secretKey"`
	Passphrase   crypto.EncryptedString `gorm:"column:passphrase;default:''" json:"passphrase"`
	Testnet      bool                   `gorm:"default:false" json:"testnet"`
	// NinjaTrader CSV bridge configuration (no API key required)
	NTDataDir            string    `gorm:"column:nt_data_dir;default:''" json:"ntDataDir"`
	NTInstrumentName     string    `gorm:"column:nt_instrument_name;default:''" json:"ntInstrumentName"`
	NTDefaultContractQty int       `gorm:"column:nt_default_contract_qty;default:0" json:"ntDefaultContractQty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (Exchange) TableName() string { return "exchanges" }

// NewExchangeStore creates a new ExchangeStore
func NewExchangeStore(db *gorm.DB) *ExchangeStore {
	return &ExchangeStore{db: db}
}

func (s *ExchangeStore) initTables() error {
	// For PostgreSQL with existing table, skip AutoMigrate
	if s.db.Dialector.Name() == "postgres" {
		var tableExists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'exchanges'`).Scan(&tableExists)
		if tableExists > 0 {
			// Still run data migrations
			s.migrateToMultiAccount()
			s.db.Model(&Exchange{}).Where("account_name = '' OR account_name IS NULL").Update("account_name", "Default")
			if err := s.cleanupIncompleteExchangeConfigs(); err != nil {
				logger.Warnf("Exchange cleanup migration warning: %v", err)
			}
			return nil
		}
	}

	if err := s.db.AutoMigrate(&Exchange{}); err != nil {
		return err
	}

	// Run migration to multi-account if needed
	if err := s.migrateToMultiAccount(); err != nil {
		logger.Warnf("Multi-account migration warning: %v", err)
	}

	// Fix empty account_name for existing records
	s.db.Model(&Exchange{}).Where("account_name = '' OR account_name IS NULL").Update("account_name", "Default")
	if err := s.cleanupIncompleteExchangeConfigs(); err != nil {
		logger.Warnf("Exchange cleanup migration warning: %v", err)
	}

	return nil
}

// missingContainsexchangeType reports whether the missing-fields list names the
// unsupported-type marker (the visibility default arm returns ["exchange_type"]
// for any type this build no longer supports). C1 P0 (PR #188 D1 port): such a
// row is a legacy crypto row and is KEPT, never deleted.
func missingContainsexchangeType(missing []string) bool {
	for _, m := range missing {
		if m == "exchange_type" {
			return true
		}
	}
	return false
}

func (s *ExchangeStore) cleanupIncompleteExchangeConfigs() error {
	var exchanges []Exchange
	if err := s.db.Find(&exchanges).Error; err != nil {
		return err
	}
	for _, exchange := range exchanges {
		missing := MissingRequiredExchangeCredentialFields(
			exchange.ExchangeType,
			string(exchange.APIKey),
			string(exchange.SecretKey),
			string(exchange.Passphrase),
			exchange.NTDataDir,
		)
		if missingContainsexchangeType(missing) {
			logger.Infof("exchange row %s type=%s unsupported in this build — kept, never deleted", exchange.ID, exchange.ExchangeType)
			continue
		}
		if len(missing) > 0 {
			if err := s.db.Delete(&Exchange{}, "id = ? AND user_id = ?", exchange.ID, exchange.UserID).Error; err != nil {
				return err
			}
			logger.Infof("🧹 Removed incomplete exchange config during migration: id=%s user=%s missing=%s", exchange.ID, exchange.UserID, strings.Join(missing, ","))
			continue
		}
		if !exchange.Enabled {
			if err := s.db.Model(&Exchange{}).Where("id = ? AND user_id = ?", exchange.ID, exchange.UserID).Update("enabled", true).Error; err != nil {
				return err
			}
			logger.Infof("🧹 Enabled complete exchange config during migration: id=%s user=%s", exchange.ID, exchange.UserID)
		}
	}
	return nil
}

// migrateToMultiAccount migrates old schema (id=exchange_type) to new schema (id=UUID)
func (s *ExchangeStore) migrateToMultiAccount() error {
	// Check if migration is needed by looking for old-style IDs (non-UUID)
	var count int64
	err := s.db.Model(&Exchange{}).
		Where("exchange_type = '' AND id IN ?", []string{"binance", "bybit", "okx", "bitget", "hyperliquid", "aster", "lighter"}).
		Count(&count).Error
	if err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	logger.Infof("🔄 Migrating %d exchange records to multi-account schema...", count)

	// Get all old records
	var records []Exchange
	err = s.db.Where("exchange_type = '' AND id IN ?", []string{"binance", "bybit", "okx", "bitget", "hyperliquid", "aster", "lighter"}).
		Find(&records).Error
	if err != nil {
		return err
	}

	// Begin transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, r := range records {
			newID := uuid.New().String()
			oldID := r.ID // This is the exchange type (e.g., "binance")

			// Update traders table to use new UUID
			if err := tx.Exec("UPDATE traders SET exchange_id = ? WHERE exchange_id = ? AND user_id = ?",
				newID, oldID, r.UserID).Error; err != nil {
				logger.Errorf("Failed to update traders for exchange %s: %v", oldID, err)
				return err
			}

			// Update the exchange record
			if err := tx.Model(&Exchange{}).
				Where("id = ? AND user_id = ?", oldID, r.UserID).
				Updates(map[string]interface{}{
					"id":            newID,
					"exchange_type": oldID,
					"account_name":  "Default",
				}).Error; err != nil {
				logger.Errorf("Failed to migrate exchange %s: %v", oldID, err)
				return err
			}

			logger.Infof("✅ Migrated exchange %s -> UUID %s for user %s", oldID, newID, r.UserID)
		}
		return nil
	})
}

func (s *ExchangeStore) initDefaultData() error {
	// No longer pre-populate exchanges - create on demand when user configures
	return nil
}

// List gets user's exchange list
func (s *ExchangeStore) List(userID string) ([]*Exchange, error) {
	var exchanges []*Exchange
	err := s.db.Where("user_id = ?", userID).Order("exchange_type, account_name").Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	return exchanges, nil
}

// GetByID gets a specific exchange by UUID
func (s *ExchangeStore) GetByID(userID, id string) (*Exchange, error) {
	var exchange Exchange
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&exchange).Error
	if err != nil {
		return nil, err
	}
	return &exchange, nil
}

// getExchangeNameAndType returns the display name and type for an exchange type
func getExchangeNameAndType(exchangeType string) (name string, typ string) {
	switch exchangeType {
	case "ninjatrader":
		return "NinjaTrader", "futures"
	default:
		return exchangeType + " Exchange", "futures"
	}
}

// Create creates a new exchange account with UUID.
// NinjaTrader fields (ntDataDir/ntInstrumentName/ntDefaultContractQty) are
// only meaningful when exchangeType=="ninjatrader"; pass "" / 0 otherwise.
func (s *ExchangeStore) Create(userID, exchangeType, accountName string, enabled bool,
	apiKey, secretKey, passphrase string, testnet bool,
	ntDataDir, ntInstrumentName string, ntDefaultContractQty int) (string, error) {

	if missing := MissingRequiredExchangeCredentialFields(
		exchangeType, apiKey, secretKey, passphrase,
		ntDataDir,
	); len(missing) > 0 {
		return "", fmt.Errorf("missing required exchange fields: %s", strings.Join(missing, ", "))
	}

	id := uuid.New().String()
	name, typ := getExchangeNameAndType(exchangeType)

	if accountName == "" {
		accountName = "Default"
	}

	logger.Debugf("🔧 ExchangeStore.Create: userID=%s, exchangeType=%s, accountName=%s, id=%s",
		userID, exchangeType, accountName, id)

	exchange := &Exchange{
		ID:                   id,
		ExchangeType:         exchangeType,
		AccountName:          accountName,
		UserID:               userID,
		Name:                 name,
		Type:                 typ,
		Enabled:              true,
		APIKey:               crypto.EncryptedString(apiKey),
		SecretKey:            crypto.EncryptedString(secretKey),
		Passphrase:           crypto.EncryptedString(passphrase),
		Testnet:              testnet,
		NTDataDir:            ntDataDir,
		NTInstrumentName:     ntInstrumentName,
		NTDefaultContractQty: ntDefaultContractQty,
	}

	if err := s.db.Create(exchange).Error; err != nil {
		return "", err
	}
	return id, nil
}

// Update updates exchange configuration by UUID.
// NinjaTrader fields (ntDataDir/ntInstrumentName/ntDefaultContractQty) are
// only meaningful when the row is type "ninjatrader"; pass "" / 0 otherwise.
func (s *ExchangeStore) Update(userID, id string, enabled bool, apiKey, secretKey, passphrase string, testnet bool,
	ntDataDir, ntInstrumentName string, ntDefaultContractQty int) error {

	logger.Debugf("🔧 ExchangeStore.Update: userID=%s, id=%s", userID, id)

	updates := map[string]interface{}{
		"enabled":    true,
		"testnet":    testnet,
		"updated_at": time.Now().UTC(),
	}
	if ntDataDir != "" {
		updates["nt_data_dir"] = ntDataDir
	}
	if ntInstrumentName != "" {
		updates["nt_instrument_name"] = ntInstrumentName
	}
	if ntDefaultContractQty != 0 {
		updates["nt_default_contract_qty"] = ntDefaultContractQty
	}

	// Only update encrypted fields if not empty
	if apiKey != "" {
		updates["api_key"] = crypto.EncryptedString(apiKey)
	}
	if secretKey != "" {
		updates["secret_key"] = crypto.EncryptedString(secretKey)
	}
	if passphrase != "" {
		updates["passphrase"] = crypto.EncryptedString(passphrase)
	}
	result := s.db.Model(&Exchange{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("exchange not found: id=%s, userID=%s", id, userID)
	}
	return nil
}

// UpdateAccountName updates the account name for an exchange
func (s *ExchangeStore) UpdateAccountName(userID, id, accountName string) error {
	result := s.db.Model(&Exchange{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"account_name": accountName,
			"updated_at":   time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("exchange not found: id=%s, userID=%s", id, userID)
	}
	return nil
}

// Delete deletes an exchange account
func (s *ExchangeStore) Delete(userID, id string) error {
	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Exchange{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("exchange not found: id=%s, userID=%s", id, userID)
	}
	logger.Infof("🗑️ Deleted exchange: id=%s, userID=%s", id, userID)
	return nil
}
