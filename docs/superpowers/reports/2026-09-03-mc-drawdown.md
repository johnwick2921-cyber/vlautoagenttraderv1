# 1E — MONTE CARLO DRAWDOWN RIG (offline, read-only, pre-registered)

**Status at this commit: PRE-REGISTRATION ONLY. No analysis has been run.**
Committed 2026-09-03 before any computation, per the dispatch. Results are
appended in a later commit; this header is not edited afterwards.

Scripts: `~/nofx-analysis/mc-drawdown/` (stdlib + numpy, seeded, re-runnable).
Engine: untouched. DB opened `mode=ro`. No knob writes, no lock, no live calls.

---

## PREMISES — VERIFIED BEFORE PRE-REGISTERING

| # | claim | measured | verdict |
|---|---|---|---|
| P1 | ~60–70 usable closed trades since 2026-08-15 | **n = 64** | **holds** |
| P2 | max_daily_trades=3, daily_loss_limit=$450, profit target $900, master OFF | all present; **and every guardrail is individually disabled too** | **holds, with an addition** |
| P3 | size clamped to 1 contract, $2/pt MNQ | confirmed exactly on 4 rows | **holds** |
| P4 | stops "30–40 pre-0B, ~20–80 post" | n=34 arms: median **26.75**, p25 21.96, p75 40.00, **max 150.00**, mean 37.75 | **broadly holds, fatter right tail** |

**P1 sample construction** (`trader_positions`, `mode=ro`):
`status != 'OPEN'` AND `created_at >= 2026-08-15` AND `pnl_corrected IS NOT NULL`
AND `source IN ('system','armed_entry','reconcile')` → **64 rows**.
70 closed in the window; **3 UNRESOLVED** (`pnl_corrected IS NULL`) excluded and
counted; **3 test-seam** (`source='e7_farside_test'`) excluded. Raw
`realized_pnl` is never read (A22).

**P2 resolved from the BOUND strategy** (`traders.strategy_id` → `a5b7662e…`,
name "MNQ"), never `LIMIT 1` — checklist class 9:

```
guardrails_enabled        = False      daily_loss_limit_usd      = 450
daily_loss_enabled        = False      daily_profit_target_usd   = 900
daily_profit_enabled      = False      max_daily_trades          = 3
max_daily_trades_enabled  = False      max_contracts_per_order   = 2
max_contracts_enabled     = False
day_plan.sessions max_trades = 10 / 7 / 10
```

> **Surprise, recorded before analysis:** the guardrails are not merely
> master-off — **each one is individually disabled as well**. Q3 is therefore
> entirely counterfactual: it asks what these limits WOULD have done, not what
> they did. Nothing in this report describes a control that is currently
> protecting the account. Also note `max_contracts_per_order = 2` in config
> while 0B's `ClampStageAContracts` caps at 1 — the clamp governs, so size is 1.

**P3** verified on ids 587–590: e.g. **590** entry 29193.25 → exit 29143.75 LONG
= −49.50 pts × $2 = **−$99.00** at qty 1. Point value $2/pt confirmed.

---

## QUESTIONS (pre-registered verbatim)

**Q1** Given the realized per-trade P&L distribution (`pnl_corrected`, n=64),
what is the distribution of MAX DRAWDOWN over the next 20 / 50 / 100 trades?
(p50, p90, p95, p99, worst)

**Q2** What is the probability of a losing streak of k = 4, 6, 8, 10 consecutive
trades within 50 trades, given the realized win rate?

**Q3** With the daily loss limit at $450 and 1 contract: on what fraction of
simulated days does the limit trip, and how much of the realized expectancy does
it forfeit (trades lost after a trip that would have been winners)? Same for
`max_daily_trades = 3`.

**Q4** Expectancy per trade with a 95% CI, and the n required to distinguish it
from zero at power 0.8 (formula stated).

**Q5** How do Q1–Q3 change if the per-trade distribution is drawn from the
pre-0B era vs post-0B (stops widened, BE/trail off)? Descriptive — post-0B n is
tiny; say so.

## METHOD (pre-registered)

- **M1** Trade-sequence bootstrap: IID resample of realized `pnl_corrected`,
  **B = 10,000** paths per horizon, AND a stationary block bootstrap
  (**mean block 5**) to preserve streakiness. Both reported; if they disagree
  materially, say which and why.
