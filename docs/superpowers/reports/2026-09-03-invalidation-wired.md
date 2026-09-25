# INVALIDATION-WIRED (checklist class 59)

**Branch:** `fix/invalidation-wired` off `75d923eb` (deployed rev `528edd78`)
**Commits:** `d2805408` · `c52c98e2` (+ this report) · pushed
**Checklist:** entry **59** — renumbered 57 → 59 AT MERGE by the integrator
(nofx-52): 57 went to a magic-epoch class that merged first. PART 1 is 50–59.
**Boot line:** `🛡 arm gate: invalidation-wired=on · armed-under surfaces=on …`
**Status:** MOSTLY LIVE. `beb42e04` merged into dev and shipped as rev
`f478ed88`, booted 2026-09-03 11:10:33 CT (marker `67ff5e9c`, zero ERRO). The
one-live-position commit is DEFERRED to the next boot by owner instruction.

---

## 1. What the four defects have in common

Every one is a record the system writes and nothing reads.

| # | the record | who read it |
|---|---|---|
| E1 | `🎯 scenario S1 → ≈invalidated @ 29285.00` | nobody — labelled "never execution-wired" |
| E2 | `armed_under_version` (attribution wave) | **zero readers** outside the store |
| E3 | `fill_quantity` on the armed row | nothing ever wrote it |
| E4 | the authored-log dedup key | blind to the row's state |

## 2. E1 — the twelve minutes

```
08:50:54  🎯 scenario S1 → ≈invalidated @ 29285.00 (price accepted through the
          level against the trade — display-only estimate, never execution-wired)
09:02:54  ⚔️ armed NY S1 leg 1 short limit 29285.00
09:03:53  filled
09:20:45  stopped at the widened stop — −$140
```

Ledger: `armed_orders` row 35, version 2, S1, short, entry 29285.00, filled.

The system reached its verdict twelve minutes before it armed the trade that
verdict condemned. The label was accurate.

### 2.1 The fix

`EntryGate` gains **leg 3**, arm path only. The verdict comes from
`kernel.EvaluatePlanScenarios` — the same function
`trader/auto_trader_levelstate.go:260` calls for the display line, with the same
`BarsSince(bars, plan.BirthMs)` windowing. There is no second predicate to
drift from the first; that is the void-parity lesson applied here.

It runs **before** the pricing legs deliberately. R:R and min-SL ask whether a
trade is well shaped. This asks whether the setup still exists. A well-shaped
trade into a dead setup is exactly the 09:02 arm.

```
entry_gate: scenario S1 invalidated at 08:50 (accepted through 29285.00)
            — price accepted through the level against the trade
```

Counted under its own class, `invalidated`, placed **first** in
`armRefusalClass` so it cannot fall through to `other` and vanish from the
tally.

### 2.2 Unresolved is not a refusal

No bars, no price, or an `UNEVALUABLE` scenario (the display path refuses to
store a status for those, so the gate must not invent one) returns `ok=false`.
The leg **passes** and logs `invalidation check unavailable for S1 — leg PASSED
(no verdict is not a refusal)`.

### 2.3 The verdict time, not the check time

The evaluator is stateless: it knows a scenario **is** invalidated, not **when**
it became so. Rendering `FormatCT(now)` would have printed the check time under
the words "invalidated at" — and twelve minutes is the entire point of this
wave.

So the evaluator stamps `scenario_invalidated_at:<trader>:<plan>:<scenario>`
**once** on the transition, never overwritten, and the gate reads it. Absent the
stamp it renders "at an earlier cycle", which is true.

## 3. E2 — the card showed one S1 and the owner held the other

The card rendered NY **v3 S1 long**, written 09:15, while the account held a
position armed under **v2 S1 short** — filled 09:03, stopped 09:20. Both
scenarios are called "S1".

`/plan/today` gains `open_position`: the position's own provenance, plus
`version_differs`. `ArmedUnderBlock` renders, in this order:

```
POSITION ARMED UNDER · v2 S1 short @ 29285.00 × 1
PLAN NOW · v3 — the rows below are THIS plan, not the position above
```

