# W-ONE-BUTTON M2 — the installation maintenance hold and gate

**Lane:** Claude-101 · **Branch:** `feat/one-button-m2-maintenance-hold` · **Base:** `0960a6ac` (dev tip at accept, unchanged since) · **Dispatch:** CTO `1790131309474-27377-000001`, rulings `1790131568827`, `1790133979048`, `1790134409642`, review `1790136965857`, `1790137248636`.

**Status:** see the L14 report on the bridge for the final verdict and the test tails at HEAD.

## What M2 is

Before a one-button update replaces the bot and the NinjaTrader AddOn, the whole installation must stop sending NEW entries, while everything that protects or closes a position keeps running. The only brake the bot had, `stop_until` plus Resume, pauses ONE consumer: one trader's AI decisions. M2 adds an installation-wide hold and one read-only verdict that says whether an update may proceed.

- **The hold** is one file: `<data dir>/updater/hold.json`.
  - Absent means not held, and today's behaviour is unchanged.
  - Present and well-formed means what it says.
  - Present but unreadable means **held** (fail-closed).
  - It is written atomically under a flock, and cleared only by the job that holds it (or `--force`). Only `cmd/maintenance-hold` writes it; an AST scan pins that. The bot and the CLI resolve the data dir through ONE resolver (`internal/installpath`), so the CLI writes the file the bot reads.
- **Every entry producer checks it at its send point:**

| Site | Where | While held |
|---|---|---|
| 1 AI | `executeDecisionWithRecord` | open_long / open_short refused (after stop_until, before any wire side effect); closes untouched |
| 2 Armed | `runArmedPlacementAt`, `placeOneStopEntry` | per-row refusal at the send point, **no early return** (the reaper / drain / confirm / cancel-settlement tail keeps running); the arm stays armed and places after the hold; a refusal never latches "placed" or cancels siblings |
| 3 Picture HTF | before `PictureHtfClaim`; `pictureHtfSend` re-checks | durable `refused` row (the window is seconds; fail-closed); a refusal after the claim is provably unsent → `refused`, never ambiguous `place_pending` |
| 4 Broker | the four `TCPTrader` entry sends | permit held across the wire write; before the B3 dedupe and `beforeSend`; protective / cancel / modify / close never take it |
| 4b Queue | `flushPendingReport` | queued entries dropped and reported, **connected or not**; never re-queued, never resent |
| 5 Planner | before `claimPlannerRead` / `claimWeeklyRead`; `CanForceReset` / `CanForceReread` | no plan row, no budget, no NO-TRADE marker; reset refused before it abandons the chain |
| 6 Resume | `ResumeEntries` | clears the trader's pause only; says entries stay refused |
| 7 AddOn | `maintenance` / `maintenance_ack` | AddOn refuses new entries; every notice answered with a name-free census |

- **Queue drops settle what the caller recorded (M-2).**
  - A never-attempted drop (zero bytes written) settles as follows: the armed `place_pending` row becomes `cancelled`, a Picture `place_pending` row becomes `refused`, and the reason reads `never sent — queued entry dropped by the maintenance hold (job <id>)`.
  - An attempted drop stays `place_pending` (ambiguous) and keeps the gate closed.
  - The AI path cannot be linked (see the new pre-existing class below). It is forgotten, said out loud and alerted, and the gate stays closed on `db_open_positions`.
- **`GET /api/installation-gate`** (`InstallationGateStatus`): every leg quotes its source, and one it cannot evaluate fails. The legs:
  - hold · go_drained · in_flight_sends · queued_signals;
  - planner_in_flight (the union of four claim maps, any trader);
  - traders_nt8 (TraderManager ∪ the Picture registry; a non-NT8 trader fails);
  - addon_ack (current connection, this job, fresh);
  - addon_census (no connected non-SIM connection, no position or working order on any account, and a nil census is not an empty one);
  - ledger_exposure (all trader ids);
  - trader_cutover:<id> (legs 1, 2, 4).
- **`GET /api/maintenance`** plus the boot line `🔒 maintenance: hold= job= since= addon_ack=`. Every field is READ, and addon_ack prints n/a at boot.
- **Wire.** The `maintenance` frame is sent only while held (at accept before the flush, on change, and re-sent every 5 s), plus one release. With no hold file the wire is byte-identical. The hello gains five omitempty epoch fields (CTO Q3). The per-connection record (accept_seq, monotonic accept time, remote port, hello, ack) is replaced on every accept.

## R4 choices (spec silent → fail-closed default)

1. Armed: per-row refusal instead of an early return, so the pass's tail keeps settling state while held.
2. Picture: a durable refusal before the claim. An opportunity seen during maintenance never trades afterwards.
3. A corrupt hold file is sent to the AddOn as held with no job, and the gate needs an ack for job "".
4. The census reads a connection as SIM only when every account seen on it is SIM. A connection with no account reads non-SIM.
5. `queued_commands` is 0 by construction: the AddOn executes frames synchronously on its read thread. A future async dispatcher must report its real depth.

## Findings on the way

- **U2 follow-through [A].** The first cut of the queue drop sat behind the "no connection" early return, so a held queue stayed parked for as long as NT8 was down. Fixed: the hold drops the queue connected or not, and `SendSignal` while held and disconnected returns `ErrEntryHeld`.
- **CTO's M-2 premise [A].** The stale-working reaper cannot retire a dropped entry's row, because it only reads `working` rows and the row is `place_pending`. Without the settle, leg 4 would fail indefinitely. Settle-on-drop was built instead (CTO approved, three conditions).
- **Pre-existing, AI path [A].** `placeEntry` returns no `orderId`, so `recordAndConfirmOrder` writes an OPEN position at the mark with `entry_order_id "<nil>"`, plus a P0 "Filled" alert, for an entry that was only queued. Filed as a checklist class; the fix needs its own wave.
- **Full-suite regressions found and fixed:**
  - SYSTEM-MAP line refs (CTO's run);
  - `ExpectedAddonBuild` lockstep;
  - brand-scope pins on the three wire files (delta documented);
  - a pre-existing race in `TestSinkInstalledBeforeStartPersistsTheFirstFrame` (counted before its insert).

## What was NOT done

- The C# AddOn is **not deployed**: not copied to AddOns, not compiled in NT8, NT8 not restarted. It was compile-checked against the installed NT8 DLLs in a Windows temp dir.
- Nothing was merged, deployed or restarted; no lock was taken; `/home/hoang/nofx` was not written.
- GUIDE_BUILT_REV was not bumped.
- No DB writes outside test temp stores.
- The pre-existing AI "queued send recorded as a fill" defect is filed, not fixed.
- M3 (authz) and M4 (the updater, and the NT8 step, which waits for the owner) are not started.
