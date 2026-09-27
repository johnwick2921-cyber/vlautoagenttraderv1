# REFUSAL AUTOPSY — 2026-08-27 (pulled forward from Sep 3)

**Window:** 2026-08-26 00:00 CT → 2026-08-27 19:25 CT · **READ-ONLY** · worktree `~/nofx-autopsy` @ dev `c9ac2da3` · branch `docs/refusal-autopsy`.
**Replay:** `scripts/refusal_autopsy.py` (in this branch) — stored 1m bars only (MPM rule), MNQ $2/pt, pnl_corrected semantics (== raw here; the last-7d Δ is proven 0).

---

## 1. CENSUS (every refused/declined entry, one row each)

**Gate refusals — 14 raw rows = 8 logical opportunities** (the 7 `arm_RR` rows are ONE arm refused across 7 cycles; deduped below):

| time (CT) | side | entry | SL | TP | R | gate (verbatim) |
|---|---|---|---|---|---|---|
| 08-26 01:00 | short | 29241.75 | 29263.00 | 29095.50 | 6.88 | `stale_reeval_refused: drift_too_big (\|13.75\| >= 4.15 = 0.25 x ATR 16.59)` |
| 08-26 01:25 | short | 29222.75 | 29234.25 | 29145.50 | 6.72 | `dead_man_watchdog: awaiting reconciliation after link gap` |
| 08-26 02:16 | short | 29230.50 | 29236.50 | 29095.50 | 22.50 | `stale_reeval_refused: sl_breached_in_fresh_bar (high 29237.50 >= sl 29236.50)` |
| 08-26 09:18 | short | 29252.25 | 29334.50 | 29145.50 | 1.30 | `stale_reeval_refused: drift_too_big (\|41.25\| >= 11.48 …)` |
| 08-26 10:55 | long | 29232.00 | 29240.00 | 29415.00 | 22.88 | `stale_reeval_refused: sl_breached_in_fresh_bar (low 29231.75 <= sl 29240.00)` |
| 08-27 02:03 | short | 29501.75 | 29509.00 | 29427.50 | 10.24 | `session_gate: MNQ open_short REFUSED — LONDON first-5m no-trade window` |
| 08-27 07:56 | short | 29537.00 | 29559.25 | 29424.50 | 5.06 | `stale_reeval_refused: drift_too_big (\|5.25\| >= 4.96 …)` |
| 08-27 13:50 | short | 29611.41 | 29638.00 | 29557.15 | 2.04 | `⚔️ arm REFUSED NY S1: R:R 2.04 below min 3.00` (×7 cycles 13:50–14:02) |

Plus **60 `superseded_wait` rows** (AI declines; 30 reconstructable with a ≥2pt-risk implied trade — section 4). No `guardrail_skip` rows exist (guardrails master=OFF, soft-audit).

## 2. REPLAY (1m bars, proposal-time close entry, SL/TP walk after t0)

All 8 logical opportunities replayed (script output verbatim above). Twin-check: both sides' math run through one symmetric function (`replay()`), long/short mirrored.

## 3. TABLE per gate

| gate | n | would-won | would-lost | ambig | Σ$ | verdict |
|---|---|---|---|---|---|---|
| `stale_reeval` | 5 | 0 | 5 | 0 | **−247.5** | **SAVING** |
| `dead_man_watchdog` | 1 | 0 | 1 | 0 | −23.0 | TOO-FEW |
| `session_gate` | 1 | 0 | 1 | 0 | −14.5 | TOO-FEW |
| `arm_RR` (dedup) | 1 | 1 | 0 | 0 | +108.5 | TOO-FEW (raw n=7, Σ +285) |
| **TOTAL** | **8** | **1** | **7** | **0** | **−176.5** | **refusals net-SAVED ≈ $176** |

## 4. HONEST-WAIT AUTOPSY (fresh-MET confirm + AI declined)

Filter: `risk_check_error='superseded_wait'` · system prompt has ≥1 `Sx confirm: … — MET` WITHOUT `(stale` · no CONFLICT trailer · implied trade reconstructable (entry = confirm ref_price · SL = invalid-ref price · TP = target_chain[0]) with side-sanity + ≥2pt risk · deduped by (scenario, side, entry, SL, TP).

**30 reconstructable declines · would-won 22 · would-lost 8 · Σ = +$1,763.0** — the AI's waiting in this window left ~$1.76k of replay-positive trades on the table (biggest: S1 00:10 short +311.2 · S3 09:31 short +266.2 · S2/S3 09:31 +212 each).

