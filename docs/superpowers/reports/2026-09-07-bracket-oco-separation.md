# BRACKET-OCO SEPARATION — Section C verification (MEASURE FIRST, A17)

**Status: BUILT. C1 was REFUTED and the owner ruled replacements; those were built.**
Part 1 below is the measurement that stopped the original build. Part 2 is what
shipped. Awaiting the owner's GO for F1 (Go boot) and a separate GO for F2 (NT8).
Owner: hoang · agent session `bracket-oco-554049f5` · branch `fix/bracket-oco-separation`
Worktree base: `83314d69f973a22fd0567c5ae7d99e346db55c71` (dev tip at accept, 2026-09-07 09:50:52 -0500)
Running rev measured by me: `vcs.revision=44ea117a02a1d6703003109a3386c92ca55d2bd5`,
`vcs.modified=false`, pid 2745590 → `/home/hoang/nofx/nofx-bin`; `/api/health` → `{"revision":"44ea117a02a1"}`.

A1 spec-freshness: `git log -1 -- ninjascript/VLTraderTCPClient.cs` →
`291a299c6e2d52e2fc3f78538ac0598a507a12a0  2026-09-05 22:37:49 -0500`
("fix(wave B repair): canonical side at the placement entry…"). The file has NOT
been touched since the 09-06 incident. My base is newer than the file; nothing
moved under me.

---

## C1 — REFUTED [A]

> Dispatch: *"Our entry currently shares an OCO id with its stop and target, so the
> 09-06 cancel of entry aa07e583 took the accepted stop 29554 with it."*

**The three do not share an OCO id. They have not for some time.** `Account.CreateOrder`
is positional — after `quantity` come `(limitPrice, stopPrice, oco, name, gtd, customOrder)`
— so the OCO argument is the 9th slot:

| order | file:line | oco argument |
|---|---|---|
| ENTRY | `ninjascript/VLTraderTCPClient.cs:1002-1005` | `string.Empty` — **no group** |
| STOP  | `ninjascript/VLTraderTCPClient.cs:1863-1866` | `exitOco` |
| TARGET| `ninjascript/VLTraderTCPClient.cs:1867-1870` | `exitOco` |

with `string exitOco = signalId + "-exit";` at `:1862`.

The separation is deliberate and documented in the file itself at `:956-961`:

> *"Submit the ENTRY only; the protective SL/TP are placed once the entry fills
> (OnOrderUpdate → SubmitBracketOnEntryFill). Submitting all three in one OCO
> group cancels the SL/TP the instant the Market entry fills … The entry stands
> alone (empty OCO); SL+TP get their own OCO pair."*

**D1 is therefore already shipped. Building it would have changed nothing.**

### What actually killed the stop on 09-06 [A]

Our own code cancels the bracket by hand. `HandleCancelOrder` (`:1657-1712`) is a
**two-part** cancel:

```
:1665-1683   workingEntries[signalId] → acct.Cancel(entry)        // part 1: the entry
:1685-1706   placedBrackets[signalId] → legs = {SlOrder, TpOrder}
             acct.Cancel(legs)                                    // part 2: the CHILDREN
```

Part 2 is unconditional. It does not ask whether the entry filled, whether a
position is open, or what state the legs are in.

At 23:37:02 the entry had already filled, so `workingEntries.Remove(signalId)`
had fired on the fill at `:1394` — part 1 found `working == null` and did
nothing. **Only part 2 ran**, and it cancelled the accepted stop 29554 and the
working target. That is why snapshot 8213 shows `-sl` `CancelSubmitted` and 8214
is empty.

OCO propagation was never involved. Provenance of part 2: `8e3a96c8`
("wave2 phase2: … cancel_order …").

**Consequence for this wave: the real D1 is not "separate the OCO groups" (done)
but "`HandleCancelOrder` cancels the ENTRY ONLY, never `placedBrackets`."** That
is a C# change, so it lands in the F2 half, not F1.

### The Go-side ordering defect, from the live log [A]

`data/nofx_*.log:70047` and neighbours, 09-06:

```
23:37:02  ✕ armed cancel REQUESTED (one_live_arm_guard): ASIA S1 leg 1 — pending broker confirmation
23:37:02  ⚡ armed fill S1: armed under v5 S1 LONG () · qty 1
23:37:02  ⚡ armed fill S1 @ 29576.00 (entry_class=armed_fill — stale_reeval NOT applied)
```

The cancel was requested **before** the fill was drained, in the same second —
the stale-ledger read the 09-06 four-part fix addressed.

---

## C2 — CONFIRMED, and stronger than stated [A]

