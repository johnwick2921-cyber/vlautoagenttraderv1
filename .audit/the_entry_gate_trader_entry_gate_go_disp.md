names rewritten to vl on 2026-09-30 (VL rename)
## 0. Provenance

- **Source tree:** `/home/hoang/vl-conform`, base `492d2067` (= dev tip at accept; my worktree HEAD `fb50903f`, a claim commit; peers have since added `e1c81258`/`4751fc84`/`e2b2b8e1`).
- **Running binary:** rev `70af663d`, PID 878451, booted **2026-09-04 08:30:11 CT**. Every RESOLVED value below is READ from that process's boot lines in `/home/hoang/vl/data/vl_2026-09-04.log` or traced through the resolver call path. `/api/config/resolved` and `/api/risk/gate-blocks` both require an `Authorization` header this session does not have — **neither was used**.
- **Clock at measurement:** 2026-09-04 08:52 CT. The NY session is ~22 min old; the measurements below are of a live, in-flight session.
- **DB:** read-only, `file:/home/hoang/vl/data/data.db?mode=ro`.

`git log -1` for every report cited:

| report | pinning commit |
|---|---|
| `docs/superpowers/reports/2026-09-02-belief-census.md` | `ee64a494 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 — every market belief labeled [R]/[X]/[T]/[I]/[O] with live effect + demotion queue (read-only)` |
| `docs/superpowers/reports/2026-09-04-two-day-audit.md` | `f3c640c3 2026-09-04 07:26:52 -0500 docs(two-day audit D3): why the blindness went unalerted — a note, not a build` |
| `docs/superpowers/reports/2026-09-02-class48-entry-gate.md` | `95767c7c 2026-09-02 17:55:15 -0500 fix(class48): ONE EntryGate for BOTH order paths — the decision path bypassed the arm-seam gates` |
| `docs/superpowers/reports/2026-09-03-invalidation-wired.md` | `f1a4b0e2 2026-09-03 14:51:17 -0500 docs: checklist class 60 — a gate signal that is time-of-day dependent, plus the standing check` |
| `docs/superpowers/reports/2026-09-02-void-parity-inputs.md` | `60f214d9 2026-09-02 22:38:48 -0500 fix(void-parity): the scope VALUE is the session day — the validator narrows to match, and the render is compact` |
| `docs/superpowers/reports/2026-09-02-no-trade-band.md` | `46efa9df 2026-09-03 09:45:46 -0500 docs: rider part 5 — wake cutoffs enforce, and the steady-state effect measured` |
| `docs/superpowers/reports/2026-09-03-expectancy-1d.md` | `38a63a9b 2026-09-03 15:26:02 -0500 docs(1D): report — the model, RED/GREEN, the live table, and two surprises` |
| `docs/superpowers/reports/2026-09-03-mc-drawdown.md` | `77e1cdfc 2026-09-03 00:39:25 -0500 docs(1E): Monte Carlo drawdown results — n=64, expectancy indistinguishable from zero (CI -31 to +18), ~1810 trades needed` |
| `docs/superpowers/reports/2026-09-02-live-bias-replay.md` | `53498adb 2026-09-02 21:02:58 -0500 docs(live-bias-replay): results — 84 session days, 252 rows; every leg NOT USABLE by D6 …` |
| `docs/superpowers/reports/2026-09-02-bias-calibration.md` | `2deab3c8 2026-09-02 20:53:20 -0500 docs(bias-calibration): results + CSVs — all three signals … NOT USABLE on holdout …` |
| `docs/superpowers/reports/2026-09-02-level-kind-replay.md` | `3961f873 2026-09-02 19:03:10 -0500 docs(level-replay part2): 1h variant results …` |
| `docs/superpowers/reports/2026-09-02-detector-redesign.md` | `0465a10b 2026-09-02 07:58:10 -0500 docs(lane4): A8 SATISFIED — raw URL returns 200 after the push; retract the private-repo claim` |
| `docs/superpowers/reports/2026-09-03-trade-excursions.md` | `0c1a808c 2026-09-03 00:05:11 -0500 docs(excursions): checklist class 54 + the wave report with the E6 distribution` |
| `docs/superpowers/reports/2026-08-30-knob-census.md` | `741bfc2a 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX …` |
| `docs/superpowers/research/INDEX.md` | `4e8e7e1a 2026-09-03 19:37:14 -0500 docs(index): the stranded-branch sweep — 25 docs-only merged and indexed unclassified, 11 name-only-docs listed as not merged` |

