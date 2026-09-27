# Page 4 — Dashboard (`/traders` + `/dashboard`)

This page has two distinct routes and primary components, but operationally they form one user-facing surface: `/traders` is the list, `/dashboard` is the per-trader detail. Both are documented here together.

## Quick reference

### /traders (list)
- **Route:** `/traders` ([AppRoutes.tsx:499-502](web/src/router/AppRoutes.tsx#L499-L502))
- **Wrapping route component:** `TradersRoute` ([AppRoutes.tsx:191-223](web/src/router/AppRoutes.tsx#L191-L223))
- **Primary source:** [web/src/components/trader/AITradersPage.tsx](web/src/components/trader/AITradersPage.tsx) (802 LOC)
- **Auth:** required; unauth → /login
- **Special path:** `/welcome` ([AppRoutes.tsx:486-498](web/src/router/AppRoutes.tsx#L486-L498)) — same `TradersRoute` with `showBeginnerOnboarding=true` for users in beginner mode

### /dashboard (per-trader detail)
- **Route:** `/dashboard?trader=<slug>` ([AppRoutes.tsx:503-506](web/src/router/AppRoutes.tsx#L503-L506))
- **Wrapping route component:** `DashboardRoute` ([AppRoutes.tsx:225-413](web/src/router/AppRoutes.tsx#L225-L413)) — owns all SWR polling and prop-drilling
- **Primary source:** [web/src/pages/TraderDashboardPage.tsx](web/src/pages/TraderDashboardPage.tsx) (1154 LOC)
- **Slug format:** `<trader_name>-<first 4 chars of trader_id>` (e.g. `mnq sIM TEST-1ef4`), decoded via `findTraderBySlug()`

### Combined sub-components
| Component | File | LOC | Used on |
|---|---|---|---|
| `EmergencyFlatButton` | `web/src/components/trader/EmergencyFlatButton.tsx` | 123 | Dashboard header |
| `DecisionAudit` | `web/src/components/trader/DecisionAudit.tsx` | 310 | Dashboard "Decisions" tab |
| `DecisionCard` | `web/src/components/trader/DecisionCard.tsx` | 481 | Dashboard "Overview" tab right column |
| `ChartTabs` | `web/src/components/charts/ChartTabs.tsx` | 435 | Dashboard "Overview" tab left column |
| `EquityChart` | `web/src/components/charts/EquityChart.tsx` | 499 | Inside ChartTabs (equity tab) |
| `AdvancedChart` | `web/src/components/charts/AdvancedChart.tsx` | 1202 | Inside ChartTabs (kline tab) |
| `PositionHistory` | `web/src/components/trader/PositionHistory.tsx` | 918 | Dashboard overview bottom |
| `GridRiskPanel` | `web/src/components/strategy/GridRiskPanel.tsx` | 313 | Dashboard (only when strategy_type==='grid_trading') |
| `TraderConfigModal` | `web/src/components/trader/TraderConfigModal.tsx` | — | Create / Edit Trader from /traders |
| `ConfigStatusGrid` | `web/src/components/trader/ConfigStatusGrid.tsx` | — | /traders summary panel |
| `TradersList` | `web/src/components/trader/TradersList.tsx` | — | /traders rows |
| `PunkAvatar` | `web/src/components/common/PunkAvatar.tsx` | — | Both routes |
| `NofxSelect` | `web/src/components/ui/select.tsx` | — | Both routes |
| `StatCard` (inline) | `TraderDashboardPage.tsx:1088-1154` | 66 | Dashboard equity/balance/pnl/positions cards |

### API endpoints captured during runtime

The list (/traders) plus the dashboard navigation in one session fired **all of these**:

```
GET  /api/config
GET  /api/my-traders                       (polled 5s on /traders, 10s on /dashboard)
GET  /api/models                           (loadConfigs on /traders)
GET  /api/exchanges                        (loadConfigs + dashboard exchanges-dashboard)
GET  /api/supported-models                 (loadConfigs)
GET  /api/equity-history?trader_id=…       (chart bootstrap)
GET  /api/account?trader_id=…              (15s)
GET  /api/status?trader_id=…               (15s)
GET  /api/positions?trader_id=…            (15s)
GET  /api/decisions/latest?trader_id=…&limit=N (30s)
GET  /api/statistics?trader_id=…           (30s)
GET  /api/positions/history?trader_id=…&limit=200 (×2 — PositionHistory does its own fetch)
GET  /api/audit/decisions?trader_id=…&limit=100   (when Decisions tab is opened)
```

The trader_id format observed in queries: `1ef40f05_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1779909574` — `<8-hex>_<UUID>_<model_provider>_<unix epoch>`.

## Layer 1: Source code map

### `TradersRoute` (route component)
- **File:** [AppRoutes.tsx:191-223](web/src/router/AppRoutes.tsx#L191-L223)
- **State:** none (delegates entirely to `AITradersPage`)
- **SWR:** `useSWR<TraderInfo[]>('traders-route', api.getTraders, { refreshInterval: 5000 })` — pre-warms the trader list before rendering, used for slug→id resolution on row clicks.
- **Renders:** `<AppChrome currentPage="traders" animateContent extraContent={showBeginnerOnboarding ? <BeginnerOnboardingPage /> : null}>` + `<AITradersPage onTraderSelect={...}>`.
- **`onTraderSelect` callback:** resolves trader by id, then `navigate(buildDashboardPath(getTraderSlug(trader)))`.

### `AITradersPage` (`/traders` content)
- **File:** [web/src/components/trader/AITradersPage.tsx](web/src/components/trader/AITradersPage.tsx) (802 LOC)
- **Props:** `{ onTraderSelect?: (traderId: string) => void }`
- **Imports:** `react`, `react-router-dom::useNavigate`, `swr::useSWR`, `../../lib/api`, `../../contexts/{AuthContext,LanguageContext}`, `../../i18n/translations::t`, `lucide-react::{Bot, Plus, MessageCircle}`, `sonner::toast`, `confirmToast` helper, plus the four modal components (`TraderConfigModal`, `ExchangeConfigModal`, `TelegramConfigModal`, `ModelConfigModal`), plus inline panels (`ConfigStatusGrid`, `TradersList`).
- **State (10):** `showCreateModal`, `showEditModal`, `showModelModal`, `showExchangeModal`, `showTelegramModal`, `editingModel`, `editingExchange`, `editingTrader`, `allModels`, `allExchanges`, `supportedModels`, `visibleTraderAddresses: Set<string>`, `visibleExchangeAddresses: Set<string>`, `copiedId`.
- **SWR:** `useSWR<TraderInfo[]>('traders', api.getTraders, { refreshInterval: 5000 })`.
- **Effects (2):** initial config load on auth ready; `window.addEventListener('agent-config-refresh', …)` (same pattern as SettingsPage — agent tool edits propagate without remount).
- **Derived data:** `configuredModels` filters `allModels` by `enabled || customApiUrl`. `configuredExchanges` filters by `enabled` (and the legacy aster/hyperliquid id check at line 140-145 — this checks `e.id === 'aster'` which is a leftover from when ids were the type name; with UUID ids today this branch can no longer fire).
- **Trader status counts:** `enabledModels.length`, `enabledExchanges.length` displayed at the top "1 ACTIVE_NODES / 1/1 ACTIVE" widget.

### `DashboardRoute` (route component)
- **File:** [AppRoutes.tsx:225-413](web/src/router/AppRoutes.tsx#L225-L413) — 189 LOC; the heaviest route component in the app
- **State (5):** `selectedTraderId`, `lastUpdate`, `decisionsLimit` (5/10/20/50/100), `accountPollOff`, `positionsPollOff`, `decisionsPollOff` (each gets latched off after 2 consecutive 4xx/5xx via `onErrorRetry`)
- **SWR (7 hooks):**
  | Key | Endpoint | Interval | Special |
  |-----|---|---|---|
  | `traders-dashboard` | `api.getTraders(true)` | 10s | — |
  | `exchanges-dashboard` | `api.getExchangeConfigs` | 60s | for chart routing |
  | `status-${traderId}` | `api.getStatus` | 15s | — |
  | `account-${traderId}` | `api.getAccount` | 15s | error-latch |
  | `positions-${traderId}` | `api.getPositions` | 15s | error-latch |
  | `decisions/latest-${traderId}-${limit}` | `api.getLatestDecisions` | 30s | error-latch |
  | `statistics-${traderId}` | `api.getStatistics` | 30s | — |
- **Slug → id resolution:** [line 262-279](web/src/router/AppRoutes.tsx#L262-L279) — when `?trader=<slug>` query changes, find the matching trader, set `selectedTraderId`; if no slug, default to `traders[0].trader_id`.
- **Effect — pollOff reset:** [line 238-242](web/src/router/AppRoutes.tsx#L238-L242) — switching trader resets all three pollOff latches.
- **Renders:** `<AppChrome currentPage="trader" animateContent><TraderDashboardPage … 19 props … /></AppChrome>`.

### `TraderDashboardPage` (`/dashboard` content)
- **File:** [web/src/pages/TraderDashboardPage.tsx](web/src/pages/TraderDashboardPage.tsx) (1154 LOC)
- **Props interface:** `TraderDashboardPageProps` ([line 99-119](web/src/pages/TraderDashboardPage.tsx#L99-L119)) — 19 fields, all data and one navigation callback. No SWR inside the page itself; pure presentational.
- **Helpers (top of file, lines 30-95):**
  - `getModelDisplayName(modelId)` — id → display name (DeepSeek / Qwen / Claude / uppercase fallback)
  - `getExchangeDisplayNameFromList(exchangeId, exchanges)` — UUID → "TYPE - account_name"
  - `getExchangeTypeFromList(exchangeId, exchanges)` — UUID → lowercase exchange_type, defaults to `'binance'` (chart fallback)
  - `isPerpDexExchange(exchangeType)` — hyperliquid|lighter|aster
  - `getWalletAddress(exchange)` — branch by exchange_type
  - `truncateAddress(address, startLen=6, endLen=4)`
- **State (~10):** `closingPosition`, `selectedChartSymbol`, `chartUpdateKey`, `chartSectionRef`, `showWalletAddress`, `copiedAddress`, `dashboardTab: 'overview' | 'decisions'`, `positionsPageSize`, `positionsCurrentPage`.
- **Derived computed at top of function body:**
  - `currentExchange = exchanges?.find(e => e.id === selectedTrader?.exchange_id)`
  - `walletAddress = getWalletAddress(currentExchange)`
  - `isPerpDex = isPerpDexExchange(currentExchange?.exchange_type)`
  - `isFutures = currentExchange?.exchange_type?.toLowerCase() === 'ninjatrader'` (Plan 4.3)
  - `currencyUnit = isFutures ? 'USD' : 'USDT'` (Plan 4.3)
  - `paginatedPositions = positions?.slice(...)` — page-size-bounded slice
- **Conditional renders (Plan 4.3 NT branches):**
  - `<th>Leverage</th>` + `<td>{pos.leverage}x</td>` — both wrapped in `{!isFutures && …}` ([line 755-762, 845-849](web/src/pages/TraderDashboardPage.tsx#L755-L849))
  - `<th>Liq</th>` + `<td>{formatPrice(pos.liquidation_price)}</td>` — both wrapped in `{!isFutures && …}` ([line 769-776, 864-868](web/src/pages/TraderDashboardPage.tsx#L769-L868))
  - 3× `<StatCard unit={currencyUnit}>` (line 559, 572, 588) — Plan 4.3 conditional USD/USDT
- **Early-exit renders:**
  - `tradersError` → "Connection Failed" empty state ([line 254-294](web/src/pages/TraderDashboardPage.tsx#L254-L294))
  - `traders && traders.length === 0` → "Empty dashboard" CTA pointing to /traders ([line 297-337](web/src/pages/TraderDashboardPage.tsx#L297-L337))
  - `!selectedTrader` (still loading) → skeleton ([line 340-365](web/src/pages/TraderDashboardPage.tsx#L340-L365))
- **Tab switcher:** Plan 4 Task 23.4 ([line 626-652](web/src/pages/TraderDashboardPage.tsx#L626-L652)) — `Overview` / `Decisions`. Decisions panel renders `<DecisionAudit traderId={…}>`.
- **Bottom section:** `{selectedTraderId && dashboardTab === 'overview' && <PositionHistory traderId={...} />}` ([line 1068-1081](web/src/pages/TraderDashboardPage.tsx#L1068-L1081)).

### Sub-component: `EmergencyFlatButton`
- **File:** [web/src/components/trader/EmergencyFlatButton.tsx](web/src/components/trader/EmergencyFlatButton.tsx) (123 LOC)
- **Props:** `{ traderId: string }`
- **State:** `open`, `busy`, `result: string | null`
- **API:** raw `fetch('/api/risk/force-flat?trader_id=...', POST)` with `Authorization: Bearer <localStorage.auth_token>`.
- **Test-ids:** `emergency-flat-button`, `confirm-flat-modal`, `confirm-flat-button`, `cancel-flat-button`, `emergency-flat-result`.

### Sub-component: `DecisionAudit`
- **File:** [web/src/components/trader/DecisionAudit.tsx](web/src/components/trader/DecisionAudit.tsx) (310 LOC)
- **Props:** `{ traderId: string }`
- **Types:** `DecisionActionRecord` (action/symbol/price/stop_loss/take_profit/confidence/reasoning), `DecisionAuditRow` (id/trader_id/cycle_number/timestamp/created_at/decisions[]/prompt_version/ai_model/ai_latency_ms/risk_check_passed/risk_check_error/execution_status/fill_price/fill_latency_ms), `FlatRow` (one row per nested action).
- **Helper:** `flattenDecisions(records)` — if a record has no actions, synthesizes a single `wait` row with zeros so the cycle still shows.
- **State:** `records`, `loading`, `error`, `expandedKey`.
- **Fetch:** `useEffect` on `traderId`, `fetch('/api/audit/decisions?trader_id=…&limit=100')`. Plain `fetch`, manual token construction.
- **Render:** 11-column `<table>` (Time, Symbol, Action, Entry, SL, TP, Confidence, Risk Check, Execution Status, Fill Price, Latency). Click row → expand reasoning panel.

### Sub-component: `ChartTabs`
- **File:** [web/src/components/charts/ChartTabs.tsx](web/src/components/charts/ChartTabs.tsx) (435 LOC)
- **Props:** `{ traderId, selectedSymbol?, updateKey?, exchangeId?, isFutures? }`
- **Module constants:**
  - `MARKET_CONFIG` ([line 28-70](web/src/components/charts/ChartTabs.tsx#L28-L70)) — 5 market types: `hyperliquid`, `crypto`, `stocks`, `forex`, `metals`. **No `futures` entry — NT exchanges fall through to `crypto` default.** This is the documented Plan 4.4 gap.
  - `INTERVALS` ([line 72-80](web/src/components/charts/ChartTabs.tsx#L72-L80)) — 1m/5m/15m/30m/1h/4h/1d
- **`getMarketTypeFromExchange()`** (line 83-89) — only matches `hyperliquid`; everything else → `crypto`. Means `ninjatrader` exchanges hit the crypto path with `defaultSymbol='BTCUSDT'`, defeating Plan 4.4 chart spec.
- **State (8):** `activeTab: 'equity' | 'kline'`, `chartSymbol`, `interval`, `symbolInput`, `marketType`, `availableSymbols`, `showDropdown`, `searchFilter`.
- **Effect:** auto-switch market type when `exchangeId` prop changes ([line 112-115](web/src/components/charts/ChartTabs.tsx#L112-L115)).
- **Backend call:** `GET /api/symbols?exchange=<exchange>` when `marketConfig.hasDropdown===true` (only Hyperliquid currently).

### Sub-component: `DecisionCard`
- **File:** [web/src/components/trader/DecisionCard.tsx](web/src/components/trader/DecisionCard.tsx) (481 LOC)
- **Props (typical pattern, not read in this pass):** `{ decision, language, onSymbolClick }`
- **Behavior:** renders one decision with symbol pill, action color (LONG green / SHORT red / HOLD gray / WAIT), confidence bar, expandable reasoning section.

### Sub-component: `PositionHistory`
- **File:** [web/src/components/trader/PositionHistory.tsx](web/src/components/trader/PositionHistory.tsx) (918 LOC — large)
- **Props:** `{ traderId }`
- **Own SWR:** fetches `/api/positions/history?trader_id=…&limit=200` independently of DashboardRoute. Hence the duplicate `positions/history` request observed in network capture.

### Sub-component: `GridRiskPanel`
- **File:** [web/src/components/strategy/GridRiskPanel.tsx](web/src/components/strategy/GridRiskPanel.tsx) (313 LOC)
- Only rendered when `status?.strategy_type === 'grid_trading'`. Inactive for AI-decision traders; documented for completeness.

### Inline `StatCard`
- **File:** [TraderDashboardPage.tsx:1088-1154](web/src/pages/TraderDashboardPage.tsx#L1088-L1154)
- **Props:** `{ title, value, unit?, change?, positive?, subtitle?, icon?, loading? }`
- Renders the 4 top tiles (Total Equity, Available Balance, Total P&L, Positions).

## Layer 2: Runtime DOM + network

### `/traders` page (initial render)

- **Screenshot:** [inventory_traders_list.png](inventory_traders_list.png)
- **Visible content:**
  - Header: "AI Traders / 1 ACTIVE_NODES / SYSTEM_READY"
  - Action buttons: `MODELS_CONFIG`, `EXCHANGE_KEYS`, `TELEGRAM_BOT`, `Create Trader`
  - Status grid: "AI Models / STANDBY / 1/1 ACTIVE" + "Exchanges / 1/1 ACTIVE" (NT shown by its "N" letter avatar)
  - Current Traders list: 1 row — `mnq sIM TEST` / `DeepSeek Model • NINJATRADER - Simtest` / status `RUNNING` / buttons `View, Edit (disabled), Stop, 在竞技场显示, ⋯`
- **Network requests (first 30s):** `/api/config`, `/api/my-traders`, `/api/models`, `/api/exchanges`, `/api/supported-models`, then `/api/my-traders` re-polled every 5s.
- **NEW observation:** the "Edit" button is disabled even though the user is the trader's owner. Likely because the trader is `RUNNING` (Edit only enabled while stopped). Verify in TraderConfigModal source — out of scope for this pass.
- **NEW observation:** the "在竞技场显示" button label is Chinese (literal: "Show in Arena/Leaderboard"). Despite `language='en'` (`🇺🇸` selector). Probably an i18n gap — the button text isn't keyed through `t()`. Cross-reference: 3 deleted pages (Competition/Leaderboard) but this feature has lingering UI text.

### `/dashboard?trader=mnq%20sIM%20TEST%20-1ef4` (after clicking View)

- **Screenshot:** [inventory_dashboard_overview.png](inventory_dashboard_overview.png)
- **URL encoding observation:** the slug is `mnq sIM TEST -1ef4` — note the space before `-1ef4`. Likely a trader_name with a trailing space. The slug format ([AppRoutes.tsx:48-51](web/src/router/AppRoutes.tsx#L48-L51)) is `${trader.trader_name}-${idPrefix}`; if `trader_name` ends with a space, the slug has a space + hyphen. Decodes correctly because `findTraderBySlug` uses `lastIndexOf('-')`. Cosmetic but worth noting.
- **DOM observations (Overview tab):**
  - Trader Header: avatar, name "mnq sIM TEST", ID prefix, Emergency Flat button (red), Trader Selector dropdown
  - Metadata line: `AI Model: DeepSeek` (blue pill), `Exchange: NINJATRADER - Simtest`, `Strategy: Balanced Strategy`, `Cycles: 4`, `Runtime: <not yet rendered, ref e284 empty>`
  - Debug bar: `SYSTEM_STATUS::ONLINE`, `LAST_UPDATE::6:50:50 PM`, `EQ::50000.00`, `PNL::0.00`
  - **4 StatCards:** Total Equity `50000.00 USD` (▼ 0.00%), Available Balance `50000.00 USD / 100.0% Free`, Total P&L `+0.00 USD ▲ +0.00%`, Positions `0 ACTIVE / Margin: 0.0%`. **Plan 4.3 working — all four cards show USD, not USDT.**
  - Tab switcher: `Overview` / `Decisions`
  - Below: Chart tabs + Positions table (empty: "no positions") + Recent Decisions panel on right + Position History at bottom

### `/dashboard` — Decisions tab

- **Screenshot:** [inventory_dashboard_decisions.png](inventory_dashboard_decisions.png)
- **Network request:** `/api/audit/decisions?trader_id=<full-id>&limit=100` ×2 (DecisionAudit fires both on mount and on re-render — see Open Questions)
- **DOM:** 11-column table, ~50 rows visible (last 100 cycles, all `wait` action with `0%` confidence, ✗ risk check). Click any row → expands reasoning panel.
- **NEW observation:** Every visible row has `Risk Check ✗`. With `risk_check_passed=false` on every cycle, no trade can fire. Plan 3 risk limits may be too tight for the current account state (kernel kill switch logic), or `min_confidence` threshold is suppressing all actions. **Worth investigating** — this is the most important runtime signal captured this pass.

### Console

Clean — only the global favicon 404. No React key warnings (Plan 4.9 fix confirmed in place). No SWR error chatter.

## Layer 3: Backend code path

### `GET /api/my-traders` → `handleTraderList` ([handler_trader.go](api/handler_trader.go))
- Auth: required
- Returns: `[]TraderInfo` — `{trader_id, trader_name, is_running, ai_model_id, exchange_id, strategy_id, strategy_name}` plus a few derived flags
- Backend reads from `store/trader.go` (User+Trader join) and consults `manager.TraderManager.GetTrader(id).IsRunning()` for live status

### `GET /api/exchanges` → `handleGetExchangeConfigs` (covered in Page 3 backend)

### `GET /api/status?trader_id=…` → `handleStatus`
- Returns: `{is_running, trader_id, call_count, runtime_minutes, strategy_type, grid_symbol?}`
- `strategy_type` is one of `"grid_trading"` | `"signal"` etc. — controls whether `GridRiskPanel` renders

### `GET /api/account?trader_id=…` → `handleAccount`
- Returns: `{balance, equity, unrealized_pnl, initial_balance, total_return_pct, total_pnl, total_pnl_pct, total_equity, available_balance, position_count, margin_used_pct}`

### `GET /api/positions?trader_id=…` → `handlePositions`
- Returns: `Position[]` (symbol, side, size/quantity, entry_price, mark_price, unrealized_pnl, leverage, liquidation_price)

### `GET /api/positions/history?trader_id=…&limit=N` → `handlePositionHistory`
- Returns closed positions; backed by `store/position_history.go`

### `GET /api/decisions/latest?trader_id=…&limit=N` → `handleLatestDecisions`
- Returns `DecisionRecord[]` — most recent N cycles, each containing nested decisions
- Used by both DashboardRoute (Recent Decisions panel) and indirectly informs the audit endpoint

### `GET /api/audit/decisions?trader_id=…&since=&limit=N` → `handleDecisionAudit` (Plan 4 Task 23.4)
- Auth: required
- Returns: `[]DecisionRecord` — full audit shape with `prompt_version`, `ai_model`, `ai_latency_ms`, `risk_check_passed`, `risk_check_error`, `execution_status`, `fill_price`, `fill_latency_ms`
- Underlying store: `store/decision.go::ListDecisionAudit(traderID, since, limit)`

### `GET /api/statistics?trader_id=…` → `handleStatistics`
- Returns: `{total_trades, winning_trades, win_rate, total_pnl, sharpe_ratio, max_drawdown}`

### `GET /api/equity-history?trader_id=…` → `handleEquityHistory`
- Returns timeseries for EquityChart
- Backed by `store/equity.go` (per-cycle equity snapshots)

### `POST /api/risk/force-flat?trader_id=…` → `handleForceFlat` (Plan 4 Task 23)
- File: [api/handler_risk.go](api/handler_risk.go)
- For non-ninjatrader brokers returns `{triggered: false, reason: "..."}`
- For ninjatrader: calls into the broker's force-flat path (CSV signal OR TCP frame depending on `NT_TRANSPORT`)
- Side effect: resets the daily PnL window via `kernel.risk_limits` package

### `POST /api/traders/:id/close-position` → `handleClosePosition`
- Body: `{symbol: string}`
- For NT: per the audit, returns HTTP 400 "Close via NT UI" because the CSV bridge has no manual-close path. NEEDS VERIFICATION — Page 4 user flow tests this only if user clicks the row-level Close button; that button is only rendered when `paginatedPositions.length > 0`. Currently `positions.length === 0`, so untested in this pass.

### `GET /api/klines?symbol=…&interval=…&limit=…&exchange=…` → `handleKlines`
- File: [api/handler_klines.go](api/handler_klines.go)
- **Page 4 trigger:** ChartTabs → AdvancedChart fires this when activeTab='kline'.
- **Known gap (Plan 4.5):** no NT/Databento branch — returns 500 for `exchange=ninjatrader`. Currently masked because ChartTabs maps NT to `binance` exchange by fallback, so a Binance kline is fetched when a NT trader is selected (wrong symbol, wrong data).

### `GET /api/symbols?exchange=…` → `handleSymbols`
- Returns `{symbols: [{symbol, name, category}]}` — only Hyperliquid actually populates the dropdown (`MARKET_CONFIG.hyperliquid.hasDropdown=true`)
- Known gap: returns 400 "Unsupported exchange" for NT

### `GET /api/traders/:id/grid-risk` → `handleGetGridRiskInfo`
- Only consumed when `strategy_type==='grid_trading'`. Currently inactive for the running NT trader.

## Layer 4: Cross-references + gaps + open questions

### Shipped fixes (Plan tags)

| Plan | File:line | What |
|---|---|---|
| Plan 4 Task 23 | `EmergencyFlatButton.tsx` (entire file) | Red kill-switch button + confirmation modal, POSTs `/api/risk/force-flat` |
| Plan 4 Task 23.4 | `TraderDashboardPage.tsx:149-152, 626-671`; `DecisionAudit.tsx` | Overview/Decisions tab switch + audit table |
| Plan 4.3 | `TraderDashboardPage.tsx:185-188, 559, 572, 588` | `isFutures` flag derived from `exchange_type==='ninjatrader'`; StatCard `unit=currencyUnit` |
| Plan 4.3 | `TraderDashboardPage.tsx:755-762, 769-776, 845-849, 864-868` | Hide Leverage + Liquidation columns when `isFutures` |
| Plan 4.9 | `DecisionAudit.tsx:75-83` (key derivation now uses `rowKey: \`${rec.id}-${i}\``) | React key uniqueness fix — verified no console warnings this pass |
| Plan 4.7.1 | `UserPreferencesPanel` (used on /agent, not Dashboard) | Placeholder; merged 2026-05-26 (commit b4b64e46) |
| Plan 1.5.6 | NT TCP layer | Write deadline + concurrent-write mutex (commit 3e4ee61c) — visible to Dashboard via `is_running=true` continuing to report cleanly |

### Known gaps (carried forward)

| Gap | Plan | File:line | Symptom | Scope |
|---|---|---|---|---|
| K-line chart shows Binance BTCUSDT for NT traders | Plan 4.4 | `ChartTabs.tsx:28-70, 83-89`; `handler_klines.go:48-78` | Selecting an NT trader and switching to K-line tab shows wrong data | ~720 LOC across Go, C#, frontend per Plan 4.4 deep spec |
| `/api/klines` 500 for `exchange=ninjatrader` | Plan 4.5 | `api/handler_klines.go` | Backend dependency for Plan 4.4 chart route | ~120 LOC, 60 min |
| `/api/symbols` 400 for `exchange=ninjatrader` | Plan 4.5 | `api/handler_symbols.go` | Symbol dropdown disabled for NT | bundled in Plan 4.5 |
| NT balance is $50k mock | Plan 4.11 | `trader/ninjatrader/trader.go:161-162` | Total Equity card always shows 50000.00 USD; doesn't reflect real NT account state | ~150 LOC + C# AddOn extension, 2 hr |

### NEW observations

| Description | File:line | Severity | Recommended scope |
|---|---|---|---|
| **All recent decisions have `risk_check_passed=false`** — visible as `Risk Check ✗` on every audit row | runtime; need to inspect `kernel.risk_limits` rejection reason | **Investigation** | Read `risk_check_error` from one of these rows — DecisionAudit captures it. Most likely cause: `min_confidence` threshold or daily-loss limit triggered by the kernel kill switch state. Worth investigating before next live test. |
| `/api/audit/decisions?trader_id=…&limit=100` fires TWICE on Decisions tab open (saw entries 166 + 167 in network log) | `DecisionAudit.tsx:99-…` (useEffect dep behavior under React.StrictMode) | Cosmetic + 2× network cost | StrictMode double-invokes effects in dev; verify if it's StrictMode-only or also in production by removing `<React.StrictMode>` in main.tsx — but don't ship that change |
| `/api/positions/history?trader_id=…&limit=200` fires TWICE on dashboard mount (entries 158 + 159) | DashboardRoute polls + PositionHistory has its own useSWR | Cosmetic + 2× network cost | Decide single source of truth: lift PositionHistory's SWR into DashboardRoute OR remove the polling from DashboardRoute. Note: this could be StrictMode again — verify. |
| The "Edit" button in `/traders` is disabled when trader is RUNNING but no tooltip explains why | `TradersList.tsx` (not read in this pass) + AITradersPage | UX | 10-min: add `title="Stop trader to edit configuration"` |
| The button "在竞技场显示" renders in Chinese with `language='en'` selected | `TradersList.tsx` | i18n gap | Find the literal in source, route through `t('showInCompetition', language)` |
| Trader name `mnq sIM TEST` has odd casing + trailing space producing slug `mnq sIM TEST -1ef4` | data (user-created) + `AppRoutes.tsx:48-51` slug builder | Cosmetic | Slug works correctly; no fix needed, but worth knowing |
| Trader header shows `Strategy: Balanced Strategy` — confirms the seeded "balanced" strategy is what's running | runtime confirmation | Info | n/a |
| Footer GitHub/Twitter/Telegram links have `href=""` (empty) — clicking does nothing | `SiteFooter.tsx` | Minor | 10-min: either populate or remove |
| ChartTabs `MARKET_CONFIG.crypto.defaultSymbol='BTCUSDT'` is used as the fallback for NT — wrong default for NQ | `ChartTabs.tsx:38-44` + `getMarketTypeFromExchange` line 83-89 | Plan 4.4 prerequisite | Add `futures` market type entry + `getMarketTypeFromExchange("ninjatrader") → 'futures'` |

### Open questions

- Are the 4 cycles visible (`Cycles: 4` in header) actually firing scan_interval polls, or are they stale from a prior session? Cross-check with `runtime_minutes` if non-empty (current snapshot showed empty ref `e284`).
- What does `risk_check_error` actually say? The audit table truncates it; need to expand one row.
- Does the Decisions tab use SWR or plain useEffect? Source shows useEffect — should be revisited because SWR would dedupe the StrictMode double-call.
- What's the relationship between `selectedTraderId` in DashboardRoute and the URL `?trader=slug` — does updating one update the other? The effect at AppRoutes.tsx:262-279 sets the id from the slug but the navigation back from `onTraderSelect` at line 401-407 only updates the URL — verify the URL change re-triggers the slug→id effect.
- Why does `currentExchange?.exchange_type?.toLowerCase()` instead of comparing case-insensitively at line 187 — does the backend already canonicalize?

### Cross-page dependencies

- **/traders → /dashboard:** View button uses `buildDashboardPath(getTraderSlug(trader))` to produce the slug-encoded URL
- **/dashboard → /strategy:** the trader header shows `Strategy: <name>` but does NOT link there. Switching to /strategy is a manual nav.
- **/dashboard → /settings:** the `Exchange:` and `AI Model:` labels are static text, not links. To edit them user must navigate to /settings manually.
- **/agent → /dashboard:** AgentChat tickers (Plan 2) and agent tools that read positions all hit `/api/positions?trader_id=…` — same endpoint dashboard polls.
- **/traders ↔ AgentChat:** both listen for `window 'agent-config-refresh'` event from AgentChat tool edits.
- **Plan 1.5 TCP bridge:** `/dashboard` is the only surface that surfaces NT bridge state — but only indirectly (via cycle count, status.is_running, etc.). NEW-2 observation: there is no explicit "TCP connected" badge anywhere on /dashboard. Plan 4.14 candidate.
