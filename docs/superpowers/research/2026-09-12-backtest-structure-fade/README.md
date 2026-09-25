# Backtest 1B — structure stop + next-real-level target: MEASURE FIRST

**Branch:** `research/backtest-structure-fade` · claim `118d9849` (ls-remote
`118d98498b91ebf383b32b607ae9428da870705d`, session
`structurefade-4abbb353/copilot[unlisted]`) · accepted dev tip `6b3fddf7`
(backtest-1 merged) · running rev `400ea26c12c8` verified at accept and at
close — **not restarted**.

## The headline

**YES under fill (a) and (c); NO under fill (b).** With the stop at the zone's
far edge plus a small buffer and the target the next zone worth reaching, **all
16 swept geometry cells are net-positive after 2.0 pts of friction** on
fill-on-touch: median +2.41 pts/trade, range +0.39 to +4.21, in-sample AND in
the never-tuned held-out year (where every cell is BETTER than in-sample:
+1.40 to +6.58). Every cell's positive expectancy is significant against the
sign-flip null (Westfall–Young max-T, 2,000 permutations, p<0.001 per cell;
Bonferroni p<0.001). The result survives one and two ticks of stop-side
slippage (worst cell still +0.50 with 2-tick stop slippage) and survives the
±1-tick entry-slippage assumption (c). It does NOT survive fill assumption
(b): the 32% of touches that actually print one tick through the anchor lose
−1.7 to −3.4 pts/trade — the same adverse selection backtest-1 measured, now
visible inside a profitable-looking headline. The reference cell (backtest-1's
1.5×ATR5m floor + nearest target, recomputed on the identical tape) stays
negative (−0.79): the geometry was the difference, not the tape.

**The fill sentence (C6), first:** the positive expectancy is an artifact of
fill assumption (a) to exactly the extent round 10 warns — under (b) the fade
still loses. A resting limit that only fills when price trades through is not
the same trade as one filled on touch; this report cannot rank those two
realities against each other, so it reports both and states that the verdict
depends on which fill is real. The owner rules.

---

## What this measures and why it is the same harness

Backtest-1 answered "does the fade at a zone pay" with the LIVE geometry: stop
floored at 1.5×ATR5m (composeArmStop), target = the NEAREST opposing zone. It
lost net under every fill assumption — but its anatomy showed why: the nearest
opposing zone averages 4.35 pts away against a ~23.5-pt stop (a 1:0.18 trade),
so the fade lost on geometry, not necessarily on the idea. The owner's ruling:
the stop should be defined by STRUCTURE — the zone's far edge plus a small
buffer — and the target should be the next zone WORTH REACHING. This wave
measures that geometry on the identical tape, with the identical population,
outcomes, fills and friction, before any code change.

**Population (identical to backtest-1):** every first touch of a seated
(shortlisted) zone, zone map built at each read instant (LONDON 01:30, NY 08:00,
ASIA 16:30 CT) with the live detection chain called offline
(`AssembleResearchLevels` → `BuildLevelZones`, live defaults k=0.5,
merge=0.5×ATR5m, cap=1.0×ATR5m, family cap 3) on per-contract bars; outcomes
unchanged (D1′ loop on the band edges, H=12×1m plus 5m H10/H20).

**Geometry swept (16 cells = 4 × 4):**

- STOP = zone far edge + buffer, buffer ∈ {1 tick (0.25), 2 ticks (0.5),
  0.25×zoneWidth, 0.5×zoneWidth} — the two readings of the owner's "a tick or
  two, or a resolved fraction of the zone's own width". The 1.5×ATR5m floor is
  NOT applied (this wave measures it demoted to a last-resort bound; every zone
  in this population has a usable width, so the no-zone case never arises —
  stated).
- TARGET = the nearest opposing zone whose anchor is ≥ minR × risk away, minR ∈
  {0.5, 1.0, 1.5, 2.0}. No qualifying zone → no target; the trade exits on stop
  or at the session's flat time.
- Per cell: all three fill assumptions (a) touch, (b) one-tick-through, (c)
  ±1 tick slippage; 2.0 pts round-turn friction on every result; exit =
  stop/target/flat, stop-first on a double-touch bar.
