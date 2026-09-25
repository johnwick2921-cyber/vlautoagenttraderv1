# Assignment 19 — runtime, operations and support source review

Base: `63968be62e44db2fb07a92883e02127b9064b0be`. Worktree: `/tmp/nofx-understanding-market-20260913`; verified revision and initially clean porcelain. This is source understanding, not a production certification. All 95 assigned files (12,529 lines) were read in full. `reads.json` records per-file hashes/ranges/semantics; `functions.json` contains 407 named-function/method entries (279 Go declarations, Python functions, shell helpers) with 58 Go literal callbacks grouped under their owning declarations. Shell helpers inside fixture heredocs are fixture text rather than production declarations. No deploy, restart, process control, live DB read/write, account mutation, external message, or network probe was performed. No tests were executed.

Evidence: **[A]** code directly read at the base, **[B]** consequence inferred from that code, **[C]** unresolved hypothesis. “Static risk” below means an identified implementation concern, not a reproduced exploit or runtime trading failure. Root reports boot-integrity ordering repaired separately; the base and this review remain immutable.

Rules consulted: supplied main AGENTS instructions, tracked `docs/superpowers/CLAUDE-canon.md`, AUDIT-CHECKLIST pre-audit R1–R10, cutover procedure and lock classes101/102, SYSTEM-MAP runtime/Stage A sections, corrected RULEBOOK scope. The daily-loss owner correction is authoritative; this review proposes no additional per-trade cap. Latest base-local document history:

- CLAUDE-canon: `07b53e65 cleanup batch 1 — B5 the mutation harness that cannot fake a verdict, B6 the worktree-add check`.
- AUDIT-CHECKLIST: `dfda15e1 test: isolate session clock fixtures from weekly backfill workers`.
- SYSTEM-MAP and RULEBOOK: `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap`.

## Runtime wiring and ownership

[A] `main.go:40` loads dotenv, initializes logger/config, constructs mandatory RSA/AES service and installs the global crypto hook **before** store reads. SQLite path can be overridden by argv; directory creation failures log but do not immediately abort. `researchsnapshot.Start` at main:77 initializes a separate sidecar using `DBPath + .research.db`, even when the trading database is PostgreSQL. `store.NewWithConfig` then initializes schema/defaults. WARN+ log shipping attaches only after the store exists; earlier warnings are journal/file only.

[A] Boot also performs historical correction/link/seam/PNL migrations and installation-ID persistence. JWT is configured at main:150; the log proves assignment, not strength. Acceptance-rule repair runs before trader loading. Order-snapshot persistence hook is registered before `LoadTradersFromStore` at main:251, because lazy TCP server creation captures this hook. Hook copies broker account/build/orders, working count and local receipt time into the store; insert failure warns without undoing cached broker truth.

[A] At this base, trader loading precedes `kernel.AssertBootIntegrity` at main:289; the adjacent comment claiming “before any trader cycles” is not supported by order alone. This is the root-owned separately repaired finding, not another fix request. Likewise role-map application and many record-only boot backfills occur after trader load. Boot reporting calls resolved kernel/trader/store functions for prompt geometry, state gates, session risk/calendar, contract-related data and read-model counts. Several expensive maintenance actions are environment-gated and backup-first (E8/regrade/waveA); episode/fade/one-setup backfills run as boot work. These are reasons a normal application launch is not a read-only probe.

[A] Sandbox synthetic bars are installed late in boot at main:640. API server receives manager/store/crypto/configured host/port; web agent is registered and started. API and Telegram use bare goroutines. SIGINT/SIGTERM causes HTTP shutdown, `StopAll`, deferred agent/store/research cleanup and a normal exit. `deploy/nofx.service` sets `Restart=on-failure`, so ordinary graceful termination does not request restart. Units are templates with actual paths substituted by the installer; installed units were not examined.

[A] `config.Init` defaults HTTP to127.0.0.1:8080 and SQLite data/data.db; insecure JWT fallback is now explicitly warned. Setting an off-loopback host warns but does **not** refuse the insecure secret combination. Transport encryption defaults false. Experience telemetry defaults true. NT account allow-list is parsed without changing accounts; trading mode defaults crypto; dormant/notional fields are labeled in source. Numeric env parsing accepts values without domain checks, including float NaN/Inf. Whether downstream consumers reject each invalid value is outside this slice.

## Priority source findings

### A — tools labeled read-only can initialize/write the database

