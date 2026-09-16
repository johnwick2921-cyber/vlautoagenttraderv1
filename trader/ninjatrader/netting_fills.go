package ninjatrader

import (
	"strings"
	"time"

	ntwire "nofx/provider/ninjatrader"
)

// Class 27 (2026-08-31 netting-orphan) — netting-fill ring.
//
// A NETTING close produces no position_close frame: an opposite-side ENTRY
// fill nets the account flat and NT8 just removes the position. The only
// evidence of the real exit price is that opposite fill itself (the Add→Flat
// pair seen in the NT8 trace). The reconcile orphan-close consults this ring
// to reconstruct the real exit — never exit=entry (the old fake $0).

// recentFill is one confirmed fill retained in the ring.
type recentFill struct {
	SignalID string
	Symbol   string
	Account  string
	Side     string
	Price    float64
	Quantity float64
	TimeMs   int64
}

// recentFillCap bounds the ring (32 fills is far more than any flat window needs).
const recentFillCap = 32

// nettingFillWindowMs: the netting fill can land up to one reconcile interval
// (20s) BEFORE the first flat observation; 25s covers it with margin.
const nettingFillWindowMs = 25_000

// recordRecentFill appends one confirmed fill to the ring. Caller holds t.mu.
func (t *TCPTrader) recordRecentFill(f ntwire.FillPayload) {
	ts := time.Now().UTC().UnixMilli()
	if f.FillTime != "" {
		if p, err := time.Parse(time.RFC3339, f.FillTime); err == nil {
			ts = p.UTC().UnixMilli()
		}
	}
	t.recentFills = append(t.recentFills, recentFill{
		SignalID: f.SignalID,
		Symbol:   f.Symbol,
		Account:  f.Account,
		Side:     strings.ToUpper(f.Side),
		Price:    f.FillPrice,
		Quantity: float64(f.Quantity),
		TimeMs:   ts,
	})
	if len(t.recentFills) > recentFillCap {
		t.recentFills = t.recentFills[len(t.recentFills)-recentFillCap:]
	}
}

// takeNettingExit returns the real exit price for an orphan row from the
// netting-fill ring: the LATEST OPPOSITE-side fill for the same symbol/account
// within [firstFlatMs−window, nowMs] — i.e. the fill that flattened the account
// when no position_close frame ever came. ok=false means NO evidence exists and
// the caller must mark the row UNRESOLVED (a visible gap beats a fake zero).
func (t *TCPTrader) takeNettingExit(account, symbol, rowSide string, firstFlatMs, nowMs int64) (float64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	side := strings.ToUpper(strings.TrimSpace(rowSide))
	opp := "LONG"
	if side == "LONG" {
		opp = "SHORT"
	}
	lo := firstFlatMs - nettingFillWindowMs
	var best *recentFill
	for i := range t.recentFills {
		f := &t.recentFills[i]
		if f.TimeMs < lo || f.TimeMs > nowMs || f.Price <= 0 {
			continue
		}
		if f.Symbol != "" && symbol != "" && !strings.EqualFold(f.Symbol, symbol) {
			continue
		}
		if f.Account != "" && account != "" && !strings.EqualFold(f.Account, account) {
			continue
		}
		if !strings.EqualFold(f.Side, opp) {
			continue
		}
		if best == nil || f.TimeMs > best.TimeMs {
			best = f
		}
	}
	if best == nil {
		return 0, false
	}
	return best.Price, true
}
