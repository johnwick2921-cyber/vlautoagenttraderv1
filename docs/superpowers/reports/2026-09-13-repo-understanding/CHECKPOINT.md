# Audit and repair checkpoint

## Completed scope

All 30 scoped assignments are reported. At baseline 63968be62e44db2fb07a92883e02127b9064b0be, 28 primary reviews cover 1,049 unique first-party files / 250,582 lines. Two bounded independent reviews examine repair 99a06543 and add no primary files. Coverage and publication validators establish artifact/hash/range/function-note consistency, not a second semantic reading or runtime correctness. The 30 reports and their 120 standard artifacts are preserved.

Current candidate: `13882f01f72c313f4454bd29a309d5e31b6ec0cd` on `fix/repo-audit-control-boundaries-20260913`. The disposition and CTO assessment include the ordered-execution and positive-entry/replacement follow-ups through this revision. Those implementation tasks are committed; they are not awaiting an unspecified future repair. CORE-TRACE.md pins 35 declarations to frozen verification tree `13882f01f72c313f4454bd29a309d5e31b6ec0cd`.

Historical checkpoints remain revision-scoped: Go full suite/build and selected race checks at 99a06543; frontend 71 files/451 tests plus build at 28a6f32e; later chart follow-up 9 focused tests/build; final-marker C# reference compilation and 89 extracted assertions. No earlier green result is relabelled as a pass at the final candidate.

## Substantive remaining limits

Receive-order processing does not reconstruct exchange chronology or pre-owner queued events. No durable inbound journal or crash-atomic process-local hook delivery exists; storage failures still need later evidence/replay. Synchronous callbacks backpressure TCP; continuing cumulative exposure does not rebuild all already-emitted terminal analytics. Broker-side atomic expected-position fencing remains absent. Ambiguous multiple-row attribution refuses. Replacement handoff is repaired, while installed NT8 scheduling/OCO, reconnect and final shutdown still need controlled verification.

UI limits include backend selected-account completeness in history/chart/orders, forming-candle marker association, bulk model payload replay/extra-model thinking knobs, submit lifecycle and DayPlanEditor draft/default/inheritance/translation issues. Existing account names are now read-only; backend binding migration is separate. Corrected-PNL fallback, stale response identity and open-order error-to-empty have named repairs and are not wholly open.

The audit did not establish production database restoration, a first live structural composition/refusal, calibrated stop buffers, target optimality or causal out-of-sample net expectancy. Historical exploratory research is qualified separately from current measurement dependencies. Main runtime, owner settings, accounts and live records were not changed by this source audit; SIM and the owner's daily-loss policy remain intact.

## Final verification

**Source checks complete at `13882f01f72c313f4454bd29a309d5e31b6ec0cd`.** The full security-updated Go suite and race run were first verified at b45b3efd. The only subsequent change, 13882f01, updates the independently reviewed go.mod protected hash; all mutation assertions remain active. Full Go tests/build and the frontend suite were rerun on that clean commit. Every runner receipt records exact command, commit before/after, clean state, exit code, duration and log SHA-256. Historical failed runs are retained and never relabelled as passes.

| Check | Verified result and scope |
| --- | --- |
| Full Go suite | PASS: `go test ./...` at b45b3efd (314.242s), then at13882f01 (3.759s, cached package results). [Final receipt](verification/go-full-release.json). |
| Race detector | PASS: full store, provider/ninjatrader, trader/ninjatrader and trader packages, 322.822s at b45b3efd. Subsequent diff is one frontend hash string. [Receipt](verification/go-race-final.json). This is not an all-repository race run. |
| Backend build | PASS: `go build ./...` with explicit worktree Git environment and VCS metadata preserved, Go1.26.8. [Receipt](verification/go-build-release.json). |
| Frontend suite | PASS:71 files /454 tests at13882f01. The preceding run correctly caught the changed go.mod hash (453 passed,1 failed); the reviewed hash update resolved it without weakening the guard. [Receipt](verification/web-tests-release.json). |
| Frontend production build | PASS at b45b3efd; the subsequent hash-only test-data edit does not enter the product bundle. Existing large-chunk warning remains. [Receipt](verification/web-build-final.json). |
| Offline futures smokes | PASS: prompt and synthetic TCP roundtrip at13882f01. [Prompt](verification/smoke-prompt-release.json), [roundtrip](verification/smoke-roundtrip-release.json). No live broker order. |
| C# | Five files compile against installed NT8 references;89 extracted-production-method assertions pass. Receipts at e333de41; those C# sources are unchanged in the final candidate. [Compile](verification/csharp-reference-compile.json), [harness](verification/csharp-harness.json). Not installed-NT8 execution. |
| Dependency security | npm audit:0 findings in selected lockfile. govulncheck:0 reachable and0 imported-package vulnerabilities in default Linux/amd64 app/test closure; GO-2026-5932 remains module-only, no fixed version. [Advisory report](../2026-09-13-crypto-security-repair.md). Remote Trivy on b45b3efd cleared its previous version-fixable alerts; other remote jobs are separately reported and not universally declared green. |
| Review consistency |30 indexed reviews /120 standard artifacts; hash/range/named-function-note and publication checks pass.35 selected core declarations are pinned to the final source commit. These checks prove artifact consistency, not runtime semantics. |
| Source backup | Standalone full-history bundle: **78,752,411 bytes**, SHA-256 `6e29a3f7d09769fbffcbfc96e9f29e4defb07cbcb4e7059a957ebc79fd030265`. Fresh clone, full fsck, exact-revision checkout and clean-tree check passed. [Restore record](verification/source-backup.json). |
| Runtime / installed NT8 / profitability | Not verified or changed by this audit. No deployment, owner-account/settings changes, live database mutation or new profitability experiment. |

The source bundle records repair13882f01 and docs9988cd55; final publication Markdown and verification receipts are additionally present in the delivery ZIP. PDF exports and the ZIP receive separate byte/hash/integrity receipts outside Git, avoiding circular self-hashes. The source restore is **not** a production database restore. Earlier progress bundles and usage-blocked reports are historical.

Guide stamp40ed5d95 identifies the backend source candidate including the patched module graph; it does not identify the running binary. AddOn candidate ID remains `2026-09-13-execution-evidence`. A future deployment must follow the existing release procedure and coordinate Go/C# installation; this report grants no deployment approval.

The final report distinguishes repaired and reproduced defects from unresolved source/runtime limits. All baseline findings without a named repair retain their original qualifications. A later documentation-only merge must retain this source revision provenance and must not be described as an additional source audit.