[A] `cmd/bars-export/main.go:34`, `cmd/gate-jwt/main.go:48`, dayplan arm/repair/sessions preview paths, and excursions reporting call `store.New`. The dependency was read: `store/store.go:63` calls `initTables` and `initDefaultData`; :144 creates tables and :242 migrates excursion zeros, while :249 initializes defaults/migrates equity. **“Dry-run” only gates the command's explicit business update, not constructor side effects.** Bars-export's “never writes” claim is false as a call-path contract. Detector-report requests `?mode=ro` yet calls the same initialization path, so it can fail during attempted schema work rather than be a clean readonly opener. `cmd/research_export` correctly uses `OpenReadOnly`; decisive-test uses read-only GORM plus `NewFromGorm` (no constructor migration), but is network-capable.

[B] A maintenance preview can change schema/defaults or legacy rows, or fail on a read-only DB. None was run against production. Scope and backup authorization must be resolved before execution; these comments are not evidence of harmlessness.

### A — obsolete executable cutover and install surfaces bypass current procedure

[A] `deploy/leveltruth-cutover.sh:9` pins a historical build SHA; :17 checks DB open positions and :22 accepts **any** count=0 substring in two journal snapshots, without current account freshness, full five-leg cutover gate, ownership lock or branch/clean verification. :30 writes RELEASE before checking replacement binary; :36–39 swaps/may kill with only `set -u`; :44 accepts an unbound recent integrity line, potentially from the old boot. Timeout does not roll back. The header says manual-only and no timers; that is an operating instruction, not enforced owner acknowledgment. Current checklist five-leg gate is much stronger.

[A] `install.sh` and `install-stable.sh` still download unpinned `NoFxAiOS/nofx` Compose assets, default installation directory to `$HOME/nofx`, overwrite Compose, and start containers. `install.sh:171` optionally deletes all three trading tables with no backup or WHERE scope. Generated env files lack explicit mode600 and use Asia/Shanghai. `start.sh:305` does git pull and rebuild/start, while restart/clean/key-rotation bypass current deployment gates. `start.sh:43` tests `command -v docker compose`, not `docker compose version`, so presence of docker does not establish the Compose plugin. These are legacy capabilities, not recommended current NT8 operating paths.

[A] `railway/start.sh:9` generates new AES/RSA keys whenever env keys are missing; they are process env only. Persistent ciphertext therefore depends on provisioned stable keys across restarts. nginx `/health` always returns200 and background backend is not supervised; `tail -f /dev/null` can keep container “healthy” with backend dead. This script does not generate JWT_SECRET. No installed/deployed usage was verified.

### A — lock helper and documentation retain competing writer paths

[A] `deploy/nofx-lock.sh:94` atomic mkdir protects acquire; :132 starts a bounded detached keeper; :261 validates session and expiry at heartbeat write; :489 owner-scoped release verifies directory removal. Keeper signals are guarded by numeric handle, lock path in command line, and group-leader check. Incomplete locks have separate states3/4; stale is explicitly not proof of abandoned ownership. These mechanisms are substantially stronger than the obsolete main instruction mirror.

[A] However `cmd_with_heartbeat` at :428 still invokes a foreground beat and spawns **another** periodic writer while acquire's keeper exists. It kills only its own beater shell and does not register that writer with release. This contradicts tracked canon's “do NOT hand-beat — a second writer” rationale, even though canon still advertises this wrapper. [B] An in-flight secondary heartbeat is outside `_stop_keeper`'s group containment; the historical orphan-write class merits focused concurrency reproduction, not a claim of an observed takeover.

[A] `cmd_reclaim` :459 rewrites holder/heartbeat but retains prior expiry and does not stop/restart a keeper for the successor. Reclaiming an expired lock therefore gives one fresh stamp but subsequent beats are refused. Tool text suggests “extend explicitly,” yet dispatcher has no extend verb. `_write_meta` errors can also be overwritten by following echo success because script lacks errexit/explicit propagation at some call sites. `cmd_check` ignores a legacy flat lock even though status prints one. These are static operational consistency gaps.

[A] Full lock-test source was read, not run. It tests acquire collision, expiry, group targeting, conditional delayed-writer release, incomplete states and removal failure. Its old `with-heartbeat` liveness assertion reads `$LOCK_DIR/heartbeat` even though heartbeat is in meta, so two missing-file reads can compare equal. Claim tests validate regex shape; they do not test remote transport failure. `nofx-claim.sh:56` remote lookup failure is not explicitly distinguished from no matching branch, and branch/default-data freshness is delegated to operator fetch; `claim_msg_of` does not implement its “first-parent” comment.

### A — security claims are stronger than actual guarantees

