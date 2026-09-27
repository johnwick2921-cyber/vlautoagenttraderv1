# CANCEL-CONFIRMATION — a send is not a settlement

**Branch:** `fix/cancel-confirmation` · **session:** `cancel-confirm-554049f5` ·
**base:** `2267ee70` (dev tip at accept) · **date:** 2026-09-06

**Running rev, measured not quoted:** `/api/health` → `f516da7cadb3`;
`go version -m /proc/1963305/exe` → `vcs.revision=f516da7cadb3bd54339c70e021c98459dbc07ec0`,
`vcs.modified=false`, built 2026-09-06T04:42:14Z.

> **THIS WAVE IS THE PRECONDITION FOR `STOP_ENTRY_SEAM=on`. THE SEAM STAYS OFF
> UNTIL THE OWNER RULES ON IT AFTER LIVE PROOF.** Nothing here turns it on and
> nothing here should be read as recommending that it be turned on today.

**Spec freshness (A1):** the vet-05 and vet-08 reports are last-written at
`a45cf551`; `SYSTEM-MAP.md` at `f516da7c`; `AUDIT-CHECKLIST.md` at `e14dd8da` —
all ancestors of my base `2267ee70`. Nothing moved under me.

---

## 1 · THE PREMISES — AND THE ONE THAT IS WRONG

### C1 — a cancel is not confirmed · **CONFIRMED, and weaker than stated** [A]

`nt.CancelOrder` (`trader/ninjatrader/tcp_trader.go:558-562`) is a one-line
pass-through to `SendCancelOrder`, which returns non-nil in exactly two cases:
no NT client connected, or `WriteFrame` failed on the socket. **Everything past
the socket returns nil** — whether the AddOn found the order, whether
`Account.Cancel` threw, whether NT8 accepted it.

Five production sites read that nil as proof and wrote the ledger terminal on it
(`armed_executor.go` :342, :447, :490, :517, :837 — :447 does it in three lines).
The sync helper wrote `cancelled` even on **ACK TIMEOUT**, with the reason
`"(ack timeout — flatten proceeds)"` (:1933). That is UNKNOWN taking the
destructive branch, and here `cancelled` *is* the destructive branch, because it
is the word that unlocks a replacement.

**The measured cost, which is the number that matters:** across the ledger's
whole lifetime, 67 rows, 47 of which reached the broker — **exactly THREE cancels
were ever confirmed by an `order_update` from NT8: ids 8, 37, 102.** The other 34
are our own assertions (12 old-reaper stale-window, 8 "gate changed", 8
boot_sweep, 5 test-seam, 1 e7_incident_kill). **`cancelled` has been ~92%
unverified for the life of the table.**

### C2 — "it has already happened" · **CORRECTED, and this changes the wave** [A]

The numbers are right and the *interpretation* is wrong.

Snapshot 1664 is exactly as described: 9 orders, all `type: stop`, 8 `Accepted`
+ 1 `Initialized`, all `limit_price 29590.5` with **no `stop_price` key at all**,
`working_count` 9. All nine signal ids read `cancelled` in `armed_orders` today,
and all nine are one slot.

**But they were not nine failed cancels.** NT8 honoured every cancel in that
slot: 22 orders placed on 2026-09-04, 22 cancels sent, 22 broker `Cancelled`
states within ~110-260 ms, **zero rejections** in a 16,936-line log. The longest
any signal outlived its ledger row's `cancelled` is **one second** (ids 87, 89);
20 of 22 are 0 s.

**What 1664 actually shows is nine concurrent PLACEMENTS for one arm slot**, and
the mechanism is arithmetic: mint every ~2 min (the D5 regression, fixed before
this wave by `3b8d6cd6`) ÷ retire after 15 min (`ARM_WORKING_STALE_MIN`) = 7-8
alive. The snapshots show exactly that — `working_count` pinned at 8 with a
transient 9 on each state-change burst.

**The control experiment is in the same table.** The S1 slot minted at the
identical cadence (ids 92, 94, 96, 98) and put **zero** orders at the broker,
because `Kind='limit'` met the ±band check while S2's `stop_entry` branch
`continue`s before ever reaching it. Same minting bug, opposite outcome.

**Also corrected:** the slot is **22 rows (placement_seq 0..21)**, not 9. The
nine in 1664 are the ones standing at one instant — a sample, not the total.

