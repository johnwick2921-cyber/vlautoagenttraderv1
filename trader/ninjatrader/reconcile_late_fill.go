package ninjatrader

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"nofx/logger"
	"nofx/market"
	"nofx/store"
)

// ── W1b E15 — a late fill materialized through the untracked path is tagged by
// the signal of its OWN fill ─────────────────────────────────────────────────
//
// CLASS 160: an NT8 AI open whose fill misses the ~3 s poll leaves an order row
// keyed by its signal (status NEW) and NO position; NT8's position is later
// materialized by reconcilePositions' untracked branch. That branch used to
// recover identity only by PRICE (StampArmedLineageIfMatched: any filled arm of
// this trader within one tick, no time bound) — so a late AI fill was anonymous
// or adopted an old arm's plan and signal, while the fill ring held the exact
// signal. The ring is consulted FIRST now:
//
//   - exactly one same-side entry signal in the window, not already some
//     position row's entry_order_id → the position is tagged with it: from its
//     own arm (the armed ledger row with that signal), or from this trader's
//     signal-keyed AI order row (which settles FILLED at the ring's price);
//   - more than one → AMBIGUOUS: untagged, WARN — never a guess;
//   - a signal that is neither this trader's arm nor this trader's AI open, or
//     a store read that fails → untagged, WARN (fail-closed);
//   - no same-side evidence in the window (the ring emptied by a restart, or
//     every candidate already explains another position) → the price-match
//     fallback runs, bounded to the SAME window (FOLD-4): only this trader's
//     FILLED arms whose updated_at is at or after firstSeen −
//     lateEntryFillWindowMs and whose signal explains no position yet; an
//     older arm, or one already in use, never matches.
//
// The AddOn sends fill frames only for ENTRY legs (Buy=long, SellShort=short),
// so the ring never hands an exit's signal to a new entry.

// lateFillVerdict is the ring's answer for one untracked position.
type lateFillVerdict int

const (
	lateFillNoEvidence lateFillVerdict = iota // the window-bounded price-match fallback may run
	lateFillOne                               // exactly one candidate — tag it
	lateFillUnresolved                        // ambiguous or unreadable — untagged, no guess
)

// lateEntryFillWindowMs: how far before the first untracked sighting an entry
// fill may land and still be the one that opened the position — the pending
// confirmation grace plus the untracked grace.
const lateEntryFillWindowMs = entryConfirmGraceMs + untrackedGraceMs

