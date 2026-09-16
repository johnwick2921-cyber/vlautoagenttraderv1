package store

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// ENTRY-MECHANICS ADDENDUM (2026-08-30) — the acceptance-rule migration fixture.
// The knob census found day_plan.acceptance_rule="2x5m" (strategy-level) and
// per-session acceptance="2x5m" on NY/ASIA/LONDON. Under the new per-condition
// entry law that string contradicts the validator (reject loops); the boot
// repair rewrites both spots 2x5m → 5m_close, idempotently.

func acceptanceMigrationConfig() string {
	raw := map[string]any{
		"day_plan": map[string]any{
			"acceptance_rule": "2x5m",
			"sessions": []any{
				map[string]any{"session": "NY", "acceptance_rule": "2x5m"},
				map[string]any{"session": "ASIA", "acceptance_rule": "2x5m"},
				map[string]any{"session": "LONDON", "acceptance_rule": "2x5m"},
			},
		},
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func TestRepairAcceptanceRuleMigration(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.Strategy().Create(&Strategy{ID: "s1", UserID: "u1", Name: "mig", Config: acceptanceMigrationConfig()}); err != nil {
		t.Fatalf("create: %v", err)
	}

	base, sess, err := st.Strategy().RepairAcceptanceRuleMigration()
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if base != 1 || sess != 3 {
		t.Fatalf("want 1 base + 3 session migrations, got %d + %d", base, sess)
	}

	got, err := st.Strategy().Get("u1", "s1")
	if err != nil || got == nil {
		t.Fatalf("get: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal([]byte(got.Config), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dp := back["day_plan"].(map[string]any)
	if dp["acceptance_rule"] != "5m_close" {
		t.Fatalf("strategy-level acceptance_rule = %v, want 5m_close", dp["acceptance_rule"])
	}
	for _, sv := range dp["sessions"].([]any) {
		if sv.(map[string]any)["acceptance_rule"] != "5m_close" {
			t.Fatalf("session override not migrated: %v", sv)
		}
	}

	// Idempotent: a second run migrates nothing.
	base2, sess2, err := st.Strategy().RepairAcceptanceRuleMigration()
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if base2 != 0 || sess2 != 0 {
		t.Fatalf("second run must be a no-op, got %d + %d", base2, sess2)
	}
}

// TestAcceptanceRuleForSelfHeals — the resolver maps any stored OLD-vocabulary
// string to the one-close rule at read, so a straggler row can never steer the
// prompt back to the reserved double close.
func TestAcceptanceRuleForSelfHeals(t *testing.T) {
	for _, in := range []string{"2x5m", "2x5m_close", "15m-close", "15m_close", "15m", "15mclose"} {
		c := &DayPlanConfig{AcceptanceRule: in}
		if got := c.AcceptanceRuleFor("NY"); got != "5m_close" {
			t.Errorf("AcceptanceRuleFor(%q) = %q, want 5m_close", in, got)
		}
	}
	c := &DayPlanConfig{AcceptanceRule: "5m_close", Sessions: []DayPlanSessionOverride{
		{Session: "NY", AcceptanceRule: strPtr("2x5m")},
	}}
	if got := c.AcceptanceRuleFor("NY"); got != "5m_close" {
		t.Errorf("session override self-heal = %q, want 5m_close", got)
	}
	// The canonical value passes through untouched.
	c2 := &DayPlanConfig{AcceptanceRule: "5m_close"}
	if got := c2.AcceptanceRuleFor("NY"); got != "5m_close" {
		t.Errorf("canonical = %q", got)
	}
	// Empty config resolves to the new shipped default (never 2x5m).
	var nilCfg *DayPlanConfig
	if got := nilCfg.AcceptanceRuleFor("NY"); got != "5m_close" {
		t.Errorf("default = %q, want 5m_close", got)
	}
	if !strings.Contains(DefaultAcceptanceRule, "5m") {
		t.Errorf("DefaultAcceptanceRule still %q", DefaultAcceptanceRule)
	}
}

func strPtr(s string) *string { return &s }

// TestArmedOrderSplitLegsPersistUnderUniqueIndex — E4: two leg rows share
// (plan_id, scenario); the migrated 3-column unique index must accept both.
func TestArmedOrderSplitLegsPersistUnderUniqueIndex(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "legs.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ao := st.ArmedOrders()
	for _, leg := range []ArmedOrderDB{
		{TraderID: "t", PlanID: "p:NY", Version: 1, Session: "NY", Scenario: "S1", Side: "short",
			EntryPx: 100, StopPx: 110, TargetPx: 80, State: "armed", LegIndex: 0, LegCount: 2},
		{TraderID: "t", PlanID: "p:NY", Version: 1, Session: "NY", Scenario: "S1", Side: "short",
			EntryPx: 98, StopPx: 106, TargetPx: 82, State: "armed", LegIndex: 1, LegCount: 2},
	} {
		if err := ao.UpsertArm(&leg); err != nil {
			t.Fatalf("leg %d upsert failed under the split index: %v", leg.LegIndex, err)
		}
	}
	rows, err := ao.ListNonTerminal("t")
	if err != nil || len(rows) != 2 {
		t.Fatalf("split rows = %d err=%v, want 2", len(rows), err)
	}
}
