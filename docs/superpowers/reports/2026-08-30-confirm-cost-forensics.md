# Confirm-Cost Forensics — did waiting for close-confirms lose the owner money?

**Date:** 2026-08-30 · **Scope:** the full confirm-bearing campaign on our tape (plans 2026-08-15 → 2026-08-28, MNQ futures, SIM)
**Data:** read-only. DB copy `/tmp/nofx-cc-db/data.db` (`.backup` of live); bars/plans/positions/arms/decisions re-read fresh this run (R1). All replay math recomputed from raw 1m bars by my own bucketing — no stored verdicts trusted (R2). Money is `pnl_corrected` for actual fills (R7). All replays on 1m bars; intrabar stop+target ambiguity resolved AGAINST the trade (R9). Every table states its n.

**Verdict (one line):** **confirms NET-COST ≈ $681** over 30 MET entries — mechanical close-drift dominated by `2x5m_close` (−$330) and `1x5m_close` (−$352) — with only one protection case (+$128) and **zero** missed winners on the tape.

---

## Q6 — Storage truth + reconstruction assumptions

### What IS stored
| Fact | Where | Stored how |
|---|---|---|
| Authored confirm rules (`touch` \| `1x5m_close` \| `2x5m_close` \| `15m_close` \| two-leg) | `plans.doc` JSON, `scenarios[].confirm{rule,ref_price,side}` | structured, permanent (159 plan rows → 32 plan_ids) |
| `confirm2` (two-leg) | `plans.doc` | **n = 0 anywhere in the DB** — no two-leg scenario was ever authored |
| Scenario stop/target | **NOT stored in plan docs.** Only `arm{entry,stop,target}` (39 armed scenarios, 18 `wait_confirm` arms across all versions) and `target_chain[]`; executor-side stop/target only in `decision_records.decision_json` of actual `open_*` decisions | structured, per-cycle |
| MET state | **NOT stored as a verdict table.** Recomputed every cycle by `kernel.EvaluateConfirm` and rendered into `decision_records.system_prompt` ("Machine-computed confirmations … `S1 confirm: … MET`") | transient per-cycle snapshot (my replay cross-checked: 25/30 replay-MET moments match a stored MET line within +5 min; the 5 others = no executor cycle ran after MET (2 touch-arms, 3 session-end-edge cases)) |
| Actual fills | `trader_positions` (566 MNQ rows; `pnl_corrected`, `cited_scenario_id`, `plan_id`, `plan_version`) + `armed_orders` (11 rows: 4 TEST-E2 seams + 7 real) | permanent |
| Refusal evidence | `decision_records` (`decision_json` action `wait`/`hold`, `cited_scenario_id`, `cot_trace` reasoning) | permanent |
| Bars | `bars` MNQ 1m only — **10,023 rows, 2026-08-19 10:00 CT → 08-28**, all `open_time_ms % 60000 == 0` | 5m/15m must be rebuilt |

### Reconstruction assumptions (every one stated)
1. **Universe = 71 plan:scenario identities** (dedupe per plan:scenario identity; params from the latest version that defines that scenario with a confirm). Raw all-versions count = 363 confirm-bearing scenario-versions (touch 5 / 1x5m 197 / 2x5m 108 / 15m 53). Plans whose session window precedes the first bar are no-data (n=7).
2. **Trigger moment** = first 1m bar touching the trigger ref after plan write, within the session window (ASIA 17:00→02:00 CT, LONDON 02:00→08:30, NY 08:30→14:45). Trigger ref = `arm.entry` when armed, else the confirm's own `ref_price` (no separate trigger-price field exists).
3. **MET** = my own 5m/15m bucketing from 1m (R2): `1x5m_close` = one 5m close beyond ref, `2x5m_close` = two consecutive, `15m_close` = one 15m close, `touch` = the trigger moment itself. Bucket close counts only if ≥ trigger touch and ≥ plan write, ≤ session end (kernel-consistent "since plan birth" semantics). 7 one-hour bar gaps (16:00–17:00 CT daily) + one weekend gap exist; none intersect session windows.
4. **Entry prices**: no-confirm entry at trigger = ref (touch guaranteed). MET entry = confirming bucket's close (hypothetical), or the actual `trader_positions.entry_price` when the scenario really filled. $ = points × $2/pt (MNQ). Favorable/adverse signed by direction; cost negative.
5. **Stop/target for Q3/Q4** (only non-armed never-met scenarios need them): stop = the price following "close above/below" in the scenario's `invalid` prose, target = `target_chain[0]`. Bracket side validated; `stop == ref` scenarios excluded (n=3) rather than guessed.
6. **Replay horizon** = session end (the plan dies at session flat). Same-bar stop+target → stop first (against the trade).
7. **Refusal** = ≥1 decision cycle after MET whose prompt shows the scenario MET, action `wait`, no `open_*/hold` citing it afterwards, and no position fill lineage.
8. Position **id 569** (08-28 NY S1 arm, `pnl_corrected = +63.0` labeled LONG 29700→29668.5) is internally inconsistent (a long losing 31.5 pts cannot be +$63; the economics match a short). Per R7 I use `pnl_corrected` and flag the anomaly.

