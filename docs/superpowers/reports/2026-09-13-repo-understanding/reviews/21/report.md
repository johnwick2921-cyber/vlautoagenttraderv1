# Assignment 21 — planning and context interfaces

**Complete:** 15/15 assigned files, 6,816 lines manually read in full; no unread assigned source. **197 named functions/methods** cataloged with exact boundaries, syntax-only call expressions, purpose and invariants in `functions.json`. Closures belong to enclosing functions; file-scope clock/provider callbacks are described with their owning file. Baseline `63968be62e44db2fb07a92883e02127b9064b0be`, isolated execution worktree `/tmp/nofx-understanding-execution-20260913` verified clean. No source/config/account/database modifications; no live data, credentials or services accessed; no assigned function/test executed. Artifact generation parses source and verifies hashes only.

[A] = directly inspected current source. [B] = consequence inferred from that source. Findings are static code/interface defects or explicit limitations, not reproduced runtime incidents. Existing comments' historical counts/boots are not fresh evidence. This review does not certify current broker state or profitability.

## Rules and graph provenance

Common review instructions were reread. AGENTS/CLAUDE canon had been read in this worker's prior assignment; applicable audit R1–R10, SYSTEM-MAP research archive boundary, and current RULEBOOK structural-stop/daily-loss sections were read again as recorded in `reads.json`. The expected `trader/AGENTS.md` does not exist in this worktree; no dangling path was treated as an instruction.

Spec freshness (`git log -1` at the review pin):

- AUDIT-CHECKLIST: `dfda15e1 2026-09-13T00:48:28-05:00 test: isolate session clock fixtures from weekly backfill workers`.
- SYSTEM-MAP and RULEBOOK: `565e8fbe 2026-09-13T00:39:02-05:00 fix: use owner daily-loss controls without requiring a per-trade cap`.

[A] RULEBOOK:195 explicitly says the configurable loss limit is DAILY and no additional per-trade cap is required. Nothing in these 15 files implements a new mandatory per-trade dollar cap. Session entry-count limits, consecutive-loss breaker, reauthor-call budgets and daily-loss switches are different controls and should not be conflated. No recommendation here restores the removed per-trade cap.

[A] Historical Understand Anything graph (`2026-07-10`, `7a8adce0`) has **zero nodes/edges for the 15 assigned file paths**. `graph.json` preserves that negative result and adds exact current source edges. The old CGC index is not presented as current. Root Go AST symbols/calls are only a declaration census and syntax-level call list, not type-resolved or runtime reachability proof.

## Source flow and ownership

[A] **Scheduling:** `auto_trader_clock.go:87–125` invokes weekly/session reads before bar-dependent cycle skips. Weekly document precedes Sunday ASIA (`planner.go:193–255,1452–1459`); each trader/session/day has a process-local claim (`:828–843,867–872`). Separate sessions can run concurrently. `runCycle` separately performs calendar capture (`auto_trader_loop.go:196`), frozen session-profile persistence (`:262`), digests (`:283`), level/scenario observations (`:298–303`) and weekly reaction summaries. While holding, equity/context are saved before the watch-only branch (`:460–480`). Calendar's closed-hours claim must be distinguished from the weekly/session wall-clock path: its only inspected production caller remains inside runCycle.

[A] **Planner input:** `assemblePlannerInputWithCtx` (`planner.go:2143–2527`) reads root-symbol bars, HTF levels, owner-scoped sticky levels, scored pool, current regime, digests, calendar, indicator/config fingerprint and weekly references. Store-deepened1m tape is shared by candle tables, RV baseline and completed-week count (`:2275–2304,2393–2406,2458–2463`). Machine zones and identity candidates accompany authored text; research snapshot UUID ties input/candidates to the authoring trace. Model chooses scenarios; gates judge them.

[A] **Authoring:** `runPlannerReadWithTriggerClaimedCtx` (`:863–1020`) claims, clock-checks and preflights before resolving client and rendering input; model calls use stream idle/total deadlines for base clients and legacy full-body fallback otherwise. `runPlannerReadCoreObserved` (`:1480–2024`) attempts at most3 calls: transport failures resend identical text; validator failures repair or reauthor with reasons. Schema/caps, required bias, structural-label provenance, facts, FVG/breakdown rules and authored scenario economics are validated. Advisory feasibility/role/fantasy-target warnings do not reject. Scheduled/death/owner failures append NO-TRADE; opportunistic wake failures keep prior plan. Append and recorded spend are separate writes (`:1979–2012`).