**Consequence for this wave:** D3, the per-slot invariant, is the half that
addresses the incident, and it is load-bearing. D1/D2 remain correct and worth
having *on their own evidence* (the 3-of-47 number above), but they are not what
caused 1664. Saying otherwise would have been a fix aimed at the wrong defect.

**OWNER RULING, 2026-09-06 — correction accepted.** C2 was nine concurrent
placements, not nine failed cancels. D1/D2 stand on the 92%-unverified evidence;
**D3 is the incident's actual fix.** Recorded here and at the head of checklist
class 81 so nobody arriving from snapshot 1664 rebuilds the wrong half.

### C3 — the one-contract rule cannot see the broker · **CONFIRMED** [A]

Nothing consulted the book before a placement. The guard sequence before
`PlaceLimitEntry` checked wrong-side and band only.

### C4 — the material facts already exist · **CONFIRMED** [A]

F12 snapshots arrive every 30 s and on every order state change; Go parses them;
leg 4 reads them. Nothing new was needed from the AddOn, and **no C# was
touched.**

### C5 — non-terminal cancel states · **CONFIRMED with counts** [A]

Every distinct order state across 360 live frames: `Accepted` 983 ·
**`CancelSubmitted` 130** · `Submitted` 44 · `Working` 32 · `Initialized` 22 ·
**`CancelPending` 22**. None appear in `terminalOrderStates`, so a cancel in
flight already reads non-terminal — the safe direction, and the reason the slot
correctly stays locked while a cancel travels.

### C6 — the seam is off because of C1-C3 · **CONFIRMED**

Wave B's own boot line says so in the running binary, citing snapshot 1664.

---

## 2 · THE FIX, AND EVERY PIN RED BEFORE GREEN (A8)

| pin | RED (quoted) | GREEN |
|---|---|---|
| **E1** slot live | `1 order(s) reached the wire while nine were already live at the broker for this slot` | refused, with signal/order/snapshot ids |
| **mint hole** (my own bug) | `a cancel_pending row minted a replacement: 1 rows before, 2 after` | refused by UpsertArm |
| **E3** timeout | — | `after 10 cycles with no book: state=cancel_pending attempts=5 sends=4 snapshot=0` |
| **E7** contract | `a cancel SEND is being read as a CONFIRMATION at 1 site(s)` | `scanned 512 production .go files; no call site treats the cancel return as confirmation` |

**E7 earned its keep immediately:** it found a **sixth** site I had not fixed
(`position_desync.go:86`). That one only logs — it writes no ledger state — but
the rule is enforced on the *shape*, because "does this block also write a
terminal state?" is not a question a text scan can answer honestly. Its wording
was already correct ("sent"); the branch was flipped to test the failure.

### A hole this wave itself opened

Adversarial review of my own in-flight code found that `UpsertArm`'s three
decision points all predate `cancel_pending`: the sort key ranked it with the
*terminal* rows, the working-row refusal did not cover it, and the mint branch
fires on "not armed AND has a signal id" — which a `cancel_pending` row
satisfies exactly. It would have minted a fresh `armed` row with the signal
cleared and placed a **second order while the first may still rest**: the 09-04
stacking arriving by a new road. Pinned RED, then refused with its own message.

### Deliberate consequences, stated rather than buried

- **`shadowed` → `cancelled`.** The condition-shadowed cancel path now goes
  through the lifecycle, so its terminal state changes from `shadowed` to
  `cancelled` with `condition_shadowed` preserved as the reason. The table has
  never held a `shadowed` row (0 of 67).
- **`ListNonTerminal` now includes `cancel_pending`.** This is a shared
  definition — the boot sweep, the sync helper and cutover leg 4's ledger side
  all read it. The change makes each *more* conservative and makes leg 4 agree
  with the broker rather than with our intentions. It is the one place this wave
  touches a shared definition, and it is required for D1 to mean anything.
- **The slot key is `(plan_id, scenario, leg_index)`** — `version` is a mutable
  last-touch column, so keying on it would split one slot into several and let a
  re-authorized slot place beside its own live order.

---

## 3 · D4 — THE RECONCILIATION, AND WHY ITS COUNTS ARE NOT HERE

The pass is built, wired (`reconcileOncePerBoot`, 1 production call site) and
three-state. **It has not produced live counts, and I will not invent them.**

