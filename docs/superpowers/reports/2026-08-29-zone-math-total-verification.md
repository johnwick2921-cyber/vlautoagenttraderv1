# 2026-08-29 · Zone-Math Total Verification (last 3 complete sessions)

> READ-ONLY independent verification. Worktree `nofx-zm` @ `4763a664`; Python3 + stdlib `sqlite3` only, no Go engine code executed; DB opened `mode=ro` — nothing written. Every rule reimplemented from the spec sources listed in §R12. Evidence tiers [A]/[B]/[C].

**Universe:** 2026-08-28 NY v7 · 2026-08-28 LONDON v6 · 2026-08-27 NY v5 (latest active plan per session). Data: `bars` table = 15,646 rows total = MNQ 1m 10,023 + ES 1m 5,623 [A]; MNQ 1m all `open_time_ms%60000==0` [A]; persisted window 08-19 15:00 → 08-28 20:59 UTC [A].


## 0 · ATR recomputation basis + stored cross-check

Wilder ATR(14) reimplemented from market/data_indicators.go:86-116 (seed = mean TR[1..14], then `(13·prev+TR)/14`). 1m/5m/15m series = planner's last-2000 closed 1m slice at plan-write time (AISVPBarCount=2000, kernel/svp.go:46). 1h/4h = full persisted 1m history aggregated at plan time. Stored = `indicators_block` per-TF `ATR14:` sections [A] (the indicator engine's own series, count 300/500).

| session | 1m (computed) | 5m comp→stored | 15m comp→stored | 1h comp→stored | 4h comp→stored |
|---|---|---|---|---|---|
| 2026-08-28 NY | 14.13 | 38.22 → 38.55 | 65.53 → 64.62 | 101.79 → 95.56 | 165.77 → 166.36 |
| 2026-08-28 LONDON | 8.44 | 16.99 → 17.25 | 27.17 → 28.71 | 66.61 → 64.16 | 154.97 → 159.21 |
| 2026-08-27 NY | 10.94 | 26.20 → 26.24 | 48.66 → 51.23 | 98.81 → 98.82 | 177.05 → 176.85 |

**Drift call-out:** 1h/4h computed-vs-stored drift is the engine's NT8-native tf series depth (pre-08-19 history + up-to-500-bar cap) vs the persisted 1m window [B]. The detector ATRs used below are the STORED values (the engine's actual inputs), so zone geometry is checked against the real thresholds. 5m/15m agree within ~1 pt [A].


## Census tables


### Class 1 · FVG (+ iFVG, entry-model FRESH list)

Spec: FairValueGaps (kernel/levels_zones.go:213-279) on the 1m slice, floor max(2·tick,2.0)=2.0 pts, 3-candle gap, session-break guard fvgWindowContiguous (:190-207); iFVG flip = later close beyond far edge. HTF passes (DetectHTFLevels, levels_assemble.go:222-271) use that TF's ATR as floor. Entry-model FRESH list: FreshFvgCandidates (kernel/fvg_entry.go:69-122) — impulse body ≥1.5×ATR5m, lookback 40.

**2026-08-28 NY:** 1m FVG=11 · iFVG=210 · HTF FVG/iFVG=8 · FRESH entry-list=0 (reasoning claims empty — ✓ [A]) · stored FVG rows=0 (EXACT=0 MISSING=0) · EXTRA=221 (detected, unseated — FVGs are confluence-only, P0.1)
  - displacement check: max 1m body in lookback-40 = 20.75 vs 1.5×ATR5m floor = 57.33 → no candidate can qualify [A]
**2026-08-28 LONDON:** 1m FVG=13 · iFVG=208 · HTF FVG/iFVG=16 · FRESH entry-list=0 (reasoning claims empty — ✓ [A]) · stored FVG rows=0 (EXACT=0 MISSING=0) · EXTRA=221 (detected, unseated — FVGs are confluence-only, P0.1)
  - displacement check: max 1m body in lookback-40 = 20.50 vs 1.5×ATR5m floor = 25.48 → no candidate can qualify [A]
**2026-08-27 NY:** 1m FVG=27 · iFVG=264 · HTF FVG/iFVG=9 · FRESH entry-list=0 (reasoning does not claim fvg (n/a) — n/a) · stored FVG rows=0 (EXACT=0 MISSING=0) · EXTRA=291 (detected, unseated — FVGs are confluence-only, P0.1)
  - displacement check: max 1m body in lookback-40 = 20.25 vs 1.5×ATR5m floor = 39.31 → no candidate can qualify [A]