[A] `security/url_validator.go:93` rejects non-HTTP schemes, blocked names and resolved private addresses, but allows unresolved non-IP names to proceed. `SafeHTTPClient` :169 checks `net.LookupIP(host)` then dials the original hostname at :196, causing independent resolution rather than binding the validated IP. [B] DNS changes between validation/dial leave a rebinding opportunity. No DNS manipulation or exploit was performed. Public/unknown special ranges not explicitly covered by `isPrivateIP` require dedicated tests before claiming comprehensive reserved-space rejection. Safe client is used by MCP configuration (`mcp/config.go:86`, search evidence) and external data fetch (`kernel/engine.go:788`, search evidence).

[A] `crypto/crypto.go:415` Scan suppresses decrypt failure and returns ciphertext; :449 Value suppresses encryption failure and returns original plaintext. No initialized service also permits plaintext persistence. `NewCryptoService` requires both keys and main installs it early, reducing this risk for the normal server path; many CLI store openers do not initialize crypto. This is a fail-open storage contract, not proof a plaintext credential was stored. AES nonstandard-length keys are SHA256 normalized, not password-strengthened. Already-prefixed input bypasses encryption based on prefix alone.

[A] `DecryptPayload` :284 accepts TS=0; nonzero TS only checks -1min/+5min age, no replay cache. AAD content is authenticated as bytes but parsed user/session/purpose values are not checked against caller identity; outer TS is not itself obligatorily bound into authenticated AAD. Caller authentication remains relevant: `api/crypto_handler.go:67` refuses when transport encryption is off; `api/server.go:175` registers decrypt under protected routing (search evidence). Do **not** misreport this as the old unauthenticated public decryption oracle. Existing handler test at `api/security_p0_test.go:358` pins off-state404.

### A — Telegram binding, reload, token and concurrency limits

[A] `telegram/bot.go:24` consumes reload only while disabled or after `runBot` ends; runBot has no reload parameter or StopReceivingUpdates call. Therefore advertised live token hot reload is unwired. Fatal initialization returns from supervisor rather than retrying. `resolveToken` reads DB only despite env fallback comments.

[A] `runBot` chooses first DB user and binds first `/start` chat when unbound, without an owner pairing code/private-chat requirement in this file. Once bound, access control is by **chat ID**, not sender identity; group-chat semantics therefore matter. `/lang` occurs before ACL and can toggle shared awaitingLang for the authorized chat. [B] First-use ownership and group participant controls need product authorization review, not an assumed exploit.

[A] Bot JWT is generated only when resolved user ID changes; `auth.GenerateJWT` at auth:85 expires after24h. No renewal occurs for unchanged user. `Manager.Run` serializes per-chat turns with60s wait, but `Reset` bypasses lane and mutates unsynchronized memory during running turn. Agents/lanes are unbounded maps. Long-term compaction appends a summary already containing previous summary; it can grow indefinitely and retains its original LLM object.

[A] `telegram/agent/apicall.go:40` accepts arbitrary tool method/path and sends a bearer JWT to localhost; server endpoints provide real authorization. Prompt-only GET-after-write verification is not an enforcement gate. Native tool loop executes returned arguments without checking tool name. Ten-iteration cap returns “Operation completed” regardless of final API result. Body reads are unbounded. Existing agent tests mock GET/POST workflows and max iterations, not long-lived reload/renewal/ownership/reset concurrency; one test uses port8080 and can contact a real local API during account-context fetch, so this reviewer did not run it.

### A — sandbox labels do not establish isolation

[A] `scripts/sandbox-up.sh:27` explicitly opens live DB mode=ro for backup despite header saying it is never opened. It scrubs some tables but does not remove model/exchange credentials; scrub command errors are hidden and ignored. Existing sandbox.db skips copying **and** scrubbing. Sandbox process runs from repo cwd and loads real dotenv; setting empty TELEGRAM_BOT_TOKEN does not override DB token resolution. Sandbox-specific loop/arm safety gates outside assignment were not audited here; retaining secrets is independently relevant even if those gates block orders. Listener wait always ends with READY, including timeout.

[A] Seed guard at `cmd/sandbox-seed/main.go:34` checks path substrings only, not resolved symlinks/file identity. It appends plans/trades and increments tests on repeated calls despite “idempotent replaces” comment; many write errors only print/are ignored. Sandbox-down's Vite regex is not worktree-scoped. These scripts must not be used as evidence that an arbitrary copy is safe to run.

## Backup, clock, diagnostics and telemetry semantics

[A] `nofx-db-backup.sh:37` uses SQLite online backup, quick_check on copy, gzip partial then rename; first snapshot per ISO week is copied into weekly and retention keeps14 daily/8 weekly by default. It protects consistency better than raw database file copying. There is no concurrency lock, cleanup trap or retention validation: same-second runs collide; failed runs can leave partials despite header; zero/negative retention can delete every retained file; shell whitespace splitting breaks paths with spaces. It backs up only main DB, not keys or research sidecar. Timer file uses host-local05:00/17:30 without explicit America/Chicago and assumes host zone. User timer installer copies hardcoded main-path service definitions; observed installation/linger/restore efficacy remain unverified.

