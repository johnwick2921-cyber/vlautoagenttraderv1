# W0 (fix/exec-admission-gate): where the dispatch does not match the code, and the defaults taken

**Lane:** Claude-101. **Base:** `855309b7`. **Spec:** CTO dispatch `1790147920736-97212-000001` (W-EXEC-TRUTH §2 W0).

**Evidence:**
- A six-lens read-only map (workflow `wf_ce4a149c-3b8`). Each lens re-read every cited line at the base, and several ran probe tests through `go test -overlay` without writing to the worktree.
- This lane's spot checks.

**Tiers:** [A] = read or run; [B] = inferred; [C] = speculation.

**Status of the defaults:** each item below is a BLOCKED question to the CTO with the fail-closed default this lane takes (R4). Work proceeds on the defaults. If the CTO overrules one, only that item is reworked.

## Confirmed as the dispatch states [A]

These are confirmed at the base: D1, D8, D9, D11 (G1, proven by execution), D12, D13, D26. The following locations moved or differ:

| Dispatch claim | What the code says |
|---|---|
| B3 key lines `tcp_trader.go:509,622,690,774` | The keys moved 6 lines down: `:515, :628, :696, :780`. The cited lines are now the M2 permit lines |
| `sessionRiskGateAt` covers "no-trade band, CME closed, T1" | It does not check CME closed: session windows are clock-of-day only. It already contains per-session max_trades, session-enable, T1 and the breaker |
| `reconcileBeforeOpenNT` flattens a B/C fill not yet in `trader_positions` | It never reads `trader_positions` or either ledger. It flattens only a held LONG from the NT8 snapshot. A held SHORT reads as flat (`ntHeldPosition` requires `positionAmt > 0`, and shorts are signed negative), so A opens on top of it |

## Defects found beyond the dispatch [A]

1. **EntryGate leg 7 is dead.** It compares `p.Side == "long"`, but every writer stores `"LONG"` (`entry_gate.go:425`; `auto_trader_decision.go:320,438`). The same-side guards in `executeOpen*` (`auto_trader_orders.go:512,660`) are dead for the same reason.
2. **Picture's wire path is dead at the base and on the live rev.**
   - The evaluator never sets `row.SignalID` after `PictureHtfClaimSubmission` (`picture_htf_evaluator.go:309-318`).
   - The send stamps against `row.SignalID == ""`, and the stamp is refused (`picture_htf_send.go:94-96`).
   - Each opportunity becomes an ambiguous `place_pending` row and is later clobbered to `expired`.
   - The send tests hand-build `SignalID:"claim"`, a canon-53 gap.
3. **Picture's `refuse()` has no stage precondition.** A later frame in the same hour rewrites `place_pending`/`working`/`filled` rows to `expired`, which is terminal, so a received FILLED frame is then dropped (`store/picture_htf.go:145-152`).
4. **Picture works across symbols.** An MNQ trader claims ES opportunities and sends on MNQ with ES stop and target. The 4h levels snapshot is shared across symbols (`picture_htf_evaluator.go:102-130`; `picture_htf_send.go:66-69`).
5. **CLASS 160 is wider than D13.**
   - An NT8-REJECTED AI entry is also recorded OPEN, with a P0 "Filled".
   - The AI's ~3 s poll correlates on the single shared `lastEntrySignalID`, so it adopts another path's fill.
   - Every AI order row collapses onto one `trader_orders` row, because `exchange_order_id` is `"<nil>"` and `CreateOrder` dedupes on it.
6. **Never-sent stop entries cancel sibling arms.** When `placeOneStopEntry` does not send, the caller still latches `placedThisPass` and `cancelOtherArmsInPlan` cancels the plan's other arms. This happened live: rows 169 and 176 were cancelled `one_live_entry: … placed` 0.3 ms after the "placed" row was itself cancelled "never placed" (`armed_executor.go:1276-1288, :1596-1655`).
7. **A resting order survives a halt.** A breaker trip, a hold, or a daily force-flat that trips while an order rests does not withdraw it, so it can still fill (`armed_executor.go:340-361`). W0 records this; it does not fix it (see Q14).
8. **Entry doors outside A/B/C:**
   - agent chat `OpenLong`/`OpenShort` (`agent/trade.go:193,199`), which reuses the last AI SL/TP maps [B];
   - `DebugPlaceTestTrade`;
   - the test-arm seam.

   Only the TCPTrader latch sees them.

