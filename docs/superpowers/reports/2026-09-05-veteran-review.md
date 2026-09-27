# VETERAN REVIEW — LEAD REPORT

**Owner:** hoang · **Date:** 2026-09-05 · **Lead agent**
**Status: COMPLETE.** All four sub-agent parts (A–D) delivered and are folded in below. Each was
verified by me against the committed CSVs and the Go source before use — never taken on its
summary — and every verification is recorded in that part's section, including one case where my
own first check was wrong and the sub-agent was right (§8, the 53-minute error). Where sub-agents
differed, I ruled and said why (§1–3, the win-rate denominator).

---

## EVIDENCE BASIS — read this before any number

This review ran in a **fresh cloud clone of the repository**, not on the owner's machine.
I verified, before dispatching anything:

- **No running engine.** `curl http://localhost:8080/api/health` → connection refused;
  nothing listening on any port. `/api/health`, `/api/expectancy` and
  `/api/config/resolved` were **not reachable** and were not called.
- **No SQLite store anywhere in the container.** Every store query named in the dispatch
  (`touch_outcomes`, `trader_positions`, `plans`, `bars`, `trade_excursions`,
  `planner_rejected_prompts`, …) was **unexecutable**.
- **No tape and no `~/nofx-analysis/`.** **No replay was performed.** Any statement in this
  report about what "would have been armed or filled" is reasoning over code and committed
  data, never a simulation.
- `docs/superpowers/plans/VL-MASTER-PLAN-v2.md` **does not exist** in the tree.
  `docs/superpowers/research/` contains only `INDEX.md`.

What rescued the review is that prior waves **committed their data**. Everything numeric
below was recomputed by me from files in this repo:

| Source | Path |
|---|---|
| the 1E rig + its input | `docs/superpowers/reports/2026-09-03-mc-drawdown-data/{mc_drawdown.py,trade_sample.csv,day_sim.csv,drawdown_paths.csv}` |
| level-touch outcomes | `docs/superpowers/reports/2026-09-04-research-conformance-data/D5b-touch_outcomes-by-{kind,ordinal,session}.csv` |
| the arms funnel | `docs/superpowers/reports/2026-09-04-two-day-audit-data/arms.csv` |
| planner shape | `docs/superpowers/reports/2026-09-04-two-day-audit.md:279-292` (SQL shown in the report) |

**Research labels** follow the house legend (`SYSTEM-MAP.md`, from `2026-09-02-belief-census.md:8-16`):
`[R]` paper/study named · `[T]` own-tape number with n · `[I]` my experience, untested here.

**Two dispatch requirements could not be met and are not claimed:** the report is on
`claude/vl-veteran-review-0905-qnhnt0` (this session is pinned to that branch), **not**
`docs/veteran-review-0905`; and there is no reachable `dev` host, so **no `curl 200`
verification was performed**.

---

## ONE-PAGE SUMMARY

**Verdict on the system as it stands.** The engineering is better than the trading. The
audit discipline here — every knob carrying a citation or a suspension, every belief
labelled `[R]/[X]/[T]/[I]/[O]` — is stronger than most professional desks I have worked on,
and I mean that. But it is scaffolding around a strategy whose central premise is
**unmeasured on its own tape, and absent where it has been measured.** The system fades
levels. Its own `touch_outcomes` say levels hold **48.8%** of the time on first touch. That
is a coin flip, and you cannot build a fade book on a coin flip. Meanwhile the machinery
that would express any edge is leaking badly: **only 20% of arms ever fill**, and a third
of them die because price reached the level *too fast*. This is not ready for live money,
and — importantly — **the system's own Stage-A rule already says so.** I would not change
that rule. I would change what it is measuring.

**The three biggest problems.**

