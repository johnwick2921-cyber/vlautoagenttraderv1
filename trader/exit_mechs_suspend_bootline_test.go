package trader

import (
	"strings"
	"testing"

	"nofx/store"
)

// D102-1 (2026-09-16) — the exit-posture boot line must read the SAME sources
// the exit mechanics honour, and say where each value came from. These tests
// call the production renderers directly (canon 53: call sites, not re-built
// inputs); the env reader is the production exitMechsSuspended() via t.Setenv.

func ds102BoolPtr(b bool) *bool { return &b }

// env seam ACTIVE + strategy toggles OFF → the line says off(strategy), not
// the pre-D102-1 "BE=on · trail=on" literal.
func TestExitPolicyBootLineStrategyOffSeamActive(t *testing.T) {
	t.Setenv("EXIT_MECHS_SUSPENDED", "0")
	line := ExitPolicyBootLine(1.5, 3.0, 1, true, ds102BoolPtr(false), ds102BoolPtr(false))
	for _, want := range []string{"BE=off(strategy)", "trail=off(strategy)", "seam=ACTIVE(env)"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q must contain %q", line, want)
		}
	}
	if strings.Contains(line, "BE=on") || strings.Contains(line, "trail=on") {
		t.Errorf("strategy toggles are OFF — the line must not claim on: %q", line)
	}
}

// main.go boots BEFORE strategies load → the strategy-source fields are n/a,
// never a literal.
func TestExitPolicyBootLineNoStrategyReadsNA(t *testing.T) {
	t.Setenv("EXIT_MECHS_SUSPENDED", "0")
	line := ExitPolicyBootLineLive(1.5)
	for _, want := range []string{"BE=n/a(strategy)", "trail=n/a(strategy)", "seam=ACTIVE(env)"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q must contain %q", line, want)
		}
	}
	if strings.Contains(line, "BE=on") || strings.Contains(line, "BE=off") ||
		strings.Contains(line, "trail=on") || strings.Contains(line, "trail=off") {
		t.Errorf("no strategy loaded — BE/trail must be n/a, not on/off: %q", line)
	}
}

// strategy toggles ON + env seam SUSPENDED (default) → both layers stated; the
// effective posture is their AND, readable from the line.
func TestExitPolicyBootLineStrategyOnSeamSuspended(t *testing.T) {
	t.Setenv("EXIT_MECHS_SUSPENDED", "")
	line := ExitPolicyBootLine(1.5, 3.0, 1, true, ds102BoolPtr(true), ds102BoolPtr(true))
	for _, want := range []string{"BE=on(strategy)", "trail=on(strategy)", "seam=SUSPENDED(env)"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q must contain %q", line, want)
		}
	}
}

// the manager call site reads the real RiskControlConfig the mechanics gate on.
func TestExitPolicyBootLineForStrategyReadsRiskControl(t *testing.T) {
	t.Setenv("EXIT_MECHS_SUSPENDED", "0")
	rc := store.RiskControlConfig{
		BreakevenEnabled: ds102BoolPtr(true),
		TrailingEnabled:  ds102BoolPtr(false),
	}
	line := ExitPolicyBootLineForStrategy(1.5, rc)
	for _, want := range []string{"BE=on(strategy)", "trail=off(strategy)", "seam=ACTIVE(env)"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q must contain %q", line, want)
		}
	}
}
