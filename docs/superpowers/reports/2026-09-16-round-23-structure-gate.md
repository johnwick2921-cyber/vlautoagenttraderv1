# Round 23 — Structure Gate (S4 Measurement)

**Lane:** DS-R001 (DeepSeek research). **Worktree:** `/home/hoang/nofx-r001`.
**Branch:** `docs/round-23-structure-gate` (claim `a74da3c3`, cut from
`origin/docs/round-23-top-down-map` @ `d15db077` per CTO order; origin/dev base `1e3ad705`).
**Dispatch:** CTO 2026-09-16 22:58Z, section 5 (S4 MEASUREMENT GATE). This report decides
whether ANY S1/S2/S3 knob defaults ON.

**ACCEPTED as S4 gate by CTO 2026-09-16 21:35 CT @8fde4429.**

Method, population, and instrument: `docs/superpowers/research/2026-09-16-round-23/README.md`
and the DS-R→DS-R001 handover (`docs/superpowers/reports/2026-09-16-round-23-HANDOVER.md`).
Data: 4,471,482 D1′ episodes / 2,680,931 level-scans / 3,093 replayed reads, full CME era,
DB copy read-only. p_null = 0.5067. All rates below are ordinal-1 first-touch holds,
Wilson CI, exact two-proportion p. Row-level ids for every cell live in
`research/2026-09-16-round-23/out-s4/{episodes,qa,trends}.jsonl` (day, session, kind, tf, price).

## Q-A — reaction rate at HTF zones under 1m-freshness vs TF-freshness

