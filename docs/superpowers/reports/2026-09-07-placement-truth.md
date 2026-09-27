# Placement truth — final combined wave

Updated 2026-09-07 23:58 CT. **[A] PROVEN in isolated tests; candidate NOT DEPLOYED.**
Implementation: `98f4ec6e499eb7dafb31e192460cb774c9a425df`.
Branch: `fix/placement-truth-0907`; locked worktree:
`/tmp/nofx-placement-truth-0907`.

This dispatch performed no production DB writes, runtime restart, Windows file
copy, NT8 compile/restart, or live order. A different dispatch deployed its own
wave during integration; its marker is preserved, not confused with this one.

## Provenance and overlapping work

The branch was cut from acceptance-time `origin/dev`
`e762476fd6e1b1791ebaa6a1dc3207b8bcbdeda9`. Claim `7b158c6e` was published before
implementation. Initial implementation commits were `6262bf42` and `7e0c5527`.

Source freshness at acceptance (`git log -1 <base> -- <path>`):

- Protocol: `c84bd247 feat(F12): the AddOn emits its order book; leg 4 asks the broker; build_id is proven by receipt`.
- Audit checklist: `7ddf73cd docs(report+checklist): session calendar report and class 88 — a liveness signal that is a side effect of activity`.
- Timing audit: `e762476f docs: pin timing audit references and preserve concurrent F2 evidence`.

The latest owner ruling controls this wave: command creation clock; all four
placements registered as `place_pending`; received-only promotion; verbatim
optional Go rejection reason now; C# producer deferred. Checklist R1–R10,
class 81, and production-call-site pins apply. Class 81 now includes this
placement recurrence; no new class number was allocated.

While this branch was being validated, `317388e7` landed a second placement
confirmation implementation. Merge `6310eaf8` preserves its fresh-book
confirmation, named confirmation evidence, rejection-sink wiring, chart
filtering/price fix, and boot contract, with these resolutions:

- Registration happens BEFORE the wire on all four paths. No post-send write
  can erase an immediate rejection.
- One fill consumer calls one rejection sink. Standalone TCPTrader use falls
  back to the same scoped store transition, never a second subscriber.
- Book promotion requires the exact ENTRY name and a live state in a fresh
  received snapshot. Children, submitted/local states and stale books cannot
  promote.
- No-frame timeout stays `place_pending`, holds the slot, and records an
  `unconfirmed:no_frame` warning. Time alone does not settle a broker state.
- The incoming C# patch remains available at `172c4e1b` for the next AddOn wave,
  but was deferred from the combined tree, preserving h1 byte-for-byte. It
  needs repair: `ageSec` is referenced in the missing-limit-price branch before
  its declaration, and an `if` statement is inserted inside a dictionary
  initializer. No NT8 compile was attempted or claimed here.

Then `171bebee` recorded the other dispatch's boot at `317388e7`. Merge
`98f4ec6e` retains its `deploy/RELEASE=317388e7`. That marker is evidence of the
other dispatch's report, not this dispatch's independent boot verification,
and not a claim that this candidate is running.

## Final behavior and source pins

Coordinates below refer to `98f4ec6e`.

**Command clock.** Limit payload timestamp:
`trader/ninjatrader/tcp_trader.go:467`; market entry `:398`; stop entry `:544`.
All use UTC command creation time with millisecond precision. The cached bar
close remains a market fact. This does not refresh market data or change the
existing market-data/entry gates.

**Payload age.** `provider/ninjatrader/tcp_server.go:2147` checks the actual
payload timestamp before enqueue (`:1057`) and again after acquiring the
writer, immediately before writing (`:2181`). Older-than-60s, missing,
malformed and future clocks are refused with signal ID and measured
age/threshold, or age unavailable. Retries preserve the original timestamp.
A fresh row or queue timestamp cannot conceal an old payload clock.

**Identity before send.** `store/armed_orders.go:391` atomically registers signal
ID and `place_pending`, conditional on an unsent `armed` row. Failure prevents
sending. All four call sites use it before the wire:

| Path | Registration in trader/armed_executor.go |
| --- | --- |
| Normal limit | :1067 |
| Normal stop | :1367 |
| Debug limit | :2176 |
| Debug stop | :2233 |

There is no post-send `working` write. Debug rows are created before sending
and read back after the call, preserving a reply that beats its return. Pending
rows are nonterminal, cannot be rewritten/minted over, and participate in boot
sweep/cancel eligibility and split-leg handling.

**Received settlement.** Entry rejection updates at `armed_executor.go:1707`;
received live entry updates promote at `:1745`. h1's pre-submit rejection is
instead a `fill.status=rejected` frame, routed at `tcp_trader.go:227` through
`:1284`. Both settle through `store/armed_orders.go:410`. Fresh broker books
confirm through `trader/place_confirm.go:21` / `store/armed_orders.go:620`.
Late acceptance cannot resurrect rejection or erase cancellation intent.
Protective-leg updates cannot settle the entry.

