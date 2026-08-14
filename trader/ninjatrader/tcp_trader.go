// Package ninjatrader — alternative Trader implementation that emits signals
// through the Plan 1.5 TCP bridge instead of the Plan 1 CSV bridge.
//
// SEPARATE TYPE from the existing CSV `Trader` (per ADR-007: additive, not a
// modification). Implements the same 19-method `trader/types.Trader`
// interface. Selection is via NT_TRANSPORT env var (transport.go).
//
// Wire-protocol contract: provider/ninjatrader/tcp_framing.go + tcp_server.go.
// Tick rounding: shared with the CSV trader via in-package call to
// RoundToTick / InstrumentTickSize (tick_rounding.go).
package ninjatrader

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"nofx/config"
	"nofx/logger"
	"nofx/market"
	"nofx/provider/databento"
	ntwire "nofx/provider/ninjatrader"
	"nofx/telemetry"
	"nofx/trader/types"
) // reflect used in GetBalance to notify parent AutoTrader

// TCPTrader satisfies trader/types.Trader using the TCP bridge.
type TCPTrader struct {
	server *ntwire.TCPServer
	symbol string // for tick rounding selection
	// boundAccount (P5.4): the NT8 sub-account this trader trades on
	// (store.Trader.Account). Empty = the AddOn's active account (legacy).
	// When set, signals carry it and the AddOn's Phase-3 routing submits there.
	boundAccount string

	// traderID (A2/G1): the OWNING trader's id, stamped on every order/modify/cancel
	// frame so the AddOn can echo it back for identity verification. Set at
	// StartCloseSync (which already receives it). Empty until then.
	traderID string

	mu       sync.Mutex
	stopLoss map[string]float64 // key: "<symbol>:<side>"
	takePrft map[string]float64
	guard    *orderGuard // B3: dupe-drop + rate breaker at the order-submission chokepoint
	lastFill ntwire.FillPayload
	hasFill  bool

	// closedAt records the wall-clock (ms) of the most recent FILL-CONFIRMED close
	// (position_close frame) per "SYMBOL|SIDE" for THIS trader's bound account. The
	// reconcile-before-open gate awaits this event — which arrives ~instantly on the
	// flatten fill for the bound account REGARDLESS of which account is the streamed
	// "active" one — instead of the positions snapshot, whose non-active-account
	// refresh is only the 30s heartbeat (the 6s-vs-30s starvation). Guarded by mu.
	closedAt map[string]int64

	// lastEntrySignalID is the signal_id of the most recent ENTRY (open). It lets
	// GetOrderStatus report ONLY the fill that belongs to the current entry (the
	// fill frame echoes signal_id), so recordAndConfirmOrder records the real
	// NT8 AverageFillPrice as entry_price instead of the stale 5m-mark reference.
	// Cleared on close so a close-path poll never matches the entry fill.
	lastEntrySignalID string

	// pending tracks signal_id → side so we can correlate fills back to
	// position state. Optional: not strictly needed for the 19-method
	// interface, but cheap insurance.
	pendingMu sync.Mutex
	pending   map[string]string // signal_id -> side

	// closeSyncOnce guards StartCloseSync so a re-entrant AutoTrader.Run never
	// spawns a second consumer racing on the single ClosedPositions() channel.
	closeSyncOnce sync.Once

	// reconcileOnce guards StartPositionReconcile (mirrors closeSyncOnce) so a
	// re-entrant AutoTrader.Run never spawns a second reconcile goroutine.
	reconcileOnce sync.Once

	// flatSince tracks, per open-position row id, the first time the reconcile loop
	// observed it NT8-flat-but-DB-open (Unix ms). It implements the flat-grace
	// window: reconcile defers orphan-closing a freshly-flat row so close-sync's
	// position_close frame can arrive and record the real ×pv P&L first. Accessed
	// ONLY from the single reconcile goroutine (reconcilePositions) → no lock needed.
	flatSince map[int64]int64

	// Plan 4 Stage 4 — reference to the parent AutoTrader (optional).
	// Used to notify the AutoTrader when the first account_balance frame arrives.
	// Set by transport.go after creating the trader.
	parentAutoTrader interface{} // *AutoTrader (avoid circular import)
}

// Compile-time interface guard (ADR-007 pattern — mirrors CSV Trader).
var _ types.Trader = (*TCPTrader)(nil)