---

## 1. How the resolved values were resolved (A11)

Boot-8 lines quoted verbatim from `/home/hoang/vl/data/vl_2026-09-04.log` (08:30:11 CT block):

```
BOOT INTEGRITY OK — rev 70af663dcb6f · built 2026-09-04T13:16:34Z · expected 70af663d · goldens PASS
🛑 min-sl guard: atr_mult=1.5 level_clearance=2tick(s)
⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off arm_rr=2.0 (gate-at-arm only; market-entry floor 2.0 unchanged)
🛡 arm gate: invalidation-wired=on · armed-under surfaces=on — the evaluator's own ≈invalidated verdict REFUSES an arm …
🚫 no-chase: max_dist=1.00×ATR max_run=1.5×ATR5m(stop floor) mode=warn counters=on [I] PROVISIONAL
📜 void scope: session-day window · 1m×2000 · one resolver for prompt AND validator (parity)
no-trade band: first_n=5m lunch=12:00–13:30 (source=code-constant, shared by gate+grader+card) · T1 taken from the enforcing gate at plan time …
conditions: live [acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim] · shadow [breakout_retest, fvg_entry]
```

Resolver traces (the values are NOT file defaults presented as live):

- **`plan_mode` = `strict`** [A]. `at.planModeFor(session)` → `store.DayPlanConfig.PlanModeFor` (`store/strategy.go:1399-1402`) → `store.ResolvePlanMode` (`store/resolve_source.go:47-55`). Live DB: `strategies.id = a5b7662e-…` (the only running trader `8d5c8af5_…`, `is_running=1`, account `Sim101`) has `day_plan.plan_mode = "strict"` and **no `plan_mode` in any of its three session overrides** (`sessions[]` carries only `replan_cap`/`acceptance_rule`/`min_grade`/`max_trades`/`enable`). So strict resolves for **ASIA, LONDON and NY alike**, source `strategy value`. Row `updated_at = 2026-09-01 13:13:06 UTC`, i.e. 2 days before boot 8 — the in-memory cached config equals the row.
- **`MinRR` = `2.0`** [A]. `at.armMinRRFor(nil)` (`trader/armed_executor.go:78-83`) → `resolvedMinRR` (`:68-73`) → `store.ResolveMinRiskReward` (`store/resolve_source.go:29-34`). The saved config carries `ai_config.risk_control.min_risk_reward_ratio = 2` → source **`saved value`**, so the `SafeDefaultMinRiskReward = 3.0` branch (`store/strategy.go:76`) is NOT taken. Boot line prints both args as `2.0`, which is the proof the resolver and the raw field agree.
- **`MinSLMult` = `1.5`** [A]. `kernel.MinSLATRMult()` (`kernel/min_sl.go:44-51`) → `MIN_SL_ATR_MULT` is **ABSENT from `/home/hoang/vl/.env`** (checked) → `MinSLATRMultDefault = 1.5` (`kernel/min_sl.go:31`).
- **no-chase 1.00 / 1.5** [A]. `NOCHASE_MAX_DIST_ATR` and `NOCHASE_MAX_RUN_PTS` both unset → `no_chase.go:77` returns `1.0`; run ceiling falls through to `MinSLMult` (`no_chase.go:89-95`).

---

## 2. Every EntryGate leg — the table

`EntryGate` is one pure function, `trader/entry_gate.go:140-297`. It has exactly **two production callers** — the two builders — and the legs inherit them:

- `entryGateForArm` (`trader/entry_gate.go:328-380`) ← **`trader/armed_executor.go:510`**
- `entryGateForDecision` (`trader/entry_gate.go:388-455`) ← **`trader/auto_trader_orders.go:333`**

Everything else calling `EntryGate(` is a test (`entry_gate_test.go`, `invalidation_leg_test.go`, `no_chase_test.go`, `one_live_position_test.go`, `plan_mode_strict_test.go`).

