# Assignment 24 — plan UI and guide source review

Baseline `63968be62e44db2fb07a92883e02127b9064b0be`, isolated `/tmp/nofx-understanding-execution-20260913`. **61/61 assigned files, 12,988 lines fully read; zero unread.** Named-function census: 184 functions/methods, plus 17 declaration/content-only file records and 243 anonymous callbacks covered within their containing functions/file. Fifteen additional dependency/test files were read fully or in explicitly recorded excerpts. This is a slice review, not whole-repository coverage.

Evidence **[A]** means exact source/test inspection; **[B]** identifies inferred runtime consequences. No browser reproduction, service calls, trade execution, live data inspection, source edits, or tests were run. A test described below was read, not demonstrated passing. Historical comments and guide performance numbers were not independently revalidated.

Rules followed: shared review instructions, main instruction file and baseline canon/checklist/SYSTEM-MAP/RULEBOOK from the preceding assignment. Spec freshness: `AUDIT-CHECKLIST.md`: `dfda15e1 2026-09-13T00:48:28-05:00 test: isolate session clock fixtures from weekly backfill workers`; SYSTEM-MAP/RULEBOOK: `565e8fbe 2026-09-13T00:39:02-05:00 fix: use owner daily-loss controls without requiring a per-trade cap`. Fixed-baseline review intentionally does not rebase. Owner daily-loss-only policy takes precedence over historical per-trade-cap assumptions.

## Purpose and end-to-end connections

`PlanCard` (`web/src/components/plan/PlanCard.tsx:33-190`) owns session/version selection, trader-scoped actions and the dashboard composition. `usePlanToday` (`usePlan.ts:17-42`) keys SWR by trader, symbol, session and optional version; historical versions do not poll. `planApi.getPlanToday` (`web/src/lib/api/plan.ts:487`) calls the authenticated HTTP client. Backend today serialization (`api/handler_plan.go:440-485`) returns document, machine level facts/windows, approval/runability, provenance and execution read models. `SessionPlanCard` (`:107-980`) renders lifecycle/no-plan states, authored versus machine evidence, historical versions, chart, scenario/zone/rules views, and edit/ask/realign dialogs.

The owner edit door sends overlays or sticky owner levels, then asks the planner to re-examine changes. `EditSheet.save` (`:113`) and `.del` (`:182`) send index patches; `BulkAddSheet.save` (`:53`) sends individual sticky levels. `SessionPlanCard.runRealign` (`:132`) requests a proposal; `RealignPanel.apply` (`:107`) and `AskPlannerPanel`'s `PlannerReply.apply` (`:128`) call the QA apply endpoint. Backend `applyPlanOverlay` (`api/handler_plan.go:1016-1079`) serializes overlay read/fold/append, validates the resulting document and price armor, and operates on the latest active plan. Ask decline is its own recorded owner QA event (`handler_plan.go:1564-1627`). Browser dialogs do not themselves authorize or place trades.

`ScenarioList`, `OrderTerms`, `StructuralGeometry`, `ArmedUnderBlock`, `ScenarioEconomics`, `OneSetupChip` and `FadePermissionChip` distinguish authored ideas, frozen geometry, evaluated outcomes, actual accepted order terms and open-position provenance. Missing outcome fields generally render UNKNOWN/unevaluated rather than fictional confirmation. `ScenarioEconomics` is absolute-distance authored arithmetic, before costs and composition; it is not realized expectancy or a guarantee of order admission.

`DeskStrip`, `AlertCenter`, `GateBlocksPanel`, `ExpectancyPanel` and `InstrumentsDrawer` fetch independent read models at different cadences. Expectancy is global, not selected-trader-only; the panel distinguishes corrected P&L/exclusions, descriptive sample floors and counterfactual data. `PlanMiniChart` fetches current market candles independently even beside a historical plan. `sessionConfig` and the timeline are browser-side static CT visual aids; backend active/runnable session fields govern the card.

`GuidePage` (`web/src/guide/GuidePage.tsx:443`) renders typed static sections via `BlockView` (`:151`) and a partial text search index (`collectHits:60`). Its one-time health revision check indicates build drift, not that every section is behaviorally correct. The only mock mode lookup (`useResolvedPlanMode:22`) reads the first public trader's resolved strategy mode once; it is not selected-trader or per-session context and explicitly falls back to “example — not a reading.”

