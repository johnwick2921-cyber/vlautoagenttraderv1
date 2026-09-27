# 2026-09-03 — Studio settings audit: every surface, every field, one table

**Dispatch:** full Studio settings audit. READ-ONLY · no lock · no code.
**Tree:** worktree `~/nofx-studioaudit`, branch `docs/studio-audit-0903`, base `b5b29ac3` (dev). Three parallel censuses: UI (`web/src/components/strategy/*`), schema/validators (`store/strategy.go`, `api/strategy.go`), engine readers (`kernel/*`, `trader/*`). All rows carry file:line evidence; the five highest-impact findings were re-verified directly [A].
**Conventions:** "reader NONE" = no production code consumes the value. "inherits" = the knob's resolver falls back to the strategy-level / default when unset. Legend for ISSUES: 🟥 dead · 🟧 silent rewrite/override · 🟨 label mismatch · 🟦 resolution split.

---

## THE ONE TABLE (grouped by surface)

### Surface A — Day Plan (DayPlanEditor)

| field | UI values | schema/validator | engine reader | resolution (two levels) | issues |
|---|---|---|---|---|---|
| `plan_enabled` | toggle | bool, default false | `auto_trader_clock.go:28` (master switch) | — | |
| `planner_model` | free text, placeholder "inherit primary" | string, no validation | `auto_trader_planner.go:71`; empty → primary model (`:33-37`) | — | 🟨 free text, no list of valid models; typo silently falls back to primary |
| `plan_mode` | advisory/direction/strict segmented | no enum validation | `auto_trader_planconfig.go:197-228` `planModeBlocked` | **session override → base → advisory** | 🟦 session override IGNORED at the arm seam (`armed_executor.go:1222` calls `planModeFor("")`) — the strategy-level value still applies to arms even when a session overrides it |
| `planner_timeframes` | chips D/4h/1h/15m | `[]string`, no validation | `auto_trader_planner.go:1985` | — | |
| `proximity_filter_atr` | 0.1–3.0 step 0.1 | no save clamp; read clamp `[0.1,3.0]→1.5` `kernel/plan_lifecycle.go:25-33` | `auto_trader_planconfig.go:139`, `engine_analysis.go:375` | — | 🟨 schema comment says 0.5–3.0 (stale); actual floor 0.1 |
| `max_levels` | 3–12 number | `≤0→8`, hard ceiling 12 (`plan_doc.go:359`) | `auto_trader_planner.go:1982`, `engine_analysis.go:366` | — | |
| `scenario_cap` | 1–5 number | read clamp `[1,5]→3`; ceiling 5 | `auto_trader_planconfig.go:145-150` | — | |
| `acceptance_rule` | segmented, **single option** "1×5m" | self-heals stored `2x5m`/`15m-close` → `5m_close` (`strategy.go:1057-1134`) | `AcceptanceRuleFor` → levelstate/planner/gates | session → base → `5m_close` | 🟨 a single-option UI is a dead selector; the shipped default block still says `2x5m` (`strategy.go:1410` vs `:1050-1055`) |
| `replan_cap` | 0–4 number | no enum validation; `0` = no re-plans | `auto_trader_planconfig.go:47-49` | session → base → 2 | |
| `approval_required` | toggle | bool | `auto_trader_planconfig.go:152-156` → HELD at `auto_trader_orders.go:309-321` | — | |
| `evening_digest` | toggle, default on | bool | `auto_trader_planconfig.go:159-166` | — | |
| `realign_cap` | 0–10 number | none | `auto_trader_planconfig.go:228-236`; enforced at API `handler_plan.go:2120` | — | API-time knob, not a cycle knob |
| `wake_on_15m_zone` / `htf_zone` / `seated_invalidation` / `ifvg` | toggles (absent=ON) | `*bool` nil→ON | `auto_trader_wake_levels.go:99/124/176/156` | — | |
| `wake_on_htf_ob` | toggle (absent=OFF) | bool default false | `auto_trader_wake_levels.go:140` | — | |
| `wake_min_interval_min` | 5–120 number | none at save; `≤0→30` (`strategy.go:1483`) | `auto_trader_wake_levels.go:260` | — | 🟨 struct comment "≤0 → 10" is stale — actual 30 |
| `seat_1h_zone` | toggle (absent=ON) | `*bool` nil→ON | `engine_analysis.go:401`, `planner:2130` | — | |
| `min_scenario_quality` | A/B/C segmented (default C) | enum **unvalidated** | `auto_trader_loop.go:443` → gates | session → base → C | 🟨 invalid stored value acts like C silently |
| `last_entry_ct` / `eod_flat_ct` | **hidden** (C3 legacy) | strings, defaults 13:00/14:45 | **READER NONE** (`auto_trader_clock.go:319-324,356-363`) | live replacements: per-session `last_entry_offset_min`/`eod_flat_offset_min` | 🟥 dead, hidden but still in schema |
| `sessions[].enable` | tri-state inherit/on/off | `*bool` nil=inherit | `sessionRunnable` `auto_trader_planconfig.go:96-133` | session → base `sessions_enabled` → registry | `sessions_enabled` has **no UI writer** (read for inherit logic only) |
| `sessions[].min_grade` / `min_scenario_quality` | inherit/A/B/C | enum unvalidated | resolvers `strategy.go:1369-1396` | session → base | 🟨 |
| `sessions[].max_trades` | inherit/custom 0–20 | int; `0` meaningful (no entries) | `auto_trader_session.go:78` | session → base | |
| `sessions[].plan_mode` / `replan_cap` / `acceptance_rule` | inherit/override | no enum validation | resolvers | session → base | 🟦 `plan_mode` override dropped at the arm seam |
| `sessions[].last_entry_offset_min` / `eod_flat_offset_min` | **no UI** | nil→15 / nil→0 | `LastEntryOffsetFor`/`EODFlatOffsetFor` → clock gates | session → base | |
| `sessions[].condition_status` | **no UI** | `*map` | `condition_status.go:35-65` | session → base → env → defaults | |

