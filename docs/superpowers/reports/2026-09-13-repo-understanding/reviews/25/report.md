# Assignment 25 — web shell, charts, chat and shared client contracts

Source review complete for **54/54 assigned files, 12,309 lines, 0 unread assigned files** at **63968be62e44db2fb07a92883e02127b9064b0be** in `/tmp/nofx-understanding-surfaces-20260913`. `functions.json` records **205 named function/method or named factory-callback entries**, with exact start/end lines, and **162 anonymous callbacks grouped under their containing function**. This is a TypeScript AST census checked against full manual reads, not 205 exported APIs. Configuration literals/type-only files are recorded at file level. The 4,152-line translation catalog was read completely in eight successive bounded ranges. Dependency reads remain explicitly separate from assigned coverage.

**Evidence:** [A] means exact source/graph/test text read or source syntax parsed; [B] means behavior inferred from those branches; no browser, test suite, application, live endpoint, database, order or wallet operation was executed. These are static findings, not reproduced incidents. Node was used only to parse source with an already-installed TypeScript parser; Python only generated review artifacts. No source/live changes or child agents. Existing E2E is deliberately not run because its Vite proxy targets the deployed `:8080` backend. No service reindex.

Standing canon and AUDIT-CHECKLIST were read in this worker's earlier assignment context (canon full; checklist 1–115 and 2281–2348), with current SYSTEM-MAP 301–320 and VL-TRADING-RULEBOOK-v1 180–200. The relevant latest spec log was `565e8fbe Sun Sep 13 00:39:02 2026 -0500 fix: use owner daily-loss controls without requiring a per-trade cap`. This review preserves the owner's daily-loss policy and does not revive a mandatory per-trade cap. Listed `web/AGENTS.md` is absent in this baseline. Findings are independent of the concurrently repaired branch; no claim that baseline findings remain unfixed there.

## Actual end-to-end boundaries

- **Current market/equity display:** `TraderDashboardPage.tsx:827` renders `ChartTabs` with `marketOnly`, resolved exchange type and selected symbol; it separately renders `EquityChart`. `ChartTabs.tsx:142–528` owns market/symbol/interval state and renders `AdvancedChart`, which requests `/api/klines` through authenticated `httpClient`. SVP is fetched by `AdvancedChart.tsx:1262–1294` from protected `/api/klines/svp`, attached once and updated via `SessionVolumeProfile.setData:467`. The primitive consumes server-computed bins/POC/VAH/VAL; it does not calculate strategy levels or place orders. `ChartWithOrders` and `TradingViewChart` have **no current production import/caller found**, so their external/legacy feeds are not the current NT8 futures source.
- **Account and command ownership:** `traders.ts:20–183` exposes my-traders, CRUD, lifecycle, close-position and NT account selection endpoints. It passes optional account for applicable commands, while server routes own authorization/trade refusal. Structured `ApiError` retains error key/params/status for selected calls, others throw generic errors. `EquityChart` queries current account first and uses account-scoped SWR history/account keys; APIs include the account query and backend `handleEquityHistory:128–191` reads scoped snapshots. Browser controls are not backend safety gates.
- **Chat:** `AgentChatPage` uses the assigned `useAgentChatStore`, persistence helpers and five assigned presentation components. Welcome suggestions invoke the send callback, including a one-contract MNQ long instruction; execution authorization remains backend-owned. Current stream functions mutate a global store and persist a snapshot to the request's captured user ID. Steps hide tool/central-brain bookkeeping; bot text is delegated to MessageRenderer (its implementation is outside this assigned slice). User text is rendered as escaped React text.
- **Auth/setup:** Router still mounts RegisterPage, SetupPage, AgentChatPage and FAQPage; welcome renders BeginnerOnboardingPage over the traders page. AuthContext handles token storage, login/register navigation and logout. RegisterPage's config flag is presentation gating; server registration endpoint owns first-user enforcement. Setup selects beginner/advanced, with beginner producing a wallet preparation request. That request is a protected POST which may create/adopt model credentials and writes process/.env wallet settings, including on the UI's balance-refresh action. There is a separate GET current-wallet endpoint, unused by this page's refresh.
- **Credentials:** Current model configuration APIs fetch the server encryption flag, optionally fetch/import its RSA key and encrypt the request. `CryptoService.encryptSensitiveData:87–155` generates fresh AES-256-GCM key/12-byte IV, 128-bit tag and AAD with user/session/time/purpose, then wraps the key with RSA-OAEP/SHA-256 and base64url encodes. The two-stage key modal itself only concatenates/validates and calls onComplete; its label is not evidence of encryption. Backend `/crypto/decrypt` is JWT-protected at server.go:175; the unauthenticated frontend helper has no current caller, so this is a dormant broken helper, **not a currently public decryption oracle**.
- **Presentation/config:** Header/Container/LanguageSwitcher/FAQSidebar and related wrappers are render/navigation helpers. Translation keys, TS interfaces, test/build configs and branding literals do not prove backend features or defaults. Explicit source file notes below cover all assigned declarations, including dormant helpers and stores.

