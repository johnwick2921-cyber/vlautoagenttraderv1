# RESEARCH-CONFORMANCE RE-CHECK — every live rule against the research verdicts

**Window** boot 8, rev `70af663d`, booted 2026-09-04 08:30:11 CT (PID 878451) · SIM-only, MNQ
**Branch** `docs/research-conformance-0904` · claim `fb50903f` · base `492d2067` (dev tip, boot-8 marker)
**Method** READ-ONLY. Resolved values are taken from the running process's own boot lines or the
resolver path — never a file default. `/api/config/resolved` and `/api/risk/gate-blocks` both
return `{"error":"Missing Authorization header"}` from this session (A11 stated, not guessed).
**Evidence classes** **[A]** verified · **[B]** inferred · **[C]** speculation

---

## 0. THE HONEST LINE AT THE TOP

**[A] The dispatch conflates two different populations, and the headline is only meaningful once
they are separated.**

| population | size | what it is |
|---|---|---|
| **live knobs** | **144** | `KnobLive` entries in `store/knob_registry_table.go` (167 total: 144 live · 16 candidate · 7 ineffective) — schema leaves with a proven consumer outside `store/` |
| **labelled beliefs** | **49** | the rules labelled [R]/[X]/[T]/[I]/[O] in the 2026-09-02 belief census (A1-A11, B1-B10, C1-C8, D1-D10, E1-E5, F1-F5) |

A knob is a *number with a reader*; a belief is a *rule with a claim about the market*. The census
never labelled the 144, and the registry never labelled the 49. **"How many of the 144 live rules
are [R]/[T] vs [I]" cannot be answered as posed** — the 144 carry no evidence labels at all. What
can be answered, and is:

### The 49 labelled beliefs — 2026-09-02 vs now

| label | 09-02 census | now | Δ |
|---|---|---|---|
| **[R]** researched | 7 | 7 | — |
| **[T]** own-tape | 7 | **9** | **+2** |
| **[O]** owner-ruled | 14 | 14 | — |
| **[I]** invented / untested | **20** | **19** | **−1** |
| **[X]** contradicted | 1 | **0** | **−1** |
| **grounded ([R]+[T]+[O])** | **28 of 49 (57.1%)** | **30 of 49 (61.2%)** | **+2** |

**[A] Three census rows are re-labelled by this audit, and none of them moved because a wave
demoted anything:**

1. **B7 (stale-confirm 2.0×ATR5m): [I] → [T]** — a citation was written into the code
   (`kernel/plan_confirm.go:118-123`, n=2,908 MET confirms).
2. **B6 (min-SL floor): [I] → [R]/[O]** — `kernel/min_sl.go:10-33` carries a full grounding and an
   explicit **owner ruling of 2026-09-02** raising 1.0 → 1.5. See §3.
3. **D9 (swing seats): [X] → [T]-positive — the census misread its own source.** See §4.

**Zero of the nine demotion-queue items were demoted.** The ungrounded share fell from 40.8% to
38.8%, and the contradicted share to zero — by correcting labels, not by changing behaviour.

Counted with `python3` over the census tables (49 rows, first label per row where a row carries a
compound like `[R/O]`); the raw token counts differ (27/15/8/8/3) because compound labels
double-count.

```bash
grep -c 'Status: KnobLive'        store/knob_registry_table.go   # 144
grep -c 'Status: KnobCandidate'   store/knob_registry_table.go   # 16
grep -c 'Status: KnobIneffective' store/knob_registry_table.go   # 7
git log -1 -- docs/superpowers/reports/2026-09-02-belief-census.md
# ee64a494 2026-09-02 08:50:38 -0500
```

---

## 1. PREMISE CORRECTIONS (A17 — measured before anything was concluded)

| # | dispatch said | measured | consequence |
|---|---|---|---|
| P1 | "trade_excursions rows now exist — quote MAE/MFE p50/p80" | **0 rows** (`SELECT COUNT(*) FROM trade_excursions` → 0) | D3 is answered from `trader_positions.mae/mfe` instead, and that substitution is stated wherever a number rests on it |
| P2 | "touch_outcomes=135+" | **359 rows** (candidate_pool 192), first written 2026-09-04 05:30:54 CT; the boot line's 293/168 was the count at 08:30:11 | D5 has more data than the dispatch assumed |
| P3 | "research rounds 1–9 (docs/superpowers/research/…)" | that directory holds **only `INDEX.md`**; every report lives in `docs/superpowers/reports/` | claims attributed to "round N" are cited to a named report or marked *source could not be located* |
| P4 | "the findings ledger §1 of VL-MASTER-PLAN-v2.md if present" | **not present on dev** | — |
| P5 | "the arm rules (D1–D5 of arms-follow-bias)" | **no report by that name exists on dev** | the arm rules are enumerated from code and the missing source is named |
| P6 | "settings registry: live=144" | **correct** — 144 `KnobLive` of 167 entries | the only dispatch premise about counts that held exactly |

---

## 2. D3 — EXITS: the stop floor against the measured pullback

### The floor, resolved

```
09-04 08:30:11  🛑 exits: stop=max(anchor+clr, 1.5×ATR5m) · anchor_max=3.0×ATR5m
                          · BE=off · trail=off · size=1 · re-arm-after-sweep=on (0B)
09-04 08:30:11  🛑 min-sl guard: atr_mult=1.5 level_clearance=2tick(s)
```

`kernel/min_sl.go:34 MinSLATRMultDefault = 1.5` · `MIN_SL_ATR_MULT` is unset in `.env`, so the
resolved value **is** the code default — stated, not assumed.

### The measured distribution

Era = `DayPlanEraStart` 2026-08-15 00:00 CT (`store/attribution.go:131-158`). Closed positions in
era: **n=71**, excluded 4 NULL `pnl_corrected` · 9 `UNRESOLVABLE` · 3 `e7_farside_test`.
Clean set with MAE/MFE present: **n=58**.

| percentile | MAE (pts) | MFE (pts) |
|---|---|---|
| p50 | **32.75** | 25.75 |
| p80 | **49.25** | 69.25 |

**[A] But this MAE distribution CANNOT answer the question the dispatch asks, and reporting it as
if it could would be the error.** MAE on a stopped-out trade is bounded by the stop we placed, so
it measures *our stop placement*, not *the market's pullback*:

| outcome | n | mean MAE | mean realized move | MAE within 2pt of the move |
|---|---|---|---|---|
| LOSER | 38 | 42.01 | 35.85 | **20 of 38 (52.6%)** |
| WINNER | 20 | 15.35 | 56.46 | 1 of 20 |

Over half of losing trades have MAE equal to their own realized move to within 2 points — they
went straight to the stop, and MAE *is* the stop distance. **The MAE p50 of 32.75 sitting almost
exactly on the recorded median `stop_floor_pts` of 33.2 (n=25 planner reads) is what censoring
predicts, not evidence of calibration.**

### The uncensored number, and what it says

The decision-relevant quantity is how much adverse room a **winning** trade actually needed:

| MAE on winners | value |
|---|---|
| n | **18** |
| p50 | **11.5 pts** |
| p80 | **22.5 pts** |
| max | 61.5 pts |

**n=18 is below the pre-registered floor of 30, so NO VERDICT is rendered (A24).** The numbers are
reported as-is. For scale only: the recorded median stop floor is 33.2 pts — **2.9× the winners'
p50 and 1.5× their p80** — and this morning the floor resolved to 44.59 and 50.73 pts against
planner-authored stops of 35.00–38.50.

**Answer to D3 as posed:** the floor sits *outside* the pullback that winning trades experience,
not inside it. The dispatch's framing assumed the risk was a floor too tight; the measured
direction is the opposite. This is stated as a measurement, not a ruling.

### The 1A wave has still not delivered its instrument

`docs/superpowers/reports/2026-09-03-trade-excursions.md` (`git log -1` → `0c1a808c 2026-09-03
00:05:11 -0500`) specified `trade_excursions` precisely so this question could be answered from
path data rather than endpoint data. **The table is still empty**, so this section rests on the
censored substitute. That is the finding, not a footnote.

---

## 3. D1 — the demotion queue, item by item

`docs/superpowers/reports/2026-09-02-belief-census.md:127-137` (`ee64a494`) ranked nine [X]/[I]
beliefs with live teeth. Status at boot 8:

| # | queue item | asked for | status at boot 8 | verdict |
|---|---|---|---|---|
| **1** | min-SL 1.0×ATR5m + 2t, [I], REJECT at write | demote to WARN-first; sweep the multiplier | **still a hard REJECT** (`kernel/engine_position.go:227-231`); multiplier **1.0 → 1.5** — but by an **owner ruling dated 2026-09-02, the census's own day**, with a written citation | **SUPERSEDED, not ignored** — see below |
| 2 | swing seats improve turns, **[X]** | re-test; likely unseat | seats still live (boot: `seats=8`) | see §4 |
| 3 | level-event wakes deserve a re-read, [T]-weak | WARN-first N=25 | boot line: cutoffs `enforce` govern LEVEL_EVENT — but that is the *cadence* cutoff, not the E5 demotion | see agent unit `cadence` |
| 4 | BD_MAX_PULLBACK / BD_CONFIRM_CLOSES, [I] | WARN-first | — | see agent unit `validator` |
| 5 | stale-confirm 2.0×ATR5m, [I] | WARN-first | boot: `stale_confirm=2.0×ATR5m` — **unchanged** | **NOT DONE** |
| 6 | zoneTFMult + ladders, [I] | sensitivity sweep first | boot: `zone-ladder=1.0/0.6/0.3/0.15` — unchanged | see agent unit `levels` |
| 7 | pre-NY sessions carry edge, **[X]**-candidate | owner decision on session enablement | ASIA/LONDON still run (registry `enabled:false` is only the default; the resolver overrides — proven by positions 587 ASIA / 588 LONDON) | **NOT DONE** |
| 8 | consumed/3rd-touch → never entry, [I] | measure react rate | touch telemetry advisory; `touch_outcomes` now n=359 — see §5 | data now exists |
| 9 | killzone/premium-window + Monday/Thursday conviction, [I] | keep advisory, add n counter or delete | prompt lines still present | advisory (no teeth) |

### Item 1 was superseded by an owner ruling — a correction to this audit's first reading

**[A] `kernel/min_sl.go:10-33` is not an uncited number.** It carries the grounding in full:

> Research grounding (week ledger, `pnl_corrected`): **15 of 27 losers were STOPPED-TOO-TIGHT** —
> they printed MFE ≥ 0.5×SL before the stop-out; the 5 biggest losers all stopped 5–44 pts from ANY
> seated level. […] **0B (owner ruling 2026-09-02): 1.0 → 1.5.** The 1.0 was **[C] code-canon with
> no citation** (knob census). **Round-7 research** tests the day-trade range at 1.5–2.5×ATR and
> finds stop-out rates above 60% on noise alone below 1.0×; our own tape has 6 of 8 losers printing
> MAE beyond the stop and 15 of 27 losers stopped-too-tight. **1.5 is the bottom of the researched
> range, not the middle — the deliberate first step from an uncited number to a cited one.**

So the queue's premise ("[I], no sweep") was true of the **1.0**, and the owner replaced it with a
**cited 1.5 on the same day the census was written**. The queue asked for WARN-first *and* a
multiplier sweep; the owner did the sweep and kept the REJECT. **This is the dispatch's "round 7" —
it exists, cited in code at `kernel/min_sl.go:21`, not as a document under `docs/`.**

The same comment names the three readers, which matters for §14: *"This value is read by THREE
gates: the arm-time gate, the AI-entry decision gate and the planner's authoring WARNING."*

### The WARN-first pre-check that exists — and the drift it exposes

A WARN-first pre-check exists and fired four times this morning:

```
09-04 08:09:03 ⚔️ arm feasibility: S1 arm stop 29520.00 too close (38.50 < 50.73 = 1.5×ATR5m)
                 — min-SL gate will refuse it (WARN — write proceeds; the gate-at-arm chain enforces)
09-04 08:09:03 ⚔️ arm feasibility: S2 arm stop 29620.00 too close (35.00 < 50.73 = 1.5×ATR5m) …
09-04 08:09:03 ⚔️ arm feasibility: S4 arm stop 29240.00 too close (38.00 < 50.73 = 1.5×ATR5m) …
09-04 08:25:34 ⚔️ arm feasibility: S2 arm stop 29518.00 too close (36.50 < 44.59 = 1.5×ATR5m) …
```

**[A] This is NOT the demotion the queue asked for.** It is `trader/auto_trader_planner.go:1671-1682`,
shipped by the **london-forensics F4 wave on 2026-08-28** — five days *before* the census ranked
min-SL #1. Its own text says the gate still enforces. The census knew about it and asked for the
gate itself to become WARN-first; that did not happen.

**And it exposes a second-order finding [A]:** the planner is systematically authoring stops
*below* its own floor — 35.00, 36.50, 38.00, 38.50 pts against floors of 44.59 and 50.73. Four
scenarios in one morning were written knowing the gate would refuse them.

---

## 4. D1 (second half) — every [X] belief still enforced

**[A] The answer is NONE — and the reason is that the census's only [X] rests on a misread of its
own source.**

### D9 "swing seats improve turn capture" is NOT contradicted

`belief-census.md:77` labels D9 **[X] research-CONTRADICTED**, citing:

> *"own tape: missed-turns 80.0/75.0/79.2% → 65.0/60.0/66.7% with swing seats (Δ −15pts)"*

The source is `docs/superpowers/reports/2026-08-28-grand-audit.md:73-74` (A17), which reads in full:

> **A17 — Missed-turns refreshed ✅**
> `scripts/leveltruth_missed_turns.py` on live (repaired) bars: baseline 80.0/75.0/79.2% → with
> swing seats 65.0/60.0/66.7% (Δ −15.0/−15.0/−12.5 pts). Reproduces the T3 result independently.
> **PROVEN.**

**`missed-turns` is a MISS rate. Falling from ~80% to ~65% is the seats working, not failing** —
and the source marks it **PROVEN** with a ✅. The census read the negative delta as a degradation
and inverted the finding.

**Corrected label: D9 is [T]-positive**, own tape, Δ −15 pts of *missed* turns. It should stay
seated. The census row needs the correction; the code needs nothing.

### The other [X]-candidate

Queue #7, "pre-NY sessions carry edge", is a **[X]-candidate** in the census's own words (own tape
0/6 −$353.5 pre-NY vs NY 3/3 +$177, weekend-P2 audit) — not a confirmed [X], and at n=6 and n=3 it
is far below any verdict floor. ASIA and LONDON still run. **No verdict (A24)**; it stays on the
drift list as an open owner question, not as a contradicted belief with teeth.

**[A] So: zero [X] beliefs are enforced as a REJECT, a MUST, a weight or an enablement — because
after correction there are no [X] beliefs.**

## 5. D5 — levels and the D1′ detector

### D1′ live rates, with n

`touch_outcomes` n=**359** (first row 2026-09-04 05:30:54 CT — the hook began firing the morning
after the two-day audit predicted it would).

| outcome | n | share |
|---|---|---|
| hold | 137 | 38.2% |
| break | 129 | 35.9% |
| **ambiguous_horizon** | **93** | **25.9%** |

Per level kind, **suppressing every cell below n=30** (A24):

| level_kind | n | hold | break | ambiguous |
|---|---|---|---|---|
| VWAP | 108 | 42 (38.9%) | 23 (21.3%) | **43 (39.8%)** |
| DEMAND | 72 | 36 (50.0%) | 32 (44.4%) | 4 |
| RTH-L | 55 | 16 (29.1%) | 35 (63.6%) | 4 |

15 further kinds hold 124 rows between them, all below n=30 — **no rate is quoted for any of them.**

**Surprise (A23, included not acted on):** VWAP resolves *ambiguous* 39.8% of the time at horizon
H=12. Two of every five VWAP touches never resolve inside the detector's window.

### Is D1′ still DESCRIPTIVE ONLY? — yes [A]

Every production reader of `touch_outcomes` / `candidate_pool` is a **writer**, a **boot counter**,
or a **reporting CLI**:

| reader | file:line | kind |
|---|---|---|
| episode + pool writer | `trader/detector_record.go:50,83,101` | WRITE |
| boot-line counter | `main.go:404` → `store/touch_outcomes.go:233` | COUNT |
| offline report | `cmd/detector-report/main.go:42,44` | CLI |

```bash
grep -rn "TouchOutcomes\|CandidatePool" --include=*.go kernel/ trader/ api/ | grep -v _test
# → detector_record.go (writer), detector_d1prime.go + detector_recorder.go (producers). No gate.
```

