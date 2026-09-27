# Assignment 15 — persistence and configuration source review

## Scope and evidence

[A] Read all **41 assigned source files / 10,001 lines** at `63968be62e44db2fb07a92883e02127b9064b0be` in `/tmp/nofx-understanding-market-20260913`. Initial pwd, HEAD and empty porcelain output verified. Every assigned SHA256 matched the root manifest at artifact generation. `reads.json` records complete assigned reads separately from additional caller/test/rule excerpts. `functions.json` covers **382 named declarations**, their exact start/end lines, and **30 local function literals grouped under enclosing declarations**. Connections in that file are actual AST call expressions and explicitly **not type-resolved call-graph edges**.

This is static source review. No service requests, production DB reads/writes, trading tests, configuration changes, deployment, source edits, or reproducer modifications were performed. **No runtime incident or successful exploit is claimed.** [A] means inspected source or generated inventory; [B] means implications of that source. Historical sample ids quoted in source comments were not re-queried and are not fresh incident evidence.

Applied `docs/superpowers/CLAUDE-canon.md` and audit checklist pre-audit R1–R10; especially classes 9 (binding), 23 (telemetry), 28 (canonicalization), 35 (recorded counters), 40 (corrected PNL/NULL), 53 (production-call-site parity), 99/107 (arm states), 113 (source-guard scope), 117/118 (bar provenance), 124 (history imports), and 126 (frozen structural geometry). Read root AGENTS instructions; the tracked keeper rule supersedes obsolete hand-heartbeat prose. Root selected this source revision; this worker did not assert it equals a live boot.

Spec freshness records (`git log -1 --format='%h %aI %s' -- <path>`, all ancestors of reviewed base):

- CLAUDE-canon: `07b53e65 2026-09-10T17:13:53-05:00 cleanup batch 1 — B5 the mutation harness that cannot fake a verdict, B6 the worktree-add check`
- AUDIT-CHECKLIST: `dfda15e1 2026-09-13T00:48:28-05:00 test: isolate session clock fixtures from weekly backfill workers`
- SYSTEM-MAP: `565e8fbe 2026-09-13T00:39:02-05:00 fix: use owner daily-loss controls without requiring a per-trade cap`
- VL-TRADING-RULEBOOK-v1: same `565e8fbe` line.

Owner's latest clarification is preserved: **existing daily-loss controls, no new mandatory per-trade cap**. `store/structural_geometry.go:32–59` resolves buffer/cost/RR, not a per-trade loss cap. `RiskCapUSD` at :96 is retained only for historical serialized records. No recommendation here restores that superseded policy.

## Findings, highest impact first

### F15-1 — A / BROKEN in source: trader deletion has side effects before ownership succeeds

[A] `store/trader.go:223–229` deletes `EquitySnapshot` with only `trader_id`, ignores that delete's error, then deletes `Trader` using `(id,user_id)`. A zero-row trader deletion is success. `api/handler_trader.go:767–788` calls this function without first resolving ownership, then looks up/stops/removes the named runtime trader after success. `api/server.go:146,205–207` mounts this under auth and `planTraderOwnership`; that middleware at `api/handler_plan.go:97–142` explicitly skips paths outside `/api/plan/` and `/api/risk/`. `api/route_registry.go:28–46` adds no further authorization; `api/server.go:851–892` authenticates a JWT and stores user identity but does not check target trader ownership.

[B] A valid user targeting a different user's known trader id can reach an id-only equity deletion and runtime removal path even though the owner-scoped trader row itself is retained. This is a **statically traced cross-user side-effect risk**, not a demonstrated exploit. Review/repair both store ownership-before-cleanup and handler ownership-before-runtime effects; cover wrong-user and zero-affected-row paths in isolated fixtures. Root owns any repair/reproducer.

### F15-2 — A / BROKEN in source: unconfirmed AI→grid switch loses preserved AI fields at serialization

