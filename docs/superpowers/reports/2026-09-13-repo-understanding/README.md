# Repository understanding — completed baseline review, repair status tracked separately

Source base: `63968be62e44db2fb07a92883e02127b9064b0be`. Audit branch: `docs/repo-understanding-20260913`; claim: `8c7bc6be8f1a425067135b612ba62933e1fc3da1`.

The owner requested a detailed, accurately traced, approximately 30-agent repository review. This directory preserves the evidence and coverage rather than equating an index with understanding. Confirmed defects were repaired on separately scoped branches with focused and independent verification; integration and final verification are tracked below. This audit does not authorize deployment or account/settings changes.

## Packaged reports

- [CTO-REPORT.md](CTO-REPORT.md): assembled CTO assessment, repair dispositions, core source trace and verification ledger. This is the source for the summary PDF.
- [FULL-AUDIT.md](FULL-AUDIT.md): the same report plus **all 30 original review-report appendices**, each with historical scope notices. Exact byte counts and SHA-256 hashes are recorded in [report-package.json](report-package.json). The delivery receipt supplies a raw URL pinned to the published commit.
- [report-package.json](report-package.json) records exact input/output hashes and sizes; [PACKAGING-UPDATES.md](PACKAGING-UPDATES.md) records the later committed repairs that supersede specific older open items. Ordered-execution repairs are committed; final merged-head results and publication stamps are recorded in CHECKPOINT.md.

Rebuild from the repository root with `python3 docs/superpowers/reports/2026-09-13-repo-understanding/tools/build-reports.py`; append `--check` to verify deterministic output without writing. Edit the linked originals, then regenerate. Original review files and evidence artifacts are retained separately. Byte/word counts in this index must be refreshed if source documents change; the manifest is authoritative for each generated package. Packaging does not re-review the source or certify runtime behavior.

## Reading order

Start with [CTO-TRADING-LOGIC.md](CTO-TRADING-LOGIC.md) for the trading-process assessment, then [REPAIR-STATUS.md](REPAIR-STATUS.md) for fixed/open/runtime-unverified dispositions and revision-scoped evidence. [CHECKPOINT.md](CHECKPOINT.md) lists publication work still due. The numbered reports preserve baseline findings even when a later repair resolves them; do not read their historical open lists as final status.

## Scope and current status

- 3,726 tracked files inventoried by path, SHA-256, byte and line count.
- 1,049 first-party source files (250,582 lines) assigned once across 28 source reviews, followed by two independent cross-boundary reviews.
- Tests, fixtures and historical non-code artifacts are inventoried; relevant ones are followed. This is **not** a claim that every test or historical artifact was manually read.
- Go AST inventory: 1,292 files parsed, 10,269 functions/literals and 78,157 syntactic calls; zero parser errors. These are syntax inventories, **not type-resolved call edges or human reading**.
- `orientation/` holds initial subsystem reviews with explicit read ledgers and unresolved concerns. Their findings require root/cross-review validation.
- **30/30 scoped reviews reported**, covering all 1,049 assigned source files / 250,582 lines. The two independent cross-boundary reviews examined repair revision `99a06543`; subsequent repairs require their own verification.
- `review-plan.json` tracks assignments; `coverage-validation.json` checks source hashes, full line ranges, and named Go/TypeScript/JavaScript function notes. It does not certify semantic understanding or runtime behavior.

## Historical maps

The recovered Understand Anything graph is dated July 10, 2026 at `7a8adce0043729950f3d7cbfaf30810c8304709a`: 3,121 nodes, 9,588 edges, 11 layers and 15 tour steps. Its SHA-256 is `23aa686864d6e1af4e6b43ae52856175356d07e8baf066fe36d0e90418a2c43f`. CGC responds with 781 indexed files and 5,095 functions, but exposes no index revision/date. Both require comparison with current source. Neither establishes current completeness.

## Evidence contract

[A] directly read/run/observed; [B] inference from cited evidence; [C] hypothesis. Static concerns, offline reproductions and runtime incidents are separate categories. Each final trace will name source revision and file/function/line, actual test results, and remaining uncertainty. A passing suite is not proof of every behavior. No fabricated coverage, caller resolution, runtime observations, profitability or universal safety claims.

## Repaired candidate and verification