- **M2** Day simulation for Q3: group realized trades by session-day, resample
  **days** (not trades) to preserve within-day clustering, apply the daily limit
  / trade cap in sequence order, count trips, and compute forfeited P&L as the
  sum of post-trip trades' realized `pnl_corrected` on that day.
- **M3** Q2 streaks: exact via the realized p(win) and n, **plus** the bootstrap
  empirical count; both reported.
- **M4** Q4: mean, sd, t; `n_required = ((z_α + z_β)·sd/μ)²`, α = 0.05
  two-sided, power 0.8. If μ ≈ 0, report "n required → ∞ at the current
  expectancy" honestly.
- **M5** Multiplicity: none applied — every figure here is descriptive, and this
  is stated rather than silently assumed.
- **M6** Sensitivity: re-run Q1 with the single largest loss and the single
  largest win removed. **If the picture flips, the sample is too thin to carry a
  verdict — say so.**

## PASS RULES (pre-registered)

- A figure ships only with its **n**, its interval, and the **row ids** behind
  it (A21). No rate without n (A24).
- Ambiguity and exclusions are **counted and shown**, never dropped.
- **M6 flip ⇒ the report's headline becomes "sample too thin", regardless of how
  clean the central estimates look.**
- This wave issues **no verdict** on size, limits or exits. It is the INPUT to
  those rulings (dispatch stop-line).

---

*Results appended below in a later commit. Nothing above is edited after this
commit — the git history is the pre-registration's proof.*

---
---

# RESULTS (appended 2026-09-03; the header above is unedited)

Re-run: `cd ~/nofx-analysis/mc-drawdown && python3 mc_drawdown.py`
Seed 20260903 · B = 10,000 · CSVs in `2026-09-03-mc-drawdown-data/`.

## The sample

**n = 64**, ids **521…590**, session-days 2026-08-18 → 2026-09-01 (11 days).

```
sum = -423.93    mean = -6.624    sd = 100.589
wins = 21   losses = 41   flat = 2    p(win) = 0.3387  (flats excluded)
min = -155.00 (id 589)   max = +311.00 (id 532)
```

## Q1 — max drawdown ($, 1 contract, $2/pt)

| horizon | method | p50 | p90 | p95 | p99 | worst |
|---|---|---|---|---|---|---|
| 20 | IID | 478 | 828 | 935 | 1,138 | 1,499 |
| 20 | block(5) | 471 | 809 | 920 | 1,110 | 1,608 |
| 50 | IID | 866 | 1,477 | 1,677 | 2,065 | 3,030 |
| 50 | block(5) | 847 | 1,436 | 1,613 | 1,981 | 2,595 |
| 100 | IID | 1,364 | 2,298 | 2,589 | 3,160 | 4,130 |
| 100 | block(5) | 1,321 | 2,216 | 2,485 | 3,016 | 4,201 |

**The two bootstraps agree throughout** (block within ~3% of IID at every
quantile), so there is no detectable streakiness beyond what IID resampling
already produces. Reported as pre-registered; no preference needed.

## Q2 — P(losing streak ≥ k within 50 trades), at p(win)=0.3387

| k | exact (recursion) | bootstrap |
|---|---|---|
| 4 | 0.9927 | 0.9865 |
| 6 | 0.8079 | 0.7446 |
| 8 | 0.4620 | 0.3963 |
| 10 | 0.2176 | 0.1741 |

The exact figure runs slightly high because it treats the 2 flat trades as
non-losses in p(win) but the bootstrap can draw them as run-breakers; the gap is
the size of that handling choice, not a disagreement about the tape.

## Q3 — guardrail counterfactual (day resample, B=10,000)

Realized: **11 session-days**, trades/day min 3, median 4, max 12.

| rule | trips on | P&L kept/day | forfeited/day | net effect/day |
|---|---|---|---|---|
| daily_loss $450 | **9.1%** of days | −36.21 | +0.00 | **+0.00** |
| max_daily_trades 3 | **81.8%** of days | −65.87 | +24.54 | **−24.54** |
| both | 82.0% of days | −66.81 | +26.39 | **−26.39** |