| # | leg | file:line | RESOLVED at boot 8 | label | grounding (report:line) | effect | CONFORMS? | prod callers |
|---|---|---|---|---|---|---|---|---|
| 0 | plan_mode **strict** | `trader/entry_gate.go:160-174` | `plan_mode=strict` (strategy value, no session override) | **[O]** | `VL-DAYPLAN-FULL-SPEC.md:49`; **not in the belief census** — post-dates `ee64a494` | **REJECT** (hard) | **NO** — spec:49 *"strict (Go-verified trigger required)… plan restricts, never compels"*; live: **every decision-path market entry refused regardless of citation** (`:161-163`) | 2 (refusing branch only via `auto_trader_orders.go:333`) |
| 1 | plan bias (direction mode) | `trader/entry_gate.go:179-184` | **INERT** — fires only when mode == `"direction"`; resolved mode is `strict` | [O] | `VL-DAYPLAN-FULL-SPEC.md:49` | REJECT when direction; dead-by-config now | yes (text matches spec) | 2 — `armed_executor.go:510`, `auto_trader_orders.go:333` |
| 2 | class-48 direction mismatch | `trader/entry_gate.go:190-197` | ACTIVE on both paths whenever a cited scenario has an authored direction | [O] | `2026-09-02-class48-entry-gate.md:206` (590 was *not* a direction mismatch at v5 — the shadow leg is what refuses it) | REJECT | yes | 2 — `armed_executor.go:510`, `auto_trader_orders.go:333` |
| 3 | **invalidation** (arm only) | `trader/entry_gate.go:205-228` | **ON**; resolver wired at `entry_gate.go:358`, left `nil` on the decision path | [O] | `2026-09-03-invalidation-wired.md:19-25` (E1 — the 12 minutes, −$140) | REJECT on arm; **fail-open + WARN** when the evaluator can't run | yes | **1** — `armed_executor.go:510` |
| 4 | shadow map 0C | `trader/entry_gate.go:233-235` | shadow list `[breakout_retest, fvg_entry]` (boot line, read; `conditionShadowedFor` `armed_executor.go:30-41`) | [O] | `2026-09-02-class48-entry-gate.md:205-206` (589 + 590 both traded `breakout_retest`) | REJECT | yes | 2 — `armed_executor.go:510`, `auto_trader_orders.go:333` |
| 5 | **R:R at the execution price** | `trader/entry_gate.go:242-259` | floor **2.0** (saved value); fallback 3.0 only when `MinRR ≤ 0` (`:253-255`) | **[T]** | `2026-09-02-belief-census.md:48` — B8 "Arm R:R ≥ 2.0 at arm time", **[T] n=18 +$994** | REJECT | **NO** — research/spec floor is **3.0**; resolved live **2.0** on BOTH paths | 2 + the arm-chain twin `armed_executor.go:1352-1354` |
| 6 | min-SL ×ATR5m | `trader/entry_gate.go:262-270` | `MinSLMult = 1.5` (env absent → `kernel/min_sl.go:31`) | **[O]** (partial [T]) | `2026-09-02-belief-census.md:46` (B6, **[I/C]**) + `:129` (rank-1 of the demotion queue) | REJECT | **NO** — census records **1.0**; resolved live **1.5** | 2 — `armed_executor.go:510`, `auto_trader_orders.go:333` |
| — | **2-tick level clearance** (the other half of B6) | `kernel/min_sl.go:36-40` | 2 ticks = 0.50 pt (MNQ tick 0.25, boot `instrument_info`) | [O] | `2026-09-02-belief-census.md:46` (B6 states "+ 2-tick clearance") | REJECT — **but NOT inside EntryGate** | **NO (scope)** | 3 — `kernel/engine_position.go:241`, `trader/armed_executor.go:392`, `auto_trader_dayplan.go:58` (boot print) |
| 7 | one_open_position | `trader/entry_gate.go:277-283` | **ON, hardcoded, no knob**; currently **INERT** — 0 open rows in `trader_positions` | [O] | `2026-09-04-two-day-audit.md:615` | REJECT (exit legs exempt via `IsExitLeg`) | yes | 2 — `armed_executor.go:510`, `auto_trader_orders.go:333` |
| — | no-chase callback | `trader/entry_gate.go:288-294` + `trader/no_chase.go:100-131` | `max_dist=1.00×ATR max_run=1.5×ATR5m mode=warn counters=on` | **[I]** | self-declared `[I] PROVISIONAL` in its own boot line (`no_chase.go:146-147`); **no report supports the numbers** | **WARN-only — refuses nothing** | yes (as WARN) | 2 — both builders |

