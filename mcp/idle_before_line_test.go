package mcp

import (
	"strings"
	"testing"
)

// INSTRUMENT HONESTY (owner ruling 2026-09-03) — idle_before must be on the
// ai_call line, not only in the conn trace.
//
// The 2026-09-03 08:11:38 cut: a planner stream reset by a peer FIN at 283.4s
// with 50,489 reasoning chars in. The conn trace carried the one number that
// might explain it — `reused=true idle_before=101212ms` — while the successful
// resend ran on a connection idle only 34,935ms. The trace is INFO-only and
// unstructured; the ai_call line is what gets grepped and counted. If cuts
// cluster above some idle threshold the fix is IdleConnTimeout below it, and
// that decision needs a queryable column, not a hand-read log.
func TestAiCallLineCarriesIdleBefore(t *testing.T) {
	for _, tc := range []struct {
		name       string
		ok         bool
		idleMs     int64
		reused     bool
		wantFields []string
	}{
		{"failure on a reused idle connection", false, 101212, true,
			[]string{"ok=false", "idle_before_ms=101212", "conn_reused=true"}},
		{"success on a fresh connection", true, 0, false,
			[]string{"ok=true", "idle_before_ms=0", "conn_reused=false"}},
		{"success on a briefly idle reuse", true, 34935, true,
			[]string{"ok=true", "idle_before_ms=34935", "conn_reused=true"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			line := AiCallLine(AiCallFields{
				Model: "deepseek-v4-pro", DurationMs: 283425, OK: tc.ok,
				Retries: 1, TTFBMs: 568, ReasoningChars: 50489,
				IdleBeforeMs: tc.idleMs, ConnReused: tc.reused,
				Class: "transport", HTTPStatus: 200, Err: "connection reset by peer",
			})
			for _, want := range tc.wantFields {
				if !strings.Contains(line, want) {
					t.Errorf("ai_call line missing %q:\n%s", want, line)
				}
			}
		})
	}
}

// The line must stay one grep-able key=value record — no field silently
// dropped when a value is zero, or the counting is wrong in exactly the
// direction that hides fresh connections.
func TestAiCallLineNeverOmitsIdleFields(t *testing.T) {
	line := AiCallLine(AiCallFields{Model: "m", OK: true})
	for _, want := range []string{"idle_before_ms=", "conn_reused="} {
		if !strings.Contains(line, want) {
			t.Errorf("a zero-valued %q was dropped — a fresh connection would vanish from the counts:\n%s", want, line)
		}
	}
}
