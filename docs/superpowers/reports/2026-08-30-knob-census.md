# Knob & Constant Provenance Census — 2026-08-30

Every tunable number in the money path, labeled by where it came from.
Read-only dispatch (isolated worktree `nofx-census` @ origin/dev `a9aa9a04`).
Live bot rev: `451926d9ff85` (pre-sunday cutover). NO code changes, NO DB writes.

## Label legend

| Label | Meaning |
|-------|---------|
| **[R]** | Researched — cited in code comment or a docs/research/plan-card artifact |
| **[D]** | Dispatch/backtest-validated — has a sweep with n and result |
| **[O]** | Owner-set — explicit owner choice stored in DB / env / ruling |
| **[C]** | Code-canon — fixed constant or shipped default, no external citation found |
| **[I]** | Inferred — a number whose origin is not documented anywhere |

`/` = multiple labels apply (e.g. [O/C] owner value overriding a code default).

---

## 1. ENV KNOBS (resolver defaults)

Money-path first; infra grouped at the end.

| Env | Default | Where | Label | Notes |
|-----|---------|-------|-------|-------|
| `MIN_SL_ATR_MULT` | **1.0** ×ATR5m | kernel/min_sl.go:23 | [C] | + `MIN_SL_LEVEL_CLEARANCE` 2 ticks (MinSLTickClearance) |
| `STALE_CONFIRM_ATR` | **2.0** ×ATR5m | kernel/plan_confirm.go / levels_volume_boot.go:19 | [C] | stale-confirm stale threshold |
| `STALE_REEVAL_DRIFT_ATR` | **0.25** ×ATR14 | trader/discard_burn.go:38 | [C] | comment: "spec-fixed default" |
| `STALE_BAR_GRACE_S` | **15** | kernel/stale_data.go:46 | [C] | B4 gate grace |
| `FAST_MARKET_ATR` | **1.5** ×ATR5m | trader/auto_trader_loop.go:86 | [I] | fast-market wake trigger |
| `FAST_MARKET_REASONING` | **fast** | trader/auto_trader_loop.go | [O] | owner live env |
| `AI_PLAN_REASONING` | **max** | trader/auto_trader_loop.go | [O] | owner live env |
| `AI_PLAN_MAX_TOKENS` | **65536** | trader/auto_trader_planner.go | [C] | 2× the 32768 truncation ceiling |
| `ARM_MIN_RR` | **2.0** | trader/armed_executor.go:33 | [D] | autopsy wave: n=18 +$994 |
| `ARM_PLACE_TICKS` | **100** | trader/armed_executor.go:23 | [C] | placement band |
| `ARM_WORKING_STALE_MIN` | **15** | trader/armed_executor.go:49 | [C] | reconcile net |
| `ARMED_CANCEL_ACK_TIMEOUT_MS` | **2000** | trader/armed_executor.go:647 | [C] | |
| `ARMED_ORDER_UPDATE_LOG_SAMPLE` | 500 | trader/armed_executor.go | [C] | DEBUG sampling |
| `BD_MIN_DISP_ATR` | **1.0** ×ATR5m | kernel/breakdown_continue.go | [D/C] | waterfall wave |
| `BD_MAX_PULLBACK` | **0.4** | kernel/breakdown_continue.go | [C] | |
| `BD_CONFIRM_CLOSES` | **2** | kernel/breakdown_continue.go | [C] | |
| `BD_MAX_LEVEL_DIST_ATR` | **5.0** | kernel/breakdown_continue.go | [C] | |
| `BD_MIN_SL_ATR` | **1.0** | kernel/breakdown_continue.go | [C] | |
| `FLIP_ATR_BUFFER` | **0.5** | kernel/plan_lifecycle.go:325 | [C] | flip invalidation buffer |
| `FLIP_CONFIRM_CLOSES` | **2** | kernel/plan_lifecycle.go:336 | [C] | |
| `DORMANT_MIN_HOLD_MIN` | **5** | kernel/plan_lifecycle.go:349 | [C] | |
| `TRANSITION_MAX_MIN` | **45** | kernel/transition.go:22 | [C] | |
| `WEEKLY_SHADOW_MULT` | **1.5** | kernel/weekly_knobs.go:93 | [C] | |
| `WEEKLY_CONFLUENCE_BAND_ATR` | **0.25** | kernel/weekly_knobs.go:82 | [C] | |
| `WEEKLY_COUNTER_MODE` | **warn** | kernel/weekly_knobs.go | [C] | |
| `HTF_VETO_MODE` | **1h** (live: `cross`) | kernel/htf_veto.go | [O] | owner live env = cross |
| `TOUCH_BAND_TICKS` | **16** | kernel/touch_telemetry.go:35 | [C] | |
| `TOUCH_EPISODE_MAX_BARS` | **12** | kernel/touch_telemetry.go:46 | [C] | |
| `TOUCH_APPROACH_BARS` | **5** | kernel/touch_telemetry.go:66 | [C] | |
| `TOUCH_VOL_LOOKBACK` | **20** | kernel/touch_telemetry.go:49 | [C] | |
| `FVG_CE_WIDTH_PTS` | **20** | kernel/fvg_entry.go:46 | [R] | cited: NQ gap sweet spot 20–80 pts |
| `POST_EXIT_DELAY_MS` | **2000** | trader/auto_trader_postexit.go:28 | [C] | |
| `ROLL_BLOCK_DAYS_BEFORE_EXPIRY` | **3** | trader/contract_roll.go:53 | [C] | |
| `PERSIST_STALL_WATCHDOG_S` | **60** | provider/ninjatrader/bar_persist.go:37 | [C] | |
| `INGEST_QUEUE_CAP` | **1024** | provider/ninjatrader/tcp_server.go | [C] | |
| `BAR_UPDATE_LOG_SAMPLE` | **500** | provider/ninjatrader/tcp_server.go | [C] | |
| `EOD_FLAT_LIMIT_TICKS` / `EOD_FLAT_MARKET_AFTER_SEC` | **0 / 0** | config/config.go:178-179 | [C] | off by default |
| `AI_MAX_RETRIES`, `AI_HTTP_TIMEOUT_SECONDS`, `AI_TEMPERATURE`, `AI_TOP_P`, `AI_TIMEOUT_SECONDS`, `AI_MAX_TOKENS`, `AI_REASONING_EFFORT`, `AI_REPLANNER_MAX_TOKENS`, `AI_TASKSTATE_*`, `AI_STREAM_IDLE_TIMEOUT_SECS`, `AI_RETRY_BACKOFF_SECONDS`, `AI_BALANCE_WARN` | provider defaults | provider/* | [C] | infra |
| `STRUCTURE_SWING_K` **2** / `STRUCTURE_MIN_SWING_ATR` **0.25** / `STRUCTURE_MSS_BODY_ATR` **1.5** | swing/MSS | kernel/levels_swing.go | [C] | |
| `OB_LOOKBACK_BARS` | 8 | kernel | [C] | |

## 2. DB-STORED (live strategy `a5b7662e`, 2026-08-30)

risk_control now lives under `ai_config.risk_control` (moved from strategy top-level).

| Key | Value | Label |
|-----|-------|-------|
| min_confidence | **60** | [O] |
| min_risk_reward_ratio | **3.0** | [O] |
| max_positions | **3** | [O] |
| btc_eth_max_leverage / altcoin_max_leverage | 5 / 5 | [O] |
| btc_eth_max_position_value_ratio / altcoin | 5 / 1 | [O] |
| max_margin_usage | 0.9 | [O] |
| min_position_size | 12 | [O] |
| daily_loss_limit_usd | **450** | [O] |
| daily_profit_target_usd | 900 | [O] |
| max_daily_trades | 3 | [O] |
| max_contracts_per_order | 2 | [O] |
| breakeven_trigger_points | **+40** | [O] |
| guardrails_enabled | false | [O] |

Day plan (`ai_config.day_plan`):

| Key | Value | Label |
|-----|-------|-------|
| plan_mode | strict | [O] |
| proximity_filter_atr | **0.3** | [O] |
| max_levels | **12** (hard ceiling 12, PlanHardMaxLevels) | [O/C] |
| scenario_cap | **5** (hard ceiling 5) | [O/C] |
| min_side_levels | **4** (was 2) | [O] |
| acceptance_rule | 2x5m | [O] |
| replan_cap | 4 (global + per-session) | [O] |
| realign_cap | 10 | [O] |
| last_entry_ct / eod_flat_ct | 13:00 / 14:45 CT | [O] |
| wake_on_htf_ob | true | [O] |
| sessions: NY / ASIA / LONDON | max_trades 10 / 7 / 10, min_grade B, acceptance 2x5m | [O] |

Safe defaults in code when unset (store/strategy.go): `SafeDefaultMinRiskReward` **3.0** [R] (cited VL-DAYPLAN-FULL-SPEC.md), `SafeDefaultMinConfidence` **60** [R/O] (owner ruling 2026-08-19: aligned 65→60, PR #54 finding). Range floors: MinRiskReward 1.0, MinConfidence 50.

## 3. KERNEL / TRADER CONSTANTS (magic numbers in code)

### Level scoring — the big unvalidated block

| Knob | Value | Where | Label |
|------|-------|-------|-------|
| zoneEvidenceByKind base | OB {1m .40, 15m .50, 1h .70, 4h .72}; FVG/IFVG/Supply/Demand {.35/.45/.65/.65} | kernel/levels_score.go:149 | [O/C] — v3 grading "owner-approved 2026-08-24, research-grounded"; 1h tier raised 2026-08-25 owner R1; numeric values themselves no sweep |
| zoneTFMult | 1m 1.0 / 15m 1.1 / 1h 1.2 / 4h 1.3 | levels_score.go:157 | [C] — R3 note: effective 4h:1m spread ≈2.3×, documented-not-changed |
| zoneReversalBonus | **1.1** | levels_score.go | [I] |
| zoneSizeMult ladder | ≤.30→1.25, ≤.60→1.10, ≤1.0→1.0, ≤1.5→0.85, ≤2.5→0.70, else 0.50 | levels_score.go:205 | [I] — never validated |
| freshness ladder | 1.0 / 0.6 / 0.3 / 0.15 | levels_score.go:437 | [I] |
| anchor decay ladder | 1.0 / 0.8 / 0.6 / 0.5 | levels_swing.go:22 | [I] |
| non-zone kind weights | SWG/SWL .85, EVWAP/PDVWAP .85, VAH/VAL/SETT .80, MID-O .60, ASH/ASL/LDN/OR/IB/EQ .70, Round/Gap .55, zone-only .30 | levels_score.go:100-122 | [I] |
| proximity default (ActivationWindowK) | **1.5** ×dATR | levels_score.go:412 | [C] — owner overrides with 0.3 |
| LevelClusterTicks | **12** = 3.0 pts | levels_score.go:678 | [I] |
| Tier1ProximityTicks | 12 | levels_score.go:257 | [C] |
| DefaultMaxLevels / PlanHardMaxLevels | 8 / **12** | plan_doc.go:248,254 | [C] |
| PlanMaxScenarios / PlanHardMaxScenarios | 3 / **5** | plan_doc.go:249,255 | [C] |
| DefaultSideQuota | **2** | plan_doc.go:657 | [C] (old rule was 3) |
| StaleConfirm ATR | **2.0** ×ATR5m | levels_volume_boot.go:19 | [C] |

### Structural / entry

| Knob | Value | Where | Label |
|------|-------|-------|-------|
| Swing k / min-swing / MSS body | 2 / 0.25×ATR / 1.5×ATR | kernel/levels_swing.go | [C] |
| FVG displacement floor | ≥ max(2×tick, noise floor 2.0 pts); CE width 20 pt; sweet spot 20–80 pt | kernel/fvg_entry.go:25-46 | [R] — cited gap research |
| MinSL | 1.0×ATR5m + 2-tick level clearance | kernel/min_sl.go | [C] |
| Breakdown/continue | BD_* defaults above | kernel/breakdown_continue.go | [D/C] |
| ARM min RR / place band / stale | 2.0 / 100 ticks / 15 min | trader/armed_executor.go | [D]/[C] |

### Sessions & timing

| Knob | Value | Where | Label |
|------|-------|-------|-------|
| Session windows (CT) | ASIA 17:00→02:00, LONDON 02:00→08:30, NY 08:30→14:45 | kernel/session_registry.go | [O] — owner contract: session end == EOD flat |
| Killzones | asia 19:00→23:00, london 02:00→05:00, ny_am 08:30→11:00, ny_pm 13:00→14:45 | session_registry.go | [O] |
| EOD flat | 14:45 CT | session_registry.go (owner contract) | [O] |
| NY AM premium window | 08:30–11:00 ET, premium 10:00–11:00 | kernel/planner_prompt.go:536 | [O] advisory |
| no_trade fixed | first 5m, 12:00–13:30 lunch | planner_prompt.go | [O] |
| Dead-man watchdog | 60 s | trader/auto_trader_loop.go | [C] |
| Planner retries | ≤2 (3 attempts) | auto_trader_planner.go:1146 | [C] |
| Post-exit delay | 2000 ms | auto_trader_postexit.go | [C] |

### C# AddOn (ninjascript/*.cs)

| Const | Value | Label |
|-------|-------|-------|
| GO_SERVER_PORT | 36974 (NOT ATI 36973) | [C] |
| HEARTBEAT_INTERVAL_MS | 30000 | [C] spec L4408 |
| RECONNECT_INTERVAL_MS | 5000 | [C] spec L4415 |
| STALE_SIGNAL_AGE_SECONDS | 60 | [C] spec L4414 |
| PROTOCOL_VERSION | 3 | [C] |
| MAX_FRAME_BYTES | 1 MB | [C] spec L4376 |
| DEFAULT_BARS_BACK | 2000 | [C] |
| FAST_MAX_ATTEMPTS | 3 | [C] |
| RollDaysBeforeExpiry | 8 | [C] |
| RecreateDebounce | 30 s | [C] |

## 4. NOT ON DEV — branch-pending (feat/entry-mechanics)

The 5 expected new knobs are ABSENT on origin/dev (0 files each): `BD_MIN_CLOSES 1`,
`MSS_MIN_DISP 0.5`, `ACCEPT_HOLD 10`, `STOP_OFFSET 2`, `RETEST_WAIT 6`.
They live only on the entry-mechanics branch — out of scope for this census;
will need labels when merged.

## 5. TOP-10 unvalidated by blast × arbitrariness

[I]/[C]-labeled knobs with the widest blast radius and no cited sweep, ranked,
with the cheapest validation test each.

| # | Knob | Value | Label | Cheapest test |
|---|------|-------|-------|---------------|
| 1 | zoneSizeMult ladder | 1.25/1.10/1.0/0.85/0.70/0.50 | [I] | Replay last 2 wks of level hits; regress size-mult against touch/react/fill rate (needs new harness) |
| 2 | zoneEvidenceByKind base table | OB .40/.50/.70/.72 … | [O/C] | Sensitivity: perturb ±0.05 per tier, measure grade-distribution shift on cached plans (after E8) |
| 3 | zoneTFMult spread | 1.0/1.1/1.2/1.3 (effective ≈2.3×) | [C] | Same as #2 — R3 note already flags revisit "after the 1h wave has live data" (owner queue) |
| 4 | freshness + anchor decay ladders | 1.0/.6/.3/.15 and 1.0/.8/.6/.5 | [I] | Replay: correlate level age at touch with fill/stop-out (needs new harness) |
| 5 | FAST_MARKET_ATR | 1.5 | [I] | Count wake reads triggered vs. actual fast moves in BarCache history (offline, yes-now) |
| 6 | proximity band | owner 0.3 (code default 1.5) | [O/C] | Journal: rate of level-removed-by-proximity vs. later-touched (offline, yes-now) |
| 7 | cluster tolerance | 3.0 pts | [I] | Replay: merged-pair distance histogram vs. tick-value noise (offline, yes-now) |
| 8 | side-quota default | 2 (old rule 3) | [C] | Compare plan side-mix before/after the 08-26 change in stored plans (offline, yes-now) |
| 9 | killzone + premium-window weighting | advisory text | [O] | Count S1 fills inside vs. outside NY premium window (offline, yes-now) |
| 10 | BD_* waterfall defaults | 1.0/0.4/2/5.0/1.0 | [D/C] | n is thin — extend the breakdown-continue sweep over more sessions (after E8) |

Validation keys: **yes-now** = offline over stored bars/plans/decisions, no new harness.
**after-E8** = needs the post-Sunday live window (2026-08-30 17:00 CT) or small harness work.
**needs-new-harness** = replay infrastructure not yet built.

## 6. Provenance summary

- **[R] researched:** FVG floor/CE-width/sweet-spot (in-code citation), safe R:R 3.0 (VL-DAYPLAN-FULL-SPEC.md), min_conf 60 default (owner ruling, PR #54).
- **[D] validated by sweep:** ARM_MIN_RR 2.0 (n=18, +$994), BD_* waterfall family (waterfall-class wave), zone grading v3 (owner-approved, research-grounded).
- **[O] owner decisions:** min_conf 60, R:R 3.0, proximity 0.3, max_levels 12, scenario_cap 5, min_side_levels 4, session windows + EOD 14:45, killzones, veto mode cross, BE +40, daily loss 450, max 3 trades.
- **[C] code-canon with no external citation:** the whole level-scoring ladder (zoneSizeMult, TFmult, freshness/anchor decay, kind weights), cluster 3 pt, side-quota 2, stale/confirm ATRs, watchdog 60 s, planner retries, C# wire constants (spec-cited).
- **[I] inferred (no documented origin anywhere):** zoneSizeMult ladder, decay ladders, FAST_MARKET_ATR, cluster tolerance.

Bottom line: **risk gates are owner-researched; the level-scoring grader is the
least-provenanced block in the machine** — a dozen multiplicative weights whose
individual values have never been swept. That is where the next validation dollar
buys the most information.
