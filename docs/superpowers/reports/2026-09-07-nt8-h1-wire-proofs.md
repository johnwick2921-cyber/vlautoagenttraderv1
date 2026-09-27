# NT8 h1 — receipt and first-arm wire-proof watch

**Build receipt confirmed; the four new-arm behavior proofs remain pending.** The owner performed copy, F5 compile and full NT8 restart. This lane only reads frames/logs and writes this report; it has sent no trading command and changed no AddOn or runtime.

## The adopted protections were DAY, not GTC

**Owner warning:** the two adopted orders were placed under g2, and both still carry `tif=Day` in h1 frame10947. They are not GTC. Day orders expire at the trading-session close and must not be expected to protect beyond that close if still working. Restarting into h1 did not change these existing orders' TIF. [NinjaTrader TimeInForce documentation](https://docs.ninjatrader.com/ninjascript/timeinforce) distinguishes Day expiry at session end from Gtc persistence until cancellation. The exact session-close instant was not established by this watch.

[A] Both adopted orders share signal `953e8986-3496-45ae-a78d-ebd6d863177e` and OCO `953e8986-3496-45ae-a78d-ebd6d863177e-exit`:

| Order ID | Leg | Price | State in frame10947 | TIF |
|---|---|---:|---|---|
| 373762238d2d4c8f8ac5d844d173579d | stop | 29640 | Accepted | Day |
| a03774c695d441989be01aae91e2adf0 | target | 29721.25 | Working | Day |

[A] These are now historical, not a current unprotected-open-position claim: native log `log.20260907.00005.txt:350` records target Filled at **22:28:53.343 CT**, :351 records stop cancel submitted, and :355 records stop Cancelled at **22:28:53.468 CT**. Received h1 snapshots10968/10969/10970 show remaining stop Accepted/CancelPending/CancelSubmitted; frame10971 received **22:28:53.391 CT**, emitted **22:28:53.469 CT**, has `orders: []`. Independent emission and receipt clocks are quoted as recorded. At22:59:07 CT API, DB and NT8 position counts were all zero.

## Received build and leg 4

[A] System journal receipt **2026-09-07T22:16:54.890994-05:00**:

```text
hello handshake OK protocol_version=3 source=vltrader-addon build_id=2026-09-07-h1
```

[A] Frame10947, emitted **22:18:25.052 CT**, received **22:18:25.419 CT**, carries `build_id=2026-09-07-h1`; expected h1 **match=yes**. Full account-redacted projected frame bodies and source row IDs are retained in [the frame evidence](2026-09-07-nt8-h1-wire-frames.json). No absent field was invented.

[A] At22:17:59 CT, live leg 4 source was `broker — NT8 order_snapshot frame (age 4s, build 2026-09-07-h1) × armed_orders ledger`, pass=false, broker2 versus ledger0 (the adopted protections). At22:59:07 CT the source was the same broker/ledger comparison with age12s, pass=false, now broker0 versus ledger1. A received h1 build does not mean a passing flatness gate.

## First h1 attempts: rejected before broker acceptance

[A] Arm116, S3, v3, placement0, signal `747dc100-da23-47a9-b683-e7a68d926adf`: Go logged WORKING at22:28:55.341896 CT, then received a rejected fill (seq2) at22:28:55.346602 CT. Native log `log.20260907.00005.txt:366` says at22:28:55.344 CT:

```text
VLTraderTCPClient: stale signal 747dc100-da23-47a9-b683-e7a68d926adf (age 715.3s) — rejecting
```

[A] Arm117, S3, v3, placement1, signal `9ba63cb5-bb72-454f-8989-a68632530560`: Go logged WORKING at22:47:24.527319 CT, then received a rejected fill (seq4) at22:47:24.527998 CT. Native log :555 says at22:47:24.528 CT:

```text
VLTraderTCPClient: stale signal 9ba63cb5-bb72-454f-8989-a68632530560 (age 1824.5s) — rejecting
```

No received snapshot through11031 (22:58:25.189 CT) contains either signal. Arm116 later reconciled to cancelled; arm117 still said working at22:59 despite its received rejection and the empty broker book. These are placement attempts, not accepted orders.

[A] The native log at22:45:25.181 CT (:528) says cancel_order found NO resting entry for arm116 and cancelled nothing. This does not prove cancellation of an actual unfilled broker entry while preserving a standing bracket: neither condition was present.

| Requested proof on new h1 activity | Status |
|---|---|
| Entry has independent OCO (empty entry OCO is the source contract) | PENDING — both attempts rejected; no accepted entry frame |
| After fill, stop and target share a different OCO | PENDING — adopted g2 children do not establish h1 placement behavior |
| Protective legs carry Gtc | PENDING — only adopted Day legs observed |
| Cancelling an unfilled entry leaves the bracket untouched | PENDING — cancel found no entry, and the prior bracket had already settled |

A single placement cannot both fill and remain unfilled for cancellation. The cancellation proof requires a suitable subsequent unfilled entry with a standing bracket to observe; the watch will not manufacture that condition with a trading command.

## Watch update — 23:04 CT

[A] Arm117 subsequently reconciled to cancelled at23:03:24.050 CT with reason `absent from a fresh NT8 order_snapshot (reconciled to the broker)`. Snapshot11041 received23:03:25.307 CT remains an empty h1 book. This is settlement of a misleading ledger state, not proof of cancelling a resting broker entry.

[A] Authenticated live FEED at23:04:24.289 CT: `last bar 48m24s ago · link Disconnected · AddOn build 2026-09-07-h1`, state=stale, verified=false, newest1m bar timestamp22:16:00 CT. BOOK at the same sample says zero working orders with ledger agreement and a29s-old h1 snapshot. Broker-book liveness and market-feed liveness are different facts.

[B] This explains the stale signal rejection pattern: `trader/ninjatrader/tcp_trader.go:274` stamps entry commands using the newest cached bar close; `ninjascript/VLTraderTCPClient.cs:810` rejects an entry timestamp older than60s against the Windows clock. The reported signal ages point to a22:17:00 CT stamp. Actual command payload timestamps were not retained in the examined evidence, so that timestamp attribution remains an inference. Fresh broker-accepted h1 order activity has not arrived. The watch remains read-only and does not bypass the stale-signal guard or reconnect anything.

## Provenance and scope

Docs-only branch `docs/nt8-h1-wire-proofs-0907`, claim `df2d3275c8b27ccf36ba2490cd0a29fbdbc01164`. Spec freshness: `40f3c2443b3d84f6d8cef9e671fb523e086423af | 2026-09-07T22:13:56-05:00 | docs: record DATA-2 provenance and owner NT8 copy command`, latest change to the DATA-2 report at acceptance. Verification follows `docs/superpowers/AUDIT-CHECKLIST.md`: received build evidence, snapshot row IDs, no invented timestamps, and no ledger state substituted for a broker fact. No new class number allocated. This report marks each evidence gap explicitly and does not certify the two Go cancel exemptions.
