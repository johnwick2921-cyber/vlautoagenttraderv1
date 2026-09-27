# Assignment 05 — conversational agent management and HTTP boundaries

Base: `63968be62e44db2fb07a92883e02127b9064b0be`. Read-only isolated worktree: `/tmp/nofx-understanding-market-20260913`. Assignment: 14 source files, **9,224 lines fully manually read**, **299 named functions/methods** inventoried. Anonymous callbacks are grouped under their enclosing function. No assigned source gaps. Additional dependency coverage is explicitly partial in `reads.json`; this is not a whole-agent or whole-repository certification.

Evidence notation: **[A]** directly read source/control flow; **[B]** consequence inferred from that source; **[C]** unresolved hypothesis. No test, exploit, trade execution, live API request, or DB mutation was performed. Findings below are static concerns, not reproduced runtime incidents. Source hashes were verified again while assembling artifacts.

## Rules and provenance

Read repository AGENTS instructions, tracked `docs/superpowers/CLAUDE-canon.md`, audit checklist excerpts (1–160, 2281–2310), SYSTEM-MAP 329–359 and VL-TRADING-RULEBOOK-v1 1–65. These are an audit reference, not an implementation spec. Recorded source-history lines from the reading:

- CLAUDE-canon: `07b53e65 cleanup batch 1 — B5 the mutation harness that cannot fake a verdict, B6 the worktree-add check`.
- AUDIT-CHECKLIST: `dfda15e1 test: isolate session clock fixtures from weekly backfill workers`.
- SYSTEM-MAP and VL-TRADING-RULEBOOK-v1: `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap`.

NT8 remains the single live futures data/execution route; SIM-only and immutable owner account/credential bindings remain governing instructions. Crypto-era USDT helpers in this slice are not justification to add mandatory per-trade caps to futures. No source changes are proposed or performed by this worker.

## Actual flow and boundaries

**HTTP identity and conversation state.** `api/agent_routes.go:11–42` authenticates chat routes and attaches the real store owner and trade policy. `web.go:82–188` parses messages, applies 55s/120s request deadlines and forwards the real owner plus a numeric conversation ID. Nonzero body `user_id` survives; only zero derives `SessionUserIDFromKey(owner)`. `agent.go:428–525` keeps these identities separate: store owner selects AI/store scope, numeric ID selects history, task state and skill/proposal/workflow state. `/clear` clears those numeric-keyed states. Raw incoming text is logged at `agent.go:439,487`, before the planner; catalog/onboarding prompts can solicit credentials, so logging policy/redaction needs a separate reviewed boundary.

**Skill routing and session management.** `skill_registry.go:59–177` loads embedded JSON; malformed JSON panics at initialization, duplicate skill names overwrite, and normalization is chiefly trimming/map preparation, not complete semantic validation. Registry context builders `:193–721` construct cached planner guidance, action contracts and field descriptions. This is not tool-side authorization. `skill_semantic_gate.go:10–246` renders fields/options and looks up owner-scoped strategy type; malformed config falls back toward AI. Option load failure and an actually empty list may both render no options. `skill_dag_runtime.go:5–50` resolves a missing/bad cursor to first step and follows only first successor; terminal advance leaves the cursor unchanged.

`skill_management_handlers.go:28–128` normalizes aliases and mirrors trader fields into old slots. Empty setters do nothing, so they cannot clear a stale value. `:203–226` refreshes target references by ID then same-name fallback; this is owner-scoped but not immutable identity binding. `:248–333` loads domain options and routes creation or simple management. `:2528–2638` resolves targets, supports bulk delete and dispatches to `skill_execution_handlers.go:1466–2473`. Lifecycle/delete paths generally require confirmation; update paths build/validate sparse payloads and delegate to owner-scoped tools. Owner checks in those tools are not fully audited here. Negative result: `loadEnabledModelOptions` (`skill_dispatcher.go:295–317`) actually includes disabled models, so a suspected inability to select a disabled model is **not supported**.

