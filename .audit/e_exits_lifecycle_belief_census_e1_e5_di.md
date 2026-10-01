names rewritten to vl on 2026-09-30 (VL rename)
SUBSYSTEM E — EXITS / LIFECYCLE + D3
Worktree /home/hoang/vl-conform @ fb50903f (base dev tip 492d2067). Deployed rev 70af663d, PID 878451, boot 2026-09-04 08:30:11 CT.
Main tree /home/hoang/vl READ ONLY. DB read via `file:/home/hoang/vl/data/data.db?mode=ro`.

REPORT PINS (`git log -1 -- <path>`, run in /home/hoang/vl-conform)

```
docs/superpowers/reports/2026-09-02-belief-census.md             ee64a494  2026-09-02 08:50:38 -0500
docs/superpowers/reports/2026-09-04-two-day-audit.md             f3c640c3  2026-09-04 07:26:52 -0500
docs/superpowers/reports/2026-09-03-trade-excursions.md          0c1a808c  2026-09-03 00:05:11 -0500
docs/superpowers/reports/2026-08-30-knob-census.md               741bfc2a  2026-09-01 07:58:16 -0500
docs/superpowers/reports/2026-09-02-0b-exit-sanity.md            d7048483  2026-09-02 08:04:57 -0500
docs/superpowers/reports/2026-09-02-level-event-wake-audit.md    a5a53bec  2026-09-02 08:27:52 -0500
docs/superpowers/reports/2026-09-01-full-system-audit.md         8f57f845  2026-09-03 11:41:54 -0500
docs/superpowers/reports/2026-09-03-expectancy-1d.md             38a63a9b  2026-09-03 15:26:02 -0500
docs/superpowers/reports/2026-09-03-mc-drawdown.md               77e1cdfc  2026-09-03 00:39:25 -0500
docs/superpowers/AUDIT-CHECKLIST.md                              158743db  2026-09-04 08:11:57 -0500
docs/superpowers/research/INDEX.md                               4e8e7e1a  2026-09-03 19:37:14 -0500
```

A11 RESOLUTION NOTE. `/api/config/resolved` and `/api/risk/gate-blocks` return `{"error":"Missing Authorization header"}` from this session — every RESOLVED value below comes from the boot-8 lines quoted in the dispatch, from the log file `/home/hoang/vl/data/vl_2026-09-04.log`, or from the resolver code path with the env confirmed absent. `grep -E "EXIT_MECHS|MIN_SL|ARM_STOP|STAGE_A" /home/hoang/vl/.env` returns NOTHING (27 assignments in that file, none of them these) [A], and `/proc/878451/environ` is unreadable to this session, so every E-subsystem resolver falls through to its shipped default. That is stated as "env absent → default", never as "the file default is live".

---

## 1. THE RULE TABLE

