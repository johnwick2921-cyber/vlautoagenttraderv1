# Repository control-boundary repairs — source candidate

Owner requested detailed traced review and repairs until complete. Source base:
`63968be62e44db2fb07a92883e02127b9064b0be`; branch
`fix/repo-audit-control-boundaries-20260913`; session
`reporepair-8d38e6ac/Codex[unlisted]`. Isolated worktree
`/tmp/nofx-audit-repairs-20260913`. No production database, settings, account,
order, deployment or runtime changes. Source review evidence is on
`docs/repo-understanding-20260913` under its separately claimed report scope.

Spec freshness: `dfda15e1425ba38671deb7ffa7a4f211a27846e2 test: isolate session clock fixtures from weekly backfill workers`
was the last AUDIT-CHECKLIST change at acceptance. Current tracked CLAUDE-canon
is read; main stays deploy-only. Branch names identify provenance, not author.

## Confirmed and repaired so far

- Authenticated object ownership: production router regression reproduces a
  foreign trader DELETE returning 200 and removing its equity child row. The
  middleware now validates all query/body trader IDs and trader path IDs. The
  delete handler additionally checks ownership; the store authorizes before
  touching children and performs both deletes transactionally. Injected child
  deletion failure verifies rollback. Own-trader reads remain available.
- Q&A: no plan plus available bars reproduces a nil row panic. Creation time
  is absent (zero) when no row exists. Historical fallback uses the existing
  trader-scoped query, tested with a foreign plan and an empty owned trader.
- Strategy updates: oversized prompt reproduces HTTP rejection after persisted
  changes. Token validation now runs before write/log/reload. Regression compares
  stored name and configuration bytes after rejection.

## Verification to date

Go 1.25.13. Focused production-route/transaction/context/rejected-save tests fail
on the preceding implementation and pass on repaired code. Existing plan/risk
ownership and strategy create/edit preservation tests included. Tests use
throwaway SQLite files and synthetic bars, never the owner's database or wire.
A first sandbox run failed because the external Go cache was read-only; it was
rerun with approved cache access. That setup failure is not a product test result.

Full suite, race coverage where relevant, independent repair review, guide rev
finalization, publication and final artifact verification are **pending**.
No runtime incident, exploitation, real fill, or deployment success is asserted.

## Execution boundary repairs

- Boot refusal: six adapter-side regression cases failed to see the boot latch;
  a ready loopback peer additionally received a limit entry with refusal true.
  Checks now precede market/limit/stop entry registration/send. Boot assertion
  runs before saved-trader autostart. Existing capability and unbound-account
  tests pass alongside the new tests. The loopback is a synthetic peer, not NT8.
- Broker-terminal retirement: same-version cancelled and filled rows each minted
  a fresh authorization in a temporary ledger. Retirement now precedes successor
  creation. Focused store tests confirm new-version and boot-sweep behavior.

These changes have not been deployed. Full combined review remains pending.

## Session mutation repair

Ask-Planner apply now resolves a session before dereferencing it; session gaps
return the existing stale-plan refusal. Apply and realign use the wrap-aware
chain date already used by plan reads, so after-midnight Asia mutations address
the previous calendar date's chain. Focused tests cover the session gap and
overnight date plus existing ask/realign/context tests. This defect was established
by source inspection; the new helper tests were not run against the old code.
Atomic proposal application/version binding remain separate open findings.

## Broker-book evidence and cancellation

Regressions reproduced missing/null `orders` parsing as an empty book, stale
book evidence authorizing cancellation, and a failed cancel send returning true.
The parser now rejects an uncomputed list (explicit `[]` remains valid). Live
book data and receipt time are read under one cache lock. Signal/row cancellation
uses the existing snapshot age limit and reports send failure as failure.
The existing independently established flat-position exception is preserved.
Focused parser, cancellation and desync tests pass. No real cancel was sent.

## History stream teardown

