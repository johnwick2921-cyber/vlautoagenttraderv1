names rewritten to vl on 2026-09-30 (VL rename)
# SUBSYSTEM F — WEEKLY READER + D6 BIAS — research-conformance re-check

Source tree `/home/hoang/vl-conform` @ `fb50903f` (contains `origin/dev` `492d2067`); every F/D6 file verified **byte-identical to origin/dev** (`git diff --stat origin/dev -- <f>` empty for `kernel/weekly_{bias,knobs,prompt}.go`, `trader/auto_trader_weekly.go`, `kernel/planner_prompt.go`, `trader/entry_gate.go`, `trader/auto_trader_transition.go`). Deployed rev `70af663d` (boot 8, 08:30:11 CT 2026-09-04, PID 878451). DB read-only via `sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro"`.

## Report provenance (`git log -1 -- <path>`, quoted verbatim)

```
2026-09-02-belief-census.md      ee64a494 Wed Sep 2 08:50:38 2026 -0500 docs: belief census 2026-09-02 — every market belief labeled [R]/[X]/[T]/[I]/[O] with live effect + demotion queue (read-only)
2026-09-02-bias-calibration.md   2deab3c8 Wed Sep 2 20:53:20 2026 -0500 docs(bias-calibration): results + CSVs — all three signals and the weekly-structure control NOT USABLE on holdout; TSMOM-252 has a real but friction-destroyed sign edge; control anti-predictive
2026-09-02-live-bias-replay.md   53498adb Wed Sep 2 21:02:58 2026 -0500 docs(live-bias-replay): results — 84 session days, 252 rows; every leg NOT USABLE by D6 (best Wilson lo 0.4018, best net t 0.96; exploration net t -1.35); four-trade premise MEASURED: the machine bias said SHORT/NEUTRAL on 09-02, not long — the longs were the AI overriding its own tree; plan bias is a label, not a direction; calls.csv all 252 rows
2026-08-30-knob-census.md        741bfc2a Tue Sep 1 07:58:16 2026 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX (docs-only — no code, no content edits; collisions suffixed, originals left in place)
2026-08-30-weekly-bias-wave.md   0340d570 Sun Aug 30 08:57:52 2026 -0500 docs(weekly-bias): pin commit ref in the wave report
2026-09-03-studio-audit.md       35e0991a Thu Sep 3 08:34:08 2026 -0500 docs(studio-audit): full Studio settings audit — ONE field table … + 15-item dead-knob list + fix spec
docs/superpowers/research/INDEX.md 4e8e7e1a Thu Sep 3 19:37:14 2026 -0500 docs(index): the stranded-branch sweep — 25 docs-only merged and indexed unclassified, 11 name-only-docs listed as not merged
```

---

## THE RULE TABLE

