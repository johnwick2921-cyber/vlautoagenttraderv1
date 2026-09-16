package store

import (
	"time"

	"gorm.io/gorm"
)

// ── D3 — CANDIDATE POOL (1B, plan appendix B2) ───────────────────────────────
//
// At every planner read, EVERY level the constructor produced — seated or cut —
// with its rank, score components, cut reason and seating propensity.
//
// This exists because the selection question is OFF-POLICY: asking "did seating
// this level help?" requires the levels that were NOT seated and the propensity
// that decided each one. A table of only the seated levels can answer how the
// chosen ones performed and can never answer whether the choosing was good.
type CandidatePoolRow struct {
	ID          uint    `gorm:"primaryKey"`
	TraderID    string  `gorm:"index"`
	Symbol      string  `gorm:"index"`
	PlanID      string  `gorm:"index"`
	PlanVersion int     `gorm:"index"`
	Session     string  `gorm:"index"`
	ReadAtMs    int64   `gorm:"index"`
	LevelPrice  float64 `gorm:"index"`
	LevelKind   string  `gorm:"index"`
	Label       string
	// Rank within the constructor's ordering, 1-based. 0 = unranked.
	Rank int `gorm:"index"`
	// Seated: did it reach the plan? The whole point of the table is that this
	// is FALSE for most rows.
	Seated bool `gorm:"index"`
	// CutReason is why it did not seat ("" when seated) — proximity, max_levels,
	// min_grade, cluster-collapse, and so on.
	CutReason string `gorm:"index"`
	// Propensity is the score/threshold pair that DECIDED it, so the decision is
	// reconstructable rather than merely recorded.
	Score     float64
	Threshold float64
	Grade     string
	// ScoreComponents is the per-component JSON breakdown ("{}" = not computed,
	// which is different from all-zero — A24).
	ScoreComponents string    `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"index"`
}

func (CandidatePoolRow) TableName() string { return "candidate_pool" }

// CandidatePoolStore persists the per-read candidate pool.
type CandidatePoolStore struct{ db *gorm.DB }

// NewCandidatePoolStore wires the table via AutoMigrate.
func NewCandidatePoolStore(db *gorm.DB) *CandidatePoolStore {
	if db != nil {
		_ = db.AutoMigrate(&CandidatePoolRow{})
	}
	return &CandidatePoolStore{db: db}
}

// CandidatePoolCap bounds the table: every read writes one row per candidate,
// so this grows fastest of the 1B tables. ~20 candidates × ~40 reads/day ≈ 800
// rows/day, so 20k spans about three weeks.
const CandidatePoolCap = 20000

// SavePool writes the whole pool for one read in a single transaction and trims
// to the cap. A partial pool is worse than none — it would look like a smaller
// candidate set — so the rows go in together or not at all.
func (s *CandidatePoolStore) SavePool(rows []CandidatePoolRow) error {
	if s == nil || s.db == nil || len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for i := range rows {
		if rows[i].CreatedAt.IsZero() {
			rows[i].CreatedAt = now
		}
		if rows[i].ScoreComponents == "" {
			rows[i].ScoreComponents = "{}"
		}
	}
	if err := s.db.Create(&rows).Error; err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&CandidatePoolRow{}).Count(&count).Error; err == nil && count > CandidatePoolCap {
		_ = s.db.Exec("DELETE FROM candidate_pool WHERE id NOT IN (SELECT id FROM candidate_pool ORDER BY id DESC LIMIT ?)", CandidatePoolCap).Error
	}
	return nil
}

// CountPool is the boot line's figure — READ, never a literal.
func (s *CandidatePoolStore) CountPool() int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var n int64
	_ = s.db.Model(&CandidatePoolRow{}).Count(&n).Error
	return n
}

// LatestPool returns one read's pool, newest first — the shape D6 reports.
func (s *CandidatePoolStore) LatestPool(limit int) ([]CandidatePoolRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rows []CandidatePoolRow
	err := s.db.Order("read_at_ms DESC, rank ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

// PoolSummary reports seated vs cut for the most recent read — the D6 shape.
func (s *CandidatePoolStore) PoolSummary() (seated, cut int, readAtMs int64) {
	if s == nil || s.db == nil {
		return 0, 0, 0
	}
	var newest int64
	if err := s.db.Model(&CandidatePoolRow{}).Select("COALESCE(MAX(read_at_ms),0)").Scan(&newest).Error; err != nil || newest == 0 {
		return 0, 0, 0
	}
	var rows []CandidatePoolRow
	if err := s.db.Where("read_at_ms = ?", newest).Find(&rows).Error; err != nil {
		return 0, 0, newest
	}
	for _, r := range rows {
		if r.Seated {
			seated++
		} else {
			cut++
		}
	}
	return seated, cut, newest
}
