# COMBINED WAVE — WINRATE FIXES (Pack A) + VOLUME LEVELS (Pack B, NO-GO)
# Implementation report 2026-08-26 · branch feat/winrate-volume · PR #76

---

## PHASE 0 — VOLUME FEASIBILITY: **NO-GO** (Pack B skipped; see prior report)

Full evidence in `docs/superpowers/reports/2026-08-26-volume-wave-phase0-stop.md`
(fresh this session): no stored 1m bars exist anywhere (no DB tables, no files,
REST klines = CoinAnk-only), so the 0.2/0.3 validation replay could not run.
Live volume IS real (TCP frame `V` = NT8 tick-volume, `VLTraderTCPClient.cs`
`bars.GetVolume(i)`), so detectors would work live — but the dispatch's gate
kills Pack B without the validation. **Pack B = skipped, plainly.** The feed fix
(bar-history persistence) unlocks Pack B + the swing-k/MSS calibration queue.

## PACK A — SHIPPED (one branch, one cutover)

### Commit ledger (each with citation in-code)

| Commit | Change | Evidence citation |
|---|---|---|
| `0428ca19` | **A1** ONH/ONL liquidity-breakout framing (planner `## Anchor roles` block); fade only on confirmed sweep-reclaim | research 94.2% broken (2,827-day NQ) + week ONH 14 · 21.4% · −131 |
| `acc9961c` | **A2** condition×session guidance in the contract Rules | reject NY 75%/+665 · acceptance 0%/−157 · sweep_reclaim 0%/−192 |
| `e651c6c7` | **A3** MIN-SL gate after min-conf: `MIN_SL_ATR_MULT` (env, default 1.0, 0=off) × ATR(14,5m Wilder from `ctx.Structure["5m"].Atr` — new `StructureState.Atr`) + 2-tick clearance past the cited anchor (`MinSLAnchorFor`) | 15/27 losers MFE≥0.5×SL; 5/5 biggest stops 5–44 pts off-map |
| `deploy/RELEASE` | marker `e651c6c7` | — |

### Cutover (flat: 0 open positions)

```
🔐 BOOT INTEGRITY OK — rev e651c6c763a4 +dirty · built 2026-08-26T06:17:34Z · expected e651c6c7 · goldens PASS
🛑 min-sl guard: atr_mult=1.0 level_clearance=2tick(s)
```
PID 2027337.

### Regression
`go test ./...` EXIT=0 · FE vitest green · goldens: NONE changed (all additions
asserted in planner_prompt_test + new min_sl_gate_test — no golden touched).

## E1–E3 PROOFS

- **E1** fresh read fired post-deploy (owner reset accepted at v11 baseline);
  plan + prompt truth is unit-verified (A1/A2 wording asserts) — the next plan
  rows are written under the new contract.
- **E2 A3 replay** (honest limits): SL distance |entry−exit| known for all 27
  losers; 5m ATR was NOT stored historically (dATR is in-memory only; decision
  prompts don't carry it) → exact refused/widened counts can't be computed for
  the old week. Behavioral split: 15/27 had MFE ≥ 0.5×SL (the too-tight class
  the gate targets); SL-distance histogram: **4 losers < 20 pts** (incl. 3.8 and
  11.2 — would refuse under any plausible 5m ATR), 7 at 20–30, 13 at 30–45,
  3 > 45. Winners' cost: UNKNOWN — SL distance is not persisted for winners
  (entries+exits only) → flagged as the data gap. **Forward fix is already in
  this build:** `StructureState.Atr` is now serialized into `structure_json`
  every cycle, so from this deploy every refusal/entry is replayable.
- **E3** Pack-B dynamic-VWAP proof N/A (Pack B skipped). Live `sl_too_tight`
  refusal: armed and logging `🛑 MIN-SL REJECT` — fires on the first tight-stop
  AI decision; quote when observed.

## OWNER QUEUE

1. **Bar-history persistence** (small additive writer on the TCP bar path) —
   unblocks Pack B validation + swing-k/MSS-FVG/trail-mult calibration. Highest
   ROI data fix.
2. Proximity 1.5→1.0 and seats 12→8 rulings: held for the volume wave (B2) —
   re-rule when Pack B re-dispatches.
3. min-conf 65 revisit (week evidence favors it; early week mixed).
4. Merge PR #76 → dev; partner catch-up rides on top (partner tree == nofx tree
   → format-patch clean).
