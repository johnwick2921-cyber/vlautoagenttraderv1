package store

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PICTURE-HTF (2026-09-19) — the durable opportunity ledger for the owner's
// two-picture method. One row per opportunity, identified by a unique
// opportunity key; every stage transition is a row update with a reason, and
// only RECEIVED broker evidence may establish working/filled/rejected.
//
// Lifecycle: watching → confirmed → place_pending → working/filled.
// Alternative outcomes: refused, expired, rejected (broker), or lost (an
// ambiguous send that stays place_pending until reconciled against NT8).

// PictureHtfOpportunityDB is one opportunity row.
type PictureHtfOpportunityDB struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`

	// OppKey is the durable uniqueness key: strategy, account, contract,
	// direction, level identity, and the H1 close time. Repeated frames and
	// restarts cannot produce a duplicate entry.
	OppKey string `gorm:"uniqueIndex;size:256"`

	TraderID   string `gorm:"index"`
	StrategyID string `gorm:"index"`
	Account    string `gorm:"index"`
	Contract   string
	Symbol     string
	Direction  string // long | short
	RuleVer    int    // the picture_htf rule version this row was evaluated under

	// Stage: watching | confirmed | place_pending | working | filled |
	// refused | expired | rejected | lost.
	Stage       string `gorm:"index"`
	StageReason string

	// Level evidence.
	LevelRole     string // resistance | support
	LevelBodyTop  float64
	LevelBodyBot  float64
	LevelWickHi   float64
	LevelWickLo   float64
	LevelBarOpen  int64 // the source 4H pivot candle open time (ms)
	LevelKnowable int64 // when the level became usable (ms)

	// Confirmation evidence.
	H1PrevClose  float64
	H1NewClose   float64
	H1Boundary   float64
	H1OpenTime   int64 // the confirming H1 candle's open time (ms)
	H1CloseTime  int64 // the confirming H1 candle's close time (ms)
	H1Completion int64 // when the completed H1 bar was received (ms)

	// Eligibility.
	WindowOpen   int64  // the new 5m interval start (ms)
	WindowClose  int64  // windowOpen + entry_window_sec (ms)
	FreshVerdict string // fresh | stale | future | unknown

	// Geometry.
	EntryRef     float64 // intended reference (5m open at evaluation)
	StopPx       float64
	StopSource   string // swing candle evidence, e.g. "5m swing low @<ts>"
	TargetPx     float64
	TargetZone   string
	Qty          float64
	RREstimate   float64 // pre-submit estimate
	RRConfigured float64

	// Submission + broker evidence.
	SignalID      string `gorm:"index"`
	SubmittedAt   int64  // wall clock at command creation
	BrokerOrderID string
	BrokerStatus  string
	FillPrice     float64
	FillQty       float64
	FillRR        float64 // actual-fill R:R, recorded separately from the estimate
	RejectReason  string  // "reason unavailable" when absent

	// Momentum observation at the last completed H1 (advisory).
	MomentumStall bool
	MomentumDir   string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName pins the table.
func (PictureHtfOpportunityDB) TableName() string { return "picture_htf_opportunities" }

// PictureHtfOppKey builds the durable uniqueness key.
func PictureHtfOppKey(strategyID, account, contract, direction, levelRole string, levelBarOpen, h1CloseTime int64) string {
	return strings.ToLower(fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d", strategyID, account, contract, direction, levelRole, levelBarOpen, h1CloseTime))
}

// PictureHtfClaim atomically inserts the row if the opportunity key is new.
// Returns (row, true) on a fresh claim, (nil, false) when the key already
// exists (duplicate frame/restart — never a second ROW), and an error on
// store failure. NOTE: row uniqueness is NOT submission uniqueness — a second
// broker order is prevented by PictureHtfClaimSubmission (the atomic
// confirmed→place_pending transition below), which is the actual
// one-execution-owner claim.
func (s *Store) PictureHtfClaim(row *PictureHtfOpportunityDB) (*PictureHtfOpportunityDB, bool, error) {
	if s == nil || s.gdb == nil {
		return nil, false, fmt.Errorf("store unavailable")
	}
	if strings.TrimSpace(row.OppKey) == "" {
		return nil, false, fmt.Errorf("picture_htf: empty opportunity key")
	}
	res := s.gdb.Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	if res.Error != nil {
		return nil, false, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, false, nil // key already claimed
	}
	return row, true, nil
}

// PictureHtfGet returns the row for an opportunity key (nil, false when absent).
func (s *Store) PictureHtfGet(oppKey string) (*PictureHtfOpportunityDB, bool, error) {
	if s == nil || s.gdb == nil {
		return nil, false, fmt.Errorf("store unavailable")
	}
	var row PictureHtfOpportunityDB
	res := s.gdb.Where("opp_key = ?", oppKey).First(&row)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	if res.Error != nil {
		return nil, false, res.Error
	}
	return &row, true, nil
}

// PictureHtfTransition moves a row from one stage to the next and records the
// reason. It refuses transitions that would fabricate broker evidence:
// only RECEIVED frames may move a row to working/filled/rejected, so the
// caller is expected to pass the stage the frame established.
func (s *Store) PictureHtfTransition(oppKey, stage, reason string) error {
	if s == nil || s.gdb == nil {
		return fmt.Errorf("store unavailable")
	}
	return s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ?", oppKey).
		Updates(map[string]any{"stage": stage, "stage_reason": reason}).Error
}

// PictureHtfRefuse settles a refusal onto an opportunity ONLY while no send of
// it has started (W-EXEC-TRUTH W0, CTO Q8): a confirmed row, or a place_pending
// row that carries no submission stamp. A row whose send started — stamped
// place_pending, working, filled — or one already terminal is never
// overwritten: a later refusal of the same hour's opportunity used to clobber
// a submitted row to 'expired', and the broker consumer then DROPPED a received
// FILLED frame for it. Returns whether the row moved.
func (s *Store) PictureHtfRefuse(oppKey, stage, reason string) (bool, error) {
	if s == nil || s.gdb == nil {
		return false, fmt.Errorf("store unavailable")
	}
	res := s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ? AND (stage = ? OR (stage = ? AND submitted_at = 0))", oppKey, "confirmed", StatePlacePending).
		Updates(map[string]any{"stage": stage, "stage_reason": reason})
	return res.RowsAffected > 0, res.Error
}

// PictureHtfMarkBroker stamps the received broker evidence. Only received
// order events may call this; absent fields stay empty (unknown/unavailable,
// never a fabricated success).
func (s *Store) PictureHtfMarkBroker(oppKey, stage, brokerOrderID, brokerStatus, rejectReason string, fillPrice, fillQty float64) error {
	if s == nil || s.gdb == nil {
		return fmt.Errorf("store unavailable")
	}
	updates := map[string]any{"stage": stage, "broker_order_id": brokerOrderID, "broker_status": brokerStatus, "reject_reason": rejectReason}
	if fillPrice > 0 {
		updates["fill_price"] = fillPrice
	}
	if fillQty > 0 {
		updates["fill_qty"] = fillQty
	}
	return s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ?", oppKey).
		Updates(updates).Error
}

// PictureHtfMarkBrokerState is the full broker-state stamp used by the live
// consumer and the reconciliation sweep: stage + order id + status + reason +
// fill price/qty + the actual-fill R:R computed by the caller. Only RECEIVED
// broker evidence may call this. Zero fill fields are left untouched.
func (s *Store) PictureHtfMarkBrokerState(oppKey, stage, brokerOrderID, brokerStatus, rejectReason string, fillPrice, fillQty, fillRR float64) error {
	if s == nil || s.gdb == nil {
		return fmt.Errorf("store unavailable")
	}
	updates := map[string]any{
		"stage":           stage,
		"broker_status":   brokerStatus,
		"reject_reason":   rejectReason,
		"broker_order_id": brokerOrderID,
	}
	if fillPrice > 0 {
		updates["fill_price"] = fillPrice
	}
	if fillQty > 0 {
		updates["fill_qty"] = fillQty
	}
	if fillRR > 0 {
		updates["fill_rr"] = fillRR
	}
	return s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ?", oppKey).
		Updates(updates).Error
}

// PictureHtfAppendBrokerStatus appends one evidence note to broker_status
// WITHOUT touching the stage or fill fields. Protection-leg events and
// reconciliation markers accumulate instead of overwriting each other; a note
// already present is not duplicated (idempotent).
func (s *Store) PictureHtfAppendBrokerStatus(oppKey, note string) error {
	if s == nil || s.gdb == nil {
		return fmt.Errorf("store unavailable")
	}
	return s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ? AND (broker_status IS NULL OR broker_status = '' OR instr(broker_status, ?) = 0)", oppKey, note).
		Update("broker_status", gorm.Expr("CASE WHEN broker_status IS NULL OR broker_status = '' THEN ? ELSE broker_status || '; ' || ? END", note, note)).Error
}

// PictureHtfBySignal returns the rows stamped with a broker signal id (the
// entry uuid — protective legs ride the same signal id).
func (s *Store) PictureHtfBySignal(signalID string) ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	if strings.TrimSpace(signalID) == "" {
		return nil, nil
	}
	var rows []PictureHtfOpportunityDB
	err := s.gdb.Where("signal_id = ?", signalID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []PictureHtfOpportunityDB{}
	}
	return rows, nil
}

// PictureHtfClaimSubmission is the atomic submission-ownership claim (addendum
// #4): it moves an opportunity from confirmed to place_pending ONLY when no
// signal is registered yet, in one SQL statement. Exactly ONE caller (across
// both executors, concurrent callbacks, and restarts) wins; the losers must
// not send. An ambiguous place_pending row (signal set, no broker evidence)
// blocks re-entry until reconciled — it is never blindly resent.
func (s *Store) PictureHtfClaimSubmission(oppKey, signalID string) (bool, error) {
	if s == nil || s.gdb == nil {
		return false, fmt.Errorf("store unavailable")
	}
	res := s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ? AND stage = ? AND (signal_id = '' OR signal_id IS NULL)", oppKey, "confirmed").
		Updates(map[string]any{"stage": "place_pending", "signal_id": signalID})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// PictureHtfStampSignal records the broker signal ID the atomic owner SENT,
// guarded by the claim's ownership marker (the synthetic signal set by
// PictureHtfClaimSubmission). It refuses to stamp a row the caller does not
// own. submitted_at is the send-side clock, distinct from broker evidence.
func (s *Store) PictureHtfStampSignal(oppKey, claimID, brokerSignalID string) error {
	if s == nil || s.gdb == nil {
		return fmt.Errorf("store unavailable")
	}
	res := s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ? AND signal_id = ?", oppKey, claimID).
		Updates(map[string]any{"signal_id": brokerSignalID, "submitted_at": time.Now().UnixMilli()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("picture_htf: signal stamp refused — the row is not owned by claim %q", claimID)
	}
	return nil
}

// PictureHtfPendingByTrader lists rows awaiting reconciliation (an ambiguous
// send that never got broker evidence) for the boot/restart sweep.
func (s *Store) PictureHtfPendingByTrader(traderID string) ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	var rows []PictureHtfOpportunityDB
	err := s.gdb.Where("trader_id = ? AND stage = ?", traderID, "place_pending").
		Order("created_at ASC").Find(&rows).Error
	return rows, err
}

// PictureHtfRecoverableByTrader lists the rows whose broker outcome is still
// unresolved and may be settled by the reconciliation sweep: place_pending
// (submission in flight, no receipt yet) AND working (a receipt was received
// but no terminal outcome — a later fill may arrive on the fill stream with
// no matching order_update frame, so a working receipt must never be treated
// as settled). Canonical stage constants only.
func (s *Store) PictureHtfRecoverableByTrader(traderID string) ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	var rows []PictureHtfOpportunityDB
	err := s.gdb.Where("trader_id = ? AND (stage = ? OR stage = ?)", traderID, StatePlacePending, StateWorking).
		Order("created_at ASC").Find(&rows).Error
	return rows, err
}

// PictureHtfRecoverableAll is PictureHtfRecoverableByTrader for EVERY trader
// (W-ONE-BUTTON M2 installation gate): place_pending or working, any trader
// id — a stopped or removed trader's send is still unresolved.
func (s *Store) PictureHtfRecoverableAll() ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	var rows []PictureHtfOpportunityDB
	err := s.gdb.Where("stage = ? OR stage = ?", StatePlacePending, StateWorking).
		Order("created_at ASC").Find(&rows).Error
	return rows, err
}

// PictureHtfByTrader lists the trader's opportunity ledger, newest first.
func (s *Store) PictureHtfByTrader(traderID string, limit int) ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []PictureHtfOpportunityDB
	err := s.gdb.Where("trader_id = ?", traderID).
		Order("created_at DESC, id DESC").
		Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []PictureHtfOpportunityDB{} // an empty computed list is [], never null
	}
	return rows, nil
}

// MigratePictureHtf creates/updates the opportunity table.
func (s *Store) MigratePictureHtf() error {
	if s == nil || s.gdb == nil {
		return fmt.Errorf("store unavailable")
	}
	return s.gdb.AutoMigrate(&PictureHtfOpportunityDB{})
}
