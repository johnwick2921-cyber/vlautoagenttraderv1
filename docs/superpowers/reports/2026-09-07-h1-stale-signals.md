# h1 stale signals — read-only timing and placement-state audit

**The 715.3s and 1824.5s were ages of a stale bar-derived signal timestamp, not measured time spent in the placement queue.** Both arm rows were created and reached the post-send WORKING log within approximately4ms. Both payload timestamps reconstruct to **2026-09-08T03:17:00Z / September7 22:17:00 CT**, the cached1m bar close. The market feed was disconnected while the order-book connection remained live.

**The measured ordering is also different from “working written after rejection”: both WORKING logs precede their received rejection.** The send path prematurely writes working; the asynchronous rejected-fill handler clears TCPTrader pending/fill caches but never corrects the armed_orders row. No armed placement path waits for acceptance or rejection before its working write.

This audit changes documentation only. The owner’s following-wave ruling is recorded below; it is not implemented or deployed here.

## Timelines: three different clocks

[A] The common authored plan is ASIA v3, `plans` composite key `(plan_id from arm116, version3)`, trigger `level_event`, created **2026-09-08 03:16:50.382317098+00:00 = September7 22:16:50.382317098 CT**. Its S3 arm is short, entry29721.25, authored stop29745.25; the subsequently composed ledger stop is29751.12. Plan authoring, arm-row creation and the signal payload's timestamp are different events.

| Measurement, September7 CT | Arm116 / placement0 | Arm117 / placement1 |
|---|---|---|
| Full signal_id | 747dc100-da23-47a9-b683-e7a68d926adf | 9ba63cb5-bb72-454f-8989-a68632530560 |
| Plan authoring/commit, ASIA v3 | 22:16:50.382317098 | same plan, no new authored plan |
| Arm CreatedAt, recorded at composition | 22:28:55.337713972 | 22:47:24.523659737 |
| Signal timestamp, reconstructed [B] | 22:17:00 / 03:17:00Z | 22:17:00 / 03:17:00Z |
| Post-send WORKING journal timestamp [A] | 22:28:55.341896 | 22:47:24.527319 |
| Creation → post-send log interval | 4.182028ms | 3.659263ms |
| NT8 stale-rejection log [A] | 22:28:55.344 | 22:47:24.528 |
| Age computed/logged by NT8 [A] | 715.3s | 1824.5s |
| Go receipt: rejected fill, matching seq+signal [A] | 22:28:55.346602, seq2 | 22:47:24.527998, seq4 |
| C8 rejection handler journal [A] | 22:28:55.346714 | 22:47:24.528068 |
| Durable error log row | log_events32833 | log_events32975 |
| Later ledger reconciliation | cancelled22:45:25.124694861 | cancelled23:03:24.050043716 |

**Measurement limits:** no retained raw outbound signal body or per-signal socket-write timestamp was found. “Post-send” is the first log after PlaceLimitEntry returned and SetSignal/SetState completed, so it is an upper bound on the completed send call, not an invented wire timestamp. The ~4ms intervals are between two Go-clock observations and include more than the socket write. Native NT8 and Go journal timestamps are quoted on their own clocks, not treated as a calibrated one-way latency measurement. NT8 prints age rounded to one decimal; the reconstructed payload timestamp is supported by the source plus the age arithmetic and the later observed cached bar, not quoted from a retained raw payload.

[A] Native file `C:\Users\hoang\Documents\NinjaTrader 8\log\log.20260907.00005.txt`:

```text
:366  2026-09-07 22:28:55:344 ... VLTraderTCPClient: stale signal 747dc100-da23-47a9-b683-e7a68d926adf (age 715.3s) — rejecting
:555  2026-09-07 22:47:24:528 ... VLTraderTCPClient: stale signal 9ba63cb5-bb72-454f-8989-a68632530560 (age 1824.5s) — rejecting
```

## What actually happened between plan authoring and send

1. [A] At22:17:26.706859 CT the journal explicitly refused S3: `one_open_position: short arm S3 refused — position 593 open (v2 S2 long on MNQ); no adds, no flips`. The existing position, not a queued signal, prevented initial authorization. The guard is `trader/armed_executor.go:693–721`; arm gate call at:462.
2. [A] The existing target filled at22:28:53.343 CT in the native log. Go scheduled a post-exit rescan at22:28:53.277194 on its clock, stating `one full decision cycle in 2s`. Cycle56 started22:28:55.285809; arm116 was composed/created22:28:55.337713972 and logged WORKING22:28:55.341896. There was no minutes-long arm116 queue hold.
3. [A] Entry rejection did not settle the arm ledger. It remained working until the stale-working reconciliation at22:45:25.124694861. The reaper asks the fresh broker book after the resolved quiet-time threshold (`armedWorkingStaleMin`, default15minutes at:123–129), finds the signal absent and writes cancelled (:1581–1611).
4. [A] Cycle66 at22:47:24.480749 created a NEW row/placement117 and a NEW UUID from the same plan. `store/armed_orders.go:269–284` creates placement_seq+1 when a terminal row carries a previous signal ID. This is re-authorization on a later cycle after reconciliation, not a transport retry of signal747dc100. The new send still used the old cached bar clock and was immediately rejected again.
5. [A] At22:47:24.452257, before that cycle, the service had already logged `FEED DOWN: no NT8 bar for 30m24s while CME is OPEN`. At23:04:24.289 the authenticated FEED row read `last bar 48m24s ago · link Disconnected · AddOn build 2026-09-07-h1`, while BOOK was fresh and empty. The normal arm path nevertheless reached its wire call.

