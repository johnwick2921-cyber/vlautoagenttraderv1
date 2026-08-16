package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P3.6-A — day-plan digest storage. Append-only 3-line digests: one per enabled
// session at its close, plus a daily roll-up at the trade-date close. They feed
// the planner's tapered week chain (the map remembers structure, the digest
// remembers the story).

// DigestStore persists session + daily digests.
type DigestStore struct {
	db *gorm.DB
}

// DigestDB is one stored digest (kind = "session" | "daily").
type DigestDB struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`
	// TraderID scopes the digest to the trader whose P&L it reports. W16/R4 — the
	// identity used to be (symbol, trade_date, session, kind), i.e. session-GLOBAL,
	// while the Text held ONE trader's entries and realized P&L. With two day-plan
	// traders on the same symbol, whichever goroutine won the race wrote the digest
	// that BOTH then read and fed to their next planner read.
	//
	// A plain index, deliberately NOT gorm's uniqueIndex: AutoMigrate would try to
	// build the unique index on any DB that already holds colliding rows and fail
	// the whole migration at boot.
	TraderID  string `gorm:"column:trader_id;not null;default:'';index:idx_digest_trader"`
	Symbol    string `gorm:"column:symbol;not null;default:'';index:idx_digest_symbol"`
	TradeDate string `gorm:"column:trade_date;not null;index:idx_digest_date"` // YYYY-MM-DD (CT)
	Session   string `gorm:"column:session;not null;default:''"`               // '' for daily
	Kind      string `gorm:"column:kind;not null"`                             // session | daily
	Text      string `gorm:"column:text;not null;default:''"`                  // 3-line digest
	CreatedAt int64  `gorm:"column:created_at"`
}

// TableName implements the gorm Tabler interface.
func (DigestDB) TableName() string { return "day_plan_digests" }

// NewDigestStore creates a DigestStore.
func NewDigestStore(db *gorm.DB) *DigestStore { return &DigestStore{db: db} }

func (s *DigestStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var n int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'day_plan_digests'`).Scan(&n)
		if n > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&DigestDB{})
}

// SaveIfAbsent writes a digest only if (trader_id, symbol, trade_date, session,
// kind) is not already stored — one digest per trader/session/day (idempotent,
// restart-safe). An empty trader_id is REFUSED: a digest that cannot say whose
// P&L it reports is worse than no digest, because the next planner read would
// consume it as fact.
func (s *DigestStore) SaveIfAbsent(d *DigestDB) (bool, error) {
	if d == nil || d.TradeDate == "" || d.Kind == "" || d.TraderID == "" {
		return false, fmt.Errorf("trader_id, trade_date and kind required")
	}
	var existing DigestDB
	err := s.db.Where("trader_id = ? AND symbol = ? AND trade_date = ? AND session = ? AND kind = ?",
		d.TraderID, d.Symbol, d.TradeDate, d.Session, d.Kind).First(&existing).Error
	if err == nil {
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	if err := s.db.Create(d).Error; err != nil {
		return false, err
	}
	return true, nil
}

// SessionDigests returns the "session" digest texts for one TRADER + symbol +
// trade_date
// (0–2 for the current date), oldest→newest.
func (s *DigestStore) SessionDigests(traderID, symbol, tradeDate string) ([]string, error) {
	var rows []*DigestDB
	err := s.db.Where("trader_id = ? AND symbol = ? AND trade_date = ? AND kind = ?", traderID, symbol, tradeDate, "session").
		Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Text)
	}
	return out, nil
}

// RecentDailies returns the most recent "daily" digest texts for one TRADER +
// symbol,
// newest first (up to limit).
func (s *DigestStore) RecentDailies(traderID, symbol string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 7
	}
	var rows []*DigestDB
	err := s.db.Where("trader_id = ? AND symbol = ? AND kind = ?", traderID, symbol, "daily").
		Order("trade_date DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Text)
	}
	return out, nil
}
