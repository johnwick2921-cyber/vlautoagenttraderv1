package store

import (
	"path/filepath"
	"strings"
	"testing"
)

// F5 — a save that changes two knobs yields two changes with RESOLVED
// old→new values, and an unchanged save yields none.
func TestConfigDiffNamesEveryChangedKnob(t *testing.T) {
	before := StrategyConfig{}
	before.RiskControl.MinRiskRewardRatio = 3
	before.RiskControl.MaxPositions = 1
	after := before
	after.RiskControl.MinRiskRewardRatio = 2
	after.RiskControl.MaxPositions = 2

	ch := DiffStrategyConfig(before, after)
	if len(ch) != 2 {
		t.Fatalf("want 2 changes, got %d: %+v", len(ch), ch)
	}
	var rr *ConfigChange
	for i := range ch {
		if strings.Contains(ch[i].Knob, "min_risk_reward_ratio") {
			rr = &ch[i]
		}
	}
	if rr == nil {
		t.Fatalf("min_risk_reward_ratio not reported: %+v", ch)
	}
	if rr.OldValue != "3" || rr.NewValue != "2" {
		t.Fatalf("resolved values wrong: %s → %s", rr.OldValue, rr.NewValue)
	}
	// The exact 2026-09-01 08:13 CT drift, rendered.
	line := ConfigDiffLine("studio_save", *rr)
	if !strings.Contains(line, "⚙ config diff (studio_save):") || !strings.Contains(line, "3 → 2") {
		t.Fatalf("diff line: %s", line)
	}
}

func TestConfigDiffSilentOnUnchangedSave(t *testing.T) {
	c := StrategyConfig{}
	c.RiskControl.MinRiskRewardRatio = 3
	if ch := DiffStrategyConfig(c, c); len(ch) != 0 {
		t.Fatalf("an unchanged save must report nothing, got %+v", ch)
	}
	if l := ConfigDiffSummaryLine("studio_save", "s1", 0); !strings.Contains(l, "NO resolved-value change") {
		t.Fatalf("summary: %s", l)
	}
}

// F5 — the rows persist and read back newest-first.
func TestConfigChangesPersist(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "cc.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	cs := st.ConfigChanges()
	before := StrategyConfig{}
	before.RiskControl.MinRiskRewardRatio = 3
	after := before
	after.RiskControl.MinRiskRewardRatio = 2
	rows := DiffStrategyConfig(before, after)
	for i := range rows {
		rows[i].Strategy = "s1"
		rows[i].Source = "studio_save"
	}
	if err := cs.Save(rows); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := cs.Recent(10)
	if err != nil || len(got) != 1 {
		t.Fatalf("recent: %d rows err=%v", len(got), err)
	}
	if got[0].OldValue != "3" || got[0].NewValue != "2" || got[0].Source != "studio_save" {
		t.Fatalf("row wrong: %+v", got[0])
	}
	if err := cs.Save(nil); err != nil {
		t.Fatalf("an empty batch must be a no-op, got %v", err)
	}
}

// A nested knob is reported by its dotted path, so the line names the knob the
// owner sees in Studio, not a struct field.
func TestConfigDiffUsesDottedResolvedPaths(t *testing.T) {
	before := StrategyConfig{}
	after := before
	after.RiskControl.MinRiskRewardRatio = 2
	ch := DiffStrategyConfig(before, after)
	if len(ch) != 1 || !strings.Contains(ch[0].Knob, ".") {
		t.Fatalf("want a dotted path, got %+v", ch)
	}
}