**Strategy creation/update.** Creation builds normalized draft/type/nested patch configuration (`skill_management_handlers.go:335–538`), strips protected risk fields, checks explicit required fields (`:567–766`) and presents summaries (`:768–1455`). AI/grid requirements differ; grid needs ten fields plus ATR or manual price bounds. Presence checks establish a path was supplied, not that its value is valid. Concrete creation occurs in `:2377–2457`, ultimately calling `toolManageStrategy` with `confirmed:true`. Config updates (`skill_execution_handlers.go:2529–2717`) load existing owner config, merge/clamp proposed patch and defer warnings for confirmation. Deferred state contains a **full config snapshot**; later persistence has no revision compare, so a concurrent config update may be overwritten [B]. Existing malformed config JSON is ignored in `loadStrategyConfigForUpdate:2631` rather than surfaced as a decoding error. Actual store merger/tool validation was not fully followed. The legacy `applyStrategyConfigPatch:830–1072` is a separate typed-field switch with parse checks and locked-field refusals, not the current nested merge validator.

**Persistent memory/preferences.** `memory.go:82–289` stores task state under numeric session ID, trims/deduplicates and heuristically filters open loops; it does not guarantee all model-generated facts are grounded or all secret-bearing fields are filtered. Missing timestamps are filled during normalization. Compression `:317–349` summarizes old history only beyond message/token thresholds, saves state first and then replaces history with recent messages. Failed summary/save retains old history. Incremental update `:350–483` uses a short window. Concurrent caller serialization around replace/read-modify-write was not established. Token estimation is rune/3 plus overhead, not model tokenizer output. Environment integer overrides accept negatives; cap snapshot can say “set” when invalid text caused fallback.

`preferences.go:21–166` validates additions (nonempty, ≤500 runes), prepends entries, caps count at 20, updates/deletes the **first** ID or case-insensitive substring match, and emits prompt bullets. Update does not repeat the 500-rune check [A]. JSON failures look like absent preferences. Read/modify/write has no local atomic transaction or mutex; caller serialization is unverified.

**Catalogs, onboarding and localization.** `entity_field_catalog.go:1–111` defines manual versus agent-editable fields/aliases; it is capability metadata, not validation. `i18n.go:80–90` and `onboard.go:599–607` provide localized templates. `model_provider_catalog.go:21–242` is a static eleven-provider catalog with defaults/custom URL/custom model flags, API-key/wallet guidance and a recommended provider. No remote provider claims were verified. Legacy wizard `onboard.go:108–358` uses numeric session state, seven crypto exchanges and independently maintained older AI defaults; direct Chinese conversation goes through planner rather than setup command recognition. Secret values are kept in memory while step metadata persists, so a restart can resume a step without its credential data. `needsSetup:37` checks global traders, not owner traders; current entrypoint reachability was not demonstrated. `saveSetupExchange:432–482` can update an existing same-owner/default-name exchange and force enabled/mainnet config; `saveSetupAIModel:484–508` updates owner/provider model. `createTraderFromSetupForStoreUser:364–430` reuses a matching binding or creates a stopped trader without strategy. No such writes were executed.

**Market display and background watcher.** `web.go:207–366` public market proxies always target Binance futures; symbol/interval shape, limits, body bounds and timeouts are present. Batch ticker is capped at 20, concurrently fetched, output order retained for successes while failed entries disappear. It does not merge stock data or read NT8. `HandleHealth:77` is static “ok,” not a bridge/data liveness probe. `stream_text.go:5–49` partitions output into callback chunks; SSE encoding uses JSON string escaping (`web.go:191–205`) and has no heartbeat producer.

`sentinel.go:54–223` is an independent Binance watcher, not live NT8 decision input. Start is not once-guarded; Stop closes once; Add/Remove compare exact symbol case; removed-symbol history remains. At least five observations compare current to `h[len(h)-5]`, spanning four sample intervals, while the alert says five minutes. Sequential scan/network delay further means no exact duration guarantee. Volume compares rolling 24h quote-volume samples, not per-minute traded volume. ParseFloat errors are ignored; no finite/positive input check is present. Alerts have no dedup/throttle. Funding alert enum exists but no funding alert is emitted here.

