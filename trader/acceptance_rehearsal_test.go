package trader

// ═══════════════════════════════════════════════════════════════════════════
// ACCEPTANCE GATE — DRESS REHEARSAL HARNESS (2026-08-16 gate, Part C).
//
// Runs ONE real planner read on the EXACT HEAD build's code paths against the
// LIVE stored data + the LIVE deployed binary's own bar cache (fetched over
// its /api/klines HTTP endpoint — the same market.FuturesBarsProvider feed the
// in-process read sees). Writes the plan row to the live DB flagged by an
// impossible production trade_date (Saturday 2026-08-15 — every scheduled read
// is IsCMEOpen-gated, so no real row can ever carry this date); the gate
// session then marks it lifecycle='expired' after the UI verification.
//
// GUARDED: skips unless NOFX_REHEARSAL=1. Costs ONE paid model call
// (owner-approved in the gate dispatch). Never touches orders, NT8, or any
// account — the planner read path has no execution surface.
//
// Divergences from the in-process Monday read (each documented in the gate
// report): (1) bars ride HTTP with a 1500-bar cap (the 5m×3000 RV-baseline
// fetch is truncated to 1500); (2) kernel.ActivePlanProvider is installed with
// the session/date pinned to the rehearsal keys (production resolves them from
// the clock, which on a Saturday night resolves to no-session); (3) the clock
// itself — everything else is the production functions, same package.
// ═══════════════════════════════════════════════════════════════════════════

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/crypto"
	"nofx/kernel"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
)

const (
	rehearsalTradeDate = "2026-08-15" // Saturday — impossible for a scheduled read (IsCMEOpen gate) = the flag
	rehearsalSession   = "NY"         // keep the PROMPT Monday-shaped (real session name)
)

// liveBars serves market.Kline slices from the DEPLOYED binary's /api/klines —
// the same BarCache the in-process provider reads. Memoized per (interval,limit)
// so the A7 double-assembly sees identical bars (matching the closed-market
// stability of the real cache).
type liveBars struct {
	baseURL string
	mu      sync.Mutex
	memo    map[string][]market.Kline
	log     []string
}

func (f *liveBars) fetch(symbol, interval string, limit int) []market.Kline {
	req := limit
	if limit > 1500 {
		limit = 1500 // /api/klines HTTP cap; divergence documented
	}
	key := fmt.Sprintf("%s|%s|%d", symbol, interval, limit)
	f.mu.Lock()
	defer f.mu.Unlock()
	if bars, ok := f.memo[key]; ok {
		return bars
	}
	u := fmt.Sprintf("%s/api/klines?exchange=ninjatrader&symbol=%s&interval=%s&limit=%d",
		f.baseURL, url.QueryEscape(symbol), url.QueryEscape(interval), limit)
	resp, err := http.Get(u)
	if err != nil {
		f.log = append(f.log, fmt.Sprintf("%s -> ERR %v", key, err))
		return nil
	}
	defer resp.Body.Close()
	var bars []market.Kline
	if err := json.NewDecoder(resp.Body).Decode(&bars); err != nil {
		f.log = append(f.log, fmt.Sprintf("%s -> decode ERR %v", key, err))
		return nil
	}
	first, last := "-", "-"
	if len(bars) > 0 {
		first = time.UnixMilli(bars[0].OpenTime).UTC().Format("01-02 15:04")
		last = time.UnixMilli(bars[len(bars)-1].CloseTime).UTC().Format("01-02 15:04")
	}
	f.log = append(f.log, fmt.Sprintf("%s (requested %d) -> %d bars [%s .. %s UTC]", key, req, len(bars), first, last))
	f.memo[key] = bars
	return bars
}

func sha12(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

func writeArtifact(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write artifact %s: %v", name, err)
	}
	t.Logf("artifact %s (%d bytes, sha %s)", name, len(content), sha12(content))
}