Nothing renders when flat, or when the two versions agree — there is nothing to
disambiguate then. The chips also carry `armed_under_version` now.

**A23 note.** `armed_under_version` is **0** on both of today's rows (35 and 36,
created 09:02 and 09:20) because the attribution wave did not boot until 10:28 —
the column existed, nothing wrote it. That is the class-56 disease in a fresh
column: 0 means both "version 0" and "never stamped". The payload sends `null`
plus `armed_under_note`, and the card renders **"version not recorded"**, never
"v0". `armedUnderVersionOf` falls back to the mutable `Version` for the log line
rather than printing zero.

## 4. E3 — filled, quantity zero

Row 35: `state=filled`, `fill_quantity=0`, beside `trader_positions.quantity=1`.
Nothing wrote the column, and 0 is a legal "nothing filled", so the row could
not be read either way.

`SetFillQuantity` is called on the same path that stamps lineage. WHERE-scoped,
idempotent, and **a zero never overwrites a measurement** — pinned, because that
is how the column would silently regain its ambiguity.

## 5. E4 — "armed" four times after it filled

The authored log fires when the ledger lookup finds no existing row. The lookup
is `ListNonTerminal` (`state IN ('armed','working')`), so once a row **fills**
it is invisible, `row.ID` stays 0, and the authored branch runs again.

The dedup value now carries `state=…`, and `armAuthoredLoggable` refuses to log
"armed" for a `filled` / `cancelled` / `expired` row at all — the second guard
is the load-bearing one, since state in the key alone would make a state change
*more* loggable, not less.

### 5.1 CORRECTION — I had this backwards, and my own fix did not work

My first pass recorded, as **[B]**, that a terminal row being invisible to
`ListNonTerminal` meant a filled scenario "can be re-armed". **That was wrong.**
Traced properly:

`UpsertArm` runs its OWN query with no state filter
(`store/armed_orders.go:166`), finds the filled row, and hits MANUAL-CANCEL-WINS
at line 206:

```go
if existing.Version == row.Version && !IsBootSweepReason(existing.StateReason) {
    return nil
}
```

So the re-arm guard **is store-derived**, is a DB read, and holds across a
restart. Nothing was ever re-armed.

What it does is return `nil` having done **nothing**, leaving `row.ID` at 0 —
and the caller then logged `⚔️ armed …` as though it had succeeded. **The guard
was real; the log was the lie.** Measured: five such lines after the 09:03:53
fill, each carrying a different ATR-drifted stop:

```
⚔️ armed NY S1 leg 1 short limit 29285.00 SL 29354.91 …
⚔️ armed NY S1 leg 1 short limit 29285.00 SL 29352.65 …
⚔️ armed NY S1 leg 1 short limit 29285.00 SL 29354.44 …
⚔️ armed NY S1 leg 1 short limit 29285.00 SL 29352.40 …
⚔️ armed NY S1 leg 1 short limit 29285.00 SL 29354.86 …
```

**And my F4 fix was ineffective.** `armAuthoredLoggable(row.State)` read
`row.State` — the DESIRED state the caller had just built as `"armed"`, never
the persisted one — so it passed every time. The drifting stop in the dedup
value made it worse: the value changed each cycle, so the dedup suppressed
nothing either. Two guards, both no-ops.

Corrected: `armedActually(row.ID, row.State)` — the id is the load-bearing
signal, because `UpsertArm` sets it on a create and on a live-row update and
leaves it zero exactly when it declines. The dedup value drops the ATR-derived
prices and keeps the entry, which is the GAR-F6 lesson the refusal path learned
in August and the authored path never did.

### 5.2 The remaining hole, reported

The guard is scoped to the SAME version. `UpsertArm`'s next branch
re-authorizes a terminal row on a **new plan version** — fresh `armed` state,
`fill_quantity` reset, `armed_under_version` re-stamped. With a position still
open, `oneLiveArmGuard` explicitly skips the same side
(`armed_executor.go:601`, "same-side add — outside this guard's scope"), so a
v3 S1 short can arm while the v2 S1 short position is still open and add to it.