## Findings and limitations, prioritized by current reachability

### 25-F1 — active market selector can send a symbol to the previous exchange [A source, B effect; medium]

`ChartTabs.tsx:181–185` calculates currentExchange as hyperliquid for that market, otherwise `exchangeId || marketConfig.exchange`. `handleMarketTypeChange:232–236` changes market/symbol but not the passed exchangeId. The dashboard supplies `ninjatrader` for an NT trader. Select Crypto: label/symbol become crypto/BTCUSDT while AdvancedChart still receives ninjatrader; Stocks/Forex/Metals similarly retain the original exchange. AdvancedChart:218 interpolates that exchange into the actual klines request. This predicts a misleading/empty market display, not an order-routing bug. The reverse case also applies when selecting NinjaTrader under another passed exchange. `ChartTabs.test.tsx` does exercise the same mobile market handler but its AdvancedChart mock destructures **only symbol**, so its assertion passes while the exchange remains wrong. Add exchange to the production-call-site assertion in a future fix.

### 25-F2 — active FAQ includes destructive/stale operating guidance [A source, B consequence; high-priority content correction]

`translations.ts:827–828` (English), corresponding Chinese text at2213–2214, calls SQLite `data.db-wal` and `data.db-shm` lock files and tells the reader to delete them. `faqData.ts:226–227` maps that entry, FAQContent's default branch renders the answer, and AppRoutes:499 mounts FAQ. WAL may contain committed state absent from the main DB; this is not a safe generic lock-recovery instruction. The text also proposes stopping processes without this deployment's lock/flat-window rules. No command was run.

The same active FAQ says stops/targets are merely AI guidance (`faqStopLossTakeProfitAnswer`, English804–805, mapped faqData:192–193), contradicting the current protective-stop futures architecture. It lists crypto-only exchange setup, old NoFxAiOS installation sources and an upstream individual's security contact. Treat it as inherited prose requiring alignment with the current runbook, not current architecture or supported deployment instructions. **Negative result:** stale `faqPRGuidelinesAnswer` points upstream, but `FAQContent` specially overrides that item with the correct project PR target; do not count that stale translation as a currently rendered wrong-PR instruction. Market/competition/data strings alone likewise do not prove removed pages exist.

### 25-F3 — initial onboarding failure leaves a blank wallet panel [A source, B visible behavior; medium]

`BeginnerOnboardingPage.loadOnboarding:24–50` stores the failure and clears loading. Render chooses `loading ? ... : data ? ... : null`; its error box is inside the data branch. On first failure, data is null, so the body hides the stored error and provides no retry. Heading still states wallet ready. On a later refresh failure, existing data makes the error visible, so those cases differ. The skip button marks completed even while preparation is pending. Current backend handler `api/handler_onboarding.go:43–96` proves this mount/refresh request may mutate wallet/model/env state; it is not a read-only balance request. Static page flow only, no wallet generated or queried.

### 25-F4 — hiding/reopening the live two-stage modal retains private-key state [A source, B behavior; medium]