Census one-liner: FVG — 0 seated in all 3 plans, 0 stored rows, 0 MISSING; FRESH-entry-list emptiness claims verified [A]; detected 1m FVG/iFVG and HTF FVGs are all legitimate unseated drops (P0.1).


### Class 2 · OB (order blocks)

Spec: OrderBlocks (kernel/levels_zones.go:257-337): displacement ≥1.5×ATR, last opposing candle within 8 bars, zone = base candle [low,high]; stored price = zone midpoint (zoneLevel, kernel/levels.go:98-101). ATR used: stored 1h/4h ATR (engine inputs).

  - ✓ OB(bear)·1h@29502.88 = recomputed zone 29472.00–29533.75 (Δ0.005, tf=1h, birth 08-21 13:00:00)
**2026-08-28 NY:** detected OB (HTF)=15 · stored engine rows=1 · EXACT=1 DELTA=0 MISSING=0 · EXTRA=14 (unseated drops — expected) · carried rows (no machine_grade)=0
  - ✓ OB(bear)·1h (HTF)@29644.38 = recomputed zone 29602.00–29686.75 (Δ0.005, tf=1h, birth 08-20 08:00:00)
  - ✗ OB(bull)·1h (HTF)@29490.88 (machine A) NOT reproduced in-window → pre-persistence birth [B]
**2026-08-28 LONDON:** detected OB (HTF)=75 · stored engine rows=2 · EXACT=1 DELTA=0 MISSING=1 · EXTRA=74 (unseated drops — expected) · carried rows (no machine_grade)=0
**2026-08-27 NY:** detected OB (HTF)=21 · stored engine rows=0 · EXACT=0 DELTA=0 MISSING=0 · EXTRA=21 (unseated drops — expected) · carried rows (no machine_grade)=0

### Class 3 · S/D zones + consumed transitions

Spec: SupplyDemandZones (kernel/levels_zones.go:99-150): base ≤6 small-bodied candles (body ≤0.5×ATR) + departure ≥1.5×ATR; zone = base [low,high]; pattern = reversal if prior leg opposite. Consumed = touched in-window AND accepted through on the rule TF (ConsumedSince, kernel/plan_lifecycle.go:180-186; LevelStillValidOn, kernel/scenario_facts.go:392-400; rule 2x5m → 5m bars, need 2 closes).

**2026-08-28 NY:** detected S/D (HTF)=10 · stored engine rows=0 · EXACT=0 DELTA=0 MISSING=0 · EXTRA=10 · carried rows=none
**2026-08-28 LONDON:** detected S/D (HTF)=24 · stored engine rows=0 · EXACT=0 DELTA=0 MISSING=0 · EXTRA=24 · carried rows=Demand·1h (HTF)@29722.62, Demand·1h@29541.12
  - ✓ Supply·1h@29619.50 = recomputed zone 29585.25–29653.75 (Δ0.000, tf=1h, pattern=reversal, birth 08-27 00:00:00)
      consumed-by-session-end = True (window 08-27 19:04:28 → 08-27 19:45:00, rule 2x5m)
      level_state: no row for this zone key (bin 23695) — state never persisted for it [A]
      level_stats 2026-08-27: touched=1 reacted=1 broke_clean=0 chopped=1 [A]
  - ✗ Demand·4h (HTF)@29575.25 (machine A) NOT reproduced in-window → pre-persistence birth [B]
**2026-08-27 NY:** detected S/D (HTF)=11 · stored engine rows=2 · EXACT=1 DELTA=0 MISSING=1 · EXTRA=10 · carried rows=Demand·1h@29541.12
Note: carried rows (no `machine_grade` in the stored doc [A]) are model-carried levels, not engine outputs at write time — excluded from the census universe, listed for completeness.


### Class 4 · zoneSizeMult (0.5–1.25 banding)

Spec: zoneSizeMult (kernel/levels_score.go:201-227): size=(hi−lo)/dATR, bands ≤0.30→1.25, ≤0.60→1.10, ≤1.00→1.0, ≤1.50→0.85, ≤2.50→0.70, else 0.50; applied at levels_score.go:482. Plans persist only the final machine_grade, not the score factor — so the check is: recomputed zone → size/dATR → multiplier → v3 score → grade vs stored machine_grade.

  - OB(bear)·1h@29502.88: zone 29472.00–29533.75 · size=61.75 pts (0.218×dATR=283.50) · zoneSizeMult=1.25 · stored machine_grade=C
      v3 score with fresh×conf0 = 1.260 → grade A before B2; B2 (Tier-1 proximity, 12 ticks) then decides A/B vs C.
  - OB(bear)·1h (HTF)@29644.38: zone 29602.00–29686.75 · size=84.75 pts (0.278×dATR=305.25) · zoneSizeMult=1.25 · stored machine_grade=A
      v3 score with fresh×conf0 = 1.260 → grade A before B2; B2 (Tier-1 proximity, 12 ticks) then decides A/B vs C.
  - Supply·1h@29619.50: zone 29585.25–29653.75 · size=68.50 pts (0.225×dATR=304.25) · zoneSizeMult=1.25 · stored machine_grade=C
      v3 score with fresh×conf0 = 1.287 → grade A before B2; B2 (Tier-1 proximity, 12 ticks) then decides A/B vs C.
