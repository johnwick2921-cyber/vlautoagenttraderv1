# Does the forming candle predict? — touch_episodes × touch_outcomes (round 13f)

**Dispatch:** 02 — TEST 13f · owner hoang · 2026-09-11 · READ-ONLY (no code, no config, no DB write, no boot)
**Branch:** `docs/forming-candle-test` — claim commit `fdd8a3b0` (ls-remote `fdd8a3b01e9a2d145e1859a852bf8c063c148382`, pushed 10:40 CT)
**Base:** worktree cut from `origin/dev` tip `616b52a9` (marker: boot 802fb00b verified)
**Running rev verified:** `deploy/RELEASE` = `802fb00b`; `go version -m nofx-bin` → `vcs.revision=802fb00b09e51f9801e8d4fbd1bf156c86865d95`, `vcs.modified=false` [A, measured]
**Data snapshot:** live DB read-only (`file:/home/hoang/nofx/data/data.db?mode=ro`), 2026-09-11 ~18:38 UTC. Every figure below is reproduced by
`docs/superpowers/reports/2026-09-11-forming-candle-test-data/analysis.py` (output in `output.txt`; the analysis population, one row per joined pair with its ids, in `pairs_primary.csv`).

**The question:** at the moment price is AT a level and the bar has not closed, does anything we already record predict whether the level holds?