func fillRoot(s string) string {
	f := strings.Fields(strings.ToUpper(strings.TrimSpace(s)))
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// lateEntryFillFor resolves the ring's entry evidence for a held position on
// (acct, sym, side) first sighted untracked at firstSeenMs.
func (t *TCPTrader) lateEntryFillFor(st *store.Store, acct, sym, side string, firstSeenMs int64) (recentFill, lateFillVerdict, string) {
	lo := firstSeenMs - lateEntryFillWindowMs
	root := fillRoot(sym)
	bySig := map[string]recentFill{}
	t.mu.Lock()
	for _, f := range t.recentFills {
		if strings.TrimSpace(f.SignalID) == "" || f.TimeMs < lo || !strings.EqualFold(f.Side, side) {
			continue
		}
		if f.Symbol != "" && fillRoot(f.Symbol) != root {
			continue
		}
		if f.Account != "" && acct != "" && !strings.EqualFold(f.Account, acct) {
			continue
		}
		if prev, ok := bySig[f.SignalID]; !ok || f.TimeMs >= prev.TimeMs {
			bySig[f.SignalID] = f
		}
	}
	t.mu.Unlock()
	if len(bySig) == 0 {
		return recentFill{}, lateFillNoEvidence, ""
	}
	sigs := make([]string, 0, len(bySig))
	for sid := range bySig {
		sigs = append(sigs, sid)
	}
	sort.Strings(sigs)
	var cands []string
	for _, sid := range sigs {
		used, err := st.Position().EntryOrderIDInUse(sid)
		if err != nil {
			return recentFill{}, lateFillUnresolved, "entry-identity read failed: " + err.Error()
		}
		if !used {
			cands = append(cands, sid)
		}
	}
	switch len(cands) {
	case 0:
		// Every same-side fill in the window already explains another
		// position: the ring holds no evidence about THIS one.
		return recentFill{}, lateFillNoEvidence, ""
	case 1:
		return bySig[cands[0]], lateFillOne, ""
	}
	return recentFill{}, lateFillUnresolved, "ambiguous: " + strings.Join(cands, ", ")
}

// stampArmedLineageInWindow is the untracked materialization's price-match
// fallback (W1b FOLD-4): StampArmedLineageIfMatched's one-tick match, over
// this trader's FILLED arms whose updated_at is at or after firstSeenMs −
// lateEntryFillWindowMs — the ring's own window, no upper bound. An older arm
// never matches: a fill at an old arm's price adopting that arm's plan and
// signal is fabricated lineage, and the terminal signal it would cache sends
// move_stop to a dead order. updated_at is zone-bearing text
// (store.LedgerClockSlack): the SQL bound only over-fetches; the window is
// judged here on the parsed instant and candidates are ordered newest instant
// first. An in-window arm whose signal is already some position row's
// entry_order_id is skipped — E15's invariant binds the fallback too: a signal
// that already explains one position never tags a second (the CTO's path, every
// ring candidate in use, reaches here with exactly such an arm). A read failure
// leaves the row untagged (WARN).
//
// W1b FOLD-6 — failed is non-empty when the fallback could NOT answer (the read
// failed, or an in-window match could not be stamped): the 🧩 line must then
// say UNTAGGED, never "no evidence". ("", "", false) with failed == "" is the
// established answer "no in-window arm of this trader matched".
func stampArmedLineageInWindow(st *store.Store, traderID string, posID int64, sym, side string, entryPx float64, firstSeenMs int64) (stamped bool, sig, failed string) {
	lo := firstSeenMs - lateEntryFillWindowMs
	rows, err := st.ArmedOrders().ListFilledSince(traderID, time.UnixMilli(lo))
	if err != nil {
		logger.Warnf("🔗 attribution: pos %d (%s %s) — armed-fill read for the price-match fallback failed: %v — left UNTAGGED (no guess)", posID, sym, side, err)
		return false, "", "armed-fill read for the price-match fallback failed — see the 🔗 WARN"
	}
	inWindow := make([]store.ArmedOrderDB, 0, len(rows))
	for _, r := range rows {
		if r.UpdatedAt.UnixMilli() < lo {
			continue
		}
		used, uerr := st.Position().EntryOrderIDInUse(r.SignalID)
		if uerr != nil {
			logger.Warnf("🔗 attribution: pos %d (%s %s) — entry-identity read for armed #%d failed: %v — left UNTAGGED (no guess)", posID, sym, side, r.ID, uerr)
			return false, "", "entry-identity read for the price-match fallback failed — see the 🔗 WARN"
		}
		if used {
			continue
		}
		inWindow = append(inWindow, r)
	}
	sort.SliceStable(inWindow, func(i, j int) bool { return inWindow[i].UpdatedAt.After(inWindow[j].UpdatedAt) })
	if r, ok := matchArmedFillByPrice(inWindow, sym, side, entryPx); ok {
		if ok, got := stampArmedLineageFromRow(st, posID, r); ok {
			return true, got, ""
		}
		return false, "", fmt.Sprintf("armed #%d matched by price in the window but its lineage stamp failed — see the 🧩 WARN", r.ID)
	}
	return false, "", ""
}

// tagLateEntryFill stamps the materialized position posID with the identity of
// its own fill f. Returns the signal stamped ("" when it stays untagged) and
// (W1b FOLD-6) what the attribution step established — the origin the 🧩
// MATERIALIZED line names: what the position WAS when tagged, or, untagged,
// why (a refusal, or a store read/write that FAILED — never worded as a
// refusal, which it did not establish).
func (t *TCPTrader) tagLateEntryFill(st *store.Store, traderID, exchangeID string, posID int64, sym, side string, f recentFill) (string, string) {
	sid := f.SignalID
	arm, err := st.ArmedOrders().FindBySignal(traderID, sid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Warnf("🔗 attribution: pos %d (%s %s) — armed lookup for fill signal %s failed: %v — left UNTAGGED (no guess)", posID, sym, side, sid, err)
		return "", "armed lookup failed — see the 🔗 WARN"
	}
	if arm != nil {
		if !strings.EqualFold(arm.Side, side) {
			logger.Warnf("🔗 attribution: pos %d (%s %s) — fill signal %s is armed row #%d of the OTHER side (%s) — left UNTAGGED (no guess)", posID, sym, side, sid, arm.ID, arm.Side)
			return "", fmt.Sprintf("is armed #%d of the OTHER side (%s)", arm.ID, arm.Side)
		}
		if ok, got := stampArmedLineageFromRow(st, posID, *arm); ok {
			logger.Infof("🔗 attribution: late armed fill — pos %d ← signal %s (armed #%d, from the fill ring; no price match)", posID, got, arm.ID)
			return got, fmt.Sprintf("this trader's late armed fill (signal %s, armed #%d)", got, arm.ID)
		}
		return "", fmt.Sprintf("is this trader's armed #%d but its lineage stamp failed — see the 🧩 WARN", arm.ID)
	}
	o, err := st.Order().GetOrderByExchangeID(exchangeID, sid)
	if err != nil {
		logger.Warnf("🔗 attribution: pos %d (%s %s) — order lookup for fill signal %s failed: %v — left UNTAGGED (no guess)", posID, sym, side, sid, err)
		return "", "order lookup failed — see the 🔗 WARN"
	}
	if o == nil || o.TraderID != traderID || o.OrderAction != "open_"+strings.ToLower(side) {
		logger.Warnf("🔗 attribution: pos %d (%s %s) — fill signal %s is neither an arm nor an AI %s open of this trader — left UNTAGGED (no guess)", posID, sym, side, sid, strings.ToLower(side))
		return "", fmt.Sprintf("is neither an arm nor an AI %s open of this trader", strings.ToLower(side))
	}
	// W1b E15 repair — only an UNRESOLVED (NEW) AI open is the CLASS 160 case
	// this path recovers. A row already FILLED / REJECTED / CANCELED is not
	// this position's entry (e.g. its fill was recorded, or it netted a
	// position flat): untagged, never re-settled.
	if o.Status != "NEW" {
		logger.Warnf("🔗 attribution: pos %d (%s %s) — fill signal %s is AI order #%d already %s (not an unresolved open) — left UNTAGGED (no guess)", posID, sym, side, sid, o.ID, o.Status)
		return "", fmt.Sprintf("is AI order #%d already %s, not an unresolved open", o.ID, o.Status)
	}
	if err := st.Position().SetEntryOrderID(posID, sid); err != nil {
		logger.Warnf("🔗 attribution: pos %d entry-order-id stamp failed (signal %s): %v", posID, sid, err)
		return "", fmt.Sprintf("is this trader's AI order #%d but the entry-order-id stamp failed — see the 🔗 WARN", o.ID)
	}
	qty := f.Quantity
	if qty <= 0 {
		qty = st.Position().QuantityOf(posID)
	}
	if err := st.Order().UpdateOrderStatus(o.ID, "FILLED", qty, f.Price, 0); err != nil {
		logger.Warnf("🔗 attribution: order #%d FILLED settle failed (signal %s): %v", o.ID, sid, err)
	} else if err := st.Order().CreateFill(lateEntryFillRow(o, traderID, exchangeID, sym, side, f.Price, qty, f.TimeMs)); err != nil {
		logger.Warnf("🔗 attribution: order #%d settled FILLED but its fill row failed (signal %s): %v", o.ID, sid, err)
	}
	logger.Infof("🔗 attribution: late AI fill — pos %d ← signal %s (order #%d, fill %.2f); plan citation stays %s", posID, sid, o.ID, f.Price, store.PlanUnresolvable)
	return sid, fmt.Sprintf("this trader's late AI fill (signal %s, order #%d)", sid, o.ID)
}

// lateEntryFillRow is the trader_fills row the normal CLASS 160 path writes
// for an NT8 AI open (recordOrderFill: same order, signal, side, symbol,
// price, quantity, zero fee on NT8), so an order this path settles FILLED
// never reads FILLED with no fill row. The trade id is keyed on the order row
// — one order, one row — so a repeated pass cannot double-count (CreateFill
// dedupes on it). CreatedAt is the fill's own time from the ring.
func lateEntryFillRow(o *store.TraderOrder, traderID, exchangeID, sym, side string, price, qty float64, fillMs int64) *store.TraderFill {
	fillSide := "BUY"
	if strings.EqualFold(side, "SHORT") {
		fillSide = "SELL"
	}
	if fillMs <= 0 {
		fillMs = time.Now().UTC().UnixMilli()
	}
	return &store.TraderFill{
		TraderID:        traderID,
		ExchangeID:      exchangeID,
		ExchangeType:    o.ExchangeType,
		OrderID:         o.ID,
		ExchangeOrderID: o.ExchangeOrderID,
		ExchangeTradeID: fmt.Sprintf("nt8-late-entry-%d", o.ID),
		Symbol:          market.Normalize(sym),
		Side:            fillSide,
		Price:           price,
		Quantity:        qty,
		QuoteQuantity:   price * qty,
		Commission:      0,
		CommissionAsset: "USDT",
		RealizedPnL:     0,
		IsMaker:         false,
		CreatedAt:       fillMs,
	}
}