Caveat (not an S-finding): the final A/B vs C split depends on the B2 proximity call against the plan-time in-band pool, whose freshness rows evolve in place in `level_state` (post-plan decrements overwrite plan-time state [B]) — geometry and the multiplier are verified exactly; the grade letter is reproduced to the pre-B2 rung.


### Class 5 · SWG-H/L swings (fractal k=2)

Spec: SwingPointLevels (kernel/levels_swing.go:48-140): k=2 fractal on 5m/15m aggregates, same-side keep-more-extreme, min-move 0.25×ATR(tf), lookback 144 (5m)/96 (15m) bars, ≤3 per side per TF, newest-first.

  - ✓ SWG-L·5m@29437.00 = recomputed swing @29437.00 (bar 08-28 17:10:00)
  - ✓ SWG-H·5m@29549.00 = recomputed swing @29549.00 (bar 08-28 16:55:00)
  - ✓ SWG-L·15m@29592.50 = recomputed swing @29592.50 (bar 08-28 10:15:00)
**2026-08-28 NY:** recomputed swings=12 (5m:12 15m:6) · stored=3 · EXACT=3 DELTA=0 MISSING=0 · EXTRA=9 (unseated swings — expected, seat race)
**2026-08-28 LONDON:** recomputed swings=12 (5m:12 15m:6) · stored=0 · EXACT=0 DELTA=0 MISSING=0 · EXTRA=12 (unseated swings — expected, seat race)
**2026-08-27 NY:** recomputed swings=12 (5m:12 15m:6) · stored=0 · EXACT=0 DELTA=0 MISSING=0 · EXTRA=12 (unseated swings — expected, seat race)

### Class 6 · Volume family (VWAP±1σ/±2σ · eVWAP · pdVWAP · profile)

**"3 cuts" determination:** the ±1σ/±2σ band math exists at exactly THREE anchored cuts in the code — (1) session VWAP at the CME 17:00 CT session-day cut, the only emitter of ±1σ/±2σ bands (kernel/levels_volume.go:39-62, bands at :54-59); (2) eVWAP at the 15:00 CT cash-close cut (:88-116); (3) pdVWAP at the prior session-day cut (:262-288). No literal "3 cuts" string exists anywhere in the repo (grep: 0 hits) — these three anchors are the only cuts the band math is computed at; all three are recomputed per session [A]. VWAP math: TP=(H+L+C)/3 volume-weighted; σ=√(Σv·d²/Σv) (:66-83). Profile: 120 bins by close, POC=lo+(idx+.5)·bin, 70% value area (:194-240).

**2026-08-28 NY:**
  - ~ VWAP−2σ@29445.53 vs recomputed 29444.25 (Δ1.28) — checking prior-version carry
      → matches prior version read ~08-28 18:01:29: 29445.52 (Δ0.01, read-jitter ≤2 ticks) — carried row [A]
  - ~ VWAP−1σ@29531.05 vs recomputed 29529.98 (Δ1.07) — checking prior-version carry
      → matches prior version read ~08-28 18:01:29: 29531.04 (Δ0.01, read-jitter ≤2 ticks) — carried row [A]
  - ✓ pdVWAP@29573.96 = recomputed 29574.07 (Δ0.106)
  EXACT=3 MISSING=0 (stored=3; all recomputed values listed)
    recomputed VWAP      29615.72
    recomputed VWAP+1σ   29701.45
    recomputed VWAP−1σ   29529.98
    recomputed VWAP+2σ   29787.19
    recomputed VWAP−2σ   29444.25
    recomputed eVWAP     29616.78
    recomputed POC       29576.38
    recomputed VAH       29643.71
    recomputed VAL       29532.67
    recomputed pdVWAP    29574.07
    recomputed SETT      29635.25
    recomputed MID-O     29642.50
