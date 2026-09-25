# WAVE A — THE RECORD

**Branch:** `fix/wave-a-the-record` · **session:** `wave-a-record-554049f5` ·
**base:** `a45cf551` (dev tip at accept) · **date:** 2026-09-05

**Scope:** four measurement repairs, zero trading behaviour. Every diff is a
write, a schema addition or a log line (A31).

**Running rev measured, not quoted (owner correction, mid-wave).**
`/api/health` → `{"revision":"36648655cfe0"}`; `go version -m /proc/1137991/exe`
→ `vcs.revision=36648655cfe03fc8dccd03403a10922f1621b24a`, `vcs.modified=false`,
built `2026-09-04T18:22:30Z`. **dev is 109 commits ahead of the running binary,
but every file in this wave's footprint is byte-identical between them** — so
C1–C4 describe the code that is actually running, and building on dev's tip
changes nothing under this wave.

**Spec freshness (A1).** All five basis reports were last written at `a45cf551`,
which IS my base — `git log -1 --` on each returns
`a45cf551 2026-09-05 18:22:18 -0500 docs(vet): restore the nine verified section
reports`. Nothing moved under me.

---

## 1 · THE PREMISES — VERIFIED, AND WHERE THEY ARE WRONG

### C1 — recorder contamination · **CONFIRMED, every number** [A]

Measured myself against the live store, not taken from the report:

| claim | measured | verdict |
|---|---|---|
| 677 rows → 423 distinct episode keys | 677 → **423** | CONFIRMED |
| RTH-L 140 rows at ONE price = 14 keys | 140 rows, all at 29199.25, **14** keys | CONFIRMED |
| 471 of 677 rows read ordinal 1 | **471** | CONFIRMED |
| whole void scope scanned per level per read | `trader/detector_record.go:57` hands the ENTIRE `scope.Bars` to `DetectTouchOutcomes` | CONFIRMED |

**The mechanism is sharper than "no watermark".** The watermark exists and works;
its WHERE clause blinds it. `store/touch_outcomes.go` `LastOpenedAtMs` (:254-266)
and `NextOrdinal` (:73-84) both filter `opened_at_ms >= sessionDayMs`, and
`detector_record.go:49` passed the **current** session-day. An episode that
opened before 17:00 CT today is therefore invisible to its own watermark (→
re-recorded every read) and to its own ordinal (→ MAX=0 → ordinal 1, every read).
**Both are CALLER defects. The store functions were always right; they were being
asked the wrong question.** The schema also carries **no UNIQUE index** on the
episode key and **no formation column**.

### C2 — excursions empty · **CORRECTED. The framing is wrong and it changes the fix.**

The premise implies a writer that never fires. It is not:

- **A29, paths shown** (`git grep -h` strips paths, which has produced a wrong
  count in this repo before): `excursionOnOpen` → **2** production call sites
  (`trader/armed_executor.go:1287`, `trader/auto_trader_decision.go:492`);
  `excursionOnBarTick` → **1** (`trader/auto_trader_risk.go:54`);
  `excursionOnClose` → **1** (`trader/auto_trader_clock.go:768`).
- The gate `excursionsEnabled()` → `dayPlanEnabled()` is **open**: 254 plans
  exist through 2026-09-04.
- The table's DDL is **already complete** for D2(b) — `mae_ts`,
  `mae_bars_after_entry`, `bars_held`, `ambiguous_bars`, `ambiguous_exit`,
  `resolution`, `pnl_corrected`, `source`, plus a UNIQUE index on `position_id`.

**The zero has two causes, neither a broken writer.** (i) No trade has closed
under a hook-carrying binary: the hooks landed in `44d4bbb7` (2026-09-02 23:46 CT)
and position 591 — the newest ever — closed 2026-09-03 09:20 CT under `33de2bef`,
which does not contain them (`git merge-base --is-ancestor` → false). The RUNNING
binary `36648655` **does** contain them (→ true), so the live path is armed and
waiting. (ii) **The backfill was built in wave 1A and never run.** That is the
real reason the historical corpus is empty, and it is one call.

So this wave **populates the record** rather than repairing a writer.