| rule | file:line | resolved value NOW | label | report:line grounding | live effect | CONFORMS? | production callers |
|---|---|---|---|---|---|---|---|
| **F1 NWOG** weekend gap = weekly level | `kernel/weekly_bias.go:47-56` (type), `:173-236` (`LastNWOGs`) | last 5 gaps · Fri last print (<16:00 CT) → Sun first print (≥17:00 CT) · CE = midpoint · Filled = any 1m bar through CE since birth | **[I]** | belief-census.md:96 (`F1 … [I] weight/advisory`) — no report in `docs/superpowers/reports/` carries an NWOG measurement | **advisory** — facts text in the Sunday weekly-read prompt only (`kernel/weekly_prompt.go:84-96`) | **yes** (with a narrowing: census says "weight/advisory"; the only weight path — the shadow band — is log-only) | 2 — `kernel/weekly_prompt.go:48` (`ComputeWeeklyFacts`), reached from `trader/auto_trader_weekly.go:219` + `kernel/weekly_prompt.go:441` |
| **F2 IPDA 20/40/60** | `kernel/weekly_bias.go:58-66` (type), `:271-295` (`IPDA`) | `[20 40 60]` trading days · HH/LL + PosPct · insufficient history renders `PosPct=-1` → the literal string `"insufficient history"`, never a fake number | **[I]** | belief-census.md:97 (`F2 … [I] advisory`); weekly-bias-wave.md:10 describes the section, cites no study | **advisory** — same prompt-facts path | **yes** | 2 — `kernel/weekly_prompt.go:49`, same two roots |
| **F3 weekly invalidation (1h close beyond px)** | `kernel/weekly_prompt.go:406-425` (`WeeklyInvalidationCrossed`); watch **removed** at `trader/auto_trader_weekly.go:322-327` | **NOT RESOLVED — no runtime path.** `WEEKLY_INVALIDATION_TF_DEFAULT` resolver (`kernel/weekly_knobs.go:110-116`) returns `"1h"` and has 0 callers | **[O]** | belief-census.md:94 (`F3 … [O] gate`) — the census still calls it a live gate | **DEAD** (was gate) | **NO** — research/ruling: *gate*. Live: **0 production callers** | **0 — DEAD.** Only `kernel/weekly_prompt.go:380` (inside the dead `ApplyWeeklyDOA`) + `kernel/weekly_prompt_test.go:132` |
| **F4 weekly DOA breach-at-write guard** | `kernel/weekly_prompt.go:372-386` (`ApplyWeeklyDOA`) | **NOT RESOLVED — no runtime path** | **[O]** | belief-census.md:95 (`F4 … [O] gate`) | **DEAD** (was gate at write) | **NO** — research/ruling: *gate*. Live: **0 production callers** | **0 — DEAD.** `kernel/weekly_prompt_test.go:229` only |
| **F5a confluence band 0.25×ATR5m** | `kernel/weekly_knobs.go:76-83` | **0.25** — `WEEKLY_CONFLUENCE_BAND_ATR` absent from `/home/hoang/vl/.env` and from `/etc/systemd/system/vl.service` (no `EnvironmentFile=`, no `Environment=`); resolver default | **[I]** | belief-census.md:98 (`F5 … [I] weight`); knob-census.md:50 labels it **[C]** code-canon | **shadow** (log-only) | **yes** (value); **census "weight" overstates it** — it is shadow-only, and studio-audit.md:101 agrees ("shadow-only (never change seating)") | 1 — `trader/auto_trader_weekly.go:358` |
| **F5b shadow mult 1.5** | `kernel/weekly_knobs.go:87-94` | **1.5** — env absent, resolver default | **[I]** | belief-census.md:98; knob-census.md:49 **[C]** | **shadow** (log-only) | **yes** | 1 — `trader/auto_trader_weekly.go:359` |
| F-extra `WEEKLY_READ_CT` | `kernel/weekly_knobs.go:23-53` | **"sun 16:30" CT** (env absent → shipped default) | **[O]** | weekly-bias-wave.md:20 (`WEEKLY_READ_CT` default "sun 16:30") | **gate** (scheduler wait/read/skip, `trader/auto_trader_weekly.go:148-156`) | **yes** | 1 — `kernel/weekly_knobs.go:64` → `trader/auto_trader_weekly.go:168` |
| F-extra `WEEKLY_COUNTER_MODE` | `kernel/weekly_knobs.go:98-105` | `"warn"` — **unreachable** | **[I]** | knob-census.md:51 **[C]**, listed as a live env knob | **DEAD** | **NO** | **0 — DEAD.** `kernel/weekly_shadow_test.go:145` only |
| F-extra draw-align tag | `kernel/weekly_prompt.go:546` (helper) / `trader/auto_trader_weekly.go:446-448` (stub) | **literal `"neutral"`** | **[M]** | class-50 comment `trader/auto_trader_weekly.go:442-445` | **label** (`decision_records.draw_align`) | **yes** | stub 1 (`trader/auto_trader_decision.go:51`); `kernel.WeeklyDrawAlignTag` **0 — DEAD** |
| F-extra Sunday-ASIA defer | `trader/auto_trader_planner.go:1446-1454` | on — Sunday ASIA planner read deferred while `weeklyDoc == nil` | **[M]** | — | **gate** (scheduling; reads doc *presence*, never direction) | **yes** | 1 — `trader/auto_trader_planner.go:213` |
| **D6a weekly doc = REFS ONLY** | `kernel/weekly_prompt.go:284` (prompt rule 2) + `:258-265` (validator r4) + `trader/auto_trader_weekly.go:250` | refs-only **ON**; r4 rejects `bull\|bear\|long\|short\|upside\|downside\|biased\|bias` in the narrative; `Conviction` pinned `"n/a"` | **[O]** | bias-calibration.md:122 ("demote the weekly-structure bias … render the chip as *WEEKLY: refs only (no directional call)*") | **REJECT at write** (r4) + label | **yes** (code); see the unproven-in-production caveat below | 1 — `trader/auto_trader_weekly.go:241` |
| **D6b transition_standdown reads plan bias to REFUSE** | `trader/auto_trader_transition.go:51` → `kernel/engine_position.go:274-279` | **ON, cap 45 min** (boot: `regime ledger: … transition_standdown=ON cap=45min`) | **[O]** | live-bias-replay.md:161-163 ("The machine blocks remain useful as FACTS … **nothing changes, nothing is promoted to a gate**") | **gate — REJECT** (returns an error; `telemetry.IncGateBlock(…, "transition_standdown")`) | **NO — drift** | 2 — `kernel/engine_position.go:275`, `trader/auto_trader_transition.go:119` |
| D6b entry-gate leg 1 bias-vs-side | `trader/entry_gate.go:179-184` | **INACTIVE** — needs `plan_mode=="direction"`; **resolved `plan_mode = "strict"`** | **[M]** | live-bias-replay.md:161-163 | gate, **inactive at the resolved value** | **yes (vacuously)** | 1 — `trader/entry_gate.go:454` |
| D6b armed-executor bias check | `trader/armed_executor.go:1331-1336` | **INACTIVE** — same `plan_mode=strict` | **[M]** | same | gate, inactive | **yes (vacuously)** | 1 — `trader/armed_executor.go:430` |
| D6b `planModeBlocked` direction branch | `trader/auto_trader_planconfig.go:209-217` | **INACTIVE** — strict takes the `:218` citation branch | **[M]** | same | gate, inactive | **yes (vacuously)** | 1 — `trader/auto_trader_orders.go:297` |
| D6b flip-fired required bias | `trader/auto_trader_planner.go:1635-1642` | active whenever a prior plan's flip fired | **[O]** | in-code P0.4-G ruling 2026-08-25; no report | **REJECT of the PLAN DOC at write** (not of a trade) | **yes** — mechanics, not belief | 1 (inline in the planner write loop) |
| **D6b `BiasArmWarning` (D2 arms-follow-bias)** | `kernel/arms_bias_coherent.go:74-123` | **NOT RESOLVED — never executed** | **[O]** | code header `kernel/arms_bias_coherent.go:9-16` quotes the owner ruling 2026-09-04 with its measurement (171 stored directional plans; hard reject would refuse **50/68 longs, 66/103 shorts**) | **DEAD** — boot advertises `bias-coherent=warn`, nothing emits it | **NO** | **0 — DEAD.** `kernel/arms_bias_coherent_warn_test.go:52/60/80/83/92` + `arms_bias_coherent_test.go:82/117` only |
| D6b `ArmableConditionsLine` (D1) | `kernel/arms_bias_coherent.go:41` | live — derived from `ResolvedConditionStatuses` | **[O]** | same header, lines 38-40 | prompt text (advisory) | **yes** | 1 — `kernel/planner_prompt.go:731` |
| **D6c bias tree rendered to the model** | `kernel/planner_prompt.go:134-210`, called at `:613` | 6 branches + computed PDH/PDL/PDC + dealing-range position + "facts match branch N" | **[X]** | live-bias-replay.md:162 ("they are rendered as **facts, not orders**") | **advisory prompt text** — branch 5 enforced nowhere in the futures path | **NO — drift on the framing** (see D6c below) | 1 — `kernel/planner_prompt.go:613` |
| **D6b/c dual bias label** | `kernel/planner_prompt.go:269-280`, rendered `:618-619` | `bias: AI yours · tree <x> · regime <y> — labels only, no MUST on either` | **[O]** | live-bias-replay.md:160-161 ("the plan bias is a **label**, not a direction") | **label** | **yes** | 3 — `kernel/planner_prompt.go:618`, `trader/auto_trader_planner.go:1927`, `api/handler_plan.go:682` |
| **D6b executor prompt bias line is SINGLE-source** | `kernel/plan_render.go:157` via `kernel/engine_analysis.go:465` | `Bias: <dir> (<conviction>) · flips: <prose>` — **no tree leg, no regime leg** | **[X]** | live-bias-replay.md:160-161 (class-50b intent: no single source reads as truth) | label (advisory; block header `:156` explicitly permits off-plan) | **NO — partial** | 3 — `kernel/engine_analysis.go:465`, `api/handler_plan.go:1343`, `api/handler_plan.go:2153` |