### Surface B — Risk Control (RiskControlEditor)

| field | UI values | schema/validator | engine reader | resolution | issues |
|---|---|---|---|---|---|
| `max_positions` | 1–3 number | clamp `[1,3]` at save AND every cycle (`strategy.go:156-160`, `engine_analysis.go:242`) | `ResolveConcurrentCap` `risk_limits.go:325-332` | strategy > env `RISK_MAX_CONCURRENT_TRADES` | |
| `btc_eth_max_leverage` / `altcoin_max_leverage` | 1–20 range (hidden on futures) | clamp `[1,20]` | prompt text + AI-leverage clamp `engine_position.go:80-88` | — | 🟨 "Trading Leverage" is AI-guidance, not a hard position cap |
| `btc_eth/altcoin_max_position_value_ratio` | **display-only** "System enforced" | clamp `[0.5,10]` | hard notional cap `engine_position.go:119-131` + `auto_trader_risk.go:232-270` | — | 🟨 rendered as a setting that cannot be set |
| `max_margin_usage` | 10–100% (hidden on futures) | clamp `[0.1,1.0]` | **prompt text only** `engine_prompt.go:86` — no gate | — | 🟥 dead as a gate (label says nothing about this) |
| `min_position_size` | 10–1000 USDT (hidden on futures) | clamp `[10,1000]` | `enforceMinPositionSize` `auto_trader_risk.go:271-295` — BUT the crypto kernel path hardcodes **12/60 USDT** `engine_position.go:91-92` | — | 🟧 hardcoded values override the knob on the crypto kernel path |
| `min_risk_reward_ratio` | 1–10 step 0.5, "1:" prefix | unset→3.0, clamp `[1.0,10.0]` | decision path `engine_position.go:151-156`, `entry_gate.go:252-255` | — | 🟦 the ARM seam never consults it — env `ARM_MIN_RR` (2.0) governs armed entries |
| `min_confidence` | 50–100 (default 60) | unset→60, clamp `[50,100]` | `engine_position.go:218-220` | — | 🟨 default template says 75 (`strategy.go` template) while UI/resolver say 60 |
| `guardrails_enabled` | master toggle (ON) | `*bool` nil→ON | `engine_analysis.go:153` | — | OFF bypasses daily loss/profit/trades/blackout/consistency (not futures size caps) |
| `daily_loss_enabled`/`limit_usd` | toggle + USD | none | `firstPositive(rc, env)` — **0 silently becomes $500 env** `engine_analysis.go:158` | strategy > `RISK_MAX_DAILY_LOSS_USD` | 🟨 no log announces the $500 substitution |
| `daily_profit_enabled`/`target_usd`, `max_daily_trades_enabled`/`trades` | toggle + number | none | `engine_analysis.go:160-164` → `risk_limits.go:241-246` | — | |
| `consecutive_loss_halt` | toggle (writes 2/0) + number | none | `auto_trader_orders.go:116,250-259` | 0=OFF; **master-independent** | |
| `reentry_cooldown_minutes` | toggle (writes 20/0) + number | none | `engine_analysis.go:619` | 0=OFF; master-independent | |
| `blackout_enabled`/`start`/`end` | toggle + HH:MM | malformed window → never halts (`cme_calendar.go:149-157`) | `engine_analysis.go:188-194` | — | |
| `consistency_enabled`/`max_day_pct` | toggle + 0–100 | none | `engine_analysis.go:196-202` | pct ≤0 → never breaches | |
| `max_contracts_per_order` | number ≥0 "always active" | none | `ResolveMaxContracts(rc, 2)` `auto_trader_orders.go:57` | venue default 2 | 🟧 clamped DOWN to 1 by env `STAGE_A_CONTRACT_CAP` (default 1, `risk_limits.go:308-320`) — the live cap is 1 regardless of the saved value |
| `max_contracts_enabled` | **no UI** (badge instead) | deprecated parse-only | **READER NONE** (`strategy.go:1736-1738`) | — | 🟥 dead |
| `max_notional_leverage` | number ≥0 "always active" | none | `ResolveNotionalLeverage(rc, 20)` | default 20 | |
| `notional_cap_enabled` | **no UI** | deprecated parse-only | **READER NONE** (`strategy.go:1743-1744`) | — | 🟥 dead |
| `hold_discipline` | toggle OFF | `*bool` nil→OFF | `auto_trader_orders.go:84` | — | |
| `breakeven_enabled`/`trigger_points` | toggle + 1–1000 step 5 | trigger `≤0→50` | `auto_trader.go:152-206` | — | 🟧 the wire is SUSPENDED by 0B (`auto_trader.go:157-159`) — the toggle is on but nothing moves |
| `trailing_enabled`/`atr_mult`/`atr_period`/`arm`/`arm_points` | toggle + ranges + select | arm enum validated at read; mult/period `≤0→default` | `auto_trader_trailing.go:44-63` | — | |