// NewTCPTrader wraps an already-Started TCPServer with the Trader interface.
// The server's lifecycle is owned by the caller (transport.go).
func NewTCPTrader(server *ntwire.TCPServer, symbol string, account ...string) *TCPTrader {
	t := &TCPTrader{
		server:   server,
		symbol:   symbol,
		stopLoss: map[string]float64{},
		takePrft: map[string]float64{},
		guard:    newOrderGuard(),
		pending:  map[string]string{},
	}
	if len(account) > 0 {
		t.boundAccount = strings.TrimSpace(account[0])
	}
	if t.boundAccount != "" {
		logger.Infof("🔗 ninjatrader/tcp: trader %s BOUND to account %s (P5.4 per-(symbol,account) routing)", symbol, t.boundAccount)
	}
	// Phase 2 — drive the bar subscription from THIS trader's symbol instead of
	// the hardwired default "MNQ". On the next (re)connect the AddOn subscribes to
	// this root and NT8 resolves the qualified front-month (VLContractResolver,
	// quarterly families). MNQ is unchanged (it is just the symbol here too).
	server.SetBarsSubscribeSymbol(symbol)
	// P5.2 — extra bar-subscription roots (bars-only until P5.4) are wired by
	// transport.go AFTER this constructor: the config symbols-list REPLACES the
	// extras (source of truth per (re)load), then the NT_EXTRA_SYMBOLS testing
	// override appends. Both dedup against the real primary set above.
	// Subscribe to inbound fills — update lastFill cache (mirrors CSV Trader).
	go func() {
		// P5.4 — router-fed per-symbol stream (no cross-trader racing). The
		// channel CLOSES when a reloaded trader re-subscribes, ending this
		// goroutine (fixes the pre-P5.4 reload leak).
		for fill := range server.SubscribeFillsFor(symbol, t.boundAccount) {
			// P5.2 split-brain defense: a symbol-tagged fill for a DIFFERENT
			// instrument must never be attributed to this trader. Empty symbol
			// = legacy (pre-P5.2) AddOn → assumed primary (back-compat).
			if fill.Symbol != "" && !equalSymbol(fill.Symbol, symbol) {
				logger.Warnf("⚠️ ninjatrader/tcp: REJECTED fill for symbol %q (this trader trades %q) — signal_id=%s (split-brain defense)",
					fill.Symbol, symbol, fill.SignalID)
				continue
			}
			// H3 account guard (defense in depth on top of the (symbol,account)
			// routing): with two same-symbol traders, NEVER cache another account's
			// fill as this trader's position. Empty account = legacy AddOn → trust
			// the symbol routing.
			if fill.Account != "" && t.boundAccount != "" && !strings.EqualFold(fill.Account, t.boundAccount) {
				logger.Warnf("⚠️ ninjatrader/tcp: REJECTED fill for account %q (this trader is bound to %q) — signal_id=%s (H3 account guard)",
					fill.Account, t.boundAccount, fill.SignalID)
				continue
			}
			t.mu.Lock()
			t.lastFill = fill
			t.hasFill = true
			t.mu.Unlock()
		}
	}()
	return t
}

// --- Trader interface methods (19 total) ---

func (t *TCPTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.placeEntry(symbol, "long", quantity)
}

func (t *TCPTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.placeEntry(symbol, "short", quantity)
}

// activeAccountName is the NT8 account the connection will trade (the streamed
// current account). "" before the first account frame.
func (t *TCPTrader) activeAccountName() string {
	if a := t.server.CurrentAccount(); a != nil {
		return *a
	}
	return ""
}

// isAccountTradeable reports whether an order may be sent for `name`: it must be a
// SIM account (per the C#-reported accounts list, which uses Account.Simulation) AND,
// if NT_ALLOWED_ACCOUNTS is configured, on that allow-list. This is the hard
// live/funded-account block (Stage-2 Phase-1). Fail-safe: unknown account → false.
func (t *TCPTrader) isAccountTradeable(name string) bool {
	if name == "" {
		return false
	}
	accts, _ := t.server.GetAccountsList()
	isSim := false
	found := false
	for _, a := range accts {
		if a.Name == name {
			isSim = a.IsSim
			found = true
			break
		}
	}
	if !found || !isSim {
		return false // unknown or non-SIM (incl. the live account) → never tradeable
	}
	if allow := config.Get().AllowedNTAccounts; len(allow) > 0 {
		for _, a := range allow {
			if a == name {
				return true
			}
		}
		return false // an allow-list is set and this account isn't on it
	}
	return true
}

