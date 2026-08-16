package config

import (
	"nofx/logger"
	"nofx/mcp"
	"nofx/telemetry"
	"os"
	"strconv"
	"strings"
)

// Global configuration instance
var global *Config

// Config is the global configuration (loaded from .env)
// Only contains truly global config, trading related config is at trader/strategy level
type Config struct {
	// Service configuration
	APIServerHost string // interface the HTTP API binds to; default 127.0.0.1 (loopback only)
	APIServerPort int
	JWTSecret     string

	// Database configuration
	DBType     string // sqlite or postgres
	DBPath     string // SQLite database file path
	// SandboxMode (SANDBOX_MODE=1) turns this process into an isolated demo: the
	// trading loop never places an order, the planner uses canned replies unless
	// SANDBOX_LLM=real, and the UI paints a SANDBOX banner. Default OFF — the live
	// bot is byte-identical unless the env var is set.
	SandboxMode bool
	DBHost     string // PostgreSQL host
	DBPort     int    // PostgreSQL port
	DBUser     string // PostgreSQL user
	DBPassword string // PostgreSQL password
	DBName     string // PostgreSQL database name
	DBSSLMode  string // PostgreSQL SSL mode

	// Security configuration
	// TransportEncryption enables browser-side encryption for API keys
	// Requires HTTPS or localhost. Set to false for HTTP access via IP.
	TransportEncryption bool

	// Experience improvement (anonymous usage statistics)
	// Helps us understand product usage and improve the experience
	// Set EXPERIENCE_IMPROVEMENT=false to disable
	ExperienceImprovement bool

	// Market data provider API keys
	AlpacaAPIKey    string // Alpaca API key for US stocks
	AlpacaSecretKey string // Alpaca secret key
	TwelveDataKey   string // TwelveData API key for forex & metals

	// Databento (NQ futures data)
	DatabentoAPIKey  string
	DatabentoDataset string // e.g., "GLBX.MDP3"

	// NinjaTrader (CSV bridge for execution)
	NinjaTraderDataDir string // e.g., "/mnt/c/Users/<u>/NofxTrader/data"

	// AllowedNTAccounts is the allow-list of NT8 sub-account names a trader may be
	// bound to (multi-account safety rail; env NT_ALLOWED_ACCOUNTS, comma-separated).
	// Empty = no extra restriction beyond the server-side SIM-only guard. A non-empty
	// list HARD-blocks selecting any account not on it (e.g. keep a funded account off
	// the list so it can never be chosen). This is the rail Stage-2 routing will enforce.
	AllowedNTAccounts []string

	// Trading mode: "crypto" (default, original behavior) or "futures"
	TradingMode string

	// Plan 3 Task 21 — Risk limits (hard server-side kill switches).
	// Defaults: $500 daily loss, 2 concurrent trades, $50k notional, 5 contracts/order.
	// Zero value disables the corresponding check.
	RiskMaxDailyLossUSD      float64 // RISK_MAX_DAILY_LOSS_USD
	RiskMaxConcurrentTrades  int     // RISK_MAX_CONCURRENT_TRADES
	RiskMaxNotionalUSD       float64 // RISK_MAX_NOTIONAL_USD
	RiskMaxContractsPerOrder int     // RISK_MAX_CONTRACTS_PER_ORDER
}

