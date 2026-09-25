# Automatic structural stops; owner-controlled daily loss

[A] The owner clarified **“i mean daily loss”** after the initial structural-stop
SIM boot. The agent had interpreted the earlier owner response as requiring an
additional per-trade dollar cap and refused all fades when that new cap was
blank. That interpretation was wrong. The correction removes the extra cap;
the system continues to choose the stop and target automatically from structure.

## Scope and unchanged risk controls

[A] Branch `fix/structural-stop-daily-loss`, claimed by
`structuraldaily-22fba7ca/Codex[unlisted]`, starts from current dev
`587148a6c52f5f4379c82cfa22a3f28ce7f44097`.
The extra `max_trade_loss_usd` config field, registry entry and UI input are
removed, as are `risk_cap_missing` / `risk_cap` admission decisions. Modeled
one-contract dollar exposure remains recorded using the instrument's point value.
The record reader retains its optional historical cap field for old evidence.
Old cap-refusal counters, if present, remain in the boot line's `other` total;
there is no new cap requirement hidden behind them.

[A] The structural prices, measured buffer, first eligible zone, net-gain test,
unchanged 2R policy, one-setup selection and existing entry gates are preserved.
Daily loss is enforced by the existing `EntryGate` daily-force-flat resolver;
this wave does not replace it with a per-trade proxy or change its switches.
The Guide, boot display, Rulebook, system map and checklist class 126 now state
the corrected owner intent. Checklist class 1 (self-imposed caps) also applies.

[A] Read-only bound-strategy measurement at acceptance: **one bound row**;
`daily_loss_limit_usd=450`, `guardrails_enabled=false`,
`daily_loss_enabled=false`, `max_trade_loss_usd` absent. Thus the saved $450
limit is currently **disabled**, not enforced. No live settings, account bindings,
owner values or switches are changed by this correction. Config keys and binding
were read through `traders.strategy_id`, never an unrelated active strategy.

## Behavioral evidence

[A] RED before production edit: the actual production-arm fixture computed
entry 29010, entry zone [29000,29010], buffer 5, stop 28995, target 29120,
modeled loss $34 including costs, then recorded quantity 0 with
`risk_cap_missing`; no arm row existed. The initial attempted reproduction still
passed because JSON decoding an omitted field retained a prepopulated fixture
cap. Clearing that field explicitly produced the quoted behavioral RED.
[RED](2026-09-13-structural-stop-daily-loss-evidence/red-missing-cap.log).

[A] GREEN: the same arm-cycle long/short fixtures now succeed without any per-trade
cap. A separate production-arm test trips the existing daily-loss resolver and
still gets quantity 0, `entry_gate` / `daily_force_flat`, with stop 28995 and target
29120 unchanged. Dollar exposure is still $16 at a $2 point value and $160 at $20
for the same six-point risk plus two-point cost fixture. This is contract arithmetic,
not a claim that MNQ calibration transfers to NQ.
[Targeted tests](2026-09-13-structural-stop-daily-loss-evidence/green-targeted.log).

[A] Two mutations ran through `scripts/mutate.sh`, with edits confirmed and
mutants building: reintroducing the extra cap refusal is KILLED by the automatic
arm test; bypassing the daily-force-flat leg is KILLED by the production daily-trip
test. The latter mutation is restored; no production daily guard code is changed.
[Cap mutant](2026-09-13-structural-stop-daily-loss-evidence/mutant-extra-cap.log),
[daily guard mutant](2026-09-13-structural-stop-daily-loss-evidence/mutant-daily-guard.log).
The UI fixture edits the existing daily-loss field and verifies that the added
per-trade input is absent.
[UI test](2026-09-13-structural-stop-daily-loss-evidence/ui-targeted.log).

## Research and deployment limits

[A] No level detector, zone merge, target selection, stop buffer, fill model or
post-entry exit is changed. The original **geometry-only** replay already separated
the missing-cap configuration refusal from candidate geometry. Its negative
expectancy remains the research result; removing a mistaken admission requirement
is not new evidence of profitability. Original artifacts must be reproduced at
their pinned source revision because their configured-admission labels described
the superseded cap policy.