---

## D6 (a) — "weekly refs-only": is the weekly doc references-only?

**YES in code, on dev, and in the deployed binary — but it has never once run.**

**The wave.** `git branch -a | grep -i weekly` → `fix/weekly-refs-only` exists (tip `3be5ea25`). Its code commits are on `origin/dev` (re-committed hashes): `882c2b7a` kernel, `830717dd` trader+api, `e7a62cb2` web, `6e1a0781` checklist, then class-50b `5a8068f7`/`40803cb3`/`1cee77a8` and the deploy marker `cf8ed4f4` (boot 22:37:38 CT 2026-09-02). Only the branch's *checklist* commit is not on dev.

**The line that makes it refs-only** — three, each load-bearing:

1. Prompt, `kernel/weekly_prompt.go:284`:
```
2. NO directional call: no bias, no conviction, no draw, no invalidation, no long/short/bull/bear language — a direction anywhere = reject. The doc is REFS ONLY.
```
2. Validator, `kernel/weekly_prompt.go:258-265` (r4) — the enforcement:
```go
for _, tok := range []string{"bull", "bear", "long", "short", "upside", "downside", "biased", "bias"} {
    if strings.Contains(low, tok) {
        return fmt.Sprintf("r4: directional token %q in narrative — the weekly doc is REFS ONLY (no bias call)", tok)
```
3. Write path, `trader/auto_trader_weekly.go:250`: `doc.Conviction = "n/a" // class 50: refs-only — the field is fixed "n/a"`, and `:256-258` keeps the deterministic rule alive as **shadow only** (`ShadowBias`/`ShadowWhy`, "never a direction").

