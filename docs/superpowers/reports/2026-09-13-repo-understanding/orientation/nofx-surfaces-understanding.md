# Repository understanding — UI, API, configuration and persistence

Base: `63968be62e44db2fb07a92883e02127b9064b0be`. Isolated worktree: `/tmp/nofx-understanding-surfaces-20260913`. Read-only orientation assignment 3, 2026-09-13. Evidence ledger: `/tmp/nofx-surfaces-reads.json`.

[A] means source directly read; [B] means consequence inferred from that source; neither means a runtime reproduction. No production files, database, accounts, configuration, orders, processes or index were changed. No test suite was run. This report is a connected critical-path reading, **not a claim to understand every function in this repository**.

## 1. Operating constraints and scope

The inherited owner instructions apply. The tracked `docs/superpowers/CLAUDE-canon.md` was read in full: its keeper-on-acquire rule supersedes the stale manual-heartbeat instructions. Worktree verification used `pwd` and `git rev-parse HEAD`; the observed base equals the assigned base. No source checkout edits were made. No tracked subsystem `AGENTS.md` was found; explicit checks for `web/AGENTS.md`, `api/AGENTS.md`, `store/AGENTS.md`, and `manager/AGENTS.md` found none in this isolated checkout. Do not pretend the missing ignored local files were read.

Relevant audit laws read: checklist classes 8 (saved versus resolved), 9 (binding versus active flag), and 114–115 (verify the verification method; labels must not imply unmeasured coverage); additional keyword excerpts cover Guide/config drift. The 4,673-line checklist and large map/rulebook were **not** fully read. The map and rulebook are useful navigation, not authoritative proof of the current function body. Example observable drift: map UI section says 14 Guide sections, whereas current `GuidePage.tsx:32` includes 15 entries.

## 2. Actual connected control surfaces

### Browser entry, authentication and selected identity

[A] `web/src/App.tsx:7` nests LanguageProvider → AuthProvider → ConfirmDialogProvider → SandboxBanner/AppRoutes. `web/src/contexts/AuthContext.tsx` restores token/user from browser storage, checks JWT expiry locally, posts login/register, listens for `unauthorized`, and removes browser authentication state at logout. That local JWT decode is a convenience check, not authorization.

[A] `web/src/lib/httpClient.ts:83` installs a Bearer-token interceptor; `handleError` clears auth and redirects on 401. Its general timeout is 30 seconds with per-call override. Some Studio requests bypass this wrapper and use explicit `fetch` with Bearer headers (`StrategyStudioPage.tsx:204,570`), so wrapper error behavior cannot be assumed for every UI call.

[A] `AppRoutes.tsx:254` obtains traders, then selection resolution follows URL → memory → stored ID → stable fallback. `selectedTrader.ts::resolveSelectedTrader` only picks from the returned trader list. Account-specific dashboard keys append selected account (`AppRoutes.tsx:318–342`), then status/account/positions/decisions/statistics are polled separately. Two-failure latches back off account/position/decision polling and reset when trader selection changes. A selected browser trader is not an authorization boundary; server-side identity must be independently checked.

### HTTP entry and authorization

[A] `api/server.go:37` constructs Gin with default logger/recovery and CORS. Public routes include health, model/exchange catalog, registration/login, some historical competition endpoints, token estimation, and ticket-authenticated SSE. The protected group at line 146 has `authMiddleware()` followed by `planTraderOwnership()`.

[A] `authMiddleware` at `server.go:851` requires exactly a Bearer header, checks token blacklist, calls `auth.ValidateJWT`, and puts user ID/email in context. I did not read `auth/auth.go`, so JWT algorithm/key storage/token generation are outside this lane's verified scope.

[A] `planTraderOwnership` at `handler_plan.go:97` only applies to `/api/plan/` and `/api/risk/` prefixes. It resolves query/body trader ID, restores the body after probing, and checks `TraderStore.List(userID)` through `traderOwnedBy`. Unknown/unowned traders return 404. It is **not a general protected-route ownership middleware**.

[A] `getTraderFromQuery` (`server.go:798`) has separate user-scoped resolution and rejects a cross-user explicit trader. With no user on public routes, it requires an explicit existing trader. `manager.GetTrader` (`trader_manager.go:56`) is merely a mutex-protected global map lookup by trader ID; it does not enforce ownership.

### Studio save → persistence → reload → engine

