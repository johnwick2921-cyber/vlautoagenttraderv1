names rewritten to vl on 2026-09-30 (VL rename)
even# SUBSYSTEM D — LEVEL GRADER · research-conformance re-check

**Snapshot** 2026-09-04 08:46:27 CT · deployed rev `70af663d` (PID 878451, booted 08:30:11 CT) · worktree `/home/hoang/vl-conform` @ `fb50903f` (base dev `492d2067`) · **READ-ONLY**, DB opened `mode=ro` only.

Report pins (`git log -1 -- <path>`):

| report | pinning commit |
|---|---|
| `docs/superpowers/reports/2026-09-02-belief-census.md` | `ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 …` |
| `docs/superpowers/reports/2026-09-02-level-kind-replay.md` | `3961f8733afa409ec4ef3edfdcff1437fdeac235 2026-09-02 19:03:10 -0500 docs(level-replay part2): 1h variant results …` |
| `docs/superpowers/reports/2026-08-24-level-grading-full-audit.md` | `99917b9a09232aae3bf435e56f4b652489317c0c 2026-08-24 22:23:41 -0500 fix(levels): full grading audit …` |
| `docs/superpowers/reports/2026-08-28-grand-audit.md` | `741bfc2a8c443feceaa0f31d30c015946b775633 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports …` |
| `docs/superpowers/reports/2026-09-02-detector-redesign.md` | `0465a10bfa4b865a8406a1d684501ec4673febc7 2026-09-02 07:58:10 -0500 docs(lane4): A8 SATISFIED …` |
| `docs/superpowers/reports/2026-08-30-knob-census.md` | `741bfc2a8c443feceaa0f31d30c015946b775633` (same archive commit) |
| `docs/superpowers/research/INDEX.md` | `4e8e7e1ae069bc0285f677a316b4771437a39a06 2026-09-03 19:37:14 -0500 docs(index): the stranded-branch sweep …` |

---

## 0. THE FOUR HEADLINES

**H1 — D5 answered: NOTHING MOVED. [A]**
`kernel/levels_score.go` is **byte-identical** at the census commit, at the deployed rev, and in the worktree:

```
git show ee64a494:kernel/levels_score.go | md5sum → dfcda708e8cad3e2b03af3affe4df5d1
git show 70af663d:kernel/levels_score.go | md5sum → dfcda708e8cad3e2b03af3affe4df5d1
md5sum kernel/levels_score.go               → dfcda708e8cad3e2b03af3affe4df5d1
git diff --stat 70af663d -- kernel/levels_score.go kernel/levels_swing.go kernel/levels_role.go … → (empty)
```

Last commit touching the file: `e86ae805 2026-08-31 08:11:57 -0500 remove min_side_levels entirely (owner ruling 2026-08-31)` — **before** the census (09-02 08:50) and before the level-kind replay (09-02 19:03). **Zero ladder terms changed.** That is exactly what `2026-09-02-level-kind-replay.md:366` ordered: *"Grader ruling stands: nothing changes; every ladder term keeps its [I] label."* **CONFORMS.**

**H2 — the boot line misreports TWO grader values (A11 violation). [A]**
`kernel/levels_volume_boot.go:14` is a single `logger.Infof` whose `seats=%d` argument is the **file constant** `DefaultMaxLevels` (`kernel/levels_score.go:54`, = 8) and whose `retuned 0.3` is a **string literal inside the format string** — neither is read from the resolver.

Resolved reality for the one running trader (`hoang`, id `8d5c8af5_…_1781246265`, `is_running=1`, strategy `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` last updated **2026-09-01 13:13:06 UTC**, i.e. before the 09-04 boot, so this is what the process loaded):

```
day_plan.max_levels            = 12     (boot line says 8)
day_plan.proximity_filter_atr  = 1      (boot line says "retuned 0.3")
```

Live corroboration, independent of the DB:

