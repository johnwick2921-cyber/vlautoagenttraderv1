# W1 — THE EPISODE CONTRACT

**Branch** `fix/episode-contract` · **base** `origin/dev` @ `757eb578` · merged, not rebased (see §F.6)
**Scope (A31)** RECORDING ONLY. No rule, threshold, gate, order, plan content, level
score or surface behaviour changed. The wave defines the unit of opportunity and
writes it down.

---

## THE HEADLINE: THE BACKFILL RECOMPUTES ZERO ROWS

The dispatch's premise was that history could be re-read into the new unit. It
cannot. Measured against `data/data.db` on 2026-09-10, read-only:

| state | rows |
|---|---|
| in-era rows examined (since `DayPlanEraStart` = 2026-08-15 00:00 CT) | **4,860** |
| `untouched_pre_era` | 0 |
| `unrecomputable:no_formation` | 4,677 |
| `unrecomputable:no_scenario_link` | 183 |
| **RECOMPUTED** | **0** |

`formed_at_ms` is present on **183 of 4,860 rows — 3.77%**. The research that
motivated this wave claimed 26%. That is not a rounding disagreement; it is a
different claim about the archive, and the archive settles it.

The two gates compose to nothing by construction: the 183 rows that survive the
formation gate are *exactly* the rows that then fail the scenario-link gate,
because **no historical row carries a scenario link at all** — the column is born
in this wave. So the ceiling on any retrospective episode study of the current
archive is zero rows. Not "few". Zero.

**This is the research's claim MEASURED, not the backfill failing.** The backfill
is working correctly: it marks every row it cannot recompute, with the reason, and
invents nothing. A backfill that had "recovered" 4,860 episodes would have been
fabricating 4,677 formation times and 4,860 scenario links.

**What it means for the next wave.** Episode-based evaluation starts from rows
written *after* this boot. There is no historical baseline to compare against, and
any experiment that assumes one is measuring its own reconstruction.

---

## A · THE UNIT

A touch was always the row. What was missing was the ladder above it —
`store/opportunity_outcome.go`:

```
never_reached → reached_declined → confirmed_not_armed → armed_not_filled → filled
```

`OpportunityOutcomeFor` derives the rung from four booleans in a single switch,
highest rung first. **Exhaustive by construction, not by a default branch**: there
is no `default:` that silently absorbs a state nobody enumerated. Adding a rung
means adding a case, and a missing case is a compile-visible gap rather than a
row quietly filed under the lowest rung.

`store/opportunity_close.go` selects on `opportunity_outcome IS NULL` — idempotent
by *predicate*, not by a "done" flag that can drift from the rows it describes.
`factsFor` is called **per row** (`opportunity_close.go:48`); a batch cannot smear
one row's facts across its neighbours.

## B · THE LINK IS A HEURISTIC AND THE COLUMN SAYS SO

Ruling (b) as dispatched. A touch has a *price*; nothing in the plan path stamps a
scenario id on it. So the link is recorded as what it is:

- column `scenario_nearest` — **never** `scenario`, which would read as fact
- `ScenarioLinkBasis = price_proximity`, plus distance in **points and in Δ**
- two anchors inside the band → **NULL**, basis `unresolved:two_scenarios_within_band`
- nothing close → `unresolved:nearest_outside_band`
- no plan at the seat → `unresolved:no_scenario_at_seat`

Ambiguity is recorded as ambiguity. There is no nearest-wins path: a tie does not
become a fact because a tie is inconvenient. A zero Δ leaves `dist_delta` NULL
(`scenario_link.go:90` guards `delta > 0`) rather than dividing.

The band is `kernel.LevelClusterTicks × 0.25` = 3.00 pts — **the map's own cluster
width**, not a new tolerance knob. Two levels the map would have merged are exactly
the two this link refuses to choose between.

`trader/scenario_anchor.go` takes `Confirm.RefPrice` when present, falls back to
`Arm.Entry`, and **counts what it cannot anchor** rather than dropping it silently.
`trader/auto_trader_planner.go` now parses the `latest.Doc` it previously fetched
and discarded — only `.Version` was ever read from it.

