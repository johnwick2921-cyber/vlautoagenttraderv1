# Round 24 — IS A HIGHER-TIMEFRAME ZONE AN ENTRY? (lane 93 / nofx-dd)

Branch `docs/round-24-htf-zone-entry` (claim `ea76c250`, scripts `4c3c54b4`). Dispatch: CTO
nofx-4c 2026-09-17 08:31 CT (owner question 02:10 CT). Cells only; no recommendation beyond
the cells. Evidence tiers: every number below is **[A]** (computed by the named script at the
named commit on the sha-pinned inputs) unless marked otherwise. Anything not in the MANIFEST is
NOT MEASURED. Research dir: `docs/superpowers/research/2026-09-17-round-24-htf-zone-entry/` (scripts, harness
copy, `run_harness.sh`, `manifest.py`; outputs in `out-r24/`, gitignored, on `~/nofx-93r24`).

## 0. The finding that comes before any cell — the HTF zone universe is DENSE

| fact | value |
|---|---|
| HTF (1h+4h+1d) zones in a read's raw universe | **mean 83.6 / read, median 79, p90 146** (3,065 of 3,093 reads have ≥1) |
| of which 1h | 75.2 / read (OB 29.3, EQH 13.0, EQL 12.5, IFVG 6.9, SUPPLY 5.5, DEMAND 5.0, FVG 3.0) |
| of which 4h · 1d | 5.2 · 3.3 per read **but present in only 221 · 223 of 3,093 reads — all 2026** (the read's contract label has no 4h/1d series before 2026: S4c's coverage shape) |
| 15m (not "HTF" for this round) | 96.2 / read |
| by year (HTF/read) | 2022 72.9 · 2023 77.7 · 2024 74.3 · 2025 77.0 · **2026 124.0** (4h+1d appear) |
| ordinal-1 episodes at an HTF zone (inside or within 1×Δ) | **675,320 of 845,076 = 80%** (inside 557,193 · near 118,127); not at any zone 167,384 |
| … sitting inside overlapping zones of BOTH polarities | **356,220 = 42%** of at-zone episodes |
| … whose own level IS an HTF zone (the D1′ at the zone midpoint) | 24,114 (`own` variant) |

Consequence: "price is at an HTF zone" is not a filter — four of five level touches already
qualify — and under a closest-zone rule "with-zone / against-zone" is decided by geometry in
the 42% of cases where zones of both polarities overlap the price. That is why this report
carries three variants of every Q1/Q2/Q5 cell and why the **`own` variant is PRIMARY** (CTO
ruling 08:56 CT): the episode's own level is the HTF zone, the D1′ is anchored at its midpoint,
and its polarity is its own. `all` and `noconflict` are context.

## 1. Definitions (exact; shared by every script through `r24lib.DEFINITIONS`)

- **Population.** The Round 23 S4 pass re-evaluated with the SAME kernel base (`git archive
  d15db077`), SAME DB copy (`nofx-r101/data/db.copy.db`, sha256 998a15ee…), SAME 3,093 reads and
  the SAME D1′ instrument — so the level universe is identical by construction (4,471,482 episodes,
  Round 23's count exactly). The writer adds `price/lo/hi/label/polarity/delta/read_at_ms/contract`
  per episode and writes `zones.jsonl` = every HTF zone in each read's raw universe, touched or
  not. Round 23's own outputs were NOT rebuilt or written to (the CTO's order). Writer diff vs
  d15db077: 120 lines in `harness/eval.go` (`zonePolarity`, `emitZones`).
- **Instrument.** `kernel.DetectTouchOutcomes` at the level price, k=3, Δ = trailing-5-day mean
  |1m close increment| per read (≈5.33 pt), H = 12×1m, exit on close; **hold** = price exits on the
  side it came FROM. p_null = 0.5067. Ambiguous outcomes are excluded from n and counted.
- **Approach vs hold-trade (the convention that reconciles the two rounds).** D1′ `entry`
  names the side price came FROM: `below` = tested as RESISTANCE, `above` = tested as SUPPORT.
  Round 23's `s4_analysis.py` labelled `below` as "long" — the **approach** direction. The
  owner's "support-for-long" is the **hold-trade** direction (approach from above → LONG at
  support; approach from below → SHORT at resistance), which is what a HOLD pays. **This report
  uses hold-trade direction; therefore R23 "agree" ≡ R24 "against-trend" and R23 "oppose" ≡ R24
  "with-trend".** §5's reference table prints both labels side by side.
- **Zone polarity.** support = DEMAND, EQL, OB(bull), iFVG(bull), FVG formed as a gap UP;
  resistance = SUPPLY, EQH, OB(bear), iFVG(bear), FVG formed as a gap DOWN (from the three-candle
  geometry at `FormedAtMs` on the zone's own series); unknown = plain FVG whose formation candle
  was not found on its series (never guessed; 2.3% of HTF zones).
- **With-zone / against-zone.** hold-trade LONG at a support zone or SHORT at a resistance zone
  = with-zone; else against-zone; unknown polarity → "unknown".
- **Zone membership.** Level price P vs zone [lo,hi] of the SAME read, zone tf ∈ {1h,4h,1d}, kinds
  OB/FVG/IFVG/SUPPLY/DEMAND/EQH/EQL. inside = lo≤P≤hi; near = within 1×Δ of the band (EQH/EQL are
  lines, so "near" is the ±1×Δ band). Closest zone by |P − midpoint| decides.
- **Variants.** `all` = every at-zone episode; `noconflict` = only ONE polarity holds P within
  the near band; `own` = the episode's level is itself an HTF zone (PRIMARY).
- **Trend.** S4c's era-wide S1 state per read (`era-trends.jsonl`: `d_trend`, `h4_trend`);
  with/against = hold-trade direction vs the trend; range; 4h = n/a before 2025-05.
- **Lookahead.** Zones and trend state are the READ snapshot (≥30 min before the session
  window, closed bars only); an episode is placed only against zones that existed at its read.
  Q3 uses own-TF bars that CLOSED at or before the anchor (stricter than the kernel's
  `OpenTime <= now`, which admits the forming bar).
- **Statistics.** hold/(hold+break), Wilson 95% CI, lift = rate − p_null in pt. Q2 criterion:
  rate ≥ p_null + 2 pt AND n ≥ 2,000 in EVERY year the cell is measurable.

## 2. Q1 — ZONE × TREND (ordinal-1; PRIMARY = `own`)

**(a) Zone side alone — nothing.** `own`, era: with-zone **0.499 [0.490,0.507] n=12,720** ·
against-zone **0.502 [0.493,0.512] n=10,873**. By year (with / against): 2022 0.484/0.473 ·
2023 0.523/0.522 · 2024 0.494/0.502 · 2025 0.476/0.507 · 2026 0.510/0.502. No year separates
the two sides beyond its CI, and the sign of the difference flips (2022, 2026 with>against;
2024, 2025 against>with). By hold-trade side, era: long-with 0.501 n=6,912 · long-against 0.506
n=6,185 · short-with 0.495 n=5,808 · short-against 0.498 n=4,688.

Context: `noconflict` era with 0.505 n=114,756 vs against 0.507 n=85,952 (year signs flip:
2022 +5.2 pt for with, 2023 −5.6, 2024 −0.4, 2025 −2.1, 2026 +4.8); `all` era with 0.500
n=274,264 vs against 0.510 n=234,495 (against holds MORE — the geometry artefact).

**(b) Zone × 4h trend (2025–26 only).** `own`: with-zone × 4h-with 0.487 [0.450,0.524] n=686 ·
with-zone × 4h-against 0.507 [0.465,0.550] n=536 · against-zone × 4h-with 0.500 n=636 ·
against-zone × 4h-against 0.468 [0.417,0.518] n=370 — all within CI of null. In `noconflict`
the S4d 4h-with penalty is large at zones: with-zone × 4h-with **0.390 [0.375,0.404] n=4,362**
vs with-zone × 4h-against **0.533 [0.519,0.546] n=5,373**; against-zone × 4h-with 0.442
n=5,973. (Under R23's label these are "oppose-4h" and "agree-4h": the same −3 pt oppose
penalty S4d found, wider when the entry is at an unambiguous zone.) By year, `noconflict`
with-zone × 4h-with: 2025 0.425 n=2,952 · 2026 **0.316** n=1,410; with-zone × 4h-against: 2025
0.464 n=3,321 · 2026 **0.643** n=2,052 — the 2026 spread is +33 pt, the 2025 spread +4 pt.

**(c) Zone × D trend.** `own` era: with-zone × D-with 0.493 [0.472,0.515] n=2,023 ·
with-zone × D-against 0.506 n=1,940 · against-zone × D-with 0.495 n=1,819 · against-zone ×
D-against 0.525 [0.500,0.550] n=1,523 — every cell within CI of null. `noconflict` era:
with-zone × D-with **0.532 [0.525,0.538] n=21,379** (+2.5 pt) · with-zone × D-against 0.516 ·
against-zone × D-with **0.479** n=17,967 (−2.7 pt) — by year the with-zone × D-with cell is
2022 0.459 n=3,729 · 2023 0.615 n=4,476 · 2024 0.514 n=6,446 · 2025 0.534 n=5,126 · 2026 0.531
n=1,602: below null in 2022, +11 pt in 2023, +0.7 to +2.7 pt after — the with-D shape S4c found.

**Four-way cross.** `own` cells are n=28–97 (NOT MEASURED at any useful width). `noconflict`
era: with-zone × 4h-against × D-with 0.586 [0.537,0.634] n=394 · with-zone × 4h-against ×
D-against **0.641** n=906 · against-zone × 4h-against × D-with **0.697** n=472 · against-zone ×
4h-with × D-with **0.397** n=1,633 · against-zone × 4h-against × D-against 0.385 n=325. The
ordering is "4h-against beats 4h-with" whatever the zone side and whatever D says; 2025 vs 2026
disagree on every cell's magnitude (e.g. with-zone × 4h-against × D-against 2025 0.322 n=174 vs
2026 0.717 n=732).

## 3. Q2 — KIND × TF × SIDE, per year (with-zone cells)

**PASSING CELLS (rate ≥ 0.5267 and n ≥ 2,000 in every measurable year): NONE — in any
variant.** `own` era cells ≥ 300: OB 1h long 0.507 n=2,251 · OB 1h short 0.491 n=2,074 · EQL 1h
long 0.499 n=2,456 · EQH 1h short 0.484 n=2,121 · DEMAND 1h long 0.499 n=859 · SUPPLY 1h short
0.504 n=779 · IFVG 1h long 0.494 n=757 · IFVG 1h short 0.501 n=483. Every 4h/1d cell is 2026-only
and n ≤ 198 (EQL 4h long 0.535 n=198; OB 4h short 0.612 [0.505,0.708] n=85; SUPPLY 4h short
0.679 n=28; OB 1d long 0.421 n=19). Context `noconflict` era: EQL 1h long **0.551** n=6,643 and
FVG 1h long **0.551** n=3,516 clear +2 pt era-wide but fail the every-year test; SUPPLY 1h short
**0.453** n=14,131 is the worst large cell. `all`: OB 1h long 0.515 n=56,782 (years 0.502 / 0.542 /
0.499 / 0.507 / 0.541 — passes 2023 and 2026 only); OB 1h short 2025 **0.420** n=8,708.

## 4. Q3 — FIRST TOUCH vs RE-ENTRY (Round 23's Q-A with the correct grader)

Grader = `kernel/levels_fresh_by_tf.go` @ dev `324927ad` ported verbatim (counting starts after
the first own-TF CLOSE fully outside the band; consecutive in-band bars = one visit); own-TF bars
from the DB copy keyed by the read's contract, exactly as the harness keyed them; closed bars
only. Population = `out-s4/qa.jsonl` (28,117 HTF-zone ordinal-1 episodes).

| tf | year | fresh | tested-1 | tested-2+ | stale (≥3) |
|---|---|---|---|---|---|
| 1h | era | 0.498 [0.483,0.512] n=4,289 | 0.509 [0.492,0.527] n=3,123 | 0.499 [0.491,0.507] n=15,339 | 0.499 n=12,734 |
| 1h | 2022 | 0.497 n=592 | 0.458 n=452 | 0.479 n=2,701 | 0.485 n=2,282 |
| 1h | 2023 | 0.532 n=931 | 0.527 n=609 | 0.520 n=3,418 | 0.521 n=2,854 |
| 1h | 2024 | 0.487 n=1,042 | 0.513 n=659 | 0.498 n=3,285 | 0.499 n=2,725 |
| 1h | 2025 | 0.500 n=968 | 0.506 n=743 | 0.486 n=3,195 | 0.485 n=2,615 |
| 1h | 2026 | 0.467 [0.432,0.503] n=756 | 0.527 n=660 | 0.508 n=2,740 | 0.502 n=2,258 |
| 4h | 2026 (=era) | 0.536 [0.477,0.595] n=274 | 0.513 n=271 | 0.518 n=740 | 0.526 n=656 |
| 1d | 2026 (=era) | 0.267 n=15 | 0.778 n=9 | 0.500 n=54 | 0.487 n=39 |

Coverage: every 1h row measurable era-wide (own-TF 1h bars exist per contract via
`historical_import`); 4h and 1d exist only for 2026 reads (1,548 and 100 rows); not_measured = 0
because the harness emitted no 4h/1d zone episodes where the series was absent. **Freshness on
own-TF bars separates nothing**: no year's fresh cell beats its re-entered cells beyond CI, and
2026 fresh 1h is the worst cell in the table. Round 23's Q-A "fresh premium" does not exist under
the correct grader either.

## 5. Q5 — PULLBACK ENTRY AT A 1h/4h ZONE

Cell = hold-trade direction WITH the D trend and AGAINST the 4h trend (the "pullback"), entry at
a 1h or 4h zone on the pullback side (with-zone), per year, per side. Under R23's approach
label this is "D-against / 4h-with".

- **`own` (PRIMARY): era with-zone 0.667 [0.496,0.802] n=33** (long 0.680 n=25 · short 0.625 n=8);
  against-zone 0.607 n=28. 2025 with-zone 0.667 n=24; 2026 0.667 n=9. **NOT MEASURED at any
  decision width** — the cell the owner trades has 33 own-zone episodes in the whole era.
- `noconflict`: era with-zone 0.586 [0.537,0.634] n=394 (long 0.589 n=341, short 0.566 n=53);
  against-zone **0.697** [0.654,0.737] n=472 (long 0.728 n=405). 2025 with 0.568 n=329 / against
  0.566 n=242; 2026 with 0.677 n=65 / against **0.835** n=230. The against-zone side is at least
  as good as the with-zone side — the zone's polarity does not carry the pullback.
- `all`: era with-zone 0.596 n=992 (long 0.599 n=785 @1h; short 0.585 n=207: @1h 0.510 n=157,
  @4h 0.820 n=50); against-zone 0.579 n=1,209. 2025 with 0.544 n=638; 2026 with 0.689 n=354.