Downstream is consistent: `WeeklyContextLine` (`:313-329`) and `WeeklyExecutorLine` (`:333-342`) render only `WEEKLY: refs only — PWH x · PWL y`; the W4 mid-week watch is *deleted* (`trader/auto_trader_weekly.go:322-327`); `weeklyCounterShadow` returns immediately (`:401-403`); `weeklyDrawAlignTag` returns the literal `"neutral"` (`:446-448`); the API ships `refs_only: true` + `pwh`/`pwl` (`api/handler_plan.go:51,59,62`); the chip renders `WEEKLY refs — PWH … · PWL …` (`web/src/components/plan/WeeklyChip.tsx:32`).

**⚠ The caveat that matters more than the code.** The refs-only path has produced **zero** weekly docs. The one live row is a **pre-wave directional doc**:

```
sqlite> select trade_date,session,version,trigger_reason from plans where session='WEEKLY';
2026-08-31|WEEKLY|1|weekly_boot_backfill      -- created_at 2026-09-03 00:06:24 UTC = 2026-09-02 19:06 CT
doc = {"bias":"neutral","conviction":"low","draw":{"name":"PWL","px":28927.25},
       "invalidation":{"px":29811.75,"basis":"1h close beyond 29811.75"},
       "weekly_levels":[PWH 29811.75, PWL 28927.25, 20d high 31103.5, 20d low 26739],
       "narrative":"… A 1h close above 29811.75 negates the bearish bias.",
       "invalidated_at":"2026-09-02 19:06 CT"}
```

It was written **2h31m before** the class-50 boot, and its narrative contains `bias`/`bearish` — **the current r4 validator would REJECT this doc**. Boot 8 still reports it: `📅 WEEKLY READ skip-fresh — week 2026-08-31 doc already stored (v1), idempotent.` [A]

Two consequences, both honest:
* **Fail-safe holds.** `weeklyLevelPx` (`:298-308`) reads only `weekly_levels`, so the running prompt/chip render `WEEKLY: refs only — PWH 29811.75 · PWL 28927.25` from a directional doc. Nothing directional escapes. [A]
* **The refs-only prompt/validator is UNPROVEN in production.** First exercise is the next Sunday read (2026-09-06 16:30 CT). Verdict-with-n: **n = 0 refs-only docs written**. [A]

**Bonus, on the record (the last live F4 firing).** The 09-02 backfill logged the DOA guard doing exactly what F3/F4 claim, seconds before the wave retired it:
```
09-02 19:06:24 [WARN] 📅 WEEKLY READ 2026-08-31 stamped NEUTRAL AT WRITE (F5 DOA) — invalidation 29811.75 already crossed by a closed 1h bar
09-02 19:06:24 [INFO] 📅 WEEKLY READ written 2026-08-31 v1 bias=neutral … invalid=29811.75 thin=false facts_hash=5b4e907886…
```
and the original Sunday read it replaced:
```
08-30 17:07:15 [INFO] 📅 WEEKLY READ written 2026-08-31 v1 bias=bear … invalid=29535.00 thin=true facts_hash=b8591ad6…
```

---

## D6 (b) — the DUAL LABEL, and does any gate read bias to REJECT?