A race-detector test drives actual TCP history-frame dispatch concurrently with
subscription teardown/replacement. The preceding implementation panicked with
`send on closed channel` in `TCPServer.readLoop`. The read lock now covers the
nonblocking send; teardown closes only under the corresponding write lock.
The same test passes under `-race` and delivers a sentinel after the churn.
This proves the exercised loopback interleaving; it is not NT8 integration proof.

## Boot sweep settlement

A temporary-ledger regression reproduced successful cancel transmission becoming
`cancelled` with snapshot ID zero. The sweep now persists cancellation intent,
uses the guarded sender, and leaves the row pending. A received cancellation
update does not bypass pending snapshot confirmation. Existing same-version
boot-sweep reauthorization occurs only after confirmation. The recorded completion
counter increments transactionally with that transition, not at send time.
The boot line now says requests, not completed cancellations. Focused boot-sweep,
reauthorization and cancel-lifecycle tests pass. Old tests that equated send with
settlement were updated to inject persisted broker evidence. Historical counter
values are not retrospectively corrected by this code change.

## First complete Go-suite result

`go test ./...` at repair checkpoint `234b0262` completed: every package passed
except store. Three tests failed: the new terminal-state fixture duplicated
canonical state names, and two older append-only fixtures expected replacement
without advancing authorization version. The fixture now enumerates canonical
terminal states; append-only scenarios explicitly advance the plan version.
Their original record-retention and no-row-per-cycle assertions remain. All
three plus the expanded same-version regression pass in a focused rerun.
This is not yet the final combined-suite result.

## Account isolation and reset synchronization

Synthetic regressions reproduced: a bound account with no balance snapshot (and
an unbound adapter) returned another account's equity; another account's open
MNQ row suppressed materialization of the bound account's held position; reset
replaced pending maps without acquiring their mutex. Balance now returns
unavailable without its own snapshot. Reconciliation's legacy fallback queries
only the same trader's unassigned-account row. Reset takes pendingMu before mu,
matching reconciliation's lock order. Focused tests pass under `-race`, including
the existing legacy-empty-account deduplication case. An older test explicitly
expecting foreign-balance fallback was corrected. No actual account was selected
or mutated. The lock-ownership regression is controlled synchronization evidence,
not proof of every possible interleaving.

## Successor provenance

The production-style fresh successor lost BootID and ArmedUnderVersion because
its early Create bypassed initialization. Regression across canonical terminal
states reproduced both blank values. Successor creation now stamps this process
and the new authorization version. Focused provenance/append-only tests pass.

## Authenticated chat memory

Both HTTP handlers accepted a caller-selected numeric conversation ID. A
synthetic authenticated `/clear` request reproduced deleting a different owner's
history while leaving its own history intact. HTTP now always derives that
identity from authenticated middleware. Normal and SSE handler regressions pass;
no model/network call or real conversation was used. Telegram's separate identity
flow is unchanged. Shared mutable AI-client selection remains an open finding.

## Cancellation evidence timing and restart budgets

[A] A regression reproduced a fresh but pre-request empty snapshot settling a
later cancellation. Confirmation now requires a valid explicit book whose receipt
is at or after the persisted cancel request; missing and future receipt times
remain unavailable. The test follows older empty, newer working and newer empty
evidence. A second regression reproduced a previous process exhausting the new
process's retry cap. The cap check now applies the process identity before
counting attempts. Both regressions and existing settlement/boot-sweep tests pass
(`/tmp/nofx-cancel-evidence-after.log`). No broker or runtime settings were touched.

## Stop-entry refusal and other scenarios

[A] The actual placement-loop regression reproduced S2 being cancelled with
`one_live_entry: S1 placed` when S1 was already through, had unknown price, or
was refused by the AddOn build gate. The helper now returns whether placement
was registered; only that result closes the pass and retires other scenarios.
All three cases pass. A separate dispatch regression confirms registration
still commits the account after an ambiguous send failure. Existing receipt,
slot and fast-rejection tests pass (`/tmp/nofx-stop-refusal-after.log`).
The limit-path behavior after an ambiguous send remains a separate review item.

## Structural prompt and advisory parity