`TwoStageKeyModal.tsx:42–347` owns part1/part2/stage and returns null on isOpen=false; neither close nor successful completion calls reset. Its caller `ExchangeConfigModal.tsx:1525–1532` keeps the same component mounted and toggles isOpen using secureInputTarget. Completion:350–359 copies the key then hides the modal. Reopening therefore retains both parts and the completed second stage, including when targeting a different exchange within the same parent instance. The two-second stage timer also has no cleanup. This is a secret-lifetime/cross-target UX issue, not demonstrated exfiltration. Clipboard diversion cannot erase clipboard history or JS memory. Final callback returns plaintext to the parent; transport encryption happens downstream.

### 25-F5 — cross-user frontend state lacks a complete ownership boundary [A source, B scenario; medium, shared-browser/account-change condition]

`TraderStatusPanel.tsx:11–16` uses one constant SWR key, `agent-sidebar-traders`. AuthContext logout/unauthorized handlers clear auth state/storage but do not clear SWR cache; no SWRConfig/cache-clear boundary was found in web source. A different account logging in within the same SPA can initially see the prior cached trader list before revalidation. Single-user operation reduces ordinary incidence, but registration/reset/user changes are not accounted for in that key. No backend authorization bypass is claimed.

`agentChatStorage.ts:44–62,119–135` copies/falls back to guest/legacy histories for every empty user history and leaves source keys intact. Tests explicitly require this behavior; they do not test two successive users. More seriously, `AgentChatPage.runAgentStream:103–389` persists the **current** global store with captured storageUserId through helpers:81–101; user-reset effects:489–540 replace that store without aborting the existing stream. Pagehide abort exists; SPA unmount/auth switch does not invoke it. An old callback can therefore serialize the new user's current conversation under the old user's key even when botId no longer matches; this is a source-level cross-user persistence scenario, not reproduced leakage. Backend stream continuation and account-reset policy are outside this slice. A future fix needs generation/identity checks at mutation and persistence, plus explicit stream lifecycle ownership.

### 25-F6 — current equity presentation can invent baseline/cycles and mix snapshot times [A source, B display effect; medium]

`EquityChart.tsx:133–199` drops every equity<=1 observation, falls back through truthy baselines to1000, and supplies `index+1` when cycle_number is absent. Backend history returns available_balance, total_pnl, total_pnl_pct and no cycle_number/pnl. The normal scoped NT baseline uses real snap.Balance and is an improvement over global initial_balance; do not label all selected-account curves fabricated. But missing/zero baseline data yields a fabricated number, and every ordinary returned row gets an invented cycle label. Header account equity polls15s while PNL and footer current equity use history polling30s, so they can represent different instants without an as-of label. Chart timestamps use browser-local `toLocaleTimeString` without timeZone/date at:187 despite the CT policy. `account` query errors are ignored, and absent account top value is displayed0.00 while historical values can remain populated. No financial measurement was recomputed from real data.

### 25-F7 — recoverable config failure sticks; polling backoff contracts can be defeated [A source, B effect; medium/low]

`useSystemConfig` listens for invalidation and uses mounted guards, but successful refetch does not clear an earlier error. Dependency `lib/config.ts:11–24` retains a rejected promise indefinitely unless explicitly invalidated and does not check HTTP status before caching JSON. A transient initial failure can remain cached across normal consumers. The hook's manual refresh is the recovery route.

`useAutoRefresh:118–174` owns per-effect inFlight/visibility/backoff and clears timer/listener, but does not cancel an active request. Consumers must guard identity/late results. `MarketTicker.fetchTickers:26–46` catches errors and returns normally, so the shared hook cannot increment backoff despite its caller comment. Empty/non-success payload can erase tickers without a stale/error label. No measured polling load claim.

### 25-F8 — global confirmation is one replaceable resolver, not a queue [A source, B scenario; low/medium]

`ConfirmDialogProvider.confirm:65–76` overwrites state.resolve. Two overlapping confirm requests leave the first promise unresolved. handleClose resolves inside a React updater; global registration has no unmount cleanup. Common usage may serialize dialogs; no actual concurrent caller incident is asserted. Historical graph explicitly says it queues requests, which the source disproves. Future verification should call the production global confirm twice, not compare duplicate mock promises.

### 25-F9 — crypto key cache can pair a new PEM with an old key after import failure [A source, B scenario; low/medium]