**The dual label is real and live.** `BiasLabelLine` (`kernel/planner_prompt.go:271-280`) is rendered into the planner prompt at `:618-619` followed by the literal `" — labels only, no MUST on either\n\n"`, stamped onto the doc at `trader/auto_trader_planner.go:1927-1929`, and shipped to the FE at `api/handler_plan.go:682`. `TreeCallWord` (`:222-252`) and `RegimeCallWord` (`:257-267`) reproduce the replay's P2 definitions **exactly**, branch-5 veto included.

Measured live (`plans`, `json_extract(doc,'$.bias_label')`): **n = 23 of 249 non-WEEKLY plan rows carry it** — every plan written since the class-50b boot; earliest 2026-09-03 00:08 CT, latest 2026-09-04 NY v2. AI-vs-tree: `long/long 11 · short/short 4 · neutral/neutral 2 · short/neutral 4 · neutral/long 1 · neutral/short 1` → **agreement 17/23 (73.9%)**; **no row shows a head-on long-vs-short contradiction** at n=23. [A]

### Every production reader of the bias fields, classified

| # | reader | class | verdict |
|---|---|---|---|
| 1 | `kernel/plan_doc.go:493` `NormalizeBiasDirection`, `:602-606` enum check | schema | **mechanics** |
| 2 | `kernel/planner_prompt.go:324` FVG-demand warn | **advisory WARN** (`trader/auto_trader_planner.go:1756`) |
| 3 | `kernel/plan_render.go:157` plan block | **label** (into the executor prompt) |
| 4 | `trader/auto_trader_planner.go:1635-1642` flip-fired required bias | **REJECT of the plan doc at write** — mechanics (honours an already-fired machine flip), not a market belief |
| 5 | `trader/auto_trader_planner.go:1927` dual-label stamp | **label** |
| 6 | `kernel/arms_bias_coherent.go:78` `BiasArmWarning` | **DEAD** — 0 production callers |
| 7 | `trader/entry_gate.go:437` → `:179-184` leg 1 | **gate, INACTIVE** (`plan_mode != "direction"`) |
| 8 | `trader/auto_trader_planconfig.go:210-216` | **gate, INACTIVE** (strict takes `:218`) |
| 9 | `trader/armed_executor.go:430`/`:510` → `:1331-1336` | **gate, INACTIVE** (same) |
| 10 | `trader/rootfix_shadow_ab.go:133` | **shadow A/B** |
| 11 | `api/handler_plan.go:679-724` | **label / diff text** |
| 12 | **`trader/auto_trader_transition.go:51`** → `kernel/engine_position.go:274-279` | **GATE — ACTIVE. REJECTS.** |

### The drift: **a live gate DOES read the AI-authored bias to refuse an entry**

```go
// trader/auto_trader_transition.go:51
bias := strings.ToLower(strings.TrimSpace(plan.Doc.Bias.Direction))
…  // :105  Dir: bias
// kernel/engine_position.go:274-279
if ctx != nil && ctx.TransitionActive {
    if blocked, msg := TransitionStanddownVerdict(d.Action, ctx.TransitionActive, ctx.TransitionDir, ctx.TransitionDetail); blocked {
        telemetry.IncGateBlock(ctx.TraderID, "transition_standdown")
        return fmt.Errorf("%s", msg)     // ← REJECT
```
Boot line: `regime ledger: … transition_standdown=ON cap=45min · flip hysteresis hold=30min`. This is a gate whose *direction* comes from `Bias.Direction`, so a bias flip changes which side gets refused. It only ever refuses entries **with** the bias (`kernel/transition.go:103-104` lets counter-direction through), which is why it is not a "trade the bias" mandate — but it is unambiguously **a gate reading bias to reject**, and live-bias-replay.md:162 says "nothing is promoted to a gate."

**How much does it bite?** Across all 20 log files (2026-08-16 → 2026-09-04): **`TRANSITION OPENED` = 9** (3 on 08-24, 2 on 08-26, 3 on 09-02, 1 on 09-03) and **`TRANSITION STAND-DOWN` refusals = 0**. So the gate arms but has refused **0 of 0** entries in 20 days. [A] `/api/risk/gate-blocks` needs an `Authorization` header this session does not have, so the in-memory counter could not be cross-read.

**The second drift: the executor never sees the second and third legs.** The dual label reaches the *planner* prompt, the stored doc and the API — but the executor prompt is fed `kernel/plan_render.go:157` `Bias: long (medium) · flips: …`, a single-source directional word with no `tree`/`regime` leg. The class-50b intent ("no single source reads as truth", live-bias-replay.md:160-161) is honoured in one of the two prompts. [A]