**The identity wave comes after.** This is the labelled stand-in, and the label is
the point: a downstream reader can tell a heuristic from a fact without asking
anyone.

## C · THE ATTAINABLE ENTRY — MEASURED BEFORE ASSUMED

`store/attainable_entry.go`, in precedence order:

| basis | meaning |
|---|---|
| `observed_fill` | MEASURED — a fill exists |
| `resting_limit:assumed_fill_at_entry` | ASSUMED, and says so |
| `stop_entry:assumed_fill_at_trigger` | ASSUMED, and says so |
| `first_tradeable_after_confirm` | ASSUMED, and says so |
| `none:never_confirmed_never_armed` | there was no entry to attain |
| `not_captured:confirmed_but_no_price_recorded` | there was one; we failed to record it |

The last two are deliberately **different values**. "No entry existed" and "an entry
existed and we lost it" are different facts about the system, and collapsing them
would hide a recording defect inside a legitimate outcome.

**The level price is never a branch.** What the level *said* is not what the tape
*offered*, and no rung falls back to the level price to avoid a NULL.

## D · WHAT IS NOT DONE, DELIBERATELY

- **No scenario identity on the touch.** Ruled to the next wave.
- **No episode table.** The ruling was to extend `touch_outcomes`; a second table
  would have been a second key for the same unit — and a second key is how the
  system ends up with two readers that happen to agree (class 97).
- **No historical terms.** `armed_orders` is a state row **mutated in place**, not
  an event log. Historical terms are unrecomputable by construction and are marked
  `unrecomputable:terms_mutated_in_place` rather than reconstructed from
  `updated_at` — which would have produced a number for every row and a true one
  for none.
- **`trade_excursions` not extended.** It is `UNIQUE(position_id)` — **filled-only
  by construction**. It cannot hold the four rungs below `filled`, which are the
  rungs the wave exists to measure. Wrong key, not a missing column.
- **The ordinal was already true** and was dropped rather than re-implemented.

## E · THE BOOT LINE NAMES ITS RESOLVER

Join key `🎫 episodes:`. Counts are READ, never literal.

```
🎫 episodes: open=2 · closed today=7 (never_reached=3 reached_declined=2
   confirmed_not_armed=1 armed_not_filled=1 filled=0) · backfill recomputed=0
   unrecomputable=4860 · k=3[I] Δ=resolved-per-read
   (kernel.MeanAbsIncrement, the tape's own scale) H=12[I]
```

Δ prints its **resolver**, not a value. Δ is resolved per read; a number printed
once at boot is a literal wearing a measurement's clothes, and would go stale
silently between the boot line and the first decision. `k` and `H` print `[I]` to
mark their env source.

**One honest seam, stated rather than buried:** `store` cannot import `kernel`
(kernel imports store). So `store/episode_detector_scope.go` reads `DETECTOR_K`
and `DETECTOR_HORIZON_BARS` through the **identical env names**. That is two
readers of one source — the closest this dependency direction allows — and
`grep DETECTOR_K` finds both sites, so a drift between them is visible rather
than silent. It is not the single-reader ideal of class 97 and is not claimed to be.

## F · CORRECTIONS TO MY OWN PREMISES

1. **The C1 cross product.** I claimed a join existed, on 1,205 rows. There was no
   join: it was 481 touches × 9 arms × 7 positions. Both the wrong measurement and
   the correction are on the record, because the wrong one is what the method
   produces if nobody checks the row count against the inputs.
2. **`touch_outcomes` has ONE writer**, at `detector_record.go:102`. My earlier
   report said two; the second site (`:115`) writes `candidate_pool`.