[A] `StrategyStudioPage.tsx:61` accepts both nested `ai_config` and old flat fields. `normalizeStrategyConfig` at 75 writes a known field list, preserving `day_plan` and `prompt_variant`. It omits `regime`; the backend merge preserves an absent existing block, but the normalized editing/export payload is not a complete copy of server schema. No regime-data-loss-on-save claim is made without testing that merge.

[A] Fetch at 201 preserves local unsaved config and selected metadata across focus refresh. Save at 552 serializes normalized config, current language, name/description and visibility, then guarded PUT. Only HTTP success produces a success toast; the response JSON warnings/reload outcomes are not inspected in this path.

[A] `api/strategy.go:268` scopes the strategy read by user and refuses default-strategy edits. The existing config is unmarshaled; an invalid stored JSON falls back to zero config. `store.MergeStrategyConfig` (`store/strategy.go:518`) marshals the baseline, normalizes legacy patches, recursively merges maps, and unmarshals. The type-switch guard preserves AI config unless explicitly confirmed. Clamp and indicator validation follow.

[A] `StrategyConfig` (`store/strategy.go:715`) keeps Go fields flat for engine compatibility but custom `MarshalJSON` at 805 nests AI fields under `ai_config`; `UnmarshalJSON` at 846 accepts nested and legacy-flat inputs. `DayPlan` and `Regime` remain root siblings. Pointer booleans preserve absent versus explicit false. Treat these schema conversions as high-cascade interfaces.

[A] `StrategyStore.Update` (`store/strategy.go:1998`) uses `WHERE id=? AND user_id=?` and updates config/metadata/UTC timestamp. `handleUpdateStrategy` persists at 378, records config diff, then performs a token-overflow rejection, then reloads bound traders at 421. That ordering has a material finding below.

[A] `manager.RemoveTrader` at 359 stops an in-memory running trader and deletes its map entry without itself persisting a stopped flag. `LoadUserTradersFromStore` at 376 reads only that user's trader/model/exchange rows, skips already-loaded traders, chooses exact model ID before deterministic legacy provider fallback, and requires enabled model/exchange. At 575, `addTraderFromStore` loads **the exact trader StrategyID**, calls `ParseConfig`, builds `AutoTraderConfig`, creates `NewAutoTrader`, registers it, and asynchronously runs it if persisted `IsRunning` was true.

[A] `Strategy.ParseConfig` at 2133 applies missing defaults in memory, not a DB rewrite. `trader/auto_trader.go:783` hands that config to `kernel.NewStrategyEngine`. `AutoTrader.GetStrategyConfig` at 1120 returns the actual engine config when an engine exists. Consequently direct raw DB writes do not imply runtime settings changed; neither the Studio active badge nor a successful DB update proves the active engine consumed the new config.

### Daily loss and structural stop ownership

[A] `RiskControlEditor.tsx:761` renders master toggle absent→true; daily toggle at 775 also absent→true; daily USD amount at 781 writes through `updateField`. Blank input becomes zero. `store/strategy.go:1710` documents master/daily flags and the zero-value environment fallback. The editor text at 800 says `$500 env default`, which describes a compiled default, not a freshly resolved environment value; a configured env override can differ.

[A] `ResolveStructuralStop` (`store/structural_geometry.go:29`) supplies provisional MNQ buffer 4.50 points, calibration identity, cost default 2 points, and explicit overrides. The current schema has no mandatory new per-trade cap. `RiskCapUSD` remains only on historical geometry records, documented never to admit/refuse a trade. `ResolveMinRiskReward` in `store/resolve_source.go:29` is a distinct shared saved-or-schema-default resolver; structural and execution call sites need independent verification of which value they pass.

[A] This report did not read the live database and therefore makes **no claim about current owner switches or $450 enforcement**. Those must be checked through the owner-bound strategy and actual runtime resolver; the chat's prior evidence is not a fresh live read.

### Plan/desk display truth

[A] `handlePlanToday` (`api/handler_plan.go:250`) resolves active/requested session, session-instance date, trader-run session permissions and rules, then fetches the trader/session chain. A committed row remains visible while a re-read is in flight. Historical versions use the requested version and version-specific overlays. Invalid overlays are surfaced as errors, not silently displayed as applied.

[A] The payload carries independently sourced scenario status/liveness, level facts, no-trade band, replan counter, open-position provenance, one-setup recorded verdict and structural geometry. `planStructuralGeometry` (`handler_plan_geometry.go:5`) reads durable records; it does not rerun composition for display.