### Surface C — Coin Source / Indicators / Prompt / Publish / Grid

| field | UI values | schema/validator | engine reader | issues |
|---|---|---|---|---|
| `coin_source.source_type` | static/ai500/oi_top/oi_low (futures: static only) | unknown → forced `ai500` (`strategy.go:264-274`) | `kernel/engine.go` candidate scan | 🟧 the switch **rewrites** `use_ai500/use_oi_top/use_oi_low` on every save (`:241-263`) — the toggles are not independent |
| `static_coins` | chips, max 10 | normalized; clamped ≤10 | `GetCandidateCoins` | |
| `excluded_coins` | chips | normalized; **no count clamp** | `filterExcludedCoins` | |
| `use_hyper_main`/`hyper_main_limit` | **no UI** | **no clamp, no default** | `engine.go:514-535` | 🟨 a ≤0 limit passes to the provider |
| `klines.primary_count` | 10–30 | clamp `[10,30]` | market block | |
| `klines.longer_count` | **no UI** | `>30→30`, **no floor** | — | 🟨 0 stays 0 (legacy field, unrendered) |
| `klines.selected_timeframes` | chip toggles (14 TFs) | `>14` truncated | market block | |
| `enable_multi_timeframe` | **no direct UI** | **forced `true`** when timeframes non-empty (`strategy.go:276-278`) | — | 🟧 an explicit `false` is silently discarded |
| `enable_raw_klines` | locked checkbox (always on) | none | kline fetch | 🟨 a toggle you cannot turn off |
| `ema/rsi/atr/boll_periods` | comma lists | **hard reject 1..500 at save** (`ValidateIndicatorPeriods`) | indicator mirror | the only fields hard-rejected at save |
| `enable_oi`, `enable_funding_rate`, NofxOS card, rankings | toggles (futures-hidden) | none | prompt/ranking fetches | 🟨 `nofxos_api_key` default is the deprecated 402 key (`cm_568c…`, CLAUDE.md) |
| `external_data_sources` | **no UI** | none | **READER NONE** | 🟥 dead schema field |
| `prompt_sections.*` / `custom_prompt` | free-text textareas | none | appended to prompts | |
| `publish_config` (`is_public`, `config_visible`) | click-card toggles | none | **overwritten from the DB row** on every read (`api/strategy.go:34-43`) | 🟧 saved in config JSON but never read back from there |
| grid: all 14 fields | FE ranges only | **no backend validation at all** (`api/strategy.go:24-26`); `distribution` enum unvalidated | grid engine files | 🟧 FE ranges are the only guard; grid is parked for futures; a dormant `GridConfigModel` table duplicates all 14 keys (`store/grid.go`) |

