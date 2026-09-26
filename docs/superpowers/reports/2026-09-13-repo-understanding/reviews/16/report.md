# Assignment 16 — persistence and configuration source review

Base: `63968be62e44db2fb07a92883e02127b9064b0be`. Worktree `/tmp/nofx-understanding-surfaces-20260913` verified at that HEAD with empty porcelain before review. All **43 assigned files, 10,006 lines and 458 named declarations** read in full. Additional caller/test excerpts are recorded separately. No source, live DB, account, environment, runtime or order mutations. No tests executed by this worker; root owns reproductions and repairs. [A] means direct source evidence, [B] static inference, not a demonstrated live failure.

Rules consulted: root AGENTS.md; tracked CLAUDE-canon (new keeper semantics supersede stale hand-heartbeat instruction); AUDIT-CHECKLIST classes 7, 19, 28, 29, 35, 40 and PART2 R1–R10. SYSTEM-MAP and corrected RULEBOOK sections on storage, bars, P&L, settings and entry economics. Latest commit for BOTH documents: `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap`. Review uses the owner correction: daily loss controls remain; no new mandatory per-trade cap is inferred. Supplied base is review target; worker did not certify the running process revision.

## Important findings and exact boundaries

1. **[A, source BROKEN] Placed terminal arm bypasses manual-cancel-wins.** `ArmedOrderStore.UpsertArm`, `store/armed_orders.go:235–385`, selects a live row first (262–263), refuses working/place_pending/unknown nonterminal and cancel_pending (270–286). However terminal+signal enters successor Create at 292–307 before same-version restriction 346–347. Production `trader/armed_executor.go:790–806` constructs a fresh candidate and invokes this when ListNonTerminal cannot find the terminal predecessor. SYSTEM-MAP:219 claims the same-version restriction also covers terminal+signal, contradicting source. `store/armed_orders_test.go:201–242` tests terminal rows WITHOUT signal IDs; `store/armed_append_only_test.go:113–136` explicitly assumes replacement without a version change. Root separately reports reproducing and repairing ordering on its repair branch; that repair is outside this immutable base review.

2. **[A, source defect; B operational impact] Successor misses provenance initialization.** The same early Create at armed_orders:307 skips BootID and ArmedUnderVersion defaults at 375–383. Real candidate literals at armed_executor:790–795 omit both. Even after moving version veto earlier, a legitimate new-version successor can therefore start with blank boot and first-authorization version zero. In-place armed refresh repairs zero version only on a subsequent upsert (314–322), while preserving its blank boot; immediate readers/sweep can see missing provenance. `store/attribution_test.go:91–129` tests initial/in-place refresh only. `attribution.go:GetArm:83–90` also still uses First without placement ordering, returning oldest placement rather than live/latest. No row count or runtime impact asserted.

3. **[A, source path defect] Foreign-boot cancellation budget reset is behind its own cap.** Store `RequestCancel:465–490` resets attempts when CancelAttemptsBoot differs. The production pass `trader/cancel_confirm.go:349–425` checks old attempts against cap at 399 before calling RequestCancel at 415. A capped old-boot row with no settlement proof hits continue and never reaches reset. `store/cancel_budget_boot_test.go:18–99` directly calls RequestCancel, bypassing the problematic production branch. Root notified for call-site reproduction; worker did not run tests.

4. **[A, source BROKEN] History recent win-rate includes unresolved denominator.** `GetHistorySummary`, position_history:39–123: recent rows loaded (105–107), corrected NULL skipped (110–113), but denominator is len(recent) (119), including skipped rows. Example static arithmetic: one resolved win + one NULL yields 50%, whereas resolved cohort is 100% n=1. This is a synthetic arithmetic example, not a live-row claim. Summary's UnresolvedExcluded is the broader GetFullStats count, not a recent-specific denominator disclosure.

5. **[A, source BROKEN] Historical current/max streak algorithm.** `calculateStreaks`, position_history:126–184, walks newest→oldest, resetting currentStreak at each transition then assigns it after the entire scan; output is the oldest run, not current run. Maximum counters update only on extending a run, so singleton maxima remain0. Zero-P&L resolved row becomes a loss (`pnl>0`); NULL is skipped and bridges runs. This is AI history presentation; the risk-breaker query is separately implemented in position_query.go and was NOT reviewed as part of this finding.

