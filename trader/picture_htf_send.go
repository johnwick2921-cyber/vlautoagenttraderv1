package trader

import (
	"fmt"
	"time"

	"nofx/logger"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// PICTURE-HTF SEND (2026-09-20) — the production submission seam. The
// evaluator has already claimed the opportunity and won the atomic
// submission ownership; this function is the LAST gate before wire: it
// re-checks the live state the claim cannot see (broker identity, flat
// book, unreconciled sends, feed freshness, sizing) and then sends a market
// entry with its protective bracket through the CONCRETE NT8 trader. The
// 19-method trader/types.Trader interface is untouched.

func init() {
	pictureHtfSubmitSeam = pictureHtfSend
}

// pictureHtfContractSize is the deterministic sizing for the two-picture
// mode: 1 contract, never more than the strategy's max-contracts clamp.
// SIM-conservative by design (the owner's tape is 1-lot MNQ); the risk that
// matters here is the R:R, which the evaluator gates before admission.
func pictureHtfContractSize(at *AutoTrader) float64 {
	maxC := float64(at.resolveMaxContracts())
	if maxC < 1 {
		maxC = 1
	}
	return 1
}

func pictureHtfSend(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, now time.Time) error {
	if e == nil || e.at == nil || row == nil {
		return fmt.Errorf("picture_htf: seam called with nil state — never sending")
	}
	at := e.at

	// --- Re-check 0 (W-ONE-BUTTON M2 site 3): the installation maintenance
	// hold, FIRST — nothing below may reach the wire while an update holds.
	// Wraps ErrMaintenanceHold so the evaluator settles the row refused
	// (provably unsent) rather than ambiguous place_pending. ---
	if reason, held := MaintenanceHeld(); held {
		return fmt.Errorf("picture_htf: send refused — %s: %w", reason, ntTrader.ErrMaintenanceHold)
	}

	// --- Re-check 0b (W-EXEC-TRUTH W0 (a)): THE ONE ADMISSION GATE again,
	// immediately before the wire, on the evidence the evaluator admitted it
	// on. The row carries no submission stamp yet, so a refusal here is
	// provably unsent and the evaluator settles it refused. ---
	if refusal, refused := at.admitEntry(admitIntent{
		Path: admitPicture, Symbol: row.Symbol, Action: "open_" + row.Direction, Now: now,
		Key: row.OppKey, Price: row.EntryRef, Picture: e.pendingAdmission,
	}); refused {
		return fmt.Errorf("picture_htf: send refused — %s", refusal)
	}

	// --- Re-check 1: feed freshness at SEND time, not claim time (on the
	// evaluation's clock — W-EXEC-TRUTH W0, class 60). ---
	if now.Sub(e.freshest5mAt).Milliseconds() > int64(e.cfg.FreshnessSec)*1000 {
		return fmt.Errorf("picture_htf: send refused — bar data is %v old (limit %ds)", now.Sub(e.freshest5mAt).Round(time.Millisecond), e.cfg.FreshnessSec)
	}
	if now.UnixMilli() > row.WindowClose {
		return fmt.Errorf("picture_htf: send refused — the entry window closed at %s", time.UnixMilli(row.WindowClose).UTC().Format(time.RFC3339))
	}

	// --- Re-check 2: the transport must be the concrete NT8 TCP trader. ---
	tcp, ok := at.trader.(*ntTrader.TCPTrader)
	if !ok {
		return fmt.Errorf("picture_htf: trader %q is not the concrete NT8 TCP trader (%T) — no wire, opportunity stays place_pending", at.id, at.trader)
	}

	// --- Re-check 2: the book must be FLAT on this symbol. ---
	if pos, err := at.trader.GetPositions(); err == nil {
		for _, p := range pos {
			if sym, _ := p["symbol"].(string); instrumentRoot(sym) == instrumentRoot(row.Symbol) {
				return fmt.Errorf("picture_htf: send refused — open position on %s (flat book required)", row.Symbol)
			}
		}
	} else {
		return fmt.Errorf("picture_htf: send refused — cannot verify a flat book: %v", err)
	}

	// --- Re-check 3: no unreconciled send may precede this one (addendum #4:
	// an ambiguous place_pending row blocks re-entry until NT8 reconciles). ---
	pending, err := at.store.PictureHtfPendingByTrader(at.id)
	if err != nil {
		return fmt.Errorf("picture_htf: send refused — pending ledger read failed: %v", err)
	}
	for _, p := range pending {
		if p.OppKey != row.OppKey {
			return fmt.Errorf("picture_htf: send refused — opportunity %s is place_pending without broker evidence (blocks re-entry)", p.OppKey)
		}
	}

	// --- Re-check 4: sizing (1 contract, clamped by the strategy knob). ---
	qty = pictureHtfContractSize(at)

	// --- The send. beforeSend stamps the broker signal under the claim's
	// ownership marker — a stamp is refused for a row this caller doesn't own.
	side := row.Direction
	sid, err := tcp.MarketEntryWithProtection(side, qty, stopPx, targetPx, func(brokerSignalID string) error {
		return at.store.PictureHtfStampSignal(row.OppKey, row.SignalID, brokerSignalID)
	})
	if err != nil {
		return fmt.Errorf("picture_htf: market entry refused: %w", err)
	}
	logger.Infof("picture-htf: %s %s qty=%.0f stop=%.2f target=%.2f sent (signal %s, opp %s)",
		side, row.Symbol, qty, stopPx, targetPx, sid, row.OppKey)
	return nil
}