| # | rule | file:line | RESOLVED NOW | label | report:line grounding | live effect | CONFORMS? | production callers |
|---|---|---|---|---|---|---|---|---|
| E-1 | Stop floor `\|entry−SL\| ≥ MIN_SL_ATR_MULT × ATR5m(14, Wilder)` | `kernel/min_sl.go:34` const, `:44` resolver; enforced `kernel/engine_position.go:227-231`, `trader/armed_executor.go:1361-1362` | **1.5** (boot 8 `min-sl guard: atr_mult=1.5`; `MIN_SL_ATR_MULT` absent from .env) | **[O]** owner ruling 2026-09-02 + **[T]** own tape. NOT [R] — see drift 8 | 0b-exit-sanity.md:10 (owner-rulings commit 4175e0b6) · AUDIT-CHECKLIST.md:621-641 (class 43) · own-tape 15/27 losers stopped-too-tight, AUDIT-CHECKLIST.md:627-628 | **REJECT** (AI-entry gate + arm feasibility) **and widen** (composeArmStop) | **NO** vs knob-census: research value **1.0 [C]** (knob-census.md:27, 133) vs resolved **1.5**. Drift is INTENTIONAL and documented at min_sl.go:20-25. The *[R] half of the citation* does not conform — see drift 8 | **12** — e.g. `kernel/engine_position.go:215` |
| E-2 | Level clearance: SL ≥ 2 ticks BEYOND the cited anchor | `kernel/min_sl.go:40` const, `:72` anchor resolver; enforced `kernel/engine_position.go:236-246` | **2 ticks = 0.50 pt** (boot 8 `level_clearance=2tick(s)`; tick 0.25 from boot 8 `NT8 instrument_info MNQ`) | **[T]** own tape | AUDIT-CHECKLIST.md:629-630 — "on the five biggest losers **0 of 5** stops sat ON a seated level and **2 of 5** sat in dead zones 40+ pts away" | **REJECT** (leg 2) + widen at arm | yes — 2 ticks live = 2 ticks researched | **3** — `kernel/engine_position.go:236`; `trader/armed_executor.go:392`; `trader/auto_trader_dayplan.go:58` |
| E-3 | Anchor dead-zone bound `ARM_STOP_ANCHOR_MAX_ATR` | `trader/arm_stop_anchor.go:35` const, `:38` resolver | **3.0×ATR5m** (boot 8 `anchor_max=3.0×ATR5m`; env absent) | **[I]** — the code says so itself: `arm_stop_anchor.go:33` "**CHOSEN DEFAULT, NOT AN OWNER RULING**: flagged in the 0B report for a ruling" | none found — no report carries a 3.0 value | gate (beyond it → `stop_unanchored`, ATR floor governs) | unknown — no research value exists to compare | **3** — `trader/armed_executor.go:392`, `:404`, `trader/exit_mechs_suspend.go:96` |
| E-4 (census **E1**) | **Breakeven at +40pt moves stop to entry** | `trader/auto_trader.go:148-189`; trigger `:195`; suspension `trader/exit_mechs_suspend.go:35,61,157` | **OFF** (boot 8 `BE=off`; `EXIT_MECHS_SUSPENDED` absent → `exitMechsSuspended()` returns **true**) — while strategy `MNQ` carries `breakeven_enabled:true`, `breakeven_trigger_points:40` [A] | **[O]** owner-ruled | belief-census.md:84 | **NONE** — `exitMechSuspendedRefuse` returns before `moveStopWire`, so no `move_stop` frame can reach the AddOn | **NO — research/owner value = ON at +40 pt; resolved value = OFF** | **1** — `trader/auto_trader_risk.go:101`. The wire hop (`auto_trader.go:176`) is UNREACHABLE while suspended |
| E-5 (census **E2**) | **Trailing 2.0×ATR14 after breakeven** | `trader/auto_trader_trailing.go:115-195`; `trailingConfig:42`; suspension `:180` | **OFF** (boot 8 `trail=off`) — while strategy carries `trailing_enabled:true`, `trailing_atr_period:14` [A] | **[O]** owner-ruled | belief-census.md:85 | **NONE** — same wire seam | **NO — research/owner value = ON, 2.0×ATR14; resolved value = OFF** | **1** — `trader/auto_trader_risk.go:104` |
| E-6 | Stage-A contract cap | `kernel/risk_limits.go:305` const, `:308` resolver, `:318` clamp | **1** (boot 8 `size=1`; `STAGE_A_CONTRACT_CAP` absent) | **[O]** (risk_limits.go:292-297 "0B, owner ruling 2026-09-02") supported by **[T]** | mc-drawdown.md:230 — expectancy indistinguishable from zero at n=64, ~1810 trades needed; risk_limits.go:293 states the release criterion "n≥30 closed trades with a positive lower-CI expectancy" | **gate** (clamps every futures order size) | yes | **2** — `trader/auto_trader_orders.go:532`, `:680` (via `resolveMaxContracts` → `kernel.ResolveMaxContracts:289` → `ClampStageAContracts`). NOT dead |
| E-7 (census **E3**) | EOD flat at session end | `trader/auto_trader_clock.go:472/476`; times `kernel/session_registry.go:107,109` | **NY 14:45 CT**, LONDON 08:30, ASIA 02:00 (boot 8 `sessions[…]`) | **[O]** | belief-census.md:86; `session_registry.go:78` "NY CONTRACT (owner, 2026-08-16)" | **gate** — force-flatten through the trader close path, bypasses hold-lock | yes — 14:45 CT live = 14:45 CT ruled | **1** — `trader/auto_trader_loop.go:314` |
| E-8 (census **E4**) | Flip/death → dormant + auto re-arm, "replan budget untouched" | `trader/auto_trader_planner.go:299-313` (dormant write + log), re-arm predicate above it | **ON**, budget untouched (boot 8 replan line: `free={… dormant/rearm}`) | **[O]** | belief-census.md:87 | **lifecycle** — entries blocked, `continue` before `store.GetReplanBudget` at `:320`, so no spend | yes | **1** path — `trader/auto_trader_planner.go:299` |
| E-9 (census **E5**) | Level-event wake deserves a re-read | `trader/auto_trader_wake_levels.go:246/250`; cadence cutoffs `:279-300` | **cutoff 25m ENFORCE · cooldown 30m ENFORCE · fast-market ≥1.5×ATR exempt · cross-session on · stale-arm-expiry on** (boot 8 `wakes:`) | **[T] weak** | level-event-wake-audit.md:21 (**n=52** re-plans / 7 days) and **:46** ("Across all 52: **7** versions carry an arm that ever reached working or filled") | **REPLAN trigger**, now cut off | yes — the audit proposed WARN-first N=25 (report:106-125); the code at `:279-284` records the owner's 2026-09-03 promotion of that same N=25 to ENFORCE | **3** — `trader/auto_trader_planner.go:274`, `:282`, `:347` |
| E-10 | `re-arm-after-sweep` (boot-line field) | `trader/exit_mechs_suspend.go:96` | **on** — but it is a **hardcoded `true` literal**, not a resolver | **[M]** mechanics | none found | **label only.** The real mechanism is class 33's `sweepPreBootArms` (`trader/armed_executor.go:201`, `trader/class33_boot_sweep.go`), which is unconditional | the printed value happens to be true, but it is **not READ** — see drift 5 | **1** — `trader/exit_mechs_suspend.go:96` |
| E-11 | `trade_excursions` writer (class 54 / wave 1A) | `trader/trade_excursion_hook.go:41` open, `:77` bar-tick, `:172` close | **logging=on, rows=0, backfilled=0, unresolved=0** (log line, `vl_2026-09-04.log`, 2 boots today) | **[M]** | trade-excursions.md:7 ("Status: NOT DEPLOYED" — now stale) · two-day-audit.md:1074 · expectancy-1d.md:163 | **advisory / telemetry**, zero gates | **NO** — the table it exists to fill has **0 rows all-time** | **4** — `trader/auto_trader_decision.go:492`, `trader/armed_executor.go:1254`, `trader/auto_trader_risk.go:54`, `trader/auto_trader_clock.go:768` |
| E-12 | Prompt statement of the arm stop-distance floor | `kernel/planner_prompt.go:733` | prompt string says **"stop distance must be ≥ 1.0× the current 5m ATR"**; the enforced floor is **1.5** | **[X]** contradicted by the live resolver | resolver `kernel/min_sl.go:34` = 1.5; boot 8 `atr_mult=1.5` | **advisory** (prompt text) — but it instructs the author to a value the gate will refuse | **NO — 1.0× in the prompt vs 1.5× resolved** | **1** — the planner prompt builder |
| E-13 | Class-45 resolved stop-floor block in the prompt | `kernel/class45_feeds_forward.go:195-201` | renders `## Minimum stop distance this cycle / X pts (1.5×ATR5m Y, resolved)` — READ, correct | **[M]** | boot 8 `prompt feeds forward: … stop-floor=1.5×ATR5m…` | advisory | yes | **1** — `trader/auto_trader_planner.go:2381` (`StopFloorMult`) |