[A] `armedMapFor` (`handler_plan_order_truth.go:64`) selects displayed-version arms by highest placement sequence/row ID per scenario leg. It separates intended plan terms, composed ledger terms and accepted current broker terms. `acceptedOrderPrices` at 146 requires a dated snapshot, unique signal-linked orders and live accepted states; no fallback to intended/ledger prices. A later terminal placement outranks an old working placement. These are critical protections against a convincing but false display.

[A] `openPositionProvenance` (`handler_plan_position_provenance.go:20`) reads entry-time version from the first open position, not the currently displayed plan version; missing attribution stays null with a note. It does not establish that multiple open positions are fully represented.

[A] `handleDesk` (`handler_desk.go:26`) delegates to `DeskStripAt(time.Now())`; the component `DeskStrip.tsx` renders returned source/age/unknown reasons, uses server cadence 5s/15s, and replaces the whole strip with an unreachable notice on error. The underlying twelve-row computation belongs to the execution lane, not this lane's full reading.

[A] Guide types stamp an explicit build SHA (`web/src/guide/types.ts:7`). `GuidePage.tsx:456–478` compares the first 12 chars against health. Missing revision does not claim drift. Matching revision only proves code/build alignment, **not that all Guide prose is behaviorally correct**.

### Storage interfaces and binding distinctions

[A] `store.Store` (`store/store.go:15`) owns GORM and legacy SQL handles plus lazy sub-stores, including plan, armed orders, accepted risk, bar history, snapshots, touches and config changes. Construction performs table/default initialization. I only read the constructor/shape, not every migration/accessor.

[A] `store.Trader` binds user, model, exchange, strategy, account, scan cadence and persisted running state. Legacy prompt/leverage fields are retained for compatibility. `handleUpdateTraderPrompt` explicitly saves a legacy column without changing the live strategy prompt. `GetFullConfig` uses user-scoped model/exchange joins but may fall back to active/default strategy for a display/config read; manager execution loading requires the exact StrategyID instead. This difference is important when diagnosing a missing binding.

[A] `StrategyStore.SetActive` changes the legacy active flag transactionally. It does not rebind existing `traders.strategy_id`; the owner-facing badge is not the execution selector. `StrategyStore.Delete` protects default/active/in-use strategy rows. General updates often return `.Error` without checking affected rows; ownership must be checked before any downstream effects.

## 3. Concrete findings requiring independent follow-up

### F1 — authenticated cross-user trader deletion effects, high priority

[A] `server.go:205` registers DELETE `/traders/:id` in the protected group, but the prefix ownership middleware skips it. `handler_trader.go:767` performs no `GetFullConfig`/`traderOwnedBy` check. `TraderStore.Delete` (`store/trader.go:223`) first deletes equity snapshots by `trader_id` alone, then performs the user-scoped trader delete. It returns only `.Error`, not affected rows. Handler then obtains the same global trader ID, stops it if running, and calls manager removal.

[B] For an authenticated user naming another user's existing trader, a zero-row trader delete can still be followed by unscoped equity deletion and in-memory stop/removal. This is a source-supported authorization defect, **not a performed exploit**. No claims about live multi-user exposure, attack occurrence, or actual data loss. Next wave should use a disposable store plus real routed handler and fake/inert trader to prove/close this boundary. Must also scope child-row deletion and reject zero-row/no-owner before side effects.

### F2 — authenticated desk/config/account reads lack owner scoping

[A] `/desk` at server456 and `/config/resolved` at152 use JWT authentication but are outside ownership prefixes. Both handlers call global manager lookup directly. `/accounts` at620 similarly returns account list and selected trader binding with no user ownership check. Manager lookup has no user check.

[B] An authenticated caller who knows another loaded trader ID can reach its desk/config/account information through these code paths. This is **not unauthenticated access**, and no secret/token read is demonstrated. The registry alone contains classifications, but resolved fields add trader-specific settings. Need real-router cross-user tests for every ID-bearing route, not just a string check that registration says `protected`.

### F3 — account selection side effect precedes owner-scoped persistence

[A] `/account/select` at server625 is likewise outside prefix ownership. `handler_account.go:109` checks target is a discovered SIM account and optional allow-list, then resets the selected trader's TCP cache before `UpdateAccount(userID,id,account)`. That store write at `store/trader.go:206` is scoped but only returns error; zero affected rows are not detected. Persistence error is logged and handler still replies200.