`CryptoService.initialize:33–39` assigns publicKeyPEM before awaited import. If an existing key is present and a replacement import fails, PEM becomes new while key remains old. Retrying the same PEM early-returns and uses the old key. Concurrent key imports likewise lack a generation lock. The normal stable-key happy path is sound by source; no broken ciphertext was produced. Downstream APIs do await initialize; they do not silently fall back to plaintext when a required crypto config fetch fails.

### Other bounded findings / dormant risks

- `t:4129–4152` has no English fallback and replaces only the first instance of each placeholder. Indonesian misses newer default-lock/grid-futures messages, so the UI can display raw keys. Translation claims “Never uploaded” for model wallet keys contradicts frontend-to-server configuration submission; local signing means server-local here, not browser-only. Actual reachability was located in ModelConfigModal:850; no credential inspected.
- `AgentStepPanel:59` trusts persisted step status/label, while storage validates arrays only; corrupted/older message schemas can crash rendering. No untrusted remote exploit established.
- `RegisterPage` classifies any message containing “limit” as whitelist capacity, lowercases beta code while saying case-sensitive, and renders AES-256 as fixed text. AuthContext register catches failures, so SetupPage's lack of a local catch is not independently a normal network-crash finding. Server first-user enforcement was not re-audited here.
- **Dormant:** config/modal Zustand stores have no production consumers beyond their export barrel; config-store stale-user load, unauth branch retaining private caches and e.id-vs-exchange_type heuristics are revival hazards. Likewise unused counter hook ignores target0, GitHub stats hook has request races, clipboard fallback ignores false execCommand, and stripLeadingIcons uses malformed astral escapes. They are not current observed UI failures. Color helper is used by ComparisonChart; index-derived color is only stable while list order stays fixed.
- **Dormant charts:** ChartWithOrders omits exchange from load dependencies, retains markers across chart recreation, overlaps fetches, aligns order times to epoch buckets and does not support week suffix, with a browser-local tooltip despite CT axes. TradingViewChart embeds external script and defaults crypto. Neither is the active NT8 chart.
- **Active SVP limitations:** primitive global autoscale ignores visible-range args and partial/frozen/inVA are not signaled. Anchor search limited to±30minutes may omit coarser sessions;40px minimum can exceed intersession space. These are visualization hypotheses, not flaws in kernel SVP measurement. The caller's syncSVP can reattach after toggle-off if an earlier fetch returns late because it rechecks series existence but not current enabled state (dependency issue, overlaps chart owner's slice).
- Numeric intraday CT formatter is DST-aware and existing tests establish intended seasonal labels by source. BusinessDay/string conversion turns a calendar date into UTC midnight and can display prior CT date, but current reviewed candle loaders supply numeric timestamps, so this remains a dormant input-contract edge.

## Verification quality and historical graph

Read five related test files, **none executed**. ChartTabs tests cover composition and symbol changes but omit exchange. Storage tests assert guest migration and snapshot semantics. Chart-time tests cover numeric DST/CT only; purported host-zone test does not change TZ. Trader slug tests prove immutable-id round trips and unique legacy bookmarks; no duplicate legacy-name test. RegisterPage tests import no production component and recreate regex/validation objects locally, including an asserted specialCharsRegex prop the current component does not pass. They prove their own test logic, not production PasswordChecklist parity. No SVP primitive/EquityChart/modal-lifetime test was located by targeted filename/content search; absence is scoped to that search.

Historical Understand Anything snapshot (July10@7a8adce0): **90 exact-path nodes,284 incident edges,46/54 assigned paths** were extracted/read; source wins. Eight assigned paths are absent. Current graph artifact records172 syntactic import/re-export and manually traced consumer edges, clearly distinguishes file imports from resolved function calls, and preserves historical edges. Corrections: confirm does not queue; MarketTicker now MNQ rather than spot strip; main Vite binds127.0.0.1 not historical0.0.0.0; ChartTabs now marketOnly/separate equity; config/modal stores need reachability caveat; API barrel includes plan API; chartTime and SVP are new boundaries. Historical test links helped locate the self-contained RegisterPage tests. CGC export belongs to root; no current semantic index claimed.

