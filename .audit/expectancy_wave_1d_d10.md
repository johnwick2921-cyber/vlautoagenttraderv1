names rewritten to vl on 2026-09-30 (VL rename)
# D10 — EXPECTANCY, wave 1D: rebuilt from the live store 2026-09-04 08:5x CT

Worktree `/home/hoang/vl-conform` (HEAD `c523a34a`; `492d2067` verified ancestor).
DB read `sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro"` only. Nothing written to `~/vl`.

## 0 — PROVENANCE (`git log -1 -- <path>`, quoted)

```
38a63a9bb2892beb91041bf7e551a8701df8cf9b Thu Sep 3 15:26:02 2026 -0500 docs(1D): report — the model, RED/GREEN, the live table, and two surprises
                                          -- docs/superpowers/reports/2026-09-03-expectancy-1d.md
77e1cdfce0df36b091514f0eb2798545d9f8e898 Thu Sep 3 00:39:25 2026 -0500 docs(1E): Monte Carlo drawdown results — n=64, expectancy indistinguishable from zero (CI -31 to +18), ~1810 trades needed
                                          -- docs/superpowers/reports/2026-09-03-mc-drawdown.md
4be2c73db6d0309bbf4603736e069d55e58a3da4 Mon Aug 31 17:36:53 2026 -0500 0C cutover record: SHIPPED 17:34:21 CT owner GO ...
                                          -- docs/superpowers/reports/2026-08-31-shadow-demotion.md
ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 Wed Sep 2 08:50:38 2026 -0500 docs: belief census 2026-09-02 ...
                                          -- docs/superpowers/reports/2026-09-02-belief-census.md
0c1a808ca1ca90dee9dad84a9d5403f11211406b Thu Sep 3 00:05:11 2026 -0500 docs(excursions): checklist class 54 ...
                                          -- docs/superpowers/reports/2026-09-03-trade-excursions.md
4e8e7e1ae069bc0285f677a316b4771437a39a06 Thu Sep 3 19:37:14 2026 -0500 docs(index): the stranded-branch sweep ...
                                          -- docs/superpowers/research/INDEX.md
```

**1D is DEPLOYED.** The report's own header (`:4`) says *"BUILT, GREEN, NOT DEPLOYED"*; it shipped
in boot 7/8. Live line, read from `/home/hoang/vl/data/vl_2026-09-04.log`:

```
09-04 08:30:11 [INFO] vl/main.go:423 📊 expectancy: cells=41 with_n>=30=0 judged_rollups=2 unresolved=3 excluded_test=3
09-04 08:30:11 [INFO] vl/main.go:291 🔐 BOOT INTEGRITY OK — rev 70af663dcb6f · built 2026-09-04T13:16:34Z · expected 70af663d · goldens PASS
```

**Rebuild method [A].** Not re-implemented in SQL — run through the PRODUCTION path
(`expectancy.LoadAndBuildAt`, `expectancy/aggregate.go:89`) against a `mode=ro` DSN, via a
read-only harness at
`/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/d10dump/main.go`
(`go build ./...` still rc=0 with it present). It reproduces the live boot line **byte-for-byte**:

```
📊 expectancy: cells=41 with_n>=30=0 judged_rollups=2 unresolved=3 excluded_test=3
```

Era constant, quoted as asked — `store/attribution.go:146` `var DayPlanEraStart = dayPlanEraStart()`,
built at `:153-159` as `time.Date(2026, 8, 15, 0, 0, 0, 0, America/Chicago)` = **1786770000000 ms**
(`TZ=America/Chicago date -d '2026-08-15 00:00:00' +%s` → 1786770000). Applied in the query at
`expectancy/aggregate.go:121-122`.

---

## 1 — THE REBUILT 1D TABLE (live, 2026-09-04)

### Exclusion ledger — reconciles exactly, and equals the report's

71 CLOSED rows with `entry_time >= era`. Assigned most-specific-first
(`aggregate.go:191-205`):

| reason | file:line | n now | report §4:138 |
|---|---|---|---|
| test seam `source='e7_farside_test'` | `aggregate.go:193` | **3** | 3 |
| `plan_id='UNRESOLVABLE'` | `aggregate.go:197` | **7** | 7 |
| no condition (no doc / no scenario) | `aggregate.go:202` | **0** | 0 |
| `pnl_corrected IS NULL` (UNRESOLVED, counted, never coerced) | `aggregate.go:222-227` | **3** | 3 |
| crypto era (never loaded — ABSENT, not excluded) | `aggregate.go:121` | **0** | 0 |
| **resolved rows in every statistic** | | **58** | 58 |

