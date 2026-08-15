package store

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// P0.2 — durable, append-only storage for the Day Plan system.
//
// Two tables:
//   - plans          : one immutable row per (plan_id, version). Never UPDATEd;
//                      a re-plan appends a new version. doc holds the plan JSON,
//                      guarded by CHECK(json_valid(doc)).
//   - plan_overlays  : append-only RFC-6902 patch versions per (plan_id,
//                      plan_version); owner/AI edits each add a new overlay_version.
//
// The executor_decisions table (store/decision.go) gains 4 additive FK columns
// (plan_id, plan_version, overlay_version, cited_scenario_id) so a decision can
// be joined back to the exact plan+overlay+scenario it cited.
//
// Writes go through a dedicated SINGLE-WRITER GOROUTINE. The append-only version
// assignment is read-max-then-insert (two statements); SetMaxOpenConns(1) alone
// serializes each statement but NOT the pair, so two concurrent appends could
// read the same max and collide on the composite PK. The writer goroutine runs
// each write closure to completion before the next, making the read+insert
// atomic without a table lock. (WAL — see store/gorm.go — lets card/replay reads
// proceed alongside; raising MaxOpenConns later stays safe because this goroutine
// remains the only writer of these tables.)

// PlanDB is one immutable plan version. TableName → "plans".
type PlanDB struct {
	PlanID        string    `gorm:"column:plan_id;primaryKey"`               // stable per (trade_date, session); see MakePlanID
	Version       int       `gorm:"column:version;primaryKey"`               // append-only, 1-based
	StrategyID    string    `gorm:"column:strategy_id;not null;default:''"`  // owning strategy
	TradeDate     string    `gorm:"column:trade_date;not null;default:''"`   // CME session-day YYYY-MM-DD
	Session       string    `gorm:"column:session;not null;default:'NY'"`    // NY|ASIA|LONDON
	TriggerReason string    `gorm:"column:trigger_reason;not null;default:''"`
	Lifecycle     string    `gorm:"column:lifecycle;not null;default:'active'"` // active|expired|died|superseded
	ModelID       string    `gorm:"column:model_id;not null;default:''"`        // resolved planner model id
	PromptHash    string    `gorm:"column:prompt_hash;not null;default:''"`
	Doc           string    `gorm:"column:doc;not null;default:'{}'"` // plan JSON; CHECK(json_valid(doc)) added in DDL
	// W11 — the indicator mirror the planner saw, FROZEN at read time (replay-safe:
	// later ai_config toggle changes never rewrite history) + the ai_config
	// fingerprint (which indicator config produced this plan).
	IndicatorsBlock string    `gorm:"column:indicators_block;not null;default:''"`
	AIConfigHash    string    `gorm:"column:ai_config_hash;not null;default:''"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName implements the gorm Tabler interface.
func (PlanDB) TableName() string { return "plans" }

// PlanOverlayDB is one append-only overlay (RFC-6902 patch) version.
type PlanOverlayDB struct {
	PlanID         string    `gorm:"column:plan_id;primaryKey"`
	PlanVersion    int       `gorm:"column:plan_version;primaryKey"`
	OverlayVersion int       `gorm:"column:overlay_version;primaryKey"` // append-only, 1-based per (plan_id, plan_version)
	OverlayID      string    `gorm:"column:overlay_id;not null;default:''"`
	Patch          string    `gorm:"column:patch;not null;default:'[]'"` // RFC-6902 JSON; CHECK(json_valid(patch)) in DDL
	Origin         string    `gorm:"column:origin;not null;default:''"`  // owner|ai
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName implements the gorm Tabler interface.
func (PlanOverlayDB) TableName() string { return "plan_overlays" }

// MakePlanID builds the stable plan identity from its (trade_date, session) key
// so the (trade_date, session, version) uniqueness the campaign mandates holds.
func MakePlanID(tradeDate, session string) string {
	return tradeDate + ":" + session
}

const plansDDL = `
CREATE TABLE IF NOT EXISTS plans (
	plan_id        TEXT    NOT NULL,
	version        INTEGER NOT NULL,
	strategy_id    TEXT    NOT NULL DEFAULT '',
	trade_date     TEXT    NOT NULL DEFAULT '',
	session        TEXT    NOT NULL DEFAULT 'NY',
	trigger_reason TEXT    NOT NULL DEFAULT '',
	lifecycle      TEXT    NOT NULL DEFAULT 'active',
	model_id       TEXT    NOT NULL DEFAULT '',
	prompt_hash    TEXT    NOT NULL DEFAULT '',
	doc            TEXT    NOT NULL DEFAULT '{}' CHECK (json_valid(doc)),
	indicators_block TEXT  NOT NULL DEFAULT '',
	ai_config_hash TEXT    NOT NULL DEFAULT '',
	created_at     DATETIME,
	PRIMARY KEY (plan_id, version)
)`

const planOverlaysDDL = `
CREATE TABLE IF NOT EXISTS plan_overlays (
	plan_id         TEXT    NOT NULL,
	plan_version    INTEGER NOT NULL,
	overlay_version INTEGER NOT NULL,
	overlay_id      TEXT    NOT NULL DEFAULT '',
	patch           TEXT    NOT NULL DEFAULT '[]' CHECK (json_valid(patch)),
	origin          TEXT    NOT NULL DEFAULT '',
	created_at      DATETIME,
	PRIMARY KEY (plan_id, plan_version, overlay_version)
)`

// PlanStore is the append-only plan/overlay store with a single-writer goroutine.
type PlanStore struct {
	db        *gorm.DB
	writeCh   chan planWriteReq
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

type planWriteReq struct {
	fn    func(*gorm.DB) error
	reply chan error
}

// NewPlanStore creates a PlanStore. The writer goroutine starts lazily on the
// first write.
func NewPlanStore(db *gorm.DB) *PlanStore {
	return &PlanStore{
		db:      db,
		writeCh: make(chan planWriteReq),
		done:    make(chan struct{}),
	}
}

// initTables creates the plans + plan_overlays tables. On SQLite (the live +
// tested dialect) it uses raw DDL so the composite PK and the json_valid CHECK
// are exact; on other dialects it falls back to AutoMigrate (no SQLite-only
// json_valid CHECK). Both are idempotent (IF NOT EXISTS / additive).
func (s *PlanStore) initTables() error {
	if s.db.Dialector.Name() == "sqlite" {
		if err := s.db.Exec(plansDDL).Error; err != nil {
			return fmt.Errorf("create plans table: %w", err)
		}
		if err := s.db.Exec(planOverlaysDDL).Error; err != nil {
			return fmt.Errorf("create plan_overlays table: %w", err)
		}
		// Enforce the campaign key (trade_date, session, version) explicitly, and
		// index the common lookups.
		s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_date_session_version ON plans(trade_date, session, version)`)
		s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_plans_strategy ON plans(strategy_id)`)
		s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_overlays_plan ON plan_overlays(plan_id, plan_version)`)
		// W11 — idempotent additive columns for EXISTING sqlite DBs (CREATE TABLE IF
		// NOT EXISTS won't add them). SQLite has no ADD COLUMN IF NOT EXISTS, so the
		// duplicate-column error on re-run is expected + swallowed (same as the
		// index Execs above).
		s.db.Exec(`ALTER TABLE plans ADD COLUMN indicators_block TEXT NOT NULL DEFAULT ''`)
		s.db.Exec(`ALTER TABLE plans ADD COLUMN ai_config_hash TEXT NOT NULL DEFAULT ''`)
		return nil
	}
	return s.db.AutoMigrate(&PlanDB{}, &PlanOverlayDB{})
}

