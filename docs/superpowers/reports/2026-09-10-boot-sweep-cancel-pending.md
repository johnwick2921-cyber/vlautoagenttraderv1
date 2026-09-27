# The boot sweep's three-state list is DELIBERATE — and centralizing it opens a hole

**Status:** finding only. No fix written by this lane. **The wave belongs to
`origin/fix/arm-state-predicate`** (claim: `arm-state-0b955fbc/root[unlisted]`,
2026-09-09T22:17:17-05:00).
**Regression is NOT on dev.** `origin/dev` still carries the safe three-state
list at `store/boot_sweep.go:47`. Everything below concerns the UNMERGED branch.

**⚠️ DO NOT MERGE `fix/arm-state-predicate` UNTIL THE SWEEP READS A NAMED
PREDICATE.** Its suite is green and the defect ships green — see the A/B below.

---

## 1. The omission is deliberate, and it is load-bearing

`ListPreBoot` hand-types `state IN ('armed','place_pending','working')` and
excludes `cancel_pending`. That exclusion is not an oversight:

- The sweep's terminal write is `ledger.SetState(r.ID, "cancelled",
  BootSweepReason)` — `trader/class33_boot_sweep.go:94`. `SetState` is a raw
  update: `Updates(map[string]any{"state": state, "state_reason": reason})`.
- `ConfirmCancel` — `store/armed_orders.go:463-465` — is documented as *"the ONLY
  way a row becomes 'cancelled' through the cancel path, and it requires the id
  of the snapshot whose book no longer listed the order. A caller with no
  snapshot cannot call it — which is the point."*
- `cancel_pending` means a cancel was SENT and NOT CONFIRMED. **The order may
  still be live at the broker.**

So sweeping a `cancel_pending` row marks it `cancelled` with **no evidence the
order is gone** — the exact blindness the 2026-09-06 cancel-confirmation wave
closed.

## 2. The path a cancel_pending row takes at boot today

| step | site | includes cancel_pending? |
|---|---|---|
| boot sweep | `store/boot_sweep.go:47` | **no** — three states, hand-typed |
| settlement pass work list | `store/armed_orders.go:484` `ListCancelPending` | **yes** — `state = ?`, **no boot_id filter** |
| consumer | `trader/cancel_confirm.go:353` `confirmPendingCancels` | per cycle |

Outcome per row in the pass: book proves the order gone → `ConfirmCancel` →
`cancelled`; else past the 90s timeout → re-request, up to cap 5; at the cap →
WARN *"NOT re-requesting and NOT promoting to cancelled"*.

**A pre-boot `cancel_pending` row IS owned — by the settlement pass, not the
sweep.** The two sites' lists differ because their INTENTS differ. That is the
whole finding.

## 3. What the branch does, and the A/B that proves it

`fix/arm-state-predicate` rewrites the line to
`.Where(NonTerminalArmStateSQL())`, and `NonTerminalArmStateSQL()` is
`"NOT (" + TerminalArmStateSQL() + ")"` — which **includes `cancel_pending`**.
Its `trader/class33_boot_sweep.go` change touches only leg-4 rendering; the sweep
loop is untouched and `SetState` gained no guard. No test on the branch names
`cancel_pending` and the boot sweep together.

Same probe, both revisions — seed one pre-boot `cancel_pending` row, run
`sweepPreBootArmsWith`:

| revision | swept | final state | `cancel_settled_snapshot_id` |
|---|---|---|---|
| `origin/dev` | **0** | `cancel_pending` | 0 |
| `fix/arm-state-predicate` @45d677f5 | **1** | **`cancelled`** | **0** |

On the branch the row is re-cancelled at the broker and marked `cancelled` with
no settling snapshot: **`ConfirmCancel` bypassed.** [A], probe preserved at
`scratchpad/cancelpending_probe_test.go.keep`.

## 4. The fix the owner ruled (2026-09-10), for the branch owner

- The sweep reads a **NAMED** predicate — `SweepableArmStateSQL()`, non-terminal
  **MINUS** `cancel_pending` — **with a comment stating why**. The sweep's intent
  is a named function, not a list.
- **Pin:** a `cancel_pending` row at boot is **NOT** swept, and **IS** picked up
  by `confirmPendingCancels`.
- Correct `store/boot_sweep.go:38`'s comment to match: it currently says
  "returns ONE trader's **non-terminal** rows", and `isTerminalArmState`
  (`trader/one_contract.go:285`) makes non-terminal FOUR states while the SQL
  lists three. **That disagreement is very likely how this became a "miss".**

## 5. Live record (A21)

- Zero `cancel_pending` rows at the time of writing.
- Only **5 rows in all history** ever requested a cancel — ids **107, 108, 111,
  122, 139** — every one at `cancel_attempts = 1`, against a cap of 5.
- **Zero rows have `cancel_settled_snapshot_id` set.** `ConfirmCancel` has never
  fired in production. The 4 that reached `cancelled` did so by another path,
  `state_reason = 'cancelled in NT8'`.

The path is barely trodden, which is why the regression ships green.

## 6. Two design holes → next wave's C-section (owner-ruled)

1. **The attempt budget survives the restart that invalidates it.**
   `RequestCancel` bumps `cancel_attempts` monotonically
   (`store/armed_orders.go:453-455`) and nothing resets it. A row that exhausted
   its 5 attempts before a restart arrives at boot already capped, hits
   `if r.CancelAttempts >= cap { … continue }` (`trader/cancel_confirm.go:404`),
   and is never re-requested and never promoted — stuck `cancel_pending` forever.
   The restart is exactly the event that changes the facts (new process, new
   broker link, new listener) and is the one event the budget ignores. Fix:
   reset on boot, or count per process.
2. **`ConfirmCancel` has never fired — 0 of 4 cancelled rows went through the
   confirmed path.** The contract says it is the only door; the record says every
   row used a different one.
