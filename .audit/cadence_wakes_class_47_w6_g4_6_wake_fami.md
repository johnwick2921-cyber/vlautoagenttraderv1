names rewritten to vl on 2026-09-30 (VL rename)
> **Note on this deliverable.** This is subagent report content returned as text, per the dispatch's OUTPUT contract. Two data fragments were written to `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/` (CSVs only — no report .md).

# CADENCE / WAKES — research-conformance re-check

Source tree `/home/hoang/vl-conform` @ `fb50903f` (claim commit on `docs/research-conformance-0904`, base = dev tip `492d2067`). Running binary rev `70af663d`, PID 878451, booted 2026-09-04 08:30:09 CT. Measurement window closes 08:49:25 CT. All DB reads via `sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro"`. **[A]** unless marked.

## `git log -1` for every report cited

| Report | Pinning commit |
|---|---|
| `docs/superpowers/reports/2026-09-02-belief-census.md` | `ee64a494 2026-09-02 08:50:38 -0500` |
| `docs/superpowers/reports/2026-09-02-level-event-wake-audit.md` | `a5a53bec 2026-09-02 08:27:52 -0500` |
| `docs/superpowers/reports/2026-09-03-wake-predicate.md` | `586261ed 2026-09-03 21:36:39 -0500` |
| `docs/superpowers/reports/2026-08-30-knob-census.md` | `741bfc2a 2026-09-01 07:58:16 -0500` |
| `docs/superpowers/reports/2026-09-04-two-day-audit.md` | `f3c640c3 2026-09-04 07:26:52 -0500` |
| `docs/superpowers/reports/2026-09-04-two-day-audit-data/verify-class47-mss-wake-ungoverned.md` | `24685b70 2026-09-03 23:51:00 -0500` |
| `docs/superpowers/research/INDEX.md` | `4e8e7e1a 2026-09-03 19:37:14 -0500` |

INDEX.md has **no wake entry** (`grep -ni wake` → one unrelated line, `INDEX.md:49`, the 08-28 missed-200pt report). The wake corpus lives entirely in `reports/`.

## The full boot line, read from the log (not the constant)

`/home/hoang/vl/data/vl_2026-09-04.log:3853`:

```
09-04 08:30:11 [INFO] vl/main.go:340 ⏱ wakes: cutoff=25m(enforce) cooldown=30m(enforce,
fast-market≥1.5×ATR exempt) cross-session=on stale-arm-expiry=on (class 47) — cutoffs govern
LEVEL_EVENT/structure_mss wakes ONLY; scheduled reads, death re-plans and owner resets are
untouched; the cutoff is NOT exempted by a fast market
```

Producer: `trader/class47_wake_cadence.go:220-225`, called from `main.go:340` (1 production caller).

---

## 1. THE RULE TABLE

