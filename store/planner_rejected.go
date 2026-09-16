package store

import (
	"time"

	"gorm.io/gorm"
)

// PlannerRejectedPrompt is the verbatim user prompt of a REJECTED planner
// attempt (planner-speed wave 1.4, 2026-08-31): the offline fast-vs-max A/B
// needs the exact prompt text the model was shown, which no log held (only a
// hash). Size-capped store — oldest rows are trimmed past the cap.
type PlannerRejectedPrompt struct {
	ID           uint   `gorm:"primaryKey"`
	TraderID     string `gorm:"index"`
	TradeDate    string `gorm:"index"`
	Session      string `gorm:"index"`
	PromptHash   string `gorm:"index"`
	Attempt      int    `gorm:"index"`
	RejectReason string `gorm:"type:text"`
	PromptText   string `gorm:"type:text"`
	// Facts (ROOT-FIX B-1, 2026-09-02) — the JSON snapshot of the PlanFacts the
	// live attempt was validated against. Without it an offline A/B could only
	// re-run the SCHEMA gate: every fact-dependent validator
	// (ValidatePlanDocWithFactsMachine, fvg, breakdown_continue) needs the same
	// price/PDH/PDL/DATR the live call saw, and those are exactly the
	// validators that reject real plans. Empty on rows written before this.
	Facts     string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}

// TableName is explicit so the cap-trim SQL never guesses.
func (PlannerRejectedPrompt) TableName() string { return "planner_rejected_prompts" }

// PlannerRejectedStore persists rejected planner prompts (verbatim) for the
// offline A/B measurement.
type PlannerRejectedStore struct {
	db *gorm.DB
}

// NewPlannerRejectedStore wires the table via AutoMigrate.
func NewPlannerRejectedStore(db *gorm.DB) *PlannerRejectedStore {
	if db != nil {
		_ = db.AutoMigrate(&PlannerRejectedPrompt{})
	}
	return &PlannerRejectedStore{db: db}
}

// plannerRejectedCap bounds the verbatim-prompt store (privacy + disk).
// OWNER RULING 2026-09-01: raised 20 → 200 so class 39 (leg normalization) has
// a sample. At 20 the store held roughly ONE session's rejects: the class-38
// forensics found n=1 usable instance of the very defect it was investigating
// because the rest had been trimmed. The 72h to 2026-09-01 carried 121
// validator rejects, so 200 spans a bad night without unbounding disk —
// prompts run ~25 KB, giving ~5 MB at the cap.
const plannerRejectedCap = 200

// SaveRejectedPrompt persists one rejected attempt's verbatim prompt + reason,
// trimming the store to the newest plannerRejectedCap rows.
func (s *PlannerRejectedStore) SaveRejectedPrompt(traderID, tradeDate, session, hash string, attempt int, reason, promptText string) error {
	return s.SaveRejectedPromptWithFacts(traderID, tradeDate, session, hash, attempt, reason, promptText, "")
}

// SaveRejectedPromptWithFacts (B-1) is SaveRejectedPrompt plus the facts
// snapshot the attempt was validated against, so an offline A/B can run the
// FULL validator chain rather than the schema gate alone.
func (s *PlannerRejectedStore) SaveRejectedPromptWithFacts(traderID, tradeDate, session, hash string, attempt int, reason, promptText, factsJSON string) error {
	if s == nil || s.db == nil {
		return nil
	}
	row := &PlannerRejectedPrompt{
		Facts:        factsJSON,
		TraderID:     traderID,
		TradeDate:    tradeDate,
		Session:      session,
		PromptHash:   hash,
		Attempt:      attempt,
		RejectReason: reason,
		PromptText:   promptText,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.db.Create(row).Error; err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&PlannerRejectedPrompt{}).Count(&count).Error; err == nil && count > plannerRejectedCap {
		_ = s.db.Exec("DELETE FROM planner_rejected_prompts WHERE id NOT IN (SELECT id FROM planner_rejected_prompts ORDER BY id DESC LIMIT ?)", plannerRejectedCap).Error
	}
	return nil
}

// Latest returns the newest stored rejected prompt (nil, gorm.ErrRecordNotFound
// when empty).
func (s *PlannerRejectedStore) Latest() (*PlannerRejectedPrompt, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var row PlannerRejectedPrompt
	if err := s.db.Order("id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
