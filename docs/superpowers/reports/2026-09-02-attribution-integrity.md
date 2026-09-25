# ATTRIBUTION INTEGRITY — one sentinel, one armed-under version

Branch `fix/attribution-integrity` · checklist class **52** (highest occupied at
merge: 51). Status: **BUILT, GREEN, STAGED — awaiting owner GO.**

## MEASURE FIRST — the dispatch's premises, checked before building

The wave was dispatched to rebuild position→plan attribution on a belief that
the join is blind for 25% of a week and that reconcile-materialized rows can
never be stamped. Measured against the live DB first (A23):

**Closed positions since the day-plan era began (2026-08-15):**

| source | n | no plan link | sentinel |
|---|---|---|---|
| `system` | 51 | **0** | 4 |
| `armed_entry` | 5 | **0** | 0 |
| `reconcile` | 11 | **3** | 0 |
| `e7_farside_test` | 3 | 2 | 0 |

September to date: **10 closed rows, zero unlinked.**

| dispatch claim | measured |
|---|---|
| C1 "ALL source=reconcile with empty plan fields" | **False** — 8 of 11 reconcile rows carry full lineage |
| C1 "25% of last 7 days, n=20" | 27 closed in 7d, 5 unlinked (18.5%), 2 of them a test source → **3 real (11%)** |
| C1 "462 of 513 all-time (90%)" | 521/586 (89%) — but **334 are May–July crypto-era rows** from before a plan link existed |
| C2 "the DB stores '' (5/5)" | **Half false** — 4 rows DO carry the sentinel (530, 539, 545, 546); both forms coexist, which is the real hazard |
| C4 "584/586 never got their stamp" | **False** — both fully stamped: `pv=6`/`pv=5`, `S2`/`S3`, `matched=1` |
| C3 "UpsertArm overwrites version" | **TRUE** — `store/armed_orders.go:159` |

The three unlinked rows (566, 571, 580) have **no arm within 30 minutes** of any
of them. There is nothing to join on, so the honest value is the sentinel — not
a guess, and not `""`.

On that measurement the owner dropped E2, E5's resolve ambition and the
order-name plumbing. The wave shipped at roughly a fifth of its dispatch.

## WHAT SHIPPED

**E4 · one sentinel** — `store/attribution.go`: `PlanUnresolvable`, a three-state
`PlanLinkState` (`unstamped` / `unresolvable` / `linked`) and
`ClassifyPlanLink`, which is the one place the states are told apart. Consumers
call it instead of comparing strings, so neither form can be missed again.
`IsPlanLinked` returns false for BOTH the sentinel and `""` — that is the point.

**E1-lite · stamp at materialization** — `trader/ninjatrader/reconcile.go:418`.
The untracked-position path knows an account, a symbol, a side and an average
price, and no order of ours; it now writes the sentinel at creation with a loud
line, so a row is never born `""`:
```
🔗 attribution: materialized MNQ long @ 29100.00 with NO recoverable lineage — plan_id=UNRESOLVABLE (never joinable; counted in the boot line)
```

**Convergence** — `ConvergePlanLinkSentinel`, idempotent behind a
`system_config` flag, WHERE-scoped to **CLOSED rows in the day-plan era only**.
Pre-era history is deliberately untouched: marking a June crypto trade
"unresolvable" would claim we looked for a plan that never existed — the same
lie in the other direction. Logs the ids it changed.

**E3 · `armed_under_version`** — `store/armed_orders.go`. Set once at first
authorization and on a genuine re-authorization of a terminal row (a new arm
reusing the row id); **excluded from the non-terminal update map**, so a
re-authorization moves `version` and never `armed_under_version`. Legacy rows
adopt their current version once on next touch, so the table self-heals rather
than needing a migration that guesses. `version` is now documented in the struct
as last-touch.

**Boot line**, counts READ from the table:
```
🔗 attribution: stamp-at-materialization=on · armed_under_version=on · unresolvable=N (sentinel "UNRESOLVABLE") · unstamped-closed=M (pre-era history)
```
`unstamped-closed` is printed beside the sentinel count deliberately: a healthy
sentinel number must not be able to hide a fresh `""` regression.

## TESTS

| id | pins | result |
|---|---|---|
| F2 | the sentinel constant, and `ClassifyPlanLink` over all three states; neither `""` nor the sentinel is a "link" | PASS |
| F2b | `StampUnresolvableLineage` + `CountUnresolvable` — a lineage-less row is countable | PASS |
| F4 | UpsertArm twice: `armed_under_version` **unchanged** (3), `version` moves (3→7) | PASS |
| F5 | convergence is scoped (era yes / pre-era no / OPEN no / linked untouched), idempotent, and the boot line reports real counts | PASS |
| F5b | class-40 P&L aggregators key on trader/session, **not** plan_id — a sentinel row's money still counts; pinned so a later wave does not "helpfully" exclude it | PASS |
| pin | the materializer carries the sentinel and logs (whitespace-tolerant regex, after an exact-match version broke on gofmt alignment) | PASS |

**Suites:** Go **27 ok / 0 FAIL**, goldens PASS, vet clean, tsc clean,
**vitest 38 files / 297 tests** (dev baseline is also 297 — verified against a
clean `origin/dev` checkout after the count differed from an earlier run).

## A15 — WHAT THE OWNER WILL STILL SEE

- **518 pre-era CLOSED rows still carry `""`**, by design. Any all-time
  "unresolvable rate" will still read ~89%; the day-plan-era rate is the
  meaningful one and is 3 rows.
- **`e7_farside_test` rows (573, 574) keep `""`** — a test source, out of scope.
- **The three converged rows gain no plan link.** They become explicitly
  unknown, not resolved. Nothing recovers a link that was never recorded.
- **`armed_under_version` is 0 on existing rows** until each is next touched.
  The self-heal is lazy on purpose — a bulk migration would have to guess.
- **1D can run on this join as it stands** (owner ruling): the era's data is
  complete apart from three explicitly-marked rows.

## ROLLBACK

Additive: one new file, one column (default 0), one guarded idempotent UPDATE
scoped to the day-plan era. `git revert` + rebuild. The converged rows would keep
the sentinel, which is harmless — it is the documented value the view already
expects. Binary rollback: `nofx-bin.prev.boot`.