Today that did not fire: NY v3 S1 was authored LONG, and the opposite-side
branch of `oneLiveArmGuard` would have refused it. **[A]** for the code path,
**[B]** for the consequence — never observed placing a second same-side arm.
Out of this dispatch's footprint (the owner scoped F2 to "in this version");
worth a ruling.

## 6. Tests

| id | pins | first run |
|---|---|---|
| T1 | the 09:02 arm intent + the 08:50 verdict → REFUSED with the message | **RED quoted**: `got reason=""` with the leg disabled |
| T2 | evaluator unavailable → passes, with the line | RED (undefined) |
| T3 | a live scenario passes; no resolver = leg off | RED (undefined) |
| — | arm-path only, asserted on the REASON | see 6.1 |
| — | the refusal classes as `invalidated`, not `other` | RED |
| T4 | the card's two lines and their ORDER; flat; agreeing; "version not recorded" | RED (no component) |
| T5 | `fill_quantity` stamped, the zero guard, the state dedup, the attribution fallback | RED |
| T6 | `go build ./...` · `go test ./...` · vitest 41 files / 310 tests · `tsc --noEmit` | GREEN |

### 6.1 A test that passed for the wrong reason

The arm-path-scope test first asserted the decision path simply does not refuse.
It failed — because the decision path refused on **R:R 2.00 below floor 3.00**,
nothing to do with invalidation. Had the fixture happened to clear R:R, the test
would have passed while proving nothing about scope. It now asserts on the
refusal **reason** and uses a target that clears the floor, with the reason in a
comment.

### 6.2 My own error, recorded

Demonstrating T1's RED, I disabled the leg with `if false &&`, then reverted with
`git checkout trader/entry_gate.go` — which discarded **the entire file's
uncommitted work**, not just the probe. Rebuilt from the same edits and
re-verified green. It is the class-45 lesson (a blunt revert destroys more than
the thing you aimed at) in a different tool, and the reason A18's "commit ~30
min" exists.

## 7. Boot line

```
🛡 arm gate: invalidation-wired=on · armed-under surfaces=on — the evaluator's
   own ≈invalidated verdict REFUSES an arm (same function as the display path;
   unresolved = pass + a line), and a position states the version it was armed
   under before the live plan's rows are read as its own
```

## 8. Cutover

Rides the next boot alongside the fast-market exemption
(`fix/wake-fastmarket-exempt`). Five-leg gate `ready:true`, owner GO, A13
rollback named by the rev it holds, A19 all four halves with the marker
**pushed** before the lock is released.

**PROOF OWED:** the next arm on an already-invalidated scenario showing the
refusal line, and the next open position's card showing its armed-under version.

---

## 9. F3 WAS INCOMPLETE — TWO MECHANISMS, AND I MISATTRIBUTED THE NUMBERS

Three sessions untangled this. My first account of it was wrong three times, and
each correction came from a peer reading my claim against the code.

**Source for the baseline evidence:** nofx-89's audit,
`docs/superpowers/reports/2026-09-01-full-system-audit.md` on branch
`docs/full-system-audit-0901` @ **`5a2da9ce`** — the A6 revision. A5.3 is
withdrawn there and carries an inline pointer so the superseded paragraph cannot
be read standalone. `7df072a3` remains a valid citation for the `fef656a4` snapshot
itself. Raw URL curl'd **200** from this session (A14). Both of my corrections were independently verified and taken in A6.

### 9.1 The measured population (current rev, nothing borrowed)

```
armed_orders            total 36 · filled 10 · filled with fill_quantity>0: 0 · with 0: 10
trader_positions        587 · system 567 · reconcile 12 · armed_entry 5 · e7_farside_test 3
row 35                  state=filled · fill_quantity=0 · state_reason '' (empty)
stamp_pending in text   0 rows
```

**Every filled armed row has `fill_quantity=0`. 10 of 10.** That is the finding
and it carries its own denominator.

### 9.2 CORRECTIONS — three of my claims were wrong

