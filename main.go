package main

import (
	"fmt"
	"log/slog"
	nofxiagent "nofx/agent"
	"nofx/api"
	"nofx/auth"
	"nofx/config"
	"nofx/crypto"
	"nofx/kernel"
	"nofx/logger"
	"nofx/manager"
	"nofx/mcp"
	_ "nofx/mcp/payment"
	_ "nofx/mcp/provider"
	"nofx/store"
	"nofx/telegram"
	"nofx/telemetry"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env environment variables
	_ = godotenv.Load()

	// Initialize logger
	logger.Init(nil)

	logger.Info("╔════════════════════════════════════════════════════════════╗")
	logger.Info("║           🚀 NOFX - AI-Powered Trading System              ║")
	logger.Info("╚════════════════════════════════════════════════════════════╝")

	// Initialize global configuration (loaded from .env)
	config.Init()
	cfg := config.Get()
	logger.Info("✅ Configuration loaded")

	// Initialize encryption service BEFORE database (so EncryptedString can decrypt on read)
	logger.Info("🔐 Initializing encryption service...")
	cryptoService, err := crypto.NewCryptoService()
	if err != nil {
		logger.Fatalf("❌ Failed to initialize encryption service: %v", err)
	}
	crypto.SetGlobalCryptoService(cryptoService)
	logger.Info("✅ Encryption service initialized successfully")

	// Initialize database from configuration
	// For backward compatibility: command line arg overrides config (SQLite only)
	if len(os.Args) > 1 {
		cfg.DBPath = os.Args[1]
	}
	// Ensure data directory exists (for SQLite)
	if cfg.DBType == "sqlite" {
		if dir := filepath.Dir(cfg.DBPath); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				logger.Errorf("Failed to create data directory: %v", err)
			}
		}
	}

	logger.Infof("📋 Initializing database (%s)...", cfg.DBType)
	dbType := store.DBTypeSQLite
	if cfg.DBType == "postgres" {
		dbType = store.DBTypePostgres
	}
	st, err := store.NewWithConfig(store.DBConfig{
		Type:     dbType,
		Path:     cfg.DBPath,
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	})
	if err != nil {
		logger.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer st.Close()

	// Initialize installation ID for experience improvement (anonymous statistics)
	initInstallationID(st)

	// Set JWT secret
	auth.SetJWTSecret(cfg.JWTSecret)
	logger.Info("🔑 JWT secret configured")

	// P0 timezone — CT is canonical for EVERY rendered time (owner rule
	// 2026-08-19). The host's local zone is ignored by every renderer.
	logger.Infof("🕐 Timezone pinned: %s (CT) — all prompts, cards, digests and logs render CT, host TZ ignored", kernel.CanonicalZone)

	// P0 2026-08-19 — print every effective AI parameter at startup so a silent
	// default can never hide again (the max_tokens=2000 disease). Any knob the
	// operator did NOT set explicitly is called out as a WARNING.
	ai := mcp.EffectiveAIParamsSnapshot(mcp.DefaultDeepSeekModel)
	logger.Infof("🧠 AI params in force: model=%s max_tokens=%d temperature=%.2f top_p=%s timeout=%ds retries=%d backoff=%ds · truncated-responses=%d",
		ai.Model, ai.MaxTokens, ai.Temperature, formatTopP(ai.TopP), ai.TimeoutSeconds, ai.MaxRetries, ai.RetryBackoffSeconds, mcp.TruncatedResponses.Load())
	unset := []string{}
	if !ai.MaxTokensSet {
		unset = append(unset, "AI_MAX_TOKENS")
	}
	if !ai.TemperatureSet {
		unset = append(unset, "AI_TEMPERATURE")
	}
	if !ai.TopPSet {
		unset = append(unset, "AI_TOP_P")
	}
	if !ai.TimeoutSet {
		unset = append(unset, "AI_TIMEOUT_SECONDS")
	}
	if !ai.MaxRetriesSet {
		unset = append(unset, "AI_MAX_RETRIES")
	}
	if !ai.RetryBackoffSet {
		unset = append(unset, "AI_RETRY_BACKOFF_SECONDS")
	}
	if len(unset) > 0 {
		logger.Warnf("⚠️ AI params at UNSET defaults (nobody chose these explicitly): %v — set them in .env if the defaults are not what you intend", unset)
	}

	// WebSocket market monitor is NO LONGER USED
	// All K-line data now comes from CoinAnk API instead of Binance WebSocket cache
	// Commented out to reduce unnecessary connections:
	// go market.NewWSMonitor(150).Start(nil)
	// logger.Info("📊 WebSocket market monitor started")
	// time.Sleep(500 * time.Millisecond)
	logger.Info("📊 Using CoinAnk API for all market data (WebSocket cache disabled)")

	// Create TraderManager
	traderManager := manager.NewTraderManager()

	// Load all traders from database to memory (may auto-start traders with IsRunning=true)
	if err := traderManager.LoadTradersFromStore(st); err != nil {
		logger.Fatalf("❌ Failed to load traders: %v", err)
	}

	// Display loaded trader information
	traders, err := st.Trader().List("default")
	if err != nil {
		logger.Fatalf("❌ Failed to get trader list: %v", err)
	}

	logger.Info("🤖 AI Trader Configurations in Database:")
	if len(traders) == 0 {
		logger.Info("  (No trader configurations, please create via Web interface)")
	} else {
		for _, t := range traders {
			status := "❌ Stopped"
			if t.IsRunning {
				status = "✅ Running"
			}
			idShort := t.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			logger.Infof("  • %s [%s] %s - AI Model: %s, Exchange: %s",
				t.Name, idShort, status, t.AIModelID, t.ExchangeID)
		}
	}

	// Plan 4 Task 23 — risk + audit endpoints are registered inside
	// api.NewServer / setupRoutes (gin router); see api/server.go for
	// POST /api/risk/force-flat, GET /api/risk/status, GET /api/audit/decisions.

	// Plan 4 Task 25 — Prometheus metrics endpoint (T25 owns this marker; T23 leaves space below).

	// Start API server
	// P1 — BOOT INTEGRITY ASSERTION. Runs before any trader cycles. A mismatch
	// with the intended release, or a drifted prompt golden, REFUSES TRADING for
	// this process (entries blocked; everything else stays read-only usable).
	integrity := kernel.AssertBootIntegrity()
	if integrity.Refused {
		logger.Errorf("%s", integrity.Line())
		logger.Errorf("🔐 TRADING REFUSED — %s", integrity.Reason)
		logger.Errorf("🔐 No new positions will be opened until this is fixed and the bot is restarted.")
	} else {
		logger.Infof("%s", integrity.Line())
	}

	// SANDBOX: a demo instance has no NT8 wire, so install a deterministic
	// synthetic bar feed — without it level_facts/price/chart/armor are all empty.
	if cfg.SandboxMode {
		api.InstallSandboxBars("MNQ", 30231.5)
		logger.Warnf("🧪 SANDBOX MODE — synthetic bars installed, canned planner replies, no live trading")
	}

	server := api.NewServer(traderManager, st, cryptoService, cfg.APIServerHost, cfg.APIServerPort)

	// Create hot-reload channel for Telegram bot; wire it to the API server
	// so that POST /api/telegram can trigger a bot restart when the token changes.
	telegramReloadCh := make(chan struct{}, 1)
	server.SetTelegramReloadCh(telegramReloadCh)

	// Start the NOFXi web agent on top of the current dev branch services.
	nofxiAgent := nofxiagent.New(traderManager, st, nil, slog.Default())
	agentWeb := nofxiagent.NewWebHandler(nofxiAgent, slog.Default())
	server.RegisterAgentHandler(agentWeb)
	nofxiAgent.Start()
	defer nofxiAgent.Stop()

	go func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("❌ Failed to start API server: %v", err)
		}
	}()

	// Start Telegram bot (if TELEGRAM_BOT_TOKEN is configured)
	go telegram.Start(cfg, st, telegramReloadCh)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("✅ System started successfully, waiting for trading commands...")
	logger.Info("📌 Tip: Use Ctrl+C to stop the system")

	<-quit
	logger.Info("📴 Shutdown signal received, closing system...")

	if err := server.Shutdown(); err != nil {
		logger.Warnf("⚠️ HTTP server shutdown error: %v", err)
	}
	logger.Info("✅ HTTP server stopped")

	// nofxiAgent.Stop() is handled by defer above

	// Stop all traders
	traderManager.StopAll()
	logger.Info("✅ System shut down safely")
}

// initInstallationID initializes the anonymous installation ID for experience improvement
// This ID is persisted in database and used for anonymous usage statistics
func initInstallationID(st *store.Store) {
	const key = "installation_id"

	// Try to load from database
	installationID, err := st.GetSystemConfig(key)
	if err != nil {
		logger.Warnf("⚠️ Failed to load installation ID: %v", err)
	}

	// Generate new ID if not exists
	if installationID == "" {
		installationID = uuid.New().String()
		if err := st.SetSystemConfig(key, installationID); err != nil {
			logger.Warnf("⚠️ Failed to save installation ID: %v", err)
		}
		logger.Infof("📊 Generated new installation ID: %s", installationID[:8]+"...")
	}

	// Set installation ID in experience module
	telemetry.SetInstallationID(installationID)
}

// formatTopP renders the top_p value for the startup log (0 = omitted).
func formatTopP(v float64) string {
	if v <= 0 {
		return "omitted"
	}
	return fmt.Sprintf("%.2f", v)
}
