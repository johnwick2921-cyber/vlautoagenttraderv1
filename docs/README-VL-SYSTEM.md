# NOFX / VL SYSTEM — OPERATOR'S MANUAL + FULL UI REFERENCE

- **Branch:** `docs/system-readme` (merged to dev) · **Docs-only** (zero code changes) · refreshed to the running rev **`717acd34`** (2026-08-27 S-dispatch; boot `BOOT INTEGRITY OK — rev 717acd34e52b +dirty · goldens PASS`; `+dirty` = only the untracked `.env.bak.0825-2157`).
- **Verify-first law:** every row below was written after reading the FE component, the API handler, and the engine path behind it. Cites are `file:line` (or `[report]`). Where code disagrees with this or older docs, CODE wins; anything unverifiable is marked `[UNVERIFIED]`.
- **Audience:** the owner (mobile — short sentences, tables), then any fresh agent.
- **Discrepancies found while writing:** collected in §9 — fixed in the 2026-08-27 planner-contract wave (C1–C9) + P0-relax wave; statuses updated below.

---

# PART A — THE SYSTEM

## 1. WHAT THIS IS

A structure-permissioned, level-anchored **MNQ SIM trader**. It never touches a live
account — `isAccountTradeable` hard-blocks non-SIM (`provider/ninjatrader` account
routing; UI AccountSelector shows SIM-only rows). The architecture is
**advisory-first**: the Go engine computes the map, the AI (DeepSeek `deepseek-v4-pro`)
writes ONE day-plan per session and then decides each cycle against it; the owner
holds the knobs (sessions, plan_mode, budgets, guardrails) and every destructive
button asks first. All orders go to **NinjaTrader Sim101** via the C# AddOn.

```
NT8 (Tradovate MNQ bars, Sim101)   ← SIM only, never live
        │ TCP (bars/account/positions/orders)
        ▼
C# AddOn (NT8 AddOns folder)       ← must be compiled + NT8 restarted to take effect
        │ framed messages (ninjascript/vltrader_tcp_PROTOCOL.md)
        ▼
Go bot (nofx-bin) ──── BarCache + bars table (SQLite data/data.db)
        │
        ├─ PLANNER: DeepSeek → 1 day-plan per session (advisory JSON)
        ├─ EXECUTOR: every ~2 min → DeepSeek decision → risk gates → Sim101
        ├─ WATCHER: in-position advisory only (zero order authority)
        └─ DB = memory: plans, positions, decisions, bars, level_state,
           touch_episodes, level_stats, gate-block counters
```

- Identity line: "one structure-permissioned level-anchored SIM trader, advisory-first,
  owner holds the knobs." SIM lock evidence: `trader/ninjatrader` `isAccountTradeable`
  refuses non-SIM accounts; UI shows SIM-only rows (`web/src/components/trader/AccountSelector.tsx:203-224`).
- Boot line: `🔐 BOOT INTEGRITY OK — rev … · goldens PASS` — refuses trading if the
  binary revision or the prompt goldens drift (`kernel/boot_integrity.go:122-159`).

## 2. THE PIPELINE

### 2.1 WAKE — who starts a cycle or a plan read

| Trigger | Who fires it | Cite |
|---|---|---|
| Session scheduled read | registry read times: ASIA 16:55 · LONDON 01:55 · NY 08:25 CT | `kernel/session_registry.go:88-110` |
| Level-event wake | seated-level invalidation / new 15m·HTF zone / HTF OB / iFVG inversion | `trader/auto_trader_wake_levels.go:279` (`trigger_reason="level_event"`) |
| MSS wake | bias-TF MSS structure event | `trader/auto_trader_planner.go:286-288` |
| Owner reset / re-read | Plan-card buttons | `trader/auto_trader_reset.go:129` (`owner_reset`), reread handler (`owner_reread`) |
| Fail-closed marker | planner read failed → `planner_fail_closed` or `replans_exhausted` no_trade version | `trader/auto_trader_planner.go:617,1019` |
| Executor cycle | scan interval (default 2 min for `hoang`; config `scan_interval_minutes`) | `trader/auto_trader_loop.go:95` |
| Post-exit kick | immediate rescan after a close (`cycle_trigger=post_exit`) | `trader/auto_trader_loop.go:264-267` |
| stale_dodge | defer a cycle across a TF close boundary when the call would straddle it | `trader/discard_burn.go:99-112` |

Market closed → the whole cycle skips (`cmeSessionClosedSkip`, 3-min backoff,
`trader/auto_trader_loop.go:130,856-863`).

### 2.2 SENSE — the 10 detector groups

All detectors run per planner read + per executor cycle on the 1m snapshot
(~2000 bars, `kernel.AISVPBarCount`), plus HTF re-runs on 1h/4h
(`kernel/levels_assemble.go:71-128`, `DetectHTFLevels:134+`).