**Last night's S2 case (explicit):** 08-27 00:50 CT ASIA, S2 short e=29577.25 sl=29596.05 tp=29539.38 → **would-WON +$75.7** (TP hit). The other overnight S2 reconstructions (01:15, 07:55, 20:21, 21:15, 22:13, 22:50) were degenerate (SL==entry in the invalid prose) and are excluded — the 00:50 case is the one clean S2 decline replay.

## 5. R:R DEEP-DIVE (owner's live question)

All 38 candidates with planned R>0 (refusals + declines). Realized outcomes at acceptance floors:

| accept-if-R≥ | n | won | Σ$ |
|---|---|---|---|
| 2.0 | 24 | 14 | **+1557.9** |
| 2.5 | 20 | 10 | +1033.4 |
| 3.0 | 19 | 9 | +1004.4 |

**Per condition-type at each floor:**

| condition | R≥2.0 | R≥2.5 | R≥3.0 |
|---|---|---|---|
| S1 | n2 / +279 | n1 / −32 | n1 / −32 |
| S2 | n5 / +415 | n3 / +310 | n2 / +281 |
| S3 | n7 / **+767** | n7 / **+767** | n7 / **+767** |
| S4 | n2 / −49 | n2 / −49 | n2 / −49 |
| S5 | n1 / +158 | n1 / +158 | n1 / +158 |
| stale_reeval | n4 / −83 | n4 / −83 | n4 / −83 |
| arm_RR | n1 / +109 | n0 / 0 | n0 / 0 |
| dead_man / session_gate | n2 / −37 | n2 / −37 | n2 / −37 |

**The per-play read:** raising the floor 2.0→3.0 in THIS window would have excluded winners worth $420.2 (S1 00:10 +311.2 @R2.32 · arm NY S1 +108.5 @R2.04) against losers worth $131.0 (S4 ×2 @R5.20/4.19 −49 · S1 11:14 @R3.28 −32 · dead_man −23 · session_gate −14.5). The floor helps only through the S4/dead-man/session rows it wouldn't have caught anyway (their R was HIGH — the losing entries had R 3.28–22.88!). **In this window R was not predictive of the replay outcome; scenario-type was** (S3 7/7 won, S4 0/2).

## 6. CAVEATS (mandatory)

- n per cell: gate table n=1–5 (all except stale_reeval are TOO-FEW); declines n=30; R-floor table n=19–24.
- **No-slippage assumption:** fills at proposal-time 1m close and exact SL/TP prints.
- **1m resolution:** same-bar SL+TP touches can't be ordered; convention = SL-first (worst case). AMBIG count in the final clean set = 0 (earlier degenerate rows were excluded, not resolved).
- **Decline reconstruction is implied, not the AI's plan:** SL comes from the scenario's invalid-prose price and TP from target_chain[0]; the AI never wrote a stop on a declined trade. Several excluded reconstructions had SL==entry (invalid level == confirm level).
- **EOD convention:** 14:45 CT same day (next day if t0 later); 0 EOD rows resulted.
- **One odd proposal kept as-is:** 08-26 10:55 long with SL ABOVE entry (would-be "+16 loss" — a model-proposed inverted bracket).
- arm_RR replay uses the NY v4 arm spec (entry 29611.41) for all 7 cycles; 4 cycles replay WON, 3 EOD — entry-time sensitivity is real (13:58+ entered after the target had already printed).

## 7. VERDICT

Per-gate one-liners:
- **stale_reeval — SAVING.** 5/5 refused trades would have lost (−$247.5). The gate earns its keep.
- **dead_man_watchdog — correct call (TOO-FEW).** Would-have-lost −23.
- **session_gate (LONDON first-5m) — correct call (TOO-FEW).** Would-have-lost −14.5.
- **arm_RR (min 3.0) — TOO-FEW but pointed.** The ONE arm opportunity it refused replayed as +108.5 (and raw n=7 +285). The 3.0 floor on resting arms is the single most fragile knob right now.
- **AI declines (honest-wait) — the biggest leak.** +$1,763 hypothetical across 30 declines; S3-type scenarios were 7/7 replay-winners that the AI repeatedly waited through.

Three knobs the data already justifies touching (report only — owner rules):
1. **arm min-R:R 3.0 → 2.0** — would have armed a trade that replayed +108.5 (n=1, but the whole point of a RESTING arm is the bracket is pre-passed; the 3.0 floor rejected R=2.04).
2. **S3-class acceptance slack** — S3 (sweep_reclaim) is 7/7 replay-positive across every R floor; if a config knob gates it anywhere, this window says don't.
3. **stale_reeval stays at 0.25×ATR** — data says SAVING; no change justified.

Needs more n before touching: anything about dead_man/session_gate (n=1 each), the R floor itself (n=24 candidates total; the 2.0-vs-3.0 delta is driven by 3 rows), and every per-condition cell except S3 (n=7).