1. **This system ships mechanisms and does not wire them — and the audit record cannot see it.**
   This is mine as lead, and it is the finding I would act on first, because it was found
   *independently by three of the four sub-agents who were not looking for it*. `RiskForceFlat`
   is returned by `DailyGuardrails.Check()` (`kernel/risk_limits.go:245`) and **discarded by its
   only production caller** (`kernel/engine_analysis.go:183`); it has zero non-test consumers, so
   **a resting arm fills straight through a tripped daily limit**. `BiasArmWarning`
   (`kernel/arms_bias_coherent.go`) — shipped 09-04 *specifically* to answer the planner-shape
   finding — has **zero production callers**; only tests. The NT8 AddOn computes `slippageTicks`
   (`ninjascript/VLTraderTCPClient.cs:1383`) and ships it; Go declares the field
   (`provider/ninjatrader/tcp_framing.go:86`) and **never reads it**. The checklist already
   recorded a fourth instance itself — `armGateVerdict` with eight call sites, every one a test
   (`two-day-audit.md:584-588`). I verified all four. `[T]` The pattern is: **the mechanism is
   built, the test is written, the test passes, and nothing calls it in production.** That means
   a passing suite and a green checklist row do not establish that a protection is live. Until
   there is a contract test asserting *production* callers, every safety claim in this project is
   unproven.
2. **The fade premise has no measured edge, and the reward side is fiction `[T]`.** First-touch
   hold rate **118/242 = 48.76%, Wilson [42.5%, 55.0%]** — a coin flip; 17 of 18 level kinds
   contain 50%. The one kind that separates does so *on the wrong side for a fade*: RTH-L holds
   **20/63 = 31.75%, Wilson [21.6%, 44.0%]**, i.e. the prior-day RTH low **breaks 68%** of the
   time — and it carries the highest average score in the candidate pool and is arm-traded 100%
   as a fade. On the other side of the trade, plans claim a median **2.55:1** while the book
   realises **1.66:1**, because the R:R floor is checked against a target the model itself chose;
   only **3 of 36** trades ever produced enough favourable excursion to reach a 2.0R target
   measured off the *minimum* stop.
3. **The planner writes longs in a vocabulary the executor cannot arm `[T]`.** The famous 09-03
   number — 0 `open_long` in 575 decisions during a +483-point rally, long scenarios arm-enabled
   4.3% against shorts at 44.4% — has a mechanism underneath it that nobody had named: **21 of
   the 23 long scenarios authored that day rode on conditions the machine could not arm** (9
   `breakout_retest`, never armable and shadowed; 8 `reclaim`, not armable until the next day; 4
   `sweep_reclaim`, split-only). From **09:20:47 to 11:58:33 the book was empty on both sides
   while price ran +199.25 points.** This is not a bias to be prompted away; it is a vocabulary
   mismatch, and the coherence check meant to catch it is the unwired `BiasArmWarning` above.

**The three biggest opportunities.**

1. **Stop fading and start trading the break, at the one place the tape gives you an edge
   `[T]`.** RTH-L breaking 68% of the time (n=63) is the only statistically separated
   result in the entire level census. It is currently traded backwards.
2. **Fix the funnel and you multiply whatever edge exists by ~5×, for free `[T]`.** A 20%
   fill rate means four of every five correct opinions never reach the market. The
   marketable-guard rejections in particular are not risk management — they are a systematic
   filter that keeps you out of exactly the fast moves that carry follow-through.
3. **Turn the $450 daily loss limit ON — it is a free option `[T]`.** On the 11 committed
   session days it trips on **1 day (9.09%)**, the −$492.00 day of 08-20, and it forfeits
   **$0.00** — because that day's limit-crossing trade was the day's *last* trade
   (`day_sim.csv`; `mc_drawdown.py:127-128` prints exactly this condition). Measured cost:
   nothing. It is currently OFF.

---

## 1–3. PART A — the way it trades, the levels, the decisions

**Delivered.** Full text: `docs/superpowers/reports/2026-09-05-veteran-part-a.md` (652 lines).

**My verification before use.** I spot-checked two numbers against the committed CSVs myself
rather than trusting the summary. (1) RTH-L hold **20/63 = 31.75%, Wilson [0.216, 0.440]** —
matches my own computation exactly. (2) The first-touch coin-flip and the per-kind census
reproduce from `D5b-touch_outcomes-by-kind.csv`. Both pass.

**One disagreement I am ruling on.** A reports win rate **33.87%** and payoff **1.66**
(breakeven 37.59%); I computed **32.8%** and **1.74** (breakeven 36.5%). The difference is the
denominator: A excludes 2 scratch trades, I counted all 64. **A's treatment is the better one**
— a scratch is not a loss and should not sit in the win-rate denominator — so **33.87% / 1.66 /
37.59% is the number to quote**, and mine is the all-64 variant. Both agree on the finding that
matters: the gap to breakeven is under 4 percentage points and n cannot resolve it.

