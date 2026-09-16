package store

import (
	"encoding/json"
	"testing"
)

// Owner ruling 2026-08-31 — min_side_levels (the per-side count knob) is
// REMOVED from the system. Old stored config JSON that still carries the field
// must load HARMLESSLY: encoding/json ignores unknown fields on unmarshal, so
// a stored strategy/session override with "min_side_levels" round-trips into a
// config without the field and without error.
func TestDayPlanConfigIgnoresRemovedMinSideLevels(t *testing.T) {
	const oldJSON = `{
		"min_side_levels": 4,
		"min_scenario_quality": "B",
		"sessions": [
			{"session": "NY", "min_side_levels": 3, "max_trades": 7},
			{"session": "ASIA", "max_trades": 10}
		]
	}`
	var c DayPlanConfig
	if err := json.Unmarshal([]byte(oldJSON), &c); err != nil {
		t.Fatalf("old JSON with min_side_levels must load without error, got %v", err)
	}
	if got := c.MinScenarioQualityFor("NY"); got != "B" {
		t.Fatalf("sibling fields must survive, got %q", got)
	}
	ov := c.SessionOverride("NY")
	if ov == nil || ov.MaxTrades == nil || *ov.MaxTrades != 7 {
		t.Fatalf("NY override must survive: %+v", ov)
	}
	if ovASIA := c.SessionOverride("ASIA"); ovASIA == nil || ovASIA.MaxTrades == nil || *ovASIA.MaxTrades != 10 {
		t.Fatalf("ASIA override must survive: %+v", ovASIA)
	}
	// And the field must be GONE from the marshal output.
	b, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsMinSide(string(b)) {
		t.Fatalf("min_side_levels must not re-appear in marshaled output: %s", b)
	}
}

func containsMinSide(s string) bool {
	return len(s) >= 15 && findSub(s, "min_side_levels") >= 0
}

func findSub(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
