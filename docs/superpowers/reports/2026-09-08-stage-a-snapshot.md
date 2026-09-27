# Stage A — research snapshot

Status: **Stage A startup and live recording verified on 2026-09-09 at revision `954f11b1`.** Machine reset booted PID 438 at 13:02:06 CT with `schema=1` and goldens PASS. Live candidate, cut, completed author attempt and deterministic export are proven below. A repair attempt and measured live admission p50 remain unproven.

Lane `stage-a-snapshot-96604090/root[unlisted]`; branch `fix/stage-a-snapshot`; accept base `cd5b9a6b9c479eae97eced7fb0abb40594bf1ac5`. The claim's `ls-remote` receipt was `af1ded7e2c0af427b7268ff2fb76cbf20b4997ca`. Source freshness for every cited repository path is in [source-freshness.csv](2026-09-08-stage-a-snapshot-data/source-freshness.csv).

## Governing scope

The governing research is `docs/superpowers/research/2026-09-08-trading-policy/README.md` at **982091d4d908f4a5b8b65022cedf5b8c7c8202d5**, verified **67,959 bytes**. Section **08 Stage A** requires five evidence objects and preservation of actual-contract price scale. Sections **02, 05 and 06** explain the frozen evidence, component/selection provenance and four clocks. Section **09** forbids prescribing target, confirmation, confluence or TTL policy from that evidence. This wave introduces no market belief, trade policy, new refusal or scoring change.

Audit protocol: `AUDIT-CHECKLIST.md`, Part 2 R1–R10; checklist number assigned only at merge. [A] means directly read/executed, [B] inference, [C] unproven. Data reads use SQLite `mode=ro`, `query_only=ON`; the census was frozen within one read transaction. Counts name their retained row IDs; they are not trade or profitability statistics.

## Running revisions and source freshness

[A] At **15:12:33 CT**, `/api/health` returned `33672fdd2cd2`; `/proc/3260027/exe` reported full revision **33672fdd2cd2fee60a2c562a9693e06ab3b13551**, `vcs.modified=false`. The initial audit used an isolated worktree of that running source; later offline parity generation added only a temporary test fixture there.

[A] While this audit ran, the plan-liveness deploy owner booted the combined candidate at **15:17:31 CT**, PID **3566770**, revision **f8bc7044cc44d58e84904a0a7761e78b420404af**, `vcs.modified=false`; its observed BOOT INTEGRITY line says goldens PASS. This is not a Stage A boot. The Stage A accept base already contains that code; the refreshed dev tip remained `cd5b9a6b`. The candidate/score/ordinal/store and NT8 source files cited below are byte-identical between the initial running revision and this branch. Planner and confirmation code additionally contains the separately owned combined-wave changes; that difference is named where relevant.

## C1 — empty components and discarded propensity

[A] Current census: **936 candidate rows, IDs 1–936, across 39 reads**. All **936** have `score_components='{}'`. **469 cut rows**, IDs listed individually in [census.json](2026-09-08-stage-a-snapshot-data/census.json), all have `score=0`, empty grade, and the same reason **`max_levels: 12 seated, cap 12`** (cut ID extrema 10–931). The dispatch's 600/181 figures are not today's sample. The pinned planner audit itself describes **600 rows / 301 excluded**, not 181; no historical denominator is silently substituted.

[A] `trader/auto_trader_planner.go:2355–2363` strips `[]ScoredLevel` to each embedded `DetectedLevel` before `recordDetectorOutputs`. `kernel/detector_recorder.go:43–49` retains scores only for the final seated rows; `:70–73` initializes components to `{}`, and the cut branches at `:83–90` infer a cap reason from the final table size. `trader/detector_record.go:108–126` persists that result. The zero cut score is lost evidence, not a computed poor score. The intermediate pool is already capped/collapsed; it does not retain every original detection.

## C2 — scoring inputs that survive

[A] `DetectedLevel` (`kernel/levels.go:70`) carries bounds, origin date, detection TF, zone pattern and formation time. `ScoredLevel` (`kernel/levels_score.go:24`) adds grade, freshness label, score, confluence, distance and role. The actual score uses type/zone evidence, freshness, capped family confluence, size and HTF/TF factors, followed by grade overrides, cluster collapse and seating passes (`kernel/levels_score.go:430–578`). `candidate_pool` retains price/kind/label plus the final seated score/grade and an observed weakest-seated threshold. It does **not** persist bounds, pattern, TF, formation, freshness, family count, raw/capped factors or applied overrides. Its `rank` is nearest-first seated output position, not a pure score ranking. No factors may be reconstructed later and presented as captured inputs.