- **Reference (unrestricted, both labellings):** R24 D-with/4h-against = R23 "D-against/4h-with":
  era long 0.550 n=1,965 · short 0.569 n=945; 2025 long 0.510 / short 0.655; 2026 long 0.617 /
  short 0.513. R24 D-against/4h-with = R23 "D-with/4h-against" (S4d's famous cell): era long
  **0.809** n=1,029 · short 0.425 n=3,431; 2025 long 0.843 n=426 / short 0.362 n=2,085; 2026 long
  0.786 n=603 / short 0.524 n=1,346. S4d's "0.786/0.810 short-only 2026" is, under the hold-trade
  label, a LONG-side cell that is 0.843 in 2025 and 0.786 in 2026 — it did not flip between the
  two measurable years; what flips is which label you read it under. (This reconciliation is the
  point of §1's convention note; the cell itself is still 2025–26 only and unrestricted by zone.)

## 6. What is NOT MEASURED, stated

- 4h and 1d zone cells before 2026 (no own-TF series under the read's contract label — S4c).
- Zone polarity for 2.3% of HTF zones (plain FVG with no formation candle on its series).
- Any `own` four-way or Q5 cell (n ≤ 97).
- 2022–2024 for any 4h-trend cell (S4c coverage).
- Anything not in the MANIFEST below.

## 7. Provenance

Base tree `d15db077` (the commit that carries the Round 23 harness `a488fd59` and the kernel it
was built against). Harness binary sha256 `df59fd66d1ea2ad8`; run: `run_harness.sh` 08:43:35 →
08:49:42 CT rc=0 (`out-r24/harness.log`). Inputs: `db.copy.db` 998a15ee230f321d ·
`out-s4/qa.jsonl` 44e22a2205719527 · `out-s4/trends.jsonl` 00c3d6d5f19cfd3e ·
`out-s4/era-trends.jsonl` 63a55eaea29e6275. Outputs: `episodes.jsonl` 529b41059b81cbdd ·
`zones.jsonl` 37df53ab742675e2. All five analyses ran at scripts commit `4c3c54b4`, sequentially,
`nice -n 19 ionice -c 3` (load rule, CTO 08:53 CT), reading the DB copy only. Reproduce:
`python3 manifest.py > out-r24/manifest.md` after the five scripts.

Method note (CTO ruling 09:05 CT): **the generated tables and the sample-id appendix are the
record; the narrative is commentary on them.** They are emitted by `manifest.py` straight from
the JSON traces and cannot drift from what ran; the narrative is typed by hand and CAN — at
4c3c54b4 §2(c) quoted four `own` D-trend cells that were not in the table (0.516/0.520/0.478/
0.523), caught by cross-checking every narrative number against the JSON before the PR and
corrected at ab110b28 to the table's 0.493/0.506/0.495/0.525. Where a narrative number and a
table disagree, the table is right and the narrative is the defect.

Process note for the record: the first pass of Q1/Q2/Q5 ran as three parallel processes beside
the Go harness at the NY open and stalled the live bar persister (99 × "persist queue stalled
2s"); the CTO's load rule followed and is now standing.

---
## MANIFEST (every cell below is reproducible from this table; anything not here is NOT MEASURED)

| Q | script | commit | inputs (sha256[:16]) | output | lookahead |
|---|---|---|---|---|---|
| Q0 | `q0_density.py` | `4c3c54b4` | zones.jsonl=37df53ab742675e2, era-trends.jsonl=63a55eaea29e6275 | `q0_density.json` |  |
| Q1 | `q1.py` | `4c3c54b4` | episodes.jsonl=529b41059b81cbdd, zones.jsonl=37df53ab742675e2, era-trends.jsonl=63a55eaea29e6275 | `q1_cells.json` | zones and trend state are the READ snapshot (>=30 min before the session window, closed bars only); an episode is placed only against zones that existed at its  |
| Q2 | `q2.py` | `4c3c54b4` | episodes.jsonl=529b41059b81cbdd, zones.jsonl=37df53ab742675e2 | `q2_cells.json` | zones and trend state are the READ snapshot (>=30 min before the session window, closed bars only); an episode is placed only against zones that existed at its  |
| Q3 | `q3.py` | `4c3c54b4` | qa.jsonl=44e22a2205719527, trends.jsonl=00c3d6d5f19cfd3e, db.copy.db=998a15ee230f321d | `q3_cells.json` | own-TF bars with open+tf <= anchor (closed strictly before the touch); origin <= open; grader port of kernel/levels_fresh_by_tf.go@324927ad |
| Q5 | `q5.py` | `4c3c54b4` | episodes.jsonl=529b41059b81cbdd, zones.jsonl=37df53ab742675e2, era-trends.jsonl=63a55eaea29e6275 | `q5_cells.json` | zones and trend state are the READ snapshot (>=30 min before the session window, closed bars only); an episode is placed only against zones that existed at its  |

Harness pass: `run_harness.sh` → binary sha256 df59fd66d1ea2ad8 (built from `git archive d15db077` + this branch's `harness/eval.go`, diff vs d15db077 = 120 lines), `out-r24/harness.log` line 1; episodes.jsonl sha256 529b41059b81cbdd (4,471,482 lines = Round 23's count), zones.jsonl sha256 37df53ab742675e2 (556,237 zones).

## Q0 — HTF zone density per read (reads=3093)

| tf | kind | reads with ≥1 | mean/read | median | p90 | total |
|---|---|---|---|---|---|---|
| 15m | * | 3087 | 96.19 | 89.0 | 158 | 297526 |
| 1d | * | 223 | 3.30 | 0.0 | 0 | 10196 |
| 1d | DEMAND | 44 | 0.04 | 0.0 | 0 | 117 |
| 1d | EQH | 215 | 1.28 | 0.0 | 0 | 3971 |
| 1d | EQL | 215 | 1.28 | 0.0 | 0 | 3966 |
| 1d | FVG | 62 | 0.03 | 0.0 | 0 | 94 |
| 1d | IFVG | 59 | 0.05 | 0.0 | 0 | 141 |
| 1d | OB | 223 | 0.44 | 0.0 | 0 | 1369 |
| 1d | SUPPLY | 215 | 0.17 | 0.0 | 0 | 538 |
| 1h | * | 3063 | 75.16 | 75.0 | 125 | 232455 |
| 1h | DEMAND | 2770 | 5.04 | 5.0 | 10 | 15591 |
| 1h | EQH | 2974 | 12.98 | 14.0 | 20 | 40150 |
| 1h | EQL | 2955 | 12.48 | 14.0 | 20 | 38603 |
| 1h | FVG | 2731 | 3.00 | 3.0 | 6 | 9268 |
| 1h | IFVG | 2774 | 6.89 | 6.0 | 15 | 21320 |
| 1h | OB | 3051 | 29.29 | 24.0 | 60 | 90584 |
| 1h | SUPPLY | 2834 | 5.48 | 5.0 | 11 | 16939 |
| 4h | * | 221 | 5.19 | 0.0 | 0 | 16060 |
| 4h | DEMAND | 206 | 0.38 | 0.0 | 0 | 1188 |
| 4h | EQH | 215 | 1.13 | 0.0 | 0 | 3485 |
| 4h | EQL | 215 | 0.96 | 0.0 | 0 | 2964 |
| 4h | FVG | 218 | 0.13 | 0.0 | 0 | 400 |
| 4h | IFVG | 209 | 0.45 | 0.0 | 0 | 1391 |
| 4h | OB | 221 | 1.79 | 0.0 | 0 | 5530 |
| 4h | SUPPLY | 215 | 0.36 | 0.0 | 0 | 1102 |
| HTF | * | 3065 | 83.64 | 79.0 | 146 | 258711 |

HTF (1h+4h+1d) zones per read by year: 2022 72.9 (median 77, reads 527) · 2023 77.7 (median 81.0, reads 706) · 2024 74.3 (median 75.0, reads 674) · 2025 77.0 (median 73, reads 665) · 2026 124.0 (median 107, reads 517)

Q1 membership counts (ordinal-1 episodes = 845076): {'reads_without_htf_zones': 2372, 'not_at_zone': 167384, 'at_zone_inside': 557193, 'at_zone_near': 118127, 'own_zone_episode': 28117, 'conflict_both_polarities': 387645}

## Q1 — ZONE × TREND

### Q1 variant `own` (PRIMARY — CTO ruling)

| scope | cell | hold rate [Wilson 95%] · n · lift |
|---|---|---|
| era | (a) with-zone | 0.499 [0.490,0.507] · n=12720 (h=6344/b=6376, ambig=2156) · lift -0.8 pt |
| era | (a) against-zone | 0.502 [0.493,0.512] · n=10873 (h=5464/b=5409, ambig=1763) · lift -0.4 pt |
| era | (a) unknown-zone | 0.535 [0.493,0.578] · n=521 (h=279/b=242, ambig=84) · lift +2.9 pt |
| era | (b) with-zone × 4h-with | 0.487 [0.450,0.524] · n=686 (h=334/b=352, ambig=98) · lift -2.0 pt |
| era | (b) with-zone × 4h-against | 0.507 [0.465,0.550] · n=536 (h=272/b=264, ambig=101) · lift +0.1 pt |
| era | (b) with-zone × 4h-range | 0.499 [0.490,0.508] · n=11498 (h=5738/b=5760, ambig=1957) · lift -0.8 pt |
| era | (b) against-zone × 4h-with | 0.500 [0.461,0.539] · n=636 (h=318/b=318, ambig=111) · lift -0.7 pt |
| era | (b) against-zone × 4h-against | 0.468 [0.417,0.518] · n=370 (h=173/b=197, ambig=80) · lift -3.9 pt |
| era | (b) against-zone × 4h-range | 0.504 [0.494,0.514] · n=9867 (h=4973/b=4894, ambig=1572) · lift -0.3 pt |
| era | (c) with-zone × D-with | 0.493 [0.472,0.515] · n=2023 (h=998/b=1025, ambig=333) · lift -1.3 pt |
| era | (c) with-zone × D-against | 0.506 [0.483,0.528] · n=1940 (h=981/b=959, ambig=340) · lift -0.1 pt |
| era | (c) with-zone × D-range | 0.498 [0.488,0.509] · n=8757 (h=4365/b=4392, ambig=1483) · lift -0.8 pt |
| era | (c) against-zone × D-with | 0.495 [0.472,0.518] · n=1819 (h=901/b=918, ambig=303) · lift -1.1 pt |
| era | (c) against-zone × D-against | 0.525 [0.500,0.550] · n=1523 (h=800/b=723, ambig=243) · lift +1.9 pt |
| era | (c) against-zone × D-range | 0.500 [0.488,0.511] · n=7531 (h=3763/b=3768, ambig=1217) · lift -0.7 pt |
| era | 4-way with-zone × 4h-with × D-with | 0.429 [0.308,0.559] · n=56 (h=24/b=32, ambig=10) · lift -7.8 pt |
| era | 4-way with-zone × 4h-with × D-against | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=14) · lift +0.1 pt |
| era | 4-way with-zone × 4h-against × D-with | 0.667 [0.496,0.802] · n=33 (h=22/b=11, ambig=20) · lift +16.0 pt |
| era | 4-way with-zone × 4h-against × D-against | 0.374 [0.281,0.476] · n=91 (h=34/b=57, ambig=14) · lift -13.3 pt |
| era | 4-way against-zone × 4h-with × D-with | 0.536 [0.437,0.632] · n=97 (h=52/b=45, ambig=19) · lift +2.9 pt |
| era | 4-way against-zone × 4h-with × D-against | 0.625 [0.484,0.748] · n=48 (h=30/b=18, ambig=9) · lift +11.8 pt |
| era | 4-way against-zone × 4h-against × D-with | 0.607 [0.424,0.764] · n=28 (h=17/b=11, ambig=18) · lift +10.0 pt |
| era | 4-way against-zone × 4h-against × D-against | 0.387 [0.237,0.562] · n=31 (h=12/b=19, ambig=4) · lift -12.0 pt |
| era | side long × with-zone | 0.501 [0.490,0.513] · n=6912 (h=3466/b=3446, ambig=797) · lift -0.5 pt |
| era | side long × against-zone | 0.506 [0.493,0.518] · n=6185 (h=3127/b=3058, ambig=769) · lift -0.1 pt |
| era | side short × with-zone | 0.495 [0.483,0.508] · n=5808 (h=2878/b=2930, ambig=1359) · lift -1.1 pt |
| era | side short × against-zone | 0.498 [0.484,0.513] · n=4688 (h=2337/b=2351, ambig=994) · lift -0.8 pt |
| 2022 | (a) with-zone | 0.484 [0.462,0.506] · n=1987 (h=962/b=1025, ambig=331) · lift -2.3 pt |
| 2022 | (a) against-zone | 0.473 [0.449,0.497] · n=1675 (h=792/b=883, ambig=312) · lift -3.4 pt |
| 2022 | (a) unknown-zone | 0.494 [0.389,0.599] · n=83 (h=41/b=42, ambig=11) · lift -1.3 pt |
| 2022 | (c) with-zone × D-with | 0.481 [0.424,0.538] · n=287 (h=138/b=149, ambig=52) · lift -2.6 pt |
| 2022 | (c) with-zone × D-against | 0.503 [0.448,0.559] · n=306 (h=154/b=152, ambig=45) · lift -0.3 pt |
| 2022 | (c) against-zone × D-with | 0.453 [0.389,0.518] · n=223 (h=101/b=122, ambig=41) · lift -5.4 pt |
| 2022 | (c) against-zone × D-against | 0.473 [0.409,0.538] · n=226 (h=107/b=119, ambig=41) · lift -3.3 pt |
| 2023 | (a) with-zone | 0.523 [0.504,0.542] · n=2606 (h=1363/b=1243, ambig=450) · lift +1.6 pt |
| 2023 | (a) against-zone | 0.522 [0.501,0.542] · n=2238 (h=1168/b=1070, ambig=395) · lift +1.5 pt |
| 2023 | (a) unknown-zone | 0.561 [0.470,0.649] · n=114 (h=64/b=50, ambig=16) · lift +5.5 pt |
| 2023 | (c) with-zone × D-with | 0.524 [0.476,0.573] · n=408 (h=214/b=194, ambig=74) · lift +1.8 pt |
| 2023 | (c) with-zone × D-against | 0.512 [0.464,0.559] · n=420 (h=215/b=205, ambig=91) · lift +0.5 pt |
| 2023 | (c) against-zone × D-with | 0.540 [0.486,0.593] · n=326 (h=176/b=150, ambig=62) · lift +3.3 pt |
| 2023 | (c) against-zone × D-against | 0.471 [0.423,0.520] · n=399 (h=188/b=211, ambig=74) · lift -3.6 pt |
| 2024 | (a) with-zone | 0.494 [0.475,0.513] · n=2661 (h=1315/b=1346, ambig=429) · lift -1.3 pt |
| 2024 | (a) against-zone | 0.502 [0.481,0.523] · n=2203 (h=1106/b=1097, ambig=279) · lift -0.5 pt |
| 2024 | (a) unknown-zone | 0.508 [0.421,0.595] · n=122 (h=62/b=60, ambig=23) · lift +0.1 pt |
| 2024 | (c) with-zone × D-with | 0.530 [0.485,0.576] · n=460 (h=244/b=216, ambig=68) · lift +2.4 pt |
| 2024 | (c) with-zone × D-against | 0.497 [0.443,0.551] · n=322 (h=160/b=162, ambig=51) · lift -1.0 pt |
| 2024 | (c) against-zone × D-with | 0.474 [0.425,0.524] · n=390 (h=185/b=205, ambig=52) · lift -3.2 pt |
| 2024 | (c) against-zone × D-against | 0.563 [0.505,0.619] · n=286 (h=161/b=125, ambig=36) · lift +5.6 pt |
| 2025 | (a) with-zone | 0.476 [0.457,0.496] · n=2537 (h=1209/b=1328, ambig=408) · lift -3.0 pt |
| 2025 | (a) against-zone | 0.507 [0.486,0.527] · n=2250 (h=1140/b=1110, ambig=281) · lift -0.0 pt |
| 2025 | (a) unknown-zone | 0.546 [0.457,0.633] · n=119 (h=65/b=54, ambig=20) · lift +4.0 pt |
| 2025 | (b) with-zone × 4h-with | 0.494 [0.436,0.553] · n=275 (h=136/b=139, ambig=34) · lift -1.2 pt |
| 2025 | (b) with-zone × 4h-against | 0.443 [0.379,0.509] · n=219 (h=97/b=122, ambig=42) · lift -6.4 pt |
| 2025 | (b) against-zone × 4h-with | 0.498 [0.439,0.557] · n=269 (h=134/b=135, ambig=20) · lift -0.9 pt |
| 2025 | (b) against-zone × 4h-against | 0.469 [0.394,0.546] · n=162 (h=76/b=86, ambig=33) · lift -3.8 pt |
| 2025 | (c) with-zone × D-with | 0.472 [0.430,0.516] · n=510 (h=241/b=269, ambig=61) · lift -3.4 pt |
| 2025 | (c) with-zone × D-against | 0.460 [0.413,0.507] · n=424 (h=195/b=229, ambig=79) · lift -4.7 pt |
| 2025 | (c) against-zone × D-with | 0.495 [0.451,0.539] · n=487 (h=241/b=246, ambig=62) · lift -1.2 pt |
| 2025 | (c) against-zone × D-against | 0.584 [0.530,0.636] · n=327 (h=191/b=136, ambig=36) · lift +7.7 pt |
| 2025 | 4-way with-zone × 4h-with × D-with | 0.429 [0.158,0.750] · n=7 (h=3/b=4, ambig=0) · lift -7.8 pt |
| 2025 | 4-way with-zone × 4h-with × D-against | 0.309 [0.191,0.460] · n=42 (h=13/b=29, ambig=12) · lift -19.7 pt |
| 2025 | 4-way with-zone × 4h-against × D-with | 0.667 [0.467,0.820] · n=24 (h=16/b=8, ambig=12) · lift +16.0 pt |
| 2025 | 4-way with-zone × 4h-against × D-against | 0.000 [0.000,0.324] · n=8 (h=0/b=8, ambig=3) · lift -50.7 pt |
| 2025 | 4-way against-zone × 4h-with × D-with | 0.500 [0.307,0.693] · n=22 (h=11/b=11, ambig=1) · lift -0.7 pt |
| 2025 | 4-way against-zone × 4h-with × D-against | 0.538 [0.355,0.712] · n=26 (h=14/b=12, ambig=3) · lift +3.2 pt |
| 2025 | 4-way against-zone × 4h-against × D-with | 0.471 [0.262,0.690] · n=17 (h=8/b=9, ambig=11) · lift -3.6 pt |
| 2025 | 4-way against-zone × 4h-against × D-against | 0.500 [0.150,0.850] · n=4 (h=2/b=2, ambig=0) · lift -0.7 pt |
| 2026 | (a) with-zone | 0.510 [0.492,0.528] · n=2929 (h=1495/b=1434, ambig=538) · lift +0.4 pt |
| 2026 | (a) against-zone | 0.502 [0.482,0.521] · n=2507 (h=1258/b=1249, ambig=496) · lift -0.5 pt |
| 2026 | (a) unknown-zone | 0.566 [0.459,0.668] · n=83 (h=47/b=36, ambig=14) · lift +6.0 pt |
| 2026 | (b) with-zone × 4h-with | 0.482 [0.434,0.530] · n=411 (h=198/b=213, ambig=64) · lift -2.5 pt |
| 2026 | (b) with-zone × 4h-against | 0.552 [0.497,0.606] · n=317 (h=175/b=142, ambig=59) · lift +4.5 pt |
| 2026 | (b) against-zone × 4h-with | 0.501 [0.451,0.552] · n=367 (h=184/b=183, ambig=91) · lift -0.5 pt |
| 2026 | (b) against-zone × 4h-against | 0.466 [0.400,0.534] · n=208 (h=97/b=111, ambig=47) · lift -4.0 pt |
| 2026 | (c) with-zone × D-with | 0.450 [0.399,0.501] · n=358 (h=161/b=197, ambig=78) · lift -5.7 pt |
| 2026 | (c) with-zone × D-against | 0.549 [0.504,0.594] · n=468 (h=257/b=211, ambig=74) · lift +4.2 pt |
| 2026 | (c) against-zone × D-with | 0.504 [0.455,0.553] · n=393 (h=198/b=195, ambig=86) · lift -0.3 pt |
| 2026 | (c) against-zone × D-against | 0.537 [0.479,0.594] · n=285 (h=153/b=132, ambig=56) · lift +3.0 pt |
| 2026 | 4-way with-zone × 4h-with × D-with | 0.429 [0.300,0.567] · n=49 (h=21/b=28, ambig=10) · lift -7.8 pt |
| 2026 | 4-way with-zone × 4h-with × D-against | 0.840 [0.653,0.936] · n=25 (h=21/b=4, ambig=2) · lift +33.3 pt |
| 2026 | 4-way with-zone × 4h-against × D-with | 0.667 [0.354,0.879] · n=9 (h=6/b=3, ambig=8) · lift +16.0 pt |
| 2026 | 4-way with-zone × 4h-against × D-against | 0.410 [0.310,0.517] · n=83 (h=34/b=49, ambig=11) · lift -9.7 pt |
| 2026 | 4-way against-zone × 4h-with × D-with | 0.547 [0.434,0.654] · n=75 (h=41/b=34, ambig=18) · lift +4.0 pt |
| 2026 | 4-way against-zone × 4h-with × D-against | 0.727 [0.518,0.869] · n=22 (h=16/b=6, ambig=6) · lift +22.1 pt |
| 2026 | 4-way against-zone × 4h-against × D-with | 0.818 [0.523,0.949] · n=11 (h=9/b=2, ambig=7) · lift +31.1 pt |
| 2026 | 4-way against-zone × 4h-against × D-against | 0.370 [0.215,0.558] · n=27 (h=10/b=17, ambig=4) · lift -13.6 pt |

### Q1 variant `noconflict` (context)

| scope | cell | hold rate [Wilson 95%] · n · lift |
|---|---|---|
| era | (a) with-zone | 0.505 [0.502,0.508] · n=114756 (h=57936/b=56820, ambig=35665) · lift -0.2 pt |
| era | (a) against-zone | 0.507 [0.504,0.510] · n=85952 (h=43592/b=42360, ambig=28536) · lift +0.0 pt |
| era | (a) unknown-zone | 0.510 [0.503,0.518] · n=17415 (h=8887/b=8528, ambig=5351) · lift +0.4 pt |
| era | (b) with-zone × 4h-with | 0.390 [0.375,0.404] · n=4362 (h=1699/b=2663, ambig=835) · lift -11.7 pt |
| era | (b) with-zone × 4h-against | 0.533 [0.519,0.546] · n=5373 (h=2862/b=2511, ambig=1293) · lift +2.6 pt |
| era | (b) with-zone × 4h-range | 0.508 [0.505,0.511] · n=105021 (h=53375/b=51646, ambig=33537) · lift +0.2 pt |
| era | (b) against-zone × 4h-with | 0.442 [0.430,0.455] · n=5973 (h=2642/b=3331, ambig=2459) · lift -6.4 pt |
| era | (b) against-zone × 4h-against | 0.495 [0.478,0.512] · n=3322 (h=1645/b=1677, ambig=818) · lift -1.2 pt |
| era | (b) against-zone × 4h-range | 0.513 [0.509,0.516] · n=76657 (h=39305/b=37352, ambig=25259) · lift +0.6 pt |
| era | (c) with-zone × D-with | 0.532 [0.525,0.538] · n=21379 (h=11369/b=10010, ambig=6189) · lift +2.5 pt |
| era | (c) with-zone × D-against | 0.516 [0.509,0.522] · n=22698 (h=11708/b=10990, ambig=6875) · lift +0.9 pt |
| era | (c) with-zone × D-range | 0.493 [0.489,0.497] · n=70679 (h=34859/b=35820, ambig=22601) · lift -1.3 pt |
| era | (c) against-zone × D-with | 0.479 [0.472,0.487] · n=17967 (h=8614/b=9353, ambig=7040) · lift -2.7 pt |
| era | (c) against-zone × D-against | 0.509 [0.501,0.517] · n=14641 (h=7458/b=7183, ambig=4352) · lift +0.3 pt |
| era | (c) against-zone × D-range | 0.516 [0.512,0.520] · n=53344 (h=27520/b=25824, ambig=17144) · lift +0.9 pt |
| era | 4-way with-zone × 4h-with × D-with | 0.625 [0.555,0.690] · n=192 (h=120/b=72, ambig=70) · lift +11.8 pt |
| era | 4-way with-zone × 4h-with × D-against | 0.439 [0.413,0.466] · n=1353 (h=594/b=759, ambig=188) · lift -6.8 pt |
| era | 4-way with-zone × 4h-against × D-with | 0.586 [0.537,0.634] · n=394 (h=231/b=163, ambig=108) · lift +8.0 pt |
| era | 4-way with-zone × 4h-against × D-against | 0.641 [0.610,0.672] · n=906 (h=581/b=325, ambig=461) · lift +13.5 pt |
| era | 4-way against-zone × 4h-with × D-with | 0.397 [0.374,0.421] · n=1633 (h=649/b=984, ambig=1036) · lift -10.9 pt |
| era | 4-way against-zone × 4h-with × D-against | 0.544 [0.473,0.612] · n=195 (h=106/b=89, ambig=23) · lift +3.7 pt |
| era | 4-way against-zone × 4h-against × D-with | 0.697 [0.654,0.737] · n=472 (h=329/b=143, ambig=252) · lift +19.0 pt |
| era | 4-way against-zone × 4h-against × D-against | 0.385 [0.333,0.439] · n=325 (h=125/b=200, ambig=13) · lift -12.2 pt |
| era | side long × with-zone | 0.522 [0.518,0.526] · n=58205 (h=30399/b=27806, ambig=13311) · lift +1.6 pt |
| era | side long × against-zone | 0.525 [0.521,0.530] · n=46497 (h=24425/b=22072, ambig=14508) · lift +1.9 pt |
| era | side short × with-zone | 0.487 [0.483,0.491] · n=56551 (h=27537/b=29014, ambig=22354) · lift -2.0 pt |
| era | side short × against-zone | 0.486 [0.481,0.491] · n=39455 (h=19167/b=20288, ambig=14028) · lift -2.1 pt |
| 2022 | (a) with-zone | 0.519 [0.512,0.526] · n=19245 (h=9986/b=9259, ambig=6829) · lift +1.2 pt |
| 2022 | (a) against-zone | 0.467 [0.459,0.475] · n=15455 (h=7219/b=8236, ambig=6265) · lift -4.0 pt |
| 2022 | (a) unknown-zone | 0.560 [0.539,0.580] · n=2264 (h=1268/b=996, ambig=513) · lift +5.3 pt |
| 2022 | (c) with-zone × D-with | 0.459 [0.443,0.475] · n=3729 (h=1713/b=2016, ambig=2308) · lift -4.7 pt |
| 2022 | (c) with-zone × D-against | 0.517 [0.504,0.531] · n=5333 (h=2760/b=2573, ambig=1341) · lift +1.1 pt |
| 2022 | (c) against-zone × D-with | 0.433 [0.418,0.447] · n=4476 (h=1937/b=2539, ambig=2324) · lift -7.4 pt |
| 2022 | (c) against-zone × D-against | 0.455 [0.439,0.471] · n=3626 (h=1650/b=1976, ambig=791) · lift -5.2 pt |
| 2023 | (a) with-zone | 0.517 [0.511,0.524] · n=24270 (h=12556/b=11714, ambig=9379) · lift +1.1 pt |
| 2023 | (a) against-zone | 0.573 [0.566,0.581] · n=18281 (h=10481/b=7800, ambig=7486) · lift +6.7 pt |
| 2023 | (a) unknown-zone | 0.497 [0.481,0.512] · n=3975 (h=1974/b=2001, ambig=1326) · lift -1.0 pt |
| 2023 | (c) with-zone × D-with | 0.615 [0.601,0.629] · n=4476 (h=2754/b=1722, ambig=1315) · lift +10.9 pt |
| 2023 | (c) with-zone × D-against | 0.424 [0.409,0.441] · n=3715 (h=1577/b=2138, ambig=1428) · lift -8.2 pt |
| 2023 | (c) against-zone × D-with | 0.536 [0.515,0.557] · n=2191 (h=1175/b=1016, ambig=1176) · lift +3.0 pt |
| 2023 | (c) against-zone × D-against | 0.569 [0.553,0.585] · n=3602 (h=2049/b=1553, ambig=1513) · lift +6.2 pt |
| 2024 | (a) with-zone | 0.501 [0.495,0.507] · n=27484 (h=13769/b=13715, ambig=8499) · lift -0.6 pt |
| 2024 | (a) against-zone | 0.505 [0.498,0.512] · n=19220 (h=9704/b=9516, ambig=5527) · lift -0.2 pt |
| 2024 | (a) unknown-zone | 0.517 [0.502,0.532] · n=4462 (h=2307/b=2155, ambig=1469) · lift +1.0 pt |
| 2024 | (c) with-zone × D-with | 0.514 [0.502,0.527] · n=6446 (h=3316/b=3130, ambig=1549) · lift +0.8 pt |
| 2024 | (c) with-zone × D-against | 0.508 [0.494,0.523] · n=4494 (h=2284/b=2210, ambig=1619) · lift +0.2 pt |
| 2024 | (c) against-zone × D-with | 0.554 [0.538,0.570] · n=3701 (h=2051/b=1650, ambig=1369) · lift +4.7 pt |
| 2024 | (c) against-zone × D-against | 0.493 [0.476,0.509] · n=3500 (h=1725/b=1775, ambig=1105) · lift -1.4 pt |
| 2025 | (a) with-zone | 0.478 [0.472,0.484] · n=28338 (h=13541/b=14797, ambig=6627) · lift -2.9 pt |
| 2025 | (a) against-zone | 0.499 [0.492,0.506] · n=20836 (h=10399/b=10437, ambig=4811) · lift -0.8 pt |
| 2025 | (a) unknown-zone | 0.483 [0.469,0.496] · n=5392 (h=2603/b=2789, ambig=1710) · lift -2.4 pt |
| 2025 | (b) with-zone × 4h-with | 0.425 [0.407,0.443] · n=2952 (h=1254/b=1698, ambig=542) · lift -8.2 pt |
| 2025 | (b) with-zone × 4h-against | 0.464 [0.447,0.481] · n=3321 (h=1542/b=1779, ambig=519) · lift -4.2 pt |
| 2025 | (b) against-zone × 4h-with | 0.458 [0.439,0.476] · n=2810 (h=1286/b=1524, ambig=457) · lift -4.9 pt |
| 2025 | (b) against-zone × 4h-against | 0.451 [0.431,0.472] · n=2348 (h=1060/b=1288, ambig=485) · lift -5.5 pt |
| 2025 | (c) with-zone × D-with | 0.534 [0.520,0.547] · n=5126 (h=2736/b=2390, ambig=618) · lift +2.7 pt |
| 2025 | (c) with-zone × D-against | 0.546 [0.533,0.559] · n=5362 (h=2929/b=2433, ambig=1516) · lift +4.0 pt |
| 2025 | (c) against-zone × D-with | 0.445 [0.430,0.460] · n=4372 (h=1945/b=2427, ambig=797) · lift -6.2 pt |
| 2025 | (c) against-zone × D-against | 0.516 [0.497,0.535] · n=2614 (h=1349/b=1265, ambig=583) · lift +0.9 pt |
| 2025 | 4-way with-zone × 4h-with × D-with | 0.827 [0.731,0.894] · n=81 (h=67/b=14, ambig=11) · lift +32.0 pt |
| 2025 | 4-way with-zone × 4h-with × D-against | 0.445 [0.410,0.480] · n=789 (h=351/b=438, ambig=123) · lift -6.2 pt |
| 2025 | 4-way with-zone × 4h-against × D-with | 0.568 [0.514,0.621] · n=329 (h=187/b=142, ambig=106) · lift +6.2 pt |
| 2025 | 4-way with-zone × 4h-against × D-against | 0.322 [0.257,0.395] · n=174 (h=56/b=118, ambig=95) · lift -18.5 pt |
| 2025 | 4-way against-zone × 4h-with × D-with | 0.264 [0.230,0.302] · n=568 (h=150/b=418, ambig=129) · lift -24.3 pt |
| 2025 | 4-way against-zone × 4h-with × D-against | 0.576 [0.503,0.647] · n=177 (h=102/b=75, ambig=0) · lift +7.0 pt |
| 2025 | 4-way against-zone × 4h-against × D-with | 0.566 [0.503,0.627] · n=242 (h=137/b=105, ambig=126) · lift +5.9 pt |
| 2025 | 4-way against-zone × 4h-against × D-against | 0.258 [0.204,0.321] · n=213 (h=55/b=158, ambig=2) · lift -24.8 pt |
| 2026 | (a) with-zone | 0.524 [0.516,0.532] · n=15419 (h=8084/b=7335, ambig=4331) · lift +1.8 pt |
| 2026 | (a) against-zone | 0.476 [0.467,0.485] · n=12160 (h=5789/b=6371, ambig=4447) · lift -3.1 pt |
| 2026 | (a) unknown-zone | 0.556 [0.529,0.583] · n=1322 (h=735/b=587, ambig=333) · lift +4.9 pt |
| 2026 | (b) with-zone × 4h-with | 0.316 [0.292,0.340] · n=1410 (h=445/b=965, ambig=293) · lift -19.1 pt |
| 2026 | (b) with-zone × 4h-against | 0.643 [0.622,0.664] · n=2052 (h=1320/b=732, ambig=774) · lift +13.7 pt |
| 2026 | (b) against-zone × 4h-with | 0.429 [0.412,0.446] · n=3163 (h=1356/b=1807, ambig=2002) · lift -7.8 pt |
| 2026 | (b) against-zone × 4h-against | 0.601 [0.570,0.631] · n=974 (h=585/b=389, ambig=333) · lift +9.4 pt |
| 2026 | (c) with-zone × D-with | 0.531 [0.506,0.555] · n=1602 (h=850/b=752, ambig=399) · lift +2.4 pt |
| 2026 | (c) with-zone × D-against | 0.569 [0.553,0.585] · n=3794 (h=2158/b=1636, ambig=971) · lift +6.2 pt |
| 2026 | (c) against-zone × D-with | 0.467 [0.450,0.484] · n=3227 (h=1506/b=1721, ambig=1374) · lift -4.0 pt |
| 2026 | (c) against-zone × D-against | 0.527 [0.500,0.554] · n=1299 (h=685/b=614, ambig=360) · lift +2.1 pt |
| 2026 | 4-way with-zone × 4h-with × D-with | 0.477 [0.387,0.570] · n=111 (h=53/b=58, ambig=59) · lift -2.9 pt |
| 2026 | 4-way with-zone × 4h-with × D-against | 0.431 [0.391,0.472] · n=564 (h=243/b=321, ambig=65) · lift -7.6 pt |
| 2026 | 4-way with-zone × 4h-against × D-with | 0.677 [0.556,0.778] · n=65 (h=44/b=21, ambig=2) · lift +17.0 pt |
| 2026 | 4-way with-zone × 4h-against × D-against | 0.717 [0.683,0.749] · n=732 (h=525/b=207, ambig=366) · lift +21.1 pt |
| 2026 | 4-way against-zone × 4h-with × D-with | 0.469 [0.439,0.499] · n=1065 (h=499/b=566, ambig=907) · lift -3.8 pt |
| 2026 | 4-way against-zone × 4h-with × D-against | 0.222 [0.090,0.452] · n=18 (h=4/b=14, ambig=23) · lift -28.4 pt |
| 2026 | 4-way against-zone × 4h-against × D-with | 0.835 [0.781,0.877] · n=230 (h=192/b=38, ambig=126) · lift +32.8 pt |
| 2026 | 4-way against-zone × 4h-against × D-against | 0.625 [0.533,0.709] · n=112 (h=70/b=42, ambig=11) · lift +11.8 pt |

### Q1 variant `all` (context)

| scope | cell | hold rate [Wilson 95%] · n · lift |
|---|---|---|
| era | (a) with-zone | 0.500 [0.498,0.502] · n=274272 (h=137240/b=137032, ambig=73413) · lift -0.6 pt |
| era | (a) against-zone | 0.510 [0.508,0.512] · n=234489 (h=119567/b=114922, ambig=64807) · lift +0.3 pt |
| era | (a) unknown-zone | 0.521 [0.514,0.527] · n=22033 (h=11471/b=10562, ambig=6306) · lift +1.4 pt |
| era | (b) with-zone × 4h-with | 0.450 [0.442,0.458] · n=13870 (h=6238/b=7632, ambig=2542) · lift -5.7 pt |
| era | (b) with-zone × 4h-against | 0.513 [0.504,0.521] · n=13874 (h=7115/b=6759, ambig=3311) · lift +0.6 pt |
| era | (b) with-zone × 4h-range | 0.502 [0.501,0.504] · n=246528 (h=123887/b=122641, ambig=67560) · lift -0.4 pt |
| era | (b) against-zone × 4h-with | 0.458 [0.450,0.466] · n=14031 (h=6424/b=7607, ambig=3774) · lift -4.9 pt |
| era | (b) against-zone × 4h-against | 0.470 [0.461,0.479] · n=10750 (h=5053/b=5697, ambig=3526) · lift -3.7 pt |
| era | (b) against-zone × 4h-range | 0.515 [0.513,0.518] · n=209708 (h=108090/b=101618, ambig=57507) · lift +0.9 pt |
| era | (c) with-zone × D-with | 0.511 [0.506,0.515] · n=46495 (h=23758/b=22737, ambig=12574) · lift +0.4 pt |
| era | (c) with-zone × D-against | 0.516 [0.512,0.521] · n=50735 (h=26199/b=24536, ambig=12722) · lift +1.0 pt |
| era | (c) with-zone × D-range | 0.493 [0.491,0.495] · n=177042 (h=87283/b=89759, ambig=48117) · lift -1.4 pt |
| era | (c) against-zone × D-with | 0.489 [0.484,0.494] · n=42091 (h=20578/b=21513, ambig=13510) · lift -1.8 pt |
| era | (c) against-zone × D-against | 0.516 [0.511,0.521] · n=37800 (h=19510/b=18290, ambig=9973) · lift +0.9 pt |
| era | (c) against-zone × D-range | 0.514 [0.512,0.517] · n=154598 (h=79479/b=75119, ambig=41324) · lift +0.7 pt |
| era | 4-way with-zone × 4h-with × D-with | 0.545 [0.515,0.574] · n=1118 (h=609/b=509, ambig=224) · lift +3.8 pt |
| era | 4-way with-zone × 4h-with × D-against | 0.534 [0.514,0.554] · n=2399 (h=1282/b=1117, ambig=299) · lift +2.8 pt |
| era | 4-way with-zone × 4h-against × D-with | 0.596 [0.565,0.626] · n=992 (h=591/b=401, ambig=452) · lift +8.9 pt |
| era | 4-way with-zone × 4h-against × D-against | 0.536 [0.516,0.557] · n=2271 (h=1218/b=1053, ambig=590) · lift +3.0 pt |
| era | 4-way against-zone × 4h-with × D-with | 0.427 [0.409,0.446] · n=2759 (h=1179/b=1580, ambig=1220) · lift -7.9 pt |
| era | 4-way against-zone × 4h-with × D-against | 0.474 [0.446,0.503] · n=1172 (h=556/b=616, ambig=129) · lift -3.2 pt |
| era | 4-way against-zone × 4h-against × D-with | 0.579 [0.551,0.607] · n=1209 (h=700/b=509, ambig=722) · lift +7.2 pt |
| era | 4-way against-zone × 4h-against × D-against | 0.348 [0.321,0.377] · n=1131 (h=394/b=737, ambig=309) · lift -15.8 pt |
| era | side long × with-zone | 0.501 [0.498,0.503] · n=146463 (h=73348/b=73115, ambig=31232) · lift -0.6 pt |
| era | side long × against-zone | 0.515 [0.512,0.518] · n=130913 (h=67413/b=63500, ambig=31777) · lift +0.8 pt |
| era | side short × with-zone | 0.500 [0.497,0.503] · n=127809 (h=63892/b=63917, ambig=42181) · lift -0.7 pt |
| era | side short × against-zone | 0.503 [0.500,0.507] · n=103576 (h=52154/b=51422, ambig=33030) · lift -0.3 pt |
| 2022 | (a) with-zone | 0.507 [0.502,0.512] · n=43599 (h=22104/b=21495, ambig=11800) · lift +0.0 pt |
| 2022 | (a) against-zone | 0.486 [0.481,0.491] · n=38657 (h=18781/b=19876, ambig=12695) · lift -2.1 pt |
| 2022 | (a) unknown-zone | 0.526 [0.509,0.544] · n=3051 (h=1606/b=1445, ambig=649) · lift +2.0 pt |
| 2022 | (c) with-zone × D-with | 0.470 [0.459,0.481] · n=7594 (h=3570/b=4024, ambig=3205) · lift -3.7 pt |
| 2022 | (c) with-zone × D-against | 0.513 [0.503,0.523] · n=10177 (h=5222/b=4955, ambig=2362) · lift +0.6 pt |
| 2022 | (c) against-zone × D-with | 0.479 [0.469,0.490] · n=7952 (h=3813/b=4139, ambig=3590) · lift -2.7 pt |
| 2022 | (c) against-zone × D-against | 0.459 [0.448,0.470] · n=8113 (h=3722/b=4391, ambig=1743) · lift -4.8 pt |
| 2023 | (a) with-zone | 0.520 [0.516,0.525] · n=55354 (h=28808/b=26546, ambig=17791) · lift +1.4 pt |
| 2023 | (a) against-zone | 0.545 [0.541,0.550] · n=48368 (h=26380/b=21988, ambig=15904) · lift +3.9 pt |
| 2023 | (a) unknown-zone | 0.536 [0.522,0.550] · n=4688 (h=2513/b=2175, ambig=1426) · lift +2.9 pt |
| 2023 | (c) with-zone × D-with | 0.553 [0.543,0.563] · n=9885 (h=5467/b=4418, ambig=2930) · lift +4.6 pt |
| 2023 | (c) with-zone × D-against | 0.489 [0.479,0.499] · n=9608 (h=4700/b=4908, ambig=3114) · lift -1.8 pt |
| 2023 | (c) against-zone × D-with | 0.514 [0.503,0.525] · n=7617 (h=3914/b=3703, ambig=2809) · lift +0.7 pt |
| 2023 | (c) against-zone × D-against | 0.544 [0.534,0.555] · n=8982 (h=4890/b=4092, ambig=3050) · lift +3.8 pt |
| 2024 | (a) with-zone | 0.498 [0.494,0.502] · n=61990 (h=30880/b=31110, ambig=17282) · lift -0.9 pt |
| 2024 | (a) against-zone | 0.516 [0.512,0.521] · n=49413 (h=25521/b=23892, ambig=12017) · lift +1.0 pt |
| 2024 | (a) unknown-zone | 0.531 [0.518,0.543] · n=5675 (h=3011/b=2664, ambig=1820) · lift +2.4 pt |
| 2024 | (c) with-zone × D-with | 0.538 [0.529,0.547] · n=11612 (h=6247/b=5365, ambig=3032) · lift +3.1 pt |
| 2024 | (c) with-zone × D-against | 0.512 [0.502,0.521] · n=10545 (h=5397/b=5148, ambig=2631) · lift +0.5 pt |
| 2024 | (c) against-zone × D-with | 0.566 [0.555,0.577] · n=8342 (h=4721/b=3621, ambig=2324) · lift +5.9 pt |
| 2024 | (c) against-zone × D-against | 0.517 [0.506,0.527] · n=8706 (h=4500/b=4206, ambig=2399) · lift +1.0 pt |
| 2025 | (a) with-zone | 0.468 [0.464,0.472] · n=56203 (h=26294/b=29909, ambig=13148) · lift -3.9 pt |
| 2025 | (a) against-zone | 0.503 [0.498,0.507] · n=48737 (h=24509/b=24228, ambig=10389) · lift -0.4 pt |
| 2025 | (a) unknown-zone | 0.484 [0.471,0.496] · n=6392 (h=3092/b=3300, ambig=1936) · lift -2.3 pt |
| 2025 | (b) with-zone × 4h-with | 0.447 [0.434,0.460] · n=5603 (h=2506/b=3097, ambig=933) · lift -5.9 pt |
| 2025 | (b) with-zone × 4h-against | 0.458 [0.445,0.470] · n=6051 (h=2770/b=3281, ambig=1420) · lift -4.9 pt |
| 2025 | (b) against-zone × 4h-with | 0.456 [0.444,0.469] · n=5912 (h=2697/b=3215, ambig=739) · lift -5.1 pt |
| 2025 | (b) against-zone × 4h-against | 0.470 [0.455,0.485] · n=4274 (h=2009/b=2265, ambig=1110) · lift -3.7 pt |
| 2025 | (c) with-zone × D-with | 0.470 [0.460,0.480] · n=9609 (h=4518/b=5091, ambig=1963) · lift -3.7 pt |
| 2025 | (c) with-zone × D-against | 0.504 [0.494,0.514] · n=10163 (h=5123/b=5040, ambig=2454) · lift -0.3 pt |
| 2025 | (c) against-zone × D-with | 0.446 [0.436,0.456] · n=9316 (h=4153/b=5163, ambig=2269) · lift -6.1 pt |
| 2025 | (c) against-zone × D-against | 0.545 [0.533,0.557] · n=6807 (h=3711/b=3096, ambig=1395) · lift +3.8 pt |
| 2025 | 4-way with-zone × 4h-with × D-with | 0.661 [0.577,0.737] · n=130 (h=86/b=44, ambig=11) · lift +15.5 pt |
| 2025 | 4-way with-zone × 4h-with × D-against | 0.420 [0.392,0.450] · n=1109 (h=466/b=643, ambig=197) · lift -8.7 pt |
| 2025 | 4-way with-zone × 4h-against × D-with | 0.544 [0.505,0.582] · n=638 (h=347/b=291, ambig=383) · lift +3.7 pt |
| 2025 | 4-way with-zone × 4h-against × D-against | 0.156 [0.122,0.197] · n=359 (h=56/b=303, ambig=95) · lift -35.1 pt |
| 2025 | 4-way against-zone × 4h-with × D-with | 0.377 [0.343,0.412] · n=762 (h=287/b=475, ambig=139) · lift -13.0 pt |
| 2025 | 4-way against-zone × 4h-with × D-against | 0.407 [0.373,0.442] · n=777 (h=316/b=461, ambig=46) · lift -10.0 pt |
| 2025 | 4-way against-zone × 4h-against × D-with | 0.605 [0.561,0.646] · n=506 (h=306/b=200, ambig=392) · lift +9.8 pt |
| 2025 | 4-way against-zone × 4h-against × D-against | 0.258 [0.204,0.321] · n=213 (h=55/b=158, ambig=2) · lift -24.8 pt |
| 2026 | (a) with-zone | 0.510 [0.506,0.514] · n=57126 (h=29154/b=27972, ambig=13392) · lift +0.4 pt |
| 2026 | (a) against-zone | 0.494 [0.490,0.499] · n=49314 (h=24376/b=24938, ambig=13802) · lift -1.2 pt |
| 2026 | (a) unknown-zone | 0.561 [0.540,0.581] · n=2227 (h=1249/b=978, ambig=475) · lift +5.4 pt |
| 2026 | (b) with-zone × 4h-with | 0.451 [0.441,0.462] · n=8267 (h=3732/b=4535, ambig=1609) · lift -5.5 pt |
| 2026 | (b) with-zone × 4h-against | 0.555 [0.544,0.566] · n=7823 (h=4345/b=3478, ambig=1891) · lift +4.9 pt |
| 2026 | (b) against-zone × 4h-with | 0.459 [0.448,0.470] · n=8119 (h=3727/b=4392, ambig=3035) · lift -4.8 pt |
| 2026 | (b) against-zone × 4h-against | 0.470 [0.458,0.482] · n=6476 (h=3044/b=3432, ambig=2416) · lift -3.7 pt |
| 2026 | (c) with-zone × D-with | 0.507 [0.496,0.519] · n=7795 (h=3956/b=3839, ambig=1444) · lift +0.1 pt |
| 2026 | (c) with-zone × D-against | 0.562 [0.552,0.572] · n=10242 (h=5757/b=4485, ambig=2161) · lift +5.5 pt |
| 2026 | (c) against-zone × D-with | 0.449 [0.438,0.459] · n=8864 (h=3977/b=4887, ambig=2518) · lift -5.8 pt |
| 2026 | (c) against-zone × D-against | 0.517 [0.504,0.531] · n=5192 (h=2687/b=2505, ambig=1386) · lift +1.1 pt |
| 2026 | 4-way with-zone × 4h-with × D-with | 0.529 [0.498,0.560] · n=988 (h=523/b=465, ambig=213) · lift +2.3 pt |
| 2026 | 4-way with-zone × 4h-with × D-against | 0.633 [0.606,0.658] · n=1290 (h=816/b=474, ambig=102) · lift +12.6 pt |
| 2026 | 4-way with-zone × 4h-against × D-with | 0.689 [0.639,0.735] · n=354 (h=244/b=110, ambig=69) · lift +18.3 pt |
| 2026 | 4-way with-zone × 4h-against × D-against | 0.608 [0.586,0.629] · n=1912 (h=1162/b=750, ambig=495) · lift +10.1 pt |
| 2026 | 4-way against-zone × 4h-with × D-with | 0.447 [0.425,0.469] · n=1997 (h=892/b=1105, ambig=1081) · lift -6.0 pt |
| 2026 | 4-way against-zone × 4h-with × D-against | 0.608 [0.559,0.654] · n=395 (h=240/b=155, ambig=83) · lift +10.1 pt |
| 2026 | 4-way against-zone × 4h-against × D-with | 0.560 [0.523,0.597] · n=703 (h=394/b=309, ambig=330) · lift +5.4 pt |
| 2026 | 4-way against-zone × 4h-against × D-against | 0.369 [0.339,0.401] · n=918 (h=339/b=579, ambig=307) · lift -13.7 pt |

## Q2 — KIND × TF × SIDE (with-zone), per year

Criterion: with-zone cell (kind, tf, side): rate >= p_null+0.02 AND n >= 2000 in every year with n>=1. **PASSING CELLS: NONE, in any variant**

### Q2 variant `own` (PRIMARY)

| kind | tf | side | era with-zone | era against-zone | by year (with-zone) | passes every year |
|---|---|---|---|---|---|---|
| DEMAND | 1h | long | 0.499 [0.466,0.533] · n=859 (h=429/b=430, ambig=91) · lift -0.7 pt | — | 2022:0.500 [0.416,0.584] n=132 (h=66 b=66)✗; 2023:0.494 [0.422,0.567] n=178 (h=88 b=90)✗; 2024:0.502 [0.434,0.571] n=199 (h=100 b=99)✗; 2025:0.470 [0.403,0.539] n=202 (h=95 b=107)✗; 2026:0.540 [0.460,0.619] n=148 (h=80 b=68)✗ | False |
| EQH | 1h | short | 0.484 [0.463,0.505] · n=2121 (h=1027/b=1094, ambig=537) · lift -2.2 pt | — | 2022:0.489 [0.436,0.543] n=333 (h=163 b=170)✗; 2023:0.493 [0.448,0.539] n=454 (h=224 b=230)✗; 2024:0.474 [0.428,0.521] n=443 (h=210 b=233)✗; 2025:0.462 [0.418,0.507] n=478 (h=221 b=257)✗; 2026:0.506 [0.458,0.554] n=413 (h=209 b=204)✗ | False |
| EQL | 1h | long | 0.499 [0.479,0.519] · n=2456 (h=1226/b=1230, ambig=321) · lift -0.8 pt | — | 2022:0.488 [0.432,0.545] n=301 (h=147 b=154)✗; 2023:0.530 [0.483,0.576] n=434 (h=230 b=204)✗; 2024:0.484 [0.443,0.525] n=560 (h=271 b=289)✗; 2025:0.511 [0.471,0.551] n=587 (h=300 b=287)✗; 2026:0.484 [0.444,0.525] n=574 (h=278 b=296)✗ | False |
| IFVG | 1h | long | 0.494 [0.459,0.530] · n=757 (h=374/b=383, ambig=82) · lift -1.3 pt | 0.487 [0.443,0.532] · n=480 (h=234/b=246, ambig=59) · lift -1.9 pt | 2022:0.492 [0.406,0.579] n=124 (h=61 b=63)✗; 2023:0.485 [0.401,0.569] n=132 (h=64 b=68)✗; 2024:0.555 [0.482,0.625] n=182 (h=101 b=81)✗; 2025:0.452 [0.379,0.528] n=168 (h=76 b=92)✗; 2026:0.477 [0.399,0.556] n=151 (h=72 b=79)✗ | False |
| IFVG | 1h | short | 0.501 [0.457,0.545] · n=483 (h=242/b=241, ambig=78) · lift -0.6 pt | 0.526 [0.481,0.570] · n=483 (h=254/b=229, ambig=95) · lift +1.9 pt | 2022:0.534 [0.438,0.627] n=103 (h=55 b=48)✗; 2023:0.600 [0.519,0.676] n=145 (h=87 b=58)✗; 2024:0.462 [0.363,0.563] n=91 (h=42 b=49)✗; 2025:0.409 [0.314,0.510] n=93 (h=38 b=55)✗; 2026:0.392 [0.270,0.529] n=51 (h=20 b=31)✗ | False |
| OB | 1h | long | 0.507 [0.486,0.527] · n=2251 (h=1141/b=1110, ambig=217) · lift +0.0 pt | 0.510 [0.490,0.530] · n=2390 (h=1220/b=1170, ambig=264) · lift +0.4 pt | 2022:0.441 [0.389,0.493] n=345 (h=152 b=193)✗; 2023:0.546 [0.501,0.589] n=493 (h=269 b=224)✗; 2024:0.530 [0.487,0.571] n=540 (h=286 b=254)✗; 2025:0.479 [0.434,0.525] n=463 (h=222 b=241)✗; 2026:0.517 [0.469,0.565] n=410 (h=212 b=198)✗ | False |
| OB | 1h | short | 0.491 [0.469,0.512] · n=2074 (h=1018/b=1056, ambig=467) · lift -1.6 pt | 0.492 [0.466,0.517] · n=1464 (h=720/b=744, ambig=278) · lift -1.5 pt | 2022:0.494 [0.448,0.541] n=447 (h=221 b=226)✗; 2023:0.512 [0.470,0.555] n=527 (h=270 b=257)✗; 2024:0.471 [0.425,0.517] n=446 (h=210 b=236)✗; 2025:0.469 [0.415,0.522] n=333 (h=156 b=177)✗; 2026:0.502 [0.447,0.556] n=321 (h=161 b=160)✗ | False |
| SUPPLY | 1h | short | 0.504 [0.469,0.539] · n=779 (h=393/b=386, ambig=186) · lift -0.2 pt | — | 2022:0.484 [0.408,0.561] n=159 (h=77 b=82)✗; 2023:0.550 [0.475,0.623] n=169 (h=93 b=76)✗; 2024:0.483 [0.404,0.563] n=149 (h=72 b=77)✗; 2025:0.462 [0.391,0.534] n=184 (h=85 b=99)✗; 2026:0.559 [0.469,0.646] n=118 (h=66 b=52)✗ | False |

### Q2 variant `noconflict` (context)

| kind | tf | side | era with-zone | era against-zone | by year (with-zone) | passes every year |
|---|---|---|---|---|---|---|
| DEMAND | 1h | long | 0.493 [0.483,0.502] · n=10020 (h=4937/b=5083, ambig=2650) · lift -1.4 pt | — | 2022:0.487 [0.463,0.511] n=1626 (h=792 b=834)✗; 2023:0.470 [0.452,0.489] n=2706 (h=1273 b=1433)✗; 2024:0.551 [0.526,0.576] n=1493 (h=823 b=670)✗; 2025:0.422 [0.404,0.441] n=2804 (h=1184 b=1620)✗; 2026:0.622 [0.596,0.647] n=1391 (h=865 b=526)✗ | False |
| EQH | 1h | short | 0.514 [0.503,0.526] · n=7016 (h=3609/b=3407, ambig=2945) · lift +0.8 pt | — | 2022:0.578 [0.546,0.609] n=950 (h=549 b=401)✗; 2023:0.576 [0.547,0.604] n=1179 (h=679 b=500)✗; 2024:0.562 [0.535,0.588] n=1354 (h=761 b=593)✗; 2025:0.441 [0.420,0.461] n=2313 (h=1019 b=1294)✗; 2026:0.493 [0.465,0.521] n=1220 (h=601 b=619)✗ | False |
| EQL | 1h | long | 0.551 [0.539,0.563] · n=6643 (h=3661/b=2982, ambig=1794) · lift +4.4 pt | — | 2022:0.552 [0.524,0.579] n=1252 (h=691 b=561)✗; 2023:0.573 [0.543,0.603] n=1029 (h=590 b=439)✗; 2024:0.558 [0.534,0.582] n=1690 (h=943 b=747)✗; 2025:0.511 [0.488,0.533] n=1862 (h=951 b=911)✗; 2026:0.600 [0.566,0.633] n=810 (h=486 b=324)✗ | False |
| FVG | 1h | long | 0.551 [0.534,0.567] · n=3516 (h=1937/b=1579, ambig=738) · lift +4.4 pt | 0.657 [0.634,0.680] · n=1582 (h=1040/b=542, ambig=758) · lift +15.1 pt | 2022:0.507 [0.450,0.563] n=300 (h=152 b=148)✗; 2023:0.541 [0.510,0.571] n=1017 (h=550 b=467)✗; 2024:0.419 [0.363,0.476] n=289 (h=121 b=168)✗; 2025:0.537 [0.495,0.579] n=540 (h=290 b=250)✗; 2026:0.602 [0.575,0.627] n=1370 (h=824 b=546)✗ | False |
| FVG | 1h | short | 0.505 [0.487,0.524] · n=2873 (h=1452/b=1421, ambig=1040) · lift -0.1 pt | 0.325 [0.307,0.345] · n=2354 (h=766/b=1588, ambig=815) · lift -18.1 pt | 2022:0.507 [0.465,0.550] n=532 (h=270 b=262)✗; 2023:0.488 [0.455,0.522] n=846 (h=413 b=433)✗; 2024:0.474 [0.431,0.517] n=519 (h=246 b=273)✗; 2025:0.639 [0.601,0.675] n=645 (h=412 b=233)✗; 2026:0.335 [0.287,0.388] n=331 (h=111 b=220)✗ | False |
| IFVG | 1h | long | 0.502 [0.493,0.511] · n=11698 (h=5868/b=5830, ambig=3034) · lift -0.5 pt | 0.504 [0.492,0.515] · n=7006 (h=3528/b=3478, ambig=1868) · lift -0.3 pt | 2022:0.505 [0.482,0.527] n=1950 (h=984 b=966)✗; 2023:0.451 [0.432,0.469] n=2722 (h=1227 b=1495)✗; 2024:0.520 [0.502,0.538] n=2956 (h=1538 b=1418)✗; 2025:0.510 [0.492,0.528] n=2987 (h=1523 b=1464)✗; 2026:0.550 [0.521,0.580] n=1083 (h=596 b=487)✗ | False |
| IFVG | 1h | short | 0.477 [0.466,0.487] · n=8290 (h=3952/b=4338, ambig=2868) · lift -3.0 pt | 0.484 [0.472,0.496] · n=6666 (h=3224/b=3442, ambig=2142) · lift -2.3 pt | 2022:0.428 [0.400,0.457] n=1150 (h=492 b=658)✗; 2023:0.588 [0.560,0.616] n=1205 (h=709 b=496)✗; 2024:0.484 [0.461,0.508] n=1668 (h=808 b=860)✗; 2025:0.478 [0.461,0.495] n=3387 (h=1619 b=1768)✗; 2026:0.368 [0.337,0.401] n=880 (h=324 b=556)✗ | False |
| OB | 1h | long | 0.527 [0.520,0.533] · n=25402 (h=13378/b=12024, ambig=4894) · lift +2.0 pt | 0.525 [0.518,0.532] · n=19985 (h=10495/b=9490, ambig=6593) · lift +1.8 pt | 2022:0.532 [0.517,0.546] n=4748 (h=2524 b=2224); 2023:0.559 [0.543,0.574] n=3869 (h=2162 b=1707); 2024:0.502 [0.491,0.513] n=7907 (h=3970 b=3937)✗; 2025:0.532 [0.519,0.545] n=5926 (h=3154 b=2772); 2026:0.531 [0.513,0.549] n=2952 (h=1568 b=1384) | False |
| OB | 1h | short | 0.500 [0.494,0.506] · n=24003 (h=11999/b=12004, ambig=9450) · lift -0.7 pt | 0.530 [0.523,0.538] · n=16830 (h=8924/b=7906, ambig=5715) · lift +2.4 pt | 2022:0.571 [0.556,0.585] n=4639 (h=2648 b=1991); 2023:0.493 [0.481,0.505] n=6867 (h=3384 b=3483)✗; 2024:0.508 [0.495,0.521] n=5787 (h=2939 b=2848)✗; 2025:0.437 [0.423,0.451] n=4573 (h=1998 b=2575)✗; 2026:0.482 [0.461,0.503] n=2137 (h=1030 b=1107)✗ | False |
| SUPPLY | 1h | short | 0.453 [0.445,0.461] · n=14131 (h=6401/b=7730, ambig=5998) · lift -5.4 pt | — | 2022:0.421 [0.400,0.443] n=2098 (h=884 b=1214)✗; 2023:0.554 [0.536,0.573] n=2830 (h=1569 b=1261); 2024:0.424 [0.408,0.440] n=3821 (h=1620 b=2201)✗; 2025:0.421 [0.405,0.438] n=3301 (h=1391 b=1910)✗; 2026:0.450 [0.429,0.472] n=2081 (h=937 b=1144)✗ | False |

### Q2 variant `all` (context)

| kind | tf | side | era with-zone | era against-zone | by year (with-zone) | passes every year |
|---|---|---|---|---|---|---|
| DEMAND | 1h | long | 0.493 [0.487,0.500] · n=24580 (h=12130/b=12450, ambig=4970) · lift -1.3 pt | — | 2022:0.521 [0.506,0.536] n=4380 (h=2282 b=2098)✗; 2023:0.461 [0.448,0.473] n=6015 (h=2770 b=3245)✗; 2024:0.515 [0.501,0.529] n=4924 (h=2535 b=2389)✗; 2025:0.457 [0.443,0.470] n=5451 (h=2489 b=2962)✗; 2026:0.539 [0.523,0.555] n=3810 (h=2054 b=1756) | False |
| EQH | 1h | short | 0.500 [0.493,0.506] · n=21461 (h=10724/b=10737, ambig=7231) · lift -0.7 pt | — | 2022:0.545 [0.527,0.562] n=3140 (h=1711 b=1429); 2023:0.499 [0.483,0.514] n=4086 (h=2038 b=2048)✗; 2024:0.529 [0.514,0.545] n=3979 (h=2107 b=1872); 2025:0.453 [0.440,0.466] n=5379 (h=2436 b=2943)✗; 2026:0.499 [0.485,0.513] n=4877 (h=2432 b=2445)✗ | False |
| EQL | 1h | long | 0.492 [0.485,0.498] · n=25162 (h=12370/b=12792, ambig=6439) · lift -1.5 pt | — | 2022:0.538 [0.521,0.555] n=3231 (h=1739 b=1492); 2023:0.523 [0.507,0.538] n=4008 (h=2095 b=1913)✗; 2024:0.475 [0.462,0.488] n=5740 (h=2725 b=3015)✗; 2025:0.469 [0.456,0.482] n=5575 (h=2615 b=2960)✗; 2026:0.484 [0.472,0.496] n=6608 (h=3196 b=3412)✗ | False |
| FVG | 1h | long | 0.526 [0.512,0.540] · n=5092 (h=2679/b=2413, ambig=965) · lift +1.9 pt | 0.596 [0.576,0.616] · n=2347 (h=1399/b=948, ambig=952) · lift +8.9 pt | 2022:0.465 [0.424,0.507] n=548 (h=255 b=293)✗; 2023:0.557 [0.532,0.582] n=1488 (h=829 b=659)✗; 2024:0.406 [0.361,0.452] n=446 (h=181 b=265)✗; 2025:0.536 [0.497,0.574] n=629 (h=337 b=292)✗; 2026:0.544 [0.522,0.566] n=1981 (h=1077 b=904)✗ | False |
| FVG | 1h | short | 0.486 [0.471,0.502] · n=4004 (h=1947/b=2057, ambig=1302) · lift -2.0 pt | 0.363 [0.346,0.381] · n=2993 (h=1088/b=1905, ambig=920) · lift -14.3 pt | 2022:0.468 [0.434,0.503] n=792 (h=371 b=421)✗; 2023:0.463 [0.433,0.494] n=997 (h=462 b=535)✗; 2024:0.418 [0.388,0.449] n=1003 (h=419 b=584)✗; 2025:0.653 [0.619,0.685] n=806 (h=526 b=280)✗; 2026:0.416 [0.369,0.465] n=406 (h=169 b=237)✗ | False |
| IFVG | 1h | long | 0.479 [0.474,0.485] · n=27265 (h=13073/b=14192, ambig=6415) · lift -2.7 pt | 0.490 [0.482,0.498] · n=16992 (h=8327/b=8665, ambig=3976) · lift -1.7 pt | 2022:0.493 [0.478,0.509] n=3853 (h=1901 b=1952)✗; 2023:0.455 [0.442,0.468] n=5827 (h=2650 b=3177)✗; 2024:0.499 [0.488,0.510] n=7790 (h=3890 b=3900)✗; 2025:0.482 [0.469,0.494] n=5998 (h=2890 b=3108)✗; 2026:0.459 [0.443,0.475] n=3797 (h=1742 b=2055)✗ | False |
| IFVG | 1h | short | 0.491 [0.483,0.498] · n=17810 (h=8738/b=9072, ambig=4563) · lift -1.6 pt | 0.506 [0.498,0.513] · n=16631 (h=8412/b=8219, ambig=4913) · lift -0.1 pt | 2022:0.494 [0.477,0.511] n=3184 (h=1573 b=1611)✗; 2023:0.591 [0.574,0.607] n=3453 (h=2040 b=1413); 2024:0.499 [0.482,0.517] n=3129 (h=1562 b=1567)✗; 2025:0.453 [0.440,0.466] n=5701 (h=2582 b=3119)✗; 2026:0.419 [0.399,0.439] n=2343 (h=981 b=1362)✗ | False |
| OB | 1h | long | 0.515 [0.511,0.519] · n=56782 (h=29261/b=27521, ambig=10810) · lift +0.9 pt | 0.529 [0.525,0.533] · n=56696 (h=29976/b=26720, ambig=14405) · lift +2.2 pt | 2022:0.502 [0.492,0.511] n=10539 (h=5286 b=5253)✗; 2023:0.542 [0.532,0.551] n=10425 (h=5647 b=4778); 2024:0.499 [0.491,0.507] n=15763 (h=7869 b=7894)✗; 2025:0.507 [0.498,0.516] n=11457 (h=5810 b=5647)✗; 2026:0.541 [0.530,0.551] n=8598 (h=4649 b=3949) | False |
| OB | 1h | short | 0.509 [0.504,0.513] · n=53860 (h=27399/b=26461, ambig=17920) · lift +0.2 pt | 0.522 [0.517,0.527] · n=41077 (h=21438/b=19639, ambig=12645) · lift +1.5 pt | 2022:0.509 [0.499,0.518] n=9951 (h=5062 b=4889)✗; 2023:0.527 [0.518,0.535] n=13584 (h=7155 b=6429)✗; 2024:0.537 [0.528,0.545] n=13102 (h=7031 b=6071); 2025:0.420 [0.410,0.431] n=8708 (h=3662 b=5046)✗; 2026:0.527 [0.517,0.538] n=8515 (h=4489 b=4026) | False |
| OB | 4h | long | 0.523 [0.506,0.541] · n=3189 (h=1669/b=1520, ambig=742) · lift +1.7 pt | 0.431 [0.408,0.455] · n=1713 (h=739/b=974, ambig=252) · lift -7.5 pt | 2026:0.523 [0.506,0.541] n=3189 (h=1669 b=1520)✗ | False |
| SUPPLY | 1h | short | 0.483 [0.477,0.489] · n=27212 (h=13143/b=14069, ambig=10390) · lift -2.4 pt | — | 2022:0.483 [0.468,0.499] n=3981 (h=1924 b=2057)✗; 2023:0.571 [0.557,0.584] n=5471 (h=3122 b=2349); 2024:0.419 [0.407,0.431] n=6114 (h=2561 b=3553)✗; 2025:0.454 [0.441,0.466] n=6499 (h=2947 b=3552)✗; 2026:0.503 [0.489,0.517] n=5147 (h=2589 b=2558)✗ | False |

## Q3 — FIRST TOUCH vs RE-ENTRY (re-entry grader on own-TF bars)

| tf | year | fresh | tested-1 | tested-2 | stale (≥3) | measured / not measured |
|---|---|---|---|---|---|---|
| 1h | 2022 | 0.497 [0.457,0.537] · n=592 (h=294/b=298, ambig=90) · lift -1.0 pt | 0.458 [0.413,0.504] · n=452 (h=207/b=245, ambig=80) · lift -4.9 pt | 0.446 [0.399,0.494] · n=419 (h=187/b=232, ambig=82) · lift -6.0 pt | 0.485 [0.465,0.506] · n=2282 (h=1107/b=1175, ambig=402) · lift -2.2 pt | 4399 / 0 |
| 1h | 2023 | 0.532 [0.500,0.564] · n=931 (h=495/b=436, ambig=108) · lift +2.5 pt | 0.527 [0.487,0.567] · n=609 (h=321/b=288, ambig=128) · lift +2.0 pt | 0.519 [0.478,0.560] · n=564 (h=293/b=271, ambig=112) · lift +1.3 pt | 0.521 [0.502,0.539] · n=2854 (h=1486/b=1368, ambig=513) · lift +1.4 pt | 5819 / 0 |
| 1h | 2024 | 0.487 [0.457,0.518] · n=1042 (h=508/b=534, ambig=109) · lift -1.9 pt | 0.513 [0.475,0.551] · n=659 (h=338/b=321, ambig=111) · lift +0.6 pt | 0.493 [0.452,0.534] · n=560 (h=276/b=284, ambig=121) · lift -1.4 pt | 0.499 [0.481,0.518] · n=2725 (h=1361/b=1364, ambig=390) · lift -0.7 pt | 5717 / 0 |
| 1h | 2025 | 0.500 [0.469,0.531] · n=968 (h=484/b=484, ambig=110) · lift -0.7 pt | 0.506 [0.470,0.542] · n=743 (h=376/b=367, ambig=119) · lift -0.1 pt | 0.491 [0.451,0.532] · n=580 (h=285/b=295, ambig=96) · lift -1.5 pt | 0.485 [0.466,0.504] · n=2615 (h=1269/b=1346, ambig=384) · lift -2.1 pt | 5615 / 0 |
| 1h | 2026 | 0.467 [0.432,0.503] · n=756 (h=353/b=403, ambig=109) · lift -4.0 pt | 0.527 [0.489,0.565] · n=660 (h=348/b=312, ambig=124) · lift +2.1 pt | 0.535 [0.491,0.579] · n=482 (h=258/b=224, ambig=104) · lift +2.9 pt | 0.502 [0.482,0.523] · n=2258 (h=1134/b=1124, ambig=426) · lift -0.4 pt | 4919 / 0 |
| 1h | era | 0.498 [0.483,0.512] · n=4289 (h=2134/b=2155, ambig=526) · lift -0.9 pt | 0.509 [0.492,0.527] · n=3123 (h=1590/b=1533, ambig=562) · lift +0.2 pt | 0.499 [0.479,0.518] · n=2605 (h=1299/b=1306, ambig=515) · lift -0.8 pt | 0.499 [0.490,0.508] · n=12734 (h=6357/b=6377, ambig=2115) · lift -0.7 pt | 26469 / 0 |
| 4h | 2026 | 0.536 [0.477,0.595] · n=274 (h=147/b=127, ambig=46) · lift +3.0 pt | 0.513 [0.454,0.572] · n=271 (h=139/b=132, ambig=69) · lift +0.6 pt | 0.452 [0.350,0.559] · n=84 (h=38/b=46, ambig=13) · lift -5.4 pt | 0.526 [0.488,0.564] · n=656 (h=345/b=311, ambig=135) · lift +1.9 pt | 1548 / 0 |
| 4h | era | 0.536 [0.477,0.595] · n=274 (h=147/b=127, ambig=46) · lift +3.0 pt | 0.513 [0.454,0.572] · n=271 (h=139/b=132, ambig=69) · lift +0.6 pt | 0.452 [0.350,0.559] · n=84 (h=38/b=46, ambig=13) · lift -5.4 pt | 0.526 [0.488,0.564] · n=656 (h=345/b=311, ambig=135) · lift +1.9 pt | 1548 / 0 |
| 1d | 2026 | 0.267 [0.109,0.519] · n=15 (h=4/b=11, ambig=6) · lift -24.0 pt | 0.778 [0.453,0.937] · n=9 (h=7/b=2, ambig=4) · lift +27.1 pt | 0.533 [0.301,0.752] · n=15 (h=8/b=7, ambig=3) · lift +2.7 pt | 0.487 [0.339,0.638] · n=39 (h=19/b=20, ambig=9) · lift -2.0 pt | 100 / 0 |
| 1d | era | 0.267 [0.109,0.519] · n=15 (h=4/b=11, ambig=6) · lift -24.0 pt | 0.778 [0.453,0.937] · n=9 (h=7/b=2, ambig=4) · lift +27.1 pt | 0.533 [0.301,0.752] · n=15 (h=8/b=7, ambig=3) · lift +2.7 pt | 0.487 [0.339,0.638] · n=39 (h=19/b=20, ambig=9) · lift -2.0 pt | 100 / 0 |

## Q5 — PULLBACK ENTRY AT A 1h/4h ZONE (D-with & 4h-against, hold-trade)

### Q5 variant `own` (PRIMARY)

| scope | cell | hold rate · n · lift |
|---|---|---|
| era | with-zone · all sides · 1h+4h | 0.667 [0.496,0.802] · n=33 (h=22/b=11, ambig=20) · lift +16.0 pt |
| era | with-zone · long · 1h+4h | 0.680 [0.484,0.828] · n=25 (h=17/b=8, ambig=12) · lift +17.3 pt |
| era | with-zone · long · 1h | 0.680 [0.484,0.828] · n=25 (h=17/b=8, ambig=12) · lift +17.3 pt |
| era | with-zone · short · 1h+4h | 0.625 [0.306,0.863] · n=8 (h=5/b=3, ambig=8) · lift +11.8 pt |
| era | with-zone · short · 1h | 0.600 [0.231,0.882] · n=5 (h=3/b=2, ambig=6) · lift +9.3 pt |
| era | with-zone · short · 4h | 0.667 [0.208,0.939] · n=3 (h=2/b=1, ambig=2) · lift +16.0 pt |
| era | against-zone · all sides · 1h+4h | 0.607 [0.424,0.764] · n=28 (h=17/b=11, ambig=18) · lift +10.0 pt |
| era | against-zone · long · 1h+4h | 0.500 [0.280,0.720] · n=16 (h=8/b=8, ambig=12) · lift -0.7 pt |
| era | against-zone · long · 1h | 0.500 [0.280,0.720] · n=16 (h=8/b=8, ambig=12) · lift -0.7 pt |
| era | against-zone · short · 1h+4h | 0.750 [0.468,0.911] · n=12 (h=9/b=3, ambig=6) · lift +24.3 pt |
| era | against-zone · short · 1h | 0.700 [0.397,0.892] · n=10 (h=7/b=3, ambig=6) · lift +19.3 pt |
| era | against-zone · short · 4h | 1.000 [0.342,1.000] · n=2 (h=2/b=0, ambig=0) · lift +49.3 pt |
| 2025 | with-zone · all sides · 1h+4h | 0.667 [0.467,0.820] · n=24 (h=16/b=8, ambig=12) · lift +16.0 pt |
| 2025 | with-zone · long · 1h+4h | 0.696 [0.491,0.844] · n=23 (h=16/b=7, ambig=12) · lift +18.9 pt |
| 2025 | with-zone · long · 1h | 0.696 [0.491,0.844] · n=23 (h=16/b=7, ambig=12) · lift +18.9 pt |
| 2025 | with-zone · short · 1h+4h | 0.000 [0.000,0.793] · n=1 (h=0/b=1, ambig=0) · lift -50.7 pt |
| 2025 | with-zone · short · 1h | 0.000 [0.000,0.793] · n=1 (h=0/b=1, ambig=0) · lift -50.7 pt |
| 2025 | against-zone · all sides · 1h+4h | 0.471 [0.262,0.690] · n=17 (h=8/b=9, ambig=11) · lift -3.6 pt |
| 2025 | against-zone · long · 1h+4h | 0.500 [0.280,0.720] · n=16 (h=8/b=8, ambig=11) · lift -0.7 pt |
| 2025 | against-zone · long · 1h | 0.500 [0.280,0.720] · n=16 (h=8/b=8, ambig=11) · lift -0.7 pt |
| 2025 | against-zone · short · 1h+4h | 0.000 [0.000,0.793] · n=1 (h=0/b=1, ambig=0) · lift -50.7 pt |
| 2025 | against-zone · short · 1h | 0.000 [0.000,0.793] · n=1 (h=0/b=1, ambig=0) · lift -50.7 pt |
| 2026 | with-zone · all sides · 1h+4h | 0.667 [0.354,0.879] · n=9 (h=6/b=3, ambig=8) · lift +16.0 pt |
| 2026 | with-zone · long · 1h+4h | 0.500 [0.095,0.905] · n=2 (h=1/b=1, ambig=0) · lift -0.7 pt |
| 2026 | with-zone · long · 1h | 0.500 [0.095,0.905] · n=2 (h=1/b=1, ambig=0) · lift -0.7 pt |
| 2026 | with-zone · short · 1h+4h | 0.714 [0.359,0.918] · n=7 (h=5/b=2, ambig=8) · lift +20.8 pt |
| 2026 | with-zone · short · 1h | 0.750 [0.301,0.954] · n=4 (h=3/b=1, ambig=6) · lift +24.3 pt |
| 2026 | with-zone · short · 4h | 0.667 [0.208,0.939] · n=3 (h=2/b=1, ambig=2) · lift +16.0 pt |
| 2026 | against-zone · all sides · 1h+4h | 0.818 [0.523,0.949] · n=11 (h=9/b=2, ambig=7) · lift +31.1 pt |
| 2026 | against-zone · long · 1h+4h | NOT MEASURED (n=0) |
| 2026 | against-zone · long · 1h | NOT MEASURED (n=0) |
| 2026 | against-zone · short · 1h+4h | 0.818 [0.523,0.949] · n=11 (h=9/b=2, ambig=6) · lift +31.1 pt |
| 2026 | against-zone · short · 1h | 0.778 [0.453,0.937] · n=9 (h=7/b=2, ambig=6) · lift +27.1 pt |
| 2026 | against-zone · short · 4h | 1.000 [0.342,1.000] · n=2 (h=2/b=0, ambig=0) · lift +49.3 pt |

### Q5 variant `noconflict` (context)

| scope | cell | hold rate · n · lift |
|---|---|---|
| era | with-zone · all sides · 1h+4h | 0.586 [0.537,0.634] · n=394 (h=231/b=163, ambig=108) · lift +8.0 pt |
| era | with-zone · long · 1h+4h | 0.589 [0.536,0.640] · n=341 (h=201/b=140, ambig=92) · lift +8.3 pt |
| era | with-zone · long · 1h | 0.589 [0.536,0.640] · n=341 (h=201/b=140, ambig=92) · lift +8.3 pt |
| era | with-zone · short · 1h+4h | 0.566 [0.433,0.691] · n=53 (h=30/b=23, ambig=16) · lift +5.9 pt |
| era | with-zone · short · 1h | 0.566 [0.433,0.691] · n=53 (h=30/b=23, ambig=16) · lift +5.9 pt |
| era | against-zone · all sides · 1h+4h | 0.697 [0.654,0.737] · n=472 (h=329/b=143, ambig=252) · lift +19.0 pt |
| era | against-zone · long · 1h+4h | 0.728 [0.683,0.769] · n=405 (h=295/b=110, ambig=176) · lift +22.2 pt |
| era | against-zone · long · 1h | 0.728 [0.683,0.769] · n=405 (h=295/b=110, ambig=176) · lift +22.2 pt |
| era | against-zone · short · 1h+4h | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=76) · lift +0.1 pt |
| era | against-zone · short · 1h | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=76) · lift +0.1 pt |
| 2025 | with-zone · all sides · 1h+4h | 0.568 [0.514,0.621] · n=329 (h=187/b=142, ambig=106) · lift +6.2 pt |
| 2025 | with-zone · long · 1h+4h | 0.569 [0.510,0.626] · n=276 (h=157/b=119, ambig=90) · lift +6.2 pt |
| 2025 | with-zone · long · 1h | 0.569 [0.510,0.626] · n=276 (h=157/b=119, ambig=90) · lift +6.2 pt |
| 2025 | with-zone · short · 1h+4h | 0.566 [0.433,0.691] · n=53 (h=30/b=23, ambig=16) · lift +5.9 pt |
| 2025 | with-zone · short · 1h | 0.566 [0.433,0.691] · n=53 (h=30/b=23, ambig=16) · lift +5.9 pt |
| 2025 | against-zone · all sides · 1h+4h | 0.566 [0.503,0.627] · n=242 (h=137/b=105, ambig=126) · lift +5.9 pt |
| 2025 | against-zone · long · 1h+4h | 0.589 [0.514,0.659] · n=175 (h=103/b=72, ambig=50) · lift +8.2 pt |
| 2025 | against-zone · long · 1h | 0.589 [0.514,0.659] · n=175 (h=103/b=72, ambig=50) · lift +8.2 pt |
| 2025 | against-zone · short · 1h+4h | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=76) · lift +0.1 pt |
| 2025 | against-zone · short · 1h | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=76) · lift +0.1 pt |
| 2026 | with-zone · all sides · 1h+4h | 0.677 [0.556,0.778] · n=65 (h=44/b=21, ambig=2) · lift +17.0 pt |
| 2026 | with-zone · long · 1h+4h | 0.677 [0.556,0.778] · n=65 (h=44/b=21, ambig=2) · lift +17.0 pt |
| 2026 | with-zone · long · 1h | 0.677 [0.556,0.778] · n=65 (h=44/b=21, ambig=2) · lift +17.0 pt |
| 2026 | against-zone · all sides · 1h+4h | 0.835 [0.781,0.877] · n=230 (h=192/b=38, ambig=126) · lift +32.8 pt |
| 2026 | against-zone · long · 1h+4h | 0.835 [0.781,0.877] · n=230 (h=192/b=38, ambig=126) · lift +32.8 pt |
| 2026 | against-zone · long · 1h | 0.835 [0.781,0.877] · n=230 (h=192/b=38, ambig=126) · lift +32.8 pt |

