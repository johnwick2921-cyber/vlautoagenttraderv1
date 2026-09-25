# Level Grading — Full Audit & Verification (2026-08-24)

Owner mandate: verify ALL levels, confirm every TF requirement is followed,
confirm grading math is correct, full trace, external research (30-min hard
rule), careful detail.

Method: full code trace (Explore agent, file:line verified) + 3 external
research passes (zone grading, detection parameters, confluence/grade bands)
against published trading sources + live-data verification of the current ASIA
plan.

---

## 1. Pipeline trace (verified file:line)

```
detectors (closed bars only)
  ExtractMultiDayLevels   kernel/levels_multiday.go:41
  RoundNumberLevels       kernel/levels_intraday.go:21
  OpeningRangeLevels      kernel/levels_intraday.go:124
  GapLevels               kernel/levels_intraday.go:57
  EqualHighsLows          kernel/levels_zones.go:31
  SupplyDemandZones       kernel/levels_zones.go:100
  FairValueGaps           kernel/levels_zones.go:159
  OrderBlocks             kernel/levels_zones.go:199
  DetectHTFLevels (≥15m)  kernel/levels_assemble.go:82   → HTF=true, TF set
       ↓
AssembleScoredLevels      kernel/levels_assemble.go:35
       ↓
ScoreLevels               kernel/levels_score.go:182
  proximity band ±proximityK×dATR · confBand 0.10×dATR
  zone:    zoneEvidence(TF-kind) × freshMult × (1+0.20·conf) × zoneTFMult[tier]
  non-zone: typeEvidence × freshMult × (1+0.20·conf) × 1.2(HTF)
  gradeFromScore: A≥1.0, B≥0.70, else C
  v3 floors/caps per TF tier (verified by exhaustive matrix, see §2)
  collapseLevelClusters (3.00) → seatHTF (≤2) → seatBothSides (3/side) → top-N
       ↓
planner ranked table (kernel/planner_prompt.go:91) + HTF zones section (:128)
       ↓
model plan → CollapsePlanLevels (3.00) → ValidatePlanDocWithFacts
       ↓
machine-grade stamp by rounded price (trader/auto_trader_planner.go:587-597)
       ↓
card display (api/handler_plan.go:359) + executor RenderPlanBlock
```

## 2. Grading math — VERIFIED CORRECT

Exhaustive matrix over {OB,FVG} × {1m,15m,1h,4h} × {cont,rev} × conf∈{0..5}
recomputed the grades and matched the live scorer byte-for-byte:

- **1m zones: never above C** (noise floor — externally supported: "1m noise",
  "sub-5m FVGs noisy", Seiden M30+). ✓
- **15m/1h zones: floor B, cap B** — as owner-approved in v3. ✓
- **4h zones: floor B, may reach A** with confluence or reversal. ✓
- `zoneTierFor` mapping: `""/1m/3m/5m→1m`, `30m→15m`, `2h→1h`, `6h/8h/12h→4h`. ✓
- Reversal (RBD/DBR) > continuation (RBR/DBD) via ×1.1 — externally supported. ✓
- Reversal classification from the leg BEFORE the base vs the departure sign. ✓
- Cluster tolerance 3.00 (12 ticks) — defensible per external sources. ✓
- Freshness decay 1.0/0.8/0.6/0.5 — near the one published quantified sample
  (0.78/0.55 relative quality for 2nd/3rd OB touch). ✓
- Departure ≥1.5×ATR = exactly the published SMC floor. Base ≤6 candles =
  published upper bound. ✓

## 3. Bugs found AND FIXED this session

1. **`zoneTierFor` unknown-TF zero-score** (kernel/levels_score.go) — the
   comment promised "unknown → 1m noise floor" but the code returned the
   unknown TF as-is; `zoneTFMult` missed → **zone score = 0** silently.
   FIXED: unknown TFs now fall back to `"1m"`. Test: `TestZoneTierForUnknownFallsBackTo1m`.