func (t *TCPTrader) placeEntry(symbol, side string, quantity float64) (map[string]interface{}, error) {
	// SAFETY RAIL (Stage-2 Phase-1, defense-in-depth): never SEND an entry for an
	// account that isn't tradeable (SIM + allow-listed). The C# AddOn enforces this
	// again right before submit; this refuses in Go before the frame is even sent.
	// The LIVE/funded account is never tradeable.
	// MULTI-ACCOUNT SAFETY: an UNBOUND trader (empty boundAccount) must NOT borrow
	// the connection's shared active account — that would execute on whatever account
	// was last globally selected (i.e. ANOTHER trader's account). Refuse outright,
	// replacing the old activeAccountName() fallback.
	tradeAcct := t.boundAccount
	if tradeAcct == "" {
		return nil, fmt.Errorf("ninjatrader/tcp: refusing %s entry on %s — trader has no bound account (select an account first); NOT falling back to the shared active account", side, symbol)
	}
	if !t.isAccountTradeable(tradeAcct) {
		return nil, fmt.Errorf("ninjatrader/tcp: refusing %s entry — account %q is not tradeable (not on allow-list / not SIM)", side, tradeAcct)
	}

	// B3 — dupe guard + rate limiter at the order-submission chokepoint: a
	// replayed / double-fired entry (same account|side|symbol|qty within a bar) is
	// DROPPED, and an order runaway trips the per-minute breaker.
	if t.guard != nil {
		key := fmt.Sprintf("%s|%s|%s|%.0f", tradeAcct, upperSideStr(side), symbol, quantity)
		if reason, ok := t.guard.admit(key, time.Now().UnixMilli()); !ok {
			gate := "b3_order_dedup"
			if strings.Contains(reason, "breaker") {
				gate = "b3_rate_breaker"
				logger.Warnf("🚨 B3 %s", reason)
			} else {
				logger.Warnf("⛔ B3 %s", reason)
			}
			// B6: B3 fires at the shared submission chokepoint (bound to an account,
			// not a trader), so it tallies under the process-wide "" trader bucket.
			telemetry.IncGateBlock("", gate)
			return nil, fmt.Errorf("ninjatrader/tcp: entry not admitted — %s", reason)
		}
	}

	// Expiry warning — defense in depth (mirrors CSV Trader).
	if days := databento.DaysUntilExpiry(symbol, time.Now()); days >= 0 && days <= 5 {
		logger.Warnf("ninjatrader/tcp: placing %s entry on %s within %d days of expiry — verify contract roll", side, symbol, days)
	}

	// CSV Trader keys SL/TP by "LONG"/"SHORT" uppercase; mirror that here.
	upperSide := upperSideStr(side)
	t.mu.Lock()
	sl := t.stopLoss[keyFor(symbol, upperSide)]
	tp := t.takePrft[keyFor(symbol, upperSide)]
	tid := t.traderID // A2 (G1) — captured under lock (set-once at StartCloseSync)
	t.mu.Unlock()
	if sl == 0 || tp == 0 {
		return nil, fmt.Errorf("ninjatrader/tcp: SetStopLoss and SetTakeProfit must be called before %s", side)
	}

	// Mirror CSV Trader: entry price for the wire is a reference value
	// computed as SL/TP midpoint. NT places market orders; entry on the
	// wire is for the AddOn's logging + protective bracket reference.
	entryRef := (sl + tp) / 2.0

	tick := InstrumentTickSize(t.symbol)
	entry := RoundToTick(entryRef, tick)
	sl = RoundToTick(sl, tick)
	tp = RoundToTick(tp, tick)

	signalID := uuid.NewString()
	payload := ntwire.SignalPayload{
		Symbol:     t.symbol,
		Account:    t.boundAccount, // P5.4 — empty = legacy (AddOn's active account)
		TraderID:   tid,            // A2 (G1) — identity stamp; server assigns Seq
		Side:       side,           // lowercase per spec L4390
		Quantity:   int(quantity),
		Entry:      entry,
		StopLoss:   sl,
		TakeProfit: tp,
		SignalID:   signalID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	// A1 (G3) — PRE-SUBMIT IDENTITY INVARIANT: the account on the outbound order MUST
	// equal this trader's bound account. It is equal by construction above, but assert
	// it at the send chokepoint so no present or future code path can ever emit an
	// order for another trader's account. A mismatch is REFUSED loudly, never sent.
	if err := assertBoundAccount("entry", symbol, payload.Account, t.boundAccount); err != nil {
		logger.Errorf("🚨 %v — REFUSING to submit", err)
		return nil, err
	}

	t.pendingMu.Lock()
	t.pending[signalID] = upperSide
	t.pendingMu.Unlock()

	// Mark this as the current entry so GetOrderStatus reports its fill (and only
	// its fill) — the real NT8 AverageFillPrice becomes the recorded entry_price.
	t.mu.Lock()
	t.lastEntrySignalID = signalID
	t.mu.Unlock()

	if err := t.server.SendSignal(payload); err != nil {
		return nil, fmt.Errorf("ninjatrader/tcp: send signal: %w", err)
	}
	return map[string]interface{}{
		"status":    "submitted",
		"symbol":    symbol,
		"side":      side,
		"quantity":  quantity,
		"signal_id": signalID,
	}, nil
}

// MoveStopToBreakeven asks the AddOn to move the resting stop for THIS trader's
// current open position (the last entry's signal_id) to newStop, tick-rounded —
// WITHOUT closing the position. Errors (no-op) if there is no tracked open entry.
// Part 2 auto-breakeven. An OLD AddOn ignores the move_stop frame, so the
// original protective stop keeps guarding the trade until the paired redeploy.
func (t *TCPTrader) MoveStopToBreakeven(side string, newStop float64) error {
	key := keyFor(t.symbol, upperSideStr(side))
	t.mu.Lock()
	sid := t.lastEntrySignalID
	cur := t.stopLoss[key]
	tid := t.traderID // A2 (G1)
	t.mu.Unlock()
	if sid == "" {
		return fmt.Errorf("ninjatrader/tcp: no open entry to move the stop for %s", t.symbol)
	}
	newStop = RoundToTick(newStop, InstrumentTickSize(t.symbol))
	// B1 STOP-WIDEN BAN: a stop may only TIGHTEN (reduce risk), never widen. A widen
	// attempt is a bug or bad input — REFUSE it and never send it to the broker.
	if cur > 0 && stopWouldWiden(side, cur, newStop) {
		logger.Warnf("⛔ stop-widen REFUSED: %s %s new stop %.2f would WIDEN from %.2f (risk-increasing) — NOT sent.",
			t.symbol, side, newStop, cur)
		return fmt.Errorf("ninjatrader/tcp: stop-widen ban — %s %s new stop %.2f widens from %.2f", t.symbol, side, newStop, cur)
	}
	if err := t.server.SendMoveStop(ntwire.MoveStopPayload{
		Symbol:      t.symbol,
		SignalID:    sid,
		NewStopLoss: newStop,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Account:     t.boundAccount, // A2 (G1) — identity stamp
		TraderID:    tid,
	}); err != nil {
		return err
	}
	// Track the new stop so a subsequent move is checked against it.
	t.mu.Lock()
	t.stopLoss[key] = newStop
	t.mu.Unlock()
	return nil
}

// assertBoundAccount (A1/G3) is the pre-submit identity invariant: the account an
// order/modify/cancel is being submitted on MUST equal the emitting trader's bound
// account. Returns an error (the caller REFUSES + 🚨) on any mismatch — the
// defense-in-depth chokepoint that makes cross-account emission structurally
// impossible. An empty bound account is itself a mismatch unless the frame is also
// account-less (legacy), which the entry path already refuses upstream.
func assertBoundAccount(op, symbol, frameAccount, boundAccount string) error {
	if frameAccount != boundAccount {
		return fmt.Errorf("A1 pre-submit invariant: %s %s frame account %q ≠ bound account %q (cross-account emission blocked)", op, symbol, frameAccount, boundAccount)
	}
	return nil
}

// stopWouldWiden reports whether moving a stop from cur→next WIDENS it (increases
// risk) for the given side. Long: widen = next below cur (stop moves further from
// price). Short: widen = next above cur. Equal or tighter → false. (B1 stop-widen
// ban — the one authority for the tighten-only invariant.)
func stopWouldWiden(side string, cur, next float64) bool {
	if strings.EqualFold(strings.TrimSpace(side), "short") {
		return next > cur
	}
	return next < cur // long (default)
}

func (t *TCPTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	return t.sendClose("long", quantity)
}

func (t *TCPTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	return t.sendClose("short", quantity)
}

// sendClose asks the AddOn to flatten the symbol's position (close at market +
// cancel the protective bracket). The resulting market-exit fill returns as a
// position_close frame, which the close-sync records to history.
func (t *TCPTrader) sendClose(side string, quantity float64) (map[string]interface{}, error) {
	// Clear the entry correlation so the close-path order poll does NOT match the
	// lingering entry fill (which would record exit≈entry, PnL≈0). The real exit
	// is recorded by close-sync off the position_close frame.
	t.mu.Lock()
	t.lastEntrySignalID = ""
	tid := t.traderID // A2 (G1)
	t.mu.Unlock()

	payload := ntwire.ClosePositionPayload{
		Symbol:   t.symbol,
		Side:     side,
		Quantity: int(quantity),
		SignalID: uuid.NewString(),
		Account:  t.boundAccount, // A2 (G1) — identity stamp
		TraderID: tid,
	}
	if err := t.server.SendClosePosition(payload); err != nil {
		return nil, fmt.Errorf("ninjatrader/tcp: send close: %w", err)
	}
	return map[string]interface{}{
		"status": "close_submitted",
		"symbol": t.symbol,
		"side":   side,
	}, nil
}

func (t *TCPTrader) SetLeverage(symbol string, leverage int) error {
	return nil // futures leverage is set at the broker, not per-order
}

func (t *TCPTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	return nil // n/a for futures
}

func (t *TCPTrader) SetStopLoss(symbol, positionSide string, quantity, stopPrice float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopLoss[keyFor(symbol, upperSideStr(positionSide))] = stopPrice
	return nil
}

func (t *TCPTrader) SetTakeProfit(symbol, positionSide string, quantity, takeProfitPrice float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.takePrft[keyFor(symbol, upperSideStr(positionSide))] = takeProfitPrice
	return nil
}

func (t *TCPTrader) CancelAllOrders(symbol string) error {
	return fmt.Errorf("ninjatrader/tcp: CancelAllOrders not supported")
}

func (t *TCPTrader) GetBalance() (map[string]interface{}, error) {
	// Plan 4.11 — serve the REAL NT SIM account from the latest account_balance
	// frame (C# AddOn → tcp_server). No more $50k mock. Until the first frame
	// arrives (AddOn connecting), report zeros so the dashboard shows "no data"
	// rather than a fabricated balance; the C# emits on connect + periodically.
	//
	// Read THIS trader's OWN bound account (P5.4), mirroring GetPositions'
	// decouple (cb00347d) — GetBalance was the missed twin. The shared NT8
	// connection streams ONE "current" account; using AccountState() (current)
	// here showed a trader the WRONG account's equity when another trader/panel
	// owned the connection (a display AND a risk-sizing bug). Prefer the bound
	// account's own snapshot; fall back to the streamed `current` ONLY when the
	// bound account has no snapshot yet (AddOn hasn't streamed it), so this never
	// regresses to zeros vs today's behavior. The returned "account" field names
	// which NT account these numbers actually reflect.
	acct, ok := t.server.AccountStateFor(t.boundAccount)
	if !ok || t.boundAccount == "" {
		acct, ok = t.server.AccountState()
	}
	if ok {
		// Plan 4 Stage 4 — notify parent AutoTrader that balance has arrived
		// (used by defer-until-balance guard in runCycle).
		// Use reflection to avoid circular import (trader/ninjatrader → trader).
		if t.parentAutoTrader != nil {
			if method := reflect.ValueOf(t.parentAutoTrader).MethodByName("SetHasReceivedBalance"); method.IsValid() {
				method.Call([]reflect.Value{reflect.ValueOf(true)})
			}
		}

		equity := acct.NetLiquidation
		if equity == 0 {
			equity = acct.CashValue + acct.UnrealizedPnL
		}
		avail := acct.BuyingPower
		if avail == 0 {
			avail = acct.CashValue
		}
		return map[string]interface{}{
			"totalEquity":           equity,
			"availableBalance":      avail,
			"totalWalletBalance":    acct.CashValue,
			"totalUnrealizedProfit": acct.UnrealizedPnL,
			// Which NT account these numbers actually reflect (bound account when
			// its snapshot exists, else the streamed current — see the decouple
			// above). Lets the dashboard label/guard the balance accurately.
			"account": acct.Account,
			// Issue 2B — NT reports its own realized/unrealized P&L per account.
			// brokerNativePnL signals GetAccountInfo to use realized+unrealized as
			// the displayed P&L instead of (equity - global InitialBalance), which
			// otherwise produces a fake % on any account whose real starting equity
			// differs from the trader's single 100000 baseline.
			"totalRealizedProfit": acct.RealizedPnL,
			"brokerNativePnL":     true,
		}, nil
	}
	return map[string]interface{}{
		"totalEquity":      0.0,
		"availableBalance": 0.0,
	}, nil
}

// IsFeedConnected reports whether the NT8 price feed is usable (delegates to the
// TCP server's latest feed_status; default-allow until the first frame arrives).
func (t *TCPTrader) IsFeedConnected() bool { return t.server.IsFeedConnected() }

// FeedStatus returns the latest NT8 price-feed status string ("" until the first
// feed_status frame).
func (t *TCPTrader) FeedStatus() string { return t.server.FeedStatus() }

// IsConnected reports whether the NT8 C# AddOn's TCP socket is currently connected
// — the RAW link, independent of feed/market-data status. The dead-man watchdog
// (B5) gates NEW entries on this: a dropped link blocks entries until a clean
// positions/orders reconciliation.
func (t *TCPTrader) IsConnected() bool { return t.server.IsConnected() }

func (t *TCPTrader) GetPositions() ([]map[string]interface{}, error) {
	// Read positions for THIS trader's OWN bound account — NOT the shared connection
	// CurrentAccount() (which the dashboard can switch for DISPLAY). Decoupling keeps
	// each running trader's monitoring — drawdown (auto_trader_risk.go), decision
	// context (auto_trader_loop.go) — reading its own account even while another
	// account is being VIEWED. Empty boundAccount does NOT fall back to current (that
	// would re-open the ef550df7 hole): PositionsFor("") returns !ok and we use the
	// fill-derived cache below (an unbound trader refuses to trade anyway). Still
	// reflects positions opened MANUALLY in NT8 (the AddOn emits a `positions`
	// snapshot on select / connect / PositionUpdate).
	acct := t.boundAccount
	if snap, ok := t.server.PositionsFor(acct); ok {
		// NT8-truth uPnL: the account_balance frame carries the account's LIVE
		// unrealized P&L. When exactly ONE position is open, that total IS this
		// position's uPnL — use it (and derive the mark) so the displayed P&L
		// matches NT8, instead of marking off a stale 5m bar close. With multiple
		// positions the account total can't be split per-position, so those fall
		// back to the mark calc.
		nonFlat := 0
		for _, p := range snap {
			if p.Quantity != 0 {
				nonFlat++
			}
		}
		var uPnLOverride *float64
		if nonFlat == 1 {
			if bal, ok := t.server.AccountStateFor(acct); ok {
				v := bal.UnrealizedPnL
				uPnLOverride = &v
			}
		}
		out := make([]map[string]interface{}, 0, nonFlat)
		for _, p := range snap {
			if p.Quantity == 0 {
				continue // flat — should not appear in a snapshot, but be safe
			}
			out = append(out, t.positionMap(p.Symbol, p.Side, float64(p.Quantity), p.AvgPrice, uPnLOverride))
		}
		return out, nil
	}

	// Fallback (no NT8 snapshot yet): the single fill-derived position.
	t.mu.Lock()
	if !t.hasFill {
		t.mu.Unlock()
		return []map[string]interface{}{}, nil
	}
	fill := t.lastFill
	t.mu.Unlock()
	return []map[string]interface{}{
		t.positionMap(t.symbol, fill.Side, float64(fill.Quantity), fill.FillPrice, nil),
	}, nil
}

// positionMap builds the UI/decision-context record for one open position, with
// uPnL marked off the latest cached bar and the futures point value. Shared by
// the NT8-snapshot path (truth) and the fill-derived fallback so both produce
// the identical shape.
func (t *TCPTrader) positionMap(symbol, side string, qty, entry float64, uPnLOverride *float64) map[string]interface{} {
	sd := upperSideStr(side)

	// Futures: contract point value (MNQ=$2/pt). positionAmt is signed (short < 0)
	// per the Binance convention GetAccountInfo expects.
	pv := market.FuturesPointValue(symbol)
	if pv <= 0 {
		pv = 1
	}
	dir := 1.0
	signedQty := qty
	if sd == "SHORT" {
		dir = -1.0
		signedQty = -qty
	}

	var mark, uPnL, uPnLPct float64
	if uPnLOverride != nil {
		// NT8-truth uPnL (account_balance UnrealizedPnL) — already correctly signed
		// by NT8. Derive the mark so the displayed "Current" is consistent with it:
		// long uPnL=(mark-entry)*qty*pv → mark=entry+uPnL/(qty*pv); short →
		// mark=entry-uPnL/(qty*pv); i.e. mark = entry + dir*uPnL/(qty*pv).
		uPnL = *uPnLOverride
		mark = entry
		if qty > 0 {
			mark = entry + dir*uPnL/(qty*pv)
		}
		if entry > 0 {
			uPnLPct = (mark - entry) / entry * 100 * dir
		}
	} else {
		// Fallback (multi-position / no account snapshot yet): mark off the latest
		// 5m BarCache close. Less precise than NT8's live uPnL but self-contained.
		mark = entry
		if bars := t.server.BarCache().Get(symbol, "5m"); len(bars) > 0 {
			mark = bars[len(bars)-1].C
		}
		uPnL = (mark - entry) * dir * qty * pv
		if entry > 0 {
			uPnLPct = (mark - entry) / entry * 100 * dir
		}
	}

	return map[string]interface{}{
		// snake_case — read by the UI Position type (web/src/types/trading.ts)
		"symbol":             symbol,
		"side":               sd,
		"entry_price":        entry,
		"mark_price":         mark,
		"quantity":           qty,
		"leverage":           1.0, // futures: margin is per-contract, not crypto leverage
		"unrealized_pnl":     uPnL,
		"unrealized_pnl_pct": uPnLPct,
		"liquidation_price":  0.0,
		"margin_used":        0.0,
		// camelCase — read by AutoTrader.GetAccountInfo (Binance-style margin calc)
		"positionAmt":      signedQty,
		"entryPrice":       entry,
		"markPrice":        mark,
		"unRealizedProfit": uPnL,
	}
}

// DebugPlaceTestTrade places a deterministic 1-contract bracket order on the
// resolved SIM account for end-to-end proof (Plan 4.11 dashboard: signal →
// NT8 SIM fill → position). It BYPASSES the AI + risk gate — a debug harness,
// NOT a trading path — and is reachable only via the SIM-gated
// /api/debug/nt-test-trade endpoint. SL/TP are priced off the latest cached
// MNQ bar (20pt stop / 40pt target → ~2:1). side = "short" else long.
// DebugUnsubscribeBars drops an EXTRA bar subscription at runtime (P5.1
// dispose-one proof / P5.3 building block): removes the root from the
// auto-subscribe list and sends bars_unsubscribe so the AddOn disposes that
// root's BarsRequests. The primary (trading) symbol is refused server-side.
func (t *TCPTrader) DebugUnsubscribeBars(symbol string) error {
	return t.server.UnsubscribeBarsSymbol(symbol)
}

// P5.3 — runtime symbol management for the owner API (flag-gated upstream).
// Bars-only: extras never touch the trading/gate/order paths (P5.4 lifts that).

// RuntimeSubscribeBars adds an extra bar-subscription root NOW (no restart).
func (t *TCPTrader) RuntimeSubscribeBars(symbol string) error {
	return t.server.SubscribeBarsSymbol(symbol)
}

// RuntimeUnsubscribeBars removes an extra root NOW (primary refused) + purges
// its cached bars.
func (t *TCPTrader) RuntimeUnsubscribeBars(symbol string) error {
	return t.server.UnsubscribeBarsSymbol(symbol)
}

// BarsSubscriptionStates exposes the per-symbol subscription lifecycle
// (pending/subscribed/error/unsubscribed, with contract/reason).
func (t *TCPTrader) BarsSubscriptionStates() map[string]ntwire.SymbolSubState {
	return t.server.BarsSubscriptionStates()
}

// BarsPrimarySymbol is the trading symbol (the root that can never be
// unsubscribed at runtime).
func (t *TCPTrader) BarsPrimarySymbol() string {
	return t.server.BarsSubscribeSymbol()
}

func (t *TCPTrader) DebugPlaceTestTrade(side string) (map[string]interface{}, error) {
	bars := t.server.BarCache().Get(t.symbol, "5m")
	if len(bars) == 0 {
		return nil, fmt.Errorf("ninjatrader/tcp: no %s bars cached; cannot price a test trade (recompile/connect NT8 first)", t.symbol)
	}
	price := bars[len(bars)-1].C
	tick := InstrumentTickSize(t.symbol)
	if side == "short" {
		_ = t.SetStopLoss(t.symbol, "short", 1, RoundToTick(price+20, tick))
		_ = t.SetTakeProfit(t.symbol, "short", 1, RoundToTick(price-40, tick))
		return t.OpenShort(t.symbol, 1, 1)
	}
	_ = t.SetStopLoss(t.symbol, "long", 1, RoundToTick(price-20, tick))
	_ = t.SetTakeProfit(t.symbol, "long", 1, RoundToTick(price+40, tick))
	return t.OpenLong(t.symbol, 1, 1)
}

// BarCache exposes the live NT8 bar cache for the Stage 4 SSE chart relay.
// Read-only access; the relay polls Get(symbol, timeframe) for snapshots +
// incremental updates.
func (t *TCPTrader) BarCache() *ntwire.BarCache { return t.server.BarCache() }

// GetServer exposes the underlying TCP server for API handlers (e.g., account selection).
// Used by the /api/accounts and /api/account/select handlers to interact with the NT AddOn.
func (t *TCPTrader) GetServer() *ntwire.TCPServer { return t.server }

// BoundAccount returns the NT sub-account THIS trader is bound to (P5.4), or ""
// if unbound. Unlike server.CurrentAccount() (the shared, streamed/display
// account that flaps to whichever account's frame arrived last), this is the
// trader's OWN account — so record/stat/equity attribution stamps the correct
// owner even while another account is being streamed/viewed. It is the missed
// twin of the GetBalance/GetPositions/reconcile decouples.
func (t *TCPTrader) BoundAccount() string { return t.boundAccount }

// flattenKey is the "SYMBOL|SIDE" key for closedAt (upper-cased, trimmed).
func flattenKey(symbol, side string) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.ToUpper(strings.TrimSpace(side))
}

