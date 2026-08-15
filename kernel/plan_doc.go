package kernel

import (
	"encoding/json"
	"fmt"
	"strings"
)

// P3.3 — the day-plan document (the schema-strict JSON the planner AI emits).
// The card renders it; the executor cites its scenarios. Reasoning-fields-FIRST
// (reasoning before the answer fields) per the contract.

// PlanBias is the plan's directional bias + explicit flip condition.
type PlanBias struct {
	Direction     string `json:"direction"`      // long | short | neutral
	Conviction    string `json:"conviction"`     // high | medium | low
	FlipCondition string `json:"flip_condition"` // e.g. "flips short on 2x5m < 30148"
}

// PlanLevel is one graded reference level with an instruction verb.
type PlanLevel struct {
	Price       float64 `json:"price"`
	Label       string  `json:"label"`       // provenance chip: PDH, ONH, nPOC·Tue, RN, EQH…
	Grade       string  `json:"grade"`       // A | B | C
	Instruction string  `json:"instruction"` // instruction verb, e.g. "fade", "reclaim-long"
}

// PlanScenario is one if/then play in the formal grammar.
type PlanScenario struct {
	ID          string    `json:"id"`           // S1, S2, S3
	Trigger     string    `json:"trigger"`      // the setup description
	Condition   string    `json:"condition"`    // reclaim|hold|sweep_reclaim|reject|acceptance|breakout_retest
	Direction   string    `json:"direction"`    // long | short
	TargetChain []float64 `json:"target_chain"` // ordered targets
	Invalid     string    `json:"invalid"`      // invalidation
	Quality     string    `json:"quality"`      // A+ | A | B
}

// PlanDoc is the full plan (stored as the plans.doc JSON).
type PlanDoc struct {
	Reasoning      string         `json:"reasoning"` // reasoning FIRST
	Bias           PlanBias       `json:"bias"`
	Levels         []PlanLevel    `json:"levels"`
	Scenarios      []PlanScenario `json:"scenarios"`
	NoTrade        []string       `json:"no_trade"`
	DeathCondition string         `json:"death_condition"`
	DayType        string         `json:"day_type,omitempty"`
}

var (
	biasDirections    = map[string]bool{"long": true, "short": true, "neutral": true}
	biasConvictions   = map[string]bool{"high": true, "medium": true, "low": true}
	levelGrades       = map[string]bool{"A": true, "B": true, "C": true}
	scenarioConds     = map[string]bool{"reclaim": true, "hold": true, "sweep_reclaim": true, "reject": true, "acceptance": true, "breakout_retest": true}
	scenarioDirs      = map[string]bool{"long": true, "short": true}
	scenarioQualities = map[string]bool{"A+": true, "A": true, "B": true}
)

const (
	planMaxLevels    = 8
	planMaxScenarios = 3
)

// ParsePlanDoc extracts the JSON object from raw model output (tolerating
// surrounding prose / code fences), unmarshals it, and validates it against the
// schema. Any failure → error, which the planner treats as a retryable/fail-closed
// event.
func ParsePlanDoc(raw string) (*PlanDoc, error) {
	js := extractJSONObject(raw)
	if js == "" {
		return nil, fmt.Errorf("no JSON object found in planner output")
	}
	var doc PlanDoc
	if err := json.Unmarshal([]byte(js), &doc); err != nil {
		return nil, fmt.Errorf("plan JSON unmarshal: %w", err)
	}
	if err := ValidatePlanDoc(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ValidatePlanDoc enforces the schema-strict rules: required fields, enum values,
// and counts (levels ≤8, scenarios 1–3).
func ValidatePlanDoc(d *PlanDoc) error {
	if d == nil {
		return fmt.Errorf("nil plan")
	}
	if strings.TrimSpace(d.Reasoning) == "" {
		return fmt.Errorf("reasoning is required (reasoning-first)")
	}
	if !biasDirections[d.Bias.Direction] {
		return fmt.Errorf("bias.direction %q invalid (long|short|neutral)", d.Bias.Direction)
	}
	if d.Bias.Conviction != "" && !biasConvictions[d.Bias.Conviction] {
		return fmt.Errorf("bias.conviction %q invalid (high|medium|low)", d.Bias.Conviction)
	}
	if strings.TrimSpace(d.DeathCondition) == "" {
		return fmt.Errorf("death_condition is required")
	}
	if len(d.Levels) > planMaxLevels {
		return fmt.Errorf("too many levels: %d (max %d)", len(d.Levels), planMaxLevels)
	}
	for i, l := range d.Levels {
		if !levelGrades[l.Grade] {
			return fmt.Errorf("level[%d].grade %q invalid (A|B|C)", i, l.Grade)
		}
	}
	if len(d.Scenarios) < 1 || len(d.Scenarios) > planMaxScenarios {
		return fmt.Errorf("scenarios count %d invalid (1..%d)", len(d.Scenarios), planMaxScenarios)
	}
	for i, s := range d.Scenarios {
		if strings.TrimSpace(s.ID) == "" {
			return fmt.Errorf("scenario[%d].id is required", i)
		}
		if !scenarioConds[s.Condition] {
			return fmt.Errorf("scenario[%d].condition %q invalid", i, s.Condition)
		}
		if !scenarioDirs[s.Direction] {
			return fmt.Errorf("scenario[%d].direction %q invalid (long|short)", i, s.Direction)
		}
		if !scenarioQualities[s.Quality] {
			return fmt.Errorf("scenario[%d].quality %q invalid (A+|A|B)", i, s.Quality)
		}
	}
	return nil
}

// extractJSONObject returns the substring from the first '{' to the matching
// last '}' (brace-balanced), or "" if none.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// NoTradePlanDoc builds the FAIL-CLOSED no-trade plan: a valid plan with a
// neutral bias and a single "no-trade" scenario, so a read failure still writes a
// concrete NO-TRADE plan row (never a stale plan, never nothing).
func NoTradePlanDoc(reason string) *PlanDoc {
	return &PlanDoc{
		Reasoning:      "FAIL-CLOSED: " + reason + " — no valid plan produced; sitting out this session.",
		Bias:           PlanBias{Direction: "neutral", Conviction: "low", FlipCondition: "n/a"},
		Levels:         nil,
		Scenarios:      []PlanScenario{{ID: "S0", Trigger: "none", Condition: "hold", Direction: "long", Invalid: "n/a", Quality: "B"}},
		NoTrade:        []string{"ENTIRE SESSION — planner fail-closed"},
		DeathCondition: "already dead (fail-closed)",
		DayType:        "no-trade",
	}
}