| # | Rule | file:line | Resolved value NOW | Label | Grounding (report:line) | Live effect | CONFORMS? | Production callers |
|---|---|---|---|---|---|---|---|---|
| W1 | scheduled read ASIA | `kernel/session_registry.go:90` + live `system_config.session_registry` | `read_ct=16:30 CT`, window 17:00→02:00, flat 02:00 | [O] | knob-census.md:100 (`last_entry_ct/eod_flat_ct` [O]) | gate → authors v1 | yes (16:30 = 16:30) | 1 — `trader/auto_trader_planner.go:206` (`inSessionReadWindow`) |
| W2 | scheduled read LONDON | `kernel/session_registry.go:99` + live registry | `read_ct=01:30 CT`, window 02:00→08:30 | [O] | knob-census.md:100 | gate → authors v1 | yes | 1 — `auto_trader_planner.go:206` |
| W3 | scheduled read NY | `kernel/session_registry.go:108` + live registry | `read_ct=08:00 CT`, window 08:30→14:45 | [O] | knob-census.md:100 | gate → authors v1 | yes | 1 — `auto_trader_planner.go:206` |
| W4 | wake trigger **level_event** | `trader/auto_trader_wake_levels.go:376` | ON; 6 sub-toggles; ≤1 candidate/cycle | **[T] weak** | belief-census.md:87 (E5 — "52 re-plans/7 days, 7 ever armed") | **full REPLAN trigger, budget-free** | **NO** — census demotion queue #3 (`belief-census.md:132`) says *demote to advisory until n improves*; live it is still a full replan trigger | 1 — `auto_trader_planner.go:347` (+ `:274`, `:282` for dormant / no_trade rows) |
| W5 | wake trigger **structure_mss** | `trader/auto_trader_transition.go:206` | ON; dedupe + `wake_min_interval_min` only. **No cutoff, no cooldown, no cross-session defer, no counters** | [I] | verify-class47-mss-wake-ungoverned.md:1-3 (verdict CONFIRMED) | full REPLAN trigger, budget-free | **NO** — boot line asserts the cutoffs govern it; the code applies none | 1 — `auto_trader_planner.go:341` |
| W6 | **wake cutoff** (minutes to session flat) | `trader/class47_wake_cadence.go:52` (`WakeCutoffMinDefault`), resolver `:61` | **25 min** — `WAKE_CUTOFF_MIN` absent from `/home/hoang/vl/.env` | [O] (owner ruling, `class47_wake_cadence.go:30`) | level-event-wake-audit.md:117-124 — N derived = 15 (last-entry) + p90 planner 9.3 min (n=255 calls) ≈ 25 | **REJECT** (wake returns; counter + WARN line) | yes on the number (25 = 25); **NO on the promotion basis** — see Drift D3 | 1 — `auto_trader_wake_levels.go:316` |
| W7 | **wake cooldown** (since last *wake-authored* version) | `trader/class47_wake_cadence.go:59`, resolver `:62` | **30 min** — `WAKE_COOLDOWN_MIN` absent from `.env` | **[I]** | **none found.** level-event-wake-audit.md:106-135 proposes only P1 (cutoff) and P2 (stream defer). `:121-122` names 30 and *declines* it ("the arm-lead sample is n=6 and version-contaminated, so 25 is what the evidence carries") | **REJECT** (wake returns) | **NO** — no report derives a 30 m wake-authored cooldown; the rule was invented alongside the ruling | 1 — `auto_trader_wake_levels.go:336` |
| W8 | fast-market cooldown exemption | `trader/class47_wake_cadence.go:164-166` (`FastMarketBypass`), threshold `trader/auto_trader_loop.go:80-87` | **≥1.5×ATR5m** — `FAST_MARKET_ATR` absent from `.env` | **[I]** | knob-census.md:31 (`FAST_MARKET_ATR` **1.5** ×ATR5m, `auto_trader_loop.go:86`, label **[I]**) | bypass — waives the **cooldown only**; the cutoff is never waived (`class47_wake_cadence.go:151-154`) | yes (1.5 = 1.5) | 1 — `auto_trader_wake_levels.go:299` |
| W9 | **cross-session claim** (stream defer) | `trader/class47_wake_cadence.go:106-116` (`anyPlannerStreamOpen`), gate `auto_trader_wake_levels.go:353` | **on** — unconditional, no knob | [R] | level-event-wake-audit.md:129-135 (P2: "a process-wide 'a planner stream is open' flag … must NOT apply to scheduled reads") | defer — wake returns, **dedupe key deliberately not consumed** (`:351-352`) so the event re-tries next cycle | yes; scheduled reads correctly exempt (`class47_wake_cadence.go:102-105`) | 1 — `auto_trader_wake_levels.go:353` |
| W10 | **stale-arm expiry** | `trader/armed_executor.go:236-252` (`SupersedeUnplacedArms`) | **on** — unconditional, no knob | [O] (class 47 F4, ruling recorded in the code comment `armed_executor.go:236`) | none found in reports | ledger write — retires never-placed arms whose plan version is superseded | yes | 1 — `armed_executor.go:244` |
| W11 | **wake_min_interval_min** | `store/strategy.go:1486-1491` (`WakeMinIntervalMinutes`), default const `:1447` | **30 min** — key ABSENT from `day_plan` of the live strategy `a5b7662e` → resolver returns `DefaultWakeMinIntervalMin=30` | **[I]** | level-event-wake-audit.md:121-122 (30 noted only as a coincidence, explicitly declined as the cutoff) | gate — throttles wake **attempts** (both wake classes share `lastPlannerWakeAt`) | yes (30 = 30) | **3** — `auto_trader_wake_levels.go:274`, `:185` (freshness window), `auto_trader_transition.go:186` |
| W12 | wake_on_15m_zone | `store/strategy.go:1454-1459` | **true** (unset → default ON) | [I] | none found | gate — enables the 15 m zone + FVG candidate class | unknown (no research value) | 1 — `auto_trader_wake_levels.go:99` |
| W13 | wake_on_htf_zone | `store/strategy.go:1461-1466` | **true** (unset → default ON) | [I] | none found | gate — 1 h/4 h S/D zones | unknown | 1 — `auto_trader_wake_levels.go:124` |
| W14 | wake_on_htf_ob | `store/strategy.go:1468-1470` | **true** — EXPLICIT in the DB (`day_plan.wake_on_htf_ob=true`); the code default is **OFF** | [O] | knob-census.md:101 (`wake_on_htf_ob | true | [O]`) | gate — 1 h/4 h order blocks | yes | 1 — `auto_trader_wake_levels.go:140` |
| W15 | wake_on_ifvg | `store/strategy.go:1479-1484` | **true** (unset → default ON) | [I] | none found | gate — inverted FVGs on all wake TFs | unknown | 1 — `auto_trader_wake_levels.go:156` |
| W16 | wake_on_seated_invalidation | `store/strategy.go:1472-1477` | **true** (unset → default ON); noise band `max(2 tick, 0.15×ATR15)`, freshness `2×wake_min_interval` = 60 min | [I] | none found | gate — seated-level invalidation candidates (**the class that fired every wake I observed**) | unknown | 1 — `auto_trader_wake_levels.go:176` |
| W17 | **death re-plan** | `trader/auto_trader_planner.go:334` → `runDeathReplan` `:765`; trigger const `store/strategy.go:1202` (`death_replan`) | cap = `replan_cap` **4** (global + per-session); **NOT** governed by class 47 | [O] | knob-census.md:98 (`replan_cap 4 [O]`); boot line `main.go:330` "spends: death_replan, owner_reread" | REPLAN, **spends** budget | yes — but **live-dead**: `SELECT count(*) FROM plans WHERE trigger_reason='death_replan'` → **0**, all time | 1 — `auto_trader_planner.go:334` (production, not test) |
| W18 | **owner re-read** | `trader/auto_trader_reread.go:126` | trigger `owner_reread`; spends budget; NOT governed by class 47 | [O] | boot line `main.go:330` free/spend list; `store/strategy.go:1304` | REPLAN, spends budget | yes — but **live-dead**: 0 rows with `trigger_reason='owner_reread'`, all time | 1 — `api/handler_plan.go:1168` (HTTP `POST` handler; `CanForceReread` at `:1142`) |
| W19 | owner reset | `trader/auto_trader_reset.go:129` | trigger `owner_reset`; budget **re-armed**; NOT governed by class 47 | [O] | `store/strategy.go:1304` (free list) | REPLAN, budget-free | yes | 1 — `auto_trader_reset.go:129`; **24 rows all time**, latest 2026-09-04 08:44:46 CT |
| W20 | flip/death → dormant (no re-plan) | `trader/auto_trader_planner.go:296-311` | on; marker `dormant:flip:` / `dormant:death:`; version unchanged, budget untouched | [O] | belief-census.md:86 (E4 [O]) | lifecycle — **pre-empts** the death re-plan for structured deaths | yes | 1 — `auto_trader_planner.go:299` |
| W21 | dormant auto re-arm | `trader/auto_trader_planner.go:262-272` | on; close-back predicate + `DORMANT_MIN_HOLD_MIN` flap guard | [O] | belief-census.md:86 (E4) | lifecycle | yes | 1 — `auto_trader_planner.go:262` |

