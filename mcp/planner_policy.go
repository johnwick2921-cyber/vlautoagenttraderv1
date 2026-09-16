package mcp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nofx/security"
)

// ── CLASS 46 (2026-09-02) — ONE SOURCE PER POLICY FIELD ─────────────────────
// The class-41 boot line printed watchdog_log=on, keepalive=30s,
// serialize_executor=off and resend_identical=on as STRING LITERALS, and its
// fixture asserted the same literals — so the line could not drift from the
// code, it could only drift from REALITY, undetectably. (Keepalive on the wire
// was 14-20 s while the line said 30.) Class 6, Go-side theatre, on the wave
// that shipped to fix transport honesty. Every field below is now a function;
// the boot line and its fixture both call these, never a literal.

// WatchdogPreTokenSeconds — the idle limit BEFORE the first data byte.
// AI_PLAN_WATCHDOG_PRE_SECS, default 600: DeepSeek closes queued requests at
// ~10 minutes, so waiting longer than that in a queue proves nothing. Heartbeat
// comment lines DO keep this alive on purpose — a queued request is alive.
func WatchdogPreTokenSeconds() int {
	return envInt("AI_PLAN_WATCHDOG_PRE_SECS", 600, 30, 1200)
}

// EffectivePostTokenSeconds (owner ruling 2026-09-02) is the limit ACTUALLY in
// force after the first token: the smaller of the resolver default and any
// per-call override the planner passes. The 2026-09-02 20:50 fire reported
// "limit 30s" while the boot line advertised post90s, because the planner
// passes a 30 s idle and the tighter bound wins. Printing the default alone
// was true of the resolver and false of the behaviour.
//
// override <= 0 means "no override" and the default stands.
func EffectivePostTokenSeconds(overrideSecs int) int {
	def := WatchdogPostTokenSeconds()
	if overrideSecs > 0 && overrideSecs < def {
		return overrideSecs
	}
	return def
}

// WatchdogPostTokenSeconds — the idle limit AFTER the first data byte,
// measured since the last DATA delta (reasoning or content), NOT since the last
// line. AI_PLAN_WATCHDOG_POST_SECS, default 90.
//
// This is the defect it exists for: the old watchdog reset on every scanned
// LINE, including DeepSeek's ": keep-alive" comments, so a generation that had
// stalled while still heartbeating ran to the 1200 s ceiling. It had never
// fired. A watchdog that cannot fire is not a watchdog.
func WatchdogPostTokenSeconds() int {
	return envInt("AI_PLAN_WATCHDOG_POST_SECS", 90, 15, 600)
}

// StormCapPerRead — the maximum PROVIDER CALLS one planner read may make
// across all its attempts. AI_PLAN_STORM_CAP, default 5.
//
// Observed 2026-09-02 01:15 CT: a 503 burst produced 3 planner attempts × 3
// client tries = 9 calls in ~7 s against an already-overloaded edge, and every
// one of them failed. Retrying harder at something that is shedding load is
// the wrong direction.
func StormCapPerRead() int { return envInt("AI_PLAN_STORM_CAP", 5, 1, 20) }

// ResendIdenticalOnProviderFailure reports the class-41 resend policy as the
// CODE implements it: the planner loop resends the identical prompt exactly
// when FailureIsProviderSide says the model never answered.
func ResendIdenticalOnProviderFailure() bool { return true }

// SerializeExecutorDuringPlannerStream reports whether the executor's provider
// call is held while a planner stream is open. Class 41 P3 measured 4 cuts in
// 71 overlapped streams vs 0 in 6 non-overlapped — no power, no effect shown —
// so it was NOT added, and this reports that fact rather than asserting it.
func SerializeExecutorDuringPlannerStream() bool { return false }

// TransportTraceEnabled reports whether httptrace is installed on the planner
// client (class 46 D6). AI_PLAN_TRACE, default ON.
func TransportTraceEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("AI_PLAN_TRACE")))
	return v == "" || v == "1" || v == "true" || v == "on" || v == "yes"
}

// DialerKeepAliveSet returns the keepalive the safe client's dialer SETS, read
// from the one place that sets it.
func DialerKeepAliveSet() time.Duration { return security.DialerKeepAlive }

// ObservedKeepAlive returns the keepalive interval actually seen on the wire,
// and false when nothing has been observed. Nothing observes it today, so this
// returns false and the boot line prints "observed=n/a" — an honest gap rather
// than a claim.
func ObservedKeepAlive() (time.Duration, bool) { return 0, false }

// PlannerClientPolicyLine (D7) renders the boot line with EVERY field read
// from the functions above. The fixture calls the same functions, so a literal
// anywhere here fails the test.
func PlannerClientPolicyLine() string {
	sched := StreamRetryBackoffSchedule()
	parts := make([]string, 0, len(sched))
	for _, d := range sched {
		parts = append(parts, d.String())
	}
	observed := "n/a"
	if v, ok := ObservedKeepAlive(); ok {
		observed = v.String()
	}
	// The planner's own per-call idle is the usual override; the EFFECTIVE
	// limit is the tighter of the two and is what actually closes a stream.
	eff := EffectivePostTokenSeconds(PlannerIdleOverrideSeconds())
	effNote := ""
	if eff != WatchdogPostTokenSeconds() {
		effNote = fmt.Sprintf("→eff%ds", eff)
	}
	return fmt.Sprintf("🛰 planner client: tries=%d backoff=%s keepalive_set=%s observed=%s watchdog=pre%ds/post%ds%s(data) resend_identical=%t serialize=%t storm_cap=%d trace=%t (class 46)",
		StreamRetryTries(), strings.Join(parts, "→"), DialerKeepAliveSet(), observed,
		WatchdogPreTokenSeconds(), WatchdogPostTokenSeconds(), effNote,
		ResendIdenticalOnProviderFailure(), SerializeExecutorDuringPlannerStream(),
		StormCapPerRead(), TransportTraceEnabled())
}

func envInt(key string, def, lo, hi int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= lo && n <= hi {
			return n
		}
	}
	return def
}

// PlannerIdleOverrideSeconds reports the per-call idle the planner passes
// (AI_PLAN_STREAM_IDLE_SECS, default 30). Read here so the boot line's
// effective figure comes from the same env the planner reads, with no import
// cycle back into kernel.
func PlannerIdleOverrideSeconds() int {
	return envInt("AI_PLAN_STREAM_IDLE_SECS", 30, 1, 3600)
}
