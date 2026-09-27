# Assignment 01 — API control surfaces and manager boundaries

Source pin: `63968be62e44db2fb07a92883e02127b9064b0be`. Isolated worktree `/tmp/nofx-understanding-market-20260913` verified clean and at this HEAD before reading. Despite its directory name, assignment 1 is `api-auth-and-manager`: **24 assigned files, 6,649 lines, 144 named functions/methods**, all manually read in full. Seven anonymous callbacks are covered under their parents. `functions.json` includes every named declaration with line boundaries, semantics and syntactic call expressions; those expressions are not asserted as type-resolved graph edges.

This is a source understanding audit, not a live incident investigation. **[A]** means source directly read during this run; **[B]** means an inference explicitly bounded by prerequisites. No service calls, live database reads/writes, orders, runtime environment reads, source mutations or tests were executed. Findings below are static defects/risks; none is described as an observed attack or actual trade failure. The assigned pin is not independently certified as the running binary. Root owns reproduction and repairs.

## Rules and freshness

Read root AGENTS.md, tracked CLAUDE-canon in full, AUDIT-CHECKLIST classes 1–22 and pre-audit R1–R10/pre-cutover excerpts, SYSTEM-MAP settings/UI and relevant bar sections, and RULEBOOK authority/risk/owner-policy excerpts. Relevant classes: 8 (resolved knobs), 9 (binding rather than is_active), 24/53 (effective boundary and production-call-site parity), 49 (missing versus computed values), and R4/R6/R9 (line evidence, bounded verdict, isolation). Historical prose is not runtime proof.

Latest commits for reference documents, read at this pin:

- `07b53e65 cleanup batch 1 — B5 the mutation harness that cannot fake a verdict, B6 the worktree-add check` — CLAUDE-canon.
- `dfda15e1 test: isolate session clock fixtures from weekly backfill workers` — AUDIT-CHECKLIST.
- `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap` — SYSTEM-MAP and VL-TRADING-RULEBOOK-v1.

The September 13 correction is explicit: the owner loss control is **daily**, and no additional mandatory per-trade cap is implied here. No policy edits are proposed by this worker.

## End-to-end boundary map

[A] `NewServer` (`api/server.go:37`) constructs Gin with logging/recovery, CORS, CryptoHandler and exchange-state cache. `setupRoutes:82` builds public `/api` routes and protected group `authMiddleware` plus `planTraderOwnership`, then mounts UI fallback last (`:639`). `Start:894` binds configured interface or `127.0.0.1`; no read/header/idle timeout is set in its `http.Server` literal. CORS is wildcard, with OPTIONS terminated before handlers (`:66`). Health only reports process response/revision; it does not certify bars, account or execution safety (`:643`).

[A] `authMiddleware:851` validates exact `Bearer <token>`, blacklist, and JWT, then sets user_id/email. The ownership middleware's name can overstate its reach: `api/handler_plan.go:97–131` only checks routes whose full paths start `/api/plan/` or `/api/risk/`. All other protected handlers need their own owner resolution. `getTraderFromQuery` (`server.go:798–848`) provides that resolution to its callers: authenticated requests select only the user's own trader; public requests may explicitly name any existing trader. It invokes `LoadUserTradersFromStore` before resolving, so this helper is not a pure read of memory.

[A] Manager identity is global: `manager/trader_manager.go:56–65` indexes by trader ID with no user argument. Its `RemoveTrader:359–373` stops a running trader and removes the global entry. Scoped DB calls and global runtime calls therefore need explicit ordering/ownership validation at their seam.

[A] Trader creation (`handler_trader.go:339–582`) validates enabled owned model with credential, owned strategy, supported enabled exchange/credentials, probes broker equity, persists a stopped trader, and attempts runtime load. Persistence success returns201 even if runtime cannot initialize, with `startup_warning`. Update (`:585–764`) verifies the existing owner, merges defaults, persists, removes/reloads, and may launch Run asynchronously. Start (`:795–897`) verifies owned full config, refuses empty NT account binding, reloads fresh config, starts Run asynchronously, and warns rather than failing on status persistence. Stop (`:900–935`) verifies ownership before runtime Stop. An HTTP success at these asynchronous boundaries is not a broker or running-loop acknowledgment.

[A] Strategy editing (`strategy.go:174–440`) defaults omitted create config; clamps limits, validates indicator periods, merges nested update fields and preserves AI configuration across unconfirmed type changes. Saving logs resolved diffs and reloads all user traders bound through `Trader.StrategyID`. Legacy `activate` only changes `is_active` (`:461`); it is not binding. Prompt preview (`:567`) builds a system prompt using simulated equity and configured first symbol. Test-run (`:623`) fetches market/quant data and optionally calls the selected user's provider (`runRealAITest:806`), but has no execution call. A request called test-run may still make network/data/paid AI calls.