71 − 3 − 7 − 0 − 3 = 58. **[A]**

### By condition — the table the rulings are made from

Every cell hand-recomputed from the raw `pnl_corrected` values and matched to the binary at 1e-9.

| condition | n | W/L/F | Σ | mean | sd | mean 95% | win% | status | Δ vs report |
|---|---|---|---|---|---|---|---|---|---|
| acceptance | 6 | 1/4/1 | +4.57 | +0.7619 | 155.4184 | [−123.60, +125.12] | 16.7% | NOT ENOUGH DATA | identical |
| breakout_retest | 9 | 1/8/0 | −581.50 | −64.6111 | 57.5955 | [−102.24, −26.98] | 11.1% | NOT ENOUGH DATA | identical |
| hold | 1 | 1/0/0 | +168.00 | +168.0000 | 0.0000 | [+168.00, +168.00] | 100% | NOT ENOUGH DATA | identical |
| reclaim | 5 | 0/5/0 | −436.50 | −87.3000 | 39.4233 | [−121.86, −52.74] | 0% | NOT ENOUGH DATA | identical |
| **reject** | **31** | 14/16/1 | +586.00 | **+18.9032** | 105.5624 | **[−18.2575, +56.0640]** | 45.2% | **FAILS** | identical |
| sweep_reclaim | 6 | 1/5/0 | −207.00 | −34.5000 | 64.7070 | [−86.28, +17.28] | 16.7% | NOT ENOUGH DATA | identical |

`reject` row ids (A21), **identical to report `:152-153`**: 521, 523, 524, 529, 533, 535, 536, 537,
538, 544, 547, 548, 549, 550, 551, 552, 553, 554, 560, 564, 565, 567, 569, 570, 575, 581, 582, 584,
585, 587, 591.

### By session

| session | n | W/L/F | Σ | mean | mean 95% | status |
|---|---|---|---|---|---|---|
| ASIA | 16 | 2/13/1 | −552.43 | −34.53 | [−60.63, −8.43] | NOT ENOUGH DATA |
| LONDON | 21 | 7/14/0 | +24.00 | +1.14 | [−42.73, +45.01] | NOT ENOUGH DATA |
| NY | 21 | 9/11/1 | +62.00 | +2.95 | [−54.32, +60.23] | NOT ENOUGH DATA |

### By entry path — **the second judged roll-up, which the report never printed**

| path | n | W/L/F | Σ | mean | sd | mean 95% | status |
|---|---|---|---|---|---|---|---|
| armed | 7 | 3/4/0 | −53.00 | −7.57 | 86.50 | [−71.65, +56.51] | NOT ENOUGH DATA |
| **decision** | **51** | 15/34/2 | −413.43 | **−8.11** | 107.91 | [−37.72, +21.51] | **FAILS** |

`judged_rollups=2` (`aggregate.go:685-689`) = `reject` (condition) + `decision` (path). Report §4
names only the first and says *"Exactly one condition clears the floor"* (`:155`) — true as written,
but the second judged roll-up is never rendered anywhere in the report. **This is checklist class 63
recurring one level up**: the report added `judged_rollups=` to stop a misleading juxtaposition, then
printed a table that accounts for only one of the two.

### By level kind — 16 kinds, largest `ONH` n=16, all below the floor

Full CSVs: `d10-1d-by-condition.csv`, `-by-session.csv`, `-by-path.csv`, `-by-level-kind.csv`,
`-cells.csv` (41 five-dimensional cells with row ids).

**MAE/MFE and stop/target shares are blank on every row**, because `trade_excursions` holds
**0 rows** (`aggregate.go:153-168` returns an empty map; live boot 8:
`📐 excursions: logging=on rows=0 backfilled=0 unresolved=0`). Blank, never 0 — conforming. See
premise correction 2 for what this hides.

---

## 2 — D10(a): cells at n ≥ 30 whose status differs from the criterion

**None — because there are none at n ≥ 30.**

- **Five-dimensional cells: 41, with n ≥ 30: ZERO.** Largest cell n = 12. Every one carries
  `Descriptive=true` / `NOT ENOUGH DATA` (`aggregate.go:507-511`). **[A]**