[A] `store/strategy.go:554–564` restores AI fields into the in-memory merged config. `api/strategy.go:328–331` calls it and logs preservation, then :354–368 marshals and persists. `StrategyConfig.MarshalJSON` at `store/strategy.go:805–842` assigns `AIConfig` only in the non-grid branch; grid config emits no AI bundle. The Go fields themselves are `json:"-"` (:733–737). Reloading the persisted grid record therefore has no saved AI fields to restore when switching back.

[A] `store/preserve_ai_config_test.go:53–64` calls the preservation helper twice on structs without intervening marshal/unmarshal, so its round-trip does not exercise this production loss boundary (checklist 53). This is deterministic serialization behavior, not a runtime incident. Test the API persistence round-trip, not just helper self-consistency.

### F15-3 — B / UNVERIFIED runtime: prior OR median bypasses tape provenance filters

[A] `store/fade_or_history.go:19–57` queries raw `BarHistoryDB` rows by symbol, `tf='5m'`, and time, with no source/contract filter. The full caller `trader/fade_facts.go:39–43,89–95` directly supplies this median to fade facts. Unlike generic historical mixed-contract price comparisons, each prior-session OR range can legitimately come from its own contract; the definite issue is that this reader can include **mixed, off-scale, imported, or otherwise unusable rows**, and it does not prove one clean contract within an OR bar. It also reads stored 5m aggregates rather than the store-deepened 1m aggregation described for trustworthy horizon input in SYSTEM-MAP.

[B] Bad OR ranges can affect recorded fade permission and downstream one-setup permission selection. No current row population or actual permission outcome was sampled. Existing path lint that only matches `.LastNBars(`/`.BarsBetween(` cannot certify this direct DB reader. Keep this as source-confirmed reader gap; do not claim mixed-source rows were actually selected.

### F15-4 — A / BROKEN coverage claim: reflected knob census cannot see emitted ai_config

[A] `store/knob_registry.go:61–101` skips JSON `-` fields while reflecting `StrategyConfig`. All five AI bundle fields use that tag in `store/strategy.go:733–737`, and their emitted JSON exists only through custom marshaling to `AIStrategyConfig` (:788–795,805–842). Thus a newly added risk/indicator/coin field is outside this reflected schema traversal. `store/knob_registry_test.go:12–41` only verifies leaves the same incomplete traversal returns; `len>=40` does not prove all serialized leaves were enumerated. Leaf-name fallback at registry :106–119 can also conceal different meanings at different paths.

The function does what its reflection rules say; the broader claim that every product schema knob is automatically guarded is false. A JSON/schema-aware independent census should compare the actual serialized product schema. This finding does not establish that a particular live knob is currently misclassified.

### F15-5 — B / UNVERIFIED runtime: follow recording and readable counters hide storage failures

[A] `store/one_setup.go:247–311` issues separate NULL-guarded updates for each known follow field and discards every field-update error, then returns only the state/backfill update error. A terminal state can therefore persist after a known-field write failed. `trader/follow_plan_wiring.go:138,233` checks only the returned error (call-site search); detailed surrounding recorder control flow was not fully reviewed in this assignment. `OneSetupCountsFor` (:190–219) sets `Readable=true` regardless of per-class query failure, and `CountFollowPlans` (:352–369) ignores all count errors after the first count.

[B] This can make a partial observation appear complete/readable and prevent later repair if terminal follow states are excluded. `OpenFollowPlans` (:316–325) selects only NULL/open/broken_no_retest/retested states. No fault-injection test was run. Prefer one transaction for coherent episode evidence and propagate/represent partial read failures.

### Other bounded source concerns