## C3 — ordinal premise corrected

[A] The claimed per-read reset is **not reproduced as the current call path**. `trader/detector_record.go:83–87` already passes the episode's own CME session day into `store/touch_outcomes.go:125`, which reads the stored maximum; formation-aware de-duplication precedes writes. Current table: **3,093 rows, IDs 1–3093**, of which **743** have ordinal 1. The old 677 rows remain classified **254 invalid:duplicate + 423 legacy:unverified**; their ordinal-1 counts are 254 + 217 = the historical 471. Newer rows include **2,287 unverified:no_formation** and **129 valid** rows, with 251 and 21 ordinal-1 rows respectively. These validity classes must not be pooled into a certified ordinal sample.

[A] Counterexamples to “every read resets to one”: valid row IDs **733–737** carry ordinals **2, 3, 4, 5, 6** for the same 29539.375 level and formation stamp. Row **739** carries ordinal 2 in a different session day. All IDs and event stamps are retained in the census. This is evidence of the corrected caller, not certification of every historical ordinal. Stage A does not modify the detector or ordinal rule.

## C4 — authoring provenance

[A] **56 `planner_read_facts` rows, IDs 1–56**: all have `version=0`, `tokens_in=0`, and empty `bias_ai` / `bias_tree`. The “regime word and little else” description is incomplete: void list/count, ATR, stop floor/multiplier, scope start/interval/bar count and a hash also survive. `persistReadFacts` (`trader/auto_trader_planner.go:2446–2481`) populates these, but its `PromptHash` is assigned `in.AIConfigHash`, not a hash of the exact submitted prompt. It has no accepted-version update or per-attempt timeline.

[A] The author/repair/resend paths exist in `runPlannerReadCoreWithFactsGradesClock` (`trader/auto_trader_planner.go:1470`). Validator rejects have attempt, reason and prompt rows; provider failures intentionally skip that store. Accepted versions have model/prompt/config identifiers and normalized plan output. These stores do not form a complete exact-input → attempts → accepted-version chain.

[A] The four cited durations are directly present in the pinned audit's [retained planning log](2026-09-08-stage-a-snapshot-data/frozen-planning-log.txt): **378.0 s at 20:47:57 CT**, **683.7 s at 21:56:43**, **423.3 s at 22:03:44**, and **683.4 s at 22:16:50**, all September 7. A fresh query of the current journal for that period returned no matching retained lines; therefore this is verification of the frozen artifact, not a claim of newly observed calls or exact recoverable request-start clocks. The failed/repair fixture will retain both durations and the reason without inferring an unobserved time from rounded elapsed text.

## C5 — four clocks

[A] `bars` has only symbol, timeframe, open-time stamp, OHLCV and convention. It cannot answer receipt/availability time; overwrites do not preserve prior versions or their correction times. Candidate `read_at_ms` identifies a read instant, while `created_at` is the DB writer's clock, not a source-event clock. Planner fact rows have only their writer's creation time and scope start; scope start is not receipt time. Plan `created_at` and confirmation's window reference previously stood in for market-history boundaries. The combined wave now adds explicit confirmation evaluation/reference evidence and authored validation; Stage A must record those existing results rather than reinterpret them.

[A] A concrete conflation is the candidate path using a read stamp while dropping `DetectedLevel.FormedAtMs`; a new DB creation stamp cannot establish when the historical bar/level became available. Another is windowing level history with `cur.CreatedAt` at `trader/auto_trader_levelstate.go:120–124`. These are different clocks, even if their values sometimes coincide. No missing source timestamp will be filled from another clock.

## C6 — actual-contract scale and remaining unknowns

[A] The live subscription receipt at **15:19:04 CT** reports **MNQ 09-26** (and separately ES 09-26) in [nt-symbols.json](2026-09-08-stage-a-snapshot-data/nt-symbols.json). `VLContractResolver.cs:51–55,129–150` uses quarterly expiries, third Friday minus eight days, and the condition `rollDate >= today`; `.c.*` aliases are stripped to a root before resolving a request contract. `VLBarsSubscriptionManager.cs:177` resolves the instrument, emits its full name, and constructs a `BarsRequest` at `:356`.