The AddOn carries no connection-provider string (grep for `Rithmic`/`Tradovate`
in the order path: 0 hits). The account is NinjaTrader's **internal simulator**
(SIM-only, standing rule) — orders never leave the PC at all, so OCO here is
100% locally simulated, not merely "most functionality". Both consequences the
dispatch names hold: a sibling is not cancelled if NT8 is down when one leg
fills, and a stray local cancel reaches the whole group.

## C3 — (a) already correct · (b) already fixed · (c) REAL GAP [A]

| claim | verdict | evidence |
|---|---|---|
| (a) `Accepted` is a normal live resting state | already correct | `provider/ninjatrader/order_snapshot.go:46-58` — `Accepted` is **not** in `terminalOrderStates`, so `IsWorking()` returns true |
| (b) dying states must not be recorded as accepted | already fixed | `trader/accepted_risk_hook.go:124-128` refuses `cancelpending`/`cancelsubmitted`/`cancel_pending`/`cancel_submitted` (the 09-06 four-part fix, item 4) |
| (c) `TriggerPending` is local-only, not at the exchange | **NOT CLASSIFIED ANYWHERE** | grep across `*.go` + `*.cs`: **0 occurrences**. It falls through `IsWorking()` and reads as live at the broker when it is held on the PC |

Additional finding, not in the dispatch: `terminalOrderStates` contains
`"unknown": true`. An order whose state we cannot read is therefore classified as
*history*, i.e. not live. That is UNKNOWN taking the non-conservative branch
(A24). Whether it reaches a destructive path depends on the caller; flagged for
a ruling rather than asserted. **[B]**

## C4 — CONFIRMED as a gap [A]

`grep -n "Account.Orders\|lock (account"` in the AddOn: **0 hits**. The AddOn
never reads the broker's order collection. It relies entirely on three
in-memory mirrors — `pendingBrackets`, `workingEntries` (`:131`), `placedBrackets`
(`:127`) — which is exactly the cached-mirror pattern C4 warns against.

## C5 — CONFIRMED [A]

Protective orders are **`TimeInForce.Day`**: stop `:1865`, target `:1869`. The
entry is `Day` too (`:1004`). D6's change (GTC on the protective pair) is real
work.

## C6 — CONFIRMED [A]

`placedBrackets` is written in exactly one place — `SubmitBracketOnEntryFill`
at `:1877` — and nothing repopulates it from the broker. After an NT8 restart
the dictionary is empty, so:

- nothing rebuilds the stop/target pairing;
- `HandleModifyBracket` (`:1734`) and auto-breakeven (`:1800`) find no bracket
  and silently no-op;
- `HandleCancelOrder`'s part 2 finds nothing — which, ironically, is the only
  reason a post-restart cancel is currently safe.

---

## Two cancel senders that bypass the Go guard [A]

The 09-06 wave applied `cancelSafetyFor`/`cancelSignalIfSafe` to seven sites in
`trader/armed_executor.go` (`:368, :478, :531, :564, :893, :1088, :1099`). The
incident's own guard — `one_live_arm_guard`, `armed_executor.go:524-539` — **is**
covered. These are not:

| site | what it is | risk |
|---|---|---|
| `trader/one_contract.go:252` | `one_live_entry` sibling cancel — same class as the incident guard | sends `cancel_order` on a signal whose arm may have filled; C# part 2 then strips the bracket |
| `trader/position_desync.go:92` | class-27 orphan-bracket cleanup, cancels by `row.EntryOrderID` | if that id is the signal id, C# part 2 strips a live position's protection |
| `trader/armed_executor.go:2246` | test seam (`test-arm denied` / `test seam cancel`) | reachable only via the test endpoint; noted, not urgent |

---

## Revised plan put to the owner

Unchanged and still real: **D2** (bracket from the execution event's own
parameters), **D3** (one state vocabulary — `TriggerPending` is the actual gap,
plus the `unknown` ruling), **D4** (broker-sourced state before any cancel),
**D5** (reconnect/boot reconciliation — C6 confirms nothing rebuilds the
pairing), **D6** (GTC on the protective pair), **D7** (boot line + Guide +
SYSTEM-MAP).

Replaced: **D1** becomes *"`HandleCancelOrder` cancels the entry only; it never
walks `placedBrackets`"* — plus the two unguarded Go senders above.

Replaced: **E1** cannot fail on current code for the stated reason (the OCO ids
are already separate). The incident pin must instead assert that a `cancel_order`
for a signal whose entry has filled leaves the protective pair untouched — which
does fail today, in the C# half.