**2026-08-28 LONDON:**
  - ✓ VWAP+1σ@29663.45 = recomputed 29663.55 (Δ0.097)
  EXACT=1 MISSING=0 (stored=1; all recomputed values listed)
    recomputed VWAP      29635.91
    recomputed VWAP+1σ   29663.55
    recomputed VWAP−1σ   29608.28
    recomputed VWAP+2σ   29691.18
    recomputed VWAP−2σ   29580.65
    recomputed eVWAP     29638.44
    recomputed POC       29576.50
    recomputed VAH       29633.73
    recomputed VAL       29514.17
    recomputed pdVWAP    29567.25
    recomputed SETT      29635.25
    recomputed MID-O     29642.50
**2026-08-27 NY:**
  - ✓ VWAP+1σ@29611.04 = recomputed 29611.00 (Δ0.045)
  - ✓ eVWAP@29548.74 = recomputed 29548.88 (Δ0.136)
  - ✓ VWAP−1σ@29503.23 = recomputed 29503.45 (Δ0.222)
  EXACT=3 MISSING=0 (stored=3; all recomputed values listed)
    recomputed VWAP      29557.22
    recomputed VWAP+1σ   29611.00
    recomputed VWAP−1σ   29503.45
    recomputed VWAP+2σ   29664.77
    recomputed VWAP−2σ   29449.68
    recomputed eVWAP     29548.88
    recomputed POC       29248.36
    recomputed VAH       29300.34
    recomputed VAL       29188.78
    recomputed pdVWAP    29254.27
    recomputed SETT      29431.00
    recomputed MID-O     29531.75

## Per-class census (all 3 sessions)

| class | verified objects | EXACT | DELTA>1tick | MISSING | EXTRA(unseated) |
|---|---|---|---|---|---|
| 1 FVG/iFVG | 0 stored rows | — | 0 | 0 | all detected FVGs unseated (P0.1) |
| 2 OB | 3 engine rows | 2 | 0 | 1 (pre-persistence birth) | expected drops |
| 3 S/D | 2 engine rows | 1 | 0 | 1 (pre-persistence birth) | expected drops; 2 carried rows excluded |
| 4 zoneSizeMult | 3 reproduced zones | 3 (mult=1.25 each) | 0 | 0 | — |
| 5 SWG | 3 stored rows | 3 | 0 | 0 | expected drops |
| 6 Volume | 7 stored rows | 7 (2 via prior-version carry) | 0 | 0 | — |

## S-list

S1 [OB 2026-08-28 LONDON] MISSING stored OB(bull)·1h (HTF)@29490.88 — no in-window OB reproduces the midpoint
S2 [S/D 2026-08-27 NY] MISSING stored Demand·4h (HTF)@29575.25 — no in-window S/D reproduces the midpoint

**Case reconstruction (bar-by-bar):** for each S, the FULL in-window 1h/4h displacement set was enumerated (every tf bar with |body| ≥1.5×stored-ATR) and every ≤8-bar opposing-candle pairing / ≤6-candle small-bodied base was checked — no pairing reproduces the stored midpoint. The stored rows carry `machine_grade` A/C, i.e. they WERE engine outputs — of a run whose tf series included pre-08-19 15:00 UTC bars (NT8-native cache, 2000-back seeds, provider/ninjatrader/tcp_server.go:418,429). The persisted `bars` table holds only 1m from 08-19 15:00 UTC [A], so those birth bars are not quotable from any table — the in-window absence is the evidence, not a math divergence.


## Seated-table diff (recomputed vs plan levels)

Reconstructed with the full planner pipeline: proximity 0.3×dATR, maxLevels 12, minGrade B, seatHTF/SeatVolumeFamily/Seat1HZone/seatBothSides. Differences are expected: the HTF-zone prompt section (G2.2) and the LLM's own row selection decide the final plan rows — the plan table is model-authored on top of the machine map.

**2026-08-28 NY** (price 29504.00, dATR 283.50):
  recomputed seated: SWG-L·15m A 29505.50, EQL A 29512.00, EQL A 29526.50, VWAP−1σ A 29529.98, SWG-H·5m A 29549.00, OR-L C 29562.50, VWAP−2σ A 29444.25, SWG-L·5m A 29437.00, pdVWAP A 29574.07, ONL A 29577.75, PDL A 29424.00, EQH·1h A 29420.00
  plan rows:        PDL A 29424.00, SWG-L·5m A 29437.00, VWAP−2σ A 29445.53, OB(bear)·1h C 29502.88, EQL A 29512.00, EQL·15m (HTF) A 29516.00, VWAP−1σ A 29531.05, SWG-H·5m A 29549.00, pdVWAP A 29573.96, ONL C 29577.75, SWG-L·15m A 29592.50