Potential repairs require current-branch comparison and targeted offline production-call-site tests. This source review does not assert existing tests pass, deployed rev equivalence, browser rendering measurements, current account values, correctness of all external libraries, or full repository coverage.

## Assigned file ledger

Every file below was fully read; exact hashes/ranges are in reads.json. Exact function boundaries and individual purposes are in functions.json.

- `web/e2e/fixtures.ts` (35 lines): Playwright page fixture optionally attaches CDP, creates isolated context. Cleanup not in finally if use rejects; source only, E2E not executed.

- `web/e2e/playwright.config.ts` (64 lines): Serial desktop/mobile E2E launches localhost3000 dev proxy to deployed8080. Specs may mutate/restore settings; not offline-safe authorization. E2E auth env optional. No execution.

- `web/eslint.config.js` (89 lines): ESLint TS/React rules relax explicit-any/unused/exhaustive-deps and newer hooks checks; lint success cannot prove effect dependencies or types. Config JS ignored.

- `web/src/chips-harness.tsx` (166 lines): Dev-only harness renders real chips and reread controls; patches gate only; not production entry. RereadButton action may still call real endpoint if clicked; no harness executed.

- `web/src/components/agent/AgentStepPanel.tsx` (109 lines): Step panel hides tool/central_brain steps; trusts status and label; unknown persisted status dereferences undefined style.

- `web/src/components/agent/ChatMessages.tsx` (157 lines): ChatMessages renders user text escaped and bot through MessageRenderer; meaningful execution excludes planning/tool internals; forwardRef and inline callbacks reviewed.

- `web/src/components/agent/MarketTicker.tsx` (208 lines): MNQ ticker polls authenticated endpoint; fetch catches errors so shared backoff cannot observe rejection; absent data becomes empty with no error/stale label.

- `web/src/components/agent/TraderStatusPanel.tsx` (119 lines): TraderStatusPanel SWR uses constant agent-sidebar-traders key rather than user id; auth provider cache cleanup must be traced for cross-user stale private data.

- `web/src/components/agent/WelcomeScreen.tsx` (191 lines): WelcomeScreen suggestions directly invoke onSend including one-contract MNQ long request; backend permissions own trade gating.

- `web/src/components/auth/LoginRequiredOverlay.tsx` (159 lines): Login overlay is presentation only; backdrop close and navigation links; lacks explicit dialog/focus keyboard management.

- `web/src/components/auth/OnboardingModeSelector.tsx` (75 lines): Onboarding mode selector defaults described as Base wallet/Claw402+GLM; current page says DeepSeek; no model authority here.

- `web/src/components/auth/RegisterPage.tsx` (365 lines): Registration config starts enabled and catches fetch failure, initialized false is gate; server must enforce single-user. Broad substring limit misclassifies rate-limit as whitelist; lowercases beta despite case-sensitive label. AES256 static footer is not encryption evidence.

- `web/src/components/charts/ChartTabs.tsx` (528 lines): ChartTabs wires active EquityChart/AdvancedChart; NT14 intervals; manual market selection except hyperliquid still uses passed exchangeId, causing stocks/forex UI to request original NT exchange. Missing exchange defaults hyperliquid; fetch symbols no auth/status/abort; symbol reset effect protects default.

- `web/src/components/charts/ChartWithOrders.tsx` (669 lines): Legacy ChartWithOrders comment says unmounted; source race and stale-marker issues: load effect omits exchange/height, no cancellation/inflight, created markers retained across recreated chart; order times floor assumes epoch bars (wrong session anchors), unsupported w falls60; tooltip still browser-local despite CT axes.

- `web/src/components/charts/EquityChart.tsx` (521 lines): Active equity/account SWR keys account-scoped after accounts fetch; no user id. Filters equity<=1, fabricates1000 baseline if absent/zero, current top account total may differ last-history PNL; browser-local time with no date; account error ignored. NT baseline uses first available_balance, verify backend meaning.

- `web/src/components/charts/TradingViewChart.tsx` (421 lines): Legacy TradingView crypto external widget; CT timezone explicit, language id fallsEN; cleanup clears DOM but external load no cancellation/error; verify unmounted before classifying.