**No rule in this subsystem is A29-DEAD.** Every rule above has ≥1 production call site. Two are *live-dead on the tape* (W17, W18 — see Drift D6) which is a different and weaker statement, and I state it as such.

### Evaluation order inside one level-event wake (`auto_trader_wake_levels.go:250-381`)

`collectLevelWakeCandidates` → `:269` per-event dedupe (`lastLevelWakeKey`) → `:274` **wake_min_interval_min** → `:299` fast-market drift measured → `:316` **cutoff** → `:328` bypass note → `:336` **cooldown** → `:353` **cross-session stream defer** → `:367-376` fire (async, non-fatal, budget-free).

---

## 2. DISPATCH D8 — "the change-based predicate (stage 4 status)"

**Answer: STEP 2 of 4 is SHIPPED AND BOOTED. Steps 3 and 4 — the change-based predicate itself — are NOT STARTED. Not staged, not on any branch, not in any commit.**

The report exists: `docs/superpowers/reports/2026-09-03-wake-predicate.md` (`git log -1` → `586261ed 2026-09-03 21:36:39 -0500`).

| Step | What it is | State | Evidence |
|---|---|---|---|
| 1 | detector redesign D1′ | shipped earlier (`2026-09-02-detector-redesign.md`, `0465a10b`) | boot 8: `detector: D1' k=3 … touch_outcomes=293 · candidate_pool=168` |
| **2** | **1B wiring** — `recordDetectorOutputs` hooked into the real planner read | **SHIPPED + MERGED + BOOTED** | wake-predicate.md:7 ("STEP 2 of 4 COMPLETE … `f1d7cf51`, NOT deployed"). It is now deployed: `git log --oneline origin/fix/wake-predicate --not origin/dev` → **empty** (branch fully contained in dev); `trader/detector_record.go` present on dev; deploy marker `26594412 deploy: wake-predicate boot marker — RELEASE=89673ccc`. Live proof: `touch_outcomes` = 359 rows and `candidate_pool` = 192 rows now (both first written 2026-09-04 05:30:54 CT), vs `touch_outcomes=0 · candidate_pool=0` in the pre-boot line the report quotes at `:19` |
| **3** | boot it, accumulate a session of real detector data | **DONE by the fact of the boot**, but never converted into a wave | wake-predicate.md:51-53 — "**NOT DONE (needs your GO)**: step 3 — boot this, let a session accumulate `touch_outcomes` / `candidate_pool`, and only then build `WakeChangesAt` on real data" |
| **4** | **`WakeChangesAt` — the change-based wake predicate** | **NOT STARTED** | `git grep -l "WakeChangesAt"` over `git rev-list --all` → hits **only** `docs/superpowers/reports/2026-09-03-wake-predicate.md` in 3 commits. **Zero Go files, at any ref.** `grep -rniE "changesAt\|change_based\|wakePredicate" --include=*.go .` → **zero hits.** `origin/fix/wake-predicate` tip = `586261ed` (a docs commit), fully merged, nothing outstanding |

