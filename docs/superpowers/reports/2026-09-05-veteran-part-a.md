# VETERAN REVIEW — PART A: the way it trades, the levels, the division of labour

**Sub-agent A · sections 1–3 · owner hoang · 2026-09-05 · READ-ONLY**
Reviewer role: thirty years in index futures, NQ since it listed, discretionary desks then my own
automated books, LLMs inside the loop. I have blown up an account and I have watched a proven edge
decay. Everything below is labelled `[R]` researched (source named) · `[T]` measured on this system's
own tape (n and interval given) · `[I]` my experience, untested here. I never dress `[I]` as fact.

---

## EVIDENCE BASIS — what I could and could not reach

This is a fresh cloud clone at `/home/user/nofx`. It is **not** the owner's machine.

**Could not reach (verified, not assumed):**

| Asked for | Status |
|---|---|
| `GET /api/health`, `/api/expectancy`, `/api/config/resolved` | **BLOCKED — no engine.** `curl -s -m 3 http://localhost:8080/api/health` returns nothing; no process listens on any port. |
| Every store query (`touch_outcomes`, `trader_positions`, `plans`, `bars`, `trade_excursions`, `candidate_pool`, `decision_records`, `armed_orders`, `plan_lifecycle_log`, `ab_confirm_log`, `nt8_order_snapshots`) | **BLOCKED — no SQLite file exists.** `find / -name "*.db" -o -name "*.sqlite*"` returns only OS mime/avahi files. `store/` source is readable; it holds no rows. |
| `~/nofx-analysis/` scripts, the live tape, NT8 logs, the journal | **BLOCKED — absent.** `ls ~/nofx-analysis` → No such file or directory. |
| `docs/superpowers/plans/VL-MASTER-PLAN-v2.md` | **BLOCKED — does not exist** in this tree (see `docs/superpowers/plans/` listing). |
| "Replay the last 10 sessions' plans against the tape" | **BLOCKED as specified** — substituted with the committed exports below. |

**Could reach, and did:** the full Go source; `AUDIT-CHECKLIST.md`; `SYSTEM-MAP.md`; `research/INDEX.md`;
~50 committed reports; and — decisively — **committed tape exports I computed on myself**, in a scratch
dir, with the commands shown inline:

- `docs/superpowers/reports/exports/2026-09-02-level-replay/` — `per_kind.csv`, `per_kind_1h.csv`, `episodes.csv` (1,168 episodes), `calibration_1h.csv`
- `docs/superpowers/reports/exports/2026-09-02-losses/` — `plans.jsonl` (23 plans), `decisions.csv` (544 executor cycles), `trades.csv`, `levels.csv`, `bars_1m.csv`
- `docs/superpowers/reports/2026-09-03-mc-drawdown-data/trade_sample.csv` (n=64 closed trades)
- `docs/superpowers/reports/2026-09-04-research-conformance-data/E-d3-mae-mfe-per-trade.csv` (61 trades) and `D5b-touch_outcomes-*.csv` (the live detector corpus)
- `docs/superpowers/reports/2026-09-04-two-day-audit-data/arms.csv`, `baseline.csv`

Every number below is either read from a cited `file:line` / `report:line`, or computed by me from one
of those CSVs with the command shown. **One reproducibility failure, stated up front:** the committed
`E-d3-summary-percentiles.csv` reports `atr_convertible n=30`; filtering the committed per-trade CSV for
rows carrying `floor_pts` gives **n=36** and I could not reconstruct the report's cohort from the
committed data. I therefore quote **my** cohort (n=36) and flag the discrepancy rather than inheriting a
number I cannot rebuild.

---

## SUMMARY — the verdict, first

**Can this system make money on NQ as configured today? No — and the reason is arithmetic, not luck.**

Over the only honest sample that exists (n=64 closed trades, ids 521–590, 2026-08-18 → 2026-09-01,
`2026-09-03-mc-drawdown-data/trade_sample.csv`), the system wins **33.87%** of decided trades and earns
**1.66:1** on the winners. Breakeven at a 1.66 payoff is a **37.59%** win rate. It runs 3.7 points of win
rate short of its own break-even line, and that is the whole deficit: **mean −$6.62/trade**. Meanwhile the
plans it writes claim a **median 2.55:1** — a payoff that would only need a 28.2% win rate. **The plan's
reward is fiction; the tape pays 1.66.**

Five things produce that gap, in the order I would fix them:

1. **The stop floor is placed exactly where losers die and nowhere near where winners breathe.** `[T]`
   Winners' worst adverse excursion never exceeds **0.661×** the `1.5×ATR5m` floor (n=10); losers' median
   is **1.014×** it (n=26). The floor is calibrated to the loss distribution.
2. **The R:R gate is a target-inflation machine.** `[T]` The planner writes targets that just clear the
   floor (6 of 17 arms land in `[2.00, 2.30)`), and only **3 of 36** real trades ever produced enough
   favourable excursion to reach a 2.0R target measured off the *minimum* stop. That 3/36 is an upper bound.
3. **The AI decision path is inert, and when it fires it is wrong.** `[T]` 544 executor cycles on
   09-01/09-02 produced **492 `wait`, 5 `open_long`, 0 `open_short`**. Four became positions; all four lost
   (−$381.50). The next day, 575 decisions produced zero `open_long` into a +483-point rally.
4. **The resting-arm layer is fade-only, and a quarter of its arms are dead on arrival.** `[T]` 17 of 65
   scenarios carried an arm; **15 were `reject`** (a fade). Of 15 arms, **3 filled (20%)**, and **4 were
   created and cancelled in the same instant** because price had already traded through the level.
5. **One session is the entire loss.** `[T]` ASIA: n=16, **−$552.43**. Everything else: n=48, **+$128.50**.
   The code already ships ASIA disabled (`kernel/session_registry.go:93`); the running config overrode it.