- [A/B] **Level aging clock is refreshed by non-state writes.** `EnsureLevel` (`store/level_state.go:160–181`) calls ordinary GORM `Update("price", ...)` on existing rows; the model's UpdatedAt is autoUpdateTime (:67). `AgedFreshness` (:282–311) starts from UpdatedAt. Production `trader/auto_trader_levelstate.go:95–113` repeatedly ensures active levels; the provider at `trader/auto_trader_dayplan.go:176–188` reads aging. [B] continued price refresh can reset the aging reference; the pure test `store/level_state_aging_test.go:56–91` constructs timestamps directly and does not cover repeated EnsureLevel. Unknown applicability to a given continuously active level, not a demonstrated stale burn.
- [A] **All-unresolved holding bucket disappears.** `store/position_query.go:537–548` emits only `r.count > 0` even if unresolved count is positive. `GetFullStats` and `GetDirectionStats` retain unknown counts; this function loses a whole unknown-only cohort. `store/pnl_truth_test.go:91–102` tests unresolved rows in a bucket already containing resolved rows. Directly inspectable edge case; consumer reachability for this specific statistic was not established.
- [A] **Model-selection paths disagree on environment-only credentials.** `AIModelStore.GetDefault` → `firstEnabledUsable` filters `api_key != ''` at `store/ai_model.go:159`, before `hasUsableAPIKey` can examine environment keys (:191–207). `GetAnyEnabled` (:175–188) has no such SQL filter. Caller populations and actual environment were intentionally not inspected.
- [A] **Completion flags can suppress retries after partial migration failure.** `CorrectHistoricalPnL` (:26–73), `BackfillPnlCorrectedAll` (:93–140), and `BackfillEntryConfidence` (`store/position_backfill.go:26–65`) continue after individual write errors and then set their done flag. Confidence scan additionally limits to 2000 rows before setting done. First PNL correction lacks the later pass's positive entry/exit guard. These are static restart/retry hazards, not fresh corrupt-row findings.
- [A] **Owner-level blank-user bucket is shared fallback.** `store/owner_level.go:70–80,94–104,112–120` includes `user_id=''` for every nonempty user despite a comment implying second users see only their own. Boot migration (:44–53) usually claims blank rows for oldest trader; new/remaining blank rows are still readable/editable across users. API caller validation not exhaustively traced; no data leak demonstrated.
- [A] **Concurrent idempotency depends on caller serialization.** Alert Emit (:57–73) count-before-create lacks unique trader/event key; opportunity close (:73–109) selects NULL but UPDATE repeats only id; NextOrdinal/SaveOutcome (`touch_outcomes.go:219–244`) are separate without episode uniqueness. Weekly matched-random PK protects duplicate storage but loser receives create error. These differ from structural-geometry's transactionally deduped counter path and PlanStore's per-instance serialization. No concurrent execution was reproduced.
- [A] **Undefined input cases in pure research helpers.** `ResolveScenarioLink` (`store/scenario_link.go:58–95`) disables band/ambiguity checking when band<=0 and can index -1 if every anchor distance is NaN. `BuildContinuous` (:37–73) does not check finite basis or row contract/order; zero measured basis is forbidden even if genuinely measured. Upstream validation was not traced, so these are robustness boundaries rather than active failures.
- [A] **Analytics definitions need honest labels.** Drawdown at `position_query.go:372–398` uses literal starting equity $10,000, not trader starting balance. Sharpe is unannualized mean/sample-SD of per-trade dollar PNL. Profit factor returns zero with no losses. These calculations are source facts, not broker/account performance claims.

## End-to-end ownership and flow

### Configuration into runtime

[A] `store/ai_model.go` keeps encrypted model keys typed as `crypto.EncryptedString`; exact user/id lookup may fall back to `default` user. Explicit new provider entries use unique id suffixes; deterministic provider choice is enabled first, newest update, smallest id. Update-with-empty-key preserves secret; clearing per-model thinking overrides writes empty strings intentionally. Cross-user/internal methods (`GetByID`, `AdoptModel`, `GetAnyEnabled`) do not supply endpoint authorization themselves.

[A] `store/trader.go:GetFullConfig` (:232–269) first resolves the user-owned trader and its user-owned model/exchange. `StrategyID` is preferred; missing or failed lookup falls back to active/default, and those strategy lookup errors are discarded. Thus a broken explicit binding may silently become another strategy; audits must join the actual trader binding and confirm runtime loading. Account choice has a separate scoped `UpdateAccount` method and is not overwritten in generic Update.