[A] NT chart flow (`handler_klines.go:23,345`) dispatches exchange=ninjatrader to the shared `market.FuturesBarsProvider`; unbound/nil/nonfutures returns `[]`, no crypto fallback. SSE (`handler_bars_sse.go:25`) consumes an opaque ticket, injects the user, resolves owned trader and reads that TCPTrader's BarCache. `stream_ticket.go` generates32 random bytes, retains user/30s expiry under mutex, deletes on first consume. The **ticket's admission** expires after30s; the already-open stream itself remains until disconnect/write error. Snapshot then 1s polling resends bars at/after lastT, with15s keepalive. Late corrections earlier than lastT are not emitted by this loop.

[A] NT account read (`handler_account.go:43`) returns account list, persisted selected binding and effective current account when a bound snapshot exists. Selection (`:109`) checks AddOn IsSim and optional allowed-account list; it resets only this TCPTrader cache and persists its binding, intentionally no shared account_select wire command. Manual position close (`handler_trader_status.go:124–317`) verifies ownership, then NT uses existing TCPTrader CloseLong/CloseShort(symbol,0); crypto instantiates a temporary adapter and records where appropriate. Debug test trade (`handler_debug.go:149`) resolves owned futures/TCP trader and calls DebugPlaceTestTrade directly, bypassing AI/risk by design. Armed seam (`handler_armed_seam.go:15`) explicitly checks ownership and delegates place/place_stop/cancel to AutoTrader test helpers; downstream env/SIM guards were not reread in this slice.

## Ranked static findings and repair boundaries

### S1 — foreign trader deletion reaches unrelated equity and runtime state

**[A] PROVEN source path; runtime exploit UNVERIFIED.** Input: valid JWT for user B, `DELETE /api/traders/A` where A belongs to another user. Route is registered protected (`server.go:205`); JWT admits B, prefix ownership middleware passes the non-plan/non-risk path. `handleDeleteTrader` (`handler_trader.go:767–792`) does not verify owner. It calls `store.Trader().Delete(B,A)`. `store/trader.go:223–229` first deletes **all EquitySnapshot rows by trader_id=A**, without user scope, ignores that deletion's error, then deletes Trader where ID=A AND UserID=B and returns only `.Error`, not a matched-row result. With ordinary zero-row deletion success, the handler continues to global manager GetTrader(A), Stop and RemoveTrader(A). Manager lookup/removal are global (`manager/trader_manager.go:56,359`).

Consequences are conditional on a foreign trader ID, reachable authenticated endpoint and ordinary DB success; this audit did not test them. Repair should verify owned trader before any dependent write/runtime mutation, make store deletion scoped/transactional, and assert no foreign equity or runtime effect. Existing `handler_plan_idor_test.go` only covers plan/risk middleware and query-helper ownership, not this registered delete route. `handler_trader_test.go` only exercises validation helpers.

### A1 — additional protected read/account endpoints miss ownership

**[A] PROVEN missing source checks; cross-user output/state effect UNVERIFIED at runtime.** Same valid-user premise and prefix middleware bypass:

| Route | Handler | Unscoped boundary |
|---|---|---|
| GET /config/resolved?trader_id=A | config_resolved.go:193 | Manager.GetTrader(A), exposes three resolved strategy values |
| GET /accounts?trader_id=A | handler_account.go:43 | Global trader/TCP list and GetByID persisted account |
| POST /account/select?trader_id=A | handler_account.go:109 | Global TCPTrader.ResetAccountState before scoped store write |
| GET /audit/decisions?trader_id=A | handler_decisions.go:28 | Global trader store audit records, optional account |
| GET /desk?trader_id=A | handler_desk.go:26 | Global DeskStripAt account/position read model |
| GET /traders/A/grid-risk | handler_trader_status.go:27 | Global grid risk model |

The account-binding **DB write is user-scoped** (`store/trader.go:206–210`), so this is not evidence that user B can persistently rebind A. The defect is earlier runtime reset and misleading success on zero-row write/failure. Do not turn this into an unsupported persistent-rebinding claim. `TestConfigResolvedIsRegisteredProtected` verifies only registration behind JWT; its bare handler tests do not supply a manager/foreign trader. Decision-audit tests (`handler_risk_test.go:152–179`) check missing ID/invalid since only. A broad ownership middleware or explicit owner checks need tests on actual registered handlers, not only a synthetic `/plan/today` callback.

