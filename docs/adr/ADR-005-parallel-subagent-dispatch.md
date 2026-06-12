# ADR-005: Parallel `general-purpose` subagent dispatch for multi-task plans

## Status

Accepted (2026-05-22). Used to land Plans 2 / 3 / 4 / 4.1 / 4.2 / 5. Lesson learned (and codified) during PR #3 Stage 5E and reinforced by Plan 4.2.

## Context

The NQ implementation plan (`docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md`) decomposes into 37 numbered tasks spread across 7 plans. Several plans contain **2–4 largely disjoint task blocks** — for example Plan 2 has tick rounding (Task 17), CME calendar (Task 18), contract roll (Task 19), and decimal-safe math (Task 20). Sequential execution of disjoint work wastes wall-clock time on what is essentially embarrassingly parallel labor.

Two execution styles were available:

1. **Sequential** — one task at a time on the main thread, simple and obviously correct.
2. **Parallel subagent dispatch** — invoke multiple `general-purpose` subagents simultaneously, each owning a disjoint task block, then verify convergence before committing.

A third style was tried and rejected (see Alternatives).

## Decision

**Multi-task plans dispatch parallel `general-purpose` subagents** with these mechanics:

1. **One subagent per disjoint task block.** A task block is "disjoint" if it touches a non-overlapping set of files (or files that can be touched additively without conflict — see ownership markers below).
2. **Ownership markers in shared files.** When two task blocks need to add code to the same file (e.g., both Plan 2 and Plan 3 add gates to `kernel/engine_analysis.go`), each new block is delimited by a `// Plan N Task M — owned by TX` comment. The convergence verifier checks that no two subagents wrote into the same region.
3. **Convergence verification phase before commit.** A serial verification pass that runs `go build ./...`, `go test ./...`, `cd web && npm run build`, and a `git diff` review against the dispatch brief. No commit until convergence is green.
4. **Plan 1 critical files are read-only inside dispatched work.** See ADR-007.
5. **Tool manifest is a hard rule** — every dispatched subagent receives `Bash, Read, Edit, Write, Grep, Glob` and **must not** be downgraded to a read-only subagent type. Lesson from PR #3 Stage 5E.

## Consequences

**Positive**

- ~3–4× faster wall-clock execution on multi-task plans. Plan 5 (4 disjoint tasks) landed in roughly the time one task would have taken sequentially.
- Forces clean task decomposition. If two tasks cannot be split cleanly, the dispatch fails fast at convergence — the plan itself needed to be re-cut.
- The ownership-marker discipline doubles as future-archaeology comments: every block is traceable to a plan + task.

**Negative**

- Convergence verification phase is real overhead. It is not optional. Plan 4.2 (the integration-bugs fix-up PR) exists because Plan 4's convergence missed five downstream consumers of the `NinjaTrader` exchange type — a reminder that "build green + tests green" is necessary but not sufficient; live-DOM verification is also required for UI changes (see CONTRIBUTING.md).
- Higher coordination cost in the dispatch brief — each subagent needs explicit lists of allowed-to-touch and forbidden-to-touch files.

**Neutral**

- Subagents do not share state. Anything a subagent learns has to come back through its returned report, which the main thread then incorporates into the next dispatch or the final commit message.

## Alternatives Considered

- **Sequential single-thread execution (rejected for multi-task plans)** — Simple, safe, and unbearably slow. Appropriate only for single-task plans (e.g., Plan 6's runbook write-up) where there is no parallelism to exploit.
- **Shared editing locks (rejected)** — Adds infrastructure (lock service, lease renewal) for a workflow that already converges in seconds. Ownership markers achieve the same outcome at zero infra cost.
- **`feature-dev:code-architect` subagent for the implementation phase (REJECTED with lesson learned)** — The `code-architect` subagent is read-only. It cannot run `Bash`, `Edit`, or `Write`. Dispatching it to do implementation work results in a subagent that reports "I would do X" but produced no diff. This was the failure mode in PR #3 Stage 5E and is now a hard rule: implementation dispatch always uses `general-purpose` subagents with the full tool manifest.

See also ADR-007 (Plan 1 critical-file integrity, which is the chief constraint on parallel dispatch) and `CONTRIBUTING.md` (the live operating procedure).