[A] After the owner reported the usage limit reset, the previously blocked
regressions ran and reproduced a universal ATR-floor prompt and false legacy
R:R/stop warnings for a reject arm. Prompt contract and facts now separate
structural reject fades from non-reject legacy floors; omitted arms do not grant
an AI-route bypass. Reject feasibility is left to composed-geometry admission,
not authored-price legacy warnings. Focused prompt, warning and class45 checks
pass (`/tmp/nofx-structural-prompt-after.log`). The existing heading and historical
reject-warning assertions were updated; legacy floor arithmetic remains tested.

## Second full Go suite and protection shape

[A] The full suite at710ea1c8 completed with two failing tests: the historical
no-one-setup-reference scan and source coordinates in SYSTEM-MAP. The former now
exempts only the AST-bounded advisory ArmFeasibilityWarnings function; the map
golden stays byte-identical. Updated coordinates and focused guard tests pass.

[A] Four protection adjudicator regressions reproduced an `-sl` name overriding
wrong side, wrong type, missing action and missing type. Known protection now
requires the order shape; incomplete named live stops remain UNKNOWN. Focused
protection, short-side and map-reference tests pass in
`/tmp/nofx-protection-shape-after.log`; map golden/prompt tests pass in
`/tmp/nofx-map-guard-after.log`. These are synthetic book tests, not a live
unprotected-position incident. Final combined suite remains due.

## Missing permission and inherited authorizations

[A] A synthetic permission-facts panic reproduced an old authorization reaching
the loopback wire while the cycle logged fail closed. Missing verdicts now retire
unplaced authorizations. Retirement query/write/panic errors propagate to the
cycle, stopping new placement. The same production-cycle test with an injected
SQLite retirement failure sends nothing; the existing allowed-versus-declined
scenario test still passes (`/tmp/nofx-unknown-arm-after.log`). Named system-map
coordinates were synchronized and the map reference check passes.

## Current admission covers the placement pass

[A] A second production-cycle regression reproduced an inherited row reaching
the wire after current scenario quality was refused. Production now passes the
IDs of successfully admitted/saved rows into placement. Old rows cannot inherit
permission merely from their stored armed state. A failed refresh does not enter
that set. Focused missing-permission, quality-refusal, valid-placement and split
fixtures pass (`/tmp/nofx-arm-current-gate-after.log`). Direct offline/debug
placement helpers retain their existing explicitly invoked test semantics.

## Frozen tape and session retirement

[A] A tickOnce regression reproduced an unplaced NY authorization surviving
16:05 CT on unchanged bars. Session/news cutoff enforcement now precedes data
cadence skips, using the tick clock. The regression and existing class32
wall-clock scheduling/EOD/T1 tests pass (`/tmp/nofx-wallclock-retire-after.log`).
The test uses an unconnected synthetic SIM adapter and temporary ledger; it
does not claim a real resting broker order was cancelled.

## Order-fill ownership

[A] Source inspection found the authenticated order-fill handler calling a
shared-store query by order ID alone. The repaired path verifies order ownership
and filters fill ownership, and reads history without a running engine. The
production-router fixture covers own order, foreign order and an inconsistent
foreign fill attached to an owned order; all pass. Before repair the fixture
failed because the stopped owned trader was unavailable, so that before run
does not itself reproduce a foreign-data disclosure. No real records were read.
Logs: `/tmp/nofx-order-fills-before.log`, `/tmp/nofx-order-fills-after.log`.

## Combined Go verification checkpoint

[A] At `99a065430cb28ced23c4992fe04ff9b13787dc3d`, `go test ./...` passed (all packages; `/tmp/nofx-repair-full-suite-03.log`). Focused race tests passed (`/tmp/nofx-repair-race-03.log`); this was not an all-package race run. `go build ./...` passed using explicit GIT_DIR/GIT_WORK_TREE to prevent Go's VCS discovery selecting the unrelated read-only `/tmp/.git` ancestor. The initial build failed at VCS discovery, not compilation. Build output: `/tmp/nofx-repair-build-02.log`, exit0. These results precede subsequent C#/frontend/overlay/limit repairs and do not certify their combined head.

