package store

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func TestScenarioDeathFirstWriterAndIdentity(t *testing.T) {
	st := newPlanTestStore(t)
	now, _ := time.Parse(time.RFC3339, "2026-09-08T10:00:00-05:00")
	r := ScenarioDeath{PlanID: "2026-09-08:NY", Version: 1, ScenarioID: "S1", Anchor: 29687.5, Price: 29700, Cause: "invalidated", Condition: "accepted above", Basis: "heuristic", ObservedAt: now}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.RecordScenarioDeath("t", r); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	r.Version = 2
	r.Anchor = 29753.25
	r.ObservedAt = now.Add(time.Hour)
	if wrote, err := st.RecordScenarioDeath("t", r); err != nil || !wrote {
		t.Fatalf("second version: %t %v", wrote, err)
	}
	for version, anchor := range []float64{29687.5, 29753.25} {
		got, err := st.ScenarioDeathFor("t", r.PlanID, version+1, "S1", anchor)
		if err != nil || got == nil || got.Anchor != anchor {
			t.Fatalf("version %d: %+v %v", version+1, got, err)
		}
	}
	if got, err := st.ScenarioDeathFor("t", r.PlanID, 2, "S1", 29687.5); err == nil || got != nil {
		t.Fatalf("overlay anchor mismatch must not borrow a time: %+v %v", got, err)
	}
	counts, err := st.PlanLivenessCounts()
	if err != nil || counts.DeathsRecorded != 2 {
		t.Fatalf("first writer counts: %+v %v", counts, err)
	}
}

func TestScenarioLivenessUnknownAndVersionIsolation(t *testing.T) {
	st := newPlanTestStore(t)
	now, _ := time.Parse(time.RFC3339, "2026-09-08T10:00:00-05:00")
	planID := "2026-09-08:NY"
	meta, _ := json.Marshal(map[string]any{"observed_at": now})
	if err := st.SetSystemConfig(ScenarioMetaKey("t", planID, 1), string(meta)); err != nil {
		t.Fatal(err)
	}
	if err := st.SetSystemConfig(ScenarioStatusKey("t", planID, 1), `{"S1":"invalidated","S2":"armed"}`); err != nil {
		t.Fatal(err)
	}
	got := st.ScenarioLivenessFor("t", planID, 1, []string{"S1", "S2"}, now)
	if got.Tradeable == nil || *got.Tradeable != 1 || got.Total != 2 {
		t.Fatalf("known count: %+v", got)
	}
	for _, v := range []ScenarioLiveness{
		st.ScenarioLivenessFor("t", planID, 2, []string{"S1", "S2"}, now),
		st.ScenarioLivenessFor("t", planID, 1, []string{"S1", "S2", "S3"}, now),
		st.ScenarioLivenessFor("t", planID, 1, []string{"S1", "S2"}, now.Add(ScenarioSnapshotMaxAge+time.Second)),
	} {
		if v.Tradeable != nil || v.Reason == "" {
			t.Fatalf("unknown became a number: %+v", v)
		}
	}
}

func TestLivenessEventsCountAttemptsNotClockTicks(t *testing.T) {
	st := newPlanTestStore(t)
	now, _ := time.Parse(time.RFC3339, "2026-09-08T10:00:00-05:00")
	for i := 0; i < 3; i++ {
		if _, err := st.RecordPlanLivenessEvent(LivenessBornDeadRefusal, "candidate", now, "refused"); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := st.PlanLivenessCounts()
	if err != nil || counts.BornDeadRefusals != 3 {
		t.Fatalf("three observations at one clock instant must count three: %+v %v", counts, err)
	}
}

// Telemetry must return an entropy failure to its warning caller, never panic.
func TestLivenessTelemetryEntropyFailureReturnsError(t *testing.T) {
	st := newPlanTestStore(t)
	uuid.SetRand(livenessBrokenEntropy{})
	defer uuid.SetRand(nil)
	now, _ := time.Parse(time.RFC3339, "2026-09-08T10:00:00-05:00")
	wrote, err := st.RecordPlanLivenessEvent(LivenessBornDeadRefusal, "attempt", now, "observed")
	if wrote || err == nil {
		t.Fatalf("telemetry entropy failure: wrote=%t err=%v", wrote, err)
	}
}

type livenessBrokenEntropy struct{}

func (livenessBrokenEntropy) Read([]byte) (int, error) {
	return 0, fmt.Errorf("injected entropy failure")
}