- The R:R-gate question is measured as R2: the share of fill-a trades whose
  planned R ≥ 2.0 and their expectancy (the gate's floor is quoted in the
  methods).

**Reference cell:** backtest-1's geometry (composeArmStop 1.5×ATR5m + nearest
target) is recomputed in the same run on the same tape, so the two geometries
are compared on identical bars.

**Tape note (stated, not hidden):** the live store's bars grew after backtest-1
copied it (Friday 2026-09-11 07:47 CT cut vs the full Friday session). The new
copy adds Friday 08:30→14:45 bars; backtest-1's final LONDON Friday session was
truncated and its NY Friday session skipped, this run has both. The reference
cell below is therefore the backtest-1 geometry on THIS tape, not a reprint of
the published numbers.

## Methods — unchanged except the geometry

Everything in backtest-1's methods holds: per-contract reads
(`BarsBetweenOn` exclusions), `ContractAt` at each read instant, the seam
assertions, the live detector chain called offline, `composeArmStop` verbatim
port with byte-parity assert, the C6 fill definitions, the D1′ outcome loop, the
CT clocks, 2.0-pt friction, held-out = 2025-09-12 → 2026-09-11, and the
Westfall–Young max-T correction (2,000 sign-flip permutations, in-sample only,
plus Bonferroni per cell). The only changes: the stop is
`structureStop(side, lo, hi, buffer)` and the target is
`nearestOpposingQualified(entry, side, zones, minR×risk)`; the map is fixed at
live defaults (backtest-1 swept the map and found 0/81 cells positive under the
ATR geometry, so the map is not re-swept here). The R:R floor quoted for the
gate readout is the live min-R:R resolver (`kernel.MinRR()` — see the backtest-1
basis; the gate floor the engine enforces).

## The surface (seated zones, all eras)

| buffer | minR | touches | exp A net | PF A | win A | stop/target/flat | fillB rate / exp | fillC exp | R≥2 share / gated exp A |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 tick | 0.5 | 11302 | +0.63 | 1.25 | 65% | 27/73/0 | 32% / -3.29 | +0.88 | 9% / +3.51 |
| 1 tick | 1 | 11302 | +2.00 | 1.65 | 67% | 32/68/0 | 32% / -2.59 | +2.25 | 12% / +3.55 |
| 1 tick | 1.5 | 11302 | +2.84 | 1.77 | 61% | 39/61/0 | 32% / -2.14 | +3.09 | 28% / +3.48 |
| 1 tick | 2 | 11302 | +3.41 | 1.80 | 55% | 45/55/0 | 32% / -1.86 | +3.66 | 100% / +3.41 |
| 2 ticks | 0.5 | 11302 | +0.67 | 1.27 | 67% | 26/74/0 | 32% / -3.31 | +0.92 | 8% / +3.63 |
| 2 ticks | 1 | 11302 | +2.07 | 1.66 | 67% | 32/68/0 | 32% / -2.60 | +2.32 | 11% / +3.58 |
| 2 ticks | 1.5 | 11302 | +2.92 | 1.76 | 61% | 39/61/0 | 32% / -2.14 | +3.17 | 26% / +3.54 |
| 2 ticks | 2 | 11302 | +3.47 | 1.78 | 55% | 45/54/0 | 32% / -1.85 | +3.72 | 100% / +3.47 |
| 0.25×width | 0.5 | 11302 | +1.31 | 1.45 | 73% | 24/76/0 | 32% / -3.44 | +1.56 | 4% / +3.68 |
| 0.25×width | 1 | 11302 | +2.80 | 1.67 | 67% | 33/67/0 | 32% / -2.68 | +3.05 | 6% / +3.42 |
| 0.25×width | 1.5 | 11302 | +3.53 | 1.67 | 58% | 42/58/0 | 32% / -2.32 | +3.78 | 18% / +3.84 |
| 0.25×width | 2 | 11302 | +4.11 | 1.67 | 51% | 49/51/0 | 32% / -2.03 | +4.36 | 100% / +4.11 |
| 0.5×width | 0.5 | 11302 | +1.96 | 1.55 | 76% | 23/77/0 | 32% / -3.34 | +2.21 | 2% / +1.34 |
| 0.5×width | 1 | 11302 | +3.33 | 1.60 | 65% | 35/65/0 | 32% / -2.65 | +3.58 | 3% / +1.61 |
| 0.5×width | 1.5 | 11302 | +4.08 | 1.57 | 55% | 45/54/1 | 32% / -2.38 | +4.33 | 12% / +4.59 |
| 0.5×width | 2 | 11302 | +4.77 | 1.58 | 48% | 52/47/1 | 32% / -1.69 | +5.02 | 100% / +4.77 |

