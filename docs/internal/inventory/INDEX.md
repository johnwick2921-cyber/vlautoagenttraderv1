# VL Page Inventory — Cross-reference Index

Companion to the 5 per-page docs + `BACKEND.md`. Use this index to find which page uses what, and to locate a function/endpoint/handler by name.

## Inventory document map

| Doc | What | LOC |
|---|---|---|
| [page-1-vl-root.md](page-1-vl-root.md) | `/` redirect logic + auth gate | ~125 |
| [page-2-agentbeta.md](page-2-agentbeta.md) | `/agent` — AgentBeta chat | ~270 |
| [page-3-config-settings.md](page-3-config-settings.md) | `/settings` — Config (4 tabs) | ~225 |
| [page-4-dashboard.md](page-4-dashboard.md) | `/traders` + `/dashboard` | ~310 |
| [page-5-strategy.md](page-5-strategy.md) | `/strategy` — Strategy Studio | ~200 |
| [BACKEND.md](BACKEND.md) | Cross-cutting backend layer | ~370 |
| `INDEX.md` (this file) | Cross-reference index | — |

## Function / component → page reference

Alphabetical. Components in `PascalCase`; module-level functions in `camelCase`; backend Go functions prefixed `Go::`.

| Symbol | File(s) | Page(s) |
|---|---|---|
| `addTraderFromStore` | `manager/trader_manager.go:~700` | Backend (TraderManager) |
| `AdvancedChart` | `web/src/components/charts/AdvancedChart.tsx` (1202 LOC) | Page 4 (Dashboard kline tab) |
| `Agent` (Go) | `agent/agent.go` | Backend, Page 2 |
| `AgentChatPage` | `web/src/pages/AgentChatPage.tsx` (919 LOC) | Page 2 |
| `AgentStepPanel` | `web/src/components/agent/AgentStepPanel.tsx` | Page 2 |
| `AITradersPage` | `web/src/components/trader/AITradersPage.tsx` (802 LOC) | Page 4 (/traders list) |
| `AppChrome` | `web/src/router/AppRoutes.tsx:122-189` | All pages (shared layout) |
| `AppRoutes` | `web/src/router/AppRoutes.tsx:415` | Pages 1, 2, 3, 4, 5 (the router itself) |
| `BarsRequest` (NT8 SDK) | `ninjascript/VLTraderTCPClient.cs` | Backend (Plan 4.4 deep spec) |
| `BeginnerOnboardingPage` | `web/src/pages/BeginnerOnboardingPage.tsx` | Page 4 (/welcome path) |
| `BuildFuturesSystemPrompt` (Go) | `kernel/engine_prompt_futures.go` | Backend (kernel), Page 5 (preview) |
| `BuildSystemPrompt` (Go) | `kernel/engine_prompt.go` | Backend |
| `buildAgentTools` (Go) | `agent/tools.go:cachedTools init` | Page 2, Backend |
| `buildDashboardPath` | `web/src/router/paths.ts:65-71` | Pages 1, 4 |
| `ChartTabs` | `web/src/components/charts/ChartTabs.tsx` (435 LOC) | Page 4 |
| `ChatInput` + `ChatInputHandle` | `web/src/components/agent/ChatInput.tsx` | Page 2 |
| `ChatMessages` | `web/src/components/agent/ChatMessages.tsx` | Page 2 |
| `cleanupActiveAgentStream` | `web/src/pages/AgentChatPage.tsx:44-51` | Page 2 |
| `CMECalendar.IsCMEOpen` (Go) | `kernel/cme_calendar.go` | Backend (Plan 2 Task 18, ADR-006) |
| `compactManageStrategyTool` (Go) | `agent/tools.go:140` | Backend, Page 2 |
| `ConfigStatusGrid` | `web/src/components/trader/ConfigStatusGrid.tsx` | Page 4 (/traders) |
| `CoinSourceEditor` | `web/src/components/strategy/CoinSourceEditor.tsx` (433 LOC) | Page 5 |
| `configBadge` | `web/src/pages/SettingsPage.tsx:24-36` | Page 3 |
| `confirmToast` | `web/src/lib/notify.tsx` | Pages 4, 5 (delete confirmations) |
| `DashboardRoute` | `web/src/router/AppRoutes.tsx:225-413` | Page 4 |
| `DecisionAudit` | `web/src/components/trader/DecisionAudit.tsx` (310 LOC) | Page 4 (Decisions tab) |
| `DecisionCard` | `web/src/components/trader/DecisionCard.tsx` (481 LOC) | Page 4 (Recent Decisions panel) |
| `DeepVoidBackground` | `web/src/components/common/DeepVoidBackground.tsx` | Pages 4, 5 |
| `defaultGridConfig` | `web/src/components/strategy/GridConfigEditor.tsx` | Page 5 |
| `EmergencyFlatButton` | `web/src/components/trader/EmergencyFlatButton.tsx` (123 LOC) | Page 4 (header) |
| `EquityChart` | `web/src/components/charts/EquityChart.tsx` (499 LOC) | Page 4 (Equity tab in ChartTabs) |
| `ExchangeConfigModal` | `web/src/components/trader/ExchangeConfigModal.tsx` (925 LOC) | Pages 3, 4 (modal opens from both) |
| `ExchangeCard` | inline in ExchangeConfigModal.tsx:101-155 | Page 3 |
| `FAQPage` | `web/src/pages/FAQPage.tsx` | (FAQ route — not in 5 live pages, but exists) |
| `findTraderBySlug` | `web/src/router/AppRoutes.tsx:53-65` | Page 4 |
| `flattenDecisions` | `web/src/components/trader/DecisionAudit.tsx:58-86` | Page 4 |
| `formatPrice` | `web/src/utils/format.ts` | Page 4 |
| `formatQuantity` | `web/src/utils/format.ts` | Page 4 |
| `ForceFlatSignaler` (Go) | `kernel/risk_limits.go` + ADR-004 | Backend, Page 4 |
| `getAIConfig` | `web/src/pages/StrategyStudioPage.tsx:56-68` | Page 5 |
| `getCurrentPageForPath` | `web/src/router/paths.ts:43-63` | Pages 1+ (HeaderBar nav highlighting) |
| `getExchangeDisplayNameFromList` | `web/src/pages/TraderDashboardPage.tsx:44-55` | Page 4 |
| `getExchangeIcon` | `web/src/components/common/ExchangeIcons.tsx` | Pages 3, 4 |
| `getExchangeTypeFromList` | `web/src/pages/TraderDashboardPage.tsx:57-66` | Page 4 |
| `getMarketTypeFromExchange` | `web/src/components/charts/ChartTabs.tsx:83-89` | Page 4 |
| `getModelDisplayName` | `web/src/pages/TraderDashboardPage.tsx:30-41` | Page 4 |
| `getShortName` | `web/src/components/trader/utils.ts` | Page 3 |
| `getTraderSlug` | `web/src/router/AppRoutes.tsx:48-51` | Page 1, 4 |
| `getUserMode` | `web/src/lib/onboarding.ts` | Page 1 (Welcome route) |
| `getWalletAddress` | `web/src/pages/TraderDashboardPage.tsx:76-89` | Page 4 |
| `GridConfigEditor` | `web/src/components/strategy/GridConfigEditor.tsx` (474 LOC) | Page 5 |
| `GridRiskPanel` | `web/src/components/strategy/GridRiskPanel.tsx` (313 LOC) | Page 4 (only if strategy_type=grid_trading) |
| `handleChangePassword` | `web/src/pages/SettingsPage.tsx:98-127` | Page 3 |
| `handleClosePosition` (FE) | `TraderDashboardPage.tsx:216-251` | Page 4 |
| `handleCopyAddress` (FE) | `TraderDashboardPage.tsx:191-200` | Page 4 |
| `handleDeleteExchange` | `SettingsPage.tsx:307-317` | Page 3 |
| `handleDeleteModel` | `SettingsPage.tsx:193-227` | Page 3 |
| `handleSaveExchange` | `SettingsPage.tsx:229-305` | Page 3 |
| `handleSaveModel` | `SettingsPage.tsx:129-191` | Page 3 |
| `handleSymbolClick` | `TraderDashboardPage.tsx:203-213` | Page 4 |
| `HeaderBar` | `web/src/components/common/HeaderBar.tsx` | All pages |
| `IndicatorEditor` | `web/src/components/strategy/IndicatorEditor.tsx` (691 LOC) | Page 5 |
| `initInstallationID` | `main.go:186-206` | Backend (bootstrap) |
| `isPerpDexExchange` | `TraderDashboardPage.tsx:69-73` | Page 4 |
| `IsVisibleAIModel` (Go) | `store/visibility.go` | Backend (Page 3) |
| `IsVisibleExchange` (Go) | `store/visibility.go` | Backend (Page 3) |
| `LegacyHashRedirect` | `web/src/router/AppRoutes.tsx:87-111` | Pages 1+ |
| `LoadingScreen` | `web/src/router/AppRoutes.tsx:67-85` | Page 1 (loading state) |
| `LoginPage` | `web/src/components/auth/LoginPage.tsx` | (Auth route) |
| `LoginRequiredOverlay` | `web/src/components/auth/LoginRequiredOverlay.tsx` | Pages 2, 4 (when auth-gated content requested without login) |
| `MarketTicker` | `web/src/components/agent/MarketTicker.tsx` (209 LOC) | Page 2 |
| `MessageRenderer` | `web/src/components/agent/MessageRenderer.tsx` | Page 2 |
| `ModelConfigModal` | `web/src/components/trader/ModelConfigModal.tsx` (1313 LOC) | Pages 3, 4 |
| `New` (NT trader, CSV) | `trader/ninjatrader/trader.go` | Backend |
| `NewTraderFromEnv` | `trader/ninjatrader/transport.go:57-71` | Backend (env-var router) |
| `NewTCPTrader` | `trader/ninjatrader/tcp_trader.go` | Backend (Plan 1.5) |
| `nextId` | `web/src/pages/AgentChatPage.tsx:40-42` | Page 2 |
| `NofxSelect` | `web/src/components/ui/select.tsx` | Pages 4, 5 |
| `normalizeStrategyConfig` | `StrategyStudioPage.tsx:70-80` | Page 5 |
| `PageNotFound` | `web/src/pages/PageNotFound.tsx` | **Orphan** (not wired into router) |
| `parseFillRow` (Go) | `provider/ninjatrader/csv_tailer.go` | Backend |
| `patchMessagesInStore` | `AgentChatPage.tsx:93-100` | Page 2 |
| `persistMessagesSnapshotForUser` | `AgentChatPage.tsx:81-86` | Page 2 |
| `PositionHistory` | `web/src/components/trader/PositionHistory.tsx` (918 LOC) | Page 4 |
| `PositionsPanel` | `web/src/components/agent/PositionsPanel.tsx` (164 LOC) | Page 2 (right sidebar) |
| `PromptSectionsEditor` | `web/src/components/strategy/PromptSectionsEditor.tsx` (178 LOC) | Page 5 |
| `proxyBinance` (Go) | `agent/web.go:339` | Backend, Page 2 |
| `PublishSettingsEditor` | `web/src/components/strategy/PublishSettingsEditor.tsx` (184 LOC) | Page 5 |
| `PunkAvatar` + `getTraderAvatar` | `web/src/components/common/PunkAvatar.tsx` | Page 4 |
| `RegisterAgentHandler` (Go) | `api/agent_routes.go:11` | Backend, Page 2 |
| `RegisterPage` | `web/src/components/auth/RegisterPage.tsx` | (Auth route) |
| `replaceMessagesInStore` | `AgentChatPage.tsx:88-91` | Page 2 |
| `ResetPasswordPage` | `web/src/components/auth/ResetPasswordPage.tsx` | (Auth route) |
| `RiskControlEditor` | `web/src/components/strategy/RiskControlEditor.tsx` (316 LOC) | Page 5 |
| `RiskLimits.Check` (Go) | `kernel/risk_limits.go` | Backend (Plan 3) |
| `runAgentStream` | `AgentChatPage.tsx:102-…` | Page 2 |
| `safeExchangeConfigFromStore` (Go) | `api/handler_exchange.go:52-75` | Backend (Page 3) |
| `safeExchangeForTool` (Go) | `agent/tools.go:1024` | Backend (Page 2) |
| `safeModelForTool` (Go) | `agent/tools.go:1135` | Backend (Page 2) |
| `safeStrategyForTool` (Go) | `agent/tools.go:1182` | Backend (Page 2) |
| `safeTraderForTool` (Go) | `agent/tools.go:1167` | Backend (Page 2) |
| `scaledFloat` (Go) | `provider/databento/historical.go` | Backend |
| `SetupPage` | `web/src/components/modals/SetupPage.tsx` | Page 1 (when system uninitialized) |
| `SettingsPage` | `web/src/pages/SettingsPage.tsx` (649 LOC) | Page 3 |
| `side` | `kernel/engine_prompt_futures.go` (helper) | Backend |
| `SiteFooter` | `web/src/components/common/SiteFooter.tsx` | Pages 4, 5 (not on 2 or 3 — `showFooter={false}`) |
| `splitComma` (Go) | `agent/web.go:329` | Backend, Page 2 |
| `StatCard` (inline) | `TraderDashboardPage.tsx:1088-1154` | Page 4 |
| `StepIndicator` | inline in ExchangeConfigModal.tsx:66-97 | Page 3 |
| `stopActiveAgentStream` | `AgentChatPage.tsx:53-79` | Page 2 |
| `StrategyStudioPage` | `web/src/pages/StrategyStudioPage.tsx` (1455 LOC) | Page 5 |
| `t` (i18n) | `web/src/i18n/translations.ts` | All pages |
| `TCPServer` (Go) | `provider/ninjatrader/tcp_server.go:65-…` | Backend (Plan 1.5) |
| `TelegramConfigModal` | `web/src/components/trader/TelegramConfigModal.tsx` (515 LOC) | Page 3, Page 4 |
| `TokenEstimateBar` | `web/src/components/strategy/TokenEstimateBar.tsx` (122 LOC) | Page 5 |
| `Toaster` (sonner) | `main.tsx:11-25` | All pages |
| `TraderConfigModal` | `web/src/components/trader/TraderConfigModal.tsx` | Page 4 |
| `TraderDashboardPage` | `web/src/pages/TraderDashboardPage.tsx` (1154 LOC) | Page 4 |
| `TraderManager` (Go) | `manager/trader_manager.go` (782 LOC) | Backend |
| `TradersList` | `web/src/components/trader/TradersList.tsx` | Page 4 |
| `TradersRoute` | `web/src/router/AppRoutes.tsx:191-223` | Page 4 (/traders) |
| `TraderStatusPanel` | `web/src/components/agent/TraderStatusPanel.tsx` (119 LOC) | Page 2 |
| `truncateAddress` | `TraderDashboardPage.tsx:92-95` | Page 4 |
| `TwoStageKeyModal` | `web/src/components/modals/TwoStageKeyModal.tsx` | Page 3 (ExchangeConfigModal sub-modal) |
| `useAgentChatStore` (Zustand) | `web/src/stores/agentChatStore.ts` | Page 2 |
| `useAuth` | `web/src/contexts/AuthContext.tsx` | All pages |
| `useLanguage` | `web/src/contexts/LanguageContext.tsx` | All pages |
| `useSystemConfig` | `web/src/hooks/useSystemConfig.ts` | Pages 1+ |
| `UserPreferencesPanel` | `web/src/components/agent/UserPreferencesPanel.tsx` (241 LOC) | Page 2 |
| `WebCryptoEnvironmentCheck` | `web/src/components/common/WebCryptoEnvironmentCheck.tsx` | Page 3 |
| `WelcomeScreen` | `web/src/components/agent/WelcomeScreen.tsx` (188 LOC) | Page 2 |
| `writeSSE`, `sseEscape` (Go) | `agent/web.go:191-205` | Backend, Page 2 |
| `xyzDexAssets` | `CoinSourceEditor.tsx:31-42` | Page 5 |