---

## 2. THE DRIFT ROW THE DISPATCH ASKED FOR — E1/E2 [O] SUSPENDED BY A WAVE

**Both halves confirmed [A] from the DB, not from a file.**

`sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro"` → strategy `MNQ` (`a5b7662e-7bf7-49bb-9f09-7efa48f95ac8`), `ai_config.risk_control`:

```
breakeven_enabled        = True
breakeven_trigger_points = 40
trailing_enabled         = True
trailing_atr_period      = 14
```

`trailing_atr_mult` and `trailing_arm` are **NOT PRESENT** in the stored config.

Against boot 8: `exits: … BE=off · trail=off …`.

**The contradiction is already on the record** — `2026-09-04-two-day-audit.md`:

- **:607** — `| **breakeven** | **OFF** — *despite the strategy carrying `breakeven_enabled: true`, `trigger=40pts`* | boot line `BE=off` |`
- **:608** — `| **trailing stop** | **OFF** — *despite `trailing_enabled: true`* | boot line `trail=off` |`
- **:622-626** — the paragraph naming both boot lines
- **:930 (finding D44)** — "**Two owner-set knobs are silently off** — strategy `MNQ` carries `breakeven_enabled: true` (trigger 40 pts) and `trailing_enabled: true`; the 0B wave suspends both, and **two boot lines in the same boot disagree**: `🛑 exits: … BE=off · trail=off` vs `🧾 ledger boot: … trailing=2.0×ATR14 arm=after_breakeven (source: studio)`"