**Resolved `plan_mode` — how I got it, and its caveat (A11).** No boot line prints it; `/api/config/resolved` (which exposes it as `day_plan.plan_mode`, `api/config_resolved.go:109`) returns `{"error":"Missing Authorization header"}` from this session. So: resolver path `store/resolve_source.go:47-55` (session override → strategy-level → `"advisory"`), reading the row the running process loaded — trader `8d5c8af5_8ef641a7-…_deepseek_1781246265` (`is_running=1`) → strategy `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` "MNQ" → `config.day_plan.plan_mode = "strict"`, all three session overrides `null`. `strategies.updated_at = 2026-09-01 13:13:06 UTC`, i.e. **3 days before** the 2026-09-04 13:30 UTC boot, so the cached-at-load config is this row. **`plan_mode = "strict"` [A].** Corroboration is one-sided: `grep "plan_mode=direction"` over all logs = **0** (consistent with strict), but `grep "refused: strict"` = **0** too — the strict legs have also never fired, so the logs prove non-direction, not strict positively.

---

## D6 (c) — is the machine bias tree rendered as FACTS ONLY?

**NO. It is rendered as branch RULES plus a MUST plus two prohibitions — not as facts.** The report's claim is live-bias-replay.md:162: *"The machine blocks remain useful as FACTS for the AI to reason from (they are rendered as facts, not orders)."* The shipped text disagrees on three counts. `kernel/planner_prompt.go:157-163,183-189`:

```
## BIAS-TREE (machine branches — your reasoning MUST state the branch you took, e.g. "bias-tree: inside-day long LOW")
  1. close > PDH → bull-continuation, conviction HIGH
  2. sweep of PDH + close back inside → bear, conviction MEDIUM  (mirror: sweep PDL + close inside → bull MEDIUM)
  3. inside the day (between PDH/PDL) → direction of close vs PDC (prior close), conviction LOW
  4. closed OUTSIDE the prior day's range but now inside → NO bias (write neutral; trade the structure, not a thesis)
  5. premium/discount: longs ONLY below the 50% mark of the dealing range, shorts ONLY above it
  6. draw-on-liquidity: the runner target is the DRAW — the nearest opposing liquidity pool beyond the first target
  computed: PDH … · PDL … · PDC … · dealing range …–… · price at NN% of the range (PREMIUM — longs disallowed by branch 5)
                                                        … · facts match branch 3 (inside day; close short PDC → short LOW)
```

1. **A MUST** — line 157: `your reasoning MUST state the branch you took`. A reporting obligation, not a direction, but it is a MUST on the bias block.
2. **Two prohibitions in imperative grammar** — line 162 (`longs ONLY below the 50% mark … shorts ONLY above it`) and line 161 (`write neutral; trade the structure, not a thesis`). These are orders.
3. **The computed line hands the model a verdict, not a measurement** — lines 183-189 render `longs disallowed by branch 5 (premium)` / `shorts disallowed by branch 5 (discount)` rather than the bare percentage.

**Mitigation, and why the drift is framing rather than teeth.** Branch 5 is enforced **nowhere** in the futures path: `grep -rn "premium\|discount" --include=*.go .` outside `planner_prompt.go`/`weekly*` hits only Binance `premiumIndex` in `agent/tools.go` and `market/data.go`. So "disallowed" is a word the model reads, not a gate — but that is exactly the class-45 hazard ("the prompt feeds forward"): the prompt orders a restriction nothing checks.

The immediately following two lines are the corrective the class-50b wave added, and they are honest — `kernel/planner_prompt.go:618-619`:
```
bias: AI yours · tree short · regime up — labels only, no MUST on either
```

**Same pattern one block down (worth naming here).** `kernel/planner_prompt.go:730`: *"ARMS FOLLOW THE BIAS … Every scenario in the plan's bias direction that has a concrete trigger price **MUST** carry an arm. A long plan with no long arm is **invalid**."* The validator that was supposed to observe this — `BiasArmWarning` — is **never called** (finding below). The prompt calls a plan invalid; nothing checks, nothing warns.

---

## The measured directional accuracy, WITH n (what the labels rest on)

