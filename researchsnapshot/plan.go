package researchsnapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// PlanTrace belongs to one authoring invocation, never shared mutable trader
// state. Queued jobs receive value copies of strings and timestamps.
type PlanTrace struct {
	Clock                                          func() time.Time
	SnapshotID, Model, ConfigVersion, SystemPrompt string
	Attempt                                        int
	Mode, Prompt, Raw                              string
	Started, Ended                                 time.Time
	pending                                        bool
}

func (p *PlanTrace) Begin(attempt int, mode, prompt string) {
	p.BeginAt(attempt, mode, prompt, p.clock())
}
func (p *PlanTrace) BeginAt(attempt int, mode, prompt string, now time.Time) {
	defer Contain(" recording")
	if p == nil {
		return
	}
	p.Attempt = attempt
	p.Mode = mode
	p.Prompt = prompt
	p.Raw = ""
	p.Started = now
	p.Ended = time.Time{}
	p.pending = true
	p.emit("attempt_started", nil, nil, nil, now)
}
func (p *PlanTrace) Reply(raw string, err error) {
	p.ReplyAt(raw, err, p.clock())
}
func (p *PlanTrace) ReplyAt(raw string, err error, now time.Time) {
	defer Contain(" recording")
	if p == nil {
		return
	}
	p.Raw = raw
	p.Ended = now
	p.emit("provider_returned", err, nil, nil, now)
}
func (p *PlanTrace) Finish(err error) {
	p.FinishAt(err, p.clock())
}
func (p *PlanTrace) FinishAt(err error, now time.Time) {
	defer Contain(" recording")
	if p == nil || !p.pending {
		return
	}
	p.emit("attempt_verdict", err, nil, nil, now)
	p.pending = false
}
func (p *PlanTrace) Published(planID string, version int, output string) {
	p.PublishedAt(planID, version, output, p.clock())
}
func (p *PlanTrace) PublishedAt(planID string, version int, output string, now time.Time) {
	defer Contain(" recording")
	if p == nil {
		return
	}
	p.emit("published", nil, &output, &version, now)
	// The identity is its own append-only event so a plan append never appears
	// successful before the store returns a version.
	id := p.SnapshotID
	Record("plan:identity", func() []Fact {
		f := NewFact("plan", "publication_identity", Value(id), Clocks{ReceiptMS: Value(now.UnixMilli()), PublicationMS: Value(now.UnixMilli())})
		f.Set("plan_id", planID)
		f.Set("plan_version", version)
		f.Set("input_snapshot_id", id)
		out := []Fact{f}
		var doc struct {
			Scenarios []map[string]json.RawMessage `json:"scenarios"`
		}
		if e := json.Unmarshal([]byte(output), &doc); e != nil {
			panic("research accepted document decode")
		}
		for _, sc := range doc.Scenarios {
			sf := NewFact("scenario", "published", Value(id), Clocks{ReceiptMS: Value(now.UnixMilli()), PublicationMS: Value(now.UnixMilli())})
			sf.Set("plan_id", planID)
			sf.Set("plan_version", version)
			for dest, src := range map[string]string{"scenario_id": "id", "target_path": "target_chain", "invalidation": "invalid", "authored_geometry": "arm"} {
				if v, ok := sc[src]; ok {
					sf.Set(dest, v)
				}
			}
			predicates := []json.RawMessage{}
			for _, key := range []string{"confirm", "confirm2"} {
				if v, ok := sc[key]; ok {
					predicates = append(predicates, v)
				}
			}
			if len(predicates) > 0 {
				sf.Set("ordered_predicates", predicates)
			}
			sf.Unknown("initial_risk", "authored entry/stop captured; actual composed risk is a later placement fact")
			sf.Unknown("permission_status", "published scenario; no order authorization verdict implied")
			out = append(out, sf)
		}
		return out
	})
}
func (p *PlanTrace) emit(event string, err error, output *string, version *int, now time.Time) {
	defer Contain(" recording")
	copy := *p
	var reason *string
	if err != nil {
		reason = Value(err.Error())
	}
	Record("plan:"+event, func() []Fact {
		clocks := Clocks{ReceiptMS: Value(now.UnixMilli())}
		if !copy.Ended.IsZero() {
			clocks.ObservationMS = Value(copy.Ended.UnixMilli())
		} else if !copy.Started.IsZero() {
			clocks.ObservationMS = Value(copy.Started.UnixMilli())
		}
		if event == "published" {
			clocks.PublicationMS = Value(now.UnixMilli())
		}
		f := NewFact("plan", event, Value(copy.SnapshotID), clocks)
		f.Set("input_snapshot_id", copy.SnapshotID)
		f.Set("model", copy.Model)
		f.Set("config_version", copy.ConfigVersion)
		f.Set("system_prompt", copy.SystemPrompt)
		f.Set("attempt", copy.Attempt)
		f.Set("attempt_mode", copy.Mode)
		f.Set("prompt", copy.Prompt)
		hash := sha256.Sum256([]byte(copy.Prompt))
		f.Set("prompt_hash", hex.EncodeToString(hash[:]))
		if !copy.Started.IsZero() {
			f.Set("attempt_started_ms", copy.Started.UnixMilli())
		}
		if !copy.Ended.IsZero() {
			f.Set("attempt_ended_ms", copy.Ended.UnixMilli())
			f.Set("duration_ms", float64(copy.Ended.Sub(copy.Started))/float64(time.Millisecond))
			f.Set("raw_output", copy.Raw)
		}
		if reason != nil {
			f.Set("rejection_reason", *reason)
		} else {
			f.Unknown("rejection_reason", "no rejection on this event")
		}
		if output != nil {
			f.Set("accepted_output", json.RawMessage(*output))
			f.Set("normalization", map[string]any{"basis": "exact raw response and final stored document; no replay of normalization", "before": copy.Raw, "after": json.RawMessage(*output)})
		}
		f.Set("plan_version", version)
		return []Fact{f}
	})
}

// Boundary clock accessor. Tests replace this only on their local trace;
// authoring's independent market/permission clock is never consumed here.
func (p *PlanTrace) clock() time.Time {
	if p != nil && p.Clock != nil {
		return p.Clock()
	}
	return time.Now()
}

func (p *PlanTrace) RequestConfig(mode, effort string, tokenCap int) {
	p.RequestConfigAt(mode, effort, tokenCap, p.clock())
}
func (p *PlanTrace) RequestConfigAt(mode, effort string, tokenCap int, now time.Time) {
	defer Contain("model configuration")
	if p == nil {
		return
	}
	id, attempt := p.SnapshotID, p.Attempt
	Record("plan:request_config", func() []Fact {
		f := NewFact("plan", "request_config", Value(id), Clocks{ObservationMS: Value(now.UnixMilli()), ReceiptMS: Value(now.UnixMilli())})
		f.Set("input_snapshot_id", id)
		f.Set("attempt", attempt)
		f.Set("model_config", map[string]any{"thinking_mode": mode, "reasoning_effort": effort, "max_tokens": tokenCap})
		return []Fact{f}
	})
}
