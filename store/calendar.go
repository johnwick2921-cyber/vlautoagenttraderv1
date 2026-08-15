package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P1.8 — per-day calendar slice storage. Each trade date's filtered event slice
// is stored ONCE (replay-frozen, append-only) so replay/audit uses the exact
// copy that was live, never a refetch. Includes the source (forexfactory /
// static / none) so an outage day is auditable.

// CalendarSliceStore persists per-trade-date calendar slices.
type CalendarSliceStore struct {
	db *gorm.DB
}

// CalendarSliceDB is one trade date's stored calendar slice.
type CalendarSliceDB struct {
	TradeDate  string `gorm:"column:trade_date;primaryKey"` // YYYY-MM-DD (CT)
	Source     string `gorm:"column:source;not null;default:''"`
	EventsJSON string `gorm:"column:events_json;not null;default:'[]'"`
	CreatedAt  int64  `gorm:"column:created_at"` // Unix ms
}

// TableName implements the gorm Tabler interface.
func (CalendarSliceDB) TableName() string { return "calendar_slices" }

// NewCalendarSliceStore creates a CalendarSliceStore.
func NewCalendarSliceStore(db *gorm.DB) *CalendarSliceStore {
	return &CalendarSliceStore{db: db}
}

func (s *CalendarSliceStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var exists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'calendar_slices'`).Scan(&exists)
		if exists > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&CalendarSliceDB{})
}

// SaveSliceIfAbsent stores a trade date's slice only if not already stored
// (replay-frozen). Returns true when a new row was written.
func (s *CalendarSliceStore) SaveSliceIfAbsent(p *CalendarSliceDB) (bool, error) {
	if p == nil || p.TradeDate == "" {
		return false, fmt.Errorf("trade_date required")
	}
	var existing CalendarSliceDB
	err := s.db.Where("trade_date = ?", p.TradeDate).First(&existing).Error
	if err == nil {
		return false, nil // frozen — never overwrite
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	if p.EventsJSON == "" {
		p.EventsJSON = "[]"
	}
	if err := s.db.Create(p).Error; err != nil {
		return false, err
	}
	return true, nil
}

// UpsertSliceUpgrade stores a LIVE-source slice, additionally replacing an
// existing row ONLY when that row is non-live (static/none) — F0.3: an outage
// day's static guess upgrades to the real feed as soon as it recovers, while
// forexfactory rows stay frozen (replay integrity, same rule as SaveSliceIfAbsent).
// Returns (wrote, upgraded, err); wrote=false means a frozen live row existed.
func (s *CalendarSliceStore) UpsertSliceUpgrade(p *CalendarSliceDB) (bool, bool, error) {
	if p == nil || p.TradeDate == "" {
		return false, false, fmt.Errorf("trade_date required")
	}
	if p.EventsJSON == "" {
		p.EventsJSON = "[]"
	}
	var existing CalendarSliceDB
	err := s.db.Where("trade_date = ?", p.TradeDate).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if cerr := s.db.Create(p).Error; cerr != nil {
			return false, false, cerr
		}
		return true, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if existing.Source == "forexfactory" {
		return false, false, nil // frozen — live rows never overwritten
	}
	if uerr := s.db.Model(&CalendarSliceDB{}).Where("trade_date = ?", p.TradeDate).Updates(map[string]interface{}{
		"source": p.Source, "events_json": p.EventsJSON, "created_at": p.CreatedAt,
	}).Error; uerr != nil {
		return false, false, uerr
	}
	return true, true, nil
}

// GetSlice returns a stored slice, or (nil, nil) if absent.
func (s *CalendarSliceStore) GetSlice(tradeDate string) (*CalendarSliceDB, error) {
	var p CalendarSliceDB
	err := s.db.Where("trade_date = ?", tradeDate).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