**Weekly structure bias — `2026-09-02-bias-calibration.md` @ `2deab3c8`:**
* D4 table, line 93 (MNQ holdout): **n = 139 weeks · raw hit .252 · Wilson lo .1870 · called-only n = 77 hit .455 (Wilson lo .3481) · net t −13.56**. Line 95 (ES holdout): **n = 139 · .281 · lo .2126 · called-only 77 / .507 / lo .3972 · net t −14.15**.
* Line 97: *"RAW holdout hit rate is 25–28% — significantly BELOW 50% … i.e. anti-predictive; even counting only called weeks it is chance (45–51%…)."*
* Line 118 verdict: **`CONTROL weekly-structure … NOT USABLE — anti-predictive on holdout`**.
* Line 122 is the ruling the refs-only wave implements verbatim, including *"Do not invert the current bias."*

**Live intraday bias — `2026-09-02-live-bias-replay.md` @ `53498adb`:**
* R1 line 110: **84 completed session days, 252 session-plan rows**; holdout 34d.
* R2 lines 117-119 (holdout): **BIAS-TREE called n = 21 · hit 0.4762 · Wilson [0.2834, 0.6763] · net t +0.70** · **REGIME n = 46 · 0.5435 · [0.4018, 0.6785] · +0.96** · **COMPOSITE n = 62 · 0.5000 · [0.3792, 0.6208] · +0.92**.
* Line 122: *"**Every leg: NOT USABLE at this n.**"* Line 136: exploration composite **n = 96, p = 0.4583, net t = −1.35** — the sign flips between periods.
* Line 149: *"The machine bias did **NOT** say long on today's losers — it said SHORT (ASIA, LONDON) and NEUTRAL (NY)."*

No holdout leg clears Wilson-lo > .50 or net t > 2 at any n on the tape. Every F1/F2/F5 knob is [I] with no report behind it; the census's [I] labels are correct and the guide's "researched default" claim is not.

---

## Findings, ranked

**1 — A29 DEAD, and the boot line lies about it. `kernel.BiasArmWarning` has 0 production callers.** Its sibling warnings in the same wave are all wired at the same site (`trader/auto_trader_planner.go:1747` ChainWarnings, `:1756` FvgDemandWarnings, `:1764` FantasyTargetWarnings); `BiasArmWarning` is not. Exhaustive search across `*.go`/`*.ts`/`*.tsx` returns only its definition and 7 test call sites. The boot line `🎯 arms: bias-coherent=warn` is a **hardcoded literal** in the format string at `trader/arms_boot_line.go:21` — nothing resolves it. Live corroboration: `grep -h "⚠ bias=" data/vl_2026-0*.log` = **0** across 20 files. The wave landed `fd3fadcd` (07:29:55 CT 2026-09-04) and IS in the deployed rev (`git merge-base --is-ancestor fd3fadcd 70af663d` → YES; `70af663d` = *"merge fix/arms-follow-bias (boot 8 …): … bias-coherence warning …"*). Class 45/49 violation (boot lines are READ, never literal) + A29. **Owner: code.**

**2 — F3 + F4 are DEAD but the census still labels them live [O] gates.** `ApplyWeeklyDOA` and `WeeklyInvalidationCrossed` have 0 production callers since class 50 (`830717dd`); `InvalidatedAt` is read only inside the dead function. belief-census.md:94-95 must be corrected to `[O] → DEAD (retired by class 50)`. **Owner: ruling/census doc.** Their last live firing is on the record (09-02 19:06:24 CT log line, above).

**3 — a live gate reads the AI bias to REJECT, against live-bias-replay.md:162.** `transition_standdown` (`trader/auto_trader_transition.go:51` → `kernel/engine_position.go:274-279`). Refusals measured: **0 over 20 log files, 9 arms**. Either the report's blanket "nothing is promoted to a gate" needs the standdown carve-out written in, or the standdown needs a bias-independent direction source. **Owner: ruling.**

**4 — the executor prompt still shows a single-source bias.** `kernel/plan_render.go:157`. The dual label exists in three of four surfaces. **Owner: code (one-line render change).**

**5 — the bias tree is rendered as orders, not facts (D6c).** `kernel/planner_prompt.go:157,161,162,183-189`. Branch 5 is enforced nowhere, so the prompt orders a restriction no code checks. **Owner: prompt.**

**6 — `ValidateWeeklyDoc` ignores two of its three parameters.** `kernel/weekly_prompt.go:236` takes `refs []float64` and `thinHistory bool`; the body (`:236-267`) references neither (verified by `awk NR>=236&&NR<=267 | grep`). The call site computes both (`trader/auto_trader_weekly.go:241`). Consequences: (a) the r3 "the draw must be a computed reference" check no longer exists, yet the doc comment at `:133-136` still says *"every computed reference the validator's r3 rule tests the draw against"* — stale by one wave; (b) the thin-history conviction clamp is gone. `WeeklyRefSet` survives with 1 real consumer (`:443`, the F5 shadow band). **Owner: code (drop the params or restore the checks) + comment.**

