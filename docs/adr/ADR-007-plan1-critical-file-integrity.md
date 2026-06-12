# ADR-007: Plan 1 critical files remain byte-identical across subsequent PRs

## Status

Accepted (2026-05-22). Enforced by review discipline in every PR since `v1.0-plan1`. Documented as a hard rule in `CONTRIBUTING.md`.

## Context

Plan 1 of the NQ implementation plan ships the **CSV bridge from Go (WSL2) to NinjaTrader 8 (Windows)**. It was the first plan to land and the first to be **validated against a live SIM environment**: on 2026-05-22, the system placed real SIM fills on SIM101 at NQ price 29807 via the NT Playback connection, with `cmd/nq_smoke` driving the round-trip.

That validation is **expensive to repeat**:

- Requires NT8 running on a Windows host with VLTrader compiled and attached to a chart.
- Requires a Databento API key with live data entitlement.
- Requires the WSL2 host in mirrored networking mode.
- Requires a session window where the operator can actually watch the fills land.

Subsequent plans (2–6) all touch the surrounding code but are **NOT** expected to alter the validated CSV bridge path. The risk: a well-meaning refactor "while we're in here" silently regresses the live-validated path, and we don't discover it until the next time someone re-runs the full SIM acceptance — which could be weeks.

## Decision

**Plan 1 critical files must be byte-identical across every subsequent PR.** Verified via `git diff <baseline> -- <file>` in every dispatch convergence phase and again at commit time. The list is locked in `CONTRIBUTING.md`:

```
provider/databento/historical.go
provider/databento/resolve.go
provider/databento/contract_calendar.go
provider/databento/client.go
provider/databento/mock_server.go        (Plan 5 — added later, locked from then)
provider/ninjatrader/csv_writer.go
provider/ninjatrader/csv_tailer.go
provider/ninjatrader/types.go
provider/ninjatrader/mock_nt.go          (Plan 5 — added later, locked from then)
cmd/nq_smoke/main.go                     (ADD-only — see Exception)
kernel/engine_prompt_futures.go
kernel/cme_calendar.go
kernel/risk_limits.go
kernel/engine_prompt_golden_test.go
market/databento_adapter.go
market/decimal_safe.go
market/data_freshness.go
trader/ninjatrader/tick_rounding.go
trader/ninjatrader/trader.go
```

**Exception (spec-authorized):**

- **`cmd/nq_smoke/main.go` ADD-only**: Plan 5 Task 29 explicitly authorizes additive sub-commands (the smoke harness gained `smoke_databento.go`, `smoke_resolver.go`, `smoke_prompt.go`, `smoke_roundtrip.go`, `smoke_all.go`, `help.go`). `main.go` itself receives only the dispatcher wiring; existing logic is preserved.
- **Plan 4 Task 24 wrappers**: retry / circuit-breaker wrappers around `provider/databento/client.go` and `provider/ninjatrader/csv_writer.go` are spec-authorized. They are written as **wrappers in separate files** (not in-place edits) so the underlying validated functions remain byte-identical.

## Consequences

**Positive**

- The live-SIM-validated path remains live-SIM-validated through every subsequent release. The bot's most expensive proof of correctness stays valid.
- Reviewers have a clear, mechanical check: `git diff v1.0-plan1 -- <list>` must show only spec-authorized changes.
- New plans that need to "tweak" a Plan 1 file have to either (a) go through a wrapper (Task 24 model) or (b) write a new file that supersedes — both make the architectural cost visible.

**Negative**

- Review ceremony on every PR. The list has 19 entries; the verification step is repeated each cycle.
- Genuine bug fixes in Plan 1 files are harder. A real defect would need a re-validation against live SIM, which is the whole point — but it does mean we don't casually fix typos in the locked files.

**Neutral**

- The list grew once (Plan 5 added the mock_server and mock_nt files when they shipped; from that point forward they are locked too). This is the expected pattern: a file enters the locked list when its first validated build ships, never earlier.

## Alternatives Considered

- **Test coverage alone (rejected)** — Plan 5 ships a good testing matrix (Databento mock, NT mock, prompt goldens, smoke sub-commands), but unit tests do not exercise the full live round-trip the way the 2026-05-22 SIM session did. Tests are necessary; they are not sufficient.
- **Git pre-commit hooks to mechanically enforce (deferred)** — Could be added: a hook that runs `git diff --name-only v1.0-plan1 -- <list>` and aborts the commit. Worth doing once the list stabilizes further. For now the discipline is review-side.
- **No special protection (rejected)** — Would defeat the purpose of having a live-validated path. The cost of one silent regression here is far higher than the cost of the review ceremony.

See also ADR-001 (the CSV bridge that defines the critical path), ADR-005 (the dispatch protocol that enforces the rule), and `CONTRIBUTING.md` (the operational checklist for every PR).
