package kernel

import (
	"os"
	"strconv"
	"strings"
)

// ResolvePlannerRetryMode (planner-speed wave 3.5, 2026-08-31) reads RETRY_MODE
// (repair|reauthor, default repair) — the one-click revert knob for the repair
// retry without a deploy.
func ResolvePlannerRetryMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("RETRY_MODE"))) {
	case "reauthor":
		return "reauthor"
	default:
		return "repair"
	}
}

// PlannerStreamIdleSeconds (planner-speed wave 4.2, 2026-08-31) reads
// AI_PLAN_STREAM_IDLE_SECS (default 30) — the idle-chunk deadline for the
// planner's SSE stream: a stalled read dies in 30s. The whole-call ceiling is
// PlannerStreamTotalSeconds (class 37) — NOT http.Client.Timeout, which
// silently bounded the LIVE stream at 600s until 2026-09-01 (the old comment
// here claimed "a live-but-slow stream is never killed"; it was, 11 times).
func PlannerStreamIdleSeconds() int {
	d := 30
	if v := strings.TrimSpace(os.Getenv("AI_PLAN_STREAM_IDLE_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			d = n
		}
	}
	return d
}

// PlannerStreamTotalSeconds (class 37, 2026-09-01) reads
// AI_PLAN_TOTAL_DEADLINE_SECS (default 1200) — the whole-call ceiling for ONE
// planner attempt on the SSE path. Evidence 2026-08-30 17:00 → 09-01 17:30 CT:
// successful max-reasoning full reads p50 448s · p90 552s · p95 581s · max
// 599.5s (right-censored at the old 600s ceiling); 11 of 80 such attempts were
// killed at 600.0s with 71k-140k reasoning chars already streaming; the
// 65536-token completion cap is ~1000s at the median 65 tok/s. 1200 = 2× the
// observed max success and covers the cap at median throughput. The resolved
// value is always > the idle deadline (a total at or below idle would fire
// first and misreport the class): total <= idle → idle + 60.
func PlannerStreamTotalSeconds() int {
	d := 1200
	if v := strings.TrimSpace(os.Getenv("AI_PLAN_TOTAL_DEADLINE_SECS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			d = n
		}
	}
	if idle := PlannerStreamIdleSeconds(); d <= idle {
		d = idle + 60
	}
	return d
}