[A] At the start of this correction the running binary was
`4127979f2fcc5615f4e8b17540f7aba74bb4ea92`, booted with integrity/goldens PASS,
MD5 `300dc70535f708be9f6278d252124ba4`. The book was flat and the weekend gate
reported its next open as Sunday 17:00 CT. First real composition/refusal proof
awaits an open-market cycle; fixtures are not live proof. The owner's deployment
GO and explicit market-closed A7 exception remain recorded. Corrected cutover
still requires fresh gates, backups, clean build, marker and observed boot.

## Source freshness before correction

- `docs/superpowers/AUDIT-CHECKLIST.md`: `4127979f merge: structural-stop candidate with verified CI corrections and class 126`
- `docs/superpowers/SYSTEM-MAP.md`: `c9147d54 fix: withhold placement when geometry refusal cannot retire an old arm`
- `docs/superpowers/VL-TRADING-RULEBOOK-v1.md`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `docs/superpowers/reports/2026-09-12-structural-stop.md`: `587148a6 docs: record structural-stop SIM boot and pending market-open proof`
- `store/knob_registry_table.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `store/strategy.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `store/structural_geometry.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `store/structural_geometry_test.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `trader/structural_fixture_test.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `trader/structural_geometry.go`: `c9147d54 fix: withhold placement when geometry refusal cannot retire an old arm`
- `trader/structural_geometry_boot.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `trader/structural_geometry_boot_test.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `trader/structural_stop_seam_test.go`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `web/src/components/strategy/RiskControlEditor.tsx`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `web/src/guide/content/guards.ts`: `c9147d54 fix: withhold placement when geometry refusal cannot retire an old arm`
- `web/src/guide/content/settings.ts`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`
- `web/src/guide/types.ts`: `8f4790ca release: prepare verified structural-stop binary and UI; hold cutover under A7`
- `web/src/types/strategy.ts`: `540c9e8d fix: compose level-fade risk from frozen zones and refuse invalid geometry`


[A] Frontend verification: **64 files / 430 tests PASS**, TypeScript PASS.
[Full frontend](2026-09-13-structural-stop-daily-loss-evidence/vitest-full.log),
[TypeScript](2026-09-13-structural-stop-daily-loss-evidence/tsc.log).

## CI race found during correction

[A] The full local Go suite passed. PR #116's race/coverage job then caught
`TestClass32FrozenTapeStillSkipsDataWork` restoring `market.FuturesBarsProvider`
while an unrelated weekly-backfill goroutine still read it (`auto_trader_weekly.go:103`).
The class-32 session/data-cycle fixtures now seed their already-completed weekly
document through the real plan store, so the weekly scheduler takes its existing
skip branch. Their session-read and frozen-data assertions are unchanged; no
production scheduler or market code changes. This is a fixture-lifetime failure,
not a reason to rerun a red job until it happens to pass.

[A] All class-32 fixtures pass 15 consecutive race-enabled repetitions.
[Race verification](2026-09-13-structural-stop-daily-loss-evidence/class32-race.log).

[A] The complete local race/coverage suite passed. CI's separate coverage workflow
also restored modules twice: once through setup-go and once through actions/cache.
The first run emitted thousands of existing-file extraction errors; the next
run stalled in that redundant cache step before tests. Remove the duplicate
restore and use `go-version-file: go.mod`, matching the release toolchain.

## Verified release candidate

[A] PR #116 merged as `0c9d4f30a0470510a15e4aa92d7fa6ec96e9c38d`. Before publication all 32 successful PR check
results completed, with one intentionally skipped container-manifest job. The
full local race/coverage suite passed; at the exact merged HEAD, full Go tests
including goldens, **64 frontend files / 430 tests**, and TypeScript passed.
The first merged frontend attempt was sandbox-blocked (`spawnSync go EPERM`);
the permission-correct rerun executed all 430 tests successfully.

[A] Clean clone `/tmp/structural-daily-loss-release/nofx`, Go go1.25.13,
`vcs.modified=false`; binary **73,422,552 bytes**,
MD5 `1dba3de51d4813481546ecb933325d4c`, SHA-256 `5738ec671b9d706e43f545c99c218bfcd18b4e982f22358c6dcf4581ee5045ca`. Guide SOURCE and RELEASE are stamped
from this binary revision in the following metadata commit; the Go binary is not
rebuilt from that deliberately different metadata revision.

[A] Backup at 00:41:13 CT: `before-correction.db`, **1,236,676,608 bytes**,
SQLite integrity `ok`, SHA-256
`cd05846b018be69073329b746cefe4330af5aeec811a9830d19d802ff7311684`.
Directory: `/home/hoang/nofx-backups/structural-daily-loss-20260913/`.
The original release marker and Guide source are preserved separately.

