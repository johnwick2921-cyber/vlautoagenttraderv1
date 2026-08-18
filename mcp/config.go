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
	MaxContext  int     // Model's max context window in tokens (0 = no limit)
	Temperature float64
	UseFullURL  bool

	// Retry configuration
	MaxRetries     int
	RetryWaitBase  time.Duration
	RetryableErrors []string

	// Timeout configuration
	Timeout time.Duration

	// Dependency injection
	Logger     Logger
	HTTPClient *http.Client
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		// Default values
		// P0 2026-08-19: deepseek-v4-pro is a REASONING model — it burns the entire
		// output budget on chain-of-thought. 2000 tokens truncated every first-pass
		// response (finish_reason=length, no decision emitted) and the retry then
		// defaulted to `wait`. 8000 gives the reasoning + decision JSON room to
		// finish (wire-proven: stop at ~3-4k with the short-reasoning instruction).
		MaxTokens:      getEnvInt("AI_MAX_TOKENS", 8000),
		Temperature:    MCPClientTemperature,
		MaxRetries:     MaxRetryTimes,
		RetryWaitBase:  2 * time.Second,
		Timeout:        DefaultTimeout,
		RetryableErrors: retryableErrors,

		// Default dependencies (use global logger)
		Logger:     logger.NewMCPLogger(),
		HTTPClient: security.SafeHTTPClient(DefaultTimeout),
	}
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

// getEnvString reads string from environment variable, returns default value if empty
func getEnvString(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