### Q5 variant `all` (context)

| scope | cell | hold rate · n · lift |
|---|---|---|
| era | with-zone · all sides · 1h+4h | 0.596 [0.565,0.626] · n=992 (h=591/b=401, ambig=452) · lift +8.9 pt |
| era | with-zone · long · 1h+4h | 0.599 [0.564,0.632] · n=785 (h=470/b=315, ambig=373) · lift +9.2 pt |
| era | with-zone · long · 1h | 0.599 [0.564,0.632] · n=785 (h=470/b=315, ambig=373) · lift +9.2 pt |
| era | with-zone · short · 1h+4h | 0.585 [0.516,0.649] · n=207 (h=121/b=86, ambig=79) · lift +7.8 pt |
| era | with-zone · short · 1h | 0.510 [0.432,0.587] · n=157 (h=80/b=77, ambig=66) · lift +0.3 pt |
| era | with-zone · short · 4h | 0.820 [0.692,0.902] · n=50 (h=41/b=9, ambig=13) · lift +31.3 pt |
| era | against-zone · all sides · 1h+4h | 0.579 [0.551,0.607] · n=1209 (h=700/b=509, ambig=722) · lift +7.2 pt |
| era | against-zone · long · 1h+4h | 0.640 [0.604,0.674] · n=725 (h=464/b=261, ambig=459) · lift +13.3 pt |
| era | against-zone · long · 1h | 0.640 [0.604,0.674] · n=725 (h=464/b=261, ambig=459) · lift +13.3 pt |
| era | against-zone · short · 1h+4h | 0.488 [0.443,0.532] · n=484 (h=236/b=248, ambig=263) · lift -1.9 pt |
| era | against-zone · short · 1h | 0.427 [0.382,0.474] · n=433 (h=185/b=248, ambig=235) · lift -7.9 pt |
| era | against-zone · short · 4h | 1.000 [0.930,1.000] · n=51 (h=51/b=0, ambig=28) · lift +49.3 pt |
| 2025 | with-zone · all sides · 1h+4h | 0.544 [0.505,0.582] · n=638 (h=347/b=291, ambig=383) · lift +3.7 pt |
| 2025 | with-zone · long · 1h+4h | 0.542 [0.501,0.582] · n=585 (h=317/b=268, ambig=361) · lift +3.5 pt |
| 2025 | with-zone · long · 1h | 0.542 [0.501,0.582] · n=585 (h=317/b=268, ambig=361) · lift +3.5 pt |
| 2025 | with-zone · short · 1h+4h | 0.566 [0.433,0.691] · n=53 (h=30/b=23, ambig=22) · lift +5.9 pt |
| 2025 | with-zone · short · 1h | 0.566 [0.433,0.691] · n=53 (h=30/b=23, ambig=22) · lift +5.9 pt |
| 2025 | against-zone · all sides · 1h+4h | 0.605 [0.561,0.646] · n=506 (h=306/b=200, ambig=392) · lift +9.8 pt |
| 2025 | against-zone · long · 1h+4h | 0.620 [0.573,0.664] · n=439 (h=272/b=167, ambig=316) · lift +11.3 pt |
| 2025 | against-zone · long · 1h | 0.620 [0.573,0.664] · n=439 (h=272/b=167, ambig=316) · lift +11.3 pt |
| 2025 | against-zone · short · 1h+4h | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=76) · lift +0.1 pt |
| 2025 | against-zone · short · 1h | 0.507 [0.391,0.624] · n=67 (h=34/b=33, ambig=76) · lift +0.1 pt |
| 2026 | with-zone · all sides · 1h+4h | 0.689 [0.639,0.735] · n=354 (h=244/b=110, ambig=69) · lift +18.3 pt |
| 2026 | with-zone · long · 1h+4h | 0.765 [0.702,0.818] · n=200 (h=153/b=47, ambig=12) · lift +25.8 pt |
| 2026 | with-zone · long · 1h | 0.765 [0.702,0.818] · n=200 (h=153/b=47, ambig=12) · lift +25.8 pt |
| 2026 | with-zone · short · 1h+4h | 0.591 [0.512,0.665] · n=154 (h=91/b=63, ambig=57) · lift +8.4 pt |
| 2026 | with-zone · short · 1h | 0.481 [0.387,0.576] · n=104 (h=50/b=54, ambig=44) · lift -2.6 pt |
| 2026 | with-zone · short · 4h | 0.820 [0.692,0.902] · n=50 (h=41/b=9, ambig=13) · lift +31.3 pt |
| 2026 | against-zone · all sides · 1h+4h | 0.560 [0.523,0.597] · n=703 (h=394/b=309, ambig=330) · lift +5.4 pt |
| 2026 | against-zone · long · 1h+4h | 0.671 [0.615,0.723] · n=286 (h=192/b=94, ambig=143) · lift +16.5 pt |
| 2026 | against-zone · long · 1h | 0.671 [0.615,0.723] · n=286 (h=192/b=94, ambig=143) · lift +16.5 pt |
| 2026 | against-zone · short · 1h+4h | 0.484 [0.437,0.532] · n=417 (h=202/b=215, ambig=187) · lift -2.2 pt |
| 2026 | against-zone · short · 1h | 0.413 [0.363,0.464] · n=366 (h=151/b=215, ambig=159) · lift -9.4 pt |
| 2026 | against-zone · short · 4h | 1.000 [0.930,1.000] · n=51 (h=51/b=0, ambig=28) · lift +49.3 pt |

