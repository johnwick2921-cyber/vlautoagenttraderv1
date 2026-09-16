package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ── WATCHDOG FIRE LOG (owner ruling 2026-09-02) ─────────────────────────────
// The stream watchdog fired for the FIRST time on 2026-09-02 20:50:44 CT, at
// 376.1 s into a call that had produced 60,034 reasoning chars and then went
// silent for exactly its 30 s post-token limit. Before class 46 it reset on
// every SSE line — heartbeat comments included — so it could not fire at all.
//
// One fire is an anecdote. The ruling: record every fire with its call age,
// the bytes already received, and what the identical resend then did, for a
// week, then read the table. The resend outcome is the load-bearing column —
// a watchdog that kills calls the resend cannot recover is worse than one that
// waits.
type WatchdogFireDB struct {
	ID       uint64    `gorm:"primaryKey;autoIncrement"`
	TraderID string    `gorm:"index"`
	At       time.Time `gorm:"index"`
	// Kind (owner ruling 2026-09-03) distinguishes the two events that end a
	// stream early: "watchdog" (we closed it) and "cut" (the peer did). They
	// share this table because the load-bearing question is the same for both
	// — did the identical resend recover it — and because the idle_before
	// analysis needs them side by side.
	Kind      string `gorm:"index"` // "watchdog" | "cut"
	Mode      string // "pre" | "post" — which timer fired (watchdog only)
	GapMs     int64  // silence measured when it fired
	LimitMs   int64  // the EFFECTIVE limit in force (min of override and default)
	CallAgeMs int64  // how long the call had been running
	Bytes     int64  // reasoning + content chars received before the stall

	// The connection the dead call rode. The 2026-09-03 08:11:38 cut arrived on
	// a connection idle 101,212ms and reused; its successful resend rode one
	// idle 34,935ms. If cuts cluster above some idle threshold, IdleConnTimeout
	// below it is the whole fix — and this is the column that decides.
	IdleBeforeMs int64 `gorm:"index"`
	Reused       bool
	ClosedBy     string // peer_fin | local_close | clean (INFERRED — see class 46)

	// The identical resend that followed. Resolved=false until it completes,
	// so an unresolved row is visibly unresolved rather than a false zero.
	Resolved   bool
	ResendOK   bool
	ResendMs   int64
	ResendNote string
}

func (WatchdogFireDB) TableName() string { return "watchdog_fires" }

// WatchdogFireStore persists the fire log.
type WatchdogFireStore struct{ db *gorm.DB }

// NewWatchdogFireStore wires the table.
func NewWatchdogFireStore(db *gorm.DB) *WatchdogFireStore {
	if db != nil {
		_ = db.AutoMigrate(&WatchdogFireDB{})
	}
	return &WatchdogFireStore{db: db}
}

// Record inserts one fire and returns its id (0 on failure — callers WARN,
// never block a trading path on a log write).
func (s *WatchdogFireStore) Record(row WatchdogFireDB) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("store required")
	}
	if row.At.IsZero() {
		row.At = time.Now().UTC()
	}
	if err := s.db.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// ResolveLatest attaches the resend outcome to this trader's newest unresolved
// fire. Returns false when there is nothing open to attach to — which is the
// honest answer, not an error.
func (s *WatchdogFireStore) ResolveLatest(traderID string, ok bool, ms int64, note string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("store required")
	}
	var row WatchdogFireDB
	err := s.db.Where("trader_id = ? AND resolved = ?", traderID, false).
		Order("id DESC").First(&row).Error
	if err != nil {
		return false, nil
	}
	return true, s.db.Model(&WatchdogFireDB{}).Where("id = ?", row.ID).
		Updates(map[string]any{"resolved": true, "resend_ok": ok, "resend_ms": ms, "resend_note": note}).Error
}

// Recent returns the newest fires, newest first — the week's table.
func (s *WatchdogFireStore) Recent(limit int) ([]WatchdogFireDB, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store required")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []WatchdogFireDB
	err := s.db.Order("id DESC").Limit(limit).Find(&out).Error
	return out, err
}