**Diagnosis.** `skill_execution_handlers.go:2789–3199` resolves an owned trader, collects safe entity metadata, runtime status/account/positions, five recent decisions and thirty logs, then asks AI under a bounded timeout or formats fallback. The reduced decision evidence loses row IDs, so diagnosis output cannot satisfy the sample-ID law from this representation alone. Fallback uses persisted TraderConfig.IsRunning rather than collected runtime flag and prioritizes text-pattern historical errors; a stale error may misdescribe a later waiting/healthy decision [B]. Account failures can become empty evidence; USDT regex and candidate-first symbol selection are crypto-oriented. These are static evidence-quality limitations, not an observed false diagnosis.

## Findings prioritized for root

### R05-1 — high: authenticated owner and mutable numeric session identity diverge

**[A]** `web.go:102–104,144–146` only derives identity when body `user_id==0`. `api/agent_routes.go:13–35` adds owner context but never rewrites that body field. `agent.go:428–525` passes supplied numeric ID into conversation state and clear operations. `memory.go:82–123` and `preferences.go:53–146` use that numeric key for durable state.

**[B]** A caller able to address another numeric session key can cross its conversation state boundary, including clearing state via `/clear` and potentially reading its context through normal responses. This does **not** establish cross-owner CRUD access: actual management tool calls continue receiving authenticated `storeUserID`. The route requires authentication; this is not an unauthenticated route finding. No running server or other user's state was accessed. A safe root-owned regression would assert body IDs cannot change authenticated session scope in both HTTP variants.

### R05-2 — high: direct trade confirmation is not bound to owner/policy/selected trader

Additional dependency trace, reported to coordinate with agent/03:

- `trade.go:40–55` TradeAction has TraderID but no owner; pending map `:57–91` is keyed only by trade ID.
- `handleTradeConfirmation:438–520` accepts text command ID, reads pending item, enforces large-order wording, removes it and calls execute. Its `userID` argument is unused; it does not recheck session policy/server flag or item age.
- `agent.go:452,499` calls this before planner routing.
- `executeTrade:175–210` revalidates trade numbers/risk and calls the broker. **Those safeguards are real and not bypassed by this finding.** `resolveTradeExecutionContext:212–248` chooses the first running manager trader matching stock/nonstock classification, without owner or TradeAction.TraderID binding.
- In contrast, proposal tool `tools.go:2731–2738` checks authenticated session and AllowTradeExecution. That protection is at proposal time, not confirmation time.

**[B]** Knowledge of a pending ID can suffice at this local handler to confirm someone else's intent; manager ordering may select a different eligible trader. Separate Get/Remove locks also leave concurrent confirmation check-and-consume non-atomic. No double execution or cross-owner execution was reproduced. Expiration cleanup exists (`CleanExpired`), so lack of age check is a stale-window concern; cleanup schedule is not established by these excerpts. Downstream NinjaTrader SIM/order guards remain outside this local trace and must not be weakened. Main trade implementation outside quoted ranges remains an explicit dependency gap.

### R05-3 — medium/high: pending prompt flag substitutes for current consent

**[A]** `handleStrategyCreateSkill:2428–2450` only asks if **both** current text is not affirmative **and** awaiting-final-confirmation flag is false. Once the flag is true, execution writes with `confirmed:true` regardless of current text. Explicit cancellation is handled earlier and does refuse. Upstream active route `central_brain.go:362–395` consults `guardStrategyCreateBeforeFinalConfirmation`, but that guard returns no block when the flag and prior prompt exist (`:631–638`).

**[B]** If planner routes a nonconsenting follow-up to execute, handler does not independently require affirmative current-turn consent. This is a **conditional static path**, not a demonstrated natural-language exploit; normal planner routing can ask or cancel instead. Fix/test ownership stays with root.

### R05-4 — medium: English creation instruction collides with trade command