**The $450 loss limit forfeits nothing** — it trips on one day in eleven, and on
that day the trip landed on the day's *last* trade, so nothing followed it.
**The 3-trade cap trips on 4 days in 5 and forfeits +$24.54/day of realized
P&L**: with a median of 4 trades per day, it mostly cuts the day short, and the
trades it removes were net positive over this sample.

## Q4 — expectancy

```
mean = -6.624   sd = 100.589   se = 12.574
95% CI [-31.268, +18.020]      t = -0.527
n_required = ((z_a + z_b)·sd/mu)^2 = ((1.960+0.842)·100.59/6.624)^2 = 1,810 trades
```

**Expectancy is not distinguishable from zero.** The interval spans −$31 to +$18
per trade. At the current effect size, separating it from zero at power 0.8 needs
**~1,810 trades** — roughly 28× the sample in hand, and at ~6 trades/day about a
year of trading.

## Q5 — era split (0B cutover 2026-09-02 07:49 CT, by timestamp)

| era | n | mean | sum | maxDD@20 p50 | p95 |
|---|---|---|---|---|---|
| pre-0B | 62 | −2.741 | −169.93 | 430 | 872 |
| post-0B | **2** (ids 589, 590: −155, −99) | — | — | — | — |

**Post-0B is two trades, both losses.** No distribution claim is possible and
none is made. Note the pre-0B mean (−2.741) is *better* than the full-sample mean
(−6.624) precisely because those two post-0B losses are in the full sample.

## M6 — sensitivity (drop the largest win AND the largest loss)

| | n | mean | maxDD@20 p50 / p95 | maxDD@50 p50 / p95 |
|---|---|---|---|---|
| full | 64 | −6.624 | 476 / 944 | 872 / 1,677 |
| trimmed | 62 | −9.354 | 472 / 930 | 898 / 1,664 |

Dropped id **532** (+311) and id **589** (−155). **The picture does not flip**:
drawdowns move by under 3% and the mean gets slightly *worse*. The drawdown
result is not an artifact of one or two extreme trades.

---

## What a normal bad week looks like, at n=64

**[A]** At roughly 6 trades/day and 5 days, a week is ~30 trades. Interpolating
the Q1 table, a **median** week's worst drawdown is around **$600**, and a
**1-in-20 bad week reaches roughly $1,200**. A run of **4 consecutive losers is
essentially certain** inside 50 trades (p=0.99), **6 in a row is the coin-flip
case** (p=0.81 exact / 0.74 bootstrap), and **8 in a row happens about 4 times in
10**. None of that is breakage. It is what a 34%-win-rate distribution with
sd≈$100 does.

**[A]** The distinguishing signal is not drawdown depth, it is **drawdown without
those statistics**: a $1,200 week is normal; a $1,200 week where trades stop
arriving, or where losses cluster at one condition or one session, is not.

**[B]** On the guardrails, at the realized distribution: the **$450 daily loss
limit is close to inert** — it trips on 9% of days and forfeited nothing in this
sample. The **3-trade cap is the expensive one**: it trips on 82% of days and
gives up ~$24.54/day of realized P&L, because the median day is 4 trades and the
trades it cuts were net positive here. **[A]** Both are currently **disabled** —
master off *and* each individually off — so neither is protecting anything today;
this is a counterfactual about what turning them on would do.

**[C]** The honest headline: **with expectancy statistically indistinguishable
from zero (CI −$31 to +$18) and ~1,810 trades needed to resolve it**, no
drawdown number here should be read as "the system loses $X". It says only how
wide the noise is at this sample size. That is the input this wave was asked for,
and per the stop-line it issues no verdict on size, limits or exits.

## Corrections made during the run, before publishing

- **Q5's era cut was wrong on the first pass.** I split on session-day, but the
  CME day rolls at 17:00 CT, so 0B's 07:49 cutover sits *inside* session-day
  2026-09-01 — every row landed pre-0B and post-0B read n=0. Re-cut on the
  timestamp, which gives the true n=2.
- **Q3's zero forfeiture is real, not a bug**: the one tripping day tripped on
  its final trade, so nothing followed. The script now says so explicitly rather
  than printing a bare +0.00.