[A] **Publication and consumption:** model/indicator/config hashes and attribution are persisted with the plan; owner edits carry by price identity (`:395–478`). `installActivePlanProviderAt` (`:2713–2786`) serves this trader's runnable current-session latest active plan, resolved with overlays, alongside stored remaining budget. Bad overlay validation warns and falls back to base (`:2794–2825`). Plan ID's `StrategyID` database field intentionally contains trader ID (`:2645–2647`); it is not evidence of an erroneous join by itself. Root-symbol argument is not an independent selection key in the active-plan closure because the trader/session chain owns the plan.

[A] **State and risk:** level state is trader/symbol/type/bin scoped; scenario status and metadata are trader/plan/version scoped (`auto_trader_levelstate.go:92,238,265`). Session first/lunch/news bands and optional account-scoped entry-count limits feed the arm risk gate (`auto_trader_session.go:35–54`; `session_risk.go:120–137`; `armed_executor.go:336–350`). Breaker is trader/day scoped and deliberately fail-open on query failure; unknown corrected outcomes break a known-loss streak (`store/position_query.go:92–133`). Daily-loss enforcement is outside this slice; boot labels decorative if master or daily leg is off (`session_risk.go:143–166`).

[A] **Observability:** weekly reasoning is references-only, with shadow bias and no directional decision authority (`auto_trader_weekly.go:260–301,426–489`). Watcher has no broker calls (`auto_trader_watcher.go:243–445`), ignores action-like AI fields and writes assessment records. StageA adapters preserve explicit missing costs and strict corrected P&L, separate observation/receipt/publication/permission, and state that placement return is not a broker acknowledgement (`research_snapshot.go:103–159,164–224,273–306`). `researchsnapshot.Record` queues builders; worker evaluates them later (`recorder.go:45–80,99–117`). Local slices/pointers passed to adapters therefore rely on the documented transfer/immutability convention; primitive outcome/placement fields are explicitly copied. No snapshot race was reproduced.

## Findings

### 1. Planner arm instructions diverge from structural reject-fade execution [A/B]

[A] `planner.go:2485–2488` supplies ATR floor fields, and `kernel/planner_prompt.go:798` instructs every arm to meet an ATR minimum stop distance; otherwise omit the arm and let the AI path enter. The same prompt `:795` states resting orders are the only entry path. Current structural-aware composer `trader/arm_stop_anchor.go:76–88` delegates reject-fade geometry to the structural core instead of the legacy widest-ATR branch. RULEBOOK:193–197 explicitly removes ATR override for usable structural invalidation. Strict mode refuses non-arm entries (`entry_gate.go:184–197`).

[B] The planner can omit or distort otherwise structurally valid reject-fade arms because its instructions describe a different execution contract and suggest an unavailable strict-mode alternative. This is source-proven prompt/contract drift; no model response or rejected trade was produced here. It is unrelated to requiring a per-trade cap. `planner.go:1694–1701` also retains generic ATR feasibility warnings, and `:1977` calls zones presentation-only despite their downstream structural role.

Parent follow-up (not independently verified by this worker): root reports the structural prompt mismatch above was reproduced and fixed in repair `710ea1c8`. This report remains a review of baseline `63968be`; do not rediscover it as an unfixed repair-branch issue.

### 2. MSS wake bypasses advertised full cadence controls [A/B]

[A] `class47_wake_cadence.go:121–126,220–224` names both `level_event` and `structure_mss` as governed. The level caller actually runs cutoff/cooldown/fast-market and any-stream checks (`auto_trader_wake_levels.go:289–360`). MSS caller (`auto_trader_transition.go:156–212`) performs only event dedupe and shared minimum-attempt interval, then calls the claimed read. Full read `planner.go:863–1020` has no class47 cadence check. Current source search found no additional production SkipForCutoff/SkipForCooldown/anyPlannerStreamOpen callsites.

