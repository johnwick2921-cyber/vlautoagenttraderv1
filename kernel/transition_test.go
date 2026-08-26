package kernel

import (
	"strings"
	"testing"
)

func TestExecutorPlanDeadVerdict(t *testing.T) {
	reason := "active plan is MACHINE-DEAD (price crossed X) — entries refused"
	for _, a := range []string{"open_long", "open_short"} {
		if blocked, msg := ExecutorPlanDeadVerdict(a, reason); !blocked || msg == "" {
			t.Fatalf("entries (%s) must be blocked with a refusal, got blocked=%v msg=%q", a, blocked, msg)
		}
	}
	for _, a := range []string{"close_long", "close_short", "hold", "wait", ""} {
		if blocked, _ := ExecutorPlanDeadVerdict(a, reason); blocked {
			t.Fatalf("management action %q must pass the dead-plan gate", a)
		}
	}
}

// R4 (2026-08-25) — the min_scenario_quality knob's verdict: fail-open on
// unknown citations, block only sub-floor scenario citations.
func TestMinScenarioQualityVerdict(t *testing.T) {
	qualities := map[string]string{"S1": "A", "S2": "B", "S3": "C"}
	if blocked, _ := MinScenarioQualityVerdict("open_long", "S2", "A", qualities); !blocked {
		t.Fatal("S2 (B) must block under floor A")
	}
	if blocked, _ := MinScenarioQualityVerdict("open_long", "S1", "A", qualities); blocked {
		t.Fatal("S1 (A) must pass under floor A")
	}
	if blocked, _ := MinScenarioQualityVerdict("open_long", "S3", "B", qualities); !blocked {
		t.Fatal("S3 (C) must block under floor B")
	}
	if blocked, _ := MinScenarioQualityVerdict("open_short", "S3", "C", qualities); blocked {
		t.Fatal("floor C must never block")
	}
	if blocked, _ := MinScenarioQualityVerdict("open_long", "S9", "A", qualities); blocked {
		t.Fatal("unknown scenario must fail open")
	}
	if blocked, _ := MinScenarioQualityVerdict("open_long", "off-plan", "A", qualities); blocked {
		t.Fatal("off-plan citation must fail open")
	}
	if blocked, _ := MinScenarioQualityVerdict("close_long", "S3", "A", qualities); blocked {
		t.Fatal("management actions are never blocked")
	}
	if blocked, _ := MinScenarioQualityVerdict("open_long", "S2", "A", nil); blocked {
		t.Fatal("nil quality map → dormant")
	}
}

func TestTransitionStanddownVerdict(t *testing.T) {
	// Active + same direction → refused with the full message.
	blocked, msg := TransitionStanddownVerdict("open_short", true, "short", "CHoCH-up 15m @29470.25 10:45")
	if !blocked {
		t.Fatalf("plan-direction entry during TRANSITION must be refused")
	}
	if !strings.HasPrefix(msg, "transition_standdown: short paused — unconfirmed CHoCH-up 15m @29470.25 10:45") {
		t.Fatalf("bad refusal message: %q", msg)
	}
	// Counter-direction entry is never paused by this (the flip owns it).
	if blocked, _ := TransitionStanddownVerdict("open_long", true, "short", "CHoCH-up 15m @29470.25 10:45"); blocked {
		t.Fatalf("counter-direction entry must NOT be paused by the stand-down")
	}
	// Inactive → pass; non-open actions → pass.
	if blocked, _ := TransitionStanddownVerdict("open_short", false, "short", "x"); blocked {
		t.Fatalf("inactive stand-down must pass")
	}
	if blocked, _ := TransitionStanddownVerdict("hold", true, "short", "x"); blocked {
		t.Fatalf("hold is never paused")
	}
}

func TestTransitionMaxMinDefault(t *testing.T) {
	if TransitionMaxMin() != DefaultTransitionMaxMin {
		t.Fatalf("default transition cap must be %d", DefaultTransitionMaxMin)
	}
}
