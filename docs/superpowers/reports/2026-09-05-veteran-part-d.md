# Section 8 — The three-day stretch and the tape (Sub-agent D)

## EVIDENCE BASIS — READ THIS FIRST

**No store, no bars, no tape, no engine were reachable from this environment, and NO REPLAY WAS
PERFORMED.** This is a fresh cloud clone at `/home/user/nofx`. There is no `data.db` anywhere in
the container, no `bars` table, no listener on `localhost:8080`, no `/api/health`, no
`/api/expectancy`, no `/api/config/resolved`, and no `~/nofx-analysis/`. Nothing below is a
simulation, and no number below was produced by replaying a tape I did not have.

What I *did* have, and what everything below rests on, is **committed evidence inside this
repository**:

1. **`docs/superpowers/reports/2026-09-04-two-day-audit-data/`** — 90 files, and it contains real
   tape: **`d6_bars_1m_0903.csv`, 1,355 one-minute MNQ bars covering 2026-09-03 00:00:00 →
   23:34:00 CT**, plus `refusals.csv` (61 rows), `cadence.csv` (589 rows), `plans.csv` (159
   scenario rows), `arms.csv` (15 rows), `trades.csv`, `baseline.csv`, `d6_timeline.csv`,
   `tape.csv`, `tape_moves_0903.csv`, `decision_intent_by_day.csv`.
2. The other five data directories named in my dispatch, of which
   `2026-09-04-research-conformance-data/` proved materially useful.
3. The Go source at the dev tip.

So **I was able to re-derive a substantial part of the two-day audit's verdict after all** — not
from the live store, but by recomputing on the audit's own committed CSVs. Every recomputation
below shows its command and its output. Where a number is the audit's and I could not reproduce
it, I say so in those words.

**Three standing caveats.** (a) The committed CSVs are the audit's *extracts*, not the store; a
defect in an extract is invisible to me. (b) `d6_bars_1m_0903.csv` covers 09-03 only — there are
**no committed 09-02 or 09-04 bars**, so anything requiring the 09-02 tape is out of reach.
(c) The Go source here is the **dev tip as of this checkout**, which is *after* the 2026-09-04
`arms-follow-bias` wave; where 09-03 behaviour differed from today's code I say which, using the
code's own dated comments.

---

## 8.0 Summary

The two-day audit's headline attribution — gates ~5% / planner shape ~55% / outage ~30% — is
**directionally right and structurally unsound**. Right, because the refused set really does lose
money on the tape (I recomputed it to within $1.50 of the audit's figure) and the planner really
did leave its own bias un-armable (I recomputed that too, and found a sharper number than the
audit reports). Unsound, because **the percentages have no denominator and no unit**: the buckets
are counted in four incommensurable currencies — scenarios, refusal events, one "window", arms —
and then assigned shares on an undisclosed basis that inverts the only count the report actually
publishes. §8.4.

Three things I found that the audit did not, or got wrong:

1. **The audit's cadence verdict is contradicted by the audit's own CSV.** It says class-47
   "suppressed **zero** wakes" across both days. `cadence.csv` carries **513 rows with
   `event=skipped`** across 09-02/09-03, of which **20 carry `reason=cooldown_enforced`**, 14 of
   them between 10:30:29 and 11:18:33 CT on 09-03 — the strongest hour of the trend day. §8.3.4.
2. **A 53-minute error at the centre of the "no setups" bucket.** The audit says the day's only
   arm-enabled long "did reach 29476.00 at **11:28 CT** — one minute *after* v6 was superseded".
   In the audit's own bars, price first reached 29476.00 at **12:21 CT**. Not one minute — 54.
   And 12:21 is inside the lunch no-trade window. §8.3.2.
3. **The fix for the audit's own #1 cause is not wired.** `BiasArmWarning` — the arms-follow-bias
   coherence check shipped 2026-09-04 in answer to the 55% finding — has **zero production
   callers**. §8.6.6.

On the trader's question: at 10:00 CT on 09-03 I would have bought the break, and the system could
not have, for a reason that is neither a gate nor the outage. **Of the 23 long scenarios the
planner wrote on 09-03, 21 rode on conditions the machine could not arm** — 9 `breakout_retest`
(never armable *and* shadowed), 8 `reclaim` (not added to the armable set until the next day), 4
`sweep_reclaim` (split-contract only). Exactly **2** used a plainly armable condition, and both
were `reject` — a touch-only fade. Every placeable long entry that day was a limit order resting
*below* the market waiting for a pullback, on a day that ran **+483.25 points**. §8.5, §8.6.

And the number that says it best, which I recomputed and which appears in no report I read: **from
09:20:47 to 11:58:33 CT — 2h 37m — the system had no resting order at all, on either side, while
the market ran +199.25 points from 29348.75 to 29548.00.** §8.5.3.

---

## 8.1 What I re-derived, and what I could not

| the audit's claim | status | where |
|---|---|---|
| +483-pt rally on 09-03 | **REPRODUCED exactly** (+483.25) | §8.2.1 |
| 0 `open_long` in 575 decisions | **REPRODUCED exactly** | §8.2.2 |
| arm-enabled long 4.3% (1/23) vs short 44.4% (8/18) | **denominators reproduced; numerators not in any committed CSV** | §8.2.3 |
| refused set loses $860.64 | **REPRODUCED to $1.50 — but at n=55, not the n=44 stated** | §8.2.4 |
| the one long missed by 8.2 pts | **REPRODUCED exactly** (8.20) | §8.2.5 |
| arm 37: 83 one-minute bars at/above 29543.75 | **REPRODUCED exactly** | §8.2.6 |
| silence 12:24:33 → 14:18:24 | **REPRODUCED to 60 seconds** (ledger says 12:23:33, 114.8 min) | §8.2.7 |
| price reached 29476.00 at 11:28 CT | **REFUTED — 12:21 CT** | §8.3.2 |
| cadence suppressed zero wakes | **REFUTED — 513 skips, 20 `cooldown_enforced`** | §8.3.4 |
| the 5% / 55% / 30% / 10% / 0% split | **NOT DERIVABLE — no denominator, no unit** | §8.4 |

Out of reach entirely, and marked so: anything needing the 09-02 tape, `decision_records`
row-level content, `touch_outcomes`, `candidate_pool`, `trade_excursions`, the NT8 logs, the live
knob resolution, or a replay.

---

## 8.2 Auditing the audit — what holds

### 8.2.1 The rally is real, and I get the same number

```
$ python3  # on docs/superpowers/reports/2026-09-04-two-day-audit-data/d6_bars_1m_0903.csv
bars n= 1355 first 2026-09-03 00:00:00 last 2026-09-03 23:34:00
CME day thru 16:00 CT   n= 960 low 29075.00 @00:23  high 29585.00 @13:15  range 510.00
low 05:30-07:00: 29101.75 2026-09-03 06:59:00
29585 - 29101.75 = 483.25
```

**[T]** The audit's anchors (`two-day-audit.md:204`: "29101.75 (06:00 low) → 29585.00 (13:00 RTH
high) = +483.25") are both present in the bar file to the tick. **+483.25 points in 6h16m, n=1,355
one-minute bars, 2026-09-03.** The 29601.00 print in the file is the next CME session (17:00+ CT)
and the audit says so; the audit's own `tape.csv` separates it correctly.