---

## 3. The three two-day-audit claims — RE-VERIFIED at boot 8

### 3.1 "the decision-path EntryGate refusal writes NO log line and NO counter"

> `2026-09-04-two-day-audit.md:919` — **D32** | **The decision-path entry-gate refusal is invisible** — `entryGateDecisionTelemetry` (`trader/entry_gate.go:477-486`) writes no log line and no counter, only `actionRecord.Error`. 19 refusals in the window produced **0** log lines and **0** `log_events` rows.
> and `:539` — | **decision** | `entryGateDecisionTelemetry` (`trader/entry_gate.go:477-486`) | ❌ **nothing** | ❌ **nothing** — only `actionRecord.Error` |

**Verdict: HALF TRUE, and the counter half is WRONG. [A]**

`entryGateDecisionTelemetry` at boot 8 is:

```go
func entryGateDecisionTelemetry(at *AutoTrader, actionRecord *store.DecisionAction, reason string) {
	telemetry.IncGateBlock(at.id, "entry_gate")      // ← entry_gate.go:478
	actionRecord.Success = false
	if !strings.HasPrefix(reason, "entry_gate:") { reason = "entry_gate: " + reason }
	actionRecord.Error = reason
}
```

- **No log line — CONFIRMED.** The function body contains no `logWarnf`/`logInfof`. `recordEntryGateRefusal` (`entry_gate.go:461-473`), which *does* log `🚦 entry-gate REFUSED …`, has exactly **one** production caller and it is arm-only: `trader/armed_executor.go:526`. **[A]**
- **No counter — WRONG.** `telemetry.IncGateBlock(at.id, "entry_gate")` has been on line 478 since `95767c7c` (2026-09-02 12:34), i.e. it predates the audit. It increments the B6 per-trader/per-gate table (`telemetry/gate_blocks.go:38-45`). **[A]**
- The audit's *operational* conclusion nevertheless survives, because that counter is **in-memory and ephemeral** by design (`telemetry/gate_blocks.go:17-18`: *"intentionally in-memory and ephemeral (a diagnostics tally, not a ledger): the table resets at the 17:00 CT CME session-day rollover"*). It is readable only through `/api/risk/gate-blocks` (auth-blocked here) or the one rollover journal line `📊 gate-block summary …` (`trader/auto_trader_loop.go:251-253`), which is INFO and therefore journald-suppressed and absent from `log_events`. **Nothing durable is written.** [A]
- The **arm** path, by contrast, writes a *persisted* counter — confirmed live in `system_config`: `arm_refusals_0b:8d5c8af5_…:2026-09-04:LONDON:entry_gate:invalidated = 1`.

**Corrected statement:** the decision-path refusal writes **no log line and no durable counter**; it does bump an ephemeral in-memory `entry_gate` gate-block tally that no reachable surface exposed in this session.

### 3.2 "plan_mode=strict refuses EVERY decision-path market entry regardless of citation"

> `2026-09-04-two-day-audit.md:630` — **[A]** `plan_mode=strict` + EntryGate leg 0 refuses EVERY decision-path market entry, regardless of citation …
> `:920` — **D33** … Since commit `c8c90dcc` (09-03 10:43 CT) the decision path is closed to market entries and nothing says so.

**Verdict: STILL TRUE at boot 8. [A]** Code unchanged:

```go
if in.PlanMode == "strict" {
    if in.Path != "arm" {
        return fmt.Sprintf("entry_gate: refused: strict — plan_mode=strict executes plan scenarios on the ARM path only, and this is a %s-path market entry", in.Path), true
    }
    …
}
```
`trader/entry_gate.go:160-163`. `entryGateForDecision` hardcodes `Path: "decision"` (`entry_gate.go:419`), so the first `if` always fires for a market entry. Resolved `plan_mode` is `strict` for every session (§1).

**And it is worse than one gate.** There are **two** strict enforcements on the decision path with **opposite semantics**, and they run in the order that makes the looser one pointless:

