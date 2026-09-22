# W-PICTURE-HTF — end-to-end verification (2026-09-20, final)

Branch `fix/picture-htf`, base `origin/dev` @ d7ca3846.
**Not merged, not deployed. Cutover HELD pending CTO review of this evidence
and owner-attended NT8 steps.**

This revision supersedes all earlier revisions. Two PRODUCTION corrections
landed this round (c79fbe0b) — the implementation differed from the agreed
behavior; the strategy itself is unchanged. All earlier replay claims were
already recomputed (e55fea50) and are restated here with the sequence proof.

## 1. Reviewed SHA + incorporated dev

- Wave head (this report's subject): **c79fbe0b** (code) — reviewed commits
  c79fbe0b, e55fea50, 0dedc0e8, 939b4507, 5dbf59eb, 845672f9, 1fa15ccb,
  4e8f7931, 38585854, 42c35e2d, efddb329, 1b8bcc8e, f33eb7da, f1099f4b,
  17f9fda6, f92dbb80, 44aa5248, ceb894b9.
- Incorporated dev: `d7ca3846cf8df7610f09162be7583c05b7a85f47`
  (`git merge-base origin/dev HEAD` == the dev tip → the branch contains all
  of current dev; dev has not moved).

## 2. THE TRADING BEHAVIOR — requirement → file:line → test → result

Legend: **[pin]** automated production-path test · **[replay]** Sept-17
historical replay evidence · **[audited]** code-level audit only ·
**[missing]** not yet evidenced (see §7).

| # | Requirement (spec) | Production call site | Evidence | Result |
|---|---|---|---|---|
| L1 | Levels from completed native NT8 4H candle BODIES (top=max(o,c), bot=min(o,c)); wicks kept separately | `kernel/picture_htf.go:56` BodyPivots4H; replay loads native stored bars | [pin] TestBodyPivots4HFindsBothPivotsAndNoLookahead · [replay] 21 levels in force on 09-17 | PASS |
| L2 | Strict 2+2 pivot confirmation; no equal-value plateaus | `kernel/picture_htf.go` (4-neighbor loop, res∧sup→skip) | [pin] TestBodyPivots4HEqualBodiesDoNotFormPivot | PASS |
| L3 | Level usable only after BOTH following candles complete | `kernel/picture_htf.go:141` ActiveLevels (KnowableAt) | [pin] TestH1CloseBreakOneTickRule (not-yet-knowable level must not fire) | PASS |
| L4 | 120-bar discovery window; retirement rules | `kernel/picture_htf.go` (window cap + far-edge-close retirement) | [pin] TestBodyPivots4HRetiresOnFarEdgeClose · [replay] 21→23 level evolution | PASS |
| R1 | 4H wick below support + close back above = bullish rejection context; mirror resistance; ADVISORY only | `kernel/picture_htf.go:158` SweepReclaim4H | [pin] TestSweepReclaim4HSupportRejection | PASS (advisory; never a prerequisite — no gate reads it) |
| H1 | Level known before the confirming H1 opened | H1CloseBreak filter `KnowableAt > newH1.OpenTime → skip` | [pin] TestH1CloseBreakOneTickRule | PASS |
| H2 | Long: prev close ≤ resistance; new COMPLETED close ≥ resistance + 1 tick; mirror short | `kernel/picture_htf.go:199` H1CloseBreak (Close only) | [pin] TestH1CloseBreakOneTickRule + TestH1CloseBreakMirroredShort | PASS |
| H3 | Forming or wick-only crossing does NOT qualify | H1CloseBreak uses Close; wick high is never read | [pin] TestH1CloseBreakOneTickRule ("wick-only crossing must not fire") | PASS |
| H4 | Multiple crossed levels: highest resistance (long) / lowest support (short) wins; rest ride as context | `kernel/picture_htf.go` sort + `res.Crossed` | [pin] TestH1CloseBreakExtremeLevelWins | PASS |
| H5 | Simultaneous H1/4H completion: preserve the pre-close breakout reference; apply retirements before target selection | `trader/picture_htf_evaluator.go:188-214` breakLevels (4H bars completed BEFORE cur.OpenTime) + as-of-now snapshot for the target | [pin] TestPictureHtfSimultaneousH1And4HCompletionPreservesBreakoutReference (both frame orders) | PASS — production corrected this round |
| H6 | Arrival-order independence | both snapshots time-derived from the cache; no frame-order state | [pin] same test, orders "4h-first" and "5m-first" | PASS — production corrected this round |
| E1 | Market entry at the START of the following native 5m interval; no wait for that candle's close / AI cycle / retest | `trader/picture_htf_evaluator.go` intervalStart = newest completed 5m open + 5m; submit seam runs in the same evaluation | [pin] TestPictureHtfH1CloseToNext5mSequenceNativeAlignment · [replay] per-break SEQUENCE block (interval opens == next native 5m bar's open) | PASS |
| E2 | Submission within the 10s window | elapsed vs EntryWindowSec gate | [pin] TestPictureHtfEvaluatorPastWindowExpires + SubmitsOnceAndAdmits (elapsed 0.5s) | PASS |
| E3 | 2-second live receipt freshness | freshest5mAt vs FreshnessSec | [pin] TestPictureHtfEvaluatorLateFrameExpires | PASS |
| E4 | Recheck immediately before send | `trader/picture_htf_send.go` pictureHtfSend (freshness/window/flat/pending/size) | [pin] TestPictureHtfSendRefusesStaleFeed / ClosedWindow / NilState / NonNTTrader | PASS |
| E5 | Late opportunity expires; historical backfill cannot trigger it | freshness + window gates; live-only sink | [pin] LateFrameExpires · TestDrainBarIngestHistoricalNeverFansOut · TestSept17ReplayNeverMintsOpportunities | PASS |
| S1 | Stop = latest qualifying confirmed 5m swing, strict 2+2 WICK pivot, preceding 24 COMPLETED bars, both confirmers completed by the H1 close | `kernel/picture_htf.go:251` StructuralSwing5M (strict wick comparisons, lookback 24, confirmDeadline) | [pin] TestStructuralSwing5MStrictAndDeadline | PASS |
| S2 | ONE tick beyond the swing extreme, SYMMETRICALLY (long −1, short +1) | evaluator long/short branches; harness mirrors | [pin] TestPictureHtfEvaluatorMirroredShortEndToEnd (stop == swing high + 0.25) | PASS — harness corrected earlier, production audited clean |
| S3 | No qualifying swing = refusal | evaluator refusal path | [pin] TestPictureHtfEvaluatorNoSwingRefuses | PASS |
| S4 | Never tighten the stop to manufacture R:R | no code path adjusts the stop after the swing; RR refusals keep the stop | [pin] TestPictureHtfEvaluatorLowRRRefuses (refusal, stop untouched) | PASS |
| T1 | Target = nearest ACTIVE opposing 4H body zone beyond entry; target its NEAR edge | `kernel/picture_htf.go:283` NearestOpposingZone (role-respecting, TargetEdge) | [pin] TestNearestOpposingZonePolarityAndNearest + LowRRRefuses (nearer zone never skipped) | PASS |
| T2 | No eligible target = refusal | evaluator refusal path | [pin] TestPictureHtfEvaluatorNoOpposingZoneRefuses | PASS |
| T3 | Enforce the ACTUAL saved strategy minimum + existing sizing/risk limits; 3R is a displayed reference only | evaluator minRR resolver (strategy → SafeDefault); seam sizing clamp | [pin] LowRRRefuses · TestPictureHtfContractSizeNeverExceedsClamp · [replay] both floors (2.5 default, live 2.0) | PASS |
| M1 | Completed-H1 close-progression momentum; advisory only; cannot reverse/flatten/cancel | `kernel/picture_htf.go` H1MomentumStall + evaluator records it on the row only | [pin] TestH1MomentumStallAdvisory | PASS (no gate reads it) |
| P1 | Durable unique opportunity row ≠ submission license; exactly-once send via atomic ownership | `store/picture_htf.go` PictureHtfClaim + PictureHtfClaimSubmission (UPDATE … WHERE stage='confirmed' AND signal_id='') | [pin] TestPictureHtfClaimSubmissionExactlyOneWinner (12 claimers, 1 winner, -race) · TestPictureHtfClaimDuplicateKeyRefuses | PASS |
| P2 | Reconciliation prevents a second send; never blindly retry an ambiguous command | place_pending blocks re-entry; seam reports "send ambiguous" and does not resend; the sweep recovers only RECEIVED evidence — a terminal outcome that was never received stays UNKNOWN (never fabricated, never resent) | [pin] TestPictureHtfAmbiguousSendStaysPending (re-entry refused at the store) · TestPictureHtfReconcileRecoversFillFromReceivedHistory · TestPictureHtfReconcileRecoversRejectionFromReceivedHistory · TestPictureHtfReconcileRecoversFillFromFillStream · TestPictureHtfReconcileAbsentBookStaysUnknownNeverResends · TestPictureHtfReconcileWorkingBookLeavesFillPriceUnknown | PASS |
| P3 | Received broker state moves the row (working/filled/rejected); absent fields stay empty | `store/picture_htf.go` PictureHtfMarkBrokerState/AppendBrokerStatus + `trader/picture_htf_broker.go` consumer (forward-only stages; protective legs append, never clobber the entry fill) | [pin] TestPictureHtfLifecycleAndBrokerEvidence · TestPictureHtfConsumeOrderUpdateEntryLifecycle · TestPictureHtfConsumeProtectiveLegsUpdateProtectionNotFill · TestPictureHtfConsumeOrderUpdateIsolation · TestPictureConsumerAndArmedStreamCoexistOnOneSubscription (both consumers on ONE subscription — neither loses frames) | PASS (live NT8 receipt still owner-gated, §7) |
| P4 | Restart after claim / after send / before ack — no double send | durable claim + atomic ownership | [pin] TestPictureHtfRestartAfterClaimSingleSubmission + ClaimSubmission pin | PASS |

