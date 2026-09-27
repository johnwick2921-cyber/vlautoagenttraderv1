package store

import "gorm.io/gorm"

// SettleFilledWithFill settles order id FILLED (UpdateOrderStatus) and writes
// its trader_fills row (CreateFill, deduped on ExchangeTradeID) as ONE unit:
// both land or neither does. W1b FOLD-13 — the late-entry-fill path settles
// before it tags; a fill-row write that failed after the status had moved
// would leave an order reading FILLED with no fill row (a fabricated
// half-record) and no NEW row left for the unresolved-order sweep to see.
// On error nothing was written.
func (s *OrderStore) SettleFilledWithFill(id int64, filledQty, avgPrice, commission float64, fill *TraderFill) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		in := &OrderStore{db: tx}
		if err := in.UpdateOrderStatus(id, "FILLED", filledQty, avgPrice, commission); err != nil {
			return err
		}
		return in.CreateFill(fill)
	})
}
