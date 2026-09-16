package trader

import (
	"encoding/json"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// D4 — the knob is OFF by default: an instrument that bills the provider must
// ship dormant.
func TestRootFixShadowDisabledByDefault(t *testing.T) {
	t.Setenv("SHADOW_AB_ENABLED", "")
	if ShadowABEnabled() {
		t.Fatal("shadow A/B must default OFF")
	}
	for _, on := range []string{"1", "true", "on", "YES"} {
		t.Setenv("SHADOW_AB_ENABLED", on)
		if !ShadowABEnabled() {
			t.Fatalf("%q should enable", on)
		}
	}
	t.Setenv("SHADOW_AB_ENABLED", "maybe")
	if ShadowABEnabled() {
		t.Fatal("junk must not enable")
	}
}

// The pre-registered sample size is resolved, bounded, and defaults to 10.
func TestRootFixShadowTargetResolved(t *testing.T) {
	t.Setenv("SHADOW_AB_N", "")
	if ShadowABTarget() != 10 {
		t.Fatalf("default target = %d, want 10", ShadowABTarget())
	}
	t.Setenv("SHADOW_AB_N", "25")
	if ShadowABTarget() != 25 {
		t.Fatalf("env target = %d", ShadowABTarget())
	}
	for _, bad := range []string{"0", "-3", "999", "junk"} {
		t.Setenv("SHADOW_AB_N", bad)
		if ShadowABTarget() != 10 {
			t.Fatalf("%q must fall back to 10, got %d", bad, ShadowABTarget())
		}
	}
}

// D4 — a disabled knob fires nothing, so nothing is billed and no sample is
// recorded. (The runner is registered but the gate is off.)
func TestRootFixShadowOffFiresNothing(t *testing.T) {
	t.Setenv("SHADOW_AB_ENABLED", "")
	at := plannerTestTrader(t)
	before, _ := store.ShadowABCount(at.store)
	at.maybeRunShadowAB("NY", "2026-09-02", "PROMPT", 8, 5, kernel.PlanFacts{Price: 29000}, nil, nil, "")
	after, _ := store.ShadowABCount(at.store)
	if before != after {
		t.Fatalf("a disabled shadow must not record a sample: %d → %d", before, after)
	}
}

// An empty prompt is never sent — a shadow call with no prompt would bill the
// provider for nothing and pollute the sample.
func TestRootFixShadowRefusesEmptyPrompt(t *testing.T) {
	t.Setenv("SHADOW_AB_ENABLED", "1")
	at := plannerTestTrader(t)
	before, _ := store.ShadowABCount(at.store)
	at.maybeRunShadowAB("NY", "2026-09-02", "   ", 8, 5, kernel.PlanFacts{Price: 29000}, nil, nil, "")
	after, _ := store.ShadowABCount(at.store)
	if before != after {
		t.Fatalf("empty prompt must not fire: %d → %d", before, after)
	}
}

// D4 — the shadow row states the verdict, both sides of the pair, and that it
// wrote nothing. Pure renderer, so the wording is pinned.
func TestRootFixShadowLineWording(t *testing.T) {
	v := ShadowABVerdict{Legal: false, Reasons: []string{"S1 breakdown_continue: void"}, Tokens: 4210, WallMs: 92000}
	line := shadowABLine(3, 10, "NY", "2026-09-02", v, 23769, 448000)
	for _, want := range []string{
		"🔬 shadow A/B 3/10", "fast=ILLEGAL", "tokens=4210", "wall=92.0s", "21% of live",
		"live max tokens=23769", "wall=448.0s", "S1 breakdown_continue: void",
		"SHADOW ONLY, no plan written, no replan budget",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("line missing %q:\n%s", want, line)
		}
	}
	ok := shadowABLine(1, 10, "ASIA", "2026-09-02", ShadowABVerdict{Legal: true, Tokens: 3000, WallMs: 60000}, 20000, 400000)
	if !strings.Contains(ok, "fast=LEGAL") || !strings.Contains(ok, "reasons=—") {
		t.Fatalf("legal row: %s", ok)
	}
	failed := shadowABLine(1, 10, "ASIA", "2026-09-02", ShadowABVerdict{Err: errShadow, Reasons: []string{"shadow call failed: x"}}, 0, 0)
	if !strings.Contains(failed, "fast=CALL-FAILED") {
		t.Fatalf("failed row: %s", failed)
	}
}

var errShadow = errShadowT{}

type errShadowT struct{}

func (errShadowT) Error() string { return "boom" }

// A long reject list is bounded — a measurement line must never flood the log.
func TestRootFixShadowLineBounded(t *testing.T) {
	long := strings.Repeat("reason-that-is-quite-long; ", 80)
	line := shadowABLine(1, 10, "NY", "2026-09-02", ShadowABVerdict{Reasons: []string{long}}, 1, 1)
	if !strings.Contains(line, "…(truncated)") || len(line) > 900 {
		t.Fatalf("line not bounded: len=%d", len(line))
	}
}

// B-3 — the criterion is stated in the boot line, so the promotion rule is
// visible without opening the Guide.
func TestRootFixShadowBootLineStatesCriterion(t *testing.T) {
	line := ShadowABBootLine(false, 10, 0)
	for _, want := range []string{"🔬 shadow A/B", "OFF", "target_n=10", "done=0",
		"legal-rate ≥ max", "median wall ≤50% of max", "never written"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line missing %q: %s", want, line)
		}
	}
	if !strings.Contains(ShadowABBootLine(true, 10, 3), "ON") {
		t.Fatal("enabled state must render ON")
	}
}

// D5 — the facts snapshot round-trips, so an offline A/B can rebuild the facts
// the live attempt was validated against.
func TestRootFixFactsSnapshotRoundTrip(t *testing.T) {
	f := kernel.PlanFacts{Price: 29614.25, DATR: 300, PDH: 29800, PDL: 29400}
	js := FactsSnapshotJSON(f)
	if js == "" || !strings.Contains(js, "29614.25") {
		t.Fatalf("snapshot: %s", js)
	}
	var back kernel.PlanFacts
	if err := jsonUnmarshalShadow(js, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Price != f.Price || back.PDH != f.PDH || back.PDL != f.PDL || back.DATR != f.DATR {
		t.Fatalf("round-trip lost facts: %+v vs %+v", back, f)
	}
}

func jsonUnmarshalShadow(s string, v any) error { return json.Unmarshal([]byte(s), v) }
