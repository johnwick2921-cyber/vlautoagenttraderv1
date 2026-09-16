package store

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"math"
	"strings"
)

type StructuralStopConfig struct {
	BufferPoints        *float64 `json:"buffer_points,omitempty"`
	RoundTripCostPoints *float64 `json:"round_trip_cost_points,omitempty"`
}

type StructuralStopPolicy struct {
	BufferPoints, CostPoints, MinRR float64
	BufferKnown, CostKnown          bool
	BufferSource, Calibration       string
	Percentile                      int
}

// C5, in-sample only: 6,181 HELD first touches, 2022-04-11–2025-09-11.
// p95 far-edge penetration 4.3675648248 pts, rounded OUTWARD to MNQ tick.
// Chosen [I] tolerance, not a validated universal buffer or 95% win probability.
const StructuralBufferMNQDefault = 4.5
const StructuralBufferPercentile = 95
const StructuralBufferCalibration = "C5-H12-IS-6181-p95-20260912"

// Until the C5 calibration is recorded, no implicit buffer exists. Explicit
// fixture/owner values remain research inputs, not externally validated stops.
func ResolveStructuralStop(c *StrategyConfig, symbol string) StructuralStopPolicy {
	p := StructuralStopPolicy{CostPoints: 2, CostKnown: true, BufferSource: "unresolved", Calibration: "unresolved"}
	if symbol == "MNQ" {
		p.BufferPoints = StructuralBufferMNQDefault
		p.BufferKnown = true
		p.BufferSource = "ResolveStructuralStop:C5_MNQ_default[I]"
		p.Calibration = StructuralBufferCalibration
		p.Percentile = StructuralBufferPercentile
	}
	if c == nil {
		return p
	}
	p.MinRR = c.RiskControl.MinRiskRewardRatio
	if c.DayPlan != nil && c.DayPlan.StructuralStop != nil {
		s := c.DayPlan.StructuralStop
		if s.BufferPoints != nil {
			p.BufferPoints = *s.BufferPoints
			p.BufferKnown = positiveFinite(*s.BufferPoints)
			p.BufferSource = "day_plan.structural_stop.buffer_points"
			p.Calibration = "owner_override[I]"
			p.Percentile = 0
		}
		if s.RoundTripCostPoints != nil {
			p.CostPoints = *s.RoundTripCostPoints
			p.CostKnown = *s.RoundTripCostPoints >= 0 && !math.IsNaN(*s.RoundTripCostPoints) && !math.IsInf(*s.RoundTripCostPoints, 0)
		}
	}
	return p
}
func positiveFinite(x float64) bool { return x > 0 && !math.IsNaN(x) && !math.IsInf(x, 0) }

// The durable composition record also serves refusals: quantity is zero until
// all admission gates pass. Missing fields are absent, not fabricated zeros.
type StructuralGeometryRecord struct {
	TradeDate        string   `json:"trade_date"`
	TraderID         string   `json:"trader_id"`
	PlanID           string   `json:"plan_id"`
	Version          int      `json:"version"`
	Scenario         string   `json:"scenario"`
	Leg              int      `json:"leg"`
	TimeMs           int64    `json:"time_ms"`
	Symbol           string   `json:"symbol"`
	Side             string   `json:"side"`
	Entry            float64  `json:"entry"`
	Stop             *float64 `json:"stop,omitempty"`
	Target           *float64 `json:"target,omitempty"`
	ZoneLo           *float64 `json:"zone_lo,omitempty"`
	ZoneHi           *float64 `json:"zone_hi,omitempty"`
	TargetLo         *float64 `json:"target_lo,omitempty"`
	TargetHi         *float64 `json:"target_hi,omitempty"`
	ZoneNames        []string `json:"zone_names,omitempty"`
	TargetNames      []string `json:"target_names,omitempty"`
	Buffer           *float64 `json:"buffer,omitempty"`
	BufferSource     string   `json:"buffer_source"`
	Calibration      string   `json:"calibration"`
	Percentile       int      `json:"percentile,omitempty"`
	StopSource       string   `json:"stop_source"`
	StopSourceReason string   `json:"stop_source_reason,omitempty"`
	RiskPoints       *float64 `json:"risk_points,omitempty"`
	GainPoints       *float64 `json:"gain_points,omitempty"`
	NetGainPoints    *float64 `json:"net_gain_points,omitempty"`
	LossUSD          *float64 `json:"loss_usd,omitempty"`
	// Historical records may carry the superseded per-trade cap; never used for admission.
	RiskCapUSD *float64 `json:"risk_cap_usd,omitempty"`
	Quantity   int      `json:"quantity"`
	Reason     string   `json:"reason"`
	Detail     string   `json:"detail"`
}

