# STRATEGY PAGE FULL CONTROL CENSUS — 2026-08-19 (read-only)

**Branch:** `docs/strategy-controls-census` · **Artifact:** this report only, zero code changes.
**Method:** 4 parallel enumeration/trace sweeps (Studio FE · trader-surface FE · Go strategy-config consumers · trader columns + env), then an **adversarial verification workflow** — 10 refuter agents re-reading every DEAD/PARTIAL/SHADOWED/clamp claim against the code: **39 claims → 28 CONFIRMED · 11 CORRECTED (detail fixed, substance held) · 0 REFUTED**. Every verdict below survived that pass; corrections are folded in. Evidence tier [A] throughout unless marked.
**Baseline inherited:** the 2026-08-17 control-inspection ("11 DEAD CONTROLS FOUND — 11 FIXED, 0 LEFT DEAD", commits `bc360a38`/`d396201e`/`e943f9c3`) walked the day-plan surface; its post-fix verdicts are inherited here, not re-litigated.

**Headline numbers:** 246 interactive FE controls walked (129 Studio + 117 trader-surface) · 18 `traders` columns traced (7 DEAD legacy columns — none has a UI control) · ~45 strategy-config keys traced · 28 env/code-only knobs (6 shadow-config, 4 dead) · **2 dead Studio toggles**, **1 FE persist gap**, **2 silent-shadow behaviors** newly registered.

---

## PART 1+2 — MASTER TABLE (control → writes-to → verdict → what it actually does)

Verdicts: **LIVE** (consumed, behavior stated) · **DEAD** (saved, zero consumers) · **PARTIAL** (one path only) · **SHADOWED** (literal/env overrides) · *action* (button, no config key). "Bundle" column = Part-3 overlap verdict where relevant, else —.

### 1A · Strategy Studio — Risk Control (`RiskControlEditor.tsx`, all keys land at `strategies.config → ai_config.risk_control.*` via `PUT /api/strategies/{id}`; nesting per `store/strategy.go:793`)