[A] Clock guard is detector-only: RTC drift decides CRITICAL at30s, NTP/Windows are measurements; missing RTC leaves statusOK with n/a, not UNKNOWN. Timestamp/state JSON written by temporary rename; overlapping direct invocations share a temp filename. Chrony remediation is root mutating installer; it writes config and uses enable --now, which need not restart already-active chrony to apply new config. No clock actions were taken.

[A] Logger text time uses `entry.Time.Format` without CT conversion, contradicting main's blanket “host TZ ignored” log. File name chosen at logger Init does not rotate at midnight; .log file is0644 and text formatting omits structured fields. WARN+ DB hook captures fields separately, but its sink contract is nonblocking/nonrecursive rather than enforced. Reinitializing logger after attaching hook can leave atomic attached flag referring to old logger. Fatal exits do not run normal deferred cleanup. `safe.Go` contains panics/callback panics but does not restart work; main does not universally use it.

[A] Gate/error counters are in-memory and reset on caller-supplied CME day differences, not durable accounting. Verified call site `trader/auto_trader_loop.go:251` rolls both from the kernel session clock. Error announce callback runs under errorMu; future reentrant callback could deadlock (current config callback only logs). Weekly map has no runtime rollover in its own implementation despite header, and far-arm comment incorrectly says unknown side increments denominator. Gate map includes observational/skew/shadow counts, so total is not strictly refused entries. Prometheus metrics are a separate process-lifetime surface.

[A] GA telemetry defaults **enabled**, sends installation/user/trader identifiers plus exchange/symbol/amount/leverage and model/token/channel data. Trade producer read at `trader/auto_trader_decision.go:411`; model callback in config. This is pseudonymous event-level data, not “only installation ID”; remote collector requests are bare goroutines with5s timeout, ignored responses/errors, no bounded queue. No telemetry settings or outbound service were inspected.

[A] Wallet cache normalizes key but queries original untrimmed address, no map eviction,30s positive cache. RPC missing result returns zero; display function suppresses failures as0.00, regular function propagates most parse/RPC errors. Address length/hex, status and response-size validation are absent locally. This supports crypto payment plumbing; no wallet or trade was queried.

## Record-only research and analytics

[A] Stage A archive is independent SQLite WAL, max1 connection, schema version1. `NewFact` emits known fields as NULL with reasons; typed nil/JSON null normalization preserves computed[]/0. Four source clocks remain distinct; capture time added at save. Batch Save validates then commits transactionally; malformed member rolls back whole batch. Recorder admission is bounded/nonblocking; builders execute on single worker, saves use2s contexts. A stuck builder/warning callback can stall worker and close (timeouts do not preempt Go functions), filling queue and counting drops while trading producer remains nonblocking. P50 measures admission only, not total capture/disk latency. Offer/Close interleaving may admit after stop check; no claim of lossless audit delivery.

[A] Export opens readonly escaped absolute URI, reads receipt membership `[from,to)`, counts NULL-receipt exclusions archive-wide, preserves source clocks and sorted revision set, serializes explicit empty object arrays and objects SHA256. Verify checks expected object counts and objects checksum, not authenticity, complete manifest semantics or rejection of extra object keys. PlanTrace snapshots invocation values and captures actual prompt/config/reply/verdict/publication; published does not equal permission, and composed initial risk stays unknown. Full callback inputs must already be immutable at producer boundaries; this package cannot guarantee producers do not capture mutable objects. No archive contents or coverage counts were read.

[A] Expectancy uses corrected PNL only and sample IDs for realized cells, raw-row reaggregation for condition/session/kind/path/era, Wilson win CI and normal mean CI with n30 floor. Missing condition/link/seam/unresolved rows are separately classified. Optional table read failure warns and loses those optional dimensions. The load query is database-global without trader/user parameter. API-level intended tenancy must be checked by its owning reviewer; this slice does not certify it. AsOf is latest included resolved position, not most recent excluded source row.

[A] `loadArms` keys plan/version/scenario, so multiple fills/rearms with same tuple overwrite without ordering; path/plannedRR attribution can be ambiguous. Excursion hit shares divide by every present excursion row even when exit_reason empty; unknown reason becomes false rather than unknown. `LevelKindFromLabel` longest prefix can accept unrelated trailing text. `FilterEra` recomputes realized cells but retains global exclusions and all E8 side table. E8 buckets omit Rule, preserve first observed rule label while mixing rules, have no row-ID list, and mark unknown/short direction suspect. This is a research labeling concern, not live gate behavior.