### A2 — validation can reject after configuration is already persisted

**[A] PROVEN source ordering; concrete overflowing fixtures not executed.** `handleUpdateStrategy` persists via Strategy.Update (`strategy.go:368`) and logs diff (`:378`) before token overflow check (`:380–397`). If all known limits are exceeded, response400 arrives after storage changed and before reload. A user believes save failed while a future reload uses the rejected config. `handleUpdateModelConfigs` likewise runs UpdateWithName (`handler_ai_model.go:217`) before ValidateThinkingKnobs (`:225`). Invalid thinking input returns400 after ordinary model fields changed; no reload follows. Map iteration can make a multi-model request partly applied before another entry fails. UpdateThinking failure is only WARN. Validate first; group intended DB changes transactionally and explicitly expose reload failure rather than implying active runtime parity.

### A3 — trader-ID tests validate a different algorithm than production

**[A] PROVEN disagreement; collision occurrence UNVERIFIED.** Production `handleCreateTrader` (`handler_trader.go:414–419`) uses first8 exchange-ID bytes + full model-ID + `time.Now().Unix()`. Same exchange/model within one second generates identical ID. `api/traderid_test.go:45–47` instead defines its own `generateTraderID` using UUID; TestTraderIDUniqueness and TestTraderIDNoCollision never call production ID generation. Their green result cannot certify production uniqueness. Extract/use one production generator and drive actual create boundary with a controlled clock/ID source; preserve negative concurrent create coverage.

### A4 — synthetic crypto close accounting is marked FILLED

**[A] PROVEN code; not an NT execution defect.** NT returns early through real TCP close, and most crypto brokers are skipped for OrderSync (`handler_trader_status.go:322–326`). The remaining path (notably kucoin in this switch) passes entryPrice into `recordClosePositionOrder` as exitPrice (`:309`). Missing price becomes `quantity*100` (`:361–367`), commission is assumed0.04%, and FILLED order plus fill is inserted without confirming broker fill. Missing orderId becomes `fmt.Sprintf("%v", nil)` ("<nil>") and is not excluded by the empty/zero check. This violates no-fabricated-values for the residual crypto path; do not report that NT fills use fabricated prices. Fix by using broker execution evidence or explicit unresolved status, and test each sync/non-sync branch.

### B1 — account and lifecycle reporting can overstate success

[A] Account selection reports200 even when UpdateAccount fails (`handler_account.go:205–216`), after resetting state. `handleGetAccounts` builds per-item IsCurrent before overriding top-level Current to bound snapshot (`:70–98`), so the two current markers can disagree. Start reports started before asynchronous Run completes (`handler_trader.go:881–896`); config reload failures are mostly WARN-only. `handleSyncBalance` computes percentage dividing by oldBalance without a zero guard (`handler_trader_status.go:90`), so zero baseline can produce non-JSON finite values after the DB write. Its NT probe is `ntTrader.New` (`exchange_account_state.go:262–266`), the legacy config constructor rather than the running bound TCP instance; this boundary warrants NT balance validation by the execution worker.

### B2 — public analytics remain independent of removed pages

[A] Public `/traders`, `/competition`, `/top-traders`, `/equity-history`, batch and public config remain mounted (`server.go:114–120`). Single equity history accepts any existing explicit ID when no JWT context (`getTraderFromQuery:814`); batch directly reads each requested ID (`handler_competition.go:328`). These are intentional public route registrations, not a claim of JWT bypass. Whether public competition respects visibility is a manager dependency not fully audited here. **Direct equity reads do not check ShowInCompetition in the inspected code.** Removing Competition/Strategy Market pages did not remove their APIs. Bound deployment exposure is unknown; listener defaults loopback.

### B3 — onboarding has global credential side effects

[A] `handleBeginnerOnboarding` (`handler_onboarding.go:43–94`) returns raw wallet key deliberately, stores user model, sets process-wide CLAW402 env and rewrites shared .env. `resolveBeginnerWallet:155–207` may adopt orphan model. UpsertEnvFile (`:286`) rewrites nonatomically, only replaces first duplicated key, and mode0600 on WriteFile does not tighten an existing file's mode. These are source risks in a multi-user/control-plane design, not proof of leaked secrets. No actual credentials were inspected. Error responses may reveal env path/persistence reason by design. Concurrent preferences similarly use unguarded read/modify/write (`agent_preferences.go:34,67`).

### B4 — route documentation disagrees with handlers and can instruct the agent incorrectly