[A] `store/strategy.go` is the configuration contract shared by kernel/API/trader, not itself a trading executor. Custom Unmarshal accepts nested and old flat schema; nested AI wins if both supplied. Merge uses JSON round-trip and recursively combines maps, mutating the caller's patch normalization. ParseConfig only applies missing coin/kline defaults; ClampLimits and ValidateIndicatorPeriods are distinct calls. Numeric risk clamps/defaults are the legacy fields, while daily-loss and mechanical-exit policies are fields consumed elsewhere. New futures strategy defaults seed static MNQ and ATR/EMA/RSI, disable crypto ranking/OI sources, keep MACD/BOLL off. Static SupportedTimeframes includes 14 intervals. Timeframe/symbol aliases and CME month-family tables are local copies requiring parity with market.

[A] DayPlan root config survives type-specific serialization. Per-session settings resolve named override → strategy/default with meaningful explicit zero for some pointer caps. AcceptanceRuleFor rewrites legacy 2x5m/15m forms to 5m_close at read; DefaultDayPlanConfig still seeds `2x5m`, so literal seed and effective rule differ. Wake spacing effective default is 30 minutes despite older 10-minute comments. Replans are **recorded spend events** only for death_replan/owner_reread, keyed by trader/date/session/reset baseline; no version arithmetic determines spends. Spend mutex is process-local; malformed/read-failed counters can read as zero/full budget. Indicator fingerprint freezes only prompt-relevant settings and excludes API secrets; token estimator is static approximation with hardcoded provider-family limits, not observed provider capacity.

### Plans and execution evidence

[A] `store/plan.go` uses stable trader-scoped chain identity and max(version)+1 inside a dedicated per-instance write goroutine. Overlay versions are independently numbered per plan/version. SQLite DDL enforces JSON validity and composite keys; ignored additive DDL/index errors can obscure schema drift. This is single-instance serialization, not a distributed transaction guarantee. Close signals shutdown but does not join an in-flight write. Lifecycle is mutable while authored doc/version stays append-only; state changes append a telemetry log and never rewrite authoring trigger. `PlanDB.StrategyID` is historically misnamed and actually holds trader id.

[A] `store/arm_state.go` is the conservative lifecycle boundary shared by Go and generated SQLite SQL: unknown states are nonterminal, cancel_pending stays exposure but is not sweepable, and armed with blank signal is authorization only. Unicode trimming/lowercase mapping is deliberately reproduced in SQL. Boot-sweep selects foreign/NULL boot ids and persists recorded sweep count. Received NT8 snapshots store raw orders JSON and clocks; accepted-risk rows append distinct broker terms with NULL for unavailable broker price, alongside simultaneous ledger terms. No update/upsert API exists for accepted-risk records, though the shared DB could still be mutated elsewhere.

[A] Structural policy has MNQ default buffer 4.5 (training p95 provenance), cost 2, and strategy RR. Explicit invalid buffer/cost becomes unknown rather than silently defaulted. `SaveStructuralGeometry` atomically stores current decision and counts each plan/version/scenario/leg/reason once per CME date. An admitted current record does not erase prior refusal count. This table does not place orders or size a live account; actual geometry/placement is the sibling trader assignment. Post-loss, shadow-AB, boot-sweep and generic counters record explicit events, not inferred historical events.

### Tape, episodes and research

[A] Contract/source migrations label existing bars additively, using measured timestamp intersections and explicit symbol-specific historical exceptions. They do not fabricate prices. Contract readers return point/window contract labels; `WindowContract` compares endpoints, may use the sole known endpoint, and does not certify interior source quality. Live readable sources are live/historical; imported history requires separate backtest eligibility. BuildContinuous creates explicitly adjusted output and never writes it.

[A] TouchOutcomeRow is the durable shared episode schema: observed detector verdict, level formation/validity, named identity or heuristic link, nullable opportunity/fade/one-setup/follow evidence. RatesBy reads only ValidityValid; ambiguous is excluded from hold/(hold+break), reported with counts/Wilson interval and descriptive floor n<200. Recorders must supply correct formation and session-day/window because this store cannot reconstruct them. The watermark/ordinal are durable queries but failure can read as zero/one. Empty trader id in opportunity/follow helpers intentionally means all traders and is an internal maintenance convention, not safe user authorization.