**What A adds that I did not have.** A located committed tape exports I had missed —
`docs/superpowers/reports/exports/2026-09-02-level-replay/` (1,168 episodes) and
`exports/2026-09-02-losses/` (23 plans, 544 executor cycles, bars) — and replayed the two
committed sessions end-to-end on them. Its three hardest findings:

1. **The reward side is fiction `[T]`.** Plans claim a median **2.55:1**; realised is **1.66:1**.
   The R:R floor is checked against a target the model itself chose, so the model simply writes a
   bigger number. Only **3 of 36** trades produced enough favourable excursion to reach a 2.0R
   target measured off the *minimum* stop — Wilson [2.9%, 21.8%], and that is an upper bound.
2. **The stop floor is calibrated to the losers `[T]`.** Winners' worst adverse excursion never
   exceeds **0.661×** the 1.5×ATR5m floor (n=10); losers' median is **1.014×** it (n=26).
3. **The AI decision path is inert, and ASIA is the whole loss `[T]`.** 544 executor cycles on
   09-01/09-02 → **492 `wait`, 5 `open_long`, 0 `open_short`**; four became positions, all four
   lost. ASIA is n=16 / **−$552.43** against ex-ASIA n=48 / **+$128.50** — and the code already
   ships ASIA `Enabled: false` (`kernel/session_registry.go:93`); the running config overrode it.

**Two corrections A makes to the dispatch's own premises, which I accept.** The executor loop is
not 2-minute: `scan_interval_minutes` defaults to **3, minimum 3** (`store/trader.go:29`), and
SYSTEM-MAP §12 labels it `[X]` — never tape-tested. And the committed
`E-d3-summary-percentiles.csv` states `atr_convertible n=30` while the per-trade CSV yields
**n=36**; A could not rebuild the report's cohort and used its own, flagged. That is the right
call and it is a live reproducibility defect in a committed artifact.

## 4–6. PART B — monitoring, execution, risk

**Delivered.** Full text: `docs/superpowers/reports/2026-09-05-veteran-part-b.md` (498 lines).

**My verification before use.** (1) B re-ran the committed 1E rig and reproduced the published
report bit-for-bit — mean **−6.624**, sd **100.589**, CI **[−31.268, +18.020]** — which matches
my own independent computation to four decimals. (2) B's arms tally, **5 of 15 marketable-guard
vs 3 filled**, matches mine from `arms.csv`. (3) I verified B's new n=65 input: position **591**,
`pnl_corrected = −140.0`, exists at `2026-09-04-two-day-audit-data/trades.csv:12`. All pass.