The report's own A15 forward-warning is the exact live state today: *"the wake drumbeat is unchanged (that is step 4)"* (wake-predicate.md:56-57). It is unchanged. The 09-03 wake numbers below are that drumbeat, throttled but not predicated.

The report also lands an **A29 finding worth carrying forward** (wake-predicate.md:16-19): `recordDetectorOutputs` announced "THE PRODUCTION CALL PATH" in a file banner and had **0 production call sites** for its entire life, while `TestDetectorWritesThroughTheProductionPath` stayed green because it called the hook directly. The standing gate `TestEveryClaimedProductionPathHasACallSite` (`trader/wiring_gate_test.go`) now exists on dev. **This is the same defect shape as Drift D1 below** — and the gate does not catch D1, because D1's claim lives in a *boot line string*, not a function doc or file banner.

---

## 3. E5 / demotion queue item #3 — was it demoted? **NO.**

**Research (belief-census.md:87, pinned `ee64a494`):**

> `| E5 | Level-event wakes deserve a re-read | trader/auto_trader_wake_levels.go:279 | **[T] weak: 52 re-plans/7 days, 7 ever armed** (2026-09-02-level-event-wake-audit.md; WARN-first N=25 proposed) | REPLAN trigger (live) |`

**Demotion queue item #3 (belief-census.md:132):**

> `| 3 | **Level-event wakes deserve a re-read** | [T]-weak (52/7d, 7 armed) | full REPLAN trigger, budget-free | WARN-first N=25 proposal (already written); demote to advisory until n improves |`

**Live code says NO — with the exact opposite motion.** `auto_trader_wake_levels.go:367-380`: on a surviving candidate the wake sets the dedupe key, sets `lastPlannerWakeAt`, logs `🗓️ level wake … waking the planner (W6, 5th wake-up)` and launches `runPlannerReadWithTriggerClaimedCtx(session, tradeDate, "level_event", …)` — a **full plan-authoring read that writes a new version**. Nothing advisory anywhere on the path. `W6-D` at `:362-366` keeps it explicitly budget-free. What actually shipped after the census is the *inverse* of "demote to advisory": the class-47 cutoffs, which shipped WARN-first, were **promoted to ENFORCE** by owner ruling 2026-09-03 (`class47_wake_cadence.go:30`, two-day-audit.md:151 and :736-737).

So: the *belief* was not demoted; a *throttle* was wrapped around it and then hardened. Those are different things and the queue item is still open.

**E5 re-measured on the current tape [T], sample ids named (A21).** 7 days ending 2026-09-04 08:49 CT:

| Quantity | Research (a5a53bec) | Now | Method |
|---|---|---|---|
| level_event re-plan versions / 7 d | **52** | **56** | `plans WHERE trigger_reason='level_event' AND created_at ≥ now−7d` |
| versions carrying an arm row | **7** | **15** | join `armed_orders` on `(plan_id, version)` — *same contaminated column the audit used* |
| versions whose arm was ever PLACED (`signal_id<>''`) | not stated | **10** | as above |
| versions whose arm reached `state='filled'` | 1 of 8 late ones | **6** | as above |

The 15 versions, by id: `2026-08-28:NY` v5 (`filled#10`,`cancelled#11`), v6 (`filled#9`); `2026-08-31:NY` v4 (`filled#21`), v5 (`cancelled#22`), v7 (`cancelled#19`); `2026-09-01:ASIA` v6 (`cancelled#30`), v7 (`cancelled#29`,`cancelled#31`); `2026-09-01:LONDON` v2 (`cancelled#23`), v5 (`cancelled#25`), v6 (`filled#24`); `2026-09-01:NY` v5 (`cancelled#26`,`filled#28`); `2026-09-02:NY` v5 (`cancelled#32`), v12 (`cancelled#33`); `2026-09-03:NY` v2 (`filled#35`), v3 (`cancelled#36`).

**The caveat the audit named is still binding** (level-event-wake-audit.md:50-53 and :144-146): `armed_orders.version` is overwritten on re-authorization, so an arm's `version` is the *latest* version that touched it, not the one it was armed under — these 15/10/6 are upper bounds. The `armed_under_version` column the audit asked for **now exists** (`armed_orders` schema) but is populated on **2 of 38 rows**, so it cannot settle the question yet. Re-run this table once that column has coverage.

**Honest read of the direction:** 15/56 (27 %) carry an arm vs 7/52 (13 %) — better, but on an upper-bound method, across a period that includes an enforcement change mid-window. That is not enough to promote E5 out of [T]-weak, and it is not the "n improves" the queue item wants.

---