### Q5 reference — the unrestricted mixed cells under BOTH labellings (R23 S4d(2) comparability)

| scope | R24 hold-trade cell | R23 approach label | long | short |
|---|---|---|---|---|
| era | D-with/4h-against | D-against/4h-with | 0.550 [0.527,0.572] · n=1965 (h=1080/b=885, ambig=962) · lift +4.3 pt | 0.569 [0.537,0.601] · n=945 (h=538/b=407, ambig=422) · lift +6.3 pt |
| era | D-against/4h-with | D-with/4h-against | 0.809 [0.784,0.832] · n=1029 (h=833/b=196, ambig=147) · lift +30.3 pt | 0.425 [0.409,0.442] · n=3431 (h=1460/b=1971, ambig=603) · lift -8.1 pt |
| 2025 | D-with/4h-against | D-against/4h-with | 0.510 [0.483,0.538] · n=1240 (h=633/b=607, ambig=716) · lift +0.4 pt | 0.655 [0.606,0.702] · n=374 (h=245/b=129, ambig=178) · lift +14.8 pt |
| 2025 | D-against/4h-with | D-with/4h-against | 0.843 [0.805,0.874] · n=426 (h=359/b=67, ambig=54) · lift +33.6 pt | 0.362 [0.341,0.383] · n=2085 (h=754/b=1331, ambig=267) · lift -14.5 pt |
| 2026 | D-with/4h-against | D-against/4h-with | 0.617 [0.581,0.651] · n=725 (h=447/b=278, ambig=246) · lift +11.0 pt | 0.513 [0.472,0.554] · n=571 (h=293/b=278, ambig=244) · lift +0.6 pt |
| 2026 | D-against/4h-with | D-with/4h-against | 0.786 [0.752,0.817] · n=603 (h=474/b=129, ambig=93) · lift +27.9 pt | 0.524 [0.498,0.551] · n=1346 (h=706/b=640, ambig=336) · lift +1.8 pt |

## APPENDIX — sample ids (first 5 per cell: day session kind tf @price ordN)

