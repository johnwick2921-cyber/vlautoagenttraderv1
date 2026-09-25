# Page 5 — Strategy Studio (`/strategy`)

## Quick reference

- **Route:** `/strategy` ([AppRoutes.tsx:507-518](web/src/router/AppRoutes.tsx#L507-L518))
- **Auth:** required
- **Primary source:** [web/src/pages/StrategyStudioPage.tsx](web/src/pages/StrategyStudioPage.tsx) (1455 LOC)
- **Sub-components:**
  | Component | File | LOC |
  |---|---|---|
  | `CoinSourceEditor` | `web/src/components/strategy/CoinSourceEditor.tsx` | 433 |
  | `IndicatorEditor` | `web/src/components/strategy/IndicatorEditor.tsx` | 691 |
  | `RiskControlEditor` | `web/src/components/strategy/RiskControlEditor.tsx` | 316 |
  | `PromptSectionsEditor` | `web/src/components/strategy/PromptSectionsEditor.tsx` | 178 |
  | `PublishSettingsEditor` | `web/src/components/strategy/PublishSettingsEditor.tsx` | 184 |
  | `GridConfigEditor` | `web/src/components/strategy/GridConfigEditor.tsx` | 474 |
  | `TokenEstimateBar` | `web/src/components/strategy/TokenEstimateBar.tsx` | 122 |
- **API endpoints captured (live, 2026-05-27):**
  - `GET /api/strategies`
  - `GET /api/models`
  - `POST /api/strategies/estimate-tokens`
  - On user action: `GET /api/strategies/default-config?lang=…`, `POST /api/strategies`, `PUT /api/strategies/:id`, `DELETE /api/strategies/:id`, `POST /api/strategies/:id/activate`, `POST /api/strategies/:id/duplicate`, `POST /api/strategies/preview-prompt`, `POST /api/strategies/test-run`, `GET /api/my-traders` (delete-guard)
- **Backend handlers:** [api/strategy.go](api/strategy.go) — 10 endpoints (listed below)

## Layer 1: Source code map

### `StrategyStudioPage`
- **File:** [web/src/pages/StrategyStudioPage.tsx:82-1455](web/src/pages/StrategyStudioPage.tsx#L82-L1455)
- **Imports:**
  - React: `useState`, `useEffect`, `useCallback`, `useRef`
  - `../contexts/{AuthContext,LanguageContext}`
  - 22 lucide-react icons (Plus, Copy, Trash2, Check, ChevronDown, ChevronRight, Settings, BarChart3, Target, Shield, Zap, Activity, Save, Sparkles, Eye, Play, FileText, Loader2, RefreshCw, Clock, Bot, Terminal, Code, Send, Download, Upload, Globe)
  - Types: `Strategy`, `StrategyConfig`, `AIStrategyConfig`, `AIModel`, `GridStrategyConfig`
  - `confirmToast`, `notify` from `../lib/notify`
  - All 7 strategy editor sub-components + `TokenEstimateBar`
  - `DeepVoidBackground`, `t`
- **Constants:**
  - `API_BASE = import.meta.env.VITE_API_BASE || ''` ([line 54](web/src/pages/StrategyStudioPage.tsx#L54))
- **Helpers (module-level):**
  - `getAIConfig(config)` ([line 56-68](web/src/pages/StrategyStudioPage.tsx#L56-L68)) — extracts `ai_config` from a strategy, with legacy-shape fallback (older configs had `coin_source/indicators/risk_control/prompt_sections/custom_prompt` at top level instead of under `ai_config`)
  - `normalizeStrategyConfig(config)` ([line 70-80](web/src/pages/StrategyStudioPage.tsx#L70-L80)) — produces the canonical shape with `strategy_type`, `language`, `ai_config`, `grid_config`, `publish_config`
- **State (15+):**
  - `strategies: Strategy[]` (list)
  - `selectedStrategy: Strategy | null`
  - `editingConfig: StrategyConfig | null` (working copy; mutated on every field edit)
  - `isLoading`, `isSaving`, `error: string | null`, `hasChanges`
  - `estimatedTokens: number`
  - `aiModels: AIModel[]`, `selectedModelId: string`
  - `expandedSections: { gridConfig, coinSource, indicators, riskControl, promptSections, customPrompt, publishSettings }` — left-panel accordion booleans (gridConfig + coinSource default open)
  - `activeRightTab: 'prompt' | 'test'`
  - `promptPreview: { system_prompt, user_prompt?, prompt_variant, config_summary } | null`
  - `isLoadingPrompt: boolean`, `selectedVariant: string` (default `'balanced'`)
  - `aiTestResult: { system_prompt?, user_prompt?, ai_response?, reasoning?, decisions?, error?, duration_ms? } | null`
  - `isRunningAiTest: boolean`
- **Refs (3):**
  - `gridConfigCacheRef: Record<string, GridStrategyConfig>` — remembers grid config per strategy id while user clicks around (rehydrated when re-selected)
  - `selectedStrategyIDRef: string` — synced with `selectedStrategy?.id`, used for slug-preservation across refreshes
  - `hasChangesRef: boolean` — synced with `hasChanges`, used to avoid clobbering local edits when the focus/visibilitychange listener fires `fetchStrategies()`
- **Effects (5):**
  1. `useEffect [fetchStrategies, fetchAiModels]` — initial load on mount
  2. `useEffect [selectedStrategy?.id]` — sync `selectedStrategyIDRef`
  3. `useEffect [hasChanges]` — sync `hasChangesRef`
  4. `useEffect [fetchStrategies, token]` — `window.focus` + `document.visibilitychange` listener → refetch strategies on tab regain (only if `document.visibilityState === 'visible'`)
  5. `useEffect [selectedStrategy?.id, editingConfig?.grid_config]` — cache grid config to `gridConfigCacheRef`
  6. `useEffect [language, token]` — when language changes, fetch a fresh default config in the new language and substitute only `ai_config.prompt_sections` + `language` — preserves user's other edits
- **Callbacks (16+):**
  - `fetchAiModels` — `GET /api/models` (filters by `enabled`, picks first as default selection)
  - `fetchStrategies` — `GET /api/strategies` — preserves current selection by id, falls back to `is_active` then `[0]`
  - `handleCreateStrategy` — GET default-config + POST /api/strategies
  - `handleDeleteStrategy` — guards on `is_active`, checks via `/api/my-traders` that the strategy isn't in use, then `DELETE /api/strategies/:id`
  - `handleDuplicateStrategy` — `POST /api/strategies/:id/duplicate`
  - `handleActivateStrategy` — `POST /api/strategies/:id/activate`
  - `handleExportStrategy(strategy)` — builds a JSON blob and triggers a browser file download (no backend call)
  - `handleImportStrategy` — file-picker → POST /api/strategies
  - `handleSaveStrategy` — `PUT /api/strategies/:id` with merged config
  - `updateConfig(patch)` — shallow merge into `editingConfig`, sets `hasChanges=true`
  - `updateAIConfig(patch)` — same but nested under `ai_config`
  - `handleStrategyTypeChange(type)` — switches between `'ai_trading'` and `'grid_trading'`; uses `gridConfigCacheRef` to restore previously-edited grid config
  - `fetchPromptPreview(variant)` — `POST /api/strategies/preview-prompt`
  - `runAiTest()` — `POST /api/strategies/test-run` with the selected model + variant
- **Render structure (3-pane layout):**
  - **Left:** strategy list (cards: name, is_active flag, is_default flag, Activate/Duplicate/Export/Delete actions)
  - **Center:** accordion editor with sections (Strategy Type selector, Coin Source, Indicators, Risk Control, Prompt Editor, Extra Prompt, Publish, Grid Config when type=grid_trading)
  - **Right:** Prompt Preview / AI Test toggle, with variant selector (`scalp` | `balanced` | `swing` | `position` | `futures`)
- **`Save` button gating:** disabled unless `hasChanges` AND `editingConfig` is non-null. Captured as `disabled` in the runtime snapshot.

### Sub-component: `CoinSourceEditor`
- **File:** [web/src/components/strategy/CoinSourceEditor.tsx:14-433](web/src/components/strategy/CoinSourceEditor.tsx#L14-L433)
- **Props:** `{ config: CoinSourceConfig, onChange, disabled?, language }`
- **`sourceTypes` constant** ([line 23-28](web/src/components/strategy/CoinSourceEditor.tsx#L23-L28)): 4 options — `static` (manual list), `ai500` (top by AI ranking), `oi_top` (open-interest growth), `oi_low` (OI decay)
- **`xyzDexAssets` set** ([line 31-42](web/src/components/strategy/CoinSourceEditor.tsx#L31-L42)) — 23 entries covering stocks (TSLA / NVDA / AAPL …), forex (EUR / JPY), commodities (GOLD / SILVER), index (XYZ100). Used to skip the USDT suffix on the `xyz:` DEX assets.
- **`handleAddCoin`** ([line 60-88](web/src/components/strategy/CoinSourceEditor.tsx#L60-L88)) — 10-coin cap, applies `xyz:` prefix for stock/forex/commodity assets, otherwise auto-appends `USDT` if not already present. **No NinjaTrader / CME futures branch.** A user typing `NQ` will get stored as `NQUSDT`, which Databento rejects.
- **`MAX_STATIC_COINS = 10`** ([line 49](web/src/components/strategy/CoinSourceEditor.tsx#L49))

### Sub-component: `IndicatorEditor`
- **File:** [web/src/components/strategy/IndicatorEditor.tsx:34-691](web/src/components/strategy/IndicatorEditor.tsx#L34-L691)
- **Props:** `{ config: IndicatorConfig, onChange, disabled?, language }`
- **Module constants:**
  - `DEFAULT_NOFXOS_API_KEY = 'cm_568c67eae410d912c54c'` ([line 7](web/src/components/strategy/IndicatorEditor.tsx#L7)) — hardcoded literal. Per CLAUDE.md: `nofxos.ai` is deprecated and returns HTTP 402; this default is dead.
  - `allTimeframes` ([line 17-32](web/src/components/strategy/IndicatorEditor.tsx#L17-L32)) — 14 entries (1m through 1w) categorized scalp/intraday/swing/position
- **`toggleTimeframe()`** ([line 44-86](web/src/components/strategy/IndicatorEditor.tsx#L44-L86)) — enforces max 4 timeframes via inline DOM toast
- **Section layout (read partial):** Quant Data, OI Ranking, NetFlow Ranking, Price Ranking, Market Sentiment (funding_rate, OI), Technical Indicators (EMA / MACD / RSI / BOLL / ATR), Raw Klines toggle.
- **NinjaTrader awareness:** None. All sections render regardless of exchange. Funding rate / OI sections especially have no meaning for CME futures.

### Sub-component: `RiskControlEditor`
- **File:** [web/src/components/strategy/RiskControlEditor.tsx:12-316](web/src/components/strategy/RiskControlEditor.tsx#L12-L316)
- **Props:** `{ config: RiskControlConfig, onChange, disabled?, language }`
- **`updateField<K>(key, value)`** helper at line 18-25
- **Render sections:**
  - Position Limits — `max_positions` (read-only display, "System enforced")
  - Trading Leverage — `btc_eth_max_leverage` slider (1-20x) + `altcoin_max_leverage` slider (1-20x). **Labels hardcoded "BTC/ETH Leverage" / "Altcoin Leverage"** — not parameterized on futures.
  - Position Value Ratio — `btc_eth_max_position_value_ratio` / `altcoin_max_position_value_ratio` sliders
  - Margin Usage — `max_margin_usage` (0.5-0.95)
  - Min Position Size — `min_position_size` (USDT label hardcoded)
  - Min Risk/Reward — `min_risk_reward_ratio`
  - Min Confidence — `min_confidence` (60-90)
- **NinjaTrader awareness:** None.

### Sub-component: `PromptSectionsEditor`
- **File:** [web/src/components/strategy/PromptSectionsEditor.tsx:45-178](web/src/components/strategy/PromptSectionsEditor.tsx#L45-L178)
- **Props:** `{ config: PromptSectionsConfig, onChange, disabled?, language }`
- 4 free-text fields: `role_definition`, `trading_frequency`, `entry_standards`, `decision_process`. Each is an unbounded textarea.

### Sub-component: `PublishSettingsEditor`
- **File:** [web/src/components/strategy/PublishSettingsEditor.tsx:13-184](web/src/components/strategy/PublishSettingsEditor.tsx#L13-L184)
- **Props (typical pattern, not fully read):** `{ config, onChange, disabled?, language }`
- Controls: `is_public` toggle, `config_visible` toggle, license metadata. Powers the Strategy Market — but Plan 1 Task 11 deleted the Strategy Market page; the publish toggles still work, just no public UI to browse them.

### Sub-component: `GridConfigEditor`
- **File:** [web/src/components/strategy/GridConfigEditor.tsx:32-474](web/src/components/strategy/GridConfigEditor.tsx#L32-L474)
- **Exports:** `GridConfigEditor` + `defaultGridConfig` constant
- Only rendered when `strategy_type === 'grid_trading'`. Inactive for AI traders.

### Sub-component: `TokenEstimateBar`
- **File:** [web/src/components/strategy/TokenEstimateBar.tsx:27-122](web/src/components/strategy/TokenEstimateBar.tsx#L27-L122)
- **Props:** `{ config: StrategyConfig, language, onTokenCountChange?: (n: number) => void }`
- **Behavior:** debounced (~1s) `POST /api/strategies/estimate-tokens` whenever `config` changes; shows estimated token usage with progress bar (4% of some limit was visible in the runtime snapshot — likely `~400` tokens / 10000 limit).

## Layer 2: Runtime DOM + network

### Initial render

- **Screenshot:** [inventory_strategy_studio.png](inventory_strategy_studio.png)
- **URL:** `http://localhost:3000/strategy`
- **DOM structure:**
  - Header: "Strategy Studio / Configure and test trading strategies" with `Settings` icon
  - 3-pane layout:
    - **Left:** "Strategies" panel with 4 strategy cards visible (refs e61, e77, e93, e108 — names not labeled in the depth-7 snapshot)
    - **Center:** Header bar (`Save` button — disabled because no edits), `4%` token progress, editor accordion. Visible buttons (the accordion section headers): `Indicators`, `Risk Control`, `Prompt Editor`, `Extra Prompt`, `Publish`. The current section (probably Coin Source) is auto-expanded but accordion header label not in the snapshot at this depth.
    - **Right:** Toggle `Prompt Preview` / `AI Test`
- **Console:** 1 error — `https://grainy-gradients.vercel.app/noise.svg 404`. External CDN reference (DeepVoidBackground or similar). Cosmetic.
- **Network requests on load:**
  | URL | Method | Status |
  |---|---|---|
  | `/api/config` | GET | 200 |
  | `/api/strategies` | GET | 200 |
  | `/api/models` | GET | 200 |
  | `/api/strategies/estimate-tokens` | POST | 200 (auto-fired after strategy selection, debounced) |
- **NEW observation:** `/api/strategies` and `/api/models` each fired TWICE on mount (entries 138/140 and 139/141). Likely StrictMode double-invocation OR the useEffect at line 207-210 firing twice because `fetchStrategies` and `fetchAiModels` are recreated by `useCallback` on identity change.

## Layer 3: Backend code path

### `GET /api/strategies` → `handleGetStrategies` ([api/strategy.go:101](api/strategy.go#L101))
- **Auth:** required
- **Returns:** `{ strategies: [...] }` — array of `Strategy` objects with full config
- **Store call:** `s.store.Strategy().List(userID)` ([store/strategy.go](store/strategy.go))

### `GET /api/strategies/default-config?lang=…` → `handleGetDefaultStrategyConfig` ([api/strategy.go:479](api/strategy.go#L479))
- **Auth:** required
- **Query:** `lang=zh|en`
- **Returns:** the canonical StrategyConfig with sensible defaults — used by `handleCreateStrategy` and by the language-change effect on the frontend

### `POST /api/strategies/estimate-tokens` → `handleEstimateTokens` ([api/strategy.go:49](api/strategy.go#L49))
- **Auth:** NOT required (pure computation)
- **Body:** complete `StrategyConfig`
- **Returns:** `{estimated_tokens, breakdown?}`

### `POST /api/strategies` → `handleCreateStrategy` ([api/strategy.go:174](api/strategy.go#L174))
- **Body:** `{name, description, lang, config?}` — config optional; backend fills in defaults if absent
- **Returns:** `{id}`
- **Side effect:** persisted via `store.Strategy().Create(userID, …)`

### `PUT /api/strategies/:id` → `handleUpdateStrategy` ([api/strategy.go:262](api/strategy.go#L262))
- **Body:** `{name, description, config}`
- **Note:** docs at server.go:269-273 say "config is merged with existing values server-side, but always send the complete section you are modifying." Worth verifying that the frontend always sends complete sections.

### `DELETE /api/strategies/:id` → `handleDeleteStrategy` ([api/strategy.go:381](api/strategy.go#L381))
- **Guard:** rejects if strategy is currently in use by any trader (returns 409 or similar). Frontend duplicates this guard via the `/api/my-traders` check before showing the confirm dialog.

### `POST /api/strategies/:id/activate` → `handleActivateStrategy` ([api/strategy.go:399](api/strategy.go#L399))
- Sets `is_active=true` for this strategy AND `is_active=false` for the previously-active one

### `POST /api/strategies/:id/duplicate` → `handleDuplicateStrategy` ([api/strategy.go:417](api/strategy.go#L417))

### `POST /api/strategies/preview-prompt` → `handlePreviewPrompt` ([api/strategy.go:492](api/strategy.go#L492))
- **Body:** `{config, prompt_variant}` plus probably `language`
- **Returns:** `{system_prompt, user_prompt?, prompt_variant, config_summary}`
- Routes into `kernel/engine_prompt.go::BuildSystemPrompt(variant)` — when `variant=='futures'`, dispatches to `kernel/engine_prompt_futures.go::BuildFuturesSystemPrompt`

### `POST /api/strategies/test-run` → `handleStrategyTestRun` ([api/strategy.go:541](api/strategy.go#L541))
- **Body:** `{config, prompt_variant, model_id}`
- **Returns:** `{system_prompt, user_prompt, ai_response, reasoning, decisions[], duration_ms}` or `{error}`
- **External call:** uses the configured AI provider to actually run the prompt. **Costs money** (real API tokens consumed).

### `GET /api/strategies/:id` → `handleGetStrategy` ([api/strategy.go via route line 258 (s.route(...))](api/server.go#L258))
- Single-strategy fetch by id

## Layer 4: Cross-references + gaps + open questions

### Shipped fixes (Plan tags)

| Plan | File:line | What |
|---|---|---|
| Plan 1 Task 9 | `kernel/engine_prompt_futures.go` | Adds `BuildFuturesSystemPrompt` / `BuildFuturesUserPrompt` invoked by `BuildSystemPrompt(variant='futures')` |
| Plan 1 Task 11 | `web/src/router/AppRoutes.tsx` | Removed Strategy Market route — the `is_public` toggle in PublishSettingsEditor still works but no public consumer |

### Known gaps

| Gap | Plan | File:line | Symptom | Scope |
|---|---|---|---|---|
| Strategy Studio is fully crypto-coupled — no NT-aware branches | Plan 4.6 | `CoinSourceEditor.tsx:60-88`, `IndicatorEditor.tsx` (all sections), `RiskControlEditor.tsx:60-128`, `TokenEstimateBar.tsx:122` | A NT trader's strategy editor still shows "BTC/ETH Leverage", "Altcoin Leverage", funding rate toggle, OI ranking — all meaningless for CME futures | ~400 LOC, 3 hr |
| `CoinSourceEditor.handleAddCoin` auto-appends `USDT` for everything except the hardcoded `xyzDexAssets` set | bundled in Plan 4.6 | `CoinSourceEditor.tsx:78` | User typing `NQ` gets `NQUSDT` (which Databento rejects). Need a CME-futures pattern check (`NQ.c.0`, `MNQ.c.0`, `ES.c.0`, etc.) | bundled |
| `IndicatorEditor.DEFAULT_NOFXOS_API_KEY` literal | open | `IndicatorEditor.tsx:7` | nofxos.ai is deprecated (HTTP 402). The default key is dead code; the field should default empty | 5-min |

### NEW observations

| Description | File:line | Severity | Recommended scope |
|---|---|---|---|
| `/api/strategies` and `/api/models` each fire TWICE on Strategy page load (network entries 138/140, 139/141 paired) | useCallback-keyed useEffect at line 207-210 + StrictMode | Cosmetic + 2× backend call | Wrap `fetchStrategies` / `fetchAiModels` in a `ref` to avoid recreation, or move to SWR which dedupes |
| Save button is `[disabled]` on initial render even when a strategy is selected | runtime; `disabled` attr on ref e133 | Expected (no hasChanges yet) but ensure clicking a field actually flips it | n/a — verify on next pass |
| Prompt Editor / Extra Prompt are separate sections | `StrategyStudioPage.tsx:256, 265` accordion buttons | Inconsistency: PromptSectionsEditor renders the 4 sub-fields, but "Extra Prompt" is a separate panel (probably `custom_prompt` free-text). Worth confirming the distinction is clear to users. | Document or merge |
| External CDN reference: `grainy-gradients.vercel.app/noise.svg` returns 404 | `DeepVoidBackground` or similar background asset | Cosmetic console noise | 5-min: bundle the SVG locally or change the URL |
| `/api/my-traders` early-mount fetch (entries 2, 3 — both `ERR_ABORTED`) — this fires from elsewhere (LegacyHashRedirect? HeaderBar? AuthContext?) before token is ready and gets cancelled. Not exclusive to Strategy page. | global | Cosmetic | Trace the unconditional `/api/my-traders` call on app boot |
| `confirmDelete` flow has redundant guard: frontend checks /api/my-traders AND backend also rejects in-use deletes | `StrategyStudioPage.tsx:350-365` | Good defensive practice (clearer error message before the dialog opens) — not a bug | n/a |
| `selectedVariant` state default `'balanced'` ([line 125](web/src/pages/StrategyStudioPage.tsx#L125)) — for a NT trader running NQ, the default should be `'futures'` | UX gap | Tied to Plan 4.6 — when strategy is attached to a NT trader, default variant to `'futures'` | bundled |

### Open questions

- The `language`-change effect (line 249-287) updates `prompt_sections` to the new language's defaults. Does this clobber user-entered prompt text? Reading the diff: it merges `defaultConfig.ai_config?.prompt_sections` into the current `ai_config`. If the user has hand-edited prompt sections, switching language WILL overwrite them. Probably intentional but easy to miss.
- Does `gridConfigCacheRef` cache across mount/unmount? Likely no (refs reset on unmount), so the cache is per-visit, not persistent.
- The default token budget threshold for the `4%` progress bar — what's the denominator? TokenEstimateBar must encode it; worth a follow-up to capture.
- Where does the `Strategy` interface in `/types` get the new `strategy_type='futures'` discriminator if/when Plan 4.6 ships? Currently `strategy_type` only has `'ai_trading'` / `'grid_trading'`.

### Cross-page dependencies

- **/strategy → /traders & /dashboard:** the strategy `is_active=true` is what new traders default to. Activating a strategy doesn't automatically update running traders (they keep their assigned strategy until edited).
- **/strategy → /agent (Page 2):** the AgentChat `manage_strategy` tool calls the same backend (`POST /api/strategies`). After such a call, AgentChat fires `agent-config-refresh` event — but `StrategyStudioPage` does NOT listen for that event (only Settings + AITradersPage do). Means the user has to manually refocus the tab to see strategy changes from the agent. ([line 220-235](web/src/pages/StrategyStudioPage.tsx#L220-L235) — only focus/visibilitychange triggers refresh).
- **/settings:** new strategy creation requires no exchange or AI model — but `runAiTest` needs `selectedModelId !== ''`, which requires at least one enabled model in Settings.
