# Netting-Orphan Wave (class 27) — 2026-08-31

**Rev:** `a0c7ff0b118769cabfb39c03d965637190461768` (marker `a0c7ff0b`)
**Cutover:** 2026-08-31 14:49:56 CT · PID 1340498 · `🔐 BOOT INTEGRITY OK — rev a0c7ff0b1187 · goldens PASS`
**Rollback kept:** `nofx-bin.prev.boot` = `2bc58ed9a4ace014236461bc06bc504284f0d4c5` (lead-time wave)
**Flat-gate at cutover:** DB open rows = 0 · NT8 `positions snapshot account=Sim101 count=0` · no live armed orders
**Trigger:** live forensics 2026-08-31 (owner screenshot): the S1 LONG @29413 was netted flat by the S3 SellShort @29459 at 13:09:16 — no `position_close` frame exists for a netting close — and 26 minutes later the orphaned S1 stop fired while flat and opened a naked short @29417.25 (closed +$39.50 by an orphaned TP at 13:38:19). The long's real +$92.00 never entered the ledger.

## Fixes shipped

| # | Fix | Where | Gate-level fixture / proof |
|---|---|---|---|
| 1 (CRITICAL) | Netting-flat cancels brackets | **C#** `VLTraderTCPClient.OnPositionUpdate`: on `MarketPosition.Flat`, sweep-cancel EVERY tracked bracket for (account, instrument) via new `CancelAllBracketsFor`. **Go** `trader/position_desync.go`: on store-OPEN vs broker-FLAT, send `cancel_order` for each open row's bracket key immediately (no 60s grace) | `TestDesyncSendsImmediateCancelOrder` (loopback wire: cancel_order frame carries the row's bracket key); `TestDesyncNoCancelWhenBrokerHolds` |
| 2 (HIGH) | Reconstruct the real exit | `trader/ninjatrader/netting_fills.go`: bounded fill ring on the TCPTrader; reconcile orphan-close consumes the LATEST opposite-side fill in the flat window as the exit (close_reason `sync`, real ×pv P&L). **NEVER exit=entry** — no evidence → `ClosePositionUnresolved` (close_reason `unresolved`, note, `ERROR` log + `exit_unresolved` telemetry) | `TestReconcileReconstructsNettingExitReplayToday` — **replay of today's 12:25–13:09 stream closes the S1 row at 29459 → +$92.00**; `TestReconcileUnresolvedWhenNoEvidence` (unresolved + excluded from stats); `TestTakeNettingExit*` |
| 3 (MEDIUM) | Dedupe 577+578 class | Root cause: the armed materializer wrote lowercase `side`; every lookup queries uppercase → reconcile's untracked path missed it and materialized a duplicate. `materializeArmedEntry` now writes UPPERCASE canonical side + case-insensitive dedupe; reconcile retries the owner lookup account-agnostically and backfills the bound account (so close-sync finds the owner) | `TestMaterializeArmedEntryDedupesCaseInsensitive`; `TestReconcileUntrackedDedupesAccountEmptyRow` |
| 4 (owner-added) | ONE-LIVE-ARM GUARD | `trader/armed_executor.go`: while a position is open, an opposite-side arm leg is refused at write AND any already-resting opposite-side order for that leg is cancelled the same cycle. Escape: a leg explicitly authored as an exit/flip leg (`kind: "exit"`). On a 1-lot netting account a second arm is not a second trade — it is an unlogged exit of the first | `TestOneLiveArmGuardRefusesOppositeSide`; `TestOneLiveArmGuardExitLegAndFlatPass` |
| 5 (owner-added) | Split-leg sanity | Leg capacity = explicit `max_contracts_per_order`, else **1** (netting-safe default). `leg_count > capacity` → arm REFUSED at write (`split_leg_capacity`). The E4 split feature remains authorable only when capacity ≥ 2 is declared | `TestSplitLegCapacity`, `TestArmLegCapacityUsesStrategyValue`; `TestSplitArmWritesTwoLedgerRows` updated to declare capacity 2 |
| — | Unknown-P&L surface | New `CloseReasonUnresolved` + `UnknownPnLReason`; all P&L sums/streaks/guardrails exclude it alongside `reconcile_flat` (8 SQL sites); UI renders both as "—" | store + `PositionHistory.tsx` edits; existing suites |