**Where an edge could come from, if one exists here:** not the levels. On the live detector corpus the only
cell that is statistically distinguishable from a coin flip is **RTH-L, which BREAKS 68% of the time**
(20/63 hold, Wilson [0.216, 0.440]) — and RTH-L carries the *highest* average score in the candidate pool
(1.60) and seats 5 of 5. The system's best-graded level is the one the tape says to trade *through*, and
the fade-only arm layer puts it on the wrong side of exactly that. The edge, if any, is in **session
selection, stop placement and exit discipline** — the three things that are currently either unmeasured,
mis-calibrated, or switched off.

---

# 1 · THE WAY IT TRADES

## 1.1 The arithmetic that decides everything

Command and output (my computation, on the committed MC sample):

```
$ cd docs/superpowers/reports/2026-09-03-mc-drawdown-data && python3 -c "..."
n 64 · wins 21 · losses 41 · flat 2
sum -423.93   mean -6.624
avg win  114.67   avg loss -69.07
realized payoff ratio (avgW/avgL) = 1.660
win rate (flats excl)             = 0.3387
breakeven win rate at 1.660       = 0.3759
win rate needed at planned 2.55   = 0.2817
```

`[T]` n=64, ids 521–590, 2026-08-18 → 2026-09-01. This reconciles exactly with
`2026-09-03-mc-drawdown.md:119-124` (sum −423.93, mean −6.624, p(win) 0.3387), so the sample is the
published one, not a private slice.

Now the planned side. I parsed every armed scenario out of the 23 committed plans:

```
$ cd docs/superpowers/reports/exports/2026-09-02-losses && python3 -c "..."   # plans.jsonl
armed scenarios with full arm{entry,stop,target}: 17
stop distance pts: median 28.69  min 17.25  max 74.50
planned R:R:       median  2.55  min  2.01  max  5.47
```

`[T]` n=17 armed scenarios, 23 plans, 2026-09-01 → 2026-09-02.

**Read those two blocks together.** At a 2.55 payoff and a 33.87% win rate this system prints money —
expectancy would be comfortably positive. At the payoff it actually realises, 1.66, it needs 37.59% and
gets 33.87%. **The entire deficit is the 35% haircut between the reward the plan promises and the reward
the tape delivers.** Nothing about the win rate is broken. The reward side is.

`2026-09-03-mc-drawdown.md:173-179` puts the confidence interval on it: mean −6.624, 95% CI
**[−31.27, +18.02]**, t = −0.527, and **~1,810 trades** needed to separate it from zero at power 0.8 —
about a year at six trades a day. `[T]` n=64.

I want to be precise about what that CI means, because it is the most-abused number in this whole
corpus: **it does not say the system is fine.** It says the sample cannot yet distinguish −$6.62 from
zero. A system that needs 1,810 trades to prove it is not losing is a system with no measurable edge,
and `[I]` in thirty years I have never seen an edge that needed a year of data to become visible turn
out to be real. Real edges in index futures show up in dozens of trades, not thousands. The honest read
of that CI is: **there is no evidence of edge here, and the burden is on the system to produce one.**

## 1.2 Where the reward haircut comes from — measured

I computed the excursion distribution against the actual stop floor. `floor_pts` in the committed CSV is
**exactly `1.5 × atr5m`** (I verified: `max |floor_pts/atr5m − 1.5| = 0`), so "MAE/floor" reads directly
as "how many stop-floor units did this trade go against me".

```
$ cd docs/superpowers/reports/2026-09-04-research-conformance-data && python3 -c "..."
atr_convertible n=36 (floor_pts present)   winners 10   losers 26
winners MAE/floor: [0.0, 0.0, 0.069, 0.1, 0.104, 0.205, 0.303, 0.51, 0.558, 0.661]
losers  MAE/floor: [0.0, 0.012, 0.07, 0.137, 0.434, 0.514, 0.677, 0.68, 0.721, 0.767,
                    0.839, 0.93, 0.989, 1.01, 1.014, 1.023, 1.041, 1.054, 1.08, 1.22,
                    1.362, 1.363, 1.472, 1.55, 1.605, 1.904]
MFE/floor  winners n=10 p50=1.555 max=2.281 | >=2.0R: 3
           losers  n=26 p50=0.312 max=1.880 | >=2.0R: 0
           all     n=36                      | >=2.0R: 3
```

`[T]` n=36, 2026-08-24 → 2026-09-03, MNQ, atr5m range 13.60–47.19 pts.

Three findings fall straight out of that block, and they are the heart of section 1.

**(a) The 1.5×ATR5m stop floor is calibrated to the losers.** No winner in the sample ever went more than
**0.661×** the floor against entry. Thirteen of twenty-six losers went **past** it. A stop at 0.7× the
floor — call it **1.05×ATR5m** — would have kept every single winner in this sample and cut the loss on
every loser by roughly 30%. `[T]` n=36, and I will label the weakness honestly: **ten winners is a thin
reed, and choosing 0.7 after seeing the data is in-sample fitting.** What is *not* in-sample fitting is
the shape: a clean separation where the winner distribution tops out below where the loser distribution
centres is the classic signature of a stop that is too wide, and `[I]` in my experience that shape is one
of the few things in this business that replicates. The min-SL floor was moved from 1.0 to 1.5 by owner
ruling on 2026-09-02 (`kernel/min_sl.go:40-68`, SYSTEM-MAP §4, labelled `[O]`). **That ruling moved the
stop away from where the winners live.** I would re-open it.