[A] `GetAPIDocs` injects registered schema verbatim (`route_registry.go:50`). Examples: trader create/update schemas say minimum scan3 while handlers accept positive1 (`server.go:198,204`; `handler_trader.go:455,648`); update says only fields being changed while Name/AIModelID/ExchangeID are binding-required; close-position schema omits required Side; duplicate docs suggest automatic copied name while handler requires Name (`strategy.go:489`); active strategy description says currently in use although bindings determine use; account select return schema says current but handler sends current_account. Strategy schema embeds deprecated NofxOS guidance. `routeRegistry` is global append-only; constructing multiple servers duplicates docs and concurrent construction has no synchronization here. These are concrete drift cases, not evidence every caller fails.

## Remaining file-specific semantics and limits

[A] `config_resolved.go:79–189` is a good no-fabrication pattern: saved unset differs from resolved default; resolver metadata exposes consumers but no secret values; uncomputed resolved list is absent, computed empty arrays are arrays. Resolution covers RR, plan mode and HTF veto only, not a full effective-config dump.

[A] `handler_expectancy.go:38–74` invokes global LoadAndBuildAt and publishes min_n, excluded/unresolved metadata, era filter and criterion. It is protected but not user-scoped in this handler. Model math/SQL is outside this slice. `handler_plan_geometry.go:5` returns nil/WARN on unavailable geometry; `openPositionProvenance` (`handler_plan_position_provenance.go:20`) selects first open position, distinguishes unrecorded version, and does not account-filter; multiple open rows or account switching need downstream contextual review.

[A] `handler_competition.go:128–188` single curve's percent is unrealized/base balance, whereas batch (`:328–448`) percent is (equity-initial)/initial. Both are computed but not equivalent metrics. Batch may append live point only if last snapshot older than30s; default history is90days despite500-record comment. Decision latest caps100 and reverses store order; broad decisions uses10,000 even though docs suggest limit parameter. Audit handler passes limit onward rather than capping itself; store maximum was not followed.

[A] External chart adapters use context.Background rather than request context; CoinAnk may fall back to Binance, TwelveData invalid parse leaves a zero element, Hyperliquid ignores numeric parse errors. These observations apply to those adapters, not live NT feed. `handleSymbols` lowercases switch selection but checks original exchange string inside, so mixed-case Hyperliquid may skip crypto mids.

[A] `errors.go` only scrubs errors routed through its helpers; arbitrary public messages are not automatically cleaned. `isValidPrivateKey` checks length/prefix not hex validity; onboarding's separate walletAddressFromPrivateKey performs actual ECDSA parsing. Utilities mask secret bytes but deliberately preserve URLs/addresses. Authenticated decrypt returns plaintext when transport encryption is enabled; no finer role/ownership check is visible in this handler. Agent chat grants authenticated sessions CanExecuteTrade=true and CanViewSensitiveSecrets=false (`agent_routes.go:13–36`); actual trade/secret policy enforcement is delegated to agent code and not certified here.

## Historical graph reconciliation

[A] Read historical Understand Anything file summaries for18 matching assigned files and first25 outgoing non-contains edges; selected115 nodes/640 outgoing edges exist but were **not all manually read**. July10@7a8adce0 is historical. Critical correction: handler_account summary says selection sends account_select; current `handler_account.go:183–202` explicitly removes that wire action. Handler_trader_status graph says protective-order cancellation/asynchronous polling across all supported exchanges; current inspected close handler has NT early return, skips OrderSync brokers and does not call pollAndUpdateOrderStatus. Graph imports connect a production API file to many store test files; these are coarse package expansion, not actual Go file imports/type-resolved calls. Server summary's350-line setupRoutes is now559 lines. Six current assigned files lack historical file nodes (config_resolved, armed_seam, desk, expectancy, plan_geometry, plan_position_provenance). No graph reindex/update was performed. Root's CGC evidence remains separately scoped; this worker did not query CGC.

## Verification plan and evidence limits

Read in full: config_resolved_test.go, handler_trader_test.go, traderid_test.go, handler_plan_idor_test.go; selected decision-audit tests. **Not run**: no broad or targeted tests, no reproducers, no network or live data. Highest-value follow-up is isolated test DB and actual protected route dispatch for S1/A1, asserting both DB and runtime remain untouched on foreign IDs. Then exercise rejected config writes, production ID collision source, NT balance source, and residual crypto accounting. Test both owner-success and foreign-refusal; superficial JWT registration is insufficient. No claim about current users/counts, effective switches, live broker inventory, P&L rows, deployment or main startup latch is made from this API slice.