## 3. Production path trace (native frame → broker state)

```
NT8 AddOn bar_update (final+emitted_at)      → C# VLBarsSubscriptionManager boundary finalization
 → Go BarCache.Upsert (provider/ninjatrader/tcp_server.go drainBarIngest)
 → fanOutLiveBars (LIVE frames only)          → trader pictureHtfLiveBars → AutoTrader.NotifyLiveBars
 → PictureHtfEvaluator.OnBars/Evaluate
     → capability gate (build ≥ 2026-09-20-p1, proven by receipt)
     → rebuildLevels (4H body pivots, final-marked completed bars)
     → H1 completion scan + advisory momentum
     → BREAK check: pre-close level snapshot (bars completed before the H1's open) + one-tick close rule
     → following 5m interval: intervalStart = newest completed 5m open + 5m; 10s window; 2s freshness
     → geometry: 5m swing stop (±1 tick) + nearest opposing 4H zone; R:R vs the saved minimum
 → shared gate: PictureHtfClaim (unique row) → PictureHtfClaimSubmission (atomic owner)
 → seam re-checks (freshness/window/flat book/no unreconciled pending/1-contract clamp)
 → TCPTrader.MarketEntryWithProtection (SIM + bound-account + B3 guard + identity assert)
 → NT8 executes; received order_update/fill frames carry rejection reason
 → PictureHtfMarkBroker records RECEIVED broker state (consumer NOT yet wired — §7)
```