// MarkCloseConfirmed records that a FILL-CONFIRMED close (position_close frame)
// arrived for (symbol, side) on this trader's bound account. Called from close-sync
// on every position_close. This is the fast, account-correct flat signal the
// reconcile-before-open gate awaits (the frame arrives on the flatten fill even for
// a non-active account, unlike the positions snapshot which lags to the 30s
// heartbeat).
func (t *TCPTrader) MarkCloseConfirmed(symbol, side string) {
	t.mu.Lock()
	if t.closedAt == nil {
		t.closedAt = make(map[string]int64)
	}
	t.closedAt[flattenKey(symbol, side)] = time.Now().UnixMilli()
	t.mu.Unlock()
}

// CloseConfirmedSince reports whether a fill-confirmed close for (symbol, side)
// arrived at or after sinceMs — the frame-path flat confirmation for reconcile.
func (t *TCPTrader) CloseConfirmedSince(symbol, side string, sinceMs int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closedAt != nil && t.closedAt[flattenKey(symbol, side)] >= sinceMs
}

// ResetAccountState clears cached fill/position state when switching accounts.
// Called by the /api/account/select handler to ensure GetPositions() fetches fresh
// data from the newly selected account (not stale cached fills from the old account).
func (t *TCPTrader) ResetAccountState() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hasFill = false
	t.lastFill = ntwire.FillPayload{}
	t.stopLoss = map[string]float64{}
	t.takePrft = map[string]float64{}
	t.pending = map[string]string{}
}

