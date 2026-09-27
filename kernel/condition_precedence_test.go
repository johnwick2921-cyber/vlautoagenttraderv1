package kernel

import (
	"strings"
	"testing"
)

// W1 (c) — ONE precedence for a condition's live|shadow status:
//
//	session override → strategy override → LIVE_CONDITIONS → SHADOW_CONDITIONS → default
//
// Before W1 the env step was a first-match scan over a string that listed the
// SHADOW tokens first, so a name in BOTH env lists resolved SHADOW although the
// resolver's own comment promised LIVE priority. Every env value here goes
// through the real ShadowConditionsEnv (t.Setenv), the string the production
// callers pass.

func TestConditionEnvCollisionResolvesLive(t *testing.T) {
	t.Setenv("SHADOW_CONDITIONS", "fvg_entry,reject")
	t.Setenv("LIVE_CONDITIONS", "fvg_entry,reject")
	env := ShadowConditionsEnv()
	for _, c := range []string{"fvg_entry", "reject"} {
		if got := ConditionStatus(c, nil, nil, env); got != ConditionLive {
			t.Errorf("%s in BOTH env lists resolved %q — LIVE_CONDITIONS outranks SHADOW_CONDITIONS", c, got)
		}
	}
	// The process boot line (levels_volume_boot.go) renders the same resolver.
	ledger := ConditionStatusLedger(nil, nil, ShadowConditionsEnv())
	if !strings.Contains(ledger, "shadow [breakout_retest]") {
		t.Errorf("boot ledger must list only breakout_retest as shadow under the collision, got %q", ledger)
	}
}

func TestConditionPrecedenceLadder(t *testing.T) {
	t.Setenv("SHADOW_CONDITIONS", "reject,hold")
	t.Setenv("LIVE_CONDITIONS", "reject,fvg_entry")
	env := ShadowConditionsEnv()

	cases := []struct {
		name          string
		cond          string
		base, session map[string]string
		want          string
	}{
		{"session beats strategy", "reject", map[string]string{"reject": "live"}, map[string]string{"reject": "shadow"}, ConditionShadow},
		{"strategy beats LIVE_CONDITIONS", "reject", map[string]string{"reject": "shadow"}, nil, ConditionShadow},
		{"LIVE_CONDITIONS beats SHADOW_CONDITIONS", "reject", nil, nil, ConditionLive},
		{"SHADOW_CONDITIONS beats the live default", "hold", nil, nil, ConditionShadow},
		{"LIVE_CONDITIONS beats the shadow default", "fvg_entry", nil, nil, ConditionLive},
		{"default when nothing names it", "breakout_retest", nil, nil, ConditionShadow},
		{"default live when nothing names it", "reclaim", nil, nil, ConditionLive},
	}
	for _, tc := range cases {
		if got := ConditionStatus(tc.cond, tc.base, tc.session, env); got != tc.want {
			t.Errorf("%s: %s resolved %q, want %q", tc.name, tc.cond, got, tc.want)
		}
	}
}

// The planner prompt's armable vocabulary comes from the same resolver, so the
// collision must move the condition from "SHADOWED — do not arm" to ARMABLE.
func TestPlannerPromptArmableLineHonoursLiveOverShadow(t *testing.T) {
	t.Setenv("SHADOW_CONDITIONS", "fvg_entry")
	t.Setenv("LIVE_CONDITIONS", "fvg_entry")
	p := plannerOutputContract(8, 3, true, true, false)
	i := strings.Index(p, "ARMABLE + LIVE")
	if i < 0 {
		t.Fatal("the prompt lost its ARMABLE + LIVE line")
	}
	line := p[i:]
	if j := strings.Index(line, "NOT armable at all"); j >= 0 {
		line = line[:j]
	}
	live := line
	if k := strings.Index(line, "armable but SHADOWED"); k >= 0 {
		live = line[:k]
		if strings.Contains(line[k:], "fvg_entry") {
			t.Errorf("fvg_entry is in both env lists and must not be listed as SHADOWED: %q", line)
		}
	}
	if !strings.Contains(live, "fvg_entry→") {
		t.Errorf("fvg_entry must be ARMABLE + LIVE under the collision: %q", line)
	}
}