**Reasons.** `provider/ninjatrader/tcp_framing.go:74` and `:161` accept optional
`reason` on fill/order_update. Nonblank supplied text is stored verbatim,
including surrounding spaces. Missing/blank becomes
`reason unavailable (NT8 frame omitted reason)`. A subsequent received reason
can fill that absence without replacing an already received reason. Logs and
the UI expose this distinction; no generic `NT8 reject` is stored.

**Display.** The card distinguishes authorized, placement pending, received
working and rejected with reason. Unconfirmed ledger entries do not draw broker
order lines. Limit entry lines use the entry price, not their protective stop.
Guide content and a marker for the final candidate accompany this report.

## Validation

**[A] Old-bar pin, actual composer and real loopback receiver:** a 1-minute bar's
close was 30 minutes old (open 31 minutes old), and its prices were supplied to
the actual entry composer. The receiver independently subtracted the emitted
payload timestamp from receipt time, as h1 does. Final combined-source run:
`/tmp/placement-combined-clock.log`.

| Entry | Reason fixture | Payload age at receiver |
| --- | --- | --- |
| Limit | omitted | 5.433 ms |
| Limit | verbatim text | 5.651 ms |
| Stop | omitted | 8.905 ms |
| Stop | verbatim text | 8.953 ms |

These are **loopback measurements, not live NT8 receipts**. Arm rows were fresh
in every fixture: checking their timestamps alone would have missed both the
715s and 1,824s incidents. The previous timing audit is context, not fresh live
proof for this candidate.

- `TestFourPlacementPathsWaitForEntryReceipt`: all four production paths send a
  loopback frame and remain pending; child updates do not promote; entry receipt
  promotes; rejection survives later working.
- `TestStopPlacementFastRejectBeforeSendReturns`: deterministic registration →
  received rejection → send return; the rejection survives.
- `TestPlacementBookRequiresLiveEntryAndSilenceKeepsSlot`: real confirmation
  path with empty, child-only, submitted, local, stale and live books.
- Actual inbound fill-frame routing, missing/verbatim reasons, registration
  failure, duplicate placement/re-authorization refusal, cancellation intent,
  reason enrichment, stale payload with fresh queue metadata, and expired retry
  are covered. Existing long/short stop wire twins pass in the full suite.
- Full `go test ./...`: **PASS on final merged `98f4ec6e`**,
  `/tmp/placement-final-merged-go.log`. Also passed on combined `6310eaf8`.
- Targeted `go test -race`: **PASS**, store/provider/TCPTrader/executor and
  combined snapshot/receipt regressions; `/tmp/placement-combined-race.log`.
- Web: **49 files / 365 tests PASS**, `/tmp/placement-combined-web-tests.log`;
  production build PASS. Final guide marker is rebuilt before publication.
- Clean-clone Go build and `git diff --check`: **PASS**.

The race run initially exposed pre-existing lazy subscription-map initialization
at the receipt router. All maps now initialize before dispatch starts
(`provider/ninjatrader/tcp_server.go:232`). The fill listener is installed
synchronously before `NewTCPTrader` returns (`tcp_trader.go:174`), and close-sync
establishes the ledger handle before placement. No sleep conceals that race.

## Candidate and remaining live proof

Undeployed binary: `/tmp/nofx-placement-truth-candidate`, built with
`go build -buildvcs=true` from clean clone `/tmp/nofx-placement-build/nofx`.

```
vcs.revision=98f4ec6e499eb7dafb31e192460cb774c9a425df
vcs.time=2026-09-08T04:55:10Z
vcs.modified=false
sha256=316ed32ba10031d26bbc7d6df5d6d3d6bd3db3e4678fd1f406c6bb94eb086c4b
```

Repo h1 source remains byte-identical: MD5
`d0a604d79163f36557af89edc9f40777`; line 44 still sets
`STALE_SIGNAL_AGE_SECONDS = 60`. C# reason production remains next-wave work.

Unanswered placements retain their pending state and slot. Expired queued
payloads are locally refused/logged; no broker rejection is invented and no
automatic replacement is minted. Existing cancellation policy still governs
unresolved rows; this is not a general cancellation rewrite or automatic
queue-expiry settlement.

**EVENT-WAIT:** watcher snapshot **11145**, received **23:55:26.112 CT**,
`build_id=2026-09-07-h1`, zero orders. Four h1 live wire proofs remain pending:
entry owns its OCO; fill creates stop+target on a different shared OCO;
protective legs carry Gtc; cancelling an unfilled entry leaves the bracket
untouched. Earlier adopted g2 orders were **Day**, not GTC, and their close was
recorded in the earlier watch report. An empty snapshot proves none of the four.