| # | Group | Kinds | Cite |
|---|---|---|---|
| 1 | Multi-day structural | PDH/PDL/PDC, RTH-H/L, AS-H/L, LDN-H/L, ONH/ONL, PWH/PWL, PMH/PML | `kernel/levels_multiday.go:42` |
| 2 | Round numbers | RN (100/50/25) | `kernel/levels_intraday.go:21` |
| 3 | Gaps | GAP (unfilled, ≥1×ATR) | `kernel/levels_intraday.go:57` |
| 4 | OR / IB | OR-H/L (5m), IB-H/L (60m) + ±1.5×/±2× extensions | `kernel/levels_intraday.go:119` |
| 5 | Liquidity | EQH/EQL (3-tick tolerance) | `kernel/levels_zones.go:34` |
| 6 | Supply/demand | SUPPLY/DEMAND (base ≤6 candles, departure ≥1.5×ATR) | `kernel/levels_zones.go:103` |
| 7 | Imbalances | FVG + iFVG (gap floor 2pt/8t; session-break guard A6) | `kernel/levels_zones.go:190` |
| 8 | Order blocks | OB (last opposite candle, 8-bar lookback) | `kernel/levels_zones.go:269` |
| 9 | Volume family | VWAP±1σ, eVWAP (15:00 CT anchor), pdPOC/VAH/VAL, nPOC, pdVWAP, SETT, MID-O | `kernel/levels_volume.go:377` |
| 10 | HTF re-run | EQH/SD/FVG/OB on 1h+4h, `HTF=true` | `kernel/levels_assemble.go:134` |

Bars live in the NT8 BarCache (in-memory) and persist to the `bars` table
(`store/bar_history.go`, 90-day retention).

### 2.3 SCORE & SEAT — verified order

1. **Proximity lock FIRST**: keep levels within `proximity_filter_atr × DailyRangeProxy`
   of price (retuned 0.3 → ±~92pt on 2026-08-26; clamp 0.1–3.0
   `trader/auto_trader_planconfig.go:145-150`). `DailyRangeProxy` = mean completed
   CME session-day H−L range (`kernel/levels_assemble.go:209+`).
2. **Score** (`kernel/levels_score.go:408-497`):
   - zones: `zoneEvidence(kind,TF,reversal×1.1) × zoneSizeMult(0.5–1.25) × freshness × (1+0.20×conf) × zoneTFMult(1.0/1.1/1.2/1.3)`
   - lines: `typeEvidence(kind) × freshness × (1+0.20×conf) × htf(×1.2)`
3. **Freshness ladders**: zones `1.0/0.6/0.3/0.15`; anchors/lines `1.0/0.8/0.6/0.5`
   (`:342-375`).
4. **Confluence**: distinct families only, cap 3 (`CONFLUENCE_CAP`) → ×1.6 max
   (`:429-464`); families: vwap · profile · prior · anchor · overnight · liquidity ·
   zone · round · gap · other (`:306-338`).
5. **Tier floors/caps** (`:466-496`): 1m zones forced C · 15m forced B (both ways) ·
   1h/4h floor B (may reach A) · B2 gate: above-C only within 12 ticks (3.00pt) of a
   Tier-1 anchor. Tier-1 = PDH/PDL/PDC/RTH/OR/ON + PWH/PWL/PMH/PML + **VAH/VAL/SETT/nPOC** (R-A13, `:254-262`).
6. **Grade**: A ≥ 1.0 · B ≥ 0.70 · else C (`:627-637`).
7. **Kind→role map** (`kernel/levels_role.go:33-45` + `RoleFor` overrides `:108-127`):

| Role | Kinds |
|---|---|
| magnet_meanrevert | VWAP, eVWAP, pdVWAP, POC, nPOC, GAP |
| liquidity_break | ONH/ONL, EQH/EQL, IB/OR H-L, AS-H/L, LDN-H/L |
| react_zone | PDH/PDL, RTH-H/L, VAH/VAL, PWH/PWL/PMH/PML, Supply, Demand, FVG, iFVG, OB |
| pivot | PDC, SETT, MID-O, RN |
| target_only | override only: consumed/done freshness, or HTF continuation zones |

8. **Seating**: collapse (≤3.00pt clusters) → priority sort → `seatHTF` (≤2, Tier-1-or-reversal)

#### Level-truth wave (2026-08-27) — deployed values ARE the documented truth

- **SWG-H / SWG-L** (swing-point lines, 5m+15m structure fractals, role
  `react_zone`): `typeEvidence 0.85`, freshness on the **anchor ladder**
  (`1.0/0.8/0.6/0.5`), recent-only lookback (5m: 144 bars · 15m: 96 bars, ≤3 per
  side per TF). Fixes the measured 43–86% of 5m swing turns with no seated level
  within ±8pt.
- **VWAP±2σ** (fair-value band lines): `typeEvidence 0.85`, VWAP family
  (one confluence family with VWAP/eVWAP/pdVWAP), anchor ladder. The σ is the
  volume-weighted standard deviation of typical prices over the **CME
  session-day window** (17:00 CT → read) — hand-calc verified 2026-08-27; the
  apparent "87pt σ" on range days is the legitimate session-day width, not an
  accumulation bug (`kernel/levels_volume.go` `vwapAndStdev`).
- **Standing rulings reconfirmed:** iFVG stays at `.35/.45/.65/.65` · anchors
  keep decay (`1.0/0.8/0.6/0.5`) · ±1σ shares the VWAP kind · zone floors/caps
  unchanged (1m→C · 15m→B · 1h/4h floor B).
- **pnl_corrected (T7):** every closed row now carries `pnl_corrected`
  (close-path stamp + boot backfill); readers `COALESCE(pnl_corrected,
  realized_pnl)`. Disagreements ≥ $0.50 log the class-killer WARN.
   → `SeatVolumeFamily` (≥1 volume seat) → `seatBothSides` (≥3 per side) → topN(8)
   → nearest-first (`:544-567, 724-825, 971-1047`).