## Source-backed findings

### 24-1 — stale index edits can act on a different current level [A source, B consequence]

`EditSheet.tsx:123-139,182-191` sends `replace/remove /levels/${levelIndex}` without a `test` operation, plan ID, version, original value or stable level ID. `SessionPlanCard.tsx:121-125` retains the clicked fact/index in dialog state. Backend `handler_plan.go:1040-1053` fetches the latest active row, folds its overlays and applies the supplied patch. The mutex protects simultaneous backend writes, but cannot establish that a browser's old index still refers to the same level. If a replan or earlier remove/reorder occurs while the form is open, a still-valid index can overwrite/remove another level. Historical door gating (`SessionPlanCard.tsx:176-181`) is good but does not bind an already-open edit to its originating document. This is not a reproduced trade failure. Review priority: identity/concurrency guard at the mutation contract and the real UI callsite.

### 24-2 — bulk drafts reset on unrelated parent renders [A source, B consequence]

`BulkAddSheet.tsx:37-47` clears text and busy state in an effect depending on `[open,onClose]`. The actual parent supplies a new inline `onClose` on every render (`SessionPlanCard.tsx:951-960`). A plan poll or other parent state update while the sheet is open reruns this reset, losing staged input. Clearing busy while a batch is in flight can also reopen the submission guard. No browser timing reproduction was attempted. This is separate from the form's intentional reset when first opened.

### 24-3 — API HTTP failures masquerade as no plan / no alerts [A source, B consequence]

`httpClient.ts:212-258` converts HTTP error responses into `success:false`, while network errors throw. `plan.ts:getPlanToday:487` returns null on unsuccessful response; `usePlan.ts:28-39` therefore resolves normally and does not set SWR error. `PlanCard.tsx:174`/`SessionPlanCard.tsx:236,263-270` can present a no-plan/night state for an HTTP failure. `plan.ts:642-650` substitutes `{alerts:[],unacked:0}` on HTTP failure, and `usePlanAlerts:65-72` exposes no error channel. `AlertCenter.tsx:197,315-323` then loses its urgent banner and can show an empty feed. Claim is limited to returned HTTP failures, not all network failures. `DeskStrip.tsx:107-140` is a useful counterexample: it explicitly surfaces null/error as unreachable.

### 24-4 — dual bias label is serialized at a different JSON location [A]

`kernel/plan_doc.go:298-301` declares top-level `bias_label`; the today handler returns the document directly (`handler_plan.go:470`). Frontend `PlanBias` (`plan.ts:14-16`) models it inside bias, `SessionPlanCard.tsx:750` passes only `doc.bias`, and `BiasBlock.tsx:73-78` reads `bias.bias_label`. No alias is inserted by the inspected serializer. The main direction still renders, but the AI/tree/regime composite label cannot arrive through this typed path. Static producer/consumer mismatch; not browser-reproduced.

### 24-5 — realign uses calendar date during the wrapped ASIA tail [A, overlap already repaired]

`handler_plan.go:2110` derives today's CT calendar date, then `:2125` looks up the ASIA plan under it. `kernel/plan_chain_date.go:33-66` assigns the after-midnight ASIA tail to yesterday's session-instance date; today, overlay and thread consumers use this helper (`handler_plan.go:288,1040,1534`). Thus at 00:30 CT, a valid previous-date ASIA chain can yield `status:skipped, reason:no_active_plan` when owner edits trigger realignment. The helper's existing tests cover this exact midnight distinction, but not this handler callsite. **Root reports this was repaired in `a6b88b7d`; baseline finding retained solely for provenance, not a request to rediscover/fix again.** No local runtime reproduction.

### 24-6 — bulk range preview promises information the write omits [A]