- **Roll-ups at n ≥ 30: exactly 2**, and both statuses match the criterion when recomputed by hand:

| roll-up | n | mean | mean_lo | criterion `n≥30 ∧ mean>0 ∧ mean_lo>0` | code status | agree? |
|---|---|---|---|---|---|---|
| condition=reject | 31 | +18.9032 | −18.2575 | 31≥30 ✓ · +18.90>0 ✓ · −18.26>0 ✗ → **FAILS** | FAILS | **yes** |
| path=decision | 51 | −8.1065 | −37.7238 | 51≥30 ✓ · −8.11>0 ✗ → **FAILS** | FAILS | **yes** |

**Answer: zero discrepancies.** The code criterion at `expectancy/aggregate.go:512-516` is
`c.Mean > 0 && c.MeanLo > 0` → PASSES, else FAILS; on a `mean > 0` cell that is arithmetically
identical to the report's "mean interval excludes 0", so the code and the prose agree. **[A]**

---

## 3 — D10(b): what the data justifies, and what it FORBIDS

### Criterion 1 — verbatim, `2026-09-03-expectancy-1d.md:25-28`

> **Floor.** `n < 30` → `DESCRIPTIVE ONLY`, no verdict rendered. At or above it the status
> is computed from the pre-registered rule: **PASSES ⟺ n ≥ MinN ∧ mean > 0 ∧ mean interval
> excludes 0**, else FAILS.

### Criterion 2 — verbatim, `2026-08-31-shadow-demotion.md:46-50` (the promotion rule)

> A shadowed condition returns to LIVE only if, at n >= 30 shadow setups on our
> own tape, its net-of-friction expectancy LOWER CONFIDENCE BOUND exceeds zero.
> Otherwise it remains shadowed, or is deleted at the court's discretion.
> No promotion on narrative… No promotion on a point estimate without its
> interval.

### Applied arithmetically

**PROMOTION — forbidden, on every candidate.**

| candidate | n | mean | lower bound | criterion | verdict |
|---|---|---|---|---|---|
| reject (only condition ≥ floor) | 31 | +18.9032 | **−18.2575** | lower bound must exceed 0 | **FORBIDDEN** — bound is negative |
| decision path (only other ≥ floor) | 51 | −8.1065 | −37.7238 | mean must exceed 0 | **FORBIDDEN** |
| breakout_retest (shadowed) | **0 shadow setups** | — | — | needs n ≥ 30 | **FORBIDDEN** — n=0 |
| fvg_entry (shadowed) | **0 shadow setups** | — | — | needs n ≥ 30 | **FORBIDDEN** — n=0 |
| every other condition | 1–9 | — | — | needs n ≥ 30 | below floor — **NO VERDICT (A24)** |

**DEMOTION — no criterion exists to apply.** Neither report defines a demotion rule; `FAILS` is a
status, not an instruction, and the report says so (§4 `:155-158`). `breakout_retest` at n=9,
mean −64.61, interval [−102.24, −26.98] — an interval entirely below zero — is nonetheless **below
the floor**, so under criterion 1 it carries **no verdict**. It is already shadowed by owner ruling
(0C), which is a ruling, not a data-driven demotion.

**Plainly: every five-dimensional cell is below the floor; only two roll-ups clear it and both FAIL;
nothing in this table justifies a single promotion or demotion today.**

### The finding that matters most: the promotion criterion is UNSATISFIABLE BY CONSTRUCTION [A]

The 0C criterion is fed by "shadow setups on our own tape". The only instrument that could count
them is `ab_confirm_log` (the E8 counterfactual logger, written at `trader/armed_executor.go:735`
with `IsCounterfactual: shadowed` at `:743`). Measured at **2026-09-04 08:54:47 CDT**:

```
ab_confirm_log total=209
  rows whose condition is 'breakout_retest' or 'fvg_entry' ...... 0
  rows with is_counterfactual = 1 ................................ 0
```

Four days after 0C shipped (2026-08-31 17:34:21 CT), **not one shadow setup has been recorded.**
The chain, each link measured:

1. `kernel/planner_prompt.go:733` tells the model, verbatim: *"breakout_retest stays a normal AI
   play: **the machine never arms it** — GAR-F4"*, and `:731` renders
   `ArmableConditionsLine(...)` (`kernel/arms_bias_coherent.go:41-61`) which lists shadowed
   conditions out of the armable set.
