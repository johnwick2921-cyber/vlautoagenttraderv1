package kernel

import (
	"strings"
	"testing"
)

// CLASS 39 (owner ruling 2026-09-01) — NORMALIZE, DON'T REJECT: on a
// non-sweep_reclaim condition, an authored arm that carries ANY legs array is
// collapsed to its single top-level arm (drop legs → re-validate → WARN), and
// only rejected — with the ORIGINAL reason — when the remaining single arm is
// itself invalid. sweep_reclaim is never normalized; one leg on sweep_reclaim
// stays a reject.
//
// Evidence: planner_rejected_prompts row 69 (2026-09-01 11:59:57 CT) embeds the
// attempt-1 output that was rejected with
//   "arm legs on breakdown_continue — arm_legs_sweep_reclaim_only"
// S1 carried ONE leg (rule=touch) whose top-level already mirrored it; dropping
// the array alone makes the arm valid. 35 of 121 validator rejects in the 72h to
// 2026-09-01 were this shape; 7 on attempt 3/3; two sessions fail-closed on it.
//
// This file uses only pre-class-39 API so E1 fails on an ASSERTION on the
// pre-fix tree (the :167 reject), not on a missing symbol.

// row69S1 is the exact arm from planner_rejected_prompts row 69, scenario S1.
func row69S1() *PlanDoc {
	return lawDoc("breakdown_continue", "1x5m_close", func(s *PlanScenario) {
		s.Direction = "short"
		s.Trigger = "5m close below 29130.00 with expanding displacement (post-lunch)"
		s.Invalid = "2x5m close back above 29168.00"
		s.TargetChain = []float64{29108.12, 29082.75, 29062.75, 29040.00}
		s.Confirm = &PlanConfirm{Rule: "1x5m_close", RefPrice: 29130.00, Side: "below"}
		s.Breakdown = &PlanBreakdownContinue{Level: 29130.00, LevelLabel: "SWG-L·5m", EntryMode: "pullback"}
		s.Arm = &PlanArmSpec{
			Enabled: true, Entry: 29130.00, Stop: 29168.00, Target: 29040.00, WaitConfirm: true,
			Legs: []PlanArmLeg{{Entry: 29130.00, Stop: 29168.00, Target: 29040.00, Size: 1, WaitConfirm: false, Rule: "touch"}},
		}
	})
}

// row69S2 / row85S1 are the two retained sweep_reclaim one-leg arms — the
// REVERSE shape the ruling says must keep rejecting (needs EXACTLY 2 legs).
func row69S2() *PlanDoc {
	return lawDoc("sweep_reclaim", "touch", func(s *PlanScenario) {
		s.Direction = "long"
		s.Trigger = "sweep below 29082.75 ONL then reclaim back above it"
		s.Invalid = "5m close below 29035.00"
		s.TargetChain = []float64{29130.00, 29179.00, 29209.25}
		s.Confirm = &PlanConfirm{Rule: "touch", RefPrice: 29082.75, Side: "below"}
		s.Confirm2 = &PlanConfirm{Rule: "1x5m_close", RefPrice: 29082.75, Side: "above"}
		s.Arm = &PlanArmSpec{
			Enabled: true, Entry: 29082.75, Stop: 29035.00, Target: 29179.00, WaitConfirm: true,
			Legs: []PlanArmLeg{{Entry: 29082.75, Stop: 29035.00, Target: 29179.00, Size: 1, WaitConfirm: false, Rule: "touch"}},
		}
	})
}

func row85S1() *PlanDoc {
	return lawDoc("sweep_reclaim", "touch", func(s *PlanScenario) {
		s.Direction = "long"
		s.Trigger = "sweep below 29048 then reclaim"
		s.Invalid = "5m close below 29024"
		s.TargetChain = []float64{29110}
		s.Confirm = &PlanConfirm{Rule: "touch", RefPrice: 29048, Side: "below"}
		s.Confirm2 = &PlanConfirm{Rule: "1m_mss", RefPrice: 29048, Side: "above"}
		s.Arm = &PlanArmSpec{
			Enabled: true, Entry: 29048, Stop: 29024, Target: 29110, WaitConfirm: true,
			Legs: []PlanArmLeg{{Entry: 29048, Stop: 29024, Target: 29110, Size: 1, WaitConfirm: false, Rule: "touch"}},
		}
	})
}

// TestClass39PinRow69S1 — E1. On the pre-fix tree this FAILS with the :167
// reject; after class 39 the legs are dropped and the single arm validates.
func TestClass39PinRow69S1(t *testing.T) {
	d := row69S1()
	err := ValidatePlanDocWithCaps(d, 8, 3)
	if err != nil {
		t.Fatalf("row 69 S1 (breakdown_continue, one leg rule=touch, top-level already mirrors it) must NORMALIZE and validate; got the reject: %v", err)
	}
	if n := len(d.Scenarios[0].Arm.Legs); n != 0 {
		t.Fatalf("legs must be DROPPED on a non-sweep condition, still %d present", n)
	}
	if a := d.Scenarios[0].Arm; !a.Enabled || a.Entry != 29130.00 || a.Stop != 29168.00 || a.Target != 29040.00 || !a.WaitConfirm {
		t.Fatalf("the single arm must be kept unchanged: %+v", a)
	}
}

// TestClass39ReversePin — E4. sweep_reclaim is NEVER normalized: one leg stays
// a reject (needs EXACTLY 2), and a legal two-leg split is untouched. These
// pass on the pre-fix tree too — they pin what must NOT change.
func TestClass39ReversePin(t *testing.T) {
	for name, d := range map[string]*PlanDoc{"row69 S2": row69S2(), "row85 S1": row85S1()} {
		err := ValidatePlanDocWithCaps(d, 8, 3)
		if err == nil || !strings.Contains(err.Error(), "needs EXACTLY 2 legs") {
			t.Fatalf("%s (sweep_reclaim, one leg) must STILL reject with 'needs EXACTLY 2 legs' — the reverse is never normalized; got %v", name, err)
		}
		if n := len(d.Scenarios[0].Arm.Legs); n != 1 {
			t.Fatalf("%s: legs must be left untouched on sweep_reclaim, got %d", name, n)
		}
	}
	// A legal split is unchanged: two legs stay two legs, no error.
	split := lawDoc("sweep_reclaim", "touch", func(s *PlanScenario) {
		s.Confirm2 = &PlanConfirm{Rule: "1m_mss", RefPrice: 29648.25, Side: "below"}
		s.Arm = &PlanArmSpec{
			Enabled: true, Entry: 29648.25, Stop: 29700.00, Target: 29441.00, WaitConfirm: true,
			Legs: []PlanArmLeg{
				{Entry: 29648.25, Stop: 29700.00, Target: 29441.00, Size: 1, WaitConfirm: false},
				{Entry: 29640.00, Stop: 29700.00, Target: 29441.00, Size: 1, WaitConfirm: true, Rule: "1m_mss"},
			},
		}
	})
	if err := ValidatePlanDocWithCaps(split, 8, 3); err != nil {
		t.Fatalf("a legal sweep_reclaim split must validate unchanged: %v", err)
	}
	if n := len(split.Scenarios[0].Arm.Legs); n != 2 {
		t.Fatalf("legal split must keep both legs, got %d", n)
	}
}
