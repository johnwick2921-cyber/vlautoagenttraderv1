# WHY NO TRADE — every entry opportunity since 09-02 10:37 CT, attributed

**READ-ONLY · no lock · no changes.** Owner: hoang · window: 09-02 10:37 CT → 09-03 ~07:45 CT.
Tree: worktree `~/nofx-notrade`, branch `docs/no-trade-forensic-0903`, base `b5b29ac3`.
CSV: `docs/superpowers/reports/exports/2026-09-02-no-trade/opportunities.csv`.
Sources: `bars` table (read-only), `armed_orders`, `decision_records`, `plans`, journals
`data/nofx_2026-09-02.log` + `nofx_2026-09-03.log` (51,413 lines in window).
Evidence tiers: [A] verified · [B] inferred.

---

## PREMISE CHECK (measured, A-order)

The five changes in boot order, verified against BOOT INTEGRITY lines: EntryGate 18:05:26 ✓ ·
prompt feeds forward 20:42:28 ✓ · no-chase 21:32:52 (WARN-only, `mode=warn counters=on [I]` boot line ✓) ·
void scope narrowed 22:41:58 ✓ · no-trade band 23:24:56 ✓. Standing: 0B stop floor 1.5×ATR5m ✓
(`atr_floor … (1.5×ATR5m …)` lines), marketable guard ✓ (`✕ armed … marketable, never placed`),
death/flip parking ✓ (`😴 … DORMANT`), 30-min wake re-plan ✓ (⏱ lines).

---

## A1 — ARMS (every armed_orders row alive in the window) [A]

Only THREE ledger rows exist in the window:

| arm | plan | side | entry | stop | target | life | final state / reason |
|---|---|---|---|---|---|---|---|
| 32 | NY v5 S3 | long | 29070.00 | 29019.67 | 29171.25 | 10:30:30 → 14:45:01 | cancelled `session ended (EOD flat)` |
| 33 | NY v12 S3 | short | 29166.80 | 29199.50 | 29058.75 | 14:10:29 → 14:10:29 | cancelled `level accepted through — marketable, never placed` |
| 34 | ASIA v13 S1 | short | 29199.50 | 29226.00 | 29132.50 | 22:15:01 → 22:15:01 | cancelled `level accepted through — marketable, never placed` |

- Arm 32: journal shows it armed-not-placed; price never returned to 29070 in NY (session low after
  10:37 stayed ≥ ~29075 until the 00:23 overnight print) → **never_reached**; EOD flat cancelled it. [A]
- Arm 33: `⚔️ armed NY S3 leg 1 short limit 29166.80 SL 29199.50 TP 29058.75` then, same minute,
  `✕ armed S3 cancelled — price 29204.25 already above entry 29166.80 (marketable, never placed)`. [A]
- Arm 34: authored 22:15:01 with `🛑 arm stop ASIA S1 leg 1 short: stop 29226.12 (authored 29226.00 WIDENED) ·
  anchor VWAP+2σ 29208.39 → beyond 29208.89 · atr_floor 29226.12 (1.5×ATR5m 17.75) · bound=atr_floor`;
  cancelled the same minute: `price 29204.25 already above entry 29199.50`. Re-armed 22:26:51, cancelled again. [A]

**Seam REFUSALS (never wrote a ledger row; journal `⚔️ arm REFUSED`, deduped once per spec) [A]:**

| time | spec | class | detail |
|---|---|---|---|
| 13:06:29 | NY S1 leg 1 (v10 long 29163.99) | `rr_refused` | R:R 1.32 after 0B widening (authored stop 29138.75 → chosen 29101.50) |
| 18:47:17 | ASIA S2 leg 1 (v6 short 29209.25) | `rr_refused` | R:R 1.30 (authored 29223.00 → chosen ≈29230.43) |
| 20:07:16 | ASIA S3 leg 1 (v8 short 29175.66) | `rr_refused` | R:R 1.75 (authored 29194.30 widened) |

Plus ~20 `⚔️ arm feasibility … too close (… < 1.5×ATR5m) — min-SL gate will refuse it (WARN)` lines
(13:00-13:02 S1/S3/S4, others through the night) — authored arms whose stops were already sub-floor
before composition. [A]