`bulkParse.ts:22-52` parses upper price; `BulkAddSheet.tsx:182-184` previews the range, but `:63-66` posts only lower price, label and note. Save therefore loses `priceHi`. Its partial-success path (`:73-86`) closes the form whenever any row succeeds and summarizes the first six attempted rows, not the successful subset, so the realign summary can name rows that were never added. `EditSheet.tsx:153-177` similarly collects grade/instruction but only sends these to a later realign proposal, not the sticky-level write; accepting a proposal is a distinct action. UI should describe the persisted contract accurately. Parser tests validate parsing ranges, not their persistence.

### 24-7 — selection and mutation scope can diverge [A source, B operator confusion]

`PlanCard.tsx:62-65` sets selected session after the first response and never restores null. Despite its auto-follow comment, a user who never clicks a tab stays on that initial session after the active session advances. Header approve/re-read/reset controls (`:115-155`) receive only trader ID and remain outside historical/sibling-session door gating. They act on the active backend session, not necessarily the displayed card/version. The backend controls eligibility and these actions may intentionally be global; the issue is the lack of explicit action-target context beside a pinned card. Existing selected-session fetch behavior is correct: tab clicks actually change the request.

### 24-8 — stale read-model attribution and decline inconsistencies [A source, B consequences]

`GateBlocksPanel.tsx:51-67` preserves old rows/loaded state across trader changes or failed fetches until a successful new response. The component is reused under `PlanCard.tsx:97`, so old-trader counters can temporarily appear under the new selection. `ExpectancyPanel.tsx:101-104` and `PlanMiniChart.tsx:151-197` also retain prior data on failures without an explicit stale/error state; the former's as-of is rendered date-only (`:156`), hiding intraday age. These are display risks, not altered trading gates.

`RealignPanel.tsx:187-195` “Keep as-is” only dismisses locally, unlike Ask's persisted decline (`AskPlannerPanel.tsx:144-156`). Ask hydrates applied status only (`:110-112`); backend decline adds a separate owner event rather than changing the proposal's applied flag (`handler_plan.go:1604-1627`), so remounted proposal bubbles can offer action again. This does not imply an existing decline was applied. The realign budget counts planner responses (`handler_plan.go:2145-2152,2213-2229`), not Apply clicks; manual requests bypass that auto cap. Guide “Apply costs 1/decline free” prose is inaccurate.

## Guide contradictions and daily-loss-only policy

These are currently served static guide statements, even where they discuss old waves. They should not be treated as new production engine defects merely because old exploratory results appear in prose.

| Subject | Exact evidence and interpretation |
|---|---|
| Structural stop / daily loss | `content/settings.ts:415-422` correctly says no separate per-trade dollar cap, one MNQ, structural stop not overridden by ATR. `guards.ts:13` agrees. Older universal ATR claims at `guards.ts:94-111,243`, `faq.ts:83`, `status.ts:86` conflict with this exception. Root repaired the overlapping planner instruction in `710ea1c8`; do not restore a mandatory dollar cap or universal ATR override. |
| Scenario quality | `guards.ts:145-147` and `faq.ts:97` broadly say it does not gate; `settings.ts:258` acknowledges a configured floor. `trader/armed_executor.go:2135-2138` actually refuses below-floor quality. Default low floor can admit all normal grades, but changing the setting is consequential. |
| Fade permission | `FadePermissionChip.tsx:50`/`guards.ts:326` emphasize descriptive labels; `plays.ts:16` requires permission for one-setup. Distinguish the historic recorded label (not itself re-read as authority) from fresh `FadePermissionAt` feeding `OneSetupAllowsAt` (`trader/one_setup_wiring.go:147-150`, `kernel/one_setup.go:146`). Broad “nothing refuses” wording needs that distinction. |
| Wake gates | `settings.ts:400-410` says WARN-only/still runs; `status.ts:87` describes enforcement. Assignment21 directly traced current level-wake cutoff/cooldown enforcement and a separate MSS bypass; neither should be flattened to a global WARN-only rule. |
| Consecutive loss | `settings.ts:460,557-559` says enabled with master; `guards.ts:126,230` correctly describes standalone breaker. Assignment21 traced the production independence. This is not a reason to enable/tune any live control. |
| Proximity width | `settings.ts:62` says higher K is tighter while `:69` gives K × daily range, which grows with K. `levels.ts:155` also carries older 0.3 retune text versus `settings.ts:67` 1.5. No live value was read here. |
| Realign spend | `settings.ts:164` and `planCard.ts:194` couple budget to Apply/decline. Handler counts planner replies, before an owner applies anything. |
| Weekly missing document | `weeklyBias.ts:37,131` and `WeeklyChip.tsx:14` imply no effect. Assignment21 traced a Sunday-ASIA weekly-document prerequisite: distinguish missing reference context from the scheduler's required initial weekly read. |
| “Live/current” constants | `tradingDay.ts:101,108-110` calls a literal table YOUR live config; `settings.ts:608` lists literal current overrides. They are source claims, not fetched account settings. Build revision matching cannot make constants live. |
| Historical measurements | `plays.ts:50`, `routines.ts:51`, `levels.ts:25` carry performance/sample claims without row IDs or sufficient artifact lineage in these passages. Treat as unverified historical research prose, not an admission rule or reproduced result. Newer `expectancy.ts` correctly discusses sample floors, corrected P&L, counterfactual/contract limitations and missing evidence. |

