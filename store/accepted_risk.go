package store

import (
	"time"

	"gorm.io/gorm"
)

// ── WAVE A / D4 — THE ACCEPTED RISK, MADE IMMUTABLE ──────────────────────────
//
// THE DEFECT. armed_orders is a MUTABLE ledger: its prices are re-composed
// every re-authorization cycle. Arm 35 is the proof — the row reads
// stop_px 29351.6284728996 while the order NT8 actually accepted and later
// filled was 29355, a 3.371527-point divergence introduced by a later cycle
// rewriting the row under a live working order. The in-place rewrite is now
// refused, but a refusal only stops the ledger getting WORSE; it does not
// preserve what the broker agreed to. Without that, R, stop adequacy,
// MAE-vs-floor and every risk quantity are UNMEASURABLE — which is exactly why
// only a handful of the 58 eligible trades have any recoverable initial stop,
// and the one that does was recovered from a rotating Windows log file.
//
// THIS TABLE IS APPEND-ONLY. There is no update path and no upsert. A later
// re-authorization writes a NEW row; the earlier one is never touched, so the
// two are BOTH readable and the drift between ledger and broker is a query
// instead of an archaeology.
//
// D5 CLOCK: every timestamp here is epoch MILLISECONDS UTC. The CT rendering is
// derived at read time. This table mixes nothing.
type AcceptedRisk struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement"`
	TraderID string `gorm:"column:trader_id;index"`
	SignalID string `gorm:"column:signal_id;index"`
	// OrderName is NT8's own name for the order, which carries the child-leg
	// suffix ("-sl" / "-tp") that names what the order IS.
	OrderName string `gorm:"column:order_name;index"`
	Symbol    string `gorm:"column:symbol;index"`
	Account   string `gorm:"column:account"`
	Side      string `gorm:"column:side"`
	OrderType string `gorm:"column:order_type"`
	Quantity  int    `gorm:"column:quantity"`

	// THE BROKER'S OWN TERMS, read from the F12 order snapshot — the same book
	// cutover leg 4 answers from. NULL when the book did not carry the order:
	// an unknown accepted price is NULL, never 0 and never the ledger's value
	// wearing the broker's name (A24).
	AcceptedEntryPx  *float64 `gorm:"column:accepted_entry_px"`
	AcceptedStopPx   *float64 `gorm:"column:accepted_stop_px"`
	AcceptedTargetPx *float64 `gorm:"column:accepted_target_px"`

	// THE LEDGER'S TERMS AT THE SAME INSTANT, so the divergence is recorded
	// rather than reconstructed. These may differ from the accepted values and
	// BOTH are readable — that is the point of the table.
	LedgerEntryPx  float64 `gorm:"column:ledger_entry_px"`
	LedgerStopPx   float64 `gorm:"column:ledger_stop_px"`
	LedgerTargetPx float64 `gorm:"column:ledger_target_px"`

	// BookAgeMs is how stale the broker book was when this was written. A
	// price copied from a stale book is a weaker fact and says so.
	BookAgeMs    int64  `gorm:"column:book_age_ms"`
	BookSource   string `gorm:"column:book_source"`
	AcceptedAtMs int64  `gorm:"column:accepted_at_ms;index"`
	CreatedAt    time.Time
}

func (AcceptedRisk) TableName() string { return "accepted_risk" }

// AcceptedRiskStore is APPEND-ONLY BY CONSTRUCTION: it exposes Append and
// readers, and no method that can modify a stored row.
type AcceptedRiskStore struct{ db *gorm.DB }

func NewAcceptedRiskStore(db *gorm.DB) *AcceptedRiskStore {
	if db != nil {
		_ = db.AutoMigrate(&AcceptedRisk{})
	}
	return &AcceptedRiskStore{db: db}
}

// Append records one acceptance. It never updates: a second acceptance for the
// same signal is a second row, and the first stays exactly as it was written.
func (s *AcceptedRiskStore) Append(r *AcceptedRisk) error {
	if s == nil || s.db == nil || r == nil {
		return nil
	}
	if r.AcceptedAtMs == 0 {
		r.AcceptedAtMs = time.Now().UTC().UnixMilli()
	}
	return s.db.Create(r).Error
}

// ForSignal returns every acceptance recorded for a signal, oldest first. More
// than one row means the terms were re-authorized, and the caller can see both.
func (s *AcceptedRiskStore) ForSignal(signalID string) ([]AcceptedRisk, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var out []AcceptedRisk
	err := s.db.Where("signal_id = ?", signalID).Order("id ASC").Find(&out).Error
	return out, err
}

// Count is the boot line's figure — READ (A11).
func (s *AcceptedRiskStore) Count() int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var n int64
	_ = s.db.Model(&AcceptedRisk{}).Count(&n).Error
	return n
}

// WithAcceptedStop counts the rows that actually recovered a broker stop —
// the number that decides whether initial risk is measurable at all.
func (s *AcceptedRiskStore) WithAcceptedStop() int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var n int64
	_ = s.db.Model(&AcceptedRisk{}).Where("accepted_stop_px IS NOT NULL").Count(&n).Error
	return n
}