3. **`formed_at_ms` is 3.8%, not 26%** — see the headline.
4. **My own SYSTEM-MAP draft overstated the schema.** It claimed *every* added
   column is a pointer. `ScenarioLinkBasis` is a plain `string`, because
   `ResolveScenarioLink` always returns a basis — so an empty basis marks a
   **pre-wave row**, not a missing reading. Caught by grepping my own prose against
   the code before committing. This is class 105's lesson applied to the commit
   that introduced class 105.
5. **My classes 105/106 landed as 109/111 — renumbered four times, across five
   dev tips in one day.** 105 went to dispatch 103; 106 and 107 went to a peer's
   generalisation of class 104 and to the boot-sweep wave; then 108 was contested
   *three ways at once* — the arm-state lane's (merged, so it held), nofx-b3's,
   and mine. The merged one wins and both unmerged ones move, which needs no
   adjudication because merge order already decided it. 109 was uncontested and
   stayed. That is A27 working exactly as written, and class 109 carries it as
   the worked example: a census tells you the ceiling, only the merge assigns the
   number. The alternative — reserving at accept — is what produced the
   75/76/77/92/93 duplicates. I also messaged nofx-b3 before taking 111, since
   coordination is cheaper than a fifth renumber.

6. **This branch is merged onto dev, not rebased, and that was forced.** Four
   rebases in, a force-push was refused by the classifier. I reconciled with a
   `-s ours` merge after verifying the tree diff was additive-only (2,262
   insertions, 11 deletions, no file lost). That merge re-parented the superseded
   pre-rebase commits, so the NEXT `git rebase` tried to replay all of them and
   conflicted on files it had already resolved. I aborted it and merged dev in
   instead. The branch therefore carries both renderings of its own history. It is
   uglier than a clean rebase and it is on the record here rather than hidden: the
   tree is correct, the duplicated commits are content-identical, and no work was
   lost — but a reviewer reading `git log` will see each early commit twice, and
   should read the merge commits for why.

## G · FOUR DEFECTS FOUND, NONE MINE, TWO FIXED WITHIN THE HOUR

Found while verifying my Guide change. **Filed, not fixed — A31 scopes this wave,
and both belong to the rebrand lane.**

### G1 — the FE suite is red at the version the lockfile pins, green at the one the main tree has

**This section was wrong in the first two drafts and is corrected here.** I wrote
that the suite "fails everywhere, main tree included". It does not. A peer
(nofx-8e) measured it GREEN in the main tree and challenged the finding. They were
right about the observation and wrong about the cause; I was right about the
observation in my tree and wrong about the cause. Neither of us was measuring
badly. **We were running different versions of vite.**

**The controlled experiment** — same worktree, same commit, same config, same test
file, only the vite version swapped, `--no-save` so nothing else moved:

| vite | result |
|---|---|
| **6.4.1** | `Test Files 1 passed · Tests 9 passed` |
| **6.4.3** | `Test Files 1 failed · Tests 9 failed` — `Error: Denied ID /…/branding/product.txt?raw` |

That is causation, not correlation: one variable, both directions, reproduced.

**Which version is the repo's?** `web/package-lock.json` is TRACKED and pins
**6.4.3** — the failing one. `web/package.json` declares `"vite": "^6.0.7"`, so the
caret range admits both. The main tree's `node_modules` holds **6.4.1**: a stale
install that predates the lockfile move and has not been reinstalled since.

So the correct statement is the inverse of my first draft:

- **The suite is red at the dependency state this repo actually declares.**
- The main tree's green is an artifact of an install that no longer matches the
  lockfile.
- Anyone running `npm ci` — a fresh clone, a new lane's worktree, the A4
  clean-clone deploy path — gets 6.4.3 and gets 12 red files.

**Mechanism**, now measured rather than assumed. `web/src/constants/branding.ts`
(added by `7d7be486`) does `import productName from '../../../branding/product.txt?raw'`.
Vite's workspace root here is `web/`, confirmed by asking vite itself:

```
searchForWorkspaceRoot('/…/web') = /…/web
```