**Correction to the premise's MAE claim.** It names 569 and 584 as the rows with
`mae=0.0`. There are **four**: **569, 579, 580, 584**. 517 rows are already NULL
(the E4 fix). And the trap is still live in the DDL, not only in the history:
`trader_positions.mae/mfe` carry `DEFAULT 0`.

### C3 — exit cause · **CONFIRMED, and the implied blocker does not exist** [A]

All 58 eligible rows carry `close_reason='sync'`. But the broker's own cause
**already reaches Go**: `provider/ninjatrader/tcp_framing.go:288`
`ExitReason string \`json:"exit_reason"\` // "sl" | "tp" | "manual"`. It is
logged at `trader/ninjatrader/close_sync.go:197` (`reason=sl`) and **used to arm
a live trading gate** at `:204` (the re-entry cooldown) — and then dropped, eight
lines earlier, when the builder was handed the literal `"sync"`.

**Two writers, not one:** `store/position_builder.go` (full close) and
`store/position.go:557` inside `ReducePositionQuantity`'s auto-close branch. The
second is unexercised on MNQ only because every live position has been one
contract; it becomes reachable the first time a multi-contract position closes in
parts. Both are fixed.

**No C# change was needed** — Wave B's boundary is untouched.

### C4 — accepted risk · **CORRECTED on where the prices live** [A]

The premise is right that nothing persists the accepted terms. My first read
suggested it was un-implementable Go-side: `OrderUpdatePayload`
(`tcp_framing.go:157-170`) carries `SignalID, OrderName, State, FillPrice,
Quantity, Symbol, Account, Seq` and **no price fields at all**.

**That was the wrong frame.** The prices ride the **F12 order_snapshot**:
`ninjascript/VLTraderTCPClient.cs:1525-1526` emits `limit_price` / `stop_price`,
Go parses them at `provider/ninjatrader/order_snapshot.go:34-35`, and
`trader/f12_leg4.go:112-172` already reads `o.StopPrice`. The AddOn fires a
snapshot immediately on any state change (`VLTraderTCPClient.cs:1600`), so an
acceptance delivers the broker's own terms. **D4 is implementable entirely
Go-side.**

**And the acceptance event was being thrown away.** `onArmedOrderUpdate`
(`trader/armed_executor.go:1204`) switched on `filled / partfilled / rejected /
cancelled` and had **no case for `accepted`**.

`store/armed_append_only_test.go` is a *refusal* test (UpsertArm won't rewrite a
working row), **not** an append-only record — so D4 genuinely had to be built.

### C5 — the clock · **CONFIRMED, and it is not the worst trap** [A→B]

The stated `created_at` (−05:00) vs `updated_at` (UTC) mismatch on `armed_orders`
is real. A larger one sits beside it, reported for the owner and **deliberately
not migrated** (D5 says: do not migrate other tables): `trader_positions.updated_at`
is documented as milliseconds and holds **epoch-SECONDS in 68 rows** — 68 of the
71 day-plan-era rows. An era filter written against it returns **3 rows instead of
71**, silently. The split is interleaved (577/578/579 are millis; 576 and 580 are
seconds), so it can never be healed by an id range — only by value range. **This
wave writes one convention and mixes nothing** (D5), and E10 pins it.

**No migration runner exists** in this repo — no `migrations/` directory, no
schema-version table. Migrations are hand-ordered `Migrate()` calls in
`store/store.go` plus a flag-guarded block in `main.go`. I followed that idiom
rather than inventing a runner.

---

## 2 · THE FIX, AND EVERY PIN RED BEFORE GREEN (A8)

| pin | RED (quoted) | GREEN |
|---|---|---|
| **E1** pre-formation | `97 of 99 recorded episodes opened BEFORE the level formed` | `pre-formation=0` |
| **E2** duplicate | `297 rows after three identical reads, expected 99` | `99` |
| **E3** ordinal | `session-day 1788472800000 has 28 episodes all carrying ordinal 1` | no duplicate ordinals |
| **E6** exit cause | `broker said "sl", the row records "sync", expected "stop"` | `stop` / `target` / `manual`; `broker=""` PASSED throughout (the eleven CEX paths are byte-identical) |
| **E7** accepted risk | — (new table) | arm 35: accepted 29355 vs ledger 29351.6284728996 → **drift 3.3715271 pts**, both readable |
| **E9** A29 wiring | removing the call site → `--- FAIL` | 4 writers, each ≥1 production call site |
| **E8/E10** safety + clock | — | no panic with the DB closed; epoch-ms UTC only |

