package store

import (
	"fmt"
	"strings"
)

// W-EXEC-TRUTH W5 — THE PICTURE HAND-OFF SETTLE (D16/D17).
//
// A Picture opportunity no longer owns a send. The evaluator claims it
// (confirmed → place_pending, claim id "picture-htf-<ms>", submitted_at 0) and
// hands it to the Day Plan: the plan source records it as a machine scenario
// and then settles the opportunity row HERE, out of place_pending, into
// "planned". The armed executor places the scenario like any other plan
// scenario; the Picture row itself never reaches the wire.
//
// "planned" is deliberately outside every reader of an unresolved send:
//   - PictureHtfRecoverableAll / PictureHtfRecoverableByTrader read
//     place_pending | working only (the installation gate, the reconcile
//     sweep, the entry latch's ledger clause);
//   - PictureSendStarted is working, or place_pending with a stamp;
//   - the broker consumer matches on signal_id, and a planned row keeps the
//     synthetic claim id, which no broker frame ever carries.
//
// This file stays out of store/picture_htf.go (lane 103 adds W4 columns
// there); it works on the same table through the same model.

// PictureStagePlanned is the stage of an opportunity recorded as a Day Plan
// scenario (the hand-off settled it; it was never sent by the Picture path).
const PictureStagePlanned = "planned"

// PictureHtfClaimPrefix is the evaluator's synthetic claim id prefix
// ("picture-htf-<ms>"): a row carrying it was claimed but never stamped with
// a broker signal.
const PictureHtfClaimPrefix = "picture-htf-"

// PictureHtfHandOff settles ONE claimed opportunity into "planned" — a
// compare-and-set on the claim: place_pending ∧ signal_id = claimID ∧
// submitted_at = 0. A row whose send started (stamped), a row another owner
// claimed, or a row already settled is never moved. Returns whether it moved.
func (s *Store) PictureHtfHandOff(oppKey, claimID, reason string) (bool, error) {
	if s == nil || s.gdb == nil {
		return false, fmt.Errorf("store unavailable")
	}
	if strings.TrimSpace(oppKey) == "" || strings.TrimSpace(claimID) == "" {
		return false, fmt.Errorf("picture hand-off: opportunity key and claim id required")
	}
	res := s.gdb.Model(&PictureHtfOpportunityDB{}).
		Where("opp_key = ? AND stage = ? AND signal_id = ? AND submitted_at = 0", oppKey, StatePlacePending, claimID).
		Updates(map[string]any{"stage": PictureStagePlanned, "stage_reason": reason})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// PictureHtfPlannedByTrader lists the trader's opportunities recorded as Day
// Plan scenarios, newest first. An empty computed list is [], never null.
func (s *Store) PictureHtfPlannedByTrader(traderID string) ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	var rows []PictureHtfOpportunityDB
	if err := s.gdb.Where("trader_id = ? AND stage = ?", traderID, PictureStagePlanned).
		Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []PictureHtfOpportunityDB{}
	}
	return rows, nil
}

// PictureHtfHandOffPendingByTrader lists the trader's claimed-but-unsettled
// hand-offs (the D17 sweep's input): place_pending, a synthetic claim id, no
// submission stamp. A legacy row the old send path stamped (submitted_at > 0)
// is never listed — it stays with the broker reconcile sweep.
func (s *Store) PictureHtfHandOffPendingByTrader(traderID string) ([]PictureHtfOpportunityDB, error) {
	if s == nil || s.gdb == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	var rows []PictureHtfOpportunityDB
	if err := s.gdb.Where("trader_id = ? AND stage = ? AND submitted_at = 0 AND signal_id LIKE ?",
		traderID, StatePlacePending, PictureHtfClaimPrefix+"%").
		Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// PictureHandOffRecord names where an opportunity's machine scenario was
// recorded.
type PictureHandOffRecord struct {
	PlanID         string
	PlanVersion    int
	OverlayVersion int // 0 = the scenario lives in a machine plan's own doc
}

// PictureHandOffRecordedFor finds the FIRST record of an opportunity's machine
// scenario in the trader's plan store: a machine overlay whose overlay_id is
// "picture:<ref>" (the plan source's id; re-appends reuse it), or a machine
// plan (trigger_reason machine:*) whose doc carries a scenario with
// machine.ref = ref. ok=false when the machine never recorded it.
func (s *Store) PictureHandOffRecordedFor(traderID, ref string) (PictureHandOffRecord, bool, error) {
	if s == nil || s.gdb == nil {
		return PictureHandOffRecord{}, false, fmt.Errorf("store unavailable")
	}
	if strings.TrimSpace(ref) == "" {
		return PictureHandOffRecord{}, false, nil
	}
	var ov []PlanOverlayDB
	if err := s.gdb.Table("plan_overlays").
		Where("overlay_id = ? AND origin LIKE ? AND plan_id IN (SELECT plan_id FROM plans WHERE strategy_id = ?)",
			"picture:"+ref, MachinePlanTriggerPrefix+"%", traderID).
		Order("plan_version ASC, overlay_version ASC").Limit(1).Find(&ov).Error; err != nil {
		return PictureHandOffRecord{}, false, err
	}
	if len(ov) == 1 {
		return PictureHandOffRecord{PlanID: ov[0].PlanID, PlanVersion: ov[0].PlanVersion, OverlayVersion: ov[0].OverlayVersion}, true, nil
	}
	var hits []struct {
		PlanID  string `gorm:"column:plan_id"`
		Version int    `gorm:"column:version"`
	}
	if err := s.gdb.Raw(`SELECT p.plan_id AS plan_id, p.version AS version
		FROM plans p, json_each(p.doc, '$.scenarios') sc
		WHERE p.strategy_id = ? AND p.trigger_reason LIKE ? AND json_extract(sc.value, '$.machine.ref') = ?
		ORDER BY p.version ASC LIMIT 1`, traderID, MachinePlanTriggerPrefix+"%", ref).Scan(&hits).Error; err != nil {
		return PictureHandOffRecord{}, false, err
	}
	if len(hits) == 1 {
		return PictureHandOffRecord{PlanID: hits[0].PlanID, PlanVersion: hits[0].Version}, true, nil
	}
	return PictureHandOffRecord{}, false, nil
}

// RedactPictureOppKey is the ONE redactor for a Picture opportunity key in a
// log line or an error (L12, CTO 1790197560916): the key is
// "<strategy>|<account>|<contract>|<direction>|<role>|<levelOpen>|<h1Close>"
// (PictureHtfOppKey), so its second segment is the ACCOUNT name. Every
// W5-authored line prints the key through this; the plan doc and the ledger
// keep the full key (storage, not a log). A key with no account segment is
// returned unchanged.
func RedactPictureOppKey(key string) string {
	parts := strings.Split(key, "|")
	if len(parts) < 2 {
		return key
	}
	parts[1] = "…"
	return strings.Join(parts, "|")
}