## Limit registration commits admission

[A] `b63747ea` reproduces the asymmetry through the actual placement loop, TCPTrader, ledger callback and TCPServer: post-registration send-age rejection allowed a second row to register. Same-plan and different-plan regressions failed before repair. Limit registration now commits the pass even when send returns an error; same-plan unplaced siblings retire, other-plan rows remain armed, and the registered row remains pending. Pre-registration refusal preserves other candidates. Focused limit/stop/map checks pass. This tests an actual transport refusal after registration, not an observed partial-write broker incident. Logs and test names are preserved in `trader/limit_registration_commit_test.go`. Admission log wording now distinguishes registration from transmission.

## Browser overlay revision binding

[A] HTTP overlay edits require the plan ID, plan version and overlay revision the owner viewed. The current-plan response exposes these fields. The API rejects stale drafts with409; the serialized plan writer rechecks latest version, lifecycle and overlay revision so planner appends or competing edits cannot invalidate a checked snapshot before append. Read failures refuse edits. Temporary-database tests cover valid edits, stale plan/overlay drafts, competing writers and retired plans; all pass (`/tmp/nofx-overlay-revision-02.log`). The first test run exposed an invalid test fixture with zero scenarios; it was corrected to a schema-valid plan. No pre-repair runtime incident is claimed. Frontend snapshot propagation is a coordinated pending lane. Historical Q&A records do not store authored version; their separate test-op guard is not promoted to equivalent revision binding.

## Request-local AgentBeta model selection

[A] Concurrent authenticated `/status` requests with two synthetic model owners reproduced replacement of the shared/background AI client. Each chat/SSE request now owns a freshly selected client for all downstream calls; history, setup state and per-user flow locks stay shared through an explicit owner reference, never a copied mutex. An unconfigured authenticated user no longer inherits the default owner's credentials. Focused identity/model/history tests pass under `-race`; all agent package tests pass (`/tmp/nofx-agent-client-after.log`, `/tmp/nofx-agent-full-04.log`). Extra tests check model identity stability and shared setup/history ownership. The first reproduction fixture lacked chat configuration and panicked; the initialized fixture then reproduced the actual shared-client mutation (`/tmp/nofx-agent-client-before-02.log`). No AI endpoint was called. This does not claim all background lifecycle or trade-confirmation ownership issues are solved.

## Integrated C# lifecycle repair

[A] Integrated original repair9140f6c9 and independent follow-up f1b7cc10 as
`e8d2243f` and `cc766e1c`. The follow-up reproduced deferred Change falsely
counted as coverage and synchronous terminal receipts lost during Submit.
Both are fixed. All33 extracted-production-method harness assertions pass;
all five sources compile against installed NT8 references. Detailed evidence
and limitations: `../2026-09-13-nt8-lifecycle-repair.md`. No AddOn deployment
or real NT8 event scheduling/OCO verification occurred. Partial completed-exit
wire semantics remain under a separate cross-boundary investigation.

## Swing evidence provenance and aggregate volume

[A] Synthetic detector fixtures reproduced two errors: zone defining wick came
from the next candle (pivot close used as an open-time lookup), and the first
bar's volume vanished from each aggregate bucket. Exact pivot-open identity
now supplies the wick while existing presentation/confirmation timestamps stay
separate. Aggregate initialization includes first-bar volume. Before tests failed;
focused swing/T3/zone/formation tests pass (`/tmp/nofx-swing-evidence-before.log`,
`/tmp/nofx-swing-evidence-after.log`). This corrects measured inputs; no strategy
expectancy improvement or recalibrated zone-width distribution is claimed.

## Delayed flatten lifecycle

[A] Repair94e08cf0 binds a scheduled fallback to immutable owned position row
and entry lineage. Known replacement, ambiguity or missing identity refuses;
Stop invalidates timers and waits for active callbacks. Immediate and delayed
close failures preserve protections. Production flatten/callback fixtures and
focused race tests pass. The broker command still has no atomic expected-position
fence, so an unseen broker replacement remains a limitation. Ordinary Stop
retains broker observers intentionally: held positions still need reconciliation.
Final observer disposal requires a safe handoff/removal design, not blindly
killing protection/close listeners when entry scheduling stops.

