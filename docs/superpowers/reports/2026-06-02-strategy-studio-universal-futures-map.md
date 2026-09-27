# Strategy Studio — Crypto → Universal (Futures / NinjaTrader) Change Map

**Repo:** `/home/hoang/nofx` · **Branch:** `feat/nt8-stage4-chart` · **HEAD at analysis:** `24634b5a` (capstone `8e9b1743` + dashboard fix)
**Date:** 2026-06-02 · **Scope:** read-only deep analysis (no code changed; this report is the only artifact)
**Method:** 12 per-section sub-agent maps (UnderstandAnything graph → CGC → canonical source at `file:line`) + MAIN re-verification of every headline wire in `/home/hoang/nofx` source + a real-browser crypto-vs-futures pass (MNQ vs BTCUSDT). Screenshots are local artifacts under `.playwright-mcp/` (git-ignored), referenced by name — not committed.

---

## 1. Executive summary (for the CTO)

**The product reality.** The company trades **CME index futures (NQ / MNQ)** through a NinjaTrader 8 TCP bridge. The Strategy Studio — the page where a desk builds, previews, tests, and activates an AI trading strategy — was written for **crypto** (USDT pairs, exchange leverage, funding rate, NofxOS/AI500 coin universes). The backend futures engine is already built (recognition, quarterly contract resolver, a dedicated futures system prompt, NT8 real-spec `instrument_info`) and the live MNQ decision path works. The **UI**, however, is only *partially* futures-aware.

**The governing rule for every change below is ADDITIVE:** keep every crypto capability **byte-identical**, **add** the futures path, **delete nothing**. Crypto and futures coexist; the instrument decides what renders.

**What is already done (capstone `8e9b1743`, a deliberately narrow pass).** A shared recognizer `isCMEFutures()` was extracted to `web/src/lib/instrument.ts`; `StrategyStudioPage` computes `isFuturesStrategy = isCMEFutures(coin_source.static_coins[0])` (`StrategyStudioPage.tsx:709-711`) and threads an `isFutures` prop into **exactly two of twelve** sections:
- **Risk Control** — hides the exchange Trading-Leverage tiers; relabels Min-Position-Size unit/description `USDT → USD`.
- **Indicators** — hides the funding-rate toggle and the "funding rate" mention in the Market-Sentiment subtitle.
- (plus the dashboard `DecisionCard`, which is outside Strategy Studio).

**What remains (the substance of this report).** Ten of twelve sections are still crypto-only or ungated. The headline gaps:

| Rank | Gap | Where | Type |
|---|---|---|---|
| 1 | **Position-Value-Ratio still shows "BTC/ETH" + "Altcoin" labels on a futures strategy** (capstone gated leverage right next to it but not this block) | `RiskControlEditor.tsx:154-209` | FE-only |
| 2 | **Prompt Preview / AI Test always renders the CRYPTO system prompt for an MNQ strategy** — the variant dropdown only emits balanced/aggressive/conservative, and the futures prompt requires `variant=="futures"` | `StrategyStudioPage.tsx:1216-1224,1324-1332` → `engine_prompt.go:22` | FE-or-BE |
| 3 | **Coin Source** shows the three crypto-only data-provider tiles (AI500, OI Increase/Decrease) + NofxOS branding for futures | `CoinSourceEditor.tsx` (no `isFutures` prop) | FE-only |
| 4 | **Indicators** still shows the whole crypto NofxOS data block (AI500/OI-Ranking/NetFlow/Price-Ranking) + the dead `cm_568c67…` key for futures | `IndicatorEditor.tsx` | FE-only |
| 5 | **Strategy-Type selector** offers "AI Grid Trading" on a futures strategy — grid is crypto-CEX-only and **cannot execute on NT8** | `StrategyStudioPage.tsx:1111-1132` | FE-only (+ optional BE guard) |
| 6 | **Live prompt-variant is chosen by broker (`exchange=="ninjatrader"`), not by symbol** — a futures symbol on a non-NT8 trader would still get the crypto prompt; the FE uses a *different* signal (`isCMEFutures(static_coins[0])`) | `auto_trader_loop.go:114` vs `StrategyStudioPage.tsx:709` | BE gap |
| 7 | **Hard R/R gate is hardcoded `≥ 3.0`** and ignores the user's `min_risk_reward_ratio`; the futures prompt advertises the user's value (default 1.5) → compliant futures trades silently rejected | `engine_position.go:133` | BE gap |
| 8 | **New Strategy can't be born futures from the UI** — the futures default seed is gated on the process env `TRADING_MODE=futures`, not per-strategy | `store/strategy.go:935-953`, `api/strategy.go:479` | BE gap |

