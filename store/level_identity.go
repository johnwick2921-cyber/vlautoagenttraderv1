package store

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"nofx/levelidentity"
	"strings"
	"time"
)

// Identity events are recorded facts, scoped to trader + plan VERSION. No
// counter is inferred from an absent legacy field or from a polling loop.
type LevelIdentityEvent struct {
	PlanID     string          `json:"plan_id"`
	Version    int             `json:"version"`
	ScenarioID string          `json:"scenario_id"`
	Kind       string          `json:"kind"`
	Detail     json.RawMessage `json:"detail"`
	At         time.Time       `json:"at"`
}

func identityEventPrefix(traderID string) string {
	return "level_identity_event:" + base64.RawURLEncoding.EncodeToString([]byte(traderID)) + ":"
}
func (s *Store) RecordLevelIdentityEvent(traderID string, e LevelIdentityEvent) (bool, error) {
	if s == nil || traderID == "" || e.PlanID == "" || e.Version <= 0 || e.ScenarioID == "" || e.At.IsZero() {
		return false, fmt.Errorf("identity event missing scope or clock")
	}
	switch e.Kind {
	case "named", "unnamed", "unresolved", "heuristic_disagreed":
	default:
		return false, fmt.Errorf("unknown identity event kind %q", e.Kind)
	}
	tuple, _ := json.Marshal([]any{e.PlanID, e.Version, e.ScenarioID, e.Kind})
	key := identityEventPrefix(traderID) + base64.RawURLEncoding.EncodeToString(tuple)
	raw, err := json.Marshal(e)
	if err != nil {
		return false, err
	}
	r := s.gdb.Exec("INSERT INTO system_config (key,value) VALUES (?,?) ON CONFLICT(key) DO NOTHING", key, string(raw))
	return r.RowsAffected == 1, r.Error
}

type LevelIdentityCounts struct{ Named, Unnamed, Unresolved, HeuristicDisagreed int64 }

func (s *Store) LevelIdentityCounts(traderID string) (LevelIdentityCounts, error) {
	var out LevelIdentityCounts
	if s == nil {
		return out, fmt.Errorf("store unavailable")
	}
	prefix := identityEventPrefix(traderID)
	var raws []string
	if err := s.gdb.Raw("SELECT value FROM system_config WHERE substr(key,1,?)=?", len(prefix), prefix).Scan(&raws).Error; err != nil {
		return out, err
	}
	for _, raw := range raws {
		var e LevelIdentityEvent
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return out, err
		}
		switch e.Kind {
		case "named":
			out.Named++
		case "unnamed":
			out.Unnamed++
		case "unresolved":
			out.Unresolved++
		case "heuristic_disagreed":
			out.HeuristicDisagreed++
		}
	}
	return out, nil
}

// The measured W-TF boot, not a guessed date or the current process start.
var LevelIdentityCaptureEra = time.Date(2026, 9, 10, 22, 10, 33, 0, time.UTC)

type IdentityBackfillRow struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}
type IdentityBackfillResult struct {
	Recomputed     int                   `json:"recomputed"`
	Unrecomputable int                   `json:"unrecomputable"`
	Untouched      int                   `json:"untouched"`
	Rows           []IdentityBackfillRow `json:"rows"`
}

// ClassifyIdentityBackfill never substitutes candle OPEN for the separate close.
// A pre-era row has no identity inputs by construction and stays untouched.
func ClassifyIdentityBackfill(r TouchOutcomeRow, in levelidentity.Inputs) (*string, string) {
	if r.CreatedAt.Before(LevelIdentityCaptureEra) {
		return nil, "untouched"
	}
	id, missing := levelidentity.ID(in)
	if id == nil {
		return nil, "unrecomputable:" + missing
	}
	return id, "recomputed"
}

// BackfillLevelIdentity reads only complete, already-recorded inputs from this
// exact plan version. No price join, inferred bounds, inferred TF or partial hash.
// Legacy plans are never rewritten. Every episode classification is durable in
// a sidecar report; only a fully evidenced existing named identity may be written.
func (s *Store) BackfillLevelIdentity(traderID string) (IdentityBackfillResult, error) {
	out := IdentityBackfillResult{Rows: []IdentityBackfillRow{}}
	if s == nil {
		return out, fmt.Errorf("store unavailable")
	}
	var rows []TouchOutcomeRow
	if err := s.gdb.Where("trader_id = ?", traderID).Find(&rows).Error; err != nil {
		return out, err
	}
	for _, r := range rows {
		in := levelidentity.Inputs{Symbol: r.Symbol, Kind: r.LevelKind}
		if r.LevelID != nil {
			p, err := s.Plan().GetPlan(r.PlanID, r.PlanVersion)
			if err == nil && p != nil && p.StrategyID == traderID {
				var doc struct {
					Levels []struct {
						ID *string `json:"id"`
						levelidentity.Inputs
					} `json:"identity_levels"`
				}
				if json.Unmarshal([]byte(p.Doc), &doc) == nil {
					for _, l := range doc.Levels {
						if l.ID != nil && *l.ID == *r.LevelID {
							in = l.Inputs
							break
						}
					}
				}
			}
		}
		id, status := ClassifyIdentityBackfill(r, in)
		if status == "recomputed" {
			if r.LevelID == nil || id == nil || *id != *r.LevelID {
				status = "unrecomputable:named_identity_mismatch"
			} else {
				if err := s.gdb.Model(&TouchOutcomeRow{}).Where("id = ? AND trader_id = ? AND level_id = ?", r.ID, traderID, *r.LevelID).Update("level_id", *id).Error; err != nil {
					return out, err
				}
			}
		}
		switch {
		case status == "untouched":
			out.Untouched++
		case status == "recomputed":
			out.Recomputed++
		case strings.HasPrefix(status, "unrecomputable:"):
			out.Unrecomputable++
		}
		out.Rows = append(out.Rows, IdentityBackfillRow{r.ID, status})
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	err = s.SetSystemConfig("level_identity_backfill:"+traderID, string(raw))
	return out, err
}