The day's high of 29585.00 first prints on the **13:15** bar. Hold that timestamp — §8.6.4 needs
it.

### 8.2.2 Zero long intents in 575 decisions — exact

`decision_intent_by_day.csv`, read whole:

```
2026-09-03,561,0,14,575,2.43     (day, wait, open_long, open_short, decisions, open_intent_pct)
```

**[T]** 0 `open_long` in 575 decisions, 2026-09-03. Reproduced. The audit's correction to its own
dispatch (`:355-357`) — that 2.43% was the *highest* open-intent rate of the eight days measured,
and four baseline days produced zero — is also confirmed straight off the same file. That
correction is good work and I would not weaken it.

### 8.2.3 The 4.3% / 44.4% — denominators reproduce, numerators are not in evidence

```
=== trade_date 2026-09-03 scenario rows 41
  long  total= 23 ;  short total= 18
```

**[T]** From `plans.csv`, 09-03 carries exactly **23 long and 18 short** scenario rows — the
audit's denominators, to the unit.

The numerators do not reproduce, because `plans.csv` **has no `arm.enabled` column**. Its columns
are `plan_id, version, trade_date, session, trigger_reason, trigger_class, created_ct, degraded,
scenario_id, side, entry, stop, target, condition_text, condition_ever_true, first_true_ct,
followed_by_arm, bias_ai, bias_tree, bias_regime, session_direction`. The audit's numbers come
from `json_extract(s.value,'$.arm.enabled')` on the live `plans` table (`two-day-audit.md:288-293`),
which is not in this repo. **So 1 and 8 are the audit's, not mine, and I cannot check them.**

What I *can* compute from `plans.csv` is the stronger fact — arms that actually reached the
ledger:

```
09-03 long : followed_by_arm contains 'arm#' = 0 / 23
09-03 short: followed_by_arm contains 'arm#' = 3 / 18   (arms 35, 36, 37)
```

**[T] Zero long arms were created on 09-03. All three arms that existed were shorts.** That is a
cleaner statement than 4.3% and it needs no denominator argument.

**A conflict the owner should know about.** `2026-09-04-research-conformance-data/d7-arm-enablement-by-date.csv`
re-measures the same quantity from the live table and gets **long 2/28 (7.1%), short 8/20 (40.0%)**
for 09-03 — different denominators from `plans.csv`'s 23/18. The conformance report explains it
(`2026-09-04-research-conformance.md:386-389`): the `plans` table kept growing after the audit's
snapshot, 09-03 ending with 16 versions. That explanation is credible and I accept it. But it
means **the headline ratio behind the "~55%" bucket moved by 22% in its long denominator within a
day**, and neither figure carries a stated snapshot time in the sentence that quotes it. A rate
whose denominator drifts that fast needs its `as-of` stamped next to it.

### 8.2.4 The $860.64 — I get $862.14, and the n is wrong

`refusals.csv` holds **61 data rows**. Event-level:

```
by knob: MIN_SL_ATR_MULT 37 · day_plan.plan_mode 13 · min_risk_reward_ratio 10 · last_entry 1
by path: decision 54, arm 7        by day: 09-03 33, 09-02 28
cf_outcome: STOP 42, TARGET 13, FLAT_AT_HORIZON 5, NEVER_FILLED 1
SUM cf_usd = -726.14      SUM cf_usd_cme = -902.02      (n=61, 0 unparseable)
```

The audit reports a **44-opportunity** set (`:668-676`): TARGET 7, STOP 31, flat 5, never-filled 1,
**net −$860.64** session-flat, −$1,036.52 CME-day. I tested eight deduplication rules against it.
One lands:

```
dedup by cf_fill_ct   -> n= 55  T=10  S=39  F=5  N=1   sum=-862.14   cme=-1038.02
audit                    n= 44  T= 7  S=31  F=5  N=1   sum=-860.64   cme=-1036.52
                                                        diff = 1.50   diff = 1.50
```

**[T] The audit's dollar figure is essentially verified — my independent recomputation is $1.50
away (0.17%), and identically $1.50 away in *both* horizons, which reads as one small item or a
rounding, not a methodology gap. The audit's verdict that the refused set loses money survives my
check and I endorse it.**

**But the set that produces −$860.64 has n=55, not the n=44 the table sits under.** The outcome
breakdown (7/31/5/1) and the dollar total do not describe the same collection: the total needs 10
TARGETs and 39 STOPs. No rule I tried yields 44 rows *and* ≈−$861; `decision_cycle` dedup gives
n=38 / −$600.30, `entry+stop+target` gives n=61 / −$726.14. The audit never publishes its dedup
rule. So: **the sign and the magnitude hold; the denominator printed beside them does not.**

Two smaller n-hygiene points in the same section:

- `refusals.csv` has **37** `MIN_SL_ATR_MULT` rows; the audit's §8 ledger says "min-SL — n=34
  events". The difference is exactly the 3 EntryGate-leg-6 refusals that §11 lists separately as
  the daily-ATR bug (`:819`). The audit is internally consistent, but §8's "n=34" is silently a
  different population from the file's 37, and a reader carrying "n=34" forward will be wrong.
- **§11's attribution tables are not exhaustive over the refusal set.** Summing them: 09-02
  (16+3+2+5) = 26, 09-03 (13+18+1+2) = 34, total **60**. The file has 61, and §1/§14 both say
  "61 refusal events". The missing row is the **09-02 01:51:52 ASIA `last_entry_cutoff`** — it
  exists in the evidence, it is counted in the headline, and it is assigned to no cause bucket.