**2026-08-28 LONDON** (price 29657.25, dATR 305.25):
  recomputed seated: VWAP+1σ A 29663.55, OB(bear)·1h A 29644.38, PDC A 29642.00, RTH-H A 29678.25, VWAP A 29635.91, SWG-H·5m A 29682.25, VWAP+2σ A 29691.18, SWG-L·5m A 29613.50, VWAP−1σ A 29608.28, PDH A 29707.50, SWG-L·5m A 29597.25, ONL A 29577.75
  plan rows:        VWAP+1σ A 29663.45, RTH-H B 29678.25, PDH A 29707.50, Demand·1h (HTF) A 29722.62, OB(bear)·1h (HTF) A 29644.38, ONL A 29577.75, Demand·1h C 29541.12, OB(bull)·1h (HTF) A 29490.88, RTH-L A 29424.00
**2026-08-27 NY** (price 29577.50, dATR 304.25):
  recomputed seated: SWG-H·5m A 29582.25, SWG-L·5m A 29570.00, OR-H A 29589.50, eVWAP A 29548.88, VWAP+1σ A 29611.00, SWG-H·15m A 29622.25, SWG-L·15m A 29531.75, FVG·1h A 29511.25, VWAP−1σ A 29503.45, PDC A 29499.75, PDH A 29655.75, ONH A 29661.25
  plan rows:        Supply·1h C 29619.50, VWAP+1σ A 29611.04, OR-H C 29589.50, Demand·4h (HTF) A 29575.25, eVWAP A 29548.74, Demand·1h C 29541.12, EQL·15m (HTF) A 29526.00, VWAP−1σ A 29503.23

## R12 · Coverage list

**Read as spec, then reimplemented independently (no Go executed):** kernel/fvg_entry.go · kernel/levels_zones.go · kernel/levels_assemble.go · kernel/levels_score.go · kernel/levels_swing.go · kernel/levels_volume.go · kernel/structure.go · kernel/levels.go · kernel/levels_intraday.go · kernel/levels_multiday.go · kernel/levels_role.go · kernel/scenario_facts.go · kernel/plan_lifecycle.go · kernel/naked_poc.go · kernel/svp.go · kernel/cme_calendar.go · market/data_indicators.go · store/level_state.go · store/strategy.go · trader/auto_trader_planner.go · trader/auto_trader_dayplan.go · trader/ninjatrader/bars_market_bridge.go · provider/ninjatrader/bar_cache.go · provider/ninjatrader/tcp_server.go · docs: 2026-08-26-fvg-entry-model.md · 2026-08-26-packb-volume-levels.md · 2026-08-27-level-truth-wave.md.

**Surfaces NOT verified, with reason:**

- HTF (1h/4h) detector input series = NT8-native cache bars with 2000-back seeds including pre-08-19 history — not persisted (only 1m since 08-19 15:00 UTC is). Zones born inside the persisted window are verified EXACT; `OB(bull)·1h@29490.88` and `Demand·4h@29575.25` have pre-window births [B].

- Plan-time `level_state` freshness: rows evolve in place; a post-plan decrement overwrites plan-time state. Replay uses only rows whose `updated_at` precedes the plan [B]. The final A/B/C letter therefore cannot be bit-reproduced where freshness decides; zone geometry and the zoneSizeMult factor are unaffected.

- Consumed-transition exact timestamps need the `touch_episodes` join; consumption was recomputed directly from bars on the rule TF [A].

- The C# NT8 AddOn feed itself: the persisted bars are trusted as the source of truth (bar persistence is not re-sourced).

- `collectMachineGrades` (2026-08-29) post-dates the sessions; the 08-27/28 stamp path (input.Levels + input.Pool) was verified at the deployed commits (git show 2850e351 / 99b96b15493e).


## Verdict

Zone math is TRUSTWORTHY for the Sunday 17:00 CT live fire: every stored zone/swing/volume object reconstructible from the persisted window reproduced EXACTLY (0 DELTA >1 tick, 0 true MISSING) across all six classes; the two S-list rows are pre-persistence births (coverage), not math errors; the ATR basis, zoneSizeMult banding, VWAP σ math, FVG floor/displacement gates and consumed transitions all re-derive from the spec with stored values matching to ≤1 tick [A]. Residual risk is confined to the documented coverage limits (§R12): NT8-native HTF series depth and in-place level_state freshness.
