# 2026-09-02 — LIVE intraday bias replay: the plan-stamping bias reconstructed and tested out-of-sample

**Dispatch:** the live intraday bias — "4H-dominant hierarchy (kernel bias tree + regime)" — gets the identical test as the bias calibration (2deab3c8, P7/D6). READ-ONLY on the engine · no lock · no live calls.
**Tree:** worktree `~/nofx-biastest`, branch `docs/live-bias-replay-0902`, base `bb8b5419` (dev HEAD at start). Scripts: `~/nofx-analysis/live-bias-replay/`. Bars source: the `bars` table (SQLite, read-only). CSVs: `docs/superpowers/reports/2026-09-02-live-bias-replay-csvs/`.

---

## PRE-REGISTRATION HEADER (fixed 2026-09-02 ~21:10 CT, BEFORE any computation)

### P0 — What "the live intraday bias" IS (quoted, no inference)

The plan's `Bias.Direction` is AI-authored; its MACHINE inputs are exactly two
blocks the planner prompt renders every session read:
1. **BIAS-TREE** — `kernel/planner_prompt.go:129 RenderBiasTree`: branch 1
   `close > PDH → bull (HIGH)`; branch 2 sweep of PDH/PDL + close back inside →
   bear/bull (MEDIUM); branch 3 inside the day → `close vs PDC → long/short
   (LOW)`; branch 4 outside-prior-range-but-inside → neutral; branch 5
   premium/discount disallows the premium side long / discount side short.
2. **REGIME** — `kernel/regime.go:50 ComputeRegime`: `trend_daily` = price vs
   EMA200 on daily bars, `trend_1h` = price vs EMA50 on 1h bars (up/down/flat,
   ±0.05% dead-band, `regimeTrendEps`).

**"4H-dominant hierarchy" note:** no literal 4h trend leg exists in the planner's
machine sections (grep-verified; the HTF-level pass uses 1h+4h but feeds the
level ladder, not the bias stamp). The reconstruction therefore covers exactly
the tree + regime legs, in the hierarchy the prompt presents them. Stated, not
hidden.

### P1 — Data inventory (measured, not looked-at-for-results)

`bars` table, read-only: 1d MNQ 2019-05-02 → 2026-09-01 (1,896) · 1h MNQ
2026-05-04 → 2026-09-02 (2,001) · 4h MNQ back to ~2025-05 (not needed by the
legs above; listed for completeness). The binding range for session-level
open/close targets is the 1h table.

### P2 — The reconstructed call (definitions fixed, no peeking after this header)

Unit = one session-plan instance = (session-day, session), sessions ASIA / LONDON
/ NY, over completed session days with 1h coverage (2026-05-05 … 2026-09-02).
Call evaluated at the session's READ time (engine registry `ReadCT`, quoted:
ASIA 16:30 · LONDON 01:30 · NY 08:00 CT) using ONLY bars closed before it:
- **price** = close of the last 1h bar with `open_time` < read time.
- **PDH / PDL / PDC** = prior session-day's 1d H / L / C (1d table).
- **tree call**: branch 1 → long; branch 1-mirror (close < PDL) → short;
  branch 3 inside → long iff price > PDC else short; branch 4 → neutral.
  Branch 2 (sweep + close back inside) and branch 5 (premium/discount veto)
  applied post-hoc as stated below — branch 5 veto: inside-premium (≥50% of
  [PDL,PDH]) disallows long, inside-discount disallows short → neutral.
- **regime call**: `up` iff price > EMA200_daily × 1.0005 AND price >
  EMA50_1h × 1.0005; `down` iff both < ×0.9995; else neutral. EMA50_1h requires
  50 prior 1h closes — the first days are `warming` and the leg is n/a → regime
  neutral (counted).
- **composite (the reconstructed stamp)**: tree call if long/short; else regime
  call; else neutral. Neutral rows are counted separately (called-only rows also
  reported).
- Legs reported individually too (tree-only, regime-only) for the D6 table.

**Target:** sign of the session's own window open→close from 1h bars:
ASIA 17:00→02:00 · LONDON 02:00→08:30 · NY 08:30→14:45 CT. `open` = close of the
last 1h bar with open_time < window start; `close` = close of the last 1h bar
with open_time < window end. **Stated intrabar error:** 1h stamps sit on the
hour; the NY "open" is the 09:00 print (includes 08:30–09:00) and the NY "close"
is the 14:00 print (excludes 14:00–14:45). Same error class as the bias
calibration's S2/S3.

### P3 — Split (fixed before looking)

Session days ordered; **exploration = first 60% (2026-05-05 … <cutoff), holdout
= last 40% (cutoff … 2026-09-02)** — the exact cutoff date computed from the
actual completed-day list and stated in the results. Verdicts from holdout only.

### P4 — Statistics (identical to the bias calibration P4/D2)

1. Hit rate `ĥ = #(call sign == realized sign)/n` on CALLED rows, 95% Wilson.
2. Mean session-window return (points) conditional on call sign, Student t vs 0.
3. Position series: +1/−1/0 per call, PnL in points, max drawdown.
4. **Net of friction:** 1 pt per side charged on |Δpos| at each re-position;
   net t-stat.
5. Neutral share quoted per leg.