### 8.2.5 "Missed by 8.2 points" — exact

```
v6 S2 long arm-enabled @ 29481.05, live 10:49:57 -> 11:27:53
  min low 10:49-11:27 (n=39 bars): 29489.25 @2026-09-03 11:26:00
  miss = 8.20 pts
```

**[T]** Reproduced to the tick against `two-day-audit.md:329-333`. Good work by the audit.

### 8.2.6 Arm 37's 83 bars — exact

```
bars with high >= 29543.75, after arm37 created 11:58:33 AND before 17:00 CT: n=83
  first: 2026-09-03 12:51:00   last: 2026-09-03 14:13:00
```

**[T]** `two-day-audit.md:472` says 83. It is 83. Reproduced.

### 8.2.7 The silence — reproduced to 60 seconds

Gaps ≥20 min in the 09-03 wake ledger (`cadence.csv`, n=180 rows that day):

```
11:20:33 -> 12:23:33    63.0 min
12:23:33 -> 14:18:24   114.8 min      <-- the outage
14:18:24 -> 14:45:00    26.6 min
14:45:00 -> 16:37:36   112.6 min
```

**[T]** The outage is real and independently confirmed from the wake ledger: **114.8 minutes with
no wake row**. The audit says 12:24:33 → 14:18:24 = 113m51s; the ledger's last row before the gap
is 12:23:33. A 60-second reconciliation between a log-line timestamp and a wake row, not an error.
I would only note that the audit presents this as *the* gap, while the same ledger shows a 63-minute
gap immediately before it and a 112.6-minute gap immediately after — the day had three, not one.

---

## 8.3 Auditing the audit — what does not hold

### 8.3.1 `price_traded_through: no` is recorded for an arm whose own creation minute traded through it

`arms.csv`, row 37: `entry_px 29543.75`, `created_ct 2026-09-03 11:58:33`,
`nearest_approach_pts 9.5`, `nearest_approach_ct 2026-09-03 12:00:00`, `price_traded_through no`.

The bars around it:

```
11:57 h=29540.25   dist to 29543.75 = +3.50
11:58 h=29548.00   dist to 29543.75 = -4.25     <-- the arm was created at 11:58:33
11:59 h=29533.00   dist = +10.75
12:00 h=29534.25   dist = +9.50                 <-- this is the recorded "nearest approach"
```

**[T] The nearest-approach measurement demonstrably begins at 12:00:00 — it skips the 11:58 and
11:59 bars, which are the only bars in the arm's life where the market was at or through the
entry.** The 11:58 bar printed 29548.00, **4.25 points through a short limit at 29543.75**.