* `data/vl_2026-09-04.log` — `kernel/levels_score.go:575  🗺️ seated 12/526 in-band levels (proximity band ±510pt, 12 of them retained)` — 81 such lines today at cap **12**, 76 at cap **24** (the `eff*2` pre-seat pool), 5 at cap 4 (the planner's HTF-zone sub-table, `trader/auto_trader_planner.go:2154` passes literal 4).
* `candidate_pool` (8 reads, 2026-09-04 00:30:54 → 08:32:00 CT): **12 seated of 24** every read except 01:30:48 (11).
* Band arithmetic: `band = proximityK × DailyRangeProxy` (`kernel/levels_score.go:417`, `kernel/levels_assemble.go:291`). The proxy is the mean completed session-day H−L in the bar window; with a ~2000-bar 1m window that is exactly the prior session day. From `bars` (MNQ 1m): session-day **2026-09-03 range = 510.00 pt** → today's band **±510pt** ⇒ **k = 1.0**. Cross-check: session-day 2026-09-02 range = 285.25 → 09-03's band `±285pt` ⇒ k = 1.0 again. At k=0.3 today's band would be ±153pt.

So the census D7 value **±0.3×dATR** and `2026-08-29-weekend-audit-p1.md:79` (*"Proximity @0.3 PROVEN live: band flipped ±458→±85pt at exactly 11:59:00 CT"*) are **stale**: the live lock is **3.3× wider** than the researched/owner-retuned value.

**H3 — D9's `[X]` is a sign error in the census, not a contradiction on the tape. [A]**
Census `belief-census.md:77` labels *"Swing seats improve turn capture"* **[X] research-CONTRADICTED**, citing `grand-audit.md:74`. That line reads:

> `scripts/leveltruth_missed_turns.py` on live (repaired) bars: baseline 80.0/75.0/79.2% → with swing seats 65.0/60.0/66.7% (Δ −15.0/−15.0/−12.5 pts). Reproduces the T3 result independently. **PROVEN.**

**Missed-turns is a MISS rate** (`kernel/levels_swing.go:12-14`: "43% of 5m fractal swing turns … with NO seated level within ±4 points"). A −15pt change in a miss rate is an **improvement**, so the cited evidence **supports** the belief. Two other reports say the same in the same direction: `2026-08-27-level-truth-wave.md:24` ("75.0/80.0/82.6% → 60.0/65.0/65.2% with swing seats (8-seat cap)") and `2026-08-27-deep-verify-22.md:38` ("**CLOSED** … 50→41"). The correct label is **[T]** (measured own tape, supportive, n = 3 sessions / 50→41 missed turns), not [X].

Consequence: the census's own "[X] with live teeth" flag (`belief-census.md:107`) and demotion-queue rank 2 (`:130`, "likely unseat or reduce") rest on the sign error. There is no *"[X]-with-teeth"* here to enforce.

Swing seats **are** still seated at boot 8 [A]: `typeEvidence` returns **0.85** for `KindSWGH/KindSWGL` (`kernel/levels_score.go:104-105`), and — see F1 below — SWG-H/SWG-L are members of `isVolumeFamilyKind` (`:783`), which gives them a **reserved seat** (`SeatVolumeFamily`, `:794`) *and* immunity from `seatBothSides` eviction (`:820`). Live proof from `candidate_pool` on 2026-09-04: SWG-H **6 of 6** candidates seated, SWG-L **5 of 7** — 11 seated across the four ASIA/LONDON reads (00:30, 01:10, 01:30, 02:00 CT); the four 07:45–08:32 CT reads produced **no** SWG candidates at all.

**H4 — D1′ is DESCRIPTIVE ONLY. Proven by exhaustive reader census (A29). [A]**
No gate, weight, filter, prompt line or grade reads `touch_outcomes` or `candidate_pool`. Full production-reference census below (§4).

---

## 1. THE RULE TABLE — every ladder/weight with its value NOW

All `file:line` are `/home/hoang/vl-conform/…` and identical to deployed `70af663d`.
"Prod callers" = the 9 production call sites that reach `scoreLevelsPool`: `kernel/engine_analysis.go:410`, `:468`; `trader/auto_trader_planner.go:714`, `:2132`, `:2154`, `:2167`; `trader/auto_trader_levelstate.go:57`, `:200`; `trader/auto_trader_watcher.go:324`.

| # | rule | file:line | resolved value NOW | label | grounding (report:line) | live effect | conforms? |
|---|---|---|---|---|---|---|---|
| D1 | grading ground / formula | `kernel/levels_score.go:132-137`; formula `:479-484` | `zone = zoneEvidence(kind,tier) × zoneSizeMult × freshness × (1+0.20·cappedConf) × zoneTFMult[tier]`; `line = typeEvidence(kind) × freshness × (1+0.20·cappedConf) × htf` | [O] (owner-approved v3 2026-08-24); the census's [R] half has no locatable citation | `2026-08-24-level-grading-full-audit.md:47-65` ("VERIFIED CORRECT"; "externally supported" with no source named) | weight | **yes** — formula unchanged |
| D2a | `zoneEvidenceByKind` | `:148-154` | OB 0.40/0.50/0.70/0.72 · FVG,IFVG,Supply,Demand 0.35/0.45/0.65/0.65 (1m/15m/1h/4h) | [I] | `belief-census.md:70`; `level-kind-replay.md:366` "keeps its [I] label" | weight | **yes** |
| D2b | `zoneTFMult` | `:157` | `{1m:1.0, 15m:1.1, 1h:1.2, 4h:1.3}` (effective 4h:1m ≈2.3× per the R3 note `:139-145`) | [I] | `belief-census.md:70`; verified drift-free at `grand-audit.md:80` | weight | **yes** |
| D3 | `zoneReversalBonus` | `:160` | **1.1** | [I] per census; `[R]`-weak per grading audit | `belief-census.md:71`; `2026-08-24-level-grading-full-audit.md:57` ("externally supported ✓" — no source cited) | weight | **yes** |
| D4 | `zoneSizeMult` ladder | `:205-227` | ≤0.30→1.25 · ≤0.60→1.10 · ≤1.00→1.00 · ≤1.50→0.85 · ≤2.50→0.70 · else 0.50 | [I] | `belief-census.md:72` | weight | **yes** |
| D5a | zone freshness ladder `zoneFreshMult` | `:378-390` | **1.0 / 0.6 / 0.3 / 0.15** (fresh/B/C/consumed) | [I] | `belief-census.md:73` | weight | **yes** — boot line `zone-ladder=1.0/0.6/0.3/0.15` matches |
| D5b | anchor freshness ladder `freshMult` | `:359-372` | **1.0 / 0.8 / 0.6 / 0.5** | [I]; the 0.8/0.6 rungs [R]-weak | `belief-census.md:73`; `2026-08-24-level-grading-full-audit.md:63-64` ("near the one published quantified sample 0.78/0.55") | weight | **yes** |
| D6 | kind weights `typeEvidence` | `:87-127` | PD*/RTH*/PW*/PM* **1.00** · ONH/ONL/nPOC **0.85** · **VWAP/POC 0.90** · VWAP±2σ 0.85 · **SWG-H/L 0.85** · eVWAP/pdVWAP 0.85 · VAH/VAL/SETT 0.80 · AS/LDN/OR/IB/EQ 0.70 · MID-O 0.60 · Round/Gap 0.55 · zone kinds 0.30 · default 0.50 | [I] | `belief-census.md:74` | weight | **yes on code; census row is WRONG** — census says "VWAP .85", code is **0.90** |
| D6b | HTF line multiplier | `:474-477` | **1.2** | [I] | `grand-audit.md:80` ("`:477` htf=1.2 — matches README-VL-SYSTEM.md:91-92 exactly") | weight | **yes** |
| D7 | proximity band (day-trade lock) | `:417` (`band = proximityK*dATR`); resolver `kernel/plan_lifecycle.go:25-30` (clamp 0.1–3.0) | **1.0 × proxy = ±510 pt today** | [O] | `belief-census.md:75` (says ±0.3); `2026-08-29-weekend-audit-p1.md:79` (0.3 proven live) | **filter** (nothing outside is generated or seated) | **NO — research/boot 0.3, resolved 1.0** |
| D8 | cluster tolerance | `:678` `LevelClusterTicks=12`; `:682-685` | **3.00 pt** (12 × 0.25) | [I]; [R]-weak | `belief-census.md:76`; `2026-08-24-level-grading-full-audit.md:62` ("defensible per external sources ✓") | filter (collapse) | **yes** |
| D8b | confluence band (mislabelled "cluster tolerance" in code) | `:418` | **0.10 × dATR = ±51 pt today** | [I] | `2026-08-26 mega-research S1`, quoted in code at `:415-417` | weight input (family confluence count) | **yes** (documented-not-changed) |
| D9 | swing seats | `:104-105` (evidence 0.85) + `:783` (volume-family membership) + `kernel/levels_swing.go:38` | seated, evidence 0.85, **reserved-seat protected** | **[T]** (census's [X] is a sign error) | `grand-audit.md:74`; `2026-08-27-level-truth-wave.md:24`; `2026-08-27-deep-verify-22.md:38` | weight + seat guarantee | **yes** (still seated, as the tape supports) — the census LABEL does not conform |
| D10 | consumed/3rd-touch/far-HTF → `target_only` | `kernel/levels_role.go:24,109-112,117` | consumed → `RoleTargetOnly`; HTF non-reversal zone → target_only | [I] doctrine | `belief-census.md:78` | role demotion (label in prompt; never a gate) | **yes** |
| — | seats / `max_levels` | boot: `kernel/levels_volume_boot.go:14` (const 8); resolver `kernel/engine_analysis.go:366-368`, `trader/auto_trader_planner.go:1982-1988`; hard ceiling `kernel/plan_doc.go:363` = 12 | **12** (at the hard ceiling) | [M] mechanics | boot line vs `🗺️ seated 12/526` | seat cap | **NO — boot 8, resolved 12** |
| — | confluence cap C14 | `:192-200` | **3** (env `CONFLUENCE_CAP` unset) | [I] | `belief-census.md` C-section absent; code cites "research shows diminishing returns" with no source | weight cap | **yes** — live-proven: VWAP−1σ scored 1.44 = 0.90×(1+0.2×3) |
| — | grade thresholds | `:663-670` | A ≥ 1.00 · B ≥ 0.70 · else C | [I] | none found | grade label | **yes** |
| — | zone tier floors/caps | `:490-513` | 1m → forced C · 15m → forced B · 1h/4h → C promoted to B, A reachable | [O] (1h wave 08-25) | `2026-08-24-level-grading-full-audit.md:50-53` | grade clamp | **yes** |
| — | B2 Tier-1 proximity gate | `:257` `Tier1ProximityTicks=12`; gate `:515-519`; `withinTier1Proximity` `:298-320` | zone may exceed C only within **3.00 pt** of a Tier-1 anchor | [O] (Pack B override 2026-08-26) | `2026-08-26-packb-volume-levels.md` (wave report; no rate) | grade clamp | **yes** |
| — | `seatHTF` | `:920-…` (`maxHTFSeats = 2`) | 2 seats, eligibility = Tier-1 kind **or** reversal zone (`:766-777`) | [O] (G2/G3/G6 + B2) | `2026-08-24-level-grading-full-audit.md` §HTF mandate | seat guarantee | **yes** |
| — | `SeatVolumeFamily` (E1) | `:794-841`, membership `:783` | 1 reserved seat for VWAP/eVWAP/pdVWAP/VWAP2σ/POC/nPOC/VAH/VAL/SETT/MID-O **+ SWG-H/SWG-L** | [O] Pack B | `2026-08-26-packb-volume-levels.md` | seat guarantee | **yes on values; see F1 (SWG is not a volume family)** |
| — | `Seat1HZone` | `:870-…`; call sites `kernel/levels_assemble.go:35`, `trader/auto_trader_planner.go:2136,2159` | ON (`seat_1h_zone=true`, boot line + DB) | [O] (1h wave R1) | `2026-08-25-1h-timeframe-research-wave.md` | seat guarantee | **yes** |
| — | `seatBothSides` / `MinSideLevels` | `:744`, `:770-…` | **3** per side (seating target only; the per-side VALIDATION quota was removed by owner ruling 2026-08-31) | [O] | code comment `:740-743` | seat rebalance | **yes** |
| — | D1′ detector | `kernel/detector_d1prime.go:120`; boot line `:279-281`; recorder `trader/detector_record.go:23` | k=3 · Δ resolved per read (12.73–13.40 pt band live) · H=12 · exit_on=close | [T] | `2026-09-02-detector-redesign.md` (D1′ amendment); `level-kind-replay.md:29-36` | **advisory / telemetry — ZERO gates** | **yes** |
| — | `HoldRate` / `FormatHoldRate` | `kernel/detector_d1prime.go:199`, `:235` | live code, **0 production callers** | [M] | — | none | **DEAD — test-only** |
| — | `CandidatePoolStore.LatestPool` | `store/candidate_pool.go:103` | **0 production callers** | [M] | — | none | **DEAD — test-only** |
| — | `TouchOutcomeStore.RatesBy` / `DetectorReport` | `store/touch_outcomes.go:131`, `:189` | reachable only from `cmd/detector-report/main.go:42` (a separate CLI binary) | [M] | — | none in the trading binary | **DEAD in the live loop** |

---

## 2. D5 PART 1 — "confirm none moved": WHICH CHANGED? **NONE.**

Diffed three ways [A]:

1. **Content hash** — `kernel/levels_score.go` md5 `dfcda708e8cad3e2b03af3affe4df5d1` at census commit `ee64a494`, at deployed rev `70af663d`, and in the worktree. `typeEvidence` alone (lines 87-127) hashes `3e0cb58567b3b8999a3012a2d77945e4` in both.
2. **History** — `git log -- kernel/levels_score.go` newest entry is `e86ae805 2026-08-31 08:11:57 -0500`. Nothing since 2026-08-31 touches `levels_score.go`, `levels_swing.go` or `levels_role.go`.
3. **Deployed-vs-worktree** — `git diff --stat 70af663d -- kernel/levels_score.go kernel/levels_swing.go kernel/levels_role.go kernel/detector_d1prime.go kernel/detector_recorder.go trader/detector_record.go store/touch_outcomes.go store/candidate_pool.go` → **empty**.

Term-by-term against the census row values (`belief-census.md:69-78`):

| census row | census value | value NOW | moved? |
|---|---|---|---|
| D2 zoneTFMult | 1.0/1.1/1.2/1.3 | 1.0/1.1/1.2/1.3 | no |
| D2 zoneEvidenceByKind | "tiers" (values not enumerated) | OB .40/.50/.70/.72; others .35/.45/.65/.65 | no |
| D3 zoneReversalBonus | ×1.1 | ×1.1 | no |
| D4 zoneSizeMult | ≤0.3 ×1.25 … >2.5 ×0.50 | identical 6-band ladder | no |
| D5 zone freshness | 1/.6/.3/.15 | 1.0/0.6/0.3/0.15 | no |
| D5 anchor freshness | 1/.8/.6/.5 | 1.0/0.8/0.6/0.5 | no |
| D6 kind weights | swing .85, VWAP .85, VAH/VAL/SETT .80, MID-O .60, Round/Gap .55, zone-only .30 | swing .85 ✓, **VWAP 0.90 ✗**, VAH/VAL/SETT .80 ✓, MID-O .60 ✓, Round/Gap .55 ✓, zone .30 ✓ | **no — the code did not move; the census row was wrong when written** |
| D7 proximity | ±0.3×dATR (owner) / default 1.5 | **1.0** resolved | **the CONFIG moved, the code did not** |
| D8 cluster tolerance | 12t (3pt) | 12t (3pt) | no |

The census's `file:line` citations are 1–3 lines stale in places (`:159`→`:160`, `:149-157`→`:148-157`, `:205-222`→`:205-227`, D5's `:437` is the *call site* not the ladder definitions at `:359`/`:378`). Because the file is byte-identical, these are citation imprecision, **not drift**.

**Live-arithmetic proof that the ladders are the ones actually executing** [A] — from `candidate_pool`, read `1788528720523` (2026-09-04 08:32:00 CT):

* `Demand·1h` score **1.716** = `0.65 (zoneEvidenceByKind[Demand][1h]) × 1.1 (reversal) × 1.25 (zoneSizeMult ≤0.30) × 1.6 (1+0.2×3, capped) × 1.2 (zoneTFMult[1h])` — exact to 3 dp. Confirms zoneEvidenceByKind, zoneReversalBonus, zoneSizeMult, ConfluenceCap and zoneTFMult all live and multiplying as specified.
* `VWAP−1σ` score **1.44** = `0.90 × 1.6` — confirms `typeEvidence(KindVWAP)=0.90` (**not** the census's 0.85) and the cap=3.
* `SWG-H` score **1.36** = `0.85 × 1.6` — confirms swing evidence 0.85.
* `PDL` score **1.92** = `1.00 × 1.6 × 1.2 (htf)` — confirms the HTF line multiplier 1.2.

---

## 3. D5 PART 2 — D1′ LIVE RATES (touch_outcomes)

**Snapshot frozen at `id ≤ 424`** (`max(id)=424`, `count=424`, taken 2026-09-04 08:46:27 CT). The table is being written live — it was 359 rows when this dispatch was drafted and 424 twenty minutes later, so any quoted rate must carry its snapshot. Episode span: `opened_at` 2026-09-02 22:10 CT → 2026-09-04 08:31 CT; all `created_at` on 2026-09-04 (first write 05:30:54 CT — the detector hook backfills the whole void scope on its first read after boot). Detector params on every row: `k=3.0, H=12, exit_on=close`, band 12.73–13.40 pt.

`p(hold) = hold / (hold + break)`; `ambiguous_horizon` recorded and **excluded** from the denominator (`kernel/detector_d1prime.go:199-210`).

### 3.1 Overall
| cell | rows | hold | break | **n** | **p(hold)** | ambiguous |
|---|---|---|---|---|---|---|
| ALL | 424 | 162 | 158 | **320** | **0.5062** | 104 (24.5% of rows) |

### 3.2 By level_kind — floor n=30, everything below SUPPRESSED
| level_kind | rows | hold | break | **n** | **p(hold)** | ambiguous |
|---|---|---|---|---|---|---|
| DEMAND | 90 | 44 | 41 | **85** | **0.5176** | 5 |
| VWAP | 114 | 43 | 27 | **70** | **0.6143** | 44 |
| RTH-L | 68 | 20 | 43 | **63** | **0.3175** | 5 |
| *OR-H, SWG-H, SUPPLY, SWG-L, OR-L, PDC, PDH, OB, RTH-H, eVWAP, VWAP±2σ, ONH, ONL, PDL, EQL* | — | — | — | **n = 15, 15, 14, 8, 7, 7, 6, 5, 5, 5, 4, 3, 3, 3, 2** | **SUPPRESSED (n<30)** | — |

Only **3 of 18 kinds** clear n=30 after 2 days of live recording. VWAP's 0.6143 is also the cell with the worst ambiguity (44 of 114 rows = 39% ambiguous) — a p computed on 61% of its own episodes.

### 3.3 By session — floor n=30
| session | rows | hold | break | **n** | **p(hold)** | ambiguous |
|---|---|---|---|---|---|---|
| NY | 180 | 77 | 76 | **153** | **0.5033** | 27 |
| LONDON | 163 | 60 | 60 | **120** | **0.5000** | 43 |
| ASIA | 81 | 25 | 22 | **47** | **0.5319** | 34 |

### 3.4 By touch ordinal — floor n=30
| ordinal | rows | hold | break | **n** | **p(hold)** |
|---|---|---|---|---|---|
| 1 | 304 | 118 | 124 | **242** | **0.4876** |
| 2–14 | 120 | 44 | 34 | **n = 8,14,13,11,11,6,6,1,2,2,2,1,1** | **ALL SUPPRESSED (n<30)** |

The ordinal-decay hypothesis (H8 in `level-kind-replay.md`) **cannot be tested at all** on the live table: only ordinal 1 clears the floor. Nothing here distinguishes 0.50 anywhere.

CSVs (frozen at `id ≤ 424`): `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/D5b-touch_outcomes-by-kind.csv`, `-by-session.csv`, `-by-ordinal.csv`, plus `D5b-candidate_pool-by-kind.csv`.

`candidate_pool` (192 rows at dispatch time, still 192 at 08:46): 8 reads, 24 candidates each, 12 seated. Distribution of the cut: OB 60 candidates / 6 seated · DEMAND 30/13 · SUPPLY 16/1 — the zone family supplies 106 of 192 candidates (55%) and takes 20 of 96 seats (21%).

---

## 4. IS D1′ STILL DESCRIPTIVE ONLY? **YES — proven by exhaustive reader census (A29). [A]**

Every Go reference to the two tables or their stores, excluding `_test.go`:

| site | direction | what it does |
|---|---|---|
| `trader/detector_record.go:50,62,73` | **WRITE** | `NextOrdinal` / `LastOpenedAtMs` watermark + `SaveOutcome` |
| `trader/detector_record.go:83,101` | **WRITE** | `BuildCandidatePool` → `SavePool` |
| `trader/auto_trader_planner.go:2333` | call site | the **one** production caller of `recordDetectorOutputs` — after the plan is authored, "changes no decision" (`trader/detector_record.go:12-14`) |
| `main.go:404` | **READ** | `CountOutcomes()` + `CountPool()` → two integers interpolated into the boot **log line** (`kernel/detector_d1prime.go:279`). Feeds nothing else. |
| `cmd/detector-report/main.go:42,44,46` | READ | a **separate CLI binary**, not linked into the trading loop |
| `kernel/level_stats_calc.go:79` | — | a **comment** naming `store.TouchOutcomeStore` as the calibrated replacement for the retired `Reacted` verdict; no call |
| `store/touch_outcomes.go:131,189,254`, `store/candidate_pool.go:69,93,103,113` | store methods | `RatesBy`/`DetectorReport` reachable only from the CLI; `LatestPool` from tests only |
| `kernel/detector_d1prime.go:199,235` (`HoldRate`, `FormatHoldRate`) | — | **0 production callers** — `kernel/detector_d1prime_test.go:78,137,156,169,188` only |

**Verdict:** the trading binary reads these tables in exactly one place — `main.go:404`, to print two counts in a log line. No risk gate, arm gate, validator, seat rule, grade, weight, prompt block or role assignment consumes a single row. `recordDetectorOutputs` is defensively wrapped (`defer recover()` at `trader/detector_record.go:33-38`) precisely so it cannot influence the loop. The boot line's own words — *"advisory, zero gates"* for the sibling telemetry, and `DetectorBootLine`'s "legacy rates retired (grep-verified: no surface renders one)" — hold.

**Three DEAD rules to say loudly (A29):** `HoldRate` and `FormatHoldRate` (`kernel/detector_d1prime.go:199,235`) and `CandidatePoolStore.LatestPool` (`store/candidate_pool.go:103`) have **zero production callers**. `RatesBy`/`DetectorReport` are dead **inside the running binary** — they live only in `cmd/detector-report`, which is not the deployed process. The calibrated rate machinery exists and nothing in the live loop can see it.

---

## 5. FINDINGS (new, not in the census)

**F1 — SWG-H/SWG-L are members of `isVolumeFamilyKind`. [A]**
`kernel/levels_score.go:783` lists `KindSWGH, KindSWGL` alongside VWAP/POC/VAH/VAL/SETT/MID-O. Swing points are not a volume family. Two consequences: (a) the E1 seat guarantee — whose stated acceptance is *"the first plan's seated table carries VWAP/POC"* (`:791-793`) — is **satisfied by a swing high**, so a read can pass E1 with zero volume levels seated; (b) a seated SWG row is **immune to `seatBothSides` eviction** (`:820`, `!isVolumeFamilyKind`), a protection meant for volume rows. Note also that `levelFamily()` (`:325-355`) has **no** SWG case, so swing falls to `"other"` for confluence counting — the same file carries two contradictory family definitions for the same kind.

**F2 — the boot line prints two file constants as if resolved (A11/class-45). [A]**
`kernel/levels_volume_boot.go:14` — `seats=%d` ← `DefaultMaxLevels` (8) while resolved is 12; `retuned 0.3` is a string literal while resolved is 1.0. Both are read at process level before any trader resolves, so the honest form is either `n/a` or the per-trader line at first arm cycle (the same file already does that correctly for the condition ledger at `:33-35`). A reader of boot 8 would conclude the day-trade lock is ±153pt; it is ±510pt.

**F3 — a hardcoded seat cap of 8 on the watcher path. [A]**
`trader/auto_trader_watcher.go:324` passes literal `8` to `AssembleScoredLevels` while the planner and engine paths resolve 12 (`engine_analysis.go:366`, `auto_trader_planner.go:1987`). The watcher therefore grades against a **different seated table** than the plan it is watching.

**F4 — `candidate_pool.score` is 0.0 for every cut row. [B]**
`kernel/detector_recorder.go:70-72,80-89`: cut records are constructed with the zero value and never assigned a score, so a cut candidate is stored as `score=0.0`, indistinguishable from a genuinely zero-scoring level (live: OB avg_score 0.156 across 60 rows because 54 of them are 0.0). `Components` is handled correctly (`"{}"` = not computed, explicitly per A24 at `:28`); `Score` is not. Class 49 ("an empty computed list is `[]`; an uncomputed one is absent") applies.

**F5 — `confBand` is commented "cluster tolerance". [A]**
`kernel/levels_score.go:418` `confBand := 0.10 * dATR // cluster tolerance` — it is the **confluence** band (±51 pt today); the actual cluster-collapse tolerance is 3.00 pt at `:678`. A reader auditing "cluster tolerance 12t (3pt)" against this line finds a 17× mismatch that is purely a stale comment.

---

## 6. COMMANDS USED (reproducible, read-only)

```
cd /home/hoang/vl-conform
git show ee64a494:kernel/levels_score.go | md5sum
git show 70af663d:kernel/levels_score.go | md5sum
md5sum kernel/levels_score.go
git log -12 --format='%h %ci %s' -- kernel/levels_score.go
git diff --stat 70af663d -- kernel/levels_score.go kernel/levels_swing.go kernel/levels_role.go

sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select json_extract(config,'\$.day_plan') from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8';"
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" \
  "select date(datetime(open_time_ms/1000,'unixepoch','+2 hours')) d, round(max(h)-min(l),2)
   from bars where symbol='MNQ' and tf='1m' group by 1;"     -- session-day roll = 17:00 CT
grep -h "proximity band" /home/hoang/vl/data/vl_2026-09-04.log | sed 's/.*proximity band/proximity band/' | sort | uniq -c
```

---

## 7. WHAT COULD NOT BE MEASURED

* `/api/config/resolved` and `/api/risk/gate-blocks` return `{"error":"Missing Authorization header"}` from this session — every "resolved" value above comes from the DB config the process loaded, the boot lines, or live log arithmetic, never from the API.
* **"Round 6.2" could not be located.** There is no rounds corpus (`docs/superpowers/research/` holds only `INDEX.md`), and `grep -rn "6\.2"` finds no grader baseline in the census or the index. The D5 comparison was therefore run against the **level-kind replay** (`2026-09-02-level-kind-replay.md:221-236, 359-366`), the **belief census** (`:69-78`), and the **grading audit** (`2026-08-24-level-grading-full-audit.md:47-65`), all cited above.
* Missed-turns has **not** been re-run at the live proximity of 1.0. Every published figure (`grand-audit.md:74` at the 08-27 default; `2026-08-29-pre-livefire-verify.md:61` = 60.9/59.1/72.7% at 0.3) was measured at a *different* band. The D9 support is real but was measured under a lock 3.3× tighter than the one running now.
* `candidate_pool` history begins 2026-09-04 00:30:54 CT (8 reads). No seated-table history exists before that, so grade-distribution drift over time is unmeasurable.
* 15 of 18 level kinds and 13 of 14 touch ordinals fall below n=30 and are reported as SUPPRESSED rather than quoted.
