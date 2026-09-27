# NEXT-WAVE BASIS — SETTLEMENT, AND WHAT "FLAT" IS READ FROM

Owner-ruled 2026-09-09 (dispatch 104 close-out): **these five findings are the
next dispatch's SECTION C, verbatim.** They were found by an adversarial
read-only census over every flatten path during the session-risk wave, verified
line by line against the tree, and deliberately left unfixed — each is outside
dispatch 104's footprint (A31).

Measured at `954f11b1` (the running rev) / branch `fix/session-risk-limits`.
Every claim is `[A]` — the exact line was read.

---

## C1 · ConfirmCancel has NEVER fired in production

```
sqlite> SELECT (cancel_settled_snapshot_id>0) AS settled, COUNT(*)
        FROM armed_orders WHERE state='cancelled' GROUP BY 1;
0|69
```

**`0` of `69` cancelled rows carry a `cancel_settled_snapshot_id`.**

`ConfirmCancel` — `store/armed_orders.go:466` — is the ONLY writer that records
evidence. It requires the id of the snapshot whose book no longer listed the
order. Its sole production caller is `trader/cancel_confirm.go:381`.

**The bypass — `trader/armed_executor.go:2198`:**

```go
_ = ledger.SetState(r.ID, "cancelled", reason+" (ack timeout — flatten proceeds)")
```

A row is promoted straight to `cancelled` on a TIMEOUT, skipping the entire
`RequestCancel → cancel_pending → ConfirmCancel` lifecycle that the 2026-09-06
cancel-confirmation wave was built to enforce.

`cancelled` is the word that frees the arm slot (`UpsertArm`,
`store/armed_orders.go`) and the word cutover leg 4 counts — so a row promoted on
a timeout can be replaced while its order still rests at the broker.

**This is built ≠ wired, on the wave that was ABOUT settlement.** The lifecycle
exists, is correct, and is tested; the path that actually retires rows at a close
does not use it. Class 95's shape reaching the settlement machinery itself.

Live instance: row 119, signal `65055e98`, `state_reason` "no active plan (ack
timeout — flatten proceeds)", `cancel_requested_at_ms=0`,
`cancel_settled_snapshot_id=0`.

## C2 · Every flatten reads "no open position" from the LOCAL STORE

Five reads, all local:

| file:line | path |
|---|---|
| `trader/auto_trader_clock.go:80` | skip gate |
| `trader/auto_trader_clock.go:158` | session cutoff |
| `trader/auto_trader_clock.go:533` | **EOD flatten** |
| `trader/auto_trader_clock.go:648` | dormancy |
| `trader/auto_trader_clock.go:721` | **T1 red-news force-flat** |

Every one: `at.store.Position().GetOpenPositions(at.id)`.

This codebase documents that store as lagging the broker in BOTH directions:
`trader/position_desync.go:18` — *"for up to ~80s after a real exit, the store
row is still OPEN"* — and `:78`, the store showing OPEN while the broker reports
FLAT.

**The broker's own answer is already in the process:** `at.liveBook(now)`
(`trader/cancel_confirm.go:236`) and `at.brokerBook()` (`trader/f12_leg4.go:182`)
serve the F12 order snapshot that cutover leg 4 answers from. The cutover gate
deliberately splits leg 1 (sqlite `trader_positions`) from legs 2/3 (broker)
*"so a per-account routing fault cannot hide behind one number"*
(`trader/class33_cutover_gate.go`). **The flatten has only the sqlite leg.**

A flatten that trusts the ledger over the broker is the 2026-09-06 naked-stop
shape arriving from the exit side: our record says flat, the broker says
otherwise, and the close is the one moment where the two must agree.

## C3 · The no-link fallback writes `cancelled` with ZERO broker contact

`trader/armed_executor.go:2095` — `return at.cancelArmedOrders(reason), 0` —
taken exactly when `at.armedTrader()` is nil, i.e. when the NT8 bridge is
absent, which is precisely when resting orders are most likely to outlive us.

`cancelArmedOrders` (`trader/armed_executor.go:2015`) walks `ListNonTerminal`
and calls `SetState(r.ID, "cancelled", reason)` on every row — no wire, no book,
no state test. Its count then feeds the operator-facing flat claim at
`trader/auto_trader_clock.go` (`"🔒 EOD-FLAT: %d armed order(s) cancelled"`).

A flat claim with zero broker contact behind it.

## C4 · "Acked" means "left the non-terminal set" — so a FILL counts as a cancel

`trader/armed_executor.go:2190` — `acked = !at.armedRowStillActive(ledger, r.ID)`
with `armedRowStillActive` at `:2207`.

A FILL arriving during the drain window is applied by `onArmedOrderUpdate`, which
writes state `filled` — terminal, therefore absent from `ListNonTerminal`,
therefore `acked = true`. The row is counted and logged as a successful cancel.

A limit that filled two seconds before the close is reported as an order we
cancelled. The position it opened is caught only by the position re-read that
follows.

## C5 · The flatten is the last unguarded cancel site

`cancelArmedOrdersSyncWith` contains **0** `cancelSafetyFor` calls. Seven
per-arm sites have one: `trader/armed_executor.go:375`, `:479`, `:534`, `:569`,
`:896`, `trader/one_contract.go:258`, `trader/position_desync.go:109`.

**This is the owner's ruling of 2026-09-07 and it stands:** *"a guard that
refuses without a book, applied to the EOD flatten and the news-halt sweep,
leaves arms live and re-opens class 33 — worse than the defect it guards."*

It is recorded because the harm is currently bounded by the FAR side: the
deployed AddOn's `HandleCancelOrder` cancels the resting entry from
`workingEntries` and refuses to read `placedBrackets` at all. **That bound lives
in C#, not in Go**, and the Go guard exists precisely so it does not have to.

---

## What ties C1–C5 together

Four of the five are the same sentence: **a word in our ledger is being written
from something other than the broker's answer.** `cancelled` from a timeout
(C1), from a missing link (C3), from a fill (C4); `flat` from a local table
(C2). C5 is the guard that would have asked the broker, deliberately absent.

The 2026-09-06 wave established the rule — *a cancel is confirmed by the
broker's book, never by the call returning.* These are the five places that
still do not.