// enqueue runs fn on the single writer goroutine and blocks for its result.
func (s *PlanStore) enqueue(fn func(*gorm.DB) error) error {
	s.startOnce.Do(func() {
		go s.writerLoop()
	})
	req := planWriteReq{fn: fn, reply: make(chan error, 1)}
	select {
	case s.writeCh <- req:
	case <-s.done:
		return fmt.Errorf("plan store closed")
	}
	select {
	case err := <-req.reply:
		return err
	case <-s.done:
		return fmt.Errorf("plan store closed")
	}
}

func (s *PlanStore) writerLoop() {
	for {
		select {
		case req := <-s.writeCh:
			req.reply <- s.execWrite(req.fn)
		case <-s.done:
			return
		}
	}
}

func (s *PlanStore) execWrite(fn func(*gorm.DB) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plan write panicked: %v", r)
		}
	}()
	return fn(s.db)
}

// Close stops the writer goroutine. Intended for tests/shutdown; the process
// Store never calls it (process exit reclaims the goroutine).
func (s *PlanStore) Close() {
	s.stopOnce.Do(func() { close(s.done) })
}

// AppendPlan appends a new immutable plan version (never UPDATE). Version is
// assigned as max(version)+1 for the plan_id inside the single-writer goroutine.
// Returns the assigned version.
func (s *PlanStore) AppendPlan(p *PlanDB) (int, error) {
	if p == nil || p.PlanID == "" {
		return 0, fmt.Errorf("plan_id required")
	}
	var assigned int
	err := s.enqueue(func(db *gorm.DB) error {
		var maxV *int
		if err := db.Model(&PlanDB{}).Where("plan_id = ?", p.PlanID).
			Select("MAX(version)").Scan(&maxV).Error; err != nil {
			return err
		}
		next := 1
		if maxV != nil {
			next = *maxV + 1
		}
		p.Version = next
		if p.Doc == "" {
			p.Doc = "{}"
		}
		if p.Lifecycle == "" {
			p.Lifecycle = "active"
		}
		if p.Session == "" {
			p.Session = "NY"
		}
		if err := db.Create(p).Error; err != nil {
			return err
		}
		assigned = next
		return nil
	})
	return assigned, err
}