**E2 was a bad pin first, and I rewrote it.** It passed on my initial fixture,
and A8 is explicit that a pin which cannot fail is not a pin. The fixture let an
episode land inside the current session-day, which raises the watermark and hides
the defect. Anchoring the whole tape before the session boundary — the actual
RTH-L situation — made it reproduce the live shape exactly.

### The blocking finding: a guard that could not guard (→ checklist class 80)

Implementing D1(a) verbatim would have shipped **a guard that does nothing on the
rows that motivated it.** `FormedAtMs` is set **only** by
`kernel/levels_zones.go` (DEMAND/SUPPLY/OB/FVG). Every LINE level is built by
`lineLevel` (`kernel/levels.go:93-95`), which never sets it:

> **503 of 677 live rows (74.3%)** — VWAP 199, **RTH-L 140**, SWG-H 29, OR-H 22,
> POC 20, SWG-L 16, OR-L 12, PDC 12, RTH-H 12, PDH 10, eVWAP 10, PDL 8, ONL 4,
> ONH 3 … **including all 140 RTH-L rows, which are the premise's own evidence.**

My E1 pin passed on the first attempt **only because the fixture supplied a
formation time**. What found it was not a test — the test was green — but asking
separately what share of the live corpus the new predicate can even evaluate.

**What I did instead of hiding it:** such episodes are recorded and marked
`unverified:no_formation`, and `RatesBy` — the ONE chokepoint `DetectorReport`
and every caller inherit — draws from `validity='valid'` only.
`TestLineLevelsCannotBeCertifiedAndAreExcludedFromRates` pins it so the limit is
enforced rather than remembered. **Giving line levels a birth time is the
follow-up that unlocks the other 74%; it is upstream of this footprint.**

**Schema corollary.** The `validity` column takes **no SQL default**. SQLite's
`ADD COLUMN … DEFAULT 'valid'` would have stamped all 677 contaminated legacy
rows as certified the moment AutoMigrate ran — the exact fabrication the column
exists to prevent. Empty means NOT CERTIFIED.

### A31 near-miss, caught by the existing suite

My first D4 patch **replaced** `case "filled", "partfilled":` with the new
accepted case, putting the entire fill body — `SetState("filled")`,
`SetFillPrice`, `materializeArmedEntry` — under an **acceptance**. A resting
order merely acknowledged by the broker would have materialized a position.
`TestArmedOrderUpdateTransitions` went red and I confirmed by stash that the
failure was mine, not pre-existing. Fixed, and
`TestAcceptedOrderUpdateChangesNoLedgerState` now states the rule directly.

---

## 3 · THE MIGRATION — THREE-STATE, MEASURED ON REAL DATA (A30)

Run against a **copy** of the live DB, never the live file:

```
BEFORE: touches=677 (valid=0 … unclassified=677) · excursions=0
RAN:    dup=254  legacy=423  mae0→NULL=4  mfe0→NULL=5  ids=[569 579 580 584]
BACKFILL: scanned=587 computed=68 unrecomputable=519 levels_resolved=567
AFTER:  touches=677 (valid=0 no_formation=0 invalid:pre_formation=0
        invalid:dup=254 legacy=423 unclassified=0) · excursions=587
        (backfilled=587 unresolvable=519) · exit-cause=broker ·
        accepted-risk rows=0 (with broker stop=0) · mae/mfe 0→NULL=4/5
IDEMPOTENT re-run: dup=0 legacy=0 mae=0 mfe=0
```

**254 + 423 = 677, and 423 is exactly the distinct-episode count** — the
classifier's arithmetic reconciles C1's headline independently.