## 4. Adversarial coverage (production-path pins)

| Case | Pin | Result |
|---|---|---|
| Forming candle / unfinalized bar | TestPictureHtfEvaluatorIgnoresUnfinalizedBars + kernel TestBodyPivots4HFormingCandleCannotBePivot | PASS |
| Wick-only crossing | kernel TestH1CloseBreakOneTickRule | PASS |
| Late frame | TestPictureHtfEvaluatorLateFrameExpires | PASS |
| Historical frame | TestDrainBarIngestHistoricalNeverFansOut + TestSept17ReplayNeverMintsOpportunities | PASS |
| Duplicate frame/restart | SubmitsOnceAndAdmits + RestartAfterClaimSingleSubmission | PASS |
| Reordered frames (H1/4H vs 5m arrival) | Simultaneous…PreservesBreakoutReference (both orders) + freshness-watch (no 5m frame → watch) | PASS |
| Missing 5m boundary frame | PastWindowExpires (window elapses fail-closed); zero-frame case now WATCHES, never poisons | PASS |
| Session boundary / halt | same window-expiry path; replay uses native session-aligned rows | PASS |
| Simultaneous H1/4H close | Simultaneous…PreservesBreakoutReference | PASS |
| DST / clock skew | epoch-ms arithmetic (no wall-time parsing); freshness = now − Go-side receipt time (same clock) | AUDITED |
| Mirrored short + exact one-tick buffer | TestPictureHtfEvaluatorMirroredShortEndToEnd | PASS |
| Missing stop / target / wrong-polarity target / insufficient R:R | NoSwingRefuses · NoOpposingZoneRefuses · kernel NearestOpposingZonePolarityAndNearest · LowRRRefuses | PASS |
| Competing old/new executors, same account/instrument | store TestPictureHtfClaimSubmissionExactlyOneWinner (12-way, -race) | PASS |
| Restart after claim / after send / before ack | RestartAfterClaimSingleSubmission · ClaimSubmission pins | PASS |
| Rejection / immediate fill / ambiguous send / partial fill | ambiguous: AmbiguousSendStaysPending · rejection reason on the wire (framing roundtrip) · live consumption pinned (TestPictureHtfConsume*: entry lifecycle, rejection reason, protective-leg fills, isolation) · partial-fill AddOn path audited in C# (VLTraderTCPClient.cs:1445 SubmitBracketOnEntryFill + filledQty>0 guard and filledQty-sized legs + AmendBracketQuantity at :2089+); live proof needs NT8 (§7) | PASS (unit/loopback) · live NT8 receipt owner-gated |
| Missing/rejected protective orders + recovery | **MISSING (§7)** — documented behavior: ambiguous row blocks re-entry until reconciled; the automatic reconciler is not built | MISSING |