## Guide contradictions removed

[A] Updated existing guide paragraphs, not only added newer notes: shared Studio
R:R replaces obsolete ARM_MIN_RR prose; reject stops are explicitly exempt from
legacy ATR-floor wording; current One Setup permission consumption replaces the
obsolete claim that exclusions can never refuse; proximity slider direction now
matches `band = proximityK * dATR`. Historical exit-posture example is labelled
legacy and points readers to resolved current boot evidence. These are source
consistency corrections, not changes to the owner's saved values.

## Dependency advisories

[A] A paginated authenticated GitHub API read confirmed five open advisories,
recorded in `dependency-alerts-at-review.jsonl` (not inferred from the push banner).
Go gnark-crypto0.19.0 is replaced by advisory-patched0.19.2 for GHSA-fj2x-735w-74vq.
Only that module version and sums changed. Wallet package builds (no tests); mcp
package tests pass (`/tmp/nofx-gnark-compatibility.log`). Four compatible npm
updates are integrated in f5409132; their private updated install and combined
frontend suite are recorded in the web repair report. This establishes affected
versions, not exploitation or reachability in MNQ trading. Default-branch alert
closure requires publication/scanning and is not claimed from a local update.

## Weekly reader calendar evidence

[A] Production weeklyDailyBars returned two seven-day aggregates for ten daily
observations, contradicting its own daily-input contract. Before regression
failed; requesting completed daily observations preserves both CME weeks and
the known prior-week high, low and close. Focused weekly tests pass in
`/tmp/nofx-weekly-calendar-after.log`; before evidence is
`/tmp/nofx-weekly-calendar-before.log`. This fixes the active weekly reader. The old mid-week watch was removed
from the loop; its legacy same-resolver fixture is not production parity proof. Generic epoch aggregation for other consumers and the information
limits of daily bars for intraday weekend-gap timing remain separate issues.

## Subsequent independent boundary repairs

[A] a982cc74 adds atomic residual/receipt accounting and cumulative entry
notional. d3e4638e retires a replaced reconciliation observer after its old close
subscription drains; ordinary Stop retains protection observers. c20d0a82
preserves valid currently excessive exits as pending and makes absent/old first
account snapshots unknown rather than flat. A separate actual callback replay
proved entry/exit consumers can reorder; its ordered-processing repair remains
pending final integration and is a deployment blocker until verified.

[A] 3f21431a emits valid cumulative EXIT fills on terminal cancellation/rejection
as well as Filled. Transient PartFilled is not an additional charged receipt.
89 extracted C# harness assertions and five-source reference compile passed;
see the terminal-exit report. Installed NT8 callback behavior remains unverified.

[A] dd11670b retains same-view order snapshots as visibly stale/UNKNOWN on refresh
failure, ignores late account-view responses and states the endpoint's trader-bound
account scope. Existing account names are read-only because backend rename and
binding migration are not implemented. Creation and saved owner bindings are unchanged.

## Final source freeze preparation

Reviewed protected-file hashes updated only for the ordered TCP execution dispatch, shared entry-receipt fence, and C# execution-evidence build marker. The mutation guard remains active. The guide revision identifies backend source candidate cd2978b7; it does not identify the running binary. Adapter replacement receipt preservation is covered by the dedicated 2026-09-13 NT8 entry receipt report.

## Final disposition index

This report preserves chronological repair checkpoints; earlier pending statements describe their named checkpoints. Current issue dispositions and exact combined verification are in [the audit final ledger](../2026-09-13-repo-understanding/CHECKPOINT.md#final-verification). All 30 review artifacts are complete. This does not claim every baseline finding was repaired, deployment occurred, or trading profitability was established. The additional x/crypto dependency finding is addressed by 40ed5d95 plus build alignment d174ca95; the [advisory report](../2026-09-13-crypto-security-repair.md) retains the unmaintained, non-imported OpenPGP module-only limitation.
