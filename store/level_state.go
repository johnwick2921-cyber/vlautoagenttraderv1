package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// P0.5 — LEVEL-STATE table (identity-keyed, cross-session persistence).
//
// A level's STATE (how many times it was tested, whether it's been consumed, its
// freshness A→B→C) must outlive the session it formed in — "a level burned in
// one session cannot return fresh in the next". Levels are RE-DERIVED each read,
// but their state persists here, keyed by level IDENTITY, not by which session
// is reading.
//
// Identity = (symbol, level_type, origin_date, bin_index). origin_date is the
// CME session-day the level FORMED (not the current day) so a naked POC keeps a
// stable key across days while its state accumulates. bin_index is the absolute
// price grid (floor(price / row-height)) so the same price lands in the same row
// regardless of session (mirrors kernel/svp.go svpBinIndex).
//
// Kernel is DB-blind (imports store for types only); this store is written FROM
// the trader layer at the session roll, exactly like decisions/equity. This file
// is the FOUNDATION (table + state API); the 17:00-CT snapshot WRITER that feeds
// it is a P1 item (RECON #2).

// Freshness grades. A level decays A→B→C with each play; below C it is done for
// the day (and marked consumed).
const (
	FreshnessA    = "A"
	FreshnessB    = "B"
	FreshnessC    = "C"
	FreshnessDone = "done"
)

// Common level types (arbitrary strings are allowed; these are the canonical set).
const (
	LevelTypePOC = "POC"
	LevelTypeVAH = "VAH"
	LevelTypeVAL = "VAL"
	LevelTypePDH = "PDH"
	LevelTypePDL = "PDL"
)

// LevelStateStore persists cross-session level state.
type LevelStateStore struct {
	db *gorm.DB
}

// LevelStateDB is one level's durable identity + state.
type LevelStateDB struct {
	LevelKey    string    `gorm:"column:level_key;primaryKey"` // symbol|type|origin_date|bin_index
	Symbol      string    `gorm:"column:symbol;not null;default:'';index:idx_levelstate_symbol"`
	LevelType   string    `gorm:"column:level_type;not null;default:''"`
	OriginDate  string    `gorm:"column:origin_date;not null;default:''"` // YYYY-MM-DD CME session-day of formation
	BinIndex    int       `gorm:"column:bin_index;not null;default:0"`
	Price       float64   `gorm:"column:price;not null;default:0"`
	TimesTested int       `gorm:"column:times_tested;not null;default:0"`
	Consumed    bool      `gorm:"column:consumed;not null;default:false"`
	Freshness   string    `gorm:"column:freshness;not null;default:'A';index:idx_levelstate_fresh"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName implements the gorm Tabler interface.
func (LevelStateDB) TableName() string { return "level_state" }

// MakeLevelKey builds the stable identity key for a level.
func MakeLevelKey(symbol, levelType, originDate string, binIndex int) string {
	return fmt.Sprintf("%s|%s|%s|%d", symbol, levelType, originDate, binIndex)
}

// NewLevelStateStore creates a LevelStateStore.
func NewLevelStateStore(db *gorm.DB) *LevelStateStore {
	return &LevelStateStore{db: db}
}

// initTables initializes the level_state table.
func (s *LevelStateStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var tableExists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'level_state'`).Scan(&tableExists)
		if tableExists > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&LevelStateDB{})
}

// EnsureLevel creates the level with fresh state if its identity is new; if it
// already exists (re-derived in a later session), its STATE is preserved and
// only the price is refreshed — a burned level never returns fresh.
func (s *LevelStateStore) EnsureLevel(l *LevelStateDB) error {
	if l == nil || l.Symbol == "" || l.LevelType == "" {
		return fmt.Errorf("symbol and level_type required")
	}
	if l.LevelKey == "" {
		l.LevelKey = MakeLevelKey(l.Symbol, l.LevelType, l.OriginDate, l.BinIndex)
	}
	if l.Freshness == "" {
		l.Freshness = FreshnessA
	}
	var existing LevelStateDB
	err := s.db.Where("level_key = ?", l.LevelKey).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(l).Error
	}
	if err != nil {
		return err
	}
	// Exists — keep state, refresh price only.
	return s.db.Model(&LevelStateDB{}).Where("level_key = ?", l.LevelKey).
		Update("price", l.Price).Error
}

// RecordTest atomically increments times_tested for a level.
func (s *LevelStateStore) RecordTest(levelKey string) error {
	return s.db.Model(&LevelStateDB{}).Where("level_key = ?", levelKey).
		Updates(map[string]any{"times_tested": gorm.Expr("times_tested + 1")}).Error
}

// MarkConsumed permanently consumes a level (price accepted through it).
func (s *LevelStateStore) MarkConsumed(levelKey string) error {
	return s.db.Model(&LevelStateDB{}).Where("level_key = ?", levelKey).
		Updates(map[string]any{"consumed": true, "freshness": FreshnessDone}).Error
}

// DecrementFreshness decays a level one grade (A→B→C→done) and returns the new
// grade. Reaching "done" also marks it consumed.
func (s *LevelStateStore) DecrementFreshness(levelKey string) (string, error) {
	var cur LevelStateDB
	if err := s.db.Where("level_key = ?", levelKey).First(&cur).Error; err != nil {
		return "", err
	}
	next := nextFreshness(cur.Freshness)
	updates := map[string]any{"freshness": next}
	if next == FreshnessDone {
		updates["consumed"] = true
	}
	if err := s.db.Model(&LevelStateDB{}).Where("level_key = ?", levelKey).Updates(updates).Error; err != nil {
		return "", err
	}
	return next, nil
}

func nextFreshness(f string) string {
	switch f {
	case FreshnessA:
		return FreshnessB
	case FreshnessB:
		return FreshnessC
	default:
		return FreshnessDone // C or already-done → done
	}
}

// Get returns a level by key, or (nil, nil) if absent.
func (s *LevelStateStore) Get(levelKey string) (*LevelStateDB, error) {
	var l LevelStateDB
	err := s.db.Where("level_key = ?", levelKey).First(&l).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ListForSymbol returns all stored levels for a symbol (any state).
func (s *LevelStateStore) ListForSymbol(symbol string) ([]*LevelStateDB, error) {
	var rows []*LevelStateDB
	err := s.db.Where("symbol = ?", symbol).Order("origin_date DESC, price ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListValid returns levels for a symbol that are still tradeable — not consumed
// and freshness above done.
func (s *LevelStateStore) ListValid(symbol string) ([]*LevelStateDB, error) {
	var rows []*LevelStateDB
	err := s.db.Where("symbol = ? AND consumed = ? AND freshness <> ?", symbol, false, FreshnessDone).
		Order("origin_date DESC, price ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
