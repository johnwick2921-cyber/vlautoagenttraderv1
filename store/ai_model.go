package store

import (
	"errors"
	"fmt"
	"nofx/crypto"
	"nofx/logger"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AIModelStore AI model storage
type AIModelStore struct {
	db *gorm.DB
}

// AIModel AI model configuration
type AIModel struct {
	ID              string                 `gorm:"primaryKey" json:"id"`
	UserID          string                 `gorm:"column:user_id;not null;default:default;index" json:"user_id"`
	Name            string                 `gorm:"not null" json:"name"`
	Provider        string                 `gorm:"not null" json:"provider"`
	Enabled         bool                   `gorm:"default:false" json:"enabled"`
	APIKey          crypto.EncryptedString `gorm:"column:api_key;default:''" json:"apiKey"`
	CustomAPIURL    string                 `gorm:"column:custom_api_url;default:''" json:"customApiUrl"`
	CustomModelName string                 `gorm:"column:custom_model_name;default:''" json:"customModelName"`
	// DeepSeek thinking knobs (4.5 API auto max): empty = inherit the env
	// defaults (DEEPSEEK_THINKING_MODE / AI_REASONING_EFFORT). A set value
	// overrides the env for THIS model only — no restart/redeploy needed.
	ThinkingMode    string    `gorm:"column:thinking_mode;default:''" json:"thinkingMode"`
	ReasoningEffort string    `gorm:"column:reasoning_effort;default:''" json:"reasoningEffort"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (AIModel) TableName() string { return "ai_models" }

// NewAIModelStore creates a new AIModelStore
func NewAIModelStore(db *gorm.DB) *AIModelStore {
	return &AIModelStore{db: db}
}

func (s *AIModelStore) initTables() error {
	// For PostgreSQL with existing table, skip AutoMigrate
	if s.db.Dialector.Name() == "postgres" {
		var tableExists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'ai_models'`).Scan(&tableExists)
		if tableExists > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&AIModel{})
}

func (s *AIModelStore) initDefaultData() error {
	// No longer pre-populate AI models - create on demand when user configures
	return nil
}