because the ancestry carries no root marker — no repo-root `package.json`,
`pnpm-workspace.yaml` or `lerna.json` in any tree (and `.git` is commented out of
vite's `ROOT_FILES`, so the worktree-vs-clone distinction I first reached for is
irrelevant). `../branding/` is therefore outside the allow root. 6.4.3 enforces
that on this import path; 6.4.1 did not.

**Blast radius:** 12 test files fail to load. Zero assertion failures. And the
count that matters: **354 tests collected red versus 414 green — 60 tests silently
do not exist**, while the file total reads 58 either way.

`npm run build` is **unaffected at both versions** — rollup does not apply
`server.fs.allow`. Verified at 6.4.3: `✓ built in 4.51s`. **The cutover is not
blocked**, and a build-only CI would never see any of this.

**Fix** is one line in `web/vitest.config.ts`:

```ts
server: { fs: { allow: ['..'] } },
```

Verified at 6.4.3, then reverted: 12 red files → 1, 344 → 413 passing.

**The finding underneath the finding.** Two lanes ran "the suite" on the same
commit, got opposite answers, and each correctly believed their own measurement.
A caret range plus a tracked lockfile plus long-lived `node_modules` directories
means **"the suite passes" is not a property of a commit** — it is a property of a
commit *and* whenever someone last ran install. Neither number is on the record
anywhere. My first draft asserted a defect in another lane's file on the strength
of a measurement whose environment I had not pinned, which is the same error in
the opposite direction.

### G2 — a tamper-guard that was red and legible for 13 hours in a suite nobody on that wave ran

With G1 unblocked, a 13th failure surfaces that had been invisible:

```
FAIL src/brand-scope.test.ts > preserves deploy/nofx-lock.sh byte for byte
  Dispatch 102 protected file changed: deploy/nofx-lock.sh
```

`web/src/test/brand-scope-baseline.json` pins sha256 of 16 protected files.
**Exactly one has drifted** — so the guard is otherwise doing its job precisely,
and would have caught this on the first commit had it been able to run.

The timeline is the finding, and the order of the two events is the whole point:

| when (CT) | commit | what |
|---|---|---|
| 09-08 18:57 | `66e2c09a` | baseline written. Pin `bcd82c52…` **correct**. Guard GREEN. |
| **09-09 13:33** | **`7d7be486`** | **G1 breaks the runner. The guard is still GREEN at this moment.** |
| 09-09 22:12 | `87ef772d` | lock keeper wave — sha becomes `9bacd04a…`. **Guard goes RED. Nobody hears.** |
| 09-10 07:16 | `963ea975` | keeper process-group fix. Still red, still unheard. |
| 09-10 08:05 | `88d40920` | keeper `/proc` read fix. |
| 09-10 10:26 | `97a6525c` | `/proc` read SIGTERM fix. |
| 09-10 11:43 | `ace51598` | INCOMPLETE-lock wave. |

Current sha256 at dev tip `757eb578`: `46fcbf76…` — against a pin of `bcd82c52…`.

**Corrected after nofx-8e's challenge, and the correction is against my own
framing.** I first wrote that G1 had blinded this guard. In the environment where
8e actually worked — the main tree, vite 6.4.1 — **the guard ran fine and was
plainly RED, naming their file, for the whole 13h31m.** It was not muted there. It
was legible and unread, because that wave ran Go and the lock suite every time and
treated those as "the suite". 8e states this plainly as their own miss, and it is
the more useful reading: a protected-file guard lived in a suite the wave had
decided, without ever deciding, was not theirs.

Both things are true at once, and which one you hit depends on your installed vite:

- at **6.4.1** (main tree): the guard **runs and is red** — a legible signal nobody read
- at **6.4.3** (lockfile-pinned): the guard's file **never loads** — no signal to read

The second is strictly worse, and it is the state anyone gets from a fresh install.

None of the six commits is at fault. Each changed a protected file for good reason,
and the guard exists precisely so a human ratifies that change by updating the
baseline. **The pin has since been ratified** at `e79bf298` by another lane, and 8e
fast-forwarded the main tree to `a8b66cd0` and re-ran: 58 files, 414 tests, green.
I did not touch their baseline and should not have — ratifying another lane's
protected-file change is exactly the conversation the guard exists to force.

