package trader

import (
	"testing"

	"nofx/store"
)

// FIX-PLANNER (2026-09-26) item 4 — the death-re-read self-backoff knob. The
// hold that parks a failed death re-read retry must resolve from
// death_reread_retry_min (nil = today's value: wake_min_interval_min), never
// hardcode 30. deathRereadHoldMinutes is the ONE resolution seam the production
// hold site (death_reread.go maybeDeathReread) reads.
func TestFpDeathRereadRetryKnobResolves(t *testing.T) {
	zero, five := 0, 5
	cases := []struct {
		name string
		cfg  *store.DayPlanConfig
		want int
	}{
		{"nil config = today (default 30)", nil, store.DefaultWakeMinIntervalMin},
		{"nil knob = today's value (wake_min_interval_min)", &store.DayPlanConfig{WakeMinIntervalMin: 17}, 17},
		{"explicit value wins", &store.DayPlanConfig{WakeMinIntervalMin: 30, DeathRereadRetryMin: &five}, 5},
		{"explicit zero = immediate retry", &store.DayPlanConfig{WakeMinIntervalMin: 30, DeathRereadRetryMin: &zero}, 0},
		{"negative clamps to zero (never a negative hold)", &store.DayPlanConfig{DeathRereadRetryMin: func() *int { v := -3; return &v }()}, 0},
	}
	for _, c := range cases {
		if got := deathRereadHoldMinutes(c.cfg); got != c.want {
			t.Fatalf("%s: deathRereadHoldMinutes = %d, want %d", c.name, got, c.want)
		}
	}
}
