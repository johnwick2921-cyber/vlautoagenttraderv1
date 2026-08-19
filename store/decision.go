package store

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DecisionStore decision log storage
type DecisionStore struct {
	db *gorm.DB
}

// DecisionRecordDB internal GORM model for decision_records table
type DecisionRecordDB struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement"`
	TraderID            string    `gorm:"column:trader_id;not null;index:idx_decision_records_trader_time"`
	Account             string    `gorm:"column:account;not null;default:''"` // ITEM 2 per-account; '' for crypto/pre-migration (quarantined by account-scoped reads)
	CycleNumber         int       `gorm:"column:cycle_number;not null"`
	Timestamp           time.Time `gorm:"not null;index:idx_decision_records_trader_time,sort:desc;index:idx_decision_records_timestamp,sort:desc"`
	SystemPrompt        string    `gorm:"column:system_prompt;default:''"`
	InputPrompt         string    `gorm:"column:input_prompt;default:''"`
	CoTTrace            string    `gorm:"column:cot_trace;default:''"`
	DecisionJSON        string    `gorm:"column:decision_json;default:''"`
	RawResponse         string    `gorm:"column:raw_response;default:''"`
	CandidateCoins      string    `gorm:"column:candidate_coins;default:''"`
	ExecutionLog        string    `gorm:"column:execution_log;default:''"`
	Decisions           string    `gorm:"column:decisions;default:'[]'"`
	Success             bool      `gorm:"default:false"`
	ErrorMessage        string    `gorm:"column:error_message;default:''"`
	// P5 (ledger-close 2026-08-19) — TYPED error class so forensics count an
	// outage in one query (the 08-18 402 storm needed LIKE-matching free text).
	// "" | "ai_payment_402" | "ai_call_failed". Additive column.
	ErrorClass          string    `gorm:"column:error_class;default:''"`
	AIRequestDurationMs int64     `gorm:"column:ai_request_duration_ms;default:0"`
	// Plan 4 Task 23 — decision audit trail fields
	PromptVersion   string    `gorm:"column:prompt_version;default:''"`
	AIModel         string    `gorm:"column:ai_model;default:''"`
	AILatencyMs     int64     `gorm:"column:ai_latency_ms;default:0"`
	RiskCheckPassed bool      `gorm:"column:risk_check_passed;default:false"`
	RiskCheckError  string    `gorm:"column:risk_check_error;default:''"`
	ExecutionStatus string    `gorm:"column:execution_status;default:''"`
	FillPrice       *float64  `gorm:"column:fill_price"`
	FillLatencyMs   *int64    `gorm:"column:fill_latency_ms"`
	CreatedAt       time.Time `json:"created_at"`
	// P0.2 day-plan FK — attribute a decision to the exact plan+overlay+scenario
	// it cited (join → plans / plan_overlays). Additive, defaults empty/zero:
	// a decision that cited no plan is unchanged.
	PlanID          string `gorm:"column:plan_id;default:''"`
	PlanVersion     int    `gorm:"column:plan_version;default:0"`
	OverlayVersion  int    `gorm:"column:overlay_version;default:0"`
	CitedScenarioID string `gorm:"column:cited_scenario_id;default:''"`
}

func (DecisionRecordDB) TableName() string { return "decision_records" }