[A] The stored `bars` key is root symbol/timeframe/open time, with no per-bar contract or adjustment-policy field. The inspected AddOn request does not explicitly set a merge/adjustment policy, and the wire does not supply one. Therefore **historical per-bar contract and back-adjustment status are UNKNOWN**. A current subscription contract cannot certify every historical bar's actual contract or orderable price scale. Stage A will retain the received contract and its basis where known, and separately expose these unsupported historical fields as NULL with reasons. It will not change roll handling or C# behavior.

## Implementation and proof status at audit checkpoint

Implemented: separate archive, five-object writers, read-only export and production mutation pins. Class 93 is assigned at integration. Full clean-clone suite/build and owner-controlled cutover/live receipts remain separate pending steps. No Stage A trading decision or runtime setting is changed by preparation.


## Implementation checkpoint (not deployed)

The original scorer→bare DetectedLevel→recorder pin failed with cut score 0/empty
grade and `{}` components, then passed after internal capture metadata survived that
seam. No score multiplication order, grade boundary, seat rule or prompt JSON changed.
Targeted existing scorer/cluster/filter tests also pass. New foundation mutation pins
failed for fabricated NULL-as-zero, receipt bound to observation, suppressed drop
count, and disabled admission budget; restored foundation passes `go test -race`.
The initial receipt mutation accidentally changed the null-count loop and survived;
that result is rejected as evidence. The corrected mutation changed the INSERT
argument and failed with a missing late-backfill row in the receipt range.

This checkpoint wires the archive before readers, candidate universe capture,
input/attempt/publication hooks, existing scenario metadata, and wire receipts.
Production removal pins, full coverage of normalization/permission/outcome fields,
whole-read latency measurement, mutation receipts, full merged suite, Guide binary
revision, deployment and live evidence remain pending. Do not read compilation as
proof that the five objects have appeared live.


## Recording coverage and NULL semantics

Schema 1 uses a separate `<DBPath>.research.db`, one append-only `research_facts`
table and five object tags. The four nullable SQL columns are `observation_ms`,
`receipt_ms`, `publication_ms`, `permission_ms`; CT strings are derived only on
export. `captured_ms` is the archive write time and never substitutes for them.
The complete field registry is `researchsnapshot/fact.go`; each registered payload
field is present as a value or JSON null (SQL `json_extract` returns NULL), with a
parallel missing-reason map. Genuine numeric zero and computed empty arrays survive.
The null counter counts registered top-level fields and four clocks, not every
nested input-document property. `dropped` counts batches, not an inferred row count.

- **Market:** received MNQ bar event/receipt clocks, raw source close stamp,
  canonical open stamp, OHLCV with wire-presence preserved, finalized/forming by
  close boundary at receipt, preceding subscription contract/build with their
  basis, and changed prior observations within a bounded connection-local window.
  Missing intervals, bid/ask/spread, roll-policy metadata and historical adjustment
  status remain NULL; the wire supplies no authoritative per-bar contract policy.
- **Candidate:** pre-deduplication universe, deterministic identity with explicit
  formation-alias caveat, raw origins/bounds/timeframe/pattern; components assigned
  where the scorer uses them, raw/capped confluence, grade and seating overrides;
  actual cut point and candidate identity. An owner-supplied grade does not invent
  a computed score. Existing detector episode results are recorded separately with
  the same read and candidate IDs; excluded candidates without a detector run have
  NULL prior episodes. Existing ordinal/detector decisions are untouched.
- **Plan:** exact PlannerInput value, snapshot ID, model and available configuration
  fingerprint; every outer author/repair/provider-failed/resend attempt, exact prompt
  SHA-256, raw response, elapsed call duration, verdict/reason, final stored output,
  normalization before/after pair and published version. Actual request reasoning
  settings/token cap are captured beside request construction. Provider-internal
  subrequests, full credential-bearing client configuration, and uncaptured token
  usage are not fabricated from estimates or default zeros.
- **Scenario:** published ID/version, ordered confirm/confirm2 objects, target path,
  authored arm geometry/invalidation, existing confirmation metadata and activation
  evaluations with reasons. EntryGate's returned verdict is recorded separately
  from activation. Composed placement risk is recorded in points with its arithmetic
  basis; it is not claimed as filled-position risk. A missing declared expiry or
  predicate event clock remains NULL, never publication time in disguise.
