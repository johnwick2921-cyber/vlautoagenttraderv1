// Package ninjatrader — TCP framing codec for Plan 1.5.
//
// Wire format: 4-byte big-endian length prefix followed by UTF-8 JSON payload.
// Max frame size: 1 MB. Oversized frames are an error (server closes the
// connection per spec L4376, L4416).
//
// Four message types per spec L4378-4410: signal | fill | heartbeat | ack.
// SEPARATE FILE from tcp_server.go per spec file-manifest L4360.
package ninjatrader

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// FrameType enumerates the 4 wire envelope types per spec L4382.
type FrameType string

const (
	FrameSignal    FrameType = "signal"
	FrameFill      FrameType = "fill"
	FrameHeartbeat FrameType = "heartbeat"
	FrameAck       FrameType = "ack"
)

// Envelope wraps every frame on the TCP stream per spec L4381-4385.
type Envelope struct {
	Type    FrameType       `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// SignalPayload is the Go-server → C#-AddOn signal frame per spec L4387-4396.
type SignalPayload struct {
	Symbol string `json:"symbol"`
	// Account (P5.4): the NT8 sub-account this order must submit on. EMPTY =
	// legacy → the AddOn's active account (byte-identical wire via omitempty).
	// When set, the AddOn's Phase-2/3 resolve+guard+route path (already
	// shipped) submits to THIS account — per-(symbol,account) routing.
	Account    string  `json:"account,omitempty"`
	Side       string  `json:"side"` // "long" | "short"
	Quantity   int     `json:"quantity"`
	Entry      float64 `json:"entry"` // tick-rounded
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
	SignalID   string  `json:"signal_id"` // UUID
	Timestamp  string  `json:"timestamp"` // RFC3339
	// A2 (G1, wire v3) — identity stamp. trader_id is the OWNING trader; seq is the
	// server's monotonic per-connection op counter. The AddOn ECHOES (trader_id,
	// account, seq) on the paired ack/fill/close/reject so Go can verify the AddOn
	// acted for the originator. omitempty → a pre-v3 wire stays byte-identical.
	TraderID string `json:"trader_id,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// FillPayload is the C#-AddOn → Go-server fill frame per spec L4398-4406.
// P5.2 adds Symbol (the order's root, e.g. "MNQ") so multi-symbol fills are
// attributable. Empty = legacy AddOn (pre-P5.2) → consumers treat it as the
// primary trading symbol (back-compat; new field is additive JSON).
type FillPayload struct {
	SignalID string `json:"signal_id"`
	Symbol   string `json:"symbol,omitempty"` // P5.2 — order's root symbol; empty = legacy (primary)
	// Account is the NT sub-account this fill executed on (H3 fix). The C# AddOn
	// already sends it (VLTraderTCPClient SendFillFrame ["account"]); Go now parses
	// it so fills route to the OWNING trader by (symbol,account) and a trader never
	// caches another account's fill. omitempty keeps legacy (account-less) fill
	// goldens byte-identical.
	Account       string  `json:"account,omitempty"`
	FillPrice     float64 `json:"fill_price"`
	FillTime      string  `json:"fill_time"` // RFC3339
	Side          string  `json:"side"`
	Quantity      int     `json:"quantity"`
	SlippageTicks float64 `json:"slippage_ticks"`
	Status        string  `json:"status"` // "filled" | "rejected" | "partial"
	// A2 (G1, wire v3) — echoed identity from the originating signal. Go verifies
	// (trader_id, account, seq) against the pending op; a present mismatch freezes the
	// trader (A4). Empty = pre-v3 AddOn (echo absent) → tolerated in the deploy window.
	TraderID string `json:"trader_id,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// P5.2 — protocol handshake. The C# AddOn sends `hello` as the FIRST frame on
// every (re)connect; the Go server replies with its own `hello`. On an explicit
// VERSION MISMATCH the server refuses the connection with a loud log (never
// silently misparse). A connection that never sends hello is a LEGACY AddOn
// (pre-P5.2): tolerated with a warning so the lockstep deploy window (new Go,
// not-yet-F5'd C#) cannot brick the bar feed.
const FrameHello FrameType = "hello"

// ProtocolVersion is the current wire protocol generation. v2 = P5.2
// (symbol-tagged fills + hello handshake). Bump ONLY with a lockstep C#+Go ship.
// v3 = A2 (G1) identity stamp + echo-verify: order/modify/cancel frames carry
// (trader_id, account, seq) and the AddOn echoes all three on ack/fill/close/reject.
// Both sides tolerate unknown fields, so a v2 peer only loses the echo check.
const ProtocolVersion = 3

// HelloPayload identifies the peer + its protocol generation.
type HelloPayload struct {
	ProtocolVersion int    `json:"protocol_version"`
	Source          string `json:"source"` // "vltrader-addon" | "nofx-go"
}

// P5.3 — subscription acks (C#-AddOn → Go-server). The AddOn confirms or
// rejects each bars_subscribe/bars_unsubscribe so the Go side (and the owner
// API) sees subscription state without reading the NT8 Output window. ADDITIVE:
// a pre-P5.3 AddOn simply never sends them (state shows "pending").
const (
	FrameSubscribed     FrameType = "subscribed"
	FrameUnsubscribed   FrameType = "unsubscribed"
	FrameSubscribeError FrameType = "subscribe_error"
)

// SubscribedPayload acks a successful bars_subscribe: the root + the resolved
// front-month contract the BarsRequests were opened against.
type SubscribedPayload struct {
	Symbol           string `json:"symbol"`
	ResolvedContract string `json:"resolved_contract"`
}

// UnsubscribedPayload acks a bars_unsubscribe; Removed counts the disposed
// (symbol|timeframe) subscriptions.
type UnsubscribedPayload struct {
	Symbol  string `json:"symbol"`
	Removed int    `json:"removed"`
}

// SubscribeErrorPayload reports a FAILED bars_subscribe (instrument unresolved
// / not in NT8's DB) so a typo'd symbol fails loudly Go-side instead of dying
// silently in the NT8 Output window.
type SubscribeErrorPayload struct {
	Symbol string `json:"symbol"`
	Reason string `json:"reason"`
}

// AckPayload acknowledges a heartbeat or a specific signal_id per spec L4410.
// A2 (G1, wire v3): an order ack ALSO echoes the originating identity so Go can
// verify it (the heartbeat ack leaves them empty). SignalID names the acked order.
type AckPayload struct {
	Acks     string `json:"acks"` // "heartbeat" or "<signal_id>"
	SignalID string `json:"signal_id,omitempty"`
	TraderID string `json:"trader_id,omitempty"`
	Account  string `json:"account,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// HeartbeatPayload is an empty struct — spec L4408 says empty payload.
type HeartbeatPayload struct{}

// Plan 4.4 Stage 2 — bar frame types (envelope format identical to signal/fill/heartbeat/ack: 4-byte big-endian length + JSON {type, payload}).
// See ninjascript/vltrader_tcp_PROTOCOL.md sections 5-8 for field semantics.
const (
	FrameBarsSubscribe   FrameType = "bars_subscribe"
	FrameBarsHistorical  FrameType = "bars_historical"
	FrameBarUpdate       FrameType = "bar_update"
	FrameBarsUnsubscribe FrameType = "bars_unsubscribe"
)

// Plan 4.11 — real NT account balance. C#-AddOn → Go-server, additive frame
// (same envelope: 4-byte BE length + JSON {type, payload}). The C# AddOn emits
// it from the resolved Sim account on AccountItemUpdate + periodically, so the
// dashboard shows the real SIM balance instead of the $50k mock. Existing
// frames are unchanged.
const FrameAccountBalance FrameType = "account_balance"

// accounts_list frame: C#-AddOn → Go-server, unsolicited event emitted on connect
// and when NT8 accounts change. Reports all available accounts with SIM detection flag.
const FrameAccountsList FrameType = "accounts_list"

// AccountSelectPayload requests the C# AddOn to switch to a different account.
// Go-server → C#-AddOn command.
const FrameAccountSelect FrameType = "account_select"

// AccountBalancePayload carries an NT account snapshot. Fields map to
// NinjaTrader AccountItem values (CashValue, BuyingPower, RealizedProfitLoss,
// UnrealizedProfitLoss). NetLiquidation is the dashboard's "total equity"
// (cash + unrealized PnL); the C# side sends it directly if NT exposes it,
// else the Go side derives it as CashValue + UnrealizedPnL.
type AccountBalancePayload struct {
	Account        string  `json:"account"` // e.g. "Sim101"
	CashValue      float64 `json:"cash_value"`
	BuyingPower    float64 `json:"buying_power"`
	RealizedPnL    float64 `json:"realized_pnl"`
	UnrealizedPnL  float64 `json:"unrealized_pnl"`
	NetLiquidation float64 `json:"net_liquidation"` // total equity
}

// Position-history fix — C#-AddOn → Go-server, additive frame (same envelope).
// Emitted when an OCO exit leg (SL or TP) fills and the position goes flat, so
// the Go side can mark the open trader_position CLOSED with the real exit price.
// NT closes positions broker-side via the bracket; the bot never issues a
// close_* order, and NT has no order-sync, so without this frame the open
// position record never transitions to CLOSED and position history stays empty.
const FramePositionClose FrameType = "position_close"

// PositionClosePayload reports a closed (or partially closed) NT position.
// PositionSide is the side that was HELD ("long"/"short"), not the exit order's
// action. RealizedPnL is left to the Go side to compute against the recorded
// entry × the futures point value (single source of truth: market.FuturesPointValue).
type PositionClosePayload struct {
	SignalID     string  `json:"signal_id"`     // entry signal_id, for correlation
	Symbol       string  `json:"symbol"`        // root symbol, e.g. "MNQ"
	PositionSide string  `json:"position_side"` // "long" | "short" (held side)
	ExitPrice    float64 `json:"exit_price"`
	Quantity     int     `json:"quantity"`
	ExitReason   string  `json:"exit_reason"` // "sl" | "tp" | "manual"
	ExitTime     string  `json:"exit_time"`   // RFC3339
	// Account: the NT8 sub-account this close is on. The C# AddOn already SENDS it
	// (SendPositionCloseFrame ["account"]); Go simply wasn't parsing it. Used by
	// close-sync owner-routing to match the close to the trader that OWNS the row.
	Account string `json:"account"` // e.g. "Sim101"
	// A2 (G1, wire v3) — echoed originator identity + op seq for echo-verify.
	TraderID string `json:"trader_id,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// Rejected exit/flatten — C#-AddOn → Go-server, additive frame. The SIM (or
// broker) REJECTED an exit/flatten order (e.g. "There is no market data available
// to drive the simulation engine" while the Tradovate feed is down). The close did
// NOT take effect — the position is STILL OPEN in NT8 — so the bot must not treat
// it as closed. Pairs with the close routing (decision closes wait for the
// position_close fill frame): this frame turns a silent rejected flatten into a
// loud alarm so the orphan can't be netted onto by the next entry.
const FramePositionCloseRejected FrameType = "position_close_rejected"

// PositionCloseRejectedPayload reports a rejected exit/flatten order.
type PositionCloseRejectedPayload struct {
	SignalID     string `json:"signal_id"`     // order name (entry signal_id, or "Close" for a flatten)
	Symbol       string `json:"symbol"`        // root symbol, e.g. "MNQ"
	PositionSide string `json:"position_side"` // "long" | "short" (held side)
	Reason       string `json:"reason"`        // NT8 reject comment, e.g. "There is no market data..."
	Account      string `json:"account"`       // e.g. "Sim101"
	RejectTime   string `json:"reject_time"`   // RFC3339
	// A2 (G1, wire v3) — echoed originator identity + op seq for echo-verify.
	TraderID string `json:"trader_id,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// Feed status — C#-AddOn → Go-server, additive frame. Carries the NT8 price-feed
// PriceStatus (from OnVLConnectionStatusUpdate). The bot gates opens/closes when
// the feed is not Connected: the SIM rejects orders with "no market data", and
// acting into a dead feed is the upstream condition behind the phantom-close mess.
const FrameFeedStatus FrameType = "feed_status"

// FeedStatusPayload reports the NT8 price-feed status.
type FeedStatusPayload struct {
	PriceStatus string `json:"price_status"` // "Connected" | "ConnectionLost" | "Connecting" | "Disconnected"
	Time        string `json:"time"`         // RFC3339
}

// instrument_info — C#-AddOn → Go-server, additive frame. Carries the RESOLVED
// NT8 instrument's REAL contract specs (MasterInstrument.PointValue/.TickSize +
// the qualified contract name) so Go can source ground-truth from NT8 and
// cross-check the hardcoded FuturesPointValue/FuturesTickSize tables (drift
// detection). The tables remain the fallback for parked/unresolved symbols.
const FrameInstrumentInfo FrameType = "instrument_info"

// InstrumentInfoPayload reports a resolved instrument's NT8 specs.
type InstrumentInfoPayload struct {
	Symbol     string  `json:"symbol"`      // root, e.g. "MNQ"
	Contract   string  `json:"contract"`    // qualified, e.g. "MNQ 06-26"
	PointValue float64 `json:"point_value"` // NT8 MasterInstrument.PointValue
	TickSize   float64 `json:"tick_size"`   // NT8 MasterInstrument.TickSize
}

// Manual close — Go-server → C#-AddOn, additive frame. The AddOn flattens the
// position for the symbol (account.Flatten), which closes at market AND cancels
// the protective bracket so no orphaned SL/TP can re-open a position. The
// flatten's market fill comes back as a position_close (reason "manual").
const FrameClosePosition FrameType = "close_position"

// ClosePositionPayload requests a flatten of the symbol's position. Quantity is
// advisory (the AddOn flattens the whole position); Side is informational.
type ClosePositionPayload struct {
	Symbol   string `json:"symbol"`
	Side     string `json:"side"` // "long" | "short" (informational)
	Quantity int    `json:"quantity"`
	SignalID string `json:"signal_id"`
	// A2 (G1, wire v3) — identity stamp: the account this flatten targets + the
	// owning trader + the op seq. The AddOn echoes them on the resulting
	// position_close so Go verifies the close came back for the right originator.
	Account  string `json:"account,omitempty"`
	TraderID string `json:"trader_id,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// A3 (G2) — account allowlist. The Go server sends this (once per connect, before
// flushing queued signals, and again whenever a new trader binds) with the FULL set
// of accounts this session's bound traders use. The AddOn REPLACES its allowlist
// with this owner-declared set and refuses to execute on anything outside it — the
// allowlist is thus established independently of the signal payload (never
// self-registered from a signal, which would be vacuous). A pre-A3 Go never sends
// it, so a new AddOn treats an all-absent allowlist as legacy → fail-open (allow),
// keeping the lockstep deploy window safe.
const FrameAccountRegister FrameType = "account_register"

// AccountRegisterPayload carries the authoritative bound-account allowlist.
type AccountRegisterPayload struct {
	Accounts []string `json:"accounts"`
}

// FrameMoveStop asks the AddOn to move a RESTING stop-loss order (keyed by the
// entry's signal_id) to a new price WITHOUT closing the position — auto-breakeven.
// Additive frame: an OLD AddOn (pre-move_stop) logs "unknown frame type" and
// ignores it, so the original protective stop keeps guarding the trade — which is
// exactly why the paired C# redeploy (cp → F5 → NT8 restart) is required to
// activate breakeven.
const FrameMoveStop FrameType = "move_stop"

// MoveStopPayload is the Go-server → C#-AddOn move-stop request.
type MoveStopPayload struct {
	Symbol      string  `json:"symbol"`
	SignalID    string  `json:"signal_id"`     // the entry's signal_id (the bracket key)
	NewStopLoss float64 `json:"new_stop_loss"` // tick-rounded new stop price
	Timestamp   string  `json:"timestamp"`     // RFC3339
	// A2 (G1, wire v3) — identity stamp: target account + owning trader + op seq.
	Account  string `json:"account,omitempty"`
	TraderID string `json:"trader_id,omitempty"`
	Seq      uint64 `json:"seq,omitempty"`
}

// Open-position read-back — C#-AddOn → Go-server, additive snapshot frame (same
// envelope: 4-byte BE length + JSON {type, payload}, snake_case). The AddOn
// emits the selected account's CURRENT open positions: on account_select, on
// connect, and on any Account.PositionUpdate (INCLUDING a MANUAL open/close in
// NT8). This is the previously-missing position read-back — before it, Go only
// knew positions it opened itself (via a fill), so it lost the position on
// account switch-back and never saw manual trades. The frame is a FULL
// per-account snapshot (the complete non-flat set); the Go side REPLACES
// acctPositions[Account] with it, so a flat instrument simply drops out.
const FramePositions FrameType = "positions"

// OpenPosition is one non-flat NT position (per instrument) within a snapshot.
type OpenPosition struct {
	Symbol   string  `json:"symbol"`    // root symbol, e.g. "MNQ"
	Side     string  `json:"side"`      // "long" | "short"
	Quantity int     `json:"quantity"`  // contracts (NT8 nets per instrument)
	AvgPrice float64 `json:"avg_price"` // NT8 Position.AveragePrice
}

// PositionsPayload is the C#-AddOn → Go-server open-position snapshot for one
// account. Positions is the COMPLETE current non-flat set for Account (NT8 is
// the source of truth); the Go side replaces its per-account cache with it.
type PositionsPayload struct {
	Account   string         `json:"account"`
	Positions []OpenPosition `json:"positions"`
}

// Bar is the compact 6-field OHLCV bar used in bars_historical and bar_update
// frames (protocol §6-7). Volume is a float because NT8 tick-volume
// instruments report fractional values.
type Bar struct {
	T int64   `json:"t"` // Unix epoch ms, UTC
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v"` // volume can be tick-volume (fractional)
}

// BarsSubscribePayload is the Go-server → C#-AddOn subscribe frame per
// protocol §5. One frame requests N timeframes for a single symbol; the AddOn
// opens one BarsRequest per timeframe.
type BarsSubscribePayload struct {
	Symbol     string   `json:"symbol"`
	Timeframes []string `json:"timeframes"` // e.g. ["1m","5m","15m","1h"]
	BarsBack   int      `json:"bars_back"`  // default 500 per protocol
}

// BarsHistoricalPayload is the C#-AddOn → Go-server one-shot batch sent when
// the initial BarsRequest completes per protocol §6. Bars are ordered
// ascending by time.
type BarsHistoricalPayload struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	Bars      []Bar  `json:"bars"` // ascending by time
}

// BarUpdatePayload is the C#-AddOn → Go-server streaming update per protocol
// §7. Bars is ALWAYS an array — a single NT8 tick can update multiple bar
// indices (MinIndex..MaxIndex) when crossing timeframe boundaries.
type BarUpdatePayload struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	Bars      []Bar  `json:"bars"` // ALWAYS an array — single tick can update multiple indices (NT8 multi-bar gotcha). Walk MinIndex..MaxIndex.
}

// BarsUnsubscribePayload is the Go-server → C#-AddOn teardown frame per
// protocol §8. Empty/nil Timeframes means "all timeframes for this symbol".
type BarsUnsubscribePayload struct {
	Symbol     string   `json:"symbol"`
	Timeframes []string `json:"timeframes,omitempty"` // empty/nil = all for this symbol
}

// AccountInfo carries a single NT account's name + SIM flag.
type AccountInfo struct {
	Name  string `json:"name"`   // e.g. "Sim101" or "PropAcct"
	IsSim bool   `json:"is_sim"` // true if this is a simulation account
}

// AccountsListPayload reports all available NT accounts discovered by the C# AddOn.
// Emitted unsolicited on connect and on account list change, so the Go server can
// populate the UI's account selector and validate account_select commands.
type AccountsListPayload struct {
	Accounts []AccountInfo `json:"accounts"`
}

// AccountSelectPayload requests the C# AddOn to switch to a different account.
// The AddOn unsubscribes from the old account's events, resolves the new account
// by name, subscribes to its events, and re-emits account_balance immediately.
type AccountSelectPayload struct {
	Account string `json:"account"` // e.g. "Sim101"
}

// ErrFrameTooLarge signals the peer sent a length header > TCPMaxFrameBytes.
// Per spec L4416, the server logs and closes the connection on this error.
var ErrFrameTooLarge = errors.New("tcp_framing: frame exceeds max size")

// WriteFrame writes a single length-prefixed JSON frame to w.
// Returns an error if the marshalled body exceeds TCPMaxFrameBytes.
func WriteFrame(w io.Writer, frameType FrameType, payload any) error {
	body, err := marshalEnvelope(frameType, payload)
	if err != nil {
		return err
	}
	if len(body) > TCPMaxFrameBytes {
		return fmt.Errorf("tcp_framing: frame too large (%d > %d)", len(body), TCPMaxFrameBytes)
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(body)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("tcp_framing: write header: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("tcp_framing: write body: %w", err)
	}
	return nil
}

// ReadFrame reads a single length-prefixed JSON frame from r.
// Returns ErrFrameTooLarge if the length header exceeds TCPMaxFrameBytes
// (server closes the connection per spec L4416).
func ReadFrame(r io.Reader) (Envelope, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return Envelope{}, err
	}
	length := binary.BigEndian.Uint32(hdr[:])
	if length > TCPMaxFrameBytes {
		return Envelope{}, ErrFrameTooLarge
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return Envelope{}, fmt.Errorf("tcp_framing: read body: %w", err)
	}
	var env Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return Envelope{}, fmt.Errorf("tcp_framing: bad JSON: %w", err)
	}
	return env, nil
}

func marshalEnvelope(frameType FrameType, payload any) ([]byte, error) {
	var raw json.RawMessage
	if payload == nil {
		raw = json.RawMessage("{}")
	} else {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("tcp_framing: marshal payload: %w", err)
		}
		raw = b
	}
	body, err := json.Marshal(Envelope{Type: frameType, Payload: raw})
	if err != nil {
		return nil, fmt.Errorf("tcp_framing: marshal envelope: %w", err)
	}
	return body, nil
}
