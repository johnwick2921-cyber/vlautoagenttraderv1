# Assignment 20 — trader admission and orders

Source-only audit at `63968be62e44db2fb07a92883e02127b9064b0be`, isolated `/tmp/nofx-understanding-surfaces-20260913`, initially clean. All **26 assigned files / 7,375 lines** fully read. Named-function census: **217 declarations**; callbacks grouped with enclosing declarations. No tests executed, production state inspected, settings changed, broker calls made, or source edits made. [A] means exact source inspected; [B] means consequences inferred from those paths. No finding below claims a runtime incident occurred during this audit.

Rules read: supplied/root AGENTS, tracked CLAUDE-canon, AUDIT-CHECKLIST initial classes and pre-audit R1–R10 / pre-cutover portion, class 121, relevant SYSTEM-MAP and rulebook sections. `trader/AGENTS.md` is absent in this worktree. Root owns branch claim and repairs. Spec freshness records, both at reviewed base: `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap` for SYSTEM-MAP.md and VL-TRADING-RULEBOOK-v1.md. Older narrative and embedded incident anecdotes were treated as historical claims, not fresh evidence. Current owner ruling is daily loss; no mandatory per-trade cap is proposed.

## End-to-end source model

[A] Startup calls `LoadTradersFromStore` at main.go:251. The manager builds traders and starts `Run` in a goroutine when stored IsRunning is true (manager/trader_manager.go:563, 716–727). Run immediately invokes tickOnce (trader/auto_trader.go:973). The cycle builds context and structure, then invokes `maybeManageArmedOrders` (auto_trader_loop.go:432), ahead of the decision model result and decision execution gates.

[A] `maybeManageArmedOrdersAt` (armed_executor.go:197–885) requires enabled day plan, store and NT exchange; boot-sweeps old rows, drains updates, resolves active plan/lifecycle, retires superseded unplaced versions, resolves bars/ATR/config, and evaluates session risk once. It computes one-setup verdicts and retires explicitly declined unplaced rows. Each scenario/leg obtains condition-derived order kind; reject-fade legs use frozen geometry, other plays retain authored/anchor/ATR widest-stop composition. Quality, bias, arm validity, R:R, HTF, open-position, canonical EntryGate, and one-setup admission precede UpsertArm. Shadow observations and split-sibling logic precede a separate placement pass.

[A] `runArmedPlacementAt` (:1174–1331) reads all nonterminal rows for the trader, evaluates account commitment once, then consumes rows whose state is armed. Limits require acceptable side and proximity; stops require seam and retest conditions plus stop-side adjudication. Both use slot/account guards and register BeginPlacement before sending through TCPTrader. Trailing passes reconcile stale rows, drain events, confirm placements and cancellations, record boot reconciliation and clear recovered-book alerts. The placement pass does not rerun EntryGate or the scenario authoring gates.

[A] A requested placement is place_pending until a received entry frame or fresh live book proves it. A cancellation normally enters cancel_pending, retains its slot and settles against a persisted fresh snapshot, preserving its ID as evidence. Entry fills materialize positions and attach plan lineage. Cancellation timeout keeps pending rather than inventing success. These are separate states and separate evidence requirements.

## Findings and repair boundaries

### S1 — boot refusal does not bind ARM entry, and startup assertion is late

[A, BROKEN source contract] The only production `kernel.TradingRefused` call found is auto_trader_orders.go:205, conditional on decision open_long/open_short. No call exists in EntryGate, maybeManageArmedOrdersAt, runArmedPlacementAt, placeOneStopEntry, TestArmPlace, TestArmPlaceStop, or the inspected TCPTrader limit/stop entry methods. The default atomic is false until assertion. main.go:289 asserts after the load/autostart chain above, contradicting its own “before any trader cycles” comment.

[B] A process whose assertion refused still has an ARM path to an entry if its remaining gates permit. Stored IsRunning creates an ordering race even for the decision path. This is a proven source path discrepancy, not a demonstrated unwanted order.

Repair boundary: moving AssertBootIntegrity before LoadTradersFromStore closes startup ordering. Its direct dependencies are build metadata, release expectation and self-contained embedded golden fixtures (kernel/boot_integrity.go:122, golden_selfcheck.go:34–125), with no loaded-trader requirement identified. **EntryGate alone is insufficient**: existing armed rows are consumed later independently. An admission check must reach the armed-state branch in runArmedPlacementAt and direct test seam entries, while retaining trailing settlement/cancel/protect/exit management. Adapter-only checks would cover direct seam calls but require dependency/call-site review. Existing kernel/boot_integrity_test.go tests the latch and assertion, not either actual entry path.

