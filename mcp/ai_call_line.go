package mcp

import "fmt"

// AI_CALL LINE (owner ruling 2026-09-03) — the one telemetry record per
// provider call, rendered by ONE pure function so a fixture can pin it.
//
// It grew field by field inside two log statements (ok / not-ok), which is how
// idle_before came to exist in the conn trace and not here. The trace is
// INFO-only prose; this line is what gets grepped, counted and put in a table.
// The 2026-09-03 08:11:38 cut is the case in point: the trace held the only
// number that might explain it (reused=true idle_before=101212ms, against
// 34,935ms on the resend that succeeded) and nothing could count it.

// AiCallFields is one call's telemetry. Zero values render as zeros — a field
// is never omitted, because an omitted field undercounts exactly the cases it
// describes (a fresh connection would vanish from the idle_before tally).
type AiCallFields struct {
	Model          string
	DurationMs     int64
	FinishReason   string
	OK             bool
	Retries        int
	TTFBMs         int64
	ReasoningChars int64

	// The connection this call rode. From the httptrace ConnTrace, hoisted out
	// of the trace line so it is queryable.
	IdleBeforeMs int64
	ConnReused   bool

	// Failure-only.
	DeadlineS    int64
	Class        string
	ProviderSide bool
	HTTPStatus   int
	RequestID    string
	Err          string
}

// AiCallLine renders the record. Pure.
func AiCallLine(f AiCallFields) string {
	if f.OK {
		finish := f.FinishReason
		if finish == "" {
			finish = "unknown"
		}
		return fmt.Sprintf("ai_call model=%s duration_ms=%d finish_reason=%s ok=true retries=%d ttfb_ms=%d reasoning_chars=%d idle_before_ms=%d conn_reused=%t http_status=%d request_id=%q",
			f.Model, f.DurationMs, finish, f.Retries, f.TTFBMs, f.ReasoningChars,
			f.IdleBeforeMs, f.ConnReused, f.HTTPStatus, f.RequestID)
	}
	return fmt.Sprintf("ai_call model=%s duration_ms=%d finish_reason=n/a ok=false retries=%d ttfb_ms=%d reasoning_chars=%d idle_before_ms=%d conn_reused=%t deadline_s=%d class=%s provider_side=%v http_status=%d request_id=%q err=%q",
		f.Model, f.DurationMs, f.Retries, f.TTFBMs, f.ReasoningChars,
		f.IdleBeforeMs, f.ConnReused, f.DeadlineS, f.Class, f.ProviderSide,
		f.HTTPStatus, f.RequestID, f.Err)
}