// SetParentAutoTrader sets the parent AutoTrader reference (Plan 4 Stage 4).
// Called by transport.go after creating the TCPTrader. Used to notify when
// the first account_balance frame arrives (defer-until-balance guard).
func (t *TCPTrader) SetParentAutoTrader(parent interface{}) {
	t.parentAutoTrader = parent
}

func (t *TCPTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	return fmt.Sprintf("%.0f", quantity), nil
}

func (t *TCPTrader) GetOrderStatus(symbol, orderID string) (map[string]interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Report a fill ONLY when it belongs to the current entry (correlate by the
	// echoed signal_id). This guards against a stale fill from a prior trade and
	// against the entry fill leaking into a close-path poll (lastEntrySignalID is
	// cleared on close). Until the matching fill arrives → "pending".
	if !t.hasFill || t.lastEntrySignalID == "" || t.lastFill.SignalID != t.lastEntrySignalID {
		return map[string]interface{}{"status": "pending"}, nil
	}
	// Canonical shape expected by recordAndConfirmOrder (auto_trader_decision.go)
	// AND pollAndUpdateOrderStatus (handler_trader_status.go): status "FILLED"
	// (uppercase) + key "avgPrice". The AddOn reports the real broker fill
	// (AverageFillPrice) as lastFill.FillPrice — THIS is the NT8-truth entry that
	// replaces the stale marketData.CurrentPrice (5m-mark) reference. Previously
	// this returned "filled"/"price", which neither poller matched, so the entry
	// was never upgraded from the frozen mark.
	status := "FILLED"
	if t.lastFill.Status == "rejected" {
		status = "REJECTED"
	}
	return map[string]interface{}{
		"status":      status,
		"avgPrice":    t.lastFill.FillPrice,
		"executedQty": float64(t.lastFill.Quantity),
		"price":       t.lastFill.FillPrice, // back-compat (prior key)
		"side":        upperSideStr(t.lastFill.Side),
	}, nil
}

func (t *TCPTrader) GetClosedPnL(start time.Time, limit int) ([]types.ClosedPnLRecord, error) {
	return nil, nil
}

func (t *TCPTrader) GetMarketPrice(symbol string) (float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.hasFill {
		return 0, fmt.Errorf("ninjatrader/tcp: no fill yet, market price unavailable; use Databento client directly")
	}
	return t.lastFill.FillPrice, nil
}

func (t *TCPTrader) CancelStopLossOrders(symbol string) error {
	return fmt.Errorf("ninjatrader/tcp: CancelStopLossOrders not supported — SL is set at entry")
}

func (t *TCPTrader) CancelTakeProfitOrders(symbol string) error {
	return fmt.Errorf("ninjatrader/tcp: CancelTakeProfitOrders not supported — TP is set at entry")
}

func (t *TCPTrader) CancelStopOrders(symbol string) error {
	return fmt.Errorf("ninjatrader/tcp: CancelStopOrders not supported")
}

func (t *TCPTrader) GetOpenOrders(symbol string) ([]types.OpenOrder, error) {
	return []types.OpenOrder{}, nil
}

// upperSideStr normalises "long"/"LONG"/"Long" → "LONG" for SL/TP map keys.
// CSV Trader keys are uppercase, so we mirror that to keep behaviour parallel.
func upperSideStr(side string) string {
	switch side {
	case "long", "LONG", "Long":
		return "LONG"
	case "short", "SHORT", "Short":
		return "SHORT"
	default:
		return side
	}
}
