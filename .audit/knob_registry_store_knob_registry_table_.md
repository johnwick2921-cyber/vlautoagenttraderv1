names rewritten to vl on 2026-09-30 (VL rename)
# SUBSYSTEM — THE KNOB REGISTRY AND THE DEMOTION QUEUE

Worktree `/home/hoang/vl-conform` @ `fb50903f` (base: dev tip). Deployed rev `70af663d`, PID 878451, boot 2026-09-04 08:30:11 CT. All DB reads via `file:/home/hoang/vl/data/data.db?mode=ro`.

**`git log -1` for every report cited:**

| report | pinning commit |
|---|---|
| `docs/superpowers/reports/2026-09-02-belief-census.md` | `ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 — every market belief labeled [R]/[X]/[T]/[I]/[O] with live effect + demotion queue (read-only)` |
| `docs/superpowers/reports/2026-08-30-knob-census.md` | `741bfc2a8c443feceaa0f31d30c015946b775633 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX` |
| `docs/superpowers/reports/2026-09-02-level-kind-replay.md` | `3961f8733afa409ec4ef3edfdcff1437fdeac235 2026-09-02 19:03:10 -0500 docs(level-replay part2): 1h variant results …` |
| `docs/superpowers/reports/2026-09-02-live-bias-replay.md` | `53498adb0221d58ce3bcd4d48ec2d347386d83f2 2026-09-02 21:02:58 -0500 docs(live-bias-replay): results …` |
| `docs/superpowers/reports/2026-09-02-bias-calibration.md` | `2deab3c8c25260ca1f7b1b5590cec34da23a9fda 2026-09-02 20:53:20 -0500` |
| `docs/superpowers/research/INDEX.md` | `4e8e7e1ae069bc0285f677a316b4771437a39a06 2026-09-03 19:37:14 -0500` |
| `store/knob_registry_table.go` (the artifact itself) | `eec2335a3ef6dec1c133de21949f4f21a103e682 2026-09-03 22:08:42 -0500 feat(settings): two labels for two different tests — ineffective vs candidate` |

---

## 0. THE CORRECTION THAT MUST LEAD — KNOBS AND BELIEFS ARE DISJOINT POPULATIONS

The dispatch treats "the 144" and "the ~49 labelled beliefs" as the same census at two zoom levels. They are not, and the overlap is smaller than either report implies.

- **The 144** are `KnobLive` rows in `store/knob_registry_table.go` — leaves of the `StrategyConfig` JSON schema, enumerated by reflection (`store/knob_registry.go:63 EnumerateSchemaKnobs`). Every one is a *saved setting*.
- **The ~49** are rows in `2026-09-02-belief-census.md` tables A–F — *rules*, most of which live as Go constants and env vars, **not** as schema leaves.

**[A] Measured consequence:** every single [I]-labelled number the knob census names in its §6 bottom line (`zoneSizeMult` ladder, freshness/anchor decay ladders, `FAST_MARKET_ATR`, `LevelClusterTicks`) is **absent from the registry** — it is a Go constant, not a settings leaf. So the registry's live population and the "invented numbers" population do not intersect at all. **A registry with zero [I] entries is not a clean bill of health; it is a scope boundary.** Anyone reading "144 live knobs, 0 invented" without this sentence has been misled.

Two further premise corrections measured here:

- **Demotion-queue line range.** The dispatch says `belief-census.md:127-137`. The nine ranked rows are at **`:129-137`**; `:125-128` is the header + table head.
- **The knob census uses a DIFFERENT legend** (`knob-census.md:9-15`): `[R] / [D] / [O] / [C] / [I]`. There is no `[T]` and no `[X]`. Its `[D]` (dispatch/backtest-validated with n) is the belief census's `[T]`; its `[C]` (code-canon, no external citation) has **no equivalent** in the six-label vocabulary this audit uses and must not be silently folded into `[I]`. Mapping used throughout: `[D]→[T]`, `[C]` reported as itself.

---

