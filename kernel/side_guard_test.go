package kernel

import (
	"strings"
	"testing"
)

// Side guards after owner ruling 2026-08-31 — the per-side COUNT concept is
// deleted (no quota, no WARN, no thin_side note). Only survivors: the two
// data-earned hard fails — 0-levels-on-a-side and empty machine map.

func sideDoc(below, above int, price float64) *PlanDoc {
	d := &PlanDoc{
		Reasoning:      "side guard test",
		Bias:           PlanBias{Direction: "neutral", Conviction: "low", FlipCondition: "none"},
		DeathCondition: "all levels consumed",
		Scenarios: []PlanScenario{
			{ID: "S1", Trigger: "hold", Condition: "hold", Direction: "long",
				TargetChain: []float64{price + 20}, Invalid: "2x5m below", Quality: "B"},
		},
	}
	for i := 0; i < below; i++ {
		d.Levels = append(d.Levels, PlanLevel{Price: price - float64(10*(i+1)), Label: "RN", Grade: "B", Instruction: "reclaim"})
	}
	for i := 0; i < above; i++ {
		d.Levels = append(d.Levels, PlanLevel{Price: price + float64(10*(i+1)), Label: "RN", Grade: "B", Instruction: "fade"})
	}
	return d
}

func sideMachineMap(prices ...float64) map[float64]string {
	m := make(map[float64]string, len(prices))
	for _, p := range prices {
		m[p] = "RN"
	}
	return m
}

// TestSideGuardOneBelowRichMapWritesClean — owner ruling 2026-08-31: a plan
// with 1 below against a rich machine map WRITES cleanly. No WARN, no note,
// no error — the count concept no longer exists.
func TestSideGuardOneBelowRichMapWritesClean(t *testing.T) {
	d := sideDoc(1, 3, 100)
	err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		sideMachineMap(99, 98, 97, 101, 102, 103), 8, 3)
	if err != nil {
		t.Fatalf("1-below + rich map must write clean (count is deleted), got err=%v", err)
	}
}

// TestSideGuardOneAboveRichMapWritesClean — the mirrored side.
func TestSideGuardOneAboveRichMapWritesClean(t *testing.T) {
	d := sideDoc(3, 1, 100)
	err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		sideMachineMap(99, 98, 97, 101, 102, 103), 8, 3)
	if err != nil {
		t.Fatalf("1-above + rich map must write clean (count is deleted), got err=%v", err)
	}
}

// TestSideGuardZeroBelowFails — the 2026-08-18 one-sided-map pathology stays a
// hard fail, with the new no-quota message.
func TestSideGuardZeroBelowFails(t *testing.T) {
	d := sideDoc(0, 4, 100)
	err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		sideMachineMap(101, 102, 103, 104, 99), 8, 3)
	if err == nil || !strings.Contains(err.Error(), "0 levels below") ||
		!strings.Contains(err.Error(), "a plan must map both directions") {
		t.Fatalf("0-below must hard fail with the new message, got err=%v", err)
	}
	if strings.Contains(err.Error(), "≥") {
		t.Fatalf("0-below message must carry NO quota language, got %q", err.Error())
	}
}

// TestSideGuardZeroAboveFails — mirrored.
func TestSideGuardZeroAboveFails(t *testing.T) {
	d := sideDoc(4, 0, 100)
	err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		sideMachineMap(99, 98, 97, 96, 101), 8, 3)
	if err == nil || !strings.Contains(err.Error(), "0 levels above") ||
		!strings.Contains(err.Error(), "a plan must map both directions") {
		t.Fatalf("0-above must hard fail with the new message, got err=%v", err)
	}
	if strings.Contains(err.Error(), "≥") {
		t.Fatalf("0-above message must carry NO quota language, got %q", err.Error())
	}
}

// TestSideGuardEmptyMachineMapFails — unchanged hard fail.
func TestSideGuardEmptyMachineMapFails(t *testing.T) {
	d := sideDoc(3, 3, 100)
	err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		map[float64]string{}, 8, 3)
	if err == nil || !strings.Contains(err.Error(), "EMPTY") {
		t.Fatalf("empty machine map must fail-closed, got err=%v", err)
	}
}

// TestSideGuardNilMachineZeroSideFails — legacy callers (nil machine universe)
// keep the 0-side pathology guard; counts never fail.
func TestSideGuardNilMachineZeroSideFails(t *testing.T) {
	d := sideDoc(0, 3, 100)
	err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		nil, 8, 3)
	if err == nil || !strings.Contains(err.Error(), "0 levels below") {
		t.Fatalf("nil-map 0-below must fail, got err=%v", err)
	}
	// 1/1 with a nil map writes clean — no count rule anywhere.
	d = sideDoc(1, 1, 100)
	if err := ValidatePlanDocWithFactsMachine(d, PlanFacts{Price: 100, DATR: 50},
		nil, 8, 3); err != nil {
		t.Fatalf("nil-map 1/1 must write clean (no count rule), got err=%v", err)
	}
}