## 5. Evidence categories (kept separate)

- **A. Automated production-path tests** — §2 matrix; Go 35/35, web 76/492, tsc clean, race clean (commands in §6).
- **B. Historical strategy replay** — Sept-17, labeled, read-only (§6).
- **C. Controlled synthetic NT8 SIM execution** — loopback wire pins (market entry frame, bracket, stamp ordering, no-send on stamp failure/incomplete bracket). Loopback is NOT NT8 execution proof; it proves the Go-side command composition only.
- **D. Naturally occurring market opportunity** — **none. Pending.**

## 6. Corrected replay (Sept-17) + exact test commands

**Replay inputs:** read-only COPY of live `data.db` (2.27 GB, `sqlite3 .backup`);
window 2026-09-16 22:00Z → 2026-09-17 22:00Z; contract **MNQ 12-26**, enforced
in the WHERE clause for the whole 30-day context ladder; 5m/1h = STORED
NT8-native rows (session-aligned, grid opens at :00 from the ETH 22:00Z
session start); 4h = disclosed ETH-grid proxy from native 1h (no stored
native 4h exists); entry price = close of the 5m bar completed at the entry
instant; symmetric ±1 tick stop; per-break SEQUENCE proof printed.

**Results (identical at 2.5 default and the live 2.0 floor):** 21 levels in
force at start · 22 H1 completions · 2 breaks (07:59Z over 29479.25 refused
R:R 0.45 · 10:59Z over 29581.50 ELIGIBLE: entry 29585.50, stop 29561.25,
target 29767.00, R:R 7.48). Outcome on the real tape: **stop-first loss**
(stop touched 11:44Z, low 29555; target touched 17:00Z). Slippage not
modeled. The pre-correction claims were withdrawn.