**The suspending authority.** `trader/exit_mechs_suspend.go:14-28` — 0B, 2026-09-02. `exitMechsSuspended()` (`:35-43`) defaults **TRUE**; `EXIT_MECHS_SUSPENDED` is absent from `.env` [A]. Both mechanisms are refused at `exitMechSuspendedRefuse` (`:61`) *before* `moveStopWire` (`:49`), the single wire hop. `0b-exit-sanity.md:30-31` records the before/after: breakeven "firing (2 moves on 09-01) → **suspended**", trail "firing (8 ratchets on 09-01) → **suspended**".

**THE SUSPENSION HAS NEVER BEEN OBSERVED FIRING [A].** `grep "⏸" vl_2026-09-0{2,3,4}.log` returns 0 / 2 / 3 hits and **every one of them is a `⏸ TRANSITION OPENED/closed` regime line** — the `⏸ … SUSPENDED (0B)` refusal has never printed. `grep -c "auto-breakeven"` = 0 on all three days. `0b-exit-sanity.md:184` says this proof is **"Still owed"**, and it still is. Note the near-miss: post-0B position **591** printed **MFE 43.5 pts against a 40-pt BE trigger** — the trigger's condition was met on the 1m tape, but the BE/trail hooks sample `markPrice` on the ~60s drawdown monitor (`trader/auto_trader_risk.go:85-104`), not the bar high, so the tick may never have sampled it. **[B]**

**Bonus honesty defect inside the drift.** `trader/auto_trader_pause.go:196-201` renders `trailing=2.0×ATR14 arm=after_breakeven **(source: studio)**`. Neither the **2.0** nor the **after_breakeven** is a studio value: `trailingConfig` (`auto_trader_trailing.go:42-59`) falls back to `defaultTrailingATRMult = 2.0` (`:26`) and to `TrailArmAfterBreakeven` in the switch default (`:56`) because both fields are **absent from the stored config**. Only `trailing_atr_period: 14` is actually stored — and it equals the code default too. So the one boot line an operator would read as authoritative attributes **code constants to the owner**, and does so while the mechanism is off. **[A]**

---

## 3. D3 — MAE/MFE AND THE 1.5×ATR5m STOP FLOOR

### 3.1 Premise correction, measured