| Label (file:line) | Key | Verdict | What it actually does | Bundle |
|---|---|---|---|---|
| 🔒 Hold discipline (RiskControlEditor.tsx:170) | `hold_discipline` | **LIVE** | ON + matching open position → AI `close_long`/`close_short` is suppressed (`holdLockSuppressesClose` trader/auto_trader_orders.go:77-105, called auto_trader_loop.go:664-671); trade rides to its NT8 bracket. Emergency-Flat and the drawdown monitor bypass it by design. | position_mode: RELATED |
| 🎯 Move stop to breakeven (:190) + Trigger points (:203) | `breakeven_enabled`, `breakeven_trigger_points` | **PARTIAL (NT8-only, by design)** | At +N pts (default 50) the resting stop is moved to entry once per position via the `move_stop` wire frame; early-returns unless `exchange=="ninjatrader"` (trader/auto_trader.go:148-151). Live-proven 6× (PR #52). | TRAIL: RELATED |
| Max Positions (:241) | `max_positions` | **LIVE** | Cycle HOLDs at the cap (kernel/engine_analysis.go:126-137) AND each open is refused executor-side (auto_trader_risk.go:292-306). Stored 0 → clamped to **1** each cycle (store/strategy.go:149-151). Futures prompt separately hardcodes "one position at a time" (engine_prompt_futures.go:123). | — |
| BTC/ETH / Altcoin leverage sliders (:295/:332, crypto-only UI) | `btc_eth_max_leverage`, `altcoin_max_leverage` | **PARTIAL (crypto-only)** | Clamp AI-requested leverage per symbol class (kernel/engine_position.go:49-75); NT8 path drops leverage entirely (tcp_trader.go:176-182, prompt says "always 1"). UI already hides on futures — consistent. | — |
| Min Risk/Reward Ratio (:453) | `min_risk_reward_ratio` | **LIVE (both venues)** | Rejects entries below the ratio measured from the real entry reference (engine_position.go:137-181); unset → 3.0. | — |
| Max Margin Usage (:489, crypto-only UI) | `max_margin_usage` | **PARTIAL (prompt-only, zero enforcement)** | One advisory line in the crypto system prompt (engine_prompt.go:86). No Go code blocks or resizes on it; futures prompt never mentions it. FE label already says "AI-guided, not enforced" — honest. | — |
| Min Position Size (:556, crypto-only UI) | `min_position_size` | **PARTIAL + SHADOWED** | Enforced only on the crypto executor branch (auto_trader_risk.go:275-289); futures branch skips it. ALSO shadowed: kernel validator rejects on hardcoded 12/60 USDT literals regardless of the configured value (engine_position.go:82-93). | — |
| Min Confidence (:603) | `min_confidence` | **LIVE + display lie at 0** | Refuses opens below the number (engine_position.go:190-191). Stored 0 is clamped **up to 65** every cycle (`SafeDefaultMinConfidence=65`, store/strategy.go:75,209-210, applied engine_analysis.go:233) — a slider showing 0 is actually gating at 65. | — |
| Risk guardrails MASTER (:639) | `guardrails_enabled` | **LIVE (default ON when unset)** | `boolOrDefault(rc.GuardrailsEnabled, true)` (engine_analysis.go:150). OFF bypasses daily-loss/profit/max-trades **and blackout and consistency** in one else-if chain (:163-195). | — |
| Daily loss limit (:655/:658) | `daily_loss_enabled`, `daily_loss_limit_usd` | **LIVE (master-gated) + silent env fallback** | Halts new cycles at the session-day realized-loss limit. **Value 0 → env `RISK_MAX_DAILY_LOSS_USD` (default $500) silently becomes the enforced limit** via `firstPositive` (engine_analysis.go:155, risk_limits.go:245) — no log announces the substitution. | — |
| Daily profit target (:682/:685) | `daily_profit_*` | **LIVE (master-gated, 0=off)** | Halts new cycles at the target; no env fallback. | — |
| Max daily trades (:709/:712) | `max_daily_trades*` | **LIVE (master-gated, 0=off)** | Blocks cycles at the CME-session-day entry count (count source auto_trader_loop.go:972-981). | — |
| Consecutive-loss halt (:733/:736) | `consecutive_loss_halt` | **LIVE (master-INDEPENDENT)** | After N losing closes this session-day, new entries refused (closes allowed), P0 alert; fails open on DB error (auto_trader_orders.go:112-130, entries-only switch :243). | — |
| Re-entry cooldown (:761/:766, futures-only UI) | `reentry_cooldown_minutes` | **LIVE (arms NT8-only)** | B7: after a stop-loss exit, same-direction re-entry becomes `wait` until minutes elapse OR price moves ≥1×ATR15 off the stop (engine_analysis.go:479,489-517; discipline/reentry_cooldown.go:74). **Arming happens only in the NT8 close path** (`NoteStopLossExit` sole caller close_sync.go:161, `ExitReason=="sl"`) — crypto never arms it. FE futures-only visibility is therefore correct. 0 = off (owner's current state). | POST_EXIT_RESCAN: RELATED |
| Consistency cap (:791/:794) | `consistency_*` | **LIVE but SILENTLY master-shadowed** | Goes passive once today's profit ≥ N% of all-time profit — but sits in the master else-if chain: master OFF disables it **with no "would have tripped" soft log** (CheckSoft covers only loss/profit/trades, risk_limits.go:197-209). | — |
| Max contracts / order (:819/:822) | `max_contracts_per_order` value / `max_contracts_enabled` toggle | value **PARTIAL (futures-only)** · toggle **DEAD** | Value clamps contracts per NT8 order; effective default **2** (`maxFuturesContracts=2.0`, auto_trader_orders.go:25; resolver risk_limits.go:260-265). **The toggle has ZERO readers** — clamp is deliberately always-on (comment risk_limits.go:255-259). Stale "10-contract default" comments lie at risk_limits.go:258, auto_trader_orders.go:50, store/strategy.go:1369-70. | — |
| Notional cap (:846/:849) | `max_notional_leverage` value / `notional_cap_enabled` toggle | value **PARTIAL (futures-only)** · toggle **DEAD** | Value caps futures notional at equity×N (default 20, always-on, master-independent; engine_analysis.go:438 + auto_trader_risk.go:245). **Toggle has ZERO readers.** | — |
| Blackout start/end CT (:874-:894) | `blackout_*` | **LIVE but SILENTLY master-shadowed** | No new decisions inside the daily CT window (engine_analysis.go:179-186); master OFF kills it with no soft alert (same else-if chain as consistency). | — |

### 1B · Strategy Studio — other sections (counts; full per-control rows live in the sweep tables, all verified consistent)

| Section | Controls | Verdict summary |
|---|---|---|
| Page-level (StrategyStudioPage.tsx) | 21 (14 action-only) | Name/description → `strategies` columns; strategy-type cards → `strategy_type` (**LIVE** — grid_trading routes every tick to the grid cycle, auto_trader.go:803-815; fully wired but **zero strategies use it today**); Style dropdown → `prompt_variant` (**LIVE**, prompt router engine_prompt.go:27, empty → ninjatrader→futures fallback); Extra Prompt → `ai_config.custom_prompt` (**LIVE both venues**, appended verbatim engine_prompt.go:169 / futures:236). |
| IndicatorEditor | 35 | Enable flags **LIVE** (each adds/removes its prompt block); periods **LIVE** (drive the actual math, engine_analysis.go:642-647); `enable_funding_rate` **PARTIAL** (force-suppressed on futures, engine_prompt.go:265,652); `enable_svp` **PARTIAL** (futures-prompt-only; renders at engine_prompt_futures.go:169 AND :247); NofxOS block crypto-only (key `cm_568c…` returns HTTP 402 — known-dead service); Raw-OHLCV checkbox rendered permanently `disabled={true}` (:933) with the value force-set in code. |
| CoinSourceEditor | 13 | **LIVE** — the SourceType switch (kernel/engine.go:366-430) reads every field; futures locked to Static List (UI) matching the futures data path. |
| PromptSectionsEditor | 12 | **LIVE all four sections, both venues** (crypto engine_prompt.go:51-134; futures engine_prompt_futures.go:96-200). |
| PublishSettingsEditor | 2 | `strategies.is_public`/`config_visible` columns — UI/market only, no trading consumer. |
| GridConfigEditor + GridRiskPanel | 15 + 1 | All keys consumed inside the grid cycle files — **LIVE code path, zero live users** (no `grid_trading` strategy exists in the DB). |
| DayPlanEditor | 26 | **Inherited from the 2026-08-17 inspection: 0 dead post-fix** (session enable/acceptance/max_trades/last_entry/eod_flat/realign_cap all consumed via `sessionRunnable()` + per-session resolvers). `day_plan.plan_enabled` is additionally the **arming switch for cadence_mode, skip-while-open, and the P2 clock** — see Part 3.1. |

### 1C · Trader surfaces (TraderConfigModal / Dashboard / Settings)

| Label (file:line) | Writes to | Verdict | What it actually does | Bundle |
|---|---|---|---|---|
| AI Scan Decision Interval (TraderConfigModal.tsx:490) | `traders.scan_interval_minutes` | **LIVE** | Sets the decision-loop ticker period = the paid-AI cadence under P10 interval mode (manager/trader_manager.go:651 → auto_trader.go:799). | cadence: SAME |
| Cadence Interval/Bar-close (:518/:531) | `traders.cadence_mode` | **PARTIAL ×2** | (a) Go: both branches inert unless futures + `day_plan.plan_enabled` (auto_trader_clock.go:564-579 gate on `dayPlanEnabled()` :22-28) — crypto always plain interval. (b) **FE persist gap: saves on CREATE only** — `handleSaveEditTrader` builds the PUT body without `cadence_mode` (web/src/components/trader/AITradersPage.tsx:257-265) though the Go update handler fully supports it (handler_trader.go:53,648-650, store/trader.go:154-155). An owner cannot switch an existing trader's cadence from the UI. | cadence: SAME |
| Cross/Isolated margin (:460/:471) | `traders.is_cross_margin` | **PARTIAL (crypto-only)** | `SetMarginMode` before each crypto entry; NT8 impl is `return nil` no-op (tcp_trader.go:458-460). | — |
| Trader name / AI model / Exchange / Strategy (:256-:356) | `traders.name/ai_model_id/exchange_id/strategy_id` | **LIVE** | Model+key selection (disabled model → trader silently refuses to load, trader_manager.go:450-473); venue+credentials; strategy supplies ALL real risk caps (empty → hard load error :632). | — |
| Show in Competition (:566/:579) | `traders.show_in_competition` | LIVE (non-trading) | Leaderboard aggregation only. | — |
| ⏸ Pause 30m/1h/session-end/custom + ▶ Resume (PauseButton.tsx:100-145) | `system_config` `trader_pause_until:<id>` | **LIVE** | P2 stop_until producer: new entries refused until the deadline; stops/targets/EOD-flat/monitor continue; survives restart; auto-resumes on expiry (auto_trader_pause.go). | — |
| Emergency Flat (EmergencyFlatButton.tsx:57,108) | none (broker + kernel kill-switch) | **LIVE** | Flattens via the trader path — deliberately bypasses hold-lock (RECON #10). | — |
| Account selector (AccountSelector.tsx:206) | `traders.account` | **LIVE (futures) + env-SHADOWED** | Empty → whole cycle skipped (auto_trader_loop.go:138-146); unbound entry refused (tcp_trader.go:253-256); rows disabled unless SIM; **`NT_ALLOWED_ACCOUNTS` env can override the persisted choice at selection AND order time** (handler_account.go:169-182, tcp_trader.go:233-239,257). | — |
| Close position (TraderDashboardPage.tsx:871) | broker order | **LIVE** | Manual close via the close path. | — |
| Day-plan door: session tabs, Ask Planner, Edit/Add/Bulk sheets, Re-align, Re-read, Reset, Alerts (24 controls, PlanCard tree) | `plan_overlays` / `owner_levels` / `plan_qa` / `plans` / `day_plan_alerts` | **LIVE** | The P5 overlay/ask surface — inherited from the 08-17 inspection + P5 hardening; all doors gated on `doorEnabled` (live session, non-historical). | — |
| Settings: model keys / exchange bindings incl. `nt_data_dir`, `nt_instrument_name`, `nt_default_contract_qty` (SettingsPage.tsx + modals) | `ai_models.*`, `exchanges.*` | **LIVE** | Feed credentials + the NT8 data dir/instrument that the trader actually uses (trader_manager.go:712) — NOT the env var (see knob list). | — |

### 1D · `traders` legacy columns with NO UI control — all dead at runtime [A, refuter-verified]

`btc_eth_leverage`, `altcoin_leverage` (create/update range-validation + GET echo only; live clamp = strategy RiskControl) · `trading_symbols` (create-only USDT format check — **the update path at handler_trader.go:687 skips even that**; universe = CoinSource) · `use_coin_pool`, `use_oi_top` (kernel reads the strategy twins) · `custom_prompt` (loaded into `at.customPrompt`, **zero reads**) · `override_base_prompt` (read only to pick a log line, trader_manager.go:744) · `system_prompt_template` (GetSystemPromptTemplate derives from the strategy, auto_trader.go:943-951). Bonus finding: `AutoStartRunningTraders` (trader_manager.go:118-131) is **dead code** — the real boot restore reads `is_running` directly at :755.

---

## PART 3 — SEMANTIC DUPLICATE SCAN (the five bundle controls)

### 3.1 `position_mode` (ai_watch / bracket_only) — **verdict: SAME. Wire the existing mechanism; do not add a twin gate.**

The "bracket_only" behavior ALREADY EXISTS, hardcoded: `skipWhileOpen()` ([trader/auto_trader_clock.go:94](../../trader/auto_trader_clock.go)) skips the entire AI decision while holding, gated on `dayPlanEnabled()` (:22-28 — futures + `day_plan.plan_enabled`). Today every day-plan futures trader IS `bracket_only`; every crypto/plan-off trader IS `ai_watch`. The governing control is `day_plan.plan_enabled` — a control whose *name says nothing about in-position AI* (the hold-lock lesson, live example).
**Instruction to the bundle:** implement `position_mode` as the conditional INSIDE `skipWhileOpen` (mode `ai_watch` → return false; `bracket_only`/default → current behavior), never as a parallel gate elsewhere. Required interactions to document in help text: (a) in `ai_watch`, AI closes come back **unless `hold_discipline` suppresses them** — two controls, one interaction; (b) `ai_watch` restores per-cycle paid AI calls while holding (cost: the E6-measured 24.2 calls/h ceiling applies); (c) the skip's desync guard (`skipGateDesync`, PR #50) must keep running in both modes.

### 3.2 Hold-lock / discipline button — Phase-0 trace (full)

| Step | Fact [A] |
|---|---|
| Label | "🔒 Hold discipline (hold-lock)" — RiskControlEditor.tsx:170-174, always visible, `Toggle` at :88 |
| Field | `risk_control.hold_discipline` → `ai_config.risk_control.hold_discipline` (`HoldDisciplineEnabled *bool`, store/strategy.go:1390) |
| Consumers | `holdLockSuppressesClose` trader/auto_trader_orders.go:77 (toggle read :84 via `hlBool(...,false)` — default OFF), sole call site auto_trader_loop.go:665 |
| Behavior | Suppresses AI `close_long`/`close_short` while a protected position is open; logged `🔒 HOLD-LOCK`, recorded as deliberate no-op; bracket exits untouched |
| Bypasses | Emergency Flat (handler_risk direct trader call) + drawdown monitor `emergencyClosePosition` — position management never trapped |
| Verdict | **LIVE [A]** — not dead, not shadowed. The bundle inherits this answer: `position_mode=ai_watch` + `hold_discipline=ON` is a *coherent* combination (AI may talk, may not close). |

### 3.3 Trailing stop / `TRAIL_ATR_MULT` — **verdict: CLEAR (no twin exists), RELATED to breakeven.**

Zero trailing-stop logic anywhere: no `trail` in Go trader/kernel, zero in `VLTraderTCPClient.cs` [A]. The only stop-management primitive is auto-breakeven's one-shot `move_stop`. **Instruction:** build the trail ON the existing `move_stop` frame + `HandleMoveStop` (in-place `Change`, OCO preserved, no-op on ~equal) — do NOT invent a second stop-amendment path; define precedence with breakeven explicitly (single writer; a trail must never move a stop backward, matching breakeven's never-backward semantics); place the controls beside the breakeven block in RiskControlEditor with the same NT8-only scoping (`exchange=="ninjatrader"` early-return); prefer a Studio field over the `TRAIL_ATR_MULT` env name (env would be a new shadow-config — this census exists to prevent that class).

### 3.4 `POST_EXIT_RESCAN` — **verdict: RELATED (add new; B7 keeps veto; dedup interaction must be explicit).**

Existing post-trade machinery: **B7 `reentry_cooldown_minutes`** (LIVE; arms only on NT8 stop-loss exits, converts same-direction re-entry to `wait`, escape at 1×ATR15 or timer; currently 0=OFF) and the P10 `skipNoNewData` dedup. A rescan-after-exit is the *opposite* force (act sooner) but shares the domain. **Instruction:** the rescan must trigger a cycle that flows through the normal gate chain so B7 (and stop_until, roll-block, session gates) retain veto; it must explicitly bypass `skipNoNewData` (else a rescan inside the same bar signature silently no-ops — document which); help text must state the B7 relationship ("rescan may fire; cooldown may still refuse the entry").

### 3.5 `STALE_DODGE` / re-eval — **verdict: RELATED (different stage than every existing staleness control).**

Existing staleness stack, none of it Studio-visible: B4 stale-entry gate (`STALE_BAR_GRACE_S` env, default 15s, kernel/stale_data.go:50), P7 single-snapshot instant + prompt honesty, P8 fail-open on missing snapshot, in-position feed alert (`INTRADE_FEED_ALERT_S`=120), and `skipNoNewData`. All of these gate BEFORE the AI call; a stale-dodge re-evaluates BEFORE EXECUTING an aged decision — a stage nothing currently covers (the ~60-160s AI-call latency window). **Instruction:** add as new, defined against the P7 snapshot timestamp (`ctx.SnapshotMs` is already stamped for exactly this comparison); do not duplicate or re-parameterize `STALE_BAR_GRACE_S`; surface it in Studio, not env.

### 3.6 `scan_interval` / `cadence_mode` — **verdict: SAME. Add NOTHING; fix the FE gap instead.**

Post-P10 confirmed state: `scan_interval_minutes` LIVE (the real decision cadence, owner-ruled); `cadence_mode` LIVE with "interval" default and "bar_close" the legacy opt-out — armed by futures+day-plan. **The one wiring gap is FE:** the edit path drops `cadence_mode` (1D above). The bundle must not introduce any new cadence/timing control; if it needs "re-eval timing" it builds on `cadenceMode()`/`tickOnce` (auto_trader_clock.go:557-580).

---

## PART 4a — DEAD / PARTIAL / SHADOWED REGISTER (proposed fixes, sized, NOT implemented)

| # | Finding | Class | Proposed fix | Size |
|---|---|---|---|---|
| 1 | `max_contracts_enabled` toggle: saved, zero readers (clamp deliberately always-on) | DEAD control | Remove the FE toggle or render it as a fixed "always on" chip with help text; do NOT wire it to the clamp (venue-safety is meant to be un-toggleable) | S |
| 2 | `notional_cap_enabled` toggle: saved, zero readers (cap always-on) | DEAD control | Same treatment as #1 | S |
| 3 | Cadence toggle persists on create only — edit PUT drops `cadence_mode` (AITradersPage.tsx:257-265; backend fully supports it) | PARTIAL (FE gap) | Add `cadence_mode: data.cadence_mode` to the update request body | S (1 line + test) |
| 4 | Min Confidence slider at 0 actually gates at 65 (`SafeDefaultMinConfidence`) | SHADOWED display | FE: show "0 → uses safe default 65" helper; or clamp the slider min to 10 with explicit 65-default hint | S |
| 5 | Daily-loss value 0 → env $500 silently enforced (`firstPositive`, no log) | SHADOWED (silent fallback) | One boot/log line when the env fallback is active + Studio helper text ("empty = $500 default") | S |
| 6 | Blackout + consistency silently dead while guardrails master OFF (no would-trip soft log; CheckSoft covers only loss/profit/trades) — **owner's live state is master OFF** | SHADOWED (master) | Extend `CheckSoft` to blackout+consistency so master-OFF still logs would-trips | S–M |
| 7 | `min_position_size`: futures branch skips it AND kernel literals 12/60 USDT override it everywhere | PARTIAL+SHADOWED | Thread the configured value into `validateDecision` (keep 12/60 as floors); or label the field crypto-advisory | M |
| 8 | Stale "10-contract default" comments (real default 2) at risk_limits.go:258, auto_trader_orders.go:50, store/strategy.go:1369-70 | comment truth | Fix the three comments | S |
| 9 | 7 dead `traders` columns + write-only `at.customPrompt`/`overrideBasePrompt` fields + dead `AutoStartRunningTraders` fn | DEAD (legacy, no UI) | Deprecation sweep: drop the loads, mark columns legacy in store comments; do NOT drop columns from SQLite (migration risk) | M |
| 10 | `trading_symbols` update path skips even the create-time USDT format check (handler_trader.go:687) | latent (column dead anyway) | Fold into #9's sweep | S |
| 11 | `NINJATRADER_DATA_DIR` env dead on the live path, and auto_trader.go:584's error text still tells the owner to set it | misleading error | Reword the error to point at the exchange-row `nt_data_dir` | S |
| 12 | `RISK_MAX_NOTIONAL_USD` + `RISK_MAX_CONTRACTS_PER_ORDER` + `DATABENTO_DATASET` env: loaded, enforced nowhere | DEAD env | Delete from config load, or comment them as reserved | S |

Deliberate non-findings (checked, working as designed): Max-Margin-Usage's "AI-guided, not enforced" label is honest; funding-rate/OI/NofxOS crypto-only hiding matches the Go suppression; B7's futures-only FE visibility matches its NT8-only arming; the Raw-OHLCV checkbox is intentionally forced-on.

## PART 4b — Env/code-only knobs (no Strategy-page control; owner may want surfaced)

**Trading-behavior (19):** `TRADING_MODE` (crypto|futures master switch) · `NT_TRANSPORT` (csv|tcp) · `POSITION_RECONCILE` (default ON — desync guard, PR #50) · `INTRADE_FEED_ALERT_S`=120 · `STALE_BAR_GRACE_S`=15 · `ROLL_BLOCK_DAYS_BEFORE_EXPIRY`=3 · `CLOCK_WARN_MS`=30000 · `NOFX_EXPECTED_REVISION` (boot-integrity entry refusal) · `NOFX_CALENDAR_STATIC` / `NOFX_HALF_DAYS` (calendar files) · `AI_BALANCE_WARN` (off) · `CLAW402_WALLET_KEY`/`CLAW402_URL` · `SANDBOX_MODE`/`SANDBOX_LLM` (demo shadow) · `RISK_MAX_DAILY_LOSS_USD`=500 (silent fallback, #5) · `RISK_MAX_CONCURRENT_TRADES`=2 (fallback when `max_positions` unset — but note the 0→1 clamp usually wins first) · `NT_TCP_LISTEN_ADDR` · `ALLOW_ACCOUNT_RESET` (destructive, off).
**Shadow-config over UI controls (4):** `NT_ALLOWED_ACCOUNTS` (overrides the account picker + persisted binding at order time) · `NT_EXTRA_SYMBOLS` (appends after the authoritative symbol list) · `NT_RUNTIME_SYMBOLS` (unset → runtime symbol API refuses) · `SANDBOX_MODE`.
**Dead reads (4):** `RISK_MAX_NOTIONAL_USD` · `RISK_MAX_CONTRACTS_PER_ORDER` · `NINJATRADER_DATA_DIR` · `DATABENTO_DATASET`.
**Infra-ish (flagged, not sized):** `LOG_DB_RETENTION_DAYS`, `NOFX_CLOCK_STATE`.

Surfacing candidates if the owner wants Studio control: `ROLL_BLOCK_DAYS_BEFORE_EXPIRY`, `INTRADE_FEED_ALERT_S`, `STALE_BAR_GRACE_S`, `AI_BALANCE_WARN` — each already has a clean single read site.

## PART 4c — Bundle placement summary (one line each)

| Bundle control | Ruling | One-line instruction |
|---|---|---|
| `position_mode` ai_watch/bracket_only | **SAME** | Wire the mode into `skipWhileOpen` (auto_trader_clock.go:94); document the `hold_discipline` and cost interactions; no new gate. |
| Trailing stop / `TRAIL_ATR_MULT` | **CLEAR + RELATED** | New control, but built on the existing `move_stop` frame with never-backward + BE precedence; Studio field, not env. |
| `POST_EXIT_RESCAN` | **RELATED** | New; must flow through the gate chain (B7 veto) and explicitly bypass `skipNoNewData`. |
| `STALE_DODGE` / re-eval | **RELATED** | New stage (pre-execution re-eval vs P7 `SnapshotMs`); don't re-parameterize `STALE_BAR_GRACE_S`; Studio, not env. |
| `scan_interval` / `cadence_mode` | **SAME** | Add nothing; fix the FE edit-path drop (register #3) as the bundle's only cadence work. |

## PR

Report-only PR on `docs/strategy-controls-census` — number parsed from the `gh pr create` output URL, stated in the chat delivery.