[B] A fresh MSS can start a read too near flat, inside the written-version cooldown, or beside a different session's planner stream even while boot prose says otherwise. Per-chain claims still prevent duplicate same-chain calls; this is not a claim that every stream overlaps. `class47_wake_cadence_test.go:306–317` tests trigger classification only; `:165–199` tests claim visibility/log wording, not MSS callsite enforcement. No runtime timing scenario executed.

### 3. Read-only planner-client accessor can clear global research statistics [A/B]

[A] `ResolvePlannerClient` is documented read-only (`planner.go:44–48`) but delegates to model pinning (`:67–136`) and global `dayplan_pinned_model` bookkeeping (`:141–157`). A changed pin calls `MatchedRandom.ResetWindow`, which deletes all rows from both matched-random tables (`store/matched_random.go:108–116`). AskPlanner invokes this accessor before its Q&A call (`api/handler_plan.go:1447`); realignment also invokes it (`:2184`). Neither global model key nor reset is trader-scoped.

[B] Q&A/client resolution is therefore not read-only with respect to learning statistics. Two traders with different model bindings can alternately reset the shared window. Reset and new pin writes are not atomic, and pin is written even after reset failure. No data deletion was performed in this review; these are explicit source paths. Model-change reset itself is intended policy, but its ownership and accessor side effect need visibility.

### 4. Closing transition drops the identity needed to clear the persisted chip [A/B]

[A] Resumption/expiry reset `at.transition` to an empty struct (`auto_trader_transition.go:82–94`), then `persistTransition(false)` builds another identity-less struct (`:127–137`). `store.TransitionKey:446–451` returns empty for empty plan ID, so no inactive record is written. API reads the old version key and returns any stored Active=true state (`api/handler_plan.go:1241–1257`).

[B] The runtime gate can close while its same-version card chip remains active. Replacement naturally moves the reader to a different version; resumption/expiry on the same version exposes the stale-chip case. Existing transition test checks opening persistence (`auto_trader_transition_test.go:55–72`) but checks only memory/context on closing (`:74–135`). No endpoint or DB reproduction was run.

### 5. Watcher structure-conflict hysteresis never advances its remembered verdict [A/B]

[A] `runWatchCycle` compares assessment conflict with `st.SCVerdict` and increments/reset counts (`auto_trader_watcher.go:395–406`), but no assignment to `SCVerdict` occurs anywhere in the inspected production source. New states start empty (`:268–271`).

[B] Repeated nonempty conflicts repeatedly reset to count1 and never reach the two-read accepted conflict from default state. This affects advisory badge and combined warning, not order authority. R1/R2/R3 thesis rails are separate and do update `Status`; they should not be described as broken by this finding. `watcher_test.go:138–155` is a source-string no-order-authority guard; it does not exercise this conflict-memory logic.

### 6. Calendar “fail-closed” is conditional fallback, not closure on unknown coverage [A/B]

[A] Missing/invalid calendar slice loads static events (`auto_trader_calendar.go:172–188`); missing/invalid static file returns nil (`:116–128`). Empty events produce no blackout windows and are returned at `:202`; `currentT1Windows` also returns nil without store/session. No whole-session refusal exists in this function despite fail-closed alert/log wording. Planner input `planner.go:2306–2323` only reads stored slices and does not use that static fallback, although later structured windows do (`:1907–1910`).

[B] If both feed snapshot and static fallback are unavailable, this surface supplies zero news protection; the log can say fail-closed with0windows. With valid static data it is a useful fallback and should not be called wholly ineffective. No calendar outage or current fallback contents were inspected. The planner's initial calendar text and entry gate can also differ during fallback.

### 7. Digest labels can freeze whole-day totals as separate session totals [A/B]

[A] `maybeWriteDigests` queries one current CME-day total (`planner.go:2557–2561`) before looping every runnable session currently outside its window (`:2563–2591`), and passes that same entries/P&L to `FormatSessionDigest`. SaveIfAbsent freezes whichever total first reaches that session/date key. Errors from activity read are ignored. Daily error digest is appended twice (`:2604–2608`).