2. **Measured on `plans` since 2026-09-01:** 25 `breakout_retest` scenarios authored; **0** carry
   `arm.enabled`. (reject 87/67 have it, sweep_reclaim 62/6.) `fvg_entry`: **0 scenarios ever
   authored**, all-time.
3. `kernel/shadow_ab.go:52` — `if sc.Arm == nil || !sc.Arm.Enabled || sc.Confirm == nil || sc.Confirm.RefPrice <= 0 { return nil }`.
   No arm block ⇒ **no E8 row, ever**.
4. `kernel/planner_prompt.go:729` closes the other door: *"with the decision path closed, a RESTING
   ORDER IS THE ONLY WAY INTO THE MARKET — a scenario with no arm cannot trade."*

So a shadowed condition can neither trade nor accumulate the evidence its own promotion rule
demands. **The demotion is a one-way door with a written-but-unreachable exit.** The report's own
§5 already said the E8 table *"cannot support a shadow/promote ruling in its current state"*
(`:206-208`) — this measurement shows the reason is structural, not just quality: there is nothing
in it to judge.

---

## 4 — OVERALL ERA EXPECTANCY vs THE MC RIG

The two waves use **different populations**, and the difference is exactly the 7 `UNRESOLVABLE`
rows. Reconciled:

| scope | filter | n | Σ | mean | sd | se | 95% CI on the mean | t | straddles 0? |
|---|---|---|---|---|---|---|---|---|---|
| MC report, `mc-drawdown.md:122,174-175` | `status!='OPEN' ∧ created_at≥era ∧ pnl_corrected NOT NULL ∧ source IN (system,armed_entry,reconcile)` | 64 | −423.93 | −6.6239 | 100.589 | 12.574 | **[−31.268, +18.020]** | −0.527 | yes |
| **same filter re-measured today, minus id 591** | same | 64 | −423.9286 | −6.6239 | 100.5891 | 12.5736 | [−31.2682, +18.0204] | −0.5268 | yes |
| **same filter, LIVE NOW** | same | **65** | −563.9286 | **−8.6758** | 101.1620 | 12.5476 | **[−33.2691, +15.9175]** | −0.6914 | **yes** |
| **1D expectancy scope, LIVE NOW** | + drop `UNRESOLVABLE`, + require a recovered condition | **58** | −466.4286 | **−8.0419** | 104.8926 | 13.7731 | **[−35.0371, +18.9533]** | −0.5839 | **yes** |

The MC figure is reproduced to the last decimal by re-running its stated filter and removing id 591
— **[A]**, the rig's sample construction is honest and re-derivable.

**Has n grown?** Barely. `64 → 65`: **exactly one new trade**, id **591** (2026-09-03 09:05 CT NY,
armed path, reject, OR-H, **−$140.00**). Nothing has closed since — max `exit_time` across the whole
table is `1788445245677` = **2026-09-03 09:20:45 CT**, which is the same `as_of` the 1D report
carried (`:137`, rendered as `2026-09-03T14:20:45Z`). The 1D population is **unchanged, 58 = 58**;
the report's table and today's are the same table.

**Does the CI still straddle zero?** **Yes, on both scopes, and it got slightly wider and more
negative.** MC scope moved from [−31.27, +18.02] to **[−33.27, +15.92]**; the 1D scope reads
**[−35.04, +18.95]**. The one added trade was a $140 loser, which pulled the mean from −6.62 to
−8.68 and did nothing for the interval. Sample size required to separate the point estimate from
zero at power 0.8, by the rig's own formula (`mc-drawdown.md:176`): **1,067 trades** at the current
n=65 effect size (was 1,810 — the number fell only because the *estimated* mean got worse, which is
not progress). **At ~1 closed trade per day over the last 32 hours, this interval is not closing.**

---

## 5 — LIVE RULE TABLE (see the `rules` array for the machine-readable form)

Every row: file:line · resolved value NOW · label · report:line · effect · CONFORMS · production
callers. Highlights:

- **`MinN = 30`** `expectancy/model.go:27` — [O], grounded `2026-08-31-shadow-demotion.md:47`
  ("at n >= 30 shadow setups") — effect **label** — conforms — 4 production refs.
- **PASSES criterion** `expectancy/aggregate.go:512-516` resolved `Mean > 0 && MeanLo > 0` — [O],
  `2026-09-03-expectancy-1d.md:26-27` — **label only, gates nothing** — conforms.