9. Dedupe: same-kind within 1 tick collapses (S4, `kernel/levels_assemble.go` `dedupeSameKind`).

### 2.4 PLAN — what the planner writes

Prompt sections in render order (`kernel/planner_prompt.go:82-244`): Session →
Regime → Indicators → Ranked levels → Consumed levels → Level roles (playbook +
`bias_ctx`) → Structure (1 line/TF) → HTF zones → Auction story → Calendar →
Recent context → Owner note → Prior-plan invalidation → Prior-plan levels →
Anchor roles → OUTPUT contract.

The plan JSON (`kernel/plan_doc.go:85+`): reasoning FIRST, bias{direction,
conviction, flip_condition}, levels[≤8], scenarios[≤3], no_trade[], death_condition,
day_type, structured death/flip.

**The 7 condition types** (2 lines each):

| Condition | The play | When written |
|---|---|---|
| reclaim | long after close back above a broken level | `reclaim` trigger text cites the level; machine: `Swept && Reclaimed` (`kernel/scenario_state.go:139-142`) |
| hold | hold inside a zone until breakout/breakdown | trigger names the zone; status via level facts |
| sweep_reclaim | fade a liquidity sweep: wick through + close back | machine: sweep (wick+close back) then reclaim (`scenario_facts.go:276-297`) |
| reject | fade a first-touch rejection at a level | `reject` cites the level + 1x/2x close rule (condition×session guidance A2) |
| acceptance | trade the 2x5m/15m close beyond a level | acceptance rule from config (`store/strategy.go:1046`) |
| breakout_retest | break, then long the retest of the broken level | trigger cites breakout + retest price |
| fvg_entry | displacement ≥1.5×ATR5m leaves a gap; enter the retrace into it; invalid = close through distal edge | machine-verified: `kernel/fvg_entry.go` (3-candle relation, gap floor, CE >20pt, displacement, origin level) |

**confirm{}** (`kernel/plan_confirm.go`): rule `touch | 1x5m_close | 2x5m_close | 15m_close`,
ref_price, side. Rendered per cycle as `S# confirm: … — MET/NOT MET` + ADDENDUM S:
`MET (stale — written X context, price now Y; treat as expired)` when
|price−ref| > 2.0×ATR5m, and a `CONFLICT` trailer when opposite-direction confirms
are MET. Advisory — the AI stays the judge.
**death/flip** (`kernel/plan_doc.go:95-118`): structured `PlanCondition{price, side, rule, flip_to}`
— machine-evaluated every cycle; the plan dies when it fires.
**quality**: A+ / A / B / C (C = machine-demoted "level consumed", `plan_doc.go:69-73`).

### 2.5 EXECUTE — decision cycle + gates

The executor reads each cycle: KEY LEVELS table (+ROLE column, +`bias_ctx`), PLAN BLOCK
+ PLAN STATUS (confirm lines, touch ties, fvg states, dead-plan warning), structure
lines, positions/balance. Plan mode:

| plan_mode | Blocks | Gate site |
|---|---|---|
| advisory (default) | nothing | — |
| direction | entries against the plan bias | `trader/auto_trader_planconfig.go:204-215` |
| strict | entries with no MATCHED scenario cited (off-plan/empty/unknown id, or direction-mismatched) | `:216-222` via `kernel/plan_render.go:311-337` |

**Gate chain, verified order** (entries):

- In `runCycle` (pre-prompt): market-closed skip · no NT8 account skip · dead-man drive
  (`auto_trader_loop.go:130-155`) → kernel pre-prompt HOLDs (`engine_analysis.go`):
  `contract_roll`, `concurrent_cap`, `daily_guardrail`, `blackout_window`,
  `consistency_rule`, `prompt_ownership_violation`.
- Post-AI in runCycle: stale-bar discard · superseded wait · stale-reeval refusal
  (`auto_trader_loop.go:670-690`) · safe-mode filter (`:715-725`).
- In `executeDecisionWithRecord` (order, `auto_trader_orders.go:133-316`):
  1 feed-gate `feed_down: NT8 price feed not Connected` · 2 dead-man `dead_man_watchdog: …`
  · 3 A4 freeze `frozen: …` · 4 boot integrity `boot_integrity_refused: …` ·
  5 owner pause `stop_until: …` · 6 contract-roll · 7 consecutive-loss
  `consecutive_loss_halt: …` · 8 last-entry `last_entry_cutoff: …` · 9 session
  `session_gate: …` · 10 plan-mode `plan_mode: …` · 11 approval `approval_required`.
- In `validateDecision` (`kernel/engine_position.go:34-296`): action/size sanity →
  SL/TP sanity → R:R (`rr_gate`, `risk/reward ratio too low (x.xx:1)…`) →
  min-confidence (`confidence too low (NN), must be ≥NN`) → A3 MIN-SL
  (`🛑 MIN-SL REJECT … sl_too_tight: …`) → HTF veto (`🛡️ HTF VETO … htf_veto: …`) →
  transition stand-down → C6 dead-plan (`executor plan gate — …`) → R4 scenario
  quality (`cited scenario S# quality 'x' is below min_scenario_quality …`) → sizing.

Closes only ever hit feed-gate + hold-lock. Management paths (EOD-flat, trail, BE,
watcher) bypass the entry gates by design.

### 2.6 IN-POSITION

