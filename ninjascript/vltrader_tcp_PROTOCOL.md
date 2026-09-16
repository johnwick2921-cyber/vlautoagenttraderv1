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

### 8b. `bars_history_request` / `bars_history_data` / `bars_history_error` — HISTORY IMPORT (wave 101)

The live `bars_subscribe` path resolves the PLATFORM's front month and can
never ask for an expired contract. These three frames are the named-contract
channel: a one-off pull of `"MNQ 09-23"` over an explicit `[from, to)` window,
answered in chunks. It never touches the live subscription state.

```json
// Go server → C# AddOn
{ "type": "bars_history_request",
  "payload": {
    "request_id": "imp-3f9a…",     // caller-chosen correlation id
    "symbol":      "MNQ",
    "contract":    "MNQ 09-23",    // EXPLICIT name — never a date rule
    "timeframe":   "1m",           // 1m 3m 5m 15m 30m 1h 2h 4h 6h 8h 12h 1d
    "from_ms":     1700000000000,  // first bar open, epoch ms UTC (0 = contract start)
    "to_ms":       1710000000000   // exclusive end (0 = contract end)
  } }

// C# AddOn → Go server, one chunk per ~8k bars, ascending by time,
// seq from 1, last=true terminates the stream.
{ "type": "bars_history_data",
  "payload": {
    "request_id": "imp-3f9a…",
    "symbol":      "MNQ",
    "contract":    "MNQ 09-23",    // the instrument's REAL ContractName — echoed
    "timeframe":   "1m",
    "seq": 2, "last": true,
    "bars": [ { "t": 1700000000000, "o": 21500.25, "h": 21501.0,
                "l": 21500.0, "c": 21500.75, "v": 42 } ]
  } }

// C# AddOn → Go server — a pull that cannot be served is ANSWERED, never silent.
{ "type": "bars_history_error",
  "payload": { "request_id": "imp-3f9a…", "contract": "MNQ 09-23",
               "reason": "unavailable: instrument MNQ 09-23 not found on this platform" } }
```

Rules the pull obeys: `MergePolicy.DoNotMerge` ALWAYS (a back-adjusted series is
a different price scale wearing the same label — the 09-10 replay damage);
`TradingHours` = CME US Index Futures ETH; the request is disposable and the
AddOn tears it down on completion or terminate. The Go importer writes only
through `store.ImportBars` (no upsert; collision = keep + count) with
`source=historical_import`, and refuses a data frame whose echoed contract
differs from the one it asked for.

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

## order_snapshot (F12, 2026-09-03) — AddOn → Go

The BROKER's working-order book. Every other frame in this protocol is an
EVENT; `order_update` fires on a state change, so a Go-side restart loses the
picture until the next change happens — which on a quiet book may be never.
Cutover leg 4 therefore had to read the Go side's own `armed_orders` ledger and
call it the broker's book.

Emitted (a) every `ORDER_SNAPSHOT_INTERVAL_MS` (30 s, riding the heartbeat loop)
and (b) immediately after any order state change.

**ACCOUNT-SCOPED.** `Account.Orders` is an account collection and the AddOn holds
no persistent per-instrument handle, so one frame covers the account and every
order carries its own `symbol`. The Go side files the book per account and
filters by instrument. This is what keeps an EMPTY book representable: an account
with no working orders still emits `orders: []`. **"No orders" and "no answer"
are different claims** and leg 4 must be able to tell them apart.

```json
{"type":"order_snapshot","payload":{
  "account":"Sim101",
  "build_id":"2026-09-03-f12",
  "emitted_at_ms":1788480000000,
  "reason":"periodic|state_change",
  "orders":[
    {"order_id":"NT-1","name":"VL-S1-entry","action":"buy|sell",
     "type":"limit|stop|stop_limit|market","limit_price":29450.25,"stop_price":0,
     "quantity":1,"filled":0,"state":"Working","oco":"oco-1","symbol":"MNQ"}
  ]}}
```

Terminal orders (`Filled`, `Cancelled`, `Rejected`, `Expired`, `Unknown`) are
omitted by the AddOn — sending the whole history every 30 s would grow without
bound. The Go side filters again; **one definition of "working" lives in Go**
(`NT8Order.IsWorking`), so the two cannot drift on what the word means.

**STALENESS IS A SHARED CONSTANT.** The Go side calls a book older than
**2 × 30 s** stale (`trader.DefaultOrderSnapshotSecs`, env
`NT8_ORDER_SNAPSHOT_SECS`). The AddOn's `ORDER_SNAPSHOT_INTERVAL_MS` must match.
Changing one without the other changes what "stale" means on only one side.

**Age is measured against RECEIPT, not `emitted_at_ms`.** Those are two machines'
clocks, and a Windows-side skew must never make a stale book look fresh.

### build_id on `hello` and `heartbeat`

`hello` now carries `build_id` alongside `protocol_version` and `source`
(`omitempty` — an older AddOn's wire stays byte-identical), so the running DLL is
identifiable from the FIRST frame. `heartbeat` has carried it since E7.

**This is the only honest answer to "which DLL is NT8 running".** `VL_BUILD_ID` in
this repo is what we INTEND to be running; NT8 keeps executing whatever was last
compiled with F5. The Go boot line prints
`build_id=<received> expected=<source> match=yes|NO` and says **NO** until a frame
proves otherwise — a change to a distributed system is proven by a received
frame, never by a ledger write on the sending side.

### Go receive extension: placement truth (2026-09-07)

`fill` and `order_update` accept optional `reason` (JSON string). On an entry
rejection Go stores that text verbatim. The deployed `2026-09-07-h1` AddOn
**does not emit this field** for these frames; Go records
`reason unavailable (NT8 frame omitted reason)` when it is missing/blank.
The C# producer is explicitly deferred to the next owner-run AddOn wave.
This additive receive-only extension does not change the protocol version or
claim that h1 supplies rejection reasons.

Entry `signal.timestamp` is UTC command creation time (RFC3339 with fractional
seconds), independent of the market bar close used to compose entry prices.
Go checks that payload timestamp before enqueue and again immediately before
writing, including reconnect/retry. Missing, invalid, future, or older-than-60s
payload clocks are refused with the measured age (or age unavailable) logged.
Retries retain the original timestamp. This does not make stale market data
fresh or bypass any existing data/entry gates.

The armed ledger registers `signal_id` and `place_pending` atomically before
sending. Only a received live entry `order_update` or fresh `order_snapshot` naming
that exact entry promotes to `working`;
a received entry rejection (`fill.status` or `order_update.state`) settles as
`rejected`. Protective-leg updates cannot promote or reject the entry row.
A local socket return is not broker acceptance. Unanswered placements retain
their pending state and slot; a queue-age refusal is logged, never presented as
an NT8 rejection or silently re-authorized as another placement.