// FindOrphanClaw402 finds a claw402 model whose user_id no longer exists in the users table.
// Used to recover wallets after account reset.
func (s *AIModelStore) FindOrphanClaw402() (*AIModel, error) {
	var model AIModel
	// Deterministic pick when several orphan claw402 rows exist (multi-entry safe):
	// most-recently-updated first, stable id tie-break.
	err := s.db.Where("provider = ? AND api_key != '' AND user_id NOT IN (SELECT id FROM users)", "claw402").
		Order("updated_at DESC, id ASC").
		First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// AdoptModel re-assigns an existing model to a new user.
func (s *AIModelStore) AdoptModel(modelID, newUserID string) error {
	return s.db.Model(&AIModel{}).Where("id = ?", modelID).
		Update("user_id", newUserID).Error
}

// List retrieves user's AI model list
func (s *AIModelStore) List(userID string) ([]*AIModel, error) {
	var models []*AIModel
	err := s.db.Where("user_id = ?", userID).Order("id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

// Get retrieves a single AI model
func (s *AIModelStore) Get(userID, modelID string) (*AIModel, error) {
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	candidates := []string{}
	if userID != "" {
		candidates = append(candidates, userID)
	}
	if userID != "default" {
		candidates = append(candidates, "default")
	}
	if len(candidates) == 0 {
		candidates = append(candidates, "default")
	}

	for _, uid := range candidates {
		var model AIModel
		err := s.db.Where("user_id = ? AND id = ?", uid, modelID).First(&model).Error
		if err == nil {
			return &model, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// GetByID retrieves an AI model by ID only
func (s *AIModelStore) GetByID(modelID string) (*AIModel, error) {
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	var model AIModel
	err := s.db.Where("id = ?", modelID).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// GetDefault retrieves the default enabled AI model
func (s *AIModelStore) GetDefault(userID string) (*AIModel, error) {
	if userID == "" {
		userID = "default"
	}
	model, err := s.firstEnabledUsable(userID)
	if err == nil {
		return model, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if userID != "default" {
		return s.firstEnabledUsable("default")
	}
	return nil, fmt.Errorf("please configure an available AI model in the system first")
}

func (s *AIModelStore) firstEnabledUsable(userID string) (*AIModel, error) {
	var models []AIModel
	err := s.db.Where("user_id = ? AND enabled = ? AND api_key != ''", userID, true).
		Order("updated_at DESC, id ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	for i := range models {
		if hasUsableAPIKey(models[i]) {
			return &models[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// GetAnyEnabled returns the first enabled AI model across all users.
// Used by single-user features (e.g. Telegram bot) that need any working LLM client.
func (s *AIModelStore) GetAnyEnabled() (*AIModel, error) {
	var models []AIModel
	err := s.db.Where("enabled = ?", true).
		Order("updated_at DESC, id ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	for i := range models {
		if hasUsableAPIKey(models[i]) {
			return &models[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func hasUsableAPIKey(model AIModel) bool {
	if strings.TrimSpace(string(model.APIKey)) != "" {
		return true
	}
	envKeyByProvider := map[string]string{
		"deepseek": "DEEPSEEK_API_KEY",
		"openai":   "OPENAI_API_KEY",
		"claude":   "ANTHROPIC_API_KEY",
		"gemini":   "GEMINI_API_KEY",
		"grok":     "XAI_API_KEY",
		"kimi":     "MOONSHOT_API_KEY",
		"minimax":  "MINIMAX_API_KEY",
		"qwen":     "DASHSCOPE_API_KEY",
	}
	envKey := envKeyByProvider[strings.ToLower(strings.TrimSpace(model.Provider))]
	return envKey != "" && strings.TrimSpace(os.Getenv(envKey)) != ""
}

// defaultProviderName returns the conventional display name for a provider.
func defaultProviderName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return "DeepSeek AI"
	case "qwen":
		return "Qwen AI"
	default:
		return provider + " AI"
	}
}

// PickProviderModel deterministically selects among models that share `provider`.
// Preference order: enabled beats disabled; then most-recently-updated (UpdatedAt);
// then lowest ID (stable tie-break). Returns the chosen model (nil if none match)
// and the number of provider candidates seen. This is the single source of truth for
// "which row wins" once multiple entries per provider exist — callers MUST log the
// chosen id when candidates > 1 so there is never a silent first-row-wins.
func PickProviderModel(models []*AIModel, provider string) (*AIModel, int) {
	norm := strings.ToLower(strings.TrimSpace(provider))
	var best *AIModel
	count := 0
	for _, m := range models {
		if m == nil || strings.ToLower(strings.TrimSpace(m.Provider)) != norm {
			continue
		}
		count++
		if best == nil || betterProviderModel(m, best) {
			best = m
		}
	}
	return best, count
}

// betterProviderModel: enabled > disabled, then newer UpdatedAt, then lower ID.
func betterProviderModel(a, b *AIModel) bool {
	if a.Enabled != b.Enabled {
		return a.Enabled
	}
	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		return a.UpdatedAt.After(b.UpdatedAt)
	}
	return a.ID < b.ID
}

// CreateEntry creates a NEW, independently-addressable AI model row for a provider,
// enabling MULTIPLE entries per provider (e.g. "DeepSeek-main" + "DeepSeek-backup").
// The id is always unique — userID_provider_<shortuuid> — so existing rows (which
// traders reference by id) are never touched. Returns the new row id.
func (s *AIModelStore) CreateEntry(userID, provider, name string, enabled bool, apiKey, customAPIURL, customModelName string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "", fmt.Errorf("provider is required")
	}
	if userID == "" {
		userID = "default"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultProviderName(provider)
	}
	id := fmt.Sprintf("%s_%s_%s", userID, provider, uuid.NewString()[:8])
	m := &AIModel{
		ID:              id,
		UserID:          userID,
		Name:            name,
		Provider:        provider,
		Enabled:         enabled,
		APIKey:          crypto.EncryptedString(apiKey),
		CustomAPIURL:    customAPIURL,
		CustomModelName: customModelName,
	}
	if err := s.db.Create(m).Error; err != nil {
		return "", err
	}
	logger.Infof("✓ Created AI model ENTRY id=%s provider=%s name=%q enabled=%v", id, provider, name, enabled)
	return id, nil
}

// Update updates AI model, creates if not exists
// IMPORTANT: If apiKey is empty string, the existing API key will be preserved (not overwritten)
func (s *AIModelStore) Update(userID, id string, enabled bool, apiKey, customAPIURL, customModelName string) error {
	return s.UpdateWithName(userID, id, "", enabled, apiKey, customAPIURL, customModelName)
}

func (s *AIModelStore) UpdateWithName(userID, id, name string, enabled bool, apiKey, customAPIURL, customModelName string) error {
	// Try exact ID match first
	var existingModel AIModel
	err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&existingModel).Error
	if err == nil {
		// Update existing model
		updates := map[string]interface{}{
			"enabled":           enabled,
			"custom_api_url":    customAPIURL,
			"custom_model_name": customModelName,
			"updated_at":        time.Now().UTC(),
		}
		if strings.TrimSpace(name) != "" {
			updates["name"] = strings.TrimSpace(name)
		}
		// If apiKey is not empty, update it (encryption handled by crypto.EncryptedString)
		if apiKey != "" {
			updates["api_key"] = crypto.EncryptedString(apiKey)
		}
		return s.db.Model(&existingModel).Updates(updates).Error
	}

	// LEGACY provider matching — SCOPED to genuinely-legacy BARE-provider ids only
	// (e.g. "deepseek"), never composite row ids ("userID_provider[_uuid]"). This match
	// is what let a second same-provider save silently overwrite the first; gating it on
	// a bare id (no underscore) lets an explicit new entry (unique id via CreateEntry)
	// keep its own row. When multiple rows share the provider, pick DETERMINISTICALLY
	// (enabled + most-recent) and LOG it — no silent first-row-wins.
	provider := id
	if !strings.Contains(id, "_") {
		var providerRows []*AIModel
		if e := s.db.Where("user_id = ? AND provider = ?", userID, provider).Find(&providerRows).Error; e == nil && len(providerRows) > 0 {
			chosen, n := PickProviderModel(providerRows, provider)
			if chosen != nil {
				if n > 1 {
					logger.Warnf("⚠️ legacy provider-match %q → %d entries; updating enabled+most-recent id=%s", provider, n, chosen.ID)
				} else {
					logger.Warnf("⚠️ Using legacy provider matching to update model: %s -> %s", provider, chosen.ID)
				}
				updates := map[string]interface{}{
					"enabled":           enabled,
					"custom_api_url":    customAPIURL,
					"custom_model_name": customModelName,
					"updated_at":        time.Now().UTC(),
				}
				if strings.TrimSpace(name) != "" {
					updates["name"] = strings.TrimSpace(name)
				}
				if apiKey != "" {
					updates["api_key"] = crypto.EncryptedString(apiKey)
				}
				return s.db.Model(chosen).Updates(updates).Error
			}
		}
	}

	// Create new record
	if provider == id && (provider == "deepseek" || provider == "qwen") {
		provider = id
	} else {
		parts := strings.Split(id, "_")
		if len(parts) >= 2 {
			provider = parts[len(parts)-1]
		} else {
			provider = id
		}
	}

	// Try to get a sensible default name from an existing model with the same provider.
	var refModel AIModel
	defaultName := ""
	if err := s.db.Where("provider = ?", provider).First(&refModel).Error; err == nil {
		defaultName = refModel.Name
	} else {
		if provider == "deepseek" {
			defaultName = "DeepSeek AI"
		} else if provider == "qwen" {
			defaultName = "Qwen AI"
		} else {
			defaultName = provider + " AI"
		}
	}
	finalName := strings.TrimSpace(name)
	if finalName == "" {
		finalName = strings.TrimSpace(defaultName)
	}

	newModelID := id
	if id == provider {
		newModelID = fmt.Sprintf("%s_%s", userID, provider)
	}

	logger.Infof("✓ Creating new AI model configuration: ID=%s, Provider=%s, Name=%s", newModelID, provider, finalName)
	newModel := &AIModel{
		ID:              newModelID,
		UserID:          userID,
		Name:            finalName,
		Provider:        provider,
		Enabled:         enabled,
		APIKey:          crypto.EncryptedString(apiKey),
		CustomAPIURL:    customAPIURL,
		CustomModelName: customModelName,
	}
	return s.db.Create(newModel).Error
}

// UpdateThinking writes the per-model DeepSeek thinking knobs (4.5 API auto
// max). Empty values are WRITTEN as-is — empty means "inherit the env default",
// and clearing an override back to inherit is a supported transition. Mirrors
// UpdateWithName's two-step lookup (exact id → scoped legacy bare-provider); a
// missing row is returned as gorm.ErrRecordNotFound and the caller decides.
func (s *AIModelStore) UpdateThinking(userID, id, mode, effort string) error {
	var existingModel AIModel
	err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&existingModel).Error
	if err != nil {
		if !strings.Contains(id, "_") {
			var providerRows []*AIModel
			if e := s.db.Where("user_id = ? AND provider = ?", userID, id).Find(&providerRows).Error; e == nil && len(providerRows) > 0 {
				if chosen, _ := PickProviderModel(providerRows, id); chosen != nil {
					return s.db.Model(chosen).Updates(map[string]interface{}{
						"thinking_mode":    mode,
						"reasoning_effort": effort,
						"updated_at":       time.Now().UTC(),
					}).Error
				}
			}
		}
		return err
	}
	return s.db.Model(&existingModel).Updates(map[string]interface{}{
		"thinking_mode":    mode,
		"reasoning_effort": effort,
		"updated_at":       time.Now().UTC(),
	}).Error
}

// Create creates an AI model
// ResolveClaw402WalletKey returns the claw402 wallet private key for a user.
// If preferredModelID is non-empty and points to a claw402 model, its key is returned first.
// Otherwise the first enabled claw402 model in the user's model list is used.
// Returns ("", nil) when no claw402 model is configured — callers should treat this as
// "no paid data routing" rather than an error.
func (s *AIModelStore) ResolveClaw402WalletKey(userID, preferredModelID string) (string, error) {
	if preferredModelID != "" {
		model, err := s.Get(userID, preferredModelID)
		if err != nil {
			return "", fmt.Errorf("failed to load selected AI model")
		}
		if model.Provider == "claw402" {
			walletKey := string(model.APIKey)
			if walletKey == "" {
				return "", fmt.Errorf("selected claw402 model is missing wallet private key")
			}
			return walletKey, nil
		}
	}

	models, err := s.List(userID)
	if err != nil {
		return "", fmt.Errorf("failed to load AI models")
	}

	// Deterministic pick among claw402 rows carrying a wallet key: enabled +
	// most-recent (same rule as PickProviderModel). No silent first-by-id wins.
	var best *AIModel
	n := 0
	for _, model := range models {
		if model == nil || model.Provider != "claw402" || string(model.APIKey) == "" {
			continue
		}
		n++
		if best == nil || betterProviderModel(model, best) {
			best = model
		}
	}
	if best != nil {
		if n > 1 {
			logger.Warnf("⚠️ claw402 wallet resolve: %d keyed entries; using enabled+most-recent id=%s", n, best.ID)
		}
		return string(best.APIKey), nil
	}

	return "", nil
}

func (s *AIModelStore) Create(userID, id, name, provider string, enabled bool, apiKey, customAPIURL string) error {
	model := &AIModel{
		ID:           id,
		UserID:       userID,
		Name:         name,
		Provider:     provider,
		Enabled:      enabled,
		APIKey:       crypto.EncryptedString(apiKey),
		CustomAPIURL: customAPIURL,
	}
	// Use FirstOrCreate to ignore if already exists
	return s.db.Where("id = ?", id).FirstOrCreate(model).Error
}

// Delete removes a user-owned AI model configuration.
func (s *AIModelStore) Delete(userID, id string) error {
	result := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&AIModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ai model not found: id=%s, userID=%s", id, userID)
	}
	logger.Infof("🗑️ Deleted AI model: id=%s, userID=%s", id, userID)
	return nil
}