func StructuralGeometryKey(r StructuralGeometryRecord) string {
	return fmt.Sprintf("structural_geometry:%s:%s:v%d:%s:leg%d", r.TraderID, r.PlanID, r.Version, r.Scenario, r.Leg)
}
func (s *Store) SaveStructuralGeometry(r StructuralGeometryRecord) error {
	if s == nil {
		return fmt.Errorf("geometry store unavailable")
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return s.Transaction(func(tx *gorm.DB) error {
		// Count each plan/version/scenario/leg/reason once per CME day. The
		// transient pending-gates record must not reset refusal deduplication.
		reasons := []string{}
		if r.StopSource == "atr_fallback" {
			reasons = append(reasons, "atr_fallback")
		}
		if r.Reason != "" && r.Reason != "admitted" && r.Reason != "pending_gates" {
			reasons = append(reasons, r.Reason)
		}
		for _, reason := range reasons {
			seenKey := "structural_seen:" + r.TradeDate + ":" + StructuralGeometryKey(r) + ":" + reason
			seen := tx.Exec("INSERT INTO system_config (key,value) VALUES (?, '1') ON CONFLICT(key) DO NOTHING", seenKey)
			if seen.Error != nil {
				return seen.Error
			}
			if seen.RowsAffected == 0 {
				continue
			}
			key := fmt.Sprintf("structural_counts:%s:%s:%s", r.TraderID, r.TradeDate, reason)
			if err := tx.Exec("INSERT INTO system_config (key,value) VALUES (?, '1') ON CONFLICT(key) DO UPDATE SET value=CAST(CAST(value AS INTEGER)+1 AS TEXT)", key).Error; err != nil {
				return err
			}
		}
		return tx.Exec("INSERT INTO system_config (key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", StructuralGeometryKey(r), string(b)).Error
	})
}

func (s *Store) StructuralGeometryCounts(traderIDs []string, tradeDate string) (map[string]int, error) {
	if s == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	out := map[string]int{}
	for _, id := range traderIDs {
		prefix := "structural_counts:" + id + ":" + tradeDate + ":"
		var rows []struct {
			Key   string
			Value int
		}
		if err := s.GormDB().Raw("SELECT key,CAST(value AS INTEGER) AS value FROM system_config WHERE substr(key,1,?)=?", len(prefix), prefix).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[strings.TrimPrefix(r.Key, prefix)] += r.Value
		}
	}
	return out, nil
}

func StructuralBufferSweepPoints() []float64 { return []float64{.25, 1.25, StructuralBufferMNQDefault} }

func (s *Store) StructuralGeometryFor(traderID, planID string, version int) ([]StructuralGeometryRecord, error) {
	if s == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	var rows []struct{ Value string }
	prefix := fmt.Sprintf("structural_geometry:%s:%s:v%d:", traderID, planID, version)
	if err := s.GormDB().Raw("SELECT value FROM system_config WHERE substr(key,1,?)=? ORDER BY key", len(prefix), prefix).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := []StructuralGeometryRecord{}
	for _, row := range rows {
		var r StructuralGeometryRecord
		if err := json.Unmarshal([]byte(row.Value), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