The detailed file ledger records additional minor prose drift (reset version/budget phrasing, settings hot-reload description, static RR defaults and deleted thin-side terminology). None establishes an account's current configuration. No financial/trading recommendation is made.

## Boundaries, negative results and existing tests

* HTTP-success handling for re-read/reset was investigated and **not promoted to a defect**: inspected backend failures return non-2xx (`handler_plan.go:1183-1188,1231-1236`).
* Public `/api/traders` intentionally requires no bearer token (`api/server.go:114`); the guide mode hook's first-trader/context choice is its limitation, not missing authentication on that public fetch. Protected config remains separately authenticated.
* `RulesBlock.tsx:58-65` conflates empty computed band and absent band, but current `RenderNoTradeBand` (`kernel/no_trade_band.go:213-216`) returns nil for zero machine windows and documents legacy prose fallback. This is a contract/future-compatibility risk, not a proven current empty-array production failure.
* Optional null/undefined geometry and excursion formatting branches, a multi-column grid alignment concern, and sparse fabricated version chips were identified as contract/visual limitations but not reproduced or proven reachable from the current backend contract. No claim of crash or missing production version is made.
* Alert initial seeding (`AlertCenter.tsx:162-173`) can run against the hook's initial empty array before the first response, so later-arriving P0 backlog may toast as “new.” Ref sets also survive trader changes. This is an additional static notification concern.
* `NoTradeBand.test.tsx` exercises live/elapsed/other-session windows and legacy prose fallback. It does not exercise a computed empty band. `bulkParse.test.ts` validates parsed ranges, not save payload fidelity. `P5_door.test.tsx` verifies `/levels/2` patch and prose/owner-note preservation, not a changed plan between opening and saving. `W13_integration.test.tsx` tests add → proposal from mocked backend shape, not midnight identity or parent-poll form reset. `W16_decline.test.tsx` verifies decline endpoint and in-place distinct outcome, not hydration after remount or RealignPanel's local dismissal. `kernel/plan_chain_date_test.go` covers 00:30 ASIA → prior date at the helper boundary. No tests executed.

The historical Understand Anything graph (`2026-07-10@7a8adce0`) has **zero nodes for these assigned paths**. `graph.json` contains that explicit absence and 187 actual current relative-import edges, labeled potentially type-only. It does not infer runtime calls from imports. Root's frontend symbol census is syntax-only; current source is authoritative. No CGC reindex/service change was attempted.

## Complete file-role ledger

Every assigned file follows, including pure content/type files. Function-level boundaries and lexical calls are in `functions.json`; full read ranges/hashes and test/dependency excerpts are in `reads.json`.

* `web/src/components/plan/AlertCenter.tsx`: P0banner/newtoast,P1feed,P2count, ack/dismiss/clear APIs. Firsteffectseedsfrominitialemptyhook=>backlogtoastedonceactualfetch. Failedfetchclearsbanner viaemptyfacade.

* `web/src/components/plan/ApproveButton.tsx`: Trader-only approval POST guarded; requires response.approved before success.

* `web/src/components/plan/ArmedUnderBlock.tsx`: Only server version_differs renderspositionprovenance; absentarmedversionexplicitunknown.