### S2 — terminal rows with a signal bypass manual-cancel-wins

[A, BROKEN] `store/armed_orders.go:291–307` creates a successor for terminal signal-bearing rows and returns before the same-version/manual-cancel check at :345. Thus the later check only protects terminal rows without signal IDs. The production caller (armed_executor.go:798–839) does not find terminal rows in ListNonTerminal and calls UpsertArm again for the same current scenario/version.

[B] Owner-cancelled or filled broker placements can be reauthorized under the same version, contrary to SYSTEM-MAP re-arm law. Existing working/place_pending and cancel_pending branches do correctly refuse replacement. Book/account guards still constrain immediate simultaneous execution; they do not restore same-version stickiness after a position/order is gone. Root was given exact ordering for repair.

### S3 — boot sweep treats successful send as broker cancellation

[A, BROKEN] class33_boot_sweep.go:91 sends through CancelOrder and :100 writes terminal cancelled immediately on nil error, without a received cancel event or book. Its counter/log use CANCELLED. It also runs before initial fill drain (armed_executor.go:210 vs :228). Existing cancel_pending rows are excluded from ListPreBoot; `TestArmSweepLeavesPendingCancelForSnapshotConfirmation` proves only that exclusion, not the unsafe working/place_pending branch.

[B] A stale working row can become locally terminal while its broker order remains live or has filled. Guards reading fresh broker state reduce successor stacking, but the claimed settlement/evidence is still false. Current C# HandleCancelOrder is entry-only, so this finding does **not** establish removal of an existing protective stop with the current AddOn source.

### A1 — stale book accepted by cancellation safety

[A, BROKEN freshness contract] arm_cancel_safety.go:174 and :189 discard liveBook's age. `liveBook` returns haveBook=true for any existing cache snapshot. A stale empty or entry-resting book can allow cancellation, whereas slot/account/reaper/settlement functions reject old evidence. Helpers also return true after send errors (:180–183), and the cancel re-request callback discards refusal/error then returns nil (armed_executor.go:1321–1325).

Negative result: current C# HandleCancelOrder (:1735–1818) never reads placedBrackets and only cancels workingEntries, protecting already-created brackets from this command. Whether that source is deployed was not inspected. Therefore historical stop-loss-removal narratives cannot be used as proof of the current risk. Separate AddOn issue to verify: it removes workingEntries before Cancel succeeds and unconditionally removes pendingBrackets even if cancellation fails or races a fill; root informed.

### A2 — authorization refusals do not uniformly invalidate old authorizations

[A] oneSetupVerdictsAt panic recovery empties verdicts (:109–113); consult fails closed on missing verdict (:234), but oneSetupRetireDeclined skips missing verdicts (:379). Existing armed rows remain eligible for placement. Ordinary gate refusal cancellation loops similarly operate on signal-bearing rows; unplaced authorized rows can survive later quality/invalidation/entry gate refusal. Geometry refusal has a stronger explicit retirement path and stops the cycle if retirement fails (structural_geometry.go:244; armed_executor.go:484–499).

[B] The intended fail-closed one-setup/error behavior does not cover inheritance in all paths (AUDIT-CHECKLIST class 121). Conditions and fixture needed: existing armed row, current missing/failed verdict, fresh empty broker book, flat account, price in band. Existing one_setup_retire_test drives explicit decline, not missing verdict, and structural_geometry_wire_test correctly covers geometry retirement write failure. This warrants a production-callsite test before repair design.

### A3 — stop outcome erased before pass bookkeeping

[A] `placeOneStopEntry` returns void (:1548) on no-op, through-cancel, slot refusal, old AddOn and send failure. Its caller (:1259–1270) always sets placedThisPass=true and cancels other arms in the plan. [B] No placement can thus retire sibling authorizations and log “reached the wire.” Stop seam/condition gates constrain reachability; no runtime seam setting was read. Return an explicit attempted/sent outcome if repairing; distinguish a failure before registration from uncertain socket outcome.

### B findings / bounded unresolved issues