// AppendOverlay appends a new overlay version for (plan_id, plan_version).
// Returns the assigned overlay_version.
func (s *PlanStore) AppendOverlay(o *PlanOverlayDB) (int, error) {
	if o == nil || o.PlanID == "" || o.PlanVersion <= 0 {
		return 0, fmt.Errorf("plan_id and plan_version required")
	}
	var assigned int
	err := s.enqueue(func(db *gorm.DB) error {
		var maxV *int
		if err := db.Model(&PlanOverlayDB{}).
			Where("plan_id = ? AND plan_version = ?", o.PlanID, o.PlanVersion).
			Select("MAX(overlay_version)").Scan(&maxV).Error; err != nil {
			return err
		}
		next := 1
		if maxV != nil {
			next = *maxV + 1
		}
		o.OverlayVersion = next
		if o.Patch == "" {
			o.Patch = "[]"
		}
		if err := db.Create(o).Error; err != nil {
			return err
		}
		assigned = next
		return nil
	})
	return assigned, err
}

// GetPlan returns a specific plan version, or (nil, nil) if absent.
func (s *PlanStore) GetPlan(planID string, version int) (*PlanDB, error) {
	var p PlanDB
	err := s.db.Where("plan_id = ? AND version = ?", planID, version).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetLatestPlan returns the newest version for a plan_id, or (nil, nil) if none.
func (s *PlanStore) GetLatestPlan(planID string) (*PlanDB, error) {
	var p PlanDB
	err := s.db.Where("plan_id = ?", planID).Order("version DESC").First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetLatestPlanForSession returns the newest plan version for a (trade_date,
// session), or (nil, nil) if none.
func (s *PlanStore) GetLatestPlanForSession(tradeDate, session string) (*PlanDB, error) {
	var p PlanDB
	err := s.db.Where("trade_date = ? AND session = ?", tradeDate, session).
		Order("version DESC").First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListRecent returns the most recent plan rows (any session) newest-first, for
// the plan-history view.
func (s *PlanStore) ListRecent(limit int) ([]*PlanDB, error) {
	if limit <= 0 {
		limit = 30
	}
	var rows []*PlanDB
	err := s.db.Order("created_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListOverlays returns all overlay versions for (plan_id, plan_version) in
// ascending overlay_version order.
func (s *PlanStore) ListOverlays(planID string, planVersion int) ([]*PlanOverlayDB, error) {
	var rows []*PlanOverlayDB
	err := s.db.Where("plan_id = ? AND plan_version = ?", planID, planVersion).
		Order("overlay_version ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