| Mechanism | What it does | Authority | Cite |
|---|---|---|---|
| Watcher | in-position AI reads; rails clamp downgrades | **zero order authority** | `trader/auto_trader_watcher.go:16-17,315` |
| Break-even | `breakeven_enabled` → stop to entry once profit ≥ trigger pts (default 50) | moves stop | `trader/auto_trader.go:191-212` |
| Trailing | ratchet stop = best ∓ mult×ATR(14,5m), arm after BE / after pts / immediate | moves stop | `trader/auto_trader_trailing.go:41-108` |
| EOD flat | session end (NY 14:45 CT, R-A15) → limit-then-market flatten | flattens | `trader/auto_trader_clock.go:410-449` |
| T1 force-flat | red-news blackout T−2 min | flattens | `auto_trader_loop.go:225-227` |
| stale_dodge | defer cycle across TF close | none | `trader/discard_burn.go` |
| Dead-man | NT8 link gap → blocks entries until reconcile | blocks entries | `trader/dead_man_watchdog.go:12-64` |

### 2.7 AFTER

- Post-exit: immediate rescan cycle (`cycle_trigger=post_exit`,
  `auto_trader_loop.go:264-267`).
- P&L corrections are additive: `pnl_corrected` + note; readers use
  `COALESCE(pnl_corrected, realized_pnl)` (`store/position.go` P0 comment).
- `level_stats` (nightly): `EvaluateLevelOutcome` grades touch ±4pt / react ≥8pt-in-3-bars
  / broke-clean / chopped (`kernel/level_stats_calc.go`; wire `ninjatrader/level_stats_wire.go`).
- `touch_episodes`: band 16t(4pt), 12-bar close, vol ratio vs 20-bar baseline,
  approach 5 bars, wick/body penetration, 1m+5m close side (`kernel/touch_telemetry.go`).
- Expectancy tables: rebuild from `position_plan_join` (S3) — condition × quality ×
  session from closed positions joined to plans via `plan_id` (unresolvables counted
  separately, never dropped; see §9 and `scripts/backfill-position-plan.py`).

## 3. SESSIONS & TIME

| Session | Window (CT) | Flat | Killzones | Read |
|---|---|---|---|---|
| ASIA | 17:00 → 02:00 | 02:00 | 19:00–23:00 | 16:55 |
| LONDON | 02:00 → 08:30 | 08:30 | 02:00–05:00 | 01:55 |
| NY | 08:30 → 14:45 | **14:45 (R-A15)** | 08:30–11:00 · 13:00–14:45 | 08:25 |

`kernel/session_registry.go:83-120`. Lunch = hard entry gate 12:00–13:30 CT
(`trader/auto_trader_session.go:123-125`). Last entry = session end − 15 min
(NY 14:30). DST via tzdb America/Chicago (`kernel/tz.go`); CME halt 16:00–17:00
untraded (`kernel/cme_calendar.go:29-33`). EOD ladder: limit at favorable side
(`EOD_FLAT_LIMIT_TICKS`), market after `EOD_FLAT_MARKET_AFTER_SEC`.

## 4. SAFETY MODEL

Hard blocks (real gates, can refuse an entry): boot integrity · SIM account check ·
feed-gate · dead-man · freeze · pause · contract-roll · consecutive-loss ·
last-entry · session · plan-mode · approval · validateDecision chain (R:R, min-conf,
MIN-SL, HTF veto, transition, dead-plan, scenario quality) · stale-data feed block ·
stale-bar discard.

Advisory (never gates): plan itself, confirm lines, CONFLICT trailer, touch
telemetry, roles, bias_ctx, watcher, plan citations, gate-block counters are
observability. Fail-open examples: min-SL skipped when 5m ATR absent
(`kernel/engine_position.go:216-218`), stale-confirm annotation skipped when ATR
missing (`kernel/plan_confirm.go`), HTF veto with no snapshot. Fail-closed:
boot integrity, SIM lock, plan-mode strict with no plan, fail-closed plan marker.

Guardrails master OFF → limits logged as `guardrail WOULD have tripped`
(soft telemetry, `engine_analysis.go:155-202`); size caps (notional×N, per-order
contracts) stay enforced regardless.

## 5. SELF-SCORING

| Surface | Accumulates | Answers |
|---|---|---|
| `level_stats` | per seated level per session-day: touch/react/broke/chopped | are our A/B/C grades honest? |
| `touch_episodes` | per touch: shape, vol ratio, approach, close side | what does price DO at our levels? |
| gate-block counters | per (trader, session-day) per gate label; 17:00 CT rollover | which gate fired how often? (`telemetry/gate_blocks.go`) |
| refusal telemetry | `ENTRY REFUSED: <reason>` in execution logs + structured error events | why did we NOT trade? |
| Decision Audit Trail | last 100 cycles: action, confidence, refusal, latency | what was the AI thinking? (`/api/audit/decisions`) |
| Discipline panel | GPA of adherence grades A–F + MAE/MFE | did entries follow the plan? |

---

# PART B — UI REFERENCE

Routes: `/traders` (AITradersPage) · `/dashboard` (TraderDashboardPage) ·
`/strategy` (StrategyStudioPage) · `/settings` · `/agent` (Beta chat) ·
`/login` `/register` `/setup` `/faq` (`web/src/router/AppRoutes.tsx:468-568`).
Removed pages (no routes): Data, Strategy Market, Competition
(`AppRoutes.tsx` route map; legacy hash redirects only).