**7 — a prompt restriction with no validator behind it.** `kernel/weekly_prompt.go:283` orders *"px copied EXACTLY"*, but r1 (`:242-249`) only checks `name != ""` and `px > 0`. A hallucinated weekly level passes. The class-38 guard (boot: `prompt/validator contract: 19 restrictions, all stated in prompt`) checks validator→prompt, not prompt→validator, so it cannot see this. **Owner: code.**

**8 — guide claims research that does not exist (GUIDE CONTENT LAW).** `web/src/guide/content/weeklyBias.ts:69-70`: `systemDefault: '0.25'` / `recommended: '⭐ 0.25 — the researched default.'` No report grounds 0.25; belief-census.md:98 says **[I]**, knob-census.md:50 says **[C]**. Same page markets a shadow-only knob as a tuning recommendation. **Owner: prompt/guide content.**

**9 — F5 is measurably inert on the live tape.** Across 2026-08-30 → 2026-09-04: **47 `🌗 SHADOW wk-seating` lines, 34 of them `0 confluent level(s)`, 33 `🌗 SHADOW wk-confl` level lines total — and 0 confluent levels on every one of the 9 seating lines since 2026-09-03**. The 0.25×ATR5m band has stopped matching anything, so the "Sep-9 promotion table" the log line advertises would be built on 13 non-zero sessions out of 47. Per-file counts in `D6-F5-live-counts.csv`. **Owner: ruling (retune or retire before the Sep-9 promotion decision).**

**10 — data note, outside F but found inside it.** `plans` stores one row per version (2026-09-03 ASIA: 7 rows / max version 7). The WEEKLY rows for week 2026-08-31 went **v1 (08-30 17:07) → v2 (skip-fresh names v2 from 08-30 17:10 through 09-02) → gone**: at 09-02 19:03 the scheduler's verdict was `read`, which requires `GetLatestPlanForTraderSession == nil`, and the 19:06 write returned **v1**, not v3. Something deleted the week's WEEKLY plan rows between the last v2 skip-fresh and 19:03 CT on 09-02. Cause **[C] unknown** — I did not investigate outside my subsystem. Timeline in `D6-weekly-doc-timeline.csv`.

---

## Commands (all read-only)

```bash
# resolved plan_mode (no boot line; /api/config/resolved needs auth)
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select config from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8';" \
  | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['day_plan']['plan_mode'])"   # -> strict

# the live weekly doc (pre-wave, directional, DOA-stamped)
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" ".mode line" \
  "select doc from plans where session='WEEKLY' order by trade_date desc, version desc limit 1;"

# dual bias label coverage
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select count(*) from plans where json_extract(doc,'\$.bias_label') is not null;"          # -> 23

# dead-rule proof
cd /home/hoang/vl-conform && grep -rn "ArmWarning" . --include=*.go            # defs + tests only
grep -rn "ApplyWeeklyDOA\|WeeklyCounterMode" --include=*.go . | grep -v _test.go # defs + a comment only

# live counters
grep -h "TRANSITION STAND-DOWN" /home/hoang/vl/data/vl_2026-0*.log | wc -l   # -> 0
grep -h "TRANSITION OPENED"     /home/hoang/vl/data/vl_2026-0*.log | wc -l   # -> 9
grep -h "⚠ bias="               /home/hoang/vl/data/vl_2026-0*.log | wc -l   # -> 0
```

## Unmeasurable from this session

* `/api/config/resolved` and `/api/risk/gate-blocks` both return `{"error":"Missing Authorization header"}` — no resolved-knob dump and no in-memory gate-block counts.
* `/proc/878451/environ` is unreadable (rc=1, permissions), so the `WEEKLY_*` resolutions rest on `/home/hoang/vl/.env` (no `WEEKLY_*` keys) + `/etc/systemd/system/vl.service` (no `EnvironmentFile=`, no `Environment=`) + the unit's own comment *"Services never read ~/.bashrc — every env var the bot needs must be in .env"*. **[B]**, not [A].
* Refs-only validator/prompt behaviour in production: **n = 0**. First exercise is Sunday 2026-09-06 16:30 CT.
* Whether the weekly `weekly_levels` prices are actually copied from the facts (no validator, and `facts_hash` covers the facts text, not the model's echo of it).