**`invalid:pre_formation` reads 0, and says why.** It is **not decidable** for
legacy rows: the old recorder never stored a formation time, so no legacy row
carries the fact the test would need. We know from the level semantics that all
14 RTH-L episodes are pre-formation, but re-deriving that per kind inside a
migration is guessing (A24). They are `legacy:unverified`, which excludes them
just as firmly and claims nothing. **No legacy row is ever marked valid.**

**Backfill range matters.** The CLI's documented `-backfill 2026-08-15` filters
`entry_time >= fromMs`, so it would have covered **71 of 587** positions and
silently left the rest out. The boot path passes epoch and an empty
symbol/trader — all history, no literal to drift.

Flag-guarded (`WAVE_A_RECORD_MIGRATE`), backup first (**no backup, no write**),
idempotent, and the line reports what is **PENDING** when the flag is off so the
counts are visible without arming anything.

---

## 4 · WHAT IS NOW MEASURABLE THAT WAS NOT

1. **Per-condition MAE/MFE distributions** — 587 excursion rows where there were
   0, with 519 honestly marked unreachable. A distribution built from 68 of 587
   now says which 68.
2. **Why a trade ended** — from the broker's own event, not a log-file
   reconstruction. And "stopped out" is no longer conflated with "lost": rows
   557 and 570 exited at their protective stop **in profit**.
3. **Ledger-vs-broker drift** — a subtraction (`accepted_stop_px − ledger_stop_px`)
   instead of an archaeology through a rotating Windows log.
4. **Which touch rows may form a rate** — one chokepoint, and the 74.3% that
   cannot be certified are visible on the boot line instead of silently inflating n.
5. **NULL vs zero** on excursions — UNMEASURED is now distinguishable from
   "never went adverse".

---

## 5 · A15 — WHAT THE OWNER WILL STILL SEE WRONG AFTER THIS BOOT

- **The touch corpus is still unusable for a rate, and now says so.** `valid=0`
  until line levels carry a formation time. This wave makes the instrument
  honest, not yet useful. Expect `no_formation` to dominate new rows.
- **`trader_positions.updated_at` still mixes epoch-ms and epoch-seconds** (68 of
  71 era rows are seconds). Untouched by design (D5). Any era filter on that
  column is still silently wrong.
- **`trade_excursions` will still gain no LIVE rows until a position opens and
  closes.** The backfill fills history; the live half is unproven until the next
  trade. CME is closed (Saturday), so this cannot be proven before Sunday's open.
- **Reconcile-opened positions still bypass the excursion open hook**
  (`trader/ninjatrader/reconcile.go` materializes with no `AutoTrader` handle).
  Four real closed trades (575, 584, 586, 591) already demonstrate it. Their
  history is covered by the backfill; the live path for that class is not.
- **`close_reason` for the 58 historical rows is unchanged** — still `sync`.
  D3(c)'s reconstruction into a separate `close_reason_reconstructed` column is
  **NOT in this wave** (see §6).
- **Two stale arms (104, 105)** from Friday's `2026-09-04:NY` plan sit in state
  `armed` with empty `signal_id` — never placed at the broker. The boot arm sweep
  is designed for exactly this; leg 4 should read 0 working orders.
- **The class-75 SYSTEM-MAP contract test does not exist in code.** The checklist
  describes it as enforcing the map, and nothing does; I searched for it and
  updated the map by hand.

---

## 6 · WHAT I DID NOT DO, AND WHY

- **D3(c) — `close_reason_reconstructed` for the 58 historical rows.** Not built.
  The route is known and exact rather than fuzzy: `trader_positions.exit_order_id`
  holds the bare signal uuid and NT8 names the child leg `<uuid>-sl`, so a string
  join resolves it without price/time matching. Deferring it keeps this wave's
  historical writes to the two that are decidable from the store alone.
- **No C# change** — Wave B owns the AddOn, and D4 turned out not to need one.
- **No change to `reconcile.go`'s three writers.** They exist precisely because no
  broker close event arrived; assigning them a stop/target cause would be the
  guessing D3(a) forbids. `trader/ninjatrader/netting_fills_test.go:110` pins one
  of them to `sync`, and it still passes.

---