**C1 — "584 of 586 armed fills" (mine to own).** 584 and 586 are
`trader_positions` **row ids**, not a count and never a ratio. The baseline
finding is: on 2026-09-01, **two** armed fills went unstamped — ids 584 and 586
— out of six closed rows that session-day (581–586). The chat figure came from
nofx-89 misreading their own row ids; the committed report was always correct.
I then published it without checking that `armed_orders` holds 36 rows in
total, which would have caught it in one query. My subsequent framing — "a
positions-era figure I repeated" — was also wrong: it was a row-id pair, not an
era figure.

Consequence: my re-run of the audit's column set returning **3** on the current
rev is not a collapse from 584. It is **3 against a baseline of 2** — the same
order of magnitude, and it makes the deterministic reading STRONGER. A race
producing 2 one day and 3 the next is unremarkable; a small, persistent,
every-day number is exactly what a fold-insensitive miss looks like.

**C2 — `;stamp_pending` transience.** nofx-52: the marker is trimmed in
`reconcile.go`, no row carries it now, so the defect is visible only in
`fill_quantity`. nofx-89: their report line 493 read `armed_orders` 24 and 28 as
`filled ;stamp_pending` at 16:40 CT against fills at 08:37:08 and 13:33:06 —
carries of roughly **8 hours and 3 hours**. Both are true. The trim is
**eventual, not prompt**: verified now, rows 24 and 28 have empty
`state_reason` and their positions are stamped. "Transient" was the assumption
that made `fill_quantity` look like the only symptom.

**C3 — RETRACTED: close-sync does NOT lose a priced close.** I wrote that
`GetOpenPositionByAccountSymbol` was on a live money-loss path. nofx-52 checked
and it is not: `close_sync.go:87-89` sets `side` to `"LONG"`/`"SHORT"` via
`strings.EqualFold` **before** the call, so it passes uppercase and matches the
uppercase rows; `reconcile.go:153-157` and `tcp_trader.go:601` normalise too.
Every caller of the account-scoped lookup already hand-normalises.

The residual there is the **inverse and small**: the 3 lowercase rows (576, 577,
579 — all `armed_entry`, all CLOSED, all 2026-08-31) would be missed by an
uppercase lookup. `UPPER(side)=UPPER(?)` is still correct at all three sites; it
is the whole fix at `GetOpenPositionBySymbol` and a much smaller one at the
account-scoped lookup. A claimed live loss path that does not exist costs
credibility on the ones that do.

### 9.3 Mechanisms 1 and 2

**Mechanism 1 — the materialization race.** `stampArmedFillLineage` returns
early when the position row does not exist yet. Confirmed for row 35: filled
**09:03:53**, position 591 materialized **09:05:14** — an 81-second gap. Fixed
in `StampArmedLineageIfMatched` (95e9a4d0), which takes `posID` directly.

**Mechanism 2 — the side-casing miss, and the DOMINANT one.**

| column | values |
|---|---|
| `armed_orders.side` | `long` 19 · `short` 17 — always lowercase |
| `trader_positions.side` | `LONG` 280 · `SHORT` 304 · `long` 1 · `short` 2 |

```
select count(*) … where side='short' and id=591  →  0
select count(*) … where side='SHORT' and id=591  →  1
```

The fill handler passes the ARMED row's lowercase side, so it could never find
an armed-entry position however good the timing was. That is why it is 10 of 10
rather than intermittent. Fixed with `UPPER(side)=UPPER(?)` (664ab6b7).

This is **class 28, canonical casing** — "one canonicalizer per identifier,
called where the value ENTERS, never at each comparison" — backfilled into
PART 1 by nofx-52 hours before this finding landed on it. Three call sites
normalise by hand; one storage path does not; the mismatch sat between two
tables.

### 9.4 NEW — a late stamp cannot fix the grade, because the reset looks for F

Positions 584 and 586 **are stamped now**: `plan_version` 6 and 5,
`cited_scenario_id` S2 and S3, `entry_order_id` populated. The reconcile got
there eventually.

Their `adherence_grade` is still **D**.

`RepairArmedLineage` clears the grade so W5 can regrade — but only when it is
`"F"` (`trader/ninjatrader/reconcile.go:588`). And a close with no citation
grades **`"D"`** (`kernel/adherence.go:52-54`, "off-plan (no scenario
cited)"). So the reset predicate looks for a grade this path does not produce,
and a late-stamped position keeps its off-plan D permanently. **[A]** — both
rows read D with full lineage, and both code lines quoted.