[A] Historical Python probes are **not production parity evidence**. Missed-turns uses5-row aggregation regardless of gaps, whole-session future swings as seats, latest plan and final high as proximity proxy. Stale-MET uses full bucket ATR at decision bucket (forming/future constituents), no source/contract filter, naive timestamp host interpretation and no row IDs. Synthetic MPM fixture proves its own high-first versus stop-first construction only. Position-plan repair hardcodes session windows and overwrites duplicate join keys in a map instead of proving exactly one match; writes immediately with no backup. These findings limit reuse of old measurements, not invalidate every historical result wholesale.

## Validation boundaries and remaining uncertainty

[A] Assigned claim/lock/mutation selftest sources were fully read. Narrow dependency reads cover store initialization, auth expiry, API decrypt guard, live telemetry producers and unit templates. Circuit-breaker test source fully read: it checks one allowance after cooldown but never reserves/tests exclusivity of that probe; `Allow` actually permits all callers after cooldown. Retry cancellation is checked only between failures, so an already-canceled context can still run first attempt. No tests run means no new PASS/FAIL claim.

Additional test files were discovered by names (not fully reviewed): calendar outage/filter, expectancy hand-computed cells/row IDs/era/minN/E8, research NULL/four clocks/admission/relative startup/wiring, logger WARN sink, gate metrics and discipline tests. Telegram agent tests were read in excerpts only. Root owns merged-head validation; this review does not duplicate its suite or infer green from test existence. GitHub push reportedly showed five dependency alerts (2high/2moderate/1low), uninvestigated; dependency manifests/advisories were not assigned and no dependency-security clearance is claimed.

[A] July10 graph subset contains154 nodes touching39/95 assigned files and981 incident edges; it is historical, not current. Read node summaries and selected runtime edges conflict with source: telemetry called opt-in/installation-only (false); Telegram API response called truncated (unbounded); crypto described partial keys, AAD identity validation and plaintext passthrough (not what these functions do); hot reload described wired (not in running loop); logger config described format field (absent). `graph.json` records corrections and evidenced current boundaries. CGC is historical per root; no fresh CGC query/reindex performed. AST calls are expressly syntax-only and not receiver/type resolution.

No runtime account bindings, NT8 build, deployed binaries, JWT value, current risk settings, broker truth, backups, user/strategy data or external payloads were inspected. The inventory describes code at one commit, never the safety or profitability of a live deployment.

## Complete assigned-file coverage

The following notes are from full source reading; exact function boundaries and local callees are in `functions.json`.

- **.github/workflows/scripts/calculate_coverage.py** (1–192): Go coverage parser uses authoritative total but unweighted average of function percentages per package; generates raw cover report and GitHub outputs.

- **.github/workflows/scripts/comment_pr.py** (1–246): Coverage PR comments via GitHub requests; first-page bot marker match; no timeout/pagination; fork JSON failure defaults non-fork; advisory only.

- **.husky/_/husky.sh** (1–36): Husky hook wrapper honors HUSKY=0, sources user rc, invokes same hook under sh -e, propagates exit.

- **branding/branding.go** (1–15): Embedded display-only product/persona text without trim.

- **calendar/calendar.go** (1–203): Weekly FF feed 10s/5MiB; high/medium currencies and CT dates; invalid event dates silently skipped, valid empty feed no fallback, timezone fallback UTC.

- **cmd/arm-state-sql/main.go** (1–19): Prints canonical store terminal/nonterminal arm SQL; no DB opened.

- **cmd/bars-export/main.go** (1–98): Export ladder bars to CSV uses store.New (migrations despite readonly claim), unfiltered BarsBetween, no source/contract columns or CSV write-error checks.

- **cmd/dayplan-arm/main.go** (1–83): Dry-run strategy arm preview; non-grid strategies, day-plan defaults; confirm persists full codec; store.New still mutates on preview.

- **cmd/dayplan-level-repair/main.go** (1–85): Burned-level repair cutoff preview; confirm ResetBurns to C; store.New initialization happens before confirm.

- **cmd/dayplan-sessions/main.go** (1–192): Existing day-plan session overrides tighten grades/caps unless allow-loosen; no validation of arbitrary grade/negative cap; roundtrip checks parse only; store.New preview side effects.

- **cmd/decisive-test/main.go** (1–164): Diagnostic hardcoded historical decision IDs/model: readonly GORM wrapper then direct DeepSeek network call after stripping prompt sections; not offline.