## 6.1 Dashboard (`pages/TraderDashboardPage.tsx`)

| Element (label) | What it does | Cite (FE → API → engine) |
|---|---|---|
| ⏸ Pause menu `30m / 1h / Session end / custom` · `▶ Resume` | blocks new entries until … CT; resume reopens | `components/trader/PauseButton.tsx:87-150` → `POST /api/traders/:id/pause` / `resume` → `api/handler_trader.go:943,1013` |
| `Emergency Flat` (modal confirm) | flattens everything now | `EmergencyFlatButton.tsx:62-77` → `POST /api/risk/force-flat` → `api/handler_risk.go:78` |
| Account selector + `Select an account` (SIM-only rows) | picks the NT8 account this trader may trade | `AccountSelector.tsx:142-224` → `GET /api/accounts`, `POST /api/account/select` → `api/handler_account.go:43,109` |
| StatCards `Total Equity / Available Balance / Total PnL / Positions` | account snapshot | `TraderDashboardPage.tsx:596-655` ← `/api/account` `handler_order.go:129` |
| Tabs `Overview / Decisions` | switch panels | `:674-686` |
| `🧠 Decision Audit Trail — last 100 cycles` | every cycle: action, confidence, refusal reason, latency | `DecisionAudit.tsx:118-148` ← `/api/audit/decisions` → `api/handler_decisions.go:28` |
| Positions table + row `Close` | closes that position | `:756-990` → `POST /api/traders/:id/close-position` → `api/handler_trader_status.go:124` |
| Chart tabs `Equity Curve / K-line` + intervals + `Search symbol…` | charts | `ChartTabs.tsx` → `/api/klines`, `/api/klines/svp`, `/api/equity-history` |
| `🧠 Recent Decisions` (5/10/20/50/100) | decision cards with reasoning + CoT toggle | `DecisionCard.tsx` ← `/api/decisions/latest` |
| `📜 Position History` + stats + filters | closed trades, win rate, PF, Sharpe | `PositionHistory.tsx` ← `/api/positions/history` |
| Debug strip `SYSTEM_STATUS::ONLINE …` | **[BUG] static string, not a health probe** | `TraderDashboardPage.tsx:571-594` |

## 6.2 Plan Card (`components/plan/*`)

Data: `GET /api/plan/today` → `api/handler_plan.go:179` (plan_final = doc + overlays).

| Element (exact label) | Meaning | Cite |
|---|---|---|
| bias word `LONG/SHORT/NEUTRAL` + `{conviction} conviction` + `↻ Flips · {…}` | plan direction + flip rule | `BiasBlock.tsx:45-80` ← `doc.bias` |
| level row: price · provenance chip (`PDH`, `nPOC·…`) · grade chip A/B/C · `m:{grade}` machine-grade · freshness dot `fresh/tested/consumed` · instruction verb · signed dist (gold when <12pt) | the seated table | `ZoneTable.tsx:64-147` ← `level_facts[]` `handler_plan.go:433-461` |
| `👤` owner marker · `📝` note marker · `⚡ conflict` ghost (strike-through = conflict, NOT consumed) | owner levels + owner-vs-AI conflicts | `ZoneTable.tsx:69,76-92` |
| touch chip `○ approaching · ◐ touching · ✕ rejected · ▲ accepted` | live touch episode state (advisory) | `ZoneTable.tsx:18-23,124-133` ← `touch_state` |
| scenario row: dot `○ armed ◌ waiting ● triggered ✕ invalidated` · `S#` · quality chip · `LEVEL CONSUMED` badge · direction · trigger · targets · invalid · confirm chip `CONFIRM MET / confirm not met` · fvg chip `ABOVE/BELOW/IN_ZONE/FILLED_INVALID · #n` | scenario machine state | `ScenarioList.tsx:53-246` ← `scenario_status`, `scenario_meta.confirm`, `fvg_states` |
| `⛔ NO-TRADE — re-read budget exhausted` + fail-closed reason | the marker written when replan budget died | `SessionPlanCard.tsx:469-484` ← `lifecycle='no_trade'` |
| `v1…vN` chips + tooltip `v{n} — {death_reason}` + historical view | plan version history | `SessionPlanCard.tsx:296-312,507-594` ← `/api/plan/versions` |
| `⚠ DEGRADED n/7` · `ADVISORY` banner · `WARMING n` · `UNCALIBRATED` | regime dark-fields / mode / data maturity | `:355-449` |
| `✎ {n} edit(s) could not carry into this version` | edits lost at re-plan — review | `:639-683` ← `uncarried_edits` |
| `⏸ TRANSITION — awaiting confirmation` | **[BUG] dead chip — `transitionState` is never called** | `:685-705`; `handler_plan.go:1078` uncalled |
| `◌ planner is writing a fresh plan…` | a read is in flight | `:331-346` ← `reading` |

**Buttons (budget semantics — verified):**

