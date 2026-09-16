package mcp

import (
	"errors"
	"strings"
)

// ── CLASS 46 (2026-09-02) — ONE CLASSIFICATION ──────────────────────────────
// Two labelling systems used to run side by side on the same failure line:
// `class=` (classifyAIError) and `timeout_source=`, whose value DEFAULTED to
// "transport" and was overridden for only four sentinels. Over the 50 audited
// DeepSeek failures it was right on 5 and wrong on 23 — it tagged 5xx bodies,
// parse failures and empty 200s as transport. A label that is wrong more often
// than it is right is worse than no label: it is the first thing a reader
// trusts. `timeout_source` is DELETED; this is the only classifier.
//
// A class answers exactly one question: WHO failed, and can the model help?
// That second half is FailureIsProviderSide — the model can only help with
// `validator`.

// FailureClass is the single label vocabulary.
type FailureClass string

const (
	ClassOK            FailureClass = "ok"
	ClassTransport     FailureClass = "transport"      // socket died: EOF, RST, refused, TLS
	ClassHTTP5xx       FailureClass = "http_5xx"       // provider accepted then failed (503 overloaded, 502…)
	ClassHTTP4xx       FailureClass = "http_4xx"       // OUR request is wrong (400, 401, 429 is 4xx but retryable — see below)
	ClassTotalDeadline FailureClass = "total_deadline" // the planner's whole-call ceiling
	ClassIdle          FailureClass = "idle"           // the stream watchdog fired
	ClassParse         FailureClass = "parse"          // a body arrived and would not parse
	ClassEmpty200      FailureClass = "empty_200"      // HTTP 200 with no usable content
	ClassTooLong       FailureClass = "too_long"       // the model's answer exceeded a limit
	ClassValidator     FailureClass = "validator"      // the DOCUMENT is wrong — the only class the model can fix
	ClassAuthConfig    FailureClass = "auth_config"    // no key / misconfigured client
	ClassContext       FailureClass = "context"        // a caller cancelled
	ClassOther         FailureClass = "other"
)

// ClassifyFailure assigns exactly one class from the Go error and the HTTP
// status the call observed (0 when no response arrived). Most-specific first.
// httpStatus is authoritative when a response DID arrive: a 503 body is
// http_5xx even though its text contains no transport tokens.
func ClassifyFailure(err error, httpStatus int) FailureClass {
	if err == nil {
		return ClassOK
	}
	msg := err.Error()
	switch {
	// Sentinels first — these are OUR deadlines and cannot be confused.
	case errors.Is(err, ErrStreamTotalDeadline):
		return ClassTotalDeadline
	case errors.Is(err, ErrStreamIdleDeadline), errors.Is(err, ErrWatchdogIdle):
		return ClassIdle
	case strings.Contains(msg, "API key not set"):
		return ClassAuthConfig
	case strings.Contains(msg, "Client.Timeout"):
		// The whole-request ceiling. It is OUR clock, not the peer's socket.
		return ClassTotalDeadline
	}
	// A status the call actually observed outranks text sniffing.
	if st := statusFrom(msg, httpStatus); st > 0 {
		switch {
		case st >= 500:
			return ClassHTTP5xx
		case st == 429:
			return ClassHTTP5xx // rate-limited: provider-side and retryable, same handling
		case st >= 400:
			return ClassHTTP4xx
		}
	}
	switch {
	case strings.Contains(msg, "empty") && strings.Contains(msg, "response"),
		strings.Contains(msg, "no content"), strings.Contains(msg, "stream produced no result"):
		return ClassEmpty200
	case strings.Contains(msg, "too long"), strings.Contains(msg, "exceeds"), errors.Is(err, ErrTooLong):
		return ClassTooLong
	case strings.Contains(msg, "no JSON object found"), strings.Contains(msg, "unmarshal"),
		strings.Contains(msg, "fail to parse"), strings.Contains(msg, "invalid character"):
		return ClassParse
	case strings.Contains(msg, "connection reset"), strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"), strings.Contains(msg, "TLS handshake"),
		strings.Contains(msg, "EOF"), strings.Contains(msg, "stream error"),
		strings.Contains(msg, "unexpected EOF"):
		return ClassTransport
	case strings.Contains(msg, "context deadline exceeded"), strings.Contains(msg, "context canceled"):
		return ClassContext
	}
	return ClassOther
}

// statusFrom prefers the observed status; falls back to "(status NNN)" in the
// message so a wrapped error still classifies.
func statusFrom(msg string, observed int) int {
	if observed >= 100 {
		return observed
	}
	i := strings.Index(msg, "(status ")
	if i < 0 {
		return 0
	}
	n := 0
	for _, r := range msg[i+len("(status "):] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
		if n > 999 {
			return 0
		}
	}
	return n
}

// FailureIsProviderSide answers the question that actually matters: did the
// model ever produce an answer this failure is ABOUT? If not, feeding the
// failure back to it as its own defect is poisoned feedback (class 34/37).
// Only `validator` is the model's to fix.
func FailureIsProviderSide(c FailureClass) bool {
	switch c {
	case ClassTransport, ClassHTTP5xx, ClassTotalDeadline, ClassIdle,
		ClassEmpty200, ClassTooLong, ClassContext:
		return true
	}
	// ClassParse is deliberately NOT here. The dispatch says
	// "parse-of-an-empty-body" is provider-side — an empty body classifies as
	// empty_200 and is covered above. A parse failure of a document the model
	// actually WROTE is the model's defect: the repair path exists for exactly
	// that, and resending the identical prompt would loop on the same
	// malformed output forever. (The pre-existing class-41 fixture caught this
	// over-generalisation; it was right.)
	return false
}

// ErrTooLong marks an answer that exceeded a size limit. Provider-side by the
// rule above: the model produced something, but not something we can use, and
// telling it "your plan is too long" is not a validator defect it can repair
// from the rejected document.
var ErrTooLong = errors.New("response too long")

// ErrWatchdogIdle is the reader error a WATCHDOG close produces, distinct from
// a peer's "unexpected EOF" (class 46 D4): the two were indistinguishable, so
// "0 idle kills" could not be told apart from "0 idle kills we can see".
var ErrWatchdogIdle = errors.New("watchdog_idle")
