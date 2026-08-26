# Page 3 — Config / Settings (`/settings`)

## Quick reference

- **Route:** `/settings`
- **Primary source:** [web/src/pages/SettingsPage.tsx](web/src/pages/SettingsPage.tsx) (649 LOC)
- **Mounted at:** [AppRoutes.tsx:473-484](web/src/router/AppRoutes.tsx#L473-L484) (auth-gated; unauth → /login)
- **Sub-components:** `ModelConfigModal` (1313 LOC), `ExchangeConfigModal` (925 LOC), `TelegramConfigModal` (515 LOC), inline `configBadge()` helper
- **Tabs:** 4 — `account` (default), `models`, `exchanges`, `telegram`
- **API endpoints called (live, captured 2026-05-27):**
  - `GET /api/config` (system config / encryption flag)
  - `GET /api/models` (configured AI models)
  - `GET /api/supported-models` (template list)
  - `GET /api/exchanges` (configured exchanges)
  - `PUT /api/user/password` (direct fetch, not via api lib)
  - `PUT /api/models` (model save)
  - `POST /api/exchanges` (create)
  - `PUT /api/exchanges` (update)
  - `DELETE /api/exchanges/:id`
  - `GET /api/server-ip` (Binance only, modal-internal)
  - `GET /api/telegram` (Telegram modal)
  - `POST /api/telegram` / `POST /api/telegram/model` / `DELETE /api/telegram/binding`
- **Backend handlers:** `handler_exchange.go`, `handler_ai_model.go`, `handler_telegram.go`, `handler_user.go`

## Layer 1: Source code map

### Primary component: `SettingsPage`

- **File:** [web/src/pages/SettingsPage.tsx:38-649](web/src/pages/SettingsPage.tsx#L38-L649)
- **Imports:**
  - React: `useState`, `useEffect`
  - `sonner` → `toast`
  - `lucide-react` → `User`, `Cpu`, `Building2`, `MessageCircle`, `Eye`, `EyeOff`, `ChevronRight`, `Plus`, `Pencil`
  - `../contexts/AuthContext` → `useAuth`
  - `../contexts/LanguageContext` → `useLanguage`
  - `../lib/api` → `api`
  - `../components/trader/ExchangeConfigModal` → `ExchangeConfigModal`
  - `../components/trader/TelegramConfigModal` → `TelegramConfigModal`
  - `../components/trader/ModelConfigModal` → `ModelConfigModal`
  - `../types` → `Exchange`, `AIModel`
- **Local helper:** `configBadge(label, active)` at [line 24-36](web/src/pages/SettingsPage.tsx#L24-L36) — pill chip used to show "API Key", "Secret", "TCP Bridge", etc.
- **Type alias:** `type Tab = 'account' | 'models' | 'exchanges' | 'telegram'` (line 22)
- **State variables (10):**
  - `activeTab: Tab` (default `'account'`)
  - **Account:** `newPassword: string`, `showPassword: boolean`, `changingPassword: boolean`
  - **Models:** `configuredModels: AIModel[]`, `supportedModels: AIModel[]`, `showModelModal: boolean`, `editingModel: string | null`
  - **Exchanges:** `exchanges: Exchange[]`, `showExchangeModal: boolean`, `editingExchange: string | null`
  - **Telegram:** `showTelegramModal: boolean`
- **Effects (2):**
  - `useEffect [activeTab]` (line 77-86) — lazy-fetches model/exchange data only when the corresponding tab is visited. Toast error on fetch failure.
  - `useEffect []` (line 88-96) — listens for the custom `agent-config-refresh` window event and re-fetches models + exchanges. Wired by AgentChatPage so an agent-tool config edit propagates to Settings without remount.
- **Callbacks (5):** `refreshModelConfigs`, `refreshExchangeConfigs`, `handleChangePassword`, `handleSaveModel`, `handleDeleteModel`, `handleSaveExchange`, `handleDeleteExchange`. None are wrapped in `useCallback` — they reallocate every render. Not visible perf issue at this LOC.
- **Conditional renders:**
  - 4 tab content blocks ([line 356-599](web/src/pages/SettingsPage.tsx#L356-L599)), guarded by `activeTab === '<name>'`
  - 3 modals ([line 604-646](web/src/pages/SettingsPage.tsx#L604-L646)), guarded by `showModelModal`/`showExchangeModal`/`showTelegramModal`
  - Empty-state cards (`No AI models configured yet`, `No exchange accounts connected yet`)
  - **NinjaTrader-aware branch** (line 508-510, 532-558) — when `exchange.exchange_type === 'ninjatrader'`, renders a single "TCP Bridge" badge instead of the API-key / Secret / Passphrase chips used for crypto exchanges.

### Sub-component: `ExchangeConfigModal`

- **File:** [web/src/components/trader/ExchangeConfigModal.tsx:157-925](web/src/components/trader/ExchangeConfigModal.tsx#L157-L925)
- **Props interface:** `ExchangeConfigModalProps` at [line 37-63](web/src/components/trader/ExchangeConfigModal.tsx#L37-L63) — `onSave` is an 18-argument callback. Eighteen.
- **Module constant:** `SUPPORTED_EXCHANGE_TEMPLATES` at [line 23-35](web/src/components/trader/ExchangeConfigModal.tsx#L23-L35) — array of 11 exchanges. Categories: 7 CEX (binance, bybit, okx, bitget, gate, kucoin, indodax), 3 DEX (hyperliquid, aster, lighter), 1 FUTURES (ninjatrader).
- **Inline sub-components:**
  - `StepIndicator({currentStep, labels})` (line 66-97) — 2-dot progress bar.
  - `ExchangeCard({template, selected, onClick, disabled})` (line 101-155) — grid card with icon + name + type badge.
- **State variables (~20):** `currentStep` (0|1), `selectedExchangeType`, `accountName`, `apiKey`, `secretKey`, `passphrase`, `testnet`, `showGuide`, `serverIP`, `loadingIP`, `copiedIP`, `webCryptoStatus`, `showBinanceGuide`, plus per-exchange bundles:
  - Aster: `asterUser`, `asterSigner`, `asterPrivateKey`
  - Hyperliquid: `hyperliquidWalletAddr`
  - Lighter: `lighterWalletAddr`, `lighterApiKeyPrivateKey`, `lighterApiKeyIndex`
  - NinjaTrader: `ntDataDir`, `ntInstrumentName` (default `'MNQ'`), `ntDefaultContractQty` (default `1`)
  - `secureInputTarget`, `isSaving`
- **External lookups (referral URLs):** `exchangeRegistrationLinks` table at [line 214-225](web/src/components/trader/ExchangeConfigModal.tsx#L214-L225) — 10 entries (no `ninjatrader` entry — NT has no signup URL).
- **Edit-mode populate effect:** [line 228-243](web/src/components/trader/ExchangeConfigModal.tsx#L228-L243). Note: reads `selectedExchange.apiKey` / `selectedExchange.secretKey` / `selectedExchange.asterPrivateKey` / `selectedExchange.lighterPrivateKey` directly — but the backend `SafeExchangeConfig` ([handler_exchange.go:28-50](api/handler_exchange.go#L28-L50)) never returns these fields in `GET /api/exchanges`. They will always be `undefined`, so the populate clears the inputs to `''`. This is by design (users re-enter secrets on edit), but the code looks like it's intending something different — see Open Questions.
- **Form sections gated by `currentExchangeType`** — separate JSX blocks for each exchange type (binance/bybit/okx/bitget/gate/kucoin/hyperliquid/aster/lighter/indodax/ninjatrader). NT block adds DataDir / Instrument / DefaultContractQty inputs and skips API-key fields.

### Sub-component: `ModelConfigModal`

- **File:** [web/src/components/trader/ModelConfigModal.tsx:33-1313](web/src/components/trader/ModelConfigModal.tsx#L33-L1313)
- LOC count alone (1313) signals this is the biggest single non-page file in the project. Carries per-provider setup-guide content for every supported model (DeepSeek / Claude / OpenAI / Gemini / Grok / Qwen / Kimi / MiniMax / claw402).
- Two-stage entry pattern: (1) select provider via cards, (2) configure (api_key, optional custom_api_url, optional custom_model_name).
- `onSave(modelId, apiKey, customApiUrl?, customModelName?)` — encrypts via `CryptoService` when `transport_encryption=true` in `/api/config`.

### Sub-component: `TelegramConfigModal`

- **File:** [web/src/components/trader/TelegramConfigModal.tsx](web/src/components/trader/TelegramConfigModal.tsx) (515 LOC)
- **Props:** `{onClose, language}` (line 44-49).
- **Two named handlers:** `handleSaveToken` (line 80) — saves token + model_id; `handleSave` (line 468) — second-stage save (only model_id update).
- Walks user through: 1) create bot via @BotFather, 2) paste token, 3) pick AI model, 4) send `/start` from their Telegram client to bind the chat_id.

## Layer 2: Runtime DOM + network

### Initial render — Account tab (default)

- **URL:** `http://localhost:3000/settings`
- **Snapshot ref:** `.playwright-mcp/page-2026-05-27T23-44-58-963Z.yml` + later snapshots
- **Visible:** 4-tab pill bar + Account section: read-only email (`johnwick2921@gmail.com`) + Change Password form.
- **Network requests:**
  | URL | Method | Status | Purpose |
  |-----|--------|--------|---------|
  | `/api/config` | GET | 200 | System config (used by HeaderBar / ConfirmDialog providers; not a Settings-tab-specific request) |
- **Console:** clean (only the global favicon 404).
- **Note:** Account tab does NOT fetch from any endpoint on mount — it reads `user.email` from `useAuth()` which is hydrated during app boot.

### Interaction 1 — Click "AI Models" tab

- Triggers `setActiveTab('models')` → useEffect fires `refreshModelConfigs()`.
- **Screenshot:** [inventory_settings_ai_models.png](inventory_settings_ai_models.png)
- **New network requests:**
  | URL | Method | Status |
  |-----|--------|--------|
  | `/api/models` | GET | 200 |
  | `/api/supported-models` | GET | 200 |
- **Visible content (live data):** "3 models configured" — claw402 (Active), DeepSeek (Active), openai (Inactive).
- Each row: lucide `Cpu` icon, model name, provider, badges (API Key, Custom Model, Base URL), status pill (`Active` emerald / `Inactive` zinc), `Pencil` edit indicator.

### Interaction 2 — Click "Exchanges" tab

- **Screenshot:** [inventory_settings_exchanges.png](inventory_settings_exchanges.png)
- **New network request:** `/api/exchanges` GET 200
- **Visible content:** "1 account connected" — `Simtest / ninjatrader / TCP Bridge` row. The TCP Bridge badge confirms the NT-specific branch at SettingsPage:532-535 is firing.

### Interaction 3 — Click "Add Exchange"

- Opens `ExchangeConfigModal` at step 0.
- **Screenshot:** [inventory_settings_exchange_modal.png](inventory_settings_exchange_modal.png)
- **DOM observations:**
  - Step indicator: "1 Select Exchange" (active) → "2 Configure"
  - "1. Environment check" panel — WebCrypto availability + transport encryption warning (currently shows "Transport encryption disabled" because `TRANSPORT_ENCRYPTION` env var is not set on this host)
  - "Centralized Exchanges" grid: 7 cards (binance, bybit, okx, bitget, gate, kucoin, indodax)
  - "Decentralized Exchanges" grid: 3 cards (hyperliquid, aster, lighter)
  - "Futures (CME)" section: 1 card (NinjaTrader)
- **No network requests fired** on modal open (form is fully client-side until submit).

### Interaction 4 — Click "Telegram" tab

- **Screenshot:** [inventory_settings_telegram.png](inventory_settings_telegram.png)
- Displays text blurb + "Configure Telegram Bot" CTA. **No network request on mount** — the modal does its own fetch only after the CTA opens it.

### Console messages (steady state)

Only the global favicon 404 (×2). No app errors, no React warnings, no unhandled promise rejections.

## Layer 3: Backend code path

### `GET /api/models`

- **Route:** [server.go:139-143](api/server.go#L139-L143) (authenticated)
- **Handler:** `handleGetModelConfigs` at [handler_ai_model.go:51-127](api/handler_ai_model.go#L51-L127)
- **Store call:** `s.store.AIModel().List(userID)` — returns `[]store.AIModel`
- **Returns:** `[]SafeModelConfig` (id / name / provider / enabled / has_api_key / customApiUrl / customModelName / walletAddress / balanceUsdc)
- **claw402 special case** (line 96-105): if `provider == "claw402"`, derive the wallet address from the stored private key and fetch USDC balance via `wallet.QueryUSDCBalanceStr`.
- **Default fallback:** if user has zero rows in `ai_models` (or all filtered by `IsVisibleAIModel`), returns the 8-provider default list with `enabled: false`.

### `PUT /api/models`

- **Route:** [server.go:144-156](api/server.go#L144-L156)
- **Handler:** `handleUpdateModelConfigs` at [handler_ai_model.go:130-…](api/handler_ai_model.go#L130) (truncated; ~200+ LOC)
- **Encryption switch:** based on `cfg.TransportEncryption` flag from `config.Get()`. When true, expects `crypto.EncryptedPayload`; decrypts via `s.cryptoHandler.cryptoService.DecryptSensitiveData`.
- **SSRF guard:** `security.ValidateURL(cleanURL)` on any `custom_api_url` ([line 193-200](api/handler_ai_model.go#L193-L200)).
- **Side effect:** rebuilds the affected traders' AI clients (via `tradersToReload` set passed back to `manager.TraderManager`). Triggers a `agent-config-refresh` window event listener pattern but server-side.

### `GET /api/exchanges`

- **Route:** authenticated, in the protected group
- **Handler:** `handleGetExchangeConfigs` at [handler_exchange.go:125-153](api/handler_exchange.go#L125-L153)
- **Store call:** `s.store.Exchange().List(userID)`
- **Visibility filter:** `store.IsVisibleExchange(exchange)` — drops disabled/template rows. Defined in [store/visibility.go](store/visibility.go).
- **Returns:** `[]SafeExchangeConfig` (sanitized — API keys never leave the server).

### `POST /api/exchanges`

- **Handler:** `handleCreateExchange` (truncated in this read — register at [server.go:228-244](api/server.go#L228-L244))
- **Request:** `CreateExchangeRequest` ([handler_exchange.go:100-122](api/handler_exchange.go#L100-L122)) — includes the 3 NT fields (`NTDataDir`, `NTInstrumentName`, `NTDefaultContractQty`).
- **Encryption switch:** same pattern as `PUT /api/models`. **Plan 4.2 fix** in `web/src/lib/api/config.ts:108-116` — frontend skips the encryption envelope when `exchange_type === 'ninjatrader'` because NT has no API key to encrypt.

### `PUT /api/exchanges`, `DELETE /api/exchanges/:id`

- Symmetric — update / delete by id. Encryption switch applies to PUT; DELETE does not need a body.

### `PUT /api/user/password`

- **Direct `fetch()` call** from SettingsPage:106-113 (does NOT use the `api` helper). Constructs Authorization header by hand from `localStorage.getItem('auth_token')`.
- **Handler:** `handleChangePassword` ([handler_user.go](api/handler_user.go)), registered at [server.go:135-138](api/server.go#L135-L138).
- **Body:** `{"new_password": "<string, min 8 chars>"}`.
- **No old-password verification on the backend** — the JWT alone is the auth signal. Confirmed by reading the route doc string in `server.go` and the line of SettingsPage:99-103 that only validates length (≥8).

### Telegram endpoints

- `GET /api/telegram` → `handler_telegram.go::handleGetTelegramConfig` — returns `{bot_token, model_id, chat_id}`.
- `POST /api/telegram` — set token + model.
- `POST /api/telegram/model` — update model only.
- `DELETE /api/telegram/binding` — clear `chat_id` for re-bind.
- Triggered only after the user opens the modal; SettingsPage itself never calls these.

## Layer 4: Cross-references + gaps + open questions

### Shipped fixes (Plan tags)

| Plan | File:line | What |
|------|-----------|------|
| Task 14 (Plan 1) | `ExchangeConfigModal.tsx:34`, `192-195` + NT form section | Added NinjaTrader exchange template + 3-field config form (DataDir / Instrument / DefaultContractQty) |
| Task 14 (Plan 1) | `SettingsPage.tsx:245-247, 267-269, 292-294` | `handleSaveExchange` threads NT fields into both create and update payloads |
| Plan 4.2 fix (BUG 4.1.A.b1) | `web/src/lib/api/config.ts:108-116` | `createExchangeEncrypted` short-circuits encryption for NT (no API key/secret to encrypt) |
| Plan 4.13 (audit followup) | `SettingsPage.tsx:508-558` | Exchange list row branches on `isNinjaTrader` to show "TCP Bridge" badge instead of crypto API-key/Secret badges |
| Plan 1 Task 11 | `handler_exchange.go:30, 47-49, 71-73, 102, 119-121` | Backend types include NT fields in `SafeExchangeConfig`, `CreateExchangeRequest`, `UpdateExchangeConfigRequest`, and the mapper |

### Known gaps (carried forward)

| Gap | Plan | File:line | Symptom | Scope |
|-----|------|-----------|---------|-------|
| `Exchange` (TS interface) does NOT include the 3 NT fields | open | `web/src/types/config.ts:21-50` | `selectedExchange.nt_data_dir` would type-error if read from the modal's edit-mode populate. Currently masked because the populate effect doesn't touch NT fields. | ~10 LOC, 5 min |
| `Exchange` lacks `has_lighter_api_key_private_key` flag | open | `web/src/types/config.ts:43-49` vs `SettingsPage.tsx:553` | SettingsPage reads `exchange.has_lighter_api_key_private_key` — works at runtime (untyped), but `tsc --strict` would flag. Same risk for `has_passphrase`, `has_aster_private_key`. | 10-min type cleanup |
| TelegramConfigModal not exercised in this pass | n/a | `TelegramConfigModal.tsx` | The modal was screenshot but not opened (open requires destructive-ish state changes — pasting token would re-bind). | future inventory pass: open + screenshot empty form state |

### NEW observations

| Description | File:line | Severity | Recommended scope |
|---|---|---|---|
| ExchangeConfigModal edit-mode populate reads `selectedExchange.apiKey/secretKey/asterPrivateKey/lighterPrivateKey` — but `GET /api/exchanges` (the only place `selectedExchange` is sourced from) returns `SafeExchangeConfig` which never includes these. The populate is effectively dead code: it always sets the fields to `''`. | `ExchangeConfigModal.tsx:228-243` | Cosmetic — works as expected (forces user to re-enter secrets) but the code reads as if it's trying to restore values | 10-min: replace `selectedExchange.apiKey || ''` with just `''` and add a comment "secrets never round-trip — user must re-enter on edit" |
| `SettingsPage.handleSaveExchange` has 18 positional parameters | `SettingsPage.tsx:229-248` | Maintainability | When adding a 12th exchange type, the param list grows again. Refactor to a single object parameter. ~40 LOC change. |
| `handleChangePassword` uses raw `fetch` instead of the `api` helper | `SettingsPage.tsx:98-127` | Inconsistency — also leaks the Authorization-header construction pattern to a non-`httpClient` codepath | 15-min: move to `api.changePassword(newPassword)` via `web/src/lib/api/` |
| No "old password" required for password change | `handler_user.go::handleChangePassword` + `SettingsPage.tsx:99-103` | **Security concern** — a stolen JWT can change the password without knowing the old one. Same risk every JWT-only system has, but worth documenting. | Out of scope for inventory pass; flag for security review |
| Mobile-only tab label rendering: `<span className="hidden sm:inline">` ([line 348](web/src/pages/SettingsPage.tsx#L348)) shows only the icon on mobile. No `aria-label` on the bare-icon button → screen-reader-unfriendly on mobile widths. | `SettingsPage.tsx:336-350` | Accessibility | Add `aria-label={tab.label}` to the `<button>`. 5-min |
| "Add Model" / "Add Exchange" empty-state buttons (when `configuredModels.length === 0` / `exchanges.length === 0`) are NOT rendered — only the "No models configured yet" text shows ([line 427-430, 502-504](web/src/pages/SettingsPage.tsx#L427-L430)). User has to click the small "Add Model" button at the top of the section. | `SettingsPage.tsx:407-505` | UX (minor — top button is still visible) | 10-min: add a primary CTA in the empty-state card |
| Page does NOT highlight in `HeaderBar` nav | `paths.ts:43-63` `getCurrentPageForPath` | UX | `/settings` is not in the `Page` union and is missing a `case` in `getCurrentPageForPath`. The Settings nav item never appears "active". Same issue noted in Page 1 doc. |

### Open questions

- Why does `Exchange` (TypeScript) keep `apiKey?: string` / `secretKey?: string` if the safe DTO never returns them? Looks like a backward-compat leftover from before `SafeExchangeConfig` was introduced. Worth a cleanup but not blocking.
- What event fires `agent-config-refresh`? Only AgentChatPage dispatches it (`window.dispatchEvent(new Event('agent-config-refresh'))`). The listener at SettingsPage:88-96 catches it — so when the user uses the Agent's `manage_exchange_config` tool, Settings auto-refreshes. Verify this still works after Plan 4.x changes.
- Should the system support multiple NinjaTrader exchanges (e.g. one for SIM and one for live)? Today the front-end allows it but `account_name` uniqueness isn't enforced visually.

### Cross-page dependencies

- **AgentChat (Page 2)** depends on `/api/models` and `/api/exchanges` being populated. AgentChat dispatches `agent-config-refresh` after tool-driven config changes; Settings listens.
- **Dashboard (Page 4)** consumes `/api/exchanges` to render the trader's exchange info ("Simtest / ninjatrader" header) and to decide whether to show Leverage / Liquidation columns.
- **Strategy (Page 5)** doesn't read exchanges directly but the trader the strategy is attached to does. Editing exchange config can affect strategy compatibility.
- **Setup flow:** when no user exists yet, `AppRoutes` shows `SetupPage` instead of routing. After first registration, the user lands on `/settings` via the `/` redirect — this is the first content page they see.