**Updated 1E numbers (n=65, B's run, input verified by me):** mean **−$8.676**, sd **$101.162**,
se **$12.548**, CI **[−$33.269, +$15.917]**, t **−0.691**. maxDD@50 p50 **$934** / p95 **$1,767**;
@100 p95 **$2,771**, p99 **$3,304**. Adding one trade moved the mean *down* and the interval still
contains zero. Post-0B is **n=3, all losses**.

**One reconciliation.** B's script reports `n_req` **1,067–1,810**; I computed **886**. These are
different questions — mine is the sample needed to exclude zero at 95% given the observed mean,
the script's is a powered detection. Both say the same thing: **the required sample is more than
an order of magnitude above what exists.**

Its three hardest findings:

1. **A daily-loss trip can neither flatten you nor stop an arm `[T]`.** `DailyGuardrails.Check()`
   returns `RiskForceFlat` (`kernel/risk_limits.go:245`) and the only production caller **discards
   it** — `} else if _, gErr := g.Check(); gErr != nil {` (`kernel/engine_analysis.go:183`) —
   then merely skips the decision cycle. `RiskForceFlat` and `RiskLimits.Classify` have **zero
   non-test consumers**, and neither `entry_gate.go` nor `armed_executor.go` contains the string
   `daily` or `guardrail`. **A resting arm fills straight through a tripped limit.** This
   materially changes my §9 item 5 — see the amendment there.
2. **The marketable guard is the largest killer of arms and is invisible to every counter `[T]`.**
   5 of 15 arms died to it and never reached the wire; one was cancelled for closing **1.70
   points** on the wrong side. It runs once per scan cycle on a bar close
   (`armed_executor.go:898`), the cancel is terminal, and it emits no `IncArmRefusal` — only a
   `logWarnf` (`:960-961`). It is the biggest leak in the system and nothing counts it.
3. **The broker computes your slippage and Go bins it `[T]`.** The AddOn calculates
   `slippageTicks` (`ninjascript/VLTraderTCPClient.cs:1383`) and ships it (`:1430`); Go declares
   the field (`provider/ninjatrader/tcp_framing.go:86`) and **never reads it** — no column, no
   telemetry. Meanwhile 3 of 3 armed fills printed the authorized price to the tick on limits the
   market had traded *through*, so the fill statistics carry no live-execution information at all.

## 7. PART C — the prompts, the laws, the settings

**Delivered.** Full text: `docs/superpowers/reports/2026-09-05-veteran-part-c.md` (943 lines).

C did the measurement for real rather than estimating: it rendered the production prompts through
the actual builders under a scratch overlay (`BuildPlannerPrompt`, `plannerOutputContract`,
`BuildFuturesDecisionSystemPrompt`) and counted with a real BPE, stating plainly that DeepSeek's
own tokenizer was unavailable so the counts are a consistent proxy. That caveat is correctly made
and correctly bounded.

**My verification before use — I checked its three load-bearing claims.**

1. **The prompt contradicts itself on a money number.** `planner_prompt.go:733` renders the floor
   from the resolver — `fmt.Sprintf("%.1f", MinSLATRMult())` — while `:752` carries the hardcoded
   literal *"min-SL ≥ 1.0×ATR5m"*. `kernel/min_sl.go:34` sets `MinSLATRMultDefault = 1.5`. So the
   prompt tells the model the floor is 1.5 in one sentence and 1.0 about 2,000 characters later,
   and the 1.0 is simply stale. **Confirmed.** A model that sizes to the 1.0 sentence authors
   stops that are silently widened at arm time and then judged by the R:R gate at the wider
   number — which is precisely the failure the checklist already recorded at `:853-857`.
2. **The executor does not know it is in `plan_mode=strict`.** `grep -iw strict` on the
   boot-verified golden `kernel/testdata/futures_mnq_plan.golden` returns **0**; the only
   substring hit is the unrelated word "Strictly". **Confirmed.** Meanwhile
   `trader/entry_gate.go:160-163` refuses every decision-path market entry outright, and the
   executor is still handed an `open_long` worked example that cannot execute.
3. **`fvg_entry` is shadow by default** — `kernel/condition_status.go` sets
   `"fvg_entry": ConditionShadow` in the owner-ruled baseline. **Confirmed**, so C's point that
   ~20% of the fixed instruction budget buys a condition that cannot place an order stands.
   I also confirmed the checklist carries exactly **80** `**Law:**` lines, as C states.

**The measurement, as delivered.** Planner prompt fully populated: **19,933 chars / 5,394 tokens**.
The output contract alone is **12,637 chars / 3,301 tokens = 73.0%** of the minimal prompt. The
`Rules:` paragraph is **one unbroken line of 10,090 chars / 2,511 tokens / 52 sentences with 157
ALL-CAPS emphasis tokens — 46.6% of the whole prompt**. Instruction is **~64–80%** of the payload
depending on how much live data is present. Modal counts, all-caps: **MUST 14 · NEVER 8 ·
SHOULD 5 · ONLY 14**; any casing: **must 20 · never 27 · should 6 · only 22**. Executor prompt:
**5,477 chars / 1,506 tokens**, byte-identical to the boot-verified golden.

**My reading of that.** `[I]` Twenty-seven `never`s and a 52-sentence unbroken paragraph is not a
checklist any more, it is a wall. I have watched desks do this to their own traders: every loss
adds a line, nothing is ever removed, and within a quarter nobody reads past the third bullet.
The prompt has been maintained the way a rulebook is maintained after incidents, not the way an
instruction is maintained for the person who has to act on it. Half of it being a single paragraph
is the tell.

**The one thing C found that I would put money on immediately.** `B3 breakdown_void_reclaimed` was
**21 of 55 rejects (38.2%)**, 15 of them on a single day. A prompt sentence created that reject
class; the class-45 VOID block killed the sentence; rejects went to **zero on 09-03 and 09-04**.
`[T]` That is the only clean before/after in the entire reject record, and it is the template —
find the sentence, kill the sentence, measure the class. C is right that it has been applied to
one rule out of nineteen.

**C's own research-law audit of the prompt, which is the assignment done properly.** Of 12 market
claims in the prompt, **6 are uncited `[I]`** — including *"Conviction: down on Monday, up
Thursday/Friday"* (`:656`), which the system's own belief census calls "pure prompt doctrine" —
**3 are `[R]`-by-reference to studies not present in this tree**, and **2 are `[T]` literals the
tape has since refuted** ("reject 75% win" against a measured 45.2%, n=31). And bias-tree branch
5's absolute *"longs ONLY below 50%"* was violated by **17 of 58 plans (29.3%)** with nothing
rejecting any of them.

## 8. PART D — the three-day stretch and the tape

**Delivered.** Full text: `docs/superpowers/reports/2026-09-05-veteran-part-d.md` (839 lines).

D made the best use of this environment of the four: it found that
`2026-09-04-two-day-audit-data/` contains **real tape** — `d6_bars_1m_0903.csv`, **1,355 one-minute
MNQ bars for 2026-09-03** — plus `refusals.csv` (61 rows), `cadence.csv` (588), `plans.csv` (159
scenario rows). So a genuine partial re-derivation of the two-day audit was possible after all. It
is still **not a replay**: no 09-02 or 09-04 bars are committed, and D marks the replay BLOCKED and
writes out the SQL for the owner to run on his own machine.

**My verification before use — I checked all three headline claims myself.**

1. **Cadence.** `cadence.csv` holds **588 rows: 513 `skipped`**, of which **493 `min_interval` and
   20 `cooldown_enforced`**. Reproduces exactly. The audit's "class-47 suppressed **nothing**"
   (`two-day-audit.md:38`) is contradicted by the audit's own committed CSV. **D is right.**
2. **The 53-minute error.** Verified, and *I got this wrong on my first pass and D did not*. I
   initially tested `high >= 29476.00` and found 10:16 CT, which appeared to refute D. That is the
   wrong side of the bar: the arm was a **long limit resting below market**, so it is filled by the
   **low**. Testing correctly: exactly **one** bar after 11:27 has `low <= 29476.00`, at **12:21
   CT**; 11:28's low was **29494.00**; and 29476.00 prints exactly once all day. The audit's
   "price did reach 29476.00 at 11:28 CT … by 8.2 points and one minute"
   (`two-day-audit.md:335-336`) is wrong by **53 minutes**. **D is right.**
3. **`BiasArmWarning` has zero production callers.** `grep -rln "BiasArmWarning" --include=*.go`
   returns exactly one non-test file — `kernel/arms_bias_coherent.go`, its own definition. Every
   other reference is a test. **D is right.**

**The ruling that matters.** The audit's headline split — gates ~5% / planner shape ~55% / outage
~30% — is *directionally* right and *structurally* unsound, and I accept D's reasoning. The
percentages **have no denominator and no unit**: the buckets are counted in four incommensurable
currencies (scenarios, refusal events, one "window", arms) and then assigned shares on an
undisclosed basis. D shows the report's own §11 counts, if actually used, invert it to gates 41.1%
/ outage 0.7%. `[T]` **I would not quote the 5/55/30 split again.** The underlying findings
survive; the arithmetic that ranks them does not. That is a real defect in the project's most
load-bearing recent document, and it is exactly the kind of thing a second pair of eyes is for.

**What D adds that changes the picture.**

- **The mechanism behind the 4.3%.** **21 of the 23 long scenarios written on 09-03 rode on
  conditions the machine could not arm** — 9 `breakout_retest` (never armable *and* shadowed), 8
  `reclaim` (not armable until the next day), 4 `sweep_reclaim` (split-only). The planner was not
  merely biased; it was writing longs in a vocabulary the executor had no way to express. `[T]`
- **From 09:20:47 to 11:58:33 the book was completely empty on both sides while price ran +199.25
  points.** `[T]`
- **The fix for the #1 cause was never wired.** `BiasArmWarning` (`kernel/arms_bias_coherent.go:74`)
  shipped 09-04 against the planner-shape finding and has zero production callers; its sibling
  `ArmableConditionsLine` did ship. Half a fix — while the prompt-contract boot line still reports
  "19 restrictions, all stated in prompt". `[T]`
- **A caveat D is right to flag:** the audit's `arm.enabled` numerators (1 and 8) appear in **no**
  committed CSV — `plans.csv` has no such column — so 4.3%/44.4% remains the audit's claim, not an
  independently verified number. I had checked only its internal arithmetic; D is correct that this
  is weaker evidence than it looked. It stays in the report labelled as the audit's.

---

## 9. MY TOP TEN

Ordered by what I would do first with my own money on the line. Each carries **what · why
(evidence) · what it takes · the number I would watch**. Cross-checked against
`AUDIT-CHECKLIST.md` so nothing already fixed is recommended again; the already-fixed items I
found are listed at the end and deliberately **not** re-recommended. Two are removals (items 9 and
10); a third, item 6, is a "do not build".

**1. Audit every safety mechanism for a production caller, and add a contract test that fails
without one.**
*Why:* four independent instances, all verified by me `[T]` — `RiskForceFlat`
(`kernel/risk_limits.go:245`, discarded at `kernel/engine_analysis.go:183`), `BiasArmWarning`
(`kernel/arms_bias_coherent.go`, only non-test reference is its own definition), `armGateVerdict`
(`trader/armed_executor.go:1338`; all 8 call sites are in `armed_executor_test.go`), and
`SlippageTicks` (`provider/ninjatrader/tcp_framing.go:86`, declared, set in a mock and a test,
never read). `[I]` A green suite that proves a function works, while nothing calls it, is the most
expensive kind of false comfort — it is how a desk discovers in a drawdown that its risk limit was
a unit test.
*What it takes:* code + a contract test in the style the project already uses for boot lines.
*Watch:* count of money-deciding mechanisms with zero production callers. Today: **at least 4**.

**2. Wire the daily loss limit to something that can stop trading — then turn it on.**
*Why:* the limit trips **1 of 11 days (9.09%)** and forfeits **$0.00/day** `[T]`
(`day_sim.csv`; `mc_drawdown.py:127-128` explains the zero — the crossing trade was the day's
last). Free insurance. But per item 1 it cannot flatten or stop a resting arm today, so switching
the knob on buys a feeling, not a stop.
*What it takes:* code first (consume `RiskForceFlat` on both order paths), then the knob.
*Watch:* days tripped; and a fixture proving a tripped limit cancels a resting arm.

**3. Fix the two prompt sentences that state the wrong money number.**
*Why:* `planner_prompt.go:733` renders the stop floor from the resolver (1.5) and `:752` carries a
stale literal saying **1.0×ATR5m**, against `MinSLATRMultDefault = 1.5` (`kernel/min_sl.go:34`)
`[T]`. A model sizing to the 1.0 sentence authors stops that get silently widened at arm time and
then judged at the wider number by the R:R gate — the exact punishment loop the checklist already
recorded at `:853-857`. The same paragraph lists `fvg_entry` as arm-legal and then forbids arming
it.
*What it takes:* prompt, one line each. This is the cheapest item on the list.
*Watch:* rejects per rule; and whether authored stop ÷ ATR5m clusters at 1.0 or 1.5.

**4. Fix the armable-condition vocabulary before touching the planner's bias.**
*Why:* **21 of the 23 long scenarios on 09-03 rode on conditions the executor cannot arm** — 9
`breakout_retest` (never armable *and* shadowed), 8 `reclaim`, 4 `sweep_reclaim` `[T]`. The 4.3%
long arm-enablement is a *symptom*; the cause is that the planner writes longs in a vocabulary the
machine has no way to express. Telling it to write more longs changes nothing.
*What it takes:* owner ruling on which conditions are armable, then code, then prompt. And wire
`BiasArmWarning`, which shipped for this exact finding and has no callers.
*Watch:* share of authored long scenarios whose condition is in the armable set; then long-side
arm-enablement on `long`-bias days (today **4.3%**, the audit's figure — not reproducible from any
committed CSV, so treat it as their claim).

**5. Convert the marketable guard from a cancel into a bounded entry.**
*Why:* **5 of 15 arms (33%) died "level accepted through — marketable, never placed"** against
only 3 filled; **9 of 15** had price trade through; one died over **1.70 points** `[T]`. The
cancel is terminal, re-armable only on a plan-version change, runs once per scan on a bar close
(`armed_executor.go:898`), and emits no counter at all — only a `logWarnf` (`:960-961`). `[I]` A
limit that refuses to become marketable is not saving you slippage; it is selecting for slow
mean-reverting approaches and discarding fast ones. That is adverse selection you are performing
on yourself.
*What it takes:* code + an owner ruling on a slippage cap. And a counter, so it stops being
invisible.
*Watch:* arms filled ÷ arms whose level was reached. Today **3 of 9 = 33%**.

**6. Do NOT add a daily trade cap — and write the ruling down.**
*Why:* `max_daily_trades_3` trips **81.84% of days** and costs **−$24.54/day** net `[T]`. It is
the intuitive control that this tape says is wrong, and it is exactly what gets added after a bad
week.
*What it takes:* an owner ruling, recorded.
*Watch:* nothing. The point is to not build it.

**7. Read the slippage the broker is already sending you.**
*Why:* the AddOn computes `slippageTicks` and ships it; Go never reads it `[T]`. Meanwhile 3 of 3
armed fills printed the authorized price to the tick on limits the market had traded *through*, so
the fill record currently carries **no live-execution information at all**. You cannot manage
execution you do not measure, and the measurement is already on the wire.
*What it takes:* code — a column and a counter.
*Watch:* mean and 90th-percentile slippage per fill, by condition.

**8. Report the effective sample as days, and publish ambiguity next to every rate.**
*Why:* n=64 trades are **11 session days**, dominated by two of them (+$469.00, −$492.00) `[T]`;
trades inside a day share one plan, one bias, one regime. The MC rig bootstraps by trade with a
mean block of 5 (`mc_drawdown.py:35-38`), which approximates a day by accident of scale, not by
design. Separately, ambiguity is the quiet defect in the level census: SWG-L and eVWAP **50%**
unclassifiable, SWG-H **48.3%**, OB **44.4%**, VWAP **38.6%** `[T]` — and it flatters exactly the
kind that looks most tradeable.
*What it takes:* data/reporting; key the block bootstrap to `session_day_ct`.
*Watch:* distinct session days (today **11**; I would not take a strategy conclusion seriously
below ~60), and ambiguous ÷ rows_all per kind.

**9. REMOVAL — retire `eVWAP`, `VWAP±2σ`, `EQL`, `ONH`, `ONL`, `PDL` as decision inputs, and cut
the `fvg_entry` instruction block.**
*Why:* n_den of **5, 4, 2, 3, 3, 3** `[T]`; `ONH`, `PDL` and `EQL` show 100% hold on n=3, n=3,
n=2 — rounding, not edges. And `fvg_entry` is `shadow` by default
(`kernel/condition_status.go`) while consuming ~20% of the fixed instruction budget — **32
mentions and 850 tokens of Rules-paragraph** buying a condition that cannot place an order `[T]`.
*What it takes:* code + prompt + owner ruling.
*Watch:* seated-kind count; prompt token count; and no decision citing a kind with n < 30.

**10. REMOVAL — cut the `Rules:` paragraph in half, and delete the 105-minute wake blackout or
promote it to a stated rule.**
*Why:* the `Rules:` paragraph is **one unbroken line of 10,090 chars / 2,511 tokens / 52 sentences
with 157 ALL-CAPS emphasis tokens — 46.6% of the whole prompt**, inside a prompt that is
**~64–80% instruction by token** with **27 `never`s** `[T]`. `[I]` That is past the point where
anyone, model or human, reads to the end. The template for cutting it already exists and is
measured: `B3 breakdown_void_reclaimed` was **21 of 55 rejects (38.2%)**; a prompt sentence caused
it, the class-45 VOID block killed the sentence, and rejects went to **zero** on 09-03 and 09-04.
Separately, NY `WindowEndCT`=14:45 with ASIA `ReadCT`=16:30 makes `inSessionReadWindow` false for
**14:45–16:30 every weekday** (`two-day-audit.md:880`) — either a deliberate no-trade window,
which belongs on the plan card, or two constants nobody diffed.
*What it takes:* prompt; and code or an owner ruling for the blackout.
*Watch:* prompt tokens (today **5,394** full-render) and rejects per rule; decisions per hour
across 14:45–16:30 (today: none, by construction).

### Cross-check — already fixed, NOT re-recommended
- **BE and the ATR trail suspended** behind `EXIT_MECHS_SUSPENDED` (`AUDIT-CHECKLIST.md:632-644`).
  Right call, clean evidence: 2 BE moves and 8 trail ratchets on 09-01, **$719.50 of giveback and
  zero trail EXITS ever**. Leave it suspended.
- **Stop floor 1.0→1.5×ATR5m, composed stop, level never invented** (`:640-644`). Note item 3: the
  prompt still tells the model 1.0 in one sentence.
- **"One gate, two ATRs"** — the decision path read the DAILY ATR (300.4) and refused against a
  phantom `1.5×ATR5m = 450.56` while the arm seam used ATR5m 12.78–14.12; all three refused
  targets printed within 28 minutes. Fixed (`:777-789`).
- **Stage-A size ceiling: 1 contract until n≥30 with a positive lower-CI expectancy** (`:645-647`).
  **The best rule in the checklist; do not touch it.** Where it stands: n=64 clears n≥30, the
  lower CI is **−$31.27**. *The system's own rule already says do not size up.*

## 10. IDEAS

**The honest go/no-go for live money, in numbers `[T]`.** n=64, mean **−$6.6239**, sd
**$100.5891**, SE **$12.5736**, **t = −0.527**, 95% CI **[−$31.27, +$18.02]**. Zero is
inside the interval; so is −$30 and +$18. Win rate **32.8%**, average win **$114.67**,
average loss **−$65.86**, payoff **1.74** — which needs **36.5%** to break even, so the
whole deficit is **3.7 percentage points, about 2.4 trades out of 64**. At this variance you
need roughly **886 trades** to resolve a mean of this size at 95%. Against that, the median
50-trade drawdown is **$866** (iid; $847 block), p90 **$1,477**, p99 **$2,065**, worst
**$3,029.50** (B=10,000). So the expected 50-trade P&L is about **−$331** while the *median*
drawdown you must sit through is **$866**. **The signal is an order of magnitude smaller
than the noise.** No-go, and it is not close — but note that "no-go" here means *not yet
measurable*, not *proven unprofitable*. Those are different findings and the system deserves
the more precise one.

**`[I]` What I would do with the 2-minute executor loop.** I would stop asking the LLM to
decide *whether* to trade and ask it only to decide *where the level is and whether the
level is still valid*. Entry, sizing and exit are arithmetic; they belong in code where they
can be tested. In my own systems the LLM earns its keep as a **state classifier** — is this
still a trend day, has this level been reclaimed, is the news window live — and loses money
the moment it is allowed to author the trade. This system currently lets it author the trade
and then spends a large gate chain undoing that. Untested here.

**`[I]` The R:R 2.0 target is fighting the 32.8% win rate.** A fade book at a coin-flip
level with a 2R target is asking for a 36.5%+ hit rate at a location where the tape gives
you 50/50 minus costs. Either the entry has to get better or the target has to come in. I
would test 1R-to-scratch-runner before I would test anything else, because it converts the
payoff problem into a win-rate problem, and win rate is the thing this system could actually
measure at n=64.

**`[I]` What I have never seen work, and this system does.** Authoring resting orders at
prices derived from a model's *narrative* rather than from live book pressure — the 71%
never-reached figure is what that looks like from the outside, and it is not a prompt bug,
it is what happens when the thing choosing the price cannot see the depth. Also: a
one-open-position constraint combined with a fade book. The two fight each other — a fade
book's whole risk profile assumes you can be wrong at one level and right at the next, and a
single slot means your first bad level costs you the session.

**`[I]` One thing I would steal from a discretionary desk.** A written "what would have to be
true" line on every plan, and an end-of-day check of whether it was. Not for the model — for
the owner. The reason this project's audit discipline is so strong and its trading is not is
that the audits measure the machine and nobody is measuring the thesis.

---

## SURPRISES — recorded, not acted on

- **The 1E rig is committed to the repository** (`mc_drawdown.py` + `trade_sample.csv`), so
  the drawdown numbers are fully reproducible by anyone who clones. That is unusually good
  practice and it is the only reason this review has numbers at all.
- **`day_sim.csv` shows the $450 limit forfeiting exactly $0.00**, which looks like a bug
  until you read `mc_drawdown.py:127-128` and find the script anticipated it: *"forfeited 0:
  every trip landed on the day's LAST trade — nothing followed it."* The author of that
  script was being careful. Worth saying out loud.
- **`docs/superpowers/plans/VL-MASTER-PLAN-v2.md` is referenced by the dispatch but is not
  in the tree.** Either it was never committed or it moved. Not investigated.
