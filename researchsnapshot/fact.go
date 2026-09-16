// Package researchsnapshot records observations without participating in a
// trading decision. Its SQLite archive is separate from the trading database.
package researchsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

const SchemaVersion = 1

var Objects = [...]string{"market", "candidate", "plan", "scenario", "exec"}

// The four clocks are independent nullable SQL columns. A missing source clock
// is never replaced by a read, publication, enqueue, or database-write clock.
type Clocks struct {
	ObservationMS *int64 `json:"observation_ms"`
	ReceiptMS     *int64 `json:"receipt_ms"`
	PublicationMS *int64 `json:"publication_ms"`
	PermissionMS  *int64 `json:"permission_ms"`
}

type Fact struct {
	Object     string            `json:"object"`
	SnapshotID *string           `json:"snapshot_id"`
	Event      string            `json:"event"`
	Clocks     Clocks            `json:"clocks"`
	Fields     map[string]any    `json:"fields"`
	Missing    map[string]string `json:"missing"`
}

// Every registered evidence field is emitted, including explicit JSON null.
// SQLite json_extract consequently returns SQL NULL for uncaptured values.
// Empty computed arrays and genuinely zero numbers remain [] and 0.
var fields = map[string][]string{
	"market":    {"root_symbol", "contract", "contract_basis", "feed", "source_timezone", "timeframe", "source_stamp_ms", "bar_open_ms", "bar_close_ms", "open", "high", "low", "close", "volume", "finalized", "forming", "correction", "previous_observation", "missing_intervals", "bid", "ask", "spread", "price_scale", "roll_information", "source_build_id"},
	"candidate": {"stable_id", "identity_basis", "root_symbol", "raw_origin", "family", "price", "lo", "hi", "formation_ms", "availability_ms", "prior_episodes", "zone_pattern", "timeframe", "freshness_at_read", "confluence_raw", "confluence_capped", "raw_score_components", "capped_score_components", "overrides", "final_score", "grade", "rank", "selection_outcome", "exclusion_reason", "role", "legacy_row_id"},
	"plan":      {"input_snapshot_id", "input_snapshot", "prompt", "system_prompt", "prompt_hash", "prompt_version", "model", "model_config", "config_version", "config", "attempt", "attempt_mode", "attempt_started_ms", "attempt_ended_ms", "duration_ms", "rejection_reason", "raw_output", "accepted_output", "normalization", "plan_id", "plan_version", "tokens_in", "tokens_out"},
	"scenario":  {"plan_id", "plan_version", "scenario_id", "ordered_predicates", "initial_risk", "risk_basis", "target_path", "predicate_timestamps", "invalidation", "expiry", "revalidation", "reason", "permission_status", "arm_id", "authored_geometry"},
	"exec":      {"root_symbol", "contract", "signal_id", "order_id", "parent_id", "order_type", "order_semantics", "side", "oco_id", "tif", "intended_entry", "composed_entry", "accepted_entry", "attainable_entry", "entry_basis", "exit_price", "exit_basis", "fills", "simulation_assumption", "costs", "size", "common_horizon", "ambiguity", "timeout", "broker_frame", "source_build_id", "reason", "pnl_corrected", "outcome_exclusion", "position_id", "plan_id", "plan_version", "scenario_id"},
}

func NewFact(object, event string, snapshot *string, clocks Clocks) Fact {
	f := Fact{Object: object, Event: event, SnapshotID: snapshot, Clocks: clocks,
		Fields: make(map[string]any), Missing: make(map[string]string)}
	for _, name := range fields[object] {
		f.Fields[name] = nil
		f.Missing[name] = "not captured by " + event
	}
	for name, value := range map[string]*int64{"observation_ms": clocks.ObservationMS, "receipt_ms": clocks.ReceiptMS, "publication_ms": clocks.PublicationMS, "permission_ms": clocks.PermissionMS} {
		if value == nil {
			f.Missing[name] = "source clock not captured by " + event
		}
	}
	if snapshot == nil {
		f.Missing["snapshot_id"] = "no input snapshot linkage supplied"
	}
	return f
}

func (f *Fact) Set(name string, value any) {
	if raw, ok := value.(json.RawMessage); ok && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		value = nil
	}
	if value != nil {
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			if v.IsNil() {
				value = nil
			}
		}
	}
	f.Fields[name] = value
	if value != nil {
		delete(f.Missing, name)
	} else if f.Missing[name] == "" {
		f.Missing[name] = "source supplied NULL"
	}
}

func (f *Fact) Unknown(name, reason string) {
	f.Fields[name] = nil
	f.Missing[name] = reason
}

func (f Fact) validate() error {
	if _, ok := fields[f.Object]; !ok {
		return fmt.Errorf("unknown research object %q", f.Object)
	}
	for _, name := range fields[f.Object] {
		v, ok := f.Fields[name]
		if !ok {
			return fmt.Errorf("research field %s is absent rather than NULL", name)
		}
		if v == nil && f.Missing[name] == "" {
			return fmt.Errorf("research field %s is NULL without a reason", name)
		}
	}
	_, err := json.Marshal(f)
	return err
}

func Value[T any](v T) *T { return &v }