- **cmd/detector-report/main.go** (1–58): Detector readonly URI still invokes migrating constructor; sensitivity flag only prints text, does not recompute.

- **cmd/excursions/main.go** (1–60): Excursion report or explicit date backfill; UTC date parse, optional trader/symbol; constructor migrations; no built-in backup.

- **cmd/gate-jwt/main.go** (1–68): Local JWT mint matches dotenv→config→auth path but opens migrating store, logs may precede token.

- **cmd/levelstats-backfill/main.go** (1–103): LevelStats backfill writes by covered days and hardcoded fallback trader; unfiltered MNQ bars coverage; errors on summary ignored.

- **cmd/nq_smoke/help.go** (1–20): Smoke help documents legacy Databento/CSV matrix, omits tcp.

- **cmd/nq_smoke/main.go** (1–174): Legacy default fetches96h-old NQ window then stdin decision→CSV signal→30s fills; unknown subcommand falls into this path.

- **cmd/nq_smoke/smoke_all.go** (1–28): Runs databento,resolver,prompt,roundtrip, excludes tcp and lists skipped components as ran.

- **cmd/nq_smoke/smoke_databento.go** (1–59): Credential-dependent Databento historical shape check >=40 bars; first bar OHLC unchecked.

- **cmd/nq_smoke/smoke_prompt.go** (1–55): Offline future prompt construction, keyword omissions WARN only.

- **cmd/nq_smoke/smoke_resolver.go** (1–40): Network resolver regex root+quarter+single-digit year; credential absence skips.

- **cmd/nq_smoke/smoke_roundtrip.go** (1–61): Temp CSV/mock roundtrip; any fill passes without comparing fields.

- **cmd/nq_smoke/smoke_tcp.go** (1–81): Loopback ephemeral test TCP server/mock client and signal; any fill passes; close-then-bind port race.

- **cmd/planner_ab/main.go** (1–162): Provider A/B called offline means outside live loop, actually network+credential use and migrating store; unchecked argv; hardcoded DeepSeek endpoint; no HTTP timeout; schema only.

- **cmd/research_export/main.go** (1–46): Actual readonly archive exporter RFC3339 receipt [from,to), verifies checksum/counts and writes stdout.

- **cmd/sandbox-seed/main.go** (1–285): Sandbox seed only substring path guard (not symlink identity), initializes/mutates multiple tables; appends plans/trades despite idempotence claim; errors often ignored.

- **config/config.go** (1–277): Loopback API defaults, insecure JWT warning fallback, permissive numeric env parsing, per-process telemetry initialization; strategies own trading knobs.

- **crypto/crypto.go** (1–469): RSA OAEP browser envelope and AES-GCM ENC:v1 storage; optional timestamp; EncryptedString silently falls back on errors; key decoding hashes nonstandard lengths.

- **deploy/fix-wsl2-clock.sh** (1–78): Root chrony installer replaces config, enables service, attempts makestep, installs cron fallback; enable --now does not restart already-running service.

- **deploy/install-autostart.sh** (1–122): Root installer detects sudo user/node, appends NT_TRANSPORT=tcp, stops instances and renders/enables service units.

- **deploy/install-clock-guard.sh** (1–26): Copies/enables clock user timer, chmod script, starts one immediate measurement.

- **deploy/install-db-backup.sh** (1–23): Copies/enables backup user timer; requires existing user session/linger setup.

- **deploy/install-journald.sh** (1–34): Root journald persistence installer, vacuum-size 2G and effective config printing.

- **deploy/leveltruth-cutover.sh** (1–52): Historical hardcoded cutover revision; DB position and journal substring checks; writes RELEASE before binary validation, no lock/full flat gate/rollback, boot match unbound to new process.

- **deploy/nofx-claim-test.sh** (1–56): Claim regex source-derived table pins for real malformed/routable messages; no execution.

- **deploy/nofx-claim.sh** (1–106): Claim regex validates routable session; remote branch presence check, branch/empty commit/push; check/audit read local remote tracking refs.

- **deploy/nofx-clock-guard.sh** (1–80): Root-free RTC/NTP/Windows drift detector, atomic JSON state; only RTC determines CRITICAL, unknown RTC still OK.

- **deploy/nofx-db-backup.sh** (1–79): Online sqlite backup, quick_check, gzip/move, first ISO-week promotion, retention; no trap removes partial files, no locking or retention input validation.

- **deploy/nofx-lock-test.sh** (1–493): Lock tests read fully; compressed keeper timing, refusal/expiry/group ownership/removal/incomplete pins; historical heartbeat-path assertion reads missing file; no execution.

