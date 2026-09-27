# Level-event wakes: re-planning a session past its useful window (read-only audit)

**Owner request:** hoang, 2026-09-02. **READ-ONLY — no code changed, nothing deployed.**
Live rev `0465a10b`. Evidence: `data/nofx_2026-08-*.log`, `data/nofx_2026-09-0*.log`, `data/data.db`.
**Tiers:** [A] verified directly · [B] inferred from strong evidence · [C] speculation.

## Resolved values this audit is measured against (A11)

No strategy row sets these, so all three resolve to their defaults [A]:

| Knob | Resolved | Source |
|---|---|---|
| `last_entry_offset_min` | **15** | `store/strategy.go:1014` `DefaultLastEntryOffsetMin` |
| `eod_flat_offset_min` | **0** | `store/strategy.go:1015` |
| `wake_min_interval_min` | **30** | `store/strategy.go:1443` `DefaultWakeMinIntervalMin` |

Session windows (`kernel/session_registry.go:85-115`): ASIA 17:00→02:00 · LONDON 02:00→08:30 ·
NY 08:30→14:45, and `WindowEndCT == FlatCT` by owner contract. So the **last-entry cutoff** is
flat − 15: ASIA 01:45 · LONDON 08:15 · NY 14:30.

## (1) Every level_event re-plan in 7 days — n=52

Minutes-to-flat is measured at PLAN-WRITE time (the journal has no separate trigger stamp; the
call that produced it started p50 377 s earlier, so trigger time is ~6 min earlier still).

| Bucket (min to session flat) | n |
|---|---|
| > 60 | 37 |
| 30 – 60 | 7 |
| 15 – 30 | 3 |
| **< 15 (past the last-entry cutoff)** | **5** |

**The 8 written within 30 minutes of flat** [A][A21]:

| Written (CT) | Session | v | min→flat | past last-entry? | arms on that version | ever placed |
|---|---|---|---|---|---|---|
| 2026-08-26 08:30:48 | LONDON | 15 | 0 | yes (+15) | — | no |
| 2026-08-27 08:06:27 | LONDON | 7 | 24 | no | — | no |
| 2026-08-28 01:43:10 | ASIA | 13 | 17 | no | — | no |
| 2026-08-28 08:16:22 | LONDON | 6 | 14 | yes (+1) | cancelled, filled | **YES** |
| 2026-08-31 14:34:06 | NY | 7 | 11 | yes (+4) | cancelled | no |
| 2026-09-02 01:51:21 | ASIA | 10 | 9 | yes (+6) | — | no |
| 2026-09-02 08:01:50 | LONDON | 5 | 29 | no | — | no |
| 2026-09-02 08:20:47 | LONDON | 6 | 10 | yes (+5) | — | no |

**Across all 52: 7 versions carry an arm that ever reached working or filled.** The other 45
produced no order. Of the 8 late ones, exactly **one** (2026-08-28 LONDON v6) shows a filled arm.

**Caveat, stated plainly [B]:** `UpsertArm` reuses the row for `(plan_id, scenario, leg_index)`
and overwrites its `version` on re-authorization, so an arm's `version` is the LATEST version
that touched it, not the version it was armed under. The single "YES" at 14 minutes to flat is
therefore more likely a row carried from an earlier LONDON version than an order placed inside
the last-entry window. Proving which needs `armed_orders` history the ledger does not keep.
**Do not use that one row to argue late re-plans produce trades.**

**Cost.** The 8 late re-plans consumed roughly 50 minutes of max-reasoning planner time
(8 × p50 377 s). Today alone, LONDON v3-v6 after NY had taken over: 369 + 588 + 644 + 497 s =
**35 minutes** of planner time re-authoring a session that had stopped trading.

## (2) Is there a no-reauthor-after cutoff? NO — and the entry cutoff it might be confused with is live

| Path | Location | Status |
|---|---|---|
| `lastEntryCT()` — legacy DAY-scoped | `trader/auto_trader_clock.go:320-326` | **UNREACHABLE.** Its own comment says so: "Kept only so an old `dp.LastEntryCT` config value is visible to a reader; nothing evaluates it." |
| `eodFlatCT()` — legacy DAY-scoped | `trader/auto_trader_clock.go:356-363` | **UNREACHABLE**, same note |
| `SessionDef.FlatCT` | `kernel/session_registry.go:48-53` | Field's own audit note: "no production path consumes this field or `EffectiveFlatCT`" |
| `LastEntryCT` config field | `store/strategy.go:944-946` | persisted, read only by the unreachable accessor above |
| **`sessionCutoffCT` + `entryBlockedByLastEntry`** | `trader/auto_trader_clock.go:328-338`, `:380-400` | **LIVE.** Session end − `last_entry_offset_min`, DST-correct, half-day aware |