[A] Scenario linking is explicitly heuristic, ambiguity NULL. Attainable entry prioritizes observed fill, then named arm assumption, then first tradeable after confirmation; the drawn level is never substituted. Episode closer delegates facts/entry observations to callbacks and preserves uncomputed attainable columns. Identity events use base64-encoded tuples; identity backfill requires full recorded inputs from exact plan version and exact existing name, leaves pre-era rows untouched, and persists per-row three-state outcomes. Follow-plan observations never imply order authority. Matched-random and excursion aggregators are global research stores, not per-user execution gates.

### Position lifecycle, PNL and observability

[A] PositionBuilder routes open/close actions; open may merge weighted entries, partial close delegates quantity reduction, full close computes weighted exit and preserves broker cause through `ExitCauseFromBroker`. NT8 caller excerpt `trader/ninjatrader/close_sync.go:154–170` computes point-value-aware PNL then calls `ProcessTradeWithExitReason`, correcting the historical graph's old direct ProcessTrade edge. Builder itself is not an ordered-event queue or FIFO engine despite introductory prose.

[A] Position query surfaces use corrected PNL, not raw fallback. Unknown closes end consecutive-loss streaks; synthetic e7 seam rows do not count. Recent-trade list retains unresolved entries but sets resolved=false and no computed percentage. Daily activity sums only resolved valid closes but counts **all entries**, including entries whose eventual PNL remains unknown. Optional account filter is on selected APIs, not every aggregate. Source-based seam grading uses a separate classifier whose source read fails open; its SQL and Go whitespace treatment differ. Global histogram does not prove all PNL readers use the same seam definition.

[A] Alert feed uses soft dismissal preserving audit rows and refuses unacked P0 dismissal. Watchdog/cut records carry fresh/reused connection idleness and explicit unresolved resend; ResolveLatest associates only by trader/newest unresolved rather than unique call id and masks lookup errors. Idle renderer sets no timeout threshold. Wave-A backup helper runs sqlite3 `.backup` before caller-authorized migration; migration function itself does not enforce backup/flag, converts every remaining zero excursion and uses no era cutoff. Re-running after real measured zeros appear requires caller policy care. Constructors that lazily AutoMigrate mean even some apparently read-oriented helper entrypoints can attempt schema writes; none were called against production here.

## Assigned-file coverage index

The following inventory is generated from this review's manually read files. Each entry's exact named function boundaries and call expressions are in `functions.json`; detailed per-file notes are in `reads.json`.

- `store/accepted_risk.go:1–120` (6 named functions): Append-only broker acceptance evidence, distinct nullable accepted prices and mutable-ledger prices; epoch-ms clock. Constructors/counts swallow DB errors; nil Append succeeds without recording.

- `store/ai_model.go:1–504` (23 named functions): Per-user encrypted model rows, exact-id lookup with default-user fallback; deterministic enabled/newest/id provider selection. Empty API-key updates preserve secret. firstEnabledUsable filters empty DB keys before env fallback, unlike GetAnyEnabled.

- `store/alert.go:1–183` (12 named functions): Trader-scoped feed/ack/dismiss with durable soft deletion; unacked P0 cannot dismiss. Emit count-before-create dedupe has no unique trader/event constraint. PostgreSQL existing-table branch skips additive migration.

- `store/arm_state.go:1–131` (10 named functions): Canonical conservative terminal classification shared with generated SQLite SQL including Unicode trim/lower parity. Unknown state remains exposure; only armed+empty signal is unplaced. Sweep excludes cancel_pending.

- `store/attainable_entry.go:1–84` (1 named functions): Pure measured-fill > assumed arm > first tradeable-after-confirm priority. LevelPrice is deliberately never used; missing recorded confirmation price differs from never attained.

- `store/bar_contract_roll.go:1–244` (6 named functions): SQLite additive contract migration uses measured 2026-09-10 per-symbol window intersections; census and newest/point/window contract resolution. ContractAt skips mixed labels but not source labels; unknown TF falls back to one minute.

- `store/bar_source.go:1–175` (4 named functions): Measured source migration labels mixed/off-scale exceptions before legacy-live default; strict live/historical readability and separate backtest historical_import eligibility. Historical sample ids are comments, not reverified runtime evidence.