## 7 · ROLLBACK

```
# code
git -C ~/nofx checkout dev && git -C ~/nofx reset --hard <prior-dev-sha>

# binary (A13: name the rollback for the rev it HOLDS)
mv ~/nofx/nofx-bin.old.36648655 ~/nofx/nofx-bin
echo 36648655cfe03fc8dccd03403a10922f1621b24a > ~/nofx/deploy/RELEASE
kill -9 $(pgrep -f nofx-bin)

# data — the migration took its own backup first
ls ~/nofx-backups/wave-a-record/
sqlite3 ~/nofx/data/data.db ".restore '~/nofx-backups/wave-a-record/<stamp>.db'"
```

The migration is additive and reversible by backup: **no row is deleted**, the
classification lives in one new column, and the excursion rows are keyed UNIQUE
on `position_id`.

---

## 8 · THE BOOT — LIVE PROOF (added after the cutover)

**One boot carrying BOTH waves**, on the owner's ruling: `f516da7c` = `e14dd8da`
(Wave A + Wave B) + the arms-census commit. Boot **2026-09-05 23:53:44 CT**, PID
1963305.

```
🔐 BOOT INTEGRITY OK — rev f516da7cadb3 · built 2026-09-06T04:42:14Z ·
   expected f516da7c · goldens PASS                       (no +dirty)
📐 wave-A record migration: 254 duplicate + 423 legacy touch rows marked
   (NEVER deleted, never blessed) · mae 0→NULL on 4 row(s) [569 579 580 584] ·
   mfe 0→NULL on 5 row(s) · backup ~/nofx-backups/wave-a-record/20260905-235345.db
📐 excursion backfill: scanned=587 computed=68 unrecomputable=519
   (no 1m coverage — those rows keep NULLs, never zeros) levels_resolved=567
📐 record: touches=677 (valid=0 no_formation=0 invalid:pre_formation=0
   invalid:dup=254 legacy=423 unclassified=0) · excursions=587 (backfilled=587
   unresolvable=519) · exit-cause=broker · accepted-risk rows=0 (with broker
   stop=0) · mae/mfe 0→NULL=4/5 · arms live=0 (superseded=6 cancelled=51
   filled=10) · migration armed backup=20260905-235345.db
🎛 entry law: … stop_entry_seam=off
```

**Live three-state counts match the dry run exactly** — 254 / 423 / 0-blessed,
and 68 computed / 519 unrecomputable. Zero panics or fatals in the boot.

**Five-reference check — all six agree on `f516da7c`:** RELEASE file · binary
`vcs.revision` · `HEAD:deploy/RELEASE` · `GUIDE_BUILT_REV` (full 40 chars) ·
`/api/health` · the running `/proc/1963305/exe`.

**The kill worked on its first invocation.** systemd recorded
`Main process exited, code=killed, status=9/KILL` at 23:53:39 and restarted at
23:53:44. The owner's "No such process" came from a *second* invocation, not a
failed one — worth recording, because "the kill did nothing" and "the kill
already happened" look identical from the shell.

### A15 — one boot line that can be misread

`🎯 arms: … stop-entry=on(reclaim) …` and `🎛 entry law: … stop_entry_seam=off`
are both true and are **about different things**. The first is the arm KIND
table (`ArmKindFor("reclaim") == ArmKindStopEntry`) — reclaim scenarios are
*authored* as stop entries. The second is the SEAM, and the seam is what decides
whether a stop entry is ever PLACED. With the seam off, **no stop entry is
placed regardless of what the kind table says.** Read together at a glance they
suggest stop entries are live. They are not.

### Still owed after this boot

- **The class-75 SYSTEM-MAP contract test does not exist.** The checklist
  describes a test that greps the map for boot-line text and fails the suite on
  a mismatch; no such test is in the tree. Filed as OWED — not built here, on
  the owner's instruction, so the map stays maintained by hand until it is.
- `valid=0` on the touch corpus, and it will stay 0 until line levels carry a
  formation time (§5). The instrument is now honest, not yet useful.
- The live half of the excursion record is unproven until a position opens and
  closes — CME was closed at boot.