`select count(*) from trade_excursions;` → **0**. Confirmed. The table's DDL is present (`store/trade_excursion.go:154` Migrate; 30 columns, unique on `position_id`), the writer code IS in the deployed rev (`git ls-tree 70af663d trader/trade_excursion_hook.go` returns the path), and the boot line prints **`📐 excursions: logging=on rows=0 backfilled=0 unresolved=0`** on both of today's boots. So the wave is LIVE and the table is EMPTY. Corroborated at `two-day-audit.md:1074` and `expectancy-1d.md:163`.

**WHY it is empty, measured [A].** The writer's first boot line appears at **`vl_2026-09-03.log:95` — `09-03 10:28:29 [INFO] … 📐 excursions: logging=on`**. The last position in the whole table (id **591**) closed at **09-03 09:20:45 CT** — 67 min 44 s earlier. No position has opened since. That matches `two-day-audit.md:1074` ("the writer shipped 67 minutes after the last trade closed") exactly.

**A forward-looking coverage gap I did not find named anywhere [B].** `excursionOnOpen` has exactly **two** production call sites: `trader/auto_trader_decision.go:492` (AI/`system` path) and `trader/armed_executor.go:1254` (`armed_entry` path). There is **no hook on the reconcile path**. In the D3 cohort below, `source` splits **system 47 / reconcile 9 / armed_entry 5** — so ~15% of positions, including id 591 itself, would never have received an entry row even had the writer been live, and `excursionOnBarTick` skips them (`trade_excursion_hook.go:89-91`, "no entry half — nothing to update").

**And the 1A report's own status line is now stale.** `trade-excursions.md:7` reads "**Status: NOT DEPLOYED.**" The wave merged (`d4aee04a`, 2026-09-02 23:57) and went live 2026-09-03 10:28:29 CT.

### 3.2 The cohort

```sql
from trader_positions
where datetime(created_at/1000,'unixepoch','-5 hours') >= '2026-08-15 00:00:00'
  and status='CLOSED' and plan_id<>'UNRESOLVABLE' and source<>'e7_farside_test'
```

**n = 61.** (71 closed rows in the era; 9 `UNRESOLVABLE` + 3 `e7_farside_test` with 2 overlapping = 10 excluded — ids 530, 539, 545, 546, 566, 571, 572, 573, 574, 580.) Range **2026-08-19 03:22:03 → 2026-09-03 09:05:14 CT**. `mae IS NULL` = **0**, `mfe IS NULL` = **0** — every row in the cohort is measured. (Across the whole table 517 of 587 closed rows are NULL, per the E4 migration `store/position_excursion_null.go:22`; none of those fall in this cohort.) Genuine single zeros: 3 `mae=0`, 5 `mfe=0`.

**Units and sign.** `kernel/excursion_path.go:22,25` — `MAEPts`/`MFEPts` are **points, ≥ 0**, magnitudes. MNQ point value = **$2** (boot 8 `NT8 instrument_info MNQ: point_value=2`), so 30.25 pts = $60.50 on the live size-1 cap.

### 3.3 MAE / MFE percentiles — nearest-rank, the same `pct` the code uses (`store/trade_excursion_stats.go:117-131`)

| population | n | MAE p50 | MAE p80 | MAE p95 | MAE max | MFE p50 | MFE p80 | MFE p95 | MFE max |
|---|---|---|---|---|---|---|---|---|---|
| **all** | **61** | **30.25** | **49.00** | 75.00 | 100.75 | **25.75** | **66.25** | 91.00 | 156.75 |
| winners (pnl>0) | 18 | 11.25 | 22.50 | 61.50 | 61.50 | 69.25 | 91.00 | 156.75 | 156.75 |
| losers (pnl≤0) | 43 | 36.75 | 50.00 | 75.00 | 100.75 | 17.50 | 36.00 | 58.50 | 89.75 |

n ≥ 30 on the "all" row, so the headline percentiles carry a verdict. **The winner and loser splits are n=18 and n=43** — the winner arm is below the n=30 floor and is reported as numbers only, **no verdict**.

These sit right on top of the 1A wave's own backfill table (`trade-excursions.md:161-188`, n=60 measured per `:105`): NY MAE p50 31.50 / p80 61.50, LONDON 35.75 / 50.00. Same tape, same shape.