* `web/src/components/plan/AskPlannerPanel.tsx`: Portaled Q&A, JSON patch preview, persisted apply/decline and local tri-state; refresh effect lacks traderId, caller omitsplanId. No persisteddeclined fieldhydration, remountoffersoldproposal. Patches parsed withoutarrayvalidation.

* `web/src/components/plan/BiasBlock.tsx`: Renders direction conviction flip and bias.bias_label; verify top-level backend bias_label mismatch.

* `web/src/components/plan/BulkAddSheet.tsx`: Sequential per-rowadd partialsuccesscloses, summary includes first6 attempted notsuccessful; rangehighpreview never persisted. effect on unstableonClose resetsdraftonparentpoll.

* `web/src/components/plan/DeskStrip.tsx`: Serverrenderedrows andunknown/stalecount, dynamic5/15spoll, null/errorunreachable; age_ms frozenbetweenfetch,noindependentclock.

* `web/src/components/plan/EditSheet.tsx`: Index-based replace/remove of current levels; add ignores displayed grade/instruction (only sent to subsequent realign), editAI ignores note/tag; parseFloat positive allows partialnumeric. Form closes whilebusy possible; no versionbinding.

* `web/src/components/plan/ExpectancyPanel.tsx`: Globalconditionexpectancy60spoll, correctedPnl/exclusions/ids/NULLstats/descriptivefloor, E8counterfactualseparate; stale retainedonsilentfetchfailure, asofdateonly, cryptoexcludedfieldnotrendered.

* `web/src/components/plan/FadePermissionChip.tsx`: Recorded label plus exclusions/details/unknown; absentnotpermission; tooltip saysnogate verifyonesetupdependency.

* `web/src/components/plan/GateBlocksPanel.tsx`: Pertrader+global counters20spoll; keepsoldrows/loadedacrosstraderchangeorerror, nofetchfreshnessstate.

* `web/src/components/plan/InstrumentsDrawer.tsx`: Descriptiveadherenceoncepertrader,60sexpectancyexcursions,retiredlevelgate; failurezeroedfacaderendersno-trades; excursionnusesallconditionnnotmeasuredn.

* `web/src/components/plan/LevelOverlayPrimitive.ts`: Chart lifecycle/accessors/views/canvaslevelbands and lines; gradealpha onlyA notA+; autoscale allanchorprices notzonebounds; axes first6 gradeA regardlessstructuralcomment.

* `web/src/components/plan/LevelZoneMap.tsx`: Frozenzonemap preservesnullwidth/broad/shortlist/formation/sourcecontext; noauthoredarrayreorder.

* `web/src/components/plan/OneSetupChip.tsx`: Recordedallowed/waiting/declined/off/unevaluated distinct; descriptiveonly noexecution.

* `web/src/components/plan/OrderTerms.tsx`: Selected legintended/composed/accepted sources, rowplacement/version provenance; currentbookage frozeninpayload notclientaged.

* `web/src/components/plan/PlanCard.tsx`: Composes desk/alerts/tabs/statistics/actions/sessioncard; selected initialized from first active session then never null again, so auto-follow comment inaccurate. Approve/reset/reread outside historical door gating; active trader mutation while sibling/version displayed.

* `web/src/components/plan/PlanErrorBoundary.tsx`: Threadrendercatch resetretry, inputsoutside; Q&Aonlyboundary notgeneralplancard.

* `web/src/components/plan/PlanFooter.tsx`: Sharedversioncontrol plusdaytype/model/recordedreplans; no newbudgetformula.

* `web/src/components/plan/PlanLiveness.tsx`: Nullabletradeable=>UNKNOWN, zerooftotalpositive=>warningonly exhaustion; noaction.

* `web/src/components/plan/PlanMiniChart.tsx`: Chartpoll15s currentklinesevenhistoricalplan, sorted/dedupseconds, oldbarsretainedonsilentfailure; facts->overlaydropszonebounds/freshness.

* `web/src/components/plan/RealignPanel.tsx`: Proposal apply shares ask endpoint; Keep-as-is only localdismiss unlike Ask persisteddecline. Timed nochange/capped/failed dismiss. Manualbutton bypasscap percontract.

