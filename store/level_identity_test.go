package store

import (
	"nofx/levelidentity"
	"strings"
	"testing"
	"time"
)

func TestIdentityBackfillRequiresEveryInput(t *testing.T) {
	now := time.Date(2026, 9, 10, 23, 0, 0, 0, time.UTC)
	p := 20000.0
	closeMs := now.Add(-time.Hour).UnixMilli()
	in := levelidentity.Inputs{Symbol: "MNQ", Kind: "PDL", Lo: &p, Hi: &p, OriginDate: "2026-09-09", TF: "1m", FormedCloseMs: &closeMs}
	row := TouchOutcomeRow{CreatedAt: now}
	if id, status := ClassifyIdentityBackfill(row, in); id == nil || status != "recomputed" {
		t.Fatalf("full inputs: %v %s", id, status)
	}
	in.TF = ""
	in.Hi = nil
	if id, status := ClassifyIdentityBackfill(row, in); id != nil || status != "unrecomputable:hi,tf" {
		t.Fatalf("partial hash: %v %s", id, status)
	}
	row.CreatedAt = now.Add(-24 * time.Hour)
	if id, status := ClassifyIdentityBackfill(row, levelidentity.Inputs{}); id != nil || status != "untouched" {
		t.Fatalf("legacy %v %s", id, status)
	}
}
func TestIdentityCountersAreVersionAndTraderScoped(t *testing.T) {
	s := newPlanTestStore(t)
	now := time.Date(2026, 9, 10, 23, 0, 0, 0, time.UTC)
	for _, trader := range []string{"A", "B"} {
		for v := 1; v <= 2; v++ {
			for i := 0; i < 2; i++ {
				_, err := s.RecordLevelIdentityEvent(trader, LevelIdentityEvent{PlanID: "P", Version: v, ScenarioID: "S1", Kind: "unnamed", At: now})
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	c, err := s.LevelIdentityCounts("A")
	if err != nil || c.Unnamed != 2 {
		t.Fatalf("counts %+v %v", c, err)
	}
	if _, err = s.RecordLevelIdentityEvent("A", LevelIdentityEvent{PlanID: "P", Version: 1, ScenarioID: "S1", Kind: "refuse", At: now}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatal("refusal entered WARN contract")
	}
}