### Surface D — Executor / guardrail env knobs shadowing schema knobs

| knob | level | reader | default | overrides schema? |
|---|---|---|---|---|
| `MIN_SL_ATR_MULT` | env | `kernel/min_sl.go:42-48` → three gates | 1.5 | env only — no schema knob |
| `ARM_MIN_RR` | env | `armed_executor.go:59-66` | 2.0 | **yes — replaces `min_risk_reward_ratio` for arms** |
| `HTF_VETO_TF`/`HTF_VETO_MODE` | env | `kernel/htf_veto.go` | 1h | joins the veto; the Studio toggle (`regime.htf_veto`) gates the **decision path only** — the arm chain always vetoes (`armed_executor.go:1257`) 🟦 |
| `regime.transition_standdown` | Studio | `auto_trader_transition.go:34` | ON | decision path only |
| `STAGE_A_CONTRACT_CAP` | env | `risk_limits.go:308-320` | **1** | **clamps `max_contracts_per_order` down to 1** |
| `FAST_MARKET_ATR`, reasoning envs, `AI_PLAN_*`, `BD_*`, `MSS_*`, `ACCEPT_HOLD_MIN`, `STALE_CONFIRM_ATR`, `FLIP_*`, `DORMANT_*`, `EOD_FLAT_*`, `ARM_*` | env | quoted in the reader census | per-knob | shadow/join schema knobs; none editable in Studio |

### Surface E — Weekly panel & session registry (admin, not Studio)

| knob | UI | reader | issues |
|---|---|---|---|
| `WEEKLY_READ_CT` / `WEEKLY_CONFLUENCE_BAND_ATR` / `WEEKLY_SHADOW_MULT` / `PLANNER_CANDLES` | guide cards only (env) | `kernel/weekly_knobs.go` | confluence + mult are **shadow-only** (never change seating) |
| `WEEKLY_COUNTER_MODE` / `WEEKLY_INVALIDATION_TF_DEFAULT` | guide cards retired | **READER NONE** (class-50) | 🟥 dead env knobs still documented |
| `session_registry` (windows/read/flat/enabled/killzones) | **no Studio surface** (admin `system_config`) | `auto_trader_registry.go:24-39`, cached per CME session-day | edits never take effect mid-session |
| `session_registry.sessions[].flat_ct` | — | **READER NONE** (`session_registry.go:50-53`) | 🟥 dead; live flat = window-end − `eod_flat_offset_min` |

### Surface F — Display-only "fields" rendered as settings

| item | where | notes |
|---|---|---|
| "Regime — AUTO" row | DayPlanEditor `:444-452` | not a setting |
| "System enforced" position-value ratios | RiskControlEditor `:513-547` | not editable, but styled as knobs |
| "Futures Risk (est.)" framing | `:1041-1071` | derived display |
| BTC/ETH & Altcoin leverage on futures | hidden | leverage is crypto-only |
| Market chip, killzone/window displays | various | derived |

