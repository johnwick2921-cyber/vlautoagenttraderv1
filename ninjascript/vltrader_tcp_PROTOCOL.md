# VLTrader TCP Wire Protocol

This is the local C#-side copy of the wire protocol. The authoritative implementation is `provider/ninjatrader/tcp_server.go` in the Go repo; any drift between this doc and the Go impl is a bug — file an issue.

Spec source: `docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md` lines 4343-4447.

## Transport

- TCP loopback: `127.0.0.1:36974` (note: NOT NinjaTrader's ATI port 36973).
- Single concurrent client: NT8 AddOn → Go server. A second client is rejected.
- Connection is opened by the C# AddOn at NT8 startup; reconnects every 5 seconds on disconnect.

## Framing

Each frame on the TCP stream is:

1. 4-byte **big-endian** unsigned 32-bit length prefix.
2. UTF-8 JSON payload of exactly `length` bytes.

Maximum frame size: **1 MB (1,048,576 bytes)**. Oversized frames are a protocol error — the receiving side closes the connection.

## Envelope

Every JSON body has the same outer shape:

```json
{
  "type": "signal" | "fill" | "heartbeat" | "ack" | "bars_subscribe" | "bars_historical" | "bar_update" | "bars_unsubscribe",
  "payload": { }
}
```

## Message types

### 1. `signal` (Go server → C# AddOn)

Outgoing trade signal. Each numeric field is tick-rounded by the Go side before emission (see `trader/ninjatrader/tick_rounding.go`).

```json
{
  "type": "signal",
  "payload": {
    "symbol": "MNQ",
    "side": "long",
    "quantity": 1,
    "entry": 21500.25,
    "stop_loss": 21450.0,
    "take_profit": 21550.0,
    "signal_id": "uuid-v4-string",
    "timestamp": "2026-05-26T18:00:00Z"
  }
}
```

**Field semantics:**

- `symbol`: NT8 instrument symbol (e.g. `MNQ`, `NQ`, `ES`, `MES`).
- `side`: `long` (buy) or `short` (sell short).
- `quantity`: number of contracts.
- `entry`: market entry reference (NT8 uses market orders; this is the AI's planned entry — used only for slippage attribution in the fill frame).
- `stop_loss`: stop price (tick-rounded).
- `take_profit`: limit price (tick-rounded).
- `signal_id`: UUID. Used as the OCO group ID and as the fill correlation key.
- `timestamp`: RFC3339 UTC. The AddOn rejects signals older than 60 seconds as stale.

### 1b. `close_position` (Go server → C# AddOn)

Flatten a held position. Historical behavior: immediate market flatten +
working-order cancel. **4.3 (limit-then-market, dormant):** `limit_price > 0`
submits a LIMIT exit at that price instead — the Go side owns the timing and
sends the market fallback as a later `close_position` frame WITHOUT
`limit_price`, whose Flatten cancels the resting limit.

```json
{
  "type": "close_position",
  "payload": {
    "symbol": "MNQ",
    "side": "long",
    "quantity": 1,
    "limit_price": 0,
    "signal_id": "uuid-v4-string",
    "account": "Sim101",
    "trader_id": "hoang",
    "seq": 42
  }
}
```

**Field semantics:**

- `limit_price` (4.3): 0/absent = market flatten (historical, byte-identical
  framing). `> 0` = submit a LIMIT exit at this price. The order is named
  `<signal_id>-lx`; its fill routes to `position_close` with reason `"limit"`
  and cancels the still-live SL/TP bracket legs (`CancelBracketsFor`) so they
  can never re-enter the now-flat position.
- All other fields unchanged (identity stamp, account routing, SIM-only guard).

### 2. `fill` (C# AddOn → Go server)

Outgoing fill notification.

```json
{
  "type": "fill",
  "payload": {
    "signal_id": "matching-uuid",
    "fill_price": 21500.5,
    "fill_time": "2026-05-26T18:00:01Z",
    "side": "long",
    "quantity": 1,
    "slippage_ticks": 1.0,
    "status": "filled"
  }
}
```

**Field semantics:**

- `signal_id`: matches the originating signal's `signal_id`.
- `fill_price`: actual average fill price.
- `fill_time`: RFC3339 UTC.
- `slippage_ticks`: `(fill_price - entry) / tick_size`, signed (positive = paid more than planned for a long, less than planned for a short).
- `status`: `filled`, `rejected`, or `partial`. Rejected fills indicate NT8 refused the order; the Go side does NOT retry — manual operator intervention is required.
- `symbol` (P5.2, protocol v2): the order's instrument root (e.g. `"MNQ"`), for multi-symbol attribution. EMPTY/absent = legacy (pre-v2) AddOn → the Go side attributes the fill to the trader's primary symbol. The Go fill consumer REJECTS (warn + drop) a non-empty symbol that doesn't match the trader's instrument — the split-brain defense.
- `account` (Phase 4): the NT8 account the fill executed on.

### 2b. `hello` (bidirectional) — P5.2, protocol v2

The C# AddOn sends `hello` as the FIRST frame on every (re)connect; the Go server replies with its own `hello`.

```json
{ "type": "hello", "payload": { "protocol_version": 2, "source": "vltrader-addon" } }
```

**Semantics:**

- `protocol_version`: the peer's wire generation (`ProtocolVersion` in `tcp_framing.go` == `PROTOCOL_VERSION` in `VLTraderTCPClient.cs`; v2 = symbol-tagged fills + this handshake). Bump ONLY with a lockstep C#+Go ship.
- Go server on MISMATCH: logs `PROTOCOL VERSION MISMATCH — refusing connection` and CLOSES (never silently misparses). The AddOn's reconnect loop will retry + log its own mismatch warning from the server's reply.
- Go server on a connection that NEVER sends hello: tolerated as a LEGACY (pre-v2) AddOn with a one-time warning — so the lockstep deploy window (new Go, not-yet-F5'd C#) cannot brick the bar feed.
- `source`: `"vltrader-addon"` or `"nofx-go"` (diagnostics only).

### 3. `heartbeat` (bidirectional)

Empty payload. Sent every 30 seconds by both sides.

```json
{ "type": "heartbeat", "payload": {} }
```

### 4. `ack` (bidirectional)

```json
{ "type": "ack", "payload": { "acks": "heartbeat" } }
```

Or for a specific signal:

```json
{ "type": "ack", "payload": { "acks": "uuid-v4-string" } }
```

### 5. `bars_subscribe` (Go server → C# AddOn) — Plan 4.4 Stage 1

Request native multi-timeframe bar subscriptions for a single NT8 instrument.
For each timeframe the C# AddOn opens one `BarsRequest` against NT8's data
engine, emits one `bars_historical` frame on the initial load, then streams
`bar_update` frames as ticks arrive.

```json
{
  "type": "bars_subscribe",
  "payload": {
    "symbol": "MNQ",
    "timeframes": ["1m", "5m", "15m", "1h"],
    "bars_back": 500
  }
}
```

**Field semantics:**

- `symbol`: NT8 instrument symbol (e.g. `MNQ`). The AddOn resolves it via
  `Instrument.GetInstrument(symbol)` exactly like the signal path.
- `timeframes`: list of timeframe strings the engine wants. Valid values mirror
  `store/strategy.go::normalizeTimeframe`: `1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h,
  6h, 8h, 12h, 1d, 3d, 1w`. The AddOn maps each to a `BarsPeriod` and opens a
  separate `BarsRequest`. Unknown values are skipped with a log warn.
- `bars_back`: number of historical bars to load on initial subscription
  (default 500 if omitted). Same value used for every requested timeframe.

Re-subscribing to a `(symbol, timeframe)` that already has an active
subscription is idempotent — the second `bars_subscribe` returns immediately
without re-opening the `BarsRequest`.

### 6. `bars_historical` (C# AddOn → Go server) — Plan 4.4 Stage 1

Sent ONCE per `(symbol, timeframe)` when the initial `BarsRequest` completes.
Carries the full historical batch in a single frame (bounded by the 1 MB
envelope ceiling — at ~80 bytes per bar that's ~13k bars headroom, more than
the engine ever requests).

```json
{
  "type": "bars_historical",
  "payload": {
    "symbol": "MNQ",
    "timeframe": "1m",
    "bars": [
      { "t": 1748352000000, "o": 21500.25, "h": 21501.00, "l": 21500.00, "c": 21500.75, "v": 42 },
      { "t": 1748352060000, "o": 21500.75, "h": 21501.25, "l": 21500.50, "c": 21501.00, "v": 38 }
    ]
  }
}
```

**Field semantics:**

- `symbol`, `timeframe`: echo the originating `bars_subscribe` keys.
- `bars`: ordered ascending by time, each bar is a compact 6-field object:
  - `t`: Unix epoch milliseconds, **UTC**. The C# side converts NT8's
    local-time `bars.GetTime(i)` to UTC using
    `bars.TradingHours.TimeZoneInfo` before emit. Charts + indicators on the
    Go side assume UTC; mis-timezoned bars are the #1 cause of off-by-an-hour
    bugs.
  - `o, h, l, c`: bar OHLC, double-precision.
  - `v`: volume, may be a `double` for tick-volume instruments — emitted as
    JSON number, no special encoding.

### 7. `bar_update` (C# AddOn → Go server) — Plan 4.4 Stage 1

Streaming updates from `BarsRequest.Update`. **The `bars` field is an array
even when only one bar changed** — a single tick can update multiple bar
indices when crossing minute boundaries with high-frequency feeds, and the
NT8 contract is to walk `MinIndex..MaxIndex`. Always emit every bar in that
range, in ascending order.

```json
{
  "type": "bar_update",
  "payload": {
    "symbol": "MNQ",
    "timeframe": "1m",
    "bars": [
      { "t": 1748352120000, "o": 21501.00, "h": 21501.50, "l": 21500.75, "c": 21501.25, "v": 17 }
    ]
  }
}
```

**Field semantics:** identical to `bars_historical`. The Go side dedupes
by `(symbol, timeframe, t)` and treats a later frame for the same `t` as an
update-in-progress.

### 8. `bars_unsubscribe` (Go server → C# AddOn) — Plan 4.4 Stage 1

Tear down one or more `(symbol, timeframe)` subscriptions cleanly.

```json
{
  "type": "bars_unsubscribe",
  "payload": {
    "symbol": "MNQ",
    "timeframes": ["1m", "5m"]
  }
}
```

Omitting `timeframes` (empty or missing) is equivalent to "all timeframes
for this symbol." The AddOn disposes the affected `BarsRequest`s, stops
emitting `bar_update` frames for them, and removes them from its
subscription registry.

### 9. `subscribed` / `unsubscribed` / `subscribe_error` (C# AddOn → Go server) — P5.3

Subscription lifecycle acks, sent by the AddOn in response to `bars_subscribe` / `bars_unsubscribe`:

```json
{ "type": "subscribed",      "payload": { "symbol": "NQ", "resolved_contract": "NQ 06-26" } }
{ "type": "unsubscribed",    "payload": { "symbol": "NQ", "removed": 14 } }
{ "type": "subscribe_error", "payload": { "symbol": "XYZ", "reason": "qualified contract 'XYZ 06-26' not found in NT8 (not loaded?)" } }
```

**Semantics:**

- `subscribed`: emitted after the BarsRequests open, with the resolved front-month (`instrument.FullName`). The Go side marks the symbol `subscribed` (visible via `GET /api/nt/symbols`).
- `unsubscribed`: emitted after `bars_unsubscribe` disposal; `removed` counts the disposed `(symbol|timeframe)` subscriptions (0 = wasn't subscribed).
- `subscribe_error`: a FAILED subscribe (instrument unresolved / not in NT8's DB) — surfaces Go-side instead of dying silently in the NT8 Output window.
- ADDITIVE: a pre-P5.3 AddOn never sends these (the Go state shows `pending`; bars still flow). A pre-P5.3 Go logs them as unknown frames (harmless warn) — ship Go before the AddOn F5, as with v2.

### 10. `move_stop` (Go server → C# AddOn) — auto-breakeven

Moves an OPEN position's resting stop-loss to a new price WITHOUT closing it.
Used by auto-breakeven (once the trade is +N points in profit → stop → entry).

```json
{ "type": "move_stop",
  "payload": { "symbol": "MNQ", "signal_id": "<entry uuid>",
               "new_stop_loss": 30352.00, "timestamp": "RFC3339" } }
```

- The AddOn finds the live bracket by `signal_id` and MODIFIES THE SAME resting
  stop order IN PLACE: `SlOrder.StopPriceChanged = new_stop_loss` →
  `account.Change(SlOrder)`. Same order object, same OCO group, no new order and no
  cancel → the take-profit and the OCO group are never disturbed.
- **Why not submit-new-then-cancel-old (the 2026-08-07 fix):** the old AddOn created
  a NEW `StopMarket` in the SAME OCO group and cancelled the old stop. NT8 OCO
  cancels the WHOLE group when any member is cancelled, so the take-profit AND the
  new stop both died → NAKED position. Proven live 2026-08-07 11:25:05 (signal
  `b846e082…`: `-tp` Cancelled + both `-sl` orders Cancelled). `account.Change`
  never issues a cancel, so no cascade is possible.
- Guards: only acts when the stop is in a changeable state (`Working`/`Accepted`);
  no-op if the bracket already exited or the stop is already at that price (½-tick
  idempotency); on a non-changeable stop or a `Change` exception it replies
  `ack: move_stop_error` (not success) so Go re-arms/retries next cycle. Success
  replies `ack: move_stop`.
- **Additive frame:** an OLD AddOn (pre-`move_stop`) logs "unknown frame type" and
  ignores it — the original stop keeps protecting the trade. Activating breakeven
  therefore REQUIRES the paired redeploy (cp `ninjascript/*.cs` → AddOns → F5 →
  clean NT8 restart).

## Failure modes

- **TCP disconnect**: the server holds the signal queue; on reconnect, it sends pending signals with their original timestamps. The C# AddOn may reject signals older than 60s as stale (emits a `status=rejected` fill).
- **Heartbeat timeout**: the server closes the connection after 60s without an ack; the C# AddOn reconnects every 5s.
- **Invalid frame** (bad length, >1 MB, malformed JSON): the receiver logs a warning and closes the connection; the other side reconnects.
- **Order rejection by NT8**: the AddOn emits a fill frame with `status=rejected`; the Go side logs the rejection and does NOT retry.

## Cross-references

- Go-side authoritative implementation: `provider/ninjatrader/tcp_server.go`.
- Go-side framing codec: `provider/ninjatrader/tcp_framing.go` (length-prefix + JSON envelope shared with this AddOn).
- Architectural rationale: `docs/adr/ADR-001-csv-bridge-vs-tcp.md`.
- Plan 1.5 spec: `docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md` lines 4343-4447.
- Plan 1 critical-file integrity guard: `docs/adr/ADR-007-plan1-critical-file-integrity.md` (Plan 1.5 is purely additive — none of the CSV bridge files are modified).
- Plan 4.4 deep spec: same plan doc, Plan 4.4 Deep Spec section. Defines `bars_subscribe`, `bars_historical`, `bar_update`, `bars_unsubscribe` envelopes consumed by the new C# `VLBarsSubscriptionManager`.
- Plan 4.4 Stage 1 C# implementation: `ninjascript/VLBarsSubscriptionManager.cs`. Isolates BarsRequest logic from the proven signal/fill/heartbeat path in `VLTraderTCPClient.cs` (which gains only a field, a constructor call, and two switch cases).