* `web/src/components/plan/RereadButton.tsx`: Poll backend eligibility30s, guarded confirmed force reread then success toast and delayed mutate, busy finally; gate retained across trader change until fetch.

* `web/src/components/plan/ResetButton.tsx`: Confirmed force reset restores chain; guarded/finally preserves responsiveness; success note warning. Control accepts trader only.

* `web/src/components/plan/RulesBlock.tsx`: Serverlive/elapsed/other windows; hasBand requiresnonempty, so computedempty[] re-promotesmodelnoTradeproseasrules.

* `web/src/components/plan/ScenarioEconomics.tsx`: Authored hypothetical/economics prices positivefiniteorUNKNOWN; abs-distance R beforecosts/composition, nofee orsidealignmentcheck.

* `web/src/components/plan/ScenarioList.tsx`: Recordedstate missing=>unevaluable, chips behindevaluablebranch; economics/orderterms alwaysvisible. ConfirmMETrequiresrecordedoutcome; authorqualityinformational.

* `web/src/components/plan/SessionPlanCard.tsx`: Lifecycle/no-plan rendering, historical versions/death history, chart/live facts, rules, scenarios, owner edit/ask/realign gates. Owner door uses is_active!==false, !historical,!no_trade; state not reset on trader/version. BiasBlock receives doc.bias only.

* `web/src/components/plan/SessionTabs.tsx`: Accessiblelabels/disabledserverrunnabletabs; arrowselectupdatesselectionbutnotDOMfocus.

* `web/src/components/plan/SessionTimelineStrip.tsx`: 30s clientclockmarker overstaticbands andkillzones, activefromserver; nottradingauthority.

* `web/src/components/plan/StructuralGeometry.tsx`: Server geometrysnapshot rendersreason/quantityMNQ/costrisk; undefined-safe formatter notnull-safe. Researchcandidatelabel.

* `web/src/components/plan/WeeklyChip.tsx`: Refs-only PWH/PWLthin/none, no directionalcall; none saysnothingchanges despiteweeklyreadprerequisiteonSunday.

* `web/src/components/plan/ZoneTable.tsx`: Factarrayindex forwarded to overlayeditor, conflictdimming/grades/touch/distance; 5trackgrid >5children; freshmissing=>consumed viahelper.

* `web/src/components/plan/bulkParse.ts`: Canonical single-token type parser, skipsbadlines, prefixparseFloat andrange upperweakvalidation; My level two-wordtoken unreachable.

* `web/src/components/plan/chips.tsx`: Grade/provenance/fresh/status/version/conflict/lifecyclewidgets. VersionChips still fabricates1..count despitecommentrealrecords; unknownlifecyclegoldactive; A+gradeCcolor.

* `web/src/components/plan/levelState.ts`: Freshness/sweep bucket, fixed12pointnearthreshold, signedroundedistance,3pointowner-vs-AI conflictingproseheuristicforvisualonly.

* `web/src/components/plan/sessionConfig.ts`: StaticCT visualbands mirroredregistry, onlyNYdefaultenabled, approximateDSTwarning; inWindowequalendsfull-day whereas toSegmentsempty.

* `web/src/components/plan/usePlan.ts`: SWR keys include trader/symbol/session/version, historical polling disabled; facade failures resolve values so SWR error absent. Versions lack symbol dimension. Alerts default zero with no error channel.

* `web/src/components/plan/vocab.ts`: CanonicalEnglish level/instruction/grade literals; parserusescasefold type list.

* `web/src/guide/GuidePage.tsx`: 15sectiontypedrenderer/searchpartialblockindex; healthfetchonceprefix12driftcheckunknownsilent. No persectionstampvalidation, allcontentEnglish.

* `web/src/guide/components/Example.tsx`: Dashedcaptionwrapperrealcomponentsmockprops.

* `web/src/guide/components/KnobCard.tsx`: Renders10mandatoryknobfieldsasstatictext; starrecommendationdisplay.

* `web/src/guide/components/MockPlanCard.tsx`: Realchipswithstaticmocklevels/scenarios; resolvesmodefirsttrader; confirmexamplelacksoutcome=>UNKNOWN whiledetailMET; notliveplan.