[B] Therefore: the plan waited for eligibility, and later cycles re-authorized after a misleading working row was reaped; the two large NT8 ages are caused by timestamp selection from stalled market data. The AI inference call is not between arm creation and placement: `auto_trader_loop.go:432` invokes maybeManageArmedOrders before `kernel.GetFullDecisionWithStrategy` at:550.

## Exact path and mismatched age checks

```text
runCycle / auto_trader_loop.go:432
  → maybeManageArmedOrders / armed_executor.go:189
  → composeArmStop :422; arm gates :462; UpsertArm :617/:658
  → runArmedPlacement :680/:950
  → PlaceLimitEntry :1067
      → uuid.NewString / ninjatrader/tcp_trader.go:452
      → Timestamp: t.feedNowUTC(symbol).Format(time.RFC3339) :463
          → feedNowUTC :274: newest 1m, else 5m cached bar OPEN + interval
          → no age validation; local-clock fallback only if neither cache has bars
      → SendSignal :478
          → assign seq; enqueue with a separate time.Now() / tcp_server.go:1047–1054
          → flushPending :2135
              → compares queue age against cutoff :2150, NOT payload.Timestamp
              → WriteFrame(FrameSignal, sig) :2164
      → returns signalID,nil :481
  → SetSignal :1072; SetState("working", "") :1073
  → WORKING log :1074
NT8 HandleSignal
  → parse payload timestamp / VLTraderTCPClient.cs:757
  → compare DateTime.UtcNow against timestamp :814–815
  → log stale reason; send fill.status=rejected :817–819
Go receive / tcp_server.go:1740
  → subscribed fill handler / ninjatrader/tcp_trader.go:207–227
  → clear pending/cache, log rejection, continue; no armed ledger update
```

[A] Go's queue cutoff is also60s (`provider/ninjatrader/tcp_server.go:46`, resolver:1279), but it starts at SendSignal's `time.Now()`. Thus a just-queued payload stamped22:17 passes the queue-age check much later. If disconnected, flushPending returns nil at:2139–2140 while the entry remains queued, another way a successful call can fail to mean a broker order exists.

[A] A write failure requeues the unsent tail with `timestamp: now` (:2172–2174), resetting that queue-age timestamp. This is a source hazard for the following wave, but **no flush-failed or queue-stale-drop log occurred in the examined22:16–22:48 interval**, and neither signal ID occurred before its same-cycle WORKING log. There is no measured715/1824s transport retry in these two cases.

## h1 staleness threshold, from the loaded source

[A] Loaded file `C:\Users\hoang\Documents\NinjaTrader 8\bin\Custom\AddOns\VLTraderTCPClient.cs`, MD5 **d0a604d79163f36557af89edc9f40777**, line55 build `2026-09-07-h1`:

```csharp
// line 44
private const int STALE_SIGNAL_AGE_SECONDS = 60;
// lines 814–815
var ageSec = (DateTime.UtcNow - sigTime.ToUniversalTime()).TotalSeconds;
if (ageSec > STALE_SIGNAL_AGE_SECONDS)
```

It rejects ages **greater than60 seconds**, not >=60. The age string is formatted with `ToString("F1")` at:818. A malformed timestamp currently bypasses this check when TryParse at:811 fails; the next wave must not treat an unreadable age as fresh.

## Every armed placement write and its rejection handling

[A] Census of tracked production Go call sites (excluding test files; including the opt-in debug seams):

| Placement path | Wire call | Unconditional working write after nil send error | Reads this placement's reject first? |
|---|---|---|---|
| Normal limit | armed_executor.go:1067 | **armed_executor.go:1073** | No |
| Normal stop | armed_executor.go:1368 | **armed_executor.go:1386** | No |
| Debug limit seam | armed_executor.go:2162 | **armed_executor.go:2180** | No |
| Debug stop seam | armed_executor.go:2212 | **armed_executor.go:2230** | No |

The normal loop drains queued `order_update` events at its beginning (:221) and after placement (:1090). That is not a per-placement received-acceptance gate. The stale-signal rejection occurs before NT8 creates an Order, so it sends a **fill rejection**, not an OrderUpdate rejection.

[A] Two separate consumers explain the lost state/reason:

- `trader/ninjatrader/tcp_trader.go:207–227` reads `fill.Status == rejected`, deletes pending entries, clears matching cached fill/lastEntrySignalID, logs the C8 error and continues. It has no armed_orders mutation or durable rejected-state cache.
- `trader/armed_executor.go:1726–1728` can consume an actual `order_update.state=rejected`, but writes **cancelled / NT8 reject**, losing both the distinct rejected state and specific reason. Its live Accepted/Working branch at:1746–1747 records accepted risk; it does not perform a received-state transition to working.