- `store/boot_sweep.go:1–88` (5 named functions): Process boot id via sync.Once; preboot sweep selects other/NULL boot and sweepable states; atomic persisted boot-swept counter; DB exposed for fixtures.

- `store/continuous.go:1–73` (1 named functions): Pure explicit-basis cumulative OHLC adjustment with continuous/mixed labels; zero basis refused. Caller owns ordering/overlap; finiteness and row-contract correspondence are not checked.

- `store/driver.go:1–281` (14 named functions): Legacy sql.DB abstraction for modernc SQLite and lib/pq PostgreSQL, environment defaults and syntax helpers. SQLite uses DELETE/FULL with one connection. Placeholder replacement is lexical, not SQL-aware.

- `store/fade_or_history.go:1–57` (1 named functions): Prior session OR median uses before location and clock, n-limited descending stored 5m rows. Raw query lacks contract/source filtering; errors collapse to (0,0).

- `store/idle_outcome.go:1–125` (3 named functions): Watchdog/cut records aggregated into fresh/reused-idle buckets with resolved/recovered/lost separation; deterministic rendering warns small n and sets no operational threshold.

- `store/indicator_fingerprint.go:1–52` (1 named functions): SHA256 first eight bytes of ordered prompt-indicator projection; no API secret or crypto ranking included. Nil/empty lists normalize by omitempty, list ordering stays meaningful.

- `store/knob_registry.go:1–215` (6 named functions): Reflection walks JSON-tagged StrategyConfig leaves, registry exact-path then leaf fallback, counted boot labels. Custom Marshaler AI fields tagged minus are invisible to enumerator; shared leaf fallback can mask collisions.

- `store/level_identity.go:1–163` (5 named functions): Base64-scoped immutable identity-event keys with allowed-kind and clock validation; counts decode durable JSON. Backfill requires exact named id in exact stored plan/version and leaves pre-era rows untouched.

- `store/level_state.go:1–383` (20 named functions): Trader-scoped price-bin state; ensure preserves burn state but price Update advances UpdatedAt. Play increment and freshness decrement separate; aging uses UpdatedAt with 17:00 CT calendar-day boundary, potentially extended by price refresh.

- `store/matched_random.go:1–138` (10 named functions): Append touch verdicts, global type tallies and ISO-week first-write snapshot. Primary-key prevents duplicate weekly rows but count/create race surfaces error. ResetWindow deletes both tables sequentially without transaction.

- `store/nt8_order_snapshot.go:1–75` (6 named functions): Received order-list JSON forensic snapshots with emitted/received clocks; insert errors returned, account+symbol latest read and timestamp prune. Gate source remains outside this table.

- `store/one_setup.go:1–436` (16 named functions): Latest per-plan scenario sidecar, once-only episode verdict stamps, day counters, forward follow-plan fields. Follow field writes individually ignore errors before state update; Readable counters can hide failed reads. Proximity stamp lacks plan-version restriction.

- `store/opportunity_backfill.go:1–102` (2 named functions): Three-state legacy classification from era/formation/scenario link. Recomputable rows marked reached_declined with session-close cause; terms not fabricated. Partial row updates, no transaction.

- `store/opportunity_close.go:1–166` (5 named functions): Optional empty trader/plan/session means all; callbacks own facts and attainable observations. NULL-selected close loop writes by id without repeating NULL predicate, allowing concurrent overwrite; records first close only under serialized use.

- `store/owner_level.go:1–120` (9 named functions): User-scoped sticky level persistence with empty legacy rows visible/editable to nonempty users; migration assigns blank owners from oldest trader. API must supply user. Internal consume/delete are id-only.

- `store/plan.go:1–547` (28 named functions): Append-only plan docs and overlay versions via per-instance single writer; lifecycle mutable with separately appended event log. StrategyID column actually stores trader id. SQLite JSON-valid checks; optional/global legacy readers retained.

- `store/pnl_correction.go:1–160` (4 named functions): Additive historical MNQ price*qty*$2 corrections and immediate close stamp. Per-row NULL guard preserves corrections, but completion flags set even after row update errors; first pass lacks nonpositive-price refusal.

