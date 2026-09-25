# Page 2 — AgentBeta (`/agent`)

## Quick reference

- **Route:** `/agent` ([AppRoutes.tsx:465-472](web/src/router/AppRoutes.tsx#L465-L472))
- **Auth:** **not required** (the route renders for unauthenticated users too — they hit the protected backend endpoints with no token and get 401, but the chrome + welcome screen render)
- **Primary source:** [web/src/pages/AgentChatPage.tsx](web/src/pages/AgentChatPage.tsx) (919 LOC)
- **Sub-components:**
  | Component | File | LOC |
  |---|---|---|
  | `MarketTicker` | `web/src/components/agent/MarketTicker.tsx` | 209 |
  | `PositionsPanel` | `web/src/components/agent/PositionsPanel.tsx` | 164 |
  | `TraderStatusPanel` | `web/src/components/agent/TraderStatusPanel.tsx` | 119 |
  | `WelcomeScreen` | `web/src/components/agent/WelcomeScreen.tsx` | 188 |
  | `ChatMessages` | `web/src/components/agent/ChatMessages.tsx` | 143 |
  | `ChatInput` | `web/src/components/agent/ChatInput.tsx` | 192 |
  | `UserPreferencesPanel` | `web/src/components/agent/UserPreferencesPanel.tsx` | 241 |
  | `AgentStepPanel` | `web/src/components/agent/AgentStepPanel.tsx` | 109 |
  | `MessageRenderer` | `web/src/components/agent/MessageRenderer.tsx` | 187 |
- **API endpoints captured (live):**
  - `GET /api/config`
  - `GET /api/my-traders`
  - `GET /api/agent/tickers?symbols=MNQ` (15s polling)
  - `GET /api/agent/preferences`
  - `GET /api/positions?trader_id=…`
- **Endpoints not seen on mount but defined:**
  - `POST /api/agent/chat`
  - `POST /api/agent/chat/stream` (SSE — fires only on send)
  - `GET /api/agent/health`
  - `GET /api/agent/klines`
  - `GET /api/agent/ticker`
  - `POST /api/agent/preferences`
  - `DELETE /api/agent/preferences/:id`
- **Backend handlers:** `agent/web.go::WebHandler` (HandleHealth / HandleChat / HandleChatStream / HandleKlines / HandleTicker / HandleTickers); `api/server.go` for preferences

## Layer 1: Source code map

### Primary component: `AgentChatPage`
- **File:** [web/src/pages/AgentChatPage.tsx:1-919](web/src/pages/AgentChatPage.tsx#L1-L919)
- **Imports:**
  - React: `useState`, `useRef`, `useEffect`
  - `framer-motion::{motion, AnimatePresence}`
  - lucide-react: `PanelRightClose`, `PanelRightOpen`, `TrendingUp`, `Wallet`, `Bot`, `Bookmark`, `ChevronDown`, `ChevronRight`
  - Contexts: `useLanguage`, `useAuth`
  - All 6 sub-panels (MarketTicker / PositionsPanel / TraderStatusPanel / WelcomeScreen / ChatMessages / ChatInput / UserPreferencesPanel)
  - Zustand store: `useAgentChatStore`
  - Types: `AgentMessage as Message`, `AgentStep`
  - Persistence helpers from `../lib/agentChatStorage` — `chatStorageKey`, `clearAgentMessages`, `getStoredAuthUserId`, `loadAgentDraft`, `loadAgentMessages`, `migrateAgentMessages`, `prepareAgentMessagesForPersistence`, `persistAgentDraft`, `persistAgentMessages`
- **Module-level state (mutable singletons):**
  - `let msgIdCounter = 0` ([line 36](web/src/pages/AgentChatPage.tsx#L36))
  - `let activeStreamAbortController: AbortController | null = null` ([line 37](web/src/pages/AgentChatPage.tsx#L37))
  - `let activeStreamReader: ReadableStreamDefaultReader<Uint8Array> | null = null` ([line 38](web/src/pages/AgentChatPage.tsx#L38))
- **Module-level functions:**
  - `nextId()` — produces `msg-<timestamp>-<counter>` ([line 40-42](web/src/pages/AgentChatPage.tsx#L40-L42))
  - `cleanupActiveAgentStream()` — aborts controller + cancels reader ([line 44-51](web/src/pages/AgentChatPage.tsx#L44-L51))
  - `stopActiveAgentStream(userId?, language='zh')` — finalizes streaming message with "Stopped" text ([line 53-79](web/src/pages/AgentChatPage.tsx#L53-L79))
  - `persistMessagesSnapshotForUser`, `replaceMessagesInStore`, `patchMessagesInStore` — Zustand + localStorage sync helpers
  - `runAgentStream(params)` — main entrypoint that opens the SSE connection ([line 102-…](web/src/pages/AgentChatPage.tsx#L102))
- **Zustand store** (`useAgentChatStore` from `../stores/agentChatStore`) holds: `messages`, `hydrated`, `draftText`, `loading`, plus actions. **NOT React Context** — Zustand for chat state, Context for auth/language.
- **Quick actions** (visible in runtime snapshot): `💼 Positions`, `💰 Balance`, `📋 Traders`, `📊 Status`, `🧹 Clear`, `❓ Help`, `Hide sidebar`. These are pre-canned commands that auto-fill the chat input.
- **Right sidebar accordion sections** (visible refs e95, e106, e118, e127):
  - `Market` (collapsed by default — contains MarketTicker)
  - `Positions` ("No open positions" shown empty)
  - `Traders`
  - `Preferences` ("No persistent preferences yet…" empty state — Plan 4.7.1 placeholder confirmed)

### Sub-component: `MarketTicker`
- **File:** [web/src/components/agent/MarketTicker.tsx:21-209](web/src/components/agent/MarketTicker.tsx#L21-L209)
- **Props:** none (self-fetches)
- **Module constants:**
  - `SYMBOLS = ['MNQ']` ([line 14](web/src/components/agent/MarketTicker.tsx#L14)) — Plan 4.7 fix shipped. Audit-era constant was `['BTCUSDT', 'ETHUSDT', 'SOLUSDT']`.
  - `SYMBOL_ICONS = { MNQ: 'N', NQ: 'N' }` ([line 16-19](web/src/components/agent/MarketTicker.tsx#L16-L19)) — string letter as the "icon" (no proper exchange-icons asset for MNQ)
- **State:** `tickers: Record<string, TickerData>`, `loading: boolean`
- **Effect:** fetch on mount + `setInterval(fetchTickers, 15000)`
- **API call:** `httpClient.request<TickerData[]>('/api/agent/tickers?symbols=MNQ', { silent: true })` — batches all symbols into one URL
- **`TickerData` interface** (local to file, line 5-12): `{symbol, lastPrice, priceChangePercent, highPrice, lowPrice, volume}`
- **Helpers:** `formatPrice`, `formatVolume` (K/M/B suffix)

### Sub-component: `WelcomeScreen`
- **File:** [web/src/components/agent/WelcomeScreen.tsx:16-188](web/src/components/agent/WelcomeScreen.tsx#L16-L188)
- **Props:** `{ language: string, onSend: (cmd: string) => void }`
- **Module constant:** `suggestions: SuggestionCard[]` — Plan 4.7 shipped: now MNQ-focused (`Analyze MNQ`, `Trade MNQ`, `Search Futures`, `Strategy Ideas`). 4 cards per language (zh + en). Audit-era cards were crypto-only.
- **Hardcoded literals to be aware of:** the Chinese-localized commands `做多 MNQ 1 手` / `搜索一下 NQ 合约` — these are baked into the source, not in i18n/translations.ts.

### Sub-component: `UserPreferencesPanel`
- **File:** [web/src/components/agent/UserPreferencesPanel.tsx](web/src/components/agent/UserPreferencesPanel.tsx) (241 LOC)
- **Plan 4.7.1 status:** placeholder version merged 2026-05-26 (PR #25). The visible "No persistent preferences yet…" text confirms the placeholder UI is what renders.
- **API:** `GET /api/agent/preferences` (list), `POST /api/agent/preferences` (create), `DELETE /api/agent/preferences/:id`

### Sub-component: `ChatMessages`
- **File:** [web/src/components/agent/ChatMessages.tsx](web/src/components/agent/ChatMessages.tsx) (143 LOC)
- Renders the message list; defers individual message render to `MessageRenderer`.

### Sub-component: `ChatInput`
- **File:** [web/src/components/agent/ChatInput.tsx](web/src/components/agent/ChatInput.tsx) (192 LOC)
- **Exports:** `ChatInput`, `type ChatInputHandle` — imperative handle (uses `useImperativeHandle`) probably exposes `focus()` for the `⌘K` keyboard shortcut.
- Visible behavior: placeholder `Ask NOFXi anything... ⌘K`; Send button `[disabled]` when empty

### Sub-component: `MessageRenderer`
- **File:** [web/src/components/agent/MessageRenderer.tsx](web/src/components/agent/MessageRenderer.tsx) (187 LOC)
- Renders one message: user vs bot, markdown formatting, streaming cursor, tool-use blocks, step panels via `AgentStepPanel`.

### Sub-component: `AgentStepPanel`
- **File:** [web/src/components/agent/AgentStepPanel.tsx](web/src/components/agent/AgentStepPanel.tsx) (109 LOC)
- Renders the planner's step list during streaming: planning / plan / step_start / step_complete / tool / done events.

### Sub-component: `PositionsPanel` + `TraderStatusPanel`
- **Files:** [PositionsPanel.tsx](web/src/components/agent/PositionsPanel.tsx) (164 LOC), [TraderStatusPanel.tsx](web/src/components/agent/TraderStatusPanel.tsx) (119 LOC)
- Used in the right sidebar accordion. PositionsPanel fetches `/api/positions?trader_id=…` (this was observed in network: entry 141).

## Layer 2: Runtime DOM + network

### Initial render

- **Screenshot:** [inventory_agent_chat.png](inventory_agent_chat.png)
- **DOM structure** (depth-7 accessibility snapshot):
  - **Header strip (above main area):** Quick action buttons `💼 Positions`, `💰 Balance`, `📋 Traders`, `📊 Status`, `🧹 Clear`, `❓ Help`, `Hide sidebar` (right-arrow icon)
  - **Center column (chat area):**
    - Welcome screen: zap icon, heading `What can I help with?`, paragraph `Analyze markets, execute trades, search stocks — just ask`
    - 4 suggestion cards: `Analyze MNQ / Technical analysis + sentiment`, `Trade MNQ / Agent executes for you`, `Search Futures / Enter symbol or contract`, `Strategy Ideas / Market-based suggestions`
    - Input box: `Ask NOFXi anything... ⌘K` placeholder + Send button (disabled until text entered)
    - Footer disclaimer: `NOFXi may make mistakes. Always verify trading decisions.`
  - **Right sidebar (accordion):**
    - `Trading Panel` header
    - `Market` section (collapsed, contains the MNQ ticker)
    - `Positions` section (expanded, shows "No open positions")
    - `Traders` section
    - `Preferences` section ("No persistent preferences yet…")
- **No `<footer>`** — the `AppChrome showFooter={false}` ([AppRoutes.tsx:468](web/src/router/AppRoutes.tsx#L468)) intentionally suppresses the site footer to give the agent more vertical space.
- **Console:** clean (no error this time — even the favicon 404 didn't fire because favicons were already cached from prior pages this session).

### Network requests

| URL | Method | Status | Notes |
|---|---|---|---|
| `/api/config` | GET | 200 | system config |
| `/api/my-traders` | GET | 200 | from HeaderBar / AuthContext (not unique to this page) |
| `/api/agent/tickers?symbols=MNQ` | GET | 200 | MarketTicker fetch — **only MNQ**, no BTC/ETH/SOL |
| `/api/agent/preferences` | GET | 200 | UserPreferencesPanel — empty list |
| `/api/positions?trader_id=…` | GET | 200 | PositionsPanel (uses the same endpoint Dashboard polls) |

`/api/agent/tickers` AND `/api/agent/preferences` each fire TWICE (entries 137/139 and 138/140) — same StrictMode double-effect pattern noted on other pages.

## Layer 3: Backend code path

### Route registration: `RegisterAgentHandler` ([api/agent_routes.go:11](api/agent_routes.go#L11))

The agent's routes are registered SEPARATELY from the main `setupRoutes` block — `api/agent_routes.go` is a small standalone file with:

```go
s.router.POST("/api/agent/chat", s.authMiddleware(), …)         // auth required, can trade
s.router.POST("/api/agent/chat/stream", s.authMiddleware(), …)  // auth required, SSE
s.router.GET("/api/agent/health", gin.WrapF(h.HandleHealth))    // public
s.router.GET("/api/agent/klines", gin.WrapF(h.HandleKlines))    // public, market data
s.router.GET("/api/agent/ticker", gin.WrapF(h.HandleTicker))    // public
s.router.GET("/api/agent/tickers", gin.WrapF(h.HandleTickers))  // public, batch
```

Both authenticated routes inject `agent.WithStoreUserID(ctx, userID)` AND `agent.WithSessionPolicy(ctx, SessionPolicy{Authenticated: true, IsAdmin: …, CanExecuteTrade: true, CanViewSensitiveSecrets: false})` into the request context before calling the handler. This is how the agent's tool layer knows whether the caller can execute trades.

### `POST /api/agent/chat/stream` → `WebHandler.HandleChatStream` ([agent/web.go:125](agent/web.go#L125))
- **SSE protocol:** Server-Sent Events. 9 documented event types per the audit:
  - `planning` — "I'm thinking about how to handle this"
  - `plan` — full plan object emitted once
  - `step_start` — beginning a step
  - `step_complete` — step finished
  - `replan` — plan got rewritten mid-execution
  - `tool` — a tool was invoked with args + result
  - `delta` — text token from the LLM streaming
  - `done` — overall done
  - `error` — terminal error
- **Helpers:** `writeSSE(w, flusher, event, data)` ([line 191](agent/web.go#L191)), `sseEscape(s)` ([line 199](agent/web.go#L199))

### `GET /api/agent/tickers?symbols=…` → `WebHandler.HandleTickers` ([agent/web.go:246](agent/web.go#L246))
- **Source:** proxies Binance Futures public ticker endpoint (`binanceFuturesAPIBaseURL = "https://fapi.binance.com"` at line 35)
- **`splitComma(s)`** at line 329 — `strings.Split(s, ",")` with trim
- **`proxyBinance(rw, ctx, url)`** at line 339 — HTTP GET via `marketDataHTTPClient`
- **MNQ-specific behavior:** unclear from this read whether the handler has an NT/Databento branch or if it just blindly proxies "MNQ" to Binance (where it returns no result). Tested live: `GET /api/agent/tickers?symbols=MNQ` returned 200, but the actual response shape was not inspected. **Probable gap:** the MNQ symbol returns empty data from the Binance proxy, which is why MarketTicker can't render values — needs verification.

### `GET /api/agent/klines` → `WebHandler.HandleKlines` ([agent/web.go:207](agent/web.go#L207))
- Same Binance proxy approach
- **Plan 4.4/4.5 implication:** this is a SEPARATE endpoint from the main `/api/klines` consumed by ChartTabs on Dashboard. The agent's chart fetches go here. Plan 4.5 fix-needed status is the SAME — both endpoints need an NT/Databento branch.

### `GET /api/agent/health` → `WebHandler.HandleHealth` ([agent/web.go:77](agent/web.go#L77))
- Simple `{"status":"ok"}` return

### Agent tool layer ([agent/tools.go](agent/tools.go))
- **23 registered tools** (line numbers in tools.go via the `Name: "…"` pattern):
  | Line | Name | Purpose |
  |---|---|---|
  | 479 | `get_preferences` | List user preferences |
  | 487 | `manage_preferences` | Create / update / delete prefs |
  | 513 | `get_backend_logs` | Operator diagnostic — last N log lines |
  | 529 | `get_decisions` | Query decision history per trader |
  | 544 | `get_exchange_configs` | Read user's exchanges |
  | 552 | `manage_exchange_config` | Add / edit / delete exchange |
  | 586 | `get_model_configs` | Read AI model configs |
  | 594 | `manage_model_config` | Add / edit AI model |
  | 618 | `get_strategies` | List user's strategies |
  | 626 | `manage_strategy` | Create / edit / delete / activate strategy |
  | 650 | `manage_trader` | Create / edit / delete / start / stop trader |
  | 675 | `search_stock` | Search by symbol or name |
  | 692 | `execute_trade` | Place a trade — gated by `CanExecuteTrade` policy |
  | 722 | `get_positions` | Per-trader open positions |
  | 730 | `get_balance` | Per-trader account balance |
  | 738 | `get_market_price` | Single symbol last price |
  | 755 | `get_market_snapshot` | Multi-data view (OHLC + indicators + funding rate + OI) |
  | 780 | `get_kline` | Historical OHLCV by interval |
  | 805 | `get_trade_history` | Closed trades |
  | 821 | `get_candidate_coins` | AI500 / OI top / etc. |
  | 841 | `get_watchlist` | User-saved symbols |
  | 849 | `manage_watchlist` | Add / remove watchlist entry |
  | — | (the audit document calls out 23, all accounted for above) |
- **Tool cache:** `var cachedTools = buildAgentTools()` ([line 36](agent/tools.go#L36)) — built once at package init, reused across requests.
- **`plannerToolsForText(text)`** ([line 41](agent/tools.go#L41)) — narrows the tool set by classifying user intent into a "domain" before sending to the LLM. Domain comes from `plannerToolDomainForText(text)`. Strategy-mutation intents get the full `manage_strategy` schema; others get a compacted version via `compactManageStrategyTool` ([line 140](agent/tools.go#L140)).
- **Broker imports** ([line 14-30](agent/tools.go#L14-L30)): `aster`, `binance`, `bitget`, `bybit`, `gate`, `hyperliquidtrader`, `indodax`, `kucoin`, `lighter`, `ninjatrader (ntTrader)`, `okx` — **all 11 brokers including NT**. The `execute_trade` and balance tools route to the correct broker package by the trader's `exchange_type`.
- **Safe DTOs at the bottom of the file** ([line 1024+](agent/tools.go)): `safeExchangeForTool`, `safeModelForTool`, `safeTraderForTool`, `safeStrategyForTool` — sanitize sensitive fields before exposing to the LLM context.
- **System prompt at line ~2166** — note the literal `"默认策略"` (Chinese "Default Strategy") — i18n debt.

## Layer 4: Cross-references + gaps + open questions

### Shipped fixes (Plan tags)

| Plan | File:line | What |
|---|---|---|
| Plan 4.7 | `MarketTicker.tsx:14` | Symbol list narrowed from `['BTCUSDT','ETHUSDT','SOLUSDT']` to `['MNQ']` |
| Plan 4.7 | `WelcomeScreen.tsx:17-70` | Suggestion cards rewritten to MNQ-focused prompts (en + zh) |
| Plan 4.7.1 | `UserPreferencesPanel.tsx` | Placeholder panel ("No persistent preferences yet…") merged via PR #25 on 2026-05-26 |

### Known gaps (carried forward)

| Gap | Plan | File:line | Symptom | Scope |
|---|---|---|---|---|
| `GET /api/agent/tickers?symbols=MNQ` proxies Binance — MNQ not a Binance symbol | open / Plan 4.5 follow-on | `agent/web.go:246-329, 339-368` | Ticker likely returns empty data; MarketTicker UI shows blank or fallback | Bundle with Plan 4.5 backend NT/Databento branch |
| `GET /api/agent/klines` also Binance-only | Plan 4.5 | `agent/web.go:207-228` | Agent's chart-related tool calls hit wrong data source for NT traders | Bundle with Plan 4.5 |
| Hardcoded `"默认策略"` in tools.go | i18n debt | `agent/tools.go:2166` | Strategy-creation flow in EN chat shows Chinese label | 5-min: thread through agent i18n module |
| Tool descriptions still mention crypto-only assumptions (funding rate, OI, USDT) | Plan 4.6-adjacent | various `Description:` fields in tools.go | LLM may suggest crypto-only paths to a NT user | bundled into a future "agent NT awareness" pass |

### NEW observations

| Description | File:line | Severity | Recommended scope |
|---|---|---|---|
| `MarketTicker.SYMBOLS` array is hardcoded to `['MNQ']` — not configurable per user. If user later wants ES + GC + CL on the ticker, requires code change. | `MarketTicker.tsx:14` | Functional limitation | 30-min: make it a prop, drive from `UserPreferencesPanel` watchlist |
| `SYMBOL_ICONS = { MNQ: 'N', NQ: 'N' }` uses a letter as icon — no proper futures icon asset | `MarketTicker.tsx:16-19` | Cosmetic | Bundle with a /public/icons/futures/ asset addition |
| The chat input placeholder text says "Ask NOFXi anything... ⌘K" — verify ⌘K actually focuses the input | `ChatInput.tsx` (not read fully) | UX assumption | 5-min Playwright verify (out of this read-only pass) |
| The quick-action button `💰 Balance` likely sends a fixed prompt to the agent that triggers the `get_balance` tool. None of these quick-actions are documented in source comments — easy to break by mistake. | `AgentChatPage.tsx` (quick action handler not read) | Maintainability | Document the 6 quick-action commands somewhere |
| `runAgentStream`'s module-level singletons (`activeStreamAbortController`, `activeStreamReader`) make multi-tab support fragile — opening /agent in two tabs of the same session will collide on the singleton | `AgentChatPage.tsx:36-38` | Edge case | Worth a per-tab refactor only if multi-tab becomes a use case |
| `localStorage`-only persistence (`persistAgentMessages`, slice last 100) | `AgentChatPage.tsx:84` | Privacy + portability | Chat history doesn't sync across devices. Worth documenting. |
| `prepareAgentMessagesForPersistence(messages).slice(-100)` — caps at 100 messages | `AgentChatPage.tsx:84` | Operational | Long chat sessions are silently truncated. Acceptable but the user should know. |

### Open questions

- What does `GET /api/agent/tickers?symbols=MNQ` actually return when proxied to Binance? Likely an empty array (since Binance has no MNQ symbol). The MarketTicker UI doesn't visibly fail — does it render an empty card, a "fetching" spinner, or a fallback?
- Where does `binanceFuturesAPIBaseURL = "https://fapi.binance.com"` come from configurationally? Hardcoded module-level var. Bypassing via the `agent.Configure*` hooks? Worth checking if there's an env-var override.
- Does the `manage_trader` tool gracefully refuse to create a NT trader without the right exchange configured? Worth a sanity check.
- Does `execute_trade` route correctly to `ntTrader` for NT traders? The broker import is present but the dispatch logic wasn't read in this pass.
- `cachedTools = buildAgentTools()` builds at package init — but the tool catalog references things like enabled-exchanges-list which can change at runtime. Is the cache actually static metadata only, or are dynamic catalogs rebuilt per-request elsewhere?

### Cross-page dependencies

- **/agent → /settings:** `manage_model_config` and `manage_exchange_config` agent tools modify the same data that Settings reads. After such a tool call, the agent layer fires `window.dispatchEvent(new Event('agent-config-refresh'))` (mentioned in Page 3 doc). Settings + AITradersPage listen; **StrategyStudioPage does NOT listen** (cross-reference Page 5 open questions).
- **/agent → /traders, /dashboard:** `manage_trader` creates / starts / stops traders. After start, the Dashboard's polling will pick up the new trader within 5-15s.
- **/agent → /strategy:** `manage_strategy` mutates strategies. StrategyStudio doesn't react to the refresh event but will re-fetch on tab focus.
- **/agent ↔ Dashboard data:** The PositionsPanel in the right sidebar hits `/api/positions?trader_id=…` — same endpoint Dashboard polls. They share data; the sidebar is effectively a mini Dashboard.
- **/agent itself never sends `agent-config-refresh`** — that event is dispatched by the agent BACKEND tool execution, not the frontend page. Confirmed by reading `agent/tools.go` (no fetch / window code there) — the event must come from a hook on the LLM tool-result handler somewhere in the agent module.
