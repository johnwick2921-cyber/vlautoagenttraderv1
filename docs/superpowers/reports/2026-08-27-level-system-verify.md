# Level-System Verification — spec vs deployed, grades, pipeline, reactions (2026-08-27)

**How this run happened.** The dispatch supplied the grading spec as ground truth;
the owner was unavailable ("review later — work autonomously"). Scope: read-only
verification of the supplied constants against the deployed binary, plus the
canceled 6-part deep verification, compressed to what the stored evidence can
bear. **Another dispatch is mid-edit on `feat/armed-orders` (WIP: `store/store.go`,
`kernel/plan_doc.go`, new `kernel/armed.go`, `store/armed_orders.go`; the branch
HEAD does not compile — `s.ArmedOrders` missing). This run touched nothing of
theirs; tests ran on a clean worktree at `dev` `a52de628`.**

Boot facts: bot PID 2701087, `deploy/RELEASE` = `99b96b15493e` (plan-lifecycle
wave, PR #85). All DB access `file:data/data.db?mode=ro`. All times CT. No code,
config, or DB changes; only this report file was created.

---

## §0 — SPEC vs DEPLOYED CODE (the constants you sent, one line per item)

| Spec item | Deployed (rev 99b96b15, `kernel/levels_score.go`) | Verdict |
|---|---|---|
| score = evidence × freshness × (1+0.20×conf, cap 3, distinct-family) × TFmult [×1.1 reversal] | zone: `zoneEvidence`(reversal inside) × **zoneSizeMult** × fresh × (1+0.2·conf) × `zoneTFMult`; non-zone: `typeEvidence` × fresh × (1+0.2·conf) × **1.2 if HTF** | formula shape ✓, but **two extra factors not in spec**: `zoneSizeMult` (0.50–1.25 banded) and the non-zone HTF ×1.2 |
| Grade A≥1.0 · B≥0.70 | `gradeFromScore`: A≥1.0, B≥0.70, else C | ✓ |
| Floors/caps 1m ≤C · 15m B..B · 1h B..A · 4h B..A | identical switch in `scoreLevelsPool`, **plus a spec-silent override: B2 — a zone graded B/A is demoted to C unless within 12 ticks (3.00 pts) of a Tier-1 anchor** (`withinTier1Proximity`) | ✓ plus B2 override |
| Zones OB .40/.50/.70/.72 | `KindOB {"1m":.40,"15m":.50,"1h":.70,"4h":.72}` | ✓ |
| S&D & FVG .35/.45/.65/.65 | `KindFVG/Supply/Demand` all `.35/.45/.65/.65` | ✓ |
| **iFVG .30 flat** | `KindIFVG {"1m":.35,"15m":.45,"1h":.65,"4h":.65}` (same as FVG) | **✗ divergence** |
| Volume: VWAP 1.0 | `KindVWAP` → **0.90** (and ±1σ band lines SHARE this kind) | **✗** |
| pdPOC 1.0 | `KindPOC` → **0.90** | **✗** |
| VAH/VAL .95 | → **0.80** | **✗** |
| nPOC .95 | → **0.85** | **✗** |
| **±2σ .85** | **not emitted at all** (`SessionVWAPLevels` emits VWAP and ±1σ only) | **✗ missing detector** |
| ±1σ .70 | shares `KindVWAP` → **0.90** | **✗** |
| pdVWAP .70 | `KindPDVWAP` → **0.85** | **✗** |
| SETT .70 | → **0.80** | **✗** |
| MID-O .55 | `KindMIDO` → **0.60** | **✗** |
| Freshness zones 1.0/0.6/0.3/0.15 | `zoneFreshMult` 1.0/0.6/0.3/0.15 | ✓ |
| **Hard anchors NO decay** | `freshMult` for anchors **1.0/0.8/0.6/0.5** (W7 ladder still decays anchors; live: PDC shown `flipped·x81` grade B, ONH `flipped·x23` C) | **✗ divergence** |
| TFmult 1.0/1.1/1.2/1.3 | `zoneTFMult` ✓; tier routing `""/1m/3m/5m→1m, 30m→15m, 2h→1h, 6h/8h/12h→4h, unknown→1m` | ✓ |
| Reversal ×1.1 | `zoneReversalBonus` 1.1 (folded into `zoneEvidence`) | ✓ |
| Confluence cap 3 distinct-family | `ConfluenceCap()` (env override, default 3) + `levelFamily` distinct-family counting | ✓ |

**Headline:** the zone table and the formula skeleton match; **every volume-family
weight diverges, ±2σ doesn't exist, iFVG isn't flat-0.30, and hard anchors still
decay.** If these constants are the intended as-deployed contract, six of the
nine volume rows, iFVG, and the anchor-freshness rule need code changes. Nothing
was changed here.

---

## §1 — PART 1: DETECTOR TRUTH (last complete NY session: 2026-08-26)

Independent recomputation from the persisted `bars` table (deduped by max(rowid)
per key; 1m → aggregates), as-of 11:56 CT, vs the system's executor KEY LEVELS
block of that minute and `level_state`:

| Kind | System (11:56 KEY LEVELS) | Recomputed from bars | Verdict |
|---|---|---|---|
| PDH | 29420.00 | 29420.00 (08-25 full-day high) | **MATCH** |
| PDL | 29095.50 | 29095.50 (08-25 full-day low) | **MATCH** |
| PDC | 29228.50 | 29228.50 (08-25 last 1m close) | **MATCH** |
| RTH-H | 29416.00 | 29416.00 (08-25 NY 08:30–14:45 high) | **MATCH** |
| RTH-L | 29145.50 | 29145.50 (08-25 NY low) | **MATCH** |
| ONH | 29310.00 | 29310.00 (overnight 08-25 17:00→08-26 08:30 high; the 17:00 bar, H=29310.00) | **MATCH** |
| ONL | (not seated; ONL 29464 in 08-27 plan) | overnight low 29095.50 (duplicates PDL → collapse explains absence) | consistent |
| VWAP | 29251.15 (bias_ctx) | 29227.15 full-session; window-dependent 29227–29245 — **cannot reproduce 29251.15 from stored bars** | **mismatch (observation §F8)** |

6/6 structural anchors exact, with bar timestamps quoted above. VWAP is the one
non-exact volume-family item: the executor's `bias_ctx` VWAP is computed over a
snapshot bar window (`ComputeBiasContext` → `closedBars(bars,…)` with the passed
slice), not the full 17:00 session, and/or persisted volumes drift from the live
feed — either way, a bars-table replay cannot reproduce the live VWAP value.

Zone detectors (S/D, FVG, OB, iFVG, EQH/EQL): a full independent port was out of
scope for this run; instead, the machine's own zone facts were cross-checked
against bars — see the drought report's finding: the machine's `sweep_reclaim
@29281` / `reject @29212.50` "triggered" events were **false-positives vs the
written trigger prose** (sweep 20–120 min stale, zero reclaim closes). That is
zone-level truth divergence between the machine evaluator and bar reality.

Unknown-TF fallback regression: `go test ./kernel/ -run
TestZoneTierForUnknownFallsBackTo1m -count=1` → **PASS** (run on dev `a52de628`
worktree; the live branch does not compile due to another dispatch's WIP).
`TestGradeRank` also PASS.

---

## §2 — PART 2: GRADE MATH

### 2a. Live-proof recomputation (exact where inputs are visible)

Executor KEY LEVELS block, 11:56 CT, price 29192.75 — every factor shown, code
constants:

| Level | evidence | fresh | conf | TFmult/HTF | score | derived | system | ✓ |
|---|---|---|---|---|---|---|---|---|
| PDC 29228.50 | 1.0 | 0.5 (done) | ≥2 (prior+zone+vwap families within band) | 1.0 | 0.5×1.4=0.70 | B | B | ✓ |
| RTH-L 29145.50 | 1.0 | 0.5 | ≥2 | 1.0 | 0.70 | B | B | ✓ |
| OR-L 29174.25 | 0.70 | 0.5 | ≤3 | 1.0 | ≤0.392+… | C | C | ✓ |
| OR-H 29281.00 | 0.70 | 0.5 | ≤3 | 1.0 | 0.392–0.56 | C | C | ✓ |
| ONH 29310.00 | 0.85 | 0.5 | 3 | 1.0 | 0.68 | C | C | ✓ (0.68 < 0.70 — borderline by design) |
| PDL 29095.50 | 1.0 | 1.0 | ≥0 | 1.0 | ≥1.0 | A | A | ✓ |
| VWAP−1σ 29181.65 | 0.90 | 1.0 | ≥1 (vwap family + OB zone within band) | 1.0 | 0.90×1.2=1.08 | A | A | ✓ |
| OB(bull)·1h 29201.50 | 0.70 | 1.0 | ≥1 | 1.2 | 0.70×1.2×1.2=1.008 | A | A | ✓ |

Under the **spec** constants the same rows would flip: PDC/RTH-L/OR-H/ONH all
lose their anchor decay (`×1.0` instead of ×0.5) → PDC 1.4→**A**, ONH 1.36→**A**,
OR-H 1.12→**A** — i.e. the "hard anchors NO decay" rule alone would promote 4 of
the 8 seated rows. Floors/caps observed live: 4h `OB(bull)·4h 29162.75 A/A`
(08-24 ASIA v3) = 4h reaching A ✓. No 15m-tier zone was seated in the last 7
days of plans, so the 15m B..B cap has no live row to quote — it is exercised by
the unit matrix (`levels_htf_test.go`) instead. One live floor anomaly: 08-26 NY
v3 row `OB(bull)·1h 29212.50 grade=C machine_grade=C` — a 1h-labeled zone stamped
**below** the 1h B-floor (tier-tag/label mismatch candidate, FAIL F4).

### 2b. 7-day census (08-20 → 08-26 plans, 795 levels, 539 stamped)

Independent recompute of evidence×freshness×(1+0.2·conf)×TFmult from the plan's
own level set matches the stored `machine_grade` in **172/539 (32%)** — and this
number is dominated by approximation, not bugs: write-time freshness, the full
detector pool, and dATR are not persisted, so confluence/freshness inputs cannot
be reconstructed exactly. The census is therefore NOT a verdict on the scorer.
What IS exact:

- **Stamp gap persists:** 256/795 levels (32%) carry no `machine_grade`.
  By day: 08-20 → 08-23 plans are **100% unstamped** (pre-fix); but recent days
  still leak: 08-24 ASIA 14/54, 08-25 ASIA 41/153, 08-25 LONDON 2/24,
  **08-26 ASIA 15/80, 08-26 LONDON 30/154, 08-26 NY 14/64**, 08-27 LONDON 3/47.
  Dominant unstamped labels: EQH 37, EQL 35, ONH 24, PDC 21, PDH 19 — the HTF
  merge fix (c1cf4fdb) covered HTF-section rows, not these. **FAIL F5.**
- Reversal ×1.1 and confluence-cap-3 mechanics are covered by unit tests and
  were not independently re-derived here (no ZonePattern persisted in plan docs).

---

## §3 — PART 3: PIPELINE TRACE — PDC 29228.50, one level end-to-end

1. **detected** — `ExtractMultiDayLevels`: prior calendar day's last 1m close;
   recomputed here = 29228.50 (08-25 1m close) ✓.
2. **scored** — evidence 1.0 × fresh 0.5 (done, 81 touches) × (1+0.2×conf) →
   0.70 → **B** (§2a row 1).
3. **cluster-collapse** — tolerance 12 ticks (3.00 pts): no sibling within 3.00
   (PDC is alone at 29228.50) → survives collapse.
4. **seating** — seated in the KEY LEVELS block (`29228.50 PDC B flipped·x81
   target_only +35.8`).
5. **planner row** — rendered in the 09:30 DAY PLAN: `29228.50 PDC [B] target`.
6. **plan JSON** — NY v7 levels[]: `{"price":29228.5,"label":"PDC","grade":"B",
   "machine_grade":"B"}` (quote from plans table).
7. **machine stamp** — `m:B` stamped by rounded price at write site ✓ (this
   level IS stamped; see F5 for the ones that aren't).
8. **card render** — `Dist` recompute check: 11:56 card shows `+35.8`; price
   29192.75 → 29228.50−29192.75 = +35.75 → +35.8 ✓. All 8 rows of that block
   check out (±0.05 rounding). **Sign convention note:** KEY LEVELS uses
   level−price (`+35.8`), PLAN STATUS uses price−level (`dist −34.5` for the
   same level from below) — two renders, opposite signs. **F7.**
9. **executor KEY LEVELS block** — verbatim: `29228.50  PDC  B  flipped·x81
   target_only     +35.8` (minGrade scope verified: min_grade=B does not cut it).
10. **level_state** — `PDH/PDL/PDC||23382 price 29228.5 times_tested=1
    consumed=1 done` (touch 00:55 CT, 08-26).
11. **scenario citing it** — NY v7 `S1 [B] reject …` → `confirm: 1x5m_close
    @29228.5 side above — MET` (advisory line), scenario never triggered per
    prose (5m closes never printed above; machine `reject occurred` lines at
    10:59/11:00 were the F9 false-positives).
12. **outcome** — no entry on PDC; plan died 11:15 (6th flip) → no_trade.
    No unexplained mutation or disappearance at any hop; the only hop with a
    known defect is the scenario-status evaluator (F9).

---

## §4 — PART 4: REACTION REALITY (8 sessions within bars coverage 08-24 14:21 →)

Seated-level union per session (all plan versions), evaluated with the exact
`EvaluateLevelOutcome` spec (±4 pts touch, ≥8 pts within 3 bars, ≥8 pt clean
break, ≥3 touches chop) over deduped 1m bars:

| Grade | n | touch% | react% | broke% | chop% |
|---|---|---|---|---|---|
| A | 243 | 76.5 | 60.5 | 64.6 | 11.9 |
| B | 14 | 71.4 | 64.3 | 50.0 | 21.4 |
| C | 17 | 94.1 | 70.6 | 82.4 | 11.8 |

By kind: anchors touched 132/158 (84%), volume 32/37 (86%), zones 48/79 (61%).
By session: 08-24 ASIA 82%, 08-25 ASIA 72%, 08-25 LONDON 42%, 08-25 NY 87%,
08-26 ASIA 100%, 08-26 LONDON 56%, 08-26 NY 50%, 08-27 LONDON 83%.

**Grades predictive: NO, on this sample.** Touch rate is *inverted* (C 94% >
A 76%) and C breaks clean more (82% vs 65%). The honest reading: C-graded rows
are mostly consumed/flipped levels sitting close to price, so they "touch" and
"break" by construction; A-grade anchors touch less but still break clean 2/3 of
the time when touched — the grade is not yet separating reaction quality, and
the B sample is too small (14) to say anything. This is exactly what the
`level_stats` nightly job is for (table exists, currently **0 rows** — it has
not populated yet).

**Inverse (turning points NOT seated):** 5m fractal swings (2-left/2-right) per
session: 171 swings total, **74 (43%) have no seated level within ±4 pts**.
Worst: 08-26 NY 94% (16/17), 08-25 LONDON 81%, 08-25 NY 47%. The map is
prior-day/overnight-anchored; intraday-formed swings only reach seats when a
zone detector catches them — 43% of swing turns are invisible to the map.

**Dist spot-check:** 8/8 rows exact (§3.8). Sign-convention inconsistency F7.

---

## §5 — PART 5: LIFECYCLE + EDGE CASES

- **Zone consumption:** level_state `S/D+FVG/OB`: 33 rows → 11 fresh-A, 3 B,
  **19 consumed (58%)**. Anomaly: **8 rows are `consumed=1` with `times_tested=0`
  and `last_play_ms=0`** (e.g. 29154.38, born 08-24 22:39, never touched) —
  consumed without any touch event. **F6.**
- **Consumed → role-flip live example:** zone 29162.75 (OB·4h A/A in the
  08-24 ASIA v3 plan) — touched 08-24 21:32, now `done/consumed`, rendered
  `flipped` on the map. Ladder holds (birth-scoped counting per plan birth).
- **Session roll (17:00 CT):** before roll the map cited PDH 29420 (08-25 high);
  after roll level_state carries `PDH/PDL/PDC … 29655.75` born 08-26 22:23 CT,
  and the 08-27 plans seat `PDH 29655.75` — 29655.75 = 08-26 full-day high,
  recomputed exactly. **RTH-H→PDH is not the roll rule (PDH = full-day high);
  the roll itself is verified exact.**
- **Midnight wrap (ASIA):** plans keyed `2026-08-26:ASIA` span calendar
  08-26 17:00 → 08-27 02:00 CT and remain active across midnight (v1 17:06 CT →
  v7 21:09 CT, all one trade_date) ✓.
- **Cross-TF dedup:** zero same-price/different-label duplicates inside any
  single plan version (checked all 89 plans) — collapse is holding at plan level.
- **Wide-zone sanity + proximity-band edge:** not independently reproducible —
  zone Lo/Hi widths are not persisted in plans (single price rows only); flag as
  a persistence gap for the level_stats job.
- **DST-safe stamps:** no DST transition inside the evidence window (next: Nov);
  all stamps CT-anchored via `chicago()` ✓.

---

## §6 — PART 6: VERDICT

**Sound per part:** §0 spec-vs-code — **NO** (six volume weights, iFVG, anchor
decay, and the ±2σ detector diverge from the supplied spec; plus two spec-silent
factors and the B2 override). Part 1 — **YES** (6/6 structural anchors exact;
VWAP unreproducible from stored bars). Part 2 — **PARTIAL** (live-proof rows
match; 7-day census cannot be exact — inputs not persisted; stamp gap persists).
Part 3 — **YES** (PDC trace clean at every hop). Part 4 — **grades predictive:
NO, with numbers above**; 43% of swing turns unseated. Part 5 — **PARTIAL**
(roll/consumption/dedup verified; consumed-without-touch anomaly + missing
width persistence).

**FAIL register:**

| # | Finding | Evidence | Size | Mislead today? |
|---|---|---|---|---|
| F1 | Volume-family evidence all off-spec: VWAP/pdPOC .90 vs 1.0; VAH/VAL .80 vs .95; nPOC .85 vs .95; pdVWAP .85 vs .70; SETT .80 vs .70; MID-O .60 vs .55; ±1σ shares VWAP .90 vs .70; **±2σ not emitted** | `typeEvidence` vs supplied table | M | Yes — calibration against the spec constants would use wrong weights |
| F2 | iFVG evidence .35/.45/.65/.65 vs spec .30 flat | `zoneEvidenceByKind` | S/M | Yes — iFVG grades higher than spec |
| F3 | Hard anchors still decay (1.0/0.8/0.6/0.5) vs spec "NO decay" | `freshMult`; live PDC B / ONH C after touches | M | Yes — spec would promote 4 of 8 seated rows (PDC/ONH/OR-H/RTH-L → A) |
| F4 | 1h-tier zone stamped C (below 1h B-floor) | 08-26 NY v3 `OB(bull)·1h 29212.50` grade C machine C | S | Possibly (tier-tag/label mismatch) |
| F5 | machine_grade stamp gap persists (256/795 unstamped; 3–41 per recent session) | per-day table §2b | S/M | Yes — card `m:` chips absent for those rows |
| F6 | 8 zones consumed with zero recorded touches (times_tested=0, last_play=0) | level_state | S | Yes — zone demoted to 0.15 fresh without a touch |
| F7 | Dist sign convention opposite between KEY LEVELS (level−price) and PLAN STATUS (price−level) | two live renders of PDC | S | Mildly — AI sees +35.8 and −34.5 for the same fact |
| F8 | Live VWAP not reproducible from stored bars (29251.15 vs 29227–29245 recomputed) | bars vs bias_ctx | S/M | Yes for replay/level_stats |
| F9 | Machine scenario-trigger evaluator false-positives vs trigger prose (drought report) | 13 `→ triggered` lines, none matching prose | M | Display-only today; dangerous if wired |
| F10 | `bars` table stores forming+final revisions as duplicate keys; stored 5m/15m aggregates inconsistent with 1m constituents (drought report) | 17,695 dup keys; 5m@08:50 O29174.25 vs 1m O29268.5 | M | Yes — any replay must dedupe by max(rowid) and aggregate upward from 1m |

**Top calibration moves (grounded only in Part-4 numbers):** (1) do not trust
grade as a reaction filter yet — C touches/breaks more than A; wait for the
level_stats nightly job before re-weighting (table exists, 0 rows); (2) the map
misses 43% of swing turns (worst in NY) — the biggest detector gap is intraday
swing capture, not zone weighting; (3) fix F5 before any grade-based stats, or
32% of rows silently vanish from every aggregate.

**What the nightly `level_stats` job will answer in 2 weeks that this one-shot
can't:** reaction rates with write-time freshness (not the post-hoc level_state
snapshot), per-family volume-weight validity (the B4 forward-validation the
provisional weights are waiting on), and a big-enough B/C sample to actually
test the grade ladder. This run's 8-session sample is too small and too
freshness-approximate to settle any of the three.