**(b) A 2.0 R:R target is not reachable on this tape.** Only **3 of 36** trades ever produced MFE ≥ 2.0×
the *minimum permitted* stop. Since every actual stop is ≥ the floor, every actual 2R target is ≥ 2×
floor, so **3/36 = 8.3% is a hard upper bound on the target-hit rate** (Wilson 95% on 3/36:
[2.9%, 21.8%]). The R:R floor resolves to **2.0** on the bound MNQ strategy — `2026-09-04-research-
conformance.md:462` (`min_risk_reward_ratio` = **2**, "LIVE — both seams") and `:670`, against a spec
value of 3.0 (`store/strategy.go:76 SafeDefaultMinRiskReward = 3.0`), recorded as drift **D-21** at
`:772` with the note that "nothing records why". `[T]`/`[O]`

**(c) The planner writes to the gate, not to the tape.** Of my 17 armed scenarios, **six** carry a planned
R:R inside `[2.00, 2.30)` — 2.01, 2.10, 2.17, 2.19, 2.20, 2.28. That is not a distribution of trade ideas;
that is a distribution of a model solving a constraint. The gate says "≥ 2.0", the model cannot move the
market, so it moves the target until the ratio clears, choosing whichever element of `target_chain`
satisfies the inequality. `[T]` n=17.

**This is the mechanism I would put in front of the owner first.** An R:R floor applied to a *model's own
target* is not a risk control. It is an instruction to the model to write a bigger number in the target
field. The floor can only do real work if the target is derived from something the model does not
control — a measured level, a measured excursion distribution, an ATR multiple — and then the floor
becomes a *filter that rejects trades*, which is what it was meant to be.

## 1.3 The exits: BE/trail suspended is the right call, and re-enabling it would not help

`trader/exit_mechs_suspend.go:14-27` suspends breakeven and the ATR trail by default, citing Round-7
research ranking ATR/Chandelier trails "in the worst group of 15 exit families across 567,000 backtests"
`[R]`, and the system's own "$719.50 of giveback with ZERO trail EXITS ever recorded" `[T]`.

The census disagrees — SYSTEM-MAP §8 records the owner ruling breakeven ON at +40 pt, and flags the
contradiction as drift D-3, "which ruling stands is open". **I can settle it with the excursion data.**
From `E-d3-summary-percentiles.csv`: `cohort_losers` MFE **p50 = 17.5 pts, p80 = 36.0, p95 = 58.5**. A
breakeven trigger at +40 points would not have fired on the median loser, or the 80th-percentile loser.
It would have touched roughly the worst ~12% of losers and, on the winner side, would have converted
winners whose MFE briefly exceeded 40 into scratches. `[T]` n=43 losers, n=18 winners (report cohort).

**The suspension should stand.** `kernel`'s comment is right and the owner ruling is wrong on this
evidence. What the tape actually shows is a *giveback* problem, not a stop-management problem: winners'
median MFE is **69.25 pts** ($138.50) while the median realised win is **$106.00** (53 pts) — the median
winner hands back about **24%** of its best price. `[T]` The fix for giveback at a 1.66 payoff is not a
trailing stop; `[I]` it is a scale-out or a target that sits where the excursion distribution actually
tops out — around 1.5× floor, not 2.5×.

## 1.4 One contract, EOD flat, no lunch — and the session that eats the account

**One contract, SIM, MNQ at $2/pt** is confirmed: `futuresOrderQuantity` caps via
`maxFuturesContracts=2.0` (`trader/auto_trader_orders.go:25`) but 0B's `ClampStageAContracts` caps at 1;
`2026-09-03-mc-drawdown.md:44-48` verifies size 1 on rows 587–590 and $2/pt on id 590. `[T]`

**Every guardrail is off.** `2026-09-03-mc-drawdown.md:33-41`: `guardrails_enabled = False`,
`daily_loss_enabled = False`, `daily_profit_enabled = False`, `max_daily_trades_enabled = False`,
`max_contracts_enabled = False` — "each one is individually disabled as well… Nothing in this report
describes a control that is currently protecting the account." `[T]`/`[O]`

The counterfactual at `:161-169` is worth the owner's attention because it cuts against intuition: the
**$450 daily loss limit forfeits nothing** (trips on 9.1% of days, and on that day it landed on the day's
last trade), while **`max_daily_trades = 3` trips on 81.8% of days and forfeits $24.54/day** of realised
P&L. `[T]` B=10,000, 11 session-days. `[I]` That is the usual result and I would act on it: **turn the
loss limit on, leave the trade cap off.** A daily loss limit is cheap insurance against the one day the
model goes mad; a trade cap is a tax on the days it is right.

**EOD flat at 14:45 CT** (`kernel/session_registry.go:106-109`, session end == flat by owner contract) and
**lunch 12:00–13:30 CT** plus **first-5-minutes** no-trade (`kernel/no_trade_band.go:37,42`) leave NY with
**280 tradeable minutes**. `[I]` That is a defensible NQ day-trading window and I would not change it. A
14:45 flat gives up the 15:00–15:15 imbalance, which is a real thing, but it also gives up the way that
window punishes a system with no discretionary hand on it. Keep it.

**Now the session that actually matters.** I split the MC sample by session:

```
$ python3 -c "..."   # trade_sample.csv
session    n       sum     mean  wins  p(win)  Wilson 95%
LONDON    21     24.00     1.14     7  0.3333  [0.172,0.546]
NY        20    202.00    10.10     9  0.4737  [0.273,0.683]
ASIA      16   -552.43   -34.53     2  0.1333  [0.037,0.379]
(blank)    7    -97.50   -13.93     3  0.4286  [0.158,0.750]

total -423.93 (n=64)   ASIA -552.43 (n=16)   ex-ASIA +128.50 (n=48, mean +2.68)
one-sided binomial P(<=2 wins in 15 | p=0.3387) = 0.0733
```

`[T]` n=64. **ASIA is more than the entire loss.** Remove it and the sample flips from −$423.93 to
**+$128.50**.