---

## DEAD-KNOB LIST (saved value cannot take effect) — 15 items

1. `day_plan.last_entry_ct` — unreachable (`auto_trader_clock.go:319-324`)
2. `day_plan.eod_flat_ct` — unreachable (`auto_trader_clock.go:356-363`)
3. `risk_control.max_contracts_enabled` — parse-only (`strategy.go:1736-1738`)
4. `risk_control.notional_cap_enabled` — parse-only (`strategy.go:1743-1744`)
5. `risk_control.max_margin_usage` as a limit — prompt text only (`engine_prompt.go:86`)
6. `session_registry.sessions[].flat_ct` — no consumer (`session_registry.go:50-53`)
7. `WEEKLY_COUNTER_MODE` — no consumer (class-50)
8. `WEEKLY_INVALIDATION_TF_DEFAULT` — no consumer (class-50)
9. `risk_control.min_position_size` on the crypto kernel path — hardcoded 12/60 override (`engine_position.go:91-92`)
10. `sessions[].plan_mode` at the arm seam — `planModeFor("")` drops the override (`armed_executor.go:1222`)
11. `min_risk_reward_ratio` for armed entries — replaced by env `ARM_MIN_RR`
12. `regime.htf_veto=false` for armed entries — arm chain always vetoes
13. `max_contracts_per_order > 1` while `STAGE_A_CONTRACT_CAP` (default 1) is unset
14. `breakeven_enabled` — 0B suspension keeps the wire untouched (`auto_trader.go:157-159`)
15. `indicator_config.external_data_sources` — zero engine consumers

---

## FIX SPEC (no code)

1. **Dead options removed.** Drop from schema + any residual UI: items 1–4, 6, 7, 8, 15 above; stop seeding `nofxos_api_key` with the 402 key; retire the dormant `GridConfigModel` table or mark it display-only in the agent surface.
2. **Duplicate knobs shown once with inherit/override.** The per-session tri-state pattern is right — extend it to the arm seam (pass the real session into `armGateVerdictFor` so `sessions[].plan_mode` actually governs arms), and make `acceptance_rule`'s single-option UI show the RESOLVED value (`5m_close` self-healed) instead of a dead selector.
3. **Resolved value displayed next to every field.** Reuse `StrategyClampWarnings` (already returned at save) and the read-time resolvers: render "saved X → resolved Y (clamped)" beside `proximity_filter_atr`, `max_positions`, R:R, confidence, `max_levels`, `scenario_cap`, `wake_min_interval_min`, and every env-shadowed knob (`max_contracts` shows the effective cap, `min_risk_reward` shows the ARM_MIN_RR note for armed entries).
4. **Every field labeled with what it actually does.**
   - `max_margin_usage` → "prompt guidance only — not enforced".
   - `min_position_size` → "crypto only; kernel path uses fixed 12/60 USDT" (or remove the hardcode — one of the two).
   - `max_contracts_per_order` → "currently capped at 1 by STAGE_A_CONTRACT_CAP".
   - `breakeven` → "suspended (0B) — toggle has no effect".
   - `regime.htf_veto` → "decision path only; armed entries always veto".
   - leverage rows → "AI-guidance, not a hard cap".
   - position-value ratio rows → move out of the form into the derived display block.
   - `min_confidence` default → reconcile template 75 vs resolver/UI 60.
   - stale comments → fix `proximity_filter_atr` (0.1–3.0) and `wake_min_interval_min` (30) schema comments.
5. **Save-time validation parity.** Classic risk fields clamp at save AND cycle; day-plan and guardrails do not — clamp day-plan ranges at save too so the UI can show the clamped result immediately, and add backend ranges to grid (FE ranges are currently the only guard).
6. **Label-vs-behavior reconciliation one-liners** for the five 🟦 rows (arm-seam plan_mode, ARM_MIN_RR, htf_veto arms, value-ratio display, publish_config source-of-truth) — either make the UI say what happens or make the code do what the label says.