## BLOCKED questions and the defaults taken

**Q1. Gate order.**
- *Mismatch:* the dispatch's order puts maintenance and session risk before the owner pause, and puts feed_down ninth. That breaks the pinned E5 contract ("a paused refusal names the pause"): `gate_order_test.go`, `maintenance_ai_gate_test.go` and `pause_test.go` anchor on source text in `auto_trader_orders.go`.
- *Default:* `admitEntry` runs **A's pinned order**: feed_down → dead_man → frozen → boot_integrity → owner pause → maintenance → roll → breaker/session → last_entry → session_gate → CME-closed (B/C) → plan_mode → approval → EntryGate.
  - The chain moves into `trader/entry_admission.go`.
  - The source-order tests are re-anchored to that file with **unchanged ordering assertions**.
  - The feed gate's close half stays in `executeDecisionWithRecord`.

**Q2. "Missing evidence ⇒ refusal".**
- *Mismatch:* this contradicts the documented fail-open contracts: EntryGate, the breaker on a DB error, feed before its first status, an unresolved roll, the cap count, invalidation.
- *Default:* the shared gates keep their fail-open rulings on every path, so A/B goldens stay byte-identical. Path C gets a **fail-closed pre-check of its own evidence**, refusing when any of these is missing or wrong:
  - entry/stop/target > 0, each on the correct side;
  - ATR5m > 0;
  - positions readable;
  - `min_rr` resolvable;
  - strategy config present.

**Q3. Duplicate steps.**
- *Mismatch:* per-session max_trades already sits inside the session gate, and `DailyForceFlatReason` is EntryGate leg D.
- *Default:* no second copies. Add a pure CME-closed step (`kernel.CMEClosedReason(now)`) for B and C; A keeps its loop gate.

**Q4. Gates the dispatch omits.**
- *Default:*
  - **last-entry cutoff:** all paths. New on B and C, which is a correction: arms can be placed between the cutoff and the EOD flat today.
  - **plan_mode:** A as today; B via EntryGate leg 0; C per Q6.
  - **HTF veto, dead-plan, quality:** stay where they are (A kernel, B authoring). C gets them as a scenario in W5.
  - **approval:** all paths, as the dispatch lists. New on B/C only when `approval_required` is ON.

**Q5. Re-entry cooldown and transition stand-down.**
- *Mismatch:*
  - `discipline.ReentryBlocked` is a destructive read: it deletes the record on unlock.
  - Its inputs (ATR15, price) are kernel-context only.
  - The transition state is runCycle-only.
- *Default:*
  - **Transition stand-down:** AI path only, per its own definition ("refused by the executor gate", `auto_trader_transition.go:19`; knob text `store/strategy.go:768-771`), documented in the guide.
  - **Re-entry cooldown:** its knob text is path-neutral (same symbol, same side, per trader), so it applies to B and C through a new **non-destructive** read with the same unlock math and ATR15 from the 15m bars. A cooldown record present with unreadable ATR15 or price ⇒ refuse. A keeps its kernel call unchanged.

**Q6. Picture under strict.**
- *Mismatch:* EntryGate leg 0 refuses every non-arm path under strict, and the live strategy is strict.
- *Default (fail-closed):* under `plan_mode=strict`, Picture is **refused**. Leg 0 names picture explicitly: "strict executes plan scenarios; Picture becomes one in W5".
  - Under direction/off, Picture runs legs D, 5, 6 and 7 plus the Q2 pre-check.
  - Live Picture exposure stays nil until W5.

**Q7. Picture min_rr.**
- *Default:* the effective floor is `max(picture knob, strategy floor)`. A nil config ⇒ refuse. The guide says so.

**Q8. Picture prerequisites for C's gate-parity and duplicate tests.**
- *Default:* W0 includes three fixes, **all shipped in the same PR as C's admission, never the stamp alone**:
  1. thread the claim id into `row.SignalID`;
  2. add a stage precondition to every transition, so refuse/expire never overwrites `place_pending`/`working`/`filled`;
  3. evaluate and send only the trader's own symbol, with a per-symbol levels key.