[B] Cross-user calls can reach cache-reset behavior even when scoped binding update touches no row. Also an owner's failed persistence can look like a successful selection. The SIM-only check is present; this finding does **not** mean a live account can be selected. Do not exercise against the owner's bindings; prove with disposable fixtures and check reload/binding behavior in a separate narrow wave.

### F4 — strategy saved before a later validation error

[A] `strategy.go:378` writes config before the token-estimate rejection at `396–404`. `StrategyStore.Update` is a standalone GORM update, no surrounding transaction wrapping subsequent checks/reload. The return skips reload after a persisted config change.

[B] A qualifying overflow request can report HTTP400 while changed config remains saved and the old engine continues until reload/restart. The frontend retains edits and says failure, but that does not roll back the database. Need a handler-level disposable-store test asserting byte-identical persistence on **every** rejection branch. Move all admission validation before persistence or provide an explicit transaction/activation model; no fix was made here.

### F5 — saved-success does not prove runtime reload succeeded

[A] `LoadUserTradersFromStore` records individual add failures in `loadErrors` then returns nil (`manager/trader_manager.go:456–472`). The save handler's reload branch only checks the aggregate error and can log that all removed traders reloaded. Its response is still success, and UI checks status only.

[B] A removed trader whose reconstruction failed may remain absent while Studio shows saved success. Need per-trader reload results, persisted-versus-active revision visibility, and inert-constructor failure tests. This is distinct from F4: a valid save can succeed in storage and fail in activation.

### F6 — narrow tests and descriptive registry must not be oversold

[A] `TestConfigResolvedIsRegisteredProtected` (`api/config_resolved_test.go:139`) reads source text to check registration group; it does not prove user ownership. Existing plan IDOR tests construct specific plan/risk routes with stub handlers; they do not cover the desk/accounts/trader-delete routes above. Registry completeness tests verify reachable schema classifications and a nonempty consumer citation, not that every consumer changes production behavior. The registry itself documents leaf-name collision fallback (`knob_registry.go:111`).

[B] Keep these useful tests, but do not call them complete end-to-end authorization or settings-effect verification. Separate test existence, execution, boundary coverage and live evidence.

## 4. Tests actually read, not executed

- `api/handler_plan_idor_test.go`: query/body plan ownership, risk force-flat ownership, no global trader fallback.
- `api/handler_plan_order_truth_test.go`: version/placement selection, three separate term sources, terminal/ambiguous/stale broker-order rejection.
- `api/config_resolved_test.go:125–165`: source registration auth guard (partial file read).
- `store/knob_registry_test.go`: reflection completeness/classification/boot counts.
- `web/src/components/strategy/RiskControlEditor.test.tsx`: daily control edits 450→400 and absence of new per-trade cap.

No pass/fail claim from this lane. Historical CI results in conversation are not a new test execution.

## 5. Explicit remaining scope

Read ledger currently lists **44 files: 21 full/manual, 23 excerpts/manual**. These counts describe inspected text, not complete function comprehension or a percentage of the repository. Truncated outputs were conservatively marked excerpt. Files only found by `rg` were not promoted to read coverage. Root's machine symbol inventory should supply function totals separately; I do not fabricate function counts from partial reading.

Unreviewed or only indexed areas include: auth implementation; most user/login/register handler logic; exchange/model secret encryption and bulk save/reload handlers; global risk feeds and stream ticket implementation; the majority of 2,395-line plan handler (ask/apply/realign/reset/approval/owner overlay mutations); most trader creation/update/pause/resume; SSE lifecycle and cancellation; web charts/account selector/position history; full SettingsPage; most PlanCard/SessionPlanCard; most Guide content and translations; most strategy clamps/defaults/registry table consumers; migrations/DB driver encryption; all individual persistence tables except selected trader/strategy/geometry paths; race/concurrency/lifecycle effects during reload; grids/crypto/brokers; runtime data and deployment evidence. The execution lane covers broker/order gates; root covers main/agent/deploy. None of that work is claimed here.

Next scoped reviews should prioritize F1–F5 with independent real-call-site fixtures, then finish API route/ownership inventory, save/activation rollback contract, account binding consumer chain, plan mutation lifecycle, and browser unknown/stale/cross-account state behavior. Findings must remain tied to this base until rechecked against a newer commit.