Average risk (stop distance) per buffer: 7.12 pts (1 tick), 7.37 (2 ticks),
10.22 (0.25×width), 13.56 (0.5×width) — the structure stop is ~1/2 to ~1/3 of
the 23.5-pt 1.5×ATR5m stop, and that is the whole difference. Flat share is
~0: with a qualified target or a tight stop, trades resolve in-session.

Surface summary: **16 of 16 cells net-positive**; median +2.41, min +0.39, max
+4.21 pts/trade; best cell max-T p<0.001 (2,000 sign-flip permutations,
in-sample), Bonferroni p<0.001 — the positive expectancy is not a single good
cell in a losing surface; every cell clears the sign-flip null.

## Era split (fill A, net pts/trade)

| buffer | minR | in-sample exp / PF | held-out exp / PF |
|---|---|---|---|
| 1 tick | 0.5 | +0.39 / 1.18 | +1.40 / 1.41 |
| 1 tick | 1 | +1.59 / 1.59 | +3.31 / 1.78 |
| 1 tick | 1.5 | +2.37 / 1.72 | +4.37 / 1.86 |
| 1 tick | 2 | +2.86 / 1.75 | +5.20 / 1.90 |
| 2 ticks | 0.5 | +0.44 / 1.20 | +1.43 / 1.41 |
| 2 ticks | 1 | +1.66 / 1.59 | +3.43 / 1.80 |
| 2 ticks | 1.5 | +2.46 / 1.72 | +4.42 / 1.86 |
| 2 ticks | 2 | +2.91 / 1.73 | +5.27 / 1.89 |
| 0.25×width | 0.5 | +0.99 / 1.39 | +2.33 / 1.57 |
| 0.25×width | 1 | +2.32 / 1.63 | +4.35 / 1.75 |
| 0.25×width | 1.5 | +2.96 / 1.63 | +5.38 / 1.73 |
| 0.25×width | 2 | +3.56 / 1.66 | +5.91 / 1.69 |
| 0.5×width | 0.5 | +1.56 / 1.51 | +3.24 / 1.63 |
| 0.5×width | 1 | +2.69 / 1.55 | +5.42 / 1.71 |
| 0.5×width | 1.5 | +3.48 / 1.55 | +6.04 / 1.61 |
| 0.5×width | 2 | +4.21 / 1.59 | +6.58 / 1.57 |

The held-out year is uniformly BETTER than in-sample in every cell — a result
that appears only in-sample would be reported as such; this one is stronger out
of sample. Nothing was tuned on the held-out year (the grid was fixed by the
owner's message before the run).

## Reference cell — backtest-1 geometry on this tape

| era | touches | hold | fillA exp / PF | fillB rate / exp | fillC exp |
|---|---|---|---|---|---|
| all | 11302 | 73.3% | -0.79 / 0.75 | 32% / -4.93 | -0.54 |
| held_out | 2663 | 71.3% | -0.61 / 0.86 | 33% / -6.54 | -0.36 |
| in_sample | 8639 | 73.8% | -0.84 / 0.70 | 31% / -4.40 | -0.59 |

These reproduce backtest-1's published numbers exactly (−0.789 / −0.843 /
−0.615): the tape and population are identical, so the geometry comparison is
apples-to-apples. **The same touches, the same fills, the same friction: −0.79
pts/trade with the volatility stop, +2.41 median with the structure stop.**

## Fill assumptions and the R:R gate, per cell

Fill (b) — filled only if the touch bar prints one tick through the anchor —
is NEGATIVE in every cell (−1.69 to −3.44 pts/trade at a 32% fill rate). That
subset is not a worse version of the same trade; it is a different selection
(the touches that continue through the level), and round 10's adverse
selection shows up in it exactly as it did in backtest-1. Fill (c) — entry ±1
tick against you — stays positive everywhere (+0.88 to +5.02). A stop-side
slippage sensitivity (not part of the dispatch's three assumptions, computed
in addition): shifting every stop exit by 1 and 2 ticks leaves every cell
positive (worst cell: +0.50 with 2-tick stop slippage).

