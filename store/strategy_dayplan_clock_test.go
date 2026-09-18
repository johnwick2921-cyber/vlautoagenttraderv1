package store

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// W-KNOB-PRUNE (2026-09-18): last_entry_ct / eod_flat_ct were DELETED from
// DayPlanConfig (unreachable since the P2 session-scope redesign). Every live
// strategy row still carries them ("last_entry_ct":"13:00","eod_flat_ct":"14:45"
// on all nine day_plan rows, sqlite3 -readonly 2026-09-18). This is the proof
// that such a row still loads through a real DB round-trip, that the two tags
// are no longer schema fields, and that the per-session offsets — the live
// clock — are what survive.
func TestDayPlanLegacyClockFieldsIgnoredOnLoad(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	raw := `{"day_plan":{"plan_enabled":true,"last_entry_ct":"13:15","eod_flat_ct":"14:30","sessions":[{"session":"NY","last_entry_offset_min":20,"eod_flat_offset_min":5}]}}`
	if err := st.Strategy().Create(&Strategy{ID: "s1", UserID: "u1", Name: "legacy-clock", Config: raw}); err != nil {
		t.Fatalf("create strategy: %v", err)
	}
	got, err := st.Strategy().Get("u1", "s1")
	if err != nil || got == nil {
		t.Fatalf("get strategy: %v", err)
	}
	var back StrategyConfig
	if err := json.Unmarshal([]byte(got.Config), &back); err != nil {
		t.Fatalf("unmarshal reloaded config (legacy clock fields must be ignored, not fatal): %v", err)
	}
	if back.DayPlan == nil || !back.DayPlan.PlanEnabled {
		t.Fatalf("day_plan lost on reload: %+v", back.DayPlan)
	}
	if back.DayPlan.LastEntryOffsetFor("NY") != 20 || back.DayPlan.EODFlatOffsetFor("NY") != 5 {
		t.Fatalf("the live per-session clock must survive: last=%d flat=%d", back.DayPlan.LastEntryOffsetFor("NY"), back.DayPlan.EODFlatOffsetFor("NY"))
	}
	// The tags are gone from the schema — reflection, not a grep.
	for _, leaf := range []string{"last_entry_ct", "eod_flat_ct"} {
		if owners := knobFieldOwners(leaf); len(owners) != 0 {
			t.Fatalf("%s still carried by %v — W-KNOB-PRUNE deleted it", leaf, owners)
		}
		for _, p := range EnumerateSchemaKnobs() {
			if strings.HasSuffix(p, "."+leaf) {
				t.Fatalf("%s still enumerated as %s", leaf, p)
			}
		}
	}
	// Re-marshal never re-emits them.
	out, _ := json.Marshal(back.DayPlan)
	if strings.Contains(string(out), "last_entry_ct") || strings.Contains(string(out), "eod_flat_ct") {
		t.Fatalf("deleted fields re-emitted: %s", out)
	}
	_ = reflect.TypeOf(DayPlanConfig{})
}