| Button | Endpoint | What it actually does | Budget | Confirm modal |
|---|---|---|---|---|
| `⟳ Re-read` | `GET/POST /api/plan/reread` (`handler_plan.go:983,1001`) | new planner call, same chain, `trigger_reason="owner_reread"` | **spends 1 re-plan** | yes — `Spend one of {n} re-reads?` |
| `↺ Reset planner` | `GET/POST /api/plan/reset` (`:1030,1044`) | abandons chain, history kept, full budget restored, fresh read; positions untouched | restores budget | yes — `Abandon this plan chain and start fresh?` |
| `⟳ Re-align plan` | `POST /api/plan/realign` (`:1906`) | planner proposes a patch; `✅ Apply merge` applies as overlay | own auto-cap + 20s debounce (manual button bypasses) | no — the patch card IS the confirm |
| `Approve` | `POST /api/plan/approve` (`:1799`) | grants entries for this CME session-day (when approval_required ON) | none | no modal (by design) |
| Edit sheet `Save` | `POST /api/plan/overlay` (`:747`) | RFC-6902 patch with B2 price armor (422 `⛔ price armor: …`); then auto-realign | none | no modal |
| `＋ Add level` / `Bulk add` | `POST /api/plan/owner-level` (`:1634`) / overlay bulk | owner level (sticky 👤, carries across re-plans) | none | no modal |

**"saved ✓" rule:** no persistent inline checkmark exists — save success is a toast
`Plan updated` (`EditSheet.tsx:148`); the only inline ✓ is the realign no-change
chip `✓ planner: no plan change needed` (`RealignPanel.tsx:84`). Verify by re-opening
the card.

## 6.3 Strategy Studio (`pages/StrategyStudioPage.tsx`)

### Risk Control (`RiskControlEditor.tsx` → `store.RiskControlConfig`)

| Label | Field (store/strategy.go) | Range/clamp/default | Engine consumer |
|---|---|---|---|
| 🔒 Hold discipline | `hold_discipline` | toggle, OFF | `trader/auto_trader_orders.go:84` |
| 🎯 Move stop to breakeven + Trigger (points) | `breakeven_enabled` / `breakeven_trigger_points` | OFF / 1–1000 step 5, default 50 | `trader/auto_trader.go:191-212` |
| 📈 Trailing profit + ATR multiple + ATR period (5m) + Arms + Arm points | `trailing_enabled` `trailing_atr_mult` `trailing_atr_period` `trailing_arm` `trailing_arm_points` | OFF / 0.5–10 step 0.5 def 2.0 / 5–50 def 14 / after_breakeven·after_trigger_points·immediate / 1–1000 def 50 | `trader/auto_trader_trailing.go:41-108` |
| Max Positions | `max_positions` | 1–3, default 3 | `kernel/engine_analysis.go:128-134` |
| Min Risk/Reward Ratio | `min_risk_reward_ratio` | 1–10 step 0.5, default 3 | `kernel/engine_position.go:142-146` |
| Min Confidence | `min_confidence` | 50–100, unset→60 | `kernel/engine_position.go:190-191` |
| Risk guardrails (master) | `guardrails_enabled` | nil→ON | `engine_analysis.go:155-202` |
| Daily loss limit (USD) / profit target / max daily trades | `daily_loss_limit_usd` `daily_profit_target_usd` `max_daily_trades` | toggles + min-0 inputs; loss ON, profit/trades OFF | `kernel/risk_limits.go:235-243` |
| Consecutive-loss halt (stop after N losers) | `consecutive_loss_halt` | toggle→2, default 0 (off); NOT master-gated | `trader/auto_trader_orders.go:107-135` |
| Re-entry cooldown after stop (futures) | `reentry_cooldown_minutes` | toggle→20, default 0 | `kernel/engine_analysis.go:585-595` |
| Consistency cap (day ≤ % of total) | `consistency_max_day_pct` | 0–100, OFF; **no backend clamp** | `engine_analysis.go:195-199` |
| Max contracts / order (AlwaysOn badge) | `max_contracts_per_order` | min 0, default 2 | `trader/auto_trader_orders.go:57` |
| Notional cap (equity × N, AlwaysOn) | `max_notional_leverage` | min 0, default 20 | `kernel/risk_limits.go:300` |
| Blackout start/end (CT) | `blackout_enabled` `blackout_start_ct` `blackout_end_ct` | free HH:MM, OFF | `engine_analysis.go:187-192` |

Crypto-only fields (hidden on futures): BTC/ETH + Altcoin leverage sliders, Position
Value Ratio (read-only), Max Margin Usage (prompt hint only), Min Position Size.

### Day Plan (`DayPlanEditor.tsx` → `store.DayPlanConfig`)

| Label | Field | Range/clamp/default | Consumer |
|---|---|---|---|
| Enable Day Plan | `plan_enabled` | toggle, **false** | `kernel/engine_analysis.go:362` |
| Planner model | `planner_model` | free text; empty → primary model | `trader/auto_trader_planner.go:31-74` |
| Plan mode | `plan_mode` | advisory / direction / strict | `trader/auto_trader_planconfig.go:195-225` |
| Planner reads | `planner_timeframes` | multiselect, default [D,4h,1h,15m] | planner structure summary |
| Proximity | `proximity_filter_atr` | **UI slider 0.5–3.0** (planner resolver accepts 0.1–3.0 — [BUG 3]) default 1.5 (live: retuned 0.3) | `engine_analysis.go:369-371` |
| Max levels / Max scenarios / Max re-plans | `max_levels` `scenario_cap` `replan_cap` | 3–12 def 8 / 1–5 def 3 / 0–4 def 2 | `engine_analysis.go:363-365` etc. |
| Acceptance | `acceptance_rule` | 2×5m / 15m | `store/strategy.go:1046` |
| Approval required | `approval_required` | toggle, false | entry gate `auto_trader_orders.go:302` |
| Digest | `evening_digest` | toggle, true | `trader/auto_trader_planner.go:1457` |
| Last entry (CT) / EOD flat (CT) | `last_entry_ct` `eod_flat_ct` | **[BUG] vestigial — nothing evaluates them** | `auto_trader_clock.go:260,299-302` |
| Max re-alignments | `realign_cap` | 0–10 def 5 | `auto_trader_planconfig.go:229-235` |