I will label the weakness precisely, because this is exactly the kind of cell that seduces people:
n=16, the binomial p-value is **0.073** (not significant at 0.05), and I tested three sessions, so
multiplicity applies — the worst of three will always look bad. **On the number alone I would not act.**

But it does not stand alone, and this is where thirty years is worth something. Three independent things
point the same way: (i) the code itself ships ASIA with `Enabled: false`
(`kernel/session_registry.go:93`) and LONDON with `Enabled: false` (`:102`) — the running config
overrode a default someone chose deliberately; (ii) 37 of 61 trades in the excursion corpus (61%) were
taken in those two overnight sessions; and (iii) `[I]` MNQ between 17:00 and 02:00 CT is the thinnest,
most gap-prone tape the contract offers, where a level-fade system with a 30-point stop and no
discretionary hand is picking up nickels in front of the Asia cash open. **Converging weak evidence plus
a strong prior is how you make this decision, and it says: honour the code's own default and stand down
overnight until there is a positive ASIA sample to point at.**

## 1.5 What a bad week looks like — and whether the owner can sit through it

`2026-09-03-mc-drawdown.md:127-139,142-152,208-220` `[T]` n=64, B=10,000:

| horizon | p50 maxDD | p95 | worst |
|---|---|---|---|
| 20 trades | $478 | $935 | $1,499 |
| 50 trades | $866 | $1,677 | $3,030 |
| 100 trades | $1,364 | $2,589 | $4,130 |

P(losing streak ≥ 4 within 50 trades) = **0.99**; ≥ 6 = **0.81**; ≥ 8 = **0.46**. The IID and block(5)
bootstraps agree within ~3% at every quantile, so there is no streakiness beyond what a 34% win rate
already produces, and the M6 trim (drop biggest win *and* biggest loss) moves drawdowns under 3%.

`[I]` This is the section I would read aloud to the owner. **Four losers in a row is certain. Six in a
row is a coin flip. Eight in a row happens four times in ten.** At one MNQ contract that is survivable
arithmetic; at any size it is the thing that ends accounts, because the operator intervenes at loss six
and turns a normal sequence into a permanent change of system. The report's own closing line is the
right instinct and I will sharpen it: **a $1,200 week is normal noise. What is not normal is a $1,200
week in which the trades stop arriving, or in which every loss sits in one session or one condition** —
and as section 1.4 shows, that second condition is already true today.

---

# 2 · LEVELS — what is seated, what is noise, what is missing

## 2.1 The query the box asked for, and what I ran instead

**BLOCKED — NO STORE IN THIS ENVIRONMENT.** The query I would have run:

```sql
-- would have run against data/data.db?mode=ro
SELECT level_kind, ordinal, COUNT(*) AS n,
       SUM(outcome='hold') AS hold, SUM(outcome='break') AS brk,
       1.0*SUM(outcome='hold')/NULLIF(SUM(outcome IN ('hold','break')),0) AS p_hold
FROM touch_outcomes
WHERE created_at >= '2026-08-15'
GROUP BY level_kind, ordinal
HAVING n >= 30
ORDER BY n DESC;
```

The committed CSVs at `2026-09-04-research-conformance-data/D5b-touch_outcomes-by-{kind,ordinal,session}.csv`
are that query's output, already grouped. I computed the Wilson intervals the box asks for:

```
$ python3 -c "..."   # Wilson 95%, cells above the n>=30 floor
kind        hold     n       p           Wilson 95%   amb%  verdict
DEMAND        44    85  0.5176 [0.4130,0.6208]   5.6%  coin flip
VWAP          43    70  0.6143 [0.4972,0.7195]  38.6%  coin flip
RTH-L         20    63  0.3175 [0.2159,0.4400]   7.4%  BREAK-biased
ordinal 1    118   242  0.4876 [0.4253,0.5503]   0.0%  coin flip
NY            77   153  0.5033 [0.4249,0.5814]  15.0%  coin flip
LONDON        60   120  0.5000 [0.4119,0.5881]  26.4%  coin flip
ASIA          25    47  0.5319 [0.3923,0.6667]  42.0%  coin flip
```

`[T]` live detector corpus, `D5b-*` CSVs. Every kind below n=30 is suppressed by the corpus's own floor
rule and I will not rank it.

## 2.2 The one signal in the corpus, and the system is on the wrong side of it

**RTH-L holds 31.75% of the time (20/63), Wilson [0.2159, 0.4400).** The upper bound is below 0.50. This
is **the only cell in the entire live corpus that is statistically distinguishable from a coin flip**, and
what it says is: *when price touches the prior RTH low, it goes through*. `[T]` n=63.

Now put that next to the grader. From `D5b-candidate_pool-by-kind.csv`:

```
level_kind, n, seated_n, avg_score, ...
RTH-L,       5,        5,      1.6      <-- highest avg_score in the pool
PDL,         3,        3,     1.92
VWAP,       10,       10,     1.44
DEMAND,     35,       16,   0.7747
OB,         67,        6,   0.1394
```

**RTH-L carries the highest average score of any high-frequency kind in the candidate pool (1.60) and
seats 5 of 5.** The scorer's most-favoured level is the one the tape says breaks two times in three. And
because the arm layer is fade-only (§3.2), seating RTH-L highly means resting *buy limits* at a level
whose measured behaviour is to be sliced through. `[T]`

That is not a subtle miscalibration. That is a sign error on the single measurable thing in the corpus.

## 2.3 The rest of the ladder is unfalsified doctrine

The published D1′ replay (`2026-09-02-level-kind-replay.md`) is the best piece of work in this repository
and its verdict is unambiguous. Pre-registered before any replay ran (`:10-129`), n≥200 holdout floor,
stationary-bootstrap and Osler random-level nulls, OOS split fixed in advance.