**THE HEADLINE, SAID FIRST (B2's stop-line, honest form):** the two tables cannot be joined reliably on the dispatch's key (level identity + session-day + ordinal) — measured median open-time mismatch −30 min, only 19.5% of key-matched pairs within ±10 min. A time-proximity join (±10 min within the same level price and CME session-day) does produce a verified population: **366 pairs, of which 245 unique-outcome pairs (primary analysis set)**. Everything below runs on that subset, with the population-selection caveat stated.

---

## B1 — what the fields actually hold (field corrections verified)

Writers cited at dev tip `616b52a9`; meaning read from the writer, not the name. Fields EXCLUDED from the prediction test are named and why.

| Field (touch_episodes) | Writer (file:line) | What it ACTUALLY holds [A] |
|---|---|---|
| `wick_pen_pts` | computed in `kernel/touch_telemetry.go:302` `penetrationStats`; persisted by the sink at `trader/auto_trader.go:706` → `store/touch_episode.go:52` | **Max** high/low penetration through the level over the episode's bars (approach-side aware). Not a candle wick; not the penetration at first touch. |
| `body_pen_pts` | same | **Max** close penetration through the level over the episode. Its max need not fall on the same bar as the wick max. |
| `penetration_pts` | same | `max(wick, body)` — derived; in the joined set it equals `wick_pen_pts` on every row. Tested for completeness, not as an independent predictor. |
| `vol_ratio` | `kernel/touch_telemetry.go:378` | **Sum** of episode volume ÷ mean of the last 20 pre-episode bars' volume (`TOUCH_VOL_LOOKBACK`). Grows mechanically with duration — tested raw AND normalised (`vol_ratio / bars_in`). |
| `approach_atr` | `kernel/touch_telemetry.go:410` | `|close(open bar) − close(window bars before open)| ÷ session ATR` — **dimensionless**, not points. |
| `bars_in` | accumulated in `kernel/touch_telemetry.go:187` `TouchUpdate` | Number of 1m ring bars with `OpenTime ≥ opened` — the episode's duration, not tick dwell. Episodes close on band exit or at `TOUCH_EPISODE_MAX_BARS` (12). |
| `touch_number` | `kernel/touch_telemetry.go:187` (in-memory `opened`, seeded from store since D2: `trader/auto_trader.go:721`, `store/touch_episode.go:96`) | Per-(level, session-day) ordinal. **Known skew:** before the 2026-09-03 ordinal seed it restarted at 1 on every boot; 123 duplicate (trader,symbol,label,price,day,number) keys persist in the table (e.g. PDC 29058.75, 09-01, touch 1 ×5 distinct episodes). |
| `close_1m` / `close_5m` | `kernel/touch_telemetry.go:341` / `:355` | **EXCLUDED — retired verdicts.** D4 (comment at `touch_telemetry.go:331`): reject/accept is true ≈69% of the time on IID noise by construction. Also measured: `close_1m == close_5m` on **all 1,977 rows** (surprise S3). |
| `shape` | `kernel/touch_telemetry.go:434` | **EXCLUDED** — a function of `close_1m` + `body_pen_pts`; predicting it from its own inputs is circular. |
| `opened_at_ms` / `closed_at_ms` / `created_at` | timestamps | Not predictors. |

Verdict side (`touch_outcomes`): `outcome ∈ {hold, break, ambiguous_horizon}`, k=3.0, H=12 **×1m bars**, exit_on=close on all 6,311 rows [A]. Writer: `kernel/detector_d1prime.go:120` `DetectTouchOutcomes`, recorded once per planner read by `trader/detector_record.go:25`, persisted at `store/touch_outcomes.go:235`. D1′ opens an episode on a **genuine touch** (bar range contains the level, previous bar's doesn't) and closes on close-crossing of `level ± k·Δ`; calibrated at p(hold)=0.5067 on IID-shuffled tape. Ambiguous rows are written, flagged and excluded from rates — never dropped.

**Two further B1 facts that shape everything:**
1. `touch_episodes` is **still being written** (created through 2026-09-11 18:30 UTC). The dispatch's "1,688 rows" is stale — measured **1,977** [A].
2. The legacy episode and the D1′ episode resolve on **the same bars** (median `outcome.closed − episode.closed` = −2 min for both hold and break; S6), because both close when price leaves the touch zone. The two measurement windows overlap by construction.

---

## B2 — the join

Overlap window: `touch_episodes` opened 08-26 10:07 → 09-11 13:21 CT; `touch_outcomes` opened 09-02 22:10 → 09-11 11:57 CT (recording began 2026-09-04 00:30 CT, scanning the session-day void scope). **962 episodes** opened inside the overlap.

| Join | Yield | Quality |
|---|---|---|
| **Strict** (trader, symbol, level_price, session_day, ordinal) | 431/962 (44.8%) | **UNRELIABLE** — pair open-time gaps n=528: median −30.5 min; only 19.5% within ±10 min. Ordinals drift between the two instruments (band-entry vs genuine-touch episode definitions; the pre-09-03 touch_number resets; `NextOrdinal` scopes by price only, `store/touch_outcomes.go:219`). |
| **Time-proximity** (same trader+symbol+level_price+CME-day, `|opened_at_ms diff| ≤ 10 min`, nearest) | **366 / 962 (38.0%)**, median gap 0.0 min | Verified pairing. 245 pairs have an outcome row claimed by exactly one episode — **the primary analysis set**. |

Why episodes do **not** join (both explanations are measured): **465** episodes have no outcome row for their (price, day) at all — the recorder only scans the levels seated in each read (`trader/detector_record.go:25`), level prices re-anchor (VWAP-family), and a band-only touch that never crosses the level never opens a D1′ episode. **131** have outcome rows but none within 10 min — different touches, ordinal drift across instruments.

Honest statement, per B2: **the specified join cannot be made reliably; the test proceeds on the time-verified subset only, and every cell below carries its n.**

## B3 — the population (time-verified pairs)

Verdicts: all pairs n=366 → hold 173, break 127, ambiguous_horizon 66. Primary (unique-outcome) n=245 → hold 130, break 80, ambiguous 35.

Cells that clear n≥30 on both sides are marked OK; everything else gets **no verdict** (D-stop-line).

| Cell | n | hold / break / amb | OK? |
|---|---|---|---|
| entry_side = above | 180 | 88 / 64 / 28 | OK |
| entry_side = below | 186 | 85 / 63 / 38 | OK |
| family SWG | 97 | 33 / 43 / 21 | OK |
| family PD | 63 | 30 / 23 / 10 | n<30 |
| family ON | 42 | 33 / 3 / 6 | n<30 |
| family OR | 59 | 27 / 18 / 14 | n<30 |
| family RTH | 41 | 20 / 12 / 9 | n<30 |
| family EQH/EQL | 29 | 13 / 13 / 3 | n<30 |
| family OB/FVG/SD | 17 | 8 / 6 / 3 | n<30 |
| family POC/nPOC | 5 | 2 / 3 / 0 | n<30 |
| family VWAP-family | 2 | 1 / 1 / 0 | n<30 |
| tf = daily/line | 212 | 113 / 60 / 39 | OK |
| tf = 5m | 76 | 29 / 35 / 12 | n<30 |
| tf = 1h / 15m / 4h | 37 / 21 / 20 | — | n<30 |
| session ASIA | 166 | 65 / 56 / 45 | OK |
| session NY | 132 | 79 / 41 / 12 | OK |
| session LONDON | 68 | 29 / 30 / 9 | n<30 |
| validity valid | 23 | 10 / 10 / 3 | n<30 |
| validity no_formation | 279 | 132 / 97 / 50 | OK |
| validity legacy | 64 | 31 / 20 / 13 | n<30 |

(A22: no P&L appears anywhere in this test, so `pnl_corrected` does not arise.) Sample ids: episode ids 1,043–1,968 and outcome ids 3–6,303 for the 245 primary pairs — the full id-level list is `pairs_primary.csv`.

## B4 — the null / base rate: NOT REPRODUCED as pinned

The dispatch pins "levels hold 48.8% of first touches, n=423, Wilson [44.1, 53.6]" as our base. **Reproduction attempts (A17):**

| Query on today's table | n | p(hold) | Wilson |
|---|---|---|---|
| pinned figure 48.8% / n=423 | — | — | **NOT REPRODUCED** |
| the 677-row 09-05 corpus, first-touch rows | 377 | 0.4695 | [42.0, 52.0] |
| the 677-row 09-05 corpus, all rows | 526 | 0.4981 | [45.6, 54.1] |
| ordinal=1, all | 892 | 0.4809 | [44.8, 51.4] |
| ordinal=1, excluding duplicates | 663 | 0.5098 | [47.2, 54.8] |
| whole table (all ordinals) | 5,087 | 0.5194 | [50.6, 53.3] |

The **n=423 does reproduce** — as the price-time **key count** of the original 677-row corpus (`touch_outcomes.created_at ≤ 2026-09-05 15:00`): 677 rows / 423 keys [A]. The 48.8% rate itself does not: vet-02 already flagged that era's figures as artifacts of a contaminated recorder ("every number anyone has quoted from it (48.8 % first-touch hold …) is an artifact of that contamination", `2026-09-05-vet-02-levels.md:30`). Separations below are therefore reported **against measured rates, not the pinned 48.8%**, and every rate carries its n and interval. The joined population's own hold rate: primary 130/210 = **61.9% [55.2, 68.2]**; all pairs 173/300 = 57.7% [52.0, 63.1] — the join selects a special subset (levels that were seated, band-touched and genuinely touched; see S5).

## B5 — field by field: hold vs break (primary n=210: hold 130, break 80)

Rank test = two-sided Mann-Whitney U. Multiple comparisons corrected two ways: Bonferroni ×7 and a Westfall-Young max-T over all 7 fields (20,000 permutations of the verdict labels — the Reality-Check-style correction the dispatch names; round 16's lesson applied to ourselves).

| Field | med(hold) | med(break) | U | p raw | p WY-max-T | p Bonf |
|---|---|---|---|---|---|---|
| `wick_pen_pts` | 5.50 | 16.25 | 3603.5 | **0.0002** | **0.0006** | 0.0012 |
| `penetration_pts` | 5.50 | 16.25 | 3603.5 | **0.0002** | **0.0006** | 0.0012 |
| `body_pen_pts` | 0.00 | 6.50 | 4527.0 | 0.0907 | 0.4478 | 0.6346 |
| `vol_ratio` | 5.16 | 8.11 | 4160.0 | 0.0151 | 0.0697 | 0.1054 |
| `bars_in` | 5.00 | 5.00 | 4620.0 | 0.1702 | 0.6106 | 1.0000 |
| `approach_atr` | 1.50 | 1.67 | 4918.0 | 0.5103 | 0.9719 | 1.0000 |
| `touch_number` | 2.00 | 2.00 | 5287.5 | 0.8312 | 1.0000 | 1.0000 |

Median-split of `wick_pen_pts` (median 10.75): low-wick half holds 77/105 = 73.3% [64.2, 80.9]; high-wick half holds 53/105 = 50.5% [41.1, 59.9]. Quartile gradient: **84.9% [72.9, 92.1] (n=53, wick ≤1pt) → 61.5% (n=52) → 53.8% (n=52) → 47.2% [34.4, 60.3] (n=53, wick >25.1pt)**. Stratified by approach side, wick separation holds raw on both: from-below p=0.0044 (n=106), from-above p=0.0164 (n=104) [B].

**Sensitivity (all 366 pairs, n=300):** wick raw p=0.0262 → WY **0.1234** (does not survive); vol_ratio WY 0.279; nothing else. The correction's verdict therefore depends on the duplicate-handling choice — reported, not hidden.

**But — is the wick separation a predictor or a restatement?** S6: both instruments close their episodes on the same bars (median gap −2 min for hold and break alike), and the break-side wick median (16.25 pts) sits just above the recorded barrier band (median `k·Δ` = 13.48 pts). `wick_pen_pts` is the **max penetration over a window that overlaps the outcome window**: a break requires price to cross ≈13.5 pts, and that crossing bar's penetration IS the recorded maximum. The separation is substantially mechanical, and the field is **not fixed at the decision instant** — it is only written when the episode closes. It cannot answer the forming-candle question as recorded.

## B6 — the duration confound

`vol_ratio` is a **sum** ÷ pre-mean: Spearman(vol_ratio, bars_in) = 0.481 (p=1.5e-13, n=210) [A]. Raw: medH 5.16 vs medB 8.11, p=0.0151. Normalised (`vol_ratio / bars_in`): medH 1.138 vs medB 1.225, **p=0.2096** — the effect collapses to noise once duration is divided out; the sign does not flip but the magnitude is gone. **`vol_ratio` only separates through duration → not a predictor.**

## B7 — the horizon

Detector's own horizon = 12 × **1m** bars (recorded). 10/20 **five-minute-bar** horizons recomputed from the persisted 5m bars for the same 245 primary pairs, same D1′ barrier rule (`level ± k·Δ`, exit_on=close), bars after the touch:

| Horizon | resolved n | hold rate | wick medH/medB (p) | vol_ratio medH/medB (p) |
|---|---|---|---|---|
| H=12×1m (recorded) | 210 | **61.9%** [55.2, 68.2] | 5.50 / 16.25 (0.0002) | 5.16 / 8.11 (0.0151) |
| 10×5m (recomputed) | 208 (2 censored) | **53.4%** | 5.50 / 16.25 (0.0004) | 4.94 / 8.77 (<1e-4) |
| 20×5m (recomputed) | 210 (0 censored) | **53.8%** | 5.50 / 16.25 (0.0004) | 4.94 / 8.77 (<1e-4) |

The detector's H=12 "hold" advantage (61.9% vs the 48–52% measured base rates) **erodes with horizon**: by 10 five-minute bars the joined population holds ~53%, statistically indistinguishable from the whole-table rate's neighbourhood. The field separations that grow at the 5m horizons (vol_ratio p<1e-4, vol_ratio/bars_in p=0.0006, bars_in p=0.007, body p=0.003) are the same overlapping-window artifact amplified by the longer episode — the episode's recorded sums live inside the window the 5m verdict walks. No field shows a separation that is (a) corrected-significant, (b) frozen before the outcome, and (c) stable across horizons.

## B8 — verdict per field

| Field | Verdict | The number |
|---|---|---|
| `wick_pen_pts` | **PREDICTS — mechanically, not decision-instant** | separates hold/break: med 5.50 vs 16.25 pts, U=3603.5, p=0.0002, WY-corrected p=0.0006, n=210; quartile gradient 84.9% → 47.2% hold. BUT the maximum is measured over the same window as the verdict and is only written at episode close — it is a restatement of what already happened, not a predictor available while the bar is unclosed. Fails to survive correction on the wider set (p_WY=0.123). |
| `penetration_pts` | DOES NOT PREDICT independently | equals `wick_pen_pts` on every joined row (wick ≥ body throughout). |
| `body_pen_pts` | DOES NOT PREDICT | p=0.09 raw; 0.45 corrected, n=210. |
| `vol_ratio` | DOES NOT PREDICT | raw p=0.015 but duration-carried: normalised p=0.21 (B6); WY 0.07. |
| `approach_atr` | DOES NOT PREDICT | p=0.51, n=210. |
| `bars_in` | DOES NOT PREDICT (at H=12) | p=0.17; its 5m-horizon "separation" is the window overlap. |
| `touch_number` | DOES NOT PREDICT | p=0.83; also carries the known ordinal-reset skew. |
| `close_1m`/`close_5m`/`shape` | **EXCLUDED (B1)** | retired verdicts; identical on all rows. |
| `opportunity_outcome` (W1, adae3bb4) | **UNTESTABLE** | populated on 6,149/6,311 outcome rows, but every populated value is `reached_declined` (162 NULL) — no variance to separate anything. |

**The one sentence.** Nothing recorded in `touch_episodes` predicts hold-vs-break at the moment price is at a level with the bar unclosed; the only separation that survives a Reality-Check-style correction is the episode-maximum penetration, which is measured over the same window as the verdict it "predicts" — so the honest answer today is: **no, nothing here predicts; the strongest pattern is the verdict restated, not a forecast.**

## What would have to be recorded (if this is to be answerable)

1. **Decision-instant partial states**, frozen at predeclared landmarks (30/60/120 s after first touch): penetration-to-date at that instant, the forming bar's true wick/body **in points**, per-bar signed volume (not a sum), and an ATR **in points** alongside the dimensionless approach measure. The current row is written only at episode close — by construction it can never answer a forming-candle question.
2. **A stable join key.** Persist approach direction (currently unexported at `kernel/touch_telemetry.go:119`), give line levels a formation time (`validity = unverified:no_formation` covers 5,380/6,311 outcome rows today), and record a level's stable identity so the two instruments can be joined without time-tolerance guesswork.
3. **n per cell for a 5-point effect** (two-proportion, p0=0.488 vs p1=0.538, 80% power, two-sided): **1,568 per group at α=0.05; 2,491 per group at Bonferroni α/7**. Today's largest joined cell is hold 130 / break 80 — an order of magnitude short of even the uncorrected requirement, and most cells are n<30 (B3). At the current write rate (~1,000 episodes and ~6,300 outcomes in two weeks, but only ~366 time-joinable pairs), reaching 1,568+ per cell for even one field requires a recording redesign, not just patience.

## Surprises (A23 — recorded, not acted on)

- **S1** `touch_episodes` is still being written (max `created_at` = 2026-09-11 18:30 UTC) despite the "retired instrument" labels in the 09-04/09-05 reports; the dispatch's 1,688 is stale — 1,977 today and growing.
- **S2** Every `touch_outcomes` row has `candidate_seated = 1` — the recorder only scans seated levels, so the off-policy "candidate" side of the selection question is unrecorded there.
- **S3** `close_1m == close_5m` on all 1,977 `touch_episodes` rows; the two columns add zero independent information.
- **S4** The touch_number skew survived into September rows (123 duplicate keys; e.g. five distinct 09-01 PDC 29058.75 episodes all recorded touch 1).
- **S5** The joined subset holds at 61.9% vs the whole-table 51.9% — the join is a selected subpopulation (seated + band-touched + genuinely touched levels); no rate from it generalises to "all touches".

## Appendix — cited files, `git log -1` (dev tip `616b52a9`)

| File | Last change |
|---|---|
| `kernel/touch_telemetry.go` | `829a6176` 2026-09-03 18:49 — merge fix/live-detector-1b |
| `kernel/detector_d1prime.go` | `f00c8b87` 2026-09-03 18:43 — feat(1B D5/D7) |
| `store/touch_episode.go` | `96b63221` 2026-09-03 18:16 — feat(data-integrity) D2 ordinal seed |
| `store/touch_outcomes.go` | `69164e1e` 2026-09-11 00:59 — one-setup: store side |
| `trader/detector_record.go` | `b3bd88c7` 2026-09-11 01:12 — one-setup: the seam |
| `trader/auto_trader.go` | `b3bd88c7` 2026-09-11 01:12 — one-setup: the seam |
| `kernel/cme_calendar.go` | `5457ac5a` 2026-09-07 19:39 — fix(session-calendar) |

Queries and outputs: `2026-09-11-forming-candle-test-data/{analysis.py, output.txt, pairs_primary.csv}`.
