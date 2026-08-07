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

	mu       sync.Mutex
	stopLoss map[string]float64 // key: "<symbol>:<side>"
	takePrft map[string]float64
	lastFill ntwire.FillPayload
	hasFill  bool

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
		for fill := range server.SubscribeFillsFor(symbol) {
			// P5.2 split-brain defense: a symbol-tagged fill for a DIFFERENT
			// instrument must never be attributed to this trader. Empty symbol
			// = legacy (pre-P5.2) AddOn → assumed primary (back-compat).
			if fill.Symbol != "" && !equalSymbol(fill.Symbol, symbol) {
				logger.Warnf("⚠️ ninjatrader/tcp: REJECTED fill for symbol %q (this trader trades %q) — signal_id=%s (split-brain defense)",
					fill.Symbol, symbol, fill.SignalID)
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

	// Expiry warning — defense in depth (mirrors CSV Trader).
	if days := databento.DaysUntilExpiry(symbol, time.Now()); days >= 0 && days <= 5 {
		logger.Warnf("ninjatrader/tcp: placing %s entry on %s within %d days of expiry — verify contract roll", side, symbol, days)
	}

	// CSV Trader keys SL/TP by "LONG"/"SHORT" uppercase; mirror that here.
	upperSide := upperSideStr(side)
	t.mu.Lock()
	sl := t.stopLoss[keyFor(symbol, upperSide)]
	tp := t.takePrft[keyFor(symbol, upperSide)]
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
		Side:       side,           // lowercase per spec L4390
		Quantity:   int(quantity),
		Entry:      entry,
		StopLoss:   sl,
		TakeProfit: tp,
		SignalID:   signalID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
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
func (t *TCPTrader) MoveStopToBreakeven(newStop float64) error {
	t.mu.Lock()
	sid := t.lastEntrySignalID
	t.mu.Unlock()
	if sid == "" {
		return fmt.Errorf("ninjatrader/tcp: no open entry to move the stop for %s", t.symbol)
	}
	newStop = RoundToTick(newStop, InstrumentTickSize(t.symbol))
	return t.server.SendMoveStop(ntwire.MoveStopPayload{
		Symbol:      t.symbol,
		SignalID:    sid,
		NewStopLoss: newStop,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
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
	t.mu.Unlock()

	payload := ntwire.ClosePositionPayload{
		Symbol:   t.symbol,
		Side:     side,
		Quantity: int(quantity),
		SignalID: uuid.NewString(),
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
	if acct, ok := t.server.AccountState(); ok {
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

func (t *TCPTrader) GetPositions() ([]map[string]interface{}, error) {
	// Prefer the NT8-reported open positions for the SELECTED account — the
	// source of truth. This survives account switch-back AND reflects positions
	// opened MANUALLY in NT8 (the AddOn emits a `positions` snapshot on select /
	// connect / PositionUpdate). The fill-derived cache below is only a fallback
	// for before the position-reporting AddOn is deployed (no snapshot yet).
	acct := ""
	if a := t.server.CurrentAccount(); a != nil {
		acct = *a
	}
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