#### CORRECTION — my mechanism was wrong; the predicate matches a SUBSET

I wrote "the reset predicate looks for a grade that path never produces". Wrong,
and nofx-89 caught it. `GradeAdherence` sets `base = "D"` for an uncited close
and then applies penalties over `gradeLetters = {A,B,C,D,F}`:

```go
if in.InNoTrade  { grade = stepDown(grade, 1) }
if !in.InKillzone { grade = stepDown(grade, 1) }
```

D is second-to-last, so **one** penalty step takes an uncited close to **F**. An
uncited close grades F when either penalty applies and D when neither does.
Verified: 566 and 571 are uncited with grade **F**; 580 is uncited with grade
**D**.

That is worse than an impossible predicate. `RepairArmedLineage` **silently
succeeds on penalised uncited rows and silently fails on clean ones** — so
anyone spot-checking would likely land on an F row, watch the repair work, and
conclude it was fine. The fix is therefore not "also match D": the predicate
must key on **the absence of lineage**, never on a letter that encodes lineage
plus two unrelated penalties.

#### Blast radius — four provably mis-graded, and the discriminator

Both peers counted 5. It is **4**, and the reason is the grade ladder.

A cited row whose direction matched grades **base A** (`plan_band` is
`armed_fill`, not `off_band`/`struct`). Two penalties take A → **C**. **Base A
can never reach D.** So `plan_matched=1 AND plan_band='armed_fill' AND grade='D'`
is impossible from correctly-ordered grading — it can only mean the row was
graded while `Cited` was false.

```
575  reconcile   v3 S2  matched=1  armed_fill  D   ← provably mis-graded
584  reconcile   v6 S2  matched=1  armed_fill  D   ← provably mis-graded
586  reconcile   v5 S3  matched=1  armed_fill  D   ← provably mis-graded
591  reconcile   v2 S1  matched=1  armed_fill  D   ← provably mis-graded
582  armed_entry v3 S2  matched=0  (none)      D   ← LEGITIMATE: base C − 1 penalty
530  system      v2 'off-plan'     matched=0   D   ← LEGITIMATE: sentinel, OffPlan
```

**582 cannot discriminate — and the reason is sharper than "explicable".**
nofx-89 read it as proof that the `armed_entry` path grades before it stamps.
With `plan_matched=0`, two hypotheses give the same letter: base **C**
("direction mismatched") minus one penalty is D, and a grade-before-stamp gives
base **D** (`!Cited`) with no penalty, also D. **The two are observationally
identical on that row.** It cannot discriminate in either direction, so the
armed-path question is **untested, not exonerated** — a wrong answer closes the
question.

**Where the four DO sit — nofx-89's own reversal.** With 582 excluded, all four
impossible-D rows are `source=reconcile` (verified: `reconcile 4`). Their A5.3
headline was "this is not a reconcile-path problem"; the corrected evidence
points the other way. Stated as **absence of a counter-example, not proof** —
the one `armed_entry` D row is precisely the undecidable one.

#### A trap in my own discriminator (nofx-89's catch)

The ladder argument alone is not a safe predicate. Run it without a lineage
clause and it returns **five**:

```
plan_matched=1 AND plan_band NOT IN ('off_band','struct') AND grade='D'
  →  572, 575, 584, 586, 591          ← five
  + plan_version>0 AND cited_scenario_id<>''
  →  575, 584, 586, 591               ← four
```

**572** is `plan_matched 1`, `plan_band armed_fill`, grade D — and
`plan_version 0`, `cited_scenario_id 'TEST-E7'`, with `source` and
`close_reason` both `e7_farside_test`. An `ARMED_TEST_SEAM` artifact, not a
trade.

My 4-in-71 is right because my query carried the lineage clause, but anyone
re-deriving the count from the ladder reasoning alone lands on **five and quotes
a test row as a live defect**. nofx-89's §D-3 flags the same seam contaminating
`store/position_query.go`'s unfiltered counts, so this is that class recurring in
a new query rather than a one-off. Denominator independently confirmed at 71.