## API endpoint → page reference

Endpoints that are accessed by each page during normal operation. Listed in alphabetical order.

| Endpoint | Method | Pages | Handler |
|---|---|---|---|
| `/api/account` | GET | 4 | `handleAccount` |
| `/api/agent/chat` | POST | 2 | `WebHandler.HandleChat` |
| `/api/agent/chat/stream` | POST | 2 | `WebHandler.HandleChatStream` (SSE) |
| `/api/agent/health` | GET | 2 | `WebHandler.HandleHealth` |
| `/api/agent/klines` | GET | 2 | `WebHandler.HandleKlines` (Binance proxy) |
| `/api/agent/preferences` | GET/POST/DELETE | 2 | `handleGetAgentPreferences` / `handleCreateAgentPreference` / `handleDeleteAgentPreference` |
| `/api/agent/ticker` | GET | 2 | `WebHandler.HandleTicker` |
| `/api/agent/tickers` | GET | 2 | `WebHandler.HandleTickers` |
| `/api/ai-costs` | GET | 4 (potentially — not in capture) | `handleGetAICosts` |
| `/api/audit/decisions` | GET | 4 (Decisions tab) | `handleDecisionAudit` |
| `/api/config` | GET | All pages (App boot) | `handleGetSystemConfig` |
| `/api/crypto/config` | GET | 3 (Settings modal) | `cryptoHandler.HandleGetCryptoConfig` |
| `/api/crypto/public-key` | GET | 3 | `cryptoHandler.HandleGetPublicKey` |
| `/api/decisions` | GET | 4 | `handleDecisions` |
| `/api/decisions/latest` | GET | 4 (Recent Decisions) | `handleLatestDecisions` |
| `/api/equity-history` | GET | 4 | `handleEquityHistory` |
| `/api/exchanges` | GET | 3, 4 | `handleGetExchangeConfigs` |
| `/api/exchanges` | POST | 3 | `handleCreateExchange` |
| `/api/exchanges` | PUT | 3 | `handleUpdateExchangeConfigs` |
| `/api/exchanges/:id` | DELETE | 3 | `handleDeleteExchange` |
| `/api/exchanges/account-state` | GET | 3 (Settings, status badges) | `handleGetExchangeAccountStates` |
| `/api/klines` | GET | 4 (when ChartTabs in kline mode) | `handleKlines` |
| `/api/models` | GET | 3, 4, 5 | `handleGetModelConfigs` |
| `/api/models` | PUT | 3 | `handleUpdateModelConfigs` |
| `/api/my-traders` | GET | 1+, 2, 4 | `handleTraderList` |
| `/api/onboarding/beginner` | POST | 1 (Setup) | `handleBeginnerOnboarding` |
| `/api/positions` | GET | 2 (right sidebar), 4 | `handlePositions` |
| `/api/positions/history` | GET | 4 (PositionHistory panel) | `handlePositionHistory` |
| `/api/risk/force-flat` | POST | 4 (EmergencyFlatButton) | `handleForceFlat` |
| `/api/risk/status` | GET | 4 (potentially — not in capture) | `handleRiskStatus` |
| `/api/server-ip` | GET | 3 (Binance section in modal) | `handleGetServerIP` |
| `/api/statistics` | GET | 4 | `handleStatistics` |
| `/api/status` | GET | 4 | `handleStatus` |
| `/api/strategies` | GET | 5, 4 (indirectly via trader's strategy_id) | `handleGetStrategies` |
| `/api/strategies` | POST | 5 | `handleCreateStrategy` |
| `/api/strategies/:id` | PUT/DELETE | 5 | `handleUpdateStrategy` / `handleDeleteStrategy` |
| `/api/strategies/:id/activate` | POST | 5 | `handleActivateStrategy` |
| `/api/strategies/:id/duplicate` | POST | 5 | `handleDuplicateStrategy` |
| `/api/strategies/default-config` | GET | 5 | `handleGetDefaultStrategyConfig` |
| `/api/strategies/estimate-tokens` | POST | 5 (TokenEstimateBar) | `handleEstimateTokens` |
| `/api/strategies/preview-prompt` | POST | 5 | `handlePreviewPrompt` |
| `/api/strategies/test-run` | POST | 5 | `handleStrategyTestRun` |
| `/api/supported-exchanges` | GET | 3 | `handleGetSupportedExchanges` |
| `/api/supported-models` | GET | 3, 4 | `handleGetSupportedModels` |
| `/api/symbols` | GET | 4 (Hyperliquid dropdown) | `handleSymbols` |
| `/api/telegram` | GET/POST | 3 (modal) | `handleGetTelegramConfig` / `handleUpdateTelegramConfig` |
| `/api/telegram/binding` | DELETE | 3 | `handleUnbindTelegram` |
| `/api/telegram/model` | POST | 3 | `handleUpdateTelegramModel` |
| `/api/traders` | POST | 4 (TraderConfigModal) | `handleCreateTrader` |
| `/api/traders/:id` | PUT/DELETE | 4 | `handleUpdateTrader` / `handleDeleteTrader` |
| `/api/traders/:id/start` | POST | 4 | `handleStartTrader` |
| `/api/traders/:id/stop` | POST | 4 | `handleStopTrader` |
| `/api/traders/:id/close-position` | POST | 4 (per-position close button) | `handleClosePosition` |
| `/api/traders/:id/config` | GET | 4 (Edit modal) | `handleGetTraderConfig` |
| `/api/traders/:id/grid-risk` | GET | 4 (only if grid_trading strategy) | `handleGetGridRiskInfo` |
| `/api/traders/:id/sync-balance` | POST | 4 | `handleSyncBalance` |
| `/api/user/password` | PUT | 3 | `handleChangePassword` (called via raw fetch, not api lib) |
| `/metrics` | GET | (Prometheus scrape — not a page) | `promhttp.Handler` |

## Cross-page workflow chains

### Workflow A — Create a NinjaTrader trader from scratch

```
Page 3 (Settings)
 ├─ AI Models tab: add DeepSeek (or other model) with API key
 │  └─ PUT /api/models
 └─ Exchanges tab: Add NinjaTrader exchange
    ├─ Select NinjaTrader card → enter DataDir, Instrument, ContractQty
    └─ POST /api/exchanges (Plan 4.2 fix: no encryption envelope for NT)

Page 5 (Strategy)
 └─ Create strategy (default config is fine)
    └─ POST /api/strategies
       └─ Activate → POST /api/strategies/:id/activate

Page 4 (/traders)
 ├─ Click "Create Trader"
 │  └─ TraderConfigModal → POST /api/traders
 │     Body: {name, ai_model_id, exchange_id, strategy_id, scan_interval_minutes}
 ├─ Click "Start" on the new row
 │  └─ POST /api/traders/:id/start
 └─ Click "View" to navigate to /dashboard
    └─ Dashboard polls 7 endpoints (status/account/positions/decisions/statistics/equity-history/audit)
```

### Workflow B — Run a test against a strategy

```
Page 5 (Strategy)
 ├─ Select strategy
 ├─ Edit Coin Source / Indicators / Risk Control / Prompts
 │  └─ POST /api/strategies/estimate-tokens (debounced as user types)
 ├─ Click "Prompt Preview"
 │  └─ POST /api/strategies/preview-prompt
 ├─ Click "AI Test"
 │  └─ POST /api/strategies/test-run (uses real AI tokens)
 └─ Save → PUT /api/strategies/:id
```

### Workflow C — Emergency-flat a runaway trader

```
Page 4 (/dashboard for the trader)
 ├─ Click red "Emergency Flat" button (top-right)
 ├─ Confirm dialog (Plan 4 Task 23.3) — Cancel does nothing
 ├─ Click "Confirm Flat"
 │  └─ POST /api/risk/force-flat?trader_id=…
 │     ├─ Backend: kernel.RiskLimits.Trip(traderID) → kill switch ON
 │     ├─ Backend: kernel.RiskLimits.ResetDailyPnL(traderID)
 │     └─ Backend: for each position → broker.<closeOpposite>
 └─ Modal shows JSON response — trader stays running but won't enter new positions until kill switch is reset
```

### Workflow D — Agent-driven config edit

```
Page 2 (/agent)
 ├─ User: "Add a Bybit exchange"
 ├─ Agent calls manage_exchange_config tool
 │  └─ Backend creates store.Exchange row
 ├─ Backend dispatches window.dispatchEvent('agent-config-refresh')
 ├─ Page 3 (Settings) listening: refreshExchangeConfigs() auto-fires
 ├─ Page 4 (/traders) listening: loadConfigs() auto-fires
 └─ Page 5 (Strategy) NOT listening — user must focus that tab to re-fetch
```

## Shipped fix summary (Plan tags → file:line)

| Plan tag | Date | Files | Page(s) | Note |
|---|---|---|---|---|
| Plan 1 Task 1-10 | 2026-05-22 | `config/`, `provider/databento/`, `provider/ninjatrader/`, `trader/ninjatrader/`, `kernel/engine_prompt_futures.go`, `market/databento_adapter.go`, `cmd/nq_smoke/` | Backend, Pages 3, 4, 5 | Plan 1 ship — SIM-validated |
| Plan 1 Task 11 | 2026-05-22 | `web/src/router/AppRoutes.tsx`, `paths.ts`, deleted DataPage/StrategyMarketPage/CompetitionPage | All pages (negative-space) | Three pages removed |
| Plan 1 Task 13 | 2026-05-23 | `store/exchange.go` + migrations | Backend, Page 3 | NT fields on Exchange |
| Plan 1 Task 14 | 2026-05-23 | `ExchangeConfigModal.tsx`, `SettingsPage.tsx`, `web/src/types/config.ts`, `api/handler_exchange.go` | Page 3 | NT exchange in modal |
| Plan 2 Task 17 | 2026-05-23 | `trader/ninjatrader/tick_rounding.go` | Backend | NQ 0.25 tick rounding |
| Plan 2 Task 18 | 2026-05-23 | `kernel/cme_calendar.go` | Backend | ADR-006 CME session gating |
| Plan 2 Task 19 | 2026-05-23 | `provider/databento/contract_calendar.go` | Backend | Quarterly roll arithmetic |
| Plan 3 | ~2026-05-24 | `kernel/risk_limits.go`, ADR-004 ForceFlatSignaler decoupling | Backend | Risk limits + kill switch |
| Plan 4 Task 23 | ~2026-05-25 | `kernel/risk_limits.go`, `api/handler_risk.go`, `api/handler_decisions.go`, `EmergencyFlatButton.tsx`, `DecisionAudit.tsx` | Backend, Page 4 | Risk + audit endpoints + UI |
| Plan 4 Task 23.4 | ~2026-05-25 | `DecisionAudit.tsx`, `TraderDashboardPage.tsx:149-152, 626-671`, `store/decision.go` audit columns | Page 4 | Decisions tab |
| Plan 4 Task 24 | ~2026-05-25 | `provider/databento/client.go:dbBreaker`, `internal/retry/` | Backend | Circuit breaker |
| Plan 4 Task 25 | ~2026-05-25 | `api/server.go:81`, `telemetry/metrics.go` | Backend | Prometheus /metrics |
| Plan 4 Task 23 | ~2026-05-25 | Plan 4 review | Backend | (cross-listed) |
| Plan 4.2 fix | 2026-05-26 | `web/src/lib/api/config.ts:108-116` | Page 3 | createExchangeEncrypted NT short-circuit |
| Plan 4.3 | 2026-05-26 | `TraderDashboardPage.tsx:185-188, 559, 572, 588, 755-868` | Page 4 | StatCard USD unit + hide Leverage/Liq for NT |
| Plan 4.5 | (still open) | `api/handler_klines.go`, `agent/web.go:207` | Page 4, 2 | **GAP** — chart endpoints for NT |
| Plan 4.6 | (still open) | `web/src/components/strategy/` | Page 5 | **GAP** — futures-aware strategy editor |
| Plan 4.7 | 2026-05-26 | `MarketTicker.tsx:14`, `WelcomeScreen.tsx:17-70` | Page 2 | MNQ-focused tickers + suggestions |
| Plan 4.7.1 | 2026-05-26 | `UserPreferencesPanel.tsx` | Page 2 | Placeholder panel (PR #25) |
| Plan 4.9 | 2026-05-26 | `DecisionAudit.tsx:75-83` | Page 4 | React key uniqueness |
| Plan 4.11 | (still open) | `trader/ninjatrader/trader.go:161-162` | Page 4 | **GAP** — NT real balance |
| Plan 4.14 | (still open) | TBD | Page 4 | **GAP** — Plan 1.5 backend visibility panel |
| Plan 1.5 spec + scaffolding | 2026-05-26 | `provider/ninjatrader/tcp_*.go`, `trader/ninjatrader/{tcp_trader, transport}.go` | Backend | TCP transport opt-in |
| Plan 1.5.6 | 2026-05-27 (commit 3e4ee61c) | `provider/ninjatrader/tcp_server.go` (writeMu, write deadline) | Backend | Concurrent-write mutex |

## Known-gap summary

| Gap | Plan # | File:line | Pages affected | Scope estimate |
|---|---|---|---|---|
| NT chart shows Binance data | Plan 4.4 | `ChartTabs.tsx:28-89` + `handler_klines.go` | Page 4 | ~720 LOC, ~33 hr (per Plan 4.4 deep spec) |
| `/api/klines` 500 for NT | Plan 4.5 | `api/handler_klines.go` | Pages 2, 4 | ~120 LOC, 60 min |
| `/api/symbols` 400 for NT | Plan 4.5 | `api/handler_symbols.go` | Page 4 | bundled in Plan 4.5 |
| Strategy Studio crypto-coupled | Plan 4.6 | `CoinSourceEditor`, `IndicatorEditor`, `RiskControlEditor` | Page 5 | ~400 LOC, 3 hr |
| NT balance $50k mock | Plan 4.11 | `trader/ninjatrader/trader.go:161-162` | Page 4 | ~150 LOC + C# AddOn, 2 hr |
| Plan 1.5 backend visibility | Plan 4.14 | TBD | Page 4 | TBD |
| Settings exchange API Key badge for NT | Plan 4.13 (audit fix) | `SettingsPage.tsx:532-558` | Page 3 | Cosmetic — already partially landed (TCP Bridge badge shows) |

## NEW observations from this inventory pass

These were not in the prior 2026-05-27 audit. Severity in column 4.

| # | Description | File:line | Severity | Page |
|---|---|---|---|---|
| N1 | `PageNotFound.tsx` exists but is NOT wired into the router; catch-all route silently redirects | `web/src/pages/PageNotFound.tsx` + `AppRoutes.tsx:519` | Cosmetic/orphan | Page 1 |
| N2 | Favicon 404 on every page load | n/a | Cosmetic | All pages |
| N3 | `LEGACY_HASH_ROUTES` may be dead code (no `#agent` hash URLs anymore) | `paths.ts:35-41` | Cosmetic | Page 1 |
| N4 | `getCurrentPageForPath` lacks `'settings'` case — Settings nav highlight may not work | `paths.ts:43-63` | UX | Pages 1, 3 |
| N5 | ExchangeConfigModal edit-mode populate reads non-existent fields from SafeExchangeConfig (always empty) | `ExchangeConfigModal.tsx:228-243` | Dead code | Page 3 |
| N6 | `handleSaveExchange` has 18 positional parameters | `SettingsPage.tsx:229-248` | Maintainability | Page 3 |
| N7 | `handleChangePassword` uses raw fetch instead of api helper | `SettingsPage.tsx:98-127` | Inconsistency | Page 3 |
| N8 | No "old password" required for password change | `handler_user.go::handleChangePassword` | **Security** | Page 3 |
| N9 | Tab buttons render only icon on mobile, no aria-label | `SettingsPage.tsx:336-350` | Accessibility | Page 3 |
| N10 | `Exchange` (TS interface) missing the new NT fields | `web/src/types/config.ts:21-50` | Type drift | Page 3 |
| N11 | **All recent decisions have `risk_check_passed=false`** — kill switch may be active | runtime; `kernel/risk_limits.go` state | **INVESTIGATION** | Page 4 |
| N12 | `/api/audit/decisions` fires TWICE on Decisions tab open | `DecisionAudit.tsx` (StrictMode) | Cosmetic + 2× cost | Page 4 |
| N13 | `/api/positions/history` fires TWICE on dashboard mount | `DashboardRoute` + `PositionHistory` have parallel SWR | Cosmetic + 2× cost | Page 4 |
| N14 | "Edit" button disabled when trader RUNNING with no tooltip explanation | `TradersList.tsx` | UX | Page 4 |
| N15 | "在竞技场显示" Chinese label in EN mode | `TradersList.tsx` | i18n gap | Page 4 |
| N16 | Footer links have empty `href=""` | `SiteFooter.tsx` | Cosmetic | All footers |
| N17 | ChartTabs has no `futures` market type; NT falls back to Binance default | `ChartTabs.tsx:38-44, 83-89` | Plan 4.4 prerequisite | Page 4 |
| N18 | `IndicatorEditor.DEFAULT_NOFXOS_API_KEY = "cm_568c67eae410d912c54c"` — dead default key | `IndicatorEditor.tsx:7` | Dead code | Page 5 |
| N19 | `/api/strategies` and `/api/models` each fire TWICE on Strategy page load | `StrategyStudioPage.tsx:207-210` + StrictMode | Cosmetic | Page 5 |
| N20 | External CDN reference `grainy-gradients.vercel.app/noise.svg` returns 404 | `DeepVoidBackground` or similar | Console noise | Page 5 |
| N21 | `/api/my-traders` fires unconditionally at boot with no token → ERR_ABORTED | global | Cosmetic | All pages |
| N22 | `MarketTicker.SYMBOLS = ['MNQ']` hardcoded — not user-configurable | `MarketTicker.tsx:14` | Functional limitation | Page 2 |
| N23 | Tool descriptions in `agent/tools.go` still mention crypto-only assumptions (funding rate, OI, USDT) | `agent/tools.go` various `Description:` fields | Plan 4.6-adjacent | Page 2 |
| N24 | `binanceFuturesAPIBaseURL` hardcoded module-level, not env-overridable | `agent/web.go:35` | Multi-exchange chart gap | Backend |
| N25 | `nofxos` import in `kernel/engine.go` looks unused; service is deprecated | `kernel/engine.go:14` | Code hygiene | Backend |
| N26 | Hardcoded `"默认策略"` literal in agent tools | `agent/tools.go:2166` | i18n debt | Backend |
| N27 | `chartUpdateKey` in TraderDashboardPage triggers chart re-fetch via `Date.now()` — fine pattern but state coupling | `TraderDashboardPage.tsx:145, 786` | n/a | Page 4 |

Total NEW observations: 27.