### 3.4 Converting the floor to points

**The recorded route does not reach the cohort.** `planner_read_facts` (`stop_floor_pts`, `atr5m`, `stop_floor_mlt`) holds **25 rows**, all created **2026-09-03 05:00:56 → 2026-09-04 13:32:00 UTC**. Its earliest row is 2026-09-03 00:00 CT; the cohort's LAST trade opened 2026-09-03 09:05 CT. **Overlap = exactly one trade (id 591).** So the per-trade conversion **cannot** be done from the machine record for 60 of 61 trades. There is no `stop_floor` line in any log file either (`grep -ohE "stop_floor[^ ]*" /home/hoang/vl/data/vl_*.log` → nothing; the floor rides the prompt via `class45_feeds_forward.go:199`, not the journal).

**So I reconstructed it, and validated the reconstruction.** From `bars` (MNQ, tf `5m`, 2454 rows, 2026-08-24 11:15 → 2026-09-04 08:40 CT), taking the ≤200 bars strictly before each fill and running an exact port of `market/data_indicators.go:86-117` (Wilder ATR(14), seeded from TR[1..14]) — i.e. the same computation `plannerATR5m` performs (`trader/auto_trader_planner.go:2808-2812`; `AcceptanceBars(bars,"2x5m")` is a pass-through because `acceptanceTFMinutes("2x5m")` returns 5, `kernel/scenario_facts.go:121-127`). Floor = `1.5 × atr5m`.

**Validation against the one overlapping machine record [A]:** position 591 filled 09-03 09:05:14 CT; reconstructed ATR5m = **46.29**, floor **69.44**. `planner_read_facts` id 7, written 09-03 **09:06:54 CT**, records `atr5m = 45.0976`, `stop_floor_pts = 67.646`. **Δ 2.6% at 100 seconds of separation.** The method holds.

**Convertible subset: n = 30** (entries 2026-08-25 06:04 → 2026-09-03 09:05 CT). 31 of 61 are excluded because the 5m bar cache does not reach back far enough. **n=30 is exactly at the dispatch's floor** — the verdict below is given, but it is a floor-grazing verdict, not a comfortable one.

| population | n | floor p50 | floor p80 | MAE p50 | MAE p80 | **MAE ÷ floor** p50 | p80 | p95 | max |
|---|---|---|---|---|---|---|---|---|---|
| convertible, all | **30** | 37.14 | 54.70 | 27.50 | 38.75 | **0.721** | **1.080** | 1.550 | 1.605 |
| …winners | 9 | 37.47 | 59.86 | 6.00 | 17.25 | 0.104 | 0.510 | 0.661 | **0.661** |
| …losers | 21 | 35.85 | 54.70 | 33.00 | 49.75 | 1.014 | 1.362 | 1.550 | 1.605 |

**MAE ≤ floor: 19 of 30 (63.3%).**

### 3.5 THE ANSWER

**The 1.5×ATR5m floor sits INSIDE the measured MAE distribution — at roughly its 63rd percentile (19/30, n=30).** Eleven of thirty trades pulled further against the fill than the floor would have placed a stop. It is not comfortably outside the pullback; it is in the middle-upper body of it.

**But the split is the finding, and it is the one the floor was designed for.** Every one of the 11 over-floor trades is a **loser** (ids 556, 559, 561, 562, 563, 565, 567, 583, 587, 589, 591; pnl −40.0 to −155.0). At n=9 winners the maximum MAE/floor ratio is **0.661** — no winning trade in the convertible subset came within a third of the floor, and the winner p50 is **0.104**. That is exactly the shape a stop floor wants: outside every winner's pullback, inside the median loser's (1.014). **I do not issue a verdict on the winner arm — n=9 is far below the n=30 floor.** The 63.3% headline is the only verdict here, and it is n=30 exactly.

**Three censoring caveats that must ride with that number:**