**591 is not a post-fix regression.** It is today's, but it was graded before my
code shipped: row 35 filled 09:03:53, position 591 materialized 09:05:14, and
the boot carrying this wave was 11:10:33.

Against the day-plan-era distribution (A 30 · B 22 · D 10 · C 5 · F 4, n=71),
**4 of the 10 Ds are provably wrong**, so an adherence rate computed today
under-reports plan-following by 4 in 71 — not 7.

#### The five §F1 ids are three different failure states

nofx-89's "25% blind" was correct when measured and is now three states behind
one number:

| ids | state |
|---|---|
| 566, 571 | uncited, grade F — blind, and ELIGIBLE for the reset |
| 580 | uncited, grade D — blind and INELIGIBLE; nothing repairs it |
| 584, 586 | lineage healed, grade still D — the stuck-D finding |

A single ratio is what hid this.

#### Two cautions for whoever fixes it (nofx-52's, both right)

- **Do not widen the predicate to `"D"`.** A genuinely uncited close *should* be
  D — 580 is exactly that. Key on lineage, not on the letter.
- **Backfill versus fix-forward needs a ruling.** A silent backfill moves a
  published grade distribution. Four rows is small enough to fix and large
  enough to notice.

Out of this wave's footprint (adherence belongs to the grader). Remediating
existing rows is a DB write and needs the owner's authorisation.

**Line-number note:** the reset reads `reconcile.go:588` in this branch and
`:576` in nofx-52's tree — same statement, and my branch adds lines above it.
Quoting the statement rather than the number is the durable citation:
`if p.Status == "CLOSED" && p.AdherenceGrade == "F"`.

### 9.4 The reason this took three sessions: the log line cannot tell them apart

```
⚡ armed fill S1 @ 29285.00: position row not materialized yet — stamp pending
```

That line prints whenever `pos == nil`, and `pos` is nil for **either** reason —
the row genuinely absent, or the lookup unable to match it. It asserts the first
as a fact. I quoted it as my "live proof" of mechanism 1 and it is not proof of
anything beyond `pos == nil`. A misdiagnosis compiled into a log line is
expensive: it sent me looking for a timing bug and hid a deterministic one.

### 9.5 The law, stated harder (nofx-89's wording)

A write on a branch almost nothing takes is not merely the equivalent of an
unperformed read — **it is worse, because it produces a green proof.** Row 35
would have looked like a pass had I asserted only "the stamp ran".

---

## 10. CLOSED — owner extended the footprint and ruled (2026-09-03)

Both open items are now implemented on this branch. §10.3's standing watch
stays open by design.

### 10.1 The reset predicate — keyed on the stamp

`RepairArmedLineage` keyed on `AdherenceGrade == "F"`, which is a SUBSET, not
an impossible value: an uncited close grades base D and steps to F under either
penalty. It succeeded on penalised uncited rows (566, 571) and failed on clean
ones (580).

It keys on **"lineage was just stamped on this row"** now, whatever letter the
row holds — if lineage was just written, the grade was computed without it and
is stale by construction. A row nothing stamps is untouched: **580 keeps its
earned D.** Pinned across D, F and C, plus the unstamped case.

### 10.2 Canonical casing at the write (class 28)

`UPPER(side)=UPPER(?)` fixed the read. `UpsertArm` now uppercases `side` where
the value **enters**, so `armed_orders` stops disagreeing with
`trader_positions` at the source. Existing lowercase rows keep working through
the fold-insensitive read.

### 10.3 The migration — FOUR rows, not seven

Flag-guarded (`ADHERENCE_REGRADE`, default OFF), backup first via the same
online `sqlite3 .backup` the C1 timer uses (**no backup, no write**), and
idempotent.

The ruling listed seven ids. Three do not qualify, and including them would
regrade a test seam and promote two closes that earned their D:

| id | why it must be left alone |
|---|---|
| 530 | cites the literal `off-plan` sentinel, `matched=0` — D is correct |
| 572 | `source=e7_farside_test`, `TEST-E7`, `plan_version 0` — not a trade |
| 582 | `matched=0` → base C, "direction mismatched", −1 penalty — D is correct |

