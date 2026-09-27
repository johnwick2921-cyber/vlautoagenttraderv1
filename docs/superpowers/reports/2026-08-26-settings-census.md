# REPORT A — FULL SETTINGS CENSUS (as running NOW, 2026-08-26)

Read-only audit. Fresh evidence only: live DB (`file:data/data.db?mode=ro`),
`.env`, boot log lines, API. Deployed binary rev **57b60b60** (1h-wave + R2/R4).
Branch: `docs/settings-week-audit`. Zero code changes.

---

## 1. BOOT LEDGER (ground truth, verbatim)

```
PID 1991583 /home/hoang/nofx/nofx-bin
🔐 BOOT INTEGRITY OK — rev 57b60b60d652 +dirty · built 2026-08-26T05:17:18Z · expected 57b60b60 · goldens PASS
🧠 AI params in force: model=deepseek-v4-pro max_tokens=32768 temperature=0.50 top_p=omitted timeout=600s retries=2 backoff=2s · truncated-responses=0
🗺️ day-plan knobs: seat_1h_zone=true min_scenario_quality=C ob_lookback_bars=8
```
(`+dirty` = untracked `.env.bak.0825-2157` only; rev + goldens verified.)

## 2. STRATEGY `a5b7662e` — ai_config (strategies.config, updated 2026-08-25 03:14Z)