**EntryGate at the arm seam:** zero `🚦 entry-gate` refusals in the whole window (`grep -c = 0`) —
the seam's legacy chain (R:R/min-SL/shadow/direction/one-live-arm) already covers the same inputs,
so the shared gate never fired there. [A]

## A2 — DECISION PATH (every open since the last fill) [A]

608 executor cycles in the window; **7 cycles carried an open action**:

| rec | time CT | action | cited | outcome |
|---|---|---|---|---|
| 36422 | 09-02 10:37:18 | open_long | S4 | **✓ succeeded — THE LAST FILL (trade 590, −99.00 pnl_corrected)** |
| 36640 | 09-02 18:48:49 | open_short | S2 | ⛔ **entry_gate min_sl**: `stop 29231.00 too close (25.25 < 450.56 = 1.5×ATR5m)` |
| 36641 | 09-02 18:50:36 | open_short | S2 | ⛔ **entry_gate min_sl**: `stop 29226.50 too close (21.00 < 450.56 = 1.5×ATR5m)` |
| 36642 | 09-02 18:52:33 | open_short | S2 | ⛔ **entry_gate min_sl**: `stop 29230.00 too close (22.00 < 450.56 = 1.5×ATR5m)` |
| 36645 | 09-02 19:00:23 | open_short | S2 | ⛔ **entry_gate rr_at_fill**: `R:R 1.42 below floor 2.00 at execution price 29197.0000 (SL 29228 TP 29153)` |
| 36703 | 09-02 20:53:52 | open_short | S2 | ⛔ **entry_gate rr_at_fill**: `R:R 1.73 … at execution price 29159.75 (SL 29193.25 TP 29101.75)` |
| 36864 | 09-03 02:12:56 | open_short | S1 | ⛔ **entry_gate rr_at_fill**: `R:R 1.76 … at execution price 29193.25 (SL 29242 TP 29107.25)` |

The 36422 open at 10:37:18 is trade 590 itself. **After it: 6 opens, 6 EntryGate refusals, 0 fills.**

## A3 — PLANS (versions authored in the window, lifecycle) [A]

NY v6→v12 (level_event wakes, all active), ASIA v1→v15, LONDON v1. Parks:
- **ASIA v8** flip-parked 20:37:16 (`2x5m close above 29175.66 → bias long`) — replaced by v9 at 20:38
  (park lived ~1 min; nothing could fire). [A]
- **ASIA v13** death-parked 22:35:01 (`5m_close above 29218.25`) — replaced by v14 at 00:08 CT.
  Its S1 (reject-short at 29199.50, the arm-34 line) **was touched once at 00:01 CT while parked**
  (1m bar straddled 29199.50; price then ran to 29249.75). [A]
- **ASIA v15** death-parked 00:34 (`5m_close above 29132.50`) — armless (S1/S2 no arms); nothing to fire. [A]
- **LONDON v1** flip-parked 02:40 (`2x5m close above 29207.50 → bias long`) — armless scenarios; no arms. [A]
- Planner preflight failure at 16:36:29 (`⛔ planner preflight refused level_event: stale_bars_2189s`) —
  the post-boot data gap left no market data for the ASIA read. [A]

---

## B — ATTRIBUTION (the single stopping cause per opportunity)

| cause | count | ids |
|---|---|---|
| `entrygate_rr_at_fill` | 3 | 36645 (1.42), 36703 (1.73), 36864 (1.76) |
| `entrygate_min_sl` (dATR defect, see C4) | 3 | 36640, 36641, 36642 |
| `marketable_guard` | 2 | arm 33, arm 34 |
| `rr_refused` (0B widening at the seam) | 3 | NY S1 13:06, ASIA S2 18:47, ASIA S3 20:07 |
| `never_reached` | 1 | arm 32 (EOD flat) |
| `parked` (condition fired while parked) | 1 | ASIA v13 S1 touch @00:01 (parked 22:35-00:08) |
| `planner_failed` | 1 | ASIA 16:36 preflight stale_bars |
| `condition_never_true` | many | armless scenarios across v1-v15 (no arm → no opportunity) |