- `web/src/components/charts/primitives/SessionVolumeProfile.ts` (583 lines): SVP primitive renders backend sessions/bins with per-session relative histogram scaling, latest price labels, global autoscale union; +/-30m anchor snapping can omit coarse intervals, min40width may exceed gap. partial/frozen/inVA ignored visually; no trading decisions; data validity delegated backend.

- `web/src/components/common/ConfirmDialog.tsx` (123 lines): ConfirmDialog single resolver state replaced on concurrent confirm: earlier promise stays pending; global registration no cleanup, resolve invoked from updater; dialog callbacks reviewed.

- `web/src/components/common/Container.tsx` (40 lines): Container layout-only polymorphic wrapper; defaults1920px and responsive padding.

- `web/src/components/common/Header.tsx` (76 lines): Header translations and three language setters; simple hides subtitle.

- `web/src/components/common/LanguageSwitcher.tsx` (33 lines): LanguageSwitcher maps supported language choices to context setter.

- `web/src/components/common/WhitelistFullPage.tsx` (132 lines): WhitelistFullPage navigates login or callback; blank official links still rendered as external links; generic capacity copy may not reflect actual denial.

- `web/src/components/faq/FAQSidebar.tsx` (60 lines): FAQSidebar grouped translated question navigation; active id assumed globally unique.

- `web/src/components/modals/SetupPage.tsx` (226 lines): Setup deletes browser auth/onboarding keys on mount and defaults beginner; submit minimal8 length delegates AuthContext; failure handling assumes register resolves; context state and storage may diverge.

- `web/src/components/modals/TwoStageKeyModal.tsx` (347 lines): Two-stage key input concatenates58+6 hex with optional0x; clipboard overwrite cosmetic not memory security; isOpen false does not clear parts/stage, timer has no cleanup, reopening retains secrets if component remains mounted; completion only callback (label encrypt is not actual crypto).

- `web/src/constants/branding.ts` (24 lines): Branding raw filesystem strings and static informational version; official links intentionally blank.

- `web/src/hooks/useCounterAnimation.ts` (51 lines): Counter animation end0 early-returns leaving previous displayed count; duration<=0/nonfinite not validated. RAF cleanup on deps change; fractional negative floor rounds down.

- `web/src/hooks/useGitHubStats.ts` (91 lines): GitHub repo/contributor queries, external only if used; no abort/mounted/request identity on owner/repo change, old responses can overwrite. Missing contributors rendered0, error preserves old data; days use abs ceil.

- `web/src/hooks/useSystemConfig.ts` (42 lines): System config hook ignores unmounted response and reacts invalidation; success never clears previous error, so recovered config can retain error UI. Shared cache implementation dependency not yet read.

- `web/src/i18n/translations.ts` (4152 lines): Full4152line three-locale catalog and t lookup reviewed. No ENfallback on missing nested IDkey; interpolation replaces first occurrence only. Current FAQ contains stale upstream install/PR/security contacts, deletion of SQLite WAL/SHM, guidance-only stops, crypto-only architecture and false never-uploaded key wording; reachability traced separately. Strings are not operational instructions or verified financial/model claims.

- `web/src/lib/agentChatStorage.ts` (142 lines): User-scoped chat keys but fallback/migration copies shared guest/legacy messages into any empty user history without removing origin; possible shared-browser cross-user leakage. JSON arrays unvalidated. Writers/removers can throw storage errors; streaming snapshots normalized false.

- `web/src/lib/api/index.ts` (15 lines): API object spreads six modules; later same-named properties override earlier. Thin export, not enforcement.

- `web/src/lib/api/traders.ts` (183 lines): Trader API wrappers target my-traders/config/actions/account endpoints; typed ApiError preserved only selected mutations. Nonarray getTraders becomes empty; request IDs interpolated without encoding; server authorization required. No API calls executed.

- `web/src/lib/autoRefresh.ts` (174 lines): Polling helpers pause hidden, prevent per-effect overlap and exponential backoff8x; no abort of in-flight request on effect cleanup or trader change, consumers must prevent stale application. Errors caught inside consumer cannot signal backoff. PollResume resets after30s.