**Wake toggles** (`trader/auto_trader_wake_levels.go`):
`Wake on 15m zones` ON · `Wake on HTF zones (1h/4h)` ON · `Wake on HTF order blocks`
OFF · `Wake on seated-level invalidation` ON · `Wake on iFVG inversion` ON ·
`Min wake interval (min)` 5–120 UI / Go ≤0→30 · `Guarantee a 1h S/D seat` ON ·
`Min scenario quality` A/B/C default C. Wakes spend NO replan budget.

**Per-session accordions** (NY/ASIA/LONDON): enable toggle + overrides (⚪ inherit /
🔸 override): `Min grade` A/B/C · `Min scenario quality` · `Max trades` 0–20 ·
`Plan mode` · `Max re-plans` 0–4 · `Acceptance`. Precedence everywhere:
**per-session > strategy-level > shipped default** (resolvers
`store/strategy.go:1007-1323`).
**[BUG] `last_entry_offset_min` / `eod_flat_offset_min` have no UI control** — they
exist and gate live behavior (offsets 15 / 0) but are not editable anywhere in the FE.

**Write paths:** create `POST /api/strategies` replaces config wholesale
(`api/strategy.go:174`); edit `PUT /api/strategies/:id` deep-merges
(`store.MergeStrategyConfig`, absent fields preserved) and hot-reloads every trader
bound to the strategy (`api/strategy.go:268,406-427`). Save toast, then verify the
card re-renders.

### Trader-level knobs (`TraderConfigModal.tsx` → `store.Trader`)

AI Scan Decision Interval (1–60, default 3) · cadence `interval`/`bar_close` ·
AI model (full name + provider) · in-position mode `ai_watch`/`bracket_only`.
(`store/trader.go:25-35`; consumers `manager/trader_manager.go:613-627`.)

## 6.4 Status surfaces

| Surface | Where | Cite |
|---|---|---|
| Boot ledger (BOOT INTEGRITY + volume-wave + touch + fvg + S-wave lines) | log `data/nofx_<date>.log` at boot; NOT shown in the UI | `main.go:227+`, `kernel/levels_volume_boot.go` |
| 402/payment banner | **[GAP] no payment banner exists in the UI**; 402 appears only in logs (`💸 DEEPSEEK PAYMENT FAILURE …`) | `trader/auto_trader_loop.go` 402 branch |
| Sandbox banner | `🧪 SANDBOX — isolated test copy · not live` | `SandboxBanner.tsx:34-35` |
| Running revision | Settings footer `running rev {revision}` | `SettingsPage.tsx:818-822` ← `/api/config` |
| Feed/clock staleness | DecisionCard hard-❌ on `verdict_hint=feed|clock` | `DecisionCard.tsx:373-434` |
| Gate-blocks panel | per-gate label + count, session-day rollover 17:00 CT | `GateBlocksPanel.tsx` ← `/api/risk/gate-blocks` |
| Gate labels decoded | `feed_down`=NT8 feed down · `dead_man`=Dead-man watchdog · `frozen`=Trader frozen · `boot_integrity`=Boot integrity · `consecutive_loss`=Consecutive-loss halt · `last_entry`=Past last-entry time · `session_gate`=Outside session window · `plan_mode`=Against the plan · `approval_required`=Awaiting approval · `clock_skew_observed`=Clock drift · `b3_order_dedup`=Duplicate order dropped · `b3_rate_breaker`=Order rate breaker · `level_burned_retouch`=Burned level re-touched · `night_transition`=Night/day transition | `GateBlocksPanel.tsx:20-35` |
| Would-trip counters | `guardrail WOULD have tripped` log + `/api/risk/errors` | `engine_analysis.go:155+` |
| Alerts center | 🔔 unacked P0/P2 + digests | `AlertCenter.tsx` ← `/api/plan/alerts` |

## 7. HOW-TO PLAYBOOKS (10)

1. **Morning check**: dashboard account card (equity/positions) → Plan Card session
   tab = NY → read bias + NO-TRADE banner → GateBlocks panel for overnight refusals →
   alerts bell. (All from `/api/account`, `/api/plan/today`, `/api/risk/gate-blocks`.)
2. **Reading the plan card**: bias first, then the levels table (grade chips + `m:`
   machine grade + freshness dots + dist column), scenarios (dots + confirm chips),
   footer (re-reads left, model, day type).
3. **NO-TRADE session**: the banner means the replan budget died. Use `↺ Reset
   planner` (restores budget, abandons chain) — not Re-read (that would fail-closed
   again). Confirm modal text spells both out.
4. **Flip plan_mode**: Strategy Studio → Day Plan → Plan mode segmented control (or
   per-session accordion override). strict = no entries without a matched scenario
   citation; direction = no against-bias entries; advisory = no plan gate.
   Revert the same way (override chip back to ⚪ inherit).
5. **Changing a knob safely**: edit → `Save` → toast `Plan updated` → confirm the
   value on re-open. Strategy edits hot-reload bound traders (no restart,
   `api/strategy.go:406-427`). Never edit `data/data.db` by hand — config is cached
   at trader load.