- `store/position_backfill.go:1–106` (2 named functions): Once-flag confidence inference from recent action/symbol decisions around entry, capped 2000 candidate positions. -1 means looked/no result, 0 unpopulated. Flag may finish partial/error run; time association is heuristic.

- `store/position_builder.go:1–211` (6 named functions): Trade action dispatch to open/average and partial/full close persistence; broker exit reason carried explicitly. No sorting or FIFO queue in this class; no-match close skipped. Full overclose clamps qty but not supplied PNL/fee.

- `store/position_query.go:1–601` (10 named functions): Strict corrected-PNL statistics with null exclusions, account-specific full/recent reads, loss streak broken by unknown close. Holding buckets with only unresolved rows omitted; drawdown assumes starting equity 10000; Sharpe unannualized PNL ratio.

- `store/post_loss_counter.go:1–82` (5 named functions): Atomic per-trader/date/session rearm counts, latest resolved losing close among newest 20 rows, bounded nonfuture loss window. Counter only, no refusal; unlike streak unknown closes skipped.

- `store/research_record.go:1–23` (2 named functions): Panic-contained research snapshot around placement timeout; observed/receipt same supplied clock, write error nullable; telemetry never gains placement authority.

- `store/scenario_link.go:1–95` (1 named functions): Pure nearest-price heuristic with two-in-band ambiguity -> NULL; named basis and distance. Nonpositive band disables band and ambiguity checks; invalid NaN anchors can leave best index -1.

- `store/seam_grading.go:1–138` (6 named functions): Source-based test-seam classifier, SQL exclusion histogram and stamp migration. Source-read error fails open. SQL does not fully trim like Go; stamp action and counts distinguished.

- `store/shadow_ab_counter.go:1–43` (2 named functions): Persisted atomic shadow experiment count across restarts; unset/parse failures return zero. Does not authorize live experiments.

- `store/strategy.go:1–2487` (84 named functions): Custom nested JSON compatibility, limits, timeframe/symbol normalization, language/futures defaults, day/session accessors, recorded replans and reset keys, CRUD, token estimates. Grid serializer drops AI bundle despite preservation helper; ParseConfig defaults do not clamp.

- `store/structural_geometry.go:1–181` (7 named functions): Structural-stop policy default MNQ buffer 4.5 and cost 2, provenance known flags and strategy MinRR. Atomic sidecar + deduped per-day refusal counts. Historical RiskCapUSD is output compatibility only; no extra per-trade admission cap.

- `store/system_counter.go:1–39` (2 named functions): Generic persisted atomic integer increment; blank keys refused for increment, malformed read parses to zero. Counter records, never infers.

- `store/touch_outcomes.go:1–461` (15 named functions): Durable episode with validity, formation, scenario identity, opportunity, fade/one-setup/follow evidence. Rates use valid only and exclude ambiguous denominator. Ordinal/watermark reads lack unique write guard and return defaults on failure.

- `store/trade_excursion_stats.go:1–159` (6 named functions): Grouped measured MAE/MFE nearest-rank distributions with unknown-level/unmeasured counts, undefined percentiles NaN rendered dash; dimension allowlist, global rows.

- `store/trader.go:1–338` (20 named functions): User-scoped trader bindings and CRUD; runtime modes explicitly allowlisted. FullConfig resolves bound strategy then active/default on missing/error. Delete removes equity by trader id before ownership-scoped deletion and ignores zero affected.

- `store/visibility.go:1–105` (6 named functions): Visibility is any configured field/enabled, not credential completeness. Separate per-exchange required-field list still requires NTDataDir for NinjaTrader. Nil model/exchange/trader/strategy not visible.

- `store/watchdog_fire.go:1–109` (5 named functions): Durable cut/watchdog event and latest-unresolved-per-trader resend attachment; Record stamps clock. ResolveLatest swallows all lookup errors as absent; concurrent calls have no call-id identity.

- `store/wave_a_migration.go:1–201` (5 named functions): Flag predicate + sqlite3 backup helper, legacy duplicate/unverified episode classification and zero-excursion NULL conversion, boot census. Migration method itself neither checks flag nor performs backup and has no zero-age cutoff; caller must gate one-time use.