| order | seam | strict semantics |
|---|---|---|
| 1st — `auto_trader_orders.go:295` | `planModeBlocked` (`auto_trader_planconfig.go:218-223`) | a **matched** citation (`kernel.ClassifyCitation`) **PASSES** |
| 2nd — `auto_trader_orders.go:333` | `EntryGate` leg 0 | the decision path is refused **outright**, matched or not |

So a correctly-cited, correctly-sided AI market entry is waved through the gate that was designed to judge it and then killed by the gate that never looks at the citation. **[A]**

### 3.3 "armGateVerdict has ZERO production callers"

> `2026-09-04-two-day-audit.md:921` — **D34** | **A29 DEAD GATE — `armGateVerdict` … has ZERO production callers**; all 8 call sites are in `armed_executor_test.go`.

**Verdict: STILL TRUE at boot 8. [A]** `armGateVerdict` now lives at `trader/armed_executor.go:1303-1310` (the audit cited `:1268`; the file has shifted). Its 8 call sites are all `trader/armed_executor_test.go:77,82,87,94,100,180,185,189`. Production calls the sibling `armGateVerdictFor` at **`trader/armed_executor.go:430`**. **DEAD — say it loudly.**

### 3.4 A correction I owe the audit — leg 3 has now fired

> `2026-09-04-two-day-audit.md:577` — **[A] Fourteen wired gate legs never fired once**: EntryGate legs 1, 2, 3, 4, 7 (direction, class-48 mismatch, invalidation, shadow, one_open_position) …

That was true for 09-02→09-03. It is **no longer true**. On **2026-09-04 02:00:46 CT** EntryGate **leg 3 fired in production for the first time**:

```
09-04 02:00:46 [WARN] 🚦 entry-gate REFUSED arm LONDON: entry_gate: scenario S2 invalidated at
2026-09-04 02:00 CT (accepted through 29579.50) — price accepted through the level against the
trade — it flipped roles · refusals this session: 1
```
(`/home/hoang/vl/data/vl_2026-09-03.log:32444` — the log file is named by the process-start calendar day, so the 09-04 LONDON session lands there.) Persisted counter: `arm_refusals_0b:…:2026-09-04:LONDON:entry_gate:invalidated = 1`. **[A]** Legs 1, 2, 4, 7 remain at zero.

---

## 4. Two more findings the enumeration turned up

### 4.1 The no-trade windows gate the DECISION path only — the arm path is not covered

`sessionEntryBlocked` (`trader/auto_trader_session.go:27-48`) is the only thing that enforces first-5m, lunch and T1, via `sessionGateDecision` (`:100-129`, which calls `kernel.InFirstNoTradeMinutes` at `:139` and `kernel.InLunchNoTrade` at `:120`). It has **exactly one production caller: `trader/auto_trader_orders.go:281`** — the decision path. **[A]**

The arm loop `maybeManageArmedOrders` (`trader/armed_executor.go:188-`) gates on `dayPlanEnabled`, exchange, the boot sweep, an **active session**, and plan lifecycle — and on nothing else. `grep` for `InLunchNoTrade|InFirstNoTradeMinutes|sessionEntryBlocked` across `trader/armed_executor.go` returns **zero hits**. So a resting armed limit can be **placed and filled inside the lunch window and inside the first 5 minutes of a session**. **[A]**

T1 red-news is the partial exception: `enforceT1ForceFlatAt` cancels working arms **first**, before the open-position check (`trader/auto_trader_clock.go:688-698`). But it returns `n+unacked > 0` when flat (`:701`), so on a cycle where no arm exists it returns `false`, the loop continues to `maybeManageArmedOrders` (`auto_trader_loop.go:432`), and a **new** arm can be placed inside the window — living at most one 2-minute cycle before the next tick cancels it. **[B]** (inference from the control flow; not observed on tape in this window.)

### 4.2 A third order path bypasses EntryGate entirely

The class-48 header itself says *"the agent chat path ran almost none"* (`trader/entry_gate.go:20-21`). At boot 8 it runs **none**: `agent/trade.go:193` and `:199` call `underlyingTrader.OpenLong/OpenShort` directly from `executeTrade`, reached from the chat trade-confirm at `agent/trade.go:488`. No `EntryGate`, no R:R floor, no min-SL, no shadow map, no one-open-position. **[A]** (`api/handler_debug.go:176` → `DebugPlaceTestTrade` is the other bypass, but it is an auth-`protected` route whose own schema string says *"bypasses AI + risk gate"* — an owner escape hatch, not a live loop.)

