# The reaper reads the broker, not silence

**Branch** `fix/reaper-reads-snapshot` · **Status** BUILT AND GREEN, not deployed.
**Checklist class 79 — "silence read as death".**

## THE PIN CANNOT BE BACKTESTED — read this first

`nt8_order_snapshots` begins **2026-09-03 21:48:22 CT**. The only three arms ever
reaped by the old predicate — 29, 30, 31 — were cancelled **2026-09-01 23:21 →
2026-09-02 00:16**, more than a day before the evidence source existed. So the
question "were those three alive at the broker when we cancelled them?" is
**permanently unanswerable**: the frame that would have answered it was not being
emitted yet.

The dispatch's pin — *20 min of order_update silence + a fresh book listing it ⇒
not reaped* — is therefore proven **on a synthetic fixture only**. The real proof
is forward: the first time a quiet resting order survives a reap it would
previously have failed. Verified independently by nofx-47.

## The defect

`reconcileStaleWorking` cancelled any working row with no `order_update` for the
stale window. `order_update` is an **event** frame — a resting limit nobody
touches emits **nothing** — so after N minutes a healthy order is
indistinguishable from a dead one. The reaper turned an absence of evidence into
a **cancel**, which is not a read: it is an execution against a live order.

No threshold fixes this. A longer window delays the wrong answer; it does not
make the evidence exist.

## The fix — three verdicts where there was a boolean

| verdict | when | action |
|---|---|---|
| **ALIVE** | fresh book lists it working | never reaped, whatever the silence |
| **GONE** | fresh book omits it, or shows it terminal | reconcile to the broker's word |
| **UNKNOWN** | no book · book too old · **row has no signal id** | WARN "link stale", **cancel nothing** |

Silence still selects *which rows to ask about*. It never decides.

`reaperUnknown = iota` makes UNKNOWN the **zero value**, so any future path that
forgets to set a verdict does nothing rather than cancelling. The failure mode of
this class is built into the type, not just the logic.

## Review findings (nofx-47) — both real, both fixed

**1. The defect survived one level inside the fix.** `orderMatchesArm` returns
false for an empty `SignalID`, so the loop matched nothing and fell through to
**GONE — which cancels**. But a row with no signal id is exactly one the broker
*cannot be asked about*: there is no name to look up. That is UNKNOWN, and it was
taking the destructive branch — this file's own header sentence, true of this
file. Now checked **first**, because it can only arise when something upstream has
already gone wrong, which is when a cancel is least wanted.

**2. A caveat I would have coded around was wrong.** It was relayed to me that all
1,450 snapshot rows carry a blank symbol and that `orders_json` would need
parsing to filter. The **row-level** column is indeed blank on all 1,450 — but the
orders *inside* carry it properly (`{"order_id":"b083ac38…","symbol":"MNQ",…}`),
which is what `WorkingOrdersFor(symbol)` reads. Verified myself before acting.
Coding around the blank column would have added a parser for a problem that does
not exist. nofx-47 corrected their own relay before I could act on it.

## Two things confirmed, neither a defect

**Broker states are broader than the terminal list.** The live book carries
`Working`(32), `Submitted`, `Initialized`, `CancelSubmitted`, `CancelPending`,
`Accepted`. None is terminal, so all read ALIVE — including an arm **mid-cancel**,
which is the safe direction. Now asserted by test rather than inherited from
"absent from the terminal list": safety by accident is not safety.

**GONE is the hot path, not the rare one.** Only **37 of 1,450** snapshots have
`order_count > 0`; the overwhelmingly common book is empty, and an empty *fresh*
book sends every stale working row to GONE. That is correct — the broker saying
"nothing" is evidence — but it means **the branch that cancels is the one that
runs most**, which is worth knowing when reading the logs.

## Ownership

The owner dispatched this wave to **two lanes**: to me as "next free CODE lane",
and to nofx-47 directly. Their empty claim landed 70 s after mine and sits as my
parent — nothing displaced, and they wrote no code. **nofx-47 has declined it and
reviewed instead**, and asked that this be stated here so they can confirm it to
the owner. Sixth misroute in two days.

`push-empty-at-accept` has now caught **two independent duplications** (this and
the lock wave). It was introduced as a branch-name collision guard; what it
actually does is make an **assignment** visible — a larger property than the
argument that produced it.