---

## Q1 — Census (n = 71 authored identities)

| rule | authored | replayable | MET | NEVER-MET | no-trigger | no-data |
|---|---|---|---|---|---|---|
| touch | 4 | 4 | 2 | 0 | 2 | 0 |
| 1x5m_close | 35 | 31 | 16 | 2 | 13 | 4 |
| 2x5m_close | 29 | 26 | 12 | 2 | 12 | 3 |
| 15m_close | 3 | 3 | 0 | 1 | 2 | 0 |
| two-leg | 0 | 0 | 0 | 0 | 0 | 0 |
| **total** | **71** | **64** | **30** | **5** | **29** | **7** |

**MET-but-entry-refused-later: n = 8** (4×1x5m, 4×2x5m) — the executor saw the MET line, chose `wait` on chase/timing grounds, and no fill ever came:
`1x5m_close`: 08-24 ASIA S1 (6 wait cycles), 08-26 NY S3 (3), 08-26 ASIA S1 (48), 08-26 ASIA S3 (45) · `2x5m_close`: 08-25 LONDON S4 (1), 08-26 ASIA S4 (17), 08-27 NY S2 (16), 08-27 NY S3 (9).
Typical words (08-26 ASIA S1, 04:32 UTC): *"S1 rejection short has already confirmed with a 5m close below 29539.38, but price has already dropped toward 29526, so chasing here gives poor R:R"* — i.e. the refusals ARE the Q2 drift, made visible.

## Q2 — Waiting cost (Δpx trigger-entry → MET-entry × direction; n = 30 MET)

| rule | n | real fills | slip$ (cost < 0) |
|---|---|---|---|
| touch | 2 | 0 | **0.0** |
| 1x5m_close | 16 | 4 | **−351.5** |
| 2x5m_close | 12 | 4 | **−457.6** |
| 15m_close | 0 | 0 | 0.0 |
| **total** | **30** | 8 | **−809.1** |

Top-3 cost cases: 08-20 NY S1 2x5m_close −$147.5 (73.75-pt drift in the wait) · 08-21 NY S2 2x5m_close −$96.5 (real fill) · 08-21 NY S1 1x5m_close −$50.5 (real fill). Purely-mechanical variant (all MET at confirming close, no real fills): 1x5m −$471.5, 2x5m −$456.6 (n=28) → −$928.1. The real-fill sub-table (38 filled entries vs their own cited-version trigger refs): **Σ −$1,365.9**, i.e. the drift is not a model artifact — actual entries paid it.

## Q3 — Missed entirely (NEVER-MET whose first target was reached before the stop): **n = 0 cases, $0**
All 5 NEVER-MET scenarios: 08-20 ASIA S2 (→ Q4), 08-24 ASIA S4 (replay: neither stop nor target touched before session end), and 3 excluded for `stop == ref` under convention 5 (08-21 LONDON S1, 08-25 LONDON S2, 08-26 LONDON S5 15m_close). **On this tape, waiting never cost a full winner.**