**Deferred (log only, next wave):** #4 orphan lineage, #5 ratchet-ack logging, #6 working-order snapshot frame.

## Verdict / evidence table

| Finding | Evidence | Severity | Fix |
|---|---|---|---|
| Netting close leaves brackets orphaned → orphan SL fired 26 min later and opened a naked short | NT8 trace: `f0bbe9af-…-sl … Filled 13:35:47.310 price=29417.25` while `positions count=0` | CRITICAL | FIX 1 (C# sweep + Go desync cancel) |
| S1 long's +$92.00 exit never recorded (reconcile wrote exit=entry, pnl 0) | rows 577/578 `realized_pnl=0.0` vs NT8 equity 52052→52216 (+$164) | HIGH | FIX 2 (netting-fill reconstruction; never exit=entry) |
| Duplicate rows for one NT8 position | 577 (`long`, armed_entry) + 578 (`LONG`, reconcile) — lowercase-vs-uppercase side lookup miss | MEDIUM | FIX 3 |
| Second arm nets the first — unlogged exit by design | S3 SellShort fill → NT8 `Long 1 → Add Short → Flat Remove`, no close frame | CRITICAL | FIX 4 |
| S3 authored 2 legs on a 1-lot account | armed_orders v6 `leg_count=2`; every live order sizes to 1 contract | HIGH | FIX 5 |

## Recomputed ledger for today (must equal +$164.00)

| Trade | NT8 truth | Store after this wave (replay-verified) |
|---|---|---|
| S2 short 29437 → 29420.75 | +$32.50 | +$32.50 ✓ |
| **S1 long 29413 → netted 29459** | **+$92.00** | **+$92.00** (reconstructed from the netting fill — `TestReconcileReconstructsNettingExitReplayToday`) |
| Orphan short 29417.25 → 29397.5 | +$39.50 | +$39.50 ✓ |
| S3 short entry (netted, no position) | — | `unresolved` (visible gap, excluded from sums) |
| **Total** | **+$164.00** | **+$164.00** ✓ (matches NT8 equity 52052.00 → 52216.00) |

The live DB's historical rows (577/578) keep their placeholder values — they predate this wave; the replay test proves the new path would have recorded +$92.00, and future netting closes land correctly.

## Deployment state

- Go binary: rev `a0c7ff0b` live (PID 1340498, boot 14:49:56 CT, goldens PASS).
- C# AddOn: **staged** at `C:\Users\hoang\Documents\NinjaTrader 8\bin\Custom\AddOns\` (sweep code present). **Pending owner F5-compile + full NT8 restart** — until then the old AddOn runs; the Go side is fully backward-compatible (`cancel_order` is an existing protocol frame), and the Go desync-cancel (FIX 1 Go half) is already live.
- Tests: full Go suite green; web vitest 287/287; tsc clean.
- Rollback: `mv nofx-bin.prev.boot nofx-bin` + revert `deploy/RELEASE` + `kill -9 1340498`.

## Notes for the owner

1. **Restart NT8 when convenient** (F5 already compiled copy is in place; full restart required) to activate the C# netting-flat bracket sweep.
2. **Pre-existing guardrail state (not this wave):** `max daily trades would trip (today=10, max=3)` — new entries are currently blocked by the daily-trades guardrail; adjust `max_daily_trades` if that is not intended.
3. Split (2-leg) arms are now refused until `max_contracts_per_order ≥ 2` is set on the strategy (owner intent: 1-lot account).