- **Execution/outcome:** sanitized received order/fill/book frames, individual book
  order semantics and present prices, transport and arm-placement linkage, actual
  placement timeout writes, and closed-position source facts. Transport is not
  acceptance. h1 does not supply accepted payload age or rejection check reasons.
  Corrected P&L is the only outcome money column; NULL stays UNRESOLVED. Pre-era,
  UNRESOLVABLE and test-seam exclusions are named. Default-zero fees are not promoted
  to measured costs. No common experiment horizon or simulation assumption is chosen.

The legacy trading tables are **untouched by migration**: recomputed 0,
unrecomputable conversions 0, untouched historical candidate IDs 1–936 at audit.
New captures correct prospective scored rows; the new research export is the only
research input. Old generic exclusions/empty components cannot become original
measurements through a backfill. No migration of data.db is performed by Stage A;
a deployment backup is still required before normal application initialization.

## Validation receipts so far

[A] The unchanged-source scorer artifact is 685,898 bytes across **64 fixtures**
(4 grade settings × 4 seat caps × 4 freshness inputs, 72 generated candidates each).
Current seated/pool JSON is byte-identical to the artifact generated from running
source `33672fdd2cd2fee60a2c562a9693e06ab3b13551`. Capture metadata is JSON-excluded.
No formula, multiplication order, grade boundary, seating result or prompt content
was changed by the record. The full old/new source comparison artifact is
`kernel/testdata/stage_a_score_legacy.json`.

[T] Three repetitions of 1,000 identical 72-candidate scorer calls: baseline
86,507 / 77,521 / 80,335 ns/op; captured 108,095 / 109,224 / 107,572 ns/op.
Median increment **27,760 ns = 0.027760 ms per scorer invocation**. This is an offline
incremental measurement, not a measured whole live planner read. Queue admission
has a 5 ms budget, select/default on full capacity, and its own measured histogram;
JSON and SQLite run on the worker. Removing the nonblocking admission fails with
“offer blocked behind the writer beyond admission budget”. Whole live read cost remains
unproven; the later sequential production-assembly benchmark below measures offline incremental cost; boot admission p50 must not be relabelled
as an end-to-end model-call or whole-read duration.

[A] Exact-line mutations failed: removing `l.Research.Score = scoreValue(score)`
loses cut propensity (0 instead of scorer-supplied 1); replacing the candidate's
reason with `shared cap reason` loses the candidate's price; binding receipt to
observation loses the late-backfill record from its receipt range; removing
`researchTrace.Reply(raw, err)` loses both 683,700 and 423,300 ms durations from the
real retry/publication path; removing each candidate/TCP/permission/outcome hook
produces zero corresponding rows. The repair fixture publishes version 2 and keeps
both attempts and the parser rejection. Mutating manifest count validation accepts
an altered count and fails its pin. Exact changed lines and full failures are in
the committed evidence logs in `2026-09-08-stage-a-snapshot-data/`.

[A] Restored focused tests passed with `-race`. Web preliminary validation:
52 test files / 369 tests passed; `tsc --noEmit` passed. The first full Go attempt
found two newly added placement calls dereferencing an already-value arm row;
corrected to pass the value. Focused production pins then passed again. This is
not yet a full merged-HEAD suite receipt.

## Live surfaces and rollback

[A15] No Stage A boot or row exists in the running bot yet. Existing candidate
history still contains unrecoverable empty components and generic cuts. Research
cannot reconstruct historical receipt availability, adjustment policy, exact
provider-internal subrequests, missing token usage, omitted h1 rejection reasons,
accepted payload age, measured commission, or an unchosen common horizon. These
fields remain NULL with reasons. A green fixture does not make them live evidence.

Rollback restores the verified preceding binary, its RELEASE and matching dist
under the normal five-leg gate. The separate research archive can be retained as
immutable evidence; the old binary does not read it. No trading schema rollback
or deletion of research records is needed. No worker drop can release an arm slot,
promote an order, authorize a scenario, or change a gate verdict.


### Second validation checkpoint

The full Go run at `0babd090` failed only the guarded SYSTEM-MAP stop-entry
coordinates after recorder call sites moved the source. Those references were
refreshed from the current symbols; the guard passed. A separate broker-price pin
now refuses to label an unlinked fill or protective book price as an entry:
raw price/semantics remain retained, while entry-specific fields remain NULL.
SQL NULL-versus-zero and startup-registration mutations fail explicitly.

