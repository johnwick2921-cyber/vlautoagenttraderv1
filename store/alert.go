package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P4.4 — in-app alert store (no external push). P0 = toast + persistent banner
// until ack · P1 = feed · P2 = digest-only. Deduped by event_id (idempotent bus).

// AlertStore persists in-app alerts.
type AlertStore struct {
	db *gorm.DB
}

// AlertDB is one alert.
type AlertDB struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID  string `gorm:"column:trader_id;not null;default:'';index:idx_alert_trader" json:"trader_id"`
	Level     string `gorm:"column:level;not null;default:'P2'" json:"level"`         // P0 | P1 | P2
	EventID   string `gorm:"column:event_id;not null;default:'';index:idx_alert_event" json:"event_id"` // dedupe key
	Kind      string `gorm:"column:kind;not null;default:''" json:"kind"`
	Title     string `gorm:"column:title;not null;default:''" json:"title"`
	Body      string `gorm:"column:body;not null;default:''" json:"body"`
	Acked     bool   `gorm:"column:acked;not null;default:false" json:"acked"`
	CreatedAt int64  `gorm:"column:created_at" json:"created_at"`
}

// TableName implements the gorm Tabler interface.
func (AlertDB) TableName() string { return "day_plan_alerts" }

// NewAlertStore creates an AlertStore.
func NewAlertStore(db *gorm.DB) *AlertStore { return &AlertStore{db: db} }

func (s *AlertStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var n int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'day_plan_alerts'`).Scan(&n)
		if n > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&AlertDB{})
}

// Emit inserts an alert, deduped by (trader_id, event_id) when event_id is set
// (idempotent bus). Returns true when a new row was written.
func (s *AlertStore) Emit(a *AlertDB) (bool, error) {
	if a == nil || a.TraderID == "" {
		return false, fmt.Errorf("trader_id required")
	}
	if a.EventID != "" {
		var n int64
		s.db.Model(&AlertDB{}).Where("trader_id = ? AND event_id = ?", a.TraderID, a.EventID).Count(&n)
		if n > 0 {
			return false, nil // duplicate event → no-op
		}
	}
	if err := s.db.Omit("ID").Create(a).Error; err != nil {
		return false, err
	}
	return true, nil
}

// List returns a trader's alerts, newest first (up to limit).
func (s *AlertStore) List(traderID string, limit int) ([]*AlertDB, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []*AlertDB
	err := s.db.Where("trader_id = ?", traderID).
		Order("id DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UnackedCount returns the number of un-acked alerts (the bell badge).
func (s *AlertStore) UnackedCount(traderID string) (int, error) {
	var n int64
	err := s.db.Model(&AlertDB{}).Where("trader_id = ? AND acked = ?", traderID, false).Count(&n).Error
	return int(n), err
}

// Ack marks an alert acknowledged (unscoped; internal callers only).
func (s *AlertStore) Ack(id int64) error {
	return s.db.Model(&AlertDB{}).Where("id = ?", id).Update("acked", true).Error
}

// AckForTrader marks an alert acknowledged ONLY if it belongs to traderID
// (IDOR guard — the bell feed is per-trader, so acks must be too). Returns
// found=false when no row matched the (id, trader_id) pair, so the handler can
// 404 instead of silently acking nothing (or another trader's alert).
func (s *AlertStore) AckForTrader(traderID string, id int64) (bool, error) {
	res := s.db.Model(&AlertDB{}).
		Where("id = ? AND trader_id = ?", id, traderID).
		Update("acked", true)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