// DecisionRecord decision record (external API struct)
type DecisionRecord struct {
	ID                  int64              `json:"id"`
	TraderID            string             `json:"trader_id"`
	Account             string             `json:"account"`
	CycleNumber         int                `json:"cycle_number"`
	Timestamp           time.Time          `json:"timestamp"`
	SystemPrompt        string             `json:"system_prompt"`
	InputPrompt         string             `json:"input_prompt"`
	CoTTrace            string             `json:"cot_trace"`
	DecisionJSON        string             `json:"decision_json"`
	RawResponse         string             `json:"raw_response"` // Raw AI response for debugging
	CandidateCoins      []string           `json:"candidate_coins"`
	ExecutionLog        []string           `json:"execution_log"`
	Success             bool               `json:"success"`
	ErrorMessage        string             `json:"error_message"`
	ErrorClass          string             `json:"error_class,omitempty"` // P5 typed class ("ai_payment_402" | "ai_call_failed")
	AIRequestDurationMs int64              `json:"ai_request_duration_ms"`
	AccountState        AccountSnapshot    `json:"account_state"`
	Positions           []PositionSnapshot `json:"positions"`
	Decisions           []DecisionAction   `json:"decisions"`
	// Plan 4 Task 23 — decision audit trail fields (created_at is JSON-aliased from Timestamp).
	PromptVersion   string    `json:"prompt_version"`
	AIModel         string    `json:"ai_model"`
	AILatencyMs     int64     `json:"ai_latency_ms"`
	RiskCheckPassed bool      `json:"risk_check_passed"`
	RiskCheckError  string    `json:"risk_check_error"`
	ExecutionStatus string    `json:"execution_status"`
	FillPrice       *float64  `json:"fill_price,omitempty"`
	FillLatencyMs   *int64    `json:"fill_latency_ms,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	// P0.2 day-plan FK attribution (empty/zero when no plan was cited).
	PlanID          string `json:"plan_id,omitempty"`
	PlanVersion     int    `json:"plan_version,omitempty"`
	OverlayVersion  int    `json:"overlay_version,omitempty"`
	CitedScenarioID string `json:"cited_scenario_id,omitempty"`
}

// AccountSnapshot account state snapshot
type AccountSnapshot struct {
	TotalBalance          float64 `json:"total_balance"`
	AvailableBalance      float64 `json:"available_balance"`
	TotalUnrealizedProfit float64 `json:"total_unrealized_profit"`
	PositionCount         int     `json:"position_count"`
	MarginUsedPct         float64 `json:"margin_used_pct"`
	InitialBalance        float64 `json:"initial_balance"`
}

// PositionSnapshot position snapshot
type PositionSnapshot struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	PositionAmt      float64 `json:"position_amt"`
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	UnrealizedProfit float64 `json:"unrealized_profit"`
	Leverage         float64 `json:"leverage"`
	LiquidationPrice float64 `json:"liquidation_price"`
}

// DecisionAction decision action
type DecisionAction struct {
	Action     string    `json:"action"`
	Symbol     string    `json:"symbol"`
	Quantity   float64   `json:"quantity"`
	Leverage   int       `json:"leverage"`
	Price      float64   `json:"price"`
	StopLoss   float64   `json:"stop_loss,omitempty"`   // Stop loss price
	TakeProfit float64   `json:"take_profit,omitempty"` // Take profit price
	Confidence int       `json:"confidence,omitempty"`  // AI confidence (0-100)
	Reasoning  string    `json:"reasoning,omitempty"`   // Brief reasoning
	OrderID    int64     `json:"order_id"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
	Error      string    `json:"error"`
}

// Statistics statistics information
type Statistics struct {
	TotalCycles         int `json:"total_cycles"`
	SuccessfulCycles    int `json:"successful_cycles"`
	FailedCycles        int `json:"failed_cycles"`
	TotalOpenPositions  int `json:"total_open_positions"`
	TotalClosePositions int `json:"total_close_positions"`
}

// NewDecisionStore creates a new DecisionStore
func NewDecisionStore(db *gorm.DB) *DecisionStore {
	return &DecisionStore{db: db}
}