The per-read input/candidate/attempt/publication adapter benchmark (three runs of
1,000 groups, bounded queue) additionally measures microsecond admission overhead;
full results travel in the evidence bundle. It does not include model/network time
or async SQLite work. Admission p50 is rendered as a microsecond-bucket upper bound,
never a fabricated exact zero below measurement resolution.

At 16:35:28 CT the Stage A lane acquired the free merge/build lock and started an
independent keeper. Main was porcelain-clean on dev `c98ed6f2`; scenario-economics
had merged its contract and class 92 and released the lock. Stage A must integrate
that tip and reread its schema before the final merged-head suite. No swap or
restart is authorized by this preparation record.


## Integrated candidate and field dictionary

[A] Integrated scenario-economics through `c98ed6f2`, merge `13017618`. Its new-authoring contract is owned by `fix/scenario-economics`; Stage A records the final JSON (including economics) without adding validation or altering it. Contract freshness: `d5e2414e0d30a275b6239f5d609f9e22f8e381d7 2026-09-08T16:02:39-05:00 feat(scenario-economics): enforce new authoring contract and preserve legacy unknowns`.

[A] Class **93** assigned after highest **92** in the fresh two-format `sort -n | uniq -c` census, retained in `checklist-at-merge.txt`. Existing duplicate 75/76/77 entries are unchanged. Restored merged integration tests passed for researchsnapshot, kernel, provider/ninjatrader and trader. Guide removal mutation also failed before its restored GREEN.

The [complete field dictionary](2026-09-08-stage-a-snapshot-data/field-dictionary.md) enumerates every registered field, its meaning and NULL semantics. Dedicated predicate timestamp projection, full config/prompt-version, legacy row links and accepted entry role remain NULL; relevant raw metadata/output/order facts remain available separately. The per-read adapter benchmark is 0.002532 / 0.002161 / 0.002655 ms (median 0.002532 ms) on its one-candidate fixture; it is not added to the separate 72-candidate scorer benchmark as a measured whole-read number.

A15 remains explicit: whole live per-read overhead, first real candidate/cut/attempt/repair and the first live export are **NOT YET PROVEN**. Queue admission is nonblocking and measured separately. The 5 ms admission test is a code budget/drop guard, not a real-time operating-system scheduling guarantee.


## Final candidate preparation receipt

[A] Source `2a96cf63278f7c836d55e14c5a02597f80280292` is the Stage A merge including scenario-economics and clean-machine documentation. Both `dev` and `fix/stage-a-snapshot` were verified by `git ls-remote` at that revision before clean-clone validation. Clone: `/tmp/stage-a-build/nofx`. Full `go test ./... -count=1` **PASS**, including prompt goldens and guarded source references; Vitest **54 files / 378 tests PASS**; `tsc --noEmit` **PASS**; final focused recorder/producer `-race` **PASS**. Exact receipt hashes are in `validation-receipt.json`; successful Go/race outputs are retained.

[A] Binary: **72,454,640 bytes**, SHA-256 `c94ee81fbe364bee2b7b9352b2c706918da8bcac49bc781895de6466872e211b`, **vcs.modified=false**. Its actual embedded revision was read into `GUIDE_BUILT_REV` at **16:49:46 CT**, THEN dist was rebuilt. Dist: **92 files / 7,918,497 bytes**, manifest SHA-256 `925919f3063014f8884604ba52fb0356f07967d17da7831de20e1037d342cf15`; its JavaScript contains that same full revision. See `candidate.json` and `dist-manifest.json`.

[A] Own preparation gate at **16:45:40 CT**: all five legs PASS, no open position, no working order, no planner read claimed. Leg 4: **broker — NT8 order_snapshot frame (age 9s, build 2026-09-07-h1)**, broker and ledger both zero. This receipt is stale for cutover. A7 prohibits **16:45–17:10 CT**; next window is **after 17:10 only if flat, no arms, no open position**, with a fresh five-leg gate/in-flight read. A3 still requires owner GO before RELEASE/swap. No kill was issued.