## 4. MEASURED SINCE BOOT 8 (2026-09-04 08:30:11 → 08:49:25 CT, **19 m 14 s**)

Source `/home/hoang/vl/data/vl_2026-09-04.log`, sliced at the `BOOT INTEGRITY OK … 70af663d` line (log line 3812).

| Event | n | Detail |
|---|---|---|
| level_event wake **FIRED** | **1** | `09-04 08:46:09 [WARN] … 🗓️ level wake seated OB(bull)·4h invalidated: close 29616.50 below 29863.50 (noise 6.73) on NY 2026-09-04 — waking the planner (W6, 5th wake-up).` |
| structure_mss wake fired | 0 | |
| skipped — `wake_min_interval_min` | 0 | |
| skipped — cutoff 25 m | 0 | NY flat 14:45 CT ⇒ 359 min to flat at the wake |
| skipped — cooldown 30 m | 0 | **see the live finding below** |
| deferred — cross-session stream open | 0 | |
| cooldown bypassed (fast market) | 0 | |
| WARN-first `would_skip` | 0 | none possible; the cutoffs enforce |
| stale arms SUPERSEDED | 0 | 1 arm row created 08:46:09, `state='armed'`, never superseded |
| scheduled reads | 0 | the NY 08:00 read ran under **boot 7** and produced `planner_fail_closed` v1 at 08:11:45 CT |
| owner reset | 1 | `08:32:00 🗓️ OWNER RESET 2026-09-04 NY — chain abandoned at v1; budget re-armed (4 re-plans).` |
| owner re-read | 0 | |
| plan DIED / DORMANT / REARMED | 0 / 0 / 0 | |
| plan versions written | 1 | `08:44:46 🗓️ PLAN written 2026-09-04 NY v2` (trigger `owner_reset`, lifecycle `active`; planner call 566.3 s, attempt 1/3 rejected on a gap-up contract, attempt 2/3 repair) |

**A24/A21 — this window is BELOW the floor for any rate claim.** 19 minutes against a 30-minute throttle and a 566-second planner read is one wake and one plan. I state the counts; I state no rate. The comparative numbers below carry the load.

### The one live finding this window produced [A]

**NY v2 was written at 08:44:46. The level wake fired at 08:46:09 — 83 seconds later — and the 30-minute cooldown did not stop it.** By design: `auto_trader_wake_levels.go:309-315` only starts the cooldown clock when `WakeCadenceGoverns(last.TriggerReason)` is true, and `WakeCadenceGoverns` (`class47_wake_cadence.go:121-127`) returns true for `level_event` and `structure_mss` **only**. v2's trigger was `owner_reset` ⇒ `HaveLastWakeVersion=false` ⇒ `SkipForCooldown()=false` (`:187`).

The A24 reasoning for that (`class47_wake_cadence.go:132-135` — an unresolved value must not manufacture a skip out of a zero) is sound. The consequence is not obviously intended: **the cooldown never protects a freshly-authored scheduled/owner/death plan.** The 30-minute drumbeat the class-47 comment complains about (`:21-25`, "NY produced 12 plan versions on that pattern") can restart within seconds of every session read. Worth an owner ruling on whether the cooldown should measure from the last plan version of *any* kind.

### Cross-boot comparison (grouped by the **in-line** date stamp, not the filename — the log file is named for the process start day, so LONDON 02:00–07:38 CT of 09-04 lives inside `vl_2026-09-03.log`)

| Calendar day (CT) | level wake FIRED | skipped min-interval | skipped cutoff | skipped cooldown | stream defers | fast-market bypasses | WARN-first `would_skip` |
|---|---|---|---|---|---|---|---|
| 09-01 (pre-class-47) | 21 | 256 | — | — | — | — | — |
| 09-02 (WARN-first from 21:19 CT) | 36 | 366 | 0 | 0 | 0 | 0 | **3** |
| **09-03 (first full ENFORCING day)** | **15** | 127 | 0 | **20** | **3** | **1** | 5 |
| 09-04 (to 08:49 CT) | 5 | 46 | **23** | 5 | 0 | 0 | 0 |

Fires per day dropped 36 → 15 across the WARN-first → enforce boundary. Attributing that entirely to the cutoffs would be sloppy — 09-03 also had 10 boots (two-day-audit.md:737) and different tape — so: **[B]** the enforcement is the largest single contributor, **[A]** the enforcement demonstrably refused 20 + 23 = 43 wakes on the cutoff/cooldown rules over 09-03/09-04.

### Counters reconcile EXACTLY with the log — class 35 "counters record, never infer" holds [A]

`system_config` keys (`store/class47_counters.go:15-27`) vs the log lines, keyed by **trade_date** (which is why the calendar split above differs):