// initTables initializes AI decision log tables
func (s *DecisionStore) initTables() error {
	// For PostgreSQL with existing table, skip AutoMigrate
	if s.db.Dialector.Name() == "postgres" {
		var tableExists int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'decision_records'`).Scan(&tableExists)
		if tableExists > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&DecisionRecordDB{})
}

// toRecord converts DB model to API struct
func (db *DecisionRecordDB) toRecord() *DecisionRecord {
	record := &DecisionRecord{
		ID:                  db.ID,
		TraderID:            db.TraderID,
		Account:             db.Account,
		CycleNumber:         db.CycleNumber,
		Timestamp:           db.Timestamp,
		SystemPrompt:        db.SystemPrompt,
		InputPrompt:         db.InputPrompt,
		CoTTrace:            db.CoTTrace,
		DecisionJSON:        db.DecisionJSON,
		RawResponse:         db.RawResponse,
		Success:             db.Success,
		ErrorMessage:        db.ErrorMessage,
		ErrorClass:          db.ErrorClass,
		AIRequestDurationMs: db.AIRequestDurationMs,
		// Plan 4 Task 23 — audit trail fields
		PromptVersion:   db.PromptVersion,
		AIModel:         db.AIModel,
		AILatencyMs:     db.AILatencyMs,
		RiskCheckPassed: db.RiskCheckPassed,
		RiskCheckError:  db.RiskCheckError,
		ExecutionStatus: db.ExecutionStatus,
		FillPrice:       db.FillPrice,
		FillLatencyMs:   db.FillLatencyMs,
		CreatedAt:       db.CreatedAt,
		// P0.2 day-plan FK attribution
		PlanID:          db.PlanID,
		PlanVersion:     db.PlanVersion,
		OverlayVersion:  db.OverlayVersion,
		CitedScenarioID: db.CitedScenarioID,
	}
	json.Unmarshal([]byte(db.CandidateCoins), &record.CandidateCoins)
	json.Unmarshal([]byte(db.ExecutionLog), &record.ExecutionLog)
	json.Unmarshal([]byte(db.Decisions), &record.Decisions)
	return record
}

// LogDecision logs decision
func (s *DecisionStore) LogDecision(record *DecisionRecord) error {
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	} else {
		record.Timestamp = record.Timestamp.UTC()
	}

	// Serialize arrays to JSON
	candidateCoinsJSON, _ := json.Marshal(record.CandidateCoins)
	executionLogJSON, _ := json.Marshal(record.ExecutionLog)
	decisionsJSON, _ := json.Marshal(record.Decisions)

	dbRecord := &DecisionRecordDB{
		TraderID:            record.TraderID,
		Account:             record.Account,
		CycleNumber:         record.CycleNumber,
		Timestamp:           record.Timestamp,
		SystemPrompt:        record.SystemPrompt,
		InputPrompt:         record.InputPrompt,
		CoTTrace:            record.CoTTrace,
		DecisionJSON:        record.DecisionJSON,
		RawResponse:         record.RawResponse,
		CandidateCoins:      string(candidateCoinsJSON),
		ExecutionLog:        string(executionLogJSON),
		Decisions:           string(decisionsJSON),
		Success:             record.Success,
		ErrorMessage:        record.ErrorMessage,
		ErrorClass:          record.ErrorClass,
		AIRequestDurationMs: record.AIRequestDurationMs,
		// Plan 4 Task 23 — audit trail fields
		PromptVersion:   record.PromptVersion,
		AIModel:         record.AIModel,
		AILatencyMs:     record.AILatencyMs,
		RiskCheckPassed: record.RiskCheckPassed,
		RiskCheckError:  record.RiskCheckError,
		ExecutionStatus: record.ExecutionStatus,
		FillPrice:       record.FillPrice,
		FillLatencyMs:   record.FillLatencyMs,
		// P0.2 day-plan FK attribution
		PlanID:          record.PlanID,
		PlanVersion:     record.PlanVersion,
		OverlayVersion:  record.OverlayVersion,
		CitedScenarioID: record.CitedScenarioID,
	}

	if err := s.db.Create(dbRecord).Error; err != nil {
		return fmt.Errorf("failed to insert decision record: %w", err)
	}
	record.ID = dbRecord.ID
	return nil
}

// GetLatestRecords gets the latest N records for specified trader (sorted by time in ascending order: old to new),
// optionally scoped to one account (mirrors GetClosedPositions): account=="" → trader-global
// (crypto + legacy); account!="" → only that NT account's decisions, excluding pre-migration
// rows (account=''). Variadic so existing callers stay trader-global unchanged.
func (s *DecisionStore) GetLatestRecords(traderID string, n int, account ...string) ([]*DecisionRecord, error) {
	var dbRecords []*DecisionRecordDB
	q := s.db.Where("trader_id = ?", traderID)
	if len(account) > 0 && account[0] != "" {
		q = q.Where("account = ?", account[0])
	}
	err := q.Order("timestamp DESC").
		Limit(n).
		Find(&dbRecords).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query decision records: %w", err)
	}

	records := make([]*DecisionRecord, len(dbRecords))
	for i, db := range dbRecords {
		records[i] = db.toRecord()
	}

	// Reverse array to sort time from old to new
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records, nil
}

// GetAllLatestRecords gets the latest N records for all traders
func (s *DecisionStore) GetAllLatestRecords(n int) ([]*DecisionRecord, error) {
	var dbRecords []*DecisionRecordDB
	err := s.db.Order("timestamp DESC").Limit(n).Find(&dbRecords).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query decision records: %w", err)
	}

	records := make([]*DecisionRecord, len(dbRecords))
	for i, db := range dbRecords {
		records[i] = db.toRecord()
	}

	// Reverse array
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records, nil
}

// GetRecordsByDate gets all records for a specified trader on a specified date
func (s *DecisionStore) GetRecordsByDate(traderID string, date time.Time) ([]*DecisionRecord, error) {
	dateStr := date.Format("2006-01-02")

	var dbRecords []*DecisionRecordDB
	err := s.db.Where("trader_id = ? AND DATE(timestamp) = ?", traderID, dateStr).
		Order("timestamp ASC").
		Find(&dbRecords).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query decision records: %w", err)
	}

	records := make([]*DecisionRecord, len(dbRecords))
	for i, db := range dbRecords {
		records[i] = db.toRecord()
	}

	return records, nil
}

// CleanOldRecords cleans old records from N days ago
func (s *DecisionStore) CleanOldRecords(traderID string, days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result := s.db.Where("trader_id = ? AND timestamp < ?", traderID, cutoffTime).
		Delete(&DecisionRecordDB{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to clean old records: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// GetStatistics gets statistics for a trader, optionally scoped to one NT
// account (ITEM 2 per-account). account=="" → trader-global (crypto + legacy);
// account!="" → cycles and positions for that account only, excluding
// pre-migration rows (account=''). Scopes BOTH the decision-cycle counts and
// the cross-table position counts.
func (s *DecisionStore) GetStatistics(traderID string, account ...string) (*Statistics, error) {
	stats := &Statistics{}
	acct := ""
	if len(account) > 0 {
		acct = account[0]
	}

	var totalCount, successCount int64
	dq := s.db.Model(&DecisionRecordDB{}).Where("trader_id = ?", traderID)
	sq := s.db.Model(&DecisionRecordDB{}).Where("trader_id = ? AND success = ?", traderID, true)
	if acct != "" {
		dq = dq.Where("account = ?", acct)
		sq = sq.Where("account = ?", acct)
	}
	dq.Count(&totalCount)
	sq.Count(&successCount)

	stats.TotalCycles = int(totalCount)
	stats.SuccessfulCycles = int(successCount)
	stats.FailedCycles = stats.TotalCycles - stats.SuccessfulCycles

	// Count from trader_positions table (cross-table). Scope by account too.
	if acct != "" {
		s.db.Raw("SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND account = ?", traderID, acct).Scan(&stats.TotalOpenPositions)
		s.db.Raw("SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND account = ? AND status = 'CLOSED'", traderID, acct).Scan(&stats.TotalClosePositions)
	} else {
		s.db.Raw("SELECT COUNT(*) FROM trader_positions WHERE trader_id = ?", traderID).Scan(&stats.TotalOpenPositions)
		s.db.Raw("SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND status = 'CLOSED'", traderID).Scan(&stats.TotalClosePositions)
	}

	return stats, nil
}

// GetAllStatistics gets statistics information for all traders
func (s *DecisionStore) GetAllStatistics() (*Statistics, error) {
	stats := &Statistics{}

	var totalCount, successCount int64
	s.db.Model(&DecisionRecordDB{}).Count(&totalCount)
	s.db.Model(&DecisionRecordDB{}).Where("success = ?", true).Count(&successCount)

	stats.TotalCycles = int(totalCount)
	stats.SuccessfulCycles = int(successCount)
	stats.FailedCycles = stats.TotalCycles - stats.SuccessfulCycles

	// Count from trader_positions table
	s.db.Raw("SELECT COUNT(*) FROM trader_positions").Scan(&stats.TotalOpenPositions)
	s.db.Raw("SELECT COUNT(*) FROM trader_positions WHERE status = 'CLOSED'").Scan(&stats.TotalClosePositions)

	return stats, nil
}

// GetAuditRecords (Plan 4 Task 23) returns decision records for the audit endpoint.
// Filters: traderID (required), since (UTC inclusive lower bound), limit (capped at maxLimit),
// and an optional account scope (mirrors GetClosedPositions): account=="" → trader-global
// (crypto + legacy); account!="" → only that NT account's decisions, excluding pre-migration
// rows (account=''). Variadic so existing callers stay trader-global unchanged.
// Returns records ordered by timestamp DESC (newest first).
func (s *DecisionStore) GetAuditRecords(traderID string, since time.Time, limit int, account ...string) ([]*DecisionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	const maxLimit = 1000
	if limit > maxLimit {
		limit = maxLimit
	}

	q := s.db.Model(&DecisionRecordDB{}).Order("timestamp DESC").Limit(limit)
	if traderID != "" {
		q = q.Where("trader_id = ?", traderID)
	}
	if !since.IsZero() {
		q = q.Where("timestamp >= ?", since.UTC())
	}
	if len(account) > 0 && account[0] != "" {
		q = q.Where("account = ?", account[0])
	}

	var dbRecords []*DecisionRecordDB
	if err := q.Find(&dbRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit decision records: %w", err)
	}

	records := make([]*DecisionRecord, len(dbRecords))
	for i, db := range dbRecords {
		records[i] = db.toRecord()
	}
	return records, nil
}

// GetLastCycleNumber gets the last cycle number for specified trader
func (s *DecisionStore) GetLastCycleNumber(traderID string) (int, error) {
	var cycleNumber *int
	err := s.db.Model(&DecisionRecordDB{}).
		Where("trader_id = ?", traderID).
		Select("MAX(cycle_number)").
		Scan(&cycleNumber).Error
	if err != nil {
		return 0, err
	}
	if cycleNumber == nil {
		return 0, nil
	}
	return *cycleNumber, nil
}