8e also checked what my report had not: whether the unread window hid drift in any
**other** protected file. It did not — 1 of 16 drifted, and it was theirs.

### G3 — dev's Go suite was red every day between 12:00 and 13:30 CT (FIXED, verified in-band)

Found by running my own verification at 12:11 CT instead of 11:58. **Eight tests
fail; all eight cite one line:**

```
🛑 arm REFUSED (session risk): no_trade_band: lunch no-trade window (12:00–13:30 CT)
```

    TestArmedOrderUpsertAndGateRR          TestLiveConditionPlacesOnLoopback
    TestShadowDemotionAuthorsInertRow      TestShadowedRestingOrderCancelledAtBoot
    TestShadowDemotionE8WritesCounterfactual   TestConfigFlipToLiveAllowsArming
    TestShadowDemotionNoWireFrameOnLoopback    TestSplitArmWritesTwoLedgerRows

**Not my merge.** Reproduced on `origin/dev` @ `43c0890f` in a clean worktree with
none of my commits: same test, same failure, same lunch refusal. I touch none of
`armed_executor`, `shadow_demotion`, `split_entry`, `session_risk` or the no-trade
band. The same suite was green at 11:26 and 11:58 today on nearly the same tree.
**The clock crossed 12:00.**

**The seam is not missing — the tests bypass it.** `sessionRiskGateAt(now)` takes
its clock as an argument and production passes one correctly from
`armed_executor.go:330`. The failing tests do `now := time.Now()` and hand the real
wall clock to a correctly-seamed rule.

That is class 60 surviving its own fix. `clock-seams.list` exists, the lint at
`trader/clock_seam_lint_test.go` reads it, and the list's header states the class
in one line:

> *a suite verified at 11:00 was red at 14:50 because a fixed-fixture test reached
> a wall-clock entry point. The entry owns the clock; the rule takes it as an
> argument.*

That is this defect, described in the file meant to prevent it, eight months of
waves ago. **The lint asserts the seam EXISTS; nothing asserts the tests USE it.**
A seam only the production path honours is half a seam.

**This is class 110 with a different variable.** 8e filed "a green suite is a claim
about an environment, not about a commit" an hour ago, with an installed package
version as the variable. Here the variable is the **wall clock**, and it needs no
divergent install: one machine, one commit, one lane, green at 11:58 and red at
12:11. Two independent instances of one class inside one afternoon is the argument
for the class.

**Owed and now paid.** "The wall-clock entry-point sweep in checklist 60" has been
on my owed list since the class-52 wave. It was not theoretical.

**FIXED on dev at `bd295804`** by nofx-8e, who owns the session-risk rule and whose
own two verification runs today (11:26, 11:58) both happened to land before noon.
`armTestClock` searches the registry for a moment inside an enabled session and
outside every no-trade sub-window — searched, not hard-coded, because the windows
are configuration and a constant would be this defect with a longer fuse. The lint
gains a third half: tests must USE a seam, not merely have one.

**Verified independently here, INSIDE the band** — the only window in which the fix
is falsifiable at all. Merged at `bd295804`, run 12:32–12:34 CDT: the eight tests
pass, the full suite is **31 ok / 0 fail**, and all three seam lints are green
including the new `TestArmEntryPointIsNotCalledFromTests`. After 13:30 none of that
could have been checked until the next day.

**Still owed, and recorded by 8e rather than left in their head:** the general lint
finds 35 sites and most are false positives, because `clock-seams.list` contains
entries named `Save` and `observe` and a textual `.Save(` cannot distinguish
`at.Save(` from `db.Save(` without resolving the receiver's type. The shipped check
is therefore scoped to `maybeManageArmedOrders`; a receiver-aware version is OWED.
**Until it exists, a time-banded rule added to any OTHER seamed entry point
reintroduces this outage with nothing failing.**