- **Pooled holdout p(hold) = 0.5413 [0.4968, 0.5852], n=484 — DOES NOT CLEAR** the bootstrap baselines
  (≈0.51–0.55). `:145-152` `[T]`
- **Every one of 16 kinds is TOO FEW** on holdout; largest cell VWAP−1σ n=63 against a 200 floor.
  `:154-178` "No kind is rankable; the table is descriptive ONLY. BH never runs." `[T]`
- The 1h variant (`:324-340`) reaches n=216 pooled at **0.6389 [0.5729, 0.7000]** — but the honest note at
  `:350-357` kills it: that sits at the **65.6th percentile of the Osler null**, "with n=216 the same
  number would arise from randomly-placed prices one day in three. **No verdict.**" `[T]`

**Two doctrines the replay actively contradicts, and which are still wired:**

1. **The consumed-touch belief.** `kernel/levels_role.go:28,107-118` routes consumed / 3rd-touch / far-HTF
   levels to `target_only, never entry` — labelled `[I]` in SYSTEM-MAP §2. The 1m replay found the
   **opposite**: 1st touch 0.5657 (n=99) vs 3rd+ 0.5294 (n=988) pooled, holdout 1st 0.6250 (n=40) vs 3rd+
   0.5290 (n=414) — "the FIRST touch holds MORE, the opposite of the belief" (`:182-191`). The live
   corpus agrees there is nothing there: ordinal 1 is **0.4876 [0.4253, 0.5503], n=242** — a pure coin
   flip on the largest single cell in the whole corpus. `[T]` The 1h replay found a *decline* (`:344-347`),
   i.e. the opposite of the 1m result. **A belief with no stable sign across two granularities of its own
   tape is not a belief; it is a knob.** I would strip the ordinal term out of the role assignment and
   let it back in only with a positive, pre-registered result.
2. **The scoring ladder.** SYSTEM-MAP §2 labels essentially the whole thing `[I]`: kind weights (structural
   1.0 · VWAP/POC 0.90 · … · zones 0.30), zone TF tiers 1.0/1.1/1.2/1.3, reversal bonus ×1.1,
   ConfluenceCap 3, the zoneSizeMult ladder, both freshness ladders, the 12-tick cluster tolerance
   (`kernel/levels_score.go:87-122,148-161,192-222,359-390,678-685`). `research/INDEX.md:34` already
   records the 08-27 level-system-verify verdict — **"weights off spec; grades not predictive"** — with
   Action **NONE (documented)**. That finding is nine days old and nothing moved.

**And the grade is not a filter.** From the 23 committed plans, 254 seated level rows:
`grade: A 193 (76%), C 47, B 14`. `[T]` A grader that says "A" to three levels in four is not grading; it
is decorating. The same pathology repeats one layer up (§3.1): 77% of scenarios are graded B.

## 2.4 Which kinds I would seat, which I would cut, what is missing

`[I]` throughout except where marked — and I want that understood, because the honest position is that
**this tape cannot yet rank levels and anyone who tells the owner otherwise is selling something.**

**Seat (mechanically defensible, cheap, and they anchor the session):** PDH/PDL/PDC, ONH/ONL, OR-H/OR-L
(first 5 min, `kernel/levels_intraday.go:111-139`), IB-H/IB-L, session VWAP. These are where the rest of
the market's stops and reference prices sit; `[I]` they earn their seat as *coordination points*, not as
predictors, and that is a different and more durable claim than "they hold 55% of the time."

**Cut or demote now:**
- **Round numbers.** 1m replay: RND-500 exploration **0.2609 [0.1255, 0.4647]** (n=23), RND-1000 holdout
  0.2000 (n=5), RND-250 exploration 0.3778 (n=45). Every cell that moves is *below* its null. `[T]`
  Hypothesis H4 pre-registered them as noise (`:99`) and nothing refuted it. They cost seats.
- **The zone family (S/D, OB, FVG, iFVG) as anything but confluence.** `D5b-candidate_pool-by-kind.csv`
  shows OB generating **67 candidates and seating 6** at avg_score 0.1394, and SUPPLY 18→2 at 0.1907 —
  the pool is dominated by a kind that almost never seats. `[T]` The system already knows this: FVG entry
  is **shadowed by default** (`kernel/condition_status.go:26-29`, "external null ×2 + own null") on the
  back of `2026-08-26-fvg-entry-model.md` — "no tradeable edge after honest costs" `[R]`. Good ruling.
  I would extend the same scepticism to OB, which is generating three-quarters of the candidate pool for
  6 seats.
- **RTH-L at score 1.60.** Not because it is noise — because it is **anti-signal** and top-weighted. `[T]`

**What is missing, and it is the important part:** every level in this system is a *price*. Not one of
them carries **who is positioned there**. `[I]` The things that actually made levels tradeable for me on
NQ were never the price itself — they were: prior-day volume *at* the level (a POC with 40% of the day's
volume behaves nothing like one with 8%); whether the approach is impulsive or grinding (velocity into
the touch); whether the level has already been swept in the overnight; and time-of-day conditioning.
The engine computes a 120-bin profile (`kernel/levels_volume.go:129-232`) and then throws the *shape*
away, keeping only POC/VAH/VAL as three prices. `[R]` for the underlying idea — this is the standard
market-profile / volume-at-price literature and the auction framework behind it — but **`[I]` for whether
it would work here, and I have not tested it on this tape.** It is the first experiment I would fund,
because it is the only proposal on this list that adds *information* rather than re-weighting prices the
system already has.