1m-touch grading (today's live A/B/C/done ladder — **VALID**), n = 24,114 HTF scans:

| grade | hold | n |
|---|---|---|
| fresh | 0.495 [0.479,0.510] | 4009 |
| tested-1 | 0.474 [0.424,0.524] | 380 |
| tested-2 | 0.446 [0.378,0.516] | 195 — NOT MEASURED |
| stale | 0.504 [0.497,0.511] | 19530 |

fresh-vs-stale: z = −1.04, p = 0.297. **The decay ladder does not separate reaction rates.**
A level's 1m-touch count predicts nothing about whether the next touch holds.

Own-TF grading (S2 semantics): **INVALID — S2 F1** (CTO 00:06Z ruling: the grader counts a
zone's own formation bars as tests, so the grades measure the birth artifact, not decay).
The column is retained in `out-s4/qa.jsonl` (`tests_tf`, `grade_tf`) but must not be read as
evidence. Re-run when DS-101 lands re-entry semantics.

## Q-B — outcome split when episode direction agrees vs opposes the S1 structure state

State defined EXACTLY as S1 (`kernel/structure.go` + `structure_map.go`,
`feat/structure-map` @ `f36ddca2`, now on dev): closed D/4h/1h bars → fractalSwings (k=2,
min-move 0.25×ATR14 Wilder) → last 3 labelled swings all HH/HL = up, all LH/LL = down, else
range. Port parity-pinned against Claude-101's own zigzag tests (up/down/range all green).

vs 4h trend: agree **0.531 [0.521,0.542] n=8732** · oppose **0.476 [0.466,0.487] n=8590** ·
range 0.506 [0.505,0.507] n=643092. agree-vs-oppose **z = +7.21, p < 10⁻¹²**.

vs D trend: agree **0.551 [0.542,0.560] n=11868** · oppose **0.435 [0.427,0.444] n=13291** ·
range 0.507 [0.505,0.508] n=635255. agree-vs-oppose **z = +18.40, p < 10⁻⁷⁵**.

The S1 structure state is the largest, most robust separator the entire round series
(1–23) has produced: ~11.6pt spread on D, ~5.5pt on 4h. Levels touched WITH the daily
trend hold 55%; levels touched against it hold 43.5%. "Range" (no clean trend) is pure
null — which is also why the raw Q1 cells all sit at ~0.51: most sessions carry no clean
structure trend, and those episodes dilute everything toward the null.

## Q-C — does HTFScoreMultiplier ×1.2 separate anything

Site fact [A]: `kernel/levels_score.go:512-517` — `htf = HTFScoreMultiplier` (1.2) applies
ONLY to non-zone kinds; zones (Supply/Demand/FVG/IFVG/OB) use `zoneTFMult` and never see
the ×1.2. It is a uniform constant on the promoted group, so it cannot reorder within it;
the empirical question is whether the promoted group differs from what it displaces.

| group | hold | n |
|---|---|---|
| HTF non-zone (×1.2 applies) | 0.498 [0.493,0.503] | 40049 |
| intraday non-zone (×1.0) | 0.508 [0.506,0.510] | 234646 |
| HTF zone (×1.2 NOT applied) | 0.508 [0.503,0.513] | 39724 |
| intraday zone (×1.0) | 0.506 [0.504,0.507] | 345995 |

HTF-nonzone vs intraday-nonzone: **z = −3.54, p = 0.0004** — the multiplier promotes a group
that holds slightly LESS than the intraday levels it displaces (≈1pt, significant only by n).
The multiplier separates in the wrong direction; nothing supports ×1.2 > 1.0.

## Q-D — is a 5m zone distinguishable from a 1m zone (today both are the "1m" tier)

Correction on the record: the DS-R handover cell (5m swing-family 0.499 n=9547 vs 1m
swing-family 0.512 n=128801, z=−2.45) was **confounded** — the 1m swing family is dominated
by EQH/EQL (not zones). Kind-pure 5m-vs-1m does not exist: the detector emits NO swing
kinds at 1m; 5m carries only SWG-H/SWG-L.

- 5m zones (SWG only): 0.499 [0.489,0.509] n=9547 → CI contains p_null → **null**.
- 5m vs 15m zones (the only kind-pure comparison): z = −0.88, p = 0.376 → not distinguishable.
- Distance buckets (5m): 0-25 0.502 n=5421 · 25-50 0.500 n=2343 · 50-100 0.488 n=1296 ·
  100-200 0.476 n=399 — nothing separates from null.

The `zoneTierFor` merge of 5m into the "1m" tier is empirically moot: 5m zones react like
noise, exactly like 1m zones.

## Lookahead check (CTO 02:23Z) — CLEAN

The S1 trend for a (day, session) is computed from bars with `OpenTime+iv <= read_time`
(the exact `ComputeStructureMap` closed-bar filter), and the session registry reads 30
minutes BEFORE the window opens (LONDON 01:30/02:00+, NY 08:00/08:30+, ASIA 16:30/17:00+).
Verified on the whole set: **4,471,482/4,471,482 episodes have `opened_at_ms > read_at_ms`;
0 violations; the minimum gap is exactly 1,800,000 ms (30 min)**. Therefore every bar in
the trend computation closed at least 30 minutes before any episode in that session could
open — no bar used in Q-B closed after any entry. Worked id: episode
{day=2026-06-02, session=LONDON, kind=AS-H, tf=1m, opened_at_ms=1780478280000} — its read
is 1780468200000 (gap 168 min), D trend "up" from 1831 closed daily bars; the LAST daily
bar the port could use opens 1780376400000 and closes at the 1780462800000 bound, strictly
before the episode entry (`close < entry` = 1 in the copy's bars table).

## Q-B extras (CTO 02:23Z cross cells)

D/4h cross: agree-both **0.540 [0.521,0.559] n=2606** · mixed **0.653 [0.626,0.680]
n=1174** · oppose-both **0.408 [0.390,0.427] n=2655** · any-range 0.506 n=653979.
agree-both vs oppose-both z = +9.56. (The mixed cell — with-trend at one TF, against at
the other — holds most; n is modest, reported, not interpreted.)

vs 1h trend: agree 0.506 [0.503,0.510] n=70611 · oppose 0.513 [0.510,0.516] n=111259 ·
range 0.504 n=478544 — **the 1h state separates nothing.**

Per-session (vs D): LONDON agree 0.546/oppose 0.449 · **NY agree 0.570/oppose 0.405** ·
ASIA agree 0.530/oppose 0.468. The effect is largest in NY.

Per-side (vs D): long agree 0.540/oppose 0.476 · short agree 0.555/**oppose 0.335** —
the oppose penalty is concentrated on the short side (n=3843).

## Q-A per-family (CTO 02:23Z check 3)

HTF zones only: fresh n=1 (NOT MEASURED — 1m touches make every HTF zone stale almost
immediately; this is the WHY-block pathology measured), stale 0.503 [0.494,0.512] n=11809.
HTF references only: fresh 0.495 [0.479,0.510] n=4008 vs stale 0.504 [0.493,0.515] n=7721,
z = −0.97, p = 0.332. **The ladder separates nothing in either family.**

## Sample ids (first 5 per cell; every row lives in out-s4/)

All rows for every cell are in `/home/hoang/nofx-r101/docs/superpowers/research/2026-09-16-round-23/out-s4/{episodes,qa,trends}.jsonl` — id = day session kind tf@opened_at_ms.

- Q-A 1m-grade fresh (n=4009): 2022-04-18 LONDON EQL 1h@1650358140000, 2022-04-18 NY EQH 1h@1650377280000, 2022-04-19 LONDON EQH 1h@1650455460000, 2022-04-19 NY EQH 1h@1650471180000, 2022-04-20 NY EQH 1h@1650556200000
- Q-A 1m-grade tested-1 (n=380): 2022-04-11 NY EQL 1h@1649790540000, 2022-04-17 NY SUPPLY 1h@1650288660000, 2022-04-17 NY OB 1h@1650288660000, 2022-04-17 NY OB 1h@1650288660000, 2022-04-20 LONDON EQH 1h@1650527820000
- Q-A 1m-grade tested-2 (n=195): 2022-04-18 ASIA EQH 1h@1650418080000, 2022-04-28 LONDON EQH 1h@1651215900000, 2022-05-01 NY EQL 1h@1651501620000, 2022-05-09 NY EQL 1h@1652189460000, 2022-05-14 ASIA EQH 1h@1652652240000
- Q-A 1m-grade stale (n=19530): 2022-04-11 LONDON SUPPLY 1h@1649766660000, 2022-04-11 LONDON OB 1h@1649766660000, 2022-04-11 NY SUPPLY 1h@1649770440000, 2022-04-11 NY SUPPLY 1h@1649776200000, 2022-04-11 NY OB 1h@1649771280000
- Q-A HTF zones fresh (n=1): 2025-04-06 NY OB 1h@1744035120000
- Q-A HTF zones stale (n=11809): 2022-04-11 LONDON OB 1h@1649766660000, 2022-04-11 NY OB 1h@1649771280000, 2022-04-11 NY OB 1h@1649780760000, 2022-04-12 LONDON IFVG 1h@1649837160000, 2022-04-12 LONDON OB 1h@1649833680000
- Q-A HTF refs fresh (n=4008): 2022-04-18 LONDON EQL 1h@1650358140000, 2022-04-18 NY EQH 1h@1650377280000, 2022-04-19 LONDON EQH 1h@1650455460000, 2022-04-19 NY EQH 1h@1650471180000, 2022-04-20 NY EQH 1h@1650556200000
- Q-A HTF refs stale (n=7721): 2022-04-11 LONDON SUPPLY 1h@1649766660000, 2022-04-11 NY SUPPLY 1h@1649770440000, 2022-04-11 NY SUPPLY 1h@1649776200000, 2022-04-12 LONDON DEMAND 1h@1649843160000, 2022-04-12 NY DEMAND 1h@1649857380000
- Q-B vs 4h agree (n=8732): 2026-06-02 LONDON RN 1m@1780470360000, 2026-06-02 LONDON RN 1m@1780470720000, 2026-06-02 LONDON RN 1m@1780481340000, 2026-06-02 LONDON GAP 1m@1780470720000, 2026-06-02 LONDON GAP 1m@1780470540000
- Q-B vs 4h oppose (n=8590): 2026-06-02 LONDON EQH 1m@1780472580000, 2026-06-02 LONDON EQH 1m@1780472580000, 2026-06-02 LONDON EQH 1m@1780470120000, 2026-06-02 LONDON SUPPLY 1m@1780478700000, 2026-06-02 LONDON DEMAND 1m@1780472520000
- Q-B vs 4h range (n=643092): 2022-04-10 LONDON AS-H 1m@1649662140000, 2022-04-10 LONDON AS-L 1m@1649661300000, 2022-04-10 LONDON ONH 1m@1649662140000, 2022-04-10 LONDON ONL 1m@1649661300000, 2022-04-10 LONDON RN 1m@1649665200000
- Q-B vs D agree (n=11868): 2026-06-02 LONDON RN 1m@1780470360000, 2026-06-02 LONDON RN 1m@1780470720000, 2026-06-02 LONDON RN 1m@1780481340000, 2026-06-02 LONDON GAP 1m@1780470720000, 2026-06-02 LONDON GAP 1m@1780470540000
- Q-B vs D oppose (n=13291): 2026-06-02 LONDON EQH 1m@1780472580000, 2026-06-02 LONDON EQH 1m@1780472580000, 2026-06-02 LONDON EQH 1m@1780470120000, 2026-06-02 LONDON SUPPLY 1m@1780478700000, 2026-06-02 LONDON DEMAND 1m@1780472520000
- Q-B vs D range (n=635255): 2022-04-10 LONDON AS-H 1m@1649662140000, 2022-04-10 LONDON AS-L 1m@1649661300000, 2022-04-10 LONDON ONH 1m@1649662140000, 2022-04-10 LONDON ONL 1m@1649661300000, 2022-04-10 LONDON RN 1m@1649665200000
- Q-B cross agree-both (n=2606): 2026-06-02 LONDON RN 1m@1780470360000, 2026-06-02 LONDON RN 1m@1780470720000, 2026-06-02 LONDON RN 1m@1780481340000, 2026-06-02 LONDON GAP 1m@1780470720000, 2026-06-02 LONDON GAP 1m@1780470540000
- Q-B cross oppose-both (n=2655): 2026-06-02 LONDON EQH 1m@1780472580000, 2026-06-02 LONDON EQH 1m@1780472580000, 2026-06-02 LONDON EQH 1m@1780470120000, 2026-06-02 LONDON SUPPLY 1m@1780478700000, 2026-06-02 LONDON DEMAND 1m@1780472520000
- Q-B cross mixed (n=1174): 2026-07-02 LONDON RN 1m@1783062060000, 2026-07-02 LONDON RN 1m@1783071900000, 2026-07-02 LONDON EQL 1m@1783071900000, 2026-07-02 LONDON SUPPLY 1m@1783080060000, 2026-07-02 LONDON SUPPLY 1m@1783062120000
- Q-C HTF non-zone (n=40049): 2022-04-11 LONDON SUPPLY 1h@1649766660000, 2022-04-11 LONDON SUPPLY 15m@1649766660000, 2022-04-11 NY PDL 1m@1649790480000, 2022-04-11 NY PDC 1m@1649790300000, 2022-04-11 NY EQL 1h@1649790540000
- Q-C intraday non-zone (n=234646): 2022-04-10 LONDON AS-H 1m@1649662140000, 2022-04-10 LONDON AS-L 1m@1649661300000, 2022-04-10 LONDON ONH 1m@1649662140000, 2022-04-10 LONDON ONL 1m@1649661300000, 2022-04-10 LONDON RN 1m@1649665200000
- Q-C HTF zone (n=39724): 2022-04-11 LONDON OB 1h@1649766660000, 2022-04-11 LONDON FVG 15m@1649767680000, 2022-04-11 LONDON IFVG 15m@1649766660000, 2022-04-11 LONDON FVG 15m@1649747100000, 2022-04-11 LONDON OB 15m@1649767980000
- Q-C intraday zone (n=345995): 2022-04-10 LONDON IFVG 1m@1649661120000, 2022-04-10 LONDON IFVG 1m@1649661120000, 2022-04-10 LONDON IFVG 1m@1649661120000, 2022-04-10 LONDON IFVG 1m@1649661120000, 2022-04-10 LONDON FVG 1m@1649661180000
- Q-D 5m zones (n=9547): 2022-04-10 LONDON SWG-L 5m@1649661120000, 2022-04-10 LONDON SWG-H 5m@1649660400000, 2022-04-10 LONDON SWG-L 5m@1649661300000, 2022-04-10 NY SWG-L 5m@1649683980000, 2022-04-10 NY SWG-H 5m@1649684940000
- Q-D 15m zones (n=61130): 2022-04-10 NY SWG-L 15m@1649684820000, 2022-04-10 NY SWG-L 15m@1649683800000, 2022-04-10 NY SWG-L 15m@1649684940000, 2022-04-11 LONDON SWG-H 15m@1649747940000, 2022-04-11 LONDON SWG-H 15m@1649752620000
- Q-B 1h agree (n=70611): 2022-04-11 LONDON RN 1m@1649749080000, 2022-04-11 LONDON RN 1m@1649747160000, 2022-04-11 LONDON EQH 1m@1649748180000, 2022-04-11 LONDON EQH 1m@1649748180000, 2022-04-11 LONDON EQH 1m@1649748120000

## What the cells support (and only this)

1. **Q-B is the one real effect.** The structure state (D trend especially) separates
   episodes by up to 11.6pt. This is evidence IN FAVOR of the S1 structure layer existing
   and being shown to the model — the round's first effect large enough to survive a glance.
2. **The freshness decay ladder separates nothing** (Q-A 1m-touch). Whether it should stop
   decaying zones is a separate question from the cells — the cells only say the ladder
   does not predict reaction.
3. **×1.2 separates in the wrong direction** (Q-C, −1pt).
4. **5m as a distinct tier is not supported** (Q-D), and equally not hurt — the merge is moot.
5. Q-A own-TF column: **INVALID until re-run** after DS-101's re-entry fix.

No recommendation beyond these cells is made. Numbers, not opinions, decide the knobs.