6. **[A schema declarations; B fresh-schema risk] Order/fill index name collision.** order.go:24 and :61 apply uniqueIndex idx_orders_exchange_unique / idx_fills_exchange_unique only to the external order/trade ID field; ExchangeID fields have no matching tag. InitTables:140 AutoMigrate followed by :145–147 CREATE UNIQUE INDEX IF NOT EXISTS of intended composite uses the SAME name. Inference: fresh GORM schema may retain globally unique external IDs, refusing legitimate equal IDs across exchanges. Index Exec errors are ignored. Root notified; no DB inspection/reproducer here.

7. **[A configuration sequence; B pool risk] SQLite PRAGMAs are not established per connection.** gorm.go:22–64 opens path, sets pool4 at45–47 and sends unchecked separate PRAGMA Exec at50,57–59. foreign_keys and busy_timeout are connection-local; this code alone does not ensure each pooled/recycled connection inherits them. WAL comment at54 still claims MaxOpenConns(1). No assertion about live connection settings.

8. **[A] Read-only facade access can migrate, and opening Store mutates data.** Store.New/NewWithConfig invoke initTables/defaults (store.go:63–126). Exchange.initTables triggers `cleanupIncompleteExchangeConfigs` (exchange.go:90–124), deleting incomplete configs and enabling disabled complete configs. Position zero→NULL migration runs unconditionally in store.go:239–243. Lazy constructors for candidate pool, config diffs and planner facts/rejections AutoMigrate while discarding errors. Consequently use of New against a supposedly read-only audit DB is unsafe; this worker never opens one. Existing migrations are described, not authorization to run them.

9. **[A] Other scoped truth limitations.** `fade_stamp.go:98–108` ignores sinceMs in OpenFadeCandidates. `calendar.go:108–132` says past dates stay frozen but does not test date; callers must enforce it. `GetLedgerDayTotal`, pnl_surface_guard:63–84, excludes unknown close reasons before counting unresolved NULL, so the count is not every unresolved close in the day. `CountCancelPending/CountCancelUnconfirmed/StateCensus` return zero/empty on DB error, unlike CountForeignBootCancelAttempts returning -1. Many telemetry counter readers similarly conflate missing/malformed/error with0. `grid.go:578–588` TotalPnL lacks explicit total_pnl column mapping while SQL alias has it (known naming pitfall); legacy surface relevance remains unproven. Exchange.Create/Update accept enabled but force true (exchange.go:269,305).

## End-to-end connections and ownership

**Facade and initialization.** New→InitGorm→SQL connection→initTables→initDefaultData, failing construction on most explicit migration failures; lazy facade accessors under one mutex return cached sub-stores. NewFromGorm wraps without migration, NewFromDB is legacy SQL-only and cannot supply GORM-backed methods. Store.Close closes DB but does not drain LogEventStore's writer. This is persistence authority, not an execution gateway. Generic ID-only methods rely on validated caller ownership; scope absence is not by itself a proven endpoint vulnerability.

**Authorization to broker receipt.** Production candidate literal/UpsertArm described above → `BeginPlacement` (412–425) atomic predicate armed+empty signal sets place_pending before socket send → `ApplyPlacementReceipt` (431–456) promotes only pending to working; rejected receipts can enrich an absent reason while preserving terminal state. `ExpirePlacement` (692–697) records timeout reason without freeing slot. `RequestCancel` persists intent/budget; confirmation pass reads a fresh persisted broker book with positive snapshot ID, then `ConfirmCancel` (495–504) stores cancelled+evidence ID. Store ConfirmCancel itself accepts zero snapshot and arbitrary prior state; SetState also permits arbitrary transitions. Thus semantic enforcement partly resides in callers, and races between read and unguarded writes need their own production tests. A Go state write does not prove a wire action.

**Market tape.** BarHistory.Migrate (202–258) creates schema, one-time backup/dedupe and old aggregate deletion if unique index absent, then contract/source migration delegates. InsertBars (265–324) batches200, requires contract and live/historical/mixed source, accepts all nonempty TFs, and upserts natural key symbol+TF+open-ms; historical cannot overwrite live/mixed. ImportBars (337–380) requires historical_import and does no updates, returning inserted/skipped counts; batches can partially commit before a later failure. LastNBarsOn (530–551) filters mixed/off-scale and optional contract, reverses newest selection to ascending; BarsBetweenOn (476–487) uses half-open time. Empty contract deliberately gives audit-style unfiltered data. TF retention (99–127) exempts imports; old PruneOlderThan (459–465) does not. No validation here of OHLC shape, timestamp alignment or actual closedness; upstream canonical bar writer owns these. Same-minute other-contract history collides because contract is not in PK.

