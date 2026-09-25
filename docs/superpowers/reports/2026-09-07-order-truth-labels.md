# ORDER-TRUTH — LABEL wave

Implementation: `1064dca0e734d4caca8450533c64756b8331abbc`, branch `fix/order-truth-labels-0907`. This report describes tested source; it does not assert a production deployment.

Spec freshness: `738ea9b72925f0061a3f16a2cb5818f4ab86d153 | 2026-09-07T21:21:45-05:00 | docs: complete chart provenance and final order-truth verification`, latest change to `2026-09-07-order-truth.md` at acceptance. Verification follows `docs/superpowers/AUDIT-CHECKLIST.md`; existing classes 6, 49 and 53 apply.

[A] BOOK now dates the atomically captured broker snapshot by Go receipt, displays its actual age and received AddOn build, and retains that provenance even when stale or the ledger comparison fails. Missing broker receipt remains UNKNOWN instead of dating ledger flatness as now. Snapshot emission time is not substituted for receipt. The existing leg-4 formatter runs against a request-local frozen snapshot and the caller clock; no trading gate or cancellation behavior changes.

[A] MODE and FEED explicitly say UNKNOWN when no feed_status has been received, even when bars are fresh. Known component facts remain visible beside the missing link state. Scenario activation/confirmation is visibly advisory and separate from the ledger order authorization chip. The guide explains both distinctions.

[A] Validation: full `go test ./...` PASS; 47 web test files / 362 tests PASS; `npm run build` PASS; precommit eslint/prettier and diff whitespace checks PASS. Tests cover fresh/stale broker receipt, misleading emission time, missing snapshot/link, rendered provenance, and activation without an authorized order. The full web suite initially exposed three unhandled network errors in the existing W19 planner tests; isolated reproduction confirmed an unmocked getPlanThread request. A test-only mock removes that leak. No runtime planner behavior changed.

## F2 receipt check, 21:57 CT

[A] Owner reported compile/restart. Latest hello receipt from the system journal: 2026-09-07T21:44:22.179663-05:00, protocol_version=3, source=vltrader-addon, build_id=2026-09-05-g2. Stored order_snapshot 10895: emitted 2026-09-07T21:57:22.315-05:00, received 2026-09-07T21:57:22.257-05:00, build_id=2026-09-05-g2, reason=periodic, orders=[]. h1 match=NO.

[A] At 21:56:16 CT the authenticated gate reported leg 4 source `broker — NT8 order_snapshot frame (age 23s, build 2026-09-05-g2)`, pass=true, zero broker working orders and zero ledger orders. This is sampled account-scoped flatness, not proof of the four h1 behaviors.

[A] NT8 trace confirms UserDataDir `C:\Users\hoang\Documents\NinjaTrader 8\`. The actual AddOns/VLTraderTCPClient.cs line 55 still declares g2; SHA256 cf2b76b4f67fefc5b74a64784706d3826d131e616efcd2419926e67409af2e18, 140520 bytes. NinjaTrader.Custom.dll modified 21:43:00.582678 CT embeds g2; SHA256 f5c48beb90c4188bbea8e24a38c66baab98c482d29a07f6c17c88853906fef25. No compile-error matches in the September 7 native log/trace files. h1 is not staged in the directory used by this compile. This wave has not copied source, restarted NT8, or issued any trading command.

The separate DATA-2 wave and h1 receipt/entry-OCO/child-OCO/Gtc/entry-only-cancel wire proofs remain pending.