* `web/src/guide/components/useResolvedPlanMode.ts`: Readsfirsttrader thenresolvedconfig once; tradersrequest lacksbearerheaders, noselectedtrader/session, unknownlabelonfailure.

* `web/src/guide/content/buttons.ts`: Staticactionsideeffects; staleSavehotreloadprose, emergencyflat contradictory market-now/limitladder.

* `web/src/guide/content/candidates.ts`: Frozenfullzonemap/provenance/unknownwidth experimentalweights; legacyseatedcandidateordering/projectionsseparatelylabeled; noauthoredarrayreorder.

* `web/src/guide/content/expectancy.ts`: Researchrecordlateness/NULL/contractlimits; correctedPnl/sampleIDs/descriptivefloor/E8limitations; olddrawerreplacementpending andmanydisplaycolumnclaimsnotcurrentUI.

* `web/src/guide/content/faq.ts`: Staticanswerswithhistoricalincidents; broadclaims no repeatedreject/nofadeATRexception/noqualitygate stale; revisiondriftmeansdifference notnecessarilyolder.

* `web/src/guide/content/glossary.ts`: Staticterms; ThinSidecontradictsdeletedMinSide; bias/confirmadvisoryclaimsoversimplifyenforcement; rawmoneyguaranteesnotrevalidatedhere.

* `web/src/guide/content/guards.ts`: Structuraldaily-loss-onlypolicy explicit; olduniversalATRfloor/shadow/quality/fadepermissionclaimscontradictnewsections; standalonebreakeroutside mastermismatchsettings.

* `web/src/guide/content/levels.ts`: Levelkinds/gradeheuristics/HTFdetectorcoverage/roles; unvalidatedmultipliersdisclosed; stale0.3band vssettings1.5, weeklysourceshistoricnotalllive.

* `web/src/guide/content/planCard.ts`: Identity/economics/bucketcompletion/orderprovenance updated; stalequalityadvisoryvsfloor, resetv1/budget, realignapplycost; descriptivearithmeticcorrect.

* `web/src/guide/content/plays.ts`: OneSetupfadepermissionrequiredandstructuralgeometrycurrent; followcounterfactualpreregistration; broadplaycatalog/legacy75%returnclaim/ATRentrylaws coexist.

* `web/src/guide/content/routines.ts`: Staticdaily/weekly/emergencychecklists; staleSYSTEM_STATUSgreen and75%performancequestionnoIDs.

* `web/src/guide/content/settings.ts`: Staticknobcatalogwithsomecurrentstructuraldailyonlyownerpolicy; contradictoryRR3vs2, proximitydirection reversed, wakeWARNvsenforce, mastermustgatebreaker, Applychargesrealign, simulatedlivevalues notread. No mandatorypertradedollarcapinupdatedstructuralcard.

* `web/src/guide/content/status.ts`: Recentorderprovenance/pendingcancel/feedcontractchanges described; bootlineslegacyATR andoldtimeoutcoexist; rowsstaticnotlivedata.

* `web/src/guide/content/tradingDay.ts`: StaticCT schedule/wakes/lunch/EOD; claims tableYOURliveconfigbutliteral; defaultclocknotcurrentaccount.

* `web/src/guide/content/weeklyBias.ts`: Refs-only/shadowcalibration, candlecoverage/historysplice; missingweeklydocfailopenclaimcontradictsSundayASIAprerequisiteinreview21. Truncated combinedread repaired100-136.

* `web/src/guide/content/welcome.ts`: SIMarchitecture/roles/canonreferences; historicalexecutorcentricflow ignoresarmfastpath; release-before-swap cannotprovealreadyrunning.

* `web/src/guide/types.ts`: Guideblockunion/knobfields/searchtypes; sharedGUIDE_BUILT_REV0c9d4f30 stampnotproofcurrentcontent.

* `web/src/lib/api/plan.ts`: Plan document/read-model types and HTTP facade: mutations mostly structured errors; some fetch failures collapse to empty arrays/zero KPI; approve uniquely throws. Reread/reset wrappers treat HTTP success as ok without payload ok; follow backend/consumers before finding.