// Init initializes global configuration (from .env)
func Init() {
	cfg := &Config{
		// SECURITY: loopback by default. The API carries broker/model credentials
		// and trader control; it must never be reachable off-host unless the
		// operator opts in explicitly via API_SERVER_HOST.
		APIServerHost:         "127.0.0.1",
		APIServerPort:         8080,
		ExperienceImprovement: true, // Default: enabled to help improve the product
		// Database defaults
		DBType:    "sqlite",
		DBPath:    "data/data.db",
		DBHost:    "localhost",
		DBPort:    5432,
		DBUser:    "postgres",
		DBName:    "nofx",
		DBSSLMode: "disable",
	}

	// Load from environment variables
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = strings.TrimSpace(v)
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "default-jwt-secret-change-in-production"
		logger.Warnf("⚠️  JWT_SECRET env var not set; using INSECURE default. " +
			"Acceptable for localhost-only paper trading. " +
			"Set JWT_SECRET in .env (e.g. `openssl rand -base64 64`) before any network-exposed deploy.")
	}

	if v := os.Getenv("API_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			cfg.APIServerPort = port
		}
	}

	// API_SERVER_HOST overrides the bind interface. "0.0.0.0" (or any non-loopback
	// address) exposes the API to the network — warn loudly, since the surface
	// includes trader control and credential-bearing config endpoints.
	if v := strings.TrimSpace(os.Getenv("API_SERVER_HOST")); v != "" {
		cfg.APIServerHost = v
		if v != "127.0.0.1" && v != "localhost" && v != "::1" {
			logger.Warnf("⚠️  API_SERVER_HOST=%q — the API is bound OFF-LOOPBACK and reachable from the network. "+
				"Ensure JWT_SECRET is strong and a firewall/reverse proxy fronts it.", v)
		}
	}

	// Transport encryption: default false for easier deployment
	// Set TRANSPORT_ENCRYPTION=true to enable (requires HTTPS or localhost)
	if v := os.Getenv("TRANSPORT_ENCRYPTION"); v != "" {
		cfg.TransportEncryption = strings.ToLower(v) == "true"
	}

	// Experience improvement: anonymous usage statistics
	// Default enabled, set EXPERIENCE_IMPROVEMENT=false to disable
	if v := os.Getenv("EXPERIENCE_IMPROVEMENT"); v != "" {
		cfg.ExperienceImprovement = strings.ToLower(v) != "false"
	}

	// Market data provider API keys
	cfg.AlpacaAPIKey = os.Getenv("ALPACA_API_KEY")
	cfg.AlpacaSecretKey = os.Getenv("ALPACA_SECRET_KEY")
	cfg.TwelveDataKey = os.Getenv("TWELVEDATA_API_KEY")

	// Databento + NinjaTrader (NQ futures path)
	cfg.DatabentoAPIKey = os.Getenv("DATABENTO_API_KEY")
	cfg.DatabentoDataset = getEnvOrDefault("DATABENTO_DATASET", "GLBX.MDP3")
	cfg.NinjaTraderDataDir = os.Getenv("NINJATRADER_DATA_DIR")
	for _, a := range strings.Split(os.Getenv("NT_ALLOWED_ACCOUNTS"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			cfg.AllowedNTAccounts = append(cfg.AllowedNTAccounts, a)
		}
	}
	cfg.TradingMode = getEnvOrDefault("TRADING_MODE", "crypto")

	// Plan 3 Task 21 — Risk limits
	cfg.RiskMaxDailyLossUSD = getEnvFloat("RISK_MAX_DAILY_LOSS_USD", 500)
	cfg.RiskMaxConcurrentTrades = getEnvInt("RISK_MAX_CONCURRENT_TRADES", 2)
	cfg.RiskMaxNotionalUSD = getEnvFloat("RISK_MAX_NOTIONAL_USD", 50_000)
	cfg.RiskMaxContractsPerOrder = getEnvInt("RISK_MAX_CONTRACTS_PER_ORDER", 5)

	// Database configuration
	if v := os.Getenv("DB_TYPE"); v != "" {
		cfg.DBType = strings.ToLower(v)
	}
	if v := os.Getenv("SANDBOX_MODE"); v == "1" || strings.EqualFold(v, "true") {
		cfg.SandboxMode = true
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DBHost = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			cfg.DBPort = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DBUser = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.DBPassword = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.DBSSLMode = v
	}

	global = cfg

	// Initialize experience improvement (installation ID will be set after database init)
	telemetry.Init(cfg.ExperienceImprovement, "")

	// Set up AI token usage tracking callback
	mcp.TokenUsageCallback = func(usage mcp.TokenUsage) {
		telemetry.TrackAIUsage(telemetry.AIUsageEvent{
			ModelProvider: usage.Provider,
			ModelName:     usage.Model,
			Channel:       usage.Channel(),
			InputTokens:   usage.PromptTokens,
			OutputTokens:  usage.CompletionTokens,
		})
	}
}

// Get returns the global configuration
func Get() *Config {
	if global == nil {
		Init()
	}
	return global
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