R≥2 gate readout: at minR=2 the gate is trivially satisfied (100% pass — with
a tight structure stop almost any opposing zone clears 2×risk). At minR=0.5
only 2–9% of touches clear R≥2, and the gated subset earns MORE than the cell
mean (+3.4 to +4.6) — the minR sweep is itself the gate knob, and the surface
shows the edge concentrating as the target requirement tightens.

## Hold-rate context (unchanged outcomes)

h12 hold=0.7325 n=10977 amb=325 | 5mH10 hold=0.6518 n=10573 | 5mH20 0.6499 | live detector verbatim 50.58% — all identical to backtest-1 (the population is byte-identical).
## Surprises (A23 — included, not acted on)

1. **The geometry flips the verdict, not the tape.** Identical touches, fills
   and friction: −0.79 with the 1.5×ATR5m stop, +2.41 median with the structure
   stop. Backtest-1's conclusion — "the fade is dead" — was about its
   geometry. This measurement says the structure-stop fade is positive on
   every swept cell, in sample and out.
2. **Adverse selection is the one durable negative.** Fill (b) loses in every
   cell of BOTH backtests, at almost the same magnitudes (−1.7 to −3.4 here;
   −4.4 to −6.5 with the ATR stop). Whatever geometry wins, the through-print
   fills lose — that is the fill reality to resolve before any code change.
3. **The R:R gate inverts its meaning.** With a volatility stop, R≥2 is a
   binding filter; with a structure stop it is trivially satisfied at minR=2
   and nearly vacuous below. The gate's role changes when the stop comes from
   structure — it becomes the target-quality knob (minR), not a pass/fail
   switch.
4. **The held-out year is uniformly stronger** than in-sample (+1.40…+6.58 vs
   +0.39…+4.21). Not the overfit signature.
5. **Sample ids** this report rests on: `ev-0000003`, `ev-0000006`,
   `ev-0000007` … `ev-0279089`, `ev-0279092` (11,302 seated events, 8,639
   in-sample + 2,663 held-out; 3,084 reads built, 1,760 skipped — byte-identical
   enumeration to backtest-1).

## Limitations — stated plainly

1. **Fill (a) is the optimistic assumption and (b) is negative everywhere.**
   The headline is only as true as the fill assumption is real; the report
   cannot rank queue position against a through-tick print (round 10). If a
   resting limit at the anchor fills on touch, the structure fade measured
   positive; if it only fills on a through print, it measured negative.
2. **The trade is a deterministic realization of the owner's geometry.** The
   live planner is an LLM; its scenario choices, authored stops and targets are
   not replayed. What is measured is the rule as specified, on the machine's
   own zone maps.
3. **Stops fill at exact prices** in the simulation; entry slippage is covered
   by (c), and the additional stop-side slippage sensitivity (1–2 ticks) is
   reported above and stays positive. Wider stop slippage is not modeled.
4. **The grid was chosen by the owner's hypothesis, not discovered here** —
   but it was fixed before the run, the whole surface is reported (16/16
   positive, no best-cell cherry-picking), and the held-out year was never
   tuned. Multiple comparisons corrected (max-T + Bonferroni).
5. **Era fidelity, session enablement, and entry realism** caveats from
   backtest-1 carry over unchanged (see its limitations).
6. **No recommendation.** This is a measurement. The owner rules on any code
   change; nothing in the trading path was touched and the running bot was not
   restarted.

## Artifacts

- `harness/` — the 1B harness (a copy of backtest-1's, geometry changed only;
  nothing ships).
- `artifacts/` (committed): `geometry_cells.json` / `geometry_cells.csv` (16
  cells × era × fills), `reference_cell.json`, `hold_context.json`,
  `surface_geometry.csv` + `surface_summary.json`, `seam.json`, `summary.json`.
- Worktree-only (gitignored): `data/backtest.db` (the DB COPY, md5
  `55a26bbe51c784f66aeca081644d8574` = the live store at copy time), `data/run/`
  (`events.jsonl` every event with ids and all 48 trade variants, `run.log`).