**Limitations:** stored rows predate the final/emitted_at wave → receipt
freshness is SIMULATED, not measured; native 4h absent (proxy, disclosed);
no market-fill reconstruction.

**Exact commands** (all run at c79fbe0b, logs quoted):
```
go test ./... -count=1                          → /tmp/gofinal5.log  exit 0, 35/35 packages
go test ./trader/ -count=1 -race -run TestPictureHtf … (race surfaces) → clean
go test ./... -count=1 (wave 18, pre-CTO-review) → /tmp/gofinal7.log  exit 0, 35/35 packages
go test ./... -count=1 (wave 18, CTO round-2 corrections) → /tmp/gofinal8.log  exit 0, 35/35 packages
go test ./... -count=1 (wave 20, CTO round-3 finding 4) → /tmp/gofinal9.log  exit 0, 35/35 packages
go test ./trader/ -count=1 -run 'TestPictureHtfConsume|TestPictureHtfReconcile|TestPictureConsumerAndArmed' → 13/13 PASS
go test ./provider/ninjatrader/ -count=1 -run 'TestOrderUpdateFanout' → 2/2 PASS (coordinated dispatch)
cd web && npx tsc --noEmit                     → clean
cd web && npx vitest run                       → /tmp/webfinal5.log 76 files / 492 tests, exit 0
go run ./cmd/picture_htf_replay --db /tmp/picture-htf-replay.db \
  --start 2026-09-16T22:00:00Z --end 2026-09-17T22:00:00Z [--min-rr 2.5|2.0]
                                               → /tmp/replay-final.log
```

## 6b. Broker-state pins — wave 18, CTO round 2 (2026-09-20)

The CTO reviewed the first wave-18 pass and found three production
mismatches; all three are corrected and re-pinned (see §6c). The pins below
drive the REAL receipt paths and reproduce the AddOn's ACTUAL wire
semantics:
- order_update frames carry `fill_price` = AverageFillPrice (actual fill
  evidence) and `quantity` = Filled count;
- the order_snapshot **excludes terminal orders** (Filled/Cancelled/Rejected/
  Expired — `VLTraderTCPClient.cs` SendOrderSnapshot) and carries **no fill
  price** — a filled order is ABSENT from the book exactly like a
  never-placed one;
- fill frames carry the actual fill price on the fill stream.

No test injects a terminal state or a fill price into the snapshot, and no
recovered fill price may come from `LimitPrice`.

**Received broker events update the correct opportunity and protection state**
(`trader/picture_htf_broker_test.go`):

| Pin | Proves |
|---|---|
| `TestPictureHtfConsumeOrderUpdateEntryLifecycle` | received entry events move the SIGNAL-NAMED row forward-only: working → filled with the wire's fill price/qty + actual-fill R:R; a stale working event cannot downgrade a filled row |
| `TestPictureHtfConsumeOrderUpdateRejectionCarriesReason` | a rejected entry closes the row with the wire rejection reason recorded |
| `TestPictureHtfConsumeProtectiveLegsUpdateProtectionNotFill` | a filled SL leg records `protection_sl_filled` WITHOUT touching the entry's stage or fill; protection notes APPEND (the SL evidence survives a later TP rejection) |
| `TestPictureHtfConsumeOrderUpdateIsolation` | an order_update for a different signal never touches the row |

**Restart/disconnect reconciliation recovers RECEIVED outcomes without
duplicate orders** (same file + real router/snapshot paths):