For completeness, “no path anywhere reads rejection” would be too broad: the separate market-order `recordAndConfirmOrder` poller (`trader/auto_trader_decision.go:362–394`) has a REJECTED branch. It is not on these four armed placement paths and does not protect their working writes. TCPTrader.GetOrderStatus has a REJECTED mapping at:1183, but the normal rejected-fill handler clears the matching identity and skips caching that rejected fill; the guard at:1172 returns pending afterward. Thus that mapping does not rescue these rejections either. Market entries also construct the bar-clock timestamp (:394) and return after SendSignal (:417), without waiting for the rejection.

## NT8's reason is not in the h1 rejection frame

[A] h1 logs the exact stale-age reason locally, then calls `SendFillFrame(..., "rejected", symbol: symbol)` at:819. SendFillFrame (:1495–1517) serializes signal_id, fill_price/time, side, quantity, slippage_ticks, status, account, symbol and identity stamp. It has **no reason field**. Go FillPayload (`provider/ninjatrader/tcp_framing.go:73–92`) and OrderUpdatePayload (:159–169) also lack a reason field.

The next wave cannot store NT8's reason verbatim by guessing `NT8 reject` or regenerating the age in Go. It needs an additive wire reason emitted by C# and preserved by Go, with received evidence after the owner's corresponding AddOn compile/restart. Historical h1 reasons are recoverable from the quoted native log only; this read-only audit does not backfill the live ledger.

## Owner ruling for the following wave — recorded, not implemented

> a row reads `placement_pending` until a RECEIVED frame names it working; a rejection writes `rejected` with NT8's reason verbatim; and an arm older than the staleness threshold is never sent — it is re-composed or refused on our side, with the age logged.

Implementation consequences from this evidence:

- Register placement identity and persist placement_pending before a fast response can arrive; a later send-return path must not overwrite a received terminal rejection. Apply received-state transitions by the exact placement identity. Cover both production paths and both debug seams.
- Preserve rejected state and verbatim wire reason. An absent reason on an older AddOn remains explicitly unavailable, not synthesized as a quoted NT8 reason. Empty snapshots alone cannot prove rejection.
- Keep plan authorization time, composition time, market-data timestamp, enqueue time and actual send time distinct. Checking only the fresh row CreatedAt would pass both incidents. Check the actual outgoing signal age at the send boundary, and require fresh inputs for any recomposition; do not merely replace its timestamp with now while retaining stale market facts. Preserve age provenance across transport retries.
- Scope TTL validation to entries; protection management and cancellation must not be accidentally disabled by an entry-age check. Leave the stale-signal guard intact while this is only an audit.

## Read-only scope, freshness and watch

Branch `docs/h1-stale-signals-0907`, claim `b4f2aedaceddbc2f04856f0b75331b5472c38399`, based on dev4b4b1425. Main deployment tree and runtime were not changed. The four examined runtime source files (armed_executor, tcp_trader, tcp_server, h1 C#) are byte-identical to Go source revision5457ac5a; the loaded C# additionally matches the quoted h1 MD5. No trading command, test placement, cancel, connection reset, configuration change, or production DB write was performed. Validation is the source call-site census, read-only SQLite/journal/native-log joins and timestamp arithmetic; no behavior tests were run for a docs-only audit.

Spec freshness (`git log -1`):

- Prior h1 report: `4b4b1425b7a83210cd498389d22a0e0cffb36c66 | 2026-09-07T23:05:10-05:00 | docs: record disconnected market feed blocking h1 wire proofs`.
- armed_executor.go: `78da55a92a2d726ddebab3f4eebf8aa5ab38f195 | 2026-09-07T10:45:00-05:00 | docs+guide: classes 78/79, guide surfaces, and the two over-reached guards reverted`.
- tcp_trader.go and tcp_server.go: `8e6cf957efb14996acf46fd7396aee9af99a78e2 | 2026-09-07T10:37:50-05:00 | feat(protection): D5/D6/D7 — a position without a stop is found, named, and given one`.
- h1 C#: `b4195e6f877032090812214b8ae4b6acae777a4f | 2026-09-07T10:53:37-05:00 | fix(exit): class 80 — a REJECTED limit exit no longer cancels the position's bracket`.

The concurrent F2 report update `9e42bebf` (2026-09-07T23:07:22-05:00, `docs(report): the near-miss the test caught, now confirmed live by rows 116/117`) was read and merged before publication. Its harmless-cancel/reaper evidence agrees with the two reconciliation timestamps above; it does not establish broker acceptance or rejection-state handling. No source file changed in that concurrent update.

The audit follows `docs/superpowers/AUDIT-CHECKLIST.md`; no new class number allocated in this documentation-only wave. The passive wire watch remains active. A third attempt, arm118 placement2/signal2b4c8bda-e8e6-4abf-8970-dd779604554f, was rejected at23:05:23.640 CT with age2903.6s (native log.00005:743); it is further stale-signal evidence, not an h1 acceptance proof. The four requested new-arm wire proofs remain pending.