- **deploy/nofx-lock.sh** (1–525): Atomic mkdir lock and bounded detached keeper; heartbeat/session ownership, release group targeting, incomplete states; with-heartbeat adds separate writer; reclaim retains expiry and no new keeper.

- **discipline/freeze.go** (1–73): Mutex-protected in-memory trader freeze latch; first reason wins; clear owner API boundary; state gone on restart.

- **discipline/reentry_cooldown.go** (1–119): Mutex-protected stop-loss cooldown per trader/symbol/normalized side; earlier timer or absolute ATR move unlock; read/delete separate critical sections.

- **expectancy/aggregate.go** (1–693): Expectancy load positions corrected PNL, optional arms/excursion/counterfactual joins; atom-based rollups and era filter; n>=30 normal mean CI gate; E8 keys omit rule; globally scoped database query.

- **expectancy/levelkind.go** (1–77): Kernel kind vocabulary; decorated label longest prefix recovery can accept unrelated suffixed text; no dedicated unknown-kind counter here.

- **expectancy/model.go** (1–240): Typed realized/counterfactual read model, nullable optional stats, MinN30, fixed historical CT era; rows retained for reaggregation.

- **hook/hooks.go** (1–40): Global unsynchronized hook registry; generic return assertion can panic; disabled/missing hooks return nil.

- **hook/http_client_hook.go** (1–23): HTTP hook result wrappers log error and return client anyway.

- **hook/ip_hook.go** (1–19): IP hook result wrapper logs error and returns IP anyway.

- **hook/trader_hook.go** (1–42): Binance/Aster hook result wrappers log error and return client anyway.

- **install-stable.sh** (1–106): Legacy upstream stable Docker installer overwrites compose, creates keys/ShanghaiTZ if absent, no chmod/env validation, starts services.

- **install.sh** (1–294): Legacy upstream main Docker installer overwrites compose; optional unscoped trade-table deletion no backup, image pull/restart; curl health accepts HTTP errors.

- **internal/retry/circuit_breaker.go** (1–69): Circuit breaker cooldown returns Allow true to all callers, despite single-probe comment; no half-open reservation.

- **internal/retry/retry.go** (1–38): Exponential 200ms..5s retry, checks cancellation only between failed attempts, no jitter.

- **levelidentity/identity.go** (1–57): Recording identity requires seven inputs and formation-close; SHA256 raw string fields and positive finite bounds, does not canonicalize or order bounds.

- **logger/config.go** (1–13): Logger level defaults info.

- **logger/db_sink.go** (1–82): WARN+ logrus hook injected once; atomic sink replacement, trader tag extraction; callback must be nonblocking/nonrecursive by contract.

- **logger/logger.go** (1–212): Logrus stdout plus boot-date data file, no rollover; formatter local entry.Time and discards structured fields in text; wrappers and MCP adapter.

- **main.go** (1–724): Boot: dotenv/logger/config/encryption before stores; migrations and trader loading precede integrity in this base; research sidecar, boot ledgers, agent/API/Telegram then SIGTERM shutdown.

- **railway/start.sh** (1–57): Railway generates ephemeral encryption keys each unset boot, nginx health always200, backend background unsupervised then tail forever; no JWT creation.

- **researchsnapshot/archive.go** (1–286): Separate SQLite WAL archive schema1, transaction atomic validated facts, nullable clocks, receipt-range deterministic export with hash; verify counts+objects hash only.

- **researchsnapshot/fact.go** (1–106): Registered five-object evidence dictionary initializes NULL+reason; typed nil normalized; validates known object/required fields; extra fields accepted.

- **researchsnapshot/plan.go** (1–184): Per-authoring PlanTrace capture snapshots of prompts/attempts/replies/verdicts/publication/model config; publishes scenario evidence and explicitly unknown actual permission/risk.

- **researchsnapshot/recorder.go** (1–216): Nonblocking128-capacity record queue, 5ms admission check; builders/SQLite on worker,2s save timeout, counted drops, p50 admission histogram; builders themselves not bounded.

- **researchsnapshot/runtime.go** (1–147): Atomic global recorder; archive-start fail disables capture, boot counts by CT captured date and reports UNKNOWN; contained producer panics.

- **safe/go.go** (1–59): Goroutine panic recovery including callback containment; Must converts panic to error; no restart of failed goroutine.

- **safe/io.go** (1–29): Reader limit+1 detects oversized body, default10MiB.

- **scripts/arm_state.py** (1–31): Python caches canonical Go arm SQL and evaluates states in memory SQLite, no handcopied enum.

- **scripts/backfill-position-plan.py** (1–125): Immediate DB migration/backfill by row id; no backup despite unused shutil; fixed CT session windows; plans dict overwrites ambiguity instead of proving exactly one.

