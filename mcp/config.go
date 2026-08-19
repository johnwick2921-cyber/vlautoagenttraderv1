package mcp

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"nofx/logger"
	"nofx/security"
)

// Config client configuration (centralized management of all configurations)
type Config struct {
	// Provider configuration
	Provider string
	APIKey   string
	BaseURL  string
	Model    string

	// Behavior configuration
	MaxTokens   int
	MaxContext  int // Model's max context window in tokens (0 = no limit)
	Temperature float64
	TopP        float64 // 0 = not sent
	UseFullURL  bool

	// Retry configuration
	MaxRetries      int
	RetryWaitBase   time.Duration
	RetryableErrors []string

	// Timeout configuration
	Timeout time.Duration

	// Dependency injection
	Logger     Logger
	HTTPClient *http.Client
}

// DefaultConfig returns default configuration.
// P0 2026-08-19 — NO silent defaults: every behaviour knob is env-driven with a
// consciously chosen safe default, and EffectiveAIParams() reports which of them
// the operator actually set (an unset param is logged as a WARNING at startup so
// the max_tokens=2000 accident class can never hide again).
func DefaultConfig() *Config {
	return &Config{
		// Default values
		// MaxTokens: the provider empirically accepts [1, 393216] (probed 2026-08-19:
		// 393216 accepted, 1048576 rejected). The ceiling is set well above any observed
		// completion (~3-4k) so a decision can never be truncated again; the 300s
		// timeout is the real bound.
		MaxTokens:       getEnvInt("AI_MAX_TOKENS", 32768),
		Temperature:     getEnvFloat("AI_TEMPERATURE", MCPClientTemperature),
		TopP:            getEnvFloat("AI_TOP_P", 0), // 0 = omit from the request
		MaxRetries:      getEnvInt("AI_MAX_RETRIES", MaxRetryTimes),
		RetryWaitBase:   time.Duration(getEnvInt("AI_RETRY_BACKOFF_SECONDS", 2)) * time.Second,
		Timeout:         ResolvedAITimeout(),
		RetryableErrors: retryableErrors,

		// Default dependencies (use global logger)
		Logger: logger.NewMCPLogger(),
		// The SAME resolved timeout the config carries. This used to be
		// SafeHTTPClient(DefaultTimeout) — the package constant — so the
		// AI_TIMEOUT_SECONDS value above was computed and then DISCARDED
		// (defect class 4): no env setting ever reached the transport.
		HTTPClient: security.SafeHTTPClient(ResolvedAITimeout()),
	}
}

// ResolvedAITimeout is the ONE resolution of the AI HTTP timeout, used by the
// config, the transport, and the per-exchange decision client alike so a literal
// can never shadow the owner's setting again (incident 2026-08-18: a hardcoded
// 180s decision-call cap killed DeepSeek reads mid-body once max_tokens was
// raised and reasoning responses started running 150s+).
//
// Precedence: AI_HTTP_TIMEOUT_SECONDS (canonical) → AI_TIMEOUT_SECONDS
// (pre-existing name, honored for backward compatibility) → 300s.
func ResolvedAITimeout() time.Duration {
	if v := os.Getenv("AI_HTTP_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return time.Duration(getEnvInt("AI_TIMEOUT_SECONDS", 300)) * time.Second
}

// getEnvInt reads integer from environment variable, returns default value if failed
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

// getEnvFloat reads a float from an environment variable, returns default if failed.
func getEnvFloat(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// EffectiveAIParams describes the effective AI-call parameters and which of them
// the operator set explicitly (false = the default is in force — logged as a
// WARNING at startup so silent defaults can never again hide a harmful value).
type EffectiveAIParams struct {
	Model               string
	MaxTokens           int
	Temperature         float64
	TopP                float64
	TimeoutSeconds      int
	MaxRetries          int
	RetryBackoffSeconds int

	MaxTokensSet    bool
	TemperatureSet  bool
	TopPSet         bool
	TimeoutSet      bool
	MaxRetriesSet   bool
	RetryBackoffSet bool
}

// EffectiveAIParamsSnapshot reports the env-driven values currently in force,
// including which knobs the operator explicitly set.
func EffectiveAIParamsSnapshot(model string) EffectiveAIParams {
	cfg := DefaultConfig()
	return EffectiveAIParams{
		Model:               model,
		MaxTokens:           cfg.MaxTokens,
		Temperature:         cfg.Temperature,
		TopP:                cfg.TopP,
		TimeoutSeconds:      int(cfg.Timeout / time.Second),
		MaxRetries:          cfg.MaxRetries,
		RetryBackoffSeconds: int(cfg.RetryWaitBase / time.Second),
		MaxTokensSet:        os.Getenv("AI_MAX_TOKENS") != "",
		TemperatureSet:      os.Getenv("AI_TEMPERATURE") != "",
		TopPSet:             os.Getenv("AI_TOP_P") != "",
		// Both names count as "operator set": ResolvedAITimeout honors the
		// canonical AI_HTTP_TIMEOUT_SECONDS first, then the legacy name — a
		// startup WARNING claiming the default is in force while the canonical
		// env drives the transport would be the honesty bug this report exists
		// to prevent.
		TimeoutSet:      os.Getenv("AI_HTTP_TIMEOUT_SECONDS") != "" || os.Getenv("AI_TIMEOUT_SECONDS") != "",
		MaxRetriesSet:   os.Getenv("AI_MAX_RETRIES") != "",
		RetryBackoffSet: os.Getenv("AI_RETRY_BACKOFF_SECONDS") != "",
	}
}

// getEnvString reads string from environment variable, returns default value if empty
func getEnvString(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