---

## 5. DISPATCH D11 — the strict interaction, measured

**Question: with `plan_mode=strict` live, CAN a long plan trade today?**

### Route (i) — the decision path: **CLOSED, unconditionally.**
`entryGateForDecision` stamps `Path: "decision"` (`entry_gate.go:419`); leg 0's first branch (`entry_gate.go:161-163`) refuses on `in.Path != "arm"` before any other test. Every AI market entry — long or short, cited or not, correctly sided or not — is refused. **[A]**

Measured since boot 8: **8 decision cycles** (`decision_records` ids 37649–37656, 08:30:21 → 08:44:19 CT), **every one `action=wait`**, `execution_log = ["AI call duration: … ms","✓ MNQ wait succeeded"]`. **Zero `open_long`/`open_short` were emitted, so leg 0 had nothing to refuse yet today. n=8.** [A]

### Route (ii) — the arm path: **OPEN in principle, refused in fact.**

**The first directional plan since boot 8** — `plans` table, `2026-09-04:NY` **v2**, written **2026-09-04 08:44:46 CT** (`created_at 2026-09-04 13:44:46.688576529+00:00`), `lifecycle=active`, `trigger_reason=owner_reset`. (The preceding row, NY v1 at 08:11:45 CT, is `no_trade`/`planner_fail_closed` and pre-dates the boot.)

| field | value |
|---|---|
| `bias.direction` | **long** |
| `bias.conviction` | medium |
| `day_type` | **balance** |

| scenario | direction | condition | quality | `arm.enabled` | entry / stop / target | R:R |
|---|---|---|---|---|---|---|
| **S1** | **long** | `sweep_reclaim` | A | **true** | 29611.25 / 29481.50 / 29720.00 | **0.84** (108.75 ÷ 129.75) |
| S2 | short | `sweep_reclaim` | B | true | 29720.00 / 29770.00 / 29601.00 | 2.38 (119 ÷ 50) |

At **08:44:46 CT** the planner's own feasibility warning called both shots (`vl_2026-09-04.log:4949-4950`):

```
⚔️ arm feasibility: S1 arm R:R 0.84 below min_risk_reward_ratio 2.00 (Studio) — the gate-at-arm
chain will refuse it every cycle (target/stop infeasible)
⚔️ arm feasibility: S2 arm stop 29770.00 too close (50.00 < 52.92 = 1.5×ATR5m) — min-SL gate will refuse it
```

At **08:46:09–08:46:10 CT** the gates ran for real (`vl_2026-09-04.log:5039-5045`):

```
5039  ⚔️ arm REFUSED NY S1 leg 1: R:R 0.84 below arm min 2.00 (studio min_risk_reward_ratio)
      · rr refusals this session: 1
5040  🛑 arm stop NY S2 leg 1 short: stop 29773.61 (authored 29770.00 WIDENED) · anchor none
      (stop_unanchored) · atr_floor 29773.61 (1.5×ATR5m 35.74) · bound=atr_floor
5042  🚫 no-chase: arm S2 has no recorded touch — run stored NULL, judged on dist alone (0.00×ATR)
5043  📏 arm far: S2 short entry 29720.00 is 125.50 pts / 3.5×ATR5m from price 29594.50
      (counted, not refused)
5045  ⚔️ armed NY S2 leg 1 short limit 29720.00 SL 29773.61 TP 29601.00
```

**Every `armed_orders` row created since boot 8 — there is exactly one:**

| id | session | scenario | side | entry | stop | target | state | armed_under_version | condition | created_at |
|---|---|---|---|---|---|---|---|---|---|---|
| 38 | NY | S2 | **SHORT** | 29720.00 | 29773.83 | 29601.00 | `armed` | 2 | `sweep_reclaim` | 2026-09-04 08:46:09.88 CT |

**Long arms since boot 8: 0. Open positions right now: 0** (last closed row is id 591, 2026-09-03). **[A]**

### The plain answer

**No. A long could not have traded today, and none did.** [A]