[A] At **16:47:41 CT**, PID **3566770** and the disk binary both held `f8bc7044cc44d58e84904a0a7761e78b420404af`, modified=false. Preserved actual running image `/tmp/stage-a-cutover/backup/nofx-bin.old.f8bc7044cc44d58e84904a0a7761e78b420404af`, SHA-256 `e2c2ce8602ca61e52d180309743593b3bf538337693e4f2c61ae83c21d457918`. Online backup `/tmp/stage-a-cutover/backup/data.db`: **751,267,840 bytes**, **integrity_check=ok**. Stage A does not migrate existing trading rows.

### Measured complete input-assembly overhead

[A] Sequential offline benchmark of the **actual `assemblePlannerInputWithCtx` call**, before Stage A at accept base `cd5b9a6b` and after at candidate source `2a96cf63`. Identical synthetic 300-bar fixture, temporary legacy DB each run; current side also installs the real separate SQLite research worker. Three runs × 100 calls each, executed sequentially after other builds/tests finished. Mean costs per run: baseline **21.042842 / 19.141682 / 21.324307 ms**; current **21.628321 / 23.165777 / 23.515908 ms**. Difference between medians of those three means: **+2.122935 ms per input assembly**. This is not a sample p50, not live latency, and excludes subsequent provider/network time.

The deliberately continuous 100-call bursts dropped **573 / 560 / 565 batches** on the current side; these are measured recorder batch counts, not lost-row estimates. Production cadence throughput is not inferred from this overload fixture. Queue saturation did not wait for SQLite. The prior admission mutation separately proves that replacing select/default with a blocking send fails the stated latency pin. Benchmark sources, both complete logs and summary are retained; the temporary benchmark source was removed before publishing the metadata receipt.

### Current production sources and wiring

| Capture surface | Source at candidate `2a96cf63` |
|---|---|
| Scorer capture | `kernel/levels_score.go:513` |
| Raw candidate assembly | `kernel/levels_assemble.go:158` |
| Candidate/input/permission/placement/outcome adapters | `trader/research_snapshot.go:20` |
| Outer retries and publication | `trader/auto_trader_planner.go:1479` |
| Received broker/market frames | `provider/ninjatrader/research_wire.go:21` |
| Bounded admission and worker | `researchsnapshot/recorder.go:45` |
| Append-only schema and export | `researchsnapshot/archive.go:25` |
| Pre-reader registration | `main.go:76` |
| Read-only export CLI | `cmd/research_export/main.go:23` |

`production-call-census.json` enumerates 87 newly defined production functions. All have at least one lexical production call except the export command's `main`, which is the Go executable entry point. Lexical method-name matches are not claimed as semantic proof; the AST writer test names enclosing callers and startup order, and actual producer-removal pins fail. The sole new table `research_facts` is written by `Archive.saveAt` via the worker and read by the export and boot counter queries. Source freshness is retained separately for the frozen audit and integrated implementation.

### Outstanding live proof, not inferred

Stage A boot line, first real captured candidate, first cut with its own reason, first author/repair timeline, first live export/checksum and live admission latency remain **NOT YET OBSERVED**. Existing NULL limitations are enumerated in the field dictionary. The confirmation-truth watcher still has no first bucket/forming/out-of-order proof at this preparation checkpoint; its earlier combined boot is not a Stage A receipt. Marker and lock release follow a passed owner-controlled boot, never a prepared build.


## Startup failure and repair (same Stage A wave)

[A] The combined deploy owner booted `6f677b55daa1c7da33b8c35f8bcc67883f36b470`, PID 3726840, at **18:11:54 CT**, clean VCS stamp and goldens PASS. The actual Stage A line was:

```text
🗄 research snapshot: schema=UNKNOWN · objects=5 · rows today market=UNKNOWN candidate=UNKNOWN plan=UNKNOWN scenario=UNKNOWN exec=UNKNOWN · null-fields=UNKNOWN · dropped=UNKNOWN · added latency p50=UNKNOWN
```

That is a failed research initialization, not a live record. The scenario-economics line separately reported contract=on / obstacle-required=on and legacy UNKNOWN by design. Both boot frames and source revision are retained in `2026-09-08-stage-a-path-repair-data/failed-live-boot.json`.

[A] The scenario-economics lane's retained reproduction opened the absolute path successfully; relative `data/data.db.research.db` became `file://data/data.db.research.db` and returned `research schema unsupported: version=0 error=SQL logic error: out of memory (1)`. Configuration uses `DB_PATH=data/data.db`; `main.go:76` appends `.research.db`. The bad URI put `data` in the authority position. The message was not evidence of RAM exhaustion. No NT8 or trading policy caused this failure.