**Positions, P&L and lineage.** Position.Create (415–443) inserts OPEN and then explicitly nulls unmeasured excursion pair. Guarded reconcile close (453–489) only changes still OPEN rows; other quantity/close methods are ID-only read-modify-write. Closed rows retain raw realized P&L, nullable corrected P&L and correction note. CorrectedPnL (953–958) returns value+known; EffectivePnL (942–947) remains raw fallback only for approved per-row contexts. Adherence write blocks test-seam grading; plan link writer captures plan/version/scenario. Attribution distinguishes unstamped vs UNRESOLVABLE vs linked, with CT era boundary and first-authorization version separate from last touch. LedgerDayTotal strict sum is reporting, not the daily-risk gate. StopTargetNear (decision:492–529) is a nearest-timestamp heuristic for opening stop/target; it has no account, side or action-success check and only scans earliest20 in window.

**Telemetry/research.** TouchEpisodeStore is older append-only touch telemetry and ordinal seed; TouchOutcomeStore methods in fade_stamp record one-time fade labels, and OpportunityOutcomeFor is deepest factual rung. TradeExcursionStore records measured nullable path/exit and corrected-only P&L; it cannot represent unfilled opportunities, hence outcome ladder. AbConfirmStore is independent counterfactual four-rule data with MNQ point value; repair is opt-in with online backup helper and leaves unrecomputable cases explicit. It cannot reconstruct bad original fill timing from arithmetic alone. CandidatePool records seated and cut populations, but global row pruning may split a read. PlannerReadFacts records rendered inputs even on successful planner reads; rejected prompt store retains verbatim rejected attempts/facts separately. All are observational at this boundary.

**Configuration, memory and external settings.** ResolveMinRiskReward/HTFVeto/PlanMode/OneSetup give resolved values plus source labels; knob registry provides historical consumer evidence, not executable enforcement. Config diff serializes configs and compares leaf values; caller must supply resolved configs. User/Exchange are user-scoped storage boundaries with hidden password hash/encrypted credential types, whereas TelegramConfig is a single plaintext-token row protected by an instance mutex and masked String formatting. No actual secrets read. Calendar/session-profile/digest store time-keyed memory; digest adds trader scope but lacks database uniqueness for its check-then-create natural identity. AI charge prices are hardcoded estimates per call, not provider billing.

## Historical map contrast and limits

Historical Understand Anything July10@7a8adce0 export contains **117 matching nodes, 12 matching file nodes and 1084 incident edges** for this assignment. The other31 assigned file paths are absent. Saved export is `historical-graph.json`; direct manual graph inspection covered the12 file summaries and45 non-containment file edges (mostly package-import expansion), not every exported edge. Those imports fan one store package dependency across many files; they are not proof each caller uses each store function. Current AST connections are explicitly syntax-only, not type resolved.

Corrections: historical ai_charge description says token costs; current GetModelPrice/Record charges one fixed per-call estimate. Historical decision summary says account/position snapshots persisted, but current DecisionRecordDB has no snapshot fields, LogDecision writes none, and toRecord restores none. Historical position language note says ClosePositionFully computes holding duration and P&L; current method accepts supplied P&L/time and writes them. Historical facade omits most newer plan/tape/arm/research stores. Core user/exchange/equity/grid role descriptions remain broadly useful but do not establish present enforcement. CGC historical index freshness is root-provided; this worker did not query or reindex CGC.

Stale current comments also matter: bar_history:13–22 and260–264 claim INSERT OR IGNORE/1m-only/single retention, superseded by actual upsert/all-TF/per-TF behavior. position.go:184–187 still recommends raw fallback despite CorrectedPnL law. Source comments containing old row counts are historical claims, not live facts verified here.

## Test evidence and unresolved branches

Read `armed_append_only_test.go` fully; manual-cancel/version-bump portions of armed_orders_test; attribution stability portion; cancel_budget_boot_test fully. Tests demonstrate coverage shape only, not this worker's execution results. Relevant discovered fixtures (not full-read/executed here) include bar_history_test, history_import_test, bar_contract_roll_test, class33_boot_sweep_test, position_side_casing_test, pnl_surface_guard_test, trade_excursion_test and test_seam_exclusion_test. Root handles independent tests/repair validation. No live sample IDs are asserted; IDs in source comments remain historical annotations. PostgreSQL compatibility, multi-connection races, API authorization reachability, runtime ordering and broker settlement require separate verification.

## Per-file coverage and exact named-function boundaries

