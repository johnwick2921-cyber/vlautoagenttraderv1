# Backend Layer Inventory

Cross-cutting backend code paths that don't belong to one specific page. Page-specific handlers (and the routes that invoke them) are documented in the per-page files; this file covers the system entrypoint, the broker switch, the engine, the data providers, the agent runtime, and the store layer schema map.

## Quick file inventory

| Subsystem | Files | Total LOC |
|---|---|---|
| main / bootstrap | `main.go` | 206 |
| api routes | `api/server.go` + `api/agent_routes.go` + `api/handler_*.go` × 19 | ~3000+ |
| trader manager | `manager/trader_manager.go` | 782 |
| auto_trader scan loop | `trader/auto_trader.go` | 723 |
| kernel decision engine | `kernel/engine.go` + supporting kernel/*.go | ~924 + many |
| Databento provider | `provider/databento/` (8 files) | 832 |
| NinjaTrader bridge | `provider/ninjatrader/` (11 files) | 1916 |
| NinjaTrader broker | `trader/ninjatrader/` (8 files) | 800 |
| agent runtime | `agent/` (60+ files) | ~very large |
| store layer | `store/` (22 files) | ~very large |

## Layer 1: Source code map

### Bootstrap — `main.go` (206 LOC)

[main.go](main.go) is the single entrypoint.

**Order of operations:**

1. **Line 28** — `godotenv.Load()` reads `.env` (silent if missing)
2. **Line 31** — `logger.Init(nil)` — initialize logger
3. **Line 38-40** — `config.Init()` + `config.Get()` — load env-driven config
4. **Line 43-49** — `crypto.NewCryptoService()` + `crypto.SetGlobalCryptoService(cs)` — encryption MUST be initialized BEFORE database (because `EncryptedString` decrypts on read)
5. **Line 53-83** — DB init (`store.NewWithConfig(...)`) supporting both SQLite and Postgres. Argv override for SQLite path (legacy). `defer st.Close()`.
6. **Line 86** — `initInstallationID(st)` — anonymous UUID for telemetry
7. **Line 89-90** — `auth.SetJWTSecret(cfg.JWTSecret)` + the misleading `🔑 JWT secret configured` log (fires UNCONDITIONALLY per CLAUDE.md — does NOT prove the default was overridden)
8. **Line 101** — `traderManager := manager.NewTraderManager()`
9. **Line 104** — `traderManager.LoadTradersFromStore(st)` — **may auto-start traders with IsRunning=true** (this is what restarts the configured NT trader on a fresh bot launch)
10. **Line 109-130** — Log loaded trader configurations
11. **Line 139** — `server := api.NewServer(traderManager, st, cryptoService, cfg.APIServerPort)`
12. **Line 143-144** — Wire telegram reload channel into the API server
13. **Line 147-150** — Construct + register the NOFXi agent. `nofxiAgent.Start()` starts the background goroutines.
14. **Line 153-157** — `go server.Start()` — gin HTTP server in a goroutine
15. **Line 160** — `go telegram.Start(cfg, st, telegramReloadCh)` — Telegram bot
16. **Line 162-181** — wait on `SIGINT`/`SIGTERM`, then graceful shutdown: `server.Shutdown()` → `nofxiAgent.Stop()` (via defer) → `traderManager.StopAll()`

**Critical security gate (per CLAUDE.md):** `config/config.go:67-69` defaults `JWTSecret = "default-jwt-secret-change-in-production"` when env unset. Verify with `grep JWT_SECRET .env`.

### `api/server.go` (657 LOC)

[api/server.go:78-…](api/server.go#L78) registers ~80 HTTP routes via the helper `s.route` / `s.routeWithSchema`. Route groups:

1. **Public unauth (no JWT):**
   - `GET /metrics` (Prometheus) — line 81
   - `GET/POST /api/health`
   - `GET /api/supported-models`, `GET /api/supported-exchanges`
   - `GET /api/config`
   - `POST /api/wallet/validate`, `POST /api/wallet/generate`
   - `GET /api/crypto/config`, `GET /api/crypto/public-key`, `POST /api/crypto/decrypt`
   - `GET /api/traders`, `GET /api/competition`, `GET /api/top-traders`
   - `GET /api/equity-history`, `POST /api/equity-history-batch`
   - `GET /api/traders/:id/public-config`
   - `GET /api/klines`, `GET /api/symbols`
   - `GET /api/strategies/public`
   - `POST /api/strategies/estimate-tokens`
   - `POST /api/register`, `POST /api/login`, `POST /api/reset-password`, `POST /api/reset-account`
2. **Authenticated (`protected := api.Group("/", s.authMiddleware())`):**
   - User: `POST /api/logout`, `PUT /api/user/password`, `GET /api/server-ip`
   - Onboarding: `POST /api/onboarding/beginner`, `GET /api/onboarding/beginner/current`
   - Agent preferences: `GET /api/agent/preferences`, `POST /api/agent/preferences`, `DELETE /api/agent/preferences/:id`
   - Traders: `GET /api/my-traders`, `GET /api/traders/:id/config`, `POST/PUT/DELETE /api/traders`, `POST /api/traders/:id/start`, `POST /api/traders/:id/stop`, `PUT /api/traders/:id/prompt`, `POST /api/traders/:id/sync-balance`, `POST /api/traders/:id/close-position`, `PUT /api/traders/:id/competition`, `GET /api/traders/:id/grid-risk`
   - AI cost: `GET /api/ai-costs`, `GET /api/ai-costs/summary`
   - Models: `GET/PUT /api/models`
   - Exchanges: `GET /api/exchanges`, `GET /api/exchanges/account-state`, `POST/PUT /api/exchanges`, `DELETE /api/exchanges/:id`
   - Telegram: `GET/POST /api/telegram`, `POST /api/telegram/model`, `DELETE /api/telegram/binding`
   - Strategies: `GET /api/strategies`, `GET /api/strategies/active`, `GET /api/strategies/default-config`, `POST /api/strategies/preview-prompt`, `POST /api/strategies/test-run`, `GET /api/strategies/:id`, `POST/PUT/DELETE /api/strategies/:id`, `POST /api/strategies/:id/activate`, `POST /api/strategies/:id/duplicate`
   - Per-trader data (query `?trader_id=`): `GET /api/status`, `/account`, `/positions`, `/positions/history`, `/trades`, `/orders`, `/orders/:id/fills`, `/open-orders`, `/decisions`, `/decisions/latest`, `/statistics`
   - **Plan 4 Task 23:** `POST /api/risk/force-flat`, `GET /api/risk/status`, `GET /api/audit/decisions`
3. **Agent routes (registered separately in `api/agent_routes.go`):**
   - `POST /api/agent/chat`, `POST /api/agent/chat/stream` (auth)
   - `GET /api/agent/health`, `/klines`, `/ticker`, `/tickers` (public)

`s.route(...)` adds schema documentation via the `routes_schema` map (used by the agent to know how to call each endpoint).

### TraderManager — `manager/trader_manager.go` (782 LOC)

Owns the runtime trader lifecycle:

- **`NewTraderManager()`** — constructor
- **`LoadTradersFromStore(st)`** — called once at boot; iterates `store.Trader().List("default")` and calls `addTraderFromStore(...)` for each
- **`addTraderFromStore(trader)`** — the big switch on `exchangeCfg.ExchangeType`. Cases for `binance`, `bybit`, `okx`, `bitget`, `gate`, `kucoin`, `indodax`, `hyperliquid`, `aster`, `lighter`, and **`ninjatrader`** ([line ~700 per CLAUDE.md](manager/trader_manager.go#L700)). Each case constructs the broker, then builds an `AutoTraderConfig` and creates the `AutoTrader`.
- **Per-trader lifecycle:** Start / Stop / Restart goroutines per AutoTrader's scan loop. Each goroutine respects `ctx.Done()` for clean shutdown.
- **`StopAll()`** — called on SIGINT/SIGTERM; iterates and signals every trader to stop.

### AutoTrader scan loop — `trader/auto_trader.go` (723 LOC)

The per-trader long-running goroutine:

- Holds `Trader` (broker), `StrategyEngine`, `AIClient`, `AutoTraderConfig`
- **Main loop** at the broker switch ([line 263-315](trader/auto_trader.go#L263-L315)) constructs the broker by `config.Exchange`:
  - Each case calls a package's constructor (e.g. `binance.New(...)`, `ntTrader.NewTraderFromEnv(cfg)` for ninjatrader)
  - **NinjaTrader special:** uses `NewTraderFromEnv(cfg)` instead of direct `New(cfg)` — this is the env-var router that picks CSV vs TCP transport per `NT_TRANSPORT` env
- **Scan loop:** sleeps `scan_interval_minutes`, calls `kernel.StrategyEngine.Run()`, processes the returned decisions, calls broker methods.
- **Wire-format for the kernel:** trader sends a `kernel.Context` (positions / account / etc.) and gets back a `[]kernel.Decision`.

### Kernel — `kernel/engine.go` (924 LOC)

Imports: `context`, `encoding/json`, `nofx/config`, `nofx/logger`, `nofx/market`, `nofx/provider/databento`, `nofx/provider/hyperliquid`, `nofx/provider/nofxos`, `nofx/security`, `nofx/store`. Note the unused `provider/nofxos` import — `nofxos.ai` is deprecated per CLAUDE.md.

Major types ([line 25-150ish](kernel/engine.go#L25)):
- `PositionInfo` — 11-field position struct (Symbol/Side/EntryPrice/MarkPrice/Quantity/Leverage/UnrealizedPnL/UnrealizedPnLPct/PeakPnLPct/LiquidationPrice/MarginUsed/UpdateTime)
- `AccountInfo` — TotalEquity / AvailableBalance / UnrealizedPnL / TotalPnL / TotalPnLPct / MarginUsed / MarginUsedPct / PositionCount
- `CandidateCoin` — Symbol + Sources[] (e.g. ["ai500","oi_top"])
- `OITopData`, `TradingStats`, `RecentOrder`

Companion files in `kernel/`:
- `engine_prompt.go` — `BuildSystemPrompt(variant)` dispatcher
- `engine_prompt_futures.go` — `BuildFuturesSystemPrompt` / `BuildFuturesUserPrompt`
- `engine_position.go` — position math
- `engine_analysis.go` — fetches OHLCV + indicators, packages for prompt
- `cme_calendar.go` — CME session gates (ADR-006)
- `risk_limits.go` — Plan 3 hard limits + kill switch
- `engine_prompt_golden_test.go` — golden-file tests (ADR-007 locked)

### Databento provider — `provider/databento/`

| File | LOC | Purpose |
|---|---|---|
| `client.go` | 154 | HTTP Basic auth client + Plan 4 Task 24 circuit breaker |
| `historical.go` | 133 | OHLCV bar fetch (`GetOHLCV(symbol, interval, from, to)`) |
| `resolve.go` | 56 | Continuous symbol (`NQ.c.0`) → specific contract (`NQM6`) |
| `contract_calendar.go` | 95 | CME contract month codes (H/M/U/Z) + roll-day arithmetic |
| `mock_server.go` | 62 | httptest-backed mock (Plan 5 Task 26) |
| Tests | 332 | Unit + integration |

**Top constants** at [client.go:23-30](provider/databento/client.go#L23-L30):
- `DefaultHistoricalBaseURL = "https://hist.databento.com/v0"`
- `DefaultTimeout = 30 * time.Second`
- `DefaultDataset = "GLBX.MDP3"` — CME Globex

**Auth model** (per package comment): HTTP Basic with API key as username, empty password.

**ADR-007 locked:** `historical.go`, `resolve.go`, `contract_calendar.go`, `client.go`, `mock_server.go` are byte-stable across PRs since `v1.0-plan1` / `v1.0-plan5`.

### NinjaTrader provider (CSV bridge + TCP server) — `provider/ninjatrader/`

| File | LOC | Purpose |
|---|---|---|
| `types.go` | 59 | Shared `SignalRow` / `FillPayload` types — **ADR-007 locked** |
| `csv_writer.go` | 74 | Plan 1 atomic temp+rename writer — **ADR-007 locked** |
| `csv_writer_test.go` | 75 | Unit tests |
| `csv_tailer.go` | 110 | Plan 1 fills file polling tailer — **ADR-007 locked** |
| `csv_tailer_test.go` | 70 | Unit tests |
| `mock_nt.go` | 155 | Plan 5 mock for integration tests — **ADR-007 locked** |
| `mock_nt_test.go` | 182 | Mock validation |
| `tcp_framing.go` | 130 | Plan 1.5 4-byte BE length-prefix + JSON envelope codec |
| `tcp_framing_test.go` | 186 | Wire-protocol tests |
| `tcp_server.go` | 396 | Plan 1.5 TCP listener on 127.0.0.1:36974 |
| `tcp_server_test.go` | 241 | Lifecycle + reconnect tests |
| `tcp_client_mock.go` | 212 | In-process mock client for integration tests |

**Plan 1.5 TCP server constants** ([tcp_server.go:27-58](provider/ninjatrader/tcp_server.go#L27-L58)):
- `TCPListenAddr = "127.0.0.1:36974"` (NOT NT's ATI port 36973)
- `TCPHeartbeatInterval = 30 * time.Second`
- `TCPHeartbeatAckTimeout = 60 * time.Second` — close conn if no ack in 60s
- `TCPStaleSignalAge = 60 * time.Second` — drop signals older than this on reconnect
- `TCPMaxFrameBytes = 1 << 20` (1 MB max frame)
- `fillChannelBuffer = 32`
- `acceptLoopPollInterval = 250 * time.Millisecond`

**TCPServer struct** has explicit `writeMu` (Plan 1.5.6) — serializes all writes to the connected client to prevent heartbeat + fill ack frame corruption. The fix added in commit `3e4ee61c` after the audit noted concurrent-write races.

### NinjaTrader broker — `trader/ninjatrader/`

| File | LOC | Purpose |
|---|---|---|
| `trader.go` | 230 | Plan 1 CSV-backed `Trader` impl — **ADR-007 locked** |
| `trader_test.go` | 41 | Compile-time + smoke test |
| `tcp_trader.go` | 250 | Plan 1.5 TCP-backed `Trader` impl |
| `transport.go` | 71 | Env-var router (`NT_TRANSPORT=csv|tcp`) — picks which Trader to construct |
| `tick_rounding.go` | 39 | NQ tick = 0.25 rounding helpers — **ADR-007 locked** |
| `tick_rounding_test.go` | 63 | Plan 2 Task 17 |
| `integration_test.go` | 106 | Plan 5 end-to-end with mock NT |

**Per [trader/ninjatrader/CLAUDE.md](trader/ninjatrader/CLAUDE.md):**
- 19-method `types.Trader` interface compliance via compile-time check `var _ types.Trader = (*Trader)(nil)`
- Method support matrix: 11 clean, 3 noop/empty, 5 error-returning
- **Intentional limitations (Plan 1.5 lifts these):** GetBalance = $50k mock, manual close not supported, no cancel/modify

**Transport router** ([transport.go:57-71](trader/ninjatrader/transport.go#L57-L71)):
```go
switch transport {
case "", "csv":  return New(cfg), nil           // Plan 1 SIM-validated
case "tcp":      return NewTCPTrader(...)        // Plan 1.5 opt-in
default:         return nil, fmt.Errorf(...)     // fail-fast on typos
}
```

Currently the running bot uses `NT_TRANSPORT=tcp`. The TCP server is a process-singleton ([transport.go:31-50](trader/ninjatrader/transport.go#L31-L50)) — `sync.Once` ensures repeated `NewTraderFromEnv` calls (e.g. on API-triggered reload) don't try to re-bind the port.

### Agent runtime — `agent/`

60+ files in this package. Key entrypoints:

- `agent/web.go` — HTTP handlers (HandleChat, HandleChatStream, HandleHealth, HandleKlines, HandleTicker, HandleTickers, plus SSE helpers `writeSSE` / `sseEscape`)
- `agent/agent.go` — `Agent` struct, `New(traderManager, st, …, slog)`, `Start()`, `Stop()`
- `agent/tools.go` — **23 LLM tools** (full list in Page 2 doc Layer 3)
- `agent/brain.go` + `central_brain.go` — main planning loop
- `agent/planner_runtime.go` + `skill_dag_runtime.go` — execution graph
- `agent/skill_*` files (~12) — the skill system (DAG, registry, dispatcher, runner, executor, outcome, semantic-gate, etc.)
- `agent/memory.go` + `reference_memory.go` — conversation memory
- `agent/preferences.go` — user prefs CRUD
- `agent/model_provider_catalog.go` — known AI provider templates
- `agent/i18n.go` — agent-side i18n (separate from frontend `web/src/i18n/`)
- `agent/onboard.go` — beginner-mode walkthrough

**Session policy** is injected per-request via context ([api/agent_routes.go:13-36](api/agent_routes.go#L13-L36)):
```go
agent.SessionPolicy{
    Authenticated: true,
    IsAdmin: <user_id == "admin">,
    CanExecuteTrade: true,
    CanViewSensitiveSecrets: false,
}
```

`CanExecuteTrade` gates the `execute_trade` tool. `CanViewSensitiveSecrets` is currently `false` for all users — the agent never sees raw API keys.

### Store layer — `store/` (22 files)

GORM-backed (`store/gorm.go`) with both SQLite (`data/data.db`) and Postgres support. Each subfile is a thin wrapper exposing CRUD around one model:

| File | Model | Purpose |
|---|---|---|
| `store.go` | — | `Store` aggregate + DB pointer |
| `driver.go` | — | DB driver selection |
| `gorm.go` | — | GORM init, AutoMigrate |
| `user.go` | `User` | accounts, password hash, JWT meta |
| `exchange.go` | `Exchange` | crypto + NT broker configs (includes NT-specific `NTDataDir`, `NTInstrumentName`, `NTDefaultContractQty`) |
| `ai_model.go` | `AIModel` | AI provider credentials |
| `ai_charge.go` | `AICharge` | per-call cost tracking |
| `strategy.go` + `strategy_schema_test.go` + `strategy_token_test.go` | `Strategy` | strategy configs |
| `trader.go` | `Trader` | the joining table: user + ai_model + exchange + strategy |
| `position.go` + `position_builder.go` + `position_history.go` + `position_query.go` | `Position` | open + closed positions |
| `order.go` | `Order` | order records |
| `decision.go` + `decision_test.go` | `Decision` | AI decisions, includes Plan 4 T23.4 audit fields (`prompt_version`, `ai_model`, `ai_latency_ms`, `risk_check_passed`, `risk_check_error`, `execution_status`, `fill_price`, `fill_latency_ms`) |
| `equity.go` | `Equity` | per-cycle equity snapshots for the equity chart |
| `grid.go` | grid state | grid trading-specific |
| `telegram_config.go` | `TelegramConfig` | bot token + binding |
| `visibility.go` | — | `IsVisibleExchange()`, `IsVisibleAIModel()` filters |

**Encrypted-at-rest fields** use the `crypto.EncryptedString` GORM hook — fields decrypted on Read, encrypted on Write. This is why crypto must initialize BEFORE the DB connection.

## Layer 3: Backend code paths (full system)

### Boot sequence (composite)

```
main.go
 ├─ config.Init()                              → reads .env / env vars
 ├─ crypto.NewCryptoService()                  → RSA keypair for transport encryption
 │                                               + AES wrapper for at-rest encryption
 ├─ store.NewWithConfig(...)                   → opens DB, runs AutoMigrate
 │  └─ initInstallationID                      → UUID for telemetry
 ├─ auth.SetJWTSecret(cfg.JWTSecret)
 ├─ manager.NewTraderManager()
 ├─ traderManager.LoadTradersFromStore(st)     → for each row in `traders` table:
 │  └─ addTraderFromStore(t)
 │     ├─ resolves AI model + exchange + strategy
 │     ├─ switch t.ExchangeType {
 │     │    case "ninjatrader":
 │     │      ntTrader.NewTraderFromEnv(cfg)
 │     │       ├─ NT_TRANSPORT=tcp → getOrStartTCPServer() singleton + NewTCPTrader
 │     │       └─ NT_TRANSPORT=csv → New(cfg) — CSV writer + tailer
 │     │    case "binance": binance.New(...)
 │     │    … 8 more brokers
 │     │  }
 │     ├─ AutoTrader{broker, kernel, ai, cfg}
 │     └─ if t.IsRunning: go AutoTrader.Start()
 ├─ api.NewServer(traderManager, st, cryptoService, port)
 │  └─ setupRoutes()                            → ~80 routes
 ├─ nofxiagent.New(...)
 │  └─ NOFXi Agent + skill DAG + memory
 ├─ go server.Start()                          → gin HTTP on :8080
 ├─ go telegram.Start(...)                     → telegram bot loop
 └─ <-quit                                     → SIGINT/SIGTERM
    ├─ server.Shutdown()
    ├─ nofxiAgent.Stop() (via defer)
    └─ traderManager.StopAll()
```

### Per-trader scan cycle

```
AutoTrader.scanLoop (every cfg.ScanIntervalMinutes)
 ├─ kernel.IsCMEOpen(now) check (futures only)              → skip if closed
 ├─ ctx := buildContext(broker, store)
 │  ├─ broker.GetBalance()                                  → AccountInfo
 │  ├─ broker.GetPositions()                                → []PositionInfo
 │  ├─ store.Decision().GetRecent(traderID, limit)          → recent decisions for AI context
 │  ├─ store.Order().GetRecent(traderID, limit)             → recent orders
 │  ├─ market.GetKlines(symbol, intervals)                  → for each symbol & timeframe
 │  └─ databento.GetOHLCV(...)                              → futures-only path
 ├─ riskLimits.Check(ctx)                                   → kill switch + daily PnL gate
 ├─ engine.Run(ctx)
 │  ├─ BuildSystemPrompt(variant)                           → futures or crypto template
 │  ├─ BuildUserPrompt(ctx)                                 → current market snapshot
 │  ├─ aiClient.Complete(systemPrompt, userPrompt)          → blocking LLM call
 │  └─ parseDecisions(response)                             → []Decision
 ├─ riskLimits.Filter(decisions)                            → drop if violates limits
 ├─ for each decision:
 │  ├─ store.Decision().Create(traderID, decision)
 │  ├─ broker.<action>(symbol, qty, price)                  → OpenLong / OpenShort / etc.
 │  └─ store.Order().Create(traderID, order)
 └─ store.Equity().Snapshot(traderID, account.TotalEquity)
```

### Force-flat path (Plan 4 Task 23)

```
POST /api/risk/force-flat?trader_id=…
 └─ handleForceFlat (api/handler_risk.go)
    ├─ resolve trader by id
    ├─ if exchange_type != ninjatrader: return {triggered: false, reason}
    └─ kernel.ForceFlatSignaler.Signal(traderID)
       ├─ broker.GetPositions()
       ├─ for each open position: broker.<closeOpposite>(...)
       │  ├─ CSV transport: WriteSignal({Direction: "CLOSE"})  → may not be supported
       │  └─ TCP transport: emit force_flat frame
       ├─ kernel.RiskLimits.Trip(traderID)                   → kill switch ON, blocks new decisions
       └─ kernel.RiskLimits.ResetDailyPnL(traderID)
```

### Plan 1.5 TCP wire (for NT)

```
Bot (Go side, TCPServer on :36974)            NT8 AddOn (Windows)
 ├─ accept connection                          dial 127.0.0.1:36974
 ├─ flush pending queue (drop > 60s old)
 ├─ heartbeat every 30s                       ack heartbeat
 ├─ on signal:
 │  └─ frame({type: "signal", payload: {...}}) → submit OCO bracket via NT8 SDK
 │                                              fill: frame({type: "fill", payload})
 │  └─ deliver to TCPTrader.fillCh             ← fill frame
 └─ on conn close (60s no ack):
    ├─ close conn cleanly                     reconnect every 5s
    └─ retain pending queue
```

## Layer 4: Cross-references + gaps + open questions

### Shipped fixes (Plan tags) — backend slice

| Plan | File | What |
|---|---|---|
| Plan 1 Task 1 | `config/config.go` | Add `DatabentoAPIKey`, `DatabentoDataset`, `NinjaTraderDataDir`, `TradingMode` fields |
| Plan 1 Task 2 | `provider/databento/historical.go` | Databento OHLCV client |
| Plan 1 Task 3 | `provider/databento/resolve.go` | Continuous symbol resolver |
| Plan 1 Task 4 | `market/databento_adapter.go` | Bar → Kline adapter |
| Plan 1 Task 5 | `provider/ninjatrader/csv_writer.go` | NT CSV writer (atomic temp+rename) |
| Plan 1 Task 6 | `provider/ninjatrader/csv_tailer.go` | NT CSV fill tailer |
| Plan 1 Task 7 | `trader/ninjatrader/trader.go` | 19-method `Trader` impl |
| Plan 1 Task 8 | `trader/auto_trader.go:263-315`, `manager/trader_manager.go:~700` | Wire NT into broker switch |
| Plan 1 Task 9 | `kernel/engine_prompt_futures.go` | Futures-specific AI prompt |
| Plan 1 Task 10 | `cmd/nq_smoke/main.go` | End-to-end smoke runner |
| Plan 1 Task 11 | `api/server.go`, web/router | Remove deleted pages (Data, StrategyMarket, Competition) |
| Plan 1 Task 13 | `store/exchange.go` | Add `NTDataDir`, `NTInstrumentName`, `NTDefaultContractQty` columns |
| Plan 2 Task 17 | `trader/ninjatrader/tick_rounding.go` | NQ 0.25 tick rounding |
| Plan 2 Task 18 | `kernel/cme_calendar.go` | CME session gating (ADR-006) |
| Plan 2 Task 19 | `provider/databento/contract_calendar.go` | Quarterly roll arithmetic |
| Plan 3 | `kernel/risk_limits.go` | Daily loss limit + kill switch (ADR-004 ForceFlatSignaler decoupling) |
| Plan 4 Task 23 | `api/handler_risk.go` (POST /api/risk/force-flat, GET /api/risk/status), `store/decision.go` (audit columns) | Risk + audit |
| Plan 4 Task 23.4 | `api/handler_decisions.go` (GET /api/audit/decisions) | Audit endpoint |
| Plan 4 Task 24 | `provider/databento/client.go:dbBreaker` | Circuit breaker for transient HTTP errors |
| Plan 4 Task 25 | `api/server.go:81` + telemetry/metrics.go | `GET /metrics` Prometheus |
| Plan 5 Task 26 | `provider/databento/mock_server.go` | httptest mock for Databento |
| Plan 5 Task 27 | `provider/ninjatrader/mock_nt.go` | NT bridge mock |
| Plan 5 Task 29 | `cmd/nq_smoke/main.go` (additive sub-commands per ADR-007 exception) | Smoke matrix |
| Plan 1.5 (spec + scaffolding) | `provider/ninjatrader/tcp_*.go`, `trader/ninjatrader/{tcp_trader, transport}.go` | TCP transport opt-in |
| Plan 1.5.6 | `provider/ninjatrader/tcp_server.go` (writeMu) | Concurrent-write mutex + heartbeat ack write deadline |

### Known gaps

| Gap | Plan | File | Symptom | Scope |
|---|---|---|---|---|
| `GetBalance` returns hardcoded `$50,000` for NT | Plan 4.11 | `trader/ninjatrader/trader.go:161-162` | Dashboard equity card always 50k; risk math sized against fantasy balance | 150 LOC Go + C# AddOn extension, ~2 hr |
| `agent/web.go::HandleKlines` + `/api/klines` have no Databento branch | Plan 4.5 | `agent/web.go:207, api/handler_klines.go` | Chart on NT trader's Dashboard returns 500 or wrong-symbol data | ~120 LOC, 60 min |
| Manual close not supported for NT | Plan 1.5.x | `trader/ninjatrader/trader.go` (CSV) | `CloseLong`/`CloseShort` returns error; position only closes via SL/TP | requires C# AddOn change |
| Cancel/modify not supported for NT | Plan 1.5.x | same | Mid-trade SL/TP changes can't be pushed | requires C# AddOn change |
| 1-sec CSV dedup race | known limitation | `provider/ninjatrader/csv_writer.go` | Two signals within 1 sec → second silently dropped; TCP transport doesn't have this issue | Use NT_TRANSPORT=tcp |
| `kernel/engine.go` imports `provider/nofxos` but the service is deprecated | technical debt | `kernel/engine.go:14` | Dead import + dead code paths | Cleanup after Plan 4.x ships |
| `cmd/nq_smoke/main.go` is the only consumer of multi-stage Databento pipeline outside `kernel/engine.go` | n/a | `cmd/nq_smoke/` | The smoke runner is the only way to test the data layer outside a full trader cycle | n/a — intended |

### NEW observations

| Description | File:line | Severity | Recommended scope |
|---|---|---|---|
| `main.go:160` starts the Telegram goroutine even when `TELEGRAM_BOT_TOKEN` is empty; the goroutine then sleeps in a check loop. Mild waste of a goroutine. | `main.go:160` + `telegram/start.go` | Cosmetic | 5-min: short-circuit `telegram.Start` when token is empty |
| `nofxiAgent.Stop()` is called via `defer` AFTER `traderManager.StopAll()` returns — but the agent might still be holding handles to traders | `main.go:151, 180` | Edge case | Test for shutdown ordering issues. Currently no symptoms reported. |
| `crypto.SetGlobalCryptoService(cs)` uses a package-level global. Cleaner DI would inject this — but the GORM `EncryptedString` hook needs a global to work (no per-conn context in GORM hooks). Documented for future reference. | `crypto/` | Architectural — n/a now | n/a |
| `agent/web.go:35` hardcodes `binanceFuturesAPIBaseURL = "https://fapi.binance.com"` — not env-overridable. Means even non-Binance traders hit Binance for ticker data via the agent. | `agent/web.go:35` | Multi-exchange chart gap | Bundle with Plan 4.5 |
| The `nofxos` import in `kernel/engine.go` looks unused-at-glance but probably has indirect coupling via `nofxos.NewClient(apiKey)` usage. Worth a `go vet`-style pass. | `kernel/engine.go:14` | Code hygiene | 10-min: grep for actual usage |
| `store/visibility.go` — IsVisibleExchange logic checks `NTDataDir != ""` as a sufficient condition for visibility. If a NT exchange row is misconfigured (saved with empty DataDir), it disappears from the list with no warning. | `store/visibility.go:64-80` | UX | Show disabled entries with a "config incomplete" warning instead of hiding |
| All HTTP handlers use `c.GetString("user_id")` to identify the caller — but the value is the `user_id` field in the JWT claim, NOT the `username`. Worth documenting because tools like `agent_routes.go` re-use this value as `storeUserID` (the convention is "user_id from JWT == user_id in store"). | `api/handler_*.go` patterns | Documentation | n/a |
| TraderManager error handling: `addTraderFromStore` returns error but main.go's `LoadTradersFromStore` swallows individual failures (logs but continues). One broken exchange config doesn't break the whole boot. | `manager/trader_manager.go` | Operational | Document explicitly |

### Open questions

- What's the relationship between `ai/` (if exists) and `agent/`? Quick file scan suggests no `ai/` package — AI provider abstractions live in `agent/model_provider_catalog.go`.
- The `mcp/` import in main.go (line 13-14): `_ "nofx/mcp/payment"` + `_ "nofx/mcp/provider"` — blank imports for side effects (init() registration). What MCP servers are bootstrapped this way? Worth a follow-up pass.
- `kernel.Decision` shape vs `store.Decision` shape — same fields or do they diverge? The audit columns are on `store.Decision`; `kernel.Decision` is the in-memory shape produced by the LLM.
- `trader/auto_trader.go` line 60 comment mentions Exchange field — what's the canonical source of truth for `exchange_type` string values? `store.Exchange.ExchangeType` is the column; the validation in `api/handler_exchange.go:347` has a `validTypes` map. Worth confirming they match.
- ADR-007 lists 19 critical files but the actual count of files in `trader/ninjatrader/` + `provider/ninjatrader/` + `provider/databento/` + `market/` + `kernel/` + `cmd/nq_smoke/` is much higher now after Plan 4-5. Verify ADR-007 file list is current.

### Cross-page (backend → frontend) dependencies

- `traderManager.LoadTradersFromStore` runs at boot AND on `/api/traders/:id/start` AND `/api/exchanges` PUT (when exchange creds change for a running trader). Frontend must NOT assume trader state survives an exchange edit.
- `crypto.EncryptedString` GORM hook means READING from `store.Exchange` triggers decryption — the `SafeExchangeConfig` DTO is what gets exposed to the frontend (page-3 backend section), but internal code paths can see the plain values.
- The `agent-config-refresh` window event is dispatched from agent tool execution and listened to by both Settings and AITradersPage. Page 5 / StrategyStudioPage does NOT listen (relies on focus/visibilitychange).
- Telegram bot consumes the same `store.Trader` + `store.Decision` data as the web UI; it's another front-end onto the same backend.
- The `manager.TraderManager` is shared across the API server AND the agent — agent's `manage_trader` tool calls `traderManager.GetTrader(id).Start()` directly. The agent's mutations are visible to the dashboard polling within one cycle.