- [A] cancelSettled uses age but no cancel-request/placement timestamp. confirmPendingCancels can cite an empty snapshot still within freshness bound but older than the order request. [B] Absolute freshness does not prove causal absence. Request-time evidence ordering deserves a dedicated test.
- [A] entryGateForArm and entryGateForDecision test p.Side against lowercase strings (:365/:430), while position storage is canonical uppercase and GetOpenPositions does not normalize (store/position.go:653). Canonical EntryGate position input can be empty. Negative: arm legacy oneLiveArmGuard checks matching symbol regardless of side; oneContractGuard separately reads broker positions. Thus this is a shared-gate inconsistency, not proof ARM can add to a position.
- [A] cancelSplitSiblingOnStopOut starts from ListNonTerminal but then searches those rows for state filled (:1076–1083); filled is terminal. [B] Filled sibling is unavailable, making the cancellation path inert under current classifier. One-contract policy presently also restricts split entry usage.
- [A] production authoring creates a fresh row with State armed and only copies existing.ID (:792–805). Later churn branch checks row.State == working (:853), which cannot be true for that constructed row. Store correctly refuses working-row rewrites; error is ignored. [B] Bracket modification churn code is inert, although durable working prices are protected.
- [A] fadeFactsAt marks IB complete after 12 available session bars without verifying contiguous first-hour buckets (:97), unlike OR's exact opening-bar requirement. BackfillFadePermission uses live FuturesBarsProvider's latest 400 5m bars for a past time; older history becomes unavailable despite store bars. Calendar/map backfill coverage is partial, not full reconstruction.
- [A] BackfillOneSetupVerdicts chooses final close from a window ending openedAt+60s (:174–182) and forces BandPts=0, while production uses a daily-ATR reachability band. [B] “recomputed” does not demonstrate identical point-in-time production input; final minute can include information after episode open. Historical research must label this difference.
- [A] NoChase boot prints ATR stop-floor run ceiling even when NOCHASE_MAX_RUN_PTS overrides runtime. lastTouchFor returns the cited level itself, so known run distance equals distance-to-level, not a separate elapsed excursion measure. No-chase and far-arm remain observation-only.
- [A] live cache order reads and separately fetched persisted snapshot IDs can differ; placement/slot diagnostics may cite an ID that did not supply the actual live orders. Cancel settlement exclusively uses persisted snapshot, avoiding this particular mismatch.

## File ownership and coverage

The four artifacts attach every assigned file and named declaration. Files group by actual role:

- Admission/placement: armed_executor, entry_gate, arm_kind_compose, arm_stop_anchor, structural_geometry, invalidation_resolver, one_setup_wiring, one_contract.
- Broker evidence/lifecycle: arm_cancel_safety, cancel_confirm, place_confirm, reaper_snapshot.
- Clock/cadence: discard_burn; its deferred kicks and stop/drift reevaluation concern paid decision calls, not ARM execution.
- Research evidence: fade_facts, fade_stamp_wiring, fade_boot, follow_plan_wiring, scenario_anchor, scenario_level_identity, one_setup_boot. These write evidence/episode records; the follow recorder never reaches an arm or wire. Fade predicate also feeds live one-setup admission, so older “LABEL ONLY, no refusal” narration describes the stamp alone, not all consumers.
- Observability: arm_far_counter, no_chase, arms_boot_line, fade_surface, scenario_economics_desk, structural_geometry_boot.

Configuration binds to the trader's strategy, with individual session overrides and explicit env resolvers. Structural reject geometry freezes zone provenance, buffer, first distinct profit-side zone and costs before admission; nullable unavailable values remain distinct. It reports one-contract loss but does not impose a removed extra per-trade cap. Legacy nonreject stops still widen according to anchor/ATR rules. Stop placement seam and AddOn capability remain separate from condition-kind availability.

## Tests and historical graph

Read fully: cancel_confirm_test, one_setup_retire_test, structural_geometry_wire_test, arm_sweep_settlement_test and kernel boot integrity tests; arm_cancel_safety_test read through the ordering test. Exact source ranges appear in reads.json. Tests establish intended contracts and useful isolated wire harnesses; this audit did not run them or claim they pass. Root should use actual production entry/cancel call sites for repairs rather than pure predicates alone. New tests should cover both directions and preserve cancellation, placement settlement and protective management while refusing entries.

Historical Understand Anything graph at July10@7a8adce0 contains **zero nodes and zero incident edges for these 26 paths**. This is absence of newer subsystems in that graph, not absence of current functionality. Current AST catalog supplies 217 named declarations; call expressions are explicitly syntax-only and not type-resolved. CGC service was not queried by this worker; root owns exported historical index evidence. Current source is authoritative.
