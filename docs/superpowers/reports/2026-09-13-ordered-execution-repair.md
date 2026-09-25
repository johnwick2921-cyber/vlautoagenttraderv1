# Ordered execution evidence: source repair and limits

Source checkpoint: `9b379c8c2ec53968cdabb1ef5071a6e3a8eddc51`, following atomic receipts in a982cc74 and pending/first-snapshot correction c20d0a82. This is offline source/test evidence. No running service, owner account/configuration or installed AddOn was changed.

## Reproduction and correction

[A] An actual entry callback materialized one contract at100; a cumulative two-contract entry at105 was already queued. The old close consumer independently applied one exit at120, closing the row with P&L40. The cycle later drained the queued entry and left no residual row. Expected receive-order accounting was one residual contract at105 and realized P&L30. Before: `/tmp/nofx-combined-review/deeper-ordering-repro-a982cc74.log` and its preserved test/overlay.

[A source/tests] A single exact account/symbol owner now receives OrderUpdate, Fill and PositionClose directly from TCP readLoop, before advisory fanout can reorder them. Echo/pending-operation checks stay at the transport boundary. Internal handled flags are excluded from JSON; untrusted input cannot assert prior handling. Existing normal, synchronous cancellation, fill-cache and close consumers skip already applied advisory events. Registration occurs only after successful trader construction; replacement and cleanup compare actual owner identity. Ordinary Stop retains observation.

[A] Real TCP tests cover both orders:

| Received observations | Residual after first exit | Realized P&L after first exit |
| --- | --- | --- |
| Entry1@100, cumulativeEntry2@105, exit1@120 |1 contract at105 |30USD |
| Entry1@100, exit1@120, cumulativeEntry2@105 |1 contract at110 |40USD |

The second path records newly observed cumulative exposure from the same immutable order after its earlier residual reached zero. It does not submit or authorize another order. Known prior notional, greater cumulative quantity, exact account/symbol/entry identity and absence of competing open exposure are required in the conditional database update. Equal/older replay cannot reopen the row. Earlier realized P&L and exit receipts remain, while final corrected P&L becomes NULL during the continuing exposure. Both examples finish at80USD total realized P&L after a final one-contract manual exit at130, using MNQ2USD/point.

[A] Full raw replay, marked advisory replay, foreign account, obsolete-owner cleanup, unrelated bracket lineage and fill-cache resurrection are tested. The cache rejects equal/older cumulative fills for an exact fully exited entry. A callback sends a real FrameSignal back over the loopback TCP connection to test reentrant outbound progress. Focused race evidence: `/tmp/nofx-combined-review/ordered-final-06.log`; independent source/test review: `/tmp/nofx-ordered-execution-independent-review.md`. Final full combined verification is recorded separately.

## Explicit limits

The guarantee is TCP receive order for a registered owner, not independently verified exchange execution chronology. OrderUpdate lacks execution timestamps sufficient to reconstruct that chronology. Initial frames delivered to legacy queues before any owner existed are not retroactively ordered. Database callbacks execute synchronously in readLoop: slow storage backpressures bars, positions and heartbeat reception; callbacks must never wait synchronously for broker replies. There is no durable inbound transport journal. A transient database failure still requires later evidence/replay; a committed database transaction cannot guarantee subsequent process-local hook delivery across a crash.

A same-order continuation does not retroactively rebuild excursion rows or previously emitted terminal analytics. Do not claim that every historical statistic becomes exact from this accounting repair. Final live NT8 scheduling, broker OCO behavior, wire reconnect/replay and controlled shutdown need a separately authorized SIM rehearsal. Source tests are not profitability evidence.

Follow-up repairs d7b70a90 and cd2978b7 preserve positive-filled rejected entry evidence, reject pre-entry snapshots as proof of flat, and retain account/symbol/signal cumulative receipt fences across same-server adapter replacement. Duplicate or older execution evidence does not renew the fence after a genuinely newer snapshot. Both replacement failures were reproduced before repair, focused race checks and independent review passed, and combined verification is in the final audit ledger. These are repaired source boundaries; the receive-order, durable-journal and installed-NT8 limits above remain.