**7-day baseline (08-26 → 09-02 10:37):** 33 ledger arms → **9 filled (27%), 24 cancelled**; 4 decision
fills (587-590). **Window (≈16 h):** 3 ledger arms → **0 filled**; 6 decision opens → **0 filled**.
The refusal share moved from "the seam refused most arms, the decision path filled sub-floor"
(yesterday morning's 587/589) to "both paths refuse or cancel everything".

---

## C — COUNTERFACTUALS (tape only) [A]

**C1 — the three 0B-widened R:R refusals:**
- NY S1 long (v10) @13:06: entry touched immediately; **authored stop 29138.75 hit at 13:12** → the
  authored-stop version was a stop-out; the widened version (chosen 29101.50) never hit by 15:00 (target
  29246.25 never reached either). Widening turned a loser into a flat-close. **No cost.**
- ASIA S2 short (v6) @18:47: entry touched 18:47; **target 29181.64 hit at 19:07 (+27.6 pt)**; authored
  stop only at 22:18. **The widening cost a target hit** — the authored R:R was 2.01 (would pass at 2.0).
- ASIA S3 short (v8) @20:07: entry touched 20:07; **authored stop 29194.30 hit at 20:16** → a stop-out
  either way (wider stop = bigger loss). **Refusal saved the loss.**

**C2 — the two marketable-guard cancels:**
- arm 33 (NY short) @14:10: a market short at the guard time would have hit **stop 29199.50 at 15:00**
  (target 29058.75 never reached) → **guard saved a stop-out**.
- arm 34 (ASIA short) @22:15: stop **29226.00 hit at 22:18** (3 min later) → **guard saved a stop-out**.

**C3 — conditions fired while parked:**
- ASIA v13 S1 reject-short: 29199.50 touched **00:01** while death-parked (22:35→00:08). One event.
- ASIA v8 (park 1 min), v15, LONDON v1: armless or too short — zero fires.

**C4 — the defect (measured):** the three `min_sl` refusals at 18:48-18:52 used
`1.5×ATR5m = 450.56` → **ATR5m = 300.4** while the arm seam in the same minutes used healthy
`1.5×ATR5m 12.78-14.12`. Cause: class-48's `entryGateForDecision` feeds the min-SL leg
`kernel.PlanDATRFor(at.id)` (`trader/entry_gate.go:267`) — but `SetPlanDATR(traderID, dATR)`
(`kernel/plan_render.go:370`) stores the **DAILY ATR**, not ATR5m. With the real ATR5m (~13.5 →
floor ≈20.2 pt) all three stops (21-25.25 pt) pass min-SL, and R:R at execution ≈2.1-2.3 → pass.
The tape reached all three targets: 29181.64 @19:07, 29162.75 @19:15, 29149 @19:16. **The wiring
defect alone is the difference between 0 fills and ~3 target hits (~+50-60 pt each).**
Secondary (cosmetic): the refusal string doubles the prefix (`entry_gate: entry_gate: …`).

---

## D — VERDICT

The dominant cause of no trade is **the new gates doing exactly the job they were built for**:
all three `rr_at_fill` refusals (R:R 1.42 / 1.73 / 1.76 at execution price) are the precise shape that
filled sub-floor yesterday morning (587 at 1.09, 589 at 1.61) and lost −217.50 — the design refusing
what yesterday's losers were, at a cost of zero target hits on those three shapes. The marketable
guard saved two stop-outs (arm 33 → stop 15:00, arm 34 → stop 22:18), and two of the three 0B-widened
refusals also prevented stop-outs. Against that, ONE real defect with ids: my class-48 min-SL wiring
reads the DAILY ATR (`PlanDATRFor` = `SetPlanDATR(…, dATR)`, `entry_gate.go:267`) where ATR5m is
required — at 18:48-18:52 a 300.4-pt dATR produced a 450.56-pt floor and refused three shorts
(36640/36641/36642) that the real ~13.5 ATR5m admits and whose targets all printed within 28 minutes
(19:07/19:15/19:16). Fix scope: swap that one read for the same 5m-ATR the arm seam uses (or the
decision structure snapshot's ATR), plus drop the double `entry_gate:` prefix. Everything else in the
window is the system flat-out refusing a tape that did come to the levels — the cost is the counts,
and the counts are what this report is.
