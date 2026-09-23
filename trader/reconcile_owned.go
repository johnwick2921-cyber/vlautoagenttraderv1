package trader

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"nofx/store"
	"nofx/telemetry"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W0 (c) — the pre-open reconcile never flattens a position a
// ledger explains ─────────────────────────────────────────────────────────────
//
// reconcileBeforeOpenNT flattened ANY position NT8 held before an AI open,
// reading neither trader_positions nor either ledger — so an armed or Picture
// fill not yet materialized (the reconciler's untracked grace) was flattened as
// an "orphan", with its bracket, and the AI opened its own (D10). A position is
// EXPLAINED — and the AI entry refused, named, as a ⛔ gate — when on this
// account, instrument and side there is:
//
//	(i)   a placed non-terminal ledger row: an armed row carrying a broker
//	      signal, or a Picture row working / stamped (a send started)
//	(ii)  an OPEN trader_positions row whose entry_order_id is a ledger signal
//	(iii) a ledger FILL younger than twice the untracked grace — the
//	      reconciler has not materialized it yet
//
// An UNEXPLAINED position keeps today's owner-ruled flatten (TRACK B).

// errPositionOwned marks the refusal: the held position belongs to a ledger.
var errPositionOwned = errors.New("position owned by a ledger row")

// reconcileRefusal maps reconcileBeforeOpenNT's outcome for the open paths: a
// ledger-owned position is a ⛔ REFUSAL (counted, stamped on the record,
// returns nil — the decision is classified refused, not failed); anything
// else stays the error it was.
func (at *AutoTrader) reconcileRefusal(err error, rec *store.DecisionAction) error {
	if !errors.Is(err, errPositionOwned) {
		return err
	}
	telemetry.IncGateBlock(at.id, "reconcile_owned")
	if rec != nil {
		rec.Success = false
		rec.Error = "reconcile_owned: " + err.Error()
	}
	return nil
}

// ledgerExplainsPosition reports the ledger row that explains a held position
// on this trader's account and symbol for side ("long"/"short"), if any.
func (at *AutoTrader) ledgerExplainsPosition(symbol, side string, now time.Time) (string, bool) {
	if at.store == nil {
		return "", false
	}
	acct, _ := at.latchScope()
	root := instrumentRoot(symbol)
	window := 2 * time.Duration(ntTrader.UntrackedGraceMs) * time.Millisecond
	onAccount := func(traderID string) bool {
		a, s, ok := at.traderScope(traderID)
		return !ok || (strings.EqualFold(a, acct) && instrumentRoot(s) == root)
	}
	signals := map[string]string{}

	// (i) armed: non-terminal rows that carry a broker signal.
	if rows, err := at.store.ArmedOrders().ListNonTerminalAllTraders(); err == nil {
		for _, r := range rows {
			if strings.TrimSpace(r.SignalID) == "" || store.IsTerminalArmState(r.State) || positionSide(r.Side) != side || !onAccount(r.TraderID) {
				continue
			}
			return fmt.Sprintf("armed #%d %s %s (%s, signal %s)", r.ID, r.Scenario, side, r.State, r.SignalID), true
		}
	}
	// (i) Picture: working, or place_pending with a submission stamp.
	if rows, err := at.store.PictureHtfRecoverableAll(); err == nil {
		for _, p := range rows {
			if !store.PictureSendStarted(p) || positionSide(p.Direction) != side || !pictureRowOnAccount(p, acct, root) {
				continue
			}
			return fmt.Sprintf("picture %s %s (%s)", p.OppKey, side, p.Stage), true
		}
	}
	// (iii) recent fills, not yet materialized — this trader and every
	// running trader in the process registry.
	ids := []string{at.id}
	pictureHtfTraders.Range(func(k, _ any) bool {
		if id, _ := k.(string); id != "" && id != at.id {
			ids = append(ids, id)
		}
		return true
	})
	for _, id := range ids {
		if !onAccount(id) {
			continue
		}
		if rows, err := at.store.ArmedOrders().ListFilled(id, 20); err == nil {
			for _, r := range rows {
				signals[r.SignalID] = fmt.Sprintf("armed #%d", r.ID)
				if positionSide(r.Side) == side && now.Sub(r.UpdatedAt) >= 0 && now.Sub(r.UpdatedAt) < window {
					return fmt.Sprintf("armed #%d %s %s filled %s ago (not yet materialized)", r.ID, r.Scenario, side, now.Sub(r.UpdatedAt).Round(time.Second)), true
				}
			}
		}
		if rows, err := at.store.PictureHtfByTrader(id, 20); err == nil {
			for _, p := range rows {
				if p.SignalID != "" {
					signals[p.SignalID] = "picture " + p.OppKey
				}
				if p.Stage == store.StateFilled && positionSide(p.Direction) == side && pictureRowOnAccount(p, acct, root) &&
					now.Sub(p.UpdatedAt) >= 0 && now.Sub(p.UpdatedAt) < window {
					return fmt.Sprintf("picture %s %s filled %s ago (not yet materialized)", p.OppKey, side, now.Sub(p.UpdatedAt).Round(time.Second)), true
				}
			}
		}
	}
	// (ii) an OPEN position row whose entry order is a ledger signal.
	if opens, err := at.store.Position().GetOpenPositions(at.id); err == nil {
		for _, op := range opens {
			if owner, ok := signals[op.EntryOrderID]; ok && op.EntryOrderID != "" && positionSide(op.Side) == side && instrumentRoot(op.Symbol) == root {
				return fmt.Sprintf("position #%d from %s", op.ID, owner), true
			}
		}
	}
	return "", false
}

func pictureRowOnAccount(p store.PictureHtfOpportunityDB, acct, root string) bool {
	if strings.TrimSpace(p.Account) != "" && !strings.EqualFold(p.Account, acct) {
		return false
	}
	return strings.TrimSpace(p.Symbol) == "" || instrumentRoot(p.Symbol) == root
}