- **scripts/leveltruth_missed_turns.py** (1–103): Historical missed-turn estimator fixed-CDT dates and hardcoded trader/session; row-count m5 grouping, full-session future swing seating against last high, not causal production evidence.

- **scripts/mpm_resolution_fixture.py** (1–62): Synthetic OHLC aggregation ordering example; local TP-first vs SL-first assumptions; proves fixture inversion only.

- **scripts/mutate-selftest.sh** (1–93): Mutation selftest builds throwaway module with no-match/build-fail/killed/survived/no-tests and restore cases.

- **scripts/mutate.sh** (1–103): Mutation file backup/sed/cmp/build/test/restore; broad FAIL catches infrastructure errors, ignores nonzero RC without FAIL and [no test files]; trap restore on signal does not explicitly exit.

- **scripts/sandbox-down.sh** (1–11): Stops sandbox via path regex and globally matching vite.sandbox config; ports are system observations not owned-process proof.

- **scripts/sandbox-reset.sh** (1–9): Stops sandbox then removes fixed sandbox db/WAL/SHM and calls up; up reads live DB despite never-open claim.

- **scripts/sandbox-up.sh** (1–70): Copies live DB readonly into sandbox then partial scrub; keeps credentials, ignores scrub errors, existing DB not rescrubbed; same cwd dotenv loaded, synthetic mode not isolation proof; listener check still prints READY on timeout.

- **scripts/stale_met_replay.py** (1–106): Historical stale-MET replay read-only SQLite; fixed epoch buckets can use forming/future constituents at decision time, no contract/source filtering or row IDs.

- **security/url_validator.go** (1–228): SSRF scheme/name/IP checks and redirects; safe dialer re-resolves checked hostname instead of pinning IP.

- **start.sh** (1–422): Docker lifecycle wrapper; missing-key creation and forced rotation, local env permissions600, pull/update/clean/restart independent of current lock/flat gates; compose detection malformed command -v.

- **telegram/agent/agent.go** (1–286): Per-chat LLM tool loop10 iterations: arbitrary authenticated local API verbs/paths, first-turn account snapshot; no local action allowlist, max iterations falsely says completed; token minted through24h auth.

- **telegram/agent/apicall.go** (1–88): Localhost API tool, authenticated JSON request,30s timeout, no path/method allowlist and unbounded response, errors rendered into model context.

- **telegram/agent/manager.go** (1–79): Per-chat semaphore60s wait, agents/lanes never evicted; Reset bypasses lane (memory race).

- **telegram/agent/prompt.go** (1–100): Prompt embeds docs/user identity and immediate-action workflows, GET-after-write is instruction not enforcement; stale crypto default examples.

- **telegram/bot.go** (1–461): DB token, first DB user and first /start chat binding, command handling and LLM responses; running bot never receives reload; JWT not renewed for unchanged user; /lang before ACL.

- **telegram/session/memory.go** (1–105): In-memory conversation summary at3000 rough tokens; appends new summaries to old summary (growth), captured initial LLM, no locks; ResetFull clears both.

- **telemetry/bar_horizon.go** (1–50): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Horizon counters atomic and non-resetting.

- **telemetry/errors.go** (1–153): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Error aggregator holds mutex while announce callback executes; lastOccurred unused.

- **telemetry/experience.go** (1–242): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. GA enabled by default; trade/model/token plus user/trader/install IDs sent asynchronously; response status ignored, no bounded queue.

- **telemetry/far_arms.go** (1–39): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Far-arm denominator increment separate from side counter; comment says unknown side counts authored but implementation does not.

- **telemetry/gate_blocks.go** (1–146): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Gate counters roll on any differing day, counts include observational/shadow names; snapshots copy state.

- **telemetry/metrics.go** (1–72): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Prometheus promauto registry labels trader/action/status and gate; latency default buckets stop10s.

- **telemetry/planner_wave.go** (1–19): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Repair regression counter process-wide; trader argument unused.

- **telemetry/shadow_conditions.go** (1–18): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Atomic shadow-arm refusal counter.

- **telemetry/weekly.go** (1–78): Process-local metrics/counters or outbound GA telemetry; exact function semantics covered in functions.json. Session resets depend on callers; unknown inputs generally ignored. Weekly shadow counters describe daily reset but only test reset present; reserved day unused.

- **wallet/balance_cache.go** (1–67): 30s normalized address cache; double-check per-address mutex; no eviction of mutex/key maps; backend called with untrimmed original address.

- **wallet/usdc.go** (1–105): Base USDC eth_call and float conversion; display helper maps errors to0.00, missing result to real zero; no status/body/address validation.
