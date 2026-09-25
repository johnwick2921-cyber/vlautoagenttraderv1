# WAVE 2 — ARMED-ORDER EXECUTOR · PHASE 0 ORDER-PATH RECON (2026-08-27)

Branch `feat/armed-orders` off merged dev (PR #84 + #85 merged, regression green,
dev Go tree == running rev 99b96b15493e). Read-only recon of the Go⇄NT8 order
bridge. **Verdict: C# delta ≈ 160-180 lines — under the ~300-line gate → proceed
past Phase 0 per dispatch.**

## 0.1 Order types today

Authority: `provider/ninjatrader/tcp_framing.go` + `ninjascript/VLTraderTCPClient.cs`
(the protocol .md is stale by its own admission).

| Capability | Frame | Status |
|---|---|---|
| MARKET entry + deferred SL/TP bracket | `signal` (Go→C#); C# `HandleSignal` L692 submits `Account.CreateOrder(OrderType.Market)` with EMPTY OCO; bracket (StopMarket+Limit OCO group `<signal_id>-exit`) placed on fill via `SubmitBracketOnEntryFill` L1449 | EXISTS |
| LIMIT **exit** (4.3 limit-then-market) | `close_position.limit_price` — Go `ClosePositionPayload.LimitPrice` (tcp_framing.go:289) + `TCPTrader.CloseWithLimit` (tcp_trader.go:484); C# `HandleClosePosition` L1063-1078 (`OrderType.Limit`, name `<signal_id>-lx`, market `Flatten` fallback L1079) | EXISTS both sides |
| In-place SL modify | `move_stop` (Go→C#); C# `HandleMoveStop` L1387 → `StopPriceChanged` + `Account.Change` (the safe-in-place pattern, 2026-08-07 OCO-cascade lesson) | EXISTS |
| LIMIT **entry** | — | **MISSING** |
| OCO bracket as one submitted unit | — (bracket follows fill, fine for armed fills) | MISSING for arbitrary positions only |
| CANCEL a working order | `TCPTrader.CancelAllOrders` returns "not supported" (tcp_trader.go:538); no C# branch | **MISSING** |
| Modify TP | — (SL only) | **MISSING** |
| Working/cancelled order events → Go | C# `OnOrderUpdate` L1173 early-returns on Accepted/Working/Cancelled — only Filled/Rejected/PartFilled emit frames | **MISSING** |

Fill/reject events exist: `fill` (filled|rejected|partial), `position_close`,
`position_close_rejected`; Go consumes them via tcp_server.go; reconnect
reconcile (`provider/ninjatrader/reconcile.go`) anchors entry prices + clears
orphans but does NOT reconcile working orders.

## 0.2 Minimal additive frames (spec)

All stay on the MANAGED API (`Account.CreateOrder/Submit/Change/Cancel`) —
never `SubmitOrderUnmanaged` (would bypass SIM guard + identity echo). Order-name
conventions preserved: entry bare `<signal_id>`, SL `-sl`, TP `-tp`, exit `-lx`,
OCO group `-exit`.

1. **`signal` + additive fields** `order_type` (`"market"` default | `"limit"`)
   + `limit_price` → C# `HandleSignal` limit branch: `OrderType.Limit, TIF.Day,
   qty, limitPrice, 0, string.Empty, signalId` + `workingEntries[signalId]`
   registry; existing pending-bracket → fill → `SubmitBracketOnEntryFill` path
   then works UNCHANGED (limit fills fire `OnOrderUpdate` like market fills).
2. **`cancel_order`** `{symbol, signal_id, account, trader_id, seq}` → C#:
   cancel the working limit entry (`workingEntries`) and/or bracket legs
   (`placedBrackets[signal_id].SlOrder/.TpOrder`) via `Account.Cancel`.
3. **`modify_bracket`** `{signal_id, new_stop_loss?, new_take_profit?}` → C#:
   mirror `HandleMoveStop` (`StopPriceChanged`/`LimitPriceChanged` +
   `Account.Change`) with the same Working/Accepted guard.
4. **`order_update`** (C#→Go) `{signal_id, order_name, state, fill_price, qty,
   account, trader_id, seq}` emitted from `OnOrderUpdate` for ALL states
   (per-order state-change dedup); `fill`/`position_close*` remain the terminal
   frames.

Estimated: **C# ≈ 160-180 lines** (25 limit entry + 30 cancel + 30 modify TP +
45 order_update + workingEntries registry) · **Go ≈ 215 lines** (payloads,
server send/recv + dispatch, TCPTrader methods, reconcile wire-up, tests).

## 0.3 Event flow back to Go — cited

- C# → Go: `OnOrderUpdate` state filter `VLTraderTCPClient.cs:1173-1183` (only
  Filled/Rejected/PartFilled); `fill` frame; `position_close` /
  `position_close_rejected` frames. Go consumer: `provider/ninjatrader/
  tcp_server.go` frame dispatch; fills persisted to `trader_fills`
  (exit lineage `nt8-exit-<id>` since 4.2).
- Dead-man watchdog blocks entries on TCP gap; reconnect reconcile
  (`provider/ninjatrader/reconcile.go`) — entry-price anchor + orphan clear,
  working-order reconciliation EXTENDED in Phase 2.3.

## Decisions locked

- Proceed to Phase 1 (Go-only, zero behavior change) now.
- Phase 2 C# frames per the spec above; AddOn recompile is the owner's F5 step
  at the Phase 5 cutover (flag every .cs diff + partner lockstep warning).
- Armed fills bypass stale_reeval by construction (fill happened AT the
  authorized price); entry class logged `armed_fill`.
