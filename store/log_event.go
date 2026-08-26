package store

import (
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// P6 — FORENSIC DURABILITY: WARN/ERROR/CRITICAL → DB (ledger-close dispatch
// 2026-08-19, U3).
//
// The journald 2G cap rotates in HOURS under per-frame INFO logging — the
// Aug 13–17 window was already gone when the zero-trade forensics ran. This
// table keeps the lines forensics actually needed (gate refusals, clock-health
// CRITICAL, ai_call failures, feed alerts) in the DB, ride-along with the
// existing backup timer.
//
// GUARANTEE: shipping NEVER blocks or crashes the trading path —
//   - Enqueue is a single select-default: channel full → the event is DROPPED
//     and a counter bumped (the tcp_server enqueueBarUpdate posture),
//   - one lazy single-writer goroutine owns every INSERT (the plan-store
//     single-writer pattern; SQLite pool is capped at 1 conn),
//   - the shipper itself never logs through logrus (recursion-proof by
//     construction; its own failures only bump droppedWrites).
//
// Retention: LOG_DB_RETENTION_DAYS (default 30), hard-delete once per day from
// the writer goroutine (EquityStore.CleanOldRecords precedent).
// Rollback note (report-only, never executed here): DROP TABLE log_events;

// LogEventDB is one shipped log line.
type LogEventDB struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	TsUTC      int64  `gorm:"column:ts_utc;not null;index:idx_log_events_ts" json:"ts_utc"` // unix ms
	Level      string `gorm:"column:level;not null;default:''" json:"level"`
	Component  string `gorm:"column:component;not null;default:''" json:"component"` // file.go:line
	TraderID   string `gorm:"column:trader_id;not null;default:'';index:idx_log_events_trader" json:"trader_id"`
	Message    string `gorm:"column:message;not null;default:''" json:"message"`
	FieldsJSON string `gorm:"column:fields_json;not null;default:'{}'" json:"fields_json"`
}

// TableName implements the gorm Tabler interface.
func (LogEventDB) TableName() string { return "log_events" }

// LogEventStore ships log events asynchronously.
type LogEventStore struct {
	db            *gorm.DB
	once          sync.Once
	ch            chan LogEventDB
	droppedFull   atomic.Int64
	droppedWrites atomic.Int64
	lastPruneDay  string
}

// NewLogEventStore creates a LogEventStore.
func NewLogEventStore(db *gorm.DB) *LogEventStore {
	return &LogEventStore{db: db, ch: make(chan LogEventDB, 1024)}
}

func (s *LogEventStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var tableExists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'log_events'`).Scan(&tableExists)
		if tableExists > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&LogEventDB{})
}

// logDBRetentionDays reads LOG_DB_RETENTION_DAYS (default 30).
func logDBRetentionDays() int {
	if v := os.Getenv("LOG_DB_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 30
}

// Enqueue ships one event without ever blocking: full buffer → drop + count.
func (s *LogEventStore) Enqueue(ev LogEventDB) {
	s.once.Do(func() { go s.writerLoop() })
	select {
	case s.ch <- ev:
	default:
		s.droppedFull.Add(1)
	}
}

// Dropped reports (queue-full drops, failed writes) — observability for the
// no-silent-caps rule.
func (s *LogEventStore) Dropped() (int64, int64) {
	return s.droppedFull.Load(), s.droppedWrites.Load()
}

func (s *LogEventStore) writerLoop() {
	for ev := range s.ch {
		if err := s.db.Create(&ev).Error; err != nil {
			s.droppedWrites.Add(1) // never log from here (recursion)
		}
		// Once-per-UTC-day retention prune, from the same single writer.
		if day := time.UnixMilli(ev.TsUTC).UTC().Format("2006-01-02"); day != s.lastPruneDay {
			s.lastPruneDay = day
			cutoff := time.Now().AddDate(0, 0, -logDBRetentionDays()).UnixMilli()
			_ = s.db.Where("ts_utc < ?", cutoff).Delete(&LogEventDB{}).Error
		}
	}
}

// Recent returns the newest events (forensics/API convenience).
func (s *LogEventStore) Recent(limit int) ([]*LogEventDB, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	var rows []*LogEventDB
	err := s.db.Order("ts_utc DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// PruneOlderThan hard-deletes events older than the cutoff (test hook; the
// writer loop calls the same statement daily).
func (s *LogEventStore) PruneOlderThan(cutoffMs int64) (int64, error) {
	res := s.db.Where("ts_utc < ?", cutoffMs).Delete(&LogEventDB{})
	return res.RowsAffected, res.Error
}