So the audit's finding is confirmed: the legacy fields persist and are unreachable [A]. But a
**session-scoped last-entry cutoff does exist and is enforced** — on **ENTRIES**, at
`trader/auto_trader_orders.go:267`.

**Nothing gates RE-AUTHORING.** The wake path (`trader/auto_trader_wake_levels.go:255-283`) is
limited by exactly two things and neither is time-of-session:

1. a dedupe key (`at.lastLevelWakeKey == ev.key` → return), and
2. the 30-minute `wake_min_interval_min` throttle shared with the MSS wake.

The comment at `:266-270` is explicit that wakes are "UNLIMITED and count against NOTHING". A
wake fired at 08:14 on LONDON is therefore legal, will spend ~6-11 minutes of max reasoning, and
lands a plan whose arms the live entry gate will refuse.

## (3) Overlapping planner streams — 3 material, not 57

Reconstructed 255 planner calls from `🧠 planner call … completed in Ns` (start = end − duration).
A naive scan finds 57 "overlaps", but 54 of them are 0-1 s adjacency: attempt N ending as attempt
N+1 opens **inside one read**. Filtering to ≥30 s of genuine concurrency [A]:

| Second stream opened | A (still running) | B | Concurrent |
|---|---|---|---|
| 2026-08-31 08:25:51 | 08:21:50→08:27:12 (321 s) | 08:25:51→08:34:13 (502 s) | **81 s** |
| 2026-08-31 08:27:12 | 08:25:51→08:34:13 (502 s) | 08:27:12→08:33:54 (402 s) | **402 s** |
| 2026-09-02 08:01:06 | LONDON v5 07:51:06→08:01:50 (644 s) | NY v1 08:01:06→08:09:44 (518 s) | **44 s** |

Both dated clusters sit at a **session handover** (08:25-08:34 and 08:01-08:09 are the
LONDON→NY boundary). The mechanism is the claim key: `MakePlanIDForTrader` is
`tradeDate + ":" + session + ":" + traderID` (`store/plan.go:89-91`), so a LONDON claim and an NY
claim are different keys and cannot see each other. `PlannerReadInFlight` is per-(date, session)
by construction. [A]

This matters beyond cost: class 41 could not exclude concurrency as a cause of mid-stream
transport cuts (4 of 81 stream calls on 09-01), and its own fix deliberately serialized the
shadow A/B for that reason.

## (4) Proposal — WARN-first, no changes made

**P1 — no level_event re-plan within N minutes of the session's flat.**

N derived, not chosen:

| Term | Value | Source |
|---|---|---|
| Last-entry cutoff already enforced | 15 min before flat | resolved `last_entry_offset_min` |
| Planner call duration, p90 | 9.3 min (555 s) | n=255 calls |
| Planner call duration, p50 | 6.3 min (377 s) | same |
| Arm lead (plan write → arm working), median | ~20 min | n=6, see caveat |

A re-plan that lands after flat−15 can place nothing, so the floor is
`15 + p90 call ≈ 25 minutes` measured **at trigger time**. **Proposed N = 25.** A stricter
N = 30 would also cover the median arm lead and would coincide with the existing
`wake_min_interval_min`, but the arm-lead sample is n=6 and version-contaminated, so 25 is what
the evidence carries. Recommend it as a resolved knob (`wake_cutoff_min`), not a literal.

**WARN-first:** log `🗓️ level wake … SKIPPED-CANDIDATE: N min to <session> flat` and count it for
one week without suppressing anything. Promote to a real skip only if the counted candidates
show the same near-zero placement rate as this audit's 8 (1 of 8, and that one contaminated).

**P2 — no wake read while another planner stream is open.**

The claim is per-(date, session); a process-wide "a planner stream is open" flag would catch the
handover overlaps that the per-session claim cannot see. WARN-first: log
`🗓️ level wake … DEFERRED-CANDIDATE: <session> stream open since HH:MM:SS` and count. Note this
would NOT have blocked today's NY read, which is a scheduled session read, not a wake — and a
scheduled read must never be deferred behind a stale session's re-plan. **The rule must apply to
WAKES only, with scheduled reads taking precedence.** Stated because the naive form of P2 would
have delayed NY v1 behind LONDON v5, which is the wrong outcome.

**Not proposed:** any change to the dedupe key, the 30-minute throttle, the entry cutoff, or the
"wakes cost no budget" ruling. Those are all working as designed.

## What this audit did not establish (A15, A20)

- Whether late re-plans ever placed an order **cannot be settled** from `armed_orders` as it
  stands, because the row's `version` is overwritten on re-authorization. A `version_armed_at`
  column, or an armed_orders history, would settle it.
- Trigger time is inferred as write-time minus call duration; the journal carries no trigger
  stamp for the wake itself.
- No claim is made that concurrency CAUSES transport cuts — 3 material overlaps and 4 cuts is far
  too small a sample to test that, and class 41's own conclusion was "no power, no effect shown".