See `functions.json` for all458 declarations with start/end, purpose, input/output and syntax-only call connections; per-file notes below are grounded in the full reads. Anonymous callback semantics are included under parent functions, plus episodeDetectorScope's file-level note. Registry literal contains no named function.

- **store/ab_confirm.go (1–461)** — Counterfactual four-rule rows keyed plan/version/scenario/rule; short arithmetic repair is optional, MNQ $2/point, distinguishes missing inputs/direction/bad fill geometry. Upsert update map omits direction/recompute; historical repair cannot establish original fill bar.

- **store/adherence_regrade.go (1–153)** — Opt-in regrade clears only CLOSED matched full-lineage D outside off_band/struct and test seam; backup helper precedes external orchestration. Scan IDs then update is not atomic predicate recheck.

- **store/ai_charge.go (1–169)** — Approximate fixed per-model call prices, not token billing. Today uses host-local midnight, explicit date UTC. Aggregate methods suppress query errors.

- **store/arm_normalized_counter.go (1–40)** — Atomic system_config increment; follow-up count read can observe later increments and masks absent/error/malformed to zero.

- **store/armed_orders.go (1–744)** — Durable per-slot placement history. BeginPlacement CAS precedes wire; received receipts guard terminal resurrection; pending cancel holds slot. Terminal successor precedes version veto and skips new BootID/ArmedUnderVersion; see report.

- **store/attribution.go (1–239)** — Three-way unstamped/unresolvable/linked identity; CT Aug15 era; one-time scoped sentinel conversion. GetArm still lowest-ID First despite append-only placements.

- **store/bar_history.go (1–551)** — Contract/source stamped OHLCV natural key excludes contract; live/mixed protect against historical replay. All TF stored despite old 1m-only comments. Import do-nothing collision; TF retention exempts historical_import.

- **store/calendar.go (1–145)** — Trade-date shared calendar; create-if-absent, upgrade non-live, refresh changed live payload. Store itself does not enforce today-only refresh or incoming-live source; caller owns those checks.

- **store/candidate_pool.go (1–133)** — Per-read seated AND cut candidates; batch create and separate global 20k-row pruning; LatestPool limit can cross reads and pruning can split oldest retained pool.

- **store/class47_counters.go (1–91)** — Durable wake/supersede counters; supersede scans only older-version armed/no-signal rows, then unguarded per-ID SetState updates (concurrent placement needs caller serialization).

- **store/config_diff.go (1–162)** — Sorted dotted JSON diff stored in capped 5000-row history. Resolving before diff is caller duty; function only marshals. Maps recurse despite comment saying whole-map.

- **store/decision.go (1–529)** — API/DB mapping carries prompts, actions, watch/structure, plan attribution and account scope; account/position snapshots not persisted by current model. Statistics swallow errors; TotalOpenPositions counts all rows. StopTargetNear heuristic lacks side/account/success matching.

- **store/digest.go (1–116)** — Trader+symbol+date/session/kind append-style digest; check-then-create has no composite uniqueness in model, so concurrent writes may duplicate.

- **store/episode_boot_line.go (1–56)** — Boot formatter renders supplied recorded counts, explicit backfill n/a, detector k/H and per-read delta source; anonymous resolver variable grouped here.

- **store/episode_detector_scope.go (1–33)** — Mirrors kernel detector env contract due import cycle: positive k default3, horizon default12; float finiteness not checked.

- **store/equity.go (1–209)** — UTC snapshot save; latest scoped optionally account, ascending output; older generic queries trader-only. Latest all-trader query omits account and timestamp tie resolves by map overwrite.

- **store/exchange.go (1–415)** — User-scoped credentials encrypted at type boundary; startup cleanup deletes incomplete configs and enables complete disabled configs. Create/Update ignore enabled argument and force true. NT fields retain legacy CSV metadata.

- **store/fade_stamp.go (1–120)** — One-time nullable fade permission stamp uses supplied evaluation clock; no order authority. OpenFadeCandidates ignores sinceMs; failure count nil-store returns zeros.

- **store/gorm.go (1–169)** — SQLite pool4 WAL/full sync UTC clock, PostgreSQL pool25. PRAGMAs via unchecked one-shot Exec; connection-local settings need pool-wide verification. Global handle overwritten per Init.

- **store/grid.go (1–601)** — Legacy separate grid config/instance/levels/events/regime CRUD; cascade delete transaction, generic ID methods require caller ownership. Stats suppress secondary errors; performance TotalPnL lacks explicit total_pnl tag.