## Graph comparison and limits

[A] Historical Understand Anything graph is July 10 at `7a8adce0`, not this base. Extracted **72 nodes and 671 incident edges** for this assignment into `historical-graph.json`; only **7/41 current assigned paths** exist there: ai_model, driver, position_builder, position_query, strategy, trader, visibility. Root's historical CGC file export likewise includes only those seven paths (781 indexed files globally). Thus **34 assigned files**, including structural geometry, arm states, plans, provenance migrations and episode contracts, are absent from both historical file inventories. No reindex/delete/watch was attempted. No CGC function-query result is claimed beyond root's exported file inventory.

Historical graph edge types: 510 imports, 40 calls, 65 contains, 50 exports, four tested_by, one documents, one related. Imports fan out one package import to many store files; these are package relationships, not proof of calls to each file. Manually inspected selected node summaries and all 40 recorded call relationships; raw incident graph is saved for root. Current AST inventory supplies a broader exact syntax census, but cannot resolve interface/dynamic calls.

Explicit corrections:

- Graph `store/driver.go:openSQLite` says WAL; current :172–207 sets **DELETE**. `boolDefault` graph says env parser; actual :269–281 returns SQL TRUE/FALSE or 1/0 literal. These are substantive summary errors, not just line drift.
- Graph `store/visibility.go:IsVisibleExchange` says fully credential-complete; actual :72–89 is **any configured field OR enabled**, independent of MissingRequiredExchangeCredentialFields.
- Graph `store/ai_model.go:Get` says raw/user-prefixed id tolerance; current :96–123 checks exact id under user then default-user candidates. Graph `hasUsableAPIKey` claims placeholder refusal; current :191–207 accepts any nonblank string or provider env key. Graph UpdateWithName mentions clamps/wallet-specific handling not present in this function.
- NT8 `recordClose → ProcessTrade` historical edge is now `recordClose → ProcessTradeWithExitReason` at `trader/ninjatrader/close_sync.go:169`. Store builder then delegates to handleClose and ExitCauseFromBroker; broker-cause preservation is now part of the boundary.
- Position statistics graph omits corrected-PNL-only population and unresolved count semantics, account filters, and unknown-close streak breaking. Current function names alone do not reveal these policy changes.
- Strategy graph omits DayPlan/Regime, reset/counter accounting and latest structural-stop policy. Its language about per-symbol grid configs overstates this single `*GridStrategyConfig` field.
- Plan immutable-doc/lifecycle-mutable distinction and trader id in `PlanDB.StrategyID` have no historic graph coverage.

## Validation and remaining work

[A] Existing tests manually read: complete `preserve_ai_config_test.go`, `knob_registry_test.go`, `structural_geometry_test.go`; excerpts of `pnl_truth_test.go`, `level_state_aging_test.go`, and `arm_state_test.go`. Structural geometry fixture verifies distinct refusal count survives pending/admitted cycles and scopes trader/version. Arm-state fixture exercises production store readers with whitespace/Unicode forms, which is stronger than a standalone predicate comparison. Test paths discovered but **not fully reviewed** include AI-model multi-entry, alert dismiss, bar contract/source, one-setup and further level-state tests. No new or existing tests were executed by this worker; root owns reproduction and merged validation.

Root update received at closeout: **root reports F15-1 separately reproduced and repaired at `576bd75b`**. This review remains evidence about base `63968be`; this worker did not inspect or validate that separate fix. The principal additional actionable finding is **F15-2 AI config lost at the actual serialization boundary**; follow with **F15-4 serialized-schema census gap**, **F15-5 partial follow writes/readable counts**, and **F15-3 unfiltered prior OR history**. Root should choose isolated fixtures before patching them. None warrants altering owner trading controls or accounts.

All assigned source coverage is complete; unresolved items concern runtime state, caller reachability beyond the named excerpts, concurrent/error reproduction, imported-history interior seams, and full consumer parity. This slice does not establish whole-repository understanding or live trading correctness.
