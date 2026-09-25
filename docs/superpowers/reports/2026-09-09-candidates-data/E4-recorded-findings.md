# RECORDED FINDINGS for experiment E4 — not fixed in W3 (owner ruling 2026-09-09)

The owner ruled that W3 changes **selection, ordering and presentation only**. The two findings
below are scoring-side and are the controlled comparison E4 exists to run. W3 must not pre-empt
them. Recorded here with the numbers so E4 starts from measurement, not from a hypothesis.

## F1 — correlated labels inflate the grade: three names at one price = 1.40×

`kernel/levels_score.go:462-476` counts `conf` as the number of **distinct families** within
`confBand` of a level. Same-family duplicates are already excluded (`seenFamilies`), so
`VWAP+1σ` + `VWAP−1σ` correctly count once. **Different families at the same price each add +1.**

The credit is a direct multiplicative factor of the score, on both branches:

```go
// kernel/levels_score.go:500  (zone branch)
evidence, size, confMult, tfMult := zoneEvidence(l), zoneSizeMult(l.Lo, l.Hi, dATR), (1 + 0.20*effConf), zoneTFMult[zoneTierFor(l.TF)]
score = evidence * size * fm * confMult * tfMult
// kernel/levels_score.go:507  (line branch)
evidence, confMult := typeEvidence(l.Kind), (1 + 0.20*effConf)
score = evidence * fm * confMult * htf
```

`ConfluenceCap()` = **3** (`:194-200`, env `CONFLUENCE_CAP`). So a PDC + a VWAP band + an OB at one
price → `conf=2` each → `confMult = 1.40`; at the cap → **1.60**, a **60% score premium for one
price wearing three names.**

The research this wave is built on says exactly this is untested — `report.md:86` **[B]**: *"A prior
high, a swing high, a supply zone, and a profile edge can all describe essentially the same
historical event. Counting four labels can double-count information."* — and `report.md:88`
**[C: proposed test]** proposes the merge-then-compare experiment. `report.md:84` records the only
direct test as **TESTED-CONTRADICTED, narrowly** (Osler's publisher strength labels failed to rank).

**Why W3 does not fix it:** recomputing `conf` over merged candidates changes `confMult` from 1.40
to 1.00 for every clustered level — a **−28.6% score change** — which E7 (empty scoring golden) and
A31 (no score-term change) forbid. **E4 owns it.**

## F2 — the two "cluster tolerance" widths differ by ~6.8×

Both are commented *"cluster tolerance"*. They are not the same thing and are not the same size.

| what | value | site |
|---|---|---|
| window confluence is **COUNTED** over | `confBand := 0.10 * dATR` | `levels_score.go:427`, used at `:468` |
| width clusters are **COLLAPSED** within | `LevelClusterTicks = 12` → **3.00 pt** fixed | `levels_score.go:717`, used at `:570` |

With today's live `dATR` (from the log's `proximity band ±203pt` at `proximity_filter_atr = 1`):
**confBand ≈ ±20.3 pt vs a 3.00 pt collapse — 6.8×.** Across today's reads the band ranged
±15.1 → ±51.0 pt (bands ±151, ±203, ±316, ±510) while the collapse stayed 3.00 pt.

**Consequence:** two levels 20 points apart — plainly distinct references on a 203-point day —
each grant the other a +0.20 credit, yet they never merge. So the inflation in F1 is not limited to
levels at "the same price"; it reaches across a window ~7× wider than the one the code treats as
"the same reference".

**Consequence for W3's merge:** merging at the 3.00 pt collapse width (as D2 specifies) would NOT
have removed this credit even under the rejected reading — the pair 20 pt apart stays unmerged.
Closing F1 properly requires deciding what ONE zone width means, which changes scores. **E4 owns it.**

## What W3 does instead

W3 merges for the card, the model's table, and the entry shortlist it builds — so the owner and the
model both SEE one candidate with three names — while the scored `conf` term is left exactly as it
is and the scoring golden stays byte-identical (E7).

---

## E4-3 · The seat race: daily levels detect, sit in band, and lose every seat (recorded 2026-09-10, W-TF + fix/collapse-keeps-names)

**Recorded, not fixed** — owner ruling 2026-09-10: score and seat-race outcomes are E4's to measure, never a wave's to tune.

**The funnel, measured on live bars at 29143.5 with the ±364pt band, running rev `770e2297`:**

| stage | 1d | 1w |
|---|---|---|
| detected (`DetectHTFLevels`, 500-bar window) | 64 | 24 |
| in band ±364 | 6 | 0 |
| after `collapseLevelClusters` (3.00pt) | 5 | 0 |
| **seated (cap 12)** | **0** | **0** |

Every one of the 12 seats went to 1h (9) or 4h (3). **Best in-band daily score 0.862 · lowest seated score 1.260.** Daily levels sit ~30% below the seat floor.

**The two OB rows, same kind, near-same distance:**

```
OB(bull)·1h  @ 29144.50   1.0 pt from price   score 1.260   grade C   tier 1h
OB(bull)·1d  @ 29126.25  17.2 pt from price   score 0.786   grade C   tier 4h   ← 38% lower
```

The daily OB is classified into the **4h tier** (multiplier 1.3 vs 1h's 1.2; evidence 0.72 vs 0.70) — a *higher* tier than the hourly — and still scores 38% lower. The tier ruling is doing what it was asked to; something else is pulling daily zones down.

**Suspect, UNCONFIRMED: `zoneSizeMult`.** The ladder (`levels_score.go`, ≤0.3×ATR ×1.25 … >2.5×ATR ×0.50) prices a zone's width against one ATR, and a daily order block is structurally wider than an hourly one — so a daily zone may be paying the ×0.50 penalty for being exactly what a daily zone is. This was not confirmed in the probe and is named here as the first thing E4 should check, not as a finding.

**What this means for the map.** The daily family is now detected and classified, and the owner will see it on the map **only when a daily level outscores an hourly one at the seat** — which, on this evidence, is rare at the current ladder. That is a measurement E4 owns. Nothing in W-TF or this fix moved a weight, multiplier, ladder or cap.

**Probe caveat.** The scorer was called directly on raw detections, bypassing `dedupeSameKind`; doubled rows in the raw probe output are an artifact of that, not a live defect.