**[A]** strategy summary `skill_management_handlers.go:1191` instructs `confirm create`; `strategyCreateConfirmationReply:540–553` accepts yes/ok/Chinese confirmations but not that phrase. More directly, `handleTradeConfirmation:441–465` parses any `CONFIRM <token>` first; with a nonnil pending store it treats “create” as trade ID and returns “Trade expired or not found.” `agent.go:452,499` intercepts before planner. Thus the recommended English phrase reaches the wrong handler under the nonnil-pending-store condition. No HTTP reproduction performed. A pure-agent regression can test recommended text against both interceptors without trading.

### Other bounded concerns and negative results

- Generated prompt update confirmation (`executeStrategyPromptUpdate:2475–2527`) also falls through on unrecognized reply when its flag is set, but no production setter of `_requires_generated_confirmation` was found. Treat as dormant/helper issue until reachability is proven.
- Bulk deletion asks about current aggregate set and re-enumerates eligible stopped owned traders on execution (`:1801–1900`); it does not bind approval to immutable IDs. Not a claim that running traders are deleted.
- `normalizeCoinSymbol:1125` turns MNQ into MNQUSDT and uppercases continuous futures suffixes. This helper is crypto-specific; current production futures reachability was not established and it must not be cited as a proven live MNQ failure.
- `parseLooseTextValue:228` and `parseStandaloneTraderUpdateArgs:253` are empty stubs; don't document them as active extraction.
- Unsupported custom endpoint clearing calls `setField(..., "")`, which ignores empty values. A stale custom endpoint can remain in model creation state; downstream validation may reject it.
- Exchange creation initially demands API key and secret for every exchange (`:2127`), despite provider-specific credential alternatives. NT8 onboarding readiness is not established by this crypto-oriented path.
- `formatFieldKnowledgeLine:596` renders numeric min/max with `%.0f`; fractional constraints lose precision in planner guidance. Ordered field fallback uses map iteration, not stable sorting.

## Tests read, not run

- `preferences_test.go` (31 lines): constructor empty/trim cases; no update-length or ambiguous-match regression.
- `onboard_test.go` (27): direct setup command test skipped and contains stale Chinese expectations.
- `model_provider_catalog_test.go` (57): three provider guidance string tests.
- `skill_registry_test.go` (55): registry/action requirement loading.
- `memory_test.go` (134): skipped old compression expectations, active incremental-summary/temp-store and normalization tests. No live DB was opened.

No broad suite was run merely to imply verification. HTTP owner binding, pending confirmation policy/atomicity, current-turn strategy consent, and recommended English confirmation deserve focused root-owned tests. Existing tests outside these five files are not represented as reviewed or absent.

## Historical graph correction and explicit gaps

Historical Understand Anything graph is July10@7a8adce0, 3,121 nodes / 9,588 edges; its selected summaries were compared to current source. `graph.json` captures selected historical nodes and incident edges as **unverified historical material**, plus source-verified corrections. It is not current type-resolved call graph. Root AST syntactic calls are labeled syntax-only in every function record. CGC was not independently queried by this worker; root coordinates the historical export.

Corrections: preferences update has no creation length check, delete has no index fallback, add prepends and context is bullets; Sentinel Add/Remove return no bool and funding emission is absent; DAG terminal cursor is retained; registry fallback ordering is not always stable; applyStrategyConfigPatch does not clamp; generic action executors generally delegate creation elsewhere; web batch has no stock merge and SSE no heartbeat.

Explicit gaps: full planner/central-brain/tool authorization implementations, full trade.go (151–174 and 291–349 not read), store merger and transactional semantics, manager-to-broker selection ownership, runtime session serialization, current deployed binary behavior, frontend request construction, public proxy consumers, upstream provider availability, root-owned repair validation. Assigned source coverage is complete. No runtime incident is claimed from static gaps. Historical graph edge data exported here must not be mistaken for manually resolved current dependencies.

Graph label follow-up: all 226 selected non-containment incident edges were inspected. Historical `imports` fan out package membership into individual files, including test files; do not interpret those as production imports of tests. Current API route call edges match source. The 100 historical containment edges are provenance, superseded by the current 299-function census.