[B] Multi-session summaries can include another session's activity or a premature/default total, and then seed the planner digest chain (`:2334–2336`). These are contextual summaries, not the actual daily-loss gate. Exact affected rows, session-chain-date behavior outside the read window, and runtime frequency were not queried.

### 8. Confirmation grace is not the documented distinct-session retry contract [A/B]

[A] Missing-confirm handling is after the3-attempt loop (`planner.go:1844–1861`), so exhausted grace nulls the otherwise parsed document without giving that defect back to another attempt. Counter key is process-store global (`:2892`); `confirmGraceExhausted` increments per check (`:2896–2906`), with no trader/session identity or dedupe and before append succeeds. A compliant doc immediately exhausts it (`:2911–2915`).

[B] The comments promising distinct sessions and retry-on-reject overstate the contract. Missing confirm may already be refused by narrower upstream schema/economics rules; reachability for a particular scenario shape is unverified here. The explicit grace path itself is not a distinct-session counter.

## Other limitations and negative results

- [A] `session_risk.go:165` always says the breaker never fires on retained max-run7, although `breakerHaltN:62–66` accepts configured thresholds below/equal7. That historical clause is not valid for every resolved knob; no change to owner threshold is warranted. Multi-strategy boot returns n/a (`:265–268`), correctly avoiding arbitrary binding selection, though wording does not distinguish ambiguous from absent binding.
- [A] `matched_random.go:25–48` attributes the closed trade to the latest plan for the entry session, not the position's immutable plan/version; reaction is MFE>MAE, not a sampled random comparator. Weekly snapshots pool by type. This limits statistical interpretation, not execution permission.
- [A] `wakeTimePrice:394–402` and `planner.priceOf:641–645` return final Close without verifying closedness despite comments. Level invalidation collector `wake_levels.go:179–199` checks freshness but not close time>plan birth; it can wake on a recent already-known violation. MSS throttle `transition.go:186–187` uses real time.Since inside an injected-now function. No feed ordering/formed-bar race reproduced.
- [A] Successful weekly docs are cached under mutex; failed weekly calls write no marker (`weekly.go:300–302`), so scheduler retries on later cycles (`:182–212`) instead of only one pair for the whole week. Sunday ASIA remains deferred while no weekly doc exists (`planner.go:1452–1459`), contrary to broad comments that weekly absence affects nothing else. This is recoverable retry behavior but can extend the dependency indefinitely during failure.
- [A] Plan append precedes replan spend (`planner.go:1979–2012`); spend failure is loudly disclosed as potential over-allow. Claim prevents same-chain concurrent provider calls but owner/death budget checks occur before claim; no atomic budget/admission transaction is proven. Same trader can have different-session reads concurrently, with shared `lastRegimeHealth`, `fastTapePending` and possibly primary AI client configuration. Data race/incorrect publication is UNVERIFIED without focused concurrency testing.
- [A] StageA explicitly labels fees unknown (`research_snapshot.go:221`) and activation not permission (`:110,124`). Detector prior episodes are not historical first-availability proof (`:265`). These honest boundaries should be preserved. Snapshot capture never supplies live trading decisions in the inspected callers.
- [A] `sessionHiLoFromBins` leaves high=-Inf on empty input because it checks positive rather than negative infinity (`dayplan.go:153–169`); production caller excludes empty bins (`:123`). This is a dormant helper edge, not an observed bad persisted profile.

## Validation scope

All assigned SHA256 hashes match the manifest. Full/manual ranges cover every assigned line. Go AST census contains197 named declarations; every one has a purpose and boundary in `functions.json`. Anonymous callbacks remain explicitly grouped. Relevant source tests were located; transition test was read fully and targeted watcher/cadence/research snapshot excerpts were read. None was executed. A source review avoids the network/LLM/account activity those production paths can invoke. Suggested next verification is parent-owned minimal offline fixtures for transition persistence, watcher conflict sequence, MSS caller cadence, and model-reset scoping; no new tests or source edits were made here.

## Per-file purpose and coverage

- `trader/auto_trader_calendar.go:1–218` — Calendar producer live3h refresh/hour retry with static fallback, day/session event mapping and drift widening. Missing slice+missing static returns zero windows despite fail-closed wording; no blanket closure.

