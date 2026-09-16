package store

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"time"
)

const (
	ScenarioDeathRecordPrefix = "scenario_death:"
	LivenessEventPrefix       = "plan_liveness_event:"
	LivenessExhaustionWarning = "exhaustion_warning"
	LivenessBornDeadRefusal   = "born_dead_refusal"
	LivenessAuthoredUnknown   = "authored_unknown"
)

// This is display freshness, never an entry/wake cutoff.
const ScenarioSnapshotMaxAge = 5 * time.Minute

type ScenarioLiveness struct {
	Total      int        `json:"total"`
	Tradeable  *int       `json:"tradeable"`
	Unknown    int        `json:"unknown"`
	ObservedAt *time.Time `json:"observed_at,omitempty"`
	Reason     string     `json:"reason,omitempty"`
}

// ScenarioLivenessFor reads only this version's current evaluator snapshot.
// Historical first-death records do not silently make a reversible verdict
// terminal. Missing/stale/unevaluable status never becomes a plausible zero.
func (s *Store) ScenarioLivenessFor(traderID, planID string, version int, ids []string, now time.Time) ScenarioLiveness {
	out := ScenarioLiveness{Total: len(ids), Unknown: len(ids), Reason: "versioned scenario snapshot unavailable"}
	if s == nil || planID == "" || version <= 0 {
		return out
	}
	raw, err := s.GetSystemConfig(ScenarioMetaKey(traderID, planID, version))
	if err != nil {
		return out
	}
	var meta struct {
		ObservedAt time.Time `json:"observed_at"`
	}
	if json.Unmarshal([]byte(raw), &meta) != nil || meta.ObservedAt.IsZero() {
		return out
	}
	out.ObservedAt = &meta.ObservedAt
	if now.Before(meta.ObservedAt) || now.Sub(meta.ObservedAt) > ScenarioSnapshotMaxAge {
		out.Reason = "versioned scenario snapshot stale or in the future"
		return out
	}
	raw, err = s.GetSystemConfig(ScenarioStatusKey(traderID, planID, version))
	if err != nil {
		return out
	}
	var statuses map[string]string
	if json.Unmarshal([]byte(raw), &statuses) != nil {
		return out
	}
	live := 0
	out.Unknown = 0
	for _, id := range ids {
		switch statuses[id] {
		case "waiting", "armed", "triggered":
			live++
		case "invalidated", "expired":
		default:
			out.Unknown++
		}
	}
	if out.Unknown > 0 {
		out.Reason = "one or more scenario verdicts unevaluable"
		return out
	}
	out.Tradeable = &live
	out.Reason = "current evaluator status; order eligibility is separate"
	return out
}

type PlanLivenessCounts struct {
	DeathsRecorded     int64
	ExhaustionWarnings int64
	BornDeadRefusals   int64
	AuthoredUnknown    int64
}

// RecordPlanLivenessEvent records a fact, not a derived plan-row count. identity
// is a version for exhaustion, or a candidate attempt for write validation.
func (s *Store) RecordPlanLivenessEvent(kind, identity string, now time.Time, detail string) (bool, error) {
	if s == nil || identity == "" || now.IsZero() {
		return false, fmt.Errorf("event identity and clock required")
	}
	switch kind {
	case LivenessExhaustionWarning, LivenessBornDeadRefusal, LivenessAuthoredUnknown:
	default:
		return false, fmt.Errorf("unknown liveness event")
	}
	if kind != LivenessExhaustionWarning {
		eventID, err := uuid.NewRandom()
		if err != nil {
			return false, fmt.Errorf("liveness event identity unavailable: %w", err)
		}
		identity += ":" + eventID.String()
	}
	raw, err := json.Marshal(struct {
		At     time.Time `json:"at"`
		Detail string    `json:"detail"`
	}{now, detail})
	if err != nil {
		return false, err
	}
	result := s.gdb.Exec("INSERT INTO system_config (key,value) VALUES (?,?) ON CONFLICT(key) DO NOTHING", LivenessEventPrefix+kind+":"+identity, string(raw))
	return result.RowsAffected == 1, result.Error
}

func (s *Store) PlanLivenessCounts() (PlanLivenessCounts, error) {
	var out PlanLivenessCounts
	if s == nil {
		return out, fmt.Errorf("store unavailable")
	}
	for prefix, dst := range map[string]*int64{
		ScenarioDeathRecordPrefix:                             &out.DeathsRecorded,
		LivenessEventPrefix + LivenessExhaustionWarning + ":": &out.ExhaustionWarnings,
		LivenessEventPrefix + LivenessBornDeadRefusal + ":":   &out.BornDeadRefusals,
		LivenessEventPrefix + LivenessAuthoredUnknown + ":":   &out.AuthoredUnknown,
	} {
		if err := s.gdb.Raw("SELECT COUNT(*) FROM system_config WHERE substr(key,1,?)=?", len(prefix), prefix).Scan(dst).Error; err != nil {
			return PlanLivenessCounts{}, err
		}
	}
	return out, nil
}

// ScenarioDeath is the existing first-observed invalidation stamp, now scoped
// to a version and bound to the anchor actually evaluated. It is not a claim
// about when an earlier candle first satisfied the authored invalidation.
type ScenarioDeath struct {
	PlanID     string    `json:"plan_id"`
	Version    int       `json:"version"`
	ScenarioID string    `json:"scenario_id"`
	Anchor     float64   `json:"anchor"`
	Price      float64   `json:"price"`
	Cause      string    `json:"cause"`
	Condition  string    `json:"condition"`
	Basis      string    `json:"basis"`
	ObservedAt time.Time `json:"observed_at"`
}

// RecordScenarioDeath extends the existing recorder; no legacy stamp is read,
// copied, deleted or guessed into the new namespace. First writer wins atomically.
func (s *Store) RecordScenarioDeath(traderID string, r ScenarioDeath) (bool, error) {
	if s == nil || r.PlanID == "" || r.Version <= 0 || r.ScenarioID == "" || r.ObservedAt.IsZero() || r.Anchor <= 0 || math.IsNaN(r.Anchor) || math.IsInf(r.Anchor, 0) || r.Price <= 0 || math.IsNaN(r.Price) || math.IsInf(r.Price, 0) || r.Cause != "invalidated" || r.Condition == "" || r.Basis == "" {
		return false, fmt.Errorf("incomplete scenario death identity or evidence")
	}
	b, err := json.Marshal(r)
	if err != nil {
		return false, err
	}
	result := s.gdb.Exec("INSERT INTO system_config (key,value) VALUES (?,?) ON CONFLICT(key) DO NOTHING", ScenarioInvalidatedAtKey(traderID, r.PlanID, r.Version, r.ScenarioID), string(b))
	return result.RowsAffected == 1, result.Error
}

// ScenarioDeathFor rejects corrupt/mismatched evidence instead of attaching a
// timestamp to a different version or overlay anchor. Missing is unknown.
func (s *Store) ScenarioDeathFor(traderID, planID string, version int, scenarioID string, anchor float64) (*ScenarioDeath, error) {
	raw, err := s.GetSystemConfig(ScenarioInvalidatedAtKey(traderID, planID, version, scenarioID))
	if err != nil || raw == "" {
		return nil, err
	}
	var r ScenarioDeath
	if err = json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, err
	}
	if r.PlanID != planID || r.Version != version || r.ScenarioID != scenarioID || r.Anchor != anchor || r.ObservedAt.IsZero() {
		return nil, fmt.Errorf("scenario death evidence does not match version and anchor")
	}
	return &r, nil
}