2. **Machine-grade stamp collision** (trader/auto_trader_planner.go) — the
   stamp map keyed by rounded price let a later detector entry overwrite an
   earlier owner "A". FIXED: keep the STRONGER grade via `kernel.GradeRank`.
   Test: `TestGradeRank`.
3. **Stale freshness doc** (kernel/level_state_provider.go) claimed
   "done → dropped (freshMult 0)"; the real P1c behavior is role-flip at 0.5.
   Doc corrected to match code.
4. **Unconditional HTF-zone mandate** (kernel/planner_prompt.go) — the prompt
   ordered "MUST include at least ONE HTF zone row" even when zero zones were
   detected (impossible rule → retry burn). FIXED: the mandate is now emitted
   only when the HTF zones section is actually rendered.

## 4. Findings REPORTED, NOT changed (owner decisions — "no hardcode" rule)

1. **TFmult double-counts the TF tier.** `zoneEvidenceByKind` already tiers by
   TF, then `zoneTFMult` multiplies again → effective 4h:1m ≈ 2.3×, not 1.3×.
   Decide intended: remove one layer or document ≈2.3×.
2. **15m/1h A→B cap stricter than consensus** — external sources treat 15m as
   the standard setup TF and 1h as medium weight, both A-capable with
   confluence. Cap is owner-approved v3; flagged, not changed.
3. **Confluence bonus unbounded** — research: diminishing returns after ~3
   levels; linear +20% stacking beyond is grade inflation. Recommend cap conf≤3
   (×1.6 max).
4. **FVG needs 1×ATR gap** — no external support; kills the published 20–80 pt
   NQ sweet spot. Recommend any-gap + 2–5 pt noise floor + size weighting.
5. **OrderBlocks unbounded lookback** — a displacement can pair with an
   opposing candle hours back (stale OB). Recommend a bounded scan.
6. **seatBothSides no-ops when nothing is cut** — executor KEY LEVELS block can
   be one-sided (write-time validation protects the PLAN, not the executor
   table).
7. **minGrade planner-only** — `FilterLevelsByMinGrade` applies to the planner
   table but not the executor block / PLAN STATUS / level-state writers /
   fail-closed maps.
8. **Prompt quality string** says `A+|A|B` while the validator (correctly)
   accepts `C` for G5 machine-demoted scenarios — prompt doesn't tell the
   model C exists.

## 5. Live verification (ASIA v3, written 18:02:25 CT with the collapse fix)

- 9 levels, all distinct (no duplicates); machine grades stamped `m:A` on 8 of 9.
- The one unstamped row — `Supply·4h` — is live evidence of finding 4.1: the
  model took it from the **HTF zones section**, which is never merged into the
  `machineGrades` map, so the write site had no grade to stamp. Display-only
  gap today; fix listed in §4.
- Distances sane after the zero-reference fix (rev a5f42da3).
- Floors/caps observed live: `Supply·4h B`, `OB(bull)·4h A` (4h A-capable ✓).

## 6. External citations (key)

- Zone quality: forextradelab.com/blog/supply-demand-zones-forex-trading-guide/,
  justmarkets.com/trading-articles/learning/4-models-of-supply-and-demand
- OB grading: ictkillzone.com/ict-order-block, liquidityscan.io/blog/unmitigated-vs-mitigated-order-blocks
- FVG: ictkillzone.com/ict-fair-value-gap
- MTF: brokeranalysis.com/blog/multi-timeframe-analysis-complete-guide
- Confluence: snappchart.app/blog/technical-indicators/confluence-in-trading
- EQH/EQL: ictkillzone.com/ict-equal-highs-lows
- Displacement: quantum-algo.com/glossary/displacement
- Freshness: priceactionninja.com/sam-seiden-supply-and-demand
- Academic anchor: Osler 2000/2003 (FRB New York) — S/R levels predict
  intraday turns; round-number clustering is real order-flow structure.