**Zero gates, zero weights, zero scorers read either table.** D1′ conforms to "descriptive only".

### A defect the new data confirms

**[A] `touch_outcomes.candidate_seated` is degenerate — all 359 rows carry `1`, none carry `0`.**

```sql
SELECT candidate_seated, COUNT(*) FROM touch_outcomes GROUP BY candidate_seated;  -- 1 | 359
```

The column cannot distinguish a seated candidate from an unseated one, so every future analysis
keyed on it is vacuous. This confirms the two-day audit's C6-F2 finding
(`docs/superpowers/reports/2026-09-04-two-day-audit.md`, `git log -1` → `f3c640c3 2026-09-04
07:26:52 -0500`) against 359 live rows rather than against the code alone.

**And `candidate_pool` records exactly one cut reason:**

| seated | cut_reason | n |
|---|---|---|
| 0 | `max_levels: 12 seated, cap 12` | 97 |
| 1 | *(empty)* | 95 |

Every cut in the window is the seat cap. Any other reason a candidate might be dropped is invisible
in the table.

---

## 6. D10 — expectancy: the data forbids both a promotion and a demotion

**The pre-registered criterion, verbatim** (`2026-09-03-expectancy-1d.md:25-28`, `git log -1` →
`38a63a9b 2026-09-03 15:26:02 -0500`):

> **Floor.** `n < 30` → `DESCRIPTIVE ONLY`, no verdict rendered. At or above it the status is
> computed from the pre-registered rule: **PASSES ⟺ n ≥ MinN ∧ mean > 0 ∧ mean interval excludes 0**,
> else FAILS.

`MinN = 30` — `expectancy/model.go:27`.

**The live boot line, today, both boots:**

```
09-04 07:38:40  📊 expectancy: cells=41 with_n>=30=0 judged_rollups=2 unresolved=3 excluded_test=3
09-04 08:30:11  📊 expectancy: cells=41 with_n>=30=0 judged_rollups=2 unresolved=3 excluded_test=3
```

**[A] D10(a): no cell at n≥30 has a status differing from the criterion, because ZERO of the 41
cells reach n≥30.**

**[A] D10(b): no promotion or demotion is justified — the data forbids both.** Every cell is below
the floor; the criterion renders `DESCRIPTIVE ONLY` and no verdict, and this audit renders none.

### Era expectancy, rebuilt from the store

`pnl_corrected` only; era ≥ 2026-08-15 00:00 CT; `UNRESOLVABLE` and `e7_farside_test` excluded.

| | value |
|---|---|
| n | **58** |
| total | **−$466.43** |
| mean | **−$8.04** |
| sd | $103.98 |
| **95% CI** | **[−$34.80, +$18.72]** |
| win rate | 31.0% |

Against the MC rig (`2026-09-03-mc-drawdown.md`, `git log -1` → `77e1cdfc 2026-09-03 00:39:25
-0500`), whose verdict line reads *"expectancy indistinguishable from zero (CI −$31…+$18, n=64)"*:

- **The CI still straddles zero** — unchanged conclusion.
- **n did not grow; it is LOWER (58 vs 64).** The rig's population and this rebuild's differ by 6
  rows. I could not reconcile the two exclusion sets from the report text, so **the discrepancy is
  reported, not resolved** — one of the two sets is wrong and it needs an owner or a follow-up to
  say which. It does not change the verdict: both straddle zero.

By condition roll-up:

| plan_band | n | Σ | mean | W/L |
|---|---|---|---|---|
| *(none)* | 49 | −$508.93 | −$10.39 | 13/34 |
| `armed_fill` | 9 | +$42.50 | +$4.72 | 5/4 |

Both below n=30 → **DESCRIPTIVE ONLY, no verdict.**

---

## 7. D7 + D11 — arms, and whether a long can trade today

### D7 — the arm-enablement asymmetry, re-measured

The two-day audit (`f3c640c3`) found the planner grants resting arms almost only to shorts.
Re-measured on the live table today:

| trade_date | side | arm-enabled | scenarios | rate |
|---|---|---|---|---|
| 2026-09-02 | long | 9 | 64 | **14.1%** |
| 2026-09-02 | short | 27 | 43 | **62.8%** |
| 2026-09-03 | long | 2 | 28 | **7.1%** |
| 2026-09-03 | short | 8 | 20 | **40.0%** |
| **2026-09-04** | **long** | **1** | **5** | **20.0%** |
| **2026-09-04** | **short** | **1** | **5** | **20.0%** |

**[A] The asymmetry is absent today — 20.0% vs 20.0%, exact parity.** But **n=5 per side**, far
below any verdict floor, so this is a reading, not a finding: one more plan version either way
flips it. **No verdict (A24).**

**A correction to my own prior number [A].** The two-day audit reported 09-03 as long 1/23 and
short 8/18. The live table now reads long 2/28 and short 8/20, because `plans` kept growing after
that snapshot — 09-03 now holds **16 versions**, latest written 2026-09-04 01:20:35 CT (the ASIA
session runs to 02:00). The audit's figure was correct at its snapshot and is superseded here.

### D11 — can a long plan trade today?

**Both routes traced.**

Plans written on 2026-09-04:

| session | v | created (CT) | bias | day_type | lifecycle | long | long armed | short | short armed |
|---|---|---|---|---|---|---|---|---|---|
| LONDON | 1 | 01:41:15 | **long** | **trend** | active | 3 | **1** | 1 | 0 |
| LONDON | 2 | 02:04:14 | short | trend | active | 0 | 0 | 2 | **1** |
| LONDON | 3 | 07:56:17 | short | balance | active | 1 | 0 | 2 | 0 |
| NY | 1 | 08:11:45 | neutral | **no-trade** | `no_trade` | 1 | 0 | 0 | 0 |

**[A] Yes — a long plan CAN arm, and one did.** LONDON v1 at 01:41:15 CT was long-biased,
`day_type: trend`, and carried an arm-enabled long scenario. So the arm route is open to longs;
the two-day audit's asymmetry was a planner *tendency*, not a structural block.

**[A] But nothing traded. Zero `armed_orders` rows and zero positions exist for 2026-09-04.**

```sql
SELECT COUNT(*) FROM armed_orders   WHERE created_at CT >= '2026-09-04 00:00';  -- 0
SELECT COUNT(*) FROM trader_positions WHERE entry_time >= 2026-09-04 05:00 CT;  -- 0
```

Two arm-enabled scenarios were authored and **neither reached the ledger**. The log says why:

```
09-04 07:40:44  ⚔️ arm S1 leg 1 wait_confirm MET (touch) — arming
09-04 07:40:44  ⚔️ arm REFUSED LONDON S1 leg 1: R:R 1.07 below arm min 2.00
                  (studio min_risk_reward_ratio) · rr refusals this session: 1
```

then the confirm re-fired seven more times (07:42:39, 07:45:01, 07:46:39, 07:48:39, 07:50:38,
07:52:38, 07:55:01) without ever producing an arm.

**[A] The binding constraint on the arm path today was the R:R-at-arm floor of 2.00, at R:R 1.07** —
and the two-day audit established that `composeArmStop` widens the stop *before* R:R is scored, so
the floor is applied to a stop the planner did not author. The min-SL floor and the R:R floor
compound: the stop is widened to clear min-SL, which lowers R:R, which then fails the R:R gate.

**Counts since boot 8 (08:30:11 CT) — the window the dispatch asks about is ~13 minutes long:**
0 plans, 0 arms, 0 positions. The D11 answer therefore rests on the full 09-04 day, and says so.

### Log-line counts, 2026-09-04 (whole day so far)

| line | count |
|---|---|
| `⚔️ armed` (a real arm placed) | **0** |
| `arm REFUSED` | 1 |
| `🚦 entry-gate` (arm-path refusal) | 0 |
| `≈armed` (display-only estimate) | 7 |
| `wait_confirm … arming` | 8 |
| `MIN-SL REJECT` (decision path) | 0 |
| `⚔️ arm feasibility` WARN | 4 |

---

## 8. D9 — guardrails: what is actually ON

Live strategy `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` ("MNQ"), bound to trader `hoang`. Resolved
from the saved config at `config.ai_config.risk_control` — **not** `config.risk_control`, which
reads null and would make every value look unset:

| knob | saved value | its own enable flag | master | **effective** |
|---|---|---|---|---|
| `guardrails_enabled` | **false** | — | — | **MASTER OFF** |
| `daily_loss_limit_usd` | 450 | `daily_loss_enabled: false` | off | **INERT (doubly)** |
| `daily_profit_target_usd` | 900 | `daily_profit_enabled: false` | off | INERT |
| `max_daily_trades` | 3 | `max_daily_trades_enabled: false` | off | **INERT (doubly)** |
| `max_contracts_per_order` | 2 | `max_contracts_enabled: false` | off | INERT |
| `notional_cap_enabled` | false | — | off | INERT |
| `blackout_enabled` | false | — | off | INERT |
| `min_risk_reward_ratio` | **2** | — | master-independent | **LIVE — both seams** |
| `min_confidence` | **60** | — | master-independent | **LIVE** |
| `max_positions` | 3 | — | — | live |
| `hold_discipline` | **true** | — | — | live |
| `breakeven_enabled` | **true**, trigger **40 pts** | — | — | **OVERRIDDEN OFF by 0B** |
| `trailing_enabled` | **true**, `atr_period: 14` | — | — | **OVERRIDDEN OFF by 0B** |

Boot 8 confirms the override: `🛑 exits: … BE=off · trail=off · size=1 (0B)`.

### Against the Monte Carlo rig

`docs/superpowers/reports/2026-09-03-mc-drawdown.md` (`git log -1` → `77e1cdfc 2026-09-03 00:39:25
-0500`), verdict line: *"expectancy indistinguishable from zero (CI −$31…+$18, n=64); maxDD@50 p95
$1,677; the 3-trade cap forfeits $24.54/day, the $450 limit is inert."*

| rig said | resolved now | conformance |
|---|---|---|
| "the $450 limit is **inert**" | `daily_loss_limit_usd: 450` · `daily_loss_enabled: false` · master off | **CONFORMS** — and it is inert *twice over* |
| "the 3-trade cap **forfeits $24.54/day**" | `max_daily_trades: 3` · `max_daily_trades_enabled: false` · master off | **DOES NOT CONFORM as framed** — the cap is not enforced at all, so it forfeits **nothing**. The $24.54/day is the cost the cap *would* impose if switched on, not a cost being paid |

**[A] This corrects the rig's framing, which reads as though the cap were active.** A sizing ruling
taken from that line would be reasoning about a constraint that is not in force.

**[A] Even the shadow counter is silent:** `kernel/engine_analysis.go:177` emits
`🔍 guardrail WOULD have tripped (master OFF, not enforced)` and records
`guardrail_would_trip` telemetry. It fired **0 times** on 2026-09-04.

The two-day audit's finding that position 590 (−$99.00) "opened through" `max_daily_trades=3` is
**re-verified at boot 8** and is more precisely stated here: the cap did not merely fail to
enforce because the master was off — its own `max_daily_trades_enabled` is `false` as well.

### What REMAINS enforced regardless of the master

Per the boot WARN, master-independent venue safety: the futures **notional×N ceiling** and the
**per-order contract clamp**, plus `isAccountTradeable` (SIM-only). Position size is pinned at
**1 contract** by 0B (`size=1`), which is below the `max_contracts_per_order: 2` that is itself
disabled — so the binding size constraint today is the 0B wave, not any guardrail.

---

## 9. D6 — bias: does anything MUST-read it?

**(a) Weekly refs-only.** Branch `fix/weekly-refs-only` exists (`1cee77a8`); the weekly reader's
live surface is covered in the agent unit below. Boot 8 shows the weekly read was skipped as fresh:
`📅 WEEKLY READ skip-fresh — week 2026-08-31 doc already stored (v1), idempotent.`

**(b) Every production reader of `Bias.Direction`, classified:**

| reader | file:line | effect | live? |
|---|---|---|---|
| EntryGate **leg 1** — "entry against plan bias" | `trader/entry_gate.go:176-184` | REJECT | **INERT — guarded by `if in.PlanMode == "direction"`, and the live `plan_mode` is `strict`** |
| plan-config direction check | `trader/auto_trader_planconfig.go:210-215` | refusal string | same mode guard |
| **flip-fired re-plan enforcement (P0.4-G)** | `trader/auto_trader_planner.go:1635-1637` | **REJECT at plan write** — *"prior plan flip already fired → bias %s is MANDATORY, got %q — the flip cannot be re-written away"* | **LIVE** |
| arms bias-coherence | `kernel/arms_bias_coherent.go:74 BiasArmWarning` | returns a **warning string** | live, **warn only** ✓ |
| prompt render | `kernel/plan_render.go:157`, `kernel/planner_prompt.go:324` | text | label |
| doc validation | `kernel/plan_doc.go:602` | enum check (long/short/neutral) | [M] mechanics |

**[A] The honest answer is not "no MUST reads bias" — one does.** `auto_trader_planner.go:1635` is
a live REJECT keyed on `Bias.Direction`. Its content, though, is a **flip-consistency** rule, not a
directional-edge claim: it refuses a re-plan that contradicts a flip the machine already fired on
bars. Label **[O]** (it was written to close a live bug — ASIA v3's flip fired "→ bias long" and v4
came back short). It does **not** assert that a direction is profitable, so it does not conflict
with the calibrations.

**(c) The tree as facts.** `kernel/planner_prompt.go:213` states the contract in the code itself:
*"The plan bias is a LABEL, not a direction"*. Consistent with (b).

**Conformance:** the directional-edge gate (leg 1) is **inert under `strict`**, and the bias
calibrations are therefore not contradicted by any live gate. **CONFORMS**, with the one MUST named
above disclosed rather than hidden.

---

## 10. D8 — cadence

**Resolved at boot 8:**

```
⏱ wakes: cutoff=25m(enforce) cooldown=30m(enforce, fast-market≥1.5×ATR exempt)
          cross-session=on stale-arm-expiry=on (class 47)
          — cutoffs govern LEVEL_EVENT/structure_mss wakes ONLY
```

**Measured on 2026-09-04 (whole day so far):**

| event | n |
|---|---|
| `waking the planner` (fired) | **2** |
| `SKIPPED: … wake_min_interval_min` | **11** |
| `⏱ wake would_skip` (class-47 WARN) | **0** |
| `⏱ wake cooldown` | **0** |
| `⏱ wake cutoff` | **0** |
| scheduled_read triggers | 2 |
| level_event triggers | 2 |
| structure MSS | 0 |

**[A] The class-47 cutoffs suppressed nothing today** — the same result the two-day audit found for
09-02 and 09-03. Across three consecutive days the enforcing cutoff has produced **zero**
suppressions; the throttle that actually shapes cadence is `wake_min_interval_min` (11 skips today).

### Stage-4 status — answered precisely

`docs/superpowers/reports/2026-09-03-wake-predicate.md` (`git log -1` → `586261ed 2026-09-03
21:36:39 -0500`), title line 1: *"step 2: the 1B wiring (steps 3-4 gated on a boot)"*, line 7:
*"STEP 2 of 4 COMPLETE (1B wiring). Branch `fix/wake-predicate` @ `f1d7cf51`, NOT deployed."*

| step | status now |
|---|---|
| 1–2 (1B wiring) | **COMPLETE and DEPLOYED** — `f1d7cf51` is an ancestor of `origin/dev`; `touch_outcomes` holds 359 rows, so the hook fires |
| 3 (boot it, accumulate a session of data) | **precondition now satisfied by events** — boot 8 ran and 359 outcomes / 192 pool rows accumulated |
| 4 (**the change-based predicate itself**) | **NOT IMPLEMENTED** — `grep -rn "changeBased\|change_based\|ChangePredicate\|materialChange\|wakePredicate" --include=*.go .` (excluding tests) returns **nothing** |

**[A] The change-based predicate does not exist in production code.** The wave's own report says
the throttle is the scheduler and the predicate is a later wave; that is still true at boot 8.

### E5 (queue item 3) — was the level-event wake demoted?

Census E5 (`belief-census.md:87`): *"[T] weak: 52 re-plans/7 days, 7 ever armed"*, demote to
WARN-first N=25. **[A] NOT DONE** — `level_event` remains a **full REPLAN trigger** and is on the
**budget-free** list at boot 8 (`replan budget: … free: <S>_scheduled_read, level_event,
structure_mss …`). It fired twice today. The class-47 cutoffs govern *when* it may wake, not
whether it is advisory.