**And the honest structural answer to "which kinds would you seat":** `2026-09-02-level-kind-replay.md:365-369`
is right — "the honest lever is time (1m retention 90d + 1h retention both live), not a new threshold."
`[T]` The 1m tape is ~14 days. The n≥200 floor needs ~30+ holdout days per single-line kind. **Stop
tuning the ladder and let the tape grow.** Every hour spent re-weighting `levels_score.go` before that
sample exists is an hour spent fitting noise.

---

# 3 · PLANNER / EXECUTOR / GATE — the division of labour

## 3.1 Would I let an LLM author plans? Yes — but not this contract

`[I]` My position after using LLMs in a live loop: a model is excellent at *reading a situation and
enumerating what could happen*, and structurally bad at *committing to a price*. This system has the
split backwards. It asks the model for entry, stop and target — three numbers that decide the P&L — and
then uses gates to check the model's arithmetic against itself.

The measurements say the same thing. Across the 23 committed plans (65 scenarios, 09-01/09-02):

```
$ python3 -c "..."   # plans.jsonl
conditions: reject 19 · sweep_reclaim 16 · reclaim 12 · breakout_retest 11 · hold 7
never authored: breakdown_continue, breakup_continue, fvg_entry, acceptance
direction: long 56 (86.2%) · short 9 (13.8%)
quality:   B 50 (76.9%) · C 9 · A 6 · A+ 0
arm-enabled: 17/65 = 26.2%   long 10/56 = 17.9%   short 7/9 = 77.8%
arms by condition: reject 15 · sweep_reclaim 2 · everything else 0
```

`[T]` n=65 scenarios, 23 plans.

**Four things in that block are damning, and each has a code citation.**

**(a) The prompt names its own failure mode and the model does it anyway.** `kernel/planner_prompt.go:707`
instructs: *"The scenario MIX must follow the regime + day_type … do **NOT** default to 2 longs + 1
rally-rejection short on every day."* The measured output is **86.2% long**. `[T]` A prompt cannot
enforce a distribution. Only a validator can, and there is no validator on the direction mix.

**(b) The quality grade is a rubber stamp.** 76.9% B, zero A+. Combined with the level grader's 76% A
(§2.3), **neither of the two quality signals the plan carries discriminates anything.** The arm gate
refuses on "quality %s below min_scenario_quality %s" (`trader/armed_executor.go:1373`) — a leg that
cannot bind when three-quarters of the population sits at one value.

**(c) 17% of the model's output goes into a bin that can never trade.** `breakout_retest` was authored 11
times and is **shadowed by default** — `kernel/condition_status.go:26-29`: *"breakout_retest = shadow (no
evidence anywhere + 80.7% stop-out falsification)"* `[T]`. Authored, validated, E8-scored, never placed.
Add the never-authored `fvg_entry` and that is two of the nine conditions the schema advertises
(`planner_prompt.go:697`) that exist only to consume tokens and plan real-estate.

**(d) The plan degrades down its own list.** Splitting the MC sample by cited scenario slot:

```
S1  n=26  sum= +461.00  mean= +17.73  wins=10
S2  n=22  sum= -142.43  mean=  -6.47  wins=7
S3  n= 7  sum= -467.50  mean= -66.79  wins=1
S4  n= 2  sum= -177.50  mean= -88.75  wins=0
```

`[T]` n=64. **S1 alone is +$461; S2–S4 together are −$787.** `[I]` This is exactly what I would predict
and it is the most useful single fact in this report for prompt design: the model's *first* idea is its
real read, and everything after it is the model filling a schema because the schema asked for a list.
n=7 and n=2 in the tail are thin and I flag that — but the monotone decline across four slots, with the
sign flipping after S1, is not a cell I would dismiss.

**What I would have the LLM author:** the *narrative* — bias with its branch, day type, the levels that
matter and why, what would invalidate the read, and **one** scenario. Not four. Not entry/stop/target as
free-form numbers.

**Where human judgement goes back in:** the session enablement (§1.4 — the ASIA decision is a judgement
call the model will never make), the size, and a standing veto. `[I]` The one control I have never
regretted on an automated book is a human "not today" — not a knob, a switch, exercised before the
session opens and never during it.

**What the machine should own outright:** stop placement (from the measured excursion distribution, §1.2
— not from the model's opinion), target placement (same source), and the entry price (from the level
list, mechanically). Those are three numbers the tape can compute and the model cannot.

## 3.2 The arm layer is fade-only, and that converts an up-bias into short exposure