Both routes are shut, for two independent reasons:
1. **Decision path — shut by policy.** Leg 0 refuses every decision-path market entry while `plan_mode=strict`, and strict is resolved for every session because the strategy-level value is `strict` with no session override.
2. **Arm path — shut by arithmetic.** The plan authored exactly one long scenario with `arm.enabled=true`, and its **R:R of 0.84 is 58% below the 2.0 floor**. It was refused at `armed_executor.go:1352-1354`, one leg *before* EntryGate even saw it. The short scenario, by contrast, armed — and only because the 0B stop composition **widened** its authored 29770.00 stop to 29773.61 to clear the 1.5×ATR5m floor, taking the composed R:R to 2.22 and letting leg 6 pass. **The stop-widening machinery rescued the short; nothing could rescue the long, because widening a long's stop makes its R:R worse, not better.** [A]

Note the interaction that makes this systematic rather than a one-day accident: the min-SL floor (1.5×ATR5m, ~53 pts today) and the R:R floor (2.0) together demand a target ≥ ~106 pts from entry on any arm whose stop is at the ATR floor. Today's long asked for 108.75 of reward against 129.75 of risk and never had a chance. That is a *joint* feasibility constraint on the planner that neither knob states on its own.

### A required abstention (A24)
Both of today's scenarios cite `sweep_reclaim`. On our own tape that condition is **`n=6`, 1W/5L/0F, Σ −207.00, Wilson [−86.28, +17.28], verdict NOT ENOUGH DATA** (`2026-09-03-expectancy-1d.md:150`), against the pre-registered floor `MinN = 30` (`expectancy/model.go:27`). **No expectancy verdict may be issued for either scenario.** The wider frame is the same: expectancy is statistically indistinguishable from zero, CI −$31 to +$18, ~1,810 trades needed (`2026-09-03-mc-drawdown.md:230-232`).

---

## 6. Commands / queries used

```bash
# resolved values, read from the running process
grep -nE "08:30:1[0-9]|08:30:2[0-9]" /home/hoang/vl/data/vl_2026-09-04.log
grep -nE "arm REFUSED|entry-gate REFUSED|armed cancel|⚔️|🚦" /home/hoang/vl/data/vl_2026-09-04.log
grep -nE "🚦 entry-gate REFUSED" /home/hoang/vl/data/vl_2026-09-03.log     # LONDON 09-04 lands here

# env overrides — all ABSENT
for k in MIN_SL_ATR_MULT ARM_MIN_RR PLAN_MODE; do grep -q "^$k=" .env && echo "$k set" || echo "$k ABSENT"; done

# the live strategy
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "SELECT config FROM strategies WHERE id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8';"

# D11 measurement
sqlite3 -header "file:/home/hoang/vl/data/data.db?mode=ro" \
  "SELECT plan_id,version,session,lifecycle,trigger_reason,created_at FROM plans ORDER BY created_at DESC LIMIT 6;"
sqlite3 -header "file:/home/hoang/vl/data/data.db?mode=ro" \
  "SELECT id,session,scenario,side,entry_px,stop_px,target_px,state,armed_under_version,condition,created_at
   FROM armed_orders WHERE created_at >= '2026-09-04 08:30:11' ORDER BY id;"
sqlite3 -header "file:/home/hoang/vl/data/data.db?mode=ro" \
  "SELECT id, datetime(timestamp,'-5 hours') ct, json_extract(decision_json,'\$.action') action, execution_log
   FROM decision_records WHERE datetime(timestamp) >= '2026-09-04 13:30:11' ORDER BY id;"
sqlite3 -header "file:/home/hoang/vl/data/data.db?mode=ro" \
  "SELECT key,value FROM system_config WHERE key LIKE 'arm_refusals%' OR key LIKE 'nochase%';"

# callers / dead code
grep -rn "EntryGate(\|armGateVerdict\|entryGateForArm\|entryGateForDecision" --include=*.go .
grep -rn "sessionEntryBlocked()\|InLunchNoTrade\|InFirstNoTradeMinutes" --include=*.go . | grep -v _test
```

**A caveat on clock seams noticed in passing (not my subsystem, reporting it anyway):** `plans.created_at` is stored in **UTC** (`…+00:00`) while `armed_orders.created_at` is stored in **CT** (`…-05:00`). Any query that joins or windows the two tables on raw string time will be 5 hours wrong. **[A]**
