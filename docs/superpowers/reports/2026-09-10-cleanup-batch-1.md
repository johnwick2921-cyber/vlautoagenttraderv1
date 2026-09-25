# CLEANUP BATCH 1 — everything recorded this week that does not move a trade

**Lane** `cleanup-batch-1-2bdef526/nofx-59[3f0200]` · **branch** `fix/cleanup-batch-1` ·
**claimed** 2026-09-10 17:04:30 CT · **running rev at accept** `adae3bb4` (verified from
`/api/health`, `/proc/2560377/exe` and the binary's own stamp).

**No trading behaviour changed** except B2, which the dispatch ordered by name: a cancel
that could never be re-requested now can be. Nothing authored, armed, placed, refused,
sized or exited differs.

## ONE LINE PER ITEM

| # | verdict | what |
|---|---|---|
| B1 | **CLOSED** | `SweepableArmStateSQL()` landed in `41023d0c`; `boot_sweep.go:38-40` already matches the SQL. No four-over-three discrepancy remains. |
| B2 | **FIXED** | The cancel budget is now per process. New column `cancel_attempts_boot`, and the cancel boot line says so. |
| B3 | **FIXED** | `api/handler_svp.go:50`'s "the cache cap" corrected; the value at `:63` untouched. |
| B4 | **FIXED** | Ten `agent/` files gofmt'd, `gofmt -l` 10 → 0. **The dispatch's proof method was wrong** — corrected below. |
| B5 | **FIXED** | `scripts/mutate.sh` + `scripts/mutate-selftest.sh` (6 pins). A verdict is refused unless the experiment demonstrably ran. |
| B6 | **FIXED** | The three-check worktree recipe added to `CLAUDE-canon.md` — **including that the check itself can be wrong.** |
| B7 | **CLOSED** | Does not reproduce. 58 files / 414 tests pass. **The dispatch's diagnosis does not hold** — detail below. |
| B8 | **CLOSED** | Already done by another lane in `29004429`, "two different rc 3s sat on adjacent lines". |
| B9 | **CLOSED** | Class 99's Law points at 107, "a single SOURCE, never a single PREDICATE". |
| B10 | **CLOSED** | The 11 is corrected to 10, with the cause named (a hardcoded `echo` above output that read 10). |
| B11 | **CLOSED** | Classes 88–91 exist in the numbered-list format; the canon contract test landed (16,531 bytes, passing); `CLAUDE.md:205` fixed and mirrored. |
| G2 | **OWED** | `web/src/brand-scope.test.ts` — the lock lane's. Not touched. See below. |

## THE THREE PREMISES THAT DID NOT SURVIVE MEASUREMENT (A17)

**B4 — `git diff -w --stat` can never read zero.** The dispatch asked for that as proof
the gofmt diff is whitespace-only. It is not achievable: gofmt also **reorders imports**
(`nofx/safe` moved into alphabetical position in `stock.go`) and **expands single-line
`if` bodies onto three lines**. Both are semantics-preserving; neither is whitespace, and
`-w` still shows them. The correct proof is that **`gofmt(before)` is byte-identical to
after**, run per file — **10 of 10 IDENTICAL**, which additionally proves no hand edit rode
along, something `-w` could not have shown either way.

**B7 — the 12-file vitest break does not reproduce, and the proposed fix would not have
worked.** CI runs `cd web && npm run test` (`.github/workflows/test.yml:54`). From `web/`:
**58 files, 414 tests, 0 failures**. The `server.fs.allow` line the dispatch specifies
governs the **vite dev server**, not vitest's transform, and the blamed import at
`web/src/constants/branding.ts:1` is still present and untouched while the suite passes.
The production build also succeeds. Failures DO appear when vitest is run from the **repo
root**, which is not how the project or CI runs it — and those are the `brand-scope.test.ts`
byte-for-byte guards, i.e. **G2, the lock lane's**, which Section E forbids this lane to
touch. Handed over, with the added detail that it surfaces only under a non-canonical
invocation.

**The split-arm red was NOT pre-existing — it was mine, and it was class 100.** The
dispatch says to state it as 104's and pre-existing. Measured: **4/4 PASS on `origin/dev`,
4/4 FAIL on this branch** — deterministic, not flaky. The cause was not any item above:
`git diff --stat origin/dev HEAD` showed **459 deletions in `kernel/levels_every_tf_test.go`
and 280 in 103's W-TF report**. This branch was not deleting their work; its **base was
older than dev**, and dev had moved **20 commits** since `580e88b3`, including 103's W-TF
merge and 104's fixture fix to `split_entry_test.go`. Rebasing onto `0975ef11` fixed the
test and reduced the diff to this lane's own 16 files, 76 deletions, every one explicable
(gofmt line restructuring, two replaced comment lines, four replaced struct/schema lines).

**This is the failure nofx-6d warned me about by name, four hours earlier**, and I still
walked into it: I read the deletion count once at merge time rather than **before every
push**, and I had pushed three times. The discipline only works at the cadence it specifies.

## B2 — THE ONE BEHAVIOUR CHANGE, AND WHY IT NEEDED A NEW COLUMN

`RequestCancel` bumped `cancel_attempts` monotonically, so a row that reached the cap
before a restart arrived at the new process still capped. `confirmPendingCancels`
(`trader/cancel_confirm.go:400`) then neither re-requested it nor promoted it — it logged
"attempt cap reached" and moved on, for the life of the ledger. The restart is precisely
the event that changes the facts the cap guards against: a new wire, a re-seeded book, a
broker that may now answer.

The budget is now counted against **`cancel_attempts_boot`**, a **new** column —
deliberately **not** the existing `boot_id`, which answers "which process AUTHORED this
row" (class 33). Two questions on one field is the failure this repo keeps meeting; they
get one field each. Empty on every historical row, which reads as "counted by a process
this one cannot identify", so the first request after this ships resets once — the correct
answer for attempts accumulated by a process that no longer exists.

```
RED    : row.CancelAttemptsBoot undefined (type ArmedOrderDB has no field) — build failed
GREEN  : TestCancelBudgetResetsAcrossBoot · TestCancelBudgetAtCapIsReRequestableAfterBoot
MUTANT : if row.CancelAttemptsBoot != ProcessBootID() {   →   if false {
         sed APPLIED (diff non-empty, +line quoted) · go build OK on the mutant · BOTH pins FAIL
```

## B5 — A HARNESS THAT CANNOT FAKE A VERDICT

Four "survivors" this week were not survivors: three seds never matched (class 89 — a
search string with three spaces where gofmt left one), and one mutant did not compile
while the harness read the absence of `--- FAIL` as a pass. Both inferred a verdict from
the **absence of a failure line** without establishing that the experiment ran.

`scripts/mutate.sh` proves two things before it will read the suite: the edit **landed**
(compared against its own backup, not against git, since the file may be dirty for
unrelated reasons) and the mutant **builds**. Verdicts are exit codes — `0 KILLED ·
1 SURVIVED · 2 NOT-APPLIED · 3 BUILD-FAILED`. A pattern selecting zero tests is also
NOT-APPLIED: a suite asked to run nothing cannot have failed to catch anything. The file
is restored on every exit path including a signal.

Proven on the real thing, not only the fixture: B2's mutation reports **KILLED**; the same
mutation with two spaces instead of one reports **NOT-APPLIED** and names the cause.

## B6 — AND THE CHECK ITSELF CAN BE WRONG

The recipe is three checks because each catches what the others cannot: the exit code
catches a refused add, `rev-parse --is-inside-work-tree` catches a directory that exists
but is not a worktree, `pwd -P` catches a `cd` that landed elsewhere. **This lane wrote
`[ -d "$W/.git" ]` today and it reported failure on a healthy worktree** — in a worktree
`.git` is a **file** containing a `gitdir:` pointer, not a directory. Correct-looking,
green in the author's head, failed closed on a good tree. Ask git; do not guess at git's
layout.

## A15 — WHAT IS STILL WRONG

- **G2 is open and is the lock lane's**: `web/src/brand-scope.test.ts` fails under a
  repo-root vitest invocation. Not touched, per Section E.
- **`api/handler_svp.go:63` still hard-codes `2000`** where it should import
  `kernel.AISVPBarCount`. The dispatch said comment only; the retyped constant is recorded,
  not fixed.
- **This report cites class 88 for the second-writer failure in one commit message**
  (`5a65caac`). `2571a6c5` reclassified it as **class 102**. The commit stands; the number
  is corrected here rather than by rewriting history.
- `CLAUDE.md` instances 2 and 3 of class 105 are untracked and outside a branch's reach.
  Instance 2 (line 205) the owner fixed by hand. **Instance 3 is open**: the SPEC-FRESHNESS
  block cites "class 73", slot 73 is a different class, and SPEC-FRESHNESS has no checklist
  slot at all. Owner-only.

## THE BOOT LINE (owner's instruction: say the change is inert)

Extended the EXISTING `CancelBootLine` rather than adding a competing line:

```
cancels: confirm=broker-snapshot · pending=0 · unconfirmed=0 ·
slot-guard=on(refuse-on-live|stale) · timeout=1m30s · stale-bound=1m0s ·
rerequest-cap=5 budget=per-process carry=UNKNOWN
(inert until a cancel is re-requested after a restart) · reconciled=n/a (no broker book yet)
```

`budget=per-process` is the clause that matters: a reader seeing `rerequest-cap=5`
alone cannot tell a per-row-forever budget (the defect) from a per-process one (the
fix) — both render identically. `carry=` is the MEASURED count of rows whose attempts
a departed process tallied, i.e. exactly those whose budget resets on their next
request; a ledger that cannot be read prints `UNKNOWN`, never 0 (A24). Every field is
read from the enforcing code (A11): the cap from `cancelReRequestMax()`, the carry from
`CountForeignBootCancelAttempts()`.

Pinned by `TestCancelBootLineStatesThePerProcessBudget` and
`TestCancelBootLineCountsTheForeignBootCarry` (carry=0 on an empty ledger, carry=1 with
one foreign-boot row). **Mutated through `scripts/mutate.sh` — B5's first real use:**
removing the clause reports KILLED with the sed confirmed applied and the mutant
building.

## NEW CLASS: 114 — THE VERIFIER IS WRONG, AND ITS WRONGNESS READS AS A RESULT

Taken at merge after a census across BOTH formats (highest occupied 113; not
reserved in advance, per the dispatch). Three instances, all from this wave: the
`git diff -w` proof that can never pass, the `[ -d "$W/.git" ]` guard that fails
closed on a healthy worktree, and the mutation verdict inferred from an absence.
It is the general case of class 89 and the mirror of class 105 — 105 is prose
drifting from code, 114 is a *check* drifting from what it checks. Both stay
invisible because the artifact keeps producing a believable answer.

**Class 100 hit this wave TWICE in one hour**, despite nofx-6d warning me by name
four hours earlier. First: dev moved 20 commits under a base of `580e88b3`, and
the branch showed 459 deletions in `kernel/levels_every_tf_test.go` — that was the
split-arm red, not 104's fixture. Second: dev gained class 113 in the minute
between a rebase and a push, and the branch showed 0 insertions / 67 deletions in
the checklist. Neither reached dev. The remedy is sharper than "check before every
push": **rebase and re-check immediately before the merge**, because dev moves in
minutes, and a check run even a minute early is a check run against the wrong tree.

## CUTOVER

**Go files changed** (`store/armed_orders.go`), so this is boot-eligible per Section D —
but the change is inert until a cancel is re-requested after a restart. Merge waits on
103's W-TF landing, per the dispatch. Suites at the rebased head `cf62e965`: **Go 31
packages exit 0** (split-arm passing), **tsc 0**, **vitest 58 files / 414 tests**.