1. **The distribution was generated under a TIGHTER floor than the one it is being judged against.** The 1.5 floor deployed 2026-09-02 07:49 CT (memory/class 43; `0b-exit-sanity.md:10`). Only **3 of 61** cohort trades were opened after it; the min-SL gate at 1.0× only existed from 2026-08-26 (`kernel/min_sl.go:10`), so **22 of 61** (08-19→08-25) ran with **no floor at all**. MAE on a stopped trade is bounded by that trade's actual stop distance, so this tape's MAE is truncated at a level tighter than 1.5×ATR5m and **understates** what pullbacks would look like under the current floor. The 63.3% is therefore an **upper** bound on "inside".
2. **These are entry→exit excursions, not free-running paths.** `ComputePathExcursion` walks only `[entryMs, exitMs]` (`kernel/excursion_path.go:66-95`). A trade that stopped out has no MAE beyond its stop by construction.
3. **The counterfactual is unrecoverable.** Whether the 1.5 floor would have saved or killed any specific trade cannot be computed — the entry/stop/target of a min-SL-refused decision is not stored (`two-day-audit.md:1078`, finding D24). I do not dress this up as a P&L counterfactual.

**Where this lands against the 1A wave's own conclusion.** `trade-excursions.md:194-203`: "A **30 pt** stop sits between the p50 and p80 of MAE for every condition… A **40 pt** stop clears p80 only for `reject` (38.75)… The p95 tail runs 52–101 pts." My convertible-subset floor p50 is **37.14 pts** and p80 **54.70** — i.e. the live floor lands in the 1A report's "30–40 pt, still inside p80 for most conditions" band. **The 1A wave and this measurement agree**, and the 1A wave's table is still empty, so its distribution came from the same `trader_positions.mae/mfe` backfill I used, not from `trade_excursions`.

---

## 4. NINE DRIFTS / DEFECTS FOUND

1. **[A] E1 breakeven +40 [O] ON in the strategy, OFF live.** `ai_config.risk_control.breakeven_enabled=true, breakeven_trigger_points=40` vs boot 8 `BE=off`. Owner: ruling (the 0B suspension needs an owner re-ratification or the strategy field needs to be turned off so the two records agree).
2. **[A] E2 trailing [O] ON in the strategy, OFF live.** `trailing_enabled=true, trailing_atr_period=14` vs boot 8 `trail=off`. Same owner.
3. **[A] Two boot lines in the same boot disagree.** `🛑 exits: … BE=off · trail=off` (`trader/exit_mechs_suspend.go:82`) vs `🧾 ledger boot: … trailing=2.0×ATR14 arm=after_breakeven (source: studio)` (`trader/auto_trader_pause.go:196-206` — it never consults `exitMechsSuspended()`). Already filed as two-day-audit D44 (`:930`); still live at boot 8. Owner: **code**.
4. **[A] "(source: studio)" is false for 2 of its 3 fields.** `trailing_atr_mult` and `trailing_arm` are absent from the stored config; 2.0 comes from `defaultTrailingATRMult` (`auto_trader_trailing.go:26`) and `after_breakeven` from the switch default (`:56`). Owner: **code**.
5. **[A] `re-arm-after-sweep=on` is a hardcoded literal in a boot line.** `trader/exit_mechs_suspend.go:96` passes `true` positionally; there is no resolver and no knob. Violates the CLAUDE.md canon "**Boot lines are READ, never literal**" (checklist 45, 49). The value is factually correct today (the class-33 boot sweep is unconditional, `armed_executor.go:201`), which is precisely why nobody has noticed. Owner: **code**.
6. **[A] The prompt states a floor the gate does not enforce.** `kernel/planner_prompt.go:733` — "the stop distance must be ≥ **1.0×** the current 5m ATR (… a 10-point stop when ATR5m is ~16 is an instant refuse)" while the resolved floor is **1.5** and `armed_executor.go:1361` refuses against 1.5. The class-45 block (`class45_feeds_forward.go:199`) correctly renders the resolved 1.5 in the SAME prompt, so the model is handed both numbers. Boot 8's `prompt/validator contract: 19 restrictions, all stated in prompt (class 38 guard)` does not catch this — the guard checks that restrictions are *stated*, not that the *numbers inside them* match the resolvers. This is checklist class 38/45 recurring at one level deeper. Owner: **prompt** (+ a guard extension).
7. **[A] Two doc comments still advertise the retired 1.0.** `kernel/min_sl.go:43` ("default 1.0; 0 = gate off") and `kernel/engine_position.go:206` ("env, default 1.0"), both beside a const of 1.5. Cosmetic but it is what a reader greps. Owner: **code**.
8. **[A] The [R] citation behind the 1.5 floor cannot be located.** `kernel/min_sl.go:21-23` cites "Round-7 research tests the day-trade range at 1.5–2.5×ATR"; `trader/exit_mechs_suspend.go:18-19` cites "Round-7 research ranks ATR/Chandelier trails in the worst group of 15 exit families across 567,000 backtests". **`2026-09-01-full-system-audit.md:403` states flatly: "D3: a research source stating '1.5–2.5×ATR stop floor' — not found in the repo (nearest: v5 build plan `stop_atr_mult: 2.5`); would verify: the owner naming the source."** My own sweep confirms it: `grep -rl "567,000\|15 exit families\|Chandelier" docs/` hits only `AUDIT-CHECKLIST.md` (which restates the same unsourced claim, `:625-626`) and `2026-09-03-trade-excursions.md:107`. **Neither the stop floor nor the BE/trail suspension may be labelled [R].** Both are correctly **[O] + [T]** — and the [T] half is real (AUDIT-CHECKLIST.md:627-630). Owner: **ruling** (the owner naming the source, or the code comments dropping the "Round-7 research" framing).
9. **[A] `trade-excursions.md:7` says "Status: NOT DEPLOYED"** — the wave merged `d4aee04a` and has been live since 2026-09-03 10:28:29 CT with 0 rows. Owner: **ruling/docs**.