- A send-time admission refusal returns a typed provably-unsent error, and the evaluator settles the row `refused`, as it does for a hold. A pre-claim refusal writes no durable row, so a transient gate does not kill the hour's opportunity, and its counter is deduped per oppKey.

**Q9. The latch.**
- *Mismatch:*
  - The book law lives in package trader (an import cycle).
  - The caller's own ledger row would self-block.
  - The B3 slot is consumed before later refusals.
  - The fixtures' clocks are fixed while the entry funcs read the wall clock.
  - A has no ledger.
- *Default:* one mutex per `*TCPServer`, keyed `canonical(account)|wire symbol`. The latch sits after the permit and before B3, and its 60 s stamp is written only after `SendSignal` is attempted. Evidence comes from an injected store-free source (nil = allow), wired in production by a pinned `wireNT8EntryLatch`. Clauses:
  1. the book must be fresh (≤ 2× the snapshot interval), and a stale or absent book ⇒ refuse; a working non-protective entry ⇒ refuse (excluding `-sl`/`-tp`/`-lx`);
  2. a non-flat positions frame on the symbol ⇒ refuse;
  3. an armed row with a signal id in place_pending/working/cancel_pending, a picture row in stage working or with `submitted_at > 0` (excluding the caller's own row, whose identity is passed in), **or TCPTrader.pending** (A's queued entries) ⇒ refuse;
  4. an entry sent on the same account|symbol within 60 s ⇒ refuse.
- A stuck row blocks fail-closed, and the WARN names its id. The clock is injectable and registered in `clock-seams.list`.

**Q10. (c) Which ledger explains a position.**
- *Default:* match on (account, canonical symbol, canonical side). A position is **explained** if any of these holds:
  - a placed, non-terminal armed or picture row exists (any trader id bound to that account);
  - an OPEN `trader_positions` row has an `entry_order_id` equal to a ledger signal id;
  - a ledger row filled within 2× `untrackedGraceMs` has not yet been materialized.
- An explained position ⇒ A is refused, naming the owner. The refusal is a ⛔ gate (sets actionRecord.Error and IncGateBlock, returns nil), not a ❌ error.
- An unexplained position keeps today's flatten.
- Held-short detection and side casing are fixed first (sign-aware, canonical).

**Q11. CLASS 160.**
- *Default:*
  - The AI order row is keyed by the **signal id** the entry returns, so rows stop collapsing.
  - OPEN is recorded only on fill evidence for **that** signal (fill frame, `RecentFillFor`, or an `order_update` filled).
  - A reject for that signal ⇒ REJECTED order row, no position, no P0.
  - A timeout ⇒ the order row stays submitted, a WARN is logged, and the decision record reads "submitted, unconfirmed".
  - A late fill materializes through reconcile's existing untracked path. Lineage stamping by signal id is a stated follow-up.
  - `TestDroppedAIEntryIsForgottenAndTheGateStaysClosed` and the drop handler's text are re-pinned to the new behaviour.

**Q12. Leg 7 and the same-side guards.**
- *Default:* fix them with canonical casing (checklist 28), inside the gate-parity matrix. This changes A/B (a second position is now refused by leg 7); the changed goldens are quoted with the reason.

**Q13. G1.**
- *Default:*
  - `maybeManageArmedOrdersAt` passes an **admitted-this-pass** set into `runArmedPlacementAt`. A row not admitted this pass is not placed; it stays armed and is never cancelled (the M2 law).
  - `admitEntry` also runs at placement, re-resolving the plan from `r.PlanID` and failing closed on a mismatch.
  - `placeOneStopEntry` returns an explicit `sent` bool, and only a real send latches `placedThisPass` and cancels siblings (defect 6).

**Q14. W0(f) hold withdraw.**
- *Mismatch:* no withdraw exists today, and `hold.json` has no withdraw intent.
- *Default:*
  - Build an **entries-only** withdraw primitive: `cancelSafetyFor` → `CancelOrder` by signal → `RequestCancel`. A cancel is complete only on an `order_update` cancelled or the order's absence from a persisted snapshot.
  - Add an additive hold field `withdraw_entries`, written only by `cmd/maintenance-hold --withdraw-entries`.
  - Execute it from `monitorTick`, not the armed pass (which is skipped in exactly the windows an update uses).
  - It covers armed and picture working rows.
  - `GET /api/maintenance` shows `withdraw: {requested, pending[], confirmed[]}`, and a pending cancel is never shown as complete.
  - Protection, reconciliation and exits are never blocked.
  - Withdrawing on a breaker or force-flat trip (defect 7) is **not** built: it is not in the dispatch.