Of 17 arms, **15 are `reject`** — a fade at a level — and 2 are `sweep_reclaim`, whose leg 1 is also a
fade at the sweep reference (`kernel/entry_law.go:39-43`: reject = *"touch-entry at the level (limit),
stop behind structure by ≥2 ticks"*). **100% of arms in this window were fades.** `[T]` n=17.

That is not the planner's preference; it is the *mechanics*. `kernel/arm_kind.go:48` makes exactly
`reject, fvg_entry, sweep_reclaim, breakup_continue, breakdown_continue` armable as limits.
`fvg_entry` is shadowed; `breakup/breakdown_continue` were never authored. **The effective armable set on
this tape was `reject` and `sweep_reclaim` — both fades.**

Consequence, measured: **shorts are 13.8% of authored scenarios but 41.2% of arms** (7 of 17), because a
"reject" in a market that has been rising is a rejection at resistance. Long scenarios get an arm 17.9%
of the time; shorts 77.8%. `[T]` n=65.

**This replicates independently.** `2026-09-04-two-day-audit.md:1` (verdict table) measures the same
asymmetry on a *different* day: *"long scenarios were arm-enabled **4.3%** of the time vs **44.4%** for
shorts"* on 09-03. Two disjoint windows, same sign, same magnitude class. `[T]`

The owner's own code records the diagnosis. `kernel/arm_kind.go:56-62`, dated 2026-09-04 — **after** the
window I measured — makes `reclaim` armable as a stop-entry, with the comment: *"19 long plans were
stranded because every long play they wrote (reclaim, breakout_retest) was un-armable, so a long-biased
plan had no way to reach the market."* That is the right fix and I cannot measure its effect: it postdates
my data and there is no store here. **BLOCKED — NO STORE IN THIS ENVIRONMENT.**

`[I]` The general lesson is worth stating plainly, because it will recur: **the set of plays you can rest
an order for silently becomes your strategy.** This desk believes it is bias-driven and regime-aware. Its
executable book was "fade whatever we can reach with a limit."

## 3.3 A quarter of the arms are dead the moment they are written

From `2026-09-04-two-day-audit-data/arms.csv`, all 15 arms (ids 23–37, 09-01 → 09-03):

| outcome | n |
|---|---|
| **filled** | **3 (20%)** |
| cancelled — `level accepted through — marketable, never placed` | **5** |
| cancelled — stale-arm expiry (no order_update in window) | 3 |
| cancelled — session ended (EOD flat) | 2 |
| cancelled — `gate changed: min_sl` | 1 |
| cancelled in NT8 | 1 |

All five marketable-guard rows carry `price_traded_through = YES`. **Four of them (ids 27, 33, 34, 36)
have `created_ct` identical to `updated_ct`** — 11:27:07/11:27:07, 14:10:29/14:10:29, 22:15:01/22:15:01,
09:20:47/09:20:47 — and in each case `nearest_approach_ct` is in the *same minute* with a **negative**
`nearest_approach_pts` (−12.50, −1.70, −12.50, −8.95). `[T]` n=15.

**Those four arms were authored at a price the market had already left, and killed in the same instant
they were born.** That is 27% of the arm population. The guard itself is correct — it exists because of
the 2026-08-30 incident (`trader/armed_executor.go:955-961`: a marketable limit "would fill INSTANTLY at
a worse price (the S2 re-place loop: fill → stop-out → re-arm → fill…)") `[R]`. The guard is not the
problem. **The problem is upstream: the plan is authored against a stale snapshot.**

The latency is measurable. `2026-09-02-deepseek-e2e-audit.md:112-118` shows three planner attempts running
**01:32:54 → 01:37:44 — 4 min 50 s** for one authoring cycle, and `:320` records a single call at
`elapsed=270.3s`. `[T]` At an MNQ ATR5m of 13.6–47.2 points (§1.2), five minutes of authoring latency is
routinely one to three ATR of drift. **You cannot rest a limit at a price you computed five minutes ago.**

And there is a second stale hop underneath it. The placement decision reads
`price = bars[len(bars)-1].Close` (`trader/armed_executor.go:896-898`) — the last closed **1-minute** bar
(`kernel.AISVPBarInterval = "1m"`, SYSTEM-MAP §1) — and places only when
`math.Abs(price-r.EntryPx) <= band` (`:964`), where `band` is computed at `:904` as
`ARM_PLACE_TICKS × tick` = 100 × 0.25 = **25 points** (default `return 100`, `trader/armed_executor.go:43-51`). That check runs once per `runCycle` (`trader/auto_trader_loop.go:432`), on a ticker of
`ScanInterval` — `scan_interval_minutes` **default 3, minimum 3** (`store/trader.go:29`).

**So an "arm" is not resting at the broker.** It is a conditional placement, evaluated at most every three
minutes, off a bar close, against a 25-point band that is roughly 1×ATR5m wide. `[I]` A level-fade order
that is not on the exchange book before price arrives is not a resting order; it is an intention. On NQ,
25 points is under a minute of ordinary movement. **The architecture promises resting liquidity and
delivers a polling loop.**

*(Note for the record: the dispatch describes "a 2-min AI executor loop"; the code default is 3 minutes,
minimum 3, and SYSTEM-MAP §12 labels `scan_interval_minutes` **`[X]` — never tape-tested.** I could not
read the resolved runtime value — **BLOCKED: `GET /api/config/resolved` unreachable, no engine.**)*

## 3.4 The decision path does almost nothing, and what it does is wrong

I parsed every executor decision in the committed export:

```
$ python3 -c "..."   # decisions.csv, 09-01 + 09-02
decision rows: 544
actions: wait 492 · open_long 5 · open_short 0
action x cited scenario: (wait,-) 492 · (open_long,S3) 2 · (open_long,S2) 2 · (open_long,S4) 1
ai_request_duration_ms: n=536 p50=7504 p90=45341 max=600000
```

`[T]` n=544 cycles, 2026-09-01 → 2026-09-02.

**Five entry intents in 544 cycles (0.92%). All long. Zero shorts.** Four became positions — ids 587–590 in
`exports/2026-09-02-losses/trades.csv` — and **all four lost**: −62.50, −65.00, −155.00, −99.00 = **−$381.50**.
Their MFE values are 25.75, **0.00**, 10.25, **1.00** points. `[T]` Two of the four never showed a single
tick of profit. All four carry `plan_matched=1` and `adherence_grade` B/B/A/A — **the system executed its
plan faithfully, and the plan was wrong.**

Then the very next day, `2026-09-04-two-day-audit.md:1` records **"0 `open_long` proposed in 575 decisions"**
during a **+483-point rally**, with the machine regime reading `up/NORMAL` on all 17 planner reads and the
plan bias `long`/`trend`. `[T]`

Put those together: across roughly **1,119 executor cycles over three consecutive sessions**, the AI
decision path produced **five entry intents, all long, all losers**, and produced **none at all** on the
one day the tape trended 483 points in the direction its own plan named. `[I]` That is not a
mis-calibrated model. That is a component that is not contributing signal — and a component that produces
no signal but occasionally produces a position is strictly worse than no component, because it converts
an absence of edge into an execution of edge-free trades.

## 3.5 The gates are net-positive, and that is the one piece of good news

I want to be fair to the thing that is working. `2026-09-04-two-day-audit.md:1` (cause table): **"GATES TOO
TIGHT — ~5%"**, with *"61 refusal events over 44 distinct opportunities, and the refused set **loses
$860.64** on the actual tape."* `[T]` And `research/INDEX.md:49` records the 08-28 weekend audit: *"gates
net −$511.8 SAVING."* `[T]`

**The gates are earning their keep.** The instinct to loosen them because the system is not trading is the
instinct that ends accounts, and the data here says the opposite: the refusals are the profitable part of
the book. The problem is not that too much is refused. It is that what *passes* has no edge.

One structural note the audit records and I would not let slide: `plan_mode=STRICT` closes the decision
path entirely (`trader/entry_gate.go:160-172`, four refusal variants), a defect the audit says "cost
nothing in this window" (`2026-09-04-two-day-audit.md:1`). `[I]` A dead gate leg that happens to be
harmless today is a live hazard tomorrow; it should be either wired or deleted, not left dormant.

## 3.6 The division of labour I would actually run

`[I]` — this is my judgement, and it is untested on this tape. I am stating it as a design, not a result.

| Layer | Owns | Why |
|---|---|---|
| **Human, pre-session** | session on/off, size, standing veto | The ASIA call (§1.4) is a judgement no model will make. Exercised before the open, never during. |
| **LLM planner** | bias + branch, day type, the level shortlist with reasoning, invalidation, **one** scenario | This is reading a situation — what the model is genuinely good at. S1 is worth +$461; S2–S4 are worth −$787 (§3.1). Stop asking for a list. |
| **Machine, deterministic** | entry price (from the level), stop (from the measured MAE distribution), target (from the measured MFE distribution) | Three numbers the tape can compute. Removing them from the model's hands kills the R:R-gaming loop (§1.2c) outright. |
| **Gates** | unchanged — they are net-positive | §3.5 |
| **Execution** | arms genuinely resting at the broker, or no arms at all | A 3-minute polling loop against a 25-point band is not a resting order (§3.3). |

The single highest-value change on that list is the third row, and it is worth saying why in one line:
**an R:R floor checked against a number the model itself chose is not a constraint — it is a prompt for a
larger number.** Derive the target from the tape and the same floor becomes a real filter that rejects
trades, which is what everyone believed it was doing all along.

---

# APPENDIX — box instructions I could not execute

| # | Instruction | Why | What I did instead |
|---|---|---|---|
| 1 | `GET /api/expectancy`, `/api/config/resolved`, `/api/health`; token via `cmd/gate-jwt` | **No engine running**; nothing listens on any port | Read the handlers and resolved defaults in code (`store/strategy.go:76`, `store/trader.go:29`, `kernel/session_registry.go:83-117`) and said so at each use |
| 2 | Query `touch_outcomes` by kind and ordinal with n and Wilson | **No SQLite store on disk** | Wrote the SQL (§2.1, marked BLOCKED) and computed Wilson myself on the committed `D5b-touch_outcomes-*.csv` |
| 3 | Query `trader_positions`, `plans`, `bars`, `trade_excursions`, `armed_orders`, `candidate_pool`, `decision_records`, `plan_lifecycle_log`, `ab_confirm_log`, `nt8_order_snapshots` | Same | Used the committed exports (`trade_sample.csv` n=64, `E-d3-mae-mfe-per-trade.csv` n=61, `plans.jsonl` n=23, `decisions.csv` n=544, `arms.csv` n=15, `baseline.csv`, `levels.csv` n=254) |
| 4 | "Replay the last 10 sessions' plans against the tape" | No tape, no `~/nofx-analysis/` scripts | Replayed the **2 committed sessions** (09-01, 09-02) I do have: authored → armed → filled → P&L, §3.1–3.4 |
| 5 | "Count each scenario type's trigger fired vs armed vs filled vs won" | **Partially blocked.** *Authored* and *armed* I computed by condition (§3.1); *filled* by arm id (§3.3). **"Trigger fired" needs `plan_lifecycle_log`, and "won" by condition needs a `cited_scenario_id`→condition join through `plans.doc`** — `trades.csv` carries only the slot (S1–S4) | Reported P&L **by slot** (§3.1d) and stated plainly that it is a proxy for condition, not the condition itself |
| 6 | Run existing analysis scripts under `~/nofx-analysis/` | Directory absent | Wrote my own one-off computations in the scratchpad; every command shown inline |
| 7 | Grep the journal; read the NT8 logs | Absent (no journald, no NT8) | Used `2026-09-02-deepseek-e2e-audit.md` for the log-derived latency figures, cited by report:line |
| 8 | Read `plans/VL-MASTER-PLAN-v2.md`; `research/` rounds 1–9 | File does not exist; `research/` holds only `INDEX.md` | Used `INDEX.md` verdicts and the individual reports under `reports/` |
| 9 | Claim `docs/veteran-part-a-0905` via `deploy/nofx-claim.sh`; merge to dev | **Forbidden by the amendment** — the lead owns all git this session | Wrote only this file; ran no git command |
| 10 | Verify `E-d3` cohort n=30 | Cohort filter not reproducible from the committed CSV | Used my own cohort (n=36, `floor_pts` present) and flagged the discrepancy in the Evidence Basis |

**Secrets:** none quoted. I read no `.env`; `.env.example` was not opened for values. No keys, tokens or
account names appear above (`Sim101` appears in the committed `trades.csv` and is deliberately not
reproduced as an identifier claim).

**Read-only compliance:** I changed exactly one file in this tree — this report. No code, config, DB, knob,
prompt, env or unit was touched; no cancels, restarts or resets; no git command was run; scratch work
lived under the session scratchpad.
