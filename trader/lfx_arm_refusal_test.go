package trader

import (
	"testing"
)

// F4 (LONDON-FORENSICS 2026-08-28) — the every-cycle REFUSED spam dedup: one
// log line per arm-spec verdict, silent until the spec (or the refusal reason)
// changes. Pure decision, tested directly.

func TestArmRefusalChangedDedup(t *testing.T) {
	var last map[string]string
	if !armRefusalChanged(&last, "plan:v12:S1", "R:R 0.93 below arm min 2.00") {
		t.Fatal("first refusal must log")
	}
	if armRefusalChanged(&last, "plan:v12:S1", "R:R 0.93 below arm min 2.00") {
		t.Fatal("identical re-refusal must stay silent")
	}
	if !armRefusalChanged(&last, "plan:v12:S1", "stop too close") {
		t.Fatal("a CHANGED refusal reason must log again")
	}
	if !armRefusalChanged(&last, "plan:v13:S1", "R:R 0.93 below arm min 2.00") {
		t.Fatal("a new plan version (spec change) must log again")
	}
	if armRefusalChanged(&last, "plan:v13:S1", "R:R 0.93 below arm min 2.00") {
		t.Fatal("same verdict on the same version stays silent")
	}
}

// GAR-F6 (grand-audit response, 2026-08-28) — the dedup key is the verdict
// CLASS, not the ATR-bearing string: live ATR drift re-logged the same refusal
// (LONDON S4 min-SL "18.29×ATR5m" → "18.67×ATR5m").
func TestArmRefusalClassIgnoresATRDrift(t *testing.T) {
	if armRefusalClass("stop 29592.00 too close (18.00 < 18.29 = 1.0×ATR5m)") != "min_sl" {
		t.Fatal("min-SL verdict must classify as min_sl")
	}
	if armRefusalClass("stop 29592.00 too close (18.00 < 18.67 = 1.0×ATR5m)") != "min_sl" {
		t.Fatal("the ATR-drifted variant must classify as the SAME class")
	}
	if armRefusalClass("R:R 1.34 below arm min 2.00") != "rr" {
		t.Fatal("R:R verdict must classify as rr")
	}
	if armRefusalClass("HTF veto: htf_veto: long vs 1h TRENDING_DOWN (LL 29577.75 @07:27)") != "veto" {
		t.Fatal("veto verdict must classify as veto")
	}

	var last map[string]string
	key1 := "plan:v12:S4"
	if !armRefusalChanged(&last, key1, "min_sl") {
		t.Fatal("first min_sl refusal must log")
	}
	if armRefusalChanged(&last, key1, "min_sl") {
		t.Fatal("ATR-drifted re-refusal of the SAME class must stay silent")
	}
	if !armRefusalChanged(&last, "plan:v12:S4", "rr") {
		t.Fatal("a different CLASS on the same spec must log")
	}
}

// F1a (LONDON-FORENSICS 2026-08-28) — the planner token knob default.
func TestAIPlanMaxTokensDefault(t *testing.T) {
	t.Setenv("AI_PLAN_MAX_TOKENS", "")
	if got := aiPlanMaxTokens(); got != 65536 {
		t.Fatalf("default = %d, want 65536 (2× the observed 32768 truncation ceiling)", got)
	}
	t.Setenv("AI_PLAN_MAX_TOKENS", "100000")
	if got := aiPlanMaxTokens(); got != 100000 {
		t.Fatalf("override = %d, want 100000", got)
	}
	t.Setenv("AI_PLAN_MAX_TOKENS", "0") // invalid → default
	if got := aiPlanMaxTokens(); got != 65536 {
		t.Fatalf("zero override = %d, want 65536 fallback", got)
	}
}