**Q15. Races and clock seams.**
- *Default:*
  - `admitEntry` takes `now` from its caller.
  - Add *At variants for `entryPaused`, `planModeBlocked` and `entryGateForDecision`.
  - Path C never touches runCycle-only state:
    - runCycle publishes the dead-man verdict atomically;
    - the roll check becomes a pure, non-writing predicate;
    - `t1WindowsFor`'s warn fields get a mutex;
    - C's refusal dedupe is C-local.

**Q16. Counters.**
- *Default:* re-admission counts and logs once per change of (path, leg, row|oppKey). EntryGate runs at re-admission with `OnNoChase=nil`, and the invalidation WARN is deduped (canon 35).

**Q17. Other entry doors.**
- *Default:* the latch covers them; `admitEntry` is not wired to them in W0. The W3(f) Sim101 check runs flat, so the latch does not refuse it.

**Q18. Picture 0B floor.**
- *Default:* leg-6 semantics refuse when `dist < 1.5×ATR5m`, with ATR5m > 0 required. `composeArmStop` is not used: it only widens a stop and has arm-only side effects.
- [C] This may refuse a large share of Picture setups; it has not been measured.

**Q19. Picture R:R price.**
- *Default:* leg 5 judges at the worse (lower R:R) of `EntryRef` and the newest 1m close, because the send is a market order.

## Build order on the defaults

1. Stop-entry `sent` bool (defect 6).
2. CLASS 160, keyed by signal (Q11).
3. Held-short and side-casing fixes (Q10, Q12).
4. The latch (Q9).
5. `admitEntry` with the matrix (Q1–Q7, Q15, Q16).
6. G1 (Q13).
7. Picture prerequisites + C admission (Q8, Q18, Q19).
8. (c) (Q10).
9. (f) (Q14).
10. Guide + checklist.

Every item is built RED first at the production call site (canon 53), and the full `go test -race ./...` passes at head before each report.

## CTO rulings (msg `1790150181685-5364-000001`, 2026-09-23 07:56Z)

All nineteen defaults were accepted. The additions:

| Q | Addition |
|---|---|
| Q6 | The strict refusal of Picture is VISIBLE on the 📷 boot line and on the plan card |
| Q8 | `refuse()` may move only `place_pending` / `confirmed` rows |
| Q9 | The exclusion is by the caller's own signal id only. The pinned wiring test and a `latch=wired\|UNWIRED` boot line are required |
| Q11 | The collapsed `trader_orders` rows are separated |
| Q14 | The breaker trip and the daily force-flat trip call the withdraw primitive. If that is more than wiring plus tests, it goes first in W1, said out loud |
| Q17 | The agent-chat `OpenLong` goes through `admitEntry` |
| Defect 4 | Own-symbol only plus a per-symbol cache key, with a test that an ES frame never produces an MNQ claim |

**Split.** Beyond about 3,000 lines, W0 splits into two PRs on the same lineage:

| PR | Scope |
|---|---|
| W0a | the stop-entry sent outcome, CLASS 160, side casing, the latch |
| W0b | `admitEntry` + the matrix, G1, the Picture prerequisites + C, (c), (f) with the trip withdraw |

## W0a as delivered (this PR)

| Item | Commit(s) | What |
|---|---|---|
| Defect 6 | `7850b7d4` | A never-sent stop entry no longer cancels its siblings. The outcome is NOT_SENT / HELD / COMMITTED |
| CLASS 160 (d) | `8edf3d97` | An NT8 AI open becomes a position only on a fill for its own signal. The order row is keyed by signal |
| Canon 28 (Q10 / Q12) | `b4978ced` | One side canonicalizer at six sites. A held SHORT is seen, and leg 7 and the same-side guards are live |
| Latch (b) | `06c18bdc`, `27e01d09` | One entry latch in the four entry functions, with production evidence wired, a pinned wiring test and the 🚦 boot line |

- No existing test or golden moved for leg 7; the full trader tree is green.
- W0b (`admitEntry`, the matrix, G1, the Picture prerequisites + C, (c), (f)) follows on a branch stacked on this one.
