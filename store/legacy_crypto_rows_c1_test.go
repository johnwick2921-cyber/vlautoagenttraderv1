package store

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"

	"vl/crypto"
)

// TestLegacyCryptoRowsLoadWithFuturesSettingsIntact is the C1 owner-data load
// proof (test side): a stored strategy / exchange / ai_model row carrying EVERY
// legacy crypto field must LOAD through its production call site with the
// futures-effective settings unchanged — the legacy fields are read-and-ignored,
// never a load failure and never a corruption of the futures configuration.
// Synthetic fixture only (never data.db or a copy of it; the real-copy check is
// the CTO's).
//
// Boot-cleanup survival IS asserted for the exchange row here (the C1 P0
// cleanup-skip is ported — unsupported types are kept, never deleted), and the
// legacy DB COLUMNS are proven at the raw-SQL level to survive a load and an
// update through the store (CTO item 2: the struct mapping is not enough — the
// physical columns must round-trip).
func TestLegacyCryptoRowsLoadWithFuturesSettingsIntact(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "c1-legacy.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	const userID = "u1"

	// 1) STRATEGY — every legacy crypto field, plus explicit futures settings.
	legacyStrategyJSON := `{
		"coin_source": {
			"source_type": "ai500",
			"static_coins": [],
			"use_ai500": true,
			"ai500_limit": 50,
			"oi_top_limit": 20,
			"use_oi_top": true,
			"use_hyper_all": true,
			"use_hyper_main": true,
			"netflow_source": true
		},
		"enable_sentinel": true,
		"watch_symbols": ["BTCUSDT", "ETHUSDT"],
		"indicators": {
			"klines": {
				"primary_timeframe": "5m",
				"selected_timeframes": ["5m", "15m", "1h"],
				"primary_count": 200
			}
		},
		"risk_control": {
			"max_margin_usage": 0.8,
			"min_position_size": 1,
			"btc_eth_max_leverage": 3,
			"altcoin_max_leverage": 5
		}
	}`
	if err := st.Strategy().Create(&Strategy{
		ID:       "s-legacy",
		UserID:   userID,
		Name:     "legacy crypto strategy",
		Config:   legacyStrategyJSON,
		IsActive: true,
	}); err != nil {
		t.Fatalf("plant legacy strategy: %v", err)
	}
	loaded, err := st.Strategy().Get(userID, "s-legacy")
	if err != nil {
		t.Fatalf("load legacy strategy: %v", err)
	}
	cfg, err := loaded.ParseConfig()
	if err != nil {
		t.Fatalf("parse legacy strategy config: %v", err)
	}
	// Futures-effective settings unchanged.
	if cfg.Indicators.Klines.PrimaryTimeframe != "5m" ||
		len(cfg.Indicators.Klines.SelectedTimeframes) != 3 ||
		cfg.Indicators.Klines.SelectedTimeframes[0] != "5m" ||
		cfg.Indicators.Klines.PrimaryCount != 200 {
		t.Fatalf("futures klines settings changed by legacy fields: %+v", cfg.Indicators.Klines)
	}
	if cfg.RiskControl.MaxMarginUsage != 0.8 || cfg.RiskControl.MinPositionSize != 1 {
		t.Fatalf("futures risk settings changed by legacy fields: %+v", cfg.RiskControl)
	}

	// 2) EXCHANGE — a pre-existing crypto row with every legacy credential
	// column set, planted at the DB level exactly as a migrated row would be.
	// The columns were dropped from the store.Exchange struct (CTO ruling:
	// columns stay, struct fields go), so AutoMigrate no longer creates them on
	// a fresh fixture — the DDL below stands in for the pre-existing legacy
	// schema (same nullable-with-default shape the CTO verified on the real
	// copy), and the UPDATE stands in for the values a legacy migration wrote.
	legacyDDL := []string{
		`ALTER TABLE exchanges ADD COLUMN hyperliquid_wallet_addr TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN hyperliquid_unified_account BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE exchanges ADD COLUMN aster_user TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN aster_signer TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN aster_private_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN lighter_wallet_addr TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN lighter_private_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN lighter_api_key_private_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE exchanges ADD COLUMN lighter_api_key_index INTEGER NOT NULL DEFAULT 0`,
	}
	for _, ddl := range legacyDDL {
		if err := st.gdb.Exec(ddl).Error; err != nil {
			t.Fatalf("add legacy column (%s): %v", ddl, err)
		}
	}
	if err := st.gdb.Create(&Exchange{
		ID:           "e-legacy",
		UserID:       userID,
		Name:         "legacy binance",
		Type:         "cex",
		ExchangeType: "binance",
		AccountName:  "Default",
		Enabled:      true,
		APIKey:       crypto.EncryptedString("legacy-api-key"),
		SecretKey:    crypto.EncryptedString("legacy-secret"),
		Passphrase:   crypto.EncryptedString(""),
	}).Error; err != nil {
		t.Fatalf("plant legacy exchange row: %v", err)
	}
	if err := st.gdb.Exec(`UPDATE exchanges SET
		hyperliquid_wallet_addr = '0xhyper',
		hyperliquid_unified_account = true,
		aster_user = 'aster-user',
		aster_signer = 'aster-signer',
		aster_private_key = 'enc:aster-priv',
		lighter_wallet_addr = '0xlighter',
		lighter_private_key = 'enc:lighter-priv',
		lighter_api_key_private_key = 'enc:lighter-apik',
		lighter_api_key_index = 7
		WHERE id = ? AND user_id = ?`, "e-legacy", userID).Error; err != nil {
		t.Fatalf("plant legacy column values: %v", err)
	}
	ex, err := st.Exchange().GetByID(userID, "e-legacy")
	if err != nil {
		t.Fatalf("load legacy exchange row: %v", err)
	}
	if ex.ExchangeType != "binance" || !ex.Enabled {
		t.Fatalf("legacy exchange identity changed: type=%q enabled=%v", ex.ExchangeType, ex.Enabled)
	}
	// The crypto credential columns were dropped from the store.Exchange struct
	// (integration, CTO ruling: columns stay, struct fields go). The row loads,
	// and the legacy columns are simply no longer surfaced.
	if len(string(ex.APIKey)) == 0 || len(string(ex.SecretKey)) == 0 {
		t.Fatalf("legacy credential columns did not round-trip (empty after decrypt)")
	}

	// 2b) RAW SQL — the legacy columns must physically hold the planted values
	// (CTO item 2). Read them straight off the DB, not through the struct.
	rawBefore := readLegacyColumns(t, st.gdb, userID, "e-legacy")
	if rawBefore.HyperliquidWalletAddr != "0xhyper" ||
		rawBefore.HyperliquidUnifiedAccount != true ||
		rawBefore.AsterUser != "aster-user" ||
		rawBefore.AsterSigner != "aster-signer" ||
		rawBefore.LighterWalletAddr != "0xlighter" ||
		rawBefore.LighterAPIKeyIndex != 7 {
		t.Fatalf("legacy DB columns wrong at raw-SQL level: %+v", rawBefore)
	}
	for name, v := range map[string]string{
		"aster_private_key":           rawBefore.AsterPrivateKey,
		"lighter_private_key":         rawBefore.LighterPrivateKey,
		"lighter_api_key_private_key": rawBefore.LighterAPIKeyPrivateKey,
	} {
		if len(v) == 0 {
			t.Fatalf("encrypted legacy column %s empty at raw-SQL level", name)
		}
	}

	// 2c) A production-path Update round-tripping the loaded values must leave
	// the legacy columns byte-identical (the store must never wipe or rewrite
	// them behind the caller's back). The crypto positional params were removed
	// from Update (CTO ruling: struct fields go, columns stay) — the call passes
	// only the fields this build still surfaces.
	if err := st.Exchange().Update(userID, "e-legacy",
		ex.Enabled,
		string(ex.APIKey), string(ex.SecretKey), string(ex.Passphrase), ex.Testnet,
		ex.NTDataDir, ex.NTInstrumentName, ex.NTDefaultContractQty,
	); err != nil {
		t.Fatalf("production update of legacy exchange row: %v", err)
	}
	rawAfter := readLegacyColumns(t, st.gdb, userID, "e-legacy")
	if rawAfter.HyperliquidWalletAddr != rawBefore.HyperliquidWalletAddr ||
		rawAfter.HyperliquidUnifiedAccount != rawBefore.HyperliquidUnifiedAccount ||
		rawAfter.AsterUser != rawBefore.AsterUser ||
		rawAfter.AsterSigner != rawBefore.AsterSigner ||
		rawAfter.LighterWalletAddr != rawBefore.LighterWalletAddr ||
		rawAfter.LighterAPIKeyIndex != rawBefore.LighterAPIKeyIndex {
		t.Fatalf("legacy DB columns changed across a store Update:\nbefore: %+v\nafter:  %+v", rawBefore, rawAfter)
	}
	if len(rawAfter.AsterPrivateKey) == 0 || len(rawAfter.LighterPrivateKey) == 0 ||
		len(rawAfter.LighterAPIKeyPrivateKey) == 0 {
		t.Fatalf("encrypted legacy columns emptied by the store Update: %+v", rawAfter)
	}
	if _, err := st.Exchange().GetByID(userID, "e-legacy"); err != nil {
		t.Fatalf("legacy exchange row unloadable after the store Update: %v", err)
	}

	// 3) AI MODEL — the disabled claw402 row (C1: it stays, it never crashes).
	if err := st.AIModel().Create(userID, "m-legacy", "Claw402 legacy", "claw402", false, "0xlegacy", "https://claw402.ai"); err != nil {
		t.Fatalf("plant legacy ai_model row: %v", err)
	}
	model, err := st.AIModel().Get(userID, "m-legacy")
	if err != nil {
		t.Fatalf("load legacy ai_model row: %v", err)
	}
	if model.Provider != "claw402" || model.Enabled {
		t.Fatalf("legacy ai_model changed on load: provider=%q enabled=%v", model.Provider, model.Enabled)
	}

	// 4) REOPEN — initTables + cleanup run again; the strategy, the ai_model AND
	// the unsupported-type exchange row must all survive (the exchange row is a
	// legacy crypto row — cleanup SKIPS it, never deletes it).
	if err := st.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	st2, err := New(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st2.Close()
	loaded2, err := st2.Strategy().Get(userID, "s-legacy")
	if err != nil {
		t.Fatalf("strategy row did not survive the reopen: %v", err)
	}
	if loaded2.Config != legacyStrategyJSON {
		t.Fatalf("strategy config changed across the reopen:\n%q\n%q", loaded2.Config, legacyStrategyJSON)
	}
	if _, err := st2.AIModel().Get(userID, "m-legacy"); err != nil {
		t.Fatalf("ai_model row did not survive the reopen: %v", err)
	}
	ex2, err := st2.Exchange().GetByID(userID, "e-legacy")
	if err != nil {
		t.Fatalf("legacy exchange row did not survive the boot-cleanup pass: %v", err)
	}
	if ex2.ExchangeType != "binance" || !ex2.Enabled {
		t.Fatalf("legacy exchange identity changed across the reopen: %+v", ex2)
	}
	// The crypto columns are no longer struct fields — the raw-SQL read is the
	// surviving assertion; the legacy bytes must be identical across the reopen.
	rawReopen := readLegacyColumns(t, st2.gdb, userID, "e-legacy")
	if rawReopen.HyperliquidWalletAddr != rawBefore.HyperliquidWalletAddr ||
		rawReopen.HyperliquidUnifiedAccount != rawBefore.HyperliquidUnifiedAccount ||
		rawReopen.AsterUser != rawBefore.AsterUser ||
		rawReopen.AsterSigner != rawBefore.AsterSigner ||
		rawReopen.LighterWalletAddr != rawBefore.LighterWalletAddr ||
		rawReopen.LighterAPIKeyIndex != rawBefore.LighterAPIKeyIndex {
		t.Fatalf("legacy DB columns changed across the reopen:\nbefore: %+v\nafter:  %+v", rawBefore, rawReopen)
	}
	if len(rawReopen.AsterPrivateKey) == 0 || len(rawReopen.LighterPrivateKey) == 0 ||
		len(rawReopen.LighterAPIKeyPrivateKey) == 0 {
		t.Fatalf("encrypted legacy columns emptied across the reopen: %+v", rawReopen)
	}
}

// legacyColumns is the raw-SQL shadow of the pre-cut crypto credential columns.
// The GORM struct still maps them (migration tolerance), but C1's data-safety
// proof is at the column level: these physical columns must round-trip a load
// and a production update untouched.
type legacyColumns struct {
	HyperliquidWalletAddr     string
	HyperliquidUnifiedAccount bool
	AsterUser                 string
	AsterSigner               string
	AsterPrivateKey           string
	LighterWalletAddr         string
	LighterPrivateKey         string
	LighterAPIKeyPrivateKey   string
	LighterAPIKeyIndex        int
}

func readLegacyColumns(t *testing.T, db *gorm.DB, userID, id string) legacyColumns {
	t.Helper()
	var c legacyColumns
	if err := db.Raw(`SELECT hyperliquid_wallet_addr, hyperliquid_unified_account,
		aster_user, aster_signer, aster_private_key,
		lighter_wallet_addr, lighter_private_key, lighter_api_key_private_key,
		lighter_api_key_index
		FROM exchanges WHERE id = ? AND user_id = ?`, id, userID).Scan(&c).Error; err != nil {
		t.Fatalf("raw-SQL read of legacy columns: %v", err)
	}
	return c
}

// TestC1RealCopyLoadEveryRow is the CTO's opt-in real-copy mode (the CTO runs
// it at the base and at the final integrated head against a copy of the latest
// backup and diffs the two outputs line-for-line; this lane NEVER runs it
// against owner data). NORMAL CI NEVER SEES OWNER DATA: VL_C1_REAL_DB unset →
// the test SKIPS.
//
// Command line (exact):
//
//	VL_C1_REAL_DB=/path/to/backup.db go test -count=1 -run '^TestC1RealCopyLoadEveryRow$' -v ./store
//
// When set, the test (a) copies the file to t.TempDir() — the given path is
// only ever opened READ-ONLY and data/ is never touched — (b) opens the COPY
// through production store.New (initTables + boot cleanup run on the copy),
// (c) loads EVERY strategy / exchange / ai_model / trader row, (d) prints one
// deterministic line per row (sorted by id within each type, types in fixed
// order), and (e) FAILS if any row errors on load.
func TestC1RealCopyLoadEveryRow(t *testing.T) {
	src := os.Getenv("VL_C1_REAL_DB")
	if src == "" {
		t.Skip("VL_C1_REAL_DB unset — the real-copy mode is opt-in and never runs in normal CI")
	}

	// (a) copy — the source path is opened read-only, never written.
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open VL_C1_REAL_DB source (read-only): %v", err)
	}
	copyPath := filepath.Join(t.TempDir(), "c1-real-copy.db")
	out, err := os.Create(copyPath)
	if err != nil {
		in.Close()
		t.Fatalf("create copy: %v", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		in.Close()
		t.Fatalf("copy db: %v", err)
	}
	out.Close()
	in.Close()

	// (b) open the COPY through the production store (cleanup runs on the copy).
	st, err := New(copyPath)
	if err != nil {
		t.Fatalf("open store on copy: %v", err)
	}
	defer st.Close()

	failures := 0

	// (c)(d)(e) STRATEGY rows — production Get + ParseConfig.
	var strategies []Strategy
	if err := st.gdb.Order("id").Find(&strategies).Error; err != nil {
		t.Fatalf("list strategies: %v", err)
	}
	for _, s := range strategies {
		loaded, err := st.Strategy().Get(s.UserID, s.ID)
		if err != nil {
			failures++
			t.Errorf("STRATEGY %s failed to load: %v", s.ID, err)
			continue
		}
		cfg, err := loaded.ParseConfig()
		if err != nil {
			failures++
			t.Errorf("STRATEGY %s failed to parse: %v", s.ID, err)
			continue
		}
		t.Logf("STRATEGY %s source=%s coins=%s klines_tf=%s klines_count=%d caps_btc_lev=%d caps_alt_lev=%d caps_btc_ratio=%v caps_alt_ratio=%v min_conf=%d max_pos=%d",
			s.ID, cfg.CoinSource.SourceType, strings.Join(cfg.CoinSource.StaticCoins, ","),
			strings.Join(cfg.Indicators.Klines.SelectedTimeframes, ","), cfg.Indicators.Klines.PrimaryCount,
			cfg.RiskControl.BTCETHMaxLeverage, cfg.RiskControl.AltcoinMaxLeverage,
			cfg.RiskControl.BTCETHMaxPositionValueRatio, cfg.RiskControl.AltcoinMaxPositionValueRatio,
			cfg.RiskControl.MinConfidence, cfg.RiskControl.MaxPositions)
	}

	// (c)(d)(e) EXCHANGE rows — production GetByID (decrypts credentials).
	var exchanges []Exchange
	if err := st.gdb.Order("id").Find(&exchanges).Error; err != nil {
		t.Fatalf("list exchanges: %v", err)
	}
	for _, e := range exchanges {
		loaded, err := st.Exchange().GetByID(e.UserID, e.ID)
		if err != nil {
			failures++
			t.Errorf("EXCHANGE %s failed to load: %v", e.ID, err)
			continue
		}
		t.Logf("EXCHANGE %s type=%s enabled=%v nt_dir=%v nt_instr=%v nt_qty=%d",
			loaded.ID, loaded.ExchangeType, loaded.Enabled,
			loaded.NTDataDir != "", loaded.NTInstrumentName != "", loaded.NTDefaultContractQty)
	}

	// (c)(d)(e) AI MODEL rows — production Get.
	var models []AIModel
	if err := st.gdb.Order("id").Find(&models).Error; err != nil {
		t.Fatalf("list ai_models: %v", err)
	}
	for _, m := range models {
		loaded, err := st.AIModel().Get(m.UserID, m.ID)
		if err != nil {
			failures++
			t.Errorf("AI_MODEL %s failed to load: %v", m.ID, err)
			continue
		}
		t.Logf("AI_MODEL %s provider=%s enabled=%v", loaded.ID, loaded.Provider, loaded.Enabled)
	}

	// (c)(d)(e) TRADER rows — production ListAll.
	traders, err := st.Trader().ListAll()
	if err != nil {
		t.Fatalf("list traders: %v", err)
	}
	sort.Slice(traders, func(i, j int) bool { return traders[i].ID < traders[j].ID })
	for _, tr := range traders {
		t.Logf("TRADER %s exchange_id=%s strategy_id=%s is_running=%v", tr.ID, tr.ExchangeID, tr.StrategyID, tr.IsRunning)
	}

	if failures > 0 {
		t.Fatalf("%d row(s) failed to load", failures)
	}
}