- **store/knob_registry_table.go (1–183)** — Literal registry per schema leaf, live/candidate/ineffective and historical consumer anchors. Method readers explicitly revise old field-grep false negatives. Registry itself is not enforcement; no named funcs.

- **store/level_stats.go (1–108)** — Level/day/trader natural key no symbol; evaluated bool outcomes, per-grade/family unscoped aggregates. SUM(boolean) SQLite-specific portability risk.

- **store/log_event.go (1–129)** — Lazy one-writer channel1024, select-default drop counters, no recursion, event-day retention. No close/drain API; counters process-only and prune errors ignored.

- **store/opportunity_outcome.go (1–67)** — Pure deepest-observed-rung classifier prioritizes filled,armed,confirmed,reached,never-reached; no trades or writes.

- **store/order.go (1–443)** — Exchange orders/fills and watermark queries; intended composite dedup may be defeated by single-field GORM index tags. Duplicate cleanup globally deletes later IDs without remapping fill foreign keys.

- **store/plan_liveness.go (1–179)** — Versioned status freshness5min; unknown nullable tradeable, reversible evaluator vs first-death history. Atomic event/death first-write keys; death read checks exact anchor/version.

- **store/plan_qa.go (1–228)** — Trader+plan chat, proposal apply, owner decline marker and reply-count debounce/cap; raw structured fields not validated in store. Decline count requires caller write coordination.

- **store/planner_read_facts.go (1–193)** — 500-row read telemetry, JSON empty vs absent; horizon numeric validity guarded by ReadHorizons presence. EncodeVoidLevels serialization error becomes [] even for invalid float.

- **store/planner_rejected.go (1–103)** — 200-row global retained rejected verbatim prompts plus validating facts; lazy migration errors ignored, insert error returned, pruning best-effort.

- **store/pnl_surface_guard.go (1–92)** — Registry of strict P&L surfaces and ledger-day total corrected-only; unknown-reason rows filtered before unresolved count; boot zero-raw is test contract, not runtime scan.

- **store/position.go (1–961)** — Position lifecycle/raw/corrected separation, account and plan identities, guarded reconcile close but general quantity/close writers unguarded. Explicit post-create MAE/MFE NULL two writes. See report for dedup and history limits.

- **store/position_excursion_null.go (1–34)** — Startup migration converts both-zero closed MAE/MFE to NULL; assumes pair means uncomputed, with no measurement provenance predicate.

- **store/position_history.go (1–319)** — History summary combines strict aggregates but recent denominator and streak logic defects remain. Exchange closed-history import validates prices/side/times, generates dedup identity, leaves corrected NULL.

- **store/repair_counters.go (1–61)** — Atomic durable repair outcomes key; malformed number coerces0, summary suppresses read errors.

- **store/resolve_source.go (1–80)** — Shared saved/default provenance for RR, HTF veto, plan mode session precedence, one-setup enabled/grade. Daily loss policy not altered.

- **store/session_profile.go (1–105)** — Frozen symbol/session-date profile, check-then-create primary key, recent list and warming count; no contract column.

- **store/store.go (1–739)** — Facade mutex protects lazy pointers; constructors initialize/migrate and defaults; some lazy accessors migrate on first read. NewFromGorm only wraps, NewFromDB lacks GORM; Close does not drain async log writer.

- **store/telegram_config.go (1–164)** — Singleton ID1 per-instance mutex; plaintext bot token stored, String masked only; BindUser replaces binding, caller must authorize. SaveToken clears model via Save(token,empty).

- **store/touch_episode.go (1–109)** — Append-only older touch telemetry, per-day counts and max ordinal seed. Counts omit symbol except MaxTouchNumber; no natural-key uniqueness.

- **store/trade_excursion.go (1–279)** — Unique position path telemetry with nullable measured fields; resolution none marker alone does not erase existing path, close nil correction leaves prior correction. Entry interval inclusive at both ends.

- **store/user.go (1–132)** — Password-hash hidden in JSON; CRUD assumes upstream hash/auth; admin seed empty hash, global delete explicit method; DB errors during EnsureAdmin count ignored.

- **store/watch_assessment.go (1–78)** — Observation store, raw/hysteresis verdict and retrospective outcome/excursion writes. Float0 defaults do not distinguish absent excursion; final outcome first-write predicate.

- **store/zerob_counters.go (1–84)** — Recorded stop-anchor/refusal counts scoped per trader/date/session/class; caller dedup required; count reader collapses errors/malformed into0.