Every clause of `stuckAdherenceWhere` exists because one of those rows proves
it. nofx-47 reached the same four independently and told the owner.

**It CLEARS the grade; it never writes one.** W5 regrades with the lineage in
hand — the same mechanism `RepairArmedLineage` uses, so there is no second
grader and no invented verdict. A cleared row is *ungraded* (excluded from
distributions) until W5 runs, which is honest rather than wrong.

#### Before / after, measured on a copy of the live store

```
pending ids : [575 584 586 591]
before      : A 30 · B 22 · C 5 · D 10 · F 4     (n=71)
after       : A 30 · B 22 · C 5 · D  6 · F 4     (4 cleared, pending recompute)
second run  : 0                                   (idempotent)
```

The boot line reports what is **pending** when the flag is off, so the count is
visible without arming anything:

```
🩹 adherence regraded=0 (ADHERENCE_REGRADE off) · 4 row(s) pending: full lineage
   still holding an off-plan D
```

### 10.4 STANDING WATCH — the row 582 could not be

Unchanged and deliberately open. The armed-path question is **untested, not
exonerated**. The discriminating observation is a future `armed_entry` close
with `plan_matched=1` and grade **D** — base A cannot reach D, so that row
would prove the ordering defect exists outside the reconcile path.

Guard any re-derivation with the lineage clause — see the 572 trap above.

### 10.5 DEV WAS RED — and the cause was mine (checklist class 60)

`TestMaybeWakePlannerOnLevelEventsThrottleDedupe` failed on `origin/dev` at
`62fd368d`, verified in a clean worktree of dev itself. I first reported it as
unrelated to this branch. **It is not unrelated — it is a consequence of the
class-47 cutoffs I built**, and I traced it after nofx-47 called it a time bomb.

The test injects a fixed clock into its FIXTURE and called the production entry
point, which reads `time.Now()`. Harmless while the cutoffs only WARNED. The
hour they began ENFORCING, the last-window cutoff began asking how many minutes
remain to the session flat:

```
10:20 CT  →  265 min to NY's 14:45 flat  →  wake proceeds  →  PASSES
14:44 CT  →    1 min                     →  SkipForCutoff  →  FAILS (empty key)
```

So the deploy lane's "27 ok / 0 FAIL" at `f478ed88` was **honest when taken**
and the identical command failed the same afternoon. Reproducible in one
direction, so re-running never surfaces it.

Two of us mis-diagnosed it first: nofx-47 blamed the fixture's bar dates (real,
but the 15m zone path gates on `FormedAtMs` vs plan birth, both
fixture-controlled), and reasoned from a stale timestamp believing it was 11:30
when it was 14:47.

**Fix — the clock seam.** `maybeWakePlannerOnLevelEvents` keeps `time.Now()`
and delegates to `maybeWakePlannerOnLevelEventsAt(now, …)`. Production is
unchanged; the test states its own clock. Recorded as **checklist class 60**,
because the general shape is bigger than one test.

#### The standing check nofx-47 asked for, run

Scanned every `*_test.go` that builds a fixed `time.Date` clock and calls an
`AutoTrader` entry point that reads `time.Now()` — eight sites. Then asked the
discriminating question: which of them reach the NEW time-sensitive logic?

```
minutesToSessionFlat / SkipForCutoff / SkipForCooldown
  → auto_trader_wake_levels.go:303, 316, 336  — and nowhere else in production
```

`latestClosedPrimaryBarMs`, `ForceReset`, `observeTransitionStanddown`,
`runPlannerReadWithTriggerClaimedCtx` and `maybeManageArmedOrders` all read the
clock for other reasons and consult **none** of the cadence inputs. So the blast
radius of the enforcing cutoffs is exactly the one path, and the one test that
reached it. **[A]** — grep over production sources, quoted above.

That is a bounded answer, not a clean bill of health: a future guard promoted to
ENFORCE puts its own inputs into the same position, which is why class 60's
probe is phrased around the promotion rather than around this test.