**Acted on in this wave.** My three recorder pins in
`trader/scenario_link_wiring_test.go` handed `time.Now()` to
`recordDetectorOutputs`. That path reaches no banded rule today — `validityFor`
checks only `formedAtMs > 0` and recording is unconditional — so they passed at any
hour by luck, not design, and the scoped lint would not have caught a band added
there later. They now take a fixed clock. `armTestClock` does not fit (it needs an
`*AutoTrader` the fixture has not built yet) and is not needed: for a path with no
session dependence, a fixed moment is strictly more deterministic than a searched
one.

**Consequence for this wave's cutover.** My 14:45 window is outside the band, so
the pre-cutover suite will be green — but that green is *itself* an environment
claim, which is why §H records the clock alongside the versions. Filed, not fixed:
A31, and the fix belongs with whoever owns the arm-path tests.

### G4 — a wall-clock flake I diagnosed wrongly, corrected by its owner

**FIXED on dev at `7871a1d2`. My diagnosis of it was wrong and the correction is
the useful part.**

I reported `TestSplitArmWritesTwoLedgerRows` as order-or-state interference
inside the package, on the evidence that it passed standalone six times and
failed when the package ran whole. nofx-8e then reproduced it **standalone** at
13:20 on the same commit. There was no interference. It was the wall clock the
entire time — and I had the disproof in my own message: one standalone RED at
13:09:00 against six GREENs from 13:09:40. I filed that flip as "a timing
component on top" of an interference story instead of as the story.

**Two independent causes**, both introduced by that morning's `armTestClock`
fix, and both found by 8e:

1. **Five-minute bucket alignment.** The leg-2 confirm needs the last five 1m
   bars to form a COMPLETE 5m bucket below the ref. Bucket boundaries are
   absolute, so the tape's meaning depends on `now` **modulo 5 minutes**, and a
   clock searched forward from `time.Now()` walks that modulus through the day.
   Identical code passed 12:43–12:59 and failed at 13:20.
2. **The trade date.** A clock based on a fixed past date (2026-08-18) appended
   the plan for `PlanChainTradeDate(sess, now)` while the arm path resolves the
   CURRENT trade date — a plan written for a date nothing looks for. The arm
   produced zero rows **and not one refusal log**. That silence is the tell: a
   refused arm says why; a missing plan says nothing at all.

The fix is both halves — today's date at a fixed 5m-aligned hour, via
`armTestClockFrom(t, at, base)`, with `armTestClock` keeping `time.Now()` for
tests whose only binding constraint is being inside a session.

**What I got right, and it was the useful half:** I flagged "got 0, not got 1".
The earlier defect in this same test was one-leg-of-two from a 25-minute skew;
zero legs means the arm never happened. Same costume, different mechanism. 8e
says filing it as a recurrence would have cost them ten minutes in the split
logic.

**What my bisect actually measured.** Testing `b11659ea`, `c11632c3`, `5e273442`
and `a98a92c7` returned green at all four — not because the commits were green
but because each happened to be tested at a passing moment. **A bisect that
varies the commit while the real variable is time is a measurement of when you
ran it.** I concluded "the commit axis says nothing, the in-package axis says
everything" when the honest conclusion was "neither axis is the variable and I
have not found it yet."

Verified green here at the merged HEAD, **run 13:25:30–13:28:34 CDT, inside the
12:00–13:30 band**: 31 packages ok, 0 fail. Run in-band deliberately, because
outside it the result would not have been falsifiable.

## H · VERIFICATION

**The environment these results were measured in**, because §G is the proof that a
suite result without one is not falsifiable — the same standard A21 sets for row
claims, applied to suite claims:

| | |
|---|---|
| tree | `/home/hoang/nofx-episode` (linked worktree), `npm ci` from the tracked lockfile |
| go | `go1.25.3` |
| node / npm | `v22.22.1` / `10.9.4` |
| **vite** | **6.4.3** — matches `web/package-lock.json`; `npm ls vite` agrees |
| vitest | `4.1.11` |
| sqlite3 | `3.45.1` |
| **wall clock** | the 12:00–13:30 CT hazard is FIXED at `bd295804`; final results below were taken 12:32–12:34 CDT **inside** the former band, deliberately (§G3) |

The main tree measures the FE suite differently at vite `6.4.1`. Any suite result
below is a claim about THIS table, not about the commit alone.


| gate | result |
|---|---|
| `go build ./...` | OK at merged HEAD |
| `go vet ./store/... ./trader/... ./kernel/...` | OK |
| `go test ./...` | **31 ok / 0 FAIL**, run 13:25:30–13:28:34 CDT — deliberately INSIDE the 12:00–13:30 band, where §G3 and §G4 are falsifiable |
| `npx tsc --noEmit` | OK |
| `npm run build` | OK — 4.60s |
| `npx vitest run` @ vite 6.4.3 (lockfile) | 12 files red from G1; 354 collected |
| `npx vitest run` @ vite 6.4.1 (main tree's stale install) | green — see §G1 |

**My Guide change is not verified by the suite at the lockfile-pinned vite**,
because 12 files including the guide's do not load there — G1, not my change. Verified instead by: `tsc --noEmit` clean, `npm run build` clean,
and the guide tests passing under the temporary G1 unblock before I reverted it.
Stated here rather than reported as green.

dev moved **five times** under this branch during the wave (`557494c7` →
`c16a182d` → `cefcf08d` → `33e6d008` → `757eb578`); the suite was re-run at each
merged HEAD rather than carried forward — a branch green alone is not green merged.


## I · CLASSES FILED

- **111 — THE UNIT AN EXPERIMENT NEEDS, WHICH THE RECORD NEVER HELD.** Every
  experiment measures value per opportunity; the record held only fills.
- **109 — A CENSUS THAT CANNOT SEE ITS OWN THIRD FORMAT.** The checklist has three
  entry shapes; a two-format census reported the ceiling as 93 while 104 existed.
  Now carries the 105→106→108→111 renumber chain as its worked example, including the three-way contest on 108 that merge order settled without adjudication.

**Two more classes are owed from §G and are NOT filed here**, because each belongs
to a lane that owns the file and because I got the first one wrong twice before
measuring it properly:

- **A suite's result is a property of a commit AND an install date.** A caret
  range plus a tracked lockfile plus long-lived `node_modules` means two lanes can
  run "the suite" on one commit and get opposite answers, both honestly. Neither
  the installed versions nor the install date appear in any report. The remedy is
  cheap: print the resolved version of the runner alongside the pass count, and
  fail when `node_modules` disagrees with the lockfile.
- **Pin the suite's own file and test counts.** Red here reads
  `12 failed | 46 passed (58)` and green reads `58 passed (58)` — but the test
  totals are **354 versus 414**. Sixty tests vanish and no number in the default
  output says so. This is the remedy for the general shape I sent 8e and which
  survives all the corrections above: *a test that cannot run reports the same
  colour as a test that passes.* 8e is taking it into the settlement wave's pins,
  and notes it is the same shape as their class 103 — a fallback producing a
  plausible value so the failure behind it stays invisible.

## J · THE FIRST BOOT SHIPPED ONE OF FOUR ITEMS WIRED

`95f387ae` booted cleanly at 12:49:54 CDT — integrity OK, goldens PASS, five
references agreeing, gate five-for-five, sweep 0/0. And it shipped **one quarter
of this wave**.

The `🎫 episodes:` line never printed. That absence was the only symptom, and it
is the reason the defect was found at all.

| function | production call sites at `95f387ae` |
|---|---|
| `ResolveScenarioLink` | **1** — `trader/detector_record.go` |
| `CloseOpenOpportunities` | **0** |
| `ResolveAttainableEntry` | **0** |
| `BackfillOpportunities` | **0** |
| `EpisodeBootLine` | **0** |

Three of four items were built, unit-tested, described in this report, and called
by nobody. What *was* live is item 1, the one the owner ruled load-bearing: 55
rows carry `scenario_link_basis`, a column that did not exist before that boot,
so those rows are proof the recorder runs. `opportunity_outcome` and
`close_cause` were NULL on all 4,915 rows, because the closer and the backfill
were never reached.

### Why the suite did not catch it

**Because the check that catches exactly this already existed and I registered
one item in it.**

`trader/wiring_gate_test.go` is the A29 standing gate. It walks the whole repo —
`store/` included — and fails when a function claiming a production call path has
zero production callers. It was written for three instances of this shape inside
24 hours in September, one of which was `detector_record.go` declaring itself
"THE PRODUCTION CALL PATH" while nothing called it.

This wave added exactly two names to it: `scenarioAnchorsFrom` and
`scenarioLinkBand` — item 1's. The other three items were never registered, so
the gate had nothing to check and reported green. **The gate was not at fault;
the registration was.**

That is the whole answer, and it is worse than "the suite was too weak". The
suite contained a purpose-built detector for this exact defect, and the defect
walked past it because I only pointed it at the item I happened to be thinking
about while writing the pin. Every unit test in the wave passed, because every
unit test proved a function works when called — which is precisely the thing A29
exists to say is not enough.

Two smaller contributors, neither exculpatory:

- **The suite cannot boot the binary.** Nothing in `go test ./...` renders the
  boot block, so a missing boot line is invisible to it. The absence was visible
  in 12 seconds of reading the boot log, which is why the pre-cutover protocol
  has a boot checklist at all.
- **I read the wave's own report as evidence.** §A–§E describe four working
  items. They describe the code correctly and say nothing about whether anything
  calls it, and I did not distinguish those two claims when checking my own work.

### The fix, and one finding inside it

All four call sites now exist, each pinned, each **mutation-verified**:

| mutation | result |
|---|---|
| remove the closer's call site | RED — `closeEpisodesForSessionClose` |
| remove the backfill call | RED — `BackfillOpportunities` |
| remove the boot line call | RED — `EpisodeBootLine` |
| remove the attainable call | RED — `ResolveAttainableEntry` |

**The first mutation SURVIVED on the first attempt, and that is a finding about
the gate itself.** Deleting the only production call to
`closeEpisodesForSessionClose` left `CloseOpenOpportunities` still counted as
called — by the wrapper that had just become dead code. The gate counts a call
made from an unreachable function as wiring, so **an unwired wrapper satisfies it
for everything inside it**. Registering only the inner function pins nothing.
Whenever a call site is a wrapper, the wrapper is the name that must be listed;
that is now stated in the gate's own comment beside the registration.

### One deviation from the order, stated rather than made quietly

The instruction was to wire the attainable entry **into the recorder**. It is
wired into the **closer** instead. At the moment a touch is recorded it has not
confirmed, armed or filled, so resolving it there would stamp
`none:never_confirmed_never_armed` onto an episode that is still open — a
fabricated fact, and exactly the kind this wave exists to avoid. The
observations it needs exist at close, and it is computed there from the SAME
facts that decide the outcome, so the two can never disagree about whether an
entry existed.

### A limitation this fix does not remove

The closer's facts come from the scenario's observed confirm/arm state, never
from `armed_orders` — that row is mutated in place, and under the settlement
wave a timed-out cancel now rests at `cancel_pending` rather than reaching
`cancelled`, so a state read would be reading a value deliberately not yet final
(flagged by nofx-8e before it could distort a count). But the touch → scenario
link is still a price-proximity heuristic and is NULL whenever two levels sit
inside the band or nothing is close. **A row with a NULL link closes as
`reached_declined`** — correct for a touch nothing was armed at, and not yet
distinguishable from one whose arm this wave cannot see. Until the identity wave
lands, the honest reading of `reached_declined` is "no arm was linked to this
touch", not "no arm existed".
