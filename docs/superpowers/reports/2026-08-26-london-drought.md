# LONDON-DROUGHT INVESTIGATION (2026-08-26, READ-ONLY)

**Trigger:** last trade early-LONDON 08-26 (06:12 CT entry, pos 563, closed
06:38 CT). Zero fills 06:38 CT → 22:00 CT: late-LONDON + ALL of NY + ASIA so
far. Investigated 2026-08-26 21:58–22:06 CT, read-only (SQLite `mode=ro`,
journalctl, no mutations).

## 1. The drought window [A]

`trader_positions` (entry_time ms-epoch):

| id | entry (CT) | exit (CT) | session | pnl |
|----|-----------|-----------|---------|-----|
| 562 | 08-26 02:22:06 | 06:01:49 | LONDON | −98.50 |
| 563 | 08-26 06:12:45 | 06:38:04 | (unattributed) | −49.50 |
| 561 | 08-25 19:11:44 | 19:25:39 | ASIA | −52.00 |

Last fill 06:38:04 CT. Prior day (08-25): 12 trades. Today: 2, both LONDON.
Day P&L −148.00.

## 2. What the AI did (decision_records, valid JSON) [A]

Cycles by session window (CT):

| window | wait | open_long | open_short |
|--------|------|-----------|------------|
| LONDON 02:00–08:30 | 55 | 1 | 2 |
| NY 08:30–14:45 | 160 | 1 | 1 |
| ASIA 17:00+ (to 22:00) | 153 | 0 | 0 |

The AI did NOT go passive in NY — it proposed 2 entries. Both were refused at
the risk gate. ASIA: zero proposals (chop, see §5).

## 3. The refusals (risk_check_error) [A]

| time (CT) | action | refusal |
|-----------|--------|---------|
| 01:00:23 | open_short | `stale_reeval_refused: drift_too_big` (\|13.75\| ≥ 4.15 = 0.25×ATR 16.5) |
| 01:25:47 | open_short | `dead_man_watchdog: awaiting reconciliation after link gap` |
| 02:16:39 | open_short | `stale_reeval_refused: sl_breached_in_fresh_bar` (high 29237.50 ≥ sl 292xx) |
| **02:22:07** | open_short | passed → pos 562 |
| **06:12:46** | open_long | passed → pos 563 |
| **09:18:10** | open_short | `stale_reeval_refused: drift_too_big` (\|41.25\| ≥ 11.48 = 0.25×ATR 45.9) |
| **10:55:02** | open_long | `stale_reeval_refused: sl_breached_in_fresh_bar` (low 29231.75 ≤ sl 292xx) |

**Both NY proposals died at `stale_reeval`.** Contributing factor: AI call
latency 2–6 min on 40/422 cycles today (`ai_latency_ms` buckets: 382 <2min,
33 2–4min, 7 4–6min). The entry computed against a 2-minute-old picture fails
the 0.25×ATR14 drift re-check. `stale_dodge=on reeval_drift=0.25×ATR14` per
ledger boot.

## 4. Re-plan exhaustion → terminal NO-TRADE [A]

`plans` (created_at UTC → CT):

- **LONDON v12, 07:38 CT** — `no_trade`, trigger `replans_exhausted`.
  Doc: "FAIL-CLOSED: re-plans exhausted (**10/4**) after 10 deaths — last:
  flip-condition…". LONDON produced 15 versions in 6.5h (10 deaths, cap 4).
- **NY v8, 11:15 CT** — `no_trade`, trigger `replans_exhausted`.
  Doc: "FAIL-CLOSED: re-plans exhausted (**6/4**) after 6 deaths — last:
  flip-condition…". NY produced 8 versions in 2.75h (6 deaths, cap 4).

**The death engine is the core drought mechanism:** plans die on their
flip-condition, every ~30–40 min (LONDON 10×, NY 6×), each death burns a
re-plan, budget (cap 4) exhausted mid-session → terminal NO-TRADE for the
remaining 3h of NY. Wake-triggered reads kept producing new versions
(LONDON v13–15, budget-free per W6) but the terminal marker means the
executor sat out the afternoon anyway.

## 5. ASIA (17:00–22:00): planner healthy, AI declining [A]

- v1 scheduled 17:06 → v2 **planner_fail_closed 18:01:15** (P0 side-quota
  "only 2 levels above price 29614.00" — the known rule, relaxed in the
  18:38 CT deploy) → v3 owner_reset 19:02 (revival, 3 attempts ~23 min) →
  v4 structure_mss 19:52 → v5 level_event 20:13 → v6 scheduled 20:45 → v7
  level_event 21:09. All v3+ active.
- 153 waits, 0 opens, 0 refusals. Scenario_meta: S1–S3 confirms **MET**.
- The AI's own reasoning (verbatim): "S1 sweep-reclaim short already played
  out after the 20:50 sweep… price has since made a fresh higher high
  (29560.25)"; "chopping within 29525–29560, just above VWAP (29541.65)";
  "volume drying up (5m bar volume 17)"; "the plan requires a FRESH
  sweep-reclaim to re-arm". Bias short LOW, death above 29590.75.
- Verdict: this is the AI correctly reading a dead setup in chop — NOT a
  refusal problem. Advisory mode: nothing blocked it.

## 6. Infra noise today (context, not the drought) [A]

- dead-man watchdog NT8 TCP link DOWN → entries BLOCKED 15:22–15:26 CT
  (post-NY), clean reconcile 15:26. Early-LONDON 01:25 CT had one too.
- 🚨 CLOCK EARLY-WARNING 17:06/17:13 CT: |drift| 47.3/55.9s vs NT8
  (tolerance 60s) — "fix WSL2 time-sync NOW, before staleness verdicts
  degrade". Log-only.
- NT8 TCP link DOWN again at the 17:45 deploy boot (playbook wave).
- Deploys today: 17:45 (planner playbook) → 18:38 (side-quota relax) →
  20:05 (bias-tree facts). ASIA v2 fail-closed happened on the pre-relax
  binary.

## 7. Root-cause ranking

1. **Plan churn × replan_cap** — flip-condition deaths (~1/35 min) exhaust
   the 4-read budget mid-session → terminal NO-TRADE → afternoon sits out.
   (LONDON 10/4, NY 6/4.)
2. **stale_reeval kills the entries that do fire** — 2/2 NY proposals and
   2/4 LONDON proposals refused on drift/SL-breach after 2–6 min AI calls.
3. **Chop + disciplined AI (ASIA)** — healthy but correctly declining.
4. Infra noise (clock drift 47–56s, TCP link gaps) — present, not causal.

## 8. Open questions (do NOT act — read-only dispatch)

- Is the flip-condition death threshold too tight for chop? (10 deaths in
  6.5h suggests the flip line crossed on noise.)
- Should replan_cap stay 4 when deaths can come from flip-conditions rather
  than real invalidation?
- stale_reeval drift 0.25×ATR14 vs 2–6 min AI latency: the gate is doing
  its job; the latency is the upstream cause (thinking=enabled,
  reasoning_effort=max).
- Refusal autopsy (would the 09:18 short / 10:55 long have won?) belongs to
  the Sep-3 queued dispatch — NOT here.