## Q4 — Protection (triggered, hit the STOP side without ever confirming): **n = 1 case, +$128**
| date | session | sid | rule | trigger | stop | saved $ |
|---|---|---|---|---|---|---|
| 2026-08-20 | ASIA | S2 | 2x5m_close | 29331.0 | 29395.0 | **+128.0** |
Extension (robustness, not the task's definition): 4 MET scenarios had the no-confirm entry stopped before the confirm even fired — 08-20 LONDON S1 (+$38.5), 08-25 LONDON S4 (+$19.5), 08-26 NY S1 (+$32.0), 08-26 ASIA S3 (+$11.8) → **+$101.8 additional saved**.

## Q5 — NET per rule (NET = Q4$ + Q2$ − Q3$, Q2 signed so cost is negative; n = MET entries)

| rule | n | missed-$ (Q3) | slippage-$ (Q2) | saved-$ (Q4) | **NET** |
|---|---|---|---|---|---|
| touch | 2 | 0 | 0.0 | 0 | **0.0** |
| 1x5m_close | 16 | 0 | −351.5 | 0 | **−351.5** |
| 2x5m_close | 12 | 0 | −457.6 | +128.0 | **−329.6** |
| 15m_close | 0 | 0 | 0.0 | 0 | 0.0 (no MET data) |
| two-leg | 0 | 0 | 0.0 | 0 | 0.0 (never authored) |
| **total** | **30** | **0** | **−809.1** | **+128.0** | **−681.1** |

### ARMS-only twin (wait_confirm arms — fills we got vs fills that never came)
Runtime resting arms (`armed_orders`, 7 real rows; docs: 18 `wait_confirm` arms across all versions, 3 in latest-defining docs):
| arm | side | entry | state | result |
|---|---|---|---|---|
| 08-27 ASIA S1 (touch) | short | 29621.01 | filled | **−$42.0** (stopped) |
| 08-28 LONDON S1 (wait_confirm) | short | 29702 / fill 29642 | filled | **−$32.0** (stopped) |
| 08-28 NY S1 | long | 29464.25 / fill 29700 | filled | **+$63.0** ⚠ side/entry vs pnl anomaly (R7 → +63, flagged) |
| 08-28 NY S2 | long | 29480 / fill 29463.25 | filled | **+$17.0** |
| 08-28 LONDON S4 | short | 29574 | cancelled "no active plan" | replay: would have stopped **−$36** → cancellation **saved $36** |
| 08-28 LONDON S2 | short | 29663.45 | cancelled in NT8 | never triggered → $0 |
| 08-28 NY S3 (wait_confirm) | short | 29586.25 | cancelled EOD flat | never triggered → $0 |
Fills got: 4 → net **+$6.0** (real pnl). Fills that never came: 3 → $0 forgone profit; 2 doc wait_confirm arms (LONDON S3, NY S4) never even rested. **Arms are not where the confirm money leaks.**

---

## Verdict

**Confirms NET-COST ≈ $681** (mechanical Q2 −$809.1, offset by one Q4 protection +$128; Q3 = $0), dominated by `2x5m_close` (−$329.6) and `1x5m_close` (−$351.5). The cost is **entry-price drift during the wait** — not missed winners, not arm failures. `15m_close` (n=0 MET) and two-leg (n=0 authored) **cannot support any verdict** — no data.

## Recommendation (each tied to its table row)

1. **`touch`: extend it.** The only rule with $0 drift by construction (Q5 row 1) and the basis of the working arms (08-27 ASIA fills). For `sweep_reclaim`/`reject` scenarios whose trigger ref is the key level itself, a touch-entry (limit at ref) captures the level without the close-drift tax.
2. **`2x5m_close`: keep only for `acceptance`-class plays.** It costs the most per entry (−$457.6 / 12, Q5 row 3) and produced the single Q4 save (+$128, 08-20 ASIA acceptance S2) plus 2 of the 4 stop-before-MET saves — its protection is real, its drift is heavy.
3. **`1x5m_close`: keep, no change.** −$351.5 / 16 (Q5 row 2) but the drift is the executor's own refusal signal (8 refusals were all chase-aversion, quote above); tightening to `touch` would forfeit the stop-before-MET protection (+$101.8 across 4 cases).
4. **No floor at 2x5m.** The tape shows `2x5m_close` as the most expensive per-entry rule and not the main protector — a blanket "floor at 2x5m" is not supported (Q5 row 3).
5. **`15m_close`: keep on probation.** n=3 authored, 0 MET, 1 never-met (excluded) — no evidence either way; do not expand it.
6. **Watch the drift, not the rule.** The dominating loss mechanism is the confirming close running past the ref (08-20 NY S1: −73.75 pts in one wait). A drift cap ("if the confirming close is > X pts beyond ref, the entry is stale") addresses the cost directly, matching the executor's existing chase-refusals.