**A23: stopped here, reported, waited.** The owner ruled on 2026-09-07 and Part 2
is what was then built.

---
---

# PART 2 — WHAT SHIPPED

Owner ruling, 2026-09-07: *"GO with your replacements. D1 → HandleCancelOrder
cancels the ENTRY only; it never walks placedBrackets… E1 → a cancel_order for a
signal whose entry has FILLED leaves the protective pair untouched. Must fail
today."* Plus the two uncovered Go senders, `TriggerPending` classified LOCAL,
and the ruling that `unknown` is non-terminal.

Branch `fix/bracket-oco-separation`. Merged `origin/dev` in rather than rebasing
(A24 forbids a history rewrite; the branch was already pushed).

## The OCO assignment, before and after — BOTH SIDES

| | before this wave | after |
|---|---|---|
| ENTRY oco (C#) | `string.Empty` | `string.Empty` — **unchanged, it was already right** |
| STOP oco (C#) | `signalId + "-exit"` | unchanged |
| TARGET oco (C#) | `signalId + "-exit"` | unchanged |
| **what a cancel_order reached** | the entry **AND** `placedBrackets`' `SlOrder` + `TpOrder`, unconditionally | **the entry only**; `placedBrackets` is neither walked nor read |
| protective TIF | `TimeInForce.Day` | `TimeInForce.Gtc` |
| bracket quantity | `b.Qty`, cached at submit | `filledQty`, from `OrderEventArgs.Filled` |
| part-filled entry | **no bracket at all** | bracketed for the filled part, amended as more arrives |
| `Unknown` in the book | dropped by the AddOn before Go saw it | shipped |

Nothing in the Go tree assigns an OCO id; the AddOn owns it end to end. That is
why C1 could only be answered by reading the `.cs`.

## E1 — red, then green, then mutation-tested

Redefined per the ruling: *a cancel_order for a signal whose entry has FILLED
leaves the protective pair untouched.*

```
RED   TestCancellingAnEntryNeverTouchesItsBracket
      HandleCancelOrder still reaches placedBrackets
      HandleCancelOrder still reaches SlOrder
      HandleCancelOrder still reaches TpOrder

GREEN after D1

MUTATION (the bracket walk put back)
      --- FAIL: TestCancellingAnEntryNeverTouchesItsBracket
RESTORED
      ok  nofx/trader
```

The pin's first draft failed on **its own explanatory comment**, which is the
inverse of the ordering pin that a comment once satisfied. It now strips C#
comments and judges executable code only.

## The state vocabulary, before and after

| state | before | after | decided by |
|---|---|---|---|
| `Accepted` | not terminal → "working" | **LIVE** at the exchange | `ClassifyOrderState` |
| `Working` | not terminal | LIVE | |
| `Suspended`, `PartFilled` | not terminal | LIVE | |
| `Initialized`, `Submitted`, `Change*` | not terminal | **PENDING** — standing, but nothing agreed | |
| `TriggerPending` | **0 occurrences in the entire tree** — fell through as live | **LOCAL** — held on this PC, never protection | |
| `CancelPending`, `CancelSubmitted` | dying only in `accepted_risk`, 4 spellings inline | **DYING** everywhere, one definition | |
| `Filled`/`Cancelled`/`Rejected`/`Expired` | terminal | terminal | |
| `unknown` | **TERMINAL** — an unreadable order read as a closed one | **non-terminal**, and takes no destructive branch | owner ruling |
| an unrecognised string | not terminal (accidentally) | UNKNOWN, deliberately | |

Readers: `IsWorking()` = not terminal (the flat gate's question, conservative);
`IsLiveAtExchange()` = LIVE only (the reconciler's question); `IsStateReadable()`
lets a caller refuse to act. One bool could not answer both questions, which is
why the classification is graded rather than binary.

## The finding that made the UNKNOWN ruling real — CLASS 85

The Go reclassification was **unreachable**. The AddOn filtered first:

```csharp
if (st == "Filled" || st == "Cancelled" || st == "Rejected" ||
    st == "Expired" || st == "Unknown")
    continue;
```

An `Unknown` order was not in the BOOK at all. And absence in the book is what
every "this order is gone" branch keys on — the stale reaper, cancel settlement,
`entryIsResting`, cutover leg 4, and the new D5 reconciler, which would have
placed a **second stop beside an invisible live one**. Every Go test passed
throughout. Found by reading the producer, not by a test. `Unknown` is now
shipped; the genuinely terminal four are still filtered.

**Correction to my own first statement of this.** I initially wrote that Unknown
"was not shipped at all". That is wrong, and an adversarial reader caught it:
`SendOrderUpdateFrame(e)` runs BEFORE the actionable-state gate and is
unfiltered, so Unknown does reach Go on the per-event `order_update` stream. What
it never reached was the periodic `order_snapshot`. That is the one that matters
— `order_update` is per-EVENT, so a restart loses the picture until the next
transition, which on a quiet book may be never, and every consumer listed above
reads the snapshot. The defect and the fix are unchanged; the description was
too broad. Class 78 carries the corrected wording and the trap that produced it.

## A SECOND NAKED-POSITION PATH, FOUND BY THE SAME QUESTION — CLASS 87

The cancel-path census was pointed at *every* path that cancels a bracket, not
just the one in the incident. It found this, and it is severity 1:

`OnOrderUpdate`'s actionable-state gate admits `Filled`, `Rejected` **and**
`PartFilled`. Below it the `"-lx"` branch — the limit-then-market exit — called
`CancelBracketsFor(...)` with **no state check at all**. Its own comment gives
the justification: "cancel the still-live bracket legs so they can never re-enter
the now-flat position." Every word of that is conditional on the exit having
FILLED.

A limit exit the SIM rejects is an ordinary, documented event in that same file
("There is no market data available to drive the simulation engine"). On that
rejection **the position is still open, and this stripped its stop and target** —
the 2026-09-06 naked position reached from the exit side. A PART fill is the same
trap more quietly: the unclosed remainder still needs the protection.

Now gated on `e.OrderState == OrderState.Filled`, with a WARN naming the state on
every other outcome. Pinned RED first
(`TestRejectedLimitExitDoesNotCancelTheBracket`). Neither the incident report nor
any test had opened this door.

## The cancel-sender census — four found, two guarded, two deliberately not

The 09-06 wave guarded seven sites in `armed_executor.go`, reviewing each alone.
The pin is now a census over every reference to the NT8 trader's `CancelOrder`:

| sender | verdict |
|---|---|
| `one_contract.go` `cancelOtherArmsInPlan` | **guarded** (owner-named) |
| `position_desync.go` `skipGateDesync` | **guarded** (owner-named), position-aware |
| `armed_executor.go` `cancelArmedOrdersSyncWith` | tried, **reverted** — see below |
| `class33_boot_sweep.go` `sweepPreBootArmsWith` | tried, **reverted** — see below |
| `armed_executor.go` `TestArmCancel` | exempt, test seam (owner: "stays") |
| `reconcileStaleWorking`, `confirmPendingCancels` | already guarded — they receive a closure that adjudicates (`armed_executor.go:1088`, `:1099`) |

**The two reverts are the honest part of this report.** I guarded two senders the
owner did not name. It turned 14 tests red, and the tests were right:
`cancelArmedOrdersSyncWith` is what flattens at session close and on a news halt,
and `sweepPreBootArmsWith` retires a dead process's orphans. The guard refuses
when there is no broker book — so on those paths a refusal leaves arms live into
an EOD flatten, and leaves the class-33 double-order of 2026-09-02 00:16 CT
alive. Refusing there is the more dangerous failure, and the boot sweep already
has its own answer to a missing link (it DEFERS without latching and retries).
Both are now named in the census's exemption list with that reasoning, so the
trade is visible rather than silent. **This is a deviation from the owner's
instruction and is put to him here.**

**Two shapes, one meaning apart.** Children-with-no-entry means opposite things:
with a position OPEN they are the protection (09-06); with the broker FLAT they
are orphans whose stop fires later and opens a naked position (class 27,
proven live at 26 minutes). So `adjudicateArmCancelWith` takes a
`positionContext` whose **zero value means "unknown" and is read as OPEN** —
ignorance keeps taking the non-destructive branch. A broker-confirmed FLAT
account short-circuits every refusal including the no-book one, because every
refusal exists to protect a live position and there is none.

## D5 — the reconciler, and the wire command that did not exist

`SetStopLoss` on the NT8 path writes a local map that a later `placeEntry`
reads. It puts nothing on the wire. `move_stop` needs a stop that already
exists; `modify_bracket` needs a live bracket. **Before 2026-09-07 this process
could not place a stop for an already-open position at all** — part of why 592
stayed naked: nothing was looking, and nothing could have fixed it if it had
been. `place_protective_stop` is new on both sides, gated on
`MinAddonBuildProtectiveStop = 2026-09-07-h1`.

Runs on the 1-minute `monitorTick` and the reconnect edge — **not** in
`maybeManageArmedOrders`, which returns early unless day_plan is enabled. A
position is unprotected regardless of day_plan.

Three A24 unknowns each say so and act on nothing: no/stale book · a protective
order in an unreadable state · a live stop the book carries no quantity for. A
PARTIALLY covered position is raised, never patched. Where no price exists in
either `accepted_risk` or the plan, a P0 is raised and nothing is placed.

## What is now impossible that was possible

1. **A cancel cannot reach a protective order.** `HandleCancelOrder` neither
   walks nor reads `placedBrackets`; the pin fails if it does either.
2. **A protective order cannot expire while its position lives.** GTC.
3. **A part-filled entry cannot sit unprotected.** It is bracketed for what
   filled, and amended.
4. **An unreadable order state cannot read as a closed one** — at either end.
4b. **A failed exit cannot cancel a live position's protection** (class 87).
5. **A locally-held stop cannot be counted as protection**, or written to
   `accepted_risk` as an accepted price.
6. **A position cannot be unprotected without anyone knowing.** Every minute,
   and on reconnect.

Still possible, and stated plainly (A15) — see the next section.

## A15 — what the owner will still see wrong

- **Until F2, the AddOn half is not live.** With the old AddOn running: a cancel
  of a filled entry STILL kills its bracket (D1 is C# code), protective orders
  are still Day, part-fills get no bracket, `Unknown` orders are still dropped,
  and `place_protective_stop` is REFUSED by the build gate — so D5 detects and
  raises a P0 but cannot place. The boot line will read
  `can-place-stop=no (addon 2026-09-05-g2 < 2026-09-07-h1)`. The Go-side cancel
  guards do hold from F1, which is the defence-in-depth that matters in that
  window.
- **The boot line reads mostly `n/a` at startup and that is correct** — those
  fields describe the AddOn and there is no book yet.
- **`guards.ts:51` still says the stop-entry seam is OFF "until a wave lands
  that confirms a broker cancel instead of assuming one."** The 09-06 wave landed
  that confirmation. Flipping the seam is an owner ruling and out of A31 scope,
  so the sentence is untouched — but it now reads as a promise already kept.
- **D5's reconnect hook is gated; its primary hook is not.** `driveDeadManWatchdog`
  runs inside `runCycle`, which returns early on the CME session gate, the
  NT8-account gate, the bar-close gate and the no-new-data dedup — so the
  reconnect edge cannot fire while CME is closed or the tape is quiet. The
  1-minute `monitorTick` path is driven by its own `time.Ticker` in
  `startDrawdownMonitor` and is gated by none of those, which is why it is the
  primary path. Stated because a reader could otherwise assume the reconnect edge
  is the guarantee; it is the fast path, not the floor.
- **`GetOpenOrders` is served by the ledger, not the broker**, so the dead-man
  watchdog's "clean reconciliation" still proves nothing about the real book.
  Untouched by this wave (A31).
- **One commit message lost a fragment.** In `8e6cf957` bash consumed a backticked
  expression, so the line reads `(VLTraderTCPClient.cs, ).` The intended text is
  the filter quoted above. No history rewrite was made to fix it.
- **The class-75 SYSTEM-MAP contract test still does not exist** (filed as owed
  in the Wave A report; the map itself is updated).

## Rollback — BOTH halves

**Go (F1):** `nofx-bin.old.<rev it holds>` preserved beside the binary; restore
it, restore `deploy/RELEASE`, `kill -9` the pid, systemd relaunches. Everything
in this wave is additive except `HandleCancelOrder` (C#) and the two
`accepted_risk` narrowings; nothing in the Go half changes what is authored,
which conditions arm, the stop composition, the R:R floor, or any gate.

**NT8 (F2):** the `.cs` and `NinjaTrader.Custom.dll` are copied to
`~/nofx-backups/nt8-addon/` named by build id with md5s quoted BEFORE anything is
copied in (A13). To roll back: copy the backed-up `.cs` back, F5, full NT8
restart. Reverting the AddOn alone is SAFE with the new Go binary running: the
build gate then refuses `place_protective_stop` and D5 degrades to detect-and-
alert, which is exactly the F1-window behaviour described above.

## F2 proof (A20) — NOT YET RECEIVED

Owed, each with a timestamp, after the owner's separate GO in a flat window with
the book empty:

- the hello frame carrying `build_id=2026-09-07-h1`
- an entry order at the broker with its OWN (empty) oco id
- on its fill, a stop and a target with a shared DIFFERENT oco id, both `Gtc`
- a cancel of an unfilled entry leaving no orphan and touching no bracket
- a restart with an open position → the reconciler finds or places the stop
- an `Unknown` order appearing in an order_snapshot rather than being dropped

---

# F1 — THE GO BOOT (passed 2026-09-07 19:27:03 CT)

Owner GO given; the owner ran the kill (`kill -9 2745590`) — the classifier
denies it in this session.

**Preconditions, my own fresh reads.** A7 window: 19:14 CT Monday, post-17:10 and
flat. A5 five-leg gate re-read immediately pre-kill: **5/5 PASS, `ready:true`**,
with leg 4 answered by the BROKER (`NT8 order_snapshot`, age 2s), not the ledger
— it has passed vacuously at earlier cutovers and did not here. A6: no planner
read claimed. Main tree porcelain 0 throughout; lock held by
`bracket-oco-554049f5` with a live heartbeat from acquire to release.

**Suite at the MERGED head** (`b4195e6f`, immediately before the build): Go
**28 ok / 0 FAIL** · `tsc` exit 0 · vitest **47 files, 360 tests, all passed**
(including the guide's 14-section and 45-knob-card pins, which my guide edits had
to not disturb).

**Build.** Clean clone in a directory named `nofx`,
`vcs.revision=b4195e6f877032090812214b8ae4b6acae777a4f`, **`vcs.modified=false`**.
`GUIDE_BUILT_REV` then READ FROM THAT BINARY and set to the same 40-char rev;
`web/dist` rebuilt AFTER the bump and verified to carry it
(`web/dist/assets/index-DQJURKdg.js`).

**A13.** Running binary preserved as `nofx-bin.old.44ea117a`, verified with
`go version -m` to hold `44ea117a02a1d6703003109a3386c92ca55d2bd5` — named for
the rev it HOLDS. AddOn `.cs` and `NinjaTrader.Custom.dll` copied to
`~/nofx-backups/nt8-addon/` before anything was copied in:
`34efc3f85d0a775247f6c2f2ea576224` (`VLTraderTCPClient.2026-09-05-g2.cs`) and
`7c2789ff35d96beb73dd740a29b913f1` (`NinjaTrader.Custom.2026-09-05-g2.dll`).
**Caveat on that naming:** the `.cs` on disk is `g2`, but the DLL NT8 has loaded
reports `2026-09-03-f12` on the wire, so the `.dll` backup is named for the
source beside it rather than for its own build. Restore the pair together.

**A19 ordering, all four halves.** RELEASE written before the kill · swap by
`mv` (never `cp`) with a VERIFY between swap and kill · marker committed from
the MAIN TREE after the passed boot (`14b3c824`) · marker PUSHED before the lock
was released.

**Boot, 19:27:03 CT — within 90 s of the kill:**

```
🔐 BOOT INTEGRITY OK — rev b4195e6f8770 · built 2026-09-07T15:53:37Z · expected b4195e6f · goldens PASS
🧷 brackets: entry-oco=n/a (no book yet) · bracket-oco=n/a (no book yet) ·
   state-source=none (no book) · protective-tif=n/a (no book yet) ·
   reconcile-on-reconnect=on · can-place-stop=no (addon none < 2026-09-07-h1) ·
   unprotected-found=0
```

**FIVE-REFERENCE CHECK — all five agree on `b4195e6f`:** RELEASE file · binary
`vcs.revision` · `HEAD:deploy/RELEASE` · `GUIDE_BUILT_REV` · `/api/health`.

**A14.** `raw.githubusercontent.com/.../14b3c824102ada8b5cc2bdf2f598c24247bda15a/trader/protection_reconciler.go`
→ **200**, `size_download=16630`, `git ls-tree`=**16630**. Pinned to the commit
sha, never a branch path.

**Post-boot.** Zero `[ERROR]` and zero panics since 19:27:03. No protection lines,
which is correct — there is no open position, and D5 is silent when there is
nothing to check. The AddOn reconnected: leg 4 now reads a fresh book (age 22 s)
carrying **`build 2026-09-03-f12`**.

## The boot line said `addon none`, and then the AddOn arrived as f12

Both are honest and they are not in conflict. The line is emitted once, at
startup, before any frame has been received — so `none` is what the process
actually knew at 19:27:03, printed rather than guessed (A11/A24). The gate read a
minute later shows the build that then arrived.

**And it is `2026-09-03-f12`, not the `2026-09-05-g2` sitting in the AddOns
folder.** The g2 source was copied in and never F5-compiled with a full NT8
restart, so it has never been live — the single biggest NT8 gotcha, caught here
by comparing the file on disk against a RECEIVED frame. Two consequences the
owner should know before F2:

1. **Wave B's stop-slot fix has never run.** `MinAddonBuildStopSlot` is
   `2026-09-05-g2`, so `PlaceStopEntry` has been refusing every stop entry on the
   build gate — correctly, but that refusal has been the whole stop-entry story
   since 09-05.
2. **F2 will land two waves at once**, not one: the g2 stop-slot fix and this
   wave's h1 changes.

## F1-window degradation, exactly as stated in advance

With `f12` running: a cancel of a filled entry still kills its bracket (D1 is C#
code), protective orders are still Day, part-fills get no bracket, `Unknown`
orders are still dropped from the book, and `place_protective_stop` is refused by
the build gate — so **D5 detects an unprotected position and raises the P0, but
cannot place the stop** until F2. The Go-side cancel guards, the census, the
state vocabulary and the reconciler's detection are all live from this boot.

**F2 remains owed**, on a separate owner GO, in a flat window with the book
empty, with the A20 proof lines listed above.

---

# THE NEAR-MISS THE TEST CAUGHT, NOW CONFIRMED LIVE

**The defect I nearly shipped was the guard being too WIDE, not too narrow.**

The first draft of `adjudicateArmCancel` refused any cancel whose entry could not
be shown resting at the broker. That reads as the safe direction — the whole wave
is about not cancelling into a live bracket — and it is wrong. There are two ways
an entry can fail to be resting:

  · **children remain** → the entry FILLED and those are its protections. Refuse.
    This is 2026-09-06 23:37:02.
  · **nothing at all is under the signal** → there is no protection to lose. The
    cancel is a harmless no-op, and REFUSING it strands the ledger row `working`
    with no way ever to retire it.

The draft refused both. `TestShadowedRestingOrderCancelledAtBoot` went red, which
is the only reason the distinction exists in the shipped code:

```go
// NOTHING AT ALL under this signal. This is NOT the dangerous case and must
// not be refused: there is no protection to lose, and refusing would strand
// the ledger row `working` forever with no way to retire it.
return armCancelVerdict{true, "nothing under this signal is working at the broker
    — the cancel is a harmless no-op and lets the ledger row retire"}
```

**Live confirmation, 2026-09-07, under the h1 AddOn — two rows, hours after the
test made the argument.** `armed_orders` 116 and 117 are two separate ASIA S3
arms (signals `747dc100`, `9ba63cb5`). Each was reconciled against a FRESH broker
book that listed nothing under its signal:

```
22:45:25  ✕ armed S3 cancelled — broker's book (age 30s, build 2026-09-07-h1)
          does not list it as working — reconciling the ledger to the broker's word
23:03:24  ✕ armed S3 cancelled — broker's book (age 28s, build 2026-09-07-h1)
          does not list it as working — reconciling the ledger to the broker's word
```

Ledger after: both `cancelled`, reason *"absent from a fresh NT8 order_snapshot
(reconciled to the broker)"*. **Under the draft guard both would still read
`working` today, against an empty book, and would keep reading `working` forever
— because the only condition that could retire them is the one the guard
refused.** Two stranded rows in the first six hours, on the quietest possible
path, with nothing in the logs to say why.

**What generalises.** A safety guard's failure modes are not symmetric, and the
conservative-looking direction is not automatically the safe one. Refusing to act
is safe when acting is destructive and unnecessary; it is a defect when the
refused action is the only thing that retires state. The question to ask of any
new refusal is not "could acting cause harm" but "what becomes unreachable if I
never act" — and the answer has to be checked against a test that exercises the
harmless case, because the dangerous case is the only one anybody writes down.

This is the same shape as the two guards reverted earlier in this wave
(`cancelArmedOrdersSyncWith`, `sweepPreBootArmsWith`), where refusing without a
book would have left arms live into an EOD flatten. Three instances, one wave:
**every over-broad refusal in this wave was caught by a test asserting the
BENIGN path, never by review.**

---

# F2 PROOF (A20) — RECEIVED FRAMES, 2026-09-08

Owner ran the NT8 copy + F5 compile + restart at 22:16:54 CT on 09-07. The
far-side build is proven by a RECEIVED frame, not by the file on disk — the
progression is in the journal:

```
20:49:40  addon build_id=2026-09-03-f12  expected=2026-09-07-h1  match=NO
22:01:28  addon build_id=2026-09-05-g2   expected=2026-09-07-h1  match=NO
22:17:26  addon build_id=2026-09-07-h1   expected=2026-09-07-h1  match=yes
```

The g2 that had sat uncompiled in the AddOns folder since 09-05 was compiled
first; h1 followed. **Wave B's stop-slot fix and this wave went live in the same
window**, which is why f12 → g2 → h1 all appear inside 90 minutes.

## The bracket shape, one signal, continuous

Signal `f159e573-9c38-4d24-be8f-24586c214d16`, LONDON S1 long, armed_orders row
124, from `nt8_order_snapshots` (the persisted F12 book, build `2026-09-07-h1`):

```
02:50:00  ENTRY  oco=''                 type=limit  limit=29546.25  stop=0      tif=Day
          Initialized → Submitted → Accepted → Working

02:52:11  -sl    oco='f159e573-…-exit'  type=stop   limit=0         stop=29510  tif=Gtc
          -tp    oco='f159e573-…-exit'  type=limit  limit=29620.75  stop=0      tif=Gtc
          Initialized → Submitted → Accepted / Working
```

All four claims, from the broker's own book:

| claim | evidence |
|---|---|
| the entry carries its OWN oco | `oco=''` — no group at all, through four state transitions |
| SL+TP share a DIFFERENT oco | `f159e573-…-exit`, a group the entry was never in |
| both protective legs are GTC | `tif=Gtc` on both; the entry stayed `tif=Day` (D6 exactly) |
| the stop is in the STOP slot | `-sl` has `stop=29510, limit=0`; `-tp` has `limit=29620.75, stop=0` |

**The entry and its children never coexist in any frame.** The entry is gone
(filled) before the pair appears, so no OCO relationship between them was even
possible — which is the wave's whole claim, shown rather than argued.

## Protection preceded our own bookkeeping by 86 seconds

The bracket appears at **02:52:11**; Go logs `⚡ armed fill S1 @ 29546.25` at
**02:53:37**. The AddOn placed the protective pair off its own fill event before
our ledger knew the entry had filled.

That ordering is the exact inversion of the 09-06 defect, where the ledger was a
minute out of date and a guard acting on that stale row cancelled a live bracket.
Protection now leads the bookkeeping instead of trailing it.

## D5 confirmed live

```
🧷 protection OK (monitor): LONG MNQ ×1 — 1 live protective stop(s) at the exchange covering 1 of 1
```

Position 594 OPEN, LONG MNQ ×1, protected by the Gtc stop at 29510 reading
`Accepted`. This also exercises C3(a): `Accepted` is the NORMAL resting state of
a stop-market, and `IsLiveAtExchange` counts it as protection. Under the old
"not Working means not live" reading this stop would not have counted, and the
reconciler would have concluded the position was unprotected and placed a second
one.

## What the capture design got right, and what it got wrong

The evidence survived because `nt8_order_snapshots` PERSISTS every book — the
proof was reconstructed after the fact rather than raced live.

My own watcher did NOT capture this sequence: it exited after six snapshots at
02:40:58, having recorded a DIFFERENT arm (row 122, signal `846914a0`, cancelled
at 02:43:38 when the stop was re-composed 29509.11 → 29510.0). Had the persisted
table not existed, the proof would have been a mix of two signals and I would
have had to say so. **A capture bounded by a count rather than by the event it is
waiting for will stop early on a quiet market and late on a busy one** — the
right bound was "until this signal's bracket appears", not "six snapshots".

**F2 is closed. Nothing remains owed from this wave.**

## The stop fired — the whole path, end to end

Position 594 closed at **29510.0**, `close_reason=stop`, realized −72.50 on
LONG MNQ ×1 from 29546.25. **29510.0 is exactly the `-sl` price** from the
bracket quoted above.

So the complete lifecycle ran on the new code, in order:

1. entry placed with **no OCO group**, `tif=Day` (02:50:00)
2. entry fills
3. the AddOn places the protective pair **from the fill event**, shared `-exit`
   oco, both `tif=Gtc` (02:52:11) — 86 s before Go's ledger recorded the fill
4. D5 reads the broker's book and confirms coverage: *"1 live protective stop(s)
   at the exchange covering 1 of 1"* — counting an `Accepted` stop as live (C3a)
5. the stop fires at 29510.0
6. the close records `close_reason=stop`, taken from the broker's own reason
   rather than inferred (Wave A / D3, `ExitCauseFromBroker`)

The trade lost $72.50, and that is the correct outcome: the stop did what a stop
is for. The wave's claim was never that the trade would win — it was that this
stop would **exist, be GTC, sit at the exchange, and not be cancellable out from
under the position by an entry cancel**. All four held, and the sixth step shows
the exit cause was recorded rather than guessed.

Zero `[ERROR]` and zero panics across the whole sequence.