Current candidate `cd2978b77da54e2fceddfb19e1d3d148bd2bfb62` on `fix/repo-audit-control-boundaries-20260913` includes the scoped Go, C#, frontend, dependency and execution-evidence repairs. Ordered receipt processing, positive rejected entry evidence, shared-server entry fencing across adapter replacement, terminal cancelled/rejected EXIT receipts, observer replacement and UNKNOWN order display are committed. [REPAIR-STATUS.md](REPAIR-STATUS.md) separates those repairs from substantive remaining limits.

The one [final verification ledger](CHECKPOINT.md#final-verification) records root-run merged-head results and publication stamps. Earlier passing fixtures retain their original revisions. All 30 baseline reports remain unchanged; the usage-blocked snapshots under interim/ are historical. Source review did not deploy the candidate, mutate owner accounts/settings, or establish profitable trade selection.

## Review index

Each report links to its read ledger, function notes and sourced graph edges in
the same numbered directory. Function-note entries can include helper/callback
groups and dependency notes; they are not claimed as a unique function census.
Reviews 29/30 do not increase primary source coverage.
[publication-validation.json](publication-validation.json) independently checks
all 30 index links,120 artifacts and complete nonduplicated baseline assignments;
it reports zero consistency errors and does not certify runtime semantics. Named baseline Python
functions now also pass independent syntax-census consistency checks (385).

| Review | Area | Assigned source files | Assigned lines | Function-note entries |
| --- | --- | ---: | ---: | ---: |
| [01](reviews/01/report.md) | api-auth-and-manager | 24 | 6649 | 144 |
| [02](reviews/02/report.md) | api-auth-and-manager | 22 | 6660 | 165 |
| [03](reviews/03/report.md) | conversational-agent | 15 | 9219 | 310 |
| [04](reviews/04/report.md) | conversational-agent | 14 | 9223 | 281 |
| [05](reviews/05/report.md) | conversational-agent | 14 | 9224 | 299 |
| [06](reviews/06/report.md) | crypto-brokers | 24 | 9015 | 213 |
| [07](reviews/07/report.md) | crypto-brokers | 25 | 9071 | 193 |
| [08](reviews/08/report.md) | kernel-engine-clocks-and-risk | 31 | 7721 | 230 |
| [09](reviews/09/report.md) | kernel-engine-clocks-and-risk | 30 | 7725 | 253 |
| [10](reviews/10/report.md) | kernel-levels-and-structure | 25 | 7199 | 217 |
| [11](reviews/11/report.md) | kernel-plans-and-permissions | 22 | 7100 | 215 |
| [12](reviews/12/report.md) | market-and-other-data-providers | 49 | 7579 | 216 |
| [13](reviews/13/report.md) | nt8-execution-adapter + ai-client-and-providers | 42 | 9436 | 397 |
| [14](reviews/14/report.md) | nt8-wire-and-addon | 20 | 10442 | 306 |
| [15](reviews/15/report.md) | persistence-and-config | 41 | 10001 | 382 |
| [16](reviews/16/report.md) | persistence-and-config | 43 | 10006 | 458 |
| [17](reviews/17/report.md) | research-tools | 106 | 8052 | 278 |
| [18](reviews/18/report.md) | research-tools | 106 | 8052 | 252 |
| [19](reviews/19/report.md) | runtime-operations-and-support | 95 | 12529 | 407 |
| [20](reviews/20/report.md) | trader-admission-and-orders | 26 | 7375 | 217 |
| [21](reviews/21/report.md) | trader-planning-and-context | 15 | 6816 | 197 |
| [22](reviews/22/report.md) | trader-runtime-and-positions | 26 | 7466 | 198 |
| [23](reviews/23/report.md) | trader-runtime-and-positions | 27 | 7481 | 227 |
| [24](reviews/24/report.md) | web-plan-and-guide | 61 | 12988 | 201 |
| [25](reviews/25/report.md) | web-shell-chart-and-chat | 54 | 12309 | 205 |
| [26](reviews/26/report.md) | web-shell-chart-and-chat | 59 | 12309 | 235 |
| [27](reviews/27/report.md) | web-trading-and-settings | 17 | 9463 | 123 |
| [28](reviews/28/report.md) | web-trading-and-settings | 16 | 9472 | 110 |
| [29](reviews/29/report.md) | independent-control-boundaries | cross-review | bounded excerpts/diff | 96 |
| [30](reviews/30/report.md) | independent-end-to-end-and-graph-validation | cross-review | bounded excerpts/diff | 32 |

[Core trading source trace](CORE-TRACE.md) pins selected production declarations to their exact repair commit. It is a source locator, not runtime coverage.
