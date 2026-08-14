package ninjatrader

import (
	"fmt"
	"strings"
	"time"

	"nofx/discipline"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
)

// Position reconcile — the durable single-source-of-truth anchor for NT8.
//
// The AI-decision write records entry_price = marketData.CurrentPrice (the last
// 5m BarCache close), a decision-time *reference* that goes stale (and freezes
// entirely when NT8 stops streaming bars). This loop periodically compares the
// open trader_positions rows against the NT8 positions snapshot (the truth) and:
//
//	(b) ENTRY TRUTH  — for a row NT8 still holds, replaces a stale entry_price
//	    with the NT8 position average (Position.AveragePrice). close-sync's
//	    realized-PnL formula is unchanged; it simply consumes a correct entry.
//	ORPHAN CLEAR     — for an OPEN row NT8 reports FLAT, marks it closed so the
//	    phantom clears and GetPositions / the risk gate agree with NT8.
//
// SAFETY: acts ONLY on a positive snapshot (PositionsFor ok == true). When NT8
// has not reported positions (disconnected / not-yet-deployed), it does NOTHING
// — never closing a real position on missing data. A grace window also exempts
// freshly-opened rows whose positions frame may not have arrived yet, and rows
// are only judged against the account NT8 is currently reporting.
const (
	reconcileInterval = 20 * time.Second
	// orphanGraceMs: a row younger than this is not orphan-closed — its positions
	// frame (PositionUpdate) may simply not have arrived yet after the open.
	orphanGraceMs = 120_000
	// flatGraceMs: once a row is observed NT8-flat-but-DB-open, reconcile waits this
	// long before orphan-closing it. NT8 publishes a flat positions snapshot at/around
	// the same instant it sends the position_close frame, but that frame reaches
	// close-sync a beat later; orphan-closing immediately makes close-sync find no
	// open row and skip → the real ×pv P&L is lost (the reconcile-FIRST race that the
	// PART-1a status guard alone did NOT cover). 60s (3 reconcile cycles) is ample
	// headroom for the frame + close-sync, while bounding how long a genuinely-
	// uncaptured row lingers before it falls to reconcile_flat "—".
	flatGraceMs = 60_000
	// reconcileDivergenceGraceMs (A4/G4): a qty divergence (NT8 held ≠ DB belief)
	// must PERSIST this long before it freezes the trader — a fill frame can be
	// momentarily in flight, so a single-pass divergence is within tolerance. 60s
	// (3 reconcile passes) matches the other reconcile grace windows.
	reconcileDivergenceGraceMs = 60_000
)

// StartPositionReconcile launches the periodic reconcile goroutine (idempotent
// via reconcileOnce). Wired next to StartCloseSync for the ninjatrader exchange.
func (t *TCPTrader) StartPositionReconcile(traderID, exchangeID, exchangeType string, st *store.Store) {
	if st == nil {
		return
	}
	t.reconcileOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(reconcileInterval)
			defer ticker.Stop()
			for range ticker.C {
				t.reconcilePositions(traderID, st)
			}
		}()
		logger.Infof("🔧 NinjaTrader position-reconcile started (anchors entry_price to NT8 avg + clears orphan rows)")
	})
}