| Counter key (trader elided) | DB value | Log lines | Match |
|---|---|---|---|
| `wake_cutoff_class47:…:2026-09-03:ASIA` | 3 | 3 (all at 00:xx CT on 09-04 — ASIA wraps) | ✓ |
| `wake_cutoff_class47:…:2026-09-03:LONDON` | 1 | 1 (the single WARN-first cutoff `would_skip`) | ✓ |
| `wake_cutoff_class47:…:2026-09-04:LONDON` | 20 | 20 (`Recorded n=1…20`; n=12–20 visible in today's file) | ✓ |
| **cutoff total** | **24** | 23 `SKIPPED` + 1 `would_skip` = **24** | ✓ |
| `wake_cooldown_class47:…:2026-09-02:ASIA` | 4 | 4 (all `would_skip`, WARN-first) | ✓ |
| `wake_cooldown_class47:…:2026-09-03:ASIA` | 11 | | |
| `wake_cooldown_class47:…:2026-09-03:NY` | 17 | | |
| **cooldown total** | **32** | 25 `SKIPPED` + 7 `would_skip` = **32** | ✓ |
| `wake_stream_defer_class47:…:2026-09-03:LONDON` | 3 | 3 `⏱ wake DEFERRED` | ✓ |
| `arms_superseded` (F4) | **absent** | 0 fires | consistent |

---

## 5. DRIFT — resolved value / label differs from research or ruling

### D1 — the boot line claims a scope the code does not have. **structure_mss wakes are UNGOVERNED.**

Boot line (`class47_wake_cadence.go:222`, printed live at `main.go:340`): *"cutoffs govern LEVEL_EVENT/**structure_mss** wakes ONLY"*.

`WakeCadenceDecision{` has **exactly one** construction site in non-test code:

```
$ grep -rn "WakeCadenceDecision{" --include=*.go .
trader/auto_trader_wake_levels.go:289:	dec := WakeCadenceDecision{        ← the level_event path
trader/class47_wake_cadence_test.go:264,272,276,280,287,294,298,353,370,394,410   ← tests only
```

`maybeWakePlannerOnMSSAt` (`trader/auto_trader_transition.go:156-211`) contains **no** cutoff, **no** cooldown, **no** `anyPlannerStreamOpen`, and **no** `IncWakeCounter`. Its only limits are the per-MSS dedupe key (`:175-178`) and the shared `wake_min_interval_min` throttle (`:184-189`) — and that throttle is itself behind `StrategyConfig != nil && DayPlan != nil`, so a nil DayPlan removes even that.

`WakeCadenceGoverns("structure_mss")` returns `true` (`:123`) but its **single** production caller is `auto_trader_wake_levels.go:311` — where it is used only to ask whether the *previous* plan version was wake-authored. It never gates an MSS wake.

Independently reproduced by a peer: `docs/superpowers/reports/2026-09-04-two-day-audit-data/verify-class47-mss-wake-ungoverned.md` (`24685b70`), verdict **CONFIRMED**, listing the same missing pieces.

**Class-53 pattern (parity tests must exercise production CALL SITES).** `trader/class47_wake_cadence_test.go:306-317` asserts `WakeCadenceGoverns("structure_mss")==true`. It pins the predicate and never the call site, so it stays green forever while the MSS wake never calls it. Exactly the shape the new `wiring_gate_test.go` was built for — and the gate misses this one, because the false claim lives in a **boot-line format string**, not a function doc or file banner.

**Fix owner: CODE** (wire the cadence into `maybeWakePlannerOnMSSAt`) **or PROMPT/boot-line** (say `LEVEL_EVENT wakes only`). Not ruling — the owner ruling at `:30` is about promotion to enforce, not scope.

### D2 — the cooldown 30 m has no research derivation. **[I] presented in the same breath as a [T]-derived 25.**

`class47_wake_cadence.go:54-59` documents *what* the cooldown measures and *why it differs* from `wake_min_interval_min`, but cites nothing for the number. `grep -rn "WAKE_COOLDOWN\|wake_cooldown" docs/` returns only *descriptions of the shipped value* (two-day-audit.md:151, :177, :736-737; no-trade-band.md:668 quoting a counter) — never a derivation. Meanwhile the audit that derived 25 explicitly **declined** 30: level-event-wake-audit.md:121-122. The boot line prints both with identical `(enforce)` styling, which reads as equal evidentiary standing. They are not equal.

**Fix owner: RULING** (own the 30 as doctrine) **or research** (derive it, e.g. from wake-authored-version → arm-working lead time once `armed_under_version` has coverage).

### D3 — WARN-first ran for **one morning on n=8**, where the research asked for **one week**.

level-event-wake-audit.md:125-127:

> **WARN-first:** log `🗓️ level wake … SKIPPED-CANDIDATE: N min to <session> flat` and count it **for one week without suppressing anything**. Promote to a real skip only if the counted candidates show the same near-zero placement rate as this audit's 8 …

What happened (two-day-audit.md:736-737): WARN-first shipped `09-02 21:19:21 CT`; enforcing from `09-03 10:28:29`. **~13 hours.** Total WARN-first sample: **8** `would_skip` lines — **1 cutoff** and **7 cooldown** (3 on calendar 09-02, 5 on 09-03). The code's own justification (`class47_wake_cadence.go:31-42`) says as much: *"One morning of live observation supplied them"* and then quotes **two** cases.

**And the promotion criterion was never evaluated.** The report said promote only if the counted candidates *"show the same near-zero placement rate"*. No placement-rate measurement of the 8 exists in any report I can find. The cutoff, the rule with the derived N, was promoted on **n=1**.

This is owner-ruled [O], so it stands. It is still a conformance gap and belongs on the record: **the number 25 is [T]-derived (n=255 planner calls); the decision to enforce it is [O] on n=1.**

**Fix owner: RULING** (already made — this is documentation of the basis, not a request to reverse).

### D4 — boot-line literals in a function that advertises "no literals in a boot line".

`class47_wake_cadence.go:218-219` doc: *"WakeCadenceBootLine (F5) — every value READ from its resolver (A12/A24: no literals in a boot line)."* `:224` passes `onOffWord(true), onOffWord(true)` — **two hardcoded `true`s** for `cross-session` and `stale-arm-expiry`. Both features *are* unconditional in code (W9, W10), so the printed `on` is factually correct — but it is correct by coincidence, not by reading. If either is ever put behind a knob or an early return, the boot line will keep printing `on`.

Same class, worse exposure: `kernel/levels_volume_boot.go:25` prints

```
🗓 session reads (owner ruling 2026-08-31, open−30): ASIA 16:30 · LONDON 01:30 · NY 08:00 CT
```

as a **pure literal string**, while the read times it names live in the admin-editable `system_config.session_registry`. They happen to agree today (I diffed both). An admin edit to `read_ct` would make that boot line lie silently.

**Fix owner: CODE.** Both are one-line changes (`onOffWord(anyPlannerStreamOpenWired())` style, and formatting the read times from `reg.Sessions`).

### D5 — the knob registry mislabels **six live wake knobs** as `KnobCandidate`.

`store/knob_registry_table.go:172-177` — `wake_min_interval_min`, `wake_on_15m_zone`, `wake_on_htf_ob`, `wake_on_htf_zone`, `wake_on_ifvg`, `wake_on_seated_invalidation` all carry `Status: KnobCandidate, Consumers: nil` with the note:

> *"no consumer found by a FIELD grep on 2026-09-03 … A METHOD-based reader would NOT appear, so this is NOT dead and must not be removed: it needs a method-level grep with the command quoted before any status change."*

**I ran the grep the note asks for, and quote it:**

```
grep -rnE "\.(WakeMinIntervalMinutes|WakeOn15mZoneEnabled|WakeOnHTFZoneEnabled|WakeOnHTFOBEnabled|WakeOnIFVGEnabled|WakeOnSeatedInvalidationEnabled)\(\)" \
  kernel trader api agent provider store --include=*.go | grep -v _test.go
```

All six resolve to production call sites: `auto_trader_wake_levels.go:99, 124, 140, 156, 176, 185, 274, 276` and `auto_trader_transition.go:186, 187`. **All six are `KnobLive`, not candidates.** The registry's own guard-rail worked exactly as intended — it stopped anyone deleting them — and the promotion it asks for has now been earned. The premise-corrected registry split (144 live / 16 candidate / 7 ineffective) should read **150 / 10 / 7** after this.

**Fix owner: CODE** (`store/knob_registry_table.go`, status + `Consumers`).

### D6 — two rules are live-dead: `death_replan` and `owner_reread` have **zero plan rows, all time**.

| Trigger | Production callers | Rows in `plans`, all time |
|---|---|---|
| `level_event` | 1 (`auto_trader_planner.go:347`) | **106** (2026-08-25 → 2026-09-04) |
| `owner_reset` | 1 (`auto_trader_reset.go:129`) | **24** (2026-08-17 → 2026-09-04) |
| `ASIA/LONDON/NY_scheduled_read` | 1 (`auto_trader_planner.go:1869`) | 24 / 17 / 19 |
| `structure_mss` | 1 (`auto_trader_planner.go:341`) | **2** — 2026-08-24 and 2026-08-26. **None in 9 days.** |
| **`death_replan`** | 1 (`auto_trader_planner.go:334`) | **0** |
| **`owner_reread`** | 1 (`api/handler_plan.go:1168`) | **0** |

`death_replan` is not an A29 dead rule — it has a real caller — but nothing has reached it since the label was introduced by class 35 on 2026-09-01. The mechanism is visible in the code: `auto_trader_planner.go:296-311` routes every `flip-condition:` / `death-condition:` death to **dormant** and `continue`s, so only the legacy "all levels consumed" death can reach `runDeathReplan` at `:334`. That means **the `replan_cap=4` budget, the `deathReplanAllowed` gate and the `replans_exhausted` NO-TRADE are all guarding a path nothing currently walks**, while the budget-free `level_event` path authored 106 versions. Worth naming, because the boot line `🧮 replan budget: … spends: death_replan, owner_reread` describes a budget with **zero recorded spends of either kind**.

`structure_mss` is a third case: the wake **fires** (1 fire on 09-01 per `waking the planner (G4.6`; the peer verify counts 3 fires + 2 skips all-time from `log_events`) but the last fire authored **no version** — 2 rows, last 08-26. So the ungoverned path of D1 is also the rarest path. That lowers D1's *urgency*, not its *validity*: the missing cutoff is a latent 14:40-CT re-plan waiting for the next MSS.

### D7 — every wake throttle is process-local and resets on every boot.

`lastPlannerWakeAt`, `lastLevelWakeKey`, `lastMSSWakeKey` and `lastPlanWritePrice` are `AutoTrader` fields, not persisted. Consequences, all [A] from code:

- `auto_trader_wake_levels.go:274` — `!at.lastPlannerWakeAt.IsZero()` means **the first wake after every restart bypasses `wake_min_interval_min` entirely.** 09-03 had 10 boots (two-day-audit.md:737): ten free wakes.
- `:269` — the per-event dedupe key resets, so an event already woken on can wake again after a restart.
- `fastMarketDrift` (`auto_trader_planner.go:1232-1246`) reads `at.lastPlanWritePrice`, which is 0 after a boot ⇒ returns `(0,0)` ⇒ **the fast-market cooldown exemption is dead until this process writes a plan.** That fail-safe direction is "no exemption" and is the correct one, but it means the exemption was unavailable for a large share of 09-03. It fired exactly **once** on the whole tape.

The two persisted counters (`wake_cutoff_class47`, `wake_cooldown_class47`) survive restarts and reconcile perfectly — the *records* are durable, the *rules* are not.

**Fix owner: CODE**, if the owner wants boot-crossing throttles. Cheap version: seed `lastPlannerWakeAt` from `GetLatestPlanForTraderSession(...).CreatedAt` when the trigger is wake-governed.

---

## 6. NOTE — not drift, but worth the owner's eye

The live `system_config.session_registry` has **`"enabled": false` for ASIA and LONDON**. They run anyway, because the strategy's per-session `enable: true` overrides win at `trader/auto_trader_planconfig.go:108-113` (explicit override is checked *before* `s.Enabled`), and `sessions_enabled` is `["NY"]` alone. The tape confirms it: `ASIA_scheduled_read` ×24, `LONDON_scheduled_read` ×17, most recently `2026-09-04 06:41:15 CT`. Correct per the documented resolver, but anyone reading the registry alone will conclude two sessions are off when both trade. Ties to belief-census demotion queue **#7** ("Pre-NY sessions carry edge — [X]-candidate, own tape 0/6 −$353.5 pre-NY vs NY 3/3 +$177"), which is an owner decision still open.

---

## 7. Commands and queries used (reproducible)

```bash
# Boot slice
grep -n "BOOT INTEGRITY OK" /home/hoang/vl/data/vl_2026-09-04.log      # → line 3812 = boot 8
tail -n +3812 /home/hoang/vl/data/vl_2026-09-04.log > boot8.log

# Wake events, grouped by the IN-LINE date stamp (NOT the filename)
cd /home/hoang/vl/data
grep -hE "⏱ wake SKIPPED: [0-9]+ min to flat" vl_2026-09-0*.log | grep -oE "^09-[0-9]{2}" | sort | uniq -c
grep -hE "⏱ wake SKIPPED: cooldown"           vl_2026-09-0*.log | grep -oE "^09-[0-9]{2}" | sort | uniq -c
grep -hF "waking the planner (W6, 5th wake-up)"   vl_2026-09-0*.log | grep -oE "^09-[0-9]{2}" | sort | uniq -c
grep -hF "waking the planner (G4.6, 4th wake-up)" vl_2026-09-0*.log | grep -oE "^09-[0-9]{2}" | sort | uniq -c

# Counters
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select key,value from system_config where key like '%wake%' or key like '%arm%super%' order by key;"

# Trigger census
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select trigger_reason,count(*),min(date(created_at,'-5 hours')),max(date(created_at,'-5 hours')) from plans group by 1;"

# Live day_plan (resolved via the resolvers, not the file defaults)
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select config from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8';" | python3 -m json.tool

# D1 proof
grep -rn "WakeCadenceDecision{" --include=*.go .          # 1 production site
grep -rn "WakeCadenceGoverns(" --include=*.go .           # 1 production caller, cooldown lookup only

# D8 proof
git grep -l "WakeChangesAt" $(git rev-list --all)         # markdown only, zero .go
git log --oneline origin/fix/wake-predicate --not origin/dev   # empty → fully merged
```

## Fragments written

- `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/cadence-wakes-rules.csv` — the 21-row rule table.
- `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/cadence-wakes-measurements.csv` — every count above with its n and note.

Nothing was written, edited, checked out or reset in `/home/hoang/vl`. DB opened `mode=ro` only.