**The good news:** the vast majority of the remaining work is **FE-only conditional gating** that reuses assets that already exist — the `isCMEFutures` recognizer (P1), `market.FuturesPointValue` / `FuturesTickSize` (P1/P3), the futures system prompt (P3), and `instrument_info` specs (P4). No new hardcoded symbols or dollar values are required. A short list of **backend gaps** (#2 BE option, #6, #7, #8, and PromptSections honoring) is itemized in §9. Grid trading is honestly **out of scope** (it has no working NT8 execution path) and should be *parked*, not ported.

**Live-verified at HEAD `24634b5a`:** an MNQ strategy renders Risk Control with **no** leverage tiers, **"Minimum notional value in USD"**, but **still** "BTC/ETH Position Value Ratio" + "Altcoin Position Value Ratio"; Indicators with **no** funding-rate toggle but the **full** NofxOS crypto data block; and a preview variant dropdown of only Balanced/Aggressive/Conservative. Switching the same scratch strategy's coin to BTCUSDT brings the leverage tiers, USDT units, and funding-rate toggle back — confirming the crypto path is intact. (Screenshots: `ss-futures-risk-control.png`, `ss-futures-indicators.png`, `ss-crypto-risk-control.png`, `ss-crypto-indicators.png`.)

---

## 2. Section inventory

The Studio page (`web/src/pages/StrategyStudioPage.tsx`, ~1474 lines) is a 3-column layout: a strategy **list** (left), the **config editors** (middle, an accordion gated by `strategy_type` via the `forStrategyType` filter at `:846-850`), and a **Preview / AI-Test** panel (right). Editors live in `web/src/components/strategy/`.

| # | Section | Component / location | Rendered @ `StrategyStudioPage.tsx` | Type gate | `isFutures` threaded? |
|---|---|---|---|---|---|
| 1 | Strategy List & sidebar actions | inline | `886-1002` (+ handlers `295-506`) | — | No |
| 2 | Header: name / desc / Activate / Save | inline | `1008-1068` (+ `426-441`, `509-548`) | — | No |
| 3 | Token Estimate Bar | `TokenEstimateBar.tsx` | `1070-1079` | ai_trading | No |
| 4 | Strategy Type selector | inline | `1081-1135` (+ `582-622`) | — | No |
| 5 | Coin Source | `CoinSourceEditor.tsx` | `731-745` | ai_trading | **No** |
| 6 | Indicators | `IndicatorEditor.tsx` | `746-761` | ai_trading | **Yes** ✅ |
| 7 | **Risk Control (deep-dive §6)** | `RiskControlEditor.tsx` | `762-779` | ai_trading | **Yes** ✅ |
| 8 | Prompt Sections | `PromptSectionsEditor.tsx` | `780-796` | ai_trading | No |
| 9 | Custom Prompt | inline textarea | `797-822` | ai_trading | No |
| 10 | Publish Settings | `PublishSettingsEditor.tsx` | `823-845` | both | No |
| 11 | Grid Config | `GridConfigEditor.tsx` | `715-729` | grid_trading | n/a (no coin_source) |
| 12 | Right Panel: Prompt Preview & AI Test | inline | `1182-1467` (+ `624-683`) | — | No |
| X1 | **Backend wire + schema** (cross-cutting) | `store/strategy.go`, `api/strategy.go`, `kernel/*` | — | — | — |

> `GridRiskPanel.tsx` is **dashboard-only** (used by `TraderDashboardPage`), not a Studio section — excluded by design.

---

## 3. Per-section map (A = what/wire · B = controls→wire · C = crypto-vs-futures render · D = crypto assumptions @ file:line · E = additive change-plan)

### Section 1 — Strategy List & sidebar actions  ·  `StrategyStudioPage.tsx:886-1002`

**(A) What / wire.** The strategy catalog + lifecycle toolbar (New, Import, per-item Export/Duplicate/Delete, Activate; `is_active`/`is_default`/`is_public` tags). Selecting an item loads its `config` into the editors — i.e. this is where the load-bearing `coin_source.static_coins[0]` (which drives `isFuturesStrategy` and the engine's candidate symbols) is chosen. Handlers `295-506` call `/api/strategies*` (`api/server.go:257-332` → `api/strategy.go` → `store/strategy.go`). Export is pure client-side (`:444-464`, no backend).

**(B) Controls.** New → `GET default-config` + `POST /strategies` (`strategy.go:479,174`); Import → `POST /strategies` (`:174`, validates `config`+`name`); Export → client Blob; Duplicate → `POST /:id/duplicate` (`:417`); Delete → `DELETE /:id` (`:381`, blocks default/active/in-use); Activate → `POST /:id/activate` (`:399` → `store.SetActive :1192`); list-item select → local state, no API.

**(C) Render.** List/tags/buttons render **identically** for MNQ and BTCUSDT — instrument-agnostic mechanics (confirmed: MNQ passes through `normalizeSymbols` with CME case preserved, no USDT append).

**(D) Crypto assumptions.**
- `is_public` / Globe "Public" tag — a *crypto Strategy-Market* concept whose consumer page was deleted; still renders, dormant for a fresh MNQ strategy (`:991-996`; type `web/src/types/strategy.ts:8`).
- New Strategy is **born crypto** (`ai500` pool) unless `TRADING_MODE=futures` (`store/strategy.go:935-953 defaultCoinSource`) — env-gated, not per-strategy.
- Agent-facing create-schema doc text is all crypto (`api/server.go:275-311`: BTCUSDT/ETHUSDT, funding-rate, leverage tiers, `cm_568c67…`).
- Export envelope carries no `instrument_type`/`asset_class` marker (`:444-451`).

**(E) Additive plan.**
1. Gate the `is_public` tag to crypto (`!isCMEFutures(item.coin_source.static_coins[0])`) — futures strategies are private/internal. *FE-only.*
2. Add an optional per-item instrument badge ("CME · MNQ" vs crypto symbol), reusing `cmeFuturesRoots`. *FE-only.*
3. Let New Strategy seed a futures strategy from the UI regardless of process mode — add an `asset_class` param to `POST /strategies` + `default-config`, reusing the existing `defaultCoinSource` futures branch (`store/strategy.go:936-944`). *BE gap.*
4. Add `asset_class` to the Export envelope (derive via `isCMEFutures`); old files still import. *FE-only.*
5. Add a futures variant of the agent create-schema doc (keep crypto string byte-identical). *BE, doc-only.*

---

### Section 2 — Header: name / description / Activate / Save  ·  `StrategyStudioPage.tsx:1008-1068`

**(A) What / wire.** The commit funnel: every editor edit lives in `editingConfig` and only persists when **Save** (`handleSaveStrategy :509-548` → `PUT /strategies/:id` → `handleUpdateStrategy api/strategy.go:262`) fires, and only goes live when **Activate** (`:426-441` → `SetActive`) fires. Save runs `MergeStrategyConfig` (`:309`) then **`ClampLimits()` unconditionally** (`:316` → `store/strategy.go:38`).

**(B) Controls.** Name/Description inputs → `setHasChanges` → PUT body name/description columns; Unsaved badge (display); Activate (shown `!is_active`); Save (shown `!is_default`, disabled `!hasChanges`).

**(C) Render.** Header is **visually instrument-agnostic** — identical for MNQ and BTCUSDT. It does **not** consume `isFuturesStrategy`.

**(D) Crypto assumptions (all backend, fire on every futures Save).**
- `ClampLimits()` clamps `BTCETHMaxLeverage`/`AltcoinMaxLeverage` into `[1,20]` even for MNQ (`store/strategy.go:81-93`).
- `StrategyClampWarnings` can emit BTC/ETH + altcoin leverage warnings (`store/strategy.go:544,571-572`).
- `NormalizeProductSchema` default-fallthrough is `ai500` (crypto) when source is empty (`store/strategy.go:139-178`).
- `validateStrategyConfig` warns on missing NofxOS key when crypto data sources are on (`api/strategy.go:21-35`).

**(E) Additive plan.** Header UI needs **no change** (already instrument-agnostic). On the BE, gate the leverage clamp + leverage warnings + ai500 default-fallthrough on `market.IsCMEFuturesSymbol(static_coins[0])` so a futures Save isn't crypto-clamped/warned — keep the crypto path byte-identical. *BE gaps (optional; today they are harmless because the gate bypasses the clamped fields for futures).* Optional FE polish: a small "CME futures" badge by the Name.

---

### Section 3 — Token Estimate Bar  ·  `TokenEstimateBar.tsx` (rendered `StrategyStudioPage.tsx:1070-1079`)

**(A) What / wire.** Advisory progress bar estimating prompt tokens vs a hardcoded 200K budget. Debounced `POST /strategies/estimate-tokens` (`api/server.go:126` → `handleEstimateTokens :49` → `StrategyConfig.EstimateTokens() store/strategy.go:1382`, a pure computation). **No trade-path effect.**

**(C) Render.** Renders/estimates identically for MNQ (no `isFutures` prop — `TokenEstimateBar.tsx:21-27`).

**(D) Crypto assumptions.** Backend estimator bills crypto-only cost terms with **no futures branch**: funding-rate term (`store/strategy.go:1444-1446`), quant OI/netflow (`:1451-1460`), OI/NetFlow/Price rankings (`:1462-1490`), a "BTC price" fixed-overhead line (`:1405-1406`), and a crypto coin-source switch in `getEffectiveCoinCount` (`:1554-1572`). Because the bar is fed the **un-gated** `editingConfig`, a futures strategy that still has `enable_funding_rate:true` (capstone hides the toggle but doesn't clear the value) has phantom crypto cost counted.

**(E) Additive plan.** Thread `isFutures` into the bar; add a futures tooltip variant; in `EstimateTokens` branch on a CME symbol so crypto-only cost buckets are skipped and the "BTC price" overhead becomes the futures spec line — keep the crypto code path untouched. *FE-only for the bar; BE for accurate accounting (low priority — currently over-counts harmlessly).*

---

### Section 4 — Strategy Type selector (AI Trading vs AI Grid Trading)  ·  `StrategyStudioPage.tsx:1081-1135`

**(A) What / wire.** Two-button mode switch setting `strategy_type`. AI Trading → the AI decision sections (Coin Source/Indicators/Risk/Prompt). Grid → the Grid Config editor. **Futures is *not* a separate `strategy_type`** — it is the `ai_trading` path with `promptVariant="futures"` chosen at runtime when `exchange=="ninjatrader"` (`auto_trader_loop.go:113-116`). Grid dispatch is `RunGridCycle` (`auto_trader.go:561-600`), entirely separate.

**(C) Render.** The "AI Grid Trading" button is **fully clickable for an MNQ strategy** — no `isFutures` gate.

**(D) Crypto assumptions.** Grid mode is crypto-CEX-only: the grid prompt hardcodes USDT investment, `Leverage: %dx`, BTCUSDT example orders, a FundingRate field (`kernel/grid_engine.go:67,116,150-151,168-169,202-203`); grid places **limit** orders, which the NT8 bridge does not support (market-only — `trader/CLAUDE.md`). `gridTradingDesc` copy is crypto-grid framed (`translations.ts:1033-1034`).

**(E) Additive plan.** When `isFuturesStrategy`, hide (or disable with a tooltip) the AI-Grid-Trading button and render single-column AI-Trading only; add `gridTradingFuturesUnsupported` i18n (en/zh/es). *FE-only.* Optional BE belt-and-suspenders: reject `grid_trading` + CME symbol in `validateStrategyConfig` (`api/strategy.go:21-35`). Keep crypto byte-identical.

---

### Section 5 — Coin Source Editor  ·  `CoinSourceEditor.tsx` (rendered `StrategyStudioPage.tsx:731-745`)

**(A) What / wire.** Picks the instrument(s). Four source types (Static List, AI500, OI Increase, OI Decrease) + add/exclude-coin inputs. Its `static_coins[0]` is the single most load-bearing field: it drives `isFuturesStrategy` (`:709-711`) and the engine candidate set (`kernel/engine.go:263-466 GetCandidateCoins`). `market.Normalize` early-returns CME symbols unchanged (no USDT append) so MNQ survives end-to-end (`market/data.go:595-599`).

**(C) Render.** **Verified live:** for MNQ the editor still shows **all four** source tiles including AI500 / OI Increase / OI Decrease, the NofxOS branding, and the placeholder "BTC, ETH, SOL…". The **only** futures-aware behavior is the USDT-skip on add (capstone).

**(D) Crypto assumptions.**
- USDT auto-append on add (`:127,:162`) — correctly **skipped** for CME roots via `isCMEFutures` (`:122-128,:157-163`) ✅ (this is the capstone's CoinSource scope).
- Source-type tiles AI500 / OI Increase / OI Decrease (`:33-38,197-225`) + their NofxOS panels (`:331-504`) + NofxOS badge/note (`:182-187`) — crypto-only data providers, **ungated** (no `isFutures` prop).
- Placeholders "BTC, ETH, SOL…" / "BTC, ETH, DOGE…" (`:260,:316`); all coinSource i18n is crypto-framed ("coins"/币种).

**(E) Additive plan.**
1. Thread `isFutures` into `CoinSourceEditor` (exactly as Indicator/Risk already receive it). *FE-only, enabler.*
2. When `isFutures`, render only the **Static List** tile (a futures strategy is a single NT8 instrument); the BE static branch already feeds the futures path — no BE change. *FE-only.*
3. Defensively gate the AI500/OI panels + NofxOS badge behind `!isFutures`. *FE-only.*
4. Futures placeholder "MNQ, NQ, ES…" from `cmeFuturesRoots`; futures i18n variant of "coins" → "Contracts". *FE-only.*
5. Leave USDT-skip as the canonical futures formatting path (already correct). *No change.*

---

### Section 6 — Risk Control Editor  ·  `RiskControlEditor.tsx` (rendered with `isFutures` @ `StrategyStudioPage.tsx:776`)

See the **deep-dive in §4** below.

---

### Section 7 — Prompt Sections Editor  ·  `PromptSectionsEditor.tsx` (rendered `StrategyStudioPage.tsx:780-796`)

**(A) What / wire.** Edits the 4 editable system-prompt chunks (role_definition, trading_frequency, entry_standards, decision_process). On the **crypto** path these feed `BuildSystemPrompt` (`engine_prompt.go:38,93,105,118`); on the **futures** path `BuildFuturesDecisionSystemPrompt` **never reads them** (`engine_prompt_futures.go` has zero PromptSections refs — only `CustomPrompt` at `:111`).

**(C) Render.** No `isFutures` prop; renders identically for MNQ. Defaults are hardcoded crypto ZH text ("你是专业的加密货币交易AI", "候选币").

**(D) Crypto assumptions.** Hardcoded crypto default bodies (`:14-43`); the real coupling is the **backend gap**: a futures user's edits here are **silently inert** on the live system prompt.

**(E) Additive plan.**
1. Thread `isFutures`; swap the 4 defaults to futures-flavored text when `isFutures` (mirror `engine_prompt_futures.go:55-72`), keeping crypto defaults byte-identical. *FE-only.*
2. **BACKEND GAP (real):** make `BuildFuturesDecisionSystemPrompt` honor `PromptSections` with the same "override-or-default" guards the crypto builder uses (`engine_prompt.go:38-126`) so saved overrides actually drive the futures prompt. *BE.*
3. Verify `GetLanguage()` (detects ZH/EN from `RoleDefinition`, `engine.go:243`) still classifies the futures default role text.

---

### Section 8 — Custom Prompt (inline textarea)  ·  `StrategyStudioPage.tsx:797-822`

**(A) What / wire.** Free-text appended to the end of the system prompt. **Already universal & additive:** consumed verbatim by BOTH builders (`engine_prompt.go:153-158` crypto, `engine_prompt_futures.go:111-115` futures); the live ninjatrader path already selects the futures builder. Help/placeholder copy is instrument-neutral.

**(D) Crypto assumption that touches it.** The **preview/test variant dropdown** (Section 12) can never emit `futures`, so when a user previews an MNQ strategy the appended custom prompt is shown inside the **crypto** wrapper, not the futures one.

**(E) Additive plan.** Fix is shared with Section 12 (send `futures` variant for futures). Optional: a `customPromptDescFutures` hint. The field/save/consume path itself needs **no change** — it is already correct.

---

### Section 9 — Publish Settings Editor  ·  `PublishSettingsEditor.tsx` (rendered `StrategyStudioPage.tsx:823-845`)

**(A) What / wire.** Two booleans (`is_public`, `config_visible`) — Strategy-Market marketplace metadata. **Zero trade-path coupling** (grep of `trader/` + `kernel/` for `IsPublic`/`ConfigVisible` returns nothing). The consumer page (StrategyMarketPage) was deleted.

**(C) Render.** Mechanically instrument-agnostic (two booleans + language, no USDT/leverage/$). Renders identically for MNQ (forStrategyType `both`, no `isFutures` gate).

**(D) Crypto assumption.** *Conceptual only* — publishing to a public crypto-retail community board has no futures-desk meaning; the consumer page is gone.

**(E) Additive plan.** Optional: hide the section for futures (add `!isFuturesStrategy` to the `forStrategyType` filter usage) since the marketplace is crypto-community and removed; keep the DB columns + PUT/GET byte-identical. Optional: neutral "Share" i18n variant. **Invariant to preserve: never wire these into the trade path.** *FE-only.*

---

### Section 10 — Grid Config Editor  ·  `GridConfigEditor.tsx` (rendered `StrategyStudioPage.tsx:715-729`, grid mode)

**(A) What / wire.** A self-contained crypto grid editor (symbol, USDT investment, leverage, grid count, price bounds, %-risk, maker-only). Separate strategy_type; never reaches the futures decision path (`RunGridCycle`, not `runCycle`).

**(D) Crypto assumptions (pervasive).** Symbol dropdown is **6 hardcoded USDT pairs** (`:84-91`, default `BTCUSDT` `:15`) — a CME symbol literally cannot be entered; USDT investment label; exchange leverage 1-5; maker-only (post-only); absolute-price `0.01`-step bounds (MNQ tick is 0.25); %-of-equity risk. **Execution gap:** grid needs `PlaceLimitOrder`; the NinjaTrader trader has **no** limit orders (market-only, no `CloseLong/CloseShort` — `trader/CLAUDE.md`), so a futures grid cannot run.

**(E) Additive plan — PARK (recommended).** Leave the grid editor + engine **byte-identical**; do **not** add a futures branch in this universalization pass. The futures product is the AI decision path, not a grid. If futures grid is ever wanted it is an **execution-engine project** (NT8 native limit orders), out of scope here. Honest minimal hardening: a BE guard rejecting `grid_trading` + CME symbol until engine support exists.

---

### Section 11 — Right Panel: Prompt Preview & AI Test  ·  `StrategyStudioPage.tsx:1182-1467`

**(A) What / wire.** "Prompt Preview" builds the system prompt from `editingConfig` (`fetchPromptPreview :624-652` → `POST /strategies/preview-prompt` → `handlePreviewPrompt api/strategy.go:492`). "AI Test" runs the strategy against **real** market data through a **real** AI call, no trade (`runAiTest :654-683` → `POST /strategies/test-run` → `handleStrategyTestRun :548`). Both derive `previewSymbol = static_coins[0] else "MNQ"` (the P3 wiring) and call `BuildSystemPrompt(equity, variant, previewSymbol)`.

**(C) Render — the critical finding (verified live).** The variant `<select>` offers only **Balanced / Aggressive / Conservative**. `BuildSystemPrompt` only emits the futures prompt when `variant=="futures"` (`engine_prompt.go:22`). **Therefore, previewing or testing an MNQ strategy shows the CRYPTO system prompt** — cryptocurrency role, USDT limits, BTC/ETH+altcoin leverage tiers, BTCUSDT/ETHUSDT example JSON — even though the symbol is correctly plumbed. The live path is *correct* (`auto_trader_loop.go:114-115` forces `futures`); this is a **preview-fidelity gap**, not a live bug.

**(D) Crypto assumptions.**
- `config_summary` unconditionally exposes `btc_eth_leverage` + `altcoin_leverage` (`api/strategy.go:540-541`, rendered `:1250-1261`).
- `test-run` unconditionally fetches crypto market-wide OI/NetFlow/Price/Quant data (`api/strategy.go:637-646`); `BuildUserPrompt` does a hardcoded `MarketDataMap["BTCUSDT"]` lookup + USDT PnL labels (`engine_prompt.go:244-247,267`).
- The **one** already-correct piece: `market.GetWithTimeframes` routes an MNQ symbol to the NT8 BarCache, not CoinAnk (`market/data.go:197-205`) — do not touch.

**(E) Additive plan.**
1. When `isFuturesStrategy`, send `prompt_variant:'futures'` to both endpoints **OR** (cleaner) gate inside the handlers: `if market.IsCMEFuturesSymbol(previewSymbol) → force variant='futures'` before `BuildSystemPrompt` (`api/strategy.go:528,678`). The BE gate is more robust (covers all callers). Backend already accepts `futures` — *FE-only if FE forces the variant; small BE change if gated server-side.*
2. Add a 4th "Futures" option (or auto-select/relabel) in the variant `<select>`; keep the 3 crypto tones byte-identical. *FE-only.*
3. Gate `config_summary` to show point_value/tick_size/contract for futures instead of the two leverage rows. *BE (`:537-543`) + FE label.*
4. Skip the crypto market-wide data fetch in test-run for CME candidates (`if !isFutures` around `:637-646`). *BE.*
5. Make `BuildUserPrompt` futures-aware (drop the BTCUSDT line + USD PnL labels) — affects test-run **and** live. *BE.*

---

## 4. Risk Control — deep-dive (the section the desk cares about most)

`RiskControlEditor.tsx:1-354`, rendered with `isFutures={isFuturesStrategy}` at `StrategyStudioPage.tsx:768-778`. Most fields are display-only ("System enforced"); only 4 are user-editable (2 leverage sliders, min-risk-reward, min-confidence). Config flows to the Go risk gate `validateDecision` (`kernel/engine_position.go:30-140`) and to both system prompts.

### Field-by-field, crypto vs futures (live-verified MNQ render + source)

| Field | Crypto (BTCUSDT) | Futures (MNQ today) | Source / gate | Status |
|---|---|---|---|---|
| Position Limits → Max Positions | shown (3) | shown (3) | `RiskControlEditor.tsx:47-60`; enforced `auto_trader_risk.go:260`; futures prompt says "One position at a time" literally (`engine_prompt_futures.go:70`) | universal |
| **Trading Leverage** — BTC/ETH + Altcoin sliders (1-20x) | shown | **hidden** ✅ | wrapped `{!isFutures && …}` `RiskControlEditor.tsx:64-152`; gate ignores leverage for futures (`engine_position.go:48-59`) | **done `8e9b1743`** |
| **Position Value Ratio** — "BTC/ETH Position Value Ratio" 5x + "Altcoin Position Value Ratio" 1x | shown | **STILL shown, crypto labels** ⚠️ | `RiskControlEditor.tsx:154-209` (NO isFutures guard); gate takes futures branch & ignores ratios (`engine_position.go:50-54`) | **RESIDUAL — see §5** |
| Risk Parameters → Min Risk/Reward (editable, default shown 1:3) | shown | shown | `RiskControlEditor.tsx:234-253` → `store…MinRiskRewardRatio`; **but the hard gate hardcodes `≥3.0`** (`engine_position.go:133`) and ignores the field | partially broken (BE) |
| Risk Parameters → Max Margin Usage (90%) | shown | shown | `RiskControlEditor.tsx:261-277`; injected into crypto prompt only (`engine_prompt.go:73`), **not** the futures prompt, **not** the gate | crypto-prompt-only; inert for futures |
| Entry → Min Position Size unit + description | "12 **USDT**" / "…in USDT" | "12 **USD**" / "…in USD" ✅ | `RiskControlEditor.tsx:296-317` (USD/USDT switch `:312`, `minPositionSizeDescFutures`); **BE error string still "USDT"** (`auto_trader_risk.go:249`) | FE done `8e9b1743`; BE residual |
| Entry → Min Confidence (slider 50-100, default 75) | shown | shown | `RiskControlEditor.tsx:331-347`; injected into **both** prompts (`engine_prompt_futures.go:26,69,78`) | universal |

**What "futures risk" actually is (the equivalent to spell out for the dev).** CME futures are sized in **contracts × point value** (`market.FuturesPointValue`, e.g. MNQ = $2/point), **settle in USD**, and have **no exchange leverage** (broker contract margin, effective leverage 1). The Go gate already models this: for a CME symbol it replaces the crypto `equity × ratio` cap with a coarse notional ceiling `equity × futuresMaxNotionalLeverage(20.0)` (`engine_position.go:15,48-54`) and the **executor** does the precise contract sizing. So the FE can safely hide/replace the crypto leverage + ratio controls **without any backend change** — the gate already bypasses them for futures.

### Risk Control — additive change-plan
1. **Gate the Position-Value-Ratio block** (`:154-209`) behind `!isFutures`, mirroring the already-gated leverage block. *FE-only; the gate already ignores these for futures.*
2. **Add a futures risk panel** in their place (rendered when `isFutures`): a read-only "Futures Notional Ceiling = equity × 20" card + a contract-sizing note ("sized in contracts via point value $X/point") sourced from `market.FuturesPointValue`/`instrument_info` (P3/P4). New i18n (3 langs). *Mostly FE; the 20x const lives at `engine_position.go:15` — optionally expose it.*
3. **Instrument-aware subtitle** for the block: keep "…enforced by code" for crypto; for futures "Notional ceiling = equity × 20 (sanity cap); precise sizing is contract-based" — the current "CODE ENFORCED equity×ratio" subtitle *misdescribes* futures. *FE-only (new i18n, mirrors `minPositionSizeDescFutures`).*
4. **BE: instrument-aware min-size error string** ("USD" for futures) at `auto_trader_risk.go:249`, reusing `market.IsCMEFuturesSymbol`. *BE, cosmetic.*
5. **BE: resolve the R/R gate divergence** — have `validateDecision` read `RiskControlConfig.MinRiskRewardRatio` instead of the hardcoded `3.0` (`engine_position.go:133`), or scope the change to the futures branch to keep crypto byte-identical. Today an MNQ strategy whose prompt advertises 1.5 R/R has compliant trades **silently rejected** by the 3.0 gate. *BE (most impactful correctness fix).*

---

## 5. The Position-Value-Ratio residual (headline)

**What it is.** On a futures (MNQ) strategy, Risk Control still renders two cards literally labeled **"BTC/ETH Position Value Ratio"** (5x) and **"Altcoin Position Value Ratio"** (1x), under a subtitle claiming "Position notional value / equity, enforced by code". Verified live (`ss-futures-risk-control.png`) and in source: `RiskControlEditor.tsx:154-209` has **no** `isFutures` guard, even though the leverage block immediately above it (`:64-152`) **is** gated. The capstone gated leverage and the USDT unit but stopped one block short.

**Why it's wrong but not dangerous.** The Go gate branches on `market.IsCMEFuturesSymbol(d.Symbol)` (`engine_position.go:48-54`) and uses the `equity × 20` notional ceiling — it **ignores** `btc_eth_max_position_value_ratio` / `altcoin_max_position_value_ratio` for futures. So these are **inert, mislabelled** controls: the values still persist (DELETE NOTHING — correct), but they describe a crypto sizing model that does not apply to MNQ.

**Additive futures equivalent.** Gate the block to crypto (`!isFutures`) and, in its place for futures, render the read-only "Futures Notional Ceiling (equity × 20) + contract-sizing" panel from §4 item 2. This deletes no crypto capability (the crypto cards render byte-identical when `isFutures=false`, confirmed live with BTCUSDT — `ss-crypto-risk-control.png`) and replaces a confusing crypto label with the real futures sizing model. **FE-only** — the backend gate already does the right thing.

---

## 6. "Done (8e9b1743)" vs "Still crypto-only" — per-section matrix

| Section | Done by `8e9b1743` | Still crypto-only / ungated | Remaining type |
|---|---|---|---|
| 1 Strategy List | nothing | `is_public` tag; env-gated futures seed; no asset-class badge; crypto agent-schema | FE + BE |
| 2 Header | nothing | `ClampLimits` leverage clamp + warnings + ai500 default on every futures Save | BE (harmless today) |
| 3 Token Bar | nothing | no `isFutures`; crypto cost model (funding/quant/ranking/"BTC price") | FE + BE |
| 4 Type selector | nothing | AI-Grid button clickable for futures (grid is crypto-only, can't run on NT8) | FE (+ opt BE guard) |
| 5 Coin Source | **USDT-skip on add** ✅ | AI500/OI tiles + NofxOS panels/badge + crypto placeholders/labels ungated | FE |
| 6 Indicators | **funding-rate toggle hidden; Market-Sentiment subtitle** ✅ | whole NofxOS data block (AI500/OI-Ranking/NetFlow/Price-Ranking) + `cm_568c67…` key ungated | FE |
| 7 Risk Control | **leverage tiers hidden; USDT→USD unit/desc** ✅ | **PVR BTC/ETH+Altcoin labels (residual)**; Max-Margin-Usage inert; BE min-size error "USDT"; R/R 3.0 hardcode | FE + BE |
| 8 Prompt Sections | nothing | crypto ZH defaults; **futures builder ignores all 4 sections** | FE + **BE** |
| 9 Custom Prompt | nothing (already universal) | preview shows crypto wrapper for MNQ (shared w/ §11) | FE/BE (shared) |
| 10 Publish | nothing | conceptually crypto-marketplace (consumer page removed) | FE (optional) |
| 11 Grid | nothing | entire editor + engine crypto-only; **no NT8 execution** | PARK |
| 12 Preview/Test | nothing | **previews/tests the CRYPTO prompt for MNQ**; crypto config_summary; crypto data fetch | FE + BE |
| X1 Backend | nothing (P1-P4 predate it) | variant gated on broker not symbol; env-gated futures default; crypto schema field names persist | BE |

> Indicators (Section 6) was one of the two failed structured sub-agents; its "done/remaining" split is reconstructed from the live render (`ss-futures-indicators.png`: NofxOS block present, funding-rate toggle absent, subtitle "OI and market sentiment data") + capstone diff (`git show 8e9b1743`). The capstone-audit sub-agent also failed; its content is reconstructed from `git show 8e9b1743 --stat` and each surviving agent's per-section "ALREADY DONE" field. See §8 coverage honesty.

---

## 7. Prioritized additive build sequence

Ordered by user-visible value ÷ effort. **FE-only** unless flagged. Each item is additive (crypto stays byte-identical; gate by `isFuturesStrategy` / `market.IsCMEFuturesSymbol`).

**Tier 1 — high value, FE-only, finishes the capstone's intent**
1. **Risk Control PVR residual** — gate `:154-209` to crypto + add the futures notional/contract panel. (§5) *FE.*
2. **Preview/Test futures fidelity** — force `prompt_variant:'futures'` for futures so the desk previews the *real* prompt; gate `config_summary` leverage rows. (§11.E 1+3) *FE force-variant, or small BE gate.*
3. **Coin Source** — thread `isFutures`; show only Static List + hide AI500/OI/NofxOS for futures; futures placeholder/labels. (§5.E) *FE.*
4. **Indicators** — gate the NofxOS crypto data block (AI500/OI-Ranking/NetFlow/Price-Ranking + dead key) for futures. (Section 6) *FE.*
5. **Strategy-Type selector** — hide AI-Grid-Trading for futures. (§4.E) *FE.*

**Tier 2 — correctness (backend), real trade impact**
6. **R/R gate** — read `MinRiskRewardRatio` instead of hardcoded `3.0` (futures-scoped to keep crypto byte-identical). (§4.E 5) *BE — fixes silently-rejected futures trades.*
7. **Live variant by symbol** — choose the futures prompt from the strategy symbol (`market.IsCMEFuturesSymbol`), not just `exchange=="ninjatrader"`, so FE and BE agree. (§Exec) *BE.*
8. **Futures default seed from the UI** — add `asset_class` to create/default-config so New Strategy can be born futures without flipping `TRADING_MODE`. (§1.E 3) *BE.*
9. **Prompt Sections honored by the futures builder** — add override-or-default guards to `BuildFuturesDecisionSystemPrompt`. (§7.E 2) *BE.*

**Tier 3 — polish / hygiene**
10. Header badge + futures-aware token bar + min-size BE error "USD" + test-run crypto-data skip + `BuildUserPrompt` futures-awareness + Publish hide-for-futures. (Mixed FE/BE, low urgency.)

**Park (explicitly out of scope):** Grid Config / grid engine — no NT8 execution path; do not port. Optionally add a BE guard rejecting `grid_trading` + CME symbol.

---

## 8. Coverage honesty

**Live-mapped (real browser at HEAD `24634b5a` + canonical `file:line` re-read by MAIN):**
- **Risk Control** — full futures + crypto render captured (`ss-futures-risk-control.png`, `ss-crypto-risk-control.png`); MAIN re-read `RiskControlEditor.tsx:60-219` confirming the leverage block IS `{!isFutures}`-gated (`:64-152`) and the PVR block is NOT (`:154-209`).
- **Indicators** — futures + crypto render captured (`ss-futures-indicators.png`, `ss-crypto-indicators.png`): funding-rate toggle hidden / subtitle "OI and market sentiment data" for MNQ; NofxOS block + `cm_568c67…` present in both.
- **Preview variant dropdown** — confirmed Balanced/Aggressive/Conservative only (accessibility snapshot), and MAIN re-read `engine_prompt.go:22-24` (futures requires `variant=="futures"`).
- **Risk gate** — MAIN re-read `engine_position.go:1-140`: futures branch (`:48-54`), `futuresMaxNotionalLeverage=20` (`:15`), hardcoded R/R `3.0` (`:133`).
- **Live variant selection** — MAIN re-read `auto_trader_loop.go:113-116` (broker-gated, not symbol-gated).
- **isFutures threading** — MAIN re-read `StrategyStudioPage.tsx:731-796`: `isFutures` passed to Indicator (`:758`) + Risk (`:776`) only; CoinSource/PromptSections get none.
- **Crypto-regression** — swapping the scratch strategy's coin MNQ→BTCUSDT brought leverage tiers, USDT units, and funding-rate back (live), proving the crypto path is byte-identical.

**Sub-agent-mapped, sampled-but-not-exhaustively-re-read by MAIN:** the full `file:line` wire tables for Sections 1-4, 8-12, and the backend cross-cutting map come from the 12 structured sub-agent maps (each "CONFIRMED in source" with grep+Read; CGC returned sparse hits for Go handlers so source was canonical). MAIN re-verified the five **headline** wires above and spot-checked several others; the long-tail citations (e.g. exact `store/strategy.go` line numbers for every default) are sub-agent-reported and should be treated as high-confidence-but-verify-on-touch.

**Reconstructed (the 2 failed structured agents):** **Indicators** and the **capstone done-audit** sub-agents completed their analysis but did not emit the StructuredOutput object. Their content here is reconstructed from (a) the live Indicators render in both modes, (b) `git show 8e9b1743 --stat` + commit body, and (c) the per-section "ALREADY DONE" fields the 10 surviving agents each reported. The capstone scope ("leverage hidden, USDT→USD, funding-rate hidden, isFutures threaded to 2 editors + DecisionCard, 2 i18n keys") is corroborated independently by every surviving agent and by MAIN's source reads — high confidence.

**Inferred / flagged (not runtime-proven):** that grid cannot execute on NT8 (read from `trader/CLAUDE.md` + absence of `PlaceLimitOrder` in `trader/ninjatrader/*.go`, not run); that selecting AI500/OI for a futures strategy yields no useful candidates (architectural, not traced to an empty result); exact `ClampLimits` numeric ranges (call site confirmed, body not opened). None of these affect the change-plan's direction.

---

## 9. Backend gaps consolidated (for planning)

These are the only items that are **not** FE-only. Each is additive (gate, don't remove crypto):

| # | Gap | Location | Impact |
|---|---|---|---|
| BE-1 | Live prompt-variant chosen by `exchange=="ninjatrader"`, not by symbol | `auto_trader_loop.go:114` | FE/BE signal divergence; futures symbol on non-NT8 trader → crypto prompt |
| BE-2 | Hard R/R gate hardcoded `≥3.0`, ignores `MinRiskRewardRatio` | `engine_position.go:133` | futures trades at advertised 1.5 R/R silently rejected |
| BE-3 | Futures default coin seed env-gated (`TRADING_MODE`), not request/per-strategy | `store/strategy.go:935-953`, `api/strategy.go:479` | can't create a futures strategy from the UI in crypto mode |
| BE-4 | `BuildFuturesDecisionSystemPrompt` ignores `PromptSections` | `engine_prompt_futures.go` | futures users' prompt-section edits are inert |
| BE-5 | Preview/test-run + `BuildUserPrompt` crypto framing (config_summary leverage, BTCUSDT lookup, USDT labels, crypto data fetch) | `api/strategy.go:537-543,637-646`; `engine_prompt.go:244-247,267` | preview infidelity (BE-side of §11) |
| BE-6 | Save-time `ClampLimits` + warnings + ai500 default ungated for futures; min-size error string "USDT" | `store/strategy.go:81-93,544,571-572,139-178`; `auto_trader_risk.go:249` | harmless today (bypassed), cosmetic |

Everything else in the build sequence is **FE-only conditional rendering** reusing `web/src/lib/instrument.ts` (`isCMEFutures` + `cmeFuturesRoots`), `market.FuturesPointValue`/`FuturesTickSize`, the futures prompt, and `instrument_info` — **no new hardcoded symbols or dollar values.**

---

*End of report. Screenshots (local, git-ignored, under `.playwright-mcp/`): `ss-futures-risk-control.png`, `ss-futures-indicators.png`, `ss-crypto-risk-control.png`, `ss-crypto-indicators.png`, `futures-risk-control.png`.*