// reconcilePositions runs one reconcile pass. See file header for semantics.
func (t *TCPTrader) reconcilePositions(traderID string, st *store.Store) {
	// Reconcile against THIS trader's OWN bound account, not the shared connection
	// CurrentAccount() (display-switchable). Otherwise viewing another account would
	// make reconcile read the WRONG account's positions and mis-clear this trader's
	// DB rows as orphans. Empty boundAccount → PositionsFor("") !ok → early return
	// (never touches the DB), preserving the ef550df7 refuse semantics.
	acct := t.boundAccount
	snap, ok := t.server.PositionsFor(acct)
	if !ok {
		// NT8 has not reported positions for this account — do NOT touch the DB.
		return
	}

	// NT8-held positions keyed "SYMBOL|SIDE" → average price (canonical form).
	held := make(map[string]float64, len(snap))
	heldQty := make(map[string]float64, len(snap))
	for _, p := range snap {
		if p.Quantity == 0 {
			continue
		}
		side := "LONG"
		if strings.EqualFold(p.Side, "short") {
			side = "SHORT"
		}
		key := market.Normalize(p.Symbol) + "|" + side
		held[key] = p.AvgPrice
		heldQty[key] = float64(p.Quantity)
	}

	rows, err := st.Position().GetOpenPositions(traderID)
	if err != nil {
		logger.Warnf("ninjatrader/tcp: reconcile list open positions failed: %v", err)
		return
	}
	nowMs := time.Now().UTC().UnixMilli()
	if t.flatSince == nil {
		t.flatSince = make(map[int64]int64)
	}
	seenFlat := make(map[int64]bool) // rows observed flat-but-open this pass (for flatSince pruning)
	for _, row := range rows {
		// Only judge rows for the account NT8 is currently reporting (avoids
		// closing another account's positions during a transient account switch).
		if row.Account != "" && acct != "" && row.Account != acct {
			continue
		}
		key := row.Symbol + "|" + row.Side
		if avg, isHeld := held[key]; isHeld {
			// (b) Entry truth — anchor a stale entry to the NT8 average.
			if avg > 0 && row.EntryPrice != avg {
				if err := st.Position().UpdateEntryPrice(row.ID, avg); err != nil {
					logger.Warnf("ninjatrader/tcp: reconcile entry update failed (row %d): %v", row.ID, err)
				} else {
					logger.Infof("🔧 reconcile: %s %s entry %.2f → NT8 avg %.2f (row %d)",
						row.Symbol, row.Side, row.EntryPrice, avg, row.ID)
				}
			}
			// (c) QTY TRUTH — alarm when NT8's held contract count diverges from the
			// DB row. This is the id=45→id=46 net-2 tell: an unclosed prior contract
			// (rejected flatten) the next entry netted onto, so NT8 holds 2 while the
			// row says 1. Log-only (no auto-flatten): loudly flagging the divergence
			// is the safe SIM action; the close routing now prevents the phantom that
			// causes it, and the entry-anchor above keeps the price honest.
			if hq, ok := heldQty[key]; ok && row.Quantity > 0 && hq != row.Quantity {
				logger.Warnf("🚨 reconcile QTY DIVERGENCE: %s %s — NT8 holds %.0f but DB row %d has %.0f. Likely an unclosed prior contract netted onto by a later entry (id=45→46 class). DB qty NOT auto-changed — investigate.",
					row.Symbol, row.Side, hq, row.ID, row.Quantity)
				// A4 (G4) — belief≠broker divergence. Debounce across passes (a fill
				// frame may be momentarily in flight); once it PERSISTS beyond the grace,
				// FREEZE the trader so it can't open on top of a mis-believed position.
				// Open-position management continues; the owner clears via the API.
				if t.divergeSince == nil {
					t.divergeSince = make(map[int64]int64)
				}
				if t.divergeSince[row.ID] == 0 {
					t.divergeSince[row.ID] = nowMs
				} else if nowMs-t.divergeSince[row.ID] >= reconcileDivergenceGraceMs {
					reason := fmt.Sprintf("reconcile qty divergence persisted: %s %s NT8=%.0f DB row %d=%.0f", row.Symbol, row.Side, hq, row.ID, row.Quantity)
					if discipline.FreezeTrader(traderID, reason, nowMs) {
						logger.Errorf("🚨 A4 FREEZE: %s — trader FROZEN (new entries blocked; management continues). Clear via /api/risk/clear-freeze after reconciling.", reason)
					}
				}
			} else if t.divergeSince != nil {
				delete(t.divergeSince, row.ID) // qty matches → reset the debounce
			}
		} else {
			// Orphan — NT8 reports flat for this (symbol,side) but the row is OPEN.
			// Skip very fresh rows (positions frame may still be in flight).
			if nowMs-row.EntryTime < orphanGraceMs {
				continue
			}
			// FLAT-GRACE (the reconcile-FIRST race). NT8 reports this row FLAT in the
			// snapshot at/around the same instant it sends the position_close frame, but
			// that frame reaches close-sync a beat later. If we orphan-close the still-
			// OPEN row now, close-sync then finds no open row and skips → the real ×pv
			// P&L is lost. So DEFER: on first observing the row flat-but-open, record the
			// time and skip; only orphan-close after it's been continuously flat
			// ≥ flatGraceMs (close-sync gets first crack — it closes the row CLOSED/sync
			// and the next pass prunes it below). A genuinely-uncaptured row (no frame
			// ever) is still OPEN after the grace → orphaned → reconcile_flat "—".
			seenFlat[row.ID] = true
			if t.flatSince[row.ID] == 0 {
				t.flatSince[row.ID] = nowMs
				logger.Infof("⏳ reconcile: %s %s row=%d NT8-flat — deferring orphan-close (awaiting close-sync frame)", row.Symbol, row.Side, row.ID)
				continue
			}
			if nowMs-t.flatSince[row.ID] < flatGraceMs {
				continue // within grace — keep deferring to close-sync
			}
			// PRICED-FRAME FALLBACK: before writing the unknown marker, check whether a
			// position_close frame carrying the REAL exit price arrived for this
			// (account,symbol,side) but found no open row when it landed (owner-routing
			// miss / frame-before-entry race). If so, close with the REAL exit + P&L
			// (reason "sync" → counted in stats, shown as a real number) instead of a
			// fabricated exit=entry pnl=0. takePricedClose is idempotent (consumed once).
			if exitPx, okp := takePricedClose(row.Account, row.Symbol, row.Side, nowMs); okp {
				pv := market.FuturesPointValue(row.Symbol)
				if pv <= 0 {
					pv = 1
				}
				pnl := 0.0
				if row.EntryPrice > 0 {
					if row.Side == "LONG" {
						pnl = (exitPx - row.EntryPrice) * row.Quantity * pv
					} else {
						pnl = (row.EntryPrice - exitPx) * row.Quantity * pv
					}
				}
				closed, perr := st.Position().ClosePosition(row.ID, exitPx, "reconcile_priced", pnl, 0, "sync")
				delete(t.flatSince, row.ID)
				if perr != nil {
					logger.Warnf("ninjatrader/tcp: reconcile priced-close failed (row %d): %v", row.ID, perr)
				} else if closed {
					logger.Infof("💵 reconcile: recovered PARKED exit price for %s %s row=%d exit=%.2f pnl=%.2f (close-sync frame had no open row when it arrived)", row.Symbol, row.Side, row.ID, exitPx, pnl)
				}
				continue
			}
			// Past the grace and still flat+open → the position_close frame was
			// genuinely never captured. Record the honest marker (exit/P&L UNKNOWN →
			// UI "—"; entry-as-exit / 0 are placeholders). ClosePosition is guarded on
			// status='OPEN' (PART-1a), so a last-moment close-sync win is still kept.
			// Never fabricate an exit NT8 didn't give us.
			closed, err := st.Position().ClosePosition(row.ID, row.EntryPrice, "reconcile", 0, 0, store.CloseReasonReconcileFlat)
			delete(t.flatSince, row.ID)
			switch {
			case err != nil:
				logger.Warnf("ninjatrader/tcp: reconcile orphan-close failed (row %d): %v", row.ID, err)
			case closed:
				logger.Infof("🔧 reconcile: closed orphan %s %s (NT8 flat ≥%ds, no frame) row=%d", row.Symbol, row.Side, flatGraceMs/1000, row.ID)
			default:
				// Row was already closed by close-sync between the snapshot and now —
				// the status guard preserved close-sync's real ×pv P&L. No clobber.
				logger.Infof("✓ reconcile: row=%d already closed by close-sync (kept real P&L); skip", row.ID)
			}
		}
	}

	// Prune flat-timers for rows no longer orphan-candidates (close-sync closed them,
	// or NT8 re-holds them) so the map can't leak and a recycled row id starts fresh.
	for id := range t.flatSince {
		if !seenFlat[id] {
			delete(t.flatSince, id)
		}
	}
}