It runs at the first cycle where a usable book exists, and **there is no usable
book right now** (§5). At process start there is none either, which is why the
boot line's reconciliation half reads `reconciled=n/a (no broker book yet)`
rather than three zeros.

The historical equivalent, from the store: of 47 broker-reaching rows, 3
confirmed_gone by a broker frame, 34 unconfirmed-by-construction (our own
assertions), 10 filled.

---

## 4 · THE WAVE IS NOT PROVEN BY A GREEN SUITE

Stated plainly, as the dispatch requires. The suite is 28 ok / 0 FAIL at the
head, and that proves the branches behave on fixtures. **None of the five proofs
Section F asks for has happened:**

- the first cancel moving request → pending → cancelled with its settling
  snapshot id — **has not happened**
- the first `cancel_pending` surviving a timeout with its WARN and counter —
  **has not happened in production** (it is pinned in test)
- the first D3 refusal — **has not happened**
- the D4 reconciliation counts on the live store — **has not happened**
- the boot line with real numbers — **has not happened**

CME is closed and the broker book is dark; none of these can happen before the
next session.

---

## 5 · A15 — WHAT THE OWNER WILL STILL SEE WRONG

- **THE BROKER BOOK IS DEAD RIGHT NOW, AND THIS BLOCKS THE CUTOVER.** Newest
  persisted snapshot is id 6159 at 2026-09-06 05:04:25 UTC — **1,813 s old**
  against a 60 s bound. The Go process is alive and logging; bars are ~36 h old
  (CME closed). **Cutover leg 4 therefore FAILS stale**, and A5 forbids
  overriding it.
- **WHILE THE BOOK IS STALE, THIS WAVE REFUSES EVERY PLACEMENT.** That is D3 as
  owner-ruled ("an unverifiable slot is not an empty slot"), and on a closed
  market it costs nothing. **In a live session with a dark AddOn it means no arm
  is placed at all** — a trading-availability consequence created by a telemetry
  dependency. **The owner ruled on this (2026-09-06): it must read as an OUTAGE,
  not as a quiet no-trade day.** So a staleness refusal now raises ONE P0 in-app
  alert per outage, carrying the book age, deduped on the outage's start instant
  and acked — banner cleared — when a fresh book returns. A later outage raises
  its own. The `arm_slot_unverifiable` counter is unchanged: one alert beside
  it, not a second counter. Pinned, and mutation-checked: removing the raise
  makes the pin fail with "got 0".
- **`stop_price` is 0 on every order the broker has ever shown us** (n=1,233,
  zero exceptions), because the AddOn's submit-slot fix is written but **not
  compiled**. Wave B's F2 is what changes that.
- **Two hardcoded strings still say leg 4 reads the ledger** —
  `kernel/levels_volume_boot.go:49` and `class33_cutover_gate.go`'s `g.Note` —
  while F12 has made the broker the source. Not this wave's footprint; flagged.
- **The AddOn acks a cancel it may have failed to execute**
  (`VLTraderTCPClient.cs:1665-1707` removes its entry *before* `Account.Cancel`
  and `SendAck`s regardless). Go-side confirmation now covers this, but the C#
  remains wrong and is Wave B's ground.
- **`cancelArmedOrdersSyncWith` still counts a FILL as a cancel-ack** — its test
  is "is the row still non-terminal", and a fill also leaves that set. Untouched
  here (it is the flatten path); flagged for its own wave.
- **A class-75 contract test DOES exist**, contrary to what the previous wave
  reported: `TestSystemMapStopEntryRefsResolve` line-checks the stop-entry
  MAPCHECK region and it caught all seven references my edits shifted — twice.
  It is scoped to that region, not to boot-line text generally.
- **The 🧾 glyph is shared by nine other log sites.** A watcher must key on the
  text `cancels:`, never on the glyph.

---

## 6 · ROLLBACK

```
git -C ~/nofx checkout dev && git -C ~/nofx reset --hard <prior-dev-sha>
mv ~/nofx/nofx-bin.old.f516da7c ~/nofx/nofx-bin      # named for the rev it HOLDS
echo f516da7c > ~/nofx/deploy/RELEASE
kill -9 $(pgrep -f nofx-bin)
```

No data migration to reverse: the three new columns are additive and default 0,
which means "no cancel has been requested for this row" — true of every
historical row. No row is deleted and no state is rewritten by this wave.