- **Q1/own/era/(a) with-zone**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1
- **Q1/own/era/(a) against-zone**: 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1
- **Q1/own/era/(a) unknown-zone**: 2022-04-18 ASIA FVG 1h @14066.625 ord1; 2022-04-19 LONDON FVG 1h @14066.625 ord1; 2022-04-19 NY FVG 1h @14066.625 ord1; 2022-04-20 NY FVG 1h @14204.625 ord1; 2022-04-27 NY FVG 1h @13110.25 ord1
- **Q1/own/era/(b) with-zone × 4h-with**: 2025-05-22 LONDON IFVG 1h @21181.375 ord1; 2025-05-22 NY EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @21208.5 ord1; 2025-05-24 ASIA EQH 1h @21231.5 ord1
- **Q1/own/era/(b) with-zone × 4h-against**: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q1/own/era/(b) with-zone × 4h-range**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1
- **Q1/own/era/(b) against-zone × 4h-with**: 2025-05-22 LONDON EQL 1h @21154 ord1; 2025-05-22 LONDON EQL 1h @21192.25 ord1; 2025-05-22 LONDON EQL 1h @21229.5 ord1; 2025-05-22 LONDON DEMAND 1h @21168.625 ord1; 2025-05-22 LONDON OB 1h @21153.25 ord1
- **Q1/own/era/(b) against-zone × 4h-against**: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q1/own/era/(b) against-zone × 4h-range**: 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1
- **Q1/own/era/(c) with-zone × D-with**: 2022-04-26 LONDON OB 1h @13155.625 ord1; 2022-04-26 LONDON OB 1h @13155.625 ord1; 2022-04-26 NY OB 1h @13155.625 ord1; 2022-04-26 NY OB 1h @13155.625 ord1; 2022-04-27 NY EQH 1h @13347.25 ord1
- **Q1/own/era/(c) with-zone × D-against**: 2022-04-27 NY DEMAND 1h @13050.875 ord1; 2022-04-27 NY IFVG 1h @13066.25 ord1; 2022-04-27 NY OB 1h @13080.125 ord1; 2022-04-27 ASIA DEMAND 1h @13123.375 ord1; 2022-04-28 LONDON IFVG 1h @13360.875 ord1
- **Q1/own/era/(c) with-zone × D-range**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1
- **Q1/own/era/(c) against-zone × D-with**: 2022-04-26 ASIA EQL 1h @13183.75 ord1; 2022-04-27 NY IFVG 1h @13213.625 ord1; 2022-04-27 ASIA IFVG 1h @13360.875 ord1; 2022-04-27 ASIA FVG 1h @13277.875 ord1; 2022-04-28 NY IFVG 1h @13360.875 ord1
- **Q1/own/era/(c) against-zone × D-against**: 2022-04-27 NY SUPPLY 1h @13145.125 ord1; 2022-04-27 NY OB 1h @13155.625 ord1; 2022-04-27 NY OB 1h @13155.625 ord1; 2022-04-27 NY OB 1h @13123.375 ord1; 2022-04-27 NY OB 1h @13135.125 ord1
- **Q1/own/era/(c) against-zone × D-range**: 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1
- **Q1/own/era/4-way with-zone × 4h-with × D-with**: 2025-07-21 NY FVG 1h @23176 ord1; 2025-07-21 NY OB 1h @23146.875 ord1; 2025-07-21 NY OB 1h @23125.25 ord1; 2025-07-21 NY OB 1h @23260 ord1; 2025-07-28 NY EQL 1h @23437 ord1
- **Q1/own/era/4-way with-zone × 4h-with × D-against**: 2025-05-22 LONDON IFVG 1h @21181.375 ord1; 2025-05-22 NY EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @21208.5 ord1; 2025-05-24 ASIA EQH 1h @21231.5 ord1
- **Q1/own/era/4-way with-zone × 4h-against × D-with**: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q1/own/era/4-way with-zone × 4h-against × D-against**: 2025-05-27 ASIA EQH 1h @21564.75 ord1; 2025-05-27 ASIA SUPPLY 1h @21533.75 ord1; 2025-07-20 LONDON EQH 1h @23281 ord1; 2025-07-20 NY EQH 1h @23281 ord1; 2025-07-20 NY EQH 1h @23288.5 ord1
- **Q1/own/era/4-way against-zone × 4h-with × D-with**: 2025-05-27 ASIA EQH 1h @21513 ord1; 2025-05-27 ASIA OB 1h @21491.25 ord1; 2025-05-27 ASIA OB 1h @21491.25 ord1; 2025-07-19 ASIA IFVG 1h @23221.75 ord1; 2025-07-20 LONDON OB 1h @23263.375 ord1
- **Q1/own/era/4-way against-zone × 4h-with × D-against**: 2025-05-22 LONDON EQL 1h @21154 ord1; 2025-05-22 LONDON EQL 1h @21192.25 ord1; 2025-05-22 LONDON EQL 1h @21229.5 ord1; 2025-05-22 LONDON DEMAND 1h @21168.625 ord1; 2025-05-22 LONDON OB 1h @21153.25 ord1
- **Q1/own/era/4-way against-zone × 4h-against × D-with**: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q1/own/era/4-way against-zone × 4h-against × D-against**: 2025-07-21 ASIA EQL 1h @23234.25 ord1; 2025-07-21 ASIA EQL 1h @23260.5 ord1; 2025-07-21 ASIA OB 1h @23260 ord1; 2025-07-21 ASIA OB 1h @23230.875 ord1; 2026-06-02 ASIA EQL 1h @30790 ord1
- **Q1/own/era/side long × with-zone**: 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1; 2022-04-12 LONDON OB 1h @14025.375 ord1; 2022-04-13 NY DEMAND 1h @14016.25 ord1; 2022-04-13 NY OB 1h @14025.375 ord1
- **Q1/own/era/side long × against-zone**: 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-13 LONDON SUPPLY 1h @14190.75 ord1; 2022-04-13 LONDON OB 1h @14247.25 ord1
- **Q1/own/era/side short × with-zone**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-12 LONDON IFVG 1h @14073.75 ord1; 2022-04-12 NY IFVG 1h @14073.75 ord1
- **Q1/own/era/side short × against-zone**: 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1; 2022-04-17 NY OB 1h @13958.75 ord1; 2022-04-18 NY EQL 1h @13902.5 ord1; 2022-04-18 NY DEMAND 1h @14016.25 ord1
- **Q1/own/2022/(a) with-zone**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1
- **Q1/own/2022/(a) against-zone**: 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1
- **Q1/own/2022/(a) unknown-zone**: 2022-04-18 ASIA FVG 1h @14066.625 ord1; 2022-04-19 LONDON FVG 1h @14066.625 ord1; 2022-04-19 NY FVG 1h @14066.625 ord1; 2022-04-20 NY FVG 1h @14204.625 ord1; 2022-04-27 NY FVG 1h @13110.25 ord1
- **Q1/own/2022/(c) with-zone × D-with**: 2022-04-26 LONDON OB 1h @13155.625 ord1; 2022-04-26 LONDON OB 1h @13155.625 ord1; 2022-04-26 NY OB 1h @13155.625 ord1; 2022-04-26 NY OB 1h @13155.625 ord1; 2022-04-27 NY EQH 1h @13347.25 ord1
- **Q1/own/2022/(c) with-zone × D-against**: 2022-04-27 NY DEMAND 1h @13050.875 ord1; 2022-04-27 NY IFVG 1h @13066.25 ord1; 2022-04-27 NY OB 1h @13080.125 ord1; 2022-04-27 ASIA DEMAND 1h @13123.375 ord1; 2022-04-28 LONDON IFVG 1h @13360.875 ord1
- **Q1/own/2022/(c) against-zone × D-with**: 2022-04-26 ASIA EQL 1h @13183.75 ord1; 2022-04-27 NY IFVG 1h @13213.625 ord1; 2022-04-27 ASIA IFVG 1h @13360.875 ord1; 2022-04-27 ASIA FVG 1h @13277.875 ord1; 2022-04-28 NY IFVG 1h @13360.875 ord1
- **Q1/own/2022/(c) against-zone × D-against**: 2022-04-27 NY SUPPLY 1h @13145.125 ord1; 2022-04-27 NY OB 1h @13155.625 ord1; 2022-04-27 NY OB 1h @13155.625 ord1; 2022-04-27 NY OB 1h @13123.375 ord1; 2022-04-27 NY OB 1h @13135.125 ord1
- **Q1/own/2023/(a) with-zone**: 2023-01-01 ASIA IFVG 1h @11061 ord1; 2023-01-01 ASIA FVG 1h @10964 ord1; 2023-01-01 ASIA OB 1h @10991.25 ord1; 2023-01-02 LONDON EQH 1h @11080 ord1; 2023-01-02 LONDON FVG 1h @11157.5 ord1
- **Q1/own/2023/(a) against-zone**: 2023-01-01 ASIA EQH 1h @11071.5 ord1; 2023-01-01 ASIA EQH 1h @11080 ord1; 2023-01-01 ASIA IFVG 1h @10953 ord1; 2023-01-01 ASIA OB 1h @11079 ord1; 2023-01-02 LONDON IFVG 1h @11151.375 ord1
- **Q1/own/2023/(a) unknown-zone**: 2023-01-05 LONDON FVG 1h @10966.75 ord1; 2023-01-05 NY FVG 1h @10966.75 ord1; 2023-01-11 NY FVG 1h @11448.125 ord1; 2023-02-19 ASIA FVG 1h @12336.625 ord1; 2023-02-22 NY FVG 1h @12128.25 ord1
- **Q1/own/2023/(c) with-zone × D-with**: 2023-01-09 LONDON EQH 1h @11170.75 ord1; 2023-01-09 LONDON EQH 1h @11181.75 ord1; 2023-01-09 LONDON EQH 1h @11188 ord1; 2023-01-09 NY EQH 1h @11170.75 ord1; 2023-01-09 NY EQH 1h @11181.75 ord1
- **Q1/own/2023/(c) with-zone × D-against**: 2023-01-09 LONDON EQL 1h @11117.5 ord1; 2023-01-09 LONDON EQL 1h @11158.5 ord1; 2023-01-09 LONDON IFVG 1h @11151.375 ord1; 2023-01-09 LONDON IFVG 1h @11157.5 ord1; 2023-01-09 LONDON OB 1h @11155.125 ord1
- **Q1/own/2023/(c) against-zone × D-with**: 2023-01-09 LONDON EQL 1h @11180.5 ord1; 2023-01-09 LONDON DEMAND 1h @11168.75 ord1; 2023-01-09 NY EQL 1h @11158.5 ord1; 2023-01-09 NY EQL 1h @11180.5 ord1; 2023-01-09 NY DEMAND 1h @11227.25 ord1
- **Q1/own/2023/(c) against-zone × D-against**: 2023-01-09 LONDON OB 1h @11157.125 ord1; 2023-01-09 LONDON OB 1h @11094.5 ord1; 2023-01-09 LONDON OB 1h @11094.5 ord1; 2023-01-09 NY OB 1h @11145.375 ord1; 2023-01-09 ASIA EQH 1h @11255.25 ord1
- **Q1/own/2024/(a) with-zone**: 2024-01-01 LONDON EQL 1h @16896 ord1; 2024-01-01 LONDON EQL 1h @16911 ord1; 2024-01-01 LONDON EQL 1h @16961.5 ord1; 2024-01-01 LONDON EQL 1h @16985 ord1; 2024-01-01 LONDON DEMAND 1h @16876.625 ord1
- **Q1/own/2024/(a) against-zone**: 2024-01-01 LONDON EQH 1h @16887.25 ord1; 2024-01-01 LONDON EQH 1h @16974.25 ord1; 2024-01-01 LONDON EQH 1h @17012.75 ord1; 2024-01-01 LONDON SUPPLY 1h @16862.625 ord1; 2024-01-01 LONDON SUPPLY 1h @16891.125 ord1
- **Q1/own/2024/(a) unknown-zone**: 2024-01-01 LONDON FVG 1h @16970.75 ord1; 2024-01-01 NY FVG 1h @16643.5 ord1; 2024-01-02 NY FVG 1h @16643.5 ord1; 2024-01-09 NY FVG 1h @16923.25 ord1; 2024-01-10 NY FVG 1h @16783 ord1
- **Q1/own/2024/(c) with-zone × D-with**: 2024-02-01 LONDON EQL 1h @17501 ord1; 2024-02-01 LONDON EQL 1h @17518 ord1; 2024-02-01 LONDON EQL 1h @17540.25 ord1; 2024-02-01 LONDON IFVG 1h @17519.125 ord1; 2024-02-01 LONDON OB 1h @17556.875 ord1
- **Q1/own/2024/(c) with-zone × D-against**: 2024-02-01 NY EQH 1h @17591 ord1; 2024-02-01 NY EQH 1h @17636 ord1; 2024-02-01 NY EQH 1h @17651 ord1; 2024-02-01 NY SUPPLY 1h @17538.25 ord1; 2024-02-01 NY SUPPLY 1h @17718 ord1
- **Q1/own/2024/(c) against-zone × D-with**: 2024-02-01 LONDON EQH 1h @17490.75 ord1; 2024-02-01 LONDON EQH 1h @17591 ord1; 2024-02-01 LONDON SUPPLY 1h @17538.25 ord1; 2024-02-01 LONDON SUPPLY 1h @17578.25 ord1; 2024-02-01 LONDON OB 1h @17538.25 ord1
- **Q1/own/2024/(c) against-zone × D-against**: 2024-02-01 NY EQL 1h @17501 ord1; 2024-02-01 NY EQL 1h @17518 ord1; 2024-02-01 NY EQL 1h @17540.25 ord1; 2024-02-01 NY DEMAND 1h @17666.375 ord1; 2024-02-01 NY IFVG 1h @17519.125 ord1
- **Q1/own/2025/(a) with-zone**: 2025-01-01 NY EQL 1h @21310.25 ord1; 2025-01-01 NY DEMAND 1h @21077.625 ord1; 2025-01-01 NY DEMAND 1h @21234.75 ord1; 2025-01-01 NY IFVG 1h @21406.5 ord1; 2025-01-01 NY OB 1h @21098.625 ord1
- **Q1/own/2025/(a) against-zone**: 2025-01-01 LONDON EQL 1h @21428 ord1; 2025-01-01 NY EQL 1h @21428 ord1; 2025-01-02 NY EQL 1h @21310.25 ord1; 2025-01-02 NY EQL 1h @21428 ord1; 2025-01-02 NY DEMAND 1h @21382.875 ord1
- **Q1/own/2025/(a) unknown-zone**: 2025-01-14 NY FVG 1h @21227 ord1; 2025-01-16 LONDON FVG 1h @21634 ord1; 2025-01-16 NY FVG 1h @21634 ord1; 2025-01-19 LONDON FVG 1h @21634 ord1; 2025-01-19 NY FVG 1h @21634 ord1
- **Q1/own/2025/(b) with-zone × 4h-with**: 2025-05-22 LONDON IFVG 1h @21181.375 ord1; 2025-05-22 NY EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @21208.5 ord1; 2025-05-24 ASIA EQH 1h @21231.5 ord1
- **Q1/own/2025/(b) with-zone × 4h-against**: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q1/own/2025/(b) against-zone × 4h-with**: 2025-05-22 LONDON EQL 1h @21154 ord1; 2025-05-22 LONDON EQL 1h @21192.25 ord1; 2025-05-22 LONDON EQL 1h @21229.5 ord1; 2025-05-22 LONDON DEMAND 1h @21168.625 ord1; 2025-05-22 LONDON OB 1h @21153.25 ord1
- **Q1/own/2025/(b) against-zone × 4h-against**: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q1/own/2025/(c) with-zone × D-with**: 2025-01-07 LONDON IFVG 1h @21406.5 ord1; 2025-01-07 LONDON IFVG 1h @21440.75 ord1; 2025-01-07 LONDON OB 1h @21420.25 ord1; 2025-01-07 NY IFVG 1h @21406.5 ord1; 2025-01-07 NY OB 1h @21420.25 ord1
- **Q1/own/2025/(c) with-zone × D-against**: 2025-01-07 LONDON EQL 1h @21310.25 ord1; 2025-01-07 LONDON DEMAND 1h @21234.75 ord1; 2025-01-07 LONDON DEMAND 1h @21382.875 ord1; 2025-01-07 LONDON DEMAND 1h @21240.75 ord1; 2025-01-07 LONDON OB 1h @21382.875 ord1
- **Q1/own/2025/(c) against-zone × D-with**: 2025-01-07 NY EQL 1h @21360 ord1; 2025-01-07 NY EQL 1h @21384.25 ord1; 2025-01-07 NY EQL 1h @21428 ord1; 2025-01-08 LONDON EQL 1h @21271.25 ord1; 2025-01-08 LONDON EQL 1h @21310.25 ord1
- **Q1/own/2025/(c) against-zone × D-against**: 2025-01-07 LONDON SUPPLY 1h @21328.625 ord1; 2025-01-07 LONDON OB 1h @21271.5 ord1; 2025-01-07 NY SUPPLY 1h @21328.625 ord1; 2025-01-07 NY OB 1h @21271.5 ord1; 2025-01-07 ASIA SUPPLY 1h @21271.5 ord1
- **Q1/own/2025/4-way with-zone × 4h-with × D-with**: 2025-07-21 NY FVG 1h @23176 ord1; 2025-07-21 NY OB 1h @23146.875 ord1; 2025-07-21 NY OB 1h @23125.25 ord1; 2025-07-21 NY OB 1h @23260 ord1; 2025-07-28 NY EQL 1h @23437 ord1
- **Q1/own/2025/4-way with-zone × 4h-with × D-against**: 2025-05-22 LONDON IFVG 1h @21181.375 ord1; 2025-05-22 NY EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @20996 ord1; 2025-05-24 ASIA EQH 1h @21208.5 ord1; 2025-05-24 ASIA EQH 1h @21231.5 ord1
- **Q1/own/2025/4-way with-zone × 4h-against × D-with**: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q1/own/2025/4-way with-zone × 4h-against × D-against**: 2025-05-27 ASIA EQH 1h @21564.75 ord1; 2025-05-27 ASIA SUPPLY 1h @21533.75 ord1; 2025-07-20 LONDON EQH 1h @23281 ord1; 2025-07-20 NY EQH 1h @23281 ord1; 2025-07-20 NY EQH 1h @23288.5 ord1
- **Q1/own/2025/4-way against-zone × 4h-with × D-with**: 2025-05-27 ASIA EQH 1h @21513 ord1; 2025-05-27 ASIA OB 1h @21491.25 ord1; 2025-05-27 ASIA OB 1h @21491.25 ord1; 2025-07-19 ASIA IFVG 1h @23221.75 ord1; 2025-07-20 LONDON OB 1h @23263.375 ord1
- **Q1/own/2025/4-way against-zone × 4h-with × D-against**: 2025-05-22 LONDON EQL 1h @21154 ord1; 2025-05-22 LONDON EQL 1h @21192.25 ord1; 2025-05-22 LONDON EQL 1h @21229.5 ord1; 2025-05-22 LONDON DEMAND 1h @21168.625 ord1; 2025-05-22 LONDON OB 1h @21153.25 ord1
- **Q1/own/2025/4-way against-zone × 4h-against × D-with**: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q1/own/2025/4-way against-zone × 4h-against × D-against**: 2025-07-21 ASIA EQL 1h @23234.25 ord1; 2025-07-21 ASIA EQL 1h @23260.5 ord1; 2025-07-21 ASIA OB 1h @23260 ord1; 2025-07-21 ASIA OB 1h @23230.875 ord1
- **Q1/own/2026/(a) with-zone**: 2026-01-01 LONDON EQH 1h @25728.75 ord1; 2026-01-01 LONDON OB 1h @25668.75 ord1; 2026-01-01 NY EQH 1h @25728.75 ord1; 2026-01-01 NY EQH 1h @25755 ord1; 2026-01-01 NY EQH 1h @25795 ord1
- **Q1/own/2026/(a) against-zone**: 2026-01-01 LONDON EQL 1h @25647.75 ord1; 2026-01-01 LONDON EQL 1h @25656 ord1; 2026-01-01 LONDON EQL 1h @25672.25 ord1; 2026-01-01 LONDON EQL 1h @25695.5 ord1; 2026-01-01 NY EQL 1h @25695.5 ord1
- **Q1/own/2026/(a) unknown-zone**: 2026-01-01 NY FVG 1h @25402.375 ord1; 2026-01-01 NY FVG 1h @25676.25 ord1; 2026-01-04 LONDON FVG 1h @25532.625 ord1; 2026-01-08 NY FVG 1h @25863.875 ord1; 2026-01-11 NY FVG 1h @25833.875 ord1
- **Q1/own/2026/(b) with-zone × 4h-with**: 2026-01-06 ASIA EQL 1h @25672.25 ord1; 2026-01-06 ASIA EQL 1h @25727 ord1; 2026-01-06 ASIA EQL 1h @25777.5 ord1; 2026-01-07 NY EQL 1h @25647.75 ord1; 2026-01-07 NY EQL 1h @25656 ord1
- **Q1/own/2026/(b) with-zone × 4h-against**: 2026-01-07 LONDON EQH 1h @25755 ord1; 2026-01-07 LONDON EQH 1h @25807 ord1; 2026-01-07 NY EQH 1h @25795 ord1; 2026-01-12 NY OB 1h @25979.5 ord1; 2026-01-19 NY IFVG 1h @25183.875 ord1
- **Q1/own/2026/(b) against-zone × 4h-with**: 2026-01-06 ASIA EQH 1h @25674.5 ord1; 2026-01-06 ASIA EQH 1h @25708 ord1; 2026-01-06 ASIA EQH 1h @25725.75 ord1; 2026-01-06 ASIA EQH 1h @25795 ord1; 2026-01-06 ASIA EQH 1h @25807 ord1
- **Q1/own/2026/(b) against-zone × 4h-against**: 2026-01-07 ASIA EQL 1h @25727 ord1; 2026-01-31 ASIA EQH 1h @25430 ord1; 2026-01-31 ASIA EQH 1h @25496.5 ord1; 2026-01-31 ASIA OB 1h @25390.5 ord1; 2026-02-02 LONDON EQH 1h @25998 ord1
- **Q1/own/2026/(c) with-zone × D-with**: 2026-01-15 LONDON EQL 1h @25803.25 ord1; 2026-01-15 LONDON IFVG 1h @25802.5 ord1; 2026-01-15 NY EQL 1h @25608.5 ord1; 2026-01-15 NY EQL 1h @25627 ord1; 2026-01-15 NY EQL 1h @25647.75 ord1
- **Q1/own/2026/(c) with-zone × D-against**: 2026-01-15 NY OB 1h @25879.75 ord1; 2026-01-18 LONDON IFVG 1h @25402.375 ord1; 2026-01-18 LONDON OB 1h @25380.25 ord1; 2026-01-18 NY SUPPLY 1h @25300.875 ord1; 2026-01-18 NY OB 1h @25380.25 ord1
- **Q1/own/2026/(c) against-zone × D-with**: 2026-01-15 LONDON EQH 1h @25825.75 ord1; 2026-01-15 NY EQH 1h @25702 ord1; 2026-01-15 NY EQH 1h @25725.75 ord1; 2026-01-15 NY EQH 1h @25744 ord1; 2026-01-15 NY EQH 1h @25755 ord1
- **Q1/own/2026/(c) against-zone × D-against**: 2026-01-15 NY EQL 1h @25867.25 ord1; 2026-01-15 NY OB 1h @25888.625 ord1; 2026-01-18 NY OB 1h @25318.25 ord1; 2026-01-19 NY EQL 1h @25257.5 ord1; 2026-01-19 NY EQL 1h @25421.25 ord1
- **Q1/own/2026/4-way with-zone × 4h-with × D-with**: 2026-06-02 ASIA OB 4h @30689.25 ord1; 2026-06-03 LONDON OB 4h @30689.25 ord1; 2026-06-03 LONDON EQL 1h @30565.25 ord1; 2026-06-03 LONDON EQL 1h @30790 ord1; 2026-06-03 LONDON DEMAND 1h @30624.125 ord1
- **Q1/own/2026/4-way with-zone × 4h-with × D-against**: 2026-01-19 NY EQH 1h @25419.25 ord1; 2026-01-19 NY SUPPLY 1h @25300.875 ord1; 2026-01-19 NY IFVG 1h @25402.375 ord1; 2026-01-19 NY OB 1h @25380.25 ord1; 2026-05-20 LONDON IFVG 1h @29406.25 ord1
- **Q1/own/2026/4-way with-zone × 4h-against × D-with**: 2026-01-19 NY IFVG 1h @25183.875 ord1; 2026-01-20 ASIA EQL 1h @25520 ord1; 2026-07-02 LONDON OB 4h @29922.375 ord1; 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-02 NY OB 4h @29922.375 ord1
- **Q1/own/2026/4-way with-zone × 4h-against × D-against**: 2026-06-03 LONDON EQH 1h @30829.5 ord1; 2026-06-03 NY EQH 4h @30668 ord1; 2026-06-03 NY EQH 1h @30644.5 ord1; 2026-06-03 NY EQH 1h @30668 ord1; 2026-06-03 NY SUPPLY 1h @30794.75 ord1
- **Q1/own/2026/4-way against-zone × 4h-with × D-with**: 2026-06-02 NY EQH 1h @30885 ord1; 2026-06-02 ASIA EQH 4h @30668 ord1; 2026-06-02 ASIA EQH 1h @30668 ord1; 2026-06-02 ASIA OB 1h @30722 ord1; 2026-06-03 LONDON EQH 4h @30668 ord1
- **Q1/own/2026/4-way against-zone × 4h-with × D-against**: 2026-01-19 NY EQL 1h @25257.5 ord1; 2026-01-19 NY EQL 1h @25421.25 ord1; 2026-01-19 NY OB 1h @25318.25 ord1; 2026-01-20 ASIA EQL 1h @25542.5 ord1; 2026-01-20 ASIA EQL 1h @25569.5 ord1
- **Q1/own/2026/4-way against-zone × 4h-against × D-with**: 2026-07-02 LONDON IFVG 4h @29910.75 ord1; 2026-07-02 LONDON EQL 1h @29922 ord1; 2026-07-02 NY EQL 1h @29922 ord1; 2026-07-02 NY IFVG 1h @29946.625 ord1; 2026-07-04 ASIA EQL 1h @29922 ord1
- **Q1/own/2026/4-way against-zone × 4h-against × D-against**: 2026-06-02 ASIA EQL 1h @30790 ord1; 2026-06-03 NY OB 4h @30689.25 ord1; 2026-06-03 NY EQL 1h @30565.25 ord1; 2026-06-03 NY DEMAND 1h @30624.125 ord1; 2026-06-03 NY OB 1h @30632.5 ord1
- **Q1/noconflict/era/(a) with-zone**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/noconflict/era/(a) against-zone**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/noconflict/era/(a) unknown-zone**: 2022-04-18 ASIA IFVG 1m @14064.25 ord1; 2022-04-18 ASIA OB 1m @14069 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA SWG-L 5m @14065 ord1
- **Q1/noconflict/era/(b) with-zone × 4h-with**: 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON OB 1m @21224.625 ord1; 2025-05-22 LONDON IFVG 15m @21225.125 ord1; 2025-05-22 NY RN 1m @21000 ord1; 2025-05-22 NY EQH 1h @20996 ord1
- **Q1/noconflict/era/(b) with-zone × 4h-against**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/noconflict/era/(b) with-zone × 4h-range**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/noconflict/era/(b) against-zone × 4h-with**: 2025-05-27 ASIA EQH 1m @21514 ord1; 2025-05-27 ASIA EQH 1m @21516.25 ord1; 2025-05-27 ASIA FVG 1m @21513.875 ord1; 2025-05-27 ASIA SWG-H 5m @21514 ord1; 2025-05-27 ASIA SWG-H 15m @21514 ord1
- **Q1/noconflict/era/(b) against-zone × 4h-against**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/noconflict/era/(b) against-zone × 4h-range**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/noconflict/era/(c) with-zone × D-with**: 2022-04-21 LONDON RN 1m @13775 ord1; 2022-04-21 LONDON FVG 1m @13773.5 ord1; 2022-04-21 LONDON EQH 15m @13773.25 ord1; 2022-04-21 NY RN 1m @13775 ord1; 2022-04-21 NY EQH 1m @13771.75 ord1
- **Q1/noconflict/era/(c) with-zone × D-against**: 2022-04-27 ASIA AS-H 1m @13244.5 ord1; 2022-04-27 ASIA RN 1m @13225 ord1; 2022-04-27 ASIA RN 1m @13250 ord1; 2022-04-27 ASIA EQH 1m @13222.5 ord1; 2022-04-27 ASIA EQH 1m @13225 ord1
- **Q1/noconflict/era/(c) with-zone × D-range**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/noconflict/era/(c) against-zone × D-with**: 2022-04-27 ASIA PDH 1m @13255 ord1; 2022-04-27 ASIA RTH-H 1m @13255 ord1; 2022-04-27 ASIA LDN-H 1m @13347.25 ord1; 2022-04-27 ASIA ONH 1m @13347.25 ord1; 2022-04-27 ASIA RN 1m @13275 ord1
- **Q1/noconflict/era/(c) against-zone × D-against**: 2022-04-26 ASIA IFVG 1m @13140.125 ord1; 2022-04-26 ASIA OB 1m @13140.375 ord1; 2022-04-27 LONDON PDC 1m @13197.5 ord1; 2022-04-27 LONDON RN 1m @13175 ord1; 2022-04-27 LONDON RN 1m @13200 ord1
- **Q1/noconflict/era/(c) against-zone × D-range**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/noconflict/era/4-way with-zone × 4h-with × D-with**: 2025-07-21 ASIA EQH 1m @23199 ord1; 2025-07-21 ASIA EQL 1m @23193.25 ord1; 2025-07-21 ASIA EQL 1m @23194.75 ord1; 2025-07-21 ASIA EQL 1m @23195.75 ord1; 2025-07-21 ASIA DEMAND 1m @23176.5 ord1
- **Q1/noconflict/era/4-way with-zone × 4h-with × D-against**: 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON OB 1m @21224.625 ord1; 2025-05-22 LONDON IFVG 15m @21225.125 ord1; 2025-05-22 NY RN 1m @21000 ord1; 2025-05-22 NY EQH 1h @20996 ord1
- **Q1/noconflict/era/4-way with-zone × 4h-against × D-with**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/noconflict/era/4-way with-zone × 4h-against × D-against**: 2025-05-27 ASIA LDN-H 1m @21537.75 ord1; 2025-05-27 ASIA ONH 1m @21537.75 ord1; 2025-05-27 ASIA RN 1m @21525 ord1; 2025-05-27 ASIA RN 1m @21550 ord1; 2025-05-27 ASIA IB-H 1m @21564.75 ord1
- **Q1/noconflict/era/4-way against-zone × 4h-with × D-with**: 2025-05-27 ASIA EQH 1m @21514 ord1; 2025-05-27 ASIA EQH 1m @21516.25 ord1; 2025-05-27 ASIA FVG 1m @21513.875 ord1; 2025-05-27 ASIA SWG-H 5m @21514 ord1; 2025-05-27 ASIA SWG-H 15m @21514 ord1
- **Q1/noconflict/era/4-way against-zone × 4h-with × D-against**: 2025-06-30 LONDON PDL 1m @22780 ord1; 2025-06-30 LONDON RTH-L 1m @22780 ord1; 2025-06-30 LONDON RN 1m @22775 ord1; 2025-06-30 LONDON RN 1m @22800 ord1; 2025-06-30 LONDON EQH 1m @22816.5 ord1
- **Q1/noconflict/era/4-way against-zone × 4h-against × D-with**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/noconflict/era/4-way against-zone × 4h-against × D-against**: 2025-07-21 ASIA PDL 1m @23223.5 ord1; 2025-07-21 ASIA RTH-L 1m @23249.5 ord1; 2025-07-21 ASIA RN 1m @23200 ord1; 2025-07-21 ASIA RN 1m @23225 ord1; 2025-07-21 ASIA RN 1m @23250 ord1
- **Q1/noconflict/era/side long × with-zone**: 2022-04-11 NY PDL 1m @13903.5 ord1; 2022-04-11 NY PDC 1m @13906.25 ord1; 2022-04-11 NY AS-L 1m @13902.5 ord1; 2022-04-11 NY ONL 1m @13902.5 ord1; 2022-04-11 NY RN 1m @13900 ord1
- **Q1/noconflict/era/side long × against-zone**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/noconflict/era/side short × with-zone**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/noconflict/era/side short × against-zone**: 2022-04-11 ASIA AS-H 1m @14018.75 ord1; 2022-04-11 ASIA RN 1m @14050 ord1; 2022-04-11 ASIA EQH 1m @13984.5 ord1; 2022-04-11 ASIA EQH 1m @14003 ord1; 2022-04-11 ASIA EQH 1m @14005 ord1
- **Q1/noconflict/2022/(a) with-zone**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/noconflict/2022/(a) against-zone**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/noconflict/2022/(a) unknown-zone**: 2022-04-18 ASIA IFVG 1m @14064.25 ord1; 2022-04-18 ASIA OB 1m @14069 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA SWG-L 5m @14065 ord1
- **Q1/noconflict/2022/(c) with-zone × D-with**: 2022-04-21 LONDON RN 1m @13775 ord1; 2022-04-21 LONDON FVG 1m @13773.5 ord1; 2022-04-21 LONDON EQH 15m @13773.25 ord1; 2022-04-21 NY RN 1m @13775 ord1; 2022-04-21 NY EQH 1m @13771.75 ord1
- **Q1/noconflict/2022/(c) with-zone × D-against**: 2022-04-27 ASIA AS-H 1m @13244.5 ord1; 2022-04-27 ASIA RN 1m @13225 ord1; 2022-04-27 ASIA RN 1m @13250 ord1; 2022-04-27 ASIA EQH 1m @13222.5 ord1; 2022-04-27 ASIA EQH 1m @13225 ord1
- **Q1/noconflict/2022/(c) against-zone × D-with**: 2022-04-27 ASIA PDH 1m @13255 ord1; 2022-04-27 ASIA RTH-H 1m @13255 ord1; 2022-04-27 ASIA LDN-H 1m @13347.25 ord1; 2022-04-27 ASIA ONH 1m @13347.25 ord1; 2022-04-27 ASIA RN 1m @13275 ord1
- **Q1/noconflict/2022/(c) against-zone × D-against**: 2022-04-26 ASIA IFVG 1m @13140.125 ord1; 2022-04-26 ASIA OB 1m @13140.375 ord1; 2022-04-27 LONDON PDC 1m @13197.5 ord1; 2022-04-27 LONDON RN 1m @13175 ord1; 2022-04-27 LONDON RN 1m @13200 ord1
- **Q1/noconflict/2023/(a) with-zone**: 2023-01-01 ASIA PDC 1m @11040.75 ord1; 2023-01-01 ASIA RN 1m @11000 ord1; 2023-01-01 ASIA RN 1m @11025 ord1; 2023-01-01 ASIA RN 1m @11100 ord1; 2023-01-01 ASIA EQH 1m @11005.5 ord1
- **Q1/noconflict/2023/(a) against-zone**: 2023-01-01 ASIA SUPPLY 15m @11107.875 ord1; 2023-01-01 ASIA OB 15m @11107.875 ord1; 2023-01-02 LONDON RN 1m @11100 ord1; 2023-01-02 LONDON FVG 1m @11104.375 ord1; 2023-01-02 LONDON FVG 1m @11099.875 ord1
- **Q1/noconflict/2023/(a) unknown-zone**: 2023-01-15 NY EQH 1m @11565.25 ord1; 2023-01-15 NY IFVG 1m @11565.75 ord1; 2023-01-15 NY OB 1m @11565.75 ord1; 2023-03-12 LONDON RN 1m @12150 ord1; 2023-03-12 LONDON EQH 1m @12142.5 ord1
- **Q1/noconflict/2023/(c) with-zone × D-with**: 2023-01-09 NY DEMAND 1m @11282.375 ord1; 2023-01-09 NY OB 1m @11281.875 ord1; 2023-01-09 ASIA RN 1m @11300 ord1; 2023-01-09 ASIA IB-H 1m @11283.25 ord1; 2023-01-09 ASIA EQH 1m @11282.5 ord1
- **Q1/noconflict/2023/(c) with-zone × D-against**: 2023-01-09 LONDON EQH 1m @11135 ord1; 2023-01-09 LONDON EQH 1m @11137.75 ord1; 2023-01-09 LONDON EQH 1m @11139.75 ord1; 2023-01-09 LONDON EQH 1m @11141.5 ord1; 2023-01-09 LONDON EQH 1m @11143 ord1
- **Q1/noconflict/2023/(c) against-zone × D-with**: 2023-01-09 LONDON RTH-L 1m @11191.25 ord1; 2023-01-09 LONDON RN 1m @11175 ord1; 2023-01-09 LONDON EQH 1m @11174.75 ord1; 2023-01-09 LONDON EQH 1m @11175.75 ord1; 2023-01-09 LONDON EQH 1m @11178 ord1
- **Q1/noconflict/2023/(c) against-zone × D-against**: 2023-01-10 NY PDH 1m @11301.75 ord1; 2023-01-10 NY AS-H 1m @11301.75 ord1; 2023-01-10 NY RN 1m @11300 ord1; 2023-01-10 NY EQH 1m @11293.5 ord1; 2023-01-10 NY EQH 1m @11294.5 ord1
- **Q1/noconflict/2024/(a) with-zone**: 2024-01-01 LONDON EQH 1m @17006.5 ord1; 2024-01-01 LONDON EQH 1m @17008.5 ord1; 2024-01-01 LONDON EQH 1m @17010.5 ord1; 2024-01-01 LONDON EQH 1m @17018.75 ord1; 2024-01-01 LONDON EQL 1m @16997 ord1
- **Q1/noconflict/2024/(a) against-zone**: 2024-01-01 LONDON EQH 1m @17019.5 ord1; 2024-01-01 LONDON SUPPLY 1m @17020 ord1; 2024-01-01 LONDON SUPPLY 1m @17019.25 ord1; 2024-01-01 LONDON OB 1m @17019.625 ord1; 2024-01-01 LONDON OB 1m @17019.625 ord1
- **Q1/noconflict/2024/(a) unknown-zone**: 2024-01-01 LONDON EQH 1m @16977.5 ord1; 2024-01-01 LONDON EQL 1m @16987.25 ord1; 2024-01-01 LONDON SUPPLY 1m @16977.875 ord1; 2024-01-01 LONDON SUPPLY 1m @16982 ord1; 2024-01-01 LONDON SUPPLY 1m @16980.5 ord1
- **Q1/noconflict/2024/(c) with-zone × D-with**: 2024-02-14 NY PDH 1m @17887 ord1; 2024-02-14 NY LDN-L 1m @17879 ord1; 2024-02-14 NY EQH 1m @17797.75 ord1; 2024-02-14 NY EQH 1m @17870 ord1; 2024-02-14 NY EQH 1m @17871.5 ord1
- **Q1/noconflict/2024/(c) with-zone × D-against**: 2024-02-01 LONDON PDC 1m @17610 ord1; 2024-02-01 LONDON AS-H 1m @17631 ord1; 2024-02-01 LONDON ONH 1m @17631 ord1; 2024-02-01 LONDON RN 1m @17625 ord1; 2024-02-01 LONDON EQH 1m @17607.25 ord1
- **Q1/noconflict/2024/(c) against-zone × D-with**: 2024-02-04 LONDON RN 1m @17675 ord1; 2024-02-04 LONDON EQH 1m @17680 ord1; 2024-02-04 LONDON EQH 1m @17681.5 ord1; 2024-02-04 LONDON EQH 1m @17683.25 ord1; 2024-02-04 LONDON EQH 1m @17687 ord1
- **Q1/noconflict/2024/(c) against-zone × D-against**: 2024-02-14 NY RN 1m @17900 ord1; 2024-02-14 NY EQH 1m @17895 ord1; 2024-02-14 NY EQH 1m @17899.5 ord1; 2024-02-14 NY EQH 1m @17901 ord1; 2024-02-14 NY EQH 1m @17902.75 ord1
- **Q1/noconflict/2025/(a) with-zone**: 2025-01-01 LONDON AS-H 1m @21398.5 ord1; 2025-01-01 LONDON ONH 1m @21398.5 ord1; 2025-01-01 LONDON RN 1m @21375 ord1; 2025-01-01 LONDON RN 1m @21400 ord1; 2025-01-01 LONDON RN 1m @21450 ord1
- **Q1/noconflict/2025/(a) against-zone**: 2025-01-01 LONDON RN 1m @21325 ord1; 2025-01-01 LONDON RN 1m @21350 ord1; 2025-01-01 LONDON EQH 1m @21319.5 ord1; 2025-01-01 LONDON EQH 1m @21320 ord1; 2025-01-01 LONDON EQH 1m @21326.25 ord1
- **Q1/noconflict/2025/(a) unknown-zone**: 2025-01-19 ASIA FVG 1m @21418.75 ord1; 2025-01-19 ASIA FVG 1h @21420.25 ord1; 2025-01-23 LONDON PDC 1m @21993.75 ord1; 2025-01-23 LONDON RTH-H 1m @21984.25 ord1; 2025-01-23 LONDON EQH 1m @21994.25 ord1
- **Q1/noconflict/2025/(b) with-zone × 4h-with**: 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON OB 1m @21224.625 ord1; 2025-05-22 LONDON IFVG 15m @21225.125 ord1; 2025-05-22 NY RN 1m @21000 ord1; 2025-05-22 NY EQH 1h @20996 ord1
- **Q1/noconflict/2025/(b) with-zone × 4h-against**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/noconflict/2025/(b) against-zone × 4h-with**: 2025-05-27 ASIA EQH 1m @21514 ord1; 2025-05-27 ASIA EQH 1m @21516.25 ord1; 2025-05-27 ASIA FVG 1m @21513.875 ord1; 2025-05-27 ASIA SWG-H 5m @21514 ord1; 2025-05-27 ASIA SWG-H 15m @21514 ord1
- **Q1/noconflict/2025/(b) against-zone × 4h-against**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/noconflict/2025/(c) with-zone × D-with**: 2025-01-07 LONDON EQH 1m @21410 ord1; 2025-01-07 LONDON EQH 1m @21414 ord1; 2025-01-07 LONDON EQH 1m @21416 ord1; 2025-01-07 LONDON EQH 1m @21417.75 ord1; 2025-01-07 LONDON EQH 1m @21419 ord1
- **Q1/noconflict/2025/(c) with-zone × D-against**: 2025-01-09 LONDON RN 1m @21050 ord1; 2025-01-09 LONDON RN 1m @21075 ord1; 2025-01-09 LONDON RN 1m @21100 ord1; 2025-01-09 LONDON RN 1m @21125 ord1; 2025-01-09 LONDON RN 1m @21150 ord1
- **Q1/noconflict/2025/(c) against-zone × D-with**: 2025-01-12 LONDON RN 1m @20900 ord1; 2025-01-12 LONDON EQH 1m @20886.25 ord1; 2025-01-12 LONDON EQH 1m @20890.5 ord1; 2025-01-12 LONDON EQH 1m @20900.5 ord1; 2025-01-12 LONDON EQH 1m @20909 ord1
- **Q1/noconflict/2025/(c) against-zone × D-against**: 2025-01-07 LONDON AS-L 1m @21344.75 ord1; 2025-01-07 LONDON ONL 1m @21344.75 ord1; 2025-01-07 LONDON RN 1m @21325 ord1; 2025-01-07 LONDON RN 1m @21350 ord1; 2025-01-07 LONDON SUPPLY 1m @21348.875 ord1
- **Q1/noconflict/2025/4-way with-zone × 4h-with × D-with**: 2025-07-21 ASIA EQH 1m @23199 ord1; 2025-07-21 ASIA EQL 1m @23193.25 ord1; 2025-07-21 ASIA EQL 1m @23194.75 ord1; 2025-07-21 ASIA EQL 1m @23195.75 ord1; 2025-07-21 ASIA DEMAND 1m @23176.5 ord1
- **Q1/noconflict/2025/4-way with-zone × 4h-with × D-against**: 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON OB 1m @21224.625 ord1; 2025-05-22 LONDON IFVG 15m @21225.125 ord1; 2025-05-22 NY RN 1m @21000 ord1; 2025-05-22 NY EQH 1h @20996 ord1
- **Q1/noconflict/2025/4-way with-zone × 4h-against × D-with**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/noconflict/2025/4-way with-zone × 4h-against × D-against**: 2025-05-27 ASIA LDN-H 1m @21537.75 ord1; 2025-05-27 ASIA ONH 1m @21537.75 ord1; 2025-05-27 ASIA RN 1m @21525 ord1; 2025-05-27 ASIA RN 1m @21550 ord1; 2025-05-27 ASIA IB-H 1m @21564.75 ord1
- **Q1/noconflict/2025/4-way against-zone × 4h-with × D-with**: 2025-05-27 ASIA EQH 1m @21514 ord1; 2025-05-27 ASIA EQH 1m @21516.25 ord1; 2025-05-27 ASIA FVG 1m @21513.875 ord1; 2025-05-27 ASIA SWG-H 5m @21514 ord1; 2025-05-27 ASIA SWG-H 15m @21514 ord1
- **Q1/noconflict/2025/4-way against-zone × 4h-with × D-against**: 2025-06-30 LONDON PDL 1m @22780 ord1; 2025-06-30 LONDON RTH-L 1m @22780 ord1; 2025-06-30 LONDON RN 1m @22775 ord1; 2025-06-30 LONDON RN 1m @22800 ord1; 2025-06-30 LONDON EQH 1m @22816.5 ord1
- **Q1/noconflict/2025/4-way against-zone × 4h-against × D-with**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/noconflict/2025/4-way against-zone × 4h-against × D-against**: 2025-07-21 ASIA PDL 1m @23223.5 ord1; 2025-07-21 ASIA RTH-L 1m @23249.5 ord1; 2025-07-21 ASIA RN 1m @23200 ord1; 2025-07-21 ASIA RN 1m @23225 ord1; 2025-07-21 ASIA RN 1m @23250 ord1
- **Q1/noconflict/2026/(a) with-zone**: 2026-01-01 LONDON RN 1m @25675 ord1; 2026-01-01 LONDON RN 1m @25700 ord1; 2026-01-01 LONDON EQH 1m @25642.25 ord1; 2026-01-01 LONDON EQH 1m @25644 ord1; 2026-01-01 LONDON EQH 1m @25658.75 ord1
- **Q1/noconflict/2026/(a) against-zone**: 2026-01-01 LONDON EQH 1m @25638.5 ord1; 2026-01-01 LONDON SWG-H 15m @25637.25 ord1; 2026-01-01 NY PDL 1m @25426.5 ord1; 2026-01-01 NY PDC 1m @25434.5 ord1; 2026-01-01 NY AS-L 1m @25449.25 ord1
- **Q1/noconflict/2026/(a) unknown-zone**: 2026-01-01 NY RN 1m @25400 ord1; 2026-01-01 NY FVG 1h @25402.375 ord1; 2026-01-04 LONDON RN 1m @25525 ord1; 2026-01-04 LONDON EQH 1m @25536.5 ord1; 2026-01-04 LONDON IFVG 1m @25516.5 ord1
- **Q1/noconflict/2026/(b) with-zone × 4h-with**: 2026-01-06 ASIA RTH-H 1m @25835 ord1; 2026-01-06 ASIA EQH 1m @25834.75 ord1; 2026-01-06 ASIA EQH 1m @25835.5 ord1; 2026-01-06 ASIA EQH 1m @25837.25 ord1; 2026-01-06 ASIA OB 1m @25835.75 ord1
- **Q1/noconflict/2026/(b) with-zone × 4h-against**: 2026-01-07 LONDON RN 1m @25800 ord1; 2026-01-07 LONDON EQH 1m @25781.75 ord1; 2026-01-07 LONDON EQH 1m @25782.75 ord1; 2026-01-07 LONDON EQH 1m @25783.5 ord1; 2026-01-07 LONDON EQH 1m @25785.25 ord1
- **Q1/noconflict/2026/(b) against-zone × 4h-with**: 2026-01-06 ASIA LDN-H 1m @25793.75 ord1; 2026-01-06 ASIA RN 1m @25700 ord1; 2026-01-06 ASIA RN 1m @25800 ord1; 2026-01-06 ASIA IB-L 1m @25716.125 ord1; 2026-01-06 ASIA EQH 1m @25675.75 ord1
- **Q1/noconflict/2026/(b) against-zone × 4h-against**: 2026-02-02 LONDON PDH 1m @25996.75 ord1; 2026-02-02 LONDON RN 1m @25925 ord1; 2026-02-02 LONDON RN 1m @25975 ord1; 2026-02-02 LONDON EQH 1m @25924 ord1; 2026-02-02 LONDON EQH 1m @25935 ord1
- **Q1/noconflict/2026/(c) with-zone × D-with**: 2026-01-19 LONDON PDL 1m @25236.25 ord1; 2026-01-19 LONDON RTH-L 1m @25236.25 ord1; 2026-01-19 LONDON AS-L 1m @25256.5 ord1; 2026-01-19 LONDON ONL 1m @25256.5 ord1; 2026-01-19 LONDON RN 1m @25100 ord1
- **Q1/noconflict/2026/(c) with-zone × D-against**: 2026-01-18 LONDON RN 1m @25400 ord1; 2026-01-18 LONDON EQH 1m @25389.75 ord1; 2026-01-18 LONDON EQH 1m @25407.75 ord1; 2026-01-18 LONDON EQH 1m @25410.25 ord1; 2026-01-18 LONDON EQL 1m @25390.25 ord1
- **Q1/noconflict/2026/(c) against-zone × D-with**: 2026-01-17 ASIA EQH 1h @25417 ord1; 2026-01-20 LONDON RN 1m @25200 ord1; 2026-01-20 LONDON EQH 1m @25198.75 ord1; 2026-01-20 LONDON EQH 1m @25199.75 ord1; 2026-01-20 LONDON EQH 1m @25201.25 ord1
- **Q1/noconflict/2026/(c) against-zone × D-against**: 2026-01-19 LONDON EQL 1m @25276.75 ord1; 2026-01-19 LONDON EQL 1m @25279.75 ord1; 2026-01-19 LONDON EQL 1m @25283 ord1; 2026-01-19 LONDON EQL 1m @25287.5 ord1; 2026-01-19 LONDON IFVG 1m @25287.375 ord1
- **Q1/noconflict/2026/4-way with-zone × 4h-with × D-with**: 2026-06-03 NY RN 1m @30450 ord1; 2026-06-04 LONDON IFVG 1m @30423.25 ord1; 2026-06-04 LONDON IFVG 1m @30409.375 ord1; 2026-06-04 LONDON IFVG 1m @30416.5 ord1; 2026-06-04 LONDON IFVG 1m @30417.5 ord1
- **Q1/noconflict/2026/4-way with-zone × 4h-with × D-against**: 2026-01-19 NY PDC 1m @25384.75 ord1; 2026-01-19 NY RN 1m @25250 ord1; 2026-01-19 NY RN 1m @25275 ord1; 2026-01-19 NY RN 1m @25375 ord1; 2026-01-19 NY EQH 1m @25379 ord1
- **Q1/noconflict/2026/4-way with-zone × 4h-against × D-with**: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY EQH 1m @25135.75 ord1; 2026-01-19 NY EQL 1m @25166.5 ord1
- **Q1/noconflict/2026/4-way with-zone × 4h-against × D-against**: 2026-06-02 ASIA DEMAND 1m @30802.75 ord1; 2026-06-02 ASIA SUPPLY 1m @30812.75 ord1; 2026-06-02 ASIA IFVG 1m @30815.875 ord1; 2026-06-02 ASIA IFVG 1m @30803.125 ord1; 2026-06-02 ASIA IFVG 1m @30815 ord1
- **Q1/noconflict/2026/4-way against-zone × 4h-with × D-with**: 2026-06-02 NY RN 1m @30800 ord1; 2026-06-02 NY SUPPLY 1m @30885.625 ord1; 2026-06-02 NY DEMAND 1m @30883.875 ord1; 2026-06-02 NY OB 1m @30885.625 ord1; 2026-06-02 NY OB 1m @30883.875 ord1
- **Q1/noconflict/2026/4-way against-zone × 4h-with × D-against**: 2026-01-20 ASIA EQH 1m @25530.75 ord1; 2026-01-20 ASIA IFVG 1m @25539.625 ord1; 2026-01-20 ASIA FVG 1m @25581.75 ord1; 2026-01-20 ASIA OB 1m @25566 ord1; 2026-01-20 ASIA OB 1m @25566 ord1
- **Q1/noconflict/2026/4-way against-zone × 4h-against × D-with**: 2026-05-20 ASIA RN 1m @29450 ord1; 2026-05-20 ASIA IB-H 1m @29444.5 ord1; 2026-05-20 ASIA DEMAND 1m @29450.375 ord1; 2026-05-20 ASIA IFVG 1m @29445.25 ord1; 2026-05-20 ASIA IFVG 1m @29447.75 ord1
- **Q1/noconflict/2026/4-way against-zone × 4h-against × D-against**: 2026-06-04 LONDON RN 1m @30425 ord1; 2026-06-04 LONDON EQL 1m @30428 ord1; 2026-06-04 LONDON IFVG 1m @30428.625 ord1; 2026-06-04 LONDON VWAP 1m @30427.75846335576 ord1; 2026-07-26 NY RN 1m @28350 ord1
- **Q1/all/era/(a) with-zone**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/all/era/(a) against-zone**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/all/era/(a) unknown-zone**: 2022-04-18 ASIA IFVG 1m @14064.25 ord1; 2022-04-18 ASIA OB 1m @14069 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA SWG-L 5m @14065 ord1
- **Q1/all/era/(b) with-zone × 4h-with**: 2025-05-22 LONDON RN 1m @21200 ord1; 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON EQH 1m @21176 ord1; 2025-05-22 LONDON EQH 1m @21177 ord1; 2025-05-22 LONDON EQH 1m @21178.25 ord1
- **Q1/all/era/(b) with-zone × 4h-against**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/all/era/(b) with-zone × 4h-range**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/all/era/(b) against-zone × 4h-with**: 2025-05-22 LONDON PDC 1m @21151.5 ord1; 2025-05-22 LONDON RN 1m @21150 ord1; 2025-05-22 LONDON RN 1m @21175 ord1; 2025-05-22 LONDON EQH 1m @21141.5 ord1; 2025-05-22 LONDON EQH 1m @21147.5 ord1
- **Q1/all/era/(b) against-zone × 4h-against**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/all/era/(b) against-zone × 4h-range**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/all/era/(c) with-zone × D-with**: 2022-04-21 LONDON RN 1m @13775 ord1; 2022-04-21 LONDON FVG 1m @13773.5 ord1; 2022-04-21 LONDON EQH 15m @13773.25 ord1; 2022-04-21 NY RN 1m @13775 ord1; 2022-04-21 NY EQH 1m @13771.75 ord1
- **Q1/all/era/(c) with-zone × D-against**: 2022-04-27 NY PDC 1m @13197.5 ord1; 2022-04-27 NY RN 1m @13050 ord1; 2022-04-27 NY RN 1m @13075 ord1; 2022-04-27 NY RN 1m @13200 ord1; 2022-04-27 NY EQH 1m @13036.25 ord1
- **Q1/all/era/(c) with-zone × D-range**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/all/era/(c) against-zone × D-with**: 2022-04-26 ASIA LDN-H 1m @13184 ord1; 2022-04-26 ASIA ONH 1m @13184 ord1; 2022-04-26 ASIA EQH 1m @13184 ord1; 2022-04-26 ASIA SUPPLY 1m @13188 ord1; 2022-04-26 ASIA SUPPLY 1m @13185.375 ord1
- **Q1/all/era/(c) against-zone × D-against**: 2022-04-26 ASIA IFVG 1m @13140.125 ord1; 2022-04-26 ASIA OB 1m @13140.375 ord1; 2022-04-27 LONDON PDC 1m @13197.5 ord1; 2022-04-27 LONDON RN 1m @13175 ord1; 2022-04-27 LONDON RN 1m @13200 ord1
- **Q1/all/era/(c) against-zone × D-range**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/all/era/4-way with-zone × 4h-with × D-with**: 2025-07-21 NY RTH-L 1m @23249.5 ord1; 2025-07-21 NY RN 1m @23125 ord1; 2025-07-21 NY RN 1m @23150 ord1; 2025-07-21 NY RN 1m @23175 ord1; 2025-07-21 NY RN 1m @23250 ord1
- **Q1/all/era/4-way with-zone × 4h-with × D-against**: 2025-05-22 LONDON RN 1m @21200 ord1; 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON EQH 1m @21176 ord1; 2025-05-22 LONDON EQH 1m @21177 ord1; 2025-05-22 LONDON EQH 1m @21178.25 ord1
- **Q1/all/era/4-way with-zone × 4h-against × D-with**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/all/era/4-way with-zone × 4h-against × D-against**: 2025-05-27 ASIA LDN-H 1m @21537.75 ord1; 2025-05-27 ASIA ONH 1m @21537.75 ord1; 2025-05-27 ASIA RN 1m @21525 ord1; 2025-05-27 ASIA RN 1m @21550 ord1; 2025-05-27 ASIA IB-H 1m @21564.75 ord1
- **Q1/all/era/4-way against-zone × 4h-with × D-with**: 2025-05-27 ASIA RN 1m @21500 ord1; 2025-05-27 ASIA OR-L 1m @21490.25 ord1; 2025-05-27 ASIA EQH 1m @21491.5 ord1; 2025-05-27 ASIA EQH 1m @21494 ord1; 2025-05-27 ASIA EQH 1m @21499.75 ord1
- **Q1/all/era/4-way against-zone × 4h-with × D-against**: 2025-05-22 LONDON PDC 1m @21151.5 ord1; 2025-05-22 LONDON RN 1m @21150 ord1; 2025-05-22 LONDON RN 1m @21175 ord1; 2025-05-22 LONDON EQH 1m @21141.5 ord1; 2025-05-22 LONDON EQH 1m @21147.5 ord1
- **Q1/all/era/4-way against-zone × 4h-against × D-with**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/all/era/4-way against-zone × 4h-against × D-against**: 2025-07-21 ASIA PDL 1m @23223.5 ord1; 2025-07-21 ASIA RTH-L 1m @23249.5 ord1; 2025-07-21 ASIA RN 1m @23200 ord1; 2025-07-21 ASIA RN 1m @23225 ord1; 2025-07-21 ASIA RN 1m @23250 ord1
- **Q1/all/era/side long × with-zone**: 2022-04-11 NY PDL 1m @13903.5 ord1; 2022-04-11 NY PDC 1m @13906.25 ord1; 2022-04-11 NY AS-L 1m @13902.5 ord1; 2022-04-11 NY ONL 1m @13902.5 ord1; 2022-04-11 NY RN 1m @13900 ord1
- **Q1/all/era/side long × against-zone**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/all/era/side short × with-zone**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/all/era/side short × against-zone**: 2022-04-11 ASIA AS-H 1m @14018.75 ord1; 2022-04-11 ASIA RN 1m @14050 ord1; 2022-04-11 ASIA EQH 1m @13984.5 ord1; 2022-04-11 ASIA EQH 1m @14003 ord1; 2022-04-11 ASIA EQH 1m @14005 ord1
- **Q1/all/2022/(a) with-zone**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q1/all/2022/(a) against-zone**: 2022-04-11 NY RTH-H 1m @14197 ord1; 2022-04-11 NY RN 1m @14050 ord1; 2022-04-11 NY RN 1m @14075 ord1; 2022-04-11 NY RN 1m @14100 ord1; 2022-04-11 NY RN 1m @14125 ord1
- **Q1/all/2022/(a) unknown-zone**: 2022-04-18 ASIA IFVG 1m @14064.25 ord1; 2022-04-18 ASIA OB 1m @14069 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA OB 1m @14066 ord1; 2022-04-18 ASIA SWG-L 5m @14065 ord1
- **Q1/all/2022/(c) with-zone × D-with**: 2022-04-21 LONDON RN 1m @13775 ord1; 2022-04-21 LONDON FVG 1m @13773.5 ord1; 2022-04-21 LONDON EQH 15m @13773.25 ord1; 2022-04-21 NY RN 1m @13775 ord1; 2022-04-21 NY EQH 1m @13771.75 ord1
- **Q1/all/2022/(c) with-zone × D-against**: 2022-04-27 NY PDC 1m @13197.5 ord1; 2022-04-27 NY RN 1m @13050 ord1; 2022-04-27 NY RN 1m @13075 ord1; 2022-04-27 NY RN 1m @13200 ord1; 2022-04-27 NY EQH 1m @13036.25 ord1
- **Q1/all/2022/(c) against-zone × D-with**: 2022-04-26 ASIA LDN-H 1m @13184 ord1; 2022-04-26 ASIA ONH 1m @13184 ord1; 2022-04-26 ASIA EQH 1m @13184 ord1; 2022-04-26 ASIA SUPPLY 1m @13188 ord1; 2022-04-26 ASIA SUPPLY 1m @13185.375 ord1
- **Q1/all/2022/(c) against-zone × D-against**: 2022-04-26 ASIA IFVG 1m @13140.125 ord1; 2022-04-26 ASIA OB 1m @13140.375 ord1; 2022-04-27 LONDON PDC 1m @13197.5 ord1; 2022-04-27 LONDON RN 1m @13175 ord1; 2022-04-27 LONDON RN 1m @13200 ord1
- **Q1/all/2023/(a) with-zone**: 2023-01-01 ASIA PDC 1m @11040.75 ord1; 2023-01-01 ASIA RTH-H 1m @10971 ord1; 2023-01-01 ASIA RN 1m @10975 ord1; 2023-01-01 ASIA RN 1m @11000 ord1; 2023-01-01 ASIA RN 1m @11025 ord1
- **Q1/all/2023/(a) against-zone**: 2023-01-01 ASIA RN 1m @11075 ord1; 2023-01-01 ASIA RN 1m @11125 ord1; 2023-01-01 ASIA RN 1m @11150 ord1; 2023-01-01 ASIA EQH 1m @10952.25 ord1; 2023-01-01 ASIA EQL 1m @10952.75 ord1
- **Q1/all/2023/(a) unknown-zone**: 2023-01-05 LONDON EQH 1m @10966.5 ord1; 2023-01-05 LONDON EQH 1m @10967.75 ord1; 2023-01-05 LONDON IFVG 1m @10966.25 ord1; 2023-01-05 LONDON IFVG 1m @10967.25 ord1; 2023-01-05 LONDON IFVG 1m @10969.125 ord1
- **Q1/all/2023/(c) with-zone × D-with**: 2023-01-09 LONDON AS-H 1m @11187.5 ord1; 2023-01-09 LONDON ONH 1m @11187.5 ord1; 2023-01-09 LONDON EQH 1m @11169.75 ord1; 2023-01-09 LONDON EQH 1m @11171.25 ord1; 2023-01-09 LONDON EQH 1m @11172.5 ord1
- **Q1/all/2023/(c) with-zone × D-against**: 2023-01-09 LONDON PDL 1m @11120.25 ord1; 2023-01-09 LONDON AS-L 1m @11123.75 ord1; 2023-01-09 LONDON ONL 1m @11123.75 ord1; 2023-01-09 LONDON RN 1m @11125 ord1; 2023-01-09 LONDON EQH 1m @11135 ord1
- **Q1/all/2023/(c) against-zone × D-with**: 2023-01-09 LONDON RTH-L 1m @11191.25 ord1; 2023-01-09 LONDON RN 1m @11175 ord1; 2023-01-09 LONDON EQH 1m @11161.75 ord1; 2023-01-09 LONDON EQH 1m @11162.75 ord1; 2023-01-09 LONDON EQH 1m @11163.75 ord1
- **Q1/all/2023/(c) against-zone × D-against**: 2023-01-09 LONDON PDC 1m @11157.25 ord1; 2023-01-09 LONDON RN 1m @11100 ord1; 2023-01-09 LONDON GAP 1m @11109.25 ord1; 2023-01-09 LONDON EQH 1m @11107.75 ord1; 2023-01-09 LONDON EQH 1m @11109.5 ord1
- **Q1/all/2024/(a) with-zone**: 2024-01-01 LONDON PDL 1m @16937.75 ord1; 2024-01-01 LONDON RTH-L 1m @16937.75 ord1; 2024-01-01 LONDON RN 1m @16850 ord1; 2024-01-01 LONDON RN 1m @16875 ord1; 2024-01-01 LONDON RN 1m @16950 ord1
- **Q1/all/2024/(a) against-zone**: 2024-01-01 LONDON RN 1m @16900 ord1; 2024-01-01 LONDON RN 1m @16925 ord1; 2024-01-01 LONDON RN 1m @16975 ord1; 2024-01-01 LONDON EQH 1m @17011.75 ord1; 2024-01-01 LONDON EQH 1m @17012.75 ord1
- **Q1/all/2024/(a) unknown-zone**: 2024-01-01 LONDON EQH 1m @16972 ord1; 2024-01-01 LONDON EQH 1m @16977.5 ord1; 2024-01-01 LONDON EQL 1m @16964.75 ord1; 2024-01-01 LONDON EQL 1m @16987.25 ord1; 2024-01-01 LONDON SUPPLY 1m @16968.375 ord1
- **Q1/all/2024/(c) with-zone × D-with**: 2024-02-01 LONDON RN 1m @17500 ord1; 2024-02-01 LONDON RN 1m @17525 ord1; 2024-02-01 LONDON RN 1m @17550 ord1; 2024-02-01 LONDON EQH 1m @17594.25 ord1; 2024-02-01 LONDON EQL 1m @17594.25 ord1
- **Q1/all/2024/(c) with-zone × D-against**: 2024-02-01 LONDON PDC 1m @17610 ord1; 2024-02-01 LONDON AS-H 1m @17631 ord1; 2024-02-01 LONDON ONH 1m @17631 ord1; 2024-02-01 LONDON RN 1m @17625 ord1; 2024-02-01 LONDON EQH 1m @17599 ord1
- **Q1/all/2024/(c) against-zone × D-with**: 2024-02-01 LONDON AS-L 1m @17579.5 ord1; 2024-02-01 LONDON ONL 1m @17579.5 ord1; 2024-02-01 LONDON RN 1m @17575 ord1; 2024-02-01 LONDON EQH 1m @17589 ord1; 2024-02-01 LONDON EQH 1m @17593 ord1
- **Q1/all/2024/(c) against-zone × D-against**: 2024-02-01 LONDON OB 15m @17596.875 ord1; 2024-02-01 NY PDH 1m @17645.5 ord1; 2024-02-01 NY RN 1m @17500 ord1; 2024-02-01 NY RN 1m @17525 ord1; 2024-02-01 NY RN 1m @17550 ord1
- **Q1/all/2025/(a) with-zone**: 2025-01-01 LONDON RTH-H 1m @21489.5 ord1; 2025-01-01 LONDON AS-H 1m @21398.5 ord1; 2025-01-01 LONDON ONH 1m @21398.5 ord1; 2025-01-01 LONDON RN 1m @21375 ord1; 2025-01-01 LONDON RN 1m @21400 ord1
- **Q1/all/2025/(a) against-zone**: 2025-01-01 LONDON RN 1m @21325 ord1; 2025-01-01 LONDON RN 1m @21350 ord1; 2025-01-01 LONDON RN 1m @21425 ord1; 2025-01-01 LONDON EQH 1m @21319.5 ord1; 2025-01-01 LONDON EQH 1m @21320 ord1
- **Q1/all/2025/(a) unknown-zone**: 2025-01-14 NY FVG 1m @21229.75 ord1; 2025-01-14 NY FVG 1h @21227 ord1; 2025-01-16 LONDON RN 1m @21625 ord1; 2025-01-16 LONDON FVG 1h @21634 ord1; 2025-01-16 NY FVG 1h @21634 ord1
- **Q1/all/2025/(b) with-zone × 4h-with**: 2025-05-22 LONDON RN 1m @21200 ord1; 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON EQH 1m @21176 ord1; 2025-05-22 LONDON EQH 1m @21177 ord1; 2025-05-22 LONDON EQH 1m @21178.25 ord1
- **Q1/all/2025/(b) with-zone × 4h-against**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/all/2025/(b) against-zone × 4h-with**: 2025-05-22 LONDON PDC 1m @21151.5 ord1; 2025-05-22 LONDON RN 1m @21150 ord1; 2025-05-22 LONDON RN 1m @21175 ord1; 2025-05-22 LONDON EQH 1m @21141.5 ord1; 2025-05-22 LONDON EQH 1m @21147.5 ord1
- **Q1/all/2025/(b) against-zone × 4h-against**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/all/2025/(c) with-zone × D-with**: 2025-01-07 LONDON RN 1m @21400 ord1; 2025-01-07 LONDON EQH 1m @21405.5 ord1; 2025-01-07 LONDON EQH 1m @21406.5 ord1; 2025-01-07 LONDON EQH 1m @21408 ord1; 2025-01-07 LONDON EQH 1m @21409.5 ord1
- **Q1/all/2025/(c) with-zone × D-against**: 2025-01-07 LONDON RN 1m @21225 ord1; 2025-01-07 LONDON RN 1m @21250 ord1; 2025-01-07 LONDON RN 1m @21375 ord1; 2025-01-07 LONDON EQH 1m @21369 ord1; 2025-01-07 LONDON EQH 1m @21379 ord1
- **Q1/all/2025/(c) against-zone × D-with**: 2025-01-07 NY PDC 1m @21430 ord1; 2025-01-07 NY RN 1m @21425 ord1; 2025-01-07 NY EQH 1m @21383.5 ord1; 2025-01-07 NY EQH 1m @21385 ord1; 2025-01-07 NY EQH 1m @21387 ord1
- **Q1/all/2025/(c) against-zone × D-against**: 2025-01-07 LONDON PDL 1m @21279.5 ord1; 2025-01-07 LONDON RTH-L 1m @21279.5 ord1; 2025-01-07 LONDON AS-L 1m @21344.75 ord1; 2025-01-07 LONDON ONL 1m @21344.75 ord1; 2025-01-07 LONDON RN 1m @21275 ord1
- **Q1/all/2025/4-way with-zone × 4h-with × D-with**: 2025-07-21 NY RTH-L 1m @23249.5 ord1; 2025-07-21 NY RN 1m @23125 ord1; 2025-07-21 NY RN 1m @23150 ord1; 2025-07-21 NY RN 1m @23175 ord1; 2025-07-21 NY RN 1m @23250 ord1
- **Q1/all/2025/4-way with-zone × 4h-with × D-against**: 2025-05-22 LONDON RN 1m @21200 ord1; 2025-05-22 LONDON RN 1m @21225 ord1; 2025-05-22 LONDON EQH 1m @21176 ord1; 2025-05-22 LONDON EQH 1m @21177 ord1; 2025-05-22 LONDON EQH 1m @21178.25 ord1
- **Q1/all/2025/4-way with-zone × 4h-against × D-with**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q1/all/2025/4-way with-zone × 4h-against × D-against**: 2025-05-27 ASIA LDN-H 1m @21537.75 ord1; 2025-05-27 ASIA ONH 1m @21537.75 ord1; 2025-05-27 ASIA RN 1m @21525 ord1; 2025-05-27 ASIA RN 1m @21550 ord1; 2025-05-27 ASIA IB-H 1m @21564.75 ord1
- **Q1/all/2025/4-way against-zone × 4h-with × D-with**: 2025-05-27 ASIA RN 1m @21500 ord1; 2025-05-27 ASIA OR-L 1m @21490.25 ord1; 2025-05-27 ASIA EQH 1m @21491.5 ord1; 2025-05-27 ASIA EQH 1m @21494 ord1; 2025-05-27 ASIA EQH 1m @21499.75 ord1
- **Q1/all/2025/4-way against-zone × 4h-with × D-against**: 2025-05-22 LONDON PDC 1m @21151.5 ord1; 2025-05-22 LONDON RN 1m @21150 ord1; 2025-05-22 LONDON RN 1m @21175 ord1; 2025-05-22 LONDON EQH 1m @21141.5 ord1; 2025-05-22 LONDON EQH 1m @21147.5 ord1
- **Q1/all/2025/4-way against-zone × 4h-against × D-with**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q1/all/2025/4-way against-zone × 4h-against × D-against**: 2025-07-21 ASIA PDL 1m @23223.5 ord1; 2025-07-21 ASIA RTH-L 1m @23249.5 ord1; 2025-07-21 ASIA RN 1m @23200 ord1; 2025-07-21 ASIA RN 1m @23225 ord1; 2025-07-21 ASIA RN 1m @23250 ord1
- **Q1/all/2026/(a) with-zone**: 2026-01-01 LONDON RN 1m @25675 ord1; 2026-01-01 LONDON RN 1m @25700 ord1; 2026-01-01 LONDON EQH 1m @25642.25 ord1; 2026-01-01 LONDON EQH 1m @25644 ord1; 2026-01-01 LONDON EQH 1m @25658.75 ord1
- **Q1/all/2026/(a) against-zone**: 2026-01-01 LONDON AS-H 1m @25650 ord1; 2026-01-01 LONDON ONH 1m @25650 ord1; 2026-01-01 LONDON RN 1m @25650 ord1; 2026-01-01 LONDON EQH 1m @25638.5 ord1; 2026-01-01 LONDON EQH 1m @25648.25 ord1
- **Q1/all/2026/(a) unknown-zone**: 2026-01-01 NY RN 1m @25400 ord1; 2026-01-01 NY RN 1m @25675 ord1; 2026-01-01 NY EQL 1m @25679.5 ord1; 2026-01-01 NY EQL 1m @25687.75 ord1; 2026-01-01 NY EQL 1m @25690.5 ord1
- **Q1/all/2026/(b) with-zone × 4h-with**: 2026-01-06 ASIA RTH-H 1m @25835 ord1; 2026-01-06 ASIA LDN-L 1m @25727 ord1; 2026-01-06 ASIA ONL 1m @25727 ord1; 2026-01-06 ASIA RN 1m @25775 ord1; 2026-01-06 ASIA EQH 1m @25775.5 ord1
- **Q1/all/2026/(b) with-zone × 4h-against**: 2026-01-07 LONDON RN 1m @25800 ord1; 2026-01-07 LONDON EQH 1m @25754 ord1; 2026-01-07 LONDON EQH 1m @25758.75 ord1; 2026-01-07 LONDON EQH 1m @25763 ord1; 2026-01-07 LONDON EQH 1m @25767.5 ord1
- **Q1/all/2026/(b) against-zone × 4h-with**: 2026-01-06 ASIA LDN-H 1m @25793.75 ord1; 2026-01-06 ASIA RN 1m @25675 ord1; 2026-01-06 ASIA RN 1m @25700 ord1; 2026-01-06 ASIA RN 1m @25725 ord1; 2026-01-06 ASIA RN 1m @25800 ord1
- **Q1/all/2026/(b) against-zone × 4h-against**: 2026-01-07 LONDON EQL 1m @25779 ord1; 2026-01-07 LONDON IFVG 1m @25779 ord1; 2026-01-07 LONDON IFVG 1m @25778.125 ord1; 2026-01-07 LONDON IFVG 1m @25778.25 ord1; 2026-01-07 ASIA RN 1m @25725 ord1
- **Q1/all/2026/(c) with-zone × D-with**: 2026-01-15 LONDON RN 1m @25800 ord1; 2026-01-15 LONDON EQH 1m @25803.75 ord1; 2026-01-15 LONDON EQH 1m @25804.75 ord1; 2026-01-15 LONDON EQL 1m @25801.25 ord1; 2026-01-15 LONDON EQL 1m @25803.25 ord1
- **Q1/all/2026/(c) with-zone × D-against**: 2026-01-15 LONDON AS-H 1m @25829.5 ord1; 2026-01-15 LONDON ONH 1m @25829.5 ord1; 2026-01-15 LONDON DEMAND 1m @25829.375 ord1; 2026-01-15 LONDON SUPPLY 1m @25872 ord1; 2026-01-15 LONDON OB 1m @25828.875 ord1
- **Q1/all/2026/(c) against-zone × D-with**: 2026-01-15 LONDON RN 1m @25825 ord1; 2026-01-15 LONDON EQH 1m @25795.75 ord1; 2026-01-15 LONDON EQH 1m @25824 ord1; 2026-01-15 LONDON EQH 1m @25826.75 ord1; 2026-01-15 LONDON EQH 1m @25827.75 ord1
- **Q1/all/2026/(c) against-zone × D-against**: 2026-01-15 LONDON EQH 1m @25830 ord1; 2026-01-15 LONDON EQH 1m @25844.25 ord1; 2026-01-15 LONDON EQL 1m @25842.75 ord1; 2026-01-15 LONDON SUPPLY 1m @25839.875 ord1; 2026-01-15 LONDON DEMAND 1m @25845.375 ord1
- **Q1/all/2026/4-way with-zone × 4h-with × D-with**: 2026-06-02 ASIA RN 1m @30700 ord1; 2026-06-02 ASIA OB 4h @30689.25 ord1; 2026-06-03 LONDON PDC 1m @30790 ord1; 2026-06-03 LONDON RTH-L 1m @30790 ord1; 2026-06-03 LONDON RN 1m @30475 ord1
- **Q1/all/2026/4-way with-zone × 4h-with × D-against**: 2026-01-19 NY PDC 1m @25384.75 ord1; 2026-01-19 NY RTH-H 1m @25418 ord1; 2026-01-19 NY LDN-H 1m @25288.75 ord1; 2026-01-19 NY RN 1m @25250 ord1; 2026-01-19 NY RN 1m @25275 ord1
- **Q1/all/2026/4-way with-zone × 4h-against × D-with**: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY RN 1m @25200 ord1; 2026-01-19 NY RN 1m @25225 ord1
- **Q1/all/2026/4-way with-zone × 4h-against × D-against**: 2026-06-02 ASIA RN 1m @30800 ord1; 2026-06-02 ASIA DEMAND 1m @30802.75 ord1; 2026-06-02 ASIA SUPPLY 1m @30812.75 ord1; 2026-06-02 ASIA IFVG 1m @30815.875 ord1; 2026-06-02 ASIA IFVG 1m @30801.75 ord1
- **Q1/all/2026/4-way against-zone × 4h-with × D-with**: 2026-06-02 NY RN 1m @30800 ord1; 2026-06-02 NY SUPPLY 1m @30885.625 ord1; 2026-06-02 NY DEMAND 1m @30883.875 ord1; 2026-06-02 NY OB 1m @30885.625 ord1; 2026-06-02 NY OB 1m @30883.875 ord1
- **Q1/all/2026/4-way against-zone × 4h-with × D-against**: 2026-01-19 NY PDL 1m @25236.25 ord1; 2026-01-19 NY RTH-L 1m @25236.25 ord1; 2026-01-19 NY AS-L 1m @25256.5 ord1; 2026-01-19 NY RN 1m @25325 ord1; 2026-01-19 NY EQH 1m @25255.25 ord1
- **Q1/all/2026/4-way against-zone × 4h-against × D-with**: 2026-05-20 LONDON RN 1m @29250 ord1; 2026-05-20 LONDON RN 1m @29275 ord1; 2026-05-20 LONDON EQH 1m @29265.5 ord1; 2026-05-20 LONDON EQH 1m @29284.75 ord1; 2026-05-20 LONDON EQL 1m @29252.25 ord1
- **Q1/all/2026/4-way against-zone × 4h-against × D-against**: 2026-06-02 ASIA IB-L 1m @30790 ord1; 2026-06-02 ASIA FVG 1m @30791 ord1; 2026-06-02 ASIA FVG 1m @30787.375 ord1; 2026-06-02 ASIA VWAP±2σ 1m @30794.435085872763 ord1; 2026-06-02 ASIA SWG-L 15m @30790 ord1
- **Q2/own/era/with-zone DEMAND 1h long**: 2022-04-12 LONDON DEMAND 1h @14016.25 ord1; 2022-04-13 NY DEMAND 1h @14016.25 ord1; 2022-04-19 NY DEMAND 1h @14016.25 ord1; 2022-04-20 NY DEMAND 1h @14016.25 ord1; 2022-04-20 NY DEMAND 1h @13915.5 ord1
- **Q2/own/era/with-zone EQH 1h short**: 2022-04-18 NY EQH 1h @14002.25 ord1; 2022-04-18 NY EQH 1h @14048.25 ord1; 2022-04-19 LONDON EQH 1h @14246.75 ord1; 2022-04-19 ASIA EQH 1h @14131.5 ord1; 2022-04-20 LONDON EQH 1h @14100 ord1
- **Q2/own/era/with-zone EQL 1h long**: 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-18 LONDON EQL 1h @13902.5 ord1; 2022-04-19 NY EQL 1h @14065 ord1; 2022-04-19 ASIA EQL 1h @14065 ord1; 2022-04-20 NY EQL 1h @13828 ord1
- **Q2/own/era/with-zone IFVG 1h long**: 2022-04-27 NY IFVG 1h @13066.25 ord1; 2022-04-28 LONDON IFVG 1h @13360.875 ord1; 2022-04-28 NY IFVG 1h @13066.25 ord1; 2022-05-02 NY IFVG 1h @13066.25 ord1; 2022-05-03 NY IFVG 1h @13066.25 ord1
- **Q2/own/era/with-zone IFVG 1h short**: 2022-04-12 LONDON IFVG 1h @14073.75 ord1; 2022-04-12 NY IFVG 1h @14073.75 ord1; 2022-04-18 NY IFVG 1h @14073.75 ord1; 2022-05-02 NY IFVG 1h @13110.25 ord1; 2022-05-03 NY IFVG 1h @13110.25 ord1
- **Q2/own/era/with-zone OB 1h long**: 2022-04-12 LONDON OB 1h @14025.375 ord1; 2022-04-13 NY OB 1h @14025.375 ord1; 2022-04-13 NY OB 1h @13958.75 ord1; 2022-04-18 LONDON OB 1h @13832 ord1; 2022-04-19 NY OB 1h @14025.375 ord1
- **Q2/own/era/with-zone OB 1h short**: 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-12 NY OB 1h @14152.5 ord1; 2022-04-12 NY OB 1h @14142.5 ord1; 2022-04-17 NY OB 1h @14000.25 ord1
- **Q2/own/era/with-zone SUPPLY 1h short**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-17 NY SUPPLY 1h @13887.5 ord1; 2022-04-18 NY SUPPLY 1h @14085.375 ord1; 2022-04-18 NY SUPPLY 1h @14190.75 ord1; 2022-04-19 LONDON SUPPLY 1h @14190.75 ord1
- **Q2/noconflict/era/with-zone DEMAND 1h long**: 2022-04-12 LONDON EQH 1m @13984.5 ord1; 2022-04-12 LONDON EQH 1m @13986.5 ord1; 2022-04-12 LONDON EQH 1m @13995.5 ord1; 2022-04-12 LONDON EQH 1m @14010.5 ord1; 2022-04-12 LONDON EQH 1m @14012 ord1
- **Q2/noconflict/era/with-zone EQH 1h short**: 2022-04-18 NY RN 1m @14050 ord1; 2022-04-18 NY EQH 15m @14049.75 ord1; 2022-04-18 NY OB 15m @14051.5 ord1; 2022-04-19 LONDON RN 1m @14250 ord1; 2022-04-19 LONDON EQH 1h @14246.75 ord1
- **Q2/noconflict/era/with-zone EQL 1h long**: 2022-04-11 NY PDL 1m @13903.5 ord1; 2022-04-11 NY PDC 1m @13906.25 ord1; 2022-04-11 NY AS-L 1m @13902.5 ord1; 2022-04-11 NY ONL 1m @13902.5 ord1; 2022-04-11 NY RN 1m @13900 ord1
- **Q2/noconflict/era/with-zone FVG 1h long**: 2022-04-27 ASIA AS-H 1m @13244.5 ord1; 2022-04-27 ASIA RN 1m @13225 ord1; 2022-04-27 ASIA RN 1m @13250 ord1; 2022-04-27 ASIA EQH 1m @13222.5 ord1; 2022-04-27 ASIA EQH 1m @13225 ord1
- **Q2/noconflict/era/with-zone FVG 1h short**: 2022-04-21 LONDON RN 1m @13775 ord1; 2022-04-21 LONDON FVG 1m @13773.5 ord1; 2022-04-21 LONDON EQH 15m @13773.25 ord1; 2022-04-21 NY RN 1m @13775 ord1; 2022-04-21 NY EQH 1m @13771.75 ord1
- **Q2/noconflict/era/with-zone IFVG 1h long**: 2022-04-28 LONDON RN 1m @13375 ord1; 2022-04-28 LONDON EQH 1m @13368 ord1; 2022-04-28 LONDON EQL 1m @13369 ord1; 2022-04-28 LONDON EQL 1m @13370 ord1; 2022-04-28 LONDON DEMAND 1m @13363.875 ord1
- **Q2/noconflict/era/with-zone IFVG 1h short**: 2022-04-12 LONDON RN 1m @14075 ord1; 2022-04-12 LONDON EQL 1m @14065.5 ord1; 2022-04-12 LONDON EQL 1m @14066.75 ord1; 2022-04-12 LONDON EQL 1m @14067.75 ord1; 2022-04-12 LONDON EQL 1m @14073.5 ord1
- **Q2/noconflict/era/with-zone OB 1h long**: 2022-04-13 NY RN 1m @13925 ord1; 2022-04-13 NY RN 1m @13950 ord1; 2022-04-13 NY RN 1m @13975 ord1; 2022-04-13 NY SUPPLY 1m @13959.625 ord1; 2022-04-13 NY SUPPLY 1m @13949.875 ord1
- **Q2/noconflict/era/with-zone OB 1h short**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q2/noconflict/era/with-zone SUPPLY 1h short**: 2022-04-11 LONDON RN 1m @14100 ord1; 2022-04-11 LONDON RN 1m @14125 ord1; 2022-04-11 LONDON EQH 1m @14086.5 ord1; 2022-04-11 LONDON EQH 1m @14095 ord1; 2022-04-11 LONDON EQH 1m @14099.5 ord1
- **Q2/all/era/with-zone DEMAND 1h long**: 2022-04-12 LONDON EQH 1m @13984.5 ord1; 2022-04-12 LONDON EQH 1m @13986.5 ord1; 2022-04-12 LONDON EQH 1m @13995.5 ord1; 2022-04-12 LONDON EQH 1m @14010.5 ord1; 2022-04-12 LONDON EQH 1m @14012 ord1
- **Q2/all/era/with-zone EQH 1h short**: 2022-04-18 NY RTH-H 1m @14002.25 ord1; 2022-04-18 NY RN 1m @14000 ord1; 2022-04-18 NY RN 1m @14050 ord1; 2022-04-18 NY EQH 1m @14000.25 ord1; 2022-04-18 NY EQH 1m @14002.25 ord1
- **Q2/all/era/with-zone EQL 1h long**: 2022-04-11 NY PDL 1m @13903.5 ord1; 2022-04-11 NY PDC 1m @13906.25 ord1; 2022-04-11 NY AS-L 1m @13902.5 ord1; 2022-04-11 NY ONL 1m @13902.5 ord1; 2022-04-11 NY RN 1m @13900 ord1
- **Q2/all/era/with-zone FVG 1h long**: 2022-04-27 ASIA AS-H 1m @13244.5 ord1; 2022-04-27 ASIA RN 1m @13225 ord1; 2022-04-27 ASIA RN 1m @13250 ord1; 2022-04-27 ASIA EQH 1m @13222.5 ord1; 2022-04-27 ASIA EQH 1m @13225 ord1
- **Q2/all/era/with-zone FVG 1h short**: 2022-04-21 LONDON RN 1m @13775 ord1; 2022-04-21 LONDON FVG 1m @13773.5 ord1; 2022-04-21 LONDON EQH 15m @13773.25 ord1; 2022-04-21 NY RN 1m @13775 ord1; 2022-04-21 NY EQH 1m @13771.75 ord1
- **Q2/all/era/with-zone IFVG 1h long**: 2022-04-27 NY PDC 1m @13197.5 ord1; 2022-04-27 NY RN 1m @13200 ord1; 2022-04-27 NY EQH 1m @13062.25 ord1; 2022-04-27 NY EQH 1m @13064 ord1; 2022-04-27 NY EQH 1m @13065.75 ord1
- **Q2/all/era/with-zone IFVG 1h short**: 2022-04-12 LONDON RN 1m @14075 ord1; 2022-04-12 LONDON EQL 1m @14065.5 ord1; 2022-04-12 LONDON EQL 1m @14066.75 ord1; 2022-04-12 LONDON EQL 1m @14067.75 ord1; 2022-04-12 LONDON EQL 1m @14073.5 ord1
- **Q2/all/era/with-zone OB 1h long**: 2022-04-12 LONDON PDC 1m @14030 ord1; 2022-04-12 LONDON RN 1m @14025 ord1; 2022-04-12 LONDON EQH 1m @14022 ord1; 2022-04-12 LONDON EQH 1m @14024.75 ord1; 2022-04-12 LONDON EQH 1m @14025.5 ord1
- **Q2/all/era/with-zone OB 1h short**: 2022-04-10 NY EQH 1m @14182.5 ord1; 2022-04-10 NY EQH 1m @14184.75 ord1; 2022-04-10 NY EQH 1m @14186 ord1; 2022-04-10 NY EQL 1m @14186 ord1; 2022-04-10 NY EQL 1m @14196.75 ord1
- **Q2/all/era/with-zone OB 4h long**: 2026-06-02 ASIA RN 1m @30700 ord1; 2026-06-02 ASIA OB 4h @30689.25 ord1; 2026-06-03 LONDON RN 1m @30700 ord1; 2026-06-03 LONDON GAP 1m @30688.5 ord1; 2026-06-03 LONDON EQL 1m @30680 ord1
- **Q2/all/era/with-zone SUPPLY 1h short**: 2022-04-11 LONDON RN 1m @14100 ord1; 2022-04-11 LONDON RN 1m @14125 ord1; 2022-04-11 LONDON EQH 1m @14086.5 ord1; 2022-04-11 LONDON EQH 1m @14095 ord1; 2022-04-11 LONDON EQH 1m @14099.5 ord1
- **Q3/1h/2022/fresh**: 2022-04-17 NY SUPPLY 1h @13887.5 ord1; 2022-04-17 NY OB 1h @13887.5 ord1; 2022-04-17 NY OB 1h @13887.5 ord1; 2022-04-18 LONDON EQL 1h @13902.5 ord1; 2022-04-18 NY EQH 1h @14048.25 ord1
- **Q3/1h/2022/tested-1**: 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-17 NY OB 1h @14000.25 ord1; 2022-04-18 ASIA EQH 1h @14048.25 ord1
- **Q3/1h/2022/tested-2**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-12 LONDON IFVG 1h @14073.75 ord1; 2022-04-12 NY IFVG 1h @14073.75 ord1
- **Q3/1h/2022/stale**: 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1; 2022-04-12 LONDON OB 1h @14025.375 ord1; 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1
- **Q3/1h/2023/fresh**: 2023-01-01 ASIA EQH 1h @11071.5 ord1; 2023-01-01 ASIA EQH 1h @11080 ord1; 2023-01-02 LONDON EQH 1h @11080 ord1; 2023-01-02 NY EQH 1h @11177 ord1; 2023-01-02 NY EQL 1h @10870.5 ord1
- **Q3/1h/2023/tested-1**: 2023-01-03 LONDON EQH 1h @10972.75 ord1; 2023-01-04 LONDON DEMAND 1h @10940.625 ord1; 2023-01-04 LONDON OB 1h @10940.625 ord1; 2023-01-05 LONDON EQL 1h @10870.5 ord1; 2023-01-05 LONDON FVG 1h @10966.75 ord1
- **Q3/1h/2023/tested-2**: 2023-01-01 ASIA FVG 1h @10964 ord1; 2023-01-02 NY EQH 1h @11071.5 ord1; 2023-01-02 NY EQH 1h @11080 ord1; 2023-01-02 NY EQL 1h @11002.25 ord1; 2023-01-02 NY DEMAND 1h @10898 ord1
- **Q3/1h/2023/stale**: 2023-01-01 ASIA IFVG 1h @11061 ord1; 2023-01-01 ASIA IFVG 1h @10953 ord1; 2023-01-01 ASIA OB 1h @11079 ord1; 2023-01-01 ASIA OB 1h @10991.25 ord1; 2023-01-02 LONDON IFVG 1h @11151.375 ord1
- **Q3/1h/2024/fresh**: 2024-01-01 LONDON EQH 1h @16887.25 ord1; 2024-01-01 LONDON EQH 1h @16974.25 ord1; 2024-01-01 LONDON EQL 1h @16896 ord1; 2024-01-01 LONDON EQL 1h @16911 ord1; 2024-01-01 LONDON EQL 1h @16961.5 ord1
- **Q3/1h/2024/tested-1**: 2024-01-01 LONDON EQH 1h @17012.75 ord1; 2024-01-01 LONDON DEMAND 1h @16876.625 ord1; 2024-01-02 NY EQH 1h @16618 ord1; 2024-01-02 NY EQL 1h @16622 ord1; 2024-01-04 NY OB 1h @16485.125 ord1
- **Q3/1h/2024/tested-2**: 2024-01-01 LONDON OB 1h @16869.25 ord1; 2024-01-01 LONDON OB 1h @16852.375 ord1; 2024-01-01 LONDON OB 1h @16852.375 ord1; 2024-01-01 LONDON OB 1h @16909.875 ord1; 2024-01-02 LONDON DEMAND 1h @16594.125 ord1
- **Q3/1h/2024/stale**: 2024-01-01 LONDON SUPPLY 1h @16862.625 ord1; 2024-01-01 LONDON SUPPLY 1h @16891.125 ord1; 2024-01-01 LONDON DEMAND 1h @16886.25 ord1; 2024-01-01 LONDON DEMAND 1h @16939.5 ord1; 2024-01-01 LONDON IFVG 1h @16836.5 ord1
- **Q3/1h/2025/fresh**: 2025-01-01 LONDON EQL 1h @21428 ord1; 2025-01-02 NY EQL 1h @21310.25 ord1; 2025-01-02 NY EQL 1h @21428 ord1; 2025-01-05 LONDON EQH 1h @21742.5 ord1; 2025-01-05 NY EQH 1h @21812.5 ord1
- **Q3/1h/2025/tested-1**: 2025-01-01 NY EQL 1h @21310.25 ord1; 2025-01-01 NY EQL 1h @21428 ord1; 2025-01-02 NY SUPPLY 1h @21328.625 ord1; 2025-01-02 NY OB 1h @21420.25 ord1; 2025-01-05 NY EQH 1h @21742.5 ord1
- **Q3/1h/2025/tested-2**: 2025-01-01 NY DEMAND 1h @21077.625 ord1; 2025-01-01 NY OB 1h @21098.625 ord1; 2025-01-01 NY OB 1h @21098.625 ord1; 2025-01-01 NY OB 1h @21098.625 ord1; 2025-01-06 NY EQH 1h @21742.5 ord1
- **Q3/1h/2025/stale**: 2025-01-01 NY DEMAND 1h @21234.75 ord1; 2025-01-01 NY IFVG 1h @21406.5 ord1; 2025-01-02 LONDON DEMAND 1h @21234.75 ord1; 2025-01-02 NY DEMAND 1h @21382.875 ord1; 2025-01-02 NY IFVG 1h @21406.5 ord1
- **Q3/1h/2026/fresh**: 2026-01-01 LONDON EQH 1h @25728.75 ord1; 2026-01-01 LONDON EQL 1h @25656 ord1; 2026-01-01 LONDON EQL 1h @25672.25 ord1; 2026-01-01 LONDON EQL 1h @25695.5 ord1; 2026-01-01 NY EQH 1h @25755 ord1
- **Q3/1h/2026/tested-1**: 2026-01-01 LONDON EQL 1h @25647.75 ord1; 2026-01-01 NY EQH 1h @25728.75 ord1; 2026-01-01 NY EQL 1h @25486 ord1; 2026-01-01 NY EQL 1h @25647.75 ord1; 2026-01-01 NY EQL 1h @25656 ord1
- **Q3/1h/2026/tested-2**: 2026-01-01 NY EQL 1h @25695.5 ord1; 2026-01-01 NY FVG 1h @25402.375 ord1; 2026-01-01 NY FVG 1h @25676.25 ord1; 2026-01-03 ASIA EQH 1h @25417 ord1; 2026-01-03 ASIA EQL 1h @25486 ord1
- **Q3/1h/2026/stale**: 2026-01-01 LONDON OB 1h @25668.75 ord1; 2026-01-01 NY SUPPLY 1h @25278.375 ord1; 2026-01-01 NY SUPPLY 1h @25360 ord1; 2026-01-01 NY SUPPLY 1h @25733 ord1; 2026-01-01 NY DEMAND 1h @25710.5 ord1
- **Q3/1h/era/fresh**: 2022-04-17 NY SUPPLY 1h @13887.5 ord1; 2022-04-17 NY OB 1h @13887.5 ord1; 2022-04-17 NY OB 1h @13887.5 ord1; 2022-04-18 LONDON EQL 1h @13902.5 ord1; 2022-04-18 NY EQH 1h @14048.25 ord1
- **Q3/1h/era/tested-1**: 2022-04-11 LONDON OB 1h @14082.5 ord1; 2022-04-11 NY EQL 1h @13902.5 ord1; 2022-04-11 NY OB 1h @14212.75 ord1; 2022-04-17 NY OB 1h @14000.25 ord1; 2022-04-18 ASIA EQH 1h @14048.25 ord1
- **Q3/1h/era/tested-2**: 2022-04-11 LONDON SUPPLY 1h @14085.375 ord1; 2022-04-11 NY SUPPLY 1h @14148.25 ord1; 2022-04-11 NY OB 1h @14082.5 ord1; 2022-04-12 LONDON IFVG 1h @14073.75 ord1; 2022-04-12 NY IFVG 1h @14073.75 ord1
- **Q3/1h/era/stale**: 2022-04-11 NY SUPPLY 1h @14085.375 ord1; 2022-04-12 LONDON DEMAND 1h @14016.25 ord1; 2022-04-12 LONDON OB 1h @14025.375 ord1; 2022-04-12 NY DEMAND 1h @14016.25 ord1; 2022-04-12 NY OB 1h @14025.375 ord1
- **Q3/4h/2026/fresh**: 2026-06-02 ASIA EQH 4h @30668 ord1; 2026-06-04 NY EQH 4h @29685 ord1; 2026-06-04 NY EQH 4h @29755 ord1; 2026-06-04 NY EQL 4h @29341.5 ord1; 2026-06-04 NY EQL 4h @29714.75 ord1
- **Q3/4h/2026/tested-1**: 2026-06-03 LONDON EQH 4h @30668 ord1; 2026-06-03 NY EQH 4h @30668 ord1; 2026-06-03 ASIA DEMAND 4h @30558.625 ord1; 2026-06-03 ASIA OB 4h @30558.625 ord1; 2026-06-06 ASIA EQL 4h @29341.5 ord1
- **Q3/4h/2026/tested-2**: 2026-06-03 ASIA EQH 4h @30668 ord1; 2026-06-04 NY DEMAND 4h @30273.5 ord1; 2026-06-04 NY OB 4h @30206 ord1; 2026-06-07 NY FVG 4h @29910.75 ord1; 2026-06-08 LONDON FVG 4h @29910.75 ord1
- **Q3/4h/2026/stale**: 2026-06-02 ASIA OB 4h @30689.25 ord1; 2026-06-03 LONDON OB 4h @30689.25 ord1; 2026-06-03 NY OB 4h @30689.25 ord1; 2026-06-03 ASIA OB 4h @30689.25 ord1; 2026-06-10 NY FVG 4h @29694.5 ord1
- **Q3/4h/era/fresh**: 2026-06-02 ASIA EQH 4h @30668 ord1; 2026-06-04 NY EQH 4h @29685 ord1; 2026-06-04 NY EQH 4h @29755 ord1; 2026-06-04 NY EQL 4h @29341.5 ord1; 2026-06-04 NY EQL 4h @29714.75 ord1
- **Q3/4h/era/tested-1**: 2026-06-03 LONDON EQH 4h @30668 ord1; 2026-06-03 NY EQH 4h @30668 ord1; 2026-06-03 ASIA DEMAND 4h @30558.625 ord1; 2026-06-03 ASIA OB 4h @30558.625 ord1; 2026-06-06 ASIA EQL 4h @29341.5 ord1
- **Q3/4h/era/tested-2**: 2026-06-03 ASIA EQH 4h @30668 ord1; 2026-06-04 NY DEMAND 4h @30273.5 ord1; 2026-06-04 NY OB 4h @30206 ord1; 2026-06-07 NY FVG 4h @29910.75 ord1; 2026-06-08 LONDON FVG 4h @29910.75 ord1
- **Q3/4h/era/stale**: 2026-06-02 ASIA OB 4h @30689.25 ord1; 2026-06-03 LONDON OB 4h @30689.25 ord1; 2026-06-03 NY OB 4h @30689.25 ord1; 2026-06-03 ASIA OB 4h @30689.25 ord1; 2026-06-10 NY FVG 4h @29694.5 ord1
- **Q3/1d/2026/fresh**: 2026-07-15 ASIA EQL 1d @28910.25 ord1; 2026-07-16 NY EQL 1d @28910.25 ord1; 2026-07-18 ASIA EQL 1d @28910.25 ord1; 2026-07-19 NY EQL 1d @28910.25 ord1; 2026-07-19 ASIA EQL 1d @28910.25 ord1
- **Q3/1d/2026/tested-1**: 2026-06-14 NY SUPPLY 1d @30775.875 ord1; 2026-06-14 NY OB 1d @30672.625 ord1; 2026-06-15 NY SUPPLY 1d @30775.875 ord1; 2026-06-15 NY OB 1d @30672.625 ord1; 2026-06-17 NY OB 1d @30672.625 ord1
- **Q3/1d/2026/tested-2**: 2026-08-19 LONDON FVG 1d @29247.875 ord1; 2026-08-19 NY FVG 1d @29247.875 ord1; 2026-08-20 NY FVG 1d @29247.875 ord1; 2026-08-22 ASIA FVG 1d @29247.875 ord1; 2026-08-23 LONDON FVG 1d @29247.875 ord1
- **Q3/1d/2026/stale**: 2026-07-20 NY OB 1d @29110.75 ord1; 2026-07-20 ASIA OB 1d @29110.75 ord1; 2026-07-21 LONDON OB 1d @29110.75 ord1; 2026-07-21 NY OB 1d @29110.75 ord1; 2026-07-21 ASIA OB 1d @29110.75 ord1
- **Q3/1d/era/fresh**: 2026-07-15 ASIA EQL 1d @28910.25 ord1; 2026-07-16 NY EQL 1d @28910.25 ord1; 2026-07-18 ASIA EQL 1d @28910.25 ord1; 2026-07-19 NY EQL 1d @28910.25 ord1; 2026-07-19 ASIA EQL 1d @28910.25 ord1
- **Q3/1d/era/tested-1**: 2026-06-14 NY SUPPLY 1d @30775.875 ord1; 2026-06-14 NY OB 1d @30672.625 ord1; 2026-06-15 NY SUPPLY 1d @30775.875 ord1; 2026-06-15 NY OB 1d @30672.625 ord1; 2026-06-17 NY OB 1d @30672.625 ord1
- **Q3/1d/era/tested-2**: 2026-08-19 LONDON FVG 1d @29247.875 ord1; 2026-08-19 NY FVG 1d @29247.875 ord1; 2026-08-20 NY FVG 1d @29247.875 ord1; 2026-08-22 ASIA FVG 1d @29247.875 ord1; 2026-08-23 LONDON FVG 1d @29247.875 ord1
- **Q3/1d/era/stale**: 2026-07-20 NY OB 1d @29110.75 ord1; 2026-07-20 ASIA OB 1d @29110.75 ord1; 2026-07-21 LONDON OB 1d @29110.75 ord1; 2026-07-21 NY OB 1d @29110.75 ord1; 2026-07-21 ASIA OB 1d @29110.75 ord1
- **Q5/own/era/with-zone * ***: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q5/own/era/with-zone long ***: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q5/own/era/with-zone long 1h**: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q5/own/era/with-zone short ***: 2025-06-30 ASIA EQH 1h @22762.5 ord1; 2026-07-02 LONDON OB 4h @29922.375 ord1; 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-02 NY OB 4h @29922.375 ord1; 2026-07-04 ASIA OB 4h @29922.375 ord1
- **Q5/own/era/with-zone short 1h**: 2025-06-30 ASIA EQH 1h @22762.5 ord1; 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-04 ASIA EQH 1h @29958 ord1; 2026-07-04 ASIA FVG 1h @29929.5 ord1; 2026-08-05 ASIA EQH 1h @29541 ord1
- **Q5/own/era/with-zone short 4h**: 2026-07-02 LONDON OB 4h @29922.375 ord1; 2026-07-02 NY OB 4h @29922.375 ord1; 2026-07-04 ASIA OB 4h @29922.375 ord1
- **Q5/own/era/against-zone * ***: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q5/own/era/against-zone long ***: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q5/own/era/against-zone long 1h**: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q5/own/era/against-zone short ***: 2025-06-30 LONDON EQL 1h @22845.5 ord1; 2026-07-02 LONDON IFVG 4h @29910.75 ord1; 2026-07-02 LONDON EQL 1h @29922 ord1; 2026-07-02 NY EQL 1h @29922 ord1; 2026-07-02 NY IFVG 1h @29946.625 ord1
- **Q5/own/era/against-zone short 1h**: 2025-06-30 LONDON EQL 1h @22845.5 ord1; 2026-07-02 LONDON EQL 1h @29922 ord1; 2026-07-02 NY EQL 1h @29922 ord1; 2026-07-02 NY IFVG 1h @29946.625 ord1; 2026-07-04 ASIA EQL 1h @29922 ord1
- **Q5/own/era/against-zone short 4h**: 2026-07-02 LONDON IFVG 4h @29910.75 ord1; 2026-08-05 ASIA EQL 4h @29577.25 ord1
- **Q5/own/2025/with-zone * ***: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q5/own/2025/with-zone long ***: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q5/own/2025/with-zone long 1h**: 2025-05-22 LONDON EQL 1h @20818.25 ord1; 2025-05-22 LONDON DEMAND 1h @20769.75 ord1; 2025-05-22 LONDON OB 1h @20780.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1; 2025-05-22 LONDON OB 1h @20851.5 ord1
- **Q5/own/2025/with-zone short ***: 2025-06-30 ASIA EQH 1h @22762.5 ord1
- **Q5/own/2025/with-zone short 1h**: 2025-06-30 ASIA EQH 1h @22762.5 ord1
- **Q5/own/2025/against-zone * ***: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q5/own/2025/against-zone long ***: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q5/own/2025/against-zone long 1h**: 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1; 2025-05-22 LONDON OB 1h @20935.75 ord1; 2025-05-25 LONDON EQH 1h @21233.5 ord1; 2025-05-26 NY EQH 1h @21208.5 ord1
- **Q5/own/2025/against-zone short ***: 2025-06-30 LONDON EQL 1h @22845.5 ord1
- **Q5/own/2025/against-zone short 1h**: 2025-06-30 LONDON EQL 1h @22845.5 ord1
- **Q5/own/2026/with-zone * ***: 2026-01-19 NY IFVG 1h @25183.875 ord1; 2026-01-20 ASIA EQL 1h @25520 ord1; 2026-07-02 LONDON OB 4h @29922.375 ord1; 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-02 NY OB 4h @29922.375 ord1
- **Q5/own/2026/with-zone long ***: 2026-01-19 NY IFVG 1h @25183.875 ord1; 2026-01-20 ASIA EQL 1h @25520 ord1
- **Q5/own/2026/with-zone long 1h**: 2026-01-19 NY IFVG 1h @25183.875 ord1; 2026-01-20 ASIA EQL 1h @25520 ord1
- **Q5/own/2026/with-zone short ***: 2026-07-02 LONDON OB 4h @29922.375 ord1; 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-02 NY OB 4h @29922.375 ord1; 2026-07-04 ASIA OB 4h @29922.375 ord1; 2026-07-04 ASIA EQH 1h @29958 ord1
- **Q5/own/2026/with-zone short 1h**: 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-04 ASIA EQH 1h @29958 ord1; 2026-07-04 ASIA FVG 1h @29929.5 ord1; 2026-08-05 ASIA EQH 1h @29541 ord1
- **Q5/own/2026/with-zone short 4h**: 2026-07-02 LONDON OB 4h @29922.375 ord1; 2026-07-02 NY OB 4h @29922.375 ord1; 2026-07-04 ASIA OB 4h @29922.375 ord1
- **Q5/own/2026/against-zone * ***: 2026-07-02 LONDON IFVG 4h @29910.75 ord1; 2026-07-02 LONDON EQL 1h @29922 ord1; 2026-07-02 NY EQL 1h @29922 ord1; 2026-07-02 NY IFVG 1h @29946.625 ord1; 2026-07-04 ASIA EQL 1h @29922 ord1
- **Q5/own/2026/against-zone long ***: —
- **Q5/own/2026/against-zone long 1h**: —
- **Q5/own/2026/against-zone short ***: 2026-07-02 LONDON IFVG 4h @29910.75 ord1; 2026-07-02 LONDON EQL 1h @29922 ord1; 2026-07-02 NY EQL 1h @29922 ord1; 2026-07-02 NY IFVG 1h @29946.625 ord1; 2026-07-04 ASIA EQL 1h @29922 ord1
- **Q5/own/2026/against-zone short 1h**: 2026-07-02 LONDON EQL 1h @29922 ord1; 2026-07-02 NY EQL 1h @29922 ord1; 2026-07-02 NY IFVG 1h @29946.625 ord1; 2026-07-04 ASIA EQL 1h @29922 ord1; 2026-07-04 ASIA DEMAND 1h @29959.875 ord1
- **Q5/own/2026/against-zone short 4h**: 2026-07-02 LONDON IFVG 4h @29910.75 ord1; 2026-08-05 ASIA EQL 4h @29577.25 ord1
- **Q5/noconflict/era/with-zone * ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/noconflict/era/with-zone long ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/noconflict/era/with-zone long 1h**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/noconflict/era/with-zone short ***: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/noconflict/era/with-zone short 1h**: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/noconflict/era/against-zone * ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/noconflict/era/against-zone long ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/noconflict/era/against-zone long 1h**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/noconflict/era/against-zone short ***: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/noconflict/era/against-zone short 1h**: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/noconflict/2025/with-zone * ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/noconflict/2025/with-zone long ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/noconflict/2025/with-zone long 1h**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/noconflict/2025/with-zone short ***: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/noconflict/2025/with-zone short 1h**: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/noconflict/2025/against-zone * ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/noconflict/2025/against-zone long ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/noconflict/2025/against-zone long 1h**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/noconflict/2025/against-zone short ***: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/noconflict/2025/against-zone short 1h**: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/noconflict/2026/with-zone * ***: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY EQH 1m @25135.75 ord1; 2026-01-19 NY EQL 1m @25166.5 ord1
- **Q5/noconflict/2026/with-zone long ***: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY EQH 1m @25135.75 ord1; 2026-01-19 NY EQL 1m @25166.5 ord1
- **Q5/noconflict/2026/with-zone long 1h**: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY EQH 1m @25135.75 ord1; 2026-01-19 NY EQL 1m @25166.5 ord1
- **Q5/noconflict/2026/against-zone * ***: 2026-05-20 ASIA RN 1m @29450 ord1; 2026-05-20 ASIA IB-H 1m @29444.5 ord1; 2026-05-20 ASIA DEMAND 1m @29450.375 ord1; 2026-05-20 ASIA IFVG 1m @29445.25 ord1; 2026-05-20 ASIA IFVG 1m @29447.75 ord1
- **Q5/noconflict/2026/against-zone long ***: 2026-05-20 ASIA RN 1m @29450 ord1; 2026-05-20 ASIA IB-H 1m @29444.5 ord1; 2026-05-20 ASIA DEMAND 1m @29450.375 ord1; 2026-05-20 ASIA IFVG 1m @29445.25 ord1; 2026-05-20 ASIA IFVG 1m @29447.75 ord1
- **Q5/noconflict/2026/against-zone long 1h**: 2026-05-20 ASIA RN 1m @29450 ord1; 2026-05-20 ASIA IB-H 1m @29444.5 ord1; 2026-05-20 ASIA DEMAND 1m @29450.375 ord1; 2026-05-20 ASIA IFVG 1m @29445.25 ord1; 2026-05-20 ASIA IFVG 1m @29447.75 ord1
- **Q5/all/era/with-zone * ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/all/era/with-zone long ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/all/era/with-zone long 1h**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/all/era/with-zone short ***: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/all/era/with-zone short 1h**: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/all/era/with-zone short 4h**: 2026-07-02 LONDON IFVG 1m @29869.625 ord1; 2026-07-02 LONDON FVG 1m @29868.625 ord1; 2026-07-02 LONDON IFVG 1m @29868.125 ord1; 2026-07-02 LONDON OB 1m @29868.125 ord1; 2026-07-02 LONDON OB 1m @29867.625 ord1
- **Q5/all/era/against-zone * ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/all/era/against-zone long ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/all/era/against-zone long 1h**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/all/era/against-zone short ***: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/all/era/against-zone short 1h**: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/all/era/against-zone short 4h**: 2026-07-02 LONDON SUPPLY 1m @29911.875 ord1; 2026-07-02 LONDON DEMAND 1m @29912.75 ord1; 2026-07-02 LONDON FVG 1m @29911.125 ord1; 2026-07-02 LONDON OB 1m @29913.875 ord1; 2026-07-02 LONDON OB 1m @29915.375 ord1
- **Q5/all/2025/with-zone * ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/all/2025/with-zone long ***: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/all/2025/with-zone long 1h**: 2025-05-22 LONDON RTH-L 1m @21113.75 ord1; 2025-05-22 LONDON RN 1m @20800 ord1; 2025-05-22 LONDON RN 1m @20825 ord1; 2025-05-22 LONDON RN 1m @20850 ord1; 2025-05-22 LONDON RN 1m @20875 ord1
- **Q5/all/2025/with-zone short ***: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/all/2025/with-zone short 1h**: 2025-06-30 NY EQL 1m @22813 ord1; 2025-06-30 NY EQL 1m @22814 ord1; 2025-06-30 NY EQL 1m @22815.25 ord1; 2025-06-30 NY EQL 1m @22818.25 ord1; 2025-06-30 NY EQL 1m @22819.75 ord1
- **Q5/all/2025/against-zone * ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/all/2025/against-zone long ***: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/all/2025/against-zone long 1h**: 2025-05-22 LONDON RN 1m @20925 ord1; 2025-05-22 LONDON RN 1m @20950 ord1; 2025-05-22 LONDON RN 1m @21000 ord1; 2025-05-22 LONDON EQH 1h @20996 ord1; 2025-05-22 LONDON SUPPLY 1h @20935.75 ord1
- **Q5/all/2025/against-zone short ***: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/all/2025/against-zone short 1h**: 2025-06-30 LONDON EQH 1m @22843.25 ord1; 2025-06-30 LONDON EQH 1m @22845.5 ord1; 2025-06-30 LONDON EQH 1m @22846.5 ord1; 2025-06-30 LONDON EQH 1m @22847.25 ord1; 2025-06-30 LONDON EQL 1m @22846.5 ord1
- **Q5/all/2026/with-zone * ***: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY RN 1m @25200 ord1; 2026-01-19 NY RN 1m @25225 ord1
- **Q5/all/2026/with-zone long ***: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY RN 1m @25200 ord1; 2026-01-19 NY RN 1m @25225 ord1
- **Q5/all/2026/with-zone long 1h**: 2026-01-19 NY RN 1m @25125 ord1; 2026-01-19 NY RN 1m @25150 ord1; 2026-01-19 NY RN 1m @25175 ord1; 2026-01-19 NY RN 1m @25200 ord1; 2026-01-19 NY RN 1m @25225 ord1
- **Q5/all/2026/with-zone short ***: 2026-07-02 LONDON RN 1m @29900 ord1; 2026-07-02 LONDON EQL 1m @29901 ord1; 2026-07-02 LONDON DEMAND 1m @29899.625 ord1; 2026-07-02 LONDON IFVG 1m @29869.625 ord1; 2026-07-02 LONDON FVG 1m @29868.625 ord1
- **Q5/all/2026/with-zone short 1h**: 2026-07-02 LONDON RN 1m @29900 ord1; 2026-07-02 LONDON EQL 1m @29901 ord1; 2026-07-02 LONDON DEMAND 1m @29899.625 ord1; 2026-07-02 LONDON OB 1h @29897.875 ord1; 2026-07-02 NY DEMAND 1m @29934.125 ord1
- **Q5/all/2026/with-zone short 4h**: 2026-07-02 LONDON IFVG 1m @29869.625 ord1; 2026-07-02 LONDON FVG 1m @29868.625 ord1; 2026-07-02 LONDON IFVG 1m @29868.125 ord1; 2026-07-02 LONDON OB 1m @29868.125 ord1; 2026-07-02 LONDON OB 1m @29867.625 ord1
- **Q5/all/2026/against-zone * ***: 2026-05-20 LONDON RN 1m @29250 ord1; 2026-05-20 LONDON RN 1m @29275 ord1; 2026-05-20 LONDON EQH 1m @29265.5 ord1; 2026-05-20 LONDON EQH 1m @29284.75 ord1; 2026-05-20 LONDON EQL 1m @29252.25 ord1
- **Q5/all/2026/against-zone long ***: 2026-05-20 LONDON RN 1m @29250 ord1; 2026-05-20 LONDON RN 1m @29275 ord1; 2026-05-20 LONDON EQH 1m @29265.5 ord1; 2026-05-20 LONDON EQH 1m @29284.75 ord1; 2026-05-20 LONDON EQL 1m @29252.25 ord1
- **Q5/all/2026/against-zone long 1h**: 2026-05-20 LONDON RN 1m @29250 ord1; 2026-05-20 LONDON RN 1m @29275 ord1; 2026-05-20 LONDON EQH 1m @29265.5 ord1; 2026-05-20 LONDON EQH 1m @29284.75 ord1; 2026-05-20 LONDON EQL 1m @29252.25 ord1
- **Q5/all/2026/against-zone short ***: 2026-07-02 LONDON RN 1m @29850 ord1; 2026-07-02 LONDON SUPPLY 1m @29911.875 ord1; 2026-07-02 LONDON SUPPLY 1m @29851.625 ord1; 2026-07-02 LONDON DEMAND 1m @29912.75 ord1; 2026-07-02 LONDON IFVG 1m @29921 ord1
- **Q5/all/2026/against-zone short 1h**: 2026-07-02 LONDON RN 1m @29850 ord1; 2026-07-02 LONDON SUPPLY 1m @29851.625 ord1; 2026-07-02 LONDON IFVG 1m @29921 ord1; 2026-07-02 LONDON IFVG 1m @29920.125 ord1; 2026-07-02 LONDON IFVG 1m @29920.5 ord1
- **Q5/all/2026/against-zone short 4h**: 2026-07-02 LONDON SUPPLY 1m @29911.875 ord1; 2026-07-02 LONDON DEMAND 1m @29912.75 ord1; 2026-07-02 LONDON FVG 1m @29911.125 ord1; 2026-07-02 LONDON OB 1m @29913.875 ord1; 2026-07-02 LONDON OB 1m @29915.375 ord1