---

## 5. COMMANDS (all read-only, reproducible)

```bash
# cohort
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
 select count(*) from trader_positions
 where datetime(created_at/1000,'unixepoch','-5 hours') >= '2026-08-15 00:00:00'
   and status='CLOSED' and plan_id<>'UNRESOLVABLE' and source<>'e7_farside_test';"   # 61

# empty excursion table
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "select count(*) from trade_excursions;"  # 0

# the recorded floor series (25 rows, 09-03/09-04 only)
sqlite3 -header -column "file:/home/hoang/vl/data/data.db?mode=ro" \
 "select id, datetime(created_at,'-5 hours') ct, session, atr5m, stop_floor_pts, stop_floor_mlt
    from planner_read_facts order by id;"

# env is absent -> every E resolver falls to its default
grep -E "EXIT_MECHS|MIN_SL|ARM_STOP|STAGE_A" /home/hoang/vl/.env    # no match

# the suspension has never fired
grep -h "⏸" /home/hoang/vl/data/vl_2026-09-0[34].log | sed 's/.*⏸/⏸/' | sort -u   # TRANSITION lines only
grep -c "auto-breakeven" /home/hoang/vl/data/vl_2026-09-0[234].log                 # 0 0 0

# writer first went live
grep -m1 -n "excursions: logging=on" /home/hoang/vl/data/vl_2026-09-03.log   # :95  09-03 10:28:29
```

Reconstruction script (Wilder ATR(14) on 200 prior 5m bars, exact port of `market/data_indicators.go:86-117`):
`/tmp/claude-1000/-home-hoang-vl/51524a30-37b9-4d23-aaa2-8ad3d23f4ea7/scratchpad/e_mae.py`

## 6. FILES WRITTEN (worktree only)

- `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/E-d3-mae-mfe-per-trade.csv` — 61 rows: `id, side, entry_px, entry_ct, mae, mfe, session, pnl, grade, source, bars_avail, atr5m, floor_pts, mae_over_floor`
- `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/E-d3-summary-percentiles.csv` — 18 rows, every percentile above with its n