- **0C promotion criterion** — prose only, `2026-08-31-shadow-demotion.md:46-50` — **ZERO production
  callers. DEAD.** Nothing in the binary computes "n shadow setups" or a lower bound for a shadowed
  condition; there is no code path that can ever return a shadowed condition to live.
- **Shadow gate, arm seam** `trader/armed_executor.go:335` and **decision path**
  `trader/entry_gate.go:233` — both REJECT, both live, 2 wiring sites each. The decision-path leg
  landed `95767c7c` (2026-09-02 12:34:26 CT, class 48) **after** ids 589/590 traded
  `breakout_retest` on the decision path that morning — the code comment at `entry_gate.go:229-232`
  names those two rows. Both are in this table (post-0B, NY, ONH, **−$254.00** combined). Escape
  closed. **[A]**

---

## 6 — DRIFT AND FINDINGS

1. **The E8 "usable" number is TWO different numbers wearing one name.** [A]
   `store/ab_confirm.go:402,430` defines usable as `recompute = 'recomputed'` → boot 8 prints
   `🧮 e8: rows=204 usable=55`. `expectancy/aggregate.go:608-612` defines usable as
   `!(EntryPx>0 ∧ |NetPnl| ≥ EntryPx) ∧ NetPnl ≠ 0` → **54**. Two near-identical magnitudes over
   **almost disjoint row sets**: at 08:54:47 CDT, total=209, boot_usable=55, 1D_usable=54,
   **overlap = 5**. 50 of the 55 rows the boot line calls usable are rows 1D excludes; 49 of the 54
   rows 1D calls usable have `recompute=''` and the boot line counts none of them. Neither
   population is wrong on its own terms; together they are checklist class 63 again. The 1D read
   model never reads the authoritative `recompute` column the data-integrity wave added.
2. **`judged_rollups=2` but only one judged roll-up is ever printed** (see §1). The reader of the
   1D report cannot see that the whole decision path (n=51, 88% of the era's trades) is a judged,
   FAILING cell.
3. **The 1D report is not in the research ledger.** `docs/superpowers/research/INDEX.md` has a row
   for the MC rig (`:74`) and none for `2026-09-03-expectancy-1d.md`.
4. **Report file:line drift after merge** (benign, but A9): route `api/server.go:569` → now **:574**;
   boot emission `main.go:317` → now **:420-423**; panel `ExpectancyPanel.tsx:92` → now **:94**.
   `PlanCard.tsx:97` still exact. All `expectancy/*.go` lines in report §2 are exact.
5. **Latent inconsistency in the era predicate.** `expectancy/aggregate.go:121` filters on
   `entry_time >= era`; `store/attribution.go:116,126,203` and the MC rig filter on
   `created_at >= era`. Measured: both give 71 CLOSED rows, 0 rows differ. Latent, not live.

---

## 7 — COMMANDS (re-runnable, read-only)

```bash
# the table, through the production path
cd /home/hoang/vl-conform
go run ./docs/superpowers/reports/2026-09-04-research-conformance-data/d10dump \
  "file:/home/hoang/vl/data/data.db?mode=ro&_pragma=busy_timeout(5000)"

# era constant
TZ=America/Chicago date -d '2026-08-15 00:00:00' +%s      # 1786770000
TZ=America/Chicago date -d '2026-09-02 07:49:06' +%s      # 1788353346  (Era0BStart)

# MC rig population, live
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
SELECT count(*), sum(pnl_corrected) FROM trader_positions
WHERE status!='OPEN' AND created_at>=1786770000000 AND pnl_corrected IS NOT NULL
  AND source IN ('system','armed_entry','reconcile');"     # 65 | -563.9286

# shadow-evidence accrual
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
SELECT count(*),
 sum(condition IN ('breakout_retest','fvg_entry')),
 sum(is_counterfactual=1),
 sum(recompute='recomputed'),
 sum(net_pnl<>0 AND NOT(entry_px>0 AND abs(net_pnl)>=entry_px))
FROM ab_confirm_log;"                                      # 209 | 0 | 0 | 55 | 54

# arm blocks by condition since 0C
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
SELECT json_extract(j.value,'\$.condition'), count(*),
       sum(json_extract(j.value,'\$.arm.enabled')=1)
FROM plans p, json_each(json_extract(p.doc,'\$.scenarios')) j
WHERE p.trade_date>='2026-09-01' GROUP BY 1;"              # breakout_retest 25 | 0
```