[A] Repair branch was cut from dev `d5b27e86` and claimed as `fix/stage-a-archive-path`; remote claim receipt `9c91538e73c32d13f9370d63a4e17a57076eefa8`. Source freshness for both archive/runtime: `ccef731d251b57d9caf08c4834767a83a0f18d32 2026-09-08T16:36:00-05:00 fix(research): preserve unknown entry roles and refresh guarded source references`. Both filesystem entry points now call `filepath.Abs` before URL construction (`researchsnapshot/archive.go:28` and `:70`). URL escaping remains responsible for literal spaces, ?, # and %. No process working directory, DB_PATH setting, schema version or trading table is changed.

[A] New startup regression **RED** on the old source: `working recorder required at relative path "data/data.db.research.db"; boot line: ... schema=UNKNOWN`. The separate read-only regression was also RED with the SQLite open error. Restored tests pass under `-race`. The working-startup pin reads the real schema and persisted candidate count; the failure pin still requires a WARN and schema=UNKNOWN. Exact-line mutations removing each path normalization fail their respective pins, and replacing the default UNKNOWN with the schema constant fails the genuine-failure pin. Logs and the peer reproduction are retained in the repair evidence directory.

[A15] Until a corrected boot produces a real schema and rows, the research half remains **unshipped**. The previous candidate/latency fixtures and healthy scenario-economics counters do not prove research availability. No historical research records are recreated or represented as original observations.


### Corrected candidate prepared

Repair binary **954f11b15f2e7615678f7d2b708c47895faebf1e**, **vcs.modified=false**, built in `/tmp/stage-a-path-build/nofx` after full `go test ./... -count=1` PASS (including goldens), Vitest **54 files / 378 tests PASS**, and TypeScript PASS. Binary **72,459,208 bytes**, SHA-256 `c7c9c72321c89a56f5348cc0a9cee554b8f63633782cc6b29973e196fae2e4cf`. GUIDE_BUILT_REV was read from that binary at **18:44:06 CT**, THEN dist built: **92 files / 7,918,609 bytes**, exact manifest retained.

The repair keeper acquired the free lock at **18:37:10 CT**. Before cutover, actual running/disk revision **6f677b55daa1c7da33b8c35f8bcc67883f36b470**, PID **3726840**, was backed up by embedded revision with matching RELEASE/dist. Online `data.db` backup at **18:40:56 CT**, **753,287,168 bytes**, returned **integrity_check=ok**. No existing trading table is migrated.

Fresh preparation gate **18:44:39 CT** passed all five legs for the one running trader; leg 4 was **NT8 order_snapshot, h1, age 2s, zero working orders**. Separate active-arm census: **0, IDs []**. Applied window: after 17:10 CT, flat, no arms, no position. A further fresh gate and independent swap verification remain required before the owner restart. Under the standing owner boot instruction, this repair prepares RELEASE and the verified binary; the agent never executes the kill.

At this receipt, repaired startup is proven by the fixture, **not yet by the running service**. The first known schema, candidate/cut/attempt and export remain live proof requirements. Publication receipt, swap verification, owner restart and passed-boot marker will distinguish preparation from shipping.

## Passed boot after owner machine reset — 2026-09-09

[A] Read-only verification at **13:13:33 CT**, after the owner reported a machine reset: systemd PID **438**, start **13:02:04 CT**; journal boot lines **13:02:06 CT**. This is retrospective observation after the reset, not a claim that a watcher acknowledged within 90 seconds. No kill was issued by this lane. The former PID 3726840 kill command is obsolete.

```text
🔐 BOOT INTEGRITY OK — rev 954f11b15f2e · built 2026-09-08T23:38:16Z · expected 954f11b1 · goldens PASS
🗄 research snapshot: schema=1 · objects=5 · rows today market=0 candidate=0 plan=0 scenario=0 exec=0 · null-fields=0 · dropped=0 · added latency p50=UNKNOWN
📐 scenario economics: contract=on · obstacle-required=on · target-path-coherent=0/0 · sub-1R-first-obstacle=0 · role-use-disagreements=0 · contradictions refused=0 · schema refusals=0 · checked=0 (new-authoring checks since boot; legacy UNKNOWN by design)
```

