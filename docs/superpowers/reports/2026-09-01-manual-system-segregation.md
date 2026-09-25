# MANUAL vs SYSTEM TRADE SEGREGATION — own-tape money verdicts

Date: 2026-09-01 · Read-only · No writes, no reclassification of stored rows.

## Method (quoted)

Classifier: for every `trader_positions` row, in precedence order —
- **D** if `close_reason = 'e7_farside_test'`.
- **A** if `plan_id` present AND (`source ∈ (system, armed_entry, reconcile)` or `plan_matched=1` or `entry_order_id` ∈ known signal-uuid set). Signal set = `armed_orders.signal_id` ∪ uuids inside `decision_records.decisions`.
- Else, NT8 trace discrimination: the day's `trace.YYYYMMDD.*.txt` files were indexed for `Cbi.Account.CreateOrder` lines (time + `name='…'`) and `Filled`/`Fill` lines. Within ±150s of the row's `entry_time` (CT): **B** if any order/fill name carries a uuid (`name='<uuid>'` or `<uuid>-sl`/`-tp` — our AddOn writes signal uuids as order names), **C** if the window has order(s) but none uuid-named.
- **E** if no trace file exists for the date (June/July — no traces before 2026-08-01).
- **E?** if the day has a trace but no order/fill in the window (resolved per-row with evidence; see row 580).

Trace coverage: `trace.20260801…` → `trace.20260901` complete; **no traces before 2026-08-01** → all May-31→Jul-31 rows are E.

## 1-2. Classification of the whole record (579 rows)

| Class | n | Σ pnl_corrected | Meaning |
|---|---|---|---|
| A system-confirmed | 57 | −108.9 | plan lineage + system source |
| B system-orphaned | 185 | −2774.3 | no stored plan_id, but uuid-named NT8 orders (ours, lineage lost) |
| **C manual** | **0** | — | none — see finding below |
| D test-seam | 3 | +5.0 | e7_farside_test (573/574 +6/−1, 572 NULL) |
| E unclassifiable (no trace) | 334 | −169.5 | May-31→Jul-31 CSV-era |
| E? resolved | 1→B | +39.5 | row 580: trace line quoted below |

**Fraction not system-generated: 0 manual rows, 3 test rows (0.5% of rows, +$5.00).** The owner's one known manual-trading episode (08-25, untracked LONG→SHORT, −$87.50 real) **never entered the ledger at all** — it is not polluting any verdict, it is simply absent.

Row 580 resolution: `2026-08-31 13:35:47 Cbi.Account.OrderUpdateCallback … name='f0bbe9af-c6ce-4444-8243-974c1ce03208-sl' orderState=Filled … averageFillPrice=29417.25` → B.

Weekly table (CT ISO weeks):

```
W22 E:8/0 · W23 E:153/0 · W24 E:91/0 · W26 E:44/−170 · W27 E:38/0      (June–July, no traces)
W31 B:2/0 · W32 B:56/+405 · W33 B:124/−3271                          (Aug 10–16)
W34 A:25/−6 · W35 A:24/−465 B:2/+52 D:3/+5 · W36 A:8/+362 B:1/+40
```

## 3. Verdict re-runs (mega window 08-20 → 08-26 CT)

My window yields n=36 closed positions (mega cited 38 — 2-row boundary difference, noted; mega's figures below are its quoted originals).

| Verdict | Original (quoted) | Recompute now (corrected) | Excl C+D | Survives / reverses / too-few |
|---|---|---|---|---|
| week Σ | −$2,160 / 38 | **−$614.0 / 36** (realized == corrected post-T7) | −614.0 (no C/D in window) | **REVERSES in magnitude** (the −1458 was a lot-math artifact: row 526 realized −1458.0 vs pnl_corrected −69.43, ×21) |
| acceptance | −$1,587 n=5 | **+$222.5 n=3** | +222.5 | **REVERSES SIGN** — the acceptance disaster was the row-526 realized artifact |
| reject | +$625.5 n=14 | **−$28.5 n=16** | −28.5 | **REVERSES SIGN** (neither mega −103 nor +625.5 matches the corrected tape) |
| ASIA | −$1,823 | **−$247.5 n=11** | −247.5 | **REVERSES in magnitude** (same artifact) |
| LONDON | −$328.5 | **−$168.5 n=11** | −168.5 | weakens |
| NY | −$8.5 | **−$8.5 n=10** | −8.5 | **SURVIVES** |
| quality A | −$1,688 n=9 | **−$160.0 n=12** | −160.0 | reverses in magnitude (A-grade "black hole" was the same artifact) |
| quality B | +$265 n=7 | **−$188.5 n=6** | −188.5 | **REVERSES SIGN** |
| quality C | −$242.5 n=5 | **−$76.0 n=14** | −76.0 | weakens |
| killzone in/out | 285 vs 282 fills | counts — calendar classification, no manual rows exist | unchanged | **SURVIVES as counts** (immune unless summed in realized P&L) |
| ARM_MIN_RR | n=18 +$994 | counterfactual gate replay | unaffected | **SURVIVES** (replay, not realized sum) |

Headline: **removing manual trades changes nothing — there are none in the record.** What actually moves every verdict is the realized→`pnl_corrected` correction (T7 + class-27 + 0A-2 backfills): the week's −$2,160 collapses to −$614, acceptance flips to positive, and the "A-quality black hole" disappears. Any Sep-3 verdict built on the old realized figures inherits the artifact.

## 4. Structurally immune analyses (counterfactual replays — unaffected by either exclusion or correction)

- E8 shadow A/B replays (`ab_confirm_log`, counterfactual rows)
- refusal-autopsy per-gate would-have replays (refused arms replayed against the tape)
- ARM_MIN_RR gate costing (n=18 +$994 — replay of refused arms)
- planner A/B harness comparisons (max vs fast prompt)
- MPM resolution fixture (1m-bar stop resolution)
- dryrun-replay scope (never ran — libfaketime FAILED; live-fire is the integration test)
- killzone in/out fill COUNTS (285 vs 282 — count-based)

## 5. Is there a manual/system flag?

**None exists.** `trader_positions.source` enum is `system | reconcile | armed_entry | e7_farside_test` — no `manual` value; manual NT8 activity is only inferable today from trace order names (non-uuid). Where the flag would live: (a) a `source='manual'` value written by the reconcile path when it materializes an NT8-held position with no signal-uuid order in the trace window, or (b) a boot-time/day-roll trace-name scan stamping a `manual` column. Today: inference only, and the inference says zero manual rows were ever recorded.

## STOP-LINES honored
Read-only throughout · no writes · no reclassification persisted (classifications live in this report) · no-trace dates marked E rather than guessed · row 580 resolved by quoted trace evidence, not assumption.