| Setting | Value | Source | Consumer | Meaning |
|---|---|---|---|---|
| min_confidence | **60** | Studio-DB risk_control | kernel/engine_position.go (min-conf gate) | min AI confidence to open |
| min_risk_reward_ratio | 3 | Studio-DB | engine_position.go F1 R:R gate | required R:R ≥3 |
| guardrails_enabled | **false** | Studio-DB | engine_analysis.go:172 | master OFF (owner's dated choice) |
| daily_loss_limit_usd | 450 · enabled=false | Studio-DB | engine_analysis.go guardrail | would-trip only |
| daily_profit_target_usd | 900 · enabled=false | Studio-DB | same | would-trip only |
| max_daily_trades | 3 · enabled=false | Studio-DB | same | would-trip only |
| max_contracts_per_order | 2 · enabled=false | Studio-DB | same | would-trip only |
| notional_cap_enabled | false | Studio-DB | same | would-trip only |
| blackout_enabled | false | Studio-DB | same | would-trip only |
| max_positions | 3 | Studio-DB | trader loop | concurrent positions cap |
| max_margin_usage | 0.9 | Studio-DB | sizing | margin ceiling |
| btc_eth_max_leverage / altcoin | 5 / 5 (risk_control) | Studio-DB | engine sizing | — |
| trader-row leverage | **10 / 5** | traders table | engine ctx | trader-row wins for this trader |
| breakeven_enabled | true · trigger **40 pts** | Studio-DB | trailing/BE logic | BE at +40 pts |
| trailing_enabled | true | Studio-DB | chandelier trail | trail active |
| hold_discipline | true | Studio-DB | discipline | min-hold honored |
| position_mode | `ai_watch` | traders row | watcher | AI watch mode |
| cadence_mode | "" (default) | traders row | loop cadence | default cycle |
| scan_interval_minutes | 2 | traders row | loop | 2-min cycles |
| account | Sim101 | traders row | execution | SIM |
| regime block | **ABSENT** → all shipped defaults | default | kernel regime | htf_veto ON · transition ON · flip-hold env 30min |
| min_scenario_quality | **ABSENT → "C"** | default (R4 knob) | engine_position.go R4 gate | no restriction; knob available, unused |
| indicators | EMA50/200·RSI14·ATR14 ON · MACD/BOLL OFF · klines 5m/4h + MTF[1h,4h,1d,15m,3m,5m] · volume ON · funding/SVP ON | Studio-DB | prompt mirror | — |

## 3. DAY_PLAN block (same strategy config)

| Setting | Value | Default | Flag? |
|---|---|---|---|
| plan_enabled | true | false | enabled |
| plan_mode | advisory | advisory | — |
| planner_timeframes | D,4h,1h,15m | same | — |
| proximity_filter_atr | **2** | 1.5 | widened band (owner) |
| max_levels | **12** | 8 | raised (owner) |
| scenario_cap | **5** | 3 | raised (owner) |
| acceptance_rule | 2x5m | 2x5m | — |
| replan_cap | **4** | 2 | raised (owner) |
| realign_cap | **10** | 5 | raised |
| last_entry_ct | 13:00 | 13:00 | session-relative now |
| eod_flat_ct | 14:45 | 14:45 | R5 ruling: stays |
| seat_1h_zone | **absent → ON** | ON | 1h-wave knob at default |
| wake knobs (W6) | **absent → all ON except HTF OB (OFF)** | ON | defaults |
| wake_min_interval_min | **absent → 30** | 30 | — |
| min_scenario_quality | absent → C | C | — |
| Per-session | NY: enable(absent=via registry) replan 4 advisory 2x5m min_grade **B** max_trades **10** · ASIA: **enable:true** replan 4 advisory 2x5m min_grade B max_trades **7** · LONDON: **enable:true** replan 4 advisory 2x5m min_grade B max_trades **10** | — | ASIA/LONDON max_trades deviate from research (A/1·A) — owner accepted |

Note: `sessions_enabled: ["NY"]` is stale-looking — the per-session `enable:true`
rows are what actually arm ASIA/LONDON (override wins). See §8.

## 4. .ENV CENSUS (secrets masked)

| Key | State | Value |
|---|---|---|
| NOFX_BACKEND_PORT / FRONTEND_PORT | set | 8080 / 3000 |
| NOFX_TIMEZONE | set | UTC |
| JWT_SECRET | set | <44 chars, masked> |
| DATA_ENCRYPTION_KEY | set | <44 chars, masked> |
| RSA_PRIVATE_KEY | set | <1730 chars, masked> |
| TRANSPORT_ENCRYPTION | set | false |
| DB_TYPE / DB_PATH | set | sqlite / data/data.db |
| TRADING_MODE | set | futures |
| DATABENTO_API_KEY / DATASET | set | <32 chars> / GLBX.MDP3 (legacy backfill only) |
| NINJATRADER_DATA_DIR | set | <34 chars> (legacy CSV path) |
| CLAW402_* | set | flash model + wallet (masked) |
| AI_MAX_TOKENS | set | 32768 |
| NT_EXTRA_SYMBOLS / NT_RUNTIME_SYMBOLS / NT_TRANSPORT | set | ES / true / tcp |
| EOD_FLAT_LIMIT_TICKS | set | **2** |
| EOD_FLAT_MARKET_AFTER_SEC | set | **10** |
| AI_HTTP_TIMEOUT_SECONDS | set | **600** (was unset → 300s timeouts killed ASIA twice) |
| AI_MAX_RETRIES | set | **2** (was unset → 3) |
| **UNSET** (→ defaults) | — | CONFLUENCE_CAP(3) · OB_LOOKBACK_BARS(8) · TRANSITION_MAX_MIN(45) · FLIP_MIN_HOLD_MIN(30) · FLIP_EVAL_MAX_STALE_S(90) · HTF_VETO_TF(1h) · STALE_* · POST_EXIT_* · CONFIRM_GRACE_SESSIONS(3) · AI_RETRY_BACKOFF_SECONDS(2) |

## 5. GUARDRAILS + WOULD-TRIP LEDGER (week, logs)

Master = **OFF** (guardrails_enabled=false). Would-trip fired per day:
08-19 **301** · 08-20 **756** · 08-21 76 · 08-22 0 · 08-23 0 · 08-24 **202** ·
08-25 **271** · 08-26 0 → **1,606 total**; breakdown: `max daily trades` **1039**,
`daily loss` **266** (others 0). Every one is advisory-only (never enforced).

## 6. SESSION REGISTRY (kernel/session_registry.go, fresh read)

- ASIA 17:00→02:00 CT (wraps midnight), Flat 02:00; LONDON 02:00→08:30; NY
  08:30→**14:45 CT** (owner contract 2026-08-16: NY ends 15:45 ET = 14:45 CT).
- Per-session last-entry / EOD-flat offsets default **15 min** before session end.

## 7. GRADING CONSTANTS (deployed rev, post-1h-wave — kernel/levels_score.go)

- Evidence `zoneEvidenceByKind` (kind × TF): OB .40/.50/**.70**/.72 ·
  FVG & iFVG & S/D .35/.45/**.65**/.65 (1m/15m/1h/4h).
- Non-zone `typeEvidence`: PDH/PDL/PDC/RTH/… 1.0 · ONH/ONL/nPOC .85 ·
  ASH/ASL/OR/IB/EQ .70 · Round/Gap .55 · zones .30.
- `zoneTFMult` 1.0/1.1/1.2/1.3 · reversal ×1.1 (documented effective 4h:1m ≈2.3× — R3).
- Floors/caps: 1m=C · 15m=B..B · 1h=**B..A** (wave) · 4h=B..A.
- Freshness ladder 1.0/0.8/0.6/0.5 (consumed role-flip 0.5).
- `gradeFromScore`: A≥1.0, B≥0.70. Confluence cap env `CONFLUENCE_CAP`=3.
- Cluster collapse 12 ticks (3.00 pts) · `DefaultMaxLevels`=8 ·
  `MinSideLevels`=3/side · `maxHTFSeats`=2 (one reservable for 1h S/D via
  `Seat1HZone`) · FVG gap floor max(2×tick, **2.0 pts**) · OB lookback
  `OB_LOOKBACK_BARS`=**8**.

## 8. FLAGS — stored ≠ likely-believed

| Setting | Stored | Owner likely believes | Verdict |
|---|---|---|---|
| min_confidence | 60 | 65 per contract | research 65; recent days favor 65 (Phase 6) |
| guardrails_enabled | false | ON? | 1,606 would-trips this week went un-enforced |
| sessions_enabled | ["NY"] | NY+ASIA+LONDON | cosmetic — per-session enable:true wins; stale field confuses |
| ASIA max_trades | 7 | research 1·A | owner accepted deviation |
| position_mode | ai_watch | — | watcher mode on |
| trader-row leverage 10 vs risk_control 5 | 10/5 | 5/5? | trader row wins |
| realign_cap | 10 | 5 default | raised |
| regime block absent | defaults | — | htf_veto ON by default; no UI row rendered for it |
| min_scenario_quality | absent → C | knob exists now | set A/B to activate the R4 gate |
| wake knobs absent | all ON (HTF OB OFF) | — | invisible-but-armed; interval 30min |
| EOD_FLAT_LIMIT_TICKS=2 | env | — | limit-then-market active |
| AI timeout 600 | env (new) | — | was 300 → two fail-closed incidents |

## 9. SETTINGS-PAGE MAP (what the Studio shows vs hides)

- **Risk Control page**: min_confidence ✓ · min_risk_reward ✓ · guardrails master
  ✓ (OFF shown) · daily loss/profit/trades/contracts/notional/blackout ✓ ·
  breakeven trigger ✓ · trailing ✓. ENV-ONLY invisible here: EOD_FLAT_LIMIT_TICKS,
  EOD_FLAT_MARKET_AFTER_SEC, AI timeouts/retries, TRANSITION_MAX_MIN,
  FLIP_MIN_HOLD_MIN, FLIP_EVAL_MAX_STALE_S, HTF_VETO_TF, CONFLUENCE_CAP,
  OB_LOOKBACK_BARS, CONFIRM_GRACE_SESSIONS.
- **Day Plan page (DayPlanEditor)**: plan_enabled, plan_mode, timeframes,
  proximity, max_levels, scenario_cap, acceptance, replan_cap, realign_cap,
  last-entry, EOD-flat, **W6 wakes (5 toggles + interval)**,
  **seat_1h_zone toggle**, **min_scenario_quality A/B/C** — all persist via the
  strategy PUT. Per-session accordion: enable, replan_cap, plan_mode,
  acceptance, **min_grade**, **min_scenario_quality**, max_trades, offsets ✓.
- **Regime**: NO regime UI rows rendered for this strategy (block absent) —
  htf_veto/transition defaults are invisible; the only surface is the gate-block
  counter endpoint. Env `HTF_VETO_TF` invisible.
- **Persistence gaps found**: none this audit for the Day-Plan page (both PUT
  paths tested in the R4 wave). Risk-control leverage mismatch (trader-row 10 vs
  config 5) is the one value that renders differently between pages.
