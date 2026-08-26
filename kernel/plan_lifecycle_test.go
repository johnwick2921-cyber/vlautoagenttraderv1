package kernel

import "testing"

func TestActivePlanLevels(t *testing.T) {
	levels := []PlanLevel{{Price: 15600}, {Price: 15450}, {Price: 16000}}
	// price 15600, dATR 120, k 1.5 → band 180 → [15420, 15780].
	got := ActivePlanLevels(levels, 15600, 120, 1.5)
	if len(got) != 2 {
		t.Fatalf("want 2 active levels (15600,15450), got %d: %v", len(got), got)
	}
	for _, l := range got {
		if l.Price == 16000 {
			t.Fatalf("16000 is outside the window, must be hidden")
		}
	}
	// dATR<=0 → cannot scale → all levels returned.
	if len(ActivePlanLevels(levels, 15600, 0, 1.5)) != 3 {
		t.Fatalf("dATR<=0 should return all levels")
	}
}

func TestPlanIsDead(t *testing.T) {
	now := nowAfter(series([][4]float64{{0, 0, 0, 0}}))

	// Level 100 accepted through (2 closes above) → still-valid false → dead.
	deadBars := series([][4]float64{{99, 99, 98, 99}, {100, 102, 100, 101}, {101, 103, 101, 102}})
	deadDoc := PlanDoc{Levels: []PlanLevel{{Price: 100}}}
	if !PlanIsDead(deadDoc, deadBars, "2x5m", nowAfter(deadBars)) {
		t.Fatalf("plan whose only level is consumed must be dead")
	}

	// A level far above price (never touched) stays valid → not dead.
	liveDoc := PlanDoc{Levels: []PlanLevel{{Price: 200}}}
	if PlanIsDead(liveDoc, deadBars, "2x5m", nowAfter(deadBars)) {
		t.Fatalf("plan with a live level must NOT be dead")
	}

	// No levels → never dead this way.
	if PlanIsDead(PlanDoc{}, deadBars, "2x5m", now) {
		t.Fatalf("plan with no levels is not dead")
	}
}