- `web/src/lib/chartTime.ts` (78 lines): DST-safe CT intraday labels, but BusinessDay/date-only parsed UTC midnight then shifted to prior CT day/month/year. Needs consumers timeframe review; no runtime plot asserted.

- `web/src/lib/clipboard.ts` (30 lines): Clipboard fallback ignores execCommand false and announces success; rejected async Clipboard API does not attempt fallback. Temporary textarea cleanup not finally.

- `web/src/lib/crypto.ts` (258 lines): Hybrid RSA-OAEP SHA256 AES-GCM256 random12byte IV and authenticated user/session/time metadata. initialize sets cached PEM before await import, so failed replacement can leave old key with new PEM and future retry early-returns. No concurrent init coordination. decrypt fetch has no auth header locally; server route access must be traced. Config/encryption flag defaultfalse until fetch; callers own enforcement.

- `web/src/lib/httpClient.ts` (321 lines): Axios attaches token globally;401 clears auth and redirects with intentionally forever-pending promise, static latch reset externally.403/404 throw generic,5xx server-message returned as business failure; contract comments overgeneralize. Timeout overrides truthy only; no response shape validation.

- `web/src/lib/onboarding.ts` (37 lines): Global localStorage user mode/wallet/completion state not account scoped; no storage exception guard. Route maps beginner welcome else traders; no authorization.

- `web/src/lib/text.ts` (28 lines): Decorative prefix regex uses four-digit Unicode escape syntax for supplementary ranges (\u1F000 etc); potential unintended ASCII range stripping. Need isolated pure proof/tests, not executed.

- `web/src/pages/BeginnerOnboardingPage.tsx` (295 lines): Onboarding invokes prepare mutation on mount and balance refresh, displays returned private key; initial load failure stores error but error rendered only inside data branch -> blank body with no retry; skip marks completed even while load pending; async no user generation/cancel.

- `web/src/pages/FAQPage.tsx` (21 lines): FAQPage delegates context language to FAQLayout.

- `web/src/router/traderSlug.ts` (37 lines): Primary immutable full trader ID exact match correct; legacy name/prefix fallback intentionally ambiguous first match. No authorization itself.

- `web/src/stores/agentChatStore.ts` (42 lines): Global Zustand chat state supports resetForUser but async updater has no user generation check; stream consumers must scope callbacks. Not persisted by store itself.

- `web/src/stores/tradersConfigStore.ts` (108 lines): Global config store derives configured status by credentials/enabled, compares special venues by e.id rather than exchange_type. Unauthenticated load does not clear prior private arrays; pending authenticated loads can repopulate after reset. Errors logged/swallowed, consumer cannot use rejection for backoff.

- `web/src/stores/tradersModalStore.ts` (75 lines): Global Zustand modal/editor selections, reset explicit. Close clears selected model/exchange; user change cleanup caller responsibility.

- `web/src/types/agent.ts` (15 lines): Type-only AgentStep status union lacks failed; persisted messages have no runtime schema validation.

- `web/src/types/config.ts` (183 lines): Config contracts include UUID exchange id distinct from exchange_type, legacy NT CSV settings, and onboarding secret response; no runtime enforcement.

- `web/src/types/index.ts` (3 lines): Type barrel only.

- `web/src/types/strategy.ts` (294 lines): Strategy/grid/risk/day-plan type contracts; defaults and acceptance comments can drift from Go; no runtime enforcement.

- `web/src/utils/traderColors.ts` (31 lines): getTraderColor derives palette from current list index, so reorder changes identity color; missing ID uses first.

- `web/vite.config.ts` (41 lines): Vite loopback3000 proxy8080; no-store modules, product branding HTML replacement. No strictPort, can autochoose another port. Product name is local trusted text.

- `web/vite.sandbox.config.ts` (15 lines): Sandbox loopback3001 strictPort proxy8081, lacks main visible-product-name HTML transform. Separate config not identical plugin behavior.

- `web/vitest.config.ts` (27 lines): Vitest jsdom setup and explicit branding filesystem allowance, excludes E2E suites. Source-only config read; no test execution.