// TestAcceptanceRehearsal executes the Part C dress rehearsal. Read the file
// header before running: it spends ONE paid model call and writes the flagged
// plan row to the live DB.
func TestAcceptanceRehearsal(t *testing.T) {
	if os.Getenv("NOFX_REHEARSAL") != "1" {
		t.Skip("rehearsal harness is armed only with NOFX_REHEARSAL=1 (paid call + live-db plan row)")
	}
	dbPath := os.Getenv("NOFX_REHEARSAL_DB")
	if dbPath == "" {
		dbPath = filepath.Join("..", "data", "data.db")
	}
	outDir := os.Getenv("NOFX_REHEARSAL_OUT")
	if outDir == "" {
		t.Fatal("NOFX_REHEARSAL_OUT must point at the artifacts directory")
	}
	baseURL := os.Getenv("NOFX_REHEARSAL_BASE")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	traderID := os.Getenv("NOFX_REHEARSAL_TRADER")
	if traderID == "" {
		t.Fatal("NOFX_REHEARSAL_TRADER must be the live trader id")
	}

	// ── crypto service FIRST: crypto.EncryptedString.Scan only decrypts when the
	// global service is installed (crypto.go:433). main.go does this at boot
	// (main.go:44-48); without it every api_key column reads back as raw "ENC:…"
	// ciphertext and every provider call 401s. Needs DATA_ENCRYPTION_KEY exported.
	cs, err := crypto.NewCryptoService()
	if err != nil {
		t.Fatalf("crypto service (is DATA_ENCRYPTION_KEY exported?): %v", err)
	}
	crypto.SetGlobalCryptoService(cs)

	// ── live store (the deployed binary's own DB; sqlite cross-process) ──
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("open live store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	// ── hydrate the trader EXACTLY like manager.addTraderFromStore ──
	traderCfg, err := st.Trader().GetByID(traderID)
	if err != nil || traderCfg == nil {
		t.Fatalf("trader row %s: %v", traderID, err)
	}
	strategy, err := st.Strategy().Get(traderCfg.UserID, traderCfg.StrategyID)
	if err != nil {
		t.Fatalf("strategy row: %v", err)
	}
	strategyConfig, err := strategy.ParseConfig()
	if err != nil {
		t.Fatalf("strategy config parse: %v", err)
	}
	aiModelCfg, err := st.AIModel().GetByID(traderCfg.AIModelID)
	if err != nil {
		t.Fatalf("ai model row: %v", err)
	}
	exchangeCfg, err := st.Exchange().GetByID(traderCfg.UserID, traderCfg.ExchangeID)
	if err != nil {
		t.Fatalf("exchange row: %v", err)
	}
	if exchangeCfg.ExchangeType != "ninjatrader" {
		t.Fatalf("rehearsal requires the ninjatrader trader, got %s", exchangeCfg.ExchangeType)
	}
	if dp := strategyConfig.DayPlan; dp == nil || !dp.PlanEnabled {
		t.Fatal("day_plan not enabled on the bound strategy — Monday would not fire (STOP: gate finding)")
	}

	cfg := AutoTraderConfig{
		ID: traderCfg.ID, Name: traderCfg.Name,
		AIModel:  aiModelCfg.Provider,
		Exchange: exchangeCfg.ExchangeType, ExchangeID: exchangeCfg.ID,
		UseQwen:         aiModelCfg.Provider == "qwen",
		CustomAPIURL:    aiModelCfg.CustomAPIURL,
		CustomModelName: aiModelCfg.CustomModelName,
		InitialBalance:  traderCfg.InitialBalance,
		StrategyConfig:  strategyConfig,
	}
	switch aiModelCfg.Provider {
	case "qwen":
		cfg.QwenKey = string(aiModelCfg.APIKey)
	case "deepseek":
		cfg.DeepSeekKey = string(aiModelCfg.APIKey)
	default:
		cfg.CustomAPIKey = string(aiModelCfg.APIKey)
	}
	if exchangeCfg.NTInstrumentName != "" {
		cfg.NinjaTraderSymbol = exchangeCfg.NTInstrumentName
	}
	if cfg.DeepSeekKey == "" && cfg.QwenKey == "" && cfg.CustomAPIKey == "" {
		t.Fatal("no decrypted API key — is DATA_ENCRYPTION_KEY exported? (EncryptedString scan)")
	}

	// mcp client exactly like NewAutoTrader (registry + provider key resolution)
	client := mcp.NewAIClientByProvider(cfg.AIModel)
	if client == nil {
		t.Fatalf("no mcp client for provider %q", cfg.AIModel)
	}
	apiKey := cfg.CustomAPIKey
	switch cfg.AIModel {
	case "qwen":
		if cfg.QwenKey != "" {
			apiKey = cfg.QwenKey
		}
	case "deepseek":
		if cfg.DeepSeekKey != "" {
			apiKey = cfg.DeepSeekKey
		}
	}
	client.SetAPIKey(apiKey, cfg.CustomAPIURL, cfg.CustomModelName)

	// ── live bars over the deployed binary's HTTP feed ──
	bars := &liveBars{baseURL: baseURL, memo: map[string][]market.Kline{}}
	prevProvider := market.FuturesBarsProvider
	market.FuturesBarsProvider = bars.fetch
	t.Cleanup(func() { market.FuturesBarsProvider = prevProvider })

	// production provider installs (same functions the live trader runs once)
	installNakedPOCProvider(st)
	installLevelStateProvider(st)

	at := &AutoTrader{
		id: cfg.ID, name: cfg.Name, aiModel: cfg.AIModel,
		exchange: cfg.Exchange, config: cfg, store: st, mcpClient: client,
	}

	// ═══ A6 — seed a STICKY OWNER LEVEL, armored the way the API armors it ═══
	// (POST /api/plan/owner-level is JWT-gated; this seeds through the SAME store
	// the handler writes, after running the handler's own kernel.LevelPriceViolation
	// armor, then deletes it in cleanup. Documented divergence: the HTTP route
	// itself is not exercised.)
	symbolForSeed := at.futuresSymbol()
	seedBars := bars.fetch(symbolForSeed, kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(seedBars) == 0 {
		t.Fatal("no live 1m bars from the deployed binary — cannot seed A6 or assemble (is NT8/BarCache cold?)")
	}
	_, seedPrice, seedDATR := kernel.AssembleScoredLevels(seedBars, kernel.DefaultSessionRegistry(), symbolForSeed, 8, time.Now(), kernel.ActivationWindowK)
	seedLevelPrice := float64(int(seedPrice/25.0)) * 25.0 // a round-ish level near price
	if reason, bad := kernel.LevelPriceViolation(seedLevelPrice, seedPrice, seedDATR); bad {
		t.Fatalf("A6 seed price %.2f rejected by the same armor the API applies: %s", seedLevelPrice, reason)
	}
	seedLvl := &store.OwnerLevelDB{
		Symbol: symbolForSeed, Price: seedLevelPrice,
		Label: "GATE REHEARSAL", Note: "acceptance gate A6 — delete me",
		ScenarioTag: "S1", CreatedAt: time.Now().Unix(),
	}
	if err := st.OwnerLevel().Save(seedLvl); err != nil {
		t.Fatalf("A6 seed save: %v", err)
	}
	t.Logf("A6 seeded owner level id=%d price=%.2f (armor OK vs price %.2f dATR %.2f)", seedLvl.ID, seedLevelPrice, seedPrice, seedDATR)
	t.Cleanup(func() {
		if err := st.OwnerLevel().Delete(seedLvl.ID); err != nil {
			t.Errorf("A6 CLEANUP FAILED — owner level %d still in the live DB: %v", seedLvl.ID, err)
		} else {
			t.Logf("A6 cleanup: owner level %d deleted", seedLvl.ID)
		}
	})

	// ═══ ① ASSEMBLE THE PLANNER INPUT (production function, live data) ═══
	input := at.assemblePlannerInput(rehearsalSession, rehearsalTradeDate)
	prompt := kernel.BuildPlannerPrompt(input)
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	writeArtifact(t, outDir, "artifact-1-planner-input.txt",
		"SYSTEM:\n"+plannerSystemPrompt+"\n\nUSER:\n"+prompt)
	writeArtifact(t, outDir, "artifact-1b-planner-input-struct.json", string(inputJSON))

	// A6 receipt: the seeded sticky owner level must ride in tagged 👤
	if !strings.Contains(prompt, "👤") {
		t.Error("A6 FAIL: no 👤 owner level in the assembled planner input")
	} else {
		t.Log("A6 PASS: 👤 owner level present in planner input")
	}

	// save the exact 1m bar set for the independent A1 recompute
	oneMin := bars.fetch(at.futuresSymbol(), kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if b, err := json.Marshal(oneMin); err == nil {
		writeArtifact(t, outDir, "bars-1m.json", string(b))
	}

	// ═══ ② THE ONE PAID CALL through the production read core ═══
	modelClient, modelID := at.resolvePlannerClient()
	hash := shortHash(prompt)
	t1Lines := kernel.T1NoTradeLines(input.Calendar)
	var rawResponses []string
	version, lifecycle, err := at.runPlannerReadCore(
		rehearsalSession, rehearsalTradeDate, modelID, hash,
		input.IndicatorsBlock, input.AIConfigHash,
		func() (string, error) {
			raw, callErr := modelClient.CallWithMessages(plannerSystemPrompt, prompt)
			rawResponses = append(rawResponses, fmt.Sprintf("── attempt %d (err=%v) ──\n%s", len(rawResponses)+1, callErr, raw))
			return raw, callErr
		}, t1Lines...)
	writeArtifact(t, outDir, "artifact-2-plan-response.json", strings.Join(rawResponses, "\n\n"))
	t.Logf("read core: version=%d lifecycle=%s err=%v attempts=%d model=%s prompt_hash=%s",
		version, lifecycle, err, len(rawResponses), modelID, hash)
	if err != nil {
		t.Fatalf("plan row write failed: %v", err)
	}
	if lifecycle != "active" {
		t.Errorf("REHEARSAL FINDING: lifecycle=%q (fail-closed path taken after %d attempts)", lifecycle, len(rawResponses))
	}

	// read back the stored row — the W11 freeze + W2 pin receipts
	row, err := st.Plan().GetLatestPlanForSession(rehearsalTradeDate, rehearsalSession)
	if err != nil || row == nil {
		t.Fatalf("stored plan row read-back: %v", err)
	}
	rowJSON, _ := json.MarshalIndent(row, "", "  ")
	writeArtifact(t, outDir, "artifact-2b-plan-row.json", string(rowJSON))
	if row.ModelID == "" || mcp.IsProviderAlias(row.ModelID) {
		t.Errorf("W2 FAIL: plan row model_id %q is empty or a provider alias", row.ModelID)
	}
	if row.PromptHash != hash {
		t.Errorf("prompt_hash mismatch: row %s vs computed %s", row.PromptHash, hash)
	}
	if row.IndicatorsBlock != input.IndicatorsBlock {
		t.Error("W11 FAIL: indicators_block on the row is not the frozen assembled block")
	}

	// ═══ ④ + A7: EXECUTOR PROMPT DOUBLE-ASSEMBLY carrying the plan ═══
	// Pinned ActivePlanProvider: the production body (installActivePlanProvider)
	// with session/date fixed to the rehearsal keys — clock is the only divergence.
	kernel.SetTraderPlanProviders(cfg.ID, kernel.TraderPlanProviders{ActivePlan: func(symbol string) *kernel.ActivePlan {
		r, e := st.Plan().GetLatestPlanForTraderSession(rehearsalTradeDate, rehearsalSession, cfg.ID)
		if e != nil || r == nil || r.Lifecycle != "active" {
			return nil
		}
		doc, ok := resolveActivePlanDoc(st, r)
		if !ok {
			return nil
		}
		replansLeft := 2 - (r.Version - 1)
		if replansLeft < 0 {
			replansLeft = 0
		}
		return &kernel.ActivePlan{Doc: doc, Session: rehearsalSession, Version: r.Version, ReplansLeft: replansLeft}
	}})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(cfg.ID, kernel.TraderPlanProviders{}) })

	buildExecutorPrompt := func() string {
		// verbatim replica of the engine_analysis.go futures context compute
		engine := kernel.NewStrategyEngine(strategyConfig)
		symbol := at.futuresSymbol()
		variant := resolvePromptVariant(at.exchange, strategyConfig.PromptVariant)
		engine.SetSVPContext("")
		if strategyConfig.Indicators.EnableSVP {
			svpLine := ""
			if b := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount); len(b) > 0 {
				svpLine = kernel.FormatSVPLine(kernel.BuildSVPProfile(b, time.Now()))
			}
			engine.SetSVPContext(svpLine)
		}
		engine.SetKeyLevelsContext("")
		maxLevels := kernel.DefaultMaxLevels
		if dp := strategyConfig.DayPlan; dp != nil && dp.MaxLevels > 0 {
			maxLevels = dp.MaxLevels
		}
		if b := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount); len(b) > 0 {
			var extra []kernel.DetectedLevel
			if kernel.NakedPOCProvider != nil {
				extra = kernel.NakedPOCProvider(symbol)
			}
			engine.SetKeyLevelsContext(kernel.BuildKeyLevelsBlock(b, kernel.DefaultSessionRegistry(), symbol, maxLevels, time.Now(), kernel.ActivationWindowK, extra...))
		}
		engine.SetPlanContext("", "")
		if plan := kernel.ActivePlanFor(cfg.ID, symbol); plan != nil {
			rule := ""
			if dp := strategyConfig.DayPlan; dp != nil {
				rule = dp.AcceptanceRule
			}
			block := kernel.RenderPlanBlock(plan.Doc, plan.Session)
			status := ""
			if b := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount); len(b) > 0 {
				_, price, dATR := kernel.AssembleScoredLevels(b, kernel.DefaultSessionRegistry(), symbol, maxLevels, time.Now(), kernel.ActivationWindowK)
				status = kernel.RenderPlanStatus(symbol, plan.Doc, b, price, dATR, rule, plan.ReplansLeft, time.Now().UnixMilli())
			}
			engine.SetPlanContext(block, status)
		}
		return engine.BuildSystemPrompt(traderCfg.InitialBalance, variant, symbol)
	}

	exec1 := buildExecutorPrompt()
	time.Sleep(2 * time.Second)
	exec2 := buildExecutorPrompt()
	writeArtifact(t, outDir, "artifact-4-executor-prompt.txt", exec1)
	writeArtifact(t, outDir, "artifact-4b-executor-prompt-2.txt", exec2)

	// A7a — the PLAN BLOCK prefix must be byte-identical across assemblies.
	planBlock := func(p string) string {
		i := strings.Index(p, "# DAY PLAN")
		j := strings.Index(p, "# PLAN STATUS")
		if i < 0 {
			return ""
		}
		if j < 0 || j < i {
			j = len(p)
		}
		// prefix region: plan block through to the status tail
		k := strings.Index(p[i:], "# Available Data")
		if k > 0 {
			return p[i : i+k]
		}
		return p[i:j]
	}
	pb1, pb2 := planBlock(exec1), planBlock(exec2)
	if pb1 == "" {
		t.Error("A7 FAIL: no PLAN BLOCK in the executor prompt")
	}
	if pb1 != pb2 {
		t.Error("A7 FAIL: PLAN BLOCK differs across consecutive assemblies")
	} else {
		t.Logf("A7a PASS: PLAN BLOCK byte-identical across assemblies (sha %s, %d bytes)", sha12(pb1), len(pb1))
	}
	if exec1 != exec2 {
		t.Logf("A7b NOTE: full prompts differ outside the plan block (expected only in STATUS facts; see diff artifact)")
	} else {
		t.Log("A7b: full prompts byte-identical (closed market — no fact drift)")
	}

	// A7c — plan JSON → RenderPlanBlock lossless: every doc field value appears.
	var doc kernel.PlanDoc
	_ = json.Unmarshal([]byte(row.Doc), &doc)
	var missing []string
	check := func(label, needle string) {
		if needle != "" && !strings.Contains(pb1, needle) {
			missing = append(missing, label+"="+needle)
		}
	}
	check("bias.direction", doc.Bias.Direction)
	check("bias.conviction", doc.Bias.Conviction)
	check("bias.flip", doc.Bias.FlipCondition)
	check("death", doc.DeathCondition)
	check("day_type", doc.DayType)
	for _, l := range doc.Levels {
		check("level.label", l.Label)
		check("level.price", strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", l.Price), "0"), "."))
		check("level.instruction", l.Instruction)
	}
	for _, s := range doc.Scenarios {
		check("scenario.id", s.ID)
		check("scenario.trigger", s.Trigger)
		check("scenario.invalid", s.Invalid)
	}
	for _, nt := range doc.NoTrade {
		check("no_trade", nt)
	}
	if len(missing) > 0 {
		t.Errorf("A7c LOSSY: %d plan fields not carried into the PLAN BLOCK: %v", len(missing), missing)
	} else {
		t.Logf("A7c PASS: all %d levels / %d scenarios / %d no-trade lines + bias/death/day_type carried into the block",
			len(doc.Levels), len(doc.Scenarios), len(doc.NoTrade))
	}

	// A3 — indicator mirror: the planner block is rendered by the SAME
	// FormatIndicatorState the executor user-prompt uses (engine_prompt.go:731);
	// re-render from the same live data and require byte-equality.
	mirror2, hash2 := at.renderIndicatorMirror(at.futuresSymbol())
	if mirror2 != input.IndicatorsBlock {
		t.Error("A3 FAIL: re-rendered indicator mirror differs from the assembled block (same data, same toggles)")
	} else {
		t.Logf("A3 PASS: indicator mirror stable + frozen on row (ai_config_hash %s)", hash2)
	}

	// bars manifest + summary
	sort.Strings(bars.log)
	writeArtifact(t, outDir, "bars-manifest.txt", strings.Join(bars.log, "\n"))
	summary := map[string]any{
		"trade_date": rehearsalTradeDate, "session": rehearsalSession,
		"version": version, "lifecycle": lifecycle, "model_id": row.ModelID,
		"prompt_hash": hash, "ai_config_hash": input.AIConfigHash,
		"attempts": len(rawResponses), "plan_block_sha": sha12(pb1),
		"executor_prompts_identical": exec1 == exec2,
		"planner_prompt_bytes":       len(prompt), "executor_prompt_bytes": len(exec1),
	}
	sj, _ := json.MarshalIndent(summary, "", "  ")
	writeArtifact(t, outDir, "summary.json", string(sj))
}