---

## 11. D2 — every [I] belief still enforced as a REJECT

The WARN-first law says an untested belief must not carry a hard REJECT without an owner ruling.
The census named three candidates (`belief-census.md:107`). Status at boot 8:

| census row | belief | still a REJECT? | label now | verdict |
|---|---|---|---|---|
| **B6** | min-SL ≥ mult×ATR5m + 2-tick clearance | **YES** — `kernel/engine_position.go:227-231` `return fmt.Errorf` | **[R]/[O], NOT [I]** — `kernel/min_sl.go:10-33` carries the week-ledger grounding *and* names the **owner ruling of 2026-09-02** that set 1.5 | **NOT a violation.** The census row describes the superseded 1.0 |
| **B2** | breakdown family: `BD_MIN_CLOSES`, `BD_MAX_PULLBACK`, `BD_MAX_LEVEL_DIST` | **YES** — `kernel/breakdown_continue.go:258-265` author-time `return fmt.Errorf` | partially [T] | **PARTIALLY MOVED — not by demotion.** E3 (entry-mechanics 2026-08-30) relaxed the confirm requirement **2 → 1** close: *"the entry law now rides on displacement quality, not on a double close"* (`breakdown_continue.go:60-64`). The census's "BD_CONFIRM_CLOSES 2" no longer exists; `BD_MIN_CLOSES` defaults to **1** |
| **B7** | stale-confirm 2.0×ATR5m | YES — gate | **[T], NOT [I]** | **THE CENSUS LABEL IS NOW WRONG.** The code carries its own citation (`kernel/plan_confirm.go:118-123`): *register S2, mega-research 2026-08-26 — the shipped 1.0×dATR unit marked only **38/2,908 (1.3%)** of the week's MET confirms stale; at 2.0×ATR5m the rule marks ~37%, matching the empirical stale mass (median \|price−ref\| = 58.75 pt)*. **n=2,908.** This is a tape-calibrated value, not an invention |

**[A] D2's answer: ZERO.** No [I] belief carries a hard REJECT without either a citation or an
owner ruling. All three census candidates dissolve on inspection:

- **B6** was owner-ruled on 2026-09-02 with a written citation (the census's own day — the row
  describes the value the ruling replaced).
- **B7** is tape-calibrated on n=2,908 and should be re-labelled **[T]**, not demoted.
- **B2** moved by E3's displacement-quality reasoning, not by an untested invention.

**This corrects this audit's own first reading, which recorded B6 as an A24 violation.** The
WARN-first law is not being broken by any live REJECT. What *is* broken is narrower and is on the
drift list: the **planner prompt still states the superseded 1.0× floor** while three gates enforce
1.5× (drift D-2).

---

## 12. D4 — the entry law, per condition

**[A] The law is ONE enum-keyed table at one chokepoint** — `kernel/entry_law.go:38-77`,
enforced by `ValidateEntryLaw` (`:133`) as a **REJECT AT WRITE with a NAMED message**.

| condition | allowed confirms | fade-touch enforced | style |
|---|---|---|---|
| `reject` | **touch only** | ✅ | touch-entry at the level (limit), stop ≥2 ticks behind structure |
| `fvg_entry` | **touch only** | ✅ | touch inside the FVG (edge..CE band, vs the FRESH list) |
| `sweep_reclaim` | touch · 1x5m_close · 1m_mss | — | split contract (E4): leg-1 touch, leg-2 1m_mss |
| `reclaim` | 1x5m_close · 1m_mss | — | reclaim-close discipline, **never 2x5m_close** |
| `breakout_retest` | touch · 1x5m_close | — | retest limit + stop-entry fallback (E7) |
| `acceptance` | time_hold · 1x5m_close | — | time_hold (E6) with 1x5m fallback |
| `hold` | time_hold · 1x5m_close | — | as `acceptance` |
| `breakdown_continue` | 1x5m_close · **2x5m_close** | — | 1 confirming close + displacement ≥ `BD_MIN_DISP_ATR`×ATR5m, or stop-entry |
| `breakup_continue` | 1x5m_close · **2x5m_close** | — | as above |

Two named rejections carry the law:

- **`fade_requires_touch`** — `entry_law.go:151-155`: a close-confirm on `reject`/`fvg_entry` is
  always refused. **LIVE, REJECT-at-write.** Declared to the model as a prompt contract restriction
  (`kernel/prompt_contract.go:121-123`, `MustAppear: "touch ONLY (fade_requires_touch)"`).
- **`2x5m_reserved`** — `entry_law.go:157`, `twoX5mReserved()` at `:87`: a double close is legal
  **only** on the breakdown/breakup pair. Also a declared contract restriction
  (`prompt_contract.go:126-127`).

**Owner ruling in the code (`entry_law.go:22`):** *"no 15m confirms ever (E1); default 1x5m_close"*
— label **[O]**.

### Against the confirm-cost research

`docs/superpowers/reports/2026-08-30-confirm-cost-forensics.md`
(`git log -1` → `8f09aa84`) measured close-confirms at a net cost of **≈ −$681**. The live law is
directionally consistent with that finding: the two **fade** conditions are forced to `touch`
(the cheap confirm), and the expensive `2x5m_close` is confined to the two displacement
conditions. **CONFORMS.**

**The dispatch's "round 3 (touch vs 2x5m cost)" source could not be located** — there is no
`round 3` document on dev (premise P3). The confirm-cost forensics report is the nearest named
measurement and is what this row is checked against.

---

## 13. THE C TABLE — every live rule, grouped by subsystem

Resolved values are from the boot-8 lines or the resolver path. `report:line` grounds the label.
`callers` is A29: a rule with 0 production callers is DEAD.

### 13.1 Entry gate (`trader/entry_gate.go`)

| leg | rule | file:line | resolved now | label | grounding | effect | CONFORMS | callers |
|---|---|---|---|---|---|---|---|---|
| 0 | strict — plan scenarios execute on the ARM path only | `:162-172` | `plan_mode = strict` (saved on strategy `a5b7662e`) | [O] | knob census / owner | **REJECT** — refuses every decision-path market entry | **NO** — see drift D-1 | 2 (`auto_trader_orders.go:333`, `armed_executor.go:495`) |
| 1 | entry against plan bias | `:176-184` | **inert** — guarded by `PlanMode=="direction"`, live mode is `strict` | [I] | census A-none | REJECT (when direction) | yes (inert) | 2 |
| 2 | class-48 scenario-direction mismatch | `:186-196` | live | [O] | class 48 | REJECT | yes | 2 |
| 3 | invalidation-wired | `:199-220` | `invalidation-wired=on`, **ARM PATH ONLY**; unresolved ⇒ leg PASSES | [O] 2026-09-03 | boot line | REJECT (arm) | yes | 2 |
| 4 | shadow (0C) | `:234` | shadow = `breakout_retest`, `fvg_entry` | [R] | fvg-entry-model → 0C demotion (INDEX.md) | REJECT if cited | yes | 2 |
| 5 | R:R at execution price | `:257` | floor **2.0** (saved) — the **researched** value is **3.0** | [T]/[O] | census B8 (n=18 +$994) vs `store/strategy.go:71-76` "the RESEARCHED value" + `VL-DAYPLAN-FULL-SPEC.md:49` "R:R≥3" | REJECT | **NO** — drift D-21 | 2 |
| 6 | min-SL at execution | `:268` | `1.5×ATR5m` + 2 ticks | **[R]/[O]** | `kernel/min_sl.go:10-33` (round-7 + week ledger; owner ruling 2026-09-02) | REJECT | yes — **but the prompt still says 1.0×**, drift D-2 | 2 |
| 7 | one_open_position | `:280` | ON, hardcoded, no knob | [O] 2026-09-03 | owner ruling | REJECT | yes | 2 |
| — | no-chase callback | `trader/no_chase.go:145` | `max_dist=1.00×ATR max_run=1.5×ATR5m` **mode=warn** | **[I]** *(self-declared "PROVISIONAL" in the boot line)* | boot line | **WARN-only** | yes — warn matches its label | 1 |

### 13.2 Arms (`trader/armed_executor.go`)

