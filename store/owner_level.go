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
	// C2 (2026-08-25) — cross-user leak: sticky levels were global per-symbol, so
	// any user's planner read (and the plan card) saw every other user's rows.
	// New rows are stamped with the creator's user_id; reads/writes/updates are
	// scoped (user_id = ? OR user_id = '' — the '' bucket is the pre-C2 legacy
	// backfilled to the original owner at migration).
	UserID    string `gorm:"column:user_id;not null;default:'';index:idx_owner_user"`
	CreatedAt int64  `gorm:"column:created_at"`
}

// TableName implements the gorm Tabler interface.
func (OwnerLevelDB) TableName() string { return "owner_levels" }

// NewOwnerLevelStore creates an OwnerLevelStore.
func NewOwnerLevelStore(db *gorm.DB) *OwnerLevelStore { return &OwnerLevelStore{db: db} }

func (s *OwnerLevelStore) initTables() error {
	// C2 (2026-08-25) — always AutoMigrate (additive): the old postgres branch
	// returned early when the table existed, which would have silently skipped
	// the new user_id column on already-provisioned databases.
	if err := s.db.AutoMigrate(&OwnerLevelDB{}); err != nil {
		return err
	}
	// Backfill legacy (pre-C2) rows to the ORIGINAL owner — the first-created
	// trader's user. Multi-user installs created later stamp their own rows.
	return s.db.Exec(`UPDATE owner_levels SET user_id = COALESCE((SELECT user_id FROM traders ORDER BY created_at ASC LIMIT 1), '') WHERE user_id IS NULL OR user_id = ''`).Error
}

// Save inserts an owner level.
func (s *OwnerLevelStore) Save(l *OwnerLevelDB) error {
	if l == nil || l.Symbol == "" || l.Price <= 0 {
		return fmt.Errorf("symbol and price required")
	}
	return s.db.Omit("ID").Create(l).Error
}

// ListActiveForUser returns the user's non-consumed levels for a symbol (sticky
// across sessions — no session/plan filter). user_id='' rows (pre-C2 legacy,
// backfilled to the original owner) stay visible only via the '' match below,
// so a second user sees ONLY their own rows.
func (s *OwnerLevelStore) ListActiveForUser(userID, symbol string) ([]*OwnerLevelDB, error) {
	var rows []*OwnerLevelDB
	q := s.db.Where("symbol = ? AND consumed = ?", symbol, false)
	if userID == "" {
		q = q.Where("user_id = ''")
	} else {
		q = q.Where("user_id = ? OR user_id = ''", userID)
	}
	err := q.Order("price ASC").Find(&rows).Error
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
// IDENTITY + label (never index), WHERE-scoped to the symbol AND the owner
// (C2: user_id = ? OR '' legacy). Returns whether a row matched.
func (s *OwnerLevelStore) UpdateNoteTag(userID, symbol string, price float64, label, note, tag string) (bool, error) {
	q := s.db.Model(&OwnerLevelDB{}).
		Where("symbol = ? AND price = ? AND label = ?", symbol, price, label)
	if userID == "" {
		q = q.Where("user_id = ''")
	} else {
		q = q.Where("user_id = ? OR user_id = ''", userID)
	}
	res := q.Updates(map[string]any{"note": note, "scenario_tag": tag})
	return res.RowsAffected > 0, res.Error
}

// Delete removes an owner level by id (unscoped — internal/test use; the API
// delete path uses DeleteForUser for C2 user scoping).
func (s *OwnerLevelStore) Delete(id int64) error {
	return s.db.Where("id = ?", id).Delete(&OwnerLevelDB{}).Error
}

// DeleteForUser removes an owner level, scoped to the owner (C2): a user can
// only delete their own rows (or '' legacy rows).
func (s *OwnerLevelStore) DeleteForUser(id int64, userID string) error {
	q := s.db.Where("id = ?", id)
	if userID == "" {
		q = q.Where("user_id = ''")
	} else {
		q = q.Where("user_id = ? OR user_id = ''", userID)
	}
	return q.Delete(&OwnerLevelDB{}).Error
}