[A] Additional pre-edit source freshness at base `587148a6`:
`trader/class32_wallclock_test.go`: `1ef5ad51 class 32: scheduled session reads fire on wall-clock`;
`.github/workflows/pr-go-test-coverage.yml`: `5c757a74 B6+B8: the worktree recipe pointer; the three CI setup failures fixed as MEASURED, plus the two dev-red Go tests the coverage job actually fails on`.

Cutover/boot evidence is appended only after it is observed.

## Observed SIM cutover

[A] Under the helper-owned main-tree lock and the owner's GO plus market-closed
exception, main fast-forwarded to release metadata `642f8808`. Fresh pre-kill
five-leg gate: all PASS; zero positions, broker working orders, unplaced arms,
and planner claims. Gate age at SIGKILL: **0.28807 seconds**.
The service's `Restart=on-failure` policy was verified. Old PID 4165029 was
signaled at **2026-09-13T01:04:11.132267-05:00**; new PID 24534 was verified at **2026-09-13T01:04:20.177698-05:00**.

```text
09-13 01:04:16 [INFO] nofx/main.go:295 🔐 BOOT INTEGRITY OK — rev 0c9d4f30a047 · built 2026-09-13T05:58:55Z · expected 0c9d4f30a047 · goldens PASS
09-13 01:04:16 [INFO] nofx/main.go:548 🎯 stop/target: stop=zone-edge+buffer buffer=4.50[I] (p95 of measured overshoot; resolver=ResolveStructuralStop:C5_MNQ_default[I]; calibration=C5-H12-IS-6181-p95-20260912; sweep=[0.25 1.25 4.5] points[I]) · atr-fallback=0 · refused today=0 (no_target=0 net<=0=0 rr<2.00=0 no_provenance=0 other=0) · target=first-distinct-eligible-zone · never-widened=asserted · research-candidate
```

[A] Disk RELEASE, committed RELEASE, Guide source, health revision and loaded
executable all identify `0c9d4f30a0470510a15e4aa92d7fa6ec96e9c38d`. Built, disk and loaded MD5
are `1dba3de51d4813481546ecb933325d4c`; all **92** dist files match
the prepared SHA-256 manifest, and the main JS asset contains the same Guide
revision. The new boot line contains no mandatory per-trade-cap requirement.

[A] Bound strategy config SHA-256 remains
`3e63209e52f9709b6366c1e7164ecdbaea5069567c0159e98593f170b5158a51`. **$450 saved daily-loss amount,
master guardrails OFF, daily-loss switch OFF**: unchanged by this deployment.
The daily-trip production fixture proves the existing gate still blocks when
tripped; it does not mean the owner's currently disabled switch is enabled.

[A] The previous `4127979f` binary and dist were moved into the correction
backup directory with the full embedded old revision in their names. The new
binary/dist, old release/source metadata, DB backup, source bundle and evidence
are retained for recovery. No DB setting was written.

[A] The initial post-boot order leg was explicitly ledger-only while the NT8
snapshot had not yet arrived. At **01:05:09 CT**, all five post-boot legs passed
with a real NT8 order snapshot (age 29 seconds, AddOn build 2026-09-07-h1), zero
broker orders, zero positions and no planner claim.

[A] At 01:04:30 CT, `system_config` prefix `structural_geometry:` contains
**0 records**, keys `[]`; this process has **0 composition lines** and **0 actual
boot-sweep result lines**. The observed weekend gate says next open Sunday
2026-09-13 17:00 CDT. The closed-market return precedes the arm/sweep call.
No live composition, live refusal, or sweep cancellation count is fabricated
from fixtures or lifetime counters. Those proofs await an open-market cycle.

[A] Closeout artifacts: [boot verification](2026-09-13-structural-stop-daily-loss-evidence/boot-verification.json),
[pre-kill gate](2026-09-13-structural-stop-daily-loss-evidence/pre-kill-gate.json),
[post-boot gate](2026-09-13-structural-stop-daily-loss-evidence/postboot-gate.json),
[proof status](2026-09-13-structural-stop-daily-loss-evidence/postboot-proof-status.json).
The corrective implementation and deployment are complete; profitability and the
first open-market composition/refusal remain separate, unproven claims.