| rule | file:line | resolved | label | effect | CONFORMS | callers |
|---|---|---|---|---|---|---|
| arm R:R floor | `:415` `armGateVerdictFor` | **2.0** | [T] census B8 (n=18) | gate | yes | 1 |
| `armGateVerdict` (legacy shape) | `:1268` | — | [M] | — | **DEAD — 0 production callers**, all 8 sites in `armed_executor_test.go` | **0 — DEAD** |
| place band | `:935` | `100t` | [I] | placement window | unlabelled | 1 |
| stale-working reaper | `:1019-1034` | **15m** | [I] | cancels a working arm | see drift D-5 | 1 (`:940`) |
| marketable wrong-side guard | `:918-924`, `:947-960` | on | [R] 08-30 incident (census B9) | cancel-before-place | yes | 1 |
| UpsertArm re-authorize on version bump | `store/armed_orders.go:155,212` | on | [R] 08-30 incident (census B10) | ledger | yes | — |
| arm normalizer (class 39) | boot line | legs on non-sweep → single arm + WARN | [O] | WARN | yes | — |
| bias coherence | `kernel/arms_bias_coherent.go:74 BiasArmWarning` | returns a warning string | [M] | **WARN-only — but 0 production callers; nothing warns** | **NO** — drift D-18 | **0 — DEAD** |
| re-arm after sweep | boot: `re-arm-after-sweep=on` | on | [O] 0B | lifecycle | yes | — |

### 13.3 Exits / 0B

| rule | file:line | resolved | label | grounding | effect | CONFORMS |
|---|---|---|---|---|---|---|
| stop composition | `main.go:335` (boot) | `max(anchor+clr, 1.5×ATR5m)`, `anchor_max=3.0×ATR5m` | **[R]/[O]** | `kernel/min_sl.go:10-33` | REJECT/compose | yes |
| breakeven | boot: `BE=off` | **OFF** | **[O]** | census E1 (owner-ruled ON at +40pt) | exit | **NO** — drift D-3 |
| trailing | boot: `trail=off` | **OFF** | **[O]** | census E2 (owner-ruled 2.0×ATR14) | exit | **NO** — drift D-3 |
| position size | boot: `size=1` | 1 contract | [O] 0B | sizing | yes |
| EOD flat | `kernel/session_registry.go:78-82` | NY 14:45 CT | [O] | census E3 | gate | yes |
| flip/death → dormant + auto re-arm | boot | on | [O] | census E4 | lifecycle | yes |

### 13.4 Validator / entry law

| rule | file:line | resolved | label | effect | CONFORMS |
|---|---|---|---|---|---|
| per-condition entry law (9 conditions) | `kernel/entry_law.go:38-77`, enforced `:133` | see §12 | [O] `:22` | **REJECT at write** | yes |
| `fade_requires_touch` | `:151-155` | live | [O] | REJECT | yes |
| `2x5m_reserved` | `:157`, `:87` | live | [O] | REJECT | yes |
| breakdown displacement | `kernel/breakdown_continue.go:265` | `BD_MIN_DISP_ATR` | [T] census B1 (waterfall replay +$243) | REJECT | yes |
| breakdown confirming closes | `:60-66` | **`BD_MIN_CLOSES = 1`** (was 2) | [T] E3 2026-08-30 | REJECT | yes — value moved with a stated reason |
| breakdown reclaim voids | `:258` | live | [O] census B3 | REJECT | yes |
| stale-confirm | `kernel/plan_confirm.go:124` | **2.0×ATR5m** | **[T]** — in-code citation, n=2,908 | gate | yes — **census label [I] is now stale** |
| prompt/validator contract | `kernel/prompt_contract.go` | **19 restrictions**, all stated in prompt | [M] class 38 | REJECT on drift | yes |

### 13.5 Levels / detector

