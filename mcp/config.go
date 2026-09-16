package mcp

import (
	"net/http"
	"os"
	"strconv"
	"strings"
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

	// DeepSeek thinking mode (https://api-docs.deepseek.com/guides/thinking_mode):
	// thinking {type: enabled|disabled} + reasoning_effort low|high|max.
	// Docs: thinking mode is ON by default with effort high; "max" is the true
	// maximum (medium/xhigh map to high). Empty string = omit from the request.
	ThinkingMode    string // "enabled" (default) | "disabled" | ""
	ReasoningEffort string // "max" (default) | "high" | "low" | ""

	// Retry configuration
	MaxRetries      int
	RetryWaitBase   time.Duration
	RetryableErrors []string
	// CLASS 41 (2026-09-02) — the planner STREAM path's own retry policy:
	// StreamTries counts CALLS (default 3 = two retries); StreamBackoff is the
	// exponential wait schedule between them (default 2s → 15s → 45s; the last
	// value repeats). A flapping edge needs time — the fixed 2s wait let call 2
	// die 18s after call 1 on 2026-09-01 23:47 CT.
	StreamTries   int
	StreamBackoff []time.Duration

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
		ThinkingMode:    getEnvString("DEEPSEEK_THINKING_MODE", "enabled"),
		ReasoningEffort: getEnvString("AI_REASONING_EFFORT", "max"),
		MaxRetries:      getEnvInt("AI_MAX_RETRIES", MaxRetryTimes),
		RetryWaitBase:   time.Duration(getEnvInt("AI_RETRY_BACKOFF_SECONDS", 2)) * time.Second,
		StreamTries:     StreamRetryTries(),
		StreamBackoff:   StreamRetryBackoffSchedule(),
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
	ThinkingMode        string
	ReasoningEffort     string

	MaxTokensSet    bool
	TemperatureSet  bool
	TopPSet         bool
	TimeoutSet      bool
	MaxRetriesSet   bool
	RetryBackoffSet bool
	ThinkingSet     bool
	ReasoningSet    bool
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
		ThinkingMode:        cfg.ThinkingMode,
		ReasoningEffort:     cfg.ReasoningEffort,
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
		ThinkingSet:     os.Getenv("DEEPSEEK_THINKING_MODE") != "",
		ReasoningSet:    os.Getenv("AI_REASONING_EFFORT") != "",
	}
}

// getEnvString reads string from environment variable, returns default value if empty
func getEnvString(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// StreamRetryTries (class 41) — AI_PLAN_STREAM_TRIES, the number of CALLS the
// planner stream path makes per planner attempt before giving up (default 3 =
// two retries). Bounded 1..6.
func StreamRetryTries() int {
	n := getEnvInt("AI_PLAN_STREAM_TRIES", 3)
	if n < 1 {
		n = 1
	}
	if n > 6 {
		n = 6
	}
	return n
}

// StreamRetryBackoffSchedule (class 41) — AI_PLAN_STREAM_BACKOFF, a comma list
// of Go durations waited before retry 1, 2, 3… on the planner stream path
// (default "2s,15s,45s"). Unparseable entries are skipped; an empty result
// falls back to the default. Beyond the list the last value repeats.
func StreamRetryBackoffSchedule() []time.Duration {
	def := []time.Duration{2 * time.Second, 15 * time.Second, 45 * time.Second}
	raw := strings.TrimSpace(os.Getenv("AI_PLAN_STREAM_BACKOFF"))
	if raw == "" {
		return def
	}
	var out []time.Duration
	for _, part := range strings.Split(raw, ",") {
		d, err := time.ParseDuration(strings.TrimSpace(part))
		if err != nil || d <= 0 {
			continue
		}
		out = append(out, d)
	}
	if len(out) == 0 {
		return def
	}
	return out
}

// streamBackoffFor returns the wait before retry number n (1-based) from the
// schedule; past the end the last entry repeats; never zero.
func streamBackoffFor(n int, sched []time.Duration) time.Duration {
	if len(sched) == 0 {
		return 2 * time.Second
	}
	if n < 1 {
		n = 1
	}
	if n > len(sched) {
		n = len(sched)
	}
	return sched[n-1]
}