### P5 — Multiplicity

Benjamini–Hochberg across the tested calls (tree-only, regime-only, composite,
× ASIA/LONDON/NY strata) on holdout hit-rate one-sided p-values.

### P6 — Strata (descriptive)

Per session (ASIA/LONDON/NY) and per call-vs-realized contingency. Tables only.

### P7 — PASS/FAIL (the SAME D6 rule as 2deab3c8, fixed before looking)

The reconstructed bias is **USABLE** iff on HOLD OUT: (a) called-rows hit-rate
Wilson **lower bound > 0.50**, AND (b) net-of-friction mean session-window
return **t > 2**. Otherwise **NOT USABLE at this n**.

### P8 — The four-trade check (descriptive)

Quote what the reconstruction stamps for the four session-plans behind today's
losers (ASIA 09-02, LONDON 09-02, NY 09-02) and whether it matches the plan
`Bias.Direction` the store records.

**Nothing below this line was known when the criteria above were written.**

---

# RESULTS (computed after the pre-registration above)

## R1 — Scope

84 completed session days (2026-05-05 … 2026-09-02), 252 session-plan rows.
Exploration 50d (05-05 … 07-16) · holdout 34d (07-17 … 09-02). Neutral (uncalled) rows counted per leg.

## R2 — Per leg, holdout (verdicts from holdout only)

| leg | called n | neutral | p(hold direction) | Wilson 95% | net-of-friction t | gross PnL (pt) | maxDD |
|---|---|---|---|---|---|---|---|
| BIAS-TREE | 21 | 81 | 0.4762 | [0.2834, 0.6763] | +0.70 | +378.0 | 271.0 |
| REGIME (D EMA200 + 1h EMA50) | 46 | 56 | 0.5435 | [0.4018, 0.6785] | +0.96 | +1140.5 | 647.8 |
| COMPOSITE | 62 | 40 | 0.5000 | [0.3792, 0.6208] | +0.92 | +1186.0 | 764.8 |

**D6 (identical rule):** no leg clears (a) hit-rate Wilson lower bound > 0.50 on holdout — the best lower
bound is regime's 0.4018 — and no leg clears (b) net t > 2 (best +0.96). **Every leg: NOT USABLE at this n.**

## R3 — Sessions (composite, holdout)

| session | called n | neutral | p | Wilson | net t |
|---|---|---|---|---|---|
| ASIA | 20 | 14 | 0.6000 | [0.3866, 0.7812] | +1.72 |
| LONDON | 20 | 14 | 0.4500 | [0.2582, 0.6579] | +0.21 |
| NY | 22 | 12 | 0.4545 | [0.2692, 0.6534] | −0.11 |

Descriptive only; every interval covers 0.50.

## R4 — Exploration consistency check

Composite exploration: p = 0.4583 [0.3622, 0.5577], n=96, net t = **−1.35** (negative). The holdout's positive net t (+0.92) is not confirmed out-of-sample —
the sign flips between periods. No BH rejection (all adjusted p far above 0.05).

## R5 — The four-trade check (the dispatch's premise, measured)

Reconstructed calls for session-day 2026-09-02 (read-time price vs PDH 29212.50 / PDL 28927.25 / PDC 29143.00):

| session | read price | tree branch | regime | composite |
|---|---|---|---|---|
| ASIA | 29139.00 | branch 3: close < PDC → **SHORT** | neutral | **SHORT** |
| LONDON | 29076.75 | branch 3: close < PDC → **SHORT** | neutral | **SHORT** |
| NY | 29040.00 | branch 3 short VETOED by branch 5 (discount) → neutral | neutral | **NEUTRAL** |

**The machine bias did NOT say long on today's losers — it said SHORT (ASIA, LONDON) and NEUTRAL (NY).**
The plans' `Bias.Direction=long` was the AI overriding its own machine tree (branch 3 was short by PDC
29143 vs price, and branch 5's discount veto forbade shorts at NY — the AI went long against both). The
premise 'this signal said long on all four losers' is therefore not supported by the reconstruction;
the LONG stamp came from the model, not the machine blocks.

## R6 — Verdict (one paragraph)

The reconstructed intraday bias — BIAS-TREE + REGIME, the machine sections that inform every session
plan's long/short stamp — is **NOT USABLE** by the identical D6 rule that judged the weekly control:
holdout hit rates 0.48–0.54 with lower bounds ≤ 0.40 (n = 21–62 called rows), net-of-friction t ≤ 0.96,
and the exploration period runs NEGATIVE (t −1.35). Like the weekly one, the plan bias is a label, not a
direction — and today's four longs were not the machine's call at all. The machine blocks remain useful
as FACTS for the AI to reason from (they are rendered as facts, not orders); nothing changes, nothing is
promoted to a gate. n is tiny (84 session days of 1h); the honest lever is the same as every other tape
inventory this week: time.

## R7 — CSVs + scripts

`exports/2026-09-02-live-bias-replay-csvs/calls.csv` — all 252 session-plan rows
(day, session, period, price, PDH/PDL/PDC, EMA200/EMA50, tree/regime/composite, open/close, realized sign,
window return). Script: `~/nofx-analysis/live-bias-replay/replay.py` (stdlib only, read-only `bars` table).
