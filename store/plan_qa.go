package store

import (
	"fmt"

	"gorm.io/gorm"
)

// P5.4 — durable Ask-Planner threads (per-plan Q&A, archived with the plan). Each
// planner reply carries its anti-sycophancy verdict {point_class, verdict,
// applied} so the sycophancy KPI is queryable (VerdictStats).

// PlanQAStore persists Ask-Planner messages.
type PlanQAStore struct {
	db *gorm.DB
}

// PlanQADB is one thread message (owner question OR planner structured reply).
type PlanQADB struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID  string `gorm:"column:trader_id;not null;default:'';index:idx_qa_trader" json:"trader_id"`
	PlanID    string `gorm:"column:plan_id;not null;default:'';index:idx_qa_plan" json:"plan_id"`
	TradeDate string `gorm:"column:trade_date;not null;default:''" json:"trade_date"`
	Session   string `gorm:"column:session;not null;default:''" json:"session"`
	Role      string `gorm:"column:role;not null;default:''" json:"role"` // owner | planner
	Content   string `gorm:"column:content;not null;default:''" json:"content"`
	// planner-reply structured fields (empty on owner rows)
	Evidence   string `gorm:"column:evidence;not null;default:''" json:"evidence"`
	PointClass string `gorm:"column:point_class;not null;default:''" json:"point_class"` // NEW-INFO | BARE-DISAGREEMENT
	Verdict    string `gorm:"column:verdict;not null;default:''" json:"verdict"`         // DEFEND | CONCEDE | PROPOSE-MERGE
	Patch      string `gorm:"column:patch;not null;default:''" json:"patch"`            // RFC-6902 (PROPOSE-MERGE only)
	Applied    bool   `gorm:"column:applied;not null;default:false" json:"applied"`
	// W13 — re-alignment provenance. Trigger "" = an owner-typed Ask-Planner
	// question (the original path); "owner-edit" = auto re-align after an overlay
	// save; "manual-realign" = the owner tapped Re-align. ChallengeType records the
	// edit kind (add-level | edit-level | delete-level | bulk-add). Cost/latency are
	// per-call so the owner can see what the automation spends. Every re-align lands
	// in THIS table so the sycophancy KPI stays one comparable series.
	Trigger       string  `gorm:"column:trigger_kind;not null;default:''" json:"trigger"`
	ChallengeType string  `gorm:"column:challenge_type;not null;default:''" json:"challenge_type"`
	CostUSD       float64 `gorm:"column:cost_usd;not null;default:0" json:"cost_usd"`
	LatencyMs     int64   `gorm:"column:latency_ms;not null;default:0" json:"latency_ms"`
	CreatedAt     int64   `gorm:"column:created_at" json:"created_at"`
}

// TableName implements the gorm Tabler interface.
func (PlanQADB) TableName() string { return "plan_qa" }

// NewPlanQAStore creates a PlanQAStore.
func NewPlanQAStore(db *gorm.DB) *PlanQAStore { return &PlanQAStore{db: db} }

func (s *PlanQAStore) initTables() error {
	if s.db.Dialector.Name() == "postgres" {
		var n int64
		s.db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'plan_qa'`).Scan(&n)
		if n > 0 {
			return nil
		}
	}
	return s.db.AutoMigrate(&PlanQADB{})
}

// Append writes one message and returns its id.
func (s *PlanQAStore) Append(m *PlanQADB) (int64, error) {
	if m == nil || m.TraderID == "" || m.PlanID == "" {
		return 0, fmt.Errorf("trader_id and plan_id required")
	}
	if err := s.db.Omit("ID").Create(m).Error; err != nil {
		return 0, err
	}
	return m.ID, nil
}

// ListForPlan returns a plan's thread, oldest-first (chat order), scoped by trader.
func (s *PlanQAStore) ListForPlan(traderID, planID string, limit int) ([]*PlanQADB, error) {
	if limit <= 0 {
		limit = 200
	}
	var rows []*PlanQADB
	err := s.db.Where("trader_id = ? AND plan_id = ?", traderID, planID).
		Order("id ASC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetForTrader returns one message iff it belongs to traderID (IDOR guard for
// Apply). Returns (nil,nil) when not found/owned.
func (s *PlanQAStore) GetForTrader(traderID string, id int64) (*PlanQADB, error) {
	var row PlanQADB
	err := s.db.Where("id = ? AND trader_id = ?", id, traderID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// MarkApplied flags a planner reply's patch as applied (trader-scoped).
func (s *PlanQAStore) MarkApplied(traderID string, id int64) error {
	return s.db.Model(&PlanQADB{}).Where("id = ? AND trader_id = ?", id, traderID).
		Update("applied", true).Error
}

// VerdictStats is the sycophancy KPI: counts across a trader's planner replies.
type VerdictStats struct {
	Total          int `json:"total"`
	NewInfo        int `json:"new_info"`
	BareDisagree   int `json:"bare_disagreement"`
	Defend         int `json:"defend"`
	Concede        int `json:"concede"`
	ProposeMerge   int `json:"propose_merge"`
	Applied        int `json:"applied"`
	DefendOnBare   int `json:"defend_on_bare"` // the anti-sycophancy signal
}

// VerdictStats aggregates a trader's planner replies for the sycophancy KPI.
func (s *PlanQAStore) VerdictStats(traderID string) (VerdictStats, error) {
	var rows []*PlanQADB
	err := s.db.Where("trader_id = ? AND role = ?", traderID, "planner").Find(&rows).Error
	if err != nil {
		return VerdictStats{}, err
	}
	var st VerdictStats
	for _, r := range rows {
		st.Total++
		switch r.PointClass {
		case "NEW-INFO":
			st.NewInfo++
		case "BARE-DISAGREEMENT":
			st.BareDisagree++
		}
		switch r.Verdict {
		case "DEFEND":
			st.Defend++
		case "CONCEDE":
			st.Concede++
		case "PROPOSE-MERGE":
			st.ProposeMerge++
		}
		if r.Applied {
			st.Applied++
		}
		if r.PointClass == "BARE-DISAGREEMENT" && r.Verdict == "DEFEND" {
			st.DefendOnBare++
		}
	}
	return st, nil
}

// CountPlannerByTrigger returns how many PLANNER replies with this trigger exist
// for a plan — the per-session re-align cap counts these (W13). Counting rows is
// restart-safe and needs no separate counter table.
func (s *PlanQAStore) CountPlannerByTrigger(traderID, planID, trigger string) (int, error) {
	var n int64
	err := s.db.Model(&PlanQADB{}).
		Where("trader_id = ? AND plan_id = ? AND trigger_kind = ? AND role = ?", traderID, planID, trigger, "planner").
		Count(&n).Error
	return int(n), err
}

// LastPlannerByTrigger returns the most recent planner reply for (plan, trigger),
// or (nil, nil) when there is none. The W13 debounce reads its CreatedAt so rapid
// saves collapse into one call even across a restart.
func (s *PlanQAStore) LastPlannerByTrigger(traderID, planID, trigger string) (*PlanQADB, error) {
	var m PlanQADB
	err := s.db.Where("trader_id = ? AND plan_id = ? AND trigger_kind = ? AND role = ?", traderID, planID, trigger, "planner").
		Order("created_at DESC, id DESC").First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
