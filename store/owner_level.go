package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P3.6-C — STICKY OWNER LEVELS. Owner-origin levels persist ACROSS sessions
// (independent of any plan/session — they survive plan expiry) until consumed by
// acceptance or deleted. Every subsequent planner read receives them, tagged 👤
// with the owner's note + scenario tag. P5's overlay API posts into this table;
// the storage semantics live here now.

// OwnerLevelStore persists sticky owner levels.
type OwnerLevelStore struct {
	db *gorm.DB
}

// OwnerLevelDB is one owner-set level.
type OwnerLevelDB struct {
	ID          int64   `gorm:"primaryKey;autoIncrement"`
	Symbol      string  `gorm:"column:symbol;not null;index:idx_owner_symbol"`
	Price       float64 `gorm:"column:price;not null"`
	Label       string  `gorm:"column:label;not null;default:''"`
	Note        string  `gorm:"column:note;not null;default:''"` // free text → goes to the AI
	ScenarioTag string  `gorm:"column:scenario_tag;not null;default:''"`
	Consumed    bool    `gorm:"column:consumed;not null;default:false"`
	CreatedAt   int64   `gorm:"column:created_at"`
}

// TableName implements the gorm Tabler interface.
func (OwnerLevelDB) TableName() string { return "owner_levels" }

// NewOwnerLevelStore creates an OwnerLevelStore.
func NewOwnerLevelStore(db *gorm.DB) *OwnerLevelStore { return &OwnerLevelStore{db: db} }

func (s *OwnerLevelStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var n int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'owner_levels'`).Scan(&n)
		if n > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&OwnerLevelDB{})
}

// Save inserts an owner level.
func (s *OwnerLevelStore) Save(l *OwnerLevelDB) error {
	if l == nil || l.Symbol == "" || l.Price <= 0 {
		return fmt.Errorf("symbol and price required")
	}
	return s.db.Omit("ID").Create(l).Error
}

// ListActive returns the owner's non-consumed levels for a symbol (sticky across
// sessions — no session/plan filter).
func (s *OwnerLevelStore) ListActive(symbol string) ([]*OwnerLevelDB, error) {
	var rows []*OwnerLevelDB
	err := s.db.Where("symbol = ? AND consumed = ?", symbol, false).
		Order("price ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// MarkConsumed consumes an owner level (accepted through).
func (s *OwnerLevelStore) MarkConsumed(id int64) error {
	return s.db.Model(&OwnerLevelDB{}).Where("id = ?", id).Update("consumed", true).Error
}

// UpdateNoteTag (UI-verification 2026-08-18) writes the edit sheet's note +
// scenario-tag changes through to the sticky owner row, re-anchored by PRICE
// IDENTITY + label (never index), WHERE-scoped to the symbol. Returns whether
// a row matched.
func (s *OwnerLevelStore) UpdateNoteTag(symbol string, price float64, label, note, tag string) (bool, error) {
	res := s.db.Model(&OwnerLevelDB{}).
		Where("symbol = ? AND price = ? AND label = ?", symbol, price, label).
		Updates(map[string]any{"note": note, "scenario_tag": tag})
	return res.RowsAffected > 0, res.Error
}

// Delete removes an owner level (owner deletion).
func (s *OwnerLevelStore) Delete(id int64) error {
	return s.db.Where("id = ?", id).Delete(&OwnerLevelDB{}).Error
}