[A] All five A19 references agree on **954f11b1**: RELEASE file, disk binary VCS revision (`modified=false`), `HEAD:deploy/RELEASE`, GUIDE_BUILT_REV, and `/api/health`. The running executable independently matches. Binary SHA-256 `c7c9c72321c89a56f5348cc0a9cee554b8f63633782cc6b29973e196fae2e4cf`, 72,459,208 bytes; all 92 on-disk dist files match the prepared manifest. Source head f416ba13 is the publication receipt around that clean binary; the later 5519d494 merge contains a separate strategy research report, not another executable build.

[A] At the 13:13:33 check, legs 1–4 passed; leg 4 was a received **h1 order_snapshot, age 4s**. Leg 5 **FAILED: plannerReadInFlight claim**. Active arms: **0, IDs []**. Therefore no fresh kill was printed or executed. The repair was already running; this marker changes documentation only.

[A] Postboot archive read transaction, scoped to `captured_ms >= 1788976926000` (13:02:06 CT), proves writes in all five objects. Counts are observations, not unique market bars or trading decisions:

| Object | Rows | Stored ID extrema |
| --- | ---: | --- |
| candidate | 1651 | 54982–224363 |
| exec | 50 | 1–238014 |
| market | 241180 | 4–242898 |
| plan | 10 | 55783–224366 |
| scenario | 7 | 211926–223510 |

Census read at `2026-09-09T18:19:22.322386+00:00`; precise query scope and selected IDs are retained in [passed-live-boot-20260909.json](2026-09-08-stage-a-path-repair-data/passed-live-boot-20260909.json).

[A] First scored candidate **54995**, receipt **13:08:08.691 CT**, event `planner_read`: price 29424.5, grade A, final_score **1.92**, seated. Raw evidence=1, fresh multiplier=1, confluence raw=4/capped=3/multiplier=1.6, HTF multiplier=1.2; recorded display overrides explicitly leave the score unchanged. Unsupported components remain NULL. First cut **54994**, same receipt: `PDH 29739.50 [PDH]: proximity distance=283.75 exceeds band=245`; raw components and score are NULL, not fabricated zero.

[A] First completed author attempt **211923**, receipt **13:17:32.483 CT**, event `attempt_verdict`, attempt=1, mode=author; recorded start=1788977288694, end=1788977852480, duration_ms=558090.835295. These clocks and duration are quoted independently as stored; no equality or provider-only latency is inferred. Rejection reason and plan version on this particular frame are NULL. **No repair-mode attempt was observed in this census**; that proof remains pending.

[A] Two successful read-only exports of receipt interval **13:08:08–13:08:10 CT** were byte-identical: **3,260,626 bytes**, SHA-256 `880ae1d9c3e5de730337358217153a8ebfac3807bececc085895bf9bc72032d2`. The production exporter validates the bundle before emitting it. Counts: market=887, candidate=801, plan=3, scenario=0, exec=1; unknown receipt exclusions=0; sole writer=954f11b15f2e7615678f7d2b708c47895faebf1e. The empty scenario array is valid for this narrow interval; the broader census above separately proves scenario capture. Export receipt and object checksum are retained in [live-export-20260909.json](2026-09-08-stage-a-path-repair-data/live-export-20260909.json).

[A] Boot p50 remains **UNKNOWN** because the quoted boot frame has no admission measurement. The earlier offline +2.122935 ms comparison is not substituted for a live statistic. No claim of lossless production recording follows from dropped=0 at startup.

Provenance: Stage A `fix/stage-a-snapshot` supplied recording implementation; this lane/session `stage-a-snapshot-96604090/root[unlisted]`, branch `fix/stage-a-archive-path`, supplied repair **954f11b1** and publication **f416ba13**. Scenario-economics supplied its separately documented contract changes and combined clean-build clock seam **6f677b55**. This lane built and gated the combined source before the September 8 swap; it did not author all constituent waves. Existing candidate suite/build receipts remain above. This passed-boot marker changes only report/evidence files and requires no new executable or restart.

Lock continuity: the machine reset removed the temporary worktree and heartbeat process. The same named owner resumed after verifying missing directory, preserved local/remote repair branch f416ba13, clean main f416ba13 and the running 954f11b1 boot. Only its missing worktree registration was restored. Automatic review initially refused combined recovery until ownership and loss checks were demonstrated, then approved the scoped recovery. The marker must be pushed before this owner releases the lock; release is a subsequent operation, not asserted in advance here.