| rule | resolved | label | effect | CONFORMS |
|---|---|---|---|---|
| seats / `max_levels` | boot prints **8** (`DefaultMaxLevels`, a file constant) — **resolved value is 12** (`day_plan.max_levels`); live log shows `🗺️ seated 12/360` | [I] | filter | **NO** — drift D-16 (the boot line does not read the resolver) |
| proximity band | boot prints *"retuned 0.3"* — **that is a string literal in the format string** (`kernel/levels_volume_boot.go:13`). **Resolved `proximity_filter_atr` = 1.0** | [O] (P2 keep) | filter | **NO** — drift D-17 |
| zone ladder | `1.0/0.6/0.3/0.15` | [I] | weight | pending sweep (queue #6) |
| family confluence | `cap=3` | [I] | weight | pending |
| swing seats | seated | **[T]-positive** — `grand-audit.md:73-74`: missed-turns 80.0→65.0%, marked **PROVEN** | weight | **yes** — the census's [X] was a misread (§4) |
| roles (consumed/3rd-touch → target_only) | `roles=on(overrides=false)` | [I] | role demotion | queue #8, data now exists |
| touch telemetry | `band=16t max_bars=12 approach=5` | [I] | **advisory, zero gates** | yes |
| **D1′ detector** | `k=3 H=12 exit_on=close`; 359 outcomes / 192 pool | [M] | **descriptive only — 0 gate readers** | **yes** (§5) |

### 13.6 Cadence / wakes · Guardrails · Sessions

| rule | resolved | label | effect | CONFORMS |
|---|---|---|---|---|
| wake cutoff | **25m (enforce)** | [O] class 47 | gate on LEVEL_EVENT/MSS | yes — but **0 suppressions in 3 days** |
| wake cooldown | **30m (enforce)**, fast-market ≥1.5×ATR exempt | [O] | gate | yes — 0 firings today |
| `wake_min_interval_min` | 30 | [I] | the real throttle — **11 skips today** | unlabelled |
| level_event as a replan trigger | budget-**free**, full REPLAN | **[T]-weak** census E5 (52 re-plans/7d, 7 armed) | REPLAN | **NO** — drift D-6 (queue #3 not done) |
| change-based predicate (stage 4) | **not implemented** | — | — | **NOT STARTED** (§10) |
| guardrails master | **OFF** | [O] | — | **NO** — drift D-7 |
| daily loss $450 / daily profit $900 / max_daily_trades 3 / contracts 2 / notional / blackout | all **`*_enabled: false`** *and* master off | [O] | inert | matches the MC rig's "inert"; **corrects its "forfeits $24.54/day"** |
| `min_confidence` | **60** (saved) | [O] | REJECT | yes |
| sessions ASIA/LONDON/NY | 17:00→02:00 / 02:00→08:30 / 08:30→14:45 CT | [O] | gate | yes |
| **pre-NY sessions carry edge** | ASIA+LONDON run | **[X]**-candidate (0/6 −$353.5 vs NY 3/3 +$177) | enablement | **NO** — drift D-8 |
| no-trade band | first 5m · lunch 12:00–13:30 CT (code constants) | [O] | gate | yes |
| void scope | session-day window · 1m×2000 · **one resolver for prompt AND validator** | [M] | parity | yes |

---

## 14. THE DRIFT LIST

Every rule whose resolved value or label differs from what the research or the ruling says it
should be. **No ruling is made here — this list is the input to them (dispatch stop-line).**

| # | rule | what the research / ruling says | what is live at boot 8 | fix owner |
|---|---|---|---|---|
| **D-1** | `plan_mode = strict` + EntryGate leg 0 | no research supports closing the decision path; the two-day audit (`f3c640c3`) found it refuses **every** decision-path market entry regardless of citation, and announces this nowhere | `strict` saved on strategy `a5b7662e`; 13 refusals on 09-03 20:35–21:12 CT | **ruling** (keep strict, or make the arm path carry the load explicitly) |
| **D-2** | **the planner prompt states the SUPERSEDED min-SL floor** | three gates enforce **1.5×ATR5m** (`kernel/min_sl.go:34`, owner ruling 2026-09-02) | `kernel/planner_prompt.go:733` still instructs the model *"the stop distance must be ≥ **1.0×** the current 5m ATR"* — **the planner authors to a floor 50% below the one that judges it.** This is the direct cause of the four `arm feasibility` WARNs on 2026-09-04 (authored stops 35.00–38.50 vs floors 44.59–50.73) | **prompt** |
| **D-2b** | the prompt cites a deleted env knob | owner ruling R1 (2026-09-03) **deleted `ARM_MIN_RR`** | `planner_prompt.go:733` still says *"must be ≥ 2.0 (ARM_MIN_RR)"* | **prompt** |
| **D-3** | breakeven / trailing | census **E1/E2 are [O] owner-ruled ON** — BE at +40pt, trail 2.0×ATR14; the saved strategy still carries `breakeven_enabled: true`, `trailing_enabled: true` | boot: **`BE=off · trail=off`** (suspended by the 0B wave). Two boot lines contradict each other on trailing | **ruling** (which of the two owner positions stands) |
| **D-4** | **the census's D9 label** | `grand-audit.md:73-74` marks swing seats **PROVEN**: missed-turns 80.0/75.0/79.2% → 65.0/60.0/66.7% (a MISS rate falling = better) | `belief-census.md:77` labels it **[X] research-CONTRADICTED**, reading the fall as a degradation | **doc** (correct the census; the code is right) |
| **D-5** | stale-working reaper | two-day audit: a **healthy** resting limit is reaped ~15 min after placement because `onArmedOrderUpdate` has no default branch and no `Touch` for `submitted/accepted/working` | `stale_working=15m`, unchanged | **code** |
| **D-6** | level_event as a full REPLAN trigger | census **E5 [T]-weak** (52 re-plans / 7 days, **7 ever armed**); queue **#3**: demote to WARN-first N=25 | full REPLAN, **budget-free**; fired twice today | **code** |
| **D-7** | guardrails | MC rig (`77e1cdfc`): *"the 3-trade cap forfeits $24.54/day"* — framed as an active constraint | master **OFF** and every limit's own `*_enabled` is **false** — the cap forfeits **nothing** | **ruling** (turn on, or restate the rig's finding) |
| **D-8** | pre-NY sessions | census queue **#7**, [X]-candidate: own tape 0/6 −$353.5 pre-NY vs NY 3/3 +$177 | ASIA + LONDON both run; today's only two arm-enabled scenarios were **both LONDON** | **ruling** |
| **D-9** | census label for stale-confirm (B7) | census says **[I]** | the code carries a tape citation, **n=2,908** (`plan_confirm.go:118-123`) — it is **[T]** | **prompt/doc** (correct the census label; no code change) |
| **D-10** | `armGateVerdict` | — | **DEAD: 0 production callers**, 8 test-only sites (`armed_executor.go:1268`) | **code** (delete or wire) |
| **D-11** | `touch_outcomes.candidate_seated` | 1B intended it to separate seated from unseated candidates | **all 359 rows = 1**; the column is degenerate (`trader/detector_record.go:66`) | **code** |
| **D-12** | `trade_excursions` | 1A wave (`0c1a808c`) specified the table so exits could be judged on path data | **0 rows**; `BackfillExcursions` has no automatic trigger | **code** |
| **D-13** | decision-path EntryGate refusal | — | writes **no log line and no `arm_refusals_0b` store counter**, so 19 refusals were invisible until `decision_records` was read. **Correction to the two-day audit (and to this report's first draft): it DOES write a telemetry counter** — `entry_gate.go:478` calls `telemetry.IncGateBlock(at.id, "entry_gate")`, readable at `/api/risk/gate-blocks` (auth-gated, so unverifiable from this session) | **code** (the log line) |
| **D-14** | no-chase | its own boot line calls it **`[I] PROVISIONAL`** and promises "the week of counts is the research" | still `mode=warn`; on the arm path it is **structurally incapable of firing** (two-day audit: 40 evaluations, all `dist=0.00×ATR`) | **code** |
| **D-16** | seat count | the resolver returns `day_plan.max_levels` = **12** (live log: `🗺️ seated 12/360`) | the boot line prints `seats=8` — `DefaultMaxLevels`, a file constant, never the resolver (`kernel/levels_volume_boot.go:13-14`) | **code** (boot line must READ, per A24) |
| **D-17** | proximity band | census D7 / weekend-audit-p1: **0.3** ("Proximity @0.3 PROVEN live") | resolved `proximity_filter_atr` = **1.0**; the boot line's "retuned 0.3" is a **string literal**, not a read | **code** (the line) + **ruling** (which value stands) |
| **D-18** | arm bias-coherence | the boot line prints `bias-coherent=warn` and the checklist records `BiasArmWarning` as shipped | **`BiasArmWarning` has 0 production callers — nothing warns**, and the boot line's "warn" is a hardcoded literal | **code** |
| **D-19** | HTF veto | census C1 + weekend-audit-p2 rule KEEP: "1h AND 4h agree blocks counter-trend entries" | reported by the agent unit as a **gate that cannot fire** (0 of 6,036 production evaluations) — **flagged for verification, not asserted here** | **code** (pending confirm) |
| **D-20** | "ARMS FOLLOW THE BIAS … a long plan with no long arm is invalid" | stated as **law** in the prompt (`kernel/planner_prompt.go:730`) | **no validator and no warn enforces it**; it is not among the 19 prompt-contract restrictions | **code or prompt** (enforce it, or stop calling it invalid) |
| **D-21** | R:R floor | `VL-DAYPLAN-FULL-SPEC.md:49` "hard gates (**R:R≥3**…)"; `store/strategy.go:71-76` names 3.0 "the RESEARCHED value" | resolved **2.0**, an explicit owner value (the code permits it: *"an owner who deliberately configures 1.5 still gets 1.5"*) | **ruling** (2.0 is a choice, but it sits below the researched bar and nothing records why) |
| **D-22** | weekly F3 invalidation + F4 DOA guard | `belief-census.md:94-95` labels both **live [O] gates** | reported by the agent unit as **0 production callers — test-only** | **code** (pending confirm) |
| **D-23** | 1D expectancy deployment status | the report header (`2026-09-03-expectancy-1d.md:4`) says **"BUILT, GREEN, NOT DEPLOYED — the cutover needs the owner's GO"** | it **is** deployed and emitting at `main.go:423` in boot 8 | **doc** |
| **D-24** | the 1D report is absent from the research ledger | `research/INDEX.md` carries the MC-rig row | no row for `2026-09-03-expectancy-1d.md` | **docs** |
| **D-15** | expectancy vs the MC rig | rig: **n=64**, CI −$31…+$18 | this rebuild: **n=58**, CI −$34.80…+$18.72 — six rows apart, exclusion sets could not be reconciled from the report text | **doc** (name the canonical set) |

---

## 15. THE KEEP LIST — what conforms

One line each. These were checked and need nothing.

- **Entry law, all 9 conditions** — one enum-keyed table, one chokepoint, named rejections
  (`entry_law.go:38-77,133`). Fades forced to `touch`, `2x5m_close` confined to the displacement
  pair — directionally consistent with the −$681 confirm-cost measurement.
- **`fade_requires_touch`** and **`2x5m_reserved`** — live REJECTs, both declared to the model as
  prompt-contract restrictions (`prompt_contract.go:121-127`).
- **Prompt/validator contract** — 19 restrictions, all stated in the prompt, class-38 guarded.
- **R:R floor 2.0** — [T] with n=18 (+$994), saved explicitly rather than defaulted, and applied at
  both seams.
- **Marketable wrong-side guard** and **UpsertArm version-bump re-authorization** — both [R] from
  the 08-30 incident; both still exactly as the incident required.
- **Breakdown displacement floor** — [T] from the waterfall replay (+$243 would-have).
- **Stale-confirm 2.0×ATR5m** — tape-calibrated on n=2,908; only its census *label* is stale.
- **D1′ detector** — descriptive only, exactly as specified: 0 gate readers, verified by a
  method-level grep across `kernel/`, `trader/`, `api/`.
- **Void scope** — one resolver for prompt AND validator (parity), session-day window, 1m×2000.
- **No-trade band** — first-5m and lunch 12:00–13:30 CT, one code constant shared by gate, grader
  and card.
- **Session windows and the NY 14:45 EOD flat** — owner contract, matched by the running registry.
- **Bias as a label** — the directional gate (leg 1) is inert under `strict`; the one live
  bias-reading REJECT enforces flip *consistency*, not a directional edge.
- **`min_confidence` 60** — saved, matches the shipped default, enforced.
- **Arm bias-coherence** — returns a warning string, never a refusal: warn matches its label.
- **`one_open_position`** — owner-ruled 2026-09-03, hardcoded, both seams.
- **Invalidation-wired** — owner-ruled, arm path only, unresolved verdict passes with a line.

---

## 16. WHAT THIS AUDIT GOT WRONG, AND WHY IT IS IN THE REPORT

Three findings in this report's own first draft were wrong. Each was caught by an adversarial
re-read, each is corrected in place above, and each is listed here because the *pattern* matters
more than the individual error.

| my first reading | what is true | why I got it wrong |
|---|---|---|
| "min-SL is an [I] belief enforced as a REJECT — an A24 violation" (D2 = 1) | it is **[R]/[O]** — `kernel/min_sl.go:10-33` carries a week-ledger grounding and names an **owner ruling of 2026-09-02**. **D2 = 0** | I read the census row and the boot line, and did not read the constant's own doc comment. The citation lives in code, not in `docs/` |
| "D9 swing seats is [X] with live teeth" | the census **misread its source**: `missed-turns` is a MISS rate, 80.0→65.0% is an improvement, and `grand-audit.md:74` marks it **PROVEN ✅** | I trusted the census's label instead of opening the report it cited |
| "seats=8, proximity retuned 0.3 — conforms" | resolved values are **`max_levels`=12** and **`proximity_filter_atr`=1.0**; the boot line prints a file constant and a **string literal** | I treated a boot line as a resolved read. The repo's own canon says *"Boot lines are READ, never literal"* — two of them are not |

**The common thread: a label or a boot line was trusted in place of the thing it describes.** That
is the same failure mode the two-day audit hit (concluding "the gate refused nothing" from a grep),
and it is why every row of §13 now carries a `file:line` and a resolved value rather than a
citation to a prior report's summary.

**One correction propagates backwards:** the two-day audit's defect D32 said the decision-path
EntryGate refusal writes "no log line and no counter". It writes no log line and no
`arm_refusals_0b` row, but it **does** call `telemetry.IncGateBlock` (`entry_gate.go:478`). That
report's finding is half right and is corrected here rather than left standing.

---

## 17. LIMITS

- **[A]** `/api/config/resolved` and `/api/risk/gate-blocks` require an Authorization header this
  session does not have. Every resolved value above comes from a boot line, the resolver code path,
  or the saved strategy row — and where a boot line proved to be a literal (§16) the resolver was
  read instead.
- **[A]** `trade_excursions` is empty, so D3 rests on `trader_positions.mae/mfe`, which is
  **right-censored by our own stops** for losing trades. The winners-only MAE (n=18) is the
  uncensored signal and is **below the n=30 floor** — reported without a verdict.
- **[A]** The live tables move while the audit runs: `touch_outcomes` was 359 at the start and 424
  twenty minutes later; `plans` for 09-03 grew from 14 versions (two-day audit) to 16. Every count
  in this report carries the instant it was taken.
- **[A]** Four dispatch premises could not be sourced: "rounds 1–9" as a directory, `VL-MASTER-PLAN-v2.md`,
  the "arms-follow-bias" report, and "round 3: 88% of breaks resolve next candle" (three `88%` hits
  in `docs/superpowers/`, none about breaks). **Round 7 *was* found** — cited in code at
  `kernel/min_sl.go:21`, not as a document.
- **[B]** Drift rows **D-19** (HTF veto as a dead gate) and **D-22** (weekly F3/F4 as dead gates)
  come from agent units whose adversarial verification had not returned when this report was
  merged. They are marked *pending confirm* rather than asserted, and neither is acted on.

---

## 18. THE DEMOTION QUEUE — one line each (owner-requested)

Read-only. **The owner rules per line; nothing below is a decision.** "What could get through"
means: what the system refuses **today** that a demotion would let reach the tape.

| # | belief | label **now** | live effect today | demotion means concretely | what could get through |
|---|---|---|---|---|---|
| **1** | min-SL ≥ **1.5**×ATR5m + 2-tick clearance | **[R]/[O]** — owner-ruled 2026-09-02, cited `kernel/min_sl.go:10-33` (census's [I] describes the retired 1.0) | **REJECT** at three gates (arm-time, AI-entry, planner WARN) | → **WARN + count**: author it, log it, place it, tally the outcomes | **stops between 1.0× and 1.5×ATR5m** — exactly the **35.00 / 36.50 / 38.00 / 38.50** pt stops the planner authored on 2026-09-04 against floors of **44.59 / 50.73** |
| **2** | swing seats improve turn capture | **[T]-positive** — census's [X] is a **misread** of `grand-audit.md:73-74` (missed-turns 80.0→65.0%, marked PROVEN ✅) | **weight** — seats levels | → unseat / reduce the seat weight | **nothing is refused.** It changes *which levels seat*, never whether an entry passes |
| **3** | level-event wakes deserve a re-read | [T]-**weak** — 52 re-plans / 7 days, **7 ever armed** | **full REPLAN trigger, budget-free** (boot 8: `free: … level_event …`) | → **WARN + count**, N=25, advisory until n improves | **nothing is refused.** It *adds* re-plans; it does not block entries |
| **4** | `BD_MAX_PULLBACK` 0.4 · `BD_MIN_CLOSES` **1** · `BD_MAX_LEVEL_DIST` 5.0 | [I] (pullback, distance) · **[T]** (closes — E3 relaxed 2→1 on displacement quality) | **author-time REJECT** (`breakdown_continue.go:238,258-265`) | → **WARN + count** on the two uncited knobs | **waterfall scenarios with a pullback deeper than 0.4× the leg, or a broken level further than 5.0×ATR5m from price** — refused at write today |
| **5** | stale-confirm 2.0×ATR5m | **[T]** — n=**2,908** MET confirms, cited `plan_confirm.go:118-123` (census's [I] is stale) | **gate** — a MET confirm beyond the band is voided | → measure stale-MET outcomes before touching it | **arms whose confirm fired ≥2.0×ATR5m from `ref_price`** — ~**37%** of MET confirms by the cited measurement |
| **6** | `zoneTFMult` + `zoneEvidenceByKind` + freshness/anchor ladders | [I] | **multiplicative weight** on every level grade | → sensitivity sweep **before** any downstream demotion | **nothing is refused directly** — but grade feeds `min_scenario_quality=C`, so a re-weighted level can pull a scenario back above the quality floor |
| **7** | pre-NY sessions carry edge | **[X]-candidate**, n=6 vs n=3 — **both below any floor** | **enablement** — ASIA + LONDON run | → disable the two sessions | **inverse: demotion REMOVES trades.** Today's only two arm-enabled scenarios were **both LONDON** |
| **8** | consumed / 3rd-touch → `target_only`, never an entry | [I] | **WARN-only** — `RoleMismatches` (`levels_role.go:443`) emits a string; **no gate reads it** | → add a counter or delete the line | **nothing is refused today.** The census's "role demotion" overstates its teeth |
| **9** | killzone / premium-FVG window + Monday-down / Thursday-up conviction | [I] — **no citation anywhere in `docs/`** | **advisory prompt lines only** | → add an n counter, or delete | **nothing is refused** |

**[A] Of the nine, only three (1, 4, 5) refuse anything today.** Two (2, 8) carry less teeth than
the census recorded — one because the label was inverted, one because the "role demotion" is a
warning string. One (7) would *remove* trades rather than admit them. And two of the three that do
refuse (1 and 5) are now **cited**, so the demotion the queue asked for is no longer the obvious
move on either.

---

## 19. LATE FINDING — the arm asymmetry was root-caused and fixed in boot 8

**[A] `kernel/armed.go:19-24` added `reclaim` to the armable set on 2026-09-04**, commit `8f103d1d`
*"feat(arms D3/B): reclaim arms as a stop-entry; waterfalls stay pullback limits"* (07:48:43 CT —
42 minutes before boot 8). The code comment names the cause outright:

> *"reclaim added 2026-09-04 by owner ruling (arms-follow-bias, B): it arms as a STOP-ENTRY beyond
> the reclaim trigger (ArmKindFor). **Until then every long-side play the planner favoured was
> un-armable, which is why long arm-enablement sat at 4.3% while shorts ran at 44%.**"*

That is the mechanism behind the two-day audit's central finding, fixed. It also explains §7's
parity reading (20.0% / 20.0% on 09-04) — a mechanism, not a coincidence, though still n=5 per side.

**But the class-38 prompt contract still pins the pre-ruling set.**
`kernel/prompt_contract.go:63` requires the prompt to contain
`"legal ONLY on fvg_entry|reject|breakdown_continue|breakup_continue"` — **without `reclaim`** —
while `planner_prompt.go:731` derives the live armable line *with* it. The guard that exists to
keep prompt and validator in sync is holding the stale half in place, and still reports
`19 restrictions, all stated in prompt`. **New drift row D-25, fix owner: code.**