6. **Adding an owner level**: Plan Card → `＋ Add level` (or tap a row) → price +
   type + grade + instruction + note + scenario tag → Save. Owner levels are 👤
   sticky and carry across re-plans. Delete via the sheet (owner-scoped).
7. **Reading a refusal**: Decision Audit Trail row → refusal reason string. Each
   message format is decoded in §2.5 and §6.4 (e.g.
   `sl_too_tight: 20.2 < 1.0×ATR (21.8) — widen or skip` = MIN-SL gate).
8. **Why did a trade happen?** Position row → `plan_id` (S3 stamp) → plan version →
   scenario → trigger/confirm. The `position_plan_join` view does the join;
   unresolvable rows are counted as their own row, never guessed.
9. **Weekly review**: pull Position History (condition/quality via plan join) ·
   `level_stats` table · `touch_episodes` · gate-block journal line (17:00 rollover).
10. **Emergency stop + flat**: trader `⏸ Pause` (blocks new entries) → `Emergency
    Flat` (flattens now) → or `Stop` the trader from the traders page. Never needed
    for account safety (SIM-only), only for experiment hygiene.

## 8. GLOSSARY (one line each)

PDH/PDL/PDC prior-day high/low/close · ONH/ONL overnight (Asia+London) high/low ·
RTH regular trading hours · OR-H/L opening range · IB initial balance ·
nPOC naked point of control · POC point of control · VAH/VAL value-area high/low ·
VWAP volume-weighted average price · eVWAP extended VWAP (15:00 CT anchor) ·
SETT settlement · MID-O midnight open · EQH/EQL equal highs/lows · FVG fair-value gap ·
iFVG inverse FVG · OB order block · S/D supply/demand · MSS market-structure shift ·
BOS break of structure · CE consecutive encroachment (gap midpoint) ·
dATR = DailyRangeProxy (mean session-day range — NOT an ATR) · ATR average true range
(Wilder-14) · MET machine-confirm verdict · m: machine grade chip · S# scenario id ·
v# plan version · R:R risk:reward · BE break-even · EOD end of day · T1 tier-1 (red
news) · CT America/Chicago time · SIM simulated account · uPnL unrealized P&L ·
PF profit factor · MAE/MFE max adverse/favorable excursion · RN round number ·
HTF higher timeframe · TF timeframe · KZ killzone · CoT chain of thought.

## 9. DISCREPANCIES FOUND (status @ running rev `717acd34`)

1. **[FIXED]** `⏸ TRANSITION` chip removed (C2, `92bf01ed` wave) — backend `transitionState` still uncalled by design.
2. **[FIXED]** `last_entry_ct` / `eod_flat_ct` hidden (C3) — live gates use per-session offsets (`LastEntryOffsetFor`/`EODFlatOffsetFor`, defaults 15/0 min).
3. **[FIXED]** Proximity slider floor 0.1 (C1) — matches resolver clamp 0.1–3.0.
4. **[FIXED]** SYSTEM_STATUS strip live from `GET /api/health` every 30s + shows REV (C4).
5. **[FIXED]** gate-blocks `?trader_id=` filter (C5) — server scopes to the trader + process-wide `""` key.
6. **[FIXED]** `scenario_meta.confirm` typed + loose cast removed (C6).
7. **[FIXED]** quality A uses `--vl-grade-a`; neutral direction renders muted (C7).
8. **[QUEUE]** role badges / `fresh·xN` / `◆ new since plan` — feature work, not a bug.
9. **[FIXED]** dead `trigger_reason`/`overlay_count` types + unused i18n keys removed (C9).
10. **[FIXED]** 402 banner on the dashboard polling `/api/risk/errors` for `ai_payment_402` (C8).
11. **[BY DESIGN]** consumed levels dimmed (not struck); strike = conflict-ghost.
12. **[BY DESIGN]** Approve is one-click (toast), no modal.

## 10. 2026-08-27 WAVES (as-built @ `717acd34`)

- **Planner contract (A1–A5):** BIAS-TREE section (6 branches, `bias-tree:` reasoning quote required) · `chain_after` link + `ChainWarnings` (WARN) · no-trade gates (≤8 lines) · killzone weighting · STOP-DOING. Boot line `📜 planner playbook: playbook=v2 …`.
- **P0 side-quota relax:** `ValidatePlanDocWithFactsMachine` — machine-thin side (the assembled map itself has < quota) → **WARN + write** with a `thin_side` note (card ⚖ chip); AI-caused omission → still rejected; 0-on-a-side / empty map → hard fail-closed. Knob **`min_side_levels`** (Studio, base + per-session; env `MIN_SIDE_LEVELS`; default 2, 3 = the old hard rule). Boot line `⚖ side-quota (P0-relax): min_side=cfg(default 2 …)`.
- **Bias-tree facts fix:** `BiasContext` carries PDH/PDL/PDC and the write site stamps them from the FULL detector universe (`ApplyUniverseDayAnchors`) — post-roll reads no longer render "no PDH/PDL anchor"; branch-5 premium/discount now anchors on the DEALING RANGE (PDH/PDL) with `BEYOND range high/low (extended)` for out-of-range prices (no more >100% figures).
- **Observability:** every planner-attempt rejection now logs `📐 planner attempt N/3 …` (was silent).