| Pin | Proves |
|---|---|
| `TestPictureHtfReconcileRecoversFillFromReceivedHistory` | a filled frame received BEFORE the row existed is recovered to filled with the frame's actual fill price and R:R; the recovered row blocks re-entry |
| `TestPictureHtfReconcileRecoversRejectionFromReceivedHistory` | a rejection received before the row existed is recovered with its reason |
| `TestPictureHtfReconcileRecoversFillFromFillStream` | a fill on the real fill router → received-fill ring recovers the row with the actual fill price (never a limit) |
| `TestPictureHtfReconcileWorkingReceiptThenFillFrameRecovers` | FINDING 4(a): a working order_update moved the row to working, then a fill frame arrived with NO filled order_update — the sweep recovers the actual fill for a WORKING row and no additional entry is submitted |
| `TestPictureHtfReconcilePendingPrefersFillOverOlderWorkingHistory` | FINDING 4(b): a pending row with older working history + newer fill evidence — the fill wins; execution evidence outranks working receipts |
| `TestPictureHtfReconcileRestartPersistedWorkingRecoversFill` | FINDING 4(c): a persisted working row survives the restart, stays working with no new evidence (restart limitation explicit), and settles when post-restart execution evidence is received — no additional entry |
| `TestPictureHtfReconcileAbsentBookStaysUnknownNeverResends` | with an explicitly empty wire book and no received history the row STAYS place_pending, marked `outcome unknown` exactly once, fill price 0 — a filled order is absent from the snapshot exactly like a never-placed one, so absence proves nothing |
| `TestPictureHtfReconcileWorkingBookLeavesFillPriceUnknown` | a present working book entry is a working receipt; its `LimitPrice` must NEVER become the fill price (the snapshot carries no fill price) |
| `TestPictureConsumerAndArmedStreamCoexistOnOneSubscription` | the picture consumer and the armed listener share ONE subscription through the real router: every fed frame reaches BOTH, neither channel is closed by the other |
| `TestOrderUpdateFanoutCoexistsNoEviction` + `TestOrderUpdateFanoutPerAccountIsolation` (`provider/ninjatrader/order_update_fanout_test.go`) | coordinated dispatch: listeners join/leave without evicting anyone; a legacy direct subscribe closes the fan-out source → listeners close (the consumers' self-heal signal); per-account isolation |
| `TestPictureHtfAmbiguousSendStaysPending` (evaluator suite) | the seam reports "send ambiguous" and re-entry is refused at the store |
| `TestPictureHtfRestartAfterClaimSingleSubmission` (evaluator suite) | restart after claim produces exactly one submission |
| `TestPictureHtfClaimSubmissionExactlyOneWinner` (store suite) | atomic claim ownership — 12 concurrent claimers, one winner |

**Wiring:** one coordinated order_update fan-out per (symbol, account) owns
the single direct subscription (`TCPServer.ListenOrderUpdates`); the armed
executor (`armedUpdateStream`) and the picture consumer
(`ensurePictureHtfBrokerConsumer`) are both LISTENERS — neither can evict the
other, and each self-heals if the underlying subscription dies. The consumer
records every entry-leg frame into a received-history (process lifetime)
even when no row matches yet; `pictureHtfReconcilePending` runs each cycle
and recovers from that history, then the received-fill ring, then the
working-order book. An outcome never received stays UNKNOWN.

## 6c. CTO round-2 corrections (all three findings fixed)

1. **Reconciliation tests supplied broker states the actual snapshot
   excludes.** The wave-18 tests injected Filled/Rejected orders into the
   snapshot lookup; the AddOn's snapshot skips terminal orders. FIXED: the
   reconciler no longer reads terminal outcomes from the snapshot. Terminal
   recovery now comes from received execution/order history only — the
   consumer-populated order_update history (process lifetime) and the
   TCPTrader received-fill ring. Pins:
   RecoversFillFromReceivedHistory / RecoversRejectionFromReceivedHistory /
   RecoversFillFromFillStream / AbsentBookStaysUnknownNeverResends.
2. **Reconciliation used `order.LimitPrice` as the fill price.** FIXED:
   `LimitPrice` is never a fill. Recovered fill prices come ONLY from the
   order_update frame's `fill_price` (AverageFillPrice) or the fill frame.
   A terminal outcome with no received price is terminal with the fill price
   left UNKNOWN — never fabricated. Pin: WorkingBookLeavesFillPriceUnknown +
   the "fill price unknown" branch of ApplyTerminal.
3. **The new consumer could replace the existing order subscription.**
   `SubscribeOrderUpdatesFor` REPLACES the (symbol, account) channel, so two
   direct subscribers close each other. FIXED: `TCPServer.ListenOrderUpdates`
   coordinated fan-out — one direct subscription per key, listeners coexist,
   and both consumers (armed executor + picture) are now listeners. Pins:
   TestOrderUpdateFanoutCoexistsNoEviction / PerAccountIsolation (provider)
   + TestPictureConsumerAndArmedStreamCoexistOnOneSubscription (trader,
   through the real router).

The prior wave-18 pin names
(`TestPictureHtfReconcileRecoversFilledAcrossRestart`,
`TestPictureHtfReconcileRejectedAndAbsent`) are REMOVED — their fixtures
injected wire-impossible states.

### Round 3 (finding 4, 2026-09-20) — working evidence must not mask execution evidence

4. **An older working receipt can hide a received fill.**
   `pictureHtfReconcilePending` read order-update history first and `continue`d
   after a working/accepted receipt, before the fill ring was ever consulted;
   and `PictureHtfPendingByTrader` only swept `place_pending`, so rows already
   marked working had the same coverage gap. FIXED: the sweep now scans
   `PictureHtfRecoverableByTrader` (place_pending AND working — a working
   receipt is not a terminal outcome), the working branch falls THROUGH to
   the fill ring instead of exiting, and a place_pending row with both older
   working history and newer fill evidence settles to the fill. Quantity and
   terminal-state correctness preserved (fill price/qty from the wire, R:R
   from the row geometry, re-entry blocked by the atomic claim). Pins:
   WorkingReceiptThenFillFrameRecovers / PendingPrefersFillOverOlderWorkingHistory /
   RestartPersistedWorkingRecoversFill. The restart limitation stays explicit:
   process-local history disappears on restart; unreceived outcomes remain
   UNKNOWN — this is duplicate prevention, not complete broker-history recovery.

## 7. Gap register — classified (per CTO request, 2026-09-20)

Each activation-relevant gap carries exactly one status:
**missing** = no implementation · **implemented, untested** = code exists,
evidence is unit/loopback only · **owner/runtime** = code and local evidence
complete; only the owner (or the live NT8 runtime) can produce the remaining
evidence.

| # | Gap | Status | Evidence | Remaining owner/runtime step |
|---|---|---|---|---|
| 1 | Broker-state consumer (received entry / rejection / fill / protective-order events update the correct opportunity) | **implemented, pinned (unit + real subscription)** | `trader/picture_htf_broker.go` consumer, a coordinated fan-out listener per trader; pins: EntryLifecycle (forward-only, wire fill price + actual-fill R:R), RejectionCarriesReason, ConsumeProtectiveLegsUpdateProtectionNotFill (append, never clobber), Isolation, CoexistOnOneSubscription (real router — the armed listener receives every frame too) — all green (`/tmp/gofinal8.log`, exit 0, 35/35) | receive a real order_update set on NT8 SIM (owner's copy → F5 → restart) |
| 2 | Reconciliation sweep (pending/ambiguous submissions recover across disconnects/restarts without another entry) | **implemented, pinned (unit + real paths)** | sweep runs every cycle over place_pending AND working rows from RECEIVED evidence only, execution evidence outranking working receipts: consumer-populated order_update history → received-fill ring → working-order book (presence only; absence proves nothing); pins: RecoversFillFromReceivedHistory, RecoversRejectionFromReceivedHistory, RecoversFillFromFillStream, WorkingReceiptThenFillFrameRecovers, PendingPrefersFillOverOlderWorkingHistory, RestartPersistedWorkingRecoversFill, AbsentBookStaysUnknownNeverResends, WorkingBookLeavesFillPriceUnknown — green. Restart limitation explicit: process-local history is gone after a restart; unreceived outcomes stay UNKNOWN (duplicate prevention, not broker-history recovery) | live NT8 snapshot + order_update sets on SIM (owner step above) |
| 3 | Partial fills (filled quantity receives protection through the actual AddOn path) | **implemented in the AddOn, untested live** | C# audit [A]: VLTraderTCPClient.cs:1445 `SubmitBracketOnEntryFill(signalId, e.Filled, e.AverageFillPrice)`; :2089+ refuses filledQty ≤ 0, sizes SL/TP legs by filledQty, `AmendBracketQuantity` on later fills — protective orders follow the FILLED quantity | NT8 partial-fill scenario (limit-touch) — owner/runtime |
| 4 | Native 4h | **live input available; historical storage absent (replay limitation, NOT a live blocker) — but subscription config AND DB storage are not data readiness** | live: `defaultAutoBarsTimeframes` (`provider/ninjatrader/tcp_server.go:518`) includes `"4h"`, so the live native 4H subscription exists [B]; storage: no native-4h rows in the DB, replay uses the disclosed ETH-grid proxy. CTO ruling: readiness verification runs BEFORE the mode is enabled (runbook Step 4, now ahead of the toggle): the actual native bars the EVALUATOR consumes — symbol, resolved contract, final/emitted_at completion evidence, ≥ pivot-window completed-4h depth — with DB storage as corroboration only | activation gate (pre-enable): evaluator-consumed native 4h bars verified; importing native 4h history remains a replay-quality choice |
| 5 | AddOn build/capability receipt | **owner/runtime** | wire pins green both sides; VL_BUILD_ID="2026-09-20-p1" floor on both ends | owner: copy → F5 → full NT8 restart → boot line receipt |
| 6 | Merged-HEAD/release checks + attended cutover | **owner/runtime (held)** | runbook: `docs/superpowers/runbooks/2026-09-20-picture-htf-activation.md`; merged-HEAD full suite required by canon before release | owner present for NT8 steps; owner's explicit "go" |
| 7 | Natural-market opportunity (category D) | **pending, separate** | none occurred; categories A–C pinned; the decision body itself is proven by the replay body (Sept-17 tape) | a live category-D opportunity when it occurs |

Cutover remains **HELD** — no activation claim is made; the honest terminal
state of this wave is *implemented, verified to the limit of what the loopback
can prove, awaiting owner/runtime evidence and CTO review*.

## 8. Rule defaults

`enabled=off` · `tick_size=0.25` · `pivot_window=120` · `swing_lookback=24` ·
`entry_window_sec=10` · `freshness_sec=2` · `min_rr` inherits the saved
strategy minimum (live bound strategy: **2.0**). Owner deviations: none
configured. Engineering defaults are agreed initial choices, not proven
optimums; a faithful implementation does not establish profitability.

## 9. Report pinning

- Wave-18 code commit: `de2527918f730e89871402c70350f6baa9a14d1b` (first
  consumer/reconciler pass — SUPERSEDED by the round-2 corrections below).
- Wave-19 code commit: `c2ba4388bcee1bb29809c57bb196bfa306fd55d2` — the CTO's
  three production-integration findings fixed (received-evidence recovery,
  fill-price honesty, coordinated dispatch), the fixes this §6b/§6c state.
- Wave-20 code commit: `9a1044df2b0ec4a1bdf453599148d55e4ae86095` — finding 4
  (working evidence masking execution evidence) fixed: the sweep scans
  place_pending AND working rows and execution evidence outranks working
  receipts.
- This report lives on the same branch; the raw URL is pinned to the full
  commit SHA with the byte count quoted in the dispatch reply, and the pinned
  blob was curl-fetched and byte-compared against the working tree.