## PART 1 — THE 144

### 1.1 Reproduction of the counts [A]

```
$ grep -c 'Status: Knob' store/knob_registry_table.go
167
$ grep -o 'Status: Knob[A-Za-z]*' store/knob_registry_table.go | sort | uniq -c
     16 Status: KnobCandidate
      7 Status: KnobIneffective
    144 Status: KnobLive
```

Confirmed: **167 entries — 144 KnobLive, 16 KnobCandidate, 7 KnobIneffective.**

**[A] Unused statuses.** `store/knob_registry.go:26-46` defines **seven** statuses. Three are used. `KnobSuspended`, `KnobAdvisory`, `KnobDisplayOnly` and `KnobInfra` have **zero entries** — which matters immediately, because §1.4 shows five knobs that are *exactly* the suspended case and are filed as live.

### 1.2 THE HONEST HEADLINE

Method: word-boundary match of each of the 144 leaf names against both census files, then manual rejection of prose false positives (`breakdown`, `config`, `distribution`, `level`, `session` all hit English prose, not a labelled table row).

> **Of the 144 live knobs, 24 are [R]/[T]/[O]-grounded, 0 are [I], 1 is [X]-flagged, and 119 are UNLABELLED — no evidence label was ever assigned to them by either census.**

Full breakdown:

| bucket | n | share |
|---|---|---|
| `[O]` owner-ruled (knob census §2 DB-stored table, `:80-104`) | 20 | 13.9% |
| `[O]` via a belief-census row (E1/E2 exit family) | 4 | 2.8% |
| **grounded total ([R]/[T]/[O])** | **24** | **16.7%** |
| `[X]`-candidate (queue #7 — `sessions_enabled`) | 1 | 0.7% |
| `[I]` | **0** | 0% |
| **UNLABELLED** | **119** | **82.6%** |

The 24 grounded are all `[O]` — **not one of the 144 live knobs carries an `[R]` or `[T]` label of its own.** (`min_confidence` and `min_risk_reward_ratio` have `[R]`-cited *safe defaults* at `knob-census.md:105`, but the live values are owner-set.) CSV: `knob_registry_labels.csv`.

### 1.3 A29 — FIFTEEN LIVE KNOBS WITH ZERO PRODUCTION CALLERS ON THE MNQ PATH [A]

Consumer-file histogram of the 144 shows 15 whose only cited reader is the crypto **grid** engine:

```
atr_multiplier · config · daily_loss_limit_pct · direction_bias_ratio · distribution ·
enable_direction_adjust · grid_count · leverage · lower_price · stop_loss_pct · symbol ·
total_investment · upper_price · use_atr_bounds · use_maker_only
```

The entire grid branch is gated by `at.IsGridStrategy()` — `trader/auto_trader.go:940-941`, `trader/auto_trader_clock.go:878-879`. The running trader `hoang` (`traders.is_running=1`) binds strategy `a5b7662e-…`, whose `strategy_type` is **`ai_trading`**, not any of the grid aliases at `store/strategy.go:285`. **These 15 are DEAD on the live path** — they would only execute if a grid strategy were running, and none is. The registry marks them `KnobLive` with a real consumer file:line, which is true in the abstract and false for this deployment.

*(Correction of an inference I made and then measured down: I initially suspected the 15 `enable_*` indicator knobs consumed at `kernel/engine_prompt.go:2xx` were crypto-only, because `BuildSystemPrompt` early-returns to `buildFuturesPrompt` at `kernel/engine_prompt.go:27-29`. **Wrong** — `writeAvailableIndicators` is called from `kernel/engine_prompt_futures.go:164` as well. Those knobs are genuinely live on futures.)*

### 1.4 FIVE LIVE KNOBS THAT CANNOT MOVE A STOP [A]

`breakeven_enabled`, `breakeven_trigger_points`, `trailing_enabled`, `trailing_atr_mult`, `trailing_atr_period` are all `KnobLive` with real consumers (`trader/auto_trader.go:196,199`; `trader/auto_trader_trailing.go:43,44`). The live DB has `breakeven_enabled=true`, `breakeven_trigger_points=40`, `trailing_enabled=true`, `trailing_atr_period=14`.

Boot 8 says **`BE=off · trail=off`**. `trader/exit_mechs_suspend.go:34-41` — `exitMechsSuspended()` defaults **TRUE**; `exitMechSuspendedRefuse` (`:61`) returns before the wire at `trader/auto_trader.go:157` (auto-breakeven) and `trader/auto_trader_trailing.go:180` (atr-trail). The single wire hop `moveStopWire` (`:48`) is never reached.

These five are the textbook `KnobSuspended` case — the status exists (`store/knob_registry.go:28`), carries the right UI label ("suspended — …", `:170`), and is used **zero** times. A Studio user reading the registry today is told five knobs are live that cannot send a frame.

### 1.5 SIX KNOBS THE OWNER EXPLICITLY SET, CLASSIFIED NOT-LIVE [A]

| knob | knob-census label | registry status | registry note |
|---|---|---|---|
| `acceptance_rule` | `[O]` `:97` | KnobCandidate | no FIELD-grep consumer |
| `wake_on_htf_ob` | `[O]` `:103` | KnobCandidate | same |
| `max_margin_usage` | `[O]` `:86` | KnobIneffective | prompt text only |
| `min_position_size` | `[O]` `:86` | KnobIneffective | — |
| `last_entry_ct` | `[O]` `:102` | KnobIneffective | audit-dead clock field |
| `eod_flat_ct` | `[O]` `:102` | KnobIneffective | audit-dead clock field |

The live DB carries values for all six (`acceptance_rule="5m_close"`, `wake_on_htf_ob=true`, `max_margin_usage=0.9`, `min_position_size=12`, `last_entry_ct="13:00"`, `eod_flat_ct="14:45"`). The registry is right and the census is stale — but the gap is worth naming: **an owner ruling stored in the DB is not evidence that anything reads it.**

Also: `min_side_levels` is `[O]`-labelled at `knob-census.md:99` and `:198` but is **not a registry key at all** — removed by owner ruling 2026-08-31 (`store/strategy.go:967`).

### 1.6 RESOLVED-VALUE DRIFT AGAINST THE KNOB CENSUS [A]

| knob | knob census value | live DB value (strategy `a5b7662e`, the running trader's) | verdict |
|---|---|---|---|
| `min_risk_reward_ratio` | **3.0** `[O]` `:81` | **2.0** | **NO** |
| `proximity_filter_atr` | **0.3** `[O]` `:96` (also belief D7 `[O]` `:75`) | **1.0** | **NO** |
| `acceptance_rule` | `2x5m` `[O]` `:97` | `5m_close` | **NO** (and the knob is KnobCandidate) |
| `sessions_enabled` | NY/ASIA/LONDON `[O]` `:104` | `["NY"]` — but see §2 queue #7 | see below |
| all other 21 labelled | — | match | yes |

Note the boot-8 volume-wave line says `proximity=cfg(resolved per-trader; retuned 0.3)`. The per-trader config resolves to **1.0**. I cannot see which value the volume wave actually resolved without `/api/config/resolved` (this session has no Authorization header — the endpoint returns `{"error":"Missing Authorization header"}`), so I flag it as an unmeasurable and report the DB value as the day-plan `proximity_filter_atr`.

### 1.7 THE ~49 LABELLED BELIEFS — SAME EXERCISE

Rows A1–A11, B1–B10, C1–C8, D1–D10, E1–E5, F1–F5 = **exactly 49**.

| bucket | n | ids |
|---|---|---|
| `[R]` | 5 | A11 B5 B9 B10 C6 |
| `[R/O]` | 2 | A9 D1 |
| `[T]` | 6 | A4 B1 B8 C5 C7 E5 |
| `[T]`+`[I]` mixed | 1 | C4 (swing-k `[T]`; min-move/MSS `[I]`) |
| `[O]` | 14 | A5 A6 A7 A10 B3 B4 C1 D7 E1 E2 E3 E4 F3 F4 |
| `[I]` | 19 | A1 A2 A3 A8 B2 B7 C2 C3 C8 D2 D3 D4 D5 D6 D8 D10 F1 F2 F5 |
| `[I/C]` | 1 | B6 |
| `[X]` | 1 | D9 |

> **Of the 49 labelled beliefs, 28 are [R]/[T]/[O]-grounded (57%), 20 are [I]-family (41%), 1 is [X] (2%).**

Set against the knobs: **the two populations invert.** 83% of live knobs are unlabelled but 0% are invented; 41% of labelled beliefs are invented but 0% are unlabelled (by construction — the census only lists what it labelled). CSV: `belief_census_49_tally.csv`.

### 1.8 NOW vs 2026-09-02 — WHAT ACTUALLY MOVED [A]

Eleven beliefs changed label, value, or teeth in the two days since `ee64a494`. CSV: `belief_label_changes_0902_to_0904.csv`.

| belief | 09-02 | now | what moved |
|---|---|---|---|
| **B6 min-SL** `:46` | `[I/C]`, REJECT @1.0× | **`[R]`+`[T]`**, REJECT @**1.5×** | `kernel/min_sl.go:18-32` now cites Round-7 research (day-trade range 1.5–2.5×ATR, >60% stop-out below 1.0×) **and** own tape (15 of 27 losers stopped-too-tight; 6 of 8 losers MAE beyond stop). Label upgraded; teeth **tightened**, not demoted. |
| **B7 stale-confirm** `:47` | `[I]`, gate | **`[T]`**, split | `kernel/plan_confirm.go:118-123` cites register S2 mega-research: 38/2,908 (1.3%) marked stale at the old unit vs **~37%** at 2.0×ATR5m, median \|price−ref\| 58.75 pt. **n=2,908.** The census's `[I]` is contradicted by the code's own citation. |
| **B2 BD_CONFIRM_CLOSES** `:42` | `[I]`, "2" | `[I]`, **1** | Default was already **1** on 09-02 (`BD_MIN_CLOSES`, E3 entry-mechanics 2026-08-30, `kernel/breakdown_continue.go:61-71`). The census row was stale when written. |
| **B2 BD_MAX_PULLBACK** `:42` | `[I]`, refusal | **`[M]` dead** | see queue #4a below |
| **E1 breakeven** `:84` | `[O]`, exit fires | `[O]`, **no teeth** | 0B suspension |
| **E2 trailing** `:85` | `[O]`, exit fires | `[O]`, **no teeth** | 0B suspension |
| **E5 level-event wakes** `:88` | `[T]`-weak, budget-free REPLAN | same label, **throttled** | class 47 |
| **A1 bias tree** `:25` | `[I]`, advisory | **`[T]`-null** | `live-bias-replay.md:117,121-122`: BIAS-TREE holdout p=0.4762 (n=21) Wilson [0.2834, 0.6763], net t +0.70; REGIME 0.5435 (n=46); COMPOSITE 0.5000 (n=62). "**Every leg: NOT USABLE at this n.**" Not `[X]` — measured-null, not contradicted. Teeth unchanged (advisory + "reasoning MUST state branch"). |
| **D1–D6, D8 ladders** `:70-76` | `[I]`, weight | `[I]`, weight | `level-kind-replay.md:366`: "**Grader ruling stands: nothing changes; every ladder term keeps its [I] label**" (84 session days, 582 episodes, all kinds TOO FEW, max ONL n=54). |
| **D9 swing seats** `:77` | `[X]`, seated | `[X]`, **still seated** | see queue #2 |
| **Queue #7 pre-NY** `:135` | `[X]`-cand, enabled | **still enabled** | see queue #7 |

**Net:** two `[I]`s were re-grounded to `[R]`/`[T]` (B6, B7), one `[I]` turned out to be dead code (B2 pullback), two `[O]`s lost their teeth entirely (E1, E2), one `[I]` got a measured null (A1), eight ladder `[I]`s were measured and explicitly kept `[I]`. **Zero of the nine demotion-queue items were demoted as the queue specified.**

---

## PART 2 — DISPATCH D1: THE DEMOTION QUEUE, ITEM BY ITEM

`belief-census.md:129-137`. CSV: `demotion_queue_verdicts.csv`.

### #1 — min-SL 1.0×ATR5m + 2t
> `| 1 | **min-SL 1.0×ATR5m + 2t** | [I] | REJECT at write (cancels working arms mid-setup — the 08-30 S1 class) | WARN-first (author + WARN, place if owner-set otherwise); sweep the multiplier (knob census queue) |`

**NOT DEMOTED — re-grounded and TIGHTENED.** `kernel/min_sl.go:34` `MinSLATRMultDefault = 1.5` (was 1.0; 0B owner ruling 2026-09-02), `:40` `MinSLTickClearance = 2`. Boot 8: `min-sl guard: atr_mult=1.5 level_clearance=2tick(s)`. Still a hard REJECT: `kernel/engine_position.go:250` returns an error on the clearance leg, `trader/armed_executor.go:1361-1362` blocks the arm on the ATR leg. **8 production sites** read the resolver. The queue asked for WARN-first; live is REJECT at a 50%-higher floor.

**[A] DRIFT FOUND HERE — the prompt still teaches 1.0×.** `kernel/planner_prompt.go:733` (arm FEASIBILITY CONTRACT) and `:752` (waterfall immediate-mode gate chain) both state, as literal text, "**the stop distance must be ≥ 1.0× the current 5m ATR**". The resolved gate is 1.5×. The class-45 resolved line *is* present (`kernel/planner_prompt.go:460` → `kernel/class45_feeds_forward.go:195-200`, "N pts (1.5×ATR5m …, resolved)"), so the prompt now carries **both numbers, contradicting each other**. Every arm the model authors at 1.0–1.49×ATR5m is refused by a floor the same prompt told it was 1.0×. This is precisely the class-45 "prompt feeds forward" failure the wave was built to close, surviving in two hardcoded strings.

### #2 — Swing seats improve level turns
> `| 2 | **Swing seats improve level turns** | [X] | seated weight; −15pt missed-turns on own tape | re-test vs baseline seats; likely unseat or reduce |`

**UNCHANGED — [X] with live teeth.** `kernel/levels_score.go:101-105`: `KindSWGH, KindSWGL → return 0.85` (anchor-class, same weight as EVWAP/PDVWAP). `kernel/levels_swing.go:38 SwingPointLevels` has **3 production callers** — `kernel/levels_assemble.go:81, 135, 184` — plus role `RoleReactZone` at `kernel/levels_role.go:40`. No re-test on the record; `level-kind-replay.md:366` explicitly leaves every grader term alone. This is the census's own answer to "any [X] enforced?" and it is still true.

### #3 — Level-event wakes deserve a re-read
> `| 3 | **Level-event wakes deserve a re-read** | [T]-weak (52/7d, 7 armed) | full REPLAN trigger, budget-free | WARN-first N=25 proposal (already written); demote to advisory until n improves |`

**PARTIALLY DEMOTED.** Class 47 shipped and then flipped to enforcing: `trader/class47_wake_cadence.go:52` `WakeCutoffMinDefault = 25`, `:59` `WakeCooldownMinDefault = 30`; the enforcement block is `trader/auto_trader_wake_levels.go:279-325` ("CLASS 47 — CADENCE CUTOFFS, ENFORCING (owner ruling 2026-09-03) … Both cutoffs shipped WARN-first … Both now RETURN"), with a fast-market exemption measured before the verdict (`:296-300`) and a shared `wake_min_interval_min` throttle at `:274`. Boot 8: `cutoff=25m(enforce) cooldown=30m(enforce, fast-market>=1.5×ATR exempt)`. **But the wake itself is not advisory** — when it clears the cadence gates it still fires a full REPLAN. The queue asked for demotion of the *trigger*; what shipped is a rate limiter on it.

### #4a — BD_MAX_PULLBACK 0.4
> `| 4 | **BD_MAX_PULLBACK 0.4 / BD_CONFIRM_CLOSES 2** | [I] | author-time refusal in the waterfall law | WARN-first; the rest of the BD family has tape, these two don't |`

**A29 — DEAD. ZERO CALLERS, PRODUCTION OR TEST.**

```
$ grep -rn "bdMaxPullbackFrac" --include=*.go .
kernel/breakdown_continue.go:52:func bdMaxPullbackFrac() float64 {     ← the definition, and nothing else
$ grep -rn "\.Pullback\b" --include=*.go .
(no output)
```

`kernel/breakdown_continue.go:52-59` resolves `BD_MAX_PULLBACK` (default 0.4) and **no code path calls it**. The `PlanBreakdownContinue.Pullback` field (`:38`) is never read either. The census, the knob census (`:41`), and the queue all describe this as a live author-time refusal. **It refuses nothing.** Setting `BD_MAX_PULLBACK` in the environment changes no behaviour whatsoever. Sibling resolvers in the same file are all wired (`bdMinDispATR` 5 sites, `bdConfirmCloses` 5, `bdMaxLevelDistATR` 2, `bdMinSLATR` 1) — this one alone is orphaned.

### #4b — BD_CONFIRM_CLOSES 2
**RELAXED, NOT DEMOTED, AND ALREADY WRONG IN THE CENSUS.** `kernel/breakdown_continue.go:61-71`: the knob is now `BD_MIN_CLOSES`, **default 1**, per E3 entry-mechanics 2026-08-30 — two days *before* the census was written. Still a hard author-time refusal (`:260-262`: "the tape shows NO confirming close beyond %.2f yet"). 4 production sites.

### #5 — stale-confirm 2.0×ATR5m
> `| 5 | **stale-confirm 2.0×ATR5m** | [I] | MET-confirm staleness gate (scenario dies on it) | WARN-first; measure stale-MET outcomes |`

**ALREADY SPLIT — the prompt half is advisory; the arm half is a gate; and the label is wrong.**
- `kernel/plan_confirm.go:155-156` states outright: "FAIL-OPEN … never gates. **Advisory text only — the AI stays the judge**". `:177` only appends a `(stale — …)` parenthetical.
- `trader/armed_executor.go:180-181` is the real gate: `continue // stale-MET is NOT the leak class`.
- The label is not `[I]`: `kernel/plan_confirm.go:118-123` cites register S2 with **n=2,908**. That is `[T]`.
Boot 8: `S-wave: stale_confirm=2.0×ATR5m`. Value unchanged. So the queue's ask ("WARN-first; measure") was **half-satisfied before the queue was written**, and the outstanding half is the arm gate.

### #6 — zoneTFMult + zoneEvidenceByKind + freshness/anchor ladders
> `| 6 | **zoneTFMult + zoneEvidenceByKind + freshness/anchor ladders** | [I] | multiplicative weights on every level grade | sensitivity sweep (knob census §5 items 2-4) before any demotion of downstream gates |`

**MEASURED, INCONCLUSIVE, UNCHANGED.** Values live and identical to the census: `zoneEvidenceByKind` OB `.40/.50/.70/.72` (`kernel/levels_score.go:148-154`), `zoneTFMult` `1.0/1.1/1.2/1.3` (`:157`), `zoneReversalBonus 1.1` (`:159`), `zoneSizeMult` `1.25/1.10/1.0/0.85/0.70/0.50` (`:205-222`), zone freshness `1.0/0.6/0.3/0.15` (`:374-378`, boot 8 `zone-ladder=1.0/0.6/0.3/0.15`), anchor `1.0/0.8/0.6/0.5` (`kernel/levels_swing.go:22`). All multiply into one expression, `kernel/levels_score.go:482`. The sweep the queue asked for became `2026-09-02-level-kind-replay.md` — 84 session days, 582 episodes — and its ruling is `:366`: "**Grader ruling stands: nothing changes; every ladder term keeps its [I] label.**" Every kind TOO FEW (max ONL n=54); pooled holdout 0.6389 sits at the 65.6th percentile of the Osler null. **Verdict discharged as "no verdict"; teeth intact.**

### #7 — Pre-NY sessions carry edge
> `| 7 | **Pre-NY sessions carry edge** | [X]-candidate (own tape 0/6 −$353.5 pre-NY vs NY 3/3 +$177, weekend P2 audit) | LONDON/ASIA sessions enabled + readable | owner decision: session enablement vs evidence |`

**UNCHANGED WITH LIVE TEETH — and the census named the wrong knob.**

Live strategy `a5b7662e` has `day_plan.sessions_enabled = ["NY"]`. That looks like a demotion. It is not:

```go
// trader/auto_trader_planconfig.go:104-113
func (at *AutoTrader) sessionRunnable(s *kernel.SessionDef) (bool, string) {
	if ov := at.sessionOverride(s.Name); ov != nil && ov.Enable != nil {
		if *ov.Enable { return true, "" }          // ← returns BEFORE the registry flag
		...
	}
	if !s.Enabled { return false, ... }
	if !at.sessionEnabledForStrategy(s.Name) { return false, ... }
```

An explicit per-session `enable` **short-circuits both** the registry `Enabled` flag (which is `false` for ASIA and LONDON at `kernel/session_registry.go:93,102`) and `sessions_enabled`. The live config's `day_plan.sessions` array carries `"enable": true` for **ASIA** and **LONDON**. So both run.

Measured on the live DB [A]:

```sql
select session, count(*), max(created_at) from plans where created_at >= '2026-08-28' group by session;
ASIA   | 46 | 2026-09-04 06:20:35
LONDON | 26 | 2026-09-04 12:56:17
NY     | 40 | 2026-09-04 13:44:46
```

**46 ASIA and 26 LONDON plans in the last seven days** — more pre-NY plans than NY plans. The `[X]`-candidate is not merely un-demoted, it is the *majority* of the machine's authoring. And the teeth are `day_plan.sessions[].enable` (registry key `enable`, `KnobLive`, consumer `trader/auto_trader_planconfig.go:73`), **not** `sessions_enabled` as `belief-census.md:135` states. Anyone who "demotes" queue #7 by editing `sessions_enabled` will change nothing.

### #8 — Consumed/3rd-touch → never entry
> `| 8 | **Consumed/3rd-touch → never entry** | [I] | role demotion of levels | measure react rate of consumed levels vs first-touch (touch telemetry exists) |`

**MEASUREMENT DONE, VERDICT WITHHELD, CODE UNCHANGED.** `kernel/levels_role.go:109-118` `RoleFor`: `"done"|"consumed" → RoleTargetOnly`, and far-HTF non-reversal zones → `RoleTargetOnly`. Two production callers (`kernel/levels_score.go:525`, `kernel/levels_role.go:207,443`). The requested measurement landed as `level-kind-replay.md:345`: ordinal holdout **1st 0.688 (n=125) · 2nd 0.571 (n=63) · 3rd+ 0.571 (n=28)** — H8 declines, *supporting* the doctrine — but the same paragraph says "the opposite direction of the 1m replay … Both results are n-tiny; **the belief has no stable direction on a calibrated instrument**." So: measured, direction unstable, doctrine keeps its teeth.

### #9 — Killzone/premium-window + Monday/Thursday conviction
> `| 9 | **Killzone/premium-window + Monday/Thursday conviction** | [I] | advisory prompt lines | keep advisory; add n counter or delete |`

**UNCHANGED — and quietly gained teeth the queue did not account for.** `kernel/planner_prompt.go:652-656` still emits, verbatim: "NY AM 08:30–11:00 CT is the primary window; 10:00–11:00 CT is the premium FVG window; mind the macro minutes. **Conviction: down on Monday, up Thursday/Friday.**" The comment at `:652` says "A4 — killzone weighting (advisory, not a gate)". **No n counter exists** — grep for any killzone/premium telemetry returns nothing.

But killzone membership is no longer purely advisory: `kernel/adherence.go:76-79` steps the adherence grade **down one letter** for "entered outside a killzone" (fed by `kernel/adherence.go:120 sess.InKillzone(t)`, populated at `trader/auto_trader_clock.go:787`). That is a label effect, not a gate — but it is an untested `[I]` belief grading the machine's own discipline. The Monday/Thursday conviction line has **zero** measurement anywhere.

### D1 SCORECARD

| verdict | count | items |
|---|---|---|
| **Demoted as the queue specified** | **0 of 9** | — |
| Partially demoted / throttled | 2 | #3 (class-47 cadence), #5 (prompt half already advisory) |
| Measured, inconclusive, teeth intact | 2 | #6, #8 |
| Unchanged with live teeth | 3 | #2, #7, #9 |
| Value changed in the *opposite* direction | 1 | #1 (1.0 → **1.5**, still REJECT) |
| Found DEAD | 1 (half of #4) | #4a `BD_MAX_PULLBACK` — zero callers |

**Zero of the nine were demoted.** One was tightened, one was dead all along.

### D1's OTHER HALF — every [X] belief still enforced as a REJECT or MUST

The census carries exactly **two** `[X]`-family items. Neither is a REJECT or a MUST, and both are live:

| item | label | live effect | still enforced? |
|---|---|---|---|
| **D9 swing seats** (`belief-census.md:77`) | `[X]` — own tape, missed-turns 80.0/75.0/79.2% → 65.0/60.0/66.7% with swing seats, Δ **−15 pts** (`grand-audit.md:74`) | **weight** — `typeEvidence 0.85`, `kernel/levels_score.go:104`; seated by `kernel/levels_assemble.go:81,135,184` | **YES — still seats, at full anchor-class weight** |
| **Queue #7 pre-NY sessions** (`:135`) | `[X]`-candidate — own tape 0/6 −$353.50 pre-NY vs NY 3/3 +$177 | **enablement gate (permissive)** — `trader/auto_trader_planconfig.go:105-113` | **YES — 46 ASIA + 26 LONDON plans in 7 days** |

**Answer to the D1 question as asked: NO [X] belief is enforced as a REJECT or a MUST.** Both `[X]` items act as a weight and an enablement respectively. That is the good news; the bad news is that neither has been touched, and #7 now authors more plans than NY does.

Cross-check of the census's companion claim ("Any `[I]` enforced as REJECT", `:108`): of its three named law-violation candidates, **B6 min-SL** is no longer `[I]` (re-grounded, and the floor went up), **B7 stale-confirm** is no longer `[I]` (n=2,908 citation) and its prompt half is advisory, and **B2** splits into one dead knob and one relaxed-but-still-refusing knob. The `[I]`-enforced-as-REJECT list, measured today, is **empty** — but not because anything was demoted.

---

## COMMANDS (all read-only)

```bash
cd /home/hoang/vl-conform
grep -o 'Status: Knob[A-Za-z]*' store/knob_registry_table.go | sort | uniq -c
grep -oP '^\t"\K[^"]+(?=".*KnobLive)' store/knob_registry_table.go | sort   # the 144
grep -oP 'Status: KnobLive, Consumers: \[\]string\{"\K[^:]+' store/knob_registry_table.go | sort | uniq -c
grep -rn "bdMaxPullbackFrac" --include=*.go .                              # 1 hit = the definition
grep -rn "SwingPointLevels" --include=*.go . | grep -v _test.go            # 3 production seats
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select session,count(*),max(created_at) from plans where created_at>='2026-08-28' group by session;"
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select config from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8';"
```