I cannot prove the 29548.00 print came at or after 11:58:33 — one-minute granularity cannot resolve
33 seconds, and it may have printed in the first half-minute. That is exactly my point: the honest
label for `price_traded_through` on this row is **CANNOT-DISTINGUISH**, not `no`. And it matters,
because "never traded through, never filled" is the evidence the audit uses to move arm 37 out of
the arm bucket and into the outage bucket (`:462-516`, "the case that looks like a broker defect
and is not").

### 8.3.2 The 53-minute error — and the lunch window it lands in

`two-day-audit.md:335-336`:

> "Price did reach 29476.00 at 11:28 CT — one minute *after* v6 was superseded, and v7's two long
> scenarios had no arms. Cause: **`never_reached`**, by 8.2 points and one minute."

From the audit's own bar file:

```
bars after 11:27 with low <= 29476.00: n=1
    2026-09-03 12:21:00  l= 29476.0
bars printing exactly 29476.0 anywhere in the day: [('2026-09-03 12:21:00','29476.0','29486.0')]
11:28  l=29494.00   11:29  l=29493.00   11:30  l=29493.25
```

**[T] Price reached 29476.00 at 12:21 CT, not 11:28 CT. The error is 53 minutes, and 29476.00
prints exactly once all day.**

This is not pedantry, it changes the cause. "Missed by 8.2 points and one minute" reads as bad
luck — a hair's breadth, nothing to fix. The truth is that the level was **not touched again for
54 minutes**, and when it finally was touched, at 12:21 CT, the system was **inside its own lunch
no-trade window** (12:00–13:30 CT, `kernel/no_trade_band.go:42`, a hardcoded constant with
`Source: SourceCodeConstant`, `:98`) and **three minutes from going silent** (last decision cycle
12:22:44, `d6_timeline.csv`).

So the correct causal chain for the day's only arm-enabled long is: *authored at a price the market
had already left; not revisited while any version held it; revisited 54 minutes later into a
window where the system refuses to trade by construction.* That is three separate defects wearing
one label. `never_reached` conceals all three.

### 8.3.3 The audit's own timeline contradicts its cadence verdict

`d6_timeline.csv`, a file the audit committed:

```
2026-09-03 10:30:29..11:15:01, 12 planner wakes SKIPPED on the class-47 30m cooldown, log,
  "n=4..15; price ran 29367 -> 29538 in this window"
```

`two-day-audit.md:38` says cadence "suppressed **nothing**". `:1035-1038` says "across both days
it suppressed **zero** wakes — its one enforcing firing was fast-market-exempted". The timeline
file says twelve, in that window alone, while price ran 171 points. §8.3.4 measures it.

### 8.3.4 Measured: 513 suppressed wakes, 20 of them class-47 cooldown

```
2026-09-02: rows=408 skipped=366  {'min_interval': 366}
2026-09-03: rows=180 skipped=147  {'min_interval': 127, 'cooldown_enforced': 20}
TWO-DAY skipped total n=513       {'min_interval': 493, 'cooldown_enforced': 20}
```

The 20 `cooldown_enforced` rows, in four clusters:

```
10:30:29 10:32:29 10:34:29 10:35:01 10:36:29 10:38:29 10:40:29 10:42:29 10:44:29   (NY v5, 9)
11:12:33 11:14:33 11:15:01 11:16:33 11:18:33                                       (NY v6, 5)
19:08:06 19:10:06 19:12:06 19:14:06 19:16:09                                       (ASIA v2, 5)
20:22:03                                                                           (ASIA v4, 1)
```

**[T] Twenty wakes carry `reason=cooldown_enforced` on 2026-09-03, fourteen of them between
10:30:29 and 11:18:33 CT.** "Suppressed zero" is false on the audit's own data under any reading
of "suppressed" I can construct. Even the narrowest — distinct enforcing *episodes* rather than
rows — gives **four**, not one.

Whether it *cost* anything is a separate question and the audit may still be right that it cost
little; a skipped planner wake is not a skipped trade. But "~0%, suppressed nothing" is an
unearned verdict, and it is the one bucket the audit rounds to zero and then stops looking at.

**And the exemption that was supposed to catch this did not fire, on an input that disagrees with
the tape.** The 10:30:29 row stamps `move_pts_15m = -29.75`, and so do the eight rows after it.
The actual completed 15-minute move at that moment, recomputed from the bars:

```
actual 15m move ending 10:30 CT: +64.75 pts (29471.00 -> 29535.75)
actual 15m move ending 10:44 CT: -29.50 pts (29535.50 -> 29506.00)
```

The fast-market exemption needs ≥1.5×ATR5m; ATR5m at the NY reads was 29.02 and 36.41
(`d6_timeline.csv`), so the bar is **43.5–54.6 points**. **|+64.75| clears it. |−29.75| does not.**
The value the ledger acted on is, to within a quarter point, the move that ended *fourteen minutes
later*.

**[T]** I state the measurement, not the mechanism: from committed data alone I cannot distinguish
a stale input in the live cadence gate from a back-fill artifact in how this CSV was written, and
I will not pretend otherwise. **It is a CANNOT-DISTINGUISH that is worth five minutes on the
owner's machine**, because if it is the former, the fast-market exemption — the one safety valve
that is supposed to let the system re-plan during exactly the move it missed — is being fed a
lagged number and cannot fire when it matters.

---

## 8.4 The attribution split has no denominator and no unit

This is the finding I care most about, because every downstream decision inherits it.

§11 is the only place the audit publishes counts. Two-day totals, taken straight from its tables
(`:814-841`):

| headline bucket | what §11 actually counts | unit | two-day total |
|---|---|---|---|
| planner shape (~55%) | long scenarios authored without `arm.enabled` | **scenarios** | 55 + 22 = **77** |
| gates (~5%) | min-SL + strict + R:R events | **refusal events** | 26 + 34 = **60** |
| outage (~30%) | "1 window" | **a window** | **1** |
| no setups (~10%) | never_reached + marketable_guard + parked + defects | **arms** | **8** |
| cadence (~0%) | — | — | **0** |

If you take those counts as the denominator — the only arithmetic the report supplies — you get
planner shape **77/146 = 52.7%** (close to 55%, so the intent is visible), and then **gates 41.1%**
and **outage 0.7%**. That is the exact inverse of the published 5% and 30%.

So the percentages are **not** count shares. Are they dollar shares? They cannot be either: the
window's realised loss is −$521.50, of which −$381.50 (73%) fell on 09-02 — *before* the outage,
which happened on 09-03 and produced no loss at all, only foregone profit. A dollar denominator
puts the outage at 0% too.

**[T] The split is a judgement, and a defensible one. It is presented as a measurement, and it is
not one.** Nowhere in 1,082 lines does the report state the unit the shares are shares *of*, or
the denominator, or the weighting that converts "one window" into 30% and "60 refusal events" into
5%. The four buckets are counted in four different currencies and then added to 100%.

**The buckets are also not mutually exclusive, and one item is double-counted.** The "no setups
~10%" bucket is anchored on "the day's only arm-enabled long missed by 8.2 pts". But the audit
itself says at `:1019` that **"71% of all arm-enabled scenarios were authored at prices their own
version never saw"** — which is a *planner-shape* defect by the audit's own definition. An arm
authored at an unreachable price is not a market that offered no setup; it is a plan that aimed at
the wrong price. The same event is the evidence for bucket 2 and an instance of bucket 3. And
temporally, the planner-shape bucket covers all of 09-03 *including* the 12:24–15:08 window that
the outage bucket also covers, with no statement of how the overlap was divided.

**What I would do instead**, and it costs nothing: publish the split in dollars of *realised loss*
and, separately, in **points of foregone move with the count of entry opportunities**, and let the
two disagree openly. The outage's honest entry is not "30%"; it is "no host from 12:24 to 14:18,
blind to ~15:08, across the 13:15 high — **0 realised loss, unquantified foregone**". A veteran
reading "30%" thinks a third of the money went there. None of it did.

**None of this rescues the gates.** My §8.2.4 recomputation agrees with the audit that the refused
set loses money, and I would not loosen a single leg on this evidence. The 5% is the one bucket
whose *direction* I independently confirmed. It is the ~55% and the ~30% that are asserted.

---

## 8.5 10:00 CT, 2026-09-03 — what I would have done

At 10:00 CT the tape looked like this (bars, verified):

```
09:55 29313.00 → 09:59 c=29363.25    (the 09:56 bar dumps to 29246.00 on 16,450 lots and is bought back whole)
10:00 o=29363.50 h=29390.50 l=29362.00 c=29380.75  v=13,176
10:15 c=29471.00      10:30 c=29535.75
```

Context I would have had on the screen: the day's low was 29075.00 at 00:23 and 29101.75 at 06:59;
London closed its range +55.00 into 08:00; the 08:30 opening drive ran +95.75 in fifteen minutes
(`tape_moves_0903.csv`); my own short at 29285 had been stopped at 29355 forty minutes earlier;
and price had just absorbed a 79-point liquidation candle at 09:56 and closed back above it. The
`10:00→10:15` window went on to be **the largest fifteen-minute move of the day, +101.50**
(`tape_moves_0903.csv`, and the audit's own `tape.csv` flags `largest_15m_move +101.50 @ 10:00`).

**[I] I would have been long by 10:03 and I would have been long for the wrong-looking reason: my
own stop had just been run.** A short stopped at 29355 on a day whose regime read `up/NORMAL` on
all 17 planner reads is not a bad trade to be exited from, it is *information* — the market paid
me to find out which way it was going. Thirty years of this and the single most reliable tell I
have is a fade that gets run in the first hour of a trend day. **[I], untested here, and I want
that label on it.**

Mechanically: buy stop above the 09:59 high (29367.75), or market on the 10:02 close at 29403 once
the 09:56 low held. Stop under the 09:56 reclaim low — call it 29340, a 63-point stop from 29403.
First target the round 29500, then trail. It filled 29500 at 10:26 and 29535 at 10:30. On one MNQ
at $2/point that is roughly **+$194 to the first target**, against a −$140 realised day.

**Three things about that trade the system could not have expressed**, and they are the real
answer to "what would it have needed":

1. **It is a continuation entry, not a touch.** I am buying strength, above the last swing, with
   no pullback. §8.6.1 shows every long the machine could place that day was a limit *below*
   the market.
2. **It is a re-entry in the opposite direction, minutes after a stop-out.** The system armed a
   re-entry cooldown at 09:20:45 (`close_sync.go:206`, per `d6_timeline.csv`) and, from 14:59:03,
   `one_open_position` forbids flips outright (`trader/entry_gate.go:272-282`, *"no adds, no
   flips"*). Flipping after a stop is not an error to be guarded against. It is most of the money
   on trend days.
3. **It is discretionary size.** I would have added at 29500. The system is one contract, and
   §8.6.5 shows the add is refused by construction.

### 8.5.3 The number nobody has written down

Arm 35 filled and closed at 09:20:45. Arm 36 was born and cancelled in the same second at 09:20:47.
Arm 37 was not created until 11:58:33.

```
EMPTY BOOK 09:20:47 -> 11:58:33 = 2:37:46 (158 min)
  bars n=158  open 29348.75  close 29532.00  low 29246.00@09:56  high 29548.00@11:58
  open->high = +199.25 pts ;  low->high = +302.00 pts
```

**[T] For two hours and thirty-eight minutes on the strongest trend day in the window — through
the 10:00–10:15 impulse, the 10:15–10:30 continuation, and the whole grind to 29548 — the system
had zero resting orders, on either side, while the market travelled +199.25 points from where it
started.** During that same stretch the AI decision loop logged **92 consecutive `wait` decisions**
(`d6_timeline.csv`, ids 37078..37169) and the cadence gate skipped **fourteen** planner wakes
(§8.3.4).

That is the whole audit in one line, and it belongs at the top of it. Not "the planner arms shorts
and not longs" — *the book was empty during the move*.

---

## 8.6 Why the system could not — the code path, construct by construct

Every citation is the dev tip in this tree. Where 09-03 differed I say so from the code's own
dated comments.

### 8.6.1 The armable vocabulary — the primary block

`kernel/armed.go:17-27`:

```go
func ArmableCondition(condition string) bool {
	switch strings.ToLower(strings.TrimSpace(condition)) {
	// reclaim added 2026-09-04 by owner ruling (arms-follow-bias, B): it arms
	// as a STOP-ENTRY beyond the reclaim trigger (ArmKindFor). Until then every
	// long-side play the planner favoured was un-armable, which is why long
	// arm-enablement sat at 4.3% while shorts ran at 44%.
	case "fvg_entry", "reject", "breakdown_continue", "breakup_continue", "reclaim":
		return true
	}
	return false
}
```

**On 09-03 the armable set was that list minus `reclaim`** — the comment dates the addition to the
following day. Now subtract what cannot be placed:

- **`fvg_entry` is SHADOWED by default** — `kernel/condition_status.go:26-29`,
  `defaultConditionStatus = {"fvg_entry": shadow, "breakout_retest": shadow}`. A shadowed
  condition is authored and scored but refused at `trader/entry_gate.go:230-235`, leg 4:
  *"condition %s is SHADOW (0C) — authored + E8-scored, never placed on any path"*.
- **`breakout_retest` is never armable at all** — `kernel/armed.go:11-14`: *"breakout_retest was
  EXCLUDED by the grand-audit response wave (F4, 2026-08-28): its replay expectancy is negative at
  every R-floor"* — **and** shadowed by the same default map. Two independent blocks.

**So the conditions that could actually place a long order on 09-03 were: `reject` and
`breakup_continue`. Two.** And both rest as limits below the market:

- `reject` is a **fade, touch-only** — `kernel/entry_law.go:151-153` refuses a close-confirm on it
  by name: *"fade_requires_touch (a %s fade enters on the touch at the level, never on a
  close-confirm)"*. `kernel/armed.go:52-59` prices it at the anchor, one tick into the trade's
  favour. A long `reject` is a bid parked under support.
- `breakup_continue` is the continuation play and **it is still a limit** —
  `kernel/arm_kind.go:41-49` puts it in the `ArmKindLimit` branch with the comment
  *"the PULLBACK limit at the broken level… ArmSpecValid requires entry_mode=pullback and it is
  NOT flipped here: the executor's stop_entry branch remains the no-retest FALLBACK it has always
  been, not an authored primary."*

**[T] Conclusion, from the code alone: on 2026-09-03 there was no construct by which the system
could buy strength. Every placeable long entry was a resting limit waiting for price to come
back.** On a day that went +483.25 points and, per §8.2.5, came back to its one authored long
price exactly zero times while that price was live.

### 8.6.2 What the planner actually wrote — the measurement

I classified all 41 of 09-03's scenarios by condition keyword out of `plans.csv`:

```
  long  breakout_retest      9      <- never armable AND shadowed
  long  reclaim              8      <- not armable until 2026-09-04
  long  reject               2      <- armable; touch-only fade
  long  sweep_reclaim        4      <- armable only via the 2-leg SPLIT contract
  short reject              11      <- armable
  short sweep_reclaim        7
```

**[T] 17 of 23 long scenarios (73.9%) rode on conditions the machine could not arm at all on that
date. Adding the 4 `sweep_reclaim`, which need the exact 2-leg split contract
(`kernel/planner_prompt.go:727`: "legs[] are the sweep_reclaim SPLIT contract and nothing else —
EXACTLY 2 legs"), exactly 2 of 23 long scenarios (8.7%) used a plainly armable single-arm
condition. Both were fades.** Shorts: 11 of 18 (61.1%) on `reject`, armable, single.

**Not one `breakup_continue` was authored all day** — the single continuation condition in the
vocabulary that could have carried a long arm, on a day the plan itself labelled `day_type: trend`
with bias `long` from 09:15 CT. The vocabulary contained the tool. The planner never reached for
it.

This is the audit's central finding, sharper. The audit frames it as a *tendency* — "the planner
grants resting arms almost exclusively to short scenarios". It is not a tendency. **It is 73.9%
structural: the model was writing long plays in a vocabulary that had no arm behind it, and
nothing in the prompt told it so.** `kernel/arms_bias_coherent.go:38-40` concedes exactly this:
*"the 18 long plans that could never comply were leaning on a condition that was un-armable AND
shadowed, and the prompt had never named either fact."*

### 8.6.3 The decision path — open at 10:00, and silent anyway

The audit's in-force table (`:172-183`) is unambiguous: **`plan_mode=strict` entered force
09-03 11:10:33 CT**; `one_open_position` at **14:59:03**. So at 10:00 CT **neither was running**.
The decision path — `trader/entry_gate.go:151-174`, whose strict leg refuses *"every %s-path market
entry"* that is not the arm path (`:162`) — was **open**.

**[T] This is the sharpest correction I have to the trader's own framing of the question.** At
10:00 CT on 09-03 the gates did not stop the long. The decision loop was open, and it proposed
`open_long` **zero times in 575 decisions** (§8.2.2), logging 92 consecutive `wait` decisions
through the impulse. The gate chain only closed the decision path at 11:10:33 — *after* the move.
Blaming strict for the 10:00 CT trade is chronologically impossible, and the audit says so at
`:1003` (*"a rule that post-dates every loss cannot have caused one"*). The block at 10:00 CT was
the **model**, sitting behind an open door.

### 8.6.4 The pricing legs and the clock

For completeness, the constructs that would have met my 10:03 entry had anything proposed it:

- **min-SL floor**, `trader/entry_gate.go:261-270`, `dist < MinSLMult × ATR5m` → refuse.
  `kernel/min_sl.go:33` `MinSLATRMultDefault = 1.5`. At the NY reads ATR5m was 29.02 then 36.41,
  giving floors of **43.52 and 54.61 points** (`d6_timeline.csv`, and 1.5×29.02=43.53 ✓,
  1.5×36.41=54.62 ✓). My 63-point stop clears it. **This leg would not have refused my trade** —
  worth saying, because it is the leg everyone suspects.
- **R:R floor 2.0**, `trader/entry_gate.go:237-259`, resolved live (`knobs.csv`:
  `min_risk_reward_ratio 2.0`, from a boot line). 63-point stop → target ≥126 points → 29529 from
  29403. Filled at 10:29. **Clears, barely, and only because I sized the stop wide.**
- **Lunch 12:00–13:30 CT**, `kernel/no_trade_band.go:42`, hardcoded, `SourceCodeConstant` (`:98`),
  blocking at `trader/auto_trader_session.go:120-123`. **The day's high printed at 13:15
  (§8.2.1) — inside it.** And the one long level was revisited at 12:21 (§8.3.2) — inside it.
- **NY last-entry 14:30 / flat 14:45** (`d6_timeline.csv`), `trader/auto_trader_clock.go:387-414`
  and `:454-472`. Two arms in the window died to it (`arms.csv` ids 26, 32, `reason_class
  session-EOD`).
- **`one_open_position`**, `trader/entry_gate.go:272-282` — from 14:59:03, *"no adds, no flips"*.

**[I]** A trend-day rule set that goes dark for ninety minutes over lunch and flattens at 14:45 has
decided in advance that the afternoon is not worth trading. On 09-03 the afternoon was where the
high was. On a balance day I would agree with the rule. Making it unconditional on a `day_type:
trend` plan is the system arguing with itself.

### 8.6.5 The stop-entry seam — a live fix one env var from being inert

The 2026-09-04 fix routes `reclaim` to a stop-entry (`kernel/arm_kind.go:60-61`). That path runs
through `trader/armed_executor.go:871-873`:

```go
if r.Kind == "stop_entry" {
    if !stopEntrySeamOn() {
        continue // seam off → the leg stays armed (never on the wire)
    }
```

and `kernel/entry_law.go:105-109`: *"the stop_entry order path is NEVER sent on the wire until the
far-side AddOn has proven the frame (D-rule). **Default OFF.**"* — it is on only when
`STOP_ENTRY_SEAM=on`.

**[A] It was on in production.** `knobs.csv` records `stop_entry_seam, ON, env STOP_ENTRY_SEAM=on
in .env; boot line "stop_entry_seam=ON"` — resolved from a boot line, not a file default. **So this
is a fragility, not a live break**, and I will not dress it up as one.

The fragility is real though: **the `continue` at `:873` is silent.** No log line, no counter, no
refusal record. A `reclaim` arm on a machine without that env var stays in state `armed` forever
and the journal shows an armed long that simply never fills — indistinguishable from a level that
was never reached. The whole arms-follow-bias wave rides on one unlogged env var.

### 8.6.6 The coherence check that answers the 55% is not wired

`kernel/arms_bias_coherent.go:74` defines `BiasArmWarning` — the D2 check that fires when a plan's
bias direction carries no armed scenario, written expressly for the NY 09-03 v7 case
(`:5-7`: *"authored a long AND a short on the identical level 29543.75, both confirms true at
11:58 CT, and armed only the short"*).

```
$ grep -rn "BiasArmWarning\|BiasCoherentArmsHint" --include=*.go .
kernel/arms_bias_coherent.go:31    (const definition)
kernel/arms_bias_coherent.go:74    (func definition)
kernel/arms_bias_coherent.go:122   (used inside BiasArmWarning itself)
kernel/arms_bias_coherent_test.go:82,117
kernel/arms_bias_coherent_warn_test.go:45,52,60,80,83,92
```

**[T] Zero production callers. The warning is never emitted; the hint text never reaches the
model.** `ArmableConditionsLine` from the same file *is* wired
(`kernel/planner_prompt.go:731`), so the model is told which conditions can be armed — that half
shipped. The half that checks whether *this plan* can trade *its own bias* runs only in tests.

`.audit/arm_rules_trader_armed_executor_go_kerne.md:110`, committed 2026-09-04, found the same
thing and adds that `BiasCoherentArmsHint`'s own doc comment calls it *"the class-34/38 hint
registry entry that guards its tokens"* while **it is not in the registry** —
`kernel/prompt_contract.go` holds 19 `Site:` entries and none is `bias_requires_an_arm`.

**[I] This is the finding I would put in front of the owner first.** The two-day audit's #1 cause
was planner shape at ~55%. Two fixes shipped the next day: the armable-set change (wired, working
— `d7-arm-enablement-by-date.csv` shows 09-04 parity at 20.0%/20.0%, n=5 per side, far below any
floor) and the coherence warn (**not wired**). Half a fix for the largest bucket, and the boot line
reports 19 restrictions all stated in prompt, so nothing complains.

---

## 8.7 What the system would have needed

Ranked by what it buys, all [I] unless labelled:

1. **A long momentum entry that is not a limit.** The vocabulary needs a condition whose arm rests
   *above* the market on a long — the `reclaim`→`stop_entry` route now does this
   (`kernel/arm_kind.go:60-61`) and it is the single most important change in the wave. Extend the
   same treatment to `breakup_continue`: `arm_kind.go:41-47` deliberately leaves it a pullback
   limit and says so, and on 09-03 that decision cost the whole afternoon. **[R]** The
   momentum/time-series-momentum literature is unusually durable here — Moskowitz, Ooi &
   Pedersen, *Time Series Momentum* (JFE 2012), and Jegadeesh & Titman (JF 1993) for the
   cross-section; both survived out-of-sample and both say the same thing: on established trend,
   continuation beats reversal. The system's armable set encodes the opposite.
2. **Wire `BiasArmWarning`.** It exists, it is tested, it has no caller. §8.6.6.
3. **Make lunch and the 14:45 flat conditional on `day_type`.** The plan already computes `trend`
   vs `balance`; nothing downstream reads it for the clock. `kernel/no_trade_band.go:42` is a
   literal.
4. **Log the silent branches.** `armed_executor.go:873` (seam off) and the decision-path refusal
   recorder the audit found writes no log line and no counter (`:528-560`, its own withdrawn first
   reading). A gate you cannot see refuse is a gate you cannot audit — and it cost the audit a
   published conclusion.
5. **Alert on blind.** ~50 minutes of a live process logging `NT8 TCP link DOWN — NEW entries
   BLOCKED` raised nothing (`:1029-1032`). **[I]** In a live book that is the one that ends you;
   everything else on this list costs opportunity, that one costs the account.
6. **Stamp the fast-market exemption's input.** §8.3.4.

---

## 8.8 The 09-02→09-04 replay — **BLOCKED: NO STORE, NO BARS IN THIS ENVIRONMENT**

I did not run it and I did not approximate it. `d6_bars_1m_0903.csv` covers 09-03 only; there are
no committed 09-02 or 09-04 bars, no `armed_orders` beyond the 15 rows in `arms.csv`, and no
`plans.doc` JSON beyond one file (`d6_plan_0903NY_v2.json`). A three-day replay under current rules
needs all of it.

**The procedure, for the owner's machine.** Read-only throughout; five minutes.

```sql
-- 0) the scenario universe, with the two fields plans.csv omits
sqlite3 "file:$HOME/nofx/data/data.db?mode=ro" -header -csv "
SELECT p.trade_date, p.session, p.version,
       datetime(p.created_at,'-5 hours') created_ct,
       json_extract(p.doc,'\$.bias.direction')        bias,
       json_extract(p.doc,'\$.day_type')              day_type,
       json_extract(s.value,'\$.id')                  scenario,
       json_extract(s.value,'\$.direction')           dir,
       json_extract(s.value,'\$.condition')           cond,
       json_extract(s.value,'\$.arm.enabled')         arm_enabled,
       json_extract(s.value,'\$.arm.kind')            arm_kind,
       json_extract(s.value,'\$.entry')               entry,
       json_extract(s.value,'\$.stop')                stop,
       json_extract(s.value,'\$.target')              target
FROM plans p, json_each(json_extract(p.doc,'\$.scenarios')) s
WHERE p.trade_date BETWEEN '2026-09-02' AND '2026-09-04';" > /tmp/replay_scen.csv

-- 1) 1m bars for the three days (the replay tape)
sqlite3 "file:$HOME/nofx/data/data.db?mode=ro" -header -csv "
SELECT open_time_ms, datetime(open_time_ms/1000,'unixepoch','-5 hours') ct, open,high,low,close,volume
FROM bars WHERE symbol='MNQ' AND tf='1m'
  AND open_time_ms BETWEEN strftime('%s','2026-09-02 05:00:00')*1000
                       AND strftime('%s','2026-09-05 05:00:00')*1000
ORDER BY open_time_ms;" > /tmp/replay_bars.csv

-- 2) ground truth to check the replay against
sqlite3 "file:$HOME/nofx/data/data.db?mode=ro" -header -csv "
SELECT id,session,scenario,side,entry_px,stop_px,target_px,kind,state,reason,
       datetime(created_at,'-5 hours') created_ct, datetime(updated_at,'-5 hours') updated_ct
FROM armed_orders WHERE created_at >= strftime('%s','2026-09-02 05:00:00')*1000;" > /tmp/replay_arms.csv
```

Then, in a scratch dir, for each scenario row in creation order, apply the **current** rules — do
not re-implement them, call them:

```
kernel.ArmableCondition(cond)                      -- reclaim now TRUE (kernel/armed.go:23)
kernel.ArmKindFor(cond)                            -- reclaim -> stop_entry (arm_kind.go:60)
kernel.IsConditionShadowed(cond, base, sess, env)  -- fvg_entry, breakout_retest -> shadow
kernel.BiasArmWarning(doc, statuses)               -- WARN only; count, never block
trader.EntryGate(EntryIntent{...})                 -- legs 0..7, entry_gate.go:140-296
```

ARM if all pass. Then walk `/tmp/replay_bars.csv` forward from `created_ct` to the arm's death
(next version, session end, or invalidation) and fill on:

- `kind=limit`, long → `low <= entry`; short → `high >= entry`; **but first apply
  `limitMarketableWrongSide(price_at_creation, entry, side)` — if it is already through, the arm is
  cancelled at birth, not filled** (`armed_executor.go:955-962`; this is what killed 5 of 15 arms
  in the audit window, §8.9);
- `kind=stop_entry`, long → `high >= entry + 2*tick`; short → `low <= entry - 2*tick`
  (`entry_law.go:113-120`, offset 2), **and only if `STOP_ENTRY_SEAM=on`**
  (`armed_executor.go:871-873`).

Report: arms authored, arms placed, arms filled, and the counterfactual P&L at authored stop/target
— **split by side**, because the whole question is whether longs can now reach the market. **Do not
report a fill rate without n and the interval.**

**The one control that makes it worth running:** run it twice, once with `ArmableCondition`
returning the 09-03 set (drop `reclaim`) and once with today's set. The difference *is* the value
of the arms-follow-bias wave, measured on the tape that motivated it. Nothing else in this repo
measures that; `d7-arm-enablement-by-date.csv` gives 09-04 parity at **n=5 per side**, which the
conformance report itself declines to call a finding
(`2026-09-04-research-conformance.md:382-384`).

---

## 8.9 Which scenario types could have armed on a trend day — the qualitative answer

Under **current** rules (dev tip), for a **long** on a trending tape:

| condition | armable? | kind | rests | verdict on a trend day |
|---|---|---|---|---|
| `reclaim` | **yes**, since 2026-09-04 (`armed.go:23`) | **stop_entry** (`arm_kind.go:60`) | **above** the market | **The only long entry that can buy strength.** Needs `STOP_ENTRY_SEAM=on` (`armed_executor.go:872`) |
| `reject` | yes (`armed.go:23`) | limit (`arm_kind.go:48`) | below | Fade. Touch-only (`entry_law.go:151-153`). Needs a pullback that a trend day may never give |
| `breakup_continue` | yes (`armed.go:23`) | **limit** (`arm_kind.go:48`) | below, at the broken level | Named the continuation play; **implemented as a pullback limit**. Structurally cannot chase |
| `fvg_entry` | yes, but **SHADOWED** (`condition_status.go:27`) | limit | below | Refused at `entry_gate.go:230-235`. Cannot trade |
| `breakout_retest` | **no** (`armed.go:11-14`) **and shadowed** | — | — | Cannot arm. 9 of 23 longs on 09-03 rode on it |
| `sweep_reclaim` | split contract only (`planner_prompt.go:727`) | limit legs | at the sweep ref | Two legs, leg 2 chained on confirm. Not a momentum entry |
| `acceptance`, `hold` | **no** (`armed.go:17-27`) | — | — | AI path only, and the AI path is closed under strict |

**Structurally could NOT arm a trend-continuation long on 09-03:** everything above except
`reject` and `breakup_continue` — and both of those rest *below* the market. **Today, exactly one
can: `reclaim`, as a stop-entry, if the seam env var is set.** That is a genuine improvement and I
would say so to the owner's face. It is also a single point of failure with a silent off-branch.

**The mechanism that kills the rest, measured.** From `arms.csv`, the complete arm ledger
09-01→09-03, **n=15**:

```
state:        cancelled 12, filled 3        (fill rate 3/15 = 20%, 09-01 00:00 -> 09-03 23:31 CT)
reason_class: marketable-guard 5, filled 3, stale-arm-expiry 3, session-EOD 2,
              invalidation 1, cancelled-in-NT8 1
```

**[T] The single largest cause of arm death in the window is the marketable guard: 5 of 15
(33.3%)** — *"level accepted through — marketable, never placed"*
(`trader/armed_executor.go:955-962`). That guard is correct — a limit the market has already passed
would fill instantly at a worse price — but it is the **exact signature of a limit-only arm
vocabulary meeting a trending tape**. The level is gone before the order is placed. On 09-03 it
killed arm 36 in the same second it was born (`created_ct` = `updated_ct` = `09:20:47`, price
29358.75 vs entry 29351.05, per `d6_timeline.csv`).

You do not fix that by loosening the guard. You fix it by giving the planner an entry type that
rests on the *other side* of the market. Which is precisely what the 09-04 ruling did, for exactly
one condition.

---

## 8.10 Findings register

| # | finding | evidence | severity |
|---|---|---|---|
| D-1 | Audit's cadence verdict ("suppressed zero wakes") contradicted by its own `cadence.csv`: 513 skips over the two days, **20 `cooldown_enforced`** on 09-03, 14 in the 10:30–11:18 impulse | §8.3.4, `cadence.csv`, `d6_timeline.csv` | **high** — a bucket rounded to 0% and closed |
| D-2 | "Price reached 29476.00 at 11:28 CT" is wrong by **53 minutes**; it was 12:21 CT, inside the lunch no-trade window | §8.3.2, `two-day-audit.md:335`, `d6_bars_1m_0903.csv` | **high** — changes the cause of the day's only long |
| D-3 | `BiasArmWarning` — the fix for the audit's own #1 cause — has **zero production callers** | §8.6.6, grep across `--include=*.go` | **high** |
| D-4 | The 5/55/30/10/0 split has **no denominator and no unit**; the only counts published invert it (gates 41.1%, outage 0.7%) | §8.4, `two-day-audit.md:814-841` | **high** |
| D-5 | 21 of 23 long scenarios on 09-03 rode on conditions the machine could not arm; **zero `breakup_continue` authored** | §8.6.2, `plans.csv`, `kernel/armed.go:23` | **high** |
| D-6 | **158 minutes with an empty book** (09:20:47–11:58:33) while price ran +199.25 pts | §8.5.3, `arms.csv`, `d6_bars_1m_0903.csv` | **high** |
| D-7 | −$860.64 verified to $1.50 but computed on **n=55**, printed under a **n=44** table | §8.2.4, `refusals.csv` | medium |
| D-8 | `arms.csv` `price_traded_through=no` on arm 37, whose creation minute printed 4.25 pts through it; measurement starts at 12:00, arm born 11:58:33 | §8.3.1 | medium |
| D-9 | Fast-market exemption fed `move15=-29.75` when the completed 15m move was **+64.75**; threshold 43.5–54.6. CANNOT-DISTINGUISH stale-input vs CSV artifact | §8.3.4 | medium — cheap to check |
| D-10 | §11 attribution tables total 60 refusals against a stated 61; the 09-02 01:51:52 `last_entry_cutoff` is in no bucket | §8.2.4 | low |
| D-11 | Stop-entry seam defaults OFF; the off-branch `continue`s **silently**. On in prod (`knobs.csv`), so a fragility not a break | §8.6.5, `armed_executor.go:871-873` | medium |
| D-12 | Arm-enablement denominator moved 23→28 within a day; the quoted rate carries no `as-of` | §8.2.3 | medium |
| D-13 | Marketable guard is the top arm-killer, **5/15 (33.3%)**, 09-01→09-03 — the signature of a limit-only vocabulary on a trending tape | §8.9, `arms.csv` | medium |

**Research-law ledger for every belief I recommended acting on:** §8.7 item 1 is **[R]** (Moskowitz,
Ooi & Pedersen 2012; Jegadeesh & Titman 1993). §8.5's trade thesis — that a fade run in the first
hour of a trend day is a buy signal — is **[I]**, my experience, **untested on this tape**, and I
have not proposed it as a rule. Every number in §8.2, §8.3, §8.5.3 and §8.9 is **[T]**, computed by
me from a CSV committed in this repository, with the file named and the output shown.

**No secrets appear in this report.** Env variables are named (`STOP_ENTRY_SEAM`,
`MIN_SL_ATR_MULT`, `SHADOW_CONDITIONS`, `LIVE_CONDITIONS`, `STOP_ENTRY_OFFSET_TICKS`); no value
from a `.env`, no key, no token, no account name is reproduced.