- `trader/auto_trader_dayplan.go:1–238` — Session-profile persistence and global once-installed nPOC/state providers; active plan pertrader. symbol-only profile identity versus contract-scoped historical touch bars; callback ownership and stale saved profile lineage need checking. Empty bins excluded at caller; helper empty hi=-Inf typo latent.

- `trader/auto_trader_levelstate.go:1–302` — Trader-scoped level durable state/consumption plus plan-version scenario status/meta/death recording; closed birth window and canonical identity; zero statuses returns without clearing prior key; scenario meta best-effort write followed by snapshot recording.

- `trader/auto_trader_matched_random.go:1–89` — Reaction proxy closes matched to latest entry-session plan not immutable entered version; weekly snapshot Sunday CT once ISOweek, counts pooled bytype; no random comparator built here.

- `trader/auto_trader_planner.go:1–3054` — Planning pipeline: pertrader/session claims, bar/clock preflight, model binding, frozen context/maps, three-attempt author/repair/resend validator chain, nonfatal wake versus NO-TRADE failure, append and separately recorded replan spend, overlays/provider/digests. Global model pin resets pooled statistics despite read-only accessor; universal ATR feasibility prompt diverges structural reject path; confirm grace outside retry and global/non-session counter; cached config/current state clocks not fully atomic; digest uses session-day totals for each closed-session label. No mandatory per-trade loss cap in this file.

- `trader/auto_trader_reread.go:1–140` — Owner reread gate session/market/budget and clock hold; rechecks budget then planner claim; alerts spend before actual claim; carry overlays from new version. First read free despite spending wording.

- `trader/auto_trader_session.go:1–176` — Session gate calls configured runnable, account-scoped maxTrades with query error fail-open, first/lunch/T1 shared resolver. Night-edge logs registry alone, may diverge per-session override.

- `trader/auto_trader_transition.go:1–212` — Transition open/resume/expiry state; clears identity before persisting inactive, likely stale card chip. MSS wake uses time.Since despite At clock; only basic throttle here versus full class47 level path; shared read may enforce later.

- `trader/auto_trader_wake_levels.go:1–403` — Level-wake collect/prioritize new zones/FVG/OB/iFVG/invalidation; invalidation lacks close after planbirth; ATR uses full provider slice. Full cadence cutoff/cooldown/fastmarket/any-stream guards, async nonfatal read, consumes key before read. wakeTimePrice claims closed but takes lastbar unfiltered.

- `trader/auto_trader_watcher.go:1–465` — Observer watch-only no broker calls. Approx thesis latest40 +/-time not signal; firstposition only. Structure hysteresis never writes SCVerdict so nonempty conflict never accumulates2 from empty. Rails WARN uses R1accepted independent R2 display; currentStop inferred from local intent flags not brokerreceipt.

- `trader/auto_trader_weekly.go:1–494` — Weekly async refs-only read/currentcontract resolver, no directional trade gate. Failed calls no row -> nextcycle retries pair despite onceweek wording; successful cache mutex; shadow maps never pruned resetWeek. Legacy stored1m lacks CloseTime, rely downstream semantics.

- `trader/class47_wake_cadence.go:1–241` — Pure wake cutoff/cooldown and log helpers; governs level_event/structure_mss. minutesToFlat uses sessionwindowend not resolved forcedflat offset; inspect downstream caller. Global syncmap stream indicator.

- `trader/detector_record.go:1–194` — Detector recording recover-contained, formation-windowed episodes, persisted watermark/ordinal and heuristic scenario link, candidate pool/follow telemetry. Delta computed over full scope not episodecausal; stated historical diagnostic basis. No order authority.

- `trader/research_snapshot.go:1–308` — StageA snapshots preserve explicit NULL/missing versus0, source clocks, stable identity, strictcorrected outcome exclusions/costunknown; async closures ownership relies Record sync serialization; inspect Record.

- `trader/session_risk.go:1–282` — Session risk band first, breaker strategy/env then defaults8/5; count trader/day not account. Explicit query error failopen. Daily boot decorative if either switch off; fixed maxrun7 neverfires claim false for configurableN<=7. No pertrade cap added.
