package ninjatrader

import (
	"strings"
	"time"

	"nofx/logger"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// StartCloseSync consumes position_close frames from the TCP bridge and records
// each close into trader_positions (real exit price + futures realized PnL),
// then clears the in-memory fill so GetPositions reports flat.
//
// NT closes positions broker-side via the OCO bracket (SL/TP); the bot never
// issues a close_* order and — unlike every crypto broker — NT has no
// order-sync. So this is the ONLY path that transitions an open NT position to
// CLOSED, which is what populates the dashboard's position history.
func (t *TCPTrader) StartCloseSync(traderID, exchangeID, exchangeType string, st *store.Store) {
	if st == nil {
		return
	}
	pb := store.NewPositionBuilder(st.Position())
	t.closeSyncOnce.Do(func() {
		go func() {
			for p := range t.server.SubscribeClosesFor(t.symbol) { // P5.4 router-fed (per-symbol)
				t.recordClose(traderID, exchangeID, exchangeType, st, pb, p)
			}
		}()
		// Rejected exit/flatten watcher. The SIM/broker refused a close (e.g. "no
		// market data" with the feed down). The position is STILL OPEN in NT8 — do
		// NOT record a close; raise a loud alarm. Because decision-driven closes no
		// longer mark the DB CLOSED off the mark (see auto_trader_decision.go), the
		// position simply stays open: the next decision cycle re-issues the close (a
		// natural bounded retry) and the periodic reconcile keeps the DB anchored to
		// NT8 truth, so the orphan can't be netted onto by the next entry.
		go func() {
			for r := range t.server.SubscribeRejectsFor(t.symbol) { // P5.4 router-fed (per-symbol)
				logger.Warnf("🚨 NT close REJECTED: %s %s — STILL OPEN in NT8, NOT recording closed (reason: %q, account: %s). Will retry on next decision cycle / reconnect.",
					r.Symbol, r.PositionSide, r.Reason, r.Account)
			}
		}()
		// Instrument-info watcher (Phase 4): NT8 reports the RESOLVED instrument's
		// real specs. Cross-check the hardcoded tables — they are CME-correct, so a
		// divergence is a drift signal worth surfacing. For a parked/unknown symbol
		// (no table entry) NT8 is the only source. The tables stay authoritative for
		// the math; this is defense-in-depth + drift detection.
		go func() {
			for in := range t.server.SubscribeInstrumentInfoFor(t.symbol) { // P5.4 router-fed (per-symbol)
				tablePV := market.FuturesPointValue(in.Symbol)
				tableTick := market.FuturesTickSize(in.Symbol)
				switch {
				case tablePV <= 0:
					logger.Infof("📐 NT8 instrument_info %s (%s): point_value=%.4g tick=%.6g — no table entry (parked/unknown); NT8 is the source.",
						in.Symbol, in.Contract, in.PointValue, in.TickSize)
				case in.PointValue != tablePV || (tableTick > 0 && in.TickSize != tableTick):
					logger.Warnf("🚨 instrument spec DRIFT %s (%s): NT8 point_value=%.4g tick=%.6g vs table point_value=%.4g tick=%.6g — verify the table.",
						in.Symbol, in.Contract, in.PointValue, in.TickSize, tablePV, tableTick)
				default:
					logger.Infof("📐 NT8 instrument_info %s (%s): point_value=%.4g tick=%.6g — matches table ✓",
						in.Symbol, in.Contract, in.PointValue, in.TickSize)
				}
			}
		}()
		logger.Infof("🔄 NinjaTrader close-sync started (records SL/TP + manual exits; alarms on rejected flattens; cross-checks NT8 instrument specs)")
	})
}

func (t *TCPTrader) recordClose(
	traderID, exchangeID, exchangeType string,
	st *store.Store,
	pb *store.PositionBuilder,
	p ntwire.PositionClosePayload,
) {
	side, action := "LONG", "close_long"
	if strings.EqualFold(p.PositionSide, "short") {
		side, action = "SHORT", "close_short"
	}

	// The open record is keyed by the canonical bot symbol (root, e.g. "MNQ"),
	// not the resolved front-month contract — prefer t.symbol.
	symbol := t.symbol
	if symbol == "" {
		symbol = p.Symbol
	}
	qty := float64(p.Quantity)
	if qty <= 0 {
		qty = 1
	}

	// Realized PnL with the futures point value (PositionBuilder's fallback
	// formula omits it). Entry from the open record; fall back to last fill.
	entry := 0.0
	if open, err := st.Position().GetOpenPositionBySymbol(traderID, symbol, side); err == nil && open != nil {
		entry = open.EntryPrice
	}
	if entry == 0 {
		t.mu.Lock()
		entry = t.lastFill.FillPrice
		t.mu.Unlock()
	}
	pv := market.FuturesPointValue(symbol)
	if pv <= 0 {
		pv = 1
	}
	realizedPnL := 0.0
	if entry > 0 {
		if side == "LONG" {
			realizedPnL = (p.ExitPrice - entry) * qty * pv
		} else {
			realizedPnL = (entry - p.ExitPrice) * qty * pv
		}
	}

	exitMs := time.Now().UTC().UnixMilli()
	if p.ExitTime != "" {
		if ts, err := time.Parse(time.RFC3339, p.ExitTime); err == nil {
			exitMs = ts.UTC().UnixMilli()
		}
	}

	if err := pb.ProcessTrade(traderID, exchangeID, exchangeType, symbol, side, action,
		qty, p.ExitPrice, 0, realizedPnL, exitMs, p.SignalID); err != nil {
		logger.Warnf("ninjatrader/tcp: record close failed (%s %s): %v", symbol, side, err)
	} else {
		logger.Infof("📕 NT position closed: %s %s qty=%.0f exit=%.2f reason=%s pnl=%.2f",
			symbol, side, qty, p.ExitPrice, p.ExitReason, realizedPnL)
	}

	// Mark flat so GetPositions stops reporting the now-closed position.
	t.mu.Lock()
	t.hasFill = false
	t.mu.Unlock()
}
